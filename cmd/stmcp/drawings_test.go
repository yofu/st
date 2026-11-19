package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/yofu/dxf"
	st "github.com/yofu/st/stlib"
)

func drawingMCPFixture(t *testing.T) (string, *modelSession) {
	t.Helper()
	dir := t.TempDir()
	inp := []byte("PROP 1 PNAME FC24\n MATERIAL FC24 HFACT 1 EFACT 1\nSECT 101 SNAME C1\n NFIG 1\n FIG 1 FPROP 1 AREA 0.09 IXX 0.001 IYY 0.001 VEN 0.002\nNODE 1 CORD 0 0 0\nNODE 2 CORD 0 0 3\nELEM 11 ESECT 101 ENODS 2 ENOD 1 2 TYPE COLUMN\n")
	lst := []byte("CODE 101 RC COLUMN C1\n CRECT -15 -15 15 15 FC24\n REINS 2.865 -10 -10 SD345 D19\n REINS 2.865 10 -10 SD345 D19\n REINS 2.865 10 10 SD345 D19\n REINS 2.865 -10 10 SD345 D19\n HOOPS 0.005 0.005 SD295 2-D10@100\n")
	for name, data := range map[string][]byte{"model.inp": inp, "model.lst": lst} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { root.Close() })
	s := &modelSession{root: root, parsePath: filepath.Join(t.TempDir(), "model.inp")}
	if _, err = s.open("model.inp"); err != nil {
		t.Fatal(err)
	}
	return dir, s
}

