package arclm

import (
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/yofu/st/matrix"
)

type bucklingInertiaSample struct {
	mu       float64
	negative int
}

var errBucklingSingularShift = errors.New("singular buckling shift")

// Counted shifts are shared across ranks and arbitrary proposed endpoints.
// Only the most recent final solve factor is retained.
type bucklingSearch struct {
	ke, kg             *matrix.COOMatrix
	plan               *matrix.SymLDLTPlan
	elastic, geometric *matrix.SymmetricCSR
	csize              int
	conf               []bool
	eps, right         float64
	lower              float64
	gershgorin         matrix.GeneralizedGershgorinBounds
	pivot              chan int
	samples            []bucklingInertiaSample
	factor             *matrix.SymLDLTFactor
	factorMu           float64
	diagnostics        *bucklingDiagnostics
	rank               int

	inertiaEvaluations, solveFactorizations, cacheHits             int
	bracketFactorizations, refinementAttempts, refinementsAccepted int
}

func newBucklingSearch(ke, kg *matrix.COOMatrix, csize int, conf []bool, eps, right float64, pivot chan int) (*bucklingSearch, error) {
	return newBucklingSearchDiagnostics(ke, kg, csize, conf, eps, right, pivot, nil)
}

func newBucklingSearchDiagnostics(ke, kg *matrix.COOMatrix, csize int, conf []bool, eps, right float64, pivot chan int, d *bucklingDiagnostics) (*bucklingSearch, error) {
	if ke == nil || kg == nil || ke.Size != kg.Size || eps <= 0 || math.IsNaN(eps) || math.IsInf(eps, 0) || right <= 0 || math.IsNaN(right) || math.IsInf(right, 0) {
		return nil, fmt.Errorf("invalid buckling search parameters")
	}
	s := &bucklingSearch{ke: ke, kg: kg, csize: csize, conf: append([]bool(nil), conf...), eps: eps, right: right, pivot: pivot}
	s.diagnostics = d
	var positive int
	var err error
	if d == nil {
		s.plan, err = matrix.NewSymLDLTPlan(kg, ke, csize, conf)
		if err != nil {
			return nil, err
		}
		positive, _, _, err = ke.SymmetricInertia(nil, 0, csize, conf, pivot)
	} else {
		_, inertia, e := d.factor("elastic_check", ke, nil, 0, csize, conf, 0, true, false)
		positive, err = inertia.Positive, e
	}
	s.inertiaEvaluations++
	if err != nil {
		return nil, fmt.Errorf("buckling KE positive-definiteness check: %w", err)
	}
	if positive != ke.Size-csize {
		return nil, fmt.Errorf("buckling KE is not positive definite")
	}
	// -right is now an initial scale rather than a hard spectral cutoff.
	// Coordinate Rayleigh quotients provide only a proposal, never a bound.
	requestedRight := right
	proposal := 0.0
	for i, fixed := range conf {
		if !fixed {
			ratio := -kg.Query(i, i) / ke.Query(i, i)
			if ratio > proposal && !math.IsInf(ratio, 0) {
				proposal = ratio
			}
		}
	}
	if proposal > 0 && proposal <= math.MaxFloat64/2 {
		right = 2 * proposal
	}
	expansions := 0
	for {
		mu, negative, e := s.countNudged(right, math.Inf(1))
		if e != nil {
			return nil, fmt.Errorf("buckling automatic upper bound %.16g: %w", right, e)
		}
		right = mu
		if negative == 0 {
			s.right = right
			break
		}
		if right > math.MaxFloat64/2 {
			return nil, fmt.Errorf("cannot represent a finite buckling upper bound")
		}
		right *= 2
		expansions++
	}
	if d != nil {
		d.log("automatic_upper_bound", map[string]interface{}{"requested_right_mu": requestedRight, "coordinate_proposal_mu": proposal, "verified_upper_mu": s.right, "expansions": expansions})
	}
	s.elastic, err = ke.ToSymmetricCSR(csize, conf)
	if err != nil {
		return nil, err
	}
	s.geometric, err = kg.ToSymmetricCSR(csize, conf)
	if err != nil {
		return nil, err
	}
	s.tightenGershgorin()
	return s, nil
}

