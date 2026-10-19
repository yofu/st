package st

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/yofu/st/arclm"
)

func floorResponseFixture() (*Frame, *arclm.Frame) {
	f := NewFrame()
	f.Ai.Nfloor = 3
	f.Ai.Boundary = []float64{-1, 1.5, 4.5, 7}
	af := arclm.NewFrame()
	for i := 0; i < 6; i++ {
		n := NewNode()
		n.Num = i + 1
		n.Coord = []float64{float64(i % 2), 0, 3 * float64(i/2)}
		f.Nodes[n.Num] = n
		a := arclm.NewNode()
		a.Num, a.Index, a.Coord = i+1, i, append([]float64(nil), n.Coord...)
		af.Nodes = append(af.Nodes, a)
	}
	s := arclm.NewSect()
	s.Num, s.E, s.Poi = 1, 100, .3
	s.Value = []float64{1, 1, 1, 1}
	for i, pair := range [][2]int{{0, 2}, {4, 2}, {1, 5}, {0, 4}, {2, 3}} {
		e := arclm.NewElem()
		e.Num, e.Sect, e.Enod = i+1, s, []*arclm.Node{af.Nodes[pair[0]], af.Nodes[pair[1]]}
		e.IsValid = i != 3
		af.Elems = append(af.Elems, e)
	}
	return f, af
}

func setFloorTestForces(t *testing.T, af *arclm.Frame, scale float64) {
	t.Helper()
	for i, e := range af.Elems {
		forces := [2]float64{1e9, 1e9}
		if i < 3 {
			forces = [][2]float64{{10, 20}, {30, 40}, {5, -6}}[i]
		}
		lower := 0
		if e.Enod[1].Coord[2] < e.Enod[0].Coord[2] {
			lower = 1
		}
		global := make([]float64, 12)
		for axis := 0; axis < 2; axis++ {
			global[6*lower+axis] = -scale * forces[axis]
			global[6*(1-lower)+axis] = scale * forces[axis]
		}
		reference := *e
		reference.Enod = make([]*arclm.Node, 2)
		for k, n := range e.Enod {
			c := *n
			c.Disp = make([]float64, 6)
			reference.Enod[k] = &c
		}
		tr, err := reference.TransMatrix()
		if err != nil {
			t.Fatal(err)
		}
		for j := 0; j < 12; j++ {
			e.Stress[j] = 0
			for k := 0; k < 12; k++ {
				e.Stress[j] += tr[j][k] * global[k]
			}
		}
	}
}

func readResponseCSV(t *testing.T, path string) []map[string]string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("missing header")
	}
	var result []map[string]string
	for _, row := range rows[1:] {
		m := make(map[string]string)
		for i, key := range rows[0] {
			m[key] = row[i]
		}
		result = append(result, m)
	}
	return result
}

func responseNumber(t *testing.T, row map[string]string, key string) float64 {
	t.Helper()
	v, err := strconv.ParseFloat(row[key], 64)
	if err != nil {
		t.Fatalf("%s: %v", key, err)
	}
	return v
}

func nearFloor(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-10*math.Max(1, math.Abs(want)) {
		t.Fatalf("got %.15g want %.15g", got, want)
	}
}

