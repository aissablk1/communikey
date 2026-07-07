package main

// model_test.go — couvre les deux fonctions pures extraites du layer CLI
// (model.go) : la construction des entrées de `model list` (forme JSON stable,
// statut correct d'une entrée sans nom) et la résolution de la valeur de secret
// (argv hérité vs stdin sécurisé). Le reste de model.go reste un mince wrapper
// autour de fail()/os.Exit, délibérément non testé.

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildModelListEntries_EmptyRendersJSONArray(t *testing.T) {
	entries := buildModelListEntries(nil, nil)
	if entries == nil {
		t.Fatal("attendu une slice non-nil (sinon --json émet null au lieu de [])")
	}
	b, err := json.Marshal(entries)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != "[]" {
		t.Fatalf(`attendu "[]", obtenu %q`, string(b))
	}
}

func TestBuildModelListEntries_NamelessSpecMarkedError(t *testing.T) {
	specs := []modelSpec{{Name: "", Kind: "openai-compatible", BaseURL: "http://x"}}
	issues := []modelRegistryIssue{{Name: modelUnnamedLabel, Reason: "name manquant"}}
	entries := buildModelListEntries(specs, issues)
	if len(entries) != 1 {
		t.Fatalf("attendu 1 entrée, obtenu %d", len(entries))
	}
	if entries[0].Status != "erreur" {
		t.Fatalf(`entrée sans nom attendue "erreur", obtenu %q`, entries[0].Status)
	}
	if entries[0].Reason == "" {
		t.Fatal("attendu une raison non vide pour l'entrée sans nom")
	}
}

func TestBuildModelListEntries_ValidSpecActive(t *testing.T) {
	specs := []modelSpec{{Name: "local", Kind: "openai-compatible", BaseURL: "http://x"}}
	entries := buildModelListEntries(specs, nil)
	if entries[0].Status != "actif" {
		t.Fatalf(`attendu "actif", obtenu %q`, entries[0].Status)
	}
}

func TestBuildModelListEntries_IssueSpecMarkedError(t *testing.T) {
	specs := []modelSpec{{Name: "bad", Kind: "weird"}}
	issues := []modelRegistryIssue{{Name: "bad", Reason: "kind inconnu"}}
	entries := buildModelListEntries(specs, issues)
	if entries[0].Status != "erreur" || entries[0].Reason != "kind inconnu" {
		t.Fatalf("attendu erreur/kind inconnu, obtenu %q/%q", entries[0].Status, entries[0].Reason)
	}
}

func TestBuildModelListEntries_SecretDeclaredVsNone(t *testing.T) {
	specs := []modelSpec{{Name: "a", Auth: "vault:X"}, {Name: "b"}}
	entries := buildModelListEntries(specs, nil)
	if entries[0].Secret != "déclaré" {
		t.Fatalf(`attendu "déclaré", obtenu %q`, entries[0].Secret)
	}
	if entries[1].Secret != "aucun" {
		t.Fatalf(`attendu "aucun", obtenu %q`, entries[1].Secret)
	}
}

func TestResolveSecretValue_FromArgv(t *testing.T) {
	name, value, err := resolveSecretValue([]string{"prov", "sekret"}, strings.NewReader(""))
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if name != "prov" || value != "sekret" {
		t.Fatalf("attendu prov/sekret, obtenu %q/%q", name, value)
	}
}

func TestResolveSecretValue_FromStdinTrimmed(t *testing.T) {
	name, value, err := resolveSecretValue([]string{"prov"}, strings.NewReader("  sekret\n"))
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if name != "prov" || value != "sekret" {
		t.Fatalf("attendu prov/sekret (trim), obtenu %q/%q", name, value)
	}
}

func TestResolveSecretValue_EmptyStdinFails(t *testing.T) {
	if _, _, err := resolveSecretValue([]string{"prov"}, strings.NewReader("   \n")); err == nil {
		t.Fatal("attendu une erreur pour un secret vide")
	}
}

func TestResolveSecretValue_NoNameFails(t *testing.T) {
	if _, _, err := resolveSecretValue(nil, strings.NewReader("x")); err == nil {
		t.Fatal("attendu une erreur sans <name>")
	}
}
