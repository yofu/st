package st

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"math"
	"path/filepath"
	"slices"
	"strings"
)

type GUIRequest struct {
	Action              string
	BasePath            string
	BaseDigest          [32]byte
	INP, LST            []byte
	NodeIDs, ElementIDs *[]int
	PreviewDir          string
	LSTPath             string
	ResultPath          string
	ResultData          []byte
	DisplacementAxes    []string
	Display             *GUIDisplay
	Revision            string
	Commands            []string
	CommandJobID        uint64
}

type GUICommandStatus struct {
	Completion uint64 `json:"completion_version,omitempty"`
	ID         uint64 `json:"job_id"`
	State      string `json:"state"`
	Completed  int    `json:"completed_commands"`
	Total      int    `json:"total_commands"`
	Command    string `json:"command,omitempty"`
	History    string `json:"history,omitempty"`
	Progress   string `json:"progress,omitempty"`
	Step       int    `json:"step,omitempty"`
	Steps      int    `json:"steps,omitempty"`
	Error      string `json:"error,omitempty"`
}

// GUIDisplay contains only display settings, never model editing commands.
type GUIDisplay struct {
	Period           string       `json:"period,omitempty"`
	Deformation      *bool        `json:"deformation,omitempty"`
	DeformationScale *float64     `json:"deformation_scale,omitempty"`
	DisplacementAxes *[]string    `json:"displacement_axes,omitempty"`
	Stress           *[]GUIStress `json:"stress,omitempty"`
	Angles           []float64    `json:"angles,omitempty"`
	Focus            []float64    `json:"focus,omitempty"`
}

type GUIStress struct {
	Type      string `json:"type"`
	Component string `json:"component"`
}

type GUIModelSnapshot struct {
	INP     []byte `json:"inp"`
	LST     []byte `json:"lst,omitempty"`
	LSTPath string `json:"lst_path,omitempty"`
}

type GUIResponse struct {
	Period           string            `json:"period,omitempty"`
	ResultPath       string            `json:"result_path,omitempty"`
	Deformation      bool              `json:"deformation"`
	DisplacementAxes []string          `json:"displacement_axes"`
	Path             string            `json:"path"`
	Changed          bool              `json:"changed"`
	Nodes            []int             `json:"node_ids"`
	Elements         []int             `json:"element_ids"`
	Error            string            `json:"error,omitempty"`
	Revision         string            `json:"revision,omitempty"`
	Snapshot         *GUIModelSnapshot `json:"snapshot,omitempty"`
	Display          *GUIDisplay       `json:"display,omitempty"`
	CommandJob       *GUICommandStatus `json:"command_job,omitempty"`
	RatedElements    int               `json:"rated_elements"`
	SrcanRate        uint              `json:"srcan_rate"`
	SyncError        string            `json:"sync_error,omitempty"`
}

// GUIModelDigest excludes camera and screenshot state.
func (f *Frame) GUIModelDigest() [32]byte {
	copy := *f
	copy.View = NewView()
	return sha256.Sum256(append(copy.GUIInpBytes(), copy.GUILstBytes()...))
}

func (f *Frame) GUIRevision() string {
	digest := f.GUIModelDigest()
	return hex.EncodeToString(digest[:])
}

// GUILstBytes serializes the current definitions, including unsaved LST edits.
func (f *Frame) GUILstBytes() []byte {
	var out strings.Builder
	for _, id := range slices.Sorted(maps.Keys(f.Sects)) {
		if a := f.Sects[id].Allow; a != nil {
			// Legacy LST readers retain surrounding quotes in names. Normalize
			// the textual representation so repeated snapshots stay identical.
			for _, line := range strings.Split(a.String(), "\n") {
				if strings.TrimSpace(line) == "" {
					continue
				}
				head, tail, quoted := strings.Cut(line, "\"")
				out.WriteString(strings.Join(strings.Fields(head), " "))
				if quoted {
					fmt.Fprintf(&out, " \"%s\"", strings.Trim(strings.TrimSpace(tail), "\""))
				}
				out.WriteByte('\n')
			}
		}
	}
	return []byte(out.String())
}

func StaticResultPeriod(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".otl":
		return "L"
	case ".ohx":
		return "X"
	case ".ohy":
		return "Y"
	}
	return ""
}

// LoadStaticResult stages parsing and updates only the selected result period.
func (f *Frame) LoadStaticResult(path string, data []byte) (err error) {
	period := StaticResultPeriod(path)
	if period == "" {
		return fmt.Errorf("expected a static result file (.otl, .ohx or .ohy)")
	}
	defer func() {
		if problem := recover(); problem != nil {
			err = fmt.Errorf("invalid static result: %v", problem)
		}
	}()
	parsed := &Frame{Nodes: make(map[int]*Node, len(f.Nodes)), Elems: make(map[int]*Elem, len(f.Elems)), ResultFileName: make(map[string]string)}
	for id := range f.Nodes {
		parsed.Nodes[id] = &Node{Disp: make(map[string][]float64), Reaction: make(map[string][]float64)}
	}
	for id, elem := range f.Elems {
		parsed.Elems[id] = &Elem{Enods: elem.Enods, Stress: make(map[string]map[int][]float64), Phinge: make(map[string]map[int]bool)}
	}
	if err := parsed.readResultData(path, data, UpdateResult, true); err != nil {
		return err
	}
	for id, n := range parsed.Nodes {
		values := n.Disp[period]
		if len(values) != 6 {
			return fmt.Errorf("static result is missing displacements for node %d", id)
		}
		for _, v := range append(values[:len(values):len(values)], n.Reaction[period]...) {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return fmt.Errorf("static result contains nonfinite result at node %d", id)
			}
		}
	}
	// Validate available member end records against the active topology.
	for id, e := range parsed.Elems {
		stress := e.Stress[period]
		if len(stress) == 0 {
			continue
		}
		active := f.Elems[id]
		for _, n := range active.Enod {
			if len(stress[n.Num]) != 6 {
				return fmt.Errorf("static result has different end nodes for member %d", id)
			}
		}
	}
	for id, n := range parsed.Nodes {
		f.Nodes[id].Disp[period] = n.Disp[period]
		// Fixed components without a reaction record have zero reaction.
		f.Nodes[id].Reaction[period] = n.Reaction[period]
		if f.Nodes[id].Reaction[period] == nil {
			f.Nodes[id].Reaction[period] = make([]float64, 6)
		}
	}
	for id, e := range parsed.Elems {
		dest := f.Elems[id]
		if !dest.IsLineElem() {
			continue
		}
		if stress := e.Stress[period]; len(stress) > 0 {
			dest.Stress[period] = stress
			dest.Phinge[period] = e.Phinge[period]
		} else {
			delete(dest.Stress, period)
			delete(dest.Phinge, period)
		}
	}
	f.ResultFileName[period] = path
	return nil
}
