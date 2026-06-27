package main

import (
	"os"
	"testing"
)

// TestDetectRealIdle runs the detector against a REAL captured idle screen
// (`cmux read-screen surface:42`, a dormant session, 2026-06-27). This is the
// regression test that caught the cursor-byte bug: the empty prompt line is
// `❯ <cursor>`, not `❯ `, so a naive `❯\s*$` failed on real terminals. §2/§29.
func TestDetectRealIdle(t *testing.T) {
	data, err := os.ReadFile("testdata/claude_idle_real.txt")
	if err != nil {
		t.Skipf("fixture absente: %v", err)
	}
	if got := DetectClaudeState(string(data)); got != StateIdle {
		t.Fatalf("écran idle réel classé %s, want idle", got)
	}
}

// fixtureIdle is a REAL Claude Code idle screen, captured 2026-06-27 via
// `cmux read-screen --surface surface:42` on a dormant session (§2 vraies
// données). The tell-tales: an empty `❯` input box framed by `───` rules, plus
// the status footer (`mod:…`, `ctx:…`, `shift+tab to cycle`, `for agents`).
const fixtureIdle = `  Petite nuance honnête : au début je t'avais donné
  61627079-5bed-4c59-8148-2843a44dc4f9, qui est en fait le répertoire où le

✻ Brewed for 1m 28s

※ recap: Tu compares deux projets d'audit WordPress : j'ai livré la comparaison
  fonctionnalités/faiblesses et confirmé ton ID de session cmux.
                                          new task? /clear to save 327.4k toke…
───────────────────────────────────────────────────────────────────────────────
❯
───────────────────────────────────────────────────────────────────────────────
  mod:Opus 4.8 (1M context) · aissabelkoussa · ctx:███░░░░░░░ 33% (670.0k)
  ⏵⏵ bypass permissions on (shift+tab to cycle) · ← for agents`

// fixtureBusy mimics the live footer Claude shows while a turn runs. The stable
// marker is "esc to interrupt". (SYNTHETIC — to be replaced with a real capture
// during the e2e spike, §29.)
const fixtureBusy = `  Je lis les fichiers et je réfléchis à la structure…

✻ Channelling… (12s · ↑ 1.4k tokens · esc to interrupt)
───────────────────────────────────────────────────────────────────────────────
❯
───────────────────────────────────────────────────────────────────────────────
  mod:Opus 4.8 · aissabelkoussa · ctx:███░░░░░░░ 33%`

// fixtureConfirm mimics a Claude permission dialog. Multiple independent signals
// (numbered "1. Yes" menu, "Do you want to proceed", "No, and tell Claude").
// (SYNTHETIC — to be replaced with a real capture during the e2e spike, §29.)
const fixtureConfirm = `╭──────────────────────────────────────────────────────╮
│ Bash command                                         │
│   rm -rf ./build                                     │
│                                                      │
│ Do you want to proceed?                              │
│ ❯ 1. Yes                                             │
│   2. Yes, and don't ask again this session           │
│   3. No, and tell Claude what to do differently      │
╰──────────────────────────────────────────────────────╯`

// fixtureShell is a plain shell prompt — NOT an agent session. Must be Unknown
// so we never inject into a non-Claude surface.
const fixtureShell = `aissabelkoussa@MacBook ~ % ls -la
total 0
drwxr-xr-x  5 aissabelkoussa  staff  160 27 juin 13:57 .
aissabelkoussa@MacBook ~ % `

func TestDetectClaudeState(t *testing.T) {
	cases := []struct {
		name   string
		screen string
		want   State
	}{
		{"idle", fixtureIdle, StateIdle},
		{"busy", fixtureBusy, StateBusy},
		{"confirm", fixtureConfirm, StateAwaitConfirm},
		{"shell", fixtureShell, StateUnknown},
		{"empty", "", StateUnknown},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := DetectClaudeState(c.screen); got != c.want {
				t.Fatalf("DetectClaudeState(%s) = %s, want %s", c.name, got, c.want)
			}
		})
	}
}

// TestConfirmBeatsBusy: a confirm dialog that also contains "esc to interrupt"
// elsewhere must still classify as AwaitConfirm (safety-first ordering — we must
// never auto-submit into a confirmation).
func TestConfirmBeatsBusy(t *testing.T) {
	screen := fixtureConfirm + "\n  (esc to interrupt)"
	if got := DetectClaudeState(screen); got != StateAwaitConfirm {
		t.Fatalf("got %s, want await-confirm (confirm must win over busy)", got)
	}
}
