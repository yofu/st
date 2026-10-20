package stgio

import (
	"image"
	"testing"

	"gioui.org/gpu/headless"
)

func TestViewAnimationRendersIntermediateFrames(t *testing.T) {
	h := mouseFixture(t)
	w := h.w
	w.frame.View.Center = []float64{280, 300}
	w.Redraw()
	h.frame()
	gpu, err := headless.NewWindow(h.size.X, h.size.Y)
	if err != nil {
		t.Skipf("Direct3D unavailable: %v", err)
	}
	defer gpu.Release()
	capture := func() *image.RGBA {
		t.Helper()
		if err := gpu.Frame(&h.ops); err != nil {
			t.Fatal(err)
		}
		img := image.NewRGBA(image.Rectangle{Max: h.size})
		if err := gpu.Screenshot(img); err != nil {
			t.Fatal(err)
		}
		return img
	}
	difference := func(a, b *image.RGBA) int {
		n := 0
		for y := w.ui.canvasRect.Min.Y; y < w.ui.canvasRect.Max.Y; y++ {
			for x := w.ui.canvasRect.Min.X; x < w.ui.canvasRect.Max.X; x++ {
				if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
					n++
				}
			}
		}
		return n
	}
	before := capture()
	w.SetAngle(60, -90)
	h.frame()
	if difference(before, capture()) != 0 {
		t.Fatal("first animated frame jumped")
	}
	duration := w.animation.duration
	h.now = h.now.Add(duration / 2)
	h.frame()
	middle := capture()
	if difference(before, middle) == 0 {
		t.Fatal("intermediate frame did not animate")
	}
	h.now = h.now.Add(duration)
	h.frame()
	end := capture()
	if difference(middle, end) == 0 {
		t.Fatal("intermediate frame already displayed the endpoint")
	}
	w.Redraw()
	h.frame()
	if difference(end, capture()) != 0 {
		t.Fatal("animation endpoint differs from normal rendering")
	}
}
