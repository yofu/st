package arclm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// geometryState is committed only at equilibrium. Orientation matrices, not
// sums of rotation vectors, are the primary rotational state. CMQ is treated
// as an equivalent nodal dead load, resolved in the initial global frame.
type geometryState struct {
	rotations                         []geometryMatrix
	axes                              []geometryMatrix
	factor                            float64
	material                          bool
	steps                             int
	baseNodal, patternNodal           []float64
	baseFixed, patternFixed, rawFixed [][]float64
	definition                        []float64
}

func copyGeometryState(s *geometryState) *geometryState {
	if s == nil {
		return nil
	}
	r := *s
	r.rotations = append([]geometryMatrix(nil), s.rotations...)
	r.axes = append([]geometryMatrix(nil), s.axes...)
	r.baseNodal = append([]float64(nil), s.baseNodal...)
	r.patternNodal = append([]float64(nil), s.patternNodal...)
	r.baseFixed = copyPlasticMatrix(s.baseFixed)
	r.patternFixed = copyPlasticMatrix(s.patternFixed)
	r.rawFixed = copyPlasticMatrix(s.rawFixed)
	r.definition = append([]float64(nil), s.definition...)
	return &r
}

// Reject a continuation after editing reference geometry, sections, supports
// or connections. Reusing rotations/history with a different model is invalid.
func geometryDefinition(f *Frame) []float64 {
	r := []float64{float64(len(f.Nodes)), float64(len(f.Elems))}
	for _, n := range f.Nodes {
		r = append(r, float64(n.Num))
		r = append(r, n.Coord...)
		for _, c := range n.Conf {
			v := 0.
			if c {
				v = 1
			}
			r = append(r, v)
		}
	}
	for _, e := range f.Elems {
		r = append(r, float64(e.Num), float64(e.Enod[0].Index), float64(e.Enod[1].Index), e.Cang, e.Sect.E, e.Sect.Poi)
		r = append(r, e.Sect.Value...)
		r = append(r, e.Sect.Exp, e.Sect.Exq)
		r = append(r, e.Sect.Yield...)
		for _, disabled := range e.Sect.ElasticEnds {
			v := 0.
			if disabled {
				v = 1
			}
			r = append(r, v)
		}
		for _, b := range e.plasticConnection().Bonds {
			r = append(r, float64(b.Num))
			r = append(r, b.Value...)
		}
	}
	return r
}
func (f *Frame) restoreGeometryAxes() {
	for i, e := range f.Elems {
		e.geometryAxes = nil
		if f.geometry != nil && i < len(f.geometry.axes) {
			a := f.geometry.axes[i]
			e.geometryAxes = &a
		}
	}
}

// GeometryConvergenceError always refers to a rolled-back requested step.
type GeometryConvergenceError struct {
	From, Target, Residual, Tolerance float64
	Iterations                        int
	Cause                             error
}

func (e *GeometryConvergenceError) Error() string {
	return fmt.Sprintf("StaticAnalysis geometry: step %.12g -> %.12g did not converge (iterations=%d residual=%.6g tolerance=%.6g): %v", e.From, e.Target, e.Iterations, e.Residual, e.Tolerance, e.Cause)
}
func (e *GeometryConvergenceError) Unwrap() error { return e.Cause }

type geometryAnalysis struct {
	f        *Frame
	ctx      context.Context
	c        *AnalysisCondition
	o        NewtonOptions
	elements []geometryElement
	free     []int
	length   float64
	w        io.Writer
	state    *geometryState
}
type geometryTrial struct {
	responses                    []*geometryResponse
	internal, external, residual []float64
	k                            []map[int]float64
	norm, tol                    float64
}

