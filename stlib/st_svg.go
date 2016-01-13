package st

import (
	"github.com/ajstarks/svgo"
	"os"
)

func rgbString(color int) string {
	return "#fff"
}

func (stw *SVGCanvas) Line(x1, y1, x2, y2 float64) {
	stw.currentCanvas.Line(int(x1), int(y1), int(x2), int(y2), stw.currentStyle.Stroke())
}

func (stw *SVGCanvas) Polyline(coord [][]float64) {
	xs := make([]int, len(coord))
	ys := make([]int, len(coord))
	for i := 0; i< len(coord); i++ {
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
	for i := 0; i< len(coord); i++ {
		xs[i] = int(coord[i][0])
		ys[i] = int(coord[i][1])
	}
	stw.currentCanvas.Polygon(xs, ys, stw.currentStyle.Fill())
}

func (stw *SVGCanvas) Circle(x, y, r float64) {
	stw.currentCanvas.Circle(int(x), int(y), int(r), stw.currentStyle.Fill())
}

func (stw *SVGCanvas) FilledCircle(x, y, r float64) {
}

func (stw *SVGCanvas) Text(x, y float64, txt string) {
	stw.currentCanvas.Text(int(x), int(y), txt, stw.currentStyle.Text())
}

func (stw *SVGCanvas) Foreground(fg int) {
	c := rgbString(fg)
	stw.currentStyle.Set("stroke", c)
	stw.currentStyle.Set("fill", c)
}

func (stw *SVGCanvas) LineStyle(ls int) {
	switch ls {
	case CONTINUOUS:
		stw.currentStyle.Delete("stroke-dasharray")
	case DOTTED:
		stw.currentStyle.Set("stroke-dasharray", "2,2")
	case DASHED:
	case DASH_DOT:
		stw.currentStyle.Set("stroke-dasharray", "10,5,2,5")
	}
}

func (stw *SVGCanvas) TextAlignment(ta int) {
	switch ta {
	case SOUTH:
	case NORTH:
	case WEST:
	case EAST:
	case CENTER:
		stw.currentStyle.Set("alignment-baseline", "central")
        stw.currentStyle.Set("text-anchor", "middle")
	}
}

func (stw *SVGCanvas) TextOrientation(to float64) {
}

func (stw *SVGCanvas) SectionAliase(s int) (string, bool) {
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
	stw.currentStyle.Set("font-size", "9px")
	stw.currentStyle.Set("fill", "black")
	stw.currentStyle.Set("stroke", "black")
}

func (stw *SVGCanvas) BondStyle(show *Show) {
	stw.currentStyle.Set("fill", "none")
	stw.currentStyle.Set("stroke", "black")
}

func (stw *SVGCanvas) PhingeStyle(show *Show) {
	stw.currentStyle.Set("fill", "black")
	stw.currentStyle.Set("stroke", "black")
}

func (stw *SVGCanvas) ConfStyle(show *Show) {
	stw.currentStyle.Set("fill", "black")
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
	return 0, 0
}

func (stw *SVGCanvas) CanvasPaperSize() (float64, float64, error) {
	return 0.0, 0.0, nil
}

func (stw *SVGCanvas) Flush() {
	stw.currentCanvas.End()
}

func (stw *SVGCanvas) CanvasDirection() int {
	return 1
}

type SVGCanvas struct {
	currentCanvas *svg.SVG
	currentStyle *Style
}

func PrintSVG(frame *Frame, otp string) error {
	s := frame.Show.Copy()
	v := frame.View.Copy()
	w, err := os.Create(otp)
	if err != nil {
		return err
	}
	defer w.Close()
	cvs := svg.New(w)
	cvs.Start(1000, 1000)
	sc := new(SVGCanvas)
	sc.currentCanvas = cvs
	w.WriteString(`<style>
    .ndcap {
        font-family: "IPAmincho";
        font-size: 9px;
    }
    .elcap {
        font-family: "IPAmincho";
        font-size: 9px;
    }
    .sttext {
        font-family: "IPAmincho";
        font-size: 9px;
    }
    .kijun {
        font-family: "IPAmincho";
        font-size: 9px;
        text-anchor: middle;
        alignment-baseline: central;
    }
</style>
`)
	DrawFrame(sc, frame, ECOLOR_BLACK, true)
	frame.Show = s
	frame.View = v
	if err != nil {
		return err
	}
	return nil
}

type Style struct {
}
func NewStyle() *Style {
	return new(Style)
}
func (s *Style) Delete(string) {
}
func (s *Style) Copy() *Style {
	return new(Style)
}
func (s *Style) Set(string, string) {
}
func (s *Style) Fill() string{
	return ""
}
func (s *Style) Stroke() string{
	return ""
}
func (s *Style) Text() string{
	return ""
}
