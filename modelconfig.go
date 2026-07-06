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
