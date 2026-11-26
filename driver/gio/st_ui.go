package stgio

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf8"

	"gioui.org/f32"
	"gioui.org/io/clipboard"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/yofu/abbrev"
)

// Classic IUP palette (matching driver/iup/st_window.go).
var (
	iupBgLabel    = color.NRGBA{R: 212, G: 208, B: 200, A: 255}
	iupBgHistory  = color.NRGBA{R: 220, G: 220, B: 220, A: 255}
	iupBgCommand  = color.NRGBA{R: 191, G: 191, B: 191, A: 255}
	iupBgField    = color.NRGBA{R: 212, G: 208, B: 200, A: 255}
	iupBgFieldAct = color.NRGBA{R: 226, G: 224, B: 218, A: 255}
	iupBgTabInact = color.NRGBA{R: 196, G: 193, B: 186, A: 255}
	iupBgButton   = color.NRGBA{R: 204, G: 200, B: 193, A: 255}
	iupBgMenu     = color.NRGBA{R: 224, G: 220, B: 213, A: 255}
	iupFgLabel    = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	iupFgCommand  = color.NRGBA{R: 0, G: 127, B: 31, A: 255}
	iupFgSection  = color.NRGBA{R: 204, G: 51, B: 0, A: 255}
	iupFgOff      = color.NRGBA{R: 128, G: 128, B: 128, A: 255}
	iupBorder     = color.NRGBA{R: 145, G: 142, B: 136, A: 255}
)

// UI widgets belong to the frame goroutine. Logs, progress and questions may
// arrive from analysis/command goroutines and are protected separately.
type userInterface struct {
	historyMu            sync.Mutex
	commandBefore        string
	historyBrowsing      bool
	lastCompletion       bool
	punctuationPending   bool
	spacePending         bool
	modelMu              sync.Mutex
	mu                   sync.Mutex
	log                  string
	progress             string
	question             *uiQuestion
	jobs                 chan uiJob
	done                 chan struct{}
	once                 sync.Once
	stop                 sync.Once
	pending              atomic.Int32
	command              widget.Editor
	history              widget.Editor
	historyText          string
	commandHistory       []string
	historyIndex         int
	draft                string
	completionInput      string
	completionCandidates []string
	completionIndex      int
	buttons              map[string]*widget.Clickable
	fields               map[string]*uiField
	checks               map[string]*widget.Bool
	values               map[string]string
	flags                map[string]bool
	properties           []string
	propNodeDetails      []string
	propElemCount        int
	propLineCount        int
	propPlateCount       int
	propUnlocked         int
	propEditable         int
	propFixedBrace       int
	propEditLines        int
	propEditPlates       int
	propSelectionKey     string
	propSectLabel        string
	propSectMixed        bool
	propEtypeMixed       bool
	propCommonEtype      int
	sectionPickerOpen    bool
	sectionPickerList    widget.List
	sections             []int
	tab                  int
	menu                 int
	panelList            [3]widget.List
	menuList             widget.List
	canvasOps            op.Ops
	canvasCall           op.CallOp
	canvasReady          bool
	canvasRevision       int
	canvasRect           image.Rectangle
	selectionOverlay     bool
	fenceOverlay         bool
	selectionPointer     pointer.ID
	selectionStart       f32.Point
	selectionEnd         f32.Point
	cursorPosition       f32.Point
	commandTail          []f32.Point
	historyRect          image.Rectangle
	commandRect          image.Rectangle
	panelRect            image.Rectangle
	menuRect             image.Rectangle
	contextItems         []menuItem
	contextPoint         image.Point
	modal                *pathDialog
	pathRequests         chan *pathDialog
	focusCommand         bool
	status               string
	commandName          string
}

type uiField struct {
	ed      widget.Editor
	dirty   bool
	focused bool
}

type uiJob struct {
	run       func()
	interrupt bool
}

type uiQuestion struct {
	title, message string
	choices        []string
	answer         chan int
}

type pathDialog struct {
	textOnly bool
	owner    *interactiveCommand
	title    string
	ed       widget.Editor
	apply    func(string)
	focus    bool
	cancel   func()
}

