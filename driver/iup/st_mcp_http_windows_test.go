package stgui

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func writeHTTPLaunchTestConfig(t *testing.T, address string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "token"), []byte(strings.Repeat("a", 64)), 0600); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(mcpHTTPLaunchConfig{Root: dir, Address: address, TokenFile: "token"})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "stmcp-http.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestMCPHTTPConcurrentStartupAndReuse(t *testing.T) {
	executable := os.Getenv("ST_MCP_EXE")
	if executable == "" {
		t.Skip("set ST_MCP_EXE to exercise the real HTTP server")
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	config := writeHTTPLaunchTestConfig(t, address)
	var starts atomic.Int32
	var child *exec.Cmd
	var done chan error
	var stopped chan struct{}
	start := func(_ string, path string) (<-chan error, error) {
		starts.Add(1)
		child = exec.Command(executable, "-http-config", path)
		child.Dir = filepath.Dir(path)
		configureMCPHTTPProcess(child)
		if err := child.Start(); err != nil {
			return nil, err
		}
		done = make(chan error, 1)
		stopped = make(chan struct{})
		go func() { done <- child.Wait(); close(stopped) }()
		return done, nil
	}
	defer func() {
		if child != nil && child.Process != nil {
			child.Process.Kill()
			select {
			case <-stopped:
			case <-time.After(5 * time.Second):
				t.Error("test server did not stop")
			}
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	var newlyStarted atomic.Int32
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			started, err := ensureMCPHTTPServer(ctx, config, start)
			if err != nil {
				t.Error(err)
				return
			}
			if started {
				newlyStarted.Add(1)
			}
		}()
	}
	wg.Wait()
	if starts.Load() != 1 || newlyStarted.Load() != 1 {
		t.Fatalf("starts=%d newlyStarted=%d; want exactly one", starts.Load(), newlyStarted.Load())
	}
	started, err := ensureMCPHTTPServer(ctx, config, start)
	if err != nil || started || starts.Load() != 1 {
		t.Fatalf("reuse: started=%v count=%d error=%v", started, starts.Load(), err)
	}
}

func TestMCPHTTPOccupiedPortDoesNotStart(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusOK} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				w.Write([]byte(`{"result":{"serverInfo":{"name":"another-server"}}}`))
			}))
			defer server.Close()
			config := writeHTTPLaunchTestConfig(t, strings.TrimPrefix(server.URL, "http://"))
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			started, err := ensureMCPHTTPServer(ctx, config, func(_, _ string) (<-chan error, error) {
				t.Error("must not launch when another server occupies the port")
				return nil, nil
			})
			if started || err == nil {
				t.Fatalf("started=%v error=%v", started, err)
			}
		})
	}
}
