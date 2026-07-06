package main

// authz.go — autorisation réseau CRYPTOGRAPHIQUE.
//
// `serve --authz` n'accepte un message que s'il est E2E **signé** par un expéditeur
// dont le fingerprint figure dans l'allowlist. Les signatures Ed25519 ⊕ ML-DSA-65
// (sur le transcript) sont vérifiables SANS déchiffrer — donc le serveur authentifie
// l'expéditeur sans jamais lire le clair. Hybride : les DEUX signatures doivent être
// valides, sinon le message est refusé (même garde que Open(), crypto.go). Les
// messages en clair (non signés) sont refusés sous --authz. Un id `From` ne suffit
// jamais (il est falsifiable) : seule la clé qui signe compte.

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/mldsa"
)

func allowlistPath(s *Store) string { return filepath.Join(s.Dir, "allowed.json") }

// pubFingerprint is the short id of an Ed25519 public key — same scheme as the
// identity fingerprint shown by `communikey id`, so an allowed entry = a peer's fingerprint.
func pubFingerprint(signPub []byte) string {
	sum := sha256.Sum256(signPub)
	return hex.EncodeToString(sum[:8])
}

// loadAllowlist returns the set of allowed sender fingerprints (from allowed.json +
// extra flags). configured=false means no allowlist at all.
func loadAllowlist(s *Store, extra []string) (allow map[string]bool, configured bool) {
	allow = map[string]bool{}
	if data, err := os.ReadFile(allowlistPath(s)); err == nil {
		var fps []string
		if json.Unmarshal(data, &fps) == nil {
			for _, fp := range fps {
				allow[strings.ToLower(strings.TrimSpace(fp))] = true
				configured = true
			}
		}
	}
	for _, fp := range extra {
		allow[strings.ToLower(strings.TrimSpace(fp))] = true
		configured = true
	}
	return allow, configured
}

// senderAllowed verifies the message is E2E-signed (Ed25519 AND ML-DSA-65, BOTH
// required) by an allowed sender. Returns false for plaintext, bad/partial
// signatures, or unknown senders.
func senderAllowed(m InboxMessage, allow map[string]bool) bool {
	sm := m.Sealed
	if sm == nil || len(sm.SenderPub) != ed25519.PublicKeySize {
		return false
	}
	mldsaPub, err := mldsa.NewPublicKey(mldsa.MLDSA65(), sm.SenderMLDSAPub)
	if err != nil {
		return false
	}
	tr := transcript(sm.EphX25519, sm.MLKEMCt, sm.Nonce, sm.Ct, sm.SenderPub, sm.SenderMLDSAPub, sealAAD(m.From, m.To))
	if !ed25519.Verify(sm.SenderPub, tr, sm.Sig) {
		return false
	}
	if mldsa.Verify(mldsaPub, tr, sm.MLDSASig, &mldsa.Options{}) != nil {
		return false
	}
	return allow[pubFingerprint(sm.SenderPub)]
}
