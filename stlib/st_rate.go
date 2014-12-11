package st

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	E_LAMBDA_B = 1.29099444874 // 1.0/math.Sqrt(6)
	P_LAMBDA_B = 0.3
	NCS        = 15.0
)

// Material
type Material interface {
}

type Steel struct {
	Name string
	F    float64
	Fu   float64
	E    float64
	Poi  float64
}

func (st Steel) Lambda() float64 {
	return math.Pi * math.Sqrt(st.E/(0.6*st.F))
}

type Concrete struct {
	Name string
	fc   float64
	E    float64
	Poi  float64
}

type SD struct {
	Name string
	Fl   float64
	Fs   float64
	E    float64
	Poi  float64
}

type Reinforce struct {
	Area     float64
	Position []float64
	Material SD
}

func NewReinforce(sd SD) Reinforce {
	return Reinforce{0.0, []float64{0.0, 0.0}, sd}
}
func (rf Reinforce) Ft(cond *Condition) float64 {
	switch cond.Period {
	default:
		return 0.0
	case "L":
		f := rf.Material.Fl
		if rf.Area > 6.158 { // D>=29
			if f > 2.0 {
				return 2.0
			} else {
				return f
			}
		} else {
			return f
		}
	case "X", "Y", "S":
		return rf.Material.Fs
	}
}
func (rf Reinforce) Ftw(cond *Condition) float64 {
	switch cond.Period {
	default:
		return 0.0
	case "L":
		return 2.0
	case "X", "Y", "S":
		return rf.Material.Fs
	}
}
func (rf Reinforce) Vertices() [][]float64 {
	d := 0.5 * math.Sqrt(rf.Area*4.0/math.Pi)
	val := math.Pi/8.0
	theta := 0.0
	vertices := make([][]float64, 16)
	for i:=0; i<16; i++ {
		c := math.Cos(theta)
		s := math.Sin(theta)
		vertices[i] = []float64{d*c + rf.Position[0], d*s + rf.Position[1]}
		theta += val
	}
	return vertices
}

type Wood struct {
	Name float64
	Fc   float64
	Ft   float64
	Fb   float64
	Fs   float64
	E    float64
	Poi  float64
}

var (
	SN400    = Steel{"SN400", 2.4, 4.0, 2100.0, 0.3}
	SN490    = Steel{"SN490", 3.3, 5.0, 2100.0, 0.3}
	SN400T40 = Steel{"SN400T40", 2.2, 4.0, 2100.0, 0.3}
	SN490T40 = Steel{"SN490T40", 3.0, 5.0, 2100.0, 0.3}

	FC24 = Concrete{"Fc24", 0.240, 210.0, 0.166666}
	FC36 = Concrete{"Fc36", 0.360, 210.0, 0.166666}

	SD295 = SD{"SD295", 2.0, 3.0, 2100.0, 0.3}
	SD345 = SD{"SD345", 2.2, 3.5, 2100.0, 0.3}
)

// Section
type SectionRate interface {
	Num()    int
	TypeString() string
	Snapshot() SectionRate
	String() string
	SetName(string)
	SetValue(string, []float64)
	Factor(string) float64
	Na(*Condition) float64
	Qa(*Condition) float64
	Ma(*Condition) float64
	Mza(*Condition) float64
}

type Shape interface {
	String() string
	Description() string
	A() float64
	Asx() float64
	Asy() float64
	Ix() float64
	Iy() float64
	J() float64
	Iw() float64
	Torsion() float64
	Zx() float64
	Zy() float64
	Vertices() [][]float64
}

// S COLUMN// {{{
type SColumn struct {
	Steel
	Shape
	num      int
	Etype    string
	Name     string
	XFace    []float64
	YFace    []float64
	BBLength []float64
	BTLength []float64
	BBFactor []float64
	BTFactor []float64
}

