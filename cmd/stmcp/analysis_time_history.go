package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yofu/st/arclm"
	st "github.com/yofu/st/stlib"
)

type timeHistoryRequest struct {
	Proposal     uint64             `json:"proposal_id"`
	Wave         string             `json:"wave"`
	OutputDir    string             `json:"output_dir"`
	Damping      *float64           `json:"damping_ratio"`
	Modes        [2]int             `json:"rayleigh_modes"`
	DT           float64            `json:"dt"`
	Nodes        []int              `json:"node_ids"`
	NLGeom       bool               `json:"nlgeom"`
	PreviewEvery int                `json:"preview_every"`
	Motion       arclm.GroundMotion `json:"motion,omitempty"`
}

type timeHistoryResult struct {
	Steps           int        `json:"steps"`
	Time            float64    `json:"time_seconds"`
	RayleighPeriods [2]float64 `json:"rayleigh_periods_seconds"`
	Files           []string   `json:"files"`
}

func (s *modelSession) startTimeHistory(o timeHistoryRequest) (any, error) {
	data, _, err := s.editBase(o.Proposal)
	if err != nil {
		return nil, err
	}
	if o.Damping == nil || *o.Damping < 0 || o.DT < 0 || o.PreviewEvery < 0 || o.PreviewEvery > 10000 {
		return nil, fmt.Errorf("provide a nonnegative damping_ratio and valid dt and preview_every")
	}
	if o.Modes != [2]int{} && (o.Modes[0] < 1 || o.Modes[1] <= o.Modes[0] || o.Modes[1] > 100) {
		return nil, fmt.Errorf("use ascending Rayleigh mode numbers from 1 to 100")
	}
	if filepath.Ext(o.Wave) == "" || !strings.EqualFold(filepath.Ext(o.Wave), ".csv") {
		return nil, fmt.Errorf("expected a root-relative ground motion CSV")
	}
	wave, err := s.readEditInput(o.Wave)
	if err != nil {
		return nil, err
	}
	o.Motion, err = arclm.ReadGroundMotionCSV(bytes.NewReader(wave))
	if err != nil {
		return nil, err
	}
	if transientSteps(o) > 100000 {
		return nil, fmt.Errorf("too many requested time steps")
	}
	if len(o.Nodes) > 200 {
		return nil, fmt.Errorf("at most 200 node_ids")
	}
	model, err := readModel(data, s.parsePath)
	if err != nil {
		return nil, err
	}
	seen := map[int]bool{}
	for _, id := range o.Nodes {
		if model.Nodes[id] == nil || seen[id] {
			return nil, fmt.Errorf("unknown or duplicate node %d", id)
		}
		seen[id] = true
	}
	if s.analysis != nil {
		select {
		case <-s.analysis.done:
		default:
			return nil, fmt.Errorf("job already running")
		}
	}
	if !filepath.IsLocal(o.OutputDir) || filepath.Clean(o.OutputDir) == "." {
		return nil, fmt.Errorf("output_dir must name a new directory within the model root")
	}
	path, err := filepath.Abs(filepath.Join(s.root.Name(), o.OutputDir))
	if err != nil {
		return nil, err
	}
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	var out *liveAnalysisOutput
	if o.PreviewEvery > 0 {
		out, err = newLiveAnalysisOutput(s.root, o.OutputDir, data)
	} else {
		err = s.root.Mkdir(o.OutputDir, 0700)
		if err == nil {
			var root *os.Root
			root, err = s.root.OpenRoot(o.OutputDir)
			if err == nil {
				out = &liveAnalysisOutput{root: root}
			}
		}
	}
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.nextJob++
	job := &analysisJob{cancel: cancel, done: make(chan struct{}), started: time.Now(), preview: out, status: analysisStatus{JobID: s.nextJob, State: "running", Phase: "starting", OutputDir: path}}
	if o.PreviewEvery > 0 {
		job.status.PreviewDir = path
	}
	s.analysis = job
	status := job.snapshot()
	go job.run(ctx, exe, filepath.Dir(s.parsePath), analysisRequest{After: data, TimeHistory: &o})
	return status, nil
}

