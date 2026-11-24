package main

import (
	"encoding/json"
	st "github.com/yofu/st/stlib"
	"maps"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
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

func TestUpdateSectionsSyncsINPAndLST(t *testing.T) {
	f := st.NewFrame()
	if _, err := f.Construct(st.GUIConstruction{
		Properties: []st.BuildProperty{{ID: 101, Material: "SN400"}},
		Sections:   []st.BuildSection{{ID: 501, Name: "G51", Property: 101, Shape: "HKYOU", Dimensions: []float64{500, 200, 10, 16}}},
	}, nil); err != nil {
		t.Fatal(err)
	}
	origLST := []byte("CODE 501 S GIRDER \"G51\"\nHKYOU 50.0 20.0 1.00 1.6 SN400 \"H-500x200x10x16(KYOU)[mm]\"\nXFACE 0.0 0.0 \"FACE LENGTH Mx\"\nYFACE 0.0 0.0 \"FACE LENGTH My\"\n")
	if err := readLst(f, origLST, "test.lst"); err != nil {
		t.Fatal(err)
	}
	var q workflowConstruction
	if err := json.Unmarshal([]byte(`{"update_sections":[{"id":501,"shape":"HKYOU","dimensions_mm":[200,200,8,12]}]}`), &q); err != nil {
		t.Fatal(err)
	}
	r, updatedLST, err := q.onPrivateFrameWithLST(f, origLST, "test.lst")
	if err != nil {
		t.Fatal(err)
	}
	if r.UpdatedSections != 1 || !slices.Equal(r.SyncedLST, []int{501}) {
		t.Fatalf("unexpected receipt: %+v", r)
	}
	if got := f.Sects[501].Figs[0].Shape.String(); !strings.Contains(got, "20.0") || !strings.Contains(got, "0.80") || !strings.Contains(got, "1.2") {
		t.Fatalf("INP section shape not updated: %q", got)
	}
	lstStr := string(updatedLST)
	if !strings.Contains(lstStr, "HKYOU 20.0 20.0 0.80 1.2 SN400 \"H-200x200x8x12(KYOU)[mm]\"") || !strings.Contains(lstStr, "XFACE 0.0 0.0") {
		t.Fatalf("LST was not synced properly:\n%s", lstStr)
	}
	if f.Sects[501].Allow == nil || !strings.Contains(f.Sects[501].Allow.String(), "20.0") {
		t.Fatalf("frame Allow not reloaded: %v", f.Sects[501].Allow)
	}
	// Non-steel LST block: implicit sync skips, explicit sync_lst:true rejects.
	rcLST := []byte("CODE 501 RC GIRDER \"G51\"\nCRECT -20.0 -30.0 30.0 20.0 FC24\n")
	if _, _, err := q.onPrivateFrameWithLST(f, rcLST, "rc.lst"); err != nil {
		t.Fatalf("implicit sync should skip non-steel LST: %v", err)
	}
	syncTrue := true
	q.UpdateSections[0].SyncLST = &syncTrue
	if _, _, err := q.onPrivateFrameWithLST(f, rcLST, "rc.lst"); err == nil {
		t.Fatal("explicit sync_lst=true accepted non-steel LST")
	}
}

func TestGridPanelGridAndUpdateNodesInOneConstruct(t *testing.T) {
	f := st.NewFrame()
	if _, err := f.Construct(st.GUIConstruction{
		Properties: []st.BuildProperty{{ID: 101, Material: "SN400"}},
		Sections:   []st.BuildSection{{ID: 501, Name: "G51", Property: 101, Shape: "HKYOU", Dimensions: []float64{200, 200, 8, 12}}},
	}, nil); err != nil {
		t.Fatal(err)
	}
	var q workflowConstruction
	raw := `{
		"grid":{"origin_m":[0,0,0],"cells":[2,2,0],"pitch_m":[1,1,0],"first_node":101,"first_element":101,"type":"GIRDER","section":501},
		"plate_sections":[{"id":711,"dead_load_kgf_m2":300,"lload_tf_m2":[0.06,0.04,0.03]}],
		"panel_grid":{"plane":"XY","coordinate_m":0,"section":711,"type":"SLAB","expected_cells":4},
		"update_nodes":[
			{"id":102,"xyz_m":[1,0,0.2]},
			{"id":104,"xyz_m":[0,1,0.2]},
			{"id":105,"xyz_m":[1,1,0.5]},
			{"id":106,"xyz_m":[2,1,0.2]},
			{"id":108,"xyz_m":[1,2,0.2]}
		]
	}`
	if err := json.Unmarshal([]byte(raw), &q); err != nil {
		t.Fatal(err)
	}
	r, err := q.onPrivateFrame(f)
	if err != nil {
		t.Fatal(err)
	}
	if r.UpdatedNodes != 5 || r.Total.Nodes != 9 || r.Total.Elements != 16 || math.Abs(f.Nodes[105].Coord[2]-0.5) > 1e-9 {
		t.Fatalf("unexpected combined dome construct result: %+v z=%v", r, f.Nodes[105].Coord)
	}
	if r.PlateArea[711] <= 4.0 {
		t.Fatalf("expected curved slab area > 4.0 m2, got %v", r.PlateArea[711])
	}
	// Coincident node move must fail.
	bad := workflowConstruction{UpdateNodes: []st.BuildNode{{ID: 105, XYZ: []float64{0, 0, 0}}}}
	if _, err := bad.onPrivateFrame(f); err == nil {
		t.Fatal("accepted update_nodes moving onto an existing node")
	}
}

func TestTransformNodesCombinedWithGridPanelGridAndExplicitOverride(t *testing.T) {
	f := st.NewFrame()
	if _, err := f.Construct(st.GUIConstruction{
		Properties: []st.BuildProperty{{ID: 101, Material: "SN400"}},
		Sections:   []st.BuildSection{{ID: 501, Name: "G51", Property: 101, Shape: "HKYOU", Dimensions: []float64{200, 200, 8, 12}}},
	}, nil); err != nil {
		t.Fatal(err)
	}
	var q workflowConstruction
	raw := `{
		"grid":{"origin_m":[0,0,0],"cells":[4,4,0],"pitch_m":[1,1,0],"first_node":101,"first_element":101,"type":"GIRDER","section":501},
		"plate_sections":[{"id":711,"dead_load_kgf_m2":300,"lload_tf_m2":[0.06,0.04,0.03]}],
		"panel_grid":{"plane":"XY","coordinate_m":0,"section":711,"type":"SLAB","expected_cells":16},
		"transform_nodes":[
			{"params":{"amp":0.5},"z":"amp * sin(pi * (x - xmin) / (xmax - xmin)) * sin(pi * (y - ymin) / (ymax - ymin))"}
		],
		"update_nodes":[
			{"id":113,"xyz_m":[2,2,0.75]}
		]
	}`
	if err := json.Unmarshal([]byte(raw), &q); err != nil {
		t.Fatal(err)
	}
	r, err := q.onPrivateFrame(f)
	if err != nil {
		t.Fatal(err)
	}
	if r.UpdatedNodes != 25 || len(r.BoundsMin) != 3 || len(r.BoundsMax) != 3 {
		t.Fatalf("unexpected receipt: %+v", r)
	}
	if math.Abs(r.BoundsMin[2]) > 1e-9 || math.Abs(r.BoundsMax[2]-0.75) > 1e-9 {
		t.Fatalf("unexpected Z bounds: min=%v max=%v", r.BoundsMin, r.BoundsMax)
	}
	if math.Abs(f.Nodes[113].Coord[2]-0.75) > 1e-9 {
		t.Fatalf("explicit update_nodes should override transform_nodes: %v", f.Nodes[113].Coord)
	}
	if r.PlateArea[711] <= 16.0 {
		t.Fatalf("expected deformed slab area > 16.0, got %v", r.PlateArea[711])
	}

	// Collapsing all nodes to one point must be rejected by ValidateMovedNodes.
	collapse := workflowConstruction{TransformNodes: []st.NodeTransform{{X: "0", Y: "0", Z: "0"}}}
	if _, err := collapse.onPrivateFrame(f); err == nil {
		t.Fatal("accepted transform_nodes collapsing nodes onto one point")
	}
}

func TestLoadNodeUpdatesFileFormatsAndSecurity(t *testing.T) {
	dir := t.TempDir()
	s := &modelSession{rootPath: dir}
	t.Cleanup(func() {
		if s.root != nil {
			s.root.Close()
		}
	})

	csvPath := filepath.Join(dir, "coords.csv")
	csvContent := "\xef\xbb\xbf# comment line\nid,x,y,z\n101, 0, 0, 0.1\n\n102, 1, 0, 0.2\n"
	if err := os.WriteFile(csvPath, []byte(csvContent), 0600); err != nil {
		t.Fatal(err)
	}
	gotCSV, err := s.loadNodeUpdatesFile("coords.csv")
	if err != nil || len(gotCSV) != 2 || gotCSV[0].ID != 101 || math.Abs(gotCSV[1].XYZ[2]-0.2) > 1e-9 {
		t.Fatalf("csv parse failed: %v, %v", gotCSV, err)
	}

	csvNoHeader := filepath.Join(dir, "noheader.csv")
	if err := os.WriteFile(csvNoHeader, []byte("101,0,0,0.5\n102,1,0,0.6\n"), 0600); err != nil {
		t.Fatal(err)
	}
	gotNoHdr, err := s.loadNodeUpdatesFile("noheader.csv")
	if err != nil || len(gotNoHdr) != 2 || math.Abs(gotNoHdr[0].XYZ[2]-0.5) > 1e-9 {
		t.Fatalf("headerless csv failed: %v, %v", gotNoHdr, err)
	}

	jsonPath := filepath.Join(dir, "coords.json")
	if err := os.WriteFile(jsonPath, []byte(`[{"id":101,"xyz_m":[0,0,1.25]}]`), 0600); err != nil {
		t.Fatal(err)
	}
	gotJSON, err := s.loadNodeUpdatesFile("coords.json")
	if err != nil || len(gotJSON) != 1 || math.Abs(gotJSON[0].XYZ[2]-1.25) > 1e-9 {
		t.Fatalf("json parse failed: %v, %v", gotJSON, err)
	}

	jsonlPath := filepath.Join(dir, "coords.jsonl")
	if err := os.WriteFile(jsonlPath, []byte("# header\n{\"id\":101,\"xyz_m\":[0,0,0.3]}\n{\"id\":102,\"xyz_m\":[1,0,0.4]}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	gotJSONL, err := s.loadNodeUpdatesFile("coords.jsonl")
	if err != nil || len(gotJSONL) != 2 || math.Abs(gotJSONL[1].XYZ[2]-0.4) > 1e-9 {
		t.Fatalf("jsonl parse failed: %v, %v", gotJSONL, err)
	}

	// Verify schema validity and presence of new fields.
	var schema map[string]any
	if err := json.Unmarshal([]byte(constructionSchema), &schema); err != nil {
		t.Fatalf("invalid constructionSchema JSON: %v", err)
	}
	props := schema["properties"].(map[string]any)
	if props["transform_nodes"] == nil || props["update_nodes_file"] == nil {
		t.Fatalf("constructionSchema missing new bulk node edit properties")
	}

	// Security and error cases.
	badCSV := filepath.Join(dir, "bad.csv")
	if err := os.WriteFile(badCSV, []byte("101,0,0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	txtFile := filepath.Join(dir, "bad.txt")
	if err := os.WriteFile(txtFile, []byte("101,0,0,0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, badPath := range []string{"../escape.csv", csvPath, "bad.txt", "bad.csv", "missing.csv"} {
		if _, err := s.loadNodeUpdatesFile(badPath); err == nil {
			t.Fatalf("expected error for update_nodes_file %q", badPath)
		}
	}
}
