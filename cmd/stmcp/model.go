package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	st "github.com/yofu/st/stlib"
	"golang.org/x/text/encoding/japanese"
)

type nodeInfo struct {
	ID          int       `json:"id"`
	Coordinates []float64 `json:"coordinates"`
	Fixed       []bool    `json:"fixed"`
	Load        []float64 `json:"load"`
}

type elemInfo struct {
	ID      int    `json:"id"`
	Type    string `json:"type"`
	Section int    `json:"section"`
	Nodes   []int  `json:"nodes"`
}

type edit = st.GUIEdit

type proposal struct {
	ID      uint64   `json:"proposal_id"`
	Changes []change `json:"changes"`
	data    []byte
	lst     []byte
	edits   []edit
}

type change struct {
	Before any `json:"before"`
	After  any `json:"after"`
}

type modelSession struct {
	root              *os.Root
	rootPath          string
	parsePath         string
	frame             *st.Frame
	source            string
	raw               []byte
	lst               []byte
	lstPath           string
	pending           *proposal
	next              uint64
	analysis          *analysisJob
	nextJob           uint64
	fig2              *fig2Draft
	nextFig2          uint64
	drawingDraft      *dxfDrawingDraft
	settingsDraft     *drawingSettingsDraft
	nextSettings      uint64
	nextDrawing       uint64
	guiID             string
	pairedGUI         string // Explicitly paired window, independent of the imported model.
	workflow          *modelWorkflow
	workflows         map[uint64]*modelWorkflow
	requests          map[string]workflowKey
	guiNodes          []int
	guiElems          []int
	guiCommandVersion uint64
	guiRevision       string
	guiDelta          bool
	guiCommands       map[string][]string
}

func readModel(data []byte, path string) (frame *st.Frame, err error) {
	// Some malformed records cause the interactive INP parser to panic.
	defer func() {
		if value := recover(); value != nil {
			frame, err = nil, fmt.Errorf("invalid INP record: %v", value)
		}
	}()
	frame = st.NewFrame()
	err = frame.ReadInpSnapshot(data, path)
	return
}

func inpPath(path string) (string, error) {
	if !filepath.IsLocal(path) || !strings.EqualFold(filepath.Ext(path), ".inp") {
		return "", fmt.Errorf("expected a relative .inp path within the server root")
	}
	return filepath.Clean(path), nil
}

func (s *modelSession) open(path string) (any, error) { return s.openPair(path, "") }

func (s *modelSession) openPair(path, lstPath string) (any, error) {
	if err := s.requireRoot(); err != nil {
		return nil, err
	}
	path, err := inpPath(path)
	if err != nil {
		return nil, err
	}
	f, err := s.root.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("expected a regular INP file")
	}
	const maxBytes = 64 << 20
	data, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxBytes {
		return nil, fmt.Errorf("INP exceeds %d bytes", maxBytes)
	}
	frame, err := readModel(data, s.parsePath)
	if err != nil {
		return nil, err
	}
	if len(frame.Nodes) == 0 {
		return nil, fmt.Errorf("INP contains no nodes")
	}
	explicit := lstPath != ""
	if lstPath == "" {
		lstPath = st.Ce(path, ".lst")
	}
	lstPath = filepath.Clean(lstPath)
	lst, err := s.readLstInput(lstPath)
	if err != nil && (explicit || !os.IsNotExist(err)) {
		return nil, err
	}
	if lst != nil {
		if err = readLst(frame, lst, lstPath); err != nil {
			return nil, err
		}
	}
	s.frame, s.source, s.raw, s.pending = frame, path, data, nil
	s.guiID, s.guiNodes, s.guiElems = "", nil, nil
	s.guiDelta, s.guiRevision = false, ""
	s.lst, s.lstPath = lst, lstPath
	result := map[string]any{"path": path, "nodes": len(frame.Nodes), "elements": len(frame.Elems), "sections": len(frame.Sects)}
	if lst != nil {
		result["lst_path"] = lstPath
	}
	return result, nil
}

func nodeView(n *st.Node) nodeInfo {
	return nodeInfo{n.Num, n.Coord, n.Conf, n.Load}
}

func elementView(e *st.Elem) elemInfo {
	v := elemInfo{ID: e.Num, Type: st.ETYPES[e.Etype], Section: e.Sect.Num}
	for _, n := range e.Enod {
		v.Nodes = append(v.Nodes, n.Num)
	}
	return v
}

