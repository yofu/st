package st

import (
	"encoding/csv"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

func TestTimeHistoryAutomaticRayleighCommand(t *testing.T) {
	f := continuationCommandFrame(t)
	f.Arclms["X"].Nodes[1].Conf[1] = false
	runContinuationCommand(t, f, map[string]string{"INIT": "", "STEP": "1;1"})
	f.Arclms["L"] = f.Arclms["X"]
	f.Arclms["L"].Nodes[1].Mass = 1
	saved := f.Arclms["L"].SaveState()
	input := filepath.Join(t.TempDir(), "wave.csv")
	if err := os.WriteFile(input, []byte("time,X,Y\n0,0,0\n.02,10,20\n.04,0,0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	w := &transientTestWindow{frame: f, dir: filepath.Dir(input)}
	err := runTimeHistoryCommand(w, input, map[string]string{"RAYLEIGH": ".02", "DT": ".01", "WAIT": ""})
	if _, ok := err.(ArclmStart); !ok {
		t.Fatal(err)
	}
	state := f.Arclms["TH"].TimeHistoryState()
	if state.RayleighModes != [2]int{1, 2} || f.Show.Period != "TH@4" {
		t.Fatalf("modes=%v display=%s", state.RayleighModes, f.Show.Period)
	}
	for _, period := range state.RayleighPeriods {
		omega := 2 * math.Pi / period
		if math.Abs(state.RayleighAlpha/(2*omega)+state.RayleighEta*omega/2-.02) > 1e-12 {
			t.Fatal("incorrect fitted damping")
		}
	}
	if !reflect.DeepEqual(saved, f.Arclms["L"].SaveState()) {
		t.Fatal("automatic modal analysis changed L")
	}
	file, err := os.Open(filepath.Join(w.dir, "wave.response.steps.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if rows[0][13] != "rayleigh_mode1" || rows[0][15] != "rayleigh_period1_s" || rows[1][13] != "1" || rows[1][14] != "2" {
		t.Fatal("missing modal selection in step CSV")
	}
	period, err := strconv.ParseFloat(rows[1][15], 64)
	if err != nil || period != state.RayleighPeriods[0] {
		t.Fatal("CSV period mismatch")
	}
}

func TestTimeHistoryAutomaticRayleighOptions(t *testing.T) {
	f := continuationCommandFrame(t)
	f.Arclms["L"] = f.Arclms["X"]
	input := filepath.Join(t.TempDir(), "wave.csv")
	os.WriteFile(input, []byte("0,0,0\n.1,0,0\n"), 0600)
	for _, args := range []map[string]string{{"RAYLEIGH": ".02"}, {"RAYLEIGH": ".02;.03", "MODES": "1;3"}} {
		_, _, o, err := prepareTimeHistory(f, input, args, filepath.Dir(input))
		if err != nil || !o.AutoRayleigh || o.Period1 != 0 || o.Period2 != 0 || o.Damping1 != .02 {
			t.Fatalf("options=%+v err=%v", o, err)
		}
	}
	for _, args := range []map[string]string{
		{"RAYLEIGH": "-.01"}, {"RAYLEIGH": "NaN"}, {"RAYLEIGH": ".02", "MODES": "2;1"}, {"RAYLEIGH": ".02", "MODES": "0;2"}, {"RAYLEIGH": ".02", "MODES": "1"}, {"RAYLEIGH": "1;.2;.02", "MODES": "1;3"},
	} {
		if _, _, _, err := prepareTimeHistory(f, input, args, filepath.Dir(input)); err == nil {
			t.Fatalf("accepted invalid options %v", args)
		}
	}
}
