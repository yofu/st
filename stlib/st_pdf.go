package st

import (
	"path/filepath"
	"os"
	"strings"

	"github.com/signintech/gopdf"
)

var (
	defaultfont = filepath.Join(os.Getenv("HOME"), ".st/fonts/ipam.ttf")
)

type PDFCanvas struct {
	*Alias
	canvas gopdf.GoPdf
	width  float64
	height float64
}

func NewPDFCanvas(width, height float64) (*PDFCanvas, error) {
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{
		Unit: "pt",
		PageSize: gopdf.Rect{W: height, H: width}, // TODO: gopdf's bug
	})
	pdf.AddPage()
	err := pdf.AddTTFFont("IPA明朝", defaultfont)
	if err != nil {
		return nil, err
	}
	err = pdf.SetFont("IPA明朝", "", 8)
	if err != nil {
		return nil, err
	}
	pdf.SetGrayFill(0.5)
	return &PDFCanvas{
		Alias:  NewAlias(),
		canvas: pdf,
		width:  width,
		height: height,
	}, nil
}

func (p *PDFCanvas) Draw(frame *Frame) {
	DrawFrame(p, frame, frame.Show.ColorMode, false)
}

func (p *PDFCanvas) SaveAs(fn string) {
	p.canvas.WritePdf(fn)
}

func (p *PDFCanvas) Line(x1, y1, x2, y2 float64) {
	p.canvas.Line(x1, y1, x2, y2)
}

func (p *PDFCanvas) Polyline(coords [][]float64) {
	for i := 0; i < len(coords) - 1; i++ {
		p.canvas.Line(coords[i][0], coords[i][1], coords[i+1][0], coords[i+1][1])
	}
}

func (p *PDFCanvas) Polygon([][]float64) {
}

func (p *PDFCanvas) Circle(float64, float64, float64) {
}

func (p *PDFCanvas) FilledCircle(float64, float64, float64) {
}

func (p *PDFCanvas) Text(x, y float64, str string) {
	r := &gopdf.Rect{x, y}
	for _, s := range strings.Split(strings.TrimSuffix(str, "\n"), "\n") {
		p.canvas.Cell(r, s) // TODO: fix
	}
}

func (p *PDFCanvas) Foreground(int) {
}

func (p *PDFCanvas) LineStyle(int) {
}

func (p *PDFCanvas) TextAlignment(int) {
}

func (p *PDFCanvas) TextOrientation(float64) {
}

func (p *PDFCanvas) SelectedNodes() []*Node {
	return nil
}

func (p *PDFCanvas) SelectedElems() []*Elem {
	return nil
}

func (p *PDFCanvas) ElemSelected() bool {
	return false
}

func (p *PDFCanvas) DefaultStyle() {
}

func (p *PDFCanvas) BondStyle(*Show) {
}

func (p *PDFCanvas) PhingeStyle(*Show) {
}

func (p *PDFCanvas) ConfStyle(*Show) {
}

func (p *PDFCanvas) SelectNodeStyle() {
}

func (p *PDFCanvas) SelectElemStyle() {
}

func (p *PDFCanvas) ShowPrintRange() bool {
	return false
}

func (p *PDFCanvas) GetCanvasSize() (int, int) {
	return int(p.width), int(p.height)
}

func (p *PDFCanvas) CanvasPaperSize() (float64, float64, error) {
	return p.width, p.height, nil
}

func (p *PDFCanvas) Flush() {
	p.canvas.AddPage()
}

func (p *PDFCanvas) CanvasDirection() int {
	return 1
}