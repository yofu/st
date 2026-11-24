package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	st "github.com/yofu/st/stlib"
)

type staticAnalysisOptions struct {
	Period     string `json:"period"`
	ResultPath string `json:"result_path"`
	NodeIDs    []int  `json:"node_ids"`
	Overwrite  bool   `json:"overwrite,omitempty"`
}

type staticNodeResult struct {
	Node int `json:"node"`
	nodeResponse
}

type staticAnalysisResult struct {
	Path  string             `json:"result_path"`
	Nodes []staticNodeResult `json:"nodes,omitempty"`
}

type staticAnalysisOutput struct {
	root         *os.Root
	path         string
	overwrite    bool
	modelPath    string
	modelDigest  [32]byte
	resultDigest [32]byte
}

func (s *modelSession) startStaticAnalysis(o staticAnalysisOptions) (any, error) {
	if s.frame == nil {
		return nil, fmt.Errorf("open an INP model first")
	}
	o.Period = strings.ToUpper(o.Period)
	if o.Period == "" {
		o.Period = "L"
	}
	ext := staticResultExtension(o.Period)
	if ext == "" {
		return nil, fmt.Errorf("period must be L, X, or Y")
	}
	if !filepath.IsLocal(o.ResultPath) || !strings.EqualFold(filepath.Ext(o.ResultPath), ext) {
		return nil, fmt.Errorf("result_path must be a relative %s path within the server root", ext)
	}
	o.ResultPath = filepath.Clean(o.ResultPath)
	if info, err := s.root.Lstat(o.ResultPath); err == nil {
		if !o.Overwrite {
			return nil, fmt.Errorf("result_path already exists: %s", o.ResultPath)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("cannot overwrite non-regular file: %s", o.ResultPath)
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	parent, err := s.root.Stat(filepath.Dir(o.ResultPath))
	if err != nil {
		return nil, fmt.Errorf("result directory: %w", err)
	}
	if !parent.IsDir() {
		return nil, fmt.Errorf("result directory is not a directory")
	}
	if len(o.NodeIDs) > 200 {
		return nil, fmt.Errorf("specify at most 200 node_ids")
	}
	seen := make(map[int]bool, len(o.NodeIDs))
	for _, id := range o.NodeIDs {
		if s.frame.Nodes[id] == nil || seen[id] {
			return nil, fmt.Errorf("unknown or duplicate node ID %d", id)
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
	if s.guiID == "" {
		if err := s.verifyOriginal(saveFile{path: s.source, original: s.raw}); err != nil {
			return nil, err
		}
	}
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	path := s.guiModelPath()
	output := &staticAnalysisOutput{root: s.root, path: o.ResultPath, overwrite: o.Overwrite, modelPath: path, modelDigest: s.frame.GUIModelDigest()}
	request := analysisRequest{
		Options:      analysisOptions{Analysis: "static", Period: o.Period, NodeIDs: o.NodeIDs},
		After:        s.raw,
		SingleStatic: true,
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.nextJob++
	job := &analysisJob{
		staticOutput: output, cancel: cancel, done: make(chan struct{}), started: time.Now(),
		status: analysisStatus{JobID: s.nextJob, State: "running", Phase: "starting"},
	}
	s.analysis = job
	status := job.snapshot()
	go job.run(ctx, exe, filepath.Dir(s.parsePath), request)
	return status, nil
}

func staticResultExtension(period string) string {
	switch period {
	case "L":
		return ".otl"
	case "X":
		return ".ohx"
	case "Y":
		return ".ohy"
	}
	return ""
}

func (s *staticAnalysisOutput) save(dir string, result *staticAnalysisResult) error {
	data, err := os.ReadFile(filepath.Join(dir, "static.otl"))
	if err != nil {
		return fmt.Errorf("read static result: %w", err)
	}
	saveFn := writeNew
	if s.overwrite {
		saveFn = writeOverwrite
	}
	if err := saveFn(s.root, s.path, data); err != nil {
		return fmt.Errorf("save static result: %w", err)
	}
	s.resultDigest = sha256.Sum256(data)
	result.Path = s.path
	return nil
}

func (s *modelSession) staticGUIRequest(path, modelPath string, digest [32]byte, axes []string) (st.GUIRequest, error) {
	if err := s.requireRoot(); err != nil {
		return st.GUIRequest{}, err
	}
	if !filepath.IsLocal(path) {
		return st.GUIRequest{}, fmt.Errorf("result path must be within the server root")
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".otl" && ext != ".ohx" && ext != ".ohy" {
		return st.GUIRequest{}, fmt.Errorf("expected a static result file (.otl, .ohx or .ohy)")
	}
	path = filepath.Clean(path)
	f, err := s.root.Open(path)
	if err != nil {
		return st.GUIRequest{}, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return st.GUIRequest{}, err
	}
	if !info.Mode().IsRegular() {
		return st.GUIRequest{}, fmt.Errorf("expected a regular result file")
	}
	const maxBytes = 64 << 20
	data, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil {
		return st.GUIRequest{}, err
	}
	if len(data) > maxBytes {
		return st.GUIRequest{}, fmt.Errorf("result file exceeds %d bytes", maxBytes)
	}
	abs, err := filepath.Abs(filepath.Join(s.modelRootPath(), path))
	if err != nil {
		return st.GUIRequest{}, err
	}
	if axes == nil {
		axes = []string{"Z"}
	}
	return st.GUIRequest{Action: "show_result", BasePath: modelPath, BaseDigest: digest, ResultPath: abs, ResultData: data, DisplacementAxes: axes}, nil
}
