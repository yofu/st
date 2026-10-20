package stgio

import (
	"image"
	"math"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/unit"
	st "github.com/yofu/st/stlib"
)

func mouseFixture(t *testing.T) *uiHarness {
	h := newUIHarness(t)
	h.metric = unit.Metric{PxPerDp: 1.5, PxPerSp: 1.5}
	h.frame()
	w := h.w
	f := w.frame
	s := st.NewSect()
	s.Num = 101
	s.Frame = f
	s.Color = 0
	f.Sects[101] = s
	f.Show.Sect[101] = true
	ns := make([]*st.Node, 3)
	for i := range ns {
		n := st.NewNode()
		n.Num = i + 1
		n.Frame = f
		n.Coord = []float64{0, float64(i * 2), float64(i * 3)}
		f.Nodes[n.Num] = n
		ns[i] = n
	}
	f.AddLineElem(1, ns[:2], s, st.COLUMN)
	f.AddLineElem(2, ns[1:], s, st.GIRDER)
	f.View.Perspective = false
	f.View.Angle = []float64{15, 45}
	f.View.Gfact = 20
	f.View.Center = []float64{800, 700}
	w.Redraw()
	h.frame()
	return h
}

func routedMouse(h *uiHarness, kind pointer.Kind, button pointer.Buttons, p f32.Point, mods key.Modifiers, at time.Duration) {
	h.router.Queue(pointer.Event{Kind: kind, Source: pointer.Mouse, PointerID: 7, Buttons: button, Position: p.Add(f32.Pt(float32(h.w.ui.canvasRect.Min.X), float32(h.w.ui.canvasRect.Min.Y))), Modifiers: mods, Time: at})
	h.frame()
}

func TestMiddleDoubleClickFitsModelAt150Percent(t *testing.T) {
	h := mouseFixture(t)
	w := h.w
	want := st.CenterView(w, append([]float64(nil), w.frame.View.Angle...))
	angle := append([]float64(nil), w.frame.View.Angle...)
	gap := platformClickSettings(0).delay / 4
	p := f32.Pt(300, 250)
	routedMouse(h, pointer.Press, pointer.ButtonTertiary, p, 0, time.Second)
	routedMouse(h, pointer.Release, 0, p, 0, time.Second+time.Millisecond)
	if w.frame.View.Center[0] == want.Center[0] {
		t.Fatal("single click fitted model")
	}
	routedMouse(h, pointer.Press, pointer.ButtonTertiary, p, 0, time.Second+gap)
	if !w.mouse.double {
		t.Fatal("second click was not recognized")
	}
	if !w.viewAnimating.Load() {
		t.Fatal("double click did not animate")
	}
	routedMouse(h, pointer.Release, 0, p, 0, time.Second+gap+time.Millisecond)
	finishViewAnimation(h)
	if math.Abs(w.frame.View.Gfact-want.Gfact) > 1e-9 || math.Abs(w.frame.View.Center[0]-want.Center[0]) > 1e-9 || math.Abs(w.frame.View.Center[1]-want.Center[1]) > 1e-9 {
		t.Fatalf("fit mismatch: got=%v want=%v", w.frame.View, want)
	}
	if w.frame.View.Angle[0] != angle[0] || w.frame.View.Angle[1] != angle[1] {
		t.Fatal("fit changed view angle")
	}
	if w.press != 0 {
		t.Fatal("double-click left drag active")
	}
}

func TestDragDoesNotBecomeMiddleDoubleClick(t *testing.T) {
	w := NewWindow(nil)
	defer w.ui.close()
	w.handlePointer(pointer.Event{Kind: pointer.Press, Buttons: pointer.ButtonTertiary, Position: f32.Pt(100, 100), Time: time.Second})
	w.handlePointer(pointer.Event{Kind: pointer.Drag, Buttons: pointer.ButtonTertiary, Position: f32.Pt(200, 100), Time: time.Second + time.Millisecond})
	w.handlePointer(pointer.Event{Kind: pointer.Release, Position: f32.Pt(100, 100), Time: time.Second + 2*time.Millisecond})
	w.handlePointer(pointer.Event{Kind: pointer.Press, Buttons: pointer.ButtonTertiary, Position: f32.Pt(100, 100), Time: time.Second + 3*time.Millisecond})
	if w.mouse.double {
		t.Fatal("drag followed by click became a double-click")
	}
}

func TestEmptyModelDoubleClickOpensNativePicker(t *testing.T) {
	h := newUIHarness(t)
	w := h.w
	calls := 0
	w.filePicker = func(fileDialogOptions) (string, bool, error) { calls++; return "", false, nil }
	p := f32.Pt(200, 180)
	gap := platformClickSettings(0).delay / 4
	routedMouse(h, pointer.Press, pointer.ButtonPrimary, p, 0, time.Second)
	routedMouse(h, pointer.Release, 0, p, 0, time.Second+time.Millisecond)
	routedMouse(h, pointer.Press, pointer.ButtonPrimary, p, 0, time.Second+gap)
	routedMouse(h, pointer.Release, 0, p, 0, time.Second+gap+time.Millisecond)
	waitUI(t, w)
	if calls != 1 {
		t.Fatalf("picker calls=%d want=1", calls)
	}
}

