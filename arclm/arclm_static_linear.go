package arclm

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/yofu/st/matrix"
)

// StaticResultCase returns the zero-based load case for a linear static LAP.
// Read it after receiving the LAP and before acknowledging it.
func (f *Frame) StaticResultCase() int { return f.staticResultCase }

type linearElement struct {
	dofs                     [12]int
	transform, local, global [][]float64
}

func (e linearElement) localDisplacement(u []float64) []float64 {
	var ue [12]float64
	for j, dof := range e.dofs {
		ue[j] = u[dof]
	}
	return matrix.MatrixVector(e.transform, ue[:])
}

type linearLoads struct {
	nodal []float64
	fixed [][]float64
}

// SetExtraFrames adds independent load cases evaluated with the primary
// frame's stiffness. The source frames supply full nodal loads and CMQ.
func (c *AnalysisCondition) SetExtraFrames(frames ...*Frame) { c.extraFrames = frames }

// SetFloorContact enables unilateral vertical supports at the given elevation.
func (c *AnalysisCondition) SetFloorContact(z float64) { c.floorContact = &z }

// SetCompressionLimit deactivates the given sections above the compressive
// axial force limit. The sign convention is positive in compression.
func (c *AnalysisCondition) SetCompressionLimit(limit float64, sections []int) {
	c.compressionLimit = limit
	c.compressionSections = make(map[int]bool, len(sections))
	for _, section := range sections {
		c.compressionSections[section] = true
	}
}

func (c *AnalysisCondition) hasStaticPost() bool {
	return c.postprocess != nil || c.floorContact != nil || c.compressionSections != nil
}

// linearElementGeometry isolates any coordinate changes made for a linear
// assembly while allowing eigenanalysis to use its chosen starting geometry.
func linearElementGeometry(original *Elem, reference bool) Elem {
	e := *original
	if reference {
		e.geometryAxes = nil
		a, b := *e.Enod[0], *e.Enod[1]
		a.Disp, b.Disp = make([]float64, 6), make([]float64, 6)
		e.Enod = []*Node{&a, &b}
	}
	return e
}

func newLinearElements(f *Frame, reference bool) ([]linearElement, error) {
	elems := make([]linearElement, len(f.Elems))
	for i, original := range f.Elems {
		if !original.IsValid && !reference {
			continue
		}
		e := linearElementGeometry(original, reference)
		if length := e.Length0(); length <= 0 || !finitePlastic(length) {
			return nil, fmt.Errorf("ELEM %d: invalid reference length", e.Num)
		}
		t, err := e.TransMatrix()
		if err != nil {
			return nil, err
		}
		k, err := e.StiffMatrix()
		if err != nil {
			return nil, err
		}
		if err := e.condenseConnections(k, nil); err != nil {
			return nil, err
		}
		el := &elems[i]
		el.transform, el.local, el.global = t, k, Transformation(k, t)
		for j := range el.dofs {
			el.dofs[j] = 6*e.Enod[j/6].Index + j%6
		}
	}
	return elems, nil
}

func (f *Frame) linearLoad(source *Frame, reference bool) (linearLoads, error) {
	load := linearLoads{nodal: make([]float64, 6*len(f.Nodes)), fixed: make([][]float64, len(f.Elems))}
	if source == nil || len(source.Nodes) != len(f.Nodes) {
		return load, fmt.Errorf("incompatible load case nodes")
	}
	for i, n := range source.Nodes {
		if n.Num != f.Nodes[i].Num {
			return load, fmt.Errorf("incompatible load case node %d", n.Num)
		}
		copy(load.nodal[6*i:6*i+6], n.Force)
	}
	byNumber := make(map[int]int, len(f.Elems))
	for i, e := range f.Elems {
		byNumber[e.Num] = i
	}
	for _, sourceElem := range source.Elems {
		if ((!reference || source != f) && !sourceElem.IsValid) || !slices.ContainsFunc(sourceElem.Cmq, func(v float64) bool { return v != 0 }) {
			continue
		}
		i, ok := byNumber[sourceElem.Num]
		if !ok {
			return load, fmt.Errorf("load case CMQ on missing ELEM %d", sourceElem.Num)
		}
		e := linearElementGeometry(f.Elems[i], reference)
		if e.Enod[0].Num != sourceElem.Enod[0].Num || e.Enod[1].Num != sourceElem.Enod[1].Num || e.Cang != sourceElem.Cang {
			return load, fmt.Errorf("incompatible load case CMQ axes on ELEM %d", e.Num)
		}
		e.Cmq = sourceElem.Cmq
		_, q, err := plasticElasticLoads(&e)
		if err != nil {
			return load, err
		}
		load.fixed[i] = q
	}
	return load, nil
}

