package main

import (
	"bytes"
	"crypto/ed25519"
	"testing"

	"filippo.io/mldsa"
)

func TestSealOpenRoundtrip(t *testing.T) {
	alice, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	bob, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte("dis à SACEM de lancer le build — broadcast famille")

	sealed, err := Seal(bob.Public(), alice, msg)
	if err != nil {
		t.Fatal(err)
	}
	pt, senderPub, err := Open(bob, sealed)
	if err != nil {
		t.Fatalf("Open a échoué: %v", err)
	}
	if !bytes.Equal(pt, msg) {
		t.Fatalf("plaintext = %q, want %q", pt, msg)
	}
	if !bytes.Equal(senderPub, alice.Public().SignPub) {
		t.Fatal("senderPub ne correspond pas à l'expéditeur")
	}
}

func TestOpenRejectsTamper(t *testing.T) {
	alice, _ := NewIdentity()
	bob, _ := NewIdentity()
	sealed, err := Seal(bob.Public(), alice, []byte("ordre signé"))
	if err != nil {
		t.Fatal(err)
	}
	// Flip a byte in the ciphertext: signature must fail.
	sealed.Ct[0] ^= 0xFF
	if _, _, err := Open(bob, sealed); err == nil {
		t.Fatal("Open a accepté un message falsifié")
	}
}

func TestOpenRejectsWrongRecipient(t *testing.T) {
	alice, _ := NewIdentity()
	bob, _ := NewIdentity()
	eve, _ := NewIdentity()
	sealed, _ := Seal(bob.Public(), alice, []byte("pour Bob seulement"))
	if _, _, err := Open(eve, sealed); err == nil {
		t.Fatal("Eve a pu ouvrir un message destiné à Bob")
	}
}

// §41 : un message scellé pour un couple from→to ne doit PAS s'ouvrir/vérifier sous
// un autre couple — l'AAD (liée dans l'AEAD ET la signature) interdit le ré-emballage.
func TestOpenRejectsRewrappedUnderDifferentIdentities(t *testing.T) {
	alice, _ := NewIdentity()
	bob, _ := NewIdentity()
	sealed, err := Seal(bob.Public(), alice, []byte("ordre"), sealAAD("alice", "bob"))
	if err != nil {
		t.Fatal(err)
	}
	// Même chiffré, ouvert sous un expéditeur usurpé (eve→bob) : doit échouer.
	if _, _, err := Open(bob, sealed, sealAAD("eve", "bob")); err == nil {
		t.Fatal("message ré-emballé sous une autre identité accepté")
	}
	// Le bon couple ouvre normalement.
	if _, _, err := Open(bob, sealed, sealAAD("alice", "bob")); err != nil {
		t.Fatalf("le bon aad devrait ouvrir: %v", err)
	}
}

func TestVaultRoundtrip(t *testing.T) {
	secret := []byte("clé privée très secrète")
	pass := []byte("corret-horse-battery-staple")
	blob, err := SealVault(secret, pass)
	if err != nil {
		t.Fatal(err)
	}
	got, err := OpenVault(blob, pass)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, secret) {
		t.Fatalf("vault roundtrip: got %q", got)
	}
	if _, err := OpenVault(blob, []byte("mauvaise passphrase")); err == nil {
		t.Fatal("vault ouvert avec une mauvaise passphrase")
	}
}

func TestIdentitySerializationRoundtrip(t *testing.T) {
	id, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	data, err := id.MarshalSecret()
	if err != nil {
		t.Fatal(err)
	}
	id2, err := UnmarshalIdentity(data)
	if err != nil {
		t.Fatal(err)
	}
	// Prove the rebuilt identity is functionally identical: a message sealed to the
	// ORIGINAL public bundle must open with the REBUILT private identity.
	sender, _ := NewIdentity()
	sealed, err := Seal(id.Public(), sender, []byte("persistence check"))
	if err != nil {
		t.Fatal(err)
	}
	pt, _, err := Open(id2, sealed)
	if err != nil {
		t.Fatalf("identité reconstruite ne déchiffre pas: %v", err)
	}
	if string(pt) != "persistence check" {
		t.Fatalf("got %q", pt)
	}
}

