package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	st "github.com/yofu/st/stlib"
)

var extendedCatalog = []tool{
	{"save_lst_as", "Save the loaded or proposed LST definitions to a new .lst file.", json.RawMessage(`{"type":"object","required":["path"],"properties":{"path":{"type":"string"},"proposal_id":{"type":"integer","minimum":1}},"additionalProperties":false}`)},
	{"query_lst", "Read the opened model's LST CODE definitions, optionally filtered by section number or taken from the pending proposal.", json.RawMessage(`{"type":"object","properties":{"sections":{"type":"array","items":{"type":"integer","minimum":1}},"proposal_id":{"type":"integer","minimum":1}},"additionalProperties":false}`)},
	{"preview_lst_edits", "Preview replacement or addition of complete ST LST CODE blocks. Unedited blocks are preserved. A single reviewed proposal can combine INP and LST changes.", json.RawMessage(`{"type":"object","required":["edits"],"properties":{"proposal_id":{"type":"integer","minimum":1},"edits":{"type":"array","minItems":1,"maxItems":100,"items":{"type":"object","required":["section"],"properties":{"section":{"type":"integer","minimum":1},"definition":{"type":"string"},"delete":{"type":"boolean"}},"additionalProperties":false}}},"additionalProperties":false}`)},
	{"preview_rc_column", "Preview a rectangular RC column in the INP and its matching LST definition. Use an existing analysis material property, specify the concrete grade if it is not already known, and give cover, steel and hoop details. Existing section yield limits are preserved.", json.RawMessage(`{"type":"object","required":["section","width_mm","depth_mm","bars","steel","cover_mm","hoop","hoop_steel"],"properties":{"proposal_id":{"type":"integer","minimum":1},"section":{"type":"integer","minimum":1},"width_mm":{"type":"number","exclusiveMinimum":0},"depth_mm":{"type":"number","exclusiveMinimum":0},"property":{"type":"integer","minimum":1},"concrete_grade":{"type":"string"},"bars":{"type":"string","description":"Symmetric perimeter layout such as 8-D19"},"steel":{"type":"string"},"cover_mm":{"type":"number","exclusiveMinimum":0},"hoop":{"type":"string","description":"For example 2x2-D10@100"},"hoop_steel":{"type":"string"}},"additionalProperties":false}`)},
	{"overwrite_model", "Replace the opened source INP and any changed LST using the reviewed proposal, refuse externally changed source files, and return backup paths.", json.RawMessage(`{"type":"object","required":["proposal_id"],"properties":{"proposal_id":{"type":"integer","minimum":1}},"additionalProperties":false}`)},
	{"start_time_history", "Run nonlinear Newmark time history with L preload, automatic Rayleigh damping and a ground motion CSV, using the current model or a proposal. Returns an analysis job ID and the new output directory.", json.RawMessage(`{"type":"object","required":["wave","output_dir","damping_ratio"],"properties":{"proposal_id":{"type":"integer","minimum":1},"wave":{"type":"string"},"output_dir":{"type":"string"},"damping_ratio":{"type":"number","minimum":0},"rayleigh_modes":{"type":"array","minItems":2,"maxItems":2,"items":{"type":"integer","minimum":1,"maximum":100}},"dt":{"type":"number","exclusiveMinimum":0},"node_ids":{"type":"array","maxItems":200,"uniqueItems":true,"items":{"type":"integer","minimum":1}},"nlgeom":{"type":"boolean"},"preview_every":{"type":"integer","minimum":1,"maximum":10000}},"additionalProperties":false}`)},
	{"start_static_analysis", "Run one linear static analysis of the opened INP asynchronously. Save the full result at a new relative path (.otl for L, .ohx for X, .ohy for Y). After completion gui_follow_analysis can display it in IUP.", json.RawMessage(`{"type":"object","required":["result_path"],"properties":{"period":{"enum":["L","X","Y"],"default":"L"},"result_path":{"type":"string","minLength":1},"node_ids":{"type":"array","items":{"type":"integer","minimum":1},"maxItems":200,"uniqueItems":true}},"additionalProperties":false}`)},
	{"gui_show_result", "Load an existing static result from the server root into the matching open IUP model and display deformation and displacement captions. The default axis is Z; an empty array disables captions.", json.RawMessage(`{"type":"object","required":["connection_id","path"],"properties":{"connection_id":{"type":"string"},"path":{"type":"string","minLength":1},"displacement_axes":{"type":"array","items":{"enum":["X","Y","Z"]},"maxItems":3,"uniqueItems":true}},"additionalProperties":false}`)},
	{"gui_status", "Inspect the active model and selection in the paired IUP instance. Obtain a connection ID in ST using :mcpconnect.", json.RawMessage(guiConnectionSchema)},
	{"gui_select", "Replace specified selections in the paired IUP instance; an empty array clears that selection.", json.RawMessage(`{"type":"object","required":["connection_id"],"properties":{"connection_id":{"type":"string"},"node_ids":{"type":"array","items":{"type":"integer","minimum":1},"uniqueItems":true},"element_ids":{"type":"array","items":{"type":"integer","minimum":1},"uniqueItems":true}},"additionalProperties":false}`)},
	{"gui_apply_proposal", "Apply the reviewed proposal to the active IUP instance and redraw. The active model must match the baseline and have no unsaved edits.", json.RawMessage(`{"type":"object","required":["connection_id","proposal_id"],"properties":{"connection_id":{"type":"string"},"proposal_id":{"type":"integer","minimum":1}},"additionalProperties":false}`)},
	{"gui_follow_analysis", "Display a completed saved static result or follow an incremental live preview in the paired IUP instance. Static results default to Z displacement captions.", json.RawMessage(`{"type":"object","required":["connection_id","job_id"],"properties":{"connection_id":{"type":"string"},"job_id":{"type":"integer","minimum":1},"displacement_axes":{"type":"array","items":{"enum":["X","Y","Z"]},"maxItems":3,"uniqueItems":true}},"additionalProperties":false}`)},
}

