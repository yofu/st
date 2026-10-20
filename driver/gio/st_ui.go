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
	commandRect          image.Rectangle
	panelRect            image.Rectangle
	menuRect             image.Rectangle
	contextItems         []menuItem
	contextPoint         image.Point
	modal                *pathDialog
	pathRequests         chan *pathDialog
	focusCommand         bool
	status               string
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
	return u
}

func (w *Window) enqueue(job func()) {
	w.enqueueJob(uiJob{run: job})
}
func (w *Window) enqueueInterrupting(job func()) { w.enqueueJob(uiJob{run: job, interrupt: true}) }
func (w *Window) enqueueJob(job uiJob) {
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
	case <-u.done:
		u.pending.Add(-1)
	default:
		u.pending.Add(-1)
		w.History("操作待ちが多いため受け付けられません。少し待って再実行してください。")
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
	style.TextSize = unit.Sp(12)
	style.Inset = layout.Inset{Top: unit.Dp(5), Bottom: unit.Dp(5), Left: unit.Dp(9), Right: unit.Dp(9)}
	style.Background = color.NRGBA{R: 230, G: 234, B: 240, A: 255}
	style.Color = color.NRGBA{R: 25, G: 35, B: 50, A: 255}
	return style.Layout(gtx)
}

func (w *Window) uiLabel(gtx layout.Context, text string) layout.Dimensions {
	s := material.Body2(w.Theme, text)
	s.TextSize = unit.Sp(12)
	return s.Layout(gtx)
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
	paint.Fill(gtx.Ops, color.NRGBA{R: 240, G: 243, B: 247, A: 255})
	menuH := min(gtx.Dp(34), size.Y)
	bottomH := min(gtx.Dp(190), max(0, size.Y-menuH)/3)
	commandH := min(gtx.Dp(36), max(0, size.Y-menuH-bottomH))
	statusH := min(gtx.Dp(24), max(0, size.Y-menuH-bottomH-commandH))
	panelW := min(gtx.Dp(290), size.X/2)
	bodyBottom := max(menuH, size.Y-bottomH-commandH-statusH)
	u.canvasRect = image.Rect(0, menuH, max(0, size.X-panelW), bodyBottom)
	u.panelRect = image.Rect(u.canvasRect.Max.X, menuH, size.X, bodyBottom)
	u.commandRect = image.Rect(0, size.Y-statusH-commandH, size.X, size.Y-statusH)
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
		return layout.Dimensions{Size: c.Constraints.Max}
	})
	w.scheduleViewAnimation(gtx)
	region(body, u.panelRect, w.layoutPanel)
	region(body, image.Rect(0, bodyBottom, size.X, u.commandRect.Min.Y), w.layoutHistory)
	region(body, u.commandRect, w.layoutCommand)
	region(body, image.Rect(0, size.Y-statusH, size.X, size.Y), w.layoutStatus)
	if u.menu >= 0 || len(u.contextItems) > 0 {
		w.layoutMenuPopup(gtx)
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

func (w *Window) keyboard(gtx layout.Context) {
	u := w.ui
	if u.focusCommand {
		gtx.Execute(key.FocusCmd{Tag: &u.command})
		u.focusCommand = false
	}
	for {
		ev, ok := gtx.Event(
			key.Filter{Focus: &u.command, Name: key.NameUpArrow}, key.Filter{Focus: &u.command, Name: key.NameDownArrow},
			key.Filter{Focus: &u.command, Name: key.NameTab, Optional: key.ModShift}, key.Filter{Name: key.NameEscape},
			key.Filter{Focus: &u.command, Name: ";", Optional: key.ModShift}, key.Filter{Focus: &u.command, Name: ":", Optional: key.ModShift},
			key.Filter{Focus: &u.command, Name: key.NameSpace, Optional: key.ModShift},
			key.Filter{Focus: &u.command, Name: "P", Required: key.ModCtrl}, key.Filter{Focus: &u.command, Name: "N", Required: key.ModCtrl},
			key.Filter{Focus: &u.command, Name: "[", Required: key.ModCtrl},
			key.Filter{Focus: &u.command, Name: "A", Required: key.ModCtrl}, key.Filter{Focus: &u.command, Name: "E", Required: key.ModCtrl}, key.Filter{Focus: &u.command, Name: "X", Required: key.ModCtrl},
			key.Filter{Name: "I", Required: key.ModCtrl}, key.Filter{Name: "M", Required: key.ModCtrl}, key.Filter{Name: "R", Required: key.ModCtrl},
			key.Filter{Name: "O", Required: key.ModCtrl}, key.Filter{Name: "S", Required: key.ModCtrl},
			key.Filter{Name: "L", Required: key.ModCtrl},
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
		case key.NameUpArrow, "P":
			w.browseHistory(true)
		case key.NameDownArrow, "N":
			w.browseHistory(false)
		case key.NameTab:
			if k.Modifiers&key.ModShift != 0 {
				w.completeCommandDirection(-1)
			} else {
				w.completeCommand()
			}
		case "I":
			w.fileDialog("Insert")
		case "A":
			w.menuAction("showall")
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
			gtx.Execute(key.FocusCmd{Tag: &u.command})
		case key.NameEscape:
			u.mu.Lock()
			q := u.question
			u.mu.Unlock()
			if q != nil {
				w.answerQuestion(q, len(q.choices))
				continue
			}
			if u.modal != nil && u.modal.cancel != nil {
				u.modal.cancel()
			}
			u.modal = nil
			u.menu = -1
			u.contextItems = nil
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
	return layout.Inset{Left: unit.Dp(8), Right: unit.Dp(8), Top: unit.Dp(3), Bottom: unit.Dp(3)}.Layout(gtx, func(c layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.Middle}.Layout(c,
			layout.Rigid(func(c layout.Context) layout.Dimensions { return w.uiLabel(c, "Command >  ") }),
			layout.Flexed(1, func(c layout.Context) layout.Dimensions {
				return material.Editor(w.Theme, &u.command, ":command / 'figure command").Layout(c)
			}),
			layout.Rigid(func(c layout.Context) layout.Dimensions {
				return w.smallButton(c, "execute", "Run", func() { w.submitCommand(u.command.Text()) })
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
	return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(c layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(c,
			layout.Rigid(func(c layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(c,
					layout.Flexed(1, func(c layout.Context) layout.Dimensions { return w.uiLabel(c, "History / 解析ログ") }),
					layout.Rigid(func(c layout.Context) layout.Dimensions {
						return w.smallButton(c, "copy-history", "Copy", func() {
							c.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(strings.NewReader(text))})
						})
					}),
					layout.Rigid(func(c layout.Context) layout.Dimensions {
						return w.smallButton(c, "clear-history", "Clear", func() { u.mu.Lock(); u.log = ""; u.mu.Unlock() })
					}),
				)
			}),
			layout.Flexed(1, func(c layout.Context) layout.Dimensions {
				e := material.Editor(w.Theme, &u.history, "")
				e.TextSize = unit.Sp(12)
				return e.Layout(c)
			}),
		)
	})
}

func (w *Window) layoutStatus(gtx layout.Context) layout.Dimensions {
	u := w.ui
	u.mu.Lock()
	progress := u.progress
	u.mu.Unlock()
	if u.pending.Load() > 0 {
		progress = "Working...  " + progress
	}
	return layout.Inset{Left: unit.Dp(8), Top: unit.Dp(3)}.Layout(gtx, func(c layout.Context) layout.Dimensions { return w.uiLabel(c, u.status+"   "+progress) })
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
