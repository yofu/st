package stgio

import (
	"fmt"
	"image"
	"math"
	"sort"
	"strconv"
	"strings"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	st "github.com/yofu/st/stlib"
)

var viewFields = []string{"GFACT", "DISTR", "DISTL", "PHI", "THETA", "FOCUSX", "FOCUSY", "FOCUSZ", "CENTERX", "CENTERY", "XMIN", "XMAX", "YMIN", "YMAX", "ZMIN", "ZMAX"}
var parameterFields = []string{"PERIOD", "GAXISSIZE", "EAXISSIZE", "BONDSIZE", "CONFSIZE", "DFACT", "QFACT", "MFACT", "RFACT"}
var showFlags = []string{"GAXIS", "EAXIS", "BOND", "CONF", "PHINGE", "YIELD", "KIJUN", "DEFORMATION"}

func (w *Window) numericSetting(name string) *float64 {
	if w.frame == nil {
		return nil
	}
	v, s := w.frame.View, w.frame.Show
	switch name {
	case "GFACT":
		return &v.Gfact
	case "DISTR":
		return &v.Dists[0]
	case "DISTL":
		return &v.Dists[1]
	case "PHI":
		return &v.Angle[0]
	case "THETA":
		return &v.Angle[1]
	case "FOCUSX":
		return &v.Focus[0]
	case "FOCUSY":
		return &v.Focus[1]
	case "FOCUSZ":
		return &v.Focus[2]
	case "CENTERX":
		return &v.Center[0]
	case "CENTERY":
		return &v.Center[1]
	case "XMIN":
		return &s.Xrange[0]
	case "XMAX":
		return &s.Xrange[1]
	case "YMIN":
		return &s.Yrange[0]
	case "YMAX":
		return &s.Yrange[1]
	case "ZMIN":
		return &s.Zrange[0]
	case "ZMAX":
		return &s.Zrange[1]
	case "GAXISSIZE":
		return &s.GlobalAxisSize
	case "EAXISSIZE":
		return &s.ElementAxisSize
	case "BONDSIZE":
		return &s.BondSize
	case "CONFSIZE":
		return &s.ConfSize
	case "DFACT":
		return &s.Dfact
	case "QFACT":
		return &s.Qfact
	case "MFACT":
		return &s.Mfact
	case "RFACT":
		return &s.Rfact
	}
	return nil
}

func (w *Window) applySetting(name, text string) error {
	if w.frame == nil {
		return fmt.Errorf("no model")
	}
	switch name {
	case "PERIOD":
		p := strings.ToUpper(strings.TrimSpace(text))
		if p == "" {
			return fmt.Errorf("PERIOD is empty")
		}
		st.SetPeriod(w, p)
		return nil
	case "PROP_SECTION":
		return w.applySelectedSection(text)
	case "PROP_ETYPE":
		return w.applySelectedEtype(text)
	case "PEN_LINE_SECT":
		return w.applyPenSection(false, text)
	case "PEN_PLATE_SECT":
		return w.applyPenSection(true, text)
	}
	p := w.numericSetting(name)
	if p == nil {
		return fmt.Errorf("unknown setting %s", name)
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return fmt.Errorf("%s: 有限の数値を入力してください", name)
	}
	if (name == "GFACT" || name == "DISTR" || name == "DISTL") && v <= 0 {
		return fmt.Errorf("%s must be positive", name)
	}
	for _, n := range parameterFields[1:] {
		if name == n && v < 0 {
			return fmt.Errorf("%s must be nonnegative", name)
		}
	}
	for _, axis := range []string{"X", "Y", "Z"} {
		if name == axis+"MIN" && v > *w.numericSetting(axis + "MAX") || name == axis+"MAX" && v < *w.numericSetting(axis + "MIN") {
			return fmt.Errorf("%s: min must not exceed max", name)
		}
	}
	for _, field := range viewFields {
		if field == name {
			w.cancelViewAnimation()
			break
		}
	}
	*p = v
	return nil
}

func isFixedBraceEtype(etype int) bool {
	return etype == st.WBRACE || etype == st.SBRACE
}

func isElemPropEditable(el *st.Elem) bool {
	return el != nil && !el.Lock && !isFixedBraceEtype(el.Etype)
}

func (w *Window) applySelectedSection(text string) error {
	if w.frame == nil {
		return fmt.Errorf("no model")
	}
	els := nonNilElems(w.SelectedElems())
	if len(els) == 0 {
		return fmt.Errorf("SECTION: 部材が選択されていません")
	}
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 {
		return fmt.Errorf("SECTION: 断面番号を入力してください")
	}
	snum, err := strconv.Atoi(fields[0])
	if err != nil {
		return fmt.Errorf("SECTION: 断面番号は整数で入力してください (%q)", fields[0])
	}
	sec, ok := w.frame.Sects[snum]
	if !ok || sec == nil {
		return fmt.Errorf("SECTION: 断面 %d は存在しません", snum)
	}
	editable := 0
	fixedBrace := 0
	changed := 0
	for _, el := range els {
		if isFixedBraceEtype(el.Etype) {
			fixedBrace++
			continue
		}
		if el.Lock {
			continue
		}
		editable++
		if el.Sect != sec {
			el.Sect = sec
			changed++
		}
	}
	if editable == 0 {
		if fixedBrace > 0 {
			return fmt.Errorf("SECTION: WBRACE / SBRACE は変更できません")
		}
		return fmt.Errorf("SECTION: 選択部材はすべてロックされています")
	}
	if changed > 0 {
		st.Snapshot(w)
		w.Changed(true)
	}
	return nil
}

func parseEtypeText(text string) (int, error) {
	word := strings.TrimSpace(text)
	if word == "" {
		return 0, fmt.Errorf("ETYPE: 種別を入力してください (COLUMN/GIRDER/BRACE/WALL/SLAB)")
	}
	switch {
	case st.Re_column.MatchString(word):
		return st.COLUMN, nil
	case st.Re_girder.MatchString(word):
		return st.GIRDER, nil
	case st.Re_brace.MatchString(word):
		return st.BRACE, nil
	case st.Re_wall.MatchString(word):
		return st.WALL, nil
	case st.Re_slab.MatchString(word):
		return st.SLAB, nil
	}
	if num, err := strconv.Atoi(word); err == nil {
		switch num {
		case st.COLUMN, st.GIRDER, st.BRACE, st.WALL, st.SLAB:
			return num, nil
		}
	}
	return 0, fmt.Errorf("ETYPE: 不明な種別です (%q)", word)
}

func (w *Window) applySelectedEtype(text string) error {
	if w.frame == nil {
		return fmt.Errorf("no model")
	}
	els := nonNilElems(w.SelectedElems())
	if len(els) == 0 {
		return fmt.Errorf("ETYPE: 部材が選択されていません")
	}
	etype, err := parseEtypeText(text)
	if err != nil {
		return err
	}
	wantLine := etype == st.COLUMN || etype == st.GIRDER || etype == st.BRACE
	editable := 0
	fixedBrace := 0
	for _, el := range els {
		if isFixedBraceEtype(el.Etype) {
			fixedBrace++
			continue
		}
		if el.Lock {
			continue
		}
		editable++
		if el.IsLineElem() != wantLine {
			return fmt.Errorf("ETYPE: %s は選択部材（線材/面材）と整合しません", st.ETYPES[etype])
		}
	}
	if editable == 0 {
		if fixedBrace > 0 {
			return fmt.Errorf("ETYPE: WBRACE / SBRACE は変更できません")
		}
		return fmt.Errorf("ETYPE: 選択部材はすべてロックされています")
	}
	changed := 0
	for _, el := range els {
		if !isElemPropEditable(el) {
			continue
		}
		if el.Etype != etype {
			el.Etype = etype
			changed++
		}
	}
	if changed > 0 {
		st.Snapshot(w)
		w.Changed(true)
	}
	return nil
}

