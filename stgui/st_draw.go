package stgui

import (
	"bytes"
	"fmt"
	"github.com/visualfc/go-iup/cd"
	"github.com/yofu/st/stlib"
	"math"
)

// FRAME
func DrawEccentric(frame *st.Frame, cvs *cd.Canvas, show *st.Show) {
	if show.Fes {
		s := cvs.SaveState()
		cvs.LineStyle(cd.CD_CONTINUOUS)
		cvs.InteriorStyle(cd.CD_SOLID)
		wcoord := make([][]float64, frame.Ai.Nfloor-1)
		rcoord := make([][]float64, frame.Ai.Nfloor-1)
		for i := 0; i < frame.Ai.Nfloor-1; i++ {
			wcoord[i] = frame.View.ProjectCoord([]float64{frame.Fes.CentreOfWeight[i+1][0], frame.Fes.CentreOfWeight[i+1][1], frame.Fes.AverageLevel[i+1]})
			rcoord[i] = frame.View.ProjectCoord([]float64{frame.Fes.CentreOfRigid[i][0], frame.Fes.CentreOfRigid[i][1], frame.Fes.AverageLevel[i+1]})
			cvs.Foreground(cd.CD_WHITE)
			cvs.FLine(wcoord[i][0], wcoord[i][1], rcoord[i][0], rcoord[i][1])
			cvs.Foreground(cd.CD_DARK_RED)
			cvs.FFilledCircle(wcoord[i][0], wcoord[i][1], show.MassSize*frame.Fes.TotalWeight[i+1])
			cvs.Foreground(cd.CD_BLUE)
			if i >= 1 {
				cvs.FLine(rcoord[i-1][0], rcoord[i-1][1], rcoord[i][0], rcoord[i][1])
			} else {
				coord := frame.View.ProjectCoord([]float64{frame.Fes.CentreOfRigid[i][0], frame.Fes.CentreOfRigid[i][1], frame.Fes.AverageLevel[i]})
				cvs.FLine(coord[0], coord[1], rcoord[i][0], rcoord[i][1])
			}
		}
		cvs.RestoreState(s)
	}
}

