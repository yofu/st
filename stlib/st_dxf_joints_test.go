package st

import (
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/yofu/dxf/entity"
)

func jointFixture() *Frame {
	f := NewFrame()
	for _, id := range []int{101, 201} {
		a := NewRCGirder(id)
		a.CShape = CRect{-10, -10, 10, 10}
		s := NewSect()
		s.Num = id
		s.Frame = f
		s.Allow = a
		s.Name = "G1"
		f.Sects[id] = s
		f.Show.Sect[id] = true
	}
	return f
}
func jointNode(f *Frame, x, y float64) *Node { n := f.AddNode(x, y, 0); n.Frame = f; return n }
func jointBeam(f *Frame, a, b *Node, id int) *Elem {
	return f.AddLineElem(-1, []*Node{a, b}, f.Sects[id], GIRDER)
}
func jointStrokes(f *Frame) []dxfMemberStroke {
	s := f.prepareDXFStrokes(2, -.1, .1, 0, 1)
	f.jointDXFStrokes(s, 2, .1, 0, 1, .03)
	return s
}
func jointArea(p []dxfXY) float64 {
	a := 0.
	for i, v := range p {
		a += xyCross(v, p[(i+1)%len(p)])
	}
	return a * .5
}
func jointBounds(ps [][]dxfXY) (lo, hi dxfXY) {
	lo, hi = dxfXY{math.Inf(1), math.Inf(1)}, dxfXY{math.Inf(-1), math.Inf(-1)}
	for _, p := range ps {
		for _, v := range p {
			for j := 0; j < 2; j++ {
				lo[j] = math.Min(lo[j], v[j])
				hi[j] = math.Max(hi[j], v[j])
			}
		}
	}
	return
}
func jointNear(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-8 {
		t.Fatalf("got %.12g want %.12g", got, want)
	}
}

func TestDXFJointContinuousAndReleases(t *testing.T) {
	for _, tc := range []struct {
		name string
		bond int
		want int
		gap  float64
	}{
		{"rigid", 0, 1, 0}, {"pin", 1, 2, .03}, {"spring", 2, 2, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := jointFixture()
			a, b, c := jointNode(f, 0, 0), jointNode(f, 2, 0), jointNode(f, 4, 0)
			e := jointBeam(f, a, b, 101)
			jointBeam(f, c, b, 101) // reversed input orientation
			if tc.bond > 0 {
				e.Bonds[10] = &Bond{Num: tc.bond}
			}
			before := append([]float64(nil), e.Strong...)
			nodes := append([]*Node(nil), e.Enod...)
			ss := jointStrokes(f)
			if len(ss) != tc.want {
				t.Fatalf("strokes %d want %d", len(ss), tc.want)
			}
			if tc.want == 1 {
				if len(ss[0].nodes) != 3 {
					t.Fatal("lost intermediate joint")
				}
				jointNear(t, ss[0].length, 4)
			} else {
				_, hi := jointBounds(dxfStrokePieces(ss[0]))
				jointNear(t, hi[0], 2-tc.gap)
			}
			if !reflect.DeepEqual(e.Strong, before) || !reflect.DeepEqual(e.Enod, nodes) || len(f.Elems) != 2 {
				t.Fatal("drawing edited model")
			}
		})
	}
	t.Run("different section", func(t *testing.T) {
		f := jointFixture()
		a, b, c := jointNode(f, 0, 0), jointNode(f, 2, 0), jointNode(f, 4, 0)
		jointBeam(f, a, b, 101)
		jointBeam(f, b, c, 201)
		s := jointStrokes(f)
		if len(s) != 2 || len(dxfStrokePieces(s[0])) != 1 || len(dxfStrokePieces(s[1])) != 1 {
			t.Fatal("different sections must retain butt ends")
		}
	})
	t.Run("ambiguous overlap", func(t *testing.T) {
		f := jointFixture()
		a, b, c, d := jointNode(f, 0, 0), jointNode(f, 2, 0), jointNode(f, 4, 0), jointNode(f, 5, 0)
		jointBeam(f, a, b, 101)
		jointBeam(f, b, c, 101)
		jointBeam(f, b, d, 101)
		if len(f.prepareDXFStrokes(2, -.1, .1, 0, 1)) != 3 {
			t.Fatal("ambiguous duplicated members joined")
		}
	})
}