func (w *Window) applyPenSection(isPlate bool, text string) error {
	if w.frame == nil {
		return fmt.Errorf("no model")
	}
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 {
		return fmt.Errorf("PEN: 断面番号を入力してください")
	}
	snum, err := strconv.Atoi(fields[0])
	if err != nil {
		return fmt.Errorf("PEN: 断面番号は整数で入力してください (%q)", fields[0])
	}
	if _, ok := w.frame.Sects[snum]; !ok {
		return fmt.Errorf("PEN: 断面 %d は存在しません", snum)
	}
	inferred := st.InferEtypeFromSect(w.frame, snum, isPlate)
	if isPlate {
		w.SetPlatePen(snum, inferred)
	} else {
		w.SetLinePen(snum, inferred)
	}
	return nil
}

func (w *Window) applyPenEtype(isPlate bool, etype int) error {
	if w.frame == nil {
		return fmt.Errorf("no model")
	}
	if isPlate {
		switch etype {
		case st.NULL, st.WALL, st.SLAB:
			sec, _ := st.ResolvePlatePen(w, w.frame)
			snum := w.PlatePen().Sect
			if snum == 0 && sec != nil {
				snum = sec.Num
			}
			w.SetPlatePen(snum, etype)
			return nil
		default:
			return fmt.Errorf("PLATE PEN: %s は面材種別ではありません", st.ETYPES[etype])
		}
	}
	switch etype {
	case st.NULL, st.COLUMN, st.GIRDER, st.BRACE:
		sec, _ := st.ResolveLinePen(w, w.frame)
		snum := w.LinePen().Sect
		if snum == 0 && sec != nil {
			snum = sec.Num
		}
		w.SetLinePen(snum, etype)
		return nil
	default:
		return fmt.Errorf("LINE PEN: %s は線材種別ではありません", st.ETYPES[etype])
	}
}

func (w *Window) pickPenFromSelected() error {
	if w.frame == nil {
		return fmt.Errorf("no model")
	}
	for _, el := range nonNilElems(w.SelectedElems()) {
		if el.Sect == nil || isFixedBraceEtype(el.Etype) {
			continue
		}
		if el.IsLineElem() {
			et := el.Etype
			if et != st.COLUMN && et != st.GIRDER && et != st.BRACE {
				et = st.NULL
			}
			w.SetLinePen(el.Sect.Num, et)
			w.History(fmt.Sprintf("LINE PEN: %d %s / %s", el.Sect.Num, el.Sect.Name, st.ETYPES[et]))
			return nil
		}
		et := el.Etype
		if et != st.WALL && et != st.SLAB {
			et = st.NULL
		}
		w.SetPlatePen(el.Sect.Num, et)
		w.History(fmt.Sprintf("PLATE PEN: %d %s / %s", el.Sect.Num, el.Sect.Name, st.ETYPES[et]))
		return nil
	}
	return fmt.Errorf("PEN: 取得可能な部材が選択されていません")
}

func nonNilElems(els []*st.Elem) []*st.Elem {
	out := make([]*st.Elem, 0, len(els))
	for _, el := range els {
		if el != nil {
			out = append(out, el)
		}
	}
	return out
}

func selectionKey(els []*st.Elem) string {
	if len(els) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d:", len(els))
	limit := min(len(els), 32)
	for i := 0; i < limit; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.Itoa(els[i].Num))
	}
	if len(els) > limit {
		fmt.Fprintf(&b, "..%d", els[len(els)-1].Num)
	}
	return b.String()
}

func (w *Window) flagValue(name string) bool {
	s := w.frame.Show
	switch name {
	case "PERSPECTIVE":
		return w.frame.View.Perspective
	case "GAXIS":
		return s.GlobalAxis
	case "EAXIS":
		return s.ElementAxis
	case "BOND":
		return s.Bond
	case "CONF":
		return s.Conf
	case "PHINGE":
		return s.Phinge
	case "YIELD":
		return s.YieldFunction
	case "KIJUN":
		return s.Kijun
	case "DEFORMATION":
		return s.PlotState&st.PLOT_DEFORMED != 0
	case "SRCAN_RATE":
		return s.SrcanRate != 0
	}
	var i, j int
	if _, e := fmt.Sscanf(name, "ETYPE/%d", &i); e == nil {
		return s.Etype[i]
	}
	if _, e := fmt.Sscanf(name, "SECT/%d", &i); e == nil {
		return s.Sect[i]
	}
	if _, e := fmt.Sscanf(name, "STRESS/%d/%d", &i, &j); e == nil {
		return s.Stress[i]&(1<<j) != 0
	}
	if _, e := fmt.Sscanf(name, "NC/%d", &i); e == nil {
		return s.NodeCaption&(1<<i) != 0
	}
	if _, e := fmt.Sscanf(name, "EC/%d", &i); e == nil {
		return s.ElemCaption&(1<<i) != 0
	}
	if _, e := fmt.Sscanf(name, "RATE/%d", &i); e == nil {
		return s.SrcanRate&(1<<i) != 0
	}
	return false
}

func (w *Window) applyFlag(name string, value bool) {
	if w.frame == nil {
		return
	}
	s := w.frame.Show
	switch name {
	case "PERSPECTIVE":
		w.cancelViewAnimation()
		w.frame.View.Perspective = value
		w.frame.View = st.CenterView(w, w.frame.View.Angle)
	case "GAXIS":
		s.GlobalAxis = value
	case "EAXIS":
		s.ElementAxis = value
	case "BOND":
		s.Bond = value
	case "CONF":
		s.Conf = value
	case "PHINGE":
		s.Phinge = value
	case "YIELD":
		s.YieldFunction = value
	case "KIJUN":
		s.Kijun = value
	case "DEFORMATION":
		if value {
			st.DeformationOn(w, false)
		} else {
			st.DeformationOff(w, false)
		}
	case "SRCAN_RATE":
		if value {
			st.SrcanRateOn(w)
		} else {
			st.SrcanRateOff(w)
		}
	default:
		var i, j int
		switch {
		case scanFlag(name, "ETYPE/%d", &i):
			s.Etype[i] = value
		case scanFlag(name, "SECT/%d", &i):
			s.Sect[i] = value
		case scanFlag(name, "STRESS/%d/%d", &i, &j):
			if value {
				s.Stress[i] |= 1 << j
			} else {
				s.Stress[i] &^= 1 << j
			}
		case scanFlag(name, "NC/%d", &i):
			if value {
				s.NodeCaption |= 1 << i
			} else {
				s.NodeCaption &^= 1 << i
			}
		case scanFlag(name, "EC/%d", &i):
			if value {
				s.ElemCaption |= 1 << i
			} else {
				s.ElemCaption &^= 1 << i
			}
		case scanFlag(name, "RATE/%d", &i):
			if i >= 0 && i < len(st.SRCANS) {
				if value {
					st.SrcanRateOn(w, st.SRCANS[i])
				} else {
					st.SrcanRateOff(w, st.SRCANS[i])
				}
			}
		}
	}
}

