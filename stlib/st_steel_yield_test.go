package st

import (
	"fmt"
	"math"
	"os"
	"testing"
)

// Cold01.xlsx, 断面性能_H鋼梁!C54:I65, rounded to the three decimals of SECT.
// Column I has a duplicated number 512 in row 5; B10 and 100x100x6x8 identify 513.
var nakanoHBeams = []struct {
	num   int
	shape HKYOU
	want  [12]float64
}{
	{501, HKYOU{50, 20, 1, 1.6}, [12]float64{169.194, -265.920, 92.160, -92.160, 67.392, -67.392, 0.608, -0.608, 40.270, -40.270, 7.961, -7.961}},
	{502, HKYOU{24.4, 17.5, 0.7, 1.1}, [12]float64{80.868, -129.696, 55.440, -55.440, 22.378, -22.378, 0.228, -0.228, 10.306, -10.306, 4.108, -4.108}},
	{503, HKYOU{45, 20, 0.9, 1.4}, [12]float64{144.983, -225.552, 80.640, -80.640, 54.691, -54.691, 0.464, -0.464, 31.213, -31.213, 6.925, -6.925}},
	{504, HKYOU{40, 20, 0.8, 1.3}, [12]float64{128.948, -196.608, 74.880, -74.880, 43.085, -43.085, 0.380, -0.380, 24.969, -24.969, 6.384, -6.384}},
	{511, HKYOU{24.4, 17.5, 0.7, 1.1}, [12]float64{80.868, -129.696, 55.440, -55.440, 22.378, -22.378, 0.228, -0.228, 10.306, -10.306, 4.108, -4.108}},
	{512, HKYOU{20, 10, 0.55, 0.8}, [12]float64{14.421, -62.688, 23.040, -23.040, 14.573, -14.573, 0.077, -0.077, 2.944, -2.944, 0.993, -0.993}},
	{513, HKYOU{10, 10, 0.6, 0.8}, [12]float64{14.409, -50.496, 23.040, -23.040, 7.258, -7.258, 0.070, -0.070, 1.535, -1.535, 0.978, -0.978}},
}

func cold01HBeamCondition() HBeamYieldCondition {
	return HBeamYieldCondition{
		StrongBucklingLength: 400, WeakBucklingLength: 400, LateralLength: 400,
		EndMomentRatio: -1, PlateBucklingFactor: 5.4, StrengthFactor: 1,
	}
}

func checkCold01HBeam(t *testing.T, beam *SGirder, want [12]float64) {
	t.Helper()
	got, err := beam.HBeamYield(cold01HBeamCondition())
	if err != nil {
		t.Fatal(err)
	}
	checkCold01Limits(t, beam.Num(), got, want)
}

func checkCold01Limits(t *testing.T, num int, got, want [12]float64) {
	t.Helper()
	names := [...]string{"NZMAX", "NZMIN", "QXMAX", "QXMIN", "QYMAX", "QYMIN", "MZMAX", "MZMIN", "MXMAX", "MXMIN", "MYMAX", "MYMIN"}
	for i, name := range names {
		if fmt.Sprintf("%.3f", got[i]) != fmt.Sprintf("%.3f", want[i]) {
			t.Errorf("SECT %d %s = %.12g; want %.3f at INP precision", num, name, got[i], want[i])
		}
	}
}

func TestHBeamYieldCold01(t *testing.T) {
	for _, tc := range nakanoHBeams {
		t.Run(fmt.Sprint(tc.num), func(t *testing.T) {
			checkCold01HBeam(t, NewSGirder(tc.num, tc.shape, SN400), tc.want)
		})
	}
}

