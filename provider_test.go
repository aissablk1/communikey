package main

import "testing"

func TestDetectAnyStateClaude(t *testing.T) {
	// Real captured idle screen → recognized as Claude, idle.
	st, name := DetectAnyState(fixtureIdle)
	if st != StateIdle || name != "claude" {
		t.Fatalf("idle fixture → (%s, %q), want (idle, claude)", st, name)
	}
	if st, name := DetectAnyState(fixtureBusy); st != StateBusy || name != "claude" {
		t.Fatalf("busy fixture → (%s, %q), want (busy, claude)", st, name)
	}
}

func TestDetectAnyStateUnrecognized(t *testing.T) {
	// A plain shell is not an agent prompt for any registered provider.
	st, name := DetectAnyState(fixtureShell)
	if st != StateUnknown || name != "" {
		t.Fatalf("shell → (%s, %q), want (unknown, \"\")", st, name)
	}
}
