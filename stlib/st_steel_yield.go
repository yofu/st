package st

import (
	"fmt"
	"math"
)

// SteelYieldCondition uses effective lengths in cm about the physical strong and
// weak axes. LateralLength is the unbraced length for major-axis bending.
// The lengths already include any end-restraint factors. When Ix == Iy,
// StrongBucklingLength applies to the local x axis.
type SteelYieldCondition struct {
	StrongBucklingLength float64
	WeakBucklingLength   float64
	LateralLength        float64
	EndMomentRatio       float64 // M2/M1, in [-1, 1], using the Cold01 sign convention
	PlateBucklingFactor  float64 // shear buckling coefficient k
	StrengthFactor       float64 // multiplier applied once to the finished capacities
}

// HBeamYieldCondition is the condition used by HBeamYield.
type HBeamYieldCondition = SteelYieldCondition

func (c SteelYieldCondition) validate() error {
	if !positiveYieldInputs(c.StrongBucklingLength, c.WeakBucklingLength, c.LateralLength, c.PlateBucklingFactor, c.StrengthFactor) {
		return fmt.Errorf("effective lengths, plate buckling factor and strength factor must be finite and positive")
	}
	if math.IsNaN(c.EndMomentRatio) || math.Abs(c.EndMomentRatio) > 1 {
		return fmt.Errorf("end moment ratio must be in [-1, 1]")
	}
	return nil
}

func (c SteelYieldCondition) momentGradientFactor() float64 {
	r := c.EndMomentRatio
	return math.Min(1.75+1.05*r+0.3*r*r, 2.3)
}

func (c SteelYieldCondition) checkShearYieldRange(bt []float64, e, f float64) error {
	limit := 1.07 * math.Sqrt(c.PlateBucklingFactor*e/f)
	if math.Max(bt[0], bt[1]) > limit {
		return fmt.Errorf("plate shear buckling outside the supported yield range (web %g, flange %g, limit %g)", bt[0], bt[1], limit)
	}
	return nil
}

// HBeamYield calculates the Cold01 H-beam limits for SN400/SN490 sections of
// width-thickness class P-I-1 through P-II whose plates reach shear yield.
// Results are unrounded, in Sect.Yield
// order (NZ, QX, QY in tf, MZ, MX, MY in tf m, MAX then MIN for each).
func (beam *SGirder) HBeamYield(c HBeamYieldCondition) ([12]float64, error) {
	var limits [12]float64
	var h HKYOU
	var rotated bool
	switch shape := beam.Shape.(type) {
	case HKYOU:
		h = shape
	case HWEAK:
		h, rotated = HKYOU(shape), true
	default:
		return limits, fmt.Errorf("H-beam yield: unsupported shape %T", beam.Shape)
	}
	if beam.Steel.Name() != "SN400" && beam.Steel.Name() != "SN490" {
		return limits, fmt.Errorf("H-beam yield: unsupported steel %q", beam.Steel.Name())
	}
	if !positiveYieldInputs(h.H, h.B, h.Tw, h.Tf) || h.H <= 2*h.Tf || h.B <= h.Tw || h.Ix() < h.Iy() {
		return limits, fmt.Errorf("H-beam yield: invalid major-axis section dimensions (cm): H=%g B=%g tw=%g tf=%g", h.H, h.B, h.Tw, h.Tf)
	}
	if !positiveYieldInputs(beam.F, beam.EL()) {
		return limits, fmt.Errorf("H-beam yield: material strength and modulus must be finite and positive")
	}
	if err := c.validate(); err != nil {
		return limits, fmt.Errorf("H-beam yield: %w", err)
	}
	if class := h.BT_ratio_category(beam.Steel); class > P_II {
		return limits, fmt.Errorf("H-beam yield: local buckling resistance for width-thickness class %s is not supported", BTRATIO[class])
	}
	if err := c.checkShearYieldRange(h.BT_ratio(), beam.EL(), beam.F); err != nil {
		return limits, fmt.Errorf("H-beam yield: %w", err)
	}

	// Nc1 and Me operate on a section in its principal axes. Use the supplied
	// effective lengths and the gross section once (independent of LST MULTI).
	section := NewSColumn(beam.Num(), h, beam.Steel)
	section.BBLength = []float64{c.StrongBucklingLength, c.WeakBucklingLength}
	nc := section.Nc1(&Condition{})
	ny := section.Ny()
	f := beam.F
	me := section.Me(c.LateralLength, c.momentGradientFactor())
	mx := steelBendingLimit(f*h.Zpx()/100, me, 0.6+0.3*c.EndMomentRatio)
	my := f * h.Zpy() / 100
	mz := f * h.Torsion() / math.Sqrt(3) / 100

	// Workbook rows 50 and 51 use both gross flanges for QX and the clear web for QY.
	qx := 0.6 * f * (2 * h.B * h.Tf)
	qy := 0.6 * f * h.Asy()
	if rotated {
		qx, qy = qy, qx
		mx, my = my, mx
	}
	return steelYieldLimits(nc, ny, qx, qy, mz, mx, my, c.StrengthFactor)
}

func steelYieldLimits(nc, ny, qx, qy, mz, mx, my, factor float64) ([12]float64, error) {
	limits := [12]float64{nc, ny, qx, qx, qy, qy, mz, mz, mx, mx, my, my}
	for i, capacity := range limits {
		v := capacity * factor
		if !positiveYieldInputs(v) {
			return [12]float64{}, fmt.Errorf("steel yield: capacity outside numeric range")
		}
		if i%2 == 1 {
			v = -v
		}
		limits[i] = v
	}
	return limits, nil
}

func positiveYieldInputs(values ...float64) bool {
	for _, v := range values {
		if !(v > 0) || math.IsInf(v, 1) {
			return false
		}
	}
	return true
}

// Cold01 lateral buckling curve. Both input moments and the result are in tf m.
func steelBendingLimit(mp, me, plasticSlenderness float64) float64 {
	lambda := math.Sqrt(mp / me)
	elasticSlenderness := 1 / math.Sqrt(0.6)
	switch {
	case lambda <= plasticSlenderness:
		return mp
	case lambda <= elasticSlenderness:
		return (1 - 0.4*(lambda-plasticSlenderness)/(elasticSlenderness-plasticSlenderness)) * mp
	default:
		return me
	}
}
