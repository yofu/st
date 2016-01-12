package stgxui

import (
	"github.com/google/gxui"
	gxmath "github.com/google/gxui/math"
	"github.com/yofu/st/stlib"
	"math"
)

const (
	CIRCLE_DIV     = 8
	PLANE_OPACITY  = float32(0.2)
	SELECT_OPACITY = 0.5
)

var (
	LineWidth        = float32(1.0)
	PlateEdgePen     = gxui.CreatePen(0.5, gxui.Gray90)
	RubberPenLeft    = gxui.CreatePen(1.5, gxui.White)
	RubberBrushLeft  = gxui.CreateBrush(OpaqueColor(gxui.Blue, 0.3))
	RubberPenRight   = gxui.CreatePen(1.5, gxui.White)
	RubberBrushRight = gxui.CreateBrush(OpaqueColor(gxui.Green, 0.3))
	RubberPenNode    = gxui.CreatePen(1.5, gxui.White)
	RubberBrushNode  = gxui.CreateBrush(OpaqueColor(gxui.Red, 0.3))
	RubberPenSnap    = gxui.CreatePen(1.0, gxui.Yellow)
	DeformationPen   = gxui.CreatePen(0.5, gxui.Gray90)
	StressTextColor  = gxui.White
	YieldedTextColor = gxui.Yellow
	BrittleTextColor = gxui.Red
)

func IntColorFloat32(col int) []float32 {
	rtn := make([]float32, 3)
	val := 65536
	for i := 0; i < 3; i++ {
		tmp := 0
		for {
			if col >= val {
				col -= val
				tmp += 1
			} else {
				rtn[i] = float32(tmp) / 255.0
				break
			}
		}
		val >>= 8
	}
	return rtn
}

func OpaqueColor(c gxui.Color, opacity float32) gxui.Color {
	return gxui.Color{c.R, c.G, c.B, opacity}
}

func Pen(color int, selected bool) gxui.Pen {
	c := IntColorFloat32(color)
	a := float32(1.0)
	if selected {
		a = SELECT_OPACITY
	}
	col := gxui.Color{c[0], c[1], c[2], a}
	return gxui.CreatePen(LineWidth, col)
}

func Brush(color int, selected bool) gxui.Brush {
	c := IntColorFloat32(color)
	a := PLANE_OPACITY
	if selected {
		a = SELECT_OPACITY
	}
	col := gxui.Color{c[0], c[1], c[2], a}
	return gxui.CreateBrush(col)
}

func (stw *Window) Line(x1, y1, x2, y2 float64) {
	p := make(gxui.Polygon, 2)
	p[0] = gxui.PolygonVertex{
		Position: gxmath.Point{
			X: int(x1),
			Y: int(y1),
		},
	}
	p[1] = gxui.PolygonVertex{
		Position: gxmath.Point{
			X: int(x2),
			Y: int(y2),
		},
	}
	stw.currentCanvas.DrawLines(p, stw.currentPen)
}

func (stw *Window) Polyline(coords [][]float64) {
	p := make(gxui.Polygon, len(coords))
	for i, v := range coords {
		p[i] = gxui.PolygonVertex{
			Position: gxmath.Point{
				X: int(v[0]),
				Y: int(v[1]),
			},
		}
	}
	stw.currentCanvas.DrawLines(p, stw.currentPen)
}

func (stw *Window) Rect(left, right, bottom, top float64) {
	vertices := make([][]float64, 4)
	vertices[0] = []float64{left, bottom}
	vertices[1] = []float64{left, top}
	vertices[2] = []float64{right, top}
	vertices[3] = []float64{right, bottom}
	stw.Polygon(vertices)
}

func (stw *Window) Polygon(coords [][]float64) {
	l := len(coords)
	p := make(gxui.Polygon, l)
	_, cw := st.ClockWise(coords[0], coords[1], coords[2])
	if cw {
		for i, v := range coords {
			p[l-1-i] = gxui.PolygonVertex{
				Position: gxmath.Point{
					X: int(v[0]),
					Y: int(v[1]),
				},
			}
		}
	} else {
		for i, v := range coords {
			p[i] = gxui.PolygonVertex{
				Position: gxmath.Point{
					X: int(v[0]),
					Y: int(v[1]),
				},
			}
		}
	}
	stw.currentCanvas.DrawPolygon(p, stw.currentPen, stw.currentBrush)
}

