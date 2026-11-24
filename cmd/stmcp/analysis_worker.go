package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/yofu/st/arclm"
)

func runAnalysisWorker(dir string, in io.Reader, out io.Writer) (err error) {
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("analysis worker: %v", failure)
		}
	}()
	var request analysisRequest
	if err := json.NewDecoder(in).Decode(&request); err != nil {
		return err
	}
	if request.TimeHistory != nil {
		return workerTimeHistory(dir, request.After, out, *request.TimeHistory)
	}
	o := request.Options
	result := &analysisResult{ProposalID: o.ProposalID, Period: o.Period}
	if o.Analysis == "static" || o.Analysis == "both" {
		for _, id := range o.NodeIDs {
			result.Nodes = append(result.Nodes, nodeComparison{Node: id})
		}
	}
	if o.Analysis == "arclm101" {
		result.Curves = &curveComparison{Nodes: o.NodeIDs, PreloadL: o.PreloadL, Steps: o.Steps, Increment: o.Increment}
	}
	models := [2][]byte{request.Before, request.After}
	order := []int{0, 1}
	if request.SingleStatic {
		result.Static = &staticAnalysisResult{}
		order = []int{1}
	}
	if o.Analysis == "arclm101" && o.PreviewEvery > 0 {
		// Start with the visible model so its first converged step can be
		// shown without waiting for the baseline comparison.
		order = []int{1, 0}
	}
	for _, i := range order {
		data := models[i]
		side := []string{"before", "after"}[i]
		phase := func(name string) {
			_ = reportWorker(out, analysisEvent{Side: side, Phase: name})
		}
		phase("reading_model")
		f, err := readModel(data, filepath.Join(dir, "model.inp"))
		if err != nil {
			return fmt.Errorf("%s: %w", side, err)
		}
		phase("extracting_loads_and_mass")
		if o.Period == "L" && f.Ai.Nfloor == 0 && len(f.Ai.Boundary) == 0 && len(f.Nodes) > 0 {
			if err := f.SetBoundary(1, 0.001); err != nil {
				return fmt.Errorf("%s boundary setup: %w", side, err)
			}
		}
		if err := f.ExtractArclm(filepath.Join(dir, "model.wgt")); err != nil {
			return fmt.Errorf("%s load and mass extraction: %w", side, err)
		}
		af := f.Arclms[o.Period]
		if af == nil {
			return fmt.Errorf("%s: missing period %s", side, o.Period)
		}
		af.Lapch = nil
		if o.Analysis == "static" || o.Analysis == "both" {
			phase("static")
			if err := workerStatic(af, dir, out, side); err != nil {
				return fmt.Errorf("%s static analysis: %w", side, err)
			}
			nodes := make(map[int]*arclm.Node, len(af.Nodes))
			for _, node := range af.Nodes {
				nodes[node.Num] = node
			}
			for j := range result.Nodes {
				c := &result.Nodes[j]
				n := nodes[c.Node]
				if n == nil {
					return fmt.Errorf("%s static analysis: no node %d", side, c.Node)
				}
				response := nodeResponse{[6]float64(n.Disp), [6]float64(n.Reaction), [6]bool(n.Conf)}
				if request.SingleStatic {
					result.Static.Nodes = append(result.Static.Nodes, staticNodeResult{Node: c.Node, nodeResponse: response})
				} else if i == 0 {
					c.Before = response
				} else {
					c.After = response
					for k := range c.DisplacementChange {
						c.DisplacementChange[k] = c.After.Displacement[k] - c.Before.Displacement[k]
						c.ReactionChange[k] = c.After.Reaction[k] - c.Before.Reaction[k]
					}
				}
			}
		}
		if o.Analysis == "vibration" || o.Analysis == "both" {
			phase("vibration")
			af.Output = progressWriter{out, side, "vibration"}
			if err := af.VibrationalEigenAnalysis(filepath.Join(dir, "modal.otp"), true, o.Modes, 1e-12, 10); err != nil {
				return fmt.Errorf("%s vibration analysis: %w", side, err)
			}
			for j, omega2 := range af.EigenValue {
				period := 2 * math.Pi / math.Sqrt(omega2)
				if i == 0 {
					result.Periods = append(result.Periods, periodComparison{Rank: j + 1, Before: period})
				} else {
					c := &result.Periods[j]
					c.After, c.Change = period, period-c.Before
				}
			}
		}
		switch o.Analysis {
		case "buckling":
			phase("buckling")
			if err := workerBuckling(af, dir, out, side, o.Modes); err != nil {
				return fmt.Errorf("%s Bclng001: %w", side, err)
			}
			for j, value := range af.EigenValue {
				if i == 0 {
					result.Buckling = append(result.Buckling, bucklingComparison{Rank: j + 1, Before: value})
				} else {
					c := &result.Buckling[j]
					c.After, c.Change = value, value-c.Before
				}
			}
		case "arclm101":
			if o.PreloadL {
				phase("preload_l")
				preload := f.Arclms["L"]
				if err := workerStaticPhase(preload, dir, out, side, "preload_l"); err != nil {
					return fmt.Errorf("%s L preload: %w", side, err)
				}
				af.RestoreState(preload.SaveState())
			}
			phase("arclm101")
			curve, err := workerArclm101(af, dir, out, side, o)
			if err != nil {
				return fmt.Errorf("%s Arclm101: %w", side, err)
			}
			if i == 0 {
				result.Curves.Before = curve
			} else {
				result.Curves.After = curve
			}
		}
	}
	if request.SingleStatic {
		result.Nodes = nil
	}
	return reportWorker(out, analysisEvent{Result: result})
}

type progressWriter struct {
	out         io.Writer
	side, phase string
}

func (w progressWriter) Write(data []byte) (int, error) {
	err := reportWorker(w.out, analysisEvent{Side: w.side, Phase: w.phase, Detail: strings.TrimSpace(string(data))})
	return len(data), err
}

type lockedWriter struct {
	mu  sync.Mutex
	out io.Writer
}

func (w *lockedWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.out.Write(data)
}

func workerStatic(af *arclm.Frame, dir string, out io.Writer, side string) error {
	return workerStaticPhase(af, dir, out, side, "static")
}

func workerStaticPhase(af *arclm.Frame, dir string, out io.Writer, side, phase string) error {
	af.Lapch = nil
	c := arclm.NewAnalysisCondition()
	c.SetOutput([]string{filepath.Join(dir, "static.otl")})
	return withPivotProgress(af, out, side, phase, func() error { return af.StaticAnalysis(nil, c) })
}

func withPivotProgress(af *arclm.Frame, out io.Writer, side, phase string, solve func() error) error {
	output := &lockedWriter{out: out}
	log := progressWriter{output, side, phase}
	af.Output = log
	free := 0
	for _, node := range af.Nodes {
		for _, fixed := range node.Conf {
			if !fixed {
				free++
			}
		}
	}
	pivots := make(chan int, 64)
	af.Pivot = pivots
	done := make(chan struct{})
	go func() {
		defer close(done)
		var last time.Time
		for pivot := range pivots {
			if pivot+1 == free || time.Since(last) >= 250*time.Millisecond {
				last = time.Now()
				fmt.Fprintf(log, "stiffness factorization pivot %d/%d", pivot+1, free)
			}
		}
	}()
	defer func() {
		close(pivots)
		<-done
		af.Pivot = nil
	}()
	return solve()
}

func analysisWorkerMain(dir string, out io.Writer) {
	if err := runAnalysisWorker(dir, os.Stdin, out); err != nil {
		_ = reportWorker(out, analysisEvent{Error: err.Error()})
		os.Exit(1)
	}
}