type idQuery struct {
	IDs   []int `json:"ids"`
	After int   `json:"after"`
	Limit int   `json:"limit"`
}

func query[T any, V any](items map[int]T, q idQuery, view func(T) V) (any, error) {
	if q.Limit < 0 || q.Limit > 200 || len(q.IDs) > 200 {
		return nil, fmt.Errorf("limit and IDs must not exceed 200")
	}
	limit := q.Limit
	if limit == 0 {
		limit = 50
	}
	ids := q.IDs
	if len(ids) == 0 {
		for id := range items {
			if id > q.After {
				ids = append(ids, id)
			}
		}
		slices.Sort(ids)
	}
	var next *int
	if len(q.IDs) == 0 && len(ids) > limit {
		ids = ids[:limit]
		next = &ids[limit-1]
	}
	result := make([]V, 0, len(ids))
	for _, id := range ids {
		item, ok := items[id]
		if !ok {
			return nil, fmt.Errorf("ID %d does not exist", id)
		}
		result = append(result, view(item))
	}
	return map[string]any{"items": result, "next_after": next}, nil
}

func (s *modelSession) preview(edits []edit) (*proposal, error) {
	if err := s.frame.ValidateGUIEdits(edits); err != nil {
		return nil, err
	}
	replacements := make(map[recordKey]map[string][]string)
	result := &proposal{edits: edits}
	for _, edit := range edits {
		fields := make(map[string][]string)
		key := recordKey{"NODE", edit.Node}
		if edit.Element > 0 {
			key = recordKey{"ELEM", edit.Element}
			member := s.frame.Elems[edit.Element]
			before := elementView(member)
			after := before
			after.Section = *edit.Section
			result.Changes = append(result.Changes, change{before, after})
			fields["ESECT"] = []string{strconv.Itoa(after.Section)}
		} else {
			node := s.frame.Nodes[edit.Node]
			before := nodeView(node)
			// The live delta path updates the imported node in place after review.
			before.Coordinates = slices.Clone(before.Coordinates)
			before.Fixed = slices.Clone(before.Fixed)
			before.Load = slices.Clone(before.Load)
			after := before
			if edit.Fixed != nil {
				after.Fixed = edit.Fixed
				fields["ICON"] = make([]string, 6)
				for j, fixed := range edit.Fixed {
					fields["ICON"][j] = "0"
					if fixed {
						fields["ICON"][j] = "1"
					}
				}
			}
			if edit.Load != nil {
				after.Load = edit.Load
				fields["VCON"] = make([]string, 6)
				for j, load := range edit.Load {
					fields["VCON"][j] = strconv.FormatFloat(load, 'g', -1, 64)
				}
			}
			result.Changes = append(result.Changes, change{before, after})
		}
		replacements[key] = fields
	}
	data, err := patchINP(s.raw, replacements)
	if err != nil {
		return nil, err
	}
	// Imported GUI snapshots are canonical and were checked on import. Disk inputs
	// may contain duplicate or remapped IDs: verify the exact edited bytes.
	if !s.guiDelta {
		parsed, err := readModel(data, s.parsePath)
		if err != nil {
			return nil, fmt.Errorf("read edited INP: %w", err)
		}
		for _, c := range result.Changes {
			switch after := c.After.(type) {
			case nodeInfo:
				n := parsed.Nodes[after.ID]
				if n == nil || !slices.Equal(n.Conf, after.Fixed) || !slices.Equal(n.Load, after.Load) {
					return nil, fmt.Errorf("edited node %d did not round-trip", after.ID)
				}
			case elemInfo:
				e := parsed.Elems[after.ID]
				if e == nil || e.Sect.Num != after.Section {
					return nil, fmt.Errorf("edited element %d did not round-trip", after.ID)
				}
			}
		}
	}
	s.next++
	result.ID, result.data = s.next, data
	s.pending = result
	return result, nil
}

func (s *modelSession) save(id uint64, path string) (any, error) {
	return s.saveFiles(id, path, false)
}

type recordKey struct {
	kind string
	id   int
}
type token struct {
	start, end int
	value      string
}
type replacement struct {
	start, end int
	text       string
}

// line starts at the next nonblank byte. Only tokenize full records for the
// IDs being changed; large INPs have thousands of unrelated records.
func splitINPField(line []byte) (word, rest []byte) {
	i := bytes.IndexAny(line, " \t")
	if i < 0 {
		return line, nil
	}
	return line[:i], bytes.TrimLeft(line[i:], " \t")
}

