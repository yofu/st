package arclm

import (
	"fmt"
	"math"
)

// geometryElement separates the reference elastic stiffness from finite
// compatibility; the optional plastic return uses current physical end forces. The frame follows the chord and the mean of the two end
// triads after their first axes have been aligned to the chord (Goto, 1983,
// section 5). Local strains must be small; refine curved/twisted members.
type geometryElement struct {
	connection *Elem
	index      int
	nodes      [2]int
	length     float64
	initial    geometryMatrix
	k          [12][12]float64
	fixed      [12]float64 // equivalent fixed-end forces in the initial global frame
}
type geometryResponse struct {
	force        [12]float64
	tangent      [12][12]float64
	axes         geometryMatrix
	energy       float64
	material     *PlasticResponse
	basicPlastic [12]float64
}

func newGeometryElement(e *Elem, index int) (geometryElement, error) {
	g := geometryElement{index: index, length: e.Length0()}
	if !finitePlastic(g.length) || g.length <= 0 || e.Sect == nil || len(e.Sect.Value) != 4 || !finitePlastic(e.Sect.E) || e.Sect.E <= 0 || !finitePlastic(e.Sect.Poi) || e.Sect.Poi <= -1 || !finitePlastic(e.Cang) {
		return g, fmt.Errorf("geometric analysis: invalid ELEM %d geometry/material", e.Num)
	}
	for _, v := range e.Sect.Value {
		if !finitePlastic(v) || v < 0 {
			return g, fmt.Errorf("ELEM %d: invalid section property", e.Num)
		}
	}
	// Always construct the reference stiffness and axes at input coordinates.
	c := *e.plasticConnection()
	c.geometryAxes = nil
	for i := 0; i < 2; i++ {
		g.nodes[i] = e.Enod[i].Index
	}
	n0, n1 := *e.Enod[0], *e.Enod[1]
	n0.Disp = make([]float64, 6)
	n1.Disp = make([]float64, 6)
	c.Enod = []*Node{&n0, &n1}
	s, w, err := c.PrincipalAxis(c.Cang)
	if err != nil {
		return g, err
	}
	d := c.Direction(true)
	for i := 0; i < 3; i++ {
		g.initial[0][i] = d[i]
		g.initial[1][i] = s[i]
		g.initial[2][i] = w[i]
	}
	k, err := c.StiffMatrix()
	if err != nil {
		return g, err
	}
	q := append([]float64(nil), e.Cmq...)
	if err := c.condenseConnections(k, q); err != nil {
		return g, err
	}
	for i := 0; i < 12; i++ {
		copy(g.k[i][:], k[i])
	}
	for n := 0; n < 4; n++ {
		x := geometryMV(g.initial, geometryVector{q[3*n], q[3*n+1], q[3*n+2]})
		copy(g.fixed[3*n:3*n+3], x[:])
	}
	g.connection = &c
	return g, nil
}

func (e *geometryElement) response(nodes []*Node, rotations []geometryMatrix) (*geometryResponse, error) {
	return e.responseMaterial(nodes, rotations, nil)
}