func TestIUPSelectionDirectionAndShiftRemoval(t *testing.T) {
	h := mouseFixture(t)
	w := h.w
	for i, p := range []float64{20, 100, 140} {
		w.frame.Nodes[i+1].Pcoord = []float64{p, 20}
	}
	w.start, w.end = f32.Pt(0, 0), f32.Pt(60, 40)
	w.selected = image.Rect(0, 0, 60, 40)
	w.finishSelection(0)
	if len(w.SelectedElems()) != 0 {
		t.Fatal("left-to-right selected partial member")
	}
	w.start, w.end = f32.Pt(60, 40), f32.Pt(0, 0)
	w.finishSelection(0)
	if len(w.SelectedElems()) != 1 || w.SelectedElems()[0].Num != 1 {
		t.Fatal("right-to-left did not select member with an enclosed endpoint")
	}
	w.start, w.end = f32.Pt(90, 0), f32.Pt(150, 40)
	w.selected = image.Rect(90, 0, 150, 40)
	w.finishSelection(0)
	if len(w.SelectedElems()) != 2 {
		t.Fatal("ordinary selection did not add")
	}
	w.finishSelection(key.ModShift)
	if len(w.SelectedElems()) != 1 || w.SelectedElems()[0].Num != 1 {
		t.Fatal("Shift did not remove")
	}
	w.start, w.end = f32.Pt(0, 0), f32.Pt(20, 20)
	w.selected = image.Rect(0, 0, 20, 20)
	w.finishSelection(key.ModAlt)
	if len(w.SelectedNodes()) != 1 {
		t.Fatal("node on rectangle boundary was omitted")
	}
	w.finishSelection(key.ModAlt | key.ModShift)
	if len(w.SelectedNodes()) != 0 || len(w.SelectedElems()) != 1 {
		t.Fatal("node removal changed member selection")
	}
	w.SelectElem(nil)
	st.ChainElem(w.frame.Elems[1], w.frame.Elems[2])
	w.start, w.end = f32.Pt(60, 20), f32.Pt(60, 20)
	w.finishSelection(key.ModCtrl)
	if len(w.SelectedElems()) != 2 {
		t.Fatal("Ctrl-click did not select member chain")
	}
}

func TestMiddleDoubleClickRejectsDistanceAndTimeout(t *testing.T) {
	for _, distance := range []bool{false, true} {
		w := NewWindow(nil)
		t.Cleanup(w.ui.close)
		s := platformClickSettings(0)
		p := f32.Pt(100, 100)
		w.handlePointer(pointer.Event{Kind: pointer.Press, Buttons: pointer.ButtonTertiary, Position: p, Time: time.Second})
		w.handlePointer(pointer.Event{Kind: pointer.Release, Position: p, Time: time.Second + time.Millisecond})
		at := time.Second + s.delay + time.Millisecond
		if distance {
			p.X += s.radius.X + 1
			at = time.Second + s.delay/2
		}
		w.handlePointer(pointer.Event{Kind: pointer.Press, Buttons: pointer.ButtonTertiary, Position: p, Time: at})
		if w.mouse.double {
			t.Fatal("distant or late second click was accepted")
		}
	}
}

func TestPlateMouseEditAndUndo(t *testing.T) {
	h := mouseFixture(t)
	w := h.w
	f := w.frame
	n := st.NewNode()
	n.Num = 4
	n.Frame = f
	n.Coord = []float64{1, 6, 2}
	n.Pcoord = []float64{400, 300}
	f.Nodes[n.Num] = n
	el := st.NewPlateElem([]*st.Node{f.Nodes[1], f.Nodes[2], f.Nodes[3]}, f.Sects[101], st.WALL)
	f.AddElem(3, el)
	w.SelectElem([]*st.Elem{el})
	st.Snapshot(w)
	w.startMouseEdit("plate")
	w.end = f32.Pt(float32(f.Nodes[1].Pcoord[0]), float32(f.Nodes[1].Pcoord[1]))
	w.mouseEditClick()
	if w.mouse.editNode != f.Nodes[1] {
		t.Fatal("old plate node was not picked")
	}
	w.end = f32.Pt(400, 300)
	w.mouseEditClick()
	if el.Enod[0] != n || !w.changed || w.mouse.editNode != nil {
		t.Fatal("plate node was not replaced")
	}
	restored, err := w.Undo()
	if err != nil || restored == nil || restored.Elems[3].Enod[0].Num != 1 {
		t.Fatalf("plate edit cannot be undone: %v", err)
	}
}

