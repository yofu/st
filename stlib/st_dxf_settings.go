package st

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const DXFDrawingSettingsVersion = 1
const DXFDrawingSettingsLimit = 4 << 20

// Model positions/lengths are mm; text offsets are paper mm.
// Pointers distinguish explicit zero overrides from inheritance.
type DXFDrawingSettings struct {
	Style          *DXFDraftingStyle        `json:"style,omitempty"`
	Sheet          *DXFSheetTemplate        `json:"sheet,omitempty"`
	Version        int                      `json:"version"`
	SourceModel    string                   `json:"source_model,omitempty"`
	SourceRevision string                   `json:"source_revision,omitempty"`
	GroundLevelMM  *float64                 `json:"ground_level_mm,omitempty"`
	Levels         []DXFDrawingLevel        `json:"levels,omitempty"`
	Grids          []DXFDrawingGrid         `json:"grids,omitempty"`
	Sections       map[int]DXFDrawingMember `json:"sections,omitempty"`
	Members        map[int]DXFDrawingMember `json:"members,omitempty"`
	Annotations    []DXFDrawingAnnotation   `json:"annotations,omitempty"`
}
type DXFDrawingLevel struct {
	Name        string  `json:"name"`
	Floor       int     `json:"floor,omitempty"`
	ElevationMM float64 `json:"elevation_mm"`
	Source      string  `json:"source,omitempty"`
	Confirmed   bool    `json:"confirmed"`
}
type DXFDrawingGrid struct {
	Name       string  `json:"name"`
	Axis       string  `json:"axis"` // X: x=constant, Y: y=constant
	PositionMM float64 `json:"position_mm"`
	Source     string  `json:"source,omitempty"`
	Confirmed  bool    `json:"confirmed"`
}
type DXFDrawingIdentity struct {
	Section int   `json:"section"`
	Nodes   []int `json:"nodes"`
}
type DXFDrawingMember struct {
	Symbol             *string             `json:"symbol,omitempty"`
	Level              *string             `json:"level,omitempty"`
	OffsetMM           *[3]float64         `json:"offset_mm,omitempty"`     // global XYZ; geometry stage
	TopOffsetMM        *float64            `json:"top_offset_mm,omitempty"` // relative to FL; geometry stage
	LabelOffsetPaperMM *[2]float64         `json:"label_offset_paper_mm,omitempty"`
	Notes              *[]string           `json:"notes,omitempty"`
	Expected           *DXFDrawingIdentity `json:"expected,omitempty"`
}
type DXFDrawingAnnotation struct {
	ID            string      `json:"id"`
	View          string      `json:"view"` // all, plan:floor, elevation:grid
	Text          string      `json:"text"`
	Element       int         `json:"element,omitempty"`     // midpoint anchor follows model edits
	PositionMM    *[2]float64 `json:"position_mm,omitempty"` // projected XY
	OffsetPaperMM [2]float64  `json:"offset_paper_mm,omitempty"`
}

// Fixed arrays otherwise silently accept/truncate JSON arrays of the wrong
// length, which could lose a manually specified offset without a warning.
func decodeDXFDrawingObject(data []byte, target any, arrays map[string]int) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for key, size := range arrays {
		raw, ok := fields[key]
		if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			continue
		}
		var values []json.RawMessage
		if err := json.Unmarshal(raw, &values); err != nil {
			return err
		}
		if len(values) != size {
			return fmt.Errorf("%s requires exactly %d coordinates", key, size)
		}
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	return d.Decode(target)
}
func (m *DXFDrawingMember) UnmarshalJSON(data []byte) error {
	type plain DXFDrawingMember
	var v plain
	if err := decodeDXFDrawingObject(data, &v, map[string]int{"offset_mm": 3, "label_offset_paper_mm": 2}); err != nil {
		return err
	}
	*m = DXFDrawingMember(v)
	return nil
}
func (a *DXFDrawingAnnotation) UnmarshalJSON(data []byte) error {
	type plain DXFDrawingAnnotation
	var v plain
	if err := decodeDXFDrawingObject(data, &v, map[string]int{"position_mm": 2, "offset_paper_mm": 2}); err != nil {
		return err
	}
	*a = DXFDrawingAnnotation(v)
	return nil
}

