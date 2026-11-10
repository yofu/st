package arclm

// plasticLoadState records actual applied nodal loads, separately from member
// stresses and reactions. On changing load pattern the previous applied loads
// remain constant; the new pattern starts at factor zero. Fixed-end patterns
// identify stages, while their accumulated effect remains in the member stress.
type plasticLoadState struct {
	steps        int // Completed requested steps in this load stage.
	factor       float64
	nodalPattern []float64
	fixedPattern [][]float64
	baseNodal    []float64
}

func copyPlasticLoading(state *plasticLoadState) *plasticLoadState {
	if state == nil {
		return nil
	}
	c := *state
	c.nodalPattern = append([]float64(nil), state.nodalPattern...)
	c.fixedPattern = copyPlasticMatrix(state.fixedPattern)
	c.baseNodal = append([]float64(nil), state.baseNodal...)
	return &c
}

func (state *plasticLoadState) appliedNodal() []float64 {
	r := append([]float64(nil), state.baseNodal...)
	for i := range r {
		r[i] += state.factor * state.nodalPattern[i]
	}
	return r
}

func (frame *Frame) loadPatterns() ([]float64, [][]float64) {
	nodal := make([]float64, 6*len(frame.Nodes))
	fixed := make([][]float64, len(frame.Elems))
	for i, node := range frame.Nodes {
		copy(nodal[6*i:6*i+6], node.Force)
	}
	for i, elem := range frame.Elems {
		fixed[i] = append([]float64(nil), elem.Cmq...)
	}
	return nodal, fixed
}

func samePlasticLoadValues(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (state *plasticLoadState) samePattern(nodal []float64, fixed [][]float64) bool {
	if !samePlasticLoadValues(state.nodalPattern, nodal) || len(state.fixedPattern) != len(fixed) {
		return false
	}
	for i := range fixed {
		if !samePlasticLoadValues(state.fixedPattern[i], fixed[i]) {
			return false
		}
	}
	return true
}

// Keep the actual load pattern and factor for ordinary linear continuation.
// POST may change the active constraints or elements, so do not use that
// state as a preload for a different formulation.
func (frame *Frame) recordStaticLoading(cond *AnalysisCondition, factor float64, rhs int) {
	frame.loading = nil
	frame.plasticLoadValid = false
	if cond.NonLinear() || cond.hasStaticPost() || rhs != 0 {
		return
	}
	nodal, fixed := frame.loadPatterns()
	frame.loading = &plasticLoadState{factor: factor, nodalPattern: nodal, fixedPattern: fixed, baseNodal: make([]float64, len(nodal))}
	frame.plasticLoad, frame.plasticLoadValid = factor, true
}

// ContinuationStep returns the last completed step for the current load pattern.
// A different pattern starts a new stage at step zero.
func (frame *Frame) ContinuationStep() int {
	if s := frame.geometry; s != nil {
		nodal, fixed := frame.loadPatterns()
		match := samePlasticLoadValues(nodal, s.patternNodal) && len(fixed) == len(s.rawFixed)
		for i := range fixed {
			if !match || !samePlasticLoadValues(fixed[i], s.rawFixed[i]) {
				return 0
			}
		}
		if match {
			return s.steps
		}
		return 0
	}
	if frame.loading == nil {
		return 0
	}
	nodal, fixed := frame.loadPatterns()
	if !frame.loading.samePattern(nodal, fixed) {
		return 0
	}
	return frame.loading.steps
}
