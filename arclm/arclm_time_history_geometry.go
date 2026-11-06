package arclm

import (
	"context"
	"fmt"
	"math"

	"github.com/yofu/st/matrix"
)

// Newmark integrates translations and TOTAL exponential rotation coordinates.
// Spatial moments are pulled back by the exact left Jacobian of exp(theta),
// including its derivative in the tangent. Adding spatial rotation increments
// directly to theta would lose this term in non-planar motion.
// Mass is translational. C0 is a fixed positive Rayleigh matrix in these same
// coordinates; this is an initial-stiffness damping model, not follower damping.
type timeHistoryMechanicalTrial struct {
	*plasticNewtonTrial
	geometry *timeHistoryGeometryTrial
}
type timeHistoryGeometryTrial struct {
	responses []*geometryResponse
	rotations []geometryMatrix
	jacobians []geometryJMatrix
}

func newTimeHistoryGeometry(f *Frame, ctx context.Context, c *AnalysisCondition, o TimeHistoryOptions) (*plasticNewtonAnalysis, *geometryAnalysis, error) {
	for _, inertia := range o.RotationalInertia {
		if inertia != 0 {
			return nil, nil, fmt.Errorf("finite-rotation time history supports translational lumped mass only; nonzero rotational inertia requires rotational inertia/gyroscopic equations")
		}
	}
	c.SetNlgeometry(true)
	c.SetNlmaterial(true)
	c.SetInit(false)
	g, err := newGeometryAnalysis(f, ctx, c)
	if err != nil {
		return nil, nil, err
	}
	if f.geometry == nil && f.loading == nil {
		n := len(f.Nodes)
		g.state = &geometryState{material: true, definition: geometryDefinition(f), rotations: make([]geometryMatrix, n), axes: make([]geometryMatrix, len(g.elements)), baseNodal: make([]float64, 6*n), patternNodal: make([]float64, 6*n), baseFixed: make([][]float64, len(g.elements)), patternFixed: make([][]float64, len(g.elements))}
		for i, node := range f.Nodes {
			g.state.rotations[i] = geometryExp(geometryVector{node.Disp[3], node.Disp[4], node.Disp[5]})
		}
		for i, e := range g.elements {
			g.state.axes[i] = e.initial
			g.state.baseFixed[i] = append([]float64(nil), e.fixed[:]...)
			g.state.patternFixed[i] = make([]float64, 12)
		}
	} else if _, err = g.setup(); err != nil {
		return nil, nil, err
	}
	// Freeze the actual static preload, including its fixed-end dead forces.
	s := g.state
	for i := range s.baseNodal {
		s.baseNodal[i] += s.factor * s.patternNodal[i]
		s.patternNodal[i] = 0
	}
	for m := range s.baseFixed {
		for i := range s.baseFixed[m] {
			s.baseFixed[m][i] += s.factor * s.patternFixed[m][i]
			s.patternFixed[m][i] = 0
		}
	}
	s.factor, s.steps = 0, 0
	a := &plasticNewtonAnalysis{frame: f, ctx: ctx, cond: c, options: g.o, length: g.length, writer: g.w, conf: make([]bool, 6*len(f.Nodes))}
	a.baseNodal = append([]float64(nil), s.baseNodal...)
	for i, n := range f.Nodes {
		for j, b := range n.Conf {
			a.conf[6*i+j] = b
			if b {
				a.csize++
			}
		}
	}
	// Use the very same reference connection condensation as the finite kernel.
	for _, e := range g.elements {
		k, t := make([][]float64, 12), make([][]float64, 12)
		for i := 0; i < 12; i++ {
			k[i] = append([]float64(nil), e.k[i][:]...)
			t[i] = make([]float64, 12)
			for j := 0; j < 3; j++ {
				t[i][3*(i/3)+j] = e.initial[i%3][j]
			}
		}
		a.elements = append(a.elements, plasticAnalysisElement{index: e.index, connection: e.connection, elastic: k, transform: t})
	}
	return a, g, nil
}

