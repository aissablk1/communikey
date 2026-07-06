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
