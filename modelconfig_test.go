package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadModelSpecsNoFile(t *testing.T) {
	t.Setenv("COMKEY_STORE_DIR", t.TempDir())

	specs, err := loadModelSpecs()
	if err != nil {
		t.Fatalf("fichier absent ne doit jamais être une erreur: %v", err)
	}
	if specs != nil {
		t.Fatalf("attendu nil, got %v", specs)
	}
}

func writeModelsJSON(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "models.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadModelSpecsValid(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("COMKEY_STORE_DIR", dir)
	writeModelsJSON(t, dir, `{
		"models": [
			{"name": "ollama", "kind": "openai-compatible", "base_url": "http://localhost:11434/v1", "model": "llama3.2"}
		]
	}`)

	specs, err := loadModelSpecs()
	if err != nil {
		t.Fatalf("JSON valide ne doit pas échouer: %v", err)
	}
	if len(specs) != 1 || specs[0].Name != "ollama" || specs[0].Kind != "openai-compatible" {
		t.Fatalf("spec attendue pour ollama, got %+v", specs)
	}
}

func TestLoadModelSpecsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("COMKEY_STORE_DIR", dir)
	writeModelsJSON(t, dir, `{ ceci n'est pas du JSON`)

	if _, err := loadModelSpecs(); err == nil {
		t.Fatal("JSON invalide doit renvoyer une erreur explicite, pas un silence")
	}
}