const guiConnectionSchema = `{"type":"object","required":["connection_id"],"properties":{"connection_id":{"type":"string"}},"additionalProperties":false}`

func init() {
	catalog = append(catalog, extendedCatalog...)
	for i, t := range catalog {
		if t.Name == "open_model" {
			catalog[i].Description += " Supply connection_id to open the same model in the paired IUP window."
			catalog[i].Input = json.RawMessage(`{"type":"object","required":["path"],"properties":{"path":{"type":"string"},"lst_path":{"type":"string"},"connection_id":{"type":"string"}},"additionalProperties":false}`)
		}
	}
}

func (s *modelSession) extensionTool(name string, raw json.RawMessage) (any, error) {
	switch name {
	case "query_lst", "preview_lst_edits", "preview_rc_column", "save_lst_as":
		return s.lstTool(name, raw)
	case "overwrite_model":
		var a struct {
			Proposal uint64 `json:"proposal_id"`
		}
		if err := decode(raw, &a); err != nil {
			return nil, err
		}
		return s.saveFiles(a.Proposal, "", true)
	case "start_time_history":
		var a timeHistoryRequest
		if err := decode(raw, &a); err != nil {
			return nil, err
		}
		if a.Motion != nil {
			return nil, fmt.Errorf("use wave to supply a ground motion CSV")
		}
		return s.startTimeHistory(a)
	}
	return s.guiTool(name, raw)
}

