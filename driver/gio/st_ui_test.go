package stgio

import (
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	st "github.com/yofu/st/stlib"
)

type uiHarness struct {
	w      *Window
	router input.Router
	ops    op.Ops
	size   image.Point
	metric unit.Metric
	now    time.Time
}

func newUIHarness(t *testing.T) *uiHarness {
	t.Helper()
	h := &uiHarness{w: NewWindow(nil), size: image.Pt(1200, 900), now: time.Unix(1, 0)}
	t.Cleanup(h.w.ui.close)
	t.Cleanup(h.w.stopInteractiveCommand)
	h.frame()
	return h
}
func (h *uiHarness) frame() {
	h.ops.Reset()
	h.w.Layout(layout.Context{Ops: &h.ops, Source: h.router.Source(), Metric: h.metric, Now: h.now, Constraints: layout.Exact(h.size)})
	h.router.Frame(&h.ops)
}
func (h *uiHarness) click(x, y float32) {
	h.router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(x, y)})
	h.frame()
	h.router.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: f32.Pt(x, y)})
	h.frame()
}
func (h *uiHarness) key(name key.Name) {
	h.router.Queue(key.Event{Name: name, State: key.Press})
	h.frame()
}
func (h *uiHarness) focus(editor any) {
	h.router.Source().Execute(key.FocusCmd{Tag: editor})
	h.frame()
	h.frame()
}
func waitUI(t *testing.T, w *Window) {
	t.Helper()
	done := make(chan struct{})
	w.enqueue(func() { close(done) })
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("command queue blocked")
	}
	w.ui.modelMu.Lock()
	w.ui.modelMu.Unlock()
}

func TestGioCommandEditorAndHistory(t *testing.T) {
	h := newUIHarness(t)
	w := h.w
	h.focus(&w.ui.command)
	h.router.Queue(key.EditEvent{Text: "'gfact 2"})
	h.frame()
	h.key(key.NameReturn)
	waitUI(t, w)
	h.frame()
	w.ui.modelMu.Lock()
	gfact := w.frame.View.Gfact
	w.ui.modelMu.Unlock()
	if gfact != 2 {
		t.Fatalf("Enter did not execute figure command: gfact=%g, text=%q", gfact, w.ui.command.Text())
	}
	if w.ui.command.Text() != "" {
		t.Fatal("command not cleared")
	}
	h.key(key.NameUpArrow)
	if got := w.ui.command.Text(); got != "'gfact 2" {
		t.Fatalf("history up: %q", got)
	}
	h.key(key.NameDownArrow)
	if w.ui.command.Text() != "" {
		t.Fatal("history down did not restore draft")
	}
	w.ui.command.SetText(":timeh")
	h.key(key.NameTab)
	if got := w.ui.command.Text(); got != ":timehistory" {
		t.Fatalf("Tab completion: %q", got)
	}
	w.ui.mu.Lock()
	log := w.ui.log
	w.ui.mu.Unlock()
	if !strings.Contains(log, "> 'gfact 2") {
		t.Fatalf("command missing from history: %s", log)
	}
}

func TestGioMenuAndSettingEvents(t *testing.T) {
	h := newUIHarness(t)
	w := h.w
	h.click(20, 17)
	if w.ui.menu != 0 {
		t.Fatalf("File click did not open menu: %d", w.ui.menu)
	}
	w.ui.button("menu-item-0-1").Click()
	h.frame()
	if w.ui.modal == nil {
		t.Fatal("Open item did not show path dialog")
	}
	h.key(key.NameEscape)
	if w.ui.modal != nil {
		t.Fatal("Escape did not close file dialog")
	}
	w.ui.button("tab-Show").Click()
	h.frame()
	if w.ui.tab != 1 {
		t.Fatal("Show tab")
	}
	field := w.ui.fields["DFACT"]
	if field == nil {
		t.Fatal("DFACT editor not laid out")
	}
	h.focus(&field.ed)
	field.ed.SetText("250")
	h.key(key.NameReturn)
	waitUI(t, w)
	h.frame()
	w.ui.modelMu.Lock()
	dfact := w.frame.Show.Dfact
	w.ui.modelMu.Unlock()
	if dfact != 250 {
		t.Fatalf("DFACT Enter: %g", dfact)
	}
	w.menuAction("cmd::view front")
	waitUI(t, w)
	finishViewAnimation(h)
	w.ui.modelMu.Lock()
	angle := append([]float64(nil), w.frame.View.Angle...)
	w.ui.modelMu.Unlock()
	if angle[0] != 0 || angle[1] != -90 {
		t.Fatalf("View Front: %v", angle)
	}
}

func TestGioSettingsValidationAndFlags(t *testing.T) {
	w := NewWindow(nil)
	defer w.ui.close()
	for _, tc := range []struct{ name, text string }{{"DFACT", "NaN"}, {"PHI", "Inf"}, {"GFACT", "0"}, {"XMIN", "2000"}, {"DFACT", "-1"}} {
		p := w.numericSetting(tc.name)
		old := *p
		if err := w.applySetting(tc.name, tc.text); err == nil {
			t.Errorf("accepted %s=%s", tc.name, tc.text)
		}
		if *p != old {
			t.Errorf("invalid input changed %s", tc.name)
		}
	}
	if err := w.applySetting("PERIOD", "th@10"); err != nil || w.frame.Show.Period != "TH@10" {
		t.Fatal("period")
	}
	for _, name := range []string{"DEFORMATION", "NC/0", "EC/1", "STRESS/1/2", "ETYPE/1", "SECT/101"} {
		w.applyFlag(name, true)
		if !w.flagValue(name) {
			t.Errorf("flag not enabled: %s", name)
		}
		w.applyFlag(name, false)
		if w.flagValue(name) {
			t.Errorf("flag not disabled: %s", name)
		}
	}
}

