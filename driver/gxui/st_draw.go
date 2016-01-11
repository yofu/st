package stgxui

import (
	"bytes"
	"fmt"
	"github.com/google/gxui"
	gxmath "github.com/google/gxui/math"
	"github.com/yofu/st/stlib"
	"github.com/yofu/st/arclm"
	"math"
	"sort"
	"sync"
)

const (
	CIRCLE_DIV = 8
	PLANE_OPACITY = float32(0.2)
	SELECT_OPACITY = 0.5
)

var (
	LineWidth = float32(1.0)
	PlateEdgePen = gxui.CreatePen(0.5, gxui.Gray90)
	RubberPenLeft = gxui.CreatePen(1.5, gxui.White)
	RubberBrushLeft = gxui.CreateBrush(OpaqueColor(gxui.Blue, 0.3))
	RubberPenRight = gxui.CreatePen(1.5, gxui.White)
	RubberBrushRight = gxui.CreateBrush(OpaqueColor(gxui.Green, 0.3))
	RubberPenNode = gxui.CreatePen(1.5, gxui.White)
	RubberBrushNode = gxui.CreateBrush(OpaqueColor(gxui.Red, 0.3))
	RubberPenSnap = gxui.CreatePen(1.0, gxui.Yellow)
	DeformationPen = gxui.CreatePen(0.5, gxui.Gray90)
	StressTextColor = gxui.White
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

func Line(canvas gxui.Canvas, pen gxui.Pen, x1, y1, x2, y2 int) {
	p := make(gxui.Polygon, 2)
	p[0] = gxui.PolygonVertex{
		Position: gxmath.Point{
			X: x1,
			Y: y1,
		},
	}
	p[1] = gxui.PolygonVertex{
		Position: gxmath.Point{
			X: x2,
			Y: y2,
		},
	}
	canvas.DrawLines(p, pen)
}

func PolyLine(canvas gxui.Canvas, pen gxui.Pen, vertices [][]int) {
	p := make(gxui.Polygon, len(vertices))
	for i, v := range vertices {
		p[i] = gxui.PolygonVertex{
			Position: gxmath.Point{
				X: v[0],
				Y: v[1],
			},
		}
	}
	canvas.DrawLines(p, pen)
}

func Polygon(canvas gxui.Canvas, pen gxui.Pen, brush gxui.Brush, vertices [][]int) {
	l := len(vertices)
	p := make(gxui.Polygon, l)
	_, cw := st.ClockWiseInt(vertices[0], vertices[1], vertices[2])
	if cw {
		for i, v := range vertices {
			p[l-1-i] = gxui.PolygonVertex{
				Position: gxmath.Point{
					X: v[0],
					Y: v[1],
				},
			}
		}
	} else {
		for i, v := range vertices {
			p[i] = gxui.PolygonVertex{
				Position: gxmath.Point{
					X: v[0],
					Y: v[1],
				},
			}
		}
	}
	canvas.DrawPolygon(p, pen, brush)
}

func Rect(canvas gxui.Canvas, pen gxui.Pen, brush gxui.Brush, left, right, bottom, top int) {
	vertices := make([][]int, 4)
	vertices[0] = []int{left, bottom}
	vertices[1] = []int{left, top}
	vertices[2] = []int{right, top}
	vertices[3] = []int{right, bottom}
	Polygon(canvas, pen, brush, vertices)
}

func Arrow(cvs gxui.Canvas, pen gxui.Pen, x1, y1, x2, y2 int, size, theta float64) {
	c := size * math.Cos(theta)
	s := size * math.Sin(theta)
	Line(cvs, pen, x1, y1, x2, y2)
	Line(cvs, pen, x2, y2, x2+int(float64(x1-x2)*c-float64(y1-y2)*s), y2+int(float64(x1-x2)*s+float64(y1-y2)*c))
	Line(cvs, pen, x2, y2, x2+int(float64(x1-x2)*c+float64(y1-y2)*s), y2+int(float64(-(x1-x2))*s+float64(y1-y2)*c))
}

func Circle(canvas gxui.Canvas, pen gxui.Pen, x, y, r int) {
	p := make(gxui.Polygon, CIRCLE_DIV+1)
	theta := 0.0
	dtheta := 2.0 * math.Pi / float64(CIRCLE_DIV)
	for i:=0; i<CIRCLE_DIV; i++ {
		p[i] = gxui.PolygonVertex{
			Position: gxmath.Point{
				X: x + int(float64(r)*math.Cos(theta)),
				Y: y + int(float64(r)*math.Sin(theta)),
			},
		}
		theta += dtheta
	}
	p[CIRCLE_DIV] = p[0]
	canvas.DrawLines(p, pen)
}

func FilledCircle(canvas gxui.Canvas, pen gxui.Pen, x, y, r int) {
	p := make(gxui.Polygon, CIRCLE_DIV)
	theta := 0.0
	dtheta := 2.0 * math.Pi / float64(CIRCLE_DIV)
	for i:=0; i<CIRCLE_DIV; i++ {
		p[i] = gxui.PolygonVertex{
			Position: gxmath.Point{
				X: x + int(float64(r)*math.Cos(theta)),
				Y: y + int(float64(r)*math.Sin(theta)),
			},
		}
		theta += dtheta
	}
	canvas.DrawPolygon(p, pen, gxui.CreateBrush(gxui.White))
}

func Text(canvas gxui.Canvas, font gxui.Font, color gxui.Color, x, y int, str string) {
	runes := []rune(str)
	r := gxmath.Rect{
		Min: gxmath.Point{X: x, Y: y},
		Max: gxmath.Point{X: x, Y: y},
	}
	offsets := font.Layout(&gxui.TextBlock{
		Runes:     runes,
		AlignRect: r,
		H:         gxui.AlignLeft,
		V:         gxui.AlignBottom,
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
	canvas.DrawRunes(font, rs, os, color)
}

func (stw *Window) DrawFrameNode() gxui.Canvas {
	canvas := stw.driver.CreateCanvas(gxmath.Size{W: stw.CanvasSize[0], H: stw.CanvasSize[1]})
	if stw.Frame == nil {
		canvas.Complete()
		return canvas
	}
	pen := gxui.CreatePen(1, gxui.White)
	var brush gxui.Brush
	font := stw.theme.DefaultFont()
	stw.Frame.View.Set(1)
	nodes := make([]*st.Node, len(stw.Frame.Nodes))
	i := 0
	for _, n := range stw.Frame.Nodes {
		nodes[i] = n
		i++
	}
	sort.Sort(st.NodeByNum{nodes})
	for _, n := range nodes {
		stw.Frame.View.ProjectNode(n)
		if stw.Frame.Show.Deformation {
			stw.Frame.View.ProjectDeformation(n, stw.Frame.Show)
		}
		if n.IsHidden(stw.Frame.Show) {
			continue
		}
		txtcolor := gxui.Green90
		for _, j := range stw.selectNode {
			if j == n {
				DrawNode(n, canvas, pen, font, txtcolor, stw.Frame.Show)
				break
			}
		}
	}
	if !stw.Frame.Show.Select {
		var wg sync.WaitGroup
		var m sync.Mutex
		for _, elem := range stw.Frame.Elems {
			wg.Add(1)
			go func(el *st.Elem) {
				defer wg.Done()
				if el.Etype >= st.WBRACE || el.Etype < st.COLUMN {
					return
				}
				for _, j := range stw.selectElem {
					if j == el {
						return
					}
				}
				m.Lock()
				if el.IsHidden(stw.Frame.Show) {
					pen = gxui.CreatePen(1, gxui.Gray90)
				} else {
					pen = gxui.CreatePen(1, gxui.Green30)
				}
				DrawElemLine(el, canvas, pen)
				m.Unlock()
			}(elem)
		}
		wg.Wait()
	}
	if stw.selectElem != nil {
		nomv := stw.Frame.Show.NoMomentValue
		stw.Frame.Show.NoMomentValue = false
		pen = gxui.CreatePen(1, gxui.Green90)
		brush = gxui.CreateBrush(OpaqueColor(gxui.Green90, SELECT_OPACITY))
		for _, el := range stw.selectElem {
			if el == nil || el.IsHidden(stw.Frame.Show) {
				continue
			}
			DrawElem(el, canvas, pen, brush, font, gxui.White, true, stw.Frame.Show)
		}
		stw.Frame.Show.NoMomentValue = nomv
	}
	if stw.rubber != nil && stw.rubber.IsComplete() {
		canvas.DrawCanvas(stw.rubber, gxmath.Point{X: 0, Y: 0})
	}
	canvas.Complete()
	return canvas
}

func (stw *Window) DrawFrame() gxui.Canvas {
	canvas := stw.driver.CreateCanvas(gxmath.Size{W: stw.CanvasSize[0], H: stw.CanvasSize[1]})
	if stw.Frame == nil {
		canvas.Complete()
		return canvas
	}
	pen := gxui.CreatePen(1, gxui.White)
	var brush gxui.Brush
	font := stw.theme.DefaultFont()
	stw.Frame.View.Set(1)
	nodes := make([]*st.Node, len(stw.Frame.Nodes))
	i := 0
	for _, n := range stw.Frame.Nodes {
		nodes[i] = n
		i++
	}
	sort.Sort(st.NodeByNum{nodes})
	for _, n := range nodes {
		stw.Frame.View.ProjectNode(n)
		if stw.Frame.Show.Deformation {
			stw.Frame.View.ProjectDeformation(n, stw.Frame.Show)
		}
		if n.IsHidden(stw.Frame.Show) {
			continue
		}
		txtcolor := gxui.White
		for _, j := range stw.selectNode {
			if j == n {
				txtcolor = gxui.Red
				break
			}
		}
		DrawNode(n, canvas, pen, font, txtcolor, stw.Frame.Show)
	}
	if !stw.Frame.Show.Select {
		els := st.SortedElem(stw.Frame.Elems, func(e *st.Elem) float64 { return -e.DistFromProjection(stw.Frame.View) })
		loop:
		for _, el := range els {
			if el.IsHidden(stw.Frame.Show) {
				continue
			}
			for _, j := range stw.selectElem {
				if j == el {
					continue loop
				}
			}
			switch stw.Frame.Show.ColorMode {
			default:
				pen = gxui.WhitePen
				brush = gxui.WhiteBrush
			case st.ECOLOR_WHITE:
				pen = gxui.WhitePen
				brush = gxui.WhiteBrush
			case st.ECOLOR_BLACK:
				pen = gxui.DefaultPen
				brush = gxui.DefaultBrush
			case st.ECOLOR_SECT:
				pen = Pen(el.Sect.Color, false)
				brush = Brush(el.Sect.Color, false)
			case st.ECOLOR_RATE:
				val, err := el.RateMax(stw.Frame.Show)
				if err != nil {
					pen = Pen(st.GREY_500, false)
					brush = Brush(st.GREY_500, false)
				} else {
					pen = Pen(st.Rainbow(val, st.RateBoundary), false)
					brush = Brush(st.Rainbow(val, st.RateBoundary), false)
				}
			case st.ECOLOR_N:
				if el.N(stw.Frame.Show.Period, 0) >= 0.0 {
					pen = Pen(st.RainbowColor[0], false) // Compression: Blue
				} else {
					pen = Pen(st.RainbowColor[6], false) // Tension: Red
				}
			case st.ECOLOR_STRONG:
				Ix, err := el.Sect.Ix(0)
				if err != nil {
					pen = gxui.WhitePen
				}
				Iy, err := el.Sect.Iy(0)
				if err != nil {
					pen = gxui.WhitePen
				}
				if Ix > Iy {
					pen = Pen(st.RainbowColor[0], false) // Strong: Blue
				} else if Ix == Iy {
					pen = Pen(st.RainbowColor[4], false) // Same: Yellow
				} else {
					pen = Pen(st.RainbowColor[6], false) // Weak: Red
				}
			}
			DrawElem(el, canvas, pen, brush, font, gxui.White, false, stw.Frame.Show)
		}
	}
	if stw.selectElem != nil {
		nomv := stw.Frame.Show.NoMomentValue
		stw.Frame.Show.NoMomentValue = false
		for _, el := range stw.selectElem {
			if el == nil || el.IsHidden(stw.Frame.Show) {
				continue
			}
			switch stw.Frame.Show.ColorMode {
			default:
				pen = gxui.WhitePen
				brush = gxui.WhiteBrush
			case st.ECOLOR_WHITE:
				pen = gxui.WhitePen
				brush = gxui.WhiteBrush
			case st.ECOLOR_BLACK:
				pen = gxui.DefaultPen
				brush = gxui.DefaultBrush
			case st.ECOLOR_SECT:
				pen = Pen(el.Sect.Color, true)
				brush = Brush(el.Sect.Color, true)
			case st.ECOLOR_RATE:
				val, err := el.RateMax(stw.Frame.Show)
				if err != nil {
					pen = Pen(st.GREY_500, true)
					brush = Brush(st.GREY_500, true)
				} else {
					pen = Pen(st.Rainbow(val, st.RateBoundary), true)
					brush = Brush(st.Rainbow(val, st.RateBoundary), true)
				}
			case st.ECOLOR_N:
				if el.N(stw.Frame.Show.Period, 0) >= 0.0 {
					pen = Pen(st.RainbowColor[0], true) // Compression: Blue
				} else {
					pen = Pen(st.RainbowColor[6], true) // Tension: Red
				}
			case st.ECOLOR_STRONG:
				Ix, err := el.Sect.Ix(0)
				if err != nil {
					pen = gxui.WhitePen
				}
				Iy, err := el.Sect.Iy(0)
				if err != nil {
					pen = gxui.WhitePen
				}
				if Ix > Iy {
					pen = Pen(st.RainbowColor[0], true) // Strong: Blue
				} else if Ix == Iy {
					pen = Pen(st.RainbowColor[4], true) // Same: Yellow
				} else {
					pen = Pen(st.RainbowColor[6], true) // Weak: Red
				}
			}
			DrawElem(el, canvas, pen, brush, font, gxui.White, true, stw.Frame.Show)
		}
		stw.Frame.Show.NoMomentValue = nomv
	}
	canvas.Complete()
	return canvas
}

func DrawNode(node *st.Node, cvs gxui.Canvas, pen gxui.Pen, font gxui.Font, txtcolor gxui.Color, show *st.Show) {
	// Caption
	var ncap bytes.Buffer
	var oncap bool
	if show.NodeCaption&st.NC_NUM != 0 {
		ncap.WriteString(fmt.Sprintf("%d\n", node.Num))
		oncap = true
	}
	if show.NodeCaption&st.NC_WEIGHT != 0 {
		if !node.Conf[2] || show.NodeCaption&st.NC_RZ == 0 {
			ncap.WriteString(fmt.Sprintf("%.3f\n", node.Weight[1] * show.Unit[0]))
			oncap = true
		}
	}
	for i, j := range []uint{st.NC_DX, st.NC_DY, st.NC_DZ, st.NC_TX, st.NC_TY, st.NC_TZ} {
		if show.NodeCaption&j != 0 {
			if !node.Conf[i] {
				if i < 3 {
					ncap.WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["DISP"]), node.ReturnDisp(show.Period, i)*100.0))
				} else {
					ncap.WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["THETA"]), node.ReturnDisp(show.Period, i)))
				}
				oncap = true
			}
		}
	}
	for i, j := range []uint{st.NC_RX, st.NC_RY, st.NC_RZ, st.NC_MX, st.NC_MY, st.NC_MZ} {
		if show.NodeCaption&j != 0 {
			if node.Conf[i] {
				var val float64
				if i == 2 && show.NodeCaption&st.NC_WEIGHT != 0 {
					val = node.ReturnReaction(show.Period, i) + node.Weight[1]
				} else {
					val = node.ReturnReaction(show.Period, i)
				}
				switch i {
				case 0, 1, 2:
					val *= show.Unit[0]
					ncap.WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["REACTION"]), val))
					arrow := 0.3
					rcoord := []float64{node.Coord[0], node.Coord[1], node.Coord[2]}
					if val >= 0.0 {
						rcoord[i] -= show.Rfact * val
						prcoord := node.Frame.View.ProjectCoord(rcoord)
						Arrow(cvs, pen, int(prcoord[0]), int(prcoord[1]), int(node.Pcoord[0]), int(node.Pcoord[1]), arrow, deg10)
					} else {
						rcoord[i] += show.Rfact * val
						prcoord := node.Frame.View.ProjectCoord(rcoord)
						Arrow(cvs, pen, int(node.Pcoord[0]), int(node.Pcoord[1]), int(prcoord[0]), int(prcoord[1]), arrow, deg10)
					}
				case 3, 4, 5:
					val *= show.Unit[0] * show.Unit[1]
					ncap.WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["REACTION"]), val))
				}
				oncap = true
			}
		}
	}
	if show.NodeCaption&st.NC_ZCOORD != 0 {
		ncap.WriteString(fmt.Sprintf("%.1f\n", node.Coord[2]))
		oncap = true
	}
	if show.NodeCaption&st.NC_PILE != 0 {
		if node.Pile != nil {
			ncap.WriteString(fmt.Sprintf("%d\n", node.Pile.Num))
			oncap = true
		}
	}
	if oncap {
		Text(cvs, font, txtcolor, int(node.Pcoord[0]), int(node.Pcoord[1]), ncap.String())
	}
	if show.NodeNormal {
		// DrawNodeNormal(node, cvs, show)
	}
	// Conffigure
	if show.Conf {
		switch node.ConfState() {
		default:
			return
		case st.CONF_PIN:
			PinFigure(cvs, node.Pcoord[0], node.Pcoord[1], show.ConfSize)
		case st.CONF_XROL, st.CONF_YROL, st.CONF_XYROL:
			RollerFigure(cvs, node.Pcoord[0], node.Pcoord[1], show.ConfSize, 0)
		case st.CONF_ZROL:
			RollerFigure(cvs, node.Pcoord[0], node.Pcoord[1], show.ConfSize, 1)
		case st.CONF_FIX:
			FixFigure(cvs, node.Pcoord[0], node.Pcoord[1], show.ConfSize)
		}
	}
}

