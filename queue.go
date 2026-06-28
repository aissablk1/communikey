package main

// queue.go — file d'attente offline. Si `remote` ne peut pas joindre la machine
// cible, le message est mis en file localement (outbox) au lieu d'être perdu ;
// `remote` rejoue la file de cette adresse à chaque appel (self-healing).

import (
	"crypto/tls"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func outboxDir(s *Store, addr string) string {
	return filepath.Join(s.Dir, "outbox", safeID(addr))
}

// enqueueOutbox stores a message for later (re)delivery to addr. Atomic.
func enqueueOutbox(s *Store, addr string, m InboxMessage) error {
	dir := outboxDir(s, addr)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(m, "", "  ")
	final := filepath.Join(dir, safeID(m.ID)+".json")
	tmp := final + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, final)
}

// flushOutbox re-sends every queued message for addr and removes those that get
// through. Best-effort: a still-unreachable addr leaves the queue intact. Returns
// the number delivered.
func flushOutbox(s *Store, addr string, cfg *tls.Config) int {
	entries, err := os.ReadDir(outboxDir(s, addr))
	if err != nil {
		return 0
	}
	sent := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		p := filepath.Join(outboxDir(s, addr), e.Name())
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var m InboxMessage
		if json.Unmarshal(data, &m) != nil {
			continue
		}
		if sendRemote(addr, m, cfg) == nil {
			_ = os.Remove(p)
			sent++
		}
	}
	return sent
}

// pendingOutbox counts queued messages for addr.
func pendingOutbox(s *Store, addr string) int {
	entries, err := os.ReadDir(outboxDir(s, addr))
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			n++
		}
	}
	return n
}
