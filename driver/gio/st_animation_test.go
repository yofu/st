package stgio

import (
	"image"
	"math"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	st "github.com/yofu/st/stlib"
)

func finishViewAnimation(h *uiHarness) {
	h.frame()
	h.w.ui.modelMu.Lock()
	duration := time.Duration(0)
	if a := h.w.animation; a != nil {
		duration = a.duration
	}
	h.w.ui.modelMu.Unlock()
	h.now = h.now.Add(duration + viewFrameInterval)
	h.frame()
}

func TestViewAnimationInterpolationAndShortestAngle(t *testing.T) {
	for _, perspective := range []bool{false, true} {
		h := newUIHarness(t)
		w := h.w
		v := w.frame.View
		v.Perspective = perspective
		v.Center = []float64{100, 200}
		v.Focus = []float64{0, 2, 4}
		v.Angle = []float64{10, 350}
		v.Gfact = 1
		v.Dists[1] = 100
		target := v.Copy()
		target.Center = []float64{300, 400}
		target.Focus = []float64{2, 4, 6}
		target.Angle = []float64{50, 10}
		target.Gfact = 4
		target.Dists[1] = 400
		w.Animate(target)
		// Mutating the caller's target must not change the pending animation.
		target.Center[0] = 999
		h.frame()
		if v.Center[0] != 100 || v.Angle[1] != 350 {
			t.Fatal("animation jumped on start")
		}
		duration := w.animation.duration
		h.now = h.now.Add(duration / 2)
		h.frame()
		if math.Abs(v.Center[0]-200) > 1e-6 || math.Abs(v.Focus[2]-5) > 1e-6 || math.Abs(v.Angle[0]-30) > 1e-6 || math.Abs(v.Angle[1]-360) > 1e-6 || math.Abs(v.Gfact-2) > 1e-6 || math.Abs(v.Dists[1]-200) > 1e-6 {
			t.Fatalf("incorrect intermediate view: %+v", v)
		}
		if text := w.ui.fields["PHI"].ed.Text(); text != "30" {
			t.Fatalf("view panel did not follow animated angle: %q", text)
		}
		h.now = h.now.Add(duration)
		h.frame()
		v = w.frame.View
		if w.viewAnimating.Load() || v.Center[0] != 300 || v.Focus[2] != 6 || v.Angle[1] != 10 || v.Gfact != 4 || v.Dists[1] != 400 {
			t.Fatalf("animation missed exact target: %+v", v)
		}
		// Isolate animation scheduling from the command editor's caret blink.
		var router input.Router
		w.scheduleViewAnimation(layout.Context{Source: router.Source(), Now: h.now})
		if _, wake := router.WakeupTime(); wake {
			t.Fatal("finished animation kept scheduling frames")
		}
	}
}

func TestAnimationInterruptAndRetarget(t *testing.T) {
	h := newUIHarness(t)
	w := h.w
	w.SetAngle(40, 90)
	h.frame()
	h.now = h.now.Add(w.animation.duration / 2)
	h.frame()
	current := w.frame.View.Copy()
	w.SetAngle(80, 180)
	if w.frame.View.Angle[0] != current.Angle[0] || w.animation.from.Angle[0] != current.Angle[0] {
		t.Fatal("retarget jumped to an old view")
	}
	h.frame()
	h.now = h.now.Add(w.animation.duration / 2)
	h.frame()
	current = w.frame.View.Copy()
	w.handlePointer(pointer.Event{Kind: pointer.Press, Buttons: pointer.ButtonTertiary, Position: f32.Pt(100, 100)})
	w.handlePointer(pointer.Event{Kind: pointer.Drag, Modifiers: key.ModShift, Position: f32.Pt(120, 130)})
	if w.viewAnimating.Load() || w.frame.View.Center[0] != current.Center[0]+20*w.CanvasMoveSpeedX() {
		t.Fatal("drag did not take over current animated view")
	}
	h.now = h.now.Add(time.Second)
	h.frame()
	if w.frame.View.Angle[0] != current.Angle[0] {
		t.Fatal("cancelled animation continued")
	}
	w.cancelCanvasDrag()
	w.SetAngle(20, 30)
	h.frame()
	w.handlePointer(pointer.Event{Kind: pointer.Scroll, Scroll: f32.Pt(0, -20), Position: f32.Pt(150, 150)})
	if w.viewAnimating.Load() {
		t.Fatal("wheel did not cancel animation")
	}
	w.SetAngle(30, 40)
	w.SetFrame(st.NewFrame())
	if w.viewAnimating.Load() {
		t.Fatal("new frame inherited animation")
	}
}

func TestAnimationDoesNotBlockWorkerAndManualSetting(t *testing.T) {
	h := newUIHarness(t)
	w := h.w
	w.menuAction("cmd::view front")
	waitUI(t, w)
	h.frame()
	if !w.viewAnimating.Load() {
		t.Fatal("menu view did not animate")
	}
	done := false
	w.enqueue(func() { done = true })
	waitUI(t, w)
	if !done {
		t.Fatal("animation blocked command worker")
	}
	if err := w.applySetting("PHI", "25"); err != nil {
		t.Fatal(err)
	}
	if w.viewAnimating.Load() || w.frame.View.Angle[0] != 25 {
		t.Fatal("manual angle did not cancel animation")
	}
}

func TestAnimationSpeedAndDisabledMode(t *testing.T) {
	h := newUIHarness(t)
	w := h.w
	w.SetCanvasAnimateSpeed(0.04)
	w.SetAngle(20, 40)
	var router input.Router
	w.scheduleViewAnimation(layout.Context{Source: router.Source(), Now: h.now})
	if at, wake := router.WakeupTime(); !wake || !at.Equal(h.now.Add(viewFrameInterval)) {
		t.Fatal("active animation did not schedule next frame")
	}
	if d := w.animation.duration; d < 416*time.Millisecond || d > 417*time.Millisecond {
		t.Fatalf("speed does not control duration: %v", d)
	}
	w.SetCanvasAnimateSpeed(0)
	w.SetAngle(30, 50)
	if w.viewAnimating.Load() || w.frame.View.Angle[0] != 30 || w.frame.View.Angle[1] != 50 {
		t.Fatal("disabled animation did not set final view")
	}
}

func TestViewRequestedBeforeInitialLayout(t *testing.T) {
	w := NewWindow(nil)
	defer w.ui.close()
	w.SetAngle(90, -90)
	w.prepareCanvas(layout.Context{Constraints: layout.Exact(image.Pt(640, 480))})
	if w.frame.View.Angle[0] != 90 || w.frame.View.Angle[1] != -90 || w.viewAnimating.Load() {
		t.Fatal("initial sizing lost requested view")
	}
}
