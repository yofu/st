package st

import (
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/yofu/dxf"
	"github.com/yofu/dxf/drawing"
	"github.com/yofu/dxf/entity"
)

func dxfExportFixture(t *testing.T) *Frame {
	t.Helper()
	f := NewFrame()
	f.Ai.Nfloor, f.Ai.Boundary = 4, []float64{-.1, 1.5, 4.5, 7}
	rc := NewRCColumn(101)
	rc.CShape = CRect{-15, -15, 15, 15}
	rc.SetName("柱Ｃ１・日本語")
	for _, p := range [][]float64{{-10, -10}, {10, -10}, {10, 10}, {-10, 10}} {
		rc.Reins = append(rc.Reins, Reinforce{Area: 2.865, Position: p})
	}
	steel := NewSGirder(201, HKYOU{H: 40, B: 20, Tw: .8, Tf: 1.2}, Steel{})
	steel.SetName("梁Ｇ１")
	pipe := NewSColumn(301, CPIPE{D: 30, T: 1}, Steel{})
	pipe.SetName("鋼管柱Ｐ１")
	for id, allow := range map[int]SectionRate{101: rc, 201: steel, 301: pipe} {
		s := NewSect()
		s.Num, s.Frame, s.Name, s.Allow = id, f, allow.Name(), allow
		f.Sects[id], f.Show.Sect[id] = s, true
	}
	var nodes []*Node
	for _, z := range []float64{0, 3, 6} {
		for _, c := range [][]float64{{0, 0}, {4, 0}, {0, 2}, {4, 2}} {
			n := f.AddNode(c[0], c[1], z)
			n.Frame = f
			nodes = append(nodes, n)
			if z == 0 {
				for i := range n.Conf {
					n.Conf[i] = true
				}
			}
			n.Disp["L"] = []float64{.001 * z, -.002 * z, -.001, 0, 0, 0}
		}
	}
	for floor := 0; floor < 2; floor++ {
		for i := 0; i < 4; i++ {
			sect := f.Sects[101]
			if i == 3 {
				sect = f.Sects[301]
			}
			f.AddLineElem(-1, []*Node{nodes[4*floor+i], nodes[4*(floor+1)+i]}, sect, COLUMN)
		}
	}
	for _, pair := range [][2]int{{4, 5}, {6, 7}, {4, 6}, {5, 7}} {
		e := f.AddLineElem(-1, []*Node{nodes[pair[0]], nodes[pair[1]]}, f.Sects[201], GIRDER)
		e.Bonds[4], e.Bonds[5] = Pin, Pin
	}
	f.AddPlateElem(-1, []*Node{nodes[4], nodes[5], nodes[7], nodes[6]}, f.Sects[201], SLAB)
	f.AddPlateElem(-1, []*Node{nodes[0], nodes[1], nodes[5]}, f.Sects[201], WALL)
	if _, err := f.AddKijun("Ｘ１通り", []float64{-1, 0, 0}, []float64{5, 0, 0}); err != nil {
		t.Fatal(err)
	}
	f.Show.PlotState |= PLOT_DEFORMED
	f.Show.Period, f.Show.Dfact = "L", 100
	return f
}

func readExportedDXF(t *testing.T, path string) *drawing.Drawing {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	handles := map[string]bool{}
	for i := 0; i+1 < len(lines); i += 2 {
		code, value := strings.TrimSpace(lines[i]), strings.TrimSpace(lines[i+1])
		if (code == "5" || code == "105") && value != "" && value != "0" {
			if handles[value] {
				t.Fatalf("duplicate object/table handle %s", value)
			}
			handles[value] = true
		}
	}
	d, err := dxf.FromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Entities()) == 0 {
		t.Fatal("empty DXF")
	}
	if problems := dxf.Audit(d, false); len(problems) != 0 {
		t.Fatalf("DXF audit: %v", problems)
	}
	// A second write/read also checks ownership, handles and reference survival.
	copyPath := filepath.Join(t.TempDir(), "roundtrip.dxf")
	if err := d.SaveAs(copyPath); err != nil {
		t.Fatal(err)
	}
	r, err := dxf.FromFile(copyPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Entities()) != len(d.Entities()) {
		t.Fatal("round trip lost entities")
	}
	for i, e := range d.Entities() {
		round := r.Entities()[i]
		if reflect.TypeOf(e) != reflect.TypeOf(round) || e.Layer().Name() != round.Layer().Name() {
			t.Fatal("round trip changed entity type or layer")
		}
		if text, ok := e.(*entity.Text); ok && text.Value != round.(*entity.Text).Value {
			t.Fatal("round trip changed Japanese text")
		}
	}
	for name, g := range d.Groups {
		if other := r.Groups[name]; other == nil || len(other.Entities()) != len(g.Entities()) {
			t.Fatalf("round trip changed group %s", name)
		}
	}
	if problems := dxf.Audit(r, false); len(problems) != 0 {
		t.Fatalf("round-trip audit: %v", problems)
	}
	return d
}

