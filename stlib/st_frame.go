package st

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/mattn/natural"
	"github.com/yofu/dxf"
	dxfcolor "github.com/yofu/dxf/color"
	"github.com/yofu/dxf/drawing"
	dxfentity "github.com/yofu/dxf/entity"
	"github.com/yofu/st/arclm"
	"github.com/yofu/unit"
)

var (
	PeriodExt = map[string]string{".inl": "L", ".otl": "L", ".ihx": "X", ".ohx": "X", ".ihy": "Y", ".ohy": "Y"}
)

// SI unit
const SI = 9.80665

var (
	// PlasticThreshold = math.Pow(arclm.RADIUS, arclm.EXPONENT)
	PlasticThreshold = arclm.RADIUS
)

// Data Extentions
var (
	InputExt  = []string{".inl", ".ihx", ".ihy"}
	OutputExt = []string{".otl", ".ohx", ".ohy"}
)

const (
	// DefaultWgt is the name of default .wgt file
	DefaultWgt = "hogtxt.wgt"
)

// Max/Min Coord
const (
	MINCOORD = -100.0
	MAXCOORD = 1000.0
)

// ReadResult Mode
const (
	UpdateResult = iota
	AddResult
	AddSearchResult
)

// Axis Vector
var (
	XAXIS = []float64{1.0, 0.0, 0.0}
	YAXIS = []float64{0.0, 1.0, 0.0}
	ZAXIS = []float64{0.0, 0.0, 1.0}
)

// Frame : Analysis Frame
type Frame struct {
	Title   string
	Name    string
	Project string
	Path    string
	Home    string

	View   *View
	Nodes  map[int]*Node
	Elems  map[int]*Elem
	Props  map[int]*Prop
	Sects  map[int]*Sect
	Piles  map[int]*Pile
	Bonds  map[int]*Bond
	Chains map[int]*Chain

	Arclms map[string]*arclm.Frame

	Eigenvalue map[int]float64

	Kijuns   map[string]*Kijun
	Measures []*Measure
	Arcs     []*Arc

	LocalAxis *Axis

	Maxenum int
	Maxnnum int
	Maxsnum int

	Nlap map[string]int

	Ai   *Aiparameter
	Wind *Windparameter
	Fes  *Fact

	Show *Show

	DataFileName   map[string]string
	ResultFileName map[string]string
	LstFileName    string

	Lapch chan int
	Endch chan error
}

// NewFrame creates New Frame
func NewFrame() *Frame {
	f := new(Frame)
	f.Title = "\"CREATED ORGAN FRAME.\""
	f.Nodes = make(map[int]*Node)
	f.Elems = make(map[int]*Elem)
	f.Sects = make(map[int]*Sect)
	f.Props = make(map[int]*Prop)
	f.Piles = make(map[int]*Pile)
	f.Bonds = make(map[int]*Bond)
	f.Chains = make(map[int]*Chain)
	f.Arclms = make(map[string]*arclm.Frame)
	f.Eigenvalue = make(map[int]float64)
	f.Kijuns = make(map[string]*Kijun)
	f.Measures = make([]*Measure, 0)
	f.Arcs = make([]*Arc, 0)
	f.View = NewView()
	f.Maxnnum = 100
	f.Maxenum = 1000
	f.Maxsnum = 900
	f.Nlap = make(map[string]int)
	f.Ai = NewAiparameter()
	f.Wind = NewWindparameter()
	f.Show = NewShow(f)
	f.DataFileName = make(map[string]string)
	f.ResultFileName = make(map[string]string)
	f.Lapch = make(chan int)
	f.Endch = make(chan error)
	return f
}

// Aiparameter : Parameter for Ai Distribution
type Aiparameter struct {
	Base     []float64
	Locate   float64
	Tfact    float64
	Gperiod  float64
	T        float64
	Rt       float64
	Nfloor   int
	Boundary []float64
	Level    []float64
	Wi       []float64
	W        []float64
	Ai       []float64
	Ci       [][]float64
	Qi       [][]float64
	Hi       [][]float64
}

// NewAiparameter creates New Aiparameter
// Default C0=0.2
func NewAiparameter() *Aiparameter {
	a := new(Aiparameter)
	a.Base = []float64{0.2, 0.2}
	a.Locate = 1.0
	a.Tfact = 0.02
	a.Gperiod = 0.6
	a.T = 0.0
	a.Rt = 1.0
	a.Nfloor = 0
	a.Boundary = make([]float64, 0)
	a.Level = make([]float64, 0)
	a.Wi = make([]float64, 0)
	a.W = make([]float64, 0)
	a.Ai = make([]float64, 0)
	a.Ci = make([][]float64, 2)
	a.Qi = make([][]float64, 2)
	a.Hi = make([][]float64, 2)
	for i := 0; i < 2; i++ {
		a.Ci[i] = make([]float64, 0)
		a.Qi[i] = make([]float64, 0)
		a.Hi[i] = make([]float64, 0)
	}
	return a
}

// Snapshot takes a Snapshot of Aiparameter
func (ai *Aiparameter) Snapshot() *Aiparameter {
	a := NewAiparameter()
	a.Base = make([]float64, 2)
	for i := 0; i < 2; i++ {
		a.Base[i] = ai.Base[i]
	}
	a.Locate = ai.Locate
	a.Tfact = ai.Tfact
	a.Gperiod = ai.Gperiod
	a.T = ai.T
	a.Rt = ai.Rt
	a.Nfloor = ai.Nfloor
	if ai.Nfloor > 0 {
		a.Boundary = make([]float64, a.Nfloor+1)
		for i := 0; i < a.Nfloor+1; i++ {
			a.Boundary[i] = ai.Boundary[i]
		}
	} else {
		a.Boundary = make([]float64, 0)
	}
	a.Ci = make([][]float64, 2)
	a.Qi = make([][]float64, 2)
	a.Hi = make([][]float64, 2)
	if len(ai.Level) > 0 {
		a.Level = make([]float64, a.Nfloor)
		a.Wi = make([]float64, a.Nfloor)
		a.W = make([]float64, a.Nfloor)
		a.Ai = make([]float64, a.Nfloor-1)
		for i := 0; i < a.Nfloor; i++ {
			a.Level[i] = ai.Level[i]
			a.Wi[i] = ai.Wi[i]
			a.W[i] = ai.W[i]
			if i == a.Nfloor-1 {
				continue
			}
			a.Ai[i] = ai.Ai[i]
		}
		for j := 0; j < 2; j++ {
			a.Ci[j] = make([]float64, a.Nfloor)
			a.Qi[j] = make([]float64, a.Nfloor)
			a.Hi[j] = make([]float64, a.Nfloor)
			for i := 0; i < a.Nfloor; i++ {
				a.Ci[j][i] = ai.Ci[j][i]
				a.Qi[j][i] = ai.Qi[j][i]
				a.Hi[j][i] = ai.Hi[j][i]
			}
		}
	}
	return a
}

type Windparameter struct {
	Roughness int
	Velocity  float64
	Factor    float64
}

func NewWindparameter() *Windparameter {
	return &Windparameter{
		Roughness: 3,
		Velocity:  30.0,
		Factor:    1.0,
	}
}

func (wp *Windparameter) Snapshot() *Windparameter {
	return &Windparameter{
		Roughness: wp.Roughness,
		Velocity:  wp.Velocity,
		Factor:    wp.Factor,
	}
}

// View : Parameter for Model View
type View struct {
	Gfact       float64
	Focus       []float64
	Angle       []float64
	Dists       []float64
	Perspective bool
	Viewpoint   [][]float64
	Center      []float64
}

// NewView creates New View
func NewView() *View {
	v := new(View)
	v.Gfact = 1.0
	v.Focus = make([]float64, 3)
	v.Angle = make([]float64, 2)
	v.Dists = make([]float64, 2)
	v.Dists[0] = 1000
	v.Dists[1] = 60000
	v.Viewpoint = make([][]float64, 3)
	for i := 0; i < 3; i++ {
		v.Viewpoint[i] = make([]float64, 3)
	}
	v.Perspective = true
	v.Center = make([]float64, 2)
	return v
}

// Copy returns a copy of View
func (v *View) Copy() *View {
	nv := NewView()
	nv.Gfact = v.Gfact
	for i := 0; i < 3; i++ {
		nv.Focus[i] = v.Focus[i]
		nv.Viewpoint[i] = v.Viewpoint[i]
	}
	for i := 0; i < 2; i++ {
		nv.Angle[i] = v.Angle[i]
		nv.Dists[i] = v.Dists[i]
		nv.Center[i] = v.Center[i]
	}
	nv.Perspective = v.Perspective
	return nv
}

// Snapshot takes a Snapshot of Frame
func (frame *Frame) Snapshot() *Frame {
	f := NewFrame()
	f.Title = frame.Title
	f.Name = frame.Name
	f.Project = frame.Project
	f.Path = frame.Path
	f.Home = frame.Home
	f.View = frame.View
	for _, p := range frame.Props {
		f.Props[p.Num] = p.Snapshot()
	}
	for _, s := range frame.Sects {
		f.Sects[s.Num] = s.Snapshot(f)
	}
	for _, p := range frame.Piles {
		f.Piles[p.Num] = p.Snapshot()
	}
	for _, p := range frame.Bonds {
		f.Bonds[p.Num] = p.Snapshot()
	}
	for _, n := range frame.Nodes {
		f.Nodes[n.Num] = n.Snapshot(f)
	}
	for _, el := range frame.Elems {
		f.Elems[el.Num] = el.Snapshot(f)
	}
	for _, c := range frame.Chains {
		newc := c.Snapshot(f)
		f.Chains[c.Num()] = newc
		for _, el := range newc.Elems() {
			el.Chain = newc
		}
	}
	for k, v := range frame.Eigenvalue {
		f.Eigenvalue[k] = v
	}
	for _, k := range frame.Kijuns {
		f.Kijuns[k.Name] = k.Snapshot()
	}
	f.Maxenum = frame.Maxenum
	f.Maxnnum = frame.Maxnnum
	f.Maxsnum = frame.Maxsnum
	for k, v := range frame.Nlap {
		f.Nlap[k] = v
	}
	f.Ai = frame.Ai.Snapshot()
	f.Show = frame.Show
	for k, v := range frame.DataFileName {
		f.DataFileName[k] = v
	}
	for k, v := range frame.ResultFileName {
		f.ResultFileName[k] = v
	}
	f.LstFileName = frame.LstFileName
	return f
}

