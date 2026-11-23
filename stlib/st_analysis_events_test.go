package st

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yofu/st/arclm"
)

type analysisWaitWindow struct {
	ExModer
	pump     func()
	finished int
}

func (w *analysisWaitWindow) Pivot() bool         { return false }
func (w *analysisWaitWindow) Redraw()             {}
func (w *analysisWaitWindow) PumpAnalysisEvents() { w.pump() }
func (w *analysisWaitWindow) AnalysisFinished()   { w.finished++ }

func TestStaticWaitPumpsEventsAndHandlesStopDuringLap(t *testing.T) {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	af := arclm.NewFrame()
	ready := make(chan struct{})
	w := &analysisWaitWindow{}
	w.pump = func() {
		select {
		case <-ready:
		default:
			close(ready)
		}
	}
	go func() {
		<-ready // The first native event must run before the solver can publish.
		af.Lapch <- 1
		<-ctx.Done()
		af.Endch <- ctx.Err() // Solver exited before receiving the LAP acknowledgement.
	}()
	var step int
	var result error
	done := make(chan error, 1)
	go func() {
		done <- monitorStaticAnalysis(w, true, ctx, af, 1, func(lap int) {
			step = lap
			stop()
		}, func(err error) {
			result = err
		})
	}()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) || !errors.Is(result, err) || step != 1 || w.finished != 1 {
			t.Fatalf("wait returned %v, final update %v, step %d, completion signals %d", err, result, step, w.finished)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("analysis wait hung while processing cancellation")
	}
}
