package stgui

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	st "github.com/yofu/st/stlib"
)

func startTestHTTP(ctx context.Context, exe, root string) (string, string, func(), error) {
	const token = "integration-test-local-loopback-credential"
	tokenFile := filepath.Join(root, "test-token")
	if err := os.WriteFile(tokenFile, []byte(token), 0600); err != nil {
		return "", "", nil, err
	}
	cmd := exec.CommandContext(ctx, exe, "-root", root, "-http", "127.0.0.1:0", "-token-file", tokenFile)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", "", nil, err
	}
	if err := cmd.Start(); err != nil {
		return "", "", nil, err
	}
	closeTestServer := func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }
	log := bufio.NewReader(stderr)
	line, err := log.ReadString('\n')
	if err != nil {
		closeTestServer()
		return "", "", nil, err
	}
	fields := strings.Fields(line)
	if len(fields) < 4 || !strings.HasPrefix(fields[3], "http://127.0.0.1:") {
		closeTestServer()
		return "", "", nil, fmt.Errorf("HTTP startup: %s", line)
	}
	go io.Copy(io.Discard, log)
	return fields[3], token, closeTestServer, nil
}

func exerciseWorkflowHTTP(ctx context.Context, exe, root, guiID string, construct bool) error {
	var url, token string
	installed := construct && os.Getenv("ST_MCP_TEST_INSTALLED_CONFIG") != ""
	if installed {
		path := os.Getenv("ST_MCP_TEST_INSTALLED_CONFIG")
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var cfg mcpHTTPLaunchConfig
		if err = json.Unmarshal(data, &cfg); err != nil {
			return err
		}
		host, _, err := net.SplitHostPort(cfg.Address)
		if err != nil || host != "127.0.0.1" && host != "::1" {
			return fmt.Errorf("expected loopback MCP server")
		}
		tokenFile := cfg.TokenFile
		if !filepath.IsAbs(tokenFile) {
			tokenFile = filepath.Join(filepath.Dir(path), tokenFile)
		}
		data, err = os.ReadFile(tokenFile)
		if err != nil {
			return err
		}
		url, token = "http://"+cfg.Address+"/mcp", strings.TrimSpace(string(data))
	} else {
		var stop func()
		var err error
		url, token, stop, err = startTestHTTP(ctx, exe, root)
		if err != nil {
			return err
		}
		defer stop()
	}
	var session string
	call := func(method string, params any) (json.RawMessage, error) {
		data, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Mcp-Session-Id", session)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("HTTP %s: %s", method, resp.Status)
		}
		session = resp.Header.Get("Mcp-Session-Id")
		var rpc struct {
			Result json.RawMessage
			Error  json.RawMessage
		}
		if err := json.NewDecoder(resp.Body).Decode(&rpc); err != nil {
			return nil, err
		}
		if len(rpc.Error) > 0 || session == "" {
			return nil, fmt.Errorf("HTTP %s: %s", method, rpc.Error)
		}
		return rpc.Result, nil
	}
	initSession := func() error {
		session = ""
		_, err := call("initialize", map[string]string{"protocolVersion": "2025-11-25"})
		return err
	}
	if err := initSession(); err != nil {
		return err
	}
	if construct {
		defer func() {
			req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
			if err == nil {
				req.Header.Set("Authorization", "Bearer "+token)
				req.Header.Set("Mcp-Session-Id", session)
				if r, e := http.DefaultClient.Do(req); e == nil {
					r.Body.Close()
				}
			}
		}()
		return exerciseConstructionHTTP(root, guiID, !installed, call)
	}
	run := func(args map[string]any) (string, error) {
		value, err := call("tools/call", map[string]any{"name": "run_model_workflow", "arguments": args})
		if err != nil {
			return "", err
		}
		var result struct {
			IsError    bool
			Structured struct{ State, Error string } `json:"structuredContent"`
		}
		if err := json.Unmarshal(value, &result); err != nil {
			return "", err
		}
		if result.IsError {
			return "tool-error", nil
		}
		if result.Structured.Error != "" {
			return "", fmt.Errorf("HTTP workflow: %s", result.Structured.Error)
		}
		return result.Structured.State, nil
	}
	for _, a := range []map[string]any{{"connection_id": guiID, "commands": []string{"'dfact 76"}}, {"commands": []string{"'dfact 75"}}} {
		state, err := run(a)
		if err != nil || state != "completed" {
			return fmt.Errorf("paired HTTP workflow: %s (%v)", state, err)
		}
	}
	if err := initSession(); err != nil {
		return err
	}
	state, err := run(map[string]any{"commands": []string{"'dfact 999"}})
	if err != nil || state != "tool-error" {
		return fmt.Errorf("new HTTP session inherited another client's window: %s (%v)", state, err)
	}
	fmt.Fprintln(os.Stderr, "HTTP workflow: paired window retained across requests and isolated across sessions")
	return nil
}

