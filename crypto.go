package main

// crypto.go — couche 0 (sécurité) du bus communikey.
//
// Primitives AUDITÉES uniquement (§38 : jamais de crypto maison) :
//   · signature       Ed25519 ⊕ ML-DSA-65     (crypto/ed25519, filippo.io/mldsa) ← hybride PQC
//   · KEM classique   X25519                  (crypto/ecdh)
//   · KEM post-quant. ML-KEM-768              (crypto/mlkem)        ← hybride PQC
//   · AEAD            AES-256-GCM             (crypto/aes+cipher)
//   · KDF             HKDF-SHA256 / PBKDF2    (crypto/hkdf, pbkdf2)
//
// Toutes ces primitives sont dans la stdlib Go 1.25, SAUF la signature post-quantique
// ML-DSA-65 : la stdlib Go n'expose pas encore crypto/mldsa publiquement (implémentation
// interne depuis Go 1.26 ; paquet public proposé pour Go 1.27 — golang/go#77626). En
// attendant, on utilise filippo.io/mldsa (mainteneur crypto de l'équipe Go ; le paquet
// est explicitement conçu comme un pont vers l'API stdlib finale — migration = simple
// changement d'import path). C'est la SEULE dépendance externe du projet, épinglée dans
// go.sum — aucune autre crypto maison ou tierce.
//
// Un message est chiffré DE BOUT EN BOUT : le bus/relais ne voit que du chiffré
// signé (zero-trust). Le secret de session dérive de DEUX KEM combinés (X25519 ⊕
// ML-KEM) — il faut casser les deux pour lire (résistance « Harvest Now Decrypt
// Later », §38.7). Symétriquement, la signature combine Ed25519 ⊕ ML-DSA-65 sur le
// MÊME transcript : il faut casser les DEUX schémas pour usurper un expéditeur.

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/hkdf"
	"crypto/mlkem"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"

	"filippo.io/mldsa"
)

const hkdfInfo = "communikey/v1 hybrid-seal"

// Identity is an agent's long-term key material. The private halves live ONLY in
// the vault (SealVault); peers need only the PublicBundle to send to this identity.
type Identity struct {
	Sign   ed25519.PrivateKey
	MLDSA  *mldsa.PrivateKey // post-quantum signature, hybrid alongside Sign (Ed25519)
	X25519 *ecdh.PrivateKey
	MLKEM  *mlkem.DecapsulationKey768
	Master []byte // 32-byte seed everything derives from (recovery anchor)
}

// PublicBundle is the shareable public identity (what a sender needs).
type PublicBundle struct {
	SignPub   []byte `json:"sign_pub"`   // 32
	MLDSAPub  []byte `json:"mldsa_pub"`  // 1952 (ML-DSA-65)
	X25519Pub []byte `json:"x25519_pub"` // 32
	MLKEMPub  []byte `json:"mlkem_pub"`  // 1184
}

// SealedMessage is the on-wire E2E envelope. None of it reveals the plaintext.
type SealedMessage struct {
	EphX25519      []byte `json:"eph_x25519"`       // ephemeral X25519 public key
	MLKEMCt        []byte `json:"mlkem_ct"`         // ML-KEM ciphertext
	Nonce          []byte `json:"nonce"`            // AES-GCM nonce
	Ct             []byte `json:"ct"`               // AES-GCM ciphertext+tag
	SenderPub      []byte `json:"sender_pub"`       // sender Ed25519 public key
	SenderMLDSAPub []byte `json:"sender_mldsa_pub"` // sender ML-DSA-65 public key
	Sig            []byte `json:"sig"`              // Ed25519 signature over the transcript
	MLDSASig       []byte `json:"mldsa_sig"`        // ML-DSA-65 signature over the SAME transcript
}

// NewIdentity generates a fresh hybrid identity from a random 32-byte master seed.
func NewIdentity() (*Identity, error) {
	master := make([]byte, 32)
	if _, err := rand.Read(master); err != nil {
		return nil, err
	}
	return deriveIdentity(master)
}

