package main

// bip39.go — phrases de récupération mnémoniques (standard BIP-39).
//
// Encode une entropie (128..256 bits) en une suite de mots d'une wordlist de 2048
// (11 bits/mot) + un checksum (ENT/32 bits, premiers bits de SHA-256). C'est le
// schéma éprouvé des wallets : une graine ↔ une phrase humaine, validée par
// checksum (un mot mal recopié est détecté). Wordlist anglaise officielle embarquée.

import (
	_ "embed"
	"crypto/sha256"
	"errors"
	"strings"
)

//go:embed bip39_english.txt
var bip39Raw string

var (
	bip39Words []string
	bip39Index map[string]int
)

func init() {
	bip39Words = strings.Fields(bip39Raw)
	bip39Index = make(map[string]int, len(bip39Words))
	for i, w := range bip39Words {
		bip39Index[w] = i
	}
}

// EntropyToMnemonic encodes entropy (16/20/24/28/32 bytes) as a BIP-39 phrase.
func EntropyToMnemonic(entropy []byte) (string, error) {
	ent := len(entropy) * 8
	if ent < 128 || ent > 256 || ent%32 != 0 {
		return "", errors.New("entropie: 128/160/192/224/256 bits requis")
	}
	if len(bip39Words) != 2048 {
		return "", errors.New("wordlist BIP-39 absente ou corrompue")
	}
	cs := ent / 32
	hash := sha256.Sum256(entropy)

	bits := make([]byte, 0, ent+cs)
	for _, b := range entropy {
		for i := 7; i >= 0; i-- {
			bits = append(bits, (b>>uint(i))&1)
		}
	}
	for i := 0; i < cs; i++ {
		bits = append(bits, (hash[i/8]>>uint(7-i%8))&1)
	}

	var words []string
	for i := 0; i < len(bits); i += 11 {
		idx := 0
		for j := 0; j < 11; j++ {
			idx = (idx << 1) | int(bits[i+j])
		}
		words = append(words, bip39Words[idx])
	}
	return strings.Join(words, " "), nil
}

// MnemonicToEntropy validates a BIP-39 phrase (checksum) and returns the entropy.
func MnemonicToEntropy(mnemonic string) ([]byte, error) {
	words := strings.Fields(strings.ToLower(strings.TrimSpace(mnemonic)))
	n := len(words)
	if n%3 != 0 || n < 12 || n > 24 {
		return nil, errors.New("phrase: 12/15/18/21/24 mots requis")
	}
	bits := make([]byte, 0, n*11)
	for _, w := range words {
		idx, ok := bip39Index[w]
		if !ok {
			return nil, errors.New("mot hors wordlist BIP-39: " + w)
		}
		for j := 10; j >= 0; j-- {
			bits = append(bits, byte((idx>>uint(j))&1))
		}
	}
	total := n * 11
	cs := total / 33
	ent := total - cs
	entropy := make([]byte, ent/8)
	for i := 0; i < ent; i++ {
		if bits[i] == 1 {
			entropy[i/8] |= 1 << uint(7-i%8)
		}
	}
	hash := sha256.Sum256(entropy)
	for i := 0; i < cs; i++ {
		if bits[ent+i] != (hash[i/8]>>uint(7-i%8))&1 {
			return nil, errors.New("checksum invalide — phrase incorrecte ou mal recopiée")
		}
	}
	return entropy, nil
}