func exerciseConstructionHTTP(root, id string, save bool, call func(string, any) (json.RawMessage, error)) error {
	raw, err := call("tools/list", map[string]any{})
	if err != nil {
		return err
	}
	var catalog struct {
		Tools []struct {
			Name        string
			InputSchema struct{ Properties map[string]json.RawMessage }
		}
	}
	if err = json.Unmarshal(raw, &catalog); err != nil {
		return err
	}
	found := slices.ContainsFunc(catalog.Tools, func(t struct {
		Name        string
		InputSchema struct{ Properties map[string]json.RawMessage }
	}) bool {
		return t.Name == "run_model_workflow" && t.InputSchema.Properties["construct"] != nil
	})
	if !found {
		return fmt.Errorf("HTTP tool catalog missing construction")
	}
	invoke := func(name string, args any, dst any) error {
		raw, err := call("tools/call", map[string]any{"name": name, "arguments": args})
		if err != nil {
			return err
		}
		var result struct {
			IsError    bool
			Structured json.RawMessage `json:"structuredContent"`
		}
		if err := json.Unmarshal(raw, &result); err != nil {
			return err
		}
		if result.IsError {
			return fmt.Errorf("%s: %s", name, raw)
		}
		return json.Unmarshal(result.Structured, dst)
	}
	q := st.GUIConstruction{
		Properties: []st.BuildProperty{{ID: 101, Material: "SN400"}},
		Sections:   []st.BuildSection{{ID: 501, Property: 101, Name: "H-200x200x8x12", Shape: "HKYOU", Dimensions: []float64{200, 200, 8, 12}}},
		Grid:       &st.BuildGrid{Cells: []int{10, 10, 0}, Pitch: []float64{1, 1, 0}, Type: "GIRDER", Section: 501},
	}
	var result struct {
		State, Error string
		Construction *st.ConstructionReceipt
		GUI          *st.GUIResponse
	}
	start := time.Now()
	// Start on the untitled frame. The same call confirms the model, sets its
	// view, and saves to the private test directory through native IUP.
	args := map[string]any{"connection_id": id, "construct": q, "commands": []string{"'angle 90 -90", "'fit"}, "wait_ms": 5000}
	if save {
		args["save_copy"] = "grid.inp"
	}
	if err := invoke("run_model_workflow", args, &result); err != nil {
		return err
	}
	elapsed := time.Since(start)
	if result.State != "completed" || result.Construction == nil || result.GUI == nil {
		return fmt.Errorf("construction workflow: %+v", result)
	}
	r := result.Construction
	figs := r.Sections[501]
	if r.Added.Nodes != 121 || r.Added.Elements != 220 || r.MemberTypes["GIRDER"] != 220 || r.Directions["X"] != 110 || r.Directions["Y"] != 110 || r.MinLength != 1 || r.MaxLength != 1 || len(figs) != 1 || figs[0].Property != 101 || figs[0].Material != "SN400" || !slices.Equal(figs[0].Dimensions, []float64{200, 200, 8, 12}) {
		return fmt.Errorf("bad GUI receipt: %+v", r)
	}
	if save {
		saved := filepath.Join(root, "grid.inp")
		data, err := os.ReadFile(saved)
		if err != nil {
			return err
		}
		f := st.NewFrame()
		if err = f.ReadInpSnapshot(data, saved); err != nil {
			return err
		}
		if len(f.Nodes) != 121 || len(f.Elems) != 220 || f.GUIRevision() != result.GUI.Revision {
			return fmt.Errorf("saved/GUI model mismatch: %s / %s", f.GUIRevision(), result.GUI.Revision)
		}
	}
	var view st.GUIResponse
	if err := invoke("gui_undo", map[string]any{}, &view); err != nil {
		return err
	}
	var listed struct{ Items []any }
	if err := invoke("query_nodes", map[string]any{"limit": 1}, &listed); err != nil || len(listed.Items) != 0 {
		return fmt.Errorf("undo left grid nodes: %v (%v)", listed, err)
	}
	if err := invoke("gui_redo", map[string]any{}, &view); err != nil || view.Revision != result.GUI.Revision {
		return fmt.Errorf("redo did not restore grid: %v", err)
	}
	fmt.Fprintf(os.Stderr, "HTTP one-call grid create, fit, verified receipt (save_copy=%t): %s; undo/redo passed\n", save, elapsed)
	gridRevision := result.GUI.Revision
	surfaces := map[string]any{
		"plate_sections": []map[string]any{{"id": 711, "dead_load_kgf_m2": 100, "lload_tf_m2": []float64{.065, .045, .035}}},
		"panel_grid":     map[string]any{"plane": "XY", "coordinate_m": 0, "section": 711, "type": "SLAB", "expected_cells": 100},
	}
	args = map[string]any{"construct": surfaces, "revision": gridRevision, "wait_ms": 5000}
	if save {
		args["save_copy"] = "slabs.inp"
	}
	start = time.Now()
	if err := invoke("run_model_workflow", args, &result); err != nil {
		return err
	}
	if result.State != "completed" || result.Construction == nil || result.GUI == nil {
		return fmt.Errorf("slab workflow: %s %s", result.State, result.Error)
	}
	elapsed = time.Since(start)
	r = result.Construction
	load := r.PlateSections[711]
	if r.Total.Nodes != 121 || r.Total.Elements != 320 || r.Added.Elements != 100 || r.MemberTypes["SLAB"] != 100 || r.PlateArea[711] != 100 || load.Dead != 100 || !slices.Equal(load.Live, []float64{.065, .045, .035}) {
		return fmt.Errorf("bad slab receipt: %+v", r)
	}
	if save {
		path := filepath.Join(root, "slabs.inp")
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		f := st.NewFrame()
		if err := f.ReadInpSnapshot(data, path); err != nil {
			return err
		}
		var weight [3]float64
		count := 0
		for _, e := range f.Elems {
			if e.Etype != st.SLAB {
				continue
			}
			if e.Sect != f.Sects[711] || e.Enods != 4 || e.Area() != 1 || e.IsBraced() {
				return fmt.Errorf("incorrect saved slab %d", e.Num)
			}
			count++
			for i, w := range e.Weight() {
				weight[i] += w
			}
		}
		if count != 100 || f.GUIRevision() != result.GUI.Revision {
			return fmt.Errorf("saved slab count or revision mismatch")
		}
		for i, w := range []float64{16.5, 14.5, 13.5} {
			if math.Abs(weight[i]-w) > 1e-10 {
				return fmt.Errorf("slab load case %d: got %g, want %g tf", i, weight[i], w)
			}
		}
	}
	// The load edit is staged before the duplicate faces are found. Neither
	// the changed weight nor an extra Undo checkpoint should reach the GUI.
	surfaces["plate_sections"] = []map[string]any{{"id": 711, "dead_load_kgf_m2": 200}}
	var rejected struct{ State, Error string }
	if err := invoke("run_model_workflow", map[string]any{"construct": surfaces, "wait_ms": 5000}, &rejected); err != nil || rejected.State != "failed" || !strings.Contains(rejected.Error, "occupied") {
		return fmt.Errorf("duplicate plate batch: %+v (%v)", rejected, err)
	}
	if err := invoke("gui_status", map[string]any{}, &view); err != nil || view.Revision != result.GUI.Revision {
		return fmt.Errorf("rejected batch changed live model: %v", err)
	}
	if err := invoke("gui_undo", map[string]any{}, &view); err != nil || view.Revision != gridRevision {
		return fmt.Errorf("single undo did not restore the bare grid: %v", err)
	}
	fmt.Fprintf(os.Stderr, "HTTP one-call 100 SLAB, 100 m2, SECT 711 and fixed/live loads verified (save_copy=%t): %s\n", save, elapsed)
	return nil
}
