package st

import (
	"fmt"
	"math"
	"testing"
)

// From Cold01.xlsx, 断面性能_冷間パイプ柱!C:E: catalog rows 11–13, 17–18;
// the twelve signed limits come independently from rows 46–57 at INP precision.
var nakanoTubeColumns = []struct {
	num     int
	product ColdFormedTubeProduct
	want    [12]float64
}{
	{201, ColdFormedTubeProduct{Name: "Cold01 C1", Shape: RPIPE{35, 35, 1.2, 1.2}, Area: 158.5, Ix: 29800, Iy: 29800, Zpx: 1990, Zpy: 1990},
		[12]float64{368.120, -380.400, 112.666, -112.666, 112.666, -112.666, 37.992, -37.992, 47.760, -47.760, 47.760, -47.760}},
	{202, ColdFormedTubeProduct{Name: "Cold01 C2", Shape: RPIPE{30, 30, 1.2, 1.2}, Area: 134.5, Ix: 18300, Iy: 18300, Zpx: 1440, Zpy: 1440},
		[12]float64{306.827, -322.800, 95.386, -95.386, 95.386, -95.386, 27.583, -27.583, 34.560, -34.560, 34.560, -34.560}},
	{203, ColdFormedTubeProduct{Name: "Cold01 C3", Shape: RPIPE{25, 25, 1.2, 1.2}, Area: 110.5, Ix: 10300, Iy: 10300, Zpx: 975, Zpy: 975},
		[12]float64{245.716, -265.200, 78.106, -78.106, 78.106, -78.106, 18.837, -18.837, 23.400, -23.400, 23.400, -23.400}},
}

func cold01TubeCondition() SteelYieldCondition {
	return SteelYieldCondition{
		StrongBucklingLength: 285, WeakBucklingLength: 285, LateralLength: 285,
		EndMomentRatio: -1, PlateBucklingFactor: 5.4, StrengthFactor: 1,
	}
}

func checkCold01Tube(t *testing.T, col *SColumn, p ColdFormedTubeProduct, want [12]float64) {
	t.Helper()
	got, err := col.ColdFormedTubeYield(p, cold01TubeCondition())
	if err != nil {
		t.Fatal(err)
	}
	checkCold01Limits(t, col.Num(), got, want)
}

func TestColdFormedTubeYieldCold01(t *testing.T) {
	for _, tc := range nakanoTubeColumns {
		t.Run(fmt.Sprint(tc.num), func(t *testing.T) {
			col := NewSColumn(tc.num, tc.product.Shape, SN400)
			checkCold01Tube(t, col, tc.product, tc.want)
		})
	}
}

func TestColdFormedTubeYieldNakanoModel(t *testing.T) {
	frame := readNakanoSteelYieldModel(t)
	for _, tc := range nakanoTubeColumns {
		t.Run(fmt.Sprint(tc.num), func(t *testing.T) {
			s := frame.Sects[tc.num]
			if s == nil || len(s.Figs) != 1 {
				t.Fatalf("missing single-figure SECT %d", tc.num)
			}
			fromFig, ok := s.Figs[0].GetSectionRate(tc.num, COLUMN).(*SColumn)
			if !ok {
				t.Fatal("missing INP steel column")
			}
			fromLst, ok := s.Allow.(*SColumn)
			if !ok {
				t.Fatal("missing LST steel column")
			}
			t.Run("SHAPE", func(t *testing.T) { checkCold01Tube(t, fromFig, tc.product, tc.want) })
			t.Run("LST", func(t *testing.T) { checkCold01Tube(t, fromLst, tc.product, tc.want) })
		})
	}
}

