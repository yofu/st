package st

import (
	"testing"

	"github.com/yofu/st/arclm"
)

func TestEigenPreviewRestoresDisplayedResults(t *testing.T) {
	f := NewFrame()
	a, b := NewNode(), NewNode()
	a.Num, b.Num, b.Coord[0] = 1, 2, 1
	f.Nodes[a.Num], f.Nodes[b.Num] = a, b
	e := NewLineElem([]*Node{a, b}, NewSect(), GIRDER)
	e.Num, e.Lock = 1, true
	f.Elems[e.Num] = e
	a.Disp["L"] = []float64{7}
	e.Stress["L"] = map[int][]float64{a.Num: {8}}
	e.Phinge["L"] = map[int]bool{a.Num: true}
	e.Values["ENERGY"] = 9

	af := arclm.NewFrame()
	aa, ab := arclm.NewNode(), arclm.NewNode()
	aa.Num, ab.Num, aa.Disp[0] = a.Num, b.Num, 12
	ae := arclm.NewElem()
	ae.Num, ae.Enod, ae.Energy, ae.Energyb = e.Num, []*arclm.Node{aa, ab}, 13, 14
	af.Nodes, af.Elems = []*arclm.Node{aa, ab}, []*arclm.Elem{ae}

	restore := f.saveEigenPreview(af, "L")
	f.ReadArclmData(af, "L")
	if a.Disp["L"][0] != 12 || e.Values["ENERGY"] != 13 || e.Phinge["L"][a.Num] || e.Lock {
		t.Fatal("mode preview did not replace display data")
	}
	restore()
	if a.Disp["L"][0] != 7 || e.Stress["L"][a.Num][0] != 8 || !e.Phinge["L"][a.Num] || !e.Lock || e.Values["ENERGY"] != 9 {
		t.Fatal("previous display values were not restored")
	}
	if _, exists := a.Reaction["L"]; exists {
		t.Fatal("preview left an extra reaction entry")
	}
	if _, exists := e.Values["ENERGYB"]; exists {
		t.Fatal("preview left an extra geometric-energy entry")
	}
}
