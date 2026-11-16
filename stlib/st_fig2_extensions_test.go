package st

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fig2ExtensionFixture() *Frame {
	f := NewFrame()
	s := NewSect()
	s.Num = 201
	f.Sects[201] = s
	for i, p := range [][]float64{{0, 0, 0}, {0, 0, 2}, {3, 0, 2}} {
		n := NewNode()
		n.Frame = f
		n.Num = 100 + i
		n.Coord = p
		f.Nodes[n.Num] = n
	}
	col := NewLineElem([]*Node{f.Nodes[100], f.Nodes[101]}, s, COLUMN)
	col.Frame = f
	col.Num = 10
	col.Bonds[4] = Pin
	col.Stress["L"] = map[int][]float64{100: {3, 0, 0, 0, 0, 0}, 101: {-3, 0, 0, 0, 0, 0}}
	beam := NewLineElem([]*Node{f.Nodes[101], f.Nodes[102]}, s, GIRDER)
	beam.Frame = f
	beam.Num = 20
	beam.Stress["L"] = map[int][]float64{101: {4, 0, 0, 0, 0, 0}, 102: {-4, 0, 0, 0, 0, 0}}
	f.Elems[10], f.Elems[20] = col, beam
	return f
}

func TestFig2ExpressionsOnModel(t *testing.T) {
	f := fig2ExtensionFixture()
	f.Path = "drawing.inp"
	name := filepath.Join(t.TempDir(), "test.fig2")
	src := `COMMON
EVAL height @.nodes[101].coord[2] ** 2
PAGE A4TATE
TITLE H=$height
EF x.num in (10, 20) and x.length() >= 2
NF 0 <= x.coord[2] < 3
EV Length "L={:.2f}" x.length()
NV Level "{:.1f}" x.coord[2] if x.num != 100 else 0
RF abs(x.N('L'))
WELDING L
`
	if err := os.WriteFile(name, []byte(src), 0600); err != nil {
		t.Fatal(err)
	}
	sheets, err := loadFig2Sheets(name, make(map[string]bool), f)
	if err != nil {
		t.Fatal(err)
	}
	w := &fig2PDFWindow{frame: f, pdf: &PDFCanvas{height: 297, width: 210}, boxes: make(map[string]*TextBox)}
	if err := applyFig2View(w, name, sheets[0].Views[0].Commands); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(w.TextBox("TITLE").Text(), " "); got != "H=4" {
		t.Fatalf("title=%s", got)
	}
	if got := f.Show.fig2ElemValues[10]; got != "L=2.00\n" {
		t.Fatalf("EV=%q", got)
	}
	if got := f.Show.fig2NodeValues[100]; got != "0.0\n" {
		t.Fatalf("NV=%q", got)
	}
	if got := f.Show.fig2RainbowValues[20]; got != 4 {
		t.Fatalf("RF=%g", got)
	}
	if got := f.Show.fig2WeldValues[101]; math.Abs(got-5) > 1e-9 {
		t.Fatalf("WELDING=%g, want 5", got)
	}
	if err := applyFig2Expression(f, "nf", []string{"x.num", "==", "101"}, false); err != nil {
		t.Fatal(err)
	}
	if !f.Nodes[100].IsHidden(f.Show) || f.Nodes[101].IsHidden(f.Show) || !f.Nodes[102].IsHidden(f.Show) {
		t.Fatal("NF did not retain node 101 only")
	}
	expr, err := compileFig2Expr("open('file')")
	if err == nil {
		_, err = fig2EvalValue(expr, f, nil)
	}
	if err == nil {
		t.Fatal("unexpected unrestricted file access")
	}
}

