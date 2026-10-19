package arclm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"

	"github.com/yofu/st/matrix"
)

// BucklingDiagnosticsOptions enables explicit, sequential legacy/new comparison.
// Directory is a parent directory: a unique run subdirectory is always created.
// CompareLimit limits automatic comparisons of distinct search shifts; it does
// not limit new-backend logging. Explicit CompareShifts are always compared.
type BucklingDiagnosticsOptions struct {
	Directory  string
	CPUProfile bool
	// CPUSeconds bounds CPU capture independently of analysis completion.
	// Zero captures until the session closes; the command defaults to 60 s.
	CPUSeconds    float64
	CompareLimit  int
	CompareShifts []float64
	CompareOnly   bool
}

// ParseBucklingDiagnostics accepts the uppercase option map used by ExMode.
// No diagnostic switches means nil, leaving the ordinary path unchanged.
func ParseBucklingDiagnostics(args map[string]string, otp string) (*BucklingDiagnosticsOptions, error) {
	directory, enabled := args["DIAG"]
	if !enabled {
		for _, key := range []string{"DIAG-COUNT", "DIAG-SHIFT", "DIAG-ONLY", "DIAG-CPUSEC"} {
			if _, ok := args[key]; ok {
				return nil, fmt.Errorf("-%s requires -diag", strings.ToLower(key))
			}
		}
		return nil, nil
	}
	if directory == "" {
		directory = otp + ".diagnostics"
	}
	o := &BucklingDiagnosticsOptions{Directory: directory, CPUProfile: true, CPUSeconds: 60, CompareLimit: 3}
	if raw, ok := args["DIAG-CPUSEC"]; ok {
		seconds, err := strconv.ParseFloat(raw, 64)
		if err != nil || seconds < 0 || math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds > float64((1<<63-1)/1e9) {
			return nil, fmt.Errorf("invalid -diag-cpusec: %q", raw)
		}
		o.CPUSeconds = seconds
	}
	if raw, ok := args["DIAG-SHIFT"]; ok {
		o.CompareLimit = 0
		for _, part := range strings.Split(raw, ",") {
			mu, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
			if err != nil || math.IsNaN(mu) || math.IsInf(mu, 0) {
				return nil, fmt.Errorf("invalid -diag-shift: %q", part)
			}
			o.CompareShifts = append(o.CompareShifts, mu)
		}
	}
	if raw, ok := args["DIAG-COUNT"]; ok {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			return nil, fmt.Errorf("invalid -diag-count: %q", raw)
		}
		o.CompareLimit = n
	}
	_, o.CompareOnly = args["DIAG-ONLY"]
	if o.CompareOnly && len(o.CompareShifts) == 0 {
		return nil, fmt.Errorf("-diag-only requires -diag-shift (mu=1/Lambda)")
	}
	return o, nil
}

type bucklingDiagnosticPlanKey struct {
	co, other   *matrix.COOMatrix
	csize       int
	constraints string
}

type bucklingDiagnostics struct {
	label                    string
	plans                    map[bucklingDiagnosticPlanKey]*matrix.SymLDLTPlan
	options                  BucklingDiagnosticsOptions
	directory                string
	logFile, cpuFile         *os.File
	encoder                  *json.Encoder
	output                   io.Writer
	started                  time.Time
	sequence                 int
	automaticComparisons     int
	compared                 map[uint64]bool
	profileStarted, closed   bool
	profileStop, profileDone chan struct{}
	profileError             error
	failure                  error
}

func startBucklingDiagnostics(options *BucklingDiagnosticsOptions, output io.Writer) (*bucklingDiagnostics, error) {
	return startEigenDiagnostics(options, output, "BCLNG")
}