// Gershgorin is an optional proposal; inertia validates all endpoints.
// The configured -right is an initial scale, and can be expanded automatically.
func (s *bucklingSearch) tightenGershgorin() {
	bounds, err := s.geometric.GeneralizedGershgorin(s.elastic, -1)
	if err != nil {
		bounds.Reason = err.Error()
	}
	s.gershgorin = bounds
	if bounds.Available {
		if bounds.Upper > 0 && bounds.Upper < s.right {
			if negative, err := s.count(bounds.Upper); err == nil && negative == 0 {
				s.right = bounds.Upper
			}
		}
		// A positive global lower bound puts every generalized root above
		// it. Requiring n negatives also verifies the smallest root, rather
		// than merely validating the requested first few modes.
		if bounds.Lower > 0 && bounds.Lower < s.right {
			if negative, err := s.count(bounds.Lower); err == nil && negative == s.elastic.Size {
				s.lower = bounds.Lower
			}
		}
	}
	if s.diagnostics != nil {
		fields := map[string]interface{}{"bounds": bounds, "effective_lower_mu": s.lower, "effective_upper_mu": s.right}
		if lower, upper, ok := bounds.PositiveReciprocalBounds(); ok {
			fields["positive_lambda_lower"] = lower
			fields["positive_lambda_upper"] = nil // null means unbounded, never JSON Infinity.
			if !math.IsInf(upper, 1) {
				fields["positive_lambda_upper"] = upper
			}
		}
		s.diagnostics.log("gershgorin_bounds", fields)
	}
}

func (s *bucklingSearch) count(mu float64) (int, error) {
	at := sort.Search(len(s.samples), func(i int) bool { return s.samples[i].mu >= mu })
	if at < len(s.samples) && s.samples[at].mu == mu {
		s.cacheHits++
		return s.samples[at].negative, nil
	}
	negative, err := s.evaluateCount(mu)
	if err != nil {
		return 0, err
	}
	if at > 0 && s.samples[at-1].negative < negative || at < len(s.samples) && s.samples[at].negative > negative {
		return 0, fmt.Errorf("non-monotone buckling inertia at %.16g", mu)
	}
	s.samples = append(s.samples, bucklingInertiaSample{})
	copy(s.samples[at+1:], s.samples[at:])
	s.samples[at] = bucklingInertiaSample{mu, negative}
	return negative, nil
}

// Nudge only singular shifts, not arbitrary assembly or factorization errors.
func (s *bucklingSearch) countNudged(mu, toward float64) (float64, int, error) {
	for retry := 0; ; retry++ {
		negative, err := s.count(mu)
		if !errors.Is(err, errBucklingSingularShift) || retry == 8 {
			return mu, negative, err
		}
		next := math.Nextafter(mu, toward)
		if next == toward || next == mu {
			return mu, negative, err
		}
		mu = next
	}
}

func (s *bucklingSearch) evaluateCount(mu float64) (int, error) {
	var zero, negative int
	var err error
	if s.diagnostics == nil {
		var f *matrix.SymLDLTFactor
		f, err = s.plan.Factor(mu, s.pivot, true)
		if err == nil {
			_, zero, negative = f.Inertia()
		}
	} else {
		_, inertia, e := s.diagnostics.factor("shift_inertia", s.kg, s.ke, mu, s.csize, s.conf, s.rank, true, false)
		zero, negative, err = inertia.Zero, inertia.Negative, e
	}
	s.inertiaEvaluations++
	if err != nil {
		return 0, err
	}
	if zero != 0 {
		return 0, fmt.Errorf("%w (zero inertia %d)", errBucklingSingularShift, zero)
	}
	return negative, nil
}

func (s *bucklingSearch) eigenvalue(rank int, progress func(float64, int)) (float64, *matrix.SymLDLTFactor, error) {
	if rank < 1 || rank > s.elastic.Size {
		return 0, nil, fmt.Errorf("invalid buckling rank %d", rank)
	}
	s.rank = rank
	lo, hi, _, _ := s.bracket(rank)
	attempts := 0
	nextAttemptWidth := math.Inf(1)
	for hi-lo > s.eps {
		// A coarse interval is enough for a cheap inverse-iteration proposal.
		// Both endpoint counts must confirm its rank before it is accepted.
		if lo > 0 && attempts < 3 && hi-lo <= .01*hi && hi-lo < nextAttemptWidth {
			attempts++
			nextAttemptWidth = (hi - lo) / 256
			s.proposeNarrowBracket(rank, lo, hi)
			lo, hi, _, _ = s.bracket(rank)
			if hi-lo <= s.eps {
				break
			}
		}
		mu := lo + .5*(hi-lo)
		if mu == lo || mu == hi {
			break
		}
		mu, negative, err := s.countNudged(mu, hi)
		if err != nil {
			return 0, nil, fmt.Errorf("buckling rank %d, shift %.16g: %w", rank, mu, err)
		}
		if negative >= rank {
			lo = mu
		} else {
			hi = mu
		}
		if progress != nil {
			progress(mu, negative)
		}
	}
	lo, hi, lowerCount, upperCount := s.bracket(rank)
	if s.diagnostics != nil {
		s.diagnostics.log("inertia_bracket", map[string]interface{}{"rank": rank, "lower_mu": lo, "upper_mu": hi, "lower_negative": lowerCount, "upper_negative": upperCount, "width": hi - lo})
	}
	if lo == 0 {
		return 0, nil, fmt.Errorf("buckling rank %d: no positive root resolved; check mode count and eps", rank)
	}
	if lowerCount < rank || upperCount >= rank {
		return 0, nil, fmt.Errorf("buckling rank %d: invalid endpoint inertia (%d, %d)", rank, lowerCount, upperCount)
	}
	if s.factor == nil || s.factorMu != hi {
		var C *matrix.SymLDLTFactor
		var err error
		if s.diagnostics == nil {
			C, err = s.plan.Factor(hi, s.pivot, false)
		} else {
			C, _, err = s.diagnostics.factor("solve_factor", s.kg, s.ke, hi, s.csize, s.conf, rank, false, false)
		}
		s.solveFactorizations++
		if err != nil {
			return 0, nil, err
		}
		if _, zero, negative := C.Inertia(); zero != 0 || negative != upperCount {
			return 0, nil, fmt.Errorf("singular upper buckling bracket")
		}
		s.factor, s.factorMu = C, hi
	}
	return lo + 0.5*(hi-lo), s.factor, nil
}