func TestColdFormedTubeYieldProductSelection(t *testing.T) {
	p := nakanoTubeColumns[0].product
	col := NewSColumn(201, p.Shape, SN400)
	c := cold01TubeCondition()
	c.LateralLength = 100000 // Both equal plastic moments reach the elastic buckling branch.
	got, err := col.ColdFormedTubeYield(p, c)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got[8]-15.20394531646539) > 1e-10 || got[8] != got[10] {
		t.Fatalf("equal axes: MXMAX=%g MYMAX=%g", got[8], got[10])
	}

	// A second catalog entry at exactly the same nominal size. Change only the
	// tabulated properties; shear and torsion still come from the original RPIPE.
	p.Area *= 0.9
	p.Ix /= 2
	p.Zpx /= 2
	c.LateralLength = 285
	second, err := col.ColdFormedTubeYield(p, c)
	if err != nil {
		t.Fatal(err)
	}
	if second[0] >= got[0] || math.Abs(second[1]-0.9*got[1]) > 1e-10 ||
		math.Abs(second[8]-23.88) > 1e-10 || math.Abs(second[10]-47.76) > 1e-10 {
		t.Fatalf("changed catalog properties: limits %v", second)
	}
	for i := 2; i < 8; i++ {
		if second[i] != got[i] {
			t.Errorf("geometry-derived limit %d changed from %g to %g", i, got[i], second[i])
		}
	}
}

func TestColdFormedTubeYieldRectangularAxes(t *testing.T) {
	p := ColdFormedTubeProduct{Shape: RPIPE{35, 25, 1, 1.2}, Area: 125, Ix: 20000, Iy: 12000, Zpx: 1500, Zpy: 1070}
	c := cold01TubeCondition()
	c.StrongBucklingLength, c.WeakBucklingLength = 3000, 280
	c.LateralLength, c.EndMomentRatio = 100000, 0
	got, err := NewSColumn(1, p.Shape, SN400).ColdFormedTubeYield(p, c)
	if err != nil {
		t.Fatal(err)
	}
	want := [12]float64{38.382, -300, 79.488, -79.488, 93.888, -93.888, 22.481, -22.481, 12.268, -12.268, 25.680, -25.680}
	checkCold01Limits(t, 1, got, want)

	// Rotate both catalog and shape. Strong/weak effective lengths follow the
	// section while the signed force and moment pairs follow the local axes.
	p.Shape = RPIPE{25, 35, 1.2, 1}
	p.Ix, p.Iy = p.Iy, p.Ix
	p.Zpx, p.Zpy = p.Zpy, p.Zpx
	c.StrengthFactor = 1.25
	rotated, err := NewSColumn(2, p.Shape, SN400).ColdFormedTubeYield(p, c)
	if err != nil {
		t.Fatal(err)
	}
	for i, src := range [...]int{0, 1, 4, 5, 2, 3, 6, 7, 10, 11, 8, 9} {
		if math.Abs(rotated[i]-got[src]*c.StrengthFactor) > 1e-10 {
			t.Errorf("rotated limit %d = %g, want %g", i, rotated[i], got[src]*c.StrengthFactor)
		}
	}
}

func TestColdFormedTubeYieldInvalidInputs(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*SColumn, *ColdFormedTubeProduct, *SteelYieldCondition)
	}{
		{"different catalog size", func(_ *SColumn, p *ColdFormedTubeProduct, _ *SteelYieldCondition) { p.Shape.Tw = 1 }},
		{"missing catalog Zpy", func(_ *SColumn, p *ColdFormedTubeProduct, _ *SteelYieldCondition) { p.Zpy = 0 }},
		{"invalid catalog inertia", func(_ *SColumn, p *ColdFormedTubeProduct, _ *SteelYieldCondition) { p.Ix = math.Inf(1) }},
		{"non-tube", func(col *SColumn, _ *ColdFormedTubeProduct, _ *SteelYieldCondition) { col.Shape = HKYOU{} }},
		{"zero length", func(_ *SColumn, _ *ColdFormedTubeProduct, c *SteelYieldCondition) { c.LateralLength = 0 }},
		{"plate shear buckling", func(_ *SColumn, _ *ColdFormedTubeProduct, c *SteelYieldCondition) { c.PlateBucklingFactor = 0.5 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, c := nakanoTubeColumns[0].product, cold01TubeCondition()
			col := NewSColumn(201, p.Shape, SN400)
			tc.edit(col, &p, &c)
			if got, err := col.ColdFormedTubeYield(p, c); err == nil || got != ([12]float64{}) {
				t.Fatalf("got %v, %v; want zero limits and an error", got, err)
			}
		})
	}
}
