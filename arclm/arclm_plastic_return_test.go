package arclm

import (
	"math"
	"reflect"
	"testing"

	"github.com/yofu/st/matrix"
)

func plasticGlobalIncrement(t *testing.T, e *Elem, local []float64) []float64 {
	t.Helper()
	trans, err := e.TransMatrix()
	if err != nil {
		t.Fatal(err)
	}
	global := make([]float64, 12)
	for i := range trans {
		for j := range trans[i] {
			global[j] += trans[i][j] * local[i]
		}
	}
	return global
}

func checkPlasticResponse(t *testing.T, e *Elem, u []float64) *PlasticResponse {
	t.Helper()
	before := append([]float64(nil), e.Stress...)
	r, err := e.TrialPlasticResponse(plasticGlobalIncrement(t, e, u))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, e.Stress) {
		t.Fatal("trial response changed committed forces")
	}
	c := e.plasticConnection()
	k, _ := c.StiffMatrix()
	k, err = c.ModifyHinge(k)
	if err != nil {
		t.Fatal(err)
	}
	elastic := make([]float64, 12)
	for i := range elastic {
		elastic[i] = u[i] - r.PlasticIncrement[i]
	}
	df := matrix.MatrixVector(k, elastic)
	for i := range before {
		if math.Abs(r.Stress[i]-before[i]-df[i]) > 1e-8 {
			t.Fatalf("force correction inconsistent at %d: force=%g predicted=%g", i, r.Stress[i], before[i]+df[i])
		}
	}
	work := Dot(r.Stress, r.PlasticIncrement, 12)
	if work < -1e-8 {
		t.Fatalf("negative plastic dissipation: %g", work)
	}
	probe := *c
	probe.Stress = r.Stress
	f, _, _, _, _ := probe.Coefficients(k)
	for end := 0; end < 2; end++ {
		if f[end] > RADIUS+1e-9 || r.Multipliers[end] < 0 || math.Abs(r.Multipliers[end]*(f[end]-RADIUS)) > 1e-7 {
			t.Fatalf("inadmissible return: end=%d yield=%g multiplier=%g", end, f[end], r.Multipliers[end])
		}
	}
	return r
}

func TestPlasticReturnUniaxialLoadUnloadReload(t *testing.T) {
	f := plasticTestFrame()
	e := f.Elems[0]
	for _, tc := range []struct {
		name      string
		increment float64
		force     float64
		stiffness float64
	}{
		{"elastic", .5, 50, 100},
		{"first_yield", .7, 95, 0},
		{"further_loading", .3, 95, 0},
		{"unload", -.2, 75, 100},
		{"elastic_reload", .1, 85, 100},
		{"plastic_reload", .3, 95, 0},
		{"reverse_yield", -2.2, -95, 0},
		{"reverse_unload", .2, -75, 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u := make([]float64, 12)
			u[6] = tc.increment
			r := checkPlasticResponse(t, e, u)
			if math.Abs(r.Stress[6]-tc.force) > 1e-9 || math.Abs(r.Stress[0]+tc.force) > 1e-9 {
				t.Fatalf("axial force=%g want=%g", r.Stress[6], tc.force)
			}
			if math.Abs(r.Tangent[6][6]-tc.stiffness) > 1e-8 {
				t.Fatalf("axial tangent=%g want=%g", r.Tangent[6][6], tc.stiffness)
			}
			if tc.name == "first_yield" && math.Abs(r.PlasticIncrement[6]-r.PlasticIncrement[0]-.25) > 1e-9 {
				t.Fatal("plastic axial deformation does not equal displacement minus elastic deformation")
			}
			du := plasticGlobalIncrement(t, e, u)
			if _, err := f.UpdateStressPlastic(du); err != nil {
				t.Fatal(err)
			}
			if math.Abs(e.Stress[6]-tc.force) > 1e-9 {
				t.Fatal("frame did not commit the corrected force")
			}
			k, _ := e.StiffMatrix()
			kt, err := e.PlasticMatrix(k)
			if err != nil || !reflect.DeepEqual(kt, r.Tangent) {
				t.Fatal("PlasticMatrix differs from the committed algorithmic tangent", err)
			}
			t.Logf("N=%g axial tangent=%g", r.Stress[6], r.Tangent[6][6])
		})
	}
}

