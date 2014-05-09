package st

import (
    "bytes"
    "errors"
    "fmt"
    "io/ioutil"
    "math"
    "os"
    "path/filepath"
    "sort"
    "strconv"
    "strings"
)


// Constants & Variables// {{{
var (
    PeriodExt = map[string]string{".inl": "L", ".otl": "L", ".ihx": "X", ".ohx": "X", ".ihy": "Y", ".ohy": "Y"}
)

const (
    RADIUS = 0.95
    EXPONENT = 1.5
    QUFACT = 1.25
)

var (
    // PlasticThreshold = math.Pow(RADIUS, EXPONENT)
    PlasticThreshold = RADIUS
)

var (
    InputExt  = []string{".inl", ".ihx", ".ihy"}
    OutputExt = []string{".otl", ".ohx", ".ohy"}
)

const (
    DEFAULT_WGT = "hogtxt.wgt"
)

const (
    MINCOORD = -100.0
    MAXCOORD = 1000.0
)

const (
    UPDATE_RESULT = iota
    ADD_RESULT
    ADDSEARCH_RESULT
)

var (
    XAXIS = []float64{1.0, 0.0, 0.0}
    YAXIS = []float64{0.0, 1.0, 0.0}
    ZAXIS = []float64{0.0, 0.0, 1.0}
)
// }}}


// type Frame// {{{
type Frame struct {
    Title   string
    Name    string
    Project string
    Path    string
    Home    string

    Base  float64
    Locate float64
    Tfact float64
    Gperiod float64
    View *View
    Nodes map[int]*Node
    Elems map[int]*Elem
    Props map[int]*Prop
    Sects map[int]*Sect
    Allows map[int]SectionRate

    Eigenvalue map[int]float64

    Kijuns map[string]*Kijun

    Maxenum int
    Maxnnum int
    Maxsnum int

    Nlap map[string]int
    Level []float64

    Show *Show

    DataFileName map[string]string
    ResultFileName map[string]string
    LstFileName string
}

func NewFrame() *Frame {
    f := new(Frame)
    f.Title = "\"CREATED ORGAN FRAME.\""
    f.Base = 0.2
    f.Locate = 1.0
    f.Tfact = 0.02
    f.Gperiod = 0.6
    f.Nodes = make(map[int]*Node)
    f.Elems = make(map[int]*Elem)
    f.Sects = make(map[int]*Sect)
    f.Allows = make(map[int]SectionRate)
    f.Props = make(map[int]*Prop)
    f.Eigenvalue = make(map[int]float64)
    f.Kijuns = make(map[string]*Kijun)
    f.View  = NewView()
    f.Maxnnum = 100
    f.Maxenum = 1000
    f.Maxsnum = 900
    f.Level = make([]float64, 0)
    f.Nlap = make(map[string]int)
    f.Show = NewShow(f)
    f.DataFileName = make(map[string]string)
    f.ResultFileName = make(map[string]string)
    return f
}
// }}}


// type View// {{{
type View struct {
    Gfact float64
    Focus []float64
    Angle []float64
    Dists []float64
    Perspective bool
    Viewpoint [][]float64
    Center []float64
}

func NewView() *View {
    v := new(View)
    v.Gfact = 1.0
    v.Focus = make([]float64,3)
    v.Angle = make([]float64,2)
    v.Dists = make([]float64,2)
    v.Dists[0] = 1000; v.Dists[1] = 5000
    v.Viewpoint = make([][]float64,3)
    for i:=0; i<3; i++ {
        v.Viewpoint[i] = make([]float64, 3)
    }
    v.Perspective = true
    v.Center = make([]float64, 2)
    return v
}

func (v *View) Copy () *View {
    nv := NewView()
    nv.Gfact = v.Gfact
    for i:=0; i<3; i++ {
        nv.Focus[i] = v.Focus[i]
        nv.Viewpoint[i] = v.Viewpoint[i]
    }
    for i:=0; i<2; i++ {
        nv.Angle[i] = v.Angle[i]
        nv.Dists[i] = v.Dists[i]
        nv.Center[i] = v.Center[i]
    }
    nv.Perspective = v.Perspective
    return nv
}
// }}}


func (frame *Frame) Bbox() (xmin, xmax, ymin, ymax, zmin, zmax float64) {
    var mins, maxs [3]float64
    first := true
    for _, j := range(frame.Nodes) {
        if j.Hide { continue }
        if first {
            for k:=0; k<3; k++ {
                mins[k] = j.Coord[k]
                maxs[k] = j.Coord[k]
            }
            first = false
        } else {
            for k:=0; k<3; k++ {
                if j.Coord[k] < mins[k] {
                    mins[k] = j.Coord[k]
                } else if maxs[k] < j.Coord[k] {
                    maxs[k] = j.Coord[k]
                }
            }
        }
    }
    return mins[0], maxs[0], mins[1], maxs[1], mins[2], maxs[2]
}


// Read
// ReadInp// {{{
func (frame *Frame) ReadInp(filename string, coord []float64, angle float64) error {
    tmp := make([]string, 0)
    nodemap := make(map[int]int)
    if len(coord)<3 {
        coord = []float64{0.0,0.0,0.0}
    }
    err := ParseFile(filename, func (words []string) error {
                                   var err error
                                   first := words[0]
                                   if strings.HasPrefix(first, "\"") {
                                       frame.Title = strings.Join(words, " ")
                                       return nil
                                   } else if strings.HasPrefix(first, "#") {
                                       return nil
                                   }
                                   switch first {
                                   default:
                                       tmp=append(tmp,words...)
                                   case "PROP", "SECT", "NODE", "ELEM":
                                       nodemap, err = frame.ParseInp(tmp, coord, angle, nodemap)
                                       tmp = words
                                   case "BASE":
                                       frame.Base, err =strconv.ParseFloat(words[1],64)
                                   case "LOCATE":
                                       frame.Locate, err =strconv.ParseFloat(words[1],64)
                                   case "TFACT":
                                       frame.Tfact, err =strconv.ParseFloat(words[1],64)
                                   case "GPERIOD":
                                       frame.Gperiod, err =strconv.ParseFloat(words[1],64)
                                   case "GFACT":
                                       frame.View.Gfact, err =strconv.ParseFloat(words[1],64)
                                   case "FOCUS":
                                       for i := 0; i < 3; i++ {
                                           frame.View.Focus[i], err = strconv.ParseFloat(words[i+1],64)
                                       }
                                   case "ANGLE":
                                       for i := 0; i < 2; i++ {
                                           frame.View.Angle[i], err = strconv.ParseFloat(words[i+1],64)
                                       }
                                   case "DISTS":
                                       for i := 0; i < 2; i++ {
                                           frame.View.Dists[i], err = strconv.ParseFloat(words[i+1],64)
                                       }
                                   }
                                   if err != nil {
                                       return err
                                   }
                                   return nil
                               })
    if err != nil {
        return err
    }
    nodemap, err = frame.ParseInp(tmp, coord, angle, nodemap)
    if err != nil {
        return err
    }
    frame.Name = filepath.Base(filename)
    frame.Project = ProjectName(filename)
    path, err := filepath.Abs(filename)
    if err != nil {
        frame.Path = filename
    } else {
        frame.Path = path
    }
    conffn  := Ce(filename, ".conf")
    prjconf := Ce(strings.Replace(filename, frame.Name, frame.Project, 1), ".conf")
    if FileExists(conffn) {
        err = frame.ReadConfigure(conffn)
        if err != nil {
            fmt.Println(err)
        }
    } else if FileExists(prjconf) {
        err = frame.ReadConfigure(prjconf)
        if err != nil {
            fmt.Println(err)
        }
    } else {
        fmt.Printf("No Configure File: %s\n", conffn)
    }
    return nil
}

func (frame *Frame) ParseInp(lis []string, coord []float64, angle float64, nodemap map[int]int) (map[int]int, error) {
    var err error
    var def, num int
    if len(lis)==0 {
        return nodemap, nil
    }
    first := lis[0]
    switch first {
    case "ELEM":
        err = frame.ParseElem(lis, nodemap)
    case "NODE":
        def, num, err = frame.ParseNode(lis, coord, angle)
        nodemap[def] = num
    case "SECT":
        err = frame.ParseSect(lis)
    case "PROP":
        err = frame.ParseProp(lis)
    }
    return nodemap, err
}

func (frame *Frame) ParseProp(lis []string) error {
    var num int64
    var err error
    p := new(Prop)
    for i, word := range(lis) {
        switch word {
        case "PROP":
            num, err = strconv.ParseInt(lis[i+1],10,64)
            p.Num = int(num)
        case "PNAME":
            p.Name, err = lis[i+1], nil
        case "HIJU":
            p.Hiju, err = strconv.ParseFloat(lis[i+1],64)
        case "E":
            p.E, err = strconv.ParseFloat(lis[i+1],64)
        case "POI":
            p.Poi, err = strconv.ParseFloat(lis[i+1],64)
        case "PCOLOR":
            var tmpcol int64
            val := 65536
            for j:=0; j<3; j++ {
                tmpcol, err = strconv.ParseInt(lis[i+1+j],10,64)
                p.Color += int(tmpcol)*val
                val >>= 8
            }
        }
        if err != nil {
            return err
        }
    }
    frame.Props[p.Num] = p
    return nil
}

func (frame *Frame) ParseSect(lis []string) error {
    var num int64
    var err error
    s := NewSect()
    tmp := make([]string,0)
    for i, word := range(lis) {
        switch word {
        case "FPROP","AREA", "IXX", "IYY", "VEN", "THICK", "SREIN":
            tmp=append(tmp,lis[i:i+2]...)
        case "XFACE","YFACE":
            tmp=append(tmp,lis[i:i+3]...)
        case "SECT":
            num, err = strconv.ParseInt(lis[i+1],10,64)
            s.Num = int(num)
        case "SNAME":
            s.Name, err = lis[i+1], nil
        case "FIG":
            err = s.ParseFig(frame,tmp)
            tmp = lis[i:i+2]
        case "LLOAD":
            for j:=0; j<3; j++ {
                s.Lload[j], err = strconv.ParseFloat(lis[i+1+j], 64)
            }
        case "EXP":
            s.Exp, err = strconv.ParseFloat(lis[i+1], 64)
        case "EXQ":
            s.Exq, err = strconv.ParseFloat(lis[i+1], 64)
        case "NZMAX":
            for j:=0; j<12; j++ {
                s.Yield[j], err = strconv.ParseFloat(lis[i+1+2*j], 64)
            }
        case "COLOR":
            var tmpcol int64
            s.Color = 0
            val := 65536
            for j:=0; j<3; j++ {
                tmpcol, err = strconv.ParseInt(lis[i+1+j],10,64)
                s.Color += int(tmpcol)*val
                val >>= 8
            }
        }
        if err != nil {
            return err
        }
    }
    err = s.ParseFig(frame,tmp)
    frame.Sects[s.Num] = s
    return nil
}

