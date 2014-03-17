package st

import (
    "bytes"
    "errors"
    "fmt"
    "math"
    "sort"
    "strings"
)


// Constants & Variables// {{{
var ETYPES = [8]string{"NONE", "COLUMN", "GIRDER", "BRACE", "WBRACE", "SBRACE", "WALL", "SLAB"}
var StressName = [6]string{"Nz", "Qx", "Qy", "Mz", "Mx", "My"}
const (
    NONE = iota
    COLUMN
    GIRDER
    BRACE
    WBRACE
    SBRACE
    WALL
    SLAB
)

const (
    STRESS_NZ = 1 << iota
    STRESS_QX
    STRESS_QY
    STRESS_MZ
    STRESS_MX
    STRESS_MY
)

const (
    RIGID_RIGID = iota
    PIN_RIGID
    RIGID_PIN
    PIN_PIN
)
var (
    RIGID = []bool{ false, false, false, false, false, false }
    PIN   = []bool{ false, false, false, false, true , true  }
)
// }}}


// type Elem// {{{
type Elem struct {
    Frame *Frame
    Num int
    Enods int
    Enod []*Node
    Sect *Sect
    Etype int
    Cang float64
    Bonds []bool
    Cmq  []float64
    Wrect []float64

    Rate []float64
    Stress map[string]map[int][]float64

    Strong []float64
    Weak []float64

    Children []*Elem
    Parent *Elem

    Hide bool
    Lock bool
}

func NewLineElem (ns []*Node, sect *Sect, etype int) *Elem {
    if etype >= WALL { return nil }
    el := new(Elem)
    el.Enods = 2
    el.Enod  = ns[:2]
    el.Sect  = sect
    el.Etype = etype
    el.Bonds = make([]bool, 12)
    el.Cmq   = make([]float64, 12)
    el.Stress = make(map[string]map[int][]float64)
    el.Strong = make([]float64, 3)
    el.Weak = make([]float64, 3)
    el.SetPrincipalAxis()
    return el
}

func NewPlateElem (ns []*Node, sect *Sect, etype int) *Elem {
    if COLUMN <= etype && etype <= SBRACE { return nil }
    el := new(Elem)
    for _, en := range ns {
        if en != nil {
            el.Enods++
        } else {
            break
        }
    }
    if el.Enods < 3 { return nil }
    el.Enod = ns[:el.Enods]
    el.Sect = sect
    el.Etype = etype
    el.Children = make([]*Elem, 2)
    el.Wrect = make([]float64, 2)
    return el
}
// }}}

func (elem *Elem) PrincipalAxis (cang float64) ([]float64, []float64) {
    d := elem.Direction(true)
    c := math.Cos(cang)
    s := math.Sin(cang)
    strong := make([]float64, 3)
    weak := make([]float64, 3)
    if d[0] == 0.0 && d[1] == 0.0 {
        strong = []float64{ -s*d[2],  c, 0.0 }
        weak   = []float64{ -c*d[2], -s, 0.0 }
    } else {
        x := Normalize([]float64{ -d[1], d[0], 0.0 })
        y := Cross(d, x)
        for i:=0; i<3; i++ {
            strong[i] =  c*x[i] + s*y[i]
            weak[i]   = -s*x[i] + c*y[i]
        }
    }
    return Normalize(strong), Normalize(weak)
}

func (elem *Elem) SetPrincipalAxis () {
    s, w := elem.PrincipalAxis(elem.Cang)
    elem.Strong = s
    elem.Weak = w
}

