package main

import "testing"

// adapters_lot2_test.go — tests des adaptateurs LOT 2 (aider, goose, opencode,
// crush, qwen-code). Fixtures CONSTRUITES À PARTIR DE TOKENS RÉELS vérifiés sur
// source primaire le 2026-07-08 (voir les commentaires de calibration dans
// adapters.go pour les fichiers exacts de chaque dépôt). La disposition (cadres,
// espaces) est représentative ; ce sont les TOKENS qui sont vérifiés. Garanties :
// (1) safety-first confirm > busy > idle > unknown, (2) double-signal idle,
// (3) abstention sur shell nu ET écran Claude, (4) attribution via DetectAnyState
// quand elle est sans ambiguïté (caveats partagés documentés à part).

// ─── aider (REPL, idle NON détecté par design — voir adapters.go) ───
const aiderBusyReal = `Added main.py to the chat.

⠋ Waiting for gpt-4o`

const aiderConfirmReal = `Run shell command?
  npm test
(Y)es/(N)o/(A)ll/(S)kip all [Yes]: `

// Écran "idle" d'aider : DOIT rester Unknown (prompt générique, aucun footer →
// indistinguable d'un shell, on ne soumet jamais).
const aiderIdleUnknown = `Tokens: 1.2k sent, 340 received. Cost: $0.01 message.

architect> `

func TestAiderProviderDetect(t *testing.T) {
	p := newAiderProvider()
	if p.Name() != "aider" {
		t.Fatalf("Name() = %q, want \"aider\"", p.Name())
	}
	cases := []struct {
		name   string
		screen string
		want   State
	}{
		{"busy", aiderBusyReal, StateBusy},
		{"confirm", aiderConfirmReal, StateAwaitConfirm},
		{"idle-is-unknown", aiderIdleUnknown, StateUnknown}, // safe-by-default
		{"shell", fixtureShell, StateUnknown},
		{"empty", "", StateUnknown},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := p.Detect(c.screen); got != c.want {
				t.Fatalf("aider.Detect(%s) = %s, want %s", c.name, got, c.want)
			}
		})
	}
	// Le busy d'aider ne doit PAS matcher le message idle "waiting for your input".
	if got := p.Detect("Aider is waiting for your input"); got != StateUnknown {
		t.Fatalf("aider a lu 'waiting for your input' comme %s, want unknown", got)
	}
}

// ─── goose (REPL rustyline) ───
const gooseIdleReal = `goose is ready

>
Enter to send · Ctrl+J newline`

const gooseBusyReal = `Waddling to conclusions...  (Ctrl+C to interrupt)`

const gooseConfirmReal = `Goose would like to call the above tool, do you allow?
> Allow
  Always Allow
  Deny
  Cancel`

func TestGooseProviderDetect(t *testing.T) {
	p := newGooseProvider()
	if p.Name() != "goose" {
		t.Fatalf("Name() = %q, want \"goose\"", p.Name())
	}
	cases := []struct {
		name   string
		screen string
		want   State
	}{
		{"idle", gooseIdleReal, StateIdle},
		{"busy", gooseBusyReal, StateBusy},
		{"confirm", gooseConfirmReal, StateAwaitConfirm},
		{"shell", fixtureShell, StateUnknown},
		{"empty", "", StateUnknown},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := p.Detect(c.screen); got != c.want {
				t.Fatalf("goose.Detect(%s) = %s, want %s", c.name, got, c.want)
			}
		})
	}
}

// ─── opencode (TUI SolidJS) ───
const opencodeIdleReal = `╹ Ask anything... "Fix a TODO in the codebase"

  tab agents    ctrl+p commands    /status    3 LSP`

const opencodeBusyReal = `╹ streaming response…

  esc interrupt    2 MCP`

const opencodeConfirmReal = `△ Permission required

Tool: bash
  Allow once
  Allow always
  Reject`

func TestOpencodeProviderDetect(t *testing.T) {
	p := newOpencodeProvider()
	if p.Name() != "opencode" {
		t.Fatalf("Name() = %q, want \"opencode\"", p.Name())
	}
	cases := []struct {
		name   string
		screen string
		want   State
	}{
		{"idle", opencodeIdleReal, StateIdle},
		{"busy", opencodeBusyReal, StateBusy},
		{"confirm", opencodeConfirmReal, StateAwaitConfirm},
		{"shell", fixtureShell, StateUnknown},
		{"empty", "", StateUnknown},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := p.Detect(c.screen); got != c.want {
				t.Fatalf("opencode.Detect(%s) = %s, want %s", c.name, got, c.want)
			}
		})
	}
}

// ─── crush (TUI Bubbletea v2) ───
const crushIdleReal = `  CRUSH    Charm™

  > Ready for instructions
  tab focus chat    ctrl+p commands    ctrl+l models`

const crushBusyReal = `  Thinking...

  > Working...
  esc cancel    ctrl+g more`

const crushConfirmReal = `╭ Permission Required ────╮
│ Tool: bash              │
│ Path: ./build           │
│                         │
│   Allow                 │
│   Allow for Session     │
│   Deny                  │
╰─────────────────────────╯`

func TestCrushProviderDetect(t *testing.T) {
	p := newCrushProvider()
	if p.Name() != "crush" {
		t.Fatalf("Name() = %q, want \"crush\"", p.Name())
	}
	cases := []struct {
		name   string
		screen string
		want   State
	}{
		{"idle", crushIdleReal, StateIdle},
		{"busy", crushBusyReal, StateBusy},
		{"confirm", crushConfirmReal, StateAwaitConfirm},
		{"shell", fixtureShell, StateUnknown},
		{"empty", "", StateUnknown},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := p.Detect(c.screen); got != c.want {
				t.Fatalf("crush.Detect(%s) = %s, want %s", c.name, got, c.want)
			}
		})
	}
}

