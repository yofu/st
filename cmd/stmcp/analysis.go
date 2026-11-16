package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	st "github.com/yofu/st/stlib"
)

type analysisOptions struct {
	ProposalID   uint64  `json:"proposal_id"`
	Analysis     string  `json:"analysis"`
	Period       string  `json:"period"`
	NodeIDs      []int   `json:"node_ids"`
	Modes        int     `json:"modes"`
	Steps        int     `json:"steps"`
	Increment    float64 `json:"increment"`
	PreloadL     bool    `json:"preload_l"`
	PreviewDir   string  `json:"preview_dir"`
	PreviewEvery int     `json:"preview_every"`
}

type analysisRequest struct {
	Options       analysisOptions
	Before, After []byte
	TimeHistory   *timeHistoryRequest
	SingleStatic  bool
}

type nodeResponse struct {
	Displacement [6]float64 `json:"displacement"`
	Reaction     [6]float64 `json:"reaction"`
	Fixed        [6]bool    `json:"fixed"`
}

type nodeComparison struct {
	Node               int          `json:"node"`
	Before             nodeResponse `json:"before"`
	After              nodeResponse `json:"after"`
	DisplacementChange [6]float64   `json:"displacement_change"`
	ReactionChange     [6]float64   `json:"reaction_change"`
}

type periodComparison struct {
	Rank   int     `json:"rank"`
	Before float64 `json:"before_seconds"`
	After  float64 `json:"after_seconds"`
	Change float64 `json:"change_seconds"`
}

type analysisResult struct {
	Static      *staticAnalysisResult `json:"static,omitempty"`
	ProposalID  uint64                `json:"proposal_id"`
	Period      string                `json:"period"`
	Nodes       []nodeComparison      `json:"nodes,omitempty"`
	Periods     []periodComparison    `json:"periods,omitempty"`
	Buckling    []bucklingComparison  `json:"buckling,omitempty"`
	TimeHistory *timeHistoryResult    `json:"time_history,omitempty"`
	Curves      *curveComparison      `json:"load_displacement,omitempty"`
}

type analysisStatus struct {
	ResultPath   string                `json:"result_path,omitempty"`
	JobID        uint64                `json:"job_id"`
	State        string                `json:"state"`
	Side         string                `json:"side,omitempty"`
	Phase        string                `json:"phase,omitempty"`
	Detail       string                `json:"detail,omitempty"`
	Elapsed      float64               `json:"elapsed_seconds"`
	Error        string                `json:"error,omitempty"`
	Step         *analysisStepProgress `json:"step_progress,omitempty"`
	OutputDir    string                `json:"output_dir,omitempty"`
	PreviewDir   string                `json:"preview_dir,omitempty"`
	PreviewError string                `json:"preview_error,omitempty"`
}

type analysisEvent struct {
	Side    string                `json:"side,omitempty"`
	Phase   string                `json:"phase,omitempty"`
	Detail  string                `json:"detail,omitempty"`
	Result  *analysisResult       `json:"result,omitempty"`
	Error   string                `json:"error,omitempty"`
	Step    *analysisStepProgress `json:"step_progress,omitempty"`
	Preview *st.LiveAnalysisStep  `json:"preview,omitempty"`
}

type analysisStepProgress struct {
	Time      *float64 `json:"time_seconds,omitempty"`
	Completed int      `json:"completed"`
	Total     int      `json:"total"`
	Factor    float64  `json:"load_factor"`
}

type analysisJob struct {
	staticOutput      *staticAnalysisOutput
	mu                sync.Mutex
	cancel            context.CancelFunc
	done              chan struct{}
	started, finished time.Time
	status            analysisStatus
	result            *analysisResult
	preview           *liveAnalysisOutput
}

func (j *analysisJob) snapshot() analysisStatus {
	status := j.status
	end := j.finished
	if end.IsZero() {
		end = time.Now()
	}
	status.Elapsed = end.Sub(j.started).Seconds()
	return status
}

