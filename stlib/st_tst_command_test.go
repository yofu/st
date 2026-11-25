package st

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type tstTestWindow struct {
	ExModer
	frame     *Frame
	cwd       string
	selected  []*Elem
	history   []string
	opened    []string
	reported  []string
	lastExCmd string
}

func newTstTestWindow(f *Frame, dir string) *tstTestWindow {
	return &tstTestWindow{
		frame: f,
		cwd:   dir,
	}
}

func (w *tstTestWindow) Frame() *Frame                 { return w.frame }
func (w *tstTestWindow) EPS() float64                  { return 1e-4 }
func (w *tstTestWindow) Cwd() string                   { return w.cwd }
func (w *tstTestWindow) LastExCommand() string         { return w.lastExCmd }
func (w *tstTestWindow) SetLastExCommand(s string)     { w.lastExCmd = s }
func (w *tstTestWindow) ElemSelected() bool            { return len(w.selected) > 0 }
func (w *tstTestWindow) SelectedElems() []*Elem        { return w.selected }
func (w *tstTestWindow) SelectElem(els []*Elem)        { w.selected = els }
func (w *tstTestWindow) Deselect()                     { w.selected = nil }
func (w *tstTestWindow) History(s string)              { w.history = append(w.history, s) }
func (w *tstTestWindow) SkipExFilenameCompletion()     {}
func (w *tstTestWindow) UseCurrentModelInputs() bool   { return true }
func (w *tstTestWindow) FileWritten(path string) { w.reported = append(w.reported, path) }
func (w *tstTestWindow) OpenBrowserArtifact(action string, paths ...string) bool {
	if action == "text" {
		w.opened = append(w.opened, paths...)
		return true
	}
	return false
}

func newTstTestFrame(t *testing.T) (*Frame, *Elem, *Elem, *Elem) {
	t.Helper()
	dir := t.TempDir()
	f := NewFrame()
	f.Path = filepath.Join(dir, "model.inp")
	f.Show.Unit = []float64{SI, 1.0}
	f.Show.UnitName = []string{"kN", "m"}

	p := &Prop{Num: 101, Name: "SN400", eL: 2.05e7, eS: 2.05e7, poi: 0.3, hiju: 7.85, EFactor: 1.0, HFactor: 1.0}
	f.Props[101] = p

	n0 := f.AddNode(0, 0, 0)
	n1 := f.AddNode(0, 0, 4)
	n2 := f.AddNode(6, 0, 4)
	n3 := f.AddNode(6, 0, 0)

	sh, err := NewHKYOU([]string{"30", "15", "0.65", "0.9"})
	if err != nil {
		t.Fatal(err)
	}
	sc := NewSColumn(201, sh, SN400)
	sc.SetName("C1")
	sectCol := f.AddSect(201)
	sectCol.Name = "C1"
	sectCol.Allow = sc
	figCol := NewFig()
	figCol.Prop = p
	figCol.Value["AREA"] = sh.A() * 1e-4
	figCol.Value["IXX"] = sh.Ix() * 1e-8
	figCol.Value["IYY"] = sh.Iy() * 1e-8
	figCol.Value["VEN"] = sh.J() * 1e-8
	sectCol.Figs = []*Fig{figCol}

	sg := NewSGirder(202, sh, SN400)
	sg.SetName("G1")
	sectGir := f.AddSect(202)
	sectGir.Name = "G1"
	sectGir.Allow = sg
	figGir := NewFig()
	figGir.Prop = p
	figGir.Value["AREA"] = sh.A() * 1e-4
	figGir.Value["IXX"] = sh.Ix() * 1e-8
	figGir.Value["IYY"] = sh.Iy() * 1e-8
	figGir.Value["VEN"] = sh.J() * 1e-8
	sectGir.Figs = []*Fig{figGir}

	// Wall panel with wood shear wall section rate and analysis braces.
	pWood := &Prop{Num: 303, Name: "GOHAN", eL: 4.5e5, eS: 4.5e5, poi: 0.3, EFactor: 1.0, HFactor: 1.0}
	f.Props[303] = pWood
	sectWall := f.AddSect(801)
	sectWall.Name = "W1"
	figWall := NewFig()
	figWall.Prop = pWood
	figWall.Value["KFACT"] = 2.5
	sectWall.Figs = []*Fig{figWall}
	ww := NewWoodWall(801)
	ww.Wood = GOHAN
	ww.Kfact = 2.5
	ww.SetName("W1")
	sectWall.Allow = ww

	col := f.AddLineElem(101, []*Node{n0, n1}, sectCol, COLUMN)
	gir := f.AddLineElem(102, []*Node{n1, n2}, sectGir, GIRDER)
	wall := f.AddPlateElem(201, []*Node{n0, n3, n2, n1}, sectWall, WALL)
	f.AddAnalysisBraces()

	// Populate stresses for L, X, Y.
	col.Stress["L"] = map[int][]float64{
		n0.Num: {10.0, 0.5, 1.0, 0.0, 2.0, 0.3},
		n1.Num: {-10.0, -0.5, -1.0, 0.0, -2.0, -0.3},
	}
	col.Stress["X"] = map[int][]float64{
		n0.Num: {5.0, 0.2, 4.0, 0.0, 8.0, 0.5},
		n1.Num: {-5.0, -0.2, -4.0, 0.0, -8.0, -0.5},
	}
	col.Stress["Y"] = map[int][]float64{
		n0.Num: {2.0, 1.0, 0.5, 0.0, 1.0, 2.0},
		n1.Num: {-2.0, -1.0, -0.5, 0.0, -1.0, -2.0},
	}

	gir.Stress["L"] = map[int][]float64{
		n1.Num: {0.5, 0.0, 3.0, 0.0, -4.5, 0.0},
		n2.Num: {-0.5, 0.0, 3.0, 0.0, 4.5, 0.0},
	}
	gir.Stress["X"] = map[int][]float64{
		n1.Num: {1.0, 0.0, 2.0, 0.0, -5.0, 0.0},
		n2.Num: {-1.0, 0.0, -2.0, 0.0, -5.0, 0.0},
	}
	gir.Stress["Y"] = map[int][]float64{
		n1.Num: {0.1, 0.0, 0.2, 0.0, -0.3, 0.0},
		n2.Num: {-0.1, 0.0, -0.2, 0.0, -0.3, 0.0},
	}

	for i, child := range wall.Children {
		if child == nil {
			continue
		}
		sign := 1.0
		if i == 1 {
			sign = -1.0
		}
		child.Stress["L"] = map[int][]float64{
			child.Enod[0].Num: {0.2 * sign, 0, 0, 0, 0, 0},
			child.Enod[1].Num: {-0.2 * sign, 0, 0, 0, 0, 0},
		}
		child.Stress["X"] = map[int][]float64{
			child.Enod[0].Num: {1.5 * sign, 0, 0, 0, 0, 0},
			child.Enod[1].Num: {-1.5 * sign, 0, 0, 0, 0, 0},
		}
		child.Stress["Y"] = map[int][]float64{
			child.Enod[0].Num: {0.1 * sign, 0, 0, 0, 0, 0},
			child.Enod[1].Num: {-0.1 * sign, 0, 0, 0, 0, 0},
		}
	}

	return f, col, gir, wall
}