type DXFDrawingIssue struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Element  int    `json:"element,omitempty"`
	Section  int    `json:"section,omitempty"`
}
type DXFResolvedDrawingMember struct {
	Element  int              `json:"element"`
	Section  int              `json:"section"`
	Settings DXFDrawingMember `json:"settings"`
}
type DXFDrawingCheckReport struct {
	Drafting      *DXFDraftingStyle          `json:"drafting,omitempty"`
	SheetTemplate *DXFSheetTemplate          `json:"sheet_template,omitempty"`
	ModelRevision string                     `json:"model_revision"`
	Issues        []DXFDrawingIssue          `json:"issues"`
	Members       []DXFResolvedDrawingMember `json:"members,omitempty"`
}

func (r *DXFDrawingCheckReport) HasErrors() bool {
	for _, i := range r.Issues {
		if i.Severity == "error" {
			return true
		}
	}
	return false
}
func DXFDrawingSettingsPath(model string) string { return Ce(model, ".drawing.json") }
func cloneDXFDrawingSettings(s *DXFDrawingSettings) *DXFDrawingSettings {
	if s == nil {
		return nil
	}
	b, _ := json.Marshal(s)
	r := DXFDrawingSettings{Sections: map[int]DXFDrawingMember{}, Members: map[int]DXFDrawingMember{}}
	_ = json.Unmarshal(b, &r)
	return &r
}
func DecodeDXFDrawingSettings(data []byte) (*DXFDrawingSettings, error) {
	if len(data) > DXFDrawingSettingsLimit {
		return nil, fmt.Errorf("drawing settings exceed 4 MiB")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	var s *DXFDrawingSettings
	if err := d.Decode(&s); err != nil {
		return nil, err
	}
	if s == nil {
		return nil, fmt.Errorf("drawing settings must be an object")
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("unexpected trailing drawing settings data")
	}
	if s.Version != DXFDrawingSettingsVersion {
		return nil, fmt.Errorf("unsupported drawing settings version %d", s.Version)
	}
	return s, nil
}
func ReadDXFDrawingSettings(path string) (*DXFDrawingSettings, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, DXFDrawingSettingsLimit+1))
	if err != nil {
		return nil, err
	}
	return DecodeDXFDrawingSettings(b)
}

