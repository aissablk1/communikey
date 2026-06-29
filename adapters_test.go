package main

import "testing"

// adapters_test.go — tests des adaptateurs Codex / Gemini (couche 2 providers).
//
// CALIBRATION (§2/§29) — les fixtures ci-dessous sont CONSTRUITES À PARTIR DE TOKENS
// DE DÉTECTION RÉELS, vérifiés sur source primaire le 2026-06-28 :
//   • Gemini : bundle installé @google/gemini-cli 0.40.1 (chunk-*.js) — "Esc to
//     cancel" / "(esc to cancel, <t>)", "Allow execution of this command?",
//     "1. Yes, allow once / allow always", modèle "gemini-2.5-pro".
//   • Codex : openai/codex tag rust-v0.142.3 — "Working (Xs • esc to interrupt)",
//     "? for shortcuts", "Would you like to run the following command?",
//     "› 1. Yes, proceed (y)", composer caret "›".
// La DISPOSITION (cadres ╭╰│) est représentative d'un TUI ; ce sont les TOKENS qui
// sont vérifiés. Une capture d'écran live reste recommandée pour figer la mise en
// page exacte (cf. caveats dans adapters.go). Ces tests garantissent : (1) la logique
// safety-first (confirm > busy > idle > unknown), (2) la double-condition idle
// (boîte vide + footer distinctif), (3) l'abstention sur un shell nu et sur un écran
// Claude, (4) l'attribution cross-provider via DetectAnyState.

// --- Fixtures Gemini CLI 0.40.1 (tokens réels) ---

const geminiIdleReal = `~/projet (main*)

╭────────────────────────────────────────────────╮
│ >                                              │
╰────────────────────────────────────────────────╯

  gemini-2.5-pro   no sandbox`

const geminiBusyReal = `⠹ (esc to cancel, 8s)

╭────────────────────────────────────────────────╮
│ >                                              │
╰────────────────────────────────────────────────╯

  gemini-2.5-pro`

const geminiConfirmReal = `╭─ run_shell_command ──────────────────────────────╮
│ rm -rf ./build                                   │
│                                                  │
│ Allow execution of this command?                 │
│  ● 1. Yes, allow once                            │
│    2. Yes, allow always                          │
│    3. No, suggest changes (esc)                  │
╰──────────────────────────────────────────────────╯`

// --- Fixtures Codex CLI rust-v0.142.3 (tokens réels) ---

const codexIdleReal = `╭────────────────────────────────────────────────╮
│ ›                                              │
╰────────────────────────────────────────────────╯

  ? for shortcuts`

const codexBusyReal = `• Working (3s • esc to interrupt)

╭────────────────────────────────────────────────╮
│ ›                                              │
╰────────────────────────────────────────────────╯

  ? for shortcuts`

const codexConfirmReal = `╭─ Codex ──────────────────────────────────────────╮
│ $ rm -rf ./build                                 │
│                                                  │
│ Would you like to run the following command?     │
│ › 1. Yes, proceed (y)                            │
│   2. No, and tell Codex what to do differently   │
╰──────────────────────────────────────────────────╯`

func TestCodexProviderDetect(t *testing.T) {
	p := newCodexProvider()
	if p.Name() != "codex" {
		t.Fatalf("Name() = %q, want \"codex\"", p.Name())
	}
	cases := []struct {
		name   string
		screen string
		want   State
	}{
		{"idle", codexIdleReal, StateIdle},
		{"busy", codexBusyReal, StateBusy},
		{"confirm", codexConfirmReal, StateAwaitConfirm},
		{"shell", fixtureShell, StateUnknown},
		{"empty", "", StateUnknown},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := p.Detect(c.screen); got != c.want {
				t.Fatalf("codex.Detect(%s) = %s, want %s", c.name, got, c.want)
			}
		})
	}
}

func TestGeminiProviderDetect(t *testing.T) {
	p := newGeminiProvider()
	if p.Name() != "gemini" {
		t.Fatalf("Name() = %q, want \"gemini\"", p.Name())
	}
	cases := []struct {
		name   string
		screen string
		want   State
	}{
		{"idle", geminiIdleReal, StateIdle},
		{"busy", geminiBusyReal, StateBusy},
		{"confirm", geminiConfirmReal, StateAwaitConfirm},
		{"shell", fixtureShell, StateUnknown},
		{"empty", "", StateUnknown},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := p.Detect(c.screen); got != c.want {
				t.Fatalf("gemini.Detect(%s) = %s, want %s", c.name, got, c.want)
			}
		})
	}
}

// TestAdapterConfirmBeatsBusy: a confirmation dialog that ALSO carries a busy
// marker must still classify as AwaitConfirm — we must never auto-submit into a
// confirmation (same safety-first guarantee as Claude's TestConfirmBeatsBusy).
func TestAdapterConfirmBeatsBusy(t *testing.T) {
	if got := newCodexProvider().Detect(codexConfirmReal + "\n  Working (12s • esc to interrupt)"); got != StateAwaitConfirm {
		t.Fatalf("codex: got %s, want await-confirm (confirm must beat busy)", got)
	}
	if got := newGeminiProvider().Detect(geminiConfirmReal + "\n  (esc to cancel, 1s)"); got != StateAwaitConfirm {
		t.Fatalf("gemini: got %s, want await-confirm (confirm must beat busy)", got)
	}
}

// TestAdaptersAbstainOnShellAndClaude: the adapters must NOT claim a plain shell
// (so csend never injects into a non-agent surface) nor a Claude idle screen
// (Claude uses `❯`, the adapters key on `›`/`>`), so adding them to the registry
// can't pollute Claude detection or shell safety.
func TestAdaptersAbstainOnShellAndClaude(t *testing.T) {
	for _, p := range []patternProvider{newCodexProvider(), newGeminiProvider()} {
		if got := p.Detect(fixtureShell); got != StateUnknown {
			t.Errorf("%s claimed a plain shell as %s, want unknown", p.Name(), got)
		}
		if got := p.Detect(fixtureIdle); got != StateUnknown {
			t.Errorf("%s claimed Claude's idle screen as %s, want unknown", p.Name(), got)
		}
	}
}

// TestDetectAnyStateCrossProvider: the registry attributes a provider-specific
// idle screen to the right provider (Claude abstains because the prompt caret and
// footer differ), proving the cross-provider registry works end-to-end.
func TestDetectAnyStateCrossProvider(t *testing.T) {
	if st, name := DetectAnyState(codexIdleReal); st != StateIdle || name != "codex" {
		t.Fatalf("codex idle → (%s, %q), want (idle, codex)", st, name)
	}
	if st, name := DetectAnyState(geminiIdleReal); st != StateIdle || name != "gemini" {
		t.Fatalf("gemini idle → (%s, %q), want (idle, gemini)", st, name)
	}
}
