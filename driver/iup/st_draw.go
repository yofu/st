package stgui

import (
	"fmt"

	"github.com/visualfc/go-iup/cd"
	st "github.com/yofu/st/stlib"
)

func (stw *Window) Line(x1, y1, x2, y2 float64) {
	stw.currentCanvas.FLine(x1, y1, x2, y2)
}

func (stw *Window) Polyline(coords [][]float64) {
	cvs := stw.currentCanvas
	cvs.Begin(cd.CD_OPEN_LINES)
	for _, c := range coords {
		cvs.FVertex(c[0], c[1])
	}
	cvs.FVertex(coords[0][0], coords[0][1])
	cvs.End()
}

func (stw *Window) Polygon(coords [][]float64) {
	cvs := stw.currentCanvas
	cvs.Begin(cd.CD_FILL)
	for _, c := range coords {
		cvs.FVertex(c[0], c[1])
	}
	cvs.FVertex(coords[0][0], coords[0][1])
	cvs.End()
}

func (stw *Window) Circle(x, y, r float64) {
	stw.currentCanvas.FCircle(x, y, r)
}

func (stw *Window) FilledCircle(x, y, r float64) {
	stw.currentCanvas.FFilledCircle(x, y, r)
}

func (stw *Window) Text(x, y float64, txt string) {
	stw.currentCanvas.FText(x, y, txt)
}

func (stw *Window) Foreground(fg int) int {
	return stw.currentCanvas.Foreground(fg)
}

func (stw *Window) DefaultStyle() {
	stw.currentCanvas.Foreground(cd.CD_WHITE)
	stw.currentCanvas.LineStyle(cd.CD_CONTINUOUS)
	stw.currentCanvas.InteriorStyle(cd.CD_HATCH)
	stw.currentCanvas.Hatch(cd.CD_FDIAGONAL)
}

func (stw *Window) BondStyle(show *st.Show) {
	stw.currentCanvas.Foreground(show.BondColor)
}

func (stw *Window) PhingeStyle(show *st.Show) {
	stw.currentCanvas.InteriorStyle(cd.CD_SOLID)
	stw.currentCanvas.Foreground(show.BondColor)
}

func (stw *Window) ConfStyle(show *st.Show) {
	stw.currentCanvas.InteriorStyle(cd.CD_SOLID)
	stw.currentCanvas.Foreground(show.ConfColor)
}

func (stw *Window) SelectNodeStyle() {
	stw.Foreground(st.RED)
}

func (stw *Window) SelectElemStyle() {
	stw.LineStyle(st.DOTTED)
	stw.currentCanvas.Hatch(cd.CD_DIAGCROSS)
}

func (stw *Window) LineStyle(ls int) {
	switch ls {
	case st.CONTINUOUS:
		stw.currentCanvas.LineStyle(cd.CD_CONTINUOUS)
	case st.DOTTED:
		stw.currentCanvas.LineStyle(cd.CD_DOTTED)
	case st.DASHED:
		stw.currentCanvas.LineStyle(cd.CD_DASHED)
	case st.DASH_DOT:
		stw.currentCanvas.LineStyle(cd.CD_DASH_DOT)
	}
}

func (stw *Window) TextAlignment(ta int) {
	switch ta {
	case st.CENTER:
		stw.currentCanvas.TextAlignment(cd.CD_CENTER)
	case st.NORTH:
		stw.currentCanvas.TextAlignment(cd.CD_NORTH)
	case st.SOUTH:
		stw.currentCanvas.TextAlignment(cd.CD_SOUTH)
	case st.EAST:
		stw.currentCanvas.TextAlignment(cd.CD_EAST)
	case st.WEST:
		stw.currentCanvas.TextAlignment(cd.CD_WEST)
	case st.SOUTH_WEST:
		stw.currentCanvas.TextAlignment(cd.CD_SOUTH_WEST)
	case st.SOUTH_EAST:
		stw.currentCanvas.TextAlignment(cd.CD_SOUTH_EAST)
	}
}

func (stw *Window) TextOrientation(deg float64) {
	stw.currentCanvas.TextOrientation(deg)
}

func (stw *Window) ShowPrintRange() bool {
	return showprintrange
}

func (stw *Window) Flush() {
	stw.currentCanvas.Flush()
}

// TEXT
func (stw *Window) DrawText(t *st.TextBox, cvs *cd.Canvas) {
	s := cvs.SaveState()
	cvs.Font(t.Font.Face(), cd.CD_PLAIN, t.Font.Size())
	cvs.Foreground(t.Font.Color())
	var x0, y0 float64
	if stw.ShowPrintRange() {
		w, h := stw.GetCanvasSize()
		w0, h0, _ := stw.CanvasPaperSize()
		x0 = 0.5 * (float64(w) - w0)
		y0 = 0.5 * (float64(h) - h0)
		fmt.Println(w, h, w0, h0, x0, y0)
	}
	for i, txt := range t.Text() {
		xpos, ypos := t.Position()
		ypos -= float64(i*t.Font.Size())*1.5 + float64(t.Font.Size())
		cvs.FText(x0+xpos, y0+ypos, txt)
	}
	cvs.RestoreState(s)
}