func newGeometryAnalysis(f *Frame, ctx context.Context, c *AnalysisCondition) (*geometryAnalysis, error) {
	if len(c.extra) > 0 || c.postprocess != nil {
		return nil, fmt.Errorf("StaticAnalysis geometry: extra RHS and legacy postprocess callbacks are not supported; run separate load stages")
	}
	if c.solver != "LLS" && c.solver != "LDLT" {
		return nil, fmt.Errorf("StaticAnalysis geometry: select LLS or LDLT (finite-rotation equations use pivoted LU)")
	}
	if c.nlap < 1 || !finitePlastic(c.delta) || c.delta == 0 || !finitePlastic(c.start) || !finitePlastic(c.max) || !finitePlastic(c.eps) || c.eps <= 0 {
		return nil, fmt.Errorf("StaticAnalysis geometry: invalid steps or tolerance")
	}
	o := c.newton
	if o.MaxIterations < 0 || o.MaxCutbacks < 0 || o.LineSearchSteps < 0 || !finitePlastic(o.AbsoluteForceTolerance) || o.AbsoluteForceTolerance < 0 || !finitePlastic(o.ReferenceLength) || o.ReferenceLength < 0 || !finitePlastic(o.MinLoadIncrement) || o.MinLoadIncrement < 0 {
		return nil, fmt.Errorf("StaticAnalysis geometry: invalid Newton options")
	}
	if o.MaxIterations == 0 {
		o.MaxIterations = 40
	}
	if o.MaxCutbacks == 0 {
		o.MaxCutbacks = 8
	}
	if o.LineSearchSteps == 0 {
		o.LineSearchSteps = 12
	}
	if o.AbsoluteForceTolerance == 0 {
		o.AbsoluteForceTolerance = 1e-8
	}
	if o.MinLoadIncrement == 0 {
		o.MinLoadIncrement = 1e-10
	}
	a := &geometryAnalysis{f: f, ctx: ctx, c: c, o: o, w: f.Output}
	if a.w == nil {
		a.w = io.Discard
	}
	if len(f.Nodes) == 0 || len(f.Elems) == 0 {
		return nil, fmt.Errorf("StaticAnalysis geometry: empty model")
	}
	for i, n := range f.Nodes {
		if n == nil || n.Index != i || len(n.Coord) != 3 || len(n.Disp) != 6 || len(n.Force) != 6 || len(n.Conf) != 6 || len(n.Reaction) != 6 {
			return nil, fmt.Errorf("StaticAnalysis geometry: invalid node %d", i)
		}
		for _, v := range n.Coord {
			if !finitePlastic(v) {
				return nil, fmt.Errorf("non-finite node coordinates")
			}
		}
		for j := 0; j < 6; j++ {
			if !finitePlastic(n.Force[j]) || !finitePlastic(n.Disp[j]) {
				return nil, fmt.Errorf("non-finite node data")
			}
			if !n.Conf[j] {
				a.free = append(a.free, 6*i+j)
			}
		}
	}
	for i, e := range f.Elems {
		if e == nil || len(e.Enod) != 2 || len(e.Bonds) != 12 || len(e.Cmq) != 12 || len(e.Stress) != 12 {
			return nil, fmt.Errorf("invalid element %d", i)
		}
		for _, n := range e.Enod {
			if n == nil || n.Index < 0 || n.Index >= len(f.Nodes) || f.Nodes[n.Index] != n {
				return nil, fmt.Errorf("invalid connectivity for ELEM %d", e.Num)
			}
		}
		for _, v := range e.Cmq {
			if !finitePlastic(v) {
				return nil, fmt.Errorf("non-finite CMQ for ELEM %d", e.Num)
			}
		}
		if !e.IsValid || e.CheckFunc != nil {
			return nil, fmt.Errorf("ELEM %d: geometric analysis requires fixed element topology (no removal/restoration callbacks)", e.Num)
		}
		if !c.init && c.nlmaterial && f.geometry == nil && e.material != nil {
			return nil, fmt.Errorf("ELEM %d: material-only history has no finite-rotation history; restart combined analysis from the elastic preload", e.Num)
		}
		if !c.init && !c.nlmaterial && e.material != nil {
			return nil, fmt.Errorf("ELEM %d: plastic history cannot be used in elastic geometric analysis", e.Num)
		}
		el, err := newGeometryElement(e, i)
		if err != nil {
			return nil, err
		}
		if c.nlmaterial {
			k := make([][]float64, 12)
			for i := range k {
				k[i] = append([]float64(nil), el.k[i][:]...)
			}
			if _, err := el.connection.returnPlasticForces(make([]float64, 12), k); err != nil {
				return nil, err
			}
		}
		a.elements = append(a.elements, el)
		a.length += el.length
	}
	a.length /= float64(len(a.elements))
	if o.ReferenceLength > 0 {
		a.length = o.ReferenceLength
	}
	return a, nil
}

