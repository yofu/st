package st

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"
)

// NodeTransform describes one expression-based bulk coordinate edit applied
// simultaneously across matching nodes. Coordinates and bounds are in metres.
type NodeTransform struct {
	NodeIDs []int              `json:"node_ids,omitzero"`
	Where   string             `json:"where,omitempty"`
	Params  map[string]float64 `json:"params,omitempty"`
	X       string             `json:"x,omitempty"`
	Y       string             `json:"y,omitempty"`
	Z       string             `json:"z,omitempty"`
	DX      string             `json:"dx,omitempty"`
	DY      string             `json:"dy,omitempty"`
	DZ      string             `json:"dz,omitempty"`
}

const maxNodeTransformStages = 100

var reservedTransformVars = map[string]bool{
	"id": true, "x": true, "y": true, "z": true,
	"xmin": true, "xmax": true, "ymin": true, "ymax": true, "zmin": true, "zmax": true,
	"cx": true, "cy": true, "cz": true,
	"model_xmin": true, "model_xmax": true, "model_ymin": true, "model_ymax": true, "model_zmin": true, "model_zmax": true,
	"model_cx": true, "model_cy": true, "model_cz": true,
	"pi": true, "e": true, "math": true, "np": true, "numpy": true,
}

type nodeBounds struct {
	min [3]float64
	max [3]float64
}

func computeNodeBounds(nodes []*Node) (nodeBounds, bool) {
	if len(nodes) == 0 {
		return nodeBounds{}, false
	}
	b := nodeBounds{
		min: [3]float64{nodes[0].Coord[0], nodes[0].Coord[1], nodes[0].Coord[2]},
		max: [3]float64{nodes[0].Coord[0], nodes[0].Coord[1], nodes[0].Coord[2]},
	}
	for _, n := range nodes[1:] {
		for axis := 0; axis < 3; axis++ {
			v := n.Coord[axis]
			if v < b.min[axis] {
				b.min[axis] = v
			}
			if v > b.max[axis] {
				b.max[axis] = v
			}
		}
	}
	return b, true
}

func bindBounds(env map[string]any, prefix string, b nodeBounds) {
	env[prefix+"xmin"] = b.min[0]
	env[prefix+"xmax"] = b.max[0]
	env[prefix+"ymin"] = b.min[1]
	env[prefix+"ymax"] = b.max[1]
	env[prefix+"zmin"] = b.min[2]
	env[prefix+"zmax"] = b.max[2]
	env[prefix+"cx"] = 0.5 * (b.min[0] + b.max[0])
	env[prefix+"cy"] = 0.5 * (b.min[1] + b.max[1])
	env[prefix+"cz"] = 0.5 * (b.min[2] + b.max[2])
}

// NodeBounds returns the overall XYZ bounding box of all nodes in the frame.
func (f *Frame) NodeBounds() ([]float64, []float64) {
	if len(f.Nodes) == 0 {
		return nil, nil
	}
	nodes := make([]*Node, 0, len(f.Nodes))
	for _, n := range f.Nodes {
		nodes = append(nodes, n)
	}
	b, ok := computeNodeBounds(nodes)
	if !ok {
		return nil, nil
	}
	return []float64{b.min[0], b.min[1], b.min[2]}, []float64{b.max[0], b.max[1], b.max[2]}
}

