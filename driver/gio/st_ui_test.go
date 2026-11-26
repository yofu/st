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
	h.click(20, 13)
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
	x, y := n.Pcoord[0]+float64(w.ui.canvasRect.Min.X), n.Pcoord[1]+float64(w.ui.canvasRect.Min.Y)
	h.router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Modifiers: key.ModAlt, Position: f32.Pt(float32(x), float32(y))})
	h.frame()
	h.router.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Modifiers: key.ModAlt, Position: f32.Pt(float32(x), float32(y))})
	h.frame()
	if len(w.SelectedNodes()) != 1 || w.SelectedNodes()[0] != n {
		t.Fatal("node not picked with canvas offset")
	}
	if math.Abs(float64(w.end.X)-n.Pcoord[0]) > 1 || math.Abs(float64(w.end.Y)-n.Pcoord[1]) > 1 {
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
	// Toggle Perspective via the left SideBar text toggle; panel events must not rotate or select the model.
	w.ui.button("tab-View").Click()
	h.frame()
	before := append([]float64(nil), w.frame.View.Angle...)
	persBtn := w.ui.buttons["toggle-PERSPECTIVE"]
	if persBtn == nil {
		t.Fatal("PERSPECTIVE toggle not laid out")
	}
	h.click(float32(w.ui.panelRect.Min.X+20), float32(w.ui.panelRect.Min.Y+88))
	if w.ui.flags["PERSPECTIVE"] {
		// In environments with fallback font metrics, click the widget directly while still verifying click isolation.
		persBtn.Click()
		h.frame()
	}
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

func TestGioIUPLikeLayoutRegions(t *testing.T) {
	h := newUIHarness(t)
	u := h.w.ui
	// SideBar spans full height on the left below the menu bar.
	if u.panelRect.Min.X != 0 || u.panelRect.Max.Y != h.size.Y || u.panelRect.Dx() <= 0 {
		t.Fatalf("left full-height panel expected, got %v in %v", u.panelRect, h.size)
	}
	// Right column stacks Canvas -> History -> Command cleanly to the right of SideBar.
	if u.canvasRect.Min.X != u.panelRect.Max.X || u.canvasRect.Max.X != h.size.X {
		t.Fatalf("canvas should sit to the right of panel: canvas=%v panel=%v", u.canvasRect, u.panelRect)
	}
	if u.historyRect.Min.X != u.panelRect.Max.X || u.historyRect.Min.Y != u.canvasRect.Max.Y || u.historyRect.Max.X != h.size.X {
		t.Fatalf("history should sit below canvas in right column: history=%v canvas=%v", u.historyRect, u.canvasRect)
	}
	if u.commandRect.Min.X != u.panelRect.Max.X || u.commandRect.Min.Y != u.historyRect.Max.Y || u.commandRect.Max.Y != h.size.Y || u.commandRect.Max.X != h.size.X {
		t.Fatalf("command bar should sit below history at bottom of right column: command=%v history=%v", u.commandRect, u.historyRect)
	}
	if got := h.w.currentModeName(); got != "SELECT" {
		t.Fatalf("default mode name = %q, want SELECT", got)
	}
	if h.w.CanvasRotateSpeedX() != 0.2 || h.w.CanvasRotateSpeedY() != 0.2 || h.w.CanvasMoveSpeedX() != 1.0 || h.w.CanvasMoveSpeedY() != 1.0 {
		t.Fatalf("unexpected default drag speeds: rot=(%g,%g) move=(%g,%g)", h.w.CanvasRotateSpeedX(), h.w.CanvasRotateSpeedY(), h.w.CanvasMoveSpeedX(), h.w.CanvasMoveSpeedY())
	}
}

func TestGioCtrlTabSwitchesSideBarTabs(t *testing.T) {
	h := newUIHarness(t)
	w := h.w
	if w.ui.tab != 0 {
		t.Fatalf("initial tab = %d, want 0", w.ui.tab)
	}
	send := func(mods key.Modifiers) {
		h.router.Queue(key.Event{Name: key.NameTab, Modifiers: mods, State: key.Press})
		h.frame()
	}
	// Cycle View -> Show -> Property -> View while command editor has focus.
	h.focus(&w.ui.command)
	for _, want := range []int{1, 2, 0} {
		send(key.ModCtrl)
		if w.ui.tab != want {
			t.Fatalf("Ctrl+Tab got tab %d, want %d", w.ui.tab, want)
		}
	}
	// Reverse cycle View -> Property -> Show -> View with Ctrl+Shift+Tab.
	for _, want := range []int{2, 1, 0} {
		send(key.ModCtrl | key.ModShift)
		if w.ui.tab != want {
			t.Fatalf("Ctrl+Shift+Tab got tab %d, want %d", w.ui.tab, want)
		}
	}
}

func TestGioPropertyTabEditSectionAndEtype(t *testing.T) {
	h := newUIHarness(t)
	w := h.w
	f := w.frame
	sec101 := f.AddSect(101)
	sec101.Name = "C1"
	sec201 := f.AddSect(201)
	sec201.Name = "G1"
	sec301 := f.AddSect(301)
	sec301.Name = "B1"
	n1 := f.AddNode(0, 0, 0)
	n2 := f.AddNode(0, 0, 3)
	n3 := f.AddNode(5, 0, 3)
	e1 := f.AddLineElem(-1, []*st.Node{n1, n2}, sec101, st.COLUMN)
	e2 := f.AddLineElem(-1, []*st.Node{n2, n3}, sec201, st.GIRDER)
	eLocked := f.AddLineElem(-1, []*st.Node{n1, n3}, sec101, st.BRACE)
	eLocked.Lock = true
	st.Snapshot(w)

	w.SelectElem([]*st.Elem{e1, e2, eLocked})
	w.ui.tab = 2
	h.frame()

	// Mixed selection should show mixed state without overwriting.
	if !w.ui.propSectMixed || !w.ui.propEtypeMixed {
		t.Fatalf("expected mixed selection state: sectMixed=%v etypeMixed=%v", w.ui.propSectMixed, w.ui.propEtypeMixed)
	}

	// 1) Edit SECTION by text input + Enter: updates unlocked elements only, supports Undo.
	sectField := w.ui.fields["PROP_SECTION"]
	if sectField == nil {
		t.Fatal("PROP_SECTION editor not laid out")
	}
	h.focus(&sectField.ed)
	sectField.ed.SetText("301")
	h.key(key.NameReturn)
	waitUI(t, w)
	h.frame()
	if e1.Sect != sec301 || e2.Sect != sec301 {
		t.Fatalf("unlocked sections not updated: e1=%v e2=%v", e1.Sect.Num, e2.Sect.Num)
	}
	if eLocked.Sect != sec101 {
		t.Fatalf("locked element section changed: %v", eLocked.Sect.Num)
	}
	if !w.IsChanged() {
		t.Fatal("window not marked changed after section edit")
	}

	// 2) Edit ETYPE by abbreviated text input ("gi" -> GIRDER).
	etypeField := w.ui.fields["PROP_ETYPE"]
	if etypeField == nil {
		t.Fatal("PROP_ETYPE editor not laid out")
	}
	h.focus(&etypeField.ed)
	etypeField.ed.SetText("gi")
	h.key(key.NameReturn)
	waitUI(t, w)
	h.frame()
	if e1.Etype != st.GIRDER || e2.Etype != st.GIRDER || eLocked.Etype != st.BRACE {
		t.Fatalf("etype text edit failed: e1=%d e2=%d locked=%d", e1.Etype, e2.Etype, eLocked.Etype)
	}

	// 3) Reject incompatible plate ETYPE ("wall") on line elements.
	if err := w.applySelectedEtype("wall"); err == nil {
		t.Fatal("expected error applying WALL to line elements")
	}
	if e1.Etype != st.GIRDER {
		t.Fatalf("failed ETYPE edit modified element: %d", e1.Etype)
	}

	// 4) 1-click ETYPE button (BRACE = 3).
	w.ui.button(fmt.Sprintf("prop-etype-%d", st.BRACE)).Click()
	h.frame()
	waitUI(t, w)
	h.frame()
	if e1.Etype != st.BRACE || e2.Etype != st.BRACE {
		t.Fatalf("1-click ETYPE button failed: e1=%d e2=%d", e1.Etype, e2.Etype)
	}

	// 5) Section picker popup: open, press Escape (closes picker without deselecting), reopen and pick 201.
	w.ui.button("prop-section-picker").Click()
	h.frame()
	if !w.ui.sectionPickerOpen {
		t.Fatal("section picker did not open")
	}
	h.key(key.NameEscape)
	if w.ui.sectionPickerOpen {
		t.Fatal("Escape did not close section picker")
	}
	if len(w.SelectedElems()) != 3 {
		t.Fatalf("Escape while closing section picker deselected elements: %d", len(w.SelectedElems()))
	}
	w.ui.button("prop-section-picker").Click()
	h.frame()
	w.ui.button("prop-section-pick-201").Click()
	h.frame()
	waitUI(t, w)
	h.frame()
	if w.ui.sectionPickerOpen {
		t.Fatal("picking a section did not close popup")
	}
	if e1.Sect != sec201 || e2.Sect != sec201 {
		t.Fatalf("section picker did not apply section 201: e1=%v e2=%v", e1.Sect.Num, e2.Sect.Num)
	}

	// 6) Escape while editing PROP_SECTION reverts text without deselecting.
	h.focus(&sectField.ed)
	sectField.ed.SetText("999")
	sectField.dirty = true
	h.key(key.NameEscape)
	if len(w.SelectedElems()) != 3 {
		t.Fatalf("Escape during property edit deselected elements: %d", len(w.SelectedElems()))
	}
	if sectField.dirty {
		t.Fatal("Escape did not clear property field dirty flag")
	}

	// 7) Undo restores previous state before section picker change.
	prev, err := w.Undo()
	if err != nil || prev == nil {
		t.Fatalf("Undo failed after property edit: %v", err)
	}
	if prev.Elems[e1.Num].Sect.Num != 301 {
		t.Fatalf("Undo did not restore section 301: got %d", prev.Elems[e1.Num].Sect.Num)
	}
}

func TestGioShowTabNodeCaptionBitsAndTextToggles(t *testing.T) {
	h := newUIHarness(t)
	w := h.w
	w.ui.tab = 1
	// Expand window height so all Show tab rows (including NODE CAPTION and ETYPE) are laid out.
	h.size = image.Pt(1200, 1800)
	h.frame()

	// Verify dZ (NC/5 -> NC_DZ), tX (NC/6 -> NC_TX), Rz (NC/11 -> NC_RZ), Mx (NC/12 -> NC_MX) map to exact bits.
	for _, tc := range []struct {
		key  string
		bit  uint
		name string
	}{
		{"NC/3", st.NC_DX, "dX"},
		{"NC/5", st.NC_DZ, "dZ"},
		{"NC/6", st.NC_TX, "tX"},
		{"NC/9", st.NC_RX, "Rx"},
		{"NC/11", st.NC_RZ, "Rz"},
		{"NC/12", st.NC_MX, "Mx"},
	} {
		btn := w.ui.buttons["toggle-"+tc.key]
		if btn == nil {
			t.Fatalf("%s (%s) text toggle not laid out", tc.name, tc.key)
		}
		btn.Click()
		h.frame()
		waitUI(t, w)
		h.frame()
		if w.frame.Show.NodeCaption&tc.bit == 0 {
			t.Fatalf("clicking %s (%s) did not set bit %#x (got %#x)", tc.name, tc.key, tc.bit, w.frame.Show.NodeCaption)
		}
	}

	// Verify ETYPE and STRESS text toggles work without checkboxes.
	if len(w.ui.checks) != 0 {
		t.Fatalf("expected no checkboxes in SideBar, got %d", len(w.ui.checks))
	}
	colBefore := w.frame.Show.Etype[st.COLUMN]
	colBtn := w.ui.buttons[fmt.Sprintf("toggle-ETYPE/%d", st.COLUMN)]
	if colBtn == nil {
		t.Fatal("COLUMN text toggle not laid out")
	}
	colBtn.Click()
	h.frame()
	waitUI(t, w)
	h.frame()
	if w.frame.Show.Etype[st.COLUMN] == colBefore {
		t.Fatal("clicking COLUMN text toggle did not toggle Show.Etype[COLUMN]")
	}

	// Verify IUP-compatible ELEM CAPTION RATE master toggle and L/S/Q/M toggles update ColorMode.
	rateBtn := w.ui.buttons["toggle-SRCAN_RATE"]
	if rateBtn == nil {
		t.Fatal("SRCAN_RATE toggle not laid out")
	}
	rateBtn.Click()
	h.frame()
	waitUI(t, w)
	h.frame()
	if w.frame.Show.SrcanRate == 0 || w.frame.Show.ColorMode != st.ECOLOR_RATE {
		t.Fatalf("RATE on failed: SrcanRate=%#x ColorMode=%d", w.frame.Show.SrcanRate, w.frame.Show.ColorMode)
	}
	rateBtn.Click()
	h.frame()
	waitUI(t, w)
	h.frame()
	if w.frame.Show.SrcanRate != 0 || w.frame.Show.ColorMode != w.DefaultColorMode() {
		t.Fatalf("RATE off failed: SrcanRate=%#x ColorMode=%d", w.frame.Show.SrcanRate, w.frame.Show.ColorMode)
	}
	rateLBtn := w.ui.buttons["toggle-RATE/0"]
	if rateLBtn == nil {
		t.Fatal("RATE/0 (L) toggle not laid out")
	}
	rateLBtn.Click()
	h.frame()
	waitUI(t, w)
	h.frame()
	if w.frame.Show.SrcanRate != st.SRCAN_L || !w.ui.flags["SRCAN_RATE"] || w.frame.Show.ColorMode != st.ECOLOR_RATE {
		t.Fatalf("RATE/0 on failed: SrcanRate=%#x master=%v ColorMode=%d", w.frame.Show.SrcanRate, w.ui.flags["SRCAN_RATE"], w.frame.Show.ColorMode)
	}
	rateLBtn.Click()
	h.frame()
	waitUI(t, w)
	h.frame()
	if w.frame.Show.SrcanRate != 0 || w.ui.flags["SRCAN_RATE"] || w.frame.Show.ColorMode != w.DefaultColorMode() {
		t.Fatalf("RATE/0 off failed: SrcanRate=%#x master=%v ColorMode=%d", w.frame.Show.SrcanRate, w.ui.flags["SRCAN_RATE"], w.frame.Show.ColorMode)
	}
}

func TestGioPropertySectionFilteringAndFixedBraceReadOnly(t *testing.T) {
	h := newUIHarness(t)
	w := h.w
	f := w.frame
	for _, s := range []struct {
		id   int
		name string
	}{
		{50, "OTHER"},
		{101, "C1"},
		{650, "B1"},
		{701, "W150"},
		{850, "S150"},
		{901, "SPRING"},
	} {
		sec := f.AddSect(s.id)
		sec.Name = s.name
	}
	n1 := f.AddNode(0, 0, 0)
	n2 := f.AddNode(0, 0, 3)
	n3 := f.AddNode(4, 0, 3)
	n4 := f.AddNode(4, 0, 0)
	lineEl := f.AddLineElem(-1, []*st.Node{n1, n2}, f.Sects[101], st.COLUMN)
	plateEl := f.AddPlateElem(-1, []*st.Node{n1, n2, n3, n4}, f.Sects[701], st.WALL)
	wbraceEl := f.AddLineElem(-1, []*st.Node{n1, n3}, f.Sects[701], st.WBRACE)
	sbraceEl := f.AddLineElem(-1, []*st.Node{n2, n4}, f.Sects[850], st.SBRACE)

	// 1) Line element: section picker candidates limited to 100..699.
	w.SelectElem([]*st.Elem{lineEl})
	w.ui.tab = 2
	h.frame()
	gotLine := w.sectionPickerCandidates()
	if len(gotLine) != 2 || gotLine[0] != 101 || gotLine[1] != 650 {
		t.Fatalf("line section candidates = %v, want [101 650]", gotLine)
	}

	// 2) Plate element: section picker candidates limited to 700..899.
	w.SelectElem([]*st.Elem{plateEl})
	h.frame()
	gotPlate := w.sectionPickerCandidates()
	if len(gotPlate) != 2 || gotPlate[0] != 701 || gotPlate[1] != 850 {
		t.Fatalf("plate section candidates = %v, want [701 850]", gotPlate)
	}

	// 3) WBRACE / SBRACE: read-only display, cannot modify SECTION or ETYPE.
	for _, el := range []*st.Elem{wbraceEl, sbraceEl} {
		w.SelectElem([]*st.Elem{el})
		h.frame()
		if w.ui.propEditable != 0 || w.ui.propFixedBrace != 1 {
			t.Fatalf("fixed brace %s should be read-only: editable=%d fixedBrace=%d", st.ETYPES[el.Etype], w.ui.propEditable, w.ui.propFixedBrace)
		}
		if got := w.sectionPickerCandidates(); len(got) != 0 {
			t.Fatalf("fixed brace should have no picker candidates, got %v", got)
		}
		origSect, origEtype := el.Sect, el.Etype
		if err := w.applySelectedSection("101"); err == nil {
			t.Fatalf("expected error modifying section on %s", st.ETYPES[origEtype])
		}
		if err := w.applySelectedEtype("column"); err == nil {
			t.Fatalf("expected error modifying etype on %s", st.ETYPES[origEtype])
		}
		if el.Sect != origSect || el.Etype != origEtype {
			t.Fatalf("fixed brace modified: sect=%v etype=%d", el.Sect.Num, el.Etype)
		}
	}
}

func TestGioIUPShortcutsAndAxisRangeCommands(t *testing.T) {
	h := newUIHarness(t)
	w := h.w
	f := w.frame
	sec := f.AddSect(101)
	sec.Name = "C1"
	n0 := f.AddNode(0, 0, 0)
	n1 := f.AddNode(4, 0, 0)
	n2 := f.AddNode(0, 0, 3)
	n3 := f.AddNode(4, 0, 3)
	n4 := f.AddNode(0, 0, 6)
	eGround := f.AddLineElem(-1, []*st.Node{n0, n1}, sec, st.GIRDER)
	eCol1 := f.AddLineElem(-1, []*st.Node{n0, n2}, sec, st.COLUMN)
	eFloor1 := f.AddLineElem(-1, []*st.Node{n2, n3}, sec, st.GIRDER)
	eCol2 := f.AddLineElem(-1, []*st.Node{n2, n4}, sec, st.COLUMN)
	f.Ai.Boundary = []float64{0, 3, 6}
	st.Snapshot(w)

	sendCtrl := func(name key.Name, mods key.Modifiers) {
		h.router.Queue(key.Event{Name: name, Modifiers: mods, State: key.Press})
		h.frame()
		waitUI(t, w)
		h.frame()
	}

	// 1) Ctrl+D isolates selected element and updates Show range.
	w.SelectElem([]*st.Elem{eGround})
	h.frame()
	sendCtrl("D", key.ModCtrl)
	if eGround.IsHidden(f.Show) || !eCol1.IsHidden(f.Show) || !eFloor1.IsHidden(f.Show) {
		t.Fatalf("Ctrl+D did not isolate eGround: ground=%v col1=%v floor1=%v", eGround.IsHidden(f.Show), eCol1.IsHidden(f.Show), eFloor1.IsHidden(f.Show))
	}
	if f.Show.Zrange[0] != 0 || f.Show.Zrange[1] != 0 {
		t.Fatalf("Ctrl+D did not update Zrange: %v", f.Show.Zrange)
	}

	// 2) Ctrl+A shows all, Ctrl+H hides selected.
	sendCtrl("A", key.ModCtrl)
	if eCol1.IsHidden(f.Show) || eFloor1.IsHidden(f.Show) {
		t.Fatal("Ctrl+A did not restore all elements")
	}
	w.SelectElem([]*st.Elem{eCol2})
	sendCtrl("H", key.ModCtrl)
	if !eCol2.IsHidden(f.Show) || eGround.IsHidden(f.Show) {
		t.Fatal("Ctrl+H did not hide selected element only")
	}

	// 3) Ctrl+Shift+A selects all currently visible elements.
	sendCtrl("A", key.ModCtrl|key.ModShift)
	if len(w.SelectedElems()) != 3 {
		t.Fatalf("Ctrl+Shift+A selected %d elements, want 3 visible", len(w.SelectedElems()))
	}

	// 4) Axis range commands: "z=0", "0<=z<=3", "x>=4" (matching IUP).
	sendCtrl("A", key.ModCtrl)
	w.ExecCommand("z=0")
	if eGround.IsHidden(f.Show) || !eCol1.IsHidden(f.Show) || !eFloor1.IsHidden(f.Show) {
		t.Fatalf("z=0 filter mismatch: ground=%v col1=%v floor1=%v", eGround.IsHidden(f.Show), eCol1.IsHidden(f.Show), eFloor1.IsHidden(f.Show))
	}
	if f.Show.Zrange[0] != 0 || f.Show.Zrange[1] != 0 {
		t.Fatalf("z=0 Zrange = %v", f.Show.Zrange)
	}
	w.ExecCommand("0 <= z <= 3")
	if eGround.IsHidden(f.Show) || eCol1.IsHidden(f.Show) || eFloor1.IsHidden(f.Show) || !eCol2.IsHidden(f.Show) {
		t.Fatalf("0<=z<=3 filter mismatch: ground=%v col1=%v floor1=%v col2=%v", eGround.IsHidden(f.Show), eCol1.IsHidden(f.Show), eFloor1.IsHidden(f.Show), eCol2.IsHidden(f.Show))
	}

	// 5) Ctrl+Up / Ctrl+Down floor navigation.
	sendCtrl(key.NameUpArrow, key.ModCtrl)
	if f.Show.Zrange[0] != 3 || f.Show.Zrange[1] != 6 {
		t.Fatalf("Ctrl+Up Zrange = %v, want [3 6]", f.Show.Zrange)
	}
	sendCtrl(key.NameDownArrow, key.ModCtrl)
	if f.Show.Zrange[0] != 0 || f.Show.Zrange[1] != 3 {
		t.Fatalf("Ctrl+Down Zrange = %v, want [0 3]", f.Show.Zrange)
	}

	// 6) Ctrl+U toggles tf,m <-> kN,m.
	u0 := f.Show.Unit[0]
	sendCtrl("U", key.ModCtrl)
	if f.Show.Unit[0] == u0 {
		t.Fatalf("Ctrl+U did not toggle unit: %v", f.Show.Unit)
	}
	sendCtrl("U", key.ModCtrl)
	if f.Show.Unit[0] != u0 {
		t.Fatalf("second Ctrl+U did not restore unit: %v", f.Show.Unit)
	}

	// 7) Ctrl+Shift+L locks selected, Ctrl+Shift+U unlocks all.
	w.SelectElem([]*st.Elem{eGround})
	sendCtrl("L", key.ModCtrl|key.ModShift)
	if !eGround.Lock {
		t.Fatal("Ctrl+Shift+L did not lock selected element")
	}
	sendCtrl("U", key.ModCtrl|key.ModShift)
	if eGround.Lock {
		t.Fatal("Ctrl+Shift+U did not unlock element")
	}

	// 8) Ctrl+Shift+N flips +/- in Period.
	w.SetPeriod("X+")
	sendCtrl("N", key.ModCtrl|key.ModShift)
	if f.Show.Period != "X-" {
		t.Fatalf("Ctrl+Shift+N period = %q, want X-", f.Show.Period)
	}
}

func TestGioCanvasBackgroundContrastAndToggle(t *testing.T) {
	h := newUIHarness(t)
	w := h.w

	// Default is white canvas: white foreground maps to black so white members/text stay visible.
	if w.CanvasDark() {
		t.Fatal("default canvas should be white")
	}
	w.Foreground(st.WHITE)
	if w.currentPen.R != 0 || w.currentPen.G != 0 || w.currentPen.B != 0 {
		t.Fatalf("white foreground on white canvas = %v, want black", w.currentPen)
	}
	// Non-white colors (e.g. red) are preserved.
	redRGB := st.IntColorList(st.RED)
	w.Foreground(st.RED)
	if int(w.currentPen.R) != redRGB[0] || int(w.currentPen.G) != redRGB[1] || int(w.currentPen.B) != redRGB[2] {
		t.Fatalf("red foreground altered: %v, want %v", w.currentPen, redRGB)
	}

	// White canvas: translucent plate fill keeps standard opacity and original RGB.
	w.DefaultStyle()
	w.Foreground(st.CYAN)
	if w.currentBrush.A != plateOpacityLightDefault || w.currentBrush.R != w.currentPen.R || w.currentBrush.G != w.currentPen.G || w.currentBrush.B != w.currentPen.B {
		t.Fatalf("light canvas plate brush = %v, pen = %v", w.currentBrush, w.currentPen)
	}

	// Switch to black canvas via :bg black command: black foreground maps to white, white stays white.
	w.ExecCommand(":bg black")
	if !w.CanvasDark() {
		t.Fatal(":bg black did not enable dark canvas")
	}
	w.DefaultStyle()
	w.Foreground(st.CYAN)
	if w.currentBrush.A != plateOpacityDarkDefault {
		t.Fatalf("dark canvas plate opacity = %#x, want %#x", w.currentBrush.A, plateOpacityDarkDefault)
	}
	if w.currentBrush.G >= w.currentPen.G || w.currentBrush.B >= w.currentPen.B {
		t.Fatalf("dark canvas translucent plate fill was not softened: brush=%v pen=%v", w.currentBrush, w.currentPen)
	}
	w.SelectElemStyle()
	if w.currentBrush.A != plateOpacityDarkSelected {
		t.Fatalf("dark canvas selected plate opacity = %#x, want %#x", w.currentBrush.A, plateOpacityDarkSelected)
	}
	w.ConfStyle(w.frame.Show)
	if w.currentBrush.A != plateOpacityOpaque {
		t.Fatalf("opaque symbol style lost alpha on dark canvas: %#x", w.currentBrush.A)
	}
	w.Foreground(st.BLACK)
	if w.currentPen.R != 255 || w.currentPen.G != 255 || w.currentPen.B != 255 {
		t.Fatalf("black foreground on black canvas = %v, want white", w.currentPen)
	}
	w.Foreground(st.WHITE)
	if w.currentPen.R != 255 || w.currentPen.G != 255 || w.currentPen.B != 255 {
		t.Fatalf("white foreground on black canvas = %v, want white", w.currentPen)
	}

	// Toggle back via Show tab button.
	w.ui.tab = 1
	h.frame()
	w.ui.button("canvas-bg").Click()
	h.frame()
	waitUI(t, w)
	h.frame()
	if w.CanvasDark() {
		t.Fatal("Show tab BG button did not switch back to white canvas")
	}

	// Verify support symbol DPI scale preserves Show.ConfSize across drawCanvas.
	if got := canvasConfScale(unit.Metric{PxPerDp: 1.5}); math.Abs(got-1.8) > 1e-9 {
		t.Fatalf("canvasConfScale(1.5) = %g, want 1.8", got)
	}
	beforeConf := w.frame.Show.ConfSize
	h.metric = unit.Metric{PxPerDp: 1.5, PxPerSp: 1.5}
	w.Redraw()
	h.frame()
	if w.frame.Show.ConfSize != beforeConf {
		t.Fatalf("drawCanvas mutated Show.ConfSize: got %g, want %g", w.frame.Show.ConfSize, beforeConf)
	}
}