func TestCtrlWheelAndAltRangeView(t *testing.T) {
	h := mouseFixture(t)
	w := h.w
	center := append([]float64(nil), w.frame.View.Center...)
	scale := w.frame.View.Gfact
	w.handlePointer(pointer.Event{Kind: pointer.Scroll, Position: f32.Pt(100, 120), Scroll: f32.Pt(0, -20), Modifiers: key.ModCtrl})
	if w.frame.View.Gfact <= scale || w.frame.View.Center[0] != center[0] || w.frame.View.Center[1] != center[1] {
		t.Fatal("Ctrl wheel moved the center or failed to zoom")
	}
	view := w.frame.View.Copy()
	w.handlePointer(pointer.Event{Kind: pointer.Press, Buttons: pointer.ButtonTertiary, Position: f32.Pt(100, 100)})
	w.handlePointer(pointer.Event{Kind: pointer.Drag, Buttons: pointer.ButtonTertiary, Position: f32.Pt(120, 130), Modifiers: key.ModAlt})
	if !w.rangeVisible || w.rangeView == nil || w.frame.View.Angle[0] != view.Angle[0] || w.frame.View.Center[0] != view.Center[0] {
		t.Fatal("Alt drag changed model instead of range view")
	}
	w.handlePointer(pointer.Event{Kind: pointer.Release, Position: f32.Pt(120, 130), Modifiers: key.ModAlt})
	rangeCenter := append([]float64(nil), w.rangeView.Center...)
	dist := w.rangeView.Dists[1]
	w.handlePointer(pointer.Event{Kind: pointer.Scroll, Position: f32.Pt(150, 150), Scroll: f32.Pt(0, -20), Modifiers: key.ModAlt | key.ModCtrl})
	if w.rangeView.Dists[1] <= dist || w.rangeView.Center[0] != rangeCenter[0] || w.frame.View.Gfact != view.Gfact {
		t.Fatal("Alt+Ctrl wheel isolation failed")
	}
}

func TestRightClickSubmitRepeatAndContextMenu(t *testing.T) {
	h := newUIHarness(t)
	w := h.w
	p := f32.Pt(200, 180)
	w.ui.command.SetText("'gfact 3")
	routedMouse(h, pointer.Press, pointer.ButtonSecondary, p, 0, time.Second)
	routedMouse(h, pointer.Release, 0, p, 0, time.Second+100*time.Millisecond)
	waitUI(t, w)
	if w.frame.View.Gfact != 3 || w.ui.command.Text() != "" {
		t.Fatal("right release did not execute input")
	}
	hit := false
	w.lastcommand = func(st.Commander) chan bool { hit = true; return nil }
	routedMouse(h, pointer.Press, pointer.ButtonSecondary, p, 0, 2*time.Second)
	routedMouse(h, pointer.Release, 0, p, 0, 2*time.Second+100*time.Millisecond)
	waitUI(t, w)
	if !hit {
		t.Fatal("short right click did not repeat")
	}
	w.lastexcommand = ":view top"
	routedMouse(h, pointer.Press, pointer.ButtonSecondary, p, key.ModShift, 3*time.Second)
	routedMouse(h, pointer.Release, 0, p, key.ModShift, 3*time.Second+100*time.Millisecond)
	waitUI(t, w)
	finishViewAnimation(h)
	if w.frame.View.Angle[0] != 90 || w.frame.View.Angle[1] != -90 {
		t.Fatal("Shift+right did not repeat previous Ex command")
	}
	routedMouse(h, pointer.Press, pointer.ButtonSecondary, p, 0, 4*time.Second)
	routedMouse(h, pointer.Release, 0, p, 0, 4*time.Second+300*time.Millisecond)
	if len(w.ui.contextItems) == 0 || w.ui.menuRect.Empty() {
		t.Fatal("long right press did not show context menu")
	}
}

func TestDoubleClickMouseEditing(t *testing.T) {
	h := mouseFixture(t)
	w := h.w
	el := w.frame.Elems[1]
	w.showCanvasCenter()
	finishViewAnimation(h)
	p := f32.Pt(float32((el.Enod[0].Pcoord[0]+el.Enod[1].Pcoord[0])/2), float32((el.Enod[0].Pcoord[1]+el.Enod[1].Pcoord[1])/2))
	gap := platformClickSettings(0).delay / 4
	routedMouse(h, pointer.Press, pointer.ButtonPrimary, p, 0, time.Second)
	routedMouse(h, pointer.Release, 0, p, 0, time.Second+time.Millisecond)
	if len(w.SelectedElems()) != 1 || w.SelectedElems()[0] != el {
		t.Fatal("first click did not pick target member")
	}
	routedMouse(h, pointer.Press, pointer.ButtonPrimary, p, 0, time.Second+gap)
	if w.mouse.edit != "bond" {
		t.Fatal("member double click did not start TOGGLEBOND")
	}
	routedMouse(h, pointer.Release, 0, p, 0, time.Second+gap+time.Millisecond)
	w.end = f32.Pt(float32(el.Enod[0].Pcoord[0]), float32(el.Enod[0].Pcoord[1]))
	w.mouseEditClick()
	if el.BondState() == 0 || !w.changed {
		t.Fatal("endpoint click did not change bond")
	}
	w.mouseEditClick()
	if el.BondState() != 0 {
		t.Fatal("second endpoint click did not toggle back")
	}
	w.mouse.edit = "plate"
	w.mouse.editNode = el.Enod[0]
	w.stopMouseEdit()
	if w.mouse.editNode != nil || w.mouse.edit != "" {
		t.Fatal("edit was not cancelled")
	}
}
