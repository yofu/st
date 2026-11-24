package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	st "github.com/yofu/st/stlib"
)

// Reuse the public tools' argument definitions so clients see the same edits,
// display settings, and static analysis options through either entry point.
func init() {
	properties := map[string]json.RawMessage{}
	for toolName, keys := range map[string][]string{
		"gui_execute_commands": {"commands", "connection_id", "revision"},
		"gui_apply_proposal":   {"proposal_id"}, "preview_edits": {"edits"}, "gui_display": {"display"},
		"preview_gui_edits": {"commands"},
	} {
		for _, t := range catalog {
			if t.Name != toolName {
				continue
			}
			var schema struct{ Properties map[string]json.RawMessage }
			if err := json.Unmarshal(t.Input, &schema); err != nil {
				panic(err)
			}
			for _, key := range keys {
				name := key
				if toolName == "preview_gui_edits" {
					name = "editor_commands"
				}
				properties[name] = schema.Properties[key]
			}
		}
	}
	for _, t := range extendedCatalog {
		if t.Name == "start_static_analysis" {
			properties["static_analysis"] = t.Input
		}
	}
	wait := json.RawMessage(`{"type":"integer","minimum":0,"maximum":5000}`)
	properties["wait_ms"] = wait
	properties["pre_commands"] = json.RawMessage(`{"type":"array","minItems":1,"maxItems":100,"items":{"type":"string","minLength":1},"description":"Native ST command lines run before construct and edits (for example :propsect or :read list before building a grid). One command per entry."}`)
	properties["commands"] = json.RawMessage(`{"type":"array","minItems":1,"maxItems":100,"items":{"type":"string","minLength":1},"description":"Native ST command lines, including interactive IUP names and PGP aliases, run after construct/edits and before static_analysis. One command per entry."}`)
	properties["post_commands"] = json.RawMessage(`{"type":"array","minItems":1,"maxItems":100,"items":{"type":"string","minLength":1},"description":"Native ST command lines run after static_analysis and show_result, before display (for example :srcal, 'srcanrate, 'srcancolor). One command per entry."}`)
	properties["request_id"] = json.RawMessage(`{"type":"string","minLength":1,"maxLength":128,"description":"Reuse on retry in the same MCP session to get the original workflow."}`)
	properties["readback"] = json.RawMessage(`{"type":"object","properties":{"files":{"type":"array","maxItems":100,"items":{"type":"string","minLength":1}},"period":{"type":"string"},"node_ids":{"type":"array","maxItems":200,"items":{"type":"integer","minimum":1}},"require_deformation":{"type":"boolean"}},"additionalProperties":false}`)
	properties["construct"] = json.RawMessage(constructionSchema)
	properties["save_copy"] = json.RawMessage(`{"type":"string","description":"Optional new root-relative .inp filename for a copy of the live model and its LST. Leaves the active GUI model open."}`)
	input, err := json.Marshal(map[string]any{"type": "object", "properties": properties, "additionalProperties": false})
	if err != nil {
		panic(err)
	}
	control := map[string]any{"type": "object", "properties": map[string]any{"workflow_id": map[string]any{"type": "integer", "minimum": 1}, "wait_ms": wait}, "additionalProperties": false}
	control["properties"].(map[string]any)["include_history"] = map[string]any{"type": "boolean"}
	status, err := json.Marshal(control)
	if err != nil {
		panic(err)
	}
	delete(control["properties"].(map[string]any), "include_history")
	control["required"] = []string{"workflow_id"}
	cancel, err := json.Marshal(control)
	if err != nil {
		panic(err)
	}
	catalog = append(catalog,
		tool{"run_model_workflow", "Use this native MCP tool for live ST model operations instead of shell scripts. Batch pre_commands (before construct, e.g. :propsect/:read list), construct (grid, panel_grid, plate_sections, update_sections with INP+LST sync, transform_nodes for expression-based bulk 3D deformation, update_nodes_file/update_nodes for explicit coordinates), edits/editor_commands/proposal_id, commands (e.g. :bclng001/:vibeig), static_analysis (supports overwrite:true and auto-sets 1-floor boundary for L), post_commands (after static_analysis, e.g. :srcal/'srcanrate/'srcancolor), display, save_copy or readback in one call. start_analysis_comparison is only for explicit before/after comparisons. Supply connection_id on first use. Answer from the completed receipt. On running use model_workflow_status. Default wait_ms=1500; 0 returns immediately.", input},
		tool{"model_workflow_status", "Read the workflow receipt without re-executing any step or reading the GUI. Omit workflow_id to recover the most recent receipt in this MCP session. With an ID, only return that workflow. Optionally wait up to wait_ms.", status},
		tool{"model_workflow_input", "Answer the current prompt on the same workflow. Use the prompt_id and workflow_id from waiting_for_input.", json.RawMessage(workflowInputSchema)},
		tool{"cancel_model_workflow", "Cancel remaining steps of the specified workflow and request cancellation of its active solver where supported. Completed changes remain visible.", cancel},
	)
}

