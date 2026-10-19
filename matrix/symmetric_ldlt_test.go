package matrix

import (
	"math"
	"math/rand"
	"reflect"
	"testing"
)

func symmetricTestCOO(a [][]float64) *COOMatrix {
	c := NewCOOMatrix(len(a))
	for i, row := range a {
		for j, v := range row {
			c.Set(i, j, v)
		}
	}
	return c
}

func checkSymmetricFactor(t *testing.T, a [][]float64, f *SymLDLTFactor, singular bool) {
	t.Helper()
	n := len(a)
	l := make([][]float64, n)
	for i := range l {
		l[i] = make([]float64, n)
		l[i][i] = 1
		for _, e := range f.lower[i] {
			l[i][e.column] = e.value
		}
	}
	maxA, maxError := 0.0, 0.0
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			v := 0.0
			for _, b := range f.blocks {
				k := b.start
				v += l[i][k] * b.a * l[j][k]
				if b.size == 2 {
					v += l[i][k]*b.b*l[j][k+1] + l[i][k+1]*b.b*l[j][k] + l[i][k+1]*b.c*l[j][k+1]
				}
			}
			want := a[f.permutation[i]][f.permutation[j]]
			maxA = math.Max(maxA, math.Abs(want))
			maxError = math.Max(maxError, math.Abs(v-want))
		}
	}
	if math.IsNaN(maxError) || maxError > 2e-12*maxA {
		t.Fatalf("A[p,p] != LDL^T: relative error %g", maxError/maxA)
	}
	rhs, expected := make([]float64, n), make([]float64, n)
	for i := range expected {
		expected[i] = math.Sin(float64(i + 1))
	}
	for i := range rhs {
		for j := range rhs {
			rhs[i] += a[i][j] * expected[j]
		}
	}
	before := append([]float64(nil), rhs...)
	x, err := f.Solve(rhs)
	if singular {
		if err == nil {
			t.Fatal("singular solve succeeded")
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, rhs) {
		t.Fatal("Solve modified RHS")
	}
	for repeat := 0; repeat < 2; repeat++ {
		x2, err := f.Solve(rhs)
		if err != nil || !reflect.DeepEqual(x, x2) {
			t.Fatal("factor was modified by Solve")
		}
	}
	rnorm, bnorm := 0.0, 0.0
	for i := range rhs {
		r := -rhs[i]
		for j := range rhs {
			r += a[i][j] * x[j]
		}
		rnorm = math.Hypot(rnorm, r)
		bnorm = math.Hypot(bnorm, rhs[i])
		if math.Abs(x[i]-expected[i]) > 2e-10 {
			t.Fatalf("solution[%d]=%g, want %g", i, x[i], expected[i])
		}
	}
	if rnorm > 2e-12*bnorm {
		t.Fatalf("solve relative residual %g", rnorm/bnorm)
	}
}

func TestSymmetricLDLTBlocksAndPermutation(t *testing.T) {
	for _, tt := range []struct {
		name      string
		a         [][]float64
		inertia   [3]int
		two, swap bool
	}{
		{"two_zero_diagonals", [][]float64{{0, 2}, {2, 0}}, [3]int{1, 0, 1}, true, false},
		{"two_nonzero_diagonals", [][]float64{{.1, 2}, {2, -.2}}, [3]int{1, 0, 1}, true, false},
		{"two_with_swap", [][]float64{{0, 0, 2}, {0, 3, .5}, {2, .5, 0}}, [3]int{2, 0, 1}, true, true},
		{"one_with_swap", [][]float64{{.01, 1, 0}, {1, 10, 1}, {0, 1, 2}}, [3]int{2, 0, 1}, false, true},
		{"swap_after_elimination", [][]float64{{5, .2, .3, .4}, {.2, 0, 0, 2}, {.3, 0, 3, .5}, {.4, 2, .5, 0}}, [3]int{3, 0, 1}, true, true},
		{"singular", [][]float64{{1, 0, 0}, {0, 0, 0}, {0, 0, -2}}, [3]int{1, 1, 1}, false, false},
		{"singular_with_two", [][]float64{{0, 1, 0}, {1, 0, 0}, {0, 0, 0}}, [3]int{1, 1, 1}, true, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := symmetricTestCOO(tt.a)
			before := c.String()
			progress := make(chan int, len(tt.a))
			f, err := c.SymmetricLDLT(0, make([]bool, len(tt.a)), progress)
			if err != nil {
				t.Fatal(err)
			}
			if c.String() != before {
				t.Fatal("factorization modified input")
			}
			p, z, neg := f.Inertia()
			if got := [3]int{p, z, neg}; got != tt.inertia {
				t.Fatalf("inertia %v, want %v", got, tt.inertia)
			}
			hasTwo, hasSwap := false, false
			for _, size := range f.BlockSizes() {
				hasTwo = hasTwo || size == 2
			}
			for i, p := range f.permutation {
				hasSwap = hasSwap || i != p
			}
			if hasTwo != tt.two || hasSwap != tt.swap {
				t.Fatalf("unexpected blocks/permutation: %v, %v", f.BlockSizes(), f.permutation)
			}
			for i := range tt.a {
				if got := <-progress; got != i {
					t.Fatalf("progress %d, want %d", got, i)
				}
			}
			checkSymmetricFactor(t, tt.a, f, z != 0)
		})
	}
}

