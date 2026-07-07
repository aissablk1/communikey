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

// modelUnnamedLabel is the placeholder shown for a models.json entry that has no
// name. Une telle entrée est rejetée par le registre (jamais "actif") ; la clé
// est partagée avec model.go pour que `model list` l'affiche cohéremment.
const modelUnnamedLabel = "(sans nom)"

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
			issues = append(issues, modelRegistryIssue{Name: modelUnnamedLabel, Reason: "name manquant"})
			continue
		}
		switch spec.Kind {
		case "openai-compatible", "anthropic":
			// protocole supporté — le provider est construit plus bas
		default:
			issues = append(issues, modelRegistryIssue{
				Name:   spec.Name,
				Reason: "kind inconnu: " + spec.Kind + ` (supportés : "openai-compatible", "anthropic")`,
			})
			continue
		}
		apiKey, err := resolveModelSecret(spec.Auth)
		if err != nil {
			issues = append(issues, modelRegistryIssue{Name: spec.Name, Reason: err.Error()})
			continue
		}
		// Routing "smart" par protocole : Anthropic (et MiniMax) parlent l'API
		// Messages native ; tout le reste réutilise l'adaptateur openai-compatible.
		switch spec.Kind {
		case "anthropic":
			providers = append(providers, newAnthropicModelProvider(spec.Name, spec.BaseURL, spec.Model, apiKey))
		default: // "openai-compatible"
			providers = append(providers, newOpenAIModelProvider(spec.Name, spec.BaseURL, spec.Model, apiKey))
		}
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