// Initializing must never overwrite manually adjusted settings.
func WriteNewDXFDrawingSettings(path string, s *DXFDrawingSettings) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if _, err = DecodeDXFDrawingSettings(b); err != nil {
		return err
	}
	out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, err = out.Write(append(b, '\n'))
	closeErr := out.Close()
	if err != nil {
		return err
	}
	return closeErr
}
func (f *Frame) DXFDrawingRevision() string {
	h := sha256.New()
	// Hash drawing-relevant inputs directly. INP serialization requires full
	// analysis parameters and is unsuitable for partially constructed GUI models.
	fmt.Fprintf(h, "boundaries=%v;", f.Ai.Boundary)
	ids := make([]int, 0, len(f.Nodes))
	for id := range f.Nodes {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		n := f.Nodes[id]
		fmt.Fprintf(h, "N%d:%v;", id, n.Coord)
	}
	ids = nil
	for id := range f.Elems {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		e := f.Elems[id]
		fmt.Fprintf(h, "E%d:%d:%d:%g:", id, e.Sect.Num, e.Etype, e.Cang)
		for _, n := range e.Enod {
			fmt.Fprintf(h, "%d,", n.Num)
		}
		for _, b := range e.Bonds {
			if b == nil {
				io.WriteString(h, "rigid,")
			} else {
				fmt.Fprintf(h, "%d:%v,", b.Num, b.Stiffness)
			}
		}
	}
	ids = nil
	for id := range f.Sects {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		sec := f.Sects[id]
		fmt.Fprintf(h, "S%d:%q:", id, sec.Name)
		for _, fig := range sec.Figs {
			fmt.Fprintf(h, "%v;", fig.Value)
			if fig.Shape != nil {
				io.WriteString(h, fig.Shape.String())
			}
		}
		if sec.Allow != nil {
			data, err := json.Marshal(sec.Allow)
			fmt.Fprintf(h, "%T:%s:%s:%v", sec.Allow, sec.Allow.Name(), data, err)
		}
	}
	if records, err := f.DXFSections(nil); err == nil {
		data, _ := json.Marshal(records)
		h.Write(data)
	}
	keys := make([]string, 0, len(f.Kijuns))
	for k := range f.Kijuns {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		g := f.Kijuns[k]
		fmt.Fprintf(h, "%q:%v:%v;", g.Name, g.Start, g.End)
	}
	return hex.EncodeToString(h.Sum(nil))
}
func (f *Frame) DefaultDXFDrawingSettings() *DXFDrawingSettings {
	s := &DXFDrawingSettings{Version: 1, Style: DefaultDXFDraftingStyle(), Sheet: DefaultDXFSheetTemplate(), SourceModel: filepath.Base(f.Path), SourceRevision: f.DXFDrawingRevision(), Sections: map[int]DXFDrawingMember{}, Members: map[int]DXFDrawingMember{}}
	for _, g := range f.Kijuns {
		if strings.EqualFold(g.Name, "GL") && math.Abs(g.Start[2]-g.End[2]) < 1e-4 {
			z := g.Start[2] * 1000
			s.GroundLevelMM = &z
			break
		}
	}
	for i, l := range f.dxfLevels() {
		if strings.EqualFold(l.name, "GL") {
			continue
		}
		source, confirmed := "inferred", false
		for _, g := range f.Kijuns {
			if strings.EqualFold(g.Name, l.name) && math.Abs(g.Start[2]-l.value) < 1e-4 {
				source, confirmed = "kjn", true
				break
			}
		}
		floor := 0
		if i+1 < len(f.Ai.Boundary) {
			floor = i + 1
		}
		s.Levels = append(s.Levels, DXFDrawingLevel{Name: l.name, Floor: floor, ElevationMM: l.value * 1000, Source: source, Confirmed: confirmed})
	}
	for axis := 0; axis < 2; axis++ {
		for _, g := range f.dxfGrids(axis) {
			source, confirmed := "inferred", false
			for _, k := range f.Kijuns {
				if strings.EqualFold(k.Name, g.name) {
					source, confirmed = "kjn", true
					break
				}
			}
			s.Grids = append(s.Grids, DXFDrawingGrid{Name: g.name, Axis: []string{"X", "Y"}[axis], PositionMM: g.value * 1000, Source: source, Confirmed: confirmed})
		}
	}
	return s
}
func overlayDXFDrawingMember(a, b DXFDrawingMember) DXFDrawingMember {
	if b.Symbol != nil {
		a.Symbol = b.Symbol
	}
	if b.Level != nil {
		a.Level = b.Level
	}
	if b.OffsetMM != nil {
		a.OffsetMM = b.OffsetMM
	}
	if b.TopOffsetMM != nil {
		a.TopOffsetMM = b.TopOffsetMM
	}
	if b.LabelOffsetPaperMM != nil {
		a.LabelOffsetPaperMM = b.LabelOffsetPaperMM
	}
	if b.Notes != nil {
		a.Notes = b.Notes
	}
	if b.Expected != nil {
		a.Expected = b.Expected
	}
	return a
}
func (f *Frame) resolvedDXFMember(e *Elem) DXFDrawingMember {
	if f.drawingSettings == nil {
		return DXFDrawingMember{}
	}
	return overlayDXFDrawingMember(f.drawingSettings.Sections[e.Sect.Num], f.drawingSettings.Members[e.Num])
}
func (f *Frame) dxfMemberSymbol(e *Elem) string {
	m := f.resolvedDXFMember(e)
	if m.Symbol != nil {
		return *m.Symbol
	}
	if e.Sect.Allow != nil && e.Sect.Allow.Name() != "" {
		return e.Sect.Allow.Name()
	}
	name := strings.TrimSpace(strings.Split(e.Sect.Name, ":")[0])
	if name == "" {
		return fmt.Sprintf("S%d", e.Sect.Num)
	}
	return name
}

