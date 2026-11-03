package arclm

import (
	"fmt"
	"math"
	"reflect"
	"testing"

	"github.com/yofu/st/matrix"
)

func TestPlasticDirectionalBoundaryAxial(t *testing.T) {
	for _, disabled := range [][2]bool{{}, {false, true}, {true, false}} {
		for _, p := range []float64{2, 1.5} {
			for _, sign := range []float64{-1, 1} {
				e := plasticTestFrame().Elems[0]
				e.Sect.ElasticEnds = disabled
				e.Sect.Exp, e.Sect.Exq = p, p
				e.Sect.Yield[0], e.Sect.Yield[1] = 115, -85 // Shifted axial surface.
				k, _ := e.StiffMatrix()
				trial := make([]float64, 12)
				trial[0], trial[6] = 15+sign*95, -15-sign*95
				for _, loading := range []bool{true, false} {
					u := make([]float64, 12)
					u[6] = -sign
					if !loading {
						u[6] *= -1
					}
					forceRate := matrix.MatrixVector(k, u)
					for _, size := range []float64{1e-8, 1, 1e8} {
						direction := append([]float64(nil), forceRate...)
						for i := range direction {
							direction[i] *= size
						}
						r, err := e.returnPlasticForcesDirectional(trial, k, direction)
						if err != nil {
							t.Fatalf("disabled=%v p=%g sign=%g load=%t: %v", disabled, p, sign, loading, err)
						}
						want := 100.0
						if loading {
							want = 0
						}
						nearNewton(t, "directional axial tangent", r.Tangent[6][6], want, 1e-8)
						for i := range trial {
							nearNewton(t, "unchanged boundary force", r.Stress[i], trial[i], 1e-8)
						}
						if loading && !r.tangentActive[0] && !r.tangentActive[1] {
							t.Fatal("loading used elastic branch")
						}
						if !loading && r.tangentActive != [2]bool{} {
							t.Fatal("unloading used plastic branch")
						}
						for end, off := range disabled {
							if off && r.tangentActive[end] {
								t.Fatal("disabled end constrained")
							}
						}
						// Compare the selected one-sided derivative with the actual
						// return map, rather than only checking the branch flags.
						const h = 1e-5
						next := append([]float64(nil), trial...)
						for i := range next {
							next[i] += h * forceRate[i]
						}
						rn, err := e.returnPlasticForces(next, k)
						if err != nil {
							t.Fatal(err)
						}
						derivative := matrix.MatrixVector(r.Tangent, u)
						for i := range derivative {
							nearNewton(t, "one-sided stress derivative", (rn.Stress[i]-r.Stress[i])/h, derivative[i], 1e-4)
						}
					}
				}
			}
		}
	}
}

func TestPlasticDirectionalMixedEndBending(t *testing.T) {
	e := plasticTestFrame().Elems[0]
	e.Sect.Value[2] = 1
	k, _ := e.StiffMatrix()
	u := make([]float64, 12)
	u[5], u[11] = .475, -.475
	trial := matrix.MatrixVector(k, u)
	for _, sign := range []float64{-1, 1} {
		direction := make([]float64, 12)
		direction[5], direction[11] = sign, sign
		forceRate := matrix.MatrixVector(k, direction)
		r, err := e.returnPlasticForcesDirectional(trial, k, forceRate)
		if err != nil {
			t.Fatal(err)
		}
		want := [2]bool{true, false}
		if sign < 0 {
			want = [2]bool{false, true}
		}
		if r.tangentActive != want {
			t.Fatalf("mixed loading/unloading: got %v want %v", r.tangentActive, want)
		}
		const h = 1e-7
		next := append([]float64(nil), trial...)
		for i := range next {
			next[i] += h * forceRate[i]
		}
		rn, err := e.returnPlasticForces(next, k)
		if err != nil {
			t.Fatal(err)
		}
		derivative := matrix.MatrixVector(r.Tangent, direction)
		for i := range derivative {
			nearNewton(t, "mixed-end one-sided derivative", (rn.Stress[i]-r.Stress[i])/h, derivative[i], 2e-3)
		}
	}
}

