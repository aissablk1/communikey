package main

import "regexp"

// adapters.go — couche 2 (providers) : adaptateurs d'état pour les CLI d'agents
// autres que Claude (Codex, Gemini, …).
//
// Le détecteur Claude (state.go) est réglé à la main sur de VRAIS écrans capturés.
// Ces adaptateurs appliquent la MÊME logique « safety-first » que Claude
// (confirm > busy > idle > unknown) : un dialogue de confirmation n'est JAMAIS lu
// comme idle (donc jamais auto-validé), et l'idle exige une DOUBLE preuve (boîte de
// saisie vide + footer distinctif) pour qu'un shell nu ne soit jamais pris pour une
// invite soumissible (le seul faux positif dangereux).
//
// CALIBRATION (§2/§29) — tokens de détection VÉRIFIÉS SUR SOURCE PRIMAIRE le
// 2026-06-28, pas devinés :
//   • Gemini : bundle réellement installé @google/gemini-cli 0.40.1
//     (…/bundle/chunk-*.js) — cancelAction "Esc to cancel" / "(esc to cancel, <t>)",
//     "Allow execution of", "Apply this change?", "Do you want to proceed?",
//     "Waiting for user confirmation…", modèle "gemini-2.5-pro|flash|flash-lite".
//   • Codex : dépôt officiel openai/codex, tag rust-v0.142.3 —
//     "Working (Xs • esc to interrupt)" (tui/src/status_indicator_widget.rs),
//     footer "? for shortcuts" / "to exit" / "to view transcript" …
//     (tui/src/bottom_pane/footer.rs), "Would you like to run/make/grant …" et
//     "{server} needs your approval." (tui/src/bottom_pane/approval_overlay.rs),
//     composer caret "›" U+203A (tui/src/bottom_pane/chat_composer.rs).
//
// CAVEATS honnêtes encore ouverts (une capture d'écran live les lèverait) :
//   • "esc to interrupt" est PARTAGÉ entre Codex et Claude : un écran Codex *busy*
//     peut être attribué à "claude" dans le registre (Claude est premier). L'ÉTAT
//     reste correct ; seul le nom de provider peut différer. L'attribution propre
//     est garantie à l'IDLE (footers disjoints). On atténue avec le gabarit
//     Codex-spécifique "(working|thinking) … to interrupt".
//   • Codex n'expose PAS le nom de modèle dans le footer persistant (UNKNOWN à la
//     source) → l'idle Codex s'appuie sur les hints de footer ("? for shortcuts"…).
//   • Le caractère exact de la boîte de saisie Gemini n'a pas été capturé en live ;
//     on accepte "›"/">" et on verrouille l'idle par le footer "gemini-\d". Si le
//     caret réel diffère, l'idle échoue du côté SÛR (→ unknown, jamais d'auto-submit).

// patternProvider is a table-driven Provider: it recognizes a CLI's on-screen
// state from supplied regex signals, reusing the same ordered, safety-first
// classification as the hand-tuned Claude detector (state.go).
type patternProvider struct {
	name    string
	confirm []*regexp.Regexp // a y/N or numbered approval dialog — never auto-answer
	busy    []*regexp.Regexp // a turn is streaming — deliver but never submit
	// idle requires BOTH: an EMPTY input box AND a provider-distinctive footer.
	// The dual signal is what keeps a plain shell (no agent footer) from ever being
	// misread as a submittable idle prompt — the only dangerous false positive.
	idlePrompt *regexp.Regexp
	idleFooter *regexp.Regexp
}

func (p patternProvider) Name() string { return p.name }

// Detect mirrors DetectClaudeState's deliberate ordering: confirm first (so a
// dialog is never auto-submitted), then busy, then a confident dual-signal idle,
// else Unknown (so the registry can fall through to another provider).
func (p patternProvider) Detect(screen string) State {
	tail := tailLines(screen, stateTailLines)
	switch {
	case anyMatch(p.confirm, tail):
		return StateAwaitConfirm
	case anyMatch(p.busy, tail):
		return StateBusy
	case p.idlePrompt != nil && p.idleFooter != nil &&
		p.idlePrompt.MatchString(tail) && p.idleFooter.MatchString(tail):
		return StateIdle
	default:
		return StateUnknown
	}
}

// anyMatch reports whether any of the regexes matches s (nil entries ignored).
func anyMatch(res []*regexp.Regexp, s string) bool {
	for _, re := range res {
		if re != nil && re.MatchString(s) {
			return true
		}
	}
	return false
}

