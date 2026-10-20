package stgio

import (
	"fmt"
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/gpu/headless"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/unit"
	st "github.com/yofu/st/stlib"
)

// Compare actual pixels, not only View.Angle/Center: switching drawing paths
// can change the displayed geometry without changing those view parameters.
func TestGioDragUsesSameModelDrawing(t *testing.T) {
	for _, state := range []int{st.PLOT_UNDEFORMED, st.PLOT_DEFORMED, st.PLOT_UNDEFORMED | st.PLOT_DEFORMED} {
		for _, pan := range []bool{false, true} {
			t.Run(fmt.Sprintf("state=%d/pan=%v", state, pan), func(t *testing.T) {
				h := newUIHarness(t)
				h.metric = unit.Metric{PxPerDp: 1.5, PxPerSp: 1.5}
				h.frame()
				w := h.w
				f := w.frame
				sect := st.NewSect()
				sect.Num = 101
				sect.Frame = f
				sect.Color = 0
				f.Sects[sect.Num] = sect
				nodes := make([]*st.Node, 2)
				for i := range nodes {
					n := st.NewNode()
					n.Num = i + 1
					n.Frame = f
					n.Coord = []float64{0, float64(i*2 - 1), float64(i * 3)}
					n.Disp["X"] = []float64{0, 0.5, 0, 0, 0, 0}
					f.Nodes[n.Num] = n
					nodes[i] = n
				}
				f.AddLineElem(1, nodes, sect, st.COLUMN)
				f.Show.Sect[sect.Num] = true
				f.Show.GlobalAxis = false
				f.Show.PlotState = state
				f.Show.Period = "X"
				f.Show.Dfact = 1
				f.Show.ColorMode = st.ECOLOR_BLACK
				f.View.Perspective = false
				f.View.Angle = []float64{0, 0}
				f.View.Center = []float64{300, 300}
				f.View.Gfact = 50
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
				assertSame := func(a, b *image.RGBA, label string) {
					t.Helper()
					changed := 0
					for y := w.ui.canvasRect.Min.Y; y < w.ui.canvasRect.Max.Y; y++ {
						for x := w.ui.canvasRect.Min.X; x < w.ui.canvasRect.Max.X; x++ {
							if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
								changed++
							}
						}
					}
					if changed != 0 {
						t.Fatalf("%s: %d canvas pixels changed with an unchanged view", label, changed)
					}
				}
				before := capture()
				ink := 0
				for y := 100; y < 350; y++ {
					for x := 230; x < 400; x++ {
						c := before.RGBAAt(x, y)
						if c.R < 200 && c.G < 200 && c.B < 200 {
							ink++
						}
					}
				}
				if ink < 50 {
					t.Fatal("test model was not drawn")
				}
				mods := key.Modifiers(0)
				if pan {
					mods = key.ModShift
				}
				start := f32.Pt(400, 400+float32(w.ui.canvasRect.Min.Y))
				h.router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, PointerID: 7, Buttons: pointer.ButtonTertiary, Modifiers: mods, Position: start})
				h.frame()
				assertSame(before, capture(), "Press")
				end := start.Add(f32.Pt(30, 20))
				h.router.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, PointerID: 7, Buttons: pointer.ButtonTertiary, Modifiers: mods, Position: end})
				h.frame()
				during := capture()
				h.router.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, PointerID: 7, Modifiers: mods, Position: end})
				h.frame()
				assertSame(during, capture(), "Release")
				w.RedrawNode()
				h.frame()
				assertSame(during, capture(), "RedrawNode")
			})
		}
	}
}