const workflowInputSchema = `{"type":"object","required":["workflow_id","input"],"properties":{"workflow_id":{"type":"integer","minimum":1},"wait_ms":{"type":"integer","minimum":0,"maximum":5000},"input":{"type":"object","required":["prompt_id","kind"],"properties":{"prompt_id":{"type":"integer","minimum":1},"kind":{"enum":["node","element","point","text","key","finish","file","choice"]},"id":{"type":"integer","minimum":1},"text":{"type":"string"},"point":{"type":"array","minItems":3,"maxItems":3,"items":{"type":"number"}}},"additionalProperties":false}},"additionalProperties":false}`

type workflowKey struct {
	id         uint64
	hash       [32]byte
	connection string
}

type workflowRequest struct {
	RequestID    string                 `json:"request_id"`
	Readback     *st.GUIReadback        `json:"readback"`
	SaveCopy     string                 `json:"save_copy"`
	PreCommands  []string               `json:"pre_commands"`
	Construct    *workflowConstruction  `json:"construct"`
	Connection   string                 `json:"connection_id"`
	Revision     string                 `json:"revision"`
	Edits        []edit                 `json:"edits"`
	Editors      []guiEdit              `json:"editor_commands"`
	Proposal     uint64                 `json:"proposal_id"`
	Commands     []string               `json:"commands"`
	Static       *staticAnalysisOptions `json:"static_analysis"`
	PostCommands []string               `json:"post_commands"`
	Display      *st.GUIDisplay         `json:"display"`
	Wait         *int                   `json:"wait_ms"`
}

type workflowStage struct {
	Name  string    `json:"name"`
	Start time.Time `json:"started_at"`
	End   time.Time `json:"finished_at,omitzero"`
	State string    `json:"state"`
}

type workflowResult struct {
	SavedCopy            string                  `json:"saved_copy,omitempty"`
	ID                   uint64                  `json:"workflow_id"`
	State                string                  `json:"state"`
	Error                string                  `json:"error,omitempty"`
	Received             time.Time               `json:"mcp_received_at"`
	Returned             time.Time               `json:"mcp_reply_at"`
	Stages               []workflowStage         `json:"stages,omitempty"`
	GUI                  *st.GUIResponse         `json:"gui,omitempty"`
	Commands             *st.GUICommandStatus    `json:"command_job,omitempty"`
	Construction         *st.ConstructionReceipt `json:"construction,omitempty"`
	ConstructionRevision string                  `json:"construction_revision,omitempty"`
	Analysis             *analysisStatus         `json:"analysis_job,omitempty"`
	Static               *staticAnalysisResult   `json:"static,omitempty"`
	ChangeCount          int                     `json:"change_count,omitempty"`
	Changes              []change                `json:"first_changes,omitempty"`
}

type modelWorkflow struct {
	inputReady chan struct{}
	connection string
	id         uint64
	mu         sync.Mutex
	result     workflowResult
	done       chan struct{}
	closing    chan struct{}
	cancel     context.CancelFunc
}