func (sect *Sect) ParseFig(frame *Frame, lis []string) error {
    var num int64
    if len(lis)==0 {
        return nil
    }
    var err error
    f := &Fig{Value: make(map[string]float64)}
    for i, word := range(lis) {
        switch word {
        case "FIG":
            num, err = strconv.ParseInt(lis[i+1],10,64)
            f.Num = int(num)
        case "FPROP":
            pnum, err := strconv.ParseInt(lis[i+1],10,64)
            if err==nil {
                if val,ok := frame.Props[int(pnum)]; ok {
                    f.Prop = val
                }
            }
        case "AREA","IXX","IYY","VEN","THICK","SREIN":
            val, err := strconv.ParseFloat(lis[i+1],64)
            if err == nil {
                f.Value[word] = val
            }
        case "XFACE","YFACE":
            tmp := make([]float64, 2)
            for j:=0; j<2; j++ {
                val, err := strconv.ParseFloat(lis[i+1+j], 64)
                if err != nil {
                    return err
                }
                tmp[j] = val
            }
            f.Value[word] = tmp[0]
            f.Value[fmt.Sprintf("%s_H",word)] = tmp[1]
        }
        if err != nil {
            return err
        }
    }
    sect.Figs=append(sect.Figs,f)
    return err
}

func (frame *Frame) ParseNode(lis []string, coord []float64, angle float64) (int, int, error) {
    var num int64
    var err error
    n := NewNode()
    llis := len(lis)
    for i, word := range(lis) {
        switch word {
        case "NODE":
            num, err = strconv.ParseInt(lis[i+1],10,64)
            n.Num = int(num)
        case "CORD":
            if llis<i+4 {
                return 0, 0, errors.New(fmt.Sprintf("ParseNode: CORD IndexError NODE %d", n.Num))
            }
            for j:=0; j<3; j++ {
                n.Coord[j], err = strconv.ParseFloat(lis[i+1+j],64)
                n.Coord[j] += coord[j]
            }
        case "ICON":
            if llis<i+7 {
                return 0, 0, errors.New(fmt.Sprintf("ParseNode: ICON IndexError NODE %d", n.Num))
            }
            for j:=0; j<6; j++ {
                if lis[i+1+j]=="0" {
                    n.Conf[j]=false
                } else {
                    n.Conf[j]=true
                }
            }
        case "VCON":
            if llis<i+7 {
                return 0, 0, errors.New(fmt.Sprintf("ParseNode: VCON IndexError NODE %d", n.Num))
            }
            for j:=0; j<6; j++ {
                n.Load[j], err = strconv.ParseFloat(lis[i+1+j],64)
            }
        }
        if err != nil {
            return 0, 0, err
        }
    }
    c := RotateVector(n.Coord, coord, []float64{0.0, 0.0, 1.0}, angle)
    n.Coord = c
    newnode := frame.SearchNode(c[0], c[1], c[2])
    if newnode == nil {
        if n.Num > frame.Maxnnum {
            frame.Maxnnum = n.Num
            frame.Nodes[n.Num] = n
            n.Frame = frame
            return n.Num, n.Num, nil
        } else {
            frame.Maxnnum++
            old := n.Num
            n.Num = frame.Maxnnum
            frame.Nodes[n.Num] = n
            n.Frame = frame
            return old, n.Num, nil
        }
    } else {
        return n.Num, newnode.Num, nil
    }
}

func (frame *Frame) ParseElem(lis []string, nodemap map[int]int) error {
    var num int64
    var err error
    e := new(Elem)
    llis := len(lis)
    for i, word := range(lis) {
        switch word {
        case "ELEM":
            num, err = strconv.ParseInt(lis[i+1],10,64)
            e.Num = int(num)
        case "ESECT":
            tmp, err := strconv.ParseInt(lis[i+1],10,64)
            if err != nil {
                return err
            }
            if val,ok := frame.Sects[int(tmp)]; ok {
                e.Sect = val
            } else {
                fmt.Printf("SECT %d doesn't exist\n", tmp)
                e.Sect = frame.AddSect(int(tmp))
            }
        case "ENODS":
            num, err = strconv.ParseInt(lis[i+1],10,64)
            e.Enods = int(num)
        case "ENOD":
            if llis<i+1+e.Enods {
                return errors.New(fmt.Sprintf("ParseElem: ENOD IndexError ELEM %d", e.Num))
            }
            en := make([]*Node,int(e.Enods))
            for j:=0;j<e.Enods;j++ {
                tmp, err := strconv.ParseInt(lis[i+1+j],10,64)
                if err != nil {
                    return err
                }
                if val,ok := frame.Nodes[nodemap[int(tmp)]]; ok {
                    en[j] = val
                } else {
                    return errors.New(fmt.Sprintf("ParseElem: Enod not found ELEM %d ENOD %d", e.Num, tmp))
                }
            }
            e.Enod = en
        case "BONDS":
            if llis<i+1+e.Enods*6 {
                return errors.New(fmt.Sprintf("ParseElem: BONDS IndexError ELEM %d", e.Num))
            }
            bon := make([]bool, int(e.Enods)*6)
            for j:=0;j<int(e.Enods)*6;j++ {
                if lis[i+1+j]=="0" {
                    bon[j] = false
                } else {
                    bon[j] = true
                }
            }
            e.Bonds = bon
        case "CMQ":
            if llis<i+1+e.Enods*6 {
                return errors.New(fmt.Sprintf("ParseElem: CMQ IndexError ELEM %d", e.Num))
            }
            cmq := make([]float64, int(e.Enods)*6)
            for j:=0;j<int(e.Enods)*6;j++ {
                cmq[j], err = strconv.ParseFloat(lis[i+1+j],64)
            }
            e.Cmq = cmq
        case "CANG":
            e.Cang, err = strconv.ParseFloat(lis[i+1],64)
        case "WRECT":
            wrect := make([]float64, 2)
            for j:=0; j<2; j++ {
                val, err := strconv.ParseFloat(lis[i+1+j], 64)
                if err != nil {
                    return err
                }
                wrect[j] = val
            }
            e.Wrect = wrect
        case "TYPE":
            err = e.setEtype(lis[i+1])
        }
        if err != nil {
            return err
        }
    }
    var el *Elem
    if e.IsLineElem() {
        el = NewLineElem(e.Enod, e.Sect, e.Etype)
        el.Num = e.Num
        el.Cang = e.Cang
        el.Cmq = e.Cmq
        el.Bonds = e.Bonds
        el.SetPrincipalAxis()
    } else {
        el = NewPlateElem(e.Enod, e.Sect, e.Etype)
        el.Num = e.Num
        el.Wrect = e.Wrect
    }
    el.Frame = frame
    if _, exist := frame.Elems[el.Num]; !exist {
        frame.Elems[el.Num] = el
        if enum := el.Num; enum > frame.Maxenum {
            frame.Maxenum = enum
        }
    } else {
        frame.Maxenum++
        el.Num = frame.Maxenum
        frame.Elems[frame.Maxenum] = el
    }
    return nil
}
// }}}


// ReadConfigure// {{{
func (frame *Frame) ReadConfigure (filename string) error {
    tmp := make([]string, 0)
    err := ParseFile(filename, func(words []string) error {
                                   var err error
                                   first := strings.Trim(strings.ToUpper(words[0]), ":")
                                   if strings.HasPrefix(first, "#") {
                                       return nil
                                   }
                                   switch first {
                                   default:
                                       tmp=append(tmp,words...)
                                   case "LEVEL":
                                       err = frame.ParseConfigure(tmp)
                                       tmp=words
                                   }
                                   if err != nil {
                                       return err
                                   }
                                   return nil
                               })
    err = frame.ParseConfigure(tmp)
    if err != nil {
        return err
    }
    return nil
}

func (frame *Frame) ParseConfigure(lis []string) (err error) {
    if len(lis)==0 {
        return nil
    }
    first := strings.Trim(strings.ToUpper(lis[0]), ":")
    switch first {
    case "LEVEL":
        err = frame.ParseLevel(lis[1:])
    }
    return err
}

func (frame *Frame) ParseLevel(lis []string) (err error) {
    size := len(lis)
    val := make([]float64, size)
    for i:=0; i<size; i++ {
        tmp, err := strconv.ParseFloat(lis[i], 64)
        if err != nil {
            return err
        }
        val[i] = tmp
    }
    frame.Level = val
    return nil
}
// }}}