func (s *modelSession) startAnalysis(o analysisOptions) (any, error) {
	if s.pending == nil || s.pending.ID != o.ProposalID {
		return nil, fmt.Errorf("provide the current proposal_id from preview_edits")
	}
	if o.Analysis == "" {
		o.Analysis = "both"
	}
	if !slices.Contains([]string{"static", "vibration", "both", "buckling", "arclm101"}, o.Analysis) {
		return nil, fmt.Errorf("analysis must be static, vibration, both, buckling, or arclm101")
	}
	o.Period = strings.ToUpper(o.Period)
	if o.Period == "" {
		o.Period = "L"
	}
	if !slices.Contains([]string{"L", "X", "Y"}, o.Period) {
		return nil, fmt.Errorf("period must be L, X, or Y")
	}
	if (o.Analysis == "static" || o.Analysis == "both" || o.Analysis == "arclm101") && (len(o.NodeIDs) == 0 || len(o.NodeIDs) > 200) {
		return nil, fmt.Errorf("specify 1 to 200 node_ids for static results or load-displacement curves")
	}
	if o.Analysis == "vibration" || o.Analysis == "buckling" {
		o.NodeIDs = nil
	}
	seen := make(map[int]bool)
	for _, id := range o.NodeIDs {
		if s.frame.Nodes[id] == nil || seen[id] {
			return nil, fmt.Errorf("unknown or duplicate node ID %d", id)
		}
		seen[id] = true
	}
	if o.Modes == 0 {
		o.Modes = 3
	}
	if o.Modes < 1 || o.Modes > 100 {
		return nil, fmt.Errorf("modes must be between 1 and 100")
	}
	if o.Analysis == "arclm101" {
		if o.Steps < 1 || o.Steps > 10000 || o.Increment == 0 || math.IsNaN(o.Increment) || math.IsInf(float64(o.Steps)*o.Increment, 0) {
			return nil, fmt.Errorf("arclm101 requires 1 to 10000 steps and a finite, nonzero increment")
		}
		if o.Steps*len(o.NodeIDs) > 10000 {
			return nil, fmt.Errorf("arclm101 supports at most 10000 step-node values per model")
		}
		if o.PreloadL && o.Period == "L" {
			return nil, fmt.Errorf("preload_l applies to X or Y loading")
		}
	} else if o.Steps != 0 || o.Increment != 0 || o.PreloadL {
		return nil, fmt.Errorf("steps, increment, and preload_l apply to arclm101")
	}
	if s.analysis != nil {
		select {
		case <-s.analysis.done:
		default:
			return nil, fmt.Errorf("job already running")
		}
	}
	if o.PreviewDir != "" && o.Analysis != "arclm101" || o.PreviewEvery != 0 && o.PreviewDir == "" {
		return nil, fmt.Errorf("preview_dir and preview_every apply to arclm101 live display")
	}
	if o.PreviewEvery < 0 || o.PreviewEvery > 10000 {
		return nil, fmt.Errorf("preview_every must be between 1 and 10000")
	}
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	var preview *liveAnalysisOutput
	var previewPath string
	if o.PreviewDir != "" {
		previewPath, err = filepath.Abs(filepath.Join(s.root.Name(), o.PreviewDir))
		if err != nil {
			return nil, err
		}
		preview, err = newLiveAnalysisOutput(s.root, o.PreviewDir, s.pending.data)
		if err != nil {
			return nil, err
		}
		if o.PreviewEvery == 0 {
			o.PreviewEvery = 10
		}
	}
	request := analysisRequest{Options: o, Before: s.raw, After: s.pending.data}
	ctx, cancel := context.WithCancel(context.Background())
	s.nextJob++
	job := &analysisJob{preview: preview, cancel: cancel, done: make(chan struct{}), started: time.Now(), status: analysisStatus{JobID: s.nextJob, State: "running", Phase: "starting"}}
	if preview != nil {
		job.status.PreviewDir = previewPath
	}
	s.analysis = job
	status := job.snapshot()
	go job.run(ctx, exe, filepath.Dir(s.parsePath), request)
	return status, nil
}

func (j *analysisJob) run(ctx context.Context, exe, scratch string, request analysisRequest) {
	defer j.cancel()
	result, err := j.execute(ctx, exe, scratch, request)
	j.mu.Lock()
	defer j.mu.Unlock()
	defer close(j.done)
	j.finished = time.Now()
	switch {
	case err == nil && result != nil:
		j.status.State, j.status.Phase, j.status.Detail = "completed", "completed", ""
		j.result = result
		if result.Static != nil {
			j.status.ResultPath = result.Static.Path
		}
	case ctx.Err() != nil:
		j.status.State = "cancelled"
	default:
		j.status.State = "failed"
		if err != nil {
			j.status.Error = err.Error()
		} else {
			j.status.Error = "analysis returned no results"
		}
	}
	if j.preview != nil {
		state, message := j.status.State, j.status.Error
		if result != nil && result.Curves != nil && result.Curves.After.Nonconvergence != nil {
			state, message = "nonconvergence", result.Curves.After.Nonconvergence.Message
		}
		if err := j.preview.finish(state, message); err != nil && j.status.PreviewError == "" {
			j.status.PreviewError = err.Error()
		}
	}
}