func workflowWait(ms *int, fallback int) (time.Duration, error) {
	if ms != nil {
		fallback = *ms
	}
	if fallback < 0 || fallback > 5000 {
		return 0, fmt.Errorf("wait_ms must be between 0 and 5000")
	}
	return time.Duration(fallback) * time.Millisecond, nil
}

func (w *modelWorkflow) receipt(wait time.Duration) workflowResult {
	return w.receiptWithHistory(wait, false)
}
func (w *modelWorkflow) receiptWithHistory(wait time.Duration, history bool) workflowResult {
	w.mu.Lock()
	ready := w.inputReady
	prompt := w.result.State == "waiting_for_input"
	w.mu.Unlock()
	if wait > 0 && !prompt {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-w.done:
		case <-ready:
		case <-timer.C:
		}
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	r := w.result
	if j := r.Commands; j != nil && !history && j.State != "failed" {
		compact := *j
		compact.History = ""
		r.Commands = &compact
	}
	r.Stages = slices.Clone(r.Stages)
	r.Returned = time.Now().UTC()
	return r
}

func (s *modelSession) startWorkflow(raw json.RawMessage) (any, error) {
	received := time.Now().UTC()
	var a workflowRequest
	if err := decode(raw, &a); err != nil {
		return nil, err
	}
	wait, err := workflowWait(a.Wait, 1500)
	if err != nil {
		return nil, err
	}
	if len(a.RequestID) > 128 || a.RequestID != "" && strings.TrimSpace(a.RequestID) == "" {
		return nil, fmt.Errorf("invalid request_id")
	}
	key, replay := s.requests[a.RequestID]
	if replay && a.Connection == "" {
		a.Connection = key.connection
	}
	if replay {
		if key.hash != a.fingerprint() {
			return nil, fmt.Errorf("request_id already used with different arguments")
		}
		if existing := s.workflows[key.id]; existing != nil {
			return existing.receipt(wait), nil
		}
		return nil, fmt.Errorf("workflow %d receipt expired; request was already executed", key.id)
	}
	if s.workflow != nil {
		select {
		case <-s.workflow.done:
		default:
			return nil, fmt.Errorf("workflow %d is active; use model_workflow_status", s.workflow.id)
		}
	}
	if a.Connection == "" {
		a.Connection = s.pairedGUI
	}
	if a.Connection == "" {
		return nil, fmt.Errorf("supply the :mcpconnect connection_id once per session")
	}
	if a.Edits != nil && (len(a.Edits) == 0 || len(a.Edits) > 100 || a.Proposal != 0) {
		return nil, fmt.Errorf("provide 1 to 100 edits or one reviewed proposal_id")
	}
	if a.Editors != nil && (len(a.Editors) == 0 || len(a.Editors) > 100 || a.Edits != nil || a.Proposal != 0) {
		return nil, fmt.Errorf("provide 1 to 100 editor_commands or a separate edits/proposal_id request")
	}
	for _, batch := range []struct {
		name string
		cmds []string
	}{{"pre_commands", a.PreCommands}, {"commands", a.Commands}, {"post_commands", a.PostCommands}} {
		if batch.cmds == nil {
			continue
		}
		if len(batch.cmds) == 0 || len(batch.cmds) > 100 {
			return nil, fmt.Errorf("provide 1 to 100 %s", batch.name)
		}
		for _, command := range batch.cmds {
			if strings.TrimSpace(command) == "" {
				return nil, fmt.Errorf("empty native command in %s", batch.name)
			}
		}
	}
	if (a.Construct != nil || len(a.PreCommands) > 0) && a.Proposal != 0 {
		return nil, fmt.Errorf("apply a reviewed proposal before constructing or running pre_commands against its result")
	}
	if a.SaveCopy != "" {
		if _, err := inpPath(a.SaveCopy); err != nil {
			return nil, err
		}
	}
	if a.Readback == nil && a.SaveCopy == "" && a.Construct == nil && len(a.PreCommands)+len(a.Edits)+len(a.Editors)+len(a.Commands)+len(a.PostCommands) == 0 && a.Proposal == 0 && a.Static == nil && a.Display == nil {
		return nil, fmt.Errorf("provide pre_commands, construct, edits, proposal_id, commands, static_analysis, post_commands or display")
	}
	if a.Proposal != 0 && (s.pending == nil || a.Proposal != s.pending.ID) {
		return nil, fmt.Errorf("provide the current proposal_id")
	}
	if s.analysis != nil {
		select {
		case <-s.analysis.done:
		default:
			return nil, fmt.Errorf("an analysis job is already running")
		}
	}
	id := uint64(1)
	if s.workflow != nil {
		id = s.workflow.id + 1
	}
	ctx, cancel := context.WithCancel(context.Background())
	w := &modelWorkflow{inputReady: make(chan struct{}), id: id, connection: a.Connection, done: make(chan struct{}), closing: make(chan struct{}), cancel: cancel, result: workflowResult{ID: id, State: "running", Received: received}}
	if s.workflows == nil {
		s.workflows = map[uint64]*modelWorkflow{}
		s.requests = map[string]workflowKey{}
	}
	if a.RequestID != "" {
		s.requests[a.RequestID] = workflowKey{id, a.fingerprint(), a.Connection}
	}
	s.workflows[id] = w
	if id > 32 {
		delete(s.workflows, id-32)
	}
	s.workflow = w
	go func() {
		defer cancel()
		err := s.executeWorkflow(ctx, w, a)
		w.mu.Lock()
		defer w.mu.Unlock()
		defer close(w.done)
		switch {
		case errors.Is(err, context.Canceled):
			w.result.State = "cancelled"
		case err != nil:
			w.result.State, w.result.Error = "failed", err.Error()
		default:
			w.result.State = "completed"
		}
	}()
	return w.receipt(wait), nil
}

