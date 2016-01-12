package st

import (
	"bytes"
	"fmt"
	"github.com/yofu/st/arclm"
	"math"
	"regexp"
	"sync"
)

const deg10 = 10.0 * math.Pi / 180.0

// Line Style
const (
	CONTINUOUS = iota
	DOTTED
	DASHED
	DASH_DOT
)

// Text Alignment
const (
	NORTH = iota
	SOUTH
	WEST
	EAST
	CENTER
	SOUTH_WEST
	SOUTH_EAST
)

const (
	LOCKED_NODE_COLOR = WHITE
	LOCKED_ELEM_COLOR = WHITE
)

type Drawer interface {
	Line(float64, float64, float64, float64)
	Polyline([][]float64)
	Polygon([][]float64)
	Circle(float64, float64, float64)
	FilledCircle(float64, float64, float64)
	Text(float64, float64, string)
	Foreground(int)
	LineStyle(int)
	TextAlignment(int)
	TextOrientation(float64)
	SectionAliase(int) (string, bool)
	SelectedNodes() []*Node
	SelectedElems() []*Elem
	ElemSelected() bool
	DefaultStyle()
	BondStyle(*Show)
	PhingeStyle(*Show)
	ConfStyle(*Show)
	SelectNodeStyle()
	SelectElemStyle()
	ShowPrintRange() bool
	GetCanvasSize() (int, int)
	CanvasPaperSize() (float64, float64, error)
	Flush()
	CanvasDirection() int
}