func NewSColumn(num int, shape Shape, material Steel) *SColumn {
	sc := &SColumn{material, shape, num, "COLUMN", "", nil, nil, nil, nil, nil, nil}
	return sc
}
func (sc *SColumn) Num() int {
	return sc.num
}
func (sc *SColumn) TypeString() string {
	return "Ｓ柱"
}
func (sc *SColumn) Snapshot() SectionRate {
	s := NewSColumn(sc.num, sc.Shape, sc.Steel)
	s.Etype = sc.Etype
	s.Name = sc.Name
	s.XFace = make([]float64, 2)
	s.YFace = make([]float64, 2)
	s.BBLength = make([]float64, 2)
	s.BTLength = make([]float64, 2)
	s.BBFactor = make([]float64, 2)
	s.BTFactor = make([]float64, 2)
	for i:=0; i<2; i++ {
		s.XFace[i] = sc.XFace[i]
		s.YFace[i] = sc.YFace[i]
		s.BBLength[i] = sc.BBLength[i]
		s.BTLength[i] = sc.BTLength[i]
		s.BBFactor[i] = sc.BBFactor[i]
		s.BTFactor[i] = sc.BTFactor[i]
	}
	return s
}
func (sc *SColumn) SetName(name string) {
	sc.Name = name
}
func (sc *SColumn) SetValue(name string, vals []float64) {
	switch name {
	case "XFACE":
		sc.XFace = vals
	case "YFACE":
		sc.YFace = vals
	case "BBLEN":
		sc.BBLength = vals
	case "BTLEN":
		sc.BTLength = vals
	case "BBFAC":
		sc.BBFactor = vals
	case "BTFAC":
		sc.BTFactor = vals
	}
}
func (sc *SColumn) String() string {
	var rtn bytes.Buffer
	rtn.WriteString(fmt.Sprintf("CODE %3d S %s %57s\n", sc.num, sc.Etype, fmt.Sprintf("\"%s\"", sc.Name)))
	line2 := fmt.Sprintf("         %%-29s %%s %%%ds\n", 35-len(sc.Steel.Name))
	rtn.WriteString(fmt.Sprintf(line2, sc.Shape.String(), sc.Steel.Name, fmt.Sprintf("\"%s\"", sc.Shape.Description())))
	if sc.XFace != nil {
		rtn.WriteString(fmt.Sprintf("         XFACE %5.1f %5.1f %48s\n", sc.XFace[0], sc.XFace[1], fmt.Sprintf("\"FACE LENGTH Mx:HEAD= %.0f,TAIL= %.0f[cm]\"", sc.XFace[0], sc.XFace[1])))
	} else {
		rtn.WriteString("         XFACE   0.0   0.0             \"FACE LENGTH Mx:HEAD= 0,TAIL= 0[cm]\"\n")
	}
	if sc.YFace != nil {
		rtn.WriteString(fmt.Sprintf("         YFACE %5.1f %5.1f %48s\n", sc.YFace[0], sc.YFace[1], fmt.Sprintf("\"FACE LENGTH My:HEAD= %.0f,TAIL= %.0f[cm]\"", sc.XFace[0], sc.XFace[1])))
	} else {
		rtn.WriteString("         YFACE   0.0   0.0             \"FACE LENGTH My:HEAD= 0,TAIL= 0[cm]\"\n")
	}
	if sc.BBLength != nil {
		rtn.WriteString(fmt.Sprintf("         BBLEN %5.1f %5.1f\n", sc.BBLength[0], sc.BBLength[1]))
	} else if sc.BBFactor != nil {
		rtn.WriteString(fmt.Sprintf("         BBFAC %5.1f %5.1f\n", sc.BBFactor[0], sc.BBFactor[1]))
	}
	if sc.BTLength != nil {
		rtn.WriteString(fmt.Sprintf("         BTLEN %5.1f %5.1f\n", sc.BTLength[0], sc.BTLength[1]))
	} else if sc.BTFactor != nil {
		rtn.WriteString(fmt.Sprintf("         BTFAC %5.1f %5.1f\n", sc.BTFactor[0], sc.BTFactor[1]))
	}
	return rtn.String()
}
func (sc *SColumn) Factor(p string) float64 {
	switch p {
	default:
		return 0.0
	case "L":
		return 1.0
	case "X", "Y", "S":
		return 1.5
	}
}
func (sc *SColumn) Lk(length float64, strong bool) float64 {
	var ind int
	if strong {
		ind = 0
	} else {
		ind = 1
	}
	if sc.BBLength != nil && sc.BBLength[ind] > 0.0 {
		return sc.BBLength[ind]
	} else if sc.BBFactor != nil && sc.BBFactor[ind] > 0.0 {
		return length * sc.BBFactor[ind]
	} else {
		return length
	}
}
func (sc *SColumn) Lb(length float64, strong bool) float64 {
	var ind int
	if strong {
		ind = 0
	} else {
		ind = 1
	}
	if sc.BTLength != nil && sc.BTLength[ind] > 0.0 {
		return sc.BTLength[ind]
	} else if sc.BTFactor != nil && sc.BTFactor[ind] > 0.0 {
		return length * sc.BTFactor[ind]
	} else {
		return length
	}
}
func (sc *SColumn) Fc(cond *Condition) float64 {
	var rtn float64
	var lambda float64
	lx := sc.Lk(cond.Length, true)
	ly := sc.Lk(cond.Length, false)
	lambda_x := lx / math.Sqrt(sc.Ix()/sc.A())
	lambda_y := ly / math.Sqrt(sc.Iy()/sc.A())
	if lambda_x >= lambda_y {
		lambda = lambda_x
	} else {
		lambda = lambda_y
	}
	val := lambda / sc.Lambda()
	if val <= 1.0 {
		nu := 1.5 + 2.0*val*val/3.0
		rtn = (1.0 - 0.4*val*val) * sc.F / nu
	} else {
		rtn = 0.277 * sc.F / (val * val)
	}
	return rtn * sc.Factor(cond.Period)
}
func (sc *SColumn) Ft(cond *Condition) float64 {
	return sc.F / 1.5 * sc.Factor(cond.Period)
}
func (sc *SColumn) Fs(cond *Condition) float64 {
	return sc.F / (1.5 * math.Sqrt(3)) * sc.Factor(cond.Period)
}
func (sc *SColumn) Fb(cond *Condition) float64 {
	l := sc.Lb(cond.Length, cond.Strong)
	var rtn float64
	fbnew := func() float64 {
		me := sc.Me(l, 1.0)
		my := sc.My(cond)
		lambda_b := math.Sqrt(my / me)
		nu := 1.5 + math.Pow(lambda_b/E_LAMBDA_B, 2.0)/1.5
		if lambda_b <= P_LAMBDA_B {
			return sc.F / nu
		} else if lambda_b <= E_LAMBDA_B {
			return (1.0 - 0.4*(lambda_b-P_LAMBDA_B)/(E_LAMBDA_B-P_LAMBDA_B)) * sc.F / nu
		} else {
			return sc.F / math.Pow(lambda_b, 2.0) / 2.17
		}
	}
	if cond.Strong {
		if hk, ok := sc.Shape.(HKYOU); ok {
			if cond.FbOld {
				rtn = 900.0 / (l * hk.H / (hk.B * hk.Tf))
			} else {
				rtn = fbnew()
			}
		} else {
			rtn = sc.F / 1.5
		}
	} else {
		if hw, ok := sc.Shape.(HWEAK); ok {
			if cond.FbOld {
				rtn = 900.0 / (l * hw.H / (hw.B * hw.Tf))
			} else {
				rtn = fbnew()
			}
		} else {
			rtn = sc.F / 1.5
		}
	}
	return rtn * sc.Factor(cond.Period)
}
func (sc *SColumn) Na(cond *Condition) float64 {
	if cond.Compression {
		return sc.Fc(cond) * sc.A()
	} else {
		return sc.Ft(cond) * sc.A()
	}
}
func (sc *SColumn) Qa(cond *Condition) float64 {
	f := sc.Fs(cond)
	if cond.Strong {
		return f * sc.Asx()
	} else {
		return f * sc.Asy()
	}
}
func (sc *SColumn) Me(length, Cb float64) float64 {
	g := sc.E / (2.0 * (1.0 + sc.Poi))
	var I float64
	Ix := sc.Ix()
	Iy := sc.Iy()
	if Ix >= Iy {
		I = Iy
	} else {
		I = Ix
	}
	return Cb * math.Sqrt(math.Pow(math.Pi, 4.0)*sc.E*I*sc.E*sc.Iw()/math.Pow(length, 4.0)+math.Pow(math.Pi, 2.0)*sc.E*I*g*sc.J()/math.Pow(length, 2.0)) * 0.01 // [tfm]
}
func (sc *SColumn) My(cond *Condition) float64 {
	if cond.Strong {
		return sc.F * sc.Zx() * 0.01 // [tfm]
	} else {
		return sc.F * sc.Zy() * 0.01 // [tfm]
	}
}
func (sc *SColumn) Ma(cond *Condition) float64 {
	f := sc.Fb(cond)
	if cond.Strong {
		return f * sc.Zx() * 0.01 // [tfm]
	} else {
		return f * sc.Zy() * 0.01 // [tfm]
	}
}
func (sc *SColumn) Mza(cond *Condition) float64 {
	return sc.Fs(cond) * sc.Torsion() * 0.01 // [tfm]
}

