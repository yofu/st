package arclm

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func transientAxial() *Frame {
	f := plasticTestFrame()
	f.Lapch, f.Pivot, f.Output = nil, nil, io.Discard
	for _, n := range f.Nodes {
		for i := range n.Force {
			n.Force[i] = 0
		}
	}
	f.Nodes[1].Mass = 1
	return f
}

func TestGroundMotionCSV(t *testing.T) {
	m, err := ReadGroundMotionCSV(strings.NewReader("\ufeff時刻[s],X加速度[cm/s²],Y加速度[cm/s²]\n0,100,-200\n.1,300,200\n.3,0,100\n"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := m.At(.05)
	if err != nil || p.X != 2 || p.Y != 0 {
		t.Fatalf("interpolation/unit conversion=%+v err=%v", p, err)
	}
	for _, s := range []string{"0,0,0\n0,1,0", "1,0,0\n2,0,0", "0,NaN,0\n1,0,0", "0,0\n1,0", "time,x,y\n0,0,0", "0,bad,0\n1,0,0"} {
		if _, err := ReadGroundMotionCSV(strings.NewReader(s)); err == nil {
			t.Fatalf("accepted %q", s)
		}
	}
	if _, err := m.At(.4); err == nil {
		t.Fatal("extrapolated beyond record")
	}
}

func TestRayleighFit(t *testing.T) {
	a, b, err := RayleighCoefficients(1, .2, .02, .03)
	if err != nil {
		t.Fatal(err)
	}
	for i, period := range []float64{1, .2} {
		w := 2 * math.Pi / period
		nearNewton(t, "Rayleigh ratio", a/(2*w)+b*w/2, []float64{.02, .03}[i], 1e-14)
	}
	if _, _, err := RayleighCoefficients(1, .2, .9, .001); err == nil {
		t.Fatal("accepted negative damping coefficient")
	}
}

func TestTimeHistoryElasticFreeVibration(t *testing.T) {
	for _, zeta := range []float64{0, .02} {
		t.Run(timeHistoryNumber(zeta), func(t *testing.T) {
			f := transientAxial()
			u := make([]float64, 12)
			u[6] = .01
			if _, err := f.UpdateStressPlastic(u); err != nil {
				t.Fatal(err)
			}
			f.UpdateForm(u)
			o := TimeHistoryOptions{BaseLoad: make([]float64, 12), TimeStep: .002, RelativeTolerance: 1e-10}
			if zeta > 0 {
				o.Period1, o.Period2, o.Damping1, o.Damping2 = 2*math.Pi/10, .2, zeta, zeta
			}
			alpha, eta, _ := RayleighCoefficients(o.Period1, o.Period2, o.Damping1, o.Damping2)
			z := (alpha + eta*100) / 20
			wd := 10 * math.Sqrt(1-z*z)
			maxError, maxEnergy := 0., 0.
			o.OnStep = func(s *TimeHistoryState) error {
				want := .01 * math.Exp(-z*10*s.Time) * (math.Cos(wd*s.Time) + z/math.Sqrt(1-z*z)*math.Sin(wd*s.Time))
				maxError = math.Max(maxError, math.Abs(f.Nodes[1].Disp[0]-want))
				maxEnergy = math.Max(maxEnergy, math.Abs(s.KineticEnergy+s.InternalWork+s.DampingWork-s.InputWork))
				if s.Residual > s.Tolerance {
					t.Errorf("unbalanced step %d: %g > %g", s.Step, s.Residual, s.Tolerance)
				}
				return nil
			}
			if err := f.TimeHistoryNewmark(context.Background(), GroundMotion{{0, 0, 0}, {2, 0, 0}}, o); err != nil {
				t.Fatal(err)
			}
			if maxError > 8e-6 || maxEnergy > 1e-10 {
				t.Fatalf("analytic error=%g work balance=%g", maxError, maxEnergy)
			}
			t.Logf("maximum analytic displacement error=%g, work balance error=%g", maxError, maxEnergy)
		})
	}
}

func TestTimeHistoryForcedAnalyticAndStepRefinement(t *testing.T) {
	// Constant ground acceleration A=1 gives u=-(1-cos(10t))/100.
	var errorsByStep []float64
	for _, dt := range []float64{.02, .01, .005} {
		f := transientAxial()
		maxError := 0.
		o := TimeHistoryOptions{TimeStep: dt, RelativeTolerance: 1e-10, OnStep: func(s *TimeHistoryState) error {
			want := -(1 - math.Cos(10*s.Time)) / 100
			maxError = math.Max(maxError, math.Abs(f.Nodes[1].Disp[0]-want))
			nearNewton(t, "absolute acceleration", s.Acceleration[6]+1, -100*f.Nodes[1].Disp[0], 1e-8)
			return nil
		}}
		if err := f.TimeHistoryNewmark(nil, GroundMotion{{0, 1, 0}, {1, 1, 0}}, o); err != nil {
			t.Fatal(err)
		}
		errorsByStep = append(errorsByStep, maxError)
	}
	if errorsByStep[0]/errorsByStep[1] < 3.5 || errorsByStep[1]/errorsByStep[2] < 3.5 {
		t.Fatalf("not second order: %v", errorsByStep)
	}
	t.Logf("dt .02/.01/.005 maximum errors: %v", errorsByStep)
}

func TestTimeHistoryMasslessRotationAndXY(t *testing.T) {
	f := diagnosticCantilever()
	f.Output, f.Lapch, f.Pivot = io.Discard, nil, nil
	for _, n := range f.Nodes {
		for j := range n.Force {
			n.Force[j] = 0
		}
	}
	f.Nodes[1].Mass = 2
	s := f.Elems[0].Sect
	s.Exp, s.Exq = 2, 2
	for j := 0; j < 6; j++ {
		s.Yield[2*j], s.Yield[2*j+1] = 1000, -1000
	}
	o := TimeHistoryOptions{TimeStep: .005, Period1: 1, Period2: .2, Damping1: .02, Damping2: .02}
	d, err := newTimeHistoryAnalysis(f, context.Background(), GroundMotion{{0, 1, 2}, {.1, 1, 2}, {.3, 0, 0}}, o)
	if err != nil {
		t.Fatal(err)
	}
	for j := 3; j < 6; j++ {
		if d.mass[6+j] != 0 {
			t.Fatal("invented rotational mass")
		}
	}
	maxMoment := 0.
	o.OnStep = func(state *TimeHistoryState) error {
		// The zero-mass tip rotations balance moment + damping at every step.
		for j := 3; j < 6; j++ {
			cv := d.cCSR.MulV(state.Velocity)
			maxMoment = math.Max(maxMoment, math.Abs(f.Elems[0].Stress[6+j]+cv[6+j]))
		}
		return nil
	}
	if err = f.TimeHistoryNewmark(nil, d.motion, o); err != nil {
		t.Fatal(err)
	}
	if f.Nodes[1].Disp[0] == 0 || f.Nodes[1].Disp[1] == 0 || maxMoment > 1e-7 {
		t.Fatalf("XY or massless equilibrium failed: u=%v moment=%g", f.Nodes[1].Disp, maxMoment)
	}
}

func TestTimeHistoryPlasticReversalAndPreload(t *testing.T) {
	f := preloadPatternFrame()
	f.Nodes[1].Mass = 1
	f.Nodes[1].Force[2] = -12
	runLinearPreload(t, f)
	f.Lapch, f.Pivot = nil, nil
	// Low axial capacity and high vertical bending capacity permit X yielding
	// while keeping the independently applied gravity load in equilibrium.
	f.Elems[0].Sect.Yield[0], f.Elems[0].Sect.Yield[1] = 100, -100
	clone, err := f.CloneForTimeHistory()
	if err != nil {
		t.Fatal(err)
	}
	saved := f.SaveState()
	plasticEnds, positive, negative := 0, false, false
	m := GroundMotion{{0, 0, 0}, {.2, 250, 0}, {.4, 0, 0}, {.6, -250, 0}, {.8, 0, 0}, {1, 250, 0}, {1.2, 0, 0}}
	o := TimeHistoryOptions{TimeStep: .005, EquilibratePreload: true, Period1: 1, Period2: .2, Damping1: .02, Damping2: .02}
	o.OnStep = func(s *TimeHistoryState) error {
		if s.Residual > s.Tolerance {
			t.Errorf("dynamic residual %g > %g", s.Residual, s.Tolerance)
		}
		for _, p := range clone.Elems[0].Phinge {
			if p {
				plasticEnds++
			}
		}
		force := clone.Elems[0].Stress[6]
		positive = positive || force > 90
		negative = negative || force < -90
		if math.Abs(force) > 95.01 {
			t.Errorf("axial force outside yield surface: %g", force)
		}
		return nil
	}
	if err = clone.TimeHistoryNewmark(nil, m, o); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved, f.SaveState()) {
		t.Fatal("preload frame was changed")
	}
	if plasticEnds == 0 || !positive || !negative {
		t.Fatalf("no cyclic plasticity: flags=%d positive=%t negative=%t", plasticEnds, positive, negative)
	}
	// Plastic N-M coupling can change vertical displacement/reaction. Verify
	// the actual preserved vertical load by an independent global balance.
	check, err := newTimeHistoryAnalysis(f, context.Background(), m, o)
	if err != nil {
		t.Fatal(err)
	}
	cv := check.cCSR.MulV(clone.transient.Velocity)
	reference := *f.Elems[0]
	reference.Enod = make([]*Node, 2)
	for i, n := range f.Elems[0].Enod {
		c := *n
		c.Disp = make([]float64, 6)
		reference.Enod[i] = &c
	}
	transform, err := reference.TransMatrix()
	if err != nil {
		t.Fatal(err)
	}
	internal := make([]float64, 12)
	for i := range internal {
		for j := range internal {
			internal[i] += transform[j][i] * clone.Elems[0].Stress[j]
		}
	}
	nearNewton(t, "preserved vertical applied load", internal[8]+cv[8]+clone.Nodes[1].Mass*clone.transient.Acceleration[8], -12, 1e-6)
	nearNewton(t, "vertical support reaction", clone.Nodes[0].Reaction[2], internal[2]+cv[2], 1e-6)
	if clone.transient.DampingWork < 0 {
		t.Fatal("negative damping work")
	}
	state := clone.SaveState()
	copyState := clone.TimeHistoryState()
	copyState.Velocity[6]++
	if reflect.DeepEqual(copyState, clone.TimeHistoryState()) {
		t.Fatal("dynamic state accessor aliases state")
	}
	clone.Initialise()
	if clone.TimeHistoryState() != nil {
		t.Fatal("initialise retained dynamic state")
	}
	clone.RestoreState(state)
	if !reflect.DeepEqual(clone.SaveState(), state) {
		t.Fatal("dynamic state restore failed")
	}
	cond := newtonTestCondition(t)
	cond.SetInit(false)
	if err := clone.Arclm101(nil, cond); err == nil {
		t.Fatal("accepted dynamic state as a static preload")
	}
}

func TestTimeHistoryCancellationAndOutput(t *testing.T) {
	f := transientAxial()
	prefix := filepath.Join(t.TempDir(), "response")
	ctx, cancel := context.WithCancel(context.Background())
	o := TimeHistoryOptions{TimeStep: .01, Output: prefix, OutputNodes: []int{2}, SaveEvery: 1, OnStep: func(s *TimeHistoryState) error {
		if s.Step == 3 {
			cancel()
		}
		return nil
	}}
	err := f.TimeHistoryNewmark(ctx, GroundMotion{{0, 0, 0}, {1, 1, 2}}, o)
	if !errors.Is(err, context.Canceled) || f.Running() || f.transient.Step != 3 {
		t.Fatalf("cancellation: %v state=%+v", err, f.transient)
	}
	for _, suffix := range []string{".steps.csv", ".nodes.csv", "@3.otp"} {
		p := prefix + suffix
		if suffix == "@3.otp" {
			p = prefix + "@3.otp"
		}
		data, err := os.ReadFile(p)
		if err != nil || len(data) == 0 {
			t.Fatalf("output %s: %v", p, err)
		}
	}
	state := f.SaveState()
	g := transientAxial()
	if err := g.TimeHistoryNewmark(nil, GroundMotion{{0, 0, 0}, {.01, 0, 0}}, TimeHistoryOptions{Output: prefix}); err == nil {
		t.Fatal("overwrote existing response")
	}
	if g.TimeHistoryState() != nil || !reflect.DeepEqual(state, f.SaveState()) {
		t.Fatal("failed output changed state")
	}
}

func TestTimeHistoryDynamicTangentFiniteDifference(t *testing.T) {
	f := preloadPatternFrame()
	f.Nodes[1].Mass = 2
	d, err := newTimeHistoryAnalysis(f, context.Background(), GroundMotion{{0, 0, 0}, {1, 1, 2}}, TimeHistoryOptions{Period1: 1, Period2: .2, Damping1: .02, Damping2: .02})
	if err != nil {
		t.Fatal(err)
	}
	if err = d.initial(); err != nil {
		t.Fatal(err)
	}
	base := f.SaveState()
	old := f.TimeHistoryState()
	old.Velocity[6], old.Velocity[7] = .2, .003
	du := make([]float64, 12)
	du[6], du[7], du[8] = 14, .005, -.01
	r, err := d.evaluate(base, old, du, .1)
	if err != nil {
		t.Fatal(err)
	}
	k := r.material.tangent.AddMat(d.c, d.opts.Gamma/(d.opts.Beta*.1))
	for i, m := range d.mass {
		k.Add(i, i, m/(d.opts.Beta*.1*.1))
	}
	maxError := 0.
	for j := 6; j < 9; j++ {
		h := 1e-6
		plus, minus := append([]float64(nil), du...), append([]float64(nil), du...)
		plus[j] += h
		minus[j] -= h
		a, e := d.evaluate(base, old, plus, .1)
		if e != nil {
			t.Fatal(e)
		}
		b, e := d.evaluate(base, old, minus, .1)
		if e != nil {
			t.Fatal(e)
		}
		for i := 6; i < 9; i++ {
			derivative := (a.material.residual[i] - b.material.residual[i]) / (2 * h)
			error := math.Abs(derivative+k.Query(i, j)) / math.Max(1, math.Abs(k.Query(i, j)))
			maxError = math.Max(maxError, error)
		}
	}
	if maxError > 2e-6 {
		t.Fatalf("dynamic tangent finite difference error=%g", maxError)
	}
	t.Logf("maximum scaled dynamic tangent error=%g", maxError)
}

func TestTimeHistoryFailedSubdivisionRestoresAllState(t *testing.T) {
	f := transientAxial()
	d, err := newTimeHistoryAnalysis(f, context.Background(), GroundMotion{{0, 0, 0}, {.05, 0, 0}, {.1, 1, 0}}, TimeHistoryOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err = d.initial(); err != nil {
		t.Fatal(err)
	}
	saved := f.SaveState()
	// No Newton correction is allowed. The first half is in equilibrium and
	// commits, then the second half fails; the complete requested step rolls back.
	d.a.options.MaxIterations = 0
	d.a.options.MaxCutbacks = 1
	if err := d.advance(.1, 0); err == nil {
		t.Fatal("expected subdivision failure")
	}
	if !reflect.DeepEqual(saved, f.SaveState()) {
		t.Fatal("failed second half left a partial material/dynamic commit")
	}
}

func TestTimeHistoryGravityAtRestAndPreloadCopy(t *testing.T) {
	f := preloadPatternFrame()
	f.Nodes[1].Mass = 1
	f.Nodes[1].Force[2] = -12
	runLinearPreload(t, f)
	f.Lapch, f.Pivot = nil, nil
	saved := f.SaveState()
	clone, err := f.CloneForTimeHistory()
	if err != nil {
		t.Fatal(err)
	}
	o := TimeHistoryOptions{EquilibratePreload: true, TimeStep: .01, Period1: 1, Period2: .2, Damping1: .02, Damping2: .02}
	if err = clone.TimeHistoryNewmark(nil, GroundMotion{{0, 0, 0}, {.2, 0, 0}}, o); err != nil {
		t.Fatal(err)
	}
	for i, n := range clone.Nodes {
		for j, x := range n.Disp {
			nearNewton(t, "gravity at rest", x, f.Nodes[i].Disp[j], 1e-10)
		}
	}
	for _, v := range clone.transient.Velocity {
		nearNewton(t, "rest velocity", v, 0, 1e-10)
	}
	nearNewton(t, "gravity at rest reaction", clone.Nodes[0].Reaction[2], 12, 1e-10)
	if !reflect.DeepEqual(saved, f.SaveState()) {
		t.Fatal("static preload copy was modified")
	}
}

func TestTimeHistoryInvalidConfigurationPreservesState(t *testing.T) {
	f := transientAxial()
	saved := f.SaveState()
	m := GroundMotion{{0, 0, 0}, {.1, 1, 0}}
	for _, o := range []TimeHistoryOptions{{Beta: 1. / 6}, {TimeStep: -1}, {RelativeTolerance: math.NaN()}, {InitialVelocity: []float64{1}}, {RotationalInertia: []float64{1}}, {Period1: 1, Period2: 1, Damping1: .02, Damping2: .02}, {OutputNodes: []int{999}}} {
		if err := f.TimeHistoryNewmark(nil, m, o); err == nil {
			t.Errorf("accepted invalid configuration: %+v", o)
		}
		if !reflect.DeepEqual(saved, f.SaveState()) || f.Running() {
			t.Fatal("invalid configuration changed the state")
		}
	}
}

func TestTimeHistoryAutomaticSubdivisionConverges(t *testing.T) {
	f := transientAxial()
	var log bytes.Buffer
	f.Output = &log
	o := TimeHistoryOptions{TimeStep: .2, Newton: NewtonOptions{MaxIterations: 1}}
	m := GroundMotion{{0, 0, 0}, {.2, 500, 0}}
	if err := f.TimeHistoryNewmark(nil, m, o); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(log.String(), "TIMEHISTORY CUTBACK:") || f.transient.Substeps < 2 || f.transient.Step != 1 {
		t.Fatalf("no successful subdivision: %s", log.String())
	}
	g := transientAxial()
	o.TimeStep = .1
	if err := g.TimeHistoryNewmark(nil, m, o); err != nil {
		t.Fatal(err)
	}
	nearNewton(t, "subdivided displacement", f.Nodes[1].Disp[0], g.Nodes[1].Disp[0], 1e-10)
	nearNewton(t, "subdivided velocity", f.transient.Velocity[6], g.transient.Velocity[6], 1e-10)
	nearNewton(t, "subdivided acceleration", f.transient.Acceleration[6], g.transient.Acceleration[6], 1e-10)
	t.Logf("requested steps=%d internal substeps=%d", f.transient.Step, f.transient.Substeps)
}

func TestTimeHistoryPlasticBeamWithMasslessRotations(t *testing.T) {
	f := diagnosticCantilever()
	f.Output, f.Lapch, f.Pivot = io.Discard, nil, nil
	for _, n := range f.Nodes {
		for j := range n.Force {
			n.Force[j] = 0
		}
	}
	f.Nodes[1].Mass = 2
	s := f.Elems[0].Sect
	s.Exp, s.Exq = 2, 2
	for j := 0; j < 6; j++ {
		s.Yield[2*j], s.Yield[2*j+1] = 100, -100
	}
	plasticSteps := 0
	o := TimeHistoryOptions{TimeStep: .005, Period1: 1, Period2: .2, Damping1: .02, Damping2: .02, OnStep: func(state *TimeHistoryState) error {
		if state.Residual > state.Tolerance {
			t.Errorf("unbalanced plastic beam: %g > %g", state.Residual, state.Tolerance)
		}
		for _, p := range f.Elems[0].Phinge {
			if p {
				plasticSteps++
			}
		}
		return nil
	}}
	if err := f.TimeHistoryNewmark(nil, GroundMotion{{0, 0, 0}, {.2, 0, 250}, {.4, 0, -250}, {.6, 0, 0}}, o); err != nil {
		t.Fatal(err)
	}
	if plasticSteps == 0 {
		t.Fatal("beam did not yield")
	}
	t.Logf("yielding end observations=%d final residual=%g tolerance=%g", plasticSteps, f.transient.Residual, f.transient.Tolerance)
}
