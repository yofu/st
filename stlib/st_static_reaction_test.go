package st

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yofu/st/arclm"
)

func TestExtractArclmRegeneratesWeightFromCurrentInput(t *testing.T) {
	dir := t.TempDir()
	inpPath := filepath.Join(dir, "model.inp")
	wgtPath := filepath.Join(dir, "model.wgt")
	input := yieldEndInputFrame()
	input.Props[1].Name = "test"
	input.Ai.Nfloor, input.Ai.Boundary = 2, []float64{-1, .5, 2}
	input.Ai.BetaX, input.Ai.BetaY = []float64{1}, []float64{1}
	s, err := input.ParseSect(strings.Fields(yieldEndTestSection), false)
	if err != nil {
		t.Fatal(err)
	}
	a, b := input.AddNode(0, 0, 0), input.AddNode(0, 0, 1)
	a.Load[2] = -3
	a.ManualWeight = [3]float64{1, 2, 3}
	input.AddElem(-1, NewLineElem([]*Node{a, b}, s, COLUMN))
	data := append(input.InpBytes(), []byte(fmt.Sprintf(
		"#STMODEL NODE {\"ID\":%d,\"Weight\":[40,40,40],\"Locked\":true}\n"+
			"#STMODEL NODE {\"ID\":%d,\"Weight\":[40,40,40]}\n", a.Num, b.Num))...)

	f := NewFrame()
	if err := f.ReadInpSnapshot(data, inpPath); err != nil {
		t.Fatal(err)
	}
	saved := f.EditableInpBytes()
	if bytes.Contains(saved, []byte(`"Weight":`)) || !bytes.Contains(saved, []byte("MANUALWEIGHT 1 2 3")) {
		t.Fatal("INP should preserve only the manual weight")
	}
	reopened := NewFrame()
	if err := reopened.ReadInpSnapshot(saved, inpPath); err != nil {
		t.Fatal(err)
	}
	node := reopened.Nodes[a.Num]
	if node == nil || !node.Lock || node.Snapshot(reopened).ManualWeight != [3]float64{1, 2, 3} || node.Load[2] != -3 {
		t.Fatalf("INP round trip did not preserve node inputs: %+v", node)
	}

	for _, trial := range []struct {
		density float64
		readWgt bool
	}{{2, false}, {0, true}, {0, false}} {
		f.Props[1].hiju = trial.density
		if trial.readWgt {
			if err := f.ReadWgt(wgtPath); err != nil {
				t.Fatal(err)
			}
		}
		if err := f.ExtractArclm(wgtPath); err != nil {
			t.Fatal(err)
		}
		for _, n := range f.Arclms["L"].Nodes {
			wantForce, wantMass := -trial.density/2, trial.density/2
			if n.Num == a.Num {
				wantForce -= 3 + 2
				wantMass += 3
			}
			if n.Force[2] != wantForce || math.Abs(n.Mass-wantMass/9.80665) > 1e-12 {
				t.Errorf("density %g, node %d: Fz=%g, mass=%g, want %g and %g",
					trial.density, n.Num, n.Force[2], n.Mass, wantForce, wantMass/9.80665)
			}
		}
	}
}

