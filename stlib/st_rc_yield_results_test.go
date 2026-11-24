package st

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"
)

func rcResultFixture() (f *Frame, w *steelYieldCommandWindow, a, b *Elem) {
	f = NewFrame()
	s := f.AddSect(301)
	s.Allow = rc301Fixture()
	s.Yield = []float64{2000, -140, 75, -75, 115, -115, 28, -28, 690, -690, 160, -160}
	a = f.AddLineElem(3051, []*Node{f.AddNode(0, 0, 0), f.AddNode(5, 0, 0)}, s, GIRDER)
	b = f.AddLineElem(3052, []*Node{a.Enod[1], f.AddNode(10, 0, 0)}, s, GIRDER)
	for i, e := range []*Elem{a, b} {
		e.Stress["X@80"] = map[int][]float64{e.Enod[0].Num: {0, 3, -11, 0, 9, -2}, e.Enod[1].Num: {0, -2, 7, 0, -(18 + float64(i)*12), 4}}
	}
	w = &steelYieldCommandWindow{dxfDimensionCommandWindow: &dxfDimensionCommandWindow{frame: f}, selected: []*Elem{a}}
	return
}
func TestRCYieldSelectedResults(t *testing.T) {
	f, w, a, b := rcResultFixture()
	original := slices.Clone(a.Sect.Yield)
	for _, cmd := range []string{":rcyield -sect=301 -period=X@79", ":rcyield -period=X@80 -axis=y", ":rcyield -elem=3051 -period=X@79"} {
		msg := yieldMessage(t, w, cmd)
		if strings.Contains(msg, "preview_id") || !strings.Contains(msg, "未反映") || !slices.Equal(a.Sect.Yield, original) {
			t.Fatal(cmd, msg)
		}
	}
	for _, cmd := range []string{":rcyield -apply", ":rcyield -qx=1,0", ":rcyield -qx=NaN,1", ":rcyield -ndiv=1", ":rcyield -qx=2,4 -axis=x -period=X@80"} {
		if err := exCommand(w, cmd, nil, nil); err == nil {
			t.Fatal("accepted", cmd)
		} else if _, ok := err.(Message); ok {
			t.Fatal("accepted", cmd, err)
		}
	}
	msg := yieldMessage(t, w, ":rcyield -period=X@80 -pick=min")
	if !strings.Contains(msg, "QY 採用 ELEM 3052") || !strings.Contains(msg, "QX 採用 ELEM 3051") {
		t.Fatal(msg)
	}
	// A result changed after review. The original token cannot authorize it.
	b.Stress["X@80"][b.Enod[1].Num][4] = -40
	if err := exCommand(w, ":rcyield -apply", nil, nil); err == nil || !strings.Contains(err.Error(), "preview does not match") || !slices.Equal(original, a.Sect.Yield) {
		t.Fatal(err)
	}
	msg = yieldMessage(t, w, ":rcyield -elem=3051 -period=X@80 -axis=x -qy=865.502,1014.57")
	_, after, _ := strings.Cut(msg, "確認後に実行  ")
	confirm, _, _ := strings.Cut(after, "\n")
	if !strings.Contains(confirm, "-member=3051") {
		t.Fatal("reference member lost", confirm)
	}
	f.rcYieldReview = nil // detached replay
	yieldMessage(t, w, confirm)
	// Qx uses My, Qx. Independent maxima are 4 tf m and 3 tf at opposite ends.
	v := RCShearInput{Width: 2000, Depth: 400, TensionArea: 2026.8, HoopRatio: .000633, HoopStrength: 295, ConcreteStrength: 24, Moment: 4 * rcGravity, Shear: 3 * rcGravity}
	want, err := v.UltimateShear()
	if err != nil {
		t.Fatal(err)
	}
	rcNear(t, a.Sect.Yield[2], math.Round(want.KN/rcGravity*1e3)/1e3, 1e-9)
	rcNear(t, a.Sect.Yield[4], 360.528, 1e-9)
}
func TestRCYieldFromINPAndColumns(t *testing.T) {
	_, w, a, b := rcResultFixture()
	s := a.Sect
	s.Allow = nil
	fig := NewFig()
	fig.Prop = &Prop{Material: FC24}
	fig.Shape = PLATE{H: 80, B: 50}
	fig.Reins = map[string][]string{"TREIN": {"4-D25", "SD345"}, "BREIN": {"4-D25", "SD345"}, "HOOP": {"2x2-D13@200", "SD295"}}
	s.Figs = []*Fig{fig}
	msg := yieldMessage(t, w, ":rcyield -sect=301")
	if !strings.Contains(msg, "INP SHAPE/配筋") || !strings.Contains(msg, "preview_id=") || s.Allow != nil {
		t.Fatal(msg)
	}
	a.Etype = COLUMN
	b.Etype = COLUMN
	s.Allow = &rc301Fixture().RCColumn
	msg = yieldMessage(t, w, ":rcyield -sect=301")
	if !strings.Contains(msg, "preview_id=") {
		t.Fatal(msg)
	}
	col := s.Allow.(*RCColumn)
	col.Reins = col.Reins[1:]
	col.Nreins--
	if m := yieldMessage(t, w, ":rcyield -sect=301"); strings.Contains(m, "preview_id=") || !strings.Contains(m, "-nm=block") {
		t.Fatal(m)
	}
	msg = yieldMessage(t, w, ":rcyield -sect=301 -nm=block")
	if !strings.Contains(msg, "preview_id=") || !strings.Contains(msg, "各軸力で正負曲げ") {
		t.Fatal(msg)
	}
	yieldMessage(t, w, ":rcyield -apply")
	for i, y := range s.Yield {
		if !positiveYieldInputs(y * float64(1-2*(i%2))) {
			t.Fatal(fmt.Sprint(s.Yield))
		}
	}
}