func newUserInterface() *userInterface {
	u := &userInterface{
		jobs: make(chan uiJob, 64), done: make(chan struct{}),
		buttons: make(map[string]*widget.Clickable), fields: make(map[string]*uiField),
		checks: make(map[string]*widget.Bool), values: make(map[string]string), flags: make(map[string]bool),
		menu: -1, historyIndex: -1,
		focusCommand: true,
		pathRequests: make(chan *pathDialog, 1),
		command:      widget.Editor{SingleLine: true, Submit: true},
		history:      widget.Editor{ReadOnly: true},
	}
	for i := range u.panelList {
		u.panelList[i].Axis = layout.Vertical
	}
	u.menuList.Axis = layout.Vertical
	u.sectionPickerList.Axis = layout.Vertical
	return u
}

func (w *Window) enqueue(job func()) {
	w.enqueueJob(uiJob{run: job})
}
func (w *Window) enqueueInterrupting(job func()) { w.enqueueJob(uiJob{run: job, interrupt: true}) }
func (w *Window) enqueueJob(job uiJob) {
	w.tryEnqueueJob(job)
}

func (w *Window) tryEnqueueJob(job uiJob) bool {
	u := w.ui
	u.once.Do(func() {
		go func() {
			for {
				select {
				case <-u.done:
					return
				case f := <-u.jobs:
					if f.interrupt {
						w.stopInteractiveCommand()
					}
					u.modelMu.Lock()
					f.run()
					u.modelMu.Unlock()
					u.pending.Add(-1)
					w.QueueRedrawAll()
				}
			}
		}()
	})
	u.pending.Add(1)
	select {
	case u.jobs <- job:
		return true
	case <-u.done:
		u.pending.Add(-1)
		return false
	default:
		u.pending.Add(-1)
		w.History("操作待ちが多いため受け付けられません。少し待って再実行してください。")
		return false
	}
}

func (u *userInterface) close() { u.stop.Do(func() { close(u.done) }) }

func (w *Window) appendLog(s string) {
	u := w.ui
	u.mu.Lock()
	u.log += strings.ReplaceAll(s, "\r\n", "\n")
	// Keep complete UTF-8 lines while bounding long analysis logs.
	if len(u.log) > 256*1024 {
		if cut := strings.IndexByte(u.log[len(u.log)-192*1024:], '\n'); cut >= 0 {
			u.log = u.log[len(u.log)-192*1024+cut+1:]
		} else {
			cut := len(u.log) - 192*1024
			for cut < len(u.log) && !utf8.RuneStart(u.log[cut]) {
				cut++
			}
			u.log = u.log[cut:]
		}
	}
	u.mu.Unlock()
	w.QueueRedrawAll()
}

type windowHistoryWriter struct{ w *Window }

func (h windowHistoryWriter) Write(p []byte) (int, error) {
	h.w.appendLog(string(p))
	return len(p), nil
}

func (w *Window) ask(title, message string, choices ...string) int {
	u := w.ui
	q := &uiQuestion{title: title, message: message, choices: choices, answer: make(chan int, 1)}
	u.mu.Lock()
	u.question = q
	u.mu.Unlock()
	w.QueueRedrawAll()
	select {
	case answer := <-q.answer:
		return answer
	case <-u.done:
		return len(choices)
	}
}

func (w *Window) answerQuestion(q *uiQuestion, answer int) {
	w.ui.mu.Lock()
	if w.ui.question == q {
		w.ui.question = nil
	}
	w.ui.mu.Unlock()
	select {
	case q.answer <- answer:
	default:
	}
	w.QueueRedrawAll()
}

func (w *Window) submitCommand(command string) {
	command = strings.TrimSpace(command)
	if command == "" {
		return
	}
	u := w.ui
	u.historyMu.Lock()
	w.addUICommandHistoryLocked(command)
	u.historyMu.Unlock()
	u.historyIndex = -1
	u.draft = ""
	u.completionCandidates = nil
	u.command.SetText("")
	u.commandBefore = ""
	u.punctuationPending = false
	u.spacePending = false
	u.historyBrowsing = false
	u.lastCompletion = false
	if err := w.SaveCommandHistory(""); err != nil {
		w.History("history.dat: " + err.Error())
	}
	w.History("> " + command)
	w.enqueueInterrupting(func() { w.SetCommandLineString(command); w.FeedCommand() })
}

