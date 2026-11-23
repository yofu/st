package st

import (
	"context"
	"sort"
	"time"

	"github.com/yofu/st/arclm"
)

// A nil tick disables GUI event dispatch in asynchronous and headless runs.
type analysisEventPump struct {
	tick <-chan time.Time
	pump func()
}

func monitorAnalysis(stw ExModer, wait bool, monitor func(analysisEventPump) error) error {
	if !wait {
		go func() {
			defer workerAnalysisFinished(stw)
			monitor(analysisEventPump{})
		}()
		return nil
	}
	defer workerAnalysisFinished(stw)
	// In synchronous IUP commands the caller owns the GUI thread. Apply LAP
	// updates here, and service native events even while a solver step is long.
	if w, ok := stw.(interface{ PumpAnalysisEvents() }); ok {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		return monitor(analysisEventPump{ticker.C, w.PumpAnalysisEvents})
	}
	return monitor(analysisEventPump{})
}

func monitorStaticAnalysis(stw ExModer, wait bool, ctx context.Context, af *arclm.Frame, total int, readLap func(int), finish func(error)) error {
	var nextPivot, endPivots func()
	if stw.Pivot() {
		var nodes []*Node
		for _, n := range stw.Frame().Nodes {
			nodes = append(nodes, n)
		}
		sort.Sort(NodeByNum{nodes})
		if w, ok := stw.(interface{ BeginAnalysisPivots([]*Node) func() }); ok {
			nextPivot = w.BeginAnalysisPivots(nodes)
		} else {
			pivot, end := make(chan int), make(chan int)
			go stw.DrawPivot(nodes, pivot, end)
			nextPivot = func() { pivot <- 1 }
			endPivots = func() { end <- 1 }
		}
	} else {
		stw.Redraw()
	}
	return monitorAnalysis(stw, wait, func(events analysisEventPump) error {
		var err error
		for {
			select {
			case <-events.tick:
				events.pump()
				continue
			case <-af.Pivot:
				if nextPivot != nil {
					nextPivot()
				}
				continue
			case step := <-af.Lapch:
				readLap(step)
				ack := 0
				if ctx.Err() != nil {
					ack = 1
				}
				// A nonlinear solver can stop while the GUI presents the LAP.
				select {
				case af.Lapch <- ack:
					// The view now owns a copy of this step. Let the solver
					// compute the next one while it is being drawn.
					stw.CurrentLap("Calculating...", step, total)
					stw.Redraw()
					continue
				case err = <-af.Endch:
				}
			case err = <-af.Endch:
			}
			if endPivots != nil {
				endPivots()
			}
			finish(err)
			return err
		}
	})
}
