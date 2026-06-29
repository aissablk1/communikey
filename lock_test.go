package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestWithLockSerializes(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "counter")
	_ = os.WriteFile(target, []byte("0"), 0o600)

	var wg sync.WaitGroup
	const N = 25
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = withLock(target, func() error {
				b, _ := os.ReadFile(target)
				n, _ := strconv.Atoi(strings.TrimSpace(string(b)))
				time.Sleep(time.Millisecond) // élargit la fenêtre de course
				return os.WriteFile(target, []byte(strconv.Itoa(n+1)), 0o600)
			})
		}()
	}
	wg.Wait()

	b, _ := os.ReadFile(target)
	n, _ := strconv.Atoi(strings.TrimSpace(string(b)))
	if n != N {
		t.Fatalf("lost updates: got %d, want %d — le verrou ne sérialise pas", n, N)
	}
}

func TestUpsertSessionConcurrent(t *testing.T) {
	s, _ := OpenStore(t.TempDir())
	var wg sync.WaitGroup
	const N = 25
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = s.UpsertSession(SessionRecord{SessionID: "agent-" + strconv.Itoa(i), FirstSeen: "t", LastSeen: "t"})
		}(i)
	}
	wg.Wait()
	recs, _ := s.ListSessions()
	if len(recs) != N {
		t.Fatalf("got %d sessions, want %d — updates concurrents perdus", len(recs), N)
	}
}