func transientSteps(o timeHistoryRequest) int {
	if o.DT == 0 {
		return len(o.Motion) - 1
	}
	total := math.Ceil(o.Motion[len(o.Motion)-1].Time / o.DT)
	if total > 100000 {
		return 100001
	}
	if total > 1 && (total-1)*o.DT >= o.Motion[len(o.Motion)-1].Time {
		total--
	}
	return int(total)
}

func workerTimeHistory(dir string, data []byte, out io.Writer, o timeHistoryRequest) (err error) {
	emit := func(e analysisEvent) error { e.Side = "after"; e.Phase = "time_history"; return reportWorker(out, e) }
	f, err := readModel(data, filepath.Join(dir, "model.inp"))
	if err != nil {
		return err
	}
	if err = f.ExtractArclm(filepath.Join(dir, "model.wgt")); err != nil {
		return err
	}
	af := f.Arclms["L"]
	if af == nil {
		return fmt.Errorf("no long-term load case")
	}
	if err = workerStaticPhase(af, dir, out, "after", "preload_l"); err != nil {
		return fmt.Errorf("L preload: %w", err)
	}
	output := filepath.Join(dir, "response")
	var floors *st.TimeHistoryFloorOutput
	if f.Ai != nil && f.Ai.Nfloor >= 2 {
		floors, err = st.NewTimeHistoryFloorOutput(f, af, output)
		if err != nil {
			return err
		}
	}
	options := arclm.TimeHistoryOptions{NonlinearGeometry: o.NLGeom, AutoRayleigh: true, Damping1: *o.Damping, Damping2: *o.Damping,
		RayleighModes: o.Modes, TimeStep: o.DT, EquilibratePreload: true, Output: output, OutputNodes: o.Nodes}
	af.Output = progressWriter{out, "after", "time_history"}
	total := transientSteps(o)
	var last time.Time
	var final *arclm.TimeHistoryState
	options.OnStep = func(state *arclm.TimeHistoryState) error {
		if floors != nil {
			if err := floors.Write(state); err != nil {
				return err
			}
		}
		final = state
		sec := state.Time
		if err := emit(analysisEvent{Step: &analysisStepProgress{Completed: state.Step, Total: total, Time: &sec}}); err != nil {
			return err
		}
		if o.PreviewEvery > 0 && state.Step > 0 && (state.Step == 1 || state.Step == total || state.Step%o.PreviewEvery == 0 && time.Since(last) >= 200*time.Millisecond) {
			shot := st.CaptureLiveAnalysisStep(af, state.Step, total, "TH", false, 0)
			shot.Time = &sec
			if err := emit(analysisEvent{Preview: shot}); err != nil {
				return err
			}
			last = time.Now()
		}
		return nil
	}
	err = af.TimeHistoryNewmark(nil, o.Motion, options)
	if floors != nil {
		err = errors.Join(err, floors.Close())
	}
	if err != nil {
		return err
	}
	files := []string{"response.steps.csv", "response.nodes.csv"}
	if floors != nil {
		files = append(files, "response.floors.csv", "response.stories.csv", "response.peaks.csv")
	}
	return reportWorker(out, analysisEvent{Result: &analysisResult{ProposalID: o.Proposal, Period: "TH", TimeHistory: &timeHistoryResult{Steps: final.Step, Time: final.Time, RayleighPeriods: final.RayleighPeriods, Files: files}}})
}

func (out *liveAnalysisOutput) copyResponses(dir string, names []string) error {
	var written []string
	for _, name := range names {
		err := out.write(name, func(dst io.Writer) error {
			src, err := os.Open(filepath.Join(dir, name))
			if err != nil {
				return err
			}
			defer src.Close()
			_, err = io.Copy(dst, src)
			return err
		})
		if err != nil {
			for _, file := range written {
				err = errors.Join(err, out.root.Remove(file))
			}
			return err
		}
		written = append(written, name)
	}
	return nil
}
