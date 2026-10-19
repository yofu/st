package matrix

import (
	"math"
	"math/rand"
	"testing"
	"time"
)

func TestSparseOrderedCongruence(t *testing.T) {
	const n = 43
	rng := rand.New(rand.NewSource(721))
	for trial := 0; trial < 12; trial++ {
		u := make([][]float64, n)
		for i := range u {
			u[i] = make([]float64, n)
			u[i][i] = 1.5
			for j := i + 1; j < n && j < i+5; j++ {
				u[i][j] = rng.Float64() - .5
			}
		}
		a := make([][]float64, n)
		for i := range a {
			a[i] = make([]float64, n)
			for j := range a[i] {
				for k := 0; k+1 < n; k += 2 {
					a[i][j] += u[k][i]*u[k+1][j] + u[k+1][i]*u[k][j]
				}
				a[i][j] += 2 * u[n-1][i] * u[n-1][j]
			}
		}
		co := symmetricTestCOO(a)
		plan, err := NewSymLDLTPlan(co, nil, 0, make([]bool, n))
		if err != nil {
			t.Fatal(err)
		}
		f, stats, err := plan.FactorWithStats(0, nil, false)
		if err != nil {
			t.Fatal(err)
		}
		p, z, neg := f.Inertia()
		if p != 22 || z != 0 || neg != 21 {
			t.Fatal(p, z, neg)
		}
		if stats.TwoByTwo == 0 {
			t.Fatal("ordered factor did not exercise 2x2 pivots")
		}
		moved := false
		for i, v := range plan.order {
			moved = moved || i != v
		}
		if !moved {
			t.Fatal("ordering was not exercised")
		}
		checkSymmetricFactor(t, a, f, false)
		ref, err := co.symmetricLDLTMapReference(nil, 0, 0, make([]bool, n), nil, false, nil)
		if err != nil {
			t.Fatal(err)
		}
		rp, rz, rn := ref.Inertia()
		if p != rp || z != rz || neg != rn {
			t.Fatal("map reference inertia differs")
		}
	}
}

