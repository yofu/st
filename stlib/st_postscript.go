package st

import (
	"github.com/yofu/ps"
	"os"
)

type PostScriptCanvas struct {
	doc *ps.Doc
	paper ps.Paper
}

func NewPostScriptCanvas(fn string, paper ps.Paper) (*PostScriptCanvas, error) {
	doc := ps.NewDoc(fn)
	doc.SetPaperSize(paper)
	doc.Canvas.NewPage("1", paper)
	return &PostScriptCanvas{
		doc: doc,
		paper: paper,
	}, nil
}

func (pc *PostScriptCanvas) WriteTo(fn string) error {
	w, err := os.Create(fn)
	defer w.Close()
	if err != nil {
		return err
	}
	_, err = pc.doc.WriteTo(w)
	if err != nil {
		return err
	}
	return nil
}

func (pc *PostScriptCanvas) Line(x1, y1, x2, y2 float64) {
	pc.doc.Canvas.FLine(x1, y1, x2, y2)
}

func (pc *PostScriptCanvas) Polyline(coord [][]float64) {
	if len(coord) < 1 {
		return
	}
	pc.doc.Canvas.FPolyline(coord)
}

func (pc *PostScriptCanvas) Polygon(coord [][]float64) {
	if len(coord) < 1 {
		return
	}
	pc.doc.Canvas.FPolygon(coord)
}

func (pc *PostScriptCanvas) Circle(x, y, d float64) {
	pc.doc.Canvas.FCircle(x, y, 0.5*d)
}

func (pc *PostScriptCanvas) FilledCircle(x, y, d float64) {
	pc.doc.Canvas.FFilledCircle(x, y, 0.5*d)
}

func (pc *PostScriptCanvas) Text(x, y float64, txt string) {
}

func (pc *PostScriptCanvas) Foreground(fg int) {
	c := IntColorFloat64(fg)
	pc.doc.Canvas.SetRGBColor(c[0], c[1], c[2])
}

func (pc *PostScriptCanvas) LineStyle(ls int) {
}

func (pc *PostScriptCanvas) TextAlignment(ta int) {
}

func (pc *PostScriptCanvas) TextOrientation(to float64) {
}

func (pc *PostScriptCanvas) SectionAlias(s int) (string, bool) {
	return "", false
}

func (pc *PostScriptCanvas) SelectedNodes() []*Node {
	return nil
}

func (pc *PostScriptCanvas) SelectedElems() []*Elem {
	return nil
}

func (pc *PostScriptCanvas) ElemSelected() bool {
	return false
}

func (pc *PostScriptCanvas) DefaultStyle() {
}

func (pc *PostScriptCanvas) BondStyle(show *Show) {
	pc.doc.Canvas.SetRGBColor(0.0, 0.0, 0.0)
}

func (pc *PostScriptCanvas) PhingeStyle(show *Show) {
	pc.doc.Canvas.SetRGBColor(0.0, 0.0, 0.0)
}

func (pc *PostScriptCanvas) ConfStyle(show *Show) {
	pc.doc.Canvas.SetRGBColor(0.0, 0.0, 0.0)
}

func (pc *PostScriptCanvas) SelectNodeStyle() {
}

func (pc *PostScriptCanvas) SelectElemStyle() {
}

func (pc *PostScriptCanvas) ShowPrintRange() bool {
	return false
}

func (pc *PostScriptCanvas) GetCanvasSize() (int, int) {
	return pc.paper.Size()
}

func (pc *PostScriptCanvas) CanvasPaperSize() (float64, float64, error) {
	w, h := pc.paper.Size()
	return float64(w), float64(h), nil
}

func (pc *PostScriptCanvas) Flush() {
	pc.doc.Canvas.NewPage("", pc.paper)
}

func (pc *PostScriptCanvas) CanvasDirection() int {
	if pc.paper.Portrait {
		return 0
	} else {
		return 1
	}
}
func (pc *PostScriptCanvas) Draw(frame *Frame) {
	DrawFrame(pc, frame, frame.Show.ColorMode, false)
}

func CentringTo(view *View, paper ps.Paper) {
	w, h := paper.Size()
	if paper.Portrait {
		view.Center[0] = float64(w) * 0.5
		view.Center[1] = float64(h) * 0.5
	} else {
		view.Center[0] = float64(h) * 0.5
		view.Center[1] = float64(w) * 0.5
	}
}

func (frame *Frame) PrintPostScript(fn string, paper ps.Paper) error {
	pc, err := NewPostScriptCanvas(fn, paper)
	if err != nil {
		return err
	}
	v := frame.View.Copy()
	CentringTo(frame.View, paper)
	pc.Draw(frame)
	err = pc.WriteTo(fn)
	frame.View = v
	return err
}
