package main

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"testing"
)

func TestAnalysisComparisonNakano(t *testing.T) {
	path := os.Getenv("ST_MCP_NAKANO_INP")
	if path == "" {
		t.Skip("set ST_MCP_NAKANO_INP to the nakano02_07.inp test model")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	session := modelSession{parsePath: t.TempDir() + "/source.inp", raw: data}
	session.frame, err = readModel(data, session.parsePath)
	if err != nil {
		t.Fatal(err)
	}
	section := 211
	proposal, err := session.preview([]edit{
		{Element: 1002, Section: &section},
		{Node: 102, Fixed: []bool{true, true, true, false, false, true}, Load: []float64{0, 0, -1.25, 0, 0, .125}},
	})
	if err != nil {
		t.Fatal(err)
	}

	var input, output bytes.Buffer
	request := analysisRequest{Options: analysisOptions{ProposalID: proposal.ID, Analysis: "both", Period: "L", NodeIDs: []int{102, 131}, Modes: 3}, Before: data, After: proposal.data}
	if err := json.NewEncoder(&input).Encode(request); err != nil {
		t.Fatal(err)
	}
	if err := runAnalysisWorker(t.TempDir(), &input, &output); err != nil {
		t.Fatal(err)
	}
	var event analysisEvent
	for output.Len() > 0 {
		line, err := output.ReadBytes('\n')
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(line, &event); err != nil {
			t.Fatal(err)
		}
	}
	r := event.Result
	if r == nil || len(r.Nodes) != 2 || len(r.Periods) != 3 {
		t.Fatalf("unexpected result: %+v", r)
	}
	assertNear := func(label string, got, want, tol float64) {
		t.Helper()
		if math.IsNaN(got) || math.Abs(got-want) > tol {
			t.Errorf("%s = %.12g, want %.12g", label, got, want)
		}
	}
	assertNear("node 102 support Z reaction change", r.Nodes[0].ReactionChange[2], 9.208140693, 1e-6)
	assertNear("node 131 Z displacement change", r.Nodes[1].DisplacementChange[2], -.002003794117, 1e-9)
	assertNear("period rank 1 before", r.Periods[0].Before, .878694277, 1e-6)
	assertNear("period rank 1 after", r.Periods[0].After, .898417898, 1e-6)
}
