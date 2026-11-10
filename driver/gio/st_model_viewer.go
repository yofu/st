package stgio

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path"
	"strings"
	"unicode/utf8"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/font/gofont"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	st "github.com/yofu/st/stlib"
	"golang.org/x/text/encoding/japanese"
)

// ViewerModelLimit bounds the size of a single local file loaded by the viewer.
const ViewerModelLimit = 32 << 20

type ViewerState struct {
	Name                           string
	Nodes, Members                 int
	SelectedNodes, SelectedMembers int
	Editing, Modified              bool
	History, Progress, Annotations string
	Question                       string
	PromptID                       int
	PromptTitle, PromptInitial     string
}
type modelViewer struct {
	fingers                 []viewerFinger
	gesture                 bool
	pickNodes               bool
	scale                   float64
	zoom                    float64
	report                  func(ViewerState)
	last                    ViewerState
	promptID                int
	promptReplies           chan viewerPromptReply
	browserAction           func(string, []string)
	question                *uiQuestion // protected by ui.mu
	questionID              int
	eps                     float64
	workerJob, workerSerial int
	startWorker             func(int, []byte)
	stopWorker              func(int)
}
type viewerFinger struct {
	id          pointer.ID
	start, last f32.Point
}

// NewModelViewer reuses Gio's ST canvas without exposing desktop-only controls.
// The browser supplies its own toolbar for file access and touch mode selection.
func NewModelViewer(window *app.Window, report func(ViewerState)) *Window {
	w := NewWindow(window)
	w.viewer = &modelViewer{scale: 1, zoom: 1, eps: 1e-3, report: report, promptReplies: make(chan viewerPromptReply, 1)}
	if browserViewer {
		w.SetHome("/workspace")
		w.SetCwd("/workspace")
		w.Theme.Shaper = text.NewShaper(text.NoSystemFonts(), text.WithCollection(gofont.Collection()))
	}
	return w
}

