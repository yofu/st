package stgui

import (
    "bytes"
    "fmt"
    "github.com/visualfc/go-iup/cd"
    "github.com/yofu/st/stlib"
    "math"
)

// FRAME
func ProjectDeformation (view *st.View, node *st.Node, show *Show) {
    p  := make([]float64, 3)
    pv := make([]float64, 3)
    pc := make([]float64, 2)
    for i:=0; i<3; i++ {
        p[i] = (node.Coord[i] + show.Dfact * node.ReturnDisp(show.Period, i)) - view.Focus[i]
        pv[i] = view.Viewpoint[0][i]*view.Dists[0] - p[i]
    }
    for i:=0; i<2; i++ {
        pc[i] = st.Dot(view.Viewpoint[i+1], p, 3)
    }
    if view.Perspective {
        vnai := st.Dot(view.Viewpoint[0], pv, 3)
        for i:=0; i<2; i++ {
            node.Dcoord[i] = view.Gfact*view.Dists[1]*pc[i]/vnai + view.Center[i]
        }
    } else {
        for i:=0; i<2; i++ {
            node.Dcoord[i] = view.Gfact*pc[i] + view.Center[i]
        }
    }
}


// NODE
func DrawNode (node *st.Node, cvs *cd.Canvas, show *Show) {
    // Caption
    var ncap bytes.Buffer
    var oncap bool
    if show.NodeCaption & NC_NUM != 0 {
        ncap.WriteString(fmt.Sprintf("%d\n", node.Num))
        oncap = true
    }
    for i, j := range []uint{NC_DX, NC_DY, NC_DZ} {
        if show.NodeCaption & j != 0 {
            if !node.Conf[i] {
                ncap.WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["DISP"]), node.ReturnDisp(show.Period, i)*100.0))
                oncap = true
            }
        }
    }
    for i, j := range []uint{NC_RX, NC_RY, NC_RZ} {
        if show.NodeCaption & j != 0 {
            if node.Conf[i] {
                ncap.WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["REACTION"]), node.ReturnReaction(show.Period, i)))
                oncap = true
            }
        }
    }
    if show.NodeCaption & NC_ZCOORD != 0 {
        ncap.WriteString(fmt.Sprintf("%.1f\n", node.Coord[2]))
        oncap = true
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
        cvs.Foreground(show.ConfColor)
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

func PinFigure (cvs *cd.Canvas, x, y , size float64) {
    val := y - 0.5*math.Sqrt(3)*size
    cvs.Begin(cd.CD_FILL)
    cvs.FVertex(x, y)
    cvs.FVertex(x+0.5*size, val)
    cvs.FVertex(x-0.5*size, val)
    cvs.End()
}

func RollerFigure (cvs *cd.Canvas, x, y , size float64, direction int) {
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

func FixFigure (cvs *cd.Canvas, x, y , size float64) {
    cvs.FLine(x-size, y, x+size, y)
    cvs.FLine(x-0.25*size, y, x-0.75*size, y-0.5*size)
    cvs.FLine(x+0.25*size, y, x-0.25*size, y-0.5*size)
    cvs.FLine(x+0.75*size, y, x+0.25*size, y-0.5*size)
}

func  DrawNodeNormal (node *st.Node, canv *cd.Canvas, show *Show) {
    v := make([]float64, 3)
    d := node.Normal(true)
    for i:=0; i<3; i++ {
        v[i] = node.Coord[i] + show.NodeNormalSize*d[i]
    }
    vec  := node.Frame.View.ProjectCoord(v)
    arrow := 0.3
    angle := 10.0*math.Pi/180.0
    canv.LineStyle(cd.CD_CONTINUOUS)
    Arrow(canv, node.Pcoord[0], node.Pcoord[1], vec[0], vec[1], arrow, angle)
}


// ELEM
func DrawElem (elem *st.Elem, cvs *cd.Canvas, show *Show) {
    var ecap bytes.Buffer
    var oncap bool
    if show.ElemCaption & EC_NUM != 0 {
        ecap.WriteString(fmt.Sprintf("%d\n", elem.Num))
        oncap = true
    }
    if show.ElemCaption & EC_SECT != 0 {
        ecap.WriteString(fmt.Sprintf("%d\n", elem.Sect.Num))
        oncap = true
    }
    if elem.IsLineElem() {
        if show.ElemCaption & EC_RATE_L != 0 || show.ElemCaption & EC_RATE_S != 0 {
            val := RateMax(elem, show)
            ecap.WriteString(fmt.Sprintf(fmt.Sprintf("%s\n",show.Formats["RATE"]),val))
            oncap = true
        }
    }
    if oncap {
        var textpos []float64
        if st.BRACE <= elem.Etype && elem.Etype <= st.SBRACE {
            coord := make([]float64, 3)
            for j, en := range elem.Enod {
                for k:=0; k<3; k++ {
                    coord[k] += (-0.5*float64(j)+0.75)*en.Coord[k]
                }
            }
            textpos = elem.Frame.View.ProjectCoord(coord)
        } else {
            textpos = make([]float64, 2)
            for _, en := range elem.Enod {
                for i:=0; i<2; i++ {
                    textpos[i] += en.Pcoord[i]
                }
            }
            for i:=0; i<2; i++ {
                textpos[i] /= float64(elem.Enods)
            }
        }
        cvs.FText(textpos[0], textpos[1], ecap.String())
    }
    if elem.IsLineElem() {
        cvs.FLine(elem.Enod[0].Pcoord[0], elem.Enod[0].Pcoord[1], elem.Enod[1].Pcoord[0], elem.Enod[1].Pcoord[1])
        if show.Bond {
            cvs.Foreground(show.BondColor)
            switch elem.BondState() {
            case st.PIN_RIGID:
                d := elem.PDirection(true)
                cvs.FCircle(elem.Enod[0].Pcoord[0]+d[0]*show.BondSize, elem.Enod[0].Pcoord[1]+d[1]*show.BondSize, show.BondSize*2)
            case st.RIGID_PIN:
                d := elem.PDirection(true)
                cvs.FCircle(elem.Enod[1].Pcoord[0]-d[0]*show.BondSize, elem.Enod[1].Pcoord[1]-d[1]*show.BondSize, show.BondSize*2)
            case st.PIN_PIN:
                d := elem.PDirection(true)
                cvs.FCircle(elem.Enod[0].Pcoord[0]+d[0]*show.BondSize, elem.Enod[0].Pcoord[1]+d[1]*show.BondSize, show.BondSize*2)
                cvs.FCircle(elem.Enod[1].Pcoord[0]-d[0]*show.BondSize, elem.Enod[1].Pcoord[1]-d[1]*show.BondSize, show.BondSize*2)
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
            for i, st := range []uint{ st.STRESS_NZ, st.STRESS_QX, st.STRESS_QY, st.STRESS_MZ, st.STRESS_MX, st.STRESS_MY } {
                if flag & st != 0 {
                    sttext[0].WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["STRESS"]), elem.ReturnStress(show.Period, 0, i)))
                    if i != 0 { // not showing NZ
                        sttext[1].WriteString(fmt.Sprintf(fmt.Sprintf("%s\n", show.Formats["STRESS"]), elem.ReturnStress(show.Period, 1, i)))
                        if i == 4 || i == 5 {
                            mcoord := MomentCoord(elem, show, i)
                            cvs.Foreground(show.MomentColor)
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
            }
            cvs.Foreground(show.StressTextColor)
            for j:=0; j<2; j++ {
                if tex := sttext[j].String(); tex != "" {
                    coord := make([]float64, 3)
                    for i, en := range elem.Enod {
                        for k:=0; k<3; k++ {
                            coord[k] += (-0.5*math.Abs(float64(i-j))+0.75)*en.Coord[k]
                        }
                    }
                    stpos := elem.Frame.View.ProjectCoord(coord)
                    // TODO: rotate text
                    if j == 0 {
                        cvs.TextAlignment(cd.CD_SOUTH)
                    } else {
                        cvs.TextAlignment(cd.CD_NORTH)
                    }
                    cvs.FText(stpos[0], stpos[1], tex[:len(tex)-1])
                    cvs.TextAlignment(DefaultTextAlignment)
                }
            }
        }
    } else {
        cvs.Begin(cd.CD_FILL)
        cvs.FVertex(elem.Enod[0].Pcoord[0], elem.Enod[0].Pcoord[1])
        for i:=1; i<elem.Enods; i++ {
            cvs.FVertex(elem.Enod[i].Pcoord[0], elem.Enod[i].Pcoord[1])
        }
        cvs.End()
        cvs.Foreground(show.PlateEdgeColor)
        cvs.Begin(cd.CD_CLOSED_LINES)
        cvs.FVertex(elem.Enod[0].Pcoord[0], elem.Enod[0].Pcoord[1])
        for i:=1; i<elem.Enods; i++ {
            cvs.FVertex(elem.Enod[i].Pcoord[0], elem.Enod[i].Pcoord[1])
        }
        cvs.End()
        if show.ElemNormal {
            DrawElemNormal(elem, cvs, show)
        }
    }
}

func DrawElementAxis (elem *st.Elem, canv *cd.Canvas, show *Show) {
    if !elem.IsLineElem() { return }
    x := make([]float64, 3)
    y := make([]float64, 3)
    z := make([]float64, 3)
    position := elem.MidPoint()
    d := elem.Direction(true)
    for i:=0; i<3; i++ {
        x[i] = position[i] + show.ElementAxisSize*elem.Strong[i]
        y[i] = position[i] + show.ElementAxisSize*elem.Weak[i]
        z[i] = position[i] + show.ElementAxisSize*d[i]
    }
    origin := elem.Frame.View.ProjectCoord(position)
    xaxis  := elem.Frame.View.ProjectCoord(x)
    yaxis  := elem.Frame.View.ProjectCoord(y)
    zaxis  := elem.Frame.View.ProjectCoord(z)
    arrow := 0.3
    angle := 10.0*math.Pi/180.0
    canv.LineStyle(cd.CD_CONTINUOUS)
    canv.Foreground(cd.CD_RED)
    Arrow(canv, origin[0], origin[1], xaxis[0], xaxis[1], arrow, angle)
    canv.Foreground(cd.CD_GREEN)
    Arrow(canv, origin[0], origin[1], yaxis[0], yaxis[1], arrow, angle)
    canv.Foreground(cd.CD_BLUE)
    Arrow(canv, origin[0], origin[1], zaxis[0], zaxis[1], arrow, angle)
    canv.Foreground(cd.CD_WHITE)
}

func DrawElemNormal (elem *st.Elem, canv *cd.Canvas, show *Show) {
    v := make([]float64, 3)
    mid := elem.MidPoint()
    d := elem.Normal(true)
    for i:=0; i<3; i++ {
        v[i] = mid[i] + show.NodeNormalSize*d[i]
    }
    vec  := elem.Frame.View.ProjectCoord(v)
    mp := elem.Frame.View.ProjectCoord(mid)
    arrow := 0.3
    angle := 10.0*math.Pi/180.0
    canv.LineStyle(cd.CD_CONTINUOUS)
    Arrow(canv, mp[0], mp[1], vec[0], vec[1], arrow, angle)
}

func MomentCoord (elem *st.Elem, show *Show, index int) [][]float64 {
    var axis []float64
    if index == 4 {
        axis = elem.Weak
    } else if index == 5 {
        axis = make([]float64, 3)
        for i:=0; i<3; i++ {
            axis[i] = -elem.Strong[i]
        }
    } else {
        return nil
    }
    ms := make([]float64, 2)
    qs := make([]float64, 2)
    for i:=0; i<2; i++ {
        ms[i] = elem.ReturnStress(show.Period, i, index)
        qs[i] = elem.ReturnStress(show.Period, i, 6-index)
    }
    l := elem.Length()
    rtn := make([][]float64, 3)
    rtn[0] = make([]float64, 3)
    for i:=0; i<3; i++ {
        rtn[0][i] = -show.Mfact * axis[i] * ms[0] + elem.Enod[0].Coord[i]
    }
    rtn[2] = make([]float64, 3)
    for i:=0; i<3; i++ {
        rtn[2][i] = show.Mfact * axis[i] * ms[1] + elem.Enod[1].Coord[i]
    }
    if math.Abs(qs[0]+qs[1]) > 1.0 {
        tmp := make([]float64, 3)
        d := elem.Direction(true)
        val := (qs[1]*l - (ms[0]+ms[1])) / (qs[0]+qs[1])
        for i:=0; i<3; i++ {
            tmp[i] = elem.Enod[0].Coord[i] + d[i]*val
        }
        ll := 0.0
        for i:=0; i<3; i++ {
            ll += math.Pow(elem.Enod[1].Coord[i] - tmp[i], 2.0)
        }
        ll = math.Sqrt(ll)
        if 0.0 < ll && ll < l {
            rtn[1] = make([]float64, 3)
            val := (qs[0]*ms[1] - qs[1]*ms[0] - qs[0]*qs[1]*l) / (qs[0]+qs[1])
            for i:=0; i<3; i++ {
                rtn[1][i] = tmp[i] + show.Mfact * axis[i] * val
            }
        } else {
            return [][]float64{rtn[0], rtn[2]}
        }
        return rtn
    } else {
        return [][]float64{rtn[0], rtn[2]}
    }
}

func RateMax(elem *st.Elem, show *Show) float64 {
     if len(elem.Rate)%3 != 0 || (show.ElemCaption & EC_RATE_L == 0 && show.ElemCaption & EC_RATE_S == 0) {
        val := 0.0
        for _, tmp := range elem.Rate {
            if tmp > val { val = tmp }
        }
        return val
    } else {
        vall := 0.0
        vals := 0.0
        for i, tmp := range elem.Rate {
            switch i%3 {
            default:
                continue
            case 0:
                if tmp > vall { vall = tmp }
            case 1:
                if tmp > vals { vals = tmp }
            }
        }
        if show.ElemCaption & EC_RATE_L == 0 { vall = 0.0 }
        if show.ElemCaption & EC_RATE_S == 0 { vals = 0.0 }
        if vall >= vals {
            return vall
        } else {
            return vals
        }
    }
}


// KIJUN
func DrawKijun (k *st.Kijun, cvs *cd.Canvas, show *Show) {
    d := k.PDirection(true)
    if (math.Abs(d[0]) <= 1e-6 && math.Abs(d[1]) <= 1e-6) { return }
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
