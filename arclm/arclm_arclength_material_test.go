package arclm

import (
	"bytes"
	"context"
	"io"
	"math"
	"path/filepath"
	"reflect"
	"testing"
)

// Plastic two-bar arch: first axial yield occurs before the elastic limit
// point. After both bars yield, P=2 Ny y/sqrt(a²+y²) decreases monotonically.
// This independent closed form checks material-driven collapse, not just an
// elastic turning point with an unused material flag.
func plasticArcArch() *Frame {
	f := arcArch()
	geometryYield(f.Sects[0], 1e6)
	f.Sects[0].Yield[0], f.Sects[0].Yield[1] = .05/RADIUS, -.05/RADIUS
	return f
}

func materialArcCondition(t *testing.T, steps int, init bool) *AnalysisCondition {
	t.Helper()
	c := NewAnalysisCondition()
	c.SetInit(init)
	c.SetNlap(steps)
	c.SetOutput([]string{filepath.Join(t.TempDir(), "plastic.otp")})
	c.SetArcLength(ArcLengthOptions{Step: .0005, MinStep: 1e-8, MaxStep: .0005, LoadScale: .01, MonitorNode: 2})
	return c
}

func TestMaterialArcArchCollapseAndDescendingPath(t *testing.T) {
	const ny, ea, a, h, reference = .05, 10., .5, .1, .01
	l0 := math.Hypot(a, h)
	ly := l0 * (1 - ny/ea)
	yieldY := math.Sqrt(ly*ly - a*a)
	collapse := 2 * ny * yieldY / ly / reference
	for _, ds := range []float64{.0005, .00025} {
		f := plasticArcArch()
		c := materialArcCondition(t, 500, true)
		peak, last, plasticPoints := 0., 0., 0
		descending := false
		c.arcLength.Step, c.arcLength.MaxStep = ds, ds
		c.arcLength.OnStep = func(f *Frame, p ArcLengthPoint) (bool, error) {
			y := h + f.Nodes[1].Disp[1]
			l := math.Hypot(a, y)
			n := math.Min(ny, ea*(l0-l)/l0)
			want := 2 * n * y / l / reference
			if math.Abs(p.Factor-want) > 3e-6 {
				t.Fatalf("y=%g load=%g exact=%g", y, p.Factor, want)
			}
			if p.Residual > p.Tolerance || math.Abs(p.Constraint) > 1e-10 {
				t.Fatalf("unbalanced point %+v", p)
			}
			if f.Elems[0].material != nil && f.Elems[0].material.plastic != [12]float64{} {
				plasticPoints++
				if p.Factor < last {
					descending = true
				}
			}
			last = p.Factor
			peak = math.Max(peak, p.Factor)
			return y <= .04, nil
		}
		if err := f.Arclm101(nil, c); err != nil {
			t.Fatal(err)
		}
		if !descending || plasticPoints < 20 || math.Abs(peak-collapse) > .001*collapse || last > .65*peak {
			t.Fatalf("peak=%g theoretical=%g last=%g plastic points=%d descending=%v", peak, collapse, last, plasticPoints, descending)
		}
		t.Logf("ds=%g peak=%.9g theoretical=%.9g final=%.9g plastic points=%d", ds, peak, collapse, last, plasticPoints)
	}
}

