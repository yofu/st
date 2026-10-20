package stgio

import (
	"image"
	"math"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
)

func TestCanvasDragWithUI(t *testing.T) {
	for _, shift := range []bool{false, true} {
		h := newUIHarness(t)
		w := h.w
		w.SetCanvasRotateSpeedX(0.03)
		w.SetCanvasRotateSpeedY(0.03)
		w.SetCanvasMoveSpeedX(0.2)
		w.SetCanvasMoveSpeedY(0.2)
		w.frame.View.Center = []float64{300, 250}
		w.frame.View.Angle = []float64{20, -30}
		w.Redraw()
		h.frame()
		angle := append([]float64(nil), w.frame.View.Angle...)
		center := append([]float64(nil), w.frame.View.Center...)
		mods := key.Modifiers(0)
		if shift {
			mods = key.ModShift
		}
		y := float32(w.ui.canvasRect.Min.Y)
		h.router.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, PointerID: 7, Position: f32.Pt(500, 450+y)})
		h.frame()
		h.router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, PointerID: 7, Buttons: pointer.ButtonTertiary, Modifiers: mods, Position: f32.Pt(200, 160+y)})
		h.frame()
		if w.start != f32.Pt(200, 160) || w.end != w.start {
			t.Fatalf("press coordinates %v %v", w.start, w.end)
		}
		if w.frame.View.Angle[0] != angle[0] || w.frame.View.Angle[1] != angle[1] || w.frame.View.Center[0] != center[0] || w.frame.View.Center[1] != center[1] {
			t.Fatal("press moved view")
		}
		for i := 0; i < 3; i++ {
			h.frame()
		}
		h.router.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, PointerID: 7, Buttons: pointer.ButtonTertiary, Modifiers: mods, Position: f32.Pt(210, 175+y)})
		h.frame()
		if shift {
			if math.Abs(w.frame.View.Center[0]-center[0]-10*w.CanvasMoveSpeedX()) > 1e-9 || math.Abs(w.frame.View.Center[1]-center[1]-15*w.CanvasMoveSpeedY()) > 1e-9 {
				t.Fatalf("pan jumped %v -> %v", center, w.frame.View.Center)
			}
		} else if math.Abs(w.frame.View.Angle[0]-angle[0]-15*w.CanvasRotateSpeedY()) > 1e-9 || math.Abs(w.frame.View.Angle[1]-angle[1]+10*w.CanvasRotateSpeedX()) > 1e-9 {
			t.Fatalf("rotation jumped %v -> %v", angle, w.frame.View.Angle)
		}
		h.router.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, PointerID: 7, Modifiers: mods, Position: f32.Pt(210, 175+y)})
		h.frame()
		// Starting again far from the previous end must not move the view.
		angle = append(angle[:0], w.frame.View.Angle...)
		center = append(center[:0], w.frame.View.Center...)
		h.router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, PointerID: 7, Buttons: pointer.ButtonTertiary, Modifiers: mods, Position: f32.Pt(600, 400+y)})
		h.frame()
		if w.start != f32.Pt(600, 400) || w.end != w.start || w.frame.View.Angle[0] != angle[0] || w.frame.View.Angle[1] != angle[1] || w.frame.View.Center[0] != center[0] || w.frame.View.Center[1] != center[1] {
			t.Fatal("next press reused previous drag position")
		}
	}
}

