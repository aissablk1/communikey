package main

import (
	"fmt"
	"regexp"
	"strings"
)

// State is the runtime state of an agent CLI session, read from its on-screen
// content. It is the heart of communikey's "intelligent" delivery: we never submit a
// message into a session that is not safely Idle.
type State int

const (
	// StateUnknown: not a recognizable agent prompt (a shell, a browser, a blank
	// or not-yet-materialized surface). Treated as NOT submittable.
	StateUnknown State = iota
	// StateIdle: the agent is waiting for input — safe to deliver AND submit.
	StateIdle
	// StateBusy: the agent is working/streaming a turn — deliver but never submit.
	StateBusy
	// StateAwaitConfirm: a y/N or numbered confirmation dialog is showing —
	// never press Enter (would blindly answer a prompt). Deliver-and-alert only.
	StateAwaitConfirm
	// StateUnreachable: the surface exists but its screen can't be read (a
	// background workspace whose PTY isn't materialized — cmux #1472). Must be
	// focused/materialized before delivery; distinct from "not an agent".
	StateUnreachable
)

func (s State) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateBusy:
		return "busy"
	case StateAwaitConfirm:
		return "await-confirm"
	case StateUnreachable:
		return "unreachable"
	default:
		return "unknown"
	}
}

// Signals are matched only against the live region (screen tail), never the
// scrollback, so old transcript text can't be mistaken for current state.
const stateTailLines = 24

var (
	// Busy: the live footer Claude renders while a turn is running.
	reBusy = regexp.MustCompile(`(?i)esc to interrupt`)

	// Confirmation dialogs — several INDEPENDENT signals so a wording change in
	// any single one doesn't blind us (the brittleness that sinks claude-squad
	// & atmux, which match one hardcoded English string).
	reConfirmMenu = regexp.MustCompile(`(?im)^\s*[│|]?\s*(?:❯|>)?\s*1\.\s*yes\b`) // numbered "1. Yes"
	reConfirmYN   = regexp.MustCompile(`(?i)\(\s*y\s*/\s*n\s*\)|\[\s*y\s*/\s*n\s*\]`)
	reConfirmWord = regexp.MustCompile(`(?i)do you want to (proceed|continue|create|make|allow|trust|run)`)
	reConfirmTell = regexp.MustCompile(`(?i)no,?\s+and tell\b`)

	// Idle: an EMPTY input box (`❯` with nothing typed) plus a Claude status
	// footer. Both required: the box alone could be a draft; the footer alone
	// could be mid-stream. After the `❯` we allow only NON-alphanumeric runes
	// (trailing spaces, box chars, and the terminal cursor — which renders as a
	// stray non-UTF-8 byte we observed on real screens). A `❯ hello` draft has a
	// letter so it won't match → we won't clobber a half-typed prompt.
	reIdlePrompt = regexp.MustCompile(`(?m)^\s*[│|]?\s*❯[^\p{L}\p{N}]*$`)
	reIdleFooter = regexp.MustCompile(`(?i)shift\+tab to cycle|\?\s*for shortcuts|for agents|bypass permissions|ctx:`)
)

// DetectClaudeState classifies a Claude Code TUI screen.
//
// Safety-first ordering is deliberate: a confirmation prompt must NEVER be read
// as idle (or we'd auto-answer a y/N), and anything ambiguous falls through to
// Busy/Unknown rather than Idle — communikey only auto-submits on a confident Idle.
func DetectClaudeState(screen string) State {
	tail := tailLines(screen, stateTailLines)
	switch {
	case reConfirmMenu.MatchString(tail) || reConfirmYN.MatchString(tail) ||
		reConfirmWord.MatchString(tail) || reConfirmTell.MatchString(tail):
		return StateAwaitConfirm
	case reBusy.MatchString(tail):
		return StateBusy
	case reIdlePrompt.MatchString(tail) && reIdleFooter.MatchString(tail):
		return StateIdle
	default:
		return StateUnknown
	}
}

// StateDebug reports which individual signals fired, to diagnose detection on
// real screens (hidden `communikey _why` command).
func StateDebug(screen string) string {
	tail := tailLines(screen, stateTailLines)
	return fmt.Sprintf("confirmMenu=%v confirmYN=%v confirmWord=%v confirmTell=%v | busy=%v | idlePrompt=%v idleFooter=%v\n=> %s",
		reConfirmMenu.MatchString(tail), reConfirmYN.MatchString(tail),
		reConfirmWord.MatchString(tail), reConfirmTell.MatchString(tail),
		reBusy.MatchString(tail), reIdlePrompt.MatchString(tail),
		reIdleFooter.MatchString(tail), DetectClaudeState(screen))
}

// tailLines returns the last n lines of s with trailing blank lines trimmed.
func tailLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
