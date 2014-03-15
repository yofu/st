package stpdf

import (
    "bytes"
    "fmt"
    "github.com/ajstarks/svgo"
    "github.com/yofu/st/stlib"
    "math"
    "os"
)

const (
    bondstyle = "fill:none; stroke:black"
    dfstyle = "stroke:black; stroke-dasharray:2,2"
)

func Print (frame *st.Frame, inp, otp string) error {
    w, err := os.Create(otp)
    if err != nil {
        return err
    }
    cvs := svg.New(w)
    cvs.Start(1000, 1000)
    frame.View.Set(1)
    // if frame.Show.GlobalAxis {
        // DrawGlobalAxis(cvs)
    // }
    // if frame.Show.Kijun {
    //     canv.TextAlignment(cd.CD_CENTER)
    //     canv.Foreground(cd.CD_GRAY)
    //     for _, k := range frame.Kijuns {
    //         if k.Hide { continue }
    //         k.Pstart = frame.View.ProjectCoord(k.Start)
    //         k.Pend   = frame.View.ProjectCoord(k.End)
    //         DrawKijun(k, canv, frame.Show)
    //     }
    //     canv.TextAlignment(DefaultTextAlignment)
    // }
    // canv.Foreground(cd.CD_WHITE)
    for _, n := range frame.Nodes {
        frame.View.ProjectNode(n)
        if frame.Show.Deformation { frame.View.ProjectDeformation(n, frame.Show) }
        if n.Hide { continue }
        // if n.Lock {
        //     canv.Foreground(LOCKED_NODE_COLOR)
        // } else {
        //     canv.Foreground(canvasFontColor)
        // }
        // switch n.ConfState() {
        // case st.CONF_PIN:
        //     canv.Foreground(cd.CD_GREEN)
        // case st.CONF_FIX:
        //     canv.Foreground(cd.CD_DARK_GREEN)
        // }
        // DrawNode(n, canv, frame.Show)
    }
    // canv.LineStyle(cd.CD_CONTINUOUS)
    // canv.Hatch(cd.CD_FDIAGONAL)
    els := st.SortedElem(frame.Elems, func (e *st.Elem) float64 { return -e.DistFromProjection() })
    for _, el := range(els) {
        if el.Hide { continue }
        if !frame.Show.Etype[el.Etype] { el.Hide = true; continue }
        if b, ok := frame.Show.Sect[el.Sect.Num]; ok {
            if !b { el.Hide = true; continue }
        }
        // canv.LineStyle(cd.CD_CONTINUOUS)
        // canv.Hatch(cd.CD_FDIAGONAL)
        // if el.Lock {
        //     canv.Foreground(LOCKED_ELEM_COLOR)
        // } else {
        //     switch color {
        //     case st.ECOLOR_WHITE:
        //         canv.Foreground(cd.CD_WHITE)
        //     case st.ECOLOR_BLACK:
        //         canv.Foreground(cd.CD_BLACK)
        //     case st.ECOLOR_SECT:
        //         canv.Foreground(el.Sect.Color)
        //     case st.ECOLOR_RATE:
        //         canv.Foreground(st.Rainbow(el.RateMax(frame.Show), st.RateBoundary))
        //     case st.ECOLOR_HEIGHT:
        //         canv.Foreground(st.Rainbow(el.MidPoint()[2], st.HeightBoundary))
        //     }
        // }
        DrawElem(el, cvs, frame.Show)
    }
    cvs.End()
    return nil
}

func DrawElem (elem *st.Elem, cvs *svg.SVG, show *st.Show) {
    var ecap bytes.Buffer
    var oncap bool
    if show.ElemCaption & st.EC_NUM != 0 {
        ecap.WriteString(fmt.Sprintf("%d\n", elem.Num))
        oncap = true
    }
    if show.ElemCaption & st.EC_SECT != 0 {
        ecap.WriteString(fmt.Sprintf("%d\n", elem.Sect.Num))
        oncap = true
    }
    if show.ElemCaption & st.EC_RATE_L != 0 || show.ElemCaption & st.EC_RATE_S != 0 {
        val := elem.RateMax(show)
        ecap.WriteString(fmt.Sprintf(fmt.Sprintf("%s\n",show.Formats["RATE"]),val))
        oncap = true
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
        cvs.Text(int(textpos[0]), int(textpos[1]), ecap.String())
    }
    if elem.IsLineElem() {
        cvs.Line(int(elem.Enod[0].Pcoord[0]), int(elem.Enod[0].Pcoord[1]), int(elem.Enod[1].Pcoord[0]), int(elem.Enod[1].Pcoord[1]), "stroke:black")
        if show.Bond {
            // cvs.Foreground(BondColor)
            switch elem.BondState() {
            case st.PIN_RIGID:
                d := elem.PDirection(true)
                cvs.Circle(int(elem.Enod[0].Pcoord[0]+d[0]*show.BondSize*0.5), int(elem.Enod[0].Pcoord[1]+d[1]*show.BondSize*0.5), int(show.BondSize), bondstyle)
            case st.RIGID_PIN:
                d := elem.PDirection(true)
                cvs.Circle(int(elem.Enod[1].Pcoord[0]-d[0]*show.BondSize*0.5), int(elem.Enod[1].Pcoord[1]-d[1]*show.BondSize*0.5), int(show.BondSize), bondstyle)
            case st.PIN_PIN:
                d := elem.PDirection(true)
                cvs.Circle(int(elem.Enod[0].Pcoord[0]+d[0]*show.BondSize*0.5), int(elem.Enod[0].Pcoord[1]+d[1]*show.BondSize*0.5), int(show.BondSize), bondstyle)
                cvs.Circle(int(elem.Enod[1].Pcoord[0]-d[0]*show.BondSize*0.5), int(elem.Enod[1].Pcoord[1]-d[1]*show.BondSize*0.5), int(show.BondSize), bondstyle)
            }
        }
        // if show.ElementAxis {
        //     DrawElementAxis(elem, cvs, show)
        // }
        // Deformation
        if show.Deformation {
            // cvs.LineStyle(cd.CD_DOTTED)
            cvs.Line(int(elem.Enod[0].Dcoord[0]), int(elem.Enod[0].Dcoord[1]), int(elem.Enod[1].Dcoord[0]), int(elem.Enod[1].Dcoord[1]), dfstyle)
            // cvs.LineStyle(cd.CD_CONTINUOUS)
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
                                xs[i+1] = int(tmp[0]); ys[i+1] = int(tmp[1])
                            }
                            cvs.Polyline(xs, ys)
                        }
                    }
                }
            }
            // cvs.Foreground(StressTextColor)
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
                    // if j == 0 {
                    //     cvs.TextAlignment(cd.CD_SOUTH)
                    // } else {
                    //     cvs.TextAlignment(cd.CD_NORTH)
                    // }
                    cvs.Text(int(stpos[0]), int(stpos[1]), tex[:len(tex)-1])
                    // cvs.TextAlignment(DefaultTextAlignment)
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
