package arclm

import (
	"fmt"
	"math"

	"github.com/yofu/st/matrix"
)

// PlasticResponse is a trial, small-displacement, associated perfect-plastic
// response. Forces, plastic deformation increments and tangent use LOCAL axes.
// The tangent differentiates Stress with respect to the local displacement
// increment, with the previously committed element stress held fixed.
// At loading/unloading switches the map has one-sided derivatives; no unique
// two-sided tangent is implied at those points.
type PlasticResponse struct {
	Stress           []float64
	Increment        []float64
	PlasticIncrement []float64
	Tangent          [][]float64
	Multipliers      [2]float64
	Yield            [2]float64
}

type plasticMaterialState struct {
	plastic [12]float64
	tangent [][]float64
}

func copyPlasticMatrix(a [][]float64) [][]float64 {
	if a == nil {
		return nil
	}
	b := make([][]float64, len(a))
	for i := range a {
		b[i] = append([]float64(nil), a[i]...)
	}
	return b
}

func copyPlasticMaterial(s *plasticMaterialState) *plasticMaterialState {
	if s == nil {
		return nil
	}
	c := *s
	c.tangent = copyPlasticMatrix(s.tangent)
	return &c
}

// Material yielding is not a connection release. Use the input connection
// definitions when condensing elastic springs and hinges for the material law.
func (elem *Elem) plasticConnection() *Elem {
	c := *elem
	c.Bonds = append([]*Sect(nil), elem.Bonds...)
	for i, b := range c.Bonds {
		if i < len(elem.plasticBonds) && elem.plasticBonds[i] != nil {
			c.Bonds[i] = elem.plasticBonds[i]
		} else if b.Num < 0 {
			c.Bonds[i] = b.Snapshot()
			c.Bonds[i].Num = 0
		}
	}
	return &c
}

// yieldPotential evaluates F^q-RADIUS^q, its gradient and Hessian in
// capacity-normalised force coordinates. The two groups are (N,T,Mx,My)
// and (Qx,Qy), as in Coefficients. For p<2, smooth |x| near zero so that the
// local Newton system has a finite, consistent Hessian. The smoothing radius
// is 1e-8 of the corresponding capacity, not an artificial hardening modulus.
func (elem *Elem) yieldPotential(y []float64, end int, centers []float64) (float64, []float64, [][]float64) {
	p, q := elem.Sect.Exp, elem.Sect.Exq
	g := make([]float64, 12)
	h := make([][]float64, 12)
	for i := range h {
		h[i] = make([]float64, 12)
	}
	value := -math.Pow(RADIUS, q)
	smooth := 0.0
	if p < 2 {
		smooth = 1e-8
	}
	for _, group := range [][]int{{0, 3, 4, 5}, {1, 2}} {
		sum, count := 0.0, 0
		d, dd := [6]float64{}, [6]float64{}
		for _, j := range group {
			index := 6*end + j
			if elem.Bonds[index].Num == 1 {
				continue
			}
			count++
			x := y[index] - centers[index]
			z := x*x + smooth*smooth
			sum += math.Pow(z, p/2)
			if z != 0 {
				d[j] = p * x * math.Pow(z, p/2-1)
				dd[j] = p * math.Pow(z, p/2-2) * (smooth*smooth + (p-1)*x*x)
			} else if p == 2 {
				dd[j] = 2
			}
		}
		if count == 0 {
			continue
		}
		// Unequal nested exponents can have a non-smooth Hessian at the
		// group's origin as well. Smooth the group norm in that case.
		base := float64(count) * math.Pow(smooth, p)
		if q != p {
			const groupSmooth = 1e-8
			sum += math.Pow(groupSmooth, p)
			base += math.Pow(groupSmooth, p)
		}
		value += math.Pow(sum, q/p) - math.Pow(base, q/p)
		a, b := 1.0, 0.0
		if q != p {
			a = (q / p) * math.Pow(sum, q/p-1)
			b = (q / p) * (q/p - 1) * math.Pow(sum, q/p-2)
		}
		for _, j := range group {
			row := 6*end + j
			g[row] = a * d[j]
			for _, k := range group {
				h[row][6*end+k] = b * d[j] * d[k]
			}
			h[row][row] += a * dd[j]
		}
	}
	return value, g, h
}

