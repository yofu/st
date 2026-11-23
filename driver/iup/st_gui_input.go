package stgui

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/yofu/go-iup/iup"
	st "github.com/yofu/st/stlib"
)

// Requests arrive on the pipe while IUP owns the command and its callbacks.
func (r *guiCommandRunner) input(id uint64, in *st.GUICommandInput) st.GUIResponse {
	r.mu.Lock()
	defer r.mu.Unlock()
	j := r.jobs[id]
	if j == nil || in == nil || j.status.Prompt == nil || in.Prompt != j.status.Prompt.ID || j.answer != nil || j.cancelRequested {
		return st.GUIResponse{Error: "no matching pending prompt"}
	}
	if !slices.Contains(j.status.Prompt.Kinds, in.Kind) {
		return st.GUIResponse{Error: "unsupported input for this prompt"}
	}
	switch in.Kind {
	case "point":
		if len(in.Point) != 3 {
			return st.GUIResponse{Error: "point needs three coordinates"}
		}
		for _, v := range in.Point {
			if math.IsInf(v, 0) || math.IsNaN(v) {
				return st.GUIResponse{Error: "invalid point"}
			}
		}
	case "node", "element":
		if in.ID <= 0 {
			return st.GUIResponse{Error: "positive id required"}
		}
	case "choice":
		if !slices.Contains([]string{"1", "2", "3"}, in.Text) {
			return st.GUIResponse{Error: "supply choice 1, 2 or 3 in text"}
		}
	case "file":
		if strings.TrimSpace(in.Text) == "" {
			return st.GUIResponse{Error: "supply a file path"}
		}
	case "key":
		if len([]rune(in.Text)) != 1 && in.Text != "Enter" && in.Text != "Backspace" {
			return st.GUIResponse{Error: "key must be one character, Enter or Backspace"}
		}
	}
	j.answer = in
	j.status.Prompt = nil
	j.status.State = "running"
	return st.GUIResponse{}
}

func (w *guiCommandWindow) beginPrompt(message string, kinds []string) func() {
	r, j := w.runner, w.job
	r.mu.Lock()
	j.nextPrompt++
	previous, previousAnswer := j.status.Prompt, j.answer
	j.status.Prompt = &st.GUIInputPrompt{ID: j.nextPrompt, Message: message, Kinds: kinds}
	j.answer = nil
	j.status.State = "waiting_for_input"
	r.mu.Unlock()
	return func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		j.status.Prompt, j.answer = previous, previousAnswer
		if previous == nil {
			j.status.State = "running"
		} else {
			j.status.State = "waiting_for_input"
		}
	}
}

// Platform-specific dialogs keep their normal controls. Polling the workflow
// tells a remote client exactly when local interaction is needed.
func (stw *Window) popup(dlg *iup.Handle) {
	if w := stw.commandContext; w != nil {
		restore := w.beginPrompt("Complete the native IUP dialog: "+dlg.GetAttribute("TITLE"), []string{})
		defer restore()
	}
	dlg.Popup(iup.CENTER, iup.CENTER)
}

func (w *guiCommandWindow) prompt(message string, kinds ...string) (*st.GUICommandInput, error) {
	r, j := w.runner, w.job
	restore := w.beginPrompt(message, kinds)
	defer restore()
	for {
		r.mu.Lock()
		answer, stop := j.answer, j.cancelRequested
		j.answer = nil
		r.mu.Unlock()
		if stop || w.guiConnection == nil {
			return nil, context.Canceled
		}
		if answer != nil {
			return answer, nil
		}
		if kinds[0] == "node" && !w.canvasInput {
			return nil, nil
		} // User finished on the canvas.
		iup.LoopStep()
		time.Sleep(10 * time.Millisecond)
	}
}

func (stw *Window) setCanvasInput(fn any) error {
	switch cb := fn.(type) {
	case func(*iup.CommonKeyAny):
		stw.canvasKey = cb
	case func(*iup.MouseButton):
		stw.canvasButton = cb
		stw.customButton = true
	}
	stw.canvasInput = stw.commandContext != nil
	return stw.canv.SetCallback(fn)
}

