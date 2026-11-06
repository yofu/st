package arclm

import (
	"context"
	"math"
	"reflect"
	"testing"

	"github.com/yofu/st/matrix"
)

func TestTimeHistoryGeometryDynamicTangent(t *testing.T) {
	for _, elastic := range []bool{false, true} {
		f := geometryBeam(1)
		f.Sects[0].Value[0] = 10
		geometryYield(f.Sects[0], .25)
		f.Sects[0].Exp, f.Sects[0].Exq = 1.5, 1.5
		f.Sects[0].ElasticEnds = [2]bool{elastic, elastic}
		f.Nodes[1].Mass = 2
		for _, n := range f.Nodes {
			for j := range n.Conf {
				n.Conf[j] = false
			}
		}
		f.Elems[0].Cmq = []float64{.01, .02, -.03, .04, -.01, .02, -.02, .03, .02, -.02, .01, .04}
		o := TimeHistoryOptions{NonlinearGeometry: true, BaseLoad: []float64{0, 0, 0, .01, -.02, .03, 0, 0, 0, .04, .05, .02}, Period1: 1, Period2: .2, Damping1: .02, Damping2: .02}
		d, err := newTimeHistoryAnalysis(f, context.Background(), GroundMotion{{0, 0, 0}, {1, 1, 2}}, o)
		if err != nil {
			t.Fatal(err)
		}
		// Nonzero fixed dead forces, without an artificial static equilibrium.
		copy(d.geometry.state.baseFixed[0], f.Elems[0].Cmq)
		base := f.SaveState()
		old := &TimeHistoryState{Velocity: make([]float64, 12), Acceleration: make([]float64, 12)}
		du := []float64{0, 0, 0, .03, -.04, .06, .04, .13, -.05, .15, .08, -.13}
		for i := range old.Velocity {
			old.Velocity[i] = .01 * float64(i)
		}
		r, err := d.evaluate(base, old, du, .1)
		if err != nil {
			t.Fatal(err)
		}
		k := r.material.tangent.AddMat(d.c, d.opts.Gamma/(d.opts.Beta*.1))
		for i, m := range d.mass {
			k.Add(i, i, m/(d.opts.Beta*.01))
		}
		maxError := 0.
		for j := range du {
			plus, minus := append([]float64(nil), du...), append([]float64(nil), du...)
			plus[j] += 1e-6
			minus[j] -= 1e-6
			a, e := d.evaluate(base, old, plus, .1)
			if e != nil {
				t.Fatal(e)
			}
			b, e := d.evaluate(base, old, minus, .1)
			if e != nil {
				t.Fatal(e)
			}
			for i := range du {
				fd := -(a.material.residual[i] - b.material.residual[i]) / 2e-6
				rel := math.Abs(fd-k.Query(i, j)) / math.Max(1, math.Abs(fd))
				maxError = math.Max(maxError, rel)
				if rel > 3e-6 {
					t.Fatalf("elastic=%v [%d,%d] tangent=%g fd=%g", elastic, i, j, k.Query(i, j), fd)
				}
			}
		}
		if !reflect.DeepEqual(base, f.SaveState()) {
			t.Fatal("trial changed committed state")
		}
		t.Logf("elastic=%v 12x12 dynamic tangent relative error %.3g", elastic, maxError)
	}
}

func transientInclinedBar() *Frame {
	f := geometryBeam(1)
	f.Sects[0].Value = []float64{80, 0, 0, 0}
	geometryYield(f.Sects[0], 1e6)
	f.Nodes[1].Coord = []float64{.8, .6, 0}
	f.Nodes[1].Mass = 1
	for j := range f.Nodes[1].Conf {
		f.Nodes[1].Conf[j] = j != 0
	}
	return f
}

