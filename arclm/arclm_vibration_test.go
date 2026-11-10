package arclm

import (
	"io"
	"math"
	"path/filepath"
	"testing"

	"github.com/yofu/st/matrix"
)

func checkVibrationVectors(t *testing.T, k, m *matrix.SymmetricCSR, values []float64, vectors [][]float64) {
	t.Helper()
	maxResidual, maxGram := 0.0, 0.0
	for mode, x := range vectors {
		kx, mx := k.MulV(x), m.MulV(x)
		rn, kn, mn := 0.0, 0.0, 0.0
		for i := range x {
			rn = math.Hypot(rn, kx[i]-values[mode]*mx[i])
			kn = math.Hypot(kn, kx[i])
			mn = math.Hypot(mn, mx[i])
		}
		r := rn / (kn + math.Abs(values[mode])*mn)
		maxResidual = math.Max(maxResidual, r)
		if math.IsNaN(r) || r > 1e-10 {
			t.Fatalf("mode %d vibration residual=%g", mode+1, r)
		}
		for j, q := range vectors {
			g := Dot(q, mx, len(x))
			if j == mode {
				g -= 1
			}
			maxGram = math.Max(maxGram, math.Abs(g))
			if math.IsNaN(g) || math.Abs(g) > 1e-10 {
				t.Fatalf("M Gram(%d,%d) error=%g", j, mode, g)
			}
		}
	}
	t.Logf("vibration: max residual=%g max M Gram error=%g", maxResidual, maxGram)
}

func vibrationMatrixModes(t *testing.T, k, m *matrix.COOMatrix, csize int, conf []bool, want []float64) ([]float64, [][]float64) {
	t.Helper()
	negative, mass, _, err := prepareVibrationMass(k, m, csize, conf)
	if err != nil {
		t.Fatal(err)
	}
	s, err := newBucklingSearch(k, negative, csize, conf, 1e-12, 1e-4, nil)
	if err != nil {
		t.Fatal(err)
	}
	var values []float64
	var vectors, products [][]float64
	for i, w := range want {
		mu, C, err := s.eigenvalue(i+1, nil)
		if err != nil {
			t.Fatalf("mode %d: %v", i+1, err)
		}
		mu, x, err := inversePencilCSRProgress(s.elastic, s.geometric, mass, C, mu, 1e-12, vectors, products, nil)
		if err != nil {
			t.Fatalf("inverse mode %d: %v", i+1, err)
		}
		if value := 1 / mu; math.Abs(value-w) > 1e-10*math.Max(1, math.Abs(w)) {
			t.Fatalf("mode %d: omega²=%g want=%g", i+1, value, w)
		}
		values = append(values, 1/mu)
		vectors = append(vectors, x)
		products = append(products, mass.MulV(x))
	}
	checkVibrationVectors(t, s.elastic, mass, values, vectors)
	return values, vectors
}

func TestVibrationRepeatedCloseAndConstraints(t *testing.T) {
	const n = 24
	want := make([]float64, n)
	for i := range want {
		want[i] = 10 + float64(i)
	}
	want[0], want[1], want[2], want[3], want[4] = 2, 2, 2+1e-8, 5, 8
	transform := make([][]float64, n)
	for i := range transform {
		transform[i] = make([]float64, n)
		transform[i][i] = 1 + .03*float64(i)
		if i > 0 {
			transform[i][0] = .6
		}
		if i+1 < n {
			transform[i][i+1] = .2
		}
	}
	k, m := matrix.NewCOOMatrix(n+1), matrix.NewCOOMatrix(n+1)
	for i := 0; i < n; i++ {
		for j := i; j < n; j++ {
			kv, mv := 0.0, 0.0
			for h := 0; h < n; h++ {
				mv += transform[h][i] * transform[h][j]
				kv += want[h] * transform[h][i] * transform[h][j]
			}
			k.Set(i+1, j+1, kv)
			m.Set(i+1, j+1, mv)
		}
	}
	k.Set(0, 0, -100)
	m.Set(0, 0, -1) // constrained row must not enter the pencil.
	conf := make([]bool, n+1)
	conf[0] = true
	vibrationMatrixModes(t, k, m, 1, conf, want[:6])
}