func scanFlag(name, format string, values ...any) bool {
	_, err := fmt.Sscanf(name, format, values...)
	return err == nil
}

func (w *Window) snapshotControls() {
	u := w.ui
	if w.frame == nil {
		return
	}
	f := w.frame
	for _, fields := range [][]string{viewFields, parameterFields} {
		for _, name := range fields {
			if p := w.numericSetting(name); p != nil {
				u.values[name] = strconv.FormatFloat(*p, 'g', 8, 64)
			}
		}
	}
	u.values["PERIOD"] = f.Show.Period
	u.values["PATH"] = f.Path
	u.values["CWD"] = w.Cwd()
	if int(f.Show.ColorMode) < len(st.ECOLORS) {
		u.values["COLOR"] = st.ECOLORS[f.Show.ColorMode]
	}
	u.flags["CANVAS_DARK"] = w.canvasDark
	for _, name := range append([]string{"PERSPECTIVE"}, showFlags...) {
		u.flags[name] = w.flagValue(name)
	}
	for i := range st.ETYPES {
		u.flags[fmt.Sprintf("ETYPE/%d", i)] = f.Show.Etype[i]
		for j := 0; j < 6; j++ {
			key := fmt.Sprintf("STRESS/%d/%d", i, j)
			u.flags[key] = w.flagValue(key)
		}
	}
	for i := range st.NODECAPTIONS {
		key := fmt.Sprintf("NC/%d", i)
		u.flags[key] = w.flagValue(key)
	}
	for i := range st.ELEMCAPTIONS {
		key := fmt.Sprintf("EC/%d", i)
		u.flags[key] = w.flagValue(key)
	}
	u.flags["SRCAN_RATE"] = w.flagValue("SRCAN_RATE")
	for i := range st.SRCANS {
		key := fmt.Sprintf("RATE/%d", i)
		u.flags[key] = w.flagValue(key)
	}
	u.sections = u.sections[:0]
	for i := range f.Sects {
		u.sections = append(u.sections, i)
		u.flags[fmt.Sprintf("SECT/%d", i)] = f.Show.Sect[i]
		u.values[fmt.Sprintf("SECT/%d", i)] = f.Sects[i].Name
	}
	sort.Ints(u.sections)
	els := nonNilElems(w.SelectedElems())
	key := selectionKey(els)
	if key != u.propSelectionKey {
		u.propSelectionKey = key
		if u.sectionPickerTarget == "" || u.sectionPickerTarget == "PROP" {
			u.sectionPickerOpen = false
		}
		for _, name := range []string{"PROP_SECTION", "PROP_ETYPE"} {
			if field := u.fields[name]; field != nil {
				field.dirty = false
			}
		}
	}
	var length, area float64
	lines, plates, unlocked := 0, 0, 0
	editable, fixedBrace, editLines, editPlates := 0, 0, 0, 0
	sectMixed, etypeMixed := false, false
	var firstSect *st.Sect
	firstEtype := -1
	for _, e := range els {
		if !e.Lock {
			unlocked++
		}
		if isFixedBraceEtype(e.Etype) {
			fixedBrace++
		}
		if e.IsLineElem() {
			lines++
			length += e.Length()
			if isElemPropEditable(e) {
				editable++
				editLines++
			}
		} else {
			plates++
			area += e.Area()
			if isElemPropEditable(e) {
				editable++
				editPlates++
			}
		}
		if firstSect == nil && e.Sect != nil {
			firstSect = e.Sect
		} else if e.Sect != firstSect {
			sectMixed = true
		}
		if firstEtype < 0 {
			firstEtype = e.Etype
		} else if e.Etype != firstEtype {
			etypeMixed = true
		}
	}
	u.propElemCount = len(els)
	u.propLineCount = lines
	u.propPlateCount = plates
	u.propUnlocked = unlocked
	u.propEditable = editable
	u.propFixedBrace = fixedBrace
	u.propEditLines = editLines
	u.propEditPlates = editPlates
	u.propSectMixed = sectMixed
	u.propEtypeMixed = etypeMixed
	if editable == 0 && (u.sectionPickerTarget == "" || u.sectionPickerTarget == "PROP") {
		u.sectionPickerOpen = false
	}
	if len(els) == 0 {
		u.values["PROP_SECTION"] = ""
		u.values["PROP_ETYPE"] = ""
		u.propSectLabel = "-"
		u.propCommonEtype = 0
	} else {
		if sectMixed || firstSect == nil {
			u.values["PROP_SECTION"] = ""
			if firstSect != nil {
				u.propSectLabel = fmt.Sprintf("混在 (先頭: %d %s)", firstSect.Num, firstSect.Name)
			} else {
				u.propSectLabel = "混在"
			}
		} else {
			u.values["PROP_SECTION"] = strconv.Itoa(firstSect.Num)
			u.propSectLabel = firstSect.Name
		}
		if etypeMixed || firstEtype < 0 || firstEtype >= len(st.ETYPES) {
			u.values["PROP_ETYPE"] = ""
			u.propCommonEtype = 0
		} else {
			u.values["PROP_ETYPE"] = st.ETYPES[firstEtype]
			u.propCommonEtype = firstEtype
		}
	}
	u.properties = []string{
		fmt.Sprintf("LINE: %d   (%.3f m)", lines, length),
		fmt.Sprintf("PLATE: %d   (%.3f m²)", plates, area),
		fmt.Sprintf("NODES: %d   ELEMS: %d", len(w.SelectedNodes()), len(els)),
	}
	if len(els) > 0 {
		e := els[0]
		codeLine := fmt.Sprintf("CODE: %d   ENODS: %d", e.Num, e.Enods)
		if e.Lock {
			codeLine += "  [LOCK]"
		}
		u.properties = append(u.properties, codeLine)
		if unlocked < len(els) {
			u.properties = append(u.properties, fmt.Sprintf("LOCKED: %d / %d", len(els)-unlocked, len(els)))
		}
		for _, n := range e.Enod {
			u.properties = append(u.properties, fmt.Sprintf("NODE %d: %.3f, %.3f, %.3f", n.Num, n.Coord[0], n.Coord[1], n.Coord[2]))
		}
	}
	u.propNodeDetails = u.propNodeDetails[:0]
	if ns := w.SelectedNodes(); len(ns) > 0 && ns[0] != nil {
		n := ns[0]
		u.propNodeDetails = append(u.propNodeDetails, fmt.Sprintf("NODE: %d", n.Num), fmt.Sprintf("XYZ: %.4f, %.4f, %.4f", n.Coord[0], n.Coord[1], n.Coord[2]))
	}
	lineSec, lineEt := st.ResolveLinePen(w, f)
	if lineSec != nil {
		u.values["PEN_LINE_SECT"] = strconv.Itoa(lineSec.Num)
		u.penLineSectLabel = lineSec.Name
	} else {
		u.values["PEN_LINE_SECT"] = ""
		u.penLineSectLabel = "-"
	}
	u.penLineEtype = lineEt
	plateSec, plateEt := st.ResolvePlatePen(w, f)
	if plateSec != nil {
		u.values["PEN_PLATE_SECT"] = strconv.Itoa(plateSec.Num)
		u.penPlateSectLabel = plateSec.Name
	} else {
		u.values["PEN_PLATE_SECT"] = ""
		u.penPlateSectLabel = "-"
	}
	u.penPlateEtype = plateEt
	u.status = fmt.Sprintf("%s | %d nodes / %d members | %s", f.Name, len(f.Nodes), len(f.Elems), f.Show.Period)
}

