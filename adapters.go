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
//   • Antigravity : successeur officiel de Gemini CLI pour les comptes individuels
//     (Gemini CLI retiré le 18/06/2026 — developers.googleblog.com, vérifié
//     2026-07-07). Binaire Homebrew cask `antigravity-cli` 1.0.16 (`agy`) installé
//     localement ; tokens extraits via `strings` sur le binaire réel (aucune capture
//     d'écran live disponible — OAuth Google requis) : "Press esc to interrupt
//     generation.", "Generating... (Enter/Esc to cancel)", "Press ? to see keyboard
//     shortcuts.", "Do you want to proceed?".
//   • ClawCodex : dépôt officiel agentforce314/clawcodex (MIT, 686★, actif —
//     dernier push 2026-07-07), vérifié via `gh search code` + raw.githubusercontent.com
//     le 2026-07-07 — "esc to interrupt · ${exitHint}" en busy vs juste `exitHint` en
//     idle (ui-tui/src/app/useInputHandlers.ts), `exitHint` = "/exit to quit clawcodex"
//     (ou "/new to start a fresh chat" en mode dashboard TUI), confirm "Do you want to
//     proceed?"/"Would you like to proceed?" (ui-tui/src/components/prompts.tsx),
//     glyph composer par défaut **"❯"** confirmé par test unitaire
//     (`composerPromptText('❯')` → `'❯'`, ui-tui/src/__tests__/prompt.test.ts) — le
//     MÊME glyph que Claude Code (state.go réutilise `reIdlePrompt`, pas le motif
//     partagé `›`/`>` des autres adaptateurs).
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
//   • Antigravity : "Press ? to see keyboard shortcuts." est extraite ADJACENTE à
//     "Press esc to interrupt generation." dans la table de chaînes du binaire — pas
//     de preuve qu'elle est EXCLUSIVE à l'idle (pourrait aussi s'afficher en busy).
//     Sans conséquence sur la sûreté : Detect() vérifie busy AVANT idle, donc un
//     écran affichant les deux est classé busy, jamais mal lu comme idle. Le caractère
//     de la boîte de saisie n'est pas confirmé → même hypothèse partagée "›"/">" que
//     Codex/Gemini. "esc to interrupt" est aussi partagé avec Claude (premier dans le
//     registre) : un écran Antigravity busy peut être attribué à "claude" — l'ÉTAT
//     reste correct, seul le nom de provider peut différer (même caveat que Codex).
//     "Allow access to this file?" (confirmée dans le binaire) n'est pas encore
//     couverte par le motif partagé reAdptConfirmVerb — écran retombe en unknown
//     (jamais en idle), pas de risque, juste sous-détecté tant que non élargi.
//   • ClawCodex : glyph composer "❯" CONFIRMÉ (pas une hypothèse) — mais c'est le
//     même glyph que Claude Code (state.go), donc un écran idle ClawCodex peut
//     techniquement matcher le reIdlePrompt de Claude ; Claude reste néanmoins premier
//     à abstenir grâce à SON footer disjoint ("shift+tab to cycle"/"? for
//     shortcuts"/… vs "/exit to quit clawcodex") — testé (TestClawCodexAbstainedByClaude
//     dans adapters_test.go). "esc to interrupt" et "do you want to proceed" restent
//     partagés avec Claude comme pour Codex/Antigravity — même caveat d'attribution,
//     état toujours correct. Pas de capture d'écran live (source = code TypeScript
//     réel du dépôt, pas un rendu observé) — un live capture confirmerait la mise en
//     page exacte.

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

