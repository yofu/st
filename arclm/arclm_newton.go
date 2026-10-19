package arclm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/yofu/st/matrix"
)

// NewtonOptions configures Arclm101. Zero fields select defaults. EPS from
// AnalysisCondition is the relative force tolerance. Rotational residuals are
// divided by ReferenceLength before taking the norm (forces and moments must
// not be added without a length scale). Geometry is fixed at input coordinates.
type NewtonOptions struct {
	MaxIterations          int
	MaxCutbacks            int
	LineSearchSteps        int
	AbsoluteForceTolerance float64
	ReferenceLength        float64
	MinLoadIncrement       float64
	DisableCutbacks        bool
}

func (cond *AnalysisCondition) SetNewtonOptions(options NewtonOptions) {
	cond.newton = options
}

// PlasticConvergenceError means that the entire requested load step was rolled
// back. It is not a certified collapse load: singularity or local failure can
// also reflect an invalid model or an inadequate numerical step.
type PlasticConvergenceError struct {
	From, Target, Residual, Tolerance float64
	Iterations                        int
	Cause                             error
}

func (e *PlasticConvergenceError) Error() string {
	return fmt.Sprintf("Arclm101: step %.12g -> %.12g did not converge (iterations=%d residual=%.6g tolerance=%.6g): %v", e.From, e.Target, e.Iterations, e.Residual, e.Tolerance, e.Cause)
}

func (e *PlasticConvergenceError) Unwrap() error { return e.Cause }

func (frame *Frame) beginAnalysis(cancel context.CancelFunc) error {
	frame.runMu.Lock()
	defer frame.runMu.Unlock()
	if frame.running {
		return fmt.Errorf("analysis is already running")
	}
	frame.running, frame.cancel = true, cancel
	return nil
}

