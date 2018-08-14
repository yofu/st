package st

import (
	"image"
	"image/color"
	"math"

	"github.com/llgcode/draw2d/draw2dimg"
)

// TODO: font

type PngCanvas struct {
	width         int
	height        int
	currentCanvas *draw2dimg.GraphicContext
}

func PrintPNG(frame *Frame, otp string, w, h int) error {
	dest := image.NewRGBA(image.Rect(0, 0, w, h))
	gc := draw2dimg.NewGraphicContext(dest)
	pc := &PngCanvas{
		width:         w,
		height:        h,
		currentCanvas: gc,
	}
	pc.currentCanvas.SetStrokeColor(color.RGBA{0x00, 0x00, 0x00, 0xff})
	pc.currentCanvas.SetLineWidth(1)
	cm := frame.Show.ColorMode
	if cm == ECOLOR_WHITE {
		cm = ECOLOR_BLACK
	}
	DrawFrame(pc, frame, cm, true)
	return draw2dimg.SaveToPngFile(otp, dest)
}

func (pc *PngCanvas) Line(x1, y1, x2, y2 float64) {
	pc.currentCanvas.BeginPath()
	pc.currentCanvas.MoveTo(x1, y1)
	pc.currentCanvas.LineTo(x2, y2)
	pc.currentCanvas.Close()
	pc.currentCanvas.Stroke()
}

func (pc *PngCanvas) Polyline(coord [][]float64) {
	if len(coord) < 2 {
		return
	}
	pc.currentCanvas.BeginPath()
	pc.currentCanvas.MoveTo(coord[0][0], coord[0][1])
	for _, c := range coord[1:] {
		pc.currentCanvas.LineTo(c[0], c[1])
	}
	pc.currentCanvas.Close()
	pc.currentCanvas.Stroke()
}

func (pc *PngCanvas) Polygon(coord [][]float64) {
	if len(coord) < 2 {
		return
	}
	pc.currentCanvas.BeginPath()
	pc.currentCanvas.MoveTo(coord[0][0], coord[0][1])
	for _, c := range coord[1:] {
		pc.currentCanvas.LineTo(c[0], c[1])
	}
	pc.currentCanvas.Close()
	pc.currentCanvas.Fill()
}

func (pc *PngCanvas) Circle(x, y, r float64) {
	pc.currentCanvas.BeginPath()
	pc.currentCanvas.MoveTo(x, y)
	pc.currentCanvas.ArcTo(x, y, r, r, 0, 2*math.Pi)
	pc.currentCanvas.Close()
	pc.currentCanvas.Stroke()
}

func (pc *PngCanvas) FilledCircle(x, y, r float64) {
	pc.currentCanvas.BeginPath()
	pc.currentCanvas.MoveTo(x, y)
	pc.currentCanvas.ArcTo(x, y, r, r, 0, 2*math.Pi)
	pc.currentCanvas.Close()
	pc.currentCanvas.Fill()
}

func (pc *PngCanvas) Text(x, y float64, str string) {
	pc.currentCanvas.FillStringAt(str, x, y)
}

func (pc *PngCanvas) Foreground(fg int) int {
	cs := IntColorList(fg)
	r, g, b, _ := pc.currentCanvas.Current.StrokeColor.RGBA()
	old := r<<16 + g<<8 + b
	pc.currentCanvas.SetStrokeColor(color.RGBA{uint8(cs[0]), uint8(cs[1]), uint8(cs[2]), 255})
	pc.currentCanvas.SetFillColor(color.RGBA{uint8(cs[0]), uint8(cs[1]), uint8(cs[2]), 51})
	return int(old)
}

func (pc *PngCanvas) LineStyle(ls int) {
	switch ls {
	case CONTINUOUS:
		pc.currentCanvas.SetLineDash(nil, 0.0)
	case DOTTED:
		pc.currentCanvas.SetLineDash([]float64{2.0, 2.0}, 0.0)
	case DASHED:
	case DASH_DOT:
		pc.currentCanvas.SetLineDash([]float64{10.0, 5.0, 2.0, 5.0}, 0.0)
	}
}

func (pc *PngCanvas) TextAlignment(ta int) {
}

func (pc *PngCanvas) TextOrientation(to float64) {
}

func (pc *PngCanvas) SectionAlias(sa int) (string, bool) {
	return "", false
}

func (pc *PngCanvas) SelectedNodes() []*Node {
	return nil
}

func (pc *PngCanvas) SelectedElems() []*Elem {
	return nil
}

func (pc *PngCanvas) ElemSelected() bool {
	return false
}

func (pc *PngCanvas) DefaultStyle() {
}

func (pc *PngCanvas) BondStyle(show *Show) {
}

func (pc *PngCanvas) PhingeStyle(show *Show) {
}

func (pc *PngCanvas) ConfStyle(show *Show) {
}

func (pc *PngCanvas) SelectNodeStyle() {
}

func (pc *PngCanvas) SelectElemStyle() {
}

func (pc *PngCanvas) ShowPrintRange() bool {
	return false
}

func (pc *PngCanvas) GetCanvasSize() (int, int) {
	return pc.width, pc.height
}

func (pc *PngCanvas) CanvasPaperSize() (float64, float64, error) {
	return float64(pc.width), float64(pc.height), nil
}

func (pc *PngCanvas) Flush() {
}

func (pc *PngCanvas) CanvasDirection() int {
	return 1
}
