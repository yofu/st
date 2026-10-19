package st

import (
	"encoding/csv"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/yofu/st/arclm"
)

type transientTestWindow struct {
	ExModer
	frame   *Frame
	dir     string
	last    int
	draws   []string
	onDraw  func()
	history []string
}

func (w *transientTestWindow) Frame() *Frame                 { return w.frame }
func (w *transientTestWindow) Cwd() string                   { return w.dir }
func (w *transientTestWindow) HistoryWriter() io.Writer      { return io.Discard }
func (w *transientTestWindow) History(s string)              { w.history = append(w.history, s) }
func (w *transientTestWindow) CurrentLap(_ string, n, _ int) { w.last = n }
func (w *transientTestWindow) SetLabel(string, string)       {}
func (w *transientTestWindow) Redraw() {
	w.draws = append(w.draws, w.frame.Show.Period)
	if w.onDraw != nil {
		w.onDraw()
	}
}
func (w *transientTestWindow) EnableLabel(string) {}

func TestTimeHistoryCommandCSVPreloadAndDisplay(t *testing.T) {
	f := continuationCommandFrame(t)
	runContinuationCommand(t, f, map[string]string{"INIT": "", "STEP": "1;1"})
	f.Arclms["L"] = f.Arclms["X"]
	f.Arclms["L"].Nodes[1].Mass = 1
	input := filepath.Join(t.TempDir(), "wave.csv")
	if err := os.WriteFile(input, []byte("time[s],X[cm/s2],Y[cm/s2]\n0,0,0\n.02,100,200\n.04,0,0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	saved := f.Arclms["L"].SaveState()
	w := &transientTestWindow{frame: f, dir: filepath.Dir(input)}
	err := runTimeHistoryCommand(w, input, map[string]string{"RAYLEIGH": "1;.2;.02", "DT": ".01", "WAIT": "", "NODES": "2", "SAVE": "2"})
	if _, ok := err.(ArclmStart); !ok {
		t.Fatal(err)
	}
	if f.Show.Period != "TH@4" || f.Nlap["TH"] != 4 || w.last != 4 {
		t.Fatalf("display=%s lap=%d", f.Show.Period, w.last)
	}
	if f.Arclms["TH"] == f.Arclms["L"] || !reflect.DeepEqual(saved, f.Arclms["L"].SaveState()) {
		t.Fatal("L state modified")
	}
	if f.Arclms["TH"].Running() || f.Arclms["TH"].TimeHistoryState().Ground != [2]float64{0, 0} {
		t.Fatal("run not complete")
	}
	file, err := os.Open(filepath.Join(filepath.Dir(input), "wave.response.nodes.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 6 || len(rows[0]) != 33 || rows[1][2] != "2" {
		t.Fatalf("response output rows=%d", len(rows))
	}
	// 1 m/s^2 and 2 m/s^2 are the input at t=.02; y is constrained,
	// hence its relative acceleration is zero and absolute acceleration is 2.
	var checked bool
	for _, row := range rows[1:] {
		if row[1] == "0.02" {
			if row[22] != "2" {
				t.Fatalf("Y absolute acceleration=%s", row[22])
			}
			checked = true
		}
	}
	if !checked {
		t.Fatal("missing .02 output time")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(input), "wave.response@4.otp")); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := prepareTimeHistory(f, input, map[string]string{"RAYLEIGH": "1;.2;.02"}, w.dir); err == nil {
		t.Fatal("accepted overwriting prior CSVs")
	}
}

func TestTimeHistoryCommandValidation(t *testing.T) {
	f := continuationCommandFrame(t)
	f.Arclms["L"] = f.Arclms["X"]
	input := filepath.Join(t.TempDir(), "wave.csv")
	os.WriteFile(input, []byte("0,0,0\n.1,10,0\n"), 0600)
	saved := f.Arclms["L"].SaveState()
	for _, args := range []map[string]string{
		{}, {"RAYLEIGH": "1;1;.02"}, {"RAYLEIGH": "1;.2;.02", "DT": "0"}, {"RAYLEIGH": "1;.2;.02", "NODES": "999"}, {"RAYLEIGH": "1;.2;.02", "SAVE": "-1"}, {"RAYLEIGH": "1;.2;.02", "PERIOD": "TH"},
	} {
		if _, _, _, err := prepareTimeHistory(f, input, args, filepath.Dir(input)); err == nil {
			t.Fatalf("accepted %v", args)
		}
		if !reflect.DeepEqual(saved, f.Arclms["L"].SaveState()) {
			t.Fatal("invalid command modified preload")
		}
	}
	af, m, o, err := prepareTimeHistory(f, input, map[string]string{"RAYLEIGH": "1;.2;.02;.03"}, filepath.Dir(input))
	if err != nil || o.Damping2 != .03 || m[1].X != .1 || af == f.Arclms["L"] {
		t.Fatalf("valid options: %+v %v", o, err)
	}
	// The API rejects unavailable load provenance; the command does not invent
	// a nodal load vector from stresses in an old imported result.
	if err = af.TimeHistoryNewmark(nil, m, arclm.TimeHistoryOptions{}); err == nil {
		t.Fatal("accepted unknown nonzero preload")
	}
}
