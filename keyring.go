package main

// keyring.go — annuaire de clés publiques des pairs + scellement E2E « sur le fil ».
//
// La crypto (crypto.go) existait ; ici on la BRANCHE sur le chemin réel des
// messages : si on connaît la clé publique du destinataire (un « contact ») et que
// notre vault est déverrouillable, `inbox`/`remote` scellent le corps avant de
// l'envoyer ; `recv` le déchiffre. Sinon, repli en clair (confiance locale).

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func (s *Store) contactsDir() string          { return filepath.Join(s.Dir, "contacts") }
func (s *Store) contactPath(a string) string  { return filepath.Join(s.contactsDir(), safeID(a)+".json") }

// SaveContact stores a peer's public bundle under its agent id.
func (s *Store) SaveContact(agent string, b PublicBundle) error {
	if err := os.MkdirAll(s.contactsDir(), 0o755); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(b, "", "  ")
	return os.WriteFile(s.contactPath(agent), data, 0o644)
}

// LoadContact returns a peer's public bundle, if known.
func (s *Store) LoadContact(agent string) (PublicBundle, bool) {
	data, err := os.ReadFile(s.contactPath(agent))
	if err != nil {
		return PublicBundle{}, false
	}
	var b PublicBundle
	if json.Unmarshal(data, &b) != nil {
		return PublicBundle{}, false
	}
	return b, true
}

// ListContacts returns the known peer agent ids.
func (s *Store) ListContacts() ([]string, error) {
	entries, err := os.ReadDir(s.contactsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			out = append(out, strings.TrimSuffix(e.Name(), ".json"))
		}
	}
	return out, nil
}

// encodeBundle renders a public bundle as one copy-pasteable token.
func encodeBundle(b PublicBundle) string {
	data, _ := json.Marshal(b)
	return "csend1:" + base64.RawURLEncoding.EncodeToString(data)
}

func decodeBundle(token string) (PublicBundle, error) {
	token = strings.TrimPrefix(strings.TrimSpace(token), "csend1:")
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return PublicBundle{}, err
	}
	var b PublicBundle
	if err := json.Unmarshal(raw, &b); err != nil {
		return PublicBundle{}, err
	}
	return b, nil
}

// maybeSeal seals body for toAgent IF a contact bundle AND an unlockable local
// identity both exist. Returns (sealed, true) on success, (nil, false) to fall back
// to plaintext (cooperative local trust).
func maybeSeal(s *Store, toAgent, body string) (*SealedMessage, bool) {
	bundle, ok := s.LoadContact(toAgent)
	if !ok {
		return nil, false
	}
	pass, ok := resolveVaultPass()
	if !ok {
		return nil, false
	}
	id, err := loadIdentity(s, pass)
	if err != nil {
		return nil, false
	}
	sealed, err := Seal(bundle, id, []byte(body))
	if err != nil {
		return nil, false
	}
	return sealed, true
}

// openBody returns a message's readable body, decrypting if it is sealed.
func openBody(s *Store, m InboxMessage) string {
	if m.Sealed == nil {
		return m.Body
	}
	pass, ok := resolveVaultPass()
	if !ok {
		return "[chiffré E2E — définis CSEND_VAULT_PASS(_FILE) pour lire]"
	}
	id, err := loadIdentity(s, pass)
	if err != nil {
		return "[chiffré E2E — vault inaccessible]"
	}
	pt, _, err := Open(id, m.Sealed)
	if err != nil {
		return "[chiffré E2E — déchiffrement échoué]"
	}
	return string(pt)
}