func (a workflowRequest) fingerprint() [32]byte {
	a.Wait = nil
	b, err := json.Marshal(a)
	if err != nil {
		panic(err)
	}
	return sha256.Sum256(b)
}

func (s *modelSession) workflowControl(name string, raw json.RawMessage) (any, error) {
	var a struct {
		Input   *st.GUICommandInput `json:"input"`
		History bool                `json:"include_history"`
		ID      *uint64             `json:"workflow_id"`
		Wait    *int                `json:"wait_ms"`
	}
	if err := decode(raw, &a); err != nil {
		return nil, err
	}
	wait, err := workflowWait(a.Wait, 0)
	if err != nil {
		return nil, err
	}
	w := s.workflow
	if a.ID == nil && name != "model_workflow_status" {
		return nil, fmt.Errorf("workflow_id is required for cancellation")
	}
	if w == nil {
		return nil, fmt.Errorf("no workflow in this MCP session")
	}
	if a.ID != nil {
		w = s.workflows[*a.ID]
		if w == nil {
			return nil, fmt.Errorf("unknown or expired workflow %d", *a.ID)
		}
	}
	if name == "model_workflow_input" {
		if a.Input == nil {
			return nil, fmt.Errorf("input required")
		}
		w.mu.Lock()
		j := w.result.Commands
		ready := w.result.State == "waiting_for_input" && j != nil
		w.mu.Unlock()
		if !ready {
			return nil, fmt.Errorf("workflow is not waiting for input")
		}
		if _, err := exchangeGUI(w.connection, st.GUIRequest{Action: "command_input", CommandJobID: j.ID, Input: a.Input}); err != nil {
			return nil, err
		}
		w.mu.Lock()
		if w.result.State == "waiting_for_input" && w.result.Commands.Prompt != nil && w.result.Commands.Prompt.ID == a.Input.Prompt {
			copy := *w.result.Commands
			copy.Prompt = nil
			copy.State = "running"
			w.result.Commands = &copy
			w.result.State = "running"
		}
		w.mu.Unlock()
	}

	if name == "cancel_model_workflow" {
		if w != s.workflow {
			return nil, fmt.Errorf("workflow %d is not the current workflow", w.id)
		}
		w.mu.Lock()
		if w.result.State == "running" || w.result.State == "waiting_for_input" {
			w.result.State = "cancelling"
			w.cancel()
		}
		w.mu.Unlock()
	}
	return w.receiptWithHistory(wait, a.History), nil
}