func PinFigure(cvs gxui.Canvas, x, y, size float64) {
	val := y + 0.5*math.Sqrt(3)*size
	vers := make([][]int, 3)
	vers[0] = []int{int(x), int(y)}
	vers[1] = []int{int(x+0.5*size), int(val)}
	vers[2] = []int{int(x-0.5*size), int(val)}
	Polygon(cvs, gxui.WhitePen, gxui.WhiteBrush, vers)
}

func RollerFigure(cvs gxui.Canvas, x, y, size float64, direction int) {
	switch direction {
	case 0:
		val1 := y + 0.5*math.Sqrt(3)*size
		val2 := y + 0.75*math.Sqrt(3)*size
		vers := make([][]int, 3)
		vers[0] = []int{int(x), int(y)}
		vers[1] = []int{int(x+0.5*size), int(val1)}
		vers[2] = []int{int(x-0.5*size), int(val1)}
		Polygon(cvs, gxui.WhitePen, gxui.WhiteBrush, vers)
		Line(cvs, gxui.WhitePen, int(x-0.5*size), int(val2), int(x+0.5*size), int(val2))
	case 1:
		val1 := x - 0.5*math.Sqrt(3)*size
		val2 := x - 0.75*math.Sqrt(3)*size
		vers := make([][]int, 3)
		vers[0] = []int{int(x), int(y)}
		vers[1] = []int{int(val1), int(y+0.5*size)}
		vers[2] = []int{int(val1), int(y-0.5*size)}
		Polygon(cvs, gxui.WhitePen, gxui.WhiteBrush, vers)
		Line(cvs, gxui.WhitePen, int(val2), int(y-0.5*size), int(val2), int(y+0.5*size))
	}
}

