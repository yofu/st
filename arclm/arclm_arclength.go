package arclm

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ArcLengthOptions selects elastic finite-rotation pseudo-arclength continuation.
// The metric is sum(dTranslation^2)+LoadScale^2*dLambda^2. Rotations are
// solved in the equilibrium equations but excluded from this length metric.
// Step lengths and LoadScale use the model's length unit. Zero values select
// defaults. Nlap is the maximum number of accepted continuation points; MAX
// is deliberately not a load bound because the path can turn in load.
type ArcLengthOptions struct {
	Step, MinStep, MaxStep, LoadScale float64
	MonitorNode                       int
	// OnStep sees an equilibrated, committed state. Returning true stops cleanly.
	OnStep func(*Frame, ArcLengthPoint) (bool, error)
}

type ArcLengthPoint struct {
	Step, Iterations                                  int
	Factor, StepSize, Residual, Tolerance, Constraint float64
}

func (c *AnalysisCondition) SetArcLength(o ArcLengthOptions) {
	c.arcLength = &o
	c.nlgeometry = true
}

func arcDot(x, y, weights []float64) float64 {
	s := 0.0
	for i := range x {
		s += x[i] * y[i] * weights[i]
	}
	return s
}

// arcSystem solves the full bordered Jacobian, so a singular mechanical
// tangent alone does not prevent passage through a simple load limit point.
func arcSystem(ctx context.Context, t *geometryTrial, p, normal, weights, force []float64, constraint float64) ([]float64, error) {
	n := len(p)
	k := make([]map[int]float64, n+1)
	b := make([]float64, n+1)
	for i := 0; i < n; i++ {
		k[i] = make(map[int]float64, len(t.k[i])+1)
		for j, v := range t.k[i] {
			k[i][j] = v
		}
		k[i][n] = -p[i]
		b[i] = force[i]
	}
	k[n] = make(map[int]float64, n+1)
	for i := 0; i <= n; i++ {
		if v := normal[i] * weights[i]; v != 0 {
			k[n][i] = v
		}
	}
	b[n] = constraint
	return geometrySolve(ctx, k, b)
}

func (a *geometryAnalysis) arcLoadPattern() []float64 {
	full := append([]float64(nil), a.state.patternNodal...)
	for m, e := range a.elements {
		for i := 0; i < 12; i++ {
			full[6*e.nodes[i/6]+i%6] -= a.state.patternFixed[m][i]
		}
	}
	p := make([]float64, len(a.free))
	for i, j := range a.free {
		p[i] = full[j]
	}
	return p
}

func (a *geometryAnalysis) arcCoordinates(factor float64) []float64 {
	q := make([]float64, len(a.free)+1)
	for i, j := range a.free {
		if j%6 < 3 {
			q[i] = a.f.Nodes[j/6].Disp[j%6]
		}
	}
	q[len(a.free)] = factor
	return q
}

