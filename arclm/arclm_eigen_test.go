package arclm

import (
	"math"
	"strings"
	"testing"

	"github.com/yofu/st/matrix"
)

func TestBisectBucklingSpectrum(t *testing.T) {
	tests := []struct {
		name  string
		roots []float64 // reciprocal eigenvalues, descending
		right float64
	}{
		{"distinct", []float64{.8, .5, .3, .2}, 1.3},
		{"repeated", []float64{.8, .5, .5, .2}, 1.3},
		{"nearby", []float64{.8, .5000001, .4999999, .2}, 1.3},
		{"below_tolerance_gap", []float64{.8, .5 + 2e-13, .5 - 2e-13, .2}, 1.3},
		{"exact_midpoint", []float64{.75, .5, .5, .25}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := len(tt.roots)
			ke, kg := matrix.NewCOOMatrix(size), matrix.NewCOOMatrix(size)
			conf := make([]bool, size)
			for j, root := range tt.roots {
				// Non-identity KE also exercises the generalized problem.
				stiffness := float64(j + 1)
				ke.Set(j, j, stiffness)
				kg.Set(j, j, -root*stiffness)
			}
			last := 0.0
			for j, want := range tt.roots {
				mu, C, err := bisectBuckling(ke, kg, 0, conf, j+1, 1e-12, tt.right, nil, nil)
				if err != nil {
					t.Fatal(err)
				}
				if math.Abs(mu-want) > 1e-12 {
					t.Fatalf("rank %d: mu=%.16g, want %.16g", j+1, mu, want)
				}
				value := 1 / mu
				if value < last {
					t.Fatalf("out of order: %g after %g", value, last)
				}
				last = value
				if C == nil || C.Size != size {
					t.Fatal("missing vector-estimation factor")
				}
				// Exercise the same pivoted solve used by Bclng001.
				x := make([]float64, size)
				for k := range x {
					x[k] = 1
				}
				x, err = C.Solve(x)
				if err != nil {
					t.Fatal(err)
				}
				x = Normalize(x)
				norm := 0.0
				for _, v := range x {
					if math.IsNaN(v) || math.IsInf(v, 0) {
						t.Fatal("non-finite vector estimate")
					}
					norm += v * v
				}
				if math.Abs(norm-1) > 1e-12 {
					t.Fatalf("invalid vector norm: %g", norm)
				}
				t.Logf("mode %d: Lambda=%.13g (want %.13g)", j+1, value, 1/want)
			}
		})
	}
}

func TestBisectBucklingConstraintsAndMissingMode(t *testing.T) {
	ke, kg := matrix.NewCOOMatrix(4), matrix.NewCOOMatrix(4)
	for j, root := range []float64{10, .8, .5, 0} {
		ke.Set(j, j, 1)
		kg.Set(j, j, -root)
	}
	conf := []bool{true, false, false, false}
	for j, want := range []float64{.8, .5} {
		mu, C, err := bisectBuckling(ke, kg, 1, conf, j+1, 1e-12, 1.3, nil, nil)
		if err != nil || math.Abs(mu-want) > 1e-12 || C.Size != 3 {
			t.Fatalf("rank %d: mu=%g, err=%v", j+1, mu, err)
		}
	}
	if _, _, err := bisectBuckling(ke, kg, 1, conf, 3, 1e-12, 1.3, nil, nil); err == nil {
		t.Fatal("accepted a nonexistent positive mode")
	}
}

func TestBisectBucklingRejectsInvalidSearch(t *testing.T) {
	ke, kg := matrix.NewCOOMatrix(1), matrix.NewCOOMatrix(1)
	ke.Set(0, 0, 1)
	kg.Set(0, 0, -.8)
	conf := []bool{false}
	for _, tt := range []struct {
		rank       int
		eps, right float64
	}{
		{0, 1e-12, 1.3}, {2, 1e-12, 1.3},
		{1, 0, 1.3}, {1, math.NaN(), 1.3}, {1, math.Inf(1), 1.3},
		{1, 1e-12, 0}, {1, 1e-12, math.NaN()}, {1, 1e-12, math.Inf(1)},
	} {
		if _, _, err := bisectBuckling(ke, kg, 0, conf, tt.rank, tt.eps, tt.right, nil, nil); err == nil {
			t.Fatalf("accepted invalid search: %+v", tt)
		}
	}
}

func TestBisectBucklingRejectsIndefiniteKE(t *testing.T) {
	ke, kg := matrix.NewCOOMatrix(2), matrix.NewCOOMatrix(2)
	ke.Set(0, 0, 1)
	ke.Set(1, 1, -1)
	kg.Set(0, 0, -.8)
	kg.Set(1, 1, -.5)
	_, _, err := bisectBuckling(ke, kg, 0, []bool{false, false}, 1, 1e-12, 1.3, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "KE is not positive definite") {
		t.Fatalf("expected positive-definiteness error, got %v", err)
	}
}

