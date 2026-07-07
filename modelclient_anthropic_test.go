package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAnthropicModelProviderCompleteSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("chemin attendu /v1/messages, got %s", r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Fatalf("x-api-key attendu 'test-key', got %q", got)
		}
		if got := r.Header.Get("anthropic-version"); got == "" {
			t.Fatal("header anthropic-version manquant")
		}
		var req anthropicRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("requête illisible: %v", err)
		}
		if req.Model != "claude-sonnet-4-6" {
			t.Fatalf("modèle attendu claude-sonnet-4-6, got %s", req.Model)
		}
		if req.MaxTokens <= 0 {
			t.Fatalf("max_tokens doit être > 0 (requis par l'API), got %d", req.MaxTokens)
		}
		_ = json.NewEncoder(w).Encode(anthropicResponse{
			Content: []anthropicContentBlock{{Type: "text", Text: "réponse native"}},
		})
	}))
	defer srv.Close()

	p := newAnthropicModelProvider("anthropic", srv.URL, "claude-sonnet-4-6", "test-key")
	got, err := p.Complete(context.Background(), "bonjour", ModelOptions{})
	if err != nil {
		t.Fatalf("Complete a échoué: %v", err)
	}
	if got != "réponse native" {
		t.Fatalf("attendu %q, got %q", "réponse native", got)
	}
}

func TestAnthropicModelProviderCompleteHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"api_error","message":"boom"}}`))
	}))
	defer srv.Close()

	p := newAnthropicModelProvider("anthropic", srv.URL, "claude-sonnet-4-6", "")
	_, err := p.Complete(context.Background(), "bonjour", ModelOptions{})
	if err == nil {
		t.Fatal("attendu une erreur sur HTTP 500")
	}
	if !strings.Contains(err.Error(), "HTTP 500") {
		t.Fatalf("erreur attendue mentionnant HTTP 500, got: %v", err)
	}
}

func TestAnthropicModelProviderCompleteEmptyContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(anthropicResponse{})
	}))
	defer srv.Close()

	p := newAnthropicModelProvider("anthropic", srv.URL, "claude-sonnet-4-6", "")
	_, err := p.Complete(context.Background(), "bonjour", ModelOptions{})
	if err == nil {
		t.Fatal("attendu une erreur sur réponse sans content")
	}
}

// TestAnthropicModelProviderModelOverride vérifie que ModelOptions.Model prime
// sur le modèle par défaut.
func TestAnthropicModelProviderModelOverride(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req anthropicRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Model != "claude-opus-4-8" {
			t.Fatalf("override attendu claude-opus-4-8, got %s", req.Model)
		}
		_ = json.NewEncoder(w).Encode(anthropicResponse{
			Content: []anthropicContentBlock{{Type: "text", Text: "ok"}},
		})
	}))
	defer srv.Close()

	p := newAnthropicModelProvider("anthropic", srv.URL, "claude-sonnet-4-6", "k")
	if _, err := p.Complete(context.Background(), "x", ModelOptions{Model: "claude-opus-4-8"}); err != nil {
		t.Fatalf("Complete override a échoué: %v", err)
	}
}

// TestBuildModelRegistryAnthropicKind vérifie le routing "smart" : une entrée
// kind:"anthropic" construit l'adaptateur NATIF, pas l'openai-compatible.
func TestBuildModelRegistryAnthropicKind(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("COMKEY_STORE_DIR", dir)
	t.Setenv("COMKEY_TEST_ANTHRO", "sk-test")
	writeModelsJSON(t, dir, `{
		"models": [
			{"name": "anthropic", "kind": "anthropic", "base_url": "https://api.anthropic.com", "model": "claude-sonnet-4-6", "auth": "env:COMKEY_TEST_ANTHRO"}
		]
	}`)

	providers, issues, err := buildModelRegistry()
	if err != nil {
		t.Fatalf("registre anthropic ne doit pas échouer: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("aucune issue attendue, got %v", issues)
	}
	p, ok := findModelProvider(providers, "anthropic")
	if !ok {
		t.Fatal("provider anthropic attendu dans le registre")
	}
	if _, isAnthropic := p.(*anthropicModelProvider); !isAnthropic {
		t.Fatalf("routing cassé : attendu *anthropicModelProvider, got %T", p)
	}
}
