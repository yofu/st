package st

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func rc301Fixture() *RCGirder {
	rg := NewRCGirder(301)
	rg.CShape = NewCRect([]float64{-25, -100, 25, 100})
	rg.Concrete = FC24
	for _, x := range []float64{-17.2, 17.2} {
		for _, y := range []float64{-89.2, -79.2, 79.2, 89.2} {
			rg.Reins = append(rg.Reins, Reinforce{Area: 5.067, Position: []float64{x, y}, Material: SD345})
		}
	}
	rg.Nreins = len(rg.Reins)
	rg.Hoops = Hoop{Ps: []float64{.000633, .002534}, Material: SD295}
	return rg
}
func rcNear(t *testing.T, got, want, tol float64) {
	t.Helper()
	if !finiteRC(got) || math.Abs(got-want) > tol {
		t.Fatalf("got %.12g, want %.12g (tolerance %g)", got, want, tol)
	}
}
func TestRCYieldNM301(t *testing.T) {
	rg := rc301Fixture()
	for _, tc := range []struct {
		name         string
		offset, peak float64
	}{{"LST", 0, 689.810605}, {"NMinteraction 301 strong", 3, 694.06689}} {
		t.Run(tc.name, func(t *testing.T) {
			r := rg.Snapshot().(*RCGirder)
			for i := range r.Reins {
				pos := &r.Reins[i].Position[1]
				*pos += math.Copysign(tc.offset, *pos)
			}
			c := NewCondition()
			c.Period = "U"
			cv := rcNMReference(&r.RCColumn, c, 20)
			last := cv.points[20]
			rcNear(t, last.n, -141.876, 1e-8)
			rcNear(t, cv.peak.n, 933.24624, 1e-7)
			rcNear(t, 2*cv.peak.n-last.n, 2008.36848, 1e-7)
			rcNear(t, cv.peak.m, tc.peak, 1e-5)
			if tc.offset != 0 {
				rcNear(t, cv.points[19].m, 117.18832, 1e-5)
				gap := cv.excess(2008.36848, last.n, cv.peak.m, 1.5, .95)
				if gap.m < 39 {
					t.Fatal("excess hidden", gap)
				}
			}
			c.Strong = false
			rcNear(t, rcNMReference(&r.RCColumn, c, 20).peak.m, 161.746522, .000001)
			c.Period = "S"
			rcNear(t, r.Mza(c), 27.930546, .000001)
		})
	}
}

var rcSheetBeams = []struct {
	id, member     int
	b, h, m, q, kn float64
}{
	{301, 3680, 500, 2000, 865.502, 1014.570, 3535.57447348419},
	{302, 3654, 500, 900, 144.978, 114.093, 757.639131754102},
	{303, 1749, 350, 2000, 503.225, 393.687, 2106.12871402761},
	{304, 3718, 350, 1620, 22.828, 77.457, 3801.24947492794},
	{305, 3319, 300, 900, 97.043, 99.094, 628.435274722350},
	{306, 1023, 500, 1350, 314.835, 135.693, 1011.91472149127},
}

func TestRCYieldQsuWorkbook(t *testing.T) {
	for _, row := range rcSheetBeams {
		v := RCShearInput{Width: row.b, Depth: row.h - 100, TensionArea: 2026.8, HoopRatio: 253.4 / (200 * row.b), HoopStrength: 295, ConcreteStrength: 24, Moment: row.m, Shear: row.q}
		q, err := v.UltimateShear()
		if err != nil {
			t.Fatal(err)
		}
		rcNear(t, q.KN, row.kn, .000001)
	}
	v := RCShearInput{Width: 500, Depth: 1900, TensionArea: 2026.8, HoopRatio: .025, HoopStrength: 295, ConcreteStrength: 24, Moment: 1, Shear: 1}
	q, err := v.UltimateShear()
	if err != nil || q.Pw != .012 {
		t.Fatal(q, err)
	}
	v.Shear = 0
	if _, err = v.UltimateShear(); err == nil {
		t.Fatal("zero Q accepted")
	}
}

func TestRCYieldNMCommandUsesReferenceGrid(t *testing.T) {
	_, w, a, _ := rcResultFixture()
	t.Chdir(t.TempDir())
	for _, method := range []string{"", " -nm=block"} {
		report := yieldMessage(t, w, ":rcyield -sect=301"+method)
		if !strings.Contains(report, "preview_id=") {
			t.Fatal(report)
		}
		yieldMessage(t, w, ":rcyield -apply")
		yieldMessage(t, w, ":nminteraction 301 -ndiv=20 -period=U -output=ref.txt"+method)
		data, err := os.ReadFile("ref.txt")
		if err != nil {
			t.Fatal(err)
		}
		for _, j := range []int{0, 8, 10} {
			line := fmt.Sprintf("%s %9.3f %s %9.3f", sectionYieldNames[j], a.Sect.Yield[j], sectionYieldNames[j+1], a.Sect.Yield[j+1])
			if !strings.Contains(string(data), line) {
				t.Fatal(method, line, string(data))
			}
		}
	}
}