func TestBucklingInverseIterationResidualAndOrthogonality(t *testing.T) {
	// A nonsingular, nonorthogonal change of coordinates produces dense,
	// non-diagonal KE = T^T*T and KG = -T^T*diag(roots)*T. Their exact
	// reciprocal eigenvalues are roots. This also catches Euclidean deflation.
	transform := [][]float64{
		{2, .4, -.2, .1},
		{0, 1.5, .3, -.2},
		{0, 0, .8, .25},
		{0, 0, 0, 1.2},
	}
	for _, tt := range []struct {
		name  string
		roots []float64
	}{
		{"repeated", []float64{.8, .5, .5, .2}},
		{"all_repeated", []float64{.5, .5, .5, .5}},
		{"nearby", []float64{.8, .5000001, .4999999, .2}},
		{"below_search_tolerance", []float64{.8, .5 + 2e-13, .5 - 2e-13, .2}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// DOF 0 is constrained. Store an unrelated eigenvalue there to
			// verify products and inner products consistently remove it.
			ke, kg := matrix.NewCOOMatrix(5), matrix.NewCOOMatrix(5)
			ke.Set(0, 0, 2)
			kg.Set(0, 0, -200)
			for i := 0; i < 4; i++ {
				for j := 0; j < 4; j++ {
					k, g := 0.0, 0.0
					for a := 0; a < 4; a++ {
						k += transform[a][i] * transform[a][j]
						g -= tt.roots[a] * transform[a][i] * transform[a][j]
					}
					ke.Set(i+1, j+1, k)
					kg.Set(i+1, j+1, g)
				}
			}
			conf := []bool{true, false, false, false, false}
			search, err := newBucklingSearch(ke, kg, 1, conf, 1e-12, 1.3, nil)
			if err != nil {
				t.Fatal(err)
			}
			var locked [][]float64
			var klocked [][]float64
			maxResidual, maxGramError := 0.0, 0.0
			for mode, want := range tt.roots {
				target, C, err := search.eigenvalue(mode+1, nil)
				if err != nil {
					t.Fatal(err)
				}
				mu, x, err := inverseBucklingCSR(search.elastic, search.geometric, C, target, 1e-12, locked, klocked)
				if err != nil {
					t.Fatalf("mode %d: %v", mode+1, err)
				}
				if math.Abs(mu-want) > 2e-11 {
					t.Fatalf("mode %d: mu=%g, want %g", mode+1, mu, want)
				}
				// Independently multiply dense matrix entries and check the
				// original buckling equation, using Lambda rather than mu.
				kx, gx := make([]float64, 4), make([]float64, 4)
				for i := 0; i < 4; i++ {
					for j := 0; j < 4; j++ {
						kx[i] += ke.Query(i+1, j+1) * x[j]
						gx[i] += kg.Query(i+1, j+1) * x[j]
					}
				}
				lambda := 1 / mu
				rnorm, knorm, gnorm := 0.0, 0.0, 0.0
				for i := range x {
					rnorm = math.Hypot(rnorm, kx[i]+lambda*gx[i])
					knorm = math.Hypot(knorm, kx[i])
					gnorm = math.Hypot(gnorm, gx[i])
				}
				residual := rnorm / (knorm + math.Abs(lambda)*gnorm)
				if math.IsNaN(residual) || residual > 1e-10 {
					t.Fatalf("mode %d: residual=%g", mode+1, residual)
				}
				maxResidual = math.Max(maxResidual, residual)
				locked = append(locked, x)
				klocked = append(klocked, search.elastic.MulV(x))
			}
			// Phi^T KE Phi = I proves independence also within a repeated
			// eigenspace, without assuming any particular choice of basis.
			for i, x := range locked {
				for j, y := range locked {
					gram := 0.0
					for a := 0; a < 4; a++ {
						for b := 0; b < 4; b++ {
							gram += x[a] * ke.Query(a+1, b+1) * y[b]
						}
					}
					want := 0.0
					if i == j {
						want = 1
					}
					err := math.Abs(gram - want)
					if math.IsNaN(err) || err > 1e-10 {
						t.Fatalf("Gram(%d,%d)=%g, want %g", i, j, gram, want)
					}
					maxGramError = math.Max(maxGramError, err)
				}
			}
			t.Logf("max residual=%.3g, max |Phi^T KE Phi-I|=%.3g", maxResidual, maxGramError)
		})
	}
}

