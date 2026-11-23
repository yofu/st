package main

import (
	"encoding/json"
	st "github.com/yofu/st/stlib"
	"maps"
	"math"
	"slices"
	"testing"
)

func TestSurfaceWeightsPreserveStructuralLayers(t *testing.T) {
	f := st.NewFrame()
	p, err := f.ParseProp([]string{"PROP", "401", "PNAME", "concrete", "HIJU", "2.4", "E", "2000000"}, false)
	if err != nil {
		t.Fatal(err)
	}
	s := f.AddSect(711)
	structural := st.NewFig()
	structural.Prop, structural.Value["THICK"] = p, 0.1
	s.Figs = append(s.Figs, structural)
	dead := 300.0
	q := plateSectionRequest{ID: 711, Dead: &dead, Live: []float64{.065, .045, .035}}
	for _, target := range []float64{300, 310, 240} {
		dead = target
		if _, err := q.apply(f); err != nil {
			t.Fatal(err)
		}
		if len(s.Figs) != 2 || s.Figs[0] != structural || structural.Prop != p || structural.Value["THICK"] != .1 || math.Abs(plateLoads(s).Dead-target) > 1e-9 || f.Props[401] != p || len(f.Props) != 2 || !slices.Equal(s.Lload, q.Live) {
			t.Fatalf("incorrect section after setting %.1f kgf/m2", target)
		}
	}
	dead = 100
	if _, err := q.apply(f); err == nil {
		t.Fatal("accepted a fixed load below structural weight")
	}
}
func TestSurfaceCellsAndExplicitPlates(t *testing.T) {
	f := st.NewFrame()
	for _, p := range [][]float64{{0, 0, 0}, {0, 2, 0}, {0, 2, 3}, {0, 0, 3}} {
		f.AddNode(p[0], p[1], p[2])
	}
	ids := slices.Sorted(maps.Keys(f.Nodes))
	sec := f.AddSect(201)
	for i := range ids {
		f.AddLineElem(-1, []*st.Node{f.Nodes[ids[i]], f.Nodes[ids[(i+1)%4]]}, sec, st.GIRDER)
	}
	var q workflowConstruction
	if err := json.Unmarshal([]byte(`{"plate_sections":[{"id":801,"dead_load_kgf_m2":100}],"panel_grid":{"type":"WALL","section":801,"plane":"YZ","coordinate_m":0,"expected_cells":1}}`), &q); err != nil {
		t.Fatal(err)
	}
	result, err := q.onPrivateFrame(f)
	if err != nil || result.PlateArea[801] != 6 {
		t.Fatalf("wall: %v, %v", result, err)
	}
	for _, bad := range [][]int{{ids[2], ids[1], ids[0], ids[3]}, {ids[0], ids[1], 9999}, {ids[0], ids[0], ids[1]}} {
		q = workflowConstruction{Plates: []st.BuildElement{{Nodes: bad, Type: "SLAB", Section: 801}}}
		if _, err := q.onPrivateFrame(f); err == nil {
			t.Fatalf("invalid plate accepted: %v", bad)
		}
	}
}
func TestNativeCommandLookup(t *testing.T) {
	for _, tt := range []struct{ query, group string }{{"bclng", "ex"}, {"srcanrate", "fig2"}, {"ADDPLATEALL", "editor_commands"}} {
		got, err := searchModelCommands(tt.query)
		if err != nil || len(got[tt.group]) == 0 {
			t.Fatalf("%s: %v %v", tt.query, got, err)
		}
	}
}
