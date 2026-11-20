package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	st "github.com/yofu/st/stlib"
)

func settingsCall(t *testing.T, s *modelSession, name string, args any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	v, err := s.call(name, raw)
	if err != nil {
		t.Fatal(err)
	}
	return v.(map[string]any)
}
func TestDXFDrawingSettingsMCPReviewSaveReload(t *testing.T) {
	dir, s := drawingMCPFixture(t)
	before, _ := os.ReadFile(filepath.Join(dir, "model.inp"))
	q := settingsCall(t, s, "query_dxf_settings", map[string]any{})
	cfg := q["settings"].(*st.DXFDrawingSettings)
	if q["exists"].(bool) {
		t.Fatal("invented settings file")
	}
	symbol := "DRAWING_ONLY"
	cfg.Members = map[int]st.DXFDrawingMember{11: {Symbol: &symbol}}
	p := settingsCall(t, s, "preview_dxf_settings", map[string]any{"settings": cfg})
	if p["draft_id"] == nil {
		t.Fatalf("%+v", p["report"].(*st.DXFDrawingCheckReport).Issues)
	}
	bound := p["after"].(*st.DXFDrawingSettings).Members[11].Expected
	if bound == nil || bound.Section != 101 || len(bound.Nodes) != 2 {
		t.Fatal("member identity was not bound")
	}
	settingsCall(t, s, "save_dxf_settings", map[string]any{"draft_id": p["draft_id"]})
	q = settingsCall(t, s, "query_dxf_settings", map[string]any{})
	cfg = q["settings"].(*st.DXFDrawingSettings)
	if *cfg.Members[11].Symbol != "DRAWING_ONLY" {
		t.Fatal("lost manual adjustment")
	}
	original, _ := os.ReadFile(filepath.Join(dir, "model.drawing.json"))
	next := "REVISED"
	m := cfg.Members[11]
	m.Symbol = &next
	cfg.Members[11] = m
	p = settingsCall(t, s, "preview_dxf_settings", map[string]any{"settings": cfg})
	saved := settingsCall(t, s, "save_dxf_settings", map[string]any{"draft_id": p["draft_id"]})
	backup, err := os.ReadFile(filepath.Join(dir, saved["backup"].(string)))
	if err != nil || !bytes.Equal(backup, original) {
		t.Fatal("missing original backup", err)
	}
	// DXF preview loads the stored profile through the scoped root.
	d := settingsCall(t, s, "preview_dxf_drawings", map[string]any{"options": map[string]any{"sections": true}})
	if d["draft_id"] == nil {
		t.Fatal(d)
	}
	settingsCall(t, s, "save_dxf_drawings", map[string]any{"draft_id": d["draft_id"], "path": "configured.dxf"})
	after, _ := os.ReadFile(filepath.Join(dir, "model.inp"))
	if !bytes.Equal(before, after) || s.frame.Sects[101].Allow.Name() != "\"C1\"" && s.frame.Sects[101].Allow.Name() != "C1" {
		t.Fatal("settings edited source model")
	}
}
func TestDXFDrawingSettingsMCPRejectStaleAndEscape(t *testing.T) {
	dir, s := drawingMCPFixture(t)
	q := settingsCall(t, s, "query_dxf_settings", map[string]any{})
	cfg := q["settings"].(*st.DXFDrawingSettings)
	for _, path := range []string{"../outside.drawing.json", "C:/outside.drawing.json", "model.inp"} {
		raw, _ := json.Marshal(map[string]any{"settings": cfg, "settings_path": path})
		if _, err := s.call("preview_dxf_settings", raw); err == nil {
			t.Fatal("accepted", path)
		}
	}
	p := settingsCall(t, s, "preview_dxf_settings", map[string]any{"settings": cfg})
	s.frame.Nodes[2].Coord[2] += .1
	raw, _ := json.Marshal(map[string]any{"draft_id": p["draft_id"]})
	if _, err := s.call("save_dxf_settings", raw); err == nil {
		t.Fatal("saved stale model settings")
	}
	s.frame.Nodes[2].Coord[2] -= .1
	p = settingsCall(t, s, "preview_dxf_settings", map[string]any{"settings": cfg})
	os.WriteFile(filepath.Join(dir, "model.drawing.json"), []byte("external edit"), 0600)
	raw, _ = json.Marshal(map[string]any{"draft_id": p["draft_id"]})
	if _, err := s.call("save_dxf_settings", raw); err == nil {
		t.Fatal("overwrote newly created settings")
	}
}
func TestDXFDrawingSettingsMCPDrawingProfileChanged(t *testing.T) {
	dir, s := drawingMCPFixture(t)
	q := settingsCall(t, s, "query_dxf_settings", map[string]any{})
	cfg := q["settings"].(*st.DXFDrawingSettings)
	p := settingsCall(t, s, "preview_dxf_settings", map[string]any{"settings": cfg})
	settingsCall(t, s, "save_dxf_settings", map[string]any{"draft_id": p["draft_id"]})
	d := settingsCall(t, s, "preview_dxf_drawings", map[string]any{"options": map[string]any{"sections": true}})
	os.WriteFile(filepath.Join(dir, "model.drawing.json"), []byte(`{"version":1}`), 0600)
	raw := []byte(fmt.Sprintf(`{"draft_id":%d,"path":"stale.dxf"}`, d["draft_id"]))
	if _, err := s.call("save_dxf_drawings", raw); err == nil {
		t.Fatal("DXF saved after settings changed")
	}
}

