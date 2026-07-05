package main

import (
	"os"
	"strings"
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

// hookInstallFor ne doit JAMAIS faire passer silencieusement un provider inconnu pour
// Claude — trouvé en lisant le code le 2026-07-05 (docs/strategy/communikey-vision-
// 2026-07-05.md) : le switch ne connaissait que "codex"/"gemini", tout le reste
// retombait sur le snippet Claude sans le dire.
func TestHookInstallForUnknownProviderWarnsExplicitly(t *testing.T) {
	out := hookInstallFor("opencode")
	if out == hookInstall {
		t.Fatal("un provider inconnu ne doit plus renvoyer le snippet Claude tel quel, sans avertissement")
	}
	if !strings.Contains(out, "opencode") || !strings.Contains(out, "communikey provider list") {
		t.Fatalf("l'avertissement doit citer le provider et pointer vers `provider list`, got: %s", out)
	}
}

func TestHookInstallForKnownProvidersUnaffected(t *testing.T) {
	if hookInstallFor("claude") != hookInstall {
		t.Fatal("claude doit toujours renvoyer le snippet Claude")
	}
	if hookInstallFor("codex") == hookInstall {
		t.Fatal("codex doit avoir son propre snippet, pas celui de Claude")
	}
	if hookInstallFor("gemini") == hookInstall {
		t.Fatal("gemini doit avoir son propre snippet, pas celui de Claude")
	}
}
