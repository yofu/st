package st

import (
	"bytes"
	"fmt"
	"github.com/ajstarks/svgo"
	"math"
	"os"
)

var (
	bondstyle  = "fill:none; stroke:black"
	confstyle  = "fill:black; stroke:none"
	dfstyle    = "stroke:black; stroke-dasharray:2,2"
	fontfamily = "font-family:IPAmincho"
	klstyle    = "stroke:black; stroke-dasharray:10, 5, 2, 5"
	kcstyle    = "fill:none; stroke:black"
	kfont      = "font-family:IPAmincho; text-anchor:middle; alignment-baseline:central"
)

func rgbString(color int) string {
	return "#fff"
}

func (stw *Window) Line(x1, y1, x2, y2 float64) {
	stw.currentCanvas.Line(int(x1), int(y1), int(x2), int(y2), stw.currentStyle.Stroke())
}

func (stw *Window) Polyline(coord [][]float64) {
	xs := make([]int, len(coord)
	ys := make([]int, len(coord)
	for i := 0; i< len(coord); i++ {
		xs[i] = int(coord[0])
		ys[i] = int(coord[1])
	}
	s := stw.currentStyle.Copy()
	s.Set("fill", "none")
	stw.currentCanvas.Polygon(xs, ys, s.Fill())
}

func (stw *Window) Polygon(coord [][]float64) {
	xs := make([]int, len(coord)
	ys := make([]int, len(coord)
	for i := 0; i< len(coord); i++ {
		xs[i] = int(coord[0])
		ys[i] = int(coord[1])
	}
	stw.currentCanvas.Polygon(xs, ys, stw.currentStyle.Fill())
}

func (stw *Window) Circle(x, y, r float64) {
	stw.currentCanvas.Circle(int(x), int(y), int(r), stw.currentStyle.Fill())
}

func (stw *Window) FilledCircle(x, y, r float64) {
}

func (stw *Window) Text(x, y float64, txt string) {
	stw.currentCanvas.Text(int(x), int(y), txt, stw.currentStyle.Text())
}

func (stw *Window) Foreground(fg int) {
	c := rgbString(fg)
	stw.currentStyle.Set("stroke", c)
	stw.currentStyle.Set("fill", c)
}

func (stw *Window) LineStyle(ls int) {
	switch ls {
	case CONTINUOUS:
		stw.currentStyle.Delete("stroke-dasharray")
	case DOTTED:
		stw.currentStyle.Set("stroke-dasharray", "2,2")
	case DASHED:
	case DASH_DOT:
		stw.currentStyle.Set("stroke-dasharray", "10,5,2,5")
	}
}

func (stw *Window) TextAlignment(ta int) {
}

func (stw *Window) TextOrientation(to float64) {
}

func (stw *Window) SectionAliase(s int) (string, bool) {
}

func (stw *Window) SelectedNodes() []*Node {
	return nil
}

func (stw *Window) SelectedElems() []*Elem {
	return nil
}

func (stw *Window) ElemSelected() bool {
	return false
}

func (stw *Window) DefaultStyle() {
	stw.currentStyle = NewStyle()
}

func (stw *Window) BondStyle(show *Show) {
	stw.currentStyle.Set("fill", "none")
	stw.currentStyle.Set("stroke", "black")
}

func (stw *Window) PhingeStyle(show *Show) {
	stw.currentStyle.Set("fill", "black")
	stw.currentStyle.Set("stroke", "black")
}

func (stw *Window) ConfStyle(show *Show) {
	stw.currentStyle.Set("fill", "black")
	stw.currentStyle.Set("stroke", "black")
}

func (stw *Window) SelectNodeStyle()
func (stw *Window) SelectElemStyle()
func (stw *Window) ShowPrintRange() bool
func (stw *Window) GetCanvasSize() (int, int)
func (stw *Window) CanvasPaperSize() (float64, float64, error)
func (stw *Window) Flush()
func (stw *Window) CanvasDirection() int

func Print(frame *Frame, otp string) error {
	s := frame.Show.Copy()
	v := frame.View.Copy()
	w, err := os.Create(otp)
	if err != nil {
		return err
	}
	defer w.Close()
	cvs := svg.New(w)
	cvs.Start(1000, 1000)
	w.WriteString(`<style>
    .ndcap {
        font-family: "IPAmincho";
        font-size: 9px;
    }
    .elcap {
        font-family: "IPAmincho";
        font-size: 9px;
    }
    .sttext {
        font-family: "IPAmincho";
        font-size: 9px;
    }
    .kijun {
        font-family: "IPAmincho";
        font-size: 9px;
        text-anchor: middle;
        alignment-baseline: central;
    }
</style>
`)
	err = PrintToCanvas(frame, cvs)
	cvs.End()
	frame.Show = s
	frame.View = v
	if err != nil {
		return err
	}
	return nil
}

func PrintToCanvas(frame *Frame, cvs *svg.SVG) error {
	frame.View.Center[0] = 500
	frame.View.Center[1] = 500
	frame.View.Set(1)
	// if frame.Show.GlobalAxis {
	// DrawGlobalAxis(cvs)
	// }
	if frame.Show.Kijun {
		for _, k := range frame.Kijuns {
			if k.IsHidden(frame.Show) {
				continue
			}
			k.Pstart = frame.View.ProjectCoord(k.Start)
			k.Pend = frame.View.ProjectCoord(k.End)
			DrawKijun(k, cvs, frame.Show)
		}
	}
	for _, n := range frame.Nodes {
		frame.View.ProjectNode(n)
		if frame.Show.Deformation {
			frame.View.ProjectDeformation(n, frame.Show)
		}
		if n.IsHidden(frame.Show) {
			continue
		}
		DrawNode(n, cvs, frame.Show)
	}
	els := SortedElem(frame.Elems, func(e *Elem) float64 { return -e.DistFromProjection(frame.View) })
	for _, el := range els {
		if el.IsHidden(frame.Show) {
			continue
		}
		DrawElem(el, cvs, frame.Show)
	}
	return nil
}

// NODE
func DrawNode(node *Node, cvs *svg.SVG, show *Show) {
	// Caption
	var ncap bytes.Buffer
	var oncap bool
	if show.NodeCaption&NC_NUM != 0 {
		ncap.WriteString(fmt.Sprintf("%d\n", node.Num))
		oncap = true
	}
	for i, j := range []uint{NC_DX, NC_DY, NC_DZ} {
		if show.NodeCaption&j != 0 {
			if !node.Conf[i] {
				ncap.WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["DISP"]), node.ReturnDisp(show.Period, i)*100.0))
				oncap = true
			}
		}
	}
	for i, j := range []uint{NC_RX, NC_RY, NC_RZ} {
		if show.NodeCaption&j != 0 {
			if node.Conf[i] {
				ncap.WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["REACTION"]), node.ReturnReaction(show.Period, i)))
				oncap = true
			}
		}
	}
	if show.NodeCaption&NC_ZCOORD != 0 {
		ncap.WriteString(fmt.Sprintf("%.1f\n", node.Coord[2]))
		oncap = true
	}
	if oncap {
		cvs.Text(int(node.Pcoord[0]), int(node.Pcoord[1]), ncap.String(), "class=\"ndcap\"")
	}
	// Conffigure
	if show.Conf {
		switch node.ConfState() {
		default:
			return
		case CONF_PIN:
			PinFigure(cvs, node.Pcoord[0], node.Pcoord[1], show.ConfSize)
		case CONF_XROL, CONF_YROL, CONF_XYROL:
			RollerFigure(cvs, node.Pcoord[0], node.Pcoord[1], show.ConfSize, 0)
		case CONF_ZROL:
			RollerFigure(cvs, node.Pcoord[0], node.Pcoord[1], show.ConfSize, 1)
		case CONF_FIX:
			FixFigure(cvs, node.Pcoord[0], node.Pcoord[1], show.ConfSize)
		}
	}
}

