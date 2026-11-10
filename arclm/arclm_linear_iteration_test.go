package arclm

import (
	"io"
	"path/filepath"
	"testing"
)

// iterationBeam returns n equal members along X restricted to the XZ plane.
// No node is supported; the tests add their own supports.
func iterationBeam(n int, span float64) *Frame {
	f := NewFrame()
	f.Output = io.Discard
	f.Lapch, f.Pivot = nil, nil
	s := NewSect()
	s.Num, s.E, s.Poi = 1, 1000, .3
	s.Value = []float64{1, 1, 1, 1}
	f.Sects = []*Sect{s}
	for i := 0; i <= n; i++ {
		v := NewNode()
		v.Index, v.Num = i, i+1
		v.Coord[0] = span * float64(i) / float64(n)
		v.Conf[1], v.Conf[3], v.Conf[5] = true, true, true
		f.Nodes = append(f.Nodes, v)
	}
	for i := 0; i < n; i++ {
		e := NewElem()
		e.Num, e.Sect = i+1, s
		e.Enod = []*Node{f.Nodes[i], f.Nodes[i+1]}
		f.Elems = append(f.Elems, e)
	}
	return f
}

func TestFloorContactHoldsNodeBelowFloor(t *testing.T) {
	// The contact node starts 2 mm below the floor elevation. It is held
	// where it is, like the supports beside it, instead of being lifted.
	f := iterationBeam(2, 4)
	f.Nodes[0].Conf[0], f.Nodes[0].Conf[2], f.Nodes[2].Conf[2] = true, true, true
	f.Nodes[1].Force[2] = -1
	c := newtonTestCondition(t)
	c.SetFloorContact(.002)
	if err := f.StaticAnalysis(nil, c); err != nil {
		t.Fatal(err)
	}
	nearNewton(t, "contact node displacement", f.Nodes[1].Disp[2], 0, 0)
	nearNewton(t, "contact reaction", f.Nodes[1].Reaction[2], 1, 1e-12)
	nearNewton(t, "left support reaction", f.Nodes[0].Reaction[2], 0, 1e-12)
	nearNewton(t, "right support reaction", f.Nodes[2].Reaction[2], 0, 1e-12)
	if f.Nodes[1].Conf[2] {
		t.Fatal("contact support was left in the frame")
	}
}

func TestFloorContactOnlySupports(t *testing.T) {
	// Every vertical support is a contact. The node pulled upward leaves the
	// floor and the two others carry the load.
	f := iterationBeam(2, 4)
	f.Nodes[0].Conf[0] = true
	f.Nodes[1].Force[2], f.Nodes[2].Force[2] = -2, .5
	c := newtonTestCondition(t)
	c.SetFloorContact(0)
	if err := f.StaticAnalysis(nil, c); err != nil {
		t.Fatal(err)
	}
	nearNewton(t, "left contact reaction", f.Nodes[0].Reaction[2], .5, 1e-12)
	nearNewton(t, "middle contact reaction", f.Nodes[1].Reaction[2], 1, 1e-12)
	nearNewton(t, "lifted node reaction", f.Nodes[2].Reaction[2], 0, 0)
	if f.Nodes[2].Disp[2] <= 0 {
		t.Fatalf("lifted node displacement: %g", f.Nodes[2].Disp[2])
	}
	for _, n := range f.Nodes {
		if n.Conf[2] {
			t.Fatalf("NODE %d: contact support was left in the frame", n.Num)
		}
	}
}

// upliftBeam has supports at x = 0, 3 and 6 and an overhang to x = 9.
// With all three supports the middle one is in tension (-1.5).
func upliftBeam() *Frame {
	f := iterationBeam(3, 9)
	f.Nodes[0].Conf[0] = true
	for i := 0; i < 3; i++ {
		f.Nodes[i].Conf[2] = true
	}
	f.Nodes[0].Force[2], f.Nodes[3].Force[2] = -1, -1
	return f
}

func TestArclm401ReleasesTensileSupport(t *testing.T) {
	f := upliftBeam()
	f.Lapch = make(chan int)
	laps := make(chan int)
	go func() {
		count := 0
		for range f.Lapch {
			count++
			f.Lapch <- 1
		}
		laps <- count
	}()
	otp := filepath.Join(t.TempDir(), "uplift.otp")
	if err := f.Arclm401(otp, true, 1e-3, nil); err != nil {
		t.Fatal(err)
	}
	close(f.Lapch)
	if count := <-laps; count != 2 {
		t.Fatalf("LAP notifications: got %d want 2", count)
	}
	nearNewton(t, "left reaction", f.Nodes[0].Reaction[2], .5, 1e-12)
	nearNewton(t, "released reaction", f.Nodes[1].Reaction[2], 0, 0)
	nearNewton(t, "right reaction", f.Nodes[2].Reaction[2], 1.5, 1e-12)
	if f.Nodes[1].Disp[2] <= 0 {
		t.Fatalf("released support displacement: %g", f.Nodes[1].Disp[2])
	}
	want := []bool{false, true, true, true, false, true}
	for j, fixed := range f.Nodes[1].Conf {
		if fixed != want[j] {
			t.Fatalf("released support was not restored: %v", f.Nodes[1].Conf)
		}
	}
}

