package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Un repli en clair (pas de contact chiffré connu) doit rester aussi VISIBLE que le
// scellement réussi — jamais un simple silence (audit du 2026-07-03 : le "· chiffré
// E2E" n'apparaissait que côté succès, l'absence de mention était le seul signal
// d'un envoi en clair).
func TestEncryptionLabel(t *testing.T) {
	if got := encryptionLabel(true); got != " · chiffré E2E" {
		t.Fatalf("scellé: got %q", got)
	}
	if got := encryptionLabel(false); got == "" || got == " · chiffré E2E" {
		t.Fatalf("repli en clair doit être explicite et distinct du scellé, got %q", got)
	}
}

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
