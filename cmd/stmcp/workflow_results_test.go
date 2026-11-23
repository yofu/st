package main

import (
	st "github.com/yofu/st/stlib"
	"testing"
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
