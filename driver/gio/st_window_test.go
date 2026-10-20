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
	"gioui.org/op"
	st "github.com/yofu/st/stlib"
)

func TestGioInputRouting(t *testing.T) {
	w := NewWindow(nil)
	var router input.Router
	var ops op.Ops
	size := image.Pt(640, 480)
	frame := func() {
		ops.Reset()
		dims := w.layoutCanvas(layout.Context{Ops: &ops, Source: router.Source(), Constraints: layout.Exact(size)})
		if dims.Size != size {
			t.Fatalf("layout size = %v, want %v", dims.Size, size)
		}
		router.Frame(&ops)
	}
	frame()
	before := append([]float64(nil), w.frame.View.Angle...)
	router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonTertiary, Position: f32.Pt(100, 100)})
	frame()
	if w.press != pointer.ButtonTertiary {
		t.Fatal("middle press was not delivered")
	}
	for _, point := range []f32.Point{f32.Pt(110, 120), f32.Pt(120, 130)} {
		router.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Buttons: pointer.ButtonTertiary, Position: point})
		frame()
	}
	if math.Abs(w.frame.View.Angle[0]-before[0]-30*w.CanvasRotateSpeedY()) > 1e-10 ||
		math.Abs(w.frame.View.Angle[1]-before[1]+20*w.CanvasRotateSpeedX()) > 1e-10 {
		t.Fatalf("rotation must use consecutive drag deltas: %v -> %v", before, w.frame.View.Angle)
	}
	router.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: f32.Pt(120, 130)})
	frame()
	if w.press != 0 {
		t.Fatal("release did not end drag")
	}
	w.frame.View.Perspective = false
	factor := w.frame.View.Gfact
	router.Queue(pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Position: f32.Pt(150, 150), Scroll: f32.Pt(0, 20)})
	frame()
	if w.frame.View.Gfact >= factor {
		t.Fatal("scroll event did not zoom out")
	}
	router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(100, 100)})
	frame()
	if !w.selecting {
		t.Fatal("primary press did not start rectangle")
	}
	router.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(60, 50)})
	frame()
	if w.selected != image.Rect(60, 50, 100, 100) {
		t.Fatalf("reverse drag rectangle = %v", w.selected)
	}
	router.Queue(key.Event{Name: key.NameEscape, State: key.Press})
	frame()
	if w.selecting {
		t.Fatal("Escape was not delivered")
	}
	size = image.Pt(900, 700)
	frame()
	if x, y := w.GetCanvasSize(); x != size.X || y != size.Y {
		t.Fatalf("canvas not resized: %d,%d", x, y)
	}
	if _, wake := router.WakeupTime(); wake {
		t.Fatal("idle drawing scheduled another frame")
	}
}

func TestGioPanAndCancel(t *testing.T) {
	w := NewWindow(nil)
	x, y := w.frame.View.Center[0], w.frame.View.Center[1]
	w.handlePointer(pointer.Event{Kind: pointer.Press, Buttons: pointer.ButtonTertiary, Position: f32.Pt(10, 10)})
	w.handlePointer(pointer.Event{Kind: pointer.Drag, Modifiers: key.ModShift, Position: f32.Pt(30, 40)})
	if w.frame.View.Center[0] != x+20*w.CanvasMoveSpeedX() || w.frame.View.Center[1] != y+30*w.CanvasMoveSpeedY() {
		t.Fatal("pan delta")
	}
	w.handlePointer(pointer.Event{Kind: pointer.Cancel})
	if w.press != 0 || w.selecting {
		t.Fatal("cancel left input state active")
	}
}

func TestGioRedrawDoesNotBlock(t *testing.T) {
	w := NewWindow(nil)
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			w.Redraw()
			w.RedrawNode()
			w.Flush()
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("redraw blocked without event-loop receiver")
	}
}

func TestGioDrawingGeometry(t *testing.T) {
	if got := circleBounds(100, 80, 20); got != image.Rect(90, 70, 110, 90) {
		t.Fatalf("circle bounds: %v", got)
	}
	if got := boundingbox([][]float64{{-0.5, -2.2}, {3.2, 1.1}}); got[0] != -2 || got[1] != -4 || got[2] != 5 || got[3] != 3 {
		t.Fatalf("bounding box clips fractional coordinates: %v", got)
	}
	for _, tc := range []struct{ align, x, y int }{
		{st.NORTH, -20, 0}, {st.SOUTH, -20, -12}, {st.WEST, 0, -6}, {st.EAST, -40, -6},
		{st.CENTER, -20, -6}, {st.SOUTH_WEST, 0, -12}, {st.SOUTH_EAST, -40, -12},
	} {
		x, y := textOffset(tc.align, image.Pt(40, 12))
		if x != tc.x || y != tc.y {
			t.Errorf("alignment %d: %d,%d", tc.align, x, y)
		}
	}
}
