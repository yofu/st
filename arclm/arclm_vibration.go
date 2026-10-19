package arclm

import (
	"errors"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/yofu/st/matrix"
)

// VibrationDiagnosticsOptions shares the eigenanalysis profiling options.
// CompareShifts are reciprocal squared frequencies, mu=1/omega^2.
type VibrationDiagnosticsOptions = BucklingDiagnosticsOptions

// prepareVibrationMass keeps zero-mass DOFs in the stiffness system. Their
// displacement is determined by K; they are not deleted or given artificial
// mass. The current Frame assembly uses nonnegative lumped diagonal masses.
func prepareVibrationMass(k, m *matrix.COOMatrix, csize int, conf []bool) (*matrix.COOMatrix, *matrix.SymmetricCSR, int, error) {
	if k == nil || m == nil || k.Size != m.Size || len(conf) != k.Size {
		return nil, nil, 0, fmt.Errorf("invalid vibration matrix dimensions")
	}
	positive := 0
	for i, fixed := range conf {
		if fixed {
			continue
		}
		mass := m.Query(i, i)
		if math.IsNaN(mass) || math.IsInf(mass, 0) || mass < 0 {
			return nil, nil, 0, fmt.Errorf("invalid vibration mass at DOF %d: %g", i, mass)
		}
		if mass > 0 {
			positive++
		}
	}
	if positive == 0 {
		return nil, nil, 0, fmt.Errorf("vibration has no positive mass on free DOFs")
	}
	csr, err := m.ToSymmetricCSR(csize, conf)
	if err != nil {
		return nil, nil, 0, err
	}
	return matrix.NewCOOMatrix(m.Size).AddMat(m, -1), csr, positive, nil
}

