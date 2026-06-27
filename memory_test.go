package main

import (
	"testing"
	"time"
)

func TestJournalAppendAndQuery(t *testing.T) {
	s, err := OpenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	entries := []JournalEntry{
		{ID: "1", TS: "2026-06-27T10:00:00Z", From: "A", To: "SACEM", Action: "submitted", TextLen: 5},
		{ID: "2", TS: "2026-06-27T11:00:00Z", From: "A", To: "ROYALEAI", Action: "staged", TextLen: 9},
		{ID: "3", TS: "2026-06-27T12:00:00Z", From: "B", To: "SACEM", Action: "refused", TextLen: 3},
	}
	for _, e := range entries {
		if err := s.AppendJournal(e); err != nil {
			t.Fatal(err)
		}
	}

	// Filter by recipient.
	got, err := s.QueryJournal(JournalFilter{To: "SACEM"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("to=SACEM: got %d, want 2", len(got))
	}

	// Filter by sender + since.
	since, _ := time.Parse(time.RFC3339, "2026-06-27T10:30:00Z")
	got, err = s.QueryJournal(JournalFilter{From: "A", Since: since})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "2" {
		t.Fatalf("from=A since=10:30: got %+v, want only id=2", got)
	}

	// Limit returns the most recent N.
	got, err = s.QueryJournal(JournalFilter{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "3" {
		t.Fatalf("limit=1: got %+v, want most-recent id=3", got)
	}
}

func TestJournalQueryEmpty(t *testing.T) {
	s, _ := OpenStore(t.TempDir())
	got, err := s.QueryJournal(JournalFilter{})
	if err != nil {
		t.Fatalf("query on empty store should not error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d, want 0", len(got))
	}
}

func TestRegistryUpsertPreservesFirstSeen(t *testing.T) {
	s, _ := OpenStore(t.TempDir())
	if err := s.UpsertSession(SessionRecord{
		SessionID: "7f384610", Provider: "claude", Workspace: "Claude Code",
		State: "idle", FirstSeen: "2026-06-27T09:00:00Z", LastSeen: "2026-06-27T09:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}
	// Update: state changes, last_seen advances, first_seen must be preserved.
	if err := s.UpsertSession(SessionRecord{
		SessionID: "7f384610", State: "busy", LastSeen: "2026-06-27T12:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}
	r, ok, err := s.GetSession("7f384610")
	if err != nil || !ok {
		t.Fatalf("get: ok=%v err=%v", ok, err)
	}
	if r.FirstSeen != "2026-06-27T09:00:00Z" {
		t.Fatalf("first_seen = %q, want preserved 09:00", r.FirstSeen)
	}
	if r.State != "busy" || r.LastSeen != "2026-06-27T12:00:00Z" {
		t.Fatalf("state/last_seen not refreshed: %+v", r)
	}
	if r.Provider != "claude" {
		t.Fatalf("provider should carry over: %q", r.Provider)
	}
}

func TestRegistryList(t *testing.T) {
	s, _ := OpenStore(t.TempDir())
	_ = s.UpsertSession(SessionRecord{SessionID: "bbb", FirstSeen: "x", LastSeen: "x"})
	_ = s.UpsertSession(SessionRecord{SessionID: "aaa", FirstSeen: "x", LastSeen: "x"})
	list, err := s.ListSessions()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].SessionID != "aaa" {
		t.Fatalf("list not sorted/complete: %+v", list)
	}
}
