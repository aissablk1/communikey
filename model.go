package main

// model.go — sous-commandes `communikey model …` : consommer directement un
// backend de modèle (Ollama, LocalAI, HuggingFace…) déclaré dans models.json.
// Distinct de `provider` (détection d'état de CLI d'agent) — voir
// modelprovider.go pour la frontière.

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

const modelUsage = `usage: communikey model presets | add <provider> [--model <m>] [--auth env:VAR|vault:NAME] | list | test <name> | call <name> "<prompt>" | secret set <name> [<valeur>]`

func cmdModel(args []string) {
	if len(args) == 0 {
		fail(modelUsage)
	}
	switch args[0] {
	case "presets":
		cmdModelPresets(args[1:])
	case "add":
		if len(args) < 2 {
			fail(`usage: communikey model add <provider> [--model <m>] [--auth env:VAR|vault:NAME]  (catalogue : « communikey model presets »)`)
		}
		cmdModelAdd(args[1], args[2:])
	case "list":
		cmdModelList(args[1:])
	case "test":
		if len(args) < 2 {
			fail("usage: communikey model test <name>")
		}
		cmdModelTest(args[1])
	case "call":
		if len(args) < 2 {
			fail(`usage: communikey model call <name> ["<prompt>"] [--json] [--timeout <durée>]  (prompt lu sur stdin si omis)`)
		}
		cmdModelCall(args[1], args[2:])
	case "secret":
		if len(args) < 2 || args[1] != "set" {
			fail("usage: communikey model secret set <name> [<valeur>]  (valeur lue sur stdin si omise)")
		}
		cmdModelSecretSet(args[2:])
	default:
		fail(modelUsage)
	}
}

type modelListEntry struct {
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	BaseURL string `json:"base_url"`
	Secret  string `json:"secret"` // "aucun" | "déclaré"
	Status  string `json:"status"` // "actif" | "erreur"
	Reason  string `json:"reason,omitempty"`
}

// buildModelListEntries projette les specs + issues du registre en lignes
// affichables/sérialisables. Pure (aucun I/O) pour être testable. Renvoie
// TOUJOURS une slice non-nil : `--json` sur une config vide doit émettre `[]`,
// jamais `null`. Une entrée sans nom est rejetée par le registre, donc marquée
// "erreur" (jamais "actif") et affichée sous modelUnnamedLabel.
func buildModelListEntries(specs []modelSpec, issues []modelRegistryIssue) []modelListEntry {
	reasonByName := map[string]string{}
	for _, iss := range issues {
		reasonByName[iss.Name] = iss.Reason
	}

	entries := []modelListEntry{}
	for _, spec := range specs {
		secret := "aucun"
		if spec.Auth != "" {
			secret = "déclaré"
		}
		name := spec.Name
		status, reason := "actif", ""
		if name == "" {
			name = modelUnnamedLabel
			status = "erreur"
			if r, ok := reasonByName[modelUnnamedLabel]; ok {
				reason = r
			} else {
				reason = "name manquant"
			}
		} else if r, bad := reasonByName[spec.Name]; bad {
			status, reason = "erreur", r
		}
		entries = append(entries, modelListEntry{
			Name: name, Kind: spec.Kind, BaseURL: spec.BaseURL,
			Secret: secret, Status: status, Reason: reason,
		})
	}
	return entries
}

func cmdModelList(args []string) {
	specs, err := loadModelSpecs()
	if err != nil {
		fail(err.Error())
	}
	_, issues, err := buildModelRegistry()
	if err != nil {
		fail(err.Error())
	}
	entries := buildModelListEntries(specs, issues)

	if wantJSON(args) {
		emitJSON(entries)
		return
	}
	if len(entries) == 0 {
		fmt.Printf("Aucun provider de modèle configuré (%s absent ou vide).\n", modelsConfigPath())
		return
	}
	fmt.Println("Providers de modèle configurés :")
	for _, e := range entries {
		line := fmt.Sprintf("  • %-14s %-20s %-8s secret=%s", e.Name, e.Kind, e.Status, e.Secret)
		if e.Reason != "" {
			line += " — " + e.Reason
		}
		fmt.Println(line)
	}
}