func TestGioQuestionKeepsUIResponsive(t *testing.T) {
	h := newUIHarness(t)
	w := h.w
	answer := make(chan bool, 1)
	w.enqueue(func() { answer <- w.Yn("Confirm", "実行しますか？") })
	var q *uiQuestion
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		w.ui.mu.Lock()
		q = w.ui.question
		w.ui.mu.Unlock()
		if q != nil {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if q == nil {
		t.Fatal("no question")
	}
	h.frame() // Model lock is held by the worker; the cached canvas must still lay out.
	w.ui.button("answer-0").Click()
	h.frame()
	select {
	case yes := <-answer:
		if !yes {
			t.Fatal("Yes answered false")
		}
	case <-time.After(time.Second):
		t.Fatal("answer blocked")
	}
	waitUI(t, w)
}

func TestGioHistoryWriterConcurrentAndBounded(t *testing.T) {
	w := NewWindow(nil)
	defer w.ui.close()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Fprintf(w.HistoryWriter(), "解析ログ %d\n", i)
			w.CurrentLap("Working", i, 50)
		}()
	}
	wg.Wait()
	w.ui.mu.Lock()
	log := w.ui.log
	w.ui.mu.Unlock()
	for i := 0; i < 50; i++ {
		if !strings.Contains(log, fmt.Sprintf("解析ログ %d\n", i)) {
			t.Errorf("missing log %d", i)
		}
	}
	w.appendLog(strings.Repeat("階", 100000))
	w.ui.mu.Lock()
	log = w.ui.log
	w.ui.mu.Unlock()
	if len(log) > 256*1024 || !utf8.ValidString(log) {
		t.Fatal("history bound or UTF-8")
	}
}

func TestGioCanvasOffsetAndSelection(t *testing.T) {
	h := newUIHarness(t)
	w := h.w
	// Frame projections are canvas-local; make one node visible in a known place.
	n := st.NewNode()
	n.Num = 101
	n.Frame = w.frame
	n.Coord = []float64{0, 0, 0}
	w.frame.Nodes[n.Num] = n
	w.frame.View.Perspective = false
	w.frame.View.Center = []float64{200, 180}
	w.Redraw()
	h.frame()
	x, y := n.Pcoord[0], n.Pcoord[1]+float64(w.ui.canvasRect.Min.Y)
	h.router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Modifiers: key.ModAlt, Position: f32.Pt(float32(x), float32(y))})
	h.frame()
	h.router.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Modifiers: key.ModAlt, Position: f32.Pt(float32(x), float32(y))})
	h.frame()
	if len(w.SelectedNodes()) != 1 || w.SelectedNodes()[0] != n {
		t.Fatal("node not picked with canvas offset")
	}
	if math.Abs(float64(w.end.Y)-n.Pcoord[1]) > 1 {
		t.Fatalf("pointer is not canvas-local: %v vs %v", w.end, n.Pcoord)
	}
	h.size = image.Pt(650, 430)
	h.frame()
	if w.canvasSize != w.ui.canvasRect.Size() {
		t.Fatal("canvas includes sidebar or menu size")
	}
}

func TestGioSaveAsAndOverwriteCancel(t *testing.T) {
	h := newUIHarness(t)
	w := h.w
	path := filepath.Join(t.TempDir(), "保存先 with spaces.inp")
	w.fileDialog("Save As")
	w.ui.modal.apply(path)
	w.ui.modal = nil
	waitUI(t, w)
	h.frame()
	original, err := os.ReadFile(path)
	if err != nil || len(original) == 0 {
		t.Fatalf("Save As: %v", err)
	}
	w.ui.modelMu.Lock()
	savedPath := w.frame.Path
	w.ui.modelMu.Unlock()
	if savedPath != path {
		t.Fatalf("Save As did not rebase path: %s", savedPath)
	}
	w.fileDialog("Save As")
	w.ui.modal.apply(path)
	w.ui.modal = nil
	var q *uiQuestion
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		w.ui.mu.Lock()
		q = w.ui.question
		w.ui.mu.Unlock()
		if q != nil {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if q == nil {
		t.Fatal("overwrite confirmation missing")
	}
	h.frame()
	w.answerQuestion(q, 2)
	waitUI(t, w)
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(original) {
		t.Fatal("cancelled overwrite changed file")
	}
}

func TestGioCanvasCacheAndPanelIsolation(t *testing.T) {
	h := newUIHarness(t)
	w := h.w
	revision := w.ui.canvasRevision
	w.History("履歴だけ更新")
	h.frame()
	w.ui.command.SetText(":analysis")
	h.frame()
	if w.ui.canvasRevision != revision {
		t.Fatal("history/command typing redrew model")
	}
	// The first checkbox is Perspective. Its events must not rotate or select the model.
	w.ui.button("tab-View").Click()
	h.frame()
	before := append([]float64(nil), w.frame.View.Angle...)
	h.click(float32(w.ui.panelRect.Min.X+20), 142)
	waitUI(t, w)
	h.frame()
	w.ui.modelMu.Lock()
	perspective := w.frame.View.Perspective
	angle := append([]float64(nil), w.frame.View.Angle...)
	w.ui.modelMu.Unlock()
	if perspective {
		t.Fatal("checkbox did not update projection")
	}
	if angle[0] != before[0] || angle[1] != before[1] || w.selecting {
		t.Fatal("panel event reached canvas")
	}
	if w.ui.canvasRevision <= revision {
		t.Fatal("setting change did not redraw model")
	}
}