func TestBucklingInverseIterationWithDistantShift(t *testing.T) {
	ke, kg := matrix.NewCOOMatrix(3), matrix.NewCOOMatrix(3)
	for i, root := range []float64{.8, .5, .2} {
		ke.Set(i, i, 1)
		kg.Set(i, i, -root)
	}
	conf := []bool{false, false, false}
	// With a distant shift, one solve is not an accurate eigenvector.
	C, err := kg.AddMat(ke, .9).SymmetricLDLT(0, conf, nil)
	if err != nil {
		t.Fatal(err)
	}
	mu, x, err := inverseBuckling(ke, kg, 0, conf, C, .8, 1e-12, nil)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(mu-.8) > 1e-12 || math.Hypot(x[1], x[2]) > 1e-9 {
		t.Fatalf("inverse iteration did not refine the vector: mu=%g, x=%v", mu, x)
	}
}

func TestBucklingInverseIterationRejectsWrongMode(t *testing.T) {
	ke, kg := matrix.NewCOOMatrix(2), matrix.NewCOOMatrix(2)
	ke.Set(0, 0, 1)
	ke.Set(1, 1, 1)
	kg.Set(0, 0, -.8)
	kg.Set(1, 1, -.5)
	conf := []bool{false, false}
	C, err := kg.AddMat(ke, .81).SymmetricLDLT(0, conf, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = inverseBuckling(ke, kg, 0, conf, C, .5, 1e-12, nil)
	if err == nil || !strings.Contains(err.Error(), "outside target") {
		t.Fatalf("accepted convergence to the wrong mode: %v", err)
	}
}

func TestBucklingWithTwoByTwoPivots(t *testing.T) {
	ke, kg := matrix.NewCOOMatrix(4), matrix.NewCOOMatrix(4)
	for i := 0; i < 4; i++ {
		ke.Set(i, i, 1)
		kg.Set(i, i, -.5)
	}
	for _, i := range []int{0, 2} {
		kg.Set(i, i+1, .25)
		kg.Set(i+1, i, .25)
	}
	conf := make([]bool, 4)
	// At shift .5 the diagonal is zero, but the matrix is nonsingular.
	// The old no-pivot factorization cannot handle it. Inertia is 2+,2-.
	shiftFactor, err := kg.AddMat(ke, .5).SymmetricLDLT(0, conf, nil)
	if err != nil {
		t.Fatal(err)
	}
	if sizes := shiftFactor.BlockSizes(); len(sizes) != 2 || sizes[0] != 2 || sizes[1] != 2 {
		t.Fatalf("expected two 2x2 blocks, got %v", sizes)
	}
	var locked [][]float64
	maxResidual, maxOrthogonality := 0.0, 0.0
	for mode, want := range []float64{.75, .75, .25, .25} {
		target, C, err := bisectBuckling(ke, kg, 0, conf, mode+1, 1e-12, 1, nil, func(mu float64, negative int) {
			if mu == .5 {
				if negative != 2 {
					t.Fatalf("2x2-block search inertia: negative=%d", negative)
				}
			}
		})
		if err != nil {
			t.Fatal(err)
		}
		// Gershgorin may change the initial interval, so the search need
		// not visit exactly .5. The explicit .5 factor above and block
		// inverse solve below guarantee 2x2 coverage independently.
		mu, x, err := inverseBuckling(ke, kg, 0, conf, C, target, 1e-12, locked)
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(mu-want) > 1e-11 {
			t.Fatalf("mode %d: mu=%g, want %g", mode+1, mu, want)
		}
		// KE=I here; compute the original-equation residual explicitly.
		lambda, normR, normX, normG := 1/mu, 0.0, 0.0, 0.0
		for i := range x {
			g := -.5*x[i] + .25*x[i^1]
			normR = math.Hypot(normR, x[i]+lambda*g)
			normX = math.Hypot(normX, x[i])
			normG = math.Hypot(normG, g)
		}
		residual := normR / (normX + math.Abs(lambda)*normG)
		maxResidual = math.Max(maxResidual, residual)
		if math.IsNaN(residual) || residual > 1e-10 {
			t.Fatalf("2x2-block spectrum residual %g", residual)
		}
		locked = append(locked, x)
		for j, q := range locked {
			want := 0.0
			if j == mode {
				want = 1
			}
			err := math.Abs(Dot(q, x, 4) - want)
			maxOrthogonality = math.Max(maxOrthogonality, err)
			if math.IsNaN(err) || err > 1e-10 {
				t.Fatalf("2x2-block spectrum orthogonality error %g", err)
			}
		}
	}
	// Explicitly exercise inverse iteration with a 2x2-block solve as well.
	C, err := kg.AddMat(ke, .55).SymmetricLDLT(0, conf, nil)
	if err != nil {
		t.Fatal(err)
	}
	if C.BlockSizes()[0] != 2 {
		t.Fatal("inverse iteration factor does not contain a 2x2 block")
	}
	if _, _, err := inverseBuckling(ke, kg, 0, conf, C, .75, 1e-12, nil); err != nil {
		t.Fatal(err)
	}
	t.Logf("2x2-block spectrum: max residual=%.3g, max orthogonality error=%.3g", maxResidual, maxOrthogonality)
}