func (j *analysisJob) execute(ctx context.Context, exe, scratch string, request analysisRequest) (*analysisResult, error) {
	dir, err := os.MkdirTemp(scratch, "analysis-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	// Send the snapshots through stdin. CommandContext terminates the
	// worker even if a solver is inside a non-interruptible factorization.
	cmd := exec.CommandContext(ctx, exe, "-analysis-worker", dir)
	input, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	cmd.Stdin = bytes.NewReader(input)
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err = cmd.Start(); err != nil {
		return nil, err
	}
	var result *analysisResult
	var workerError error
	scanner := bufio.NewScanner(pipe)
	scanner.Buffer(make([]byte, 4096), 16<<20)
	for scanner.Scan() {
		var event analysisEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			workerError = fmt.Errorf("invalid response from analysis worker: %w", err)
			_ = cmd.Process.Kill()
			break
		}
		if event.Preview != nil && j.preview != nil && ctx.Err() == nil {
			if err := j.preview.publish(event.Preview); err != nil {
				j.mu.Lock()
				j.status.PreviewError = err.Error()
				j.mu.Unlock()
			}
		}
		if event.Error != "" {
			workerError = fmt.Errorf("%s", event.Error)
		}
		if event.Result != nil {
			result = event.Result
		}
		if event.Phase != "" {
			j.mu.Lock()
			if j.status.Side != event.Side || j.status.Phase != event.Phase {
				j.status.Step = nil
			}
			j.status.Side, j.status.Phase, j.status.Detail = event.Side, event.Phase, event.Detail
			if event.Step != nil {
				j.status.Step = event.Step
			}
			j.mu.Unlock()
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		_ = cmd.Process.Kill()
		workerError = scanErr
	}
	waitErr := cmd.Wait()
	if workerError != nil {
		return nil, workerError
	}
	if waitErr != nil {
		return nil, fmt.Errorf("analysis worker exited: %w", waitErr)
	}
	if result == nil {
		return nil, fmt.Errorf("analysis worker returned no results")
	}
	if j.staticOutput != nil {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if result.Static == nil {
			return nil, fmt.Errorf("analysis worker returned no static result")
		}
		if err := j.staticOutput.save(dir, result.Static); err != nil {
			return nil, err
		}
	}
	if result.TimeHistory != nil {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := j.preview.copyResponses(dir, result.TimeHistory.Files); err != nil {
			return nil, err
		}
		for i, name := range result.TimeHistory.Files {
			result.TimeHistory.Files[i] = filepath.Join(j.status.OutputDir, name)
		}
	}
	return result, nil
}

func (s *modelSession) analysisTool(name string, raw json.RawMessage) (any, error) {
	if name == "start_static_analysis" {
		var args staticAnalysisOptions
		if err := decode(raw, &args); err != nil {
			return nil, err
		}
		return s.startStaticAnalysis(args)
	}
	if name == "start_analysis_comparison" {
		var args analysisOptions
		if err := decode(raw, &args); err != nil {
			return nil, err
		}
		return s.startAnalysis(args)
	}
	var args struct {
		JobID uint64 `json:"job_id"`
	}
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	job := s.analysis
	if job == nil || job.status.JobID != args.JobID {
		return nil, fmt.Errorf("unknown analysis job %d", args.JobID)
	}
	job.mu.Lock()
	defer job.mu.Unlock()
	switch name {
	case "analysis_status":
		return job.snapshot(), nil
	case "cancel_analysis":
		if job.status.State == "running" {
			job.cancel()
			job.status.State = "cancelling"
		}
		return job.snapshot(), nil
	case "analysis_results":
		if job.result == nil {
			return nil, fmt.Errorf("job %d is %s", args.JobID, job.status.State)
		}
		return job.result, nil
	}
	return nil, fmt.Errorf("unknown analysis tool %s", name)
}

func (s *modelSession) closeAnalysis() {
	if s.analysis != nil {
		s.analysis.cancel()
		<-s.analysis.done
	}
}

func reportWorker(out io.Writer, event analysisEvent) error {
	return json.NewEncoder(out).Encode(event)
}