func (frame *Frame) finishAnalysis() {
	frame.runMu.Lock()
	cancel := frame.cancel
	frame.running, frame.cancel = false, nil
	frame.runMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

type plasticAnalysisElement struct {
	index      int
	connection *Elem
	transform  [][]float64
	elastic    [][]float64
	fixed      []float64
}

type plasticNewtonAnalysis struct {
	frame        *Frame
	ctx          context.Context
	cond         *AnalysisCondition
	options      NewtonOptions
	elements     []plasticAnalysisElement
	conf         []bool
	csize        int
	length       float64
	fixedGlobal  []float64
	baseNodal    []float64
	patternNodal []float64
	patternFixed [][]float64
	writer       io.Writer
}

type plasticNewtonTrial struct {
	responses []*PlasticResponse
	internal  []float64
	residual  []float64
	tangent   *matrix.COOMatrix
	norm, tol float64
}

func finitePlastic(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// Condense fixed-end loads by the same connection eliminations as the elastic
// stiffness. Unlike ModifyCMQ this works on copies and includes elastic springs.
func plasticElasticLoads(elem *Elem) ([][]float64, []float64, error) {
	k, err := elem.StiffMatrix()
	if err != nil {
		return nil, nil, err
	}
	q := append([]float64(nil), elem.Cmq...)
	for index, bond := range elem.Bonds {
		if bond.Num == 0 {
			continue
		}
		if k[index][index] == 0 {
			empty := q[index] == 0
			for i := 0; i < 12; i++ {
				empty = empty && k[i][index] == 0 && k[index][i] == 0
			}
			if empty {
				continue
			} // An axial brace has no bending/torsion column to release.
			return nil, nil, fmt.Errorf("ELEM %d: zero stiffness at connection %d", elem.Num, index)
		}
		k1 := bond.Value[1] * elem.Length() / (4 * elem.Sect.E * elem.Sect.Value[1])
		k2 := bond.Value[2] * elem.Length() / (4 * elem.Sect.E * elem.Sect.Value[2])
		old := copyPlasticMatrix(k)
		force := q[index]
		for i := 0; i < 12; i++ {
			spring := k2
			if i == 2 || i == 4 || i == 8 || i == 10 {
				spring = k1
			}
			factor := old[i][index] / old[index][index] / (1 + spring)
			q[i] -= factor * force
			for j := 0; j < 12; j++ {
				k[i][j] -= factor * old[index][j]
			}
		}
	}
	for i := range q {
		if !finitePlastic(q[i]) {
			return nil, nil, fmt.Errorf("ELEM %d: non-finite fixed-end force", elem.Num)
		}
		for _, v := range k[i] {
			if !finitePlastic(v) {
				return nil, nil, fmt.Errorf("ELEM %d: non-finite elastic stiffness", elem.Num)
			}
		}
	}
	return k, q, nil
}

func newPlasticNewtonAnalysis(frame *Frame, ctx context.Context, cond *AnalysisCondition) (*plasticNewtonAnalysis, error) {
	if cond == nil || cond.nlap <= 0 || !finitePlastic(cond.delta) || cond.delta == 0 || !finitePlastic(cond.start) || !finitePlastic(cond.max) || !finitePlastic(cond.eps) || cond.eps <= 0 {
		return nil, fmt.Errorf("Arclm101: invalid load steps or tolerance")
	}
	if cond.nlgeometry {
		return nil, fmt.Errorf("Arclm101: geometric nonlinearity is not implemented; material analysis uses the input geometry")
	}
	if len(cond.extra) != 0 || cond.postprocess != nil {
		return nil, fmt.Errorf("Arclm101: extra RHS and postprocess callbacks are not supported by the plastic Newton solver")
	}
	if cond.solver != "LLS" && cond.solver != "LDLT" {
		return nil, fmt.Errorf("Arclm101: supported direct solvers are LLS and LDLT, got %q", cond.solver)
	}
	o := cond.newton
	if o.MaxIterations < 0 || o.MaxCutbacks < 0 || o.LineSearchSteps < 0 || !finitePlastic(o.AbsoluteForceTolerance) || o.AbsoluteForceTolerance < 0 || !finitePlastic(o.ReferenceLength) || o.ReferenceLength < 0 || !finitePlastic(o.MinLoadIncrement) || o.MinLoadIncrement < 0 {
		return nil, fmt.Errorf("Arclm101: invalid Newton options")
	}
	if o.MaxIterations == 0 {
		o.MaxIterations = 30
	}
	if o.MaxCutbacks == 0 {
		o.MaxCutbacks = 6
	}
	if o.LineSearchSteps == 0 {
		o.LineSearchSteps = 12
	}
	if o.AbsoluteForceTolerance == 0 {
		o.AbsoluteForceTolerance = 1e-9
	}
	if o.MinLoadIncrement == 0 {
		o.MinLoadIncrement = 1e-10
	}
	a := &plasticNewtonAnalysis{frame: frame, ctx: ctx, cond: cond, options: o, conf: make([]bool, 6*len(frame.Nodes)), writer: frame.Output}
	a.fixedGlobal = make([]float64, len(a.conf))
	if a.writer == nil {
		a.writer = io.Discard
	}
	for i, node := range frame.Nodes {
		if node == nil || len(node.Coord) != 3 || len(node.Conf) != 6 || len(node.Disp) != 6 || len(node.Reaction) != 6 || len(node.Force) != 6 {
			return nil, fmt.Errorf("Arclm101: invalid node data at index %d", i)
		}
		if node.Index != i {
			return nil, fmt.Errorf("Arclm101: node %d has inconsistent index", node.Num)
		}
		for j := 0; j < 6; j++ {
			a.conf[6*i+j] = node.Conf[j]
			if node.Conf[j] {
				a.csize++
			}
			if !finitePlastic(node.Force[j]) || !finitePlastic(node.Disp[j]) {
				return nil, fmt.Errorf("Arclm101: non-finite node data at node %d", node.Num)
			}
		}
	}
	for index, elem := range frame.Elems {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if elem == nil {
			return nil, fmt.Errorf("Arclm101: nil element at index %d", index)
		}
		if !elem.IsValid {
			continue
		}
		if elem.Sect == nil || len(elem.Sect.Yield) != 12 || len(elem.Sect.Value) != 4 || len(elem.Enod) != 2 || len(elem.Bonds) != 12 || len(elem.Cmq) != 12 || len(elem.Stress) != 12 || len(elem.Phinge) != 2 {
			return nil, fmt.Errorf("ELEM %d: invalid section or element data", elem.Num)
		}
		if !finitePlastic(elem.Sect.Exp) || !finitePlastic(elem.Sect.Exq) || elem.Sect.Exp <= 1 || elem.Sect.Exq <= 1 {
			return nil, fmt.Errorf("ELEM %d: plastic exponents must be finite and greater than one", elem.Num)
		}
		for _, bond := range elem.Bonds {
			if bond == nil || len(bond.Value) != 4 {
				return nil, fmt.Errorf("ELEM %d: invalid connection", elem.Num)
			}
		}
		connection := elem.plasticConnection()
		for i, bond := range connection.Bonds {
			if bond.Num == 1 {
				continue
			}
			j := i % 6
			width := .5 * (elem.Sect.Yield[2*j] - elem.Sect.Yield[2*j+1])
			center := .5 * (elem.Sect.Yield[2*j] + elem.Sect.Yield[2*j+1])
			if !finitePlastic(width) || width <= 0 || !finitePlastic(center) {
				return nil, fmt.Errorf("ELEM %d: invalid plastic capacity at component %d", elem.Num, i)
			}
		}
		// Copies of the nodes prevent trial displacements from changing axes or
		// lengths. This is a material-only, small-displacement formulation.
		connection.Enod = make([]*Node, 2)
		for end, node := range elem.Enod {
			if node == nil || node.Index < 0 || node.Index >= len(frame.Nodes) || frame.Nodes[node.Index] != node {
				return nil, fmt.Errorf("ELEM %d: invalid end node", elem.Num)
			}
			n := *node
			n.Coord = append([]float64(nil), node.Coord...)
			n.Disp = make([]float64, 6)
			connection.Enod[end] = &n
		}
		t, err := connection.TransMatrix()
		if err != nil {
			return nil, err
		}
		k, q, err := plasticElasticLoads(connection)
		if err != nil {
			return nil, err
		}
		a.length = math.Max(a.length, connection.Length())
		a.elements = append(a.elements, plasticAnalysisElement{index, connection, t, k, q})
		for i := 0; i < 12; i++ {
			row := 6*connection.Enod[i/6].Index + i%6
			for j := 0; j < 12; j++ {
				a.fixedGlobal[row] += t[j][i] * q[j]
			}
		}
	}
	if len(a.elements) == 0 {
		return nil, fmt.Errorf("Arclm101: no active elements")
	}
	if o.ReferenceLength > 0 {
		a.length = o.ReferenceLength
	}
	if !finitePlastic(a.length) || a.length <= 0 {
		return nil, fmt.Errorf("Arclm101: invalid reference length")
	}
	a.baseNodal = make([]float64, len(a.conf))
	a.patternNodal, a.patternFixed = frame.loadPatterns()
	return a, nil
}

func (a *plasticNewtonAnalysis) norm(v []float64) float64 {
	n := 0.0
	for i, value := range v {
		if a.conf[i] {
			continue
		}
		if i%6 >= 3 {
			value /= a.length
		}
		n = math.Hypot(n, value)
	}
	return n
}

func (a *plasticNewtonAnalysis) forceTolerance(internal, external []float64, target float64) float64 {
	reference := math.Max(a.norm(external), a.norm(internal))
	reference = math.Max(reference, math.Abs(target)*a.norm(a.fixedGlobal))
	reference = math.Max(reference, a.norm(a.baseNodal))
	reference = math.Max(reference, math.Abs(target)*a.norm(a.patternNodal))
	return a.options.AbsoluteForceTolerance + a.cond.eps*reference
}

func (a *plasticNewtonAnalysis) external(target float64) []float64 {
	v := append([]float64(nil), a.baseNodal...)
	for i := range v {
		v[i] += target * a.patternNodal[i]
	}
	return v
}

func (a *plasticNewtonAnalysis) evaluate(base *FrameState, du []float64, from, target float64) (*plasticNewtonTrial, error) {
	size := len(a.conf)
	r := &plasticNewtonTrial{responses: make([]*PlasticResponse, len(a.frame.Elems)), internal: make([]float64, size), residual: make([]float64, size), tangent: matrix.NewCOOMatrix(size)}
	for _, el := range a.elements {
		if err := a.ctx.Err(); err != nil {
			return nil, err
		}
		g := make([]float64, 12)
		for end, node := range el.connection.Enod {
			copy(g[6*end:6*end+6], du[6*node.Index:6*node.Index+6])
		}
		local := matrix.MatrixVector(el.transform, g)
		trial := matrix.MatrixVector(el.elastic, local)
		for i := range trial {
			trial[i] += base.Stress[el.index][i] + (target-from)*el.fixed[i]
		}
		response, err := el.connection.returnPlasticForces(trial, el.elastic)
		if err != nil {
			return nil, err
		}
		for i := range response.Increment {
			response.Increment[i] = response.Stress[i] - base.Stress[el.index][i]
		}
		r.responses[el.index] = response
		globalK := Transformation(response.Tangent, el.transform)
		for i := 0; i < 12; i++ {
			row := 6*el.connection.Enod[i/6].Index + i%6
			for j := 0; j < 12; j++ {
				r.internal[row] += el.transform[j][i] * response.Stress[j]
				col := 6*el.connection.Enod[j/6].Index + j%6
				if globalK[i][j] != 0 {
					r.tangent.Add(row, col, globalK[i][j])
				}
			}
		}
	}
	external := a.external(target)
	for i := range a.frame.Nodes {
		for j := 0; j < 6; j++ {
			index := 6*i + j
			r.residual[index] = external[index] - r.internal[index]
		}
	}
	r.norm = a.norm(r.residual)
	r.tol = a.forceTolerance(r.internal, external, target)
	if !finitePlastic(r.norm) || !finitePlastic(r.tol) {
		return nil, fmt.Errorf("Arclm101: non-finite equilibrium residual")
	}
	return r, nil
}

func (a *plasticNewtonAnalysis) solveLinear(k *matrix.COOMatrix, rhs []float64) ([]float64, error) {
	if err := a.ctx.Err(); err != nil {
		return nil, err
	}
	reduced := make([]float64, 0, len(rhs)-a.csize)
	for i, v := range rhs {
		if !a.conf[i] {
			reduced = append(reduced, v)
		}
	}
	// Never let optional UI pivot drawing block numerical progress or Stop.
	progress := make(chan int, 32)
	drained := make(chan struct{})
	go func() {
		defer close(drained)
		for p := range progress {
			if a.frame.Pivot != nil {
				select {
				case a.frame.Pivot <- p:
				default:
				}
			}
		}
	}()
	defer func() { close(progress); <-drained }()
	var solution []float64
	if a.cond.solver == "LDLT" {
		factor, err := k.SymmetricLDLT(a.csize, a.conf, progress)
		if err != nil {
			return nil, err
		}
		_, zero, negative := factor.Inertia()
		if zero != 0 || negative != 0 {
			return nil, fmt.Errorf("non-positive plastic tangent: zero=%d negative=%d", zero, negative)
		}
		solution, err = factor.Solve(reduced)
		if err != nil {
			return nil, err
		}
	} else {
		ll := k.ToLLS(a.csize, a.conf)
		answers, err := ll.Solve(progress, reduced)
		if err != nil {
			return nil, err
		}
		_, zero, negative := ll.Sylvester()
		if zero != 0 || negative != 0 {
			return nil, fmt.Errorf("non-positive plastic tangent: zero=%d negative=%d", zero, negative)
		}
		solution = answers[0]
	}
	if err := a.ctx.Err(); err != nil {
		return nil, err
	}
	full := make([]float64, len(rhs))
	index := 0
	for i := range full {
		if !a.conf[i] {
			if !finitePlastic(solution[index]) {
				return nil, fmt.Errorf("non-finite Newton correction")
			}
			full[i] = solution[index]
			index++
		}
	}
	return full, nil
}

func (a *plasticNewtonAnalysis) commit(base *FrameState, du []float64, target float64, r *plasticNewtonTrial) {
	for i, response := range r.responses {
		if response != nil {
			a.frame.Elems[i].commitPlasticResponse(response)
		}
	}
	external := a.external(target)
	for i, node := range a.frame.Nodes {
		for j := 0; j < 6; j++ {
			node.Disp[j] = base.Disp[i][j] + du[6*i+j]
			node.Reaction[j] = 0
			if node.Conf[j] {
				node.Reaction[j] = r.internal[6*i+j] - external[6*i+j]
			}
		}
	}
	a.frame.plasticLoad, a.frame.plasticLoadValid = target, true
	steps := 0
	if old := a.frame.loading; old != nil && old.samePattern(a.patternNodal, a.patternFixed) {
		steps = old.steps
	}
	a.frame.loading = copyPlasticLoading(&plasticLoadState{steps: steps, factor: target, nodalPattern: a.patternNodal, fixedPattern: a.patternFixed, baseNodal: a.baseNodal})
}

func (a *plasticNewtonAnalysis) step(from, target float64) error {
	// Trial evaluation never modifies the frame. Reference the committed
	// arrays until the one final commit instead of deep-copying all connection
	// definitions and tangent caches on every Newton step.
	base := &FrameState{Disp: make([][]float64, len(a.frame.Nodes)), Stress: make([][]float64, len(a.frame.Elems))}
	for i, node := range a.frame.Nodes {
		base.Disp[i] = node.Disp
	}
	for i, elem := range a.frame.Elems {
		base.Stress[i] = elem.Stress
	}
	du := make([]float64, len(a.conf))
	r, err := a.evaluate(base, du, from, target)
	failure := &PlasticConvergenceError{From: from, Target: target, Cause: err}
	for iteration := 0; iteration <= a.options.MaxIterations && err == nil; iteration++ {
		failure.Iterations, failure.Residual, failure.Tolerance = iteration, r.norm, r.tol
		fmt.Fprintf(a.writer, "ARCLM101 NEWTON: LOAD=%.12g ITER=%d RESIDUAL=%.6e TOL=%.6e\n", target, iteration, r.norm, r.tol)
		if r.norm <= r.tol {
			if err := a.ctx.Err(); err != nil {
				return err
			}
			a.commit(base, du, target, r)
			return nil
		}
		if iteration == a.options.MaxIterations {
			err = fmt.Errorf("Newton iteration limit")
			break
		}
		correction, solveErr := a.solveLinear(r.tangent, r.residual)
		if solveErr != nil {
			err = solveErr
			break
		}
		accepted := false
		for line, scale := 0, 1.0; line < a.options.LineSearchSteps; line, scale = line+1, scale*.5 {
			next := append([]float64(nil), du...)
			for i := range next {
				next[i] += scale * correction[i]
			}
			trial, trialErr := a.evaluate(base, next, from, target)
			if errors.Is(trialErr, context.Canceled) {
				return trialErr
			}
			if trialErr == nil && (trial.norm <= trial.tol || trial.norm < (1-1e-4*scale)*r.norm) {
				du, r, accepted = next, trial, true
				break
			}
		}
		if !accepted {
			err = fmt.Errorf("Newton line search failed to reduce the equilibrium residual")
			break
		}
	}
	if errors.Is(err, context.Canceled) {
		return err
	}
	failure.Cause = err
	return failure
}

func (a *plasticNewtonAnalysis) advance(from, target float64, depth int) error {
	err := a.step(from, target)
	original := err
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return err
	}
	if a.options.DisableCutbacks || depth >= a.options.MaxCutbacks || math.Abs(target-from)*.5 < a.options.MinLoadIncrement {
		return err
	}
	mid := from + (target-from)*.5
	if mid == from || mid == target {
		return err
	}
	// A failed trial has not committed anything. Only actual cutbacks need
	// a full snapshot, because their first half may commit before the second fails.
	base := a.frame.SaveState()
	fmt.Fprintf(a.writer, "ARCLM101 CUTBACK: FROM=%.12g TARGET=%.12g MID=%.12g DEPTH=%d\n", from, target, mid, depth+1)
	if err = a.advance(from, mid, depth+1); err == nil {
		err = a.advance(mid, target, depth+1)
	}
	if err != nil {
		a.frame.RestoreState(base)
		if errors.Is(err, context.Canceled) {
			return err
		}
		failure := &PlasticConvergenceError{From: from, Target: target, Cause: err}
		var child *PlasticConvergenceError
		if errors.As(original, &child) {
			failure.Residual, failure.Tolerance, failure.Iterations = child.Residual, child.Tolerance, child.Iterations
		}
		return failure
	}
	return nil
}

func writePlasticStep(frame *Frame, name string, step int) error {
	if name == "" {
		name = "hogtxt.otp"
	}
	ext := filepath.Ext(name)
	name = strings.TrimSuffix(name, ext) + fmt.Sprintf("@%d", step) + ext
	w, err := os.Create(name)
	if err != nil {
		return err
	}
	_, writeErr := frame.WriteTo(w)
	closeErr := w.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

func (frame *Frame) arclm101Newton(cancel context.CancelFunc, cond *AnalysisCondition) error {
	started := time.Now()
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
	if err := frame.beginAnalysis(end); err != nil {
		stop()
		return err
	}
	defer frame.finishAnalysis()
	a, err := newPlasticNewtonAnalysis(frame, ctx, cond)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.writer, "ARCLM101: SOLVER=%s EPS=%.6e ABS_FORCE_TOL=%.6e REFERENCE_LENGTH=%.6g MAX_ITERATIONS=%d MAX_CUTBACKS=%d CUTBACKS=%t GEOMETRY=INITIAL\n", cond.solver, cond.eps, a.options.AbsoluteForceTolerance, a.length, a.options.MaxIterations, a.options.MaxCutbacks, !a.options.DisableCutbacks)
	from, preload := cond.start, cond.start
	known := !cond.init && frame.loading != nil
	newStage := false
	if cond.init {
		from = 0
	} else if known {
		loading := frame.loading
		if len(loading.baseNodal) != len(a.conf) || len(loading.nodalPattern) != len(a.conf) {
			return fmt.Errorf("Arclm101: saved applied loads have incompatible dimensions")
		}
		if loading.samePattern(a.patternNodal, a.patternFixed) {
			from, preload = loading.factor, loading.factor
			a.baseNodal = append([]float64(nil), loading.baseNodal...)
			if cond.startSet && math.Abs(cond.start-from) > 1e-10*math.Max(1, math.Abs(from)) {
				return fmt.Errorf("Arclm101: START=%g differs from the saved load factor %g", cond.start, from)
			}
		} else {
			from, newStage = 0, true
			a.baseNodal = loading.appliedNodal()
		}
	} else if frame.plasticLoadValid {
		from, preload = frame.plasticLoad, frame.plasticLoad
		if cond.startSet && math.Abs(cond.start-from) > 1e-10*math.Max(1, math.Abs(from)) {
			return fmt.Errorf("Arclm101: START=%g differs from the saved load factor %g", cond.start, from)
		}
	}
	if cond.maxSet && (cond.max-preload)*cond.delta < 0 {
		return fmt.Errorf("Arclm101: MAX lies opposite the increment direction")
	}
	var entry *FrameState
	if preload != from {
		entry = frame.SaveState()
	}
	if cond.init {
		frame.Initialise()
		// Fixed-end forces are load dependent, not a permanent initial stress.
		for _, elem := range frame.Elems {
			for i := range elem.Stress {
				elem.Stress[i] = 0
			}
		}
	}
	if known {
		fmt.Fprintf(a.writer, "ARCLM101 LOAD_STAGE: NEW_PATTERN=%t PRESERVED_NODAL_LOAD_NORM=%.6e FROM=%.12g\n", newStage, a.norm(a.baseNodal), from)
		// Preserve specified loads, not a force vector inferred from possibly
		// unbalanced stresses. Re-equilibrate a linear preload under these actual
		// loads, including its numerical residual or initial material yielding.
		if err := a.step(from, from); err != nil {
			return fmt.Errorf("Arclm101: applied preload could not be equilibrated: %w", err)
		}
	}
	if !cond.init && !known {
		// A supplied continuation state must already balance the declared loads.
		internal, residual, external := make([]float64, len(a.conf)), make([]float64, len(a.conf)), a.external(from)
		for _, el := range a.elements {
			for i := 0; i < 12; i++ {
				row := 6*el.connection.Enod[i/6].Index + i%6
				for j := 0; j < 12; j++ {
					internal[row] += el.transform[j][i] * frame.Elems[el.index].Stress[j]
				}
			}
		}
		for i := range frame.Nodes {
			for j := 0; j < 6; j++ {
				residual[6*i+j] = external[6*i+j] - internal[6*i+j]
			}
		}
		norm := a.norm(residual)
		tol := a.forceTolerance(internal, external, from)
		if !finitePlastic(norm) || norm > tol {
			return fmt.Errorf("Arclm101: continuation state is not in equilibrium at load %g (residual=%g tolerance=%g); applied-load history is unavailable, rerun the preload analysis before -noinit", from, norm, tol)
		}
		for _, el := range a.elements {
			probe := *el.connection
			probe.Stress = frame.Elems[el.index].Stress
			values, _, _, _, _ := probe.Coefficients(el.elastic)
			for _, value := range values {
				if !finitePlastic(value) || value > RADIUS+1e-8 {
					return fmt.Errorf("Arclm101: continuation stress of ELEM %d is outside the yield surface", probe.Num)
				}
			}
		}
	}
	if preload != from {
		if err := a.advance(from, preload, 0); err != nil {
			frame.RestoreState(entry)
			return err
		}
		from = preload
	}
	output := ""
	if len(cond.otp) > 0 {
		output = cond.otp[0]
	}
	origin := from
	offset := frame.ContinuationStep()
	for lap := 1; lap <= cond.nlap; lap++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		target := origin + float64(lap)*cond.delta
		if cond.maxSet {
			if cond.delta > 0 {
				target = math.Min(target, cond.max)
			} else {
				target = math.Max(target, cond.max)
			}
			if target == from {
				break
			}
		}
		if !finitePlastic(target) || target == from {
			return fmt.Errorf("Arclm101: load increment is not representable at load %g", from)
		}
		if err := a.advance(from, target, 0); err != nil {
			return err
		}
		step := offset + lap
		if err := writePlasticStep(frame, output, step); err != nil {
			return err
		}
		frame.loading.steps = step
		fmt.Fprintf(a.writer, "ARCLM101 CONVERGED: STEP=%d/%d LOAD=%.12g TIME=%.6fsec\n", step, offset+cond.nlap, target, time.Since(started).Seconds())
		from = target
		if frame.Lapch != nil {
			select {
			case frame.Lapch <- step:
			case <-ctx.Done():
				return ctx.Err()
			}
			select {
			case result := <-frame.Lapch:
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
