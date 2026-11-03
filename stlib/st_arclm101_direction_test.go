package st

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/yofu/st/arclm"
)

// Set ST_ARCLM101_MODEL to run the actual input model through loading,
// unloading and reloading. No private model data or machine paths are embedded
// in the test. ST_ARCLM101_RESULTS optionally retains the logs and step outputs.
func TestArclm101ModelLoadingCycles(t *testing.T) {
	input := os.Getenv("ST_ARCLM101_MODEL")
	if input == "" {
		t.Skip("set ST_ARCLM101_MODEL to exercise the full input model")
	}
	for _, solver := range []string{"LLS", "LDLT"} {
		for _, onlyStart := range []bool{false, true} {
			name := solver + "_both"
			if onlyStart {
				name = solver + "_start"
			}
			t.Run(name, func(t *testing.T) { runArclm101ModelCycle(t, input, name, solver, onlyStart) })
		}
	}
}

func runArclm101ModelCycle(t *testing.T, input, name, solver string, onlyStart bool) {
	t.Helper()
	dir := t.TempDir()
	if parent := os.Getenv("ST_ARCLM101_RESULTS"); parent != "" {
		dir = filepath.Join(parent, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	f := NewFrame()
	if err := f.ReadInp(input, nil, 0, false); err != nil {
		t.Fatal(err)
	}
	if f.Sects[221] == nil {
		t.Fatal("regression model must contain SECT 221")
	}
	f.Sects[221].ElasticEnds = [2]bool{false, onlyStart}
	f.Path = filepath.Join(dir, "model.inp")
	if err := f.ExtractArclm(filepath.Join(dir, "model.wgt")); err != nil {
		t.Fatal(err)
	}
	log, err := os.Create(filepath.Join(dir, "run.log"))
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	l, x := f.Arclms["L"], f.Arclms["X"]
	l.Output, x.Output = log, log
	l.Pivot = make(chan int, 6*len(l.Nodes)+1)
	l.Lapch = make(chan int, 1)
	x.Pivot = nil
	// Cache transforms in the input geometry: Arclm101 is material-only.
	transforms := make([][][]float64, len(x.Elems))
	stiffness := make([][][]float64, len(x.Elems))
	length := 0.0
	for i, e := range x.Elems {
		transforms[i], err = e.TransMatrix()
		if err != nil {
			t.Fatal(err)
		}
		stiffness[i], err = e.StiffMatrix()
		if err != nil {
			t.Fatal(err)
		}
		length = math.Max(length, e.Length())
	}
	baseLoad, pattern := make([]float64, 6*len(x.Nodes)), make([]float64, 6*len(x.Nodes))
	for i, n := range x.Nodes {
		copy(pattern[6*i:6*i+6], n.Force)
		copy(baseLoad[6*i:6*i+6], l.Nodes[i].Force)
	}
	c := arclm.NewAnalysisCondition()
	c.SetOutput([]string{filepath.Join(dir, "L.otl")})
	if err := l.StaticAnalysis(func() {}, c); err != nil {
		t.Fatal(err)
	}
	x.RestoreState(l.SaveState())
	cf, err := os.Create(filepath.Join(dir, "cycles.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer cf.Close()
	cw := csv.NewWriter(cf)
	defer cw.Flush()
	cw.Write([]string{"phase", "step", "load_factor", "max_abs_Ux_m", "residual", "tolerance", "plastic_ends"})
	maxRatio := 0.0
	norm := func(v []float64) float64 {
		r := 0.0
		for i, value := range v {
			if x.Nodes[i/6].Conf[i%6] {
				continue
			}
			if i%6 >= 3 {
				value /= length
			}
			r = math.Hypot(r, value)
		}
		return r
	}
	check := func(phase string, step int, factor float64) error {
		internal, external := make([]float64, len(pattern)), make([]float64, len(pattern))
		for ei, e := range x.Elems {
			if !e.IsValid {
				continue
			}
			tr := transforms[ei]
			for i := 0; i < 12; i++ {
				row := 6*e.Enod[i/6].Index + i%6
				for j := 0; j < 12; j++ {
					internal[row] += tr[j][i] * e.Stress[j]
				}
			}
			yield, _, _, _, _ := e.Coefficients(stiffness[ei])
			for end, v := range yield {
				if math.IsNaN(v) || math.IsInf(v, 0) || v > arclm.RADIUS+1e-8 {
					return fmt.Errorf("ELEM %d end %d outside yield surface: %g", e.Num, end, v)
				}
				if e.Sect.ElasticEnds[end] && e.Phinge[end] {
					return fmt.Errorf("elastic end yielded: ELEM %d end %d", e.Num, end)
				}
			}
		}
		residual := make([]float64, len(pattern))
		for i := range pattern {
			external[i] = baseLoad[i] + factor*pattern[i]
			residual[i] = external[i] - internal[i]
			if x.Nodes[i/6].Conf[i%6] && math.Abs(x.Nodes[i/6].Reaction[i%6]+residual[i]) > 1e-8 {
				return fmt.Errorf("reaction mismatch at dof %d", i)
			}
		}
		rn := norm(residual)
		reference := math.Max(math.Max(norm(internal), norm(external)), math.Max(norm(baseLoad), math.Abs(factor)*norm(pattern)))
		tolerance := 1e-9 + 1e-12*reference
		maxRatio = math.Max(maxRatio, rn/tolerance)
		if !isFiniteDirectionTest(rn) || rn > 1.05*tolerance+1e-12 {
			return fmt.Errorf("step %d: residual=%g tolerance=%g", step, rn, tolerance)
		}
		maxu, plastic := 0.0, 0
		for _, n := range x.Nodes {
			maxu = math.Max(maxu, math.Abs(n.Disp[0]))
		}
		for _, e := range x.Elems {
			for _, ph := range e.Phinge {
				if ph {
					plastic++
				}
			}
		}
		return cw.Write([]string{phase, fmt.Sprint(step), fmt.Sprintf("%.12g", factor), fmt.Sprintf("%.12g", maxu), fmt.Sprintf("%.12g", rn), fmt.Sprintf("%.12g", tolerance), fmt.Sprint(plastic)})
	}
	for stage, phase := range []string{"load", "unload", "reload"} {
		from, delta := 0.0, .01
		if stage == 1 {
			from, delta = 2, -.01
		}
		c = arclm.NewAnalysisCondition()
		c.SetInit(false)
		c.SetSolver(solver)
		c.SetNlap(200)
		c.SetDelta(delta)
		c.SetOutput([]string{filepath.Join(dir, "X.otx")})
		x.Lapch = make(chan int)
		done := make(chan error, 1)
		go func() { done <- x.Arclm101(nil, c) }()
		completed := 0
		var checkErr error
	loop:
		for {
			select {
			case step := <-x.Lapch:
				completed++
				checkErr = check(phase, step, from+float64(completed)*delta)
				cw.Flush()
				if checkErr != nil {
					x.Lapch <- 1
				} else {
					x.Lapch <- 0
				}
			case err = <-done:
				break loop
			}
		}
		if checkErr != nil {
			t.Fatal(checkErr)
		}
		if err != nil {
			t.Fatalf("%s: %v", phase, err)
		}
		if completed != 200 || x.ContinuationStep() != (stage+1)*200 {
			t.Fatalf("%s: completed=%d continuation=%d", phase, completed, x.ContinuationStep())
		}
		maxu := 0.0
		for _, n := range x.Nodes {
			maxu = math.Max(maxu, math.Abs(n.Disp[0]))
		}
		t.Logf("%s completed=200 factor=%g max|Ux|=%.12g m max residual/tolerance=%.6g", phase, from+200*delta, maxu, maxRatio)
	}
	if err := cw.Error(); err != nil {
		t.Fatal(err)
	}
}

func isFiniteDirectionTest(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