func sameDXFPoint(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if math.Abs(a[i]-b[i]) > 1e-6 {
			return false
		}
	}
	return true
}

func assertDXFPoint(t *testing.T, got, want []float64) {
	t.Helper()
	if len(got) < len(want) {
		t.Fatalf("point %v want %v", got, want)
	}
	for i, v := range want {
		if math.Abs(got[i]-v) > 1e-6 {
			t.Fatalf("point %v want %v", got, want)
		}
	}
}

func fixtureDXFPath(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	if output := os.Getenv("ST_DXF_TEST_OUTPUT"); output != "" {
		dir = filepath.Join(output, "fixtures")
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(dir, name+".dxf")
}

func TestDXFPlanAndElevationCompatibility(t *testing.T) {
	for _, tc := range []struct {
		name string
		axis int
	}{{"plan", 2}, {"elevationX", 1}, {"elevationY", 0}} {
		t.Run(tc.name, func(t *testing.T) {
			f := dxfExportFixture(t)
			axes := make(map[int][]float64)
			for id, e := range f.Elems {
				axes[id] = append(append([]float64(nil), e.Strong...), e.Weak...)
			}
			path := fixtureDXFPath(t, tc.name)
			var err error
			if tc.axis == 2 {
				err = f.WriteDxfPlan(path, 2, 1000, 150, 200)
			} else {
				err = f.WriteDxfCrosssection(path, tc.axis, -.1, .1, 1, 1000, 150, 200)
			}
			if err != nil {
				t.Fatal(err)
			}
			d := readExportedDXF(t, path)
			labels := make(map[string]bool)
			for _, e := range d.Entities() {
				if text, ok := e.(*entity.Text); ok {
					labels[text.Value] = true
					if text.Height != 150 {
						t.Fatalf("text height %g", text.Height)
					}
				}
			}
			if !labels["柱Ｃ１・日本語"] || !labels["梁Ｇ１"] {
				t.Fatalf("missing Japanese section labels: %v", labels)
			}
			if tc.axis == 2 && (!labels["鋼管柱Ｐ１"] || !labels["Ｘ１通り"]) {
				t.Fatalf("missing pipe/grid label: %v", labels)
			}
			if len(d.EntitiesOnLayer("HARI")) == 0 || len(d.EntitiesOnLayer("HASHIRA")) == 0 {
				t.Fatal("missing member layers")
			}
			if _, err := d.LineType("MCDASH6"); err != nil {
				t.Fatal(err)
			}
			for id, e := range f.Elems {
				if !reflect.DeepEqual(axes[id], append(append([]float64(nil), e.Strong...), e.Weak...)) {
					t.Fatalf("export changed member %d axes", id)
				}
			}
			if tc.axis == 2 {
				// Three RC columns have 4 main bars each, the pipe has 2 contours.
				circles := 0
				for _, e := range d.EntitiesOnLayer("HASHIRA") {
					if c, ok := e.(*entity.Circle); ok {
						circles++
						if c.Radius <= 0 {
							t.Fatal("invalid section radius")
						}
						if c.Radius == 150 || c.Radius == 140 {
							assertDXFPoint(t, c.Center, []float64{4000, 2000, 0})
						}
					}
				}
				if circles != 14 {
					t.Fatalf("section circles=%d want 14 (3 RC columns x 4 bars + pipe inner/outer)", circles)
				}
			}
		})
	}
}

func TestDXF3DCompatibility(t *testing.T) {
	f := dxfExportFixture(t)
	path := fixtureDXFPath(t, "model3d")
	if err := f.WriteDxf3D(path, 1000); err != nil {
		t.Fatal(err)
	}
	d := readExportedDXF(t, path)
	lines := 0
	for _, e := range f.Elems {
		if !e.IsLineElem() {
			continue
		}
		lines++
		layer := ETYPES[e.Etype] + strconv.Itoa(e.Sect.Num)
		found := false
		for _, drawn := range d.EntitiesOnLayer(layer) {
			if l, ok := drawn.(*entity.Line); ok {
				wantStart, wantEnd := make([]float64, 3), make([]float64, 3)
				for i := 0; i < 3; i++ {
					wantStart[i], wantEnd[i] = e.Enod[0].Coord[i]*1000, e.Enod[1].Coord[i]*1000
				}
				if reflect.DeepEqual(l.Start, wantStart) && reflect.DeepEqual(l.End, wantEnd) {
					found = true
				}
			}
		}
		if !found {
			t.Fatalf("missing member %d geometry", e.Num)
		}
		wantStart, wantEnd := make([]float64, 3), make([]float64, 3)
		for i := 0; i < 3; i++ {
			wantStart[i] = (e.Enod[0].Coord[i] + 100*e.Enod[0].Disp["L"][i]) * 1000
			wantEnd[i] = (e.Enod[1].Coord[i] + 100*e.Enod[1].Disp["L"][i]) * 1000
		}
		found = false
		for _, drawn := range d.EntitiesOnLayer("DEFORMATION") {
			l := drawn.(*entity.Line)
			if sameDXFPoint(l.Start, wantStart) && sameDXFPoint(l.End, wantEnd) {
				found = true
			}
		}
		if !found {
			t.Fatalf("incorrect deformed coordinates for member %d", e.Num)
		}
	}
	deformed := d.EntitiesOnLayer("DEFORMATION")
	if len(deformed) != lines {
		t.Fatalf("deformed lines %d want %d", len(deformed), lines)
	}
	if d.Layers["DEFORMATION"].LineType.Name() != "HIDDEN" {
		t.Fatal("deformation line type lost")
	}
	if len(d.EntitiesOnLayer("CONF111111")) != 4 {
		t.Fatal("support points lost")
	}
	if g := d.Groups["PINRIGID"]; g == nil || len(g.Entities()) != 4 {
		t.Fatal("pin group memberships lost")
	}
	faces := 0
	for _, e := range d.Entities() {
		if _, ok := e.(*entity.ThreeDFace); ok {
			faces++
		}
	}
	if faces != 2 {
		t.Fatalf("3DFACE count %d", faces)
	}
	if len(d.EntitiesOnLayer("SECTION")) == 0 {
		t.Fatal("section figures lost")
	}
}

// Optional real-model regression. Outputs are retained only when requested;
// input and LST files are read without generating loads or saving the model.
func TestDXFRealModelCompatibility(t *testing.T) {
	input := os.Getenv("ST_DXF_TEST_INP")
	if input == "" {
		t.Skip("set ST_DXF_TEST_INP to an existing INP/LST pair")
	}
	data, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	f := NewFrame()
	if err = f.ReadInpSnapshot(data, input); err != nil {
		t.Fatal(err)
	}
	lst := os.Getenv("ST_DXF_TEST_LST")
	if lst == "" {
		lst = Ce(input, ".lst")
	}
	if err = f.ReadLst(lst); err != nil {
		t.Fatal(err)
	}
	dir := os.Getenv("ST_DXF_TEST_OUTPUT")
	if dir == "" {
		dir = t.TempDir()
	} else if err = os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if len(f.Ai.Boundary) < 2 {
		t.Fatal("model has no floor boundaries")
	}
	// Pick the most populated Y plane for a real elevation rather than
	// flattening every parallel frame into the same view.
	yCounts := make(map[float64]int)
	y, maxCount := 0., 0
	for _, n := range f.Nodes {
		yCounts[n.Coord[1]]++
	}
	for coord, count := range yCounts {
		if count > maxCount || count == maxCount && coord < y {
			y, maxCount = coord, count
		}
	}
	for _, tc := range []struct {
		name  string
		write func(string) error
	}{
		{"plan", func(p string) error { return f.WriteDxfPlan(p, 1, 1000, 150, 200) }},
		{"elevation", func(p string) error { return f.WriteDxfCrosssection(p, 1, y-.01, y+.01, 1, 1000, 150, 200) }},
		{"model3d", func(p string) error { return f.WriteDxf3D(p, 1000) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, tc.name+".dxf")
			if err := tc.write(path); err != nil {
				t.Fatal(err)
			}
			d := readExportedDXF(t, path)
			if tc.name == "model3d" {
				lines, faces := 0, 0
				for _, e := range d.Entities() {
					if l, ok := e.(*entity.Line); ok && strings.HasPrefix(l.Layer().Name(), "SECTION") == false && l.Layer().Name() != "DEFORMATION" {
						lines++
					}
					if _, ok := e.(*entity.ThreeDFace); ok {
						faces++
					}
				}
				wantLines, wantFaces := 0, 0
				for _, e := range f.Elems {
					if e.IsLineElem() {
						wantLines++
					} else if e.Enods == 3 || e.Enods == 4 {
						wantFaces++
					}
				}
				if lines != wantLines || faces != wantFaces {
					t.Fatalf("members %d/%d faces %d/%d", lines, wantLines, faces, wantFaces)
				}
			}
			t.Logf("%d entities: %s", len(d.Entities()), path)
		})
	}
}
