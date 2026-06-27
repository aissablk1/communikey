package main

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestShamirRoundtripAnyThresholdSubset(t *testing.T) {
	secret := make([]byte, 32) // e.g. a key seed
	if _, err := rand.Read(secret); err != nil {
		t.Fatal(err)
	}
	shares, err := ShamirSplit(secret, 5, 3) // 3-of-5
	if err != nil {
		t.Fatal(err)
	}
	if len(shares) != 5 {
		t.Fatalf("got %d shares, want 5", len(shares))
	}
	// Any 3 distinct shares must recover the secret.
	subsets := [][]int{{0, 1, 2}, {0, 2, 4}, {1, 3, 4}, {2, 3, 4}}
	for _, idx := range subsets {
		got, err := ShamirCombine([][]byte{shares[idx[0]], shares[idx[1]], shares[idx[2]]})
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, secret) {
			t.Fatalf("subset %v did not recover the secret", idx)
		}
	}
}

func TestShamirThresholdSecrecy(t *testing.T) {
	secret := []byte("clé-de-recovery-32-octets-exacte")
	shares, err := ShamirSplit(secret, 5, 3)
	if err != nil {
		t.Fatal(err)
	}
	// Fewer than threshold (2 of 3 needed) must NOT yield the secret.
	got, err := ShamirCombine([][]byte{shares[0], shares[1]})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(got, secret) {
		t.Fatal("2 parts (< seuil 3) ont révélé le secret — propriété de seuil violée")
	}
}

func TestShamirValidatesParams(t *testing.T) {
	if _, err := ShamirSplit([]byte("x"), 3, 1); err == nil {
		t.Fatal("seuil 1 doit être rejeté")
	}
	if _, err := ShamirSplit([]byte("x"), 2, 3); err == nil {
		t.Fatal("seuil > parts doit être rejeté")
	}
	if _, err := ShamirSplit(nil, 3, 2); err == nil {
		t.Fatal("secret vide doit être rejeté")
	}
}

func TestShamirRejectsDuplicateOrMalformed(t *testing.T) {
	shares, _ := ShamirSplit([]byte("secret"), 4, 2)
	if _, err := ShamirCombine([][]byte{shares[0], shares[0]}); err == nil {
		t.Fatal("coordonnées x dupliquées doivent être rejetées")
	}
}

// gfInv must be a true field inverse for every non-zero element.
func TestGFInverse(t *testing.T) {
	for a := 1; a < 256; a++ {
		if gfMul(byte(a), gfInv(byte(a))) != 1 {
			t.Fatalf("gfInv faux pour %d", a)
		}
	}
}