func parseViewerModel(name string, data []byte) (frame *st.Frame, err error) {
	name = path.Base(strings.ReplaceAll(name, `\`, "/"))
	if !strings.EqualFold(path.Ext(name), ".inp") {
		return nil, fmt.Errorf(".inp ファイルを選択してください")
	}
	if len(data) == 0 || len(data) > ViewerModelLimit {
		return nil, fmt.Errorf("ファイルが空か、32 MiBを超えています")
	}
	// Some legacy parser paths assume the next INP field exists. A malformed
	// file must not destroy the viewer or replace the currently displayed frame.
	defer func() {
		if p := recover(); p != nil {
			frame = nil
			err = fmt.Errorf("INPの書式を確認してください: %v", p)
		}
	}()
	if !utf8.Valid(data) {
		decoded, e := japanese.ShiftJIS.NewDecoder().Bytes(data)
		if e != nil {
			return nil, e
		}
		data = decoded
	}
	frame = st.NewFrame()
	if err = frame.ReadInpString(strings.TrimPrefix(string(data), "\ufeff"), name, nil, 0, false); err != nil {
		return nil, err
	}
	if len(frame.Nodes) == 0 {
		return nil, fmt.Errorf("節点が見つかりません")
	}
	for _, n := range frame.Nodes {
		for _, v := range n.Coord {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return nil, fmt.Errorf("有限でない節点座標があります")
			}
		}
	}
	for _, el := range frame.Elems {
		if el.Sect == nil || len(el.Enod) < 2 || len(el.Enod) != el.Enods {
			return nil, fmt.Errorf("部材 %d の断面または節点が無効です", el.Num)
		}
		// Legacy INP permits omitted BONDS; the shared renderer expects
		// six end-condition slots per end even when all are rigid.
		if el.IsLineElem() {
			if len(el.Bonds) == 0 {
				el.Bonds = make([]*st.Bond, 12)
			}
			if len(el.Cmq) == 0 {
				el.Cmq = make([]float64, 12)
			}
		}
		for _, n := range el.Enod {
			if n == nil {
				return nil, fmt.Errorf("部材 %d に未定義の節点があります", el.Num)
			}
		}
	}
	// The prototype uses orthographic projection for predictable touch zoom.
	frame.View.Perspective = false
	if math.IsNaN(frame.View.Gfact) || math.IsInf(frame.View.Gfact, 0) || frame.View.Gfact <= 0 {
		frame.View.Gfact = 1
	}
	for i, a := range frame.View.Angle {
		if math.IsNaN(a) || math.IsInf(a, 0) {
			frame.View.Angle[i] = 0
		}
	}
	// The toolbar renders filenames in HTML, so a browser system font is used.
	frame.Name = name
	return frame, nil
}

// OpenViewerModel reads bytes already selected by the user in the browser.
// It commits only a successfully parsed model; done runs after the attempt.
func (w *Window) OpenViewerModel(name string, data []byte, done func(error)) bool {
	if w.viewer == nil {
		return false
	}
	return w.tryEnqueueJob(uiJob{interrupt: true, run: func() {
		if w.viewerAnalysisRunning() {
			if done != nil {
				done(fmt.Errorf("解析中です。完了または停止してからモデルを開いてください"))
			}
			return
		}
		f, err := parseViewerModel(name, data)
		if err == nil && browserViewer {
			f.Path = path.Join("/workspace", f.Name)
			err = os.WriteFile(f.Path, data, 0644)
		}
		if err == nil {
			w.SetFrame(f)
			clear(w.textBox)
			if browserViewer {
				w.SetCwd("/workspace")
				w.AddRecent(f.Path)
			}
			w.UndoStack = st.NewUndoStack(10)
			w.PushUndo(f)
			w.Changed(false)
			w.History("OPEN: " + f.Name)
			w.Deselect()
			w.viewer.fingers = nil
			w.fitViewer()
			w.Redraw()
		}
		if done != nil {
			done(err)
		}
	}})
}

func (w *Window) viewerDisplayAction(action string) bool {
	if w.viewer == nil {
		return false
	}
	switch action {
	case "node", "member", "fit", "clear":
	default:
		return false
	}
	return w.tryEnqueueJob(uiJob{run: func() {
		switch action {
		case "node":
			w.viewer.pickNodes = true
		case "member":
			w.viewer.pickNodes = false
		case "fit":
			w.fitViewer()
		case "clear":
			w.Deselect()
		}
		w.Redraw()
	}})
}

func (w *Window) fitViewer() {
	if w.canvasSize.X <= 0 || w.canvasSize.Y <= 0 || w.frame == nil || len(w.frame.Nodes) == 0 {
		return
	}
	view := w.frame.View
	w.frame.View = st.CenterView(w, view.Angle)
	if w.frame.View == view { // a single projected point or zero projected span
		view.Set(w.CanvasDirection())
		for _, n := range w.frame.Nodes {
			p := view.ProjectCoord(n.Coord)
			view.Center[0] += float64(w.canvasSize.X)/2 - p[0]
			view.Center[1] += float64(w.canvasSize.Y)/2 - p[1]
			break
		}
	}
	w.viewer.zoom = 1
}

func (w *Window) layoutModelViewer(gtx layout.Context) layout.Dimensions {
	u := w.ui
	size := gtx.Constraints.Max
	w.updateViewerPrompt()
	w.collectCanvasInput(gtx, image.Rectangle{Max: size})
	u.mu.Lock()
	log, progress, question := u.log, u.progress, u.question
	if question != w.viewer.question {
		w.viewer.question = question
		w.viewer.questionID++
	}
	questionID := w.viewer.questionID
	u.mu.Unlock()
	current := w.viewer.last
	current.History, current.Progress = log, progress
	if u.modelMu.TryLock() {
		oldSize := w.canvasSize
		w.prepareCanvas(gtx)
		if oldSize != size {
			if oldSize == (image.Point{}) {
				w.fitViewer()
			} else {
				// The keyboard/history can resize the canvas during an edit.
				// Keep zoom and shift its center with the viewport.
				center := w.frame.View.Center
				center[0] += float64(size.X-oldSize.X) / 2
				center[1] += float64(size.Y-oldSize.Y) / 2
			}
			w.viewer.fingers = nil
		}
		w.viewer.scale = math.Max(1, float64(gtx.Dp(100))/100)
		w.applyCanvasInput()
		w.advanceViewAnimation(gtx.Now)
		if w.sceneDirty.Swap(false) || !u.canvasReady {
			u.canvasOps.Reset()
			macro := op.Record(&u.canvasOps)
			scene := gtx
			scene.Ops = &u.canvasOps
			w.drawCanvas(scene)
			for _, n := range w.SelectedNodes() {
				if n == nil || n.IsHidden(w.frame.Show) {
					continue
				}
				w.currentBrush = color.NRGBA{R: 30, G: 110, B: 230, A: 255}
				w.FilledCircle(n.Pcoord[0], n.Pcoord[1], 5*w.viewer.scale)
			}
			u.canvasCall = macro.Stop()
			u.canvasReady = true
		}
		w.snapshotCommandTail()
		f := w.frame
		current = ViewerState{Name: f.Name, Nodes: len(f.Nodes), Members: len(f.Elems), SelectedNodes: len(w.SelectedNodes()), SelectedMembers: len(w.SelectedElems()), Editing: w.Executing(), Modified: w.IsChanged(), History: log, Progress: progress, Annotations: w.viewerAnnotations()}
		if d := u.modal; d != nil {
			current.PromptID, current.PromptTitle, current.PromptInitial = w.viewer.promptID, d.title, d.ed.Text()
		}
		u.modelMu.Unlock()
	}
	if u.canvasReady {
		u.canvasCall.Add(gtx.Ops)
	}
	w.drawCanvasOverlay(gtx)
	w.scheduleViewAnimation(gtx)
	current.Question = ""
	if question != nil {
		current.Question = viewerQuestionJSON(questionID, question)
	}
	if current != w.viewer.last {
		w.viewer.last = current
		if w.viewer.report != nil {
			w.viewer.report(current)
		}
	}
	return layout.Dimensions{Size: size}
}

func (w *Window) handleViewerPointer(e pointer.Event) {
	v := w.viewer
	if w.ui.modal != nil || v.question != nil {
		v.fingers = nil
		return
	}
	if e.Kind == pointer.Cancel {
		v.fingers = nil
		v.gesture = false
		return
	}
	if e.Kind == pointer.Scroll && len(v.fingers) == 0 && e.Scroll.Y != 0 {
		w.viewerZoom(math.Exp(-float64(e.Scroll.Y)/(160*v.scale)), e.Position, e.Position)
		return
	}
	index := -1
	for i, f := range v.fingers {
		if f.id == e.PointerID {
			index = i
			break
		}
	}
	switch e.Kind {
	case pointer.Press:
		if index >= 0 || (e.Source == pointer.Mouse && e.Buttons&pointer.ButtonPrimary == 0) {
			return
		}
		if len(v.fingers) == 0 {
			v.gesture = false
		}
		v.fingers = append(v.fingers, viewerFinger{id: e.PointerID, start: e.Position, last: e.Position})
		if len(v.fingers) > 1 {
			v.gesture = true
		}
	case pointer.Move, pointer.Drag, pointer.Release:
		if index < 0 {
			return
		}
		w.moveViewerFinger(index, e.Position)
		if e.Kind == pointer.Release {
			tap := len(v.fingers) == 1 && !v.gesture
			v.fingers = append(v.fingers[:index], v.fingers[index+1:]...)
			if tap {
				w.pickViewer(e.Position)
			}
		}
	}
}

func viewerDistance(p f32.Point) float64 { return math.Hypot(float64(p.X), float64(p.Y)) }
func (w *Window) moveViewerFinger(i int, pos f32.Point) {
	v := w.viewer
	f := &v.fingers[i]
	if len(v.fingers) == 1 {
		if !v.gesture && viewerDistance(pos.Sub(f.start)) < 6*v.scale {
			return
		}
		d := pos.Sub(f.last)
		f.last = pos
		v.gesture = true
		if d != (f32.Point{}) && !w.fixRotate && w.frame != nil && len(w.frame.Nodes) > 0 {
			angle := w.frame.View.Angle
			angle[0] = math.Max(-90, math.Min(90, angle[0]+float64(d.Y)*0.35/v.scale))
			angle[1] = math.Remainder(angle[1]-float64(d.X)*0.35/v.scale, 360)
			w.sceneDirty.Store(true)
		}
	} else if len(v.fingers) == 2 {
		other := v.fingers[1-i].last
		old := f.last
		f.last = pos
		before, after := viewerDistance(old.Sub(other)), viewerDistance(pos.Sub(other))
		factor := 1.0
		if before > 4*v.scale && after > 4*v.scale {
			factor = after / before
		}
		w.viewerZoom(factor, old.Add(other).Mul(.5), pos.Add(other).Mul(.5))
	} else {
		f.last = pos
	}
}

func (w *Window) viewerZoom(factor float64, from, to f32.Point) {
	v := w.viewer
	if w.frame == nil || len(w.frame.Nodes) == 0 || factor <= 0 || math.IsNaN(factor) {
		return
	}
	zoom := math.Max(.05, math.Min(100, v.zoom*factor))
	w.WindowZoom(zoom/v.zoom, float64(from.X), float64(from.Y))
	v.zoom = zoom
	if !w.fixMove {
		center := w.frame.View.Center
		center[0] += float64(to.X - from.X)
		center[1] += float64(to.Y - from.Y)
	}
	w.sceneDirty.Store(true)
}

func (w *Window) pickViewer(pos f32.Point) {
	f := w.frame
	if f == nil || len(f.Nodes) == 0 {
		return
	}
	// A gesture and a tap can be queued before the next paint.
	f.View.Set(w.CanvasDirection())
	for _, n := range f.Nodes {
		f.View.ProjectNode(n)
	}
	if w.interactiveClick(pointer.Event{Position: pos}, pointer.ButtonPrimary) {
		return
	}
	x, y := float64(pos.X), float64(pos.Y)
	if w.viewer.pickNodes {
		n := f.PickNode(x, y, 14*w.viewer.scale)
		if n != nil && f.Nodes[n.Num] == n {
			st.MergeSelectNode(w, []*st.Node{n}, false)
		}
	} else if e := f.PickElem(x, y, 10*w.viewer.scale); e != nil {
		st.MergeSelectElem(w, []*st.Elem{e}, false)
	}
	w.sceneDirty.Store(true)
}