func (a *geometryAnalysis) arcStep(from, ds float64, previous, weights, p []float64) (point ArcLengthPoint, secant []float64, err error) {
	base := a.f.SaveState()
	state := copyGeometryState(a.state)
	success := false
	defer func() {
		if !success {
			a.f.RestoreState(base)
			a.state = state
		}
	}()
	t, err := a.evaluate(from)
	if err != nil {
		return point, nil, err
	}
	n := len(p)
	zero := make([]float64, n)
	tangent, err := arcSystem(a.ctx, t, p, previous, weights, zero, 1)
	if err != nil {
		return point, nil, err
	}
	norm := math.Sqrt(arcDot(tangent, tangent, weights))
	if !finitePlastic(norm) || norm == 0 {
		return point, nil, fmt.Errorf("arc length: invalid predictor")
	}
	sign := 1.0
	if arcDot(tangent, previous, weights) < 0 {
		sign = -1
	}
	for i := range tangent {
		tangent[i] *= sign / norm
	}
	q0 := a.arcCoordinates(from)
	a.increment(tangent[:n], ds)
	factor := from + ds*tangent[n]
	arcTol := math.Max(1e-12, ds*1e-9)
	for iter := 0; iter <= a.o.MaxIterations; iter++ {
		if err = a.ctx.Err(); err != nil {
			return point, nil, err
		}
		t, err = a.evaluate(factor)
		if err != nil {
			return point, nil, err
		}
		q := a.arcCoordinates(factor)
		for i := range q {
			q[i] -= q0[i]
		}
		g := arcDot(tangent, q, weights) - ds
		if t.norm <= t.tol && math.Abs(g) <= arcTol {
			// Reject a far-away solution even when it satisfies the normal plane.
			// This prevents Newton from silently jumping across a folded branch.
			if math.Sqrt(arcDot(q, q, weights)) > 2*ds {
				return point, nil, fmt.Errorf("arc length: correction left local path neighborhood")
			}
			a.commit(t, factor)
			point = ArcLengthPoint{Factor: factor, StepSize: ds, Residual: t.norm, Tolerance: t.tol, Constraint: g, Iterations: iter}
			success = true
			return point, q, nil
		}
		if iter == a.o.MaxIterations {
			break
		}
		res := make([]float64, n)
		for i, j := range a.free {
			res[i] = t.residual[j]
		}
		d, solveErr := arcSystem(a.ctx, t, p, tangent, weights, res, -g)
		if solveErr != nil {
			return point, nil, solveErr
		}
		a.increment(d[:n], 1)
		factor += d[n]
		if !finitePlastic(factor) {
			return point, nil, fmt.Errorf("arc length: non-finite factor")
		}
	}
	return point, nil, fmt.Errorf("arc length: corrector did not converge from %.12g with ds=%.6g", from, ds)
}