func (elem *Elem) AxisToCang (vector []float64, strong bool) float64 {
    vector = Normalize(vector)
    d := elem.Direction(true)
    uv := Dot(d, vector, 3)
    if uv == 1.0 {
        elem.Cang = 0.0
        elem.SetPrincipalAxis()
        return elem.Cang
    }
    newvec := make([]float64, 3)
    for i:=0; i<3; i++ {
        newvec[i] = vector[i] - uv*d[i]
    }
    newvec = Normalize(newvec)
    d1, d2 := elem.PrincipalAxis(0.0)
    c1 := Dot(d1, newvec, 3)
    c2 := Dot(d2, newvec, 3)
    if strong {
        if c2 >= 0.0 {
            if -1.0-1e-3 <= c1 && c1 <= -1.0 {
                elem.Cang = math.Pi
            } else if 1.0 <= c1 && c1 <= 1.0+1e-3 {
                elem.Cang = 0.0
            } else {
                elem.Cang = math.Acos(c1)
            }
        } else {
            if -1.0-1e-3 <= c1 && c1 <= -1.0 {
                elem.Cang = -math.Pi
            } else if 1.0 <= c1 && c1 <= 1.0+1e-3 {
                elem.Cang = 0.0
            } else {
                elem.Cang = -math.Acos(c1)
            }
        }
    } else {
        if c1 >= 0.0 {
            if -1.0-1e-3 <= c2 && c2 <= -1.0 {
                elem.Cang = -math.Pi
            } else if 1.0 <= c2 && c2 <= 1.0+1e-3 {
                elem.Cang = 0.0
            } else {
                elem.Cang = -math.Acos(c2)
            }
        } else {
            if -1.0-1e-3 <= c2 && c2 <= -1.0 {
                elem.Cang = math.Pi
            } else if 1.0 <= c2 && c2 <= 1.0+1e-3 {
                elem.Cang = 0.0
            } else {
                elem.Cang = math.Acos(c2)
            }
        }
    }
    elem.SetPrincipalAxis()
    return elem.Cang
}


// Sort// {{{
// type Elems []*Elem
// func (e Elems) Len() int { return len(e) }
// func (e Elems) Swap(i, j int) {
//     tmp := e[i]
//     e[i] = e[j]
//     e[j] = tmp
// }

// type ElemByNum struct { Elems }
// func (e ElemByNum) Less(i, j int) bool { return e.Elems[i].Num < e.Elems[j].Num }
// }}}
func SortedElem (els map[int]*Elem, compare func (*Elem) float64) []*Elem {
    l := len(els)
    elems := make(map[float64][]*Elem, l)
    keys := make([]float64, l)
    sortedelems := make([]*Elem, 0)
    for _, el := range els {
        val := compare(el)
        if _, ok := elems[val]; !ok {
            elems[val] = make([]*Elem, 1)
            elems[val][0] = el
        } else {
            elems[val] = append(elems[val], el)
        }
    }
    for k := range elems {
        keys = append(keys, k)
    }
    sort.Float64s(keys)
    for _, k := range keys {
        sortedelems = append(sortedelems, elems[k]...)
    }
    return sortedelems
}


// Etype// {{{
func Etype (str string) int {
    for i, j := range(ETYPES) {
        if j==str {
            return i
        }
    }
    return 0
}

func (elem *Elem) setEtype(str string) error {
    for i, j := range(ETYPES) {
        if j==str {
            elem.Etype = i
            return nil
        }
    }
    return errors.New("setEtype: Etype not found")
}

func (elem *Elem) IsLineElem () bool {
    return elem.Etype <= SBRACE && elem.Enods == 2
}
// }}}