func commandNames[T any](values map[string]T, prefix string) map[string]bool {
	result := make(map[string]bool)
	for name := range values {
		result[prefix+abbrev.MustCompile(name).Longest()] = true
	}
	return result
}

func (u *userInterface) button(id string) *widget.Clickable {
	if u.buttons[id] == nil {
		u.buttons[id] = new(widget.Clickable)
	}
	return u.buttons[id]
}

func (w *Window) smallButton(gtx layout.Context, id, text string, action func()) layout.Dimensions {
	b := w.ui.button(id)
	for b.Clicked(gtx) {
		action()
		w.QueueRedrawAll()
	}
	style := material.Button(w.Theme, b, text)
	style.TextSize = unit.Sp(11)
	style.CornerRadius = unit.Dp(0)
	style.Inset = layout.Inset{Top: unit.Dp(3), Bottom: unit.Dp(3), Left: unit.Dp(7), Right: unit.Dp(7)}
	style.Background = iupBgButton
	style.Color = iupFgLabel
	return widget.Border{Color: iupBorder, Width: unit.Dp(1)}.Layout(gtx, style.Layout)
}

func (w *Window) flatButton(gtx layout.Context, id, text string, bg, fg color.NRGBA, inset layout.Inset, action func()) layout.Dimensions {
	b := w.ui.button(id)
	for b.Clicked(gtx) {
		action()
		w.QueueRedrawAll()
	}
	style := material.Button(w.Theme, b, text)
	style.TextSize = unit.Sp(11)
	style.CornerRadius = unit.Dp(0)
	style.Inset = inset
	style.Background = bg
	style.Color = fg
	return style.Layout(gtx)
}

func (w *Window) uiLabel(gtx layout.Context, text string) layout.Dimensions {
	s := material.Body2(w.Theme, text)
	s.TextSize = unit.Sp(11)
	s.Color = iupFgLabel
	return s.Layout(gtx)
}

func (w *Window) setCommandName(name string) {
	w.ui.mu.Lock()
	w.ui.commandName = name
	w.ui.mu.Unlock()
}

func (w *Window) currentModeName() string {
	if w.interaction.Load() == nil {
		return "SELECT"
	}
	w.ui.mu.Lock()
	name := w.ui.commandName
	w.ui.mu.Unlock()
	if name == "" {
		return "COMMAND"
	}
	return name
}

func region(gtx layout.Context, r image.Rectangle, draw layout.Widget) layout.Dimensions {
	offset := op.Offset(r.Min).Push(gtx.Ops)
	defer offset.Pop()
	area := clip.Rect{Max: r.Size()}.Push(gtx.Ops)
	defer area.Pop()
	gtx.Constraints = layout.Exact(r.Size())
	return draw(gtx)
}