func TestTimeHistoryGeometryIndependentOscillator(t *testing.T) {
	// Independent ODE: x''=-80*(sqrt((.8+x)^2+.36)-1)*(.8+x)/length.
	// RK4 resolves that equation directly; it has no beam, rotations or Newton.
	force := func(x float64) float64 { l := math.Hypot(.8+x, .6); return 80 * (l - 1) * (.8 + x) / l }
	step := func(x, v, h float64) (float64, float64) {
		k1x, k1v := v, -force(x)
		k2x, k2v := v+h*k1v/2, -force(x+h*k1x/2)
		k3x, k3v := v+h*k2v/2, -force(x+h*k2x/2)
		k4x, k4v := v+h*k3v, -force(x+h*k3x)
		return x + h*(k1x+2*k2x+2*k3x+k4x)/6, v + h*(k1v+2*k2v+2*k3v+k4v)/6
	}
	errors := []float64{}
	for _, dt := range []float64{.02, .01, .005} {
		f := transientInclinedBar()
		f.Nodes[1].Disp[0] = .15
		v := make([]float64, 12)
		xRef, vRef, tRef := .15, 0., 0.
		maxError, maxEnergy := 0., 0.
		o := TimeHistoryOptions{NonlinearGeometry: true, BaseLoad: make([]float64, 12), InitialVelocity: v, TimeStep: dt, RelativeTolerance: 1e-11}
		e0 := 40 * math.Pow(math.Hypot(.95, .6)-1, 2)
		o.OnStep = func(s *TimeHistoryState) error {
			for tRef < s.Time-1e-12 {
				h := math.Min(.00005, s.Time-tRef)
				xRef, vRef = step(xRef, vRef, h)
				tRef += h
			}
			x := f.Nodes[1].Disp[0]
			maxError = math.Max(maxError, math.Abs(x-xRef))
			maxEnergy = math.Max(maxEnergy, math.Abs(s.KineticEnergy+40*math.Pow(math.Hypot(.8+x, .6)-1, 2)-e0))
			nearNewton(t, "dynamic force balance", s.Acceleration[6]+force(x), 0, 1e-7)
			nearNewton(t, "support reaction", f.Nodes[0].Reaction[0], -force(x), 1e-8)
			return nil
		}
		if err := f.TimeHistoryNewmark(nil, GroundMotion{{0, 0, 0}, {1, 0, 0}}, o); err != nil {
			t.Fatal(err)
		}
		errors = append(errors, maxError)
		t.Logf("dt=%g RK4 error=%g energy drift=%g", dt, maxError, maxEnergy)
	}
	if errors[0]/errors[1] < 3.5 || errors[1]/errors[2] < 3.5 || errors[2] > 1e-4 {
		t.Fatalf("time refinement %v", errors)
	}
}

func TestTimeHistoryGeometryPlasticReversal(t *testing.T) {
	f := transientInclinedBar()
	geometryYield(f.Sects[0], 1)
	maxPlastic, positive, negative := 0., false, false
	plasticReference := 0.
	o := TimeHistoryOptions{NonlinearGeometry: true, TimeStep: .005, RelativeTolerance: 1e-10}
	o.OnStep = func(s *TimeHistoryState) error {
		x := .8 + f.Nodes[1].Disp[0]
		l := math.Hypot(x, .6)
		e := f.Elems[0]
		n := e.Stress[6]
		want := math.Max(-RADIUS, math.Min(RADIUS, 80*(l-1-plasticReference)))
		nearNewton(t, "independent 1D return mapping", n, want, 1e-7)
		plasticReference = l - 1 - want/80
		maxPlastic = math.Max(maxPlastic, math.Abs(e.material.plastic[6]))
		positive = positive || n > .94
		negative = negative || n < -.94
		nearNewton(t, "physical acceleration balance", s.Acceleration[6]+s.Ground[0]+n*x/l, 0, 1e-7)
		if math.Abs(n) > RADIUS+1e-7 {
			t.Fatal("yield violation")
		}
		return nil
	}
	if err := f.TimeHistoryNewmark(nil, GroundMotion{{0, 0, 0}, {.2, -4, 0}, {.5, 4, 0}, {.8, -4, 0}, {1, 0, 0}, {1.5, 0, 0}}, o); err != nil {
		t.Fatal(err)
	}
	if maxPlastic < 1e-3 || !positive || !negative {
		t.Fatalf("plastic/reversal fixture: %g %v %v", maxPlastic, positive, negative)
	}
	t.Logf("plastic basic deformation max=%g; tension/compression yield both reached", maxPlastic)
}

