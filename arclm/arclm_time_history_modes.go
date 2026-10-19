package arclm

import (
	"fmt"
	"math"
	"time"

	"github.com/yofu/st/matrix"
)

// Near-coincident anchors do not define two useful independent Rayleigh
// conditions. Default selection skips the entire cluster by an inertia count,
// instead of computing every vector of a high-multiplicity root.
const rayleighEnergyTolerance = 1e-8 // relative mu error bound for period estimation

const rayleighModeSeparation = 1e-6 // relative omega squared separation

// Keep a relative interval for each requested mode. A very high frequency can
// lie below the initial absolute search tolerance; refine that interval instead
// of misreporting a missing finite mode.
func rayleighEigenvalue(s *bucklingSearch, rank int, progress func(float64, int)) (float64, error) {
	for retry := 0; retry < 12; retry++ {
		mu, _, err := s.eigenvalue(rank, progress)
		if err == nil {
			if s.eps > mu*1.5e-8 {
				s.eps = mu * 1e-8
				continue
			}
			return mu, nil
		}
		lo, hi, _, _ := s.bracket(rank)
		if lo != 0 || hi <= 0 {
			return 0, err
		}
		if s.check != nil {
			if e := s.check(); e != nil {
				return 0, e
			}
		}
		s.eps = math.Min(s.eps*.01, hi*1e-10)
		if s.eps <= 0 {
			return 0, err
		}
	}
	return 0, fmt.Errorf("Rayleigh mode %d: cannot resolve a positive modal interval", rank)
}

// A final inertia bracket can be too close to a pole for accurate linear
// solves on a structural model with widely different translational/rotational
// scales. Move the inverse-iteration shift away, while verifying by inertia
// that it has not crossed any other eigenvalue. Periods still come from the
// residual-checked Rayleigh quotient, not from this displaced shift.
func rayleighVectorFactor(s *bucklingSearch, rank int, startRelative float64) (*matrix.SymLDLTFactor, error) {
	_, hi, _, upperCount := s.bracket(rank)
	for relative := startRelative; relative >= 1e-13; relative *= .1 {
		candidate := hi * (1 + relative)
		shift, count, err := s.countNudged(candidate, math.Inf(1))
		if err != nil {
			return nil, err
		}
		if count != upperCount {
			continue
		}
		factor, err := s.plan.Factor(shift, nil, false)
		if err != nil {
			return nil, err
		}
		return factor, nil
	}
	return nil, fmt.Errorf("Rayleigh mode %d: no stable inverse-iteration shift within its inertia interval", rank)
}

