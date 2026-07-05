package main

// agentteams.go — découverte LECTURE SEULE des Agent Teams natifs de Claude Code.
//
// Périmètre volontairement borné : ce fichier ne fait QUE lister les équipes et
// leurs membres (visibilité, comme `communikey agents`). Il n'écrit JAMAIS dans
// ~/.claude/teams/ (la doc officielle l'interdit explicitement : « don't edit it
// by hand — your changes are overwritten »), et ne relaie PAS encore de messages
// dans la mailbox de l'équipe — ce format n'est pas documenté publiquement et
// n'a jamais été observé sur une vraie session (aucun teammate n'a rejoint avant
// que la session de capture ne se bloque sur le dialogue de confiance du
// dossier, 2026-07-05). router.go réserve déjà la voie (ChannelBridge,
// TargetInfo.Bridge) pour le jour où la livraison sera possible ; ce fichier
// fournit la DÉCOUVERTE, pas encore la livraison.
//
// Schéma vérifié sur un VRAI fichier capturé le 2026-07-05 (session Claude Code
// réelle, CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1, pseudo-tty), pas deviné (§2/§29).
// Doc officielle : code.claude.com/docs/en/agent-teams — « the team config
// contains a members array with each teammate's name, agent ID, and agent
// type » ; « teammates can read this file to discover other team members ».

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// AgentTeamMember mirrors one entry of config.json's "members" array — noms de
// champs exacts (camelCase) vérifiés sur un vrai fichier, pas devinés.
type AgentTeamMember struct {
	AgentID       string   `json:"agentId"`
	Name          string   `json:"name"`
	AgentType     string   `json:"agentType"`
	JoinedAt      int64    `json:"joinedAt"` // epoch millis
	TmuxPaneID    string   `json:"tmuxPaneId"`
	Cwd           string   `json:"cwd"`
	Subscriptions []string `json:"subscriptions"`
	BackendType   string   `json:"backendType"`
}

// AgentTeamConfig mirrors ~/.claude/teams/{team-name}/config.json.
type AgentTeamConfig struct {
	Name          string            `json:"name"`
	CreatedAt     int64             `json:"createdAt"`
	LeadAgentID   string            `json:"leadAgentId"`
	LeadSessionID string            `json:"leadSessionId"`
	Members       []AgentTeamMember `json:"members"`
}

// claudeTeamsDir is Claude Code's OWN path (pas celui de communikey, pas
// overridable via COMKEY_STORE_DIR) — ~/.claude/teams, documenté officiellement.
func claudeTeamsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "teams")
}

// discoverAgentTeams liste chaque Agent Team active en lisant (jamais en
// écrivant) ~/.claude/teams/*/config.json. Un fichier absent/malformé pour UNE
// équipe (ex. en cours de nettoyage) n'empêche jamais de voir les autres — les
// erreurs sont remontées, jamais avalées en silence (§29).
func discoverAgentTeams() ([]AgentTeamConfig, []error) {
	root := claudeTeamsDir()
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, []error{err}
	}
	var teams []AgentTeamConfig
	var errs []error
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(root, e.Name(), "config.json")
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue // équipe en cours de création/nettoyage — pas une erreur
			}
			errs = append(errs, fmt.Errorf("%s: %w", path, err))
			continue
		}
		var cfg AgentTeamConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			errs = append(errs, fmt.Errorf("%s: config.json invalide: %w", path, err))
			continue
		}
		teams = append(teams, cfg)
	}
	return teams, errs
}

// cmdTeams : communikey teams — liste les Agent Teams natives actives et leurs
// membres. Lecture seule ; pas de livraison de message (voir l'en-tête).
func cmdTeams(args []string) {
	teams, errs := discoverAgentTeams()
	for _, e := range errs {
		fmt.Fprintf(os.Stderr, "communikey: %v\n", e)
	}
	if len(teams) == 0 {
		fmt.Println("Aucune Agent Team native active (CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS doit " +
			"être activé et une équipe en cours ailleurs — voir docs/cross-vendor-setup.md).")
		return
	}
	for _, team := range teams {
		fmt.Printf("Équipe %s (lead: %s) :\n", team.Name, team.LeadAgentID)
		for _, m := range team.Members {
			fmt.Printf("  • %-24s %-12s %s\n", m.Name, m.AgentType, m.Cwd)
		}
	}
}