// solvePlasticDense solves several right-hand sides with partial pivoting.
// Only the element's at-most 14 by 14 local return system is dense.
func solvePlasticDense(a, rhs [][]float64) ([][]float64, bool) {
	n := len(a)
	b, x := copyPlasticMatrix(a), copyPlasticMatrix(rhs)
	for k := 0; k < n; k++ {
		pivot := k
		for i := k + 1; i < n; i++ {
			if math.Abs(b[i][k]) > math.Abs(b[pivot][k]) {
				pivot = i
			}
		}
		if math.Abs(b[pivot][k]) < 1e-13 || math.IsNaN(b[pivot][k]) || math.IsInf(b[pivot][k], 0) {
			return nil, false
		}
		b[k], b[pivot] = b[pivot], b[k]
		x[k], x[pivot] = x[pivot], x[k]
		for i := k + 1; i < n; i++ {
			factor := b[i][k] / b[k][k]
			for j := k + 1; j < n; j++ {
				b[i][j] -= factor * b[k][j]
			}
			for j := range x[i] {
				x[i][j] -= factor * x[k][j]
			}
		}
	}
	for i := n - 1; i >= 0; i-- {
		for j := range x[i] {
			for k := i + 1; k < n; k++ {
				x[i][j] -= b[i][k] * x[k][j]
			}
			x[i][j] /= b[i][i]
		}
	}
	return x, true
}

func (elem *Elem) plasticSystem(y, trial, beta, centers []float64, stiffness [][]float64, active []int) ([]float64, [][]float64, float64) {
	n := 12 + len(active)
	r, jac := make([]float64, n), make([][]float64, n)
	for i := range jac {
		jac[i] = make([]float64, n)
		if i < 12 {
			jac[i][i] = 1
			r[i] = y[i] - trial[i]
		}
	}
	for a, end := range active {
		v, g, h := elem.yieldPotential(y, end, centers)
		r[12+a] = v
		for i := 0; i < 12; i++ {
			jac[12+a][i] = g[i]
			for j := 0; j < 12; j++ {
				jac[i][12+a] += stiffness[i][j] * g[j]
				for k := 0; k < 12; k++ {
					jac[i][k] += stiffness[i][j] * beta[a] * h[j][k]
				}
			}
			r[i] += beta[a] * jac[i][12+a]
		}
	}
	norm := 0.0
	for _, v := range r {
		norm = math.Hypot(norm, v)
	}
	return r, jac, norm
}

// Solve s = sTrial - K sum(gamma*dG/ds), G<=0, gamma>=0, gamma*G=0.
// No inverse of the singular 12-DOF elastic beam stiffness is needed.
func (elem *Elem) returnPlasticForces(trial []float64, k [][]float64) (*PlasticResponse, error) {
	if elem.Sect.Exp <= 1 || elem.Sect.Exq <= 1 || math.IsNaN(elem.Sect.Exp) || math.IsNaN(elem.Sect.Exq) || math.IsInf(elem.Sect.Exp, 0) || math.IsInf(elem.Sect.Exq, 0) {
		return nil, fmt.Errorf("ELEM %d: plastic exponents must be finite and greater than one", elem.Num)
	}
	scale, centers, t := make([]float64, 12), make([]float64, 12), make([]float64, 12)
	b := copyPlasticMatrix(k)
	maxB := 0.0
	for i := 0; i < 12; i++ {
		j := i % 6
		width := .5 * (elem.Sect.Yield[2*j] - elem.Sect.Yield[2*j+1])
		center := .5 * (elem.Sect.Yield[2*j] + elem.Sect.Yield[2*j+1])
		if elem.Bonds[i].Num == 1 {
			width, center = 1, 0
		}
		if width <= 0 || math.IsNaN(width) || math.IsInf(width, 0) || math.IsNaN(center) || math.IsInf(center, 0) || math.IsNaN(trial[i]) || math.IsInf(trial[i], 0) {
			return nil, fmt.Errorf("ELEM %d: invalid plastic capacity or force at component %d", elem.Num, i)
		}
		if i >= 6 && (j == 0 || j == 3) {
			center = -center
		}
		scale[i], centers[i], t[i] = width, center/width, trial[i]/width
	}
	for i := range b {
		for j := range b[i] {
			b[i][j] /= scale[i] * scale[j]
			maxB = math.Max(maxB, math.Abs(b[i][j]))
		}
	}
	if maxB == 0 || math.IsNaN(maxB) || math.IsInf(maxB, 0) {
		return nil, fmt.Errorf("ELEM %d: invalid elastic stiffness for plastic return", elem.Num)
	}
	for i := range b {
		for j := range b[i] {
			b[i][j] /= maxB
		}
	}
	// Enumerating two end surfaces also handles redundant constraints such as
	// pure axial yielding at both ends: a single independent surface suffices.
	for _, active := range [][]int{nil, {0}, {1}, {0, 1}} {
		y, beta := append([]float64(nil), t...), make([]float64, len(active))
		converged := false
		var jac [][]float64
		for iter := 0; iter < 80; iter++ {
			r, a, norm := elem.plasticSystem(y, t, beta, centers, b, active)
			jac = a
			if norm < 2e-12 {
				converged = true
				break
			}
			rhs := make([][]float64, len(r))
			for i := range r {
				rhs[i] = []float64{-r[i]}
			}
			delta, ok := solvePlasticDense(a, rhs)
			if !ok {
				break
			}
			accepted := false
			for step := 1.0; step >= 1.0/(1<<24); step *= .5 {
				ny, nb := append([]float64(nil), y...), append([]float64(nil), beta...)
				for i := range ny {
					ny[i] += step * delta[i][0]
				}
				for i := range nb {
					nb[i] += step * delta[12+i][0]
				}
				_, _, next := elem.plasticSystem(ny, t, nb, centers, b, active)
				if next < norm {
					y, beta, accepted = ny, nb, true
					break
				}
			}
			if !accepted {
				break
			}
		}
		if !converged {
			continue
		}
		valid := true
		for _, v := range beta {
			valid = valid && v >= -1e-11
		}
		response := &PlasticResponse{Stress: make([]float64, 12), Increment: make([]float64, 12), PlasticIncrement: make([]float64, 12)}
		for end := 0; end < 2; end++ {
			v, _, _ := elem.yieldPotential(y, end, centers)
			valid = valid && v <= 1e-10 && !math.IsNaN(v)
			response.Yield[end] = math.Pow(math.Max(0, v+math.Pow(RADIUS, elem.Sect.Exq)), 1/elem.Sect.Exq)
		}
		if !valid {
			continue
		}
		for a, end := range active {
			_, g, _ := elem.yieldPotential(y, end, centers)
			response.Multipliers[end] = math.Max(0, beta[a]) / maxB
			for i := range g {
				response.PlasticIncrement[i] += response.Multipliers[end] * g[i] / scale[i]
			}
		}
		// Differentiate the converged return equations, including the Hessian
		// of the curved yield surface, rather than using a continuum tangent.
		rhs := make([][]float64, len(jac))
		for i := range rhs {
			rhs[i] = make([]float64, 12)
			if i < 12 {
				for j := 0; j < 12; j++ {
					rhs[i][j] = k[i][j] / scale[i]
				}
			}
		}
		x, ok := solvePlasticDense(jac, rhs)
		if !ok {
			continue
		}
		response.Tangent = make([][]float64, 12)
		for i := 0; i < 12; i++ {
			response.Stress[i] = y[i] * scale[i]
			response.Tangent[i] = make([]float64, 12)
			for j := 0; j < 12; j++ {
				response.Tangent[i][j] = scale[i] * x[i][j]
			}
		}
		return response, nil
	}
	return nil, fmt.Errorf("ELEM %d: plastic return did not converge to an admissible state", elem.Num)
}