// For K>0, sqrt(r^T K^-1 r)/(mu*sqrt(x^T K x)), r=mu*K*x-M*x,
// is the relative residual of the self-adjoint K^-1*M problem in the K norm.
// It bounds the distance to the spectrum in mu and avoids judging massless,
// very stiff DOFs against tiny inertial forces. The original force residual is
// still computed and logged, rather than relaxing the existing eigen solver.
func rayleighEnergyResidual(k *matrix.SymLDLTFactor, mu float64, x, kx, gx []float64) (float64, error) {
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

func validateTimeHistoryRayleigh(o TimeHistoryOptions) error {
	if !o.AutoRayleigh {
		if o.RayleighModes != [2]int{} {
			return fmt.Errorf("Rayleigh modes require automatic period selection")
		}
		_, _, err := RayleighCoefficients(o.Period1, o.Period2, o.Damping1, o.Damping2)
		return err
	}
	if o.Period1 != 0 || o.Period2 != 0 {
		return fmt.Errorf("automatic Rayleigh periods cannot be combined with explicit periods")
	}
	if !finitePlastic(o.Damping1) || !finitePlastic(o.Damping2) || o.Damping1 < 0 || o.Damping2 < 0 {
		return fmt.Errorf("invalid automatic Rayleigh damping ratios")
	}
	if p := o.RayleighModes; p != [2]int{} && (p[0] < 1 || p[1] <= p[0]) {
		return fmt.Errorf("Rayleigh modes must be two positive ascending ranks")
	}
	return nil
}

func (d *timeHistoryAnalysis) resolveRayleighModes() error {
	a := d.a
	if err := a.ctx.Err(); err != nil {
		return err
	}
	m := matrix.NewCOOMatrix(len(d.mass))
	positive := 0
	scale := 0.
	for i, v := range d.mass {
		if v != 0 {
			m.Add(i, i, v)
		}
		if !a.conf[i] && v > 0 {
			positive++
			if k := d.k0.Query(i, i); k > 0 {
				scale = math.Max(scale, v/k)
			}
		}
	}
	modes := d.opts.RayleighModes
	automaticPair := modes == [2]int{}
	if automaticPair {
		modes = [2]int{1, 2}
	}
	if modes[1] > positive {
		return fmt.Errorf("automatic Rayleigh periods require two finite modes; requested %v but only %d positive-mass free DOFs exist; specify explicit periods if needed", modes, positive)
	}
	eps := scale * 1e-12
	if eps <= 0 || !finitePlastic(eps) {
		return fmt.Errorf("invalid modal scale for automatic Rayleigh periods")
	}
	fmt.Fprintf(a.writer, "TIMEHISTORY MODAL: START REQUESTED_MODES=%v MASS=SAME_AS_TIMEHISTORY STIFFNESS=K0\n", modes)
	negativeMass := matrix.NewCOOMatrix(m.Size).AddMat(m, -1)
	s, err := newBucklingSearchChecked(d.k0, negativeMass, a.csize, a.conf, eps, scale*2, nil, nil, a.ctx.Err)
	if err != nil {
		return fmt.Errorf("Rayleigh vibration modes: %w", err)
	}
	// Resolve a modal interval to a relative 1e-8 of the verified upper
	// spectral scale, rather than pushing inertia into roundoff at the pole.
	s.eps = math.Max(eps, s.right*1e-8)
	eps = s.eps
	mass, err := m.ToSymmetricCSR(a.csize, a.conf)
	if err != nil {
		return err
	}
	kFactor, err := d.k0.SymmetricLDLT(a.csize, a.conf, nil)
	if err != nil {
		return err
	}
	quality := func(mu float64, x, kx, gx []float64) (float64, error) {
		return rayleighEnergyResidual(kFactor, mu, x, kx, gx)
	}
	var locked, mLocked [][]float64
	var muFirst, omegaFirst float64
	lastLog := time.Now()
	for anchor := 0; anchor < 2; anchor++ {
		if err := a.ctx.Err(); err != nil {
			return err
		}
		if anchor == 1 && automaticPair {
			// Count every root in the first frequency cluster, including exact
			// multiplicities and close modes, using the same verified inertia.
			_, count, e := s.countNudged(muFirst/(1+2*rayleighModeSeparation), 0)
			if e != nil {
				return fmt.Errorf("Rayleigh distinct-mode search: %w", e)
			}
			modes[1] = count + 1
			if modes[1] < 2 {
				modes[1] = 2
			}
			if modes[1] > positive {
				return fmt.Errorf("automatic Rayleigh periods: no distinct second finite frequency; specify explicit periods")
			}
		}
		rank := modes[anchor]
		mu, e := rayleighEigenvalue(s, rank, func(mu float64, count int) {
			if time.Since(lastLog) >= 500*time.Millisecond {
				fmt.Fprintf(a.writer, "TIMEHISTORY MODAL: MODE=%d MU=%.12g COUNT=%d\n", rank, mu, count)
				lastLog = time.Now()
			}
		})
		if e != nil {
			return fmt.Errorf("Rayleigh mode %d: %w", rank, e)
		}
		eps = s.eps
		target := mu
		var x []float64
		for _, offset := range []float64{1e-3, 1e-5, 1e-7, 1e-9, 1e-11} {
			factor, err := rayleighVectorFactor(s, rank, offset)
			if err != nil {
				return err
			}
			mu, x, e = inversePencilCSRProgressQuality(s.elastic, s.geometric, mass, factor, target, eps, locked, mLocked, nil, a.ctx.Err, quality, rayleighEnergyTolerance)
			if e == nil || a.ctx.Err() != nil {
				break
			}
			fmt.Fprintf(a.writer, "TIMEHISTORY MODAL: RETRY MODE=%d SHIFT_OFFSET=%g CAUSE=%v\n", rank, offset, e)
		}
		if e != nil {
			return fmt.Errorf("Rayleigh mode %d inverse iteration: %w", rank, e)
		}
		omega2 := 1 / mu
		period := 2 * math.Pi * math.Sqrt(mu)
		if !finitePlastic(omega2) || omega2 <= 0 || !finitePlastic(period) || period <= 0 {
			return fmt.Errorf("invalid Rayleigh mode %d eigenvalue", rank)
		}
		if anchor == 1 && omega2 <= omegaFirst*(1+rayleighModeSeparation) {
			return fmt.Errorf("Rayleigh modes %d and %d have coincident or very close frequencies; choose distinct -modes or explicit periods", modes[0], rank)
		}
		kx, mx := s.elastic.MulV(x), mass.MulV(x)
		nr, nk, nm := 0., 0., 0.
		for i := range x {
			nr = math.Hypot(nr, kx[i]-omega2*mx[i])
			nk = math.Hypot(nk, kx[i])
			nm = math.Hypot(nm, omega2*mx[i])
		}
		residual := nr / (nk + nm)
		energyResidual, e := quality(mu, x, kx, s.geometric.MulV(x))
		if e != nil {
			return e
		}
		if !finitePlastic(residual) || energyResidual > rayleighEnergyTolerance {
			return fmt.Errorf("Rayleigh mode %d energy residual too large: %g", rank, energyResidual)
		}
		margin := math.Max(4*eps, 2*rayleighEnergyTolerance*mu)
		if mu <= margin {
			return fmt.Errorf("Rayleigh mode %d: modal interval is too broad", rank)
		}
		_, below, e := s.countNudged(mu-margin, 0)
		if e != nil {
			return e
		}
		_, above, e := s.countNudged(mu+margin, math.Inf(1))
		if e != nil {
			return e
		}
		if below < rank || above >= rank {
			return fmt.Errorf("Rayleigh mode %d: residual-checked root has inconsistent inertia (%d,%d)", rank, below, above)
		}
		locked = append(locked, x)
		mLocked = append(mLocked, mx)
		fmt.Fprintf(a.writer, "TIMEHISTORY MODAL: SELECTED_MODE=%d OMEGA2=%.14g PERIOD=%.14gs ENERGY_RESIDUAL=%.6e FORCE_RESIDUAL=%.6e\n", rank, omega2, period, energyResidual, residual)
		if anchor == 0 {
			muFirst, omegaFirst = mu, omega2
			d.opts.Period1 = period
		} else {
			d.opts.Period2 = period
		}
	}
	d.opts.RayleighModes = modes
	return a.ctx.Err()
}