func (w *Window) Layout(gtx layout.Context) layout.Dimensions {
	if w.viewer != nil {
		return w.layoutModelViewer(gtx)
	}
	u := w.ui
	if u.modal != nil && u.modal.owner != nil && w.interaction.Load() != u.modal.owner {
		u.modal.cancel()
		u.modal = nil
	}
	select {
	case d := <-u.pathRequests:
		if d.owner == nil || w.interaction.Load() == d.owner {
			u.modal = d
		} else if d.cancel != nil {
			d.cancel()
		}
	default:
	}
	size := gtx.Constraints.Max
	paint.Fill(gtx.Ops, iupBgLabel)
	menuH := min(gtx.Dp(26), size.Y)
	panelW := min(gtx.Dp(236), size.X/2)
	commandH := min(gtx.Dp(30), max(0, size.Y-menuH))
	historyH := min(gtx.Dp(136), max(0, size.Y-menuH-commandH)/3)
	canvasBottom := max(menuH, size.Y-historyH-commandH)
	// Match IUP layout: MenuBar on top, full-height SideBar on the left,
	// and Canvas -> History -> [Mode | Command | Coord/Status] stacked on the right.
	u.panelRect = image.Rect(0, menuH, panelW, size.Y)
	u.canvasRect = image.Rect(panelW, menuH, size.X, canvasBottom)
	u.historyRect = image.Rect(panelW, canvasBottom, size.X, size.Y-commandH)
	u.commandRect = image.Rect(panelW, size.Y-commandH, size.X, size.Y)
	// Command input has explicit filters; focused field editors keep their keys.
	w.interactiveKeyboard(gtx)
	w.keyboard(gtx)
	u.mu.Lock()
	blocked := u.question != nil
	u.mu.Unlock()
	body := gtx
	if blocked || u.modal != nil {
		body = gtx.Disabled()
	}
	// Keep the input tag in the window coordinate system at every DPI setting.
	w.collectCanvasInput(body, u.canvasRect)
	region(body, image.Rect(0, 0, size.X, menuH), w.layoutMenuBar)
	region(body, u.canvasRect, func(c layout.Context) layout.Dimensions {
		if u.modelMu.TryLock() {
			w.prepareCanvas(c)
			w.applyCanvasInput()
			w.advanceViewAnimation(c.Now)
			if w.sceneDirty.Swap(false) || !u.canvasReady {
				u.canvasOps.Reset()
				macro := op.Record(&u.canvasOps)
				scene := c
				scene.Ops = &u.canvasOps
				w.drawCanvas(scene)
				u.canvasCall = macro.Stop()
				u.canvasReady = true
				u.canvasRevision++
			}
			w.snapshotControls()
			w.snapshotCommandTail()
			u.modelMu.Unlock()
		}
		// A recorded call keeps the model visible during commands and questions.
		if u.canvasReady {
			u.canvasCall.Add(c.Ops)
		}
		// Like IUP's front-canvas rectangle, this is independent of model data
		// and is repainted even while the command worker holds modelMu.
		w.drawCanvasOverlay(c)
		// Subtle border around canvas matching IUP's BORDER=YES.
		widget.Border{Color: iupBorder, Width: unit.Dp(1)}.Layout(c, func(c layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: c.Constraints.Max}
		})
		return layout.Dimensions{Size: c.Constraints.Max}
	})
	w.scheduleViewAnimation(gtx)
	region(body, u.panelRect, w.layoutPanel)
	region(body, u.historyRect, w.layoutHistory)
	region(body, u.commandRect, w.layoutCommand)
	if u.menu >= 0 || len(u.contextItems) > 0 {
		w.layoutMenuPopup(gtx)
	}
	if u.sectionPickerOpen {
		w.layoutSectionPickerPopup(gtx)
	}
	u.mu.Lock()
	question := u.question
	u.mu.Unlock()
	if question != nil {
		w.layoutQuestion(gtx, question)
	} else if u.modal != nil {
		w.layoutPathDialog(gtx)
	}
	return layout.Dimensions{Size: size}
}

func (w *Window) cycleSideBarTab(reverse bool) {
	w.ui.sectionPickerOpen = false
	if reverse {
		w.ui.tab = (w.ui.tab + 2) % 3
	} else {
		w.ui.tab = (w.ui.tab + 1) % 3
	}
}

func (w *Window) anyPanelFieldFocused(gtx layout.Context) bool {
	for _, f := range w.ui.fields {
		if f != nil && gtx.Focused(&f.ed) {
			return true
		}
	}
	return false
}