func FixFigure(cvs gxui.Canvas, x, y, size float64) {
	Line(cvs, gxui.WhitePen, int(x-size), int(y), int(x+size), int(y))
	Line(cvs, gxui.WhitePen, int(x-0.25*size), int(y), int(x-0.75*size), int(y+0.5*size))
	Line(cvs, gxui.WhitePen, int(x+0.25*size), int(y), int(x-0.25*size), int(y+0.5*size))
	Line(cvs, gxui.WhitePen, int(x+0.75*size), int(y), int(x+0.25*size), int(y+0.5*size))
}

func DrawElem(elem *st.Elem, cvs gxui.Canvas, pen gxui.Pen, brush gxui.Brush, font gxui.Font, txtcolor gxui.Color, selected bool, show *st.Show) {
	var ecap bytes.Buffer
	var oncap bool
	if show.ElemCaption&st.EC_NUM != 0 {
		ecap.WriteString(fmt.Sprintf("%d\n", elem.Num))
		oncap = true
	}
	if show.ElemCaption&st.EC_SECT != 0 {
		// if al, ok := sectionaliases[elem.Sect.Num]; ok {
		// 	ecap.WriteString(fmt.Sprintf("%s\n", al))
		// } else {
		ecap.WriteString(fmt.Sprintf("%d\n", elem.Sect.Num))
		// }
		oncap = true
	}
	if show.ElemCaption&st.EC_WIDTH != 0 {
		ecap.WriteString(fmt.Sprintf("%.3f\n", elem.Width()))
		oncap = true
	}
	if show.ElemCaption&st.EC_HEIGHT != 0 {
		ecap.WriteString(fmt.Sprintf("%.3f\n", elem.Height()))
		oncap = true
	}
	if show.ElemCaption&st.EC_RATE_L != 0 || show.ElemCaption&st.EC_RATE_S != 0 {
		val, err := elem.RateMax(show)
		if err == nil {
			ecap.WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["RATE"]), val))
			oncap = true
		}
	}
	if show.ElemCaption&st.EC_PREST != 0 {
		if elem.Prestress != 0.0 {
			ecap.WriteString(fmt.Sprintf("%.3f\n", elem.Prestress * show.Unit[0]))
			oncap = true
		}
	}
	if show.ElemCaption&st.EC_STIFF_X != 0 {
		stiff := elem.LateralStiffness("X", false) * show.Unit[0] / show.Unit[1]
		if stiff != 0.0 {
			if stiff == 1e16 {
				ecap.WriteString("∞")
			} else {
				ecap.WriteString(fmt.Sprintf("%.3f\n", stiff))
			}
			oncap = true
		}
	}
	if show.ElemCaption&st.EC_STIFF_Y != 0 {
		stiff := elem.LateralStiffness("Y", false) * show.Unit[0] / show.Unit[1]
		if stiff != 0.0 {
			if stiff == 1e16 {
				ecap.WriteString("∞")
			} else {
				ecap.WriteString(fmt.Sprintf("%.3f\n", stiff))
			}
			oncap = true
		}
	}
	if oncap {
		var textpos []float64
		if st.BRACE <= elem.Etype && elem.Etype <= st.SBRACE {
			coord := make([]float64, 3)
			for j, en := range elem.Enod {
				for k := 0; k < 3; k++ {
					coord[k] += (-0.5*float64(j) + 0.75) * en.Coord[k]
				}
			}
			textpos = elem.Frame.View.ProjectCoord(coord)
		} else {
			textpos = make([]float64, 2)
			for _, en := range elem.Enod {
				for i := 0; i < 2; i++ {
					textpos[i] += en.Pcoord[i]
				}
			}
			for i := 0; i < 2; i++ {
				textpos[i] /= float64(elem.Enods)
			}
		}
		Text(cvs, font, txtcolor, int(textpos[0]), int(textpos[1]), ecap.String())
	}
	if elem.IsLineElem() {
		Line(cvs, pen, int(elem.Enod[0].Pcoord[0]), int(elem.Enod[0].Pcoord[1]), int(elem.Enod[1].Pcoord[0]), int(elem.Enod[1].Pcoord[1]))
		pd := elem.PDirection(true)
		if show.Bond {
			switch elem.BondState() {
			case st.PIN_RIGID:
				Circle(cvs, pen, int(elem.Enod[0].Pcoord[0]+pd[0]*show.BondSize), int(elem.Enod[0].Pcoord[1]+pd[1]*show.BondSize), int(show.BondSize))
			case st.RIGID_PIN:
				Circle(cvs, pen, int(elem.Enod[1].Pcoord[0]-pd[0]*show.BondSize), int(elem.Enod[1].Pcoord[1]-pd[1]*show.BondSize), int(show.BondSize))
			case st.PIN_PIN:
				Circle(cvs, pen, int(elem.Enod[0].Pcoord[0]+pd[0]*show.BondSize), int(elem.Enod[0].Pcoord[1]+pd[1]*show.BondSize), int(show.BondSize))
				Circle(cvs, pen, int(elem.Enod[1].Pcoord[0]-pd[0]*show.BondSize), int(elem.Enod[1].Pcoord[1]-pd[1]*show.BondSize), int(show.BondSize))
			}
		}
		if show.Phinge {
			ph1 := elem.Phinge[show.Period][elem.Enod[0].Num]
			ph2 := elem.Phinge[show.Period][elem.Enod[1].Num]
			switch {
			case ph1 && !ph2:
				FilledCircle(cvs, pen, int(elem.Enod[0].Pcoord[0]+pd[0]*show.BondSize), int(elem.Enod[0].Pcoord[1]+pd[1]*show.BondSize), int(show.BondSize))
			case !ph1 && ph2:
				FilledCircle(cvs, pen, int(elem.Enod[1].Pcoord[0]-pd[0]*show.BondSize), int(elem.Enod[1].Pcoord[1]-pd[1]*show.BondSize), int(show.BondSize))
			case ph1 && ph2:
				FilledCircle(cvs, pen, int(elem.Enod[0].Pcoord[0]+pd[0]*show.BondSize), int(elem.Enod[0].Pcoord[1]+pd[1]*show.BondSize), int(show.BondSize))
				FilledCircle(cvs, pen, int(elem.Enod[1].Pcoord[0]-pd[0]*show.BondSize), int(elem.Enod[1].Pcoord[1]-pd[1]*show.BondSize), int(show.BondSize))
			}
		}
		if show.ElementAxis {
			// DrawElementAxis(elem, cvs, show)
		}
		// Deformation
		if show.Deformation {
			Line(cvs, DeformationPen, int(elem.Enod[0].Dcoord[0]), int(elem.Enod[0].Dcoord[1]), int(elem.Enod[1].Dcoord[0]), int(elem.Enod[1].Dcoord[1]))
		}
		// Stress
		var flag uint
		if f, ok := show.Stress[elem.Sect.Num]; ok {
			flag = f
		} else if f, ok := show.Stress[elem.Etype]; ok {
			flag = f
		}
		if flag != 0 {
			sttext := make([]bytes.Buffer, 2)
			for i, st := range []uint{st.STRESS_NZ, st.STRESS_QX, st.STRESS_QY, st.STRESS_MZ, st.STRESS_MX, st.STRESS_MY} {
				if flag&st != 0 {
					switch i {
					case 0:
						sttext[0].WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["STRESS"]), elem.ReturnStress(show.Period, 0, i) * show.Unit[0]))
					case 1, 2:
						sttext[0].WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["STRESS"]), elem.ReturnStress(show.Period, 0, i) * show.Unit[0]))
						sttext[1].WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["STRESS"]), elem.ReturnStress(show.Period, 1, i) * show.Unit[0]))
					case 3:
						sttext[0].WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["STRESS"]), elem.ReturnStress(show.Period, 0, i) * show.Unit[0] * show.Unit[1]))
						sttext[1].WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["STRESS"]), elem.ReturnStress(show.Period, 1, i) * show.Unit[0] * show.Unit[1]))
					case 4, 5:
						if !show.NoMomentValue {
							sttext[0].WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["STRESS"]), elem.ReturnStress(show.Period, 0, i) * show.Unit[0] * show.Unit[1]))
							sttext[1].WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["STRESS"]), elem.ReturnStress(show.Period, 1, i) * show.Unit[0] * show.Unit[1]))
						}
						mcoord := elem.MomentCoord(show, i)
						// cvs.Foreground(MomentColor)
						// cvs.Begin(cd.CD_OPEN_LINES)
						l := len(mcoord) + 2
						vers := make([][]int, l)
						vers[0] = []int{int(elem.Enod[0].Pcoord[0]), int(elem.Enod[0].Pcoord[1])}
						for i, c := range mcoord {
							tmp := elem.Frame.View.ProjectCoord(c)
							vers[i+1] = []int{int(tmp[0]), int(tmp[1])}
						}
						vers[l-1] = []int{int(elem.Enod[1].Pcoord[0]), int(elem.Enod[1].Pcoord[1])}
						PolyLine(cvs, pen, vers)
					}
				}
			}
			for j := 0; j < 2; j++ {
				if tex := sttext[j].String(); tex != "" {
					coord := make([]float64, 3)
					for i, en := range elem.Enod {
						for k := 0; k < 3; k++ {
							coord[k] += (-0.5*math.Abs(float64(i-j)) + 0.75) * en.Coord[k]
						}
					}
					stpos := elem.Frame.View.ProjectCoord(coord)
					// if j == 0 {
					// 	cvs.TextAlignment(cd.CD_SOUTH)
					// } else {
					// 	cvs.TextAlignment(cd.CD_NORTH)
					// }
					deg := math.Atan2(pd[1], pd[0]) * 180.0 / math.Pi
					if deg > 90.0 {
						deg -= 180.0
					} else if deg < -90.0 {
						deg += 180.0
					}
					// cvs.TextOrientation(deg)
					Text(cvs, font, StressTextColor, int(stpos[0]), int(stpos[1]), tex[:len(tex)-1])
					// cvs.TextAlignment(DefaultTextAlignment)
					// cvs.TextOrientation(0.0)
				}
			}
		}
		if show.YieldFunction {
			f, err := elem.YieldFunction(show.Period)
			for j := 0; j < 2; j++ {
				var ycol gxui.Color
				switch err[j].(type) {
				default:
					ycol = StressTextColor
				case arclm.YieldedError:
					ycol = YieldedTextColor
				case arclm.BrittleFailureError:
					ycol = BrittleTextColor
				}
				coord := make([]float64, 3)
				for i, en := range elem.Enod {
					for k := 0; k < 3; k++ {
						coord[k] += (-0.5*math.Abs(float64(i-j)) + 0.75) * en.Coord[k]
					}
				}
				stpos := elem.Frame.View.ProjectCoord(coord)
				// if j == 0 {
				// 	cvs.TextAlignment(cd.CD_SOUTH)
				// } else {
				// 	cvs.TextAlignment(cd.CD_NORTH)
				// }
				deg := math.Atan2(pd[1], pd[0]) * 180.0 / math.Pi
				if deg > 90.0 {
					deg -= 180.0
				} else if deg < -90.0 {
					deg += 180.0
				}
				// cvs.TextOrientation(deg)
				Text(cvs, font, ycol, int(stpos[0]), int(stpos[1]), fmt.Sprintf("%.3f", f[j]))
				// cvs.TextAlignment(DefaultTextAlignment)
				// cvs.TextOrientation(0.0)
			}
		}
		if elem.Etype == st.WBRACE || elem.Etype == st.SBRACE {
			if elem.Eldest {
				if elem.Parent.Wrect != nil && (elem.Parent.Wrect[0] != 0.0 || elem.Parent.Wrect[1] != 0.0) {
					// DrawWrect(elem.Parent, cvs, show)
				}
			}
		} else {
			if show.Draw[elem.Etype] {
				// DrawSection(elem, cvs, show)
			} else {
				if dr, ok := show.Draw[elem.Sect.Num]; ok {
					if dr {
						// DrawSection(elem, cvs, show)
					}
				}
			}
		}
	} else {
		if elem.Enods < 2 {
			return
		} else if elem.Enods == 2 {
			Line(cvs, pen, int(elem.Enod[0].Pcoord[0]), int(elem.Enod[0].Pcoord[1]), int(elem.Enod[1].Pcoord[0]), int(elem.Enod[1].Pcoord[1]))
			return
		}
		vers := make([][]int, elem.Enods)
		for i, en := range elem.Enod {
			vers[i] = []int{int(en.Pcoord[0]), int(en.Pcoord[1])}
		}
		Polygon(cvs, PlateEdgePen, brush, vers)
		if elem.Wrect != nil && (elem.Wrect[0] != 0.0 || elem.Wrect[1] != 0.0) {
			// DrawWrect(elem, cvs, show)
		}
		if show.ElemNormal {
			// DrawElemNormal(elem, cvs, show)
		}
	}
}

func DrawElemLine(elem *st.Elem, cvs gxui.Canvas, pen gxui.Pen) {
	Line(cvs, pen, int(elem.Enod[0].Pcoord[0]), int(elem.Enod[0].Pcoord[1]), int(elem.Enod[1].Pcoord[0]), int(elem.Enod[1].Pcoord[1]))
}
