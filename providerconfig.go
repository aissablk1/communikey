package main

// providerconfig.go — étendre le registre de providers SANS RECOMPILER.
//
// Les trois providers calibrés (claude/codex/gemini, adapters.go/state.go) restent
// des détecteurs compilés en dur — zéro risque de régression sur une détection déjà
// éprouvée. Ce fichier ajoute une couche PUREMENT ADDITIVE : un fichier JSON
// optionnel (~/.claude/communikey/providers.json, ou COMKEY_STORE_DIR) où on peut
// déclarer un provider SUPPLÉMENTAIRE (OpenCode, Cursor Agent, Kiro CLI…) sans
// toucher au code — le levier d'échelle proposé dans docs/strategy. Aucun fichier
// = aucun changement de comportement (rétro-compatible par construction).
//
// Zéro dépendance externe (§1/§38, cohérent avec le zéro-dep du reste du projet) :
// JSON stdlib, pas YAML.

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

// providerSpec is the JSON-serializable shape of a patternProvider (adapters.go).
type providerSpec struct {
	Name       string   `json:"name"`
	Confirm    []string `json:"confirm,omitempty"`
	Busy       []string `json:"busy,omitempty"`
	IdlePrompt string   `json:"idle_prompt,omitempty"`
	IdleFooter string   `json:"idle_footer,omitempty"`
}

func providersConfigPath() string {
	return filepath.Join(DefaultStoreDir(), "providers.json")
}

