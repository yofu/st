package stgio

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget/material"
	st "github.com/yofu/st/stlib"
)

func boundingbox(coords [][]float64) []int {
	maxx := coords[0][0]
	minx := coords[0][0]
	maxy := coords[0][1]
	miny := coords[0][1]
	for i := 0; i < len(coords); i++ {
		if coords[i][0] > maxx {
			maxx = coords[i][0]
		}
		if coords[i][0] < minx {
			minx = coords[i][0]
		}
		if coords[i][1] > maxy {
			maxy = coords[i][1]
		}
		if coords[i][1] < miny {
			miny = coords[i][1]
		}
	}
	return []int{int(math.Floor(minx - 1)), int(math.Floor(miny - 1)), int(math.Ceil(maxx + 1)), int(math.Ceil(maxy + 1))}
}

func (stw *Window) Line(x1 float64, y1 float64, x2 float64, y2 float64) {
	bb := boundingbox([][]float64{{x1, y1}, {x2, y2}})
	bounds := image.Rect(bb[0], bb[1], bb[2], bb[3])
	defer clip.RRect{Rect: bounds, SE: 0, SW: 0, NW: 0, NE: 0}.Push(stw.context.Ops).Pop()
	var path clip.Path
	path.Begin(stw.context.Ops)
	path.MoveTo(f32.Pt(float32(x1), float32(y1)))
	path.LineTo(f32.Pt(float32(x2), float32(y2)))
	paint.FillShape(stw.context.Ops, stw.currentPen,
		clip.Stroke{
			Path:  path.End(),
			Width: 1,
		}.Op())
}

func (stw *Window) Polyline(coords [][]float64) {
	if len(coords) < 2 {
		return
	}
	bb := boundingbox(coords)
	bounds := image.Rect(bb[0], bb[1], bb[2], bb[3])
	defer clip.RRect{Rect: bounds, SE: 0, SW: 0, NW: 0, NE: 0}.Push(stw.context.Ops).Pop()
	var path clip.Path
	path.Begin(stw.context.Ops)
	path.MoveTo(f32.Pt(float32(coords[0][0]), float32(coords[0][1])))
	for i := 1; i < len(coords); i++ {
		path.LineTo(f32.Pt(float32(coords[i][0]), float32(coords[i][1])))
	}
	path.Close()
	paint.FillShape(stw.context.Ops, stw.currentPen,
		clip.Stroke{
			Path:  path.End(),
			Width: 1,
		}.Op())
}

func (stw *Window) Polygon(coords [][]float64) {
	if len(coords) < 3 {
		return
	}
	bb := boundingbox(coords)
	bounds := image.Rect(bb[0], bb[1], bb[2], bb[3])
	defer clip.RRect{Rect: bounds, SE: 0, SW: 0, NW: 0, NE: 0}.Push(stw.context.Ops).Pop()
	var path clip.Path
	path.Begin(stw.context.Ops)
	path.MoveTo(f32.Pt(float32(coords[0][0]), float32(coords[0][1])))
	for i := 1; i < len(coords); i++ {
		path.LineTo(f32.Pt(float32(coords[i][0]), float32(coords[i][1])))
	}
	path.Close()
	paint.FillShape(stw.context.Ops, stw.currentBrush,
		clip.Outline{
			Path: path.End(),
		}.Op())
}

// The Drawer circle size is a diameter, matching the IUP/CD driver.
func circleBounds(x, y, diameter float64) image.Rectangle {
	r := diameter / 2
	return image.Rect(int(math.Floor(x-r)), int(math.Floor(y-r)), int(math.Ceil(x+r)), int(math.Ceil(y+r)))
}

func (stw *Window) Circle(x, y, r float64) {
	if r <= 0 {
		return
	}
	path := clip.Ellipse(circleBounds(x, y, r)).Path(stw.context.Ops)
	paint.FillShape(stw.context.Ops, stw.currentPen, clip.Stroke{Path: path, Width: 1}.Op())
}

func (stw *Window) FilledCircle(x, y, r float64) {
	if r <= 0 {
		return
	}
	paint.FillShape(stw.context.Ops, stw.currentBrush, clip.Ellipse(circleBounds(x, y, r)).Op(stw.context.Ops))
}

