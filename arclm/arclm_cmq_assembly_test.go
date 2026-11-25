package arclm

import (
	"io"
	"math"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/yofu/st/matrix"
)

func inclinedCMQTestFrame() *Frame {
	f := NewFrame()
	s := NewSect()
	s.Num = 1
	s.E = 2.05e7
	s.Poi = 0.3
	s.Value = []float64{0.01, 0.00012, 0.00005, 0.00002}
	for j := 0; j < 6; j++ {
		s.Yield[2*j] = 1000
		s.Yield[2*j+1] = -1000
	}
	f.Sects = []*Sect{s}
	n0, n1, n2 := NewNode(), NewNode(), NewNode()
	n0.Num, n0.Index = 1, 0
	n1.Num, n1.Index = 2, 1
	n2.Num, n2.Index = 3, 2
	n0.Coord = []float64{0, 0, 1}
	n1.Coord = []float64{3, 4, 3}
	n2.Coord = []float64{7, 1, 0}
	for j := 0; j < 6; j++ {
		n0.Conf[j] = true
		n1.Conf[j] = false
		n2.Conf[j] = true
	}
	n1.Force = []float64{0.2, -0.3, -0.5, 0.01, -0.02, 0.03}
	f.Nodes = []*Node{n0, n1, n2}

	e := NewElem()
	e.Num = 1
	e.Sect = s
	e.Enod = []*Node{n0, n1}
	e.Cang = 0.73
	for j := 0; j < 12; j++ {
		e.Bonds[j] = NewRigid()
	}
	// Asymmetric fixed-end actions with non-zero (Ci+Cj)/L in both bending planes.
	copy(e.Cmq, []float64{
		0.40, 0.25, 0.55, 0.0, -0.65, 0.30,
		0.80, 0.65, 1.35, 0.0, 1.05, -0.50,
	})
	copy(e.Stress, e.Cmq)

	// Second rigid member stabilizes node 2 even when elem 1 is pinned at both ends.
	support := NewElem()
	support.Num = 2
	support.Sect = s
	support.Enod = []*Node{n1, n2}
	for j := 0; j < 12; j++ {
		support.Bonds[j] = NewRigid()
	}
	f.Elems = []*Elem{e, support}
	return f
}

func makePinBond() *Sect {
	b := NewRigid()
	b.Num = 1
	return b
}

func makeSpringBond(num int, ky, kz float64) *Sect {
	b := NewRigid()
	b.Num = num
	b.Value = []float64{0, ky, kz, 0}
	return b
}

