package arclm

import (
	"math"
	"testing"

	"github.com/yofu/st/matrix"
)

func TestBucklingSearchReusesRepeatedRoots(t *testing.T) {
	ke, kg := matrix.NewCOOMatrix(4), matrix.NewCOOMatrix(4)
	for i := 0; i < 4; i++ {
		ke.Set(i, i, float64(i+1))
		kg.Set(i, i, -.5*float64(i+1))
	}
	s, err := newBucklingSearch(ke, kg, 0, make([]bool, 4), 1e-12, 1.3, nil)
	if err != nil {
		t.Fatal(err)
	}
	var first *matrix.SymLDLTFactor
	evaluations := 0
	for rank := 1; rank <= 4; rank++ {
		mu, C, err := s.eigenvalue(rank, nil)
		if err != nil || math.Abs(mu-.5) > 1e-12 {
			t.Fatalf("rank %d: %g %v", rank, mu, err)
		}
		if rank == 1 {
			first = C
			evaluations = s.inertiaEvaluations
		} else {
			if C != first || s.inertiaEvaluations != evaluations {
				t.Fatal("repeated root was refactored")
			}
		}
	}
	if s.solveFactorizations != 1 || s.cacheHits < 3 {
		t.Fatal("missing factor or inertia reuse")
	}
	t.Logf("4 repeated modes: inertia=%d solve factors=%d cached nodes=%d", s.inertiaEvaluations, s.solveFactorizations, s.cacheHits)
}

