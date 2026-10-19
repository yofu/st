package arclm

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yofu/st/matrix"
)

func diagnosticEvents(t *testing.T, dir string) []map[string]interface{} {
	t.Helper()
	f, err := os.Open(filepath.Join(dir, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	var events []map[string]interface{}
	for {
		var e map[string]interface{}
		err := dec.Decode(&e)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, e)
	}
	return events
}

func TestParseBucklingDiagnostics(t *testing.T) {
	if o, e := ParseBucklingDiagnostics(nil, "x.otp"); o != nil || e != nil {
		t.Fatal("ordinary command enabled diagnostics")
	}
	o, err := ParseBucklingDiagnostics(map[string]string{"DIAG": ""}, "x.otp")
	if err != nil || o.Directory != "x.otp.diagnostics" || o.CompareLimit != 3 || !o.CPUProfile {
		t.Fatal(o, err)
	}
	for _, args := range []map[string]string{
		{"DIAG-ONLY": ""}, {"DIAG": "", "DIAG-ONLY": ""}, {"DIAG": "", "DIAG-COUNT": "-1"},
		{"DIAG": "", "DIAG-SHIFT": "NaN"}, {"DIAG": "", "DIAG-SHIFT": "Inf"}, {"DIAG": "", "DIAG-SHIFT": ""},
		{"DIAG": "", "DIAG-CPUSEC": "-1"}, {"DIAG": "", "DIAG-CPUSEC": "NaN"}, {"DIAG": "", "DIAG-CPUSEC": "Inf"},
	} {
		if _, err := ParseBucklingDiagnostics(args, "x.otp"); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestDiagnosticsMatchedShiftsAndLegacyFailure(t *testing.T) {
	d, err := startBucklingDiagnostics(&BucklingDiagnosticsOptions{Directory: t.TempDir(), CompareLimit: 1}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	defer d.close(nil)
	ke, kg := matrix.NewCOOMatrix(2), matrix.NewCOOMatrix(2)
	ke.Set(0, 0, 1)
	ke.Set(1, 1, 1)
	kg.Set(0, 0, -.5)
	kg.Set(1, 1, -.5)
	kg.Set(0, 1, .25)
	kg.Set(1, 0, .25)
	_, inertia, err := d.factor("shift_inertia", kg, ke, .5, 0, []bool{false, false}, 1, true, false)
	if err != nil || inertia.Positive != 1 || inertia.Negative != 1 {
		t.Fatal(inertia, err)
	}
	// A second real factorization is logged, but exceeds the comparison budget.
	if _, _, err := d.factor("shift_inertia", kg, ke, .6, 0, []bool{false, false}, 1, true, false); err != nil {
		t.Fatal(err)
	}
	events := diagnosticEvents(t, d.directory) // read while the session remains open
	ends, comparisons := map[string]int{}, 0
	for _, e := range events {
		if e["event"] == "operation_end" {
			ends[e["backend"].(string)]++
			if _, ok := e["wall_seconds"]; !ok {
				t.Fatal("missing timing")
			}
			if _, ok := e["allocated_bytes"]; !ok {
				t.Fatal("missing memory measurement")
			}
		}
		if e["event"] == "comparison" {
			comparisons++
			if e["shift_mu"] != .5 || e["legacy_valid"] != false || e["pivoted_valid"] != true {
				t.Fatalf("comparison=%v", e)
			}
			if e["authoritative_backend"] != "pivoted" {
				t.Fatal("legacy result influenced analysis")
			}
		}
	}
	if ends["legacy"] != 1 || ends["pivoted"] != 2 || comparisons != 1 {
		t.Fatal(ends, comparisons)
	}
}

func TestDiagnosticSearchPreservesEigenpairs(t *testing.T) {
	ke, kg := matrix.NewCOOMatrix(4), matrix.NewCOOMatrix(4)
	for i, r := range []float64{.8, .5, .5, .2} {
		ke.Set(i, i, float64(i+1))
		kg.Set(i, i, -r*float64(i+1))
	}
	conf := make([]bool, 4)
	d, err := startBucklingDiagnostics(&BucklingDiagnosticsOptions{Directory: t.TempDir(), CompareLimit: 2}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	defer d.close(nil)
	s, err := newBucklingSearchDiagnostics(ke, kg, 0, conf, 1e-12, 1.3, nil, d)
	if err != nil {
		t.Fatal(err)
	}
	var locked, klocked [][]float64
	for rank, want := range []float64{.8, .5, .5, .2} {
		mu, C, err := s.eigenvalue(rank+1, nil)
		if err != nil {
			t.Fatal(err)
		}
		mu, x, err := d.inverse(s, C, mu, 1e-12, locked, klocked, rank+1)
		if err != nil || math.Abs(mu-want) > 1e-11 {
			t.Fatal(mu, err)
		}
		kx := s.elastic.MulV(x)
		for _, q := range locked {
			if math.Abs(Dot(q, kx, 4)) > 1e-10 {
				t.Fatal("diagnostics changed orthogonality")
			}
		}
		locked = append(locked, x)
		klocked = append(klocked, kx)
	}
}

func diagnosticCantilever() *Frame {
	f := NewFrame()
	f.Output = io.Discard
	f.Pivot = make(chan int, 12)
	a, b := NewNode(), NewNode()
	a.Num = 1
	b.Num = 2
	a.Index = 0
	b.Index = 1
	b.Coord[0] = 1
	b.Force[0] = -1
	for i := range a.Conf {
		a.Conf[i] = true
	}
	s := NewSect()
	s.Num = 1
	s.E = 100
	s.Poi = .3
	s.Value = []float64{1, 1, 1, 1}
	e := NewElem()
	e.Num = 1
	e.Sect = s
	e.Enod = []*Node{a, b}
	f.Nodes = []*Node{a, b}
	f.Elems = []*Elem{e}
	f.Sects = []*Sect{s}
	return f
}

func TestDiagnosticOnlyRealAssemblyAndCPUProfile(t *testing.T) {
	dir := t.TempDir()
	otp := filepath.Join(dir, "unused.otp")
	f := diagnosticCantilever()
	var output bytes.Buffer
	f.Output = &output
	err := f.Bclng001WithDiagnostics(otp, true, 2, 1e-12, 1, &BucklingDiagnosticsOptions{Directory: dir, CPUProfile: true, CompareShifts: []float64{.5}, CompareOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(otp); !os.IsNotExist(err) {
		t.Fatal("comparison-only wrote eigenvalue output")
	}
	runs, err := filepath.Glob(filepath.Join(dir, "bclng-*"))
	if err != nil || len(runs) != 1 {
		t.Fatal(runs, err)
	}
	events := diagnosticEvents(t, runs[0])
	if events[len(events)-1]["event"] != "session_end" || events[len(events)-1]["status"] != "complete" {
		t.Fatal("missing completed log")
	}
	seenModel, seenComparison := false, false
	for _, event := range events {
		if event["event"] == "model" {
			seenModel = true
		}
		if event["event"] == "comparison" {
			seenComparison = true
		}
	}
	if !seenModel || !seenComparison {
		t.Fatal("assembly/comparison was not logged")
	}
	profile, err := os.Open(filepath.Join(runs[0], "cpu.pprof"))
	if err != nil {
		t.Fatal(err)
	}
	defer profile.Close()
	z, err := gzip.NewReader(profile)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(z)
	z.Close()
	if err != nil || len(data) == 0 {
		t.Fatal("incomplete CPU profile", err)
	}
	if !strings.Contains(output.String(), "DIAG BEGIN") || !strings.Contains(output.String(), "DIAG END") {
		t.Fatal("missing intermediate console log")
	}
}

func TestDiagnosticCloseOnAnalysisError(t *testing.T) {
	f := NewFrame()
	f.Output = io.Discard
	f.Pivot = make(chan int, 6)
	f.Nodes = []*Node{NewNode()}
	dir := t.TempDir()
	if err := f.Bclng001WithDiagnostics(filepath.Join(dir, "x.otp"), true, 1, 1e-12, 1, &BucklingDiagnosticsOptions{Directory: dir, CPUProfile: true}); err == nil {
		t.Fatal("singular model succeeded")
	}
	runs, _ := filepath.Glob(filepath.Join(dir, "bclng-*"))
	events := diagnosticEvents(t, runs[0])
	if events[len(events)-1]["status"] != "error" {
		t.Fatal("analysis error not logged")
	}
	// Starting another CPU profile proves that the deferred cleanup released it.
	d, err := startBucklingDiagnostics(&BucklingDiagnosticsOptions{Directory: dir, CPUProfile: true}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.close(nil); err != nil {
		t.Fatal(err)
	}
}

func TestDiagnosticFullAnalysisMatchesOrdinary(t *testing.T) {
	run := func(f *Frame, options *BucklingDiagnosticsOptions) {
		t.Helper()
		done := make(chan struct{})
		defer close(done)
		go func() {
			for {
				select {
				case <-done:
					return
				case <-f.Lapch:
					f.Lapch <- 1
				}
			}
		}()
		if err := f.Bclng001WithDiagnostics(filepath.Join(t.TempDir(), "model.otp"), true, 2, 1e-12, 1, options); err != nil {
			t.Fatal(err)
		}
	}
	ordinary, measured := diagnosticCantilever(), diagnosticCantilever()
	run(ordinary, nil)
	run(measured, &BucklingDiagnosticsOptions{Directory: t.TempDir(), CompareLimit: 2})
	for i, want := range ordinary.EigenValue {
		if math.Abs(measured.EigenValue[i]-want) > 1e-10*math.Abs(want) {
			t.Fatal("diagnostics changed eigenvalue")
		}
		for j, v := range ordinary.EigenVector[i] {
			if math.Abs(measured.EigenVector[i][j]-v) > 1e-10 {
				t.Fatal("diagnostics changed eigenvector")
			}
		}
	}
}

func TestDiagnosticBusyProfileDoesNotStopOwner(t *testing.T) {
	opts := &BucklingDiagnosticsOptions{Directory: t.TempDir(), CPUProfile: true}
	first, err := startBucklingDiagnostics(opts, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	defer first.close(nil)
	if _, err := startBucklingDiagnostics(opts, io.Discard); err == nil {
		t.Fatal("accepted competing process-wide CPU profile")
	}
	if err := first.close(nil); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(filepath.Join(first.directory, "cpu.pprof"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	z, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.ReadAll(z)
	z.Close()
	if err != nil {
		t.Fatal("owner profile was damaged", err)
	}
}

func TestDiagnosticTimedCPUProfileAvailableBeforeSessionEnd(t *testing.T) {
	d, err := startBucklingDiagnostics(&BucklingDiagnosticsOptions{Directory: t.TempDir(), CPUProfile: true, CPUSeconds: .01}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	defer d.close(nil)
	select {
	case <-d.profileDone:
	case <-time.After(3 * time.Second):
		t.Fatal("CPU profile was not finalized while analysis remained active")
	}
	file, err := os.Open(filepath.Join(d.directory, "cpu.pprof"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	z, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.ReadAll(z)
	z.Close()
	if err != nil {
		t.Fatal(err)
	}
	if d.closed {
		t.Fatal("timed CPU profile closed the analysis session")
	}
}