func TestKEKPAssemGlobalVectorMatchCondensedCMQ(t *testing.T) {
	cases := []struct {
		name      string
		configure func(*Elem)
	}{
		{
			name: "both_ends_pinned",
			configure: func(e *Elem) {
				for _, dof := range []int{4, 5, 10, 11} {
					e.Bonds[dof] = makePinBond()
				}
			},
		},
		{
			name: "i_pinned_j_spring",
			configure: func(e *Elem) {
				e.Bonds[4], e.Bonds[5] = makePinBond(), makePinBond()
				e.Bonds[10], e.Bonds[11] = makeSpringBond(7, 40, 25), makeSpringBond(7, 40, 25)
			},
		},
		{
			name: "both_ends_spring",
			configure: func(e *Elem) {
				e.Bonds[4], e.Bonds[5] = makeSpringBond(7, 40, 25), makeSpringBond(7, 40, 25)
				e.Bonds[10], e.Bonds[11] = makeSpringBond(8, 90, 60), makeSpringBond(8, 90, 60)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := inclinedCMQTestFrame()
			e := f.Elems[0]
			tc.configure(e)

			origCmq := append([]float64(nil), e.Cmq...)
			origStress := append([]float64(nil), e.Stress...)

			ke, loadKE, err := f.KE(1)
			if err != nil {
				t.Fatal(err)
			}
			kp, loadKP, err := f.KP(1)
			if err != nil {
				t.Fatal(err)
			}
			_, conf, vecGV, err := f.AssemGlobalVector(1)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(e.Cmq, origCmq) || !reflect.DeepEqual(e.Stress, origStress) {
				t.Fatal("KE/KP/AssemGlobalVector mutated input CMQ or Stress")
			}

			// Expected condensed local stiffness and fixed-end loads on elem 1,
			// plus elem 2's rigid stiffness for global comparison.
			wantKLocal, wantQ, err := plasticElasticLoads(e)
			if err != nil {
				t.Fatal(err)
			}
			tm, err := e.TransMatrix()
			if err != nil {
				t.Fatal(err)
			}
			wantK1 := Transformation(wantKLocal, tm)
			k2Local, err := f.Elems[1].StiffMatrix()
			if err != nil {
				t.Fatal(err)
			}
			tm2, err := f.Elems[1].TransMatrix()
			if err != nil {
				t.Fatal(err)
			}
			wantK2 := Transformation(k2Local, tm2)
			size := 6 * len(f.Nodes)
			wantK := make([][]float64, size)
			for i := range wantK {
				wantK[i] = make([]float64, size)
			}
			addElemK := func(el *Elem, ke [][]float64) {
				for a := 0; a < 2; a++ {
					for i := 0; i < 6; i++ {
						row := 6*el.Enod[a].Index + i
						for b := 0; b < 2; b++ {
							for j := 0; j < 6; j++ {
								col := 6*el.Enod[b].Index + j
								wantK[row][col] += ke[6*a+i][6*b+j]
							}
						}
					}
				}
			}
			addElemK(e, wantK1)
			addElemK(f.Elems[1], wantK2)
			for i := 0; i < size; i++ {
				for j := 0; j < size; j++ {
					if math.Abs(ke.Query(i, j)-wantK[i][j]) > 1e-8 {
						t.Fatalf("KE[%d][%d]=%g, want %g", i, j, ke.Query(i, j), wantK[i][j])
					}
					if math.Abs(kp.Query(i, j)-wantK[i][j]) > 1e-8 {
						t.Fatalf("KP[%d][%d]=%g, want %g", i, j, kp.Query(i, j), wantK[i][j])
					}
				}
			}

			wantLoad := make([]float64, size)
			tt := matrix.MatrixTranspose(tm)
			gFixed := matrix.MatrixVector(tt, wantQ)
			for i := 0; i < 2; i++ {
				for j := 0; j < 6; j++ {
					if !e.Enod[i].Conf[j] {
						wantLoad[6*e.Enod[i].Index+j] = -gFixed[6*i+j]
					}
				}
			}
			for i := 0; i < size; i++ {
				if math.Abs(loadKE[i]-wantLoad[i]) > 1e-10 {
					t.Fatalf("KE load[%d]=%g, want %g", i, loadKE[i], wantLoad[i])
				}
				if math.Abs(loadKP[i]-wantLoad[i]) > 1e-10 {
					t.Fatalf("KP load[%d]=%g, want %g", i, loadKP[i], wantLoad[i])
				}
			}
			_, _, vecKE := f.AssemConf(loadKE, 1)
			for i := range vecKE {
				if math.Abs(vecGV[i]-vecKE[i]) > 1e-10 {
					t.Fatalf("AssemGlobalVector[%d]=%g, want %g", i, vecGV[i], vecKE[i])
				}
			}

			// Verify KE*u_static == f_KE for the StaticAnalysis solution across all 6 free DOFs.
			f.Lapch, f.Pivot, f.Output = nil, nil, io.Discard
			cond := NewAnalysisCondition()
			cond.SetOutput([]string{filepath.Join(t.TempDir(), "static.otl")})
			if err := f.StaticAnalysis(nil, cond); err != nil {
				t.Fatal(err)
			}
			uFull := make([]float64, size)
			for i, n := range f.Nodes {
				copy(uFull[6*i:6*i+6], n.Disp)
			}
			freeIdx := 0
			for i := 0; i < size; i++ {
				if conf[i] {
					continue
				}
				var ku float64
				for j := 0; j < size; j++ {
					if !conf[j] {
						ku += ke.Query(i, j) * uFull[j]
					}
				}
				if math.Abs(ku-vecKE[freeIdx]) > 1e-8 {
					t.Fatalf("KE*u_static[%d]=%g, want RHS %g", i, ku, vecKE[freeIdx])
				}
				freeIdx++
			}
		})
	}
}

