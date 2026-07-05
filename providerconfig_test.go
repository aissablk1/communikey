package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Fichier absent = zéro-config par défaut : aucune erreur, aucun provider.
func TestLoadUserProviderSpecsNoFile(t *testing.T) {
	t.Setenv("COMKEY_STORE_DIR", t.TempDir())

	specs, err := loadUserProviderSpecs()
	if err != nil {
		t.Fatalf("fichier absent ne doit jamais être une erreur: %v", err)
	}
	if specs != nil {
		t.Fatalf("attendu nil, got %v", specs)
	}
}

func writeProvidersJSON(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "providers.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadUserProviderSpecsValid(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("COMKEY_STORE_DIR", dir)
	writeProvidersJSON(t, dir, `{
		"providers": [
			{"name": "opencode", "idle_prompt": "^>\\s*$", "idle_footer": "opencode-\\d"}
		]
	}`)

	specs, err := loadUserProviderSpecs()
	if err != nil {
		t.Fatalf("JSON valide ne doit pas échouer: %v", err)
	}
	if len(specs) != 1 || specs[0].Name != "opencode" {
		t.Fatalf("spec attendue pour opencode, got %+v", specs)
	}
}

func TestLoadUserProviderSpecsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("COMKEY_STORE_DIR", dir)
	writeProvidersJSON(t, dir, `{ ceci n'est pas du JSON`)

	if _, err := loadUserProviderSpecs(); err == nil {
		t.Fatal("JSON invalide doit renvoyer une erreur explicite, pas un silence")
	}
}

func TestCompileProviderSpecNoPatternsSkipped(t *testing.T) {
	p, err, ok := compileProviderSpec(providerSpec{Name: "kiro-cli"})
	if err != nil {
		t.Fatalf("un spec sans pattern n'est pas une erreur: %v", err)
	}
	if ok {
		t.Fatal("un spec sans pattern ne doit rien enregistrer (ok doit être false)")
	}
	if p != nil {
		t.Fatal("aucun Provider attendu")
	}
}

func TestCompileProviderSpecMissingName(t *testing.T) {
	if _, err, _ := compileProviderSpec(providerSpec{IdlePrompt: "^>$"}); err == nil {
		t.Fatal("un provider sans nom doit échouer explicitement")
	}
}

func TestCompileProviderSpecBadRegex(t *testing.T) {
	if _, err, ok := compileProviderSpec(providerSpec{Name: "x", Busy: []string{"(unclosed"}}); err == nil || !ok {
		t.Fatalf("une regex invalide doit échouer explicitement (err=%v ok=%v)", err, ok)
	}
}

// La compilation doit produire un Provider qui détecte RÉELLEMENT selon les mêmes
// règles safety-first que les adaptateurs existants (confirm > busy > idle double
// signal > unknown).
func TestCompileProviderSpecDetectsCorrectly(t *testing.T) {
	spec := providerSpec{
		Name:       "opencode",
		Confirm:    []string{`(?i)proceed\?`},
		Busy:       []string{`(?i)generating`},
		IdlePrompt: `(?m)^>\s*$`,
		IdleFooter: `(?i)opencode-\d`,
	}
	p, err, ok := compileProviderSpec(spec)
	if err != nil || !ok || p == nil {
		t.Fatalf("compilation attendue OK: err=%v ok=%v p=%v", err, ok, p)
	}
	if p.Name() != "opencode" {
		t.Fatalf("nom attendu opencode, got %s", p.Name())
	}
	cases := map[string]State{
		"Proceed? (y/n)":               StateAwaitConfirm,
		"Generating response…":         StateBusy,
		">\nopencode-3":                StateIdle,
		"$ ":                           StateUnknown, // shell nu : jamais idle sans le double signal
	}
	for screen, want := range cases {
		if got := p.Detect(screen); got != want {
			t.Fatalf("Detect(%q) = %v, want %v", screen, got, want)
		}
	}
}

// Un provider utilisateur cassé (regex invalide) ne doit JAMAIS empêcher le
// démarrage ni affecter les autres — juste être ignoré (bruyamment sur stderr,
// mais loadUserProviders() elle-même ne renvoie que les specs valides).
func TestLoadUserProvidersSkipsBrokenEntriesButKeepsValid(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("COMKEY_STORE_DIR", dir)
	writeProvidersJSON(t, dir, `{
		"providers": [
			{"name": "cassé", "busy": ["(unclosed"]},
			{"name": "opencode", "idle_prompt": "^>\\s*$", "idle_footer": "opencode-\\d"}
		]
	}`)

	got := loadUserProviders()
	if len(got) != 1 || got[0].Name() != "opencode" {
		t.Fatalf("attendu 1 provider valide (opencode), got %v", got)
	}
}
