package arclm

import (
	"math"
	"testing"

	"github.com/yofu/st/matrix"
)

func TestInertiaUpperExpansionAndSingularEndpoint(t *testing.T) {
	ke, kg := matrix.NewCOOMatrix(2), matrix.NewCOOMatrix(2)
	ke.Set(0, 0, 1)
	ke.Set(1, 1, 1)
	kg.Set(0, 1, -.8)
	// Coordinate quotients are zero, so start at the supplied .1. The .8
	// endpoint is exactly singular; expansion must nudge past it.
	s, err := newBucklingSearch(ke, kg, 0, make([]bool, 2), 1e-12, .1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.right <= .8 {
		t.Fatal("upper bound was not expanded", s.right)
	}
	mu, _, err := s.eigenvalue(1, nil)
	if err != nil || math.Abs(mu-.8) > 1e-12 {
		t.Fatal(mu, err)
	}
	if _, _, err = s.eigenvalue(2, nil); err == nil {
		t.Fatal("invented a second positive mode")
	}
}

func nonDominantKnownSpectrum() (*matrix.COOMatrix, *matrix.COOMatrix, []float64) {
	roots := []float64{.95, .7 + 4e-14, .7, .7, .3, .1, 0, -.2}
	n := len(roots)
	ke, kg := matrix.NewCOOMatrix(n), matrix.NewCOOMatrix(n)
	k00, g00 := 1.0, -roots[0]
	for i := 1; i < n; i++ {
		v := 1.1 + .03*float64(i)
		k00 += v * v
		g00 -= roots[i] * v * v
		ke.Set(i, i, 1)
		kg.Set(i, i, -roots[i])
		ke.Set(0, i, v)
		kg.Set(0, i, -roots[i]*v)
	}
	ke.Set(0, 0, k00)
	kg.Set(0, 0, g00)
	return ke, kg, roots
}

func TestInertiaBoundsNonDominantAllRanksAndVectors(t *testing.T) {
	ke, kg, roots := nonDominantKnownSpectrum()
	s, err := newBucklingSearch(ke, kg, 0, make([]bool, len(roots)), 1e-12, 1e6, nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.gershgorin.Available {
		t.Fatal("test did not bypass Gershgorin")
	}
	// Query in an arbitrary order, including a rank inside a repeated cluster.
	for _, rank := range []int{6, 1, 4, 2, 3, 5, 1} {
		mu, _, err := s.eigenvalue(rank, nil)
		if err != nil || math.Abs(mu-roots[rank-1]) > 1e-12 {
			t.Fatalf("rank %d mu=%g err=%v", rank, mu, err)
		}
		lo, hi, nlo, nhi := s.bracket(rank)
		if !(nlo >= rank && nhi < rank && lo < roots[rank-1]+1e-14 && hi > roots[rank-1]-1e-14) {
			t.Fatal("rank was not bracketed", rank, lo, hi, nlo, nhi)
		}
	}
	var locked, klocked [][]float64
	maxResidual, maxGram := 0.0, 0.0
	for rank := 1; rank <= 6; rank++ {
		mu, C, err := s.eigenvalue(rank, nil)
		if err != nil {
			t.Fatal(err)
		}
		mu, x, err := inverseBucklingCSR(s.elastic, s.geometric, C, mu, 1e-12, locked, klocked)
		if err != nil {
			t.Fatal(err)
		}
		kx, gx := s.elastic.MulV(x), s.geometric.MulV(x)
		rn, kn, gn := 0.0, 0.0, 0.0
		for i := range x {
			rn = math.Hypot(rn, kx[i]+gx[i]/mu)
			kn = math.Hypot(kn, kx[i])
			gn = math.Hypot(gn, gx[i])
		}
		r := rn / (kn + gn/mu)
		maxResidual = math.Max(maxResidual, r)
		if math.IsNaN(r) || r > 1e-10 {
			t.Fatal("residual", r)
		}
		locked = append(locked, x)
		klocked = append(klocked, kx)
		for j, q := range locked {
			g := Dot(q, kx, len(x))
			if j == rank-1 {
				g -= 1
			}
			maxGram = math.Max(maxGram, math.Abs(g))
			if math.Abs(g) > 1e-10 {
				t.Fatal("KE Gram error", g)
			}
		}
	}
	if _, _, err = s.eigenvalue(7, nil); err == nil {
		t.Fatal("zero root was returned as positive")
	}
	t.Logf("non-DD repeated/close: residual=%g KE Gram=%g accepted=%d", maxResidual, maxGram, s.refinementsAccepted)
}

func TestInertiaProposalRejectsAdjacentRoot(t *testing.T) {
	ke, kg := matrix.NewCOOMatrix(3), matrix.NewCOOMatrix(3)
	for i, r := range []float64{.8, .5, .2} {
		ke.Set(i, i, 1)
		kg.Set(i, i, -r)
	}
	s, err := newBucklingSearch(ke, kg, 0, make([]bool, 3), 1e-12, 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.confirmCandidate(2, .1, .9, .8) {
		t.Fatal("accepted rank-1 candidate for rank 2")
	}
	if s.confirmCandidate(2, .1, .9, .2) {
		t.Fatal("accepted rank-3 candidate for rank 2")
	}
	mu, _, err := s.eigenvalue(2, nil)
	if err != nil || math.Abs(mu-.5) > 1e-12 {
		t.Fatal("lost target after rejected proposals", mu, err)
	}
	before := s.inertiaEvaluations
	for _, sample := range append([]bucklingInertiaSample(nil), s.samples...) {
		if _, err := s.count(sample.mu); err != nil {
			t.Fatal(err)
		}
	}
	if s.inertiaEvaluations != before {
		t.Fatal("cached shifts were refactored")
	}
}

func TestInertiaBoundsSparse20000NonDominant(t *testing.T) {
	if testing.Short() {
		t.Skip("20,000 DOF eigenvalue/vector test")
	}
	const n = 20000
	ke, kg := matrix.NewCOOMatrix(n), matrix.NewCOOMatrix(n)
	for i := 0; i < n; i += 2 {
		root := .1
		if i == 0 {
			root = .9
		} else if i == 2 {
			root = .6
		}
		// Each SPD block is not row diagonally dominant. KG=-root*KE
		// gives a repeated known generalized root on each two-dimensional block.
		ke.Set(i, i, 1)
		ke.Set(i, i+1, 2)
		ke.Set(i+1, i+1, 5)
		kg.Set(i, i, -root)
		kg.Set(i, i+1, -2*root)
		kg.Set(i+1, i+1, -5*root)
	}
	s, err := newBucklingSearch(ke, kg, 0, make([]bool, n), 1e-12, 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.gershgorin.Available {
		t.Fatal("unexpected Gershgorin finite bound")
	}
	var locked, klocked [][]float64
	for rank, want := range []float64{.9, .9, .6, .6} {
		mu, C, err := s.eigenvalue(rank+1, nil)
		if err != nil {
			t.Fatal(err)
		}
		mu, x, err := inverseBucklingCSR(s.elastic, s.geometric, C, mu, 1e-12, locked, klocked)
		if err != nil || math.Abs(mu-want) > 1e-11 {
			t.Fatal(mu, want, err)
		}
		kx, gx := s.elastic.MulV(x), s.geometric.MulV(x)
		rn, kn, gn := 0.0, 0.0, 0.0
		for i := range x {
			rn = math.Hypot(rn, kx[i]+gx[i]/mu)
			kn = math.Hypot(kn, kx[i])
			gn = math.Hypot(gn, gx[i])
		}
		if r := rn / (kn + gn/mu); math.IsNaN(r) || r > 1e-10 {
			t.Fatal("20k residual", r)
		}
		locked = append(locked, x)
		klocked = append(klocked, kx)
		for j, q := range locked {
			g := Dot(q, kx, n)
			if j == rank {
				g -= 1
			}
			if math.Abs(g) > 1e-10 {
				t.Fatal("20k KE Gram", g)
			}
		}
	}
	t.Logf("20k non-DD repeated modes: inertia=%d final=%d bracket=%d accepted=%d", s.inertiaEvaluations, s.solveFactorizations, s.bracketFactorizations, s.refinementsAccepted)
}