func TestTimeHistoryFloorResponsesAndPeaks(t *testing.T) {
	f, af := floorResponseFixture()
	prefix := filepath.Join(t.TempDir(), "floor")
	f.Show.Unit[0] = 1000 // CSV uses analysis force units independently of display.
	w, err := NewTimeHistoryFloorOutput(f, af, prefix)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if len(w.stories[0].members) != 2 || len(w.stories[1].members) != 2 {
		t.Fatal("cut membership", w.stories)
	}
	for step := 0; step < 3; step++ {
		s := &arclm.TimeHistoryState{Step: step, Time: .1 * float64(step), Acceleration: make([]float64, 36)}
		for i, n := range af.Nodes {
			floor := i / 2
			value := [][3]float64{{0, .03, .09}, {0, .06, .03}, {0, -.09, -.06}}[step][floor]
			n.Disp[0] = value
		}
		if step == 1 {
			s.Ground = [2]float64{1, -2}
			s.Acceleration[12], s.Acceleration[18] = 1, -3
			s.Acceleration[13], s.Acceleration[19] = 2, 0
			s.Acceleration[24], s.Acceleration[30] = 5, -9
			s.Acceleration[25], s.Acceleration[31] = 1, 2
		}
		if step == 2 {
			s.Acceleration[24], s.Acceleration[30] = 3, -1
		}
		setFloorTestForces(t, af, []float64{1, 2, -1}[step])
		if err := w.Write(s); err != nil {
			t.Fatal(err)
		}
		peaks := readResponseCSV(t, prefix+".peaks.csv")
		if len(peaks) != 24 {
			t.Fatal("peaks unavailable during analysis")
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	stories := readResponseCSV(t, prefix+".stories.csv")
	if len(stories) != 6 {
		t.Fatal("story rows", len(stories))
	}
	for step := 0; step < 3; step++ {
		for story := 0; story < 2; story++ {
			row := stories[2*step+story]
			factor := []float64{1, 2, -1}[step]
			nearFloor(t, responseNumber(t, row, "member_shear_x"), factor*[]float64{15, 35}[story])
			nearFloor(t, responseNumber(t, row, "member_shear_y"), factor*[]float64{14, 34}[story])
		}
	}
	nearFloor(t, responseNumber(t, stories[0], "drift_x_rad"), .01)
	nearFloor(t, responseNumber(t, stories[0], "incremental_drift_x_rad"), 0)
	nearFloor(t, responseNumber(t, stories[3], "incremental_drift_x_rad"), -.03)
	floors := readResponseCSV(t, prefix+".floors.csv")
	if len(floors) != 9 {
		t.Fatal("floor rows")
	}
	nearFloor(t, responseNumber(t, floors[3], "mean_absolute_acc_x_cm_s2"), 100)
	nearFloor(t, responseNumber(t, floors[4], "mean_absolute_acc_x_cm_s2"), 0)
	nearFloor(t, responseNumber(t, floors[5], "mean_absolute_acc_x_cm_s2"), -100)
	nearFloor(t, responseNumber(t, floors[5], "node_max_abs_acc_x_cm_s2"), 800)
	if floors[5]["node_max_abs_acc_x_node"] != "6" {
		t.Fatal("wrong acceleration node")
	}
	for _, row := range readResponseCSV(t, prefix+".peaks.csv") {
		if row["location"] == "story" && row["index"] == "1" && row["quantity"] == "drift" && row["direction"] == "X" {
			nearFloor(t, responseNumber(t, row, "maximum"), .02)
			nearFloor(t, responseNumber(t, row, "minimum"), -.03)
			nearFloor(t, responseNumber(t, row, "absolute_maximum_signed"), -.03)
			nearFloor(t, responseNumber(t, row, "absolute_maximum_time_s"), .2)
		}
		if row["location"] == "floor" && row["index"] == "2" && row["quantity"] == "node_absolute_acceleration" && row["direction"] == "X" {
			nearFloor(t, responseNumber(t, row, "maximum"), 600)
			nearFloor(t, responseNumber(t, row, "minimum"), -800)
			if row["maximum_node"] != "5" || row["minimum_node"] != "6" || row["absolute_maximum_node"] != "6" {
				t.Fatal("node acceleration envelope ignored non-dominant sign", row)
			}
		}
	}
	// An end exactly on the boundary belongs to the upper group; its lower
	// member is included once, and a member starting there is excluded below.
	f.Ai.Boundary[1] = 3
	check, err := NewTimeHistoryFloorOutput(f, af, filepath.Join(t.TempDir(), "boundary"))
	if err != nil {
		t.Fatal(err)
	}
	defer check.Close()
	if len(check.stories[0].members) != 2 {
		t.Fatal("boundary-end membership")
	}
}

func TestTimeHistoryFloorValidationAndOverwrite(t *testing.T) {
	f, af := floorResponseFixture()
	dir := t.TempDir()
	prefix := filepath.Join(dir, "bad")
	f.Ai.Boundary[1] = f.Ai.Boundary[0]
	if _, err := NewTimeHistoryFloorOutput(f, af, prefix); err == nil {
		t.Fatal("accepted nonmonotone boundary")
	}
	f.Ai.Boundary = []float64{-1, .1, .2, 7}
	if _, err := NewTimeHistoryFloorOutput(f, af, prefix); err == nil {
		t.Fatal("accepted empty floor")
	}
	f, af = floorResponseFixture()
	af.Elems[1] = nil
	if _, err := NewTimeHistoryFloorOutput(f, af, prefix); err == nil {
		t.Fatal("accepted nil member")
	}
	f, af = floorResponseFixture()
	if err := os.WriteFile(prefix+".peaks.csv", []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewTimeHistoryFloorOutput(f, af, prefix); err == nil {
		t.Fatal("overwrote existing peaks")
	}
	w, err := NewTimeHistoryFloorOutput(f, af, filepath.Join(dir, "valid"))
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := w.Write(&arclm.TimeHistoryState{Step: 1, Time: .1, Acceleration: make([]float64, 36)}); err == nil {
		t.Fatal("accepted missing initial response")
	}
}

func TestTimeHistoryDrawIntervalKeepsAllResponses(t *testing.T) {
	var reference *arclm.FrameState
	for _, setting := range []string{"default", "0", "1", "10"} {
		t.Run(setting, func(t *testing.T) {
			f := continuationCommandFrame(t)
			runContinuationCommand(t, f, map[string]string{"INIT": "", "STEP": "1;1"})
			f.Arclms["L"] = f.Arclms["X"]
			f.Arclms["L"].Nodes[1].Mass = 1
			input := filepath.Join(t.TempDir(), "wave.csv")
			os.WriteFile(input, []byte("0,0,0\n.023,100,0\n"), 0600)
			w := &transientTestWindow{frame: f, dir: filepath.Dir(input)}
			args := map[string]string{"RAYLEIGH": "1;.2;.02", "DT": ".001", "WAIT": "", "NODES": "1"}
			if setting != "default" {
				args["DRAW"] = setting
			}
			if _, ok := runTimeHistoryCommand(w, input, args).(ArclmStart); !ok {
				t.Fatal("command failed")
			}
			want := []string{"TH@10", "TH@20", "TH@23"}
			if setting == "0" {
				want = []string{"TH@23"}
			}
			if setting == "1" {
				want = nil
				for i := 1; i <= 23; i++ {
					want = append(want, fmt.Sprintf("TH@%d", i))
				}
			}
			if !reflect.DeepEqual(w.draws, want) {
				t.Fatalf("draws=%v want=%v", w.draws, want)
			}
			if f.Show.PlotState&PLOT_DEFORMED == 0 || f.Show.Period != "TH@23" {
				t.Fatal("deformation display not updated")
			}
			for step := 1; step <= 23; step++ {
				if _, ok := f.Nodes[2].Disp[fmt.Sprintf("TH@%d", step)]; !ok {
					t.Fatal("skipped saved response")
				}
			}
			floors := readResponseCSV(t, filepath.Join(w.dir, "wave.response.floors.csv"))
			stories := readResponseCSV(t, filepath.Join(w.dir, "wave.response.stories.csv"))
			if len(floors) != 48 || len(stories) != 24 {
				t.Fatal("drawing skipped numerical output")
			}
			if responseNumber(t, floors[47], "mean_absolute_acc_x_cm_s2") == 0 {
				t.Fatal("-nodes selection incorrectly filtered floor output")
			}
			state := f.Arclms["TH"].SaveState()
			if reference == nil {
				reference = state
			} else if !reflect.DeepEqual(reference, state) {
				t.Fatal("drawing changed numerical response")
			}
		})
	}
	for _, bad := range []string{"-1", "abc", "1.5"} {
		if _, err := timeHistoryDrawInterval(map[string]string{"DRAW": bad}); err == nil {
			t.Fatal("accepted invalid draw interval")
		}
	}
}

func TestTimeHistoryStopDuringDrawDoesNotBlock(t *testing.T) {
	f := continuationCommandFrame(t)
	runContinuationCommand(t, f, map[string]string{"INIT": "", "STEP": "1;1"})
	f.Arclms["L"] = f.Arclms["X"]
	f.Arclms["L"].Nodes[1].Mass = 1
	input := filepath.Join(t.TempDir(), "wave.csv")
	os.WriteFile(input, []byte("0,0,0\n.023,100,0\n"), 0600)
	w := &transientTestWindow{frame: f, dir: filepath.Dir(input)}
	w.onDraw = func() {
		if f.Arclms["TH"].Running() {
			f.Arclms["TH"].Stop()
		}
	}
	if _, ok := runTimeHistoryCommand(w, input, map[string]string{"RAYLEIGH": "1;.2;.02", "DT": ".001", "WAIT": "", "DRAW": "10"}).(ArclmStart); !ok {
		t.Fatal("command failed")
	}
	if f.Arclms["TH"].Running() || f.Arclms["TH"].TimeHistoryState().Step != 10 || len(w.draws) != 1 {
		t.Fatal("Stop/draw handshake failed")
	}
	if !strings.Contains(strings.Join(w.history, "\n"), "canceled") {
		t.Fatal("Stop error not reported")
	}
	rows := readResponseCSV(t, filepath.Join(w.dir, "wave.response.stories.csv"))
	if len(rows) != 11 {
		t.Fatal("lost partial response output")
	}
}
