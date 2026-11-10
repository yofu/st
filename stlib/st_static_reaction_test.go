package st

import (
	"io"
	"math"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yofu/st/arclm"
)

func TestStaticGravityReactionIncludesCMQAndNodalLoad(t *testing.T) {
	f := yieldEndInputFrame()
	dir := t.TempDir()
	f.Path = filepath.Join(dir, "model.inp")
	f.Ai.Nfloor, f.Ai.Boundary = 2, []float64{-1, .5, 2}
	f.Ai.BetaX, f.Ai.BetaY = []float64{1}, []float64{1}
	s, err := f.ParseSect(strings.Fields(yieldEndTestSection), false)
	if err != nil {
		t.Fatal(err)
	}
	ground, a, b := NewNode(), NewNode(), NewNode()
	ground.Num, a.Num, b.Num = 1, 2, 3
	a.Coord[2], b.Coord[2], b.Coord[0] = 1, 1, 6
	for _, n := range []*Node{ground, a, b} {
		for j := range n.Conf {
			n.Conf[j] = true
		}
		f.Nodes[n.Num] = n
	}
	a.Load[2] = -2
	e := NewLineElem([]*Node{a, b}, s, GIRDER)
	e.Num, e.Frame, e.Cmq[2] = 1, f, 1.5
	f.Elems[e.Num] = e

	if err := f.ExtractArclm(filepath.Join(dir, "model.wgt")); err != nil {
		t.Fatal(err)
	}
	af := f.Arclms["L"]
	af.Lapch, af.Pivot, af.Output = nil, nil, io.Discard
	c := arclm.NewAnalysisCondition()
	c.SetOutput([]string{filepath.Join(dir, "model.otl")})
	if err := af.StaticAnalysis(nil, c); err != nil {
		t.Fatal(err)
	}
	f.ReadArclmData(af, "L")
	halfWeight := s.Weight()[1] * e.Length() / 2
	for _, tc := range []struct {
		node *Node
		want float64
	}{{a, halfWeight + 1.5 + 2}, {b, halfWeight}} {
		if got := tc.node.ReturnReaction("L", 2); math.Abs(got-tc.want) > 1e-12 {
			t.Errorf("node %d Rz=%g, want %g", tc.node.Num, got, tc.want)
		}
	}
	if e.Cmq[2] != 1.5 {
		t.Errorf("extraction changed input CMQ: %g", e.Cmq[2])
	}
}