// Write// {{{
func (elem *Elem) InpString () string {
    var rtn bytes.Buffer
    if elem.IsLineElem() {
        rtn.WriteString(fmt.Sprintf("ELEM %5d ESECT %3d ENODS %d ENOD",elem.Num, elem.Sect.Num, elem.Enods))
        for i:=0; i<elem.Enods; i++ {
            rtn.WriteString(fmt.Sprintf(" %d", elem.Enod[i].Num))
        }
        rtn.WriteString(" BONDS ")
        for i:=0; i<elem.Enods; i++ {
            for j:=0; j<6; j++ {
                if elem.Bonds[6*i+j] {
                    rtn.WriteString(fmt.Sprintf(" %d", 1))
                } else {
                    rtn.WriteString(fmt.Sprintf(" %d", 0))
                }
            }
            if i<elem.Enods-1 {
                rtn.WriteString(" ")
            } else {
                rtn.WriteString("\n")
            }
        }
        rtn.WriteString(fmt.Sprintf("           CANG %7.5f\n", elem.Cang))
        rtn.WriteString("           CMQ ")
        for i:=0; i<elem.Enods; i++ {
            for j:=0; j<6; j++ {
                rtn.WriteString(fmt.Sprintf(" %3.1f", elem.Cmq[6*i+j]))
            }
            if i<elem.Enods-1 {
                rtn.WriteString(" ")
            } else {
                rtn.WriteString("\n")
            }
        }
        rtn.WriteString(fmt.Sprintf("           TYPE %s\n", ETYPES[elem.Etype]))
        return rtn.String()
    } else {
        rtn.WriteString(fmt.Sprintf("ELEM %5d ESECT %3d ENODS %d ENOD",elem.Num, elem.Sect.Num, elem.Enods))
        for i:=0; i<elem.Enods; i++ {
            rtn.WriteString(fmt.Sprintf(" %d", elem.Enod[i].Num))
        }
        rtn.WriteString(" BONDS ")
        for i:=0; i<elem.Enods; i++ {
            for j:=0; j<6; j++ {
                rtn.WriteString(fmt.Sprintf(" %d", 0))
            }
            if i<elem.Enods-1 {
                rtn.WriteString(" ")
            } else {
                rtn.WriteString("\n")
            }
        }
        rtn.WriteString(fmt.Sprintf("                     EBANS 1 EBAN 1 BNODS %d BNOD", elem.Enods))
        for i:=0; i<elem.Enods; i++ {
            rtn.WriteString(fmt.Sprintf(" %d", elem.Enod[i].Num))
        }
        rtn.WriteString(fmt.Sprintf("\n           TYPE %s\n", ETYPES[elem.Etype]))
        if elem.Wrect != nil && (elem.Wrect[0] != 0.0 || elem.Wrect[1] != 0.0) {
            rtn.WriteString(fmt.Sprintf("           WRECT %.3f %.3f\n", elem.Wrect[0], elem.Wrect[1]))
        }
        return rtn.String()
    }
}

func (elem *Elem) InlString (period int) string {
    if !elem.IsLineElem() { return "" }
    var rtn bytes.Buffer
    rtn.WriteString(fmt.Sprintf("%5d %6d ", elem.Num, elem.Sect.Num))
    rtn.WriteString(fmt.Sprintf(" %5d %5d ", elem.Enod[0].Num, elem.Enod[1].Num))
    rtn.WriteString(fmt.Sprintf("%8.5f", elem.Cang))
    for i:=0; i<2; i++ {
        for j:=3; j<6; j++ {
            if elem.Bonds[6*i+j] {
                rtn.WriteString(" 1")
            } else {
                rtn.WriteString(" 0")
            }
        }
    }
    if period==0 {
        for i:=0; i<12; i++ {
            rtn.WriteString(fmt.Sprintf(" %10.8f", elem.Cmq[i]))
        }
    } else {
        for i:=0; i<12; i++ {
            rtn.WriteString(fmt.Sprintf(" %10.8f", 0.0))
        }
    }
    rtn.WriteString("\n")
    return rtn.String()
}
// }}}


// Amount// {{{
func (elem *Elem) Amount() float64 {
    if elem.IsLineElem() {
        return elem.Length()
    } else {
        return elem.Area()
    }
}

func (elem *Elem) Length() float64 {
    sum := 0.0
    for i:=0; i<3; i++ {
        sum += math.Pow((elem.Enod[1].Coord[i]-elem.Enod[0].Coord[i]),2)
    }
    return math.Sqrt(sum)
}

func (elem *Elem) Area() float64 {
    if elem.Enods <= 2 { return 0.0 }
    var area float64
    ds := make([]float64, elem.Enods-1)
    vs := make([][]float64, elem.Enods-1)
    for i:=1; i<elem.Enods; i++ {
        for j:=0; j<3; j++ {
            ds[i-1] += math.Pow((elem.Enod[i].Coord[j]-elem.Enod[0].Coord[j]), 2)
        }
        vs[i-1] = Direction(elem.Enod[0], elem.Enod[i], false)
    }
    for i:=0; i<elem.Enods-2; i++ {
        area += 0.5*math.Sqrt(ds[i]*ds[i+1] - math.Pow(Dot(vs[i], vs[i+1], 3), 2))
    }
    return area
}