func (sc *SColumn) Vertices() [][]float64 {
	return sc.Shape.Vertices()
}

// }}}

// HKYOU// {{{
type HKYOU struct {
	H, B, Tw, Tf float64
}

func NewHKYOU(lis []string) (HKYOU, error) {
	hk := HKYOU{0.0, 0.0, 0.0, 0.0}
	var val float64
	var err error
	val, err = strconv.ParseFloat(lis[0], 64)
	if err != nil {
		return hk, err
	}
	hk.H = val
	val, err = strconv.ParseFloat(lis[1], 64)
	if err != nil {
		return hk, err
	}
	hk.B = val
	val, err = strconv.ParseFloat(lis[2], 64)
	if err != nil {
		return hk, err
	}
	hk.Tw = val
	val, err = strconv.ParseFloat(lis[3], 64)
	if err != nil {
		return hk, err
	}
	hk.Tf = val
	return hk, nil
}
func (hk HKYOU) String() string {
	return fmt.Sprintf("HKYOU %5.1f %5.1f %4.1f %4.1f", hk.H, hk.B, hk.Tw, hk.Tf)
}
func (hk HKYOU) Description() string {
	return fmt.Sprintf("H-%dx%dx%dx%d(KYOU)[mm]", int(hk.H*10), int(hk.B*10), int(hk.Tw*10), int(hk.Tf*10))
}
func (hk HKYOU) A() float64 {
	return hk.H*hk.B - (hk.H-2*hk.Tf)*(hk.B-hk.Tw)
}
func (hk HKYOU) Asx() float64 {
	return 2.0 * hk.B * hk.Tf / 1.5
}
func (hk HKYOU) Asy() float64 {
	return (hk.H - 2*hk.Tf) * hk.Tw
}
func (hk HKYOU) Ix() float64 {
	return (hk.B*math.Pow(hk.H, 3.0) - (hk.B-hk.Tw)*math.Pow(hk.H-2*hk.Tf, 3.0)) / 12.0
}
func (hk HKYOU) Iy() float64 {
	return 2.0*hk.Tf*math.Pow(hk.B, 3.0)/12.0 + (hk.H-2*hk.Tf)*math.Pow(hk.Tw, 3.0)/12.0
}
func (hk HKYOU) J() float64 {
	return 2.0*hk.B*math.Pow(hk.Tf, 3.0)/3.0 + (hk.H-2*hk.Tf)*math.Pow(hk.Tw, 3.0)/3.0
}
func (hk HKYOU) Iw() float64 {
	return math.Pow(hk.H, 2.0) * math.Pow(hk.B, 3.0) * hk.Tf / 24.0
}
func (hk HKYOU) Torsion() float64 {
	if hk.Tf >= hk.Tw {
		return hk.J() / hk.Tf
	} else {
		return hk.J() / hk.Tw
	}
}
func (hk HKYOU) Zx() float64 {
	return hk.Ix() / hk.H * 2.0
}
func (hk HKYOU) Zy() float64 {
	return hk.Iy() / hk.B * 2.0
}

func (hk HKYOU) Vertices() [][]float64 {
	h := hk.H * 0.5
	b := hk.B * 0.5
	w := hk.Tw * 0.5
	f := hk.Tf
	vertices := make([][]float64, 12)
	vertices[0] = []float64{-b, -h}
	vertices[1] = []float64{b, -h}
	vertices[2] = []float64{b, -(h-f)}
	vertices[3] = []float64{w, -(h-f)}
	vertices[4] = []float64{w, h-f}
	vertices[5] = []float64{b, h-f}
	vertices[6] = []float64{b, h}
	vertices[7] = []float64{-b, h}
	vertices[8] = []float64{-b, h-f}
	vertices[9] = []float64{-w, h-f}
	vertices[10] = []float64{-w, -(h-f)}
	vertices[11] = []float64{-b, -(h-f)}
	return vertices
}

// }}}

// HWEAK// {{{
type HWEAK struct {
	H, B, Tw, Tf float64
}

func NewHWEAK(lis []string) (HWEAK, error) {
	hw := HWEAK{0.0, 0.0, 0.0, 0.0}
	var val float64
	var err error
	val, err = strconv.ParseFloat(lis[0], 64)
	if err != nil {
		return hw, err
	}
	hw.H = val
	val, err = strconv.ParseFloat(lis[1], 64)
	if err != nil {
		return hw, err
	}
	hw.B = val
	val, err = strconv.ParseFloat(lis[2], 64)
	if err != nil {
		return hw, err
	}
	hw.Tw = val
	val, err = strconv.ParseFloat(lis[3], 64)
	if err != nil {
		return hw, err
	}
	hw.Tf = val
	return hw, nil
}
func (hw HWEAK) String() string {
	return fmt.Sprintf("HWEAK %5.1f %5.1f %4.1f %4.1f", hw.H, hw.B, hw.Tw, hw.Tf)
}
func (hw HWEAK) Description() string {
	return fmt.Sprintf("H-%dx%dx%dx%d(WEAK)[mm]", int(hw.H*10), int(hw.B*10), int(hw.Tw*10), int(hw.Tf*10))
}
func (hw HWEAK) A() float64 {
	return hw.H*hw.B - (hw.H-2*hw.Tf)*(hw.B-hw.Tw)
}
func (hw HWEAK) Asx() float64 {
	return (hw.H - 2*hw.Tf) * hw.Tw
}
func (hw HWEAK) Asy() float64 {
	return 2.0 * hw.B * hw.Tf / 1.5
}
func (hw HWEAK) Ix() float64 {
	return 2.0*hw.Tf*math.Pow(hw.B, 3.0)/12.0 + (hw.H-2*hw.Tf)*math.Pow(hw.Tw, 3.0)/12.0
}
func (hw HWEAK) Iy() float64 {
	return (hw.B*math.Pow(hw.H, 3.0) - (hw.B-hw.Tw)*math.Pow(hw.H-2*hw.Tf, 3.0)) / 12.0
}
func (hw HWEAK) J() float64 {
	return 2.0*hw.B*math.Pow(hw.Tf, 3.0)/3.0 + (hw.H-2*hw.Tf)*math.Pow(hw.Tw, 3.0)/3.0
}
func (hw HWEAK) Iw() float64 {
	return 0.0
}
func (hw HWEAK) Torsion() float64 {
	if hw.Tf >= hw.Tw {
		return hw.J() / hw.Tf
	} else {
		return hw.J() / hw.Tw
	}
}
func (hw HWEAK) Zx() float64 {
	return hw.Ix() / hw.B * 2.0
}
func (hw HWEAK) Zy() float64 {
	return hw.Iy() / hw.H * 2.0
}

