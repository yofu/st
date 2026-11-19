package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHTTPRetainsModelSessionAndAuthenticates(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "model.inp"), []byte("NODE 10 CORD 0 0 0\nNODE 11 CORD 1 0 0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	token := "test-local-MCP-token-01234567890123456789"
	handler := newMCPHTTP(dir, token)
	server := httptest.NewServer(handler)
	defer server.Close()
	defer handler.Close()
	session := ""
	call := func(method string, id int, params any) (*http.Response, []byte) {
		t.Helper()
		message := map[string]any{"jsonrpc": "2.0", "method": method, "params": params}
		if id != 0 {
			message["id"] = id
		}
		data, _ := json.Marshal(message)
		req, _ := http.NewRequest("POST", server.URL+"/mcp", bytes.NewReader(data))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		if session != "" {
			req.Header.Set("Mcp-Session-Id", session)
		}
		r, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		return r, body
	}
	response, body := call("initialize", 1, map[string]string{"protocolVersion": "2025-11-25"})
	if response.StatusCode != 200 {
		t.Fatalf("initialize: %d %s", response.StatusCode, body)
	}
	session = response.Header.Get("Mcp-Session-Id")
	if session == "" {
		t.Fatal("missing session ID")
	}
	response, _ = call("notifications/initialized", 0, map[string]any{})
	if response.StatusCode != 202 {
		t.Fatal("notification not accepted")
	}
	response, body = call("tools/call", 2, map[string]any{"name": "open_model", "arguments": map[string]string{"path": "model.inp"}})
	if response.StatusCode != 200 || bytes.Contains(body, []byte(`"isError":true`)) {
		t.Fatalf("open: %d %s", response.StatusCode, body)
	}
	_, body = call("tools/call", 3, map[string]any{"name": "query_nodes", "arguments": map[string]any{"ids": []int{11}}})
	if !bytes.Contains(body, []byte(`"coordinates":[1,0,0]`)) {
		t.Fatalf("session lost model: %s", body)
	}
	req, _ := http.NewRequest("POST", server.URL+"/mcp", bytes.NewBufferString(`{}`))
	r, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != 401 {
		t.Fatal("unauthenticated HTTP request accepted")
	}
	req, _ = http.NewRequest("POST", server.URL+"/mcp", bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Origin", "https://untrusted.example")
	r, err = server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != 403 {
		t.Fatal("cross-origin request accepted")
	}
	req, _ = http.NewRequest("DELETE", server.URL+"/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Mcp-Session-Id", session)
	r, err = server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != 204 {
		t.Fatal("failed to close HTTP session")
	}
	response, _ = call("ping", 4, map[string]any{})
	if response.StatusCode != 404 {
		t.Fatal("closed session was reused")
	}
}

func TestUnavailableRootKeepsMCPTransportAlive(t *testing.T) {
	var input, output bytes.Buffer
	for _, request := range []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25"}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"open_model","arguments":{"path":"model.inp"}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"ping"}`,
	} {
		input.WriteString(request + "\n")
	}
	if err := run(filepath.Join(t.TempDir(), "missing"), &input, &output); err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(&output)
	for id := 1; id <= 4; id++ {
		var reply struct {
			ID     int
			Result json.RawMessage
		}
		if err := decoder.Decode(&reply); err != nil {
			t.Fatal(err)
		}
		if reply.ID != id {
			t.Fatal("MCP transport stopped after filesystem failure")
		}
		if id == 3 && !bytes.Contains(reply.Result, []byte(`"isError":true`)) {
			t.Fatal("unavailable file operation did not report its error")
		}
	}
}
