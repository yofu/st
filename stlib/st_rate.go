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

type Breadther interface {
	Breadth(bool) float64
}

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
func (rf Reinforce) Radius() float64 {
	return math.Sqrt(rf.Area / math.Pi)
}
func (rf Reinforce) Vertices() [][]float64 {
	d := rf.Radius()
	val := math.Pi / 8.0
	theta := 0.0
	vertices := make([][]float64, 16)
	for i := 0; i < 16; i++ {
		c := math.Cos(theta)
		s := math.Sin(theta)
		vertices[i] = []float64{d*c + rf.Position[0], d*s + rf.Position[1]}
		theta += val
	}
	return vertices
}

type Wood struct {
	Name string
	fc   float64
	ft   float64
	fb   float64
	fs   float64
	e    float64
	poi  float64
}

var (
	SN400    = Steel{"SN400", 2.4, 4.0, 2100.0, 0.3}
	SN490    = Steel{"SN490", 3.3, 5.0, 2100.0, 0.3}
	SN400T40 = Steel{"SN400T40", 2.2, 4.0, 2100.0, 0.3}
	SN490T40 = Steel{"SN490T40", 3.0, 5.0, 2100.0, 0.3}
	HT600    = Steel{"HT600", 6.0, 8.0, 2100.0, 0.3}
	HT700    = Steel{"HT700", 7.0, 9.0, 2100.0, 0.3}

	// ALUMINIUM
	A6061T6 = Steel{"A6061T6", 2.141, 4.0, 700.0, 0.3}
	A6063T5 = Steel{"A6063T5", 1.121, 4.0, 700.0, 0.3}

	// CARBON
	M40J = Steel{"M40J", 5.438, 8.157, 1100.0, 0.3}
	T300 = Steel{"T300", 7.477, 11.216, 1100.0, 0.3}

	FC18 = Concrete{"Fc18", 0.180, 210.0, 0.166666}
	FC24 = Concrete{"Fc24", 0.240, 210.0, 0.166666}
	FC27 = Concrete{"Fc27", 0.270, 210.0, 0.166666}
	FC30 = Concrete{"Fc30", 0.300, 210.0, 0.166666}
	FC36 = Concrete{"Fc36", 0.360, 210.0, 0.166666}

	SD295 = SD{"SD295", 2.0, 3.0, 2100.0, 0.3}
	SD345 = SD{"SD345", 2.2, 3.5, 2100.0, 0.3}
	SD390 = SD{"SD390", 2.2, 3.9, 2100.0, 0.3}

	S_E70     = Wood{"S-E70", 0.2386, 0.1774, 0.2997, 0.0183, 70.0, 6.5}
	E70SUGI   = S_E70
	H_E70     = Wood{"H-E70", 0.1835, 0.1346, 0.2263, 0.0214, 70.0, 6.5}
	E70HINOKI = H_E70
	H_E90     = Wood{"H-E90", 0.2508, 0.1896, 0.3120, 0.0214, 90.0, 6.5}
	E90HINOKI = H_E90
	M_E90     = Wood{"M-E90", 0.1714, 0.1285, 0.2142, 0.0244, 90.0, 6.5}
	M_E110    = Wood{"M-E110", 0.2508, 0.1896, 0.3120, 0.0244, 110.0, 6.5}
	E95_F270  = Wood{"E95-F270", 0.2221, 0.1927, 0.2753, 0.0367, 95.0, 6.5}
	E95_F315  = Wood{"E95-F315", 0.2651, 0.2314, 0.3212, 0.0367, 95.0, 6.5}
	E120_F330 = Wood{"E120-F330", 0.2569, 0.2263, 0.3303, 0.0367, 120.0, 6.5}

	GOHAN = Wood{"GOHAN", 0.0, 0.0, 0.0, 0.012, 0.0, 0.0}
)

// Section
type SectionRate interface {
	Num() int
	TypeString() string
	Snapshot() SectionRate
	String() string
	Name() string
	SetName(string)
	SetValue(string, []float64)
	Factor(string) float64
	Na(*Condition) float64
	Qa(*Condition) float64
	Ma(*Condition) float64
	Mza(*Condition) float64
	Amount() Amount
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
	Breadth(bool) float64
}

type SColumn struct {
	Steel
	Shape
	num      int
	Etype    string
	name     string
	XFace    []float64
	YFace    []float64
	BBLength []float64
	BTLength []float64
	BBFactor []float64
	BTFactor []float64
	multi    float64
}

