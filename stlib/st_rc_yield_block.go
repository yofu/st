package st

import (
	"fmt"
	"math"
)

// For asymmetric reinforcement the symmetric Mu expression cannot distinguish
// sagging and hogging. Use plane sections, elastic-perfectly-plastic bars,
// zero concrete tension, and a Whitney compression block (epsilon_cu=0.003,
// 0.85 Fc; beta1 0.85 through 28 MPa, reducing 0.05/7 MPa to 0.65).
// Work in the same tf/cm material units as the RC design section.
type rcStressBlock struct {
	rc                  *RCColumn
	axis                int
	width, height, beta float64
}

func newRCBlock(rc *RCColumn, axis int) rcStressBlock {
	strong := axis == 1
	return rcStressBlock{rc, axis, rc.Breadth(strong), rc.Height(strong), math.Max(.65, .85-.05*math.Max(0, rc.fc*100-28)/7)}
}

// Integral of the part of a circular bar above the compression-block edge.
// Return displaced concrete area and its first moment about the bar center.
func rcBarSegment(area, distance float64) (float64, float64) {
	radius := math.Sqrt(area / math.Pi)
	if distance >= radius {
		return 0, 0
	}
	if distance <= -radius {
		return area, 0
	}
	h := math.Sqrt(radius*radius - distance*distance)
	return radius*radius*math.Acos(distance/radius) - distance*h, 2 * h * h * h / 3
}

// t=c/(h+c) parametrizes neutral-axis depth from all tension to all compression.
type rcBlockState struct {
	rcNMPoint
	transverse float64 // orthogonal-axis moment in tf m about the rectangle center
}

func (b rcStressBlock) forces(t, side float64) rcBlockState {
	fc := .85 * b.rc.fc
	depth := 0.
	c := 0.
	if t >= 1 {
		depth = b.height
		c = math.Inf(1)
	} else if t > 0 {
		c = b.height * t / (1 - t)
		depth = math.Min(b.height, b.beta*c)
	}
	n := fc * b.width * depth
	m := n * (b.height - depth) / 2
	cross := 0.0
	for _, bar := range b.rc.Reins {
		y := side * bar.Position[b.axis]
		stress := -bar.Material.Fs
		if c > 0 {
			stress = math.Max(-bar.Material.Fs, math.Min(bar.Material.Fs, bar.Material.E*.003*(1-(b.height/2-y)/c)))
		}
		area, first := rcBarSegment(bar.Area, b.height/2-depth-y)
		f := bar.Area*stress - fc*area
		n += f
		m += f*y - fc*first
		cross += f * bar.Position[1-b.axis]
	}
	return rcBlockState{rcNMPoint{n, side * m / 100}, cross / 100}
}
func (b rcStressBlock) atN(n, side float64) rcBlockState {
	lo, hi := 0., 1.
	for range 52 {
		mid := (lo + hi) / 2
		if b.forces(mid, side).n < n {
			lo = mid
		} else {
			hi = mid
		}
	}
	return b.forces((lo+hi)/2, side)
}

// Intersect the convex polygon spanned by four directional section states
// at a common N with each moment axis. A projected moment alone can carry an
// unwanted transverse moment in an asymmetric layout. Each intercept uses
// either one force-balanced state or a linear interpolation of two states.
func rcAxisIntercepts(states [4][2]float64, axis int) (float64, float64) {
	lo, hi := math.Inf(1), math.Inf(-1)
	add := func(x float64) { lo = math.Min(lo, x); hi = math.Max(hi, x) }
	for i, a := range states {
		u, v := a[axis], a[1-axis]
		if math.Abs(v) < 1e-10 {
			add(u)
		}
		for _, b := range states[:i] {
			w := b[1-axis]
			if (v < 0 && w > 0) || (v > 0 && w < 0) {
				add(u + (b[axis]-u)*v/(v-w))
			}
		}
	}
	return lo, hi
}

type rcBlockAxes [2]rcStressBlock

// arclm uses one M center at both member ends while the end-force sign
// reverses. Use the smaller magnitude of the two signed intercepts.
func (axes rcBlockAxes) capacities(n float64) [2]float64 {
	var states [4][2]float64
	for axis, b := range axes {
		for j, side := range []float64{1, -1} {
			pt := b.atN(n, side)
			states[axis*2+j][axis] = pt.m
			states[axis*2+j][1-axis] = pt.transverse
		}
	}
	var result [2]float64
	for i := range result {
		lo, hi := rcAxisIntercepts(states, i)
		result[i] = math.Min(-lo, hi)
	}
	return result
}
func (axes rcBlockAxes) axialLimit(limit float64) float64 {
	feasible := func(n float64) bool { m := axes.capacities(n); return m[0] >= 0 && m[1] >= 0 }
	if feasible(limit) {
		return limit
	}
	inside, outside := 0., limit
	for range 48 {
		mid := (inside + outside) / 2
		if feasible(mid) {
			inside = mid
		} else {
			outside = mid
		}
	}
	return inside
}
func rcBlockCurves(rc *RCColumn, ndiv int) (curves [2]rcNMCurve, err error) {
	// Subtract actual steel disks from the concrete block. Reject overlapping
	// or protruding disks so the concrete area cannot be subtracted twice.
	for i, bar := range rc.Reins {
		r := bar.Radius()
		for axis, x := range bar.Position {
			if x-r < rc.Bound(axis) || x+r > rc.Bound(axis+2) {
				return curves, fmt.Errorf("REINS %d protrudes from CRECT", i+1)
			}
		}
		for j, other := range rc.Reins[:i] {
			if math.Hypot(bar.Position[0]-other.Position[0], bar.Position[1]-other.Position[1])+1e-8 < r+other.Radius() {
				return curves, fmt.Errorf("REINS %d and %d overlap", j+1, i+1)
			}
		}
	}
	axes := rcBlockAxes{newRCBlock(rc, 1), newRCBlock(rc, 0)}
	atZero := axes.capacities(0)
	if !positiveYieldInputs(atZero[0], atZero[1]) {
		return curves, fmt.Errorf("no pure-axis bending capacity at zero axial force")
	}
	low := axes.axialLimit(axes[0].forces(0, 1).n)
	high := axes.axialLimit(axes[0].forces(1, 1).n)
	for axis := range curves {
		curves[axis].points = make([]rcNMPoint, ndiv+1)
	}
	for i := 0; i <= ndiv; i++ {
		n := high - float64(i)*(high-low)/float64(ndiv)
		if i == ndiv {
			n = low
		}
		moments := axes.capacities(n)
		for axis, m := range moments {
			cv := &curves[axis]
			pt := rcNMPoint{n, math.Max(0, m)}
			cv.points[i] = pt
			if pt.m > cv.peak.m {
				cv.peak = pt
			}
		}
	}
	return curves, nil
}