func TestKPPreservesStressAndCondensesCMQAfterPlasticUpdate(t *testing.T) {
	f := plasticTestFrame()
	e := f.Elems[0]
	e.Bonds[5] = makePinBond()
	e.Bonds[10] = makeSpringBond(7, 3, 4)
	for _, j := range []int{1, 2} {
		e.Sect.Yield[2*j], e.Sect.Yield[2*j+1] = 1000, -1000
	}
	copy(e.Cmq, []float64{0, 0.3, 0.4, 0, -0.5, 0.6, 0, 0.7, 0.6, 0, 0.8, -0.4})
	copy(e.Stress, e.Cmq)

	initialK, initialLoad, err := f.KE(1)
	if err != nil {
		t.Fatal(err)
	}
	_, _, initialVector, err := f.AssemGlobalVector(1)
	if err != nil {
		t.Fatal(err)
	}
	u := make([]float64, 12)
	u[6], u[11] = .7, 1.2
	if _, err := f.UpdateStressPlastic(plasticGlobalIncrement(t, e, u)); err != nil {
		t.Fatal(err)
	}
	committedCmq := append([]float64(nil), e.Cmq...)
	committedStress := append([]float64(nil), e.Stress...)

	if !e.Phinge[1] || e.Bonds[6].Num >= 0 {
		t.Fatal("test did not produce a plastic marker on the axial DOF")
	}
	elastic, loadKE, err := f.KE(1)
	if err != nil {
		t.Fatal(err)
	}
	_, _, vector, err := f.AssemGlobalVector(1)
	if err != nil {
		t.Fatal(err)
	}
	_, loadKP, err := f.KP(1)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loadKE, initialLoad) || !reflect.DeepEqual(loadKP, initialLoad) || !reflect.DeepEqual(vector, initialVector) {
		t.Fatal("yielding changed the applied load pattern")
	}
	for i := range u {
		for j := range u {
			nearNewton(t, "elastic stiffness after yield", elastic.Query(i, j), initialK.Query(i, j), 1e-12)
		}
	}
	if !reflect.DeepEqual(e.Cmq, committedCmq) || !reflect.DeepEqual(e.Stress, committedStress) {
		t.Fatal("KP mutated committed CMQ or Stress after plastic update")
	}
	_, wantQ, err := plasticElasticLoads(e.plasticConnection())
	if err != nil {
		t.Fatal(err)
	}
	tm, err := e.TransMatrix()
	if err != nil {
		t.Fatal(err)
	}
	gFixed := matrix.MatrixVector(matrix.MatrixTranspose(tm), wantQ)
	for i := 0; i < 2; i++ {
		for j := 0; j < 6; j++ {
			want := 0.0
			if !e.Enod[i].Conf[j] {
				want = -gFixed[6*i+j]
			}
			if math.Abs(loadKP[6*i+j]-want) > 1e-10 {
				t.Fatalf("KP load[%d]=%g, want %g", 6*i+j, loadKP[6*i+j], want)
			}
		}
	}
}