func (f *Frame) staticArcLength(cancel context.CancelFunc, c *AnalysisCondition) error {
	if c.nlmaterial {
		return fmt.Errorf("arc length: material nonlinearity is not yet supported")
	}
	o := *c.arcLength
	for _, v := range []float64{o.Step, o.MinStep, o.MaxStep, o.LoadScale} {
		if !finitePlastic(v) || v < 0 {
			return fmt.Errorf("arc length: lengths/scales must be finite and nonnegative")
		}
	}
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
	defer stop()
	a, err := newGeometryAnalysis(f, ctx, c)
	if err != nil {
		return err
	}
	if o.Step == 0 {
		o.Step = .02 * a.length
	}
	if o.MinStep == 0 {
		o.MinStep = o.Step * 1e-5
	}
	if o.MaxStep == 0 {
		o.MaxStep = o.Step
	}
	if o.LoadScale == 0 {
		o.LoadScale = .01 * a.length
	}
	if o.MinStep > o.Step || o.Step > o.MaxStep {
		return fmt.Errorf("arc length: require min step <= step <= max step")
	}
	monitor := -1
	for i, n := range f.Nodes {
		if n.Num == o.MonitorNode {
			monitor = i
		}
	}
	if o.MonitorNode != 0 && monitor < 0 {
		return fmt.Errorf("arc length: monitor node %d not found", o.MonitorNode)
	}
	entry := f.SaveState()
	from, err := a.setup()
	if err != nil {
		f.RestoreState(entry)
		return err
	}
	if c.init && c.start != 0 {
		if err = a.advance(0, c.start, 0); err != nil {
			f.RestoreState(entry)
			return err
		}
		from = c.start
	} else if err = a.step(from, from); err != nil {
		f.RestoreState(entry)
		return err
	}
	weights := make([]float64, len(a.free)+1)
	for i, j := range a.free {
		if j%6 < 3 {
			weights[i] = 1
		}
	}
	weights[len(a.free)] = o.LoadScale * o.LoadScale
	previous := make([]float64, len(weights))
	previous[len(a.free)] = 1 / o.LoadScale
	if !c.init && len(a.state.arcDirection) == len(previous) {
		copy(previous, a.state.arcDirection)
	}
	p := a.arcLoadPattern()
	loadNorm := 0.0
	for _, v := range p {
		loadNorm += v * v
	}
	if loadNorm == 0 {
		return fmt.Errorf("arc length: zero reference load")
	}
	output := "arclength.otl"
	if len(c.otp) > 0 && c.otp[0] != "" {
		output = c.otp[0]
	}
	flags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	if !c.init && a.state.steps > 0 {
		flags = os.O_CREATE | os.O_WRONLY | os.O_APPEND
	}
	curve, err := os.OpenFile(output+".arc.txt", flags, 0666)
	if err != nil {
		return err
	}
	defer curve.Close()
	info, err := curve.Stat()
	if err != nil {
		return err
	}
	if info.Size() == 0 {
		if _, err = fmt.Fprintln(curve, "step\tload_factor\tux\tuy\tuz\tds\titerations\tresidual\ttolerance\tconstraint\ttangent_positive\ttangent_zero\ttangent_negative"); err != nil {
			return err
		}
	}
	fmt.Fprintf(a.w, "ARC LENGTH: normal-plane continuation ds=%g min=%g max=%g load_scale=%g\n", o.Step, o.MinStep, o.MaxStep, o.LoadScale)
	ds := o.Step
	offset := a.state.steps
	for lap := 1; lap <= c.nlap; lap++ {
		step := offset + lap
		var point ArcLengthPoint
		var secant []float64
		for cut := 0; ; cut++ {
			point, secant, err = a.arcStep(from, ds, previous, weights, p)
			if err == nil {
				break
			}
			if errors.Is(err, context.Canceled) || a.o.DisableCutbacks || cut >= a.o.MaxCutbacks || ds/2 < o.MinStep {
				return fmt.Errorf("arc length step %d: %w", step, err)
			}
			ds *= .5
			fmt.Fprintf(a.w, "ARC CUTBACK: STEP=%d DS=%.9g: %v\n", step, ds, err)
		}
		point.Step = step
		a.state.steps++
		f.geometry.steps = a.state.steps
		from = point.Factor
		previous = secant
		a.state.arcDirection = append([]float64(nil), secant...)
		f.geometry = copyGeometryState(a.state)
		var disp [3]float64
		if monitor >= 0 {
			copy(disp[:], f.Nodes[monitor].Disp[:3])
		}
		inertia := geometryInertia{-1, -1, -1}
		trial, diagnosticErr := a.evaluate(from)
		if diagnosticErr == nil {
			inertia, diagnosticErr = geometryTangentInertia(trial.k)
		}
		if diagnosticErr != nil {
			inertia = geometryInertia{-1, -1, -1}
			fmt.Fprintf(a.w, "ARC STABILITY DIAGNOSTIC FAILED: STEP=%d: %v\n", step, diagnosticErr)
		}
		if _, err = fmt.Fprintf(curve, "%d\t%.15g\t%.15g\t%.15g\t%.15g\t%.9g\t%d\t%.9g\t%.9g\t%.9g\t%d\t%d\t%d\n", step, from, disp[0], disp[1], disp[2], ds, point.Iterations, point.Residual, point.Tolerance, point.Constraint, inertia.positive, inertia.zero, inertia.negative); err != nil {
			return err
		}
		name := strings.TrimSuffix(output, filepath.Ext(output)) + fmt.Sprintf("_ARC_%d", step) + filepath.Ext(output)
		if err = writeStaticResult(f, name); err != nil {
			return err
		}
		fmt.Fprintf(a.w, "ARC CONVERGED: STEP=%d LOAD=%.12g DS=%.6g ITER=%d RESIDUAL=%.6g ARC=%.3g\n", step, from, ds, point.Iterations, point.Residual, point.Constraint)
		if o.OnStep != nil {
			done, e := o.OnStep(f, point)
			if e != nil {
				return e
			}
			if done {
				return writeStaticResult(f, output)
			}
		}
		if f.Lapch != nil {
			select {
			case f.Lapch <- step:
			case <-ctx.Done():
				return ctx.Err()
			}
			select {
			case result := <-f.Lapch:
				if result != 0 {
					return fmt.Errorf("arc length cancelled: %w", context.Canceled)
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		if point.Iterations <= 4 {
			ds = math.Min(o.MaxStep, ds*1.2)
		} else if point.Iterations > 8 {
			ds = math.Max(o.MinStep, ds*.7)
		}
	}
	return writeStaticResult(f, output)
}
