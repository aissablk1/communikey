package main

// modelsecret.go — résolution des secrets déclarés dans models.json ("auth":
// "env:NAME" | "vault:NAME"). Réutilise le vault existant (crypto.go:
// SealVault/OpenVault, Argon2id → AES-256-GCM) via un fichier scellé DISTINCT
// de celui de l'identité — les secrets de provider modèle ne sont pas du
// matériel d'identité (pas de Shamir, pas de signature) : pas de raison de les
// coupler à recovery.go au-delà des deux primitives génériques de crypto.go.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func modelSecretsPath() string {
	return filepath.Join(DefaultStoreDir(), "model-secrets.json")
}

// loadModelSecretsBlob reads the sealed blob, if any. Fichier absent → (nil, nil).
func loadModelSecretsBlob() (*VaultBlob, error) {
	data, err := os.ReadFile(modelSecretsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var b VaultBlob
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, fmt.Errorf("model-secrets.json invalide: %w", err)
	}
	return &b, nil
}

// openModelSecrets decrypts the sealed map. Aucun fichier → map vide, pas une
// erreur. Fichier présent mais vault verrouillé → erreur explicite.
func openModelSecrets() (map[string]string, error) {
	blob, err := loadModelSecretsBlob()
	if err != nil {
		return nil, err
	}
	if blob == nil {
		return map[string]string{}, nil
	}
	pass, ok := resolveVaultPass()
	if !ok {
		return nil, fmt.Errorf("vault verrouillé : définis COMKEY_VAULT_PASS(_FILE)")
	}
	pt, err := OpenVault(blob, pass)
	if err != nil {
		return nil, err
	}
	var secrets map[string]string
	if err := json.Unmarshal(pt, &secrets); err != nil {
		return nil, fmt.Errorf("model-secrets.json corrompu: %w", err)
	}
	return secrets, nil
}

// saveModelSecret seals name=value into the model-secrets vault, merging with
// any existing entries.
func saveModelSecret(name, value string) error {
	pass, ok := resolveVaultPass()
	if !ok {
		return fmt.Errorf("vault verrouillé : définis COMKEY_VAULT_PASS(_FILE) avant d'écrire un secret")
	}
	secrets, err := openModelSecrets()
	if err != nil {
		return err
	}
	secrets[name] = value
	pt, err := json.Marshal(secrets)
	if err != nil {
		return err
	}
	blob, err := SealVault(pt, pass)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(blob, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(DefaultStoreDir(), 0o755); err != nil {
		return err
	}
	return os.WriteFile(modelSecretsPath(), data, 0o600)
}

// resolveModelSecret resolves an "auth" field: "" (no secret), "env:NAME", or
// "vault:NAME". Erreur explicite si la source déclarée ne résout à rien —
// jamais un appel silencieusement non-authentifié à un provider qui l'exige (§29).
func resolveModelSecret(auth string) (string, error) {
	switch {
	case auth == "":
		return "", nil
	case strings.HasPrefix(auth, "env:"):
		name := strings.TrimPrefix(auth, "env:")
		v := os.Getenv(name)
		if v == "" {
			return "", fmt.Errorf("variable d'environnement %s non définie (auth: %q)", name, auth)
		}
		return v, nil
	case strings.HasPrefix(auth, "vault:"):
		name := strings.TrimPrefix(auth, "vault:")
		secrets, err := openModelSecrets()
		if err != nil {
			return "", err
		}
		v, ok := secrets[name]
		if !ok {
			return "", fmt.Errorf("secret %q absent du vault (communikey model secret set %s <valeur>)", name, name)
		}
		return v, nil
	default:
		return "", fmt.Errorf("auth invalide %q (attendu: vide, \"env:NOM\" ou \"vault:NOM\")", auth)
	}
}
