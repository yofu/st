package matrix

import (
	"fmt"
	"math"
	"time"
)

type symmetricEntry struct {
	column int
	value  float64
}

type symmetricBlock struct {
	start, size int
	a, b, c     float64
}

// SymLDLTFactor represents A[p,p] = L D L^T, with unit lower triangular L
// and symmetric 1x1/2x2 blocks in D. It owns its storage and can be reused by
// Solve without mutation. It is separate from the static-analysis LLS solver.
type SymLDLTFactor struct {
	Size        int
	permutation []int
	lower       [][]symmetricEntry
	blocks      []symmetricBlock
	inertiaOnly bool
}

func finiteSym(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// SymmetricLDLT applies unblocked Bunch-Kaufman symmetric diagonal pivoting.
// Only the upper triangle of co is used, matching ToLLS. Constraints are
// removed before factorization. The input is not modified. Schur complements
// and L use sparse storage, retaining all fill without a numerical drop rule.
// Singular factors are returned for inertia queries; Solve rejects them.
// progress has the same one-notification-per-DOF convention as LLSMatrix.LDLT.
func (co *COOMatrix) SymmetricLDLT(csize int, conf []bool, progress chan int) (*SymLDLTFactor, error) {
	return co.symmetricLDLT(nil, 0, csize, conf, progress, false, nil)
}

// SymmetricLDLTShift factors co + shift*other without allocating an intermediate
// COO matrix. The same upper-triangle convention as SymmetricLDLT applies.
func (co *COOMatrix) SymmetricLDLTShift(other *COOMatrix, shift float64, csize int, conf []bool, progress chan int) (*SymLDLTFactor, error) {
	return co.symmetricLDLT(other, shift, csize, conf, progress, false, nil)
}

// SymmetricInertia computes only the inertia of co + shift*other, omitting L
// storage and solve preparation. Passing nil for other factors co alone.
func (co *COOMatrix) SymmetricInertia(other *COOMatrix, shift float64, csize int, conf []bool, progress chan int) (positive, zero, negative int, err error) {
	f, err := co.symmetricLDLT(other, shift, csize, conf, progress, true, nil)
	if err != nil {
		return 0, 0, 0, err
	}
	positive, zero, negative = f.Inertia()
	return
}

func (co *COOMatrix) symmetricLDLT(other *COOMatrix, shift float64, csize int, conf []bool, progress chan int, inertiaOnly bool, stats *SymLDLTStatistics) (*SymLDLTFactor, error) {
	start := time.Now()
	plan, err := NewSymLDLTPlan(co, other, csize, conf)
	if err != nil {
		return nil, err
	}
	build := time.Since(start).Seconds()
	f, err := plan.factor(shift, progress, inertiaOnly, stats)
	if stats != nil {
		stats.BuildSeconds += build
	}
	return f, err
}

// scaledSymmetricBlock avoids overflow in the determinant/sign calculation.
func scaledSymmetricBlock(b symmetricBlock) (a, off, c, scale, det float64) {
	scale = math.Max(math.Abs(b.a), math.Max(math.Abs(b.b), math.Abs(b.c)))
	if scale == 0 {
		return
	}
	a, off, c = b.a/scale, b.b/scale, b.c/scale
	det = math.FMA(a, c, -off*off)
	return
}

func solveSymmetricBlock(b symmetricBlock, r, s float64) (float64, float64, error) {
	a, off, c, scale, det := scaledSymmetricBlock(b)
	if scale == 0 || det == 0 {
		return 0, 0, fmt.Errorf("singular 2x2 block")
	}
	r, s = r/scale, s/scale
	return (c*r - off*s) / det, (a*s - off*r) / det, nil
}

// Inertia counts the signs of the eigenvalues of D, including its 2x2 blocks.
// No arbitrary small-pivot cutoff is used: a cutoff would alter the root count.
func (f *SymLDLTFactor) Inertia() (positive, zero, negative int) {
	count := func(v float64) {
		if v > 0 {
			positive++
		} else if v < 0 {
			negative++
		} else {
			zero++
		}
	}
	for _, b := range f.blocks {
		if b.size == 1 {
			count(b.a)
			continue
		}
		a, _, c, _, det := scaledSymmetricBlock(b)
		if det < 0 {
			positive++
			negative++
		} else if det > 0 {
			count(a + c)
			count(a + c)
		} else {
			zero++
			count(a + c)
		}
	}
	return
}

// BlockSizes returns a copy of the 1x1/2x2 pivot-block structure.
func (f *SymLDLTFactor) BlockSizes() []int {
	sizes := make([]int, len(f.blocks))
	for i, b := range f.blocks {
		sizes[i] = b.size
	}
	return sizes
}

// Solve applies the permutation, L, block diagonal D and L^T solves. Neither
// the factor nor rhs is modified; an exactly singular factor is rejected.
func (f *SymLDLTFactor) Solve(rhs []float64) ([]float64, error) {
	if f.inertiaOnly {
		return nil, fmt.Errorf("inertia-only factor cannot solve")
	}
	if len(rhs) != f.Size {
		return nil, fmt.Errorf("symmetric LDLT RHS size mismatch")
	}
	x := make([]float64, f.Size)
	for i, p := range f.permutation {
		if !finiteSym(rhs[p]) {
			return nil, fmt.Errorf("non-finite symmetric LDLT RHS")
		}
		x[i] = rhs[p]
		for _, e := range f.lower[i] {
			x[i] -= e.value * x[e.column]
		}
	}
	for _, b := range f.blocks {
		k := b.start
		if b.size == 1 {
			if b.a == 0 {
				return nil, fmt.Errorf("singular symmetric LDLT block %d", k)
			}
			x[k] /= b.a
		} else {
			var err error
			x[k], x[k+1], err = solveSymmetricBlock(b, x[k], x[k+1])
			if err != nil {
				return nil, err
			}
		}
	}
	for i := f.Size - 1; i >= 0; i-- {
		for _, e := range f.lower[i] {
			x[e.column] -= e.value * x[i]
		}
	}
	answer := make([]float64, f.Size)
	for i, p := range f.permutation {
		if !finiteSym(x[i]) {
			return nil, fmt.Errorf("non-finite symmetric LDLT solution")
		}
		answer[p] = x[i]
	}
	return answer, nil
}
