package arclm

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestYieldEndSectionRoundTrip(t *testing.T) {
	for _, disabled := range [][2]bool{{}, {false, true}, {true, false}, {true, true}} {
		s := plasticTestFrame().Elems[0].Sect
		s.ElasticEnds = disabled
		text := s.InlString()
		if strings.Contains(text, "YIELDEND") != (disabled != [2]bool{}) {
			t.Fatalf("unexpected default serialization: %s", text)
		}
		got, err := ParseArclmSect(strings.Fields(text))
		if err != nil || got.ElasticEnds != disabled {
			t.Fatalf("round trip: %v %v", got, err)
		}
		if s.Snapshot().ElasticEnds != disabled {
			t.Fatal("snapshot lost end selection")
		}
	}
	base := plasticTestFrame().Elems[0].Sect.InlString()
	for _, suffix := range []string{"YIELDEND", "YIELDEND 1", "YIELDEND 2 0", "YIELDEND 1 x", "YIELDEND -1 0"} {
		if _, err := ParseArclmSect(strings.Fields(base + suffix)); err == nil {
			t.Fatalf("accepted malformed option %q", suffix)
		}
	}
}

func TestYieldEndSinglePlasticBending(t *testing.T) {
	for enabled := 0; enabled < 2; enabled++ {
		f := plasticTestFrame()
		e := f.Elems[0]
		e.Sect.Value[2] = 1
		e.Sect.ElasticEnds[1-enabled] = true
		// Isolate bending; the enabled end's bending capacity is 95.
		for j := 0; j < 5; j++ {
			e.Sect.Yield[2*j], e.Sect.Yield[2*j+1] = 1e12, -1e12
		}
		u := make([]float64, 12)
		u[5], u[11] = .8, -.8
		r := checkPlasticResponse(t, e, u)
		sign := 1.0 - 2*float64(enabled)
		nearNewton(t, "yielding moment", r.Stress[6*enabled+5], sign*95, 1e-8)
		nearNewton(t, "elastic end moment", r.Stress[6*(1-enabled)+5], -sign*192.5, 1e-8)
		if r.Multipliers[enabled] <= 0 || r.Multipliers[1-enabled] != 0 || r.Yield[1-enabled] != 0 {
			t.Fatalf("incorrect active ends: %v", r)
		}
		for j := 0; j < 6; j++ {
			nearNewton(t, "elastic end plastic increment", r.PlasticIncrement[6*(1-enabled)+j], 0, 0)
		}
		e.commitPlasticResponse(r)
		if !e.Phinge[enabled] || e.Phinge[1-enabled] {
			t.Fatalf("incorrect hinge flags: %v", e.Phinge)
		}
		for j := 0; j < 6; j++ {
			if e.Bonds[6*(1-enabled)+j].Num != 0 {
				t.Fatal("elastic end connection changed")
			}
		}
		// Unloading must restore the elastic material tangent without adding
		// plastic history at either end.
		for i := range u {
			u[i] *= -.1
		}
		r = checkPlasticResponse(t, e, u)
		if r.Multipliers != [2]float64{} {
			t.Fatal("unloading produced plastic flow")
		}
		t.Logf("enabled end=%d: yielding M=%g, opposite elastic M=%g", enabled+1, sign*95, -sign*192.5)
	}
}

func TestYieldEndSingleEndTangent(t *testing.T) {
	for _, disabled := range [][2]bool{{false, true}, {true, false}} {
		for _, pq := range [][2]float64{{1.5, 1.5}, {2, 2}, {2, 3}, {3, 2}} {
			e := plasticTestFrame().Elems[0]
			e.Sect.ElasticEnds = disabled
			e.Sect.Exp, e.Sect.Exq = pq[0], pq[1]
			for _, j := range []int{1, 2} {
				e.Sect.Yield[2*j], e.Sect.Yield[2*j+1] = 1000, -1000
			}
			u := make([]float64, 12)
			u[6], u[7], u[8], u[9], u[10], u[11] = 1.2, .12, -.08, .4, .1, -.15
			r := checkPlasticResponse(t, e, u)
			if r.Multipliers[0]+r.Multipliers[1] <= 0 {
				t.Fatal("finite difference case did not yield")
			}
			for j := 0; j < 12; j++ {
				const h = 1e-6
				plus, minus := append([]float64(nil), u...), append([]float64(nil), u...)
				plus[j] += h
				minus[j] -= h
				rp, rm := checkPlasticResponse(t, e, plus), checkPlasticResponse(t, e, minus)
				for i := 0; i < 12; i++ {
					fd := (rp.Stress[i] - rm.Stress[i]) / (2 * h)
					if math.Abs(fd-r.Tangent[i][j]) > 2e-6*math.Max(1, math.Abs(fd)) || math.Abs(r.Tangent[i][j]-r.Tangent[j][i]) > 1e-7 {
						t.Fatalf("disabled=%v p,q=%v tangent[%d][%d]=%g, FD=%g", disabled, pq, i, j, r.Tangent[i][j], fd)
					}
				}
			}
		}
	}
}

func TestYieldEndBothElasticWithoutCapacities(t *testing.T) {
	f := plasticTestFrame()
	f.Lapch, f.Pivot = nil, nil
	e := f.Elems[0]
	e.Sect.ElasticEnds = [2]bool{true, true}
	e.Sect.Yield = make([]float64, 12)
	e.Sect.Exp, e.Sect.Exq = 0, 0
	if err := f.Arclm101(nil, newtonTestCondition(t)); err != nil {
		t.Fatal(err)
	}
	nearNewton(t, "elastic force above old capacity", e.Stress[6], 120, 1e-8)
	nearNewton(t, "elastic displacement", f.Nodes[1].Disp[0], 1.2, 1e-8)
	if e.Phinge[0] || e.Phinge[1] || e.material.plastic != [12]float64{} {
		t.Fatal("elastic member developed plastic history")
	}
}

func TestYieldEndNewtonRedistributionAndRestore(t *testing.T) {
	for _, solver := range []string{"LLS", "LDLT"} {
		f := newtonParallelFrame()
		e := f.Elems[0]
		e.Sect.ElasticEnds = [2]bool{false, true}
		c := newtonTestCondition(t)
		c.SetSolver(solver)
		c.SetNlap(10)
		c.SetDelta(.1)
		if err := f.Arclm101(nil, c); err != nil {
			t.Fatal(err)
		}
		nearNewton(t, "plastic force", e.Stress[6], 95, 1e-8)
		nearNewton(t, "parallel elastic force", f.Elems[1].Stress[6], 125, 1e-8)
		nearNewton(t, "equilibrium reaction", f.Nodes[0].Reaction[0], -227, 1e-8)
		if !e.Phinge[0] || e.Phinge[1] {
			t.Fatalf("incorrect final end state: %v", e.Phinge)
		}
		for i := 6; i < 12; i++ {
			nearNewton(t, "elastic end history", e.material.plastic[i], 0, 0)
		}
		saved := f.SaveState()
		f.Initialise()
		if e.Sect.ElasticEnds != [2]bool{false, true} {
			t.Fatal("initialization lost end selection")
		}
		f.RestoreState(saved)
		if !reflect.DeepEqual(f.SaveState(), saved) {
			t.Fatal("restore changed single-end plastic history")
		}
	}
}
