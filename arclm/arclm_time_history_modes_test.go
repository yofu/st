package arclm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/yofu/st/matrix"
)

func rayleighDiagonalTest(eigenvalues []float64) *timeHistoryAnalysis {
	k := matrix.NewCOOMatrix(len(eigenvalues))
	m := make([]float64, len(eigenvalues))
	for i, v := range eigenvalues {
		m[i] = 1
		k.Add(i, i, v)
	}
	a := &plasticNewtonAnalysis{ctx: context.Background(), conf: make([]bool, len(m)), writer: io.Discard}
	return &timeHistoryAnalysis{a: a, mass: m, k0: k, opts: TimeHistoryOptions{AutoRayleigh: true, Damping1: .02, Damping2: .02}}
}

func TestAutomaticRayleighRepeatedAndCloseModes(t *testing.T) {
	for _, eigenvalues := range [][]float64{{100, 100, 400, 900}, {100, 100 * (1 + .5e-6), 400, 900}} {
		d := rayleighDiagonalTest(eigenvalues)
		if err := d.resolveRayleighModes(); err != nil {
			t.Fatal(err)
		}
		if d.opts.RayleighModes != [2]int{1, 3} {
			t.Fatalf("selected=%v", d.opts.RayleighModes)
		}
		nearNewton(t, "first period", d.opts.Period1, 2*math.Pi/10, 1e-10)
		nearNewton(t, "second distinct period", d.opts.Period2, 2*math.Pi/20, 1e-10)
		d = rayleighDiagonalTest(eigenvalues)
		d.opts.RayleighModes = [2]int{1, 2}
		if err := d.resolveRayleighModes(); err == nil {
			t.Fatal("accepted explicitly coincident/close anchors")
		}
	}
	// The multiplicity is skipped by a count, rather than solving every mode.
	values := make([]float64, 65)
	for i := range values {
		values[i] = 100
	}
	values[64] = 400
	d := rayleighDiagonalTest(values)
	var log bytes.Buffer
	d.a.writer = &log
	if err := d.resolveRayleighModes(); err != nil {
		t.Fatal(err)
	}
	if d.opts.RayleighModes != [2]int{1, 65} || strings.Count(log.String(), "SELECTED_MODE=") != 2 {
		t.Fatalf("multiplicity handling: %s", log.String())
	}
	allSame := rayleighDiagonalTest([]float64{100, 100, 100})
	if err := allSame.resolveRayleighModes(); err == nil {
		t.Fatal("accepted only one distinct frequency")
	}
}

func TestAutomaticRayleighExplicitModesAndScale(t *testing.T) {
	for _, scale := range []float64{1e-8, 1, 1e8} {
		d := rayleighDiagonalTest([]float64{100 * scale, 400 * scale, 900 * scale})
		d.opts.RayleighModes = [2]int{1, 3}
		if err := d.resolveRayleighModes(); err != nil {
			t.Fatal(err)
		}
		nearNewton(t, "scaled first period", d.opts.Period1*math.Sqrt(scale), 2*math.Pi/10, 1e-9)
		nearNewton(t, "scaled third period", d.opts.Period2*math.Sqrt(scale), 2*math.Pi/30, 1e-9)
	}
	d := rayleighDiagonalTest([]float64{100, 1e16})
	if err := d.resolveRayleighModes(); err != nil {
		t.Fatal(err)
	}
	nearNewton(t, "widely separated second period", d.opts.Period2*1e8, 2*math.Pi, 1e-8)
}

