package st

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/yofu/dxf/drawing"
	"github.com/yofu/dxf/entity"
)

type dxfDimensionCommandWindow struct {
	ExModer
	frame *Frame
	dir   string
}

func (w *dxfDimensionCommandWindow) Frame() *Frame    { return w.frame }
func (w *dxfDimensionCommandWindow) Cwd() string      { return w.dir }
func (w *dxfDimensionCommandWindow) Recent() []string { return nil }
func (w *dxfDimensionCommandWindow) EPS() float64     { return 1e-6 }
func (w *dxfDimensionCommandWindow) History(string)   {}

func TestDXFDimensionedCommands(t *testing.T) {
	f := dxfExportFixture(t)
	f.AddKijun("Y2", []float64{-1, 2, 0}, []float64{5, 2, 0})
	w := &dxfDimensionCommandWindow{frame: f, dir: t.TempDir()}
	for _, spec := range []struct{ name, args string }{
		{"plan", "-floor=2"}, {"jiku", "-name=Y2"}, {"crosssection", "-axis=1 -min=-0.01 -max=0.01"},
	} {
		path := filepath.Join(w.dir, spec.name+".dxf")
		if err := exCommand(w, fmt.Sprintf(":%s %s %s -dim", spec.name, path, spec.args), nil, nil); err != nil {
			t.Fatal(err)
		}
		d := readExportedDXF(t, path)
		for _, e := range d.Entities() {
			if c, ok := e.(*entity.Circle); ok && len(c.Direction) > 0 {
				if math.Abs(c.Direction[0])+math.Abs(c.Direction[1]) > 1e-9 || c.Direction[2] != 1 {
					t.Fatal("section circle is outside the drawing XY plane")
				}
			}
		}
		if len(dimensionMeasurements(t, d, "DIM_SPAN")) == 0 {
			t.Fatal("command did not enable dimensions")
		}
	}
}

func dimensionMeasurements(t *testing.T, d *drawing.Drawing, layer string) []float64 {
	t.Helper()
	var values []float64
	for _, e := range d.EntitiesOnLayer(layer) {
		dim, ok := e.(*entity.Dimension)
		if !ok {
			t.Fatalf("non-dimension on %s", layer)
		}
		if dim.Measurement <= 0 || math.IsNaN(dim.Measurement) {
			t.Fatalf("bad dimension %+v", dim)
		}
		block, err := d.Block(dim.BlockName)
		if err != nil || len(block.Entities) == 0 {
			t.Fatal("dimension has no graphics", err)
		}
		values = append(values, math.Round(dim.Measurement*1e6)/1e6)
	}
	sort.Float64s(values)
	return values
}

func TestDXFDimensionedPlanAndElevation(t *testing.T) {
	for _, axis := range []int{2, 1, 0} {
		f := dxfExportFixture(t)
		f.Kijuns = make(map[string]*Kijun)
		f.AddKijun("X1", []float64{0, -1, 0}, []float64{0, 3, 0})
		f.AddKijun("X2", []float64{4, -1, 0}, []float64{4, 3, 0})
		f.AddKijun("Y1", []float64{-1, 0, 0}, []float64{5, 0, 0})
		f.AddKijun("Y2", []float64{-1, 2, 0}, []float64{5, 2, 0})
		before := make(map[int][]float64)
		for id, e := range f.Elems {
			before[id] = append(append([]float64(nil), e.Strong...), e.Weak...)
		}
		path := filepath.Join(t.TempDir(), "dimensioned.dxf")
		var err error
		if axis == 2 {
			err = f.WriteDxfPlanDimensions(path, 2, 1000, 150, 200)
		} else {
			err = f.WriteDxfCrosssectionDimensions(path, axis, -.01, .01, 1, 1000, 150, 200)
		}
		if err != nil {
			t.Fatal(err)
		}
		for id, e := range f.Elems {
			if !reflect.DeepEqual(before[id], append(append([]float64(nil), e.Strong...), e.Weak...)) {
				t.Fatal("drawing changed member axes")
			}
		}
		d := readExportedDXF(t, path)
		wantSpan, wantTotal := []float64{2000, 4000}, []float64{2000, 4000}
		if axis != 2 {
			span := 4000.
			if axis == 0 {
				span = 2000
			}
			wantSpan, wantTotal = []float64{span}, []float64{span, 6000}
			if got := dimensionMeasurements(t, d, "DIM_HEIGHT"); !reflect.DeepEqual(got, []float64{3000, 3000}) {
				t.Fatalf("story heights %v", got)
			}
		}
		if got := dimensionMeasurements(t, d, "DIM_SPAN"); !reflect.DeepEqual(got, wantSpan) {
			t.Fatalf("axis%d spans=%v want%v", axis, got, wantSpan)
		}
		if got := dimensionMeasurements(t, d, "DIM_TOTAL"); !reflect.DeepEqual(got, wantTotal) {
			t.Fatalf("axis%d totals=%v want%v", axis, got, wantTotal)
		}
		labels := map[string]bool{}
		for _, e := range d.Entities() {
			if text, ok := e.(*entity.Text); ok {
				labels[text.Value] = true
			}
		}
		if !labels["柱Ｃ１・日本語"] || !labels["梁Ｇ１"] || !labels["X1"] && axis != 0 || !labels["Y1"] && axis != 1 {
			t.Fatalf("missing grid/member labels %v", labels)
		}
	}
}