// ─── qwen-code (TUI Ink, fork gemini-cli) ───
const qwenIdleReal = `╭──────────────────────────────────╮
│ >                                │
╰──────────────────────────────────╯
  ? for shortcuts     42% context used`

const qwenBusyReal = `✦ Thinking (3s · esc to cancel)
╭──────────────────────────────────╮
│ >                                │
╰──────────────────────────────────╯`

const qwenConfirmReal = `Apply this change?
  ● 1. Yes, allow once
    2. Yes, allow always
    3. No, suggest changes (esc)`

func TestQwenCodeProviderDetect(t *testing.T) {
	p := newQwenCodeProvider()
	if p.Name() != "qwen-code" {
		t.Fatalf("Name() = %q, want \"qwen-code\"", p.Name())
	}
	cases := []struct {
		name   string
		screen string
		want   State
	}{
		{"idle", qwenIdleReal, StateIdle},
		{"busy", qwenBusyReal, StateBusy},
		{"confirm", qwenConfirmReal, StateAwaitConfirm},
		{"shell", fixtureShell, StateUnknown},
		{"empty", "", StateUnknown},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := p.Detect(c.screen); got != c.want {
				t.Fatalf("qwen-code.Detect(%s) = %s, want %s", c.name, got, c.want)
			}
		})
	}
}

// TestLot2AbstainOnShellAndClaude : les 5 nouveaux adaptateurs ne doivent JAMAIS
// revendiquer un shell nu (sinon injection dans une surface non-agent) ni l'écran
// idle de Claude (glyphe ❯ + footer Claude). C'est l'invariant de sûreté central.
func TestLot2AbstainOnShellAndClaude(t *testing.T) {
	for _, p := range []patternProvider{
		newAiderProvider(), newGooseProvider(), newOpencodeProvider(),
		newCrushProvider(), newQwenCodeProvider(),
	} {
		if got := p.Detect(fixtureShell); got != StateUnknown {
			t.Errorf("%s a revendiqué un shell nu comme %s, want unknown", p.Name(), got)
		}
		if got := p.Detect(fixtureIdle); got != StateUnknown {
			t.Errorf("%s a revendiqué l'idle de Claude comme %s, want unknown", p.Name(), got)
		}
	}
}

// TestLot2ConfirmBeatsBusy : un dialogue de confirmation qui porte AUSSI un marqueur
// busy doit rester AwaitConfirm — ne jamais auto-soumettre dans une confirmation.
func TestLot2ConfirmBeatsBusy(t *testing.T) {
	cases := []struct {
		name   string
		p      patternProvider
		screen string
	}{
		{"aider", newAiderProvider(), aiderConfirmReal + "\n⠋ Waiting for gpt-4o"},
		{"goose", newGooseProvider(), gooseConfirmReal + "\n(Ctrl+C to interrupt)"},
		{"opencode", newOpencodeProvider(), opencodeConfirmReal + "\n  esc interrupt"},
		{"crush", newCrushProvider(), crushConfirmReal + "\n  Thinking..."},
		{"qwen-code", newQwenCodeProvider(), qwenConfirmReal + "\n(3s · esc to cancel)"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.p.Detect(c.screen); got != StateAwaitConfirm {
				t.Fatalf("%s: got %s, want await-confirm (confirm must beat busy)", c.name, got)
			}
		})
	}
}

// TestLot2DetectAnyStateCrossProvider : le registre attribue les écrans idle SANS
// AMBIGUÏTÉ (goose/opencode/crush ont des footers distinctifs qu'aucun provider
// antérieur ne revendique). qwen-code est traité à part (caveat partagé ci-dessous).
func TestLot2DetectAnyStateCrossProvider(t *testing.T) {
	cases := []struct {
		screen string
		name   string
	}{
		{gooseIdleReal, "goose"},
		{opencodeIdleReal, "opencode"},
		{crushIdleReal, "crush"},
	}
	for _, c := range cases {
		if st, name := DetectAnyState(c.screen); st != StateIdle || name != c.name {
			t.Fatalf("%s idle → (%s, %q), want (idle, %s)", c.name, st, name, c.name)
		}
	}
}

// TestQwenIdleAttributedToCodex : qwen-code partage EXACTEMENT la signature idle de
// Codex (boîte ›/> vide + footer "? for shortcuts"). Codex étant registré avant,
// le registre attribue l'idle de qwen à "codex". L'ÉTAT (idle) reste correct — seul
// le nom diffère. Ce test documente et verrouille ce comportement attendu (analogue
// à TestAntigravityBusyAttributedToClaudeFirst pour le busy partagé).
func TestQwenIdleAttributedToCodex(t *testing.T) {
	// Le détecteur qwen SEUL lit bien un idle.
	if got := newQwenCodeProvider().Detect(qwenIdleReal); got != StateIdle {
		t.Fatalf("qwen-code.Detect(idle) = %s, want idle", got)
	}
	// Dans le registre, Codex (antérieur) revendique le même écran — état correct.
	if st, name := DetectAnyState(qwenIdleReal); st != StateIdle || name != "codex" {
		t.Fatalf("qwen idle via registre → (%s, %q), want (idle, codex) [footer partagé]", st, name)
	}
}
