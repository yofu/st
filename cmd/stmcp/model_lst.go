package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	st "github.com/yofu/st/stlib"
	"golang.org/x/text/encoding/japanese"
)

func readLst(f *st.Frame, data []byte, name string) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("invalid LST record: %v", p)
		}
	}()
	return f.ReadLstString(string(data), name)
}

func (s *modelSession) readLstInput(path string) ([]byte, error) {
	if !filepath.IsLocal(path) || !strings.EqualFold(filepath.Ext(path), ".lst") {
		return nil, fmt.Errorf("expected a root-relative .lst path")
	}
	return s.readEditInput(path)
}

func (s *modelSession) proposedLst(p *proposal) []byte {
	if p.lst != nil {
		return p.lst
	}
	return s.lst
}

type lstEdit struct {
	Section    int    `json:"section"`
	Definition string `json:"definition,omitempty"`
	Delete     bool   `json:"delete,omitempty"`
}

type lstBlock struct {
	id   int
	text string
}

func lstBlocks(data []byte) []lstBlock {
	var result []lstBlock
	start, offset, id := 0, 0, 0
	for _, line := range strings.SplitAfter(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 1 && strings.EqualFold(fields[0], "CODE") {
			if offset > start {
				result = append(result, lstBlock{id, string(data[start:offset])})
			}
			id, _ = strconv.Atoi(fields[1])
			start = offset
		}
		offset += len(line)
	}
	if start < len(data) {
		result = append(result, lstBlock{id, string(data[start:])})
	}
	return result
}

func lstText(text string) string {
	if utf8.ValidString(text) {
		return text
	}
	decoded, err := japanese.ShiftJIS.NewDecoder().String(text)
	if err != nil {
		return text
	}
	return decoded
}

func amendLst(data []byte, in []lstEdit, frame *st.Frame) ([]byte, []change, error) {
	if len(in) == 0 || len(in) > 100 {
		return nil, nil, fmt.Errorf("specify 1 to 100 LST section edits")
	}
	edits := make(map[int]string, len(in))
	for _, e := range in {
		if e.Section <= 0 || frame.Sects[e.Section] == nil {
			return nil, nil, fmt.Errorf("unknown INP section %d", e.Section)
		}
		if _, exists := edits[e.Section]; exists {
			return nil, nil, fmt.Errorf("section %d edited more than once", e.Section)
		}
		if e.Delete {
			if strings.TrimSpace(e.Definition) != "" {
				return nil, nil, fmt.Errorf("do not supply a definition when deleting section %d", e.Section)
			}
			edits[e.Section] = ""
			continue
		}
		definition := strings.TrimSpace(e.Definition)
		blocks := lstBlocks([]byte(definition))
		if len(blocks) != 1 || blocks[0].id != e.Section {
			return nil, nil, fmt.Errorf("provide one CODE %d block per section", e.Section)
		}
		if !utf8.Valid(data) {
			var err error
			definition, err = japanese.ShiftJIS.NewEncoder().String(definition)
			if err != nil {
				return nil, nil, fmt.Errorf("section %d cannot be represented in the original LST encoding: %w", e.Section, err)
			}
		}
		edits[e.Section] = definition
	}
	var out bytes.Buffer
	var changes []change
	eol := "\n"
	if bytes.Contains(data, []byte("\r\n")) {
		eol = "\r\n"
	}
	view := func(id int, text string) any { return map[string]any{"lst_section": id, "definition": lstText(text)} }
	format := func(text string) string {
		return strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\n", eol) + eol
	}
	seen := make(map[int]bool)
	for _, block := range lstBlocks(data) {
		if seen[block.id] {
			return nil, nil, fmt.Errorf("duplicate LST CODE %d", block.id)
		}
		if replacement, ok := edits[block.id]; ok {
			seen[block.id] = true
			if replacement == "" {
				changes = append(changes, change{view(block.id, block.text), nil})
				delete(edits, block.id)
				continue
			}
			out.WriteString(format(replacement))
			changes = append(changes, change{view(block.id, block.text), view(block.id, replacement)})
			delete(edits, block.id)
		} else {
			out.WriteString(block.text)
		}
	}
	for _, e := range in {
		replacement, pending := edits[e.Section]
		if pending && e.Delete {
			return nil, nil, fmt.Errorf("no LST definition for section %d", e.Section)
		}
		if pending {
			if out.Len() > 0 && !bytes.HasSuffix(out.Bytes(), []byte("\n")) {
				out.WriteString(eol)
			}
			out.WriteString(format(replacement))
			changes = append(changes, change{nil, view(e.Section, replacement)})
		}
	}
	for _, e := range in {
		frame.Sects[e.Section].Allow = nil
	}
	if err := readLst(frame, out.Bytes(), "proposal.lst"); err != nil {
		return nil, nil, err
	}
	for _, e := range in {
		if !e.Delete && frame.Sects[e.Section].Allow == nil {
			return nil, nil, fmt.Errorf("ST could not load the LST definition for section %d", e.Section)
		}
	}
	return append([]byte{}, out.Bytes()...), changes, nil
}

