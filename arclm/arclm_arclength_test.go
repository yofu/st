package arclm

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Symmetric shallow two-bar arch. All rotations are fixed and bending
// stiffness tends to zero. Its exact axial load-displacement law contains
// both a maximum and a minimum, independently checking the continuation.
func arcArch() *Frame {
	f := NewFrame()
	f.Output = io.Discard
	f.Lapch = nil
	f.Pivot = nil
	s := NewSect()
	s.Num = 1
	s.E = 1000
	s.Poi = .3
	s.Value = []float64{.01, 1e-13, 1e-13, 1e-13}
	f.Sects = []*Sect{s}
	for i, c := range [][3]float64{{-.5, 0, 0}, {0, .1, 0}, {.5, 0, 0}} {
		n := NewNode()
		n.Num = i + 1
		n.Index = i
		copy(n.Coord, c[:])
		for j := range n.Conf {
			n.Conf[j] = true
		}
		f.Nodes = append(f.Nodes, n)
	}
	f.Nodes[1].Conf[1] = false
	f.Nodes[1].Force[1] = -.01
	for i, p := range [][2]int{{0, 1}, {1, 2}} {
		e := NewElem()
		e.Num = i + 1
		e.Sect = s
		e.Enod = []*Node{f.Nodes[p[0]], f.Nodes[p[1]]}
		f.Elems = append(f.Elems, e)
	}
	return f
}

func TestArcLengthStopDuringProgressWait(t *testing.T) {
	for _, acknowledge := range []bool{false, true} {
		t.Run(fmtBool(acknowledge), func(t *testing.T) {
			f := arcArch()
			f.Lapch = make(chan int)
			c := NewAnalysisCondition()
			c.SetNlap(3)
			c.SetOutput([]string{filepath.Join(t.TempDir(), "stop.otl")})
			reached := make(chan struct{})
			c.SetArcLength(ArcLengthOptions{Step: .002, OnStep: func(_ *Frame, p ArcLengthPoint) (bool, error) {
				if p.Step == 1 {
					close(reached)
				}
				return false, nil
			}})
			done := make(chan error, 1)
			go func() { done <- f.StaticAnalysis(nil, c) }()
			defer func() {
				if f.Running() {
					_ = f.Stop()
				}
			}()
			select {
			case <-reached:
			case <-time.After(10 * time.Second):
				t.Fatal("first step did not converge")
			}
			if acknowledge {
				select {
				case <-f.Lapch:
				case <-time.After(10 * time.Second):
					t.Fatal("progress did not arrive")
				}
			}
			if err := f.Stop(); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("got %v", err)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("Stop did not release progress wait")
			}
			if f.Running() || f.geometry == nil || f.geometry.steps != 1 {
				t.Fatal("lost committed state or running lifecycle")
			}
		})
	}
}

func fmtBool(acknowledge bool) string {
	if acknowledge {
		return "waiting_for_acknowledgement"
	}
	return "waiting_for_progress_reader"
}

