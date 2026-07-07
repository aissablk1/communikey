package main

import (
	"strings"
	"testing"
)

// TestModelPresetsIntegrity vérifie que chaque preset du catalogue est
// structurellement valide — un preset cassé produirait une entrée models.json
// non fonctionnelle (base_url vide, kind inconnu…) sans que rien ne l'attrape.
func TestModelPresetsIntegrity(t *testing.T) {
	if len(modelPresets) < 40 {
		t.Fatalf("catalogue trop petit (%d) — l'objectif est ~50 providers", len(modelPresets))
	}
	for id, p := range modelPresets {
		if id == "" {
			t.Error("id de preset vide")
		}
		if p.Label == "" {
			t.Errorf("%s: Label vide", id)
		}
		switch p.Kind {
		case "openai-compatible", "anthropic":
		default:
			t.Errorf("%s: Kind inconnu %q (attendu openai-compatible|anthropic)", id, p.Kind)
		}
		if !strings.HasPrefix(p.BaseURL, "http://") && !strings.HasPrefix(p.BaseURL, "https://") {
			t.Errorf("%s: BaseURL %q ne commence pas par http(s)://", id, p.BaseURL)
		}
		if strings.HasSuffix(p.BaseURL, "/") {
			t.Errorf("%s: BaseURL %q se termine par '/' (casse l'ajout de /chat/completions)", id, p.BaseURL)
		}
		if p.Model == "" {
			t.Errorf("%s: Model par défaut vide", id)
		}
		switch p.Src {
		case "clawcodex", "known":
		default:
			t.Errorf("%s: Src inconnu %q (attendu clawcodex|known)", id, p.Src)
		}
		// Un preset sans clé (AuthEnv vide) ne doit être qu'un serveur LOCAL —
		// sinon on appellerait un endpoint distant sans authentification.
		if p.AuthEnv == "" && !strings.Contains(p.BaseURL, "localhost") && !strings.Contains(p.BaseURL, "127.0.0.1") {
			t.Errorf("%s: sans AuthEnv mais BaseURL non-local %q", id, p.BaseURL)
		}
	}
}

func TestFindPreset(t *testing.T) {
	if _, ok := findPreset("groq"); !ok {
		t.Fatal("preset 'groq' attendu dans le catalogue")
	}
	if _, ok := findPreset("anthropic"); !ok {
		t.Fatal("preset 'anthropic' attendu dans le catalogue")
	}
	if _, ok := findPreset("provider-inexistant-xyz"); ok {
		t.Fatal("preset inexistant ne doit pas être trouvé")
	}
}

func TestPresetAuthField(t *testing.T) {
	if got := presetAuthField(modelPreset{AuthEnv: "GROQ_API_KEY"}); got != "env:GROQ_API_KEY" {
		t.Fatalf("attendu env:GROQ_API_KEY, got %q", got)
	}
	if got := presetAuthField(modelPreset{AuthEnv: ""}); got != "" {
		t.Fatalf("serveur local sans clé attendu \"\", got %q", got)
	}
}

func TestSortedPresetIDs(t *testing.T) {
	ids := sortedPresetIDs()
	if len(ids) != len(modelPresets) {
		t.Fatalf("sortedPresetIDs retourne %d ids, catalogue en a %d", len(ids), len(modelPresets))
	}
	for i := 1; i < len(ids); i++ {
		if ids[i-1] > ids[i] {
			t.Fatalf("ids non triés à l'index %d: %q > %q", i, ids[i-1], ids[i])
		}
	}
}

// TestAnthropicKindPresets verrouille le fait que anthropic ET minimax utilisent
// le format Messages natif (routing "smart" — vérifié à la source clawcodex).
func TestAnthropicKindPresets(t *testing.T) {
	for _, id := range []string{"anthropic", "minimax"} {
		p, ok := findPreset(id)
		if !ok {
			t.Fatalf("preset %q manquant", id)
		}
		if p.Kind != "anthropic" {
			t.Errorf("%s: Kind attendu 'anthropic', got %q", id, p.Kind)
		}
	}
}

// TestPresetToSpec vérifie le routing "smart" (le kind vient du catalogue) et
// l'application des overrides.
func TestPresetToSpec(t *testing.T) {
	spec, ok := presetToSpec("anthropic", "", "")
	if !ok {
		t.Fatal("anthropic doit être dans le catalogue")
	}
	if spec.Kind != "anthropic" {
		t.Fatalf("kind smart attendu 'anthropic', got %q", spec.Kind)
	}
	if spec.Auth != "env:ANTHROPIC_API_KEY" {
		t.Fatalf("auth attendu env:ANTHROPIC_API_KEY, got %q", spec.Auth)
	}

	spec2, _ := presetToSpec("groq", "llama-x", "vault:groq")
	if spec2.Model != "llama-x" || spec2.Auth != "vault:groq" {
		t.Fatalf("overrides non appliqués: %+v", spec2)
	}

	spec3, _ := presetToSpec("ollama", "", "")
	if spec3.Auth != "" {
		t.Fatalf("ollama (local) doit avoir auth vide, got %q", spec3.Auth)
	}

	if _, ok := presetToSpec("provider-inexistant-xyz", "", ""); ok {
		t.Fatal("provider inconnu doit renvoyer false")
	}
}

// TestUpsertModelSpec vérifie l'ajout puis le remplacement idempotent (pas de
// doublon) via un round-trip disque réel.
func TestUpsertModelSpec(t *testing.T) {
	t.Setenv("COMKEY_STORE_DIR", t.TempDir())

	added, err := upsertModelSpec(modelSpec{Name: "groq", Kind: "openai-compatible", BaseURL: "https://api.groq.com/openai/v1", Model: "m1", Auth: "env:GROQ_API_KEY"})
	if err != nil || !added {
		t.Fatalf("premier add doit ajouter: added=%v err=%v", added, err)
	}

	added2, err := upsertModelSpec(modelSpec{Name: "groq", Kind: "openai-compatible", BaseURL: "https://api.groq.com/openai/v1", Model: "m2", Auth: "env:GROQ_API_KEY"})
	if err != nil || added2 {
		t.Fatalf("second add du même nom doit remplacer: added=%v err=%v", added2, err)
	}

	specs, err := loadModelSpecs()
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 1 {
		t.Fatalf("attendu 1 spec (pas de doublon), got %d", len(specs))
	}
	if specs[0].Model != "m2" {
		t.Fatalf("remplacement non effectif: %+v", specs[0])
	}
}