type modelPresetEntry struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Kind    string `json:"kind"`
	BaseURL string `json:"base_url"`
	Model   string `json:"model"`
	AuthEnv string `json:"auth_env,omitempty"`
	Source  string `json:"source"` // "clawcodex" (base_url vérifiée) | "known" (à valider)
}

// cmdModelPresets liste le catalogue vérifié (modelpresets.go). Lecture seule :
// il n'écrit rien dans models.json — c'est `model add` qui active un provider.
func cmdModelPresets(args []string) {
	ids := sortedPresetIDs()
	entries := make([]modelPresetEntry, 0, len(ids))
	for _, id := range ids {
		p := modelPresets[id]
		entries = append(entries, modelPresetEntry{
			ID: id, Label: p.Label, Kind: p.Kind, BaseURL: p.BaseURL,
			Model: p.Model, AuthEnv: p.AuthEnv, Source: p.Src,
		})
	}
	if wantJSON(args) {
		emitJSON(entries)
		return
	}
	fmt.Printf("Catalogue de providers de modèle (%d) — activer : « communikey model add <id> »\n", len(entries))
	for _, e := range entries {
		auth := e.AuthEnv
		if auth == "" {
			auth = "(local, sans clé)"
		}
		flag := ""
		if e.Source == "known" {
			flag = "  [à valider]"
		}
		fmt.Printf("  • %-15s %-18s %-30s %s%s\n", e.ID, e.Kind, e.Model, auth, flag)
	}
}

// cmdModelAdd active un provider du catalogue en écrivant son entrée (avec le bon
// `kind` — routing "smart") dans models.json. Idempotent (remplace si déjà présent).
func cmdModelAdd(provider string, rest []string) {
	modelOverride, authOverride := "", ""
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case "--model":
			if i+1 >= len(rest) {
				fail("--model attend une valeur")
			}
			modelOverride = rest[i+1]
			i++
		case "--auth":
			if i+1 >= len(rest) {
				fail("--auth attend une valeur (env:VAR ou vault:NAME)")
			}
			authOverride = rest[i+1]
			i++
		default:
			fail("argument inconnu: " + rest[i])
		}
	}

	spec, ok := presetToSpec(provider, modelOverride, authOverride)
	if !ok {
		fail(fmt.Sprintf("provider %q absent du catalogue (voir « communikey model presets »)", provider))
	}
	added, err := upsertModelSpec(spec)
	if err != nil {
		fail(err.Error())
	}
	verb := "ajouté"
	if !added {
		verb = "mis à jour"
	}
	fmt.Printf("✓ provider %q %s dans %s (kind=%s, model=%s)\n", spec.Name, verb, modelsConfigPath(), spec.Kind, spec.Model)
	switch {
	case strings.HasPrefix(spec.Auth, "env:"):
		env := strings.TrimPrefix(spec.Auth, "env:")
		fmt.Printf("  → clé via l'environnement : export %s=<clé>\n", env)
		fmt.Printf("    (ou dans le vault : communikey model add %s --auth vault:%s && communikey model secret set %s <clé>)\n", provider, provider, provider)
	case strings.HasPrefix(spec.Auth, "vault:"):
		name := strings.TrimPrefix(spec.Auth, "vault:")
		fmt.Printf("  → clé dans le vault : communikey model secret set %s <clé>\n", name)
	default:
		fmt.Println("  → serveur local sans clé — assure-toi qu'il tourne.")
	}
	fmt.Printf("  → teste : communikey model test %s\n", provider)
}

