package stlibui

import (
	"fmt"
	"math"

	"github.com/andlabs/ui"
	st "github.com/yofu/st/stlib"
)

func mkSolidBrush(color uint32, alpha float64) *ui.DrawBrush {
	factor := 1.0
	brush := new(ui.DrawBrush)
	brush.Type = ui.DrawBrushTypeSolid
	component := uint8((color >> 16) & 0xFF)
	brush.R = float64(component) / 255 * factor
	component = uint8((color >> 8) & 0xFF)
	brush.G = float64(component) / 255 * factor
	component = uint8(color & 0xFF)
	brush.B = float64(component) / 255 * factor
	brush.A = alpha
	return brush
}

func (stw *Window) Line(x1 float64, y1 float64, x2 float64, y2 float64) {
	path := ui.DrawNewPath(ui.DrawFillModeWinding)
	path.NewFigure(x1, y1)
	path.LineTo(x2, y2)
	path.End()
	stw.currentDrawParam.Context.Stroke(path, stw.currentPen, stw.currentStrokeParam)
	path.Free()
}

func (stw *Window) Polyline(coords [][]float64) {
	path := ui.DrawNewPath(ui.DrawFillModeWinding)
	path.NewFigure(coords[0][0], coords[0][1])
	for _, c := range coords[1:] {
		path.LineTo(c[0], c[1])
	}
	path.CloseFigure()
	path.End()
	stw.currentDrawParam.Context.Stroke(path, stw.currentPen, stw.currentStrokeParam)
	path.Free()
}

func (stw *Window) Polygon(coords [][]float64) {
	path := ui.DrawNewPath(ui.DrawFillModeWinding)
	path.NewFigure(coords[0][0], coords[0][1])
	for _, c := range coords[1:] {
		path.LineTo(c[0], c[1])
	}
	path.CloseFigure()
	path.End()
	stw.currentDrawParam.Context.Fill(path, stw.currentBrush)
	path.Free()
}

func (stw *Window) Circle(x float64, y float64, r float64) {
	path := ui.DrawNewPath(ui.DrawFillModeWinding)
	path.NewFigureWithArc(x, y, 0.5*r, 0.0, 0.0, false)
	path.ArcTo(x, y, 0.5*r, 0.0, math.Pi*2.0, false)
	path.CloseFigure()
	path.End()
	stw.currentDrawParam.Context.Stroke(path, stw.currentPen, stw.currentStrokeParam)
	path.Free()
}

func (stw *Window) FilledCircle(x float64, y float64, r float64) {
	path := ui.DrawNewPath(ui.DrawFillModeWinding)
	path.NewFigureWithArc(x, y, 0.5*r, 0.0, 0.0, false)
	path.ArcTo(x, y, 0.5*r, 0.0, math.Pi*2.0, false)
	path.CloseFigure()
	path.End()
	stw.currentDrawParam.Context.Fill(path, stw.currentBrush)
	path.Free()
}

func (stw *Window) Text(x float64, y float64, txt string) {
	str := ui.NewAttributedString(txt)
	tl := ui.DrawNewTextLayout(&ui.DrawTextLayoutParams{
		String:      str,
		DefaultFont: stw.currentFont,
		Width:       500,
		Align:       ui.DrawTextAlign(ui.DrawTextAlignLeft),
	})
	stw.currentDrawParam.Context.Text(tl, x, y-float64(stw.currentFont.Size))
	tl.Free()
}

func (stw *Window) Foreground(col int) int {
	r := stw.currentPen.R
	g := stw.currentPen.G
	b := stw.currentPen.B
	old := int(r)<<16 + int(g)<<8 + int(b)
	stw.currentPen = mkSolidBrush(0x000000, 1.0)
	stw.currentBrush = mkSolidBrush(uint32(col), PLATE_OPACITY)
	return old
}

func (stw *Window) LineStyle(ls int) {
	stw.DrawOption.LineStyle(ls)
	if ls == st.CONTINUOUS {
		stw.currentStrokeParam = &ui.DrawStrokeParams{
			Cap:        ui.DrawLineCapFlat,
			Join:       ui.DrawLineJoinMiter,
			Thickness:  LINE_THICKNESS,
			MiterLimit: ui.DrawDefaultMiterLimit,
		}
	} else {
		ld := stw.LineDash()
		ldash := make([]float64, len(ld))
		for i, l := range ld {
			ldash[i] = float64(l)
		}
		stw.currentStrokeParam = &ui.DrawStrokeParams{
			Cap:        ui.DrawLineCapFlat,
			Join:       ui.DrawLineJoinMiter,
			Thickness:  LINE_THICKNESS,
			MiterLimit: ui.DrawDefaultMiterLimit,
			Dashes:     ldash,
		}
	}
}

func (stw *Window) TextAlignment(int) {
}

func (stw *Window) TextOrientation(float64) {
}

func (stw *Window) DefaultStyle() {
	stw.Foreground(0x000000)
	stw.LineStyle(st.CONTINUOUS)
}

func (stw *Window) BondStyle(*st.Show) {
}

func (stw *Window) PhingeStyle(*st.Show) {
}

func (stw *Window) ConfStyle(*st.Show) {
}

func (stw *Window) SelectNodeStyle() {
}

func (stw *Window) SelectElemStyle() {
	stw.LineStyle(st.DOTTED)
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
	stw.currentArea.QueueRedrawAll()
}

func (stw *Window) CanvasDirection() int {
	return 1
}
