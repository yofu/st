package arclm

import (
	"encoding/gob"
	"fmt"
	"io"
)

// The worker wire keeps mutable solver state independent from the UI frame.
// Nodes and sections are indexed so a member never carries duplicate geometry.
type frameWire struct {
	Nodes            []*Node
	Sections         []*Sect
	SectionCount     int
	Elems            []elemWire
	EigenValue       []float64
	EigenVector      [][]float64
	PlasticLoad      float64
	PlasticLoadValid bool
	Loading          *loadingWire
	Geometry         *geometryWire
	Transient        *TimeHistoryState
}

type elemWire struct {
	Num, Section              int
	Ends                      [2]int
	Bonds, OriginalBonds      []int
	Cang, Energy, EnergyB     float64
	Strong, Weak, Cmq, Stress []float64
	Phinge                    []bool
	Valid                     bool
	Plastic                   *materialWire
}

type materialWire struct {
	Plastic [12]float64
	Tangent [][]float64
}

type loadingWire struct {
	Steps       int
	Factor      float64
	Nodal, Base []float64
	Fixed       [][]float64
}

type geometryWire struct {
	Rotations, Axes                   []geometryMatrix
	Factor                            float64
	Material                          bool
	Steps                             int
	BaseNodal, PatternNodal           []float64
	BaseFixed, PatternFixed, RawFixed [][]float64
	Definition                        []float64
}

// EncodeWorkerFrame records a completed solver state without channels or callbacks.
func EncodeWorkerFrame(out io.Writer, frame *Frame) error {
	data := frameWire{
		Nodes: frame.Nodes, SectionCount: len(frame.Sects), EigenValue: frame.EigenValue, EigenVector: frame.EigenVector,
		PlasticLoad: frame.plasticLoad, PlasticLoadValid: frame.plasticLoadValid, Transient: frame.transient,
	}
	sections := make(map[*Sect]int)
	addSection := func(s *Sect) int {
		if s == nil {
			return -1
		}
		if index, ok := sections[s]; ok {
			return index
		}
		index := len(data.Sections)
		data.Sections = append(data.Sections, s)
		sections[s] = index
		return index
	}
	// Preserve the frame section table's order, including repeated references.
	for _, section := range frame.Sects {
		if section == nil {
			return fmt.Errorf("analysis section is nil")
		}
		sections[section] = len(data.Sections)
		data.Sections = append(data.Sections, section)
	}
	nodeIndex := make(map[*Node]int, len(frame.Nodes))
	for i, node := range frame.Nodes {
		nodeIndex[node] = i
	}
	encodeBonds := func(bonds []*Sect) []int {
		result := make([]int, len(bonds))
		for i, bond := range bonds {
			result[i] = addSection(bond)
		}
		return result
	}
	data.Elems = make([]elemWire, len(frame.Elems))
	for i, elem := range frame.Elems {
		if elem == nil || len(elem.Enod) != 2 {
			return fmt.Errorf("invalid analysis element at index %d", i)
		}
		a, aOK := nodeIndex[elem.Enod[0]]
		b, bOK := nodeIndex[elem.Enod[1]]
		if !aOK || !bOK {
			return fmt.Errorf("analysis element %d references an unknown node", elem.Num)
		}
		entry := elemWire{
			Num: elem.Num, Section: addSection(elem.Sect), Ends: [2]int{a, b},
			Bonds: encodeBonds(elem.Bonds), OriginalBonds: encodeBonds(elem.plasticBonds),
			Cang: elem.Cang, Energy: elem.Energy, EnergyB: elem.Energyb,
			Strong: elem.Strong, Weak: elem.Weak, Cmq: elem.Cmq, Stress: elem.Stress,
			Phinge: elem.Phinge, Valid: elem.IsValid,
		}
		if s := elem.material; s != nil {
			entry.Plastic = &materialWire{s.plastic, s.tangent}
		}
		data.Elems[i] = entry
	}
	if s := frame.loading; s != nil {
		data.Loading = &loadingWire{s.steps, s.factor, s.nodalPattern, s.baseNodal, s.fixedPattern}
	}
	if s := frame.geometry; s != nil {
		data.Geometry = &geometryWire{s.rotations, s.axes, s.factor, s.material, s.steps, s.baseNodal, s.patternNodal, s.baseFixed, s.patternFixed, s.rawFixed, s.definition}
	}
	return gob.NewEncoder(out).Encode(data)
}

// DecodeWorkerFrame restores the state and creates fresh solver channels.
func DecodeWorkerFrame(in io.Reader) (*Frame, error) {
	var data frameWire
	if err := gob.NewDecoder(in).Decode(&data); err != nil {
		return nil, err
	}
	if data.SectionCount < 0 || data.SectionCount > len(data.Sections) {
		return nil, fmt.Errorf("invalid worker section table")
	}
	f := NewFrame()
	f.Nodes = data.Nodes
	f.Sects = data.Sections[:data.SectionCount]
	f.EigenValue, f.EigenVector = data.EigenValue, data.EigenVector
	f.plasticLoad, f.plasticLoadValid, f.transient = data.PlasticLoad, data.PlasticLoadValid, data.Transient
	if s := data.Loading; s != nil {
		f.loading = &plasticLoadState{steps: s.Steps, factor: s.Factor, nodalPattern: s.Nodal, baseNodal: s.Base, fixedPattern: s.Fixed}
	}
	if s := data.Geometry; s != nil {
		f.geometry = &geometryState{s.Rotations, s.Axes, s.Factor, s.Material, s.Steps, s.BaseNodal, s.PatternNodal, s.BaseFixed, s.PatternFixed, s.RawFixed, s.Definition}
	}
	section := func(index int) *Sect {
		if index < 0 {
			return nil
		}
		return data.Sections[index]
	}
	decodeBonds := func(indices []int) []*Sect {
		if len(indices) == 0 {
			return nil
		}
		result := make([]*Sect, len(indices))
		for i, index := range indices {
			result[i] = section(index)
		}
		return result
	}
	f.Elems = make([]*Elem, len(data.Elems))
	for i, e := range data.Elems {
		elem := &Elem{
			Num: e.Num, Sect: section(e.Section), Enod: []*Node{f.Nodes[e.Ends[0]], f.Nodes[e.Ends[1]]},
			Bonds: decodeBonds(e.Bonds), plasticBonds: decodeBonds(e.OriginalBonds),
			Cang: e.Cang, Energy: e.Energy, Energyb: e.EnergyB,
			Strong: e.Strong, Weak: e.Weak, Cmq: e.Cmq, Stress: e.Stress,
			Phinge: e.Phinge, IsValid: e.Valid,
		}
		if s := e.Plastic; s != nil {
			elem.material = &plasticMaterialState{s.Plastic, s.Tangent}
		}
		f.Elems[i] = elem
	}
	f.restoreGeometryAxes()
	return f, nil
}
