package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveVaultPass(t *testing.T) {
	os.Unsetenv("COMKEY_VAULT_PASS")
	os.Unsetenv("COMKEY_VAULT_PASS_FILE")
	if _, ok := resolveVaultPass(); ok {
		t.Fatal("ne devrait rien résoudre sans env ni fichier")
	}

	// env fallback
	os.Setenv("COMKEY_VAULT_PASS", "envpass")
	defer os.Unsetenv("COMKEY_VAULT_PASS")
	if p, ok := resolveVaultPass(); !ok || string(p) != "envpass" {
		t.Fatalf("env: got %q ok=%v", p, ok)
	}

	// file wins over env, trailing newline trimmed
	f := filepath.Join(t.TempDir(), "pass.txt")
	if err := os.WriteFile(f, []byte("filepass\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	os.Setenv("COMKEY_VAULT_PASS_FILE", f)
	defer os.Unsetenv("COMKEY_VAULT_PASS_FILE")
	if p, ok := resolveVaultPass(); !ok || string(p) != "filepass" {
		t.Fatalf("le fichier doit primer (sans newline): got %q ok=%v", p, ok)
	}
}
