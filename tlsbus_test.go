package main

import (
	"crypto/tls"
	"testing"
)

func TestNetworkTLSPinning(t *testing.T) {
	s, _ := OpenStore(t.TempDir())
	cfg, fp, err := serverTLSConfig(s)
	if err != nil {
		t.Fatal(err)
	}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", cfg)
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

	// Bon pin → handshake TLS + livraison.
	if err := sendRemote(addr, InboxMessage{From: "A", To: "B", Body: "tls hi"}, tlsClientConfig(fp)); err != nil {
		t.Fatalf("livraison TLS (bon pin): %v", err)
	}
	msgs, _ := s.Inbox().Receive("B", true)
	if len(msgs) != 1 || msgs[0].Body != "tls hi" {
		t.Fatalf("non livré via TLS: %+v", msgs)
	}

	// Mauvais pin → rejeté (pas de livraison).
	if err := sendRemote(addr, InboxMessage{From: "A", To: "B", Body: "x"}, tlsClientConfig("deadbeefdeadbeef")); err == nil {
		t.Fatal("un fingerprint incorrect a été accepté")
	}
}

func TestServerCertPersists(t *testing.T) {
	s, _ := OpenStore(t.TempDir())
	_, fp1, err := loadOrCreateServerCert(s)
	if err != nil {
		t.Fatal(err)
	}
	_, fp2, err := loadOrCreateServerCert(s)
	if err != nil {
		t.Fatal(err)
	}
	if fp1 == "" || fp1 != fp2 {
		t.Fatalf("le certificat doit persister entre deux appels: %q vs %q", fp1, fp2)
	}
}