func TestDXFDrawingMCPReviewSaveAndUnsavedState(t *testing.T) {
	dir, s := drawingMCPFixture(t)
	before, _ := os.ReadFile(filepath.Join(dir, "model.inp"))
	// Simulate a gui_get_model import whose current LST differs from disk.
	s.frame.Sects[101].Allow.(*st.RCColumn).SetName("UNSAVED_C1")
	value, err := s.call("query_dxf_sections", []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	records := value.(map[string]any)["sections"].([]st.DXFSection)
	if records[0].Symbol != "UNSAVED_C1" || records[0].Shape != "300×300 mm" || len(records[0].Bars) != 4 {
		t.Fatal(records)
	}
	preview, err := s.call("preview_dxf_drawings", []byte(`{"options":{"sections":true,"paper":"A4P","section_columns":1,"section_rows":1}}`))
	if err != nil {
		t.Fatal(err)
	}
	id := preview.(map[string]any)["draft_id"].(uint64)
	if _, err = os.Stat(filepath.Join(dir, "sheets.dxf")); !os.IsNotExist(err) {
		t.Fatal("preview wrote output")
	}
	for _, path := range []string{"../outside.dxf", "model.inp", "C:/outside.dxf"} {
		raw, _ := json.Marshal(map[string]any{"draft_id": id, "path": path})
		if _, err = s.call("save_dxf_drawings", raw); err == nil {
			t.Fatal("invalid output accepted", path)
		}
	}
	raw := []byte(fmt.Sprintf(`{"draft_id":%d,"path":"sheets.dxf"}`, id))
	if _, err = s.call("save_dxf_drawings", raw); err != nil {
		t.Fatal(err)
	}
	if _, err = s.call("save_dxf_drawings", raw); err == nil {
		t.Fatal("overwrote existing file")
	}
	d, err := dxf.FromFile(filepath.Join(dir, "sheets.dxf"))
	if err != nil || len(d.Layouts()) != 2 {
		t.Fatal("invalid saved sheets", err)
	}
	if issues := dxf.Audit(d, false); len(issues) > 0 {
		t.Fatal(issues)
	}
	s.frame.Nodes[2].Coord[2] = 3.1
	if _, err = s.call("save_dxf_drawings", []byte(fmt.Sprintf(`{"draft_id":%d,"path":"stale.dxf"}`, id))); err == nil {
		t.Fatal("saved stale model draft")
	}
	after, _ := os.ReadFile(filepath.Join(dir, "model.inp"))
	if !bytes.Equal(before, after) {
		t.Fatal("drawing changed INP")
	}
	if _, err = s.call("preview_dxf_drawings", []byte(`{"options":{"sections":true,"unexpected":1}}`)); err == nil {
		t.Fatal("unknown options accepted")
	}
	if _, err = s.call("query_dxf_sections", []byte(`{"kjn_path":"../secret.kjn"}`)); err == nil {
		t.Fatal("unscoped KJN accepted")
	}
}

func TestDXFDrawingMCPStdioCatalogAndOutput(t *testing.T) {
	dir, _ := drawingMCPFixture(t)
	var in, out bytes.Buffer
	requests := []string{
		`{"id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`,
		`{"id":2,"method":"tools/list"}`,
		`{"id":3,"method":"tools/call","params":{"name":"open_model","arguments":{"path":"model.inp"}}}`,
		`{"id":4,"method":"tools/call","params":{"name":"query_dxf_sections","arguments":{}}}`,
		`{"id":5,"method":"tools/call","params":{"name":"preview_dxf_drawings","arguments":{"options":{"sections":true,"paper":"A3"}}}}`,
		`{"id":6,"method":"tools/call","params":{"name":"save_dxf_drawings","arguments":{"draft_id":1,"path":"stdio.dxf"}}}`,
	}
	for _, r := range requests {
		in.WriteString(`{"jsonrpc":"2.0",` + r[1:] + "\n")
	}
	if err := run(dir, &in, &out); err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(&out)
	for id := 1; id <= len(requests); id++ {
		var r struct {
			ID     int
			Error  json.RawMessage
			Result struct {
				IsError bool
				Tools   []tool
			}
		}
		if err := decoder.Decode(&r); err != nil {
			t.Fatal(err)
		}
		if r.ID != id || r.Error != nil || r.Result.IsError {
			t.Fatalf("request %d failed: %+v", id, r)
		}
		if id == 2 {
			seen := map[string]bool{}
			for _, tool := range r.Result.Tools {
				seen[tool.Name] = true
			}
			if !seen["preview_dxf_drawings"] || !seen["save_dxf_drawings"] || !seen["query_dxf_sections"] {
				t.Fatal("drawing tools not advertised")
			}
		}
	}
	if _, err := dxf.FromFile(filepath.Join(dir, "stdio.dxf")); err != nil {
		t.Fatal(err)
	}
}

func TestDXFDrawingMCPReviewedLSTProposal(t *testing.T) {
	_, s := drawingMCPFixture(t)
	definition := "CODE 101 RC COLUMN NEW_C1\n CRECT -20 -20 20 20 FC24\n HOOPS .005 .005 SD295 2-D10@100\n"
	raw, _ := json.Marshal(map[string]any{"edits": []map[string]any{{"section": 101, "definition": definition}}})
	if _, err := s.call("preview_lst_edits", raw); err != nil {
		t.Fatal(err)
	}
	id := s.pending.ID
	value, err := s.call("query_dxf_sections", []byte(fmt.Sprintf(`{"proposal_id":%d}`, id)))
	if err != nil {
		t.Fatal(err)
	}
	sections := value.(map[string]any)["sections"].([]st.DXFSection)
	if sections[0].Shape != "400×400 mm" {
		t.Fatal("drawing did not use reviewed LST", sections[0])
	}
	if _, err = s.call("preview_dxf_drawings", []byte(fmt.Sprintf(`{"proposal_id":%d,"options":{"sections":true}}`, id))); err != nil {
		t.Fatal(err)
	}
	s.pending = nil
	if _, err = s.call("save_dxf_drawings", []byte(`{"draft_id":1,"path":"stale-proposal.dxf"}`)); err == nil {
		t.Fatal("saved stale proposal")
	}
}

func TestDXFDrawingMCPNakano(t *testing.T) {
	input := os.Getenv("ST_DXF_TEST_INP")
	if input == "" {
		t.Skip("set ST_DXF_TEST_INP for real-model MCP export")
	}
	dir := filepath.Dir(input)
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	s := modelSession{root: root, parsePath: filepath.Join(t.TempDir(), "model.inp")}
	if _, err = s.open(filepath.Base(input)); err != nil {
		t.Fatal(err)
	}
	q, err := s.call("query_dxf_sections", []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	count := len(q.(map[string]any)["sections"].([]st.DXFSection))
	preview, err := s.call("preview_dxf_drawings", []byte(`{"options":{"title":"MCP nakano02_07","views":[{"kind":"plan","floor":3},{"kind":"elevation","grid":"Y2"}],"sections":true}}`))
	if err != nil {
		t.Fatal(err)
	}
	report := preview.(map[string]any)["report"].(*st.DXFDocumentReport)
	if len(report.Sections) != count || report.Pages < 3 {
		t.Fatal("incomplete MCP report")
	}
	// Save via the real tool into a separate root; never write to source folder.
	output := os.Getenv("ST_DXF_TEST_OUTPUT")
	if output == "" {
		output = t.TempDir()
	}
	if err = os.MkdirAll(output, 0755); err != nil {
		t.Fatal(err)
	}
	exportRoot, err := os.OpenRoot(output)
	if err != nil {
		t.Fatal(err)
	}
	defer exportRoot.Close()
	s.root = exportRoot
	name := "mcp_nakano_sheets.dxf"
	// Test output is replaceable; the save tool itself always uses O_EXCL.
	_ = exportRoot.Remove(name)
	if _, err = s.call("save_dxf_drawings", []byte(`{"draft_id":1,"path":"mcp_nakano_sheets.dxf"}`)); err != nil {
		t.Fatal(err)
	}
	d, err := dxf.FromFile(filepath.Join(output, name))
	if err != nil {
		t.Fatal(err)
	}
	if issues := dxf.Audit(d, false); len(issues) > 0 {
		t.Fatal(issues)
	}
	t.Logf("MCP: %d sections, %d sheets, output %s", count, report.Pages, filepath.Join(output, name))
}