func (w *guiCommandWindow) finishInteraction() error {
	for w.canvasInput {
		in, err := w.prompt(w.cname.GetAttribute("VALUE"), "node", "element", "point", "text", "key", "finish")
		if err != nil {
			w.EscapeCB()
			return err
		}
		if in == nil {
			continue
		}
		if err = w.applyCanvasInput(in); err != nil {
			w.EscapeCB()
			return err
		}
	}
	return nil
}

func (w *guiCommandWindow) applyCanvasInput(in *st.GUICommandInput) error {
	key := func(code int32) { w.canvasKey(&iup.CommonKeyAny{Sender: w.canv, Key: code}) }
	var p []float64
	switch in.Kind {
	case "key":
		switch in.Text {
		case "Enter":
			key(KEY_ENTER)
		case "Backspace":
			key(KEY_BS)
		default:
			key([]rune(in.Text)[0])
		}
		return nil
	case "text":
		w.cline.SetAttribute("VALUE", in.Text)
		key(KEY_ENTER)
		return nil
	case "node", "element":
		f := w.Frame()
		if in.Kind == "node" {
			n := f.Nodes[in.ID]
			if n == nil {
				return fmt.Errorf("unknown node %d", in.ID)
			}
			p = n.Pcoord
		} else {
			el := f.Elems[in.ID]
			if el == nil {
				return fmt.Errorf("unknown element %d", in.ID)
			}
			p = f.View.ProjectCoord(el.MidPoint())
		}
		// Keyboard-only selectors accept an exact ID even for coincident projections.
		if !w.customButton {
			w.cline.SetAttribute("VALUE", strconv.Itoa(in.ID))
			key(KEY_ENTER)
			return nil
		}
		x, y := math.Round(p[0]), math.Round(p[1])
		if in.Kind == "node" && f.PickNode(x, y, nodeSelectPixel) != f.Nodes[in.ID] || in.Kind == "element" && f.PickElem(x, y, dotSelectPixel) != f.Elems[in.ID] {
			return fmt.Errorf("cannot uniquely pick %s %d in the current view", in.Kind, in.ID)
		}
	case "point":
		p = w.Frame().View.ProjectCoord(in.Point)
	case "finish":
		if !w.customButton {
			w.EscapeCB()
			return nil
		}
		w.canvasButton(&iup.MouseButton{Sender: w.canv, Button: BUTTON_RIGHT, Status: "    3     "})
		return nil
	}
	if !w.customButton {
		return fmt.Errorf("this command expects keyboard input")
	}
	x, y := int32(math.Round(p[0])), int32(math.Round(p[1]))
	w.dbuff.UpdateYAxis(&y)
	cb := w.canvasButton
	cb(&iup.MouseButton{Sender: w.canv, Button: BUTTON_LEFT, Pressed: 1, X: x, Y: y, Status: "  1       "})
	if w.canvasInput {
		w.canvasButton(&iup.MouseButton{Sender: w.canv, Button: BUTTON_LEFT, X: x, Y: y, Status: "          "})
	}
	return nil
}

func (stw *Window) commandFile(dir, pattern string, save bool) (string, bool) {
	if w := stw.commandContext; w != nil {
		action := "Open"
		if save {
			action = "Save"
		}
		in, err := w.prompt(action+" "+pattern+" ("+dir+")", "file")
		if err != nil {
			return "", false
		}
		return in.Text, true
	}
	if save {
		return iup.GetSaveFile(dir, pattern)
	}
	return iup.GetOpenFile(dir, pattern)
}

// WaitCommandAnalysis keeps an ex pipeline on its current stage until the
// native solver's preview restoration and completion notification finish.
func (w *guiCommandWindow) WaitCommandAnalysis() error {
	for {
		w.runner.mu.Lock()
		finished, failed, cancelled, msg := w.job.finished, w.job.failed, w.job.cancelRequested, w.job.status.Error
		if finished {
			w.job.finished = false
			w.job.stopAnalysis = nil
		}
		w.runner.mu.Unlock()
		if finished {
			if cancelled {
				return context.Canceled
			}
			if failed {
				return fmt.Errorf("%s", msg)
			}
			return nil
		}
		iup.LoopStep()
		time.Sleep(10 * time.Millisecond)
	}
}