func (s *modelSession) guiTool(name string, raw json.RawMessage) (any, error) {
	var a struct {
		ID       string   `json:"connection_id"`
		Proposal uint64   `json:"proposal_id"`
		Job      uint64   `json:"job_id"`
		Nodes    *[]int   `json:"node_ids"`
		Elems    *[]int   `json:"element_ids"`
		Path     string   `json:"path"`
		Axes     []string `json:"displacement_axes"`
	}
	if err := decode(raw, &a); err != nil {
		return nil, err
	}
	var q st.GUIRequest
	switch name {
	case "gui_status":
		q.Action = "status"
	case "gui_select":
		if a.Nodes == nil && a.Elems == nil {
			return nil, fmt.Errorf("provide node_ids or element_ids")
		}
		q.Action, q.NodeIDs, q.ElementIDs = "select", a.Nodes, a.Elems
	case "gui_show_result":
		if s.frame == nil {
			return nil, fmt.Errorf("open an INP model first")
		}
		path, err := filepath.Abs(filepath.Join(s.root.Name(), s.source))
		if err != nil {
			return nil, err
		}
		q, err = s.staticGUIRequest(a.Path, path, s.frame.GUIModelDigest(), a.Axes)
		if err != nil {
			return nil, err
		}
	case "gui_follow_analysis":
		j := s.analysis
		if j == nil || a.Job != j.status.JobID {
			return nil, fmt.Errorf("unknown analysis job %d", a.Job)
		}
		j.mu.Lock()
		status := j.snapshot()
		output := j.staticOutput
		j.mu.Unlock()
		switch {
		case output != nil:
			if status.State != "completed" {
				return nil, fmt.Errorf("static job %d is %s; display it after completion", a.Job, status.State)
			}
			var err error
			q, err = s.staticGUIRequest(output.path, output.modelPath, output.modelDigest, a.Axes)
			if err != nil {
				return nil, err
			}
			if sha256.Sum256(q.ResultData) != output.resultDigest {
				return nil, fmt.Errorf("the saved result was changed since job %d completed", a.Job)
			}
		case status.PreviewDir != "":
			if a.Axes != nil {
				return nil, fmt.Errorf("displacement_axes applies to saved static results")
			}
			q.Action, q.PreviewDir = "follow", status.PreviewDir
		default:
			return nil, fmt.Errorf("job has no saved static result or IUP live preview; use start_static_analysis to save a static result")
		}
	case "gui_apply_proposal":
		if s.pending == nil || s.pending.ID != a.Proposal {
			return nil, fmt.Errorf("provide the current proposal_id")
		}
		if err := s.verifyOriginal(saveFile{path: s.source, original: s.raw}); err != nil {
			return nil, err
		}
		if s.lst != nil {
			if err := s.verifyOriginal(saveFile{path: s.lstPath, original: s.lst}); err != nil {
				return nil, err
			}
		}
		path, err := filepath.Abs(filepath.Join(s.root.Name(), s.source))
		if err != nil {
			return nil, err
		}
		q = st.GUIRequest{Action: "apply", BasePath: path, BaseDigest: s.frame.GUIModelDigest(), INP: s.pending.data, LST: s.proposedLst(s.pending)}
	default:
		return nil, fmt.Errorf("unknown GUI tool")
	}
	return exchangeGUI(a.ID, q)
}

func exchangeGUI(id string, q st.GUIRequest) (st.GUIResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	return st.ExchangeGUI(ctx, id, q)
}

func (s *modelSession) openInGUI(path, lstPath, connection string) (any, error) {
	// Commit the MCP model only after the paired window accepts it.
	candidate := *s
	opened, err := candidate.openPair(path, lstPath)
	if err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(filepath.Join(s.root.Name(), candidate.source))
	if err != nil {
		return nil, err
	}
	lstName := ""
	if candidate.lst != nil {
		lstName, err = filepath.Abs(filepath.Join(s.root.Name(), candidate.lstPath))
		if err != nil {
			return nil, err
		}
	}
	q := st.GUIRequest{Action: "open_model", BasePath: abs, BaseDigest: candidate.frame.GUIModelDigest(), INP: candidate.raw, LST: candidate.lst, LSTPath: lstName}
	reply, err := exchangeGUI(connection, q)
	if err != nil {
		return nil, err
	}
	*s = candidate
	result := opened.(map[string]any)
	result["gui"] = reply
	return result, nil
}
