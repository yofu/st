package st

import (
	"math"
	"testing"
)

func TestApplyNodeTransformsAnalyticAndSimultaneous(t *testing.T) {
	f := NewFrame()
	if _, err := f.Construct(GUIConstruction{
		Properties: []BuildProperty{{ID: 101, Material: "SN400"}},
		Sections:   []BuildSection{{ID: 501, Name: "G51", Property: 101, Shape: "HKYOU", Dimensions: []float64{200, 200, 8, 12}}},
		Grid:       &BuildGrid{Origin: []float64{0, 0, 0}, Cells: []int{10, 10, 0}, Pitch: []float64{1, 1, 0}, FirstNode: 101, FirstElement: 101, Type: "GIRDER", Section: 501},
	}, nil); err != nil {
		t.Fatal(err)
	}

	// Stage 1: spherical dome via pure math expression (no shape-specific code).
	// Stage 2: simultaneous 90-degree XY rotation around (cx, cy) verifying pre-stage reads.
	moved, err := f.ApplyNodeTransforms([]NodeTransform{
		{
			Params: map[string]float64{"R": 30},
			Z:      "sqrt(R**2 - (x - cx)**2 - (y - cy)**2) - sqrt(R**2 - (xmax - cx)**2 - (ymax - cy)**2)",
		},
		{
			Params: map[string]float64{"deg": 90},
			X:      "cx + (x - cx)*cos(radians(deg)) - (y - cy)*sin(radians(deg))",
			Y:      "cy + (x - cx)*sin(radians(deg)) + (y - cy)*cos(radians(deg))",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(moved) != 121 {
		t.Fatalf("expected 121 moved nodes, got %d", len(moved))
	}
	if err := f.ValidateMovedNodes(moved); err != nil {
		t.Fatal(err)
	}

	// Node 161 is center (5, 5): stays at (5, 5), Z = 30 - sqrt(850).
	wantApexZ := 30.0 - math.Sqrt(850.0)
	c161 := f.Nodes[161].Coord
	if math.Abs(c161[0]-5) > 1e-9 || math.Abs(c161[1]-5) > 1e-9 || math.Abs(c161[2]-wantApexZ) > 1e-9 {
		t.Fatalf("unexpected apex node 161 coord: %v (want Z=%.6f)", c161, wantApexZ)
	}

	// Node 101 was (0, 0); rotated +90 deg around (5, 5) -> (10, 0), Z = 0.
	c101 := f.Nodes[101].Coord
	if math.Abs(c101[0]-10) > 1e-9 || math.Abs(c101[1]-0) > 1e-9 || math.Abs(c101[2]-0) > 1e-9 {
		t.Fatalf("unexpected rotated corner node 101 coord: %v", c101)
	}

	bmin, bmax := f.NodeBounds()
	if math.Abs(bmin[2]) > 1e-9 || math.Abs(bmax[2]-wantApexZ) > 1e-9 {
		t.Fatalf("unexpected bounds: min=%v max=%v", bmin, bmax)
	}
}

func TestApplyNodeTransformsWhereSubsetAndMathBuiltins(t *testing.T) {
	f := NewFrame()
	// Two layers: Z=0 (4 nodes) and Z=3 (4 nodes shifted in X by +2 so target cx != model_cx).
	for _, p := range [][]float64{
		{0, 0, 0}, {4, 0, 0}, {0, 4, 0}, {4, 4, 0},
		{2, 0, 3}, {6, 0, 3}, {2, 4, 3}, {6, 4, 3},
	} {
		f.AddNode(p[0], p[1], p[2])
	}
	// Model X range is [0, 6] (model_cx = 3); Z=3 subset X range is [2, 6] (cx = 4).
	moved, err := f.ApplyNodeTransforms([]NodeTransform{
		{
			Where: "z > 1",
			DZ:    "clamp((cx - model_cx) + 0.5*math.sin(pi/2) + 0*atan2(1, 1) + 0*hypot(3, 4) + 0*degrees(0) + 0*sign(-1) + 0*floor(0.9) + 0*ceil(0.1) + 0*exp(0) + 0*log(1), 0, 2)",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(moved) != 4 {
		t.Fatalf("expected 4 roof nodes moved, got %d", len(moved))
	}
	// cx - model_cx = 4 - 3 = 1, + 0.5*sin(pi/2) = 1.5 -> roof Z = 3 + 1.5 = 4.5.
	for _, n := range f.Nodes {
		if n.Coord[2] < 1 {
			if n.Coord[2] != 0 {
				t.Fatalf("base node %d should not have moved: %v", n.Num, n.Coord)
			}
		} else if math.Abs(n.Coord[2]-4.5) > 1e-9 {
			t.Fatalf("roof node %d expected Z=4.5, got %v", n.Num, n.Coord)
		}
	}
}

func TestApplyNodeTransformsRejectsErrorsAtomically(t *testing.T) {
	f := NewFrame()
	n1 := f.AddNode(0, 0, 0)
	n2 := f.AddNode(2, 0, 0)

	cases := []struct {
		name string
		spec NodeTransform
	}{
		{"empty_axes", NodeTransform{}},
		{"both_abs_and_rel", NodeTransform{X: "x + 1", DX: "1"}},
		{"reserved_param", NodeTransform{Params: map[string]float64{"cx": 1}, DZ: "1"}},
		{"builtin_param", NodeTransform{Params: map[string]float64{"sin": 1}, DZ: "1"}},
		{"unknown_node_id", NodeTransform{NodeIDs: []int{999}, DZ: "1"}},
		{"duplicate_node_id", NodeTransform{NodeIDs: []int{n1.Num, n1.Num}, DZ: "1"}},
		{"where_matches_none", NodeTransform{Where: "z > 100", DZ: "1"}},
		{"sqrt_domain_on_second_node", NodeTransform{Z: "sqrt(1 - x)"}},
		{"div_by_zero", NodeTransform{Z: "1 / x"}},
		{"asin_domain", NodeTransform{Z: "asin(x)"}},
		{"log_domain", NodeTransform{Z: "log(x)"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := f.ApplyNodeTransforms([]NodeTransform{tc.spec}); err == nil {
				t.Fatalf("expected error for %s", tc.name)
			}
			// Verify coordinates remained untouched after failure.
			if n1.Coord[0] != 0 || n1.Coord[2] != 0 || n2.Coord[0] != 2 || n2.Coord[2] != 0 {
				t.Fatalf("coordinates mutated after failed transform %s: n1=%v n2=%v", tc.name, n1.Coord, n2.Coord)
			}
		})
	}
}