func (hw HWEAK) Vertices() [][]float64 {
	h := hw.H * 0.5
	b := hw.B * 0.5
	w := hw.Tw * 0.5
	f := hw.Tf
	vertices := make([][]float64, 12)
	vertices[0] = []float64{-h, -b}
	vertices[1] = []float64{-h, b}
	vertices[2] = []float64{-(h-f), b}
	vertices[3] = []float64{-(h-f), w}
	vertices[4] = []float64{h-f, w}
	vertices[5] = []float64{h-f, b}
	vertices[6] = []float64{h, b}
	vertices[7] = []float64{h, -b}
	vertices[8] = []float64{h-f, -b}
	vertices[9] = []float64{h-f, -w}
	vertices[10] = []float64{-(h-f), -w}
	vertices[11] = []float64{-(h-f), -b}
	return vertices
}

// }}}

// RPIPE// {{{
type RPIPE struct {
	H, B, Tw, Tf float64
}

func NewRPIPE(lis []string) (RPIPE, error) {
	rp := RPIPE{0.0, 0.0, 0.0, 0.0}
	var val float64
	var err error
	val, err = strconv.ParseFloat(lis[0], 64)
	if err != nil {
		return rp, err
	}
	rp.H = val
	val, err = strconv.ParseFloat(lis[1], 64)
	if err != nil {
		return rp, err
	}
	rp.B = val
	val, err = strconv.ParseFloat(lis[2], 64)
	if err != nil {
		return rp, err
	}
	rp.Tw = val
	val, err = strconv.ParseFloat(lis[3], 64)
	if err != nil {
		return rp, err
	}
	rp.Tf = val
	return rp, nil
}
func (rp RPIPE) String() string {
	return fmt.Sprintf("RPIPE %5.1f %5.1f %4.1f %4.1f", rp.H, rp.B, rp.Tw, rp.Tf)
}
func (rp RPIPE) Description() string {
	return fmt.Sprintf("BOX-%dx%dx%dx%d[mm]", int(rp.H*10), int(rp.B*10), int(rp.Tw*10), int(rp.Tf*10))
}
func (rp RPIPE) A() float64 {
	return rp.H*rp.B - (rp.H-2*rp.Tf)*(rp.B-2*rp.Tw)
}
func (rp RPIPE) Asx() float64 {
	return 2.0 * (rp.B - 2*rp.Tw) * rp.Tf
}
func (rp RPIPE) Asy() float64 {
	return 2.0 * (rp.H - 2*rp.Tf) * rp.Tw
}
func (rp RPIPE) Ix() float64 {
	return (rp.B*math.Pow(rp.H, 3.0) - (rp.B-2*rp.Tw)*math.Pow(rp.H-2*rp.Tf, 3.0)) / 12.0
}
func (rp RPIPE) Iy() float64 {
	return (rp.H*math.Pow(rp.B, 3.0) - (rp.H-2*rp.Tf)*math.Pow(rp.B-2*rp.Tw, 3.0)) / 12.0
}
func (rp RPIPE) J() float64 {
	return 4.0 * math.Pow((rp.H-rp.Tf)*(rp.B-rp.Tw), 2.0) / (2.0 * ((rp.B-rp.Tw)/rp.Tf + (rp.H-rp.Tf)/rp.Tw))
}
func (rp RPIPE) Iw() float64 {
	return 0.0
}
func (rp RPIPE) Torsion() float64 {
	if rp.Tf >= rp.Tw {
		return 2.0 * (rp.B - rp.Tw) * (rp.H - rp.Tf) * rp.Tw
	} else {
		return 2.0 * (rp.B - rp.Tw) * (rp.H - rp.Tf) * rp.Tf
	}
}
func (rp RPIPE) Zx() float64 {
	return rp.Ix() / rp.H * 2.0
}
func (rp RPIPE) Zy() float64 {
	return rp.Iy() / rp.B * 2.0
}

func (rp RPIPE) Vertices() [][]float64 {
	h := rp.H * 0.5
	b := rp.B * 0.5
	hw := h - rp.Tw
	bf := b - rp.Tf
	vertices := make([][]float64, 9)
	vertices[0] = []float64{-b, -h}
	vertices[1] = []float64{b, -h}
	vertices[2] = []float64{b, h}
	vertices[3] = []float64{-b, h}
	vertices[4] = nil
	vertices[5] = []float64{-bf, -hw}
	vertices[6] = []float64{bf, -hw}
	vertices[7] = []float64{bf, hw}
	vertices[8] = []float64{-bf, hw}
	return vertices
}

// }}}

// CPIPE// {{{
type CPIPE struct {
	D, T float64
}

func NewCPIPE(lis []string) (CPIPE, error) {
	cp := CPIPE{0.0, 0.0}
	var val float64
	var err error
	val, err = strconv.ParseFloat(lis[0], 64)
	if err != nil {
		return cp, err
	}
	cp.D = val
	val, err = strconv.ParseFloat(lis[1], 64)
	if err != nil {
		return cp, err
	}
	cp.T = val
	return cp, nil
}
func (cp CPIPE) String() string {
	return fmt.Sprintf("CPIPE %5.1f %5.1f", cp.D, cp.T)
}
func (cp CPIPE) Description() string {
	return fmt.Sprintf("PIPE-%dx%d[mm]", int(cp.D*10), int(cp.T*10))
}
func (cp CPIPE) A() float64 {
	return 0.25 * math.Pi * (math.Pow(cp.D, 2.0) - math.Pow(cp.D-2*cp.T, 2.0))
}
func (cp CPIPE) Asx() float64 {
	return 0.5 * cp.A()
}
func (cp CPIPE) Asy() float64 {
	return 0.5 * cp.A()
}
func (cp CPIPE) Ix() float64 {
	return 0.015625 * math.Pi * (math.Pow(cp.D, 4.0) - math.Pow(cp.D-2*cp.T, 4.0))
}
func (cp CPIPE) Iy() float64 {
	return 0.015625 * math.Pi * (math.Pow(cp.D, 4.0) - math.Pow(cp.D-2*cp.T, 4.0))
}
func (cp CPIPE) J() float64 {
	return 4.0 * math.Pow(0.25*math.Pi*math.Pow(cp.D-cp.T, 2.0), 2.0) * cp.T / (math.Pi * (cp.D - cp.T))
}
func (cp CPIPE) Iw() float64 {
	return 0.0
}
func (cp CPIPE) Torsion() float64 {
	return 2.0 * 0.25 * math.Pi * math.Pow(cp.D-cp.T, 2.0) * cp.T
}
func (cp CPIPE) Zx() float64 {
	return cp.Ix() / cp.D * 2.0
}
func (cp CPIPE) Zy() float64 {
	return cp.Iy() / cp.D * 2.0
}