func NewSColumn(num int, shape Shape, material Steel) *SColumn {
	return &SColumn{
		Steel:    material,
		Shape:    shape,
		num:      num,
		Etype:    "COLUMN",
		name:     "",
		XFace:    nil,
		YFace:    nil,
		BBLength: nil,
		BTLength: nil,
		BBFactor: nil,
		BTFactor: nil,
		multi:    1.0,
	}
}
func (sc *SColumn) Num() int {
	return sc.num
}
func (sc *SColumn) TypeString() string {
	return "Ｓ　柱　"
}
func (sc *SColumn) Snapshot() SectionRate {
	s := NewSColumn(sc.num, sc.Shape, sc.Steel)
	s.Etype = sc.Etype
	s.name = sc.name
	if sc.XFace != nil {
		s.XFace = make([]float64, 2)
		s.XFace[0] = sc.XFace[0]
		s.XFace[1] = sc.XFace[1]
	}
	if sc.YFace != nil {
		s.YFace = make([]float64, 2)
		s.YFace[0] = sc.YFace[0]
		s.YFace[1] = sc.YFace[1]
	}
	if sc.BBLength != nil {
		s.BBLength = make([]float64, 2)
		s.BBLength[0] = sc.BBLength[0]
		s.BBLength[1] = sc.BBLength[1]
	}
	if sc.BTLength != nil {
		s.BTLength = make([]float64, 2)
		s.BTLength[0] = sc.BTLength[0]
		s.BTLength[1] = sc.BTLength[1]
	}
	if sc.BBFactor != nil {
		s.BBFactor = make([]float64, 2)
		s.BBFactor[0] = sc.BBFactor[0]
		s.BBFactor[1] = sc.BBFactor[1]
	}
	if sc.BTFactor != nil {
		s.BTFactor = make([]float64, 2)
		s.BTFactor[0] = sc.BTFactor[0]
		s.BTFactor[1] = sc.BTFactor[1]
	}
	return s
}
func (sc *SColumn) Name() string {
	return sc.name
}
func (sc *SColumn) SetName(name string) {
	sc.name = name
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
	case "MULTI":
		sc.multi = vals[0]
	}
}
func (sc *SColumn) String() string {
	var rtn bytes.Buffer
	rtn.WriteString(fmt.Sprintf("CODE %3d S %s %57s\n", sc.num, sc.Etype, fmt.Sprintf("\"%s\"", sc.name)))
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
	var lambda_x, lambda_y, lambda float64
	lx := sc.Lk(cond.Length, true)
	ly := sc.Lk(cond.Length, false)
	if an, ok := sc.Shape.(ANGLE); ok {
		if lx > ly {
			lambda = lx / math.Sqrt(an.Imin()/an.A())
		} else {
			lambda = ly / math.Sqrt(an.Imin()/an.A())
		}
		lambda_x = lambda
		lambda_y = lambda
	} else {
		lambda_x = lx / math.Sqrt(sc.Ix()/sc.A())
		lambda_y = ly / math.Sqrt(sc.Iy()/sc.A())
		if lambda_x >= lambda_y {
			lambda = lambda_x
		} else {
			lambda = lambda_y
		}
	}
	val := lambda / sc.Lambda()
	if val <= 1.0 {
		nu := 1.5 + 2.0*val*val/3.0
		rtn = (1.0 - 0.4*val*val) * sc.F / nu
	} else {
		rtn = 0.277 * sc.F / (val * val)
	}
	rtn *= sc.Factor(cond.Period)
	if cond.Verbose {
		cond.Buffer.WriteString(fmt.Sprintf("#     座屈長さ[cm]: Lkx=%.3f, Lky=%.3f\n", lx, ly))
		check := ""
		if lambda > 200 {
			check = " λ>200"
		}
		cond.Buffer.WriteString(fmt.Sprintf("#     細長比: λx=%.3f, λy=%.3f: λ=%.3f%s\n", lambda_x, lambda_y, lambda, check))
		cond.Buffer.WriteString(fmt.Sprintf("#     許容圧縮応力度: Fc=%.3f [tf/cm2]\n", rtn))
	}
	return rtn
}
func (sc *SColumn) Ft(cond *Condition) float64 {
	return sc.F / 1.5 * sc.Factor(cond.Period)
}
func (sc *SColumn) Fs(cond *Condition) float64 {
	return sc.F / (1.5 * math.Sqrt(3)) * sc.Factor(cond.Period)
}
func (sc *SColumn) Fb(cond *Condition) float64 {
	l := sc.Lb(cond.Length, cond.Strong)
	if cond.Verbose {
		cond.Buffer.WriteString(fmt.Sprintf("#     横座屈長さ[cm]: Lb=%.3f\n", l))
	}
	var rtn float64
	fbnew := func() float64 {
		me := sc.Me(l, 1.0)
		my := sc.My(cond)
		lambda_b := math.Sqrt(my / me)
		nu := 1.5 + math.Pow(lambda_b/E_LAMBDA_B, 2.0)/1.5
		if cond.Verbose {
			cond.Buffer.WriteString(fmt.Sprintf("#     弾性横座屈モーメント、降伏モーメント、横座屈細長比: Me=%.3f [tfm], My=%.3f [tfm], λb=%.3f\n", me, my, lambda_b))
		}
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
				if rtn > sc.F/1.5 {
					rtn = sc.F / 1.5
				}
			} else {
				rtn = fbnew()
			}
		} else if an, ok := sc.Shape.(ANGLE); ok {
			if cond.FbOld {
				rtn = 900.0 / (l * an.H / (an.B * an.Tf))
				if rtn > sc.F/1.5 {
					rtn = sc.F / 1.5
				}
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
				if rtn > sc.F/1.5 {
					rtn = sc.F / 1.5
				}
			} else {
				rtn = fbnew()
			}
		} else if an, ok := sc.Shape.(ANGLE); ok {
			if cond.FbOld {
				rtn = 900.0 / (l * an.B / (an.H * an.Tw))
				if rtn > sc.F/1.5 {
					rtn = sc.F / 1.5
				}
			} else {
				rtn = fbnew()
			}
		} else {
			rtn = sc.F / 1.5
		}
	}
	rtn *= sc.Factor(cond.Period)
	if cond.Verbose {
		if cond.Strong {
			cond.Buffer.WriteString(fmt.Sprintf("#     許容曲げ応力度(Mx): Fb=%.3f [tf/cm2]\n", rtn))
		} else {
			cond.Buffer.WriteString(fmt.Sprintf("#     許容曲げ応力度(My): Fb=%.3f [tf/cm2]\n", rtn))
		}
	}
	return rtn
}
func (sc *SColumn) Na(cond *Condition) float64 {
	switch sc.Shape.(type) {
	case SAREA:
		return sc.Ft(cond) * sc.A() * sc.multi
	default:
		if cond.Compression {
			return sc.Fc(cond) * sc.A() * sc.multi
		} else {
			return sc.Ft(cond) * sc.A() * sc.multi
		}
	}
}
func (sc *SColumn) Qa(cond *Condition) float64 {
	f := sc.Fs(cond)
	if cond.Strong { // for Qy
		return f * sc.Asy() * sc.multi
	} else { // for Qx
		return f * sc.Asx() * sc.multi
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
	return Cb * math.Sqrt(math.Pow(math.Pi, 4.0)*sc.E*I*sc.E*sc.Iw()/math.Pow(length, 4.0)+math.Pow(math.Pi, 2.0)*sc.E*I*g*sc.J()/math.Pow(length, 2.0)) * 0.01 * sc.multi // [tfm]
}
func (sc *SColumn) My(cond *Condition) float64 {
	if cond.Strong {
		return sc.F * sc.Zx() * 0.01 * sc.multi // [tfm]
	} else {
		return sc.F * sc.Zy() * 0.01 * sc.multi // [tfm]
	}
}
func (sc *SColumn) Ma(cond *Condition) float64 {
	f := sc.Fb(cond)
	if cond.Strong {
		return f * sc.Zx() * 0.01 * sc.multi // [tfm]
	} else {
		return f * sc.Zy() * 0.01 * sc.multi // [tfm]
	}
}
func (sc *SColumn) Mza(cond *Condition) float64 {
	return sc.Fs(cond) * sc.Torsion() * 0.01 * sc.multi // [tfm]
}

func (sc *SColumn) Vertices() [][]float64 {
	return sc.Shape.Vertices()
}

func (sc *SColumn) Amount() Amount {
	a := NewAmount()
	a["STEEL"] = sc.A() * 0.0001
	return a
}

type HKYOU struct {
	H, B, Tw, Tf float64
}

func NewHKYOU(lis []string) (HKYOU, error) {
	hk := HKYOU{0.0, 0.0, 0.0, 0.0}
	if len(lis) < 4 {
		return hk, NotEnoughArgs("NewHKYOU")
	}
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
	vertices[2] = []float64{b, -(h - f)}
	vertices[3] = []float64{w, -(h - f)}
	vertices[4] = []float64{w, h - f}
	vertices[5] = []float64{b, h - f}
	vertices[6] = []float64{b, h}
	vertices[7] = []float64{-b, h}
	vertices[8] = []float64{-b, h - f}
	vertices[9] = []float64{-w, h - f}
	vertices[10] = []float64{-w, -(h - f)}
	vertices[11] = []float64{-b, -(h - f)}
	return vertices
}

func (hk HKYOU) Breadth(strong bool) float64 {
	if strong {
		return hk.B
	} else {
		return hk.H
	}
}

type HWEAK struct {
	H, B, Tw, Tf float64
}

func NewHWEAK(lis []string) (HWEAK, error) {
	hw := HWEAK{0.0, 0.0, 0.0, 0.0}
	if len(lis) < 4 {
		return hw, NotEnoughArgs("NewHWEAK")
	}
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
	vertices[2] = []float64{-(h - f), b}
	vertices[3] = []float64{-(h - f), w}
	vertices[4] = []float64{h - f, w}
	vertices[5] = []float64{h - f, b}
	vertices[6] = []float64{h, b}
	vertices[7] = []float64{h, -b}
	vertices[8] = []float64{h - f, -b}
	vertices[9] = []float64{h - f, -w}
	vertices[10] = []float64{-(h - f), -w}
	vertices[11] = []float64{-(h - f), -b}
	return vertices
}

func (hw HWEAK) Breadth(strong bool) float64 {
	if strong {
		return hw.H
	} else {
		return hw.B
	}
}

type CROSS struct {
	Hkyou HKYOU
	Hweak HWEAK
}

func NewCROSS(lis []string) (CROSS, error) {
	cr := CROSS{
		Hkyou: HKYOU{0.0, 0.0, 0.0, 0.0},
		Hweak: HWEAK{0.0, 0.0, 0.0, 0.0},
	}
	if len(lis) < 8 {
		return cr, NotEnoughArgs("NewCROSS")
	}
	hk, err := NewHKYOU(lis[:4])
	if err != nil {
		return cr, err
	}
	cr.Hkyou = hk
	hw, err := NewHWEAK(lis[4:])
	if err != nil {
		return cr, err
	}
	cr.Hweak = hw
	return cr, nil
}
func (cr CROSS) String() string {
	return fmt.Sprintf("CROSS %5.1f %5.1f %4.1f %4.1f %5.1f %5.1f %4.1f %4.1f", cr.Hkyou.H, cr.Hkyou.B, cr.Hkyou.Tw, cr.Hkyou.Tf, cr.Hweak.H, cr.Hweak.B, cr.Hweak.Tw, cr.Hweak.Tf)
}
func (cr CROSS) Description() string {
	return fmt.Sprintf("H-%dx%dx%dx%d+H-%dx%dx%dx%d[mm]", int(cr.Hkyou.H*10), int(cr.Hkyou.B*10), int(cr.Hkyou.Tw*10), int(cr.Hkyou.Tf*10), int(cr.Hweak.H*10), int(cr.Hweak.B*10), int(cr.Hweak.Tw*10), int(cr.Hweak.Tf*10))
}
func (cr CROSS) A() float64 {
	return cr.Hkyou.A() + cr.Hweak.A()
}
func (cr CROSS) Asx() float64 {
	return cr.Hkyou.Asx() + cr.Hweak.Asx()
}
func (cr CROSS) Asy() float64 {
	return cr.Hkyou.Asy() + cr.Hweak.Asy()
}
func (cr CROSS) Ix() float64 {
	return cr.Hkyou.Ix() + cr.Hweak.Ix()
}
func (cr CROSS) Iy() float64 {
	return cr.Hkyou.Iy() + cr.Hweak.Iy()
}
func (cr CROSS) J() float64 {
	return cr.Hkyou.J() + cr.Hweak.J()
}
func (cr CROSS) Iw() float64 {
	return cr.Hkyou.Iw() + cr.Hweak.Iw()
}
func (cr CROSS) Torsion() float64 {
	thick := 0.0
	if cr.Hkyou.Tf > thick {
		thick = cr.Hkyou.Tf
	}
	if cr.Hkyou.Tw > thick {
		thick = cr.Hkyou.Tw
	}
	if cr.Hweak.Tf > thick {
		thick = cr.Hweak.Tf
	}
	if cr.Hweak.Tw > thick {
		thick = cr.Hweak.Tw
	}
	return cr.J() / thick
}
func (cr CROSS) Zx() float64 {
	return cr.Ix() / cr.Hkyou.H * 2.0
}
func (cr CROSS) Zy() float64 {
	return cr.Iy() / cr.Hweak.H * 2.0
}

func (cr CROSS) Vertices() [][]float64 {
	vk := cr.Hkyou.Vertices()
	vw := cr.Hweak.Vertices()
	vertices := make([][]float64, 25)
	for i := 0; i < 12; i++ {
		vertices[i] = make([]float64, 2)
		vertices[i+13] = make([]float64, 2)
		for j := 0; j < 2; j++ {
			vertices[i][j] = vk[i][j]
			vertices[i+13][j] = vw[i][j]
		}
	}
	vertices[12] = nil
	return vertices
}

func (cr CROSS) Breadth(strong bool) float64 {
	if strong {
		return cr.Hweak.H
	} else {
		return cr.Hkyou.H
	}
}

type RPIPE struct {
	H, B, Tw, Tf float64
}

func NewRPIPE(lis []string) (RPIPE, error) {
	rp := RPIPE{0.0, 0.0, 0.0, 0.0}
	if len(lis) < 4 {
		return rp, NotEnoughArgs("NewRPIPE")
	}
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

func (rp RPIPE) Breadth(strong bool) float64 {
	if strong {
		return rp.B
	} else {
		return rp.H
	}
}

type CPIPE struct {
	D, T float64
}

func NewCPIPE(lis []string) (CPIPE, error) {
	cp := CPIPE{0.0, 0.0}
	if len(lis) < 2 {
		return cp, NotEnoughArgs("NewCPIPE")
	}
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
	val := math.Pi / 8.0
	theta := 0.0
	vertices := make([][]float64, 33)
	for i := 0; i < 16; i++ {
		c := math.Cos(theta)
		s := math.Sin(theta)
		vertices[i] = []float64{d * c, d * s}
		vertices[i+17] = []float64{dt * c, dt * s}
		theta += val
	}
	vertices[16] = nil
	return vertices
}

func (cp CPIPE) Breadth(strong bool) float64 {
	return cp.D
}

type TKYOU struct {
	H, B, Tw, Tf float64
}

func NewTKYOU(lis []string) (TKYOU, error) {
	tk := TKYOU{0.0, 0.0, 0.0, 0.0}
	if len(lis) < 4 {
		return tk, NotEnoughArgs("NewTKYOU")
	}
	var val float64
	var err error
	val, err = strconv.ParseFloat(lis[0], 64)
	if err != nil {
		return tk, err
	}
	tk.H = val
	val, err = strconv.ParseFloat(lis[1], 64)
	if err != nil {
		return tk, err
	}
	tk.B = val
	val, err = strconv.ParseFloat(lis[2], 64)
	if err != nil {
		return tk, err
	}
	tk.Tw = val
	val, err = strconv.ParseFloat(lis[3], 64)
	if err != nil {
		return tk, err
	}
	tk.Tf = val
	return tk, nil
}
func (tk TKYOU) String() string {
	return fmt.Sprintf("TKYOU %5.1f %5.1f %4.1f %4.1f", tk.H, tk.B, tk.Tw, tk.Tf)
}
func (tk TKYOU) Description() string {
	return fmt.Sprintf("T-%dx%dx%dx%d(KYOU)[mm]", int(tk.H*10), int(tk.B*10), int(tk.Tw*10), int(tk.Tf*10))
}
func (tk TKYOU) A() float64 {
	return tk.H*tk.B - (tk.H-tk.Tf)*(tk.B-tk.Tw)
}
func (tk TKYOU) Asx() float64 {
	return tk.B * tk.Tf / 1.5
}
func (tk TKYOU) Asy() float64 {
	return (tk.H - tk.Tf) * tk.Tw / 1.5
}
func (tk TKYOU) Cy() float64 {
	return ((tk.B-tk.Tw)*tk.Tf*0.5*tk.Tf + tk.H*tk.Tw*0.5*tk.H) / tk.A()
}
func (tk TKYOU) Ix() float64 {
	cy := tk.Cy()
	return (tk.B-tk.Tw)*math.Pow(tk.Tf, 3.0)/12.0 + tk.Tw*math.Pow(tk.H, 3.0)/12.0 + (tk.B-tk.Tw)*tk.Tf*math.Pow(cy-0.5*tk.Tf, 2.0) + tk.H*tk.Tw*math.Pow(0.5*tk.H-cy, 2.0)
}
func (tk TKYOU) Iy() float64 {
	return tk.Tf*math.Pow(tk.B, 3.0)/12.0 + (tk.H-tk.Tf)*math.Pow(tk.Tw, 3.0)/12.0
}
func (tk TKYOU) J() float64 {
	return tk.B*math.Pow(tk.Tf, 3.0)/3.0 + (tk.H-tk.Tf)*math.Pow(tk.Tw, 3.0)/3.0
}
func (tk TKYOU) Iw() float64 {
	return 0.0
}
func (tk TKYOU) Torsion() float64 {
	if tk.Tf >= tk.Tw {
		return tk.J() / tk.Tf
	} else {
		return tk.J() / tk.Tw
	}
}
func (tk TKYOU) Zx() float64 {
	cy := tk.Cy()
	if cy >= tk.H*0.5 {
		return tk.Ix() / cy
	} else {
		return tk.Ix() / (tk.H - cy)
	}
}
func (tk TKYOU) Zy() float64 {
	return tk.Iy() / tk.B * 2.0
}

func (tk TKYOU) Vertices() [][]float64 {
	c := tk.Cy()
	h := tk.H - c
	b := tk.B * 0.5
	w := tk.Tw * 0.5
	f := tk.Tf
	vertices := make([][]float64, 8)
	vertices[0] = []float64{-w, -h}
	vertices[1] = []float64{w, -h}
	vertices[2] = []float64{w, c - f}
	vertices[3] = []float64{b, c - f}
	vertices[4] = []float64{b, c}
	vertices[5] = []float64{-b, c}
	vertices[6] = []float64{-b, c - f}
	vertices[7] = []float64{-w, c - f}
	return vertices
}

func (tk TKYOU) Breadth(strong bool) float64 {
	if strong {
		return tk.B
	} else {
		return tk.H
	}
}

type TWEAK struct {
	H, B, Tw, Tf float64
}

func NewTWEAK(lis []string) (TWEAK, error) {
	tk := TWEAK{0.0, 0.0, 0.0, 0.0}
	if len(lis) < 4 {
		return tk, NotEnoughArgs("NewTWEAK")
	}
	var val float64
	var err error
	val, err = strconv.ParseFloat(lis[0], 64)
	if err != nil {
		return tk, err
	}
	tk.H = val
	val, err = strconv.ParseFloat(lis[1], 64)
	if err != nil {
		return tk, err
	}
	tk.B = val
	val, err = strconv.ParseFloat(lis[2], 64)
	if err != nil {
		return tk, err
	}
	tk.Tw = val
	val, err = strconv.ParseFloat(lis[3], 64)
	if err != nil {
		return tk, err
	}
	tk.Tf = val
	return tk, nil
}
func (tk TWEAK) String() string {
	return fmt.Sprintf("TWEAK %5.1f %5.1f %4.1f %4.1f", tk.H, tk.B, tk.Tw, tk.Tf)
}
func (tk TWEAK) Description() string {
	return fmt.Sprintf("T-%dx%dx%dx%d(WEAK)[mm]", int(tk.H*10), int(tk.B*10), int(tk.Tw*10), int(tk.Tf*10))
}
func (tk TWEAK) A() float64 {
	return tk.H*tk.B - (tk.H-tk.Tf)*(tk.B-tk.Tw)
}
func (tk TWEAK) Asx() float64 {
	return (tk.H - tk.Tf) * tk.Tw / 1.5
}
func (tk TWEAK) Asy() float64 {
	return tk.B * tk.Tf / 1.5
}
func (tk TWEAK) Cx() float64 {
	return ((tk.B-tk.Tw)*tk.Tf*0.5*tk.Tf + tk.H*tk.Tw*0.5*tk.H) / tk.A()
}
func (tk TWEAK) Ix() float64 {
	return tk.Tf*math.Pow(tk.B, 3.0)/12.0 + (tk.H-tk.Tf)*math.Pow(tk.Tw, 3.0)/12.0
}
func (tk TWEAK) Iy() float64 {
	cx := tk.Cx()
	return (tk.B-tk.Tw)*math.Pow(tk.Tf, 3.0)/12.0 + tk.Tw*math.Pow(tk.H, 3.0)/12.0 + (tk.B-tk.Tw)*tk.Tf*math.Pow(cx-0.5*tk.Tf, 2.0) + tk.H*tk.Tw*math.Pow(0.5*tk.H-cx, 2.0)
}
func (tk TWEAK) J() float64 {
	return tk.B*math.Pow(tk.Tf, 3.0)/3.0 + (tk.H-tk.Tf)*math.Pow(tk.Tw, 3.0)/3.0
}
func (tk TWEAK) Iw() float64 {
	return 0.0
}
func (tk TWEAK) Torsion() float64 {
	if tk.Tf >= tk.Tw {
		return tk.J() / tk.Tf
	} else {
		return tk.J() / tk.Tw
	}
}
func (tk TWEAK) Zx() float64 {
	return tk.Ix() / tk.B * 2.0
}
func (tk TWEAK) Zy() float64 {
	cx := tk.Cx()
	if cx >= tk.H*0.5 {
		return tk.Iy() / cx
	} else {
		return tk.Iy() / (tk.H - cx)
	}
}

func (tk TWEAK) Vertices() [][]float64 {
	c := tk.Cx()
	h := tk.H - c
	b := tk.B * 0.5
	w := tk.Tw * 0.5
	f := tk.Tf
	vertices := make([][]float64, 8)
	vertices[0] = []float64{-h, -w}
	vertices[1] = []float64{-h, w}
	vertices[2] = []float64{c - f, w}
	vertices[3] = []float64{c - f, b}
	vertices[4] = []float64{c, b}
	vertices[5] = []float64{c, -b}
	vertices[6] = []float64{c - f, -b}
	vertices[7] = []float64{c - f, -w}
	return vertices
}

func (tk TWEAK) Breadth(strong bool) float64 {
	if strong {
		return tk.H
	} else {
		return tk.B
	}
}

type CKYOU struct {
	H, B, Tw, Tf float64
}

func NewCKYOU(lis []string) (CKYOU, error) {
	ck := CKYOU{0.0, 0.0, 0.0, 0.0}
	if len(lis) < 4 {
		return ck, NotEnoughArgs("NewCKYOU")
	}
	var val float64
	var err error
	val, err = strconv.ParseFloat(lis[0], 64)
	if err != nil {
		return ck, err
	}
	ck.H = val
	val, err = strconv.ParseFloat(lis[1], 64)
	if err != nil {
		return ck, err
	}
	ck.B = val
	val, err = strconv.ParseFloat(lis[2], 64)
	if err != nil {
		return ck, err
	}
	ck.Tw = val
	val, err = strconv.ParseFloat(lis[3], 64)
	if err != nil {
		return ck, err
	}
	ck.Tf = val
	return ck, nil
}
func (ck CKYOU) String() string {
	return fmt.Sprintf("CKYOU %5.1f %5.1f %4.1f %4.1f", ck.H, ck.B, ck.Tw, ck.Tf)
}
func (ck CKYOU) Description() string {
	return fmt.Sprintf("C-%dx%dx%dx%d(KYOU)[mm]", int(ck.H*10), int(ck.B*10), int(ck.Tw*10), int(ck.Tf*10))
}
func (ck CKYOU) A() float64 {
	return ck.H*ck.B - (ck.H-2*ck.Tf)*(ck.B-ck.Tw)
}
func (ck CKYOU) Asx() float64 {
	return 2.0 * ck.B * ck.Tf / 1.5
}
func (ck CKYOU) Asy() float64 {
	return (ck.H - 2*ck.Tf) * ck.Tw
}
func (ck CKYOU) Ix() float64 {
	return (ck.B*math.Pow(ck.H, 3.0) - (ck.B-ck.Tw)*math.Pow(ck.H-2*ck.Tf, 3.0)) / 12.0
}
func (ck CKYOU) Cx() float64 {
	return (2.0*ck.B*ck.Tf*0.5*ck.B + (ck.H-2*ck.Tf)*ck.Tw*0.5*ck.Tw) / ck.A()
}
func (ck CKYOU) Iy() float64 {
	cx := ck.Cx()
	return 2.0*ck.Tf*math.Pow(ck.B, 3.0)/12.0 + (ck.H-2*ck.Tf)*math.Pow(ck.Tw, 3.0)/12.0 + 2.0*ck.B*ck.Tf*math.Pow(0.5*ck.B-cx, 2.0) + (ck.H-2*ck.Tf)*ck.Tw*math.Pow(cx-0.5*ck.Tw, 2.0)
}
func (ck CKYOU) J() float64 {
	return 2.0*ck.B*math.Pow(ck.Tf, 3.0)/3.0 + (ck.H-2*ck.Tf)*math.Pow(ck.Tw, 3.0)/3.0
}
func (ck CKYOU) Iw() float64 {
	return math.Pow(ck.H, 2.0) * math.Pow(ck.B, 3.0) * ck.Tf * (3.0*ck.B*ck.Tf + 2.0*ck.H*ck.Tw) / (12.0 * (6.0*ck.B*ck.Tf + ck.H*ck.Tw))
}
func (ck CKYOU) Torsion() float64 {
	if ck.Tf >= ck.Tw {
		return ck.J() / ck.Tf
	} else {
		return ck.J() / ck.Tw
	}
}
func (ck CKYOU) Zx() float64 {
	return ck.Ix() / ck.H * 2.0
}
func (ck CKYOU) Zy() float64 {
	cx := ck.Cx()
	if cx >= ck.B*0.5 {
		return ck.Iy() / cx
	} else {
		return ck.Iy() / (ck.B - cx)
	}
}

func (ck CKYOU) Vertices() [][]float64 {
	h := ck.H * 0.5
	c := ck.Cx()
	b := ck.B - c
	w := ck.Tw - c
	f := ck.Tf
	vertices := make([][]float64, 8)
	vertices[0] = []float64{-c, -h}
	vertices[1] = []float64{b, -h}
	vertices[2] = []float64{b, -(h - f)}
	vertices[3] = []float64{w, -(h - f)}
	vertices[4] = []float64{w, h - f}
	vertices[5] = []float64{b, h - f}
	vertices[6] = []float64{b, h}
	vertices[7] = []float64{-c, h}
	return vertices
}

func (ck CKYOU) Breadth(strong bool) float64 {
	if strong {
		return ck.B
	} else {
		return ck.H
	}
}

type CWEAK struct {
	H, B, Tw, Tf float64
}

func NewCWEAK(lis []string) (CWEAK, error) {
	cw := CWEAK{0.0, 0.0, 0.0, 0.0}
	if len(lis) < 4 {
		return cw, NotEnoughArgs("NewCWEAK")
	}
	var val float64
	var err error
	val, err = strconv.ParseFloat(lis[0], 64)
	if err != nil {
		return cw, err
	}
	cw.H = val
	val, err = strconv.ParseFloat(lis[1], 64)
	if err != nil {
		return cw, err
	}
	cw.B = val
	val, err = strconv.ParseFloat(lis[2], 64)
	if err != nil {
		return cw, err
	}
	cw.Tw = val
	val, err = strconv.ParseFloat(lis[3], 64)
	if err != nil {
		return cw, err
	}
	cw.Tf = val
	return cw, nil
}
func (cw CWEAK) String() string {
	return fmt.Sprintf("CWEAK %5.1f %5.1f %4.1f %4.1f", cw.H, cw.B, cw.Tw, cw.Tf)
}
func (cw CWEAK) Description() string {
	return fmt.Sprintf("C-%dx%dx%dx%d(WEAK)[mm]", int(cw.H*10), int(cw.B*10), int(cw.Tw*10), int(cw.Tf*10))
}
func (cw CWEAK) A() float64 {
	return cw.H*cw.B - (cw.H-2*cw.Tf)*(cw.B-cw.Tw)
}
func (cw CWEAK) Asx() float64 {
	return (cw.H - 2*cw.Tf) * cw.Tw
}
func (cw CWEAK) Asy() float64 {
	return 2.0 * cw.B * cw.Tf / 1.5
}
func (cw CWEAK) Cy() float64 {
	return (2.0*cw.B*cw.Tf*0.5*cw.B + (cw.H-2*cw.Tf)*cw.Tw*0.5*cw.Tw) / cw.A()
}
func (cw CWEAK) Ix() float64 {
	cx := cw.Cy()
	return 2.0*cw.Tf*math.Pow(cw.B, 3.0)/12.0 + (cw.H-2*cw.Tf)*math.Pow(cw.Tw, 3.0)/12.0 + 2.0*cw.B*cw.Tf*math.Pow(0.5*cw.B-cx, 2.0) + (cw.H-2*cw.Tf)*cw.Tw*math.Pow(cx-0.5*cw.Tw, 2.0)
}
func (cw CWEAK) Iy() float64 {
	return (cw.B*math.Pow(cw.H, 3.0) - (cw.B-cw.Tw)*math.Pow(cw.H-2*cw.Tf, 3.0)) / 12.0
}
func (cw CWEAK) J() float64 {
	return 2.0*cw.B*math.Pow(cw.Tf, 3.0)/3.0 + (cw.H-2*cw.Tf)*math.Pow(cw.Tw, 3.0)/3.0
}
func (cw CWEAK) Iw() float64 {
	return math.Pow(cw.H, 2.0) * math.Pow(cw.B, 3.0) * cw.Tf * (3.0*cw.B*cw.Tf + 2.0*cw.H*cw.Tw) / (12.0 * (6.0*cw.B*cw.Tf + cw.H*cw.Tw))
}
func (cw CWEAK) Torsion() float64 {
	if cw.Tf >= cw.Tw {
		return cw.J() / cw.Tf
	} else {
		return cw.J() / cw.Tw
	}
}
func (cw CWEAK) Zx() float64 {
	cx := cw.Cy()
	if cx >= cw.B*0.5 {
		return cw.Ix() / cx
	} else {
		return cw.Ix() / (cw.B - cx)
	}
}
func (cw CWEAK) Zy() float64 {
	return cw.Iy() / cw.H * 2.0
}

func (cw CWEAK) Vertices() [][]float64 {
	h := cw.H * 0.5
	c := cw.Cy()
	b := cw.B - c
	w := cw.Tw - c
	f := cw.Tf
	vertices := make([][]float64, 8)
	vertices[0] = []float64{-h, -c}
	vertices[1] = []float64{-h, b}
	vertices[2] = []float64{-(h - f), b}
	vertices[3] = []float64{-(h - f), w}
	vertices[4] = []float64{h - f, w}
	vertices[5] = []float64{h - f, b}
	vertices[6] = []float64{h, b}
	vertices[7] = []float64{h, -c}
	return vertices
}

func (cw CWEAK) Breadth(strong bool) float64 {
	if strong {
		return cw.H
	} else {
		return cw.B
	}
}

type PLATE struct {
	H, B float64
}

func NewPLATE(lis []string) (PLATE, error) {
	pl := PLATE{0.0, 0.0}
	if len(lis) < 2 {
		return pl, NotEnoughArgs("NewPLATE")
	}
	var val float64
	var err error
	val, err = strconv.ParseFloat(lis[0], 64)
	if err != nil {
		return pl, err
	}
	pl.H = val
	val, err = strconv.ParseFloat(lis[1], 64)
	if err != nil {
		return pl, err
	}
	pl.B = val
	return pl, nil
}
func (pl PLATE) String() string {
	return fmt.Sprintf("PLATE %5.1f %5.1f", pl.H, pl.B)
}
func (pl PLATE) Description() string {
	return fmt.Sprintf("%dx%d[mm]", int(pl.H*10), int(pl.B*10))
}
func (pl PLATE) A() float64 {
	return pl.H * pl.B
}
func (pl PLATE) Asx() float64 {
	return pl.H * pl.B / 1.5
}
func (pl PLATE) Asy() float64 {
	return pl.H * pl.B / 1.5
}
func (pl PLATE) Ix() float64 {
	return pl.B * math.Pow(pl.H, 3.0) / 12.0
}
func (pl PLATE) Iy() float64 {
	return pl.H * math.Pow(pl.B, 3.0) / 12.0
}
func (pl PLATE) J() float64 {
	var h, b float64
	if pl.H >= pl.B {
		h = pl.H
		b = pl.B
	} else {
		h = pl.B
		b = pl.H
	}
	if h > 10.0*b {
		return h * math.Pow(b, 3.0) / 3.0
	} else {
		return math.Pi / 16.0 * math.Pow(b, 3.0) * math.Pow(h, 3.0) / (math.Pow(b, 2.0) + math.Pow(h, 2.0))
	}
}
func (pl PLATE) Iw() float64 {
	return 0.0
}
func (pl PLATE) Torsion() float64 {
	if pl.H >= pl.B {
		return pl.H * math.Pow(pl.B, 2.0) / 3.0
	} else {
		return pl.B * math.Pow(pl.H, 2.0) / 3.0
	}
}
func (pl PLATE) Zx() float64 {
	return pl.B * math.Pow(pl.H, 2.0) / 6.0
}
func (pl PLATE) Zy() float64 {
	return pl.H * math.Pow(pl.B, 2.0) / 6.0
}

func (pl PLATE) Vertices() [][]float64 {
	h := pl.H * 0.5
	b := pl.B * 0.5
	vertices := make([][]float64, 10)
	vertices[0] = []float64{-b, -h}
	vertices[1] = []float64{b, -h}
	vertices[2] = []float64{b, h}
	vertices[3] = []float64{-b, h}
	vertices[4] = nil
	vertices[5] = []float64{-b, -h}
	vertices[6] = []float64{b, h}
	vertices[7] = nil
	vertices[8] = []float64{b, -h}
	vertices[9] = []float64{-b, h}
	return vertices
}

func (pl PLATE) Breadth(strong bool) float64 {
	if strong {
		return pl.B
	} else {
		return pl.H
	}
}

type ANGLE struct {
	H, B, Tw, Tf float64
}

func NewANGLE(lis []string) (ANGLE, error) {
	an := ANGLE{0.0, 0.0, 0.0, 0.0}
	if len(lis) < 4 {
		return an, NotEnoughArgs("NewANGLE")
	}
	var val float64
	var err error
	val, err = strconv.ParseFloat(lis[0], 64)
	if err != nil {
		return an, err
	}
	an.H = val
	val, err = strconv.ParseFloat(lis[1], 64)
	if err != nil {
		return an, err
	}
	an.B = val
	val, err = strconv.ParseFloat(lis[2], 64)
	if err != nil {
		return an, err
	}
	an.Tw = val
	val, err = strconv.ParseFloat(lis[3], 64)
	if err != nil {
		return an, err
	}
	an.Tf = val
	return an, nil
}
func (an ANGLE) String() string {
	return fmt.Sprintf("ANGLE %5.1f %5.1f %4.1f %4.1f", an.H, an.B, an.Tw, an.Tf)
}
func (an ANGLE) Description() string {
	return fmt.Sprintf("L-%dx%dx%dx%d[mm]", int(an.H*10), int(an.B*10), int(an.Tw*10), int(an.Tf*10))
}
func (an ANGLE) A() float64 {
	return an.H*an.Tw + an.B*an.Tf - an.Tw*an.Tf
}
func (an ANGLE) Asx() float64 {
	return an.B * an.Tf / 1.5
}
func (an ANGLE) Asy() float64 {
	return an.H * an.Tw / 1.5
}
func (an ANGLE) Cx() float64 {
	return (an.B*an.Tf*0.5*an.B + (an.H-an.Tf)*an.Tw*0.5*an.Tw) / an.A()
}
func (an ANGLE) Cy() float64 {
	return (an.H*an.Tw*0.5*an.H + (an.B-an.Tw)*an.Tf*0.5*an.Tf) / an.A()
}
func (an ANGLE) Ix() float64 {
	cy := an.Cy()
	return (an.B-an.Tw)*math.Pow(an.Tf, 3.0)/12.0 + an.Tw*math.Pow(an.H, 3.0)/12.0 + (an.B-an.Tw)*an.Tf*math.Pow(cy-0.5*an.Tf, 2.0) + an.H*an.Tw*math.Pow(0.5*an.H-cy, 2.0)
}
func (an ANGLE) Iy() float64 {
	cx := an.Cx()
	return (an.H-an.Tf)*math.Pow(an.Tw, 3.0)/12.0 + an.Tf*math.Pow(an.B, 3.0)/12.0 + (an.H-an.Tf)*an.Tw*math.Pow(cx-0.5*an.Tw, 2.0) + an.B*an.Tf*math.Pow(0.5*an.B-cx, 2.0)
}
func (an ANGLE) Imin() float64 {
	cx := an.Cx()
	cy := an.Cy()
	Ix := an.Ix()
	Iy := an.Iy()
	Ixy := math.Abs(0.25 * ((math.Pow(cx, 2.0)-math.Pow(an.B-cx, 2.0))*(math.Pow(-cy+an.Tf, 2.0)-math.Pow(cy, 2.0)) + (math.Pow(cx, 2.0)-math.Pow(cx-an.Tw, 2.0))*(math.Pow(an.H-cy, 2.0)-math.Pow(-cy+an.Tf, 2.0))))
	var theta float64
	if Ix == Iy {
		theta = 0.25 * math.Pi
	} else {
		theta = 0.5 * (math.Atan(2.0 * Ixy / (Iy - Ix)))
	}
	Iu := Ix*math.Pow(math.Sin(theta), 2.0) + Iy*math.Pow(math.Cos(theta), 2.0) + Ixy*math.Sin(2.0*theta)
	Iv := Iy*math.Pow(math.Sin(theta), 2.0) + Ix*math.Pow(math.Cos(theta), 2.0) - Ixy*math.Sin(2.0*theta)
	if Iu > Iv {
		return Iv
	} else {
		return Iu
	}
}
func (an ANGLE) J() float64 {
	return an.B*math.Pow(an.Tf, 3.0)/3.0 + (an.H-an.Tf)*math.Pow(an.Tw, 3.0)/3.0
}
func (an ANGLE) Iw() float64 {
	return 0.0
}
func (an ANGLE) Torsion() float64 {
	if an.Tf >= an.Tw {
		return an.J() / an.Tf
	} else {
		return an.J() / an.Tw
	}
}
func (an ANGLE) Zx() float64 {
	cy := an.Cy()
	if cy >= an.H*0.5 {
		return an.Ix() / cy
	} else {
		return an.Ix() / (an.H - cy)
	}
}
func (an ANGLE) Zy() float64 {
	cx := an.Cx()
	if cx >= an.B*0.5 {
		return an.Iy() / cx
	} else {
		return an.Iy() / (an.B - cx)
	}
}

func (an ANGLE) Vertices() [][]float64 {
	cx := an.Cx()
	cy := an.Cy()
	h := an.H - cy
	b := an.B - cx
	w := an.Tw - cx
	f := an.Tf - cy
	vertices := make([][]float64, 6)
	vertices[0] = []float64{-cx, -cy}
	vertices[1] = []float64{b, -cy}
	vertices[2] = []float64{b, f}
	vertices[3] = []float64{w, f}
	vertices[4] = []float64{w, h}
	vertices[5] = []float64{-cx, h}
	return vertices
}

func (an ANGLE) Breadth(strong bool) float64 {
	if strong {
		return an.B
	} else {
		return an.H
	}
}

type SAREA struct {
	Area float64
}

func NewSAREA(lis []string) (SAREA, error) {
	sa := SAREA{0.0}
	if len(lis) < 1 {
		return sa, NotEnoughArgs("NewSAREA")
	}
	var val float64
	var err error
	val, err = strconv.ParseFloat(lis[0], 64)
	if err != nil {
		return sa, err
	}
	sa.Area = val
	return sa, nil
}
func (sa SAREA) String() string {
	return fmt.Sprintf("SAREA %5.1f", sa.Area)
}
func (sa SAREA) Description() string {
	return fmt.Sprintf("%d[mm2]", int(sa.Area*100))
}
func (sa SAREA) A() float64 {
	return sa.Area
}
func (sa SAREA) Asx() float64 {
	return sa.Area * 0.5
}
func (sa SAREA) Asy() float64 {
	return sa.Area * 0.5
}
func (sa SAREA) Ix() float64 {
	return sa.Area * 4.0 * math.Pi
}
func (sa SAREA) Iy() float64 {
	return sa.Area * 4.0 * math.Pi
}
func (sa SAREA) J() float64 {
	return sa.Area * 8.0 * math.Pi
}
func (sa SAREA) Iw() float64 {
	return 0.0
}
func (sa SAREA) Torsion() float64 {
	return math.Pow(sa.Area, 1.5) / (4.0 * math.Sqrt(math.Pi))
}
func (sa SAREA) Zx() float64 {
	return math.Pow(sa.Area, 1.5) / (4.0 * math.Sqrt(math.Pi))
}
func (sa SAREA) Zy() float64 {
	return math.Pow(sa.Area, 1.5) / (4.0 * math.Sqrt(math.Pi))
}

func (sa SAREA) Vertices() [][]float64 {
	d := math.Sqrt(sa.Area/math.Pi) * 2.0
	val := math.Pi / 8.0
	theta := 0.0
	vertices := make([][]float64, 16)
	for i := 0; i < 16; i++ {
		c := math.Cos(theta)
		s := math.Sin(theta)
		vertices[i] = []float64{d * c, d * s}
		theta += val
	}
	return vertices
}

func (sa SAREA) Breadth(strong bool) float64 {
	return math.Sqrt(sa.Area/math.Pi) * 2.0
}

type THICK struct {
	Thickness float64
}

func NewTHICK(lis []string) (THICK, error) {
	th := THICK{0.0}
	if len(lis) < 1 {
		return th, NotEnoughArgs("NewTHICK")
	}
	var val float64
	var err error
	val, err = strconv.ParseFloat(lis[0], 64)
	if err != nil {
		return th, err
	}
	th.Thickness = val
	return th, nil
}
func (th THICK) String() string {
	return fmt.Sprintf("THICK %5.1f", th.Thickness)
}
func (th THICK) Description() string {
	return fmt.Sprintf("%d[mm]", int(th.Thickness*10))
}
func (th THICK) A() float64 {
	return 0.0
}
func (th THICK) Asx() float64 {
	return 0.0
}
func (th THICK) Asy() float64 {
	return 0.0
}
func (th THICK) Ix() float64 {
	return 0.0
}
func (th THICK) Iy() float64 {
	return 0.0
}
func (th THICK) J() float64 {
	return 0.0
}
func (th THICK) Iw() float64 {
	return 0.0
}
func (th THICK) Torsion() float64 {
	return 0.0
}
func (th THICK) Zx() float64 {
	return 0.0
}
func (th THICK) Zy() float64 {
	return 0.0
}

func (th THICK) Vertices() [][]float64 {
	vertices := make([][]float64, 4)
	b := 100.0
	vertices[0] = []float64{b * 0.5, th.Thickness * 0.5}
	vertices[1] = []float64{b * 0.5, -th.Thickness * 0.5}
	vertices[2] = []float64{-b * 0.5, -th.Thickness * 0.5}
	vertices[3] = []float64{-b * 0.5, th.Thickness * 0.5}
	return vertices
}

func (th THICK) Breadth(strong bool) float64 {
	return 100.0
}

type SGirder struct {
	SColumn
}

func NewSGirder(num int, shape Shape, material Steel) *SGirder {
	sc := NewSColumn(num, shape, material)
	sc.Etype = "GIRDER"
	return &SGirder{*sc}
}
func (sg *SGirder) TypeString() string {
	return "Ｓ　大梁"
}

type SBrace struct {
	SColumn
}

func NewSBrace(num int, shape Shape, material Steel) *SBrace {
	sc := NewSColumn(num, shape, material)
	sc.Etype = "BRACE"
	return &SBrace{*sc}
}
func (sg *SBrace) TypeString() string {
	return "Ｓ　筋違"
}

type SWall struct {
	Steel
	Shape
	num   int
	name  string
	Wrect []float64
}

func NewSWall(num int, shape Shape, material Steel) *SWall {
	return &SWall{
		Steel: material,
		Shape: shape,
		num:   num,
		name:  "",
		Wrect: make([]float64, 2),
	}
}
func (sw *SWall) Num() int {
	return sw.num
}
func (sw *SWall) TypeString() string {
	return "Ｓ　壁　"
}
func (sw *SWall) Snapshot() SectionRate {
	r := NewSWall(sw.num, sw.Shape, sw.Steel)
	r.name = sw.name
	for i := 0; i < 2; i++ {
		r.Wrect[i] = sw.Wrect[i]
	}
	return sw
}
func (sw *SWall) String() string {
	var rtn bytes.Buffer
	rtn.WriteString(fmt.Sprintf("CODE %3d S WALL %57s\n", sw.num, fmt.Sprintf("\"%s\"", sw.name)))
	line2 := fmt.Sprintf("         %%-29s %%s %%%ds\n", 35-len(sw.Steel.Name))
	rtn.WriteString(fmt.Sprintf(line2, sw.Shape.String(), sw.Steel.Name, fmt.Sprintf("\"%s\"", sw.Shape.Description())))
	return rtn.String()
}
func (sw *SWall) Name() string {
	return sw.name
}
func (sw *SWall) SetName(name string) {
	sw.name = name
}
func (sw *SWall) SetValue(name string, vals []float64) {
	switch name {
	case "WRECT":
		sw.Wrect = vals
	}
}
func (sw *SWall) Factor(p string) float64 {
	switch p {
	default:
		return 0.0
	case "L":
		return 1.0
	case "X", "Y", "S":
		return 1.5
	}
}
func (sw *SWall) Fs(cond *Condition) float64 {
	return sw.F / (1.5 * math.Sqrt(3)) * sw.Factor(cond.Period)
}
func (sw *SWall) Thickness() float64 {
	return sw.Shape.(THICK).Thickness
}
func (sw *SWall) Na(cond *Condition) float64 {
	fs := sw.Fs(cond)
	r := 1.0 // TODO: set windowrate
	Qa := r * sw.Thickness() * cond.Length * fs
	return 0.5 * Qa
}
func (sw *SWall) Qa(cond *Condition) float64 {
	return 0.0
}
func (sw *SWall) Ma(cond *Condition) float64 {
	return 0.0
}
func (sw *SWall) Mza(cond *Condition) float64 {
	return 0.0
}

func (sw *SWall) Amount() Amount {
	return nil
}

type CShape interface {
	String() string
	Bound(int) float64
	Breadth(bool) float64
	Area() float64
	Ix() float64
	Iy() float64
	J() float64
	Height(bool) float64
	Vertices() [][]float64
}

// TODO: implement CCircle

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
func (cr CRect) Ix() float64 {
	return cr.Breadth(true) * math.Pow(cr.Height(true), 3.0) / 12.0
}
func (cr CRect) Iy() float64 {
	return cr.Breadth(false) * math.Pow(cr.Height(false), 3.0) / 12.0
}
func (cr CRect) J() float64 {
	b := cr.Breadth(true)
	h := cr.Height(true)
	return math.Pi / 16.0 * math.Pow(b, 3.0) * math.Pow(h, 3.0) / (math.Pow(b, 2.0) + math.Pow(h, 2.0))
}

func (cr CRect) Vertices() [][]float64 {
	vertices := make([][]float64, 4)
	vertices[0] = []float64{cr.Left, cr.Lower}
	vertices[1] = []float64{cr.Right, cr.Lower}
	vertices[2] = []float64{cr.Right, cr.Upper}
	vertices[3] = []float64{cr.Left, cr.Upper}
	return vertices
}

// RCColumn
type RCColumn struct {
	Concrete
	CShape
	num    int
	name   string
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
	case "FC18":
		rc.Concrete = FC18
	case "FC24":
		rc.Concrete = FC24
	case "FC27":
		rc.Concrete = FC27
	case "FC30":
		rc.Concrete = FC30
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
	return "ＲＣ柱　"
}
func (rc *RCColumn) Snapshot() SectionRate {
	r := NewRCColumn(rc.num)
	r.Concrete = rc.Concrete
	r.CShape = rc.CShape
	r.name = rc.name
	r.Nreins = rc.Nreins
	r.Reins = make([]Reinforce, r.Nreins)
	for i, rf := range rc.Reins {
		r.Reins[i] = rf
	}
	r.Hoops = rc.Hoops
	for i := 0; i < 2; i++ {
		r.XFace[i] = rc.XFace[i]
		r.YFace[i] = rc.YFace[i]
	}
	return r
}
func (rc *RCColumn) Name() string {
	return rc.name
}
func (rc *RCColumn) SetName(name string) {
	rc.name = name
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
				cond.Buffer.WriteString(fmt.Sprintf("#     1. Neutral Axis is outside of section\n"))
				cond.Buffer.WriteString(fmt.Sprintf("#        Ma is determined by concrete.\n"))
			}
			return xn, fc, nil
		} else { // NeutralAxis is outside of section, Ma is determined by reinforcement
			num := 0.5*ft*b*math.Pow(h, 2.0) + NCS*ft*rc.LiAi(cond) - NCS*ryc*cond.N
			den := ft*b*h - NCS*cond.N + NCS*ft*rc.Ai()
			xn = num / den
			if cond.Verbose {
				cond.Buffer.WriteString(fmt.Sprintf("#     2. Neutral Axis is outside of section\n"))
				cond.Buffer.WriteString(fmt.Sprintf("#        Ma is determined by reinforcement.\n"))
			}
			return xn, xn / (NCS * (xn - ryc)) * ft, nil
		}
	} else {
		k1 := 0.5 * fc * b
		k2 := NCS*fc*rc.Ai() - cond.N
		k3 := -NCS * fc * rc.LiAi(cond)
		ryt := rc.FarSideReins(cond)
		D1 := k2*k2 - 4.0*k1*k3
		if D1 >= 0.0 {
			xn := (-k2 + math.Sqrt(D1)) / (2.0 * k1)
			if xn >= 0.0 {
				if (ryt-xn)/xn*fc*NCS <= ft { // NeutralAxis is inside of section, Ma is determined by concrete
					if cond.Verbose {
						cond.Buffer.WriteString(fmt.Sprintf("#     3. Neutral Axis is inside of section\n"))
						cond.Buffer.WriteString(fmt.Sprintf("#        Ma is determined by concrete.\n"))
					}
					return xn, fc, nil
				} else { // NeutralAxis is inside of section, Ma is determined by reinforcement
					k1 := 0.5 * ft * b
					k2 := NCS*ft*rc.Ai() + NCS*cond.N
					k3 := -NCS*ft*rc.LiAi(cond) - NCS*ryt*cond.N
					D2 := k2*k2 - 4.0*k1*k3
					if D2 >= 0.0 {
						xn := (-k2 + math.Sqrt(D2)) / (2.0 * k1)
						if xn >= 0.0 {
							if cond.Verbose {
								cond.Buffer.WriteString(fmt.Sprintf("#     4. Neutral Axis is inside of section\n"))
								cond.Buffer.WriteString(fmt.Sprintf("#        Ma is determined by reinforcement.\n"))
							}
							return xn, xn / (NCS * (ryt - xn)) * ft, nil
						}
					}
				}
			}
		}
		num := ft*rc.LiAi(cond) + ryt*cond.N
		den := ft*rc.Ai() + cond.N
		xn = num / den
		if cond.Verbose {
			cond.Buffer.WriteString(fmt.Sprintf("#     5. Neutral Axis is outside of section\n"))
			cond.Buffer.WriteString(fmt.Sprintf("#        Ma is determined by reinforcement.\n"))
		}
		return xn, -xn / (NCS * (ryt - xn)) * ft, nil
	}
}
func (rc *RCColumn) Nmax(cond *Condition) float64 {
	fc := rc.Fc(cond)
	b := rc.Breadth(cond.Strong)
	h := rc.Height(cond.Strong)
	return fc*b*h + NCS*fc*rc.Ai()
}
func (rc *RCColumn) Nmin(cond *Condition) float64 {
	if rc.Nreins == 0 {
		return 0.0
	}
	ft := rc.Reins[0].Ft(cond)
	return -ft * rc.Ai()
}
func (rc *RCColumn) Na(cond *Condition) float64 {
	if cond.Compression {
		if cond.Verbose {
			cond.Buffer.WriteString(fmt.Sprintf("#     許容圧縮応力度: Fc= %.3f [tf/cm2]\n", rc.Fc(cond)))
		}
		return rc.Fc(cond) * rc.Area()
	} else {
		if rc.Reins == nil {
			return 0.0
		}
		area := 0.0
		ft := 0.0
		rtn := 0.0
		for _, r := range rc.Reins {
			area += r.Area
			ft = r.Ft(cond)
			rtn += r.Area * r.Ft(cond)
		}
		if cond.Verbose {
			cond.Buffer.WriteString(fmt.Sprintf("#     鉄筋総断面積: at= %.3f [cm2]\n#     鉄筋許容引張応力度 Ft= %.3f [tf/cm2]\n", area, ft))
		}
		return rtn
	}
}
func (rc *RCColumn) Alpha(d float64, cond *Condition) float64 {
	alpha := 4.0 / (math.Abs(cond.M*100.0/(cond.Q*d)) + 1.0)
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
	if cond.Verbose {
		cond.Buffer.WriteString(fmt.Sprintf("#     許容せん断応力度: b=%.3f, d=%.3f, α=%.3f, Fs=%.3f [tf/cm2]\n", b, d, alpha, fs))
	}
	switch cond.Period {
	default:
		fmt.Println("unknown period")
		return 0.0
	case "L":
		return 7 / 8.0 * b * d * alpha * fs
	case "X", "Y", "S":
		var pw float64
		if cond.Strong { // for Qy
			pw = rc.Hoops.Ps[1]
		} else { // for Qx
			pw = rc.Hoops.Ps[0]
		}
		if pw < 0.002 {
			return 7 / 8.0 * b * d * 2.0 / 3.0 * alpha * fs
		} else if pw > 0.012 {
			pw = 0.012
		}
		if cond.Verbose {
			cond.Buffer.WriteString(fmt.Sprintf("#     せん断補強筋比: pw=%.6f\n", pw))
		}
		return 7 / 8.0 * b * d * (2.0/3.0*alpha*fs + 0.5*rc.Hoops.Ftw(cond)*(pw-0.002))
	}
}
func (rc *RCColumn) Ma(cond *Condition) float64 {
	b := rc.Breadth(cond.Strong)
	h := rc.Height(cond.Strong)
	xn, sigma, err := rc.NeutralAxis(cond)
	if err != nil {
		return 0.0
	}
	if cond.Verbose {
		cond.Buffer.WriteString(fmt.Sprintf("#     中立軸: xn= %.3f [cm]\n#     許容応力度: σ= %.3f [tf/cm2]\n", xn, sigma))
	}
	if xn >= h {
		return (sigma/xn*(b*h*(3.0*math.Pow(xn, 2.0)-3.0*xn*h+math.Pow(h, 2.0))/3.0+NCS*(math.Pow(xn, 2.0)*rc.Ai()-2.0*xn*rc.LiAi(cond)+rc.Li2Ai(cond))) - cond.N*(xn-h/2.0)) * 0.01 // [tfm]
	} else if xn <= 0 {
		return -(NCS*sigma/xn*(math.Pow(xn, 2.0)*rc.Ai()-2.0*xn*rc.LiAi(cond)+rc.Li2Ai(cond)) + cond.N*(xn-h/2.0)) * 0.01 // [tfm]
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
	// aw := 0.7133
	// lw := 15.0
	kaburi := 5.0
	b0 := b - kaburi*2.0 - dw
	d0 := d - kaburi*2.0 - dw
	A0 := b0 * d0
	var T1, T2, T3 float64
	if b >= d {
		T1 = b * d * d * fs * 4.0 / 3.0 / 100.0 // [tfm]
	} else {
		T1 = b * b * d * fs * 4.0 / 3.0 / 100.0 // [tfm]
	}
	// T2 = aw * 2.0 * wft * A0 / lw / 100.0                // [tfm]
	T2 = wft * A0 * rc.Hoops.Ps[1] * b / 100.0           // [tfm]
	T3 = rc.Ai() * 2.0 * ft * A0 / (2*b0 + 2*d0) / 100.0 // [tfm]
	if cond.Verbose {
		cond.Buffer.WriteString(fmt.Sprintf("#     許容ねじりモーメント: T1= %.3f [tfm] T2= %.3f [tfm] T3= %.3f [tfm]\n", T1, T2, T3))
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
func (rc *RCColumn) Amount() Amount {
	a := NewAmount()
	area := rc.Area()
	a["REINS"] = (rc.Ai() + area*(rc.Hoops.Ps[0]+rc.Hoops.Ps[1])) * 0.0001
	a["CONCRETE"] = area * 0.0001
	a["FORMWORK"] = (rc.Breadth(true) + rc.Height(true)) * 2 * 0.01
	return a
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
	alpha := 4.0 / (math.Abs(cond.M*100.0/(cond.Q*d)) + 1.0)
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
		cond.Buffer.WriteString(fmt.Sprintf("#     許容せん断応力度: b=%.3f, d=%.3f, α=%.3f, Fs=%.3f [tf/cm2]\n", b, d, alpha, fs))
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
			return 7 / 8.0 * b * d * fs
		} else if pw > 0.006 {
			pw = 0.006
		}
		if cond.Verbose {
			cond.Buffer.WriteString(fmt.Sprintf("#     せん断補強筋比: pw=%.6f\n", pw))
		}
		return 7 / 8.0 * b * d * (alpha*fs + 0.5*rg.Hoops.Ftw(cond)*(pw-0.002))
	case "X", "Y", "S":
		if pw < 0.002 {
			// fmt.Printf("shortage in pw: %.6f\n", pw)
			return 7 / 8.0 * b * d * fs
		} else if pw > 0.012 {
			pw = 0.012
		}
		if cond.Verbose {
			cond.Buffer.WriteString(fmt.Sprintf("#     せん断補強筋比: pw=%.6f\n", pw))
		}
		return 7 / 8.0 * b * d * (alpha*fs + 0.5*rg.Hoops.Ftw(cond)*(pw-0.002))
	}
}
func (rg *RCGirder) Amount() Amount {
	a := NewAmount()
	area := rg.Area()
	a["REINS"] = (rg.Ai() + area*(rg.Hoops.Ps[0]+rg.Hoops.Ps[1])) * 0.0001
	a["CONCRETE"] = area * 0.0001
	a["FORMWORK"] = (rg.Breadth(true) + rg.Height(true)*2) * 0.01
	return a
}

type RCWall struct {
	Concrete
	num      int
	name     string
	Thick    float64
	Srein    float64
	Material SD
	Wrect    []float64
	XFace    []float64
	YFace    []float64
}

func NewRCWall(num int) *RCWall {
	rw := new(RCWall)
	rw.num = num
	rw.Wrect = make([]float64, 2)
	rw.XFace = make([]float64, 2)
	rw.YFace = make([]float64, 2)
	return rw
}
func (rw *RCWall) SetSrein(lis []string) error {
	val, err := strconv.ParseFloat(lis[0], 64)
	if err != nil {
		return err
	}
	rw.Srein = val
	rw.Material = SetSD(lis[1])
	return nil
}
func (rw *RCWall) SetConcrete(lis []string) error {
	switch lis[0] {
	case "THICK":
		val, err := strconv.ParseFloat(lis[1], 64)
		if err != nil {
			return err
		}
		rw.Thick = val
	}
	switch lis[2] {
	case "FC18":
		rw.Concrete = FC18
	case "FC24":
		rw.Concrete = FC24
	case "FC27":
		rw.Concrete = FC27
	case "FC30":
		rw.Concrete = FC30
	case "FC36":
		rw.Concrete = FC36
	}
	return nil
}
func (rw *RCWall) Num() int {
	return rw.num
}
func (rw *RCWall) TypeString() string {
	return "ＲＣ壁　"
}
func (rw *RCWall) Snapshot() SectionRate {
	r := NewRCWall(rw.num)
	r.name = rw.name
	r.Thick = rw.Thick
	for i := 0; i < 2; i++ {
		r.Wrect[i] = rw.Wrect[i]
		r.XFace[i] = rw.XFace[i]
		r.YFace[i] = rw.YFace[i]
	}
	return rw
}
func (rw *RCWall) String() string {
	return ""
}
func (rw *RCWall) Name() string {
	return rw.name
}
func (rw *RCWall) SetName(name string) {
	rw.name = name
}
func (rw *RCWall) SetValue(name string, vals []float64) {
	switch name {
	case "WRECT":
		rw.Wrect = vals
	case "XFACE":
		rw.XFace = vals
	case "YFACE":
		rw.YFace = vals
	}
}
func (rw *RCWall) Factor(p string) float64 {
	switch p {
	default:
		return 0.0
	case "L":
		return 1.0
	case "X", "Y", "S":
		return 2.0
	}
}
func (rw *RCWall) Fs(cond *Condition) float64 {
	var rtn float64
	f1 := rw.fc / 30.0
	f2 := 0.005 + rw.fc/100.0
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
func (rw *RCWall) Na(cond *Condition) float64 {
	fs := rw.Fs(cond)
	var Qc, Qw, Qa float64
	r := 1.0 // TODO: set windowrate
	Qc = r * rw.Thick * cond.Length * fs
	switch cond.Period {
	case "L":
		Qa = Qc
	case "X", "Y", "S":
		Qa = Qc
		le := cond.Width - (rw.XFace[0] + rw.XFace[1])
		if le >= 0.0 {
			l0 := cond.Width
			Qw = r * rw.Thick * rw.Srein * cond.Length * le / l0 * rw.Material.Fs
			if Qw > Qc {
				Qa = Qw
			}
		}
	}
	return 0.5 * Qa
}
func (rw *RCWall) Qa(cond *Condition) float64 {
	return 0.0
}
func (rw *RCWall) Ma(cond *Condition) float64 {
	return 0.0
}
func (rw *RCWall) Mza(cond *Condition) float64 {
	return 0.0
}

func (rw *RCWall) Amount() Amount {
	a := NewAmount()
	thick := rw.Thick
	a["REINS"] = thick * 0.01 * rw.Srein
	a["CONCRETE"] = thick * 0.01
	a["FORMWORK"] = 2.0
	return a
}

type RCSlab struct {
	RCWall
}

func NewRCSlab(num int) *RCSlab {
	rs := new(RCSlab)
	rs.num = num
	rs.Wrect = make([]float64, 2)
	rs.XFace = make([]float64, 2)
	rs.YFace = make([]float64, 2)
	return rs
}
func (rs *RCSlab) TypeString() string {
	return "ＲＣ床　"
}

func (rs *RCSlab) Amount() Amount {
	a := NewAmount()
	thick := rs.Thick
	a["REINS"] = thick * 0.01 * rs.Srein
	a["CONCRETE"] = thick * 0.01
	a["FORMWORK"] = 1.0
	return a
}

func SetSD(name string) SD {
	switch name {
	default:
		return SD295
	case "SD295":
		return SD295
	case "SD345":
		return SD345
	case "SD390":
		return SD390
	}
}

type WoodColumn struct {
	Wood
	Shape
	num      int
	Etype    string
	name     string
	XFace    []float64
	YFace    []float64
	BBLength []float64
	BTLength []float64
	BBFactor []float64
	BTFactor []float64
	multi    float64
}

func NewWoodColumn(num int, shape Shape, material Wood) *WoodColumn {
	wc := &WoodColumn{
		Wood:     material,
		Shape:    shape,
		num:      num,
		Etype:    "COLUMN",
		name:     "",
		XFace:    nil,
		YFace:    nil,
		BBLength: nil,
		BTLength: nil,
		BBFactor: nil,
		BTFactor: nil,
		multi:    1.0,
	}
	return wc
}
func (wc *WoodColumn) Num() int {
	return wc.num
}
func (wc *WoodColumn) TypeString() string {
	return "木　柱　"
}
func (wc *WoodColumn) Snapshot() SectionRate {
	w := NewWoodColumn(wc.num, wc.Shape, wc.Wood)
	w.Etype = wc.Etype
	w.name = wc.name
	if wc.XFace != nil {
		w.XFace = make([]float64, 2)
		w.XFace[0] = wc.XFace[0]
		w.XFace[1] = wc.XFace[1]
	}
	if wc.YFace != nil {
		w.YFace = make([]float64, 2)
		w.YFace[0] = wc.YFace[0]
		w.YFace[1] = wc.YFace[1]
	}
	if wc.BBLength != nil {
		w.BBLength = make([]float64, 2)
		w.BBLength[0] = wc.BBLength[0]
		w.BBLength[1] = wc.BBLength[1]
	}
	if wc.BTLength != nil {
		w.BTLength = make([]float64, 2)
		w.BTLength[0] = wc.BTLength[0]
		w.BTLength[1] = wc.BTLength[1]
	}
	if wc.BBFactor != nil {
		w.BBFactor = make([]float64, 2)
		w.BBFactor[0] = wc.BBFactor[0]
		w.BBFactor[1] = wc.BBFactor[1]
	}
	if wc.BTFactor != nil {
		w.BTFactor = make([]float64, 2)
		w.BTFactor[0] = wc.BTFactor[0]
		w.BTFactor[1] = wc.BTFactor[1]
	}
	return w
}
func (wc *WoodColumn) Name() string {
	return wc.name
}
func (wc *WoodColumn) SetName(name string) {
	wc.name = name
}
func (wc *WoodColumn) SetValue(name string, vals []float64) {
	switch name {
	case "XFACE":
		wc.XFace = vals
	case "YFACE":
		wc.YFace = vals
	case "BBLEN":
		wc.BBLength = vals
	case "BTLEN":
		wc.BTLength = vals
	case "BBFAC":
		wc.BBFactor = vals
	case "BTFAC":
		wc.BTFactor = vals
	case "MULTI":
		wc.multi = vals[0]
	}
}
func (wc *WoodColumn) String() string {
	var rtn bytes.Buffer
	rtn.WriteString(fmt.Sprintf("CODE %3d WOOD %s %54s\n", wc.num, wc.Etype, fmt.Sprintf("\"%s\"", wc.name)))
	line2 := fmt.Sprintf("         %%-29s %%s %%%ds\n", 35-len(wc.Wood.Name))
	rtn.WriteString(fmt.Sprintf(line2, wc.Shape.String(), wc.Wood.Name, fmt.Sprintf("\"%s\"", wc.Shape.Description())))
	if wc.XFace != nil {
		rtn.WriteString(fmt.Sprintf("         XFACE %5.1f %5.1f %48s\n", wc.XFace[0], wc.XFace[1], fmt.Sprintf("\"FACE LENGTH Mx:HEAD= %.0f,TAIL= %.0f[cm]\"", wc.XFace[0], wc.XFace[1])))
	} else {
		rtn.WriteString("         XFACE   0.0   0.0             \"FACE LENGTH Mx:HEAD= 0,TAIL= 0[cm]\"\n")
	}
	if wc.YFace != nil {
		rtn.WriteString(fmt.Sprintf("         YFACE %5.1f %5.1f %48s\n", wc.YFace[0], wc.YFace[1], fmt.Sprintf("\"FACE LENGTH My:HEAD= %.0f,TAIL= %.0f[cm]\"", wc.XFace[0], wc.XFace[1])))
	} else {
		rtn.WriteString("         YFACE   0.0   0.0             \"FACE LENGTH My:HEAD= 0,TAIL= 0[cm]\"\n")
	}
	if wc.BBLength != nil {
		rtn.WriteString(fmt.Sprintf("         BBLEN %5.1f %5.1f\n", wc.BBLength[0], wc.BBLength[1]))
	} else if wc.BBFactor != nil {
		rtn.WriteString(fmt.Sprintf("         BBFAC %5.1f %5.1f\n", wc.BBFactor[0], wc.BBFactor[1]))
	}
	if wc.BTLength != nil {
		rtn.WriteString(fmt.Sprintf("         BTLEN %5.1f %5.1f\n", wc.BTLength[0], wc.BTLength[1]))
	} else if wc.BTFactor != nil {
		rtn.WriteString(fmt.Sprintf("         BTFAC %5.1f %5.1f\n", wc.BTFactor[0], wc.BTFactor[1]))
	}
	return rtn.String()
}
func (wc *WoodColumn) Factor(p string) float64 {
	switch p {
	default:
		return 0.0
	case "L":
		return 1.1 / 3.0
	case "X", "Y", "S":
		return 2.0 / 3.0
	}
}
func (wc *WoodColumn) Lk(length float64, strong bool) float64 {
	var ind int
	if strong {
		ind = 0
	} else {
		ind = 1
	}
	if wc.BBLength != nil && wc.BBLength[ind] > 0.0 {
		return wc.BBLength[ind]
	} else if wc.BBFactor != nil && wc.BBFactor[ind] > 0.0 {
		return length * wc.BBFactor[ind]
	} else {
		return length
	}
}
func (wc *WoodColumn) Lb(length float64, strong bool) float64 {
	var ind int
	if strong {
		ind = 0
	} else {
		ind = 1
	}
	if wc.BTLength != nil && wc.BTLength[ind] > 0.0 {
		return wc.BTLength[ind]
	} else if wc.BTFactor != nil && wc.BTFactor[ind] > 0.0 {
		return length * wc.BTFactor[ind]
	} else {
		return length
	}
}
func (wc *WoodColumn) Fc(cond *Condition) float64 {
	var rtn float64
	var lambda float64
	lx := wc.Lk(cond.Length, true)
	ly := wc.Lk(cond.Length, false)
	lambda_x := lx / math.Sqrt(wc.Ix()/wc.A())
	lambda_y := ly / math.Sqrt(wc.Iy()/wc.A())
	if lambda_x >= lambda_y {
		lambda = lambda_x
	} else {
		lambda = lambda_y
	}
	if lambda < 30.0 {
		rtn = 1.0
	} else if lambda < 100.0 {
		rtn = 1.3 - 0.01*lambda
	} else {
		rtn = 3000.0 / (lambda * lambda)
	}
	if cond.Verbose {
		cond.Buffer.WriteString(fmt.Sprintf("#     座屈長さ[cm]: Lkx=%.3f, Lky=%.3f\n", lx, ly))
		check := ""
		if lambda > 150 {
			check = " λ>150"
		}
		cond.Buffer.WriteString(fmt.Sprintf("#     細長比: λx=%.3f, λy=%.3f: λ=%.3f%s\n", lambda_x, lambda_y, lambda, check))
		cond.Buffer.WriteString(fmt.Sprintf("#     許容圧縮応力度: Fc=%.3f [tf/cm2]\n", rtn))
	}
	return rtn * wc.Factor(cond.Period) * wc.fc
}
func (wc *WoodColumn) Ft(cond *Condition) float64 {
	return wc.ft * wc.Factor(cond.Period)
}
func (wc *WoodColumn) Fs(cond *Condition) float64 {
	return wc.fs * wc.Factor(cond.Period)
}
func (wc *WoodColumn) Fb(cond *Condition) float64 {
	return wc.fb * wc.Factor(cond.Period)
}
func (wc *WoodColumn) Na(cond *Condition) float64 {
	if cond.Compression {
		return wc.Fc(cond) * wc.A() * wc.multi
	} else {
		return wc.Ft(cond) * wc.A() * wc.multi
	}
}
func (wc *WoodColumn) Qa(cond *Condition) float64 {
	f := wc.Fs(cond)
	if cond.Strong { // for Qy
		return f * wc.Asy() * wc.multi
	} else { // for Qx
		return f * wc.Asx() * wc.multi
	}
}
func (wc *WoodColumn) Ma(cond *Condition) float64 {
	f := wc.Fb(cond)
	if cond.Strong {
		return f * wc.Zx() * 0.01 * wc.multi // [tfm]
	} else {
		return f * wc.Zy() * 0.01 * wc.multi // [tfm]
	}
}
func (wc *WoodColumn) Mza(cond *Condition) float64 {
	return wc.Fs(cond) * wc.Torsion() * 0.01 * wc.multi // [tfm]
}

func (wc *WoodColumn) Vertices() [][]float64 {
	return wc.Shape.Vertices()
}

func (wc *WoodColumn) Amount() Amount {
	a := NewAmount()
	a["WOOD"] = wc.A() * 0.0001
	return a
}

type WoodGirder struct {
	WoodColumn
}

func NewWoodGirder(num int, shape Shape, material Wood) *WoodGirder {
	wc := NewWoodColumn(num, shape, material)
	wc.Etype = "GIRDER"
	return &WoodGirder{*wc}
}
func (wg *WoodGirder) TypeString() string {
	return "木　大梁"
}

type WoodWall struct {
	Wood
	num   int
	name  string
	Thick float64
	Wrect []float64
}

func NewWoodWall(num int) *WoodWall {
	ww := new(WoodWall)
	ww.num = num
	ww.Wrect = make([]float64, 2)
	return ww
}
func (ww *WoodWall) SetWood(lis []string) error {
	switch lis[0] {
	case "THICK":
		val, err := strconv.ParseFloat(lis[1], 64)
		if err != nil {
			return err
		}
		ww.Thick = val
	}
	switch lis[2] {
	case "GOHAN":
		ww.Wood = GOHAN
	}
	return nil
}
func (ww *WoodWall) Num() int {
	return ww.num
}
func (ww *WoodWall) TypeString() string {
	return "木　壁　"
}
func (ww *WoodWall) Snapshot() SectionRate {
	r := NewWoodWall(ww.num)
	r.name = ww.name
	r.Thick = ww.Thick
	for i := 0; i < 2; i++ {
		r.Wrect[i] = ww.Wrect[i]
	}
	return ww
}
func (ww *WoodWall) String() string {
	return ""
}
func (ww *WoodWall) Name() string {
	return ww.name
}
func (ww *WoodWall) SetName(name string) {
	ww.name = name
}
func (ww *WoodWall) SetValue(name string, vals []float64) {
	switch name {
	case "WRECT":
		ww.Wrect = vals
	}
}
func (ww *WoodWall) Factor(p string) float64 {
	switch p {
	default:
		return 0.0
	case "L":
		return 1.0
	case "X", "Y", "S":
		return 2.0
	}
}
func (ww *WoodWall) Fs(cond *Condition) float64 {
	rtn := ww.fs // [tf/cm2]
	switch cond.Period {
	default:
		rtn = 0.0
	case "L":
		rtn *= 1.0
	case "X", "Y", "S":
		rtn *= 2.0
	}
	return rtn
}
func (ww *WoodWall) Na(cond *Condition) float64 {
	fs := ww.Fs(cond)
	r := 1.0 // TODO: set windowrate
	Qa := r * ww.Thick * cond.Length * fs
	return 0.5 * Qa
}
func (ww *WoodWall) Qa(cond *Condition) float64 {
	return 0.0
}
func (ww *WoodWall) Ma(cond *Condition) float64 {
	return 0.0
}
func (ww *WoodWall) Mza(cond *Condition) float64 {
	return 0.0
}

func (ww *WoodWall) Amount() Amount {
	return nil
}

type WoodSlab struct {
	WoodWall
}

func NewWoodSlab(num int) *WoodSlab {
	ww := NewWoodWall(num)
	return &WoodSlab{*ww}
}

func (ws *WoodSlab) TypeString() string {
	return "木　床　"
}

// Condition
type Condition struct {
	Period      string
	Length      float64
	Width       float64
	Compression bool
	Strong      bool
	Positive    bool
	FbOld       bool
	N           float64
	M           float64
	Q           float64
	Sign        float64
	Verbose     bool
	Buffer      *bytes.Buffer
	Nfact       float64
	Qfact       float64
	Mfact       float64
	Bfact       float64
	Wfact       float64
	Skipshort   bool
	Temporary   bool
}

func NewCondition() *Condition {
	return &Condition{
		Period:      "L",
		Length:      0.0,
		Width:       0.0,
		Compression: false,
		Strong:      true,
		Positive:    true,
		FbOld:       false,
		N:           0.0,
		M:           0.0,
		Q:           0.0,
		Sign:        1.0,
		Verbose:     false,
		Buffer:      new(bytes.Buffer),
		Nfact:       1.0,
		Qfact:       2.0,
		Mfact:       1.0,
		Bfact:       1.0,
		Wfact:       2.0,
		Skipshort:   false,
		Temporary:   false,
	}
}

func Rate1(sr SectionRate, stress []float64, cond *Condition) ([]float64, string, error) {
	if len(stress) < 12 {
		return nil, "", errors.New("Rate: Not enough number of Stress")
	}
	rate := make([]float64, 12)
	fa := make([]float64, 12)
	var ind int
	var otp bytes.Buffer
	verbose := false
	for i := 0; i < 2; i++ {
		if cond.Verbose {
			if i == 0 {
				cond.Buffer.WriteString(fmt.Sprintf("# 算定詳細\n"))
				verbose = cond.Verbose
			} else {
				cond.Verbose = false
			}
		}
		if i == 0 {
			cond.N = stress[6*i]
			cond.M = stress[6*i+4]
			cond.Q = stress[6*i+2]
		} else {
			cond.N = -stress[6*i]
			cond.M = -stress[6*i+4]
			cond.Q = -stress[6*i+2]
		}
		ind = 6*i + 0
		cond.Compression = cond.N >= 0.0
		na := sr.Na(cond)
		if na == 0.0 && stress[ind] != 0.0 {
			return rate, "", ZeroAllowableError{"Na"}
		}
		rate[ind] = math.Abs(stress[ind] / na)
		fa[ind] = na
		cond.Strong = true
		ind = 6*i + 2
		cond.Positive = cond.M >= 0.0
		qay := sr.Qa(cond)
		if qay == 0.0 && stress[ind] != 0.0 {
			return rate, "", ZeroAllowableError{"Qay"}
		}
		rate[ind] = math.Abs(stress[ind] / qay)
		fa[ind] = qay
		ind = 6*i + 4
		max := sr.Ma(cond)
		if max == 0.0 && stress[ind] != 0.0 {
			rate[ind] = 10.0
		} else {
			rate[ind] = math.Abs(stress[ind] / max)
		}
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
		ind = 6*i + 1
		qax := sr.Qa(cond)
		if qax == 0.0 && stress[ind] != 0.0 {
			return rate, "", ZeroAllowableError{"Qax"}
		}
		rate[ind] = math.Abs(stress[ind] / qax)
		fa[ind] = qax
		ind = 6*i + 5
		may := sr.Ma(cond)
		if may == 0.0 && stress[ind] != 0.0 {
			rate[ind] = 10.0
		} else {
			rate[ind] = math.Abs(stress[ind] / may)
		}
		fa[ind] = may
		ind = 6*i + 3
		maz := sr.Mza(cond)
		if maz == 0.0 && stress[ind] != 0.0 {
			return rate, "", ZeroAllowableError{"Maz"}
		}
		rate[ind] = math.Abs(stress[ind] / maz)
		fa[ind] = maz
	}
	cond.Verbose = verbose
	for i := 0; i < 6; i++ {
		for j := 0; j < 2; j++ {
			otp.WriteString(fmt.Sprintf(" %8.3f(%8.2f)", stress[6*j+i], stress[6*j+i]*SI))
			if i == 0 || i == 3 {
				break
			}
		}
	}
	if cond.Verbose {
		otp.WriteString("\n" + cond.Buffer.String())
		cond.Buffer.Reset()
	} else {
		otp.WriteString("\n")
	}
	otp.WriteString("     許容値:")
	for i := 0; i < 6; i++ {
		for j := 0; j < 2; j++ {
			otp.WriteString(fmt.Sprintf(" %8.3f(%8.2f)", fa[6*j+i], fa[6*j+i]*SI))
			if i == 0 || i == 3 {
				break
			}
		}
	}
	otp.WriteString("\n     安全率:")
	for i := 0; i < 6; i++ {
		for j := 0; j < 2; j++ {
			otp.WriteString(fmt.Sprintf(" %8.3f          ", rate[6*j+i]))
			if i == 0 || i == 3 {
				break
			}
		}
	}
	otp.WriteString("\n")
	return rate, otp.String(), nil
}

func Rate2(sr SectionRate, stress float64, cond *Condition) (float64, string, error) {
	var rate float64
	var otp bytes.Buffer
	cond.Compression = stress >= 0.0
	na := sr.Na(cond)
	if na == 0.0 && stress != 0.0 {
		return rate, "", ZeroAllowableError{"Na"}
	}
	rate = math.Abs(stress / na)
	otp.WriteString(fmt.Sprintf(" %8.3f(%8.2f)\n", stress, stress*SI))
	if cond.Verbose {
		otp.WriteString(cond.Buffer.String())
		cond.Buffer.Reset()
	}
	otp.WriteString(fmt.Sprintf("     許容値: %8.3f(%8.2f)\n     安全率: %8.3f\n", na, na*SI, rate))
	return rate, otp.String(), nil
}