func (w *Window) keyboard(gtx layout.Context) {
	u := w.ui
	if u.focusCommand {
		gtx.Execute(key.FocusCmd{Tag: &u.command})
		u.focusCommand = false
	}
	for {
		ev, ok := gtx.Event(
			key.Filter{Focus: &u.command, Name: key.NameUpArrow}, key.Filter{Focus: &u.command, Name: key.NameDownArrow},
			key.Filter{Name: key.NameUpArrow, Required: key.ModCtrl}, key.Filter{Name: key.NameDownArrow, Required: key.ModCtrl},
			key.Filter{Focus: &u.command, Name: key.NameTab, Optional: key.ModShift},
			key.Filter{Name: key.NameTab, Required: key.ModCtrl, Optional: key.ModShift},
			key.Filter{Name: key.NameEscape},
			key.Filter{Name: key.NameDeleteForward},
			key.Filter{Focus: &u.command, Name: ";", Optional: key.ModShift}, key.Filter{Focus: &u.command, Name: ":", Optional: key.ModShift},
			key.Filter{Focus: &u.command, Name: key.NameSpace, Optional: key.ModShift},
			key.Filter{Focus: &u.command, Name: "P", Required: key.ModCtrl}, key.Filter{Focus: &u.command, Name: "N", Required: key.ModCtrl},
			key.Filter{Name: "N", Required: key.ModCtrl | key.ModShift},
			key.Filter{Focus: &u.command, Name: "[", Required: key.ModCtrl},
			key.Filter{Name: "A", Required: key.ModCtrl, Optional: key.ModShift},
			key.Filter{Focus: &u.command, Name: "E", Required: key.ModCtrl}, key.Filter{Focus: &u.command, Name: "X", Required: key.ModCtrl},
			key.Filter{Name: "D", Required: key.ModCtrl}, key.Filter{Name: "H", Required: key.ModCtrl},
			key.Filter{Name: "F", Required: key.ModCtrl},
			key.Filter{Name: "U", Required: key.ModCtrl, Optional: key.ModShift},
			key.Filter{Name: "Z", Required: key.ModCtrl}, key.Filter{Name: "Y", Required: key.ModCtrl},
			key.Filter{Name: "I", Required: key.ModCtrl}, key.Filter{Name: "M", Required: key.ModCtrl}, key.Filter{Name: "R", Required: key.ModCtrl},
			key.Filter{Name: "O", Required: key.ModCtrl}, key.Filter{Name: "S", Required: key.ModCtrl},
			key.Filter{Name: "L", Required: key.ModCtrl, Optional: key.ModShift},
		)
		if !ok {
			break
		}
		k, ok := ev.(key.Event)
		if !ok || k.State != key.Press {
			continue
		}
		u.mu.Lock()
		hasQuestion := u.question != nil
		u.mu.Unlock()
		if (hasQuestion || u.modal != nil) && k.Name != key.NameEscape {
			continue
		}
		switch k.Name {
		case ";", ":":
			if u.command.Text() == "" {
				u.punctuationPending = true
			}
		case key.NameSpace:
			if !strings.Contains(u.command.Text(), " ") {
				u.spacePending = true
			}
		case key.NameUpArrow:
			if k.Modifiers&key.ModCtrl != 0 {
				w.menuAction("next-floor")
			} else {
				w.browseHistory(true)
			}
		case "P":
			w.browseHistory(true)
		case key.NameDownArrow:
			if k.Modifiers&key.ModCtrl != 0 {
				w.menuAction("prev-floor")
			} else {
				w.browseHistory(false)
			}
		case "N":
			if k.Modifiers&key.ModShift != 0 {
				w.menuAction("flip-period-sign")
			} else {
				w.browseHistory(false)
			}
		case key.NameTab:
			if k.Modifiers&key.ModCtrl != 0 {
				w.cycleSideBarTab(k.Modifiers&key.ModShift != 0)
			} else if k.Modifiers&key.ModShift != 0 {
				w.completeCommandDirection(-1)
			} else {
				w.completeCommand()
			}
		case "D":
			w.menuAction("isolate")
		case "H":
			w.menuAction("hide")
		case "F":
			w.menuAction("focus-selected")
		case "U":
			if k.Modifiers&key.ModShift != 0 {
				w.menuAction("unlock-all")
			} else {
				w.menuAction("toggle-unit")
			}
		case "Z":
			if u.command.Text() == "" && !w.anyPanelFieldFocused(gtx) {
				w.menuAction("undo")
			}
		case "Y":
			if u.command.Text() == "" && !w.anyPanelFieldFocused(gtx) {
				w.menuAction("redo")
			}
		case key.NameDeleteForward:
			if u.command.Text() == "" && !w.anyPanelFieldFocused(gtx) {
				w.menuAction("delete-direct")
			}
		case "I":
			w.fileDialog("Insert")
		case "A":
			if k.Modifiers&key.ModShift != 0 {
				w.menuAction("select-visible")
			} else {
				w.menuAction("showall")
			}
		case "E":
			w.menuAction("edit-inp")
		case "X":
			w.ui.tab = 1
			w.ui.panelList[1].Position = layout.Position{First: max(0, len(w.panelRows())-len(w.ui.sections)-1)}
		case "M":
			w.menuAction("reload")
		case "R":
			w.menuAction("readall")
		case "[":
			w.setCommandText("")
			u.historyBrowsing = false
			u.lastCompletion = false
		case "O":
			w.fileDialog("Open")
		case "S":
			w.menuAction("save")
		case "L":
			if k.Modifiers&key.ModShift != 0 {
				w.menuAction("lock-selected")
			} else {
				gtx.Execute(key.FocusCmd{Tag: &u.command})
			}
		case key.NameEscape:
			u.mu.Lock()
			q := u.question
			u.mu.Unlock()
			if q != nil {
				w.answerQuestion(q, len(q.choices))
				continue
			}
			if u.sectionPickerOpen {
				u.sectionPickerOpen = false
				continue
			}
			if w.cancelPropertyFieldEdit(gtx) {
				continue
			}
			if u.modal != nil && u.modal.cancel != nil {
				u.modal.cancel()
			}
			u.modal = nil
			u.menu = -1
			u.contextItems = nil
			u.sectionPickerOpen = false
			w.setCommandText("")
			u.historyBrowsing = false
			u.lastCompletion = false
			u.selectionOverlay = false
			w.deselectOnEscape()
		}
	}
}

