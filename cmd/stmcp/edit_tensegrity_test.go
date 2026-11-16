package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	st "github.com/yofu/st/stlib"
)

func TestTensegrityRealIUPSaveParity(t *testing.T) {
	input := os.Getenv("ST_TEST_TENSEGRITY")
	if input == "" {
		t.Skip("set ST_TEST_TENSEGRITY to tens09.inp")
	}
	const tension = 601
	for _, c := range []struct {
		name        string
		ids         []int
		selected    []int
		keep        int
		compression int
	}{
		{"add", nil, []int{1404}, 2, 504},
		{"connected", nil, []int{7265}, 0, 510},
		{"delete", []int{504}, nil, 1, 201},
	} {
		t.Run(c.name, func(t *testing.T) {
			runDir := t.TempDir()
			for _, ext := range []string{".inp", ".otl", ".ohx", ".ohy"} {
				data, err := os.ReadFile(strings.TrimSuffix(input, filepath.Ext(input)) + ext)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(runDir, "source"+ext), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			root, err := os.OpenRoot(runDir)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			s := modelSession{root: root, parsePath: filepath.Join(runDir, "placeholder.inp")}
			if _, err = s.open("source.inp"); err != nil {
				t.Fatal(err)
			}
			ref, err := readModel(s.raw, s.parsePath)
			if err != nil {
				t.Fatal(err)
			}
			w := &editWindow{frame: ref, selection: st.NewSelection()}
			nodes, err := selectIDs(c.ids, ref.Nodes, "node")
			if err != nil {
				t.Fatal(err)
			}
			elems, err := selectIDs(c.selected, ref.Elems, "element")
			if err != nil {
				t.Fatal(err)
			}
			command := guiEdit{Command: fmt.Sprintf("TENSEGRITY%s %d %d %d", strings.ToUpper(c.name), c.compression, tension, c.keep), Confirm: true, Nodes: &c.ids, Elems: &c.selected}
			var commands []guiEdit
			var wantIDs []int
			switch c.name {
			case "add":
				candidates, err := st.TensegrityAddCandidates(ref, elems, c.compression, tension, c.keep)
				if err != nil || len(candidates) != 4 {
					t.Fatalf("expected four candidates: %d %v", len(candidates), err)
				}
				for _, el := range candidates {
					ref.AddElem(-1, el)
				}
			case "delete":
				if _, err = s.previewGUI(guiEditArgs{Commands: []guiEdit{command}}); err == nil || !strings.Contains(err.Error(), "lacks valid L") {
					t.Fatalf("missing force results must stop deletion: %v", err)
				}
				for _, ext := range []string{".otl", ".ohx", ".ohy"} {
					path := "source" + ext
					if err := ref.ReadResult(filepath.Join(runDir, path), st.UpdateResult); err != nil {
						t.Fatal(err)
					}
					commands = append(commands, guiEdit{Command: ":read result " + path})
				}
				candidates, err := st.TensegrityDeleteCandidates(ref, nodes, tension, c.keep)
				if err != nil || len(candidates) != 1 {
					t.Fatalf("expected one candidate: %d %v", len(candidates), err)
				}
				wantIDs = []int{candidates[0].Num}
				preview := command
				preview.Confirm = false
				inspect, err := s.previewGUI(guiEditArgs{Commands: append(append([]guiEdit(nil), commands...), preview)})
				if err != nil {
					t.Fatal(err)
				}
				if got := inspect.(map[string]any)["element_ids"]; !reflect.DeepEqual(got, wantIDs) {
					t.Fatalf("deletion candidates %v, want %v", got, wantIDs)
				}
				w.SelectElem(candidates)
				st.DeleteSelected(w)
			case "connected":
				wantIDs = existingIDs(st.TensegrityConnected(ref, elems[0], c.compression, tension), ref.Elems, func(el *st.Elem) int { return el.Num })
				if !reflect.DeepEqual(wantIDs, []int{1783, 1805}) {
					t.Fatalf("expected multiple compression members: %v", wantIDs)
				}
			}
			ref.View.Set(1)
			if err := st.SaveFile(w, filepath.Join(runDir, "iup.inp")); err != nil {
				t.Fatal(err)
			}
			rpcCommands := make([]map[string]any, 0, len(commands)+1)
			for _, item := range commands {
				rpcCommands = append(rpcCommands, map[string]any{"command": item.Command})
			}
			operation := map[string]any{"command": command.Command, "confirm": true}
			if c.ids != nil {
				operation["node_ids"] = c.ids
			}
			if c.selected != nil {
				operation["element_ids"] = c.selected
			}
			rpcCommands = append(rpcCommands, operation)
			raw, err := json.Marshal(map[string]any{"commands": rpcCommands})
			if err != nil {
				t.Fatal(err)
			}
			value, err := s.call("preview_gui_edits", raw)
			if err != nil {
				t.Fatal(err)
			}
			result := value.(map[string]any)
			if c.name == "connected" {
				if !reflect.DeepEqual(result["element_ids"], wantIDs) || result["proposal_id"] != nil {
					t.Fatalf("connected selection %v, want %v", result["element_ids"], wantIDs)
				}
				return
			}
			if _, err = s.save(s.pending.ID, "mcp.inp"); err != nil {
				t.Fatal(err)
			}
			if _, err = s.open("mcp.inp"); err != nil {
				t.Fatal(err)
			}
			actual := s.frame
			if _, err = s.open("iup.inp"); err != nil {
				t.Fatal(err)
			}
			if !sameModelTopology(s.frame, actual) {
				t.Fatal("IUP and MCP outputs have different nodes or connectivity")
			}
			for _, diff := range diffEditorINP(editorINP(s.frame), editorINP(actual)) {
				record := diff.Before
				if record == nil {
					record = diff.After
				}
				key := record.(map[string]string)["record"]
				if key != "HEADER DISTS" && key != "HEADER GFACT" && key != "HEADER FOCUS" && key != "HEADER ANGLE" {
					t.Fatalf("model record differs: %+v", diff)
				}
			}
			t.Logf("%d nodes, %d elements, %d sections, deletion candidates %v", len(actual.Nodes), len(actual.Elems), len(actual.Sects), wantIDs)
		})
	}
}