// ApplyNodeTransforms evaluates one or more expression-based coordinate
// transformations in sequence on a private frame. Each stage evaluates all
// requested axes simultaneously against pre-stage coordinates.
func (f *Frame) ApplyNodeTransforms(specs []NodeTransform) (map[int]bool, error) {
	if len(specs) == 0 {
		return nil, nil
	}
	if len(specs) > maxNodeTransformStages {
		return nil, fmt.Errorf("too many transform_nodes stages (max %d)", maxNodeTransformStages)
	}
	if len(f.Nodes) == 0 {
		return nil, fmt.Errorf("transform_nodes requires at least one node in the model")
	}

	allIDs := slices.Sorted(maps.Keys(f.Nodes))
	allNodes := make([]*Node, len(allIDs))
	for i, id := range allIDs {
		allNodes[i] = f.Nodes[id]
	}

	moved := make(map[int]bool)
	for stageIdx, spec := range specs {
		absSrc := [3]string{strings.TrimSpace(spec.X), strings.TrimSpace(spec.Y), strings.TrimSpace(spec.Z)}
		relSrc := [3]string{strings.TrimSpace(spec.DX), strings.TrimSpace(spec.DY), strings.TrimSpace(spec.DZ)}
		axisNames := [3]string{"x", "y", "z"}

		var absExpr [3]*FloatExpr
		var relExpr [3]*FloatExpr
		hasAny := false
		for axis := 0; axis < 3; axis++ {
			if absSrc[axis] != "" && relSrc[axis] != "" {
				return nil, fmt.Errorf("transform_nodes[%d]: cannot specify both %s and d%s", stageIdx, axisNames[axis], axisNames[axis])
			}
			if absSrc[axis] != "" {
				ex, err := CompileFloatExpr(absSrc[axis])
				if err != nil {
					return nil, fmt.Errorf("transform_nodes[%d].%s: %w", stageIdx, axisNames[axis], err)
				}
				absExpr[axis] = ex
				hasAny = true
			}
			if relSrc[axis] != "" {
				ex, err := CompileFloatExpr(relSrc[axis])
				if err != nil {
					return nil, fmt.Errorf("transform_nodes[%d].d%s: %w", stageIdx, axisNames[axis], err)
				}
				relExpr[axis] = ex
				hasAny = true
			}
		}
		if !hasAny {
			return nil, fmt.Errorf("transform_nodes[%d]: specify at least one of x, y, z, dx, dy, dz", stageIdx)
		}

		var whereExpr *FloatExpr
		if src := strings.TrimSpace(spec.Where); src != "" {
			ex, err := CompileFloatExpr(src)
			if err != nil {
				return nil, fmt.Errorf("transform_nodes[%d].where: %w", stageIdx, err)
			}
			whereExpr = ex
		}

		env := make(map[string]any, 28+len(spec.Params))
		env["pi"] = math.Pi
		env["e"] = math.E
		for k, v := range spec.Params {
			if reservedTransformVars[k] {
				return nil, fmt.Errorf("transform_nodes[%d].params: %q is a reserved variable name", stageIdx, k)
			}
			if _, isBuiltin := fig2Builtin(k); isBuiltin {
				return nil, fmt.Errorf("transform_nodes[%d].params: %q conflicts with a built-in function", stageIdx, k)
			}
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return nil, fmt.Errorf("transform_nodes[%d].params.%s must be finite", stageIdx, k)
			}
			env[k] = v
		}

		var candidates []*Node
		if len(spec.NodeIDs) > 0 {
			seen := make(map[int]bool, len(spec.NodeIDs))
			candidates = make([]*Node, 0, len(spec.NodeIDs))
			for _, id := range spec.NodeIDs {
				if id <= 0 {
					return nil, fmt.Errorf("transform_nodes[%d]: invalid NODE %d", stageIdx, id)
				}
				if seen[id] {
					return nil, fmt.Errorf("transform_nodes[%d]: duplicate NODE %d in node_ids", stageIdx, id)
				}
				n := f.Nodes[id]
				if n == nil {
					return nil, fmt.Errorf("transform_nodes[%d]: unknown NODE %d", stageIdx, id)
				}
				seen[id] = true
				candidates = append(candidates, n)
			}
		} else {
			candidates = allNodes
		}

		modelBounds, _ := computeNodeBounds(allNodes)
		bindBounds(env, "model_", modelBounds)

		candBounds, _ := computeNodeBounds(candidates)
		bindBounds(env, "", candBounds)

		targets := candidates
		if whereExpr != nil {
			targets = make([]*Node, 0, len(candidates))
			for _, n := range candidates {
				env["id"] = float64(n.Num)
				env["x"] = n.Coord[0]
				env["y"] = n.Coord[1]
				env["z"] = n.Coord[2]
				ok, err := whereExpr.EvalBool(env)
				if err != nil {
					return nil, fmt.Errorf("transform_nodes[%d].where at NODE %d: %w", stageIdx, n.Num, err)
				}
				if ok {
					targets = append(targets, n)
				}
			}
		}
		if len(targets) == 0 {
			return nil, fmt.Errorf("transform_nodes[%d]: matched 0 nodes", stageIdx)
		}

		// Re-bind xmin..cz to the matched target set so expressions like
		// (x - cx) work naturally on partial selections (e.g. a roof plane)
		// while model_xmin..model_cz remain available for the whole frame.
		targetBounds, _ := computeNodeBounds(targets)
		bindBounds(env, "", targetBounds)

		nextCoords := make([][3]float64, len(targets))
		for i, n := range targets {
			env["id"] = float64(n.Num)
			env["x"] = n.Coord[0]
			env["y"] = n.Coord[1]
			env["z"] = n.Coord[2]
			coord := [3]float64{n.Coord[0], n.Coord[1], n.Coord[2]}
			for axis := 0; axis < 3; axis++ {
				if absExpr[axis] != nil {
					v, err := absExpr[axis].EvalFloat(env)
					if err != nil {
						return nil, fmt.Errorf("transform_nodes[%d].%s at NODE %d: %w", stageIdx, axisNames[axis], n.Num, err)
					}
					coord[axis] = v
				} else if relExpr[axis] != nil {
					dv, err := relExpr[axis].EvalFloat(env)
					if err != nil {
						return nil, fmt.Errorf("transform_nodes[%d].d%s at NODE %d: %w", stageIdx, axisNames[axis], n.Num, err)
					}
					v := n.Coord[axis] + dv
					if math.IsNaN(v) || math.IsInf(v, 0) {
						return nil, fmt.Errorf("transform_nodes[%d].d%s at NODE %d produced nonfinite coordinate", stageIdx, axisNames[axis], n.Num)
					}
					coord[axis] = v
				}
			}
			nextCoords[i] = coord
		}

		for i, n := range targets {
			copy(n.Coord, nextCoords[i][:])
			moved[n.Num] = true
		}
	}

	return moved, nil
}

// ValidateMovedNodes checks that node positions remain unique and that every
// element referencing any moved node retains positive finite length or area.
func (f *Frame) ValidateMovedNodes(moved map[int]bool) error {
	if len(moved) == 0 {
		return nil
	}
	if err := f.ValidateNodePositions(); err != nil {
		return err
	}
	for _, e := range f.Elems {
		touched := false
		for _, n := range e.Enod {
			if n != nil && moved[n.Num] {
				touched = true
				break
			}
		}
		if !touched {
			continue
		}
		if e.IsLineElem() {
			length := e.Length()
			if !(length > 0) || math.IsInf(length, 0) {
				return fmt.Errorf("ELEM %d has invalid length after node update", e.Num)
			}
		} else {
			area := e.Area()
			if !(area > 0) || math.IsInf(area, 0) || math.IsNaN(area) {
				return fmt.Errorf("ELEM %d has invalid area after node update", e.Num)
			}
		}
	}
	return nil
}
