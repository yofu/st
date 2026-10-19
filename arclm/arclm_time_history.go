package arclm

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/yofu/st/matrix"
)

// TimeHistoryOptions configures a small-displacement, material nonlinear
// Newmark analysis. Zero Beta/Gamma select 1/4,1/2. Mass is Node.Mass on the
// three translational DOFs; optional rotational inertias are in mass*length^2.
// The existing vibration/static assembly functions are not changed.
type TimeHistoryOptions struct {
	AutoRayleigh                         bool   // obtain periods from the same K0 and M used by this run
	RayleighModes                        [2]int // zero: mode 1 and the next distinct frequency; otherwise explicit ascending ranks
	Beta, Gamma, TimeStep                float64
	Period1, Period2, Damping1, Damping2 float64
	RelativeTolerance                    float64
	Newton                               NewtonOptions
	Solver                               string
	EquilibratePreload                   bool
	BaseLoad                             []float64 // nil: use saved applied loads, or an unloaded zero state
	InitialVelocity                      []float64 // full global vector; constrained entries must be zero
	RotationalInertia                    []float64 // optional: three values per node
	MinTimeStep                          float64
	Output                               string                        // prefix for .steps.csv and .nodes.csv; empty disables file output
	OutputNodes                          []int                         // node numbers; nil records all nodes
	SaveEvery                            int                           // optional OTP snapshots; zero disables them
	OnStep                               func(*TimeHistoryState) error // called at t=0 and requested output times
}

// TimeHistoryState is copied on access and by Frame.SaveState/RestoreState.
// Velocity/Acceleration are relative to the moving ground; Ground is m/s^2.
// Work quantities use the model's force/length units (tf*m for the usual model).
type TimeHistoryState struct {
	RayleighModes                                       [2]int
	RayleighPeriods                                     [2]float64
	RayleighAlpha, RayleighEta                          float64
	Time                                                float64
	Step, Substeps, Iterations                          int
	Ground                                              [2]float64
	Velocity, Acceleration                              []float64
	Residual, Tolerance                                 float64
	KineticEnergy, InputWork, InternalWork, DampingWork float64
}

func copyTimeHistoryState(s *TimeHistoryState) *TimeHistoryState {
	if s == nil {
		return nil
	}
	c := *s
	c.Velocity = append([]float64(nil), s.Velocity...)
	c.Acceleration = append([]float64(nil), s.Acceleration...)
	return &c
}

func (f *Frame) TimeHistoryState() *TimeHistoryState { return copyTimeHistoryState(f.transient) }

// CloneForTimeHistory makes an independent analysis frame, including material
// history and original connection definitions, without copying mutexes/channels.
func (f *Frame) CloneForTimeHistory() (*Frame, error) {
	if f.Running() {
		return nil, fmt.Errorf("cannot copy a running analysis")
	}
	c := NewFrame()
	c.Output, c.Lapch, c.Pivot = f.Output, nil, nil
	sections := make(map[*Sect]*Sect)
	copySect := func(s *Sect) *Sect {
		if s == nil {
			return nil
		}
		if sections[s] == nil {
			sections[s] = s.Snapshot()
		}
		return sections[s]
	}
	for _, s := range f.Sects {
		c.Sects = append(c.Sects, copySect(s))
	}
	for _, n := range f.Nodes {
		if n == nil {
			return nil, fmt.Errorf("nil node in preload")
		}
		v := *n
		v.Coord, v.Conf, v.Force = append([]float64(nil), n.Coord...), append([]bool(nil), n.Conf...), append([]float64(nil), n.Force...)
		v.Disp, v.Reaction = append([]float64(nil), n.Disp...), append([]float64(nil), n.Reaction...)
		c.Nodes = append(c.Nodes, &v)
	}
	for _, e := range f.Elems {
		if e == nil {
			return nil, fmt.Errorf("nil element in preload")
		}
		v := *e
		v.Sect, v.Bonds = copySect(e.Sect), copyPlasticBonds(e.Bonds)
		v.Cmq, v.Stress = append([]float64(nil), e.Cmq...), append([]float64(nil), e.Stress...)
		v.Strong, v.Weak = append([]float64(nil), e.Strong...), append([]float64(nil), e.Weak...)
		v.Enod = make([]*Node, len(e.Enod))
		for i, n := range e.Enod {
			if n == nil || n.Index < 0 || n.Index >= len(c.Nodes) || f.Nodes[n.Index] != n {
				return nil, fmt.Errorf("invalid preload element end")
			}
			v.Enod[i] = c.Nodes[n.Index]
		}
		v.restorePlasticState(e.savePlasticState())
		c.Elems = append(c.Elems, &v)
	}
	c.RestoreState(f.SaveState())
	return c, nil
}