func (a *geometryAnalysis) evaluate(factor float64) (*geometryTrial, error) {
	return a.evaluateDirectional(factor, nil)
}

func (a *geometryAnalysis) evaluateDirectional(factor float64, direction []float64) (*geometryTrial, error) {
	n := 6 * len(a.f.Nodes)
	t := &geometryTrial{internal: make([]float64, n), external: make([]float64, n), residual: make([]float64, n), k: make([]map[int]float64, len(a.free))}
	var globalDirection []float64
	if direction != nil {
		globalDirection = make([]float64, n)
		for i, j := range a.free {
			globalDirection[j] = direction[i]
		}
	}
	indices := make([]int, n)
	for i := range indices {
		indices[i] = -1
	}
	for i, j := range a.free {
		indices[j] = i
		t.k[i] = make(map[int]float64)
	}
	for i := 0; i < n; i++ {
		t.external[i] = a.state.baseNodal[i] + factor*a.state.patternNodal[i]
	}
	for m := range a.elements {
		if err := a.ctx.Err(); err != nil {
			return nil, err
		}
		e := &a.elements[m]
		var material *geometryMaterialTrial
		if a.c.nlmaterial {
			material = &geometryMaterialTrial{history: a.f.Elems[e.index].material}
			for i := 0; i < 12; i++ {
				material.fixed[i] = a.state.baseFixed[m][i] + factor*a.state.patternFixed[m][i]
			}
			if globalDirection != nil {
				d := [12]float64{}
				for i := 0; i < 12; i++ {
					d[i] = globalDirection[6*e.nodes[i/6]+i%6]
				}
				material.direction = &d
			}
		}
		r, err := e.responseMaterial(a.f.Nodes, a.state.rotations, material)
		if err != nil {
			return nil, fmt.Errorf("ELEM %d: %w", a.f.Elems[e.index].Num, err)
		}
		t.responses = append(t.responses, r)
		for i := 0; i < 12; i++ {
			row := 6*e.nodes[i/6] + i%6
			t.internal[row] += r.force[i]
			t.external[row] -= a.state.baseFixed[m][i] + factor*a.state.patternFixed[m][i]
			if ri := indices[row]; ri >= 0 {
				for j := 0; j < 12; j++ {
					col := 6*e.nodes[j/6] + j%6
					if ci := indices[col]; ci >= 0 && r.tangent[i][j] != 0 {
						t.k[ri][ci] += r.tangent[i][j]
					}
				}
			}
		}
	}
	for i := range t.residual {
		t.residual[i] = t.external[i] - t.internal[i]
	}
	t.norm = geometryNorm(t.residual, a.free, a.length)
	scale := math.Max(geometryNorm(t.external, a.free, a.length), geometryNorm(t.internal, a.free, a.length))
	t.tol = a.o.AbsoluteForceTolerance + a.c.eps*scale
	if !finitePlastic(t.norm) || !finitePlastic(t.tol) {
		return nil, fmt.Errorf("non-finite residual")
	}
	return t, nil
}