func (cp CPIPE) Vertices() [][]float64 {
	d := 0.5 * cp.D
	dt := d - cp.T
	val := math.Pi/8.0
	theta := 0.0
	vertices := make([][]float64, 33)
	for i:=0; i<16; i++ {
		c := math.Cos(theta)
		s := math.Sin(theta)
		vertices[i] = []float64{d*c, d*s}
		vertices[i+17] = []float64{dt*c, dt*s}
		theta += val
	}
	vertices[16] = nil
	return vertices
}
// }}}

// S GIRDER// {{{
type SGirder struct {
	SColumn
}

func NewSGirder(num int, shape Shape, material Steel) *SGirder {
	sc := NewSColumn(num, shape, material)
	sc.Etype = "GIRDER"
	return &SGirder{*sc}
}

// }}}

type CShape interface {
	String() string
	Bound(int) float64
	Breadth(bool) float64
	Area() float64
	Height(bool) float64
	Vertices() [][]float64
}

// TODO: implement CCircle

// CRect// {{{
type CRect struct {
	Left, Lower, Right, Upper float64
}

func NewCRect(b []float64) CRect {
	return CRect{b[0], b[1], b[2], b[3]}
}
func (cr CRect) String() string {
	return fmt.Sprintf("CRECT %6.1f %6.1f %6.1f %6.1f", cr.Left, cr.Lower, cr.Right, cr.Upper)
}
func (cr CRect) Bound(side int) float64 {
	switch side {
	case 0:
		return cr.Left
	case 1:
		return cr.Lower
	case 2:
		return cr.Right
	case 3:
		return cr.Upper
	}
	return 0.0
}
func (cr CRect) Breadth(strong bool) float64 {
	if strong {
		return cr.Right - cr.Left
	} else {
		return cr.Upper - cr.Lower
	}
}
func (cr CRect) Height(strong bool) float64 {
	if strong {
		return cr.Upper - cr.Lower
	} else {
		return cr.Right - cr.Left
	}
}
func (cr CRect) Area() float64 {
	return cr.Breadth(true) * cr.Height(true)
}

func (cr CRect) Vertices() [][]float64 {
	vertices := make([][]float64, 4)
	vertices[0] = []float64{cr.Left, cr.Lower}
	vertices[1] = []float64{cr.Right, cr.Lower}
	vertices[2] = []float64{cr.Right, cr.Upper}
	vertices[3] = []float64{cr.Left, cr.Upper}
	return vertices
}

// }}}

// RCColumn
type RCColumn struct {
	Concrete
	CShape
	num    int
	Name   string
	Nreins int
	Reins  []Reinforce
	Hoops  Hoop
	XFace  []float64
	YFace  []float64
}
type Hoop struct {
	Ps       []float64
	Name     string
	Material SD
}
func (hp Hoop) Ftw(cond *Condition) float64 {
	switch cond.Period {
	default:
		return 0.0
	case "L":
		return 2.0
	case "X", "Y", "S":
		return hp.Material.Fs
	}
}

