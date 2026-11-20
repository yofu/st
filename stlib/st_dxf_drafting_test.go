package st

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yofu/dxf"
	"github.com/yofu/dxf/entity"
)

func TestDXFDraftingPaperUnitsAndDimensions(t *testing.T) {
	f := dxfExportFixture(t)
	dec := 1
	f.drawingSettings = &DXFDrawingSettings{Version: 1, Style: &DXFDraftingStyle{TextHeightMM: 3, GridTextHeightMM: 2.8, NoteTextHeightMM: 2.7, Dimension: DXFDraftingDimension{TextHeightMM: 2.6, ArrowSizeMM: 1.8, Decimals: &dec}}}
	for _, scale := range []float64{50, 100} {
		path := filepath.Join(t.TempDir(), "view.dxf")
		if err := f.WriteDxfCrosssectionDimensions(path, 2, 1.5, 4.5, 1, 1000, 2.5*scale, 3*scale); err != nil {
			t.Fatal(err)
		}
		d := readExportedDXF(t, path)
		if d.Header().LtScale != 1 {
			t.Fatal("global line scale")
		}
		for _, tc := range []struct {
			layer    string
			weight   int
			lineType string
		}{{"HASHIRA", 35, "Continuous"}, {"HARI", 25, "Continuous"}, {"GRID", 13, "ST_CENTER"}, {"DIM_SPAN", 13, "Continuous"}} {
			layer := d.Layers[tc.layer]
			if layer == nil || layer.LineWeight() != tc.weight || layer.LineType.Name() != tc.lineType || int(layer.Color) != 7 {
				t.Fatal("wrong layer profile", tc)
			}
		}
		for _, e := range d.EntitiesOnLayer("GRID") {
			p := e.(entity.LineProperties)
			if _, line := e.(*entity.Line); line {
				if p.LineTypeName() != "ByLayer" {
					t.Fatal("grid pattern not inherited")
				}
			} else if p.LineTypeName() != "Continuous" {
				t.Fatal("grid bubbles became dashed")
			}
		}
		for _, e := range d.EntitiesOnLayer("TXTGIRDER") {
			if tx, ok := e.(*entity.Text); ok {
				jointNear(t, tx.Height/scale, 3)
				jointNear(t, tx.WidthFactor, .9)
			}
		}
		for _, e := range d.EntitiesOnLayer("GRID_TEXT") {
			if tx, ok := e.(*entity.Text); ok {
				jointNear(t, tx.Height/scale, 2.8)
			}
		}
		ds, err := d.DimStyle("Standard")
		if err != nil {
			t.Fatal(err)
		}
		jointNear(t, ds.TextHeight/scale, 2.6)
		jointNear(t, ds.ArrowSize/scale, 1.8)
		if ds.Decimals != 1 {
			t.Fatal("precision was lost")
		}
		for _, e := range d.Entities() {
			if dim, ok := e.(*entity.Dimension); ok {
				block, err := d.Block(dim.BlockName)
				if err != nil {
					t.Fatal(err)
				}
				for _, g := range block.Entities {
					if tx, ok := g.(*entity.Text); ok {
						jointNear(t, tx.Height/scale, 2.6)
						if tx.Style.FontName != "msgothic.ttc" {
							t.Fatal("dimension font")
						}
					}
				}
			}
		}
	}
}

func TestDXFDraftingStyleValidationAndColors(t *testing.T) {
	for _, bad := range []*DXFDraftingStyle{
		{TextHeightMM: math.NaN()}, {TextWidthFactor: .1}, {Dimension: DXFDraftingDimension{Decimals: dxfPtr(7)}},
		{Layers: map[string]DXFDraftingLayer{"typo": {}}},
		{Layers: map[string]DXFDraftingLayer{"beam": {LineType: "typo"}}},
		{Layers: map[string]DXFDraftingLayer{"beam": {LineWeightMM: dxfPtr(-1.)}}},
	} {
		if _, err := ResolveDXFDraftingStyle(bad); err == nil {
			t.Fatal("accepted invalid profile", bad)
		}
	}
	style, err := ResolveDXFDraftingStyle(&DXFDraftingStyle{Monochrome: dxfPtr(false), Layers: map[string]DXFDraftingLayer{"beam": {Color: dxfPtr(1), LineWeightMM: dxfPtr(.26)}}})
	if err != nil {
		t.Fatal(err)
	}
	d := dxf.NewDrawing()
	d.AddLayer("HARI", dxf.DefaultColor, dxf.DefaultLineType, true)
	d.Line(0, 0, 0, 100, 0, 0)
	applyDXFDraftingEntities(d, style, 50)
	if int(d.Layers["HARI"].Color) != 1 || d.Layers["HARI"].LineWeight() != 30 {
		t.Fatal("color/nearest DXF weight")
	}
}

