package stgui

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	st "github.com/yofu/st/stlib"
)

func exerciseModelWorkflow(ctx context.Context, root string, baselineDZ float64, tool func(string, map[string]any, any) error) error {
	type receipt struct {
		ID           uint64 `json:"workflow_id"`
		State, Error string
		GUI          *st.GUIResponse
		Job          *st.GUICommandStatus `json:"command_job"`
		Static       *struct {
			Path  string `json:"result_path"`
			Nodes []struct {
				Node         int
				Displacement []float64
			}
		}
		Changes []struct{ Before, After struct{ Load []float64 } } `json:"first_changes"`
		Stages  []struct {
			Name, State string
			Start       time.Time `json:"started_at"`
			End         time.Time `json:"finished_at"`
		}
	}
	run := func(name string, args map[string]any) (receipt, error) {
		var r receipt
		err := tool(name, args, &r)
		return r, err
	}
	// One client call must change the live model, run the solver, persist the
	// output, and redraw. This session paired earlier, so no ID is supplied.
	start := time.Now()
	r, err := run("run_model_workflow", map[string]any{
		"edits":           []any{map[string]any{"node": 131, "load": []float64{1, 2, -2, 0, 0, 0}}},
		"static_analysis": map[string]any{"period": "L", "result_path": "workflow.otl", "node_ids": []int{131}},
		"display":         map[string]any{"deformation_scale": 77}, "wait_ms": 5000,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "workflow edit + static + display in one call: %v\n", time.Since(start))
	if r.State != "completed" || r.GUI == nil || !r.GUI.Deformation || r.Static == nil || r.Static.Path != "workflow.otl" || len(r.Static.Nodes) != 1 || r.Static.Nodes[0].Node != 131 || len(r.Static.Nodes[0].Displacement) != 6 || math.Abs(r.Static.Nodes[0].Displacement[2]) < 1e-8 {
		return fmt.Errorf("edit/analysis/display result: %+v (static %+v)", r, r.Static)
	}
	if r.GUI.Display == nil || *r.GUI.Display.DeformationScale != 77 || len(r.GUI.DisplacementAxes) != 1 || r.GUI.DisplacementAxes[0] != "Z" {
		return fmt.Errorf("final display: %+v", r.GUI)
	}
	if len(r.Changes) != 1 || len(r.Changes[0].Before.Load) != 6 || len(r.Changes[0].After.Load) != 6 || r.Changes[0].Before.Load[2] != -3 || r.Changes[0].After.Load[2] != -2 {
		return fmt.Errorf("incorrect before/after load: %+v", r.Changes)
	}
	dz := r.Static.Nodes[0].Displacement[2]
	if math.Abs(dz-baselineDZ) < 1e-10 {
		return fmt.Errorf("static result insensitive to the edited nodal load: %g", dz)
	}
	fmt.Fprintf(os.Stderr, "  node 131 dZ before/after %.12g / %.12g m\n", baselineDZ, dz)
	if len(r.Stages) != 5 {
		return fmt.Errorf("missing workflow stages: %+v", r.Stages)
	}
	for _, stage := range r.Stages {
		if stage.State != "completed" || stage.Start.IsZero() || stage.End.Before(stage.Start) {
			return fmt.Errorf("invalid stage timing: %+v", stage)
		}
		fmt.Fprintf(os.Stderr, "  %s %v\n", stage.Name, stage.End.Sub(stage.Start))
	}
	if info, err := os.Stat(filepath.Join(root, "workflow.otl")); err != nil || info.Size() == 0 {
		return fmt.Errorf("missing workflow result: %v", err)
	}

	r, err = run("run_model_workflow", map[string]any{"commands": []string{":extractarclm %<.wgt", ":arclm101 -init -period=L -step=2;0.0001;0;1 %<.otl", "'dfact 81"}, "wait_ms": 5000})
	if err != nil {
		return err
	}
	if r.State != "completed" || r.Job == nil || r.Job.Completed != 3 || r.Job.Step != 2 || len(r.Job.Timing) != 3 || r.GUI == nil || r.GUI.Period != "L@2" || *r.GUI.Display.DeformationScale != 81 {
		return fmt.Errorf("incremental analysis/display: %+v job %+v gui %+v", r, r.Job, r.GUI)
	}
	for i, timing := range r.Job.Timing {
		if timing.StartedAt.IsZero() || timing.FinishedAt.Before(timing.StartedAt) || (i > 0 && timing.StartedAt.Before(r.Job.Timing[i-1].FinishedAt)) {
			return fmt.Errorf("unordered command timestamps: %+v", r.Job.Timing)
		}
	}
	r, err = run("run_model_workflow", map[string]any{"commands": []string{":arclm101 -init -period=L -step=1000;0.0001;0;1 %<.otl", "'dfact 999"}, "wait_ms": 0})
	if err != nil {
		return err
	}
	if r.State != "running" || r.ID == 0 {
		return fmt.Errorf("expected async workflow: %+v", r)
	}
	if err := tool("cancel_model_workflow", map[string]any{"workflow_id": r.ID - 1}, nil); err == nil {
		return fmt.Errorf("stale ID cancelled an active workflow")
	}
	if err := tool("gui_undo", map[string]any{}, nil); err == nil {
		return fmt.Errorf("allowed an intervening edit during the active workflow")
	}
	for r.State == "running" && (r.Job == nil || r.Job.Completed != 0 || r.Job.Step < 1) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
		r, err = run("model_workflow_status", map[string]any{"workflow_id": r.ID})
		if err != nil {
			return err
		}
	}
	if r.State != "running" {
		return fmt.Errorf("incremental analysis finished before cancellation: %+v, job %+v", r, r.Job)
	}
	fmt.Fprintf(os.Stderr, "incremental workflow reached step %d/%d\n", r.Job.Step, r.Job.Steps)
	r, err = run("cancel_model_workflow", map[string]any{"workflow_id": r.ID, "wait_ms": 5000})
	if err != nil {
		return err
	}
	if r.State != "cancelled" || r.Job == nil || r.Job.State != "cancelled" || r.Job.Completed != 0 || len(r.Job.Timing) != 1 || r.GUI == nil || r.GUI.Display == nil || *r.GUI.Display.DeformationScale == 999 {
		return fmt.Errorf("cancelled workflow: %+v, job %+v", r, r.Job)
	}
	start = time.Now()
	r, err = run("run_model_workflow", map[string]any{"edits": []any{map[string]any{"node": 131, "load": []float64{1, 2, -3, 0, 0, 0}}}})
	if err != nil {
		return err
	}
	if r.State != "completed" {
		return fmt.Errorf("default wait on short edit: %+v", r)
	}
	fmt.Fprintf(os.Stderr, "workflow edit in one call: %v\n", time.Since(start))

	var selected struct {
		State       string
		ChangeCount int `json:"change_count"`
		GUI         *st.GUIResponse
	}
	if err := tool("run_model_workflow", map[string]any{"editor_commands": []any{map[string]any{"command": "CONFPIN"}}, "wait_ms": 5000}, &selected); err != nil {
		return err
	}
	if selected.State != "completed" || selected.ChangeCount < 1 || selected.GUI == nil || len(selected.GUI.Nodes) != 1 || selected.GUI.Nodes[0] != 131 {
		return fmt.Errorf("interactive edit lost selection: %+v", selected)
	}
	if err := tool("gui_undo", map[string]any{}, nil); err != nil {
		return err
	}

	r, err = run("run_model_workflow", map[string]any{"commands": []string{"'dfact 75", ":missing-native-command", "'dfact 999"}})
	if err != nil {
		return err
	}
	if r.State != "failed" || r.Error == "" || r.Job == nil || r.Job.Completed != 1 || len(r.Job.Timing) != 2 || r.GUI == nil || *r.GUI.Display.DeformationScale != 75 {
		return fmt.Errorf("failed workflow: %+v", r)
	}
	return nil
}
