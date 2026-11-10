package arclm

import (
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestLinearStaticLAPIndependence(t *testing.T) {
	for _, count := range []int{1, 2, 4, 10} {
		f := diagnosticCantilever()
		f.Lapch, f.Pivot = nil, nil
		f.Nodes[0].Force[0] = 3
		f.Elems[0].Cmq[0], f.Elems[0].Cmq[6] = -5, -5
		c := newtonTestCondition(t)
		c.SetNlap(count)
		c.SetDelta(1 / float64(count))
		if err := f.StaticAnalysis(nil, c); err != nil {
			t.Fatal(err)
		}
		nearNewton(t, "total displacement", f.Nodes[1].Disp[0], .04, 1e-12)
		nearNewton(t, "support reaction", f.Nodes[0].Reaction[0], -12, 1e-12)
		nearNewton(t, "support end force", f.Elems[0].Stress[0], -9, 1e-12)
		nearNewton(t, "free end force", f.Elems[0].Stress[6], -1, 1e-12)
		nearNewton(t, "original CMQ", f.Elems[0].Cmq[6], -5, 0)
	}
}

func TestLinearStaticContinuationAndExtraLoad(t *testing.T) {
	f := diagnosticCantilever()
	f.Lapch, f.Pivot = nil, nil
	c := newtonTestCondition(t)
	c.SetDelta(.25)
	c.SetMax(.5)
	if err := f.StaticAnalysis(nil, c); err != nil {
		t.Fatal(err)
	}
	c = newtonTestCondition(t)
	c.SetInit(false)
	c.SetDelta(.25)
	if err := f.StaticAnalysis(nil, c); err != nil {
		t.Fatal(err)
	}
	nearNewton(t, "continued displacement", f.Nodes[1].Disp[0], -.01, 1e-12)

	extra := diagnosticCantilever()
	extra.Nodes[1].Force[0] = -2
	extra.Nodes[0].Force[0] = 4
	c = newtonTestCondition(t)
	c.SetNlap(2)
	c.SetDelta(.5)
	c.SetExtraFrames(extra)
	f.Lapch = make(chan int)
	done := make(chan error)
	go func() { done <- f.StaticAnalysis(nil, c) }()
	var cases []int
running:
	for {
		select {
		case <-f.Lapch:
			cases = append(cases, f.StaticResultCase())
			f.Lapch <- 0
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
			break running
		}
	}
	if !slices.Equal(cases, []int{0, 0, 1}) {
		t.Fatalf("published load cases: %v", cases)
	}
	if len(c.Output()) != 2 {
		t.Fatalf("outputs: %v", c.Output())
	}
	b, err := os.ReadFile(c.Output()[1])
	if err != nil || !strings.Contains(string(b), "-0.020000") || !strings.Contains(string(b), "-2.000000") {
		t.Fatalf("extra load result: %v\n%s", err, b)
	}
	nearNewton(t, "primary load kept for continuation", f.Nodes[1].Disp[0], -.01, 1e-12)
}

func TestLinearStaticIterativeSolversWithUnloadedCase(t *testing.T) {
	for _, solver := range []string{"CG", "CRS", "PCG"} {
		t.Run(solver, func(t *testing.T) {
			f, extra := diagnosticCantilever(), diagnosticCantilever()
			f.Lapch, f.Pivot = nil, nil
			clear(extra.Nodes[1].Force)
			c := newtonTestCondition(t)
			c.SetSolver(solver)
			c.SetExtraFrames(extra)
			if err := f.StaticAnalysis(nil, c); err != nil {
				t.Fatal(err)
			}
			nearNewton(t, "primary case displacement", f.Nodes[1].Disp[0], -.01, 1e-12)
		})
	}
}

func TestStaticBeamConnectionEquilibrium(t *testing.T) {
	const length, ei = 6.0, 4100.0
	const shearA, shearB, momentA, momentB = 2.7, 6.3, 3.6, -5.4
	for _, conn := range []struct {
		name string
		a, b float64
	}{
		{"both pinned", 0, 0},
		{"equal springs", 3000, 3000},
		{"unequal springs", 3000, 1000},
	} {
		for _, solver := range []string{"linear", "material", "geometry"} {
			t.Run(conn.name+"/"+solver, func(t *testing.T) {
				f := diagnosticCantilever()
				f.Lapch, f.Pivot = nil, nil
				for _, n := range f.Nodes {
					for i := range n.Conf {
						n.Conf[i], n.Force[i] = true, 0
					}
				}
				f.Nodes[1].Coord[0] = length
				e := f.Elems[0]
				e.Sect.E = ei
				e.Sect.ElasticEnds = [2]bool{true, true}
				e.Cmq[1], e.Cmq[7], e.Cmq[5], e.Cmq[11] = shearA, shearB, momentA, momentB
				for i, spring := range []float64{conn.a, conn.b} {
					bond := NewSect()
					bond.Num = 1
					if spring > 0 {
						bond.Num, bond.Value[2] = 2, spring
					}
					e.Bonds[6*i+5] = bond
				}
				c := newtonTestCondition(t)
				c.SetNlmaterial(solver == "material")
				c.SetNlgeometry(solver == "geometry")
				if err := f.StaticAnalysis(nil, c); err != nil {
					t.Fatal(err)
				}
				// Solve the independent two-rotation equilibrium equations,
				// then obtain shears by total force and moment balance.
				aa, bb, ab := 4*ei/length+conn.a, 4*ei/length+conn.b, 2*ei/length
				det := aa*bb - ab*ab
				ma := conn.a * (bb*momentA - ab*momentB) / det
				mb := conn.b * (aa*momentB - ab*momentA) / det
				vb := (length*shearB + momentA + momentB - ma - mb) / length
				va := shearA + shearB - vb
				for _, want := range []struct {
					node, dof int
					value     float64
				}{{0, 1, va}, {1, 1, vb}, {0, 5, ma}, {1, 5, mb}} {
					nearNewton(t, "member end force", e.Stress[6*want.node+want.dof], want.value, 1e-9)
					nearNewton(t, "support reaction", f.Nodes[want.node].Reaction[want.dof], want.value, 1e-9)
				}
				nearNewton(t, "original CMQ", e.Cmq[5], momentA, 0)
				if _, err := os.Stat(c.Output()[0]); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestStaticAnalysisMaterialFlag(t *testing.T) {
	for _, delta := range []float64{1, .25} {
		f := newtonParallelFrame()
		c := newtonTestCondition(t)
		c.SetNlmaterial(true)
		c.SetDelta(delta)
		if err := f.StaticAnalysis(nil, c); err != nil {
			t.Fatal(err)
		}
		nearNewton(t, "plastic redistribution", f.Elems[0].Stress[6], 95, 1e-8)
		nearNewton(t, "displacement", f.Nodes[1].Disp[0], 1.25, 1e-8)
		if _, err := os.Stat(c.Output()[0]); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLinearStaticPOST(t *testing.T) {
	t.Run("floor contact and release", func(t *testing.T) {
		f := diagnosticCantilever()
		f.Lapch, f.Pivot = nil, nil
		f.Nodes[1].Coord[0], f.Nodes[1].Coord[2] = 0, 1
		for i := range f.Nodes[1].Conf {
			f.Nodes[1].Conf[i], f.Nodes[1].Force[i] = i != 2, 0
		}
		f.Nodes[1].Force[2] = 30
		c := newtonTestCondition(t)
		c.SetStart(-2)
		c.SetFloorContact(.9)
		contact := false
		c.SetPostprocess(func(f *Frame, _ [][]float64, _, _ []float64) (float64, bool) {
			if f.Nodes[1].Reaction[2] > 0 {
				contact = true
				nearNewton(t, "floor elevation", f.Nodes[1].Coord[2]+f.Nodes[1].Disp[2], .9, 1e-12)
				nearNewton(t, "floor reaction", f.Nodes[1].Reaction[2], 20, 1e-12)
				nearNewton(t, "original support reaction", f.Nodes[0].Reaction[2], 10, 1e-12)
			}
			return c.Delta(), true
		})
		if err := f.StaticAnalysis(nil, c); err != nil {
			t.Fatal(err)
		}
		if !contact || f.Nodes[1].Conf[2] || !f.Nodes[0].Conf[2] {
			t.Fatal("floor contact did not activate and release during load reversal")
		}
		nearNewton(t, "free upward displacement", f.Nodes[1].Disp[2], .3, 1e-12)
		nearNewton(t, "no floor reaction", f.Nodes[1].Reaction[2], 0, 0)
	})
	t.Run("compression member deactivation and reactivation", func(t *testing.T) {
		f := newtonParallelFrame()
		f.Elems[0].Sect.Num, f.Elems[1].Sect.Num = 1, 2
		f.Nodes[1].Force[0] = 20
		c := newtonTestCondition(t)
		c.SetStart(-2)
		c.SetCompressionLimit(0, []int{1})
		inactive := false
		c.SetPostprocess(func(f *Frame, _ [][]float64, _, _ []float64) (float64, bool) {
			if f.Nodes[1].Disp[0] < -.15 {
				inactive = true
				nearNewton(t, "compression displacement", f.Nodes[1].Disp[0], -.2, 1e-12)
				nearNewton(t, "inactive member force", f.Elems[0].Stress[0], 0, 0)
			}
			return c.Delta(), true
		})
		if err := f.StaticAnalysis(nil, c); err != nil {
			t.Fatal(err)
		}
		if !inactive || !f.Elems[0].IsValid {
			t.Fatal("member did not deactivate and reactivate during load reversal")
		}
		nearNewton(t, "tension displacement", f.Nodes[1].Disp[0], .1, 1e-12)
		nearNewton(t, "reactivated member force", f.Elems[0].Stress[6], 10, 1e-12)
	})
	t.Run("callback runs before the final commit", func(t *testing.T) {
		f := diagnosticCantilever()
		f.Lapch, f.Pivot = nil, nil
		c := newtonTestCondition(t)
		calls := 0
		c.SetPostprocess(func(f *Frame, _ [][]float64, _, _ []float64) (float64, bool) {
			calls++
			f.Nodes[1].Conf[0] = true
			return c.Delta(), true
		})
		if err := f.StaticAnalysis(nil, c); err != nil {
			t.Fatal(err)
		}
		if calls != 2 || math.Abs(f.Nodes[1].Reaction[0]-1) > 1e-12 || f.Nodes[1].Disp[0] != 0 {
			t.Fatalf("calls=%d, tip=%+v", calls, f.Nodes[1])
		}
	})
	t.Run("POST cannot be combined with material analysis", func(t *testing.T) {
		f := diagnosticCantilever()
		c := newtonTestCondition(t)
		c.SetNlmaterial(true)
		c.SetFloorContact(0)
		c.SetOutput([]string{filepath.Join(t.TempDir(), "invalid.otl")})
		if err := f.StaticAnalysis(nil, c); err == nil {
			t.Fatal("unsupported POST and NLMAT combination was accepted")
		}
	})
}
