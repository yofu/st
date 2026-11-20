package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	st "github.com/yofu/st/stlib"
)

func init() {
	catalog = append(catalog,
		tool{"query_dxf_settings", "Read persisted drawing settings (including style, title blocks and rotations) or generate initial candidates from INP/LST/KJN. Returns settings, source/confirmation flags, inherited member values and structured consistency issues; never edits the model.", json.RawMessage(`{"type":"object","properties":{"settings_path":{"type":"string"},"kjn_path":{"type":"string"},"proposal_id":{"type":"integer","minimum":1}},"additionalProperties":false}`)},
		tool{"preview_dxf_settings", "Review complete drawing settings before saving. Binds new member overrides to current section/node IDs; does not rebind existing identities. Returns before/after, check report and draft_id without writing files.", json.RawMessage(`{"type":"object","required":["settings"],"properties":{"settings":{"type":"object"},"settings_path":{"type":"string"},"kjn_path":{"type":"string"},"proposal_id":{"type":"integer","minimum":1}},"additionalProperties":false}`)},
		tool{"save_dxf_settings", "Save the latest reviewed settings draft at its reviewed root-relative .drawing.json path. Rejects changed settings/model/proposal/KJN; creates a backup before replacing existing manual adjustments.", json.RawMessage(`{"type":"object","required":["draft_id"],"properties":{"draft_id":{"type":"integer","minimum":1}},"additionalProperties":false}`)},
	)
}

type drawingSettingsDraft struct {
	id                                 uint64
	path, kjn, source, guiID, revision string
	proposal                           uint64
	original, data                     []byte
}

func drawingSettingsPath(path, source string) (string, error) {
	if path == "" {
		path = st.DXFDrawingSettingsPath(source)
	}
	if !filepath.IsLocal(path) || !strings.HasSuffix(strings.ToLower(path), ".drawing.json") {
		return "", fmt.Errorf("expected a root-relative .drawing.json path")
	}
	return filepath.Clean(path), nil
}
func (s *modelSession) readDrawingSettings(path string, explicit bool) (*st.DXFDrawingSettings, []byte, error) {
	data, err := s.readEditInput(path)
	if os.IsNotExist(err) && !explicit {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	cfg, err := st.DecodeDXFDrawingSettings(data)
	return cfg, data, err
}
func (s *modelSession) drawingSettingsTool(name string, raw json.RawMessage) (any, error) {
	if s.frame == nil {
		return nil, fmt.Errorf("open an INP model or call gui_get_model first")
	}
	if name == "save_dxf_settings" {
		var a struct {
			ID uint64 `json:"draft_id"`
		}
		if err := decode(raw, &a); err != nil {
			return nil, err
		}
		d := s.settingsDraft
		if d == nil || a.ID != d.id {
			return nil, fmt.Errorf("provide the latest settings draft_id")
		}
		f, err := s.drawingFrame(d.kjn, d.proposal)
		if err != nil {
			return nil, err
		}
		if d.source != s.source || d.guiID != s.guiID || d.revision != f.DXFDrawingRevision() {
			return nil, fmt.Errorf("drawing source changed; preview settings again")
		}
		file := saveFile{path: d.path, original: d.original}
		if err = s.verifyOriginal(file); err != nil {
			return nil, err
		}
		backup := ""
		if d.original == nil {
			err = writeNew(s.root, d.path, d.data)
		} else {
			var key [8]byte
			if _, err = rand.Read(key[:]); err != nil {
				return nil, err
			}
			suffix := hex.EncodeToString(key[:])
			stage := d.path + ".stage-" + suffix
			backup = d.path + ".bak-" + suffix
			if err = writeNew(s.root, stage, d.data); err != nil {
				return nil, err
			}
			defer s.root.Remove(stage)
			if err = writeNew(s.root, backup, d.original); err != nil {
				return nil, err
			}
			if err = s.verifyOriginal(file); err != nil {
				return nil, err
			}
			err = s.renameModelFile(stage, d.path)
		}
		if err != nil {
			return nil, err
		}
		s.settingsDraft = nil
		s.drawingDraft = nil
		return map[string]any{"path": d.path, "bytes": len(d.data), "backup": backup}, nil
	}
	var a struct {
		Settings json.RawMessage `json:"settings"`
		Path     string          `json:"settings_path"`
		KJN      string          `json:"kjn_path"`
		Proposal uint64          `json:"proposal_id"`
	}
	if err := decode(raw, &a); err != nil {
		return nil, err
	}
	f, err := s.drawingFrame(a.KJN, a.Proposal)
	if err != nil {
		return nil, err
	}
	path, err := drawingSettingsPath(a.Path, s.source)
	if err != nil {
		return nil, err
	}
	before, original, err := s.readDrawingSettings(path, false)
	if err != nil {
		return nil, err
	}
	cfg := before
	if name == "preview_dxf_settings" {
		s.settingsDraft = nil
		cfg, err = st.DecodeDXFDrawingSettings(a.Settings)
		if err != nil {
			return nil, err
		}
		// Populate only new bindings. Existing stale bindings remain errors.
		for id, m := range cfg.Members {
			if m.Expected == nil && f.Elems[id] != nil {
				e := f.Elems[id]
				identity := &st.DXFDrawingIdentity{Section: e.Sect.Num}
				for _, n := range e.Enod {
					identity.Nodes = append(identity.Nodes, n.Num)
				}
				m.Expected = identity
				cfg.Members[id] = m
			}
		}
	} else if len(a.Settings) > 0 {
		return nil, fmt.Errorf("query does not accept settings")
	}
	if cfg == nil {
		cfg = f.DefaultDXFDrawingSettings()
		cfg.SourceModel = filepath.Base(s.source)
	}
	report := f.CheckDXFDrawingSettings(cfg)
	result := map[string]any{"settings": cfg, "report": report, "settings_path": path, "exists": original != nil}
	if name == "preview_dxf_settings" {
		if report.HasErrors() {
			result["saved"] = false
			return result, nil
		}
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return nil, err
		}
		data = append(data, '\n')
		s.nextSettings++
		s.settingsDraft = &drawingSettingsDraft{id: s.nextSettings, path: path, kjn: a.KJN, source: s.source, guiID: s.guiID, revision: f.DXFDrawingRevision(), proposal: a.Proposal, original: bytes.Clone(original), data: data}
		result["draft_id"] = s.nextSettings
		result["before"] = before
		result["after"] = cfg
	}
	return result, nil
}
