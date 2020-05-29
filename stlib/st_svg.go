package st

import (
	"bytes"
	"fmt"
	"os"

	svg "github.com/ajstarks/svgo"
)

type SVGCanvas struct {
	currentCanvas *svg.SVG
	currentStyle  *Style
	writer        *os.File
	width         float64
	height        float64
}

func NewSVGCanvas(otp string, width, height float64) (*SVGCanvas, error) {
	w, err := os.Create(otp)
	if err != nil {
		return nil, err
	}
	cvs := svg.New(w)
	cvs.Startunit(int(width), int(height), "mm")
	return &SVGCanvas{
		currentCanvas: cvs,
		currentStyle:  nil,
		writer:        w,
		width:         width * 90.0 / 25.4,
		height:        height * 90.0 / 25.4,
	}, nil
}

func (stw *SVGCanvas) Line(x1, y1, x2, y2 float64) {
	stw.currentCanvas.Line(int(x1), int(y1), int(x2), int(y2), stw.currentStyle.Stroke())
}

func (stw *SVGCanvas) Polyline(coord [][]float64) {
	xs := make([]int, len(coord))
	ys := make([]int, len(coord))
	for i := 0; i < len(coord); i++ {
		xs[i] = int(coord[i][0])
		ys[i] = int(coord[i][1])
	}
	s := stw.currentStyle.Copy()
	s.Set("fill", "none")
	stw.currentCanvas.Polygon(xs, ys, s.Fill())
}

func (stw *SVGCanvas) Polygon(coord [][]float64) {
	xs := make([]int, len(coord))
	ys := make([]int, len(coord))
	for i := 0; i < len(coord); i++ {
		xs[i] = int(coord[i][0])
		ys[i] = int(coord[i][1])
	}
	s := stw.currentStyle.Copy()
	s.Set("stroke", "none")
	stw.currentCanvas.Polygon(xs, ys, s.Fill())
}

func (stw *SVGCanvas) Circle(x, y, d float64) {
	stw.currentCanvas.Circle(int(x), int(y), int(0.5*d), stw.currentStyle.Fill())
}

func (stw *SVGCanvas) FilledCircle(x, y, r float64) {
}

func (stw *SVGCanvas) Text(x, y float64, txt string) {
	deg := stw.currentStyle.Value("rotate")
	if deg != "" {
		stw.currentCanvas.Text(int(x), int(y), txt, stw.currentStyle.Text(), fmt.Sprintf(" transform=\"rotate(%s %d %d)\"", deg, int(x), int(y)))
	} else {
		stw.currentCanvas.Text(int(x), int(y), txt, stw.currentStyle.Text())
	}
}

func (stw *SVGCanvas) Foreground(fg int) int {
	if fg == WHITE {
		fg = BLACK
	}
	c := IntHexColor(fg)
	stw.currentStyle.Set("stroke", c)
	stw.currentStyle.Set("fill", c)
	stw.currentStyle.Set("fill-opacity", "0.5")
	return WHITE
}

func (stw *SVGCanvas) LineStyle(ls int) {
	switch ls {
	case CONTINUOUS:
		stw.currentStyle.Delete("stroke-dasharray")
	case DOTTED:
		stw.currentStyle.Set("stroke-dasharray", "2,2")
	case DASHED:
		stw.currentStyle.Set("stroke-dasharray", "10,5")
	case DASH_DOT:
		stw.currentStyle.Set("stroke-dasharray", "10,5,2,5")
	}
}

func (stw *SVGCanvas) TextAlignment(ta int) {
	switch ta {
	case SOUTH:
		stw.currentStyle.Set("alignment-baseline", "central")
		stw.currentStyle.Set("text-anchor", "middle")
	case NORTH:
		stw.currentStyle.Set("alignment-baseline", "hanging")
		stw.currentStyle.Set("text-anchor", "middle")
	case WEST:
		stw.currentStyle.Set("text-anchor", "start")
	case EAST:
		stw.currentStyle.Set("text-anchor", "end")
	case CENTER:
		stw.currentStyle.Set("alignment-baseline", "central")
		stw.currentStyle.Set("text-anchor", "middle")
	}
}

func (stw *SVGCanvas) TextOrientation(to float64) {
	if to == 0.0 {
		stw.currentStyle.Delete("rotate")
	} else {
		stw.currentStyle.Set("rotate", fmt.Sprintf("%f", to))
	}
}

