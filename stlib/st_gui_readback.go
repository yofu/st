package st

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

// GUIReadback selects result data from the final GUI frame.
type GUIReadback struct {
	Files              []string `json:"files,omitempty"`
	Period             string   `json:"period,omitempty"`
	Nodes              []int    `json:"node_ids,omitempty"`
	RequireDeformation bool     `json:"require_deformation,omitempty"`
}
type GUIFileInfo struct {
	Path     string    `json:"path"`
	Bytes    int64     `json:"bytes"`
	Modified time.Time `json:"modified_at"`
}
type GUIResultView struct {
	Files                 []GUIFileInfo     `json:"files,omitempty"`
	ModeVersion           uint64            `json:"mode_version,omitempty"`
	Period                string            `json:"period"`
	Path                  string            `json:"result_path,omitempty"`
	Eigenvalue            *float64          `json:"file_eigenvalue,omitempty"`
	NodesWithDisplacement int               `json:"nodes_with_displacement"`
	NonzeroNodes          int               `json:"nonzero_nodes"`
	Displacements         map[int][]float64 `json:"displacements,omitempty"`
}
type GUIAnalysisOutput struct {
	BeforeRead  uint64    `json:"prior_mode_version,omitempty"`
	Kind        string    `json:"kind"`
	Period      string    `json:"load_case"`
	Path        string    `json:"result_path"`
	Eigenvalues []float64 `json:"eigenvalues"`
}

func (f *Frame) ReadGUIResults(q GUIReadback) (*GUIResultView, error) {
	period := strings.ToUpper(strings.TrimSpace(q.Period))
	if period == "" {
		period = f.Show.Period
	}
	r := &GUIResultView{Period: period, Path: f.ResultFileName[period]}
	if rank, err := strconv.Atoi(strings.TrimPrefix(period, "B")); strings.HasPrefix(period, "B") && err == nil && rank > 0 {
		if value, ok := f.Eigenvalue[rank-1]; ok {
			r.Eigenvalue = &value
			r.ModeVersion = f.eigenReadVersion
		}
	}
	for _, n := range f.Nodes {
		d, ok := n.Disp[period]
		if !ok {
			continue
		}
		if len(d) != 6 {
			return nil, fmt.Errorf("incomplete displacement for %s node %d", period, n.Num)
		}
		nonzero := false
		for _, v := range d {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return nil, fmt.Errorf("nonfinite displacement for %s node %d", period, n.Num)
			}
			nonzero = nonzero || v != 0
		}
		r.NodesWithDisplacement++
		if nonzero {
			r.NonzeroNodes++
		}
	}
	if len(q.Nodes) > 200 {
		return nil, fmt.Errorf("request at most 200 nodes")
	}
	if len(q.Nodes) > 0 {
		r.Displacements = map[int][]float64{}
	}
	for _, id := range q.Nodes {
		n := f.Nodes[id]
		if n == nil || n.Disp[period] == nil {
			return nil, fmt.Errorf("no displacement for %s node %d", period, id)
		}
		r.Displacements[id] = slices.Clone(n.Disp[period])
	}
	if q.Period != "" && r.NodesWithDisplacement == 0 {
		return nil, fmt.Errorf("no displacements loaded for %s", period)
	}
	if q.RequireDeformation && (f.Show.Period != period || f.Show.PlotState&PLOT_DEFORMED == 0 || r.NodesWithDisplacement == 0) {
		return nil, fmt.Errorf("deformation for %s is not displayed", period)
	}
	if len(q.Files) > 100 {
		return nil, fmt.Errorf("request at most 100 output files")
	}
	for _, name := range q.Files {
		if name == "" {
			return nil, fmt.Errorf("empty output file name")
		}
		path := name
		if strings.HasPrefix(name, "%<") {
			if f.Path == "" {
				return nil, fmt.Errorf("no model path for %s", name)
			}
			path = PruneExt(f.Path) + strings.TrimPrefix(name, "%<")
		} else if !filepath.IsAbs(path) {
			path = filepath.Join(filepath.Dir(f.Path), path)
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("expected an output file: %s", path)
		}
		r.Files = append(r.Files, GUIFileInfo{Path: path, Bytes: info.Size(), Modified: info.ModTime().UTC()})
	}
	return r, nil
}

// reportCommandFile records a file after the native writer has succeeded.
func reportCommandFile(w any, path string) {
	if sink, ok := w.(interface{ FileWritten(string) }); ok && path != "" {
		sink.FileWritten(path)
	}
}
