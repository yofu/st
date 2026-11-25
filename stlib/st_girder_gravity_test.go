package st

import (
	"bytes"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yofu/st/arclm"
)

func solveGravityTest(t *testing.T, af *arclm.Frame) {
	t.Helper()
	af.Lapch, af.Pivot, af.Output = nil, nil, io.Discard
	c := arclm.NewAnalysisCondition()
	c.SetOutput([]string{filepath.Join(t.TempDir(), "static.otl")})
	if err := af.StaticAnalysis(nil, c); err != nil {
		t.Fatal(err)
	}
}

func TestGirderGravityEndActions(t *testing.T) {
	for _, tc := range []struct {
		name              string
		dx, dy, dz, angle float64
		triangle, pinned  bool
	}{
		{"horizontal", 6, 0, 0, 0, false, false},
		{"inclined and rolled", 3, 4, 2, .73, false, false},
		{"near vertical", .1, .1, 4, 1.2, false, false},
		{"vertical", 0, 0, 4, .7, false, false},
		{"triangular fixed", 6, 0, 0, 0, true, false},
		{"triangular pinned", 6, 0, 0, 0, true, true},
		{"triangular inclined and rolled", 3, 4, 2, .73, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, reverse := range []bool{false, true} {
				f := yieldEndInputFrame()
				f.Path = filepath.Join(t.TempDir(), "beam.inp")
				f.Ai.Nfloor, f.Ai.Boundary = 2, []float64{-1, .5, 10}
				f.Ai.BetaX, f.Ai.BetaY = []float64{1}, []float64{1}
				s, err := f.ParseSect(strings.Fields(yieldEndTestSection), false)
				if err != nil {
					t.Fatal(err)
				}
				f.AddNode(0, 0, 0)
				a, b := f.AddNode(0, 0, 1), f.AddNode(tc.dx, tc.dy, 1+tc.dz)
				for _, n := range f.Nodes {
					for j := range n.Conf {
						n.Conf[j] = true
					}
				}
				nodes := []*Node{a, b}
				if reverse {
					nodes[0], nodes[1] = b, a
				}
				e := f.AddLineElem(-1, nodes, s, GIRDER)
				e.Cang = tc.angle
				if tc.pinned {
					for _, j := range []int{4, 5, 10, 11} {
						e.Bonds[j] = Pin
					}
				}
				length := e.Length()
				// A unit line weight, or a ramp from 0 at a to 1 at b.
				axial := [2]float64{length / 2, length / 2}
				shear := axial
				moment := [2]float64{-length * length / 12, length * length / 12}
				if tc.triangle {
					f.Props[1].hiju = 0
					load, err := RightAngledTriangle(1, length, 0, length, 0, CmqEps)
					if err != nil {
						t.Fatal(err)
					}
					if err := e.DistributeGirder([]*Cmq{load, load, load}, reverse); err != nil {
						t.Fatal(err)
					}
					axial = [2]float64{length / 6, length / 3}
					shear = [2]float64{3 * length / 20, 7 * length / 20}
					moment = [2]float64{-length * length / 30, length * length / 20}
					if tc.pinned {
						shear, moment = axial, [2]float64{}
					}
				}
				if err := f.ExtractArclm(Ce(f.Path, ".wgt")); err != nil {
					t.Fatal(err)
				}
				af := f.Arclms["L"]
				solveGravityTest(t, af)
				dir := [3]float64{tc.dx / length, tc.dy / length, tc.dz / length}
				// Independent global statics: axial projection along the chord,
				// transverse projection perpendicular to it, moments about Z cross chord.
				for _, n := range af.Nodes {
					if n.Num != a.Num && n.Num != b.Num {
						continue
					}
					i := 0
					if n.Num == b.Num {
						i = 1
					}
					var want [6]float64
					for j := 0; j < 3; j++ {
						want[j] = (axial[i] - shear[i]) * dir[2] * dir[j]
					}
					want[2] += shear[i]
					want[3], want[4] = -dir[1]*moment[i], dir[0]*moment[i]
					for j, w := range want {
						if got := n.Reaction[j]; !(math.Abs(got-w) < 1e-10) {
							t.Errorf("reverse=%t node=%d component=%d: got %g want %g", reverse, n.Num, j, got, w)
						}
					}
				}
			}
		})
	}
}

