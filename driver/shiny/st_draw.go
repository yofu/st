package stshiny

import (
	"image"
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
	cvs := stw.buffer.RGBA()
	for {
		cvs.SetRGBA(x, y, stw.currentPen)
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

func Blend(cvs *image.RGBA, x, y int, src color.RGBA) {
	dst := cvs.RGBAAt(x, y)
	sa := int(src.A) << 8
	da := int(dst.A) * int(255 - src.A)
	cvs.SetRGBA(x, y, color.RGBA{
		uint8((int(src.R) * sa + int(dst.R) * da) >> 16),
		uint8((int(src.G) * sa + int(dst.G) * da) >> 16),
		uint8((int(src.B) * sa + int(dst.B) * da) >> 16),
		uint8((sa + da) >> 8),
	})
}

func (stw *Window) filltriangle(c1, c2, c3 []float64) {
	x1 := int(c1[0])
	y1 := int(c1[1])
	x2 := int(c2[0])
	y2 := int(c2[1])
	x3 := int(c3[0])
	y3 := int(c3[1])
	if y1 > y2 {
		x1, x2 = x2, x1
		y1, y2 = y2, y1
	}
	if y1 > y3 {
		x1, x3 = x3, x1
		y1, y3 = y3, y1
	}
	if y2 > y3 {
		x2, x3 = x3, x2
		y2, y3 = y3, y2
	}
	if y1 == y3 {
		return
	}
	var a1, a2, a3 float64
	top := false
	if y1 != y2 {
		a1 = float64(x2-x1) / float64(y2-y1)
	} else {
		top = true
	}
	a2 = float64(x3-x1) / float64(y3-y1)
	if a1 == a2 {
		return
	}
	if y2 != y3 {
		a3 = float64(x3-x2) / float64(y3-y2)
	}
	cvs := stw.buffer.RGBA()
	if top {
		sx := float64(x1)
		ex := float64(x2)
		if x1 < x2 {
			for y := y2; y != y3; y++ {
				sx += a2
				ex += a3
				x := int(sx)
				end := int(ex)
				for {
					Blend(cvs, x, y, stw.currentBrush)
					if x >= end {
						break
					}
					x++
				}
			}
		} else {
			for y := y2; y != y3; y++ {
				sx += a2
				ex += a3
				x := int(sx)
				end := int(ex)
				for {
					Blend(cvs, x, y, stw.currentBrush)
					if x <= end {
						break
					}
					x--
				}
			}
		}
	} else {
		sx := float64(x1)
		ex := float64(x1)
		if a1 < a2 {
			for y := y1; y != y2; y++ {
				sx += a1
				ex += a2
				x := int(sx)
				end := int(ex)
				for {
					Blend(cvs, x, y, stw.currentBrush)
					if x >= end {
						break
					}
					x++
				}
			}
			for y := y2; y != y3; y++ {
				sx += a3
				ex += a2
				x := int(sx)
				end := int(ex)
				for {
					Blend(cvs, x, y, stw.currentBrush)
					if x >= end {
						break
					}
					x++
				}
			}
		} else {
			for y := y1; y != y2; y++ {
				sx += a1
				ex += a2
				x := int(sx)
				end := int(ex)
				for {
					Blend(cvs, x, y, stw.currentBrush)
					if x <= end {
						break
					}
					x--
				}
			}
			for y := y2; y != y3; y++ {
				sx += a3
				ex += a2
				x := int(sx)
				end := int(ex)
				for {
					Blend(cvs, x, y, stw.currentBrush)
					if x <= end {
						break
					}
					x--
				}
			}
		}
	}
}

func (stw *Window) Polygon(coords [][]float64) {
	if len(coords) < 3 {
		return
	}
	for i := 0; i< len(coords)-2; i++ {
		stw.filltriangle(coords[0], coords[i+1], coords[i+2])
	}
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
	cvs := stw.buffer.RGBA()
	for cx = 0; cx <= cy; cx++ {
		if dd > 0 {
			dd += dy
			dy += 8
			cy--
		}
		cvs.SetRGBA(cy+x, cx+y, color.RGBA{0xff, 0xff, 0xff, 0xff})
		cvs.SetRGBA(cx+x, cy+y, color.RGBA{0xff, 0xff, 0xff, 0xff})
		cvs.SetRGBA(-cx+x, cy+y, color.RGBA{0xff, 0xff, 0xff, 0xff})
		cvs.SetRGBA(-cy+x, cx+y, color.RGBA{0xff, 0xff, 0xff, 0xff})
		cvs.SetRGBA(-cy+x, -cx+y, color.RGBA{0xff, 0xff, 0xff, 0xff})
		cvs.SetRGBA(-cx+x, -cy+y, color.RGBA{0xff, 0xff, 0xff, 0xff})
		cvs.SetRGBA(cx+x, -cy+y, color.RGBA{0xff, 0xff, 0xff, 0xff})
		cvs.SetRGBA(cy+x, -cx+y, color.RGBA{0xff, 0xff, 0xff, 0xff})
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
	stw.currentBrush = color.RGBA{uint8(col[0]), uint8(col[1]), uint8(col[2]), 0x77}
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

