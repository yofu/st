package stgio

import (
	"image"
	"math"
	"path/filepath"
	"strings"
	"time"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	st "github.com/yofu/st/stlib"
)

type mouseClickSettings struct {
	delay  time.Duration
	radius f32.Point
}
type mouseState struct {
	settings            mouseClickSettings
	lastTime, pressedAt time.Duration
	lastPosition        f32.Point
	lastButton          pointer.Buttons
	lastID              pointer.ID
	moved, double       bool
	edit, lastEdit      string
	editNode            *st.Node
}

func (w *Window) beginMouseClick(e pointer.Event, button pointer.Buttons) {
	m := &w.mouse
	s := platformClickSettings(w.nativeOwner.Load())
	m.settings = s
	d := e.Position.Sub(m.lastPosition)
	m.double = e.Time > 0 && m.lastTime > 0 && e.Time >= m.lastTime && e.Time-m.lastTime <= s.delay && m.lastButton == button && m.lastID == e.PointerID && math.Abs(float64(d.X)) <= float64(s.radius.X) && math.Abs(float64(d.Y)) <= float64(s.radius.Y)
	m.pressedAt = e.Time
	m.moved = false
	if w.rangeVisible {
		w.rangeVisible = false
		w.sceneDirty.Store(true)
	}
	if !m.double {
		return
	}
	m.lastTime = 0
	if button == pointer.ButtonTertiary {
		if e.Modifiers&key.ModAlt != 0 && w.frame != nil {
			w.ensureRangeView()
			copy(w.rangeView.Angle, w.frame.View.Angle)
		}
		w.showCanvasCenter()
	} else if button == pointer.ButtonPrimary && !w.selectsNodes(e.Modifiers) {
		if w.frame == nil || len(w.frame.Nodes) == 0 {
			w.fileDialog("Open")
			return
		}
		if els := w.SelectedElems(); len(els) > 0 && els[0] != nil {
			if els[0].IsLineElem() {
				w.startMouseEdit("bond")
			} else {
				w.startMouseEdit("plate")
			}
		}
	}
}

func (w *Window) endMouseClick(e pointer.Event, button pointer.Buttons) {
	m := &w.mouse
	if !m.double && !m.moved && m.pressedAt > 0 {
		m.lastTime = m.pressedAt
		m.lastPosition = w.start
		m.lastButton = button
		m.lastID = e.PointerID
	} else {
		m.lastTime = 0
	}
	if button == pointer.ButtonSecondary {
		w.rightMouseRelease(e)
	}
	if button == pointer.ButtonTertiary && w.rangeVisible {
		w.rangeVisible = false
		w.sceneDirty.Store(true)
	}
	if button == pointer.ButtonTertiary && !m.double && (w.frame == nil || len(w.frame.Nodes) == 0) {
		w.popupRecent(e.Position)
	}
}

func (w *Window) showCanvasCenter() {
	if w.frame == nil || len(w.frame.Nodes) == 0 {
		return
	}
	w.Animate(st.CenterView(w, append([]float64(nil), w.frame.View.Angle...)))
}

func (w *Window) rightMouseRelease(e pointer.Event) {
	if w.mouse.edit != "" {
		w.stopMouseEdit()
		w.Deselect()
		w.Redraw()
		return
	}
	if command := strings.TrimSpace(w.ui.command.Text()); command != "" {
		w.submitCommand(command)
		return
	}
	if e.Time-w.mouse.pressedAt < 200*time.Millisecond {
		if e.Modifiers&key.ModShift != 0 {
			if command := w.lastexcommand; command != "" {
				w.enqueueInterrupting(func() { w.ExecCommand(command); w.Redraw() })
			}
		} else if w.mouse.lastEdit != "" {
			w.startMouseEdit(w.mouse.lastEdit)
		} else if c := w.lastcommand; c != nil {
			w.enqueueInterrupting(func() { w.startInteractiveCommand(c); w.Redraw() })
		}
		return
	}
	w.ui.contextPoint = image.Pt(int(e.Position.X), int(e.Position.Y)).Add(w.ui.canvasRect.Min)
	w.ui.menu = -1
	w.ui.menuList.Position = layout.Position{}
	w.ui.contextItems = []menuItem{{"Open...", "Open"}, {"Save", "save"}, {"Save As...", "Save As"}, {"Fit / 全体表示", "fit"}, {"Undo", "cmd::undo"}, {"Deselect", "deselect"}, {"Show All", "showall"}, {"Hide Selected", "hide"}, {"Selected Properties", "tab:2"}}
}