func DrawElem(stw Drawer, elem *Elem, show *Show) {
	var ecap bytes.Buffer
	var oncap bool
	if show.ElemCaption&EC_NUM != 0 {
		ecap.WriteString(fmt.Sprintf("%d\n", elem.Num))
		oncap = true
	}
	if show.ElemCaption&EC_SECT != 0 {
		if al, ok := stw.SectionAliase(elem.Sect.Num); ok {
			ecap.WriteString(fmt.Sprintf("%s\n", al))
		} else {
			ecap.WriteString(fmt.Sprintf("%d\n", elem.Sect.Num))
		}
		oncap = true
	}
	if show.ElemCaption&EC_WIDTH != 0 {
		ecap.WriteString(fmt.Sprintf("%.3f\n", elem.Width()))
		oncap = true
	}
	if show.ElemCaption&EC_HEIGHT != 0 {
		ecap.WriteString(fmt.Sprintf("%.3f\n", elem.Height()))
		oncap = true
	}
	if show.ElemCaption&EC_PREST != 0 {
		if elem.Prestress != 0.0 {
			ecap.WriteString(fmt.Sprintf("%.3f\n", elem.Prestress * show.Unit[0]))
			oncap = true
		}
	}
	if show.ElemCaption&EC_STIFF_X != 0 {
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
	if show.ElemCaption&EC_STIFF_Y != 0 {
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
	if show.ElemCaption&EC_DRIFT_X != 0 {
		drift := elem.StoryDrift("X")
		if drift != 0.0 && !math.IsNaN(drift) {
			ecap.WriteString(fmt.Sprintf("1/%.0f\n", 1.0/math.Abs(drift)))
			oncap = true
		}
	}
	if show.ElemCaption&EC_DRIFT_Y != 0 {
		drift := elem.StoryDrift("Y")
		if drift != 0.0 && !math.IsNaN(drift) {
			ecap.WriteString(fmt.Sprintf("1/%.0f\n", 1.0/math.Abs(drift)))
			oncap = true
		}
	}
	if show.SrcanRate != 0 {
		val, err := elem.RateMax(show)
		if err == nil {
			ecap.WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["RATE"]), val))
			oncap = true
		}
	}
	if show.Energy {
		val, err := elem.Energy()
		if err == nil {
			ecap.WriteString(fmt.Sprintf("%.6f", val))
			oncap = true
		}
	}
	if oncap {
		var textpos []float64
		if BRACE <= elem.Etype && elem.Etype <= SBRACE {
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
		stw.Text(textpos[0], textpos[1], ecap.String())
	}
	if elem.IsLineElem() {
		stw.Line(elem.Enod[0].Pcoord[0], elem.Enod[0].Pcoord[1], elem.Enod[1].Pcoord[0], elem.Enod[1].Pcoord[1])
		pd := elem.PDirection(true)
		if show.Bond {
			stw.BondStyle(show)
			switch elem.BondState() {
			case PIN_RIGID:
				stw.Circle(elem.Enod[0].Pcoord[0]+pd[0]*show.BondSize, elem.Enod[0].Pcoord[1]+pd[1]*show.BondSize, show.BondSize*2)
			case RIGID_PIN:
				stw.Circle(elem.Enod[1].Pcoord[0]-pd[0]*show.BondSize, elem.Enod[1].Pcoord[1]-pd[1]*show.BondSize, show.BondSize*2)
			case PIN_PIN:
				stw.Circle(elem.Enod[0].Pcoord[0]+pd[0]*show.BondSize, elem.Enod[0].Pcoord[1]+pd[1]*show.BondSize, show.BondSize*2)
				stw.Circle(elem.Enod[1].Pcoord[0]-pd[0]*show.BondSize, elem.Enod[1].Pcoord[1]-pd[1]*show.BondSize, show.BondSize*2)
			}
		}
		if show.Phinge {
			stw.PhingeStyle(show)
			ph1 := elem.Phinge[show.Period][elem.Enod[0].Num]
			ph2 := elem.Phinge[show.Period][elem.Enod[1].Num]
			switch {
			case ph1 && !ph2:
				stw.FilledCircle(elem.Enod[0].Pcoord[0]+pd[0]*show.BondSize, elem.Enod[0].Pcoord[1]+pd[1]*show.BondSize, show.BondSize*2)
			case !ph1 && ph2:
				stw.FilledCircle(elem.Enod[1].Pcoord[0]-pd[0]*show.BondSize, elem.Enod[1].Pcoord[1]-pd[1]*show.BondSize, show.BondSize*2)
			case ph1 && ph2:
				stw.FilledCircle(elem.Enod[0].Pcoord[0]+pd[0]*show.BondSize, elem.Enod[0].Pcoord[1]+pd[1]*show.BondSize, show.BondSize*2)
				stw.FilledCircle(elem.Enod[1].Pcoord[0]-pd[0]*show.BondSize, elem.Enod[1].Pcoord[1]-pd[1]*show.BondSize, show.BondSize*2)
			}
		}
		if show.ElementAxis {
			DrawElementAxis(stw, elem, show)
		}
		// Deformation
		if show.Deformation {
			stw.LineStyle(DOTTED)
			stw.Foreground(show.DeformationColor)
			stw.Line(elem.Enod[0].Dcoord[0], elem.Enod[0].Dcoord[1], elem.Enod[1].Dcoord[0], elem.Enod[1].Dcoord[1])
			stw.LineStyle(CONTINUOUS)
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
			for i, st := range []uint{STRESS_NZ, STRESS_QX, STRESS_QY, STRESS_MZ, STRESS_MX, STRESS_MY} {
				if flag&st != 0 {
					switch i {
					case 0:
						sttext[0].WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["STRESS"]), elem.ReturnStress(show.Period, 0, i) * show.Unit[0]))
					case 1, 2:
						vali := elem.ReturnStress(show.Period, 0, i) * show.Unit[0]
						valj := elem.ReturnStress(show.Period, 1, i) * show.Unit[0]
						if !show.NoShearValue {
							sttext[0].WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["STRESS"]), vali))
							sttext[1].WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["STRESS"]), valj))
						}
						if show.ShearArrow {
							arrow := 0.3
							qcoord := elem.MidPoint()
							rcoord := elem.MidPoint()
							prcoord := elem.Frame.View.ProjectCoord(rcoord)
							val := 0.5 * (vali - valj)
							var vec []float64
							if i == 1 {
								vec = elem.Strong
							} else {
								vec = elem.Weak
							}
							if val >= 0.0 {
								for j:=0; j<3; j++ {
									qcoord[j] -= show.Qfact * val * vec[j]
								}
								pqcoord := elem.Frame.View.ProjectCoord(qcoord)
								Arrow(stw, pqcoord[0], pqcoord[1], prcoord[0], prcoord[1], arrow, deg10)
							} else {
								for j:=0; j<3; j++ {
									qcoord[j] += show.Qfact * val * vec[j]
								}
								pqcoord := elem.Frame.View.ProjectCoord(qcoord)
								Arrow(stw, prcoord[0], prcoord[1], pqcoord[0], pqcoord[1], arrow, deg10)
							}
						}
					case 3:
						sttext[0].WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["STRESS"]), elem.ReturnStress(show.Period, 0, i) * show.Unit[0] * show.Unit[1]))
						sttext[1].WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["STRESS"]), elem.ReturnStress(show.Period, 1, i) * show.Unit[0] * show.Unit[1]))
					case 4, 5:
						if !show.NoMomentValue {
							sttext[0].WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["STRESS"]), elem.ReturnStress(show.Period, 0, i) * show.Unit[0] * show.Unit[1]))
							sttext[1].WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["STRESS"]), elem.ReturnStress(show.Period, 1, i) * show.Unit[0] * show.Unit[1]))
						}
						if show.MomentFigure {
							mcoord := elem.MomentCoord(show, i)
							stw.Foreground(show.MomentColor)
							coords := make([][]float64, len(mcoord)+2)
							coords[0] = []float64{elem.Enod[0].Pcoord[0], elem.Enod[0].Pcoord[1]}
							for i, c := range mcoord {
								coords[i+1] = elem.Frame.View.ProjectCoord(c)
							}
							coords[len(mcoord)+1] = []float64{elem.Enod[1].Pcoord[0], elem.Enod[1].Pcoord[1]}
							stw.Polyline(coords)
						}
					}
				}
			}
			stw.Foreground(show.StressTextColor)
			for j := 0; j < 2; j++ {
				if tex := sttext[j].String(); tex != "" {
					coord := make([]float64, 3)
					for i, en := range elem.Enod {
						for k := 0; k < 3; k++ {
							coord[k] += (-0.5*math.Abs(float64(i-j)) + 0.75) * en.Coord[k]
						}
					}
					stpos := elem.Frame.View.ProjectCoord(coord)
					if j == 0 {
						stw.TextAlignment(SOUTH)
					} else {
						stw.TextAlignment(NORTH)
					}
					deg := math.Atan2(pd[1], pd[0]) * 180.0 / math.Pi
					if deg > 90.0 {
						deg -= 180.0
					} else if deg < -90.0 {
						deg += 180.0
					}
					stw.TextOrientation(deg)
					stw.Text(stpos[0], stpos[1], tex[:len(tex)-1])
					stw.TextAlignment(show.DefaultTextAlignment)
					stw.TextOrientation(0.0)
				}
			}
		}
		if show.YieldFunction {
			f, err := elem.YieldFunction(show.Period)
			for j := 0; j < 2; j++ {
				switch err[j].(type) {
				default:
					stw.Foreground(show.StressTextColor)
				case arclm.YieldedError:
					stw.Foreground(show.YieldedTextColor)
				case arclm.BrittleFailureError:
					stw.Foreground(show.BrittleTextColor)
				}
				coord := make([]float64, 3)
				for i, en := range elem.Enod {
					for k := 0; k < 3; k++ {
						coord[k] += (-0.5*math.Abs(float64(i-j)) + 0.75) * en.Coord[k]
					}
				}
				stpos := elem.Frame.View.ProjectCoord(coord)
				if j == 0 {
					stw.TextAlignment(SOUTH)
				} else {
					stw.TextAlignment(NORTH)
				}
				deg := math.Atan2(pd[1], pd[0]) * 180.0 / math.Pi
				if deg > 90.0 {
					deg -= 180.0
				} else if deg < -90.0 {
					deg += 180.0
				}
				stw.TextOrientation(deg)
				stw.Text(stpos[0], stpos[1], fmt.Sprintf("%.3f", f[j]))
				stw.TextAlignment(show.DefaultTextAlignment)
				stw.TextOrientation(0.0)
			}
		}
		if elem.Etype == WBRACE || elem.Etype == SBRACE {
			if elem.Eldest {
				if elem.Parent.Wrect != nil && (elem.Parent.Wrect[0] != 0.0 || elem.Parent.Wrect[1] != 0.0) {
					DrawWrect(stw, elem.Parent, show)
				}
			}
		} else {
			if show.Draw[elem.Etype] {
				DrawSection(stw, elem, show)
			} else {
				if dr, ok := show.Draw[elem.Sect.Num]; ok {
					if dr {
						DrawSection(stw, elem, show)
					}
				}
			}
		}
	} else {
		if elem.Enods < 2 {
			return
		} else if elem.Enods == 2 {
			stw.Line(elem.Enod[0].Pcoord[0], elem.Enod[0].Pcoord[1], elem.Enod[1].Pcoord[0], elem.Enod[1].Pcoord[1])
			return
		}
		coords := make([][]float64, elem.Enods)
		for i := 0; i < elem.Enods; i++ {
			coords[i] = []float64{elem.Enod[i].Pcoord[0], elem.Enod[i].Pcoord[1]}
		}
		stw.Polygon(coords)
		stw.Foreground(show.PlateEdgeColor)
		stw.Polyline(coords)
		// Stress
		var flag uint
		if f, ok := show.Stress[elem.Sect.Num]; ok {
			flag = f
		} else if f, ok := show.Stress[elem.Etype-2]; ok {
			flag = f
		}
		if flag != 0 {
			var sttext bytes.Buffer
			for i, st := range []uint{STRESS_NZ, STRESS_QX, STRESS_QY} {
				if flag&st != 0 {
					vec := []float64{0.0, 0.0, 0.0}
					switch i {
					case 0:
						vec[2] = 1.0
					case 1:
						vec[0] = 1.0
					case 2:
						vec[1] = 1.0
					}
					val := elem.PlateStress(show.Period, vec) * show.Unit[0]
					if val != 0.0 && !show.NoShearValue {
						sttext.WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["STRESS"]), val))
					}
					if show.ShearArrow {
						arrow := 0.3
						qcoord := elem.MidPoint()
						rcoord := elem.MidPoint()
						prcoord := elem.Frame.View.ProjectCoord(rcoord)
						if val >= 0.0 {
							for j:=0; j<3; j++ {
								qcoord[j] -= show.Qfact * val * vec[j]
							}
							pqcoord := elem.Frame.View.ProjectCoord(qcoord)
							Arrow(stw, pqcoord[0], pqcoord[1], prcoord[0], prcoord[1], arrow, deg10)
						} else {
							for j:=0; j<3; j++ {
								qcoord[j] += show.Qfact * val * vec[j]
							}
							pqcoord := elem.Frame.View.ProjectCoord(qcoord)
							Arrow(stw, prcoord[0], prcoord[1], pqcoord[0], pqcoord[1], arrow, deg10)
						}
					}
				}
			}
			if tex := sttext.String(); tex != "" {
				tcoord := elem.Frame.View.ProjectCoord(elem.MidPoint())
				stw.Text(tcoord[0], tcoord[1], tex)
			}
		}
		if elem.Wrect != nil && (elem.Wrect[0] != 0.0 || elem.Wrect[1] != 0.0) {
			DrawWrect(stw, elem, show)
		}
		if show.ElemNormal {
			DrawElemNormal(stw, elem, show)
		}
	}
}

