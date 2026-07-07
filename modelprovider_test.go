package main

import "testing"

func TestBuildModelRegistryNoFile(t *testing.T) {
	t.Setenv("COMKEY_STORE_DIR", t.TempDir())

	providers, issues, err := buildModelRegistry()
	if err != nil {
		t.Fatalf("fichier absent ne doit jamais échouer: %v", err)
	}
	if len(providers) != 0 || len(issues) != 0 {
		t.Fatalf("attendu registre vide, got providers=%v issues=%v", providers, issues)
	}
}

func TestBuildModelRegistryValidEntry(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("COMKEY_STORE_DIR", dir)
	writeModelsJSON(t, dir, `{
		"models": [
			{"name": "ollama", "kind": "openai-compatible", "base_url": "http://localhost:11434/v1", "model": "llama3.2"}
		]
	}`)

	providers, issues, err := buildModelRegistry()
	if err != nil {
		t.Fatalf("registre valide ne doit pas échouer: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("aucune issue attendue, got %v", issues)
	}
	p, ok := findModelProvider(providers, "ollama")
	if !ok {
		t.Fatal("provider ollama attendu dans le registre")
	}
	if p.Name() != "ollama" {
		t.Fatalf("Name() attendu ollama, got %s", p.Name())
	}
}

func TestBuildModelRegistryUnknownKind(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("COMKEY_STORE_DIR", dir)
	writeModelsJSON(t, dir, `{
		"models": [
			{"name": "bad", "kind": "native-huggingface", "base_url": "https://example.com"}
		]
	}`)

	providers, issues, err := buildModelRegistry()
	if err != nil {
		t.Fatalf("kind inconnu ne doit pas être une erreur fatale: %v", err)
	}
	if len(providers) != 0 {
		t.Fatalf("aucun provider attendu, got %v", providers)
	}
	if len(issues) != 1 || issues[0].Name != "bad" {
		t.Fatalf("issue attendue pour 'bad', got %v", issues)
	}
}

func TestBuildModelRegistryUnresolvedSecret(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("COMKEY_STORE_DIR", dir)
	writeModelsJSON(t, dir, `{
		"models": [
			{"name": "hf", "kind": "openai-compatible", "base_url": "https://api-inference.huggingface.co/v1", "auth": "env:COMKEY_TEST_MISSING_HF_KEY"}
		]
	}`)

	providers, issues, err := buildModelRegistry()
	if err != nil {
		t.Fatalf("secret non résolu ne doit pas être une erreur fatale: %v", err)
	}
	if len(providers) != 0 {
		t.Fatalf("aucun provider attendu, got %v", providers)
	}
	if len(issues) != 1 || issues[0].Name != "hf" {
		t.Fatalf("issue attendue pour 'hf', got %v", issues)
	}
}
