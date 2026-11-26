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
	ox, oy := float32(h.w.ui.canvasRect.Min.X), float32(h.w.ui.canvasRect.Min.Y)
	h.router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(180+ox, 140+oy)})
	h.frame()
	h.w.pointerPosition = func() (f32.Point, bool) { return f32.Pt(300+ox, 220+oy), true }
	// Exercise the overlay independently of the model cache and a stale offset
	// in event coordinates. Its actual border must still end at the cursor.
	h.w.ui.modelMu.Lock()
	h.router.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(340+ox, 280+oy)})
	h.frame()
	h.w.ui.modelMu.Unlock()
	if err := gpu.Frame(&h.ops); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rectangle{Max: h.size})
	if err := gpu.Screenshot(img); err != nil {
		t.Fatal(err)
	}
	for _, p := range []image.Point{image.Pt(180+int(ox), 141+int(oy)), image.Pt(300+int(ox), 219+int(oy))} {
		c := img.RGBAAt(p.X, p.Y)
		if c.R < 240 || c.G >= 230 || c.B >= 230 {
			t.Fatalf("selection missing at canvas coordinate %v: %v", p, c)
		}
	}
	for _, p := range []image.Point{image.Pt(178+int(ox), 141+int(oy)), image.Pt(302+int(ox), 219+int(oy)), image.Pt(240+int(ox), 180+int(oy))} {
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
		// Confirm all four UI regions (menu, left panel, history, command bar) actually paint text/controls,
		// and History text starts on the left side directly above SELECT / Command.
		u := h.w.ui
		historyLeft := image.Rect(u.historyRect.Min.X, u.historyRect.Min.Y, min(u.historyRect.Max.X, u.historyRect.Min.X+240), u.historyRect.Max.Y)
		for _, r := range []image.Rectangle{
			image.Rect(0, 0, 400, u.panelRect.Min.Y),
			image.Rect(u.panelRect.Min.X, u.panelRect.Min.Y, u.panelRect.Max.X, min(u.panelRect.Max.Y, u.panelRect.Min.Y+160)),
			historyLeft,
			u.commandRect,
		} {
			count := 0
			for y := r.Min.Y; y < r.Max.Y; y++ {
				for x := r.Min.X; x < r.Max.X; x++ {
					p := img.RGBAAt(x, y)
					if p.R < 160 && p.G < 160 && p.B < 160 {
						count++
					}
				}
			}
			if count < 30 {
				t.Errorf("tab %d: UI region %v is blank (%d dark pixels)", tab, r, count)
			}
		}
		// Confirm the bottom of the left panel has uniform gray background (no stray white/cream column).
		bottomPanel := image.Rect(u.panelRect.Min.X+4, u.panelRect.Max.Y-40, u.panelRect.Max.X-4, u.panelRect.Max.Y-4)
		for y := bottomPanel.Min.Y; y < bottomPanel.Max.Y; y++ {
			for x := bottomPanel.Min.X; x < bottomPanel.Max.X; x++ {
				p := img.RGBAAt(x, y)
				if p.R > 230 || p.G > 230 || p.B > 230 {
					t.Fatalf("tab %d: bright non-gray pixel %v in bottom panel at (%d,%d)", tab, p, x, y)
				}
			}
		}
	}
}

func TestGioCanvasBackgroundContrastRendering(t *testing.T) {
	gpu, err := headless.NewWindow(300, 200)
	if err != nil {
		t.Skipf("Direct3D unavailable: %v", err)
	}
	defer gpu.Release()

	// 1) White background: white foreground (st.WHITE) must render visible dark pixels.
	w := NewWindow(nil)
	defer w.ui.close()
	var ops op.Ops
	w.layoutCanvas(layout.Context{Ops: &ops, Constraints: layout.Exact(image.Pt(300, 200))})
	w.Foreground(st.WHITE)
	w.Line(40, 100, 260, 100)
	if err := gpu.Frame(&ops); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 300, 200))
	if err := gpu.Screenshot(img); err != nil {
		t.Fatal(err)
	}
	if bg := img.RGBAAt(10, 10); bg.R < 245 || bg.G < 245 || bg.B < 245 {
		t.Fatalf("expected white canvas background, got %v", bg)
	}
	darkLine := 0
	for x := 60; x < 240; x++ {
		for y := 98; y <= 102; y++ {
			p := img.RGBAAt(x, y)
			if p.R < 220 && p.G < 220 && p.B < 220 {
				darkLine++
			}
		}
	}
	if darkLine < 100 {
		t.Fatalf("white line on white canvas was not rendered in black: %d dark pixels", darkLine)
	}

	// 2) Black background: black foreground (st.BLACK) must render visible bright pixels.
	w.SetCanvasDark(true)
	ops.Reset()
	w.layoutCanvas(layout.Context{Ops: &ops, Constraints: layout.Exact(image.Pt(300, 200))})
	w.Foreground(st.BLACK)
	w.Line(40, 100, 260, 100)
	if err := gpu.Frame(&ops); err != nil {
		t.Fatal(err)
	}
	if err := gpu.Screenshot(img); err != nil {
		t.Fatal(err)
	}
	if bg := img.RGBAAt(10, 10); bg.R > 10 || bg.G > 10 || bg.B > 10 {
		t.Fatalf("expected black canvas background, got %v", bg)
	}
	brightLine := 0
	for x := 60; x < 240; x++ {
		for y := 98; y <= 102; y++ {
			p := img.RGBAAt(x, y)
			if p.R > 40 && p.G > 40 && p.B > 40 {
				brightLine++
			}
		}
	}
	if brightLine < 100 {
		t.Fatalf("black line on black canvas was not rendered in white: %d bright pixels", brightLine)
	}
}
