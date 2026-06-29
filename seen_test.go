package main

import (
	"net"
	"testing"
)

func TestMarkSeen(t *testing.T) {
	s, _ := OpenStore(t.TempDir())
	if !s.markSeen("abc") {
		t.Fatal("1re fois → doit être nouveau")
	}
	if s.markSeen("abc") {
		t.Fatal("2e fois → doit être déjà vu")
	}
	if !s.markSeen("xyz") {
		t.Fatal("autre clé → nouveau")
	}
}

func TestNetworkDedup(t *testing.T) {
	s, _ := OpenStore(t.TempDir())
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, e := ln.Accept()
			if e != nil {
				return
			}
			handleBusConn(s, c, nil)
		}
	}()
	addr := ln.Addr().String()

	m := InboxMessage{ID: "dup1", To: "B", Body: "once"}
	if err := sendRemote(addr, m, nil); err != nil {
		t.Fatal(err)
	}
	if err := sendRemote(addr, m, nil); err != nil { // replay du même message
		t.Fatal(err)
	}
	msgs, _ := s.Inbox().Receive("B", true)
	if len(msgs) != 1 {
		t.Fatalf("anti-replay: %d message(s) livré(s), want 1", len(msgs))
	}
}