// RayleighCoefficients fits zeta(w)=alpha/(2w)+eta*w/2 at two periods.
func RayleighCoefficients(t1, t2, z1, z2 float64) (alpha, eta float64, err error) {
	for _, v := range []float64{t1, t2, z1, z2} {
		if !finitePlastic(v) {
			return 0, 0, fmt.Errorf("non-finite Rayleigh input")
		}
	}
	if t1 == 0 && t2 == 0 && z1 == 0 && z2 == 0 {
		return 0, 0, nil
	}
	if t1 <= 0 || t2 <= 0 || t1 == t2 || z1 < 0 || z2 < 0 {
		return 0, 0, fmt.Errorf("Rayleigh periods must be positive and distinct, damping ratios nonnegative")
	}
	w1, w2 := 2*math.Pi/t1, 2*math.Pi/t2
	eta = 2 * (z2*w2 - z1*w1) / (w2*w2 - w1*w1)
	alpha = 2*z1*w1 - eta*w1*w1
	if !finitePlastic(alpha) || !finitePlastic(eta) || alpha < 0 || eta < 0 {
		return 0, 0, fmt.Errorf("Rayleigh inputs produce negative or invalid damping coefficients")
	}
	return
}

type timeHistoryAnalysis struct {
	a          *plasticNewtonAnalysis
	opts       TimeHistoryOptions
	motion     GroundMotion
	mass       []float64
	c, k0      *matrix.COOMatrix
	cCSR       *matrix.SymmetricCSR
	alpha, eta float64
	minStep    float64
}

func assembleTimeHistoryElastic(a *plasticNewtonAnalysis) *matrix.COOMatrix {
	k := matrix.NewCOOMatrix(len(a.conf))
	for _, el := range a.elements {
		g := Transformation(el.elastic, el.transform)
		for i := 0; i < 12; i++ {
			row := 6*el.connection.Enod[i/6].Index + i%6
			for j := 0; j < 12; j++ {
				if g[i][j] != 0 {
					k.Add(row, 6*el.connection.Enod[j/6].Index+j%6, g[i][j])
				}
			}
		}
	}
	return k
}