func TestPlasticReturnAxialBendingAnalytical(t *testing.T) {
	f := plasticTestFrame()
	e := f.Elems[0]
	e.Sect.Value[2] = 1 // Both force-pair stiffness eigenvalues are 2EA/L=2EI/L=200.
	u := make([]float64, 12)
	u[6], u[5], u[11] = .9, .4, -.4
	r := checkPlasticResponse(t, e, u)
	factor := RADIUS / math.Hypot(.9, .8)
	for index, want := range map[int]float64{0: -90 * factor, 6: 90 * factor, 5: 80 * factor, 11: -80 * factor} {
		if math.Abs(r.Stress[index]-want) > 1e-8 {
			t.Fatalf("component=%d force=%g analytical=%g", index, r.Stress[index], want)
		}
	}
	if r.Multipliers[0] <= 0 || r.Multipliers[1] <= 0 {
		t.Fatal("coupled analytical case must exercise both independent end constraints")
	}
	t.Logf("coupled N=%g M=%g yield=%v", r.Stress[6], r.Stress[5], r.Yield)
}

func TestPlasticAlgorithmicTangentFiniteDifference(t *testing.T) {
	for _, exponents := range [][2]float64{{2, 2}, {1.5, 1.5}, {2, 3}, {3, 2}} {
		f := plasticTestFrame()
		e := f.Elems[0]
		e.Sect.Exp, e.Sect.Exq = exponents[0], exponents[1]
		for _, j := range []int{1, 2} {
			e.Sect.Yield[2*j], e.Sect.Yield[2*j+1] = 1000, -1000
		}
		u := make([]float64, 12)
		u[6], u[7], u[8], u[9], u[10], u[11] = 1.2, .12, -.08, .4, .1, -.15
		r := checkPlasticResponse(t, e, u)
		maxError, asymmetry := 0.0, 0.0
		for j := 0; j < 12; j++ {
			const step = 1e-6
			plus, minus := append([]float64(nil), u...), append([]float64(nil), u...)
			plus[j] += step
			minus[j] -= step
			rp := checkPlasticResponse(t, e, plus)
			rm := checkPlasticResponse(t, e, minus)
			for i := 0; i < 12; i++ {
				fd := (rp.Stress[i] - rm.Stress[i]) / (2 * step)
				error := math.Abs(fd-r.Tangent[i][j]) / math.Max(1, math.Abs(fd))
				maxError = math.Max(maxError, error)
				asymmetry = math.Max(asymmetry, math.Abs(r.Tangent[i][j]-r.Tangent[j][i]))
			}
		}
		if maxError > 2e-6 || asymmetry > 1e-7 {
			t.Fatalf("p,q=%v: tangent FD error=%g asymmetry=%g", exponents, maxError, asymmetry)
		}
		t.Logf("p,q=%v max scaled FD error=%g max asymmetry=%g", exponents, maxError, asymmetry)
	}
}

func TestPlasticReturnCoupledUnloadReload(t *testing.T) {
	f := plasticTestFrame()
	e := f.Elems[0]
	u := make([]float64, 12)
	u[6], u[5], u[11] = .9, .4, -.4
	r := checkPlasticResponse(t, e, u)
	e.commitPlasticResponse(r)
	saved := f.SaveState()
	other := plasticTestFrame()
	other.RestoreState(saved)
	other.Elems[0].material.plastic[0] += 1
	other.Elems[0].material.tangent[0][0] += 1
	if !reflect.DeepEqual(f.SaveState(), saved) {
		t.Fatal("plastic deformation or tangent cache shares mutable snapshot data")
	}
	unload := make([]float64, 12)
	for i := range unload {
		unload[i] = -.1 * u[i]
	}
	r = checkPlasticResponse(t, e, unload)
	if r.Multipliers != [2]float64{} {
		t.Fatal("inward coupled increment must unload elastically")
	}
	e.commitPlasticResponse(r)
	reload := make([]float64, 12)
	for i := range reload {
		reload[i] = .3 * u[i]
	}
	r = checkPlasticResponse(t, e, reload)
	if r.Multipliers[0]+r.Multipliers[1] <= 0 {
		t.Fatal("coupled reload did not return to the yield surface")
	}
	e.commitPlasticResponse(r)
	result := f.SaveState()
	f.RestoreState(saved)
	e.commitPlasticResponse(checkPlasticResponse(t, e, unload))
	e.commitPlasticResponse(checkPlasticResponse(t, e, reload))
	if !reflect.DeepEqual(f.SaveState(), result) {
		t.Fatal("plastic history or tangent was lost on restore")
	}
}

