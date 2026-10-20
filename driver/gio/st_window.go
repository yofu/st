package stgio

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/yofu/abbrev"
	st "github.com/yofu/st/stlib"
)

var (
	altselectnode  = true
	tailnodes      []*st.Node
	showprintrange = false
	PLATE_OPACITY  = uint8(0x55)
)

type Window struct {
	nativeOwner      atomic.Uintptr
	filePicker       func(fileDialogOptions) (string, bool, error)
	pointerPosition  func() (f32.Point, bool)
	historyDirectory string
	historyReadError error
	recentReadError  error
	persistenceMu    sync.Mutex
	recentMu         sync.Mutex
	ui               *userInterface
	*st.DrawOption
	*st.Directory
	*st.RecentFiles
	*st.UndoStack
	*st.TagFrame
	*st.Selection
	*st.CommandBuffer
	*st.CommandLine
	*st.Alias
	frame                          *st.Frame
	Theme                          *material.Theme
	window                         *app.Window
	context                        layout.Context
	canvasSize                     image.Point
	pointerTag                     struct{}
	start, end                     f32.Point
	dragLast                       f32.Point
	dragID                         pointer.ID
	canvasEvents                   []event.Event
	mouse                          mouseState
	rangeView                      *st.View
	rangeVisible                   bool
	fixRotate, fixMove             bool
	animation                      *viewAnimation
	viewAnimating                  atomic.Bool
	commandDraft                   atomic.Pointer[commandDraft]
	interaction                    atomic.Pointer[interactiveCommand]
	interactionFocus, commandFocus atomic.Bool
	press                          pointer.Buttons
	selected                       image.Rectangle
	selecting                      bool
	sceneDirty                     atomic.Bool
	textAngle                      float64
	// history         *Dialog
	currentPen      color.NRGBA
	currentBrush    color.NRGBA
	papersize       uint
	changed         bool
	lastexcommand   string
	lastfig2command string
	lastcommand     func(st.Commander) chan bool
	textBox         map[string]*st.TextBox
	textAlignment   int
	// glasses         map[string]*Glass
}

func NewWindow(w *app.Window) *Window {
	return &Window{
		ui:            newUserInterface(),
		DrawOption:    st.NewDrawOption(),
		Directory:     st.NewDirectory("", ""),
		RecentFiles:   st.NewRecentFiles(3),
		UndoStack:     st.NewUndoStack(10),
		TagFrame:      st.NewTagFrame(),
		Selection:     st.NewSelection(),
		CommandBuffer: st.NewCommandBuffer(),
		CommandLine:   st.NewCommandLine(),
		Alias:         newWindowAlias(),
		frame:         st.NewFrame(),
		// Theme:         material.NewTheme(gofont.Collection()),
		Theme:  material.NewTheme(),
		window: w,
		// history:         nil,
		// currentStrokeParam: &ui.DrawStrokeParams{
		// 	Cap:        ui.DrawLineCapFlat,
		// 	Join:       ui.DrawLineJoinMiter,
		// 	Thickness:  LINE_THICKNESS,
		// 	MiterLimit: ui.DrawDefaultMiterLimit,
		// },
		currentPen:      color.NRGBA{R: 0, G: 0, B: 0, A: 0xff},
		currentBrush:    color.NRGBA{R: 0, G: 0, B: 0, A: 0xff},
		papersize:       st.A4_TATE,
		changed:         false,
		lastexcommand:   "",
		lastfig2command: "",
		lastcommand:     nil,
		textBox:         make(map[string]*st.TextBox),
		textAlignment:   st.SOUTH_WEST,
		// glasses:         make(map[string]*Glass),
	}
}

func (stw *Window) QueueRedrawAll() {
	if stw.window != nil {
		stw.window.Invalidate()
	}
}

// Run owns the Gio event loop. Input is delivered through layout.Context.Event,
// rather than as top-level Window.Event values.
func (stw *Window) Run(w *app.Window) error {
	stw.window = w
	var ops op.Ops
	for {
		ev := w.Event()
		stw.platformWindowEvent(ev)
		switch e := ev.(type) {
		case app.DestroyEvent:
			if s := stw.interaction.Load(); s != nil {
				s.cancelCommand()
			}
			if err := stw.SaveCommandHistory(""); err != nil {
				stw.History(err.Error())
			}
			stw.ui.close()
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			stw.Layout(gtx)
			e.Frame(gtx.Ops)
		}
	}
}

