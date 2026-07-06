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

func TestOpenAIModelProviderCompleteJSONErrorField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"error": {"message": "modèle introuvable"}}`))
	}))
	defer srv.Close()

	p := newOpenAIModelProvider("test", srv.URL, "llama3.2", "")
	_, err := p.Complete(context.Background(), "bonjour", ModelOptions{})
	if err == nil {
		t.Fatal("attendu une erreur quand le body JSON contient un champ error, même en HTTP 200")
	}
	if !strings.Contains(err.Error(), "modèle introuvable") {
		t.Fatalf("erreur attendue mentionnant le message provider, got: %v", err)
	}
}