// TrialPlasticResponse does not commit forces or material history. Callers can
// evaluate multiple trial increments from a saved/committed state for Newton
// iteration or finite-difference verification.
func (elem *Elem) TrialPlasticResponse(globalIncrement []float64) (*PlasticResponse, error) {
	if len(globalIncrement) != 12 {
		return nil, fmt.Errorf("ELEM %d: plastic displacement increment must have 12 components", elem.Num)
	}
	c := elem.plasticConnection()
	t, err := c.TransMatrix()
	if err != nil {
		return nil, err
	}
	k, err := c.StiffMatrix()
	if err != nil {
		return nil, err
	}
	k, err = c.ModifyHinge(k)
	if err != nil {
		return nil, err
	}
	u := matrix.MatrixVector(t, globalIncrement)
	trial := matrix.MatrixVector(k, u)
	for i := range trial {
		trial[i] += elem.Stress[i]
	}
	r, err := c.returnPlasticForces(trial, k)
	if err != nil {
		return nil, err
	}
	for i := range r.Increment {
		r.Increment[i] = r.Stress[i] - elem.Stress[i]
	}
	return r, nil
}

func (elem *Elem) commitPlasticResponse(r *PlasticResponse) {
	if elem.material == nil {
		elem.material = &plasticMaterialState{}
	}
	for i := range elem.Stress {
		elem.Stress[i] = r.Stress[i]
		elem.material.plastic[i] += r.PlasticIncrement[i]
	}
	elem.material.tangent = copyPlasticMatrix(r.Tangent)
	for end := 0; end < 2; end++ {
		elem.Phinge[end] = r.Yield[end] >= RADIUS-1e-9
		for j := 0; j < 6; j++ {
			i := 6*end + j
			if elem.Bonds[i].Num < 0 {
				elem.setPlasticBond(i, 0)
			}
			if elem.Phinge[end] && elem.Bonds[i].Num == 0 && (!elem.IsBrace() || j == 0) {
				num := -1
				if elem.IsBrace() {
					force := r.Stress[6]
					num = -2
					if force < 0 {
						num = -3
					}
				}
				elem.setPlasticBond(i, num)
			}
		}
	}
}
