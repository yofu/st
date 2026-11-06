package arclm

import "math"

// geometryJet carries the value, gradient and Hessian of a scalar with respect
// to the twelve spatial increments of one beam. This differentiates finite
// compatibility, not a difference of stresses at a user-chosen step size.
type geometryJet struct {
	v float64
	d [12]float64
	h [12][12]float64
}

func gj(v float64) geometryJet           { return geometryJet{v: v} }
func gjVar(v float64, i int) geometryJet { r := gj(v); r.d[i] = 1; return r }
func ga(a, b geometryJet) geometryJet {
	a.v += b.v
	for i := 0; i < 12; i++ {
		a.d[i] += b.d[i]
		for j := 0; j < 12; j++ {
			a.h[i][j] += b.h[i][j]
		}
	}
	return a
}
func gs(a geometryJet, b float64) geometryJet {
	a.v *= b
	for i := 0; i < 12; i++ {
		a.d[i] *= b
		for j := 0; j < 12; j++ {
			a.h[i][j] *= b
		}
	}
	return a
}
func gm(a, b geometryJet) geometryJet {
	r := gj(a.v * b.v)
	for i := 0; i < 12; i++ {
		r.d[i] = a.d[i]*b.v + a.v*b.d[i]
		for j := 0; j < 12; j++ {
			r.h[i][j] = a.h[i][j]*b.v + a.d[i]*b.d[j] + a.d[j]*b.d[i] + a.v*b.h[i][j]
		}
	}
	return r
}
func gu(a geometryJet, v, d, h float64) geometryJet {
	r := gj(v)
	for i := 0; i < 12; i++ {
		r.d[i] = d * a.d[i]
		for j := 0; j < 12; j++ {
			r.h[i][j] = d*a.h[i][j] + h*a.d[i]*a.d[j]
		}
	}
	return r
}
func gi(a geometryJet) geometryJet { return gu(a, 1/a.v, -1/(a.v*a.v), 2/(a.v*a.v*a.v)) }
func gq(a geometryJet) geometryJet { r := math.Sqrt(a.v); return gu(a, r, .5/r, -.25/(a.v*r)) }

type geometryVector [3]float64

// Matrices store columns: the three material axes in global coordinates.
type geometryMatrix [3]geometryVector
type geometryJVector [3]geometryJet
type geometryJMatrix [3]geometryJVector

func gv(a geometryVector) geometryJVector { return geometryJVector{gj(a[0]), gj(a[1]), gj(a[2])} }
func gva(a, b geometryJVector) geometryJVector {
	return geometryJVector{ga(a[0], b[0]), ga(a[1], b[1]), ga(a[2], b[2])}
}
func gvs(a geometryJVector, b geometryJet) geometryJVector {
	return geometryJVector{gm(a[0], b), gm(a[1], b), gm(a[2], b)}
}
func gdot(a, b geometryJVector) geometryJet {
	return ga(ga(gm(a[0], b[0]), gm(a[1], b[1])), gm(a[2], b[2]))
}
func gcross(a, b geometryJVector) geometryJVector {
	return geometryJVector{ga(gm(a[1], b[2]), gs(gm(a[2], b[1]), -1)), ga(gm(a[2], b[0]), gs(gm(a[0], b[2]), -1)), ga(gm(a[0], b[1]), gs(gm(a[1], b[0]), -1))}
}
func gunit(a geometryJVector) geometryJVector { return gvs(a, gi(gq(gdot(a, a)))) }
func geometryIdentity() geometryMatrix        { return geometryMatrix{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}} }
func geometryCross(a, b geometryVector) geometryVector {
	return geometryVector{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
}
func geometryDot(a, b geometryVector) float64 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }
func geometryMV(a geometryMatrix, b geometryVector) (r geometryVector) {
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			r[i] += a[j][i] * b[j]
		}
	}
	return
}
func geometryMM(a, b geometryMatrix) (r geometryMatrix) {
	for j := 0; j < 3; j++ {
		r[j] = geometryMV(a, b[j])
	}
	return
}
func geometryExp(v geometryVector) geometryMatrix {
	t2 := geometryDot(v, v)
	a, b := 1-t2/6+t2*t2/120, .5-t2/24+t2*t2/720
	if t2 > 1e-10 {
		t := math.Sqrt(t2)
		a = math.Sin(t) / t
		b = 2 * math.Pow(math.Sin(t/2)/t, 2)
	}
	r := geometryIdentity()
	for j := 0; j < 3; j++ {
		c := geometryCross(v, r[j])
		cc := geometryCross(v, c)
		for i := 0; i < 3; i++ {
			r[j][i] += a*c[i] + b*cc[i]
		}
	}
	return r
}

// Quaternion extraction avoids the singular trace formula at a pi rotation.
func geometryLog(r geometryMatrix) geometryVector {
	q := [4]float64{}
	tr := r[0][0] + r[1][1] + r[2][2]
	if tr > 0 {
		s := 2 * math.Sqrt(1+tr)
		q = [4]float64{s / 4, (r[1][2] - r[2][1]) / s, (r[2][0] - r[0][2]) / s, (r[0][1] - r[1][0]) / s}
	} else {
		i := 0
		if r[1][1] > r[i][i] {
			i = 1
		}
		if r[2][2] > r[i][i] {
			i = 2
		}
		j, k := (i+1)%3, (i+2)%3
		s := 2 * math.Sqrt(1+r[i][i]-r[j][j]-r[k][k])
		q[i+1] = s / 4
		q[0] = (r[j][k] - r[k][j]) / s
		q[j+1] = (r[j][i] + r[i][j]) / s
		q[k+1] = (r[k][i] + r[i][k]) / s
	}
	if q[0] < 0 {
		for i := range q {
			q[i] = -q[i]
		}
	}
	n := math.Sqrt(q[1]*q[1] + q[2]*q[2] + q[3]*q[3])
	s := 2.0
	if n > 1e-12 {
		s = 2 * math.Atan2(n, q[0]) / n
	}
	return geometryVector{s * q[1], s * q[2], s * q[3]}
}

func geometryJLog(r geometryJMatrix) geometryJVector {
	w := gs(gq(ga(gj(1), ga(ga(r[0][0], r[1][1]), r[2][2]))), .5)
	v := geometryJVector{ga(r[1][2], gs(r[2][1], -1)), ga(r[2][0], gs(r[0][2], -1)), ga(r[0][1], gs(r[1][0], -1))}
	v = gvs(v, gs(gi(w), .25))
	n2 := gdot(v, v)
	var f geometryJet
	if n2.v < 1e-6 {
		f = ga(gj(2), ga(gs(n2, 1.0/3), ga(gs(gm(n2, n2), 3.0/20), gs(gm(gm(n2, n2), n2), 5.0/56))))
	} else {
		n := gq(n2)
		s := math.Sqrt(1 - n.v*n.v)
		asin := gu(n, math.Asin(n.v), 1/s, n.v/(s*s*s))
		f = gs(gm(asin, gi(n)), 2)
	}
	return gvs(v, f)
}

// exp(delta rotation)*R evaluated to second order at delta=0 is exact for
// the gradient/Hessian. Committed rotations use the full exponential above.
func geometryJRotate(r geometryMatrix, offset int) (a geometryJMatrix) {
	v := geometryJVector{gjVar(0, offset), gjVar(0, offset+1), gjVar(0, offset+2)}
	for j := 0; j < 3; j++ {
		x := gv(r[j])
		c := gcross(v, x)
		a[j] = gva(x, gva(c, gvs(gcross(v, c), gj(.5))))
	}
	return
}