// NODE
func DrawNode(node *st.Node, cvs *cd.Canvas, show *st.Show) {
	// Caption
	var ncap bytes.Buffer
	var oncap bool
	if show.NodeCaption&st.NC_NUM != 0 {
		ncap.WriteString(fmt.Sprintf("%d\n", node.Num))
		oncap = true
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
				ncap.WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["REACTION"]), node.ReturnReaction(show.Period, i)))
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
		cvs.FText(node.Pcoord[0], node.Pcoord[1], ncap.String())
	}
	if show.NodeNormal {
		DrawNodeNormal(node, cvs, show)
	}
	// Conffigure
	if show.Conf {
		cvs.InteriorStyle(cd.CD_SOLID)
		cvs.Foreground(ConfColor)
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

func PinFigure(cvs *cd.Canvas, x, y, size float64) {
	val := y - 0.5*math.Sqrt(3)*size
	cvs.Begin(cd.CD_FILL)
	cvs.FVertex(x, y)
	cvs.FVertex(x+0.5*size, val)
	cvs.FVertex(x-0.5*size, val)
	cvs.End()
}

func RollerFigure(cvs *cd.Canvas, x, y, size float64, direction int) {
	switch direction {
	case 0:
		val1 := y - 0.5*math.Sqrt(3)*size
		val2 := y - 0.75*math.Sqrt(3)*size
		cvs.Begin(cd.CD_FILL)
		cvs.FVertex(x, y)
		cvs.FVertex(x+0.5*size, val1)
		cvs.FVertex(x-0.5*size, val1)
		cvs.End()
		cvs.FLine(x-0.5*size, val2, x+0.5*size, val2)
	case 1:
		val1 := x - 0.5*math.Sqrt(3)*size
		val2 := x - 0.75*math.Sqrt(3)*size
		cvs.Begin(cd.CD_FILL)
		cvs.FVertex(x, y)
		cvs.FVertex(val1, y+0.5*size)
		cvs.FVertex(val1, y-0.5*size)
		cvs.End()
		cvs.FLine(val2, y-0.5*size, val2, y+0.5*size)
	}
}

func FixFigure(cvs *cd.Canvas, x, y, size float64) {
	cvs.FLine(x-size, y, x+size, y)
	cvs.FLine(x-0.25*size, y, x-0.75*size, y-0.5*size)
	cvs.FLine(x+0.25*size, y, x-0.25*size, y-0.5*size)
	cvs.FLine(x+0.75*size, y, x+0.25*size, y-0.5*size)
}

func DrawNodeNormal(node *st.Node, canv *cd.Canvas, show *st.Show) {
	v := make([]float64, 3)
	d := node.Normal(true)
	for i := 0; i < 3; i++ {
		v[i] = node.Coord[i] + show.NodeNormalSize*d[i]
	}
	vec := node.Frame.View.ProjectCoord(v)
	arrow := 0.3
	angle := 10.0 * math.Pi / 180.0
	canv.LineStyle(cd.CD_CONTINUOUS)
	Arrow(canv, node.Pcoord[0], node.Pcoord[1], vec[0], vec[1], arrow, angle)
}

// ELEM
func DrawElem(elem *st.Elem, cvs *cd.Canvas, show *st.Show) {
	var ecap bytes.Buffer
	var oncap bool
	if show.ElemCaption&st.EC_NUM != 0 {
		ecap.WriteString(fmt.Sprintf("%d\n", elem.Num))
		oncap = true
	}
	if show.ElemCaption&st.EC_SECT != 0 {
		if al, ok := sectionaliases[elem.Sect.Num]; ok {
			ecap.WriteString(fmt.Sprintf("%s\n", al))
		} else {
			ecap.WriteString(fmt.Sprintf("%d\n", elem.Sect.Num))
		}
		oncap = true
	}
	if show.ElemCaption&st.EC_RATE_L != 0 || show.ElemCaption&st.EC_RATE_S != 0 {
		val := elem.RateMax(show)
		ecap.WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["RATE"]), val))
		oncap = true
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
		cvs.FText(textpos[0], textpos[1], ecap.String())
	}
	if elem.IsLineElem() {
		cvs.FLine(elem.Enod[0].Pcoord[0], elem.Enod[0].Pcoord[1], elem.Enod[1].Pcoord[0], elem.Enod[1].Pcoord[1])
		pd := elem.PDirection(true)
		if show.Bond {
			cvs.Foreground(BondColor)
			switch elem.BondState() {
			case st.PIN_RIGID:
				cvs.FCircle(elem.Enod[0].Pcoord[0]+pd[0]*show.BondSize, elem.Enod[0].Pcoord[1]+pd[1]*show.BondSize, show.BondSize*2)
			case st.RIGID_PIN:
				cvs.FCircle(elem.Enod[1].Pcoord[0]-pd[0]*show.BondSize, elem.Enod[1].Pcoord[1]-pd[1]*show.BondSize, show.BondSize*2)
			case st.PIN_PIN:
				cvs.FCircle(elem.Enod[0].Pcoord[0]+pd[0]*show.BondSize, elem.Enod[0].Pcoord[1]+pd[1]*show.BondSize, show.BondSize*2)
				cvs.FCircle(elem.Enod[1].Pcoord[0]-pd[0]*show.BondSize, elem.Enod[1].Pcoord[1]-pd[1]*show.BondSize, show.BondSize*2)
			}
		}
		if show.Phinge {
			cvs.InteriorStyle(cd.CD_SOLID)
			cvs.Foreground(BondColor)
			ph1 := elem.Phinge[show.Period][elem.Enod[0].Num]
			ph2 := elem.Phinge[show.Period][elem.Enod[1].Num]
			switch {
			case ph1 && !ph2:
				cvs.FFilledCircle(elem.Enod[0].Pcoord[0]+pd[0]*show.BondSize, elem.Enod[0].Pcoord[1]+pd[1]*show.BondSize, show.BondSize*2)
			case !ph1 && ph2:
				cvs.FFilledCircle(elem.Enod[1].Pcoord[0]-pd[0]*show.BondSize, elem.Enod[1].Pcoord[1]-pd[1]*show.BondSize, show.BondSize*2)
			case ph1 && ph2:
				cvs.FFilledCircle(elem.Enod[0].Pcoord[0]+pd[0]*show.BondSize, elem.Enod[0].Pcoord[1]+pd[1]*show.BondSize, show.BondSize*2)
				cvs.FFilledCircle(elem.Enod[1].Pcoord[0]-pd[0]*show.BondSize, elem.Enod[1].Pcoord[1]-pd[1]*show.BondSize, show.BondSize*2)
			}
		}
		if show.ElementAxis {
			DrawElementAxis(elem, cvs, show)
		}
		// Deformation
		if show.Deformation {
			cvs.LineStyle(cd.CD_DOTTED)
			cvs.FLine(elem.Enod[0].Dcoord[0], elem.Enod[0].Dcoord[1], elem.Enod[1].Dcoord[0], elem.Enod[1].Dcoord[1])
			cvs.LineStyle(cd.CD_CONTINUOUS)
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
						sttext[0].WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["STRESS"]), elem.ReturnStress(show.Period, 0, i)))
					case 1, 2, 3:
						sttext[0].WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["STRESS"]), elem.ReturnStress(show.Period, 0, i)))
						sttext[1].WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["STRESS"]), elem.ReturnStress(show.Period, 1, i)))
					case 4, 5:
						if !show.NoMomentValue {
							sttext[0].WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["STRESS"]), elem.ReturnStress(show.Period, 0, i)))
							sttext[1].WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["STRESS"]), elem.ReturnStress(show.Period, 1, i)))
						}
						mcoord := elem.MomentCoord(show, i)
						cvs.Foreground(MomentColor)
						cvs.Begin(cd.CD_OPEN_LINES)
						cvs.FVertex(elem.Enod[0].Pcoord[0], elem.Enod[0].Pcoord[1])
						for _, c := range mcoord {
							tmp := elem.Frame.View.ProjectCoord(c)
							cvs.FVertex(tmp[0], tmp[1])
						}
						cvs.FVertex(elem.Enod[1].Pcoord[0], elem.Enod[1].Pcoord[1])
						cvs.End()
					}
				}
			}
			cvs.Foreground(StressTextColor)
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
						cvs.TextAlignment(cd.CD_SOUTH)
					} else {
						cvs.TextAlignment(cd.CD_NORTH)
					}
					deg := math.Atan2(pd[1], pd[0]) * 180.0 / math.Pi
					if deg > 90.0 {
						deg -= 180.0
					} else if deg < -90.0 {
						deg += 180.0
					}
					cvs.TextOrientation(deg)
					cvs.FText(stpos[0], stpos[1], tex[:len(tex)-1])
					cvs.TextAlignment(DefaultTextAlignment)
					cvs.TextOrientation(0.0)
				}
			}
		}
		if show.YieldFunction {
			f, err := elem.YieldFunction(show.Period)
			for j := 0; j < 2; j++ {
				switch err[j].(type) {
				default:
					cvs.Foreground(StressTextColor)
				case st.YieldedError:
					cvs.Foreground(YieldedTextColor)
				case st.BrittleFailureError:
					cvs.Foreground(BrittleTextColor)
				}
				coord := make([]float64, 3)
				for i, en := range elem.Enod {
					for k := 0; k < 3; k++ {
						coord[k] += (-0.5*math.Abs(float64(i-j)) + 0.75) * en.Coord[k]
					}
				}
				stpos := elem.Frame.View.ProjectCoord(coord)
				// TODO: rotate text
				if j == 0 {
					cvs.TextAlignment(cd.CD_SOUTH)
				} else {
					cvs.TextAlignment(cd.CD_NORTH)
				}
				cvs.FText(stpos[0], stpos[1], fmt.Sprintf("%.3f", f[j]))
				cvs.TextAlignment(DefaultTextAlignment)
			}
		}
		if elem.Etype == st.WBRACE || elem.Etype == st.SBRACE {
			if elem.Eldest {
				if elem.Parent.Wrect != nil && (elem.Parent.Wrect[0] != 0.0 || elem.Parent.Wrect[1] != 0.0) {
					DrawWrect(elem.Parent, cvs, show)
				}
			}
		}
	} else {
		cvs.InteriorStyle(cd.CD_HATCH)
		cvs.Begin(cd.CD_FILL)
		cvs.FVertex(elem.Enod[0].Pcoord[0], elem.Enod[0].Pcoord[1])
		for i := 1; i < elem.Enods; i++ {
			cvs.FVertex(elem.Enod[i].Pcoord[0], elem.Enod[i].Pcoord[1])
		}
		cvs.End()
		cvs.Foreground(PlateEdgeColor)
		cvs.Begin(cd.CD_CLOSED_LINES)
		cvs.FVertex(elem.Enod[0].Pcoord[0], elem.Enod[0].Pcoord[1])
		for i := 1; i < elem.Enods; i++ {
			cvs.FVertex(elem.Enod[i].Pcoord[0], elem.Enod[i].Pcoord[1])
		}
		cvs.End()
		if elem.Wrect != nil && (elem.Wrect[0] != 0.0 || elem.Wrect[1] != 0.0) {
			DrawWrect(elem, cvs, show)
		}
		if show.ElemNormal {
			DrawElemNormal(elem, cvs, show)
		}
	}
}