func TestDXFSheetMetadataLayoutAndRoundtrip(t *testing.T) {
	f := documentFixture(t)
	cfg := f.DefaultDXFDrawingSettings()
	cfg.Sheet = &DXFSheetTemplate{Office: "構造設計事務所", Project: "確認用構造図", Structure: "鉄骨造", Date: "2026-11-21", DrawnBy: "作図担当", CheckedBy: "確認担当", StartNumber: 7, Revisions: []DXFSheetRevision{{Mark: "A", Date: "2026-11-21", Description: "図枠検証"}}, Notes: []string{"寸法単位：mm"}, Pages: map[int]DXFSheetPage{1: {Title: "伏図・軸組図"}}}
	d, r, err := f.BuildDXFDocument(DXFDocumentOptions{Columns: 2, Views: []DXFViewSpec{{Kind: "plan", Floor: 2}, {Kind: "elevation", Grid: "Y1"}}, Settings: cfg})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Sheets) != 1 || r.Sheets[0].Number != "S-07" || r.Sheets[0].Title != "伏図・軸組図" {
		t.Fatal(r.Sheets)
	}
	if r.Sheets[0].ContentBox[2] >= r.Sheets[0].TitleBox[0] {
		t.Fatal("overlapping content/title boxes")
	}
	for _, p := range r.Placements {
		if p.PaperBox[2] > r.Sheets[0].ContentBox[2] || p.PaperBox[1] < r.Sheets[0].ContentBox[1] {
			t.Fatal("view enters title region")
		}
	}
	if err = d.UseLayout("S001"); err != nil {
		t.Fatal(err)
	}
	present := map[string]bool{}
	for _, e := range d.Entities() {
		if tx, ok := e.(*entity.Text); ok {
			present[tx.Value] = true
		}
	}
	// All of these fit on one row; long fields are checked separately below.
	for _, text := range []string{"構造設計事務所", "確認用構造図", "鉄骨造", "2026-11-21", "作図担当", "確認担当", "S-07", "伏図・軸組図"} {
		if !present[text] {
			t.Fatal("missing title information", text)
		}
	}
	layout, _ := d.Layout("S001")
	if layout.PlotSettings.Flags&128 == 0 || layout.PlotSettings.Flags&64 != 0 || layout.PlotSettings.StyleSheet != "" {
		t.Fatal("lineweight plotting is not explicit")
	}
	path := filepath.Join(t.TempDir(), "sheet.dxf")
	if err = d.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	readExportedDXF(t, path)
}

func TestDXFSheetValidationAndOverflow(t *testing.T) {
	f := documentFixture(t)
	views := []DXFViewSpec{{Kind: "plan", Floor: 2}, {Kind: "plan", Floor: 2}}
	for _, sheet := range []*DXFSheetTemplate{
		{Layout: "typo"}, {MarginMM: -1}, {TitleWidthMM: 1},
		{Office: strings.Repeat("長い説明", 1000)},
		{Pages: map[int]DXFSheetPage{1: {Number: "SAME"}, 2: {Number: "SAME"}}},
		{Revisions: []DXFSheetRevision{{Mark: "A"}}},
	} {
		_, _, err := f.BuildDXFDocument(DXFDocumentOptions{Views: views, Settings: &DXFDrawingSettings{Version: 1, Sheet: sheet}})
		if err == nil {
			t.Fatal("invalid/overflowing sheet metadata accepted", sheet)
		}
	}
	// Explicit compact output is still available.
	_, r, err := f.BuildDXFDocument(DXFDocumentOptions{Views: views, Settings: &DXFDrawingSettings{Version: 1, Sheet: &DXFSheetTemplate{Layout: "bottom"}}})
	if err != nil {
		t.Fatal(err)
	}
	if r.Sheets[0].ContentBox[1] != 40 {
		t.Fatal("bottom layout did not reserve title strip")
	}
	// All supported paper orientations must retain an empty right title column.
	for _, paper := range []string{"A3", "A4", "A3P", "A4P"} {
		_, r, err := f.BuildDXFDocument(DXFDocumentOptions{Paper: paper, Views: views[:1]})
		if err != nil {
			t.Fatal(paper, err)
		}
		if r.Sheets[0].TitleBox[2]-r.Sheets[0].TitleBox[0] != 45 {
			t.Fatal(paper, "title width")
		}
	}
}