func TestDXFJointTAndCross(t *testing.T) {
	for _, pin := range []bool{false, true} {
		t.Run(map[bool]string{false: "rigid", true: "pin"}[pin], func(t *testing.T) {
			f := jointFixture()
			a, b, c, d := jointNode(f, 0, 0), jointNode(f, 2, 0), jointNode(f, 4, 0), jointNode(f, 2, 2)
			jointBeam(f, a, b, 101)
			jointBeam(f, b, c, 101)
			e := jointBeam(f, b, d, 201)
			if pin {
				e.Bonds[4] = &Bond{Num: 1}
			}
			s := jointStrokes(f)
			if len(s) != 2 {
				t.Fatal("main beam not joined", len(s))
			}
			main := dxfStrokePieces(s[0])
			branch := dxfStrokePieces(s[1])
			if len(main) != 1 || len(branch) != 1 {
				t.Fatal("missing bodies")
			}
			_, mh := jointBounds(main)
			jointNear(t, mh[0], 4)
			low, _ := jointBounds(branch)
			want := .1
			if pin {
				want += .03
			}
			jointNear(t, low[1], want)
			jointNear(t, math.Abs(jointArea(branch[0])), .2*(2-want))
		})
	}
	t.Run("cross", func(t *testing.T) {
		f := jointFixture()
		c := jointNode(f, 0, 0)
		jointBeam(f, jointNode(f, -2, 0), c, 101)
		jointBeam(f, c, jointNode(f, 2, 0), 101)
		jointBeam(f, jointNode(f, 0, -2), c, 201)
		jointBeam(f, c, jointNode(f, 0, 2), 201)
		s := jointStrokes(f)
		if len(s) != 2 {
			t.Fatal("cross chains", len(s))
		}
		p := dxfStrokePieces(s[1])
		if len(p) != 2 {
			t.Fatal("through beam must split around host", len(p))
		}
		area := 0.
		for _, q := range p {
			area += math.Abs(jointArea(q))
		}
		jointNear(t, area, .2*(4-.2))
	})
	t.Run("projected crossing is unconnected", func(t *testing.T) {
		f := jointFixture()
		jointBeam(f, jointNode(f, -2, 0), jointNode(f, 2, 0), 101)
		jointBeam(f, jointNode(f, 0, -2), jointNode(f, 0, 2), 201)
		s := jointStrokes(f)
		for _, v := range s {
			p := dxfStrokePieces(v)
			if len(p) != 1 {
				t.Fatal("trimmed unconnected crossing")
			}
			jointNear(t, math.Abs(jointArea(p[0])), .8)
		}
	})
}

func TestDXFJointColumnFaces(t *testing.T) {
	for _, angle := range []float64{0, math.Pi / 4} {
		for _, pin := range []bool{false, true} {
			t.Run(fmtJointCase(angle, pin), func(t *testing.T) {
				f := jointFixture()
				rc := NewRCColumn(301)
				rc.CShape = CRect{-15, -15, 15, 15}
				s := NewSect()
				s.Num = 301
				s.Allow = rc
				s.Frame = f
				f.Sects[301] = s
				a := jointNode(f, 0, 0)
				b := f.AddNode(0, 0, 3)
				b.Frame = f
				col := f.AddLineElem(-1, []*Node{a, b}, s, COLUMN)
				col.Cang = angle
				col.SetPrincipalAxis()
				e := jointBeam(f, a, jointNode(f, 2, 0), 101)
				if pin {
					e.Bonds[4] = &Bond{Num: 1}
				}
				ss := f.prepareDXFStrokes(2, -.1, 1.5, 0, 1)
				f.jointDXFStrokes(ss, 2, 1.5, 0, 1, .03)
				pieces := dxfStrokePieces(ss[0])
				if len(pieces) != 1 {
					t.Fatal("missing beam", len(pieces))
				}
				p := pieces[0]
				lo, _ := jointBounds(pieces)
				if angle == 0 {
					want := .15
					if pin {
						want += .03
					}
					jointNear(t, lo[0], want)
				} else if pin {
					jointNear(t, lo[0], .15*math.Sqrt2+.03)
				} else {
					tip := false
					for _, v := range p {
						if math.Abs(v[0]-.15*math.Sqrt2) < 1e-8 && math.Abs(v[1]) < 1e-8 {
							tip = true
						}
					}
					if !tip {
						t.Fatal("rotated face was replaced by an overlapping straight end cap", p)
					}
				}
				// For the square rotated 45 degrees, its right boundary is x+y=r
				// above y=0 and x-y=r below; every beam boundary vertex must clear it.
				for _, v := range p {
					boundary := .15
					if angle != 0 {
						boundary = .15*math.Sqrt2 - math.Abs(v[1])
					}
					clearance := 0.
					if pin {
						clearance = .03
					}
					if v[0] < boundary+clearance-1e-8 {
						t.Fatal("overlap/gap violation", v, boundary)
					}
				}
			})
		}
	}
}
func fmtJointCase(a float64, p bool) string {
	if a == 0 {
		if p {
			return "square pin"
		}
		return "square rigid"
	}
	if p {
		return "rotated pin"
	}
	return "rotated rigid"
}

