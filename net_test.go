package main

import (
	"net"
	"testing"
)

// Real loopback TCP: a message sent via sendRemote must land in the server's inbox.
func TestNetworkLoopbackDeliver(t *testing.T) {
	s, _ := OpenStore(t.TempDir())
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, e := ln.Accept()
			if e != nil {
				return
			}
			handleBusConn(s, c, nil)
		}
	}()

	addr := ln.Addr().String()
	if err := sendRemote(addr, InboxMessage{From: "A", To: "BOB", Body: "hello over tcp"}, nil); err != nil {
		t.Fatal(err)
	}
	msgs, _ := s.Inbox().Receive("BOB", true)
	if len(msgs) != 1 || msgs[0].Body != "hello over tcp" || msgs[0].From != "A" {
		t.Fatalf("message non livré via réseau: %+v", msgs)
	}
}

func TestNetworkRejectsBadFrame(t *testing.T) {
	s, _ := OpenStore(t.TempDir())
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	go func() {
		for {
			c, e := ln.Accept()
			if e != nil {
				return
			}
			handleBusConn(s, c, nil)
		}
	}()
	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.Write([]byte("ceci n'est pas du json\n"))
	buf := make([]byte, 64)
	n, _ := conn.Read(buf)
	if n == 0 || string(buf[:2]) != `{"` {
		t.Fatalf("réponse inattendue à un frame invalide: %q", string(buf[:n]))
	}
}

func TestIsLoopbackAddr(t *testing.T) {
	cases := map[string]bool{
		"127.0.0.1:9777": true, "localhost:80": true, "[::1]:9777": true,
		"0.0.0.0:9777": false, "192.168.1.10:9777": false,
	}
	for addr, want := range cases {
		if got := isLoopbackAddr(addr); got != want {
			t.Fatalf("isLoopbackAddr(%q)=%v want %v", addr, got, want)
		}
	}
}