func TestTstCompactFullAndNonMutating(t *testing.T) {
	f, col, gir, _ := newTstTestFrame(t)
	w := newTstTestWindow(f, filepath.Dir(f.Path))
	w.SelectElem([]*Elem{col})

	// Run :tst before :srcal -> must succeed using default Condition without mutating col.
	if err := ExMode(w, ":tst"); err != nil {
		t.Fatal(err)
	}
	if col.Condition != nil || col.MaxRate != nil || col.SrcalTex != "" {
		t.Fatalf(":tst mutated element state: cond=%v maxRate=%v tex=%q", col.Condition, col.MaxRate, col.SrcalTex)
	}
	if len(w.history) == 0 {
		t.Fatal("expected :tst output in history")
	}
	compact := w.history[len(w.history)-1]
	t.Logf("compact output:\n%s", compact)
	if !strings.Contains(compact, "ELEM 101") || !strings.Contains(compact, "MAX:") || !strings.Contains(compact, "長期(L") {
		t.Fatalf("unexpected compact output:\n%s", compact)
	}

	// Run :srcal with custom options, then confirm :tst inherits Condition and -full matches .tst.
	if err := ExMode(w, ":srcal -qfact=1.5 -torsion"); err != nil {
		t.Fatal(err)
	}
	savedCond := col.Condition.Snapshot()
	if !savedCond.RCTorsion || savedCond.Qfact != 1.5 {
		t.Fatalf("Condition.Snapshot lost RCTorsion or Qfact: %+v", savedCond)
	}
	savedRate := append([]float64(nil), col.MaxRate...)
	savedTex := col.SrcalTex

	w.history = nil
	if err := ExMode(w, ":tst -v"); err != nil {
		t.Fatal(err)
	}
	verboseOut := w.history[len(w.history)-1]
	if !strings.Contains(verboseOut, "# 断面性能詳細") || !strings.Contains(verboseOut, "# 算定詳細") {
		t.Fatalf("expected verbose details in :tst -v output:\n%s", verboseOut)
	}
	if col.Condition.Verbose || col.Condition.Qfact != 1.5 || !col.Condition.RCTorsion {
		t.Fatalf(":tst -v mutated element Condition: %+v", col.Condition)
	}
	for i := range savedRate {
		if col.MaxRate[i] != savedRate[i] {
			t.Fatalf(":tst -v mutated MaxRate[%d]: %g vs %g", i, col.MaxRate[i], savedRate[i])
		}
	}
	if col.SrcalTex != savedTex {
		t.Fatal(":tst -v mutated SrcalTex")
	}

	// Verify :tst -full matches OutputRateInformation byte-for-byte.
	w.SelectElem([]*Elem{gir})
	w.history = nil
	if err := ExMode(w, ":tst -full"); err != nil {
		t.Fatal(err)
	}
	fullOut := w.history[len(w.history)-1]
	_, wantFull, err := gir.OutputRateInformation("L", "X", "X", "Y", "Y", -1.0, nil, 0.0)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimRight(fullOut, "\n") != strings.TrimRight(wantFull, "\n") {
		t.Fatalf(":tst -full does not match OutputRateInformation\ngot:\n%s\nwant:\n%s", fullOut, wantFull)
	}
}

