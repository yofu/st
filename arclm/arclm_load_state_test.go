package arclm

import (
	"bytes"
	"io"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func preloadPatternFrame() *Frame {
	f := plasticTestFrame()
	f.Lapch, f.Pivot = nil, nil
	for _, node := range f.Nodes {
		for j := 0; j < 6; j++ {
			node.Force[j] = 0
		}
	}
	for j := 0; j < 3; j++ {
		f.Nodes[1].Conf[j] = false
	}
	for j := 0; j < 6; j++ {
		f.Elems[0].Sect.Yield[2*j], f.Elems[0].Sect.Yield[2*j+1] = 1000, -1000
	}
	return f
}

func runLinearPreload(t *testing.T, f *Frame) {
	t.Helper()
	f.Output = io.Discard
	f.Pivot = make(chan int, 6*len(f.Nodes))
	f.Lapch = make(chan int)
	done := make(chan struct{})
	defer close(done)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-f.Lapch:
				select {
				case f.Lapch <- 0:
				case <-done:
					return
				}
			}
		}
	}()
	c := NewAnalysisCondition()
	c.SetOutput([]string{filepath.Join(t.TempDir(), "gravity.otl")})
	if err := f.StaticAnalysis(func() {}, c); err != nil {
		t.Fatal(err)
	}
}

func TestLoadStageGravityThenLateral(t *testing.T) {
	l := preloadPatternFrame()
	l.Nodes[0].Force[2], l.Nodes[1].Force[2] = -3, -12
	l.Elems[0].Cmq[2], l.Elems[0].Cmq[8] = 5, 5
	runLinearPreload(t, l)
	if l.loading == nil {
		t.Fatal("linear preload lost applied-load provenance")
	}
	x := preloadPatternFrame()
	x.Nodes[0].Force[0], x.Nodes[1].Force[0] = 5, 40
	x.RestoreState(l.SaveState())
	var log bytes.Buffer
	x.Output = &log
	c := newtonTestCondition(t)
	c.SetInit(false)
	c.SetStart(0)
	if err := x.Arclm101(nil, c); err != nil {
		t.Fatal(err)
	}
	nearNewton(t, "additional X factor", x.plasticLoad, 1, 0)
	nearNewton(t, "X displacement", x.Nodes[1].Disp[0], .4, 1e-9)
	nearNewton(t, "preserved gravity displacement", x.Nodes[1].Disp[2], -17.0/1200, 1e-9)
	nearNewton(t, "X support load and reaction", x.Nodes[0].Reaction[0], -45, 1e-8)
	nearNewton(t, "gravity fixed-end/support load reaction", x.Nodes[0].Reaction[2], 25, 1e-8)
	if !strings.Contains(log.String(), "NEW_PATTERN=true") {
		t.Fatal("load pattern change not recognised")
	}
	nearNewton(t, "applied gravity held constant", x.loading.appliedNodal()[8], -12, 0)
	nearNewton(t, "applied X load", x.loading.appliedNodal()[6], 40, 0)
	// Resume the same X pattern rather than preserving X and adding it again.
	c = newtonTestCondition(t)
	c.SetInit(false)
	c.SetDelta(-.5)
	if err := x.Arclm101(nil, c); err != nil {
		t.Fatal(err)
	}
	nearNewton(t, "same-pattern unload factor", x.plasticLoad, .5, 0)
	nearNewton(t, "same-pattern unload X displacement", x.Nodes[1].Disp[0], .2, 1e-9)
	nearNewton(t, "same-pattern unload gravity reaction", x.Nodes[0].Reaction[2], 25, 1e-8)
	// A third independent pattern must preserve L+0.5X as the new base.
	y := preloadPatternFrame()
	y.Nodes[0].Force[1], y.Nodes[1].Force[1] = -2, 24
	y.RestoreState(x.SaveState())
	c = newtonTestCondition(t)
	c.SetInit(false)
	c.SetStart(0)
	c.SetDelta(.25)
	if err := y.Arclm101(nil, c); err != nil {
		t.Fatal(err)
	}
	nearNewton(t, "third-pattern factor", y.plasticLoad, .25, 0)
	nearNewton(t, "third-pattern applied X", y.loading.appliedNodal()[6], 20, 0)
	nearNewton(t, "third-pattern applied Y", y.loading.appliedNodal()[7], 6, 0)
	nearNewton(t, "third-pattern applied gravity", y.loading.appliedNodal()[8], -12, 0)
	nearNewton(t, "third-pattern gravity reaction", y.Nodes[0].Reaction[2], 25, 1e-8)
	nearNewton(t, "third-pattern Y reaction", y.Nodes[0].Reaction[1], -5.5, 1e-8)
}

func TestLoadStageSnapshotIndependenceAndInitialise(t *testing.T) {
	f := newtonParallelFrame()
	if err := f.Arclm101(nil, newtonTestCondition(t)); err != nil {
		t.Fatal(err)
	}
	saved := f.SaveState()
	other := newtonParallelFrame()
	other.RestoreState(saved)
	other.loading.baseNodal[0] += 999
	other.loading.nodalPattern[0] += 999
	other.loading.fixedPattern[0][0] += 999
	if !reflect.DeepEqual(f.SaveState(), saved) {
		t.Fatal("applied loads share mutable snapshot data")
	}
	other.Initialise()
	if other.loading != nil || other.plasticLoadValid {
		t.Fatal("initialise kept the applied-load history")
	}
}

func TestLoadStagePreloadResidualIsCorrected(t *testing.T) {
	l := preloadPatternFrame()
	l.Nodes[1].Force[0] = 20
	runLinearPreload(t, l)
	// Mimic a small numerical residual from the legacy linear solve.
	l.Elems[0].Stress[0] -= 1e-5
	l.Elems[0].Stress[6] += 1e-5
	x := preloadPatternFrame()
	x.Nodes[1].Force[1] = 12
	x.RestoreState(l.SaveState())
	c := newtonTestCondition(t)
	c.SetInit(false)
	c.SetStart(0)
	if err := x.Arclm101(nil, c); err != nil {
		t.Fatal(err)
	}
	nearNewton(t, "preload external force retained", x.Elems[0].Stress[6], 20, 1e-8)
	nearNewton(t, "new load equilibrium", x.Elems[0].Stress[7], 12, 1e-8)
	nearNewton(t, "preload force not inferred from residual", x.loading.appliedNodal()[6], 20, 0)
}

func TestLoadStagePreservesPlasticBase(t *testing.T) {
	l := newtonParallelFrame()
	if err := l.Arclm101(nil, newtonTestCondition(t)); err != nil {
		t.Fatal(err)
	}
	x := newtonParallelFrame()
	for _, node := range x.Nodes {
		for j := 0; j < 6; j++ {
			node.Force[j] = 0
		}
	}
	x.Nodes[1].Conf[1] = false
	x.Nodes[1].Force[1] = 1
	x.RestoreState(l.SaveState())
	// State restoration includes constraints, so declare the new free Y DOF.
	x.Nodes[1].Conf[1] = false
	c := newtonTestCondition(t)
	c.SetInit(false)
	c.SetStart(0)
	if err := x.Arclm101(nil, c); err != nil {
		t.Fatal(err)
	}
	nearNewton(t, "plastic base axial load", x.Elems[0].Stress[6]+x.Elems[1].Stress[6], 220, 1e-7)
	nearNewton(t, "new Y load", x.Elems[0].Stress[7]+x.Elems[1].Stress[7], 1, 1e-7)
}
