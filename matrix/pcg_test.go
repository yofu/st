package matrix

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
)

func TestPCGConverges(t *testing.T) {
	t.Run("scalar", func(t *testing.T) {
		a := symmetricTestCOO([][]float64{{2}}).ToLLS(0, []bool{false})
		x, err := a.PCG(context.Background(), a, 1e-12, 1, []float64{1})
		if err != nil || x[0][0] != .5 {
			t.Fatalf("x=%v, err=%v", x, err)
		}
	})
	t.Run("sparse with fill and multiple right-hand sides", func(t *testing.T) {
		// Incomplete factorization drops the fill between columns 1 and 3.
		values := [][]float64{
			{4, -1, 0, -1},
			{-1, 5, -1, 0},
			{0, -1, 4, -1},
			{-1, 0, -1, 5},
		}
		a := symmetricTestCOO(values).ToLLS(0, make([]bool, 4))
		rhs := [][]float64{{1, 2, 3, 5}, {-2, 1, 4, 0}, {0, 0, 0, 0}}
		for run := 0; run < 2; run++ {
			x, err := a.PCG(context.Background(), a, 1e-12, 20, rhs...)
			if err != nil {
				t.Fatal(err)
			}
			for k, b := range rhs {
				var norm, residual float64
				for i, row := range values {
					r := b[i]
					norm = math.Hypot(norm, r)
					for j, entry := range row {
						r -= entry * x[k][j]
					}
					residual = math.Hypot(residual, r)
				}
				if residual > 1e-12*norm {
					t.Fatalf("RHS %d residual=%g, norm=%g, x=%v", k, residual, norm, x[k])
				}
			}
		}
	})
}

func TestPCGReturnsErrors(t *testing.T) {
	toLLS := func(values [][]float64) *LLSMatrix {
		return symmetricTestCOO(values).ToLLS(0, make([]bool, len(values)))
	}
	a := toLLS([][]float64{{2, 1}, {1, 3}})
	identity := toLLS([][]float64{{1, 0}, {0, 1}})
	for _, test := range []struct {
		name  string
		a, c  *LLSMatrix
		rhs   []float64
		limit int
		want  string
	}{
		{"zero incomplete pivot", a, toLLS([][]float64{{1, 1}, {1, 1}}), []float64{1, 1}, 10, "invalid incomplete LDLT pivot 1"},
		{"negative curvature", toLLS([][]float64{{1, 2}, {2, 1}}), identity, []float64{1, -1}, 10, "p·Ap"},
		{"iteration limit", a, identity, []float64{1, 0}, 1, "did not converge in 1 iterations"},
		{"non-finite right-hand side", a, identity, []float64{math.NaN(), 0}, 10, "non-finite right-hand side"},
		{"incompatible right-hand side", a, identity, []float64{1}, 10, "has size 1, want 2"},
	} {
		t.Run(test.name, func(t *testing.T) {
			x, err := test.a.PCG(context.Background(), test.c, 1e-12, test.limit, test.rhs)
			if x != nil || err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("x=%v, err=%v, want %q", x, err, test.want)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := a.PCG(ctx, identity, 1e-12, 10, []float64{1, 0}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled PCG: %v", err)
	}
}
