package stgui

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/yofu/go-iup/iup"
	st "github.com/yofu/st/stlib"
)

func TestMCPStaticResultOnIUP(t *testing.T) { testMCPOnIUP(t, false) }

func TestMCPLiveEditingOnIUP(t *testing.T) { testMCPOnIUP(t, true) }

func testMCPOnIUP(t *testing.T, liveEditing bool) {
	exe, input := os.Getenv("ST_MCP_EXE"), os.Getenv("ST_MCP_NAKANO_INP")
	if exe == "" || input == "" {
		t.Skip("set ST_MCP_EXE and ST_MCP_NAKANO_INP for the Windows IUP integration test")
	}
	model, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile("model.inp", model, 0600); err != nil {
		t.Fatal(err)
	}
	if lst, err := os.ReadFile(st.Ce(input, ".lst")); err == nil {
		if err := os.WriteFile("model.lst", lst, 0600); err != nil {
			t.Fatal(err)
		}
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := iup.Open(); err != nil {
		t.Fatal(err)
	}
	defer iup.Close()
	w := NewWindow(dir)
	defer StopLogging()
	defer w.Dlg.Destroy()
	if err := w.guiConnectionCommand(""); err != nil {
		t.Fatal(err)
	}
	defer w.stopGUIConnection()
	if liveEditing {
		f := st.NewFrame()
		if err := f.ReadInpSnapshot(model, filepath.Join(dir, "model.inp")); err != nil {
			t.Fatal(err)
		}
		if data, err := os.ReadFile("model.lst"); err == nil {
			if err := f.ReadLstString(string(data), filepath.Join(dir, "model.lst")); err != nil {
				t.Fatal(err)
			}
		}
		// Model a GUI edit that has not been saved or added to undo yet.
		f.Nodes[131].Coord[0] += .123456789
		f.Sects[201].Allow.SetName("UNSAVED_SECTION")
		w.SetFrame(f)
		w.Changed(true)
		w.SelectNode([]*st.Node{f.Nodes[131]})
		w.SelectElem([]*st.Elem{f.Elems[1001]})
	}
	history := w.hist.GetAttribute("VALUE")
	_, suffix, ok := strings.Cut(history, "MCP GUI connection ID: ")
	if !ok || len(suffix) < 32 {
		t.Fatal("missing connection ID in IUP history")
	}
	id := suffix[:32]

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		if liveEditing {
			result <- exerciseMCPLiveGUI(ctx, exe, dir, id)
		} else {
			result <- exerciseMCPStaticGUI(ctx, exe, dir, id)
		}
	}()
	for {
		select {
		case err := <-result:
			if err != nil {
				t.Fatal(err)
			}
			f := w.Frame()
			if liveEditing {
				if f.Sects[201].Allow.Name() != "UNSAVED_SECTION" || f.Show.SrcanRate != 15 || f.Show.ColorMode != st.ECOLOR_RATE {
					t.Fatal("native commands lost unsaved LST or rate diagram state")
				}
				t.Log("native :srcal, 'srcanrate and 'srcancolor completed with unsaved geometry, loads and LST; .rat output verified")
				if !w.IsChanged() || f.Show.PlotState&st.PLOT_DEFORMED == 0 || f.Show.Dfact != 75 || f.Show.Stress[st.COLUMN]&1 == 0 || len(w.SelectedNodes()) != 1 || w.SelectedNodes()[0] != f.Nodes[131] {
					t.Fatal("live edit/undo lost display settings, dirty state or selection")
				}
				// An intervening GUI edit must reject the stale expected revision.
				baseline := f.GUIRevision()
				digest, baselineINP := f.GUIModelDigest(), f.GUIInpBytes()
				f.Nodes[131].Load[0] += 1
				reply := w.handleGUIRequest(st.GUIRequest{Action: "undo", Revision: baseline})
				if reply.Error == "" || f.Nodes[131].Load[0] == 0 {
					t.Fatal("stale GUI revision was accepted")
				}
				reply = w.handleGUIRequest(st.GUIRequest{Action: "apply", BasePath: f.Path, BaseDigest: digest, INP: baselineINP})
				if reply.Error == "" {
					t.Fatal("stale proposal overwrote intervening GUI edit")
				}
				t.Logf("real IUP unsaved edit, selection, MCP edit, display, undo/redo and revision conflict verified (%d nodes, %d members)", len(f.Nodes), len(f.Elems))
				return
			}
			n := f.Nodes[131]
			if n == nil || math.Abs(n.ReturnDisp("L", 2)) < 1e-6 || n.Pcoord[1] == n.Dcoord[1] {
				t.Fatal("IUP did not project the loaded deformation")
			}
			t.Logf("IUP node 131 dZ = %.12g m, projected Y %.3f -> %.3f", n.ReturnDisp("L", 2), n.Pcoord[1], n.Dcoord[1])
			return
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		default:
			iup.LoopStep()
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func exerciseMCPStaticGUI(ctx context.Context, exe, root, id string) error {
	cmd := exec.CommandContext(ctx, exe, "-root", root)
	in, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		in.Close()
		return err
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		in.Close()
		return err
	}
	defer func() { in.Close(); _ = cmd.Wait() }()
	r, enc := bufio.NewReader(out), json.NewEncoder(in)
	var count int
	call := func(method string, params any, dest any) error {
		count++
		if err := enc.Encode(map[string]any{"jsonrpc": "2.0", "id": count, "method": method, "params": params}); err != nil {
			return err
		}
		line, err := r.ReadBytes('\n')
		if err != nil {
			return err
		}
		var reply struct {
			Error  json.RawMessage `json:"error"`
			Result struct {
				IsError    bool            `json:"isError"`
				Content    json.RawMessage `json:"content"`
				Structured json.RawMessage `json:"structuredContent"`
			} `json:"result"`
		}
		if err := json.Unmarshal(line, &reply); err != nil {
			return err
		}
		if len(reply.Error) > 0 || reply.Result.IsError {
			return fmt.Errorf("%s: %s %s", method, reply.Error, reply.Result.Content)
		}
		if dest != nil {
			return json.Unmarshal(reply.Result.Structured, dest)
		}
		return nil
	}
	tool := func(name string, args map[string]any, dest any) error {
		return call("tools/call", map[string]any{"name": name, "arguments": args}, dest)
	}
	if err := call("initialize", map[string]string{"protocolVersion": "2025-11-25"}, nil); err != nil {
		return err
	}
	var opened struct {
		GUI st.GUIResponse `json:"gui"`
	}
	if err := tool("open_model", map[string]any{"path": "model.inp", "connection_id": id}, &opened); err != nil {
		return err
	}
	if filepath.Base(opened.GUI.Path) != "model.inp" || opened.GUI.Changed {
		return fmt.Errorf("unexpected IUP open response: %+v", opened.GUI)
	}
	var job struct {
		ID    uint64 `json:"job_id"`
		State string `json:"state"`
		Path  string `json:"result_path"`
	}
	if err := tool("start_static_analysis", map[string]any{"period": "L", "result_path": "MCP result.otl", "node_ids": []int{131}}, &job); err != nil {
		return err
	}
	args := map[string]any{"job_id": job.ID}
	for job.State == "running" {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(25 * time.Millisecond):
		}
		if err := tool("analysis_status", args, &job); err != nil {
			return err
		}
	}
	if job.State != "completed" || job.Path != "MCP result.otl" {
		return fmt.Errorf("static analysis did not save output: %+v", job)
	}
	var results struct {
		Static struct {
			Nodes []struct {
				Displacement [6]float64 `json:"displacement"`
			} `json:"nodes"`
		} `json:"static"`
	}
	if err := tool("analysis_results", args, &results); err != nil {
		return err
	}
	if len(results.Static.Nodes) != 1 || math.Abs(results.Static.Nodes[0].Displacement[2]) < 1e-6 {
		return fmt.Errorf("missing nonzero displacement for node 131")
	}
	args["connection_id"] = id
	var view st.GUIResponse
	if err := tool("gui_follow_analysis", args, &view); err != nil {
		return err
	}
	if view.Period != "L" || !view.Deformation || len(view.DisplacementAxes) != 1 || view.DisplacementAxes[0] != "Z" || filepath.Base(view.ResultPath) != "MCP result.otl" || view.Changed {
		return fmt.Errorf("unexpected IUP display: %+v", view)
	}
	if err := tool("gui_show_result", map[string]any{"connection_id": id, "path": "MCP result.otl", "displacement_axes": []string{}}, &view); err != nil {
		return err
	}
	if len(view.DisplacementAxes) != 0 || !view.Deformation {
		return fmt.Errorf("failed to disable captions: %+v", view)
	}
	if err := tool("gui_show_result", map[string]any{"connection_id": id, "path": "MCP result.otl"}, nil); err != nil {
		return err
	}
	if err := tool("start_static_analysis", map[string]any{"period": "L", "result_path": "MCP result.otl"}, nil); err == nil || !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("expected an error for the existing output file; got %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "broken.otl"), []byte("broken"), 0600); err != nil {
		return err
	}
	if err := tool("gui_show_result", map[string]any{"connection_id": id, "path": "broken.otl"}, nil); err == nil {
		return fmt.Errorf("IUP accepted a malformed .otl")
	}
	if err := tool("gui_status", map[string]any{"connection_id": id}, &view); err != nil {
		return err
	}
	if filepath.Base(view.ResultPath) != "MCP result.otl" || !view.Deformation || len(view.DisplacementAxes) != 1 {
		return fmt.Errorf("a malformed result changed the IUP display: %+v", view)
	}
	if err := in.Close(); err != nil {
		return err
	}
	if _, err := io.Copy(io.Discard, r); err != nil {
		return err
	}
	f, err := os.Stat(filepath.Join(root, "MCP result.otl"))
	if err != nil {
		return err
	}
	if f.Size() == 0 {
		return fmt.Errorf("saved .otl is empty")
	}
	return nil
}