// newAntigravityProvider builds the Google Antigravity CLI adapter — successeur
// officiel de Gemini CLI pour les comptes individuels (retiré le 18/06/2026).
//
// Calibré par extraction statique (`strings`) sur le binaire réellement installé
// (Homebrew cask antigravity-cli 1.0.16, `agy`) — pas de capture d'écran live
// (OAuth Google requis, non automatisable ici) : busy "Press esc to interrupt
// generation." / "Generating... (Enter/Esc to cancel)" ; confirm partagé (« Do you
// want to proceed? » confirmée dans le binaire) ; footer idle "Press ? to see
// keyboard shortcuts.". Voir les CAVEATS du header de ce fichier.
func newAntigravityProvider() patternProvider {
	return patternProvider{
		name:    "antigravity",
		confirm: []*regexp.Regexp{reAdptConfirmNum, reAdptConfirmYN, reAdptConfirmVerb},
		busy: []*regexp.Regexp{
			// Antigravity-specific templates (disambiguate from Claude's bare "esc to interrupt").
			regexp.MustCompile(`(?i)esc to interrupt generation`),
			regexp.MustCompile(`(?i)generating\.{3}.{0,20}esc to cancel`),
			// Fallback: shared "esc to interrupt" (state still correct if attributed via order).
			regexp.MustCompile(`(?i)esc to interrupt`),
		},
		idlePrompt: reAdptIdleBox,
		idleFooter: regexp.MustCompile(`(?i)press \? to see keyboard shortcuts`),
	}
}

