package arclm

import (
	"fmt"
	"math"
)

// admissiblePlasticDirection solves the directional return equations at the
// current trial state. On a weakly active surface, loading requires a
// nonnegative plastic multiplier rate; an inactive surface must move inward
// or tangentially. Already-positive multipliers have ordinary, two-sided
// derivatives. Enumerating the end sets also permits redundant axial surfaces
// to be represented by a single independent constraint.
func (elem *Elem) admissiblePlasticDirection(y, beta, centers, scale []float64, jac [][]float64, active []int, direction []float64) bool {
	rhs := make([][]float64, len(jac))
	norm := 0.0
	for i := range rhs {
		rhs[i] = make([]float64, 1)
		if i < 12 {
			rhs[i][0] = direction[i] / scale[i]
			if !finitePlastic(rhs[i][0]) {
				return false
			}
			norm = math.Max(norm, math.Abs(rhs[i][0]))
		}
	}
	if norm == 0 {
		return true
	}
	// Classification must not depend on the size of the load increment.
	for i := 0; i < 12; i++ {
		rhs[i][0] /= norm
	}
	rate, ok := solvePlasticDense(jac, rhs)
	if !ok {
		return false
	}
	var selected [2]bool
	for i, end := range active {
		selected[end] = true
		if !finitePlastic(rate[12+i][0]) || beta[i] <= 1e-11 && rate[12+i][0] < -1e-10 {
			return false
		}
	}
	for end := 0; end < 2; end++ {
		if selected[end] || elem.Sect.ElasticEnds[end] {
			continue
		}
		v, g, _ := elem.yieldPotential(y, end, centers)
		if math.Abs(v) > 1e-10 {
			continue
		}
		dot, magnitude := 0.0, 0.0
		for i, n := range g {
			term := n * rate[i][0]
			dot += term
			magnitude += math.Abs(term)
		}
		if !finitePlastic(dot) || dot > 1e-10*math.Max(1, magnitude) {
			return false
		}
	}
	return true
}

// solveDirectionalCorrection makes the selected boundary tangents consistent
// with the Newton direction they produce. Start with the ordinary (elastic
// at a weak boundary) tangent, so a single yielded member can unload without
// trying to factor its zero loading stiffness. No trial state is committed.
func (a *plasticNewtonAnalysis) solveDirectionalCorrection(base *FrameState, du []float64, from, target float64, trial *plasticNewtonTrial) ([]float64, error) {
	current := trial
	for pass := 0; pass < a.options.MaxIterations; pass++ {
		direction, err := a.solveLinear(current.tangent, trial.residual)
		if err != nil {
			return nil, err
		}
		weak := false
		for _, response := range current.responses {
			if response != nil && response.directional {
				weak = true
				break
			}
		}
		if !weak {
			return direction, nil
		}
		selected, err := a.evaluateDirectional(base, du, from, target, direction)
		if err != nil {
			return nil, err
		}
		changed := false
		for i, response := range current.responses {
			if response != nil && response.tangentActive != selected.responses[i].tangentActive {
				changed = true
				break
			}
		}
		if !changed {
			return direction, nil
		}
		current = selected
	}
	return nil, fmt.Errorf("loading/unloading tangent selection did not stabilize after %d passes", a.options.MaxIterations)
}