func NewRCColumn(num int) *RCColumn {
	rc := new(RCColumn)
	rc.num = num
	rc.Nreins = 0
	rc.Reins = make([]Reinforce, 0)
	rc.XFace = make([]float64, 2)
	rc.YFace = make([]float64, 2)
	return rc
}
func (rc *RCColumn) AddReins(lis []string) error {
	rf := NewReinforce(SetSD(lis[3]))
	val, err := strconv.ParseFloat(lis[0], 64)
	if err != nil {
		return err
	}
	rf.Area = val
	for i := 0; i < 2; i++ {
		val, err = strconv.ParseFloat(lis[1+i], 64)
		if err != nil {
			return err
		}
		rf.Position[i] = val
	}
	rc.Nreins++
	rc.Reins = append(rc.Reins, rf)
	return nil
}
func (rc *RCColumn) SetHoops(lis []string) error {
	ps := make([]float64, 2)
	for i := 0; i < 2; i++ {
		val, err := strconv.ParseFloat(lis[i], 64)
		if err != nil {
			return err
		}
		ps[i] = val
	}
	rc.Hoops = Hoop{ps, strings.Join(lis[3:], " "), SetSD(lis[2])}
	return nil
}
func (rc *RCColumn) SetConcrete(lis []string) error {
	switch lis[0] {
	case "CRECT":
		vals := make([]float64, 4)
		for i := 0; i < 4; i++ {
			val, err := strconv.ParseFloat(lis[1+i], 64)
			if err != nil {
				return err
			}
			vals[i] = val
		}
		rc.CShape = NewCRect(vals)
	}
	switch lis[5] {
	case "FC24":
		rc.Concrete = FC24
	case "FC36":
		rc.Concrete = FC36
	}
	return nil
}
func (rc *RCColumn) String() string {
	return ""
}
func (rc *RCColumn) Num() int {
	return rc.num
}
func (rc *RCColumn) TypeString() string {
	return "ＲＣ柱"
}
func (rc *RCColumn) Snapshot() SectionRate {
	r := NewRCColumn(rc.num)
	r.Concrete = rc.Concrete
	r.CShape = rc.CShape
	r.Name = rc.Name
	r.Nreins = rc.Nreins
	r.Reins = make([]Reinforce, r.Nreins)
	for i, rf := range rc.Reins {
		r.Reins[i] = rf
	}
	r.Hoops = rc.Hoops
	for i:=0; i<2; i++ {
		r.XFace[i] = rc.XFace[i]
		r.YFace[i] = rc.YFace[i]
	}
	return r
}
func (rc *RCColumn) SetName(name string) {
}
func (rc *RCColumn) SetValue(name string, vals []float64) {
	switch name {
	case "XFACE":
		rc.XFace = vals
	case "YFACE":
		rc.YFace = vals
	}
}
func (rc *RCColumn) Factor(p string) float64 {
	switch p {
	default:
		return 0.0
	case "L":
		return 1.0
	case "X", "Y", "S":
		return 2.0
	}
}
func (rc *RCColumn) Fs(cond *Condition) float64 {
	var rtn float64
	f1 := rc.fc / 30.0
	f2 := 0.005 + rc.fc/100.0
	if f1 <= f2 {
		rtn = f1
	} else {
		rtn = f2
	}
	switch cond.Period {
	default:
		rtn = 0.0
	case "L":
		rtn *= 1.0
	case "X", "Y", "S":
		rtn *= 1.5
	}
	return rtn
}
func (rc *RCColumn) Fc(cond *Condition) float64 {
	return rc.fc / 3.0 * rc.Factor(cond.Period)
}
func (rc *RCColumn) Ai() float64 {
	if rc.Reins == nil {
		return 0.0
	}
	rtn := 0.0
	for _, r := range rc.Reins {
		rtn += r.Area
	}
	return rtn
}
func (rc *RCColumn) LiAi(cond *Condition) float64 {
	if rc.Reins == nil {
		return 0.0
	}
	rtn := 0.0
	for _, r := range rc.Reins {
		if cond.Strong {
			if cond.Positive {
				rtn += r.Area * (rc.Bound(3) - r.Position[1])
			} else {
				rtn += r.Area * (r.Position[1] - rc.Bound(1))
			}
		} else {
			if cond.Positive {
				rtn += r.Area * (rc.Bound(2) - r.Position[0])
			} else {
				rtn += r.Area * (r.Position[0] - rc.Bound(0))
			}
		}
	}
	return rtn
}
func (rc *RCColumn) Li2Ai(cond *Condition) float64 {
	if rc.Reins == nil {
		return 0.0
	}
	rtn := 0.0
	for _, r := range rc.Reins {
		if cond.Strong {
			if cond.Positive {
				rtn += r.Area * math.Pow(rc.Bound(3)-r.Position[1], 2.0)
			} else {
				rtn += r.Area * math.Pow(r.Position[1]-rc.Bound(1), 2.0)
			}
		} else {
			if cond.Positive {
				rtn += r.Area * math.Pow(rc.Bound(2)-r.Position[0], 2.0)
			} else {
				rtn += r.Area * math.Pow(r.Position[0]-rc.Bound(0), 2.0)
			}
		}
	}
	return rtn
}
func (rc *RCColumn) FarSideReins(cond *Condition) float64 {
	if rc.Reins == nil {
		return 0.0
	}
	rtn := 0.0
	var tmp float64
	for _, r := range rc.Reins {
		if cond.Strong {
			if cond.Positive {
				tmp = rc.Bound(3) - r.Position[1]
			} else {
				tmp = r.Position[1] - rc.Bound(1)
			}
		} else {
			if cond.Positive {
				tmp = rc.Bound(2) - r.Position[0]
			} else {
				tmp = r.Position[0] - rc.Bound(0)
			}
		}
		if tmp > rtn {
			rtn = tmp
		}
	}
	return rtn
}
func (rc *RCColumn) NearSideReins(cond *Condition) float64 {
	if rc.Reins == nil {
		return 0.0
	}
	rtn := 0.0
	var tmp float64
	for _, r := range rc.Reins {
		if cond.Strong {
			if cond.Positive {
				tmp = rc.Bound(3) - r.Position[1]
			} else {
				tmp = r.Position[1] - rc.Bound(1)
			}
		} else {
			if cond.Positive {
				tmp = rc.Bound(2) - r.Position[0]
			} else {
				tmp = r.Position[0] - rc.Bound(0)
			}
		}
		if tmp < rtn {
			rtn = tmp
		}
	}
	return rtn
}
func (rc *RCColumn) NeutralAxis(cond *Condition) (float64, float64, error) {
	if rc.Nreins == 0 {
		return 0.0, 0.0, errors.New("NeutralAxis: No Reinforce")
	}
	fc := rc.Fc(cond)
	ft := rc.Reins[0].Ft(cond)
	if cond.N < -ft*rc.Ai() {
		return 0.0, 0.0, errors.New("NeutralAxis: Tension is too much")
	}
	b := rc.Breadth(cond.Strong)
	h := rc.Height(cond.Strong)
	nmax := fc*b*h + NCS*fc*rc.Ai()
	if cond.N > nmax {
		return 0.0, 0.0, errors.New("NeutralAxis: Compression is too much")
	}
	num := 0.5*fc*b*math.Pow(h, 2.0) + NCS*fc*rc.LiAi(cond)
	den := nmax - cond.N
	xn := num / den
	if xn > h {
		ryc := rc.NearSideReins(cond)
		if (xn-ryc)/xn*fc*NCS <= ft { // NeutralAxis is outside of section, Ma is determined by concrete
			if cond.Verbose {
				fmt.Println("# 1. Neutral Axis is outside of section")
				fmt.Println("#    Ma is determined by concrete.")
			}
			return xn, fc, nil
		} else { // NeutralAxis is outside of section, Ma is determined by reinforcement
			num := 0.5*ft*b*math.Pow(h, 2.0) + NCS*ft*rc.LiAi(cond) - NCS*ryc*cond.N
			den := ft*b*h - NCS*cond.N + NCS*ft*rc.Ai()
			xn = num / den
			if cond.Verbose {
				fmt.Println("# 2. Neutral Axis is outside of section")
				fmt.Println("#    Ma is determined by reinforcement.")
			}
			return xn, xn / (NCS * (xn - ryc)) * ft, nil
		}
	} else {
		k1 := 0.5 * fc * b
		k2 := NCS*fc*rc.Ai() - cond.N
		k3 := -NCS * fc * rc.LiAi(cond)
		ryt := rc.FarSideReins(cond)
		D1 := k2 * k2 - 4.0 * k1 * k3
		if D1 >= 0.0 {
			xn := (-k2 + math.Sqrt(D1)) / (2.0 * k1)
			if xn >= 0.0 {
				if (ryt-xn)/xn*fc*NCS <= ft { // NeutralAxis is inside of section, Ma is determined by concrete
					if cond.Verbose {
						fmt.Println("# 3. Neutral Axis is inside of section")
						fmt.Println("#    Ma is determined by concrete.")
					}
					return xn, fc, nil
				} else { // NeutralAxis is inside of section, Ma is determined by reinforcement
					k1 := 0.5 * ft * b
					k2 := NCS*ft*rc.Ai() + NCS*cond.N
					k3 := -NCS*ft*rc.LiAi(cond) - NCS*ryt*cond.N
					D2 := k2 * k2 - 4.0 * k1 * k3
					if D2 >= 0.0 {
						xn := (-k2 + math.Sqrt(D2)) / (2.0 * k1)
						if xn >= 0.0 {
							if cond.Verbose {
								fmt.Println("# 4. Neutral Axis is inside of section")
								fmt.Println("#    Ma is determined by reinforcement.")
							}
							return xn, xn / (NCS * (ryt - xn)) * ft, nil
						}
					}
				}
			}
		}
		num := ft * rc.LiAi(cond) + ryt * cond.N
		den := ft * rc.Ai() + cond.N
		xn = num / den
		if cond.Verbose {
			fmt.Println("# 5. Neutral Axis is outside of section")
			fmt.Println("#    Ma is determined by reinforcement.")
		}
		return xn, -xn / (NCS * (ryt - xn)) * ft, nil
	}
}
func (rc *RCColumn) Na(cond *Condition) float64 {
	if cond.Compression {
		if cond.Verbose {
			fmt.Printf("# Fc= %.3f [tf/cm2]\n# Ac= %.3f [cm2]\n", rc.Fc(cond), rc.Area())
		}
		return rc.Fc(cond) * rc.Area()
	} else {
		if rc.Reins == nil {
			return 0.0
		}
		rtn := 0.0
		for _, r := range rc.Reins {
			rtn += r.Area * r.Ft(cond)
		}
		return rtn
	}
}
func (rc *RCColumn) Alpha(d float64, cond *Condition) float64 {
	alpha := 4.0 / (math.Abs(cond.M * 100.0 / (cond.Q * d)) + 1.0)
	if alpha < 1.0 {
		alpha = 1.0
	} else if alpha > 1.5 {
		alpha = 1.5
	}
	return alpha
}
func (rc *RCColumn) Qa(cond *Condition) float64 {
	b := rc.Breadth(cond.Strong)
	d := rc.FarSideReins(cond)
	fs := rc.Fs(cond)
	alpha := rc.Alpha(d, cond)
	switch cond.Period {
	default:
		fmt.Println("unknown period")
		return 0.0
	case "L":
		return 7/8.0 * b * d * alpha * fs
	case "X", "Y", "S":
		var pw float64
		if cond.Strong { // for Qy
			pw = rc.Hoops.Ps[1]
		} else { // for Qx
			pw = rc.Hoops.Ps[0]
		}
		if pw < 0.002 {
			// fmt.Printf("shortage in pw: %.6f\n", pw)
			return 7/8.0 * b * d * fs
		} else if pw > 0.012 {
			pw = 0.012
		}
		return 7/8.0 * b * d * (fs + 0.5 * rc.Hoops.Ftw(cond) * (pw - 0.002))
	}
}
func (rc *RCColumn) Ma(cond *Condition) float64 {
	b := rc.Breadth(cond.Strong)
	h := rc.Height(cond.Strong)
	xn, sigma, err := rc.NeutralAxis(cond)
	if err != nil {
		if cond.Verbose {
			fmt.Println(err)
		}
		return 0.0
	}
	if cond.Verbose {
		fmt.Printf("# xn= %.3f [m]\n# sigma= %.3f [tf/cm2]\n", xn, sigma)
	}
	if xn >= h {
		return (sigma/xn*(b*h*(3.0*math.Pow(xn, 2.0)-3.0*xn*h+math.Pow(h, 2.0))/3.0+NCS*(math.Pow(xn, 2.0)*rc.Ai()-2.0*xn*rc.LiAi(cond)+rc.Li2Ai(cond))) - cond.N*(xn-h/2.0)) * 0.01 // [tfm]
	} else {
		return (sigma*(b*math.Pow(xn, 2.0)/3.0+NCS*(xn*rc.Ai()-2.0*rc.LiAi(cond)+rc.Li2Ai(cond)/xn)) - cond.N*(xn-h/2.0)) * 0.01 // [tfm]
	}
}
func (rc *RCColumn) Mza(cond *Condition) float64 {
	if rc.Reins == nil || len(rc.Reins) < 1 {
		return 0.0
	}
	ft := rc.Reins[0].Ft(cond)
	fs := rc.Fs(cond)
	wft := rc.Hoops.Ftw(cond)
	b := rc.Breadth(true)
	d := rc.Height(true)
	dw := 11.0 // TODO: set dw, aw, lw & kaburi
	aw := 0.7133
	lw := 20.0
	kaburi := 5.0
	b0 := b - kaburi*2.0 - dw
	d0 := d - kaburi*2.0 - dw
	A0 := b0 * d0
	var T1, T2, T3 float64
	if b >= d {
		T1 = b * d * d * fs * 4.0 / 3.0 / 100.0 // [tfm]
	} else {
		T1 = b * b * d * fs * 4.0 /3.0 / 100.0 // [tfm]
	}
	T2 = aw * 2.0 * wft * A0 / lw / 100.0 // [tfm]
	T3 = rc.Ai() * 2.0 * ft * A0 / (2 * b0 + 2 * d0) / 100.0 // [tfm]
	if cond.Verbose {
		fmt.Printf("# T1= %.3f [tfm] T2= %.3f [tfm] T3= %.3f [tfm]\n", T1, T2, T3)
	}
	if T1 <= T2 {
		if T1 <= T3 {
			return T1
		} else {
			return T3
		}
	} else {
		if T2 <= T3 {
			return T2
		} else {
			return T3
		}
	}
}

