package arclm

import (
	"bytes"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/yofu/st/matrix"
)

func TestGeometryStabilityColumnBuckling(t *testing.T) {
	euler := math.Pi * math.Pi / 4
	for _, tc := range []struct {
		name     string
		load     float64
		unstable bool
	}{{"below Euler", .8, false}, {"above Euler", 1.5, true}} {
		t.Run(tc.name, func(t *testing.T) {
			f := geometryBeam(16)
			var log bytes.Buffer
			f.Output = &log
			tip := f.Nodes[len(f.Nodes)-1]
			tip.Force[0], tip.Force[1] = -tc.load*euler, 1e-5
			if err := f.StaticAnalysis(nil, geometryCondition(t, 1, 1)); err != nil {
				t.Fatal(err)
			}
			expected := "GEOMETRY STABILITY: equilibrium FROM=0 LOAD=1 POSITIVE=48 ZERO=0 NEGATIVE=0"
			if tc.unstable {
				expected = "WARNING: GEOMETRY STABILITY: unstable equilibrium FROM=0 LOAD=1 POSITIVE=47 ZERO=0 NEGATIVE=1"
			}
			if !strings.Contains(log.String(), expected) || (!tc.unstable && strings.Contains(log.String(), "WARNING:")) {
				t.Fatalf("unexpected column stability at %.3g Pe:\n%s", tc.load, log.String())
			}
			if tc.unstable != (tip.Disp[1] < 0) {
				t.Fatalf("unexpected equilibrium deflection %g", tip.Disp[1])
			}
			geometryBalance(t, f, 2e-8)
			t.Log(expected)
		})
	}
}

func TestGeometryStabilityShallowTrussSnapThrough(t *testing.T) {
	const halfSpan, rise, ea = 1.0, .5, 1000.0
	initialLength := math.Hypot(halfSpan, rise)
	criticalLength := math.Cbrt(initialLength * halfSpan * halfSpan)
	criticalHeight := math.Sqrt(criticalLength*criticalLength - halfSpan*halfSpan)
	criticalLoad := 2 * ea * (initialLength - criticalLength) * criticalHeight / (initialLength * criticalLength)
	f := NewFrame()
	f.Lapch, f.Pivot = nil, nil
	var log bytes.Buffer
	f.Output = &log
	for i, xy := range [][2]float64{{-halfSpan, 0}, {0, rise}, {halfSpan, 0}} {
		n := NewNode()
		n.Num, n.Index = i+1, i
		copy(n.Coord, xy[:])
		for j := range n.Conf {
			n.Conf[j] = i != 1 || j != 1
		}
		f.Nodes = append(f.Nodes, n)
	}
	f.Nodes[1].Force[1] = -criticalLoad
	s := NewSect()
	s.Num, s.E, s.Poi = 1, ea, .3
	s.Value[0] = 1
	f.Sects = []*Sect{s}
	for i := 0; i < 2; i++ {
		e := NewElem()
		e.Num, e.Sect, e.Enod = i+1, s, f.Nodes[i:i+2]
		f.Elems = append(f.Elems, e)
	}
	c := geometryCondition(t, 1, .98)
	if err := f.StaticAnalysis(nil, c); err != nil {
		t.Fatal(err)
	}
	before := rise + f.Nodes[1].Disp[1]
	if before <= criticalHeight || strings.Contains(log.String(), "WARNING:") {
		t.Fatalf("expected stable upper branch, y=%g > ycrit=%g\n%s", before, criticalHeight, log.String())
	}
	log.Reset()
	c = geometryCondition(t, 1, .04)
	c.SetInit(false)
	c.SetMax(1.02)
	if err := f.StaticAnalysis(nil, c); err != nil {
		t.Fatalf("snap-through: %v\n%s", err, log.String())
	}
	after := rise + f.Nodes[1].Disp[1]
	if after >= -rise {
		t.Fatalf("expected the snapped branch, y=%g\n%s", after, log.String())
	}
	length := math.Hypot(halfSpan, after)
	nearNewton(t, "post-snap analytical force", 2*ea*(initialLength-length)*after/(initialLength*length), 1.02*criticalLoad, 1e-8)
	for _, message := range []string{"trial tangent is indefinite or singular", "UPDATE_FRACTION=0.25", "possible branch change, converged tangent is positive definite", "POSITIVE=0 ZERO=0 NEGATIVE=1", "POSITIVE=1 ZERO=0 NEGATIVE=0"} {
		if !strings.Contains(log.String(), message) {
			t.Fatalf("missing %q in snap-through log\n%s", message, log.String())
		}
	}
	t.Logf("critical load %.9g, apex height %.9g -> %.9g", criticalLoad, before, after)
}

