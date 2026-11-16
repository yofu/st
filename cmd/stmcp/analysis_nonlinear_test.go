package main

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"testing"

	"github.com/yofu/st/arclm"
	st "github.com/yofu/st/stlib"
)

func TestWorkerArclm101ConvergedCurveAndLimit(t *testing.T) {
	f := arclm.NewFrame()
	a, b := arclm.NewNode(), arclm.NewNode()
	a.Num, b.Num, b.Index, b.Coord[0] = 10, 20, 1, 1
	for i := range a.Conf {
		a.Conf[i], b.Conf[i] = true, i != 0
	}
	b.Force[0] = 100
	f.Nodes = []*arclm.Node{a, b}
	section := arclm.NewSect()
	section.Num, section.E, section.Poi, section.Exp, section.Exq = 1, 100, .3, 2, 2
	section.Value = []float64{1, 1, 1, 1}
	for i := range 6 {
		section.Yield[2*i], section.Yield[2*i+1] = 50, -50
	}
	member := arclm.NewElem()
	member.Num, member.Sect, member.Enod = 1, section, f.Nodes
	f.Elems = []*arclm.Elem{member}
	f.Sects = []*arclm.Sect{section}

	view := st.NewFrame()
	view.Nodes[10], view.Nodes[20] = st.NewNode(), st.NewNode()
	view.Nodes[10].Num, view.Nodes[20].Num = 10, 20
	view.Nodes[20].Coord[0] = 1
	view.AddLineElem(1, []*st.Node{view.Nodes[10], view.Nodes[20]}, view.AddSect(101), st.GIRDER)
	var published int
	var events bytes.Buffer
	curve, err := workerArclm101(f, t.TempDir(), &events, "after", analysisOptions{NodeIDs: []int{20}, Steps: 3, Increment: .2, Period: "X", PreviewEvery: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(curve.Points) != 2 || curve.Nonconvergence == nil || curve.Nonconvergence.LastConverged != .4 ||
		math.Abs(curve.Nonconvergence.Target-.6) > 1e-12 {
		t.Fatalf("unexpected convergence limit: %+v", curve)
	}
	for i, p := range curve.Points {
		if p.Step != i+1 || math.Abs(p.Factor-p.Displacements[0][0]) > 1e-10 {
			t.Fatalf("unexpected point: %+v", p)
		}
	}
	decoder := json.NewDecoder(&events)
	accepted := 0
	for {
		var event analysisEvent
		if err := decoder.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
		if s := event.Preview; s != nil {
			published++
			if s.Step != published || s.Period != "X" {
				t.Fatalf("unexpected preview step %d", s.Step)
			}
			if err := view.ApplyLiveAnalysisStep(s); err != nil {
				t.Fatal(err)
			}
			if got := view.Nodes[20].Disp[st.LiveAnalysisPeriod][0]; math.Abs(got-s.Factor) > 1e-10 {
				t.Fatalf("preview displacement: %g", got)
			}
			if got := view.Elems[1].N(st.LiveAnalysisPeriod, 0); math.Abs(math.Abs(got)-s.Factor*100) > 1e-8 {
				t.Fatalf("preview axial force: %g", got)
			}
		}
		if event.Step != nil {
			accepted++
			if event.Side != "after" || event.Step.Completed != accepted || event.Step.Total != 3 {
				t.Fatalf("unexpected progress: %+v", event.Step)
			}
		}
	}
	if published != 2 {
		t.Fatalf("expected first and last converged previews, got %d", published)
	}
	if accepted != len(curve.Points) {
		t.Fatalf("reported %d steps, expected %d", accepted, len(curve.Points))
	}
}