type rcColumnEdit struct {
	Proposal  uint64  `json:"proposal_id"`
	Section   int     `json:"section"`
	Width     float64 `json:"width_mm"`
	Depth     float64 `json:"depth_mm"`
	Property  int     `json:"property"`
	Bars      string  `json:"bars"`
	Steel     string  `json:"steel"`
	Cover     float64 `json:"cover_mm"`
	Hoop      string  `json:"hoop"`
	HoopSteel string  `json:"hoop_steel"`
	Grade     string  `json:"concrete_grade"`
}

func (s *modelSession) editBase(id uint64) ([]byte, []byte, error) {
	if id == 0 {
		return s.raw, s.lst, nil
	}
	if s.pending == nil || id != s.pending.ID {
		return nil, nil, fmt.Errorf("unknown pending proposal %d", id)
	}
	return s.pending.data, s.proposedLst(s.pending), nil
}

func (s *modelSession) pairProposal(inp, lst []byte, changes []change) (any, error) {
	s.next++
	s.pending = &proposal{ID: s.next, Changes: changes, data: inp, lst: lst}
	return s.pending, nil
}

func (s *modelSession) previewRCColumn(o rcColumnEdit) (any, error) {
	inp, lst, err := s.editBase(o.Proposal)
	if err != nil {
		return nil, err
	}
	f, err := readModel(inp, s.parsePath)
	if err != nil {
		return nil, err
	}
	sec := f.Sects[o.Section]
	if sec == nil || len(sec.Figs) != 1 {
		return nil, fmt.Errorf("select an existing single-figure section")
	}
	for _, el := range f.Elems {
		if el.Sect == sec && el.Etype != st.COLUMN {
			return nil, fmt.Errorf("section %d is also used by non-column element %d", o.Section, el.Num)
		}
	}
	fig := sec.Figs[0]
	prop := fig.Prop
	if o.Property != 0 {
		prop = f.Props[o.Property]
	}
	if prop == nil {
		return nil, fmt.Errorf("unknown material property")
	}
	var previous *st.RCColumn
	if lst != nil {
		if err := readLst(f, lst, "proposal.lst"); err != nil {
			return nil, err
		}
		previous, _ = sec.Allow.(*st.RCColumn)
	}
	grade := strings.ToUpper(o.Grade)
	if grade == "" {
		if concrete, ok := prop.Material.(st.Concrete); ok {
			grade = concrete.Name()
		} else if previous != nil {
			grade = previous.Concrete.Name()
		}
	}
	if !slices.Contains([]string{"FC18", "FC21", "FC24", "FC27", "FC30", "FC36"}, grade) {
		return nil, fmt.Errorf("provide concrete_grade (FC18, FC21, FC24, FC27, FC30 or FC36) for the LST")
	}
	if prop.Material != nil {
		concrete, ok := prop.Material.(st.Concrete)
		if !ok {
			return nil, fmt.Errorf("the selected INP property is not concrete")
		}
		if concrete.Name() != grade {
			return nil, fmt.Errorf("concrete_grade disagrees with the selected INP property")
		}
	}
	for _, d := range []float64{o.Width, o.Depth, o.Cover} {
		if !(d > 0) || math.IsInf(d, 0) {
			return nil, fmt.Errorf("width_mm, depth_mm and cover_mm must be positive")
		}
	}
	if math.Trunc(o.Width) != o.Width || math.Trunc(o.Depth) != o.Depth {
		return nil, fmt.Errorf("section dimensions must use whole millimetres")
	}
	if min(o.Width, o.Depth) <= 2*o.Cover {
		return nil, fmt.Errorf("cover is too large for the section")
	}
	countText, diam, ok := strings.Cut(strings.ToUpper(o.Bars), "-")
	count, _ := strconv.Atoi(countText)
	if !ok || count < 4 || count > 40 || count%4 != 0 || !strings.HasPrefix(diam, "D") {
		return nil, fmt.Errorf("bars must be a symmetric perimeter layout such as 8-D19; specify a multiple of four bars")
	}
	if o.Steel == "" || o.Hoop == "" || o.HoopSteel == "" {
		return nil, fmt.Errorf("steel, hoop and hoop_steel are required")
	}
	before := editorINP(f)
	fig.Prop = prop
	fig.SetShapeProperty(st.PLATE{B: o.Width / 10, H: o.Depth / 10})
	for _, key := range []string{"THICK", "KFACT", "SREIN"} {
		delete(fig.Value, key)
	}
	for _, key := range []string{"TREIN", "BREIN", "RREIN", "LREIN", "CREIN", "WREIN"} {
		delete(fig.Reins, key)
	}
	fig.Reins["KABURI"] = []string{strconv.FormatFloat(o.Cover/10, 'g', -1, 64)}
	fig.Reins["HOOP"] = []string{o.Hoop, o.HoopSteel}
	if count == 4 {
		fig.Reins["TREIN"] = []string{"2-" + diam, o.Steel}
		fig.Reins["BREIN"] = []string{"2-" + diam, o.Steel}
	} else {
		fig.Reins["CREIN"] = []string{fmt.Sprintf("%d-%s", count/4+1, diam), fmt.Sprintf("%d-%s", count/4-1, diam), o.Steel}
	}
	rc := st.NewRCColumn(o.Section)
	rect := []string{"CRECT", fmt.Sprint(-o.Width / 20), fmt.Sprint(-o.Depth / 20), fmt.Sprint(o.Width / 20), fmt.Sprint(o.Depth / 20), grade}
	if err := rc.SetConcrete(rect); err != nil {
		return nil, err
	}
	if previous != nil {
		rc.XFace, rc.YFace = previous.XFace, previous.YFace
	}
	if err := rc.AutoLayoutReins(maps.Clone(fig.Reins)); err != nil {
		return nil, err
	}
	if len(rc.Reins) != count || len(rc.Hoops.Ps) != 2 {
		return nil, fmt.Errorf("invalid reinforcement or hoop specification")
	}
	coordinates := make(map[[2]float64]bool, count)
	for _, bar := range rc.Reins {
		point := [2]float64{bar.Position[0], bar.Position[1]}
		if bar.Area <= 0 || bar.Material.Name != o.Steel || math.Abs(point[0]) > o.Width/20-o.Cover/10 || math.Abs(point[1]) > o.Depth/20-o.Cover/10 || coordinates[point] {
			return nil, fmt.Errorf("invalid bar size, material or placement")
		}
		coordinates[point] = true
	}
	if rc.Hoops.Material.Name != o.HoopSteel || slices.ContainsFunc(rc.Hoops.Ps, func(x float64) bool { return !(x > 0) || math.IsInf(x, 0) }) {
		return nil, fmt.Errorf("invalid hoop specification")
	}
	name := fmt.Sprintf("RC-%gx%g-%s", o.Width, o.Depth, o.Bars)
	sec.Name = name
	rc.SetName(name)
	after := editorINP(f)
	data := mergeEditorINP(inp, before, after)
	check, err := readModel(data, s.parsePath)
	if err != nil {
		return nil, err
	}
	if !sameModelTopology(f, check) || len(diffEditorINP(after, editorINP(check))) > 0 {
		return nil, fmt.Errorf("edited RC definition did not survive INP reload")
	}
	definition := strings.Split(rc.String(), "\n")
	for i, line := range definition {
		if strings.HasPrefix(strings.TrimSpace(line), "CRECT ") {
			definition[i] = "         " + strings.Join(rect, " ")
			break
		}
	}
	newLst, _, err := amendLst(lst, []lstEdit{{Section: o.Section, Definition: strings.Join(definition, "\n")}}, check)
	if err != nil {
		return nil, err
	}
	return s.pairProposal(data, newLst, append(diffEditorINP(editorINP(s.frame), after), diffLst(s.lst, newLst)...))
}