func (w *Window) layoutCommand(gtx layout.Context) layout.Dimensions {
	u := w.ui
	if draft := w.commandDraft.Swap(nil); draft != nil {
		w.setCommandText(draft.text)
		u.command.SetCaret(draft.caret, draft.caret)
		u.focusCommand = true
	}
	if u.focusCommand {
		gtx.Execute(key.FocusCmd{Tag: &u.command})
		u.focusCommand = false
	}
	for {
		e, ok := u.command.Update(gtx)
		if !ok {
			break
		}
		if _, ok := e.(widget.SubmitEvent); ok {
			w.commandChanged() // Enter can arrive in the same frame as the final edit.
			w.submitCommand(u.command.Text())
		}
		if _, ok := e.(widget.ChangeEvent); ok {
			w.commandChanged()
		}
	}
	paint.Fill(gtx.Ops, iupBgHistory)
	paint.FillShape(gtx.Ops, iupBorder, clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, 1)).Op())
	mode := w.currentModeName()
	u.mu.Lock()
	progress := u.progress
	u.mu.Unlock()
	coordText := u.status
	if u.pending.Load() > 0 {
		if progress != "" {
			coordText = "Working... " + progress
		} else {
			coordText = "Working..."
		}
	} else if progress != "" {
		coordText = progress
	}
	return layout.Inset{Top: unit.Dp(1)}.Layout(gtx, func(c layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.Middle}.Layout(c,
			// Left: IUP cname (mode display in green on history gray).
			layout.Rigid(func(c layout.Context) layout.Dimensions {
				width := min(c.Dp(112), c.Constraints.Max.X/3)
				c.Constraints = layout.Exact(image.Pt(width, c.Constraints.Max.Y))
				paint.FillShape(c.Ops, iupBgHistory, clip.Rect{Max: c.Constraints.Max}.Op())
				layout.Inset{Left: unit.Dp(6), Right: unit.Dp(4)}.Layout(c, func(c layout.Context) layout.Dimensions {
					return layout.W.Layout(c, func(c layout.Context) layout.Dimensions {
						s := material.Body2(w.Theme, mode)
						s.TextSize = unit.Sp(13)
						s.Color = iupFgCommand
						s.MaxLines = 1
						return s.Layout(c)
					})
				})
				paint.FillShape(c.Ops, iupBorder, clip.Rect(image.Rect(width-1, 0, width, c.Constraints.Max.Y)).Op())
				return layout.Dimensions{Size: c.Constraints.Max}
			}),
			// Center: IUP cline (command input on darker gray background).
			layout.Flexed(1, func(c layout.Context) layout.Dimensions {
				paint.FillShape(c.Ops, iupBgCommand, clip.Rect{Max: c.Constraints.Max}.Op())
				return layout.Inset{Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(c, func(c layout.Context) layout.Dimensions {
					return layout.W.Layout(c, func(c layout.Context) layout.Dimensions {
						e := material.Editor(w.Theme, &u.command, "Command")
						e.TextSize = unit.Sp(13)
						e.Color = iupFgLabel
						e.HintColor = color.NRGBA{R: 95, G: 95, B: 95, A: 255}
						return e.Layout(c)
					})
				})
			}),
			// Right: IUP coord/status readout on history gray.
			layout.Rigid(func(c layout.Context) layout.Dimensions {
				width := min(c.Dp(280), c.Constraints.Max.X/2)
				if width < c.Dp(60) {
					return layout.Dimensions{}
				}
				c.Constraints = layout.Exact(image.Pt(width, c.Constraints.Max.Y))
				paint.FillShape(c.Ops, iupBgHistory, clip.Rect{Max: c.Constraints.Max}.Op())
				paint.FillShape(c.Ops, iupBorder, clip.Rect(image.Rect(0, 0, 1, c.Constraints.Max.Y)).Op())
				layout.Inset{Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(c, func(c layout.Context) layout.Dimensions {
					return layout.W.Layout(c, func(c layout.Context) layout.Dimensions {
						s := material.Body2(w.Theme, coordText)
						s.TextSize = unit.Sp(13)
						s.Color = iupFgLabel
						s.MaxLines = 1
						return s.Layout(c)
					})
				})
				return layout.Dimensions{Size: c.Constraints.Max}
			}),
		)
	})
}

