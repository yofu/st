package stgui

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yofu/go-iup/iup"
	st "github.com/yofu/st/stlib"
)

// Uses the normal MCP GUI command queue and a mapped native window, without
// attaching to an existing user's ST or starting an HTTP server.
func testNativeAnalysisWait(t *testing.T, w *Window, dir string) {
	path := filepath.Join(dir, "model.inp")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	r := &guiCommandRunner{jobs: make(map[uint64]*guiCommandJob)}
	w.guiConnection = &guiConnection{commands: r}
	run := func(commands ...string) *st.GUICommandStatus {
		t.Helper()
		reply := w.startGUICommands(st.GUIRequest{Commands: commands}, "")
		if reply.Error != "" {
			t.Fatal(reply.Error)
		}
		deadline := time.Now().Add(20 * time.Second)
		for time.Now().Before(deadline) {
			iup.LoopStep()
			w.advanceGUICommands()
			job := r.response(reply.CommandJob.ID, 0, true).CommandJob
			if job.Completion != 0 {
				return job
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatalf("GUI workflow did not finish: %v", commands)
		return nil
	}
	var baseline float64
	var f *st.Frame
	for _, wait := range []bool{false, true} {
		// Extraction adds analysis members. Start both runs from the same INP.
		f = st.NewFrame()
		if err := f.ReadInpSnapshot(data, path); err != nil {
			t.Fatal(err)
		}
		f.Path = path
		if st.FileExists(st.Ce(path, ".lst")) {
			if err := f.ReadLst(st.Ce(path, ".lst")); err != nil {
				t.Fatal(err)
			}
		}
		w.SetFrame(f)
		if !wait {
			w.Dlg.Show()
		}
		name, flags := "async.otl", ""
		if wait {
			name, flags = "wait.otl", "-wait "
		}
		f.Show.PlotState &^= st.PLOT_DEFORMED
		start := time.Now()
		job := run(":extractarclm %<.wgt", ":analysis -period=L "+flags+name, "'deformation L")
		if job.State != "completed" || job.Completed != 3 || f.Show.Period != "L" || f.Show.PlotState&st.PLOT_DEFORMED == 0 {
			t.Fatalf("wait=%v: workflow %s (%s), completed %d, display %s", wait, job.State, job.Error, job.Completed, f.Show.Period)
		}
		if info, err := os.Stat(filepath.Join(dir, name)); err != nil || info.Size() == 0 {
			t.Fatalf("wait=%v: missing output %s: %v", wait, name, err)
		}
		node := f.Nodes[131]
		dz := node.ReturnDisp("L", 2)
		if math.IsNaN(dz) || math.Abs(dz) < 1e-6 || (wait && math.Abs(dz-baseline) > 1e-12) {
			t.Fatalf("wait=%v: node 131 dZ=%g, baseline=%g", wait, dz, baseline)
		}
		if node.Pcoord[0] == node.Dcoord[0] && node.Pcoord[1] == node.Dcoord[1] {
			t.Fatalf("wait=%v: node 131 was not projected at its deformed position", wait)
		}
		baseline = dz
		t.Logf("wait=%v: extraction, analysis and deformed display in %s; node 131 dZ=%.12g m", wait, time.Since(start), dz)
	}
	if err := os.WriteFile("wave.csv", []byte("0,0,0\n.01,1,0\n.02,0,0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	job := run(":timehistory wave.csv -rayleigh=1;.2;.02 -draw=1 -wait", "'dfact 80")
	if job.State != "completed" || job.Step != 2 || f.Show.Period != "TH@2" || f.Show.Dfact != 80 {
		t.Fatalf("time history wait: %s (%s), step %d, period %s", job.State, job.Error, job.Step, f.Show.Period)
	}
	job = run(":analysis -period=L -wait missing/result.otl", "'dfact 999")
	if job.State != "failed" || job.Completed != 0 || len(job.Timing) != 1 || f.Show.Dfact == 999 {
		t.Fatalf("solver failure did not stop the workflow: %s (%s), completed %d", job.State, job.Error, job.Completed)
	}
	job = run(":arclm101 -init -wait -period=L -step=2;0.0001;0;1 incremental.otl", "'dfact 81")
	if job.State != "completed" || job.Completed != 2 || job.Step != 2 || f.Show.Period != "L@2" || f.Show.Dfact != 81 {
		t.Fatalf("incremental wait: %s (%s), completed %d, step %d, period %s", job.State, job.Error, job.Completed, job.Step, f.Show.Period)
	}
	// A native timer must still run *inside* the synchronous command. Request
	// cancellation just as an MCP client would, and try a reentrant queue tick.
	timer := iup.Timer("TIME=15")
	defer timer.Destroy()
	stopped := false
	if err := timer.SetCallback(func(*iup.TimerAction) {
		j := r.active
		if !stopped && j != nil && j.executing && j.status.Step >= 1 {
			stopped = true
			r.cancel(j.status.ID)
			w.advanceGUICommands()
		}
	}); err != nil {
		t.Fatal(err)
	}
	enableTimerAction(timer)
	timer.SetAttribute("RUN", "YES")
	job = run(":arclm101 -init -wait -period=L -step=1000;0.0001;0;1 cancel.otl", "'dfact 999")
	timer.SetAttribute("RUN", "NO")
	if !stopped || job.State != "cancelled" || len(job.Timing) != 1 || f.Show.Dfact != 81 {
		t.Fatalf("cancel during wait: stopped=%v, %s (%s), stages %d", stopped, job.State, job.Error, len(job.Timing))
	}
}
