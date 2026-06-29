package main

// seen.go — anti-replay. On enregistre une clé de déduplication par message livré
// (au réseau) ; un doublon est acquitté mais PAS re-livré. La création de fichier
// O_EXCL est atomique et lock-free → sûre entre process concurrents.
//
// Pour un message scellé, la clé est le NONCE AES-GCM : il est couvert par la
// signature Ed25519, donc un attaquant ne peut pas le modifier sans invalider la
// signature → l'anti-replay est cryptographiquement solide (pas un simple id From).

import (
	"os"
	"path/filepath"
)

func (s *Store) seenDir() string { return filepath.Join(s.Dir, "seen") }

// markSeen records key; returns true if it is NEW (first time), false if already
// seen. Empty keys can't be deduped (treated as new).
func (s *Store) markSeen(key string) bool {
	if key == "" {
		return true
	}
	if err := os.MkdirAll(s.seenDir(), 0o755); err != nil {
		return true // can't persist → fail open (deliver)
	}
	f, err := os.OpenFile(filepath.Join(s.seenDir(), safeID(key)), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return false // already exists → already seen
	}
	_ = f.Close()
	return true
}
