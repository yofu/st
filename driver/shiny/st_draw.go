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
		e2 = eps << 1
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
	sa := int(src.A)
	da := (int(dst.A) * int(255 - src.A)) >> 8
	cvs.SetRGBA(x, y, color.RGBA{
		uint8((int(src.R) * sa + int(dst.R) * da) >> 8),
		uint8((int(src.G) * sa + int(dst.G) * da) >> 8),
		uint8((int(src.B) * sa + int(dst.B) * da) >> 8),
		uint8(sa + da),
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
	dx12 := x2 - x1
	dy12 := y2 - y1
	dx23 := x3 - x2
	dy23 := y3 - y2
	endy12 := y2
	dx13 := x3 - x1
	if dx13 < 0 {
		dx13 = -dx13
	}
	dy13 := y3 - y1
	if dy13 < 0 {
		dy13 = -dy13
	}
	var sx12 int
	if x1 < x2 {
		sx12 = 1
	} else {
		sx12 = -1
	}
	var sx13 int
	if x1 < x3 {
		sx13 = 1
	} else {
		sx13 = -1
	}
	var sx23 int
	if x2 < x3 {
		sx23 = 1
	} else {
		sx23 = -1
	}
	sy13 := 1
	eps13 := dx13 - dy13
	x13 := x1
	y13 := y1
	endx13 := x3
	endy13 := y3
	var e13 int
	cvs := stw.buffer.RGBA()
	var sx int
	x12 := x1
	x23 := x2
	for {
		Blend(cvs, x13, y13, stw.currentBrush)
		if x13 == endx13 && y13 == endy13 {
			break
		}
		e13 = eps13 << 1
		if e13 > -dy13 {
			eps13 = eps13 - dy13
			x13 = x13 + sx13
		}
		if e13 < dx13 {
			if y13 < endy12 {
				for {
					if ((x12 - x1) * dy12 - dx12 * (y13 - y1)) * sx12 >= 0 {
						break
					}
					x12 = x12 + sx12
				}
				x := x12
				if x < x13 {
					sx = 1
				} else {
					sx = -1
				}
				for {
					Blend(cvs, x, y13, stw.currentBrush)
					if x == x13 {
						break
					}
					x = x + sx
				}
			} else if dy23 > 0 {
				for {
					if ((x23 - x2) * dy23 - dx23 * (y13 - y2)) * sx23 >= 0 {
						break
					}
					x23 = x23 + sx23
				}
				x := x23
				if x < x13 {
					sx = 1
				} else {
					sx = -1
				}
				for {
					Blend(cvs, x, y13, stw.currentBrush)
					if x == x13 {
						break
					}
					x = x + sx
				}
			}
			eps13 = eps13 + dx13
			y13 = y13 + sy13
		}
	}
	return
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

func (stw *Window) SectionAlias(int) (string, bool) {
	return "", false
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

func (stw *Window) CanvasPaperSize() (float64, float64, error) {
	return 0.0, 0.0, nil
}

func (stw *Window) Flush() {
}

func (stw *Window) CanvasDirection() int {
	return 1
}