func (w *Window) cancelPropertyFieldEdit(gtx layout.Context) bool {
	u := w.ui
	cancelled := false
	for _, name := range []string{"PROP_SECTION", "PROP_ETYPE", "PEN_LINE_SECT", "PEN_PLATE_SECT"} {
		field := u.fields[name]
		if field == nil {
			continue
		}
		if gtx.Focused(&field.ed) || field.focused || field.dirty {
			field.dirty = false
			field.ed.SetText(u.values[name])
			cancelled = true
		}
	}
	if cancelled {
		gtx.Execute(key.FocusCmd{Tag: &u.command})
	}
	return cancelled
}

func (w *Window) settingEditor(gtx layout.Context, name string) layout.Dimensions {
	return w.settingEditorLabeled(gtx, name, name)
}

func (w *Window) settingEditorCore(gtx layout.Context, name, hint string) layout.Dimensions {
	u := w.ui
	field := u.fields[name]
	if field == nil {
		field = &uiField{ed: widget.Editor{SingleLine: true, Submit: true}}
		u.fields[name] = field
	}
	focused := gtx.Focused(&field.ed)
	commit := false
	for {
		e, ok := field.ed.Update(gtx)
		if !ok {
			break
		}
		switch e.(type) {
		case widget.ChangeEvent:
			field.dirty = true
		case widget.SubmitEvent:
			commit = true
		}
	}
	if field.focused && !focused && field.dirty {
		commit = true
	}
	if commit {
		text := field.ed.Text()
		field.dirty = false
		w.enqueue(func() {
			if err := w.applySetting(name, text); err != nil {
				w.History(err.Error())
			}
			w.Redraw()
		})
	}
	if !focused && !field.dirty && !commit {
		if field.ed.Text() != u.values[name] {
			field.ed.SetText(u.values[name])
		}
	}
	field.focused = focused
	bg := iupBgField
	if focused {
		bg = iupBgFieldAct
	}
	return widget.Border{Color: iupBorder, Width: unit.Dp(1)}.Layout(gtx, func(c layout.Context) layout.Dimensions {
		macro := op.Record(c.Ops)
		dims := layout.Inset{Top: unit.Dp(1), Bottom: unit.Dp(1), Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(c, func(c layout.Context) layout.Dimensions {
			e := material.Editor(w.Theme, &field.ed, hint)
			e.TextSize = unit.Sp(11)
			e.Color = iupFgLabel
			e.HintColor = iupFgOff
			return e.Layout(c)
		})
		call := macro.Stop()
		paint.FillShape(c.Ops, bg, clip.Rect{Max: dims.Size}.Op())
		call.Add(c.Ops)
		return dims
	})
}

func (w *Window) settingEditorLabeled(gtx layout.Context, name, label string) layout.Dimensions {
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(c layout.Context) layout.Dimensions {
			c.Constraints.Min.X = c.Dp(66)
			return layout.Inset{Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(c, func(c layout.Context) layout.Dimensions {
				return w.uiLabel(c, label)
			})
		}),
		layout.Flexed(1, func(c layout.Context) layout.Dimensions {
			return w.settingEditorCore(c, name, "")
		}),
	)
}

func (w *Window) propertySectionEditor(gtx layout.Context) layout.Dimensions {
	u := w.ui
	hint := ""
	if u.propSectMixed {
		hint = "混在"
	}
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(c layout.Context) layout.Dimensions {
			c.Constraints.Min.X = c.Dp(66)
			return layout.Inset{Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(c, func(c layout.Context) layout.Dimensions {
				return w.uiLabel(c, "SECTION")
			})
		}),
		layout.Flexed(1, func(c layout.Context) layout.Dimensions {
			return w.settingEditorCore(c, "PROP_SECTION", hint)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(2)}.Layout),
		layout.Rigid(func(c layout.Context) layout.Dimensions {
			bg := iupBgButton
			if u.sectionPickerOpen && u.sectionPickerTarget == "PROP" {
				bg = iupBgCommand
			}
			return widget.Border{Color: iupBorder, Width: unit.Dp(1)}.Layout(c, func(c layout.Context) layout.Dimensions {
				return w.flatButton(c, "prop-section-picker", "▼", bg, iupFgLabel,
					layout.Inset{Top: unit.Dp(1), Bottom: unit.Dp(1), Left: unit.Dp(5), Right: unit.Dp(5)},
					func() {
						u.menu = -1
						u.contextItems = nil
						if u.sectionPickerOpen && u.sectionPickerTarget == "PROP" {
							u.sectionPickerOpen = false
						} else {
							u.sectionPickerOpen = true
							u.sectionPickerTarget = "PROP"
							u.sectionPickerList.Position = layout.Position{}
						}
					},
				)
			})
		}),
	)
}

func (w *Window) propertyEtypeEditor(gtx layout.Context) layout.Dimensions {
	u := w.ui
	hint := ""
	if u.propEtypeMixed {
		hint = "混在"
	}
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(c layout.Context) layout.Dimensions {
			c.Constraints.Min.X = c.Dp(66)
			return layout.Inset{Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(c, func(c layout.Context) layout.Dimensions {
				return w.uiLabel(c, "ETYPE")
			})
		}),
		layout.Flexed(1, func(c layout.Context) layout.Dimensions {
			return w.settingEditorCore(c, "PROP_ETYPE", hint)
		}),
	)
}

func (w *Window) propertyEtypeButtons(gtx layout.Context) layout.Dimensions {
	u := w.ui
	type etypeChoice struct {
		code  int
		short string
		line  bool
	}
	choices := []etypeChoice{
		{st.COLUMN, "COL", true},
		{st.GIRDER, "GIR", true},
		{st.BRACE, "BRC", true},
		{st.WALL, "WAL", false},
		{st.SLAB, "SLB", false},
	}
	inset := layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(2), Right: unit.Dp(2)}
	children := make([]layout.FlexChild, 0, len(choices)*2-1)
	for i, ch := range choices {
		ch := ch
		if i > 0 {
			children = append(children, layout.Rigid(layout.Spacer{Width: unit.Dp(2)}.Layout))
		}
		children = append(children, layout.Flexed(1, func(c layout.Context) layout.Dimensions {
			compatible := u.propEditable > 0 && ((ch.line && u.propEditPlates == 0) || (!ch.line && u.propEditLines == 0))
			active := u.propCommonEtype == ch.code
			bg := iupBgButton
			fg := iupFgLabel
			if active {
				bg = iupBgCommand
			}
			if !compatible {
				bg = iupBgLabel
				fg = iupFgOff
			}
			return widget.Border{Color: iupBorder, Width: unit.Dp(1)}.Layout(c, func(c layout.Context) layout.Dimensions {
				d := w.flatButton(c, fmt.Sprintf("prop-etype-%d", ch.code), ch.short, bg, fg, inset, func() {
					if !compatible {
						w.History(fmt.Sprintf("ETYPE: %s は現在の選択部材に適用できません", st.ETYPES[ch.code]))
						return
					}
					name := st.ETYPES[ch.code]
					w.enqueue(func() {
						if err := w.applySelectedEtype(name); err != nil {
							w.History(err.Error())
						}
						w.Redraw()
					})
				})
				if active {
					paint.FillShape(c.Ops, iupFgSection, clip.Rect(image.Rect(0, 0, d.Size.X, c.Dp(2))).Op())
				}
				return d
			})
		}))
	}
	return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(4)}.Layout(gtx, func(c layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.Middle}.Layout(c, children...)
	})
}