func TestDXFSheetRotationAndFixedPaperOffset(t *testing.T) {
	f := documentFixture(t)
	cfg := &DXFDrawingSettings{Version: 1, Sheet: &DXFSheetTemplate{ViewRotations: map[string]int{"elevation:Y1": 90}}}
	d, r, err := f.BuildDXFDocument(DXFDocumentOptions{Views: []DXFViewSpec{{Kind: "elevation", Grid: "Y1"}, {Kind: "elevation", Grid: "Y1", Rotation: dxfPtr(0)}}, Settings: cfg})
	if err != nil {
		t.Fatal(err)
	}
	if r.Placements[0].Rotation != 90 || r.Placements[1].Rotation != 0 {
		t.Fatal("rotation precedence")
	}
	rotated := 0
	for _, e := range d.Entities() {
		if v, ok := e.(*entity.Viewport); ok && v.TwistAngle != 0 {
			rotated++
			jointNear(t, v.TwistAngle, -90)
			if len(v.ViewTarget) != 3 || v.ViewCenter[0] != 0 || v.ViewCenter[1] != 0 {
				t.Fatal("rotated view center is not target-relative")
			}
		}
	}
	for _, e := range d.Entities() {
		if _, ok := e.(*entity.Insert); ok && e.Layer().Name() == "SHEET_VIEWPORT" {
			t.Fatal("model block is on a non-plot layer")
		}
	}
	if rotated != 1 {
		t.Fatal("rotated viewport missing")
	}
	// Changing text height/rotation must not change a reviewed paper offset.
	f = jointFixture()
	e := jointBeam(f, jointNode(f, 0, 0), jointNode(f, 4, 0), 101)
	symbol := "FIXED"
	cfg = &DXFDrawingSettings{Version: 1, Style: &DXFDraftingStyle{TextHeightMM: 3.5}, Members: map[int]DXFDrawingMember{e.Num: {Symbol: &symbol, LabelOffsetPaperMM: &[2]float64{2, 1}}}}
	private, _, err := f.WithDXFDrawingSettings(cfg)
	if err != nil {
		t.Fatal(err)
	}
	private.drawingRotation = 90
	baselineCfg := cloneDXFDrawingSettings(cfg)
	m := baselineCfg.Members[e.Num]
	m.LabelOffsetPaperMM = &[2]float64{}
	baselineCfg.Members[e.Num] = m
	base, _, _ := f.WithDXFDrawingSettings(baselineCfg)
	base.drawingRotation = 90
	coordinates := func(frame *Frame) []float64 {
		path := filepath.Join(t.TempDir(), "offset.dxf")
		if err := frame.WriteDxfCrosssectionDimensions(path, 2, -.1, .1, 1, 1000, 125, 100); err != nil {
			t.Fatal(err)
		}
		drawing := readExportedDXF(t, path)
		return drawing.EntitiesOnLayer("DRAWING_FIXED_LABELS")[0].(*entity.Text).Coord1
	}
	a, b := coordinates(private), coordinates(base)
	jointNear(t, -(a[1]-b[1])/50, 2)
	jointNear(t, (a[0]-b[0])/50, 1)
	if _, err := parseDXFDocumentOptions(map[string]string{"GRIDS": "Y1", "ROTATE": "90"}, false); err != nil {
		t.Fatal(err)
	}
	if _, err := parseDXFDocumentOptions(map[string]string{"GRIDS": "Y1", "ROTATE": "45"}, false); err == nil {
		t.Fatal("invalid command rotation")
	}
}

func TestDXFDraftingNakano(t *testing.T) {
	input := os.Getenv("ST_DXF_TEST_INP")
	if input == "" {
		t.Skip("set ST_DXF_TEST_INP")
	}
	data, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	f := NewFrame()
	if err = f.ReadInpSnapshot(data, input); err != nil {
		t.Fatal(err)
	}
	if err = f.ReadLst(Ce(input, ".lst")); err != nil {
		t.Fatal(err)
	}
	f, err = f.dimensionFrame()
	if err != nil {
		t.Fatal(err)
	}
	before := f.DXFDrawingRevision()
	cfg := f.DefaultDXFDrawingSettings()
	cfg.Sheet = &DXFSheetTemplate{ViewRotations: map[string]int{"elevation:Y2": 90, "elevation:X1": 90}, Project: "nakano02_07 構造図", Office: "ST 自動作図（検証用）", Structure: "鉄骨造", Date: "2026-11-21", DrawnBy: "検証出力", StartNumber: 7, Notes: []string{"寸法単位：mm", "図面設定・製図スタイルの出力確認用"}, Revisions: []DXFSheetRevision{{Mark: "A", Date: "2026-11-21", Description: "製図スタイル・図枠確認"}}}
	dir := os.Getenv("ST_DXF_DRAFTING_TEST_OUTPUT")
	if dir == "" {
		dir = t.TempDir()
	}
	if err = os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	d, r, err := f.BuildDXFDocument(DXFDocumentOptions{Views: []DXFViewSpec{{Kind: "plan", Floor: 3}, {Kind: "elevation", Grid: "Y2"}, {Kind: "elevation", Grid: "X1"}}, Sections: true, Settings: cfg})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "nakano_drafting_sheets.dxf")
	if err = d.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	readExportedDXF(t, path)
	raw, _ := json.MarshalIndent(r, "", "  ")
	if err = os.WriteFile(filepath.Join(dir, "nakano_drafting_report.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	raw, _ = json.MarshalIndent(cfg, "", "  ")
	if err = os.WriteFile(filepath.Join(dir, "nakano_drafting.drawing.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	if before != f.DXFDrawingRevision() {
		t.Fatal("drafting modified the model")
	}
	for _, v := range r.Placements {
		if v.Kind == "elevation" && (v.Rotation != 90 || v.Scale > 100) {
			t.Fatal("rotated real-model elevation should fit at 1:100 or larger", v)
		}
	}
	if len(r.Sheets) != r.Pages {
		t.Fatal("missing page metadata")
	}
	t.Logf("%d sheets, %d sections, %d issues; %s through %s", r.Pages, len(r.Sections), len(r.DrawingIssues), r.Sheets[0].Number, r.Sheets[len(r.Sheets)-1].Number)
}
