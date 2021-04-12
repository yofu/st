package stgio

import (
	"fmt"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/op" // op is used for recording different operations.
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	st "github.com/yofu/st/stlib"
)

func (stw *Window) Line(x1 float64, y1 float64, x2 float64, y2 float64) {
	stk := op.Save(stw.context.Ops)
	var path clip.Path
	path.Begin(stw.context.Ops)
	path.Move(f32.Pt(float32(x1), float32(y1)))
	path.LineTo(f32.Pt(float32(x2), float32(y2)))
	clip.Stroke{
		Path: path.End(),
		Style: clip.StrokeStyle{
			Width: 1,
			Cap:   clip.FlatCap,
			Join:  clip.BevelJoin,
			Miter: 0,
		},
	}.Op().Add(stw.context.Ops)
	paint.Fill(stw.context.Ops, stw.currentPen)
	stk.Load()
}

func (stw *Window) Polyline(coords [][]float64) {
	stk := op.Save(stw.context.Ops)
	var path clip.Path
	path.Begin(stw.context.Ops)
	path.Move(f32.Pt(float32(coords[0][0]), float32(coords[0][1])))
	for i := 1; i < len(coords); i++ {
		path.LineTo(f32.Pt(float32(coords[i][0]), float32(coords[i][1])))
	}
	path.LineTo(f32.Pt(float32(coords[0][0]), float32(coords[0][1])))
	clip.Stroke{
		Path: path.End(),
		Style: clip.StrokeStyle{
			Width: 1,
			Cap:   clip.FlatCap,
			Join:  clip.BevelJoin,
			Miter: 0,
		},
	}.Op().Add(stw.context.Ops)
	paint.Fill(stw.context.Ops, stw.currentPen)
	stk.Load()
}

func (stw *Window) Polygon(coords [][]float64) {
	stk := op.Save(stw.context.Ops)
	var path clip.Path
	path.Begin(stw.context.Ops)
	path.Move(f32.Pt(float32(coords[0][0]), float32(coords[0][1])))
	for i := 1; i < len(coords); i++ {
		path.LineTo(f32.Pt(float32(coords[i][0]), float32(coords[i][1])))
	}
	path.LineTo(f32.Pt(float32(coords[0][0]), float32(coords[0][1])))
	clip.Outline{
		Path: path.End(),
	}.Op().Add(stw.context.Ops)
	paint.Fill(stw.context.Ops, stw.currentBrush)
	stk.Load()
}

func (stw *Window) Circle(x float64, y float64, r float64) {
	stk := op.Save(stw.context.Ops)
	var path clip.Path
	path.Begin(stw.context.Ops)
	path.Move(f32.Pt(float32(x-r/2), float32(y-r/2)))
	path.Arc(f32.Pt(float32(r/2), float32(r/2)), f32.Pt(float32(r/2), float32(r/2)), 2*math.Pi)
	clip.Stroke{
		Path: path.End(),
		Style: clip.StrokeStyle{
			Width: 1,
			Cap:   clip.FlatCap,
			Join:  clip.BevelJoin,
			Miter: 0,
		},
	}.Op().Add(stw.context.Ops)
	paint.Fill(stw.context.Ops, stw.currentPen)
	stk.Load()
}

func (stw *Window) FilledCircle(x float64, y float64, r float64) {
	stk := op.Save(stw.context.Ops)
	var path clip.Path
	path.Begin(stw.context.Ops)
	path.Move(f32.Pt(float32(x-r/2), float32(y-r/2)))
	path.Arc(f32.Pt(float32(r/2), float32(r/2)), f32.Pt(float32(r/2), float32(r/2)), 2*math.Pi)
	clip.Outline{
		Path: path.End(),
	}.Op().Add(stw.context.Ops)
	paint.Fill(stw.context.Ops, stw.currentBrush)
	stk.Load()
}

func (stw *Window) Text(x float64, y float64, txt string) {
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

func (stw *Window) TextAlignment(int) {
}

func (stw *Window) TextOrientation(float64) {
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

func (stw *Window) Flush() {
	stw.QueueRedrawAll()
}

func (stw *Window) CanvasDirection() int {
	return 1
}