func newTimeHistoryAnalysis(f *Frame, ctx context.Context, motion GroundMotion, o TimeHistoryOptions) (*timeHistoryAnalysis, error) {
	if err := motion.Validate(); err != nil {
		return nil, err
	}
	if o.Beta == 0 {
		o.Beta = .25
	}
	if o.Gamma == 0 {
		o.Gamma = .5
	}
	if o.RelativeTolerance == 0 {
		o.RelativeTolerance = 1e-8
	}
	if o.Solver == "" {
		o.Solver = "LDLT"
	}
	if o.MinTimeStep == 0 {
		o.MinTimeStep = 1e-8
	}
	// Restrict this first implementation to the unconditionally stable Newmark
	// parameter region; nonlinear convergence/accuracy still require small dt.
	if !finitePlastic(o.Beta) || !finitePlastic(o.Gamma) || o.Gamma < .5 || o.Beta < .25*math.Pow(o.Gamma+.5, 2) || !finitePlastic(o.TimeStep) || o.TimeStep < 0 || !finitePlastic(o.RelativeTolerance) || o.RelativeTolerance <= 0 || !finitePlastic(o.MinTimeStep) || o.MinTimeStep <= 0 || o.SaveEvery < 0 {
		return nil, fmt.Errorf("invalid Newmark/time step/tolerance options")
	}
	cond := NewAnalysisCondition()
	cond.SetEps(o.RelativeTolerance)
	cond.SetNewtonOptions(o.Newton)
	cond.SetSolver(o.Solver)
	a, err := newPlasticNewtonAnalysis(f, ctx, cond)
	if err != nil {
		return nil, err
	}
	if f.transient != nil {
		return nil, fmt.Errorf("time history requires a static preload; start a new independent frame from L")
	}
	d := &timeHistoryAnalysis{a: a, opts: o, motion: append(GroundMotion(nil), motion...), mass: make([]float64, len(a.conf)), minStep: o.MinTimeStep}
	if err := validateTimeHistoryRayleigh(o); err != nil {
		return nil, err
	}
	if o.BaseLoad != nil {
		if len(o.BaseLoad) != len(a.conf) {
			return nil, fmt.Errorf("invalid base load dimensions")
		}
		a.baseNodal = append([]float64(nil), o.BaseLoad...)
	} else if f.loading != nil {
		a.baseNodal = f.loading.appliedNodal()
	} else {
		// A zero initial model is unambiguous; nonzero preload must have actual
		// load provenance rather than inferring applied loads from member stress.
		for _, n := range f.Nodes {
			for _, v := range append(append([]float64(nil), n.Force...), n.Disp...) {
				if v != 0 {
					return nil, fmt.Errorf("static applied-load history unavailable; rerun L")
				}
			}
		}
		for _, e := range f.Elems {
			for _, v := range append(append([]float64(nil), e.Stress...), e.Cmq...) {
				if v != 0 {
					return nil, fmt.Errorf("static applied-load history unavailable; rerun L")
				}
			}
		}
	}
	for _, v := range a.baseNodal {
		if !finitePlastic(v) {
			return nil, fmt.Errorf("invalid base load")
		}
	}
	a.patternNodal = make([]float64, len(a.conf))
	if len(o.RotationalInertia) != 0 && len(o.RotationalInertia) != 3*len(f.Nodes) {
		return nil, fmt.Errorf("invalid rotational inertia dimensions")
	}
	positive := 0
	for i, n := range f.Nodes {
		if !finitePlastic(n.Mass) || n.Mass < 0 {
			return nil, fmt.Errorf("invalid node mass: %d", n.Num)
		}
		for j := 0; j < 6; j++ {
			v := n.Mass
			if j >= 3 {
				v = 0
				if len(o.RotationalInertia) > 0 {
					v = o.RotationalInertia[3*i+j-3]
				}
			}
			if !finitePlastic(v) || v < 0 {
				return nil, fmt.Errorf("invalid rotational inertia: %d", n.Num)
			}
			d.mass[6*i+j] = v
			if v > 0 && !a.conf[6*i+j] {
				positive++
			}
		}
	}
	if positive == 0 {
		return nil, fmt.Errorf("no positive mass on free DOFs")
	}
	d.k0 = assembleTimeHistoryElastic(a)
	if o.AutoRayleigh {
		if err := d.resolveRayleighModes(); err != nil {
			return nil, err
		}
	}
	d.alpha, d.eta, err = RayleighCoefficients(d.opts.Period1, d.opts.Period2, d.opts.Damping1, d.opts.Damping2)
	if err != nil {
		return nil, err
	}
	d.c = matrix.NewCOOMatrix(len(a.conf)).AddMat(d.k0, d.eta)
	for i, m := range d.mass {
		if m*d.alpha != 0 {
			d.c.Add(i, i, m*d.alpha)
		}
	}
	d.cCSR, err = d.c.ToSymmetricCSR(0, make([]bool, len(a.conf)))
	return d, err
}