func TestSparseMinimumDegreeReducesStarFill(t *testing.T) {
	const n = 120
	co := NewCOOMatrix(n)
	for i := 0; i < n; i++ {
		co.Set(i, i, 200)
		if i > 0 {
			co.Set(0, i, 1)
		}
	}
	old := new(SymLDLTStatistics)
	if _, err := co.symmetricLDLTMapReference(nil, 0, 0, make([]bool, n), nil, false, old); err != nil {
		t.Fatal(err)
	}
	f, s, err := co.SymmetricLDLTWithStats(nil, 0, 0, make([]bool, n), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if s.StoredLNNZ != n-1 || s.StoredLNNZ*10 >= old.StoredLNNZ {
		t.Fatalf("ordered L=%d natural L=%d", s.StoredLNNZ, old.StoredLNNZ)
	}
	p, z, neg := f.Inertia()
	if p != n || z != 0 || neg != 0 {
		t.Fatal(p, z, neg)
	}
	t.Logf("star: stored L %d -> %d", old.StoredLNNZ, s.StoredLNNZ)
}

func TestSparsePlanShiftOwnershipAndCancellation(t *testing.T) {
	const n = 40
	a, b := NewCOOMatrix(n), NewCOOMatrix(n)
	for i := 0; i < n; i++ {
		a.Set(i, i, 3)
		b.Set(i, i, 1)
		if i+1 < n {
			a.Set(i, i+1, 1)
			b.Set(i, i+1, -1)
		}
	}
	plan, err := NewSymLDLTPlan(a, b, 0, make([]bool, n))
	if err != nil {
		t.Fatal(err)
	}
	f, err := plan.Factor(1, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	rhs := make([]float64, n)
	for i := range rhs {
		rhs[i] = 4 * math.Sin(float64(i))
	}
	x, err := f.Solve(rhs)
	if err != nil {
		t.Fatal(err)
	}
	for _, shift := range []float64{0, -4, 1, 10} {
		g, err := plan.Factor(shift, nil, true)
		if err != nil {
			t.Fatal(err)
		}
		r, err := a.symmetricLDLTMapReference(b, shift, 0, make([]bool, n), nil, true, nil)
		if err != nil {
			t.Fatal(err)
		}
		p, z, neg := g.Inertia()
		rp, rz, rn := r.Inertia()
		if p != rp || z != rz || neg != rn {
			t.Fatal("shift inertia differs", shift)
		}
	}
	// Plans snapshot their inputs, and later factors cannot corrupt saved L/D.
	a.Set(0, 0, 900)
	x2, err := f.Solve(rhs)
	if err != nil {
		t.Fatal(err)
	}
	for i := range x {
		if x[i] != x2[i] || math.Abs(x[i]-math.Sin(float64(i))) > 1e-13 {
			t.Fatal("factor storage alias", i)
		}
	}
}

func TestSparseGrid20000(t *testing.T) {
	if testing.Short() {
		t.Skip("20,000 DOF sparse factorization")
	}
	const width, height = 100, 200
	const n = width * height
	co := NewCOOMatrix(n)
	for i := 0; i < n; i++ {
		co.Set(i, i, 5)
		if i%width+1 < width {
			co.Set(i, i+1, -1)
		}
		if i+width < n {
			co.Set(i, i+width, -1)
		}
	}
	start := time.Now()
	plan, err := NewSymLDLTPlan(co, nil, 0, make([]bool, n))
	if err != nil {
		t.Fatal(err)
	}
	analysis := time.Since(start)
	f, s, err := plan.FactorWithStats(0, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	p, z, neg := f.Inertia()
	if p != n || z != 0 || neg != 0 {
		t.Fatal(p, z, neg)
	}
	if s.StoredLNNZ > n*100 || s.PeakSchurEntries > n*200 {
		t.Fatalf("excessive sparse fill: %+v", s)
	}
	expected, rhs := make([]float64, n), make([]float64, n)
	for i := range expected {
		expected[i] = math.Sin(float64(i) * .17)
	}
	for i := range rhs {
		rhs[i] = 5 * expected[i]
		if i%width > 0 {
			rhs[i] -= expected[i-1]
		}
		if i%width+1 < width {
			rhs[i] -= expected[i+1]
		}
		if i >= width {
			rhs[i] -= expected[i-width]
		}
		if i+width < n {
			rhs[i] -= expected[i+width]
		}
	}
	x, err := f.Solve(rhs)
	if err != nil {
		t.Fatal(err)
	}
	maxError := 0.0
	for i := range x {
		maxError = math.Max(maxError, math.Abs(x[i]-expected[i]))
	}
	if maxError > 1e-11 {
		t.Fatal("20k solve error", maxError)
	}
	t.Logf("DOF=%d analysis=%s numeric=%.6fs stored_L=%d peak_Schur=%d max_error=%g", n, analysis, s.NumericSeconds, s.StoredLNNZ, s.PeakSchurEntries, maxError)
}

func TestSparseIndefinite20000(t *testing.T) {
	const n = 20000
	co := NewCOOMatrix(n)
	for i := 0; i < n; i += 2 {
		co.Set(i, i+1, 2)
	}
	f, s, err := co.SymmetricLDLTWithStats(nil, 0, 0, make([]bool, n), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	p, z, neg := f.Inertia()
	if p != n/2 || z != 0 || neg != n/2 || s.TwoByTwo != n/2 {
		t.Fatal("20k 2x2 inertia", p, z, neg, s.TwoByTwo)
	}
	rhs := make([]float64, n)
	for i := range rhs {
		rhs[i] = math.Sin(float64(i))
	}
	x, err := f.Solve(rhs)
	if err != nil {
		t.Fatal(err)
	}
	for i := range x {
		if math.Abs(2*x[i^1]-rhs[i]) > 1e-14 {
			t.Fatal("20k block solve", i)
		}
	}
	t.Logf("DOF=%d 2x2_blocks=%d numeric=%.6fs peak_Schur=%d", n, s.TwoByTwo, s.NumericSeconds, s.PeakSchurEntries)
}

func TestSparseOrderedConstraints(t *testing.T) {
	const n = 40
	co := NewCOOMatrix(n + 2)
	conf := make([]bool, n+2)
	conf[0], conf[19] = true, true
	a := make([][]float64, n)
	for i := range a {
		a[i] = make([]float64, n)
		a[i][i] = 100
		if i > 0 {
			a[0][i] = 1
			a[i][0] = 1
		}
	}
	index := make([]int, 0, n)
	for i, fixed := range conf {
		if !fixed {
			index = append(index, i)
		}
	}
	for i, row := range a {
		for j, v := range row {
			if v != 0 {
				co.Set(index[i], index[j], v)
			}
		}
	}
	co.Set(0, 0, -1)
	co.Set(0, 1, 3)
	co.Set(19, 19, 0)
	f, err := co.SymmetricLDLT(2, conf, nil)
	if err != nil {
		t.Fatal(err)
	}
	checkSymmetricFactor(t, a, f, false)
}
