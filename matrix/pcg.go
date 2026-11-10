package matrix

import (
	"context"
	"fmt"
	"math"
)

// PCG solves each right-hand side using a shared incomplete LDLT factor of
// the supplied preconditioner. rtol is relative to the Euclidean RHS norm.
// The coefficient matrix and preconditioner are left intact.
func (ll *LLSMatrix) PCG(ctx context.Context, preconditioner *LLSMatrix, rtol float64, maxIterations int, rhs ...[]float64) ([][]float64, error) {
	if preconditioner == nil || preconditioner.Size != ll.Size {
		return nil, fmt.Errorf("PCG: preconditioner size mismatch")
	}
	if !positiveFinite(rtol) || rtol >= 1 || maxIterations < 1 {
		return nil, fmt.Errorf("PCG: invalid tolerance or iteration limit")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	for i, b := range rhs {
		if len(b) != ll.Size {
			return nil, fmt.Errorf("PCG: right-hand side %d has size %d, want %d", i+1, len(b), ll.Size)
		}
	}
	factor := preconditioner.Copy()
	if err := factor.incompleteLDLT(ctx); err != nil {
		return nil, err
	}
	answers := make([][]float64, len(rhs))
	for i, b := range rhs {
		x, err := ll.pcg(ctx, factor, b, rtol, maxIterations)
		if err != nil {
			return nil, fmt.Errorf("PCG: right-hand side %d: %w", i+1, err)
		}
		answers[i] = x
	}
	return answers, nil
}

func (ll *LLSMatrix) pcg(ctx context.Context, factor *LLSMatrix, b []float64, rtol float64, maxIterations int) ([]float64, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	bnorm := pcgNorm(b)
	if math.IsNaN(bnorm) || math.IsInf(bnorm, 0) {
		return nil, fmt.Errorf("non-finite right-hand side")
	}
	x := make([]float64, ll.Size)
	if bnorm == 0 {
		return x, nil
	}
	r := append([]float64(nil), b...)
	z := make([]float64, ll.Size)
	p := make([]float64, ll.Size)
	q := make([]float64, ll.Size)
	factor.applyPCGPreconditioner(z, r)
	copy(p, z)
	rho := Dot(r, z, ll.Size)
	residual := 1.0
	for iteration := 1; iteration <= maxIterations; iteration++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !positiveFinite(rho) {
			return nil, fmt.Errorf("non-positive or non-finite r·z at iteration %d: %g", iteration, rho)
		}
		ll.mulSymmetricInto(q, p)
		curvature := Dot(p, q, ll.Size)
		if !positiveFinite(curvature) {
			return nil, fmt.Errorf("non-positive or non-finite p·Ap at iteration %d: %g", iteration, curvature)
		}
		alpha := rho / curvature
		if !positiveFinite(alpha) {
			return nil, fmt.Errorf("invalid step length at iteration %d: %g", iteration, alpha)
		}
		for i := range x {
			x[i] += alpha * p[i]
			r[i] -= alpha * q[i]
		}

		residual = pcgNorm(r) / bnorm
		restart := residual <= rtol || iteration == maxIterations
		if restart {
			// Check the actual residual before accepting a solution. If the
			// recurrence drifted, restart the direction from that residual.
			ll.mulSymmetricInto(q, x)
			for i := range r {
				r[i] = b[i] - q[i]
			}
			residual = pcgNorm(r) / bnorm
			if residual <= rtol {
				return x, ctx.Err()
			}
		}
		if math.IsNaN(residual) || math.IsInf(residual, 0) {
			return nil, fmt.Errorf("non-finite residual at iteration %d", iteration)
		}
		if iteration == maxIterations {
			break
		}
		factor.applyPCGPreconditioner(z, r)
		nextRho := Dot(r, z, ll.Size)
		if restart {
			copy(p, z)
		} else {
			beta := nextRho / rho
			for i := range p {
				p[i] = z[i] + beta*p[i]
			}
		}
		rho = nextRho
	}
	return nil, fmt.Errorf("did not converge in %d iterations (relative residual %.3g, tolerance %.3g)", maxIterations, residual, rtol)
}

// Both operations traverse the stored triangle and do not require DiagUp.
func (ll *LLSMatrix) mulSymmetricInto(dst, x []float64) {
	clear(dst)
	for col, diag := range ll.diag {
		dst[col] += diag.value * x[col]
		for n := diag.down; n != nil; n = n.down {
			dst[n.row] += n.value * x[col]
			dst[col] += n.value * x[n.row]
		}
	}
}

func (ll *LLSMatrix) applyPCGPreconditioner(dst, r []float64) {
	copy(dst, r)
	for col, diag := range ll.diag {
		for n := diag.down; n != nil; n = n.down {
			dst[n.row] -= n.value * dst[col]
		}
	}
	for i, diag := range ll.diag {
		dst[i] /= diag.value
	}
	ll.BSUpper(dst)
}

func positiveFinite(v float64) bool {
	return v > 0 && !math.IsInf(v, 0)
}

func pcgNorm(x []float64) float64 {
	var norm float64
	for _, v := range x {
		norm = math.Hypot(norm, v)
	}
	return norm
}
