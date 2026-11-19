package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	st "github.com/yofu/st/stlib"
)

func init() {
	catalog = append(catalog,
		tool{"gui_execute_commands", "Execute any native ex (:...) or Fig2 ('...) commands, including :srcal and 'srcanrate, in the connected IUP window, directly using its unsaved model, results, LST and selection. Commands run in order without an editing whitelist. Returns command_job.job_id immediately; poll gui_command_status until completed/failed before continuing. File operations and native dialogs behave as in IUP. :srcal uses current LST by default; use -reload to read it from disk.", json.RawMessage(guiCommandsSchema)},
		tool{"gui_command_status", "Read an IUP command job's progress, log and failure. On completion refresh the MCP baseline/selection from the GUI snapshot. A failed batch retains earlier commands' effects; inspect completed_commands and error.", json.RawMessage(guiCommandStatusSchema)},
		tool{"gui_get_model", "Import the active IUP model including unsaved edits, LST definitions and selected node/member IDs. Replaces the MCP baseline and clears its proposal. Preview edits, then gui_apply_proposal to update IUP immediately.", json.RawMessage(guiConnectionSchema)},
		tool{"gui_display", "Change IUP display settings immediately without editing the model. Stress replaces the visible stress components; empty stress or displacement_axes clears them. Angles are two view angles in degrees and focus is XYZ in metres.", json.RawMessage(guiDisplaySchema)},
		tool{"gui_undo", "Undo one edit using IUP's existing undo history, redraw, and refresh the MCP baseline and selection. An attached baseline rejects intervening model edits; gui_get_model refreshes it.", json.RawMessage(guiRevisionSchema)},
		tool{"gui_redo", "Redo one edit using IUP's undo history, redraw, and refresh the MCP baseline and selection.", json.RawMessage(guiRevisionSchema)},
	)
}

const guiRevisionSchema = `{"type":"object","required":["connection_id"],"properties":{"connection_id":{"type":"string"},"revision":{"type":"string"}},"additionalProperties":false}`
const guiDisplaySchema = `{"type":"object","required":["connection_id","display"],"properties":{"connection_id":{"type":"string"},"revision":{"type":"string"},"display":{"type":"object","properties":{"period":{"type":"string","minLength":1},"deformation":{"type":"boolean"},"deformation_scale":{"type":"number","exclusiveMinimum":0},"displacement_axes":{"type":"array","maxItems":3,"uniqueItems":true,"items":{"enum":["X","Y","Z"]}},"stress":{"type":"array","items":{"type":"object","required":["type","component"],"properties":{"type":{"enum":["COLUMN","GIRDER","BRACE","TRUSS","WBRACE","SBRACE","WALL","SLAB"]},"component":{"enum":["N","NZ","QX","QY","MZ","MX","MY"]}},"additionalProperties":false}},"angles":{"type":"array","minItems":2,"maxItems":2,"items":{"type":"number"}},"focus":{"type":"array","minItems":3,"maxItems":3,"items":{"type":"number"}}},"additionalProperties":false}},"additionalProperties":false}`

func (s *modelSession) importGUI(connection string, reply st.GUIResponse) error {
	if reply.Snapshot == nil {
		return fmt.Errorf("GUI did not return a model snapshot; rebuild st.exe")
	}
	root, err := filepath.Abs(s.modelRootPath())
	if err != nil {
		return err
	}
	source, err := filepath.Rel(root, reply.Path)
	if err != nil {
		return err
	}
	source, err = inpPath(source)
	if err != nil {
		return fmt.Errorf("GUI model must be within the MCP server root: %w", err)
	}
	f, err := readModel(reply.Snapshot.INP, reply.Path)
	if err != nil {
		return err
	}
	lstPath := st.Ce(source, ".lst")
	if reply.Snapshot.LSTPath != "" {
		relative, err := filepath.Rel(root, reply.Snapshot.LSTPath)
		if err == nil && filepath.IsLocal(relative) {
			lstPath = relative
		}
	}
	if len(reply.Snapshot.LST) > 0 {
		if err := readLst(f, reply.Snapshot.LST, lstPath); err != nil {
			return err
		}
	}
	if f.GUIRevision() != reply.Revision {
		if !bytes.Equal(f.GUILstBytes(), reply.Snapshot.LST) {
			return fmt.Errorf("GUI snapshot LST changed when parsed")
		}
		old, next := strings.Split(string(reply.Snapshot.INP), "\n"), strings.Split(string(f.GUIInpBytes()), "\n")
		for i := 0; i < min(len(old), len(next)); i++ {
			if old[i] != next[i] {
				return fmt.Errorf("GUI snapshot changed at line %d: %q -> %q", i+1, old[i], next[i])
			}
		}
		return fmt.Errorf("GUI snapshot did not retain all model values")
	}
	s.frame, s.source, s.raw, s.pending = f, source, reply.Snapshot.INP, nil
	s.lst, s.lstPath = reply.Snapshot.LST, lstPath
	if s.guiID != connection {
		s.guiCommandVersion = 0
	}
	s.guiID, s.guiNodes, s.guiElems = connection, reply.Nodes, reply.Elements
	return nil
}

const guiCommandsSchema = `{"type":"object","required":["connection_id","commands"],"properties":{"connection_id":{"type":"string"},"commands":{"type":"array","minItems":1,"maxItems":100,"items":{"type":"string","minLength":2}},"revision":{"type":"string"}},"additionalProperties":false}`
const guiCommandStatusSchema = `{"type":"object","required":["connection_id","job_id"],"properties":{"connection_id":{"type":"string"},"job_id":{"type":"integer","minimum":1}},"additionalProperties":false}`