func DrawElemLine(stw Drawer, elem *Elem) {
	stw.Line(elem.Enod[0].Pcoord[0], elem.Enod[0].Pcoord[1], elem.Enod[1].Pcoord[0], elem.Enod[1].Pcoord[1])
}

func Arrow(stw Drawer, x1, y1, x2, y2, size, theta float64) {
	c := size * math.Cos(theta)
	s := size * math.Sin(theta)
	stw.Line(x1, y1, x2, y2)
	stw.Line(x2, y2, x2+((x1-x2)*c-(y1-y2)*s), y2+((x1-x2)*s+(y1-y2)*c))
	stw.Line(x2, y2, x2+((x1-x2)*c+(y1-y2)*s), y2+(-(x1-x2)*s+(y1-y2)*c))
}

func DrawGlobalAxis(stw Drawer, frame *Frame, color uint) {
	origin := frame.View.ProjectCoord([]float64{0.0, 0.0, 0.0})
	xaxis := frame.View.ProjectCoord([]float64{frame.Show.GlobalAxisSize, 0.0, 0.0})
	yaxis := frame.View.ProjectCoord([]float64{0.0, frame.Show.GlobalAxisSize, 0.0})
	zaxis := frame.View.ProjectCoord([]float64{0.0, 0.0, frame.Show.GlobalAxisSize})
	size := 0.3
	stw.LineStyle(CONTINUOUS)
	switch color {
	case ECOLOR_BLACK:
		stw.Foreground(BLACK)
		Arrow(stw, origin[0], origin[1], xaxis[0], xaxis[1], size, deg10)
		Arrow(stw, origin[0], origin[1], yaxis[0], yaxis[1], size, deg10)
		Arrow(stw, origin[0], origin[1], zaxis[0], zaxis[1], size, deg10)
	default:
		stw.Foreground(RED)
		Arrow(stw, origin[0], origin[1], xaxis[0], xaxis[1], size, deg10)
		stw.Foreground(GREEN)
		Arrow(stw, origin[0], origin[1], yaxis[0], yaxis[1], size, deg10)
		stw.Foreground(BLUE)
		Arrow(stw, origin[0], origin[1], zaxis[0], zaxis[1], size, deg10)
		stw.Foreground(WHITE)
	}
}