func (s *modelSession) lstTool(name string, raw json.RawMessage) (any, error) {
	switch name {
	case "save_lst_as":
		var a struct {
			Path     string `json:"path"`
			Proposal uint64 `json:"proposal_id"`
		}
		if err := decode(raw, &a); err != nil {
			return nil, err
		}
		_, lst, err := s.editBase(a.Proposal)
		if err != nil {
			return nil, err
		}
		if lst == nil {
			return nil, fmt.Errorf("no LST definition to save")
		}
		if !filepath.IsLocal(a.Path) || !strings.EqualFold(filepath.Ext(a.Path), ".lst") {
			return nil, fmt.Errorf("expected a root-relative .lst path")
		}
		path := filepath.Clean(a.Path)
		if err := writeNew(s.root, path, lst); err != nil {
			return nil, err
		}
		return map[string]any{"path": path}, nil
	case "query_lst":
		var a struct {
			Sections []int  `json:"sections"`
			Proposal uint64 `json:"proposal_id"`
		}
		if err := decode(raw, &a); err != nil {
			return nil, err
		}
		_, lst, err := s.editBase(a.Proposal)
		if err != nil {
			return nil, err
		}
		items := make([]lstEdit, 0)
		for _, block := range lstBlocks(lst) {
			if block.id > 0 && (len(a.Sections) == 0 || slices.Contains(a.Sections, block.id)) {
				items = append(items, lstEdit{Section: block.id, Definition: lstText(block.text)})
			}
		}
		return map[string]any{"path": s.lstPath, "sections": items}, nil
	case "preview_lst_edits":
		var a struct {
			Edits    []lstEdit `json:"edits"`
			Proposal uint64    `json:"proposal_id"`
		}
		if err := decode(raw, &a); err != nil {
			return nil, err
		}
		inp, lst, err := s.editBase(a.Proposal)
		if err != nil {
			return nil, err
		}
		f, err := readModel(inp, s.parsePath)
		if err != nil {
			return nil, err
		}
		out, _, err := amendLst(lst, a.Edits, f)
		if err != nil {
			return nil, err
		}
		return s.pairProposal(inp, out, append(diffEditorINP(editorINP(s.frame), editorINP(f)), diffLst(s.lst, out)...))
	case "preview_rc_column":
		var a rcColumnEdit
		if err := decode(raw, &a); err != nil {
			return nil, err
		}
		return s.previewRCColumn(a)
	}
	return nil, fmt.Errorf("unknown LST tool")
}

func diffLst(before, after []byte) []change {
	a, b := make(map[int]string), make(map[int]string)
	ids := make(map[int]bool)
	for _, block := range lstBlocks(before) {
		if block.id > 0 {
			a[block.id] = block.text
			ids[block.id] = true
		}
	}
	for _, block := range lstBlocks(after) {
		if block.id > 0 {
			b[block.id] = block.text
			ids[block.id] = true
		}
	}
	view := func(id int, text string) any {
		if text == "" {
			return nil
		}
		return lstEdit{Section: id, Definition: lstText(text)}
	}
	var result []change
	for _, id := range slices.Sorted(maps.Keys(ids)) {
		if a[id] != b[id] {
			result = append(result, change{view(id, a[id]), view(id, b[id])})
		}
	}
	return result
}
