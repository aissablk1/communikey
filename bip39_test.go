package main

import (
	"bytes"
	"crypto/rand"
	"strings"
	"testing"
)

// Official BIP-39 test vectors (Trezor reference set).
func TestBIP39KnownVectors(t *testing.T) {
	cases := []struct {
		entropy  string // hex-ish via bytes below
		bytes    []byte
		mnemonic string
	}{
		{
			bytes:    bytes.Repeat([]byte{0x00}, 16),
			mnemonic: "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about",
		},
		{
			bytes:    bytes.Repeat([]byte{0xff}, 16),
			mnemonic: "zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo wrong",
		},
		{
			bytes:    bytes.Repeat([]byte{0x00}, 32),
			mnemonic: "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon art",
		},
	}
	for _, c := range cases {
		got, err := EntropyToMnemonic(c.bytes)
		if err != nil {
			t.Fatal(err)
		}
		if got != c.mnemonic {
			t.Fatalf("EntropyToMnemonic mismatch:\n got: %s\nwant: %s", got, c.mnemonic)
		}
		back, err := MnemonicToEntropy(c.mnemonic)
		if err != nil {
			t.Fatalf("MnemonicToEntropy(%q): %v", c.mnemonic, err)
		}
		if !bytes.Equal(back, c.bytes) {
			t.Fatalf("roundtrip entropy mismatch")
		}
	}
}

func TestBIP39Roundtrip32(t *testing.T) {
	seed := make([]byte, 32)
	if _, err := rand.Read(seed); err != nil {
		t.Fatal(err)
	}
	m, err := EntropyToMnemonic(seed)
	if err != nil {
		t.Fatal(err)
	}
	if len(strings.Fields(m)) != 24 {
		t.Fatalf("32 bytes → %d mots, want 24", len(strings.Fields(m)))
	}
	back, err := MnemonicToEntropy(m)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(back, seed) {
		t.Fatal("roundtrip 32 octets a échoué")
	}
}

func TestBIP39RejectsBadChecksum(t *testing.T) {
	// Swap a word for another valid word → checksum must fail.
	bad := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon zoo"
	if _, err := MnemonicToEntropy(bad); err == nil {
		t.Fatal("checksum invalide non détecté")
	}
}

func TestBIP39RejectsUnknownWord(t *testing.T) {
	if _, err := MnemonicToEntropy("abandon abandon notaword"); err == nil {
		t.Fatal("mot hors wordlist non rejeté")
	}
}

// The identity must survive a full BIP-39 round trip: master → phrase → master →
// derived identity opens a message sealed to the ORIGINAL identity.
func TestIdentityRecoverableFromPhrase(t *testing.T) {
	id, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	master, err := id.MarshalSecret()
	if err != nil {
		t.Fatal(err)
	}
	phrase, err := EntropyToMnemonic(master)
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := MnemonicToEntropy(phrase)
	if err != nil {
		t.Fatal(err)
	}
	id2, err := UnmarshalIdentity(recovered)
	if err != nil {
		t.Fatal(err)
	}
	sender, _ := NewIdentity()
	sealed, _ := Seal(id.Public(), sender, []byte("recovered by phrase"))
	pt, _, err := Open(id2, sealed)
	if err != nil || string(pt) != "recovered by phrase" {
		t.Fatalf("identité non récupérée via phrase: %v / %q", err, pt)
	}
}