func TestSlabGravityOnActiveAndSkippedGirder(t *testing.T) {
	f := yieldEndInputFrame()
	f.Path = filepath.Join(t.TempDir(), "roof.inp")
	f.Ai.Nfloor, f.Ai.Boundary = 2, []float64{-1, .5, 10}
	f.Ai.BetaX, f.Ai.BetaY = []float64{1}, []float64{1}
	beam, err := f.ParseSect(strings.Fields(yieldEndTestSection), false)
	if err != nil {
		t.Fatal(err)
	}
	panel := f.AddSect(711)
	fig := NewFig()
	fig.Prop = &Prop{} // gravity-only panel
	panel.Figs = []*Fig{fig}
	panel.Lload = []float64{1.2, 1.2, 1.2}
	f.AddNode(0, 0, 0)
	a, b := f.AddNode(0, 0, 1), f.AddNode(3, 0, 5)
	f.AddPlateElem(-1, []*Node{a, b, f.AddNode(3, 5, 5), f.AddNode(0, 5, 1)}, panel, SLAB)
	e := f.AddLineElem(-1, []*Node{b, a}, beam, GIRDER)
	e.Cang = .7
	a.ManualWeight = [3]float64{1, 2, 3}
	a.Load[2] = -4
	for _, n := range f.Nodes {
		for j := range n.Conf {
			n.Conf[j] = true
		}
	}
	// 5x5 panel: 7.5 tf per corner. Half of that corner load reaches
	// the sloping edge, plus 2.5 tf at each end from the girder itself.
	for _, skip := range []bool{false, true, false} {
		e.Skip[0] = skip
		if err := f.ExtractArclm(Ce(f.Path, ".wgt")); err != nil {
			t.Fatal(err)
		}
		af := f.Arclms["L"]
		for _, n := range af.Nodes {
			if n.Num != a.Num && n.Num != b.Num {
				continue
			}
			weight, force := 10.0, -3.75
			if skip {
				force -= 6.25
			}
			if n.Num == a.Num {
				weight += 3
				force -= 6
			}
			if !(math.Abs(n.Mass*9.80665-weight) < 1e-10 && math.Abs(n.Force[2]-force) < 1e-10) {
				t.Fatalf("skip=%t node=%d: mass=%g Fz=%g, want %g and %g", skip, n.Num, n.Mass, n.Force[2], weight/9.80665, force)
			}
		}
		solveGravityTest(t, af)
		var r [3]float64
		for _, n := range af.Nodes {
			for j := range r {
				r[j] += n.Reaction[j]
			}
		}
		if !(math.Abs(r[0]) < 1e-10 && math.Abs(r[1]) < 1e-10 && math.Abs(r[2]-41) < 1e-10) {
			t.Fatalf("skip=%t: support reactions %v, want [0 0 41]", skip, r)
		}
	}
}

func TestSetouchiSlabGravity(t *testing.T) {
	path := os.Getenv("ST_SETOUCHI_TEST_INP")
	if path == "" {
		t.Skip("set ST_SETOUCHI_TEST_INP to setouchi01.inp")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	f := NewFrame()
	dir := t.TempDir()
	if err = f.ReadInpSnapshot(data, filepath.Join(dir, "model.inp")); err != nil {
		t.Fatal(err)
	}
	roof := make(map[int]bool)
	for _, e := range f.Elems {
		if e.Sect.Num == 711 && !e.IsLineElem() {
			e.Etype = SLAB
			for _, n := range e.Enod {
				roof[n.Num] = true
			}
		}
	}
	if len(roof) != 163 {
		t.Fatalf("unexpected section 711 roof node count %d", len(roof))
	}
	first := extractBraceTestInputs(t, f)
	if err = f.WeightDistribution(Ce(f.Path, ".wgt")); err != nil {
		t.Fatal(err)
	}
	second := extractBraceTestInputs(t, f)
	for _, p := range []string{"L", "X", "Y"} {
		if !bytes.Equal(first[p], second[p]) {
			t.Fatalf("%s input changed on repeated extraction", p)
		}
		af := f.Arclms[p]
		solveGravityTest(t, af)
		var rx, ry, rz, ux372, maxH float64
		for _, n := range af.Nodes {
			rx += n.Reaction[0]
			ry += n.Reaction[1]
			rz += n.Reaction[2]
			if roof[n.Num] {
				maxH = math.Max(maxH, 1000*math.Hypot(n.Disp[0], n.Disp[1]))
			}
			if n.Num == 372 {
				ux372 = 1000 * n.Disp[0]
			}
		}
		t.Logf("%s: reactions=(%.8f, %.8f, %.8f) tf, node 372 Ux=%.5f mm, max roof horizontal=%.5f mm", p, rx, ry, rz, ux372, maxH)
		switch p {
		case "L":
			if !(math.Abs(rx) < 1e-8 && math.Abs(ry) < 1e-8 && math.Abs(rz-49.627) < .0005) {
				t.Fatal("gravity resultant does not balance the reported vertical weight")
			}
			if !(ux372 > 2 && ux372 < 4 && maxH < 11) {
				t.Fatal("unexpected lateral roof displacement under gravity")
			}
		case "X", "Y":
			// Reference maxima from the same INP before changing gravity generation.
			want := map[string]float64{"X": 42.12079773, "Y": 29.20237243}
			if !(math.Abs(maxH-want[p]) < 1e-5) {
				t.Fatal("lateral load case changed")
			}
		}
	}
}