func (stw *SVGCanvas) SectionAlias(s int) (string, bool) {
	return "", false
}

func (stw *SVGCanvas) SelectedNodes() []*Node {
	return nil
}

func (stw *SVGCanvas) SelectedElems() []*Elem {
	return nil
}

func (stw *SVGCanvas) ElemSelected() bool {
	return false
}

func (stw *SVGCanvas) DefaultStyle() {
	stw.currentStyle = NewStyle()
	stw.currentStyle.Set("font-family", "IPAmincho")
	stw.currentStyle.Set("font-size", "8pt")
	stw.currentStyle.Set("fill", "black")
	stw.currentStyle.Set("fill-opacity", "0.5")
	stw.currentStyle.Set("stroke", "black")
}

func (stw *SVGCanvas) BondStyle(show *Show) {
	stw.currentStyle.Set("fill", "none")
	stw.currentStyle.Set("stroke", "black")
}

func (stw *SVGCanvas) PhingeStyle(show *Show) {
	stw.currentStyle.Set("fill", "black")
	stw.currentStyle.Set("fill-opacity", "1.0")
	stw.currentStyle.Set("stroke", "black")
}

func (stw *SVGCanvas) ConfStyle(show *Show) {
	stw.currentStyle.Set("fill", "black")
	stw.currentStyle.Set("fill-opacity", "1.0")
	stw.currentStyle.Set("stroke", "black")
}

func (stw *SVGCanvas) SelectNodeStyle() {
}

func (stw *SVGCanvas) SelectElemStyle() {
}

func (stw *SVGCanvas) ShowPrintRange() bool {
	return false
}

func (stw *SVGCanvas) GetCanvasSize() (int, int) {
	return int(stw.width), int(stw.height)
}

func (stw *SVGCanvas) CanvasPaperSize() (float64, float64, error) {
	return stw.width, stw.height, nil
}

func (stw *SVGCanvas) Flush() {
	stw.currentCanvas.End()
}

func (stw *SVGCanvas) CanvasDirection() int {
	return 1
}

func (stw *SVGCanvas) Close() {
	stw.writer.Close()
}

func (stw *SVGCanvas) Draw(frame *Frame, tb []*TextBox) {
	DrawFrame(stw, frame, frame.Show.ColorMode, false)
	for _, t := range tb {
		if t.IsHidden(frame.Show) {
			continue
		}
		DrawText(stw, t)
	}
	stw.Flush()
}

func PrintSVG(frame *Frame, tb []*TextBox, otp string, width, height float64) error {
	sc, err := NewSVGCanvas(otp, width, height)
	if err != nil {
		return err
	}
	v := frame.View.Copy()
	frame.View.Center[0] = 0.5 * sc.width
	frame.View.Center[1] = 0.5 * sc.height
	sc.Draw(frame, tb)
	sc.Close()
	frame.View = v
	return nil
}

type Style struct {
	value map[string]string
}

func NewStyle() *Style {
	s := new(Style)
	s.value = make(map[string]string)
	return s
}

func (s *Style) Set(k, v string) {
	s.value[k] = v
}

func (s *Style) Value(k string) string {
	return s.value[k]
}

func (s *Style) Delete(k string) {
	delete(s.value, k)
}

func (s *Style) String() string {
	var b bytes.Buffer
	for k, v := range s.value {
		b.WriteString(fmt.Sprintf("%s: %s;", k, v))
	}
	return b.String()
}

func (s *Style) Stroke() string {
	var b bytes.Buffer
	for k, v := range s.value {
		switch k {
		default:
		case "stroke", "stroke-dasharray":
			b.WriteString(fmt.Sprintf("%s: %s;", k, v))
		}
	}
	return b.String()
}

func (s *Style) Fill() string {
	var b bytes.Buffer
	for k, v := range s.value {
		switch k {
		default:
		case "stroke", "stroke-dasharray", "fill", "fill-opacity":
			b.WriteString(fmt.Sprintf("%s: %s;", k, v))
		}
	}
	return b.String()
}

func (s *Style) Text() string {
	var b bytes.Buffer
	for k, v := range s.value {
		switch k {
		default:
		case "font-family", "font-size", "text-anchor", "alignment-baseline":
			b.WriteString(fmt.Sprintf("%s: %s;", k, v))
		}
	}
	return b.String()
}

func (s *Style) Copy() *Style {
	n := NewStyle()
	for k, v := range s.value {
		n.value[k] = v
	}
	return n
}