func (w *Window) penSectionEditor(gtx layout.Context, isPlate bool) layout.Dimensions {
	u := w.ui
	label, fieldName, pickerBtn, target := "LINE", "PEN_LINE_SECT", "pen-line-picker", "PEN_LINE"
	if isPlate {
		label, fieldName, pickerBtn, target = "PLATE", "PEN_PLATE_SECT", "pen-plate-picker", "PEN_PLATE"
	}
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(c layout.Context) layout.Dimensions {
			c.Constraints.Min.X = c.Dp(66)
			return layout.Inset{Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(c, func(c layout.Context) layout.Dimensions {
				return w.uiLabel(c, label)
			})
		}),
		layout.Flexed(1, func(c layout.Context) layout.Dimensions {
			return w.settingEditorCore(c, fieldName, "")
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(2)}.Layout),
		layout.Rigid(func(c layout.Context) layout.Dimensions {
			bg := iupBgButton
			if u.sectionPickerOpen && u.sectionPickerTarget == target {
				bg = iupBgCommand
			}
			return widget.Border{Color: iupBorder, Width: unit.Dp(1)}.Layout(c, func(c layout.Context) layout.Dimensions {
				return w.flatButton(c, pickerBtn, "▼", bg, iupFgLabel,
					layout.Inset{Top: unit.Dp(1), Bottom: unit.Dp(1), Left: unit.Dp(5), Right: unit.Dp(5)},
					func() {
						u.menu = -1
						u.contextItems = nil
						if u.sectionPickerOpen && u.sectionPickerTarget == target {
							u.sectionPickerOpen = false
						} else {
							u.sectionPickerOpen = true
							u.sectionPickerTarget = target
							u.sectionPickerList.Position = layout.Position{}
						}
					},
				)
			})
		}),
	)
}

func (w *Window) penEtypeButtons(gtx layout.Context, isPlate bool) layout.Dimensions {
	u := w.ui
	type penChoice struct {
		code  int
		short string
	}
	choices := []penChoice{
		{st.COLUMN, "COL"},
		{st.GIRDER, "GIR"},
		{st.BRACE, "BRC"},
		{st.NULL, "NULL"},
	}
	prefix := "pen-line-etype-"
	current := u.penLineEtype
	if isPlate {
		choices = []penChoice{
			{st.WALL, "WAL"},
			{st.SLAB, "SLB"},
			{st.NULL, "NULL"},
		}
		prefix = "pen-plate-etype-"
		current = u.penPlateEtype
	}
	inset := layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(2), Right: unit.Dp(2)}
	children := make([]layout.FlexChild, 0, len(choices)*2-1)
	for i, ch := range choices {
		ch := ch
		if i > 0 {
			children = append(children, layout.Rigid(layout.Spacer{Width: unit.Dp(2)}.Layout))
		}
		children = append(children, layout.Flexed(1, func(c layout.Context) layout.Dimensions {
			active := current == ch.code
			bg := iupBgButton
			if active {
				bg = iupBgCommand
			}
			return widget.Border{Color: iupBorder, Width: unit.Dp(1)}.Layout(c, func(c layout.Context) layout.Dimensions {
				d := w.flatButton(c, fmt.Sprintf("%s%d", prefix, ch.code), ch.short, bg, iupFgLabel, inset, func() {
					w.enqueue(func() {
						if err := w.applyPenEtype(isPlate, ch.code); err != nil {
							w.History(err.Error())
						}
						w.Redraw()
					})
				})
				if active {
					paint.FillShape(c.Ops, iupFgSection, clip.Rect(image.Rect(0, 0, d.Size.X, c.Dp(2))).Op())
				}
				return d
			})
		}))
	}
	return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(4)}.Layout(gtx, func(c layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.Middle}.Layout(c, children...)
	})
}

func (w *Window) sectionPickerCandidatesFor(target string) []int {
	u := w.ui
	var allowLine, allowPlate bool
	switch target {
	case "PEN_LINE":
		allowLine = true
	case "PEN_PLATE":
		allowPlate = true
	default:
		if u.propEditable == 0 {
			return nil
		}
		allowLine = u.propEditLines > 0
		allowPlate = u.propEditPlates > 0
	}
	out := make([]int, 0, len(u.sections))
	for _, id := range u.sections {
		if allowLine && id >= 100 && id <= 699 {
			out = append(out, id)
		} else if allowPlate && id >= 700 && id <= 899 {
			out = append(out, id)
		}
	}
	return out
}

func (w *Window) sectionPickerCandidates() []int {
	return w.sectionPickerCandidatesFor("PROP")
}

func (w *Window) propertyReadOnlyRow(gtx layout.Context, label, value string) layout.Dimensions {
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(c layout.Context) layout.Dimensions {
			c.Constraints.Min.X = c.Dp(66)
			return layout.Inset{Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(c, func(c layout.Context) layout.Dimensions {
				return w.uiLabel(c, label)
			})
		}),
		layout.Flexed(1, func(c layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(c, func(c layout.Context) layout.Dimensions {
				s := material.Body2(w.Theme, value)
				s.TextSize = unit.Sp(11)
				s.Color = iupFgLabel
				s.MaxLines = 1
				return s.Layout(c)
			})
		}),
	)
}