// ReadData// {{{
func (frame *Frame) ReadData (filename string) error {
    f, err := ioutil.ReadFile(filename)
    if err != nil {
        return err
    }
    ext := filepath.Ext(filename)
    var period string
    if p, ok := PeriodExt[ext]; ok {
        period = p
    } else {
        period = strings.ToUpper(ext[1:])
    }
    var lis []string
    if ok := strings.HasSuffix(string(f),"\r\n"); ok {
        lis = strings.Split(string(f),"\r\n")
    } else {
        lis = strings.Split(string(f),"\n")
    }
    var words []string
    for _, k := range strings.Split(lis[0]," ") {
        if k!="" {
            words=append(words,k)
        }
    }
    nums := make([]int, 3)
    for i:=0; i<3; i++ {
        num, err := strconv.ParseInt(words[i], 10, 64)
        if err != nil {
            return err
        }
        nums[i] = int(num)
    }
    // Sect
    for _, j := range lis[1:1+nums[2]] {
        var words []string
        for _, k := range strings.Split(j," ") {
            if k!="" {
                words=append(words,k)
            }
        }
        if len(words)==0 {
            continue
        }
        num, err := strconv.ParseInt(words[0], 10, 64)
        if err != nil {
            return err
        }
        snum := int(num)
        if _, ok := frame.Sects[snum]; !ok {
            sect := frame.AddSect(snum) // TODO: set E, poi, ...
            for i:=0; i<12; i++ {
                val, err := strconv.ParseFloat(words[7+i], 64)
                if err != nil {
                    return err
                }
                sect.Yield[i] = val
            }
            if len(words)>=20 {
                tp, err := strconv.ParseInt(words[19], 10 ,64)
                if err != nil {
                    return err
                }
                sect.Type = int(tp)
            }
        }
    }
    // Node1
    for _, j := range lis[1+nums[2]:1+nums[2]+nums[0]] {
        var words []string
        for _, k := range strings.Split(j," ") {
            if k!="" {
                words=append(words,k)
            }
        }
        if len(words)==0 {
            continue
        }
        num, err := strconv.ParseInt(words[0], 10, 64)
        if err != nil {
            return err
        }
        nnum := int(num)
        if _, ok := frame.Nodes[nnum]; !ok {
            fmt.Printf("Append Node %d\n", nnum)
        }
    }
    // Elem
    for _, j := range lis[1+nums[2]+nums[0]:1+nums[2]+nums[0]+nums[1]] {
        var words []string
        for _, k := range strings.Split(j," ") {
            if k!="" {
                words=append(words,k)
            }
        }
        if len(words)==0 {
            continue
        }
        num, err := strconv.ParseInt(words[0], 10, 64)
        if err != nil {
            return err
        }
        enum := int(num)
        if _, ok := frame.Elems[enum]; !ok {
            sec, err := strconv.ParseInt(words[1], 10, 64)
            if err != nil {
                return err
            }
            ns := make([]*Node, 2)
            for i:=0; i<2; i++ {
                tmp, err := strconv.ParseInt(words[2+i], 10, 64)
                if err != nil {
                    return err
                }
                ns[i] = frame.Nodes[int(tmp)]
            }
            sect := frame.Sects[int(sec)]
            newel := frame.AddLineElem(enum, ns, sect, sect.Type-1) // TODO: set etype, cang, ...
            // TODO: search parent
            for _, el := range frame.SearchElem(ns...) {
                if el.Etype == sect.Type+1 {
                    el.Adopt(newel)
                }
            }
        }
    }
    // Node2
    for _, j := range lis[1+nums[2]+nums[0]+nums[1]:1+nums[2]+nums[0]+nums[1]+nums[0]] {
        var words []string
        for _, k := range strings.Split(j," ") {
            if k!="" {
                words=append(words,k)
            }
        }
        if len(words)==0 {
            continue
        }
        num, err := strconv.ParseInt(words[0], 10, 64)
        if err != nil {
            return err
        }
        nnum := int(num)
        if node, ok := frame.Nodes[nnum]; ok {
            force := make([]float64, 6)
            for i:=0; i<6; i++ {
                val, err := strconv.ParseFloat(words[7+i], 64)
                if err != nil {
                    return err
                }
                force[i] = val
            }
            node.Force[period] = force
        }
    }
    frame.DataFileName[period] = filename
    return nil
}
// }}}


// ReadResult// {{{
func (frame *Frame) ReadResult (filename string, mode uint) error {
    f, err := ioutil.ReadFile(filename)
    if err != nil {
        return err
    }
    ext := filepath.Ext(filename)
    var period string
    if p, ok := PeriodExt[ext]; ok {
        period = p
    } else {
        period = strings.ToUpper(ext[1:])
    }
    var lis []string
    if ok := strings.HasSuffix(string(f),"\r\n"); ok {
        lis = strings.Split(string(f),"\r\n")
    } else {
        lis = strings.Split(string(f),"\n")
    }
    tmpline := 0
    for _, j := range lis {
        if strings.HasPrefix(strings.Trim(j, " "), "**") {
            tmpline++
            break
        }
        tmpline++
    }
    for _, j := range lis[tmpline:] {
        if strings.HasPrefix(strings.Trim(j, " "), "NO") {
            tmpline++
            break
        }
        tmpline++
    }
    for _, j := range lis[tmpline:] {
        if j=="" {
            tmpline++
            continue
        }
        break
    }
    for { // Reading Elem Stress
        j := strings.Join([]string{lis[tmpline], lis[tmpline+1]}, " ")
        tmpline+=2
        var words []string
        for _, k := range strings.Split(j," ") {
            if k!="" {
                words=append(words,k)
            }
        }
        if len(words)==0 {
            continue
        }
        if strings.HasPrefix(strings.Trim(words[0], " "), "**") {
            break
        }
        num, err := strconv.ParseInt(words[0], 10, 64)
        if err != nil {
            return err
        }
        enum := int(num)
        enod := make([]int, 2)
        stress := make([][]float64, 2)
        if elem, ok := frame.Elems[enum]; ok {
            if !elem.IsLineElem() { continue }
            if mode == UPDATE_RESULT { elem.Stress[period] = make(map[int][]float64) }
            var tmp []float64
            for i:=0; i<2; i++ {
                num, err := strconv.ParseInt(words[2+7*i], 10, 64)
                if err != nil {
                    return err
                }
                enod[i] = int(num)
                tmp = make([]float64, 6)
                for k:=0; k<6; k++ {
                    val, err := strconv.ParseFloat(words[3+7*i+k], 64)
                    if err != nil {
                        return err
                    }
                    tmp[k] = val
                }
                switch mode {
                case UPDATE_RESULT:
                    elem.Stress[period][int(num)]=tmp
                case ADD_RESULT, ADDSEARCH_RESULT:
                    if elem.Stress[period][int(num)] != nil {
                        for ind:=0; ind<6; ind++ {
                            elem.Stress[period][int(num)][ind] += tmp[ind]
                        }
                    }
                }
                stress[i] = tmp
            }
        } else {
            if mode == ADDSEARCH_RESULT {
                if _, ok := frame.Nodes[enod[0]]; ok {
                    if _, ok2 := frame.Nodes[enod[1]]; ok2 {
                        for _, el := range frame.SearchElem(frame.Nodes[0], frame.Nodes[1]) {
                            if !el.IsLineElem() { continue }
                            fmt.Println("ReadResult: ELEM %d -> ELEM %d", enum, el.Num)
                            for i:=0; i<2; i++ {
                                for j:=0; j<6; j++ {
                                    el.Stress[period][enod[i]][j] += stress[i][j]
                                }
                            }
                            break
                        }
                    }
                }
            } else {
                fmt.Printf("ELEM %d not found\n", enum)
            }
        }
    }
    for _, j := range lis[tmpline:] {
        if strings.HasPrefix(strings.Trim(j, " "), "NO") {
            tmpline++
            break
        }
        tmpline++
    }
    for _, j := range lis[tmpline:] {
        tmpline++
        var words []string
        for _, k := range strings.Split(j," ") {
            if k!="" {
                words=append(words,k)
            }
        }
        if len(words)==0 {
            continue
        }
        if strings.HasPrefix(strings.Trim(words[0], " "), "**") {
            break
        }
        num, err := strconv.ParseInt(words[0], 10, 64)
        if err != nil {
            return err
        }
        nnum := int(num)
        if node, ok := frame.Nodes[nnum]; ok {
            tmp := make([]float64, 6)
            for k:=0; k<6; k++ {
                val, err := strconv.ParseFloat(words[1+k], 64)
                if err != nil {
                    return err
                }
                tmp[k] = val
            }
            switch mode {
            case UPDATE_RESULT:
                node.Disp[period] = tmp
            case ADD_RESULT, ADDSEARCH_RESULT:
                for ind:=0; ind<6; ind++ {
                    node.Disp[period][ind] += tmp[ind]
                }
            }
        } else {
            fmt.Printf("NODE %d not found\n", nnum)
        }
    }
    for _, j := range lis[tmpline:] {
        if strings.HasPrefix(strings.Trim(j, " "), "NO") {
            tmpline++
            break
        }
        tmpline++
    }
    for _, j := range lis[tmpline:] {
        var words []string
        for _, k := range strings.Split(j," ") {
            if k!="" {
                words=append(words,k)
            }
        }
        if len(words)==0 {
            continue
        }
        num, err := strconv.ParseInt(words[0], 10, 64)
        if err != nil {
            return err
        }
        nnum := int(num)
        if node, ok := frame.Nodes[nnum]; ok {
            if _, ok := node.Reaction[period]; !ok {
                node.Reaction[period] = make([]float64, 6)
            }
            ind, err := strconv.ParseInt(words[1], 10, 64)
            val, err := strconv.ParseFloat(words[2], 64)
            if err != nil {
                return err
            }
            switch mode {
            case UPDATE_RESULT:
                node.Reaction[period][ind-1] = val
            case ADD_RESULT, ADDSEARCH_RESULT:
                node.Reaction[period][ind-1] += val
            }
        } else {
            fmt.Printf("NODE %d not found\n", nnum)
        }
    }
    frame.ResultFileName[period] = filename
    return nil
}
// }}}


// ReadBuckling// {{{
func (frame *Frame) ReadBuckling (filename string) error {
    tmp := make([][]string, 0)
    err := ParseFile(filename, func(words []string) error {
                                   var err error
                                   first := strings.ToUpper(words[0])
                                   if strings.HasPrefix(first, "#") {
                                       return nil
                                   }
                                   switch first {
                                   default:
                                       tmp=append(tmp,words)
                                   case "EIGEN":
                                       err = frame.ParseEigen(tmp)
                                       tmp=[][]string{words}
                                   }
                                   if err != nil {
                                       return err
                                   }
                                   return nil
                               })
    err = frame.ParseEigen(tmp)
    if err != nil {
        return err
    }
    return nil
}

func (frame *Frame) ParseEigen(lis [][]string) (err error) {
    if strings.ToUpper(lis[0][0]) == "EIGEN" {
        eig := strings.Split(lis[0][2], "=")
        eigmode, err := strconv.ParseInt(eig[0], 10, 64)
        if err != nil {
            return err
        }
        eigval, err := strconv.ParseFloat(eig[1], 64)
        frame.Eigenvalue[int(eigmode-1)] = eigval
        period := fmt.Sprintf("B%d", eigmode)
        for _, l := range lis[1:] {
            if strings.ToUpper(l[0]) == "NODE:" {
                nnum, err := strconv.ParseInt(l[1], 10, 64)
                if err != nil {
                    return err
                }
                disp := make([]float64, 6)
                for i:=0; i<6; i++ {
                    val, err := strconv.ParseFloat(l[3+i], 64)
                    if err != nil {
                        return err
                    }
                    disp[i] = val
                }
                frame.Nodes[int(nnum)].Disp[period] = disp
            }
        }
    }
    return nil
}
// }}}


