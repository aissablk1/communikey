package main

import "testing"

func TestSenderAllowed(t *testing.T) {
	alice, _ := NewIdentity()
	bob, _ := NewIdentity()
	sealed, err := Seal(bob.Public(), alice, []byte("hi"))
	if err != nil {
		t.Fatal(err)
	}
	allow := map[string]bool{pubFingerprint(alice.Public().SignPub): true}

	if !senderAllowed(InboxMessage{To: "bob", Sealed: sealed}, allow) {
		t.Fatal("alice (autorisée, signée) devrait passer")
	}
	if senderAllowed(InboxMessage{To: "bob", Sealed: sealed}, map[string]bool{"deadbeef": true}) {
		t.Fatal("expéditeur hors allowlist accepté")
	}
	if senderAllowed(InboxMessage{To: "bob", Body: "x"}, allow) {
		t.Fatal("message en clair accepté sous authz")
	}
	sealed.Ct[0] ^= 0xFF // falsifie le chiffré → signature invalide
	if senderAllowed(InboxMessage{To: "bob", Sealed: sealed}, allow) {
		t.Fatal("signature falsifiée acceptée")
	}
}

func TestLoadAllowlist(t *testing.T) {
	s, _ := OpenStore(t.TempDir())
	if _, configured := loadAllowlist(s, nil); configured {
		t.Fatal("vide → non configuré")
	}
	a, configured := loadAllowlist(s, []string{"ABC123"})
	if !configured || !a["abc123"] {
		t.Fatalf("--allow non pris en compte / casse non normalisée: %v", a)
	}
}
