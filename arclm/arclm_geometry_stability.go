package arclm

import (
	"fmt"
	"math"

	"github.com/yofu/st/matrix"
)

type geometryInertia struct {
	positive, zero, negative int
}

// The symmetric part governs the incremental quadratic work. The Newton
// correction still uses the full spatial Jacobian, which can be nonsymmetric
// at an unbalanced configuration.
func geometryTangentInertia(rows []map[int]float64) (geometryInertia, error) {
	n := len(rows)
	if n == 0 {
		return geometryInertia{}, nil
	}
	type entry struct {
		row, col int
		value    float64
	}
	var upper []entry
	scale := make([]float64, n)
	for i, row := range rows {
		for j, v := range row {
			r, c := i, j
			if i < j {
				v = .5*v + .5*rows[j][i]
			} else if i > j {
				if _, exists := rows[j][i]; exists {
					continue
				}
				r, c, v = j, i, .5*v
			}
			if !finitePlastic(v) {
				return geometryInertia{}, fmt.Errorf("non-finite symmetric tangent entry at %d,%d", i+1, j+1)
			}
			if v != 0 {
				upper = append(upper, entry{r, c, v})
				scale[r] = math.Max(scale[r], math.Abs(v))
				scale[c] = math.Max(scale[c], math.Abs(v))
			}
		}
	}
	for i, v := range scale {
		scale[i] = math.Sqrt(v)
	}
	k := matrix.NewCOOMatrix(n)
	for _, e := range upper {
		// Positive diagonal congruence preserves inertia and equilibrates
		// stiffnesses of translational and rotational degrees of freedom.
		k.Set(e.row, e.col, (e.value/scale[e.row])/scale[e.col])
	}
	p, z, m, err := k.SymmetricInertia(nil, 0, 0, make([]bool, n), nil)
	return geometryInertia{p, z, m}, err
}

// Large updates warrant interior samples even when their residual decreases.
// The threshold is relative to the model length (or to one radian).
func (a *geometryAnalysis) largeStabilityCorrection(du []float64, scale float64) bool {
	const fraction = .01
	for i, v := range du {
		limit := fraction
		if a.free[i]%6 < 3 {
			limit *= a.length
		}
		if math.Abs(scale*v) > limit {
			return true
		}
	}
	return false
}

func (a *geometryAnalysis) checkStability(t *geometryTrial, from, target float64, iter int, fraction float64, converged, trialWarning bool) bool {
	inertia, err := geometryTangentInertia(t.k)
	if err != nil {
		fmt.Fprintf(a.w, "WARNING: GEOMETRY STABILITY CHECK FAILED FROM=%.12g LOAD=%.12g ITER=%d: %v\n", from, target, iter, err)
		return trialWarning
	}
	if converged {
		label := "GEOMETRY STABILITY: equilibrium"
		switch {
		case inertia.negative != 0:
			label = "WARNING: GEOMETRY STABILITY: unstable equilibrium"
		case inertia.zero != 0:
			label = "WARNING: GEOMETRY STABILITY: critical equilibrium"
		case trialWarning:
			label = "WARNING: GEOMETRY STABILITY: possible branch change, converged tangent is positive definite"
		}
		fmt.Fprintf(a.w, "%s FROM=%.12g LOAD=%.12g POSITIVE=%d ZERO=%d NEGATIVE=%d\n", label, from, target, inertia.positive, inertia.zero, inertia.negative)
	} else if inertia.negative != 0 || inertia.zero != 0 {
		fmt.Fprintf(a.w, "WARNING: GEOMETRY STABILITY: trial tangent is indefinite or singular FROM=%.12g LOAD=%.12g ITER=%d UPDATE_FRACTION=%g POSITIVE=%d ZERO=%d NEGATIVE=%d\n", from, target, iter, fraction, inertia.positive, inertia.zero, inertia.negative)
		return true
	}
	return trialWarning
}