func TestArcLengthResumePreservesResultFiles(t *testing.T) {
	f := arcArch()
	dir := t.TempDir()
	output := filepath.Join(dir, "resume.otl")
	run := func(init bool) {
		c := NewAnalysisCondition()
		c.SetInit(init)
		c.SetNlap(2)
		c.SetOutput([]string{output})
		c.SetArcLength(ArcLengthOptions{Step: .002, OnStep: func(_ *Frame, p ArcLengthPoint) (bool, error) {
			if p.Step != f.geometry.steps {
				t.Fatalf("callback step %d != committed %d", p.Step, f.geometry.steps)
			}
			return false, nil
		}})
		if err := f.StaticAnalysis(nil, c); err != nil {
			t.Fatal(err)
		}
	}
	run(true)
	first, err := os.ReadFile(filepath.Join(dir, "resume_ARC_1.otl"))
	if err != nil {
		t.Fatal(err)
	}
	curve, err := os.ReadFile(output + ".arc.txt")
	if err != nil {
		t.Fatal(err)
	}
	run(false)
	after, err := os.ReadFile(filepath.Join(dir, "resume_ARC_1.otl"))
	if err != nil || !bytes.Equal(first, after) {
		t.Fatalf("first result overwritten: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "resume_ARC_4.otl")); err != nil {
		t.Fatal(err)
	}
	after, err = os.ReadFile(output + ".arc.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(after, curve) || len(strings.Split(strings.TrimSpace(string(after)), "\n")) != 5 {
		t.Fatal("curve history was not appended")
	}
	state := f.SaveState()
	for _, opts := range []ArcLengthOptions{{Step: .001, MinStep: .002}, {MonitorNode: 999}} {
		c := NewAnalysisCondition()
		c.SetOutput([]string{output})
		c.SetArcLength(opts)
		if err := f.StaticAnalysis(nil, c); err == nil {
			t.Fatal("accepted invalid options")
		}
		if !reflect.DeepEqual(state, f.SaveState()) {
			t.Fatal("invalid options changed the prior state")
		}
	}
}

func TestArcLengthArchAnalyticalTurningPoints(t *testing.T) {
	for _, ds := range []float64{.004, .002} {
		f := arcArch()
		c := NewAnalysisCondition()
		c.SetNlap(300)
		c.SetOutput([]string{filepath.Join(t.TempDir(), "arch.otl")})
		maxLoad := -math.MaxFloat64
		minLoad := math.MaxFloat64
		descending := false
		recovered := false
		last := 0.0
		c.SetArcLength(ArcLengthOptions{Step: ds, MaxStep: ds, LoadScale: .01, MonitorNode: 2, OnStep: func(f *Frame, p ArcLengthPoint) (bool, error) {
			y := .1 + f.Nodes[1].Disp[1]
			l := math.Hypot(.5, y)
			l0 := math.Hypot(.5, .1)
			exact := 2 * 10 / l0 * (l0 - l) * y / l / .01
			if math.Abs(exact-p.Factor) > 2e-5 {
				t.Fatalf("analytical force mismatch ds=%g y=%g lambda=%g exact=%g", ds, y, p.Factor, exact)
			}
			if p.Residual > p.Tolerance {
				t.Fatalf("unbalanced point: %+v", p)
			}
			if p.Factor < last {
				descending = true
			}
			if descending && p.Factor > last && p.Factor > 0 {
				recovered = true
			}
			last = p.Factor
			maxLoad = math.Max(maxLoad, p.Factor)
			minLoad = math.Min(minLoad, p.Factor)
			return recovered, nil
		}})
		if err := f.StaticAnalysis(nil, c); err != nil {
			t.Fatal(err)
		}
		// dP/dy=0 at l=(a^2*l0)^(1/3).
		l0 := math.Hypot(.5, .1)
		lc := math.Cbrt(.25 * l0)
		yc := math.Sqrt(lc*lc - .25)
		peak := 2 * 10 / l0 * (l0 - lc) * yc / lc / .01
		if !descending || !recovered || math.Abs(maxLoad-peak) > .005*peak || math.Abs(minLoad+peak) > .005*peak {
			t.Fatalf("ds=%g extrema [%g,%g], exact +/- %g; descended=%t recovered=%t", ds, minLoad, maxLoad, peak, descending, recovered)
		}
	}
}

func TestArcLengthCallbackCancellationKeepsEquilibrium(t *testing.T) {
	f := arcArch()
	c := NewAnalysisCondition()
	c.SetNlap(10)
	c.SetOutput([]string{filepath.Join(t.TempDir(), "cancel.otl")})
	c.SetArcLength(ArcLengthOptions{Step: .002, OnStep: func(f *Frame, p ArcLengthPoint) (bool, error) { return true, context.Canceled }})
	if err := f.StaticAnalysis(nil, c); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	// A callback error leaves its last equilibrated state, never a Newton trial.
	if f.geometry == nil || !finitePlastic(f.geometry.factor) {
		t.Fatal("lost equilibrated state")
	}
}

func TestArcLengthFailedCorrectorRollsBack(t *testing.T) {
	f := arcArch()
	c := NewAnalysisCondition()
	c.SetOutput([]string{filepath.Join(t.TempDir(), "rollback.otl")})
	c.SetNewtonOptions(NewtonOptions{MaxIterations: 1, DisableCutbacks: true})
	c.SetArcLength(ArcLengthOptions{Step: 1, LoadScale: .01})
	if err := f.StaticAnalysis(nil, c); err == nil {
		t.Fatal("expected large-step corrector failure")
	}
	for _, n := range f.Nodes {
		for _, v := range n.Disp {
			if v != 0 {
				t.Fatalf("trial displacement leaked after failure: %g", v)
			}
		}
	}
	if f.geometry != nil && f.geometry.factor != 0 {
		t.Fatal("trial load factor leaked after failure")
	}
}

func TestArcLengthRejectsInvalidOptions(t *testing.T) {
	for _, o := range []ArcLengthOptions{{Step: -1}, {Step: .01, MinStep: .02}, {LoadScale: math.NaN()}, {MonitorNode: 999}} {
		f := arcArch()
		c := NewAnalysisCondition()
		c.SetOutput([]string{filepath.Join(t.TempDir(), "bad.otl")})
		c.SetArcLength(o)
		if err := f.StaticAnalysis(nil, c); err == nil {
			t.Fatalf("accepted %+v", o)
		}
	}
}

func TestArcLengthResumePreservesPathDirection(t *testing.T) {
	run := func(f *Frame, steps int, init bool) {
		c := NewAnalysisCondition()
		c.SetInit(init)
		c.SetNlap(steps)
		c.SetOutput([]string{filepath.Join(t.TempDir(), "resume.otl")})
		c.SetArcLength(ArcLengthOptions{Step: .002, MaxStep: .002, LoadScale: .01})
		if err := f.StaticAnalysis(nil, c); err != nil {
			t.Fatal(err)
		}
	}
	one, two := arcArch(), arcArch()
	run(one, 80, true)
	run(two, 40, true)
	if two.geometry.arcDirection[len(two.geometry.arcDirection)-1] >= 0 {
		t.Fatal("test restart must be on descending branch")
	}
	var wire bytes.Buffer
	if err := EncodeWorkerFrame(&wire, two); err != nil {
		t.Fatal(err)
	}
	restored, err := DecodeWorkerFrame(&wire)
	if err != nil {
		t.Fatal(err)
	}
	two = restored
	two.Output = io.Discard
	two.Lapch = nil
	two.Pivot = nil
	run(two, 40, false)
	if math.Abs(one.geometry.factor-two.geometry.factor) > 1e-8 || math.Abs(one.Nodes[1].Disp[1]-two.Nodes[1].Disp[1]) > 1e-10 {
		t.Fatalf("resume changed path: lambda %g/%g u %g/%g", one.geometry.factor, two.geometry.factor, one.Nodes[1].Disp[1], two.Nodes[1].Disp[1])
	}
}
