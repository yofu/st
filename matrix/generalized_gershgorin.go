package matrix

import (
	"fmt"
	"math"
)

// GeneralizedGershgorinBounds encloses the real eigenvalues of scaleA*A x=mu*B x.
// Available is false if this sufficient diagonal-dominance condition cannot
// provide a finite enclosure; that says nothing about whether B is SPD.
type GeneralizedGershgorinBounds struct {
	Available       bool    `json:"available"`
	Lower           float64 `json:"lower_mu"`
	Upper           float64 `json:"upper_mu"`
	NonDominantRows int     `json:"non_dominant_rows"`
	Reason          string  `json:"reason,omitempty"`
}

// PositiveReciprocalBounds converts mu to Lambda=1/mu for positive roots.
// A hull crossing zero gives no finite upper Lambda bound. ok=false means
// there is no available enclosure for positive reciprocal eigenvalues.
func (g GeneralizedGershgorinBounds) PositiveReciprocalBounds() (lower, upper float64, ok bool) {
	if !g.Available || g.Upper <= 0 {
		return 0, 0, false
	}
	lower = math.Max(0, math.Nextafter(1/g.Upper, math.Inf(-1)))
	upper = math.Inf(1)
	if g.Lower > 0 {
		upper = math.Nextafter(1/g.Lower, math.Inf(1))
	}
	return lower, upper, true
}

// GeneralizedGershgorin uses O(nnz(A)+nnz(B)) work and O(n) extra storage.
// For a maximal component of an eigenvector,
//
//	|scaleA*a_ii - mu*b_ii| <= R_Ai + |mu|*R_Bi.
//
// When b_ii>R_Bi, every real mu therefore belongs to a quotient interval
// [scaleA*a_ii-R_Ai, scaleA*a_ii+R_Ai] / [b_ii-R_Bi,b_ii+R_Bi]
// for at least one row. The hull of these row intervals contains the spectrum.
// The caller must establish that its pencil has real eigenvalues (e.g. A
// symmetric and B SPD). This does not enclose arbitrary complex eigenvalues.
// Row sums and interval endpoints are rounded outwards; callers should still
// verify proposed search endpoints using inertia near singular shifts.
func (a *SymmetricCSR) GeneralizedGershgorin(b *SymmetricCSR, scaleA float64) (GeneralizedGershgorinBounds, error) {
	result := GeneralizedGershgorinBounds{}
	if a == nil || b == nil || a.Size != b.Size || a.Size == 0 || !finiteSym(scaleA) {
		return result, fmt.Errorf("invalid generalized Gershgorin dimensions or scale")
	}
	n := a.Size
	diagA, diagB, ra, rb := make([]float64, n), make([]float64, n), make([]float64, n), make([]float64, n)
	up := func(x float64) float64 { return math.Nextafter(x, math.Inf(1)) }
	down := func(x float64) float64 { return math.Nextafter(x, math.Inf(-1)) }
	for i := 0; i < n; i++ {
		for j := a.offsets[i]; j < a.offsets[i+1]; j++ {
			value := a.values[j]
			if a.columns[j] == i {
				diagA[i] = scaleA * value
			} else if value != 0 && scaleA != 0 {
				ra[i] = up(ra[i] + up(math.Abs(scaleA)*math.Abs(value)))
			}
		}
		for j := b.offsets[i]; j < b.offsets[i+1]; j++ {
			value := b.values[j]
			if b.columns[j] == i {
				diagB[i] = value
			} else if value != 0 {
				rb[i] = up(rb[i] + math.Abs(value))
			}
		}
		if !(diagB[i] > rb[i]) {
			result.NonDominantRows++
		}
	}
	if result.NonDominantRows != 0 {
		result.Reason = "B is not strictly row diagonally dominant with positive diagonal"
		return result, nil
	}
	lower, upper := math.Inf(1), math.Inf(-1)
	for i := 0; i < n; i++ {
		// Include multiplication rounding of scaleA*a_ii as well as sums.
		nlo, nhi := down(down(diagA[i])-ra[i]), up(up(diagA[i])+ra[i])
		dlo, dhi := down(diagB[i]-rb[i]), up(diagB[i]+rb[i])
		if !finiteSym(nlo) || !finiteSym(nhi) || !finiteSym(dlo) || !finiteSym(dhi) || dlo <= 0 {
			result.Reason = "overflow or insufficient diagonal-dominance margin"
			return result, nil
		}
		for _, num := range []float64{nlo, nhi} {
			for _, den := range []float64{dlo, dhi} {
				q := num / den
				lower = math.Min(lower, down(q))
				upper = math.Max(upper, up(q))
			}
		}
	}
	if !finiteSym(lower) || !finiteSym(upper) {
		result.Reason = "non-finite generalized Gershgorin enclosure"
		return result, nil
	}
	result.Available, result.Lower, result.Upper = true, lower, upper
	return result, nil
}
