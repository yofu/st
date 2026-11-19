package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	st "github.com/yofu/st/stlib"
)

const dxfOptionsSchema = `{"type":"object","properties":{
"title":{"type":"string"},"font":{"type":"string"},"paper":{"enum":["A3","A4","A3P","A4P"]},"scale":{"type":"number","minimum":0,"maximum":10000},
"columns":{"type":"integer","minimum":1,"maximum":4},"rows":{"type":"integer","minimum":1,"maximum":4},
"views":{"type":"array","maxItems":100,"items":{"type":"object","required":["kind"],"properties":{"kind":{"enum":["plan","elevation"]},"floor":{"type":"integer","minimum":1},"grid":{"type":"string"},"title":{"type":"string"}},"additionalProperties":false}},
"sections":{"type":"boolean"},"section_ids":{"type":"array","maxItems":500,"items":{"type":"integer","minimum":1}},
"section_columns":{"type":"integer","minimum":1,"maximum":4},"section_rows":{"type":"integer","minimum":1,"maximum":4}
},"additionalProperties":false}`

func init() {
	catalog = append(catalog,
		tool{"query_dxf_sections", "Inspect the current model's section schedule in mm, including current unsaved LST geometry/rebar, use floors and missing-data warnings. Call gui_get_model first to import an unsaved GUI model. Optional kjn_path is root-relative; default is the source INP basename with .kjn.", json.RawMessage(`{"type":"object","properties":{"section_ids":{"type":"array","items":{"type":"integer","minimum":1},"maxItems":500},"kjn_path":{"type":"string"},"proposal_id":{"type":"integer","minimum":1}},"additionalProperties":false}`)},
		tool{"preview_dxf_drawings", "Build a reviewable DXF draft from the current model, or a reviewed proposal. Options select floor plans, named X/Y-grid elevations, RC/S section cards, paper size, automatic/fixed view scale and paginated layouts. Returns draft_id, exact placements/scales and warnings. Does not save output files or edit INP/LST.", json.RawMessage(`{"type":"object","required":["options"],"properties":{"options":` + dxfOptionsSchema + `,"kjn_path":{"type":"string"},"proposal_id":{"type":"integer","minimum":1}},"additionalProperties":false}`)},
		tool{"save_dxf_drawings", "Save the latest reviewed DXF draft to a new root-relative .dxf. Never overwrites; rejects a changed model/proposal since preview.", json.RawMessage(`{"type":"object","required":["draft_id","path"],"properties":{"draft_id":{"type":"integer","minimum":1},"path":{"type":"string"}},"additionalProperties":false}`)},
	)
}

type dxfDrawingDraft struct {
	id                        uint64
	data                      []byte
	report                    *st.DXFDocumentReport
	revision, source, guiID   string
	proposal                  uint64
	proposalData, proposalLst []byte
}

func (s *modelSession) drawingFrame(kjn string, proposalID uint64) (*st.Frame, error) {
	if s.frame == nil {
		return nil, fmt.Errorf("open an INP model or call gui_get_model first")
	}
	f := s.frame.Snapshot()
	if proposalID != 0 {
		if s.pending == nil || s.pending.ID != proposalID {
			return nil, fmt.Errorf("provide the current proposal_id")
		}
		var err error
		f, err = readModel(s.pending.data, s.parsePath)
		if err != nil {
			return nil, err
		}
		if lst := s.proposedLst(s.pending); len(lst) > 0 {
			if err = readLst(f, lst, s.lstPath); err != nil {
				return nil, err
			}
		}
	}
	f.Path = "" // No unscoped sidecar reads from the original GUI/model path.
	if kjn == "" && len(f.Kijuns) > 0 {
		return f, nil
	}
	explicit := kjn != ""
	if kjn == "" {
		kjn = st.Ce(s.source, ".kjn")
	}
	if !filepath.IsLocal(kjn) || !strings.EqualFold(filepath.Ext(kjn), ".kjn") {
		return nil, fmt.Errorf("expected a root-relative .kjn path")
	}
	if err := s.requireRoot(); err != nil {
		return nil, err
	}
	data, err := s.readEditInput(kjn)
	if os.IsNotExist(err) && !explicit {
		return f, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) > 1<<20 {
		return nil, fmt.Errorf("KJN exceeds 1 MiB")
	}
	if explicit {
		f.Kijuns = map[string]*st.Kijun{}
	}
	scanner := bufio.NewScanner(strings.NewReader(lstText(string(data))))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		if len(fields) != 7 {
			return nil, fmt.Errorf("KJN requires a name and six coordinates")
		}
		for _, word := range fields[1:] {
			value, parseErr := strconv.ParseFloat(word, 64)
			if parseErr != nil || math.IsNaN(value) || math.IsInf(value, 0) {
				return nil, fmt.Errorf("invalid KJN coordinate")
			}
		}
		if err = f.ParseKjn(fields); err != nil {
			return nil, err
		}
	}
	if err = scanner.Err(); err != nil {
		return nil, err
	}
	return f, nil
}

