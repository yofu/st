package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yofu/dxf"
	st "github.com/yofu/st/stlib"
)

func TestDXFDraftingMCPPersistAndExport(t *testing.T) {
	dir, s := drawingMCPFixture(t)
	q := settingsCall(t, s, "query_dxf_settings", map[string]any{})
	cfg := q["settings"].(*st.DXFDrawingSettings)
	cfg.Style = &st.DXFDraftingStyle{TextHeightMM: 3, Font: "msgothic.ttc"}
	cfg.Sheet = &st.DXFSheetTemplate{Project: "MCP 製図確認", Office: "出力確認", Date: "2026-11-21", DrawnBy: "検証", StartNumber: 21}
	p := settingsCall(t, s, "preview_dxf_settings", map[string]any{"settings": cfg})
	if p["draft_id"] == nil {
		t.Fatal(p)
	}
	settingsCall(t, s, "save_dxf_settings", map[string]any{"draft_id": p["draft_id"]})
	if _, err := s.open("model.inp"); err != nil {
		t.Fatal(err)
	}
	preview := settingsCall(t, s, "preview_dxf_drawings", map[string]any{"options": map[string]any{"sections": true}})
	r := preview["report"].(*st.DXFDocumentReport)
	if r.Sheets[0].Number != "S-21" || r.SheetTemplate.Project != "MCP 製図確認" || r.Drafting.TextHeightMM != 3 {
		t.Fatal("persisted drafting settings not applied", r)
	}
	settingsCall(t, s, "save_dxf_drawings", map[string]any{"draft_id": preview["draft_id"], "path": "drafting.dxf"})
	d, err := dxf.FromFile(filepath.Join(dir, "drafting.dxf"))
	if err != nil {
		t.Fatal(err)
	}
	if issues := dxf.Audit(d, false); len(issues) != 0 {
		t.Fatal(issues)
	}
	if _, err = os.Stat(filepath.Join(dir, "model.drawing.json")); err != nil {
		t.Fatal(err)
	}
}
