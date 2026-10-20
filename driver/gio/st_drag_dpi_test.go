package stgio

import (
	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/unit"
	"testing"
)

func TestCanvasPointerAt150Percent(t *testing.T) {
	h := newUIHarness(t)
	h.metric = unit.Metric{PxPerDp: 1.5, PxPerSp: 1.5}
	h.frame()
	y := float32(h.w.ui.canvasRect.Min.Y)
	h.router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(200, 160+y)})
	h.frame()
	for _, p := range []f32.Point{f32.Pt(250, 210), f32.Pt(400, 350), f32.Pt(180, 140)} {
		h.router.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: p.Add(f32.Pt(0, y))})
		h.frame()
		if h.w.start != f32.Pt(200, 160) || h.w.end != p {
			t.Fatalf("150%% pointer mismatch: start=%v end=%v want=%v canvas=%v", h.w.start, h.w.end, p, h.w.ui.canvasRect)
		}
	}
}

func TestCanvasNativeCursorAndOverlayWhileBusy(t *testing.T) {
	for _, pan := range []bool{false, true} {
		h := newUIHarness(t)
		h.metric = unit.Metric{PxPerDp: 1.5, PxPerSp: 1.5}
		h.frame()
		w := h.w
		y := float32(w.ui.canvasRect.Min.Y)
		button := pointer.ButtonPrimary
		mods := key.Modifiers(0)
		if pan {
			button = pointer.ButtonTertiary
			mods = key.ModShift
		}
		w.SetCanvasMoveSpeedX(0.2)
		w.SetCanvasMoveSpeedY(0.2)
		center := append([]float64(nil), w.frame.View.Center...)
		h.router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: button, Modifiers: mods, Position: f32.Pt(200, 160+y)})
		h.frame()
		revision := w.ui.canvasRevision
		actual := f32.Pt(300, 260+y)
		w.pointerPosition = func() (f32.Point, bool) { return actual, true }
		// Simulate a persistent event-coordinate offset. Native client pixels
		// must win, rather than multiplying local coordinates by the DPI scale.
		w.ui.modelMu.Lock()
		h.router.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Buttons: button, Modifiers: mods, Position: actual.Add(f32.Pt(40, 60))})
		h.frame()
		w.ui.modelMu.Unlock()
		if !pan && (!w.ui.selectionOverlay || w.ui.selectionStart != f32.Pt(200, 160) || w.ui.selectionEnd != f32.Pt(300, 260)) {
			t.Fatalf("overlay lag/offset while busy: %v -> %v", w.ui.selectionStart, w.ui.selectionEnd)
		}
		if w.ui.canvasRevision != revision {
			t.Fatal("busy worker changed cached model")
		}
		h.frame()
		if w.end != f32.Pt(300, 260) {
			t.Fatalf("model input differs from cursor: %v", w.end)
		}
		if pan {
			if w.frame.View.Center[0] != center[0]+20 || w.frame.View.Center[1] != center[1]+20 {
				t.Fatalf("pan used uncorrected pointer: %v", w.frame.View.Center)
			}
		} else if w.ui.canvasRevision != revision {
			t.Fatal("selection motion redrew the model")
		}
		h.router.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Modifiers: mods, Position: actual.Add(f32.Pt(40, 60))})
		h.frame()
		if w.ui.selectionOverlay || w.press != 0 {
			t.Fatal("release left overlay active")
		}
		if w.end != f32.Pt(300, 260) {
			t.Fatal("release reintroduced coordinate offset")
		}
	}
}
