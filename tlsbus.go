package main

// tlsbus.go — TLS 1.3 pour le réseau communikey, avec échange de clés HYBRIDE
// post-quantique. Go 1.24 négocie **X25519MLKEM768** par défaut en TLS 1.3 quand
// les deux pairs le supportent : le transport résiste donc à « Harvest Now, Decrypt
// Later » (§38.7) — en plus du payload déjà chiffrable E2E.
//
// Auth serveur : certificat self-signed ed25519 (stocké dans le store) ; le client
// épingle le fingerprint du serveur (`--pin`) → confiance sans PKI. Sur loopback, le
// pin est facultatif.
//
// Auth MUTUELLE (client → serveur) : `serve --tls --authz` exige aussi un certificat
// CLIENT (serverTLSConfigMutual), vérifié contre la MÊME allowlist que --authz —
// un pair non autorisé est rejeté au handshake TLS, avant même la lecture du frame.
// Le certificat client dérive de la clé de signature Ed25519 de l'identité
// (clientTLSCert), généré en mémoire — jamais persisté en clair sur disque.

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func tlsDir(s *Store) string { return filepath.Join(s.Dir, "tls") }

func certFingerprint(der []byte) string {
	sum := sha256.Sum256(der)
	return hex.EncodeToString(sum[:])
}

// loadOrCreateServerCert returns the server's TLS cert (generating a self-signed
// ed25519 one on first use) and its sha256 fingerprint (for the client to pin).
func loadOrCreateServerCert(s *Store) (tls.Certificate, string, error) {
	dir := tlsDir(s)
	certP, keyP := filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem")
	if cp, err := os.ReadFile(certP); err == nil {
		if kp, err := os.ReadFile(keyP); err == nil {
			if cert, err := tls.X509KeyPair(cp, kp); err == nil {
				return cert, certFingerprint(cert.Certificate[0]), nil
			}
		}
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return tls.Certificate{}, "", err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, "", err
	}
	tmpl := x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "communikey"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, pub, priv)
	if err != nil {
		return tls.Certificate{}, "", err
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return tls.Certificate{}, "", err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	_ = os.MkdirAll(dir, 0o700)
	_ = os.WriteFile(certP, certPEM, 0o644)
	_ = os.WriteFile(keyP, keyPEM, 0o600)
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return tls.Certificate{}, "", err
	}
	return cert, certFingerprint(der), nil
}

// serverTLSConfig builds a TLS 1.3 server config (hybrid PQC by default in Go 1.24).
func serverTLSConfig(s *Store) (*tls.Config, string, error) {
	cert, fp, err := loadOrCreateServerCert(s)
	if err != nil {
		return nil, "", err
	}
	return &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS13}, fp, nil
}

// tlsClientConfig builds a TLS 1.3 client config that pins the server's cert
// fingerprint (empty pin = accept, for loopback/trusted use). clientCert is
// optional (nil = present no client certificate) — pass one from clientTLSCert
// to authenticate to a server enforcing mutual TLS (cf. serverTLSConfigMutual).
func tlsClientConfig(pin string, clientCert *tls.Certificate) *tls.Config {
	cfg := &tls.Config{
		InsecureSkipVerify: true, // we verify by pinning the leaf fingerprint ourselves
		MinVersion:         tls.VersionTLS13,
		VerifyConnection: func(cs tls.ConnectionState) error {
			if pin == "" {
				return nil
			}
			if len(cs.PeerCertificates) == 0 {
				return errors.New("aucun certificat serveur")
			}
			if got := certFingerprint(cs.PeerCertificates[0].Raw); !strings.EqualFold(got, pin) {
				return fmt.Errorf("fingerprint serveur %s ≠ pin attendu %s", got[:16], pin)
			}
			return nil
		},
	}
	if clientCert != nil {
		cfg.Certificates = []tls.Certificate{*clientCert}
	}
	return cfg
}

// clientTLSCert builds a self-signed TLS client certificate from the caller's
// Ed25519 SIGNING key (id.Sign) — its public key is exactly the identity's
// fingerprint (pubFingerprint, authz.go), so the SAME allowlist (allowed.json)
// gates both message-level authz (--authz) and TLS-layer mutual auth. Built
// ENTIRELY IN MEMORY: unlike the server's own throwaway cert (persisted to
// tls/key.pem, a key that exists ONLY for the transport), the identity's real
// signing key is never written to disk in clear — it lives only in the vault.
func clientTLSCert(id *Identity) (tls.Certificate, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, err
	}
	pub, ok := id.Sign.Public().(ed25519.PublicKey)
	if !ok {
		return tls.Certificate{}, errors.New("identité: clé de signature invalide")
	}
	tmpl := x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "communikey-client"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, pub, id.Sign)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: id.Sign}, nil
}

// serverTLSConfigMutual extends serverTLSConfig with MUTUAL TLS: the server
// REQUIRES a client certificate and verifies its Ed25519 public key's fingerprint
// against `allow` — the SAME allowlist `--authz` already uses for message
// signatures (authz.go, senderAllowed). An unauthorized peer is rejected at the
// TLS HANDSHAKE, before the connection ever reaches message framing/parsing —
// closing the gap noted in net.go's header (§38: any TCP peer could otherwise
// reach the framing code, even if only signed messages were ever accepted).
func serverTLSConfigMutual(s *Store, allow map[string]bool) (*tls.Config, string, error) {
	cfg, fp, err := serverTLSConfig(s)
	if err != nil {
		return nil, "", err
	}
	cfg.ClientAuth = tls.RequireAnyClientCert // no CA infra: we verify identity ourselves below
	cfg.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
		if len(rawCerts) == 0 {
			return errors.New("certificat client requis (authentification mutuelle active)")
		}
		cert, err := x509.ParseCertificate(rawCerts[0])
		if err != nil {
			return fmt.Errorf("certificat client invalide: %w", err)
		}
		pub, ok := cert.PublicKey.(ed25519.PublicKey)
		if !ok {
			return errors.New("certificat client: clé publique non-Ed25519")
		}
		if !allow[pubFingerprint(pub)] {
			return errors.New("certificat client: fingerprint non autorisé")
		}
		return nil
	}
	return cfg, fp, nil
}
