package stshiny

import (
	"image/color"
	"github.com/yofu/st/stlib"
)

func (stw *Window) Line(x1, y1, x2, y2 float64) {
	dx := int(x2) - int(x1)
	if dx < 0 {
		dx = -dx
	}
	dy := int(y2) - int(y1)
	if dy < 0 {
		dy = -dy
	}
	var sx, sy int
	if x1 < x2 {
		sx = 1
	} else {
		sx = -1
	}
	if y1 < y2 {
		sy = 1
	} else {
		sy = -1
	}
	eps := dx - dy
	x := int(x1)
	y := int(y1)
	endx := int(x2)
	endy := int(y2)
	var e2 int
	for {
		stw.currentCanvas.SetRGBA(x, y, stw.currentPen)
		if x == endx && y == endy {
			break
		}
		e2 = 2.0*eps
		if e2 > -dy {
			eps = eps - dy
			x = x + sx
		}
		if e2 < dx {
			eps = eps + dx
			y = y + sy
		}
	}
	return
}

func (stw *Window) Polyline([][]float64) {
}

func (stw *Window) Polygon([][]float64) {
}

func (stw *Window) Circle(x1, y1, d float64) {
	cx := 0
	cy := int(0.5 * float64(d) + 1)
	dd := - int(d) * int(d) + 4*cy*cy - 4*cy + 2
	dx := 4
	dy := -8*cy + 8
	x := int(x1)
	y := int(y1)
	if (int(d)&1) == 0 {
		x++
		y++
	}
	for cx = 0; cx <= cy; cx++ {
		if dd > 0 {
			dd += dy
			dy += 8
			cy--
		}
		stw.currentCanvas.SetRGBA(cy+x, cx+y, color.RGBA{0xff, 0xff, 0xff, 0xff})
		stw.currentCanvas.SetRGBA(cx+x, cy+y, color.RGBA{0xff, 0xff, 0xff, 0xff})
		stw.currentCanvas.SetRGBA(-cx+x, cy+y, color.RGBA{0xff, 0xff, 0xff, 0xff})
		stw.currentCanvas.SetRGBA(-cy+x, cx+y, color.RGBA{0xff, 0xff, 0xff, 0xff})
		stw.currentCanvas.SetRGBA(-cy+x, -cx+y, color.RGBA{0xff, 0xff, 0xff, 0xff})
		stw.currentCanvas.SetRGBA(-cx+x, -cy+y, color.RGBA{0xff, 0xff, 0xff, 0xff})
		stw.currentCanvas.SetRGBA(cx+x, -cy+y, color.RGBA{0xff, 0xff, 0xff, 0xff})
		stw.currentCanvas.SetRGBA(cy+x, -cx+y, color.RGBA{0xff, 0xff, 0xff, 0xff})
		dd += dx
		dx += 8
	}
}

func (stw *Window) FilledCircle(float64, float64, float64) {
}

func (stw *Window) Text(float64, float64, string) {
}

func (stw *Window) Foreground(fg int) {
	col := st.IntColorList(fg)
	stw.currentPen = color.RGBA{uint8(col[0]), uint8(col[1]), uint8(col[2]), 0xff}
}

func (stw *Window) LineStyle(int) {
}

func (stw *Window) TextAlignment(int) {
}

func (stw *Window) TextOrientation(float64) {
}

func (stw *Window) SectionAliase(int) (string, bool) {
	return "", false
}

func (stw *Window) SelectedNodes() []*st.Node {
	return nil
}

func (stw *Window) SelectedElems() []*st.Elem {
	return nil
}

func (stw *Window) ElemSelected() bool {
	return false
}

func (stw *Window) DefaultStyle() {
}

func (stw *Window) BondStyle(*st.Show) {
}

func (stw *Window) PhingeStyle(*st.Show) {
}

func (stw *Window) ConfStyle(*st.Show) {
}

func (stw *Window) SelectNodeStyle() {
}

func (stw *Window) SelectElemStyle() {
}

func (stw *Window) ShowPrintRange() bool {
	return false
}

func (stw *Window) GetCanvasSize() (int, int) {
	return 0, 0
}

func (stw *Window) CanvasPaperSize() (float64, float64, error) {
	return 0.0, 0.0, nil
}

func (stw *Window) Flush() {
}

func (stw *Window) CanvasDirection() int {
	return 1
}

