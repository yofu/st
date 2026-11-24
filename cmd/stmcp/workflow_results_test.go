package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	st "github.com/yofu/st/stlib"
)

func TestWorkflowDisplayedModeProvenance(t *testing.T) {
	a := &st.GUIAnalysisOutput{Kind: "buckling", Path: "model.otp", BeforeRead: 4, Eigenvalues: []float64{25.69755000525501}}
	w := modelWorkflow{result: workflowResult{Commands: &st.GUICommandStatus{Timing: []st.GUICommandTiming{{Analysis: a}}}}}
	rounded := 25.6976
	good := st.GUIResultView{Path: a.Path, NonzeroNodes: 2, Period: "B1", ModeVersion: 5, Eigenvalue: &rounded}
	if err := w.verifyDisplayedMode(&good); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*st.GUIResultView){func(v *st.GUIResultView) { v.ModeVersion = 4 }, func(v *st.GUIResultView) { v.Path = "old.otp" }, func(v *st.GUIResultView) { v.Period = "B2" }, func(v *st.GUIResultView) { v.NonzeroNodes = 0 }, func(v *st.GUIResultView) { n := 0.; v.Eigenvalue = &n }} {
		bad := good
		change(&bad)
		if err := w.verifyDisplayedMode(&bad); err == nil {
			t.Fatalf("accepted stale or missing mode: %+v", bad)
		}
	}
}

func TestWorkflowPreAndPostCommandsValidation(t *testing.T) {
	s := &modelSession{pairedGUI: "conn1"}
	for _, raw := range []string{
		`{"pre_commands":[]}`,
		`{"post_commands":[]}`,
		`{"pre_commands":["  "]}`,
		`{"post_commands":[""]}`,
		`{"pre_commands":[":propsect a.inp"],"proposal_id":1}`,
	} {
		if _, err := s.startWorkflow(json.RawMessage(raw)); err == nil {
			t.Fatalf("accepted invalid workflow request: %s", raw)
		}
	}
	a := workflowRequest{PreCommands: []string{":propsect a.inp"}, PostCommands: []string{":srcal -skipshort a.rat"}}
	b := workflowRequest{PreCommands: []string{":propsect b.inp"}, PostCommands: []string{":srcal -skipshort a.rat"}}
	if a.fingerprint() == b.fingerprint() {
		t.Fatal("pre_commands did not affect workflow fingerprint")
	}
}

func TestStaticAnalysisAutoBoundaryAndOverwrite(t *testing.T) {
	dir := t.TempDir()
	f := st.NewFrame()
	if _, err := f.Construct(st.GUIConstruction{
		Properties: []st.BuildProperty{{ID: 101, Material: "SN400"}},
		Sections:   []st.BuildSection{{ID: 501, Name: "G51", Property: 101, Shape: "HKYOU", Dimensions: []float64{200, 200, 8, 12}}},
		Grid:       &st.BuildGrid{Origin: []float64{0, 0, 0}, Cells: []int{1, 1, 0}, Pitch: []float64{2, 2, 0}, FirstNode: 101, FirstElement: 101, Type: "GIRDER", Section: 501},
	}, nil); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int{101, 102, 103, 104} {
		copy(f.Nodes[id].Conf, []bool{true, true, true, false, false, false})
	}
	inp := f.GUIInpBytes()
	// Worker must auto-configure 1-floor boundary for period L when Ai.Boundary is unset.
	var in, out bytes.Buffer
	req := analysisRequest{
		Options:      analysisOptions{Analysis: "static", Period: "L", NodeIDs: []int{101}},
		After:        inp,
		SingleStatic: true,
	}
	if err := json.NewEncoder(&in).Encode(req); err != nil {
		t.Fatal(err)
	}
	workDir := t.TempDir()
	if err := runAnalysisWorker(workDir, &in, &out); err != nil {
		t.Fatalf("auto-boundary static L failed: %v", err)
	}
	var lastEvent analysisEvent
	for out.Len() > 0 {
		line, err := out.ReadBytes('\n')
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(line, &lastEvent); err != nil {
			t.Fatal(err)
		}
	}
	if lastEvent.Result == nil || lastEvent.Result.Static == nil || len(lastEvent.Result.Static.Nodes) != 1 {
		t.Fatalf("unexpected static worker result: %+v", lastEvent.Result)
	}
	if lastEvent.Result.Static.Nodes[0].Reaction[2] <= 0 {
		t.Fatalf("expected positive vertical reaction from self-weight, got %v", lastEvent.Result.Static.Nodes[0].Reaction)
	}

	// Verify staticAnalysisOutput.save respects overwrite flag and startStaticAnalysis validation.
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	outFirst := &staticAnalysisOutput{root: root, path: "out_L.otl", overwrite: false}
	var res staticAnalysisResult
	if err := outFirst.save(workDir, &res); err != nil {
		t.Fatalf("initial save failed: %v", err)
	}
	if err := outFirst.save(workDir, &res); err == nil {
		t.Fatal("save without overwrite accepted existing file")
	}
	outOverwrite := &staticAnalysisOutput{root: root, path: "out_L.otl", overwrite: true}
	if err := outOverwrite.save(workDir, &res); err != nil {
		t.Fatalf("overwrite save failed: %v", err)
	}
	// Also verify startStaticAnalysis rejects existing path without overwrite:true.
	if err := os.WriteFile(filepath.Join(dir, "single.inp"), inp, 0644); err != nil {
		t.Fatal(err)
	}
	s := &modelSession{root: root, rootPath: dir}
	if _, err := s.open("single.inp"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.startStaticAnalysis(staticAnalysisOptions{Period: "L", ResultPath: "out_L.otl"}); err == nil {
		t.Fatal("startStaticAnalysis without overwrite accepted existing result_path")
	}
}
