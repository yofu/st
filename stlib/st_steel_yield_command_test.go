package st

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/yofu/st/arclm"
)

type steelYieldCommandWindow struct {
	*dxfDimensionCommandWindow
	selected []*Elem
	changed  int
}

func (w *steelYieldCommandWindow) Changed(on bool) {
	if on {
		w.changed++
	}
}
func (w *steelYieldCommandWindow) UndoEnabled() bool      { return false }
func (w *steelYieldCommandWindow) ElemSelected() bool     { return len(w.selected) > 0 }
func (w *steelYieldCommandWindow) SelectedElems() []*Elem { return w.selected }

func yieldMessage(t *testing.T, w ExModer, cmd string) string {
	t.Helper()
	result := exCommand(w, cmd, nil, nil)
	msg, ok := result.(Message)
	if !ok {
		t.Fatalf("%s: %v", cmd, result)
	}
	return string(msg)
}

func TestSteelYieldCommandReview(t *testing.T) {
	f, el, _ := steelLengthFixture()
	fig := NewFig()
	fig.Value["AREA"] = 0.01
	el.Sect.Figs = []*Fig{fig}
	unselected := f.AddLineElem(-1, []*Node{f.AddNode(0, 2, 0), f.AddNode(8, 2, 0)}, el.Sect, GIRDER)
	w := &steelYieldCommandWindow{dxfDimensionCommandWindow: &dxfDimensionCommandWindow{frame: f}, selected: []*Elem{el}}
	for _, cmd := range []string{":steelyield -apply", ":steelyield -sect=511 -elem=1001", ":steelyield -sect=511 -lk=0", ":steelyield -ratio=NaN", ":steelyield -product=1,2"} {
		err := exCommand(w, cmd, nil, nil)
		if _, ok := err.(Message); err == nil || ok {
			t.Fatal("invalid request passed", cmd)
		}
	}
	conflict := yieldMessage(t, w, ":steelyield")
	for _, v := range []string{fmt.Sprintf("ELEM %d", unselected.Num), "案2", "NZMAX", "MYMIN", "未反映"} {
		if !strings.Contains(conflict, v) {
			t.Fatalf("missing %s in %s", v, conflict)
		}
	}
	blocked := exCommand(w, ":steelyield -apply", nil, nil)
	if blocked == nil || f.steelYieldReview != nil || w.changed != 0 {
		t.Fatalf("conflict accepted: %v", blocked)
	}
	cmd := ":steelyield -lk=4 -lb=4"
	msg := yieldMessage(t, w, cmd)
	if !strings.Contains(msg, "preview_id=") || w.changed != 0 || el.Sect.Yield[0] != 0 {
		t.Fatal(msg)
	}
	for _, name := range steelYieldNames {
		if !strings.Contains(msg, name+" ") {
			t.Error("missing capacity", name)
		}
	}
	f.Arclms["L"] = arclm.NewFrame()
	el.Sect.Yield[0] = 11 // manual change between review and confirmation
	err := exCommand(w, ":steelyield -apply", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "preview") || el.Sect.Yield[0] != 11 || w.changed != 0 || f.Arclms["L"] == nil {
		t.Fatal(err)
	}
	yieldMessage(t, w, cmd)
	w.selected = nil // confirmation retains the reviewed sections
	yieldMessage(t, w, ":steelyield -apply")
	if w.changed != 1 || !slices.Equal(el.Sect.Yield, nakanoHBeams[4].want[:]) || len(f.Arclms) != 0 {
		t.Fatal("applied limits", el.Sect.Yield)
	}
	yieldMessage(t, w, cmd+" -sect=511")
	yieldMessage(t, w, ":steelyield -apply")
	if w.changed != 1 {
		t.Fatal("no-op marked the model changed")
	}
}