func (d *timeHistoryAnalysis) external(t float64) ([]float64, GroundMotionSample, error) {
	p, err := d.motion.At(t)
	if err != nil {
		return nil, p, err
	}
	f := append([]float64(nil), d.a.baseNodal...)
	for i, m := range d.mass {
		if i%6 == 0 {
			f[i] -= m * p.X
		}
		if i%6 == 1 {
			f[i] -= m * p.Y
		}
	}
	return f, p, nil
}

func (d *timeHistoryAnalysis) kinetic(v []float64) float64 {
	s := 0.
	for i, x := range v {
		s += .5 * d.mass[i] * x * x
	}
	return s
}

func (d *timeHistoryAnalysis) initial() error {
	a, f := d.a, d.a.frame
	if d.opts.EquilibratePreload {
		if err := a.step(0, 0); err != nil {
			return fmt.Errorf("time history preload equilibrium: %w", err)
		}
	}
	base := f.SaveState()
	r, err := a.evaluate(base, make([]float64, len(a.conf)), 0, 0)
	if err != nil {
		return err
	}
	v := make([]float64, len(a.conf))
	if d.opts.InitialVelocity != nil {
		if len(d.opts.InitialVelocity) != len(v) {
			return fmt.Errorf("invalid initial velocity dimensions")
		}
		copy(v, d.opts.InitialVelocity)
	}
	for i, x := range v {
		if !finitePlastic(x) || a.conf[i] && x != 0 {
			return fmt.Errorf("invalid initial velocity at DOF %d", i)
		}
	}
	fext, p, err := d.external(0)
	if err != nil {
		return err
	}
	cv := d.cCSR.MulV(v)
	acc := make([]float64, len(v))
	imbalance := make([]float64, len(v))
	for i := range v {
		if !a.conf[i] {
			if d.mass[i] > 0 {
				acc[i] = (fext[i] - r.internal[i] - cv[i]) / d.mass[i]
			} else {
				imbalance[i] = fext[i] - r.internal[i] - cv[i]
			}
		}
	}
	tol := a.options.AbsoluteForceTolerance + a.cond.eps*math.Max(a.norm(fext), a.norm(r.internal))
	if a.norm(imbalance) > tol {
		return fmt.Errorf("initial zero-mass DOFs are not in equilibrium: residual=%g tolerance=%g", a.norm(imbalance), tol)
	}
	// Determine the algebraic accelerations without dividing by zero mass.
	// For stiffness-proportional damping differentiate its algebraic velocity
	// equation; without it use the differentiated stiffness constraint at rest.
	zeroConf := append([]bool(nil), a.conf...)
	zeroSize := a.csize
	for i, m := range d.mass {
		if !zeroConf[i] && m > 0 {
			zeroConf[i] = true
			zeroSize++
		}
	}
	if zeroSize < len(v) {
		op := r.tangent
		if d.eta > 0 {
			op = d.k0
		} else {
			for _, x := range v {
				if x != 0 {
					return fmt.Errorf("nonzero initial velocity with massless DOFs requires stiffness-proportional damping")
				}
			}
		}
		csr, e := op.ToSymmetricCSR(0, make([]bool, len(v)))
		if e != nil {
			return e
		}
		rhs := csr.MulV(acc)
		for i := range rhs {
			rhs[i] = -rhs[i]
		}
		if d.eta > 0 {
			kt, e := r.tangent.ToSymmetricCSR(0, make([]bool, len(v)))
			if e != nil {
				return e
			}
			kv := kt.MulV(v)
			for i := range rhs {
				rhs[i] -= kv[i] / d.eta
			}
		}
		z := *a
		z.conf, z.csize = zeroConf, zeroSize
		az, e := z.solveLinear(op, rhs)
		if e != nil {
			return fmt.Errorf("initial massless acceleration: %w", e)
		}
		for i := range acc {
			if !zeroConf[i] {
				acc[i] = az[i]
			}
		}
	}
	initialResidual := make([]float64, len(v))
	for i := range v {
		if !finitePlastic(acc[i]) {
			return fmt.Errorf("non-finite initial acceleration at DOF %d", i)
		}
		initialResidual[i] = fext[i] - r.internal[i] - cv[i] - d.mass[i]*acc[i]
	}
	initialNorm := a.norm(initialResidual)
	if !finitePlastic(initialNorm) || !finitePlastic(tol) || !finitePlastic(d.kinetic(v)) || initialNorm > tol {
		return fmt.Errorf("invalid initial dynamic equilibrium: residual=%g tolerance=%g", initialNorm, tol)
	}
	f.transient = &TimeHistoryState{RayleighModes: d.opts.RayleighModes, RayleighPeriods: [2]float64{d.opts.Period1, d.opts.Period2}, RayleighAlpha: d.alpha, RayleighEta: d.eta, Ground: [2]float64{p.X, p.Y}, Velocity: v, Acceleration: acc, KineticEnergy: d.kinetic(v), Residual: initialNorm, Tolerance: tol}
	for i, n := range f.Nodes {
		for j := 0; j < 6; j++ {
			q := 6*i + j
			n.Reaction[j] = 0
			if a.conf[q] {
				n.Reaction[j] = r.internal[q] + cv[q] + d.mass[q]*acc[q] - fext[q]
			}
		}
	}
	f.loading = nil
	f.plasticLoadValid = false
	return nil
}

