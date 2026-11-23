package stgui

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	st "github.com/yofu/st/stlib"
	"math"
	"os"
	"os/exec"
	"time"
)

func exerciseMCPLiveGUI(ctx context.Context, exe, root, id string) error {
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
		if err := call("tools/call", map[string]any{"name": name, "arguments": args}, dest); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		return nil
	}
	if err := call("initialize", map[string]string{"protocolVersion": "2025-11-25"}, nil); err != nil {
		return err
	}
	var view st.GUIResponse
	args := map[string]any{"connection_id": id}
	if err := tool("gui_get_model", args, &view); err != nil {
		return err
	}
	if !view.Changed || len(view.Nodes) != 1 || view.Nodes[0] != 131 || len(view.Elements) != 1 || view.Elements[0] != 1001 {
		return fmt.Errorf("snapshot lost GUI selection/unsaved state: %+v", view)
	}
	var query struct {
		Items []struct {
			ID                int `json:"id"`
			Coordinates, Load []float64
		} `json:"items"`
	}
	if err := tool("query_nodes", map[string]any{"ids": []int{131}}, &query); err != nil {
		return err
	}
	if len(query.Items) != 1 || math.Abs(query.Items[0].Coordinates[0]-4.223456789) > 1e-12 {
		return fmt.Errorf("unsaved coordinate lost precision: %+v", query.Items)
	}
	originalRevision := view.Revision
	var proposal struct {
		ID int `json:"proposal_id"`
	}
	if err := tool("preview_edits", map[string]any{"edits": []any{map[string]any{"node": 131, "load": []float64{1, 2, -3, 0, 0, 0}}}}, &proposal); err != nil {
		return err
	}
	if err := tool("gui_apply_proposal", map[string]any{"connection_id": id, "proposal_id": proposal.ID}, &view); err != nil {
		return err
	}
	editedRevision := view.Revision
	if editedRevision == originalRevision || !view.Changed || len(view.Nodes) != 1 || view.Nodes[0] != 131 {
		return fmt.Errorf("edit was not applied or selection was lost: %+v", view)
	}
	if err := tool("query_nodes", map[string]any{"ids": []int{131}}, &query); err != nil {
		return err
	}
	if len(query.Items) != 1 || query.Items[0].Load[2] != -3 || math.Abs(query.Items[0].Coordinates[0]-4.223456789) > 1e-12 {
		return fmt.Errorf("MCP baseline did not follow GUI edit: %+v", query.Items)
	}
	display := map[string]any{"period": "L", "deformation": true, "deformation_scale": 75, "displacement_axes": []string{"Z"}, "stress": []any{map[string]string{"type": "COLUMN", "component": "N"}}, "angles": []float64{20, 30}, "focus": []float64{1, 2, 3}}
	if err := tool("gui_display", map[string]any{"connection_id": id, "display": display}, &view); err != nil {
		return err
	}
	if !view.Deformation || len(view.DisplacementAxes) != 1 || view.DisplacementAxes[0] != "Z" || view.Revision != editedRevision {
		return fmt.Errorf("display changed model inputs or did not update: %+v", view)
	}
	// Reject the whole display request before changing any settings.
	if err := tool("gui_display", map[string]any{"connection_id": id, "display": map[string]any{"period": "X", "angles": []float64{1}}}, nil); err == nil {
		return fmt.Errorf("malformed display request was accepted")
	}
	if err := tool("gui_status", args, &view); err != nil {
		return err
	}
	if view.Period != "L" {
		return fmt.Errorf("failed display request partially changed period")
	}
	if err := tool("gui_undo", args, &view); err != nil {
		return err
	}
	if view.Revision != originalRevision {
		return fmt.Errorf("undo did not restore unsaved GUI baseline")
	}
	if err := tool("query_nodes", map[string]any{"ids": []int{131}}, &query); err != nil {
		return err
	}
	if query.Items[0].Load[2] != 0 || math.Abs(query.Items[0].Coordinates[0]-4.223456789) > 1e-12 {
		return fmt.Errorf("undo lost unsaved coordinate")
	}
	if err := tool("gui_redo", args, &view); err != nil {
		return err
	}
	if view.Revision != editedRevision {
		return fmt.Errorf("redo did not restore MCP edit")
	}
	// A model-editing command with omitted selections uses the GUI selection.
	if err := tool("preview_gui_edits", map[string]any{"commands": []any{map[string]any{"command": "CONFPIN"}}}, &proposal); err != nil {
		return err
	}
	if proposal.ID == 0 {
		return fmt.Errorf("GUI selection was not used by the editor")
	}
	if err := tool("gui_apply_proposal", map[string]any{"connection_id": id, "proposal_id": proposal.ID}, &view); err != nil {
		return err
	}
	if err := tool("gui_undo", args, &view); err != nil {
		return err
	}
	if view.Revision != editedRevision {
		return fmt.Errorf("undo failed for second consecutive edit")
	}
	if err := tool("gui_display", map[string]any{"connection_id": id, "display": display}, nil); err != nil {
		return err
	}
	var job struct {
		ID    uint64 `json:"job_id"`
		State string `json:"state"`
	}
	if err := tool("start_static_analysis", map[string]any{"result_path": "unsaved.otl", "period": "L", "node_ids": []int{131}}, &job); err != nil {
		return err
	}
	for job.State == "running" {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(25 * time.Millisecond):
		}
		if err := tool("analysis_status", map[string]any{"job_id": job.ID}, &job); err != nil {
			return err
		}
	}
	if job.State != "completed" {
		return fmt.Errorf("unsaved-model static analysis: %s", job.State)
	}
	if err := tool("gui_follow_analysis", map[string]any{"connection_id": id, "job_id": job.ID}, &view); err != nil {
		return err
	}
	if !view.Changed || !view.Deformation {
		return fmt.Errorf("analysis display lost the unsaved edit state")
	}
	var original struct {
		Static struct {
			Nodes []struct{ Displacement [6]float64 }
		}
	}
	if err := tool("analysis_results", map[string]any{"job_id": job.ID}, &original); err != nil {
		return err
	}
	if len(original.Static.Nodes) != 1 {
		return fmt.Errorf("missing baseline static displacement")
	}
	if err := exerciseModelWorkflow(ctx, root, original.Static.Nodes[0].Displacement[2], tool); err != nil {
		return err
	}
	if err := tool("gui_display", map[string]any{"display": display}, nil); err != nil {
		return err
	}
	runCommands := func(commands []string, wantState string) (st.GUIResponse, error) {
		var response st.GUIResponse
		batch := map[string]any{"commands": commands}
		if wantState == "failed" {
			batch["wait_ms"] = 5000
		}
		if err := tool("gui_execute_commands", batch, &response); err != nil {
			return response, err
		}
		if response.CommandJob == nil {
			return response, fmt.Errorf("missing command job")
		}
		if wantState == "failed" && response.CommandJob.State == "running" {
			return response, fmt.Errorf("bounded wait did not return the failed command")
		}
		jobID := response.CommandJob.ID
		for response.CommandJob.State == "running" {
			select {
			case <-ctx.Done():
				return response, ctx.Err()
			case <-time.After(25 * time.Millisecond):
			}
			if err := tool("gui_command_status", map[string]any{"connection_id": id, "job_id": jobID}, &response); err != nil {
				return response, err
			}
		}
		if response.SyncError != "" {
			return response, fmt.Errorf("command snapshot: %s", response.SyncError)
		}
		if response.CommandJob.State != wantState {
			return response, fmt.Errorf("unexpected native command outcome: %+v", response.CommandJob)
		}
		return response, nil
	}
	// Use all three current load cases before section checking, retaining the
	// unsaved coordinate/load edits and the in-memory LST definition.
	view, err = runCommands([]string{":extractarclm %<.wgt", ":analysis -all %<.otl", ":srcal", "'srcanrate", "'srcancolor"}, "completed")
	if err != nil {
		return err
	}
	if view.RatedElements == 0 || view.SrcanRate != 15 || !view.Changed || view.CommandJob.Completed != 5 {
		return fmt.Errorf("section checking or rate diagram was not applied: %+v", view)
	}
	// Invalid commands stop a batch and report the exact completed prefix.
	failed, err := runCommands([]string{"'dfact 76", ":missing-native-command", "'dfact 999"}, "failed")
	if err != nil {
		return err
	}
	if failed.CommandJob.Completed != 1 || failed.CommandJob.Error == "" || failed.Display == nil || *failed.Display.DeformationScale != 76 {
		return fmt.Errorf("failed batch continued or hid its error: %+v", failed)
	}
	if _, err := runCommands([]string{"'dfact 75"}, "completed"); err != nil {
		return err
	}
	if err := tool("query_nodes", map[string]any{"ids": []int{131}}, &query); err != nil {
		return err
	}
	if query.Items[0].Load[2] != -3 || math.Abs(query.Items[0].Coordinates[0]-4.223456789) > 1e-12 {
		return fmt.Errorf("native commands lost unsaved model inputs")
	}
	if info, err := os.Stat(root + "/model.rat"); err != nil || info.Size() == 0 {
		return fmt.Errorf("section checking did not write rat: %v", err)
	}
	return exerciseWorkflowHTTP(ctx, exe, root, id, false)
}