func startEigenDiagnostics(options *BucklingDiagnosticsOptions, output io.Writer, label string) (*bucklingDiagnostics, error) {

	if options == nil {
		return nil, nil
	}
	if options.Directory == "" || options.CompareLimit < 0 || options.CompareOnly && len(options.CompareShifts) == 0 || options.CPUSeconds < 0 || math.IsNaN(options.CPUSeconds) || math.IsInf(options.CPUSeconds, 0) || options.CPUSeconds > float64((1<<63-1)/1e9) {
		return nil, fmt.Errorf("invalid buckling diagnostics options")
	}
	for _, mu := range options.CompareShifts {
		if math.IsNaN(mu) || math.IsInf(mu, 0) {
			return nil, fmt.Errorf("non-finite diagnostic shift")
		}
	}
	if output == nil {
		output = io.Discard
	}
	if err := os.MkdirAll(options.Directory, 0755); err != nil {
		return nil, err
	}
	directory, err := os.MkdirTemp(options.Directory, strings.ToLower(label)+"-")
	if err != nil {
		return nil, err
	}
	directory, err = filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(directory, "events.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return nil, err
	}
	d := &bucklingDiagnostics{label: label, options: *options, directory: directory, logFile: f, encoder: json.NewEncoder(f), output: output, started: time.Now(), compared: make(map[uint64]bool)}
	if options.CPUProfile {
		d.cpuFile, err = os.OpenFile(filepath.Join(directory, "cpu.pprof"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err == nil {
			err = pprof.StartCPUProfile(d.cpuFile)
			d.profileStarted = err == nil
		}
		if err != nil {
			if d.cpuFile != nil {
				d.cpuFile.Close()
			}
			f.Close()
			return nil, fmt.Errorf("start buckling CPU profile: %w", err)
		}
		d.profileStop, d.profileDone = make(chan struct{}), make(chan struct{})
		go func(seconds float64) {
			if seconds > 0 {
				timer := time.NewTimer(time.Duration(seconds * 1e9))
				select {
				case <-d.profileStop:
				case <-timer.C:
				}
				timer.Stop()
			} else {
				<-d.profileStop
			}
			pprof.StopCPUProfile()
			d.profileError = d.cpuFile.Close()
			close(d.profileDone)
		}(d.options.CPUSeconds)
	}
	fmt.Fprintf(output, "%s DIAGNOSTICS: %s\n", label, directory)
	d.log("session_begin", map[string]interface{}{"analysis": label, "go_version": runtime.Version(), "goos": runtime.GOOS, "goarch": runtime.GOARCH, "gomaxprocs": runtime.GOMAXPROCS(0), "cpu_count": runtime.NumCPU(), "options": options, "memory_scope": "process-wide; includes concurrent work and instrumentation", "cpu_scope": "whole process; use backend/phase/shift pprof labels", "timing_scope": "instrumented; comparisons exclude GUI pivot notifications"})
	if d.failure != nil {
		d.close(d.failure)
		return nil, d.failure
	}
	return d, nil
}

func (d *bucklingDiagnostics) log(event string, fields map[string]interface{}) {
	if d == nil || d.failure != nil {
		return
	}
	fields["event"] = event
	fields["timestamp"] = time.Now().Format(time.RFC3339Nano)
	fields["elapsed_seconds"] = time.Since(d.started).Seconds()
	// No buffered writer: begin/end records are visible before completion.
	if err := d.encoder.Encode(fields); err != nil {
		d.failure = fmt.Errorf("write buckling diagnostics: %w", err)
	}
}

func (d *bucklingDiagnostics) measure(phase, backend string, shift *float64, rank int, fn func(map[string]interface{}) error) error {
	d.sequence++
	id := d.sequence
	base := func() map[string]interface{} {
		f := map[string]interface{}{"id": id, "phase": phase, "backend": backend, "rank": rank}
		if shift != nil {
			f["shift_mu"] = *shift
		}
		return f
	}
	d.log("operation_begin", base())
	if d.failure != nil {
		return d.failure
	}
	if shift != nil {
		fmt.Fprintf(d.output, "%s DIAG BEGIN %d %s %s mu=%.17g rank=%d\n", d.label, id, phase, backend, *shift, rank)
	} else {
		fmt.Fprintf(d.output, "%s DIAG BEGIN %d %s %s\n", d.label, id, phase, backend)
	}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()
	fields := base()
	var err error
	labels := []string{"phase", phase, "backend", backend}
	if shift != nil {
		labels = append(labels, "shift", strconv.FormatFloat(*shift, 'g', 17, 64))
	}
	pprof.Do(context.Background(), pprof.Labels(labels...), func(context.Context) { err = fn(fields) })
	wall := time.Since(start)
	runtime.ReadMemStats(&after)
	fields["wall_seconds"] = wall.Seconds()
	fields["allocated_bytes"] = after.TotalAlloc - before.TotalAlloc
	fields["allocations"] = after.Mallocs - before.Mallocs
	fields["heap_alloc_bytes"] = after.HeapAlloc
	fields["heap_inuse_bytes"] = after.HeapInuse
	fields["sys_bytes"] = after.Sys
	fields["gc_cycles"] = after.NumGC - before.NumGC
	fields["gc_pause_seconds"] = float64(after.PauseTotalNs-before.PauseTotalNs) / 1e9
	if err != nil {
		fields["error"] = err.Error()
	}
	d.log("operation_end", fields)
	fmt.Fprintf(d.output, "%s DIAG END %d %s %s wall=%.3fs alloc=%.1fMB gc=%d error=%v\n", d.label, id, phase, backend, wall.Seconds(), float64(after.TotalAlloc-before.TotalAlloc)/1e6, after.NumGC-before.NumGC, err)
	return errors.Join(err, d.failure)
}

func (d *bucklingDiagnostics) close(analysisError error) error {
	if d == nil || d.closed {
		return nil
	}
	d.closed = true
	fields := map[string]interface{}{"status": "complete"}
	if analysisError != nil {
		fields["status"] = "error"
		fields["error"] = analysisError.Error()
	}
	d.log("session_end", fields)
	var cpuErr error
	if d.profileStarted {
		close(d.profileStop)
		<-d.profileDone
		cpuErr = d.profileError
	}
	syncErr := d.logFile.Sync()
	closeErr := d.logFile.Close()
	return errors.Join(d.failure, cpuErr, syncErr, closeErr)
}

type diagnosticInertia struct {
	Positive int `json:"positive"`
	Zero     int `json:"zero"`
	Negative int `json:"negative"`
}

func (d *bucklingDiagnostics) legacy(phase string, co, other *matrix.COOMatrix, mu float64, csize int, conf []bool, rank int) (diagnosticInertia, error) {
	var inertia diagnosticInertia
	// Buffered notifications suppress the legacy solver's per-pivot stdout
	// without adding GUI waits. This buffer is allocated outside the timer.
	progress := make(chan int, co.Size-csize)
	err := d.measure(phase, "legacy", &mu, rank, func(fields map[string]interface{}) error {
		fields["inertia_only"] = false
		prepare := time.Now()
		input := co
		if other != nil {
			input = co.AddMat(other, mu)
		}
		ll := input.ToLLS(csize, conf)
		fields["prepare_seconds"] = time.Since(prepare).Seconds()
		before := ll.DiagnosticStatistics()
		start := time.Now()
		_, err := ll.LDLT(progress)
		fields["numeric_seconds"] = time.Since(start).Seconds()
		if err == nil {
			start = time.Now()
			ll.DiagUp()
			fields["solve_prepare_seconds"] = time.Since(start).Seconds()
		}
		after := ll.DiagnosticStatistics()
		fields["size"] = ll.Size
		fields["legacy_stored_entries_before"] = before.StoredEntries
		fields["legacy_stored_entries_after"] = after.StoredEntries
		fields["legacy_added_entries"] = after.StoredEntries - before.StoredEntries
		fields["legacy_max_column_entries"] = after.MaxColumnEntries
		fields["finite"] = after.Finite
		if err != nil {
			return err
		}
		if !after.Finite {
			return fmt.Errorf("non-finite legacy factor")
		}
		inertia = diagnosticInertia{after.Positive, after.Zero, after.Negative}
		fields["inertia"] = inertia
		return nil
	})
	return inertia, err
}

// factor uses the new result for the actual analysis; legacy is evidence only.
func (d *bucklingDiagnostics) factor(phase string, co, other *matrix.COOMatrix, mu float64, csize int, conf []bool, rank int, inertiaOnly, forceCompare bool) (*matrix.SymLDLTFactor, diagnosticInertia, error) {
	key := math.Float64bits(mu)
	automatic := !forceCompare && other != nil && !d.compared[key] && d.automaticComparisons < d.options.CompareLimit
	compare := forceCompare || automatic
	if automatic {
		d.automaticComparisons++
	}
	var old diagnosticInertia
	var oldErr error
	legacyID := 0
	if compare {
		d.compared[key] = true
		old, oldErr = d.legacy(phase, co, other, mu, csize, conf, rank)
		legacyID = d.sequence
	}
	var f *matrix.SymLDLTFactor
	var inertia diagnosticInertia
	err := d.measure(phase, "pivoted", &mu, rank, func(fields map[string]interface{}) error {
		fields["inertia_only"] = inertiaOnly
		var stats *matrix.SymLDLTStatistics
		var err error
		constraintKey := make([]byte, len(conf))
		for i, fixed := range conf {
			if fixed {
				constraintKey[i] = 1
			}
		}
		planKey := bucklingDiagnosticPlanKey{co, other, csize, string(constraintKey)}
		if d.plans == nil {
			d.plans = make(map[bucklingDiagnosticPlanKey]*matrix.SymLDLTPlan)
		}
		plan := d.plans[planKey]
		analysisSeconds := 0.0
		if plan == nil {
			started := time.Now()
			plan, err = matrix.NewSymLDLTPlan(co, other, csize, conf)
			analysisSeconds = time.Since(started).Seconds()
			if err != nil {
				return err
			}
			d.plans[planKey] = plan
		}
		f, stats, err = plan.FactorWithStats(mu, nil, inertiaOnly)
		stats.BuildSeconds += analysisSeconds
		fields["analysis_seconds"] = analysisSeconds
		fields["factor_statistics"] = stats
		fields["size"] = stats.Size
		fields["prepare_seconds"] = stats.BuildSeconds
		fields["numeric_seconds"] = stats.NumericSeconds
		if err != nil {
			return err
		}
		inertia.Positive, inertia.Zero, inertia.Negative = f.Inertia()
		fields["inertia"] = inertia
		return nil
	})
	if compare {
		fields := map[string]interface{}{"phase": phase, "shift_mu": mu, "rank": rank, "legacy_id": legacyID, "pivoted_id": d.sequence, "legacy_inertia": old, "pivoted_inertia": inertia, "legacy_valid": oldErr == nil, "pivoted_valid": err == nil, "authoritative_backend": "pivoted"}
		if oldErr != nil {
			fields["legacy_error"] = oldErr.Error()
		}
		if err != nil {
			fields["pivoted_error"] = err.Error()
		}
		if oldErr == nil && err == nil {
			fields["inertia_equal"] = old == inertia
		}
		d.log("comparison", fields)
	}
	return f, inertia, errors.Join(err, d.failure)
}

func (d *bucklingDiagnostics) explicitShifts(ke, kg *matrix.COOMatrix, csize int, conf []bool) error {
	for _, mu := range d.options.CompareShifts {
		// Singular or failed factors are logged and do not prevent measuring
		// later requested shifts. Diagnostic I/O errors still abort the run.
		d.factor("explicit_shift", kg, ke, mu, csize, conf, 0, false, true)
		if d.failure != nil {
			return d.failure
		}
	}
	return nil
}

func (d *bucklingDiagnostics) inverse(s *bucklingSearch, C *matrix.SymLDLTFactor, target, eps float64, locked, klocked [][]float64, rank int) (float64, []float64, error) {
	var mu float64
	var x []float64
	err := d.measure("inverse_iteration", "pivoted_solve", &target, rank, func(fields map[string]interface{}) error {
		var err error
		mu, x, err = inverseBucklingCSRProgress(s.elastic, s.geometric, C, target, eps, locked, klocked, func(iteration int, residual, orthogonality float64) {
			fields["iterations"] = iteration
			f := map[string]interface{}{"rank": rank, "iteration": iteration}
			if !math.IsNaN(residual) && !math.IsInf(residual, 0) {
				f["residual"] = residual
				fields["residual"] = residual
			}
			if !math.IsNaN(orthogonality) && !math.IsInf(orthogonality, 0) {
				f["orthogonality"] = orthogonality
				fields["orthogonality"] = orthogonality
			}
			d.log("vector_iteration", f)
		})
		return err
	})
	return mu, x, err
}