// ReadZoubun// {{{
func (frame *Frame) ReadZoubun (filename string) error {
    tmp := make([][]string, 0)
    var period string
    ext := strings.ToUpper(filepath.Ext(filename)[1:])
    nlap := 0
    err := ParseFile(filename, func(words []string) error {
                                   var err error
                                   first := strings.ToUpper(words[0])
                                   if strings.HasPrefix(first, "#") {
                                       return nil
                                   }
                                   switch first {
                                   default:
                                       if strings.HasPrefix(first, "LAP") {
                                           nlap++
                                           err = frame.ParseZoubun(tmp, period)
                                           tmp=[][]string{words}
                                           period = fmt.Sprintf("%s@%s", ext, strings.Split(strings.Split(first, ":")[1], "/")[0])
                                       } else {
                                           tmp=append(tmp,words)
                                       }
                                   case "\"DISPLACEMENT\"", "\"STRESS\"", "\"REACTION\"", "\"CURRENT":
                                       err = frame.ParseZoubun(tmp, period)
                                       tmp=[][]string{words}
                                   }
                                   if err != nil {
                                       return err
                                   }
                                   return nil
                               })
    if err != nil {
        return err
    }
    err = frame.ParseZoubun(tmp, period)
    if err != nil {
        return err
    }
    frame.Nlap[ext] = nlap
    return nil
}

func (frame *Frame) ParseZoubun (lis [][]string, period string) error {
    var err error
    if len(lis)==0 {
        return nil
    }
    first := strings.ToUpper(lis[0][0])
    switch first {
    case "\"STRESS\"":
        err = frame.ParseZoubunStress(lis, period)
    case "\"REACTION\"":
        err = frame.ParseZoubunReaction(lis, period)
    case "\"CURRENT":
        err = frame.ParseZoubunForm(lis, period)
    }
    return err
}

func (frame *Frame) ParseZoubunStress (lis [][]string, period string) error {
    for _, l := range lis {
        if strings.ToUpper(l[0]) == "ELEM" {
            enum, err := strconv.ParseInt(l[1], 10, 64)
            if err != nil {
                return err
            }
            if el, ok := frame.Elems[int(enum)]; ok {
                nnum, err := strconv.ParseInt(strings.Trim(l[5], ":"), 10, 64)
                if err != nil {
                    return err
                }
                stress := make([]float64, 6)
                for i:=0; i<6; i++ {
                    val, err := strconv.ParseFloat(l[7+2*i], 64)
                    if err != nil {
                        return err
                    }
                    stress[i] = val
                }
                val, err := strconv.ParseFloat(l[19], 64)
                if err != nil {
                    return err
                }
                ph := val > PlasticThreshold
                if el.Stress[period] == nil { el.Stress[period] = make(map[int][]float64) }
                if el.Phinge[period] == nil { el.Phinge[period] = make(map[int]bool) }
                el.Stress[period][int(nnum)] = stress
                el.Phinge[period][int(nnum)] = ph
            }
        }
    }
    return nil
}

func (frame *Frame) ParseZoubunReaction (lis [][]string, period string) error {
    for _, l := range lis {
        if strings.ToUpper(l[0]) == "NODE:" {
            nnum, err := strconv.ParseInt(l[1], 10, 64)
            if err != nil {
                return err
            }
            if n, ok := frame.Nodes[int(nnum)]; ok {
                if n.Reaction[period] == nil { n.Reaction[period] = make([]float64, 6) }
                ind, err := strconv.ParseInt(l[2], 10, 64)
                if err != nil {
                    return err
                }
                val, err := strconv.ParseFloat(l[5], 64)
                if err != nil {
                    return err
                }
                n.Reaction[period][ind-1] = val
            }
        }
    }
    return nil
}

func (frame *Frame) ParseZoubunForm (lis [][]string, period string) error {
    for _, l := range lis {
        if strings.ToUpper(l[0]) == "NODE:" {
            nnum, err := strconv.ParseInt(l[1], 10, 64)
            if err != nil {
                return err
            }
            if n, ok := frame.Nodes[int(nnum)]; ok {
                disp := make([]float64, 6)
                for i:=0; i<6; i++ {
                    val, err := strconv.ParseFloat(l[3+i], 64)
                    if err != nil {
                        return err
                    }
                    if i < 3 {
                        disp[i] = val - n.Coord[i]
                    } else {
                        disp[i] = val
                    }
                }
                n.Disp[period] = disp
            }
        }
    }
    return nil
}
// }}}


// ReadLst// {{{
func (frame *Frame) ReadLst (filename string) error {
    tmp := make([][]string, 0)
    err := ParseFile(filename, func(words []string) error {
                                   var err error
                                   first := strings.ToUpper(words[0])
                                   if strings.HasPrefix(first, "#") {
                                       return nil
                                   }
                                   switch first {
                                   default:
                                       tmp=append(tmp,words)
                                   case "CODE":
                                       err = frame.ParseLst(tmp)
                                       tmp=[][]string{words}
                                   }
                                   if err != nil {
                                       return err
                                   }
                                   return nil
                               })
    if err != nil {
        return err
    }
    err = frame.ParseLst(tmp)
    if err != nil {
        return err
    }
    frame.LstFileName = filename
    return nil
}

func (frame *Frame) ParseLst (lis [][]string) error {
    var err error
    if len(lis)==0 || len(lis[0])<=2 {
        return nil
    }
    first := strings.ToUpper(lis[0][0])
    switch first {
    case "CODE":
        mat := strings.ToUpper(lis[0][2])
        switch mat {
        case "S":
            err = frame.ParseLstSteel(lis)
        case "RC":
            err = frame.ParseLstRC(lis)
        }
    }
    return err
}

func (frame *Frame) ParseLstSteel (lis [][]string) error {
    var num int
    var sr SectionRate
    var shape Shape
    var material Steel
    var err error
    tmp, err := strconv.ParseInt(lis[0][1], 10, 64)
    num = int(tmp)
    var size int
    switch lis[1][0] {
    case "HKYOU":
        size = 4
        shape, err = NewHKYOU(lis[1][1:1+size])
    case "HWEAK":
        size = 4
        shape, err = NewHWEAK(lis[1][1:1+size])
    case "RPIPE":
        size = 4
        shape, err = NewRPIPE(lis[1][1:1+size])
    case "CPIPE":
        size = 2
        shape, err = NewCPIPE(lis[1][1:1+size])
    }
    if err != nil { return err }
    switch lis[1][1+size] {
    default:
        material = SN400
    case "SN400":
        material = SN400
    case "SN490":
        material = SN490
    }
    switch lis[0][3] {
    case "COLUMN":
        sr = NewSColumn(num, shape, material)
    case "GIRDER":
        sr = NewSGirder(num, shape, material)
    }
    for _, words := range(lis[2:]) {
        first := strings.ToUpper(words[0])
        switch first {
        case "XFACE", "YFACE", "BBLEN", "BTLEN", "BBFAC", "BTFAC":
            vals := make([]float64, 2)
            for i:=0; i<2; i++ {
                val, err := strconv.ParseFloat(words[1+i], 64)
                if err == nil { vals[i] = val }
            }
            sr.SetValue(first, vals)
        }
        if err != nil {
            return err
        }
    }
    if err != nil {
        return err
    }
    sr.SetName(strings.Trim(lis[0][4], "\""))
    frame.Allows[num] = sr
    return nil
}

func (frame *Frame) ParseLstRC (lis [][]string) error {
    var num int
    var sr SectionRate
    var err error
    tmp, err := strconv.ParseInt(lis[0][1], 10, 64)
    num = int(tmp)
    switch lis[0][3] {
    case "COLUMN":
        sr = NewRCColumn(num)
    case "GIRDER":
        sr = NewRCGirder(num)
    }
    for _, words := range(lis[2:]) {
        first := strings.ToUpper(words[0])
        switch first {
        case "REINS":
            switch sr.(type) {
            case *RCColumn:
                err = sr.(*RCColumn).AddReins(words[1:])
            case *RCGirder:
                err = sr.(*RCGirder).AddReins(words[1:])
            }
        case "HOOPS":
            switch sr.(type) {
            case *RCColumn:
                err = sr.(*RCColumn).SetHoops(words[1:])
            case *RCGirder:
                err = sr.(*RCGirder).SetHoops(words[1:])
            }
        case "CRECT":
            switch sr.(type) {
            case *RCColumn:
                err = sr.(*RCColumn).SetConcrete(words)
            case *RCGirder:
                err = sr.(*RCGirder).SetConcrete(words)
            }
        case "XFACE", "YFACE", "BBLEN", "BTLEN", "BBFAC", "BTFAC":
            vals := make([]float64, 2)
            for i:=0; i<2; i++ {
                val, err := strconv.ParseFloat(words[1+i], 64)
                if err == nil { vals[i] = val }
            }
            sr.SetValue(first, vals)
        }
        if err != nil {
            return err
        }
    }
    if err != nil {
        return err
    }
    sr.SetName(lis[0][4])
    frame.Allows[num] = sr
    return nil
}
// }}}


// ReadRat// {{{
func (frame *Frame) ReadRat (filename string) error {
    err := ParseFile(filename, func(words []string) error {
                                   enum, err := strconv.ParseInt(words[1], 10, 64)
                                   rate := make([]float64, len(words)-4)
                                   for i:=0; i<len(words)-4; i++ {
                                       val, _ := strconv.ParseFloat(words[4+i], 64)
                                       rate[i] = val
                                   }
                                   if err != nil {
                                       return err
                                   }
                                   if el, ok := frame.Elems[int(enum)]; ok {
                                       el.Rate = rate
                                   }
                                   return nil
                               })
    if err != nil {
        return err
    }
    return nil
}
// }}}


// ReadWgt// {{{
func (frame *Frame) ReadWgt (filename string) error {
    f, err := ioutil.ReadFile(filename)
    if err != nil {
        return err
    }
    var lis []string
    if ok := strings.HasSuffix(string(f),"\r\n"); ok {
        lis = strings.Split(string(f),"\r\n")
    } else {
        lis = strings.Split(string(f),"\n")
    }
    num := len(frame.Nodes)
    rwgtloop:
        for _, j := range lis {
            if num == 0 { break }
            var words []string
            for _, k := range strings.Split(j," ") {
                if k!="" {
                    words=append(words,k)
                }
            }
            if len(words)==0 {
                continue
            }
            nnum, err := strconv.ParseInt(words[0], 10, 64)
            if err != nil { continue rwgtloop }
            if n, ok := frame.Nodes[int(nnum)]; ok {
                if len(words) < 4 { return errors.New(fmt.Sprintf("ReadWgt: Index Error (NODE %d)", nnum)) }
                wgt := make([]float64, 3)
                for i:=0; i<3; i++ {
                    val, err := strconv.ParseFloat(words[1+i], 64)
                    if err != nil { continue rwgtloop }
                    wgt[i] = val
                }
                n.Weight = wgt
                num--
            }
        }
    return nil
}
// }}}


