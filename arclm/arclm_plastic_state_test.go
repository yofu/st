package arclm

import (
	"math"
	"path/filepath"
	"reflect"
	"testing"
)

func plasticTestFrame() *Frame {
	f := diagnosticCantilever()
	f.Pivot = make(chan int, 128)
	for j := range f.Nodes[1].Conf {
		f.Nodes[1].Conf[j] = j != 0
	}
	f.Nodes[1].Force[0] = 120
	s := f.Elems[0].Sect
	s.Exp, s.Exq = 2, 2
	for j := 0; j < 6; j++ {
		s.Yield[2*j], s.Yield[2*j+1] = 100, -100
	}
	return f
}

func TestPlasticYieldGradientFiniteDifference(t *testing.T) {
	cases := []struct {
		name string
		p, q float64
		v    [6]float64
	}{
		{"axial", 2, 2, [6]float64{.5}},
		{"mixed_signs", 2, 2, [6]float64{.6, -.2, .1, -.3, .4, -.8}},
		{"default_exponents", 1.5, 1.5, [6]float64{-.3, .4, -.2, .1, -.6, .2}},
		{"nested_norm", 2, 3, [6]float64{-.4, .1, -.3, .2, -.5, .7}},
		{"empty_shear_group", 3, 2, [6]float64{.4, 0, 0, -.2, .3, -.5}},
		{"empty_axial_bending_group", 3, 2, [6]float64{0, -.3, .5}},
		{"zero", 1.5, 2, [6]float64{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := plasticTestFrame()
			e := f.Elems[0]
			e.Sect.Exp, e.Sect.Exq = tc.p, tc.q
			for j := 0; j < 6; j++ {
				// Asymmetric capacities test the different end signs of N/Mz.
				center, radius := float64(j+1), 60+float64(j)*7
				e.Sect.Yield[2*j], e.Sect.Yield[2*j+1] = center+radius, center-radius
				for end := 0; end < 2; end++ {
					c := center
					if end == 1 && (j == 0 || j == 3) {
						c = -c
					}
					e.Stress[6*end+j] = c + (1-2*float64(end))*tc.v[j]*radius
				}
			}
			// Released components must have zero derivative even with nonzero stress.
			e.Bonds[10].Num = 1
			k, err := e.StiffMatrix()
			if err != nil {
				t.Fatal(err)
			}
			_, gradient, _, _, _ := e.Coefficients(k)
			maxError := 0.0
			for end := 0; end < 2; end++ {
				for j := 0; j < 6; j++ {
					index := 6*end + j
					original := e.Stress[index]
					width := .5 * (e.Sect.Yield[2*j] - e.Sect.Yield[2*j+1])
					h := 1e-6 * width
					e.Stress[index] = original + h
					plus, _, _, _, _ := e.Coefficients(k)
					e.Stress[index] = original - h
					minus, _, _, _, _ := e.Coefficients(k)
					e.Stress[index] = original
					fd := (plus[end] - minus[end]) / (2 * h)
					diff := math.Abs(gradient[end][j] - fd)
					maxError = math.Max(maxError, diff)
					if math.IsNaN(gradient[end][j]) || math.IsInf(gradient[end][j], 0) || diff > 1e-8 {
						t.Fatalf("end=%d component=%d derivative=%g finite difference=%g error=%g", end, j, gradient[end][j], fd, diff)
					}
				}
			}
			t.Logf("max absolute derivative error=%g", maxError)
		})
	}
}

func runPlasticTestAnalysis(t *testing.T, f *Frame, output string) {
	t.Helper()
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
	cond := NewAnalysisCondition()
	cond.SetOutput([]string{output})
	cond.SetNlap(1)
	if err := f.Arclm101(func() {}, cond); err != nil {
		t.Fatal(err)
	}
}

func TestPlasticInitialiseAndReanalysis(t *testing.T) {
	f := plasticTestFrame()
	e := f.Elems[0]
	e.Bonds[5].Num = 1 // Preserve an input release and an input elastic spring.
	e.Bonds[10].Num = 7
	e.Bonds[10].Value[1], e.Bonds[10].Value[2] = 3, 4
	// A second elastic bar provides a stable redistribution path after yield.
	second := NewElem()
	second.Num, second.Sect, second.Enod = 2, e.Sect.Snapshot(), e.Enod
	for j := 0; j < 6; j++ {
		second.Sect.Yield[2*j], second.Sect.Yield[2*j+1] = 1000, -1000
	}
	f.Elems = append(f.Elems, second)
	f.Nodes[1].Force[0] = 220
	input := f.SaveState()
	dir := t.TempDir()
	runPlasticTestAnalysis(t, f, filepath.Join(dir, "first.otp"))
	if !e.Phinge[0] || !e.Phinge[1] || e.Bonds[0].Num != -1 {
		t.Fatal("test did not reach material plasticity")
	}
	first := f.SaveState()
	f.Initialise()
	if got := f.SaveState(); !reflect.DeepEqual(got, input) {
		t.Fatal("initialise failed to restore the original connections and initial state")
	}
	// Running twice, without an explicit Initialise between calls, must also work.
	runPlasticTestAnalysis(t, f, filepath.Join(dir, "second.otp"))
	if !reflect.DeepEqual(f.SaveState(), first) {
		t.Fatal("second analysis differs from the first analysis")
	}
	runPlasticTestAnalysis(t, f, filepath.Join(dir, "third.otp"))
	if !reflect.DeepEqual(f.SaveState(), first) {
		t.Fatal("automatic initialisation failed before the third analysis")
	}
}

