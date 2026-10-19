package matrix

import (
	"math"
	"testing"
)

func TestGeneralizedGershgorinRealBounds(t *testing.T) {
	for _, roots := range [][]float64{{.8, .5, .5, .2}, {2, 1, 0, -3}, {1e-100, 2e-100, 3e-100}} {
		n := len(roots)
		for _, scale := range []float64{1e-100, 1, 1e100} {
			a, b := NewCOOMatrix(n), NewCOOMatrix(n)
			for i, root := range roots {
				b.Set(i, i, scale*float64(i+1))
				a.Set(i, i, -scale*float64(i+1)*root)
			}
			ac, _ := a.ToSymmetricCSR(0, make([]bool, n))
			bc, _ := b.ToSymmetricCSR(0, make([]bool, n))
			bounds, err := ac.GeneralizedGershgorin(bc, -1)
			if err != nil || !bounds.Available {
				t.Fatal(bounds, err)
			}
			for _, r := range roots {
				if !(bounds.Lower < r && r < bounds.Upper) {
					t.Fatalf("root %g outside outward bounds %+v", r, bounds)
				}
			}
		}
	}
}

func TestGeneralizedGershgorinNonDiagonalPencil(t *testing.T) {
	const n = 32
	// A=T^T diag(mu) T, B=T^T T, with T=I+sum_i t_i e_i e_0^T.
	// These real generalized roots are known independently of the enclosure.
	a, b := NewCOOMatrix(n+1), NewCOOMatrix(n+1)
	roots := make([]float64, n)
	aa, bb := .7, 1.0
	roots[0] = .7
	for i := 1; i < n; i++ {
		roots[i] = .1 + .01*float64(i)
		v := .01 * math.Sin(float64(i))
		aa += roots[i] * v * v
		bb += v * v
		a.Set(i+1, i+1, roots[i])
		b.Set(i+1, i+1, 1)
		a.Set(1, i+1, roots[i]*v)
		b.Set(1, i+1, v)
	}
	a.Set(1, 1, aa)
	b.Set(1, 1, bb)
	// The constrained row must have no effect, even if its B diagonal is zero.
	a.Set(0, 0, 1e50)
	b.Set(0, 1, 10)
	conf := make([]bool, n+1)
	conf[0] = true
	ac, _ := a.ToSymmetricCSR(1, conf)
	bc, _ := b.ToSymmetricCSR(1, conf)
	bounds, err := ac.GeneralizedGershgorin(bc, 1)
	if err != nil || !bounds.Available {
		t.Fatal(bounds, err)
	}
	for _, r := range roots {
		if r < bounds.Lower || r > bounds.Upper {
			t.Fatal("missed known generalized root", r, bounds)
		}
	}
}

func TestGeneralizedGershgorinUnavailableAndOverflow(t *testing.T) {
	a := symmetricTestCOO([][]float64{{2, 0}, {0, 3}})
	ac, _ := a.ToSymmetricCSR(0, make([]bool, 2))
	// SPD does not imply strict row diagonal dominance.
	b := symmetricTestCOO([][]float64{{1, 2}, {2, 5}})
	bc, _ := b.ToSymmetricCSR(0, make([]bool, 2))
	bounds, err := ac.GeneralizedGershgorin(bc, 1)
	if err != nil || bounds.Available || bounds.NonDominantRows != 1 {
		t.Fatal(bounds, err)
	}
	bc, _ = a.ToSymmetricCSR(0, make([]bool, 2))
	bounds, err = ac.GeneralizedGershgorin(bc, math.MaxFloat64)
	if err != nil || bounds.Available {
		t.Fatal("overflow accepted", bounds, err)
	}
	if _, err = ac.GeneralizedGershgorin(nil, 1); err == nil {
		t.Fatal("nil B accepted")
	}
}

func TestGershgorinPositiveReciprocalBounds(t *testing.T) {
	g := GeneralizedGershgorinBounds{Available: true, Lower: .2, Upper: .8}
	lo, hi, ok := g.PositiveReciprocalBounds()
	if !ok || lo >= 1.25 || hi <= 5 {
		t.Fatal(lo, hi, ok)
	}
	g.Lower = -1
	lo, hi, ok = g.PositiveReciprocalBounds()
	if !ok || lo >= 1.25 || !math.IsInf(hi, 1) {
		t.Fatal(lo, hi, ok)
	}
	g.Upper = -.1
	if _, _, ok = g.PositiveReciprocalBounds(); ok {
		t.Fatal("accepted negative-only spectrum")
	}
}
