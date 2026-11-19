package stgui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type mcpHTTPLaunchConfig struct {
	Root      string `json:"root"`
	Address   string `json:"address"`
	TokenFile string `json:"token_file"`
}

type mcpHTTPStarter func(executable, config string) (<-chan error, error)

// Optional configuration keeps existing STDIO-only installations working.
func (stw *Window) ensureMCPHTTP() error {
	configPath := os.Getenv("ST_MCP_HTTP_CONFIG")
	explicit := configPath != ""
	if !explicit {
		executable, err := os.Executable()
		if err != nil {
			return err
		}
		configPath = filepath.Join(filepath.Dir(executable), "stmcp-http.json")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return nil
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	started, err := ensureMCPHTTPServer(ctx, configPath, startMCPHTTPProcess)
	if err != nil {
		return fmt.Errorf("MCP HTTP: %w", err)
	}
	if started {
		stw.History("MCP HTTP server started")
	} else {
		stw.History("MCP HTTP server already running")
	}
	return nil
}

func ensureMCPHTTPServer(ctx context.Context, configPath string, start mcpHTTPStarter) (bool, error) {
	configPath, err := filepath.Abs(configPath)
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false, err
	}
	var config mcpHTTPLaunchConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return false, err
	}
	host, _, err := net.SplitHostPort(config.Address)
	if err != nil || (host != "127.0.0.1" && host != "::1") {
		return false, fmt.Errorf("server address must be a loopback IP and port")
	}
	tokenPath := config.TokenFile
	if !filepath.IsAbs(tokenPath) {
		tokenPath = filepath.Join(filepath.Dir(configPath), tokenPath)
	}
	tokenData, err := os.ReadFile(tokenPath)
	if err != nil {
		return false, err
	}
	token := strings.TrimSpace(string(tokenData))
	if len(token) < 32 {
		return false, fmt.Errorf("HTTP token must contain at least 32 characters")
	}
	started := false
	err = withMCPHTTPStartupLock(ctx, config.Address, func() error {
		running, err := probeMCPHTTP(ctx, config.Address, token)
		if err != nil {
			return err
		}
		if running {
			return nil
		}
		executable := filepath.Join(filepath.Dir(configPath), "stmcp-http.exe")
		exited, err := start(executable, configPath)
		if err != nil {
			return err
		}
		started = true
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			running, err := probeMCPHTTP(ctx, config.Address, token)
			if err != nil {
				return err
			}
			if running {
				return nil
			}
			select {
			case err := <-exited:
				return fmt.Errorf("server exited before becoming ready: %v", err)
			case <-ctx.Done():
				return fmt.Errorf("server did not become ready: %w", ctx.Err())
			case <-ticker.C:
			}
		}
	})
	return started, err
}

func startMCPHTTPProcess(executable, config string) (<-chan error, error) {
	cmd := exec.Command(executable, "-http-config", config)
	cmd.Dir = filepath.Dir(executable)
	configureMCPHTTPProcess(cmd)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	return exited, nil
}

// Authenticate and check the MCP server identity instead of relying on a PID
// or accepting any process which happens to occupy the configured port.
func probeMCPHTTP(ctx context.Context, address, token string) (bool, error) {
	dialer := net.Dialer{Timeout: 200 * time.Millisecond}
	connection, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		return false, nil
	}
	connection.Close()
	requestBody := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"st-auto-start","version":"1"}}}`)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+address+"/mcp", bytes.NewReader(requestBody))
	if err != nil {
		return false, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{Timeout: time.Second, Transport: &http.Transport{Proxy: nil, DisableKeepAlives: true}}
	response, err := client.Do(request)
	if err != nil {
		return false, fmt.Errorf("configured port is occupied but MCP did not respond: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return false, fmt.Errorf("configured HTTP server returned %d; check its authentication and configuration", response.StatusCode)
	}
	var reply struct {
		Result struct {
			ServerInfo struct {
				Name string `json:"name"`
			} `json:"serverInfo"`
		} `json:"result"`
		Error json.RawMessage `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&reply); err != nil {
		return false, fmt.Errorf("invalid MCP response: %w", err)
	}
	if len(reply.Error) > 0 || reply.Result.ServerInfo.Name != "stmcp" {
		return false, fmt.Errorf("configured port is not an ST MCP server")
	}
	if session := response.Header.Get("Mcp-Session-Id"); session != "" {
		closeRequest, _ := http.NewRequestWithContext(ctx, http.MethodDelete, "http://"+address+"/mcp", nil)
		closeRequest.Header.Set("Authorization", "Bearer "+token)
		closeRequest.Header.Set("Mcp-Session-Id", session)
		if closed, err := client.Do(closeRequest); err == nil {
			closed.Body.Close()
		}
	}
	return true, nil
}
