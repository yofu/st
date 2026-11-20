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

func TestDXFMemberLabelPlacement(t *testing.T) {
	d := dxf.NewDrawing()
	d.AddLayer("TXTCOLUMN", dxf.DefaultColor, dxf.DefaultLineType, true)
	for _, s := range []string{"C1", "C2", "C3", "C4"} {
		d.Text(s, 0, 0, 0, 10)
	}
	if left := arrangeDXFMemberLabels(d); left != 0 {
		t.Fatal("labels could not be arranged", left)
	}
	var boxes [][4]float64
	for _, e := range d.EntitiesOnLayer("TXTCOLUMN") {
		lo, hi := e.BBox()
		box := [4]float64{lo[0], lo[1], hi[0], hi[1]}
		for _, other := range boxes {
			if overlapDXFBox(box, other, 0) {
				t.Fatal("member symbols overlap")
			}
		}
		boxes = append(boxes, box)
	}
	if len(d.EntitiesOnLayer("LABEL_LEADER")) != 3 {
		t.Fatal("moved symbols lack leaders")
	}
}

func TestDXFDocumentCommands(t *testing.T) {
	f := documentFixture(t)
	w := &dxfDimensionCommandWindow{frame: f, dir: t.TempDir()}
	for _, spec := range []struct {
		command string
		pages   int
	}{
		{":drawings all.dxf -plans=2 -grids=Y1 -sections -sect=101,201", 3},
		{":dxfsections sections.dxf -sect=101,201 -sectioncolumns=1 -sectionrows=1", 2},
	} {
		if err := exCommand(w, spec.command, nil, nil); err != nil {
			t.Fatal(err)
		}
		file := strings.Fields(spec.command)[1]
		d := readExportedDXF(t, filepath.Join(w.dir, file))
		if len(d.Layouts()) != spec.pages+1 {
			t.Fatal("command lost sheets")
		}
		style, err := d.Style("ST_DRAWING", false)
		if err != nil || style.FontName != "msgothic.ttc" {
			t.Fatal("Japanese drawing font lost", err)
		}
	}
}

func TestDXFDocumentDoesNotUseTemporaryDisplayFilters(t *testing.T) {
	f := documentFixture(t)
	o := DXFDocumentOptions{Views: []DXFViewSpec{{Kind: "plan", Floor: 2}}}
	a, reportA, err := f.BuildDXFDocument(o)
	if err != nil {
		t.Fatal(err)
	}
	f.Show.Etype[COLUMN] = false
	for _, e := range f.Elems {
		e.Hide()
	}
	for _, n := range f.Nodes {
		n.Hide()
	}
	f.Show.Zrange = []float64{100, 101}
	b, reportB, err := f.BuildDXFDocument(o)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Entities()) != len(b.Entities()) || reportA.Placements[0].ModelBox != reportB.Placements[0].ModelBox {
		t.Fatal("GUI display changed drawing contents")
	}
	if f.Show.Etype[COLUMN] || f.Show.Zrange[0] != 100 {
		t.Fatal("drawing changed GUI display")
	}
}

func documentFixture(t *testing.T) *Frame {
	f := dxfExportFixture(t)
	f.Kijuns = map[string]*Kijun{}
	f.AddKijun("X1", []float64{0, -1, 0}, []float64{0, 3, 0})
	f.AddKijun("X2", []float64{4, -1, 0}, []float64{4, 3, 0})
	f.AddKijun("Y1", []float64{-1, 0, 0}, []float64{5, 0, 0})
	f.AddKijun("Y2", []float64{-1, 2, 0}, []float64{5, 2, 0})
	return f
}

func TestDXFSectionRecordsAndCards(t *testing.T) {
	f := documentFixture(t)
	rc := f.Sects[101].Allow.(*RCColumn)
	rc.Concrete = Concrete{name: "Fc24"}
	rc.Hoops.Name = "2-D10@100"
	for i := range rc.Reins {
		rc.Reins[i].Caption = "D19"
	}
	records, err := f.DXFSections([]int{301, 101, 201})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 3 || records[0].ID != 101 || records[0].Shape != "300×300 mm" || records[0].MainBars != "4-D19" || records[0].Hoops != "2-D10@100" {
		t.Fatalf("wrong section data %+v", records)
	}
	if len(records[0].Bars) != 4 || records[0].Bars[0].X != -100 || records[0].Bars[0].Y != -100 {
		t.Fatal("bar coordinates are not in mm")
	}
	if records[1].Shape != "HKYOU 400×200×8×12" {
		t.Fatal(records[1].Shape)
	}
	d, err := buildSectionCard(records[0], 10, 120, 100, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.EntitiesOnLayer("SECTION_REBAR")) != 4 {
		t.Fatal("missing RC bars")
	}
	for _, e := range d.EntitiesOnLayer("SECTION_REBAR") {
		c := e.(*entity.Circle)
		if math.Abs(c.Radius-math.Sqrt(2.865/math.Pi)*10) > 1e-9 {
			t.Fatal("wrong bar radius")
		}
	}
	if len(records[0].Floors) == 0 {
		t.Fatal("missing use floors")
	}
	// No invented geometry for an undefined section, and clear failure for IDs.
	s := NewSect()
	s.Num = 999
	f.Sects[999] = s
	missing, err := f.DXFSections([]int{999})
	if err != nil || len(missing[0].Warnings) == 0 {
		t.Fatal("missing definition did not warn", err)
	}
	if _, err = f.DXFSections([]int{123456}); err == nil {
		t.Fatal("unknown section accepted")
	}
	area := NewSect()
	area.Num = 401
	area.Allow = NewSBrace(401, SAREA{Area: 2.5}, Steel{})
	f.Sects[401] = area
	areaOnly, err := f.DXFSections([]int{401})
	if err != nil {
		t.Fatal(err)
	}
	if areaOnly[0].Shape != "断面積 250 mm²（外形未定義）" || len(areaOnly[0].vertices) != 0 || len(areaOnly[0].circles) != 0 {
		t.Fatal("area-only section was given an invented shape")
	}
}