func DrawElementAxis(stw Drawer, elem *Elem, show *Show) {
	if !elem.IsLineElem() {
		return
	}
	x := make([]float64, 3)
	y := make([]float64, 3)
	z := make([]float64, 3)
	position := elem.MidPoint()
	d := elem.Direction(true)
	for i := 0; i < 3; i++ {
		x[i] = position[i] + show.ElementAxisSize*elem.Strong[i]
		y[i] = position[i] + show.ElementAxisSize*elem.Weak[i]
		z[i] = position[i] + show.ElementAxisSize*d[i]
	}
	origin := elem.Frame.View.ProjectCoord(position)
	xaxis := elem.Frame.View.ProjectCoord(x)
	yaxis := elem.Frame.View.ProjectCoord(y)
	zaxis := elem.Frame.View.ProjectCoord(z)
	arrow := 0.3
	stw.LineStyle(CONTINUOUS)
	stw.Foreground(RED)
	Arrow(stw, origin[0], origin[1], xaxis[0], xaxis[1], arrow, deg10)
	stw.Foreground(GREEN)
	Arrow(stw, origin[0], origin[1], yaxis[0], yaxis[1], arrow, deg10)
	stw.Foreground(BLUE)
	Arrow(stw, origin[0], origin[1], zaxis[0], zaxis[1], arrow, deg10)
	stw.Foreground(WHITE)
}

func DrawWrect(stw Drawer, elem *Elem, show *Show) {
}

func DrawElemNormal(stw Drawer, elem *Elem, show *Show) {
}

// func DrawClosedLine(elem *Elem, cvs *cd.Canvas, position []float64, scale float64, vertices [][]float64) {
func DrawClosedLine(stw Drawer, elem *Elem, position []float64, scale float64, vertices [][]float64) {
	// coord := make([]float64, 3)
	// cvs.Begin(cd.CD_CLOSED_LINES)
	// for _, v := range vertices {
	// 	if v == nil {
	// 		cvs.End()
	// 		cvs.Begin(cd.CD_CLOSED_LINES)
	// 		continue
	// 	}
	// 	coord[0] = position[0] + (v[0]*elem.Strong[0]+v[1]*elem.Weak[0])*0.01*scale
	// 	coord[1] = position[1] + (v[0]*elem.Strong[1]+v[1]*elem.Weak[1])*0.01*scale
	// 	coord[2] = position[2] + (v[0]*elem.Strong[2]+v[1]*elem.Weak[2])*0.01*scale
	// 	pc := elem.Frame.View.ProjectCoord(coord)
	// 	cvs.FVertex(pc[0], pc[1])
	// }
	// cvs.End()
}

// func DrawSection(elem *Elem, cvs *cd.Canvas, show *Show) {
func DrawSection(stw Drawer, elem *Elem, show *Show) {
	// if al, ok := elem.Frame.Allows[elem.Sect.Num]; ok {
	// 	position := elem.MidPoint()
	// 	switch al.(type) {
	// 	case *SColumn:
	// 		sh := al.(*SColumn).Shape
	// 		switch sh.(type) {
	// 		case HKYOU, HWEAK, RPIPE, CPIPE, PLATE:
	// 			vertices := sh.Vertices()
	// 			DrawClosedLine(elem, cvs, position, show.DrawSize, vertices)
	// 		}
	// 	case *RCColumn:
	// 		rc := al.(*RCColumn)
	// 		vertices := rc.CShape.Vertices()
	// 		DrawClosedLine(elem, cvs, position, show.DrawSize, vertices)
	// 		for _, reins := range rc.Reins {
	// 			vertices = reins.Vertices()
	// 			DrawClosedLine(elem, cvs, position, show.DrawSize, vertices)
	// 		}
	// 	case *RCGirder:
	// 		rg := al.(*RCGirder)
	// 		vertices := rg.CShape.Vertices()
	// 		DrawClosedLine(elem, cvs, position, show.DrawSize, vertices)
	// 		for _, reins := range rg.Reins {
	// 			vertices = reins.Vertices()
	// 			DrawClosedLine(elem, cvs, position, show.DrawSize, vertices)
	// 		}
	// 	case *WoodColumn:
	// 		sh := al.(*WoodColumn).Shape
	// 		switch sh.(type) {
	// 		case PLATE:
	// 			vertices := sh.Vertices()
	// 			DrawClosedLine(elem, cvs, position, show.DrawSize, vertices)
	// 		}
	// 	}
	// }
}

