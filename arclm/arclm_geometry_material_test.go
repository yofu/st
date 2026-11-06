package arclm

import (
	"math"
	"testing"
)

func geometryYield(s *Sect, cap float64) {
	s.Exp, s.Exq = 2, 2
	for j := 0; j < 6; j++ {
		s.Yield[2*j] = cap
		s.Yield[2*j+1] = -cap
	}
}

func TestCombinedPhysicalYieldAndTangent(t *testing.T) {
	for _, tc := range []struct {
		name string
		p    float64
		ends [2]bool
	}{{"both_p2", 2, [2]bool{}}, {"both_p1.5", 1.5, [2]bool{}}, {"start_p1.5", 1.5, [2]bool{false, true}}, {"elastic", 2, [2]bool{true, true}}} {
		t.Run(tc.name, func(t *testing.T) { combinedPhysicalYieldAndTangent(t, tc.p, tc.ends) })
	}
}

func combinedPhysicalYieldAndTangent(t *testing.T, exponent float64, ends [2]bool) {
	f := geometryBeam(1)
	f.Sects[0].Value[0] = 10
	geometryYield(f.Sects[0], .25)
	f.Sects[0].Exp, f.Sects[0].Exq = exponent, exponent
	f.Sects[0].ElasticEnds = ends
	e, err := newGeometryElement(f.Elems[0], 0)
	if err != nil {
		t.Fatal(err)
	}
	f.Nodes[1].Disp[0] = .04
	f.Nodes[1].Disp[1] = .13
	f.Nodes[1].Disp[2] = -.05
	rot := []geometryMatrix{geometryExp(geometryVector{.03, -.04, .06}), geometryExp(geometryVector{.15, .08, -.13})}
	m := &geometryMaterialTrial{history: &plasticMaterialState{}}
	m.history.plastic[5] = .005
	m.history.plastic[6] = .002
	m.fixed = [12]float64{.01, .02, -.03, .04, -.01, .02, -.02, .03, .02, -.02, .01, .04}
	r, err := e.responseMaterial(f.Nodes, rot, m)
	if err != nil {
		t.Fatal(err)
	}
	if ends != [2]bool{true, true} && r.material.Multipliers[0]+r.material.Multipliers[1] <= 0 {
		t.Fatal("fixture did not yield")
	}
	for end, y := range r.material.Yield {
		if y > RADIUS+1e-8 {
			t.Fatalf("end %d outside physical yield surface: %g", end, y)
		}
	}
	for block := 0; block < 4; block++ {
		p := geometryVector{}
		for j := 0; j < 3; j++ {
			p[j] = r.force[3*block+j] + m.fixed[3*block+j]
		}
		for j := 0; j < 3; j++ {
			got := geometryDot(r.axes[j], p)
			if math.Abs(got-r.material.Stress[3*block+j]) > 1e-9 {
				t.Fatalf("yielded physical force mismatch: %g vs %g", got, r.material.Stress[3*block+j])
			}
		}
	}
	h := 1e-6
	for j := 0; j < 12; j++ {
		n, d := j/6, j%6
		u := f.Nodes[n].Disp[d]
		old := rot[n]
		var sides [2]*geometryResponse
		for side, sign := range []float64{-1, 1} {
			if d < 3 {
				f.Nodes[n].Disp[d] = u + sign*h
			} else {
				v := geometryVector{}
				v[d-3] = sign * h
				rot[n] = geometryMM(geometryExp(v), old)
			}
			sides[side], err = e.responseMaterial(f.Nodes, rot, m)
			if err != nil {
				t.Fatal(err)
			}
		}
		f.Nodes[n].Disp[d] = u
		rot[n] = old
		for i := 0; i < 12; i++ {
			fd := (sides[1].force[i] - sides[0].force[i]) / (2 * h)
			if math.Abs(fd-r.tangent[i][j]) > 2e-6*math.Max(1, math.Abs(fd)) {
				t.Fatalf("combined tangent [%d,%d]: %g finite difference %g", i, j, r.tangent[i][j], fd)
			}
		}
	}
	if m.history.plastic[5] != .005 || m.history.plastic[6] != .002 {
		t.Fatal("trial modified committed history")
	}
	// Superposed rigid rotation must rotate forces, preserving local yield and
	// the plastic basic deformation, including rotated dead-load vectors.
	Q := geometryExp(geometryVector{.6, .3, -.8})
	for i, n := range f.Nodes {
		v := geometryVector{n.Coord[0] + n.Disp[0], n.Coord[1] + n.Disp[1], n.Coord[2] + n.Disp[2]}
		v = geometryMV(Q, v)
		for j := 0; j < 3; j++ {
			n.Disp[j] = v[j] - n.Coord[j]
		}
		rot[i] = geometryMM(Q, rot[i])
	}
	for block := 0; block < 4; block++ {
		v := geometryMV(Q, geometryVector{m.fixed[3*block], m.fixed[3*block+1], m.fixed[3*block+2]})
		copy(m.fixed[3*block:3*block+3], v[:])
	}
	r2, err := e.responseMaterial(f.Nodes, rot, m)
	if err != nil {
		t.Fatal(err)
	}
	for block := 0; block < 4; block++ {
		want := geometryMV(Q, geometryVector{r.force[3*block], r.force[3*block+1], r.force[3*block+2]})
		for j := 0; j < 3; j++ {
			if math.Abs(want[j]-r2.force[3*block+j]) > 2e-9 {
				t.Fatal("plastic response not objective")
			}
		}
	}
}