// resolveModelProviderOrFail renvoie le provider vivant nommé `name`, ou termine
// le programme (fail) avec la raison précise de l'issue du registre si l'entrée
// existe mais n'a pu être chargée. Factorise le bloc partagé entre test/call.
func resolveModelProviderOrFail(name string) ModelProvider {
	providers, issues, err := buildModelRegistry()
	if err != nil {
		fail(err.Error())
	}
	if p, ok := findModelProvider(providers, name); ok {
		return p
	}
	for _, iss := range issues {
		if iss.Name == name {
			fail(fmt.Sprintf("provider %q non enregistré : %s", name, iss.Reason))
		}
	}
	fail(fmt.Sprintf("provider %q non enregistré (voir « communikey model list »)", name))
	return nil // inatteignable : fail() termine le processus
}

func cmdModelTest(name string) {
	p := resolveModelProviderOrFail(name)
	ctx, cancel := context.WithTimeout(context.Background(), modelDefaultTimeout)
	defer cancel()
	if _, err := p.Complete(ctx, "ping", ModelOptions{}); err != nil {
		fail(fmt.Sprintf("provider %q injoignable : %v", name, err))
	}
	fmt.Printf("✓ provider %s répond\n", name)
}

func cmdModelCall(name string, rest []string) {
	p := resolveModelProviderOrFail(name)

	var positional []string
	timeout := modelDefaultTimeout
	asJSON := false
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case "--json":
			asJSON = true
		case "--timeout":
			if i+1 >= len(rest) {
				fail("--timeout attend une durée (ex: 45s)")
			}
			d, err := time.ParseDuration(rest[i+1])
			if err != nil {
				fail("durée --timeout invalide: " + err.Error())
			}
			timeout = d
			i++
		default:
			positional = append(positional, rest[i])
		}
	}

	var prompt string
	if len(positional) > 0 {
		prompt = strings.Join(positional, " ")
	} else {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fail("lecture stdin: " + err.Error())
		}
		prompt = strings.TrimSpace(string(data))
	}
	if prompt == "" {
		fail("prompt vide (argument ou stdin requis)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	resp, err := p.Complete(ctx, prompt, ModelOptions{Timeout: timeout})
	if err != nil {
		fail(err.Error())
	}
	if asJSON {
		emitJSON(struct {
			Provider string `json:"provider"`
			Prompt   string `json:"prompt"`
			Response string `json:"response"`
		}{Provider: name, Prompt: prompt, Response: resp})
		return
	}
	fmt.Println(resp)
}

// resolveSecretValue extrait (name, value) des arguments de `secret set`. La
// valeur est lue sur `stdin` quand elle n'est pas passée en argument — chemin
// RECOMMANDÉ : un secret en argv est visible dans l'historique shell, `ps` et
// `/proc` (§5/§38). La forme `<name> <valeur>` reste acceptée (rétro-compat,
// scripts), mais déconseillée. Pure (stdin injecté) pour être testable.
func resolveSecretValue(args []string, stdin io.Reader) (name, value string, err error) {
	if len(args) < 1 {
		return "", "", fmt.Errorf("usage: communikey model secret set <name> [<valeur>]  (valeur lue sur stdin si omise)")
	}
	name = args[0]
	if len(args) >= 2 {
		value = args[1]
	} else {
		data, rerr := io.ReadAll(stdin)
		if rerr != nil {
			return "", "", fmt.Errorf("lecture stdin: %w", rerr)
		}
		value = strings.TrimSpace(string(data))
	}
	if value == "" {
		return "", "", fmt.Errorf("valeur de secret vide (argument ou stdin requis)")
	}
	return name, value, nil
}

func cmdModelSecretSet(args []string) {
	name, value, err := resolveSecretValue(args, os.Stdin)
	if err != nil {
		fail(err.Error())
	}
	if err := saveModelSecret(name, value); err != nil {
		fail(err.Error())
	}
	fmt.Printf("✓ secret %q enregistré dans le vault (%s)\n", name, modelSecretsPath())
}
