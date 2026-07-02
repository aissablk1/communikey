package main

// tlsbus.go — TLS 1.3 pour le réseau communikey, avec échange de clés HYBRIDE
// post-quantique. Go 1.24 négocie **X25519MLKEM768** par défaut en TLS 1.3 quand
// les deux pairs le supportent : le transport résiste donc à « Harvest Now, Decrypt
// Later » (§38.7) — en plus du payload déjà chiffrable E2E.
//
// Auth : certificat self-signed ed25519 (stocké dans le store) ; le client épingle
// le fingerprint du serveur (`--pin`) → confiance sans PKI. Sur loopback, le pin est
// facultatif.

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
// fingerprint (empty pin = accept, for loopback/trusted use).
func tlsClientConfig(pin string) *tls.Config {
	return &tls.Config{
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
}