// For descending mu, N(mu)=#(mu_j>mu). The rank-r root must satisfy
// N(lo)>=r and N(hi)<r. A zero lower endpoint is an unresolved boundary only;
// eigenvalue rejects it rather than returning a nonexistent positive mode.
func (s *bucklingSearch) bracket(rank int) (lo, hi float64, lowerCount, upperCount int) {
	lo, hi = s.lower, s.right
	lowerCount, upperCount = -1, 0
	for _, point := range s.samples {
		if point.mu < lo || point.mu > hi {
			continue
		}
		if point.negative >= rank {
			lo = point.mu
			lowerCount = point.negative
		} else {
			hi = point.mu
			upperCount = point.negative
		}
	}
	s.cacheHits++
	return
}

// Inverse iteration proposes a location; it never determines root counts.
// Even convergence to an adjacent root is harmless: inertia rejects it.
func (s *bucklingSearch) proposeNarrowBracket(rank int, lo, hi float64) {
	s.refinementAttempts++
	s.bracketFactorizations++
	var f *matrix.SymLDLTFactor
	var err error
	if s.diagnostics == nil {
		f, err = s.plan.Factor(hi, s.pivot, false)
	} else {
		f, _, err = s.diagnostics.factor("bracket_factor", s.kg, s.ke, hi, s.csize, s.conf, rank, false, false)
	}
	if err != nil {
		return
	}
	var candidate float64
	if s.diagnostics == nil {
		candidate, _, err = inverseBucklingCSR(s.elastic, s.geometric, f, lo+.5*(hi-lo), hi-lo, nil, nil)
	} else {
		err = s.diagnostics.measure("bracket_estimate", "pivoted_solve", &hi, rank, func(fields map[string]interface{}) error {
			var e error
			candidate, _, e = inverseBucklingCSRProgress(s.elastic, s.geometric, f, lo+.5*(hi-lo), hi-lo, nil, nil, func(iteration int, residual, orthogonality float64) {
				fields["iterations"] = iteration
				if finite := !math.IsNaN(residual) && !math.IsInf(residual, 0); finite {
					fields["residual"] = residual
				}
			})
			return e
		})
	}
	if err != nil || candidate <= lo || candidate >= hi {
		return
	}
	s.confirmCandidate(rank, lo, hi, candidate)
}

func (s *bucklingSearch) confirmCandidate(rank int, lo, hi, candidate float64) bool {
	if math.IsNaN(candidate) || candidate <= lo || candidate >= hi {
		return false
	}
	delta := math.Max(s.eps/4, 32*(math.Nextafter(candidate, math.Inf(1))-candidate))
	left, right := math.Max(lo, candidate-delta), math.Min(hi, candidate+delta)
	if left >= right || right-left >= .5*(hi-lo) {
		return false
	}
	left, nlo, err := s.countNudged(left, lo)
	if err != nil {
		return false
	}
	right, nhi, err := s.countNudged(right, hi)
	if err != nil {
		return false
	}
	accepted := nlo >= rank && nhi < rank
	if accepted {
		s.refinementsAccepted++
	}
	if s.diagnostics != nil {
		s.diagnostics.log("bracket_proposal", map[string]interface{}{"rank": rank, "candidate_mu": candidate, "lower_mu": left, "upper_mu": right, "lower_negative": nlo, "upper_negative": nhi, "accepted": accepted})
	}
	return accepted
}