func (elem *Elem) Weight () []float64 {
    rtn := make([]float64, 3)
    if elem.IsLineElem() {
        l := elem.Length()
        w := elem.Sect.Weight()
        for i:=0; i<3; i++ {
            rtn[i] = l * w[i]
        }
        return rtn
    } else {
        a := elem.Area()
        w := elem.Sect.Weight()
        for i:=0; i<3; i++ {
            rtn[i] = a * w[i]
        }
        return rtn
    }
}
// }}}


// Analysis// {{{
func (elem *Elem) Distribute () {
    // TODO: CMQ
    w := elem.Weight()
    for i:=0; i<3; i++ {
        for _, en := range elem.Enod {
            en.Weight[i] += w[i] / float64(elem.Enods)
        }
    }
}

// TODO: implement Width
func (elem *Elem) Width () float64 {
    return 0.0
}

// TODO: implement Height
func (elem *Elem) Height () float64 {
    return 0.0
}

func (elem *Elem) RectToBrace (nbrace int, rfact float64) []*Elem {
    if elem.Enods!=4 || elem.Sect.Figs[0].Prop.E == 0.0 {
        return nil
    }
    if thick, ok := elem.Sect.Figs[0].Value["THICK"]; ok {
        poi := elem.Sect.Figs[0].Prop.Poi
        l := elem.Width()
        h := elem.Height()
        // TODO: check wrate calculation
        wrate1 := elem.Wrect[0]/l
        wrate2 := 1.25*math.Sqrt((elem.Wrect[0]*elem.Wrect[1])/(l*h))
        var wrate float64
        if wrate1 > wrate2 {
            wrate = wrate1
        } else {
            wrate = wrate2
        }
        Ae := (1.0 - wrate)*rfact/1.2*math.Pow(l*l+h*h,1.5)*thick/(4.0*(1.0+poi)*l*h)
        Ae *= 2.0/float64(nbrace)
        f := NewFig()
        f.Prop = elem.Sect.Figs[0].Prop
        f.Value["AREA"] = Ae
        var sec *Sect
        sec = elem.Frame.SearchBraceSect(f, elem.Etype-2)
        if sec == nil {
            elem.Frame.Maxsnum++
            sec = elem.Frame.AddSect(elem.Frame.Maxsnum)
            sec.Figs = []*Fig{f}
            sec.Type = elem.Etype-2
            sec.Color = elem.Sect.Color
            for i:=0; i<12; i++ {
                if i%2==0 {
                    sec.Yield[i] = 100.0
                } else {
                    sec.Yield[i] = -100.0
                }
            }
        }
        rtn := make([]*Elem, nbrace)
        for i:=0; i<2; i++ {
            rtn[i] = NewLineElem([]*Node{elem.Enod[i], elem.Enod[i+2]}, sec, elem.Etype-2)
        }
        return rtn
    } else {
        return nil
    }
}

func (elem *Elem) Adopt (child *Elem) int {
    if elem.Children == nil {
        elem.Children = make([]*Elem, 2)
    }
    for i:=0; i<2; i++ {
        if elem.Children[i] == nil {
            elem.Children[i] = child
            child.Parent = elem
            return i
        }
    }
    return -1
}

// func (elem *Elem) StiffMatrix() []float64 {
//     l   := elem.Length()
//     E   := elem.Sect.E
//     Poi := elem.Sect.Poi
//     A   := elem.Sect.A
//     IX  := elem.Sect.IX
//     IY  := elem.Sect.IY
//     J   := elem.Sect.J
//     G   := 0.5 * E / (1.0+Poi)
// }
// }}}


// Vector// {{{
func (elem *Elem) Direction(normalize bool) []float64 {
    vec := make([]float64,3)
    var l float64
    if normalize {
        l = elem.Length()
    } else {
        l = 1.0
    }
    for i:=0; i<3; i++ {
        vec[i] = (elem.Enod[1].Coord[i]-elem.Enod[0].Coord[i])/l
    }
    return vec
}

func (elem *Elem) Normal (normalize bool) []float64 {
    if elem.Enods<3 { return nil }
    v1 := Direction(elem.Enod[0], elem.Enod[1], false)
    v2 := Direction(elem.Enod[0], elem.Enod[2], false)
    if normalize {
        return Normalize(Cross(v1, v2))
    } else {
        return Cross(v1, v2)
    }
}