func TestDXFJointRigidMiterAndShortPin(t *testing.T) {
	f := jointFixture()
	a, b, c := jointNode(f, 0, 0), jointNode(f, 2, 0), jointNode(f, 2, 2)
	jointBeam(f, a, b, 101)
	jointBeam(f, b, c, 101)
	s := jointStrokes(f)
	if len(s) != 2 {
		t.Fatal("bend should not be merged")
	}
	for _, p := range s[0].ends[1] {
		found := false
		for _, q := range s[1].ends[0] {
			if xyLength(xySub(p, q)) < 1e-8 {
				found = true
			}
		}
		if !found {
			t.Fatal("miter ends do not coincide", s)
		}
	}
	f = jointFixture()
	e := jointBeam(f, jointNode(f, 0, 0), jointNode(f, .04, 0), 101)
	e.Bonds[4], e.Bonds[10] = Pin, Pin
	if len(dxfStrokePieces(jointStrokes(f)[0])) != 0 {
		t.Fatal("short pin member inverted")
	}
}

func TestDXFJointPolygonDifference(t *testing.T) {
	subject := []dxfXY{{0, 0}, {4, 0}, {4, 2}, {0, 2}}
	cases := []struct {
		name   string
		clips  [][]dxfXY
		area   float64
		pieces int
	}{
		{"none", nil, 8, 1},
		{"touch", [][]dxfXY{{{-1, 0}, {0, 0}, {0, 2}, {-1, 2}}}, 8, 1},
		{"split", [][]dxfXY{{{1, -1}, {2, -1}, {2, 3}, {1, 3}}}, 6, 2},
		{"overlapping clips", [][]dxfXY{{{1, -1}, {2.5, -1}, {2.5, 3}, {1, 3}}, {{2, -1}, {3, -1}, {3, 3}, {2, 3}}}, 4, 2},
		{"same clips", [][]dxfXY{{{1, -1}, {2, -1}, {2, 3}, {1, 3}}, {{1, -1}, {2, -1}, {2, 3}, {1, 3}}}, 6, 2},
		{"contained", [][]dxfXY{{{1, .5}, {2, .5}, {2, 1.5}, {1, 1.5}}}, 7, 2},
		{"covered", [][]dxfXY{{{-1, -1}, {5, -1}, {5, 3}, {-1, 3}}}, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ps := dxfPolygonRemainder(subject, tc.clips)
			if len(ps) != tc.pieces {
				t.Fatalf("pieces %d want %d: %v", len(ps), tc.pieces, ps)
			}
			area := 0.
			for _, p := range ps {
				area += jointArea(p)
			}
			jointNear(t, area, tc.area)
		})
	}
}

