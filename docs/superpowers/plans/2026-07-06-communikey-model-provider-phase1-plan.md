# communikey — Client multi-provider de modèles (Phase 1) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Donner à communikey une commande `model` capable d'appeler directement un backend
d'inférence (Ollama, LocalAI, HuggingFace Inference API…) déclaré dans un fichier de config JSON,
sans toucher à l'orchestration de sessions CLI existante.

**Architecture:** Une interface `ModelProvider` (distincte de `Provider`), un registre construit
**uniquement** depuis `~/.claude/communikey/models.json` (rien en dur, contrairement au registre
`Provider`), un seul adaptateur v1 (« compatible OpenAI », `kind: "openai-compatible"`), des
secrets scellés dans le vault existant (`SealVault`/`OpenVault` de `crypto.go`), et trois
sous-commandes CLI (`list`, `test`, `call`) plus une commande d'administration (`secret set`).

**Tech Stack:** Go 1.25 stdlib uniquement (`net/http`, `encoding/json`, `context`) — aucune
nouvelle dépendance dans `go.mod`.

## Global Constraints

- Go **1.25.0** (`go.mod`), module `github.com/aissablk1/communikey`, **package `main`** unique —
  toute erreur de compilation dans un fichier fait échouer `go test ./...` pour tout le paquet.
- **Zéro nouvelle dépendance externe** — `net/http`/`encoding/json`/`context`/`time` stdlib
  seulement ; ne pas toucher `go.mod`.
- **Un seul `kind` supporté en v1 : `"openai-compatible"`.** Tout autre `kind` est une **issue**
  reportée (jamais une erreur fatale globale, jamais un crash).
- **`models.json` absent = zéro provider** (zéro-config rétrocompatible, comme `providers.json`).
- **Secrets** : réutiliser exclusivement `SealVault`/`OpenVault` (`crypto.go`, Argon2id →
  AES-256-GCM) et `resolveVaultPass()` (`bus.go`) — **aucune nouvelle primitive crypto**.
- **Erreurs toujours explicites** via `fail()` (`main.go`) — jamais une chaîne vide qui ressemble
  à une réponse valide, jamais un échec silencieux.
- **Tests** : `httptest.Server` uniquement — **jamais** de vrai appel réseau dans `go test ./...`.
- **Portée strictement Phase 1** — pas de streaming, pas d'autre `kind`, pas de hooks enrichis, pas
  de workers-comme-nœuds-du-bus (Phases 2/3, hors de ce plan).
- **Style de commit** : conventionnel, message en français, **aucune mention d'IA/co-autorat**
  (§21 — pas de `Co-Authored-By` IA).

---

## Task 1: Parsing déclaratif de `models.json`

**Files:**
- Create: `modelconfig.go`
- Test: `modelconfig_test.go`

**Interfaces:**
- Consumes: `DefaultStoreDir() string` (`memory.go`, existant).
- Produces: `type modelSpec struct { Name, Kind, BaseURL, Model, Auth string }` (tags JSON :
  `name`, `kind`, `base_url`, `model`, `auth,omitempty`) ; `func modelsConfigPath() string` ;
  `func loadModelSpecs() ([]modelSpec, error)` ; helper de test `writeModelsJSON(t, dir, content
  string)` (réutilisé par les tâches suivantes).

- [ ] **Step 1: Écrire le test qui échoue**

Créer `modelconfig_test.go` :

```go
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
```

- [ ] **Step 2: Vérifier que ça échoue**