func TestTstPlateExpansionPipeMaxAndFileOutput(t *testing.T) {
	f, col, gir, wall := newTstTestFrame(t)
	w := newTstTestWindow(f, filepath.Dir(f.Path))

	// Selecting a WALL panel expands to its WBRACE children.
	w.SelectElem([]*Elem{wall})
	if err := ExMode(w, ":tst"); err != nil {
		t.Fatal(err)
	}
	out := w.history[len(w.history)-1]
	for _, child := range wall.Children {
		if child != nil && !strings.Contains(out, fmt.Sprintf("ELEM %d ", child.Num)) {
			t.Fatalf("missing child ELEM %d in plate :tst output:\n%s", child.Num, out)
		}
	}
	if !strings.Contains(out, "SECT 801(W1)/") {
		t.Fatalf("expected original/derived section label for WBRACE:\n%s", out)
	}

	// Pipe from :elem sect 202 | :tst works and -max reduces to 1 element.
	w.Deselect()
	w.history = nil
	if err := ExMode(w, ":elem sect 202 | :tst"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(w.history[len(w.history)-1], "ELEM 102") {
		t.Fatalf("expected ELEM 102 from pipe, got:\n%s", w.history[len(w.history)-1])
	}

	// Explicit IDs + -max + -out + -open.
	w.Deselect()
	w.history = nil
	outFile := filepath.Join(filepath.Dir(f.Path), "excerpt.tst")
	if err := ExMode(w, ":tst 101 102 -max -out="+outFile+" -open"); err != nil {
		t.Fatal(err)
	}
	if len(w.opened) != 1 || w.opened[0] != outFile {
		t.Fatalf("expected OpenBrowserArtifact with %s, got %v", outFile, w.opened)
	}
	if len(w.reported) != 1 || w.reported[0] != outFile {
		t.Fatalf("expected ReportCommandFile with %s, got %v", outFile, w.reported)
	}
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "部材:") || !strings.Contains(string(data), "各断面種別の入力情報の確認") {
		t.Fatalf("unexpected excerpt .tst file content:\n%s", string(data))
	}

	// Limit truncation and unknown option error.
	w.SelectElem([]*Elem{col, gir})
	w.history = nil
	if err := ExMode(w, ":tst -limit=1"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(w.history[len(w.history)-1], "showing 1 of 2 elements") {
		t.Fatalf("expected limit notice, got:\n%s", w.history[len(w.history)-1])
	}
	if err := ExMode(w, ":tst -unknownflag"); err == nil {
		t.Fatal("expected error on unknown option")
	}
	w.Deselect()
	if err := ExMode(w, ":tst"); err == nil {
		t.Fatal("expected error when no elements are selected")
	}
}

func TestTstNoStressNoticeAndSkipWithoutSectionRate(t *testing.T) {
	f, col, _, _ := newTstTestFrame(t)
	w := newTstTestWindow(f, filepath.Dir(f.Path))
	col.Stress = make(map[string]map[int][]float64)

	// Add an element without SectionRate (curtain member).
	sectNoRate := f.AddSect(999)
	sectNoRate.Name = "NO_RATE"
	noRate := f.AddLineElem(199, []*Node{col.Enod[0], col.Enod[1]}, sectNoRate, GIRDER)

	w.SelectElem([]*Elem{col, noRate})
	if err := ExMode(w, ":tst"); err != nil {
		t.Fatal(err)
	}
	out := w.history[len(w.history)-1]
	if !strings.Contains(out, "[注意: 解析結果未読込]") {
		t.Fatalf("expected no-stress notice, got:\n%s", out)
	}
	if !strings.Contains(out, "Skipped 1 element(s) without section rate: 199") {
		t.Fatalf("expected skipped element note, got:\n%s", out)
	}
}