func TestVibrationMasslessDOF(t *testing.T) {
	k, m := matrix.NewCOOMatrix(2), matrix.NewCOOMatrix(2)
	k.Set(0, 0, 4)
	k.Set(0, 1, 1)
	k.Set(1, 1, 3)
	m.Set(0, 0, 2)
	_, vectors := vibrationMatrixModes(t, k, m, 0, make([]bool, 2), []float64{11.0 / 6})
	if math.Abs(vectors[0][1]+vectors[0][0]/3) > 1e-10 {
		t.Fatal("massless coordinate was not statically coupled")
	}
	negative, _, _, _ := prepareVibrationMass(k, m, 0, make([]bool, 2))
	s, err := newBucklingSearch(k, negative, 0, make([]bool, 2), 1e-12, 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = s.eigenvalue(2, nil); err == nil {
		t.Fatal("infinite-frequency mode returned as finite")
	}
}

func TestVibrationTwoByTwoPivotSolve(t *testing.T) {
	k, m := matrix.NewCOOMatrix(2), matrix.NewCOOMatrix(2)
	k.Set(0, 0, 1)
	k.Set(0, 1, .5)
	k.Set(1, 1, 1)
	m.Set(0, 0, 1)
	m.Set(1, 1, 1)
	conf := make([]bool, 2)
	vibrationMatrixModes(t, k, m, 0, conf, []float64{.5, 1.5})
	negative, mass, _, err := prepareVibrationMass(k, m, 0, conf)
	if err != nil {
		t.Fatal(err)
	}
	C, err := negative.SymmetricLDLTShift(k, 1, 0, conf, nil)
	if err != nil {
		t.Fatal(err)
	}
	if blocks := C.BlockSizes(); len(blocks) != 1 || blocks[0] != 2 {
		t.Fatal("expected a mandatory 2x2 pivot", blocks)
	}
	kc, _ := k.ToSymmetricCSR(0, conf)
	gc, _ := negative.ToSymmetricCSR(0, conf)
	mu, x, err := inversePencilCSRProgress(kc, gc, mass, C, 2.0/3, 1e-12, nil, nil, nil)
	if err != nil || math.Abs(1/mu-1.5) > 1e-10 {
		t.Fatal(mu, err)
	}
	checkVibrationVectors(t, kc, mass, []float64{1 / mu}, [][]float64{x})
}

func TestVibrationRejectsInvalidMassAndStiffness(t *testing.T) {
	k, m := matrix.NewCOOMatrix(2), matrix.NewCOOMatrix(2)
	k.Set(0, 0, 1)
	k.Set(1, 1, 1)
	for _, v := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		m.Set(0, 0, v)
		if _, _, _, err := prepareVibrationMass(k, m, 0, make([]bool, 2)); err == nil {
			t.Fatal("invalid mass accepted", v)
		}
	}
	m.Set(0, 0, 1)
	k.Set(1, 1, 0)
	negative, _, _, err := prepareVibrationMass(k, m, 0, make([]bool, 2))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = newBucklingSearch(k, negative, 0, make([]bool, 2), 1e-12, 10, nil); err == nil {
		t.Fatal("rigid mechanism was not rejected")
	}
}

func TestVibrationalEigenAnalysisFrameAndDiagnostics(t *testing.T) {
	for _, diagnostics := range []bool{false, true} {
		f := diagnosticCantilever()
		f.Output = io.Discard
		f.Lapch = nil
		f.Nodes[1].Mass = 2
		k, load, err := f.KE(1)
		if err != nil {
			t.Fatal(err)
		}
		csize, conf, _ := f.AssemConf(load, 1)
		m := f.AssemMassMatrix()
		// Unit-length cantilever: axial EA/m and two bending modes 3EI/m.
		want := []float64{50, 150, 150}
		dir := t.TempDir()
		otp := filepath.Join(dir, "vibration.otp")
		if diagnostics {
			err = f.VibrationalEigenAnalysisWithDiagnostics(otp, true, 3, 1e-12, 10, &VibrationDiagnosticsOptions{Directory: dir, CPUProfile: true})
		} else {
			err = f.VibrationalEigenAnalysis(otp, true, 3, 1e-12, 10)
		}
		if err != nil {
			t.Fatal(err)
		}
		for i, v := range f.EigenValue {
			if math.Abs(v-want[i]) > 1e-9 {
				t.Fatal("frame omega squared", v, want[i])
			}
		}
		var vectors [][]float64
		for _, full := range f.EigenVector {
			var x []float64
			for i, v := range full {
				if !conf[i] {
					x = append(x, v)
				} else if v != 0 {
					t.Fatal("nonzero constrained displacement")
				}
			}
			vectors = append(vectors, x)
		}
		kc, _ := k.ToSymmetricCSR(csize, conf)
		mc, _ := m.ToSymmetricCSR(csize, conf)
		checkVibrationVectors(t, kc, mc, f.EigenValue, vectors)
		if diagnostics {
			runs, _ := filepath.Glob(filepath.Join(dir, "vibration-*"))
			if len(runs) != 1 {
				t.Fatal(runs)
			}
			events := diagnosticEvents(t, runs[0])
			if events[len(events)-1]["status"] != "complete" {
				t.Fatal(events[len(events)-1])
			}
			seen := false
			for _, e := range events {
				if e["event"] == "vibration_mode" {
					seen = true
					if e["metric"] != "M" {
						t.Fatal(e)
					}
				}
			}
			if !seen {
				t.Fatal("no vibration mode events")
			}
		}
	}
}