func TestPlasticReturnFrameFailureIsAtomic(t *testing.T) {
	f := plasticTestFrame()
	e := NewElem()
	e.Num, e.Sect, e.Enod = 2, f.Elems[0].Sect.Snapshot(), f.Elems[0].Enod
	e.Sect.Yield[0], e.Sect.Yield[1] = 0, 0
	f.Elems = append(f.Elems, e)
	before := f.SaveState()
	u := make([]float64, 12)
	u[6] = 1.2
	if _, err := f.UpdateStressPlastic(u); err == nil {
		t.Fatal("invalid capacity must fail")
	}
	if !reflect.DeepEqual(before, f.SaveState()) {
		t.Fatal("a failed element partially committed the frame")
	}
}

func TestPlasticReturnConnectionsAndAssembly(t *testing.T) {
	f := plasticTestFrame()
	e := f.Elems[0]
	e.Bonds[5] = NewSect() // Num=1 is a release with zero spring stiffness.
	e.Bonds[5].Num = 1
	e.Bonds[10].Num = 7
	e.Bonds[10].Value[1], e.Bonds[10].Value[2] = 3, 4
	for _, j := range []int{1, 2} {
		e.Sect.Yield[2*j], e.Sect.Yield[2*j+1] = 1000, -1000
	}
	u := make([]float64, 12)
	u[6], u[11] = .7, 1.2
	r := checkPlasticResponse(t, e, u)
	if r.Multipliers[0] != 0 || r.Multipliers[1] <= 0 {
		t.Fatal("expected only the unreleased J end to be plastic", r.Multipliers)
	}
	if math.Abs(r.Stress[5]) > 1e-10 {
		t.Fatal("released input moment was not preserved")
	}
	if _, err := f.UpdateStressPlastic(plasticGlobalIncrement(t, e, u)); err != nil {
		t.Fatal(err)
	}
	if e.Bonds[5].Num != 1 || e.Bonds[10].Num != 7 {
		t.Fatal("material update changed the input connection")
	}
	k, _, err := f.KP(1)
	if err != nil {
		t.Fatal(err)
	}
	trans, _ := e.TransMatrix()
	want := Transformation(r.Tangent, trans)
	for i := range want {
		for j := range want[i] {
			if math.Abs(k.Query(i, j)-want[i][j]) > 1e-9 {
				t.Fatal("KP does not assemble the material tangent, or applied the connections twice")
			}
		}
	}
}

func TestPlasticReturnRotatedElement(t *testing.T) {
	f := plasticTestFrame()
	e := f.Elems[0]
	e.Sect.Exp, e.Sect.Exq = 1.5, 1.5
	for _, j := range []int{1, 2} {
		e.Sect.Yield[2*j], e.Sect.Yield[2*j+1] = 1000, -1000
	}
	u := make([]float64, 12)
	u[6], u[7], u[8], u[9], u[10], u[11] = 1.2, .12, -.08, .4, .1, -.15
	reference := checkPlasticResponse(t, e, u)
	e.Enod[1].Coord = []float64{1.0 / 3, 2.0 / 3, 2.0 / 3}
	e.Cang = .37
	r := checkPlasticResponse(t, e, u)
	for i := range r.Stress {
		if math.Abs(r.Stress[i]-reference.Stress[i]) > 1e-8 {
			t.Fatal("rotation changed the local material force")
		}
		for j := range r.Stress {
			if math.Abs(r.Tangent[i][j]-reference.Tangent[i][j]) > 1e-7 {
				t.Fatal("rotation changed the local material tangent")
			}
		}
	}
	global := plasticGlobalIncrement(t, e, u)
	trans, _ := e.TransMatrix()
	worldTangent := Transformation(r.Tangent, trans)
	maxError := 0.0
	for j := range global {
		const h = 1e-6
		plus, minus := append([]float64(nil), global...), append([]float64(nil), global...)
		plus[j] += h
		minus[j] -= h
		rp, err := e.TrialPlasticResponse(plus)
		if err != nil {
			t.Fatal(err)
		}
		rm, err := e.TrialPlasticResponse(minus)
		if err != nil {
			t.Fatal(err)
		}
		for i := range global {
			fd := 0.0
			for k := range global {
				fd += trans[k][i] * (rp.Stress[k] - rm.Stress[k]) / (2 * h)
			}
			maxError = math.Max(maxError, math.Abs(fd-worldTangent[i][j])/math.Max(1, math.Abs(fd)))
		}
	}
	if maxError > 2e-6 {
		t.Fatalf("global tangent FD error=%g", maxError)
	}
	t.Logf("rotated global tangent max scaled FD error=%g", maxError)
}