func TestPlasticStateRestoreAndIndependence(t *testing.T) {
	f := plasticTestFrame()
	e := f.Elems[0]
	initial := f.SaveState()
	u := make([]float64, 12)
	u[6] = 1.2
	if _, err := f.UpdateStressPlastic(u); err != nil {
		t.Fatal(err)
	}
	e.Energy, e.Energyb = 12, 34
	f.UpdateForm(u)
	f.Nodes[0].Reaction[0] = -120
	plastic := f.SaveState()
	f.Initialise()
	f.RestoreState(plastic)
	if !reflect.DeepEqual(f.SaveState(), plastic) {
		t.Fatal("plastic stress, bonds, hinges, energies or nodal state not restored")
	}
	// Restoring to another frame must transfer both current and original bonds.
	other := plasticTestFrame()
	other.RestoreState(plastic)
	other.Elems[0].Bonds[0].Value[0] = 999
	other.Elems[0].Phinge[0] = false
	other.Elems[0].plasticBonds[0].Value[1] = 999
	if !reflect.DeepEqual(f.SaveState(), plastic) {
		t.Fatal("restored frames alias each other or the saved state")
	}
	other.RestoreState(plastic)
	other.Initialise()
	if !reflect.DeepEqual(other.SaveState(), initial) {
		t.Fatal("initialisation after cross-frame restore lost the original bonds")
	}
	// Reusing one snapshot must not depend on earlier restore/mutation operations.
	e.Bonds[0].Num = 123
	e.Phinge[0] = false
	f.RestoreState(plastic)
	if !reflect.DeepEqual(f.SaveState(), plastic) {
		t.Fatal("saved plastic state was modified through a restored element")
	}
	f.RestoreState(initial)
	if !reflect.DeepEqual(f.SaveState(), initial) {
		t.Fatal("rollback to the pre-yield state failed")
	}
	if _, err := f.UpdateStressPlastic(u); err != nil {
		t.Fatal(err)
	}
	e.Energy, e.Energyb = 12, 34
	f.UpdateForm(u)
	f.Nodes[0].Reaction[0] = -120
	if !reflect.DeepEqual(f.SaveState(), plastic) {
		t.Fatal("repeating a step after rollback changed the result")
	}
}

func TestPlasticBondResetPreservesInputAndSharedBonds(t *testing.T) {
	f := plasticTestFrame()
	e := f.Elems[0]
	shared := NewRigid()
	e.Bonds[0], e.Bonds[6] = shared, shared
	e.setPlasticBond(0, -2)
	e.setPlasticBond(6, -3)
	if shared.Num != 0 {
		t.Fatal("plastic transition modified a shared input bond")
	}
	// An input release can also change during brace unloading; restore it to 1.
	e.Bonds[4].Num = 1
	e.setPlasticBond(4, 0)
	e.setPlasticBond(0, 0) // Unloading must not replace the original definition.
	e.Phinge[0], e.Phinge[1] = true, true
	f.Initialise()
	if e.Bonds[0].Num != 0 || e.Bonds[6].Num != 0 || e.Bonds[4].Num != 1 || e.Phinge[0] || e.Phinge[1] {
		t.Fatal("initialise failed for brace states or changed input release")
	}
}

func TestPlasticBraceStateRestoreAndInitialise(t *testing.T) {
	for _, tc := range []struct {
		name string
		u    float64
		num  int
	}{{"tension", 1.2, -2}, {"compression", -1.2, -3}} {
		t.Run(tc.name, func(t *testing.T) {
			f := plasticTestFrame()
			e := f.Elems[0]
			e.Sect.Value = []float64{1, 0, 0, 0}
			initial := f.SaveState()
			u := make([]float64, 12)
			u[6] = tc.u
			if _, err := f.UpdateStressPlastic(u); err != nil {
				t.Fatal(err)
			}
			if e.Bonds[0].Num != tc.num || e.Bonds[6].Num != tc.num {
				t.Fatalf("brace did not reach the expected state %d", tc.num)
			}
			plastic := f.SaveState()
			f.Initialise()
			if !reflect.DeepEqual(f.SaveState(), initial) {
				t.Fatal("brace did not return to its input state")
			}
			f.RestoreState(plastic)
			if !reflect.DeepEqual(f.SaveState(), plastic) {
				t.Fatal("brace plastic state was not restored")
			}
			f.Initialise()
			if !reflect.DeepEqual(f.SaveState(), initial) {
				t.Fatal("brace input state lost after plastic-state restoration")
			}
		})
	}
}
