package stshiny

import (
	"github.com/yofu/st/stlib"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
	"image"
	"image/color"
	"strings"
)

var (
	PLATE_OPACITY uint8 = 0x77
)

func line(cvs *image.RGBA, x1, y1, x2, y2 int, color color.RGBA) {
	dx := x2 - x1
	if dx < 0 {
		dx = -dx
	}
	dy := y2 - y1
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
	x := x1
	y := y1
	endx := x2
	endy := y2
	var e2 int
	for i := 0; ; i++ {
		cvs.SetRGBA(x, y, color)
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
	linedash, dashlen := stw.LineDashProperty()
	ind := 0
	for i := 0; ; i++ {
		if i >= dashlen {
			i = 0
		}
		if i == linedash[ind] {
			i = 0
			ind++
			if ind >= len(linedash) {
				ind = 0
			}
		}
		if ind&1 == 0 {
			cvs.SetRGBA(x, y, stw.currentPen)
		}
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

func (stw *Window) Polyline(coords [][]float64) {
	for i := 0; i < len(coords)-1; i++ {
		stw.Line(coords[i][0], coords[i][1], coords[i+1][0], coords[i+1][1])
	}
}

func Blend(cvs *image.RGBA, x, y int, src color.RGBA) {
	dst := cvs.RGBAAt(x, y)
	sa := int(src.A)
	da := (int(dst.A) * int(255-src.A)) >> 8
	cvs.SetRGBA(x, y, color.RGBA{
		uint8((int(src.R)*sa + int(dst.R)*da) >> 8),
		uint8((int(src.G)*sa + int(dst.G)*da) >> 8),
		uint8((int(src.B)*sa + int(dst.B)*da) >> 8),
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
					if ((x12-x1)*dy12-dx12*(y13-y1))*sx12 >= 0 {
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
					if ((x23-x2)*dy23-dx23*(y13-y2))*sx23 >= 0 {
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

func (stw *Window) fillquadrangle(c1, c2, c3, c4 []float64) {
	ind := 0
	maxy := 0.0
	coords := [][]float64{c1, c2, c3, c4}
	for i, c := range coords {
		if c[1] > maxy {
			maxy = c[1]
			ind = i
		}
	}
	i1 := ind - 2
	if i1 < 0 {
		i1 += 4
	}
	i2 := ind - 3
	if i2 < 0 {
		i2 += 4
	}
	i3 := ind - 1
	if i3 < 0 {
		i3 += 4
	}
	i4 := ind
	if i4 < 0 {
		i4 += 4
	}
	x1 := int(coords[i1][0])
	y1 := int(coords[i1][1])
	x2 := int(coords[i2][0])
	y2 := int(coords[i2][1])
	x3 := int(coords[i3][0])
	y3 := int(coords[i3][1])
	x4 := int(coords[i4][0])
	y4 := int(coords[i4][1])
	dx12 := x2 - x1
	dy12 := y2 - y1
	dx13 := x3 - x1
	dy13 := y3 - y1
	dx24 := x4 - x2
	dy24 := y4 - y2
	dx34 := x4 - x3
	dy34 := y4 - y3
	endy12 := y2
	endy13 := y3
	dx14 := x4 - x1
	if dx14 < 0 {
		dx14 = -dx14
	}
	dy14 := y4 - y1
	if dy14 < 0 {
		dy14 = -dy14
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
	var sx14 int
	if x1 < x4 {
		sx14 = 1
	} else {
		sx14 = -1
	}
	var sx24 int
	if x2 < x4 {
		sx24 = 1
	} else {
		sx24 = -1
	}
	var sx34 int
	if x3 < x4 {
		sx34 = 1
	} else {
		sx34 = -1
	}
	sy14 := 1
	eps14 := dx14 - dy14
	x14 := x1
	y14 := y1
	endx14 := x4
	endy14 := y4
	var e14 int
	cvs := stw.buffer.RGBA()
	x12 := x1
	x24 := x2
	x13 := x1
	x34 := x3
	// TODO: fix
	for {
		// Blend(cvs, x14, y14, stw.currentBrush)
		if x14 == endx14 && y14 == endy14 {
			break
		}
		e14 = eps14 << 1
		if e14 > -dy14 {
			eps14 = eps14 - dy14
			x14 = x14 + sx14
		}
		if e14 < dx14 {
			var sx, ex int
			if y14 < endy12 {
				for {
					if ((x12-x1)*dy12-dx12*(y14-y1))*sx12 >= 0 {
						break
					}
					x12 = x12 + sx12
				}
				sx = x12
			} else if dy24 > 0 {
				for {
					if ((x24-x2)*dy24-dx24*(y14-y2))*sx24 >= 0 {
						break
					}
					x24 = x24 + sx24
				}
				sx = x24
			}
			if y14 < endy13 {
				for {
					if ((x13-x1)*dy13-dx13*(y14-y1))*sx13 >= 0 {
						break
					}
					x13 = x13 + sx13
				}
				ex = x13
			} else if dy34 > 0 {
				for {
					if ((x34-x3)*dy34-dx34*(y14-y3))*sx34 >= 0 {
						break
					}
					x34 = x34 + sx34
				}
				ex = x34
			}
			if sx > ex {
				sx, ex = ex, sx
			}
			for x := sx; x <= ex; x++ {
				Blend(cvs, x, y14, stw.currentBrush)
			}
			eps14 = eps14 + dx14
			y14 = y14 + sy14
		}
	}
	if y1 > y2 {
		if dx12 < 0 {
			dx12 = -dx12
		}
		if dy12 < 0 {
			dy12 = -dy12
		}
		eps := dx12 - dy12
		x := x1
		y := y1
		x12 = x1
		endx := x2
		endy := y2
		var sy12 int
		if y1 < y2 {
			sy12 = 1
		} else {
			sy12 = -1
		}
		var e12 int
		for {
			if x == endx && y == endy {
				break
			}
			if y < y1 {
				cvs.SetRGBA(x, y, stw.currentPen)
			}
			e12 = eps << 1
			if e12 > -dy12 {
				eps = eps - dy12
				x = x + sx12
			}
			if e12 < dx12 {
				for {
					if ((x12-x2)*dy24-dx24*(y-y2))*sx12 >= 0 {
						break
					}
					x12 = x12 + sx12
				}
				cx := x12
				var sx int
				if x12 < x {
					sx = 1
				} else {
					sx = -1
				}
				for {
					if y < y1 {
						Blend(cvs, cx, y, stw.currentBrush)
					}
					if cx == x {
						break
					}
					cx = cx + sx
				}
				eps = eps + dx12
				y = y + sy12
			}
		}
	} else if y1 > y3 {
		if dx13 < 0 {
			dx13 = -dx13
		}
		if dy13 < 0 {
			dy13 = -dy13
		}
		eps := dx13 - dy13
		x := x1
		y := y1
		x13 = x1
		endx := x3
		endy := y3
		var sy13 int
		if y1 < y3 {
			sy13 = 1
		} else {
			sy13 = -1
		}
		var e13 int
		for {
			if x == endx && y == endy {
				break
			}
			if y < y1 {
				cvs.SetRGBA(x, y, stw.currentPen)
			}
			e13 = eps << 1
			if e13 > -dy13 {
				eps = eps - dy13
				x = x + sx13
			}
			if e13 < dx13 {
				for {
					if ((x13-x3)*dy34-dx34*(y-y3))*sx13 >= 0 {
						break
					}
					x13 = x13 + sx13
				}
				cx := x13
				var sx int
				if x13 < x {
					sx = 1
				} else {
					sx = -1
				}
				for {
					if y < y1 {
						Blend(cvs, cx, y, stw.currentBrush)
					}
					if cx == x {
						break
					}
					cx = cx + sx
				}
				eps = eps + dx13
				y = y + sy13
			}
		}
	}
	return
}

func (stw *Window) Polygon(coords [][]float64) {
	if len(coords) < 3 {
		return
	}
	switch len(coords) {
	case 3:
		stw.filltriangle(coords[0], coords[1], coords[2])
	case 4:
		stw.fillquadrangle(coords[0], coords[1], coords[2], coords[3])
	default:
		for i := 0; i < len(coords)-2; i++ {
			stw.filltriangle(coords[0], coords[i+1], coords[i+2])
		}
	}
}

func (stw *Window) Circle(x1, y1, d float64) {
	cx := 0
	cy := int(0.5*float64(d) + 1)
	dd := -int(d)*int(d) + 4*cy*cy - 4*cy + 2
	dx := 4
	dy := -8*cy + 8
	x := int(x1)
	y := int(y1)
	if (int(d) & 1) == 0 {
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

func (stw *Window) Text(x, y float64, str string) {
	if str == "" {
		return
	}
	x0 := fixed.I(int(x))
	d := &font.Drawer{
		Dst:  stw.buffer.RGBA(),
		Src:  image.NewUniform(stw.font.color),
		Face: stw.font.face,
		Dot:  fixed.Point26_6{x0, fixed.I(int(y))},
	}
	ss := strings.Split(str, "\n")
	d.Dot.Y -= fixed.Int26_6(int(float64(stw.font.height) * 1.2 * float64(len(ss)-1)))
	for _, s := range ss {
		d.DrawString(s)
		d.Dot.Y += fixed.Int26_6(int(float64(stw.font.height) * 1.2))
		d.Dot.X = x0
	}
}

func (stw *Window) Foreground(fg int) {
	col := st.IntColorList(fg)
	stw.currentPen = color.RGBA{uint8(col[0]), uint8(col[1]), uint8(col[2]), 0xff}
	stw.currentBrush = color.RGBA{uint8(col[0]), uint8(col[1]), uint8(col[2]), PLATE_OPACITY}
	stw.font.color = color.RGBA{uint8(col[0]), uint8(col[1]), uint8(col[2]), 0xff}
}

func (stw *Window) TextAlignment(int) {
}

func (stw *Window) TextOrientation(float64) {
}

func (stw *Window) DefaultStyle() {
	stw.currentPen = color.RGBA{0xff, 0xff, 0xff, 0xff}
	stw.currentBrush = color.RGBA{0xff, 0xff, 0xff, 0x77}
	stw.font.color = color.RGBA{0xff, 0xff, 0xff, 0xff}
	PLATE_OPACITY = 0x77
}

func (stw *Window) BondStyle(*st.Show) {
}

func (stw *Window) PhingeStyle(*st.Show) {
}

func (stw *Window) ConfStyle(*st.Show) {
}

func (stw *Window) SelectNodeStyle() {
	stw.font.color = color.RGBA{0xff, 0x00, 0x00, 0xff}
}

func (stw *Window) SelectElemStyle() {
	stw.LineStyle(st.DOTTED)
	PLATE_OPACITY = 0xcc
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