func (elem *Elem) MidPoint() []float64 {
    rtn := make([]float64, 3)
    for i:=0; i<3; i++ {
        tmp := 0.0
        for _, n := range elem.Enod {
            tmp += n.Coord[i]
        }
        rtn[i] = tmp / float64(elem.Enods)
    }
    return rtn
}

func (elem *Elem) PLength() float64 {
    sum := 0.0
    for i:=0; i<2; i++ {
        sum += math.Pow((elem.Enod[1].Pcoord[i]-elem.Enod[0].Pcoord[i]),2)
    }
    return math.Sqrt(sum)
}

func (elem *Elem) PDirection(normalize bool) []float64 {
    vec := make([]float64,2)
    var l float64
    if normalize {
        l = elem.PLength()
    } else {
        l = 1.0
    }
    for i:=0; i<2; i++ {
        vec[i] = (elem.Enod[1].Pcoord[i]-elem.Enod[0].Pcoord[i])/l
    }
    return vec
}
// }}}


// Modify// {{{
func (elem *Elem) Move (x, y, z float64) {
    newenod := make([]*Node, elem.Enods)
    for i:=0; i<elem.Enods; i++ {
        newenod[i] = elem.Frame.CoordNode(elem.Enod[i].Coord[0]+x, elem.Enod[i].Coord[1]+y, elem.Enod[i].Coord[2]+z)
    }
    elem.Enod = newenod
}

func (elem *Elem) Copy (x, y, z float64) *Elem {
    newenod := make([]*Node, elem.Enods)
    for i:=0; i<elem.Enods; i++ {
        newenod[i] = elem.Frame.CoordNode(elem.Enod[i].Coord[0]+x, elem.Enod[i].Coord[1]+y, elem.Enod[i].Coord[2]+z)
    }
    if elem.IsLineElem() {
        newelem := elem.Frame.AddLineElem(-1, newenod, elem.Sect, elem.Etype)
        newelem.Cang = elem.Cang
        for i:=0; i<2; i++ {
            for j:=0; j<6; j++ {
                newelem.Bonds[6*i+j] = elem.Bonds[6*i+j]
                newelem.Cmq[6*i+j] = elem.Cmq[6*i+j]
            }
        }
        return newelem
    } else {
        return elem.Frame.AddPlateElem(-1, newenod, elem.Sect, elem.Etype)
    }
}

func (elem *Elem) Mirror(coord, vec []float64, del bool) *Elem {
    newenod := make([]*Node, elem.Enods)
    var add bool
    for i:=0; i<elem.Enods; i++ {
        newcoord := elem.Enod[i].MirrorCoord(coord, vec)
        newenod[i] = elem.Frame.CoordNode(newcoord[0], newcoord[1], newcoord[2])
        if !add && (newenod[i] != elem.Enod[i]) { add = true }
    }
    if add {
        if del {
            elem.Enod = newenod
            return elem
        } else {
            if elem.IsLineElem() {
                e := elem.Frame.AddLineElem(-1, newenod, elem.Sect, elem.Etype)
                for i:=0; i<6*elem.Enods; i++ {
                    e.Bonds[i] = elem.Bonds[i]
                }
                return e
            } else {
                return elem.Frame.AddPlateElem(-1, newenod, elem.Sect, elem.Etype)
            }
        }
    } else {
        return nil
    }
}
// }}}


// Divide// {{{
func (elem *Elem) DividingPoint (ratio float64) []float64 {
    rtn := make([]float64, 3)
    for i:=0; i<3; i++ {
        rtn[i] = elem.Enod[0].Coord[i]*(1.0-ratio) + elem.Enod[1].Coord[i]*ratio
    }
    return rtn
}

func (elem *Elem) AxisCoord (axis int, coord float64) (rtn []float64, err error) {
    if !elem.IsLineElem() { return rtn, errors.New("AxisCoord: PlateElem") }
    den := elem.Direction(false)[axis]
    if den == 0.0 {
        return rtn, errors.New("AxisCoord: Cannot Divide")
    }
    k := (coord - elem.Enod[0].Coord[axis]) / den
    return elem.DividingPoint(k), nil
}