// Sparse row LU with scaled partial pivoting. Spatial rotational moments give
// a nonsymmetric Jacobian away from equilibrium; symmetric LDLT/CG would drop
// that term. Indefinite equilibria are allowed, but a singular pivot is not.
func geometrySolve(ctx context.Context, k []map[int]float64, b []float64) ([]float64, error) {
	n := len(b)
	scale := make([]float64, n)
	rows := make([]map[int]float64, n)
	rhs := append([]float64(nil), b...)
	for i := 0; i < n; i++ {
		v := math.Abs(k[i][i])
		if v == 0 {
			for _, x := range k[i] {
				v = math.Max(v, math.Abs(x))
			}
		}
		if v == 0 {
			return nil, fmt.Errorf("zero stiffness at free DOF %d", i+1)
		}
		scale[i] = 1 / math.Sqrt(v)
	}
	for i := 0; i < n; i++ {
		rows[i] = make(map[int]float64, len(k[i]))
		for j, v := range k[i] {
			rows[i][j] = v * scale[i] * scale[j]
		}
		rhs[i] *= scale[i]
	}
	for p := 0; p < n; p++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		pivot := p
		big := math.Abs(rows[p][p])
		for i := p + 1; i < n; i++ {
			if x := math.Abs(rows[i][p]); x > big {
				big = x
				pivot = i
			}
		}
		if !finitePlastic(big) || big < 1e-14 {
			return nil, fmt.Errorf("singular geometric tangent at free DOF %d", p+1)
		}
		rows[p], rows[pivot] = rows[pivot], rows[p]
		rhs[p], rhs[pivot] = rhs[pivot], rhs[p]
		cols := make([]int, 0, len(rows[p]))
		for j := range rows[p] {
			if j > p {
				cols = append(cols, j)
			}
		}
		sort.Ints(cols)
		for i := p + 1; i < n; i++ {
			v := rows[i][p]
			if v == 0 {
				continue
			}
			v /= rows[p][p]
			delete(rows[i], p)
			for _, j := range cols {
				rows[i][j] -= v * rows[p][j]
			}
			rhs[i] -= v * rhs[p]
		}
	}
	x := make([]float64, n)
	for i := n - 1; i >= 0; i-- {
		v := rhs[i]
		cols := make([]int, 0, len(rows[i]))
		for j := range rows[i] {
			if j > i {
				cols = append(cols, j)
			}
		}
		sort.Ints(cols)
		for _, j := range cols {
			v -= rows[i][j] * x[j]
		}
		x[i] = v / rows[i][i]
		if !finitePlastic(x[i]) {
			return nil, fmt.Errorf("non-finite geometric solution")
		}
	}
	for i := range x {
		x[i] *= scale[i]
	}
	return x, nil
}

