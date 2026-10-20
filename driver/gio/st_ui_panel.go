package stgio

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"gioui.org/layout"
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
	if name == "PERIOD" {
		p := strings.ToUpper(strings.TrimSpace(text))
		if p == "" {
			return fmt.Errorf("PERIOD is empty")
		}
		st.SetPeriod(w, p)
		return nil
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
			if value {
				s.SrcanRate |= 1 << i
			} else {
				s.SrcanRate &^= 1 << i
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
	u.properties = []string{fmt.Sprintf("NODES: %d", len(w.SelectedNodes())), fmt.Sprintf("ELEMENTS: %d", len(w.SelectedElems()))}
	var length, area float64
	lines, plates := 0, 0
	for _, e := range w.SelectedElems() {
		if e == nil {
			continue
		}
		if e.Enods == 2 {
			lines++
			length += e.Length()
		} else {
			plates++
			area += e.Area()
		}
	}
	u.properties = append(u.properties, fmt.Sprintf("LINE: %d   LENGTH: %.4f m", lines, length), fmt.Sprintf("PLATE: %d   AREA: %.4f m²", plates, area))
	if els := w.SelectedElems(); len(els) > 0 && els[0] != nil {
		e := els[0]
		u.properties = append(u.properties, fmt.Sprintf("CODE: %d", e.Num))
		if e.Sect != nil {
			u.properties = append(u.properties, fmt.Sprintf("SECTION: %d %s", e.Sect.Num, e.Sect.Name))
		}
		if e.Etype >= 0 && e.Etype < len(st.ETYPES) {
			u.properties = append(u.properties, fmt.Sprintf("ETYPE: %s", st.ETYPES[e.Etype]))
		}
		for _, n := range e.Enod {
			u.properties = append(u.properties, fmt.Sprintf("NODE %d: %.3f, %.3f, %.3f", n.Num, n.Coord[0], n.Coord[1], n.Coord[2]))
		}
	}
	if ns := w.SelectedNodes(); len(ns) > 0 && ns[0] != nil {
		n := ns[0]
		u.properties = append(u.properties, fmt.Sprintf("NODE: %d", n.Num), fmt.Sprintf("XYZ: %.4f, %.4f, %.4f", n.Coord[0], n.Coord[1], n.Coord[2]))
	}
	u.status = fmt.Sprintf("%s | %d nodes / %d members | %s", f.Name, len(f.Nodes), len(f.Elems), f.Show.Period)
}

func (w *Window) settingEditor(gtx layout.Context, name string) layout.Dimensions {
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
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(c layout.Context) layout.Dimensions { c.Constraints.Min.X = c.Dp(92); return w.uiLabel(c, name) }),
		layout.Flexed(1, func(c layout.Context) layout.Dimensions {
			e := material.Editor(w.Theme, &field.ed, "")
			e.TextSize = unit.Sp(12)
			return e.Layout(c)
		}))
}

func (w *Window) checkControl(gtx layout.Context, key, label string) layout.Dimensions {
	u := w.ui
	b := u.checks[key]
	if b == nil {
		b = new(widget.Bool)
		u.checks[key] = b
	}
	if b.Update(gtx) {
		value := b.Value
		w.enqueue(func() { w.applyFlag(key, value); w.Redraw() })
	} else {
		b.Value = u.flags[key]
	}
	style := material.CheckBox(w.Theme, b, label)
	style.TextSize = unit.Sp(12)
	style.Size = unit.Dp(18)
	return style.Layout(gtx)
}

func (w *Window) layoutPanel(gtx layout.Context) layout.Dimensions {
	u := w.ui
	return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(c layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(c,
			layout.Rigid(func(c layout.Context) layout.Dimensions {
				children := []layout.FlexChild{}
				for i, title := range []string{"View", "Show", "Property"} {
					i, title := i, title
					children = append(children, layout.Rigid(func(c layout.Context) layout.Dimensions {
						label := title
						if u.tab == i {
							label = "• " + title
						}
						return w.smallButton(c, "tab-"+title, label, func() { u.tab = i })
					}))
				}
				return layout.Flex{}.Layout(c, children...)
			}),
			layout.Flexed(1, func(c layout.Context) layout.Dimensions {
				rows := w.panelRows()
				return material.List(w.Theme, &u.panelList[u.tab]).Layout(c, len(rows), func(c layout.Context, i int) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4)}.Layout(c, rows[i])
				})
			}))
	})
}

func (w *Window) panelRows() []layout.Widget {
	u := w.ui
	rows := []layout.Widget{}
	heading := func(label string) {
		rows = append(rows, func(c layout.Context) layout.Dimensions { return w.uiLabel(c, "── "+label+" ──") })
	}
	edit := func(name string) {
		rows = append(rows, func(c layout.Context) layout.Dimensions { return w.settingEditor(c, name) })
	}
	check := func(key, label string) {
		rows = append(rows, func(c layout.Context) layout.Dimensions { return w.checkControl(c, key, label) })
	}
	switch u.tab {
	case 0:
		heading("VIEW")
		rows = append(rows, func(c layout.Context) layout.Dimensions {
			return w.smallButton(c, "fit-panel", "Fit / 全体表示", func() { w.menuAction("fit") })
		})
		check("PERSPECTIVE", "Perspective / 透視投影")
		for _, name := range viewFields {
			edit(name)
		}
	case 1:
		heading("COLOR MODE")
		rows = append(rows, func(c layout.Context) layout.Dimensions {
			return w.smallButton(c, "color-mode", u.values["COLOR"], func() { w.menuAction("color") })
		})
		heading("PARAMETER")
		for _, name := range parameterFields {
			edit(name)
		}
		heading("SHOW")
		for _, name := range showFlags {
			check(name, name)
		}
		heading("NODE CAPTION")
		for i, name := range st.NODECAPTIONS {
			check(fmt.Sprintf("NC/%d", i), name)
		}
		heading("ELEM CAPTION")
		for i, name := range st.ELEMCAPTIONS {
			check(fmt.Sprintf("EC/%d", i), name)
		}
		heading("ETYPE / STRESS")
		for i, name := range st.SRCANS {
			check(fmt.Sprintf("RATE/%d", i), name)
		}
		for i, name := range st.ETYPES {
			i, name := i, name
			check(fmt.Sprintf("ETYPE/%d", i), name)
			rows = append(rows, func(c layout.Context) layout.Dimensions {
				children := []layout.FlexChild{}
				for j, label := range []string{"N", "Qx", "Qy", "Mz", "Mx", "My"} {
					j, label := j, label
					children = append(children, layout.Flexed(1, func(c layout.Context) layout.Dimensions {
						return w.checkControl(c, fmt.Sprintf("STRESS/%d/%d", i, j), label)
					}))
				}
				return layout.Flex{}.Layout(c, children...)
			})
		}
		heading("SECTION")
		for _, i := range u.sections {
			key := fmt.Sprintf("SECT/%d", i)
			check(key, fmt.Sprintf("%d %s", i, u.values[key]))
		}
	case 2:
		heading("SELECTED")
		for _, text := range u.properties {
			text := text
			rows = append(rows, func(c layout.Context) layout.Dimensions { return w.uiLabel(c, text) })
		}
		rows = append(rows, func(c layout.Context) layout.Dimensions {
			return w.smallButton(c, "deselect", "Deselect / 選択解除", func() { w.menuAction("deselect") })
		})
	}
	return rows
}