// ReadKjn// {{{
func (frame *Frame) ReadKjn (filename string) error {
    err := ParseFile(filename, func(words []string) error {
                                   if strings.HasPrefix(words[0], "#") {
                                       return nil
                                   }
                                   var err error
                                   if _, ok := frame.Kijuns[words[0]]; ok {
                                       fmt.Printf("KIJUN %s already exists\n", words[0])
                                   } else {
                                       err = frame.ParseKjn(words)
                                   }
                                   if err != nil {
                                       return err
                                   }
                                   return nil
                               })
    if err != nil {
        return err
    }
    return nil
}

func (frame *Frame) ParseKjn (lis []string) error {
    k := NewKijun()
    k.Name = lis[0]
    for i:=0; i<3; i++ {
        tmp, err := strconv.ParseFloat(lis[i+1], 64)
        if err != nil {
            return err
        }
        k.Start[i] = tmp
    }
    for i:=0; i<3; i++ {
        tmp, err := strconv.ParseFloat(lis[i+4], 64)
        if err != nil {
            return err
        }
        k.End[i] = tmp
    }
    frame.Kijuns[lis[0]] = k
    return nil
}
// }}}


// Write
// WriteInp// {{{
func (frame *Frame) WriteInp(fn string) error {
    var nums, otp bytes.Buffer
    var pkeys, skeys, nkeys, ekeys []int
    var pnum, snum, nnum, enum int
    fmt.Printf("Save: %s\n",fn)
    // Frame
    otp.WriteString(fmt.Sprintf("BASE    %5.3f\n", frame.Base))
    otp.WriteString(fmt.Sprintf("LOCATE  %5.3f\n", frame.Locate))
    otp.WriteString(fmt.Sprintf("TFACT   %5.3f\n", frame.Tfact))
    otp.WriteString(fmt.Sprintf("GPERIOD %5.3f\n\n", frame.Gperiod))
    otp.WriteString(fmt.Sprintf("GFACT %.1f\n", frame.View.Gfact))
    otp.WriteString(fmt.Sprintf("FOCUS %.1f %.1f %.1f\n", frame.View.Focus[0], frame.View.Focus[1], frame.View.Focus[2]))
    otp.WriteString(fmt.Sprintf("ANGLE %.1f %.1f\n", frame.View.Angle[0], frame.View.Angle[1]))
    otp.WriteString(fmt.Sprintf("DISTS %.1f %.1f\n\n", frame.View.Dists[0], frame.View.Dists[1]))
    // Prop
    for k := range(frame.Props) {
        pkeys = append(pkeys, k)
    }
    sort.Ints(pkeys)
    for _, k := range(pkeys) {
        otp.WriteString(frame.Props[k].InpString())
        pnum++
    }
    otp.WriteString("\n")
    // Sect
    for k := range(frame.Sects) {
        skeys = append(skeys, k)
    }
    sort.Ints(skeys)
    for _, k := range(skeys) {
        if k > 900 { break }
        otp.WriteString(frame.Sects[k].InpString())
        snum++
    }
    otp.WriteString("\n")
    // Node
    for k := range(frame.Nodes) {
        nkeys = append(nkeys, k)
    }
    sort.Ints(nkeys)
    for _, k := range(nkeys) {
        otp.WriteString(frame.Nodes[k].InpString())
        nnum++
    }
    // sort.Sort(NodeByNum{frame.Nodes})
    // for _, n := range(frame.Nodes) {
    //     otp.WriteString(n.InpString())
    // }
    otp.WriteString("\n")
    // Elem
    for k := range(frame.Elems) {
        ekeys = append(ekeys, k)
    }
    sort.Ints(ekeys)
    for _, k := range(ekeys) {
        if frame.Elems[k].Etype == WBRACE || frame.Elems[k].Etype == SBRACE { continue }
        otp.WriteString(frame.Elems[k].InpString())
        enum++
    }
    nums.WriteString(fmt.Sprintf("%s\n", frame.Title))
    nums.WriteString(fmt.Sprintf("NNODE %d\n", nnum))
    nums.WriteString(fmt.Sprintf("NELEM %d\n", enum))
    nums.WriteString(fmt.Sprintf("NPROP %d\n", pnum))
    nums.WriteString(fmt.Sprintf("NSECT %d\n\n", snum))
    // Write
    w, err := os.Create(fn)
    defer w.Close()
    if err != nil {
        return err
    }
    nums.WriteTo(w)
    otp.WriteTo(w)
    return nil
}
// }}}


// WriteOutput// {{{
func (frame *Frame) WriteOutput (fn string, p string) error {
    var otp bytes.Buffer
    var nkeys, ekeys []int
    // Elem
    otp.WriteString("\n\n** FORCES OF MEMBER\n\n")
    otp.WriteString("  NO   KT NODE         N        Q1        Q2        MT        M1        M2\n\n")
    for k := range(frame.Elems) {
        ekeys = append(ekeys, k)
    }
    sort.Ints(ekeys)
    for _, k := range(ekeys) {
        if !frame.Elems[k].IsLineElem() { continue }
        otp.WriteString(frame.Elems[k].OutputStress(p))
    }
    // Node
    otp.WriteString("\n\n** DISPLACEMENT OF NODE\n\n")
    otp.WriteString("  NO          U          V          W         KSI         ETA       OMEGA\n\n")
    for k := range(frame.Nodes) {
        nkeys = append(nkeys, k)
    }
    sort.Ints(nkeys)
    for _, k := range(nkeys) {
        otp.WriteString(frame.Nodes[k].OutputDisp(p))
    }
    otp.WriteString("\n\n** REACTION\n\n")
    otp.WriteString("  NO  DIRECTION              R    NC\n\n")
    for _, k := range(nkeys) {
        for i:=0; i<6; i++ {
            if frame.Nodes[k].Conf[i] {
                otp.WriteString(frame.Nodes[k].OutputReaction(p, i))
            }
        }
    }
    // Write
    w, err := os.Create(fn)
    defer w.Close()
    if err != nil {
        return err
    }
    otp.WriteTo(w)
    return nil
}
// }}}


func (frame *Frame) Distance (n1, n2 *Node) (dx, dy, dz, d float64) {
    dx = n2.Coord[0] - n1.Coord[0]
    dy = n2.Coord[1] - n1.Coord[1]
    dz = n2.Coord[2] - n1.Coord[2]
    d = math.Sqrt(math.Pow(dx, 2) + math.Pow(dy, 2) + math.Pow(dz, 2))
    return
}

func (frame *Frame) Direction (n1, n2 *Node, normalize bool) []float64 {
    var l float64
    d := make([]float64, 3)
    for i:=0; i<3; i++ {
        d[i] = n2.Coord[i] - n1.Coord[i]
    }
    if normalize {
        for i:=0; i<3; i++ {
            l+=d[i]*d[i]
        }
        l = math.Sqrt(l)
        for i:=0; i<3; i++ {
            d[i] /= l
        }
        return d
    } else {
        return d
    }
}

func (frame *Frame) Move (x, y, z float64) {
    for _, n := range frame.Nodes {
        n.Move(x, y, z)
    }
}

func (frame *Frame) Rotate (center, vector []float64, angle float64) {
    for _, n := range frame.Nodes {
        n.Rotate(center, vector, angle)
    }
}

func (frame *Frame) DefaultSect () *Sect {
    snums := make([]int, len(frame.Sects))
    i := 0
    for k, _ := range(frame.Sects) {
        snums[i] = int(k)
        i++
    }
    sort.Ints(snums)
    return frame.Sects[snums[0]]
}


// Add// {{{
func (frame *Frame) AddSect (num int) *Sect {
    sec := NewSect()
    sec.Num = num
    frame.Sects[num] = sec
    return sec
}

func (frame *Frame) AddNode (x, y, z float64) *Node {
    node := NewNode()
    node.Coord[0] = x
    node.Coord[1] = y
    node.Coord[2] = z
    frame.Maxnnum++
    node.Num = frame.Maxnnum
    frame.Nodes[node.Num] = node
    return node
}

func (frame *Frame) SearchNode (x, y, z float64) *Node {
    for _, n := range(frame.Nodes) {
        if math.Sqrt(math.Pow(x-n.Coord[0],2) + math.Pow(y-n.Coord[1],2) + math.Pow(z-n.Coord[2],2))<=1e-4 {
            return n
        }
    }
    return nil
}

func (frame *Frame) CoordNode (x, y, z float64) *Node {
    for _, n := range(frame.Nodes) {
        if math.Sqrt(math.Pow(x-n.Coord[0],2) + math.Pow(y-n.Coord[1],2) + math.Pow(z-n.Coord[2],2))<=1e-4 {
            return n
        }
    }
    return frame.AddNode(x, y, z)
}

func (frame *Frame) AddElem (enum int, el *Elem) {
    if enum < 0 {
        frame.Maxenum++
        el.Frame = frame
        el.Num = frame.Maxenum
        frame.Elems[el.Num] = el
    } else {
        if _, exist := frame.Elems[enum]; exist {
            fmt.Printf("AddElem: Elem %d already exists\n")
            frame.AddElem(-1, el)
        } else {
            el.Num = enum
            el.Frame= frame
            frame.Elems[el.Num] = el
        }
    }
}

func (frame *Frame) AddLineElem (enum int, ns []*Node, sect *Sect, etype int) (elem *Elem) {
    elem = NewLineElem(ns, sect, etype)
    frame.AddElem(enum, elem)
    return elem
}

func (frame *Frame) AddPlateElem (enum int, ns []*Node, sect *Sect, etype int) (elem *Elem) {
    elem = NewPlateElem(ns, sect, etype)
    frame.AddElem(enum, elem)
    return elem
}
// }}}


