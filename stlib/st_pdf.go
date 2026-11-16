package st

import (
	"fmt"
	"golang.org/x/image/font/gofont/goregular"
	"math"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/signintech/gopdf"
)

var (
	defaultfont = filepath.Join(home, ".st/fonts/ipam.ttf")
	scale       = 0.25
	opaque      = gopdf.Transparency{
		Alpha:         1.0,
		BlendModeType: gopdf.NormalBlendMode,
	}
)

type PDFCanvas struct {
	*Alias
	canvas                   *gopdf.GoPdf
	width                    float64
	coordScale               float64
	originX, originY         float64
	alignment                int
	fig2                     bool
	height                   float64
	torient                  float64
	transparency             gopdf.Transparency
	dash                     []float64
	fig2FontSize, foreground int
	stressLabels             []fig2StressLabel
	reservedLabels           []fig2LabelRect
	layoutErr                error
	placingLabels            bool
	strokeWidth              float64
}

func NewPDFCanvas(name, style string, width, height float64) (*PDFCanvas, error) {
	var psize gopdf.Rect
	switch name {
	case "A3":
		psize = *gopdf.PageSizeA3
	case "A4":
		psize = *gopdf.PageSizeA4
	}
	if psize.W == 0 || psize.H == 0 {
		return nil, fmt.Errorf("unknown paper size %q", name)
	}
	return newPDFCanvas(psize, width/scale, height/scale, scale, true)
}

func newPDFCanvas(psize gopdf.Rect, width, height, factor float64, addPage bool) (*PDFCanvas, error) {
	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{
		PageSize: psize,
	})
	if addPage {
		pdf.AddPage()
	}
	err := pdf.AddTTFFont("IPA明朝", defaultfont)
	if err != nil && runtime.GOOS == "js" {
		err = pdf.AddTTFFontData("IPA明朝", goregular.TTF)
	}
	if err != nil {
		return nil, err
	}
	err = pdf.SetFont("IPA明朝", "", 8)
	if err != nil {
		return nil, err
	}
	return &PDFCanvas{
		Alias:      NewAlias(),
		canvas:     pdf,
		width:      width,
		coordScale: factor,
		height:     height,
		torient:    0.0,
		transparency: gopdf.Transparency{
			Alpha:         0.5,
			BlendModeType: gopdf.NormalBlendMode,
		},
	}, nil
}

func (p *PDFCanvas) Draw(frame *Frame, textBox []*TextBox) {
	if p.fig2 {
		p.SetFig2FontSize(6)
	}
	DrawFrame(p, frame, frame.Show.ColorMode, false)
	p.Foreground(BLACK)
	for _, t := range textBox {

		if !t.IsHidden(frame.Show) {
			if p.fig2 {
				p.drawFig2Text(t)
			} else {
				DrawText(p, t)
			}
		}
	}
	if p.fig2 {
		p.drawFig2StressLabels()
	}
}

func (p *PDFCanvas) SaveAs(fn string) error {
	return p.canvas.WritePdf(fn)
}

func (p *PDFCanvas) Line(x1, y1, x2, y2 float64) {
	if !finitePDF(x1, y1, x2, y2) {
		return
	}
	p.canvas.SetTransparency(opaque)
	p.strokePath([][]float64{{x1, y1}, {x2, y2}})
}

func (p *PDFCanvas) Polyline(coords [][]float64) {
	if p.fig2 {
		p.canvas.SetTransparency(opaque)
		p.strokePath(coords)
		return
	}
	p.canvas.SetTransparency(opaque)
	points := make([]gopdf.Point, len(coords))
	for i := 0; i < len(coords); i++ {
		if !finitePDF(coords[i][0], coords[i][1]) {
			return
		}
		points[i] = gopdf.Point{X: p.x(coords[i][0]), Y: p.y(coords[i][1])}
	}
	p.canvas.Polygon(points, "D")
}

func (p *PDFCanvas) Polygon(coords [][]float64) {
	p.canvas.SetTransparency(p.transparency)
	points := make([]gopdf.Point, len(coords))
	for i := 0; i < len(coords); i++ {
		if !finitePDF(coords[i][0], coords[i][1]) {
			return
		}
		points[i] = gopdf.Point{X: p.x(coords[i][0]), Y: p.y(coords[i][1])}
	}
	p.canvas.Polygon(points, "F")
}

func (p *PDFCanvas) Circle(x, y, d float64) {
	if !finitePDF(x, y, d) {
		return
	}
	p.canvas.SetTransparency(opaque)
	p.canvas.Oval(p.x(x-d/2), p.y(y-d/2), p.x(x+d/2), p.y(y+d/2))
}

func (p *PDFCanvas) FilledCircle(x, y, d float64) {
	if !finitePDF(x, y, d) {
		return
	}
	p.canvas.SetTransparency(opaque)
	p.canvas.Oval(p.x(x-d/2), p.y(y-d/2), p.x(x+d/2), p.y(y+d/2))
}

func (p *PDFCanvas) x(x float64) float64 { return (p.originX + x) * p.coordScale }
func (p *PDFCanvas) y(y float64) float64 { return (p.originY + y) * p.coordScale }

func (p *PDFCanvas) Text(x, y float64, str string) {
	if !finitePDF(x, y, p.torient) {
		return
	}
	p.canvas.SetTransparency(opaque)
	x, y = p.x(x), p.y(y)
	if p.fig2 {
		p.fig2TextPoints(x, y, str)
		return
	}
	p.canvas.Rotate(p.torient, x, y)
	for _, line := range strings.Split(str, "\n") {
		left := x
		p.canvas.SetX(left)
		p.canvas.SetY(y)
		p.canvas.Text(line)
		y += 8
	}
	p.canvas.RotateReset()
}