func (a *geometryAnalysis) increment(du []float64, alpha float64) {
	full := make([]float64, 6*len(a.f.Nodes))
	for i, j := range a.free {
		full[j] = alpha * du[i]
	}
	for i, n := range a.f.Nodes {
		for j := 0; j < 3; j++ {
			n.Disp[j] += full[6*i+j]
		}
		v := geometryVector{full[6*i+3], full[6*i+4], full[6*i+5]}
		a.state.rotations[i] = geometryMM(geometryExp(v), a.state.rotations[i])
		r := geometryLog(a.state.rotations[i])
		copy(n.Disp[3:], r[:])
	}
}
func (a *geometryAnalysis) commit(t *geometryTrial, factor float64) {
	a.state.factor = factor
	for m, e := range a.elements {
		r := t.responses[m]
		a.state.axes[m] = r.axes
		el := a.f.Elems[e.index]
		if r.material != nil {
			response := *r.material
			response.PlasticIncrement = append([]float64(nil), r.basicPlastic[:]...)
			el.commitPlasticResponse(&response)
		}
		for block := 0; block < 4; block++ {
			var p geometryVector
			for i := 0; i < 3; i++ {
				j := 3*block + i
				p[i] = r.force[j] + a.state.baseFixed[m][j] + factor*a.state.patternFixed[m][j]
			}
			for i := 0; i < 3; i++ {
				el.Stress[3*block+i] = geometryDot(r.axes[i], p)
			}
		}
		el.Energy = r.energy
	}
	for i, n := range a.f.Nodes {
		for j := 0; j < 6; j++ {
			n.Reaction[j] = 0
			if n.Conf[j] {
				n.Reaction[j] = t.internal[6*i+j] - t.external[6*i+j]
			}
		}
	}
	a.f.geometry = copyGeometryState(a.state)
	a.f.restoreGeometryAxes()
	a.f.loading = nil
	a.f.plasticLoadValid = false
	a.f.transient = nil
}
func (a *geometryAnalysis) step(from, target float64) error {
	base := a.f.SaveState()
	state := copyGeometryState(a.state)
	success := false
	defer func() {
		if !success {
			a.f.RestoreState(base)
			a.state = state
		}
	}()
	failure := &GeometryConvergenceError{From: from, Target: target, Cause: fmt.Errorf("iteration limit")}
	trialWarning := false
	t, err := a.evaluate(target)
	if err != nil {
		return err
	}
	for iter := 0; iter <= a.o.MaxIterations; iter++ {
		failure.Iterations = iter
		failure.Residual = t.norm
		failure.Tolerance = t.tol
		fmt.Fprintf(a.w, "GEOMETRY ITER: LOAD=%.12g ITER=%d RESIDUAL=%.6e TOLERANCE=%.6e\n", target, iter, t.norm, t.tol)
		converged := t.norm <= t.tol
		if converged || !trialWarning {
			trialWarning = a.checkStability(t, from, target, iter, 0, converged, trialWarning)
		}
		if converged {
			a.commit(t, target)
			success = true
			return nil
		}
		if iter == a.o.MaxIterations {
			break
		}
		du, err := a.correction(target, t)
		if err != nil {
			failure.Cause = err
			return failure
		}
		disp := make([][]float64, len(a.f.Nodes))
		for i, n := range a.f.Nodes {
			disp[i] = append([]float64(nil), n.Disp...)
		}
		rot := append([]geometryMatrix(nil), a.state.rotations...)
		accepted := false
		alpha := 1.0
		// Use full Newton steps, as in the tangent-stiffness iteration. A force
		// norm line search can stall slender beams: a rotation produces a large
		// transient axial residual that the next Newton iteration eliminates.
		// Backtrack only to stay in the admissible finite-rotation chart; failed
		// equilibrations are retried with smaller load increments below.
		move := func(scale float64) {
			for i, n := range a.f.Nodes {
				copy(n.Disp, disp[i])
				a.state.rotations[i] = rot[i]
			}
			a.increment(du, scale)
		}
		for ls := 0; ls < a.o.LineSearchSteps; ls++ {
			move(alpha)
			trial, e := a.evaluate(target)
			if errors.Is(e, context.Canceled) {
				return e
			}
			if e == nil {
				if !trialWarning && (trial.norm > t.norm || a.largeStabilityCorrection(du, alpha)) {
					// A jump can cross the unstable region while both Newton
					// endpoints are stable. Inspect the interior of the update.
					for _, fraction := range [...]float64{.5, .25, .75} {
						move(alpha * fraction)
						interior, err := a.evaluate(target)
						if errors.Is(err, context.Canceled) {
							return err
						}
						if err == nil && a.checkStability(interior, from, target, iter, fraction, false, false) {
							trialWarning = true
							break
						}
					}
					move(alpha)
				}
				t = trial
				accepted = true
				break
			}
			alpha *= .5
		}
		if !accepted {
			failure.Cause = fmt.Errorf("rotation backtracking could not find an admissible configuration")
			return failure
		}
	}
	return failure
}
func (a *geometryAnalysis) advance(from, target float64, depth int) error {
	err := a.step(from, target)
	if err == nil || errors.Is(err, context.Canceled) || a.o.DisableCutbacks || depth >= a.o.MaxCutbacks || math.Abs(target-from)/2 < a.o.MinLoadIncrement {
		return err
	}
	base := a.f.SaveState()
	state := copyGeometryState(a.state)
	mid := (from + target) / 2
	if mid == from || mid == target {
		return err
	}
	fmt.Fprintf(a.w, "GEOMETRY CUTBACK: FROM=%.12g TARGET=%.12g MID=%.12g\n", from, target, mid)
	child := a.advance(from, mid, depth+1)
	if child == nil {
		child = a.advance(mid, target, depth+1)
	}
	if child == nil {
		return nil
	}
	a.f.RestoreState(base)
	a.state = state
	if errors.Is(child, context.Canceled) {
		return child
	}
	return err
}