// Independent beam-side rotation equilibrium for an asymmetric triangular load.
// Node rotations are free in the assembled vectors so pin moments remain observable.
func TestCMQInitialStressAndCombinedStiffness(t *testing.T) {
	const length, ei = 6., 4100.
	for _, tc := range []struct {
		name string
		a, b float64
	}{{"pins", 0, 0}, {"unequal springs", 3000, 1000}} {
		t.Run(tc.name, func(t *testing.T) {
			f := diagnosticCantilever()
			e := f.Elems[0]
			e.Sect.E = ei
			f.Nodes[1].Coord[0] = length
			for _, n := range f.Nodes {
				clear(n.Conf)
				clear(n.Force)
			}
			for i, k := range []float64{tc.a, tc.b} {
				e.Bonds[6*i+5] = makePinBond()
				if k > 0 {
					e.Bonds[6*i+5] = makeSpringBond(7, 0, k)
				}
			}
			e.Cmq[1], e.Cmq[7], e.Cmq[5], e.Cmq[11] = 2.7, 6.3, 3.6, -5.4
			raw := append([]float64(nil), e.Cmq...)
			f.Nodes[1].Disp[0] = .2
			if err := f.Initialise(); err != nil {
				t.Fatal(err)
			}
			aa, bb, ab := 4*ei/length+tc.a, 4*ei/length+tc.b, 2*ei/length
			det := aa*bb - ab*ab
			mi := tc.a * (bb*3.6 + ab*5.4) / det
			mj := tc.b * (-aa*5.4 - ab*3.6) / det
			vj := 6.3 + (3.6-5.4-mi-mj)/length
			for j, want := range map[int]float64{1: 9 - vj, 7: vj, 5: mi, 11: mj} {
				nearNewton(t, "initial end force", e.Stress[j], want, 1e-12)
			}
			_, kgLoad, err := f.KG(0)
			if err != nil {
				t.Fatal(err)
			}
			_, combinedLoad, err := f.KEKG(0)
			if err != nil {
				t.Fatal(err)
			}
			for j := range e.Stress {
				nearNewton(t, "KG stress residual", kgLoad[j], -e.Stress[j], 1e-12)
				nearNewton(t, "KEKG stress residual", combinedLoad[j], -e.Stress[j], 1e-12)
			}
			kjj := tc.b * (1 - tc.b*aa/det) // stiffness at the j end
			kij := tc.a * tc.b * ab / det
			du := make([]float64, 12)
			for _, rotation := range []float64{0, .002, -.002} {
				du[11] = rotation
				if _, err := f.UpdateStress(du); err != nil {
					t.Fatal(err)
				}
				mi += kij * rotation
				mj += kjj * rotation
				nearNewton(t, "incremental end i moment", e.Stress[5], mi, 1e-12)
				nearNewton(t, "incremental end j moment", e.Stress[11], mj, 1e-12)
			}
			if !reflect.DeepEqual(raw, e.Cmq) {
				t.Fatal("stress recovery changed input CMQ")
			}
			for _, axial := range []float64{0, 20} {
				clear(e.Stress)
				e.Stress[0], e.Stress[6] = axial, -axial
				k, _, err := f.KE(1)
				if err != nil {
					t.Fatal(err)
				}
				g, _, err := f.KG(1)
				if err != nil {
					t.Fatal(err)
				}
				kg, _, err := f.KEKG(1)
				if err != nil {
					t.Fatal(err)
				}
				nearNewton(t, "spring rotational stiffness", k.Query(11, 11), kjj, 1e-10)
				for i := range du {
					for j := range du {
						nearNewton(t, "KEKG entry", kg.Query(i, j), k.Query(i, j)+g.Query(i, j), 1e-10)
					}
				}
			}
		})
	}
}

func TestInitialiseInvalidConnectionPreservesState(t *testing.T) {
	f := inclinedCMQTestFrame()
	f.Elems[1].Bonds[5] = makeSpringBond(7, 0, -1)
	f.Elems[1].Cmq[5] = 1
	f.Nodes[1].Disp[0] = .007
	f.Elems[0].Stress[1] = 3
	before := f.SaveState()
	if err := f.Initialise(); err == nil {
		t.Fatal("expected invalid rotational spring to be rejected")
	}
	if !reflect.DeepEqual(before, f.SaveState()) {
		t.Fatal("failed initialisation discarded committed state")
	}
}

func TestModifyCMQBothEndsPinnedBalancesShear(t *testing.T) {
	f := inclinedCMQTestFrame()
	e := f.Elems[0]
	for _, dof := range []int{4, 5, 10, 11} {
		e.Bonds[dof] = makePinBond()
	}
	_, wantQ, err := plasticElasticLoads(e)
	if err != nil {
		t.Fatal(err)
	}
	e.ModifyCMQ()
	for i := 0; i < 12; i++ {
		if math.Abs(e.Cmq[i]-wantQ[i]) > 1e-10 {
			t.Fatalf("ModifyCMQ Cmq[%d]=%g, want %g", i, e.Cmq[i], wantQ[i])
		}
		if math.Abs(e.Stress[i]-wantQ[i]) > 1e-10 {
			t.Fatalf("ModifyCMQ Stress[%d]=%g, want %g", i, e.Stress[i], wantQ[i])
		}
	}
}