// Shared, provider-agnostic confirmation signals — several INDEPENDENT cues so a
// single wording change can't blind us (same defense-in-depth as state.go's
// reConfirm* set). Verbatim phrasings verified in Gemini 0.40.1 and Codex
// rust-v0.142.3 (see file header).
var (
	// Numbered "N. Yes…" menu, optionally prefixed by a box rule, bullet or caret.
	// Codex: "› 1. Yes, proceed (y)" ; Gemini: "● 1. Yes, allow once".
	reAdptConfirmNum = regexp.MustCompile(`(?im)^\s*[│|]?\s*(?:[❯›>●○*•]\s*)?\d+\.\s*yes\b`)
	// Inline (y/n) / [y/n] prompt.
	reAdptConfirmYN = regexp.MustCompile(`(?i)\(\s*y\s*/\s*n\s*\)|\[\s*y\s*/\s*n\s*\]`)
	// Verbal approval prompts. Gemini: "Allow execution of", "Apply this change?",
	// "Do you want to proceed?", "Waiting for user confirmation". Codex: "Would you
	// like to run/make/grant …", "Do you want to approve network…", "needs your
	// approval".
	reAdptConfirmVerb = regexp.MustCompile(`(?i)do you want to (proceed|continue|run|allow|apply|execute)|allow execution of|apply this change|would you like to (run|make|grant|proceed|approve)|do you want to approve network|needs your approval|waiting for user confirmation`)

	// An EMPTY boxed prompt using `›`/`>` (deliberately NOT Claude's `❯`, so these
	// adapters never claim a Claude screen). Any letter/digit after the caret means
	// it's a half-typed draft, not idle — so we won't clobber it.
	reAdptIdleBox = regexp.MustCompile(`(?m)^\s*[│|]?\s*[›>][^\p{L}\p{N}]*$`)
)

// newCodexProvider builds the OpenAI Codex CLI adapter.
//
// Calibré sur openai/codex tag rust-v0.142.3 (§2/§29) : busy "Working|Thinking
// (Xs • esc to interrupt)" (status_indicator_widget.rs) ; footer "? for shortcuts"
// / "to exit" / "to view transcript" / … (bottom_pane/footer.rs) ; confirm "Would
// you like to …" + "needs your approval" (bottom_pane/approval_overlay.rs) ;
// composer caret "›" (bottom_pane/chat_composer.rs). Le nom de modèle N'EST PAS
// dans le footer persistant (UNKNOWN à la source) → on n'y compte pas.
func newCodexProvider() patternProvider {
	return patternProvider{
		name:    "codex",
		confirm: []*regexp.Regexp{reAdptConfirmNum, reAdptConfirmYN, reAdptConfirmVerb},
		busy: []*regexp.Regexp{
			// Codex-specific template (disambiguates from Claude's bare "esc to interrupt").
			regexp.MustCompile(`(?i)\b(working|thinking)\b.{0,40}to interrupt`),
			// Fallback: shared "esc to interrupt" (state still correct if attributed via order).
			regexp.MustCompile(`(?i)esc to interrupt`),
		},
		idlePrompt: reAdptIdleBox,
		idleFooter: regexp.MustCompile(`(?i)\?\s*for shortcuts|\bfor commands\b|to view transcript|to queue message|for file paths|to paste images|\bto exit\b`),
	}
}

// newGeminiProvider builds the Google Gemini CLI adapter.
//
// Calibré sur le bundle installé @google/gemini-cli 0.40.1 (§2/§29) : busy
// "(esc to cancel, <t>)" / cancelAction "Esc to cancel" ; confirm "Allow execution
// of", "Apply this change?", "Do you want to proceed?", "Waiting for user
// confirmation" ; footer modèle "gemini-2.5-pro|flash|flash-lite". ("context left"
// et "GEMINI.md" — suppositions initiales — ne figurent PAS dans le footer réel et
// ont été retirés.)
func newGeminiProvider() patternProvider {
	return patternProvider{
		name:    "gemini",
		confirm: []*regexp.Regexp{reAdptConfirmNum, reAdptConfirmYN, reAdptConfirmVerb},
		busy: []*regexp.Regexp{
			regexp.MustCompile(`(?i)esc to cancel`),
		},
		idlePrompt: reAdptIdleBox,
		idleFooter: regexp.MustCompile(`(?i)gemini-\d`),
	}
}