func TestTimeHistoryGeometryElasticLimit(t *testing.T) {
	var histories [2][]float64
	for i, geo := range []bool{false, true} {
		f := transientAxial()
		o := TimeHistoryOptions{NonlinearGeometry: geo, TimeStep: .01, Period1: 1, Period2: .2, Damping1: .02, Damping2: .02, RelativeTolerance: 1e-11}
		o.OnStep = func(s *TimeHistoryState) error { histories[i] = append(histories[i], f.Nodes[1].Disp[0]); return nil }
		if err := f.TimeHistoryNewmark(nil, GroundMotion{{0, 0, 0}, {.1, .1, 0}, {.3, -.1, 0}, {.5, 0, 0}}, o); err != nil {
			t.Fatal(err)
		}
	}
	for i, v := range histories[0] {
		nearNewton(t, "axial elastic limit", histories[1][i], v, 2e-11)
	}
}

func TestTimeHistoryGeometryPreloadAndSpatialBalance(t *testing.T) {
	f := geometryBeam(2)
	geometryYield(f.Sects[0], 100)
	f.Sects[0].Value[0] = 100
	for i, n := range f.Nodes {
		if i > 0 {
			n.Mass = 1
			for j := range n.Conf {
				n.Conf[j] = false
			}
		}
	}
	f.Nodes[2].Force[1] = .2
	f.Nodes[2].Force[2] = -.15
	f.Nodes[2].Force[3] = .03
	c := geometryCondition(t, 10, .1)
	c.SetNlmaterial(true)
	if err := f.StaticAnalysis(nil, c); err != nil {
		t.Fatal(err)
	}
	saved := f.SaveState()
	clone, err := f.CloneForTimeHistory()
	if err != nil {
		t.Fatal(err)
	}
	motion := GroundMotion{{0, 0, 0}, {.1, .1, .2}, {.2, -.1, -.2}, {.3, 0, 0}}
	o := TimeHistoryOptions{NonlinearGeometry: true, EquilibratePreload: true, TimeStep: .01, Period1: 1, Period2: .2, Damping1: .02, Damping2: .02}
	d, err := newTimeHistoryAnalysis(clone, context.Background(), motion, o)
	if err != nil {
		t.Fatal(err)
	}
	maxResidual, maxReaction := 0., 0.
	o.OnStep = func(s *TimeHistoryState) error {
		internal := make([]float64, 6*len(clone.Nodes))
		for _, e := range clone.Elems {
			for block := 0; block < 4; block++ {
				p := geometryMV(*e.geometryAxes, geometryVector{e.Stress[3*block], e.Stress[3*block+1], e.Stress[3*block+2]})
				for j := 0; j < 3; j++ {
					internal[6*e.Enod[block/2].Index+3*(block%2)+j] += p[j]
				}
			}
		}
		cv := d.cCSR.MulV(s.Velocity)
		for i, n := range clone.Nodes {
			// Independent J^-T for damping coordinate moments.
			theta := geometryVector{n.Disp[3], n.Disp[4], n.Disp[5]}
			t2 := geometryDot(theta, theta)
			coef := 1. / 12.
			if t2 > 1e-8 {
				a := math.Sqrt(t2)
				coef = (1 - .5*a/math.Tan(.5*a)) / t2
			}
			v := geometryVector{cv[6*i+3], cv[6*i+4], cv[6*i+5]}
			cross := geometryCross(theta, v)
			cross2 := geometryCross(theta, cross)
			for k := 0; k < 3; k++ {
				cv[6*i+3+k] = v[k] + .5*cross[k] + coef*cross2[k]
			}
			for k := 0; k < 6; k++ {
				balance := internal[6*i+k] + cv[6*i+k] - n.Force[k]
				if k < 3 {
					balance += n.Mass * s.Acceleration[6*i+k]
					if k < 2 {
						balance += n.Mass * s.Ground[k]
					}
				}
				if n.Conf[k] {
					maxReaction = math.Max(maxReaction, math.Abs(balance-n.Reaction[k]))
				} else {
					maxResidual = math.Max(maxResidual, math.Abs(balance))
				}
			}
		}
		return nil
	}
	if err = clone.TimeHistoryNewmark(nil, motion, o); err != nil {
		t.Fatal(err)
	}
	if maxResidual > 1e-7 || maxReaction > 1e-10 {
		t.Fatalf("spatial residual=%g reaction error=%g", maxResidual, maxReaction)
	}
	if !reflect.DeepEqual(saved, f.SaveState()) {
		t.Fatal("original preload changed")
	}
	t.Logf("3D spatial residual=%g reaction error=%g", maxResidual, maxReaction)
}