func (stw *Window) prepareCanvas(gtx layout.Context) {
	stw.context = gtx
	previousSize := stw.canvasSize
	stw.canvasSize = gtx.Constraints.Max
	if previousSize != stw.canvasSize {
		if previousSize == (image.Point{}) && stw.animation != nil && stw.animation.owner == stw.frame {
			// A view requested before the first frame must survive initial sizing.
			stw.frame.View = stw.animation.to.Copy()
		}
		stw.cancelViewAnimation()
		stw.sceneDirty.Store(true)
	}
	if previousSize == (image.Point{}) && stw.frame != nil && len(stw.frame.Nodes) > 0 {
		stw.frame.View = st.CenterView(stw, stw.frame.View.Angle)
	}
}

// Receive and register input every frame, even while the model worker is busy.
// Model changes are applied later, with modelMu held by Layout.
func (stw *Window) collectCanvasInput(gtx layout.Context, bounds image.Rectangle) {
	// Command replacement can complete without any subsequent pointer event.
	if stw.ui.fenceOverlay && !stw.fenceCommandActive() {
		stw.ui.fenceOverlay = false
		stw.ui.selectionOverlay = false
	}

	var events []event.Event
	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target:  &stw.pointerTag,
			Kinds:   pointer.Press | pointer.Release | pointer.Move | pointer.Drag | pointer.Scroll | pointer.Cancel,
			ScrollY: pointer.ScrollRange{Min: -1000000, Max: 1000000},
		}, key.Filter{Name: key.NameEscape}, key.FocusFilter{Target: &stw.pointerTag})
		if !ok {
			break
		}
		switch ev := ev.(type) {
		case pointer.Event:
			// This tag is registered in window coordinates, without a region
			// transform. Convert to canvas pixels exactly once for every kind.
			ev.Position = ev.Position.Sub(f32.Pt(float32(bounds.Min.X), float32(bounds.Min.Y)))
			if ev.Kind == pointer.Press {
				target := any(&stw.ui.command)
				if stw.interaction.Load() != nil {
					target = &stw.pointerTag
				}
				gtx.Execute(key.FocusCmd{Tag: target})
				if ev.Buttons&(pointer.ButtonPrimary|pointer.ButtonTertiary|pointer.ButtonSecondary) != 0 {
					gtx.Execute(pointer.GrabCmd{Tag: &stw.pointerTag, ID: ev.PointerID})
				}
			}
			events = append(events, ev)
		default:
			events = append(events, ev)
		}
	}
	// Reconcile the final mouse sample with the actual client-area cursor.
	// This avoids a persistent endpoint offset under Windows DPI virtualization.
	// Press retains its event-time origin; touch/pen also retain event positions.
	if n := len(events); n > 0 {
		if e, ok := events[n-1].(pointer.Event); ok && e.Source == pointer.Mouse && (e.Kind == pointer.Drag || e.Kind == pointer.Release) {
			position, valid := stw.clientPointerPosition()
			if valid {
				e.Position = position.Sub(f32.Pt(float32(bounds.Min.X), float32(bounds.Min.Y)))
				events[n-1] = e
			}
		}
	}
	for _, e := range events {
		if p, ok := e.(pointer.Event); ok && p.Kind != pointer.Cancel {
			stw.ui.cursorPosition = p.Position
		}
		stw.ui.fenceOverlay = stw.fenceCommandActive()
		if stw.interaction.Load() == nil || stw.ui.fenceOverlay {
			stw.ui.updateSelectionOverlay(e)
		} else {
			stw.ui.selectionOverlay = false
		}
		// Coalesce only consecutive motion with the same pointer/modifiers.
		if e, ok := e.(pointer.Event); ok && (e.Kind == pointer.Move || e.Kind == pointer.Drag) {
			if n := len(stw.canvasEvents); n > 0 {
				if prev, ok := stw.canvasEvents[n-1].(pointer.Event); ok && prev.Kind == e.Kind && prev.Source == e.Source && prev.PointerID == e.PointerID && prev.Buttons == e.Buttons && prev.Modifiers == e.Modifiers {
					stw.canvasEvents[n-1] = e
					continue
				}
			}
		}
		stw.canvasEvents = append(stw.canvasEvents, e)
	}
	area := clip.Rect(bounds).Push(gtx.Ops)
	pointer.CursorCrosshair.Add(gtx.Ops)
	event.Op(gtx.Ops, &stw.pointerTag)
	area.Pop()
}

