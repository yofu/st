package st

import (
	"github.com/llgcode/draw2d/draw2dimg"
	"image"
)

// TODO: font

type PngCanvas struct {
	currentCanvas *draw2dimg.GraphicContext
}

func PrintPNG (frame *Frame, otp string) error {
	dest := image.NewRGBA(image.Rect(0.0, 0.0, 210.0, 297.0))
	gc := draw2dimg.NewGraphicContext(dest)
	pc := &PngCanvas{
		currentCanvas: gc,
	}
	DrawFrame(pc, frame, ECOLOR_BLACK, true)
	return draw2dimg.SaveToPngFile(otp, dest)
}

func (pc *PngCanvas) Line(x1, y1, x2, y2 float64) {
	pc.currentCanvas.MoveTo(x1, y1)
	pc.currentCanvas.LineTo(x2, y2)
	pc.currentCanvas.Close()
	pc.currentCanvas.Stroke()
}

func (pc *PngCanvas) Polyline(coord [][]float64) {
	if len(coord) < 2 {
		return
	}
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
	pc.currentCanvas.MoveTo(coord[0][0], coord[0][1])
	for _, c := range coord[1:] {
		pc.currentCanvas.LineTo(c[0], c[1])
	}
	pc.currentCanvas.Close()
	pc.currentCanvas.Fill()
}

func (pc *PngCanvas) Circle(x, y, r float64) {
}

func (pc *PngCanvas) FilledCircle(x, y, r float64) {
}

func (pc *PngCanvas) Text(x, y float64, str string) {
	pc.currentCanvas.FillStringAt(str, x, y)
}

func (pc *PngCanvas) Foreground(fg int) {
}

func (pc *PngCanvas) LineStyle(ls int) {
}

func (pc *PngCanvas) TextAlignment(ta int) {
}

func (pc *PngCanvas) TextOrientation(to float64) {
}

func (pc *PngCanvas) SectionAliase(sa int) (string, bool) {
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
	return 0, 0
}

func (pc *PngCanvas) CanvasPaperSize() (float64, float64, error) {
	return 0.0, 0.0, nil
}

func (pc *PngCanvas) Flush() {
}

func (pc *PngCanvas) CanvasDirection() int {
	return 1
}
