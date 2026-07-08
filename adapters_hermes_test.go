package main

import "testing"

// Fixtures SYNTHÉTIQUES — dérivées des tokens vérifiés sur source primaire (cli.py
// 0.14.0 local + repo 0.18.2), PAS des captures d'écran live. L'adaptateur reste
// `provisoire` tant qu'une vraie capture n'a pas confirmé le rendu (ANSI/couleurs).

const hermesConfirmSynthetic = `⚠️  Dangerous Command

  rm -rf /tmp/build

❯ 1. Allow once
  2. Allow for this session
  3. Add to permanent allowlist
  4. Deny

  ↑/↓ to select, Enter to confirm  (28s)
`

const hermesBusySynthetic = `⚕ ❯
msg=interrupt · /queue · /bg · /steer · Ctrl+C cancel
`

const hermesConfirmBeatsBusySynthetic = `⚕ ❯
msg=interrupt · /queue
⚠️  Dangerous Command
❯ 1. Allow once
  4. Deny
`

// Prompt idle Hermes : "❯ " nu (même glyph que Claude), placeholder vide, aucun footer.
const hermesGenericIdleSynthetic = `agent output above

❯
`

func TestHermesConfirm(t *testing.T) {
	if got := newHermesProvider().Detect(hermesConfirmSynthetic); got != StateAwaitConfirm {
		t.Fatalf("Detect(confirm) = %s, want await-confirm", got)
	}
}

func TestHermesBusy(t *testing.T) {
	if got := newHermesProvider().Detect(hermesBusySynthetic); got != StateBusy {
		t.Fatalf("Detect(busy) = %s, want busy", got)
	}
}

// La confirmation prime sur busy (un écran affichant les deux ne doit jamais être
// auto-validé).
func TestHermesConfirmBeatsBusy(t *testing.T) {
	if got := newHermesProvider().Detect(hermesConfirmBeatsBusySynthetic); got != StateAwaitConfirm {
		t.Fatalf("Detect(confirm+busy) = %s, want await-confirm (confirm > busy)", got)
	}
}

// Hermes n'a PAS d'idle détectable (voir adapters_hermes.go) : un prompt idle "❯" nu
// comme un shell nu → StateUnknown, jamais lu comme soumissible.
func TestHermesAbstainsOnIdleAndShell(t *testing.T) {
	if got := newHermesProvider().Detect(hermesGenericIdleSynthetic); got != StateUnknown {
		t.Fatalf("Detect(idle ❯ nu) = %s, want unknown (idle non détecté = sûr)", got)
	}
	if got := newHermesProvider().Detect("user@host:~/project$ "); got != StateUnknown {
		t.Fatalf("Detect(shell nu) = %s, want unknown", got)
	}
}

// Attribution registre : busy est Hermes-distinctif ("⚕"/"msg=interrupt" — revendiqué
// par aucun provider antérieur) → attribué à "hermes". Le confirm est vérifié en état
// (l'attribution de nom peut varier si un provider antérieur revendique aussi un
// dialogue de confirmation ; l'ÉTAT await-confirm, lui, doit tenir).
func TestHermesRegistryAttribution(t *testing.T) {
	if st, name := DetectAnyState(hermesBusySynthetic); st != StateBusy || name != "hermes" {
		t.Fatalf("registre busy → (%s, %q), want (busy, hermes)", st, name)
	}
	if st, _ := DetectAnyState(hermesConfirmSynthetic); st != StateAwaitConfirm {
		t.Fatalf("registre confirm → %s, want await-confirm", st)
	}
}
