package arclm

import (
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestBucklingFixedEndMomentProducesLateralTorsionalMode(t *testing.T) {
	const n = 16
	const moment = 1.0
	f := geometryBeam(n)
	for i, node := range f.Nodes {
		for j := range node.Conf {
			node.Conf[j] = true
		}
		node.Conf[4] = false
		if i > 0 && i < n {
			node.Conf[2], node.Conf[3] = false, false
		}
	}
	for _, e := range f.Elems {
		// Opposite fixed-end moments give a constant moment field. The
		// intermediate nodal loads cancel and the end rotations are fixed.
		e.Cmq[5], e.Cmq[11] = moment, -moment
	}
	path := filepath.Join(t.TempDir(), "ltb.otp")
	if err := f.Bclng001(path, true, 1, 1e-12, 1); err != nil {
		t.Fatal(err)
	}
	s := f.Sects[0]
	g := s.E / (2 * (1 + s.Poi))
	want := math.Pi * math.Sqrt(s.E*s.Value[1]*g*s.Value[3]) / moment
	nearNewton(t, "lateral-torsional buckling multiplier", f.EigenValue[0], want, .002*want)
	for _, e := range f.Elems {
		if e.Cmq[5] != moment || e.Cmq[11] != -moment {
			t.Fatal("buckling changed the input fixed-end moments")
		}
	}
	t.Logf("LTB from CMQ only: got %.10g, theory %.10g", f.EigenValue[0], want)
}

func TestEigenAnalysesRestoreStateForContinuation(t *testing.T) {
	for _, kind := range []string{"buckling", "vibration"} {
		for _, init := range []bool{false, true} {
			name := kind + "/noinit"
			if init {
				name = kind + "/init"
			}
			t.Run(name, func(t *testing.T) {
				f, control := plasticTestFrame(), plasticTestFrame()
				for _, frame := range []*Frame{f, control} {
					frame.Output, frame.Lapch, frame.Pivot = io.Discard, nil, nil
					tip := frame.Nodes[1]
					tip.Conf[3], tip.Conf[1], tip.Conf[5] = false, false, false
					tip.Force[0], tip.Mass = -20, 2
					e := frame.Elems[0]
					e.Bonds[4] = NewSect()
					e.Bonds[4].Num = 1
					e.Cmq[4] = .2
					c := newtonTestCondition(t)
					c.SetDelta(.5)
					if err := frame.Arclm101(nil, c); err != nil {
						t.Fatal(err)
					}
				}
				saved, raw := f.SaveState(), append([]float64(nil), f.Elems[0].Cmq...)
				f.Lapch = make(chan int)
				const modes = 2
				path := filepath.Join(t.TempDir(), "modes.otp")
				done := make(chan error, 1)
				go func() {
					if kind == "buckling" {
						done <- f.Bclng001(path, init, modes, 1e-12, 1)
					} else {
						done <- f.VibrationalEigenAnalysis(path, init, modes, 1e-12, 1)
					}
				}()
				published := 0
			reading:
				for {
					select {
					case lap := <-f.Lapch:
						if lap > 0 && lap <= len(f.EigenVector) && f.EigenVector[lap-1] != nil {
							if !reflect.DeepEqual(f.Nodes[1].Disp, f.EigenVector[lap-1][6:]) {
								t.Error("preview is not the single eigenmode")
							}
							published++
						}
						f.Lapch <- 1
					case err := <-done:
						if err != nil {
							t.Fatal(err)
						}
						break reading
					}
				}
				if published != modes || !reflect.DeepEqual(saved, f.SaveState()) || !reflect.DeepEqual(raw, f.Elems[0].Cmq) {
					t.Fatalf("mode publication or state restoration failed, previews=%d", published)
				}
				if output, err := os.ReadFile(path); err != nil || !strings.Contains(string(output), "EIGEN VALUE 2=") {
					t.Fatalf("mode file was not written: %v", err)
				}
				// An output error takes the same restoration path.
				f.Lapch = nil
				bad := filepath.Join(t.TempDir(), "missing", "modes.otp")
				var err error
				if kind == "buckling" {
					err = f.Bclng001(bad, init, 1, 1e-12, 1)
				} else {
					err = f.VibrationalEigenAnalysis(bad, init, 1, 1e-12, 1)
				}
				if err == nil || !strings.Contains(err.Error(), "modes.otp") || !reflect.DeepEqual(saved, f.SaveState()) {
					t.Fatalf("output error did not restore the state: %v", err)
				}
				for _, frame := range []*Frame{f, control} {
					c := newtonTestCondition(t)
					c.SetInit(false)
					c.SetDelta(.25)
					if err := frame.Arclm101(nil, c); err != nil {
						t.Fatal(err)
					}
				}
				if !reflect.DeepEqual(f.SaveState(), control.SaveState()) {
					t.Fatal("continuation differs from the run without an intervening eigenanalysis")
				}
			})
		}
	}
}

func TestBucklingStiffPortalVectorResidual(t *testing.T) {
	f := NewFrame()
	f.Output, f.Lapch, f.Pivot = io.Discard, nil, nil
	for i, xy := range [][2]float64{{0, 0}, {0, .5}, {0, 1}, {2, 0}, {2, .5}, {2, 1}} {
		node := NewNode()
		node.Num, node.Index = i+1, i
		copy(node.Coord, xy[:])
		for j := range node.Conf {
			node.Conf[j] = i%3 == 0 || (j != 0 && j != 1 && j != 5)
		}
		if i%3 == 2 {
			node.Force[1] = -1
		}
		f.Nodes = append(f.Nodes, node)
	}
	s := NewSect()
	s.Num, s.E, s.Poi = 1, 1, .3
	s.Value = []float64{1e8, 1, 1, 1}
	f.Sects = []*Sect{s}
	for i, ends := range [][2]int{{0, 1}, {1, 2}, {3, 4}, {4, 5}, {2, 5}} {
		e := NewElem()
		e.Num, e.Sect, e.Enod = i+1, s, []*Node{f.Nodes[ends[0]], f.Nodes[ends[1]]}
		f.Elems = append(f.Elems, e)
	}

	elems, err := newLinearElements(f, false)
	if err != nil {
		t.Fatal(err)
	}
	size := 6 * len(f.Nodes)
	k, conf, fixed := f.linearSystem(elems, size)
	load, err := f.linearLoad(f, false)
	if err != nil {
		t.Fatal(err)
	}
	answers, err := k.ToLLS(fixed, conf).SolveRelative(nil, 1e-12, f.linearRHS(elems, load, 1, conf, make([]float64, size)))
	if err != nil {
		t.Fatal(err)
	}
	g, _, err := f.bucklingPreloadGeometric(elems, load, f.FillConf(answers[0]))
	if err != nil {
		t.Fatal(err)
	}

	// Both residual norms are independently evaluated on the assembled pencil.
	kCSR, err := k.ToSymmetricCSR(fixed, conf)
	if err != nil {
		t.Fatal(err)
	}
	gCSR, err := g.ToSymmetricCSR(fixed, conf)
	if err != nil {
		t.Fatal(err)
	}
	factor, err := k.SymmetricLDLT(fixed, conf, nil)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "portal.otp")
	if err := f.Bclng001(path, true, 2, 1e-12, 1); err != nil {
		t.Fatal(err)
	}
	forceFloor := false
	for i, eigenvalue := range f.EigenValue {
		x := f.RemoveConf(f.EigenVector[i])
		kx, gx := kCSR.MulV(x), gCSR.MulV(x)
		mu := 1 / eigenvalue
		force := pencilForceResidual(mu, kx, gx)
		energy, err := pencilEnergyResidual(factor, mu, x, kx, gx)
		if err != nil || !finitePlastic(force) || force > 1e-6 || energy > 1e-8 {
			t.Fatalf("mode %d force=%g energy=%g err=%v", i+1, force, energy, err)
		}
		forceFloor = forceFloor || force > 1e-10
		for j := 0; j <= i; j++ {
			want := 0.0
			if j == i {
				want = 1
			}
			nearNewton(t, "KE orthogonality", Dot(f.RemoveConf(f.EigenVector[j]), kx, len(x)), want, 1e-8)
		}
		t.Logf("mode %d lambda=%.12g force=%g energy=%g", i+1, eigenvalue, force, energy)
	}
	if !forceFloor {
		t.Fatal("portal did not exercise the residual fallback")
	}
}
