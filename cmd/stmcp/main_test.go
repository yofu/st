package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestStdioModelRoundTrip(t *testing.T) {
	root := t.TempDir()
	src := []byte("NODE 10 CORD 0.123456789 0 0 ICON 1 1 1 0 0 0\r\n# retain this comment\r\nNODE 11 CORD 1 0 0\r\n")
	if err := os.WriteFile(filepath.Join(root, "base.inp"), src, 0600); err != nil {
		t.Fatal(err)
	}
	var in, out bytes.Buffer
	for _, line := range []string{
		`{"id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`,
		`{"method":"notifications/initialized"}`,
		`{"id":2,"method":"tools/list"}`,
		`{"id":3,"method":"tools/call","params":{"name":"open_model","arguments":{"path":"base.inp"},"_meta":{"progressToken":3}}}`,
		`{"id":4,"method":"tools/call","params":{"name":"preview_edits","arguments":{"edits":[{"node":10,"fixed":[true,true,true,false,false,true]},{"node":11,"load":[0,0,0.000000000031,0,0,0]}]}}}`,
		`{"id":5,"method":"tools/call","params":{"name":"save_model_as","arguments":{"path":"base.inp","proposal_id":1}}}`,
		`{"id":6,"method":"tools/call","params":{"name":"save_model_as","arguments":{"path":"../outside.inp","proposal_id":1}}}`,
		`{"id":7,"method":"tools/call","params":{"name":"save_model_as","arguments":{"path":"saved.inp","proposal_id":2}}}`,
		`{"id":8,"method":"tools/call","params":{"name":"save_model_as","arguments":{"path":"saved.inp","proposal_id":1}}}`,
		`{"id":9,"method":"tools/call","params":{"name":"save_model_as","arguments":{"path":"saved.inp","proposal_id":1}}}`,
		`{"id":10,"method":"tools/call","params":{"name":"open_model","arguments":{"path":"saved.inp"}}}`,
		`{"id":11,"method":"tools/call","params":{"name":"query_nodes","arguments":{"ids":[11]}}}`,
		`{"id":12,"method":"tools/call","params":{"name":"list_model_editors","_meta":{"progressToken":"list"}}}`,
		`{"id":13,"method":"tools/call","params":{"name":"query_nodes","arguments":{"ids":[null]},"_meta":{"progressToken":13}}}`,
		`{"id":14,"method":"tools/call","params":{"name":"list_model_editors","arguments":{"unexpected":true},"_meta":{"progressToken":14}}}`,
	} {
		fmt.Fprintf(&in, "{\"jsonrpc\":\"2.0\",%s\n", line[1:])
	}
	if err := run(root, &in, &out); err != nil {
		t.Fatal(err)
	}
	d := json.NewDecoder(&out)
	for id := 1; id <= 14; id++ {
		var resp struct {
			ID     int
			Error  json.RawMessage
			Result struct {
				IsError           bool
				StructuredContent json.RawMessage
			}
		}
		if err := d.Decode(&resp); err != nil {
			t.Fatal(err)
		}
		wantError := id == 5 || id == 6 || id == 7 || id == 9 || id == 13 || id == 14
		if resp.ID != id || resp.Error != nil || resp.Result.IsError != wantError {
			t.Fatalf("request %d: %+v", id, resp)
		}
		if id == 11 {
			var result struct{ Items []nodeInfo }
			if err := json.Unmarshal(resp.Result.StructuredContent, &result); err != nil {
				t.Fatal(err)
			}
			if len(result.Items) != 1 || result.Items[0].Load[2] != 3.1e-11 {
				t.Fatal(result)
			}
		}
	}
	base, err := os.ReadFile(filepath.Join(root, "base.inp"))
	if err != nil || !bytes.Equal(src, base) {
		t.Fatalf("source changed: %v", err)
	}
	saved, err := os.ReadFile(filepath.Join(root, "saved.inp"))
	want := []byte("NODE 10 CORD 0.123456789 0 0 ICON 1 1 1 0 0 1\r\n# retain this comment\r\nNODE 11 VCON 0 0 3.1e-11 0 0 0 CORD 1 0 0\r\n")
	if err != nil || !bytes.Equal(saved, want) {
		t.Fatalf("saved %q: %v", saved, err)
	}
}
