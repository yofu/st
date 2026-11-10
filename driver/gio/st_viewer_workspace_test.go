package stgio

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestViewerWorkspaceAnalysis(t *testing.T) {
	if !browserViewer {
		t.Skip("exercises the browser filesystem")
	}
	input := os.Getenv("ST_VIEWER_TEST_INP")
	if input == "" {
		t.Skip("set ST_VIEWER_TEST_INP and stage the corresponding .inl file")
	}
	data, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	h, state := setupViewerEditor(t)
	w := h.w
	ch := make(chan error, 1)
	reopen := func() {
		t.Helper()
		if !w.OpenViewerModel(filepath.Base(input), data, func(err error) { ch <- err }) {
			t.Fatal("open rejected")
		}
		waitUI(t, w)
		if err := <-ch; err != nil {
			t.Fatal(err)
		}
		h.frame()
	}
	reopen()

	base := strings.TrimSuffix(filepath.Base(input), ".inp")
	viewerEditorCommand(t, h, ":read "+base+".inl")
	if w.frame.Arclms["L"] == nil {
		t.Fatalf("analysis input failed: %s", state.History)
	}
	// An imported INL may contain generated braces and omit nonlinear
	// section data. Extract from the original INP for time history.
	reopen()
	viewerEditorCommand(t, h, ":extractarclm extracted.wgt")
	if contents, err := os.ReadFile("/workspace/extracted.wgt"); err != nil || len(contents) == 0 {
		t.Fatalf("weight output missing: %v, %s", err, state.History)
	}
	viewerEditorCommand(t, h, ":analysis -period=L -wait linear.otl")
	result, err := os.ReadFile("/workspace/linear.otl")
	if err != nil || len(result) < 100 {
		t.Fatalf("linear output missing: %v (%d bytes), %s", err, len(result), state.History)
	}
	solved := 0
	for _, n := range w.frame.Nodes {
		if len(n.Disp["L"]) == 6 {
			solved++
		}
	}
	if solved == 0 {
		t.Fatalf("analysis results were not applied to the canvas model: %s", state.History)
	}
	viewerEditorCommand(t, h, "'deformation L")
	viewerEditorCommand(t, h, ":svg view.svg")
	if contents, err := os.ReadFile("/workspace/view.svg"); err != nil || !strings.Contains(string(contents), "<svg") {
		t.Fatalf("SVG missing: %v, %s", err, state.History)
	}
	viewerEditorCommand(t, h, ":pdf view.pdf")
	if contents, err := os.ReadFile("/workspace/view.pdf"); err != nil || !strings.HasPrefix(string(contents), "%PDF") {
		t.Fatalf("PDF missing: %v, %s", err, state.History)
	}
	if err := os.WriteFile("/workspace/wave.csv", []byte("time,ax,ay\n0,0,0\n0.01,1,0\n0.02,0,0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	viewerEditorCommand(t, h, ":timehistory wave.csv -rayleigh=0.2;1.0;0.02 -output=web-test -wait")
	af := w.frame.Arclms["TH"]
	if af == nil || af.TimeHistoryState() == nil || af.TimeHistoryState().Step != 2 {
		t.Fatalf("time-history steps did not complete: %s", state.History)
	}
	if contents, err := os.ReadFile("/workspace/web-test.nodes.csv"); err != nil || len(contents) < 100 {
		t.Fatalf("time-history CSV missing: %v, %s", err, state.History)
	}
	viewerEditorCommand(t, h, "'period L")
	if !w.SubmitViewerCommand(":arclm101 -period=X -nlap=1 -delta=0.0125 -wait pushover.otx") {
		t.Fatal("pushover command was rejected")
	}
	deadline := time.Now().Add(45 * time.Second)
	for w.ui.pending.Load() > 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	h.frame()
	if contents, err := os.ReadFile("/workspace/summary.txt"); err != nil || len(contents) < 30 || !strings.HasPrefix(state.Progress, "Completed") {
		t.Fatalf("pushover summary missing: %v, %s", err, state.History)
	}

	for _, cmd := range []string{":bclng001 -mode=1 buckling.otp", ":vibeig -mode=1 vibration.otp"} {
		output := filepath.Join("/workspace", strings.Fields(cmd)[2])
		viewerEditorCommand(t, h, cmd)
		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			h.frame()
			if strings.HasPrefix(state.Progress, "Failed") {
				break
			}
			if contents, err := os.ReadFile(output); err == nil && len(contents) > 100 && strings.HasPrefix(state.Progress, "Completed") {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		if contents, err := os.ReadFile(output); err != nil || len(contents) < 100 || !strings.HasPrefix(state.Progress, "Completed") {
			t.Fatalf("%s failed: %v, %s", cmd, err, state.History)
		}
	}
	t.Logf("Wasm workspace: static displacement on %d nodes, %d-byte OTL, SVG, PDF, 2 time-history steps, pushover, buckling and vibration", solved, len(result))
}