func (elem *Elem) DivideAtCoord (x, y, z float64) (ns []*Node, els []*Elem, err error) {
    if !elem.IsLineElem() { return nil, nil, errors.New("DivideAtCoord: PlateElem")}
    els = make([]*Elem, 2)
    els[0] = elem
    n := elem.Frame.CoordNode(x, y, z)
    newelem := elem.Frame.AddLineElem(-1, []*Node{n, elem.Enod[1]}, elem.Sect, elem.Etype)
    els[1] = newelem
    elem.Enod[1] = n
    ns = []*Node{n}
    for j:=0; j<6; j++ {
        els[1].Bonds[6+j] = elem.Bonds[6+j]
        elem.Bonds[6+j] = false
    }
    return
}

func (elem *Elem) DivideAtNode (n *Node, position int, del bool) (rn []*Node, els []*Elem, err error) {
    if !elem.IsLineElem() { return nil, nil, errors.New("DivideAtNode: PlateElem")}
    els = make([]*Elem,2)
    if del {
        switch position {
        default:
            return nil, nil, errors.New("DivideAtNode: Unknown Position")
        case -1, 0:
            els := elem.Frame.SearchElem(elem.Enod[0])
            if len(els)==1 { delete(elem.Frame.Nodes, elem.Enod[0].Num) }
            elem.Enod[0] = n
            return []*Node{n}, []*Elem{elem}, nil
        case 1, 2:
            els := elem.Frame.SearchElem(elem.Enod[1])
            if len(els)==1 { delete(elem.Frame.Nodes, elem.Enod[1].Num) }
            elem.Enod[1] = n
            return []*Node{n}, []*Elem{elem}, nil
        }
    } else {
        switch position {
        default:
            return nil, nil, errors.New("DivideAtNode: Unknown Position")
        case 1, -1:
            els[0] = elem
            newelem := elem.Frame.AddLineElem(-1, []*Node{n, elem.Enod[1]}, elem.Sect, elem.Etype)
            els[1] = newelem
            elem.Enod[1] = n
            for j:=0; j<6; j++ {
                els[1].Bonds[6+j] = elem.Bonds[6+j]
                elem.Bonds[6+j] = false
            }
            return []*Node{n}, els, nil
        case 0:
            newelem := elem.Frame.AddLineElem(-1, []*Node{n, elem.Enod[0]}, elem.Sect, elem.Etype)
            els[0] = newelem
            els[1] = elem
            return []*Node{n}, els, nil
        case 2:
            newelem := elem.Frame.AddLineElem(-1, []*Node{elem.Enod[1], n}, elem.Sect, elem.Etype)
            els[0] = elem
            els[1] = newelem
            return []*Node{n}, els, nil
        }
    }
}

func (elem *Elem) DivideAtRate (k float64) (n []*Node, els []*Elem, err error) {
    c := elem.DividingPoint(k)
    return elem.DivideAtCoord(c[0], c[1], c[2])
}

func (elem *Elem) DivideAtMid () (n []*Node, els []*Elem, err error) {
    return elem.DivideAtRate(0.5)
}

func (elem *Elem) DivideAtAxis (axis int, coord float64) (n []*Node, els []*Elem, err error) {
    c, err := elem.AxisCoord(axis, coord)
    if err != nil { return }
    return elem.DivideAtCoord(c[0], c[1], c[2])
}

func (elem *Elem) OnNode (num int) []*Node {
    var num2 int
    if num>=elem.Enods {
        return nil
    } else if num==elem.Enods-1 {
        num2 = 0
    } else {
        num2 = num+1
    }
    candidate := elem.Frame.NodeInBox(elem.Enod[num], elem.Enod[num2])
    direction := elem.Frame.Direction(elem.Enod[num], elem.Enod[num2], false)
    ons := make([]*Node, len(candidate))
    i := 0
    nodes := make(map[float64]*Node, 0)
    var keys []float64
    for _, n := range candidate {
        if n.Num == elem.Enod[num].Num || n.Num == elem.Enod[num2].Num { continue }
        d := elem.Frame.Direction(elem.Enod[num], n, false)
        _, _, _, l := elem.Frame.Distance(elem.Enod[num], n)
        if IsParallel(direction, d, 1e-4) {
            nodes[l] = n
            keys = append(keys, l)
            ons[i] = n
            i++
        }
    }
    sort.Float64s(keys)
    sortednodes := make([]*Node, i)
    for j, k := range keys {
        sortednodes[j] = nodes[k]
    }
    return sortednodes
}

