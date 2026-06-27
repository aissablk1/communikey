package main

// memory.go — couche 6 (mémoire) du bus csend.
//
// Store durable et interrogeable : « les sessions passent, la mémoire reste ».
//   · JOURNAL  append-only JSONL de tous les messages (qui→qui, quand, canal,
//              action, hash du texte — jamais le clair, §35) → interrogeable.
//   · REGISTRE map session-id → métadonnées (provider, workspace, états, vu le…).
//
// Conçu pour INTEROPÉRER avec claude-mem (§1 : on ne reconstruit pas un moteur
// sémantique) : le journal est l'enregistrement exportable.
//
// Le répertoire est INJECTABLE (OpenStore(dir)) pour des tests hermétiques.

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Store is a directory-backed persistent journal + session registry.
type Store struct {
	Dir string
}

// OpenStore ensures dir exists and returns a Store.
func OpenStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{Dir: dir}, nil
}

// DefaultStoreDir is ~/.claude/csend.
func DefaultStoreDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "csend")
}

func (s *Store) journalPath() string  { return filepath.Join(s.Dir, "journal.jsonl") }
func (s *Store) registryPath() string { return filepath.Join(s.Dir, "sessions.json") }

// --- Journal (append-only) ---

// JournalEntry is one durable record of a bus delivery. Never carries plaintext.
type JournalEntry struct {
	ID        string `json:"id"`
	TS        string `json:"ts"` // RFC3339
	From      string `json:"from"`
	To        string `json:"to"`
	Workspace string `json:"workspace,omitempty"`
	Provider  string `json:"provider,omitempty"`
	Channel   string `json:"channel,omitempty"` // inbox | inject | bridge
	Action    string `json:"action"`            // submitted | staged | refused | queued
	Reason    string `json:"reason,omitempty"`
	TextSHA   string `json:"text_sha,omitempty"`
	TextLen   int    `json:"text_len"`
}

// AppendJournal appends one record. Auditing must never block delivery, but here
// we surface the error so callers can decide; the bus wiring ignores it.
func (s *Store) AppendJournal(e JournalEntry) error {
	if e.TS == "" {
		e.TS = time.Now().UTC().Format(time.RFC3339)
	}
	line, err := json.Marshal(e)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(s.journalPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	return err
}

// JournalFilter narrows a query. Zero values mean "no constraint".
type JournalFilter struct {
	From  string
	To    string
	Since time.Time
	Limit int // 0 = all; otherwise the most recent N matches
}

// QueryJournal returns matching entries, oldest→newest (most-recent-N if Limit>0).
func (s *Store) QueryJournal(f JournalFilter) ([]JournalEntry, error) {
	file, err := os.Open(s.journalPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var out []JournalEntry
	sc := bufio.NewScanner(file)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		var e JournalEntry
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			continue // skip a corrupt line rather than fail the whole query
		}
		if f.From != "" && e.From != f.From {
			continue
		}
		if f.To != "" && e.To != f.To {
			continue
		}
		if !f.Since.IsZero() {
			if ts, perr := time.Parse(time.RFC3339, e.TS); perr == nil && ts.Before(f.Since) {
				continue
			}
		}
		out = append(out, e)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if f.Limit > 0 && len(out) > f.Limit {
		out = out[len(out)-f.Limit:]
	}
	return out, nil
}

// --- Registry (session metadata over time) ---

// SessionRecord is the persisted knowledge about one agent session.
type SessionRecord struct {
	SessionID string `json:"session_id"`
	Provider  string `json:"provider,omitempty"`
	Workspace string `json:"workspace,omitempty"`
	Ref       string `json:"ref,omitempty"`
	State     string `json:"state,omitempty"`
	FirstSeen string `json:"first_seen"`
	LastSeen  string `json:"last_seen"`
}

func (s *Store) loadRegistry() (map[string]SessionRecord, error) {
	m := map[string]SessionRecord{}
	data, err := os.ReadFile(s.registryPath())
	if err != nil {
		if os.IsNotExist(err) {
			return m, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *Store) saveRegistry(m map[string]SessionRecord) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.registryPath(), data, 0o644)
}

// UpsertSession records or refreshes a session. FirstSeen is preserved across
// updates; LastSeen and mutable fields (state, workspace, ref) are refreshed.
func (s *Store) UpsertSession(r SessionRecord) error {
	if r.SessionID == "" {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if r.LastSeen == "" {
		r.LastSeen = now
	}
	m, err := s.loadRegistry()
	if err != nil {
		return err
	}
	if prev, ok := m[r.SessionID]; ok {
		r.FirstSeen = prev.FirstSeen // immutable
		if r.Provider == "" {
			r.Provider = prev.Provider
		}
	} else if r.FirstSeen == "" {
		r.FirstSeen = r.LastSeen
	}
	m[r.SessionID] = r
	return s.saveRegistry(m)
}

// GetSession returns a record by id.
func (s *Store) GetSession(id string) (SessionRecord, bool, error) {
	m, err := s.loadRegistry()
	if err != nil {
		return SessionRecord{}, false, err
	}
	r, ok := m[id]
	return r, ok, nil
}

// ListSessions returns all known sessions, sorted by SessionID for stable output.
func (s *Store) ListSessions() ([]SessionRecord, error) {
	m, err := s.loadRegistry()
	if err != nil {
		return nil, err
	}
	out := make([]SessionRecord, 0, len(m))
	for _, r := range m {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SessionID < out[j].SessionID })
	return out, nil
}
