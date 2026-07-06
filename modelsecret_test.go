package main

import "testing"

func TestResolveModelSecretEmpty(t *testing.T) {
	v, err := resolveModelSecret("")
	if err != nil {
		t.Fatalf("auth vide ne doit jamais échouer: %v", err)
	}
	if v != "" {
		t.Fatalf("attendu chaîne vide, got %q", v)
	}
}

func TestResolveModelSecretEnv(t *testing.T) {
	t.Setenv("COMKEY_TEST_HF_KEY", "sk-from-env")
	v, err := resolveModelSecret("env:COMKEY_TEST_HF_KEY")
	if err != nil {
		t.Fatalf("résolution env échouée: %v", err)
	}
	if v != "sk-from-env" {
		t.Fatalf("attendu sk-from-env, got %q", v)
	}
}

func TestResolveModelSecretEnvMissing(t *testing.T) {
	if _, err := resolveModelSecret("env:COMKEY_TEST_ABSENT_VAR"); err == nil {
		t.Fatal("variable d'environnement absente doit échouer explicitement")
	}
}

func TestResolveModelSecretInvalidPrefix(t *testing.T) {
	if _, err := resolveModelSecret("bogus:x"); err == nil {
		t.Fatal("préfixe auth invalide doit échouer explicitement")
	}
}

func TestModelSecretVaultRoundtrip(t *testing.T) {
	t.Setenv("COMKEY_STORE_DIR", t.TempDir())
	t.Setenv("COMKEY_VAULT_PASS", "test-passphrase")

	if err := saveModelSecret("huggingface", "sk-vault-test"); err != nil {
		t.Fatalf("saveModelSecret a échoué: %v", err)
	}

	v, err := resolveModelSecret("vault:huggingface")
	if err != nil {
		t.Fatalf("résolution vault échouée: %v", err)
	}
	if v != "sk-vault-test" {
		t.Fatalf("attendu sk-vault-test, got %q", v)
	}
}

func TestResolveModelSecretVaultMissingKey(t *testing.T) {
	t.Setenv("COMKEY_STORE_DIR", t.TempDir())
	t.Setenv("COMKEY_VAULT_PASS", "test-passphrase")

	if err := saveModelSecret("huggingface", "sk-vault-test"); err != nil {
		t.Fatalf("saveModelSecret a échoué: %v", err)
	}

	if _, err := resolveModelSecret("vault:gemini"); err == nil {
		t.Fatal("clé absente du vault doit échouer explicitement")
	}
}

func TestResolveModelSecretVaultNoPassphrase(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("COMKEY_STORE_DIR", dir)
	t.Setenv("COMKEY_VAULT_PASS", "test-passphrase")
	if err := saveModelSecret("huggingface", "sk-vault-test"); err != nil {
		t.Fatalf("saveModelSecret a échoué: %v", err)
	}

	// Vault existant, mais plus aucune passphrase disponible.
	t.Setenv("COMKEY_VAULT_PASS", "")
	t.Setenv("COMKEY_VAULT_PASS_FILE", "")

	if _, err := resolveModelSecret("vault:huggingface"); err == nil {
		t.Fatal("vault existant sans passphrase disponible doit échouer explicitement")
	}
}
