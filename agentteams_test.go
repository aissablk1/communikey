package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Fixture = le VRAI config.json capturé le 2026-07-05 (session Claude Code réelle,
// CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1), UUIDs remplacés par des valeurs stables
// pour le test — la STRUCTURE et les noms de champs sont ceux réellement observés,
// rien n'est deviné (§2/§29).
const realAgentTeamConfigFixture = `{
  "name": "session-2a1598bb",
  "createdAt": 1783258279972,
  "leadAgentId": "team-lead@session-2a1598bb",
  "leadSessionId": "2a1598bb-c89b-4929-a3b9-ce8b85b4a482",
  "members": [
    {
      "agentId": "team-lead@session-2a1598bb",
      "name": "team-lead",
      "agentType": "team-lead",
      "joinedAt": 1783258279972,
      "tmuxPaneId": "leader",
      "cwd": "/private/tmp/scratch",
      "subscriptions": [],
      "backendType": "in-process"
    }
  ]
}`

func writeTeamConfig(t *testing.T, home, teamName, content string) {
	t.Helper()
	dir := filepath.Join(home, ".claude", "teams", teamName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverAgentTeamsRealSchema(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeTeamConfig(t, home, "session-2a1598bb", realAgentTeamConfigFixture)

	teams, errs := discoverAgentTeams()
	if len(errs) != 0 {
		t.Fatalf("aucune erreur attendue: %v", errs)
	}
	if len(teams) != 1 {
		t.Fatalf("1 équipe attendue, got %d", len(teams))
	}
	tm := teams[0]
	if tm.Name != "session-2a1598bb" || tm.LeadAgentID != "team-lead@session-2a1598bb" ||
		tm.LeadSessionID != "2a1598bb-c89b-4929-a3b9-ce8b85b4a482" || tm.CreatedAt != 1783258279972 {
		t.Fatalf("champs top-level mal parsés: %+v", tm)
	}
	if len(tm.Members) != 1 {
		t.Fatalf("1 membre attendu, got %d", len(tm.Members))
	}
	m := tm.Members[0]
	if m.AgentID != "team-lead@session-2a1598bb" || m.AgentType != "team-lead" ||
		m.TmuxPaneID != "leader" || m.BackendType != "in-process" {
		t.Fatalf("membre mal parsé: %+v", m)
	}
	if m.Subscriptions == nil || len(m.Subscriptions) != 0 {
		t.Fatalf("subscriptions attendu tableau vide (pas nil), got %v", m.Subscriptions)
	}
}

func TestDiscoverAgentTeamsNoDirYieldsEmpty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	teams, errs := discoverAgentTeams()
	if teams != nil || errs != nil {
		t.Fatalf("attendu (nil, nil) sans ~/.claude/teams, got teams=%v errs=%v", teams, errs)
	}
}

func TestDiscoverAgentTeamsSkipsMalformedButKeepsOthers(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeTeamConfig(t, home, "session-bad", `{ceci n'est pas du json`)
	writeTeamConfig(t, home, "session-good", `{"name":"session-good","members":[]}`)

	teams, errs := discoverAgentTeams()
	if len(errs) != 1 {
		t.Fatalf("1 erreur attendue pour session-bad, got %d: %v", len(errs), errs)
	}
	if len(teams) != 1 || teams[0].Name != "session-good" {
		t.Fatalf("session-good attendue malgré l'erreur de session-bad, got %+v", teams)
	}
}

// Une équipe en cours de nettoyage (dossier créé mais config.json pas encore
// écrit, ou déjà supprimé) ne doit jamais être une erreur — juste ignorée.
func TestDiscoverAgentTeamsIgnoresDirWithoutConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".claude", "teams", "session-vide"), 0o755); err != nil {
		t.Fatal(err)
	}

	teams, errs := discoverAgentTeams()
	if len(errs) != 0 || len(teams) != 0 {
		t.Fatalf("dossier sans config.json ne doit produire ni équipe ni erreur, got teams=%v errs=%v", teams, errs)
	}
}