func TestAutomaticRayleighUsesTransientMassAndInitialStiffness(t *testing.T) {
	f := diagnosticCantilever()
	f.Output, f.Lapch, f.Pivot = io.Discard, nil, nil
	for _, n := range f.Nodes {
		for i := range n.Force {
			n.Force[i] = 0
		}
	}
	f.Nodes[1].Mass = 2
	s := f.Elems[0].Sect
	s.Exp, s.Exq = 2, 2
	for j := 0; j < 6; j++ {
		s.Yield[2*j], s.Yield[2*j+1] = 1000, -1000
	}
	f.EigenValue = []float64{777}
	f.EigenVector = [][]float64{{123}}
	saved := f.SaveState()
	o := TimeHistoryOptions{AutoRayleigh: true, Damping1: .02, Damping2: .03, TimeStep: .01}
	d, err := newTimeHistoryAnalysis(f, context.Background(), GroundMotion{{0, 0, 0}, {.02, 0, 0}}, o)
	if err != nil {
		t.Fatal(err)
	}
	// Axial EA/(m L)=50. Condensing massless tip rotation gives
	// bending 3EI/(m L^3)=150, twice; no invented rotational inertia.
	nearNewton(t, "axial period", d.opts.Period1, 2*math.Pi/math.Sqrt(50), 1e-10)
	nearNewton(t, "bending period", d.opts.Period2, 2*math.Pi/math.Sqrt(150), 1e-10)
	for i, period := range []float64{d.opts.Period1, d.opts.Period2} {
		w := 2 * math.Pi / period
		nearNewton(t, "automatic damping ratio", d.alpha/(2*w)+d.eta*w/2, []float64{.02, .03}[i], 1e-12)
	}
	if !reflect.DeepEqual(saved, f.SaveState()) || f.EigenValue[0] != 777 || f.EigenVector[0][0] != 123 {
		t.Fatal("modal selection changed preload or reused/overwrote old modes")
	}
	if err = f.TimeHistoryNewmark(nil, GroundMotion{{0, 0, 0}, {.02, 0, 0}}, o); err != nil {
		t.Fatal(err)
	}
	state := f.TimeHistoryState()
	if state.RayleighModes != [2]int{1, 2} || state.RayleighPeriods[0] != d.opts.Period1 || state.RayleighAlpha != d.alpha {
		t.Fatalf("missing Rayleigh metadata: %+v", state)
	}
	t.Logf("modes=%v periods=%v alpha=%g eta=%g", state.RayleighModes, state.RayleighPeriods, state.RayleighAlpha, state.RayleighEta)
}

type cancelRayleighWriter struct{ cancel context.CancelFunc }

func (w cancelRayleighWriter) Write(p []byte) (int, error) {
	if strings.Contains(string(p), "SELECTED_MODE=") {
		w.cancel()
	}
	return len(p), nil
}

func TestAutomaticRayleighCancellationAndInvalidOptions(t *testing.T) {
	d := rayleighDiagonalTest([]float64{100, 400, 900})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d.a.ctx, d.a.writer = ctx, cancelRayleighWriter{cancel}
	if err := d.resolveRayleighModes(); !errors.Is(err, context.Canceled) {
		t.Fatalf("modal cancellation: %v", err)
	}
	for _, o := range []TimeHistoryOptions{
		{AutoRayleigh: true, Period1: 1, Damping1: .02, Damping2: .02},
		{AutoRayleigh: true, RayleighModes: [2]int{2, 1}},
		{AutoRayleigh: true, Damping1: math.NaN()},
		{AutoRayleigh: true, Damping2: -.02},
		{RayleighModes: [2]int{1, 2}},
	} {
		if err := validateTimeHistoryRayleigh(o); err == nil {
			t.Fatalf("accepted invalid options %+v", o)
		}
	}
	d = rayleighDiagonalTest([]float64{100})
	if err := d.resolveRayleighModes(); err == nil {
		t.Fatal("accepted insufficient finite modes")
	}
	d = rayleighDiagonalTest([]float64{100, 400})
	d.opts.RayleighModes = [2]int{1, 3}
	if err := d.resolveRayleighModes(); err == nil {
		t.Fatal("accepted missing explicit mode")
	}
	// Cancellation hooks leave the established non-context solver API intact.
	sentinel := fmt.Errorf("stop modal setup")
	k, m := matrix.NewCOOMatrix(2), matrix.NewCOOMatrix(2)
	k.Add(0, 0, 100)
	k.Add(1, 1, 400)
	m.Add(0, 0, -1)
	m.Add(1, 1, -1)
	if _, err := newBucklingSearchChecked(k, m, 0, []bool{false, false}, 1e-12, 1, nil, nil, func() error { return sentinel }); !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
}
