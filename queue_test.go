package main

import (
	"net"
	"testing"
)

func TestOutboxEnqueueFlush(t *testing.T) {
	s, _ := OpenStore(t.TempDir())
	dead := "127.0.0.1:1" // refus de connexion immédiat

	_ = enqueueOutbox(s, dead, InboxMessage{ID: "q1", To: "B", Body: "a"})
	_ = enqueueOutbox(s, dead, InboxMessage{ID: "q2", To: "B", Body: "b"})
	if pendingOutbox(s, dead) != 2 {
		t.Fatalf("pending=%d want 2", pendingOutbox(s, dead))
	}
	if n := flushOutbox(s, dead, nil); n != 0 {
		t.Fatalf("flush vers une adresse morte a livré %d", n)
	}
	if pendingOutbox(s, dead) != 2 {
		t.Fatal("la file doit rester intacte après un flush échoué")
	}

	// Vrai serveur → le flush vide la file.
	srv, _ := OpenStore(t.TempDir())
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
			handleBusConn(srv, c, nil)
		}
	}()
	addr := ln.Addr().String()
	_ = enqueueOutbox(s, addr, InboxMessage{ID: "q3", To: "B", Body: "c"})
	if n := flushOutbox(s, addr, nil); n != 1 {
		t.Fatalf("flush a livré %d want 1", n)
	}
	if pendingOutbox(s, addr) != 0 {
		t.Fatal("la file doit être vide après un flush réussi")
	}
	if msgs, _ := srv.Inbox().Receive("B", true); len(msgs) != 1 || msgs[0].Body != "c" {
		t.Fatalf("le serveur n'a pas reçu le message rejoué: %+v", msgs)
	}
}