func (w *Window) layoutHistory(gtx layout.Context) layout.Dimensions {
	u := w.ui
	u.mu.Lock()
	text := u.log
	u.mu.Unlock()
	if text != u.historyText {
		start, end := u.history.Selection()
		atEnd := end == u.history.Len()
		u.history.SetText(text)
		u.historyText = text
		if atEnd {
			u.history.SetCaret(u.history.Len(), u.history.Len())
		} else {
			u.history.SetCaret(start, end)
		}
	}
	paint.Fill(gtx.Ops, iupBgHistory)
	paint.FillShape(gtx.Ops, iupBorder, clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, 1)).Op())
	return layout.Inset{Top: unit.Dp(1)}.Layout(gtx, func(c layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.Start}.Layout(c,
			layout.Flexed(1, func(c layout.Context) layout.Dimensions {
				c.Constraints = layout.Exact(c.Constraints.Max)
				return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(c, func(c layout.Context) layout.Dimensions {
					e := material.Editor(w.Theme, &u.history, "")
					e.TextSize = unit.Sp(13)
					e.Color = iupFgLabel
					return e.Layout(c)
				})
			}),
			layout.Rigid(func(c layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(2), Right: unit.Dp(4)}.Layout(c, func(c layout.Context) layout.Dimensions {
					btnInset := layout.Inset{Top: unit.Dp(1), Bottom: unit.Dp(1), Left: unit.Dp(5), Right: unit.Dp(5)}
					return layout.Flex{Alignment: layout.Middle}.Layout(c,
						layout.Rigid(func(c layout.Context) layout.Dimensions {
							return w.flatButton(c, "copy-history", "Copy", iupBgButton, iupFgLabel, btnInset, func() {
								c.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(strings.NewReader(text))})
							})
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(2)}.Layout),
						layout.Rigid(func(c layout.Context) layout.Dimensions {
							return w.flatButton(c, "clear-history", "Clear", iupBgButton, iupFgLabel, btnInset, func() {
								u.mu.Lock()
								u.log = ""
								u.mu.Unlock()
							})
						}),
					)
				})
			}),
		)
	})
}

func (w *Window) fileDialog(kind string) {
	u := w.ui
	if w.usesNativeFileDialog() {
		u.menu = -1
		w.enqueueInterrupting(func() {
			path, selected, err := w.pickFile(kind)
			if err != nil {
				w.History(err.Error())
				return
			}
			if selected {
				w.applyChosenFile(kind, path, true)
			}
		})
		return
	}
	d := &pathDialog{title: kind + " file", ed: widget.Editor{SingleLine: true, Submit: true}, focus: true}
	d.ed.SetText(u.values["PATH"])
	d.apply = func(path string) { w.enqueueInterrupting(func() { w.applyChosenFile(kind, path, false) }) }

	u.modal = d
	u.menu = -1
}

