package st

import (
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
