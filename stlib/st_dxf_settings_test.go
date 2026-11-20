package st

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yofu/dxf"
	"github.com/yofu/dxf/entity"
)

func drawingPtr[T any](v T) *T { return &v }
func drawingHasIssue(r *DXFDrawingCheckReport, code string) bool {
	for _, i := range r.Issues {
		if i.Code == code {
			return true
		}
	}
	return false
}
func drawingSettingsFixture(t *testing.T) (*Frame, *DXFDrawingSettings, int) {
	t.Helper()
	f := dxfExportFixture(t)
	s := f.DefaultDXFDrawingSettings()
	s.GroundLevelMM = drawingPtr(0.)
	ids := SortedElem(f.Elems, func(e *Elem) float64 { return float64(e.Num) })
	return f, s, ids[0].Num
}
func TestDXFDrawingSettingsPersistence(t *testing.T) {
	f, s, id := drawingSettingsFixture(t)
	e := f.Elems[id]
	notes := []string{"確認済みの注記"}
	s.Sections[e.Sect.Num] = DXFDrawingMember{Symbol: drawingPtr("共通"), OffsetMM: &[3]float64{20, 0, 0}, Notes: &notes}
	s.Members[id] = DXFDrawingMember{Symbol: drawingPtr("個別"), OffsetMM: &[3]float64{}, Notes: &[]string{}, Expected: &DXFDrawingIdentity{Section: e.Sect.Num, Nodes: []int{e.Enod[0].Num, e.Enod[1].Num}}}
	file := filepath.Join(t.TempDir(), "model.drawing.json")
	if err := WriteNewDXFDrawingSettings(file, s); err != nil {
		t.Fatal(err)
	}
	if err := WriteNewDXFDrawingSettings(file, &DXFDrawingSettings{Version: 1}); err == nil {
		t.Fatal("overwrote manual settings")
	}
	loaded, err := ReadDXFDrawingSettings(file)
	if err != nil {
		t.Fatal(err)
	}
	r := f.CheckDXFDrawingSettings(loaded)
	if r.HasErrors() {
		t.Fatal(r.Issues)
	}
	var m DXFDrawingMember
	for _, v := range r.Members {
		if v.Element == id {
			m = v.Settings
		}
	}
	if *m.Symbol != "個別" || *m.OffsetMM != [3]float64{} || m.Notes == nil || len(*m.Notes) != 0 {
		t.Fatal("explicit zero/empty did not override defaults", m)
	}
	before := f.DXFDrawingRevision()
	clone, _, err := f.WithDXFDrawingSettings(loaded)
	if err != nil {
		t.Fatal(err)
	}
	if f.DXFDrawingRevision() != before || f.drawingSettings != nil {
		t.Fatal("edited source frame")
	}
	snapshot := clone.Snapshot()
	snapshot.drawingSettings.Members[id] = DXFDrawingMember{}
	if clone.dxfMemberSymbol(clone.Elems[id]) != "個別" {
		t.Fatal("settings snapshot shared state")
	}
}
func TestDXFDrawingSettingsValidation(t *testing.T) {
	for _, raw := range []string{`null`, `{}`, `{"version":2}`, `{"version":1,"typo":1}`, `{"version":1,"members":{"1":{"unknown":0}}}`, `{"version":1}{}`, `{"version":1,"members":{"1":{"offset_mm":[1,2,3,4]}}}`, `{"version":1,"annotations":[{"position_mm":[1]}]}`} {
		if _, err := DecodeDXFDrawingSettings([]byte(raw)); err == nil {
			t.Fatal("accepted", raw)
		}
	}
	if _, err := DecodeDXFDrawingSettings(bytes.Repeat([]byte(" "), DXFDrawingSettingsLimit+1)); err == nil {
		t.Fatal("size limit")
	}
	f, s, id := drawingSettingsFixture(t)
	for _, tc := range []struct {
		name, code string
		change     func(*DXFDrawingSettings)
	}{
		{"missing member", "missing_member", func(s *DXFDrawingSettings) { s.Members[999999] = DXFDrawingMember{} }},
		{"missing section", "missing_section", func(s *DXFDrawingSettings) { s.Sections[999999] = DXFDrawingMember{} }},
		{"identity", "member_identity_changed", func(s *DXFDrawingSettings) {
			s.Members[id] = DXFDrawingMember{Expected: &DXFDrawingIdentity{Section: 999, Nodes: []int{1, 2}}}
		}},
		{"invalid grid", "invalid_grid", func(s *DXFDrawingSettings) { s.Grids = append(s.Grids, DXFDrawingGrid{Name: "BAD", Axis: "Z"}) }},
		{"duplicate level", "invalid_level", func(s *DXFDrawingSettings) { s.Levels = append(s.Levels, s.Levels[0]) }},
		{"unknown FL", "unknown_level", func(s *DXFDrawingSettings) { s.Members[id] = DXFDrawingMember{Level: drawingPtr("MISSING")} }},
		{"NaN", "invalid_offset", func(s *DXFDrawingSettings) { s.Members[id] = DXFDrawingMember{OffsetMM: &[3]float64{math.NaN(), 0, 0}} }},
		{"anchor", "missing_annotation_anchor", func(s *DXFDrawingSettings) {
			s.Annotations = []DXFDrawingAnnotation{{ID: "n1", View: "all", Text: "note", Element: 99999}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := cloneDXFDrawingSettings(s)
			tc.change(c)
			r := f.CheckDXFDrawingSettings(c)
			if !r.HasErrors() || !drawingHasIssue(r, tc.code) {
				t.Fatal(r.Issues)
			}
		})
	}
	s.SourceRevision = "old"
	r := f.CheckDXFDrawingSettings(s)
	if !drawingHasIssue(r, "source_changed") {
		t.Fatal("missing stale-model warning")
	}
}
func TestDXFDrawingSettingsInferenceAndKJN(t *testing.T) {
	f := dxfExportFixture(t)
	s := f.DefaultDXFDrawingSettings()
	if s.GroundLevelMM != nil {
		t.Fatal("invented GL")
	}
	if !drawingHasIssue(f.CheckDXFDrawingSettings(s), "inferred_datums") {
		t.Fatal("inference presented as confirmed")
	}
	f.AddKijun("GL", []float64{0, 0, -.1}, []float64{1, 0, -.1})
	f.AddKijun("2FL", []float64{0, 0, 3.05}, []float64{1, 0, 3.05})
	s = f.DefaultDXFDrawingSettings()
	if s.GroundLevelMM == nil || *s.GroundLevelMM != -100 {
		t.Fatal("lost GL")
	}
	if len(s.Levels) == 0 || s.Levels[0].Source != "kjn" || !s.Levels[0].Confirmed {
		t.Fatal(s.Levels)
	}
	revision := f.DXFDrawingRevision()
	f.Kijuns["2fl"].Start[2] += .05
	if revision == f.DXFDrawingRevision() {
		t.Fatal("KJN change not detected")
	}
}
func TestDXFDrawingSettingsRendering(t *testing.T) {
	f, s, _ := drawingSettingsFixture(t)
	e := SortedElem(f.Elems, func(e *Elem) float64 { return float64(e.Num) })[0]
	s.Sections[e.Sect.Num] = DXFDrawingMember{Symbol: drawingPtr("DRAWING_COLUMN")}
	s.Grids = append(s.Grids, DXFDrawingGrid{Name: "X_EXTRA", Axis: "X", PositionMM: 2000, Confirmed: true})
	s.Levels = []DXFDrawingLevel{{Name: "2FL", Floor: 2, ElevationMM: 3050, Confirmed: true}, {Name: "RFL", Floor: 3, ElevationMM: 6050, Confirmed: true}}
	s.Annotations = []DXFDrawingAnnotation{{ID: "note", View: "plan:2", Text: "確認用注記", PositionMM: &[2]float64{500, -500}, OffsetPaperMM: [2]float64{2, 3}}}
	private, report, err := f.WithDXFDrawingSettings(s)
	if err != nil {
		t.Fatal(err, report)
	}
	levels := private.dxfLevels()
	if levels[1].value != 3.05 {
		t.Fatal("FL not applied", levels)
	}
	if _, _, err = private.resolveDxfJiku("X_EXTRA"); err != nil {
		t.Fatal(err)
	}
	// Paper offsets survive a changed plot scale; model coordinates do not move.
	for _, scale := range []float64{50, 100} {
		d := dxf.NewDrawing()
		if err := private.drawDXFDrawingAnnotations(d, DXFViewSpec{Kind: "plan", Floor: 2}, scale); err != nil {
			t.Fatal(err)
		}
		tx := d.EntitiesOnLayer("DRAWING_NOTES")[0].(*entity.Text)
		jointNear(t, tx.Coord1[0], 500+2*scale)
		jointNear(t, tx.Coord1[1], -500+3*scale)
	}
	before := f.DXFDrawingRevision()
	d, r, err := f.BuildDXFDocument(DXFDocumentOptions{Views: []DXFViewSpec{{Kind: "plan", Floor: 2}}, Sections: true, SectionIDs: []int{e.Sect.Num}, Settings: s})
	if err != nil {
		t.Fatal(err)
	}
	if f.DXFDrawingRevision() != before {
		t.Fatal("model changed")
	}
	if len(r.DrawingIssues) == 0 || r.Sections[0].Symbol != "DRAWING_COLUMN" {
		t.Fatal("report did not use settings", r)
	}
	found := false
	for _, b := range d.Blocks() {
		for _, e := range b.Entities {
			if tx, ok := e.(*entity.Text); ok && tx.Value == "確認用注記" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("annotation missing from view block")
	}
	path := filepath.Join(t.TempDir(), "configured.dxf")
	if err = d.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	readExportedDXF(t, path)
}
func TestDXFDrawingSettingsNativeCommands(t *testing.T) {
	f := documentFixture(t)
	dir := t.TempDir()
	f.Path = filepath.Join(dir, "model.inp")
	w := &dxfDimensionCommandWindow{frame: f, dir: dir}
	if err := exCommand(w, ":drawinginit model.drawing.json", nil, nil); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "model.drawing.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = exCommand(w, ":drawinginit model.drawing.json", nil, nil); err == nil {
		t.Fatal("reinitialized settings")
	}
	if err = exCommand(w, ":drawingcheck model.drawing.json", nil, nil); err != nil {
		t.Fatal(err)
	}
	if err = exCommand(w, ":drawings configured.dxf -plans=2 -settings=model.drawing.json", nil, nil); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("export rewrote settings")
	}
}