func (a *geometryAnalysis) setup() (float64, error) {
	f, c := a.f, a.c
	n := len(f.Nodes)
	nodal, raw := f.loadPatterns()
	fixed := make([][]float64, len(a.elements))
	for m, e := range a.elements {
		fixed[m] = append([]float64(nil), e.fixed[:]...)
	}
	s := &geometryState{rotations: make([]geometryMatrix, n), axes: make([]geometryMatrix, len(a.elements)), baseNodal: make([]float64, 6*n), patternNodal: nodal, baseFixed: make([][]float64, len(a.elements)), patternFixed: fixed, rawFixed: raw}
	s.definition = geometryDefinition(f)
	s.material = c.nlmaterial
	for i := range s.rotations {
		s.rotations[i] = geometryIdentity()
	}
	for m, e := range a.elements {
		s.baseFixed[m] = make([]float64, 12)
		s.axes[m] = e.initial
	}
	if c.init {
		f.Initialise()
	} else {
		if f.transient != nil {
			return 0, fmt.Errorf("geometric continuation requires a static state")
		}
		old := f.geometry
		if old != nil {
			if old.material && !c.nlmaterial {
				return 0, fmt.Errorf("combined history requires both nlgeom and nlmat for continuation")
			}
			if len(old.rotations) != n || len(old.axes) != len(a.elements) || !samePlasticLoadValues(old.definition, s.definition) {
				return 0, fmt.Errorf("geometric continuation model changed; restore the original model or initialize")
			}
			s.rotations = append([]geometryMatrix(nil), old.rotations...)
			match := samePlasticLoadValues(nodal, old.patternNodal) && len(raw) == len(old.rawFixed)
			for m := range raw {
				match = match && samePlasticLoadValues(raw[m], old.rawFixed[m])
			}
			if match {
				s.factor = old.factor
				s.steps = old.steps
				s.baseNodal = append([]float64(nil), old.baseNodal...)
				s.baseFixed = copyPlasticMatrix(old.baseFixed)
			} else {
				for i := range nodal {
					s.baseNodal[i] = old.baseNodal[i] + old.factor*old.patternNodal[i]
				}
				for m := range raw {
					for j := 0; j < 12; j++ {
						s.baseFixed[m][j] = old.baseFixed[m][j] + old.factor*old.patternFixed[m][j]
					}
				}
			}
		} else if l := f.loading; l != nil {
			for i, node := range f.Nodes {
				s.rotations[i] = geometryExp(geometryVector{node.Disp[3], node.Disp[4], node.Disp[5]})
			}
			if l.samePattern(nodal, raw) {
				s.factor = l.factor
				copy(s.baseNodal, l.baseNodal)
			} else {
				s.baseNodal = l.appliedNodal()
				for m, e := range a.elements {
					copyElem := *f.Elems[e.index]
					copyElem.Cmq = l.fixedPattern[m]
					oldEl, err := newGeometryElement(&copyElem, m)
					if err != nil {
						return 0, err
					}
					for j := 0; j < 12; j++ {
						s.baseFixed[m][j] = l.factor * oldEl.fixed[j]
					}
				}
			}
		} else {
			for _, node := range f.Nodes {
				for _, v := range node.Disp {
					if v != 0 {
						return 0, fmt.Errorf("geometric continuation needs saved applied-load history; rerun the preload")
					}
				}
			}
		}
	}
	if !c.init && c.startSet && math.Abs(c.start-s.factor) > 1e-10*math.Max(1, math.Abs(s.factor)) {
		return 0, fmt.Errorf("START=%g differs from saved load factor %g", c.start, s.factor)
	}
	a.state = s
	return s.factor, nil
}