func TestExtractAndReadInitialCMQStress(t *testing.T) {
	f := yieldEndInputFrame()
	f.Path = filepath.Join(t.TempDir(), "cmq.inp")
	f.Ai.Nfloor, f.Ai.Boundary = 2, []float64{-1, .5, 2}
	f.Ai.BetaX, f.Ai.BetaY = []float64{1}, []float64{1}
	s, err := f.ParseSect(strings.Fields(yieldEndTestSection), false)
	if err != nil {
		t.Fatal(err)
	}
	f.AddNode(0, 0, 0)
	a, b := f.AddNode(0, 0, 1), f.AddNode(6, 0, 1)
	e := f.AddLineElem(103, []*Node{a, b}, s, GIRDER)
	e.Bonds[5], e.Bonds[11] = Pin, Pin
	e.Cmq[1], e.Cmq[5], e.Cmq[7], e.Cmq[11] = 2.7, 3.6, 6.3, -5.4
	if err := f.ExtractArclm(Ce(f.Path, ".wgt")); err != nil {
		t.Fatal(err)
	}
	if err := f.SaveAsArclm(f.Path); err != nil {
		t.Fatal(err)
	}
	loaded := arclm.NewFrame()
	if err := loaded.ReadInput(f.DataFileName["L"]); err != nil {
		t.Fatal(err)
	}
	for _, af := range []*arclm.Frame{f.Arclms["L"], loaded} {
		el := af.Elems[0]
		for j, want := range map[int]float64{1: 3, 5: 0, 7: 6, 11: 0} {
			if math.Abs(el.Cmq[j]-e.Cmq[j]) > 1e-12 || math.Abs(el.Stress[j]-want) > 1e-12 {
				t.Fatalf("DOF %d: input CMQ %g, initial stress %g, want %g and %g", j, el.Cmq[j], el.Stress[j], e.Cmq[j], want)
			}
		}
	}
	e.Stress["L"] = map[int][]float64{a.Num: {0, 1.25, 0, 0, 0, 0}, b.Num: {0, 2.5, 0, 0, 0, 0}}
	if err := f.ExtractArclm(Ce(f.Path, ".wgt")); err != nil {
		t.Fatal(err)
	}
	stress := f.Arclms["L"].Elems[0].Stress
	if stress[1] != 1.25 || stress[7] != 2.5 {
		t.Fatalf("extraction replaced saved stresses: %v", stress)
	}
}

func TestStaticGravityReactionIncludesCMQAndNodalLoad(t *testing.T) {
	f := yieldEndInputFrame()
	dir := t.TempDir()
	f.Path = filepath.Join(dir, "model.inp")
	f.Ai.Nfloor, f.Ai.Boundary = 2, []float64{-1, .5, 2}
	f.Ai.BetaX, f.Ai.BetaY = []float64{1}, []float64{1}
	s, err := f.ParseSect(strings.Fields(yieldEndTestSection), false)
	if err != nil {
		t.Fatal(err)
	}
	ground, a, b := NewNode(), NewNode(), NewNode()
	ground.Num, a.Num, b.Num = 1, 2, 3
	a.Coord[2], b.Coord[2], b.Coord[0] = 1, 1, 6
	for _, n := range []*Node{ground, a, b} {
		for j := range n.Conf {
			n.Conf[j] = true
		}
		f.Nodes[n.Num] = n
	}
	a.Load[2] = -2
	e := NewLineElem([]*Node{a, b}, s, GIRDER)
	e.Num, e.Frame, e.Cmq[2] = 1, f, 1.5
	f.Elems[e.Num] = e

	if err := f.ExtractArclm(filepath.Join(dir, "model.wgt")); err != nil {
		t.Fatal(err)
	}
	af := f.Arclms["L"]
	af.Lapch, af.Pivot, af.Output = nil, nil, io.Discard
	c := arclm.NewAnalysisCondition()
	c.SetOutput([]string{filepath.Join(dir, "model.otl")})
	if err := af.StaticAnalysis(nil, c); err != nil {
		t.Fatal(err)
	}
	f.ReadArclmData(af, "L")
	halfWeight := s.Weight()[1] * e.Length() / 2
	for _, tc := range []struct {
		node *Node
		want float64
	}{{a, halfWeight + 1.5 + 2}, {b, halfWeight}} {
		if got := tc.node.ReturnReaction("L", 2); math.Abs(got-tc.want) > 1e-12 {
			t.Errorf("node %d Rz=%g, want %g", tc.node.Num, got, tc.want)
		}
	}
	if e.Cmq[2] != 1.5 {
		t.Errorf("extraction changed input CMQ: %g", e.Cmq[2])
	}
}
