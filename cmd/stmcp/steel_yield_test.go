package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	st "github.com/yofu/st/stlib"
)

func openYieldTestModel(t *testing.T) (*modelSession, string, func(string, any) any) {
	t.Helper()
	path := os.Getenv("ST_STEEL_YIELD_TEST_INP")
	if path == "" {
		t.Skip("set ST_STEEL_YIELD_TEST_INP to nakano02_07.inp")
	}
	dir := t.TempDir()
	t.Chdir(dir)
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { root.Close() })
	for _, ext := range []string{".inp", ".lst"} {
		data, err := os.ReadFile(st.Ce(path, ext))
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, "base"+ext), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	s := modelSession{root: root, parsePath: filepath.Join(dir, "mcp.inp")}
	call := func(name string, args any) any {
		t.Helper()
		raw, err := json.Marshal(args)
		if err != nil {
			t.Fatal(err)
		}
		v, err := s.call(name, raw)
		if err != nil {
			t.Fatal(name, err)
		}
		return v
	}
	call("open_model", map[string]string{"path": "base.inp"})
	return &s, dir, call
}

func TestMCPSteelYieldNakanoSaveAs(t *testing.T) {
	s, dir, call := openYieldTestModel(t)
	cmd := ":steelyield -elem=2399 -design=2 -factor=0.9"
	preview := func(c string) map[string]any {
		t.Helper()
		return call("preview_gui_edits", map[string]any{"commands": []map[string]string{{"command": c}}}).(map[string]any)
	}
	initial := slices.Clone(s.frame.Sects[511].Yield)
	p := preview(cmd)
	log := p["history"].(string)
	if p["proposal_id"] != nil || s.pending != nil || !strings.Contains(log, "4.0000 m (BTFAC 2") || !strings.Contains(log, "使用部材 78 本") {
		t.Fatalf("invalid preview: %s", log)
	}
	_, tail, ok := strings.Cut(log, "preview_id=")
	id, _, _ := strings.Cut(tail, "\n")
	if !ok || len(id) != 24 {
		t.Fatalf("missing review id")
	}
	_, confirmation, ok := strings.Cut(log, "確認後に実行  ")
	command, _, _ := strings.Cut(confirmation, "\n")
	if !ok || !strings.Contains(command, "-sect=511") || !strings.HasSuffix(command, "-apply="+id) {
		t.Fatal("missing replayable confirmation", command)
	}
	result := preview(command)
	if result["proposal_id"] == nil || s.pending == nil || !slices.Equal(s.frame.Sects[511].Yield, initial) {
		t.Fatal("missing or prematurely applied proposal", result)
	}
	c := st.SteelYieldCondition{StrongBucklingLength: 400, WeakBucklingLength: 400, LateralLength: 400, EndMomentRatio: -1, PlateBucklingFactor: 5.4, StrengthFactor: 0.9}
	want, err := s.frame.Sects[511].Allow.(*st.SGirder).HBeamYield(c)
	if err != nil {
		t.Fatal(err)
	}
	// Compare with the direct section formula at exactly the INP precision.
	for i, x := range want {
		want[i], err = strconv.ParseFloat(fmt.Sprintf("%.3f", x), 64)
		if err != nil {
			t.Fatal(err)
		}
	}
	call("save_model_as", map[string]any{"proposal_id": s.pending.ID, "path": "reviewed.inp"})
	unchanged, err := os.ReadFile(filepath.Join(dir, "base.inp"))
	if err != nil || !bytes.Equal(unchanged, s.raw) {
		t.Fatal("save-as changed the source", err)
	}
	call("open_model", map[string]string{"path": "reviewed.inp"})
	frame := s.frame
	if !slices.Equal(want[:], frame.Sects[511].Yield) || frame.Sects[511].Allow == nil {
		t.Fatalf("reopened SECT 511 %v, want %v", frame.Sects[511].Yield, want)
	}
	if err = frame.ExtractArclm(filepath.Join(dir, "weights.wgt")); err != nil {
		t.Fatal(err)
	}
	for _, period := range []string{"L", "X", "Y"} {
		found := false
		for _, s := range frame.Arclms[period].Sects {
			if s.Num == 511 {
				found = true
				if !slices.Equal(s.Yield, want[:]) {
					t.Fatalf("%s SECT 511 %v", period, s.Yield)
				}
			}
		}
		if !found {
			t.Fatal("missing section in " + period)
		}
	}
}