func (w *Window) popupRecent(position f32.Point) {
	w.ui.menu = -1
	w.ui.contextPoint = image.Pt(int(position.X), int(position.Y)).Add(w.ui.canvasRect.Min)
	w.ui.contextItems = nil
	w.ui.menuList.Position = layout.Position{}
	for _, p := range w.Recent() {
		if p != "" {
			w.ui.contextItems = append(w.ui.contextItems, menuItem{filepath.Base(p), "recent:" + p})
		}
	}
}

func (w *Window) selectsNodes(modifiers key.Modifiers) bool {
	return (modifiers&key.ModAlt != 0) == w.AltSelectNode()
}

func (w *Window) startMouseEdit(mode string) {
	w.mouse.edit = mode
	w.mouse.lastEdit = mode
	w.mouse.editNode = nil
	if mode == "bond" {
		w.History("TOGGLEBOND: 端部節点をクリックしてください。右クリック/Escで終了。")
	} else {
		w.History("EDITPLATEELEM: 変更元節点、変更先節点の順にクリックしてください。右クリック/Escで終了。")
	}
}
func (w *Window) stopMouseEdit() { w.mouse.edit = ""; w.mouse.editNode = nil }

func (w *Window) mouseEditClick() {
	n := w.frame.PickNode(float64(w.end.X), float64(w.end.Y), 15)
	if n == nil {
		return
	}
	changed := false
	if w.mouse.edit == "bond" {
		for _, el := range w.SelectedElems() {
			if el != nil && el.IsLineElem() {
				if err := el.ToggleBond(n.Num); err == nil {
					changed = true
				}
			}
		}
	} else if w.mouse.editNode == nil {
		w.mouse.editNode = n
		return
	} else {
		old := w.mouse.editNode
		if n == old {
			return
		}
		for _, el := range w.SelectedElems() {
			if el != nil && !el.IsLineElem() {
				for i, en := range el.Enod {
					if en == old {
						el.Enod[i] = n
						changed = true
					}
				}
			}
		}
		w.mouse.editNode = nil
	}
	if changed {
		w.changed = true
		st.Snapshot(w)
		w.Redraw()
	}
}

func (w *Window) ensureRangeView() {
	if w.rangeView == nil {
		w.rangeView = st.NewView()
		w.rangeView.Dists = []float64{1000, 3000}
		w.rangeView.Angle = []float64{20, 225}
		w.rangeView.Center = []float64{100, 100}
	}
}

func (w *Window) drawMouseRange() {
	if !w.rangeVisible || w.frame == nil || len(w.frame.Nodes) == 0 {
		return
	}
	w.ensureRangeView()
	v := w.rangeView
	x0, x1, y0, y1, z0, z1 := w.frame.Bbox(false)
	v.Focus = []float64{(x0 + x1) / 2, (y0 + y1) / 2, (z0 + z1) / 2}
	v.Set(w.CanvasDirection())
	points := make([][]float64, 8)
	for i := 0; i < 8; i++ {
		p := []float64{x0, y0, z0}
		if i&1 != 0 {
			p[0] = x1
		}
		if i&2 != 0 {
			p[1] = y1
		}
		if i&4 != 0 {
			p[2] = z1
		}
		points[i] = v.ProjectCoord(p)
	}
	w.Foreground(st.DARK_GREEN)
	for i := 0; i < 8; i++ {
		for bit := 1; bit <= 4; bit *= 2 {
			if i&bit == 0 {
				j := i | bit
				w.Line(points[i][0], points[i][1], points[j][0], points[j][1])
			}
		}
	}
}