func (elem *Elem) DivideAtOns () (n []*Node, els []*Elem, err error) {
    if !elem.IsLineElem() { return nil, nil, errors.New("DivideAtCoord: PlateElem")}
    ns := elem.OnNode(0)
    l := len(ns)
    if l==0 { return nil, nil, nil }
    els = make([]*Elem, l+1)
    els[0] = elem
    for i:=l-1; i>=0; i-- {
        _, newels, err := elem.DivideAtNode(ns[i], 1, false)
        if err != nil {
            return nil, nil, err
        }
        els[i+1] = newels[1]
    }
    return
}
// }}}

func (elem *Elem) BetweenNode (index, size int) []*Node {
    var rtn []*Node
    var dst []float64
    var all bool
    if size < 0 {
        all = true
        rtn = make([]*Node, 0)
    } else {
        all = false
        rtn = make([]*Node, size)
        dst = make([]float64, size)
    }
    if size==0 || !elem.IsLineElem() { return rtn }
    d := elem.Direction(true)
    L := elem.Length()
    maxlen := 1000.0
    cand := 0
    for _, n := range elem.Frame.Nodes {
        if n.Hide { continue }
        if n == elem.Enod[0] || n == elem.Enod[1] { continue }
        d2 := Direction(elem.Enod[index], n, false)
        var ip float64
        if index == 0 {
            ip = Dot(d, d2, 3)
        } else {
            ip = -Dot(d, d2, 3)
        }
        if 0 < ip && ip < L {
            if all {
                rtn = append(rtn, n)
            } else {
                tmpd := Distance(elem.Enod[index], n)
                if cand < size {
                    last := true
                    for i:=0; i<cand; i++ {
                        if tmpd < dst[i] {
                            for j:=cand; j>i; j-- {
                                rtn[j] = rtn[j-1]
                                dst[j]=dst[j-1]
                            }
                            rtn[i] = n
                            dst[i] = tmpd
                            last = false
                            break
                        }
                    }
                    if last {
                        rtn[cand] = n
                        dst[cand] = tmpd
                    }
                    maxlen  = dst[cand]
                } else {
                    if tmpd < maxlen {
                        first := true
                        for i:=size-1; i>0; i-- {
                            if tmpd > dst[i-1] {
                                for j:=size-1; j>i; j-- {
                                    rtn[j] = rtn[j-1]
                                    dst[j] = dst[j-1]
                                }
                                rtn[i] = n
                                dst[i] = tmpd
                                first = false
                                break
                            }
                        }
                        if first {
                            for i:=size-1; i>0; i-- {
                                rtn[i] = rtn[i-1]
                                dst[i] = dst[i-1]
                            }
                            rtn[0] = n
                            dst[0] = tmpd
                        }
                        maxlen = dst[size-1]
                    }
                }
            }
            cand++
        }
    }
    if all {
        return rtn[:cand]
    } else {
        return rtn[:size]
    }
}


// Enod// {{{
func (elem *Elem) EnodIndex (side int) (int, error) {
    if 0 <= side && side < elem.Enods {
        return side, nil
    } else {
        for i, en := range elem.Enod {
            if en.Num == side {
                return i, nil
            }
        }
    }
    return -1, errors.New("EnodIndex: Not Found")
}

func (elem *Elem) RefNnum (nnum int) (int, error) {
    if 0 <= nnum && nnum < elem.Enods {
        return elem.Enod[nnum].Num, nil
    } else {
        for _, en := range elem.Enod {
            if en.Num == nnum {
                return nnum, nil
            }
        }
    }
    return 0, errors.New("RefNnum: Not Found")
}

func (elem *Elem) RefEnod (nnum int) (*Node, error) {
    if 0 <= nnum && nnum < elem.Enods {
        return elem.Enod[nnum], nil
    } else {
        for _, en := range elem.Enod {
            if en.Num == nnum {
                return en, nil
            }
        }
    }
    return nil, errors.New("RefEnod: Not Found")
}
// }}}


