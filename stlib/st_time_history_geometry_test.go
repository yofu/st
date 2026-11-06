package st

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestTimeHistoryGeometryCommand(t *testing.T) {
	f := continuationCommandFrame(t)
	runContinuationCommand(t, f, map[string]string{"INIT": "", "NLGEOM": "", "STEP": "1;1"})
	f.Arclms["L"] = f.Arclms["X"]
	f.Arclms["L"].Nodes[1].Mass = 1
	before := f.Arclms["L"].SaveState()
	dir := t.TempDir()
	input := filepath.Join(dir, "wave.csv")
	if err := os.WriteFile(input, []byte("0,0,0\n.02,100,200\n.04,0,0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	w := &transientTestWindow{frame: f, dir: dir}
	err := runTimeHistoryCommand(w, input, map[string]string{"RAYLEIGH": "1;.2;.02", "DT": ".01", "NLGEOM": "", "WAIT": "", "NODES": "2"})
	if _, ok := err.(ArclmStart); !ok {
		t.Fatal(err)
	}
	s := f.Arclms["TH"].TimeHistoryState()
	if s == nil || s.Step != 4 || !s.NonlinearGeometry || f.Show.Period != "TH@4" {
		t.Fatalf("state=%+v messages=%v", s, w.history)
	}
	if !reflect.DeepEqual(before, f.Arclms["L"].SaveState()) {
		t.Fatal("L modified")
	}
}
