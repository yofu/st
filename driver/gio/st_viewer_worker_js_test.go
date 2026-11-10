//go:build js && wasm

package stgio

import (
	"image"
	"os"
	"path/filepath"
	"strings"
	"syscall/js"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/pointer"
	"gioui.org/unit"
)

func TestViewerDedicatedWorker(t *testing.T) {
	host := js.Global().Get("stTestWorkerStart")
	if host.Type() != js.TypeFunction {
		t.Skip("requires the dedicated Worker test host")
	}
	input := os.Getenv("ST_VIEWER_TEST_INP")
	data, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	w := NewModelViewer(nil, nil)
	t.Cleanup(w.ui.close)
	h := &uiHarness{w: w, size: image.Pt(900, 750), metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, now: time.Unix(1, 0)}
	h.frame()
	opened := make(chan error, 1)
	if !w.OpenViewerModel(filepath.Base(input), data, func(err error) { opened <- err }) {
		t.Fatal("open rejected")
	}
	if err := <-opened; err != nil {
		t.Fatal(err)
	}
	waitUI(t, w)
	h.frame()

	events := make(chan string, 16)
	applied := make(chan error, 1)
	finish := js.FuncOf(func(_ js.Value, args []js.Value) any {
		var result []byte
		if !args[1].IsNull() {
			result = make([]byte, args[1].Get("byteLength").Int())
			js.CopyBytesToGo(result, args[1])
		}
		if !w.FinishViewerWorker(args[0].Int(), result, args[2].String(), func(err error) {
			if err == nil && args[2].String() != "" {
				err = os.ErrInvalid
			}
			applied <- err
		}) {
			t.Error("worker completion rejected")
		}
		return nil
	})
	defer finish.Release()
	progress := js.FuncOf(func(_ js.Value, args []js.Value) any {
		select {
		case events <- args[0].String():
		default:
		}
		return nil
	})
	defer progress.Release()
	w.SetViewerAnalysisWorker(func(id int, model []byte) {
		value := js.Global().Get("Uint8Array").New(len(model))
		js.CopyBytesToJS(value, model)
		host.Invoke(id, value, finish, progress)
	}, func(id int) { js.Global().Call("stTestWorkerStop", id) })
	run := func(command string) {
		t.Helper()
		if !w.SubmitViewerCommand(command) {
			t.Fatal("command rejected")
		}
	}
	await := func(timeout time.Duration) error {
		t.Helper()
		select {
		case err := <-applied:
			waitUI(t, w)
			h.frame()
			return err
		case <-time.After(timeout):
			t.Fatalf("Worker did not finish: %s", w.ui.log)
			return nil
		}
	}

	run(":extractarclm worker.wgt")
	if err := await(35 * time.Second); err != nil {
		t.Fatal("extract failed")
	}
	if f := w.frame.Arclms["L"]; f == nil || len(f.Nodes) == 0 {
		t.Fatal("missing extracted analysis state")
	}
	dofs := 0
	for _, n := range w.frame.Arclms["L"].Nodes {
		for _, fixed := range n.Conf {
			if !fixed {
				dofs++
			}
		}
	}
	t.Logf("%s: %d free DOFs", w.frame.Name, dofs)

	if dofs < 4000 {
		for _, command := range []string{
			":analysis -period=L gravity.otl",
			":arclm101 -period=X -nlap=1 -delta=0.0125 pushover.otx",
			":arclm101 -period=X -nlap=1 -delta=0.0125",
		} {
			run(command)
			if err := await(45 * time.Second); err != nil {
				t.Fatalf("%s: %v", command, err)
			}
		}
		if w.frame.Arclms["X"].ContinuationStep() != 2 || w.frame.Show.Period != "X@2" {
			t.Fatal("nonlinear continuation state was lost between Workers")
		}
		if summary, err := os.ReadFile("/workspace/summary.txt"); err != nil || !strings.Contains(string(summary), "COMPLETED STEPS: 2") {
			t.Fatalf("continued pushover summary: %v", err)
		}
		if err := os.WriteFile("/workspace/wave.csv", []byte("time,ax,ay\n0,0,0\n0.01,1,0\n0.02,0,0\n"), 0644); err != nil {
			t.Fatal(err)
		}
		run("'period L")
		waitUI(t, w)
		for _, command := range []string{
			":timehistory wave.csv -rayleigh=0.2;1.0;0.02 -output=worker-response",
			":bclng001 -period=L -mode=1 buckling.otp",
			":vibeig -period=L -mode=1 vibration.otp",
		} {
			run(command)
			if err := await(45 * time.Second); err != nil {
				t.Fatalf("%s: %v", command, err)
			}
		}
		if w.frame.Arclms["TH"].TimeHistoryState().Step != 2 || len(w.frame.Arclms["L"].EigenValue) != 1 {
			t.Fatal("missing time-history or vibration result")
		}
		return
	}

	for len(events) > 0 {
		<-events
	}
	previous := w.frame.Arclms["L"]
	run(":vibeig -mode=6 cancelled.otp")
	timeout := time.After(35 * time.Second)
	for {
		select {
		case phase := <-events:
			if strings.HasPrefix(phase, "Calculating") {
				goto started
			}
		case <-timeout:
			t.Fatal("worker did not report calculation progress")
		}
	}
started:
	a := w.frame.View.Angle[0]
	h.router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(200, 200)})
	h.frame()
	h.router.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(200, 235)})
	h.frame()
	h.router.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(200, 235)})
	h.frame()
	if w.frame.View.Angle[0] == a {
		t.Fatal("view did not respond during analysis")
	}
	start := time.Now()
	run(":stop L")
	if err := await(3 * time.Second); err == nil {
		t.Fatal("stopped analysis reported success")
	}
	if w.frame.Arclms["L"] != previous || w.viewer.workerJob != 0 {
		t.Fatal("stopping discarded previously completed solver state")
	}
	if _, err := os.Stat("/workspace/cancelled.otp"); !os.IsNotExist(err) {
		t.Fatalf("cancelled output was published: %v", err)
	}
	t.Logf("rotation worked during calculation; stop acknowledged in %v", time.Since(start))
	run(":extractarclm restarted.wgt")
	if err := await(35 * time.Second); err != nil {
		t.Fatal("worker did not restart")
	}
}