func TestNewtonYieldBoundarySingleMemberUnloadReload(t *testing.T) {
	for _, disabled := range [][2]bool{{}, {false, true}, {true, false}} {
		for _, solver := range []string{"LLS", "LDLT"} {
			f := plasticTestFrame()
			f.Lapch, f.Pivot = nil, nil
			f.Elems[0].Sect.ElasticEnds = disabled
			for i, factor := range []float64{95.0 / 120, .5, 95.0 / 120, -.5, -95.0 / 120, -.5, 0} {
				c := newtonTestCondition(t)
				c.SetSolver(solver)
				c.SetInit(i == 0)
				from := 0.0
				if i > 0 {
					from = f.plasticLoad
				}
				c.SetDelta(factor - from)
				if err := f.Arclm101(nil, c); err != nil {
					t.Fatalf("%s disabled=%v target=%g: %v", solver, disabled, factor, err)
				}
				nearNewton(t, "single-member force", f.Elems[0].Stress[6], 120*factor, 1e-8)
				nearNewton(t, "single-member displacement", f.Nodes[1].Disp[0], 1.2*factor, 1e-8)
			}
		}
	}
}

func TestNewtonDirectionalPlasticCycles(t *testing.T) {
	for _, disabled := range [][2]bool{{}, {false, true}, {true, false}} {
		for _, solver := range []string{"LLS", "LDLT"} {
			t.Run(fmt.Sprintf("%s_%v", solver, disabled), func(t *testing.T) {
				f := newtonParallelFrame()
				f.Elems[0].Sect.ElasticEnds = disabled
				for i, tc := range []struct{ factor, displacement, plastic float64 }{{1, 1.25, .3}, {0, .15, .3}, {1, 1.25, .3}, {0, .15, .3}, {-1, -1.25, -.3}, {0, -.15, -.3}, {1, 1.25, .3}} {
					c := newtonTestCondition(t)
					c.SetSolver(solver)
					c.SetInit(i == 0)
					c.SetNlap(10)
					from := 0.0
					if i > 0 {
						from = f.plasticLoad
					}
					c.SetDelta((tc.factor - from) / 10)
					if err := f.Arclm101(nil, c); err != nil {
						t.Fatal(err)
					}
					nearNewton(t, "cycle displacement", f.Nodes[1].Disp[0], tc.displacement, 1e-7)
					e := f.Elems[0]
					nearNewton(t, "cycle plastic extension", e.material.plastic[6]-e.material.plastic[0], tc.plastic, 1e-7)
					nearNewton(t, "cycle equilibrium", e.Stress[6]+f.Elems[1].Stress[6], 220*tc.factor, 1e-7)
					for end, off := range disabled {
						if off {
							for j := 0; j < 6; j++ {
								nearNewton(t, "disabled-end plastic history", e.material.plastic[6*end+j], 0, 0)
							}
						}
					}
				}
			})
		}
	}
}

func TestDirectionalCorrectionDoesNotCommit(t *testing.T) {
	f := newtonParallelFrame()
	c := newtonTestCondition(t)
	if err := f.Arclm101(nil, c); err != nil {
		t.Fatal(err)
	}
	before := f.SaveState()
	e := f.Elems[0].plasticConnection()
	k, _ := e.StiffMatrix()
	direction := make([]float64, 12)
	direction[0], direction[6] = -1, 1
	for _, sign := range []float64{1, -1} {
		for i := range direction {
			direction[i] *= sign
		}
		if _, err := e.returnPlasticForcesDirectional(e.Stress, k, direction); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(before, f.SaveState()) {
		t.Fatal("direction selection changed committed state")
	}
	if math.Abs(f.plasticLoad-1) > 1e-12 {
		t.Fatal("direction selection changed load factor")
	}
}
