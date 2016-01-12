package stgui

import (
	"github.com/visualfc/go-iup/cd"
	"github.com/yofu/st/stlib"
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
	cvs.End()
}

func (stw *Window) Polygon(coords [][]float64) {
	cvs := stw.currentCanvas
	cvs.Begin(cd.CD_FILL)
	for _, c := range coords {
		cvs.FVertex(c[0], c[1])
	}
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

func (stw *Window) Foreground(fg int) {
	stw.currentCanvas.Foreground(fg)
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
func DrawText(t *TextBox, cvs *cd.Canvas) {
	s := cvs.SaveState()
	cvs.Font(t.Font.Face, cd.CD_PLAIN, t.Font.Size)
	cvs.Foreground(t.Font.Color)
	for i, txt := range t.Text() {
		xpos := t.position[0]
		ypos := t.position[1] - float64(i*t.Font.Size)*1.5 - float64(t.Font.Size)
		cvs.FText(xpos, ypos, txt)
	}
	cvs.RestoreState(s)
}