func (s *modelSession) dxfDrawingTool(name string, raw json.RawMessage) (any, error) {
	if s.frame == nil {
		return nil, fmt.Errorf("open an INP model or call gui_get_model first")
	}
	if name == "save_dxf_drawings" {
		var a struct {
			ID   uint64 `json:"draft_id"`
			Path string `json:"path"`
		}
		if err := decode(raw, &a); err != nil {
			return nil, err
		}
		d := s.drawingDraft
		if d == nil || d.id != a.ID {
			return nil, fmt.Errorf("provide the latest drawing draft_id")
		}
		if d.source != s.source || d.guiID != s.guiID || d.revision != s.frame.GUIRevision() {
			return nil, fmt.Errorf("model changed since drawing preview; preview again")
		}
		if d.proposal != 0 && (s.pending == nil || s.pending.ID != d.proposal || !bytes.Equal(s.pending.data, d.proposalData) || !bytes.Equal(s.proposedLst(s.pending), d.proposalLst)) {
			return nil, fmt.Errorf("proposal changed since drawing preview; preview again")
		}
		if !filepath.IsLocal(a.Path) || !strings.EqualFold(filepath.Ext(a.Path), ".dxf") {
			return nil, fmt.Errorf("expected a root-relative .dxf path")
		}
		if err := s.requireRoot(); err != nil {
			return nil, err
		}
		if err := writeNew(s.root, filepath.Clean(a.Path), d.data); err != nil {
			return nil, err
		}
		return map[string]any{"path": filepath.Clean(a.Path), "bytes": len(d.data), "report": d.report}, nil
	}
	var a struct {
		Options  st.DXFDocumentOptions `json:"options"`
		IDs      []int                 `json:"section_ids"`
		KJN      string                `json:"kjn_path"`
		Proposal uint64                `json:"proposal_id"`
	}
	if name == "query_dxf_sections" {
		var q struct {
			IDs      []int  `json:"section_ids"`
			KJN      string `json:"kjn_path"`
			Proposal uint64 `json:"proposal_id"`
		}
		if err := decode(raw, &q); err != nil {
			return nil, err
		}
		a.IDs, a.KJN, a.Proposal = q.IDs, q.KJN, q.Proposal
	} else {
		var p struct {
			Options  st.DXFDocumentOptions `json:"options"`
			KJN      string                `json:"kjn_path"`
			Proposal uint64                `json:"proposal_id"`
		}
		if err := decode(raw, &p); err != nil {
			return nil, err
		}
		a.Options, a.KJN, a.Proposal = p.Options, p.KJN, p.Proposal
	}
	if len(a.IDs) > 500 {
		return nil, fmt.Errorf("too many section IDs")
	}
	f, err := s.drawingFrame(a.KJN, a.Proposal)
	if err != nil {
		return nil, err
	}
	if name == "query_dxf_sections" {
		records, err := f.DXFSections(a.IDs)
		if err != nil {
			return nil, err
		}
		return map[string]any{"sections": records, "model_revision": s.frame.GUIRevision()}, nil
	}
	a.Options.DisableSidecar = true
	d, report, err := f.BuildDXFDocument(a.Options)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if _, err = d.WriteTo(&out); err != nil {
		return nil, err
	}
	s.nextDrawing++
	draft := &dxfDrawingDraft{id: s.nextDrawing, data: out.Bytes(), report: report, revision: s.frame.GUIRevision(), source: s.source, guiID: s.guiID, proposal: a.Proposal}
	if a.Proposal != 0 {
		draft.proposalData = append([]byte(nil), s.pending.data...)
		draft.proposalLst = append([]byte(nil), s.proposedLst(s.pending)...)
	}
	s.drawingDraft = draft
	return map[string]any{"draft_id": draft.id, "bytes": len(draft.data), "report": report, "model_revision": draft.revision, "proposal_id": a.Proposal}, nil
}