func (w *Window) modalBox(gtx layout.Context, title, message string, body layout.Widget) layout.Dimensions {
	paint.FillShape(gtx.Ops, color.NRGBA{A: 100}, clip.Rect{Max: gtx.Constraints.Max}.Op())
	// A full-window click target prevents pointer input reaching the canvas.
	w.ui.button("modal-shield").Layout(gtx, func(c layout.Context) layout.Dimensions { return layout.Dimensions{Size: c.Constraints.Max} })
	return layout.Center.Layout(gtx, func(c layout.Context) layout.Dimensions {
		c.Constraints.Max.X = min(c.Constraints.Max.X, c.Dp(600))
		c.Constraints.Min = image.Point{}
		return widget.Border{Color: color.NRGBA{R: 90, G: 105, B: 125, A: 255}, Width: unit.Dp(1)}.Layout(c, func(c layout.Context) layout.Dimensions {
			macro := op.Record(c.Ops)
			dims := layout.UniformInset(unit.Dp(16)).Layout(c, func(c layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(c,
					layout.Rigid(w.popupTitle(title).Layout),
					layout.Rigid(func(c layout.Context) layout.Dimensions { return w.uiLabel(c, message) }),
					layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout), layout.Rigid(body))
			})
			call := macro.Stop()
			paint.FillShape(c.Ops, color.NRGBA{R: 250, G: 252, B: 255, A: 255}, clip.Rect{Max: dims.Size}.Op())
			call.Add(c.Ops)
			return dims
		})
	})
}

func (w *Window) layoutQuestion(gtx layout.Context, q *uiQuestion) {
	w.modalBox(gtx, q.title, q.message, func(c layout.Context) layout.Dimensions {
		children := []layout.FlexChild{}
		for i, label := range q.choices {
			i, label := i, label
			children = append(children, layout.Rigid(func(c layout.Context) layout.Dimensions {
				return w.smallButton(c, fmt.Sprintf("answer-%d", i), label, func() { w.answerQuestion(q, i+1) })
			}))
		}
		return layout.Flex{}.Layout(c, children...)
	})
}

func (w *Window) layoutPathDialog(gtx layout.Context) {
	d := w.ui.modal
	apply := func() {
		path := strings.TrimSpace(d.ed.Text())
		if path != "" {
			w.ui.modal = nil
			if !d.textOnly {
				path = filepath.Clean(path)
			}
			d.apply(path)
		}
	}
	if d.focus {
		gtx.Execute(key.FocusCmd{Tag: &d.ed})
		d.focus = false
	}
	for {
		e, ok := d.ed.Update(gtx)
		if !ok {
			break
		}
		if _, ok := e.(widget.SubmitEvent); ok {
			apply()
		}
	}
	message, placeholder := "ファイルのフルパスを入力してください。", "C:\\path\\model.inp"
	if d.textOnly {
		message = "値を入力してEnterまたはOKで確定してください。"
		placeholder = "入力"
	}
	w.modalBox(gtx, d.title, message, func(c layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(c,
			layout.Rigid(material.Editor(w.Theme, &d.ed, placeholder).Layout),
			layout.Rigid(func(c layout.Context) layout.Dimensions {
				return layout.Flex{}.Layout(c,
					layout.Rigid(func(c layout.Context) layout.Dimensions { return w.smallButton(c, "file-ok", "OK", apply) }),
					layout.Rigid(func(c layout.Context) layout.Dimensions {
						return w.smallButton(c, "file-cancel", "Cancel", func() {
							if d.cancel != nil {
								d.cancel()
							}
							w.ui.modal = nil
						})
					}),
				)
			}))
	})
}

// Cancel the interactive producer before clearing selection, so its last step
// cannot reselect an entity after Esc. The worker owns the model lock.
func (w *Window) deselectOnEscape() {
	w.enqueueInterrupting(func() {
		w.cancelCanvasDrag()
		w.stopMouseEdit()
		w.cancelViewAnimation()
		w.Deselect()
		w.Redraw()
	})
}