type timeHistoryTrial struct {
	material                                  *plasticNewtonTrial
	velocity, acceleration, damping, external []float64
	ground                                    GroundMotionSample
	norm, tol                                 float64
}

func (d *timeHistoryAnalysis) evaluate(base *FrameState, old *TimeHistoryState, du []float64, target float64) (*timeHistoryTrial, error) {
	a := d.a
	dt := target - old.Time
	beta, gamma := d.opts.Beta, d.opts.Gamma
	r, err := a.evaluate(base, du, 0, 0)
	if err != nil {
		return nil, err
	}
	v, acc := make([]float64, len(du)), make([]float64, len(du))
	for i, x := range du {
		if !a.conf[i] {
			acc[i] = (x - dt*old.Velocity[i] - dt*dt*(.5-beta)*old.Acceleration[i]) / (beta * dt * dt)
			v[i] = old.Velocity[i] + dt*((1-gamma)*old.Acceleration[i]+gamma*acc[i])
		}
	}
	cv := d.cCSR.MulV(v)
	ext, p, err := d.external(target)
	if err != nil {
		return nil, err
	}
	ma := make([]float64, len(du))
	for i := range du {
		ma[i] = d.mass[i] * acc[i]
		r.residual[i] = ext[i] - r.internal[i] - cv[i] - ma[i]
	}
	norm := a.norm(r.residual)
	ref := math.Max(math.Max(a.norm(ext), a.norm(r.internal)), math.Max(a.norm(cv), a.norm(ma)))
	tol := a.options.AbsoluteForceTolerance + a.cond.eps*ref
	if !finitePlastic(norm) || !finitePlastic(tol) {
		return nil, fmt.Errorf("non-finite dynamic residual")
	}
	return &timeHistoryTrial{r, v, acc, cv, ext, p, norm, tol}, nil
}