// Columns are the spatial angular increment corresponding to each theta
// coordinate increment. The jets differentiate J itself, not element forces.
func timeHistoryRotationJacobian(theta geometryVector) (geometryJMatrix, error) {
	v := geometryJVector{gjVar(theta[0], 0), gjVar(theta[1], 1), gjVar(theta[2], 2)}
	x := gdot(v, v)
	if !finitePlastic(x.v) || x.v >= math.Pow(math.Pi-1e-4, 2) {
		return geometryJMatrix{}, fmt.Errorf("total nodal rotation reaches pi; finite-rotation time history chart exceeded")
	}
	var a, b geometryJet
	if x.v < 1e-4 {
		x2 := gm(x, x)
		x3 := gm(x2, x)
		a = ga(gj(.5), ga(gs(x, -1./24), ga(gs(x2, 1./720), gs(x3, -1./40320))))
		b = ga(gj(1./6), ga(gs(x, -1./120), ga(gs(x2, 1./5040), gs(x3, -1./362880))))
	} else {
		r := gq(x)
		cos := gu(r, math.Cos(r.v), -math.Sin(r.v), -math.Cos(r.v))
		sin := gu(r, math.Sin(r.v), math.Cos(r.v), -math.Sin(r.v))
		a = gm(ga(gj(1), gs(cos, -1)), gi(x))
		b = gm(ga(r, gs(sin, -1)), gi(gm(x, r)))
	}
	var j geometryJMatrix
	for k := 0; k < 3; k++ {
		unit := geometryVector{}
		unit[k] = 1
		c := gcross(v, gv(unit))
		j[k] = gva(gv(unit), gva(gvs(c, a), gvs(gcross(v, c), b)))
	}
	return j, nil
}

func (d *timeHistoryAnalysis) mechanical(base *FrameState, du, direction []float64) (*timeHistoryMechanicalTrial, error) {
	if d.geometry == nil {
		r, e := d.a.evaluate(base, du, 0, 0)
		return &timeHistoryMechanicalTrial{plasticNewtonTrial: r}, e
	}
	g, f := d.geometry, d.a.frame
	n := len(du)
	r := &timeHistoryMechanicalTrial{plasticNewtonTrial: &plasticNewtonTrial{internal: make([]float64, n), residual: make([]float64, n), tangent: matrix.NewCOOMatrix(n)}, geometry: &timeHistoryGeometryTrial{rotations: make([]geometryMatrix, len(f.Nodes)), jacobians: make([]geometryJMatrix, len(f.Nodes))}}
	nodes := make([]*Node, len(f.Nodes))
	for i, node := range f.Nodes {
		c := *node
		c.Disp = make([]float64, 6)
		for j := 0; j < 6; j++ {
			c.Disp[j] = base.Disp[i][j] + du[6*i+j]
		}
		nodes[i] = &c
		v := geometryVector{c.Disp[3], c.Disp[4], c.Disp[5]}
		var err error
		r.geometry.jacobians[i], err = timeHistoryRotationJacobian(v)
		if err != nil {
			return nil, fmt.Errorf("NODE %d: %w", node.Num, err)
		}
		r.geometry.rotations[i] = geometryExp(v)
	}
	for m, e := range g.elements {
		if err := d.a.ctx.Err(); err != nil {
			return nil, err
		}
		mat := &geometryMaterialTrial{history: f.Elems[e.index].material}
		copy(mat.fixed[:], g.state.baseFixed[m])
		var h [12][12]float64
		var gd [12]float64
		for i := 0; i < 12; i++ {
			if i%6 < 3 {
				h[i][i] = 1
			} else {
				j := r.geometry.jacobians[e.nodes[i/6]]
				for k := 0; k < 3; k++ {
					h[i][6*(i/6)+3+k] = j[k][i%6-3].v
				}
			}
			if direction != nil {
				for j := 0; j < 12; j++ {
					gd[i] += h[i][j] * direction[6*e.nodes[j/6]+j%6]
				}
			}
		}
		if direction != nil {
			mat.direction = &gd
		}
		response, err := e.responseMaterial(nodes, r.geometry.rotations, mat)
		if err != nil {
			return nil, fmt.Errorf("ELEM %d: %w", f.Elems[e.index].Num, err)
		}
		r.geometry.responses = append(r.geometry.responses, response)
		var force [12]float64
		var kh [12][12]float64
		for i := 0; i < 12; i++ {
			force[i] = response.force[i] + mat.fixed[i]
			for j := 0; j < 12; j++ {
				for k := 0; k < 12; k++ {
					kh[i][j] += response.tangent[i][k] * h[k][j]
				}
			}
		}
		for i := 0; i < 12; i++ {
			row := 6*e.nodes[i/6] + i%6
			for a := 0; a < 12; a++ {
				r.internal[row] += h[a][i] * force[a]
			}
			for j := 0; j < 12; j++ {
				v := 0.
				for a := 0; a < 12; a++ {
					v += h[a][i] * kh[a][j]
				}
				if i/6 == j/6 && i%6 >= 3 && j%6 >= 3 {
					jac := r.geometry.jacobians[e.nodes[i/6]]
					for a := 0; a < 3; a++ {
						v += jac[i%6-3][a].d[j%6-3] * force[6*(i/6)+3+a]
					}
				}
				if v != 0 {
					r.tangent.Add(row, 6*e.nodes[j/6]+j%6, v)
				}
			}
		}
	}
	return r, nil
}