func DrawWrect(elem *st.Elem, cvs *cd.Canvas, show *st.Show) {
	cvs.LineStyle(cd.CD_DOTTED)
	wrns := make([][]float64, 4)
	for i, n := range elem.WrectCoord() {
		wrns[i] = elem.Frame.View.ProjectCoord(n)
	}
	cvs.Begin(cd.CD_CLOSED_LINES)
	for i := 0; i < 4; i++ {
		cvs.FVertex(wrns[i][0], wrns[i][1])
	}
	cvs.End()
	cvs.LineStyle(cd.CD_CONTINUOUS)
}

func DrawElementAxis(elem *st.Elem, canv *cd.Canvas, show *st.Show) {
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
	angle := 10.0 * math.Pi / 180.0
	canv.LineStyle(cd.CD_CONTINUOUS)
	canv.Foreground(cd.CD_RED)
	Arrow(canv, origin[0], origin[1], xaxis[0], xaxis[1], arrow, angle)
	canv.Foreground(cd.CD_GREEN)
	Arrow(canv, origin[0], origin[1], yaxis[0], yaxis[1], arrow, angle)
	canv.Foreground(cd.CD_BLUE)
	Arrow(canv, origin[0], origin[1], zaxis[0], zaxis[1], arrow, angle)
	canv.Foreground(cd.CD_WHITE)
}