func (p *PDFCanvas) Foreground(fg int) int {
	if fg == WHITE {
		fg = BLACK
	}
	p.foreground = fg
	lis := IntColorList(fg)
	p.canvas.SetStrokeColor(uint8(lis[0]), uint8(lis[1]), uint8(lis[2]))
	p.canvas.SetFillColor(uint8(lis[0]), uint8(lis[1]), uint8(lis[2]))
	p.canvas.SetTextColor(uint8(lis[0]), uint8(lis[1]), uint8(lis[2]))
	// return r<<16 + g<<8 + b
	return fg
}

func (p *PDFCanvas) LineStyle(ls int) {
	if p.fig2 {
		p.dash = map[int][]float64{DOTTED: {.5, .5}, DASHED: {1, 1}, DASH_DOT: {6, 2, .5, 2}}[ls]
		return
	}
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

func (p *PDFCanvas) TextAlignment(alignment int) {
	p.alignment = alignment
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
	if p.fig2 {
		p.SetFig2Stroke(.05, nil)
	}
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

func finitePDF(values ...float64) bool {
	for _, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}

func (p *PDFCanvas) SetFig2Stroke(width float64, dash []float64) {
	p.strokeWidth = width
	p.canvas.SetLineWidth(width * fig2PointsPerMM)
	p.dash = dash
}

func (p *PDFCanvas) strokePath(coords [][]float64) {
	for _, point := range coords {
		if len(point) < 2 || !finitePDF(point[0], point[1]) {
			return
		}
	}
	pattern := p.dash
	if len(pattern)%2 == 1 {
		pattern = append(append([]float64(nil), pattern...), pattern...)
	}
	index, used := 0, 0.0
	for i := 1; i < len(coords); i++ {
		a, b := coords[i-1], coords[i]
		x0, y0, x1, y1 := p.x(a[0]), p.y(a[1]), p.x(b[0]), p.y(b[1])
		if !p.fig2 || len(pattern) == 0 {
			p.canvas.Line(x0, y0, x1, y1)
			continue
		}
		dx, dy := x1-x0, y1-y0
		length := math.Hypot(dx, dy)
		for start := 0.0; start < length; {
			left := pattern[index] - used
			end := min(length, start+left)
			if end <= start {
				index = (index + 1) % len(pattern)
				used = 0
				continue
			}
			if index%2 == 0 {
				p.canvas.Line(x0+dx*start/length, y0+dy*start/length, x0+dx*end/length, y0+dy*end/length)
			}
			if end-start >= left-1e-9 {
				index = (index + 1) % len(pattern)
				used = 0
			} else {
				used += end - start
			}
			start = end
		}
	}
}

func (p *PDFCanvas) fig2TextPoints(x, y float64, str string) {
	size := float64(p.fig2FontSize)
	if size <= 0 {
		size = 6
	}
	// GoPdf's rotation is counterclockwise in PDF coordinates, while the
	// shared renderer measures the slope in downward screen coordinates.
	p.canvas.Rotate(-p.torient, x, y)
	for i, line := range strings.Split(str, "\n") {
		w, _ := p.canvas.MeasureTextWidth(line)
		dx, dy := fig2TextOrigin(p.alignment, w, size)
		if !p.placingLabels && strings.TrimSpace(line) != "" {
			p.reservedLabels = append(p.reservedLabels, fig2LabelBounds(x-p.originX*p.coordScale, y-p.originY*p.coordScale, p.torient, dx, dy+float64(i)*size*1.2, w, size))
		}
		p.canvas.SetX(x + dx)
		p.canvas.SetY(y + dy + float64(i)*size*1.2)
		p.canvas.Text(line)
	}
	p.canvas.RotateReset()
}

func (p *PDFCanvas) drawFig2Text(b *TextBox) {
	p.SetFig2FontSize(b.Font.Size())
	p.Foreground(b.Font.Color())
	p.TextOrientation(0)
	p.TextAlignment(fig2NorthWest)
	x, y := b.Position()
	y = p.height - y
	for _, line := range b.Text() {
		if strings.TrimSpace(line) != "" {
			p.Text(x, y, line)
		}
		y += 1.2 * float64(p.fig2FontSize) / fig2PointsPerMM
	}
	p.SetFig2FontSize(6)
}

func (p *PDFCanvas) SetFig2FontSize(size int) {
	if p.fig2 {
		p.fig2FontSize = size
		p.canvas.SetFontSize(float64(size))
	}
}

func (p *PDFCanvas) Fig2DeformationLine(a, b []float64) bool {
	if !p.fig2 {
		return false
	}
	width, dash, color := p.strokeWidth, p.dash, p.foreground
	p.SetFig2Stroke(.05, []float64{1, 1})
	p.Foreground(BLACK)
	p.Line(a[0], a[1], b[0], b[1])
	p.SetFig2Stroke(width, dash)
	p.Foreground(color)
	return true
}

func (p *PDFCanvas) Fig2MomentLine(coords [][]float64) bool {
	if !p.fig2 {
		return false
	}
	width, dash := p.strokeWidth, p.dash
	p.SetFig2Stroke(.01, nil)
	p.Polyline(coords)
	p.SetFig2Stroke(width, dash)
	return true
}
