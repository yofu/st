package main

import (
	"errors"
	"fmt"
	st "github.com/yofu/st/stlib"
	"io"
	"path/filepath"
	"time"

	"github.com/yofu/st/arclm"
)

type bucklingComparison struct {
	Rank   int     `json:"rank"`
	Before float64 `json:"before_factor"`
	After  float64 `json:"after_factor"`
	Change float64 `json:"change_factor"`
}

type curveComparison struct {
	Nodes     []int     `json:"node_ids"`
	PreloadL  bool      `json:"preload_l"`
	Steps     int       `json:"requested_steps"`
	Increment float64   `json:"increment"`
	Before    loadCurve `json:"before"`
	After     loadCurve `json:"after"`
}

type loadCurve struct {
	Points         []curvePoint         `json:"points"`
	Nonconvergence *curveNonconvergence `json:"nonconvergence,omitempty"`
}

type curvePoint struct {
	Step          int          `json:"step"`
	Factor        float64      `json:"load_factor"`
	Displacements [][6]float64 `json:"displacements"`
}

type curveNonconvergence struct {
	Target        float64 `json:"target_factor"`
	LastConverged float64 `json:"last_converged_factor"`
	Message       string  `json:"message"`
}

func workerBuckling(af *arclm.Frame, dir string, out io.Writer, side string, modes int) error {
	return withPivotProgress(af, out, side, "buckling", func() error {
		return af.Bclng001(filepath.Join(dir, "buckling.otp"), true, modes, 1e-12, 10)
	})
}

func workerArclm101(af *arclm.Frame, dir string, out io.Writer, side string, o analysisOptions) (loadCurve, error) {
	curve := loadCurve{Points: make([]curvePoint, 0, o.Steps)}
	nodes := make(map[int]*arclm.Node, len(af.Nodes))
	for _, node := range af.Nodes {
		nodes[node.Num] = node
	}
	for _, id := range o.NodeIDs {
		if nodes[id] == nil {
			return curve, fmt.Errorf("no node %d", id)
		}
	}
	c := arclm.NewAnalysisCondition()
	c.SetNlap(o.Steps)
	c.SetDelta(o.Increment)
	c.SetInit(!o.PreloadL)
	if o.PreloadL {
		c.SetStart(0)
	}
	c.SetOutput([]string{filepath.Join(dir, "material.otp")})

	output := &lockedWriter{out: out}
	af.Output = progressWriter{output, side, "arclm101"}
	laps := make(chan int)
	af.Lapch = laps
	af.Pivot = nil
	done := make(chan struct{})
	lastPublished := 0
	previewEnabled := side == "after" && o.PreviewEvery > 0
	publish := func(step curvePoint) {
		live := st.CaptureLiveAnalysisStep(af, step.Step, o.Steps, o.Period, o.PreloadL, step.Factor)
		_ = reportWorker(output, analysisEvent{Preview: live})
		lastPublished = step.Step
	}
	go func() {
		defer close(done)
		var lastFrame time.Time
		for step := range laps {
			point := curvePoint{Step: step, Factor: float64(step) * o.Increment, Displacements: make([][6]float64, len(o.NodeIDs))}
			for i, id := range o.NodeIDs {
				point.Displacements[i] = [6]float64(nodes[id].Disp)
			}
			curve.Points = append(curve.Points, point)
			_ = reportWorker(output, analysisEvent{Side: side, Phase: "arclm101", Detail: fmt.Sprintf("converged step %d/%d; load factor %.12g", step, o.Steps, point.Factor),
				Step: &analysisStepProgress{Completed: step, Total: o.Steps, Factor: point.Factor}})
			if previewEnabled && (step == 1 || step == o.Steps || step%o.PreviewEvery == 0 && time.Since(lastFrame) >= 200*time.Millisecond) {
				publish(point)
				lastFrame = time.Now()
			}
			laps <- 0
		}
	}()
	err := func() error {
		defer func() {
			close(laps)
			<-done
			af.Lapch = nil
		}()
		return af.Arclm101(nil, c)
	}()
	var nonconvergence *arclm.PlasticConvergenceError
	if previewEnabled && len(curve.Points) > 0 && (err == nil || errors.As(err, &nonconvergence)) {
		last := curve.Points[len(curve.Points)-1]
		if last.Step != lastPublished {
			publish(last)
		}
	}
	if errors.As(err, &nonconvergence) && len(curve.Points) > 0 {
		curve.Nonconvergence = &curveNonconvergence{Target: nonconvergence.Target, LastConverged: curve.Points[len(curve.Points)-1].Factor, Message: nonconvergence.Error()}
		return curve, nil
	}
	return curve, err
}