func (stw *Window) clientPointerPosition() (f32.Point, bool) {
	if stw.pointerPosition != nil {
		return stw.pointerPosition()
	}
	return platformPointerPosition(stw.nativeOwner.Load())
}

func (stw *Window) applyCanvasInput() {
	for _, ev := range stw.canvasEvents {
		switch e := ev.(type) {
		case pointer.Event:
			stw.handlePointer(e)
		case key.Event:
			if e.State == key.Press {
				stw.cancelCanvasDrag()
				stw.stopMouseEdit()
				stw.cancelViewAnimation()
			}
		}
	}
	clear(stw.canvasEvents)
	stw.canvasEvents = stw.canvasEvents[:0]
}

func (stw *Window) canvasInput(gtx layout.Context) {
	stw.collectCanvasInput(gtx, image.Rectangle{Max: gtx.Constraints.Max})
	stw.applyCanvasInput()
}

func (stw *Window) drawCanvas(gtx layout.Context) {
	stw.context = gtx
	area := clip.Rect{Max: stw.canvasSize}.Push(gtx.Ops)
	defer area.Pop()
	paint.Fill(gtx.Ops, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	if stw.frame != nil {
		// Use the same geometry, deformation and visibility rules during a drag
		// as before/after it. DrawFrameNode omits parts of the normal drawing.
		st.DrawFrame(stw, stw.frame, stw.frame.Show.ColorMode, false)
		stw.drawSelection()
		stw.drawMouseRange()
	}
}

func (stw *Window) layoutCanvas(gtx layout.Context) layout.Dimensions {
	stw.prepareCanvas(gtx)
	stw.interactiveKeyboard(gtx)
	stw.canvasInput(gtx)
	stw.advanceViewAnimation(gtx.Now)
	stw.drawCanvas(gtx)
	stw.drawCanvasOverlay(gtx)
	stw.scheduleViewAnimation(gtx)
	return layout.Dimensions{Size: stw.canvasSize}
}

func (stw *Window) handlePointer(e pointer.Event) {
	switch e.Kind {
	case pointer.Press:
		// Additional buttons or another pointer must not replace the drag origin.
		if stw.press != 0 {
			return
		}
		stw.cancelViewAnimation()
		buttons := e.Buttons & (pointer.ButtonPrimary | pointer.ButtonTertiary | pointer.ButtonSecondary)
		if buttons == 0 {
			return
		}
		stw.start = e.Position
		stw.end = e.Position
		stw.dragLast = e.Position
		stw.dragID = e.PointerID
		stw.press = buttons
		if stw.interaction.Load() == nil || buttons&pointer.ButtonTertiary != 0 {
			stw.beginMouseClick(e, buttons)
		} else {
			stw.mouse.double = false
		}
		if e.Buttons&pointer.ButtonTertiary == 0 && e.Buttons&pointer.ButtonPrimary != 0 {
			stw.selecting = true
			stw.selected = image.Rect(int(e.Position.X), int(e.Position.Y), int(e.Position.X), int(e.Position.Y))
		}
	case pointer.Drag:
		stw.interactivePosition(e.Position)
		if stw.press != 0 && e.PointerID == stw.dragID {
			stw.updateCanvasDrag(e.Position, e.Modifiers)
		}
	case pointer.Move:
		stw.interactivePosition(e.Position)
		if stw.press == 0 {
			stw.end = e.Position
		}
	case pointer.Release:
		if stw.press == 0 || e.PointerID != stw.dragID {
			return
		}
		stw.updateCanvasDrag(e.Position, e.Modifiers)
		if e.Buttons&stw.press != 0 {
			return
		}
		handled := stw.interactiveClick(e, stw.press)
		if stw.selecting && !stw.mouse.double && !handled {
			if stw.mouse.edit != "" {
				stw.mouseEditClick()
			} else {
				stw.finishSelection(e.Modifiers)
			}
			stw.sceneDirty.Store(true)
		}
		if !handled {
			stw.endMouseClick(e, stw.press)
		}
		stw.cancelCanvasDrag()
	case pointer.Cancel:
		// Cancel has no meaningful Position; keep the last real cursor position.
		stw.cancelCanvasDrag()
		stw.mouse.lastTime = 0
	case pointer.Scroll:
		if stw.press != 0 {
			return
		}
		stw.cancelViewAnimation()
		stw.end = e.Position
		if stw.interactiveWheel(e) {
			return
		}
		if stw.frame != nil && e.Scroll.Y != 0 {
			stw.sceneDirty.Store(true)
			factor := 1.0
			if e.Scroll.Y > 0 {
				factor = -1
			}
			v := stw.frame.View
			if e.Modifiers&key.ModAlt != 0 {
				stw.ensureRangeView()
				v = stw.rangeView
			}
			x, y := float64(e.Position.X), float64(e.Position.Y)
			if e.Modifiers&key.ModCtrl != 0 {
				x, y = v.Center[0], v.Center[1]
			}
			stw.zoomMouseView(v, factor, x, y)
		}
	}
}

func (stw *Window) cancelCanvasDrag() {
	stw.selecting = false
	stw.press = 0
}

func (stw *Window) updateCanvasDrag(position f32.Point, modifiers key.Modifiers) {
	dx, dy := position.X-stw.dragLast.X, position.Y-stw.dragLast.Y
	stw.dragLast, stw.end = position, position
	s := stw.mouse.settings
	if math.Abs(float64(position.X-stw.start.X)) > float64(s.radius.X) || math.Abs(float64(position.Y-stw.start.Y)) > float64(s.radius.Y) {
		stw.mouse.moved = true
	}
	if stw.mouse.double {
		return
	}
	if stw.press&pointer.ButtonTertiary != 0 && stw.frame != nil {
		v := stw.frame.View
		if modifiers&key.ModAlt != 0 {
			stw.ensureRangeView()
			v = stw.rangeView
			stw.rangeVisible = true
		}
		if !stw.fixMove && (modifiers&key.ModShift != 0 || stw.fixRotate) {
			v.Center[0] += float64(dx) * stw.CanvasMoveSpeedX()
			v.Center[1] += float64(dy) * stw.CanvasMoveSpeedY()
		} else if !stw.fixRotate {
			v.Angle[0] += float64(dy) * stw.CanvasRotateSpeedY()
			v.Angle[1] -= float64(dx) * stw.CanvasRotateSpeedX()
		}
		stw.sceneDirty.Store(true)
	}
	if stw.selecting {
		stw.selected = image.Rectangle{Min: image.Pt(int(stw.start.X), int(stw.start.Y)), Max: image.Pt(int(position.X), int(position.Y))}.Canon()
	}
}

func (stw *Window) Frame() *st.Frame {
	return stw.frame
}

func (stw *Window) SetFrame(frame *st.Frame) {
	stw.cancelViewAnimation()
	stw.frame = frame
	stw.sceneDirty.Store(true)
}

func (stw *Window) AddTail(n *st.Node) {
	if tailnodes == nil {
		tailnodes = []*st.Node{n}
	} else {
		tailnodes = append(tailnodes, n)
	}
}

func (stw *Window) EndTail() {
	tailnodes = nil
}

func (stw *Window) TailLine() {
}

func (stw *Window) CurrentPointerPosition() []int {
	return []int{int(stw.end.X), int(stw.end.Y)}
}

func (stw *Window) FeedCommand() {
	command := stw.CommandLineString()
	if command != "" {
		stw.AddCommandHistory(command)
		stw.ClearCommandLine()
		// stw.ClearTypewrite()
		stw.ExecCommand(command)
		stw.Redraw()
	}
}

func (stw *Window) ExecCommand(command string) {
	stw.cancelViewAnimation()
	stw.mouse.lastEdit = ""
	stw.stopMouseEdit()
	if stw.frame == nil {
		if strings.HasPrefix(command, ":") {
			err := st.ExMode(stw, command)
			if err != nil {
				if _, ok := err.(st.NotRedraw); ok {
					return
				} else {
					st.ErrorMessage(stw, err, st.ERROR)
				}
			}
		} else if strings.HasPrefix(command, "'") {
			err := st.Fig2Mode(stw, command)
			if err != nil {
				st.ErrorMessage(stw, err, st.ERROR)
			}
		}
		return
	}
	switch {
	default:
		if c, ok := stw.CommandAlias(strings.ToUpper(command)); ok {
			stw.lastcommand = c
			stw.startInteractiveCommand(c)
		} else {
			stw.History(fmt.Sprintf("command doesn't exist: %s", command))
		}
	case strings.HasPrefix(command, ":"):
		err := st.ExMode(stw, command)
		if err != nil {
			if _, ok := err.(st.NotRedraw); ok {
				return
			} else {
				st.ErrorMessage(stw, err, st.ERROR)
			}
		}
	case strings.HasPrefix(command, "'"):
		err := st.Fig2Mode(stw, command)
		if err != nil {
			st.ErrorMessage(stw, err, st.ERROR)
		}
	}
}

func (stw *Window) LastExCommand() string {
	return stw.lastexcommand
}

func (stw *Window) SetLastExCommand(command string) {
	stw.lastexcommand = command
}

func (stw *Window) Complete() string {
	var rtn []string
	str := stw.LastWord()
	switch {
	case strings.HasPrefix(str, ":"):
		i := 0
		rtn = make([]string, len(st.ExAbbrev))
		for ab := range st.ExAbbrev {
			pat := abbrev.MustCompile(ab)
			l := fmt.Sprintf(":%s", pat.Longest())
			if strings.HasPrefix(l, str) {
				rtn[i] = l
				i++
			}
		}
		rtn = rtn[:i]
		sort.Strings(rtn)
	case strings.HasPrefix(str, "'"):
		i := 0
		rtn = make([]string, len(st.Fig2Abbrev))
		for ab := range st.Fig2Abbrev {
			pat := abbrev.MustCompile(ab)
			l := fmt.Sprintf("'%s", pat.Longest())
			if strings.HasPrefix(l, str) {
				rtn[i] = l
				i++
			}
		}
		rtn = rtn[:i]
		sort.Strings(rtn)
	default:
		if lis, ok := stw.ContextComplete(); ok {
			rtn = lis
		} else {
			rtn = st.CompleteFileName(str, stw.frame.Path, stw.Recent())
		}
	}
	if len(rtn) == 0 {
		return str
	} else {
		stw.StartCompletion(rtn)
		return rtn[0]
	}
}

func (stw *Window) LastFig2Command() string {
	return stw.lastfig2command
}

func (stw *Window) SetLastFig2Command(c string) {
	stw.lastfig2command = c
}

func (stw *Window) ShowCenter() {
	stw.SetAngle(stw.frame.View.Angle[0], stw.frame.View.Angle[1])
}

func (stw *Window) Print() {
}

func (stw *Window) SaveAS() {
	if stw.usesNativeFileDialog() {
		path, selected, err := stw.pickFile("Save As")
		if err != nil {
			stw.History(err.Error())
			return
		}
		if selected {
			stw.applyChosenFile("Save As", path, true)
		}
		return
	}
	answer := make(chan string, 1)
	d := &pathDialog{title: "Save As", ed: widget.Editor{SingleLine: true, Submit: true}, focus: true}
	if stw.frame != nil {
		d.ed.SetText(stw.frame.Path)
	}
	d.apply = func(path string) {
		select {
		case answer <- path:
		default:
		}
	}
	d.cancel = func() {
		select {
		case answer <- "":
		default:
		}
	}
	select {
	case stw.ui.pathRequests <- d:
	case <-stw.ui.done:
		return
	}
	stw.QueueRedrawAll()
	var path string
	select {
	case path = <-answer:
	case <-stw.ui.done:
		return
	}
	if path == "" {
		return
	}
	if filepath.Ext(path) == "" {
		path += ".inp"
	}
	if st.FileExists(path) && !stw.Yn("Save", "上書きしますか？") {
		return
	}
	if err := st.SaveFile(stw, path); err != nil {
		stw.History(err.Error())
	} else {
		st.Rebase(stw, path)
	}
}

func (stw *Window) SaveFileSelected(string) error {
	return nil
}

func (stw *Window) SearchFile(fn string) (string, error) {
	return fn, fmt.Errorf("file not found: %s", fn)
}

func (stw *Window) ShapeData(sh st.Shape) {
	// shapedata = sh
	// var tb *st.TextBox
	// if t, tok := stw.textBox["SHAPE"]; tok {
	// 	tb = t
	// } else {
	// 	tb = st.NewTextBox(NewFont())
	// 	tb.Show()
	// 	w, h := stw.GetCanvasSize()
	// 	tb.SetPosition(float64(w-250), float64(h-225))
	// 	stw.textBox["SHAPE"] = tb
	// }
	// var otp bytes.Buffer
	// otp.WriteString(fmt.Sprintf("%s\n", sh.String()))
	// otp.WriteString(fmt.Sprintf("A   = %10.4f [cm2]\n", sh.A()))
	// otp.WriteString(fmt.Sprintf("Asx = %10.4f [cm2]\n", sh.Asx()))
	// otp.WriteString(fmt.Sprintf("Asy = %10.4f [cm2]\n", sh.Asy()))
	// otp.WriteString(fmt.Sprintf("Ix  = %10.4f [cm4]\n", sh.Ix()))
	// otp.WriteString(fmt.Sprintf("Iy  = %10.4f [cm4]\n", sh.Iy()))
	// otp.WriteString(fmt.Sprintf("J   = %10.4f [cm4]\n", sh.J()))
	// otp.WriteString(fmt.Sprintf("Zx  = %10.4f [cm3]\n", sh.Zx()))
	// otp.WriteString(fmt.Sprintf("Zy  = %10.4f [cm3]\n", sh.Zy()))
	// tb.SetText(strings.Split(otp.String(), "\n"))
}

func (stw *Window) ToggleFixRotate() {
	stw.fixRotate = !stw.fixRotate
}

func (stw *Window) ToggleFixMove() {
	stw.fixMove = !stw.fixMove
}

func (stw *Window) SetShowPrintRange(val bool) {
	showprintrange = val
	st.SetPosition(stw)
}

func (stw *Window) ToggleShowPrintRange() {
	showprintrange = !showprintrange
	st.SetPosition(stw)
}

func (stw *Window) CurrentLap(message string, step, total int) {
	stw.ui.mu.Lock()
	stw.ui.progress = fmt.Sprintf("%s %d / %d", message, step, total)
	stw.ui.mu.Unlock()
	stw.QueueRedrawAll()
}

func (stw *Window) SectionData(sec *st.Sect) {
	// var tb *st.TextBox
	// if t, tok := stw.textBox["SECTION"]; tok {
	// 	tb = t
	// } else {
	// 	tb = st.NewTextBox(NewFont())
	// 	tb.Show()
	// 	w, h := stw.GetCanvasSize()
	// 	tb.SetPosition(float64(w-250), float64(h-500))
	// 	stw.textBox["SECTION"] = tb
	// }
	// tb.SetText(strings.Split(sec.InpString(), "\n"))
	// if al := sec.Allow; al != nil {
	// 	tb.AddText(strings.Split(al.String(), "\n")...)
	// }
	// tb.ScrollToTop()
}

func (stw *Window) TextBox(name string) *st.TextBox {
	// if _, tok := stw.textBox[name]; !tok {
	// 	stw.textBox[name] = st.NewTextBox(stw.currentFont)
	// }
	// return stw.textBox[name]
	return nil
}

func (stw *Window) TextBoxes() []*st.TextBox {
	rtn := make([]*st.TextBox, len(stw.textBox))
	i := 0
	for _, t := range stw.textBox {
		rtn[i] = t
		i++
	}
	return rtn
}

func (stw *Window) SetAngle(phi, theta float64) {
	if stw.frame == nil {
		return
	}
	target := stw.frame.View.Copy()
	target.Angle = []float64{phi, theta}
	if len(stw.frame.Nodes) > 0 {
		target = st.CenterView(stw, target.Angle)
	}
	stw.Animate(target)
}

func (stw *Window) SetPaperSize(name uint) {
	stw.papersize = name
}

func (stw *Window) PaperSize() uint {
	return stw.papersize
}

func (stw *Window) SetPeriod(period string) {
	if stw.frame != nil {
		stw.frame.Show.Period = period
	}
}

func (stw *Window) Pivot() bool {
	return false
}

func (stw *Window) DrawPivot([]*st.Node, chan int, chan int) {
}

func (stw *Window) SetColorMode(mode uint) {
	stw.frame.Show.ColorMode = mode
}

func (stw *Window) DefaultColorMode() uint {
	return st.ECOLOR_BLACKSECT
}

func (stw *Window) History(str string) {
	if str == "" {
		return
	}
	if !strings.HasSuffix(str, "\n") {
		str += "\n"
	}
	stw.appendLog(str)
}

func (stw *Window) HistoryWriter() io.Writer { return windowHistoryWriter{stw} }

func (stw *Window) Yn(title, question string) bool {
	return stw.ask(title, question, "はい", "いいえ") == 1
}

func (stw *Window) Yna(title, question, another string) int {
	return stw.ask(title, question, "はい", "いいえ", another)
}

func (stw *Window) EnableLabel(string) {
}

func (stw *Window) DisableLabel(string) {
}

func (stw *Window) SetLabel(k, v string) {
}

func (stw *Window) GetCanvasSize() (int, int) {
	if stw.canvasSize.X == 0 || stw.canvasSize.Y == 0 {
		return 1000, 1000
	}
	return stw.canvasSize.X, stw.canvasSize.Y
}

func (stw *Window) Changed(c bool) {
	stw.changed = c
}

func (stw *Window) IsChanged() bool {
	return stw.changed
}

func (stw *Window) Redraw() {
	stw.sceneDirty.Store(true)
	stw.QueueRedrawAll()
}

func (stw *Window) RedrawNode() {
	// Keep the common Window interface without switching drawing rules.
	stw.Redraw()
}

func (stw *Window) EPS() float64 {
	return 1e-3
}

func (stw *Window) SetEPS(float64) {
}

func (stw *Window) Close(bang bool) {
	if !bang && stw.changed {
		if !stw.Yn("Quit", "変更を保存せず終了しますか？") {
			return
		}
	}
	err := stw.SaveRecent()
	if err != nil {
		fmt.Println(err)
	}
	err = stw.SaveCommandHistory("")
	if err != nil {
		fmt.Println(err)
	}
	if stw.window != nil {
		stw.window.Perform(system.ActionClose)
	}
}

// for st.Selector
func (stw *Window) ToggleAltSelectNode() {
	altselectnode = !altselectnode
}

func (stw *Window) AltSelectNode() bool {
	return altselectnode
}

func (stw *Window) SetAltSelectNode(a bool) {
	altselectnode = a
}

func (stw *Window) Zoom(factor float64, x, y float64) float64 {
	return stw.zoomMouseView(stw.frame.View, factor, x, y)
}

func (stw *Window) zoomMouseView(view *st.View, factor float64, x, y float64) float64 {
	val := math.Pow(2.0, factor/stw.CanvasScaleSpeed())
	view.Center[0] += (val - 1.0) * (view.Center[0] - x)
	view.Center[1] += (val - 1.0) * (view.Center[1] - y)
	if view.Perspective {
		view.Dists[1] *= val
		if view.Dists[1] < 0.0 {
			view.Dists[1] = 0.0
		}
	} else {
		view.Gfact *= val
		if view.Gfact < 0.0 {
			view.Gfact = 0.0
		}
	}
	return val
}

func (stw *Window) WindowZoom(factor float64, x, y float64) {
	stw.frame.View.Center[0] += (factor - 1.0) * (stw.frame.View.Center[0] - x)
	stw.frame.View.Center[1] += (factor - 1.0) * (stw.frame.View.Center[1] - y)
	if stw.frame.View.Perspective {
		stw.frame.View.Dists[1] *= factor
		if stw.frame.View.Dists[1] < 0.0 {
			stw.frame.View.Dists[1] = 0.0
		}
	} else {
		stw.frame.View.Gfact *= factor
		if stw.frame.View.Gfact < 0.0 {
			stw.frame.View.Gfact = 0.0
		}
	}
}

var (
	selectbox = []float64{0.0, 0.0, 0.0, 0.0}
)

func (stw *Window) SetTextboxPosition() {
	sx, sy, _, ph := st.GetClipCoord(stw)
	w, _ := stw.GetCanvasSize()
	cw := float64(w)
	// ch := float64(h)
	if t, tok := stw.textBox["SECTION"]; tok {
		t.SetPosition(cw-250, ph+sy-500)
	}
	if t, tok := stw.textBox["PAGETITLE"]; tok {
		t.SetPosition(sx+100, ph-100)
	}
	if t, tok := stw.textBox["TITLE"]; tok {
		t.SetPosition(sx+100, ph-200)
	}
	if t, tok := stw.textBox["DATA"]; tok {
		t.SetPosition(sx+100, 200)
	}
}
