package st

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/yofu/st/arclm"
)

const yieldEndTestSection = `SECT 221 SNAME test
 NFIG 1 FIG 1 FPROP 1 AREA 1 IXX 1 IYY 1 VEN 1
 EXP 2 EXQ 2
 NZMAX 100 NZMIN -100 QXMAX 100 QXMIN -100 QYMAX 100 QYMIN -100
 MZMAX 100 MZMIN -100 MXMAX 100 MXMIN -100 MYMAX 100 MYMIN -100
 COLOR 255 255 255`

func yieldEndInputFrame() *Frame {
	f := NewFrame()
	f.Props[1] = &Prop{Num: 1, eL: 100, eS: 100, hiju: 1, poi: .3}
	return f
}

func TestYieldEndInputRoundTripAndSnapshot(t *testing.T) {
	for _, option := range []string{"", "YIELDEND 1 1", "YIELDEND 1 0", "YIELDEND 0 1", "YIELDEND 0 0"} {
		f := yieldEndInputFrame()
		want := [2]bool{}
		if option != "" {
			want, _ = arclm.ParseYieldEnds(strings.Fields(option)[1:])
		}
		s, err := f.ParseSect(strings.Fields(yieldEndTestSection+"\n"+option), false)
		if err != nil || s.ElasticEnds != want {
			t.Fatalf("parse %q: %v %v", option, s, err)
		}
		text := s.InpString()
		if strings.Contains(text, "YIELDEND") != (want != [2]bool{}) {
			t.Fatalf("unexpected default serialization: %s", text)
		}
		f2 := yieldEndInputFrame()
		got, err := f2.ParseSect(strings.Fields(text), false)
		if err != nil || got.ElasticEnds != want || got.Yield[10] != 100 || got.Figs[0].Value["IYY"] != 1 {
			t.Fatalf("round trip: %v %v", got, err)
		}
		copy := s.Snapshot(f)
		if copy.ElasticEnds != want {
			t.Fatal("snapshot lost end selection")
		}
		copy.ElasticEnds[0] = !copy.ElasticEnds[0]
		if s.ElasticEnds != want {
			t.Fatal("snapshot shares end selection")
		}
	}
	for _, option := range []string{"YIELDEND", "YIELDEND 1", "YIELDEND 2 0", "YIELDEND 1 x", "YIELDEND 1 COLOR 255 255 255"} {
		f := yieldEndInputFrame()
		if _, err := f.ParseSect(strings.Fields(yieldEndTestSection+"\n"+option), false); err == nil {
			t.Fatalf("accepted invalid option %q", option)
		}
		if f.Sects[221] != nil {
			t.Fatal("invalid input partially inserted a section")
		}
	}
}

func TestYieldEndExtractArclm(t *testing.T) {
	f := yieldEndInputFrame()
	f.Path = filepath.Join(t.TempDir(), "model.inp")
	f.Ai.Nfloor, f.Ai.Boundary = 2, []float64{-1, .5, 2}
	f.Ai.BetaX, f.Ai.BetaY = []float64{1}, []float64{1}
	s, err := f.ParseSect(strings.Fields(yieldEndTestSection+"\nYIELDEND 1 0"), false)
	if err != nil {
		t.Fatal(err)
	}
	a, b := NewNode(), NewNode()
	a.Num, b.Num, b.Coord[2] = 1, 2, 1
	f.Nodes[1], f.Nodes[2] = a, b
	e := NewLineElem([]*Node{a, b}, s, COLUMN)
	e.Num, e.Frame = 1, f
	f.Elems[1] = e
	if err := f.ExtractArclm(filepath.Join(filepath.Dir(f.Path), "model.wgt")); err != nil {
		t.Fatal(err)
	}
	for _, period := range []string{"L", "X", "Y"} {
		af := f.Arclms[period]
		if af == nil || len(af.Elems) != 1 {
			t.Fatalf("missing extracted frame %s", period)
		}
		ae := af.Elems[0]
		if ae.Sect.ElasticEnds != [2]bool{false, true} || ae.Enod[0].Num != a.Num || ae.Enod[1].Num != b.Num {
			t.Fatalf("lost end selection/order in %s: %v", period, ae)
		}
	}
}
