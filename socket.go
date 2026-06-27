package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

// socket.go — direct JSON-RPC client for the cmux Unix socket.
//
// Why bypass the `cmux` CLI for surface I/O: the CLI defaults `workspace_id` to
// the CALLER's workspace ($CMUX_WORKSPACE_ID), so any read/send aimed at a
// surface in another workspace fails ("Surface is not a terminal"). Speaking the
// socket directly and passing the surface's OWN workspace_id reads and writes any
// surface in any workspace — visible or background — instantly, no flicker, no
// focus change. Protocol (traced 2026-06-27):
//
//   >> {"id":"<hex>","method":"surface.read_text","params":{...}}\n
//   << {"result":{...,"base64":"…"},"id":"<hex>","ok":true}\n

func socketPath() string {
	if p := os.Getenv("CMUX_SOCKET_PATH"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Application Support", "cmux", "cmux.sock")
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// rpcCall sends one request and returns its `result` object. An `ok:false` reply
// (or a transport error) becomes a Go error.
func rpcCall(method string, params map[string]any) (map[string]any, error) {
	conn, err := net.DialTimeout("unix", socketPath(), 3*time.Second)
	if err != nil {
		return nil, fmt.Errorf("socket cmux injoignable: %w", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(6 * time.Second))

	req, _ := json.Marshal(map[string]any{"id": newID(), "method": method, "params": params})
	if _, err := conn.Write(append(req, '\n')); err != nil {
		return nil, err
	}
	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil && len(line) == 0 {
		return nil, err
	}
	var resp struct {
		OK     bool           `json:"ok"`
		Result map[string]any `json:"result"`
		Error  any            `json:"error"`
	}
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, fmt.Errorf("réponse RPC illisible: %w", err)
	}
	if !resp.OK {
		return nil, fmt.Errorf("RPC %s a échoué: %v", method, resp.Error)
	}
	return resp.Result, nil
}