// Preserve all source bytes outside edited tokens, including comments, numeric
// precision, unused definitions and records the parser does not serialize.
func patchINP(data []byte, edits map[recordKey]map[string][]string) ([]byte, error) {
	var patches []replacement
	seen := make(map[recordKey]bool)
	var key recordKey
	var record []token
	flush := func() error {
		fields := edits[key]
		if fields == nil {
			return nil
		}
		if seen[key] {
			return fmt.Errorf("ambiguous duplicate %s %d", key.kind, key.id)
		}
		seen[key] = true
		for _, name := range []string{"ICON", "VCON", "ESECT"} {
			values := fields[name]
			if values == nil {
				continue
			}
			index := slices.IndexFunc(record, func(t token) bool { return t.value == name })
			if index < 0 {
				if name == "ESECT" {
					return fmt.Errorf("missing ESECT on element %d", key.id)
				}
				end := record[1].end
				patches = append(patches, replacement{end, end, " " + name + " " + strings.Join(values, " ")})
				continue
			}
			if len(record) <= index+len(values) {
				return fmt.Errorf("truncated %s on %s %d", name, key.kind, key.id)
			}
			for j, value := range values {
				t := record[index+1+j]
				patches = append(patches, replacement{t.start, t.end, value})
			}
		}
		return nil
	}
	for offset := 0; offset < len(data); {
		n := bytes.IndexByte(data[offset:], '\n')
		if n < 0 {
			n = len(data) - offset
		}
		line := bytes.TrimRight(data[offset:offset+n], "\r")
		start := offset
		offset += n + 1
		trimmed := bytes.TrimLeft(line, " \t")
		start += len(line) - len(trimmed)
		line = trimmed
		head, tail := splitINPField(line)
		if len(head) == 0 || head[0] == '#' || head[0] == '"' {
			continue
		}
		switch string(head) {
		case "NODE", "ELEM", "PROP", "SECT", "PILE", "BOND", "CHAIN", "NODESET", "ELEMSET", "}":
			if err := flush(); err != nil {
				return nil, err
			}
			key, record = recordKey{}, nil
			if string(head) == "NODE" || string(head) == "ELEM" {
				idWord, _ := splitINPField(tail)
				id, _ := strconv.Atoi(string(idWord))
				key = recordKey{string(head), id}
			}
		}
		if edits[key] == nil {
			continue
		}
		for len(line) > 0 {
			word, rest := splitINPField(line)
			if len(word) == 0 {
				break
			}
			record = append(record, token{start, start + len(word), string(word)})
			start += len(line) - len(rest)
			line = rest
		}
	}
	if err := flush(); err != nil {
		return nil, err
	}
	for key := range edits {
		if !seen[key] {
			return nil, fmt.Errorf("no source record for %s %d", key.kind, key.id)
		}
	}
	slices.SortStableFunc(patches, func(a, b replacement) int { return a.start - b.start })
	var out bytes.Buffer
	offset := 0
	for _, patch := range patches {
		out.Write(data[offset:patch.start])
		out.WriteString(patch.text)
		offset = patch.end
	}
	out.Write(data[offset:])
	return out.Bytes(), nil
}

