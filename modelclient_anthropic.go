package main

// modelclient_anthropic.go — adaptateur ModelProvider NATIF pour l'API Messages
// d'Anthropic (POST /v1/messages, en-têtes x-api-key + anthropic-version). C'est
// le SEUL protocole non-openai du catalogue : la plupart des providers réutilisent
// l'adaptateur openai-compatible, mais Anthropic (et MiniMax, qui expose le même
// format wire sur .../anthropic) parlent Messages — d'où ce client dédié (§1 : un
// seul client par FORMAT, pas par marque). Zéro dépendance externe (stdlib).

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// anthropicAPIVersion est la version d'API envoyée dans l'en-tête obligatoire
// "anthropic-version" (valeur stable documentée par Anthropic).
const anthropicAPIVersion = "2023-06-01"

// anthropicDefaultMaxTokens borne la génération quand aucun override n'est fourni.
// L'API Messages EXIGE max_tokens ; on met un plafond raisonnable, surchargeable.
const anthropicDefaultMaxTokens = 4096

type anthropicModelProvider struct {
	name         string
	baseURL      string
	defaultModel string
	apiKey       string
	httpClient   *http.Client
}

func newAnthropicModelProvider(name, baseURL, defaultModel, apiKey string) *anthropicModelProvider {
	return &anthropicModelProvider{
		name:         name,
		baseURL:      baseURL,
		defaultModel: defaultModel,
		apiKey:       apiKey,
		httpClient:   &http.Client{},
	}
}

func (p *anthropicModelProvider) Name() string { return p.name }

type anthropicReqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model     string                `json:"model"`
	MaxTokens int                   `json:"max_tokens"`
	Messages  []anthropicReqMessage `json:"messages"`
}

type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicResponse struct {
	Content []anthropicContentBlock `json:"content"`
	Error   *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Complete implements ModelProvider via l'API Messages native.
func (p *anthropicModelProvider) Complete(ctx context.Context, prompt string, opts ModelOptions) (string, error) {
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

	reqBody, err := json.Marshal(anthropicRequest{
		Model:     model,
		MaxTokens: anthropicDefaultMaxTokens,
		Messages:  []anthropicReqMessage{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return "", fmt.Errorf("model %s: encodage requête: %w", p.name, err)
	}

	url := p.baseURL + "/v1/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("model %s: requête invalide: %w", p.name, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", anthropicAPIVersion)
	if p.apiKey != "" {
		req.Header.Set("x-api-key", p.apiKey)
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

	var parsed anthropicResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("model %s: réponse illisible: %w", p.name, err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("model %s: erreur provider: %s", p.name, parsed.Error.Message)
	}

	// Concatène les blocs texte (l'API renvoie une liste de blocs typés).
	var out bytes.Buffer
	for _, block := range parsed.Content {
		if block.Type == "text" {
			out.WriteString(block.Text)
		}
	}
	if out.Len() == 0 {
		return "", fmt.Errorf("model %s: réponse vide (aucun bloc texte)", p.name)
	}
	return out.String(), nil
}