func TestDXFDrawingSettingsFixedLabelsAndMerge(t *testing.T) {
	f := jointFixture()
	a, b, c := jointNode(f, 0, 0), jointNode(f, 2, 0), jointNode(f, 4, 0)
	e := jointBeam(f, a, b, 101)
	jointBeam(f, b, c, 101)
	symbol := "固定符号"
	settings := &DXFDrawingSettings{Version: 1, Members: map[int]DXFDrawingMember{e.Num: {Symbol: &symbol}}}
	baseline, _, err := f.WithDXFDrawingSettings(settings)
	if err != nil {
		t.Fatal(err)
	}
	if len(baseline.prepareDXFStrokes(2, -.1, .1, 0, 1)) != 2 {
		t.Fatal("member override lost by aggregation")
	}
	m := settings.Members[e.Num]
	m.LabelOffsetPaperMM = &[2]float64{2, 1}
	settings.Members[e.Num] = m
	configured, _, err := f.WithDXFDrawingSettings(settings)
	if err != nil {
		t.Fatal(err)
	}
	for _, paperScale := range []float64{50, 100} {
		base, err := baseline.buildDxfCrosssection(2, -.1, .1, 1, 1000, 2.5*paperScale, 100, false)
		if err != nil {
			t.Fatal(err)
		}
		d, err := configured.buildDxfCrosssection(2, -.1, .1, 1, 1000, 2.5*paperScale, 100, false)
		if err != nil {
			t.Fatal(err)
		}
		var original *entity.Text
		for _, e := range base.Entities() {
			if tx, ok := e.(*entity.Text); ok && tx.Value == symbol {
				original = tx
			}
		}
		fixed := d.EntitiesOnLayer("DRAWING_FIXED_LABELS")
		if original == nil || len(fixed) != 1 {
			t.Fatal("missing fixed label")
		}
		tx := fixed[0].(*entity.Text)
		jointNear(t, tx.Coord1[0]-original.Coord1[0], 2*paperScale)
		jointNear(t, tx.Coord1[1]-original.Coord1[1], paperScale)
		coord := append([]float64(nil), tx.Coord1...)
		arrangeDXFMemberLabels(d)
		for i, v := range coord {
			jointNear(t, tx.Coord1[i], v)
		}
		style, err := d.Style("ST_DRAWING", false)
		if err != nil || style.FontName != "msgothic.ttc" {
			t.Fatal("Japanese label font missing", err)
		}
	}
	// Direct dimensioned exports also load the persisted profile.
	dir := t.TempDir()
	f.Path = filepath.Join(dir, "model.inp")
	if err = WriteNewDXFDrawingSettings(DXFDrawingSettingsPath(f.Path), settings); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "plan.dxf")
	if err = f.WriteDxfCrosssectionDimensions(path, 2, -.1, .1, 1, 1000, 125, 100); err != nil {
		t.Fatal(err)
	}
	d := readExportedDXF(t, path)
	if len(d.EntitiesOnLayer("DRAWING_FIXED_LABELS")) != 1 {
		t.Fatal("sidecar not applied to dimensioned export")
	}
}
func TestDXFDrawingSettingsNakano(t *testing.T) {
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
	s := f.DefaultDXFDrawingSettings()
	report := f.CheckDXFDrawingSettings(s)
	if report.HasErrors() {
		t.Fatal(report.Issues)
	}
	if !drawingHasIssue(report, "inp_lst_mismatch") {
		t.Fatal("real INP/LST mismatch not reported")
	}
	dir := os.Getenv("ST_DXF_SETTINGS_TEST_OUTPUT")
	if dir == "" {
		dir = t.TempDir()
	}
	if err = os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	settingsPath := filepath.Join(dir, "nakano02_07.drawing.json")
	// The output directory is dedicated to this test, use a fresh name on reruns.
	if _, err = os.Stat(settingsPath); err == nil {
		settingsPath = filepath.Join(t.TempDir(), "nakano02_07.drawing.json")
	}
	if err = WriteNewDXFDrawingSettings(settingsPath, s); err != nil {
		t.Fatal(err)
	}
	loaded, err := ReadDXFDrawingSettings(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	after := f.CheckDXFDrawingSettings(loaded)
	b1, _ := json.Marshal(report)
	b2, _ := json.Marshal(after)
	if !bytes.Equal(b1, b2) {
		t.Fatal("settings roundtrip lost resolved data")
	}
	cfg := cloneDXFDrawingSettings(loaded)
	cfg.Annotations = []DXFDrawingAnnotation{{ID: "regeneration", View: "plan:3", Text: "図面設定 再出力確認", PositionMM: &[2]float64{0, -1200}}}
	if _, err = f.WriteDXFDocument(filepath.Join(dir, "nakano_settings_sheets.dxf"), DXFDocumentOptions{Title: "nakano 図面設定確認", Views: []DXFViewSpec{{Kind: "plan", Floor: 3}, {Kind: "elevation", Grid: "Y2"}}, Sections: true, Settings: cfg}); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.MarshalIndent(report, "", "  ")
	os.WriteFile(filepath.Join(dir, "nakano_settings_check.json"), raw, 0600)
	if f.DXFDrawingRevision() != before {
		t.Fatal("source model changed")
	}
	readExportedDXF(t, filepath.Join(dir, "nakano_settings_sheets.dxf"))
	var codes []string
	for _, i := range report.Issues {
		codes = append(codes, i.Code)
	}
	t.Logf("%d members, %d levels, %d grids, %d issues: %s", len(report.Members), len(s.Levels), len(s.Grids), len(report.Issues), strings.Join(codes, ","))
}
