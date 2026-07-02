package main

// net.go — couche 5 (réseau) : fédération minimale machine-à-machine.
//
// `communikey serve` ouvre un listener TCP qui reçoit des messages (JSON ligne par
// ligne) et les dépose dans l'inbox local ; `communikey remote` envoie vers une autre
// machine. Le payload PEUT être chiffré E2E (SealedMessage), donc même un relais
// ne voit que du chiffré. v1 = loopback/LAN de confiance ; TLS hybride PQC + auth
// mutuelle = phase suivante (§38 : ne pas exposer hors loopback sans durcir).

import (
	"bufio"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
)

const defaultServeAddr = "127.0.0.1:9777"

// serveBus listens and delivers each received frame to the local inbox. Blocking.
// A non-nil cfg makes it a TLS 1.3 listener (hybrid PQC in Go 1.24).
func serveBus(s *Store, addr string, cfg *tls.Config, allow map[string]bool) error {
	var ln net.Listener
	var err error
	if cfg != nil {
		ln, err = tls.Listen("tcp", addr, cfg)
	} else {
		ln, err = net.Listen("tcp", addr)
	}
	if err != nil {
		return err
	}
	defer ln.Close()
	fmt.Printf("communikey serve — écoute sur %s (Ctrl+C pour arrêter)\n", ln.Addr())
	if !isLoopbackAddr(addr) && allow == nil {
		fmt.Println("⚠ hors loopback sans --authz : n'expose que sur un réseau de confiance (§38).")
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go handleBusConn(s, conn, allow)
	}
}

// handleBusConn reads one JSON message frame and delivers it. Pure of the listener
// so it is unit-testable over any net.Conn (real loopback in tests).
func handleBusConn(s *Store, conn net.Conn, allow map[string]bool) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil && len(line) == 0 {
		return
	}
	var m InboxMessage
	if json.Unmarshal(line, &m) != nil || m.To == "" {
		fmt.Fprintln(conn, `{"ok":false,"error":"frame invalide"}`)
		return
	}
	if allow != nil && !senderAllowed(m, allow) {
		fmt.Fprintln(conn, `{"ok":false,"error":"expediteur non autorise (E2E signe requis)"}`)
		return
	}
	// Anti-replay : un message scellé est dédupliqué sur son NONCE signé (inaltérable
	// sans casser la signature) ; sinon sur l'id. Un doublon est acquitté, pas re-livré.
	dedup := m.ID
	if m.Sealed != nil && len(m.Sealed.Nonce) > 0 {
		dedup = "n:" + hex.EncodeToString(m.Sealed.Nonce)
	}
	if !s.markSeen(dedup) {
		fmt.Fprintln(conn, `{"ok":true,"duplicate":true}`)
		return
	}
	if m.ID == "" {
		m.ID = newID()
	}
	if m.TS == "" {
		m.TS = nowRFC3339()
	}
	if err := s.Inbox().Deliver(m); err != nil {
		fmt.Fprintln(conn, `{"ok":false,"error":"livraison"}`)
		return
	}
	_ = s.AppendJournal(JournalEntry{
		ID: m.ID, TS: m.TS, From: m.From, To: m.To, Channel: "network",
		Action: "submitted", TextSHA: sha6(m.Body), TextLen: len(m.Body),
	})
	fmt.Fprintln(conn, `{"ok":true}`)
}

// sendRemote delivers one message to a remote communikey serve. A non-nil cfg dials over
// TLS 1.3 (hybrid PQC) with optional fingerprint pinning.
func sendRemote(addr string, m InboxMessage, cfg *tls.Config) error {
	var conn net.Conn
	var err error
	if cfg != nil {
		conn, err = tls.DialWithDialer(&net.Dialer{Timeout: 5 * time.Second}, "tcp", addr, cfg)
	} else {
		conn, err = net.DialTimeout("tcp", addr, 5*time.Second)
	}
	if err != nil {
		return fmt.Errorf("connexion à %s impossible: %w", addr, err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	data, _ := json.Marshal(m)
	if _, err := conn.Write(append(data, '\n')); err != nil {
		return err
	}
	resp, _ := bufio.NewReader(conn).ReadString('\n')
	if !strings.Contains(resp, `"ok":true`) {
		return fmt.Errorf("rejet du serveur distant: %s", strings.TrimSpace(resp))
	}
	return nil
}

func isLoopbackAddr(addr string) bool {
	host := addr
	if h, _, err := net.SplitHostPort(addr); err == nil {
		host = h
	}
	if host == "" || host == "localhost" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

// --- CLI ---

func cmdServe(args []string) {
	addr := defaultServeAddr
	useTLS, authz := false, false
	var allowFlags []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--addr":
			if i+1 < len(args) {
				addr = args[i+1]
				i++
			}
		case "--tls":
			useTLS = true
		case "--authz":
			authz = true
		case "--allow":
			if i+1 < len(args) {
				allowFlags = append(allowFlags, args[i+1])
				i++
			}
		}
	}
	s := mustStore()
	var cfg *tls.Config
	if useTLS {
		c, fp, err := serverTLSConfig(s)
		if err != nil {
			fail(err.Error())
		}
		cfg = c
		fmt.Printf("TLS 1.3 hybride PQC actif. Fingerprint à épingler côté client :\n  %s\n", fp)
	}
	var allow map[string]bool
	if authz || len(allowFlags) > 0 {
		a, configured := loadAllowlist(s, allowFlags)
		if !configured {
			fail("--authz sans allowlist : ajoute des fingerprints (allowed.json ou --allow <fp>)")
		}
		allow = a
		fmt.Printf("Autorisation active : %d expéditeur(s) autorisé(s) — E2E signé requis.\n", len(allow))
	}
	if err := serveBus(s, addr, cfg, allow); err != nil {
		fail(err.Error())
	}
}

func cmdRemote(args []string) {
	useTLS := false
	pin := ""
	var pos []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--tls":
			useTLS = true
		case "--pin":
			if i+1 < len(args) {
				pin = args[i+1]
				i++
			}
		default:
			pos = append(pos, args[i])
		}
	}
	if len(pos) < 3 {
		fail("usage: communikey remote [--tls --pin <fingerprint>] <hôte:port> <agent> <message…>")
	}
	addr, to := pos[0], pos[1]
	body := strings.Join(pos[2:], " ")
	m := InboxMessage{ID: newID(), TS: nowRFC3339(), From: selfAgentID(), To: to}
	enc := ""
	if sealed, ok := maybeSeal(mustStore(), to, body); ok {
		m.Sealed = sealed
		enc = " · chiffré E2E"
	} else {
		m.Body = body
	}
	var cfg *tls.Config
	tlsNote := ""
	if useTLS {
		cfg = tlsClientConfig(pin)
		tlsNote = " · TLS PQC"
	}
	s := mustStore()
	if n := flushOutbox(s, addr, cfg); n > 0 {
		fmt.Printf("  (%d message(s) en file ré-envoyé(s) à %s)\n", n, addr)
	}
	if err := sendRemote(addr, m, cfg); err != nil {
		if enqueueOutbox(s, addr, m) == nil {
			fmt.Printf("⚠ %s injoignable — message mis en file (%d en attente). Relance plus tard: communikey remote …\n", addr, pendingOutbox(s, addr))
			return
		}
		fail(err.Error())
	}
	fmt.Printf("✓ envoyé à %s via %s [network%s]%s\n", to, addr, tlsNote, enc)
}