func (stw *Window) Circle(x, y, r float64) {
	p := make(gxui.Polygon, CIRCLE_DIV+1)
	theta := 0.0
	dtheta := 2.0 * math.Pi / float64(CIRCLE_DIV)
	for i := 0; i < CIRCLE_DIV; i++ {
		p[i] = gxui.PolygonVertex{
			Position: gxmath.Point{
				X: int(x + r*math.Cos(theta)),
				Y: int(y + r*math.Sin(theta)),
			},
		}
		theta += dtheta
	}
	p[CIRCLE_DIV] = p[0]
	stw.currentCanvas.DrawLines(p, stw.currentPen)
}

func (stw *Window) FilledCircle(x, y, r float64) {
	p := make(gxui.Polygon, CIRCLE_DIV+1)
	theta := 0.0
	dtheta := 2.0 * math.Pi / float64(CIRCLE_DIV)
	for i := 0; i < CIRCLE_DIV; i++ {
		p[i] = gxui.PolygonVertex{
			Position: gxmath.Point{
				X: int(x + r*math.Cos(theta)),
				Y: int(y + r*math.Sin(theta)),
			},
		}
		theta += dtheta
	}
	p[CIRCLE_DIV] = p[0]
	stw.currentCanvas.DrawPolygon(p, stw.currentPen, gxui.CreateBrush(gxui.White))
}

func (stw *Window) Text(x, y float64, str string) {
	font := stw.currentFont
	runes := []rune(str)
	r := gxmath.Rect{
		Min: gxmath.Point{X: int(x), Y: int(y)},
		Max: gxmath.Point{X: int(x), Y: int(y)},
	}
	offsets := font.Layout(&gxui.TextBlock{
		Runes:     runes,
		AlignRect: r,
		H:         stw.currentHorizontalAlignment,
		V:         stw.currentVerticalAlignment,
	})
	l := len(runes)
	rs := make([]rune, l)
	os := make([]gxmath.Point, l)
	pos := 0
	for i, r := range runes {
		if r == '\n' {
			continue
		}
		rs[pos] = r
		os[pos] = offsets[i]
		pos++
	}
	rs = rs[:pos]
	os = os[:pos]
	stw.currentCanvas.DrawRunes(font, rs, os, stw.currentFontColor)
}

func (stw *Window) Foreground(fg int) {
	stw.currentPen = Pen(fg, false)
	stw.currentBrush = Brush(fg, false)
}

func (stw *Window) LineStyle(ls int) {
	// TODO
}

func (stw *Window) TextAlignment(ta int) {
	switch ta {
	case st.SOUTH:
		stw.currentHorizontalAlignment = gxui.AlignLeft
		stw.currentVerticalAlignment = gxui.AlignBottom
	case st.NORTH:
		stw.currentHorizontalAlignment = gxui.AlignLeft
		stw.currentVerticalAlignment = gxui.AlignBottom
	case st.WEST:
		stw.currentHorizontalAlignment = gxui.AlignLeft
		stw.currentVerticalAlignment = gxui.AlignBottom
	case st.EAST:
		stw.currentHorizontalAlignment = gxui.AlignLeft
		stw.currentVerticalAlignment = gxui.AlignBottom
	}
}

func (stw *Window) TextOrientation(to float64) {
	// TODO
}

func (stw *Window) ShowPrintRange() bool {
	return showprintrange
}

func (stw *Window) CanvasPaperSize() (float64, float64, error) {
	return 0.0, 0.0, nil
}

func (stw *Window) Flush() {
	stw.currentCanvas.Complete()
	stw.draw.SetCanvas(stw.currentCanvas)
}

func (stw *Window) CanvasDirection() int {
	return 1
}
