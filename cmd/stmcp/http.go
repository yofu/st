package main

import (
	"bufio"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type httpConfig struct {
	Root      string `json:"root"`
	Address   string `json:"address"`
	TokenFile string `json:"token_file"`
}

type mcpHTTPSession struct {
	mu     sync.Mutex
	input  *io.PipeWriter
	output *bufio.Reader
	pipe   *io.PipeReader
	last   time.Time
}

func (s *mcpHTTPSession) close() { s.input.Close(); s.pipe.Close() }

type mcpHTTP struct {
	root, token string
	mu          sync.Mutex
	sessions    map[string]*mcpHTTPSession
}

func newMCPHTTP(root, token string) *mcpHTTP {
	return &mcpHTTP{root: root, token: token, sessions: make(map[string]*mcpHTTPSession)}
}

func (h *mcpHTTP) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for id, s := range h.sessions {
		s.close()
		delete(h.sessions, id)
	}
}

func (h *mcpHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil || (host != "127.0.0.1" && host != "localhost" && host != "::1") {
		http.Error(w, "invalid host", http.StatusForbidden)
		return
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || u.Host != r.Host || u.Scheme != "http" {
			http.Error(w, "invalid origin", http.StatusForbidden)
			return
		}
	}
	credential := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if h.token == "" || subtle.ConstantTimeCompare([]byte(credential), []byte(h.token)) != 1 {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if r.URL.Path != "/mcp" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	id := r.Header.Get("Mcp-Session-Id")
	if r.Method == http.MethodDelete {
		h.mu.Lock()
		s := h.sessions[id]
		delete(h.sessions, id)
		h.mu.Unlock()
		if s == nil {
			http.Error(w, "unknown session", http.StatusNotFound)
			return
		}
		s.close()
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST, DELETE")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	var request struct {
		Version string          `json:"jsonrpc"`
		Method  string          `json:"method"`
		ID      json.RawMessage `json:"id"`
	}
	if json.Unmarshal(data, &request) != nil || request.Method == "" || request.Version != "2.0" || (len(request.ID) == 0 && !strings.HasPrefix(request.Method, "notifications/")) {
		http.Error(w, "expected one JSON-RPC request", http.StatusBadRequest)
		return
	}
	h.mu.Lock()
	// Sessions are local to this server and retain the same model state as STDIO.
	for key, s := range h.sessions {
		if time.Since(s.last) > time.Hour {
			s.close()
			delete(h.sessions, key)
		}
	}
	s := h.sessions[id]
	if request.Method == "initialize" && id == "" {
		if len(request.ID) == 0 || len(h.sessions) >= 32 {
			h.mu.Unlock()
			http.Error(w, "cannot initialize session", http.StatusBadRequest)
			return
		}
		var nonce [16]byte
		if _, err := rand.Read(nonce[:]); err != nil {
			h.mu.Unlock()
			http.Error(w, "session creation failed", http.StatusInternalServerError)
			return
		}
		id = hex.EncodeToString(nonce[:])
		inReader, inWriter := io.Pipe()
		outReader, outWriter := io.Pipe()
		s = &mcpHTTPSession{input: inWriter, output: bufio.NewReader(outReader), pipe: outReader, last: time.Now()}
		h.sessions[id] = s
		go func() {
			err := run(h.root, inReader, outWriter)
			inReader.CloseWithError(err)
			outWriter.CloseWithError(err)
		}()
	}
	if s == nil {
		h.mu.Unlock()
		http.Error(w, "unknown MCP session", http.StatusNotFound)
		return
	}
	s.last = time.Now()
	h.mu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.input.Write(append(data, '\n')); err != nil {
		http.Error(w, "MCP session closed", http.StatusInternalServerError)
		return
	}
	if len(request.ID) == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	response, err := s.output.ReadBytes('\n')
	if err != nil {
		http.Error(w, "MCP session closed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Mcp-Session-Id", id)
	_, _ = w.Write(response)
}

func serveMCPHTTP(root, address, tokenFile string) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil || (host != "127.0.0.1" && host != "::1") {
		return fmt.Errorf("HTTP address must use a loopback IP and port")
	}
	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return fmt.Errorf("HTTP token: %w", err)
	}
	token := strings.TrimSpace(string(data))
	if len(token) < 32 {
		return fmt.Errorf("HTTP token must contain at least 32 characters")
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	handler := newMCPHTTP(root, token)
	defer handler.Close()
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 90 * time.Second}
	fmt.Fprintf(os.Stderr, "stmcp: listening on http://%s/mcp (root %s)\n", listener.Addr(), root)
	return server.Serve(listener)
}
