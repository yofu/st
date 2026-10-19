package arclm

import (
	"bytes"
	"context"
	"errors"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func newtonParallelFrame() *Frame {
	f := plasticTestFrame()
	f.Lapch, f.Pivot = nil, nil
	f.Nodes[1].Force[0] = 220
	f.Nodes[0].Force[0] = 7
	e := NewElem()
	e.Num, e.Enod, e.Sect = 2, f.Nodes, f.Elems[0].Sect.Snapshot()
	for j := 0; j < 6; j++ {
		e.Sect.Yield[2*j], e.Sect.Yield[2*j+1] = 1000, -1000
	}
	f.Elems = append(f.Elems, e)
	return f
}

func newtonTestCondition(t *testing.T) *AnalysisCondition {
	t.Helper()
	c := NewAnalysisCondition()
	c.SetEps(1e-10)
	c.SetOutput([]string{filepath.Join(t.TempDir(), "newton.result.otp")})
	return c
}

func nearNewton(t *testing.T, label string, got, want, tol float64) {
	t.Helper()
	if !finitePlastic(got) || math.Abs(got-want) > tol {
		t.Fatalf("%s: got %.14g want %.14g tolerance=%g", label, got, want, tol)
	}
}

func TestNewtonPlasticRedistributionAndReactions(t *testing.T) {
	for _, solver := range []string{"LLS", "LDLT"} {
		t.Run(solver, func(t *testing.T) {
			f := newtonParallelFrame()
			var log bytes.Buffer
			f.Output = &log
			c := newtonTestCondition(t)
			c.SetSolver(solver)
			cancelled := 0
			if err := f.Arclm101(func() { cancelled++ }, c); err != nil {
				t.Fatal(err)
			}
			nearNewton(t, "displacement", f.Nodes[1].Disp[0], 1.25, 1e-8)
			nearNewton(t, "plastic bar force", f.Elems[0].Stress[6], 95, 1e-8)
			nearNewton(t, "elastic bar force", f.Elems[1].Stress[6], 125, 1e-8)
			nearNewton(t, "support reaction including support load", f.Nodes[0].Reaction[0], -227, 1e-8)
			nearNewton(t, "free reaction", f.Nodes[1].Reaction[0], 0, 0)
			p := f.Elems[0].material.plastic
			nearNewton(t, "plastic extension (committed once)", p[6]-p[0], .3, 1e-8)
			if !strings.Contains(log.String(), "ITER=2") {
				t.Fatal("redistribution did not exercise repeated Newton evaluation")
			}
			if cancelled != 1 || f.Running() {
				t.Fatal("analysis lifecycle not completed")
			}
			t.Logf("u=%g N=[%g,%g] R=%g", f.Nodes[1].Disp[0], f.Elems[0].Stress[6], f.Elems[1].Stress[6], f.Nodes[0].Reaction[0])
		})
	}
}

func TestNewtonIncrementSizeAndContinuation(t *testing.T) {
	one, many := newtonParallelFrame(), newtonParallelFrame()
	c := newtonTestCondition(t)
	if err := one.Arclm101(nil, c); err != nil {
		t.Fatal(err)
	}
	c = newtonTestCondition(t)
	c.SetNlap(10)
	c.SetDelta(.1)
	if err := many.Arclm101(nil, c); err != nil {
		t.Fatal(err)
	}
	for i := range one.Elems {
		for j := 0; j < 12; j++ {
			nearNewton(t, "step-size force", many.Elems[i].Stress[j], one.Elems[i].Stress[j], 1e-7)
			nearNewton(t, "step-size plastic deformation", many.Elems[i].material.plastic[j], one.Elems[i].material.plastic[j], 1e-8)
		}
	}
	saved := one.SaveState()
	c = newtonTestCondition(t)
	c.SetInit(false)
	c.SetNlap(2)
	c.SetDelta(-.25)
	if err := one.Arclm101(nil, c); err != nil {
		t.Fatal(err)
	}
	nearNewton(t, "unload factor", one.plasticLoad, .5, 1e-12)
	nearNewton(t, "unload displacement", one.Nodes[1].Disp[0], .7, 1e-8)
	nearNewton(t, "unload first force", one.Elems[0].Stress[6], 40, 1e-7)
	nearNewton(t, "unload second force", one.Elems[1].Stress[6], 70, 1e-7)
	c = newtonTestCondition(t)
	c.SetInit(false)
	c.SetDelta(.5)
	if err := one.Arclm101(nil, c); err != nil {
		t.Fatal(err)
	}
	nearNewton(t, "reload displacement", one.Nodes[1].Disp[0], 1.25, 1e-8)
	other := newtonParallelFrame()
	other.RestoreState(saved)
	c = newtonTestCondition(t)
	c.SetInit(false)
	c.SetDelta(-.5)
	if err := other.Arclm101(nil, c); err != nil {
		t.Fatal(err)
	}
	nearNewton(t, "restored continuation displacement", other.Nodes[1].Disp[0], .7, 1e-8)
	c = newtonTestCondition(t)
	c.SetInit(false)
	c.SetStart(1)
	before := other.SaveState()
	if err := other.Arclm101(nil, c); err == nil {
		t.Fatal("inconsistent START should fail")
	}
	if !reflect.DeepEqual(other.SaveState(), before) {
		t.Fatal("invalid continuation modified the state")
	}
}

func TestNewtonOverCapacityFailsAndRollsBack(t *testing.T) {
	f := plasticTestFrame()
	f.Lapch, f.Pivot = nil, nil
	before := f.SaveState()
	var log bytes.Buffer
	f.Output = &log
	c := newtonTestCondition(t)
	c.SetNewtonOptions(NewtonOptions{MaxCutbacks: 3})
	err := f.Arclm101(nil, c)
	var failure *PlasticConvergenceError
	if !errors.As(err, &failure) {
		t.Fatalf("over-capacity load was not rejected: %v", err)
	}
	if failure.Target != 1 || failure.Residual <= failure.Tolerance {
		t.Fatalf("incorrect convergence failure diagnostics: %v", err)
	}
	if !reflect.DeepEqual(f.SaveState(), before) {
		t.Fatal("failed requested step did not restore all history")
	}
	if !strings.Contains(log.String(), "ARCLM101 CUTBACK:") {
		t.Fatal("failure did not exercise cutbacks")
	}
	name := strings.TrimSuffix(c.otp[0], ".otp") + "@1.otp"
	if _, err := os.Stat(name); !os.IsNotExist(err) {
		t.Fatal("unconverged output was written", err)
	}
	t.Logf("correctly rejected force 120 > capacity 95; %v", failure)
}

func TestNewtonFailedLaterStepPreservesLastOutputState(t *testing.T) {
	f := newtonParallelFrame()
	for j := 0; j < 6; j++ {
		f.Elems[1].Sect.Yield[2*j], f.Elems[1].Sect.Yield[2*j+1] = 200, -200
	}
	f.Lapch = make(chan int)
	c := newtonTestCondition(t)
	c.SetNlap(2)
	c.SetNewtonOptions(NewtonOptions{MaxCutbacks: 2})
	done := make(chan error, 1)
	go func() { done <- f.Arclm101(nil, c) }()
	select {
	case lap := <-f.Lapch:
		if lap != 1 {
			t.Fatal("unexpected lap")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("first step stalled")
	}
	first := f.SaveState()
	f.Lapch <- 0
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("over-capacity second step accepted")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("failed step stalled")
	}
	if !reflect.DeepEqual(f.SaveState(), first) {
		t.Fatal("failed second step lost the first converged state")
	}
	name := strings.TrimSuffix(c.otp[0], ".otp")
	if _, err := os.Stat(name + "@1.otp"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(name + "@2.otp"); !os.IsNotExist(err) {
		t.Fatal("failed second output exists")
	}
}

func TestNewtonFixedEndForcesAndSupportLoads(t *testing.T) {
	f := plasticTestFrame()
	f.Lapch, f.Pivot = nil, nil
	f.Nodes[0].Force[0], f.Nodes[1].Force[0] = 3, 0
	f.Elems[0].Cmq[0], f.Elems[0].Cmq[6] = -5, -5
	input := append([]float64(nil), f.Elems[0].Cmq...)
	c := newtonTestCondition(t)
	c.SetNlap(2)
	c.SetDelta(.5)
	if err := f.Arclm101(nil, c); err != nil {
		t.Fatal(err)
	}
	nearNewton(t, "distributed load displacement", f.Nodes[1].Disp[0], .05, 1e-9)
	nearNewton(t, "distributed load support reaction", f.Nodes[0].Reaction[0], -13, 1e-8)
	nearNewton(t, "zero free force", f.Elems[0].Stress[6], 0, 1e-8)
	if !reflect.DeepEqual(input, f.Elems[0].Cmq) {
		t.Fatal("assembly modified input CMQ")
	}
	c = newtonTestCondition(t)
	c.SetInit(false)
	c.SetDelta(-.5)
	if err := f.Arclm101(nil, c); err != nil {
		t.Fatal(err)
	}
	nearNewton(t, "half-load reaction", f.Nodes[0].Reaction[0], -6.5, 1e-8)
}

func TestNewtonReleasedFixedEndLoads(t *testing.T) {
	f := plasticTestFrame()
	f.Lapch, f.Pivot = nil, nil
	for _, node := range f.Nodes {
		for j := 0; j < 6; j++ {
			node.Conf[j], node.Force[j] = true, 0
		}
	}
	e := f.Elems[0]
	e.Bonds[5] = NewSect()
	e.Bonds[5].Num = 1
	e.Cmq[1], e.Cmq[7], e.Cmq[5], e.Cmq[11] = -60, -60, -10, 10
	for j := 0; j < 6; j++ {
		e.Sect.Yield[2*j], e.Sect.Yield[2*j+1] = 1000, -1000
	}
	c := newtonTestCondition(t)
	if err := f.Arclm101(nil, c); err != nil {
		t.Fatal(err)
	}
	nearNewton(t, "released moment", e.Stress[5], 0, 1e-9)
	nearNewton(t, "opposite fixed moment", e.Stress[11], 15, 1e-8)
	nearNewton(t, "released-end shear", f.Nodes[0].Reaction[1], -45, 1e-8)
	nearNewton(t, "fixed-end shear", f.Nodes[1].Reaction[1], -75, 1e-8)
	nearNewton(t, "global moment balance", f.Nodes[1].Reaction[1]+f.Nodes[1].Reaction[5]+60, 0, 1e-8)
}

func TestNewtonCoupledEquilibrium(t *testing.T) {
	f := newtonParallelFrame()
	f.Nodes[1].Conf[5] = false
	f.Nodes[1].Force[5] = 110
	for _, e := range f.Elems {
		for _, j := range []int{1, 2} {
			e.Sect.Yield[2*j], e.Sect.Yield[2*j+1] = 1000, -1000
		}
	}
	c := newtonTestCondition(t)
	c.SetSolver("LDLT")
	if err := f.Arclm101(nil, c); err != nil {
		t.Fatal(err)
	}
	n, m := 0.0, 0.0
	for _, e := range f.Elems {
		n += e.Stress[6]
		m += e.Stress[11]
	}
	nearNewton(t, "free axial equilibrium", n, 220, 1e-7)
	nearNewton(t, "free bending equilibrium", m, 110, 1e-7)
	nearNewton(t, "vertical equilibrium", f.Nodes[0].Reaction[1]+f.Nodes[1].Reaction[1], 0, 1e-7)
	nearNewton(t, "moment equilibrium", f.Nodes[0].Reaction[5]+f.Nodes[1].Reaction[1]+110, 0, 1e-7)
	t.Logf("coupled u=%g rotation=%g sumN=%g sumM=%g", f.Nodes[1].Disp[0], f.Nodes[1].Disp[5], n, m)
}

func TestNewtonDefaultPlasticExponents(t *testing.T) {
	f := newtonParallelFrame()
	f.Nodes[1].Conf[5] = false
	f.Nodes[1].Force[5] = 110
	for _, e := range f.Elems {
		e.Sect.Exp, e.Sect.Exq = 1.5, 1.5
		for _, j := range []int{1, 2} {
			e.Sect.Yield[2*j], e.Sect.Yield[2*j+1] = 1000, -1000
		}
	}
	c := newtonTestCondition(t)
	if err := f.Arclm101(nil, c); err != nil {
		t.Fatal(err)
	}
	n, m := 0.0, 0.0
	for _, e := range f.Elems {
		n += e.Stress[6]
		m += e.Stress[11]
	}
	nearNewton(t, "p=1.5 axial equilibrium", n, 220, 1e-7)
	nearNewton(t, "p=1.5 bending equilibrium", m, 110, 1e-7)
	t.Logf("p=q=1.5 sumN=%g sumM=%g", n, m)
}

func TestNewtonStartMaxAndDottedOutputPath(t *testing.T) {
	f := newtonParallelFrame()
	c := newtonTestCondition(t)
	c.SetStart(.5)
	c.SetDelta(.5)
	c.SetMax(1.25)
	c.SetNlap(10)
	dir := filepath.Join(t.TempDir(), "dir.with.dots")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	c.SetOutput([]string{filepath.Join(dir, "model.with.dots.otp")})
	if err := f.Arclm101(nil, c); err != nil {
		t.Fatal(err)
	}
	nearNewton(t, "MAX bound", f.plasticLoad, 1.25, 1e-12)
	if _, err := os.Stat(filepath.Join(dir, "model.with.dots@2.otp")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "model.with.dots@3.otp")); !os.IsNotExist(err) {
		t.Fatal("extra step after MAX")
	}
}

func TestNewtonStopWhileAwaitingProgress(t *testing.T) {
	f := newtonParallelFrame()
	f.Lapch = make(chan int)
	c := newtonTestCondition(t)
	c.SetDelta(.25)
	c.SetNlap(4)
	done := make(chan error, 1)
	cancelCount := 0
	go func() { done <- f.Arclm101(func() { cancelCount++ }, c) }()
	select {
	case <-f.Lapch:
	case <-time.After(10 * time.Second):
		t.Fatal("progress stalled")
	}
	first := f.SaveState()
	if !f.Running() {
		t.Fatal("missing running flag")
	}
	if err := f.Stop(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Stop did not release progress wait")
	}
	if f.Running() || cancelCount != 1 {
		t.Fatal("cancel lifecycle failed")
	}
	if !reflect.DeepEqual(first, f.SaveState()) {
		t.Fatal("Stop changed the last converged state")
	}
	if err := f.Stop(); err == nil {
		t.Fatal("Stop after completion should fail")
	}
}

func TestNewtonCutbackRecoversCurvedResponse(t *testing.T) {
	f := newtonParallelFrame()
	f.Nodes[1].Conf[5] = false
	f.Nodes[1].Force[5] = 110
	for _, e := range f.Elems {
		for _, j := range []int{1, 2} {
			e.Sect.Yield[2*j], e.Sect.Yield[2*j+1] = 1000, -1000
		}
	}
	var log bytes.Buffer
	f.Output = &log
	c := newtonTestCondition(t)
	c.SetEps(1e-8)
	c.SetNewtonOptions(NewtonOptions{MaxIterations: 2, MaxCutbacks: 10})
	if err := f.Arclm101(nil, c); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(log.String(), "ARCLM101 CUTBACK:") {
		t.Fatalf("test did not exercise a successful cutback: %s", log.String())
	}
	n, m := 0.0, 0.0
	for _, e := range f.Elems {
		n += e.Stress[6]
		m += e.Stress[11]
	}
	nearNewton(t, "cutback axial equilibrium", n, 220, 1e-5)
	nearNewton(t, "cutback bending equilibrium", m, 110, 1e-5)
	nearNewton(t, "cutback final factor", f.plasticLoad, 1, 0)
	t.Logf("successful cutbacks=%d N=%g M=%g", strings.Count(log.String(), "ARCLM101 CUTBACK:"), n, m)
}

func TestNewtonAxialBracesWithRotationalReleases(t *testing.T) {
	f := newtonParallelFrame()
	for _, e := range f.Elems {
		e.Sect.Value = []float64{1, 0, 0, 0}
		for _, j := range []int{3, 4, 5, 9, 10, 11} {
			e.Bonds[j] = NewSect()
			e.Bonds[j].Num = 1
		}
	}
	c := newtonTestCondition(t)
	if err := f.Arclm101(nil, c); err != nil {
		t.Fatal(err)
	}
	nearNewton(t, "brace displacement", f.Nodes[1].Disp[0], 1.25, 1e-8)
	nearNewton(t, "brace reaction", f.Nodes[0].Reaction[0], -227, 1e-8)
	if f.Elems[0].Bonds[0].Num != -2 || f.Elems[0].Bonds[6].Num != -2 {
		t.Fatal("brace tension state missing")
	}
}

func TestNewtonRotatedFrameKeepsInputGeometry(t *testing.T) {
	f := newtonParallelFrame()
	for j := 0; j < 6; j++ {
		f.Nodes[1].Conf[j] = false
		f.Nodes[0].Force[j], f.Nodes[1].Force[j] = 0, 0
	}
	direction := []float64{1.0 / 3, 2.0 / 3, 2.0 / 3}
	copy(f.Nodes[1].Coord, direction)
	for j := 0; j < 3; j++ {
		f.Nodes[1].Force[j] = 220 * direction[j]
	}
	c := newtonTestCondition(t)
	c.SetNlap(2)
	c.SetDelta(.5)
	if err := f.Arclm101(nil, c); err != nil {
		t.Fatal(err)
	}
	for j := 0; j < 3; j++ {
		nearNewton(t, "rotated displacement", f.Nodes[1].Disp[j], 1.25*direction[j], 1e-8)
		nearNewton(t, "rotated reaction", f.Nodes[0].Reaction[j], -220*direction[j], 1e-8)
		nearNewton(t, "input geometry", f.Nodes[1].Coord[j], direction[j], 0)
	}
	nearNewton(t, "rotated first force", f.Elems[0].Stress[6], 95, 1e-8)
}

func TestNewtonInvalidInputsPreserveState(t *testing.T) {
	for _, which := range []string{"nil", "geometry", "solver", "tolerance", "max", "capacity"} {
		t.Run(which, func(t *testing.T) {
			f := newtonParallelFrame()
			c := newtonTestCondition(t)
			switch which {
			case "nil":
				c = nil
			case "geometry":
				c.SetNlgeometry(true)
			case "solver":
				c.SetSolver("CG")
			case "tolerance":
				c.SetEps(0)
			case "max":
				c.SetMax(-1)
			case "capacity":
				f.Elems[0].Sect.Yield[0], f.Elems[0].Sect.Yield[1] = 0, 0
			}
			before := f.SaveState()
			if err := f.Arclm101(nil, c); err == nil {
				t.Fatal("invalid input accepted")
			}
			if !reflect.DeepEqual(before, f.SaveState()) || f.Running() {
				t.Fatal("invalid input changed the state or running flag")
			}
		})
	}
	// A supplied old state must be balanced as stored, not after silently
	// projecting an inadmissible force to the yield surface.
	f := plasticTestFrame()
	f.Lapch, f.Pivot = nil, nil
	f.Nodes[1].Force[0] = 95
	f.Elems[0].Stress[0], f.Elems[0].Stress[6] = -120, 120
	c := newtonTestCondition(t)
	c.SetInit(false)
	c.SetStart(1)
	before := f.SaveState()
	if err := f.Arclm101(nil, c); err == nil || !strings.Contains(err.Error(), "not in equilibrium") {
		t.Fatal("unbalanced continuation state accepted", err)
	}
	if !reflect.DeepEqual(before, f.SaveState()) {
		t.Fatal("unbalanced continuation state modified")
	}
}