// KIJUN
var fl = regexp.MustCompile("FL$")

func DrawKijun(stw Drawer, k *Kijun, show *Show) {
	d := k.PDirection(true)
	if math.Abs(d[0]) <= 1e-6 && math.Abs(d[1]) <= 1e-6 {
		return
	}
	stw.LineStyle(DASH_DOT)
	stw.Line(k.Pstart[0], k.Pstart[1], k.Pend[0], k.Pend[1])
	stw.LineStyle(CONTINUOUS)
	switch {
	default:
		stw.TextAlignment(CENTER)
		stw.Circle(k.Pstart[0]-d[0]*show.KijunSize, k.Pstart[1]-d[1]*show.KijunSize, show.KijunSize*2)
		if k.Name == "" {
		} else if k.Name[0] == '_' {
			stw.Text(k.Pstart[0]-d[0]*show.KijunSize, k.Pstart[1]-d[1]*show.KijunSize, k.Name[1:])
		} else {
			stw.Text(k.Pstart[0]-d[0]*show.KijunSize, k.Pstart[1]-d[1]*show.KijunSize, k.Name)
		}
		stw.TextAlignment(show.DefaultTextAlignment)
	case fl.MatchString(k.Name):
		if d[0] >= 0.0 {
			stw.TextAlignment(SOUTH_WEST)
		} else {
			stw.TextAlignment(SOUTH_EAST)
		}
		deg := math.Atan2(d[1], d[0]) * 180.0 / math.Pi
		if deg > 90.0 {
			deg -= 180.0
		} else if deg < -90.0 {
			deg += 180.0
		}
		stw.TextOrientation(deg)
		stw.Text(k.Pstart[0], k.Pstart[1], k.Name)
		stw.TextOrientation(0.0)
		stw.TextAlignment(show.DefaultTextAlignment)
	}
}

func DrawMeasure(stw Drawer, m *Measure, show *Show) {
	n1 := make([]float64, 3)
	n2 := make([]float64, 3)
	n3 := make([]float64, 3)
	n4 := make([]float64, 3)
	for i := 0; i < 3; i++ {
		d1 := m.Direction[i] * m.Gap
		d2 := m.Direction[i] * (m.Gap + m.Extension)
		n1[i] = m.Start[i] + d1
		n2[i] = m.Start[i] + d2
		n3[i] = m.End[i] + d2
		n4[i] = m.End[i] + d1
	}
	pn1 := m.Frame.View.ProjectCoord(n1)
	pn2 := m.Frame.View.ProjectCoord(n2)
	pn3 := m.Frame.View.ProjectCoord(n3)
	pn4 := m.Frame.View.ProjectCoord(n4)
	stw.Line(pn1[0], pn1[1], pn2[0], pn2[1])
	stw.Line(pn2[0], pn2[1], pn3[0], pn3[1])
	stw.Line(pn3[0], pn3[1], pn4[0], pn4[1])
	stw.FilledCircle(pn2[0], pn2[1], m.ArrowSize)
	stw.FilledCircle(pn3[0], pn3[1], m.ArrowSize)
	xpos := 0.5 * (pn2[0] + pn3[0])
	ypos := 0.5 * (pn2[1] + pn3[1])
	pd := make([]float64, 2)
	pd[0] = pn3[0] - pn2[0]
	pd[1] = pn3[1] - pn2[1]
	l := math.Sqrt(pd[0]*pd[0] + pd[1]*pd[1])
	if l != 0.0 {
		pd[0] /= l
		pd[1] /= l
	}
	deg := math.Atan2(pd[1], pd[0])*180.0/math.Pi + m.Rotate
	if deg > 90.0 {
		deg -= 180.0
	} else if deg < -90.0 {
		deg += 180.0
	}
	stw.TextOrientation(deg)
	stw.Text(xpos, ypos, m.Text)
	stw.TextOrientation(0.0)
}

// TEXT
func DrawText(stw Drawer, t TextBox) {
	// s := cvs.SaveState()
	// cvs.Font(t.Font.Face, cd.CD_PLAIN, t.Font.Size)
	// cvs.Foreground(t.Font.Color)
	fsize := 8
	for i, txt := range t.Text() {
		xpos, ypos := t.Position()
		ypos -= float64(i*fsize)*1.5 + float64(fsize)
		stw.Text(xpos, ypos, txt)
	}
	// cvs.RestoreState(s)
}