// newClawCodexProvider builds the ClawCodex CLI adapter (agentforce314/clawcodex,
// MIT, « Python rebuild of Claude Code » with 25 LLM backend integrations).
//
// Calibré sur le dépôt officiel (main, vérifié 2026-07-07) — busy "esc to
// interrupt · <exitHint>" ; idle juste `<exitHint>` = "/exit to quit clawcodex" ou
// "/new to start a fresh chat" (ui-tui/src/app/useInputHandlers.ts) ; confirm "Do
// you want to proceed?"/"Would you like to proceed?" (ui-tui/src/components/
// prompts.tsx) ; glyph composer par défaut CONFIRMÉ "❯" (ui-tui/src/__tests__/
// prompt.test.ts) — réutilise reIdlePrompt de state.go (même glyph que Claude,
// PAS le reAdptIdleBox partagé ›/>). Voir les CAVEATS du header de ce fichier.
func newClawCodexProvider() patternProvider {
	return patternProvider{
		name:    "clawcodex",
		confirm: []*regexp.Regexp{reAdptConfirmNum, reAdptConfirmYN, reAdptConfirmVerb},
		busy: []*regexp.Regexp{
			// Shared with Claude/Codex/Antigravity by design (real token, confirmed in
			// useInputHandlers.ts) — state stays correct even if attribution goes to
			// "claude" first (same documented caveat as Codex/Antigravity).
			regexp.MustCompile(`(?i)esc to interrupt`),
		},
		idlePrompt: reIdlePrompt, // state.go — same "❯" glyph, CONFIRMED via prompt.test.ts
		idleFooter: regexp.MustCompile(`(?i)to quit clawcodex|to start a fresh chat`),
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// LOT 2 (2026-07-08) — adaptateurs pour d'autres CLI d'agents de code.
// Tokens VÉRIFIÉS SUR SOURCE PRIMAIRE (dépôts officiels, raw GitHub, 2026-07-08).
// Aucune capture d'écran live (chaque agent exige un compte/exécution) — les
// tokens viennent du CODE réel, pas d'un rendu observé ; une capture live figerait
// la mise en page exacte. Même logique safety-first que ci-dessus : confirm > busy
// > idle (double signal) > unknown ; un shell nu ou un écran Claude ne sont JAMAIS
// lus comme idle.
// ─────────────────────────────────────────────────────────────────────────────

// newAiderProvider — Aider (Aider-AI/aider, REPL prompt_toolkit ligne-à-ligne, PAS
// un TUI plein écran). Calibré (§2/§29) sur main : confirm suffixe "(Y)es/(N)o"
// (aider/io.py confirm_ask) ; busy spinner "Waiting for <modèle>"/"Waiting for LLM"
// (aider/waiting.py, base_coder.py). IDLE VOLONTAIREMENT NON DÉTECTÉ : le prompt
// idle est "> "/"architect> " (aider/io.py) — trop générique — et aider n'a AUCUN
// footer persistant (0 bottom_toolbar dans io.py, vérifié). Sans second signal, un
// shell nu serait indistinguable → on renvoie UNKNOWN (jamais d'auto-submit), le
// côté sûr. On détecte donc seulement confirm et busy (déjà utile : ne jamais
// soumettre pendant une confirmation ou une génération).
func newAiderProvider() patternProvider {
	return patternProvider{
		name: "aider",
		confirm: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\(y\)es/\(n\)o`), // suffixe distinctif d'aider
			reAdptConfirmYN, reAdptConfirmVerb,
		},
		busy: []*regexp.Regexp{
			// "Waiting for LLM" / "Waiting for <modèle>". RE2 n'a pas de look-ahead :
			// on ancre sur des tokens de modèle pour NE PAS matcher l'idle
			// "Aider is waiting for your input" (aider/io.py) — "your input" ne matche pas.
			regexp.MustCompile(`(?i)waiting for (llm\b|[\w.:/-]*(gpt|claude|sonnet|opus|haiku|gemini|llama|deepseek|mistral|qwen|kimi|glm|grok|command|o[1-9])[\w.:/-]*)`),
		},
		idlePrompt: nil, // pas de double signal fiable → idle non détecté (sûr)
		idleFooter: nil,
	}
}

// newGooseProvider — goose (block/goose, REPL rustyline, Rust). Calibré (§2/§29)
// sur main : confirm "…do you allow?" + menu Allow/Deny (crates/goose-cli/src/
// session/mod.rs) ; busy spinner se terminant par "(Ctrl+C to interrupt)" (le
// préfixe est un message aléatoire — on matche le hint d'interruption, pas le
// verbe : output.rs + thinking.rs) ; idle prompt rustyline "> " + hint distinctif
// "Enter to send · Ctrl+J newline" ou bannière "goose is ready" (completion.rs,
// output.rs). NB : l'art ASCII "( O)>" de la bannière n'est PAS utilisé comme
// détecteur d'idle (il apparaît aussi au démarrage).
func newGooseProvider() patternProvider {
	return patternProvider{
		name: "goose",
		confirm: []*regexp.Regexp{
			regexp.MustCompile(`(?i)do you allow`),
			reAdptConfirmNum, reAdptConfirmYN, reAdptConfirmVerb,
		},
		busy: []*regexp.Regexp{
			regexp.MustCompile(`(?i)ctrl\+c to interrupt`),
		},
		// Prompt rustyline nu "> " : générique SEUL, mais verrouillé par le footer
		// distinctif ci-dessous (un shell — même un PS2 "> " — n'a jamais "Enter to
		// send"/"goose is ready"). Double signal obligatoire.
		idlePrompt: regexp.MustCompile(`(?m)^\s*>\s*$`),
		idleFooter: regexp.MustCompile(`(?i)enter to send|goose is ready`),
	}
}

// newOpencodeProvider — opencode (sst/opencode, TUI SolidJS @opentui, branche dev).
// Calibré (§2/§29) : confirm dialogue "Permission required" + boutons "Allow once"/
// "Allow always"/"Reject" (packages/tui/src/routes/session/permission.tsx) ; busy
// "esc … interrupt" dans la ligne de statut du prompt (component/prompt/index.tsx,
// rendu seulement si status ≠ idle) ; idle placeholder "Ask anything…" (ou "Run a
// command…" en mode shell) + rangée de hints "agents"/"commands" (prompt/index.tsx,
// home.tsx). CAVEAT : "esc … interrupt" est partagé avec Claude/Codex (registrés
// avant) — l'ÉTAT reste correct, seul le nom d'attribution peut différer.
func newOpencodeProvider() patternProvider {
	return patternProvider{
		name: "opencode",
		confirm: []*regexp.Regexp{
			regexp.MustCompile(`(?i)permission required`),
			regexp.MustCompile(`(?i)allow once|allow always`),
			reAdptConfirmVerb,
		},
		busy: []*regexp.Regexp{
			regexp.MustCompile(`(?i)esc\b.{0,16}interrupt`),
		},
		// Le placeholder "Ask anything…" n'apparaît QU'À l'idle (saisie vide) — signal
		// distinctif ; verrouillé par la rangée de hints agents/commands / footer.
		idlePrompt: regexp.MustCompile(`(?i)ask anything\.\.\.|run a command\.\.\.`),
		idleFooter: regexp.MustCompile(`(?i)\bagents\b|\bcommands\b|/status`),
	}
}

// newCrushProvider — crush (charmbracelet/crush, TUI Bubbletea v2, branche main ;
// le TUI vit dans internal/ui/). Calibré (§2/§29) : confirm dialogue "Permission
// Required" + boutons "Allow"/"Allow for Session"/"Deny" (internal/ui/dialog/
// permissions.go) ; busy label spinner "Thinking"/"Summarizing" ou placeholders de
// travail ("Working!", "Processing…", "Brrrrr…") ou binding "esc … cancel"
// (internal/ui/model/ui.go, chat/assistant.go, keys.go) ; idle placeholders "Ready!"/
// "Ready for instructions" (readyPlaceholders) + marque persistante "CRUSH"/"Charm™"
// (header.go). Les placeholders sont aléatoires → on matche l'UNION, pas un mot.
// CAVEAT : "Permission Required" est partagé avec opencode (registré avant) —
// l'ÉTAT (confirm) reste correct, seul le nom d'attribution peut différer.
func newCrushProvider() patternProvider {
	return patternProvider{
		name: "crush",
		confirm: []*regexp.Regexp{
			regexp.MustCompile(`(?i)permission required`),
			regexp.MustCompile(`(?i)allow for session`),
			reAdptConfirmVerb,
		},
		busy: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\bthinking\b|\bsummarizing\b`),
			regexp.MustCompile(`(?i)working!|working\.\.\.|processing\.\.\.|brrrr|prrrr`),
		},
		idlePrompt: regexp.MustCompile(`(?i)ready!|ready\.\.\.|ready\?|ready for instructions`),
		idleFooter: regexp.MustCompile(`CRUSH|Charm`),
	}
}