// VibrationalEigenAnalysisWithDiagnostics solves K*x=omega^2*M*x. K must be
// positive definite after constraints. M may have zero diagonal masses.
// Finite modes are ordered by ascending omega^2 and normalized to Phi^T M Phi=I.
// The established EigenValue/OTP convention (omega squared) is preserved.
// eps is an absolute tolerance on mu=1/omega^2; right is an initial scale,
// automatically expanded or narrowed using inertia.
func (frame *Frame) VibrationalEigenAnalysisWithDiagnostics(otp string, init bool, n int, eps, right float64, options *VibrationDiagnosticsOptions) (analysisErr error) {
	if n < 1 || eps <= 0 || right <= 0 || math.IsNaN(eps) || math.IsInf(eps, 0) || math.IsNaN(right) || math.IsInf(right, 0) {
		return fmt.Errorf("invalid vibration search parameters")
	}
	d, err := startEigenDiagnostics(options, frame.Output, "VIBRATION")
	if err != nil {
		return err
	}
	if d != nil {
		defer func() { analysisErr = errors.Join(analysisErr, d.close(analysisErr)) }()
	}
	if init {
		frame.Initialise()
	}
	started := time.Now()
	measure := func(phase, backend string, rank int, fn func() error) error {
		if d == nil {
			return fn()
		}
		return d.measure(phase, backend, nil, rank, func(map[string]interface{}) error { return fn() })
	}
	var k, m *matrix.COOMatrix
	var load []float64
	if err = measure("assemble_ke", "assembly", 0, func() error { var e error; k, load, e = frame.KE(1); return e }); err != nil {
		return err
	}
	csize, conf, _ := frame.AssemConf(load, 1)
	if err = measure("assemble_mass", "assembly", 0, func() error { m = frame.AssemMassMatrix(); return nil }); err != nil {
		return err
	}
	negativeMass, mass, positive, err := prepareVibrationMass(k, m, csize, conf)
	if err != nil {
		return err
	}
	if d != nil {
		d.log("model", map[string]interface{}{"analysis": "vibration", "eigenvalue_kind": "omega_squared", "shift_variable": "mu=1/omega_squared", "total_dofs": k.Size, "free_dofs": k.Size - csize, "constrained_dofs": csize, "positive_mass_dofs": positive, "requested_modes": n, "eps_mu": eps, "right_mu": right})
		if err = d.explicitShifts(k, negativeMass, csize, conf); err != nil {
			return err
		}
		if options.CompareOnly {
			return nil
		}
	}
	if n > positive {
		return fmt.Errorf("vibration requested %d modes but only %d free DOFs have positive mass", n, positive)
	}
	searchStarted := time.Now()
	s, err := newBucklingSearchDiagnostics(k, negativeMass, csize, conf, eps, right, nil, d)
	if err != nil {
		return fmt.Errorf("vibration: %w", err)
	}
	searchTime := time.Since(searchStarted)
	var vectorTime time.Duration
	frame.EigenValue = make([]float64, n)
	frame.EigenVector = make([][]float64, n)
	locked, massLocked := make([][]float64, 0, n), make([][]float64, 0, n)
	lastProgress := time.Now()
	notify := func(lap, rank int) error {
		if frame.Lapch == nil {
			return nil
		}
		return measure("progress_wait", "ui", rank, func() error { frame.Lapch <- lap; <-frame.Lapch; return nil })
	}
	for i := 0; i < n; i++ {
		searchStarted = time.Now()
		lap := 0
		mu, factor, e := s.eigenvalue(i+1, func(mu float64, negative int) {
			lap++
			if time.Since(lastProgress) < 200*time.Millisecond {
				return
			}
			lastProgress = time.Now()
			fmt.Fprintf(frame.Output, "VIBRATION MODE %d: mu=%.14g count=%d\n", i+1, mu, negative)
			notify(lap+1, i+1)
		})
		searchTime += time.Since(searchStarted)
		if e != nil {
			return fmt.Errorf("vibration mode %d: %w", i+1, e)
		}
		vectorStarted := time.Now()
		var x []float64
		var residual, orthogonality float64
		refine := func(fields map[string]interface{}) error {
			var e error
			mu, x, e = inversePencilCSRProgress(s.elastic, s.geometric, mass, factor, mu, eps, locked, massLocked, func(iteration int, r, o float64) {
				residual, orthogonality = r, o
				if fields != nil {
					fields["iterations"] = iteration
					if !math.IsNaN(r) && !math.IsInf(r, 0) {
						fields["residual"] = r
					}
					if !math.IsNaN(o) && !math.IsInf(o, 0) {
						fields["orthogonality"] = o
					}
					fields["metric"] = "M"
				}
			})
			return e
		}
		if d == nil {
			e = refine(nil)
		} else {
			target := mu
			e = d.measure("inverse_iteration", "pivoted_solve", &target, i+1, refine)
		}
		vectorTime += time.Since(vectorStarted)
		if e != nil {
			return fmt.Errorf("vibration mode %d: %w", i+1, e)
		}
		omega2 := 1 / mu
		if math.IsNaN(omega2) || math.IsInf(omega2, 0) || omega2 <= 0 {
			return fmt.Errorf("non-finite vibration eigenvalue for mode %d", i+1)
		}
		locked = append(locked, x)
		massLocked = append(massLocked, mass.MulV(x))
		full := frame.FillConf(x)
		frame.EigenValue[i], frame.EigenVector[i] = omega2, full
		if _, _, e = frame.UpdateStressEnergy(full); e != nil {
			return e
		}
		frame.UpdateReaction(k, full)
		frame.UpdateForm(full)
		fmt.Fprintf(frame.Output, "VIBRATION EIG %d: omega^2=%.14g f=%.14gHz T=%.14gs\n", i+1, omega2, math.Sqrt(omega2)/(2*math.Pi), 2*math.Pi/math.Sqrt(omega2))
		if d != nil {
			d.log("vibration_mode", map[string]interface{}{"rank": i + 1, "omega_squared": omega2, "frequency_hz": math.Sqrt(omega2) / (2 * math.Pi), "period_seconds": 2 * math.Pi / math.Sqrt(omega2), "residual": residual, "orthogonality": orthogonality, "metric": "M"})
		}
		if e = notify(i+1, i+1); e != nil {
			return e
		}
	}
	totalFactors := s.inertiaEvaluations + s.solveFactorizations + s.bracketFactorizations
	fmt.Fprintf(frame.Output, "VIBRATION: inertia=%d factors=%d bracket_factors=%d accepted=%d total_factors=%d search=%.3fs vectors=%.3fs total=%.3fs\n", s.inertiaEvaluations, s.solveFactorizations, s.bracketFactorizations, s.refinementsAccepted, totalFactors, searchTime.Seconds(), vectorTime.Seconds(), time.Since(started).Seconds())
	if d != nil {
		d.log("vibration_summary", map[string]interface{}{"inertia_evaluations": s.inertiaEvaluations, "solve_factorizations": s.solveFactorizations, "bracket_factorizations": s.bracketFactorizations, "total_factorizations": totalFactors, "proposals": s.refinementAttempts, "accepted": s.refinementsAccepted, "cache_hits": s.cacheHits, "search_seconds": searchTime.Seconds(), "vector_seconds": vectorTime.Seconds()})
	}
	if otp == "" {
		otp = "hogtxt.otp"
	}
	w, err := os.Create(otp)
	if err != nil {
		return err
	}
	defer w.Close()
	_, err = frame.WriteBclngTo(w)
	return err
}