// STRESS// {{{
func (elem *Elem) ReturnStress (period string, nnum int, index int) float64 {
    if period == "" || !elem.IsLineElem() { return 0.0 }
    if pind := strings.Index(period, "+"); pind>=0 {
        return elem.ReturnStress(period[:pind], nnum, index) + elem.ReturnStress(period[pind+1:], nnum, index)
    }
    if mind := strings.Index(period, "-"); mind>=0 {
        ps := strings.Split(period, "-")
        val := elem.ReturnStress(ps[0], nnum, index)
        for i:=1; i<len(ps); i++ {
            val -= elem.ReturnStress(ps[i], nnum, index)
        }
        return val
    }
    if val, ok := elem.Stress[period]; ok {
        if nnum == 0 || nnum == 1 {
            return val[elem.Enod[nnum].Num][index]
        } else {
            for _, en := range elem.Enod {
                if en.Num == nnum {
                    return val[nnum][index]
                }
            }
            return 0.0
        }
    } else {
        return 0.0
    }
}

func (elem *Elem) N (period string, nnum int) float64 {
    return elem.ReturnStress(period, nnum, 0)
}
func (elem *Elem) QX (period string, nnum int) float64 {
    return elem.ReturnStress(period, nnum, 1)
}
func (elem *Elem) QY (period string, nnum int) float64 {
    return elem.ReturnStress(period, nnum, 2)
}
func (elem *Elem) MT (period string, nnum int) float64 {
    return elem.ReturnStress(period, nnum, 3)
}
func (elem *Elem) MX (period string, nnum int) float64 {
    return elem.ReturnStress(period, nnum, 4)
}
func (elem *Elem) MY (period string, nnum int) float64 {
    return elem.ReturnStress(period, nnum, 5)
}
// }}}


// Bond// {{{
func (elem *Elem) ChangeBond (bond []bool, side... int) error {
    if !elem.IsLineElem() || len(bond) < 6*len(side) { return errors.New("ChangeBond: Failed") }
    for _, i := range side {
        for j:=0; j<6; j++ {
            elem.Bonds[6*i+j] = bond[6*i+j]
        }
    }
    return nil
}

func (elem *Elem) ToggleBond (side int) error {
    if !elem.IsLineElem() { return errors.New("ToggleBond: PlateElem") }
    if ind, err := elem.EnodIndex(side); err == nil {
        if elem.Bonds[6*ind+4] || elem.Bonds[6*ind+5] {
            for i:=4; i<6; i++ {
                elem.Bonds[6*ind+i] = false
            }
        } else {
            for i:=4; i<6; i++ {
                elem.Bonds[6*ind+i] = true
            }
        }
        return nil
    } else {
        return err
    }
}

func (elem *Elem) BondState () (rtn int) {
    for i:=0; i<2; i++ {
        if elem.Bonds[6*i+4] || elem.Bonds[6*i+5] {
            rtn += i+1
        }
    }
    return
}
// }}}

func (elem *Elem) MomentCoord (show *Show, index int) [][]float64 {
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


func (elem *Elem) RateMax(show *Show) float64 {
    returnratemax := func (e *Elem) float64 {
                         if e.Rate == nil { return 0.0 }
                         if len(e.Rate)%3 != 0 || (show.ElemCaption & EC_RATE_L == 0 && show.ElemCaption & EC_RATE_S == 0) {
                             val := 0.0
                             for _, tmp := range e.Rate {
                                 if tmp > val { val = tmp }
                             }
                             return val
                         } else {
                             vall := 0.0
                             vals := 0.0
                             for i, tmp := range e.Rate {
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
                         }}
    if elem.IsLineElem() {
        return returnratemax(elem)
    } else {
        rtn := 0.0
        num := 0
        if elem.Children != nil {
            for _, el := range elem.Children {
                if el != nil {
                    rtn += returnratemax(el)
                    num++
                }
            }
        }
        if num > 0 {
            return rtn/float64(num)
        } else {
            return 0.0
        }
    }
}


func (elem *Elem) DistFromProjection () float64 {
    v := elem.Frame.View
    vec := make([]float64, 3)
    coord := elem.MidPoint()
    for i:=0; i<3; i++ {
        vec[i] = coord[i] - v.Focus[i]
    }
    return v.Dists[0] - Dot(vec, v.Viewpoint[0], 3)
}
