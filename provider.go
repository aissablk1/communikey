package main

// provider.go — couche 2 (providers) : détection d'état PLUGGABLE par provider.
//
// Aujourd'hui Claude est implémenté (détection éprouvée sur de vrais écrans).
// Codex / Gemini / etc. sont des points d'extension : on AJOUTE un Provider quand
// on dispose de VRAIS écrans capturés à calibrer (§2 — jamais de fixture inventée).
// Le contrat reste « adressé par écran » : un détecteur ne voit que du texte.

// Provider classifies one agent CLI's on-screen state.
type Provider interface {
	Name() string
	// Detect returns the recognized state, or StateUnknown if this provider does
	// not recognize the screen (so the registry can try the next one).
	Detect(screen string) State
}

// claudeProvider wraps the battle-tested Claude Code detector (state.go).
type claudeProvider struct{}

func (claudeProvider) Name() string          { return "claude" }
func (claudeProvider) Detect(s string) State { return DetectClaudeState(s) }

// providers is the ordered detection registry. Claude stays FIRST so its
// battle-tested detector wins on real Claude screens; the table-driven Codex and
// Gemini adapters (adapters.go) fall through only when Claude abstains. Their
// regexes are still provisional — calibrate on real screens (§2/§29) — but they're
// safe to register: idle requires a dual distinctive signal, and busy/confirm only
// widen the "do not submit" net (never a wrongful submit).
var providers = []Provider{
	claudeProvider{},
	newCodexProvider(),  // provisoire — patterns à calibrer sur de vrais écrans (§2/§29)
	newGeminiProvider(), // provisoire — patterns à calibrer sur de vrais écrans (§2/§29)
}

// DetectAnyState tries each registered provider; the first non-Unknown match wins.
// Returns the detected state and the provider name ("" when unrecognized).
func DetectAnyState(screen string) (State, string) {
	for _, p := range providers {
		if st := p.Detect(screen); st != StateUnknown {
			return st, p.Name()
		}
	}
	return StateUnknown, ""
}