Run: `go test ./... -run TestLoadModelSpecs -v`
Expected: FAIL à la compilation — `undefined: loadModelSpecs` (et `undefined: modelSpec` selon
l'ordre de résolution). Normal : `modelconfig.go` n'existe pas encore, et le paquet `main` compile
tous les fichiers ensemble.

- [ ] **Step 3: Implémentation minimale**

Créer `modelconfig.go` :

```go
package main

// modelconfig.go — parsing de ~/.claude/communikey/models.json (déclaratif,
// miroir de providerconfig.go). Fichier absent = zéro provider de modèle par
// défaut (rétro-compatible, zéro-config). JSON invalide = erreur EXPLICITE
// (jamais un échec silencieux, §29). Zéro dépendance externe (JSON stdlib).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// modelSpec is one entry of models.json.
type modelSpec struct {
	Name    string `json:"name"`
	Kind    string `json:"kind"`            // seule valeur supportée v1 : "openai-compatible"
	BaseURL string `json:"base_url"`        // ex. "http://localhost:11434/v1"
	Model   string `json:"model"`           // modèle par défaut si ModelOptions.Model est vide
	Auth    string `json:"auth,omitempty"`  // "", "env:NOM", ou "vault:NOM"
}

func modelsConfigPath() string {
	return filepath.Join(DefaultStoreDir(), "models.json")
}

// loadModelSpecs reads models.json. Fichier absent → (nil, nil). JSON invalide
// → erreur explicite.
func loadModelSpecs() ([]modelSpec, error) {
	data, err := os.ReadFile(modelsConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var doc struct {
		Models []modelSpec `json:"models"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("models.json invalide: %w", err)
	}
	return doc.Models, nil
}
```

- [ ] **Step 4: Vérifier que ça passe**

Run: `go test ./... -run TestLoadModelSpecs -v`
Expected: `PASS` — les 3 tests (`TestLoadModelSpecsNoFile`, `TestLoadModelSpecsValid`,
`TestLoadModelSpecsInvalidJSON`) verts.

- [ ] **Step 5: Commit**

```bash
git add modelconfig.go modelconfig_test.go
git commit -m "feat(model): parsing declaratif de models.json"
```

---

## Task 2: Interface `ModelProvider` + adaptateur compatible OpenAI

**Files:**
- Create: `modelprovider.go` (interface + options, scaffold sans test dédié — voir note)
- Create: `modelclient_openai.go`
- Test: `modelclient_openai_test.go`

**Interfaces:**
- Consumes: rien de nouveau (fichiers autonomes).
- Produces: `type ModelOptions struct { Model string; Timeout time.Duration }` ;
  `type ModelProvider interface { Name() string; Complete(ctx context.Context, prompt string, opts ModelOptions) (string, error) }` ;
  `const modelDefaultTimeout = 30 * time.Second` ;
  `func newOpenAIModelProvider(name, baseURL, defaultModel, apiKey string) *openAIModelProvider`
  (implémente `ModelProvider`).

> Note : `modelprovider.go` (Step 1 ci-dessous) est un scaffold pur (une interface Go n'a pas de
> comportement à tester tant que rien ne l'implémente) — le cycle TDD rouge/vert démarre au
> Step 2, sur l'adaptateur qui la consomme réellement.

- [ ] **Step 1: Créer l'interface (scaffold)**

Créer `modelprovider.go` :

```go
package main

// modelprovider.go — couche "modèles" : interface pluggable pour consommer un
// backend d'inférence (Ollama, LocalAI, HuggingFace…). Contrairement à
// provider.go (détection d'état d'un CLI par lecture d'écran), il n'existe ici
// aucun détecteur "éprouvé sur écrans réels" à figer en dur : tout provider de
// modèle est déclaré dans ~/.claude/communikey/models.json (modelconfig.go).
// Rien n'est enregistré par défaut — fichier absent = zéro provider.

import (
	"context"
	"time"
)

// modelDefaultTimeout is used when ModelOptions.Timeout is zero.
const modelDefaultTimeout = 30 * time.Second

// ModelOptions are per-call overrides.
type ModelOptions struct {
	Model   string        // override du modèle par défaut du spec ; "" = défaut du spec
	Timeout time.Duration // 0 = modelDefaultTimeout
}