func TestBucklingSearchRankOrder(t *testing.T) {
	ke, kg := matrix.NewCOOMatrix(4), matrix.NewCOOMatrix(4)
	roots := []float64{.8, .5 + 2e-13, .5 - 2e-13, .2}
	for i, r := range roots {
		ke.Set(i, i, 1)
		kg.Set(i, i, -r)
	}
	s, err := newBucklingSearch(ke, kg, 0, make([]bool, 4), 1e-12, 1.3, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, rank := range []int{4, 1, 3, 2, 1} {
		mu, _, err := s.eigenvalue(rank, nil)
		if err != nil || math.Abs(mu-roots[rank-1]) > 1e-12 {
			t.Fatalf("rank %d: %g %v", rank, mu, err)
		}
	}
}

func TestBucklingGershgorinBracketAndFallback(t *testing.T) {
	ke, kg := matrix.NewCOOMatrix(3), matrix.NewCOOMatrix(3)
	for i, root := range []float64{.8, .5, .2} {
		ke.Set(i, i, float64(i+1))
		kg.Set(i, i, -root*float64(i+1))
	}
	s, err := newBucklingSearch(ke, kg, 0, make([]bool, 3), 1e-12, 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !s.gershgorin.Available || s.lower <= 0 || s.lower >= .2 || s.right <= .8 || s.right >= 1 {
		t.Fatalf("wrong Gershgorin bracket: [%g,%g] %+v", s.lower, s.right, s.gershgorin)
	}
	for rank, want := range []float64{.8, .5, .2} {
		mu, _, err := s.eigenvalue(rank+1, nil)
		if err != nil || math.Abs(mu-want) > 1e-12 {
			t.Fatal(mu, want, err)
		}
	}
	// A proportional pencil has three exact roots .5 even when its SPD
	// KE does not meet the Gershgorin diagonal-dominance condition.
	ke.Set(0, 0, 1)
	ke.Set(1, 1, 5)
	ke.Set(0, 1, 2)
	kg.Set(0, 0, -.5)
	kg.Set(1, 1, -2.5)
	kg.Set(2, 2, -1.5)
	kg.Set(0, 1, -1)
	s, err = newBucklingSearch(ke, kg, 0, make([]bool, 3), 1e-12, 10, nil)
	if err != nil || s.gershgorin.Available || s.lower != 0 || s.right <= .5 {
		t.Fatal("fallback failed", s, err)
	}
	for rank := 1; rank <= 3; rank++ {
		mu, _, err := s.eigenvalue(rank, nil)
		if err != nil || math.Abs(mu-.5) > 1e-12 {
			t.Fatal("fallback root", mu, err)
		}
	}
}

// A star congruence gives a non-diagonal KE and an order-changing sparse union.
// Its known spectrum includes an exact repeated root and a separate close root.
func TestOrderedBucklingRepeatedAndCloseVectors(t *testing.T) {
	const n = 32
	ke, kg := matrix.NewCOOMatrix(n), matrix.NewCOOMatrix(n)
	roots := make([]float64, n)
	for i := range roots {
		roots[i] = .1 + .01*float64(i)
	}
	roots[0], roots[1], roots[2] = .9, .9, .9-1e-8
	// T = I + sum_i t_i e_i e_0^T; KE=T^T T, KG=-T^T diag(mu) T.
	k00, g00 := 1.0, -roots[0]
	for i := 1; i < n; i++ {
		v := .03 * math.Sin(float64(i))
		k00 += v * v
		g00 -= roots[i] * v * v
		ke.Set(i, i, 1)
		kg.Set(i, i, -roots[i])
		ke.Set(0, i, v)
		kg.Set(0, i, -roots[i]*v)
	}
	ke.Set(0, 0, k00)
	kg.Set(0, 0, g00)
	s, err := newBucklingSearch(ke, kg, 0, make([]bool, n), 1e-12, 1.3, nil)
	if err != nil {
		t.Fatal(err)
	}
	var locked, klocked [][]float64
	maxResidual, maxGram := 0.0, 0.0
	for rank, want := range []float64{.9, .9, .9 - 1e-8, .41} {
		target, C, err := s.eigenvalue(rank+1, nil)
		if err != nil {
			t.Fatal(err)
		}
		mu, x, err := inverseBucklingCSR(s.elastic, s.geometric, C, target, 1e-12, locked, klocked)
		if err != nil || math.Abs(mu-want) > 1e-11 {
			t.Fatalf("rank %d: mu=%g want=%g err=%v", rank+1, mu, want, err)
		}
		kx, gx := s.elastic.MulV(x), s.geometric.MulV(x)
		rnorm, knorm, gnorm := 0.0, 0.0, 0.0
		for j := range x {
			rnorm = math.Hypot(rnorm, kx[j]+gx[j]/mu)
			knorm = math.Hypot(knorm, kx[j])
			gnorm = math.Hypot(gnorm, gx[j])
		}
		residual := rnorm / (knorm + gnorm/mu)
		maxResidual = math.Max(maxResidual, residual)
		if math.IsNaN(residual) || residual > 1e-10 {
			t.Fatal("original equation residual", residual)
		}
		locked = append(locked, x)
		klocked = append(klocked, kx)
		for i, q := range locked {
			dot := Dot(q, kx, n)
			if i == rank {
				dot -= 1
			}
			maxGram = math.Max(maxGram, math.Abs(dot))
			if math.Abs(dot) > 1e-10 {
				t.Fatal("KE orthogonality", dot)
			}
		}
	}
	t.Logf("ordered repeated/close spectrum: max residual=%g max KE Gram error=%g", maxResidual, maxGram)
}

func benchmarkBucklingMatrices() (*matrix.COOMatrix, *matrix.COOMatrix, []bool) {
	const width = 12
	const n = width * width
	ke, kg := matrix.NewCOOMatrix(n), matrix.NewCOOMatrix(n)
	for i := 0; i < n; i++ {
		ke.Set(i, i, 1)
		kg.Set(i, i, -4)
		for _, j := range []int{i + 1, i + width} {
			if j >= n || j == i+1 && i/width != j/width {
				continue
			}
			kg.Set(i, j, 1)
			kg.Set(j, i, 1)
		}
	}
	return ke, kg, make([]bool, n)
}

// Includes initialization, four ranked searches, and deflated inverse iteration.
func BenchmarkBucklingFourModes144(b *testing.B) {
	ke, kg, conf := benchmarkBucklingMatrices()
	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		s, err := newBucklingSearch(ke, kg, 0, conf, 1e-9, 10, nil)
		if err != nil {
			b.Fatal(err)
		}
		var locked, klocked [][]float64
		for rank := 1; rank <= 4; rank++ {
			mu, C, err := s.eigenvalue(rank, nil)
			if err != nil {
				b.Fatal(err)
			}
			_, x, err := inverseBucklingCSR(s.elastic, s.geometric, C, mu, 1e-9, locked, klocked)
			if err != nil {
				b.Fatal(err)
			}
			locked = append(locked, x)
			klocked = append(klocked, s.elastic.MulV(x))
		}
	}
}