func TestSymmetricLDLTRandomCongruences(t *testing.T) {
	rng := rand.New(rand.NewSource(37))
	twoBlocks, permutations := 0, 0
	for n := 1; n <= 14; n++ {
		for trial := 0; trial < 8; trial++ {
			u := make([][]float64, n)
			d := make([]float64, n)
			positive := 0
			for i := range u {
				u[i] = make([]float64, n)
				u[i][i] = 1.5 + rng.Float64()
				for j := i + 1; j < n; j++ {
					u[i][j] = rng.Float64() - .5
				}
				d[i] = .2 + 3*rng.Float64()
				if rng.Intn(2) == 0 {
					d[i] = -d[i]
				} else {
					positive++
				}
			}
			// Orthogonal column rotations keep U nonsingular and preserve the
			// known inertia of U^T D U, while forcing varied pivot decisions.
			for rotation := 0; n > 1 && rotation < 3*n; rotation++ {
				i, j := rng.Intn(n), rng.Intn(n-1)
				if j >= i {
					j++
				}
				angle := 2 * math.Pi * rng.Float64()
				s, c := math.Sincos(angle)
				for k := range u {
					x, y := u[k][i], u[k][j]
					u[k][i], u[k][j] = c*x-s*y, s*x+c*y
				}
			}
			a := make([][]float64, n)
			for i := range a {
				a[i] = make([]float64, n)
				for j := range a[i] {
					for k := 0; k < n; k++ {
						a[i][j] += u[k][i] * d[k] * u[k][j]
					}
				}
			}
			f, err := symmetricTestCOO(a).SymmetricLDLT(0, make([]bool, n), nil)
			if err != nil {
				t.Fatal(err)
			}
			p, z, neg := f.Inertia()
			if p != positive || z != 0 || neg != n-positive {
				t.Fatalf("n=%d trial=%d: inertia %d,%d,%d", n, trial, p, z, neg)
			}
			ip, iz, in, err := symmetricTestCOO(a).SymmetricInertia(nil, 0, 0, make([]bool, n), nil)
			if err != nil || ip != p || iz != z || in != neg {
				t.Fatalf("n=%d trial=%d: inertia-only path differs: %d,%d,%d, %v", n, trial, ip, iz, in, err)
			}
			for _, size := range f.BlockSizes() {
				if size == 2 {
					twoBlocks++
				}
			}
			for i, p := range f.permutation {
				if i != p {
					permutations++
				}
			}
			checkSymmetricFactor(t, a, f, false)
		}
	}
	if twoBlocks == 0 || permutations == 0 {
		t.Fatal("random congruences did not exercise 2x2 pivots and permutations")
	}
	t.Logf("112 known-inertia congruences: %d two-by-two blocks, %d permuted indices", twoBlocks, permutations)
}

func TestSymmetricLDLTScalingAndConstraints(t *testing.T) {
	for _, scale := range []float64{1e-150, 1, 1e150} {
		a := [][]float64{{.1 * scale, 2 * scale}, {2 * scale, -.2 * scale}}
		c := NewCOOMatrix(3)
		c.Set(0, 0, -100)
		// Only upper entries are required, matching the documented API.
		for i := 0; i < 2; i++ {
			for j := i; j < 2; j++ {
				c.Set(i+1, j+1, a[i][j])
			}
		}
		f, err := c.SymmetricLDLT(1, []bool{true, false, false}, nil)
		if err != nil {
			t.Fatal(err)
		}
		p, z, n := f.Inertia()
		if p != 1 || z != 0 || n != 1 {
			t.Fatal("wrong scaled-block inertia")
		}
		checkSymmetricFactor(t, a, f, false)
	}
}

func TestSymmetricTwoByTwoInertia(t *testing.T) {
	for _, tt := range []struct {
		a, b, c float64
		want    [3]int
	}{
		{0, 1, 0, [3]int{1, 0, 1}},
		{2, 1, 2, [3]int{2, 0, 0}},
		{-2, 1, -2, [3]int{0, 0, 2}},
		{1, 1, 1, [3]int{1, 1, 0}},
		{-1, 1, -1, [3]int{0, 1, 1}},
		{0, 0, 0, [3]int{0, 2, 0}},
		{1e300, 1e300, -1e300, [3]int{1, 0, 1}},
		{1e-300, 1e-300, -1e-300, [3]int{1, 0, 1}},
	} {
		f := &SymLDLTFactor{Size: 2, blocks: []symmetricBlock{{start: 0, size: 2, a: tt.a, b: tt.b, c: tt.c}}}
		p, z, n := f.Inertia()
		if got := [3]int{p, z, n}; got != tt.want {
			t.Fatalf("block (%g,%g,%g): inertia %v, want %v", tt.a, tt.b, tt.c, got, tt.want)
		}
	}
}

func TestSymmetricLDLTInvalidInputsAndLegacySolve(t *testing.T) {
	c := symmetricTestCOO([][]float64{{4, 1}, {1, 3}})
	if _, err := c.SymmetricLDLT(1, []bool{false, false}, nil); err == nil {
		t.Fatal("accepted wrong constraint count")
	}
	c.Set(0, 0, math.NaN())
	if _, err := c.SymmetricLDLT(0, []bool{false, false}, nil); err == nil {
		t.Fatal("accepted NaN")
	}
	c.Set(0, 0, 4)
	rhs := []float64{6, 7}
	legacy, err := c.ToLLS(0, []bool{false, false}).Solve(nil, rhs)
	if err != nil {
		t.Fatal(err)
	}
	f, err := c.SymmetricLDLT(0, []bool{false, false}, nil)
	if err != nil {
		t.Fatal(err)
	}
	x, err := f.Solve(rhs)
	if err != nil || math.Abs(x[0]-1) > 1e-14 || math.Abs(x[1]-2) > 1e-14 {
		t.Fatalf("new solver: x=%v, err=%v", x, err)
	}
	for i := range x {
		if math.Abs(x[i]-legacy[0][i]) > 1e-14 {
			t.Fatal("static solver regression")
		}
	}
	if _, err := f.Solve([]float64{1}); err == nil {
		t.Fatal("accepted wrong RHS size")
	}
}
