package arclm

import (
	"context"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBucklingTorsionUnderAxialLoad(t *testing.T) {
	for _, test := range []struct {
		name  string
		force float64
	}{
		{"compression", -1},
		{"tension", 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := diagnosticCantilever()
			tip := f.Nodes[1]
			tip.Force[0] = test.force
			for i := range tip.Conf {
				tip.Conf[i] = i != 0 && i != 3
			}
			s := f.Elems[0].Sect
			s.Value[3] = .04
			out := filepath.Join(t.TempDir(), "torsion.otp")
			err := runTorsionOrPCGTest(t, f, func() error {
				return f.Bclng001(out, true, 1, 1e-12, 10)
			})
			if test.force > 0 {
				if err == nil || !strings.Contains(err.Error(), "no positive root") {
					t.Fatalf("tension: eigenvalues %v, err=%v", f.EigenValue, err)
				}
				if _, err := os.Stat(out); !os.IsNotExist(err) {
					t.Fatalf("tension produced an eigenvalue output: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			a, ix, iy, j := s.Value[0], s.Value[1], s.Value[2], s.Value[3]
			g := s.E / (2 * (1 + s.Poi))
			want := g * j * a / ((ix + iy) * math.Abs(test.force))
			if got := f.EigenValue[0]; math.Abs(got-want) > 1e-10*want {
				t.Fatalf("compression: got %.12g, want %.12g", got, want)
			}
		})
	}
}

func TestStaticAnalysisPCG(t *testing.T) {
	for _, singular := range []bool{false, true} {
		f := diagnosticCantilever()
		if singular {
			f.Elems[0].Sect.Value[1] = 0
		}
		c := NewAnalysisCondition()
		c.SetSolver("PCG")
		c.SetEps(1e-12)
		out := filepath.Join(t.TempDir(), "pcg.otl")
		c.SetOutput([]string{out})
		err := runTorsionOrPCGTest(t, f, func() error { return f.StaticAnalysis(nil, c) })
		if singular {
			if err == nil || !strings.Contains(err.Error(), "matrix singular at NODE 2[2] (UZ)") {
				t.Fatalf("singular PCG: %v", err)
			}
			if _, err := os.Stat(out); !os.IsNotExist(err) {
				t.Fatalf("failed PCG produced a result file: %v", err)
			}
		} else {
			if err != nil {
				t.Fatal(err)
			}
			nearNewton(t, "PCG axial displacement", f.Nodes[1].Disp[0], -.01, 1e-12)
		}
	}
}

func TestStaticAnalysisPCGStop(t *testing.T) {
	f := diagnosticCantilever()
	// Leave LAP unread so Stop can interrupt publication as well as the solve.
	c := NewAnalysisCondition()
	c.SetSolver("PCG")
	c.SetOutput([]string{filepath.Join(t.TempDir(), "pcg.otl")})
	done := make(chan error, 1)
	go func() { done <- f.StaticAnalysis(nil, c) }()
	deadline := time.Now().Add(3 * time.Second)
	for !f.Running() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if err := f.Stop(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("stop returned %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("PCG static analysis did not stop")
	}
}

func runTorsionOrPCGTest(t *testing.T, f *Frame, run func() error) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- run() }()
	timeout := time.After(3 * time.Second)
	for {
		select {
		case <-f.Lapch:
			f.Lapch <- 0
		case err := <-done:
			return err
		case <-timeout:
			t.Fatal("analysis did not finish")
			return nil
		}
	}
}