// Return generalized applied forces and include the derivative of dead
// spatial moments. Translational ground inertia is unchanged by the pullback.
func (d *timeHistoryAnalysis) externalMechanical(t float64, r *timeHistoryMechanicalTrial) ([]float64, GroundMotionSample, error) {
	p, s, err := d.external(t)
	if err != nil || r.geometry == nil {
		return p, s, err
	}
	for i, jac := range r.geometry.jacobians {
		o := 6*i + 3
		m := append([]float64(nil), p[o:o+3]...)
		for j := 0; j < 3; j++ {
			p[o+j] = 0
			for a := 0; a < 3; a++ {
				p[o+j] += jac[j][a].v * m[a]
			}
			for k := 0; k < 3; k++ {
				for a := 0; a < 3; a++ {
					v := -jac[j][a].d[k] * m[a]
					if v != 0 {
						r.tangent.Add(o+j, o+k, v)
					}
				}
			}
		}
	}
	return p, s, nil
}

func (d *timeHistoryAnalysis) solve(a *plasticNewtonAnalysis, k *matrix.COOMatrix, b []float64) ([]float64, error) {
	if d.geometry == nil {
		return a.solveLinear(k, b)
	}
	free := []int{}
	for i, c := range a.conf {
		if !c {
			free = append(free, i)
		}
	}
	rows := make([]map[int]float64, len(free))
	rhs := make([]float64, len(free))
	for i, row := range free {
		rows[i] = make(map[int]float64)
		rhs[i] = b[row]
		for j, col := range free {
			if v := k.Query(row, col); v != 0 {
				rows[i][j] = v
			}
		}
	}
	// A sparse symmetric factor is only a preconditioner. Check and refine
	// against ALL entries of the nonsymmetric system; otherwise fall back to
	// pivoted LU. This avoids expensive LU fill in the common nearly-symmetric
	// elastic regime without deleting geometric/material skew terms.
	if factor, err := k.SymmetricLDLT(a.csize, a.conf, nil); err == nil {
		x := make([]float64, len(free))
		residual := append([]float64(nil), rhs...)
		for iteration := 0; iteration < 8; iteration++ {
			if err := a.ctx.Err(); err != nil {
				return nil, err
			}
			delta, err := factor.Solve(residual)
			if err != nil {
				break
			}
			for i := range x {
				x[i] += delta[i]
			}
			norm, reference := 0., 0.
			for i, row := range rows {
				v := 0.
				for j, c := range row {
					v += c * x[j]
				}
				residual[i] = rhs[i] - v
				norm = math.Hypot(norm, residual[i])
				reference = math.Hypot(reference, math.Max(math.Abs(rhs[i]), math.Abs(v)))
			}
			if finitePlastic(norm) && norm <= 1e-13+1e-11*reference {
				full := make([]float64, len(b))
				for i, j := range free {
					full[j] = x[i]
				}
				return full, nil
			}
		}
	}
	x, err := geometrySolve(a.ctx, rows, rhs)
	if err != nil {
		return nil, err
	}
	full := make([]float64, len(b))
	for i, j := range free {
		full[j] = x[i]
	}
	return full, nil
}

func (d *timeHistoryAnalysis) correction(base *FrameState, old *TimeHistoryState, du []float64, target float64, r *timeHistoryTrial, k *matrix.COOMatrix) ([]float64, error) {
	for pass := 0; pass < d.a.options.MaxIterations; pass++ {
		v, err := d.solve(d.a, k, r.material.residual)
		if err != nil {
			return nil, err
		}
		if r.material.geometry == nil {
			return v, nil
		}
		weak := false
		for _, e := range r.material.geometry.responses {
			weak = weak || e.material.directional
		}
		if !weak {
			return v, nil
		}
		next, err := d.evaluateDirectional(base, old, du, target, v)
		if err != nil {
			return nil, err
		}
		changed := false
		for i, e := range r.material.geometry.responses {
			changed = changed || e.material.tangentActive != next.material.geometry.responses[i].material.tangentActive
		}
		if !changed {
			return v, nil
		}
		dt := target - old.Time
		k = next.material.tangent.AddMat(d.c, d.opts.Gamma/(d.opts.Beta*dt))
		for i, m := range d.mass {
			if m != 0 {
				k.Add(i, i, m/(d.opts.Beta*dt*dt))
			}
		}
		r = next
	}
	return nil, fmt.Errorf("dynamic loading/unloading tangent selection did not stabilize")
}

func (d *timeHistoryAnalysis) commitGeometry(r *timeHistoryMechanicalTrial) {
	g := d.geometry
	g.state.rotations = append([]geometryMatrix(nil), r.geometry.rotations...)
	// Static commit expects spatial forces without fixed-end loads. Reactions
	// are replaced below by the complete dynamic equilibrium reaction.
	t := &geometryTrial{responses: r.geometry.responses, internal: make([]float64, len(d.mass)), external: make([]float64, len(d.mass))}
	g.commit(t, 0)
}

