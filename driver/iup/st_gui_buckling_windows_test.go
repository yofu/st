package stgui

import (
	"context"
	"fmt"
	st "github.com/yofu/st/stlib"
	"math"
	"os"
	"path/filepath"
	"strings"
)

func exerciseBucklingWorkflow(ctx context.Context, tool func(string, map[string]any, any) error) error {
	type receipt struct {
		ID           uint64 `json:"workflow_id"`
		State, Error string
		Job          *st.GUICommandStatus `json:"command_job"`
		GUI          *st.GUIResponse
	}
	call := func(name string, a map[string]any) (receipt, error) {
		var v receipt
		err := tool(name, a, &v)
		return v, err
	}
	until := func(r receipt, desired string) (receipt, error) {
		for r.State != desired {
			if ctx.Err() != nil {
				return r, ctx.Err()
			}
			if r.State != "running" && r.State != "waiting_for_input" {
				return r, fmt.Errorf("wanted %s: %+v, job %+v", desired, r, r.Job)
			}
			var err error
			r, err = call("model_workflow_status", map[string]any{"workflow_id": r.ID, "wait_ms": 250, "include_history": true})
			if err != nil {
				return r, err
			}
		}
		return r, nil
	}
	var search map[string][]string
	if err := tool("list_model_editors", map[string]any{"search": "testnode"}, &search); err != nil || len(search["gui_commands"]) != 1 {
		return fmt.Errorf("live alias lookup %v: %v", search, err)
	}
	// OPEN requests a path even when invoked by a PGP alias. Use the same
	// temporary INP as the native-window test and keep the existing baseline.
	rOpen, e := call("run_model_workflow", map[string]any{"commands": []string{"TESTOPEN"}, "wait_ms": 0})
	if e != nil {
		return e
	}
	rOpen, e = until(rOpen, "waiting_for_input")
	if e != nil {
		return e
	}
	path := rOpen.Job.Prompt
	if len(path.Kinds) != 1 || path.Kinds[0] != "file" {
		return fmt.Errorf("missing file prompt: %+v", path)
	}
	if _, e = call("model_workflow_input", map[string]any{"workflow_id": rOpen.ID, "input": map[string]any{"prompt_id": path.ID + 1, "kind": "file", "text": "not-used.inp"}}); e == nil {
		return fmt.Errorf("accepted stale prompt")
	}
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	rOpen, e = call("model_workflow_input", map[string]any{"workflow_id": rOpen.ID, "input": map[string]any{"prompt_id": path.ID, "kind": "file", "text": filepath.Join(dir, "model.inp")}, "wait_ms": 1000})
	if e != nil {
		return e
	}
	if _, e = until(rOpen, "completed"); e != nil {
		return e
	}
	rCancel, e := call("run_model_workflow", map[string]any{"commands": []string{"TESTNODE", "'period SHOULD_NOT_RUN"}, "wait_ms": 0})
	if e != nil {
		return e
	}
	rCancel, e = until(rCancel, "waiting_for_input")
	if e != nil {
		return e
	}
	rCancel, e = call("cancel_model_workflow", map[string]any{"workflow_id": rCancel.ID, "wait_ms": 2000})
	if e != nil || rCancel.State != "cancelled" || rCancel.Job.Completed != 0 {
		return fmt.Errorf("cancelled input: %+v %v", rCancel, e)
	}
	args := map[string]any{"request_id": "buckling-once", "commands": []string{":extractarclm %<_test.wgt", ":bclng001 -mode=1 -period=L -right=100 %<_test.otp", ":read buckling %<_test.otp", "'deformation B1"}, "readback": map[string]any{"period": "B1", "require_deformation": true, "files": []string{"%<_test.otp"}, "node_ids": []int{131}}, "wait_ms": 0}
	first, err := call("run_model_workflow", args)
	if err != nil {
		return err
	}
	again, err := call("run_model_workflow", args)
	if err != nil || again.ID != first.ID {
		return fmt.Errorf("in-flight retry %v, %v", again, err)
	}
	r, err := until(first, "completed")
	if err != nil {
		return err
	}
	if r.Job == nil || len(r.Job.Timing) != 4 || r.GUI == nil || r.GUI.Readback == nil || r.GUI.Period != "B1" || !r.GUI.Deformation {
		return fmt.Errorf("incomplete receipt: %+v", r)
	}
	count := 0
	var eigen float64
	for _, c := range r.Job.Timing {
		if c.State != "completed" {
			return fmt.Errorf("command failed: %+v", c)
		}
		if c.Analysis != nil {
			count++
			if c.Analysis.Kind != "buckling" || len(c.Analysis.Eigenvalues) != 1 {
				return fmt.Errorf("wrong analysis: %+v", c.Analysis)
			}
			eigen = c.Analysis.Eigenvalues[0]
			if c.Analysis.Path != r.GUI.Readback.Path {
				return fmt.Errorf("displayed another artifact")
			}
		}
	}
	view := r.GUI.Readback
	if len(view.Files) != 1 || view.Files[0].Bytes <= 0 || count != 1 || eigen <= 0 || view.Eigenvalue == nil || math.Abs(*view.Eigenvalue-eigen) > 5.1e-6*eigen || view.NonzeroNodes == 0 || len(view.Displacements[131]) != 6 {
		return fmt.Errorf("wrong mode: %+v, solver %.12g", view, eigen)
	}
	fmt.Fprintf(os.Stderr, "buckling once: solver %.12g, file %.12g, %d/%d nonzero nodes\n", eigen, *view.Eigenvalue, view.NonzeroNodes, view.NodesWithDisplacement)
	r, err = call("run_model_workflow", map[string]any{"commands": []string{"TESTNODE"}, "wait_ms": 0})
	if err != nil {
		return err
	}
	r, err = until(r, "waiting_for_input")
	if err != nil {
		return err
	}
	if r.Job == nil || r.Job.Prompt == nil {
		return fmt.Errorf("missing prompt")
	}
	r, err = call("model_workflow_input", map[string]any{"workflow_id": r.ID, "input": map[string]any{"prompt_id": r.Job.Prompt.ID, "kind": "node", "id": 131}, "wait_ms": 1000})
	if err != nil {
		return err
	}
	r, err = until(r, "completed")
	if err != nil {
		return err
	}
	if r.GUI == nil || len(r.GUI.Nodes) != 1 || r.GUI.Nodes[0] != 131 {
		return fmt.Errorf("alias did not select node 131: %+v", r.GUI)
	}
	multi, e := call("run_model_workflow", map[string]any{"commands": []string{"SELECTELEM", "SETEPS"}, "wait_ms": 0})
	if e != nil {
		return e
	}
	for _, input := range []map[string]any{{"kind": "element", "id": 1001}, {"kind": "text", "text": "0.0001"}} {
		multi, e = until(multi, "waiting_for_input")
		if e != nil {
			return e
		}
		input["prompt_id"] = multi.Job.Prompt.ID
		multi, e = call("model_workflow_input", map[string]any{"workflow_id": multi.ID, "input": input, "wait_ms": 1000})
		if e != nil {
			return e
		}
	}
	multi, e = until(multi, "completed")
	if e != nil {
		return e
	}
	if multi.Job.Completed != 2 || multi.GUI == nil || len(multi.GUI.Elements) != 1 || multi.GUI.Elements[0] != 1001 {
		return fmt.Errorf("input did not advance to the next command: %+v", multi)
	}
	old, err := call("model_workflow_status", map[string]any{"workflow_id": first.ID, "include_history": true})
	if err != nil || old.Job == nil || strings.Count(old.Job.History, "MCP > :bclng001") != 1 || strings.Count(old.Job.History, "EIG 1:") != 1 {
		return fmt.Errorf("lost original analysis log: %v", err)
	}
	again, err = call("run_model_workflow", args)
	if err != nil || again.ID != first.ID || again.Job == nil || again.Job.ID != old.Job.ID || again.Job.History != "" {
		return fmt.Errorf("completed retry executed again: %+v %v", again, err)
	}
	bad, err := call("run_model_workflow", map[string]any{"readback": map[string]any{"period": "1", "require_deformation": true}, "wait_ms": 2000})
	if err != nil || bad.State != "failed" || bad.Error == "" {
		return fmt.Errorf("accepted nonexistent display key 1: %+v %v", bad, err)
	}
	args["commands"] = []string{"'deformation B2"}
	if _, err := call("run_model_workflow", args); err == nil {
		return fmt.Errorf("request_id accepted changed commands")
	}
	return nil
}