func PinFigure(cvs *svg.SVG, x, y, size float64) {
	xs := make([]int, 3)
	ys := make([]int, 3)
	val := y + 0.5*math.Sqrt(3)*size
	xs[0] = int(x)
	ys[0] = int(y)
	xs[1] = int(x + 0.5*size)
	ys[1] = int(val)
	xs[2] = int(x - 0.5*size)
	ys[2] = int(val)
	cvs.Polygon(xs, ys, confstyle)
}

func RollerFigure(cvs *svg.SVG, x, y, size float64, direction int) {
	xs := make([]int, 3)
	ys := make([]int, 3)
	xs[0] = int(x)
	ys[0] = int(y)
	switch direction {
	case 0:
		val1 := y + 0.5*math.Sqrt(3)*size
		val2 := y + 0.75*math.Sqrt(3)*size
		xs[1] = int(x + 0.5*size)
		ys[1] = int(val1)
		xs[2] = int(x - 0.5*size)
		ys[2] = int(val1)
		cvs.Polygon(xs, ys, confstyle)
		cvs.Line(xs[2], int(val2), xs[1], int(val2))
	case 1:
		val1 := x - 0.5*math.Sqrt(3)*size
		val2 := x - 0.75*math.Sqrt(3)*size
		xs[1] = int(val1)
		ys[1] = int(y + 0.5*size)
		xs[2] = int(val1)
		ys[2] = int(y - 0.5*size)
		cvs.Polygon(xs, ys, confstyle)
		cvs.Line(int(val2), ys[2], int(val2), ys[1])
	}
}

