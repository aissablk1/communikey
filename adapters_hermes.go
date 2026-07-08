package main

import "regexp"

// adapters_hermes.go — couche 2 (providers) : adaptateur d'état pour Hermes Agent
// (Nous Research), le CLI/TUI Python auto-améliorant
// (github.com/NousResearch/hermes-agent, MIT). Même logique safety-first que les
// autres (confirm > busy > idle > unknown).
//
// CALIBRATION (§2/§29) — tokens VÉRIFIÉS SUR SOURCE PRIMAIRE, jamais devinés :
//   - Front-end réel : `hermes` (entry point Python `hermes_cli.main`) rend le REPL
//     `cli.py` (prompt_toolkit). Le TUI node `ui-tui` (ink) ne se lance que sur
//     opt-in (`--tui`, `HERMES_TUI=1`, `display.interface: tui`). Vérifié en ligne
//     (repo `main` = 0.18.2) ET cross-vérifié dans l'install locale 0.14.0 (tous les
//     tokens ci-dessous présents dans les deux → stables). `_resolve_use_tui` absent
//     en 0.14.0 → cette version rend TOUJOURS `cli.py` Python.
//   - confirm : panneau "⚠️  Dangerous Command" + options "1. Allow once" /
//     "2. Allow for this session" / "3. Add to permanent allowlist" / "4. Deny" —
//     validées par Entrée (défaut surligné = Allow once = APPROUVE) OU un chiffre,
//     PAS de binding y/n → la détection est NON négociable. Confirm secondaire
//     (slash destructif) : "Type 1/2/3 or use ↑/↓ then Enter. ESC/Ctrl+C cancels."
//     (cli.py, `_get_approval_display_fragments`).
//   - busy : prompt de travail "⚕ ❯" (style `prompt-working`) ; placeholder inline
//     "msg=interrupt · /queue · /bg · /steer · Ctrl+C cancel" ; "Processing
//     command..." ; "⚡ Interrupting agent...".
//   - idle : NON DÉTECTÉ (sûr, comme aider). Le prompt idle est "❯ " (le MÊME glyph
//     que Claude Code) avec un placeholder VIDE et AUCUN footer persistant distinctif
//     (`cli.py` n'a pas de `bottom_toolbar`/`rprompt`) → pas de double signal fiable.
//     Détecter l'idle reviendrait soit à revendiquer un écran Claude, soit à lire un
//     shell nu comme soumissible. On s'abstient : un Hermes idle → StateUnknown, jamais
//     auto-validé. Une capture d'écran live pourrait un jour révéler un footer sûr.
//
// STATUT : provisoire (comme codex/gemini/antigravity/clawcodex) — calibré sur SOURCE,
// pas sur capture d'écran live. Les fixtures de test sont SYNTHÉTIQUES (dérivées des
// tokens vérifiés), pas des captures réelles `*Real`. Détails et sources :
// docs/strategy/2026-07-08-hermes-agent-clarification.md.
func newHermesProvider() patternProvider {
	return patternProvider{
		name: "hermes",
		confirm: []*regexp.Regexp{
			regexp.MustCompile(`(?i)dangerous command`),
			regexp.MustCompile(`(?i)allow for this session|add to permanent allowlist`),
			regexp.MustCompile(`(?i)type\s*1/2/3\s*or use`),
		},
		busy: []*regexp.Regexp{
			regexp.MustCompile(`⚕`), // prompt de travail "⚕ ❯" — absent à l'idle
			regexp.MustCompile(`msg=interrupt`),
			regexp.MustCompile(`(?i)processing command|interrupting agent`),
		},
		// idle NON détecté (voir en-tête) : idlePrompt/idleFooter laissés nil → Detect
		// ne renvoie jamais StateIdle pour Hermes (abstention sûre).
	}
}