func (f *Frame) staticGeometry(cancel context.CancelFunc, c *AnalysisCondition) error {
	ctx, stop := context.WithCancel(context.Background())
	var once sync.Once
	end := func() {
		once.Do(func() {
			stop()
			if cancel != nil {
				cancel()
			}
		})
	}
	if err := f.beginAnalysis(end); err != nil {
		stop()
		return err
	}
	defer f.finishAnalysis()
	a, err := newGeometryAnalysis(f, ctx, c)
	if err != nil {
		return err
	}
	entry := f.SaveState()
	from, err := a.setup()
	if err != nil {
		f.RestoreState(entry)
		return err
	}
	fmt.Fprintf(a.w, "GEOMETRY: finite compatibility, separated tangent, spatial rotations, pivoted LU; MATERIAL=%t EPS=%g ABS_FORCE_TOL=%g REFERENCE_LENGTH=%g\n", c.nlmaterial, c.eps, a.o.AbsoluteForceTolerance, a.length)
	start := from
	if c.init {
		start = c.start
	}
	if c.geometryIncremental && !c.maxSet {
		c.max = start + float64(c.nlap)*c.delta
		if !finitePlastic(c.max) {
			f.RestoreState(entry)
			return fmt.Errorf("non-finite combined target load")
		}
	}
	if (c.max-start)*c.delta < 0 {
		f.RestoreState(entry)
		return fmt.Errorf("MAX lies opposite the geometric load increment direction")
	}
	if err = a.step(from, from); err != nil {
		f.RestoreState(entry)
		return fmt.Errorf("geometric preload: %w", err)
	}
	if c.init && c.start != 0 {
		if err = a.advance(0, c.start, 0); err != nil {
			f.RestoreState(entry)
			return err
		}
		from = c.start
	}
	if (c.max-from)*c.delta < 0 {
		return fmt.Errorf("MAX lies opposite the geometric load increment direction")
	}
	origin := from
	offset := a.state.steps
	for lap := 1; lap <= c.nlap; lap++ {
		target := origin + float64(lap)*c.delta
		if c.delta > 0 {
			target = math.Min(target, c.max)
		} else {
			target = math.Max(target, c.max)
		}
		if target == from {
			break
		}
		if !finitePlastic(target) {
			return fmt.Errorf("non-finite geometric load factor")
		}
		if err = a.advance(from, target, 0); err != nil {
			return err
		}
		a.state.steps = offset + lap
		f.geometry.steps = a.state.steps
		name := "hogtxt0.otp"
		if len(c.otp) > 0 && c.otp[0] != "" {
			name = c.otp[0]
		}
		if c.geometryIncremental {
			if err := writePlasticStep(f, name, offset+lap); err != nil {
				return err
			}
		} else {
			if lap < c.nlap && target != c.max {
				ext := filepath.Ext(name)
				name = strings.TrimSuffix(name, ext) + fmt.Sprintf("_LAP_%d_%d", lap, c.nlap) + ext
			}
			w, e := os.Create(name)
			if e != nil {
				return e
			}
			_, e = f.WriteTo(w)
			ce := w.Close()
			if e != nil {
				return e
			}
			if ce != nil {
				return ce
			}
		}

		fmt.Fprintf(a.w, "GEOMETRY CONVERGED: STEP=%d LOAD=%.12g\n", offset+lap, target)
		from = target
		if f.Lapch != nil {
			select {
			case f.Lapch <- offset + lap:
			case <-ctx.Done():
				return ctx.Err()
			}
			select {
			case result := <-f.Lapch:
				if result != 0 {
					return fmt.Errorf("analysis cancelled: %w", context.Canceled)
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return nil
}