// loadUserProviderSpecs reads the optional user config. Fichier absent → (nil, nil),
// zéro-config par défaut. JSON invalide → erreur EXPLICITE (jamais un provider cassé
// qui échoue en silence, §29).
func loadUserProviderSpecs() ([]providerSpec, error) {
	data, err := os.ReadFile(providersConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var doc struct {
		Providers []providerSpec `json:"providers"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("providers.json invalide: %w", err)
	}
	return doc.Providers, nil
}

// compileProviderSpec turns one JSON spec into a live Provider. Un spec sans AUCUN
// pattern renvoie ok=false (rien à enregistrer — pas une erreur, juste incomplet).
func compileProviderSpec(spec providerSpec) (p Provider, err error, ok bool) {
	if spec.Name == "" {
		return nil, fmt.Errorf("provider sans nom"), true
	}
	if len(spec.Confirm) == 0 && len(spec.Busy) == 0 && spec.IdlePrompt == "" && spec.IdleFooter == "" {
		return nil, nil, false
	}
	compile := func(list []string) ([]*regexp.Regexp, error) {
		out := make([]*regexp.Regexp, 0, len(list))
		for _, pat := range list {
			re, err := regexp.Compile(pat)
			if err != nil {
				return nil, fmt.Errorf("regex invalide %q: %w", pat, err)
			}
			out = append(out, re)
		}
		return out, nil
	}
	confirm, err := compile(spec.Confirm)
	if err != nil {
		return nil, fmt.Errorf("provider %q: %w", spec.Name, err), true
	}
	busy, err := compile(spec.Busy)
	if err != nil {
		return nil, fmt.Errorf("provider %q: %w", spec.Name, err), true
	}
	var idlePrompt, idleFooter *regexp.Regexp
	if spec.IdlePrompt != "" {
		if idlePrompt, err = regexp.Compile(spec.IdlePrompt); err != nil {
			return nil, fmt.Errorf("provider %q idle_prompt: %w", spec.Name, err), true
		}
	}
	if spec.IdleFooter != "" {
		if idleFooter, err = regexp.Compile(spec.IdleFooter); err != nil {
			return nil, fmt.Errorf("provider %q idle_footer: %w", spec.Name, err), true
		}
	}
	return patternProvider{
		name: spec.Name, confirm: confirm, busy: busy,
		idlePrompt: idlePrompt, idleFooter: idleFooter,
	}, nil, true
}

// loadUserProviders compiles every usable spec from the user config. Une erreur de
// compilation s'affiche sur stderr (jamais silencieuse) mais n'empêche jamais le
// démarrage — un provider cassé ne doit pas bloquer claude/codex/gemini.
func loadUserProviders() []Provider {
	specs, err := loadUserProviderSpecs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "communikey: providers.json ignoré (%v)\n", err)
		return nil
	}
	var out []Provider
	for _, spec := range specs {
		p, err, ok := compileProviderSpec(spec)
		if err != nil {
			fmt.Fprintf(os.Stderr, "communikey: %v\n", err)
			continue
		}
		if ok {
			out = append(out, p)
		}
	}
	return out
}

// --- communikey provider list | communikey provider test <name> ---

type providerStatusEntry struct {
	name, status, source string
}

// builtinProviderStatus : statut connu des 5 providers compilés en dur (fait établi
// dans le code, pas une donnée de config).
var builtinProviderStatus = map[string]providerStatusEntry{
	"claude":      {"claude", "calibré", "détecteur éprouvé sur de vrais écrans capturés"},
	"codex":       {"codex", "provisoire", "calibré sur openai/codex tag rust-v0.142.3 — jamais confirmé sur écran live"},
	"gemini":      {"gemini", "provisoire", "individuel retiré 18/06/2026 (licence Enterprise seule) — calibré sur @google/gemini-cli 0.40.1"},
	"antigravity": {"antigravity", "provisoire", "successeur officiel de Gemini CLI — calibré par extraction statique sur antigravity-cli 1.0.16, jamais confirmé sur écran live"},
	"clawcodex":   {"clawcodex", "provisoire", "MIT, 25 backends LLM — calibré sur agentforce314/clawcodex (dépôt officiel), jamais confirmé sur écran live"},
}

// knownAbsentProviders : liste informative (aucun pattern, rien enregistré dans le
// registre de détection) — vue le 2026-07-03 sur le panneau cmux Intégrations→CLI
// Hooks, revérifiée non-changée le 2026-07-05 (cf. docs/strategy). Purement
// documentaire pour `provider list` ; ne détecte rien.
var knownAbsentProviders = []providerStatusEntry{
	{"opencode", "absent", "cmux Intégrations→CLI Hooks, jamais calibré"},
	{"cursor-agent", "absent", "cmux Intégrations→CLI Hooks, jamais calibré"},
	{"droid", "absent", "Factory AI — cmux Intégrations→CLI Hooks, jamais calibré"},
	{"hermes", "absent", "vraisemblablement Hermes Agent/Nous Research — cf. vision 07-05, non confirmé visuellement"},
	{"pi-agent", "absent", "vraisemblablement @earendil-works/pi-coding-agent, installé localement — non calibré"},
	{"kiro-cli", "absent", "AWS — cmux Intégrations→CLI Hooks, jamais calibré"},
}

func cmdProvider(args []string) {
	if len(args) == 0 {
		fail("usage: communikey provider list  |  communikey provider test <name>")
	}
	switch args[0] {
	case "list":
		cmdProviderList()
	case "test":
		if len(args) < 2 {
			fail("usage: communikey provider test <name>  (l'écran à tester est lu sur stdin)")
		}
		cmdProviderTest(args[1])
	default:
		fail("usage: communikey provider list  |  communikey provider test <name>")
	}
}

func cmdProviderList() {
	fmt.Println("Providers enregistrés (détection active) :")
	registered := map[string]bool{}
	for _, p := range providers {
		registered[p.Name()] = true
		e, ok := builtinProviderStatus[p.Name()]
		if !ok {
			e = providerStatusEntry{p.Name(), "personnalisé", "chargé depuis " + providersConfigPath()}
		}
		fmt.Printf("  • %-14s %-12s %s\n", e.name, e.status, e.source)
	}
	fmt.Println("\nConnus mais absents du registre (aucun pattern) :")
	for _, e := range knownAbsentProviders {
		if registered[e.name] { // déjà enregistré via providers.json — pas d'entrée en double
			continue
		}
		fmt.Printf("  • %-14s %-12s %s\n", e.name, e.status, e.source)
	}
	fmt.Printf("\nAjouter un provider : déclarer ses patterns dans %s (clé \"providers\": [...]).\n", providersConfigPath())
	fmt.Println("Regex Go stdlib (paquet regexp) : ancrez idle_prompt/idle_footer avec (?m) pour que")
	fmt.Println("^/$ matchent par LIGNE et non sur tout l'écran — comme le font déjà codex/gemini.")
}

func cmdProviderTest(name string) {
	var target Provider
	for _, p := range providers {
		if p.Name() == name {
			target = p
			break
		}
	}
	if target == nil {
		fail(fmt.Sprintf("provider %q non enregistré (patterns absents) — voir « communikey provider list »", name))
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fail(err.Error())
	}
	st := target.Detect(string(data))
	fmt.Printf("provider=%s état détecté=%s\n", name, st)
}
