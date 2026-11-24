package st

import (
	"fmt"
	"math"
	"slices"
)

// N and moment are in tf and tf m, respectively.
type rcNMPoint struct{ n, m float64 }
type rcNMCurve struct {
	points []rcNMPoint
	peak   rcNMPoint
}

// Sample exactly the axial-force grid used in :nminteraction, endpoints included.
func rcNMReference(rc *RCColumn, cond *Condition, divisions int) rcNMCurve {
	c := *cond
	high, low := rc.Nmax(&c), rc.Nmin(&c)
	curve := rcNMCurve{points: make([]rcNMPoint, divisions+1)}
	for i := range curve.points {
		c.N = high - float64(i)*(high-low)/float64(divisions)
		var moment float64
		if c.Period == "U" {
			moment = rc.Mu(&c)
		} else {
			moment = rc.Ma(&c)
		}
		pt := rcNMPoint{c.N, moment}
		curve.points[i] = pt
		if pt.m > curve.peak.m {
			curve.peak = pt
		}
	}
	return curve
}

// Check each reference ordinate at its own N. The displayed error refers to
// the reference samples, rather than comparing two curves at different N.
func (c rcNMCurve) excess(nmax, nmin, moment, exp, radius float64) rcNMPoint {
	mid, half := (nmax+nmin)/2, (nmax-nmin)/2
	var worst rcNMPoint
	for _, v := range c.points {
		r := math.Pow(radius, exp) - math.Pow(math.Abs((v.n-mid)/half), exp)
		if r <= 0 {
			continue
		}
		diff := moment*math.Pow(r, 1/exp) - v.m
		if diff > worst.m {
			worst = rcNMPoint{v.n, diff}
		}
	}
	return worst
}

const rcGravity = 9.80665

// RCShearInput is the beam Qsu worksheet's input in mm, mm² and N/mm².
// Moment and shear must be a corresponding pair in kN m and kN.
type RCShearInput struct {
	Width, Depth, TensionArea, HoopRatio, HoopStrength, ConcreteStrength float64
	Moment, Shear                                                        float64
}
type RCShearCapacity struct{ KN, SpanRatio, Pt, Pw float64 }

// UltimateShear reproduces RC終局「梁」, including the cap on pw and the
// worksheet's unclamped shear span. Pt is a percentage; Pw is a fraction.
func (v RCShearInput) UltimateShear() (RCShearCapacity, error) {
	var q RCShearCapacity
	if !positiveYieldInputs(v.Width, v.Depth, v.TensionArea, v.HoopStrength, v.ConcreteStrength, math.Abs(v.Shear)) ||
		!finiteRC(v.Moment) || !finiteRC(v.HoopRatio) || v.HoopRatio < 0 {
		return q, fmt.Errorf("Qsu: invalid geometry, reinforcement or M/Q")
	}
	q.SpanRatio = math.Abs(v.Moment/v.Shear) * 1000 / v.Depth
	q.Pt = 100 * v.TensionArea / (v.Width * v.Depth)
	q.Pw = math.Min(.012, v.HoopRatio)
	tau := .068*math.Pow(q.Pt, .23)*(v.ConcreteStrength+18)/(q.SpanRatio+.12) + .85*math.Sqrt(q.Pw*v.HoopStrength)
	q.KN = tau * v.Width * (7 * v.Depth / 8) / 1000
	if !finiteRC(q.SpanRatio) || !positiveYieldInputs(q.KN) {
		return q, fmt.Errorf("Qsu: invalid result")
	}
	return q, nil
}
func finiteRC(x float64) bool { return !math.IsInf(x, 0) && !math.IsNaN(x) }

func checkRCSection(rc *RCColumn) error {
	rect, ok := rc.CShape.(CRect)
	if !ok || !positiveYieldInputs(rc.fc, rect.Height(true)-11.1, rect.Breadth(true)-11.1) {
		return fmt.Errorf("rectangular RC section with a positive torsion core required")
	}
	if len(rc.Reins) == 0 || rc.Nreins != len(rc.Reins) || len(rc.Hoops.Ps) != 2 || !positiveYieldInputs(rc.Hoops.Material.Fs) {
		return fmt.Errorf("REINS / HOOPS data incomplete")
	}
	for _, pw := range rc.Hoops.Ps {
		if !positiveYieldInputs(pw) || pw >= 1 {
			return fmt.Errorf("invalid hoop reinforcement ratio %g", pw)
		}
	}
	for _, bar := range rc.Reins {
		if len(bar.Position) != 2 || !positiveYieldInputs(bar.Area, bar.Material.Fs, bar.Material.E) {
			return fmt.Errorf("invalid longitudinal bar")
		}
		for axis, pos := range bar.Position {
			if !finiteRC(pos) || pos <= rect.Bound(axis) || pos >= rect.Bound(axis+2) {
				return fmt.Errorf("bar outside concrete section")
			}
		}
	}
	return nil
}

// Curves are uniaxial. Mirror symmetry is needed in each axis' projected
// reinforcement distribution for the symmetric Mu formula to apply.
func rcSymmetric(rc *RCColumn) bool {
	for axis := 0; axis < 2; axis++ {
		mid := (rc.Bound(axis) + rc.Bound(axis+2)) / 2
		for _, b := range rc.Reins {
			pos := b.Position[axis] - mid
			var a, mirrored float64
			for _, r := range rc.Reins {
				if r.Material != b.Material {
					continue
				}
				x := r.Position[axis] - mid
				if math.Abs(x-pos) < 1e-7 {
					a += r.Area
				}
				if math.Abs(x+pos) < 1e-7 {
					mirrored += r.Area
				}
			}
			if math.Abs(a-mirrored) > 1e-7 {
				return false
			}
		}
	}
	return true
}

func rcCentered(src *RCColumn) *RCColumn {
	r := *src
	rect := r.CShape.(CRect)
	cx, cy := (rect.Left+rect.Right)/2, (rect.Upper+rect.Lower)/2
	if cx == 0 && cy == 0 {
		return &r
	}
	r.CShape = NewCRect([]float64{rect.Left - cx, rect.Lower - cy, rect.Right - cx, rect.Upper - cy})
	r.Reins = slices.Clone(r.Reins)
	for i, bar := range r.Reins {
		r.Reins[i].Position = []float64{bar.Position[0] - cx, bar.Position[1] - cy}
	}
	return &r
}