func TestSteelYieldCommandNakanoRoundTrip(t *testing.T) {
	f := readNakanoSteelYieldModel(t)
	dir := t.TempDir()
	t.Chdir(dir)
	f.Path = filepath.Join(dir, "work.inp")
	w := &steelYieldCommandWindow{dxfDimensionCommandWindow: &dxfDimensionCommandWindow{frame: f}}
	type expected struct {
		y        [12]float64
		exp, exq float64
		ends     [2]bool
	}
	want := map[int]expected{}
	checkAndApply := func(ids []int, options string) {
		t.Helper()
		var labels []string
		for _, id := range ids {
			labels = append(labels, fmt.Sprint(id))
		}
		cmd := ":steelyield -sect=" + strings.Join(labels, ",") + " " + options
		msg := yieldMessage(t, w, cmd)
		if !strings.Contains(msg, "preview_id=") {
			t.Fatalf("blocked: %s", msg)
		}
		seen := 0
		for _, el := range f.Elems {
			if !slices.Contains(ids, el.Sect.Num) {
				continue
			}
			seen++
			if !strings.Contains(msg, fmt.Sprintf("ELEM %d  SECT %d ", el.Num, el.Sect.Num)) {
				t.Fatalf("missing ELEM %d", el.Num)
			}
		}
		yieldMessage(t, w, ":steelyield -apply")
		t.Logf("SECT %v: reviewed %d members", ids, seen)
	}
	// Change a real input condition so each section differs from the source.
	condition := cold01HBeamCondition()
	condition.StrengthFactor = 0.9
	var beams []int
	for _, tc := range nakanoHBeams {
		s := f.Sects[tc.num]
		raw, err := s.Allow.(*SGirder).HBeamYield(condition)
		if err != nil {
			t.Fatal(err)
		}
		y, err := steelYieldINPLimits(raw)
		if err != nil || slices.Equal(s.Yield, y[:]) {
			t.Fatal(tc.num, err)
		}
		want[tc.num] = expected{y, s.Exp, s.Exq, s.ElasticEnds}
		beams = append(beams, tc.num)
	}
	checkAndApply(beams, "-lk=4 -lb=4 -factor=0.9")
	for _, tc := range nakanoTubeColumns {
		s, p := f.Sects[tc.num], tc.product
		c := cold01TubeCondition()
		c.StrengthFactor = 0.9
		raw, err := s.Allow.(*SColumn).ColdFormedTubeYield(p, c)
		if err != nil {
			t.Fatal(err)
		}
		y, err := steelYieldINPLimits(raw)
		if err != nil || slices.Equal(s.Yield, y[:]) {
			t.Fatal(tc.num, err)
		}
		want[tc.num] = expected{y, s.Exp, s.Exq, s.ElasticEnds}
		checkAndApply([]int{tc.num}, fmt.Sprintf("-lk=2.85 -lb=2.85 -factor=0.9 -product=%g,%g,%g,%g,%g,%g,%g,%g,%g", p.Shape.H, p.Shape.B, p.Shape.Tw, p.Shape.Tf, p.Area, p.Ix, p.Iy, p.Zpx, p.Zpy))
	}
	path := filepath.Join(dir, "reviewed.inp")
	if err := f.WriteInp(path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	reopened := NewFrame()
	if err := reopened.ReadInpSnapshot(data, path); err != nil {
		t.Fatal(err)
	}
	for _, frame := range []*Frame{f, reopened} {
		for id, expected := range want {
			if !slices.Equal(frame.Sects[id].Yield, expected.y[:]) {
				t.Fatalf("SECT %d got %v want %v", id, frame.Sects[id].Yield, expected.y)
			}
		}
		if err := frame.ExtractArclm(filepath.Join(dir, "weights.wgt")); err != nil {
			t.Fatal(err)
		}
		for _, p := range []string{"L", "X", "Y"} {
			seen := 0
			for _, s := range frame.Arclms[p].Sects {
				if e, ok := want[s.Num]; ok {
					seen++
					if !slices.Equal(s.Yield, e.y[:]) || s.Exp != e.exp || s.Exq != e.exq || s.ElasticEnds != e.ends {
						t.Fatalf("%s arclm SECT %d: %+v", p, s.Num, s)
					}
				}
			}
			if seen != len(want) {
				t.Fatalf("%s: %d of %d sections", p, seen, len(want))
			}
		}
	}
}