// deriveIdentity deterministically derives the WHOLE hybrid identity from a single
// 32-byte master seed (domain-separated HKDF per key). One seed ⇒ one identity ⇒
// recoverable from one BIP-39 phrase or one Shamir secret.
func deriveIdentity(master []byte) (*Identity, error) {
	if len(master) != 32 {
		return nil, errors.New("graine maître: 32 octets requis")
	}
	edSeed, err := hkdf.Key(sha256.New, master, nil, "communikey/id/ed25519", 32)
	if err != nil {
		return nil, err
	}
	mldsaSeed, err := hkdf.Key(sha256.New, master, nil, "communikey/id/mldsa", 32)
	if err != nil {
		return nil, err
	}
	xSeed, err := hkdf.Key(sha256.New, master, nil, "communikey/id/x25519", 32)
	if err != nil {
		return nil, err
	}
	mlSeed, err := hkdf.Key(sha256.New, master, nil, "communikey/id/mlkem", 64)
	if err != nil {
		return nil, err
	}
	mldsaPriv, err := mldsa.NewPrivateKey(mldsa.MLDSA65(), mldsaSeed)
	if err != nil {
		return nil, err
	}
	xPriv, err := ecdh.X25519().NewPrivateKey(xSeed)
	if err != nil {
		return nil, err
	}
	dk, err := mlkem.NewDecapsulationKey768(mlSeed)
	if err != nil {
		return nil, err
	}
	return &Identity{
		Sign:   ed25519.NewKeyFromSeed(edSeed),
		MLDSA:  mldsaPriv,
		X25519: xPriv,
		MLKEM:  dk,
		Master: append([]byte(nil), master...),
	}, nil
}

// Public returns the shareable public bundle.
func (id *Identity) Public() PublicBundle {
	return PublicBundle{
		SignPub:   id.Sign.Public().(ed25519.PublicKey),
		MLDSAPub:  id.MLDSA.PublicKey().Bytes(),
		X25519Pub: id.X25519.PublicKey().Bytes(),
		MLKEMPub:  id.MLKEM.EncapsulationKey().Bytes(),
	}
}

// transcript is the exact byte sequence that is both AEAD-bound and doubly signed
// (Ed25519 ⊕ ML-DSA-65). Il lie désormais AUSSI les DEUX clés publiques de
// l'expéditeur et l'AAD applicative (from→to) : chaque signature s'engage sur QUI
// parle et À QUI — un message ré-emballé sous une autre identité (nouveau From/To)
// ne vérifie plus (durcissement anti-replay §41).
func transcript(ephPub, mlkemCt, nonce, ct, senderPub, senderMLDSAPub, aad []byte) []byte {
	t := make([]byte, 0, len(ephPub)+len(mlkemCt)+len(nonce)+len(ct)+len(senderPub)+len(senderMLDSAPub)+len(aad))
	t = append(t, ephPub...)
	t = append(t, mlkemCt...)
	t = append(t, nonce...)
	t = append(t, ct...)
	t = append(t, senderPub...)
	t = append(t, senderMLDSAPub...)
	t = append(t, aad...)
	return t
}

// sealAAD lie l'identité applicative (expéditeur → destinataire) au message. Passée
// à Seal/Open, elle est liée dans l'AEAD ET la signature. Couple vide → nil (rétro-
// compat des appels sans contexte applicatif, ex. tests crypto purs).
func sealAAD(from, to string) []byte {
	if from == "" && to == "" {
		return nil
	}
	return []byte(from + "\x1f" + to)
}

// firstAAD extrait l'AAD optionnelle (variadique → 0 ou 1 valeur).
func firstAAD(aad [][]byte) []byte {
	if len(aad) > 0 {
		return aad[0]
	}
	return nil
}

func deriveKey(xShared, mlShared []byte) ([]byte, error) {
	secret := append(append([]byte{}, xShared...), mlShared...)
	return hkdf.Key(sha256.New, secret, nil, hkdfInfo, 32)
}

// Seal encrypts plaintext to `to` and signs it as `from`. Hybrid KEM: the AEAD key
// derives from X25519 ⊕ ML-KEM, so confidentiality holds unless BOTH are broken.
// Hybrid signature: the transcript is signed by BOTH Ed25519 and ML-DSA-65, so
// forging a sender requires breaking BOTH schemes.
func Seal(to PublicBundle, from *Identity, plaintext []byte, aad ...[]byte) (*SealedMessage, error) {
	eph, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	toX, err := ecdh.X25519().NewPublicKey(to.X25519Pub)
	if err != nil {
		return nil, fmt.Errorf("clé X25519 destinataire invalide: %w", err)
	}
	xShared, err := eph.ECDH(toX)
	if err != nil {
		return nil, err
	}
	toML, err := mlkem.NewEncapsulationKey768(to.MLKEMPub)
	if err != nil {
		return nil, fmt.Errorf("clé ML-KEM destinataire invalide: %w", err)
	}
	mlShared, mlCt := toML.Encapsulate()

	key, err := deriveKey(xShared, mlShared)
	if err != nil {
		return nil, err
	}
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	a := firstAAD(aad)
	ct := gcm.Seal(nil, nonce, plaintext, a)

	ephPub := eph.PublicKey().Bytes()
	senderPub := from.Sign.Public().(ed25519.PublicKey)
	senderMLDSAPub := from.MLDSA.PublicKey().Bytes()
	tr := transcript(ephPub, mlCt, nonce, ct, senderPub, senderMLDSAPub, a)
	sig := ed25519.Sign(from.Sign, tr)
	mldsaSig, err := from.MLDSA.SignDeterministic(tr, &mldsa.Options{})
	if err != nil {
		return nil, fmt.Errorf("signature ML-DSA: %w", err)
	}
	return &SealedMessage{
		EphX25519: ephPub, MLKEMCt: mlCt, Nonce: nonce, Ct: ct,
		SenderPub: senderPub, SenderMLDSAPub: senderMLDSAPub,
		Sig: sig, MLDSASig: mldsaSig,
	}, nil
}