func (e *geometryElement) responseMaterial(nodes []*Node, rotations []geometryMatrix, material *geometryMaterialTrial) (*geometryResponse, error) {
	var end [2]geometryJMatrix
	var chord geometryJVector
	for i := 0; i < 3; i++ {
		a, b := nodes[e.nodes[0]], nodes[e.nodes[1]]
		chord[i] = ga(gjVar(b.Disp[i], 6+i), gs(gjVar(a.Disp[i], i), -1))
		chord[i].v += b.Coord[i] - a.Coord[i]
	}
	length := gq(gdot(chord, chord))
	if !finitePlastic(length.v) || length.v < 1e-10*e.length {
		return nil, fmt.Errorf("collapsed element chord")
	}
	x := gvs(chord, gi(length))
	var y geometryJVector
	for i := 0; i < 2; i++ {
		end[i] = geometryJRotate(geometryMM(rotations[e.nodes[i]], e.initial), 6*i+3)
		v := gcross(end[i][0], x)
		c := ga(gj(1), gdot(end[i][0], x))
		if c.v < 1e-6 {
			return nil, fmt.Errorf("element end rotated through pi relative to its chord; refine the member")
		}
		aligned := gva(end[i][1], gva(gcross(v, end[i][1]), gvs(gcross(v, gcross(v, end[i][1])), gi(c))))
		y = gva(y, aligned)
	}
	if gdot(y, y).v < 1e-10 {
		return nil, fmt.Errorf("element relative twist reaches pi; refine the member")
	}
	y = gunit(y)
	z := gunit(gcross(x, y))
	y = gcross(z, x)
	frame := geometryJMatrix{x, y, z}
	var b [12]geometryJet
	b[6] = ga(length, gj(-e.length))
	for n := 0; n < 2; n++ {
		var relative geometryJMatrix
		for j := 0; j < 3; j++ {
			for i := 0; i < 3; i++ {
				relative[j][i] = gdot(frame[i], end[n][j])
			}
		}
		if relative[0][0].v+relative[1][1].v+relative[2][2].v < -.99 {
			return nil, fmt.Errorf("large local beam rotation; refine the member")
		}
		r := geometryJLog(relative)
		for i := 0; i < 3; i++ {
			b[6*n+3+i] = r[i]
		}
	}

	r := &geometryResponse{}
	var q [12]float64
	var kb [12][12]float64
	if material == nil {
		for a := 0; a < 12; a++ {
			for c := 0; c < 12; c++ {
				q[a] += e.k[a][c] * b[c].v
				for i := 0; i < 12; i++ {
					kb[a][i] += e.k[a][c] * b[c].d[i]
				}
			}
			r.energy += .5 * b[a].v * q[a]
		}
	} else {
		var err error
		q, kb, err = e.materialResponse(b, frame, material, r)
		if err != nil {
			return nil, err
		}
	}
	// The same finite compatibility is used for both the plastic return and
	// the separated tangent: p=B^T q, dp=B^T dq+sum(q_a Hessian(b_a)).
	for i := 0; i < 12; i++ {
		for a := 0; a < 12; a++ {
			r.force[i] += b[a].d[i] * q[a]
		}
		for j := 0; j < 12; j++ {
			for a := 0; a < 12; a++ {
				r.tangent[i][j] += b[a].d[i]*kb[a][j] + q[a]*b[a].h[i][j]
			}
		}
	}
	// The Hessian is conjugate to coordinates in exp(delta)*R. Convert it to
	// the derivative of spatial moments used in the global residual. This
	// rotational connection term is skew and must not be silently symmetrized.
	for _, o := range []int{3, 9} {
		m := geometryVector{r.force[o], r.force[o+1], r.force[o+2]}
		for j := 0; j < 3; j++ {
			v := geometryVector{}
			v[j] = 1
			c := geometryCross(m, v)
			for i := 0; i < 3; i++ {
				r.tangent[o+i][o+j] -= .5 * c[i]
			}
		}
	}
	for j := 0; j < 3; j++ {
		for i := 0; i < 3; i++ {
			r.axes[j][i] = frame[j][i].v
		}
	}
	for i := range r.force {
		if !finitePlastic(r.force[i]) {
			return nil, fmt.Errorf("non-finite element response")
		}
		for _, v := range r.tangent[i] {
			if !finitePlastic(v) {
				return nil, fmt.Errorf("non-finite element tangent")
			}
		}
	}
	return r, nil
}

func geometryNorm(v []float64, free []int, length float64) float64 {
	n := 0.0
	for _, i := range free {
		x := v[i]
		if i%6 >= 3 {
			x /= length
		}
		n = math.Hypot(n, x)
	}
	return n
}