type RCGirder struct {
	RCColumn
}

func NewRCGirder(num int) *RCGirder {
	rc := NewRCColumn(num)
	return &RCGirder{*rc}
}
func (rg *RCGirder) TypeString() string {
	return "ＲＣ大梁"
}
func (rg *RCGirder) Alpha(d float64, cond *Condition) float64 {
	alpha := 4.0 / (math.Abs(cond.M * 100.0 / (cond.Q * d)) + 1.0)
	if alpha < 1.0 {
		alpha = 1.0
	} else if alpha > 2.0 {
		alpha = 2.0
	}
	return alpha
}
func (rg *RCGirder) Qa(cond *Condition) float64 {
	b := rg.Breadth(cond.Strong)
	d := rg.FarSideReins(cond)
	fs := rg.Fs(cond)
	alpha := rg.Alpha(d, cond)
	if cond.Verbose {
		fmt.Println(b, d, alpha, fs)
	}
	var pw float64
	if cond.Strong { // for Qy
		pw = rg.Hoops.Ps[1]
	} else { // for Qx
		pw = rg.Hoops.Ps[0]
	}
	switch cond.Period {
	default:
		fmt.Println("unknown period")
		return 0.0
	case "L":
		if pw < 0.002 {
			// fmt.Printf("shortage in pw: %.6f\n", pw)
			return 7/8.0 * b * d * fs
		} else if pw > 0.006 {
			pw = 0.006
		}
		return 7/8.0 * b * d * (alpha * fs + 0.5 * rg.Hoops.Ftw(cond) * (pw - 0.002))
	case "X", "Y", "S":
		if pw < 0.002 {
			// fmt.Printf("shortage in pw: %.6f\n", pw)
			return 7/8.0 * b * d * fs
		} else if pw > 0.012 {
			pw = 0.012
		}
		return 7/8.0 * b * d * (alpha * fs + 0.5 * rg.Hoops.Ftw(cond) * (pw - 0.002))
	}
}