func TestArclm401ExtraWeightKeepsSupport(t *testing.T) {
	f := upliftBeam()
	otp := filepath.Join(t.TempDir(), "uplift.otp")
	if err := f.Arclm401(otp, true, 1e-3, map[int]float64{f.Nodes[1].Num: 2}); err != nil {
		t.Fatal(err)
	}
	nearNewton(t, "left reaction", f.Nodes[0].Reaction[2], 1.25, 1e-12)
	nearNewton(t, "middle reaction", f.Nodes[1].Reaction[2], -1.5, 1e-12)
	nearNewton(t, "right reaction", f.Nodes[2].Reaction[2], 2.25, 1e-12)
}

func TestArclm401UnstableAfterRelease(t *testing.T) {
	// Without the load at the left end both remaining supports cannot hold
	// the overhang, and the analysis must stop instead of iterating forever.
	f := upliftBeam()
	f.Nodes[0].Force[2] = 0
	if err := f.Arclm401(filepath.Join(t.TempDir(), "uplift.otp"), true, 1e-3, nil); err == nil {
		t.Fatal("expected an error for a support set that cannot carry the load")
	}
	if !f.Nodes[0].Conf[2] || !f.Nodes[1].Conf[2] || !f.Nodes[2].Conf[2] {
		t.Fatal("supports were not restored after the failure")
	}
}

func TestArclm301SoilSpringSecant(t *testing.T) {
	// Axial soil spring: area A/(3.16 sqrt(y)) for an elongation y [cm] > 0.1.
	// With d = 3.16 P L sqrt(100 d)/(E A) the fixed point is
	// 100 (3.16 P L/E A)^2.
	f := iterationBeam(1, 1)
	for j := range f.Nodes[0].Conf {
		f.Nodes[0].Conf[j] = true
		f.Nodes[1].Conf[j] = j != 0
	}
	f.Nodes[1].Force[0] = 5
	if err := f.Arclm301(filepath.Join(t.TempDir(), "soil.otp"), true, []int{1}, 1e-13); err != nil {
		t.Fatal(err)
	}
	nearNewton(t, "soil spring elongation", f.Nodes[1].Disp[0], 100*3.16*3.16*.005*.005, 1e-11)
	nearNewton(t, "soil spring force", f.Elems[0].Stress[6], 5, 1e-11)
	nearNewton(t, "support reaction", f.Nodes[0].Reaction[0], -5, 1e-11)
	nearNewton(t, "section area is unchanged", f.Sects[0].Value[0], 1, 0)
}

func TestArclm301MatchesStaticAnalysisWithoutSoil(t *testing.T) {
	// A member load on a beam with a spring at one end and a pin at the
	// other: both analyses use the same connection condensation and reactions.
	build := func() *Frame {
		f := iterationBeam(2, 6)
		for j := range f.Nodes[0].Conf {
			f.Nodes[0].Conf[j], f.Nodes[2].Conf[j] = true, true
		}
		e := f.Elems[0]
		e.Bonds[4] = &Sect{Num: 5, Value: []float64{0, 300, 300, 0}, Yield: make([]float64, 12), Type: -1}
		f.Elems[1].Bonds[10] = &Sect{Num: 1, Value: []float64{0, 0, 0, 0}, Yield: make([]float64, 12), Type: -1}
		e.Cmq[2], e.Cmq[8], e.Cmq[4], e.Cmq[10] = .9, 2.1, -.6, .9
		f.Nodes[1].Force[2] = -1
		return f
	}
	static, soil := build(), build()
	if err := static.StaticAnalysis(nil, newtonTestCondition(t)); err != nil {
		t.Fatal(err)
	}
	if err := soil.Arclm301(filepath.Join(t.TempDir(), "soil.otp"), true, nil, 1e-12); err != nil {
		t.Fatal(err)
	}
	for i, e := range static.Elems {
		for j, v := range e.Stress {
			nearNewton(t, "member force", soil.Elems[i].Stress[j], v, 1e-12)
		}
	}
	total := 0.0
	for i, n := range static.Nodes {
		for j, v := range n.Reaction {
			nearNewton(t, "reaction", soil.Nodes[i].Reaction[j], v, 1e-12)
		}
		total += soil.Nodes[i].Reaction[2]
	}
	nearNewton(t, "vertical equilibrium", total, 1+.9+2.1, 1e-12)
}
