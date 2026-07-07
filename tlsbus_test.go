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
	if err := sendRemote(addr, InboxMessage{From: "A", To: "B", Body: "tls hi"}, tlsClientConfig(fp, nil)); err != nil {
		t.Fatalf("livraison TLS (bon pin): %v", err)
	}
	msgs, _ := s.Inbox().Receive("B", true)
	if len(msgs) != 1 || msgs[0].Body != "tls hi" {
		t.Fatalf("non livré via TLS: %+v", msgs)
	}

	// Mauvais pin → rejeté (pas de livraison).
	if err := sendRemote(addr, InboxMessage{From: "A", To: "B", Body: "x"}, tlsClientConfig("deadbeefdeadbeef", nil)); err == nil {
		t.Fatal("un fingerprint incorrect a été accepté")
	}
}

// --- Authentification mutuelle (serve --tls --authz) ---

func TestMutualTLSAcceptsAllowedClient(t *testing.T) {
	s, _ := OpenStore(t.TempDir())
	allowedID, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	allow := map[string]bool{pubFingerprint(allowedID.Public().SignPub): true}

	cfg, fp, err := serverTLSConfigMutual(s, allow)
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
			handleBusConn(s, c, allow)
		}
	}()
	addr := ln.Addr().String()

	clientCert, err := clientTLSCert(allowedID)
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := Seal(allowedID.Public(), allowedID, []byte("mutual hi"), sealAAD("A", "B"))
	if err != nil {
		t.Fatal(err)
	}
	m := InboxMessage{From: "A", To: "B", Sealed: sealed}
	if err := sendRemote(addr, m, tlsClientConfig(fp, &clientCert)); err != nil {
		t.Fatalf("client autorisé rejeté: %v", err)
	}
}

func TestMutualTLSRejectsUnknownClient(t *testing.T) {
	s, _ := OpenStore(t.TempDir())
	allowedID, _ := NewIdentity()
	strangerID, _ := NewIdentity() // PAS dans l'allowlist
	allow := map[string]bool{pubFingerprint(allowedID.Public().SignPub): true}

	cfg, fp, err := serverTLSConfigMutual(s, allow)
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
			handleBusConn(s, c, allow)
		}
	}()
	addr := ln.Addr().String()

	clientCert, err := clientTLSCert(strangerID)
	if err != nil {
		t.Fatal(err)
	}
	m := InboxMessage{From: "eve", To: "B", Body: "intrusion"}
	if err := sendRemote(addr, m, tlsClientConfig(fp, &clientCert)); err == nil {
		t.Fatal("un certificat client hors allowlist a été accepté au handshake TLS")
	}
}

func TestMutualTLSRejectsNoClientCert(t *testing.T) {
	s, _ := OpenStore(t.TempDir())
	allowedID, _ := NewIdentity()
	allow := map[string]bool{pubFingerprint(allowedID.Public().SignPub): true}

	cfg, fp, err := serverTLSConfigMutual(s, allow)
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
			handleBusConn(s, c, allow)
		}
	}()
	addr := ln.Addr().String()

	// Aucun certificat client présenté (nil) alors que le serveur l'exige.
	m := InboxMessage{From: "A", To: "B", Body: "sans cert"}
	if err := sendRemote(addr, m, tlsClientConfig(fp, nil)); err == nil {
		t.Fatal("une connexion sans certificat client a été acceptée alors que l'auth mutuelle est active")
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