func (s *modelSession) call(name string, raw json.RawMessage) (any, error) {
	switch name {
	case "run_model_workflow":
		return s.startWorkflow(raw)
	case "model_workflow_status", "cancel_model_workflow", "model_workflow_input":
		return s.workflowControl(name, raw)
	}
	if s.workflow != nil {
		select {
		case <-s.workflow.done:
		default:
			return nil, fmt.Errorf("workflow %d is active; use model_workflow_status or cancel_model_workflow", s.workflow.id)
		}
	}
	if name == "query_dxf_settings" || name == "preview_dxf_settings" || name == "save_dxf_settings" {
		return s.drawingSettingsTool(name, raw)
	}
	if name == "query_dxf_sections" || name == "preview_dxf_drawings" || name == "save_dxf_drawings" {
		return s.dxfDrawingTool(name, raw)
	}
	switch name {
	case "list_fig2", "read_fig2", "save_fig2_as", "save_model_as", "save_lst_as", "overwrite_model", "start_static_analysis", "start_time_history", "gui_show_result":
		if err := s.requireRoot(); err != nil {
			return nil, err
		}
	}

	switch name {
	case "list_fig2", "read_fig2", "preview_fig2", "save_fig2_as":
		return s.fig2Tool(name, raw)
	case "list_model_editors":
		var a struct {
			Search     *string `json:"search"`
			Connection string  `json:"connection_id"`
		}
		if err := decode(raw, &a); err != nil {
			return nil, err
		}
		if a.Search != nil {
			results, err := searchModelCommands(*a.Search)
			if err != nil {
				return nil, err
			}
			id := a.Connection
			if id == "" {
				id = s.pairedGUI
			}
			if id != "" {
				matches, err := s.searchGUICommands(id, *a.Search)
				if err != nil {
					return nil, err
				}
				results["gui_commands"] = matches
			}
			return results, nil
		}
		return listModelEditors(), nil
	case "gui_status", "gui_select", "gui_follow_analysis", "gui_show_result", "gui_get_model", "gui_display", "gui_undo", "gui_redo", "gui_execute_commands", "gui_command_status":
		return s.guiTool(name, raw)
	case "start_analysis_comparison", "start_static_analysis", "analysis_status", "cancel_analysis", "analysis_results":
		return s.analysisTool(name, raw)
	}
	if name == "open_model" {
		var args struct {
			Path         string `json:"path"`
			LstPath      string `json:"lst_path"`
			ConnectionID string `json:"connection_id"`
		}
		if err := decode(raw, &args); err != nil {
			return nil, err
		}
		if args.ConnectionID != "" {
			return s.openInGUI(args.Path, args.LstPath, args.ConnectionID)
		}
		return s.openPair(args.Path, args.LstPath)
	}
	if s.frame == nil {
		return nil, fmt.Errorf("open an INP model first")
	}
	switch name {
	case "query_nodes", "query_elements", "query_sections":
		var args idQuery
		if err := decode(raw, &args); err != nil {
			return nil, err
		}
		switch name {
		case "query_nodes":
			return query(s.frame.Nodes, args, nodeView)
		case "query_elements":
			return query(s.frame.Elems, args, elementView)
		default:
			return query(s.frame.Sects, args, func(sec *st.Sect) map[string]any {
				name := sec.Name
				if !utf8.ValidString(name) {
					if decoded, err := japanese.ShiftJIS.NewDecoder().String(name); err == nil {
						name = decoded
					}
				}
				info := map[string]any{"id": sec.Num, "name": name, "line": sec.HasArea(0), "plate": sec.HasThick(0)}
				if sec.HasThick(0) {
					info["plate_loads"] = plateLoads(sec)
				}
				return info
			})
		}
	case "query_lst", "preview_lst_edits", "preview_rc_column", "save_lst_as", "overwrite_model", "gui_apply_proposal", "start_time_history":
		return s.extensionTool(name, raw)
	case "preview_gui_edits":
		var args guiEditArgs
		if err := decode(raw, &args); err != nil {
			return nil, err
		}
		return s.previewGUI(args)
	case "preview_edits":
		var args struct {
			Edits []edit `json:"edits"`
		}
		if err := decode(raw, &args); err != nil {
			return nil, err
		}
		return s.preview(args.Edits)
	case "save_model_as":
		var args struct {
			ID   uint64 `json:"proposal_id"`
			Path string `json:"path"`
		}
		if err := decode(raw, &args); err != nil {
			return nil, err
		}
		return s.save(args.ID, args.Path)
	}
	return nil, fmt.Errorf("unknown tool %s", name)
}

func (s *modelSession) searchGUICommands(id, query string) ([]string, error) {
	if s.guiCommands == nil {
		s.guiCommands = make(map[string][]string)
	}
	all, cached := s.guiCommands[id]
	if !cached {
		reply, err := exchangeGUI(id, st.GUIRequest{Action: "command_search", Search: ""})
		if err != nil {
			if reply.Error == "unknown GUI action" {
				s.guiCommands[id] = nil
				return nil, nil
			}
			return nil, err
		}
		slices.Sort(reply.Matches)
		all = slices.Compact(reply.Matches)
		s.guiCommands[id] = all
	}
	needle := strings.ToLower(query)
	var matched []string
	for _, name := range all {
		if strings.Contains(strings.ToLower(name), needle) {
			matched = append(matched, name)
		}
	}
	return matched, nil
}
