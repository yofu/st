package stgio

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	st "github.com/yofu/st/stlib"
)

// SetViewerBrowserAction connects the shared command dispatcher to browser UI.
func (w *Window) SetViewerBrowserAction(action func(string, []string)) {
	w.viewer.browserAction = action
}

// OpenBrowserArtifact is used by the shared dispatcher for operations that
// need a browser equivalent of a desktop window or application.
func (w *Window) OpenBrowserArtifact(action string, paths ...string) bool {
	if w.viewer == nil {
		return false
	}
	if action == "fig2-help" {
		w.History("コマンドの引数は画面下のコマンド一覧と履歴から確認できます。")
		return true
	}
	if action != "fig2print" && w.viewer.browserAction != nil {
		w.viewer.browserAction(action, paths)
	}
	return true
}

type viewerFont struct {
	face  string
	size  int
	color int
}

func (f *viewerFont) Face() string        { return f.face }
func (f *viewerFont) SetFace(face string) { f.face = face }
func (f *viewerFont) Size() int           { return f.size }
func (f *viewerFont) SetSize(size int)    { f.size = size }
func (f *viewerFont) Color() int          { return f.color }
func (f *viewerFont) SetColor(color int)  { f.color = color }

func (w *Window) viewerTextBox(name string) *st.TextBox {
	if box := w.textBox[name]; box != nil {
		return box
	}
	box := st.NewTextBox(&viewerFont{size: 12})
	box.SetPosition(12, float64(w.canvasSize.Y)-12-30*float64(len(w.textBox)))
	w.textBox[name] = box
	return box
}

func (w *Window) viewerAnnotations() string {
	type annotation struct {
		X, Y float64
		Text string
	}
	var result []annotation
	names := make([]string, 0, len(w.textBox))
	for name := range w.textBox {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		box := w.textBox[name]
		if box.IsHidden(w.frame.Show) {
			continue
		}
		x, y := box.Position()
		scale := math.Max(1, w.viewer.scale)
		result = append(result, annotation{x / scale, (float64(w.canvasSize.Y) - y) / scale, strings.Join(box.Text(), "\n")})
	}
	if len(result) == 0 {
		return ""
	}
	data, err := json.Marshal(result)
	if err != nil {
		w.History(fmt.Sprint(err))
		return ""
	}
	return string(data)
}

func viewerQuestionJSON(id int, question *uiQuestion) string {
	info := struct {
		ID             int
		Title, Message string
		Choices        []string
	}{id, question.title, question.message, question.choices}
	data, _ := json.Marshal(info)
	return string(data)
}

// ReplyViewerQuestion resolves a shared ex command confirmation in HTML.
func (w *Window) ReplyViewerQuestion(id, choice int) bool {
	w.ui.mu.Lock()
	q := w.ui.question
	if w.viewer == nil || q == nil || w.viewer.questionID != id ||
		w.viewer.question != q || choice < 1 || choice > len(q.choices) {
		w.ui.mu.Unlock()
		return false
	}
	w.ui.mu.Unlock()
	w.answerQuestion(q, choice)
	return true
}
