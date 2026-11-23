package st

import (
	"bytes"
	"fmt"
	"github.com/yofu/st/arclm"
	"os"
	"path/filepath"
	"testing"
)

func TestBucklingOutputReadback(t *testing.T) {
	f := NewFrame()
	n := NewNode()
	n.Num = 19
	f.Nodes[n.Num] = n
	af := arclm.NewFrame()
	af.Nodes = []*arclm.Node{{Num: 19}}
	af.EigenValue = []float64{25.69755000525501, 100}
	af.EigenVector = [][]float64{{.3, 0, 0, 0, 0, 0}, {0, 1, 0, 0, 0, 0}}
	var out bytes.Buffer
	if _, err := af.WriteBclngTo(&out); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "modes.otp")
	write := func(data string) {
		t.Helper()
		if err := os.WriteFile(file, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(out.String())
	if err := f.ReadBuckling(file); err != nil {
		t.Fatal(err)
	}
	f.Show.Period = "B1"
	f.Show.PlotState |= PLOT_DEFORMED
	q := GUIReadback{Period: "B1", Nodes: []int{19}, RequireDeformation: true}
	v, err := f.ReadGUIResults(q)
	if err != nil || v.Eigenvalue == nil || *v.Eigenvalue != 25.6976 || v.NonzeroNodes != 1 || v.Displacements[19][0] != .3 || v.Path != file {
		t.Fatalf("readback %+v: %v", v, err)
	}
	for _, bad := range []string{"", "EIGEN VALUE 1=bad", "EIGEN VALUE 1=NaN", "EIGEN VALUE 1=1\nNODE: 99 {dU}= 1 2 3 4 5 6", "EIGEN VALUE 1=1\nNODE: 19 {dU}= 0 0", "EIGEN VALUE 1=9\nNODE: 19 {dU}= 0 0 0 0 0 0\nEIGEN VALUE 2=not-a-number"} {
		write(bad)
		if f.ReadBuckling(file) == nil || f.Eigenvalue[0] != 25.6976 || n.Disp["B1"][0] != .3 {
			t.Fatalf("non-atomic or successful bad read %q", bad)
		}
	}
	if f.ReadBuckling(file+".missing") == nil {
		t.Fatal("missing file succeeded")
	}
	write("EIGEN VALUE 1=2.0\nNODE: 19 {dU}= 0 1 0 0 0 0")
	if err := f.ReadBuckling(file); err != nil || len(f.Eigenvalue) != 1 || n.Disp["B2"] != nil {
		t.Fatalf("stale mode: %v", err)
	}
	for _, p := range []string{"1", "B2"} {
		f.Show.Period = p
		if _, err := f.ReadGUIResults(GUIReadback{Period: p, RequireDeformation: true}); err == nil {
			t.Fatal(fmt.Sprintf("accepted missing %s", p))
		}
	}
}
