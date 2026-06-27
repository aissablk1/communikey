package main

// net.go — couche 5 (réseau) : fédération minimale machine-à-machine.
//
// `csend serve` ouvre un listener TCP qui reçoit des messages (JSON ligne par
// ligne) et les dépose dans l'inbox local ; `csend remote` envoie vers une autre
// machine. Le payload PEUT être chiffré E2E (SealedMessage), donc même un relais
// ne voit que du chiffré. v1 = loopback/LAN de confiance ; TLS hybride PQC + auth
// mutuelle = phase suivante (§38 : ne pas exposer hors loopback sans durcir).

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
)

const defaultServeAddr = "127.0.0.1:9777"

// serveBus listens and delivers each received frame to the local inbox. Blocking.
func serveBus(s *Store, addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer ln.Close()
	fmt.Printf("csend serve — écoute sur %s (Ctrl+C pour arrêter)\n", ln.Addr())
	if !isLoopbackAddr(addr) {
		fmt.Println("⚠ hors loopback : sans TLS + auth, n'expose que sur un réseau de confiance (§38).")
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go handleBusConn(s, conn)
	}
}

// handleBusConn reads one JSON message frame and delivers it. Pure of the listener
// so it is unit-testable over any net.Conn (real loopback in tests).
func handleBusConn(s *Store, conn net.Conn) {
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

// sendRemote delivers one message to a remote csend serve.
func sendRemote(addr string, m InboxMessage) error {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
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
	for i := 0; i < len(args); i++ {
		if args[i] == "--addr" && i+1 < len(args) {
			addr = args[i+1]
			i++
		}
	}
	if err := serveBus(mustStore(), addr); err != nil {
		fail(err.Error())
	}
}

func cmdRemote(args []string) {
	if len(args) < 3 {
		fail("usage: csend remote <hôte:port> <agent> <message…>")
	}
	addr, to := args[0], args[1]
	body := strings.Join(args[2:], " ")
	m := InboxMessage{ID: newID(), TS: nowRFC3339(), From: selfAgentID(), To: to, Body: body}
	if err := sendRemote(addr, m); err != nil {
		fail(err.Error())
	}
	fmt.Printf("✓ envoyé à %s via %s [network]\n", to, addr)
}
