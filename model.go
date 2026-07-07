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

func cmdModel(args []string) {
	if len(args) == 0 {
		fail(`usage: communikey model list | test <name> | call <name> "<prompt>" | secret set <name> <valeur>`)
	}
	switch args[0] {
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
			fail("usage: communikey model secret set <name> <valeur>")
		}
		cmdModelSecretSet(args[2:])
	default:
		fail(`usage: communikey model list | test <name> | call <name> "<prompt>" | secret set <name> <valeur>`)
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

func cmdModelList(args []string) {
	specs, err := loadModelSpecs()
	if err != nil {
		fail(err.Error())
	}
	_, issues, err := buildModelRegistry()
	if err != nil {
		fail(err.Error())
	}
	reasonByName := map[string]string{}
	for _, iss := range issues {
		reasonByName[iss.Name] = iss.Reason
	}

	var entries []modelListEntry
	for _, spec := range specs {
		secret := "aucun"
		if spec.Auth != "" {
			secret = "déclaré"
		}
		status, reason := "actif", ""
		if r, bad := reasonByName[spec.Name]; bad {
			status, reason = "erreur", r
		}
		entries = append(entries, modelListEntry{
			Name: spec.Name, Kind: spec.Kind, BaseURL: spec.BaseURL,
			Secret: secret, Status: status, Reason: reason,
		})
	}

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

func cmdModelTest(name string) {
	providers, issues, err := buildModelRegistry()
	if err != nil {
		fail(err.Error())
	}
	p, ok := findModelProvider(providers, name)
	if !ok {
		for _, iss := range issues {
			if iss.Name == name {
				fail(fmt.Sprintf("provider %q non enregistré : %s", name, iss.Reason))
			}
		}
		fail(fmt.Sprintf("provider %q non enregistré (voir « communikey model list »)", name))
	}
	ctx, cancel := context.WithTimeout(context.Background(), modelDefaultTimeout)
	defer cancel()
	if _, err := p.Complete(ctx, "ping", ModelOptions{}); err != nil {
		fail(fmt.Sprintf("provider %q injoignable : %v", name, err))
	}
	fmt.Printf("✓ provider %s répond\n", name)
}

func cmdModelCall(name string, rest []string) {
	providers, issues, err := buildModelRegistry()
	if err != nil {
		fail(err.Error())
	}
	p, ok := findModelProvider(providers, name)
	if !ok {
		for _, iss := range issues {
			if iss.Name == name {
				fail(fmt.Sprintf("provider %q non enregistré : %s", name, iss.Reason))
			}
		}
		fail(fmt.Sprintf("provider %q non enregistré (voir « communikey model list »)", name))
	}

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

func cmdModelSecretSet(args []string) {
	if len(args) < 2 {
		fail("usage: communikey model secret set <name> <valeur>")
	}
	name, value := args[0], args[1]
	if err := saveModelSecret(name, value); err != nil {
		fail(err.Error())
	}
	fmt.Printf("✓ secret %q enregistré dans le vault (%s)\n", name, modelSecretsPath())
}