func TestFig2JointRatesFromPythonReport(t *testing.T) {
	f := fig2ExtensionFixture()
	path := filepath.Join(t.TempDir(), "model.ratj")
	f.Path = Ce(path, ".inp")
	data := `------------------------------------------------------------
ELEM 10 NODE 100 101 SECT 201
0 NODE 100 PERIOD  L  N= 3.000 Q= 0.000 rate= 0.20000
0 NODE 100 PERIOD L+X N= 4.000 Q= 0.000 rate= 0.35000
1 NODE 101 PERIOD L-Y N= 5.000 Q= 0.000 rate= 0.47000
============================================================
SECT 201 ELEM 10 rateL= 0.20000 rateS= 0.47000
`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	if err := setupFig2Joint(f, []string{"SHORT"}, false); err != nil {
		t.Fatal(err)
	}
	for mode, want := range map[string]float64{"LONG": .2, "SHORT": .47, "ULTIMATE": .47} {
		if got, ok := fig2JointRate(f.Elems[10], mode); !ok || got != want {
			t.Fatalf("%s: %v %v", mode, got, ok)
		}
	}
	if _, ok := fig2JointRate(f.Elems[20], "LONG"); ok {
		t.Fatal("unexpected joint rate for a rigid member")
	}
}

func TestFig2RotatedLabelPlacement(t *testing.T) {
	var placed []fig2LabelRect
	for _, angle := range []float64{0, 90, -35, 35, 90, 0} {
		r := fig2LabelBounds(100, 100, angle, -9, 4, 18, 6)
		dx, dy, ok := fig2PlaceLabel(r, 200, 200, placed)
		if !ok {
			t.Fatal("no label position")
		}
		r.x0 += dx
		r.x1 += dx
		r.y0 += dy
		r.y1 += dy
		for _, old := range placed {
			if r.overlaps(old) {
				t.Fatal("overlapping label")
			}
		}
		placed = append(placed, r)
	}
}

type fig2ResultCapture struct {
	Drawer
	labels []fig2StressLabel
	angle  float64
}

func (*fig2ResultCapture) Line(float64, float64, float64, float64) {}
func (*fig2ResultCapture) Foreground(int) int                      { return BLACK }
func (*fig2ResultCapture) TextAlignment(int)                       {}
func (d *fig2ResultCapture) TextOrientation(angle float64)         { d.angle = angle }
func (d *fig2ResultCapture) QueueFig2Stress(x, y float64, value string) bool {
	d.labels = append(d.labels, fig2StressLabel{x: x, y: y, angle: d.angle, text: value})
	return true
}

func TestFig2ResultsForProjectedPointAndFixedNode(t *testing.T) {
	frame := fig2ExtensionFixture()
	show := frame.Show
	show.Fig2Document = true
	show.Bond, show.Phinge, show.Conf = false, false, false
	column := frame.Elems[10]
	for _, n := range column.Enod {
		n.Pcoord = []float64{10, 20}
	}
	column.Stress["L"][column.Enod[0].Num][0] = math.Copysign(0, -1)
	show.Stress[COLUMN] = STRESS_NZ
	d := &fig2ResultCapture{}
	DrawElem(d, column, show)
	if len(d.labels) != 1 || d.labels[0].text != "-0.000" || d.labels[0].angle != 0 ||
		d.labels[0].x != 10 || d.labels[0].y != 20 {
		t.Fatalf("point projection must preserve the signed axial-force label: %#v", d.labels)
	}

	n := column.Enod[0]
	n.Conf[2] = true
	n.Disp["L"] = []float64{0, 0, math.Copysign(0, -1), 0, 0, 0}
	show.NodeCaption = NC_DZ
	show.Period = "L"
	d.labels = nil
	DrawNode(d, n, show)
	if len(d.labels) != 1 || d.labels[0].text != "-0.000" {
		t.Fatalf("Fig2 must include stored displacement at a fixed degree of freedom: %#v", d.labels)
	}
	show.Fig2Document = false
	d.labels = nil
	DrawNode(d, n, show)
	if len(d.labels) != 0 {
		t.Fatalf("ordinary node display should omit fixed-degree captions: %#v", d.labels)
	}
}

func TestFig2ReportsUnplaceableResultCoordinate(t *testing.T) {
	p := &PDFCanvas{fig2: true}
	if !p.QueueFig2Stress(math.NaN(), 10, "1.000") || p.layoutErr == nil {
		t.Fatal("a result with an invalid PDF position must give a layout error")
	}
}
