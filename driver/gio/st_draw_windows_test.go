package stgio

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/gpu/headless"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	st "github.com/yofu/st/stlib"
)

func TestGioSelectionOverlayFollowsCursor(t *testing.T) {
	h := newUIHarness(t)
	h.metric = unit.Metric{PxPerDp: 1.5, PxPerSp: 1.5}
	h.frame()
	gpu, err := headless.NewWindow(h.size.X, h.size.Y)
	if err != nil {
		t.Skipf("Direct3D unavailable: %v", err)
	}
	defer gpu.Release()
	y := float32(h.w.ui.canvasRect.Min.Y)
	h.router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(180, 140+y)})
	h.frame()
	h.w.pointerPosition = func() (f32.Point, bool) { return f32.Pt(300, 220+y), true }
	// Exercise the overlay independently of the model cache and a stale offset
	// in event coordinates. Its actual border must still end at the cursor.
	h.w.ui.modelMu.Lock()
	h.router.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(340, 280+y)})
	h.frame()
	h.w.ui.modelMu.Unlock()
	if err := gpu.Frame(&h.ops); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rectangle{Max: h.size})
	if err := gpu.Screenshot(img); err != nil {
		t.Fatal(err)
	}
	for _, p := range []image.Point{image.Pt(180, 141+int(y)), image.Pt(300, 219+int(y))} {
		c := img.RGBAAt(p.X, p.Y)
		if c.R < 240 || c.G >= 230 || c.B >= 230 {
			t.Fatalf("selection missing at canvas coordinate %v: %v", p, c)
		}
	}
	for _, p := range []image.Point{image.Pt(178, 141+int(y)), image.Pt(302, 219+int(y)), image.Pt(240, 180+int(y))} {
		c := img.RGBAAt(p.X, p.Y)
		if c.R < 240 || c.G < 240 || c.B < 240 {
			t.Fatalf("selection extends beyond cursor at %v: %v", p, c)
		}
	}
}

// Exercise the actual Windows Direct3D renderer, not just the generated paths.
func TestGioWindowsRendering(t *testing.T) {
	gpu, err := headless.NewWindow(500, 300)
	if err != nil {
		t.Skipf("Direct3D unavailable: %v", err)
	}
	defer gpu.Release()
	w := NewWindow(nil)
	var ops op.Ops
	w.layoutCanvas(layout.Context{Ops: &ops, Constraints: layout.Exact(image.Pt(500, 300))})
	w.Foreground(0)
	w.Line(50, 50, 150, 50)
	w.Circle(250, 100, 40)
	w.FilledCircle(350, 100, 40)
	w.TextAlignment(st.NORTH)
	w.Text(250, 180, "Node 123 / Stress")
	// Empty/degenerate geometry must be harmless.
	w.Polyline(nil)
	w.Polygon([][]float64{{0, 0}})
	w.Circle(0, 0, 0)
	if err := gpu.Frame(&ops); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 500, 300))
	if err := gpu.Screenshot(img); err != nil {
		t.Fatal(err)
	}
	dark := func(r image.Rectangle) int {
		n := 0
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				p := img.RGBAAt(x, y)
				if p.R < 245 && p.G < 245 && p.B < 245 {
					n++
				}
			}
		}
		return n
	}
	for _, tc := range []struct {
		name    string
		area    image.Rectangle
		minimum int
	}{
		{"line", image.Rect(70, 48, 130, 53), 30},
		{"circle edge", image.Rect(268, 97, 273, 103), 2},
		{"filled circle", image.Rect(345, 95, 355, 105), 90},
		{"text", image.Rect(100, 180, 400, 210), 50},
	} {
		if n := dark(tc.area); n < tc.minimum {
			t.Errorf("%s missing: %d non-white pixels", tc.name, n)
		}
	}
	if n := dark(image.Rect(245, 95, 255, 105)); n != 0 {
		t.Errorf("outline circle center is filled: %d pixels", n)
	}
}

func TestGioUIWindowsRendering(t *testing.T) {
	h := newUIHarness(t)
	gpu, err := headless.NewWindow(h.size.X, h.size.Y)
	if err != nil {
		t.Skipf("Direct3D unavailable: %v", err)
	}
	defer gpu.Release()
	h.w.History("解析ログ: test response / completed")
	for _, tab := range []int{0, 1, 2} {
		h.w.ui.tab = tab
		h.frame()
		if err := gpu.Frame(&h.ops); err != nil {
			t.Fatal(err)
		}
		img := image.NewRGBA(image.Rectangle{Max: h.size})
		if err := gpu.Screenshot(img); err != nil {
			t.Fatal(err)
		}
		// Confirm all four requested UI regions actually paint text/controls.
		for _, r := range []image.Rectangle{image.Rect(0, 0, 400, 34), image.Rect(918, 34, 1190, 140), image.Rect(0, 650, 500, 730), image.Rect(0, 840, 500, 875)} {
			count := 0
			for y := r.Min.Y; y < r.Max.Y; y++ {
				for x := r.Min.X; x < r.Max.X; x++ {
					p := img.RGBAAt(x, y)
					if p.R < 180 && p.G < 180 && p.B < 180 {
						count++
					}
				}
			}
			if count < 30 {
				t.Errorf("tab %d: UI region %v is blank (%d dark pixels)", tab, r, count)
			}
		}
	}
}