func DrawNode(stw Drawer, node *Node, show *Show) {
	// Caption
	var ncap bytes.Buffer
	var oncap bool
	if show.NodeCaption&NC_NUM != 0 {
		ncap.WriteString(fmt.Sprintf("%d\n", node.Num))
		oncap = true
	}
	if show.NodeCaption&NC_WEIGHT != 0 {
		if !node.Conf[2] || show.NodeCaption&NC_RZ == 0 {
			ncap.WriteString(fmt.Sprintf("%.3f\n", node.Weight[1] * show.Unit[0]))
			oncap = true
		}
	}
	for i, j := range []uint{NC_DX, NC_DY, NC_DZ, NC_TX, NC_TY, NC_TZ} {
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
	for i, j := range []uint{NC_RX, NC_RY, NC_RZ, NC_MX, NC_MY, NC_MZ} {
		if show.NodeCaption&j != 0 {
			if node.Conf[i] {
				var val float64
				if i == 2 && show.NodeCaption&NC_WEIGHT != 0 {
					val = node.ReturnReaction(show.Period, i) + node.Weight[1]
				} else {
					val = node.ReturnReaction(show.Period, i)
				}
				switch i {
				case 0, 1, 2:
					stw.LineStyle(CONTINUOUS)
					val *= show.Unit[0]
					ncap.WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["REACTION"]), val))
					arrow := 0.3
					rcoord := []float64{node.Coord[0], node.Coord[1], node.Coord[2]}
					if val >= 0.0 {
						rcoord[i] -= show.Rfact * val
						prcoord := node.Frame.View.ProjectCoord(rcoord)
						Arrow(stw, prcoord[0], prcoord[1], node.Pcoord[0], node.Pcoord[1], arrow, deg10)
					} else {
						rcoord[i] += show.Rfact * val
						prcoord := node.Frame.View.ProjectCoord(rcoord)
						Arrow(stw, node.Pcoord[0], node.Pcoord[1], prcoord[0], prcoord[1], arrow, deg10)
					}
				case 3, 4, 5:
					val *= show.Unit[0] * show.Unit[1]
					ncap.WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["REACTION"]), val))
				}
				oncap = true
			}
		}
	}
	if show.NodeCaption&NC_ZCOORD != 0 {
		ncap.WriteString(fmt.Sprintf("%.1f\n", node.Coord[2]))
		oncap = true
	}
	if show.NodeCaption&NC_PILE != 0 {
		if node.Pile != nil {
			ncap.WriteString(fmt.Sprintf("%d\n", node.Pile.Num))
			oncap = true
		}
	}
	if oncap {
		stw.Text(node.Pcoord[0], node.Pcoord[1], ncap.String())
	}
	if show.NodeNormal {
		DrawNodeNormal(stw, node, show)
	}
	// Conffigure
	if show.Conf {
		stw.ConfStyle(show)
		switch node.ConfState() {
		default:
			return
		case CONF_PIN:
			PinFigure(stw, node.Pcoord[0], node.Pcoord[1], show.ConfSize)
		case CONF_XROL, CONF_YROL, CONF_XYROL:
			RollerFigure(stw, node.Pcoord[0], node.Pcoord[1], show.ConfSize, 0)
		case CONF_ZROL:
			RollerFigure(stw, node.Pcoord[0], node.Pcoord[1], show.ConfSize, 1)
		case CONF_FIX:
			FixFigure(stw, node.Pcoord[0], node.Pcoord[1], show.ConfSize)
		}
	}
}

func DrawNodeNum(stw Drawer, node *Node) {
	stw.Text(node.Pcoord[0], node.Pcoord[1], fmt.Sprintf("%d", node.Num))
}

func PinFigure(stw Drawer, x, y, size float64) {
	val := y - 0.5*math.Sqrt(3)*size
	coords := make([][]float64, 3)
	coords[0] = []float64{x, y}
	coords[1] = []float64{x+0.5*size, val}
	coords[2] = []float64{x-0.5*size, val}
	stw.Polygon(coords)
}

func RollerFigure(stw Drawer, x, y, size float64, direction int) {
	switch direction {
	case 0:
		val1 := y - 0.5*math.Sqrt(3)*size
		val2 := y - 0.75*math.Sqrt(3)*size
		coords := make([][]float64, 3)
		coords[0] = []float64{x, y}
		coords[1] = []float64{x+0.5*size, val1}
		coords[2] = []float64{x-0.5*size, val1}
		stw.Polygon(coords)
		stw.Line(x-0.5*size, val2, x+0.5*size, val2)
	case 1:
		val1 := x - 0.5*math.Sqrt(3)*size
		val2 := x - 0.75*math.Sqrt(3)*size
		coords := make([][]float64, 3)
		coords[0] = []float64{x, y}
		coords[1] = []float64{val1, y+0.5*size}
		coords[2] = []float64{val1, y-0.5*size}
		stw.Polygon(coords)
		stw.Line(val2, y-0.5*size, val2, y+0.5*size)
	}
}

func FixFigure(stw Drawer, x, y, size float64) {
	stw.Line(x-size, y, x+size, y)
	stw.Line(x-0.25*size, y, x-0.75*size, y-0.5*size)
	stw.Line(x+0.25*size, y, x-0.25*size, y-0.5*size)
	stw.Line(x+0.75*size, y, x+0.25*size, y-0.5*size)
}