// --- Signature hybride Ed25519 ⊕ ML-DSA-65 (durcissement post-quantique) ---

func TestPublicBundleIncludesMLDSAKey(t *testing.T) {
	id, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	pub := id.Public()
	if len(pub.MLDSAPub) != mldsa.MLDSA65().PublicKeySize() {
		t.Fatalf("MLDSAPub: got %d octets, want %d (ML-DSA-65)", len(pub.MLDSAPub), mldsa.MLDSA65().PublicKeySize())
	}
}

func TestSealedMessageIncludesBothSignatures(t *testing.T) {
	alice, _ := NewIdentity()
	bob, _ := NewIdentity()
	sealed, err := Seal(bob.Public(), alice, []byte("double signature"))
	if err != nil {
		t.Fatal(err)
	}
	if len(sealed.Sig) != ed25519.SignatureSize {
		t.Fatalf("Sig (Ed25519): got %d octets, want %d", len(sealed.Sig), ed25519.SignatureSize)
	}
	if len(sealed.MLDSASig) != mldsa.MLDSA65().SignatureSize() {
		t.Fatalf("MLDSASig: got %d octets, want %d (ML-DSA-65)", len(sealed.MLDSASig), mldsa.MLDSA65().SignatureSize())
	}
	if len(sealed.SenderMLDSAPub) != mldsa.MLDSA65().PublicKeySize() {
		t.Fatalf("SenderMLDSAPub: got %d octets, want %d", len(sealed.SenderMLDSAPub), mldsa.MLDSA65().PublicKeySize())
	}
}

// §65/durcissement : falsifier la signature ML-DSA SEULE (Ed25519 intacte) doit
// suffire à faire échouer Open — c'est tout l'intérêt de l'hybride (« il faut casser
// les DEUX pour usurper »), pas seulement Ed25519.
func TestOpenRejectsTamperedMLDSASignatureAlone(t *testing.T) {
	alice, _ := NewIdentity()
	bob, _ := NewIdentity()
	sealed, err := Seal(bob.Public(), alice, []byte("ordre signé hybride"))
	if err != nil {
		t.Fatal(err)
	}
	sealed.MLDSASig[0] ^= 0xFF // Ed25519 (Sig) reste valide, seul ML-DSA est corrompu
	if _, _, err := Open(bob, sealed); err == nil {
		t.Fatal("Open a accepté une signature ML-DSA falsifiée alors qu'Ed25519 seule était valide")
	}
}

// Symétriquement : falsifier Ed25519 SEULE (ML-DSA intacte) doit aussi échouer.
func TestOpenRejectsTamperedEd25519SignatureAlone(t *testing.T) {
	alice, _ := NewIdentity()
	bob, _ := NewIdentity()
	sealed, err := Seal(bob.Public(), alice, []byte("ordre signé hybride"))
	if err != nil {
		t.Fatal(err)
	}
	sealed.Sig[0] ^= 0xFF // ML-DSA reste valide, seul Ed25519 est corrompu
	if _, _, err := Open(bob, sealed); err == nil {
		t.Fatal("Open a accepté une signature Ed25519 falsifiée alors que ML-DSA seule était valide")
	}
}

// Un expéditeur qui substitue sa propre clé ML-DSA (sans en posséder la clé privée
// assortie) est rejeté : la clé publique ML-DSA est liée dans le transcript ET la
// signature Ed25519 s'engage dessus — un ré-emballage de bundle ne vérifie plus.
func TestOpenRejectsSubstitutedMLDSAPublicKey(t *testing.T) {
	alice, _ := NewIdentity()
	mallory, _ := NewIdentity()
	bob, _ := NewIdentity()
	sealed, err := Seal(bob.Public(), alice, []byte("usurpation de clé ML-DSA"))
	if err != nil {
		t.Fatal(err)
	}
	sealed.SenderMLDSAPub = mallory.Public().MLDSAPub
	if _, _, err := Open(bob, sealed); err == nil {
		t.Fatal("Open a accepté une clé publique ML-DSA substituée")
	}
}