// newQwenCodeProvider — Qwen Code (QwenLM/qwen-code, TUI Ink/React, fork de
// gemini-cli, branche main). Calibré (§2/§29) : confirm "Apply this change?" /
// "Allow execution of:" / "Do you want to proceed?" / "Waiting for user
// confirmation…" + "Yes, allow once" (components/messages/ToolConfirmationMessage.
// tsx) ; busy suffixe "· esc to cancel)" (components/LoadingIndicator.tsx) ; idle
// prompt "> " (InputPrompt.tsx) + footer "? for shortcuts" (Footer.tsx). CAVEATS :
// (1) "esc to cancel" est PARTAGÉ avec Gemini (registré avant) → un écran qwen busy
// peut être attribué à "gemini" ; état correct. (2) idle "> " + "? for shortcuts"
// est la MÊME signature que Codex (registré avant) → un écran qwen idle peut être
// attribué à "codex" ; l'ÉTAT (idle) reste correct, seul le nom diffère. Les
// chaînes UI passent par i18n t() → matches possibles seulement en locale `en`.
func newQwenCodeProvider() patternProvider {
	return patternProvider{
		name: "qwen-code",
		confirm: []*regexp.Regexp{
			reAdptConfirmNum, reAdptConfirmYN, reAdptConfirmVerb,
			regexp.MustCompile(`(?i)apply this change\?|allow execution of|waiting for user confirmation`),
		},
		busy: []*regexp.Regexp{
			regexp.MustCompile(`(?i)esc to cancel`),
		},
		idlePrompt: reAdptIdleBox,
		idleFooter: regexp.MustCompile(`(?i)\?\s*for shortcuts`),
	}
}
