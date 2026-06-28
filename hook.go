package main

// hook.go — RÉCEPTION LIVE : faire qu'une session reçoive ses messages sans polling.
//
// `csend hook` est conçu pour être câblé comme hook `UserPromptSubmit` de Claude
// Code : à chaque prompt, il draine l'inbox de la session et imprime les messages
// → ils surgissent dans le contexte de l'agent. C'est ce qui transforme la
// plomberie en conversation. `csend watch` est l'équivalent « tail live » pour un
// humain / un pane.

import (
	"fmt"
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

// cmdHook drains this session's inbox and prints the messages (consuming them so
// each surfaces once). Prints nothing when empty — zero noise in the hook.
func cmdHook(args []string) {
	for _, a := range args {
		if a == "--install" {
			fmt.Println(hookInstall)
			return
		}
	}
	s := mustStore()
	agent := selfAgentID()
	msgs, err := s.Inbox().Receive(agent, true)
	if err != nil || len(msgs) == 0 {
		return
	}
	fmt.Printf("[csend] %d message(s) reçus via le bus inter-agents :\n", len(msgs))
	for _, m := range msgs {
		fmt.Printf("  • de %s : %s\n", m.From, openBody(s, m))
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