// Committed end forces already include the fixed-end load and are in the
// saved current axes. Reusing them for trapezoidal work avoids another full
// nonlinear tangent evaluation at the old time, without advancing history.
func (d *timeHistoryAnalysis) committedMechanical(base *FrameState) (*timeHistoryMechanicalTrial, error) {
	f := d.a.frame
	n := len(d.mass)
	r := &timeHistoryMechanicalTrial{plasticNewtonTrial: &plasticNewtonTrial{internal: make([]float64, n), tangent: matrix.NewCOOMatrix(n)}, geometry: &timeHistoryGeometryTrial{jacobians: make([]geometryJMatrix, len(f.Nodes))}}
	for _, e := range f.Elems {
		for block := 0; block < 4; block++ {
			v := geometryMV(*e.geometryAxes, geometryVector{e.Stress[3*block], e.Stress[3*block+1], e.Stress[3*block+2]})
			for j := 0; j < 3; j++ {
				r.internal[6*e.Enod[block/2].Index+3*(block%2)+j] += v[j]
			}
		}
	}
	for i, u := range base.Disp {
		j, err := timeHistoryRotationJacobian(geometryVector{u[3], u[4], u[5]})
		if err != nil {
			return nil, err
		}
		r.geometry.jacobians[i] = j
		o := 6*i + 3
		v := append([]float64(nil), r.internal[o:o+3]...)
		for k := 0; k < 3; k++ {
			r.internal[o+k] = 0
			for a := 0; a < 3; a++ {
				r.internal[o+k] += j[k][a].v * v[a]
			}
		}
	}
	return r, nil
}

// Equilibrate the saved dead preload in the same rotation coordinates as the
// transient run, before initializing velocity/acceleration. This also handles
// partially constrained rotation coordinates consistently.
func (d *timeHistoryAnalysis) equilibrateGeometry() error {
	f := d.a.frame
	base := f.SaveState()
	du := make([]float64, len(d.mass))
	for iteration := 0; iteration <= d.a.options.MaxIterations; iteration++ {
		r, err := d.mechanical(base, du, nil)
		if err != nil {
			return err
		}
		// Ground acceleration does not belong to the static preload.
		p := append([]float64(nil), d.a.baseNodal...)
		for i, jac := range r.geometry.jacobians {
			o := 6*i + 3
			m := append([]float64(nil), p[o:o+3]...)
			for j := 0; j < 3; j++ {
				p[o+j] = 0
				for a := 0; a < 3; a++ {
					p[o+j] += jac[j][a].v * m[a]
				}
				for k := 0; k < 3; k++ {
					for a := 0; a < 3; a++ {
						r.tangent.Add(o+j, o+k, -jac[j][a].d[k]*m[a])
					}
				}
			}
		}
		for i := range p {
			r.residual[i] = p[i] - r.internal[i]
		}
		tol := d.a.options.AbsoluteForceTolerance + d.a.cond.eps*math.Max(d.a.norm(p), d.a.norm(r.internal))
		if d.a.norm(r.residual) <= tol {
			for i, n := range f.Nodes {
				for j := 0; j < 6; j++ {
					n.Disp[j] = base.Disp[i][j] + du[6*i+j]
				}
			}
			d.commitGeometry(r)
			return nil
		}
		if iteration == d.a.options.MaxIterations {
			break
		}
		delta, err := d.solve(d.a, r.tangent, r.residual)
		if err != nil {
			return err
		}
		for i := range du {
			du[i] += delta[i]
		}
	}
	return fmt.Errorf("geometric time history preload did not converge")
}

// Reactions are reported as physical global forces/moments, not generalized
// theta forces. Recover the spatial moments from J^T*m=Q.
func (d *timeHistoryAnalysis) physicalReactions(r *timeHistoryMechanicalTrial) {
	if r.geometry == nil {
		return
	}
	for i, n := range d.a.frame.Nodes {
		j := r.geometry.jacobians[i]
		var a geometryMatrix
		for c := 0; c < 3; c++ {
			for row := 0; row < 3; row++ {
				a[c][row] = j[row][c].v
			}
		}
		v := geometryVector{n.Reaction[3], n.Reaction[4], n.Reaction[5]}
		det := geometryDot(a[0], geometryCross(a[1], a[2]))
		m := geometryVector{geometryDot(v, geometryCross(a[1], a[2])) / det, geometryDot(a[0], geometryCross(v, a[2])) / det, geometryDot(a[0], geometryCross(a[1], v)) / det}
		copy(n.Reaction[3:], m[:])
	}
}