func (d *timeHistoryAnalysis) step(target float64) error {
	a, f := d.a, d.a.frame
	old := f.transient
	dt := target - old.Time
	if dt <= 0 {
		return fmt.Errorf("invalid time step")
	}
	base := &FrameState{Disp: make([][]float64, len(f.Nodes)), Stress: make([][]float64, len(f.Elems))}
	for i, n := range f.Nodes {
		base.Disp[i] = n.Disp
	}
	for i, e := range f.Elems {
		base.Stress[i] = e.Stress
	}
	du := make([]float64, len(a.conf))
	for i := range du {
		if !a.conf[i] {
			du[i] = dt*old.Velocity[i] + dt*dt*(.5-d.opts.Beta)*old.Acceleration[i]
		}
	}
	var trial *timeHistoryTrial
	var err error
	for iteration := 0; iteration <= a.options.MaxIterations; iteration++ {
		if err = a.ctx.Err(); err != nil {
			return err
		}
		trial, err = d.evaluate(base, old, du, target)
		if err != nil {
			return err
		}
		if trial.norm <= trial.tol {
			return d.commit(base, old, du, target, iteration, trial)
		}
		if iteration == a.options.MaxIterations {
			break
		}
		k := trial.material.tangent.AddMat(d.c, d.opts.Gamma/(d.opts.Beta*dt))
		for i, m := range d.mass {
			if m != 0 {
				k.Add(i, i, m/(d.opts.Beta*dt*dt))
			}
		}
		correction, e := a.solveLinear(k, trial.material.residual)
		if e != nil {
			return e
		}
		accepted := false
		for search, scale := 0, 1.; search < a.options.LineSearchSteps; search, scale = search+1, scale*.5 {
			candidate := make([]float64, len(du))
			for i := range du {
				candidate[i] = du[i] + scale*correction[i]
			}
			next, e := d.evaluate(base, old, candidate, target)
			if e == nil && (next.norm < trial.norm || next.norm <= next.tol) {
				du = candidate
				accepted = true
				break
			}
		}
		if !accepted {
			return fmt.Errorf("dynamic Newton line search failed at time %g: residual=%g tolerance=%g", target, trial.norm, trial.tol)
		}
	}
	return fmt.Errorf("dynamic Newton iteration limit at time %g: residual=%g tolerance=%g", target, trial.norm, trial.tol)
}

func (d *timeHistoryAnalysis) commit(base *FrameState, old *TimeHistoryState, du []float64, target float64, iterations int, r *timeHistoryTrial) error {
	a, f := d.a, d.a.frame
	previous, err := a.evaluate(base, make([]float64, len(du)), 0, 0)
	if err != nil {
		return err
	}
	ext, _, err := d.external(old.Time)
	if err != nil {
		return err
	}
	cv := d.cCSR.MulV(old.Velocity)
	s := &TimeHistoryState{Time: target, Step: old.Step, Substeps: old.Substeps + 1, Iterations: iterations, Ground: [2]float64{r.ground.X, r.ground.Y}, Velocity: r.velocity, Acceleration: r.acceleration, Residual: r.norm, Tolerance: r.tol, KineticEnergy: d.kinetic(r.velocity), InputWork: old.InputWork, InternalWork: old.InternalWork, DampingWork: old.DampingWork}
	s.RayleighModes, s.RayleighPeriods, s.RayleighAlpha, s.RayleighEta = old.RayleighModes, old.RayleighPeriods, old.RayleighAlpha, old.RayleighEta
	for i, x := range du {
		s.InputWork += .5 * (ext[i] + r.external[i]) * x
		s.InternalWork += .5 * (previous.internal[i] + r.material.internal[i]) * x
		s.DampingWork += .5 * (cv[i] + r.damping[i]) * x
	}
	for _, v := range []float64{s.KineticEnergy, s.InputWork, s.InternalWork, s.DampingWork} {
		if !finitePlastic(v) {
			return fmt.Errorf("non-finite dynamic energy/work at time %g", target)
		}
	}
	for i, response := range r.material.responses {
		if response != nil {
			f.Elems[i].commitPlasticResponse(response)
		}
	}
	for i, n := range f.Nodes {
		for j := 0; j < 6; j++ {
			q := 6*i + j
			n.Disp[j] = base.Disp[i][j] + du[q]
			n.Reaction[j] = 0
			if a.conf[q] {
				n.Reaction[j] = r.material.internal[q] + r.damping[q] + d.mass[q]*r.acceleration[q] - r.external[q]
			}
		}
	}
	f.transient = s
	return nil
}

