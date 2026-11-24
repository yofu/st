package st

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestAnalysisBracesTrackPanelEdits(t *testing.T) {
	f := yieldEndInputFrame()
	panelSect := f.AddSect(801)
	fig := NewFig()
	fig.Prop = f.Props[1]
	fig.Value["THICK"] = .2
	panelSect.Figs = []*Fig{fig}
	n := []*Node{f.AddNode(0, 0, 0), f.AddNode(4, 0, 0), f.AddNode(4, 0, 3), f.AddNode(0, 0, 3)}
	wall := f.AddPlateElem(-1, n, panelSect, WALL)
	f.AddAnalysisBraces()
	a, b := wall.Children[0], wall.Children[1]
	before := a.Sect
	area := before.Figs[0].Value["AREA"]
	if len(f.Elems) != 3 || a.Sect != b.Sect || a.Etype != WBRACE || area <= 0 {
		t.Fatal("missing original wall braces")
	}
	f.ElemSet["diagonal"] = []*Elem{a}
	// An old failed adoption can leave a third generated brace on the same
	// diagonal. A standalone member using the same section on another edge
	// must survive the repair.
	duplicate := f.AddLineElem(-1, slices.Clone(a.Enod), before, WBRACE)
	standalone := f.AddLineElem(-1, []*Node{n[0], n[1]}, before, WBRACE)
	// An unchanged extraction must preserve a loaded result and its member pointer.
	a.Stress["L"] = map[int][]float64{n[0].Num: {1, 2, 3, 4, 5, 6}}
	f.AddAnalysisBraces()
	if wall.Children[0] != a || len(a.Stress["L"]) != 1 {
		t.Fatal("unchanged wall lost its brace result")
	}
	if f.Elems[duplicate.Num] != nil || f.Elems[standalone.Num] != standalone {
		t.Fatal("incorrect reconciliation of parentless braces")
	}
	fig.Value["THICK"] *= 2
	wall.Skip[1] = true
	f.AddAnalysisBraces()
	if wall.Children[0] != a || wall.Children[1] != b || f.ElemSet["diagonal"][0] != a || a.Sect == before ||
		a.Sect.Figs[0].Value["AREA"] != 2*area || a.Sect != b.Sect || !a.Skip[1] || !b.Skip[1] ||
		len(a.Stress) != 0 || f.Sects[before.Num] != standalone.Sect {
		t.Fatal("section change did not update the existing diagonals")
	}
	f.DeleteElem(standalone.Num)
	// Rotate the node list: the second diagonal becomes the first, and one
	// member reverses direction. Both physical diagonals keep their numbers.
	wall.Enod = []*Node{n[1], n[2], n[3], n[0]}
	f.AddAnalysisBraces()
	if wall.Children[0] != b || wall.Children[1] != a || !b.Eldest || a.Eldest || !slices.Equal(a.Enod, []*Node{n[2], n[0]}) {
		t.Fatal("incorrect reordered diagonals")
	}
	wall.Enods = 3
	wall.Enod = wall.Enod[:3]
	f.AddAnalysisBraces()
	if len(f.Elems) != 1 || wall.Children[0] != nil || wall.Children[1] != nil || len(f.ElemSet["diagonal"]) != 0 || len(f.Sects) != 1 {
		t.Fatal("three-node wall retained derived elements or sections")
	}
}

func extractBraceTestInputs(t *testing.T, f *Frame) map[string][]byte {
	t.Helper()
	if err := f.ExtractArclm(Ce(f.Path, ".wgt")); err != nil {
		t.Fatal(err)
	}
	if err := f.SaveAsArclm(f.Path); err != nil {
		t.Fatal(err)
	}
	files := make(map[string][]byte)
	for _, p := range []string{"L", "X", "Y"} {
		data, err := os.ReadFile(f.DataFileName[p])
		if err != nil {
			t.Fatal(err)
		}
		files[p] = data
	}
	return files
}

func TestNakanoRepeatedBraceExtraction(t *testing.T) {
	f := readNakanoSteelYieldModel(t)
	f.Path = filepath.Join(t.TempDir(), "extracted.inp")
	first := extractBraceTestInputs(t, f)
	count := func() int {
		total := 0
		for _, e := range f.Elems {
			if e.Etype == SBRACE {
				total++
			}
		}
		return total
	}
	if n := count(); n != 134 {
		t.Fatalf("first extraction: %d slab braces, want 134", n)
	}
	second := extractBraceTestInputs(t, f)
	for p, data := range first {
		if !bytes.Equal(data, second[p]) {
			t.Fatalf("%s analysis input changed on repeated extraction", p)
		}
	}
	if n := count(); n != 134 {
		t.Fatalf("second extraction: %d slab braces", n)
	}
	var panel *Elem
	for _, e := range f.Elems {
		if e.Etype == SLAB && len(e.Children) == 2 && e.Children[0] != nil &&
			e.Wrect[0] == 0 && e.Wrect[1] == 0 && !slices.Contains(e.Skip, true) && (panel == nil || panel.Num > e.Num) {
			panel = e
		}
	}
	if panel == nil {
		t.Fatal("expected braced slab without an opening in all three load cases")
	}
	a, b := panel.Children[0].Num, panel.Children[1].Num
	area := panel.Children[0].Sect.Figs[0].Value["AREA"]
	l, h := panel.PlateSize()
	panel.Wrect = []float64{.2 * l, .2 * h} // opening reduction = 1.25 * sqrt(.2 * .2) = .25
	changed := extractBraceTestInputs(t, f)
	if count() != 134 {
		t.Fatal("editing one opening changed the brace count")
	}
	for p, af := range f.Arclms {
		if bytes.Equal(second[p], changed[p]) {
			t.Fatalf("%s: opening not reflected in analysis input", p)
		}
		seen := 0
		for _, e := range af.Elems {
			if e.Num == a || e.Num == b {
				seen++
				if math.Abs(e.Sect.Value[0]-.75*area) > 1e-12*area {
					t.Fatalf("%s ELEM %d: area %g, expected %g", p, e.Num, e.Sect.Value[0], .75*area)
				}
			}
		}
		if seen != 2 {
			t.Fatalf("%s: expected both original brace numbers after opening edit", p)
		}
	}
	f.DeleteElem(panel.Num)
	deleted := extractBraceTestInputs(t, f)
	if count() != 132 || f.Elems[a] != nil || f.Elems[b] != nil {
		t.Fatal("deleted slab retained its braces")
	}
	for p, af := range f.Arclms {
		if bytes.Equal(changed[p], deleted[p]) {
			t.Fatalf("%s: slab deletion missing from input", p)
		}
		for _, e := range af.Elems {
			if e.Num == a || e.Num == b {
				t.Fatalf("%s: deleted slab's brace %d still in analysis", p, e.Num)
			}
		}
	}
	t.Logf("identical L/X/Y inputs at 134 braces; SLAB %d opening updated ELEM %d/%d, deletion left 132 braces", panel.Num, a, b)
}