func DrawElemNormal(elem *st.Elem, canv *cd.Canvas, show *st.Show) {
	v := make([]float64, 3)
	mid := elem.MidPoint()
	d := elem.Normal(true)
	for i := 0; i < 3; i++ {
		v[i] = mid[i] + show.NodeNormalSize*d[i]
	}
	vec := elem.Frame.View.ProjectCoord(v)
	mp := elem.Frame.View.ProjectCoord(mid)
	arrow := 0.3
	angle := 10.0 * math.Pi / 180.0
	canv.LineStyle(cd.CD_CONTINUOUS)
	Arrow(canv, mp[0], mp[1], vec[0], vec[1], arrow, angle)
}

// KIJUN
func DrawKijun(k *st.Kijun, cvs *cd.Canvas, show *st.Show) {
	d := k.PDirection(true)
	if math.Abs(d[0]) <= 1e-6 && math.Abs(d[1]) <= 1e-6 {
		return
	}
	cvs.LineStyle(cd.CD_DASH_DOT)
	cvs.FLine(k.Pstart[0], k.Pstart[1], k.Pend[0], k.Pend[1])
	cvs.LineStyle(cd.CD_CONTINUOUS)
	cvs.FCircle(k.Pstart[0]-d[0]*show.KijunSize, k.Pstart[1]-d[1]*show.KijunSize, show.KijunSize*2)
	if k.Name[0] == '_' {
		cvs.FText(k.Pstart[0]-d[0]*show.KijunSize, k.Pstart[1]-d[1]*show.KijunSize, k.Name[1:])
	} else {
		cvs.FText(k.Pstart[0]-d[0]*show.KijunSize, k.Pstart[1]-d[1]*show.KijunSize, k.Name)
	}
}

// TEXT
func DrawText(t *TextBox, cvs *cd.Canvas) {
	s := cvs.SaveState()
	cvs.Font(t.Font.Face, cd.CD_PLAIN, t.Font.Size)
	cvs.Foreground(t.Font.Color)
	for i, txt := range t.Value {
		xpos := t.Position[0]
		ypos := t.Position[1] - float64(i*t.Font.Size)*1.5 - float64(t.Font.Size)
		cvs.FText(xpos, ypos, txt)
	}
	cvs.RestoreState(s)
}