func DrawNodeNormal(stw Drawer, node *Node, show *Show) {
	v := make([]float64, 3)
	d := node.Normal(true)
	for i := 0; i < 3; i++ {
		v[i] = node.Coord[i] + show.NodeNormalSize*d[i]
	}
	vec := node.Frame.View.ProjectCoord(v)
	arrow := 0.3
	stw.LineStyle(CONTINUOUS)
	Arrow(stw, node.Pcoord[0], node.Pcoord[1], vec[0], vec[1], arrow, deg10)
}

func DrawFrame(stw Drawer, frame *Frame, color uint, flush bool) {
	if frame == nil {
		if flush {
			stw.Flush()
		}
		return
	}
	stw.DefaultStyle()
	show := frame.Show
	frame.View.Set(stw.CanvasDirection())
	if show.GlobalAxis {
		DrawGlobalAxis(stw, frame, color)
	}
	if show.Kijun {
		stw.Foreground(show.KijunColor)
		for _, k := range frame.Kijuns {
			if k.IsHidden(show) {
				continue
			}
			k.Pstart = frame.View.ProjectCoord(k.Start)
			k.Pend = frame.View.ProjectCoord(k.End)
			DrawKijun(stw, k, show)
		}
	}
	if show.Measure {
		stw.TextAlignment(SOUTH)
		stw.Foreground(show.MeasureColor)
		for _, m := range frame.Measures {
			if m.IsHidden(show) {
				continue
			}
			DrawMeasure(stw, m, show)
		}
		stw.TextAlignment(show.DefaultTextAlignment)
	}
	stw.Foreground(WHITE)
	for _, n := range frame.Nodes {
		frame.View.ProjectNode(n)
		if show.Deformation {
			frame.View.ProjectDeformation(n, show)
		}
		if n.IsHidden(show) {
			continue
		}
		if color == ECOLOR_BLACK {
			stw.Foreground(BLACK)
		} else {
			if n.Lock {
				stw.Foreground(LOCKED_NODE_COLOR)
			} else {
				switch n.ConfState() {
				case CONF_FREE:
					stw.Foreground(show.CanvasFontColor)
				case CONF_PIN:
					stw.Foreground(GREEN)
				case CONF_FIX:
					stw.Foreground(DARK_GREEN)
				default:
					stw.Foreground(CYAN)
				}
			}
			for _, j := range stw.SelectedNodes() {
				if j == n {
					stw.SelectNodeStyle()
					break
				}
			}
		}
		DrawNode(stw, n, show)
	}
	if !frame.Show.Select {
		els := SortedElem(frame.Elems, func(e *Elem) float64 { return -e.DistFromProjection(frame.View) })
	loop:
		for _, el := range els {
			if el.IsHidden(frame.Show) {
				continue
			}
			for _, j := range stw.SelectedElems() {
				if j == el {
					continue loop
				}
			}
			stw.DefaultStyle()
			if el.Lock {
				stw.Foreground(LOCKED_ELEM_COLOR)
			} else {
				switch color {
				case ECOLOR_WHITE:
					stw.Foreground(WHITE)
				case ECOLOR_BLACK:
					stw.Foreground(BLACK)
				case ECOLOR_SECT:
					stw.Foreground(el.Sect.Color)
				case ECOLOR_RATE:
					val, err := el.RateMax(show)
					if err != nil {
						stw.Foreground(DARK_GRAY)
					} else {
						stw.Foreground(Rainbow(val, RateBoundary))
					}
				case ECOLOR_N:
					if el.N(frame.Show.Period, 0) >= 0.0 {
						stw.Foreground(RainbowColor[0]) // Compression: Blue
					} else {
						stw.Foreground(RainbowColor[6]) // Tension: Red
					}
				case ECOLOR_STRONG:
					if el.IsLineElem() {
						Ix, err := el.Sect.Ix(0)
						if err != nil {
							stw.Foreground(WHITE)
						}
						Iy, err := el.Sect.Iy(0)
						if err != nil {
							stw.Foreground(WHITE)
						}
						if Ix > Iy {
							stw.Foreground(RainbowColor[0]) // Strong: Blue
						} else if Ix == Iy {
							stw.Foreground(RainbowColor[4]) // Same: Yellow
						} else {
							stw.Foreground(RainbowColor[6]) // Weak: Red
						}
					} else {
						stw.Foreground(el.Sect.Color)
					}
				case ECOLOR_ENERGY:
					val, err := el.Energy()
					if err != nil {
						stw.Foreground(DARK_GRAY)
					} else {
						stw.Foreground(Rainbow(val, EnergyBoundary))
					}
				}
			}
			DrawElem(stw, el, show)
		}
	}
	nomv := show.NoMomentValue
	nosv := show.NoShearValue
	show.NoMomentValue = false
	show.NoShearValue = false
	for _, el := range stw.SelectedElems() {
		if el == nil || el.IsHidden(frame.Show) {
			continue
		}
		stw.SelectElemStyle()
		if el.Lock {
			stw.Foreground(LOCKED_ELEM_COLOR)
		} else {
			switch color {
			case ECOLOR_WHITE:
				stw.Foreground(WHITE)
			case ECOLOR_BLACK:
				stw.Foreground(BLACK)
			case ECOLOR_SECT:
				stw.Foreground(el.Sect.Color)
			case ECOLOR_RATE:
				val, err := el.RateMax(frame.Show)
				if err != nil {
					stw.Foreground(DARK_GRAY)
				} else {
					stw.Foreground(Rainbow(val, RateBoundary))
				}
			case ECOLOR_N:
				if el.N(frame.Show.Period, 0) >= 0.0 {
					stw.Foreground(RainbowColor[0]) // Compression: Blue
				} else {
					stw.Foreground(RainbowColor[6]) // Tension: Red
				}
			case ECOLOR_ENERGY:
				val, err := el.Energy()
				if err != nil {
					stw.Foreground(DARK_GRAY)
				} else {
					stw.Foreground(Rainbow(val, EnergyBoundary))
				}
			}
		}
		DrawElem(stw, el, show)
	}
	show.NoMomentValue = nomv
	show.NoShearValue = nosv
	stw.DefaultStyle()
	if frame.Fes != nil {
		// DrawEccentric(stw, frame, show)
	}
	if stw.ShowPrintRange() {
		if color == ECOLOR_BLACK {
			stw.Foreground(BLACK)
		} else {
			stw.Foreground(GRAY)
		}
		DrawPrintRange(stw)
	}
	stw.DefaultStyle()
	DrawLegend(stw, show)
	// DrawRange(stw, RangeView)
	if flush {
		stw.Flush()
	}
}

