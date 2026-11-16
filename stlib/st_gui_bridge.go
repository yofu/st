package st

import (
	"crypto/sha256"
	"fmt"
	"math"
	"path/filepath"
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
}

type GUIResponse struct {
	Period           string   `json:"period,omitempty"`
	ResultPath       string   `json:"result_path,omitempty"`
	Deformation      bool     `json:"deformation"`
	DisplacementAxes []string `json:"displacement_axes"`
	Path             string   `json:"path"`
	Changed          bool     `json:"changed"`
	Nodes            []int    `json:"node_ids"`
	Elements         []int    `json:"element_ids"`
	Error            string   `json:"error,omitempty"`
}

// GUIModelDigest excludes camera and screenshot state.
func (f *Frame) GUIModelDigest() [32]byte {
	copy := *f
	copy.View = NewView()
	return sha256.Sum256(copy.EditableInpBytes())
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