func TestTimeHistoryGeometryNonsymmetricSolve(t *testing.T) {
	for _, values := range [][4]float64{{2, 1, 1.01, 3}, {2, 4, -5, 3}, {0, 1, -1, 0}} {
		k := matrix.NewCOOMatrix(2)
		for i := 0; i < 2; i++ {
			for j := 0; j < 2; j++ {
				k.Set(i, j, values[2*i+j])
			}
		}
		a := &plasticNewtonAnalysis{ctx: context.Background(), conf: []bool{false, false}}
		d := &timeHistoryAnalysis{geometry: &geometryAnalysis{}}
		x, err := d.solve(a, k, []float64{1, 2})
		if err != nil {
			t.Fatal(err)
		}
		y := k.MulV(0, a.conf, x)
		nearNewton(t, "full nonsymmetric row 0", y[0], 1, 1e-10)
		nearNewton(t, "full nonsymmetric row 1", y[1], 2, 1e-10)
	}
}

func TestTimeHistoryGeometryLinearPreloadRest(t *testing.T) {
	f := preloadPatternFrame()
	f.Nodes[1].Mass = 1
	f.Nodes[1].Force[2] = -12
	runLinearPreload(t, f)
	before := f.SaveState()
	clone, err := f.CloneForTimeHistory()
	if err != nil {
		t.Fatal(err)
	}
	var initial []float64
	o := TimeHistoryOptions{NonlinearGeometry: true, EquilibratePreload: true, TimeStep: .01, Period1: 1, Period2: .2, Damping1: .02, Damping2: .02}
	o.OnStep = func(s *TimeHistoryState) error {
		if s.Step == 0 {
			initial = append([]float64(nil), clone.Nodes[1].Disp...)
		}
		for j, v := range clone.Nodes[1].Disp {
			nearNewton(t, "preload at rest", v, initial[j], 1e-9)
		}
		return nil
	}
	if err = clone.TimeHistoryNewmark(nil, GroundMotion{{0, 0, 0}, {.1, 0, 0}}, o); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, f.SaveState()) {
		t.Fatal("linear L preload mutated")
	}
}

func TestTimeHistoryGeometryRollbackAndGuards(t *testing.T) {
	f := transientInclinedBar()
	d, err := newTimeHistoryAnalysis(f, context.Background(), GroundMotion{{0, 0, 0}, {.05, 0, 0}, {.1, 1, 0}}, TimeHistoryOptions{NonlinearGeometry: true})
	if err != nil {
		t.Fatal(err)
	}
	if err = d.initial(); err != nil {
		t.Fatal(err)
	}
	saved := f.SaveState()
	d.a.options.MaxIterations = 0
	d.a.options.MaxCutbacks = 1
	if err = d.advance(.1, 0); err == nil {
		t.Fatal("expected failure")
	}
	if !reflect.DeepEqual(saved, f.SaveState()) {
		t.Fatal("failed half step left state committed")
	}
	for _, o := range []TimeHistoryOptions{{NonlinearGeometry: true, RotationalInertia: []float64{1, 0, 0, 0, 0, 0}}, {}} {
		if err = f.TimeHistoryNewmark(nil, GroundMotion{{0, 0, 0}, {.1, 0, 0}}, o); err == nil {
			t.Fatal("accepted incompatible history/inertia")
		}
		if !reflect.DeepEqual(saved, f.SaveState()) {
			t.Fatal("invalid options changed state")
		}
	}
}
