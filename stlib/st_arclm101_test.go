package st

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/yofu/st/arclm"
)

func continuationCommandFrame(t *testing.T) *Frame {
	t.Helper()
	f := NewFrame()
	f.Path = filepath.Join(t.TempDir(), "model.inp")
	f.Show.Period = "X"
	f.Ai.Nfloor, f.Ai.Boundary = 2, []float64{-1, .5, 2}
	af := arclm.NewFrame()
	af.Output, af.Lapch, af.Pivot = io.Discard, nil, nil
	a, b := arclm.NewNode(), arclm.NewNode()
	a.Num, b.Num, b.Index, b.Coord[0], b.Force[0] = 1, 2, 1, 1, 40
	for j := 0; j < 6; j++ {
		a.Conf[j], b.Conf[j] = true, j != 0
	}
	s := arclm.NewSect()
	s.Num, s.E, s.Poi, s.Exp, s.Exq = 1, 100, .3, 2, 2
	s.Value = []float64{1, 1, 1, 1}
	for j := 0; j < 6; j++ {
		s.Yield[2*j], s.Yield[2*j+1] = 1000, -1000
	}
	e := arclm.NewElem()
	e.Num, e.Enod, e.Sect = 1, []*arclm.Node{a, b}, s
	af.Nodes, af.Elems, af.Sects = []*arclm.Node{a, b}, []*arclm.Elem{e}, []*arclm.Sect{s}
	f.Arclms["X"] = af
	for i := 1; i <= 2; i++ {
		n := NewNode()
		n.Num, n.Coord[2] = i, float64(i-1)
		f.Nodes[i] = n
	}
	return f
}

// Mirrors the command's per-step handshake: results and summary must be saved
// before acknowledging the solver, so it cannot mutate the next step's state.
func runContinuationCommand(t *testing.T, f *Frame, args map[string]string) {
	t.Helper()
	per, af, cond, offset, err := prepareArclm101(f, args, "")
	if err != nil {
		t.Fatal(err)
	}
	af.Lapch = make(chan int)
	done := make(chan error, 1)
	go func() { done <- af.Arclm101(nil, cond) }()
	for {
		select {
		case step := <-af.Lapch:
			if step <= offset {
				t.Errorf("step=%d offset=%d", step, offset)
			}
			f.Show.Period = fmt.Sprintf("%s@%d", per, step)
			f.ReadArclmData(af, f.Show.Period)
			f.Nlap[per] = step
			if err := f.ZoubunSummary(filepath.Join(filepath.Dir(f.Path), "summary.txt"), per, cond); err != nil {
				t.Error(err)
			}
			af.Lapch <- 0
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
			f.ResultFileName[per] = cond.Output()[0]
			return
		}
	}
}

func summaryStepRows(t *testing.T, f *Frame) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(filepath.Dir(f.Path), "summary.txt"))
	if err != nil {
		t.Fatal(err)
	}
	var rows []string
	for _, line := range strings.Split(string(data), "\n") {
		var step int
		var drift, shear float64
		if n, _ := fmt.Sscanf(line, "%d %f %f", &step, &drift, &shear); n == 3 {
			rows = append(rows, line)
		}
	}
	return rows
}

func TestArclm101DisplayedStepContinuationAndSummary(t *testing.T) {
	f := continuationCommandFrame(t)
	runContinuationCommand(t, f, map[string]string{"INIT": "", "STEP": "80;0.0125"})
	if f.Show.Period != "X@80" {
		t.Fatal(f.Show.Period)
	}
	oldRows := summaryStepRows(t, f)
	first := filepath.Join(filepath.Dir(f.Path), "model@1.otp")
	before, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	// No PERIOD, NOINIT, START, MAX or filename is needed for continuation.
	runContinuationCommand(t, f, map[string]string{"NLAP": "80", "DELTA": "0.0125"})
	rows := summaryStepRows(t, f)
	if f.Show.Period != "X@160" || f.Nlap["X"] != 160 {
		t.Fatal("lost cumulative step", f.Show.Period)
	}
	if len(oldRows) != 80 || len(rows) != 160 || !reflect.DeepEqual(oldRows, rows[:80]) {
		t.Fatal("previous summary rows changed or disappeared")
	}
	after, err := os.ReadFile(first)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("old output changed", err)
	}
	for _, step := range []int{81, 160} {
		if _, err := os.Stat(filepath.Join(filepath.Dir(f.Path), fmt.Sprintf("model@%d.otp", step))); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range []map[string]string{{"STEP": "80;.0125"}, {"NOINIT": "", "STEP": "80;.0125;2;3"}} {
		_, _, cond, offset, err := prepareArclm101(f, args, "")
		if err != nil || offset != 160 || cond.Nlap() != 80 {
			t.Fatalf("legacy syntax: %v offset=%d", err, offset)
		}
	}
	f.Show.Period = "X@80"
	if _, _, _, _, err := prepareArclm101(f, map[string]string{}, ""); err == nil {
		t.Fatal("accepted old display as a material snapshot")
	}
	_, _, _, offset, err := prepareArclm101(f, map[string]string{"INIT": ""}, "")
	if err != nil || offset != 0 {
		t.Fatal("explicit init did not reset numbering", err)
	}
}

func TestArclm101InvalidOptionsDoNotChangeState(t *testing.T) {
	f := continuationCommandFrame(t)
	before := f.Arclms["X"].SaveState()
	for _, args := range []map[string]string{
		{"INIT": "", "NOINIT": ""}, {"NLAP": "0"}, {"DELTA": "0"}, {"DELTA": "NaN"}, {"EPS": "-1"}, {"STEP": "x;.1"}, {"STEP": "1;2;3;4;5"}, {"PERIOD": "MISSING"},
	} {
		if _, _, _, _, err := prepareArclm101(f, args, ""); err == nil {
			t.Errorf("accepted %v", args)
		}
		if !reflect.DeepEqual(before, f.Arclms["X"].SaveState()) {
			t.Fatal("invalid options changed state")
		}
	}
}

func TestArclm101DifferentPeriodPreservesPreloadAndStartsNewSteps(t *testing.T) {
	f := continuationCommandFrame(t)
	runContinuationCommand(t, f, map[string]string{"INIT": "", "STEP": "2;.5"})
	source := f.Arclms["X"]
	other := continuationCommandFrame(t).Arclms["X"]
	other.Nodes[1].Force[0] = 80
	f.Arclms["Y"] = other
	before := source.SaveState()
	per, af, cond, offset, err := prepareArclm101(f, map[string]string{"PERIOD": "Y", "STEP": "2;.5"}, filepath.Join(t.TempDir(), "y.oty"))
	if err != nil || per != "Y" || offset != 0 {
		t.Fatalf("new stage: per=%s offset=%d err=%v", per, offset, err)
	}
	// The copied X preload is 40; Y adds 80 at its own factor one.
	if err := af.Arclm101(nil, cond); err != nil {
		t.Fatal(err)
	}
	if af.ContinuationStep() != 2 || !reflect.DeepEqual(before, source.SaveState()) {
		t.Fatal("new stage numbering or source state changed")
	}
	if d := af.Nodes[1].Disp[0]; d < 1.2-1e-9 || d > 1.2+1e-9 {
		t.Fatalf("preload was lost: displacement=%g want=1.2", d)
	}
}
