package main

import "testing"

func TestRegisterSelfAppearsInFleet(t *testing.T) {
	s, _ := OpenStore(t.TempDir())
	if err := registerSelf(s, "A-bash", "bash", "myproj"); err != nil {
		t.Fatal(err)
	}
	if err := registerSelf(s, "B-codex", "codex", "other"); err != nil {
		t.Fatal(err)
	}
	recs, err := s.ListSessions()
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d agents, want 2", len(recs))
	}
	// register created the inbox → a message delivers and counts as pending.
	if err := s.Inbox().Deliver(InboxMessage{ID: "m", TS: "t", To: "A-bash", Body: "hi"}); err != nil {
		t.Fatal(err)
	}
	if n, _ := s.Inbox().Pending("A-bash"); n != 1 {
		t.Fatalf("pending=%d want 1", n)
	}
	r, ok, _ := s.GetSession("B-codex")
	if !ok || r.Provider != "codex" {
		t.Fatalf("provider non conservé: %+v", r)
	}
}

func TestProviderSuffix(t *testing.T) {
	if providerSuffix("") != "" {
		t.Fatal("provider vide → pas de suffixe")
	}
	if providerSuffix("gemini") != " (gemini)" {
		t.Fatalf("suffixe = %q", providerSuffix("gemini"))
	}
}
