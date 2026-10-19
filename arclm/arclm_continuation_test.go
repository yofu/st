package arclm

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestNewtonContinuationCumulativeSteps(t *testing.T) {
	f := newtonParallelFrame()
	c := newtonTestCondition(t)
	c.SetNlap(80)
	c.SetDelta(.0125)
	if err := f.Arclm101(nil, c); err != nil {
		t.Fatal(err)
	}
	if f.ContinuationStep() != 80 {
		t.Fatalf("step=%d", f.ContinuationStep())
	}
	dir := filepath.Dir(c.Output()[0])
	first := filepath.Join(dir, "newton.result@1.otp")
	before, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	saved := f.SaveState()
	// Resume on a distinct frame, exercising copied load and plastic history.
	resumed := newtonParallelFrame()
	resumed.RestoreState(saved)
	c.SetInit(false)
	resumed.Lapch = make(chan int)
	steps := make(chan []int, 1)
	go func() {
		var got []int
		for i := 0; i < 80; i++ {
			got = append(got, <-resumed.Lapch)
			resumed.Lapch <- 0
		}
		steps <- got
	}()
	if err := resumed.Arclm101(nil, c); err != nil {
		t.Fatal(err)
	}
	for i, step := range <-steps {
		if step != i+81 {
			t.Fatalf("notification=%d want=%d", step, i+81)
		}
	}
	if resumed.ContinuationStep() != 160 || f.ContinuationStep() != 80 {
		t.Fatal("step history was lost or aliased")
	}
	nearNewton(t, "continued factor", resumed.plasticLoad, 2, 1e-12)
	after, err := os.ReadFile(first)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("previous output was overwritten", err)
	}
	for _, step := range []int{80, 81, 160} {
		if _, err := os.Stat(filepath.Join(dir, fmt.Sprintf("newton.result@%d.otp", step))); err != nil {
			t.Fatal(err)
		}
	}
	one := newtonParallelFrame()
	full := newtonTestCondition(t)
	full.SetNlap(160)
	full.SetDelta(.0125)
	if err := one.Arclm101(nil, full); err != nil {
		t.Fatal(err)
	}
	for i, e := range one.Elems {
		for j, force := range e.Stress {
			nearNewton(t, "continued force", resumed.Elems[i].Stress[j], force, 1e-7)
		}
	}
	nearNewton(t, "continued displacement", resumed.Nodes[1].Disp[0], one.Nodes[1].Disp[0], 1e-8)
	// An explicit fresh analysis restarts both load and output numbering.
	resumed.Lapch = nil
	reset := newtonTestCondition(t)
	reset.SetDelta(.25)
	if err := resumed.Arclm101(nil, reset); err != nil {
		t.Fatal(err)
	}
	if resumed.ContinuationStep() != 1 {
		t.Fatal("fresh analysis did not restart at step 1")
	}
	nearNewton(t, "fresh factor", resumed.plasticLoad, .25, 1e-12)
	resumed.Nodes[1].Force[0]++
	if resumed.ContinuationStep() != 0 {
		t.Fatal("new pattern retained previous step numbers")
	}
	resumed.Initialise()
	if resumed.ContinuationStep() != 0 {
		t.Fatal("initialise retained step history")
	}
}