func TestMaterialArcSaveRestoreWorkerAndContinuation(t *testing.T) {
	continuous, split := plasticArcArch(), plasticArcArch()
	run := func(f *Frame, steps int, init bool) {
		if err := f.Arclm101(nil, materialArcCondition(t, steps, init)); err != nil {
			t.Fatal(err)
		}
	}
	run(continuous, 140, true)
	run(split, 80, true)
	if split.geometry.arcDirection[len(split.geometry.arcDirection)-1] >= 0 || split.Elems[0].material == nil || split.Elems[0].material.plastic == [12]float64{} {
		t.Fatal("restart must be on a plastic descending branch")
	}
	saved := split.SaveState()
	run(split, 2, false)
	split.RestoreState(saved)
	if !reflect.DeepEqual(saved, split.SaveState()) {
		t.Fatal("SaveState/RestoreState lost plastic or geometric state")
	}
	var wire bytes.Buffer
	if err := EncodeWorkerFrame(&wire, split); err != nil {
		t.Fatal(err)
	}
	restored, err := DecodeWorkerFrame(&wire)
	if err != nil {
		t.Fatal(err)
	}
	restored.Output, restored.Lapch, restored.Pivot = io.Discard, nil, nil
	if !reflect.DeepEqual(saved, restored.SaveState()) {
		t.Fatal("worker round trip lost state")
	}
	run(restored, 60, false)
	if math.Abs(continuous.geometry.factor-restored.geometry.factor) > 1e-9 || continuous.ContinuationStep() != restored.ContinuationStep() {
		t.Fatal("resume changed load or step count")
	}
	for i, e := range continuous.Elems {
		for j, p := range e.material.plastic {
			if math.Abs(p-restored.Elems[i].material.plastic[j]) > 1e-10 {
				t.Fatal("resume changed plastic history")
			}
		}
		for j, s := range e.Stress {
			if math.Abs(s-restored.Elems[i].Stress[j]) > 1e-9 {
				t.Fatal("resume changed member stress")
			}
		}
	}
	for i, n := range continuous.Nodes {
		for j, u := range n.Disp {
			if math.Abs(u-restored.Nodes[i].Disp[j]) > 1e-10 {
				t.Fatal("resume changed displacement")
			}
		}
	}
	// Failed correctors and an interrupted step restore every plastic variable.
	state := restored.SaveState()
	c := materialArcCondition(t, 1, false)
	c.arcLength.Step, c.arcLength.MaxStep = 1, 1
	c.SetNewtonOptions(NewtonOptions{MaxIterations: 1, DisableCutbacks: true})
	if err := restored.Arclm101(nil, c); err == nil {
		t.Fatal("large step should fail")
	}
	if !reflect.DeepEqual(state, restored.SaveState()) {
		t.Fatal("failed step leaked trial plastic history")
	}
	c = materialArcCondition(t, 1, false)
	c.nlmaterial = true
	aAnalysis, err := newGeometryAnalysis(restored, context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = aAnalysis.setup(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	aAnalysis.ctx = ctx
	_, _, err = aAnalysis.arcStep(restored.geometry.factor, .0005, restored.geometry.arcDirection, []float64{1, .0001}, aAnalysis.arcLoadPattern())
	if err == nil || !reflect.DeepEqual(state, restored.SaveState()) {
		t.Fatal("cancelled step leaked trial state")
	}
}

func TestMaterialArcCMQLoadDerivative(t *testing.T) {
	f := geometryBeam(1)
	f.Sects[0].Value[0] = 10
	geometryYield(f.Sects[0], .25)
	e, err := newGeometryElement(f.Elems[0], 0)
	if err != nil {
		t.Fatal(err)
	}
	f.Nodes[1].Disp[0], f.Nodes[1].Disp[1], f.Nodes[1].Disp[2] = .04, .13, -.05
	rot := []geometryMatrix{geometryExp(geometryVector{.03, -.04, .06}), geometryExp(geometryVector{.15, .08, -.13})}
	m := &geometryMaterialTrial{
		fixed:     [12]float64{.01, .02, -.03, .04, -.01, .02, -.02, .03, .02, -.02, .01, .04},
		fixedRate: [12]float64{-.02, .03, .01, -.01, .02, .02, .04, -.01, .02, .01, -.03, .02},
	}
	r, err := e.responseMaterial(f.Nodes, rot, m)
	if err != nil {
		t.Fatal(err)
	}
	if r.material.Multipliers[0]+r.material.Multipliers[1] == 0 {
		t.Fatal("fixture must yield")
	}
	const h = 1e-5
	var sides [2]*geometryResponse
	for side, sign := range []float64{-1, 1} {
		trial := *m
		for i := range trial.fixed {
			trial.fixed[i] += sign * h * m.fixedRate[i]
		}
		sides[side], err = e.responseMaterial(f.Nodes, rot, &trial)
		if err != nil {
			t.Fatal(err)
		}
	}
	for i, rate := range r.loadDerivative {
		fd := (sides[1].force[i] - sides[0].force[i]) / (2 * h)
		if math.Abs(fd-rate) > 2e-7*math.Max(1, math.Abs(fd)) {
			t.Fatalf("CMQ load derivative [%d]=%g finite difference=%g", i, rate, fd)
		}
	}
	// At weak yield, a positive displacement with decreasing CMQ can unload;
	// classifying from the displacement direction alone gives the wrong branch.
	f = geometryBeam(1)
	f.Sects[0].Value = []float64{1, 0, 0, 0}
	geometryYield(f.Sects[0], .2)
	e, err = newGeometryElement(f.Elems[0], 0)
	if err != nil {
		t.Fatal(err)
	}
	f.Nodes[1].Disp[0] = RADIUS * .2
	rot = []geometryMatrix{geometryIdentity(), geometryIdentity()}
	d := [12]float64{}
	d[6] = 1
	for _, factorDirection := range []float64{-20, 20} {
		m = &geometryMaterialTrial{direction: &d, loadDirection: factorDirection}
		m.fixedRate[0], m.fixedRate[6] = -.1, .1
		r, err = e.responseMaterial(f.Nodes, rot, m)
		if err != nil {
			t.Fatal(err)
		}
		want := 0.
		if factorDirection < 0 {
			want = 1
		}
		if math.Abs(r.tangent[6][6]-want) > 1e-8 {
			t.Fatalf("load direction %g tangent=%g want=%g", factorDirection, r.tangent[6][6], want)
		}
	}
}

func TestMaterialArcBeamThreeHingeCollapse(t *testing.T) {
	// Fixed-fixed beam, point load at x=1 of L=3: Pu=3 Mp. Low axial
	// stiffness suppresses membrane action so the finite-rotation curve stays
	// close to the independent small-deflection plastic-collapse solution.
	const mp = .01
	f := geometryBeam(1)
	f.Sects[0].Value[0] = .0001
	geometryYield(f.Sects[0], 1e6)
	f.Sects[0].Yield[10], f.Sects[0].Yield[11] = mp/RADIUS, -mp/RADIUS
	middle := f.Nodes[1]
	middle.Conf[0] = true
	middle.Force[1] = -3 * mp
	right := NewNode()
	right.Num, right.Index, right.Coord[0] = 3, 2, 3
	for i := range right.Conf {
		right.Conf[i] = true
	}
	f.Nodes = append(f.Nodes, right)
	e := NewElem()
	e.Num, e.Sect, e.Enod = 2, f.Sects[0], []*Node{middle, right}
	f.Elems = append(f.Elems, e)
	c := materialArcCondition(t, 80, true)
	max := 0.
	c.arcLength.OnStep = func(_ *Frame, p ArcLengthPoint) (bool, error) {
		if p.Residual > p.Tolerance {
			t.Fatal("unbalanced point")
		}
		max = math.Max(max, p.Factor)
		return false, nil
	}
	if err := f.Arclm101(nil, c); err != nil {
		t.Fatal(err)
	}
	if math.Abs(max-1) > .0001 || math.Abs(f.Elems[0].Stress[11]) < .99*mp || math.Abs(f.Elems[1].Stress[5]) < .99*mp {
		t.Fatalf("collapse factor=%g moments=%g,%g", max, f.Elems[0].Stress[11], f.Elems[1].Stress[5])
	}
	t.Logf("three-hinge collapse factor %.9g (theoretical 1)", max)
}

func TestMaterialArcStaticEntryAndElasticUnloading(t *testing.T) {
	f, g := plasticArcArch(), plasticArcArch()
	c := materialArcCondition(t, 80, true)
	if err := f.Arclm101(nil, c); err != nil {
		t.Fatal(err)
	}
	c = materialArcCondition(t, 80, true)
	c.SetNlmaterial(true)
	if err := g.StaticAnalysis(nil, c); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(f.SaveState(), g.SaveState()) {
		t.Fatal("StaticAnalysis -nlmat and Arclm101 arc paths differ")
	}
	state := f.SaveState()
	before := f.Elems[0].material.plastic
	// Reversing the displacement along the yielded branch unloads both bars.
	c.SetInit(false)
	a, err := newGeometryAnalysis(f, context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.setup(); err != nil {
		t.Fatal(err)
	}
	previous := append([]float64(nil), f.geometry.arcDirection...)
	for i := range previous {
		previous[i] = -previous[i]
	}
	p, _, err := a.arcStep(f.geometry.factor, .0001, previous, []float64{1, .0001}, a.arcLoadPattern())
	if err != nil {
		t.Fatal(err)
	}
	if p.Residual > p.Tolerance {
		t.Fatal("unloading is not equilibrated")
	}
	for i, v := range before {
		if math.Abs(v-f.Elems[0].material.plastic[i]) > 1e-10 {
			t.Fatal("elastic unloading added plastic strain")
		}
	}
	if f.Nodes[1].Disp[1] <= state.Disp[1][1] {
		t.Fatal("fixture did not reverse displacement")
	}
	f.RestoreState(state)
	if !reflect.DeepEqual(state, f.SaveState()) {
		t.Fatal("unloading trial cannot be restored")
	}
}

func TestMaterialArcBorderedCMQColumnFiniteDifference(t *testing.T) {
	f := geometryBeam(1)
	f.Sects[0].Value[0] = 10
	geometryYield(f.Sects[0], .25)
	copy(f.Elems[0].Cmq, []float64{.01, .02, -.03, .04, -.01, .02, -.02, .03, .02, -.02, .01, .04})
	c := materialArcCondition(t, 1, true)
	c.SetNlmaterial(true)
	a, err := newGeometryAnalysis(f, context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.setup(); err != nil {
		t.Fatal(err)
	}
	f.Nodes[1].Disp[0], f.Nodes[1].Disp[1] = .04, .13
	a.state.rotations[1] = geometryExp(geometryVector{0, 0, .08})
	const factor, h = .7, 1e-5
	trial, err := a.evaluate(factor)
	if err != nil {
		t.Fatal(err)
	}
	if trial.responses[0].material.Multipliers[0]+trial.responses[0].material.Multipliers[1] == 0 {
		t.Fatal("fixture must yield")
	}
	lo, err := a.evaluate(factor - h)
	if err != nil {
		t.Fatal(err)
	}
	hi, err := a.evaluate(factor + h)
	if err != nil {
		t.Fatal(err)
	}
	n := len(a.free)
	normal, weights := make([]float64, n+1), make([]float64, n+1)
	normal[n], weights[n] = 1, 1
	d, err := a.arcCorrection(factor, trial, a.arcLoadPattern(), normal, weights, make([]float64, n), 1)
	if err != nil {
		t.Fatal(err)
	}
	for i, j := range a.free {
		fd := (hi.residual[j] - lo.residual[j]) / (2 * h)
		kd := 0.
		for k, v := range trial.k[i] {
			kd += v * d[k]
		}
		if math.Abs(kd-fd*d[n]) > 1e-7*math.Max(1, math.Abs(fd)) {
			t.Fatalf("bordered load column [%d]: Kdu=%g finite difference=%g", i, kd, fd*d[n])
		}
	}
	if f.Elems[0].material != nil {
		t.Fatal("linearization committed trial plastic history")
	}
}