func DrawFrameNode(stw Drawer, frame *Frame, color uint, flush bool) {
	if frame == nil {
		if flush {
			stw.Flush()
		}
		return
	}
	show := frame.Show
	frame.View.Set(stw.CanvasDirection())
	if show.GlobalAxis {
		DrawGlobalAxis(stw, frame, color)
	}
	for _, n := range frame.Nodes {
		frame.View.ProjectNode(n)
		if show.Deformation {
			frame.View.ProjectDeformation(n, show)
		}
		stw.Foreground(GREEN)
		for _, j := range stw.SelectedNodes() {
			if j == n {
				DrawNode(stw, n, show)
				break
			}
		}
	}
	if !show.Select {
		stw.LineStyle(CONTINUOUS)
		var wg sync.WaitGroup
		var m sync.Mutex
		for _, elem := range frame.Elems {
			wg.Add(1)
			go func(el *Elem) {
				defer wg.Done()
				if el.Etype >= WBRACE || el.Etype < COLUMN {
					return
				}
				for _, j := range stw.SelectedElems() {
					if j == el {
						return
					}
				}
				m.Lock()
				if el.IsHidden(show) {
					stw.Foreground(DARK_GRAY)
				} else {
					stw.Foreground(DARK_GREEN)
				}
				DrawElemLine(stw, el)
				m.Unlock()
			}(elem)
		}
		wg.Wait()
	}
	if stw.ElemSelected() {
		nomv := show.NoMomentValue
		show.NoMomentValue = false
		for _, el := range stw.SelectedElems() {
			stw.LineStyle(DOTTED)
			if el == nil || el.IsHidden(show) {
				continue
			}
			if el.Lock {
				stw.Foreground(LOCKED_ELEM_COLOR)
			} else {
				stw.Foreground(GREEN)
			}
			DrawElem(stw, el, show)
		}
		show.NoMomentValue = nomv
		stw.LineStyle(CONTINUOUS)
	}
	// stw.DrawRange(stw.dbuff, RangeView)
	if flush {
		stw.Flush()
	}
}

func DrawPrintRange(stw Drawer) {
	w, h := stw.GetCanvasSize()
	centrex := 0.5 * float64(w)
	centrey := 0.5 * float64(h)
	width, height, err := stw.CanvasPaperSize()
	if err != nil {
		// stw.errormessage(err, ERROR)
		return
	}
	width *= 0.5
	height *= 0.5
	coords := make([][]float64, 4)
	coords[0] = []float64{centrex-width, centrey-height}
	coords[1] = []float64{centrex+width, centrey-height}
	coords[2] = []float64{centrex+width, centrey+height}
	coords[3] = []float64{centrex-width, centrey+height}
	stw.Polyline(coords)
}

func DrawLegend(stw Drawer, show *Show) {
	if !show.NoLegend && show.ColorMode == ECOLOR_RATE {
		ox := float64(show.LegendPosition[0])
		oy := float64(show.LegendPosition[1])
		sz := float64(show.LegendSize)
		for _, col := range RainbowColor {
			stw.Foreground(col)
			coords := make([][]float64, 4)
			coords[0] = []float64{ox, oy}
			coords[1] = []float64{ox, oy+sz}
			coords[2] = []float64{ox+sz, oy+sz}
			coords[3] = []float64{ox+sz, oy}
			stw.Polygon(coords)
			oy += show.LegendLineSep * sz
		}
		stw.Foreground(GRAY)
		stw.TextAlignment(WEST)
		ox += 2 * sz
		oy = float64(show.LegendPosition[1]) - 0.5*(show.LegendLineSep-1.0)*sz
		stw.Text(ox, oy, "0.0")
		oy += show.LegendLineSep * sz
		for i, val := range RateBoundary {
			if i == 3 {
				stw.Text(ox, oy, fmt.Sprintf("%.5f", val))
			} else {
				stw.Text(ox, oy, fmt.Sprintf("%.1f", val))
			}
			oy += show.LegendLineSep * sz
		}
		stw.Text(ox - 2*sz, oy + sz, "安全率の凡例")
		stw.TextAlignment(show.DefaultTextAlignment)
	}
}