func (w *Window) layoutSectionPickerPopup(gtx layout.Context) {
	u := w.ui
	target := u.sectionPickerTarget
	if target == "" {
		target = "PROP"
	}
	if u.tab != 2 || (target == "PROP" && u.propEditable == 0) {
		u.sectionPickerOpen = false
		return
	}
	candidates := w.sectionPickerCandidatesFor(target)
	shield := u.button("section-picker-shield")
	for shield.Clicked(gtx) {
		u.sectionPickerOpen = false
	}
	shield.Layout(gtx, func(c layout.Context) layout.Dimensions { return layout.Dimensions{Size: c.Constraints.Max} })
	width := min(gtx.Dp(260), max(gtx.Dp(140), gtx.Constraints.Max.X-u.panelRect.Max.X))
	x := u.panelRect.Max.X
	if x+width > gtx.Constraints.Max.X {
		x = max(0, gtx.Constraints.Max.X-width)
	}
	y := min(u.panelRect.Min.Y+gtx.Dp(110), max(0, gtx.Constraints.Max.Y-gtx.Dp(60)))
	itemH := 24
	rowCount := max(1, len(candidates))
	height := min(gtx.Dp(unit.Dp(rowCount*itemH+6)), max(gtx.Dp(36), gtx.Constraints.Max.Y-y-gtx.Dp(8)))
	rect := image.Rect(x, y, x+width, y+height)
	itemInset := layout.Inset{Top: unit.Dp(3), Bottom: unit.Dp(3), Left: unit.Dp(8), Right: unit.Dp(8)}
	current := u.values["PROP_SECTION"]
	isMixed := u.propSectMixed
	btnPrefix := "prop-section-pick-"
	switch target {
	case "PEN_LINE":
		current = u.values["PEN_LINE_SECT"]
		isMixed = false
		btnPrefix = "pen-line-pick-"
	case "PEN_PLATE":
		current = u.values["PEN_PLATE_SECT"]
		isMixed = false
		btnPrefix = "pen-plate-pick-"
	}
	region(gtx, rect, func(c layout.Context) layout.Dimensions {
		paint.Fill(c.Ops, iupBgMenu)
		return widget.Border{Color: iupBorder, Width: unit.Dp(1)}.Layout(c, func(c layout.Context) layout.Dimensions {
			if len(candidates) == 0 {
				return itemInset.Layout(c, func(c layout.Context) layout.Dimensions {
					s := material.Body2(w.Theme, "該当する断面候補がありません")
					s.TextSize = unit.Sp(11)
					s.Color = iupFgOff
					return s.Layout(c)
				})
			}
			return material.List(w.Theme, &u.sectionPickerList).Layout(c, len(candidates), func(c layout.Context, i int) layout.Dimensions {
				id := candidates[i]
				idText := strconv.Itoa(id)
				label := fmt.Sprintf("%d  %s", id, u.values[fmt.Sprintf("SECT/%d", id)])
				bg := iupBgMenu
				if idText == current && !isMixed {
					bg = iupBgCommand
				}
				b := u.button(fmt.Sprintf("%s%d", btnPrefix, id))
				for b.Clicked(c) {
					u.sectionPickerOpen = false
					w.enqueue(func() {
						var err error
						switch target {
						case "PEN_LINE":
							err = w.applyPenSection(false, idText)
						case "PEN_PLATE":
							err = w.applyPenSection(true, idText)
						default:
							err = w.applySelectedSection(idText)
						}
						if err != nil {
							w.History(err.Error())
						}
						w.Redraw()
					})
					w.QueueRedrawAll()
				}
				return b.Layout(c, func(c layout.Context) layout.Dimensions {
					macro := op.Record(c.Ops)
					dims := itemInset.Layout(c, func(c layout.Context) layout.Dimensions {
						c.Constraints.Min.X = c.Constraints.Max.X
						return layout.W.Layout(c, func(c layout.Context) layout.Dimensions {
							s := material.Body2(w.Theme, label)
							s.TextSize = unit.Sp(11)
							s.Color = iupFgLabel
							s.MaxLines = 1
							return s.Layout(c)
						})
					})
					call := macro.Stop()
					paint.FillShape(c.Ops, bg, clip.Rect{Max: dims.Size}.Op())
					call.Add(c.Ops)
					return dims
				})
			})
		})
	})
}

func (w *Window) textToggle(gtx layout.Context, key, label string, alignLeft bool) layout.Dimensions {
	u := w.ui
	b := u.button("toggle-" + key)
	for b.Clicked(gtx) {
		next := !u.flags[key]
		u.flags[key] = next
		w.enqueue(func() { w.applyFlag(key, next); w.Redraw() })
		w.QueueRedrawAll()
	}
	fg := iupFgOff
	if u.flags[key] {
		fg = iupFgLabel
	}
	return b.Layout(gtx, func(c layout.Context) layout.Dimensions {
		inset := layout.Inset{Top: unit.Dp(1), Bottom: unit.Dp(1), Left: unit.Dp(6), Right: unit.Dp(4)}
		if !alignLeft {
			inset = layout.Inset{Top: unit.Dp(1), Bottom: unit.Dp(1), Left: unit.Dp(2), Right: unit.Dp(2)}
		}
		return inset.Layout(c, func(c layout.Context) layout.Dimensions {
			draw := func(c layout.Context) layout.Dimensions {
				s := material.Body2(w.Theme, label)
				s.TextSize = unit.Sp(11)
				s.Color = fg
				s.MaxLines = 1
				return s.Layout(c)
			}
			if alignLeft {
				c.Constraints.Min.X = c.Constraints.Max.X
				return layout.W.Layout(c, draw)
			}
			return layout.Center.Layout(c, draw)
		})
	})
}

func (w *Window) perspectiveToggle(gtx layout.Context) layout.Dimensions {
	u := w.ui
	b := u.button("toggle-PERSPECTIVE")
	for b.Clicked(gtx) {
		next := !u.flags["PERSPECTIVE"]
		u.flags["PERSPECTIVE"] = next
		w.enqueue(func() { w.applyFlag("PERSPECTIVE", next); w.Redraw() })
		w.QueueRedrawAll()
	}
	label := "AXONOMETRIC"
	if u.flags["PERSPECTIVE"] {
		label = "PERSPECTIVE"
	}
	return b.Layout(gtx, func(c layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(1), Bottom: unit.Dp(1), Left: unit.Dp(6), Right: unit.Dp(4)}.Layout(c, func(c layout.Context) layout.Dimensions {
			c.Constraints.Min.X = c.Constraints.Max.X
			return layout.W.Layout(c, func(c layout.Context) layout.Dimensions {
				s := material.Body2(w.Theme, label)
				s.TextSize = unit.Sp(11)
				s.Color = iupFgLabel
				s.MaxLines = 1
				return s.Layout(c)
			})
		})
	})
}

func (w *Window) compactToggle(gtx layout.Context, key, label string) layout.Dimensions {
	return w.textToggle(gtx, key, label, false)
}

func (w *Window) sectionHeading(gtx layout.Context, label string) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(1)}.Layout(gtx, func(c layout.Context) layout.Dimensions {
		s := material.Body2(w.Theme, label)
		s.TextSize = unit.Sp(11)
		s.Color = iupFgSection
		return s.Layout(c)
	})
}