// ModelProvider calls a language-model backend and returns generated text.
// Distinct de Provider (provider.go) : ModelProvider PARLE à un modèle, Provider
// DÉTECTE l'état d'un CLI par lecture d'écran — deux couches, pas de confusion
// de vocabulaire dans le code.
type ModelProvider interface {
	Name() string
	Complete(ctx context.Context, prompt string, opts ModelOptions) (string, error)
}
```

- [ ] **Step 2: Écrire le test qui échoue**

Créer `modelclient_openai_test.go` :

```go
package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAIModelProviderCompleteSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openAIChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("requête illisible côté serveur: %v", err)
		}
		if req.Model != "llama3.2" {
			t.Fatalf("modèle attendu llama3.2, got %s", req.Model)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization attendu 'Bearer test-key', got %q", got)
		}
		_ = json.NewEncoder(w).Encode(openAIChatResponse{
			Choices: []struct {
				Message openAIChatMessage `json:"message"`
			}{{Message: openAIChatMessage{Role: "assistant", Content: "réponse de test"}}},
		})
	}))
	defer srv.Close()

	p := newOpenAIModelProvider("test", srv.URL, "llama3.2", "test-key")
	got, err := p.Complete(context.Background(), "bonjour", ModelOptions{})
	if err != nil {
		t.Fatalf("Complete a échoué: %v", err)
	}
	if got != "réponse de test" {
		t.Fatalf("attendu %q, got %q", "réponse de test", got)
	}
}

func TestOpenAIModelProviderCompleteHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "boom"}`))
	}))
	defer srv.Close()

	p := newOpenAIModelProvider("test", srv.URL, "llama3.2", "")
	_, err := p.Complete(context.Background(), "bonjour", ModelOptions{})
	if err == nil {
		t.Fatal("attendu une erreur sur HTTP 500")
	}
	if !strings.Contains(err.Error(), "HTTP 500") {
		t.Fatalf("erreur attendue mentionnant HTTP 500, got: %v", err)
	}
}

func TestOpenAIModelProviderCompleteEmptyChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(openAIChatResponse{})
	}))
	defer srv.Close()

	p := newOpenAIModelProvider("test", srv.URL, "llama3.2", "")
	_, err := p.Complete(context.Background(), "bonjour", ModelOptions{})
	if err == nil {
		t.Fatal("attendu une erreur sur réponse sans choices")
	}
}
```

- [ ] **Step 3: Vérifier que ça échoue**

Run: `go test ./... -run TestOpenAIModelProvider -v`
Expected: FAIL à la compilation — `undefined: newOpenAIModelProvider`, `undefined: openAIChatRequest`,
`undefined: openAIChatResponse`, `undefined: openAIChatMessage`.

- [ ] **Step 4: Implémentation minimale**

Créer `modelclient_openai.go` :

```go
package main

// modelclient_openai.go — adaptateur ModelProvider générique "compatible
// OpenAI" (Chat Completions). Couvre Ollama et LocalAI nativement, et tout
// autre endpoint qui parle le même format déclaré via models.json — sans
// code Go supplémentaire (§1 : ne pas réinventer un client par marque).

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type openAIModelProvider struct {
	name         string
	baseURL      string
	defaultModel string
	apiKey       string
	httpClient   *http.Client
}

func newOpenAIModelProvider(name, baseURL, defaultModel, apiKey string) *openAIModelProvider {
	return &openAIModelProvider{
		name:         name,
		baseURL:      baseURL,
		defaultModel: defaultModel,
		apiKey:       apiKey,
		httpClient:   &http.Client{},
	}
}

func (p *openAIModelProvider) Name() string { return p.name }

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatRequest struct {
	Model    string              `json:"model"`
	Messages []openAIChatMessage `json:"messages"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message openAIChatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Complete implements ModelProvider.
