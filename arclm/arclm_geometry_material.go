package arclm

import "fmt"

// The history is plastic basic deformation in the element's co-rotating
// reference axes. It is unchanged by a superposed rigid motion. The yield
// surfaces, however, use actual current local end forces (including CMQ),
// not the generalized forces conjugate to chord extension/end rotations.
type geometryMaterialTrial struct {
	history   *plasticMaterialState
	fixed     [12]float64  // total, current load level, initial GLOBAL dead forces
	direction *[12]float64 // spatial correction, for one-sided yield tangents
}

func geometrySpatialB(b geometryJet, i, j int) float64 {
	v := b.h[i][j]
	for _, o := range []int{3, 9} {
		if i >= o && i < o+3 && j >= o && j < o+3 {
			g := geometryVector{b.d[o], b.d[o+1], b.d[o+2]}
			unit := geometryVector{}
			unit[j-o] = 1
			cross := geometryCross(g, unit)
			v -= .5 * cross[i-o]
		}
	}
	return v
}

func (e *geometryElement) materialResponse(b [12]geometryJet, axes geometryJMatrix, m *geometryMaterialTrial, r *geometryResponse) (q [12]float64, dq [12][12]float64, err error) {
	var c [12][12]float64
	var dc [12][12][12]float64 // dC[force][basic force][spatial increment]
	var fixed [12]geometryJet
	var elastic, trial [12]float64
	var kb, ck, kp [12][12]float64
	var history [12]float64
	if m.history != nil {
		history = m.history.plastic
	}
	// C=T B^T maps generalized basic forces to physical local end forces.
	// Its derivative includes both changing axes and the finite compatibility.
	for i := 0; i < 12; i++ {
		block, axis := i/3, i%3
		for j := 0; j < 3; j++ {
			g := 3*block + j
			fixed[i] = ga(fixed[i], gs(axes[axis][j], m.fixed[g]))
			for a := 0; a < 12; a++ {
				c[i][a] += axes[axis][j].v * b[a].d[g]
				for k := 0; k < 12; k++ {
					dc[i][a][k] += axes[axis][j].d[k]*b[a].d[g] + axes[axis][j].v*geometrySpatialB(b[a], g, k)
				}
			}
		}
	}
	for a := 0; a < 12; a++ {
		for j := 0; j < 12; j++ {
			elastic[a] += e.k[a][j] * (b[j].v - history[j])
			for k := 0; k < 12; k++ {
				kb[a][k] += e.k[a][j] * b[j].d[k]
			}
		}
	}
	for i := 0; i < 12; i++ {
		trial[i] = fixed[i].v
		for a := 0; a < 12; a++ {
			trial[i] += c[i][a] * elastic[a]
			for j := 0; j < 12; j++ {
				ck[i][j] += c[i][a] * e.k[a][j]
			}
		}
	}
	for i := 0; i < 12; i++ {
		for j := 0; j < 12; j++ {
			for a := 0; a < 12; a++ {
				kp[i][j] += ck[i][a] * c[j][a]
			}
		}
	}
	k := make([][]float64, 12)
	for i := range k {
		k[i] = append([]float64(nil), kp[i][:]...)
	}
	response, err := e.connection.returnPlasticForcesLinearized(trial[:], k, nil, true)
	if err != nil {
		return q, dq, err
	}
	// d(trial)-d(Kphysical)*plasticFlow is the RHS of the differentiated
	// implicit return equations. Omitting dK would give an inconsistent tangent
	// after finite rotation, even with a consistent material-only tangent.
	rhs := func(flow []float64) (a [12][12]float64) {
		var ctflow, kctflow [12]float64
		for j := 0; j < 12; j++ {
			for i := 0; i < 12; i++ {
				ctflow[j] += c[i][j] * flow[i]
			}
		}
		for i := 0; i < 12; i++ {
			for j := 0; j < 12; j++ {
				kctflow[i] += e.k[i][j] * ctflow[j]
			}
		}
		for i := 0; i < 12; i++ {
			for g := 0; g < 12; g++ {
				a[i][g] = fixed[i].d[g]
				for j := 0; j < 12; j++ {
					a[i][g] += dc[i][j][g]*(elastic[j]-kctflow[j]) + c[i][j]*kb[j][g]
					for h := 0; h < 12; h++ {
						a[i][g] -= ck[i][j] * dc[h][j][g] * flow[h]
					}
				}
			}
		}
		return
	}
	loadDerivative := rhs(response.PlasticIncrement)
	if m.direction != nil && response.directional {
		d := make([]float64, 12)
		for i := 0; i < 12; i++ {
			for j := 0; j < 12; j++ {
				d[i] += loadDerivative[i][j] * m.direction[j]
			}
		}
		response, err = e.connection.returnPlasticForcesLinearized(trial[:], k, d, true)
		if err != nil {
			return q, dq, err
		}
		loadDerivative = rhs(response.PlasticIncrement)
	}
	var dp [12][12]float64
	for i := 0; i < 12; i++ {
		for g := 0; g < 12; g++ {
			for j := 0; j < 12; j++ {
				dp[i][g] += response.plasticDerivative[i][j] * loadDerivative[j][g]
			}
		}
	}
	for a := 0; a < 12; a++ {
		for i := 0; i < 12; i++ {
			r.basicPlastic[a] += c[i][a] * response.PlasticIncrement[i]
		}
	}
	for a := 0; a < 12; a++ {
		q[a] = elastic[a]
		for j := 0; j < 12; j++ {
			q[a] -= e.k[a][j] * r.basicPlastic[j]
		}
		for g := 0; g < 12; g++ {
			dq[a][g] = kb[a][g]
			for j := 0; j < 12; j++ {
				for i := 0; i < 12; i++ {
					dq[a][g] -= e.k[a][j] * (dc[i][j][g]*response.PlasticIncrement[i] + c[i][j]*dp[i][g])
				}
			}
		}
		r.energy += .5 * (b[a].v - history[a] - r.basicPlastic[a]) * q[a]
	}
	r.material = response
	return
}

// At weakly active surfaces choose loading/unloading tangents using the
// correction produced by the global system itself. Trials do not commit
// plastic history, including during active-set selection and backtracking.
func (a *geometryAnalysis) correction(factor float64, trial *geometryTrial) ([]float64, error) {
	b := make([]float64, len(a.free))
	for i, j := range a.free {
		b[i] = trial.residual[j]
	}
	current := trial
	for pass := 0; pass < a.o.MaxIterations; pass++ {
		du, err := geometrySolve(a.ctx, current.k, b)
		if err != nil {
			return nil, err
		}
		weak := false
		for _, r := range current.responses {
			if r.material != nil && r.material.directional {
				weak = true
				break
			}
		}
		if !weak {
			return du, nil
		}
		next, err := a.evaluateDirectional(factor, du)
		if err != nil {
			return nil, err
		}
		changed := false
		for i, r := range current.responses {
			if r.material != nil && r.material.tangentActive != next.responses[i].material.tangentActive {
				changed = true
				break
			}
		}
		if !changed {
			return du, nil
		}
		current = next
	}
	return nil, fmt.Errorf("combined loading/unloading tangent selection did not stabilize")
}