// Search// {{{
func (frame *Frame) NodeInBox (n1, n2 *Node) []*Node {
    var minx, miny, minz float64
    var maxx, maxy, maxz float64
    if n1.Coord[0] < n2.Coord[0] {
        minx = n1.Coord[0]; maxx = n2.Coord[0]
    } else {
        minx = n2.Coord[0]; maxx = n1.Coord[0]
    }
    if n1.Coord[1] < n2.Coord[1] {
        miny = n1.Coord[1]; maxy = n2.Coord[1]
    } else {
        miny = n2.Coord[1]; maxy = n1.Coord[1]
    }
    if n1.Coord[2] < n2.Coord[2] {
        minz = n1.Coord[2]; maxz = n2.Coord[2]
    } else {
        minz = n2.Coord[2]; maxz = n1.Coord[2]
    }
    rtn := make([]*Node, 0)
    var i int
    for _, n := range frame.Nodes {
        if minx <= n.Coord[0] && n.Coord[0] <= maxx && miny <= n.Coord[1] && n.Coord[1] <= maxy && minz <= n.Coord[2] && n.Coord[2] <= maxz {
            rtn = append(rtn, n)
            i++
        }
    }
    return rtn[:i]
}

func (frame *Frame) SearchElem (ns... *Node) []*Elem{
    els := make([]*Elem, 0)
    num := 0
    l := len(ns)
    for _, el := range(frame.Elems) {
        count := 0
        found := make([]bool, len(el.Enod))
        loopse:
            for _, n := range(ns) {
                for i, en := range(el.Enod) {
                    if found[i] { continue }
                    if en == n {
                        found[i] = true
                        count++
                        continue loopse
                    }
                }
            }
        if count == l { els = append(els, el); num++ }
    }
    return els[:num]
}

func (frame *Frame) NodeToElemAny (ns... *Node) []*Elem {
    els := make([]*Elem, 0)
    num := 0
    for _, el := range(frame.Elems) {
        loop:
            for _, en := range(el.Enod) {
                for _, n := range(ns) {
                    if en == n { els = append(els, el); num++; break loop }
                }
            }
    }
    return els[:num]
}

func (frame *Frame) NodeToElemAll (ns... *Node) []*Elem {
    var add, found bool
    num := 0
    els := make([]*Elem, 0)
    for _, el := range(frame.Elems) {
        add = true
        for _, en := range(el.Enod) {
            found = false
            for _, n := range(ns) {
                if en == n { found = true; break }
            }
            if !found { add=false; break }
        }
        if add { els = append(els, el); num++ }
    }
    return els[:num]
}

func (frame *Frame) ElemToNode (els... *Elem) []*Node {
    var add bool
    ns := make([]*Node, 0)
    for _, el := range(els) {
        for _, en := range(el.Enod) {
            add = true
            for _, n := range(ns) {
                if en == n { add=false; break }
            }
            if add { ns = append(ns, en) }
        }
    }
    return ns
}

func abs (val int) int {
    if val >= 0 {
        return val
    } else {
        return -val
    }
}
func (frame *Frame) Fence (axis int, coord float64, plate bool) []*Elem {
    rtn := make([]*Elem, 0)
    for _, el := range frame.Elems {
        if el.Hide { continue }
        if plate || el.IsLineElem() {
            sign := 0
            for i, en := range(el.Enod) {
                if en.Coord[axis]-coord > 0 {
                    sign++
                } else {
                    sign--
                }
                if i+1 != abs(sign) {
                    rtn = append(rtn, el)
                    break
                }
            }
        }
    }
    return rtn
}

func (frame *Frame) Cutter (axis int, coord float64) error {
    for _, el := range frame.Fence(axis, coord, false) {
        _, _, err := el.DivideAtAxis(axis, coord)
        if err != nil { return err }
    }
    return nil
}

func (frame *Frame) LineConnected (n *Node) []*Node {
    var add bool
    els := frame.SearchElem(n)
    ns := make([]*Node, 0)
    for _, el := range(els) {
        if el.IsLineElem() {
            for _, en := range(el.Enod) {
                if en == n { continue }
                add = true
                for _, n := range(ns) {
                    if en == n { add=false; break }
                }
                if add { ns = append(ns, en) }
            }
        }
    }
    return ns
}

func (frame *Frame) Connected (n *Node) []*Node {
    i := 0
    els := frame.SearchElem(n)
    ns := frame.ElemToNode(els...)
    rtn := make([]*Node, len(ns)-1)
    for _, val := range(ns) {
        if val!=n { rtn[i] = val; i++ }
    }
    return rtn
}

// TODO: check if this func works as intended
func (frame *Frame) SearchBraceSect (f *Fig, t int) *Sect {
    for _, sec := range frame.Sects {
        if sec.Num <= 900 { continue }
        if (sec.Type == t) && (sec.Figs[0].Prop == f.Prop) &&
           (sec.Figs[0].Value["AREA"] == f.Value["AREA"]) &&
           (sec.Figs[0].Value["IXX"] == 0.0) && (sec.Figs[0].Value["IYY"] == 0.0) {
            return sec
        }
    }
    return nil
}
// }}}


// Modify Frame// {{{
func (frame *Frame) NodeNoReference() []*Node {
    nnums := make(map[int]int, len(frame.Nodes))
    for _, n := range frame.Nodes {
        nnums[n.Num] = 0
    }
    for _, el := range frame.Elems {
        for _, en := range el.Enod {
            nnums[en.Num]++
        }
    }
    rtn := make([]*Node, 0)
    for num, ref := range nnums {
        if ref == 0 {
            rtn = append(rtn, frame.Nodes[num])
        }
    }
    return rtn
}

func (frame *Frame) ElemSameNode() []*Elem {
    rtn := make([]*Elem, 0)
    for _, el := range frame.Elems {
        loop:
            for i, en := range el.Enod[:el.Enods-1] {
                for _, em := range el.Enod[i+1:] {
                    if en == em { rtn = append(rtn, el); break loop }
                }
            }
    }
    return rtn
}

func (frame *Frame) NodeDuplication(eps float64) map[*Node]*Node {
    dups := make(map[*Node]*Node, 0)
    keys := make([]int, len(frame.Nodes))
    i := 0
    for _, k := range(frame.Nodes) {
        if k != nil {
            keys[i] = k.Num
            i++
        }
    }
    sort.Ints(keys)
    for j, k := range(keys[:i]) {
        if _, ok := dups[frame.Nodes[k]]; ok { continue }
        loop:
            for _, m := range(keys[j+1:i]) {
                for n:=0; n<3; n++ {
                    if math.Abs(frame.Nodes[k].Coord[n] - frame.Nodes[m].Coord[n]) > eps { continue loop }
                }
                dups[frame.Nodes[m]] = frame.Nodes[k]
            }
    }
    return dups
}

func (frame *Frame) ReplaceNode(nmap map[*Node]*Node) {
    for _, el := range frame.Elems {
        for i, en := range el.Enod {
            for k, v := range nmap {
                if en == k {
                    el.Enod[i] = v
                    break
                }
            }
        }
    }
    for k := range nmap {
        delete(frame.Nodes, k.Num)
    }
}

func (frame *Frame) Cat (e1, e2 *Elem, n *Node) error {
    if !e1.IsLineElem() || !e2.IsLineElem() { return NotLineElem("Cat") }
    var ind1, ind2 int
    for i, en := range e1.Enod {
        if en == n {
            ind1 = i
            break
        }
    }
    for i, en := range e2.Enod {
        if en == n {
            ind2 = 1-i
            break
        }
    }
    e1.Enod[ind1] = e2.Enod[ind2]
    for j:=0; j<6; j++ {
        e1.Bonds[6*ind1+j] = e2.Bonds[6*ind1+j]
    }
    delete(frame.Nodes, n.Num)
    delete(frame.Elems, e2.Num)
    return nil
}

func (frame *Frame) JoinLineElem (e1, e2 *Elem, parallel bool) error {
    if !e1.IsLineElem() || !e2.IsLineElem() { return NotLineElem("JoinLineElem") }
    if parallel && !IsParallel(e1.Direction(true), e2.Direction(true), 1e-4) { return NotParallel("JoinLineElem") }
    for i, en1 := range e1.Enod {
        for _, en2 := range e2.Enod {
            if en1 == en2 {
                for _, el := range frame.SearchElem(e1.Enod[i]) {
                    if el == e1 || el == e2 { continue }
                    return errors.New(fmt.Sprintf("JoinLineElem: NODE %d has more than 2 elements", e1.Enod[i].Num))
                }
                return e1.Frame.Cat(e1, e2, e1.Enod[i])
            }
        }
    }
    return errors.New("JoinLineElem: No Common Enod")
}

func (frame *Frame) JoinPlateElem (e1, e2 *Elem) error {
    if e1.IsLineElem() || e2.IsLineElem() { return NotPlateElem("JoinPlateElem") }
    var n2 *Node
    for i, en1 := range e1.Enod {
        for j, en2 := range e2.Enod {
            if en1 == en2 {
                if i == e1.Enods-1 {
                    n2 = e1.Enod[0]
                } else {
                    n2 = e1.Enod[i+1]
                }
                for h, en3 := range e2.Enod {
                    if n2 == en3 {
                        switch h - j {
                        case 1, 1-e2.Enods:
                            for k:=0; k<2; k++ {
                                num1 := i+k
                                if num1 > e1.Enods { num1 -= e1.Enods}
                                num2 := j-k-1
                                if num2 < 0 { num2 += e2.Enods }
                                e1.Enod[num1] = e2.Enod[num2]
                            }
                        case -1, e2.Enods-1:
                            for k:=0; k<2; k++ {
                                num1 := i+k
                                if num1 > e1.Enods { num1 -= e1.Enods}
                                num2 := j+k+1
                                if num2 > e2.Enods { num2 -= e2.Enods }
                                e1.Enod[num1] = e2.Enod[num2]
                            }
                        }
                        delete(frame.Elems, e2.Num)
                        return nil
                    }
                }
                if i == 0 {
                    n2 = e1.Enod[e1.Enods-1]
                } else {
                    n2 = e1.Enod[i-1]
                }
                for h, en3 := range e2.Enod {
                    if n2 == en3 {
                        switch h - j {
                        case 1, 1-e2.Enods:
                            for k:=0; k<2; k++ {
                                num1 := i-k
                                if num1 < 0 { num1 += e1.Enods}
                                num2 := j-k-1
                                if num2 < 0 { num2 += e2.Enods }
                                e1.Enod[num1] = e2.Enod[num2]
                            }
                        case -1, e2.Enods-1:
                            for k:=0; k<2; k++ {
                                num1 := i-k
                                if num1 < 0 { num1 += e1.Enods}
                                num2 := j+k+1
                                if num2 > e2.Enods { num2 -= e2.Enods }
                                e1.Enod[num1] = e2.Enod[num2]
                            }
                        }
                        delete(frame.Elems, e2.Num)
                        return nil
                    }
                }
                return errors.New(fmt.Sprintf("JoinPlateElem: Only 1 Common Enod %d", en1.Num))
            }
        }
    }
    return errors.New("JoinPlateElem: No Common Enod")
}