func TestDXFDrawingSettingsMCPDrawingGridChanged(t *testing.T) {
	dir, s := drawingMCPFixture(t)
	file := filepath.Join(dir, "model.kjn")
	if err := os.WriteFile(file, []byte("Y1 0 0 0 4 0 0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	d := settingsCall(t, s, "preview_dxf_drawings", map[string]any{"options": map[string]any{"sections": true}})
	if err := os.WriteFile(file, []byte("Y1 0 1 0 4 1 0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	raw := []byte(fmt.Sprintf(`{"draft_id":%d,"path":"stale-grid.dxf"}`, d["draft_id"]))
	if _, err := s.call("save_dxf_drawings", raw); err == nil {
		t.Fatal("saved drawing after KJN changed")
	}
}

func TestDXFDrawingSettingsMCPNakano(t *testing.T) {
	input := os.Getenv("ST_DXF_TEST_INP")
	if input == "" {
		t.Skip("set ST_DXF_TEST_INP")
	}
	dir := t.TempDir()
	base := filepath.Base(input)
	originals := map[string][]byte{}
	for _, ext := range []string{".inp", ".lst", ".kjn"} {
		data, err := os.ReadFile(st.Ce(input, ext))
		if err != nil {
			t.Fatal(err)
		}
		name := st.Ce(base, ext)
		originals[name] = data
		if err = os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	s := &modelSession{root: root, rootPath: dir, parsePath: filepath.Join(t.TempDir(), "model.inp")}
	if _, err = s.open(base); err != nil {
		t.Fatal(err)
	}
	q := settingsCall(t, s, "query_dxf_settings", map[string]any{})
	cfg := q["settings"].(*st.DXFDrawingSettings)
	symbol := "G51_DRAWING"
	cfg.Sections = map[int]st.DXFDrawingMember{501: {Symbol: &symbol}}
	p := settingsCall(t, s, "preview_dxf_settings", map[string]any{"settings": cfg})
	if p["draft_id"] == nil {
		t.Fatalf("%+v", p["report"])
	}
	settingsCall(t, s, "save_dxf_settings", map[string]any{"draft_id": p["draft_id"]})
	// Reopen the INP: settings persist independently of the prior model session.
	if _, err = s.open(base); err != nil {
		t.Fatal(err)
	}
	q = settingsCall(t, s, "query_dxf_settings", map[string]any{})
	loaded := q["settings"].(*st.DXFDrawingSettings)
	if *loaded.Sections[501].Symbol != symbol {
		t.Fatal("lost settings after reopening")
	}
	d := settingsCall(t, s, "preview_dxf_drawings", map[string]any{"options": map[string]any{"views": []map[string]any{{"kind": "plan", "floor": 3}, {"kind": "elevation", "grid": "Y2"}}, "sections": true}})
	report := d["report"].(*st.DXFDocumentReport)
	if len(report.DrawingIssues) == 0 {
		t.Fatal("consistency issues missing")
	}
	settingsCall(t, s, "save_dxf_drawings", map[string]any{"draft_id": d["draft_id"], "path": "configured.dxf"})
	for name, before := range originals {
		after, _ := os.ReadFile(filepath.Join(dir, name))
		if !bytes.Equal(before, after) {
			t.Fatal("edited original model data", name)
		}
	}
	if output := os.Getenv("ST_DXF_SETTINGS_TEST_OUTPUT"); output != "" {
		if err = os.MkdirAll(output, 0755); err != nil {
			t.Fatal(err)
		}
		data, _ := os.ReadFile(filepath.Join(dir, "configured.dxf"))
		if err = os.WriteFile(filepath.Join(output, "mcp_nakano_settings.dxf"), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("MCP reopened model: %d sections, %d pages, %d structured issues", len(report.Sections), report.Pages, len(report.DrawingIssues))
}