func (stw *Window) Text(x float64, y float64, txt string) {
	if txt == "" {
		return
	}
	gtx := stw.context
	gtx.Constraints = layout.Constraints{Max: image.Pt(100000, 100000)}
	label := material.Body2(stw.Theme, txt)
	label.Color = stw.currentPen
	macro := op.Record(gtx.Ops)
	dims := label.Layout(gtx)
	call := macro.Stop()
	dx, dy := textOffset(stw.textAlignment, dims.Size)
	transform := op.Affine(f32.Affine2D{}.
		Rotate(f32.Point{}, float32(-stw.textAngle*math.Pi/180)).
		Offset(f32.Pt(float32(x), float32(y)))).Push(gtx.Ops)
	defer transform.Pop()
	offset := op.Offset(image.Pt(dx, dy)).Push(gtx.Ops)
	defer offset.Pop()
	call.Add(gtx.Ops)
}

// Highlight the selected geometry even though this driver does not yet draw
// the IUP/CD dotted-line and hatch styles.
func (stw *Window) drawSelection() {
	for _, e := range stw.SelectedElems() {
		if e == nil || e.IsHidden(stw.frame.Show) || len(e.Enod) < 2 {
			continue
		}
		var path clip.Path
		path.Begin(stw.context.Ops)
		for i, n := range e.Enod {
			p := f32.Pt(float32(n.Pcoord[0]), float32(n.Pcoord[1]))
			if i == 0 {
				path.MoveTo(p)
			} else {
				path.LineTo(p)
			}
		}
		if e.Enods > 2 {
			path.Close()
		}
		paint.FillShape(stw.context.Ops, color.NRGBA{R: 255, G: 120, B: 20, A: 230}, clip.Stroke{Path: path.End(), Width: 3}.Op())
	}
}

func textOffset(alignment int, size image.Point) (int, int) {
	switch alignment {
	case st.NORTH:
		return -size.X / 2, 0
	case st.SOUTH:
		return -size.X / 2, -size.Y
	case st.WEST:
		return 0, -size.Y / 2
	case st.EAST:
		return -size.X, -size.Y / 2
	case st.CENTER:
		return -size.X / 2, -size.Y / 2
	case st.SOUTH_EAST:
		return -size.X, -size.Y
	default:
		return 0, -size.Y
	}
}

func (stw *Window) Foreground(col int) int {
	lis := st.IntColorList(col)
	old := int(stw.currentPen.R)<<16 + int(stw.currentPen.G)<<8 + int(stw.currentPen.B)
	stw.currentPen = color.NRGBA{uint8(lis[0]), uint8(lis[1]), uint8(lis[2]), 0xff}
	stw.currentBrush = color.NRGBA{uint8(lis[0]), uint8(lis[1]), uint8(lis[2]), PLATE_OPACITY}
	// stw.font.color = color.RGBA{uint8(col[0]), uint8(col[1]), uint8(col[2]), 0xff}
	return old
}

func (stw *Window) LineStyle(ls int) {
}

func (stw *Window) TextAlignment(alignment int) {
	stw.textAlignment = alignment
}

func (stw *Window) TextOrientation(degrees float64) {
	stw.textAngle = degrees
}

func (stw *Window) DefaultStyle() {
	stw.Foreground(0x000000)
	stw.LineStyle(st.CONTINUOUS)
	PLATE_OPACITY = 0x55
}

func (stw *Window) BondStyle(show *st.Show) {
	stw.Foreground(0x000000)
}

func (stw *Window) PhingeStyle(show *st.Show) {
	stw.Foreground(0x000000)
	PLATE_OPACITY = 0xff
}

func (stw *Window) ConfStyle(show *st.Show) {
	stw.Foreground(0x000000)
	PLATE_OPACITY = 0xff
}

func (stw *Window) SelectNodeStyle() {
	stw.Foreground(st.RainbowColor[6])
}

func (stw *Window) SelectElemStyle() {
	stw.LineStyle(st.DOTTED)
	PLATE_OPACITY = 0x88
}

func (stw *Window) ShowPrintRange() bool {
	return showprintrange
}

func (stw *Window) CanvasPaperSize() (float64, float64, error) {
	w, h := stw.GetCanvasSize()
	length := math.Min(float64(w), float64(h)) * 0.9
	val := 1.0 / math.Sqrt(2)
	switch stw.papersize {
	default:
		return 0.0, 0.0, fmt.Errorf("unknown papersize")
	case st.A4_TATE, st.A3_TATE:
		return length * val, length, nil
	case st.A4_YOKO, st.A3_YOKO:
		return length, length * val, nil
	}
}

// FrameEvent.Frame submits the completed drawing. Scheduling another frame
// here would make every DrawFrame trigger an endless redraw loop.
func (stw *Window) Flush() {
}

func (stw *Window) CanvasDirection() int {
	return 1
}
