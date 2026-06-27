package main

import "testing"

func TestInboxDeliverReceive(t *testing.T) {
	s, _ := OpenStore(t.TempDir())
	ib := s.Inbox()

	if err := ib.Deliver(InboxMessage{ID: "m1", TS: "2026-06-27T10:00:00Z", From: "A", To: "SACEM", Body: "build"}); err != nil {
		t.Fatal(err)
	}
	if err := ib.Deliver(InboxMessage{ID: "m2", TS: "2026-06-27T09:00:00Z", From: "B", To: "SACEM", Body: "deploy"}); err != nil {
		t.Fatal(err)
	}

	// Pending count, and a peek (markRead=false) leaves them pending.
	if n, _ := ib.Pending("SACEM"); n != 2 {
		t.Fatalf("pending = %d, want 2", n)
	}
	peek, err := ib.Receive("SACEM", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(peek) != 2 || peek[0].ID != "m2" { // sorted by TS: 09:00 before 10:00
		t.Fatalf("peek order wrong: %+v", peek)
	}
	if n, _ := ib.Pending("SACEM"); n != 2 {
		t.Fatalf("peek must not consume; pending = %d, want 2", n)
	}

	// Consume (markRead=true) drains the mailbox.
	got, _ := ib.Receive("SACEM", true)
	if len(got) != 2 {
		t.Fatalf("consume got %d, want 2", len(got))
	}
	if n, _ := ib.Pending("SACEM"); n != 0 {
		t.Fatalf("after consume pending = %d, want 0", n)
	}
	again, _ := ib.Receive("SACEM", true)
	if len(again) != 0 {
		t.Fatalf("second consume got %d, want 0", len(again))
	}
}

func TestInboxIsolationPerAgent(t *testing.T) {
	s, _ := OpenStore(t.TempDir())
	ib := s.Inbox()
	_ = ib.Deliver(InboxMessage{ID: "x", TS: "t", To: "ALICE", Body: "hi"})
	if n, _ := ib.Pending("BOB"); n != 0 {
		t.Fatalf("BOB should have 0, got %d", n)
	}
	if n, _ := ib.Pending("ALICE"); n != 1 {
		t.Fatalf("ALICE should have 1, got %d", n)
	}
}

func TestInboxSafeIDNoTraversal(t *testing.T) {
	s, _ := OpenStore(t.TempDir())
	ib := s.Inbox()
	// A malicious recipient handle must not escape the inbox root.
	if err := ib.Deliver(InboxMessage{ID: "../../evil", TS: "t", To: "../../../etc", Body: "x"}); err != nil {
		t.Fatalf("deliver should sanitize, not error: %v", err)
	}
	// The sanitized recipient still receives it; no path traversal occurred.
	if n, _ := ib.Pending("../../../etc"); n != 1 {
		t.Fatalf("sanitized recipient pending = %d, want 1", n)
	}
}

func TestRouteDecision(t *testing.T) {
	cases := []struct {
		name string
		in   TargetInfo
		want Channel
	}{
		{"coop wins even if live", TargetInfo{Cooperates: true, Live: true}, ChannelInbox},
		{"bridge when only bridge", TargetInfo{Bridge: true}, ChannelBridge},
		{"inject when mute but live", TargetInfo{Live: true}, ChannelInject},
		{"queue when offline", TargetInfo{}, ChannelQueue},
		{"coop over bridge", TargetInfo{Cooperates: true, Bridge: true}, ChannelInbox},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Route(c.in); got != c.want {
				t.Fatalf("Route(%+v) = %s, want %s", c.in, got, c.want)
			}
		})
	}
}