func readNakanoSteelYieldModel(t *testing.T) *Frame {
	t.Helper()
	path := os.Getenv("ST_STEEL_YIELD_TEST_INP")
	if path == "" {
		t.Skip("set ST_STEEL_YIELD_TEST_INP to nakano02_07.inp with its companion LST")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	frame := NewFrame()
	if err := frame.ReadInpSnapshot(data, path); err != nil {
		t.Fatal(err)
	}
	if err := frame.ReadLst(Ce(path, ".lst")); err != nil {
		t.Fatal(err)
	}
	return frame
}

// Exercise the INP SHAPE/PROP and the independently parsed LST on the original model.
func TestHBeamYieldNakanoModel(t *testing.T) {
	frame := readNakanoSteelYieldModel(t)
	for _, tc := range nakanoHBeams {
		t.Run(fmt.Sprint(tc.num), func(t *testing.T) {
			s := frame.Sects[tc.num]
			if s == nil || len(s.Figs) != 1 {
				t.Fatalf("missing single-figure SECT %d", tc.num)
			}
			fromFig, ok := s.Figs[0].GetSectionRate(tc.num, GIRDER).(*SGirder)
			if !ok {
				t.Fatal("missing INP steel girder")
			}
			fromLst, ok := s.Allow.(*SGirder)
			if !ok {
				t.Fatal("missing LST steel girder")
			}
			checkCold01HBeam(t, fromFig, tc.want)
			checkCold01HBeam(t, fromLst, tc.want)
		})
	}
}

func TestHBeamYieldEffectiveLengthsAndAxes(t *testing.T) {
	h := nakanoHBeams[0].shape
	beam := NewSGirder(1, h, SN400)
	c := cold01HBeamCondition()
	c.StrongBucklingLength, c.WeakBucklingLength = 2000, 10
	c.LateralLength, c.EndMomentRatio = 1000, 1
	limits, err := beam.HBeamYield(c)
	if err != nil {
		t.Fatal(err)
	}
	// Strong-axis flexural buckling governs; the moment gradient reaches the Cb cap.
	if math.Abs(limits[0]-160.363526927) > 1e-8 || math.Abs(limits[8]-41.880830639) > 1e-8 {
		t.Fatalf("NZMAX=%g MXMAX=%g", limits[0], limits[8])
	}
	c.StrengthFactor = 1.25
	weak := NewSGirder(2, HWEAK(h), SN400)
	rotated, err := weak.HBeamYield(c)
	if err != nil {
		t.Fatal(err)
	}
	for i, src := range [...]int{0, 1, 4, 5, 2, 3, 6, 7, 10, 11, 8, 9} {
		if rotated[i] != limits[src]*c.StrengthFactor {
			t.Errorf("rotated/scaled limit %d = %g, want %g", i, rotated[i], limits[src]*c.StrengthFactor)
		}
	}

	c = cold01HBeamCondition()
	for _, length := range []float64{1, 4000} {
		c.LateralLength = length
		got, err := beam.HBeamYield(c)
		if err != nil {
			t.Fatal(err)
		}
		// Cold01 rows 38, 42, 45: full plastic moment at short Lb and Me at long Lb.
		want := beam.F * h.Zpx() / 100
		if length == 4000 {
			want = beam.Me(length, 1)
		}
		if math.Abs(got[8]-want) > 1e-12 || math.Abs(got[0]-169.19386856321404) > 1e-10 {
			t.Errorf("Lb=%g: MXMAX=%g want %g; NZMAX=%g", length, got[8], want, got[0])
		}
	}
}

func TestHBeamYieldRejectsUnsupportedInputs(t *testing.T) {
	h := nakanoHBeams[0].shape
	for _, tc := range []struct {
		name string
		edit func(*SGirder, *HBeamYieldCondition)
	}{
		{"zero effective length", func(_ *SGirder, c *HBeamYieldCondition) { c.WeakBucklingLength = 0 }},
		{"NaN moment ratio", func(_ *SGirder, c *HBeamYieldCondition) { c.EndMomentRatio = math.NaN() }},
		{"infinite length", func(_ *SGirder, c *HBeamYieldCondition) { c.LateralLength = math.Inf(1) }},
		{"moment ratio out of range", func(_ *SGirder, c *HBeamYieldCondition) { c.EndMomentRatio = 1.1 }},
		{"zero multiplier", func(_ *SGirder, c *HBeamYieldCondition) { c.StrengthFactor = 0 }},
		{"slender web", func(b *SGirder, _ *HBeamYieldCondition) { b.Shape = HKYOU{50, 20, 0.4, 1.6} }},
		{"shear buckling", func(_ *SGirder, c *HBeamYieldCondition) { c.PlateBucklingFactor = 1.5 }},
		{"invalid shape dimensions", func(b *SGirder, _ *HBeamYieldCondition) { b.Shape = HKYOU{2, 20, 1, 1.6} }},
		{"non-H section", func(b *SGirder, _ *HBeamYieldCondition) { b.Shape = RPIPE{} }},
		{"unknown steel", func(b *SGirder, _ *HBeamYieldCondition) { b.Steel.name = "unknown" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, c := NewSGirder(1, h, SN400), cold01HBeamCondition()
			tc.edit(b, &c)
			if got, err := b.HBeamYield(c); err == nil || got != ([12]float64{}) {
				t.Fatalf("got %v, %v; want no capacities and an error", got, err)
			}
		})
	}
}
