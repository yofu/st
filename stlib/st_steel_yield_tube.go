package st

import (
	"fmt"
	"math"
)

// ColdFormedTubeProduct holds the tabulated properties of one cold-formed
// rectangular hollow section. Shape gives the nominal RPIPE dimensions in cm;
// X and Y are the local section axes. Name identifies the product in diagnostics.
type ColdFormedTubeProduct struct {
	Name     string
	Shape    RPIPE
	Area     float64 // cm²
	Ix, Iy   float64 // cm⁴
	Zpx, Zpy float64 // cm³
}

// ColdFormedTubeYield calculates Cold01's cold-formed column sheet, returning
// unrounded Sect.Yield limits (forces in tf, moments in tf m). That sheet takes
// the local compression and bending limits as Ny and Mp. A, I and Zp come from
// the product table; shear areas and closed-section torsion use the nominal tube.
// The supported shear range is the plate-yield branch of rows 42–43.
func (column *SColumn) ColdFormedTubeYield(product ColdFormedTubeProduct, c SteelYieldCondition) ([12]float64, error) {
	return column.coldFormedTubeYield(product, c, c.LateralLength, c.LateralLength)
}

func (column *SColumn) coldFormedTubeYield(product ColdFormedTubeProduct, c SteelYieldCondition, lateralX, lateralY float64) ([12]float64, error) {
	var zero [12]float64
	tube, ok := column.Shape.(RPIPE)
	if !ok {
		return zero, fmt.Errorf("cold-formed tube yield: expected RPIPE, got %T", column.Shape)
	}
	if !positiveYieldInputs(tube.H, tube.B, tube.Tw, tube.Tf) || tube.H <= 2*tube.Tf || tube.B <= 2*tube.Tw {
		return zero, fmt.Errorf("cold-formed tube yield: invalid RPIPE dimensions (cm): %+v", tube)
	}
	if err := product.validate(tube); err != nil {
		return zero, err
	}
	if err := c.validate(); err != nil {
		return zero, fmt.Errorf("cold-formed tube yield: %w", err)
	}
	f, e := column.F, column.EL()
	g := e / (2 * (1 + column.Poi()))
	if !positiveYieldInputs(f, e, g) {
		return zero, fmt.Errorf("cold-formed tube yield: material strength and moduli must be finite and positive")
	}
	if err := c.checkShearYieldRange(tube.BT_ratio(), e, f); err != nil {
		return zero, fmt.Errorf("cold-formed tube yield: %w", err)
	}

	lx, ly := c.StrongBucklingLength, c.WeakBucklingLength
	if product.Iy > product.Ix {
		lx, ly = ly, lx
	}
	ny := f * product.Area
	ne := math.Pi * math.Pi * e * math.Min(product.Ix/(lx*lx), product.Iy/(ly*ly))
	nc := steelFlexuralBucklingLimit(ny, ne)

	meTimesLength := c.momentGradientFactor() * math.Pi * math.Sqrt(e*math.Min(product.Ix, product.Iy)*g*tube.J()) / 100
	mx, my := f*product.Zpx/100, f*product.Zpy/100
	mp := math.Max(mx, my)
	plasticSlenderness := 0.6 + 0.3*c.EndMomentRatio
	// Rows 37–38 reduce the axis with larger Mp, or both axes if they are equal.
	if mx == mp {
		mx = steelBendingLimit(mp, meTimesLength/lateralX, plasticSlenderness)
	}
	if my == mp {
		my = steelBendingLimit(mp, meTimesLength/lateralY, plasticSlenderness)
	}

	qx := 0.6 * f * tube.Asx()
	qy := 0.6 * f * tube.Asy()
	mz := f / math.Sqrt(3) * tube.Torsion() / 100
	return steelYieldLimits(nc, ny, qx, qy, mz, mx, my, c.StrengthFactor)
}

func (p ColdFormedTubeProduct) validate(tube RPIPE) error {
	// Allow only unit-conversion roundoff when matching the nominal size.
	const toleranceCM = 1e-8
	for _, delta := range [...]float64{tube.H - p.Shape.H, tube.B - p.Shape.B, tube.Tw - p.Shape.Tw, tube.Tf - p.Shape.Tf} {
		if !(math.Abs(delta) <= toleranceCM) {
			return fmt.Errorf("cold-formed tube yield: product %q size %s does not match model %s", p.Name, p.Shape, tube)
		}
	}
	if !positiveYieldInputs(p.Area, p.Ix, p.Iy, p.Zpx, p.Zpy) {
		return fmt.Errorf("cold-formed tube yield: product %q requires finite positive A, Ix, Iy, Zpx and Zpy", p.Name)
	}
	return nil
}