func (frame *Frame) CatByNode (n *Node, parallel bool) error {
    els := frame.SearchElem(n)
    var d []float64
    var num int
    cat := make([]*Elem, 2)
    for _, el := range els {
        if el != nil {
            num++
            if num > 2 { return errors.New(fmt.Sprintf("CatByNode: NODE %d has more than 2 elements", n.Num)) }
            if !el.IsLineElem() { return errors.New(fmt.Sprintf("CatByNode: NODE %d has WALL/SLAB", n.Num)) }
            tmpd := el.Direction(false)
            if d != nil {
                if parallel && !IsParallel(d, tmpd, 1e-4) {
                    return NotParallel("CatByNode")
                }
            }
            cat[num-1] = el
            d = tmpd
        }
    }
    frame.Cat(cat[0], cat[1], n)
    return nil
}

func (frame *Frame) Intersect (e1, e2 *Elem, cross bool, sign1, sign2 int, del1, del2 bool) ([]*Node, []*Elem, error) {
    if !e1.IsLineElem() || !e2.IsLineElem() { return nil, nil, NotLineElem("Intersect") }
    k1, k2, d, err := DistLineLine(e1.Enod[0].Coord, e1.Direction(false), e2.Enod[0].Coord, e2.Direction(false))
    if err == nil {
        if d > 1e-4 {
            return nil, nil, errors.New(fmt.Sprintf("Intersect: Distance= %.3f", d))
        }
        if !cross || (( 0.0 < k1 && k1 < 1.0) && (0.0 < k2 && k2 < 1.0)) {
            var ns []*Node
            var els []*Elem
            var err error
            d1 := e1.Direction(false)
            n := frame.CoordNode(e1.Enod[0].Coord[0]+k1*d1[0], e1.Enod[0].Coord[1]+k1*d1[1], e1.Enod[0].Coord[2]+k1*d1[2])
            switch {
            default:
            case k1 < 0.0:
                ns, els, err = e1.DivideAtNode(n, 0, del1)
            case 0.0 < k1 && k1 < 1.0:
                ns, els, err = e1.DivideAtNode(n, 1*sign1, del1)
            case 1.0 < k1:
                ns, els, err = e1.DivideAtNode(n, 2, del1)
            }
            switch {
            default:
            case k2 < 0.0:
                ns, els, err = e2.DivideAtNode(n, 0, del2)
            case 0.0 < k2 && k2 < 1.0:
                ns, els, err = e2.DivideAtNode(n, 1*sign2, del2)
            case 1.0 < k2:
                ns, els, err = e2.DivideAtNode(n, 2, del2)
            }
            if err != nil {
                return nil, nil, err
            } else {
                return ns, els, nil
            }
        } else {
            return nil, nil, errors.New(fmt.Sprintf("Intersect: Not Cross k1= %.3f, k2= %.3f", k1, k2))
        }
    } else {
        return nil, nil, err
    }
}

func (frame *Frame) Trim (e1, e2 *Elem, sign int) ([]*Node, []*Elem, error) {
    return frame.Intersect(e1, e2, true, 1, sign, false, true)
}

func (frame *Frame) Extend (e1, e2 *Elem) ([]*Node, []*Elem, error) {
    return frame.Intersect(e1, e2, false, 1, 1, false, true)
}

func (frame *Frame) Upside () {
    for _, el := range frame.Elems {
        el.Enod = Upside(el.Enod)
    }
}
// }}}


// ExtractArclm// {{{
func (frame *Frame) ExtractArclm () {
    frame.WeightDistribution()
    for _, el := range frame.Elems {
        if !el.IsLineElem() {
            brs := el.RectToBrace(2, 1.0)
            if brs != nil {
                for _, br := range brs {
                    frame.AddElem(-1, br)
                    el.Adopt(br)
                }
            }
        }
    }
    var ekeys []int
    snum := 0
    for _, sect := range frame.Sects {
        if sect.HasArea() { snum++ }
    }
    for k := range frame.Elems {
        ekeys = append(ekeys, k)
    }
    sort.Ints(ekeys)
    for _, k := range ekeys {
        el := frame.Elems[k]
        if el.Sect.Type == 0 {
            el.Sect.Type = el.Etype
            snum--
            if snum < 0 {
                break
            }
        }
    }
}

func (frame *Frame) WeightDistribution () {
    var otp bytes.Buffer
    var nkeys, ekeys []int
    amount := make(map[int]float64)
    total := make([]float64, 3)
    otp.WriteString("3.2 : 節点重量\n\n")
    otp.WriteString("節点ごとに重量を集計した結果を記す。\n")
    otp.WriteString("柱，壁は階高の中央で上下に分配するものとする。\n\n")
    otp.WriteString(fmt.Sprintf(" 節点番号          積載荷重別の重量 [%s]\n\n", frame.Show.UnitName[0]))
    otp.WriteString("                 床用     柱梁用     地震用\n")
    for _, el := range frame.Elems {
        el.Distribute()
        if el.Etype != WBRACE || el.Etype != SBRACE {
            amount[el.Sect.Num] += el.Amount()
        }
    }
    for k := range(frame.Nodes) {
        nkeys = append(nkeys, k)
    }
    sort.Ints(nkeys)
    for _, k := range(nkeys) {
        otp.WriteString(frame.Nodes[k].WgtString())
        for i:=0; i<3; i++ {
            total[i] += frame.Nodes[k].Weight[i]
        }
    }
    otp.WriteString(fmt.Sprintf("\n       計  %10.3f %10.3f %10.3f\n\n", total[0], total[1], total[2]))
    otp.WriteString("各断面の部材総量（参考資料）\n\n")
    otp.WriteString(" 断面番号   長さ,面積[m,m2]\n")
    for k := range(amount) {
        ekeys = append(ekeys, k)
    }
    sort.Ints(ekeys)
    for _, k := range ekeys {
        otp.WriteString(fmt.Sprintf("%9d %9.3f\n", k, amount[k]))
    }
    otp.WriteString("\n")
    switch frame.Show.UnitName[0] {
    default:
        otp.WriteString(fmt.Sprintf("Unit Factor  =%7.5f \"[%s]\"\n\n", frame.Show.Unit[0], frame.Show.UnitName[0]))
    case "tf":
        otp.WriteString(fmt.Sprintf("Unit Factor  =%7.5f \"Classic Units [%s]\"\n\n", frame.Show.Unit[0], frame.Show.UnitName[0]))
    case "kN":
        otp.WriteString(fmt.Sprintf("Unit Factor  =%7.5f \"SI Units [%s]\"\n\n", frame.Show.Unit[0], frame.Show.UnitName[0]))
    }
    otp.WriteString(frame.Ai())
    w, err := os.Create(filepath.Join(frame.Home, DEFAULT_WGT))
    defer w.Close()
    if err != nil {
        return
    }
    otp.WriteTo(w)
}

func (frame *Frame) Ai () string {
    size := len(frame.Level) + 1
    weight := make([]float64, size)
    level  := make([]float64, size)
    nnum   := make([]int, size)
    maxheight := MINCOORD
    for _, n := range frame.Nodes {
        height := n.Coord[2]
        if height < frame.Level[0] {
            weight[0] += n.Weight[2]
            level[0] += n.Coord[2]
            nnum[0]++
        } else if height >= frame.Level[size-2] {
            weight[size-1] += n.Weight[2]
            level[size-1] += n.Coord[2]
            nnum[size-1]++
        } else {
            for i:=1; i<size-1; i++ {
                if height < frame.Level[i] {
                    weight[i] += n.Weight[2]
                    level[i] += n.Coord[2]
                    nnum[i]++
                    break
                }
            }
        }
    }
    W := make([]float64, size)
    for i:=0; i<size; i++ {
        level[i] /= float64(nnum[i])
        if level[i] > maxheight { maxheight = level[i] }
        for j:=size-1; j>=i; j-- {
            W[i] += weight[j]
        }
    }
    Ai := make([]float64, size-1)
    T := maxheight*frame.Tfact
    var Rt float64
    if T < frame.Gperiod {
        Rt = 1.0
    } else if T < 2.0 * frame.Gperiod {
        Rt = 1.0 - 0.2 * math.Pow((T/frame.Gperiod - 1.0), 2.0)
    } else {
        Rt = 1.6 * frame.Gperiod / T
    }
    tt := 2.0 * T / (1.0 + 3.0 * T)
    for i:=0; i<size-1; i++ {
        alpha := W[i+1]/W[1]
        Ai[i] = 1.0 + (1.0/math.Sqrt(alpha) - alpha) *tt
    }
    Ci := make([]float64, size)
    Qi := make([]float64, size)
    Hi := make([]float64, size)
    facts := make([]float64, size)
    for i:=0; i<size; i++ {
        if i==0 {
            Ci[0] = 0.5 * frame.Locate * Rt * frame.Base
            Qi[0] = Ci[0] * W[0]
            facts[0] = Ci[0]
        } else {
            Ci[i] = frame.Locate * Rt * Ai[i-1] * frame.Base
            Qi[i] = Ci[i] * W[i]
            Hi[i-1] = Qi[i-1] - Qi[i]
            if i > 1 { facts[i-1] = Hi[i-1] / weight[i-1] }
        }
    }
    Hi[size-1] = Qi[size-1]
    facts[size-1] = Hi[size-1] / weight[size-1]
    for _, n := range frame.Nodes {
        height := n.Coord[2]
        if height < frame.Level[0] {
            n.Factor = facts[0]
        } else if height >= frame.Level[size-2] {
            n.Factor = facts[size-1]
        } else {
            for i:=1; i<size-1; i++ {
                if height < frame.Level[i] {
                    n.Factor = facts[i]
                    break
                }
            }
        }
    }
    var rtn bytes.Buffer
    rtn.WriteString("3.3 : Ai分布型地震荷重\n\n")
    rtn.WriteString("水平荷重は建築基準法施行令第88条および建設省告示1793号に従い、Ａi分布型の地震力とする。\n\n")
    rtn.WriteString(fmt.Sprintf("階数       　　　    n =%d\n", size))
    rtn.WriteString(fmt.Sprintf("高さ         　　　  H =%.3f\n", maxheight))
    rtn.WriteString(fmt.Sprintf("１次固有周期         T1=%5.3fH=%5.3f\n", frame.Tfact, T))
    rtn.WriteString(fmt.Sprintf("地盤周期             Tc=%5.3f\n", frame.Gperiod))
    rtn.WriteString(fmt.Sprintf("振動特性係数         Rt=%5.3f\n", Rt))
    rtn.WriteString(fmt.Sprintf("地域係数             Z =%5.3f\n", frame.Locate))
    rtn.WriteString(fmt.Sprintf("標準層せん断力係数   Co=%5.3f\n", frame.Base))
    rtn.WriteString(fmt.Sprintf("基礎部分の震度       Cf=%5.3f\n\n", facts[0]))
    rtn.WriteString("各階平均高さ      :")
    for i:=0; i<size; i++ {
        rtn.WriteString(fmt.Sprintf(" %10.3f", level[i]))
    }
    rtn.WriteString("\n各階重量       wi :")
    for i:=0; i<size; i++ {
        rtn.WriteString(fmt.Sprintf(" %10.3f", weight[i]))
    }
    rtn.WriteString("\n        Wi = Σwi :")
    for i:=0; i<size; i++ {
        rtn.WriteString(fmt.Sprintf(" %10.3f", W[i]))
    }
    rtn.WriteString("\n               Ai :           ")
    for i:=0; i<size-1; i++ {
        rtn.WriteString(fmt.Sprintf(" %10.3f", Ai[i]))
    }
    rtn.WriteString("\n層せん断力係数 Ci :           ")
    for i:=1; i<size; i++ {
        rtn.WriteString(fmt.Sprintf(" %10.3f", Ci[i]))
    }
    rtn.WriteString("\n層せん断力     Qi :           ")
    for i:=1; i<size; i++ {
        rtn.WriteString(fmt.Sprintf(" %10.3f", Qi[i]))
    }
    rtn.WriteString("\n各階外力       Hi :           ")
    for i:=1; i<size; i++ {
        rtn.WriteString(fmt.Sprintf(" %10.3f", Hi[i]))
    }
    rtn.WriteString("\n外力係数    Hi/wi :")
    for i:=0; i<size; i++ {
        rtn.WriteString(fmt.Sprintf(" %10.3f", facts[i]))
    }
    rtn.WriteString("\n")
    return rtn.String()
}

