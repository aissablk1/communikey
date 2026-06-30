package main

// hook.go — RÉCEPTION LIVE : faire qu'une session reçoive ses messages sans polling.
//
// `csend hook` est conçu pour être câblé comme hook `UserPromptSubmit` de Claude
// Code : à chaque prompt, il draine l'inbox de la session et imprime les messages
// → ils surgissent dans le contexte de l'agent. C'est ce qui transforme la
// plomberie en conversation. `csend watch` est l'équivalent « tail live » pour un
// humain / un pane.

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

const hookInstall = `Câble csend comme hook de réception. Dans ~/.claude/settings.json :

  "hooks": {
    "UserPromptSubmit": [
      { "hooks": [ { "type": "command", "command": "csend hook" } ] }
    ]
  }

→ à chaque prompt, les messages reçus via le bus inter-agents apparaissent dans le
  contexte de la session. (Pose aussi CSEND_AGENT_ID pour une identité stable.)`

// hookPayload est le sous-ensemble du JSON que les CLIs passent au hook sur stdin
// (Claude/Codex : UserPromptSubmit ; Gemini : BeforeAgent). On ne l'EXIGE pas :
// absent → on retombe sur CSEND_AGENT_ID / l'identité courante (rétro-compat).
type hookPayload struct {
	HookEventName string `json:"hook_event_name"`
	SessionID     string `json:"session_id"`
	Cwd           string `json:"cwd"`
}

// hookIdentity dérive une identité d'agent STABLE depuis le payload (session_id),
// pour que la réception « marche toute seule » sans poser CSEND_AGENT_ID à la main.
// C'est LE gap « zéro-config » du chemin hook.
func hookIdentity(p hookPayload) string {
	if env := os.Getenv("CSEND_AGENT_ID"); env != "" {
		return env
	}
	if p.SessionID != "" {
		base := p.SessionID
		if len(base) > 8 {
			base = base[:8]
		}
		return "sess-" + base
	}
	return selfAgentID()
}

// readHookStdin lit (sans bloquer sur un terminal interactif) le payload du hook.
func readHookStdin() hookPayload {
	var p hookPayload
	if fi, _ := os.Stdin.Stat(); fi != nil && (fi.Mode()&os.ModeCharDevice) == 0 {
		if data, err := io.ReadAll(io.LimitReader(os.Stdin, 1<<20)); err == nil && len(data) > 0 {
			_ = json.Unmarshal(data, &p)
		}
	}
	return p
}

// cmdHook drains this session's inbox and emits the messages as the agent's added
// context (consuming them so each surfaces once). Prints nothing when empty — zéro
// bruit dans le hook. Provider-aware : émet la forme attendue par chaque CLI.
func cmdHook(args []string) {
	for _, a := range args {
		if a == "--install" {
			fmt.Println(hookInstall)
			return
		}
	}
	p := readHookStdin()
	s := mustStore()
	agent := hookIdentity(p)
	msgs, err := s.Inbox().Receive(agent, true)
	if err != nil || len(msgs) == 0 {
		return
	}
	text := fmt.Sprintf("[csend] %d message(s) reçus via le bus inter-agents :\n", len(msgs))
	for _, m := range msgs {
		who := m.From
		if m.Provider != "" {
			who = m.Provider + ":" + m.From // cross-vendor visible dans le contexte injecté
		}
		text += fmt.Sprintf("  • de %s : %s\n", who, openBody(s, m))
	}
	switch p.HookEventName {
	case "BeforeAgent": // Gemini : contexte additionnel via stdout brut
		fmt.Print(text)
	default: // Claude / Codex : hookSpecificOutput.additionalContext
		b, _ := json.Marshal(map[string]any{
			"hookSpecificOutput": map[string]any{
				"hookEventName":     "UserPromptSubmit",
				"additionalContext": text,
			},
		})
		fmt.Println(string(b))
	}
}

// cmdWatch tails the inbox live, printing new messages as they arrive.
func cmdWatch(args []string) {
	interval := 2 * time.Second
	for i := 0; i < len(args); i++ {
		if args[i] == "--interval" && i+1 < len(args) {
			var n int
			if _, e := fmt.Sscanf(args[i+1], "%d", &n); e == nil && n > 0 {
				interval = time.Duration(n) * time.Second
			}
			i++
		}
	}
	s := mustStore()
	agent := selfAgentID()
	fmt.Printf("csend watch — inbox de %s (Ctrl+C pour arrêter)\n", agent)
	for {
		if msgs, _ := s.Inbox().Receive(agent, true); len(msgs) > 0 {
			for _, m := range msgs {
				fmt.Printf("[%s] %s : %s\n", m.TS, m.From, openBody(s, m))
			}
		}
		time.Sleep(interval)
	}
}
