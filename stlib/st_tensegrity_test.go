package st

import (
	"reflect"
	"testing"
)

func TestTensegrityDeleteEqualAndMissingForces(t *testing.T) {
	f := NewFrame()
	sec := f.AddSect(211)
	n := f.AddNode(0, 0, 0)
	for i, id := range []int{2003, 2001, 2002} {
		el := f.AddLineElem(id, []*Node{n, f.AddNode(float64(i+1), 0, 0)}, sec, COLUMN)
		for _, p := range []string{"L", "X", "Y"} {
			el.Stress[p] = map[int][]float64{n.Num: {2}}
		}
	}
	delete(f.Elems[2003].Stress, "Y")
	if els, err := TensegrityDeleteCandidates(f, []*Node{n}, 211, 1); err == nil || len(els) != 0 {
		t.Fatalf("missing Y force must stop deletion: %v, %v", els, err)
	}
	f.Elems[2003].Stress["Y"] = map[int][]float64{n.Num: {2}}
	for i := 0; i < 20; i++ {
		els, err := TensegrityDeleteCandidates(f, []*Node{n}, 211, 1)
		if err != nil {
			t.Fatal(err)
		}
		ids := make([]int, len(els))
		for i, el := range els {
			ids[i] = el.Num
		}
		if !reflect.DeepEqual(ids, []int{2001, 2002}) {
			t.Fatalf("tied candidates %v", ids)
		}
	}
}
