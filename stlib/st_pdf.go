package st

import (
	"fmt"
	"path/filepath"

	"github.com/signintech/gopdf"
)

var (
	defaultfont = filepath.Join(home, ".st/fonts/ipam.ttf")
	scale       = 0.25
	opaque      = gopdf.Transparency{
		Alpha:         1.0,
		BlendModeType: "",
	}
)

type PDFCanvas struct {
	*Alias
	canvas       gopdf.GoPdf
	width        float64
	height       float64
	torient      float64
	transparency gopdf.Transparency
}

func NewPDFCanvas(name, style string, width, height float64) (*PDFCanvas, error) {
	pdf := gopdf.GoPdf{}
	var psize gopdf.Rect
	switch name {
	case "A3":
		psize = *gopdf.PageSizeA3
	case "A4":
		psize = *gopdf.PageSizeA4
	}
	pdf.Start(gopdf.Config{
		PageSize: psize,
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
	return &PDFCanvas{
		Alias:   NewAlias(),
		canvas:  pdf,
		width:   width / scale,
		height:  height / scale,
		torient: 0.0,
		transparency: gopdf.Transparency{
			Alpha:         0.5,
			BlendModeType: "",
		},
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
	return p.canvas.WritePdf(fn)
}

func (p *PDFCanvas) Line(x1, y1, x2, y2 float64) {
	p.canvas.SetTransparency(opaque)
	p.Foreground(BLACK)
	p.canvas.Line(x1*scale, y1*scale, x2*scale, y2*scale)
}

func (p *PDFCanvas) Polyline(coords [][]float64) {
	p.canvas.SetTransparency(opaque)
	points := make([]gopdf.Point, len(coords))
	for i := 0; i < len(coords); i++ {
		points[i] = gopdf.Point{X: coords[i][0], Y: coords[i][1]}
	}
	p.canvas.Polygon(points, "D")
}

func (p *PDFCanvas) Polygon(coords [][]float64) {
	p.canvas.SetTransparency(p.transparency)
	points := make([]gopdf.Point, len(coords))
	for i := 0; i < len(coords); i++ {
		points[i] = gopdf.Point{X: coords[i][0], Y: coords[i][1]}
	}
	p.canvas.Polygon(points, "F")
}

func (p *PDFCanvas) Circle(x, y, d float64) {
	p.canvas.SetTransparency(opaque)
	p.canvas.Oval(x-d/2, y-d/2, x+d/2, y+d/2)
}

func (p *PDFCanvas) FilledCircle(x, y, d float64) {
	// TODO: fill
	p.canvas.SetTransparency(opaque)
	p.canvas.Oval(x-d/2, y-d/2, x+d/2, y+d/2)
}

func (p *PDFCanvas) Text(x, y float64, str string) {
	p.canvas.SetTransparency(opaque)
	p.canvas.SetX(x)
	p.canvas.SetY(y)
	p.canvas.Rotate(p.torient, x, y)
	p.canvas.Text(str)
	p.canvas.RotateReset()
}

func (p *PDFCanvas) Foreground(fg int) int {
	if fg == WHITE {
		fg = BLACK
	}
	// r, g, b := p.canvas.GetDrawColor()
	lis := IntColorList(fg)
	p.canvas.SetStrokeColor(uint8(lis[0]), uint8(lis[1]), uint8(lis[2]))
	p.canvas.SetFillColor(uint8(lis[0]), uint8(lis[1]), uint8(lis[2]))
	p.canvas.SetTextColor(uint8(lis[0]), uint8(lis[1]), uint8(lis[2]))
	// return r<<16 + g<<8 + b
	return fg
}

func (p *PDFCanvas) LineStyle(ls int) {
	switch ls {
	case CONTINUOUS:
		p.canvas.SetLineType("solid")
	case DOTTED:
		p.canvas.SetLineType("dotted")
	case DASHED:
		p.canvas.SetLineType("dashed")
	case DASH_DOT:
		// TODO: add dash-dot
		p.canvas.SetLineType("dashed")
	}
}

func (p *PDFCanvas) TextAlignment(int) {
}

func (p *PDFCanvas) TextOrientation(angle float64) {
	p.torient = angle
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