func TestLinearStaticRelativeSingularityAtObliqueRigidMode(t *testing.T) {
	nonzeroPivot := false
	for _, solver := range []string{"LLS", "CG", "CRS", "PCG"} {
		for _, stiffness := range []float64{1e-9, 100, 1e9} {
			f := diagnosticCantilever()
			f.Pivot, f.Lapch = nil, nil
			tip := f.Nodes[1]
			tip.Num, tip.Coord[0], tip.Coord[1] = 47, 3, 4
			for j := range tip.Conf {
				tip.Conf[j] = j != 0 && j != 1
			}
			f.Elems[0].Sect.E = stiffness
			f.Elems[0].Sect.Value = []float64{1, 0, 0, 0}
			c := newtonTestCondition(t)
			c.SetSolver(solver)
			err := f.StaticAnalysis(nil, c)
			var pivot *matrix.SingularPivotError
			if !errors.As(err, &pivot) || pivot.Index != 1 || !strings.Contains(err.Error(), "NODE 47[1] (UY)") {
				t.Fatalf("%s E=%g: wrong singularity/DOF: %v", solver, stiffness, err)
			}
			nonzeroPivot = nonzeroPivot || pivot.Pivot != 0
		}
	}
	if !nonzeroPivot {
		t.Fatal("did not exercise a roundoff pivot")
	}
}

func TestLinearStaticSingularFirstFreeDOF(t *testing.T) {
	f := diagnosticCantilever()
	f.Pivot, f.Lapch = nil, nil
	tip := f.Nodes[1]
	tip.Num = 91
	for j := range tip.Conf {
		tip.Conf[j] = j != 2
	}
	f.Elems[0].Sect.Value = []float64{1, 0, 0, 0}
	err := f.StaticAnalysis(nil, newtonTestCondition(t))
	var pivot *matrix.SingularPivotError
	if !errors.As(err, &pivot) || pivot.Index != 0 || !strings.Contains(err.Error(), "NODE 91[2] (UZ)") {
		t.Fatalf("wrong first free DOF: %v", err)
	}
}

func TestGeometryTangentInertiaUsesSymmetricPart(t *testing.T) {
	rows := []map[int]float64{{1: 4}, {0: -2}, {}}
	got, err := geometryTangentInertia(rows)
	if err != nil {
		t.Fatal(err)
	}
	want := geometryInertia{positive: 1, zero: 1, negative: 1}
	if got != want {
		t.Fatalf("symmetric tangent inertia %+v, want %+v", got, want)
	}
}

func TestLinearStaticRelativePivotKeepsSoftSupport(t *testing.T) {
	f := diagnosticCantilever()
	f.Pivot, f.Lapch = nil, nil
	stiff, soft := f.Nodes[1], NewNode()
	soft.Num, soft.Index, soft.Coord[0] = 31, 2, 2
	f.Nodes = append(f.Nodes, soft)
	for _, n := range []*Node{stiff, soft} {
		for j := range n.Conf {
			n.Conf[j] = j != 0
		}
	}
	e := NewElem()
	e.Num, e.Enod, e.Sect = 2, []*Node{f.Nodes[0], soft}, f.Elems[0].Sect.Snapshot()
	e.Sect.E = 1e-14
	f.Elems = append(f.Elems, e)
	stiff.Force[0], soft.Force[0] = 100, e.Sect.E/2
	if err := f.StaticAnalysis(nil, newtonTestCondition(t)); err != nil {
		t.Fatal(err)
	}
	nearNewton(t, "stiff displacement", stiff.Disp[0], 1, 1e-12)
	nearNewton(t, "soft displacement", soft.Disp[0], 1, 1e-12)
}
