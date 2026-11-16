package st

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// Model annotations are stored in INP comments so older readers can still
// open the structural model. One record per object keeps lines short.
const modelRecordPrefix = "#STMODEL"

type modelLock struct {
	ID     int
	Weight []float64 `json:",omitempty"` // Legacy cached weight, read for display only.
	Locked bool      `json:",omitempty"`
}

type modelArc struct {
	Nodes []int
}

func (frame *Frame) modelRecords() string {
	var out strings.Builder
	write := func(kind string, value any) {
		data, err := json.Marshal(value)
		if err != nil {
			panic(fmt.Sprintf("INP %s: %v", kind, err))
		}
		fmt.Fprintf(&out, "%s %s %s\r\n", modelRecordPrefix, kind, data)
	}
	for _, id := range slices.Sorted(maps.Keys(frame.Nodes)) {
		n := frame.Nodes[id]
		if n.Lock {
			write("NODE", modelLock{ID: id, Locked: true})
		}
	}
	for _, id := range slices.Sorted(maps.Keys(frame.Elems)) {
		if frame.Elems[id].Lock {
			write("ELEM", modelLock{ID: id, Locked: true})
		}
	}
	for _, key := range slices.Sorted(maps.Keys(frame.Kijuns)) {
		k := frame.Kijuns[key]
		write("KIJUN", [3]any{k.Name, k.Start, k.End})
	}
	for _, m := range frame.Measures {
		copy := *m
		copy.Frame = nil
		write("MEASURE", copy)
	}
	for _, arc := range frame.Arcs {
		record := modelArc{}
		for _, n := range arc.Enod {
			if frame.Nodes[n.Num] != n {
				record.Nodes = nil
				break
			}
			record.Nodes = append(record.Nodes, n.Num)
		}
		if len(record.Nodes) == 3 {
			write("ARC", record)
		}
	}
	return out.String()
}

func (frame *Frame) readModelRecord(text string, nodemap map[int]int, coord []float64, angle float64) error {
	kind, data, _ := strings.Cut(text, " ")
	unmarshal := func(out any) error {
		if err := json.Unmarshal([]byte(data), out); err != nil {
			return fmt.Errorf("invalid INP %s record: %w", kind, err)
		}
		return nil
	}
	point := func(p []float64) []float64 {
		p = slices.Clone(p)
		for i := range p {
			p[i] += coord[i]
		}
		return RotateVector(p, coord, ZAXIS, angle)
	}
	switch kind {
	case "NODE", "ELEM":
		var r modelLock
		if err := unmarshal(&r); err != nil {
			return err
		}
		if kind == "NODE" {
			if n := frame.Nodes[nodemap[r.ID]]; n != nil {
				n.Lock = r.Locked
				if len(r.Weight) == 3 {
					copy(n.Weight, r.Weight)
				}
			}
		} else if el := frame.Elems[r.ID]; el != nil && angle == 0 && slices.Equal(coord, []float64{0, 0, 0}) {
			el.Lock = r.Locked
		}
	case "KIJUN":
		var r [3]json.RawMessage
		if err := unmarshal(&r); err != nil {
			return err
		}
		k := NewKijun()
		for i, dst := range []any{&k.Name, &k.Start, &k.End} {
			if err := json.Unmarshal(r[i], dst); err != nil {
				return err
			}
		}
		if k.Name == "" || len(k.Start) != 3 || len(k.End) != 3 {
			return fmt.Errorf("invalid INP KIJUN geometry")
		}
		k.Start, k.End = point(k.Start), point(k.End)
		frame.Kijuns[strings.ToLower(k.Name)] = k
		frame.Show.Kijun = true
	case "MEASURE":
		m := &Measure{}
		if err := unmarshal(m); err != nil {
			return err
		}
		if len(m.Start) != 3 || len(m.End) != 3 || len(m.Direction) != 3 {
			return fmt.Errorf("invalid INP MEASURE geometry")
		}
		m.Frame = frame
		m.Start, m.End = point(m.Start), point(m.End)
		m.Direction = RotateVector(m.Direction, []float64{0, 0, 0}, ZAXIS, angle)
		frame.Measures = append(frame.Measures, m)
		frame.Show.Measure = true
	case "ARC":
		var r modelArc
		if err := unmarshal(&r); err != nil {
			return err
		}
		if len(r.Nodes) != 3 {
			return fmt.Errorf("invalid INP ARC nodes")
		}
		nodes := make([]*Node, 3)
		for i, id := range r.Nodes {
			nodes[i] = frame.Nodes[nodemap[id]]
			if nodes[i] == nil {
				return fmt.Errorf("INP ARC references missing node %d", id)
			}
		}
		a := NewArc(nodes)
		if a == nil {
			return fmt.Errorf("invalid INP ARC geometry")
		}
		a.Frame = frame
		frame.Arcs = append(frame.Arcs, a)
		frame.Show.Arc = true
	}
	return nil
}
