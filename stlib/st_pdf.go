package st

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jung-kurt/gofpdf"
)

var (
	defaultfont = "ipam.ttf"
	scale       = 0.25
)

type PDFCanvas struct {
	*Alias
	canvas *gofpdf.Fpdf
	width  float64
	height float64
}

func NewPDFCanvas(name, style string, width, height float64) (*PDFCanvas, error) {
	pdf := gofpdf.New(style, "mm", name, filepath.Join(os.Getenv("HOME"), ".st/fonts"))
	if err := pdf.Error(); err != nil {
		return nil, err
	}
	pdf.AddUTF8Font("IPA明朝", "", defaultfont)
	pdf.SetFont("IPA明朝", "", 8)
	pdf.AddPage()
	if err := pdf.Error(); err != nil {
		return nil, err
	}
	pdf.SetLineWidth(0.1)
	return &PDFCanvas{
		Alias:  NewAlias(),
		canvas: pdf,
		width:  width / scale,
		height: height / scale,
	}, nil
}

func (p *PDFCanvas) Draw(frame *Frame, textBox []*TextBox) {
	DrawFrame(p, frame, frame.Show.ColorMode, false)
	p.Foreground(BLACK)
	for _, t := range textBox {
		fmt.Println(t.value, t.position)
		if !t.IsHidden(frame.Show) {
			DrawText(p, t)
		}
	}
}

func (p *PDFCanvas) SaveAs(fn string) error {
	p.canvas.OutputFileAndClose(fn)
	if err := p.canvas.Error(); err != nil {
		return err
	}
	return nil
}

func (p *PDFCanvas) Line(x1, y1, x2, y2 float64) {
	p.Foreground(BLACK)
	p.canvas.MoveTo(x1*scale, y1*scale)
	p.canvas.LineTo(x2*scale, y2*scale)
}

func (p *PDFCanvas) Polyline(coords [][]float64) {
	p.Foreground(BLACK)
	points := make([]gofpdf.PointType, len(coords))
	for i := 0; i < len(coords); i++ {
		points[i] = gofpdf.PointType{X: coords[i][0] * scale, Y: coords[i][1] * scale}
	}
	p.canvas.Polygon(points, "D")
}

func (p *PDFCanvas) Polygon(coords [][]float64) {
	p.canvas.SetAlpha(0.3, "Normal")
	points := make([]gofpdf.PointType, len(coords))
	for i := 0; i < len(coords); i++ {
		points[i] = gofpdf.PointType{X: coords[i][0] * scale, Y: coords[i][1] * scale}
	}
	p.canvas.Polygon(points, "DF")
	p.canvas.SetAlpha(1.0, "Normal")
}

func (p *PDFCanvas) Circle(x, y, d float64) {
	p.canvas.Circle(x*scale, y*scale, d*0.5*scale, "D")
}

func (p *PDFCanvas) FilledCircle(x, y, d float64) {
	p.canvas.Circle(x*scale, y*scale, d*0.5*scale, "DF")
}

func (p *PDFCanvas) Text(x, y float64, str string) {
	for _, s := range strings.Split(strings.TrimSuffix(str, "\n"), "\n") {
		p.canvas.Text(x*scale, y*scale, s)
	}
}

func (p *PDFCanvas) Foreground(fg int) int {
	if fg == WHITE {
		fg = BLACK
	}
	r, g, b := p.canvas.GetDrawColor()
	lis := IntColorList(fg)
	p.canvas.SetDrawColor(lis[0], lis[1], lis[2])
	p.canvas.SetFillColor(lis[0], lis[1], lis[2])
	p.canvas.SetTextColor(lis[0], lis[1], lis[2])
	return r<<16 + g<<8 + b
}

func (p *PDFCanvas) LineStyle(ls int) {
	switch ls {
	case CONTINUOUS:
		p.canvas.SetDashPattern([]float64{}, 0.0)
	case DOTTED:
		p.canvas.SetDashPattern([]float64{2.0, 2.0}, 0.0)
	case DASHED:
		p.canvas.SetDashPattern([]float64{10.0, 5.0}, 0.0)
	case DASH_DOT:
		p.canvas.SetDashPattern([]float64{10.0, 5.0, 2.0, 5.0}, 0.0)
	}
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
	p.Foreground(BLACK)
	p.LineStyle(CONTINUOUS)
}

func (p *PDFCanvas) BondStyle(show *Show) {
	p.Foreground(BLACK)
}

func (p *PDFCanvas) PhingeStyle(show *Show) {
	p.Foreground(BLACK)
}

func (p *PDFCanvas) ConfStyle(show *Show) {
	p.Foreground(BLACK)
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