// Open verifies the sender signatures — Ed25519 AND ML-DSA-65, BOTH must be valid —
// then decrypts. Returns the plaintext and the sender's Ed25519 public key (for
// authorization decisions by the caller).
func Open(id *Identity, m *SealedMessage, aad ...[]byte) (plaintext, senderPub []byte, err error) {
	a := firstAAD(aad)
	if len(m.SenderPub) != ed25519.PublicKeySize {
		return nil, nil, errors.New("clé d'expéditeur invalide")
	}
	mldsaPub, err := mldsa.NewPublicKey(mldsa.MLDSA65(), m.SenderMLDSAPub)
	if err != nil {
		return nil, nil, fmt.Errorf("clé ML-DSA d'expéditeur invalide: %w", err)
	}
	tr := transcript(m.EphX25519, m.MLKEMCt, m.Nonce, m.Ct, m.SenderPub, m.SenderMLDSAPub, a)
	if !ed25519.Verify(m.SenderPub, tr, m.Sig) {
		return nil, nil, errors.New("signature Ed25519 invalide (message falsifié, mauvais expéditeur, ou ré-emballé)")
	}
	if err := mldsa.Verify(mldsaPub, tr, m.MLDSASig, &mldsa.Options{}); err != nil {
		return nil, nil, fmt.Errorf("signature ML-DSA invalide — hybride : les DEUX signatures doivent être valides (%w)", err)
	}
	ephPub, err := ecdh.X25519().NewPublicKey(m.EphX25519)
	if err != nil {
		return nil, nil, err
	}
	xShared, err := id.X25519.ECDH(ephPub)
	if err != nil {
		return nil, nil, err
	}
	mlShared, err := id.MLKEM.Decapsulate(m.MLKEMCt)
	if err != nil {
		return nil, nil, err
	}
	key, err := deriveKey(xShared, mlShared)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := newGCM(key)
	if err != nil {
		return nil, nil, err
	}
	pt, err := gcm.Open(nil, m.Nonce, m.Ct, a)
	if err != nil {
		return nil, nil, fmt.Errorf("déchiffrement échoué: %w", err)
	}
	return pt, m.SenderPub, nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// --- Vault: identity private keys at rest, sealed by a passphrase ---

// pbkdf2Iters is deliberately high (OWASP-grade for PBKDF2-SHA256). Argon2id is the
// preferred upgrade (needs golang.org/x/crypto) — tracked in the design (§38).
const pbkdf2Iters = 600_000

// VaultBlob is the encrypted-at-rest serialization of secret material.
type VaultBlob struct {
	Salt  []byte `json:"salt"`
	Nonce []byte `json:"nonce"`
	Ct    []byte `json:"ct"`
}

// SealVault encrypts plaintext under a passphrase (PBKDF2-SHA256 → AES-256-GCM).
func SealVault(plaintext, passphrase []byte) (*VaultBlob, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	key, err := pbkdf2.Key(sha256.New, string(passphrase), salt, pbkdf2Iters, 32)
	if err != nil {
		return nil, err
	}
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return &VaultBlob{Salt: salt, Nonce: nonce, Ct: gcm.Seal(nil, nonce, plaintext, nil)}, nil
}

// OpenVault reverses SealVault.
func OpenVault(b *VaultBlob, passphrase []byte) ([]byte, error) {
	key, err := pbkdf2.Key(sha256.New, string(passphrase), b.Salt, pbkdf2Iters, 32)
	if err != nil {
		return nil, err
	}
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	pt, err := gcm.Open(nil, b.Nonce, b.Ct, nil)
	if err != nil {
		return nil, errors.New("vault: passphrase invalide ou coffre corrompu")
	}
	return pt, nil
}

// --- Identity serialization (private — only ever written inside a vault) ---

// MarshalSecret returns the 32-byte master seed — the single secret from which the
// whole identity derives. Wrap it with SealVault, split it with Shamir, or encode
// it as a BIP-39 phrase.
func (id *Identity) MarshalSecret() ([]byte, error) {
	if len(id.Master) != 32 {
		return nil, errors.New("identité sans graine maître")
	}
	return append([]byte(nil), id.Master...), nil
}

// UnmarshalIdentity rebuilds an Identity from its 32-byte master seed.
func UnmarshalIdentity(master []byte) (*Identity, error) {
	return deriveIdentity(master)
}