func TestCombinedInclinedBarsCycle(t *testing.T) {
	f := geometryBeam(1)
	a := 1 / math.Sqrt2
	f.Nodes[1].Coord = []float64{a, a, 0}
	for j := range f.Nodes[1].Conf {
		f.Nodes[1].Conf[j] = j != 0
	}
	s := f.Sects[0]
	s.Value = []float64{1, 0, 0, 0}
	geometryYield(s, .2)
	se := s.Snapshot()
	se.Value[0] = .5
	se.ElasticEnds = [2]bool{true, true}
	f.Sects = append(f.Sects, se)
	e := NewElem()
	e.Num = 2
	e.Enod = f.Elems[0].Enod
	e.Sect = se
	f.Elems = append(f.Elems, e)
	f.Nodes[1].Force[0] = .35
	c := geometryCondition(t, 20, .05)
	c.SetNlmaterial(true)
	if err := f.StaticAnalysis(nil, c); err != nil {
		t.Fatal(err)
	}
	lo, hi := a, 2.
	for k := 0; k < 100; k++ {
		x := (lo + hi) / 2
		l := math.Hypot(x, a)
		force := (RADIUS*.2 + .5*(l-1)) * x / l
		if force < .35 {
			lo = x
		} else {
			hi = x
		}
	}
	want := (lo+hi)/2 - a
	if math.Abs(f.Nodes[1].Disp[0]-want) > 1e-7 {
		t.Fatalf("combined axial/geometric reference: %g vs %g", f.Nodes[1].Disp[0], want)
	}
	geometryBalance(t, f, 2e-8)
	loaded := f.SaveState()
	plastic := f.Elems[0].material.plastic
	for _, el := range f.Elems {
		if !el.Sect.ElasticEnds[0] {
			k, _ := el.StiffMatrix()
			y, _, _, _, _ := el.Coefficients(k)
			if y[0] > RADIUS+1e-8 || y[1] > RADIUS+1e-8 {
				t.Fatalf("stored physical force outside yield: %v", y)
			}
		}
	}
	c.SetInit(false)
	c.SetDelta(-.05)
	c.SetMax(0)
	if err := f.StaticAnalysis(nil, c); err != nil {
		t.Fatal(err)
	}
	length := math.Hypot(a+want, a)
	plasticLength := length - 1 - RADIUS*.2
	residualLength := 1 + plasticLength/1.5
	wantResidual := math.Sqrt(residualLength*residualLength-a*a) - a
	if math.Abs(f.Nodes[1].Disp[0]-wantResidual) > 2e-7 {
		t.Fatalf("residual deformation: %g vs %g", f.Nodes[1].Disp[0], wantResidual)
	}
	for j, v := range plastic {
		if math.Abs(v-f.Elems[0].material.plastic[j]) > 2e-8 {
			t.Fatal("elastic unloading added plastic strain")
		}
	}
	c.SetDelta(.05)
	c.SetMax(1)
	if err := f.StaticAnalysis(nil, c); err != nil {
		t.Fatal(err)
	}
	if math.Abs(f.Nodes[1].Disp[0]-loaded.Disp[1][0]) > 2e-7 {
		t.Fatal("reload drift")
	}
	geometryBalance(t, f, 2e-8)
	// Failed steps must restore all material variables, flags and rotations.
	saved := f.SaveState()
	c.SetDelta(1)
	c.SetNlap(1)
	c.SetMax(2)
	c.SetNewtonOptions(NewtonOptions{MaxIterations: 1, DisableCutbacks: true})
	if err := f.StaticAnalysis(nil, c); err == nil {
		t.Fatal("fixture did not fail")
	}
	for i, n := range f.Nodes {
		for j, v := range n.Disp {
			if v != saved.Disp[i][j] {
				t.Fatal("failed step modified displacement")
			}
		}
	}
	for i, el := range f.Elems {
		if el.material.plastic != saved.plastic[i].material.plastic {
			t.Fatal("failed step modified plastic history")
		}
	}
}

func TestCombinedWeakBoundaryTangent(t *testing.T) {
	f := geometryBeam(1)
	s := f.Sects[0]
	s.Value = []float64{1, 0, 0, 0}
	geometryYield(s, .2)
	e, err := newGeometryElement(f.Elems[0], 0)
	if err != nil {
		t.Fatal(err)
	}
	f.Nodes[1].Disp[0] = RADIUS * .2
	rot := []geometryMatrix{geometryIdentity(), geometryIdentity()}
	for _, sign := range []float64{-1, 1} {
		d := [12]float64{}
		d[6] = sign
		m := &geometryMaterialTrial{direction: &d}
		r, err := e.responseMaterial(f.Nodes, rot, m)
		if err != nil {
			t.Fatal(err)
		}
		want := 1.
		if sign > 0 {
			want = 0
		}
		if math.Abs(r.tangent[6][6]-want) > 1e-8 {
			t.Fatalf("boundary sign=%g tangent=%g want=%g", sign, r.tangent[6][6], want)
		}
		if !r.material.directional {
			t.Fatal("yield boundary not recognized")
		}
	}
}

func TestCombinedElasticLimit(t *testing.T) {
	f, g := geometryBeam(8), geometryBeam(8)
	for _, frame := range []*Frame{f, g} {
		frame.Nodes[8].Force[1] = 3
		geometryYield(frame.Sects[0], 1e4)
	}
	c := geometryCondition(t, 10, .1)
	if err := f.StaticAnalysis(nil, c); err != nil {
		t.Fatal(err)
	}
	c = geometryCondition(t, 10, .1)
	c.SetNlmaterial(true)
	if err := g.StaticAnalysis(nil, c); err != nil {
		t.Fatal(err)
	}
	for i, n := range g.Nodes {
		for j, v := range n.Disp {
			if math.Abs(v-f.Nodes[i].Disp[j]) > 3e-8 {
				t.Fatal("combined elastic limit differs from geometric elastic solver")
			}
		}
	}
	geometryBalance(t, g, 3e-8)
}