func TestRCYieldBlockDirectional(t *testing.T) {
	rc := rc301Fixture().RCColumn
	// Remove a bottom row. Reversing the section exchanges the two moments.
	rc.Reins = slices.DeleteFunc(rc.Reins, func(r Reinforce) bool { return r.Position[1] == -79.2 })
	rc.Nreins = len(rc.Reins)
	if rcSymmetric(&rc) {
		t.Fatal("unbalanced bars marked symmetric")
	}
	b := newRCBlock(&rc, 1)
	// Closed-form reference for a chosen NA depth c=80 cm, fully included
	// compression bars, outside-block tension bars, strains limited by Fs.
	var n, m float64
	for _, bar := range rc.Reins {
		y := bar.Position[1]
		sig := math.Max(-3.5, math.Min(3.5, 2100*.003*(1-(100-y)/80)))
		if y > 100-.85*80 {
			sig -= .85 * .24
		}
		n += bar.Area * sig
		m += bar.Area * sig * y / 100
	}
	cn := .85 * .24 * 50 * .85 * 80
	n += cn
	m += cn * (100 - .85*80/2) / 100
	rcNear(t, b.atN(n, 1).m, m, 1e-8)
	pos, neg := b.atN(0, 1).m, b.atN(0, -1).m
	if !(pos > 0 && neg < 0 && math.Abs(pos+neg) > 1) {
		t.Fatalf("direction lost %g %g", pos, neg)
	}
	s := rc.Snapshot().(*RCColumn)
	for i := range s.Reins {
		s.Reins[i].Position[1] *= -1
	}
	flipped := newRCBlock(s, 1)
	rcNear(t, flipped.atN(0, 1).m, -neg, 1e-8)
	cs, err := rcBlockCurves(&rc, 20)
	if err != nil {
		t.Fatal(err)
	}
	for _, pt := range cs[0].points {
		rcNear(t, pt.m, math.Max(0, math.Min(b.atN(pt.n, 1).m, -b.atN(pt.n, -1).m)), 1e-7)
	}
	// A four-state section envelope whose positive x extremum carries My.
	lo, hi := rcAxisIntercepts([4][2]float64{{10, 2}, {-8, 0}, {0, 5}, {0, -3}}, 0)
	rcNear(t, lo, -8, 1e-12)
	rcNear(t, hi, 6, 1e-12)
}

type rcUndoWindow struct {
	*steelYieldCommandWindow
	stack *UndoStack
}

func (w *rcUndoWindow) UndoEnabled() bool            { return true }
func (w *rcUndoWindow) PushUndo(f *Frame)            { w.stack.PushUndo(f) }
func (w *rcUndoWindow) CaptureUndoBaseline(f *Frame) { w.stack.CaptureUndoBaseline(f) }

func TestRCYieldNakanoReview(t *testing.T) {
	f := readNakanoSteelYieldModel(t)
	dir := t.TempDir()
	t.Chdir(dir)
	f.Path = filepath.Join(dir, "base.inp")
	w := &rcUndoWindow{&steelYieldCommandWindow{dxfDimensionCommandWindow: &dxfDimensionCommandWindow{frame: f}}, NewUndoStack(10)}
	for _, row := range rcSheetBeams {
		s := f.Sects[row.id]
		before := slices.Clone(s.Yield)
		cmd := fmt.Sprintf(":rcyield -elem=%d -qy=%g,%g", row.member, row.m, row.q)
		report := yieldMessage(t, w, cmd)
		if !strings.Contains(report, "preview_id=") || !strings.Contains(report, "QX 保留") || !slices.Equal(before, s.Yield) {
			t.Fatal(report)
		}
		yieldMessage(t, w, ":rcyield -apply")
		rcNear(t, s.Yield[4], math.Round(row.kn/rcGravity*1000)/1000, 1e-9)
		rcNear(t, s.Yield[5], -s.Yield[4], 1e-10)
		if s.Yield[2] != before[2] || s.Yield[3] != before[3] {
			t.Fatal("held shear overwritten")
		}
		if row.id == 301 {
			rcNear(t, s.Yield[8], 689.811, 1e-10)
			undo, err := w.stack.Undo()
			if err != nil || !slices.Equal(undo.Sects[301].Yield, before) {
				t.Fatal("undo", err)
			}
			redo, err := w.stack.Redo()
			if err != nil || !slices.Equal(redo.Sects[301].Yield, s.Yield) {
				t.Fatal("redo", err)
			}
		}
	}
	path := filepath.Join(dir, "rc-reviewed.inp")
	if err := f.WriteInp(path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	reopened := NewFrame()
	if err = reopened.ReadInpSnapshot(data, path); err != nil {
		t.Fatal(err)
	}
	if err = reopened.ExtractArclm(filepath.Join(dir, "test.wgt")); err != nil {
		t.Fatal(err)
	}
	for _, row := range rcSheetBeams {
		want := f.Sects[row.id].Yield
		if !slices.Equal(reopened.Sects[row.id].Yield, want) {
			t.Fatal("saved SECT", row.id)
		}
		for _, p := range []string{"L", "X", "Y"} {
			found := false
			for _, s := range reopened.Arclms[p].Sects {
				if s.Num == row.id {
					found = true
					if !slices.Equal(s.Yield, want) {
						t.Fatal(p, row.id, s.Yield)
					}
				}
			}
			if !found {
				t.Fatal("missing section", p, row.id)
			}
		}
	}
}