func (f *Frame) CheckDXFDrawingSettings(s *DXFDrawingSettings) *DXFDrawingCheckReport {
	r := &DXFDrawingCheckReport{ModelRevision: f.DXFDrawingRevision(), Issues: []DXFDrawingIssue{}}
	add := func(severity, code, msg string, e, section int) {
		r.Issues = append(r.Issues, DXFDrawingIssue{severity, code, msg, e, section})
	}
	if s == nil || s.Version != 1 {
		add("error", "settings_version", "図面設定のバージョンが不正です", 0, 0)
		return r
	}
	if resolved, err := ResolveDXFDraftingStyle(s.Style); err != nil {
		add("error", "drafting_style", err.Error(), 0, 0)
	} else {
		r.Drafting = resolved
	}
	if resolved, err := ResolveDXFSheetTemplate(s.Sheet); err != nil {
		add("error", "sheet_template", err.Error(), 0, 0)
	} else {
		r.SheetTemplate = resolved
	}
	if s.SourceRevision != "" && s.SourceRevision != r.ModelRevision {
		add("warning", "source_changed", "設定作成後にモデル・LST・通り定義が変わっています。対応を確認してください", 0, 0)
	}
	if s.GroundLevelMM == nil {
		add("warning", "ground_level_unconfirmed", "GLが未指定です。最下節点をGLと断定せず、図面基準高さを指定してください", 0, 0)
	} else if !finiteDXF(*s.GroundLevelMM) {
		add("error", "invalid_ground_level", "GLは有限値で指定してください", 0, 0)
	}
	levels := map[string]bool{"GL": true}
	floors := map[int]bool{}
	unconfirmed := false
	for _, l := range s.Levels {
		k := strings.ToUpper(l.Name)
		if strings.TrimSpace(l.Name) == "" || levels[k] || !finiteDXF(l.ElevationMM) || l.Floor < 0 || l.Floor >= len(f.Ai.Boundary) && l.Floor != 0 {
			add("error", "invalid_level", fmt.Sprintf("階定義 %q の名前・高さ・階番号が不正または重複しています", l.Name), 0, 0)
		}
		if l.Floor > 0 {
			if floors[l.Floor] {
				add("error", "duplicate_floor", fmt.Sprintf("階番号 %d のFLが重複しています", l.Floor), 0, 0)
			}
			floors[l.Floor] = true
		}
		levels[k] = true
		unconfirmed = unconfirmed || !l.Confirmed
	}
	grids := map[string]bool{}
	for _, g := range s.Grids {
		k := strings.ToUpper(g.Name)
		if strings.TrimSpace(g.Name) == "" || grids[k] || (g.Axis != "X" && g.Axis != "Y") || !finiteDXF(g.PositionMM) {
			add("error", "invalid_grid", fmt.Sprintf("通り %q の名前・軸・座標が不正または重複しています", g.Name), 0, 0)
		}
		grids[k] = true
		unconfirmed = unconfirmed || !g.Confirmed
	}
	if unconfirmed {
		add("warning", "inferred_datums", "未確認の階・通り候補があります。確認後 confirmed=true にしてください", 0, 0)
	}
	validate := func(m DXFDrawingMember, e, sec int) {
		if m.Symbol != nil && strings.TrimSpace(*m.Symbol) == "" {
			add("error", "empty_symbol", "部材符号が空です", e, sec)
		}
		if m.Level != nil && !levels[strings.ToUpper(*m.Level)] {
			add("error", "unknown_level", "部材の基準階が見つかりません: "+*m.Level, e, sec)
		}
		if m.Level != nil && strings.EqualFold(*m.Level, "GL") && s.GroundLevelMM == nil {
			add("error", "unknown_ground_level", "部材がGLを参照していますがGLが未指定です", e, sec)
		}
		if m.TopOffsetMM != nil {
			if !finiteDXF(*m.TopOffsetMM) {
				add("error", "invalid_offset", "天端差は有限値で指定してください", e, sec)
			}
			if m.Level == nil {
				add("error", "missing_reference_level", "天端差には基準階の指定が必要です", e, sec)
			}
		}
		if m.OffsetMM != nil {
			for _, v := range m.OffsetMM {
				if !finiteDXF(v) {
					add("error", "invalid_offset", "寄りは有限値で指定してください", e, sec)
					break
				}
			}
		}
		if m.LabelOffsetPaperMM != nil {
			for _, v := range m.LabelOffsetPaperMM {
				if !finiteDXF(v) {
					add("error", "invalid_label_offset", "符号位置は有限値で指定してください", e, sec)
					break
				}
			}
		}
	}
	sectionIDs := make([]int, 0, len(s.Sections))
	for id := range s.Sections {
		sectionIDs = append(sectionIDs, id)
	}
	sort.Ints(sectionIDs)
	for _, id := range sectionIDs {
		m := s.Sections[id]
		if f.Sects[id] == nil {
			add("error", "missing_section", "設定対象の断面がありません", 0, id)
		}
		if m.Expected != nil {
			add("error", "section_identity", "expected は部材単位で指定してください", 0, id)
		}
		validate(m, 0, id)
	}
	memberIDs := make([]int, 0, len(s.Members))
	for id := range s.Members {
		memberIDs = append(memberIDs, id)
	}
	sort.Ints(memberIDs)
	for _, id := range memberIDs {
		m := s.Members[id]
		e := f.Elems[id]
		if e == nil {
			add("error", "missing_member", "設定対象の部材がありません", id, 0)
			continue
		}
		if m.Expected != nil {
			actual := make([]int, len(e.Enod))
			for i, n := range e.Enod {
				actual[i] = n.Num
			}
			sort.Ints(actual)
			expected := append([]int(nil), m.Expected.Nodes...)
			sort.Ints(expected)
			if e.Sect.Num != m.Expected.Section || fmt.Sprint(actual) != fmt.Sprint(expected) {
				add("error", "member_identity_changed", "部材の断面・接続節点が設定の記録と異なります", id, e.Sect.Num)
			}
		} else {
			add("warning", "member_identity_unbound", "部材設定に expected（断面・節点）の記録がありません", id, e.Sect.Num)
		}
	}
	ids := make([]int, 0, len(f.Elems))
	for id := range f.Elems {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	pending := false
	for _, id := range ids {
		e := f.Elems[id]
		m := overlayDXFDrawingMember(s.Sections[e.Sect.Num], s.Members[id])
		validate(m, id, e.Sect.Num)
		r.Members = append(r.Members, DXFResolvedDrawingMember{id, e.Sect.Num, m})
		pending = pending || m.TopOffsetMM != nil
		if m.OffsetMM != nil {
			for _, v := range m.OffsetMM {
				pending = pending || v != 0
			}
		}
	}
	if pending {
		add("warning", "geometry_metadata_only", "寄り・天端差は保存・検証されていますが、形状への反映は次の実装段階です。現在の外形は解析軸基準です", 0, 0)
	}
	views := func(view string) bool {
		if view == "all" {
			return true
		}
		kind, key, ok := strings.Cut(view, ":")
		if !ok {
			return false
		}
		if kind == "plan" {
			n, err := strconv.Atoi(key)
			return err == nil && n > 0 && n < len(f.Ai.Boundary)
		}
		if kind == "elevation" {
			if grids[strings.ToUpper(key)] {
				return true
			}
			_, ok := f.Kijuns[strings.ToLower(key)]
			return ok
		}
		return false
	}
	if s.Sheet != nil {
		keys := make([]string, 0, len(s.Sheet.ViewRotations))
		for key := range s.Sheet.ViewRotations {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if !views(key) {
				add("error", "unknown_rotated_view", "回転設定の表示対象が見つかりません: "+key, 0, 0)
			}
		}
	}
	noteIDs := map[string]bool{}
	for _, a := range s.Annotations {
		if a.ID == "" || noteIDs[a.ID] || strings.TrimSpace(a.Text) == "" || !views(a.View) || (a.Element == 0) == (a.PositionMM == nil) {
			add("error", "invalid_annotation", "注記のID・表示対象・本文・配置方法が不正です: "+a.ID, a.Element, 0)
		}
		if a.Element != 0 && f.Elems[a.Element] == nil {
			add("error", "missing_annotation_anchor", "注記の参照部材がありません: "+a.ID, a.Element, 0)
		}
		for _, v := range a.OffsetPaperMM {
			if !finiteDXF(v) {
				add("error", "invalid_annotation_coordinate", "注記位置は有限値で指定してください", a.Element, 0)
			}
		}
		if a.PositionMM != nil {
			for _, v := range a.PositionMM {
				if !finiteDXF(v) {
					add("error", "invalid_annotation_coordinate", "注記位置は有限値で指定してください", a.Element, 0)
				}
			}
		}
		noteIDs[a.ID] = true
	}
	if records, err := f.DXFSections(nil); err != nil {
		add("error", "section_data", err.Error(), 0, 0)
	} else {
		for _, rec := range records {
			for _, w := range rec.Warnings {
				code := "section_data"
				if strings.Contains(w, "異なります") {
					code = "inp_lst_mismatch"
				}
				add("warning", code, w, 0, rec.ID)
			}
		}
	}
	return r
}
func (f *Frame) applyDXFDrawingSettings(s *DXFDrawingSettings) (*DXFDrawingCheckReport, error) {
	r := f.CheckDXFDrawingSettings(s)
	if r.HasErrors() {
		return r, fmt.Errorf("図面設定にエラーがあります: %s", firstDXFDrawingError(r))
	}
	f.drawingSettings = cloneDXFDrawingSettings(s)
	for _, g := range s.Grids {
		start, end := []float64{0, 0, 0}, []float64{1, 1, 0}
		axis := 0
		if g.Axis == "Y" {
			axis = 1
		}
		start[axis], end[axis] = g.PositionMM/1000, g.PositionMM/1000
		key := strings.ToLower(g.Name)
		delete(f.Kijuns, key)
		_, _ = f.AddKijun(g.Name, start, end)
	}
	return r, nil
}
func firstDXFDrawingError(r *DXFDrawingCheckReport) string {
	for _, i := range r.Issues {
		if i.Severity == "error" {
			return i.Message
		}
	}
	return ""
}
func (f *Frame) loadDXFDrawingSettings(path string) (*DXFDrawingSettings, error) {
	explicit := path != ""
	if !explicit {
		if f.Path == "" {
			return nil, nil
		}
		path = DXFDrawingSettingsPath(f.Path)
	}
	s, err := ReadDXFDrawingSettings(path)
	if os.IsNotExist(err) && !explicit {
		return nil, nil
	}
	return s, err
}

// WithDXFDrawingSettings returns an isolated drawing model, never an edited
// analysis frame. Reports retain inherited values for review through MCP.
func (f *Frame) WithDXFDrawingSettings(s *DXFDrawingSettings) (*Frame, *DXFDrawingCheckReport, error) {
	r := f.Snapshot()
	report, err := r.applyDXFDrawingSettings(s)
	if err != nil {
		return nil, report, err
	}
	return r, report, nil
}