func (frame *Frame) SaveAsArclm (name string) error {
    if name == "" { name = frame.Path }
    nums := make([]int, 3)
    otp := make([]bytes.Buffer, 3)
    var skeys, nkeys, ekeys []int
    // Sect
    for k := range(frame.Sects) {
        skeys = append(skeys, k)
    }
    sort.Ints(skeys)
    for _, k := range(skeys) {
        if frame.Sects[k].HasArea() {
            str := frame.Sects[k].InlString()
            nums[2]++
            for i:=0; i<3; i++ {
                otp[i].WriteString(str)
            }
        }
    }
    // Node: Coord
    for k := range(frame.Nodes) {
        nkeys = append(nkeys, k)
    }
    sort.Ints(nkeys)
    for _, k := range(nkeys) {
        str := frame.Nodes[k].InlCoordString()
        nums[0]++
        for i:=0; i<3; i++ {
            otp[i].WriteString(str)
        }
    }
    // Elem
    for k := range(frame.Elems) {
        ekeys = append(ekeys, k)
    }
    sort.Ints(ekeys)
    for _, k := range(ekeys) {
        if frame.Elems[k].IsLineElem() {
            for i:=0; i<3; i++ {
                otp[i].WriteString(frame.Elems[k].InlString(i))
            }
            nums[1]++
        }
    }
    // Node: Boundary Condition
    for _, k := range(nkeys) {
        for i:=0; i<3; i++ {
            otp[i].WriteString(frame.Nodes[k].InlConditionString(i))
        }
    }
    numstr := fmt.Sprintf("%5d %5d %5d\n", nums[0], nums[1], nums[2])
    // Write
    for i, ext := range InputExt {
        fn := Ce(name, ext)
        w, err := os.Create(fn)
        defer w.Close()
        if err != nil {
            return err
        }
        w.WriteString(numstr)
        otp[i].WriteTo(w)
    }
    return nil
}
// }}}


func (frame *Frame) Facts (fn string, etypes []int) error {
    var err error
    l := len(frame.Level)+1
    if l == 1 {
        return errors.New("Facts: Level = []")
    }
    nodes := make([][]*Node, l)
    elems := make([][]*Elem, l-1)
    for i:=0; i<l; i++ {
        nodes[i] = make([]*Node, 0)
        if i<l-1 {
            elems[i] = make([]*Elem, 0)
        }
    }
    fact_node:
        for _, n := range frame.Nodes {
            for i:=0; i<l-1; i++ {
                if n.Coord[2] < frame.Level[i] {
                    nodes[i] = append(nodes[i], n)
                    continue fact_node
                }
            }
            nodes[l-1] = append(nodes[l-1], n)
        }
    for _, el := range frame.Elems {
        contained := false
        for _, et := range etypes {
            if el.Etype == et {
                contained = true
                break
            }
        }
        if !contained { continue }
        for i:=0; i<l-1; i++ {
            if (el.Enod[0].Coord[2] - frame.Level[i])*(el.Enod[1].Coord[2] - frame.Level[i]) < 0 {
                elems[i] = append(elems[i], el)
                break
            }
        }
    }
    f := NewFact(l, true, frame.Base/0.2)
    f.SetFileName([]string{frame.DataFileName["L"], frame.DataFileName["X"], frame.DataFileName["Y"]},
                  []string{frame.ResultFileName["L"], frame.ResultFileName["X"], frame.ResultFileName["Y"]})
    err = f.CalcFact(nodes, elems)
    if err != nil {
        return err
    }
    fmt.Println(f)
    err = f.WriteTo(fn)
    if err != nil {
        return err
    }
    return nil
}


// Modify View// {{{
func (frame *Frame) SetFocus() {
    xmin, xmax, ymin, ymax, zmin, zmax := frame.Bbox()
    mins := []float64{ xmin, ymin, zmin }
    maxs := []float64{ xmax, ymax, zmax }
    for i:=0; i<3; i++ {
        frame.View.Focus[i] = 0.5*(mins[i]+maxs[i])
    }
}
// }}}


// Projection// {{{
// direction: 0 -> origin=bottomleft, x=[1,0], y=[0,1]
//            1 -> origin=topleft,    x=[1,0], y=[0,-1]
func (view *View) Set(direction int) {
    a0 := view.Angle[0]*math.Pi / 180 // phi
    a1 := view.Angle[1]*math.Pi / 180 // theta
    c0 := math.Cos(a0); s0 := math.Sin(a0)
    c1 := math.Cos(a1); s1 := math.Sin(a1)
    if direction == 0 {
        view.Viewpoint[0][0] = c1*c0
        view.Viewpoint[0][1] = s1*c0
        view.Viewpoint[0][2] = s0
        view.Viewpoint[1][0] = -s1
        view.Viewpoint[1][1] = c1
        view.Viewpoint[1][2] = 0.0
        view.Viewpoint[2][0] = -c1*s0
        view.Viewpoint[2][1] = -s1*s0
        view.Viewpoint[2][2] = c0
    } else if direction == 1 {
        view.Viewpoint[0][0] = c1*c0
        view.Viewpoint[0][1] = s1*c0
        view.Viewpoint[0][2] = s0
        view.Viewpoint[1][0] = -s1
        view.Viewpoint[1][1] = c1
        view.Viewpoint[1][2] = 0.0
        view.Viewpoint[2][0] = c1*s0
        view.Viewpoint[2][1] = s1*s0
        view.Viewpoint[2][2] = -c0
    }
}

func (view *View) ProjectCoord (coord []float64) []float64 {
    rtn := make([]float64, 2)
    p  := make([]float64, 3)
    pv := make([]float64, 3)
    pc := make([]float64, 2)
    for i:=0; i<3; i++ {
        p[i] = coord[i] - view.Focus[i]
        pv[i] = view.Viewpoint[0][i]*view.Dists[0] - p[i]
    }
    for i:=0; i<2; i++ {
        pc[i] = Dot(view.Viewpoint[i+1], p, 3)
    }
    if view.Perspective {
        vnai := Dot(view.Viewpoint[0], pv, 3)
        for i:=0; i<2; i++ {
            rtn[i] = view.Gfact*view.Dists[1]*pc[i]/vnai + view.Center[i]
        }
    } else {
        for i:=0; i<2; i++ {
            rtn[i] = view.Gfact*pc[i] + view.Center[i]
        }
    }
    return rtn
}

func (view *View) ProjectNode (node *Node) {
    p  := make([]float64, 3)
    pv := make([]float64, 3)
    pc := make([]float64, 2)
    for i:=0; i<3; i++ {
        p[i] = node.Coord[i] - view.Focus[i]
        pv[i] = view.Viewpoint[0][i]*view.Dists[0] - p[i]
    }
    for i:=0; i<2; i++ {
        pc[i] = Dot(view.Viewpoint[i+1], p, 3)
    }
    if view.Perspective {
        vnai := Dot(view.Viewpoint[0], pv, 3)
        for i:=0; i<2; i++ {
            node.Pcoord[i] = view.Gfact*view.Dists[1]*pc[i]/vnai + view.Center[i]
        }
    } else {
        for i:=0; i<2; i++ {
            node.Pcoord[i] = view.Gfact*pc[i] + view.Center[i]
        }
    }
}

func (view *View) ProjectDeformation (node *Node, show *Show) {
    p  := make([]float64, 3)
    pv := make([]float64, 3)
    pc := make([]float64, 2)
    for i:=0; i<3; i++ {
        p[i] = (node.Coord[i] + show.Dfact * node.ReturnDisp(show.Period, i)) - view.Focus[i]
        pv[i] = view.Viewpoint[0][i]*view.Dists[0] - p[i]
    }
    for i:=0; i<2; i++ {
        pc[i] = Dot(view.Viewpoint[i+1], p, 3)
    }
    if view.Perspective {
        vnai := Dot(view.Viewpoint[0], pv, 3)
        for i:=0; i<2; i++ {
            node.Dcoord[i] = view.Gfact*view.Dists[1]*pc[i]/vnai + view.Center[i]
        }
    } else {
        for i:=0; i<2; i++ {
            node.Dcoord[i] = view.Gfact*pc[i] + view.Center[i]
        }
    }
}

// }}}
