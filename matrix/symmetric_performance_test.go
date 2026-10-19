package matrix

import (
	"math"
	"testing"
)

func TestSymmetricCSRAndShiftedInertia(t *testing.T) {
	kg := symmetricTestCOO([][]float64{{-2, .3, 0, .2}, {.3, 1, 1, 0}, {0, 1, -.1, .4}, {.2, 0, .4, -.5}})
	ke := symmetricTestCOO([][]float64{{3, .1, 0, 0}, {.1, 2, .2, 0}, {0, .2, 4, .3}, {0, 0, .3, 1}})
	conf := []bool{true, false, false, false}
	x := []float64{.2, -.7, .9}
	csr, err := ke.ToSymmetricCSR(1, conf)
	if err != nil {
		t.Fatal(err)
	}
	y := csr.MulV(x)
	for i := range y {
		want := 0.0
		for j := range x {
			want += ke.Query(i+1, j+1) * x[j]
		}
		if math.Abs(y[i]-want) > 1e-14 {
			t.Fatal("CSR product differs from dense product")
		}
	}
	for _, shift := range []float64{0, .1, .5, 2} {
		explicit := kg.AddMat(ke, shift)
		full, err := kg.SymmetricLDLTShift(ke, shift, 1, conf, nil)
		if err != nil {
			t.Fatal(err)
		}
		p, z, n, err := kg.SymmetricInertia(ke, shift, 1, conf, nil)
		if err != nil {
			t.Fatal(err)
		}
		fp, fz, fn := full.Inertia()
		if p != fp || z != fz || n != fn {
			t.Fatal("inertia-only path disagrees with solve path")
		}
		a := make([][]float64, 3)
		for i := range a {
			a[i] = make([]float64, 3)
			for j := range a[i] {
				a[i][j] = explicit.Query(i+1, j+1)
			}
		}
		checkSymmetricFactor(t, a, full, z != 0)
	}
	// Upper-only storage, zero rows, and snapshot semantics.
	upper := NewCOOMatrix(3)
	upper.Set(0, 1, 2)
	sparse, err := upper.ToSymmetricCSR(0, make([]bool, 3))
	if err != nil {
		t.Fatal(err)
	}
	upper.Set(0, 1, 10)
	got := sparse.MulV([]float64{3, 4, 5})
	if got[0] != 8 || got[1] != 6 || got[2] != 0 {
		t.Fatal("CSR upper mirroring/snapshot failure")
	}
	if _, err := upper.ToSymmetricCSR(1, make([]bool, 3)); err == nil {
		t.Fatal("accepted incorrect constraint count")
	}
}

func performanceGrid900() *COOMatrix {
	const n = 900
	a := NewCOOMatrix(n)
	for i := 0; i < n; i++ {
		a.Set(i, i, .37)
		for _, j := range []int{i + 1, i + 30} {
			if j >= n || j == i+1 && i/30 != j/30 {
				continue
			}
			a.Set(i, j, -1)
			a.Set(j, i, -1)
		}
	}
	return a
}

func BenchmarkSymmetricLDLTGrid900(b *testing.B) {
	a, conf := performanceGrid900(), make([]bool, 900)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := a.SymmetricLDLT(0, conf, nil); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSymmetricInertiaGrid900(b *testing.B) {
	a, conf := performanceGrid900(), make([]bool, 900)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, _, err := a.SymmetricInertia(nil, 0, 0, conf, nil); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSymmetricProduct900(b *testing.B) {
	a, conf := performanceGrid900(), make([]bool, 900)
	x := make([]float64, 900)
	for i := range x {
		x[i] = math.Sin(float64(i))
	}
	csr, err := a.ToSymmetricCSR(0, conf)
	if err != nil {
		b.Fatal(err)
	}
	b.Run("COO", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			a.MulV(0, conf, x)
		}
	})
	b.Run("CSR", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			csr.MulV(x)
		}
	})
}
