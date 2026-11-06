package arclm

import (
	"context"
	"io"
	"math"
	"path/filepath"
	"testing"
)

func geometryBeam(n int) *Frame {
	f := NewFrame()
	f.Output = io.Discard
	f.Lapch = nil
	f.Pivot = nil
	s := NewSect()
	s.Num = 1
	s.E = 1
	s.Poi = .3
	s.Value = []float64{1e6, 1, 1, 1}
	f.Sects = []*Sect{s}
	for i := 0; i <= n; i++ {
		v := NewNode()
		v.Index = i
		v.Num = i + 1
		v.Coord[0] = float64(i) / float64(n)
		for j := 0; j < 6; j++ {
			v.Conf[j] = i == 0 || j == 2 || j == 3 || j == 4
		}
		f.Nodes = append(f.Nodes, v)
	}
	for i := 0; i < n; i++ {
		e := NewElem()
		e.Num = i + 1
		e.Sect = s
		e.Enod = []*Node{f.Nodes[i], f.Nodes[i+1]}
		f.Elems = append(f.Elems, e)
	}
	return f
}
func geometryCondition(t *testing.T, steps int, delta float64) *AnalysisCondition {
	t.Helper()
	c := NewAnalysisCondition()
	c.SetNlgeometry(true)
	c.SetNlap(steps)
	c.SetDelta(delta)
	c.SetOutput([]string{filepath.Join(t.TempDir(), "geometry.otp")})
	return c
}
func geometryBalance(t *testing.T, f *Frame, tol float64) {
	t.Helper()
	sum, mom := geometryVector{}, geometryVector{}
	internal := make([]float64, 6*len(f.Nodes))
	for _, e := range f.Elems {
		var ef, em geometryVector
		for n := 0; n < 2; n++ {
			p := geometryMV(*e.geometryAxes, geometryVector{e.Stress[6*n], e.Stress[6*n+1], e.Stress[6*n+2]})
			m := geometryMV(*e.geometryAxes, geometryVector{e.Stress[6*n+3], e.Stress[6*n+4], e.Stress[6*n+5]})
			x := geometryVector{}
			for j := 0; j < 3; j++ {
				x[j] = e.Enod[n].Coord[j] + e.Enod[n].Disp[j]
			}
			arm := geometryCross(x, p)
			for j := 0; j < 3; j++ {
				internal[6*e.Enod[n].Index+j] += p[j]
				internal[6*e.Enod[n].Index+3+j] += m[j]
				ef[j] += p[j]
				em[j] += m[j] + arm[j]
			}
		}
		for j := 0; j < 3; j++ {
			if math.Abs(ef[j]) > tol || math.Abs(em[j]) > tol {
				t.Fatalf("ELEM %d not equilibrated: %v %v", e.Num, ef, em)
			}
		}
	}
	for i, n := range f.Nodes {
		x := geometryVector{}
		p, m := geometryVector{}, geometryVector{}
		for j := 0; j < 3; j++ {
			x[j] = n.Coord[j] + n.Disp[j]
			p[j] = n.Force[j] + n.Reaction[j]
			m[j] = n.Force[3+j] + n.Reaction[3+j]
		}
		arm := geometryCross(x, p)
		for j := 0; j < 3; j++ {
			sum[j] += p[j]
			mom[j] += m[j] + arm[j]
		}
		for j := 0; j < 6; j++ {
			if math.Abs(internal[6*i+j]-n.Force[j]-n.Reaction[j]) > tol {
				t.Fatalf("NODE %d DOF %d equilibrium/reaction error %g", n.Num, j, internal[6*i+j]-n.Force[j]-n.Reaction[j])
			}
		}
	}
	for j := 0; j < 3; j++ {
		if math.Abs(sum[j]) > tol || math.Abs(mom[j]) > tol {
			t.Fatalf("global balance: %v %v", sum, mom)
		}
	}
}
func TestGeometryRigidMotionAndTangent(t *testing.T) {
	f := geometryBeam(1)
	f.Sects[0].Value[0] = 10
	e, err := newGeometryElement(f.Elems[0], 0)
	if err != nil {
		t.Fatal(err)
	}
	rot := []geometryMatrix{geometryIdentity(), geometryIdentity()}
	zero, err := e.response(f.Nodes, rot)
	if err != nil {
		t.Fatal(err)
	}
	k, _ := f.Elems[0].StiffMatrix()
	for i := 0; i < 12; i++ {
		for j := 0; j < 12; j++ {
			if math.Abs(zero.tangent[i][j]-k[i][j]) > 1e-11 {
				t.Fatalf("initial tangent [%d,%d] %g != %g", i, j, zero.tangent[i][j], k[i][j])
			}
		}
	}
	r := geometryExp(geometryVector{.8, -.4, 1.2})
	shift := geometryVector{.3, -.2, .6}
	for i, n := range f.Nodes {
		x := geometryMV(r, geometryVector{n.Coord[0], n.Coord[1], n.Coord[2]})
		for j := 0; j < 3; j++ {
			n.Disp[j] = x[j] + shift[j] - n.Coord[j]
		}
		rot[i] = r
	}
	out, err := e.response(f.Nodes, rot)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range out.force {
		if math.Abs(v) > 1e-12 {
			t.Fatalf("rigid motion created force %g", v)
		}
	}
	f.Nodes[1].Disp[0] += .08
	f.Nodes[1].Disp[1] -= .12
	rot[1] = geometryMM(geometryExp(geometryVector{.11, .04, -.08}), rot[1])
	out, err = e.response(f.Nodes, rot)
	if err != nil {
		t.Fatal(err)
	}
	h := 2e-6
	for j := 0; j < 12; j++ {
		n, d := j/6, j%6
		original := f.Nodes[n].Disp[d]
		oldR := rot[n]
		var responses [2]*geometryResponse
		for side, sign := range []float64{-1, 1} {
			if d < 3 {
				f.Nodes[n].Disp[d] = original + sign*h
			} else {
				v := geometryVector{}
				v[d-3] = sign * h
				rot[n] = geometryMM(geometryExp(v), oldR)
			}
			responses[side], err = e.response(f.Nodes, rot)
			if err != nil {
				t.Fatal(err)
			}
		}
		f.Nodes[n].Disp[d] = original
		rot[n] = oldR
		for i := 0; i < 12; i++ {
			fd := (responses[1].force[i] - responses[0].force[i]) / (2 * h)
			if math.Abs(fd-out.tangent[i][j]) > 2e-7*math.Max(1, math.Abs(fd)) {
				t.Fatalf("finite rotation tangent [%d,%d]: analytic %g finite difference %g", i, j, out.tangent[i][j], fd)
			}
		}
	}
}
func TestGeometryMomentAndLoadPartition(t *testing.T) {
	var answers [2][]float64
	for run, steps := range []int{1, 20} {
		f := geometryBeam(16)
		f.Nodes[16].Force[5] = .5
		c := geometryCondition(t, steps, 1/float64(steps))
		if err := f.StaticAnalysis(nil, c); err != nil {
			t.Fatal(err)
		}
		geometryBalance(t, f, 3e-8)
		answers[run] = append([]float64(nil), f.Nodes[16].Disp...)
		if math.Abs(f.Nodes[16].Disp[0]+1-math.Sin(.5)/.5) > 5e-5 || math.Abs(f.Nodes[16].Disp[1]-(1-math.Cos(.5))/.5) > 2e-5 || math.Abs(f.Nodes[16].Disp[5]-.5) > 1e-8 {
			t.Fatalf("circular arc error: %v", f.Nodes[16].Disp)
		}
	}
	for j := 0; j < 6; j++ {
		if math.Abs(answers[0][j]-answers[1][j]) > 1e-8 {
			t.Fatalf("load partition dependence: %v %v", answers[0], answers[1])
		}
	}
}
func TestGeometryForceCantileverAndCycle(t *testing.T) {
	f := geometryBeam(8)
	f.Nodes[8].Force[1] = .5
	c := geometryCondition(t, 1, 1)
	if err := f.StaticAnalysis(nil, c); err != nil {
		t.Fatal(err)
	}
	geometryBalance(t, f, 2e-8)
	loaded := f.SaveState()
	c.SetInit(false)
	c.SetDelta(-.1)
	c.SetNlap(10)
	c.SetMax(0)
	if err := f.StaticAnalysis(nil, c); err != nil {
		t.Fatal(err)
	}
	for _, n := range f.Nodes {
		for _, v := range n.Disp {
			if math.Abs(v) > 2e-8 {
				t.Fatalf("elastic unload residual displacement %g", v)
			}
		}
	}
	c.SetDelta(.1)
	c.SetMax(1)
	if err := f.StaticAnalysis(nil, c); err != nil {
		t.Fatal(err)
	}
	geometryBalance(t, f, 2e-8)
	for i, n := range f.Nodes {
		for j, v := range n.Disp {
			if math.Abs(v-loaded.Disp[i][j]) > 2e-8 {
				t.Fatalf("reload drift %g", v-loaded.Disp[i][j])
			}
		}
	}
	f.RestoreState(loaded)
	if f.geometry == loaded.geometry {
		t.Fatal("aliased rotation state")
	}
}
func TestGeometryRollbackAndAxes(t *testing.T) {
	f := geometryBeam(1)
	f.Nodes[1].Coord = []float64{.03, .03, math.Sqrt(1 - .0018)}
	s, w, err := f.Elems[0].PrincipalAxis(.2)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(Dot(s, w, 3)) > 1e-14 {
		t.Fatal("nonorthogonal near-vertical axes")
	}
	f = geometryBeam(4)
	f.Nodes[4].Force[1] = 1
	c := geometryCondition(t, 1, 1)
	c.SetNewtonOptions(NewtonOptions{MaxIterations: 1, DisableCutbacks: true})
	if err = f.StaticAnalysis(nil, c); err == nil {
		t.Fatal("unconverged step accepted")
	}
	for _, n := range f.Nodes {
		for _, v := range n.Disp {
			if v != 0 {
				t.Fatal("failed step was not rolled back")
			}
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = geometrySolve(ctx, []map[int]float64{{0: 1}}, []float64{1}); err != context.Canceled {
		t.Fatalf("cancellation: %v", err)
	}
}

func TestGeometryConnectionsCMQAndPreload(t *testing.T) {
	// Tip rotational spring: the beam-tip curvature and spring rotation both
	// contribute to the nodal rotation. At small load, theta=M L/EI+M/k.
	f := geometryBeam(1)
	b := NewSect()
	b.Num = 2
	b.Value = []float64{0, 0, 2, 0}
	f.Elems[0].Bonds[11] = b
	f.Nodes[1].Force[5] = 1e-5
	c := geometryCondition(t, 1, 1)
	if err := f.StaticAnalysis(nil, c); err != nil {
		t.Fatal(err)
	}
	if math.Abs(f.Nodes[1].Disp[5]-1.5e-5) > 1e-9 {
		t.Fatalf("spring rotation %g", f.Nodes[1].Disp[5])
	}
	geometryBalance(t, f, 2e-8)
	// CMQ is an equivalent dead nodal load. A fully constrained beam has no
	// displacement, but has the fixed-end reactions, including loads at supports.
	f = geometryBeam(1)
	for _, n := range f.Nodes {
		for j := range n.Conf {
			n.Conf[j] = true
		}
	}
	f.Elems[0].Cmq[1] = .3
	f.Elems[0].Cmq[7] = .7
	f.Nodes[0].Force[1] = .2
	c = geometryCondition(t, 1, 1)
	if err := f.StaticAnalysis(nil, c); err != nil {
		t.Fatal(err)
	}
	if math.Abs(f.Nodes[0].Reaction[1]-.1) > 1e-12 || math.Abs(f.Nodes[1].Reaction[1]-.7) > 1e-12 {
		t.Fatalf("fixed-end reactions %v %v", f.Nodes[0].Reaction, f.Nodes[1].Reaction)
	}
	// Changed load pattern preserves the previous CMQ and support load.
	f.Elems[0].Cmq[1] = 0
	f.Elems[0].Cmq[7] = 0
	f.Nodes[0].Force[1] = 0
	f.Nodes[1].Force[1] = .1
	c.SetInit(false)
	if err := f.StaticAnalysis(nil, c); err != nil {
		t.Fatal(err)
	}
	if math.Abs(f.Nodes[1].Reaction[1]-.6) > 1e-12 {
		t.Fatalf("lost CMQ preload: %v", f.Nodes[1].Reaction)
	}
	// Re-equilibrate an ordinary linear preload under its specified load,
	// then add a second load pattern. This must agree with a fresh combined run.
	f = geometryBeam(4)
	f.Nodes[4].Force[1] = .2
	linear := NewAnalysisCondition()
	linear.SetOutput([]string{filepath.Join(t.TempDir(), "linear.otp")})
	f.Lapch = make(chan int)
	done := make(chan error, 1)
	go func() { done <- f.StaticAnalysis(nil, linear) }()
	<-f.Lapch
	f.Lapch <- 0
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	f.Lapch = nil
	f.Nodes[4].Force[1] = 0
	f.Nodes[4].Force[5] = .1
	c = geometryCondition(t, 1, 1)
	c.SetInit(false)
	if err := f.StaticAnalysis(nil, c); err != nil {
		t.Fatal(err)
	}
	g := geometryBeam(4)
	g.Nodes[4].Force[1] = .2
	g.Nodes[4].Force[5] = .1
	if err := g.StaticAnalysis(nil, geometryCondition(t, 1, 1)); err != nil {
		t.Fatal(err)
	}
	for j, v := range g.Nodes[4].Disp {
		if math.Abs(v-f.Nodes[4].Disp[j]) > 2e-8 {
			t.Fatalf("linear preload re-equilibration: %v vs %v", f.Nodes[4].Disp, g.Nodes[4].Disp)
		}
	}
}

func geometrySpaceBeam(n int, width float64, mask int) *Frame {
	f := NewFrame()
	f.Output = io.Discard
	f.Lapch = nil
	f.Pivot = nil
	s := NewSect()
	s.Num = 1
	s.E = 21e6
	s.Poi = .25
	h, t := .3, .01
	// Exact hollow-rectangle area/inertias; Bredt thin-wall torsion on midlines.
	a := width*h - (width-2*t)*(h-2*t)
	ix := (width*math.Pow(h, 3) - (width-2*t)*math.Pow(h-2*t, 3)) / 12
	iy := (h*math.Pow(width, 3) - (h-2*t)*math.Pow(width-2*t, 3)) / 12
	j := 2 * t * math.Pow((width-t)*(h-t), 2) / (width + h - 2*t)
	s.Value = []float64{a, ix, iy, j}
	f.Sects = []*Sect{s}
	ends := [][3]float64{{0, 0, 0}, {10, 0, 0}, {10, 10, 0}, {10, 10, -10}}
	for m := 0; m < 3; m++ {
		for k := 0; k < n; k++ {
			v := NewNode()
			v.Index = len(f.Nodes)
			v.Num = v.Index + 1
			for d := 0; d < 3; d++ {
				v.Coord[d] = ends[m][d] + (ends[m+1][d]-ends[m][d])*float64(k)/float64(n)
			}
			for d := 0; d < 6; d++ {
				v.Conf[d] = v.Index == 0
			}
			f.Nodes = append(f.Nodes, v)
		}
	}
	v := NewNode()
	v.Index = len(f.Nodes)
	v.Num = v.Index + 1
	copy(v.Coord, ends[3][:])
	f.Nodes = append(f.Nodes, v)
	for i := 0; i < 3*n; i++ {
		e := NewElem()
		e.Num = i + 1
		e.Sect = s
		e.Enod = []*Node{f.Nodes[i], f.Nodes[i+1]}
		if mask&(1<<(i/n)) != 0 {
			e.Cang = math.Pi / 2
		}
		f.Elems = append(f.Elems, e)
	}
	return f
}

// Independent geometrically exact rod BVP (reference.py in the verification
// bundle): three perpendicular 10 m members, 0.2 x 0.3 m box, t=.01,
// E=21e6, nu=.25, tip load 3. This is a supplementary finite-rotation test,
// not a claim to reproduce the ambiguous 1983 printed section properties.
func TestGeometrySpatialLargeRotation(t *testing.T) {
	f := geometrySpaceBeam(8, .2, 0)
	f.Nodes[24].Force[2] = 3
	c := geometryCondition(t, 10, .1)
	if err := f.StaticAnalysis(nil, c); err != nil {
		t.Fatal(err)
	}
	reference := []float64{.67696584417134, 4.170423019273409, 6.459469584750753}
	for j, want := range reference {
		if math.Abs(f.Nodes[24].Disp[j]-want) > .0011 {
			t.Fatalf("spatial elastica component %d: %g vs %g", j, f.Nodes[24].Disp[j], want)
		}
	}
	geometryBalance(t, f, 2e-7)
	saved := f.SaveState()
	c.SetInit(false)
	c.SetDelta(-.1)
	c.SetMax(0)
	if err := f.StaticAnalysis(nil, c); err != nil {
		t.Fatal(err)
	}
	for _, n := range f.Nodes {
		for _, v := range n.Disp {
			if math.Abs(v) > 2e-7 {
				t.Fatalf("spatial unload drift: %g", v)
			}
		}
	}
	c.SetDelta(.1)
	c.SetMax(1)
	if err := f.StaticAnalysis(nil, c); err != nil {
		t.Fatal(err)
	}
	for i, n := range f.Nodes {
		for j, v := range n.Disp {
			if math.Abs(v-saved.Disp[i][j]) > 2e-7 {
				t.Fatalf("spatial reload drift: %g", v-saved.Disp[i][j])
			}
		}
	}
	f.Nodes[1].Coord[0] += .01
	if err := f.StaticAnalysis(nil, c); err == nil {
		t.Fatal("accepted continuation after reference geometry edit")
	}
}
