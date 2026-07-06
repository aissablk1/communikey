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