func (s *modelSession) closeWorkflow() {
	if w := s.workflow; w != nil {
		close(w.closing)
		w.cancel()
		<-w.done
	}
}

func (w *modelWorkflow) stage(ctx context.Context, name string, fn func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	w.mu.Lock()
	w.result.Stages = append(w.result.Stages, workflowStage{Name: name, State: "running", Start: time.Now().UTC()})
	w.mu.Unlock()
	err := fn()
	w.mu.Lock()
	stage := &w.result.Stages[len(w.result.Stages)-1]
	stage.End, stage.State = time.Now().UTC(), "completed"
	if errors.Is(err, context.Canceled) {
		stage.State = "cancelled"
	} else if err != nil {
		stage.State = "failed"
	}
	w.mu.Unlock()
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func (s *modelSession) executeWorkflow(ctx context.Context, w *modelWorkflow, a workflowRequest) error {
	// The workflow is the sole owner of this session's mutable model until done.
	// GUI revisions protect against edits by the user or other MCP sessions.
	gui := func(name string, args map[string]any) (st.GUIResponse, error) {
		args["connection_id"] = a.Connection
		raw, err := json.Marshal(args)
		if err != nil {
			return st.GUIResponse{}, err
		}
		value, err := s.guiTool(name, raw)
		if err != nil {
			return st.GUIResponse{}, err
		}
		reply := value.(st.GUIResponse)
		w.mu.Lock()
		view := reply
		if j := reply.CommandJob; j != nil {
			if w.result.State != "cancelling" {
				if j.State == "waiting_for_input" {
					if w.result.State != j.State {
						close(w.inputReady)
						w.inputReady = make(chan struct{})
					}
					w.result.State = j.State
				} else {
					w.result.State = "running"
				}
			}
			w.result.Commands = j
			view.CommandJob = nil
		}
		if reply.Construction != nil {
			w.result.Construction = reply.Construction
			view.Construction = nil
		}
		if reply.Revision != "" || reply.Path != "" {
			if view.Readback == nil && w.result.GUI != nil {
				view.Readback = w.result.GUI.Readback
			}
			w.result.GUI = &view
		}
		w.mu.Unlock()
		if reply.SyncError != "" {
			return reply, fmt.Errorf("synchronize GUI model: %s", reply.SyncError)
		}
		return reply, nil
	}
	revision := a.Revision
	runCommandBatch := func(stageName string, cmds []string, requireModel bool) error {
		w.mu.Lock()
		var priorTiming []st.GUICommandTiming
		if w.result.Commands != nil && len(w.result.Commands.Timing) > 0 {
			priorTiming = slices.Clone(w.result.Commands.Timing)
		}
		w.mu.Unlock()
		mergeTiming := func() {
			if len(priorTiming) == 0 {
				return
			}
			w.mu.Lock()
			if w.result.Commands != nil {
				merged := *w.result.Commands
				merged.Timing = append(slices.Clone(priorTiming), merged.Timing...)
				w.result.Commands = &merged
			}
			w.mu.Unlock()
		}
		return w.stage(ctx, stageName, func() error {
			started, err := gui("gui_execute_commands", map[string]any{"commands": cmds, "revision": revision})
			mergeTiming()
			if err != nil {
				return err
			}
			if started.CommandJob == nil {
				return fmt.Errorf("GUI did not return a command job")
			}
			id := started.CommandJob.ID
			ticker := time.NewTicker(15 * time.Millisecond)
			defer ticker.Stop()
			stop := ctx.Done()
			for {
				select {
				case <-w.closing:
					// A native modal operation can outlive an MCP transport. Leave
					// the GUI responsible for suppressing the remaining commands.
					_, err := exchangeGUI(a.Connection, st.GUIRequest{Action: "cancel_commands", CommandJobID: id})
					if err != nil {
						return err
					}
					return context.Canceled
				case <-stop:
					stop = nil
					if _, err := exchangeGUI(a.Connection, st.GUIRequest{Action: "cancel_commands", CommandJobID: id}); err != nil {
						return fmt.Errorf("cancel GUI job %d: %w", id, err)
					}
				case <-ticker.C:
				}
				reply, err := gui("gui_workflow_command_status", map[string]any{"job_id": id})
				mergeTiming()
				if err != nil {
					return err
				}
				job := reply.CommandJob
				if job == nil || job.ID != id {
					return fmt.Errorf("missing response for GUI command job %d", id)
				}
				switch job.State {
				case "running", "waiting_for_input":
					continue
				case "completed":
					if requireModel && reply.Revision == "" {
						return fmt.Errorf("no GUI model after the command batch")
					}
					revision = reply.Revision
					return nil
				case "cancelled":
					return context.Canceled
				default:
					return fmt.Errorf("command %q after %d completed commands: %s", job.Command, job.Completed, job.Error)
				}
			}
		})
	}
	if len(a.PreCommands) > 0 {
		if err := runCommandBatch("pre_commands", a.PreCommands, a.Construct != nil || a.Static != nil); err != nil {
			return err
		}
	}
	if a.Construct != nil {
		err := w.stage(ctx, "construct", func() error {
			reply, err := gui("gui_construct", map[string]any{"construct": a.Construct, "revision": revision})
			if err != nil {
				return err
			}
			if reply.Construction == nil || reply.Revision == "" || s.guiID != a.Connection || s.guiRevision != reply.Revision || s.frame == nil {
				return fmt.Errorf("construction may already be applied; missing a synchronized receipt")
			}
			actual := st.ConstructionCounts{Properties: len(s.frame.Props), Sections: len(s.frame.Sects), Nodes: len(s.frame.Nodes), Elements: len(s.frame.Elems)}
			if actual != reply.Construction.Total {
				return fmt.Errorf("construction may already be applied; receipt totals %+v differ from GUI model %+v", reply.Construction.Total, actual)
			}
			revision = reply.Revision
			w.mu.Lock()
			w.result.ConstructionRevision = revision
			w.mu.Unlock()
			return nil
		})
		if err != nil {
			return err
		}
	}
	if a.Construct == nil && (a.Edits != nil || a.Editors != nil || (a.Static != nil && a.Proposal == 0 && len(a.Commands) == 0)) {
		err := w.stage(ctx, "read_gui", func() error {
			reply, err := gui("gui_sync_model", map[string]any{"revision": revision})
			if err == nil {
				revision = reply.Revision
			}
			return err
		})
		if err != nil {
			return err
		}
	}
	if a.Proposal != 0 || a.Edits != nil || a.Editors != nil {
		err := w.stage(ctx, "edit", func() error {
			id := a.Proposal
			if a.Edits != nil {
				p, err := s.preview(a.Edits)
				if err != nil {
					return err
				}
				id = p.ID
			}
			if a.Editors != nil {
				v, err := s.previewGUI(guiEditArgs{Commands: a.Editors})
				if err != nil {
					return err
				}
				var changed bool
				id, changed = v.(map[string]any)["proposal_id"].(uint64)
				if !changed {
					return nil
				}
			}
			count := len(s.pending.Changes)
			changes := slices.Clone(s.pending.Changes[:min(10, count)])
			if err := ctx.Err(); err != nil {
				return err
			}
			reply, err := gui("gui_apply_proposal", map[string]any{"proposal_id": id, "revision": revision})
			if err == nil {
				w.mu.Lock()
				w.result.ChangeCount, w.result.Changes = count, changes
				w.mu.Unlock()
			}
			if err == nil {
				revision = reply.Revision
			}
			return err
		})
		if err != nil {
			return err
		}
	}
	if len(a.Commands) > 0 {
		if err := runCommandBatch("commands", a.Commands, a.Static != nil); err != nil {
			return err
		}
	}
	if a.Static != nil {
		err := w.stage(ctx, "static_analysis", func() error {
			if err := s.requireRoot(); err != nil {
				return err
			}
			if s.guiID != a.Connection {
				return fmt.Errorf("no imported model for the requested GUI")
			}
			if _, err := s.startStaticAnalysis(*a.Static); err != nil {
				return err
			}
			j := s.analysis
			tick := time.NewTicker(20 * time.Millisecond)
			defer tick.Stop()
			for {
				j.mu.Lock()
				status := j.snapshot()
				var result *staticAnalysisResult
				if j.result != nil {
					result = j.result.Static
				}
				j.mu.Unlock()
				w.mu.Lock()
				w.result.Analysis, w.result.Static = &status, result
				w.mu.Unlock()
				switch status.State {
				case "completed":
					return nil
				case "cancelled":
					return context.Canceled
				case "failed":
					return fmt.Errorf("%s", status.Error)
				}
				select {
				case <-ctx.Done():
					j.cancel()
					<-j.done
				case <-j.done:
				case <-tick.C:
				}
			}
		})
		if err != nil {
			return err
		}
		err = w.stage(ctx, "show_result", func() error {
			args := map[string]any{"job_id": s.analysis.status.JobID}
			if a.Display != nil && a.Display.DisplacementAxes != nil {
				args["displacement_axes"] = *a.Display.DisplacementAxes
			}
			reply, err := gui("gui_follow_analysis", args)
			if err == nil {
				revision = reply.Revision
			}
			return err
		})
		if err != nil {
			return err
		}
	}
	if len(a.PostCommands) > 0 {
		if err := runCommandBatch("post_commands", a.PostCommands, false); err != nil {
			return err
		}
	}
	if a.Display != nil {
		err := w.stage(ctx, "display", func() error {
			reply, err := gui("gui_display", map[string]any{"display": a.Display, "revision": revision})
			if err == nil {
				revision = reply.Revision
			}
			return err
		})
		if err != nil {
			return err
		}
	}
	if a.SaveCopy != "" {
		if err := w.stage(ctx, "save_copy", func() error {
			if _, err := gui("gui_sync_model", map[string]any{"revision": revision}); err != nil {
				return err
			}
			if err := s.saveGUICopy(a.SaveCopy); err != nil {
				return err
			}
			w.mu.Lock()
			w.result.SavedCopy = a.SaveCopy
			w.mu.Unlock()
			return nil
		}); err != nil {
			return err
		}
	}
	if a.Readback != nil {
		if err := w.stage(ctx, "readback", func() error {
			reply, err := gui("gui_readback", map[string]any{"readback": a.Readback, "revision": revision})
			if err == nil && reply.Readback == nil {
				return fmt.Errorf("GUI did not return requested results")
			}
			if err == nil {
				err = w.verifyDisplayedMode(reply.Readback)
			}
			return err
		}); err != nil {
			return err
		}
	}

	return nil
}

// A display flag can refer to a mode left over from an earlier run on the same path.
func (w *modelWorkflow) verifyDisplayedMode(view *st.GUIResultView) error {
	if view == nil || !strings.HasPrefix(view.Period, "B") {
		return nil
	}
	rank, err := strconv.Atoi(strings.TrimPrefix(view.Period, "B"))
	if err != nil || rank < 1 {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	job := w.result.Commands
	if job == nil {
		return nil
	}
	for i := len(job.Timing) - 1; i >= 0; i-- {
		a := job.Timing[i].Analysis
		if a == nil {
			continue
		}
		if rank > len(a.Eigenvalues) || view.ModeVersion <= a.BeforeRead || !strings.EqualFold(filepath.Clean(a.Path), filepath.Clean(view.Path)) || view.NonzeroNodes == 0 || view.Eigenvalue == nil {
			return fmt.Errorf("%s has not been loaded from this workflow's %s output %s", view.Period, a.Kind, a.Path)
		}
		expected := a.Eigenvalues[rank-1]
		if math.Abs(expected-*view.Eigenvalue) > 5.1e-6*math.Abs(expected) {
			return fmt.Errorf("%s file eigenvalue does not match the solver", view.Period)
		}
		return nil
	}
	return nil
}