func (w *Window) layoutPanel(gtx layout.Context) layout.Dimensions {
	u := w.ui
	paint.Fill(gtx.Ops, iupBgLabel)
	dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(c layout.Context) layout.Dimensions {
			children := []layout.FlexChild{}
			tabInset := layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(8), Right: unit.Dp(8)}
			for i, title := range []string{"View", "Show", "Property"} {
				i, title := i, title
				children = append(children, layout.Flexed(1, func(c layout.Context) layout.Dimensions {
					bg := iupBgTabInact
					if u.tab == i {
						bg = iupBgLabel
					}
					d := w.flatButton(c, "tab-"+title, title, bg, iupFgLabel, tabInset, func() { u.tab = i; u.sectionPickerOpen = false })
					if u.tab == i {
						paint.FillShape(c.Ops, iupFgSection, clip.Rect(image.Rect(0, 0, d.Size.X, c.Dp(2))).Op())
					} else {
						paint.FillShape(c.Ops, iupBorder, clip.Rect(image.Rect(0, d.Size.Y-1, d.Size.X, d.Size.Y)).Op())
					}
					paint.FillShape(c.Ops, iupBorder, clip.Rect(image.Rect(d.Size.X-1, 0, d.Size.X, d.Size.Y)).Op())
					return d
				}))
			}
			return layout.Flex{}.Layout(c, children...)
		}),
		layout.Flexed(1, func(c layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(3), Bottom: unit.Dp(4), Left: unit.Dp(5), Right: unit.Dp(6)}.Layout(c, func(c layout.Context) layout.Dimensions {
				rows := w.panelRows()
				return material.List(w.Theme, &u.panelList[u.tab]).Layout(c, len(rows), func(c layout.Context, i int) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(1), Bottom: unit.Dp(1)}.Layout(c, rows[i])
				})
			})
		}),
	)
	// Right border separating left SideBar from the Canvas/History/Command column.
	paint.FillShape(gtx.Ops, iupBorder, clip.Rect(image.Rect(gtx.Constraints.Max.X-1, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Op())
	return dims
}

func (w *Window) panelRows() []layout.Widget {
	u := w.ui
	rows := []layout.Widget{}
	heading := func(label string) {
		rows = append(rows, func(c layout.Context) layout.Dimensions { return w.sectionHeading(c, label) })
	}
	edit := func(name, label string) {
		rows = append(rows, func(c layout.Context) layout.Dimensions { return w.settingEditorLabeled(c, name, label) })
	}
	toggle := func(key, label string) {
		rows = append(rows, func(c layout.Context) layout.Dimensions { return w.textToggle(c, key, label, true) })
	}
	toggleRow := func(items [][2]string) {
		rows = append(rows, func(c layout.Context) layout.Dimensions {
			children := make([]layout.FlexChild, 0, len(items))
			for _, item := range items {
				k, lbl := item[0], item[1]
				children = append(children, layout.Flexed(1, func(c layout.Context) layout.Dimensions {
					return w.compactToggle(c, k, lbl)
				}))
			}
			return layout.Flex{Alignment: layout.Middle}.Layout(c, children...)
		})
	}
	leftToggleRow := func(items [][2]string) {
		rows = append(rows, func(c layout.Context) layout.Dimensions {
			children := make([]layout.FlexChild, 0, len(items))
			for _, item := range items {
				k, lbl := item[0], item[1]
				children = append(children, layout.Flexed(1, func(c layout.Context) layout.Dimensions {
					return w.textToggle(c, k, lbl, true)
				}))
			}
			return layout.Flex{Alignment: layout.Middle}.Layout(c, children...)
		})
	}
	stressRow := func(etype int) {
		toggleRow([][2]string{
			{fmt.Sprintf("STRESS/%d/0", etype), "N"},
			{fmt.Sprintf("STRESS/%d/1", etype), "Qx"},
			{fmt.Sprintf("STRESS/%d/2", etype), "Qy"},
			{fmt.Sprintf("STRESS/%d/3", etype), "Mz"},
			{fmt.Sprintf("STRESS/%d/4", etype), "Mx"},
			{fmt.Sprintf("STRESS/%d/5", etype), "My"},
		})
	}
	switch u.tab {
	case 0:
		// Match IUP View tab order: VIEW -> DISTS -> ANGLE -> FOCUS -> CENTER -> RANGE.
		heading("VIEW")
		edit("GFACT", "GFACT")
		rows = append(rows, func(c layout.Context) layout.Dimensions { return w.perspectiveToggle(c) })
		heading("DISTS")
		edit("DISTR", "R")
		edit("DISTL", "L")
		heading("ANGLE")
		edit("PHI", "PHI")
		edit("THETA", "THETA")
		heading("FOCUS")
		edit("FOCUSX", "X")
		edit("FOCUSY", "Y")
		edit("FOCUSZ", "Z")
		heading("CENTER")
		edit("CENTERX", "X")
		edit("CENTERY", "Y")
		heading("RANGE")
		edit("XMAX", "Xmax")
		edit("XMIN", "Xmin")
		edit("YMAX", "Ymax")
		edit("YMIN", "Ymin")
		edit("ZMAX", "Zmax")
		edit("ZMIN", "Zmin")
		rows = append(rows, func(c layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(4)}.Layout(c, func(c layout.Context) layout.Dimensions {
				return w.smallButton(c, "fit-panel", "Fit / 全体表示", func() { w.menuAction("fit") })
			})
		})
	case 1:
		// Match IUP Show tab order: COLOR MODE -> PARAMETER -> SHOW -> NODE CAPTION -> ELEM CAPTION -> ETYPE -> SECTION.
		heading("COLOR MODE")
		rows = append(rows, func(c layout.Context) layout.Dimensions {
			bgLabel := "BG: WHITE"
			if u.flags["CANVAS_DARK"] {
				bgLabel = "BG: BLACK"
			}
			return layout.Flex{Alignment: layout.Middle}.Layout(c,
				layout.Flexed(1, func(c layout.Context) layout.Dimensions {
					return w.smallButton(c, "color-mode", u.values["COLOR"], func() { w.menuAction("color") })
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
				layout.Rigid(func(c layout.Context) layout.Dimensions {
					return w.smallButton(c, "canvas-bg", bgLabel, func() { w.menuAction("toggle-bg") })
				}),
			)
		})
		heading("PARAMETER")
		edit("PERIOD", "PERIOD")
		edit("GAXISSIZE", "GAXIS")
		edit("EAXISSIZE", "EAXIS")
		edit("BONDSIZE", "BOND")
		edit("CONFSIZE", "CONF")
		edit("DFACT", "DFACT")
		edit("QFACT", "QFACT")
		edit("MFACT", "MFACT")
		edit("RFACT", "RFACT")
		heading("SHOW")
		for _, name := range showFlags {
			toggle(name, name)
		}
		heading("NODE CAPTION")
		// Indices match st.NODECAPTIONS / st.NC_* bit positions:
		// 0:NC_NUM, 1:NC_WEIGHT, 2:NC_ZCOORD, 3..8:dX..tZ, 9..14:Rx..Mz, 15:NC_PILE.
		if len(st.NODECAPTIONS) >= 3 {
			toggle("NC/0", "CODE")
			toggle("NC/1", "WEIGHT")
			toggle("NC/2", "ZCOORD")
		}
		if len(st.NODECAPTIONS) >= 9 {
			toggleRow([][2]string{
				{"NC/3", "dX"}, {"NC/4", "dY"}, {"NC/5", "dZ"},
				{"NC/6", "tX"}, {"NC/7", "tY"}, {"NC/8", "tZ"},
			})
		}
		if len(st.NODECAPTIONS) >= 15 {
			toggleRow([][2]string{
				{"NC/9", "Rx"}, {"NC/10", "Ry"}, {"NC/11", "Rz"},
				{"NC/12", "Mx"}, {"NC/13", "My"}, {"NC/14", "Mz"},
			})
		}
		for i := 15; i < len(st.NODECAPTIONS); i++ {
			toggle(fmt.Sprintf("NC/%d", i), strings.TrimPrefix(st.NODECAPTIONS[i], "NC_"))
		}
		heading("ELEM CAPTION")
		// Match IUP ELEM CAPTION layout: CODE, SECT, RATE master toggle, and L/S/Q/M row.
		if len(st.ELEMCAPTIONS) >= 2 {
			toggle("EC/0", "CODE")
			toggle("EC/1", "SECT")
		}
		toggle("SRCAN_RATE", "RATE")
		if len(st.SRCANS) > 0 {
			items := make([][2]string, 0, len(st.SRCANS))
			for i, name := range st.SRCANS {
				items = append(items, [2]string{fmt.Sprintf("RATE/%d", i), strings.TrimPrefix(name, "SRCAN_")})
			}
			toggleRow(items)
		}
		heading("ETYPE")
		for _, etype := range []int{st.COLUMN, st.GIRDER, st.BRACE, st.TRUSS} {
			if etype < len(st.ETYPES) {
				toggle(fmt.Sprintf("ETYPE/%d", etype), st.ETYPES[etype])
				stressRow(etype)
			}
		}
		if st.WALL < len(st.ETYPES) && st.WBRACE < len(st.ETYPES) {
			leftToggleRow([][2]string{
				{fmt.Sprintf("ETYPE/%d", st.WALL), st.ETYPES[st.WALL]},
				{fmt.Sprintf("ETYPE/%d", st.WBRACE), st.ETYPES[st.WBRACE]},
			})
			stressRow(st.WBRACE)
		}
		if st.SLAB < len(st.ETYPES) && st.SBRACE < len(st.ETYPES) {
			leftToggleRow([][2]string{
				{fmt.Sprintf("ETYPE/%d", st.SLAB), st.ETYPES[st.SLAB]},
				{fmt.Sprintf("ETYPE/%d", st.SBRACE), st.ETYPES[st.SBRACE]},
			})
			stressRow(st.SBRACE)
		}
		heading("SECTION")
		for _, i := range u.sections {
			key := fmt.Sprintf("SECT/%d", i)
			toggle(key, fmt.Sprintf("%d %s", i, u.values[key]))
		}
	case 2:
		heading("SELECTED")
		for i := 0; i < min(3, len(u.properties)); i++ {
			text := u.properties[i]
			rows = append(rows, func(c layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(4)}.Layout(c, func(c layout.Context) layout.Dimensions {
					return w.uiLabel(c, text)
				})
			})
		}
		heading("PROPERTY")
		if u.propElemCount > 0 {
			if u.propEditable > 0 {
				rows = append(rows, func(c layout.Context) layout.Dimensions { return w.propertySectionEditor(c) })
				if u.propSectLabel != "" {
					sectName := u.propSectLabel
					rows = append(rows, func(c layout.Context) layout.Dimensions {
						return layout.Inset{Left: unit.Dp(74)}.Layout(c, func(c layout.Context) layout.Dimensions {
							s := material.Body2(w.Theme, sectName)
							s.TextSize = unit.Sp(10)
							s.Color = iupFgOff
							s.MaxLines = 1
							return s.Layout(c)
						})
					})
				}
				rows = append(rows, func(c layout.Context) layout.Dimensions { return w.propertyEtypeEditor(c) })
				rows = append(rows, func(c layout.Context) layout.Dimensions { return w.propertyEtypeButtons(c) })
			} else {
				sectVal := u.values["PROP_SECTION"]
				if u.propSectMixed {
					sectVal = u.propSectLabel
				} else if u.propSectLabel != "" && u.propSectLabel != "-" {
					sectVal = strings.TrimSpace(sectVal + " " + u.propSectLabel)
				}
				if sectVal == "" {
					sectVal = "-"
				}
				etypeVal := u.values["PROP_ETYPE"]
				if u.propEtypeMixed {
					etypeVal = "混在"
				} else if etypeVal == "" {
					etypeVal = "-"
				}
				rows = append(rows, func(c layout.Context) layout.Dimensions { return w.propertyReadOnlyRow(c, "SECTION", sectVal) })
				rows = append(rows, func(c layout.Context) layout.Dimensions { return w.propertyReadOnlyRow(c, "ETYPE", etypeVal) })
				note := "選択部材はロックされています"
				if u.propFixedBrace > 0 {
					note = "WBRACE / SBRACE は表示のみ（変更不可）"
				}
				rows = append(rows, func(c layout.Context) layout.Dimensions {
					return layout.Inset{Left: unit.Dp(4)}.Layout(c, func(c layout.Context) layout.Dimensions {
						s := material.Body2(w.Theme, note)
						s.TextSize = unit.Sp(10)
						s.Color = iupFgOff
						return s.Layout(c)
					})
				})
			}
			for i := 3; i < len(u.properties); i++ {
				text := u.properties[i]
				rows = append(rows, func(c layout.Context) layout.Dimensions {
					return layout.Inset{Left: unit.Dp(4)}.Layout(c, func(c layout.Context) layout.Dimensions {
						return w.uiLabel(c, text)
					})
				})
			}
		} else {
			rows = append(rows, func(c layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(4)}.Layout(c, func(c layout.Context) layout.Dimensions {
					s := material.Body2(w.Theme, "部材を選択すると SECTION / ETYPE を編集できます")
					s.TextSize = unit.Sp(10)
					s.Color = iupFgOff
					return s.Layout(c)
				})
			})
		}
		if len(u.propNodeDetails) > 0 {
			heading("NODE")
			for _, text := range u.propNodeDetails {
				text := text
				rows = append(rows, func(c layout.Context) layout.Dimensions {
					return layout.Inset{Left: unit.Dp(4)}.Layout(c, func(c layout.Context) layout.Dimensions {
						return w.uiLabel(c, text)
					})
				})
			}
		}
		rows = append(rows, func(c layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(4)}.Layout(c, func(c layout.Context) layout.Dimensions {
				return w.smallButton(c, "deselect", "Deselect / 選択解除", func() { w.menuAction("deselect") })
			})
		})
		heading("PEN (部材追加)")
		rows = append(rows, func(c layout.Context) layout.Dimensions { return w.penSectionEditor(c, false) })
		if u.penLineSectLabel != "" && u.penLineSectLabel != "-" {
			name := u.penLineSectLabel
			rows = append(rows, func(c layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(74)}.Layout(c, func(c layout.Context) layout.Dimensions {
					s := material.Body2(w.Theme, name)
					s.TextSize = unit.Sp(10)
					s.Color = iupFgOff
					s.MaxLines = 1
					return s.Layout(c)
				})
			})
		}
		rows = append(rows, func(c layout.Context) layout.Dimensions { return w.penEtypeButtons(c, false) })
		rows = append(rows, func(c layout.Context) layout.Dimensions { return w.penSectionEditor(c, true) })
		if u.penPlateSectLabel != "" && u.penPlateSectLabel != "-" {
			name := u.penPlateSectLabel
			rows = append(rows, func(c layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(74)}.Layout(c, func(c layout.Context) layout.Dimensions {
					s := material.Body2(w.Theme, name)
					s.TextSize = unit.Sp(10)
					s.Color = iupFgOff
					s.MaxLines = 1
					return s.Layout(c)
				})
			})
		}
		rows = append(rows, func(c layout.Context) layout.Dimensions { return w.penEtypeButtons(c, true) })
		rows = append(rows, func(c layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(4)}.Layout(c, func(c layout.Context) layout.Dimensions {
				return w.smallButton(c, "pen-pick-selected", "選択部材からペン取得", func() {
					w.enqueue(func() {
						if err := w.pickPenFromSelected(); err != nil {
							w.History(err.Error())
						}
						w.Redraw()
					})
				})
			})
		})
	}
	return rows
}