func (p *openAIModelProvider) Complete(ctx context.Context, prompt string, opts ModelOptions) (string, error) {
	model := opts.Model
	if model == "" {
		model = p.defaultModel
	}
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = modelDefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	reqBody, err := json.Marshal(openAIChatRequest{
		Model:    model,
		Messages: []openAIChatMessage{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return "", fmt.Errorf("model %s: encodage requête: %w", p.name, err)
	}

	url := p.baseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("model %s: requête invalide: %w", p.name, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("model %s: appel réseau échoué (%s): %w", p.name, url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("model %s: lecture réponse: %w", p.name, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("model %s: HTTP %d: %s", p.name, resp.StatusCode, string(body))
	}

	var parsed openAIChatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("model %s: réponse illisible: %w", p.name, err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("model %s: erreur provider: %s", p.name, parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("model %s: réponse vide (aucun choice)", p.name)
	}
	return parsed.Choices[0].Message.Content, nil
}
```

- [ ] **Step 5: Vérifier que ça passe**

Run: `go test ./... -run TestOpenAIModelProvider -v`
Expected: `PASS` — les 3 tests verts.

- [ ] **Step 6: Commit**

```bash
git add modelprovider.go modelclient_openai.go modelclient_openai_test.go
git commit -m "feat(model): adaptateur ModelProvider compatible OpenAI"
```

---

## Task 3: Secrets de provider via le vault existant

**Files:**
- Create: `modelsecret.go`
- Test: `modelsecret_test.go`

**Interfaces:**
- Consumes: `resolveVaultPass() ([]byte, bool)` (`bus.go`, existant) ; `SealVault(plaintext,
  passphrase []byte) (*VaultBlob, error)`, `OpenVault(b *VaultBlob, passphrase []byte) ([]byte,
  error)`, `type VaultBlob struct { Salt, Nonce, Ct []byte }` (`crypto.go`, existant) ;
  `DefaultStoreDir() string` (`memory.go`, existant).
- Produces: `func resolveModelSecret(auth string) (string, error)` ; `func saveModelSecret(name,
  value string) error` ; `func modelSecretsPath() string`.

- [ ] **Step 1: Écrire le test qui échoue**

Créer `modelsecret_test.go` :

```go
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
```

- [ ] **Step 2: Vérifier que ça échoue**

Run: `go test ./... -run TestResolveModelSecret -v`
Expected: FAIL à la compilation — `undefined: resolveModelSecret`, `undefined: saveModelSecret`.

- [ ] **Step 3: Implémentation minimale**

Créer `modelsecret.go` :

```go
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
```

- [ ] **Step 4: Vérifier que ça passe**

Run: `go test ./... -run TestResolveModelSecret -v` puis `go test ./... -run TestModelSecretVaultRoundtrip -v`
Expected: `PASS` — les 7 tests verts.

- [ ] **Step 5: Commit**

```bash
git add modelsecret.go modelsecret_test.go
git commit -m "feat(model): secrets de provider via le vault existant"
```

---

## Task 4: Construction du registre (`buildModelRegistry`)

**Files:**
- Modify: `modelprovider.go` (ajoute le registre à la suite de l'interface définie en Task 2)
- Test: `modelprovider_test.go`

**Interfaces:**
- Consumes: `loadModelSpecs() ([]modelSpec, error)` (Task 1) ; `resolveModelSecret(auth string)
  (string, error)` (Task 3) ; `newOpenAIModelProvider(name, baseURL, defaultModel, apiKey string)
  *openAIModelProvider` + `ModelProvider` (Task 2).
- Produces: `type modelRegistryIssue struct { Name, Reason string }` ; `func buildModelRegistry()
  ([]ModelProvider, []modelRegistryIssue, error)` ; `func findModelProvider(providers
  []ModelProvider, name string) (ModelProvider, bool)`.

- [ ] **Step 1: Écrire le test qui échoue**

Créer `modelprovider_test.go` (réutilise `writeModelsJSON` défini dans `modelconfig_test.go`,
même paquet de test) :

```go
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
```

- [ ] **Step 2: Vérifier que ça échoue**

Run: `go test ./... -run TestBuildModelRegistry -v`
Expected: FAIL à la compilation — `undefined: buildModelRegistry`, `undefined: findModelProvider`.

- [ ] **Step 3: Implémentation minimale**

Ajouter à la fin de `modelprovider.go` (après l'interface du Task 2) :

```go
// modelRegistryIssue records one models.json entry that failed to become a
// live ModelProvider — reported by `model list`, never silent (§29).
type modelRegistryIssue struct {
	Name   string
	Reason string
}

// buildModelRegistry loads models.json and constructs a ModelProvider per valid
// entry. Une entrée invalide (kind inconnu, secret non résolu) est signalée dans
// issues et SAUTÉE — elle n'empêche jamais les autres entrées de charger (même
// résilience que loadUserProviders en provider.go).
func buildModelRegistry() ([]ModelProvider, []modelRegistryIssue, error) {
	specs, err := loadModelSpecs()
	if err != nil {
		return nil, nil, err
	}
	var providers []ModelProvider
	var issues []modelRegistryIssue
	for _, spec := range specs {
		if spec.Name == "" {
			issues = append(issues, modelRegistryIssue{Name: "(sans nom)", Reason: "name manquant"})
			continue
		}
		if spec.Kind != "openai-compatible" {
			issues = append(issues, modelRegistryIssue{
				Name:   spec.Name,
				Reason: "kind inconnu: " + spec.Kind + ` (seul "openai-compatible" est supporté)`,
			})
			continue
		}
		apiKey, err := resolveModelSecret(spec.Auth)
		if err != nil {
			issues = append(issues, modelRegistryIssue{Name: spec.Name, Reason: err.Error()})
			continue
		}
		providers = append(providers, newOpenAIModelProvider(spec.Name, spec.BaseURL, spec.Model, apiKey))
	}
	return providers, issues, nil
}

// findModelProvider returns the named provider, or (nil,false).
func findModelProvider(providers []ModelProvider, name string) (ModelProvider, bool) {
	for _, p := range providers {
		if p.Name() == name {
			return p, true
		}
	}
	return nil, false
}
```

- [ ] **Step 4: Vérifier que ça passe**

Run: `go test ./... -run TestBuildModelRegistry -v`
Expected: `PASS` — les 4 tests verts.

- [ ] **Step 5: Commit**

```bash
git add modelprovider.go modelprovider_test.go
git commit -m "feat(model): construction du registre de providers de modele"
```

---

## Task 5: Sous-commandes CLI `model list|test|call|secret set`

**Files:**
- Create: `model.go`
- Modify: `main.go` (ajout du `case "model":` dans le switch de `main()`, ligne de `usage`)

**Interfaces:**
- Consumes: `buildModelRegistry()`, `findModelProvider()` (Task 4) ; `loadModelSpecs()`,
  `modelsConfigPath()` (Task 1) ; `saveModelSecret()`, `modelSecretsPath()` (Task 3) ;
  `ModelOptions`, `modelDefaultTimeout` (Task 2) ; `fail(msg string)` (`main.go`, existant) ;
  `wantJSON(args []string) bool`, `emitJSON(v any)` (`jsonout.go`, existant).
- Produces: `func cmdModel(args []string)`.

> Note sur les tests : dans ce projet, les fonctions `cmd*` (plomberie CLI, appellent `fail()` qui
> fait `os.Exit(1)`) ne sont **jamais** testées directement — vérifié par `grep` (`cmdProvider`,
> `cmdHook`, `cmdJournal` n'apparaissent dans aucun `_test.go`). Ce Task suit la même convention :
> la logique testable est déjà couverte par les Tasks 1-4 ; ce Task est vérifié **manuellement**
> (Step 4), comme `provider test` l'est déjà.

- [ ] **Step 1: Créer `model.go`**

```go
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
```

- [ ] **Step 2: Câbler dans `main.go`**

Dans `main.go`, ajouter juste après `case "provider": cmdProvider(os.Args[2:])` (repère existant,
ligne ~80) :

```go
	case "model":
		cmdModel(os.Args[2:]) // consomme un backend de modèle (Ollama, LocalAI, HuggingFace…) — indépendant du backend terminal
```

Dans la constante `usage` de `main.go`, ajouter après la ligne `communikey provider test <name> …` :

```
  communikey model list                 providers de modèle configurés (models.json)
  communikey model test <name>          vérifie qu'un provider de modèle répond
  communikey model call <name> "<p>"    appelle un provider de modèle (prompt en argument ou stdin)
  communikey model secret set <n> <v>   enregistre un secret de provider dans le vault
```

- [ ] **Step 3: Vérifier que ça compile**

Run: `go build ./...`
Expected: aucune sortie, code de sortie 0.

- [ ] **Step 4: Vérification manuelle (pas de test automatisé — convention du projet)**

```bash
# 1. Config vide → message explicite, pas de crash
COMKEY_STORE_DIR=$(mktemp -d) go run . model list
# Expected: "Aucun provider de modèle configuré (…/models.json absent ou vide)."

# 2. Avec un Ollama réellement lancé en local (si disponible) :
mkdir -p ~/.claude/communikey
cat > ~/.claude/communikey/models.json <<'EOF'
{"models":[{"name":"ollama","kind":"openai-compatible","base_url":"http://localhost:11434/v1","model":"llama3.2"}]}
EOF
go run . model list
# Expected: une ligne "ollama … actif secret=aucun"
go run . model test ollama
# Expected (Ollama lancé) : "✓ provider ollama répond"
go run . model call ollama "dis bonjour en un mot"
# Expected: une réponse texte non vide
```

- [ ] **Step 5: Commit**

```bash
git add model.go main.go
git commit -m "feat(model): sous-commandes CLI model list|test|call|secret set"
```

---

## Task 6: Vérification finale + documentation

**Files:**
- Modify: `CHANGELOG.md` (nouvelle entrée)
- Modify: `README.md` (mention de `communikey model`, si le README liste déjà les sous-commandes)

**Interfaces:** aucune nouvelle — cette tâche ne fait que vérifier et documenter ce qui précède.

> Avant de toucher `CHANGELOG.md`/`README.md` : `git pull`/`git status` d'abord — ces deux
> fichiers sont des cibles fréquentes d'autres sessions actives sur ce dépôt (§7). Ne committer
> que les lignes ajoutées par ce plan, jamais un `git add -A`.

- [ ] **Step 1: Suite complète**

Run: `go vet ./...`
Expected: aucune sortie, code de sortie 0.

Run: `go build ./...`
Expected: aucune sortie, code de sortie 0.

Run: `go test ./...`
Expected: `ok  	github.com/aissablk1/communikey	<durée>` — tous les tests des Tasks 1-4 verts,
zéro appel réseau réel (uniquement `httptest.Server`).

- [ ] **Step 2: CHANGELOG**

Ajouter en tête de `CHANGELOG.md` (sous le dernier titre de version/Unreleased existant — vérifier
le format exact du fichier avant d'insérer, il peut avoir changé depuis ce plan) :

```markdown
### Ajouté
- `communikey model list|test|call` — client multi-provider de modèles (Ollama, LocalAI,
  HuggingFace Inference API et tout endpoint compatible OpenAI), déclaratif via
  `~/.claude/communikey/models.json`, secrets scellés dans le vault existant
  (`communikey model secret set`). Phase 1 : voir
  `docs/superpowers/specs/2026-07-06-communikey-model-provider-design.md`.
```

- [ ] **Step 3: Commit final**

```bash
git add CHANGELOG.md README.md
git commit -m "docs(model): changelog + verification complete (Phase 1)"
```

---

## Self-Review (fait par l'auteur du plan)

- **Couverture du spec** : §4 Architecture → Tasks 1,2,4,5. §5 Composants → Tasks 1,3,5. §6 Flux
  de données → Task 2/4/5. §7 Sécurité → Task 3 + `fail()` partout. §8 Non-goals → aucun task ne
  les implémente (streaming, autres `kind`, hooks, workers — absents, correct). §9 Phases → ce plan
  = Phase 1 uniquement. §10 Tests → couvert par les Steps de chaque Task + Task 6.
- **Scan de placeholders** : aucun "TBD"/"TODO" ; chaque étape a du code complet, jamais une
  description sans implémentation.
- **Cohérence des types** : `ModelProvider.Complete(ctx context.Context, prompt string, opts
  ModelOptions) (string, error)` identique dans `openAIModelProvider`, `buildModelRegistry` et
  `model.go`. `modelSpec{Name,Kind,BaseURL,Model,Auth}` et `modelRegistryIssue{Name,Reason}`
  utilisés de façon cohérente à travers les Tasks 1, 4, 5.

---

**Auteur** : Aïssa BELKOUSSA
