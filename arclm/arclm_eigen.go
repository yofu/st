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
	return inversePencilCSRProgressChecked(ke, kg, metric, C, target, eps, locked, klocked, progress, nil)
}

func inversePencilCSRProgressChecked(ke, kg, metric *matrix.SymmetricCSR, C *matrix.SymLDLTFactor, target, eps float64, locked, klocked [][]float64, progress func(int, float64, float64), check func() error) (float64, []float64, error) {
	return inversePencilCSRProgressQuality(ke, kg, metric, C, target, eps, locked, klocked, progress, check, nil)
}

// quality optionally supplies an operator-scaled residual for automatic
// damping-period estimates on models with very different DOF stiffness scales.
// The default quality uses the relative force residual.
func inversePencilCSRProgressQuality(ke, kg, metric *matrix.SymmetricCSR, C *matrix.SymLDLTFactor, target, eps float64, locked, klocked [][]float64, progress func(int, float64, float64), check func() error, quality func(float64, []float64, []float64, []float64) (float64, float64, error)) (float64, []float64, error) {
	if check != nil {
		if err := check(); err != nil {
			return 0, nil, err
		}
	}
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
		if check != nil {
			if err := check(); err != nil {
				return 0, nil, err
			}
		}
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
		residualTolerance := bucklingVectorTolerance
		if quality == nil {
			residual = pencilForceResidual(mu, kx, gx)
		} else {
			residual, residualTolerance, err = quality(mu, x, kx, gx)
			if err != nil {
				return 0, nil, err
			}
		}
		orthogonality := math.Abs(Dot(x, metricX, size) - 1)
		for _, kq := range klocked {
			orthogonality = math.Max(orthogonality, math.Abs(Dot(x, kq, size)))
		}
		if progress != nil {
			progress(iteration+1, residual, orthogonality)
		}
		if residual <= residualTolerance && orthogonality <= bucklingVectorTolerance {
			convergedPasses++
			// A lightly contaminated vector in a tight cluster can leave a
			// residual floor in the next M-deflated mode. Refine two more
			// times before locking it; rotations within a repeated eigenspace
			// are allowed and no vector-change stopping test is imposed.
			if metric != ke && convergedPasses < 3 {
				continue
			}
			// Deflation must not silently move the answer to a different root.
			rootTolerance := bucklingVectorTolerance
			if metric == ke {
				rootTolerance = residualTolerance
			}
			if math.Abs(mu-target) > math.Max(4*eps, 8*rootTolerance*math.Abs(target)) {
				return 0, nil, fmt.Errorf("buckling inverse iteration converged outside target: %g vs %g", mu, target)
			}
			return mu, x, nil
		}
		convergedPasses = 0
	}
	return 0, nil, fmt.Errorf("buckling inverse iteration did not converge in 100 iterations (residual %.3g)", residual)
}

// pencilForceResidual is also the relative residual of KE*x + Lambda*KG*x = 0.
func pencilForceResidual(mu float64, kx, gx []float64) float64 {
	rnorm, knorm, gnorm := 0.0, 0.0, 0.0
	for j := range kx {
		rnorm = math.Hypot(rnorm, gx[j]+mu*kx[j])
		knorm = math.Hypot(knorm, kx[j])
		gnorm = math.Hypot(gnorm, gx[j])
	}
	return rnorm / (gnorm + math.Abs(mu)*knorm)
}

// inverseVector keeps the ordinary force check for well-scaled modes. If
// force cancellation prevents that check from passing, use the elastic
// energy norm with its own tolerance and a separate force residual ceiling.
func (s *bucklingSearch) inverseVector(c *matrix.SymLDLTFactor, target, eps float64, locked, klocked [][]float64, rank int, progress func(int, float64, float64)) (float64, []float64, error) {
	const forceCeiling = 1e-6
	const energyTolerance = 1e-8
	iterations := 0
	usedEnergy := false
	quality := func(mu float64, x, kx, gx []float64) (float64, float64, error) {
		iterations++
		usedEnergy = false
		force := pencilForceResidual(mu, kx, gx)
		if iterations < 4 || !finitePlastic(force) || force <= bucklingVectorTolerance || force > forceCeiling {
			return force, bucklingVectorTolerance, nil
		}
		if s.elasticFactor == nil {
			factor, err := s.ke.SymmetricLDLT(s.csize, s.conf, nil)
			if err != nil {
				return 0, energyTolerance, fmt.Errorf("buckling energy residual factor: %w", err)
			}
			s.elasticFactor = factor
			s.solveFactorizations++
		}
		usedEnergy = true
		energy, err := pencilEnergyResidual(s.elasticFactor, mu, x, kx, gx)
		if s.diagnostics != nil {
			s.diagnostics.log("buckling_energy_residual", map[string]interface{}{"rank": rank, "iteration": iterations, "force_residual": force, "energy_residual": energy})
		}
		return energy, energyTolerance, err
	}
	mu, x, err := inversePencilCSRProgressQuality(s.elastic, s.geometric, s.elastic, c, target, eps, locked, klocked, progress, nil, quality)
	if err != nil || !usedEnergy {
		return mu, x, err
	}
	// Recheck the rank around the accepted quotient. This guards against a
	// residual near a different mode when the force check was insufficient.
	margin := math.Max(4*eps, 8*energyTolerance*mu)
	if margin >= mu {
		return 0, nil, fmt.Errorf("buckling mode %d: residual interval reaches zero", rank)
	}
	_, below, err := s.countNudged(mu-margin, 0)
	if err != nil {
		return 0, nil, err
	}
	_, above, err := s.countNudged(mu+margin, math.Inf(1))
	if err != nil {
		return 0, nil, err
	}
	if below < rank || above >= rank {
		return 0, nil, fmt.Errorf("buckling mode %d: residual-checked root has inconsistent inertia (%d,%d)", rank, below, above)
	}
	return mu, x, nil
}

// For K>0, sqrt(r^T K^-1 r)/(mu*sqrt(x^T K x)), r=mu*K*x+G*x,
// is the relative residual of the self-adjoint -K^-1*G problem in the K norm.
// It bounds the distance to the spectrum in mu and avoids judging massless,
// very stiff DOFs against tiny inertial forces. The original force residual is
// still computed and logged, rather than relaxing the existing eigen solver.
func pencilEnergyResidual(k *matrix.SymLDLTFactor, mu float64, x, kx, gx []float64) (float64, error) {
	r := make([]float64, len(x))
	for i := range r {
		r[i] = gx[i] + mu*kx[i]
	}
	z, err := k.Solve(r)
	if err != nil {
		return 0, err
	}
	energy, norm := Dot(r, z, len(r)), Dot(x, kx, len(x))
	if energy < 0 || norm <= 0 {
		return 0, fmt.Errorf("invalid modal energy residual")
	}
	value := math.Sqrt(energy/norm) / math.Abs(mu)
	if !finitePlastic(value) {
		return 0, fmt.Errorf("non-finite modal energy residual")
	}
	return value, nil
}