func TestDXFDimensionedSidecarAndInvalidInput(t *testing.T) {
	f := dxfExportFixture(t)
	f.Path = filepath.Join(t.TempDir(), "model.inp")
	f.Kijuns = map[string]*Kijun{}
	kjn := "X-A 0 -1 0 0 3 0\nX-B 4 -1 0 4 3 0\nY-A -1 0 0 5 0 0\nY-B -1 2 0 5 2 0\n2FL -1 2 3 5 2 3\nRFL -1 2 6 5 2 6\n"
	if err := os.WriteFile(Ce(f.Path, ".kjn"), []byte(kjn), 0600); err != nil {
		t.Fatal(err)
	}
	r, err := f.dimensionFrame()
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Kijuns) != 0 || len(r.Kijuns) != 6 {
		t.Fatal("sidecar mutated the source")
	}
	axis, coord, err := r.resolveDxfJiku("Y-A")
	if err != nil || axis != 1 || coord != 0 {
		t.Fatal(axis, coord, err)
	}
	// Endpoint order and reversed member order must not select the wrong view.
	k := r.Kijuns["y-a"]
	k.Start, k.End = k.End, k.Start
	axis, coord, err = r.resolveDxfJiku("Y-A")
	if err != nil || axis != 1 || coord != 0 {
		t.Fatal(axis, coord, err)
	}
	r.AddKijun("slant", []float64{0, 0, 0}, []float64{1, 1, 0})
	if _, _, err = r.resolveDxfJiku("slant"); err == nil {
		t.Fatal("slanted view silently flattened")
	}
	for _, scale := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if err = f.WriteDxfPlanDimensions(filepath.Join(t.TempDir(), "bad.dxf"), 2, scale, 150, 200); err == nil {
			t.Fatal("invalid scale accepted")
		}
	}
}

func TestDXFDimensionedRejectsOutsideReversedMember(t *testing.T) {
	f := dxfExportFixture(t)
	s := NewSect()
	s.Num, s.Name, s.Frame = 401, "OUTSIDE", f
	f.Sects[401] = s
	// Reversed endpoint order used to admit this member below the requested
	// story: first Z=3 meets the lower bound while last Z=0 meets the upper.
	a, b := f.AddNode(1, 1, 3), f.AddNode(1, 1, 0)
	f.AddLineElem(-1, []*Node{a, b}, s, COLUMN)
	path := filepath.Join(t.TempDir(), "filtered.dxf")
	if err := f.WriteDxfPlanDimensions(path, 2, 1000, 150, 200); err != nil {
		t.Fatal(err)
	}
	d := readExportedDXF(t, path)
	for _, e := range d.Entities() {
		if text, ok := e.(*entity.Text); ok && text.Value == "OUTSIDE" {
			t.Fatal("member outside the story leaked into drawing")
		}
	}
}

func TestDXFDimensionedNakano(t *testing.T) {
	input := os.Getenv("ST_DXF_TEST_INP")
	if input == "" {
		t.Skip("set ST_DXF_TEST_INP to nakano02_07.inp and its LST/KJN sidecars")
	}
	data, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	f := NewFrame()
	if err = f.ReadInpSnapshot(data, input); err != nil {
		t.Fatal(err)
	}
	if err = f.ReadLst(Ce(input, ".lst")); err != nil {
		t.Fatal(err)
	}
	r, err := f.dimensionFrame()
	if err != nil {
		t.Fatal(err)
	}
	dir := os.Getenv("ST_DXF_TEST_OUTPUT")
	if dir == "" {
		dir = t.TempDir()
	}
	if err = os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"plan", "Y2", "X1"} {
		path := filepath.Join(dir, "dimensioned_"+name+".dxf")
		if name == "plan" {
			err = f.WriteDxfPlanDimensions(path, 3, 1000, 150, 200)
		} else {
			axis, c, e := r.resolveDxfJiku(name)
			if e != nil {
				t.Fatal(e)
			}
			err = f.WriteDxfCrosssectionDimensions(path, axis, c-.1, c+.1, 1, 1000, 150, 200)
		}
		if err != nil {
			t.Fatal(err)
		}
		d := readExportedDXF(t, path)
		spans := dimensionMeasurements(t, d, "DIM_SPAN")
		if name != "X1" {
			want := []float64{2720, 4100, 5795}
			if name == "plan" {
				want = []float64{2720, 4000, 4100, 5795}
			}
			if !reflect.DeepEqual(spans, want) {
				t.Fatalf("%s spans %v", name, spans)
			}
		} else if !reflect.DeepEqual(spans, []float64{4000}) {
			t.Fatalf("X1 spans %v", spans)
		}
		if name != "plan" {
			heights := dimensionMeasurements(t, d, "DIM_HEIGHT")
			if !reflect.DeepEqual(heights, []float64{2300, 2650, 2650, 2650, 2650, 2650, 2750, 2750, 2850}) {
				t.Fatalf("%s floor heights %v", name, heights)
			}
		}
		t.Logf("%s: spans=%v totals=%v output=%s", name, spans, dimensionMeasurements(t, d, "DIM_TOTAL"), path)
	}
}