func staticResultName(name string, lap, count int, final bool) string {
	if name == "" {
		name = "hogtxt0.otp"
	}
	if final {
		return name
	}
	ext := filepath.Ext(name)
	return strings.TrimSuffix(name, ext) + fmt.Sprintf("_LAP_%d_%d", lap, count) + ext
}

func writeStaticResult(f *Frame, name string) error {
	w, err := os.Create(name)
	if err != nil {
		return err
	}
	_, writeErr := f.WriteTo(w)
	closeErr := w.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

// linearResult uses the same local operator and CMQ for the RHS, element
// forces and reaction recovery. It also supplies trial forces for inactive
// elements so a unilateral member can re-enter the active set.
func (f *Frame) linearResult(elems []linearElement, load linearLoads, factor float64, conf []bool, u []float64) ([][]float64, []float64) {
	reactions := make([]float64, len(u))
	for i, p := range load.nodal {
		reactions[i] = -factor * p
	}
	trial := make([][]float64, len(elems))
	for i, el := range elems {
		if el.local == nil {
			clear(f.Elems[i].Stress)
			continue
		}
		localU := el.localDisplacement(u)
		stress := matrix.MatrixVector(el.local, localU)
		if load.fixed != nil {
			for j, q := range load.fixed[i] {
				stress[j] += factor * q
			}
		}
		trial[i] = stress
		e := f.Elems[i]
		if e.IsValid {
			copy(e.Stress, stress)
			for j, dof := range el.dofs {
				for k, force := range stress {
					reactions[dof] += el.transform[k][j] * force
				}
			}
		} else {
			clear(e.Stress)
		}
	}
	for i, n := range f.Nodes {
		copy(n.Disp, u[6*i:6*i+6])
		for j := range n.Reaction {
			dof := 6*i + j
			if !conf[dof] {
				reactions[dof] = 0
			}
			n.Reaction[j] = reactions[dof]
		}
	}
	return trial, reactions
}

func (f *Frame) linearSystem(elems []linearElement, size int) (*matrix.COOMatrix, []bool, int) {
	k := matrix.NewCOOMatrix(size)
	conf := make([]bool, size)
	csize := 0
	for i, n := range f.Nodes {
		for j, fixed := range n.Conf {
			conf[6*i+j] = fixed
			if fixed {
				csize++
			}
		}
	}
	for i, el := range elems {
		if !f.Elems[i].IsValid {
			continue
		}
		for j, row := range el.dofs {
			for m, col := range el.dofs {
				if v := el.global[j][m]; v != 0 {
					k.Add(row, col, v)
				}
			}
		}
	}
	return k, conf, csize
}

func (f *Frame) linearRHS(elems []linearElement, load linearLoads, factor float64, conf []bool, prescribed []float64) []float64 {
	rhs := make([]float64, len(load.nodal))
	for i, p := range load.nodal {
		rhs[i] = factor * p
	}
	for i, el := range elems {
		if !f.Elems[i].IsValid {
			continue
		}
		for j, dof := range el.dofs {
			if conf[dof] {
				continue
			}
			for k, q := range load.fixed[i] {
				rhs[dof] -= factor * el.transform[k][j] * q
			}
			for k, col := range el.dofs {
				rhs[dof] -= el.global[j][k] * prescribed[col]
			}
		}
	}
	return f.RemoveConf(rhs)
}

func (f *Frame) staticLinear(cancel context.CancelFunc, c *AnalysisCondition) error {
	if c.nlap < 1 || !finitePlastic(c.delta) || c.delta == 0 || !finitePlastic(c.start) || !finitePlastic(c.max) ||
		(c.floorContact != nil && !finitePlastic(*c.floorContact)) || !finitePlastic(c.compressionLimit) {
		return fmt.Errorf("StaticAnalysis: invalid load steps or POST parameters")
	}
	if c.hasStaticPost() && (len(c.extra) != 0 || len(c.extraFrames) != 0) {
		return fmt.Errorf("StaticAnalysis: POST requires separate analyses for each load case")
	}
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
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
		return err
	}
	defer f.finishAnalysis()
	accepted := f.SaveState()
	initialFloor := make([]bool, len(f.Nodes))
	for i, n := range f.Nodes {
		initialFloor[i] = n.Conf[2]
	}
	success := false
	defer func() {
		if !success {
			f.RestoreState(accepted)
		}
		if c.floorContact != nil {
			for i, n := range f.Nodes {
				if !initialFloor[i] {
					n.Conf[2] = false
				}
			}
		}
	}()
	from := c.start
	if c.init {
		f.Initialise()
	} else {
		if f.geometry != nil || f.transient != nil || (f.loading != nil && f.loading.steps != 0) {
			return fmt.Errorf("StaticAnalysis: saved state requires its original nonlinear solver")
		}
		nodal, fixed := f.loadPatterns()
		if f.loading != nil && f.loading.samePattern(nodal, fixed) {
			if c.startSet && math.Abs(c.start-f.loading.factor) > 1e-10*math.Max(1, math.Abs(f.loading.factor)) {
				return fmt.Errorf("StaticAnalysis: START differs from the saved load factor")
			}
			from = f.loading.factor
		} else {
			for _, n := range f.Nodes {
				if slices.ContainsFunc(n.Disp, func(v float64) bool { return v != 0 }) ||
					slices.ContainsFunc(n.Reaction, func(v float64) bool { return v != 0 }) {
					return fmt.Errorf("StaticAnalysis: -noinit requires a saved linear state with the same load pattern")
				}
			}
		}
	}
	if (c.max-from)*c.delta < 0 {
		return fmt.Errorf("StaticAnalysis: MAX lies opposite the load increment direction")
	}
	elems, err := newLinearElements(f, true)
	if err != nil {
		return err
	}
	mainLoad, err := f.linearLoad(f, true)
	if err != nil {
		return err
	}
	loads := []linearLoads{mainLoad}
	for _, source := range c.extraFrames {
		load, err := f.linearLoad(source, true)
		if err != nil {
			return err
		}
		loads = append(loads, load)
	}
	names := slices.Clone(c.otp)
	for len(names) < len(loads)+len(c.extra) {
		name := fmt.Sprintf("hogtxt%d.otp", len(names))
		if len(names) != 0 && names[0] != "" {
			ext := filepath.Ext(names[0])
			name = strings.TrimSuffix(names[0], ext) + fmt.Sprintf("_%02d", len(names)) + ext
		}
		names = append(names, name)
	}
	c.otp = names
	started := time.Now()
	log := func(s string) {
		if f.Output != nil {
			fmt.Fprintf(f.Output, "%s: %fs\n", s, time.Since(started).Seconds())
		}
	}
	var solver Solver
	switch c.solver {
	case "PCG":
		solver = llsPCG(ctx, c.eps, log)
	case "CG":
		solver = LLS_CG(c.eps, log)
	case "CRS":
		solver = CRS_CG(c.eps, log)
	}
	const pivotTolerance = 1e-12
	// The direct solver checks its own factorization. Iterative solvers
	// need a complete pivot check once per set of supports/active elements.
	stiffnessChecked := false

	if c.init {
		for _, e := range f.Elems {
			if c.compressionSections[e.Sect.Num] {
				e.IsValid = true
			}
		}
	}
	publish := func(lap int) error {
		if f.Lapch == nil {
			return ctx.Err()
		}
		select {
		case f.Lapch <- lap:
		case <-ctx.Done():
			return ctx.Err()
		}
		// The consumer acknowledges a delivered LAP before it waits on Endch.
		if <-f.Lapch != 0 {
			return fmt.Errorf("analysis cancelled: %w", context.Canceled)
		}
		return ctx.Err()
	}
	delta := c.delta
	for lap := 1; ; lap++ {
		target := linearTarget(from, delta, c.max)
		for retry := 0; ; retry++ {
			if target == from && target != c.max {
				return fmt.Errorf("StaticAnalysis: load increment is not representable at %g", from)
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if retry >= 50 {
				return fmt.Errorf("StaticAnalysis: POST did not settle at LAP %d (load %g)", lap, target)
			}
			prescribed := make([]float64, 6*len(f.Nodes))
			if c.floorContact != nil {
				for i, n := range f.Nodes {
					if n.Conf[2] && !initialFloor[i] {
						prescribed[6*i+2] = *c.floorContact - n.Coord[2]
					}
				}
			}
			k, conf, csize := f.linearSystem(elems, len(prescribed))
			factors := []float64{target}
			rhs := [][]float64{f.linearRHS(elems, mainLoad, target, conf, prescribed)}
			final := lap >= c.nlap && target == c.max
			if final {
				for _, load := range loads[1:] {
					rhs = append(rhs, f.linearRHS(elems, load, 1, conf, prescribed))
					factors = append(factors, 1)
				}
				for _, extra := range c.extra {
					if len(extra) != len(conf)-csize {
						return fmt.Errorf("StaticAnalysis: incompatible extra RHS size")
					}
					rhs = append(rhs, extra)
					factors = append(factors, 1)
				}
			}
			answers := make([][]float64, len(rhs))
			if csize < len(conf) {
				if solver.solve == nil {
					ll := k.ToLLS(csize, conf)
					log("ToLLS")
					answers, err = ll.SolveRelative(f.Pivot, pivotTolerance, rhs...)
					log("Solve")
				} else {
					if !stiffnessChecked {
						if _, err := k.ToLLS(csize, conf).LDLTRelative(nil, pivotTolerance); err != nil {
							return f.CheckSingularNode(err)
						}
						stiffnessChecked = true
					}
					answers, err = solver.Solve(k, csize, conf, rhs...)
				}
				if err != nil {
					return f.CheckSingularNode(err)
				}
			}
			retryLap := false
			for i, answer := range answers {
				u := f.FillConf(answer)
				for j, v := range prescribed {
					u[j] += v
					if !finitePlastic(u[j]) {
						return fmt.Errorf("StaticAnalysis: non-finite displacement at DOF %d", j)
					}
				}
				load := linearLoads{}
				if i < len(loads) {
					load = loads[i]
				} else {
					load.nodal = f.FillConf(rhs[i])
				}
				trial, reaction := f.linearResult(elems, load, factors[i], conf, u)
				if i == 0 {
					changed, nextDelta, err := f.staticPost(c, trial, u, reaction, conf, accepted, initialFloor, delta)
					if err != nil {
						return err
					}
					if changed {
						stiffnessChecked = false
						retryLap = true
						if nextDelta != delta {
							delta = nextDelta
							target = linearTarget(from, delta, c.max)
						}
						break
					}
					f.recordStaticLoading(c, target, 0)
				}
				if final {
					if err := writeStaticResult(f, staticResultName(names[i], lap, c.nlap, true)); err != nil {
						return err
					}
				}
				f.staticResultCase = i
				if i == 0 {
					log(fmt.Sprintf("LAP %d: TOTAL = %.6g", lap, target))
				}
				if err := publish(lap); err != nil {
					return err
				}
				if i == 0 {
					accepted = f.SaveState()
				}
			}
			if retryLap {
				continue
			}
			if final {
				f.RestoreState(accepted)
				success = true
				log("End")
				return nil
			}
			break
		}
		from = target
	}
}

func linearTarget(from, delta, limit float64) float64 {
	target := from + delta
	tol := 8 * (math.Nextafter(1, 2) - 1) * math.Max(math.Abs(limit), math.Abs(target))
	if (delta > 0 && target >= limit-tol) || (delta < 0 && target <= limit+tol) {
		return limit
	}
	return target
}

// staticPost updates the active set using the equilibrated trial solution.
// Any support or member change requires another solve at the same load.
func (f *Frame) staticPost(c *AnalysisCondition, trial [][]float64, u, reaction []float64, conf []bool, accepted *FrameState, initialFloor []bool, delta float64) (bool, float64, error) {
	if !c.hasStaticPost() {
		return false, delta, nil
	}
	changed := false
	if c.floorContact != nil {
		z := *c.floorContact
		for i, n := range f.Nodes {
			if initialFloor[i] {
				continue
			}
			if n.Conf[2] {
				tol := 1e-10 * math.Max(1, math.Abs(reaction[6*i+2]))
				if reaction[6*i+2] < -tol {
					n.Conf[2] = false
					changed = true
				}
			} else {
				tol := 1e-10 * math.Max(1, math.Max(math.Abs(z), math.Abs(n.Coord[2])))
				if n.Coord[2]+u[6*i+2] < z-tol {
					n.Conf[2] = true
					changed = true
				}
			}
		}
	}
	for i, e := range f.Elems {
		if !c.compressionSections[e.Sect.Num] {
			continue
		}
		tol := 1e-10 * math.Max(1, math.Abs(c.compressionLimit))
		if (e.IsValid && trial[i][0] > c.compressionLimit+tol) ||
			(!e.IsValid && trial[i][0] < c.compressionLimit-tol) {
			e.IsValid = !e.IsValid
			changed = true
		}
	}
	if c.postprocess == nil {
		return changed, delta, nil
	}
	du, dr := slices.Clone(u), slices.Clone(reaction)
	df := make([][]float64, len(f.Elems))
	valid := make([]bool, len(f.Elems))
	for i, e := range f.Elems {
		valid[i] = e.IsValid
		df[i] = make([]float64, 12)
		for j, v := range e.Stress {
			df[i][j] = v - accepted.Stress[i][j]
		}
	}
	for i := range f.Nodes {
		for j := 0; j < 6; j++ {
			du[6*i+j] -= accepted.Disp[i][j]
			dr[6*i+j] -= accepted.Reaction[i][j]
		}
	}
	nextDelta, accept := c.postprocess(f, df, du, dr)
	for i, n := range f.Nodes {
		changed = changed || !slices.Equal(n.Conf, conf[6*i:6*i+6])
	}
	for i, e := range f.Elems {
		changed = changed || valid[i] != e.IsValid
	}
	if changed || accept {
		return changed, delta, nil
	}
	if !finitePlastic(nextDelta) || nextDelta == 0 || math.Signbit(nextDelta) != math.Signbit(delta) {
		return false, delta, fmt.Errorf("StaticAnalysis: invalid POST load increment")
	}
	if nextDelta == delta {
		return false, delta, fmt.Errorf("StaticAnalysis: POST rejected equilibrium without changing supports, elements or load increment")
	}
	return true, nextDelta, nil
}
