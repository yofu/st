package arclm

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/yofu/st/matrix"
)

const bucklingVectorTolerance = 1e-10

// inverseBuckling uses a previously factored nonsingular shift of KG + sigma*KE.
// All vectors use free DOFs. locked must be KE-orthonormal. The result retains
// KE normalization (including when expanded to full DOFs by the caller).
func inverseBuckling(ke, kg *matrix.COOMatrix, csize int, conf []bool, C *matrix.SymLDLTFactor, target, eps float64, locked [][]float64) (float64, []float64, error) {
	elastic, err := ke.ToSymmetricCSR(csize, conf)
	if err != nil {
		return 0, nil, err
	}
	geometric, err := kg.ToSymmetricCSR(csize, conf)
	if err != nil {
		return 0, nil, err
	}
	klocked := make([][]float64, len(locked))
	for j, q := range locked {
		if len(q) != elastic.Size {
			return 0, nil, fmt.Errorf("buckling locked vector %d has invalid size", j)
		}
		klocked[j] = elastic.MulV(q)
	}
	return inverseBucklingCSR(elastic, geometric, C, target, eps, locked, klocked)
}

// inverseBucklingCSR reuses compressed operators and KE*locked between modes.
func inverseBucklingCSR(ke, kg *matrix.SymmetricCSR, C *matrix.SymLDLTFactor, target, eps float64, locked, klocked [][]float64) (float64, []float64, error) {
	return inverseBucklingCSRProgress(ke, kg, C, target, eps, locked, klocked, nil)
}

func inverseBucklingCSRProgress(ke, kg *matrix.SymmetricCSR, C *matrix.SymLDLTFactor, target, eps float64, locked, klocked [][]float64, progress func(int, float64, float64)) (float64, []float64, error) {
	return inversePencilCSRProgress(ke, kg, ke, C, target, eps, locked, klocked, progress)
}

// inversePencilCSRProgress factors kg+sigma*ke but allows an independent
// orthogonality metric: KE for buckling and M for finite vibration modes.
// The RHS remains KE*x, irrespective of the normalization metric.
func inversePencilCSRProgress(ke, kg, metric *matrix.SymmetricCSR, C *matrix.SymLDLTFactor, target, eps float64, locked, klocked [][]float64, progress func(int, float64, float64)) (float64, []float64, error) {
	if ke == nil || kg == nil || metric == nil || metric.Size != ke.Size {
		return 0, nil, fmt.Errorf("invalid inverse iteration operators or metric")
	}

	size := ke.Size
	if C == nil || C.Size != size || size < 1 || len(locked) >= size || target <= 0 || math.IsNaN(target) || math.IsInf(target, 0) || eps <= 0 || math.IsNaN(eps) || math.IsInf(eps, 0) {
		return 0, nil, fmt.Errorf("invalid buckling inverse iteration parameters")
	}
	if kg.Size != size || len(klocked) != len(locked) {
		return 0, nil, fmt.Errorf("invalid buckling cached operators")
	}
	for j, q := range locked {
		if len(q) != size || len(klocked[j]) != size {
			return 0, nil, fmt.Errorf("buckling locked vector %d has invalid size", j)
		}
	}
	orthonormalize := func(y []float64) ([]float64, error) {
		// Scale before dot products to avoid overflow near a singular shift.
		scale := 0.0
		for _, v := range y {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return nil, fmt.Errorf("non-finite buckling inverse iterate")
			}
			scale = math.Max(scale, math.Abs(v))
		}
		if scale == 0 {
			return nil, fmt.Errorf("zero buckling inverse iterate")
		}
		for j := range y {
			y[j] /= scale
		}
		// Two passes of modified Gram-Schmidt in the selected inner product.
		for pass := 0; pass < 2; pass++ {
			for j, q := range locked {
				alpha := Dot(klocked[j], y, size)
				for k := range y {
					y[k] -= alpha * q[k]
				}
			}
		}
		ky := metric.MulV(y)
		norm2 := Dot(y, ky, size)
		if norm2 <= 0 || math.IsNaN(norm2) || math.IsInf(norm2, 0) {
			return nil, fmt.Errorf("inverse iterate has no positive metric norm after deflation")
		}
		norm := math.Sqrt(norm2)
		for j := range y {
			y[j] /= norm
			ky[j] /= norm
		}
		return ky, nil
	}
	// A reproducible local seed avoids dependence on the global random source.
	rng := rand.New(rand.NewSource(int64(len(locked) + 1)))
	x := make([]float64, size)
	for j := range x {
		x[j] = 2*rng.Float64() - 1
	}
	metricX, err := orthonormalize(x)
	if err != nil {
		return 0, nil, err
	}
	kx := metricX
	if metric != ke {
		kx = ke.MulV(x)
	}
	residual := math.Inf(1)
	convergedPasses := 0
	for iteration := 0; iteration < 100; iteration++ {
		// (KG + sigma*KE) y = KE*x, with the factor reused each iteration.
		y, err := C.Solve(kx)
		if err != nil {
			return 0, nil, fmt.Errorf("buckling inverse solve: %w", err)
		}
		metricX, err = orthonormalize(y)
		if err != nil {
			return 0, nil, err
		}
		x = y
		kx = metricX
		if metric != ke {
			kx = ke.MulV(x)
		}
		gx := kg.MulV(x)
		mu := -Dot(x, gx, size) / Dot(x, kx, size)
		if mu <= 0 || math.IsNaN(mu) || math.IsInf(mu, 0) {
			return 0, nil, fmt.Errorf("invalid buckling Rayleigh quotient: %g", mu)
		}
		// This is also the relative residual of KE*x + Lambda*KG*x = 0.
		rnorm, knorm, gnorm := 0.0, 0.0, 0.0
		for j := range x {
			rnorm = math.Hypot(rnorm, gx[j]+mu*kx[j])
			knorm = math.Hypot(knorm, kx[j])
			gnorm = math.Hypot(gnorm, gx[j])
		}
		residual = rnorm / (gnorm + math.Abs(mu)*knorm)
		orthogonality := math.Abs(Dot(x, metricX, size) - 1)
		for _, kq := range klocked {
			orthogonality = math.Max(orthogonality, math.Abs(Dot(x, kq, size)))
		}
		if progress != nil {
			progress(iteration+1, residual, orthogonality)
		}
		if residual <= bucklingVectorTolerance && orthogonality <= bucklingVectorTolerance {
			convergedPasses++
			// A lightly contaminated vector in a tight cluster can leave a
			// residual floor in the next M-deflated mode. Refine two more
			// times before locking it; rotations within a repeated eigenspace
			// are allowed and no vector-change stopping test is imposed.
			if metric != ke && convergedPasses < 3 {
				continue
			}
			// Deflation must not silently move the answer to a different root.
			if math.Abs(mu-target) > math.Max(4*eps, 8*bucklingVectorTolerance*math.Abs(target)) {
				return 0, nil, fmt.Errorf("buckling inverse iteration converged outside target: %g vs %g", mu, target)
			}
			return mu, x, nil
		}
		convergedPasses = 0
	}
	return 0, nil, fmt.Errorf("buckling inverse iteration did not converge in 100 iterations (residual %.3g)", residual)
}