func TestVibration20000RepeatedModes(t *testing.T) {
	if testing.Short() {
		t.Skip("20,000 DOF vibration modes")
	}
	const n = 20000
	k, m := matrix.NewCOOMatrix(n), matrix.NewCOOMatrix(n)
	for i := 0; i < n; i += 2 {
		omega := 100.0
		if i == 0 {
			omega = 1
		} else if i == 2 {
			omega = 4
		}
		m.Set(i, i, 1)
		m.Set(i, i+1, 2)
		m.Set(i+1, i+1, 5)
		k.Set(i, i, omega)
		k.Set(i, i+1, 2*omega)
		k.Set(i+1, i+1, 5*omega)
	}
	vibrationMatrixModes(t, k, m, 0, make([]bool, n), []float64{1, 1, 4, 4})
}

func TestVibrationTipMassMatchesAutomaticRayleigh(t *testing.T) {
	const length, young, mass = 3.0, 200.0, 2.0
	const ix, iy = 1.0, 2.0
	newColumn := func() *Frame {
		f := diagnosticCantilever()
		f.Lapch, f.Pivot = nil, nil
		tip := f.Nodes[1]
		tip.Coord[0], tip.Coord[2] = 0, length
		tip.Mass = mass
		clear(tip.Force)
		section := f.Elems[0].Sect
		section.E = young
		section.Value = []float64{4, ix, iy, 1}
		section.ElasticEnds = [2]bool{true, true}
		return f
	}

	vibration := newColumn()
	if err := vibration.VibrationalEigenAnalysis(filepath.Join(t.TempDir(), "tip-mass.otp"), true, 2, 1e-12, 1); err != nil {
		t.Fatal(err)
	}
	transient := newColumn()
	motion := GroundMotion{{0, 0, 0}, {.02, .1, .2}}
	opts := TimeHistoryOptions{AutoRayleigh: true, Damping1: .02, Damping2: .02, TimeStep: .01}
	if err := transient.TimeHistoryNewmark(nil, motion, opts); err != nil {
		t.Fatal(err)
	}
	state := transient.TimeHistoryState()
	if state.RayleighModes != [2]int{1, 2} {
		t.Fatalf("auto Rayleigh selected modes %v", state.RayleighModes)
	}
	for i, inertia := range []float64{ix, iy} {
		// A free tip rotation gives the cantilever stiffness 3EI/L^3.
		want := 2 * math.Pi * math.Sqrt(mass*length*length*length/(3*young*inertia))
		period := 2 * math.Pi / math.Sqrt(vibration.EigenValue[i])
		nearNewton(t, "vibration period vs theory", period, want, 1e-10)
		nearNewton(t, "automatic Rayleigh period vs theory", state.RayleighPeriods[i], want, 1e-10)
		nearNewton(t, "vibration vs automatic Rayleigh", period, state.RayleighPeriods[i], 1e-10)
		mode := vibration.EigenVector[i]
		if math.Hypot(mode[9], mode[10]) == 0 {
			t.Fatalf("mode %d has no tip bending rotation", i+1)
		}
		t.Logf("mode %d: theory %.12g s, vibration %.12g s, Rayleigh %.12g s", i+1, want, period, state.RayleighPeriods[i])
	}

	m := vibration.AssemMassMatrix()
	for i, want := range []float64{mass, mass, mass, 0, 0, 0} {
		nearNewton(t, "tip mass matrix diagonal", m.Query(6+i, 6+i), want, 0)
	}
}
