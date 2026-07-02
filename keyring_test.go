package main

import (
	"os"
	"testing"
)

func TestContactAndTokenRoundtrip(t *testing.T) {
	s, _ := OpenStore(t.TempDir())
	id, _ := NewIdentity()
	b := id.Public()

	tok := encodeBundle(b)
	got, err := decodeBundle(tok)
	if err != nil {
		t.Fatal(err)
	}
	if fingerprint(got) != fingerprint(b) {
		t.Fatal("token roundtrip changed the bundle")
	}
	if err := s.SaveContact("BOB", b); err != nil {
		t.Fatal(err)
	}
	lb, ok := s.LoadContact("BOB")
	if !ok || fingerprint(lb) != fingerprint(b) {
		t.Fatal("contact roundtrip failed")
	}
	cs, _ := s.ListContacts()
	if len(cs) != 1 || cs[0] != "BOB" {
		t.Fatalf("ListContacts = %+v", cs)
	}
}

// The real E2E wiring: A (with B's public key) seals a message that only B can read.
func TestE2EWireCrossStore(t *testing.T) {
	os.Setenv("COMKEY_VAULT_PASS", "tp")
	os.Unsetenv("COMKEY_VAULT_PASS_FILE")
	os.Setenv("COMKEY_AGENT_ID", "A") // l'aad lie from→to : le scellement et la lecture doivent l'accorder
	defer os.Unsetenv("COMKEY_VAULT_PASS")
	defer os.Unsetenv("COMKEY_AGENT_ID")

	a, _ := OpenStore(t.TempDir()) // sender
	b, _ := OpenStore(t.TempDir()) // recipient
	idA, _ := NewIdentity()
	idB, _ := NewIdentity()
	if err := saveIdentity(a, idA, []byte("tp")); err != nil {
		t.Fatal(err)
	}
	if err := saveIdentity(b, idB, []byte("tp")); err != nil {
		t.Fatal(err)
	}
	if err := a.SaveContact("B", idB.Public()); err != nil {
		t.Fatal(err)
	}

	sealed, ok := maybeSeal(a, "B", "message secret")
	if !ok {
		t.Fatal("maybeSeal devrait sceller (contact + vault présents)")
	}
	if got := openBody(b, InboxMessage{From: "A", To: "B", Sealed: sealed}); got != "message secret" {
		t.Fatalf("openBody (recipient) = %q", got)
	}

	// Sans contact → pas de scellement (repli clair).
	if _, ok := maybeSeal(a, "INCONNU", "x"); ok {
		t.Fatal("maybeSeal ne devrait pas sceller sans contact")
	}
	// Message en clair lu tel quel.
	if openBody(b, InboxMessage{Body: "clair"}) != "clair" {
		t.Fatal("openBody plaintext")
	}
	// Un tiers (A) ne peut PAS ouvrir un message scellé pour B.
	if got := openBody(a, InboxMessage{From: "A", To: "B", Sealed: sealed}); got == "message secret" {
		t.Fatal("A n'aurait pas dû pouvoir déchiffrer un message destiné à B")
	}
}
