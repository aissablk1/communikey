package main

import (
	"os"
	"testing"
)

// L'identité du hook se dérive du session_id (zéro-config), mais COMKEY_AGENT_ID
// garde la priorité quand il est posé explicitement.
func TestHookIdentityDerivesFromSessionID(t *testing.T) {
	os.Unsetenv("COMKEY_AGENT_ID")
	if got := hookIdentity(hookPayload{SessionID: "bb751108-3b17-4abc"}); got != "sess-bb751108" {
		t.Fatalf("identité dérivée = %q, attendu sess-bb751108", got)
	}
	if got := hookIdentity(hookPayload{}); got == "" {
		t.Fatalf("sans session_id, fallback non vide attendu")
	}
	os.Setenv("COMKEY_AGENT_ID", "claude-dev")
	defer os.Unsetenv("COMKEY_AGENT_ID")
	if got := hookIdentity(hookPayload{SessionID: "ignore"}); got != "claude-dev" {
		t.Fatalf("COMKEY_AGENT_ID devrait primer, obtenu %q", got)
	}
}