type RCWall struct {
	Concrete
}

func SetSD(name string) SD {
	switch name {
	default:
		return SD295
	case "SD295":
		return SD295
	case "SD345":
		return SD345
	}
}

// Condition
type Condition struct {
	Period      string
	Length      float64
	Compression bool
	Strong      bool
	Positive    bool
	FbOld       bool
	N           float64
	M           float64
	Q           float64
	Sign        float64
	Verbose     bool
}

func NewCondition() *Condition {
	c := new(Condition)
	c.Period = "L"
	c.Length = 0.0
	c.Compression = false
	c.Strong = true
	c.Positive = true
	c.FbOld = false
	c.N = 0.0
	c.M = 0.0
	c.Q = 0.0
	c.Sign = 1.0
	return c
}

func Rate(sr SectionRate, stress []float64, cond *Condition) ([]float64, string, error) {
	if len(stress) < 12 {
		return nil, "", errors.New("Rate: Not enough number of Stress")
	}
	rate := make([]float64, 12)
	fa := make([]float64, 12)
	var ind int
	for i:=0; i<2; i++ {
		if i == 0 {
			cond.N = stress[6*i]
			cond.M = stress[6*i+4]
			cond.Q = stress[6*i+2]
		} else {
			cond.N = -stress[6*i]
			cond.M = -stress[6*i+4]
			cond.Q = -stress[6*i+2]
		}
		ind = 6*i+0
		cond.Compression = cond.N >= 0.0
		na := sr.Na(cond)
		if cond.Verbose {
			if cond.Compression {
				fmt.Printf("# N= %.3f / Na= %.3f (COMPRESSION)\n", stress[ind], na)
			} else {
				fmt.Printf("# N= %.3f / Na= %.3f (TENSION)\n", stress[ind], na)
			}
		}
		if na == 0.0 && stress[ind] != 0.0 {
			return rate, "", ZeroAllowableError{"Na"}
		}
		rate[ind] = math.Abs(stress[ind] / na)
		fa[ind] = na
		cond.Strong = true
		ind = 6*i+2
		cond.Positive = cond.M >= 0.0
		qay := sr.Qa(cond)
		if cond.Verbose {
			fmt.Printf("# Qy= %.3f / Qay= %.3f\n", stress[ind], qay)
		}
		if qay == 0.0 && stress[ind] != 0.0 {
			return rate, "", ZeroAllowableError{"Qay"}
		}
		rate[ind] = math.Abs(stress[ind] / qay)
		fa[ind] = qay
		ind = 6*i+4
		max := sr.Ma(cond)
		if cond.Verbose {
			fmt.Printf("# Mx= %.3f / Max= %.3f\n", stress[ind], max)
		}
		if max == 0.0 && stress[ind] != 0.0 {
			return rate, "", ZeroAllowableError{"MaX"}
		}
		rate[ind] = math.Abs(stress[ind] / max)
		fa[ind] = max
		cond.Strong = false
		if i == 0 {
			cond.M = stress[6*i+5]
			cond.Q = stress[6*i+1]
		} else {
			cond.M = -stress[6*i+5]
			cond.Q = -stress[6*i+1]
		}
		cond.Positive = cond.M >= 0.0
		ind = 6*i+1
		qax := sr.Qa(cond)
		if cond.Verbose {
			fmt.Printf("# Qx= %.3f / Qax= %.3f\n", stress[ind], qax)
		}
		if qax == 0.0 && stress[ind] != 0.0 {
			return rate, "", ZeroAllowableError{"Qax"}
		}
		rate[ind] = math.Abs(stress[ind] / qax)
		fa[ind] = qax
		ind = 6*i+5
		may := sr.Ma(cond)
		if cond.Verbose {
			fmt.Printf("# My= %.3f / May= %.3f\n", stress[ind], may)
		}
		if may == 0.0 && stress[ind] != 0.0 {
			return rate, "", ZeroAllowableError{"May"}
		}
		rate[ind] = math.Abs(stress[ind] / may)
		fa[ind] = may
		ind = 6*i+3
		maz := sr.Mza(cond)
		if cond.Verbose {
			fmt.Printf("# Mz= %.3f / Maz= %.3f\n", stress[ind], maz)
		}
		if maz == 0.0 && stress[ind] != 0.0 {
			return rate, "", ZeroAllowableError{"Maz"}
		}
		rate[ind] = math.Abs(stress[ind] / maz)
		fa[ind] = maz
	}
	var otp bytes.Buffer
	for i:=0; i<6; i++ {
		for j:=0; j<2; j++ {
			otp.WriteString(fmt.Sprintf(" %8.3f(%8.2f)", stress[6*j+i], stress[6*j+i]*SI))
			if i == 0 || i == 3 {
				break
			}
		}
	}
	otp.WriteString("\n     許容値:")
	for i:=0; i<6; i++ {
		for j:=0; j<2; j++ {
			otp.WriteString(fmt.Sprintf(" %8.3f(%8.2f)", fa[6*j+i], fa[6*j+i]*SI))
			if i == 0 || i == 3 {
				break
			}
		}
	}
	otp.WriteString("\n     安全率:")
	for i:=0; i<6; i++ {
		for j:=0; j<2; j++ {
			otp.WriteString(fmt.Sprintf(" %8.3f          ", rate[6*j+i]))
			if i == 0 || i == 3 {
				break
			}
		}
	}
	otp.WriteString("\n")
	return rate, otp.String(), nil
}