// Bbox returns the bounding box of nodes in 3D space.
func (frame *Frame) Bbox(hide bool) (xmin, xmax, ymin, ymax, zmin, zmax float64) {
	var mins, maxs [3]float64
	first := true
	for _, j := range frame.Nodes {
		if hide && j.IsHidden(frame.Show) {
			continue
		}
		if first {
			for k := 0; k < 3; k++ {
				mins[k] = j.Coord[k]
				maxs[k] = j.Coord[k]
			}
			first = false
		} else {
			for k := 0; k < 3; k++ {
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

// Bbox returns the bounding box of nodes in a projected 2D space.
func (frame *Frame) Bbox2D(hide bool) (xmin, xmax, ymin, ymax float64) {
	var mins, maxs [2]float64
	first := true
	for _, j := range frame.Nodes {
		if hide && j.IsHidden(frame.Show) {
			continue
		}
		if first {
			for k := 0; k < 2; k++ {
				mins[k] = j.Pcoord[k]
				maxs[k] = j.Pcoord[k]
			}
			first = false
		} else {
			for k := 0; k < 2; k++ {
				if j.Pcoord[k] < mins[k] {
					mins[k] = j.Pcoord[k]
				} else if maxs[k] < j.Pcoord[k] {
					maxs[k] = j.Pcoord[k]
				}
			}
		}
	}
	return mins[0], maxs[0], mins[1], maxs[1]
}

// ReadInp reads an input file for st frame.
func (frame *Frame) ReadInp(filename string, coord []float64, angle float64, overwrite bool) error {
	tmp := make([]string, 0)
	nodemap := make(map[int]int)
	if len(coord) < 3 {
		coord = []float64{0.0, 0.0, 0.0}
	}
	var chain *Chain
	err := ParseFile(filename, func(words []string) error {
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
			tmp = append(tmp, words...)
		case "PROP", "SECT", "PILE", "BOND", "NODE", "ELEM":
			nodemap, err = frame.ParseInp(tmp, coord, angle, nodemap, overwrite, chain)
			tmp = words
		case "CHAIN":
			nodemap, err = frame.ParseInp(tmp, coord, angle, nodemap, overwrite, chain)
			tmp = make([]string, 0)
			chain = NewChain(frame, nil, nil, nil, func(c *Chain) bool { return true }, nil, nil)
		case "}":
			nodemap, err = frame.ParseInp(tmp, coord, angle, nodemap, overwrite, chain)
			tmp = make([]string, 0)
			if chain != nil && chain.Size() > 0 {
				frame.Chains[chain.Elems()[0].Num] = chain
				for _, el := range chain.Elems() {
					el.Chain = chain
				}
			}
			chain = nil
		case "BASE":
			val, err := strconv.ParseFloat(words[1], 64)
			if err == nil {
				frame.Ai.Base = []float64{val, val}
			}
			if len(words) > 2 {
				val, err := strconv.ParseFloat(words[2], 64)
				if err == nil {
					frame.Ai.Base[1] = val
				}
			}
		case "LOCATE":
			val, err := strconv.ParseFloat(words[1], 64)
			if err == nil {
				frame.Ai.Locate = val
			}
		case "TFACT":
			val, err := strconv.ParseFloat(words[1], 64)
			if err == nil {
				frame.Ai.Tfact = val
			}
		case "GPERIOD":
			val, err := strconv.ParseFloat(words[1], 64)
			if err == nil {
				frame.Ai.Gperiod = val
			}
		case "NFLOOR":
			val, err := strconv.ParseInt(words[1], 10, 64)
			if err == nil {
				frame.Ai.Nfloor = int(val)
			}
			frame.Ai.Boundary = make([]float64, frame.Ai.Nfloor+1)
		case "HEIGHT":
			if frame.Ai.Nfloor == 0 {
				frame.Ai.Nfloor = len(words) - 2
				frame.Ai.Boundary = make([]float64, len(words)-1)
			}
			if len(words) <= frame.Ai.Nfloor+1 {
				return errors.New(fmt.Sprintf("ReadInp: HEIGHT: not enough boundaries (%d < %d)", len(words)-1, frame.Ai.Nfloor+1))
			}
			for i := 0; i < frame.Ai.Nfloor+1; i++ {
				val, err := strconv.ParseFloat(words[1+i], 64)
				if err != nil {
					break
				}
				frame.Ai.Boundary[i] = val
			}
		case "ROUGHNESS":
			val, err := strconv.ParseInt(words[1], 10, 64)
			if err == nil {
				frame.Wind.Roughness = int(val)
			}
		case "VELOCITY":
			val, err := strconv.ParseFloat(words[1], 64)
			if err == nil {
				frame.Wind.Velocity = val
			}
		case "WFACT":
			val, err := strconv.ParseFloat(words[1], 64)
			if err == nil {
				frame.Wind.Factor = val
			}
		case "GFACT":
			frame.View.Gfact, err = strconv.ParseFloat(words[1], 64)
		case "FOCUS":
			for i := 0; i < 3; i++ {
				frame.View.Focus[i], err = strconv.ParseFloat(words[i+1], 64)
			}
		case "ANGLE":
			for i := 0; i < 2; i++ {
				frame.View.Angle[i], err = strconv.ParseFloat(words[i+1], 64)
			}
		case "DISTS":
			for i := 0; i < 2; i++ {
				frame.View.Dists[i], err = strconv.ParseFloat(words[i+1], 64)
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
	nodemap, err = frame.ParseInp(tmp, coord, angle, nodemap, overwrite, chain)
	if err != nil {
		return err
	}
	if chain != nil && chain.Size() > 0 {
		frame.Chains[chain.Elems()[0].Num] = chain
		for _, el := range chain.Elems() {
			el.Chain = chain
		}
	}
	frame.Name = filepath.Base(filename)
	frame.Project = ProjectName(filename)
	path, err := filepath.Abs(filename)
	if err != nil {
		frame.Path = filename
	} else {
		frame.Path = path
	}
	conffn := Ce(filename, ".conf")
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
		// fmt.Printf("No Configure File: %s\n", conffn)
	}
	return nil
}

// ParseInp parses a list of strings and switches to each parsing function according to the first string.
func (frame *Frame) ParseInp(lis []string, coord []float64, angle float64, nodemap map[int]int, overwrite bool, chain *Chain) (map[int]int, error) {
	var err error
	var def int
	var node *Node
	if len(lis) == 0 {
		return nodemap, nil
	}
	first := lis[0]
	switch first {
	case "ELEM":
		_, err = frame.ParseElem(lis, nodemap, chain)
	case "NODE":
		node, def, err = frame.ParseNode(lis, coord, angle)
		if err == nil {
			nodemap[def] = node.Num
		}
	case "SECT":
		_, err = frame.ParseSect(lis, overwrite)
	case "PROP":
		_, err = frame.ParseProp(lis, overwrite)
	case "PILE":
		_, err = frame.ParsePile(lis, overwrite)
	case "BOND":
		_, err = frame.ParseBond(lis, overwrite)
	}
	return nodemap, err
}

// ParseProp parses PROP information.
func (frame *Frame) ParseProp(lis []string, overwrite bool) (*Prop, error) {
	var num int64
	var err error
	p := new(Prop)
	for i, word := range lis {
		switch word {
		case "PROP":
			num, err = strconv.ParseInt(lis[i+1], 10, 64)
			p.Num = int(num)
		case "PNAME":
			p.Name, err = lis[i+1], nil
		case "HIJU":
			p.Hiju, err = strconv.ParseFloat(lis[i+1], 64)
		case "E":
			p.EL, err = strconv.ParseFloat(lis[i+1], 64)
			p.ES = p.EL
		case "ES":
			p.ES, err = strconv.ParseFloat(lis[i+1], 64)
		case "POI":
			p.Poi, err = strconv.ParseFloat(lis[i+1], 64)
		case "PCOLOR":
			var tmpcol int64
			val := 65536
			for j := 0; j < 3; j++ {
				tmpcol, err = strconv.ParseInt(lis[i+1+j], 10, 64)
				p.Color += int(tmpcol) * val
				val >>= 8
			}
		}
		if err != nil {
			return nil, err
		}
	}
	if _, ok := frame.Props[p.Num]; ok {
		if !overwrite {
			return nil, nil
		}
	}
	frame.Props[p.Num] = p
	return p, nil
}

// ParsePile parses PILE information.
func (frame *Frame) ParsePile(lis []string, overwrite bool) (*Pile, error) {
	var num int64
	var err error
	p := new(Pile)
	for i, word := range lis {
		switch word {
		case "PILE":
			num, err = strconv.ParseInt(lis[i+1], 10, 64)
			p.Num = int(num)
		case "INAME":
			p.Name = lis[i+1]
		case "MOMENT":
			p.Moment, err = strconv.ParseFloat(lis[i+1], 64)
		}
		if err != nil {
			return nil, err
		}
	}
	if _, ok := frame.Piles[p.Num]; ok {
		if !overwrite {
			return nil, nil
		}
	}
	frame.Piles[p.Num] = p
	return p, nil
}

// ParseBond parses BOND information.
func (frame *Frame) ParseBond(lis []string, overwrite bool) (*Bond, error) {
	var num int64
	var err error
	b := new(Bond)
	for i, word := range lis {
		switch word {
		case "BOND":
			num, err = strconv.ParseInt(lis[i+1], 10, 64)
			b.Num = int(num)
		case "BNAME":
			b.Name = lis[i+1]
		case "KR":
			b.Stiffness = make([]float64, 2)
			for j := 0; j < 2; j++ {
				val, err := strconv.ParseFloat(lis[i+1+j], 64)
				if err != nil {
					return nil, err
				}
				b.Stiffness[j] = val
			}
		}
		if err != nil {
			return nil, err
		}
	}
	if b.Num == 1 {
		frame.Bonds[1] = Pin
		return Pin, nil
	}
	if _, ok := frame.Bonds[b.Num]; ok {
		if !overwrite {
			return nil, nil
		}
	}
	frame.Bonds[b.Num] = b
	return b, nil
}

// ParseSect parses SECT information.
func (frame *Frame) ParseSect(lis []string, overwrite bool) (*Sect, error) {
	var num int64
	var err error
	s := NewSect()
	key := ""
	tmp := make([]string, 0)
	figmap := make(map[string][]string)
	skip := 0
	for i, word := range lis {
		if skip > 0 {
			skip--
			continue
		}
		switch word {
		default:
			tmp = append(tmp, word)
		case "FPROP", "AREA", "IXX", "IYY", "VEN", "THICK", "SREIN", "SIGMA", "XFACE", "YFACE":
			if key != "" {
				figmap[key] = tmp
				tmp = make([]string, 0)
			}
			key = word
		case "SECT":
			num, err = strconv.ParseInt(lis[i+1], 10, 64)
			s.Num = int(num)
			s.Original = int(num)
			skip = 1
		case "SNAME":
			s.Name = ToUtf8string(lis[i+1])
			skip = 1
		case "NFIG":
			skip = 1
		case "FIG":
			if len(figmap) > 0 {
				if len(tmp) > 0 {
					figmap[key] = tmp
				}
				err = s.ParseFig(frame, figmap)
				figmap = make(map[string][]string)
				tmp = make([]string, 0)
			}
			key = word
		case "LLOAD":
			for j := 0; j < 3; j++ {
				s.Lload[j], err = strconv.ParseFloat(lis[i+1+j], 64)
			}
			skip = 3
		case "PERPL":
			for j := 0; j < 3; j++ {
				s.Perpl[j], err = strconv.ParseFloat(lis[i+1+j], 64)
			}
			skip = 3
		case "EXP":
			s.Exp, err = strconv.ParseFloat(lis[i+1], 64)
			skip = 1
		case "EXQ":
			s.Exq, err = strconv.ParseFloat(lis[i+1], 64)
			skip = 1
		case "NZMAX":
			for j := 0; j < 12; j++ {
				s.Yield[j], err = strconv.ParseFloat(lis[i+1+2*j], 64)
			}
			skip = 23
		case "BSECT": // TODO: implement
			skip = 1
		case "COLOR":
			var tmpcol int64
			s.Color = 0
			val := 65536
			for j := 0; j < 3; j++ {
				tmpcol, err = strconv.ParseInt(lis[i+1+j], 10, 64)
				s.Color += int(tmpcol) * val
				val >>= 8
			}
			skip = 3
		}
		if err != nil {
			return nil, err
		}
	}
	if len(figmap) > 0 {
		if len(tmp) > 0 {
			figmap[key] = tmp
		}
		err = s.ParseFig(frame, figmap)
		if err != nil {
			return nil, err
		}
	}
	s.Frame = frame
	if _, ok := frame.Sects[s.Num]; ok {
		if !overwrite {
			return nil, nil
		}
	}
	frame.Sects[s.Num] = s
	frame.Show.Sect[s.Num] = true
	return s, nil
}

// ParseFig parses FIG information.
func (sect *Sect) ParseFig(frame *Frame, figmap map[string][]string) error {
	var num int64
	if len(figmap) == 0 {
		return nil
	}
	var err error
	f := &Fig{Value: make(map[string]float64)}
	for key, data := range figmap {
		if len(data) < 1 {
			continue
		}
		switch key {
		case "FIG":
			num, err = strconv.ParseInt(data[0], 10, 64)
			f.Num = int(num)
		case "FPROP":
			pnum, err := strconv.ParseInt(data[0], 10, 64)
			if err == nil {
				if val, ok := frame.Props[int(pnum)]; ok {
					f.Prop = val
				} else {
					return fmt.Errorf("PROP %d doesn't exist", pnum)
				}
			}
		case "AREA":
			val, err := strconv.ParseFloat(data[0], 64)
			if err != nil {
				return err
			}
			if len(data) >= 2 {
				num, den, err := unit.Parse(strings.Join(data[1:], " "))
				if err != nil {
					return err
				}
				uval, err := unit.NewValue(val, num, den).As(unit.M, unit.M)
				if err != nil {
					return err
				}
				val = uval
			}
			f.Value[key] = val
		case "IXX", "IYY", "VEN":
			val, err := strconv.ParseFloat(data[0], 64)
			if err != nil {
				return err
			}
			if len(data) >= 2 {
				num, den, err := unit.Parse(strings.Join(data[1:], " "))
				if err != nil {
					return err
				}
				uval, err := unit.NewValue(val, num, den).As(unit.Power(unit.M, 4)...)
				if err != nil {
					return err
				}
				val = uval
			}
			f.Value[key] = val
		case "THICK":
			val, err := strconv.ParseFloat(data[0], 64)
			if err != nil {
				return err
			}
			if len(data) >= 2 {
				num, den, err := unit.Parse(strings.Join(data[1:], " "))
				if err != nil {
					return err
				}
				uval, err := unit.NewValue(val, num, den).As(unit.M)
				if err != nil {
					return err
				}
				val = uval
			}
			f.Value[key] = val
		case "SREIN":
			val, err := strconv.ParseFloat(data[0], 64)
			if err != nil {
				return err
			}
			f.Value[key] = val
		case "SIGMA":
			if len(data) < 6 {
				continue
			}
			var val float64
			var err error
			if strings.HasPrefix(data[0], "FC") {
				val, err = strconv.ParseFloat(strings.TrimPrefix(data[0], "FC"), 64)
			} else {
				val, err = strconv.ParseFloat(data[0], 64)
			}
			if err != nil {
				return err
			}
			f.Value["FC"] = val
			if strings.HasPrefix(data[1], "SD") {
				val, err = strconv.ParseFloat(strings.TrimPrefix(data[1], "SD"), 64)
			} else {
				val, err = strconv.ParseFloat(data[1], 64)
			}
			if err != nil {
				return err
			}
			f.Value["SD"] = val
			if strings.HasPrefix(data[2], "D") {
				val, err = strconv.ParseFloat(strings.TrimPrefix(data[2], "D"), 64)
			} else {
				val, err = strconv.ParseFloat(data[2], 64)
			}
			if err != nil {
				return err
			}
			f.Value["RD"] = val
			val, err = strconv.ParseFloat(data[3], 64)
			if err != nil {
				return err
			}
			f.Value["RA"] = val
			if strings.HasPrefix(data[4], "@") {
				val, err = strconv.ParseFloat(strings.TrimPrefix(data[4], "@"), 64)
			} else {
				val, err = strconv.ParseFloat(data[4], 64)
			}
			if err != nil {
				return err
			}
			f.Value["PITCH"] = val
			val, err = strconv.ParseFloat(data[5], 64)
			if err != nil {
				return err
			}
			f.Value["SINDOU"] = val
		case "XFACE", "YFACE":
			if len(data) < 2 {
				continue
			}
			tmp := make([]float64, 2)
			for j := 0; j < 2; j++ {
				val, err := strconv.ParseFloat(data[j], 64)
				if err != nil {
					return err
				}
				tmp[j] = val
			}
			f.Value[key] = tmp[0]
			f.Value[fmt.Sprintf("%s_H", key)] = tmp[1]
		}
		if err != nil {
			return err
		}
	}
	sect.Figs = append(sect.Figs, f)
	return err
}

// ParseNode parses NODE information.
func (frame *Frame) ParseNode(lis []string, coord []float64, angle float64) (*Node, int, error) {
	var num int64
	var err error
	n := NewNode()
	llis := len(lis)
	for i, word := range lis {
		switch word {
		case "NODE":
			num, err = strconv.ParseInt(lis[i+1], 10, 64)
			n.Num = int(num)
		case "CORD":
			if llis < i+4 {
				return nil, 0, errors.New(fmt.Sprintf("ParseNode: CORD IndexError NODE %d", n.Num))
			}
			for j := 0; j < 3; j++ {
				n.Coord[j], err = strconv.ParseFloat(lis[i+1+j], 64)
				n.Coord[j] += coord[j]
			}
		case "ICON":
			if llis < i+7 {
				return nil, 0, errors.New(fmt.Sprintf("ParseNode: ICON IndexError NODE %d", n.Num))
			}
			for j := 0; j < 6; j++ {
				if lis[i+1+j] == "0" {
					n.Conf[j] = false
				} else {
					n.Conf[j] = true
				}
			}
		case "VCON":
			if llis < i+7 {
				return nil, 0, errors.New(fmt.Sprintf("ParseNode: VCON IndexError NODE %d", n.Num))
			}
			for j := 0; j < 6; j++ {
				n.Load[j], err = strconv.ParseFloat(lis[i+1+j], 64)
			}
		case "PCON":
			if llis < i+2 {
				return nil, 0, errors.New(fmt.Sprintf("ParseNode: PCON IndexError NODE %d", n.Num))
			}
			var pnum int64
			pnum, err = strconv.ParseInt(lis[i+1], 10, 64)
			if err != nil {
				return nil, 0, err
			}
			if p, ok := frame.Piles[int(pnum)]; ok {
				n.Pile = p
			} else {
				return nil, 0, errors.New(fmt.Sprintf("ParseNode: Pile %d doesn't exist NODE %d", pnum, n.Num))
			}
		}
		if err != nil {
			return nil, 0, err
		}
	}
	c := RotateVector(n.Coord, coord, []float64{0.0, 0.0, 1.0}, angle)
	n.Coord = c
	newnode := frame.SearchNode(c[0], c[1], c[2])
	if newnode == nil {
		if _, exist := frame.Nodes[n.Num]; exist {
			frame.Maxnnum++
			old := n.Num
			n.Num = frame.Maxnnum
			frame.Nodes[n.Num] = n
			n.Frame = frame
			return n, old, nil
		} else {
			if n.Num > frame.Maxnnum {
				frame.Maxnnum = n.Num
			}
			frame.Nodes[n.Num] = n
			n.Frame = frame
			return n, n.Num, nil
		}
	} else {
		return newnode, n.Num, nil
	}
}

// ParseElem parses ELEM information.
func (frame *Frame) ParseElem(lis []string, nodemap map[int]int, chain *Chain) (*Elem, error) {
	var num int64
	var err error
	e := new(Elem)
	llis := len(lis)
	for i, word := range lis {
		switch word {
		case "ELEM":
			num, err = strconv.ParseInt(lis[i+1], 10, 64)
			e.Num = int(num)
		case "ESECT":
			tmp, err := strconv.ParseInt(lis[i+1], 10, 64)
			if err != nil {
				return nil, err
			}
			if val, ok := frame.Sects[int(tmp)]; ok {
				e.Sect = val
			} else {
				fmt.Printf("SECT %d doesn't exist\n", tmp)
				e.Sect = frame.AddSect(int(tmp))
			}
		case "ENODS":
			num, err = strconv.ParseInt(lis[i+1], 10, 64)
			e.Enods = int(num)
		case "ENOD":
			if llis < i+1+e.Enods {
				return nil, errors.New(fmt.Sprintf("ParseElem: ENOD IndexError ELEM %d", e.Num))
			}
			en := make([]*Node, int(e.Enods))
			for j := 0; j < e.Enods; j++ {
				tmp, err := strconv.ParseInt(lis[i+1+j], 10, 64)
				if err != nil {
					return nil, err
				}
				if val, ok := frame.Nodes[nodemap[int(tmp)]]; ok {
					en[j] = val
				} else {
					return nil, errors.New(fmt.Sprintf("ParseElem: Enod not found ELEM %d ENOD %d", e.Num, tmp))
				}
			}
			e.Enod = en
		case "BONDS":
			if llis < i+1+e.Enods*6 {
				return nil, errors.New(fmt.Sprintf("ParseElem: BONDS IndexError ELEM %d", e.Num))
			}
			bon := make([]*Bond, int(e.Enods)*6)
			for j := 0; j < int(e.Enods)*6; j++ {
				if lis[i+1+j] == "0" {
					bon[j] = nil
				} else {
					tmp, err := strconv.ParseInt(lis[i+1+j], 10, 64)
					if err != nil {
						return nil, err
					}
					if val, ok := frame.Bonds[int(tmp)]; ok {
						bon[j] = val
					} else if int(tmp) == 1 {
						frame.Bonds[1] = Pin
						bon[j] = Pin
					} else {
						return nil, errors.New(fmt.Sprintf("ParseElem: unknown BOND %d at ELEM %d", tmp, e.Num))
					}
				}
			}
			e.Bonds = bon
		case "CMQ":
			if llis < i+1+e.Enods*6 {
				return nil, errors.New(fmt.Sprintf("ParseElem: CMQ IndexError ELEM %d", e.Num))
			}
			cmq := make([]float64, int(e.Enods)*6)
			for j := 0; j < int(e.Enods)*6; j++ {
				cmq[j], err = strconv.ParseFloat(lis[i+1+j], 64)
			}
			e.Cmq = cmq
		case "CANG":
			e.Cang, err = strconv.ParseFloat(lis[i+1], 64)
		case "WRECT":
			wrect := make([]float64, 2)
			for j := 0; j < 2; j++ {
				val, err := strconv.ParseFloat(lis[i+1+j], 64)
				if err != nil {
					return nil, err
				}
				wrect[j] = val
			}
			e.Wrect = wrect
		case "PREST":
			val, err := strconv.ParseFloat(lis[i+1], 64)
			if err != nil {
				return nil, err
			}
			e.Prestress = val
		case "TYPE":
			err = e.setEtype(lis[i+1])
		}
		if err != nil {
			return nil, err
		}
	}
	var el *Elem
	if e.IsLineElem() {
		el = NewLineElem(e.Enod, e.Sect, e.Etype)
		el.Num = e.Num
		el.Cang = e.Cang
		el.Cmq = e.Cmq
		el.Bonds = e.Bonds
		el.Prestress = e.Prestress
		el.SetPrincipalAxis()
		if chain != nil {
			chain.Append(el)
		}
	} else {
		el = NewPlateElem(e.Enod, e.Sect, e.Etype)
		el.Num = e.Num
		if e.Wrect != nil {
			el.Wrect = e.Wrect
		}
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
	return el, nil
}

// ReadConfigure reads a configuration file.
func (frame *Frame) ReadConfigure(filename string) error {
	tmp := make([]string, 0)
	err := ParseFile(filename, func(words []string) error {
		var err error
		first := strings.Trim(strings.ToUpper(words[0]), ":")
		if strings.HasPrefix(first, "#") {
			return nil
		}
		switch first {
		default:
			tmp = append(tmp, words...)
		case "LEVEL":
			err = frame.ParseConfigure(tmp)
			tmp = words
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

// ParseConfigure parses list of strings as configuration.
func (frame *Frame) ParseConfigure(lis []string) (err error) {
	if len(lis) == 0 {
		return nil
	}
	first := strings.Trim(strings.ToUpper(lis[0]), ":")
	switch first {
	case "LEVEL":
		err = frame.ParseLevel(lis[1:])
	}
	return err
}

// ParseLevel parses LEVEL information.
func (frame *Frame) ParseLevel(lis []string) (err error) {
	size := len(lis)
	val := make([]float64, size+2)
	val[0] = MINCOORD
	for i := 0; i < size; i++ {
		tmp, err := strconv.ParseFloat(lis[i], 64)
		if err != nil {
			return err
		}
		val[1+i] = tmp
	}
	val[size+1] = MAXCOORD
	frame.Ai.Nfloor = size + 1
	frame.Ai.Boundary = val
	return nil
}

// ReadData reads an input file for arclm frame.
func (frame *Frame) ReadData(filename string) error {
	ext := filepath.Ext(filename)
	var period string
	if p, ok := PeriodExt[ext]; ok {
		period = p
	} else {
		period = strings.ToUpper(ext[1:])
	}
	af := arclm.NewFrame()
	af.ReadInput(filename)
	frame.Arclms[period] = af
	f, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	var lis []string
	if ok := strings.HasSuffix(string(f), "\r\n"); ok {
		lis = strings.Split(string(f), "\r\n")
	} else {
		lis = strings.Split(string(f), "\n")
	}
	var words []string
	for _, k := range strings.Split(lis[0], " ") {
		if k != "" {
			words = append(words, k)
		}
	}
	nums := make([]int, 3)
	for i := 0; i < 3; i++ {
		num, err := strconv.ParseInt(words[i], 10, 64)
		if err != nil {
			return err
		}
		nums[i] = int(num)
	}
	// Sect
	for _, j := range lis[1 : 1+nums[2]] {
		var words []string
		for _, k := range strings.Split(j, " ") {
			if k != "" {
				words = append(words, k)
			}
		}
		if len(words) == 0 {
			continue
		}
		num, err := strconv.ParseInt(words[0], 10, 64)
		if err != nil {
			return err
		}
		snum := int(num)
		if _, ok := frame.Sects[snum]; !ok {
			sect := frame.AddSect(snum) // TODO: set E, poi, ...
			for i := 0; i < 12; i++ {
				val, err := strconv.ParseFloat(words[7+i], 64)
				if err != nil {
					return err
				}
				sect.Yield[i] = val
			}
			if len(words) >= 20 {
				tp, err := strconv.ParseInt(words[19], 10, 64)
				if err != nil {
					return err
				}
				sect.Type = int(tp)
			}
			if len(words) >= 21 {
				oc, err := strconv.ParseInt(words[20], 10, 64)
				if err != nil {
					// return err
					continue // for backward compatibility
				}
				sect.Original = int(oc)
			}
		}
	}
	// Node1
	for _, j := range lis[1+nums[2] : 1+nums[2]+nums[0]] {
		var words []string
		for _, k := range strings.Split(j, " ") {
			if k != "" {
				words = append(words, k)
			}
		}
		if len(words) == 0 {
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
	// wg := new(sync.WaitGroup)
	for _, j := range lis[1+nums[2]+nums[0] : 1+nums[2]+nums[0]+nums[1]] {
		var words []string
		for _, k := range strings.Split(j, " ") {
			if k != "" {
				words = append(words, k)
			}
		}
		if len(words) == 0 {
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
			for i := 0; i < 2; i++ {
				tmp, err := strconv.ParseInt(words[2+i], 10, 64)
				if err != nil {
					return err
				}
				ns[i] = frame.Nodes[int(tmp)]
			}
			sect := frame.Sects[int(sec)]
			newel := frame.AddLineElem(enum, ns, sect, sect.Type) // TODO: set etype, cang, ...
			if newel.Etype == WBRACE || newel.Etype == SBRACE {
				// wg.Add(1)
				// go func (nel *Elem, n []*Node) {
				// defer wg.Done()
				for _, el := range frame.SearchElem(ns...) {
					if el.Etype == sect.Type+2 && el.IsDiagonal(ns[0], ns[1]) {
						el.Adopt(newel)
					}
				}
				// }(newel, ns)
			}
		}
	}
	// wg.Wait()
	// Node2
	for _, j := range lis[1+nums[2]+nums[0]+nums[1] : 1+nums[2]+nums[0]+nums[1]+nums[0]] {
		var words []string
		for _, k := range strings.Split(j, " ") {
			if k != "" {
				words = append(words, k)
			}
		}
		if len(words) == 0 {
			continue
		}
		num, err := strconv.ParseInt(words[0], 10, 64)
		if err != nil {
			return err
		}
		nnum := int(num)
		if node, ok := frame.Nodes[nnum]; ok {
			force := make([]float64, 6)
			for i := 0; i < 6; i++ {
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

// ReadResult reads an output file of analysis.
func (frame *Frame) ReadResult(filename string, mode uint) error {
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
	if ok := strings.HasSuffix(string(f), "\r\n"); ok {
		lis = strings.Split(string(f), "\r\n")
	} else {
		lis = strings.Split(string(f), "\n")
	}
	tmpline := 0
	pat1 := regexp.MustCompile("^ *\\*\\* *FORCES")
	for _, j := range lis {
		if pat1.MatchString(j) {
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
		if j == "" {
			tmpline++
			continue
		}
		break
	}
	if tmpline > len(lis)-1 {
		return errors.New("ReadResult: no data")
	}
	for { // Reading Elem Stress
		j := strings.Join([]string{lis[tmpline], lis[tmpline+1]}, " ")
		tmpline += 2
		var words []string
		for _, k := range strings.Split(j, " ") {
			if k != "" {
				words = append(words, k)
			}
		}
		if len(words) == 0 {
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
			if !elem.IsLineElem() {
				continue
			}
			if mode == UpdateResult {
				elem.Stress[period] = make(map[int][]float64)
			}
			var tmp []float64
			for i := 0; i < 2; i++ {
				num, err := strconv.ParseInt(words[2+7*i], 10, 64)
				if err != nil {
					return err
				}
				enod[i] = int(num)
				tmp = make([]float64, 6)
				for k := 0; k < 6; k++ {
					val, err := strconv.ParseFloat(words[3+7*i+k], 64)
					if err != nil {
						return err
					}
					tmp[k] = val
				}
				switch mode {
				case UpdateResult:
					elem.Stress[period][int(num)] = tmp
				case AddResult, AddSearchResult:
					if elem.Stress[period][int(num)] != nil {
						for ind := 0; ind < 6; ind++ {
							elem.Stress[period][int(num)][ind] += tmp[ind]
						}
					}
				}
				stress[i] = tmp
			}
		} else {
			if mode == AddSearchResult {
				if _, ok := frame.Nodes[enod[0]]; ok {
					if _, ok2 := frame.Nodes[enod[1]]; ok2 {
						for _, el := range frame.SearchElem(frame.Nodes[0], frame.Nodes[1]) {
							if !el.IsLineElem() {
								continue
							}
							fmt.Printf("ReadResult: ELEM %d -> ELEM %d\n", enum, el.Num)
							for i := 0; i < 2; i++ {
								for j := 0; j < 6; j++ {
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
	tmpline -= 2
	pat2 := regexp.MustCompile("^ *\\*\\* *DISPLACEMENT")
	for _, j := range lis[tmpline:] {
		if pat2.MatchString(j) {
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
		tmpline++
		var words []string
		for _, k := range strings.Split(j, " ") {
			if k != "" {
				words = append(words, k)
			}
		}
		if len(words) == 0 {
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
			for k := 0; k < 6; k++ {
				val, err := strconv.ParseFloat(words[1+k], 64)
				if err != nil {
					return err
				}
				tmp[k] = val
			}
			switch mode {
			case UpdateResult:
				node.Disp[period] = tmp
			case AddResult, AddSearchResult:
				for ind := 0; ind < 6; ind++ {
					node.Disp[period][ind] += tmp[ind]
				}
			}
		} else {
			fmt.Printf("NODE %d not found\n", nnum)
		}
	}
	tmpline -= 2
	pat3 := regexp.MustCompile("^ *\\*\\* *REACTION")
	for _, j := range lis[tmpline:] {
		if pat3.MatchString(j) {
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
		var words []string
		for _, k := range strings.Split(j, " ") {
			if k != "" {
				words = append(words, k)
			}
		}
		if len(words) == 0 {
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
			case UpdateResult:
				node.Reaction[period][ind-1] = val
			case AddResult, AddSearchResult:
				node.Reaction[period][ind-1] += val
			}
		} else {
			fmt.Printf("NODE %d not found\n", nnum)
		}
	}
	frame.ResultFileName[period] = filename
	return nil
}

// ReadBuckling reads an output file of bucling analysis.
func (frame *Frame) ReadBuckling(filename string) error {
	tmp := make([][]string, 0)
	err := ParseFile(filename, func(words []string) error {
		var err error
		first := strings.ToUpper(words[0])
		if strings.HasPrefix(first, "#") {
			return nil
		}
		switch first {
		default:
			tmp = append(tmp, words)
		case "EIGEN":
			err = frame.ParseEigen(tmp)
			tmp = [][]string{words}
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

// ParseEigen parses eigenvalues and eigenvectors.
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
				for i := 0; i < 6; i++ {
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

// ReadVibrationalEigenmode reads an output file of vibrational eigenvalue analysis.
func (frame *Frame) ReadVibrationalEigenmode(filename string) error {
	var mode int
	tmp := make([][]string, 0)
	err := ParseFile(filename, func(words []string) error {
		var err error
		if len(words) == 0 {
			return nil
		}
		if words[0] == "DEIGABGENERAL" && words[2] == "VECTOR" {
			if len(tmp) > 0 {
				err = frame.ParseVibrationalEigenmode(mode, tmp)
				if err != nil {
					return err
				}
				tmp = [][]string{}
			}
			val, err := strconv.ParseInt(words[3], 10, 64)
			if err != nil {
				return err
			}
			mode = int(val)
			return nil
		}
		if mode > 0 && len(words) == 7 {
			tmp = append(tmp, words)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return frame.ParseVibrationalEigenmode(mode, tmp)
}

// ParseVibrationalEigenmode parses eigenvalues and eigenvectors.
func (frame *Frame) ParseVibrationalEigenmode(mode int, lis [][]string) (err error) {
	period := fmt.Sprintf("B%d", mode)
	for _, l := range lis {
		nnum, err := strconv.ParseInt(l[0], 10, 64)
		if err != nil {
			return err
		}
		disp := make([]float64, 6)
		for i := 0; i < 6; i++ {
			val, err := strconv.ParseFloat(l[1+i], 64)
			if err != nil {
				return err
			}
			disp[i] = val
		}
		frame.Nodes[int(nnum)].Disp[period] = disp
	}
	return nil
}

// ReadZoubun reads an output file of push-over analysis.
func (frame *Frame) ReadZoubun(filename string) error {
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
				tmp = [][]string{words}
				period = fmt.Sprintf("%s@%d", ext, nlap)
			} else {
				tmp = append(tmp, words)
			}
		case "\"DISPLACEMENT\"", "\"STRESS\"", "\"REACTION\"", "\"CURRENT":
			err = frame.ParseZoubun(tmp, period)
			tmp = [][]string{words}
		}
		if err != nil {
			return errors.New(fmt.Sprintf("%s, LAP:%d", err.Error(), nlap))
		}
		return nil
	})
	if err != nil {
		return err
	}
	err = frame.ParseZoubun(tmp, period)
	if err != nil {
		return errors.New(fmt.Sprintf("%s, LAP:%d", err.Error(), nlap))
	}
	frame.Nlap[ext] = nlap
	return nil
}

// ParseZoubun parses a list of strings and switches to each parsing function according to the first string.
func (frame *Frame) ParseZoubun(lis [][]string, period string) error {
	var err error
	if len(lis) == 0 {
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

// ParseZoubunStress parses stress information.
func (frame *Frame) ParseZoubunStress(lis [][]string, period string) error {
	for _, l := range lis {
		if strings.ToUpper(l[0]) == "ELEM" {
			enum, err := strconv.ParseInt(l[1], 10, 64)
			if err != nil {
				return errors.New(fmt.Sprintf("ParseZoubunStress: %s enum %s", err, l[1]))
			}
			if el, ok := frame.Elems[int(enum)]; ok {
				var val float64
				var err error
				nnum, err := strconv.ParseInt(strings.Trim(l[5], ":"), 10, 64)
				if err != nil {
					return errors.New(fmt.Sprintf("ParseZoubunStress: %s ELEM: %d nnum", err, el.Num))
				}
				stress := make([]float64, 6)
				for i := 0; i < 6; i++ {
					val, err := strconv.ParseFloat(l[7+2*i], 64)
					if err != nil {
						return errors.New(fmt.Sprintf("ParseZoubunStress: %s ELEM: %d stress", err, el.Num))
					}
					stress[i] = val
				}
				if len(l) < 20 {
					if strings.HasPrefix(l[len(l)-1], "f=") {
						val, err = strconv.ParseFloat(strings.Trim(l[len(l)-1], "f="), 64)
					} else {
						return errors.New(fmt.Sprintf("ParseZoubunStress: Index Error ELEM: %d function1", el.Num))
					}
				} else {
					val, err = strconv.ParseFloat(l[19], 64)
				}
				if err != nil {
					return errors.New(fmt.Sprintf("ParseZoubunStress: %s ELEM: %d function2", err, el.Num))
				}
				ph := val >= PlasticThreshold
				if el.Stress[period] == nil {
					el.Stress[period] = make(map[int][]float64)
				}
				if el.Phinge[period] == nil {
					el.Phinge[period] = make(map[int]bool)
				}
				el.Stress[period][int(nnum)] = stress
				el.Phinge[period][int(nnum)] = ph
			}
		}
	}
	return nil
}

// ParseZoubunReaction parses reaction information.
func (frame *Frame) ParseZoubunReaction(lis [][]string, period string) error {
	for _, l := range lis {
		if strings.ToUpper(l[0]) == "NODE:" {
			nnum, err := strconv.ParseInt(l[1], 10, 64)
			if err != nil {
				return err
			}
			if n, ok := frame.Nodes[int(nnum)]; ok {
				if n.Reaction[period] == nil {
					n.Reaction[period] = make([]float64, 6)
				}
				ind, err := strconv.ParseInt(l[2], 10, 64)
				if err != nil {
					return err
				}
				val, err := strconv.ParseFloat(strings.Trim(l[5], "\r"), 64)
				if err != nil {
					return err
				}
				n.Reaction[period][ind-1] = val
			}
		}
	}
	return nil
}

// ParseZoubunForm parses current form information.
func (frame *Frame) ParseZoubunForm(lis [][]string, period string) error {
	for _, l := range lis {
		if strings.ToUpper(l[0]) == "NODE:" {
			nnum, err := strconv.ParseInt(l[1], 10, 64)
			if err != nil {
				return err
			}
			if n, ok := frame.Nodes[int(nnum)]; ok {
				disp := make([]float64, 6)
				if len(l) < 9 {
					return errors.New(fmt.Sprintf("ParseZoubunForm: Index Error NODE %d", n.Num))
				}
				for i := 0; i < 6; i++ {
					val, err := strconv.ParseFloat(strings.Trim(l[3+i], "\r"), 64)
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

// ReadLst reads an input file for section list.
func (frame *Frame) ReadLst(filename string) error {
	tmp := make([][]string, 0)
	err := ParseFile(filename, func(words []string) error {
		var err error
		first := strings.ToUpper(words[0])
		if strings.HasPrefix(first, "#") {
			return nil
		}
		switch first {
		default:
			tmp = append(tmp, words)
		case "CODE":
			err = frame.ParseLst(tmp)
			tmp = [][]string{words}
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

// ParseLst parses section lists.
func (frame *Frame) ParseLst(lis [][]string) error {
	var err error
	if len(lis) == 0 || len(lis[0]) <= 2 {
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
		case "WOOD":
			err = frame.ParseLstWood(lis)
		}
	}
	return err
}

// ParseLstSteel parses steel sections.
func (frame *Frame) ParseLstSteel(lis [][]string) error {
	var num int
	var sr SectionRate
	var shape Shape
	var material Steel
	var err error
	tmp, err := strconv.ParseInt(lis[0][1], 10, 64)
	num = int(tmp)
	if _, ok := frame.Sects[num]; !ok {
		return nil
	}
	var size int
	switch lis[1][0] {
	case "HKYOU":
		size = 4
		shape, err = NewHKYOU(lis[1][1 : 1+size])
	case "HWEAK":
		size = 4
		shape, err = NewHWEAK(lis[1][1 : 1+size])
	case "CROSS":
		size = 8
		shape, err = NewCROSS(lis[1][1 : 1+size])
	case "RPIPE":
		size = 4
		shape, err = NewRPIPE(lis[1][1 : 1+size])
	case "CPIPE":
		size = 2
		shape, err = NewCPIPE(lis[1][1 : 1+size])
	case "TKYOU":
		size = 4
		shape, err = NewTKYOU(lis[1][1 : 1+size])
	case "CKYOU":
		size = 4
		shape, err = NewCKYOU(lis[1][1 : 1+size])
	case "CWEAK":
		size = 4
		shape, err = NewCWEAK(lis[1][1 : 1+size])
	case "PLATE":
		size = 2
		shape, err = NewPLATE(lis[1][1 : 1+size])
	case "ANGLE":
		size = 4
		shape, err = NewANGLE(lis[1][1 : 1+size])
	case "SAREA":
		size = 1
		shape, err = NewSAREA(lis[1][1 : 1+size])
	case "THICK":
		size = 1
		shape, err = NewTHICK(lis[1][1 : 1+size])
	default:
		return nil
	}
	if err != nil {
		return err
	}
	switch lis[1][1+size] {
	default:
		material = SN400
	case "SN400":
		material = SN400
	case "SN490":
		material = SN490
	case "SN400T40":
		material = SN400T40
	case "SN490T40":
		material = SN490T40
	case "HT600":
		material = HT600
	case "HT700":
		material = HT700
	case "A6061T6":
		material = A6061T6
	case "A6063T5":
		material = A6063T5
	case "M40J":
		material = M40J
	case "T300":
		material = T300
	}
	switch lis[0][3] {
	case "COLUMN":
		sr = NewSColumn(num, shape, material)
	case "GIRDER":
		sr = NewSGirder(num, shape, material)
	case "BRACE":
		sr = NewSBrace(num, shape, material)
	case "WALL":
		sr = NewSWall(num, shape, material)
	default:
		return nil
	}
	for _, words := range lis[2:] {
		first := strings.ToUpper(words[0])
		switch first {
		case "XFACE", "YFACE", "BBLEN", "BTLEN", "BBFAC", "BTFAC":
			vals := make([]float64, 2)
			for i := 0; i < 2; i++ {
				val, err := strconv.ParseFloat(words[1+i], 64)
				if err != nil {
					return err
				}
				vals[i] = val
			}
			sr.SetValue(first, vals)
		case "MULTI":
			val, err := strconv.ParseFloat(words[1], 64)
			if err != nil {
				return err
			}
			sr.SetValue(first, []float64{val})
		}
		if err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}
	sr.SetName(strings.Trim(lis[0][4], "\""))
	frame.Sects[num].Allow = sr
	return nil
}

// ParseLstSteel parses RC sections.
func (frame *Frame) ParseLstRC(lis [][]string) error {
	var num int
	var sr SectionRate
	var err error
	tmp, err := strconv.ParseInt(lis[0][1], 10, 64)
	num = int(tmp)
	if _, ok := frame.Sects[num]; !ok {
		return nil
	}
	switch lis[0][3] {
	case "COLUMN":
		sr = NewRCColumn(num)
	case "GIRDER":
		sr = NewRCGirder(num)
	case "WALL":
		sr = NewRCWall(num)
	case "SLAB":
		sr = NewRCSlab(num)
	default:
		return nil
	}
	for _, words := range lis[1:] {
		first := strings.ToUpper(words[0])
		switch first {
		case "REINS":
			switch sr.(type) {
			case *RCColumn:
				err = sr.(*RCColumn).AddReins(words[1:])
			case *RCGirder:
				err = sr.(*RCGirder).AddReins(words[1:])
			}
		case "SREIN":
			switch sr.(type) {
			case *RCWall:
				err = sr.(*RCWall).SetSrein(words[1:])
			case *RCSlab:
				err = sr.(*RCSlab).SetSrein(words[1:])
			}
		case "HOOPS":
			switch sr.(type) {
			case *RCColumn:
				err = sr.(*RCColumn).SetHoops(words[1:])
			case *RCGirder:
				err = sr.(*RCGirder).SetHoops(words[1:])
			}
		case "CRECT", "THICK":
			switch sr.(type) {
			case *RCColumn:
				err = sr.(*RCColumn).SetConcrete(words)
			case *RCGirder:
				err = sr.(*RCGirder).SetConcrete(words)
			case *RCWall:
				err = sr.(*RCWall).SetConcrete(words)
			case *RCSlab:
				err = sr.(*RCSlab).SetConcrete(words)
			}
		case "XFACE", "YFACE", "BBLEN", "BTLEN", "BBFAC", "BTFAC", "WRECT":
			vals := make([]float64, 2)
			for i := 0; i < 2; i++ {
				val, err := strconv.ParseFloat(words[1+i], 64)
				if err == nil {
					vals[i] = val
				}
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
	frame.Sects[num].Allow = sr
	return nil
}

// ParseLstSteel parses wood sections.
func (frame *Frame) ParseLstWood(lis [][]string) error {
	var num int
	var sr SectionRate
	var err error
	tmp, err := strconv.ParseInt(lis[0][1], 10, 64)
	num = int(tmp)
	if _, ok := frame.Sects[num]; !ok {
		return nil
	}
	switch lis[0][3] {
	case "COLUMN", "GIRDER":
		var size int
		var shape Shape
		var material Wood
		switch lis[1][0] {
		case "PLATE":
			size = 2
			shape, err = NewPLATE(lis[1][1 : 1+size])
		default:
			return nil
		}
		if err != nil {
			return err
		}
		switch lis[1][1+size] {
		default:
			material = S_E70
		case "S-E70", "E70SUGI":
			material = S_E70
		case "H-E70", "E70HINOKI":
			material = H_E70
		case "H-E90", "E90HINOKI":
			material = H_E90
		case "M-E90":
			material = M_E90
		case "M-E110":
			material = M_E110
		case "E95-F270":
			material = E95_F270
		case "E95-F315":
			material = E95_F315
		case "E120-F330":
			material = E120_F330
		}
		if lis[0][3] == "COLUMN" {
			sr = NewWoodColumn(num, shape, material)
		} else {
			sr = NewWoodGirder(num, shape, material)
		}
	case "WALL":
		sr = NewWoodWall(num)
		switch lis[1][0] {
		case "THICK":
			sr.(*WoodWall).SetWood(lis[1])
		default:
			return nil
		}
	case "SLAB":
		sr = NewWoodSlab(num)
		switch lis[1][0] {
		case "THICK":
			sr.(*WoodSlab).SetWood(lis[1])
		default:
			return nil
		}
	default:
		return nil
	}
	for _, words := range lis[2:] {
		first := strings.ToUpper(words[0])
		switch first {
		case "XFACE", "YFACE", "BBLEN", "BTLEN", "BBFAC", "BTFAC":
			vals := make([]float64, 2)
			for i := 0; i < 2; i++ {
				val, err := strconv.ParseFloat(words[1+i], 64)
				if err == nil {
					vals[i] = val
				}
			}
			sr.SetValue(first, vals)
		case "MULTI":
			val, err := strconv.ParseFloat(words[1], 64)
			if err != nil {
				return err
			}
			sr.SetValue(first, []float64{val})
		}
		if err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}
	sr.SetName(strings.Trim(lis[0][4], "\""))
	frame.Sects[num].Allow = sr
	return nil
}

// ReadRat reads an output file for section rate.
func (frame *Frame) ReadRat(filename string) error {
	err := ParseFile(filename, func(words []string) error {
		enum, err := strconv.ParseInt(words[1], 10, 64)
		rate := make([]float64, len(words)-4)
		for i := 0; i < len(words)-4; i++ {
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

// ReadWgt reads an output file for node weight.
func (frame *Frame) ReadWgt(filename string) error {
	f, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	var lis []string
	if ok := strings.HasSuffix(string(f), "\r\n"); ok {
		lis = strings.Split(string(f), "\r\n")
	} else {
		lis = strings.Split(string(f), "\n")
	}
	num := len(frame.Nodes)
rwgtloop:
	for _, j := range lis {
		if num == 0 {
			break
		}
		var words []string
		for _, k := range strings.Split(j, " ") {
			if k != "" {
				words = append(words, k)
			}
		}
		if len(words) == 0 {
			continue
		}
		nnum, err := strconv.ParseInt(words[0], 10, 64)
		if err != nil {
			continue rwgtloop
		}
		if n, ok := frame.Nodes[int(nnum)]; ok {
			if len(words) < 4 {
				return errors.New(fmt.Sprintf("ReadWgt: Index Error (NODE %d)", nnum))
			}
			wgt := make([]float64, 3)
			for i := 0; i < 3; i++ {
				val, err := strconv.ParseFloat(words[1+i], 64)
				if err != nil {
					continue rwgtloop
				}
				wgt[i] = val
			}
			n.Weight = wgt
			num--
		}
	}
	return nil
}

// ReadKjn reads an input file for kijun.
func (frame *Frame) ReadKjn(filename string) error {
	err := ParseFile(filename, func(words []string) error {
		if strings.HasPrefix(words[0], "#") {
			return nil
		}
		var err error
		err = frame.ParseKjn(words)
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

// ParseKjn parses kijun information.
func (frame *Frame) ParseKjn(lis []string) error {
	k := NewKijun()
	k.Name = lis[0]
	for i := 0; i < 3; i++ {
		tmp, err := strconv.ParseFloat(lis[i+1], 64)
		if err != nil {
			return err
		}
		k.Start[i] = tmp
	}
	for i := 0; i < 3; i++ {
		tmp, err := strconv.ParseFloat(lis[i+4], 64)
		if err != nil {
			return err
		}
		k.End[i] = tmp
	}
	frame.Kijuns[strings.ToLower(lis[0])] = k
	return nil
}

func (frame *Frame) Filename() (string, string) {
	arclm := false
	if frame.Show.PlotState&PLOT_DEFORMED != 0 || frame.Show.SrcanRate != 0 {
		arclm = true
	} else if frame.Show.NodeCaption != 0 {
		for _, nc := range []uint{NC_DX, NC_DY, NC_DZ, NC_TX, NC_TY, NC_TZ, NC_RX, NC_RY, NC_RZ, NC_MX, NC_MY, NC_MZ} {
			if frame.Show.NodeCaption&nc != 0 {
				arclm = true
				break
			}
		}
	} else if frame.Show.ElemCaption != 0 {
		for _, ec := range []uint{EC_RATE_L, EC_RATE_S, EC_STIFF_X, EC_STIFF_Y, EC_DRIFT_X, EC_DRIFT_Y} {
			if frame.Show.ElemCaption&ec != 0 {
				arclm = true
				break
			}
		}
	} else {
		for _, v := range frame.Show.Stress {
			if v != 0 {
				arclm = true
				break
			}
		}
	}
	var in, out string
	if arclm {
		if s, ok := frame.DataFileName[frame.Show.Period]; ok {
			in = s
		} else {
			in = "-"
		}
		if s, ok := frame.ResultFileName[frame.Show.Period]; ok {
			out = s
		} else {
			out = "-"
		}
	} else {
		in = frame.Path
		out = "-"
	}
	return in, out
}

// WriteInp writes an input file for st frame.
func (frame *Frame) WriteInp(fn string) error {
	var bnum, pnum, snum, inum, nnum, enum, cnum int
	// Bond
	bonds := make([]*Bond, len(frame.Bonds))
	for _, b := range frame.Bonds {
		bonds[bnum] = b
		bnum++
	}
	sort.Sort(BondByNum{bonds})
	// Prop
	props := make([]*Prop, len(frame.Props))
	for _, p := range frame.Props {
		props[pnum] = p
		pnum++
	}
	sort.Sort(PropByNum{props})
	// Sect
	sects := make([]*Sect, len(frame.Sects))
	for _, sec := range frame.Sects {
		if sec.Num < 100 || sec.Num > 900 {
			continue
		}
		sects[snum] = sec
		snum++
	}
	sects = sects[:snum]
	sort.Sort(SectByNum{sects})
	// Pile
	var piles []*Pile
	if len(frame.Piles) >= 1 {
		piles = make([]*Pile, len(frame.Piles))
		for _, i := range frame.Piles {
			piles[inum] = i
			inum++
		}
		sort.Sort(PileByNum{piles})
	}
	// Node
	nodes := make([]*Node, len(frame.Nodes))
	for _, n := range frame.Nodes {
		nodes[nnum] = n
		nnum++
	}
	sort.Sort(NodeByNum{nodes})
	// Elem
	elems := make([]*Elem, len(frame.Elems))
	for _, el := range frame.Elems {
		if el.Etype == WBRACE || el.Etype == SBRACE {
			continue
		}
		if el.Chain == nil {
			elems[enum] = el
			enum++
		}
	}
	elems = elems[:enum]
	sort.Sort(ElemByNum{elems})
	// Chain
	chains := make([]*Chain, len(frame.Chains))
	for _, c := range frame.Chains {
		chains[cnum] = c
		cnum++
	}
	chains = chains[:cnum]
	sort.Slice(chains, func(i, j int) bool {
		return chains[i].Elems()[0].Num < chains[j].Elems()[0].Num
	})
	return writeinp(fn, frame.Title, frame.View, frame.Ai, frame.Wind, bonds, props, sects, piles, nodes, elems, chains)
}

// WriteOutput writes an output file of analysis.
func (frame *Frame) WriteOutput(fn string, p string) error {
	var otp bytes.Buffer
	var nkeys, ekeys []int
	// Elem
	otp.WriteString("\n\n** FORCES OF MEMBER\n\n")
	otp.WriteString("  NO   KT NODE         N        Q1        Q2        MT        M1        M2\n\n")
	for k := range frame.Elems {
		ekeys = append(ekeys, k)
	}
	sort.Ints(ekeys)
	for _, k := range ekeys {
		if !frame.Elems[k].IsLineElem() {
			continue
		}
		otp.WriteString(frame.Elems[k].OutputStress(p))
	}
	// Node
	otp.WriteString("\n\n** DISPLACEMENT OF NODE\n\n")
	otp.WriteString("  NO          U          V          W         KSI         ETA       OMEGA\n\n")
	for k := range frame.Nodes {
		nkeys = append(nkeys, k)
	}
	sort.Ints(nkeys)
	for _, k := range nkeys {
		otp.WriteString(frame.Nodes[k].OutputDisp(p))
	}
	otp.WriteString("\n\n** REACTION\n\n")
	otp.WriteString("  NO  DIRECTION              R    NC\n\n")
	for _, k := range nkeys {
		for i := 0; i < 6; i++ {
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
	otp = AddCR(otp)
	otp.WriteTo(w)
	return nil
}

// WriteReaction writes an output file of reaction.
func (frame *Frame) WriteReaction(fn string, direction int, unit float64) error {
	ns := make([]*Node, len(frame.Nodes))
	for nnum, n := range frame.Nodes {
		ns[nnum] = n
	}
	sort.Sort(NodeByNum{ns})
	return WriteReaction(fn, ns, direction, frame.Show.Unit[0])
}

// ReportZoubunDisp writes an output file which reports displacement data of push-over analysis.
func (frame *Frame) ReportZoubunDisp(fn string, ns []*Node, pers []string, direction int) error {
	var otp bytes.Buffer
	sort.Sort(NodeByZCoord{ns})
	otp.WriteString(fmt.Sprintf("ZOUBUN DISP: %s\n", frame.Name))
	for _, per := range pers {
		per = strings.ToUpper(per)
		if nlap, ok := frame.Nlap[per]; ok {
			otp.WriteString(fmt.Sprintf("PERIOD: %s, DIRECTION: %d\n", per, direction))
			otp.WriteString("LAP     ")
			for _, n := range ns {
				otp.WriteString(fmt.Sprintf(" %8d", n.Num))
			}
			otp.WriteString("\n")
			for i := 0; i < nlap; i++ {
				nper := fmt.Sprintf("%s@%d", per, i+1)
				otp.WriteString(fmt.Sprintf("%8s", nper))
				for _, n := range ns {
					if d, ok := n.Disp[nper]; ok {
						otp.WriteString(fmt.Sprintf(" %8.5f", d[direction]))
					} else {
						otp.WriteString(" --------")
					}
				}
				otp.WriteString("\n")
			}
			otp.WriteString("\n")
		} else {
			return errors.New(fmt.Sprintf("ReportZoubunDisp: unknown period: %s", per))
		}
	}
	w, err := os.Create(fn)
	defer w.Close()
	if err != nil {
		return err
	}
	otp = AddCR(otp)
	otp.WriteTo(w)
	return nil
}

// ReportZoubunReaction writes an output file which reports reaction data of push-over analysis.
func (frame *Frame) ReportZoubunReaction(fn string, ns []*Node, pers []string, direction int) error {
	var otp bytes.Buffer
	sort.Sort(NodeByNum{ns})
	otp.WriteString(fmt.Sprintf("ZOUBUN REACTION: %s\n", frame.Name))
	for _, per := range pers {
		per = strings.ToUpper(per)
		if nlap, ok := frame.Nlap[per]; ok {
			otp.WriteString(fmt.Sprintf("PERIOD: %s, DIRECTION: %d\n", per, direction))
			otp.WriteString("LAP     ")
			for _, n := range ns {
				otp.WriteString(fmt.Sprintf(" %8d", n.Num))
			}
			otp.WriteString("\n")
			for i := 0; i < nlap; i++ {
				nper := fmt.Sprintf("%s@%d", per, i+1)
				otp.WriteString(fmt.Sprintf("%8s", nper))
				for _, n := range ns {
					if r, ok := n.Reaction[nper]; ok {
						otp.WriteString(fmt.Sprintf(" %8.3f", r[direction]))
					} else {
						otp.WriteString(" --------")
					}
				}
				otp.WriteString("\n")
			}
			otp.WriteString("\n")
		} else {
			return errors.New(fmt.Sprintf("ReportZoubunReaction: unknown period: %s", per))
		}
	}
	w, err := os.Create(fn)
	defer w.Close()
	if err != nil {
		return err
	}
	otp = AddCR(otp)
	otp.WriteTo(w)
	return nil
}

// WriteKjn writes an input file for kijun.
func (frame *Frame) WriteKjn(fn string) error {
	var otp bytes.Buffer
	ks := make([]string, len(frame.Kijuns))
	i := 0
	for k := range frame.Kijuns {
		ks[i] = k
		i++
	}
	natural.Sort(ks)
	for _, k := range ks {
		otp.WriteString(frame.Kijuns[k].String())
	}
	w, err := os.Create(fn)
	defer w.Close()
	if err != nil {
		return err
	}
	otp = AddCR(otp)
	otp.WriteTo(w)
	return nil
}

// WritePlateWeight reports weight data of plate elements.
func (frame *Frame) WritePlateWeight(fn string) error {
	sects := make([]*Sect, len(frame.Sects))
	snum := 0
	for _, sec := range frame.Sects {
		if sec.Num < 700 || sec.Num >= 900 {
			continue
		}
		sects[snum] = sec
		snum++
	}
	sects = sects[:snum]
	sort.Sort(SectByNum{sects})
	var otp bytes.Buffer
	otp.WriteString("CODE E          t        SLAB     FRAME    EQ\n")
	otp.WriteString("     [tf/m2]    [m]      [tf/m2]  [tf/m2]  [tf/m2]\n")
	for _, sec := range sects {
		w := sec.Weight()
		if sec.HasBrace() {
			otp.WriteString(fmt.Sprintf("%4d %10.1f %6.3f    %6.3f   %6.3f   %6.3f\n", sec.Num, sec.Figs[0].Prop.EL, sec.Figs[0].Value["THICK"], w[0], w[1], w[2]))
		} else {
			otp.WriteString(fmt.Sprintf("%4d %10s %6s    %6.3f   %6.3f   %6.3f\n", sec.Num, "", "", w[0], w[1], w[2]))
		}
	}
	w, err := os.Create(fn)
	defer w.Close()
	if err != nil {
		return err
	}
	otp.WriteTo(w)
	return nil
}

// WriteDxf2D writes a dxf file of frame with projected coordinates.
func (frame *Frame) WriteDxf2D(filename string, scale float64) error {
	d := dxf.NewDrawing()
	for _, n := range frame.Nodes {
		frame.View.ProjectNode(n)
	}
	for _, el := range frame.Elems {
		_, err := d.Layer(fmt.Sprintf("%s%d", ETYPES[el.Etype], el.Sect.Num), true)
		if err != nil {
			d.AddLayer(fmt.Sprintf("%s%d", ETYPES[el.Etype], el.Sect.Num), dxf.ColorIndex(IntColorList(el.Sect.Color)), dxf.DefaultLineType, true)
		}
		if el.IsLineElem() {
			d.Line(el.Enod[0].Pcoord[0]*scale, el.Enod[0].Pcoord[1]*scale, 0.0, el.Enod[1].Pcoord[0]*scale, el.Enod[1].Pcoord[1]*scale, 0.0)
		}
	}
	err := d.SaveAs(filename)
	if err != nil {
		return err
	}
	return nil
}

// DrawDxfSection adds a section figure to a drawing.
func (frame *Frame) DrawDxfSection(d *drawing.Drawing, el *Elem, position []float64, scale float64) error {
	if !el.IsLineElem() {
		return NotLineElem("DrawDxfSection")
	}
	if frame.Sects[el.Sect.Num].Allow == nil {
		return fmt.Errorf("section not found")
	}
	switch al := frame.Sects[el.Sect.Num].Allow.(type) {
	case *SColumn:
		sh := al.Shape
		switch sh.(type) {
		case HKYOU, HWEAK, CROSS, RPIPE, PLATE, ANGLE:
			vertices := sh.Vertices()
			el.DrawDxfSection(d, position, scale, vertices)
		case CPIPE:
			direction := el.Direction(true)
			c, err := d.Circle(position[0]*scale, position[1]*scale, position[2]*scale, sh.(CPIPE).D*0.01*0.5*scale)
			if err == nil {
				dxf.SetExtrusion(c, direction)
			}
			c, err = d.Circle(position[0]*scale, position[1]*scale, position[2]*scale, (sh.(CPIPE).D*0.5-sh.(CPIPE).T)*0.01*scale)
			if err == nil {
				dxf.SetExtrusion(c, direction)
			}
		}
	case *RCColumn:
		vertices := al.CShape.Vertices()
		el.DrawDxfSection(d, position, scale, vertices)
		direction := el.Direction(true)
		for _, reins := range al.Reins {
			pos := make([]float64, 3)
			pos[0] = (position[0] + (reins.Position[0]*el.Strong[0]+reins.Position[1]*el.Weak[0])*0.01) * scale
			pos[1] = (position[1] + (reins.Position[0]*el.Strong[1]+reins.Position[1]*el.Weak[1])*0.01) * scale
			pos[2] = (position[2] + (reins.Position[0]*el.Strong[2]+reins.Position[1]*el.Weak[2])*0.01) * scale
			c, err := d.Circle(pos[0], pos[1], pos[2], reins.Radius()*0.01*scale)
			if err == nil {
				dxf.SetExtrusion(c, direction)
			}
		}
	case *RCGirder:
		vertices := al.CShape.Vertices()
		el.DrawDxfSection(d, position, scale, vertices)
		direction := el.Direction(true)
		for _, reins := range al.Reins {
			pos := make([]float64, 3)
			pos[0] = (position[0] + (reins.Position[0]*el.Strong[0]+reins.Position[1]*el.Weak[0])*0.01) * scale
			pos[1] = (position[1] + (reins.Position[0]*el.Strong[1]+reins.Position[1]*el.Weak[1])*0.01) * scale
			pos[2] = (position[2] + (reins.Position[0]*el.Strong[2]+reins.Position[1]*el.Weak[2])*0.01) * scale
			c, err := d.Circle(pos[0], pos[1], pos[2], reins.Radius()*0.01*scale)
			if err == nil {
				dxf.SetExtrusion(c, direction)
			}
		}
	case *WoodColumn:
		sh := al.Shape
		switch sh.(type) {
		case PLATE:
			vertices := sh.Vertices()
			el.DrawDxfSection(d, position, scale, vertices)
		}
	}
	return nil
}

// WriteDxf3D writes a dxf file of frame with 3D coordinates.
func (frame *Frame) WriteDxf3D(filename string, scale float64) error {
	d := dxf.NewDrawing()
	for _, n := range frame.Nodes {
		val := 0
		for i := 0; i < 6; i++ {
			if n.Conf[i] {
				val += 1 << uint(5-i)
			}
		}
		if val != 0 {
			_, err := d.Layer(fmt.Sprintf("CONF%06b", val), true)
			if err != nil {
				d.AddLayer(fmt.Sprintf("CONF%06b", val), dxf.DefaultColor, dxf.DefaultLineType, true)
			}
			d.Point(n.Coord[0]*scale, n.Coord[1]*scale, n.Coord[2]*scale)
		}
	}
	for _, el := range frame.Elems {
		_, err := d.Layer(fmt.Sprintf("%s%d", ETYPES[el.Etype], el.Sect.Num), true)
		if err != nil {
			d.AddLayer(fmt.Sprintf("%s%d", ETYPES[el.Etype], el.Sect.Num), dxf.ColorIndex(IntColorList(el.Sect.Color)), dxf.DefaultLineType, true)
		}
		if el.IsLineElem() {
			l, err := d.Line(el.Enod[0].Coord[0]*scale, el.Enod[0].Coord[1]*scale, el.Enod[0].Coord[2]*scale, el.Enod[1].Coord[0]*scale, el.Enod[1].Coord[1]*scale, el.Enod[1].Coord[2]*scale)
			if err != nil {
				continue
			}
			switch el.BondState() {
			case PIN_RIGID:
				d.Group("PINRIGID", "PIN-RIGID", l)
			case RIGID_PIN:
				d.Group("RIGIDPIN", "RIGID-PIN", l)
			case PIN_PIN:
				d.Group("PINPIN", "PIN-PIN", l)
			}
			_, err = d.Layer("SECTION", true)
			if err != nil {
				d.AddLayer("SECTION", dxf.DefaultColor, dxf.DefaultLineType, true)
			}
			frame.DrawDxfSection(d, el, el.MidPoint(), scale)
		} else {
			switch el.Enods {
			case 3:
				d.ThreeDFace([][]float64{[]float64{el.Enod[0].Coord[0] * scale, el.Enod[0].Coord[1] * scale, el.Enod[0].Coord[2] * scale},
					[]float64{el.Enod[1].Coord[0] * scale, el.Enod[1].Coord[1] * scale, el.Enod[1].Coord[2] * scale},
					[]float64{el.Enod[2].Coord[0] * scale, el.Enod[2].Coord[1] * scale, el.Enod[2].Coord[2] * scale}})
			case 4:
				d.ThreeDFace([][]float64{[]float64{el.Enod[0].Coord[0] * scale, el.Enod[0].Coord[1] * scale, el.Enod[0].Coord[2] * scale},
					[]float64{el.Enod[1].Coord[0] * scale, el.Enod[1].Coord[1] * scale, el.Enod[1].Coord[2] * scale},
					[]float64{el.Enod[2].Coord[0] * scale, el.Enod[2].Coord[1] * scale, el.Enod[2].Coord[2] * scale},
					[]float64{el.Enod[3].Coord[0] * scale, el.Enod[3].Coord[1] * scale, el.Enod[3].Coord[2] * scale}})
			}
		}
	}
	err := d.SaveAs(filename)
	if err != nil {
		return err
	}
	return nil
}

// WriteDxfCrosssection writes a dxf file of crosssection
func (frame *Frame) WriteDxfCrosssection(filename string, axis int, min, max float64, side int, scale float64, textheight float64, axissize float64) error {
	if side != 0 && side != 1 {
		return fmt.Errorf("unknown side")
	}
	var x, y, kx, ky int
	switch axis {
	default:
		return fmt.Errorf("unknown axis")
	case 0:
		x = 1
		y = 2
		kx = 1
		ky = 0
	case 1:
		x = 0
		y = 2
		kx = 0
		ky = 1
	case 2:
		x = 0
		y = 1
		kx = 0
		ky = 1
	}
	d := dxf.NewDrawing()
	d.Header().LtScale = 100.0
	MCDASH6, _ := d.AddLineType("MCDASH6", "MiniCad+ Generated", 3.175, -1.058075, 0.352692, -0.705771)
	SIMPLEX, _ := d.AddStyle("SIMPLEX", "simplex.shx", "bigfont.shx", true)
	SIMPLEX.WidthFactor = 0.75
	d.AddLayer("KIJUN", dxfcolor.Red, MCDASH6, false)
	d.AddLayer("AXIS", dxf.DefaultColor, dxf.DefaultLineType, false)
	d.AddLayer("MOJI", dxf.DefaultColor, dxf.DefaultLineType, false)
dxfkijun:
	for _, k := range frame.Kijuns {
		switch axis {
		case 0, 1:
			if k.Start[1-axis] != k.End[1-axis] {
				continue dxfkijun
			}
		case 2:
			if k.Start[2] != 0.0 || k.End[2] != 0.0 {
				continue dxfkijun
			}
		}
		d.Layer("KIJUN", true)
		l, _ := d.Line(k.Start[kx]*scale, k.Start[ky]*scale, 0.0, k.End[kx]*scale, k.End[ky]*scale, 0.0)
		l.SetLtscale(2.0)
		d.Layer("AXIS", true)
		direction := k.Direction()
		d.Circle(k.Start[kx]*scale-direction[kx]*axissize, k.Start[ky]*scale-direction[ky]*axissize, 0.0, axissize)
		d.Layer("MOJI", true)
		t, _ := d.Text(k.Name, k.Start[kx]*scale-direction[kx]*axissize, k.Start[ky]*scale-direction[ky]*axissize, 0.0, textheight)
		t.Anchor(dxfentity.CENTER_CENTER)
	}
	setlayer := func(et int) {
		lname := "0"
		col := dxf.DefaultColor
		lt := dxf.DefaultLineType
		switch et {
		case COLUMN:
			lname = "HASHIRA"
			col = dxfcolor.Yellow
		case GIRDER:
			lname = "HARI"
			col = dxfcolor.Cyan
		case BRACE:
			lname = "BRACE"
			col = dxfcolor.Green
		}
		_, err := d.Layer(lname, true)
		if err != nil {
			d.AddLayer(lname, col, lt, true)
		}
	}
	settextlayer := func(et int) {
		lname := "0"
		col := dxf.DefaultColor
		lt := dxf.DefaultLineType
		switch et {
		case COLUMN:
			lname = "TXTCOLUMN"
			col = dxfcolor.Yellow
		case GIRDER:
			lname = "TXTGIRDER"
			col = dxfcolor.Cyan
		case BRACE:
			lname = "TXTBRACE"
			col = dxfcolor.Green
		}
		_, err := d.Layer(lname, true)
		if err != nil {
			d.AddLayer(lname, col, lt, true)
		}
	}
	chains := make(map[int]*Chain)
	for _, el := range frame.Elems {
		if !el.IsLineElem() || el.Etype == WBRACE || el.Etype == SBRACE {
			continue
		}
		if min <= el.Enod[0].Coord[axis] && el.Enod[1].Coord[axis] <= max {
			if el.Chain != nil && el.Chain.IsStraight(1e-2) { // TODO: curved Chain
				chains[el.Chain.Num()] = el.Chain
				continue
			}
			direction := el.Direction(true)
			textcoord := []float64{
				0.5*(el.Enod[0].Coord[x]+el.Enod[1].Coord[x])*scale - direction[y]*textheight*0.5,
				0.5*(el.Enod[0].Coord[y]+el.Enod[1].Coord[y])*scale + direction[x]*textheight*0.5,
			}
			name := strings.TrimSpace(strings.Split(el.Sect.Name, ":")[0])
			setlayer(el.Etype)
			if al := frame.Sects[el.Sect.Num].Allow; al != nil {
				name = al.Name()
				if b, ok := al.(Breadther); ok {
					b1 := b.Breadth(true) * 0.01
					b2 := b.Breadth(false) * 0.01
					diff := []float64{0.5 * (b1*el.Strong[x] + b2*el.Weak[x]), 0.5 * (b1*el.Strong[y] + b2*el.Weak[y])}
					d.LwPolyline(true,
						[]float64{(el.Enod[0].Coord[x] + diff[0]) * scale, (el.Enod[0].Coord[y] + diff[1]) * scale, 0.0},
						[]float64{(el.Enod[1].Coord[x] + diff[0]) * scale, (el.Enod[1].Coord[y] + diff[1]) * scale, 0.0},
						[]float64{(el.Enod[1].Coord[x] - diff[0]) * scale, (el.Enod[1].Coord[y] - diff[1]) * scale, 0.0},
						[]float64{(el.Enod[0].Coord[x] - diff[0]) * scale, (el.Enod[0].Coord[y] - diff[1]) * scale, 0.0})
					for i := 0; i < 2; i++ {
						textcoord[i] += diff[i] * scale
					}
				} else {
					d.Line(el.Enod[0].Coord[x]*scale, el.Enod[0].Coord[y]*scale, 0.0, el.Enod[1].Coord[x]*scale, el.Enod[1].Coord[y]*scale, 0.0)
				}
			} else {
				d.Line(el.Enod[0].Coord[x]*scale, el.Enod[0].Coord[y]*scale, 0.0, el.Enod[1].Coord[x]*scale, el.Enod[1].Coord[y]*scale, 0.0)
			}
			settextlayer(el.Etype)
			t, _ := d.Text(name, textcoord[0], textcoord[1], 0.0, textheight)
			t.Anchor(dxfentity.CENTER_BOTTOM)
			t.Rotation = math.Atan2(el.Enod[1].Coord[y]-el.Enod[0].Coord[y], el.Enod[1].Coord[x]-el.Enod[0].Coord[x]) * 180.0 / math.Pi
		}
	}
	for _, v := range chains {
		el1 := v.Elems()[0]
		el2 := v.Elems()[v.Size()-1]
		direction := el1.Direction(true)
		textcoord := []float64{
			0.5*(el1.Enod[0].Coord[x]+el2.Enod[1].Coord[x])*scale - direction[y]*textheight*0.5,
			0.5*(el1.Enod[0].Coord[y]+el2.Enod[1].Coord[y])*scale + direction[x]*textheight*0.5,
		}
		name := strings.TrimSpace(strings.Split(el1.Sect.Name, ":")[0])
		setlayer(el1.Etype)
		if al := frame.Sects[el1.Sect.Num].Allow; al != nil {
			name = al.Name()
			if b, ok := al.(Breadther); ok {
				b1 := b.Breadth(true) * 0.01
				b2 := b.Breadth(false) * 0.01
				diff := []float64{0.5 * (b1*el1.Strong[x] + b2*el1.Weak[x]), 0.5 * (b1*el1.Strong[y] + b2*el1.Weak[y])}
				d.LwPolyline(true,
					[]float64{(el1.Enod[0].Coord[x] + diff[0]) * scale, (el1.Enod[0].Coord[y] + diff[1]) * scale, 0.0},
					[]float64{(el2.Enod[1].Coord[x] + diff[0]) * scale, (el2.Enod[1].Coord[y] + diff[1]) * scale, 0.0},
					[]float64{(el2.Enod[1].Coord[x] - diff[0]) * scale, (el2.Enod[1].Coord[y] - diff[1]) * scale, 0.0},
					[]float64{(el1.Enod[0].Coord[x] - diff[0]) * scale, (el1.Enod[0].Coord[y] - diff[1]) * scale, 0.0})
				for i := 0; i < 2; i++ {
					textcoord[i] += diff[i] * scale
				}
			} else {
				d.Line(el1.Enod[0].Coord[x]*scale, el1.Enod[0].Coord[y]*scale, 0.0, el2.Enod[1].Coord[x]*scale, el2.Enod[1].Coord[y]*scale, 0.0)
			}
		} else {
			d.Line(el1.Enod[0].Coord[x]*scale, el1.Enod[0].Coord[y]*scale, 0.0, el2.Enod[1].Coord[x]*scale, el2.Enod[1].Coord[y]*scale, 0.0)
		}
		settextlayer(el1.Etype)
		t, _ := d.Text(name, textcoord[0], textcoord[1], 0.0, textheight)
		t.Anchor(dxfentity.CENTER_BOTTOM)
		t.Rotation = math.Atan2(el1.Enod[1].Coord[y]-el1.Enod[0].Coord[y], el1.Enod[1].Coord[x]-el1.Enod[0].Coord[x]) * 180.0 / math.Pi
	}
	for _, el := range frame.Fence(axis, []float64{min, max}[side], false) {
		setlayer(el.Etype)
		os := make([]float64, 3)
		ow := make([]float64, 3)
		for i := 0; i < 3; i++ {
			os[i] = el.Strong[i]
			ow[i] = el.Weak[i]
		}
		if axis == 2 {
			s, w, err := PrincipalAxis([]float64{0.0, 0.0, 1.0}, el.Cang)
			if err != nil {
				continue
			}
			el.Strong = s
			el.Weak = w
		} else {
			s, w, err := PrincipalAxis([]float64{0.0, 0.0, 1.0}, el.Cang-0.5*math.Pi)
			if err != nil {
				continue
			}
			el.Strong = s
			el.Weak = w
		}
		err := frame.DrawDxfSection(d, el, []float64{el.Enod[0].Coord[x], el.Enod[0].Coord[y], 0.0}, scale)
		if err == nil {
			settextlayer(el.Etype)
			name := frame.Sects[el.Sect.Num].Allow.Name()
			textcoord := []float64{
				el.Enod[0].Coord[x]*scale + textheight,
				el.Enod[0].Coord[y]*scale + textheight,
			}
			d.Text(name, textcoord[0], textcoord[1], 0.0, textheight)
		}
		for i := 0; i < 3; i++ {
			el.Strong[i] = os[i]
			el.Weak[i] = ow[i]
		}
	}
	return d.SaveAs(filename)
}

// WriteDxfPlan writes a dxf file of plan
func (frame *Frame) WriteDxfPlan(filename string, floor int, scale float64, textheight float64, axissize float64) error {
	if floor <= 0 || floor >= len(frame.Ai.Boundary) {
		return fmt.Errorf("out of bounds")
	}
	min := frame.Ai.Boundary[floor-1]
	max := frame.Ai.Boundary[floor]
	return frame.WriteDxfCrosssection(filename, 2, min, max, 1, scale, textheight, axissize)
}

// Check checks whether all elements are valid or not.
func (frame *Frame) Check() ([]*Node, []*Elem, bool) {
	ok := true
	ns := make([]*Node, len(frame.Nodes))
	els := make([]*Elem, len(frame.Elems))
	nnum := 0
	enum := 0
	for _, el := range frame.Elems {
		if v, err := el.IsValidElem(); !v {
			ok = false
			fmt.Println(err.Error())
			els[enum] = el
			enum++
		}
	}
	return ns[:nnum], els[:enum], ok
}

// Distance measures the distance between n1 and n2 with original coordinates.
func (frame *Frame) Distance(n1, n2 *Node) (dx, dy, dz, d float64) {
	dx = n2.Coord[0] - n1.Coord[0]
	dy = n2.Coord[1] - n1.Coord[1]
	dz = n2.Coord[2] - n1.Coord[2]
	d = math.Sqrt(math.Pow(dx, 2) + math.Pow(dy, 2) + math.Pow(dz, 2))
	return
}

// DeformedDistance measures the distance between n1 and n2 with deformed coordinates.
func (frame *Frame) DeformedDistance(n1, n2 *Node) (dx, dy, dz, d float64) {
	dx = n2.Coord[0] + n2.ReturnDisp(frame.Show.Period, 0) - n1.Coord[0] - n1.ReturnDisp(frame.Show.Period, 0)
	dy = n2.Coord[1] + n2.ReturnDisp(frame.Show.Period, 1) - n1.Coord[1] - n1.ReturnDisp(frame.Show.Period, 1)
	dz = n2.Coord[2] + n2.ReturnDisp(frame.Show.Period, 2) - n1.Coord[2] - n1.ReturnDisp(frame.Show.Period, 2)
	d = math.Sqrt(math.Pow(dx, 2) + math.Pow(dy, 2) + math.Pow(dz, 2))
	return
}

// Direction returns a vector from n1 to n2.
func (frame *Frame) Direction(n1, n2 *Node, normalize bool) []float64 {
	var l float64
	d := make([]float64, 3)
	for i := 0; i < 3; i++ {
		d[i] = n2.Coord[i] - n1.Coord[i]
	}
	if normalize {
		for i := 0; i < 3; i++ {
			l += d[i] * d[i]
		}
		l = math.Sqrt(l)
		for i := 0; i < 3; i++ {
			d[i] /= l
		}
		return d
	} else {
		return d
	}
}

// Move moves frame.
func (frame *Frame) Move(x, y, z float64) {
	for _, n := range frame.Nodes {
		n.Move(x, y, z)
	}
}

// Scale scales frame.
func (frame *Frame) Scale(center []float64, factor float64) {
	for _, n := range frame.Nodes {
		n.Scale(center, factor, factor, factor)
	}
}

// Rotate rotates frame.
func (frame *Frame) Rotate(center, vector []float64, angle float64) {
	for _, n := range frame.Nodes {
		n.Rotate(center, vector, angle)
	}
}

// DefaultProp returns the first PROP.
func (frame *Frame) DefaultProp() *Prop {
	pnums := make([]int, len(frame.Props))
	i := 0
	for k, _ := range frame.Props {
		pnums[i] = int(k)
		i++
	}
	sort.Ints(pnums)
	return frame.Props[pnums[0]]
}

// DefaultSect returns the first SECT.
func (frame *Frame) DefaultSect() *Sect {
	l := len(frame.Sects)
	if l == 0 {
		s := frame.AddSect(101)
		return s
	}
	snums := make([]int, l)
	i := 0
	for k, _ := range frame.Sects {
		snums[i] = int(k)
		i++
	}
	sort.Ints(snums)
	return frame.Sects[snums[0]]
}

// AddSect adds SECT.
func (frame *Frame) AddSect(num int) *Sect {
	sec := NewSect()
	sec.Num = num
	frame.Sects[num] = sec
	frame.Show.Sect[num] = true
	return sec
}

// AddPropAndSect adds PROP & SECT from an input file for st frame.
func (frame *Frame) AddPropAndSect(filename string, overwrite bool) error {
	newsects := make(map[int]*Sect)
	tmp := make([]string, 0)
	var sec *Sect
	err := ParseFile(filename, func(words []string) error {
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
			tmp = append(tmp, words...)
		case "PROP", "SECT":
			sec, err = frame.ParsePropAndSect(tmp, overwrite)
			tmp = words
		}
		if err != nil {
			return err
		}
		if sec != nil {
			newsects[sec.Num] = sec
		}
		return nil
	})
	if err != nil {
		return err
	}
	sec, err = frame.ParsePropAndSect(tmp, overwrite)
	if err != nil {
		return err
	}
	if sec != nil {
		newsects[sec.Num] = sec
	}
	if len(newsects) > 0 {
		for _, el := range frame.Elems {
			if sec, ok := newsects[el.Sect.Num]; ok {
				el.Sect = sec
			}
		}
	}
	return nil
}

// ParsePropAndSect parses PROP & SECT.
func (frame *Frame) ParsePropAndSect(lis []string, overwrite bool) (*Sect, error) {
	var err error
	var sec *Sect
	if len(lis) == 0 {
		return nil, nil
	}
	first := lis[0]
	switch first {
	case "SECT":
		sec, err = frame.ParseSect(lis, overwrite)
	case "PROP":
		_, err = frame.ParseProp(lis, overwrite)
	}
	return sec, err
}

// AddNode adds NODE at (x, y, z).
func (frame *Frame) AddNode(x, y, z float64) *Node {
	node := NewNode()
	node.Coord[0] = x
	node.Coord[1] = y
	node.Coord[2] = z
	frame.Maxnnum++
	node.Num = frame.Maxnnum
	frame.Nodes[node.Num] = node
	return node
}

// SearchNode searches NODE at (x, y, z).
func (frame *Frame) SearchNode(x, y, z float64) *Node {
	for _, n := range frame.Nodes {
		if math.Sqrt(math.Pow(x-n.Coord[0], 2)+math.Pow(y-n.Coord[1], 2)+math.Pow(z-n.Coord[2], 2)) <= 1e-4 {
			return n
		}
	}
	return nil
}

// CoordNode returns NODE at (x, y, z).
// If not found, it creates new NODE.
func (frame *Frame) CoordNode(x, y, z, eps float64) (*Node, bool) {
	for _, n := range frame.Nodes {
		if math.Sqrt(math.Pow(x-n.Coord[0], 2)+math.Pow(y-n.Coord[1], 2)+math.Pow(z-n.Coord[2], 2)) <= eps {
			return n, false
		}
	}
	return frame.AddNode(x, y, z), true
}

// AddElem adds ELEM to frame.
func (frame *Frame) AddElem(enum int, el *Elem) {
	if enum < 0 {
		frame.Maxenum++
		el.Frame = frame
		el.Num = frame.Maxenum
		frame.Elems[el.Num] = el
	} else {
		if _, exist := frame.Elems[enum]; exist {
			fmt.Printf("AddElem: Elem %d already exists\n", enum)
			frame.AddElem(-1, el)
		} else {
			el.Num = enum
			el.Frame = frame
			frame.Elems[el.Num] = el
		}
	}
}

// AddLineElem adds a line element to frame.
func (frame *Frame) AddLineElem(enum int, ns []*Node, sect *Sect, etype int) (elem *Elem) {
	elem = NewLineElem(ns, sect, etype)
	frame.AddElem(enum, elem)
	return elem
}

// AddPlateElem adds a plate element to frame.
func (frame *Frame) AddPlateElem(enum int, ns []*Node, sect *Sect, etype int) (elem *Elem) {
	elem = NewPlateElem(ns, sect, etype)
	frame.AddElem(enum, elem)
	return elem
}

// AddKijun adds a kijun to frame.
func (frame *Frame) AddKijun(name string, start, end []float64) (*Kijun, error) {
	if _, exists := frame.Kijuns[name]; exists {
		return nil, errors.New(fmt.Sprintf("kijun %s already exists", name))
	}
	k := NewKijun()
	k.Name = name
	k.Start = start
	k.End = end
	frame.Kijuns[strings.ToLower(name)] = k
	return k, nil
}

// AddMeasure adds a measure to frame.
func (frame *Frame) AddMeasure(start, end, direction []float64) *Measure {
	m := NewMeasure(start, end, direction)
	m.Frame = frame
	frame.Measures = append(frame.Measures, m)
	return m
}

// AddArc adds ARC to frame.
func (frame *Frame) AddArc(p [][]float64, eps float64) {
	n1, _ := frame.CoordNode(p[0][0], p[0][1], p[0][2], eps)
	n2, _ := frame.CoordNode(p[1][0], p[1][1], p[1][2], eps)
	n3, _ := frame.CoordNode(p[2][0], p[2][1], p[2][2], eps)
	a := NewArc([]*Node{n1, n2, n3})
	a.Frame = frame
	frame.Arcs = append(frame.Arcs, a)
}

// NodeInBox returns nodes contained in the box which has n1 and n2 on both ends of its diagonal.
func (frame *Frame) NodeInBox(n1, n2 *Node, eps float64) []*Node {
	var minx, miny, minz float64
	var maxx, maxy, maxz float64
	if n1.Coord[0] < n2.Coord[0] {
		minx = n1.Coord[0]
		maxx = n2.Coord[0]
	} else {
		minx = n2.Coord[0]
		maxx = n1.Coord[0]
	}
	if n1.Coord[1] < n2.Coord[1] {
		miny = n1.Coord[1]
		maxy = n2.Coord[1]
	} else {
		miny = n2.Coord[1]
		maxy = n1.Coord[1]
	}
	if n1.Coord[2] < n2.Coord[2] {
		minz = n1.Coord[2]
		maxz = n2.Coord[2]
	} else {
		minz = n2.Coord[2]
		maxz = n1.Coord[2]
	}
	rtn := make([]*Node, 0, len(frame.Nodes))
	var i int
	for _, n := range frame.Nodes {
		if minx-eps <= n.Coord[0] && n.Coord[0] <= maxx+eps && miny-eps <= n.Coord[1] && n.Coord[1] <= maxy+eps && minz-eps <= n.Coord[2] && n.Coord[2] <= maxz+eps {
			rtn = append(rtn, n)
			i++
		}
	}
	return rtn[:i]
}

// SearchElem searches such element that has all nodes of ns in its ENOD.
// ENOD ⊇ ns
// If len(ns) == 0, it returns all elements.
func (frame *Frame) SearchElem(ns ...*Node) []*Elem {
	els := make([]*Elem, 0, len(frame.Elems))
	num := 0
	l := len(ns)
	for _, el := range frame.Elems {
		count := 0
		found := make([]bool, len(el.Enod))
	loopse:
		for _, n := range ns {
			for i, en := range el.Enod {
				if found[i] {
					continue
				}
				if en == n {
					found[i] = true
					count++
					continue loopse
				}
			}
		}
		if count == l {
			els = append(els, el)
			num++
		}
	}
	return els[:num]
}

// NodeToElemAny searches such element that any of its ENOD is in ns.
// ENOD ∩ ns ≠ φ
// If len(ns) == 0, it returns no element.
func (frame *Frame) NodeToElemAny(ns ...*Node) []*Elem {
	els := make([]*Elem, 0, len(frame.Elems))
	num := 0
	for _, el := range frame.Elems {
	loop:
		for _, en := range el.Enod {
			for _, n := range ns {
				if en == n {
					els = append(els, el)
					num++
					break loop
				}
			}
		}
	}
	return els[:num]
}

// NodeToElemAll searches such element that all of its ENOD is in ns.
// ENOD ⊆ ns
// If len(ns) == 0, it returns no element.
func (frame *Frame) NodeToElemAll(ns ...*Node) []*Elem {
	var add, found bool
	num := 0
	els := make([]*Elem, 0, len(frame.Elems))
	for _, el := range frame.Elems {
		add = true
		for _, en := range el.Enod {
			found = false
			for _, n := range ns {
				if en == n {
					found = true
					break
				}
			}
			if !found {
				add = false
				break
			}
		}
		if add {
			els = append(els, el)
			num++
		}
	}
	return els[:num]
}

// ElemToNode collects such node that is used by elements in els.
// In other words, it returns the union of ENOD of elements in els.
// ∪ ENOD
// If len(els) == 0, it returns no node.
func (frame *Frame) ElemToNode(els ...*Elem) []*Node {
	var add bool
	ns := make([]*Node, 0, len(frame.Nodes))
	num := 0
	for _, el := range els {
		for _, en := range el.Enod {
			add = true
			for _, n := range ns {
				if en == n {
					add = false
					break
				}
			}
			if add {
				ns = append(ns, en)
				num++
			}
		}
	}
	return ns[:num]
}

// Isolated searches such element that is not connected to supported nodes.
func (frame *Frame) Isolated() ([]*Node, []*Elem) {
	current := make([]*Node, 0)
	for _, n := range frame.Nodes {
		for i := 0; i < 6; i++ {
			if n.Conf[i] {
				current = append(current, n)
				break
			}
		}
	}
	checked := current
	for {
		next := make([]*Node, 0)
		for _, n := range current {
			for _, el := range frame.SearchElem(n) {
				if !el.IsLineElem() {
					continue
				}
				cand := el.Otherside(n)
				add := true
				for _, cn := range current {
					if cand == cn {
						add = false
						break
					}
				}
				for _, cn := range checked {
					if cand == cn {
						add = false
						break
					}
				}
				if add {
					next = append(next, cand)
				}
			}
		}
		if len(next) == 0 {
			break
		} else {
			checked = append(checked, next...)
			current = next
		}
	}
	inodes := make([]*Node, 0)
	for _, n := range frame.Nodes {
		add := true
		for _, cn := range checked {
			if cn == n {
				add = false
			}
		}
		if add {
			inodes = append(inodes, n)
		}
	}
	return inodes, frame.NodeToElemAll(inodes...)
}

func abs(val int) int {
	if val >= 0 {
		return val
	} else {
		return -val
	}
}
func (frame *Frame) Fence(axis int, coord float64, plate bool) []*Elem {
	rtn := make([]*Elem, 0, len(frame.Elems))
	num := 0
	for _, el := range frame.Elems {
		if el.IsHidden(frame.Show) {
			continue
		}
		if plate || el.IsLineElem() {
			sign := 0
			for i, en := range el.Enod {
				if en.Coord[axis]-coord > 0 {
					sign++
				} else {
					sign--
				}
				if i+1 != abs(sign) {
					rtn = append(rtn, el)
					num++
					break
				}
			}
		}
	}
	return rtn[:num]
}

func (frame *Frame) FenceLine(x1, y1, x2, y2 float64) []*Elem {
	rtn := make([]*Elem, len(frame.Elems))
	k := 0
	for _, el := range frame.Elems {
		if el.IsHidden(frame.Show) {
			continue
		}
		add := false
		sign := 0
		for i, en := range el.Enod {
			if DotLine(x1, y1, x2, y2, en.Pcoord[0], en.Pcoord[1]) > 0 {
				sign++
			} else {
				sign--
			}
			if i+1 != abs(sign) {
				add = true
				break
			}
		}
		if add {
			if el.IsLineElem() {
				if DotLine(el.Enod[0].Pcoord[0], el.Enod[0].Pcoord[1], el.Enod[1].Pcoord[0], el.Enod[1].Pcoord[1], x1, y1)*DotLine(el.Enod[0].Pcoord[0], el.Enod[0].Pcoord[1], el.Enod[1].Pcoord[0], el.Enod[1].Pcoord[1], x2, y2) < 0 {
					rtn[k] = el
					k++
				}
			} else {
				addx := false
				sign := 0
				for i, j := range el.Enod {
					if math.Max(x1, x2) < j.Pcoord[0] {
						sign++
					} else if j.Pcoord[0] < math.Min(x1, x2) {
						sign--
					}
					if i+1 != abs(sign) {
						addx = true
						break
					}
				}
				if addx {
					addy := false
					sign := 0
					for i, j := range el.Enod {
						if math.Max(y1, y2) < j.Pcoord[1] {
							sign++
						} else if j.Pcoord[1] < math.Min(y1, y2) {
							sign--
						}
						if i+1 != abs(sign) {
							addy = true
							break
						}
					}
					if addy {
						rtn[k] = el
						k++
					}
				}
			}
		}
	}
	return rtn[:k]
}

func (frame *Frame) Cutter(axis int, coord float64, eps float64) error {
	for _, el := range frame.Fence(axis, coord, false) {
		_, _, err := el.DivideAtAxis(axis, coord, eps)
		if err != nil {
			return err
		}
	}
	return nil
}

func (frame *Frame) LineConnected(n *Node) []*Node {
	var add bool
	els := frame.SearchElem(n)
	ns := make([]*Node, 0)
	for _, el := range els {
		if el.IsLineElem() {
			for _, en := range el.Enod {
				if en == n {
					continue
				}
				add = true
				for _, n := range ns {
					if en == n {
						add = false
						break
					}
				}
				if add {
					ns = append(ns, en)
				}
			}
		}
	}
	return ns
}

func (frame *Frame) Connected(n *Node) []*Node {
	i := 0
	els := frame.SearchElem(n)
	ns := frame.ElemToNode(els...)
	rtn := make([]*Node, len(ns)-1)
	for _, val := range ns {
		if val != n {
			rtn[i] = val
			i++
		}
	}
	return rtn
}

// TODO: check if this func works as intended
func (frame *Frame) SearchBraceSect(f *Fig, t int) *Sect {
	for _, sec := range frame.Sects {
		if sec.Num <= 900 {
			continue
		}
		if (sec.Type == t) && (sec.Figs[0].Prop == f.Prop) &&
			(sec.Figs[0].Value["AREA"] == f.Value["AREA"]) &&
			(sec.Figs[0].Value["IXX"] == 0.0) && (sec.Figs[0].Value["IYY"] == 0.0) {
			return sec
		}
	}
	return nil
}

func (frame *Frame) DeleteNode(num int) {
	var node *Node
	if n, ok := frame.Nodes[num]; ok {
		node = n
	} else {
		return
	}
	for _, el := range frame.Elems {
		for _, en := range el.Enod {
			if en == node {
				frame.DeleteElem(el.Num)
			}
		}
	}
	arcs := make([]*Arc, len(frame.Arcs))
	i := 0
del_arc:
	for _, a := range frame.Arcs {
		for _, en := range a.Enod {
			if en == node {
				continue del_arc
			}
		}
		arcs[i] = a
		i++
	}
	frame.Arcs = arcs[:i]
	delete(frame.Nodes, num)
	if frame.Maxnnum == num {
		frame.Maxnnum--
	}
}

func (frame *Frame) DeleteElem(num int) {
	if el, ok := frame.Elems[num]; ok {
		if el.Chain != nil {
			el.Chain.Delete(el)
		}
	}
	delete(frame.Elems, num)
	if frame.Maxenum == num {
		frame.Maxenum--
	}
}

func (frame *Frame) DeleteSect(num int) {
	delete(frame.Sects, num)
	if frame.Maxsnum == num {
		frame.Maxsnum--
	}
}

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
	for _, a := range frame.Arcs {
		for _, en := range a.Enod {
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
		if el.HasSameNode() {
			rtn = append(rtn, el)
		}
	}
	return rtn
}

func (frame *Frame) NodeDuplication(eps float64) map[*Node]*Node {
	return NodeDuplication(frame.Nodes, eps)
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
		frame.DeleteNode(k.Num)
	}
}

func (frame *Frame) MergeNode(ns []*Node) {
	c := make([]float64, 3)
	num := 0
	for _, n := range ns {
		if n == nil {
			continue
		}
		for i := 0; i < 3; i++ {
			c[i] += n.Coord[i]
		}
		num++
	}
	if num > 0 {
		for i := 0; i < 3; i++ {
			c[i] /= float64(num)
		}
		var del bool
		delmap := make(map[*Node]*Node)
		var n0 *Node
		for _, n := range ns {
			if n == nil {
				continue
			}
			if del {
				delmap[n] = n0
			} else {
				n.Coord = c
				n0 = n
				del = true
			}
		}
		frame.ReplaceNode(delmap)
	}
}

func (frame *Frame) ElemDuplication(ignoresect []int) map[*Elem]*Elem {
	dups := make(map[*Elem]*Elem, 0)
	elems := make([]*Elem, len(frame.Elems))
	enum := 0
	var add bool
	for _, el := range frame.Elems {
		if el.Etype == WBRACE || el.Etype == SBRACE {
			continue
		}
		add = true
		if ignoresect != nil {
			for _, sec := range ignoresect {
				if el.Sect.Num == sec {
					add = false
					break
				}
			}
		}
		if add {
			elems[enum] = el
			enum++
		}
	}
	elems = elems[:enum]
	sort.Sort(ElemByNum{elems})
	sort.Stable(ElemBySumEnod{elems})
	for i, el := range elems {
		if _, ok := dups[el]; ok {
			continue
		}
		sum1 := 0
		for _, en := range el.Enod {
			sum1 += en.Num
		}
		for _, el2 := range elems[i+1:] {
			sum2 := 0
			for _, en := range el2.Enod {
				sum2 += en.Num
			}
			if sum1 == sum2 {
				if CompareNodes(el.Enod, el2.Enod) {
					dups[el2] = el
				}
			} else {
				break
			}
		}
	}
	return dups
}

func (frame *Frame) BandWidth() int {
	rtn := 0
	for _, el := range frame.Elems {
		if el.IsLineElem() {
			val := abs(el.Enod[1].Num - el.Enod[0].Num)
			if val > rtn {
				rtn = val
			}
		}
	}
	return rtn
}

func (frame *Frame) NodeSort(d int) (int, error) {
	nnum := 0
	nodes := make([]*Node, len(frame.Nodes))
	for _, n := range frame.Nodes {
		nodes[nnum] = n
		nnum++
	}
	sort.Sort(NodeByNum{nodes})
	switch d {
	case 0:
		sort.Stable(NodeByXCoord{nodes})
	case 1:
		sort.Stable(NodeByYCoord{nodes})
	case 2:
		sort.Stable(NodeByZCoord{nodes})
	default:
		return 0, errors.New("NodeSort: Unknown Direction")
	}
	newnodes := make(map[int]*Node)
	num := 101
	for _, n := range nodes {
		newnodes[num] = n
		n.Num = num
		num++
	}
	frame.Nodes = newnodes
	return frame.BandWidth(), nil
}

func (frame *Frame) Suspicious() ([]*Node, []*Elem, error) {
	var otp bytes.Buffer
	ns := make([]*Node, 0)
	els := make([]*Elem, 0)
susnode:
	for _, n := range frame.Nodes {
		es := frame.SearchElem(n)
		err1 := true
		err2 := true
		for _, el := range es {
			if el.IsLineElem() && el.Etype <= BRACE {
				err1 = false
				if el.IsRigid(n.Num) {
					err2 = false
				}
			}
			if !err1 && !err2 {
				continue susnode
			}
		}
		if err1 {
			ns = append(ns, n)
			otp.WriteString(fmt.Sprintf("no line elem: %d\n", n.Num))
		} else if err2 {
			ns = append(ns, n)
			otp.WriteString(fmt.Sprintf("all pin: %d\n", n.Num))
		}
	}
	for _, el := range frame.Elems {
		b, err := el.IsValidElem()
		if !b {
			els = append(els, el)
			otp.WriteString(fmt.Sprintf("%s\n", err.Error()))
		}
	}
	if len(ns) == 0 && len(els) == 0 {
		return ns, els, nil
	} else {
		return ns, els, errors.New(otp.String())
	}
}

func (frame *Frame) Cat(e1, e2 *Elem, n *Node, del bool) error {
	if !e1.IsLineElem() || !e2.IsLineElem() {
		return NotLineElem("Cat")
	}
	var ind1, ind2 int
	for i, en := range e1.Enod {
		if en == n {
			ind1 = i
			break
		}
	}
	for i, en := range e2.Enod {
		if en == n {
			ind2 = 1 - i
			break
		}
	}
	e1.Enod[ind1] = e2.Enod[ind2]
	for j := 0; j < 6; j++ {
		e1.Bonds[6*ind1+j] = e2.Bonds[6*ind1+j]
	}
	if del {
		frame.DeleteNode(n.Num)
	}
	frame.DeleteElem(e2.Num)
	return nil
}

func CommonEnod(els ...*Elem) ([]*Node, error) {
	if len(els) < 2 {
		return nil, errors.New("too few elems")
	}
	num := len(els[0].Enod)
	rtn := make([]*Node, num)
	for i := 0; i < num; i++ {
		rtn[i] = els[0].Enod[i]
	}
	for _, el := range els[1:] {
		ok := make([]bool, len(rtn))
		for _, en := range el.Enod {
			for i, en2 := range rtn {
				if en == en2 {
					ok[i] = true
				}
			}
		}
		size := 0
		tmp := make([]*Node, len(rtn))
		for i := 0; i < len(rtn); i++ {
			if ok[i] {
				tmp[size] = rtn[i]
				size++
			}
		}
		rtn = tmp[:size]
	}
	return rtn, nil
}

func (frame *Frame) JoinLineElem(e1, e2 *Elem, parallel, others bool) error {
	if !e1.IsLineElem() || !e2.IsLineElem() {
		return NotLineElem("JoinLineElem")
	}
	if parallel && !IsParallel(e1.Direction(true), e2.Direction(true), 1e-4) {
		return NotParallel("JoinLineElem")
	}
	del := true
	for i, en1 := range e1.Enod {
		for _, en2 := range e2.Enod {
			if en1 == en2 {
				for _, el := range frame.SearchElem(e1.Enod[i]) {
					if el.Etype == WBRACE || el.Etype == SBRACE {
						continue
					}
					if el == e1 || el == e2 {
						continue
					}
					if others {
						return errors.New(fmt.Sprintf("JoinLineElem: NODE %d has more than 2 elements", e1.Enod[i].Num))
					} else {
						del = false
						break
					}
				}
				return e1.Frame.Cat(e1, e2, e1.Enod[i], del)
			}
		}
	}
	return errors.New("JoinLineElem: No Common Enod")
}

func (frame *Frame) JoinPlateElem(e1, e2 *Elem) error {
	if e1.IsLineElem() || e2.IsLineElem() {
		return NotPlateElem("JoinPlateElem")
	}
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
						case 1, 1 - e2.Enods:
							for k := 0; k < 2; k++ {
								num1 := i + k
								if num1 > e1.Enods {
									num1 -= e1.Enods
								}
								num2 := j - k - 1
								if num2 < 0 {
									num2 += e2.Enods
								}
								e1.Enod[num1] = e2.Enod[num2]
							}
						case -1, e2.Enods - 1:
							for k := 0; k < 2; k++ {
								num1 := i + k
								if num1 >= e1.Enods {
									num1 -= e1.Enods
								}
								num2 := j + k + 1
								if num2 >= e2.Enods {
									num2 -= e2.Enods
								}
								e1.Enod[num1] = e2.Enod[num2]
							}
						}
						frame.DeleteElem(e2.Num)
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
						case 1, 1 - e2.Enods:
							for k := 0; k < 2; k++ {
								num1 := i - k
								if num1 < 0 {
									num1 += e1.Enods
								}
								num2 := j - k - 1
								if num2 < 0 {
									num2 += e2.Enods
								}
								e1.Enod[num1] = e2.Enod[num2]
							}
						case -1, e2.Enods - 1:
							for k := 0; k < 2; k++ {
								num1 := i - k
								if num1 < 0 {
									num1 += e1.Enods
								}
								num2 := j + k + 1
								if num2 > e2.Enods {
									num2 -= e2.Enods
								}
								e1.Enod[num1] = e2.Enod[num2]
							}
						}
						frame.DeleteElem(e2.Num)
						return nil
					}
				}
				return errors.New(fmt.Sprintf("JoinPlateElem: Only 1 Common Enod %d", en1.Num))
			}
		}
	}
	return errors.New("JoinPlateElem: No Common Enod")
}

func (frame *Frame) CatByNode(n *Node, parallel bool) error {
	els := frame.SearchElem(n)
	var d []float64
	var num int
	cat := make([]*Elem, 2)
	for _, el := range els {
		if el != nil {
			num++
			if num > 2 {
				return errors.New(fmt.Sprintf("CatByNode: NODE %d has more than 2 elements", n.Num))
			}
			if !el.IsLineElem() {
				return errors.New(fmt.Sprintf("CatByNode: NODE %d has WALL/SLAB", n.Num))
			}
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
	frame.Cat(cat[0], cat[1], n, true)
	return nil
}

func (frame *Frame) Intersect(e1, e2 *Elem, cross bool, sign1, sign2 int, del1, del2 bool, eps float64) ([]*Node, []*Elem, error) {
	if e1 == e2 {
		return nil, nil, errors.New("Intersect: the same element")
	}
	if !e1.IsLineElem() || !e2.IsLineElem() {
		return nil, nil, NotLineElem("Intersect")
	}
	k1, k2, d, err := DistLineLine(e1.Enod[0].Coord, e1.Direction(false), e2.Enod[0].Coord, e2.Direction(false))
	if err != nil {
		return nil, nil, err
	}
	if d > eps {
		return nil, nil, errors.New(fmt.Sprintf("Intersect: Distance= %.5f", d))
	}
	if !cross || ((-eps <= k1 && k1 <= 1.0+eps) && (-eps <= k2 && k2 <= 1.0+eps)) {
		var ns []*Node
		var els []*Elem
		var tmpels []*Elem
		var err error
		d1 := e1.Direction(false)
		n, _ := frame.CoordNode(e1.Enod[0].Coord[0]+k1*d1[0], e1.Enod[0].Coord[1]+k1*d1[1], e1.Enod[0].Coord[2]+k1*d1[2], eps)
		switch {
		default:
		case k1 < -eps:
			ns, els, err = e1.DivideAtNode(n, 0, del1)
		case -eps <= k1 && k1 <= 1.0+eps:
			ns, els, err = e1.DivideAtNode(n, 1*sign1, del1)
		case 1.0+eps < k1:
			ns, els, err = e1.DivideAtNode(n, 2, del1)
		}
		if err != nil {
			switch err.(type) {
			case ElemDivisionError:
				break
			default:
				return nil, nil, err
			}
		}
		switch {
		default:
		case k2 < -eps:
			ns, tmpels, err = e2.DivideAtNode(n, 0, del2)
		case -eps <= k2 && k2 <= 1.0+eps:
			ns, tmpels, err = e2.DivideAtNode(n, 1*sign2, del2)
		case 1.0+eps < k2:
			ns, tmpels, err = e2.DivideAtNode(n, 2, del2)
		}
		if err != nil {
			switch err.(type) {
			case ElemDivisionError:
				els = append(els, tmpels...)
				return ns, els, nil
			default:
				return nil, nil, err
			}
		}
		els = append(els, tmpels...)
		return ns, els, nil
	} else {
		return nil, nil, errors.New(fmt.Sprintf("Intersect: Not Cross k1= %.5f, k2= %.5f", k1, k2))
	}
}

func (frame *Frame) CutByElem(cutter, cuttee *Elem, cross bool, sign int, del bool, eps float64) ([]*Node, []*Elem, error) {
	if !cutter.IsLineElem() || !cuttee.IsLineElem() {
		return nil, nil, NotLineElem("CutByElem")
	}
	k1, k2, d, err := DistLineLine(cutter.Enod[0].Coord, cutter.Direction(false), cuttee.Enod[0].Coord, cuttee.Direction(false))
	if err != nil {
		return nil, nil, err
	}
	if d > eps {
		return nil, nil, errors.New(fmt.Sprintf("CutByElem: Distance= %.5f", d))
	}
	if !cross || ((-eps <= k1 && k1 <= 1.0+eps) && (-eps <= k2 && k2 <= 1.0+eps)) {
		var ns []*Node
		var els []*Elem
		var err error
		d1 := cutter.Direction(false)
		n, _ := frame.CoordNode(cutter.Enod[0].Coord[0]+k1*d1[0], cutter.Enod[0].Coord[1]+k1*d1[1], cutter.Enod[0].Coord[2]+k1*d1[2], eps)
		switch {
		default:
		case k2 < -eps:
			ns, els, err = cuttee.DivideAtNode(n, 0, del)
		case -eps <= k2 && k2 <= 1.0+eps:
			ns, els, err = cuttee.DivideAtNode(n, 1*sign, del)
		case 1.0+eps < k2:
			ns, els, err = cuttee.DivideAtNode(n, 2, del)
		}
		if err != nil {
			switch err.(type) {
			case ElemDivisionError:
				return ns, els, nil
			default:
				return nil, nil, err
			}
		}
		return ns, els, nil
	} else {
		return nil, nil, errors.New(fmt.Sprintf("CutByElem: Not Cross k1= %.5f, k2= %.5f", k1, k2))
	}
}

func (frame *Frame) IntersectionPoint(cutter, cuttee *Elem, cross bool, eps float64) (*Node, error) {
	if !cutter.IsLineElem() || !cuttee.IsLineElem() {
		return nil, NotLineElem("IntersectionPoint")
	}
	k1, k2, d, err := DistLineLine(cutter.Enod[0].Coord, cutter.Direction(false), cuttee.Enod[0].Coord, cuttee.Direction(false))
	if err != nil {
		return nil, err
	}
	if d > eps {
		return nil, errors.New(fmt.Sprintf("IntersectionPoint: Distance= %.5f", d))
	}
	if !cross || (-eps <= k2 && k2 <= 1.0+eps) {
		d1 := cutter.Direction(false)
		n, _ := frame.CoordNode(cutter.Enod[0].Coord[0]+k1*d1[0], cutter.Enod[0].Coord[1]+k1*d1[1], cutter.Enod[0].Coord[2]+k1*d1[2], eps)
		return n, nil
	} else {
		return nil, errors.New(fmt.Sprintf("IntersectionPoint: Not Cross k1= %.5f, k2= %.5f", k1, k2))
	}
}

func (frame *Frame) Trim(e1, e2 *Elem, sign int, eps float64) ([]*Node, []*Elem, error) {
	return frame.CutByElem(e1, e2, true, sign, true, eps)
}

func (frame *Frame) Extend(e1, e2 *Elem, eps float64) ([]*Node, []*Elem, error) {
	return frame.CutByElem(e1, e2, false, 1, true, eps)
}

func (frame *Frame) Fillet(e1, e2 *Elem, sign1, sign2 int, eps float64) ([]*Node, []*Elem, error) {
	return frame.Intersect(e1, e2, false, sign1, sign2, true, true, eps)
}

func (frame *Frame) IntersectAll(elems []*Elem, eps float64) error {
	l := len(elems)
	if elems == nil || l <= 1 {
		return nil
	}
	checked := make([]*Elem, 1)
	sort.Sort(ElemByNum{elems})
	ind := 0
	for {
		if elems[ind].IsLineElem() {
			_, els, err := elems[ind].DivideAtOns(eps)
			if err != nil {
				return err
			}
			checked = els
			frame.Lapch <- ind + 1
			break
		}
		ind++
		if ind >= l-1 {
			return errors.New("Intersectall: no line elem")
		}
	}
	for i, el := range elems[ind+1:] {
		if !el.IsLineElem() {
			continue
		}
		for _, ce := range checked {
			_, els, err := frame.CutByElem(el, ce, true, 1, false, eps)
			if err != nil {
				continue
			}
			if len(els) >= 2 {
				checked = append(checked, els[1])
			}
		}
		_, els, err := el.DivideAtOns(eps)
		if err != nil {
			continue
		}
		checked = append(checked, els...)
		frame.Lapch <- ind + 2 + i
	}
	return nil
}

func (frame *Frame) IsUpside() bool {
	for _, el := range frame.Elems {
		if !IsUpside(el.Enod) {
			return false
		}
	}
	return true
}

func (frame *Frame) Upside() {
	for _, el := range frame.Elems {
		el.Upside()
	}
}

func (frame *Frame) SetBoundary(num int, eps float64) error {
	nodes := make([]*Node, 0)
	nnum := 0
	for _, n := range frame.Nodes {
		nodes = append(nodes, n)
		nnum++
	}
	nodes = nodes[:nnum]
	sort.Sort(NodeByZCoord{nodes})
	dhs := make([]float64, num-1)
	bounds := make([]float64, num+1)
	var dh float64
	for i := 0; i < nnum-1; i++ {
		dh = nodes[i+1].Coord[2] - nodes[i].Coord[2]
		for j := num - 2; j > 0; j-- {
			if dh > dhs[j] {
				for k := 0; k < j; k++ {
					dhs[k] = dhs[k+1]
					bounds[k+1] = bounds[k+2]
				}
				dhs[j] = dh
				bounds[j+1] = nodes[i+1].Coord[2]
				break
			}
		}
	}
	bounds[0] = nodes[0].Coord[2] - eps
	bounds[num] = nodes[nnum-1].Coord[2] + eps
	for i := 1; i < num; i++ {
		bounds[i] += eps
	}
	sort.Float64s(bounds)
	frame.Ai.Nfloor = num
	frame.Ai.Boundary = make([]float64, num+1)
	for i := 0; i < num+1; i++ {
		frame.Ai.Boundary[i] = bounds[i]
	}
	return nil
}

func (frame *Frame) ExtractArclm(fn string) error {
	cmqs := make(map[int][]float64)
	for _, el := range frame.Elems {
		if el.IsLineElem() {
			cmqs[el.Num] = make([]float64, 12)
			for i := 0; i < 12; i++ {
				cmqs[el.Num][i] = el.Cmq[i]
			}
		}
	}
	defer func() {
		for _, n := range frame.Nodes {
			for i := 0; i < 3; i++ {
				n.Weight[i] = 0.0
			}
		}
		for _, el := range frame.Elems {
			if el.IsLineElem() && el.Etype <= TRUSS {
				for i := 0; i < 12; i++ {
					el.Cmq[i] = cmqs[el.Num][i]
				}
			}
		}
	}()
	err := frame.WeightDistribution(fn)
	if err != nil {
		return err
	}
	for _, el := range SortedElem(frame.Elems, func(e *Elem) float64 { return float64(e.Num) }) {
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
	sects := make([]*Sect, 0)
	snum := 0
	for _, sect := range frame.Sects {
		if sect.HasArea(0) {
			sects = append(sects, sect)
			snum++
		}
	}
	sects = sects[:snum]
	sort.Sort(SectByNum{sects})
	bonds := make([]*Bond, 0)
	bnum := 0
	for _, b := range frame.Bonds {
		bonds = append(bonds, b)
		bnum++
	}
	if bnum == 0 {
		bonds = []*Bond{Pin}
		bnum = 1
	} else {
		bonds = bonds[:bnum]
		sort.Sort(BondByNum{bonds})
	}
	nodes := make([]*Node, 0)
	nnum := 0
	for _, n := range frame.Nodes {
		nodes = append(nodes, n)
		nnum++
	}
	nodes = nodes[:nnum]
	sort.Sort(NodeByNum{nodes})
	elems := make([]*Elem, 0)
	enum := 0
	for _, el := range frame.Elems {
		if el.IsLineElem() {
			elems = append(elems, el)
			enum++
		}
	}
	elems = elems[:enum]
	sort.Sort(ElemByNum{elems})
	set := snum
	for _, el := range elems {
		if el.Sect.Type == 0 {
			el.Sect.Type = el.Etype
			set--
			if set <= 0 {
				break
			}
		}
	}
	for _, p := range []string{"L", "X", "Y"} {
		af := arclm.NewFrame()
		af.Sects = make([]*arclm.Sect, snum+bnum)
		arclmsects := make(map[int]int)
		for i, sec := range sects {
			yield := make([]float64, 12)
			for j := 0; j < 12; j++ {
				yield[j] = sec.Yield[j]
			}
			var E float64
			if p == "L" {
				E = sec.Figs[0].Prop.EL
			} else {
				E = sec.Figs[0].Prop.ES
			}
			af.Sects[i] = &arclm.Sect{
				Num:      sec.Num,
				E:        E,
				Poi:      sec.Figs[0].Prop.Poi,
				Value:    sec.ArclmValue(),
				Yield:    yield,
				Type:     sec.Type,
				Exp:      sec.Exp,
				Exq:      sec.Exq,
				Original: sec.Original,
			}
			arclmsects[sec.Num] = i
		}
		for i, b := range bonds {
			yield := make([]float64, 12)
			af.Sects[snum+i] = &arclm.Sect{
				Num:      b.Num,
				E:        0.0,
				Poi:      0.0,
				Value:    []float64{0.0, b.Stiffness[0], b.Stiffness[1], 0.0},
				Yield:    yield,
				Type:     -1,
				Exp:      0.0,
				Exq:      0.0,
				Original: b.Num,
			}
			arclmsects[b.Num] = snum + i
		}
		af.Nodes = make([]*arclm.Node, nnum)
		arclmnodes := make(map[int]int)
		for i, n := range nodes {
			an := arclm.NewNode()
			an.Num = n.Num
			for j := 0; j < 3; j++ {
				an.Coord[j] = n.Coord[j]
			}
			var disp []float64
			var reaction []float64
			if d, ok := n.Disp[p]; ok {
				disp = d
			}
			if r, ok := n.Reaction[p]; ok {
				reaction = r
			}
			for j := 0; j < 6; j++ {
				an.Conf[j] = n.Conf[j]
				an.Force[j] = n.Load[j]
				if disp != nil {
					an.Disp[j] = n.Disp[p][j]
				}
				if n.Conf[j] {
					if reaction != nil {
						an.Reaction[j] = n.Reaction[p][j]
					}
				}
			}
			switch p {
			case "L":
				if !an.Conf[2] {
					an.Force[2] -= n.Weight[1]
				}
			case "X":
				if !an.Conf[0] {
					an.Force[0] += n.Factor[0] * n.Weight[2]
				}
			case "Y":
				if !an.Conf[1] {
					an.Force[1] += n.Factor[1] * n.Weight[2]
				}
			}
			if n.Pile != nil {
				switch p {
				case "X":
					an.Force[4] += n.Pile.Moment
				case "Y":
					an.Force[3] -= n.Pile.Moment
				}
			}
			n.Force[p] = make([]float64, 6)
			for j := 0; j < 6; j++ {
				n.Force[p][j] = an.Force[j]
			}
			an.Index = i
			an.Mass = n.Weight[2] / 9.80665
			af.Nodes[i] = an
			arclmnodes[n.Num] = i
		}
		for _, el := range frame.Elems {
			for i := 0; i < 3; i++ {
				var val float64
				switch p {
				case "L":
					val = el.ConvertPerpendicularLoad(1, i)
				case "X", "Y":
					val = el.ConvertPerpendicularLoad(2, i)
				}
				if val != 0.0 {
					a := el.Amount()
					for _, n := range el.Enod {
						an := af.Nodes[arclmnodes[n.Num]]
						an.Force[i] += val * a / float64(el.Enods)
					}
				}
			}
		}
		af.Elems = make([]*arclm.Elem, enum)
		for i, el := range elems {
			ae := arclm.NewElem()
			ae.Num = el.Num
			ae.Sect = af.Sects[arclmsects[el.Sect.Num]]
			for j := 0; j < 2; j++ {
				ae.Enod[j] = af.Nodes[arclmnodes[el.Enod[j].Num]]
			}
			ae.Cang = el.Cang
			var stress map[int][]float64
			if s, ok := el.Stress[p]; ok {
				stress = s
			}
			if p == "L" {
				for k, en := range ae.Enod {
					if !en.Conf[2] {
						en.Force[2] += el.Cmq[2+6*k]
					}
				}
				for k, en := range el.Enod {
					if !en.Conf[2] {
						en.Force[p][2] += el.Cmq[2+6*k]
					}
				}
			}
			for j := 0; j < 12; j++ {
				if el.Bonds[j] == nil {
					ae.Bonds[j] = arclm.Rigid
				} else {
					ae.Bonds[j] = af.Sects[arclmsects[el.Bonds[j].Num]]
				}
				if p == "L" {
					ae.Cmq[j] = el.Cmq[j]
				}
				if stress != nil {
					if j < 6 {
						ae.Stress[j] = stress[el.Enod[0].Num][j]
					} else {
						ae.Stress[j] = stress[el.Enod[1].Num][j-6]
					}
				} else {
					if p == "L" {
						ae.Stress[j] = el.Cmq[j]
					}
				}
			}
			af.Elems[i] = ae
		}
		frame.Arclms[p] = af
	}
	return nil
}

func (frame *Frame) WeightDistribution(fn string) error {
	var otp bytes.Buffer
	var ekeys []int
	nodes := make([]*Node, len(frame.Nodes))
	nnum := 0
	for _, n := range frame.Nodes {
		nodes[nnum] = n
		nnum++
	}
	sort.Sort(NodeByNum{nodes})
	amount := make(map[int]float64)
	for _, el := range frame.Elems {
		err := el.Distribute()
		if err != nil {
			return err
		}
		if el.Etype != WBRACE || el.Etype != SBRACE {
			amount[el.Sect.Num] += el.Amount()
		}
	}
	aistr, err := frame.AiDistribution()
	if err != nil {
		return err
	}
	wstr, err := frame.WindPressure()
	if err != nil {
		return err
	}
	total := make([]float64, 3)
	otp.WriteString("3.2 : 節点重量\n\n")
	otp.WriteString("節点ごとに重量を集計した結果を記す。\n")
	otp.WriteString("柱，壁は階高の中央で上下に分配するものとする。\n\n")
	otp.WriteString(fmt.Sprintf(" 節点番号          積載荷重別の重量 [%s]\n\n", frame.Show.UnitName[0]))
	otp.WriteString("                 床用     柱梁用     地震用\n")
	for _, n := range nodes {
		otp.WriteString(n.WgtString(frame.Show.Unit[0]))
		for i := 0; i < 3; i++ {
			total[i] += n.Weight[i]
		}
	}
	otp.WriteString(fmt.Sprintf("\n       計  %10.3f %10.3f %10.3f\n\n", frame.Show.Unit[0]*total[0], frame.Show.Unit[0]*total[1], frame.Show.Unit[0]*total[2]))
	otp.WriteString("各断面の部材総量（参考資料）\n\n")
	otp.WriteString(" 断面番号 長さ,面積 重量(柱梁用)\n             [m,m2]     [ton]\n")
	for k := range amount {
		ekeys = append(ekeys, k)
	}
	sort.Ints(ekeys)
	for _, k := range ekeys {
		otp.WriteString(fmt.Sprintf("%9d %9.3f %9.3f 【%s】\n", k, amount[k], amount[k]*frame.Sects[k].Weight()[1], frame.Sects[k].Name))
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
	otp.WriteString(aistr)
	otp.WriteString("\n")
	otp.WriteString(wstr)
	if fn == "" {
		fn = filepath.Join(frame.Home, DefaultWgt)
	}
	w, err := os.Create(fn)
	defer w.Close()
	if err != nil {
		return err
	}
	otp = AddCR(otp)
	otp.WriteTo(w)
	return nil
}

func (frame *Frame) AiDistribution() (string, error) {
	if frame.Ai.Nfloor == 0 && len(frame.Ai.Boundary) == 0 {
		return "", fmt.Errorf("level isn't set up")
	}
	size := frame.Ai.Nfloor
	frame.Ai.Wi = make([]float64, size)
	frame.Ai.Level = make([]float64, size)
	total := make([]float64, 3)
	nnum := make([]int, size)
	maxheight := MINCOORD
	for _, n := range frame.Nodes {
		for i := 0; i < 3; i++ {
			total[i] += n.Weight[i]
		}
		height := n.Coord[2]
		if height < frame.Ai.Boundary[0] {
			continue
		}
		for i := 0; i < size; i++ {
			if height < frame.Ai.Boundary[i+1] {
				frame.Ai.Wi[i] += n.Weight[2]
				frame.Ai.Level[i] += n.Coord[2]
				nnum[i]++
				break
			}
		}
		// if height < frame.Ai.Boundary[1] {
		//     weight[0] += n.Weight[2]
		//     level[0] += n.Coord[2]
		//     nnum[0]++
		// } else if height >= frame.Ai.Boundary[size-3] {
		//     weight[size-1] += n.Weight[2]
		//     level[size-1] += n.Coord[2]
		//     nnum[size-1]++
		// } else {
		//     for i:=2; i<size-2; i++ {
		//         if height < frame.Ai.Boundary[i] {
		//             weight[i] += n.Weight[2]
		//             level[i] += n.Coord[2]
		//             nnum[i]++
		//             break
		//         }
		//     }
		// }
	}
	frame.Ai.W = make([]float64, size)
	for i := 0; i < size; i++ {
		frame.Ai.Level[i] /= float64(nnum[i])
		if frame.Ai.Level[i] > maxheight {
			maxheight = frame.Ai.Level[i]
		}
		for j := size - 1; j >= i; j-- {
			frame.Ai.W[i] += frame.Ai.Wi[j]
		}
	}
	frame.Ai.Ai = make([]float64, size-1)
	frame.Ai.T = maxheight * frame.Ai.Tfact
	if frame.Ai.T < frame.Ai.Gperiod {
		frame.Ai.Rt = 1.0
	} else if frame.Ai.T < 2.0*frame.Ai.Gperiod {
		frame.Ai.Rt = 1.0 - 0.2*math.Pow((frame.Ai.T/frame.Ai.Gperiod-1.0), 2.0)
	} else {
		frame.Ai.Rt = 1.6 * frame.Ai.Gperiod / frame.Ai.T
	}
	tt := 2.0 * frame.Ai.T / (1.0 + 3.0*frame.Ai.T)
	for i := 0; i < size-1; i++ {
		alpha := frame.Ai.W[i+1] / frame.Ai.W[1]
		frame.Ai.Ai[i] = 1.0 + (1.0/math.Sqrt(alpha)-alpha)*tt
	}
	facts := make([][]float64, 2)
	for d := 0; d < 2; d++ {
		frame.Ai.Ci[d] = make([]float64, size)
		frame.Ai.Qi[d] = make([]float64, size)
		frame.Ai.Hi[d] = make([]float64, size)
		facts[d] = make([]float64, size)
		for i := 0; i < size; i++ {
			if i == 0 {
				frame.Ai.Ci[d][0] = 0.5 * frame.Ai.Locate * frame.Ai.Rt * frame.Ai.Base[d]
				frame.Ai.Qi[d][0] = frame.Ai.Ci[d][0] * frame.Ai.W[0]
				facts[d][0] = frame.Ai.Ci[d][0]
			} else {
				frame.Ai.Ci[d][i] = frame.Ai.Locate * frame.Ai.Rt * frame.Ai.Ai[i-1] * frame.Ai.Base[d]
				frame.Ai.Qi[d][i] = frame.Ai.Ci[d][i] * frame.Ai.W[i]
				frame.Ai.Hi[d][i-1] = frame.Ai.Qi[d][i-1] - frame.Ai.Qi[d][i]
				if i > 1 {
					facts[d][i-1] = frame.Ai.Hi[d][i-1] / frame.Ai.Wi[i-1]
				}
			}
		}
		if size > 1 {
			frame.Ai.Hi[d][size-1] = frame.Ai.Qi[d][size-1]
			facts[d][size-1] = frame.Ai.Hi[d][size-1] / frame.Ai.Wi[size-1]
		}
	}
	for _, n := range frame.Nodes {
		height := n.Coord[2]
		if height < frame.Ai.Boundary[0] {
			n.Factor = []float64{facts[0][0], facts[1][0]}
			continue
		} else if height >= frame.Ai.Boundary[size] {
			n.Factor = []float64{facts[0][size-1], facts[1][size-1]}
			continue
		}
		for i := 0; i < size; i++ {
			if height < frame.Ai.Boundary[i+1] {
				n.Factor = []float64{facts[0][i], facts[1][i]}
				break
			}
		}
	}
	var rtn bytes.Buffer
	rtn.WriteString("3.3 : Ai分布型地震荷重\n\n")
	rtn.WriteString("水平荷重は建築基準法施行令第88条および建設省告示1793号に従い、Ａi分布型の地震力とする。\n\n")
	rtn.WriteString(fmt.Sprintf("階数       　　　    n =%d\n", size))
	rtn.WriteString(fmt.Sprintf("高さ         　　　  H =%.3f\n", maxheight))
	rtn.WriteString(fmt.Sprintf("１次固有周期         T1=%5.3fH=%5.3f\n", frame.Ai.Tfact, frame.Ai.T))
	rtn.WriteString(fmt.Sprintf("地盤周期             Tc=%5.3f\n", frame.Ai.Gperiod))
	rtn.WriteString(fmt.Sprintf("振動特性係数         Rt=%5.3f\n", frame.Ai.Rt))
	rtn.WriteString(fmt.Sprintf("地域係数             Z =%5.3f\n", frame.Ai.Locate))
	rtn.WriteString(fmt.Sprintf("標準層せん断力係数   Cox=%5.3f, Coy=%5.3f\n", frame.Ai.Base[0], frame.Ai.Base[1]))
	rtn.WriteString(fmt.Sprintf("基礎部分の震度       Cfx=%5.3f, Cfy=%5.3f\n\n", facts[0][0], facts[1][0]))
	rtn.WriteString(fmt.Sprintf("床用積載荷重による総荷重    = %10.3f\n", frame.Show.Unit[0]*total[0]))
	rtn.WriteString(fmt.Sprintf("柱梁用積載荷重による総荷重  = %10.3f\n", frame.Show.Unit[0]*total[1]))
	rtn.WriteString(fmt.Sprintf("地震用積載荷重による総荷重  = %10.3f\n\n", frame.Show.Unit[0]*total[2]))
	rtn.WriteString("各階平均高さ      :")
	for i := 0; i < size; i++ {
		rtn.WriteString(fmt.Sprintf(" %10.3f", frame.Ai.Level[i]))
	}
	rtn.WriteString("\n各階重量       wi :")
	for i := 0; i < size; i++ {
		rtn.WriteString(fmt.Sprintf(" %10.3f", frame.Show.Unit[0]*frame.Ai.Wi[i]))
	}
	rtn.WriteString("\n        Wi = Σwi :")
	for i := 0; i < size; i++ {
		rtn.WriteString(fmt.Sprintf(" %10.3f", frame.Show.Unit[0]*frame.Ai.W[i]))
	}
	rtn.WriteString("\n               Ai :           ")
	for i := 0; i < size-1; i++ {
		rtn.WriteString(fmt.Sprintf(" %10.3f", frame.Ai.Ai[i]))
	}
	for d, str := range []string{"X", "Y"} {
		rtn.WriteString(fmt.Sprintf("\n%s方向", str))
		rtn.WriteString("\n層せん断力係数 Ci :           ")
		for i := 1; i < size; i++ {
			rtn.WriteString(fmt.Sprintf(" %10.3f", frame.Ai.Ci[d][i]))
		}
		rtn.WriteString("\n層せん断力     Qi :           ")
		for i := 1; i < size; i++ {
			rtn.WriteString(fmt.Sprintf(" %10.3f", frame.Show.Unit[0]*frame.Ai.Qi[d][i]))
		}
		rtn.WriteString("\n各階外力       Hi :           ")
		for i := 1; i < size; i++ {
			rtn.WriteString(fmt.Sprintf(" %10.3f", frame.Show.Unit[0]*frame.Ai.Hi[d][i]))
		}
		rtn.WriteString("\n外力係数    Hi/wi :")
		for i := 0; i < size; i++ {
			rtn.WriteString(fmt.Sprintf(" %10.3f", facts[d][i]))
		}
	}
	rtn.WriteString("\n")
	return rtn.String(), nil
}

func (frame *Frame) WindPressure() (string, error) {
	if frame.Ai.Nfloor == 0 && len(frame.Ai.Boundary) == 0 {
		return "", fmt.Errorf("level isn't set up")
	}
	xmin, xmax, ymin, ymax, zmin, zmax := frame.Bbox(false)
	breadth := ymax - ymin
	length := xmax - xmin
	height := zmax - zmin
	roughness := frame.Wind.Roughness
	var zb, zg, alpha, er, gf0, gf1, gf float64
	switch roughness {
	case 1:
		zb = 5.0
		zg = 250.0
		alpha = 0.10
		gf0 = 2.0
		gf1 = 1.8
	case 2:
		zb = 5.0
		zg = 350.0
		alpha = 0.15
		gf0 = 2.2
		gf1 = 2.0
	case 3:
		zb = 5.0
		zg = 450.0
		alpha = 0.20
		gf0 = 2.5
		gf1 = 2.1
	case 4:
		zb = 10.0
		zg = 550.0
		alpha = 0.27
		gf0 = 3.1
		gf1 = 2.3
	}
	if height <= zb {
		er = 1.7 * math.Pow(zb/zg, alpha)
	} else {
		er = 1.7 * math.Pow(height/zg, alpha)
	}
	if height <= 10.0 {
		gf = gf0
	} else if height >= 40.0 {
		gf = gf1
	} else {
		gf = gf0 + (height-10.0)/(40.0-10.0)*(gf1-gf0)
	}
	e := er * er * gf
	q := 0.6 * e * math.Pow(frame.Wind.Velocity*frame.Wind.Factor, 2.0) / 9.80665
	size := len(frame.Ai.Level) - 1
	z := make([]float64, size)
	kz := make([]float64, size)
	cpe := make([]float64, size)
	cpi := make([]float64, size)
	cf := make([]float64, size)
	wx := make([]float64, size)
	wy := make([]float64, size)
	qx := make([]float64, size)
	qy := make([]float64, size)
	for i := 0; i < size; i++ {
		z[i] = frame.Ai.Level[i+1]
		if height <= zb {
			kz[i] = 1.0
		} else {
			if z[i] <= zb {
				kz[i] = math.Pow(zb/height, 2*alpha)
			} else {
				kz[i] = math.Pow(z[i]/height, 2*alpha)
			}
		}
		cpe[i] = 0.8 * kz[i]
		cpi[i] = -0.2
		cf[i] = cpe[i] - cpi[i]
		wx[i] = q * cf[i] * (frame.Ai.Level[i+1] - frame.Ai.Level[i]) * breadth * 0.001
		wy[i] = q * cf[i] * (frame.Ai.Level[i+1] - frame.Ai.Level[i]) * length * 0.001
		for j := 0; j <= i; j++ {
			qx[j] += wx[i]
			qy[j] += wy[i]
		}
	}
	var otp bytes.Buffer
	otp.WriteString("風圧力の算定・閉鎖型の建築物\n建築基準法令第87条\n\n")
	otp.WriteString(fmt.Sprintf("建物高さ H [m]               : %.3f\n", height))
	otp.WriteString(fmt.Sprintf("地表面粗度区分               : %d\n", frame.Wind.Roughness))
	otp.WriteString(fmt.Sprintf("基準風速 V0 [m/s]            : %.3f\n", frame.Wind.Velocity))
	otp.WriteString(fmt.Sprintf("風速倍率                     : %.3f\n", frame.Wind.Factor))
	otp.WriteString(fmt.Sprintf("Zb[m]                        : %.3f\n", zb))
	otp.WriteString(fmt.Sprintf("ZG[m]                        : %.3f\n", zg))
	otp.WriteString(fmt.Sprintf("α                           : %.3f\n", alpha))
	otp.WriteString(fmt.Sprintf("Er                           : %.3f\n", er))
	otp.WriteString(fmt.Sprintf("Gf                           : %.3f\n", gf))
	otp.WriteString(fmt.Sprintf("E=Er^2 Gf                    : %.3f\n", e))
	switch frame.Show.UnitName[0] {
	case "tf":
		otp.WriteString(fmt.Sprintf("q=0.6 E V0^2[kgf/m2]        : %.3f\n", q*frame.Show.Unit[0]))
	case "kN":
		otp.WriteString(fmt.Sprintf("q=0.6 E V0^2[N/m2]          : %.3f\n", q*frame.Show.Unit[0]))
	default:
		otp.WriteString(fmt.Sprintf("q=0.6 E V0^2[kgf/m2]        : %.3f\n", q*frame.Show.Unit[0]))
	}
	otp.WriteString("\n風圧力を算定する高さ Z1[m]   :")
	for i := 0; i < size; i++ {
		otp.WriteString(fmt.Sprintf(" %10.3f", z[i]))
	}
	otp.WriteString("\n張間方向幅 b1[m]             :")
	for i := 0; i < size; i++ {
		otp.WriteString(fmt.Sprintf(" %10.3f", breadth))
	}
	otp.WriteString("\n桁行方向幅 b2[m]             :")
	for i := 0; i < size; i++ {
		otp.WriteString(fmt.Sprintf(" %10.3f", length))
	}
	otp.WriteString("\nkz                           :")
	for i := 0; i < size; i++ {
		otp.WriteString(fmt.Sprintf(" %10.3f", kz[i]))
	}
	otp.WriteString("\n外圧係数 Cpe                 :")
	for i := 0; i < size; i++ {
		otp.WriteString(fmt.Sprintf(" %10.3f", cpe[i]))
	}
	otp.WriteString("\n内圧係数 Cpi                 :")
	for i := 0; i < size; i++ {
		otp.WriteString(fmt.Sprintf(" %10.3f", cpi[i]))
	}
	otp.WriteString("\n風力係数 Cf                  :")
	for i := 0; i < size; i++ {
		otp.WriteString(fmt.Sprintf(" %10.3f", cf[i]))
	}
	otp.WriteString(fmt.Sprintf("\n外力 Wx=Cf (Zi-Zi+1) b1[%s]  :", frame.Show.UnitName[0]))
	for i := 0; i < size; i++ {
		otp.WriteString(fmt.Sprintf(" %10.3f", wx[i]*frame.Show.Unit[0]))
	}
	otp.WriteString(fmt.Sprintf("\n     Wy=Cf (Zi-Zi+1) b2[%s]  :", frame.Show.UnitName[0]))
	for i := 0; i < size; i++ {
		otp.WriteString(fmt.Sprintf(" %10.3f", wy[i]*frame.Show.Unit[0]))
	}
	otp.WriteString(fmt.Sprintf("\n層せん断力 Qwx=ΣWx[%s]      :", frame.Show.UnitName[0]))
	for i := 0; i < size; i++ {
		otp.WriteString(fmt.Sprintf(" %10.3f", qx[i]*frame.Show.Unit[0]))
	}
	otp.WriteString(fmt.Sprintf("\n           Qwy=ΣWy[%s]      :", frame.Show.UnitName[0]))
	for i := 0; i < size; i++ {
		otp.WriteString(fmt.Sprintf(" %10.3f", qy[i]*frame.Show.Unit[0]))
	}
	otp.WriteString(fmt.Sprintf("\n地震層せん断力 Qex[%s]       :", frame.Show.UnitName[0]))
	for i := 0; i < size; i++ {
		otp.WriteString(fmt.Sprintf(" %10.3f", frame.Ai.Qi[0][i+1]*frame.Show.Unit[0]))
	}
	otp.WriteString(fmt.Sprintf("\n               Qey[%s]       :", frame.Show.UnitName[0]))
	for i := 0; i < size; i++ {
		otp.WriteString(fmt.Sprintf(" %10.3f", frame.Ai.Qi[1][i+1]*frame.Show.Unit[0]))
	}
	otp.WriteString("\n\n")
	check := false
	for i := 0; i < size; i++ {
		fmt.Println(qx[i], frame.Ai.Qi[0][i+1], qy[i], frame.Ai.Qi[1][i+1])
		if qx[i] > frame.Ai.Qi[0][i+1] {
			if frame.Ai.Qi[0][i+1] != 0.0 {
				otp.WriteString(fmt.Sprintf("風圧力の方が大きい: X方向 %d階 C0=%.3f相当\n", i+1, frame.Ai.Base[0]*qx[i]/frame.Ai.Qi[0][i+1]))
			} else {
				otp.WriteString(fmt.Sprintf("風圧力の方が大きい: X方向 %d階 Qex=0.0\n", i+1))
			}
			check = true
		}
		if qy[i] > frame.Ai.Qi[1][i+1] {
			if frame.Ai.Qi[1][i+1] != 0.0 {
				otp.WriteString(fmt.Sprintf("風圧力の方が大きい: Y方向 %d階 C0=%.3f相当\n", i+1, frame.Ai.Base[1]*qy[i]/frame.Ai.Qi[1][i+1]))
			} else {
				otp.WriteString(fmt.Sprintf("風圧力の方が大きい: Y方向 %d階 Qey=0.0\n", i+1))
			}
			check = true
		}
	}
	if !check {
		otp.WriteString("以上より、風圧力は地震力よりも小さい。\n")
	}
	return otp.String(), nil
}

func (frame *Frame) SaveAsArclm(name string) error {
	if name == "" {
		name = frame.Path
	}
	for i, p := range []string{"L", "X", "Y"} {
		fn := Ce(name, InputExt[i])
		err := frame.Arclms[p].SaveInput(fn)
		if err != nil {
			return err
		}
		frame.DataFileName[p] = fn
	}
	return nil
}

func (frame *Frame) ReadArclmData(af *arclm.Frame, per string) {
	for _, an := range af.Nodes {
		if n, ok := frame.Nodes[an.Num]; ok {
			disp := make([]float64, 6)
			reaction := make([]float64, 6)
			for i := 0; i < 6; i++ {
				disp[i] = an.Disp[i]
				reaction[i] = an.Reaction[i]
			}
			n.Disp[per] = disp
			n.Reaction[per] = reaction
		}
	}
	for _, ael := range af.Elems {
		if el, ok := frame.Elems[ael.Num]; ok {
			stress := make(map[int][]float64, 2)
			for i, n := range ael.Enod {
				stress[n.Num] = make([]float64, 6)
				for j := 0; j < 6; j++ {
					stress[n.Num][j] = ael.Stress[6*i+j]
				}
			}
			el.Stress[per] = stress
			el.Values["ENERGY"] = ael.Energy
			el.Values["ENERGYB"] = ael.Energyb
			if ael.IsValid {
				el.Lock = false
			} else {
				el.Lock = true
			}
		}
	}
}

// SectionRate
func (frame *Frame) SectionRateCalculation(fn string, long, x1, x2, y1, y2 string, sign float64, cond *Condition) error {
	var enum int
	elems := make([]*Elem, len(frame.Elems))
	for _, el := range frame.Elems {
		if !el.IsLineElem() {
			continue
		}
		elems[enum] = el
		enum++
	}
	if enum == 0 {
		return nil
	}
	var otp, rat, rlt bytes.Buffer
	var rate []float64
	var rate2 float64
	var err error
	stname := []string{"鉛直時Z    :", "水平時X    :", "水平時X負  :", "水平時Y    :", "水平時Y負  :"}
	pername := []string{"長期       :", "短期X正方向:", "短期X負方向:", "短期Y正方向:", "短期Y負方向:"}
	calc1 := func(allow SectionRate, st1, st2, fact []float64, sign float64) ([]float64, error) {
		stress := make([]float64, 12)
		if st2 != nil {
			for i := 0; i < 2; i++ {
				for j := 0; j < 6; j++ {
					stress[6*i+j] = st1[6*i+j] + sign*fact[j]*st2[6*i+j]
				}
			}
		} else {
			stress = st1
		}
		rate, txt, err := Rate1(allow, stress, cond)
		if err != nil {
			return rate, err
		}
		otp.WriteString(txt)
		return rate, nil
	}
	calc2 := func(allow SectionRate, n1, n2, fact, sign float64) (float64, error) {
		stress := n1 + sign*fact*n2
		rate, txt, err := Rate2(allow, stress, cond)
		if err != nil {
			return rate, err
		}
		otp.WriteString(txt)
		return rate, nil
	}
	maxrate := func(rate ...float64) float64 {
		rtn := 0.0
		for _, val := range rate {
			if val > rtn {
				rtn = val
			}
		}
		return rtn
	}
	factor := []float64{cond.Nfact, cond.Qfact, cond.Qfact, 1.0, cond.Mfact, cond.Mfact}
	otp.WriteString("断面算定 \"S,RC,SRC\" 長期,短期,終局\n")
	otp.WriteString("使用ファイル\n")
	otp.WriteString(fmt.Sprintf("入力データ               =%s\n", frame.DataFileName["L"]))
	otp.WriteString(fmt.Sprintf("鉛直荷重時解析結果       =%s\n", frame.ResultFileName["L"]))
	otp.WriteString(fmt.Sprintf("水平荷重時解析結果 X方向 =%s\n", frame.ResultFileName["X"]))
	otp.WriteString(fmt.Sprintf("                   Y方向 =%s\n", frame.ResultFileName["Y"]))
	otp.WriteString(fmt.Sprintf("仮定断面                 =%s\n", frame.LstFileName))
	otp.WriteString("\n単位系 tf(kN),tfm(kNm)\n")
	otp.WriteString("\nAs:鉄骨 Ar:主筋 Ac:コンクリート Ap:ＰＣストランド\n")
	otp.WriteString("N:軸力 Q:せん断力 Mt:ねじりモーメント M:曲げモーメント\n")
	otp.WriteString("添字 i:始端 j:終端 c:中央\n")
	otp.WriteString("a:許容 u:終局\n")
	elems = elems[:enum]
	sort.Sort(ElemByNum{elems})
	maxrateelem := make(map[int][]*Elem)
	for _, el := range elems {
		al, original, err := el.GetSectionRate()
		if err != nil {
			continue
		}
		switch el.Etype {
		case COLUMN, GIRDER:
			var isrc bool
			switch al.(type) {
			case *RCColumn, *RCGirder:
				isrc = true
			default:
				isrc = false
			}
			if isrc && cond.Qfact < 2.0 {
				factor[1] = 2.0
				factor[2] = 2.0
			} else {
				factor[1] = cond.Qfact
				factor[2] = cond.Qfact
			}
			var qlrate, qsrate, qurate, mlrate, msrate, murate float64
			msrates := make([]float64, 4)
			cond.Length = el.Length() * 100.0 // [cm]
			otp.WriteString(strings.Repeat("-", 202))
			otp.WriteString(fmt.Sprintf("\n部材:%d 始端:%d 終端:%d 断面:%d=%s 材長=%.1f[cm] Mx内法=%.1f[cm] My内法=%.1f[cm]\n", el.Num, el.Enod[0].Num, el.Enod[1].Num, el.Sect.Num, strings.Replace(al.TypeString(), "　", "", -1), cond.Length, cond.Length, cond.Length))
			otp.WriteString("応力       :        N                Qxi                Qxj                Qyi                Qyj                 Mt                Mxi                Mxj                Myi                Myj\n")
			stress := make([][]float64, 5)
			for p, per := range []string{long, x1, x2, y1, y2} {
				stress[p] = make([]float64, 12)
				if st, ok := el.Stress[per]; ok {
					for i := 0; i < 2; i++ {
						for j := 0; j < 6; j++ {
							stress[p][6*i+j] = st[el.Enod[i].Num][j]
						}
					}
				}
				if (p == 2 && x1 == x2) || (p == 4 && y1 == y2) {
					continue
				}
				otp.WriteString(stname[p])
				for i := 0; i < 6; i++ {
					for j := 0; j < 2; j++ {
						otp.WriteString(fmt.Sprintf(" %8.3f(%8.2f)", stress[p][6*j+i], stress[p][6*j+i]*SI))
						if i == 0 || i == 3 {
							break
						}
					}
				}
				otp.WriteString("\n")
			}
			otp.WriteString("\n")
			if cond.Verbose {
				switch al.(type) {
				case *SColumn:
					sh := al.(*SColumn).Shape
					otp.WriteString(fmt.Sprintf("# 断面性能詳細\n"))
					otp.WriteString(fmt.Sprintf("#     断面積:             A   = %10.4f [cm2]\n", sh.A()))
					otp.WriteString(fmt.Sprintf("#     Qax算定用断面積:    Asx = %10.4f [cm2]\n", sh.Asx()))
					otp.WriteString(fmt.Sprintf("#     Qay算定用断面積:    Asy = %10.4f [cm2]\n", sh.Asy()))
					otp.WriteString(fmt.Sprintf("#     断面二次モーメント: Ix  = %10.4f [cm4]\n", sh.Ix()))
					otp.WriteString(fmt.Sprintf("#                         Iy  = %10.4f [cm4]\n", sh.Iy()))
					if an, ok := sh.(ANGLE); ok {
						otp.WriteString(fmt.Sprintf("#                         Imin= %10.4f [cm4]\n", an.Imin()))
					}
					otp.WriteString(fmt.Sprintf("#     一様ねじり定数      J   = %10.4f [cm4]\n", sh.J()))
					otp.WriteString(fmt.Sprintf("#     断面係数:           Zx  = %10.4f [cm3]\n", sh.Zx()))
					otp.WriteString(fmt.Sprintf("#                         Zy  = %10.4f [cm3]\n", sh.Zy()))
				case *WoodColumn:
					sh := al.(*WoodColumn).Shape
					otp.WriteString(fmt.Sprintf("# 断面性能詳細\n"))
					otp.WriteString(fmt.Sprintf("#     断面積:             A   = %10.4f [cm2]\n", sh.A()))
					otp.WriteString(fmt.Sprintf("#     Qax算定用断面積:    Asx = %10.4f [cm2]\n", sh.Asx()))
					otp.WriteString(fmt.Sprintf("#     Qay算定用断面積:    Asy = %10.4f [cm2]\n", sh.Asy()))
					otp.WriteString(fmt.Sprintf("#     断面二次モーメント: Ix  = %10.4f [cm4]\n", sh.Ix()))
					otp.WriteString(fmt.Sprintf("#                         Iy  = %10.4f [cm4]\n", sh.Iy()))
					otp.WriteString(fmt.Sprintf("#     一様ねじり定数:     J   = %10.4f [cm4]\n", sh.J()))
					otp.WriteString(fmt.Sprintf("#     断面係数:           Zx  = %10.4f [cm3]\n", sh.Zx()))
					otp.WriteString(fmt.Sprintf("#                         Zy  = %10.4f [cm3]\n", sh.Zy()))
				case *RCGirder:
					rc := al.(*RCGirder)
					otp.WriteString(fmt.Sprintf("# 断面性能詳細\n"))
					otp.WriteString(fmt.Sprintf("#     断面積:             A  = %12.2f [cm2]\n", rc.Area()))
					otp.WriteString(fmt.Sprintf("#     断面二次モーメント: Ix = %12.2f [cm4]\n", rc.Ix()))
					otp.WriteString(fmt.Sprintf("#                         Iy = %12.2f [cm4]\n", rc.Iy()))
					otp.WriteString(fmt.Sprintf("#     一様ねじり定数:     J  = %12.2f [cm4]\n", rc.J()))
				}
			}
			if cond.Temporary {
				cond.Period = "S"
			} else {
				cond.Period = "L"
			}
			otp.WriteString(pername[0])
			rate, err = calc1(al, stress[0], nil, nil, 1.0)
			qlrate = maxrate(rate[1], rate[2], rate[7], rate[8])
			if isrc {
				mlrate = maxrate(rate[4], rate[5], rate[10], rate[11])
			} else {
				if rate[0] >= 1.0 {
					mlrate = 10.0
				} else {
					mlrate = maxrate(rate[4], rate[5], rate[10], rate[11]) / (1.0 - rate[0])
				}
			}
			if cond.Skipshort {
				otp.WriteString(fmt.Sprintf("\nMAX:Q/QaL=%.5f Q/QaS=%.5f M/MaL=%.5f M/MaS=%.5f\n", qlrate, qsrate, mlrate, msrate))
				el.Rate = []float64{qlrate, qsrate, qurate, mlrate, msrate, murate}
				break
			}
			cond.Period = "S"
			var s float64
			for p := 1; p < 5; p++ {
				if p%2 == 1 {
					otp.WriteString("\n")
					s = 1.0
				} else {
					s = sign
				}
				otp.WriteString(pername[p])
				rate, err = calc1(al, stress[0], stress[p], factor, s)
				qsrate = maxrate(qsrate, rate[1], rate[2], rate[7], rate[8])
				if isrc {
					msrates[p-1] = maxrate(rate[4], rate[5], rate[10], rate[11])
				} else {
					if rate[0] >= 1.0 {
						msrates[p-1] = 10.0
					} else {
						msrates[p-1] = maxrate(rate[4], rate[5], rate[10], rate[11]) / (1.0 - rate[0])
					}
				}
			}
			msrate = maxrate(msrates...)
			otp.WriteString(fmt.Sprintf("\nMAX:Q/QaL=%.5f Q/QaS=%.5f M/MaL=%.5f M/MaS=%.5f\n", qlrate, qsrate, mlrate, msrate))
			el.Rate = []float64{qlrate, qsrate, qurate, mlrate, msrate, murate}
		case BRACE, WBRACE, SBRACE:
			var qlrate, qsrate, qurate float64
			var fact float64
			if el.Etype == BRACE {
				fact = cond.Bfact
				cond.Length = el.Length() * 100.0 // [cm]
			} else {
				fact = cond.Wfact
				if bros, ok := el.Brother(); ok {
					cond.Length = 0.5 * (el.Length() + bros.Length()) * 100.0 // [cm]
				} else {
					cond.Length = el.Length() * 100.0 // [cm]
				}
			}
			cond.Width = el.Width() * 100.0 // [cm]
			otp.WriteString(strings.Repeat("-", 202))
			var sectstring string
			if original {
				sectstring = fmt.Sprintf("◯%d/　%d", el.OriginalSection().Num, el.Sect.Num)
			} else {
				sectstring = fmt.Sprintf("　%d/◯%d", el.OriginalSection().Num, el.Sect.Num)
			}
			otp.WriteString(fmt.Sprintf("\n部材:%d 始端:%d 終端:%d 断面:%s=%s 材長=%.1f[cm] Mx内法=%.1f[cm] My内法=%.1f[cm]\n", el.Num, el.Enod[0].Num, el.Enod[1].Num, sectstring, strings.Replace(al.TypeString(), "　", "", -1), cond.Length, cond.Length, cond.Length))
			otp.WriteString("応力       :        N\n")
			stress := make([]float64, 5)
			for p, per := range []string{long, x1, x2, y1, y2} {
				if st, ok := el.Stress[per]; ok {
					stress[p] = st[el.Enod[0].Num][0]
				}
				if (p == 2 && x1 == x2) || (p == 4 && y1 == y2) {
					continue
				}
				otp.WriteString(stname[p])
				otp.WriteString(fmt.Sprintf(" %8.3f(%8.2f)", stress[p], stress[p]*SI))
				otp.WriteString("\n")
			}
			otp.WriteString("\n")
			if cond.Temporary {
				cond.Period = "S"
			} else {
				cond.Period = "L"
			}
			otp.WriteString(pername[0])
			rate2, err = calc2(al, stress[0], 0.0, 0.0, 1.0)
			qlrate = rate2
			if cond.Skipshort {
				otp.WriteString(fmt.Sprintf("\nMAX:Q/QaL=%.5f Q/QaS=%.5f\n", qlrate, qsrate))
				el.Rate = []float64{qlrate, qsrate, qurate}
				break
			}
			cond.Period = "S"
			var s float64
			for p := 1; p < 5; p++ {
				if p%2 == 1 {
					otp.WriteString("\n")
					s = 1.0
				} else {
					s = sign
				}
				otp.WriteString(pername[p])
				rate2, err = calc2(al, stress[0], stress[p], fact, s)
				qsrate = maxrate(qsrate, rate2)
			}
			otp.WriteString(fmt.Sprintf("\nMAX:Q/QaL=%.5f Q/QaS=%.5f\n", qlrate, qsrate))
			el.Rate = []float64{qlrate, qsrate, qurate}
		}
		rat.WriteString(el.OutputRate())
		for _, r := range el.Rate {
			if r >= 1.0 {
				rlt.WriteString(el.OutputRateRlt())
				break
			}
		}
		switch el.Etype {
		case COLUMN, GIRDER:
			if mels, ok := maxrateelem[al.Num()]; ok {
				for ind, pos := range []int{0, 1, 3, 4} {
					if el.Rate[pos] > mels[ind].Rate[pos] {
						maxrateelem[al.Num()][ind] = el
					}
				}
			} else {
				maxrateelem[al.Num()] = []*Elem{el, el, el, el}
			}
		case BRACE, WBRACE, SBRACE:
			if mels, ok := maxrateelem[al.Num()]; ok {
				for ind, pos := range []int{0, 1} {
					if el.Rate[pos] > mels[ind].Rate[pos] {
						maxrateelem[al.Num()][ind] = el
					}
				}
			} else {
				maxrateelem[al.Num()] = []*Elem{el, el}
			}
		}
	}
	otp.WriteString("==========================================================================================================================================================================================================\n各断面種別の許容,終局曲げ安全率の最大値\n\n")
	keys := make([]int, len(maxrateelem))
	i := 0
	for k := range maxrateelem {
		keys[i] = k
		i++
	}
	sort.Ints(keys)
	maxql := 0.0
	maxqs := 0.0
	maxml := 0.0
	maxms := 0.0
	for _, k := range keys {
		otp.WriteString(fmt.Sprintf("断面記号: %d %s   As=   0.00[cm2] Ar=   0.00[cm2] Ac=    0.00[cm2] MAX:", k, frame.Sects[k].Allow.TypeString()))
		els := maxrateelem[k]
		switch els[0].Etype {
		case COLUMN, GIRDER:
			otp.WriteString(fmt.Sprintf("Q/QaL=%.5f Q/QaS=%.5f M/MaL=%.5f M/MaS=%.5f\n", els[0].Rate[0], els[1].Rate[1], els[2].Rate[3], els[3].Rate[4]))
			if els[0].Rate[0] > maxql {
				maxql = els[0].Rate[0]
			}
			if els[1].Rate[1] > maxqs {
				maxqs = els[1].Rate[1]
			}
			if els[2].Rate[3] > maxml {
				maxml = els[2].Rate[3]
			}
			if els[3].Rate[4] > maxms {
				maxms = els[3].Rate[4]
			}
		case BRACE, WBRACE, SBRACE:
			otp.WriteString(fmt.Sprintf("Q/QaL=%.5f Q/QaS=%.5f\n", els[0].Rate[0], els[1].Rate[1]))
			if els[0].Rate[0] > maxql {
				maxql = els[0].Rate[0]
			}
			if els[1].Rate[1] > maxqs {
				maxqs = els[1].Rate[1]
			}
		}
	}
	otp.WriteString(fmt.Sprintf("\n安全率の最大値\n Q/QaL=%7.5f Q/QaS=%7.5f\n M/MaL=%7.5f M/MaS=%7.5f\n", maxql, maxqs, maxml, maxms))
	otp.WriteString("==========================================================================================================================================================================================================\n各断面種別の入力情報の確認\n\n")
	otp.WriteString("                              A[cm2]      Ix[cm4]      Iy[cm4]       J[cm4]\n")
	otp.WriteString("                              t[cm]\n")
	otp.WriteString(frame.CheckLst(keys))
	w, err := os.Create(Ce(fn, ".tst"))
	defer w.Close()
	if err != nil {
		return err
	}
	otp = AddCR(otp)
	otp.WriteTo(w)
	wrat, err := os.Create(Ce(fn, ".rat"))
	defer wrat.Close()
	if err != nil {
		return err
	}
	rat = AddCR(rat)
	rat.WriteTo(wrat)
	wrlt, err := os.Create(Ce(fn, ".rlt"))
	defer wrlt.Close()
	if err != nil {
		return err
	}
	rlt = AddCR(rlt)
	rlt.WriteTo(wrlt)
	return nil
}

func (frame *Frame) CheckLst(secnum []int) string {
	var otp bytes.Buffer
	for _, snum := range secnum {
		if snum < 700 {
			a1, _ := frame.Sects[snum].Area(0)
			ix1, _ := frame.Sects[snum].Ix(0)
			iy1, _ := frame.Sects[snum].Iy(0)
			j1, _ := frame.Sects[snum].J(0)
			var a2, ix2, iy2, j2 float64
			switch al := frame.Sects[snum].Allow.(type) {
			case *SColumn:
				sh := al.Shape
				a2 = sh.A()
				ix2 = sh.Ix()
				iy2 = sh.Iy()
				j2 = sh.J()
			case *SGirder:
				sh := al.Shape
				a2 = sh.A()
				ix2 = sh.Ix()
				iy2 = sh.Iy()
				j2 = sh.J()
			case *RCColumn:
				sh := al.CShape
				a2 = sh.Area()
				ix2 = sh.Ix()
				iy2 = sh.Iy()
				j2 = sh.J()
			case *RCGirder:
				sh := al.CShape
				a2 = sh.Area()
				ix2 = sh.Ix()
				iy2 = sh.Iy()
				j2 = sh.J()
			case *WoodColumn:
				sh := al.Shape
				a2 = sh.A()
				ix2 = sh.Ix()
				iy2 = sh.Iy()
				j2 = sh.J()
			case *WoodGirder:
				sh := al.Shape
				a2 = sh.A()
				ix2 = sh.Ix()
				iy2 = sh.Iy()
				j2 = sh.J()
			default:
			}
			otp.WriteString(fmt.Sprintf("断面記号: %3d 応力解析: %12.3f %12.3f %12.3f %12.3f\n", snum, a1*1e4, ix1*1e8, iy1*1e8, j1*1e8))
			otp.WriteString(fmt.Sprintf("              断面算定: %12.3f %12.3f %12.3f %12.3f\n", a2, ix2, iy2, j2))
		} else {
			t1, _ := frame.Sects[snum].Thick(0)
			var t2 float64
			switch al := frame.Sects[snum].Allow.(type) {
			case *SWall:
				t2 = al.Thickness()
			case *RCWall:
				t2 = al.Thick
			case *RCSlab:
				t2 = al.Thick
			case *WoodWall:
				t2 = al.Thick
			case *WoodSlab:
				t2 = al.Thick
			default:
			}
			otp.WriteString(fmt.Sprintf("断面記号: %3d 応力解析: %12.3f\n", snum, t1*1e2))
			otp.WriteString(fmt.Sprintf("              断面算定: %12.3f\n", t2))
		}
	}
	return otp.String()
}

func (frame *Frame) Facts(fn string, etypes []int, skipany, skipall []int, period []string) error {
	var err error
	l := frame.Ai.Nfloor
	if l < 2 {
		return errors.New("Facts: Nfloor < 2")
	}
	nodes := make([][]*Node, l)
	elems := make([][]*Elem, l-1)
	for i := 0; i < l; i++ {
		nodes[i] = make([]*Node, 0)
		if i < l-1 {
			elems[i] = make([]*Elem, 0)
		}
	}
	var cont bool
fact_node:
	for _, n := range frame.Nodes {
		if n.Coord[2] < frame.Ai.Boundary[0] {
			continue
		}
		if skipany != nil || skipall != nil {
			els := frame.SearchElem(n)
			cont = true
			for _, el := range els {
				for _, sany := range skipany {
					if el.Sect.Num == sany {
						continue fact_node
					}
				}
				for _, sall := range skipall {
					if el.Sect.Num != sall {
						cont = false
					}
				}
			}
			if cont {
				continue fact_node
			}
		}
		for i := 0; i < l; i++ {
			if n.Coord[2] < frame.Ai.Boundary[i+1] {
				nodes[i] = append(nodes[i], n)
				continue fact_node
			}
		}
	}
	for _, el := range frame.Elems {
		contained := false
		for _, et := range etypes {
			if el.Etype == et {
				contained = true
				break
			}
		}
		if skipany != nil || skipall != nil {
			for _, sany := range skipany {
				if el.Sect.Num == sany {
					contained = false
				}
			}
			for _, sall := range skipall {
				if el.Sect.Num == sall {
					contained = false
				}
			}
		}
		if !contained {
			continue
		}
		for i := 0; i < l-1; i++ {
			if (el.Enod[0].Coord[2]-frame.Ai.Boundary[i+1])*(el.Enod[1].Coord[2]-frame.Ai.Boundary[i+1]) < 0 {
				elems[i] = append(elems[i], el)
				// break
			}
		}
	}
	f := NewFact(l, true, frame.Ai.Base[0]/0.2, frame.Ai.Base[1]/0.2)
	perx := strings.Split(period[0], "@")[0]
	pery := strings.Split(period[1], "@")[0]
	f.SetFileName([]string{frame.DataFileName["L"], frame.DataFileName[perx], frame.DataFileName[pery]},
		[]string{frame.ResultFileName["L"], frame.ResultFileName[perx], frame.ResultFileName[pery]})
	for i := 0; i < len(nodes); i++ {
		sort.Sort(NodeByNum{nodes[i]})
	}
	for i := 0; i < len(elems); i++ {
		sort.Sort(ElemByNum{elems[i]})
	}
	err = f.CalcFact(nodes, elems, period)
	if err != nil {
		return err
	}
	fmt.Println(f)
	err = f.WriteTo(fn)
	if err != nil {
		return err
	}
	frame.Fes = f
	return nil
}

func (frame *Frame) FactsBySect(fn string, etypes []int, nsect, esect [][]int, period []string) error {
	var err error
	if len(nsect) != len(esect) {
		return fmt.Errorf("size error")
	}
	l := len(nsect)
	nodes := make([][]*Node, l+1)
	nnums := make([][]int, l)
	elems := make([][]*Elem, l)
	for i := 0; i < l+1; i++ {
		nodes[i] = make([]*Node, 0)
		if i < l {
			nnums[i] = make([]int, 0)
			elems[i] = make([]*Elem, 0)
		}
	}
	var contained bool
	for _, el := range frame.Elems {
		for _, et := range etypes {
			if el.Etype == et {
				contained = true
				break
			}
		}
		if !contained {
			continue
		}
	factbysect_elem1:
		for ni, ns := range nsect {
			for _, s := range ns {
				if el.Sect.Num == s {
					for _, en := range el.Enod {
						nnums[ni] = append(nnums[ni], en.Num)
					}
					break factbysect_elem1
				}
			}
		}
	factbysect_elem2:
		for ei, es := range esect {
			for _, s := range es {
				if el.Sect.Num == s {
					elems[ei] = append(elems[ei], el)
					break factbysect_elem2
				}
			}
		}
	}
factbysect_node:
	for _, n := range frame.Nodes {
		for i := 0; i < l; i++ {
			for _, nn := range nnums[i] {
				if n.Num == nn {
					nodes[i] = append(nodes[i], n)
					continue factbysect_node
				}
			}
		}
		nodes[l] = append(nodes[l], n)
	}
	f := NewFact(l+1, true, frame.Ai.Base[0]/0.2, frame.Ai.Base[1]/0.2)
	perx := strings.Split(period[0], "@")[0]
	pery := strings.Split(period[1], "@")[0]
	f.SetFileName([]string{frame.DataFileName["L"], frame.DataFileName[perx], frame.DataFileName[pery]},
		[]string{frame.ResultFileName["L"], frame.ResultFileName[perx], frame.ResultFileName[pery]})
	for i := 0; i < len(nodes); i++ {
		sort.Sort(NodeByNum{nodes[i]})
	}
	for i := 0; i < len(elems); i++ {
		sort.Sort(ElemByNum{elems[i]})
	}
	err = f.CalcFact(nodes, elems, period)
	if err != nil {
		return err
	}
	fmt.Println(f)
	err = f.WriteTo(fn)
	if err != nil {
		return err
	}
	frame.Fes = f
	return nil
}

func (frame *Frame) AmountProp(fn string, props ...int) error {
	total := 0.0
	sects := make([]*Sect, len(frame.Sects))
	snum := 0
	for _, sec := range frame.Sects {
		if sec.Num > 900 {
			continue
		}
		sects[snum] = sec
		snum++
	}
	sects = sects[:snum]
	sort.Sort(SectByNum{sects})
	var otp bytes.Buffer
	otp.WriteString("断面 名前                                     長さ     断面積   単位重量 重量\n")
	otp.WriteString("                                              面積     板厚\n")
	for _, sec := range sects {
		size := sec.PropSize(props)
		if size == 0.0 {
			continue
		}
		amount := sec.TotalAmount()
		weight := sec.PropWeight(props)
		totalweight := amount * weight
		otp.WriteString(fmt.Sprintf("%4d %-40s %8.3f %8.4f %8.4f %8.3f\n", sec.Num, sec.Name, amount, size, weight, totalweight))
		total += totalweight
	}
	otp.WriteString(fmt.Sprintf("%79s\n", fmt.Sprintf("合計: %8.3f", total)))
	w, err := os.Create(fn)
	defer w.Close()
	if err != nil {
		return err
	}
	otp.WriteTo(w)
	return nil
}

func (frame *Frame) AmountLst(fn string, sects ...int) error {
	var otp bytes.Buffer
	otp.WriteString("断面 名前                                    長さ/面積     鉄骨量     ＲＣ量     鉄筋量   型枠面積\n")
	otp.WriteString("                                                [m/m2]       [tf]       [m3]       [tf]       [m2]\n")
	total := NewAmount()
	total["REINS"] = 0.0
	total["CONCRETE"] = 0.0
	total["FORMWORK"] = 0.0
	total["STEEL"] = 0.0
	for _, s := range sects {
		if sec, ok := frame.Sects[s]; ok {
			var a Amount
			tot := 0.0
			if sec.HasArea(0) {
				if al := sec.Allow; al != nil {
					tot = sec.TotalAmount()
					if tot == 0.0 {
						continue
					}
					al := al.Amount()
					a = NewAmount()
					if val, ok := al["REINS"]; ok {
						a["REINS"] = val * tot * 7.8
					}
					h, err := sec.Hiju(0)
					if err == nil {
						if val, ok := al["CONCRETE"]; ok {
							a["CONCRETE"] = val * tot * h / 2.4
						}
					}
					if val, ok := al["FORMWORK"]; ok {
						a["FORMWORK"] = val * tot
					}
					if val, ok := al["STEEL"]; ok {
						a["STEEL"] = val * tot * 7.8
					}
				}
			} else if sec.HasThick(0) {
				if !sec.HasBrace() {
					continue
				}
				if al := sec.Allow; al != nil {
					tot = sec.TotalAmount()
					al := al.Amount()
					a = NewAmount()
					if val, ok := al["REINS"]; ok {
						a["REINS"] = val * tot * 2 * 7.8
					}
					if val, ok := al["CONCRETE"]; ok {
						a["CONCRETE"] = val * tot
					}
					if val, ok := al["FORMWORK"]; ok {
						a["FORMWORK"] = val * tot
					}
				}
			}
			if a != nil {
				total["REINS"] += a["REINS"]
				total["CONCRETE"] += a["CONCRETE"]
				total["FORMWORK"] += a["FORMWORK"]
				total["STEEL"] += a["STEEL"]
				otp.WriteString(fmt.Sprintf("%4d %-40s %8.3f %10.4f %10.4f %10.4f %10.4f\n", sec.Num, sec.Name, tot, a["STEEL"], a["CONCRETE"], a["REINS"], a["FORMWORK"]))
			}
		}
	}
	otp.WriteString("--------------------------------------------------------------------------------------------------\n")
	otp.WriteString(fmt.Sprintf("                                                       %10.4f %10.4f %10.4f %10.4f\n", total["STEEL"], total["CONCRETE"], total["REINS"], total["FORMWORK"]))
	w, err := os.Create(fn)
	defer w.Close()
	if err != nil {
		return err
	}
	otp.WriteTo(w)
	return nil
}

func (view *View) SetVectorAngle(vec []float64) error {
	if len(vec) < 3 {
		return errors.New("SetVectorAngle: vector size error")
	}
	l1 := math.Sqrt(vec[0]*vec[0] + vec[1]*vec[1] + vec[2]*vec[2])
	l2 := math.Sqrt(vec[0]*vec[0] + vec[1]*vec[1])
	if l2 != 0.0 {
		view.Angle[0] = math.Atan2(vec[2], l1)
		if vec[1] >= 0.0 {
			view.Angle[1] = math.Acos(vec[0]/l2)*180.0/math.Pi - 180.0
		} else {
			view.Angle[1] = -math.Acos(vec[0]/l2)*180.0/math.Pi + 180.0
		}
	}
	return nil
}

func (frame *Frame) ShowPlane(n1, n2, n3 *Node, eps float64) error {
	v1 := Direction(n1, n2, true)
	v2 := Direction(n2, n3, true)
	if IsParallel(v1, v2, eps) {
		return errors.New("nodes are on the same line")
	}
	nv := Cross(v1, v2)
	frame.View.SetVectorAngle(nv)
	num := 0
	ns := make([]*Node, len(frame.Nodes))
	for _, n := range frame.Nodes {
		n.Hide()
		if math.Abs(Dot(nv, Direction(n1, n, true), 3)) < eps {
			n.Show()
			ns[num] = n
			num++
		}
	}
	for _, el := range frame.Elems {
		el.Hide()
	}
	for _, el := range frame.NodeToElemAll(ns[:num]...) {
		el.Show()
	}
	return nil
}

func (frame *Frame) SetFocus(coord []float64) {
	if coord == nil {
		xmin, xmax, ymin, ymax, zmin, zmax := frame.Bbox(true)
		mins := []float64{xmin, ymin, zmin}
		maxs := []float64{xmax, ymax, zmax}
		for i := 0; i < 3; i++ {
			frame.View.Focus[i] = 0.5 * (mins[i] + maxs[i])
		}
	} else {
		for i := 0; i < 3; i++ {
			frame.View.Focus[i] = coord[i]
		}
	}
	for {
		if frame.View.Angle[1] <= 360.0 {
			break
		}
		frame.View.Angle[1] -= 360.0
	}
	for {
		if frame.View.Angle[1] >= 0.0 {
			break
		}
		frame.View.Angle[1] += 360.0
	}
}

func (frame *Frame) PickElem(x, y, eps float64) *Elem {
	el := frame.PickLineElem(x, y, eps)
	if el == nil {
		els := frame.PickPlateElem(x, y)
		if len(els) > 0 {
			el = els[0]
		}
	}
	return el
}

func (frame *Frame) PickLineElem(x, y, eps float64) *Elem {
	mindist := eps
	var rtn *Elem
	var icoord, jcoord []float64
	for _, v := range frame.Elems {
		if v.IsHidden(frame.Show) {
			continue
		}
		if frame.Show.PlotState&PLOT_UNDEFORMED != 0 {
			icoord = v.Enod[0].Pcoord
			jcoord = v.Enod[1].Pcoord
		} else {
			icoord = v.Enod[0].Dcoord
			jcoord = v.Enod[1].Dcoord
		}
		if v.IsLineElem() && (math.Min(icoord[0], jcoord[0]) <= x+eps && math.Max(icoord[0], jcoord[0]) >= x-eps) && (math.Min(icoord[1], jcoord[1]) <= y+eps && math.Max(icoord[1], jcoord[1]) >= y-eps) {
			dist := math.Abs(DotLine(icoord[0], icoord[1], jcoord[0], jcoord[1], x, y))
			if plen := math.Hypot(icoord[0]-jcoord[0], icoord[1]-jcoord[1]); plen > 1E-12 {
				dist /= plen
			}
			if dist < mindist {
				mindist = dist
				rtn = v
			}
		}
	}
	return rtn
}

func (frame *Frame) PickPlateElem(x, y float64) []*Elem {
	rtn := make(map[int]*Elem)
	var coords [][]float64
	for _, el := range frame.Elems {
		if el.IsHidden(frame.Show) {
			continue
		}
		coords = make([][]float64, el.Enods)
		if frame.Show.PlotState&PLOT_UNDEFORMED != 0 {
			for i := 0; i < el.Enods; i++ {
				coords[i] = el.Enod[i].Pcoord
			}
		} else {
			for i := 0; i < el.Enods; i++ {
				coords[i] = el.Enod[i].Dcoord
			}
		}
		if !el.IsLineElem() {
			add := true
			sign := 0
			for i := 0; i < el.Enods; i++ {
				var j int
				if i == el.Enods-1 {
					j = 0
				} else {
					j = i + 1
				}
				if DotLine(coords[i][0], coords[i][1], coords[j][0], coords[j][1], x, y) > 0 {
					sign++
				} else {
					sign--
				}
				if i+1 != abs(sign) {
					add = false
					break
				}
			}
			if add {
				rtn[el.Num] = el
			}
		}
	}
	return SortedElem(rtn, func(e *Elem) float64 { return e.DistFromProjection(frame.View) })
}

func (frame *Frame) PickAxis(x, y, eps float64) (*Axis, int) {
	if frame.LocalAxis == nil {
		return nil, -1
	}
	mindist := eps
	var rtn *Axis
	d := -1
	start, end := frame.LocalAxis.Project(frame.View)
	for i := 0; i < 3; i++ {
		if (math.Min(start[0], end[i][0]) <= x+eps && math.Max(start[0], end[i][0]) >= x-eps) && (math.Min(start[1], end[i][1]) <= y+eps && math.Max(start[1], end[i][1]) >= y-eps) {
			dist := math.Abs(DotLine(start[0], start[1], end[i][0], end[i][1], x, y))
			if plen := math.Hypot(start[0]-end[i][0], start[1]-end[i][1]); plen > 1E-12 {
				dist /= plen
			}
			if dist < mindist {
				mindist = dist
				rtn = frame.LocalAxis
				d = i
			}
		}
	}
	return rtn, d
}

func (frame *Frame) PickNode(x, y, eps float64) *Node {
	mindist := eps
	var rtn *Node
	var dist float64
	for _, v := range frame.Nodes {
		if v.IsHidden(frame.Show) {
			continue
		}
		if frame.Show.PlotState&PLOT_UNDEFORMED != 0 {
			dist = math.Hypot(x-v.Pcoord[0], y-v.Pcoord[1])
		} else {
			dist = math.Hypot(x-v.Dcoord[0], y-v.Dcoord[1])
		}
		if dist < mindist {
			mindist = dist
			rtn = v
		} else if dist == mindist {
			if rtn.DistFromProjection(frame.View) > v.DistFromProjection(frame.View) {
				rtn = v
			}
		}
	}
	if frame.Show.Kijun {
		for _, k := range frame.Kijuns {
			if k.IsHidden(frame.Show) {
				continue
			}
			for _, n := range [][]float64{k.Start, k.End} {
				pc := frame.View.ProjectCoord(n)
				dist := math.Hypot(x-pc[0], y-pc[1])
				if dist < mindist {
					mindist = dist
					rtn = NewNode()
					rtn.Coord = n
					rtn.Pcoord = pc
				}
			}
		}
	}
	return rtn
}

func (frame *Frame) BoundedArea(x, y float64, maxdepth int) ([]*Node, []*Elem, error) {
	var cand *Elem
	xmin := 1e6
	for _, el := range frame.Elems {
		if el.IsHidden(frame.Show) || !el.IsLineElem() {
			continue
		}
		if el.Enod[0].Pcoord[1] == el.Enod[1].Pcoord[1] {
			continue
		}
		if (el.Enod[0].Pcoord[1]-y)*(el.Enod[1].Pcoord[1]-y) < 0 {
			xval := el.Enod[0].Pcoord[0] + (el.Enod[1].Pcoord[0]-el.Enod[0].Pcoord[0])*((y-el.Enod[0].Pcoord[1])/(el.Enod[1].Pcoord[1]-el.Enod[0].Pcoord[1])) - x
			if xval > 0 && xval < xmin {
				cand = el
				xmin = xval
			}
		}
	}
	if cand == nil {
		return nil, nil, fmt.Errorf("no candidate")
	}
	origin := []float64{x, y}
	_, cw := ClockWise(origin, cand.Enod[0].Pcoord, cand.Enod[1].Pcoord)
	c := NewChain(frame, cand.Enod[1], cand, func(c *Chain, el *Elem) bool {
		return true
	}, func(c *Chain) bool {
		return c.Node().Num == cand.Enod[0].Num
	}, func(c *Chain) error {
		if c.Size() > maxdepth {
			return fmt.Errorf("too much recursion")
		}
		return nil
	}, func(c *Chain, els []*Elem) []*Elem {
		minangle := 1e6
		ind := 0
		for i, el := range els {
			angle, tmpcw := ClockWise(c.Node().Pcoord, origin, el.Otherside(c.Node()).Pcoord)
			angle = math.Abs(angle)
			if cw != tmpcw && angle < minangle {
				ind = i
				minangle = angle
			}
		}
		els[0], els[ind] = els[ind], els[0]
		return els
	})
	rtnns := []*Node{cand.Enod[0], cand.Enod[1]}
	rtnels := []*Elem{cand}
	for c.Next() {
		rtnns = append(rtnns, c.Node())
		rtnels = append(rtnels, c.Elem())
	}
	if err := c.Err(); err != nil {
		return nil, nil, err
	}
	return rtnns[:len(rtnns)-1], rtnels, nil
}

// direction: 0 -> origin=bottomleft, x=[1,0], y=[0,1]
//            1 -> origin=topleft,    x=[1,0], y=[0,-1]
func (view *View) Set(direction int) {
	a0 := view.Angle[0] * math.Pi / 180 // phi
	a1 := view.Angle[1] * math.Pi / 180 // theta
	c0 := math.Cos(a0)
	s0 := math.Sin(a0)
	c1 := math.Cos(a1)
	s1 := math.Sin(a1)
	if direction == 0 {
		view.Viewpoint[0][0] = c1 * c0
		view.Viewpoint[0][1] = s1 * c0
		view.Viewpoint[0][2] = s0
		view.Viewpoint[1][0] = -s1
		view.Viewpoint[1][1] = c1
		view.Viewpoint[1][2] = 0.0
		view.Viewpoint[2][0] = -c1 * s0
		view.Viewpoint[2][1] = -s1 * s0
		view.Viewpoint[2][2] = c0
	} else if direction == 1 {
		view.Viewpoint[0][0] = c1 * c0
		view.Viewpoint[0][1] = s1 * c0
		view.Viewpoint[0][2] = s0
		view.Viewpoint[1][0] = -s1
		view.Viewpoint[1][1] = c1
		view.Viewpoint[1][2] = 0.0
		view.Viewpoint[2][0] = c1 * s0
		view.Viewpoint[2][1] = s1 * s0
		view.Viewpoint[2][2] = -c0
	}
}

func (view *View) ProjectCoord(coord []float64) []float64 {
	rtn := make([]float64, 2)
	p := make([]float64, 3)
	pv := make([]float64, 3)
	pc := make([]float64, 2)
	for i := 0; i < 3; i++ {
		p[i] = coord[i] - view.Focus[i]
		pv[i] = view.Viewpoint[0][i]*view.Dists[0] - p[i]
	}
	for i := 0; i < 2; i++ {
		pc[i] = Dot(view.Viewpoint[i+1], p, 3)
	}
	if view.Perspective {
		vnai := Dot(view.Viewpoint[0], pv, 3)
		for i := 0; i < 2; i++ {
			rtn[i] = view.Gfact*view.Dists[1]*pc[i]/vnai + view.Center[i]
		}
	} else {
		for i := 0; i < 2; i++ {
			rtn[i] = view.Gfact*pc[i] + view.Center[i]
		}
	}
	return rtn
}

func (view *View) ProjectNode(node *Node) {
	p := make([]float64, 3)
	pv := make([]float64, 3)
	pc := make([]float64, 2)
	for i := 0; i < 3; i++ {
		p[i] = node.Coord[i] - view.Focus[i]
		pv[i] = view.Viewpoint[0][i]*view.Dists[0] - p[i]
	}
	for i := 0; i < 2; i++ {
		pc[i] = Dot(view.Viewpoint[i+1], p, 3)
	}
	if view.Perspective {
		vnai := Dot(view.Viewpoint[0], pv, 3)
		for i := 0; i < 2; i++ {
			node.Pcoord[i] = view.Gfact*view.Dists[1]*pc[i]/vnai + view.Center[i]
		}
	} else {
		for i := 0; i < 2; i++ {
			node.Pcoord[i] = view.Gfact*pc[i] + view.Center[i]
		}
	}
}

func (view *View) ProjectDeformation(node *Node, show *Show) {
	p := make([]float64, 3)
	pv := make([]float64, 3)
	pc := make([]float64, 2)
	for i := 0; i < 3; i++ {
		p[i] = (node.Coord[i] + show.Dfact*node.ReturnDisp(show.Period, i)) - view.Focus[i]
		pv[i] = view.Viewpoint[0][i]*view.Dists[0] - p[i]
	}
	for i := 0; i < 2; i++ {
		pc[i] = Dot(view.Viewpoint[i+1], p, 3)
	}
	if view.Perspective {
		vnai := Dot(view.Viewpoint[0], pv, 3)
		for i := 0; i < 2; i++ {
			node.Dcoord[i] = view.Gfact*view.Dists[1]*pc[i]/vnai + view.Center[i]
		}
	} else {
		for i := 0; i < 2; i++ {
			node.Dcoord[i] = view.Gfact*pc[i] + view.Center[i]
		}
	}
}

func WriteInp(fn string, view *View, ai *Aiparameter, wind *Windparameter, els []*Elem) error {
	var bnum, pnum, snum, inum, nnum, enum int
	elems := make([]*Elem, 0)
	for _, el := range els {
		if el == nil {
			continue
		}
		elems = append(elems, el)
		enum++
	}
	elems = elems[:enum]
	sort.Sort(ElemByNum{elems})
	// Bond
	var add bool
	bonds := make([]*Bond, 0)
	for _, el := range elems {
		for _, eb := range el.Bonds {
			if eb == nil {
				continue
			}
			add = true
			for _, b := range bonds {
				if eb == b {
					add = false
					break
				}
			}
			if add {
				bonds = append(bonds, eb)
				bnum++
			}
		}
	}
	bonds = bonds[:bnum]
	sort.Sort(BondByNum{bonds})
	// Sect
	sects := make([]*Sect, 0)
	for _, el := range elems {
		add = true
		for _, s := range sects {
			if el.Sect == s {
				add = false
				break
			}
		}
		if add {
			sects = append(sects, el.Sect)
			snum++
		}
	}
	sects = sects[:snum]
	sort.Sort(SectByNum{sects})
	// Prop
	props := make([]*Prop, 0)
	for _, sec := range sects {
		add = true
		for _, f := range sec.Figs {
			for _, p := range props {
				if f.Prop == p {
					add = false
					break
				}
			}
			if add {
				props = append(props, f.Prop)
				pnum++
			}
		}
	}
	props = props[:pnum]
	sort.Sort(PropByNum{props})
	// Node
	nodes := make([]*Node, 0)
	for _, el := range elems {
		for _, en := range el.Enod {
			add = true
			for _, n := range nodes {
				if en == n {
					add = false
					break
				}
			}
			if add {
				nodes = append(nodes, en)
				nnum++
			}
		}
	}
	nodes = nodes[:nnum]
	sort.Sort(NodeByNum{nodes})
	// Pile
	piles := make([]*Pile, 0)
	for _, n := range nodes {
		if n.Pile != nil {
			add = true
			for _, i := range piles {
				if n.Pile == i {
					add = false
					break
				}
			}
			if add {
				piles = append(piles, n.Pile)
				inum++
			}
		}
	}
	piles = piles[:inum]
	sort.Sort(PileByNum{piles})
	return writeinp(fn, "\"CREATED ORGAN FRAME.\"", view, ai, wind, bonds, props, sects, piles, nodes, elems, nil)
}

func writeinp(fn, title string, view *View, ai *Aiparameter, wind *Windparameter, bonds []*Bond, props []*Prop, sects []*Sect, piles []*Pile, nodes []*Node, elems []*Elem, chains []*Chain) error {
	var otp bytes.Buffer
	inum := len(piles)
	// Frame
	nelem := len(elems)
	for _, c := range chains {
		nelem += c.Size()
	}
	if len(bonds) == 0 {
		bonds = []*Bond{Pin}
	}
	otp.WriteString(fmt.Sprintf("%s\n", title))
	otp.WriteString(fmt.Sprintf("NNODE %d\n", len(nodes)))
	otp.WriteString(fmt.Sprintf("NELEM %d\n", nelem))
	otp.WriteString(fmt.Sprintf("NPROP %d\n", len(props)))
	otp.WriteString(fmt.Sprintf("NSECT %d\n", len(sects)))
	otp.WriteString(fmt.Sprintf("NBOND %d\n", len(bonds)))
	if inum >= 1 {
		otp.WriteString(fmt.Sprintf("NPILE %d\n", inum))
	}
	otp.WriteString("\n")
	otp.WriteString(fmt.Sprintf("BASE    %5.3f %5.3f\n", ai.Base[0], ai.Base[1]))
	otp.WriteString(fmt.Sprintf("LOCATE  %5.3f\n", ai.Locate))
	otp.WriteString(fmt.Sprintf("TFACT   %5.3f\n", ai.Tfact))
	otp.WriteString(fmt.Sprintf("GPERIOD %5.3f\n", ai.Gperiod))
	if ai.Nfloor > 0 {
		otp.WriteString(fmt.Sprintf("NFLOOR %d\n", ai.Nfloor))
		otp.WriteString("HEIGHT")
		for i := 0; i < ai.Nfloor+1; i++ {
			otp.WriteString(fmt.Sprintf(" %.3f", ai.Boundary[i]))
		}
		otp.WriteString("\n")
	}
	otp.WriteString("\n")
	otp.WriteString(fmt.Sprintf("ROUGHNESS %d\n", wind.Roughness))
	otp.WriteString(fmt.Sprintf("VELOCITY  %5.3f\n", wind.Velocity))
	otp.WriteString(fmt.Sprintf("WFACT     %5.3f\n", wind.Factor))
	otp.WriteString("\n")
	otp.WriteString(fmt.Sprintf("GFACT %.1f\n", view.Gfact))
	otp.WriteString(fmt.Sprintf("FOCUS %.1f %.1f %.1f\n", view.Focus[0], view.Focus[1], view.Focus[2]))
	otp.WriteString(fmt.Sprintf("ANGLE %.1f %.1f\n", view.Angle[0], view.Angle[1]))
	otp.WriteString(fmt.Sprintf("DISTS %.1f %.1f\n\n", view.Dists[0], view.Dists[1]))
	// Bond
	for _, b := range bonds {
		otp.WriteString(b.InpString())
	}
	otp.WriteString("\n")
	// Prop
	for _, p := range props {
		otp.WriteString(p.InpString())
	}
	otp.WriteString("\n")
	// Sect
	for _, sec := range sects {
		otp.WriteString(sec.InpString())
	}
	otp.WriteString("\n")
	// Pile
	if inum >= 1 {
		for _, i := range piles {
			otp.WriteString(i.InpString())
		}
		otp.WriteString("\n")
	}
	// Node
	for _, n := range nodes {
		otp.WriteString(n.InpString())
	}
	otp.WriteString("\n")
	// Elem
	for _, el := range elems {
		otp.WriteString(el.InpString())
	}
	// Chain
	for _, c := range chains {
		otp.WriteString("CHAIN {\n")
		for _, el := range c.Elems() {
			otp.WriteString(el.InpString())
		}
		otp.WriteString("}\n")
	}
	// Write
	w, err := os.Create(fn)
	defer w.Close()
	if err != nil {
		return err
	}
	otp = AddCR(otp)
	otp.WriteTo(w)
	return nil
}

func WriteOutput(fn string, p string, els []*Elem) error {
	var otp bytes.Buffer
	// Elem
	otp.WriteString("\n\n** FORCES OF MEMBER\n\n")
	otp.WriteString("  NO   KT NODE         N        Q1        Q2        MT        M1        M2\n\n")
	sort.Sort(ElemByNum{els})
	for _, el := range els {
		if !el.IsLineElem() {
			continue
		}
		otp.WriteString(el.OutputStress(p))
	}
	// Node
	var add bool
	ns := make([]*Node, 0)
	for _, el := range els {
		if el == nil {
			continue
		}
		for _, en := range el.Enod {
			add = true
			for _, n := range ns {
				if en == n {
					add = false
					break
				}
			}
			if add {
				ns = append(ns, en)
			}
		}
	}
	sort.Sort(NodeByNum{ns})
	otp.WriteString("\n\n** DISPLACEMENT OF NODE\n\n")
	otp.WriteString("  NO          U          V          W         KSI         ETA       OMEGA\n\n")
	for _, n := range ns {
		otp.WriteString(n.OutputDisp(p))
	}
	otp.WriteString("\n\n** REACTION\n\n")
	otp.WriteString("  NO  DIRECTION              R    NC\n\n")
	for _, n := range ns {
		for i := 0; i < 6; i++ {
			if n.Conf[i] {
				otp.WriteString(n.OutputReaction(p, i))
			}
		}
	}
	// Write
	w, err := os.Create(fn)
	defer w.Close()
	if err != nil {
		return err
	}
	otp = AddCR(otp)
	otp.WriteTo(w)
	return nil
}

func WriteReaction(fn string, ns []*Node, direction int, unit float64) error {
	var otp bytes.Buffer
	r := make([]float64, 3)
	otp.WriteString(" NODE   XCOORD   YCOORD   ZCOORD     WEIGHT       LONG      XSEIS      YSEIS        W+L      W+L+X      W+L-X      W+L+Y      W+L-Y PILE\n")
	for _, n := range ns {
		if n == nil {
			continue
		}
		if !n.Conf[direction] {
			continue
		}
		otp.WriteString(fmt.Sprintf(" %4d %8.3f %8.3f %8.3f", n.Num, n.Coord[0], n.Coord[1], n.Coord[2]))
		wgt := n.Weight[1] * unit
		for i, per := range []string{"L", "X", "Y"} {
			if rea, ok := n.Reaction[per]; ok {
				r[i] = rea[direction] * unit
			} else {
				r[i] = 0.0
			}
		}
		otp.WriteString(fmt.Sprintf(" %10.3f %10.3f %10.3f %10.3f %10.3f %10.3f %10.3f %10.3f %10.3f", wgt, r[0], r[1], r[2], wgt+r[0], wgt+r[0]+r[1], wgt+r[0]-r[1], wgt+r[0]+r[2], wgt+r[0]-r[2]))
		if n.Pile != nil {
			otp.WriteString(fmt.Sprintf(" %4d\n", n.Pile.Num))
		} else {
			otp.WriteString("\n")
		}
	}
	w, err := os.Create(fn)
	defer w.Close()
	if err != nil {
		return err
	}
	otp.WriteTo(w)
	return nil
}

func WriteFig2(fn string, view *View, show *Show) error {
	var otp bytes.Buffer
	if view.Perspective {
		otp.WriteString("perspective\n")
	} else {
		otp.WriteString("axonometric\n")
	}
	otp.WriteString(fmt.Sprintf("gfact %.1f\n", view.Gfact))
	otp.WriteString(fmt.Sprintf("gaxis %.1f\n", show.GlobalAxisSize))
	otp.WriteString(fmt.Sprintf("focus %.3f %.3f %.3f\n", view.Focus[0], view.Focus[1], view.Focus[2]))
	otp.WriteString(fmt.Sprintf("angle %.1f %.1f\n", view.Angle[0], view.Angle[1]))
	otp.WriteString(fmt.Sprintf("dists %.0f %.0f\n", view.Dists[0], view.Dists[1]))
	otp.WriteString("elem")
	for i, etname := range ETYPES {
		if i == NULL || i == TRUSS {
			continue
		}
		if show.Etype[i] {
			otp.WriteString(fmt.Sprintf(" %s", strings.ToLower(etname)))
		}
	}
	otp.WriteString("\n")
	if show.Conf {
		otp.WriteString(fmt.Sprintf("conf %.1f\n", show.ConfSize))
	}
	if show.Bond {
		otp.WriteString(fmt.Sprintf("hinge %.1f\n", show.BondSize))
	} else {
		otp.WriteString("hinge 0")
	}
	if show.PlotState&PLOT_DEFORMED != 0 {
		otp.WriteString(fmt.Sprintf("dfact %.1f\ndeformation %s\n", show.Dfact, strings.ToLower(show.Period)))
	}
	if show.ColorMode == ECOLOR_RATE {
		otp.WriteString("srcancolor\n")
	}
	if show.SrcanRate != 0 {
		// TODO: set srcanrate value
		otp.WriteString("srcanrate\n")
	}
	for i, etname := range ETYPES {
		if i == NULL || i == TRUSS {
			continue
		}
		if show.Stress[i] != 0 {
			for j, stname := range []string{"n", "qx", "qy", "mz", "mx", "my"} {
				if show.Stress[i]&(1<<uint(j)) != 0 {
					otp.WriteString(fmt.Sprintf("stress %s %s %s\n", strings.ToLower(etname), strings.ToLower(show.Period), stname))
				}
			}
		}
	}
	w, err := os.Create(fn)
	if err != nil {
		return err
	}
	defer w.Close()
	otp.WriteTo(w)
	return nil
}

func NodeDuplication(nodes map[int]*Node, eps float64) map[*Node]*Node {
	dups := make(map[*Node]*Node, 0)
	keys := make([]int, len(nodes))
	i := 0
	for _, k := range nodes {
		if k != nil {
			keys[i] = k.Num
			i++
		}
	}
	sort.Ints(keys)
	for j, k := range keys[:i] {
		if _, ok := dups[nodes[k]]; ok {
			continue
		}
	loop:
		for _, m := range keys[j+1 : i] {
			for n := 0; n < 3; n++ {
				if math.Abs(nodes[k].Coord[n]-nodes[m].Coord[n]) > eps {
					continue loop
				}
			}
			dups[nodes[m]] = nodes[k]
		}
	}
	return dups
}
