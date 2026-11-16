package main

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestGUIEditRoundTrip(t *testing.T) {
	root := t.TempDir()
	const base = "\"edit test\"\r\nPROP 1 PNAME STEEL\r\n E 21000000 POI 0.3\r\nSECT 101 SNAME TEST\r\n NFIG 1\r\n FIG 1 FPROP 1\r\n AREA 0.0123456789 IXX 0.001 IYY 0.001 VEN 0.001\r\nNODE 10 CORD 0.123456789 0 0\r\n# keep this comment\r\nNODE 11 CORD 2 0 0\r\nNODE 12 CORD 4 1 0\r\nELEM 21 ESECT 101 ENODS 2 ENOD 10 11 TYPE GIRDER\r\n"
	if err := os.WriteFile(filepath.Join(root, "base.inp"), []byte(base), 0600); err != nil {
		t.Fatal(err)
	}
	dir, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	s := modelSession{root: dir, parsePath: filepath.Join(root, "placeholder.inp")}
	if _, err := s.open("base.inp"); err != nil {
		t.Fatal(err)
	}

	in := []byte(`{"commands":[
		{"command":":move 0 0 0.00012345","node_ids":[12],"element_ids":[]},
		{"command":"ADDLINEELEM","inputs":[{"kind":"node","id":11},{"kind":"node","id":12}]},
		{"command":"BONDPIN","element_ids":[21]},
		{"command":"CONFPIN","node_ids":[10]},
		{"command":"KIJUN","inputs":[{"kind":"node","id":10},{"kind":"node","id":11},{"kind":"text","text":"通り A"}]},
		{"command":"MEASURE","inputs":[{"kind":"node","id":10},{"kind":"node","id":11},{"kind":"key","text":"X"}]},
		{"command":":lock","node_ids":[11],"element_ids":[21]}
	]}`)
	v, err := s.call("preview_gui_edits", in)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.(map[string]any)["changes"].([]change)) < 6 || s.pending == nil {
		t.Fatalf("missing changes: %v", v)
	}
	first := s.pending.ID
	for _, bad := range []string{
		`{"commands":[{"command":"KIJUN","inputs":[{"kind":"node","id":10},{"kind":"node","id":11}]}]}`,
		`{"commands":[{"command":"TRIM","inputs":[{"kind":"node","id":10}]}]}`,
		`{"commands":[{"command":":move 1 0 0","node_ids":[12]},{"command":":write overwritten.inp"}]}`,
		`{"commands":[{"command":":insert ../outside.inp 0","node_ids":[12]}]}`,
	} {
		if _, err := s.call("preview_gui_edits", []byte(bad)); err == nil || s.pending.ID != first {
			t.Fatalf("unexpected failure handling for %s: %v", bad, err)
		}
	}
	if _, err := s.call("preview_gui_edits", []byte(`{"proposal_id":1,"commands":[{"command":":move 0 0 1","node_ids":[11],"element_ids":[]}]}`)); err != nil {
		t.Fatal(err)
	}
	if s.pending.ID != first+1 {
		t.Fatal("proposal was not advanced")
	}
	saved := s.pending.data
	if !bytes.Contains(saved, []byte("# keep this comment")) || !bytes.Contains(saved, []byte("AREA 0.0123456789")) {
		t.Fatalf("source contents lost: %s", saved)
	}
	if _, err := s.call("save_model_as", []byte(`{"proposal_id":2,"path":"saved.inp"}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.open("saved.inp"); err != nil {
		t.Fatal(err)
	}
	f := s.frame
	if len(f.Elems) != 2 || f.Nodes[10].Coord[0] != 0.123456789 || f.Nodes[12].Coord[2] != 0.00012345 || !slices.Equal(f.Nodes[10].Conf, []bool{true, true, true, false, false, false}) ||
		f.Nodes[11].Coord[2] != 1 || !f.Elems[21].Lock || f.Elems[21].Bonds[4] == nil ||
		len(f.Kijuns) != 1 || f.Kijuns["通り a"] == nil || len(f.Measures) != 1 {
		t.Fatalf("incorrect round trip: elems %d coords %v %v %v kijun %v measures %d", len(f.Elems), f.Nodes[10].Coord, f.Nodes[11].Coord, f.Nodes[12].Coord, f.Kijuns, len(f.Measures))
	}
	original, err := os.ReadFile(filepath.Join(root, "base.inp"))
	if err != nil || string(original) != base {
		t.Fatal("original changed", err)
	}
}

func TestGUIAddNodePrecisionAndChains(t *testing.T) {
	const source = "\"compound\"\nPROP 1 PNAME STEEL\n E 21000000 POI 0.3\nSECT 101 SNAME TEST\n NFIG 1\n FIG 1 FPROP 1\n AREA 1 IXX 1 IYY 1 VEN 1\nNODE 1 CORD 0 0 0\nNODE 2 CORD 2 0 0\nNODE 3 CORD 4 0 0\nCHAIN {\nELEM 10 ESECT 101 ENODS 2 ENOD 1 2 TYPE GIRDER\nELEM 11 ESECT 101 ENODS 2 ENOD 2 3 TYPE GIRDER\n}\n"
	s := modelSession{raw: []byte(source), parsePath: filepath.Join(t.TempDir(), "model.inp")}
	var err error
	s.frame, err = readModel(s.raw, s.parsePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.call("preview_gui_edits", []byte(`{"commands":[{"command":":addnode 6.0123456789 0 0"},{"command":":conf 111000","node_ids":[1]}]}`)); err != nil {
		t.Fatal(err)
	}
	f, err := readModel(s.pending.data, s.parsePath)
	if err != nil || len(f.Nodes) != 4 || len(f.Chains) != 1 || f.Nodes[f.Maxnnum].Coord[0] != 6.0123456789 || !strings.Contains(string(s.pending.data), "CHAIN {\nELEM 10") {
		t.Fatalf("chain round trip: %v (%s)", err, s.pending.data)
	}
}

func TestGUISectionAndUndo(t *testing.T) {
	s := modelSession{raw: []byte("PROP 1 PNAME STEEL\n E 21000000 POI 0.3\nNODE 1 CORD 0 0 0\n"), parsePath: filepath.Join(t.TempDir(), "model.inp")}
	var err error
	s.frame, err = readModel(s.raw, s.parsePath)
	if err != nil {
		t.Fatal(err)
	}
	script := []byte(`{"commands":[
        {"command":":add sect 95"},
        {"command":"ADD_SECTION_FIG 95 1"},
        {"command":"EDIT_SECTION_FIG 95 1 AREA=0.01234567891 IXX=0.00000123456789"},
        {"command":"SECTION_COLOR 95 10 20 30"},
        {"command":"UNDO"},
        {"command":"REDO"}
    ]}`)
	if _, err = s.call("preview_gui_edits", script); err != nil {
		t.Fatal(err)
	}
	f, err := readModel(s.pending.data, s.parsePath)
	if err != nil {
		t.Fatal(err)
	}
	section := f.Sects[95]
	if section == nil || section.Color != 0x0a141e || section.Figs[0].Value["AREA"] != 0.01234567891 {
		t.Fatalf("section edit was not restored: %+v", section)
	}
}

func TestGUIArcAndProjectionInputs(t *testing.T) {
	s := modelSession{raw: []byte("NODE 1 CORD 0 0 0\nNODE 2 CORD 1 1 0\nNODE 3 CORD 2 0 0\n"), parsePath: filepath.Join(t.TempDir(), "model.inp")}
	var err error
	s.frame, err = readModel(s.raw, s.parsePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.call("preview_gui_edits", []byte(`{"commands":[
        {"command":"ADDARC","inputs":[{"kind":"node","id":1},{"kind":"node","id":2},{"kind":"node","id":3},{"kind":"finish"}]},
        {"command":"FENCE","inputs":[{"kind":"fence","point":[0,0,0],"end_point":[2,0,0]}]}
    ]}`)); err != nil {
		t.Fatal(err)
	}
	f, err := readModel(s.pending.data, s.parsePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Arcs) != 1 || f.Arcs[0].Radius != 1 {
		t.Fatal("arc was not restored")
	}
}
