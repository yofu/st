package st

import (
	"fmt"
	"math"
	"slices"

	"github.com/yofu/st/arclm"
)

const LiveAnalysisPeriod = "MCP"
const LiveAnalysisModel = "model.inp"
const LiveAnalysisStatusFile = "status.json"

func LiveAnalysisStepFile(step int) string {
	return fmt.Sprintf("step-%05d.json", step)
}

type LiveAnalysisStatus struct {
	State string
	Error string
}

type LiveAnalysisStep struct {
	Step, Total int
	Period      string
	PreloadL    bool
	Time        *float64 `json:",omitempty"`
	Factor      float64
	Nodes       []LiveAnalysisNode
	Elements    []LiveAnalysisElement
}

type LiveAnalysisNode struct {
	ID             int
	Disp, Reaction [6]float64
}

type LiveAnalysisElement struct {
	ID       int
	EndNodes [2]int
	Stress   [12]float64
	Hinge    [2]bool
}

func CaptureLiveAnalysisStep(frame *arclm.Frame, step, total int, period string, preloadL bool, factor float64) *LiveAnalysisStep {
	result := &LiveAnalysisStep{Step: step, Total: total, Period: period, PreloadL: preloadL, Factor: factor,
		Nodes: make([]LiveAnalysisNode, 0, len(frame.Nodes)), Elements: make([]LiveAnalysisElement, 0, len(frame.Elems))}
	for _, n := range frame.Nodes {
		result.Nodes = append(result.Nodes, LiveAnalysisNode{ID: n.Num, Disp: [6]float64(n.Disp), Reaction: [6]float64(n.Reaction)})
	}
	for _, e := range frame.Elems {
		result.Elements = append(result.Elements, LiveAnalysisElement{ID: e.Num, EndNodes: [2]int{e.Enod[0].Num, e.Enod[1].Num}, Stress: [12]float64(e.Stress), Hinge: [2]bool(e.Phinge)})
	}
	return result
}

// ApplyLiveAnalysisStep updates only the dedicated display result, so the
// frame keeps a bounded amount of data and no analysis continuation state.
func (frame *Frame) ApplyLiveAnalysisStep(step *LiveAnalysisStep) error {
	if step.Step < 1 || step.Step > step.Total || math.IsNaN(step.Factor) || math.IsInf(step.Factor, 0) {
		return fmt.Errorf("invalid live analysis step")
	}
	for _, r := range step.Nodes {
		if frame.Nodes[r.ID] == nil {
			return fmt.Errorf("live analysis node %d is missing", r.ID)
		}
	}
	for _, r := range step.Elements {
		el := frame.Elems[r.ID]
		if el == nil || !el.IsLineElem() || el.Enod[0].Num != r.EndNodes[0] || el.Enod[1].Num != r.EndNodes[1] {
			return fmt.Errorf("live analysis element %d does not match the model", r.ID)
		}
	}
	for _, r := range step.Nodes {
		n := frame.Nodes[r.ID]
		n.Disp[LiveAnalysisPeriod] = slices.Clone(r.Disp[:])
		n.Reaction[LiveAnalysisPeriod] = slices.Clone(r.Reaction[:])
	}
	for _, r := range step.Elements {
		el := frame.Elems[r.ID]
		el.Stress[LiveAnalysisPeriod] = map[int][]float64{r.EndNodes[0]: slices.Clone(r.Stress[:6]), r.EndNodes[1]: slices.Clone(r.Stress[6:])}
		el.Phinge[LiveAnalysisPeriod] = map[int]bool{r.EndNodes[0]: r.Hinge[0], r.EndNodes[1]: r.Hinge[1]}
	}
	return nil
}