func TestDXFDocumentLayoutDimensionsAndPagination(t *testing.T) {
	f := documentFixture(t)
	o := DXFDocumentOptions{Title: "構造図", Columns: 2, Rows: 1, Views: []DXFViewSpec{{Kind: "plan", Floor: 2}, {Kind: "elevation", Grid: "Y1"}}, Sections: true, SectionIDs: []int{101, 201, 301}, SectionColumns: 2, SectionRows: 1}
	d, report, err := f.BuildDXFDocument(o)
	if err != nil {
		t.Fatal(err)
	}
	if report.Pages != 3 || len(report.Placements) != 5 {
		t.Fatalf("wrong pagination %+v", report)
	}
	for i, p := range report.Placements {
		if p.Scale <= 0 || p.PaperBox[0] < report.Sheets[p.Page-1].ContentBox[0] || p.PaperBox[2] > report.Sheets[p.Page-1].ContentBox[2] || p.PaperBox[1] < report.Sheets[p.Page-1].ContentBox[1] || p.PaperBox[3] > report.Sheets[p.Page-1].ContentBox[3] {
			t.Fatal("placement outside sheet", p)
		}
		for _, q := range report.Placements[:i] {
			if p.Page == q.Page && p.PaperBox[0] < q.PaperBox[2] && p.PaperBox[2] > q.PaperBox[0] && p.PaperBox[1] < q.PaperBox[3] && p.PaperBox[3] > q.PaperBox[1] {
				t.Fatal("overlapping cells")
			}
		}
		if (p.ModelBox[2]-p.ModelBox[0])/p.Scale > p.PaperBox[2]-p.PaperBox[0]-4+1e-6 || (p.ModelBox[3]-p.ModelBox[1])/p.Scale > p.PaperBox[3]-p.PaperBox[1]-12+1e-6 {
			t.Fatal("viewport crops geometry", p)
		}
	}
	path := filepath.Join(t.TempDir(), "sheets.dxf")
	if err = d.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	r := readExportedDXF(t, path)
	if len(r.Layouts()) != 4 {
		t.Fatal("layouts lost on read")
	}
	seen := map[string]bool{}
	dimCount := 0
	for _, b := range r.Blocks() {
		if !strings.HasPrefix(b.Name, "ST_VIEW_") {
			continue
		}
		for _, e := range b.Entities {
			if dim, ok := e.(*entity.Dimension); ok {
				if seen[dim.BlockName] {
					t.Fatal("views share dimension graphics", dim.BlockName)
				}
				seen[dim.BlockName] = true
				graphics, err := r.Block(dim.BlockName)
				if err != nil || len(graphics.Entities) == 0 {
					t.Fatal("dimension graphics lost", err)
				}
				if dim.Measurement <= 0 {
					t.Fatal("dimension measurement lost")
				}
				dimCount++
			}
		}
	}
	if dimCount != 9 {
		t.Fatalf("dimension count %d want 9", dimCount)
	}
	if _, _, err = f.BuildDXFDocument(DXFDocumentOptions{Scale: 1, Views: o.Views}); err == nil {
		t.Fatal("accepted scale which crops drawing")
	}
	if _, _, err = f.BuildDXFDocument(DXFDocumentOptions{Sections: true, SectionIDs: []int{101}, SectionColumns: 5}); err == nil {
		t.Fatal("invalid layout accepted")
	}
	// Reading a sidecar and composing sheets must not edit source grid state.
	if _, _, err = f.resolveDxfJiku("Y1"); err != nil {
		t.Fatal(err)
	}
}

func TestDXFDocumentNakano(t *testing.T) {
	input := os.Getenv("ST_DXF_TEST_INP")
	if input == "" {
		t.Skip("set ST_DXF_TEST_INP for the real INP/LST/KJN test")
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
	before := f.GUIRevision()
	dir := os.Getenv("ST_DXF_TEST_OUTPUT")
	if dir == "" {
		dir = t.TempDir()
	}
	if err = os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	report, err := f.WriteDXFDocument(filepath.Join(dir, "nakano_sheets.dxf"), DXFDocumentOptions{Title: "nakano02_07 構造図", Views: []DXFViewSpec{{Kind: "plan", Floor: 3}, {Kind: "elevation", Grid: "Y2"}, {Kind: "elevation", Grid: "X1"}}, Sections: true})
	if err != nil {
		t.Fatal(err)
	}
	if before != f.GUIRevision() || len(f.Kijuns) != 0 {
		t.Fatal("document changed the original model")
	}
	if len(report.Sections) == 0 || report.Pages < 4 {
		t.Fatal("missing section pages")
	}
	d := readExportedDXF(t, filepath.Join(dir, "nakano_sheets.dxf"))
	if len(d.Layouts()) != report.Pages+1 {
		t.Fatal("missing sheet layouts")
	}
	counts := map[int]int{}
	for _, p := range report.Placements {
		if p.Kind == "section" {
			counts[p.Section]++
		}
	}
	for _, s := range report.Sections {
		if counts[s.ID] != 1 {
			t.Fatal("section duplicated or missing", s.ID)
		}
	}
	raw, _ := json.MarshalIndent(report, "", "  ")
	if err = os.WriteFile(filepath.Join(dir, "nakano_sheets.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("%d pages, %d sections, %d warnings; scales=%v", report.Pages, len(report.Sections), len(report.Warnings), report.Placements[:3])
}