func (d *timeHistoryAnalysis) advance(target float64, depth int) error {
	from := d.a.frame.transient.Time
	err := d.step(target)
	if err == nil || d.a.ctx.Err() != nil || errors.Is(err, context.Canceled) {
		return err
	}
	if d.a.options.DisableCutbacks || depth >= d.a.options.MaxCutbacks || (target-from)*.5 < d.minStep {
		return fmt.Errorf("time history step %g -> %g failed: %w", from, target, err)
	}
	fmt.Fprintf(d.a.writer, "TIMEHISTORY CUTBACK: FROM=%.12g TARGET=%.12g DEPTH=%d CAUSE=%v\n", from, target, depth, err)
	saved := d.a.frame.SaveState()
	mid := from + (target-from)*.5
	if e := d.advance(mid, depth+1); e != nil {
		d.a.frame.RestoreState(saved)
		return e
	}
	if e := d.advance(target, depth+1); e != nil {
		d.a.frame.RestoreState(saved)
		return e
	}
	return nil
}

// TimeHistoryNewmark performs a new transient run from a static preload. Stop
// cancels it; completed requested steps remain committed. Failed subdivided
// steps restore the complete preceding requested-step state, including v/a.
func (f *Frame) TimeHistoryNewmark(ctx context.Context, motion GroundMotion, options TimeHistoryOptions) (resultErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	if err := f.beginAnalysis(cancel); err != nil {
		cancel()
		return err
	}
	defer f.finishAnalysis()
	d, err := newTimeHistoryAnalysis(f, ctx, motion, options)
	if err != nil {
		return err
	}
	entry := f.SaveState()
	if err = d.initial(); err != nil {
		f.RestoreState(entry)
		return err
	}
	out, err := newTimeHistoryOutput(f, d.opts)
	if err != nil {
		f.RestoreState(entry)
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, out.close()) }()
	fmt.Fprintf(d.a.writer, "TIMEHISTORY: BETA=%g GAMMA=%g RAYLEIGH_MODES=%v RAYLEIGH_PERIODS=%v RAYLEIGH_ALPHA=%g RAYLEIGH_ETA=%g GEOMETRY=INITIAL MASS=TRANSLATIONAL INPUT=m/s^2\n", d.opts.Beta, d.opts.Gamma, d.opts.RayleighModes, [2]float64{d.opts.Period1, d.opts.Period2}, d.alpha, d.eta)
	emit := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := out.write(f); err != nil {
			return err
		}
		if d.opts.OnStep != nil {
			if err := d.opts.OnStep(f.TimeHistoryState()); err != nil {
				return err
			}
		}
		return nil
	}
	if err = emit(); err != nil {
		return err
	}
	end := d.motion[len(d.motion)-1].Time
	index := 1
	for step := 1; f.transient.Time < end; step++ {
		target := 0.
		if d.opts.TimeStep == 0 {
			target = d.motion[index].Time
			index++
		} else {
			target = math.Min(float64(step)*d.opts.TimeStep, end)
		}
		if !finitePlastic(target) || target <= f.transient.Time {
			return fmt.Errorf("time increment is not representable")
		}
		if err = d.advance(target, 0); err != nil {
			return err
		}
		f.transient.Step = step
		fmt.Fprintf(d.a.writer, "TIMEHISTORY CONVERGED: STEP=%d TIME=%.12g RESIDUAL=%.6e TOLERANCE=%.6e\n", step, target, f.transient.Residual, f.transient.Tolerance)
		if err = emit(); err != nil {
			return err
		}
		if f.Lapch != nil {
			select {
			case f.Lapch <- step:
			case <-ctx.Done():
				return ctx.Err()
			}
			select {
			case ack := <-f.Lapch:
				if ack != 0 {
					return context.Canceled
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return nil
}