func TestDXFJointExport(t *testing.T) {
	f := jointFixture()
	a, b, c := jointNode(f, 0, 0), jointNode(f, 2, 0), jointNode(f, 4, 0)
	jointBeam(f, a, b, 101)
	jointBeam(f, c, b, 101)
	d, err := f.buildDxfCrosssection(2, -.1, .1, 1, 1000, 150, 100, false)
	if err != nil {
		t.Fatal(err)
	}
	lines, labels := 0, 0
	for _, e := range d.Entities() {
		switch e.(type) {
		case *entity.LwPolyline:
			lines++
		case *entity.Text:
			labels++
		}
	}
	if lines != 1 || labels != 1 {
		t.Fatalf("merged drawing: %d outlines %d labels", lines, labels)
	}
	path := filepath.Join(t.TempDir(), "joints.dxf")
	if dir := os.Getenv("ST_DXF_TEST_OUTPUT"); dir != "" {
		path = filepath.Join(dir, "joint_continuous.dxf")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := d.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	readExportedDXF(t, path)
}

func TestDXFJointNakano(t *testing.T) {
	input := os.Getenv("ST_DXF_TEST_INP")
	if input == "" {
		t.Skip("set ST_DXF_TEST_INP for real joint geometry")
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
	f, err = f.dimensionFrame()
	if err != nil {
		t.Fatal(err)
	}
	views := []struct {
		axis     int
		min, max float64
		x, y     int
	}{{2, f.Ai.Boundary[2], f.Ai.Boundary[3], 0, 1}, {1, -.1, .1, 0, 2}, {0, -.1, .1, 1, 2}}
	totalMerged := 0
	for _, v := range views {
		raw := 0
		for _, e := range f.Elems {
			if !e.IsLineElem() || e.Etype == WBRACE || e.Etype == SBRACE {
				continue
			}
			a, b := e.Enod[0].Coord, e.Enod[1].Coord
			if a[v.axis] >= v.min && a[v.axis] <= v.max && b[v.axis] >= v.min && b[v.axis] <= v.max && math.Hypot(a[v.x]-b[v.x], a[v.y]-b[v.y]) > dxfJointTol {
				raw++
			}
		}
		strokes := f.prepareDXFStrokes(v.axis, v.min, v.max, v.x, v.y)
		totalMerged += raw - len(strokes)
		f.jointDXFStrokes(strokes, v.axis, v.max, v.x, v.y, .03)
		pieces, trimmed := 0, 0
		for _, s := range strokes {
			ps := dxfStrokePieces(s)
			pieces += len(ps)
			if len(s.clips) > 0 {
				trimmed++
			}
			for _, p := range ps {
				for i, a := range p {
					b := p[(i+1)%len(p)]
					for _, tval := range []float64{.25, .5, .75} {
						sample := xyAdd(a, xyMul(xySub(b, a), tval))
						for _, clip := range s.clips {
							if dxfPointLocation(sample, clip) == 1 {
								t.Fatalf("member %d boundary crosses host face: %v", s.id, sample)
							}
						}
					}
				}
			}
		}
		t.Logf("axis %d: %d source members -> %d strokes, %d visible pieces, %d trimmed", v.axis, raw, len(strokes), pieces, trimmed)
	}
	if totalMerged == 0 {
		t.Fatal("real model did not exercise aggregation")
	}
}

func TestDXFJointObliqueAndCenterline(t *testing.T) {
	for _, width := range []bool{true, false} {
		f := jointFixture()
		c := jointNode(f, 0, 0)
		host := jointBeam(f, jointNode(f, 0, -2), c, 101)
		host.Etype = COLUMN
		h2 := jointBeam(f, c, jointNode(f, 0, 2), 101)
		h2.Etype = COLUMN
		e := jointBeam(f, c, jointNode(f, 2, 2), 201)
		if !width {
			e.Sect.Allow = nil
		}
		ss := jointStrokes(f)
		if len(ss) != 2 {
			t.Fatal("strokes", len(ss))
		}
		p := dxfStrokePieces(ss[1])
		if len(p) != 1 {
			t.Fatal("oblique pieces", p)
		}
		// The two rails of an oblique beam meet the same vertical column face,
		// at different distances from its origin.
		count := 0
		for _, v := range p[0] {
			if v[0] < .1-1e-8 {
				t.Fatal("overlapping oblique beam", v)
			}
			if math.Abs(v[0]-.1) < 1e-8 {
				count++
			}
		}
		want := 2
		if !width {
			want = 1
		}
		if count != want {
			t.Fatal("not trimmed to column face", p)
		}
	}
}
