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
    PeriodExt = map[string]string{".otl": "L", ".ohx": "X", ".ohy": "Y"}
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
    View  *View
    Nodes map[int]*Node
    Elems map[int]*Elem
    Props map[int]*Prop
    Sects map[int]*Sect

    Kijuns map[string]*Kijun

    Maxenum int
    Maxnnum int
    Maxsnum int

    Level []float64

    Unit []float64
    UnitName []string
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
    f.Props = make(map[int]*Prop)
    f.Kijuns = make(map[string]*Kijun)
    f.View  = NewView()
    f.Maxnnum = 100
    f.Maxenum = 1000
    f.Maxsnum = 900
    f.Level = make([]float64, 0)
    f.Unit = []float64{1.0, 1.0}
    f.UnitName = []string{"tf", "m"}
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
    viewpoint [][]float64
    Center []float64
}

func NewView() *View {
    v := new(View)
    v.Gfact = 1.0
    v.Focus = make([]float64,3)
    v.Angle = make([]float64,2)
    v.Dists = make([]float64,2)
    v.Dists[0] = 1000; v.Dists[1] = 5000
    v.viewpoint = make([][]float64,3)
    for i:=0; i<3; i++ {
        v.viewpoint[i] = make([]float64, 3)
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
        nv.viewpoint[i] = v.viewpoint[i]
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
func (frame *Frame) ReadInp(filename string, coord []float64) error {
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
                                       nodemap, err = frame.ParseInp(tmp, coord, nodemap)
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
    nodemap, err = frame.ParseInp(tmp, coord, nodemap)
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
        err = frame.ReadConfigure(conffn)
        if err != nil {
            fmt.Println(err)
        }
    } else {
        fmt.Printf("No Configure File: %s\n", conffn)
    }
    return nil
}

func (frame *Frame) ParseInp(lis []string, coord []float64, nodemap map[int]int) (map[int]int, error) {
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
        def, num, err = frame.ParseNode(lis, coord)
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
            for j:=0; j<3; j++ {
                tmpcol, err = strconv.ParseInt(lis[i+1+j],10,64)
                p.Color += int(tmpcol)*int(math.Pow(16,float64(4-2*j)))
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
        case "NZMAX":
            for j:=0; j<12; j++ {
                s.Yield[j], err = strconv.ParseFloat(lis[i+1+2*j], 64)
            }
        case "COLOR":
            var tmpcol int64
            s.Color = 0
            for j:=0; j<3; j++ {
                tmpcol, err = strconv.ParseInt(lis[i+1+j],10,64)
                s.Color += int(tmpcol)*int(math.Pow(16,float64(4-2*j)))
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

func (frame *Frame) ParseNode(lis []string, coord []float64) (int, int, error) {
    var num int64
    var err error
    n := NewNode()
    for i, word := range(lis) {
        switch word {
        case "NODE":
            num, err = strconv.ParseInt(lis[i+1],10,64)
            n.Num = int(num)
        case "CORD":
            for j:=0; j<3; j++ {
                n.Coord[j], err = strconv.ParseFloat(lis[i+1+j],64)
                n.Coord[j] += coord[j]
            }
        case "ICON":
            for j:=0; j<6; j++ {
                if lis[i+1+j]=="0" {
                    n.Conf[j]=false
                } else {
                    n.Conf[j]=true
                }
            }
        case "VCON":
            for j:=0; j<6; j++ {
                n.Load[j], err = strconv.ParseFloat(lis[i+1+j],64)
            }
        }
        if err != nil {
            return 0, 0, err
        }
    }
    newnode := frame.SearchNode(n.Coord[0], n.Coord[1], n.Coord[2])
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
            frame.AddLineElem(ns, sect, sect.Type-1) // TODO: set etype, cang, ...
            // TODO: search parent
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
        if _, ok := frame.Nodes[nnum]; !ok {
            fmt.Printf("Append Node %d\n", nnum)
        }
    }
    return nil
}
// }}}


// ReadResult// {{{
func (frame *Frame) ReadResult (filename string) error {
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
    for {
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
        if elem, ok := frame.Elems[enum]; ok {
            elem.Stress[period] = make(map[int][]float64)
            for i:=0; i<2; i++ {
                num, err := strconv.ParseInt(words[2+7*i], 10, 64)
                if err != nil {
                    return err
                }
                tmp := make([]float64, 6)
                for k:=0; k<6; k++ {
                    val, err := strconv.ParseFloat(words[3+7*i+k], 64)
                    if err != nil {
                        return err
                    }
                    tmp[k] = val
                }
                elem.Stress[period][int(num)]=tmp
            }
        } else {
            fmt.Printf("ELEM %d not found\n", enum)
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
            node.Disp[period]=tmp
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
            node.Reaction[period][ind-1]=val
        } else {
            fmt.Printf("NODE %d not found\n", nnum)
        }
    }
    return nil
}
// }}}


// func (frame *Frame) ReadBuckling (filename string) error {
// }


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
    if err != nil {
        return err
    }
    nums.WriteTo(w)
    otp.WriteTo(w)
    w.Close()
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

func (frame *Frame) AddElem (els... *Elem) {
    for _, el := range els {
        frame.Maxenum++
        el.Frame = frame
        el.Num = frame.Maxenum
        frame.Elems[el.Num] = el
    }
}

func (frame *Frame) AddLineElem (ns []*Node, sect *Sect, etype int) (elem *Elem) {
    elem = NewLineElem(ns, sect, etype)
    frame.AddElem(elem)
    return elem
}

func (frame *Frame) AddPlateElem (ns []*Node, sect *Sect, etype int) (elem *Elem) {
    elem = NewPlateElem(ns, sect, etype)
    frame.AddElem(elem)
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
    if !e1.IsLineElem() || !e2.IsLineElem() { return errors.New("Cat: Not LineElem") }
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
    if !e1.IsLineElem() || !e2.IsLineElem() { return errors.New("JoinLineElem: Not LineElem") }
    if parallel && !IsParallel(e1.Direction(false), e2.Direction(false)) { return errors.New("JoinLineElem: Not Parallel") }
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
                if parallel && !IsParallel(d, tmpd) {
                    return errors.New("CatByNode: Not Parallel")
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
    if !e1.IsLineElem() || !e2.IsLineElem() { return nil, nil, errors.New("Intersect: PlateElem") }
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
                frame.AddElem(brs...)
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
    otp.WriteString(fmt.Sprintf(" 節点番号          積載荷重別の重量 [%s]\n\n", frame.UnitName[0]))
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
    switch frame.UnitName[0] {
    default:
        otp.WriteString(fmt.Sprintf("Unit Factor  =%7.5f \"[%s]\"\n\n", frame.Unit[0], frame.UnitName[0]))
    case "tf":
        otp.WriteString(fmt.Sprintf("Unit Factor  =%7.5f \"Classic Units [%s]\"\n\n", frame.Unit[0], frame.UnitName[0]))
    case "kN":
        otp.WriteString(fmt.Sprintf("Unit Factor  =%7.5f \"SI Units [%s]\"\n\n", frame.Unit[0], frame.UnitName[0]))
    }
    otp.WriteString(frame.Ai())
    w, err := os.Create(filepath.Join(frame.Home, DEFAULT_WGT))
    if err != nil {
        return
    }
    otp.WriteTo(w)
    w.Close()
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
        if err != nil {
            return err
        }
        w.WriteString(numstr)
        otp[i].WriteTo(w)
        w.Close()
    }
    return nil
}
// }}}


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
        view.viewpoint[0][0] = c1*c0
        view.viewpoint[0][1] = s1*c0
        view.viewpoint[0][2] = s0
        view.viewpoint[1][0] = -s1
        view.viewpoint[1][1] = c1
        view.viewpoint[1][2] = 0.0
        view.viewpoint[2][0] = -c1*s0
        view.viewpoint[2][1] = -s1*s0
        view.viewpoint[2][2] = c0
    } else if direction == 1 {
        view.viewpoint[0][0] = c1*c0
        view.viewpoint[0][1] = s1*c0
        view.viewpoint[0][2] = s0
        view.viewpoint[1][0] = -s1
        view.viewpoint[1][1] = c1
        view.viewpoint[1][2] = 0.0
        view.viewpoint[2][0] = c1*s0
        view.viewpoint[2][1] = s1*s0
        view.viewpoint[2][2] = -c0
    }
}

func (view *View) ProjectCoord (coord []float64) []float64 {
    rtn := make([]float64, 2)
    p  := make([]float64, 3)
    pv := make([]float64, 3)
    pc := make([]float64, 2)
    for i:=0; i<3; i++ {
        p[i] = coord[i] - view.Focus[i]
        pv[i] = view.viewpoint[0][i]*view.Dists[0] - p[i]
    }
    for i:=0; i<2; i++ {
        pc[i] = Dot(view.viewpoint[i+1], p, 3)
    }
    if view.Perspective {
        vnai := Dot(view.viewpoint[0], pv, 3)
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
        pv[i] = view.viewpoint[0][i]*view.Dists[0] - p[i]
    }
    for i:=0; i<2; i++ {
        pc[i] = Dot(view.viewpoint[i+1], p, 3)
    }
    if view.Perspective {
        vnai := Dot(view.viewpoint[0], pv, 3)
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
        pv[i] = view.viewpoint[0][i]*view.Dists[0] - p[i]
    }
    for i:=0; i<2; i++ {
        pc[i] = Dot(view.viewpoint[i+1], p, 3)
    }
    if view.Perspective {
        vnai := Dot(view.viewpoint[0], pv, 3)
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