func FixFigure(cvs *svg.SVG, x, y, size float64) {
	cvs.Line(int(x-size), int(y), int(x+size), int(y))
	cvs.Line(int(x-0.25*size), int(y), int(x-0.75*size), int(y+0.5*size))
	cvs.Line(int(x+0.25*size), int(y), int(x-0.25*size), int(y+0.5*size))
	cvs.Line(int(x+0.75*size), int(y), int(x+0.25*size), int(y+0.5*size))
}

// ELEM
func DrawElem(elem *Elem, cvs *svg.SVG, show *Show) {
	var ecap bytes.Buffer
	var oncap bool
	if show.ElemCaption&EC_NUM != 0 {
		ecap.WriteString(fmt.Sprintf("%d\n", elem.Num))
		oncap = true
	}
	if show.ElemCaption&EC_SECT != 0 {
		ecap.WriteString(fmt.Sprintf("%d\n", elem.Sect.Num))
		oncap = true
	}
	if show.ElemCaption&EC_RATE_L != 0 || show.ElemCaption&EC_RATE_S != 0 {
		val, err := elem.RateMax(show)
		if err == nil {
			ecap.WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["RATE"]), val))
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
		cvs.Text(int(textpos[0]), int(textpos[1]), ecap.String(), "class=\"elcap\"")
	}
	if elem.IsLineElem() {
		var lc string
		switch show.ColorMode {
		default:
			lc = "stroke:black"
		case ECOLOR_SECT:
			lc = fmt.Sprintf("stroke:%s", IntHexColor(elem.Sect.Color))
		case ECOLOR_RATE:
			val, err := elem.RateMax(show)
			if err != nil {
				lc = "stroke:darkgrey"
			} else {
				lc = fmt.Sprintf("stroke:%s", IntHexColor(Rainbow(val, RateBoundary)))
			}
		}
		cvs.Line(int(elem.Enod[0].Pcoord[0]), int(elem.Enod[0].Pcoord[1]), int(elem.Enod[1].Pcoord[0]), int(elem.Enod[1].Pcoord[1]), lc)
		if show.Bond {
			switch elem.BondState() {
			case PIN_RIGID:
				d := elem.PDirection(true)
				cvs.Circle(int(elem.Enod[0].Pcoord[0]+d[0]*show.BondSize), int(elem.Enod[0].Pcoord[1]+d[1]*show.BondSize), int(show.BondSize), bondstyle)
			case RIGID_PIN:
				d := elem.PDirection(true)
				cvs.Circle(int(elem.Enod[1].Pcoord[0]-d[0]*show.BondSize), int(elem.Enod[1].Pcoord[1]-d[1]*show.BondSize), int(show.BondSize), bondstyle)
			case PIN_PIN:
				d := elem.PDirection(true)
				cvs.Circle(int(elem.Enod[0].Pcoord[0]+d[0]*show.BondSize), int(elem.Enod[0].Pcoord[1]+d[1]*show.BondSize), int(show.BondSize), bondstyle)
				cvs.Circle(int(elem.Enod[1].Pcoord[0]-d[0]*show.BondSize), int(elem.Enod[1].Pcoord[1]-d[1]*show.BondSize), int(show.BondSize), bondstyle)
			}
		}
		// if show.ElementAxis {
		//     DrawElementAxis(elem, cvs, show)
		// }
		// Deformation
		if show.Deformation {
			cvs.Line(int(elem.Enod[0].Dcoord[0]), int(elem.Enod[0].Dcoord[1]), int(elem.Enod[1].Dcoord[0]), int(elem.Enod[1].Dcoord[1]), dfstyle)
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
					sttext[0].WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["STRESS"]), elem.ReturnStress(show.Period, 0, i)))
					if i != 0 { // not showing NZ
						sttext[1].WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["STRESS"]), elem.ReturnStress(show.Period, 1, i)))
						if i == 4 || i == 5 {
							mcoord := elem.MomentCoord(show, i)
							mcsize := len(mcoord) + 2
							xs := make([]int, mcsize)
							ys := make([]int, mcsize)
							xs[0] = int(elem.Enod[0].Pcoord[0])
							ys[0] = int(elem.Enod[0].Pcoord[1])
							xs[mcsize-1] = int(elem.Enod[1].Pcoord[0])
							ys[mcsize-1] = int(elem.Enod[1].Pcoord[1])
							for i, c := range mcoord {
								tmp := elem.Frame.View.ProjectCoord(c)
								xs[i+1] = int(tmp[0])
								ys[i+1] = int(tmp[1])
							}
							cvs.Polyline(xs, ys)
						}
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
					// TODO: rotate text
					// if j == 0 {
					//     cvs.TextAlignment(cd.CD_SOUTH)
					// } else {
					//     cvs.TextAlignment(cd.CD_NORTH)
					// }
					if j == 0 {
						cvs.Text(int(stpos[0]), int(stpos[1]), tex[:len(tex)-1], "class=\"sttext_0\"")
					} else {
						cvs.Text(int(stpos[0]), int(stpos[1]), tex[:len(tex)-1], "class=\"sttext_1\"")
					}
				}
			}
		}
	} else {
		xs := make([]int, elem.Enods)
		ys := make([]int, elem.Enods)
		for i, n := range elem.Enod {
			xs[i] = int(n.Pcoord[0])
			ys[i] = int(n.Pcoord[1])
		}
		cvs.Polygon(xs, ys, "fill:none;stroke:black")
		// if show.ElemNormal {
		//     DrawElemNormal(elem, cvs, show)
		// }
	}
}

func DrawKijun(k *Kijun, cvs *svg.SVG, show *Show) {
	d := k.PDirection(true)
	if math.Abs(d[0]) <= 1e-6 && math.Abs(d[1]) <= 1e-6 {
		return
	}
	cvs.Line(int(k.Pstart[0]), int(k.Pstart[1]), int(k.Pend[0]), int(k.Pend[1]), klstyle)
	cvs.Circle(int(k.Pstart[0]-d[0]*show.KijunSize), int(k.Pstart[1]-d[1]*show.KijunSize), int(show.KijunSize), kcstyle)
	if k.Name[0] == '_' {
		cvs.Text(int(k.Pstart[0]-d[0]*show.KijunSize), int(k.Pstart[1]-d[1]*show.KijunSize), k.Name[1:], "class=\"kijun\"")
	} else {
		cvs.Text(int(k.Pstart[0]-d[0]*show.KijunSize), int(k.Pstart[1]-d[1]*show.KijunSize), k.Name, "class=\"kijun\"")
	}
}