func TestCanvasDragIgnoresUnrelatedPositions(t *testing.T) {
	w := NewWindow(nil)
	defer w.ui.close()
	w.SetCanvasMoveSpeedX(0.2)
	w.SetCanvasMoveSpeedY(0.2)
	x, y := w.frame.View.Center[0], w.frame.View.Center[1]
	w.handlePointer(pointer.Event{Kind: pointer.Press, PointerID: 7, Buttons: pointer.ButtonTertiary, Position: f32.Pt(200, 160)})
	// Hover, wheel coordinates, an extra button and a second pointer are not
	// valid origins for the next movement of the captured pointer.
	w.handlePointer(pointer.Event{Kind: pointer.Move, PointerID: 8, Position: f32.Pt(800, 700)})
	w.handlePointer(pointer.Event{Kind: pointer.Scroll, Position: f32.Pt(900, 800), Scroll: f32.Pt(0, 20)})
	w.handlePointer(pointer.Event{Kind: pointer.Press, PointerID: 7, Buttons: pointer.ButtonTertiary | pointer.ButtonSecondary, Position: f32.Pt(210, 175)})
	w.handlePointer(pointer.Event{Kind: pointer.Drag, PointerID: 8, Position: f32.Pt(700, 600)})
	w.handlePointer(pointer.Event{Kind: pointer.Drag, PointerID: 7, Modifiers: key.ModShift, Position: f32.Pt(210, 175)})
	if w.start != f32.Pt(200, 160) || math.Abs(w.frame.View.Center[0]-x-2) > 1e-9 || math.Abs(w.frame.View.Center[1]-y-3) > 1e-9 {
		t.Fatalf("unrelated event replaced origin: %v %v", w.start, w.frame.View.Center)
	}
	w.handlePointer(pointer.Event{Kind: pointer.Cancel})
	if w.end != f32.Pt(210, 175) {
		t.Fatal("Cancel reset pointer to zero")
	}
	w.handlePointer(pointer.Event{Kind: pointer.Press, PointerID: 7, Buttons: pointer.ButtonTertiary, Position: f32.Pt(500, 450)})
	w.handlePointer(pointer.Event{Kind: pointer.Release, PointerID: 7, Modifiers: key.ModShift, Position: f32.Pt(505, 455)})
	if math.Abs(w.frame.View.Center[0]-x-3) > 1e-9 || math.Abs(w.frame.View.Center[1]-y-4) > 1e-9 {
		t.Fatal("next drag reused old origin or missed release movement")
	}
}

func TestCanvasSelectionBatchedAndOutside(t *testing.T) {
	h := newUIHarness(t)
	w := h.w
	y := float32(w.ui.canvasRect.Min.Y)
	h.router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(200, 160+y)}, pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(250, 210+y)})
	h.frame()
	h.frame()
	if w.start != f32.Pt(200, 160) || w.end != f32.Pt(250, 210) || w.selected != image.Rect(200, 160, 250, 210) {
		t.Fatalf("selection misplaced: start=%v end=%v rect=%v", w.start, w.end, w.selected)
	}
	x := float32(w.ui.canvasRect.Max.X + 50)
	h.router.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(x, 210+y)})
	h.frame()
	if w.end != f32.Pt(x, 210) {
		t.Fatalf("drag lost outside canvas: %v", w.end)
	}
	h.router.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: f32.Pt(x+10, 220+y)})
	h.frame()
	if w.selecting || w.press != 0 {
		t.Fatal("release lost outside canvas")
	}
	if w.selected != image.Rect(200, 160, int(x+10), 220) {
		t.Fatalf("release position omitted: %v", w.selected)
	}
}

func TestCanvasDragSurvivesBusyWorker(t *testing.T) {
	h := newUIHarness(t)
	w := h.w
	y := float32(w.ui.canvasRect.Min.Y)
	center := append([]float64(nil), w.frame.View.Center...)
	h.router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonTertiary, Modifiers: key.ModShift, Position: f32.Pt(200, 160+y)})
	h.frame()
	w.ui.modelMu.Lock()
	h.router.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Buttons: pointer.ButtonTertiary, Modifiers: key.ModShift, Position: f32.Pt(210, 175+y)})
	h.frame()
	w.ui.modelMu.Unlock()
	h.frame()
	h.router.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Buttons: pointer.ButtonTertiary, Modifiers: key.ModShift, Position: f32.Pt(220, 190+y)})
	h.frame()
	if math.Abs(w.frame.View.Center[0]-center[0]-20*w.CanvasMoveSpeedX()) > 1e-9 || math.Abs(w.frame.View.Center[1]-center[1]-30*w.CanvasMoveSpeedY()) > 1e-9 {
		t.Fatalf("worker interrupted drag: %v -> %v, press=%v", center, w.frame.View.Center, w.press)
	}
}
