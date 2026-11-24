package stgui

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/yofu/go-iup/iup"
	st "github.com/yofu/st/stlib"
)

// Exercise the native IUP undo baseline on a private frame; no HTTP service
// or connection to a running user's window is needed.
func TestRCYieldIUPUndo(t *testing.T) {
	path := os.Getenv("ST_STEEL_YIELD_TEST_INP")
	if path == "" {
		t.Skip("set ST_STEEL_YIELD_TEST_INP for native IUP test")
	}
	dir := t.TempDir()
	t.Chdir(dir)
	f := st.NewFrame()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.ReadInpSnapshot(data, path); err != nil {
		t.Fatal(err)
	}
	if err = f.ReadLst(st.Ce(path, ".lst")); err != nil {
		t.Fatal(err)
	}
	f.Path = filepath.Join(dir, "private.inp")
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := iup.Open(); err != nil {
		t.Fatal(err)
	}
	defer iup.Close()
	w := NewWindow(dir)
	defer StopLogging()
	defer w.Dlg.Destroy()
	w.SetFrame(f)
	w.EnableUndo()
	before := slices.Clone(f.Sects[301].Yield)
	result := st.ExMode(w, ":rcyield -elem=3680 -qy=865.502,1014.57")
	if result != nil || !strings.Contains(w.hist.GetAttribute("VALUE"), "preview_id=") || !slices.Equal(before, f.Sects[301].Yield) {
		t.Fatal(result)
	}
	result = st.ExMode(w, ":rcyield -apply")
	if result != nil || f.Sects[301].Yield[4] != 360.528 {
		t.Fatal(result)
	}
	after := slices.Clone(f.Sects[301].Yield)
	for _, step := range []struct {
		action string
		want   []float64
	}{{"undo", before}, {"redo", after}} {
		v := w.handleGUIRequest(st.GUIRequest{Action: step.action})
		if v.Error != "" || !slices.Equal(w.Frame().Sects[301].Yield, step.want) {
			t.Fatalf("%s: %s %v", step.action, v.Error, w.Frame().Sects[301].Yield)
		}
	}
}
