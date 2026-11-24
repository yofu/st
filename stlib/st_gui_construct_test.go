package st

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestConstructExplicitAndRejectPartialEdits(t *testing.T) {
	f := NewFrame()
	a := f.AddNode(0, 0, 0)
	q := GUIConstruction{
		Properties: []BuildProperty{{11, "SN400"}},
		Sections:   []BuildSection{{21, "B1", 11, "HKYOU", []float64{200, 200, 8, 12}}},
		Nodes:      []BuildNode{{700, []float64{1, 2, 2}}},
		Elements:   []BuildElement{{5000, []int{a.Num, 700}, "BRACE", 21}},
	}
	captures := 0
	capture := func() {
		captures++
		if len(f.Nodes) != 1 {
			t.Fatal("undo captured partial batch")
		}
	}
	r, err := f.Construct(q, capture)
	if err != nil || r.MinLength != 3 || r.MaxLength != 3 || r.Directions["oblique"] != 1 || r.MemberTypes["BRACE"] != 1 {
		t.Fatalf("explicit batch: %+v %v", r, err)
	}
	if captures != 1 || f.Nodes[a.Num] != a || f.Elems[5000].Enod[0] != a || f.Elems[5000].Frame != f || f.Maxnnum != 700 || f.Maxenum != 5000 {
		t.Fatal("incorrect references or counters")
	}
	digest := f.GUIRevision()
	noCommit := func() { t.Fatal("invalid batch reached undo/commit") }
	for _, bad := range []GUIConstruction{
		q,
		{Properties: []BuildProperty{{12, "SN490"}}, Nodes: []BuildNode{{701, []float64{9, 9, 9}}}, Elements: []BuildElement{{5001, []int{701, 42}, "COLUMN", 21}}},
		{Nodes: []BuildNode{{701, []float64{1, 2, 2.00001}}}},
		{Elements: []BuildElement{{5001, []int{700, a.Num}, "COLUMN", 21}}},
		{Sections: []BuildSection{{22, "bad", 11, "HKYOU", []float64{200, 200, 8, 101}}}},
		{Grid: &BuildGrid{Cells: []int{1000000000, 1000000000, 0}, Pitch: []float64{1, 1, 0}, Type: "GIRDER", Section: 21}},
	} {
		if _, err = f.Construct(bad, noCommit); err == nil || f.GUIRevision() != digest {
			t.Fatalf("partially accepted invalid batch: %+v", bad)
		}
	}
	r, err = f.Construct(GUIConstruction{Grid: &BuildGrid{Origin: []float64{10, 0, 0}, Cells: []int{0, 0, 2}, Pitch: []float64{0, 0, 3}, Type: "COLUMN", Section: 21}}, nil)
	if err != nil || r.Directions["Z"] != 2 || r.Added.Nodes != 3 || r.MinLength != 3 || r.Total.Elements != 3 || !slices.Equal(f.Nodes[703].Coord, []float64{10, 0, 6}) {
		t.Fatalf("extension: %+v %v", r, err)
	}
}

type autoExtractTestWindow struct {
	ExModer
	cwd   string
	found string
}

func (w autoExtractTestWindow) Cwd() string { return w.cwd }
func (w autoExtractTestWindow) SearchFile(fn string) (string, error) {
	if w.found != "" {
		return w.found, nil
	}
	return "", os.ErrNotExist
}

func TestResolveExistingFileAndAutoExtractArclm(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "unique_mcp_resolve_test.inp")
	if err := os.WriteFile(target, []byte("END\n"), 0644); err != nil {
		t.Fatal(err)
	}
	w := autoExtractTestWindow{cwd: dir, found: target}
	got := resolveExistingFile(w, filepath.Join(dir, "missing.inp"), "unique_mcp_resolve_test.inp")
	if got != target {
		t.Fatalf("resolveExistingFile = %q, want %q", got, target)
	}

	f := NewFrame()
	if _, err := f.Construct(GUIConstruction{
		Properties: []BuildProperty{{101, "SN400"}},
		Sections:   []BuildSection{{501, "G51", 101, "HKYOU", []float64{200, 200, 8, 12}}},
		Grid:       &BuildGrid{Origin: []float64{0, 0, 0}, Cells: []int{1, 1, 0}, Pitch: []float64{2, 2, 0}, FirstNode: 101, FirstElement: 101, Type: "GIRDER", Section: 501},
	}, nil); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int{101, 102, 103, 104} {
		copy(f.Nodes[id].Conf, []bool{true, true, true, false, false, false})
	}
	af, err := ensureArclmExtracted(w, f, ":bclng001", "L")
	if err != nil || af == nil {
		t.Fatalf("ensureArclmExtracted failed: af=%v err=%v", af, err)
	}
	if f.Ai.Nfloor != 1 || len(f.Ai.Boundary) != 2 {
		t.Fatalf("expected auto-configured 1-floor boundary, got Nfloor=%d Boundary=%v", f.Ai.Nfloor, f.Ai.Boundary)
	}
}
