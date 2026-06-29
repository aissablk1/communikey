package main

// lock.go — verrou consultatif inter-process, portable (aucune dépendance, marche
// sur tout OS via un lockfile O_EXCL). Sérialise les séquences read-modify-write
// (registre, relations) pour éviter les « lost updates » quand plusieurs sessions
// écrivent en même temps — réel dès qu'on a des armées de sessions.

import (
	"os"
	"time"
)

const (
	lockWait  = 5 * time.Second        // attente max avant de continuer best-effort
	lockStale = 30 * time.Second       // un lock plus vieux que ça est volé (process mort)
	lockTick  = 12 * time.Millisecond  // backoff entre tentatives
)

// withLock runs fn while holding an exclusive lock derived from target. If the lock
// can't be acquired within lockWait (high contention), it proceeds anyway
// (best-effort) rather than deadlocking — correctness is favored under normal
// contention, liveness under pathological contention.
func withLock(target string, fn func() error) error {
	lockPath := target + ".lock"
	deadline := time.Now().Add(lockWait)
	for {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			f.Close()
			defer os.Remove(lockPath)
			return fn()
		}
		if fi, statErr := os.Stat(lockPath); statErr == nil && time.Since(fi.ModTime()) > lockStale {
			_ = os.Remove(lockPath) // steal a stale lock (holder likely died)
			continue
		}
		if time.Now().After(deadline) {
			return fn() // best-effort: don't deadlock
		}
		time.Sleep(lockTick)
	}
}
