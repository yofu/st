package main

import (
	"slices"
	"strings"
	"testing"
)

func TestMCPRCYieldNakanoSaveAs(t *testing.T) {
	s, _, call := openYieldTestModel(t)
	preview := func(command string) map[string]any {
		return call("preview_gui_edits", map[string]any{"commands": []map[string]string{{"command": command}}}).(map[string]any)
	}
	before := slices.Clone(s.frame.Sects[301].Yield)
	p := preview(":rcyield -elem=3680 -qy=865.502,1014.57")
	log := p["history"].(string)
	if !strings.Contains(log, "QX 保留") || p["proposal_id"] != nil || !slices.Equal(before, s.frame.Sects[301].Yield) {
		t.Fatal(log)
	}
	_, tail, ok := strings.Cut(log, "確認後に実行  ")
	command, _, _ := strings.Cut(tail, "\n")
	if !ok || !strings.HasPrefix(command, ":rcyield -sect=301") || !strings.Contains(command, "-member=3680") {
		t.Fatal(log)
	}
	applied := preview(command)
	if applied["proposal_id"] == nil || s.pending == nil || !slices.Equal(before, s.frame.Sects[301].Yield) {
		t.Fatal(applied)
	}
	call("save_model_as", map[string]any{"proposal_id": s.pending.ID, "path": "rc.inp"})
	call("open_model", map[string]string{"path": "rc.inp"})
	want := []float64{2008.368, -141.876, before[2], before[3], 360.528, -360.528, 27.931, -27.931, 689.811, -689.811, 161.747, -161.747}
	if got := s.frame.Sects[301].Yield; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
