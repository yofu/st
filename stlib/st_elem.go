package st

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/yofu/dxf/drawing"
	"github.com/yofu/st/arclm"
)

var ETYPES = []string{"NULL", "COLUMN", "GIRDER", "BRACE", "TRUSS", "WBRACE", "SBRACE", "WALL", "SLAB"}
var ETYPENAME = []string{"", "柱", "梁", "ブレース", "トラス", "壁ブレース", "床ブレース", "壁", "床"}
var StressName = []string{"Nz", "Qx", "Qy", "Mz", "Mx", "My"}

const (
	NULL = iota
	COLUMN
	GIRDER
	BRACE
	TRUSS
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
	RIGID = []bool{false, false, false, false, false, false}
	PIN   = []bool{false, false, false, false, true, true}
)

type Elem struct {
	Frame *Frame
	Num   int
	Enods int
	Enod  []*Node
	Sect  *Sect
	Etype int
	Cang  float64
	Bonds []*Bond
	Cmq   []float64
	Wrect []float64

	Skip []bool

	Rate    map[string][]float64
	MaxRate []float64
	Stress  map[string]map[int][]float64

	Values    map[string]float64
	Prestress float64

	Phinge map[string]map[int]bool

	Strong []float64
	Weak   []float64

	Children []*Elem
	Parent   *Elem
	Eldest   bool
	Chain    *Chain

	SrcalTex string

	Condition *Condition

	hide bool
	Lock bool
}

func NewLineElem(ns []*Node, sect *Sect, etype int) *Elem {
	if etype >= WALL {
		return nil
	}
	el := new(Elem)
	el.Enods = 2
	el.Enod = ns[:2]
	el.Sect = sect
	el.Etype = etype
	el.Skip = make([]bool, 3)
	el.Bonds = make([]*Bond, 12)
	el.Cmq = make([]float64, 12)
	el.Stress = make(map[string]map[int][]float64)
	el.Values = make(map[string]float64)
	el.Phinge = make(map[string]map[int]bool)
	el.Strong = make([]float64, 3)
	el.Weak = make([]float64, 3)
	el.SetPrincipalAxis()
	return el
}

func NewPlateElem(ns []*Node, sect *Sect, etype int) *Elem {
	if COLUMN <= etype && etype <= SBRACE {
		return nil
	}
	el := new(Elem)
	for _, en := range ns {
		if en != nil {
			el.Enods++
		} else {
			break
		}
	}
	el.Enod = ns[:el.Enods]
	el.Sect = sect
	el.Etype = etype
	el.Skip = make([]bool, 3)
	el.Values = make(map[string]float64)
	el.Children = make([]*Elem, 2)
	el.Wrect = make([]float64, 2)
	return el
}

func (elem *Elem) Number() int {
	return elem.Num
}

func (elem *Elem) Enode(ind int) int {
	return elem.Enod[ind].Num
}

func (elem *Elem) Snapshot(frame *Frame) *Elem {
	if elem == nil {
		return nil
	}
	if elem.Etype == WBRACE || elem.Etype == SBRACE {
		return elem
	}
	var el *Elem
	enod := make([]*Node, elem.Enods)
	for i, en := range elem.Enod {
		enod[i] = frame.Nodes[en.Num]
	}
	if elem.IsLineElem() {
		el = NewLineElem(enod, frame.Sects[elem.Sect.Num], elem.Etype)
		el.Cang = elem.Cang
		for i := 0; i < 12; i++ {
			el.Bonds[i] = elem.Bonds[i]
			el.Cmq[i] = elem.Cmq[i]
		}
		el.MaxRate = make([]float64, len(elem.MaxRate))
		for i, r := range elem.MaxRate {
			el.MaxRate[i] = r
		}
		for i := 0; i < 3; i++ {
			el.Strong[i] = elem.Strong[i]
			el.Weak[i] = elem.Weak[i]
		}
		if elem.Parent != nil {
			el.Parent = frame.Elems[elem.Parent.Num]
		}
		el.Eldest = elem.Eldest
	} else {
		el = NewPlateElem(enod, frame.Sects[elem.Sect.Num], elem.Etype)
		if elem.Wrect != nil {
			for i := 0; i < 2; i++ {
				el.Wrect[i] = elem.Wrect[i]
			}
		}
	}
	for i := 0; i < 3; i++ {
		el.Skip[i] = elem.Skip[i]
	}
	el.Num = elem.Num
	el.Frame = frame
	el.hide = elem.hide
	el.Lock = elem.Lock
	return el
}

func (elem *Elem) PrincipalAxis(cang float64) ([]float64, []float64, error) {
	return PrincipalAxis(elem.Direction(true), cang)
}

func (elem *Elem) SetPrincipalAxis() error {
	s, w, err := elem.PrincipalAxis(elem.Cang)
	if err != nil {
		return err
	}
	elem.Strong = s
	elem.Weak = w
	return nil
}

func (elem *Elem) AxisToCang(vector []float64, strong bool) (float64, error) {
	vector = Normalize(vector)
	d := elem.Direction(true)
	uv := Dot(d, vector, 3)
	if uv == 1.0 {
		elem.Cang = 0.0
		elem.SetPrincipalAxis()
		return elem.Cang, nil
	}
	newvec := make([]float64, 3)
	for i := 0; i < 3; i++ {
		newvec[i] = vector[i] - uv*d[i]
	}
	newvec = Normalize(newvec)
	d1, d2, err := elem.PrincipalAxis(0.0)
	if err != nil {
		return 0.0, err
	}
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
	return elem.Cang, nil
}

func (elem *Elem) ElementAxes(deformed bool, period string, dfact float64) ([]float64, []float64, []float64, []float64) {
	if !deformed {
		return elem.MidPoint(), elem.Direction(true), elem.Strong, elem.Weak
	} else {
		position := elem.DeformedMidPoint(period, dfact)
		d := elem.DeformedDirection(true, period, dfact)
		strong, weak, err := PrincipalAxis(d, elem.Cang)
		if err != nil {
			return position, d, elem.Strong, elem.Weak
		} else {
			return position, d, strong, weak
		}
	}
}

type Elems []*Elem

func (e Elems) Len() int { return len(e) }
func (e Elems) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

type ElemByNum struct{ Elems }

func (e ElemByNum) Less(i, j int) bool { return e.Elems[i].Num < e.Elems[j].Num }

type ElemBySumEnod struct{ Elems }

func (e ElemBySumEnod) Less(i, j int) bool {
	var sum1, sum2 int
	for _, en1 := range e.Elems[i].Enod {
		sum1 += en1.Num
	}
	for _, en2 := range e.Elems[j].Enod {
		sum2 += en2.Num
	}
	return sum1 < sum2
}

type ElemByMidPoint struct{ Elems }

func (e ElemByMidPoint) Less(i, j int) bool {
	m1 := e.Elems[i].MidPoint()
	m2 := e.Elems[j].MidPoint()
	for n := 0; n < 3; n++ {
		if m1[n] == m2[n] {
			continue
		}
		return m1[n] < m2[n]
	}
	return false
}

func SortedElem(els map[int]*Elem, compare func(*Elem) float64) []*Elem {
	l := len(els)
	elems := make(map[float64][]*Elem, l)
	keys := make([]float64, l)
	num := 0
	for _, el := range els {
		val := compare(el)
		if _, ok := elems[val]; !ok {
			elems[val] = make([]*Elem, 0)
		}
		elems[val] = append(elems[val], el)
		num++
	}
	sortedelems := make([]*Elem, num)
	i := 0
	for k := range elems {
		keys[i] = k
		i++
	}
	keys = keys[:i]
	sort.Float64s(keys)
	i = 0
	for _, k := range keys {
		for _, el := range elems[k] {
			sortedelems[i] = el
			i++
		}
	}
	return sortedelems
}

func Etype(str string) int {
	for i, j := range ETYPES {
		if j == str {
			return i
		}
	}
	return 0
}

func (elem *Elem) setEtype(str string) error {
	for i, j := range ETYPES {
		if j == str {
			elem.Etype = i
			return nil
		}
	}
	return errors.New("setEtype: Etype not found")
}

func (elem *Elem) setSkip(str string) error {
	if len(str) < 3 {
		return errors.New("setSkip: unknown skip format")
	}
	for i := 0; i < 3; i++ {
		if str[i] == '1' {
			elem.Skip[i] = true
		} else {
			elem.Skip[i] = false
		}
	}
	return nil
}

func (elem *Elem) IsSkipAt(ind int) bool {
	return elem.Skip[ind]
}

func (elem *Elem) IsSkipAny() bool {
	for i := 0; i < 3; i ++ {
		if elem.Skip[i] {
			return true
		}
	}
	return false
}

func (elem *Elem) IsSkipAll() bool {
	for i := 0; i < 3; i ++ {
		if !elem.Skip[i] {
			return false
		}
	}
	return true
}

func (elem *Elem) SkipString() string {
	rtn := make([]string, 3)
	for i := 0; i < 3; i++ {
		if elem.Skip[i] {
			rtn[i] = "1"
		} else {
			rtn[i] = "0"
		}
	}
	return strings.Join(rtn , "")
}

// IsLineElem reports whether the element is line element or not.
func (elem *Elem) IsLineElem() bool {
	return elem.Etype <= SBRACE && elem.Enods == 2
}

func (elem *Elem) Hide() {
	elem.hide = true
}

func (elem *Elem) Show() {
	elem.hide = false
}

func (elem *Elem) IsHidden(show *Show) bool {
	if elem.hide {
		return true
	}
	for _, en := range elem.Enod {
		if en.IsHidden(show) {
			return true
		}
	}
	if !show.Etype[elem.Etype] {
		return true
	}
	if b, ok := show.Sect[elem.Sect.Num]; ok {
		if !b {
			return true
		}
	}
	return false
}

func (elem *Elem) IsNotEditable(show *Show) bool {
	return elem == nil || elem.IsHidden(show) || elem.Lock
}

func (elem *Elem) InpString() string {
	var rtn bytes.Buffer
	if elem.IsLineElem() {
		rtn.WriteString(fmt.Sprintf("ELEM %5d ESECT %3d ENODS %d ENOD", elem.Num, elem.Sect.Num, elem.Enods))
		for i := 0; i < elem.Enods; i++ {
			rtn.WriteString(fmt.Sprintf(" %d", elem.Enod[i].Num))
		}
		rtn.WriteString(" BONDS ")
		for i := 0; i < elem.Enods; i++ {
			for j := 0; j < 6; j++ {
				if elem.Bonds[6*i+j] == nil {
					rtn.WriteString(" 0")
				} else {
					rtn.WriteString(fmt.Sprintf(" %d", elem.Bonds[6*i+j].Num))
				}
			}
			if i < elem.Enods-1 {
				rtn.WriteString(" ")
			} else {
				rtn.WriteString("\n")
			}
		}
		rtn.WriteString(fmt.Sprintf("           CANG %7.5f\n", elem.Cang))
		rtn.WriteString("           CMQ ")
		for i := 0; i < elem.Enods; i++ {
			for j := 0; j < 6; j++ {
				rtn.WriteString(fmt.Sprintf(" %15.12f", elem.Cmq[6*i+j]))
			}
			if i < elem.Enods-1 {
				rtn.WriteString(" ")
			} else {
				rtn.WriteString("\n")
			}
		}
		rtn.WriteString(fmt.Sprintf("           TYPE %s\n", ETYPES[elem.Etype]))
		if elem.Prestress != 0.0 {
			rtn.WriteString(fmt.Sprintf("           PREST %.3f\n", elem.Prestress))
		}
		if elem.IsSkipAny() {
			rtn.WriteString("           SKIP ")
			rtn.WriteString(elem.SkipString())
			rtn.WriteString("\n")
		}
		return rtn.String()
	} else {
		rtn.WriteString(fmt.Sprintf("ELEM %5d ESECT %3d ENODS %d ENOD", elem.Num, elem.Sect.Num, elem.Enods))
		for i := 0; i < elem.Enods; i++ {
			rtn.WriteString(fmt.Sprintf(" %d", elem.Enod[i].Num))
		}
		rtn.WriteString(" BONDS ")
		for i := 0; i < elem.Enods; i++ {
			for j := 0; j < 6; j++ {
				rtn.WriteString(fmt.Sprintf(" %d", 0))
			}
			if i < elem.Enods-1 {
				rtn.WriteString(" ")
			} else {
				rtn.WriteString("\n")
			}
		}
		rtn.WriteString(fmt.Sprintf("                     EBANS 1 EBAN 1 BNODS %d BNOD", elem.Enods))
		for i := 0; i < elem.Enods; i++ {
			rtn.WriteString(fmt.Sprintf(" %d", elem.Enod[i].Num))
		}
		rtn.WriteString(fmt.Sprintf("\n           TYPE %s\n", ETYPES[elem.Etype]))
		if elem.Wrect != nil && (elem.Wrect[0] != 0.0 || elem.Wrect[1] != 0.0) {
			rtn.WriteString(fmt.Sprintf("           WRECT %.3f %.3f\n", elem.Wrect[0], elem.Wrect[1]))
		}
		if elem.IsSkipAny() {
			rtn.WriteString("           SKIP ")
			rtn.WriteString(elem.SkipString())
			rtn.WriteString("\n")
		}
		return rtn.String()
	}
}

func (elem *Elem) OutputStress(p string) string {
	var rtn bytes.Buffer
	rtn.WriteString(fmt.Sprintf("%5d %4d %4d", elem.Num, elem.Sect.Num, elem.Enod[0].Num))
	if stress, ok := elem.Stress[p]; ok {
		for _, st := range stress[elem.Enod[0].Num] {
			rtn.WriteString(fmt.Sprintf(" %15.12f", st))
		}
		rtn.WriteString(fmt.Sprintf("\n           %4d", elem.Enod[1].Num))
		for _, st := range stress[elem.Enod[1].Num] {
			rtn.WriteString(fmt.Sprintf(" %15.12f", st))
		}
	} else {
		rtn.WriteString(strings.Repeat(fmt.Sprintf(" %15.12f", 0.0), 6))
		rtn.WriteString(fmt.Sprintf("\n           %4d", elem.Enod[1].Num))
		rtn.WriteString(strings.Repeat(fmt.Sprintf(" %15.12f", 0.0), 6))
	}
	rtn.WriteString("\n")
	return rtn.String()
}

func (elem *Elem) OutputMaxRate() string {
	if elem.MaxRate == nil {
		return ""
	}
	var rat bytes.Buffer
	rat.WriteString(fmt.Sprintf("ELEM: %5d SECT: %4d", elem.Num, elem.Sect.Num))
	for _, r := range elem.MaxRate {
		rat.WriteString(fmt.Sprintf(" %8.5f", r))
	}
	rat.WriteString("\n")
	return rat.String()
}

func (elem *Elem) OutputRateRlt() string {
	if elem.MaxRate == nil {
		return ""
	}
	var rlt bytes.Buffer
	rlt.WriteString(fmt.Sprintf("ELEM: %5d SECT: %4d MAX:", elem.Num, elem.Sect.Num))
	str := []string{"Q/QaL", "Q/QaS", "Q/Qu", "M/MaL", "M/MaS", "M/Mu"}
	for i, r := range elem.MaxRate {
		rlt.WriteString(fmt.Sprintf(" %s=%8.5f", str[i], r))
	}
	rlt.WriteString("\n")
	return rlt.String()
}

func (elem *Elem) OutputRateInformation(long, x1, x2, y1, y2 string, sign float64, alpha []float64, angle float64) (SectionRate, string, error) {
	al, original, err := elem.GetSectionRate()
	if err != nil {
		return al, "", fmt.Errorf("rate is nil: ELEM %d", elem.Num)
	}
	if elem.Condition == nil {
		return al, "", fmt.Errorf("condition is nil")
	}
	stname := []string{"鉛直時Z    :", "水平時X    :", "水平時X負  :", "水平時Y    :", "水平時Y負  :"}
	var pername []string
	if angle != 0.0 {
		pername = []string{"長期       :", "短期P正方向:", "短期P負方向:", "短期Q正方向:", "短期Q負方向:"}
	} else {
		pername = []string{"長期       :", "短期X正方向:", "短期X負方向:", "短期Y正方向:", "短期Y負方向:"}
	}
	var otp, tex bytes.Buffer
	var rate []float64
	var rate2 float64
	calc1 := func(allow SectionRate, cond *Condition, st1, st2, fact []float64, sign float64) ([]float64, error) {
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
		rate, txt, textxt, err := Rate1(allow, stress, cond)
		if err != nil {
			return rate, err
		}
		otp.WriteString(txt)
		tex.WriteString(textxt)
		return rate, nil
	}
	calc1angle := func(allow SectionRate, cond *Condition, st1, st2, st3, fact []float64, sign float64, angle float64, direction int) ([]float64, error) {
		stress := make([]float64, 12)
		theta := math.Pi/180.0 * angle
		if st2 != nil {
			for i := 0; i < 2; i++ {
				for j := 0; j < 6; j++ {
					if direction == 0 {
						stress[6*i+j] = st1[6*i+j] + sign*fact[j]*(st2[6*i+j]*math.Cos(theta)+st3[6*i+j]*math.Sin(theta))
					} else {
						stress[6*i+j] = st1[6*i+j] + sign*fact[j]*(-st2[6*i+j]*math.Sin(theta)+st3[6*i+j]*math.Cos(theta))
					}
				}
			}
		} else {
			stress = st1
		}
		rate, txt, textxt, err := Rate1(allow, stress, cond)
		if err != nil {
			return rate, err
		}
		otp.WriteString(txt)
		tex.WriteString(textxt)
		return rate, nil
	}
	calc2 := func(allow SectionRate, cond *Condition, n1, n2, fact, sign float64) (float64, error) {
		stress := n1 + sign*fact*n2
		rate, txt, textxt, err := Rate2(allow, stress, cond)
		if err != nil {
			return rate, err
		}
		otp.WriteString(txt)
		tex.WriteString(textxt)
		return rate, nil
	}
	calc2angle := func(allow SectionRate, cond *Condition, n1, n2, n3, fact, sign float64, angle float64, direction int) (float64, error) {
		theta := math.Pi/180.0 * angle
		stress := 0.0
		if direction == 0 {
			stress = n1 + sign*fact*(n2*math.Cos(theta)+n3*math.Sin(theta))
		} else {
			stress = n1 + sign*fact*(-n2*math.Sin(theta)+n3*math.Cos(theta))
		}
		rate, txt, textxt, err := Rate2(allow, stress, cond)
		if err != nil {
			return rate, err
		}
		otp.WriteString(txt)
		tex.WriteString(textxt)
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
	factor := make([][]float64, 4)
	if alpha != nil {
		factor[0] = []float64{alpha[0], alpha[0], alpha[0], alpha[0], alpha[0], alpha[0]}
		factor[1] = []float64{alpha[0], alpha[0], alpha[0], alpha[0], alpha[0], alpha[0]}
		factor[2] = []float64{alpha[1], alpha[1], alpha[1], alpha[1], alpha[1], alpha[1]}
		factor[3] = []float64{alpha[1], alpha[1], alpha[1], alpha[1], alpha[1], alpha[1]}
	} else {
		for i := 0; i < 4; i++ {
			factor[i] = []float64{elem.Condition.Nfact, elem.Condition.Qfact, elem.Condition.Qfact, 1.0, elem.Condition.Mfact, elem.Condition.Mfact}
		}
	}
	switch elem.Etype {
	case COLUMN, GIRDER:
		var isrc bool
		switch al.(type) {
		case *RCColumn, *RCGirder:
			isrc = true
		default:
			isrc = false
		}
		if isrc && elem.Condition.Qfact < 1.5 {
			for i := 0; i < 4; i++ {
				factor[i][1] = 1.5
				factor[i][2] = 1.5
			}
		} else {
			for i := 0; i < 4; i++ {
				factor[i][1] = elem.Condition.Qfact
				factor[i][2] = elem.Condition.Qfact
			}
		}
		elem.Condition.Length = elem.Length() * 100.0 // [cm]
		otp.WriteString(strings.Repeat("-", 202))
		otp.WriteString(fmt.Sprintf("\n部材:%d 始端:%d 終端:%d 断面:%d=%s 材長=%.1f[cm] Mx内法=%.1f[cm] My内法=%.1f[cm]", elem.Num, elem.Enod[0].Num, elem.Enod[1].Num, elem.Sect.Num, strings.Replace(al.TypeString(), "　", "", -1), elem.Condition.Length, elem.Condition.Length, elem.Condition.Length))
		if alpha != nil {
			otp.WriteString(fmt.Sprintf(" αx=%.3f αy=%.3f", alpha[0], alpha[1]))
		}
		if angle != 0.0 {
			otp.WriteString(fmt.Sprintf(" θ=%.3f°", angle))
		}
		otp.WriteString("\n応力       :        N                Qxi                Qxj                Qyi                Qyj                 Mt                Mxi                Mxj                Myi                Myj\n")
		tex.WriteString(fmt.Sprintf("\\multicolumn{11}{l}{\\textsb{部材:%d 始端:%d 終端:%d 断面:%d=%s 材長=%.1f[cm] Mx内法=%.1f[cm] My内法=%.1f[cm]}}\\\\\n", elem.Num, elem.Enod[0].Num, elem.Enod[1].Num, elem.Sect.Num, strings.Replace(al.TypeString(), "　", "", -1), elem.Condition.Length, elem.Condition.Length, elem.Condition.Length))
		tex.WriteString("応力       &      $N$&$Q_{xi}$&$Q_{yi}$&$Q_{xj}$&$Q_{yj}$&   $M_t$&$M_{xi}$&$M_{xj}$&$M_{yi}$&$M_{yj}$\\\\\n")
		tex.WriteString("           &     [kN]&    [kN]&    [kN]&    [kN]&    [kN]&   [kNm]&   [kNm]&   [kNm]&   [kNm]&   [kNm]\\\\\n")
		stress := make([][]float64, 5)
		var qlrate, qsrate, qurate, mlrate, msrate, murate float64
		msrates := make([]float64, 4)
		for p, per := range []string{long, x1, x2, y1, y2} {
			stress[p] = make([]float64, 12)
			for i := 0; i < 2; i++ {
				for j := 0; j < 6; j++ {
					stress[p][6*i+j] = elem.ReturnStress(per, i, j)
				}
			}
			if (p == 2 && x1 == x2) || (p == 4 && y1 == y2) {
				continue
			}
			otp.WriteString(stname[p])
			tex.WriteString(strings.Replace(stname[p], ":", "", -1))
			for i := 0; i < 6; i++ {
				for j := 0; j < 2; j++ {
					otp.WriteString(fmt.Sprintf(" %8.3f(%8.2f)", stress[p][6*j+i], stress[p][6*j+i]*SI))
					tex.WriteString(fmt.Sprintf(" &%8.3f", stress[p][6*j+i]*SI))
					if i == 0 || i == 3 {
						break
					}
				}
			}
			otp.WriteString("\n")
			tex.WriteString("\\\\\n")
		}
		otp.WriteString("\n")
		tex.WriteString("\\\\\n")
		if elem.Condition.Verbose {
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
				case *SGirder:
					sh := al.(*SGirder).Shape
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
				case *WoodGirder:
					sh := al.(*WoodGirder).Shape
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
		if elem.Condition.Temporary != "" {
			elem.Condition.Period = elem.Condition.Temporary
		} else {
			elem.Condition.Period = "L"
		}
		otp.WriteString(pername[0])
		tex.WriteString(strings.Replace(pername[0], ":", "", -1))
		rate, err = calc1(al, elem.Condition, stress[0], nil, nil, 1.0)
		if isrc {
			if elem.Condition.RCTorsion {
				qlrate = maxrate(rate[1]+rate[3], rate[2]+rate[3], rate[7]+rate[3], rate[8]+rate[3]) // TODO: use T1,T2 if T1,T2>T3
				mlrate = maxrate(rate[4]+rate[3], rate[5]+rate[3], rate[10]+rate[3], rate[11]+rate[3]) // TODO: use T3 if T1,T2<T3
			} else {
				qlrate = maxrate(rate[1], rate[2], rate[7], rate[8])
				mlrate = maxrate(rate[4], rate[5], rate[10], rate[11])
			}
		} else {
				qlrate = maxrate(rate[1], rate[2], rate[7], rate[8])
			if rate[0] >= 1.0 {
				mlrate = 10.0
			} else {
				if elem.BondState() == PIN_PIN { // 両端ピン柱の場合は軸力の検定比を表示
					mlrate = rate[0]
				} else {
					mlrate = maxrate(rate[4], rate[5], rate[10], rate[11]) / (1.0 - rate[0])
				}
			}
		}
		if elem.Condition.Skipshort {
			otp.WriteString(fmt.Sprintf("\nMAX:Q/QaL=%.5f Q/QaS=%.5f M/MaL=%.5f M/MaS=%.5f\n", qlrate, qsrate, mlrate, msrate))
			tex.WriteString(fmt.Sprintf("\\\\\n\\multicolumn{11}{l}{MAX:$Q/Q_{aL}=%.5f, Q/Q_{aS}=%.5f, M/M_{aL}=%.5f, M/M_{aS}=%.5f$}\\\\\n\\\\ \\hline\n\\\\\n", qlrate, qsrate, mlrate, msrate))
			elem.MaxRate = []float64{qlrate, qsrate, qurate, mlrate, msrate, murate}
			break
		}
		elem.Condition.Period = "S"
		var s float64
		for p := 1; p < 5; p++ {
			if p%2 == 1 {
				otp.WriteString("\n")
				tex.WriteString("\\\\\n")
				s = 1.0
			} else {
				s = sign
			}
			otp.WriteString(pername[p])
			tex.WriteString(strings.Replace(pername[p], ":", "", -1))
			if angle != 0.0 {
				if p <= 2 {
					rate, err = calc1angle(al, elem.Condition, stress[0], stress[p], stress[p+2], factor[p-1], s, angle, 0)
				} else {
					rate, err = calc1angle(al, elem.Condition, stress[0], stress[p-2], stress[p], factor[p-1], s, angle, 1)
				}
			} else {
				rate, err = calc1(al, elem.Condition, stress[0], stress[p], factor[p-1], s)
			}
			if isrc {
				if elem.Condition.RCTorsion {
					qsrate = maxrate(qsrate, rate[1]+rate[3], rate[2]+rate[3], rate[7]+rate[3], rate[8]+rate[3])
					msrates[p-1] = maxrate(rate[4]+rate[3], rate[5]+rate[3], rate[10]+rate[3], rate[11]+rate[3])
				} else {
					qsrate = maxrate(qsrate, rate[1], rate[2], rate[7], rate[8])
					msrates[p-1] = maxrate(rate[4], rate[5], rate[10], rate[11])
				}
			} else {
				qsrate = maxrate(qsrate, rate[1], rate[2], rate[7], rate[8])
				if rate[0] >= 1.0 {
					msrates[p-1] = 10.0
				} else {
					if elem.BondState() == PIN_PIN { // 両端ピン柱の場合は軸力の検定比を表示
						msrates[p-1] = rate[0]
					} else {
						msrates[p-1] = maxrate(rate[4], rate[5], rate[10], rate[11]) / (1.0 - rate[0])
					}
				}
			}
		}
		msrate = maxrate(msrates...)
		otp.WriteString(fmt.Sprintf("\nMAX:Q/QaL=%.5f Q/QaS=%.5f M/MaL=%.5f M/MaS=%.5f\n", qlrate, qsrate, mlrate, msrate))
		tex.WriteString(fmt.Sprintf("\\\\\n\\multicolumn{11}{l}{MAX:$Q/Q_{aL}=%.5f, Q/Q_{aS}=%.5f, M/M_{aL}=%.5f, M/M_{aS}=%.5f$}\\\\\n\\\\ \\hline\n\\\\\n", qlrate, qsrate, mlrate, msrate))
		elem.MaxRate = []float64{qlrate, qsrate, qurate, mlrate, msrate, murate}
	case BRACE, WBRACE, SBRACE:
		var isrc bool
		switch al.(type) {
		case *RCWall, *RCGirder:
			isrc = true
		default:
			isrc = false
		}
		var qlrate, qsrate, qurate float64
		var fact float64
		if elem.Etype == BRACE {
			fact = elem.Condition.Bfact
			elem.Condition.Length = elem.Length() * 100.0 // [cm]
		} else {
			fact = elem.Condition.Wfact
			if bros, ok := elem.Brother(); ok {
				elem.Condition.Length = 0.5 * (elem.Length() + bros.Length()) * 100.0 // [cm]
			} else {
				elem.Condition.Length = elem.Length() * 100.0 // [cm]
			}
		}
		if isrc && fact < 2.0 {
			fact = 2.0
		}
		elem.Condition.Width = elem.Width() * 100.0 // [cm]
		elem.Condition.Height = elem.Height() * 100.0 // [cm]
		otp.WriteString(strings.Repeat("-", 202))
		var sectstring string
		if original {
			sectstring = fmt.Sprintf("◯%d/　%d", elem.OriginalSection().Num, elem.Sect.Num)
		} else {
			sectstring = fmt.Sprintf("　%d/◯%d", elem.OriginalSection().Num, elem.Sect.Num)
		}
		otp.WriteString(fmt.Sprintf("\n部材:%d 始端:%d 終端:%d 断面:%s=%s 材長=%.1f[cm] Mx内法=%.1f[cm] My内法=%.1f[cm]", elem.Num, elem.Enod[0].Num, elem.Enod[1].Num, sectstring, strings.Replace(al.TypeString(), "　", "", -1), elem.Condition.Length, elem.Condition.Length, elem.Condition.Length))
		otp.WriteString("\n応力       :        N\n")
		tex.WriteString(fmt.Sprintf("\\multicolumn{11}{l}{\\textsb{部材:%d 始端:%d 終端:%d 断面:%s=%s 材長=%.1f[cm] Mx内法=%.1f[cm] My内法=%.1f[cm]}}\\\\\n", elem.Num, elem.Enod[0].Num, elem.Enod[1].Num, sectstring, strings.Replace(al.TypeString(), "　", "", -1), elem.Condition.Length, elem.Condition.Length, elem.Condition.Length))
		tex.WriteString("応力       &      $N$\\\\\n")
		if alpha != nil {
			otp.WriteString(fmt.Sprintf(" αx=%.3f αy=%.3f", alpha[0], alpha[1]))
		}
		if angle != 0.0 {
			otp.WriteString(fmt.Sprintf(" θ=%.3f°", angle))
		}
		stress := make([]float64, 5)
		for p, per := range []string{long, x1, x2, y1, y2} {
			if st, ok := elem.Stress[per]; ok {
				stress[p] = st[elem.Enod[0].Num][0]
			}
			if (p == 2 && x1 == x2) || (p == 4 && y1 == y2) {
				continue
			}
			otp.WriteString(stname[p])
			otp.WriteString(fmt.Sprintf(" %8.3f(%8.2f)", stress[p], stress[p]*SI))
			otp.WriteString("\n")
			tex.WriteString(strings.Replace(stname[p], ":", "", -1))
			tex.WriteString(fmt.Sprintf(" &%8.3f\\\\\n", stress[p]*SI))
		}
		otp.WriteString("\n")
		if elem.Condition.Temporary != "" {
			elem.Condition.Period = elem.Condition.Temporary
		} else {
			elem.Condition.Period = "L"
		}
		otp.WriteString(pername[0])
		tex.WriteString(strings.Replace(pername[0], ":", "", -1))
		rate2, err = calc2(al, elem.Condition, stress[0], 0.0, 0.0, 1.0)
		qlrate = rate2
		if elem.Condition.Skipshort {
			otp.WriteString(fmt.Sprintf("\nMAX:Q/QaL=%.5f Q/QaS=%.5f\n", qlrate, qsrate))
			tex.WriteString(fmt.Sprintf("\\\\\n\\multicolumn{11}{l}{MAX:$Q/Q_{aL}=%.5f, Q/Q_{aS}=%.5f$}\\\\\n\\\\ \\hline\n\\\\\n", qlrate, qsrate))
			elem.MaxRate = []float64{qlrate, qsrate, qurate}
			break
		}
		elem.Condition.Period = "S"
		var s float64
		for p := 1; p < 5; p++ {
			if p%2 == 1 {
				otp.WriteString("\n")
				s = 1.0
			} else {
				s = sign
			}
			otp.WriteString(pername[p])
			tex.WriteString(strings.Replace(pername[p], ":", "", -1))
			if angle != 0.0 {
				if p <= 2 {
					rate2, err = calc2angle(al, elem.Condition, stress[0], stress[p], stress[p+2], fact*factor[p-1][0], s, angle, 0)
				} else {
					rate2, err = calc2angle(al, elem.Condition, stress[0], stress[p-2], stress[p], fact*factor[p-1][0], s, angle, 1)
				}
			} else {
				rate2, err = calc2(al, elem.Condition, stress[0], stress[p], fact*factor[p-1][0], s)
			}
			qsrate = maxrate(qsrate, rate2)
		}
		otp.WriteString(fmt.Sprintf("\nMAX:Q/QaL=%.5f Q/QaS=%.5f\n", qlrate, qsrate))
		tex.WriteString(fmt.Sprintf("\\\\\n\\multicolumn{11}{l}{MAX:$Q/Q_{aL}=%.5f, Q/Q_{aS}=%.5f$}\\\\\n\\\\ \\hline\n\\\\\n", qlrate, qsrate))
		elem.MaxRate = []float64{qlrate, qsrate, qurate}
	}
	elem.SrcalTex = tex.String()
	return al, otp.String(), nil
}

func (elem *Elem) Amount() float64 {
	if elem.IsLineElem() {
		return elem.Length()
	} else {
		return elem.Area()
	}
}

func (elem *Elem) Length() float64 {
	sum := 0.0
	for i := 0; i < 3; i++ {
		sum += math.Pow((elem.Enod[1].Coord[i] - elem.Enod[0].Coord[i]), 2)
	}
	return math.Sqrt(sum)
}

func (elem *Elem) DeformedLength(p string, fact float64) float64 {
	sum := 0.0
	for i := 0; i < 3; i++ {
		sum += math.Pow((elem.Enod[1].Coord[i] + elem.Enod[1].ReturnDisp(p, i)*fact - elem.Enod[0].Coord[i] - elem.Enod[0].ReturnDisp(p, i)*fact), 2)
	}
	return math.Sqrt(sum)
}

func (elem *Elem) EdgeLength(ind int) float64 {
	if ind >= elem.Enods-1 {
		return 0.0
	}
	jind := ind + 1
	if jind >= elem.Enods {
		jind -= elem.Enods
	}
	sum := 0.0
	for i := 0; i < 3; i++ {
		sum += math.Pow((elem.Enod[jind].Coord[i] - elem.Enod[ind].Coord[i]), 2)
	}
	return math.Sqrt(sum)
}

func (elem *Elem) Area() float64 {
	if elem.Enods <= 2 {
		return 0.0
	}
	var area float64
	ds := make([]float64, elem.Enods-1)
	vs := make([][]float64, elem.Enods-1)
	for i := 1; i < elem.Enods; i++ {
		for j := 0; j < 3; j++ {
			ds[i-1] += math.Pow((elem.Enod[i].Coord[j] - elem.Enod[0].Coord[j]), 2)
		}
		vs[i-1] = Direction(elem.Enod[0], elem.Enod[i], false)
	}
	for i := 0; i < elem.Enods-2; i++ {
		val := ds[i]*ds[i+1] - math.Pow(Dot(vs[i], vs[i+1], 3), 2)
		if val < 0.0 {
			// val can be very small negative value.
			// This is because 2 nodes are very near.
			// TODO: If val is large negative value, Should 0.5*sqrt(abs(val)) be added to area?
			continue
		}
		area += 0.5 * math.Sqrt(val)
	}
	return area
}

func (elem *Elem) DeformedArea(p string, fact float64) float64 {
	if elem.Enods <= 2 {
		return 0.0
	}
	var area float64
	ds := make([]float64, elem.Enods-1)
	vs := make([][]float64, elem.Enods-1)
	for i := 1; i < elem.Enods; i++ {
		for j := 0; j < 3; j++ {
			ds[i-1] += math.Pow((elem.Enod[i].Coord[j] + elem.Enod[i].ReturnDisp(p, j)*fact - elem.Enod[0].Coord[j] - elem.Enod[0].ReturnDisp(p, j)*fact), 2)
		}
		vs[i-1] = DeformedDirection(elem.Enod[0], elem.Enod[i], false, p)
	}
	for i := 0; i < elem.Enods-2; i++ {
		area += 0.5 * math.Sqrt(ds[i]*ds[i+1]-math.Pow(Dot(vs[i], vs[i+1], 3), 2))
	}
	return area
}

func (elem *Elem) Weight() []float64 {
	rtn := make([]float64, 3)
	a := elem.Amount()
	w := elem.Sect.Weight()
	for i := 0; i < 3; i++ {
		p := elem.ConvertPerpendicularLoad(i, 2)
		rtn[i] = a * (w[i] + p)
	}
	return rtn
}

func (elem *Elem) ConvertPerpendicularLoad(ind int, direction int) float64 {
	if elem.Sect.Perpl == nil || len(elem.Sect.Perpl) < 3 || (elem.Sect.Perpl[1] == 0.0 && elem.Sect.Perpl[2] == 0.0) {
		return 0.0
	}
	if ind < 0 || ind > 2 {
		return 0.0
	}
	if direction < 0 || direction > 2 {
		return 0.0
	}
	edir := elem.Normal(true)
	return edir[direction] * elem.Sect.Perpl[ind]
}

func (elem *Elem) Distribute() error {
	var err error
	w := elem.Sect.Weight() // tf/m or tf/m2
	switch elem.Etype {
	case SLAB:
		if elem.Enods < 3 {
			return fmt.Errorf("Distribute: too few enods: ELEM %d", elem.Num)
		}
		els, factor, err := elem.PlateDivision(false)
		if err != nil {
			return err
		}
		area := 0.0
		for _, el := range els {
			area += el.Area()
		}
		for _, el := range els {
			c := make([]*Cmq, 3)
			for i := 0; i < 3; i++ {
				c[i], err = Polygon(el, w[i]*factor)
				if err != nil {
					return err
				}
			}
			var edge *Elem
			for _, e := range elem.Frame.SearchElem(el.Enod[0], el.Enod[1]) {
				if e == elem {
					continue
				}
				if e.Etype == GIRDER {
					edge = e
					break
				}
			}
			if edge != nil {
				if ok, _ := edge.IsEdgeOfWall(); ok {
					if el.Enod[0].Num == edge.Enod[0].Num {
						edge.DistributeWeight(c, false)
					} else {
						edge.DistributeWeight(c, true)
					}
				} else {
					if el.Enod[0].Num == edge.Enod[0].Num {
						edge.DistributeGirder(c, false)
					} else {
						edge.DistributeGirder(c, true)
					}
				}
			} else {
				el.DistributeWeight(c, false)
			}
		}
		return nil
	case WALL:
		if elem.Enods < 3 {
			return fmt.Errorf("Distribute: too few enods: ELEM %d", elem.Num)
		}
		els, factor, err := elem.PlateDivision(false)
		if err != nil {
			return err
		}
		for _, el := range els {
			c := make([]*Cmq, 3)
			for i := 0; i < 3; i++ {
				c[i], err = Polygon(el, w[i]*factor)
				if err != nil {
					return err
				}
			}
			el.DistributeWeight(c, false)
		}
		return nil
	case GIRDER:
		if elem.Enods > 2 {
			return fmt.Errorf("Distribute: too many enods: ELEM %d", elem.Num)
		}
		l := elem.Length()
		c := make([]*Cmq, 3)
		for i := 0; i < 3; i++ {
			c[i], err = Uniform(l, w[i])
			if err != nil {
				return err
			}
		}
		if ok, _ := elem.IsEdgeOfWall(); ok {
			elem.DistributeWeight(c, false)
		} else {
			elem.DistributeGirder(c, false)
		}
		return nil
	case COLUMN, BRACE:
		l := elem.Length()
		c := make([]*Cmq, 3)
		for i := 0; i < 3; i++ {
			c[i], err = Uniform(l, w[i])
			if err != nil {
				return err
			}
		}
		elem.DistributeWeight(c, false)
		return nil
	default:
		return fmt.Errorf("Distribute: unknown etype: ELEM %d ETYPE %d", elem.Num, elem.Etype)
	}
}

func (elem *Elem) DistributeWeight(c []*Cmq, invert bool) {
	for i := 0; i < 3; i++ {
		if invert {
			elem.Enod[0].Weight[i] += c[i].Qj0
			elem.Enod[1].Weight[i] += c[i].Qi0
		} else {
			elem.Enod[0].Weight[i] += c[i].Qi0
			elem.Enod[1].Weight[i] += c[i].Qj0
		}
	}
}

func (elem *Elem) DistributeGirder(c []*Cmq, invert bool) {
	if invert {
		elem.Cmq[2] += c[1].Qj0
		elem.Cmq[4] -= c[1].Cj
		elem.Cmq[8] += c[1].Qi0
		elem.Cmq[10] -= c[1].Ci
	} else {
		elem.Cmq[2] += c[1].Qi0
		elem.Cmq[4] += c[1].Ci
		elem.Cmq[8] += c[1].Qj0
		elem.Cmq[10] += c[1].Cj
	}
	elem.DistributeWeight(c, invert)
}

func (elem *Elem) IsEdgeOfWall() (bool, error) {
	plates, err := elem.EdgeOf()
	if err != nil {
		return false, err
	}
	for _, plate := range plates {
		if plate.Etype == WALL && plate.IsBraced() {
			return true, nil
		}
	}
	return false, nil
}

func (elem *Elem) Width() float64 {
	sum := 0.0
	if elem.IsLineElem() {
		for i := 0; i < 2; i++ {
			sum += math.Pow((elem.Enod[1].Coord[i] - elem.Enod[0].Coord[i]), 2)
		}
	} else {
		ns := elem.LowerEnod()
		for i := 0; i < 2; i++ {
			sum += math.Pow((ns[1].Coord[i] - ns[0].Coord[i]), 2)
		}
	}
	return math.Sqrt(sum)
}

func (elem *Elem) EffectiveWidth() float64 {
	if elem.IsLineElem() {
		return elem.Width()
	} else {
		x1, x2, err := elem.Sect.Xface(0)
		if err != nil {
			return elem.Width()
		}
		return elem.Width() - (x1 + x2)
	}
}

func (elem *Elem) Height() float64 {
	if elem.IsLineElem() {
		return math.Abs(elem.Enod[1].Coord[2] - elem.Enod[0].Coord[2])
	} else {
		max := -1e+16
		min := 1e+16
		for _, en := range elem.Enod {
			if en.Coord[2] > max {
				max = en.Coord[2]
			}
			if en.Coord[2] < min {
				min = en.Coord[2]
			}
		}
		return max - min
	}
}

func (elem *Elem) PlateSize() (float64, float64) {
	if elem.Enods != 4 {
		return 0.0, 0.0
	}
	w1 := Distance(elem.Enod[0], elem.Enod[1])
	w2 := Distance(elem.Enod[2], elem.Enod[3])
	h1 := Distance(elem.Enod[1], elem.Enod[2])
	h2 := Distance(elem.Enod[3], elem.Enod[0])
	return 0.5 * (w1 + w2), 0.5 * (h1 + h2)
}

func (elem *Elem) PlateDivision(add bool) ([]*Elem, float64, error) {
	if elem.Enods < 3 {
		return nil, 0.0, fmt.Errorf("PlateDivision: too few enods: ELEM %d", elem.Num)
	}
	switch elem.Enods {
	default:
		return nil, 0.0, fmt.Errorf("PlateDivision: too many enods: ELEM %d", elem.Num)
	case 3:
		d := make([]float64, 3)
		sum := 0.0
		for i := 0; i < 3; i++ {
			j := i + 1
			k := i + 2
			if j >= 3 {
				j -= 3
			}
			if k >= 3 {
				k -= 3
			}
			d[i] = Distance(elem.Enod[j], elem.Enod[k])
			sum += d[i]
		}
		if sum == 0.0 {
			return nil, 0.0, fmt.Errorf("PlateDivision: zero area: ELEM %d", elem.Num)
		}
		coord := make([]float64, 3)
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				coord[i] += d[j] * elem.Enod[j].Coord[i] / sum
			}
		}
		n := NewNode() // inner centre
		n.Coord = coord
		if add {
			n = elem.Frame.AddNode(coord[0], coord[1], coord[2])
		}
		rtn := []*Elem{
			NewPlateElem([]*Node{elem.Enod[0], elem.Enod[1], n}, elem.Sect, elem.Etype),
			NewPlateElem([]*Node{elem.Enod[1], elem.Enod[2], n}, elem.Sect, elem.Etype),
			NewPlateElem([]*Node{elem.Enod[2], elem.Enod[0], n}, elem.Sect, elem.Etype),
		}
		area := 0.0
		for _, el := range rtn {
			area += el.Area()
		}
		return rtn, elem.Area() / area, nil
	case 4:
		cand := make([][]*Elem, 2)
		nodes := make([][]*Node, 2)
		for i := 0; i < 2; i++ {
			nodes[i] = make([]*Node, 2)
			for j := 0; j < 2; j++ {
				i0 := i + 2*j
				i1 := i0 + 1
				if i1 >= 4 {
					i1 -= 4
				}
				i2 := i0 + 2
				if i2 >= 4 {
					i2 -= 4
				}
				im := i0 - 1
				if im < 0 {
					im += 4
				}
				v := Direction(elem.Enod[i0], elem.Enod[i1], true)
				l := Distance(elem.Enod[i0], elem.Enod[i1])
				v1 := Direction(elem.Enod[i0], elem.Enod[im], true)
				v2 := Direction(elem.Enod[i1], elem.Enod[i2], true)
				tanA := math.Sqrt(2.0/(1.0+Dot(v, v1, 3)) - 1.0)
				tanB := math.Sqrt(2.0/(1.0-Dot(v, v2, 3)) - 1.0)
				t := tanB / (tanA + tanB)
				h := t * tanA
				vh1 := RotateVector(elem.Enod[i1].Coord, elem.Enod[i0].Coord, Cross(v, v1), 0.5*math.Pi)
				vh2 := RotateVector(elem.Enod[i0].Coord, elem.Enod[i1].Coord, Cross(v2, v), 0.5*math.Pi)
				vec := make([]float64, 3)
				coord := make([]float64, 3)
				for k := 0; k < 3; k++ {
					vec[k] = 0.5 * (vh1[k] - elem.Enod[i0].Coord[k] + vh2[k] - elem.Enod[i1].Coord[k]) * h
					coord[k] = elem.Enod[i0].Coord[k] + t*l*v[k] + vec[k]
				}
				n := NewNode()
				n.Coord = coord
				if add {
					n = elem.Frame.AddNode(coord[0], coord[1], coord[2])
				}
				nodes[i][j] = n
			}
			i0 := i
			i1 := i0 + 1
			i2 := i0 + 2
			i3 := i0 + 3 - i*4
			cand[i] = []*Elem{
				NewPlateElem([]*Node{elem.Enod[i0], elem.Enod[i1], nodes[i][0]}, elem.Sect, elem.Etype),
				NewPlateElem([]*Node{elem.Enod[i1], elem.Enod[i2], nodes[i][1], nodes[i][0]}, elem.Sect, elem.Etype),
				NewPlateElem([]*Node{elem.Enod[i2], elem.Enod[i3], nodes[i][1]}, elem.Sect, elem.Etype),
				NewPlateElem([]*Node{elem.Enod[i3], elem.Enod[i0], nodes[i][0], nodes[i][1]}, elem.Sect, elem.Etype),
			}
		}
		var sum0, sum1 float64
		for i := 0; i < 4; i++ {
			sum0 += cand[0][i].Area()
			sum1 += cand[1][i].Area()
		}
		if sum0 <= sum1 {
			if add {
				for i := 0; i < 2; i++ {
					elem.Frame.DeleteNode(nodes[1][i].Num)
				}
			}
			return cand[0], elem.Area() / sum0, nil
		} else {
			if add {
				for i := 0; i < 2; i++ {
					elem.Frame.DeleteNode(nodes[0][i].Num)
				}
			}
			return cand[1], elem.Area() / sum1, nil
		}
	}
}

func (elem *Elem) IsBraced() bool {
	if elem.Enods != 4 || elem.Sect.Figs[0].Prop.ES() == 0.0 {
		return false
	} else {
		return true
	}
}

func (elem *Elem) RectToBrace(nbrace int, rfact float64) []*Elem {
	if !elem.IsBraced() {
		return nil
	}
	var thick float64
	if t, ok := elem.Sect.Figs[0].Value["THICK"]; ok {
		thick = t
	} else if k, ok := elem.Sect.Figs[0].Value["KFACT"]; ok {
		thick = k * 0.2*150/GOHAN.G()/1e4
	} else {
		return nil
	}
	poi := elem.Sect.Figs[0].Prop.Poi()
	l, h := elem.PlateSize()
	// TODO: check wrate calculation
	wrate1 := elem.Wrect[0] / l
	wrate2 := 1.25 * math.Sqrt((elem.Wrect[0]*elem.Wrect[1])/(l*h))
	var wrate float64
	if wrate1 > wrate2 {
		wrate = wrate1
	} else {
		wrate = wrate2
	}
	Ae := (1.0 - wrate) * rfact / 1.2 * math.Pow(l*l+h*h, 1.5) * thick / (4.0 * (1.0 + poi) * l * h)
	Ae *= 2.0 / float64(nbrace)
	f := NewFig()
	f.Prop = elem.Sect.Figs[0].Prop
	f.Value["AREA"] = Ae
	var sec *Sect
	sec = elem.Frame.SearchBraceSect(f, elem.Etype-2, elem.Sect.Num)
	if sec == nil {
		elem.Frame.Maxsnum++
		sec = elem.Frame.AddSect(elem.Frame.Maxsnum)
		sec.Figs = []*Fig{f}
		sec.Type = elem.Etype - 2
		sec.Color = elem.Sect.Color
		sec.Original = elem.Sect.Num
		for i := 0; i < 12; i++ {
			if i%2 == 0 {
				sec.Yield[i] = 100.0
			} else {
				sec.Yield[i] = -100.0
			}
		}
	}
	rtn := make([]*Elem, nbrace)
	for i := 0; i < 2; i++ {
		rtn[i] = NewLineElem([]*Node{elem.Enod[i], elem.Enod[i+2]}, sec, elem.Etype-2)
		for j := 0; j < 3; j++ {
			rtn[i].Skip[j] = elem.Skip[j]
		}
	}
	return rtn
}

func (elem *Elem) Adopt(child *Elem) int {
	if elem.Children == nil {
		elem.Children = make([]*Elem, 2)
	}
	for i := 0; i < 2; i++ {
		if elem.Children[i] == nil {
			elem.Children[i] = child
			child.Parent = elem
			if i == 0 {
				child.Eldest = true
			} else {
				child.Eldest = false
			}
			return i
		}
	}
	return -1
}

func (elem *Elem) Brother() (*Elem, bool) {
	if elem.Parent == nil {
		return nil, false
	}
	if elem.Parent.Children == nil || len(elem.Parent.Children) == 0 {
		return nil, false
	}
	for _, el := range elem.Parent.Children {
		if el == elem {
			continue
		}
		return el, true
	}
	return nil, false
}

func (elem *Elem) OriginalSection() *Sect {
	switch elem.Etype {
	default:
		return elem.Sect
	case WBRACE, SBRACE:
		return elem.Parent.Sect
	}
}

func (elem *Elem) GetSectionRate() (SectionRate, bool, error) {
	var al SectionRate
	original := false
	if a := elem.Frame.Sects[elem.Sect.Num].Allow; a != nil {
		al = a
	} else if oa := elem.Frame.Sects[elem.OriginalSection().Num].Allow; oa != nil {
		al = oa
		original = true
	} else {
		if len(elem.Sect.Figs) == 0 || elem.Sect.Figs[0] == nil {
			return nil, false, fmt.Errorf("cannot find SectionRate")
		}
		if a := elem.Sect.Figs[0].GetSectionRate(elem.Sect.Num, elem.Etype); a != nil {
			al = a
			sign, _ := elem.Sect.SplitName()
			al.SetName(sign)
			elem.Sect.Allow = a
		} else {
			return nil, false, fmt.Errorf("cannot find SectionRate")
		}
	}
	return al, original, nil
}

func (elem *Elem) EdgedBy() ([]*Elem, error) {
	if elem.IsLineElem() {
		return nil, NotPlateElem("EdgedBy")
	}
	rtn := make([]*Elem, elem.Enods)
	i := 0
	for i := 0; i < elem.Enods-1; i++ {
		for _, el := range elem.Frame.SearchElem(elem.Enod[i : i+2]...) {
			if el.Etype == COLUMN || el.Etype == GIRDER {
				rtn[i] = el
				i++
				break
			}
		}
	}
	for _, el := range elem.Frame.SearchElem(elem.Enod[elem.Enods-1], elem.Enod[0]) {
		if el.Etype == COLUMN || el.Etype == GIRDER {
			rtn[i] = el
			i++
			break
		}
	}
	return rtn[:i], nil
}

func (elem *Elem) EdgeOf() ([]*Elem, error) {
	if !elem.IsLineElem() {
		return nil, NotLineElem("EdgeOf")
	}
	els := elem.Frame.SearchElem(elem.Enod...)
	rtn := make([]*Elem, len(els))
	i := 0
	for _, el := range els {
		if el.Etype == WALL || el.Etype == SLAB {
			rtn[i] = el
			i++
		}
	}
	return rtn[:i], nil
}

func (elem *Elem) YieldFunction(period string) ([]float64, []error) {
	err := make([]error, 2)
	fc := make([]float64, 6)
	fu := make([]float64, 6)
	f := make([]float64, 2)
	f1 := make([]float64, 2)
	f2 := make([]float64, 2)
	var value float64
	for j := 0; j < 6; j++ {
		fc[j] = 0.5 * (elem.Sect.Yield[2*j] + elem.Sect.Yield[2*j+1])
		fu[j] = 0.5 * (elem.Sect.Yield[2*j] - elem.Sect.Yield[2*j+1])
	}
	for i := 0; i < 2; i++ {
		for j := 0; j < 6; j++ {
			switch j {
			case 0, 3: // Nz, Mz
				value = math.Abs(elem.ReturnStress(period, i, j)-math.Pow(-1.0, float64(i))*fc[j]) / fu[j]
				f1[i] += math.Pow(value, elem.Sect.Exp)
			case 1, 2: // Qx, Qy
				value = math.Abs(elem.ReturnStress(period, i, j)-fc[j]) / fu[j]
				if value*arclm.QUFACT > 1.0 {
					err[i] = arclm.BrittleFailure(elem, i)
				}
				f2[i] += math.Pow(value, elem.Sect.Exp)
			case 4, 5: // Mx, My
				value = math.Abs(elem.ReturnStress(period, i, j)-fc[j]) / fu[j]
				f1[i] += math.Pow(value, elem.Sect.Exp)
			}
		}
		f[i] = math.Pow(math.Pow(f1[i], elem.Sect.Exq/elem.Sect.Exp)+math.Pow(f2[i], elem.Sect.Exq/elem.Sect.Exp), 1.0/elem.Sect.Exq)
		if f[i] > arclm.RADIUS && err[i] == nil {
			err[i] = arclm.Yielded(elem, i)
		}
	}
	return f, err
}

func (elem *Elem) Direction(normalize bool) []float64 {
	vec := make([]float64, 3)
	var l float64
	if normalize {
		l = elem.Length()
	} else {
		l = 1.0
	}
	for i := 0; i < 3; i++ {
		vec[i] = (elem.Enod[1].Coord[i] - elem.Enod[0].Coord[i]) / l
	}
	return vec
}

func (elem *Elem) DeformedDirection(normalize bool, p string, fact float64) []float64 {
	vec := make([]float64, 3)
	for i := 0; i < 3; i++ {
		vec[i] = elem.Enod[1].Coord[i] + elem.Enod[1].ReturnDisp(p, i)*fact - elem.Enod[0].Coord[i] - elem.Enod[0].ReturnDisp(p, i)*fact
	}
	if normalize {
		return Normalize(vec)
	} else {
		return vec
	}
}

func (elem *Elem) EdgeDirection(ind int, normalize bool) []float64 {
	if ind >= elem.Enods-1 {
		return nil
	}
	jind := ind + 1
	if jind >= elem.Enods {
		jind -= elem.Enods
	}
	vec := make([]float64, 3)
	var l float64
	if normalize {
		l = elem.EdgeLength(ind)
	} else {
		l = 1.0
	}
	for i := 0; i < 3; i++ {
		vec[i] = (elem.Enod[jind].Coord[i] - elem.Enod[ind].Coord[i]) / l
	}
	return vec
}

func (elem *Elem) LowerEnod() []*Node {
	if elem.IsLineElem() {
		return nil
	}
	cand := elem.Enod[0]
	ind := 0
	for i, n := range elem.Enod[1:] {
		if n.Coord[2] < cand.Coord[2] {
			ind = i + 1
			cand = n
		}
	}
	switch ind {
	case 0:
		if elem.Enod[elem.Enods-1].Coord[2] < elem.Enod[1].Coord[2] {
			return []*Node{elem.Enod[elem.Enods-1], elem.Enod[0]}
		} else {
			return []*Node{elem.Enod[0], elem.Enod[1]}
		}
	case elem.Enods - 1:
		if elem.Enod[elem.Enods-2].Coord[2] < elem.Enod[1].Coord[2] {
			return []*Node{elem.Enod[elem.Enods-2], elem.Enod[elem.Enods-1]}
		} else {
			return []*Node{elem.Enod[elem.Enods-1], elem.Enod[0]}
		}
	default:
		if elem.Enod[ind-1].Coord[2] < elem.Enod[ind+1].Coord[2] {
			return []*Node{elem.Enod[ind-1], elem.Enod[ind]}
		} else {
			return []*Node{elem.Enod[ind], elem.Enod[ind+1]}
		}
	}
}

func (elem *Elem) HorizontalDirection(normalize bool) []float64 {
	if elem.IsLineElem() {
		return nil
	}
	ns := elem.LowerEnod()
	vec := make([]float64, 3)
	for i := 0; i < 2; i++ {
		vec[i] = (ns[1].Coord[i] - ns[0].Coord[i])
	}
	if normalize {
		l := math.Sqrt(vec[0]*vec[0] + vec[1]*vec[1])
		if l == 0.0 {
			return vec
		}
		for i := 0; i < 2; i++ {
			vec[i] /= l
		}
	}
	return vec
}

func (elem *Elem) Normal(normalize bool) []float64 {
	if elem.Enods < 3 {
		return nil
	}
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
	for i := 0; i < 3; i++ {
		tmp := 0.0
		for _, n := range elem.Enod {
			tmp += n.Coord[i]
		}
		rtn[i] = tmp / float64(elem.Enods)
	}
	return rtn
}

func (elem *Elem) DeformedMidPoint(p string, fact float64) []float64 {
	rtn := make([]float64, 3)
	for i := 0; i < 3; i++ {
		tmp := 0.0
		for _, n := range elem.Enod {
			tmp += n.Coord[i] + n.ReturnDisp(p, i)*fact
		}
		rtn[i] = tmp / float64(elem.Enods)
	}
	return rtn
}

func (elem *Elem) WrectCoord() [][]float64 {
	if elem.Enods != 4 {
		return nil
	}
	if elem.Wrect == nil || (elem.Wrect[0] == 0.0 || elem.Wrect[1] == 0.0) {
		return nil
	}
	coord := elem.MidPoint()
	rtn := make([][]float64, 4)
	vec := elem.HorizontalDirection(true)
	rtn[0] = []float64{coord[0] + 0.5*vec[0]*elem.Wrect[0], coord[1] + 0.5*vec[1]*elem.Wrect[0], coord[2] + 0.5*elem.Wrect[1]}
	rtn[1] = []float64{coord[0] + 0.5*vec[0]*elem.Wrect[0], coord[1] + 0.5*vec[1]*elem.Wrect[0], coord[2] - 0.5*elem.Wrect[1]}
	rtn[2] = []float64{coord[0] - 0.5*vec[0]*elem.Wrect[0], coord[1] - 0.5*vec[1]*elem.Wrect[0], coord[2] - 0.5*elem.Wrect[1]}
	rtn[3] = []float64{coord[0] - 0.5*vec[0]*elem.Wrect[0], coord[1] - 0.5*vec[1]*elem.Wrect[0], coord[2] + 0.5*elem.Wrect[1]}
	return rtn
}

func (elem *Elem) IsParallel(v []float64, eps float64) bool {
	if !elem.IsLineElem() {
		return false
	}
	return IsParallel(elem.Direction(false), v, eps)
}

func (elem *Elem) IsOrthogonal(v []float64, eps float64) bool {
	if !elem.IsLineElem() {
		return false
	}
	return IsOrthogonal(elem.Direction(false), v, eps)
}

func (elem *Elem) PLength() float64 {
	sum := 0.0
	for i := 0; i < 2; i++ {
		sum += math.Pow((elem.Enod[1].Pcoord[i] - elem.Enod[0].Pcoord[i]), 2)
	}
	return math.Sqrt(sum)
}

func (elem *Elem) PDirection(normalize bool) []float64 {
	vec := make([]float64, 2)
	var l float64
	if normalize {
		l = elem.PLength()
	} else {
		l = 1.0
	}
	for i := 0; i < 2; i++ {
		vec[i] = (elem.Enod[1].Pcoord[i] - elem.Enod[0].Pcoord[i]) / l
	}
	return vec
}

func (elem *Elem) Move(x, y, z float64, eps float64) {
	newenod := make([]*Node, elem.Enods)
	for i := 0; i < elem.Enods; i++ {
		var created bool
		newenod[i], created = elem.Frame.CoordNode(elem.Enod[i].Coord[0]+x, elem.Enod[i].Coord[1]+y, elem.Enod[i].Coord[2]+z, eps)
		if created {
			for j := 0; j < 6; j++ {
				newenod[i].Conf[j] = elem.Enod[i].Conf[j]
			}
		}
	}
	elem.Enod = newenod
}

func (elem *Elem) Copy(x, y, z float64, eps float64) *Elem {
	newenod := make([]*Node, elem.Enods)
	for i := 0; i < elem.Enods; i++ {
		var created bool
		newenod[i], created = elem.Frame.CoordNode(elem.Enod[i].Coord[0]+x, elem.Enod[i].Coord[1]+y, elem.Enod[i].Coord[2]+z, eps)
		if created {
			for j := 0; j < 6; j++ {
				newenod[i].Conf[j] = elem.Enod[i].Conf[j]
			}
		}
	}
	if elem.IsLineElem() {
		newelem := elem.Frame.AddLineElem(-1, newenod, elem.Sect, elem.Etype)
		newelem.Cang = elem.Cang
		for i := 0; i < 2; i++ {
			for j := 0; j < 6; j++ {
				newelem.Bonds[6*i+j] = elem.Bonds[6*i+j]
				newelem.Cmq[6*i+j] = elem.Cmq[6*i+j]
			}
		}
		return newelem
	} else {
		return elem.Frame.AddPlateElem(-1, newenod, elem.Sect, elem.Etype)
	}
}

func (elem *Elem) CopyRotate(center, vector []float64, angle float64, eps float64) *Elem {
	newenod := make([]*Node, elem.Enods)
	for i := 0; i < elem.Enods; i++ {
		c := RotateVector(elem.Enod[i].Coord, center, vector, angle)
		var created bool
		newenod[i], created = elem.Frame.CoordNode(c[0], c[1], c[2], eps)
		if created {
			for j := 0; j < 6; j++ {
				newenod[i].Conf[j] = elem.Enod[i].Conf[j]
			}
		}
	}
	if elem.IsLineElem() {
		newelem := elem.Frame.AddLineElem(-1, newenod, elem.Sect, elem.Etype)
		newelem.Cang = elem.Cang
		for i := 0; i < 2; i++ {
			for j := 0; j < 6; j++ {
				newelem.Bonds[6*i+j] = elem.Bonds[6*i+j]
				newelem.Cmq[6*i+j] = elem.Cmq[6*i+j]
			}
		}
		return newelem
	} else {
		return elem.Frame.AddPlateElem(-1, newenod, elem.Sect, elem.Etype)
	}
}

func (elem *Elem) Mirror(coord, vec []float64, del bool, eps float64) *Elem {
	newenod := make([]*Node, elem.Enods)
	var add bool
	for i := 0; i < elem.Enods; i++ {
		newcoord := elem.Enod[i].MirrorCoord(coord, vec)
		newenod[i], _ = elem.Frame.CoordNode(newcoord[0], newcoord[1], newcoord[2], eps)
		if !add && (newenod[i] != elem.Enod[i]) {
			add = true
		}
	}
	if add {
		if del {
			elem.Enod = newenod
			return elem
		} else {
			if elem.IsLineElem() {
				e := elem.Frame.AddLineElem(-1, newenod, elem.Sect, elem.Etype)
				for i := 0; i < 6*elem.Enods; i++ {
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

func (elem *Elem) Offset(value, angle, eps float64) *Elem {
	vec := make([]float64, 3)
	c := math.Cos(angle)
	s := math.Sin(angle)
	for i := 0; i < 3; i++ {
		vec[i] = value * (elem.Strong[i]*c + elem.Weak[i]*s)
	}
	return elem.Copy(vec[0], vec[1], vec[2], eps)
}

func (elem *Elem) Invert() {
	if len(elem.Enod) == 0 {
		return
	}
	if elem.IsLineElem() {
		newenod := make([]*Node, elem.Enods)
		newbonds := make([]*Bond, 6*elem.Enods)
		newcmq := make([]float64, 6*elem.Enods)
		ind := elem.Enods - 1
		for i := 0; i < elem.Enods; i++ {
			newenod[i] = elem.Enod[ind]
			for j := 0; j < 6; j++ {
				newbonds[i] = elem.Bonds[6*ind+j]
				newcmq[i] = elem.Cmq[6*ind+j]
			}
			ind--
		}
		elem.Enod = newenod
		elem.Bonds = newbonds
		elem.Cmq = newcmq
	} else {
		newenod := make([]*Node, elem.Enods)
		ind := elem.Enods - 1
		for i := 0; i < elem.Enods; i++ {
			newenod[i] = elem.Enod[ind]
			ind--
		}
		elem.Enod = newenod
	}
}

func (elem *Elem) Upside() {
	if len(elem.Enod) == 0 {
		return
	}
	newenod := Upside(elem.Enod)
	if elem.IsLineElem() && newenod[0] != elem.Enod[0] {
		newbonds := make([]*Bond, 12)
		newcmq := make([]float64, 12)
		for i := 0; i < 6; i++ {
			newbonds[i] = elem.Bonds[6+i]
			newbonds[6+i] = elem.Bonds[i]
			newcmq[i] = elem.Cmq[6+i]
			newcmq[6+i] = elem.Cmq[i]
		}
		elem.Bonds = newbonds
		elem.Cmq = newcmq
	}
	elem.Enod = newenod
}

func (elem *Elem) IsValidElem() (bool, error) {
	valid := true
	var otp bytes.Buffer
	otp.WriteString(fmt.Sprintf("ELEM: %d", elem.Num))
	if elem.Etype == NULL {
		valid = false
		otp.WriteString(", NULL Etype")
	}
	if elem.HasSameNode() {
		valid = false
		otp.WriteString(", Same Node\n")
	}
	if !elem.IsLineElem() {
		ok, err := elem.IsValidPlate()
		if !ok {
			valid = false
			otp.WriteString(fmt.Sprintf(", Invalid(%s)", err.Error()))
		}
	}
	if valid {
		return true, nil
	} else {
		return false, errors.New(otp.String())
	}
}

func (elem *Elem) HasSameNode() bool {
	for i, en := range elem.Enod[:elem.Enods-1] {
		for _, em := range elem.Enod[i+1:] {
			if en == em {
				return true
			}
		}
	}
	return false
}

func (elem *Elem) IsValidPlate() (bool, error) {
	switch elem.Enods {
	case 0, 1, 2:
		return false, errors.New(fmt.Sprintf("IsValidPlate: too few Enods %d", elem.Enods))
	case 3:
		if OnTheSameLine(elem.Enod[0].Coord, elem.Enod[1].Coord, elem.Enod[2].Coord, 5e-3) {
			return false, errors.New("IsValidPlate: 3 nodes are on the same line")
		}
		return true, nil
	case 4:
		if elem.Area() == 0.0 {
			return false, errors.New("IsValidPlate: Area = 0.0")
		}
		if OnTheSameLine(elem.Enod[0].Coord, elem.Enod[1].Coord, elem.Enod[2].Coord, 5e-3) ||
			OnTheSameLine(elem.Enod[0].Coord, elem.Enod[1].Coord, elem.Enod[3].Coord, 5e-3) ||
			OnTheSameLine(elem.Enod[0].Coord, elem.Enod[2].Coord, elem.Enod[3].Coord, 5e-3) ||
			OnTheSameLine(elem.Enod[1].Coord, elem.Enod[2].Coord, elem.Enod[3].Coord, 5e-3) {
			return false, errors.New("IsValidPlate: 3 nodes are on the same line")
		}
		if !OnTheSamePlane(elem.Enod[0].Coord, elem.Enod[1].Coord, elem.Enod[2].Coord, elem.Enod[3].Coord, 5e-3) {
			return false, errors.New("IsValidPlate: 4 nodes are not on the same plate")
		}
		return true, nil
	default:
		return false, errors.New(fmt.Sprintf("IsValidPlate: too many Enods %d", elem.Enods))
	}
}

func (elem *Elem) DividingPoint(ratio float64) []float64 {
	rtn := make([]float64, 3)
	for i := 0; i < 3; i++ {
		rtn[i] = elem.Enod[0].Coord[i]*(1.0-ratio) + elem.Enod[1].Coord[i]*ratio
	}
	return rtn
}

func (elem *Elem) EdgeDividingPoint(num int, ratio float64) []float64 {
	rtn := make([]float64, 3)
	start := num
	end := num + 1
	if end >= elem.Enods {
		end -= elem.Enods
	}
	for i := 0; i < 3; i++ {
		rtn[i] = elem.Enod[start].Coord[i]*(1.0-ratio) + elem.Enod[end].Coord[i]*ratio
	}
	return rtn
}

func (elem *Elem) AxisCoord(axis int, coord float64) (rtn []float64, err error) {
	if !elem.IsLineElem() {
		return rtn, NotLineElem("AxisCoord")
	}
	den := elem.Direction(false)[axis]
	if den == 0.0 {
		return rtn, errors.New("AxisCoord: Cannot Divide")
	}
	k := (coord - elem.Enod[0].Coord[axis]) / den
	return elem.DividingPoint(k), nil
}

func (elem *Elem) DivideAtCoord(x, y, z float64, eps float64) (ns []*Node, els []*Elem, err error) {
	if !elem.IsLineElem() {
		return nil, nil, NotLineElem("DivideAtCoord")
	}
	n, _ := elem.Frame.CoordNode(x, y, z, eps)
	for i := 0; i < elem.Enods; i++ {
		if n == elem.Enod[i] {
			return nil, nil, DivideAtEnod("DivideAtCoord")
		}
	}
	els = make([]*Elem, 2)
	els[0] = elem
	newelem := elem.Frame.AddLineElem(-1, []*Node{n, elem.Enod[1]}, elem.Sect, elem.Etype)
	els[1] = newelem
	elem.Enod[1] = n
	ns = []*Node{n}
	for j := 0; j < 6; j++ {
		els[1].Bonds[6+j] = elem.Bonds[6+j]
		elem.Bonds[6+j] = nil
	}
	els[1].Cang = elem.Cang
	els[1].SetPrincipalAxis()
	if elem.Chain != nil {
		newelem.Chain = elem.Chain
		ind, _ := elem.Chain.Has(elem)
		elem.Chain.AppendAt(newelem, ind+1)
	} else {
		chain := ChainElem(elem, newelem)
		elem.Frame.Chains[elem.Num] = chain
	}
	return
}

func (elem *Elem) DivideAtNode(n *Node, position int, del bool) (rn []*Node, els []*Elem, err error) {
	if !elem.IsLineElem() {
		return nil, nil, NotLineElem("DivideAtNode")
	}
	for i := 0; i < elem.Enods; i++ {
		if n == elem.Enod[i] {
			return nil, nil, DivideAtEnod("DivideAtNode")
		}
	}
	els = make([]*Elem, 2)
	if del {
		switch position {
		default:
			return nil, nil, errors.New("DivideAtNode: Unknown Position")
		case -1, 0:
			els := elem.Frame.SearchElem(elem.Enod[0])
			if len(els) == 1 {
				delete(elem.Frame.Nodes, elem.Enod[0].Num)
			} else {
				if elem.Chain != nil {
					ind, _ := elem.Chain.Has(elem)
					elem.Chain.DivideAt(ind)
				}
			}
			elem.Enod[0] = n
			return []*Node{n}, []*Elem{elem}, nil
		case 1, 2:
			els := elem.Frame.SearchElem(elem.Enod[1])
			if len(els) == 1 {
				delete(elem.Frame.Nodes, elem.Enod[1].Num)
			} else {
				if elem.Chain != nil {
					ind, _ := elem.Chain.Has(elem)
					elem.Chain.DivideAt(ind + 1)
				}
			}
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
			for j := 0; j < 6; j++ {
				els[1].Bonds[6+j] = elem.Bonds[6+j]
				elem.Bonds[6+j] = nil
			}
			els[1].Cang = elem.Cang
			els[1].SetPrincipalAxis()
			if elem.Chain != nil {
				newelem.Chain = elem.Chain
				ind, _ := elem.Chain.Has(elem)
				elem.Chain.AppendAt(newelem, ind+1)
			} else {
				chain := ChainElem(elem, newelem)
				elem.Frame.Chains[elem.Num] = chain
			}
			return []*Node{n}, els, nil
		case 0:
			newelem := elem.Frame.AddLineElem(-1, []*Node{n, elem.Enod[0]}, elem.Sect, elem.Etype)
			els[0] = elem
			els[1] = newelem
			els[1].Cang = elem.Cang
			els[1].SetPrincipalAxis()
			if elem.Chain != nil {
				newelem.Chain = elem.Chain
				ind, _ := elem.Chain.Has(elem)
				elem.Chain.AppendAt(newelem, ind)
			} else {
				chain := ChainElem(newelem, elem)
				elem.Frame.Chains[elem.Num] = chain
			}
			return []*Node{n}, els, nil
		case 2:
			newelem := elem.Frame.AddLineElem(-1, []*Node{elem.Enod[1], n}, elem.Sect, elem.Etype)
			els[0] = elem
			els[1] = newelem
			els[1].Cang = elem.Cang
			els[1].SetPrincipalAxis()
			if elem.Chain != nil {
				newelem.Chain = elem.Chain
				ind, _ := elem.Chain.Has(elem)
				elem.Chain.AppendAt(newelem, ind+1)
			} else {
				chain := ChainElem(elem, newelem)
				elem.Frame.Chains[elem.Num] = chain
			}
			return []*Node{n}, els, nil
		}
	}
}

func (elem *Elem) DivideAtRate(k float64, eps float64) (n []*Node, els []*Elem, err error) {
	c := elem.DividingPoint(k)
	return elem.DivideAtCoord(c[0], c[1], c[2], eps)
}

func (elem *Elem) DivideAtMid(eps float64) (n []*Node, els []*Elem, err error) {
	return elem.DivideAtRate(0.5, eps)
}

func (elem *Elem) DivideAtAxis(axis int, coord float64, eps float64) (n []*Node, els []*Elem, err error) {
	c, err := elem.AxisCoord(axis, coord)
	if err != nil {
		return
	}
	return elem.DivideAtCoord(c[0], c[1], c[2], eps)
}

func (elem *Elem) DivideAtLength(length float64, eps float64) (n []*Node, els []*Elem, err error) {
	return elem.DivideAtRate(length/elem.Length(), eps)
}

func (elem *Elem) DivideInN(n int, eps float64) (rn []*Node, els []*Elem, err error) {
	if !elem.IsLineElem() {
		return nil, nil, NotLineElem("DivideInN")
	}
	if n == 1 {
		return nil, []*Elem{elem}, nil
	}
	rate := make([]float64, n-1)
	for i := 1; i < n; i++ {
		rate[i-1] = float64(i) / float64(i+1)
	}
	rn = make([]*Node, n-1)
	els = make([]*Elem, n)
	els[0] = elem
	for i := n - 2; i >= 0; i-- {
		newn, newels, err := elem.DivideAtRate(rate[i], eps)
		if err != nil {
			return nil, nil, err
		}
		rn[i] = newn[0]
		els[i+1] = newels[1]
	}
	return
}

func (elem *Elem) OnNode(num int, eps float64) []*Node {
	var num2 int
	if num >= elem.Enods {
		return nil
	} else if num == elem.Enods-1 {
		num2 = 0
	} else {
		num2 = num + 1
	}
	candidate := elem.Frame.NodeInBox(elem.Enod[num], elem.Enod[num2], eps)
	direction := elem.Frame.Direction(elem.Enod[num], elem.Enod[num2], true)
	ons := make([]*Node, len(candidate))
	i := 0
	nodes := make(map[float64]*Node, 0)
	var keys []float64
	for _, n := range candidate {
		if n.Num == elem.Enod[num].Num || n.Num == elem.Enod[num2].Num {
			continue
		}
		d := elem.Frame.Direction(elem.Enod[num], n, false)
		_, _, _, l := elem.Frame.Distance(elem.Enod[num], n)
		fac := Dot(direction, d, 3)
		sum := 0.0
		for j := 0; j < 3; j++ {
			sum += math.Pow(d[j]-fac*direction[j], 2.0)
		}
		if math.Sqrt(sum) < eps {
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

func (elem *Elem) DivideAtOns(eps float64) (rn []*Node, els []*Elem, err error) {
	if elem.IsLineElem() {
		rn = elem.OnNode(0, eps)
		l := len(rn)
		if l == 0 {
			return nil, []*Elem{elem}, nil
		}
		els = make([]*Elem, l+1)
		els[0] = elem
		for i := l - 1; i >= 0; i-- {
			_, newels, err := elem.DivideAtNode(rn[i], 1, false)
			if err != nil {
				return nil, nil, err
			}
			els[i+1] = newels[1]
		}
		return rn, els, nil
	} else {
		switch elem.Enods {
		default:
			return nil, nil, errors.New("DivideAtOns: Enod != 3 or 4")
		case 3:
			rns := make([][]*Node, 3)
			for i := 0; i < 3; i++ {
				rns[i] = elem.OnNode(i, eps)
			}
			if len(rns[0]) == 0 {
				if len(rns[1]) == 0 || len(rns[2]) == 0 {
					return nil, nil, errors.New("DivideAtOns: cannot divide")
				}
				newel := elem.Frame.AddPlateElem(-1, []*Node{elem.Enod[0], elem.Enod[1], rns[1][0], rns[2][0]}, elem.Sect, elem.Etype)
				elem.Enod[0] = rns[1][0]
				elem.Enod[1] = rns[2][0]
				return []*Node{rns[1][0], rns[2][0]}, []*Elem{elem, newel}, nil
			} else {
				if len(rns[1]) != 0 && len(rns[2]) != 0 {
					newel1 := elem.Frame.AddPlateElem(-1, []*Node{rns[0][0], elem.Enod[1], rns[1][0]}, elem.Sect, elem.Etype)
					newel2 := elem.Frame.AddPlateElem(-1, []*Node{rns[2][0], rns[1][0], elem.Enod[2]}, elem.Sect, elem.Etype)
					newel3 := elem.Frame.AddPlateElem(-1, []*Node{rns[0][0], rns[1][0], rns[2][0]}, elem.Sect, elem.Etype)
					elem.Enod[1] = rns[0][0]
					elem.Enod[2] = rns[2][0]
					return []*Node{rns[0][0], rns[1][0], rns[2][0]}, []*Elem{elem, newel1, newel2, newel3}, nil
				} else if len(rns[1]) == 0 {
					newel := elem.Frame.AddPlateElem(-1, []*Node{rns[0][0], elem.Enod[1], elem.Enod[2], rns[2][0]}, elem.Sect, elem.Etype)
					elem.Enod[1] = rns[0][0]
					elem.Enod[2] = rns[2][0]
					return []*Node{rns[0][0], rns[2][0]}, []*Elem{elem, newel}, nil
				} else {
					newel := elem.Frame.AddPlateElem(-1, []*Node{elem.Enod[0], rns[0][0], rns[1][0], elem.Enod[2]}, elem.Sect, elem.Etype)
					elem.Enod[0] = rns[0][0]
					elem.Enod[2] = rns[1][0]
					return []*Node{rns[0][0], rns[1][0]}, []*Elem{elem, newel}, nil
				}
			}
		case 4:
			for i := 0; i < 2; i++ {
				rn1 := elem.OnNode(i, eps)
				rn2 := elem.OnNode(i+2, eps)
				if len(rn1) != 1 || len(rn2) != 1 {
					continue
				}
				newel := elem.Frame.AddPlateElem(-1, []*Node{rn1[0], elem.Enod[1+i], elem.Enod[2+i], rn2[0]}, elem.Sect, elem.Etype)
				elem.Enod[1+i] = rn1[0]
				elem.Enod[2+i] = rn2[0]
				return []*Node{rn1[0], rn2[0]}, []*Elem{elem, newel}, nil
			}
			return nil, nil, errors.New("DivideAtOns: divide pattern is indeterminate")
		}
	}
}

func (elem *Elem) DivideAtElem(eps float64) ([]*Elem, error) {
	if elem.IsLineElem() {
		return nil, NotPlateElem("DivideAtElem")
	}
	if elem.Enods != 4 {
		return nil, ElemDivisionError{"DivideAtElem", "enods != 4"}
	}
	var cand *Elem
	var ind1, ind2, ind3, ind4 int
	var ns1, ns2 []*Node
	divdiag := -1
divatdiag:
	for i := 0; i < 2; i++ {
		els := elem.Frame.SearchElem(elem.Enod[i], elem.Enod[i+2])
		for _, el := range els {
			if el.IsLineElem() && el.Etype <= GIRDER {
				divdiag = i
				break divatdiag
			}
		}
	}
	switch divdiag {
	case 0:
		newel := elem.Frame.AddPlateElem(-1, []*Node{elem.Enod[2], elem.Enod[3], elem.Enod[0]}, elem.Sect, elem.Etype)
		elem.Enod = elem.Enod[:3]
		elem.Enods = 3
		return []*Elem{elem, newel}, nil
	case 1:
		newel := elem.Frame.AddPlateElem(-1, []*Node{elem.Enod[1], elem.Enod[2], elem.Enod[3]}, elem.Sect, elem.Etype)
		elem.Enod = []*Node{elem.Enod[0], elem.Enod[1], elem.Enod[3]}
		elem.Enods = 3
		return []*Elem{elem, newel}, nil
	}
divatelem:
	for i := 0; i < 2; i++ {
		ns1 = elem.OnNode(i, eps)
		if len(ns1) == 0 {
			continue
		}
		ns2 = elem.OnNode(i+2, eps)
		if len(ns2) == 0 {
			continue
		}
		for p := 0; p < len(ns1); p++ {
			for q := 0; q < len(ns2); q++ {
				els := elem.Frame.SearchElem(ns1[p], ns2[q])
				if len(els) > 0 {
					cand = els[0]
					ind1 = p
					ind2 = q
					ind3 = i + 1
					ind4 = i + 2
					break divatelem
				}
			}
		}
	}
	if cand == nil {
		return nil, ElemDivisionError{"DivideAtElem", "no element"}
	}
	newel := elem.Frame.AddPlateElem(-1, []*Node{ns1[ind1], elem.Enod[ind3], elem.Enod[ind4], ns2[ind2]}, elem.Sect, elem.Etype)
	elem.Enod[ind3] = ns1[ind1]
	elem.Enod[ind4] = ns2[ind2]
	return []*Elem{elem, newel}, nil
}

func (elem *Elem) BetweenNode(index, size int) []*Node {
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
	if size == 0 || !elem.IsLineElem() {
		return rtn
	}
	d := elem.Direction(true)
	L := elem.Length()
	maxlen := 1000.0
	cand := 0
	for _, n := range elem.Frame.Nodes {
		if n.hide {
			continue
		}
		if n == elem.Enod[0] || n == elem.Enod[1] {
			continue
		}
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
					for i := 0; i < cand; i++ {
						if tmpd < dst[i] {
							for j := cand; j > i; j-- {
								rtn[j] = rtn[j-1]
								dst[j] = dst[j-1]
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
					maxlen = dst[cand]
				} else {
					if tmpd < maxlen {
						first := true
						for i := size - 1; i > 0; i-- {
							if tmpd > dst[i-1] {
								for j := size - 1; j > i; j-- {
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
							for i := size - 1; i > 0; i-- {
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

func (elem *Elem) EnodIndex(side int) (int, error) {
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

func (elem *Elem) RefNnum(nnum int) (int, error) {
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

// Otherside returns the other ENOD.
// If given node is not found in ENOD, it returns nil.
func (elem *Elem) Otherside(n *Node) *Node {
	for i, en := range elem.Enod {
		if en == n {
			return elem.Enod[1-i]
		}
	}
	return nil
}

func (elem *Elem) RefEnod(nnum int) (*Node, error) {
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

func (elem *Elem) PruneEnod() bool {
	ns := make([]*Node, elem.Enods)
	nnum := 0
pruneenod:
	for _, n := range elem.Enod {
		for j := 0; j < nnum; j++ {
			if n == ns[j] {
				continue pruneenod
			}
		}
		ns[nnum] = n
		nnum++
	}
	if elem.Enods == nnum {
		return false
	} else {
		elem.Enod = ns[:nnum]
		elem.Enods = nnum
		return true
	}
}

func (elem *Elem) IsDiagonal(n1, n2 *Node) bool {
	if elem.IsLineElem() {
		return false
	}
	i1, err := elem.EnodIndex(n1.Num)
	if err != nil {
		return false
	}
	i2, err := elem.EnodIndex(n2.Num)
	if err != nil {
		return false
	}
	switch i1 - i2 {
	case 2, -2:
		return true
	default:
		return false
	}
}

func (elem *Elem) ReturnStress(period string, nnum int, index int) float64 {
	if period == "" || !elem.IsLineElem() || elem.Stress == nil {
		return 0.0
	}
	return PeriodValue(period, func(p string, s float64) float64 {
		if val, ok := elem.Stress[p]; ok {
			if nnum == 0 || nnum == 1 {
				if rtn, ok := val[elem.Enod[nnum].Num]; ok {
					return s * rtn[index]
				} else {
					return 0.0
				}
			} else {
				for _, en := range elem.Enod {
					if en.Num == nnum {
						return s * val[nnum][index]
					}
				}
				return 0.0
			}
		} else {
			return 0.0
		}
	})
}

func (elem *Elem) N(period string, nnum int) float64 {
	return elem.ReturnStress(period, nnum, 0)
}
func (elem *Elem) QX(period string, nnum int) float64 {
	return elem.ReturnStress(period, nnum, 1)
}
func (elem *Elem) QY(period string, nnum int) float64 {
	return elem.ReturnStress(period, nnum, 2)
}
func (elem *Elem) MT(period string, nnum int) float64 {
	return elem.ReturnStress(period, nnum, 3)
}
func (elem *Elem) MX(period string, nnum int) float64 {
	return elem.ReturnStress(period, nnum, 4)
}
func (elem *Elem) MY(period string, nnum int) float64 {
	return elem.ReturnStress(period, nnum, 5)
}

func (elem *Elem) VectorStress(period string, nnum int, vec []float64) float64 {
	var sign int
	var ind int
	var err error
	if nnum == 0 || nnum == 1 {
		sign = 1 - nnum*2
		nnum = elem.Enod[nnum].Num
	} else {
		ind, err = elem.EnodIndex(nnum)
		if err != nil {
			return 0.0
		}
		sign = 1 - ind*2
	}
	ezaxis := elem.Direction(true)
	exaxis := elem.Strong
	eyaxis := elem.Weak
	rtn := 0.0
	for i, j := range [][]float64{ezaxis, exaxis, eyaxis} {
		rtn += Dot(j, vec, 3) * elem.ReturnStress(period, nnum, i)
	}
	return float64(sign) * rtn
}

func (elem *Elem) PlateStress(period string, vec []float64) float64 {
	if !elem.IsBraced() || elem.Children == nil {
		return 0.0
	}
	rtn := 0.0
	for _, el := range elem.Children {
		if el == nil {
			return 0.0
		}
		rtn += el.VectorStress(period, 0, vec)
	}
	return rtn
}

func (elem *Elem) LateralStiffness(period string, abs bool, chain bool) float64 {
	if elem.IsLineElem() {
		var axis []float64
		var index int
		switch period {
		default:
			return 0.0
		case "X":
			axis = XAXIS
			index = 0
		case "Y":
			axis = YAXIS
			index = 1
		}
		shear := elem.VectorStress(period, 0, axis)
		if shear == 0.0 {
			return 0.0
		}
		var delta float64
		if chain && elem.Chain != nil {
			for _, el := range elem.Chain.Elems() {
				delta += el.Delta(period, index)
			}
		} else {
			delta = elem.Delta(period, index)
		}
		if delta == 0.0 {
			// return 1e16
			return 0.0
		}
		if abs {
			return math.Abs(shear / delta)
		} else {
			return shear / delta
		}
	} else {
		return 0.0
	}
}

func (elem *Elem) Delta(period string, index int) float64 {
	if elem.IsLineElem() {
		return elem.Enod[1].ReturnDisp(period, index) - elem.Enod[0].ReturnDisp(period, index)
	} else {
		return 0.0
	}
}

func (elem *Elem) StoryDrift(period string, index int, chain bool) float64 {
	if elem.IsLineElem() {
		if chain && elem.Chain != nil {
			delta := 0.0
			height := 0.0
			for _, el := range elem.Chain.Elems() {
				delta += el.Delta(period, index)
				height += el.Direction(false)[2]
			}
			return delta / height
		} else {
			delta := elem.Delta(period, index)
			height := elem.Direction(false)[2]
			return delta / height
		}
	} else {
		return 0.0
	}
}

func (elem *Elem) UltimateTensileForce(period string) float64 {
	if !elem.IsLineElem() {
		return 0.0
	}
	var axis []float64
	switch period {
	default:
		return 0.0
	case "X":
		axis = YAXIS
	case "Y":
		axis = XAXIS
	}
	maxrate := 0.0
	plates, err := elem.EdgeOf()
	if err != nil {
		return 0.0
	}
	for _, plate := range plates {
		if elem.Num == 1011 {
			fmt.Println(plate.Num, plate.Etype, plate.IsBraced(), IsParallel(axis, plate.Normal(true), 1e-3))
		}
		if plate.Etype == WALL && plate.IsBraced() {
			if !IsParallel(axis, plate.Normal(true), 1e-3) {
				continue
			}
			r, err := plate.RateMax(elem.Frame.Show)
			if err != nil {
				continue
			}
			if r > maxrate {
				maxrate = r
			}
		}
	}
	if maxrate > 1.0 {
		maxrate = 1.0
	} else if maxrate == 0.0 {
		maxrate = 1.0
	}
	nl := elem.N("L", elem.Enod[0].Num)
	ne := elem.N(period, elem.Enod[0].Num)
	if ne < 0.0 {
		return nl + ne/maxrate
	} else {
		return nl - ne/maxrate
	}
}

func (elem *Elem) ChangeBond(bond []*Bond, side ...int) error {
	if !elem.IsLineElem() {
		return NotLineElem("ChangeBond")
	}
	if len(bond) < 6*len(side) {
		return errors.New("ChangeBond: Failed")
	}
	for _, i := range side {
		for j := 0; j < 6; j++ {
			elem.Bonds[6*i+j] = bond[6*i+j]
		}
	}
	return nil
}

func (elem *Elem) ToggleBond(side int) error {
	if !elem.IsLineElem() {
		return NotLineElem("ToggleBond")
	}
	if ind, err := elem.EnodIndex(side); err == nil {
		if elem.Bonds[6*ind+4] != nil || elem.Bonds[6*ind+5] != nil {
			for i := 4; i < 6; i++ {
				elem.Bonds[6*ind+i] = nil
			}
		} else {
			for i := 4; i < 6; i++ {
				elem.Bonds[6*ind+i] = Pin
			}
		}
		return nil
	} else {
		return err
	}
}

func (elem *Elem) BondState() (rtn int) {
	for i := 0; i < 2; i++ {
		if elem.Bonds[6*i+4] != nil || elem.Bonds[6*i+5] != nil {
			rtn += i + 1
		}
	}
	return
}

func (elem *Elem) IsPin(index int) bool {
	var i int
	switch index {
	case 0, 1:
		i = index
	default:
		val, err := elem.EnodIndex(index)
		if err != nil {
			return false
		}
		i = val
	}
	return elem.Bonds[6*i+4] == Pin || elem.Bonds[6*i+5] == Pin
}

func (elem *Elem) IsRigid(index int) bool {
	return !elem.IsPin(index)
}

func (elem *Elem) MomentCoord(show *Show, index int) [][]float64 {
	var axis []float64
	coord := make([][]float64, 2)
	if show.PlotState&PLOT_UNDEFORMED != 0 {
		if index == 4 {
			axis = elem.Weak
		} else if index == 5 {
			axis = make([]float64, 3)
			for i := 0; i < 3; i++ {
				axis[i] = -elem.Strong[i]
			}
		} else {
			return nil
		}
		for i := 0; i < 2; i++ {
			coord[i] = make([]float64, 3)
			for j := 0; j < 3; j++ {
				coord[i][j] = elem.Enod[i].Coord[j]
			}
		}
	} else {
		var strong, weak []float64
		s, w, err := PrincipalAxis(elem.DeformedDirection(true, show.Period, show.Dfact), elem.Cang)
		if err != nil {
			strong = elem.Strong
			weak = elem.Weak
		} else {
			strong = s
			weak = w
		}
		if index == 4 {
			axis = weak
		} else if index == 5 {
			axis = make([]float64, 3)
			for i := 0; i < 3; i++ {
				axis[i] = -strong[i]
			}
		} else {
			return nil
		}
		for i := 0; i < 2; i++ {
			coord[i] = make([]float64, 3)
			for j := 0; j < 3; j++ {
				coord[i][j] = elem.Enod[i].Coord[j] + elem.Enod[i].ReturnDisp(show.Period, j)*show.Dfact
			}
		}
	}
	ms := make([]float64, 2)
	qs := make([]float64, 2)
	for i := 0; i < 2; i++ {
		ms[i] = elem.ReturnStress(show.Period, i, index)
		qs[i] = elem.ReturnStress(show.Period, i, 6-index)
	}
	l := elem.Length()
	rtn := make([][]float64, 3)
	rtn[0] = make([]float64, 3)
	for i := 0; i < 3; i++ {
		rtn[0][i] = -show.Mfact*axis[i]*ms[0] + coord[0][i]
	}
	rtn[2] = make([]float64, 3)
	for i := 0; i < 3; i++ {
		rtn[2][i] = show.Mfact*axis[i]*ms[1] + coord[1][i]
	}
	if qs[0] != -qs[1] {
		tmp := make([]float64, 3)
		d := elem.Direction(true)
		val := (qs[1]*l - (ms[0] + ms[1])) / (qs[0] + qs[1])
		for i := 0; i < 3; i++ {
			tmp[i] = coord[0][i] + d[i]*val
		}
		ll := 0.0
		for i := 0; i < 3; i++ {
			ll += math.Pow(coord[1][i]-tmp[i], 2.0)
		}
		ll = math.Sqrt(ll)
		if 0.0 < ll && ll < l {
			rtn[1] = make([]float64, 3)
			val := (qs[0]*ms[1] - qs[1]*ms[0] - qs[0]*qs[1]*l) / (qs[0] + qs[1])
			for i := 0; i < 3; i++ {
				rtn[1][i] = tmp[i] + show.Mfact*axis[i]*val
			}
		} else {
			return [][]float64{rtn[0], rtn[2]}
		}
		return rtn
	} else {
		return [][]float64{rtn[0], rtn[2]}
	}
}

// func (elem *Elem) IsError (show *Show, val float64) bool {
//     if elem.IsLineElem() {
//         if e.Rate == nil { return false }
//         if len(e.Rate)%3 != 0 || (show.ElemCaption & EC_RATE_L == 0 && show.ElemCaption & EC_RATE_S == 0) {
//             for _, tmp := range e.Rate {
//                 if tmp > val { return true }
//             }
//             return false
//         } else {
//             rtnl := false
//             rtns := false
//             for i, tmp := range e.Rate {
//                 switch i%3 {
//                 default:
//                     continue
//                 case 0:
//                     if tmp > vall { rtnl = true }
//                 case 1:
//                     if tmp > vals { rtns = true }
//                 }
//             }
//             if show.ElemCaption & EC_RATE_L == 0 { rtnl = false }
//             if show.ElemCaption & EC_RATE_S == 0 { rtns = false }
//             return rtnl || rtns
//         }
//     } else {
//         if elem.Children != nil {
//     }
// }

func (elem *Elem) RateMax(show *Show) (float64, error) {
	returnratemax := func(els ...*Elem) (float64, error) {
		if len(els) == 0 || els[0] == nil {
			return 0.0, errors.New("RateMax: no value")
		}
		if els[0].MaxRate == nil {
			return 0.0, errors.New("RateMax: no value")
		}
		l := len(els[0].MaxRate)
		for _, el := range els[1:] {
			if el.MaxRate == nil {
				return 0.0, errors.New("RateMax: no value")
			}
			if len(el.MaxRate) != l {
				return 0.0, errors.New("RateMax: different size")
			}
		}
		if show == nil || show.SrcanRate == 0 {
			val := 0.0
			for i := 0; i < l; i++ {
				tmp := 0.0
				for _, el := range els {
					tmp += el.MaxRate[i]
				}
				if tmp > val {
					val = tmp
				}
			}
			return val / float64(len(els)), nil
		} else if l%3 != 0 {
			val := 0.0
			for i := 0; i < l; i++ {
				if i < 2 && (show.SrcanRate&SRCAN_Q != 0) || i >= 2 && (show.SrcanRate&SRCAN_M != 0) {
					tmp := 0.0
					for _, el := range els {
						tmp += el.MaxRate[i]
					}
					if tmp > val {
						val = tmp
					}
				}
			}
			return val / float64(len(els)), nil
		} else {
			vall := 0.0
			vals := 0.0
			for i := 0; i < l; i++ {
				if i < 3 && (show.SrcanRate&SRCAN_Q != 0) || i >= 3 && (show.SrcanRate&SRCAN_M != 0) {
					tmpl := 0.0
					tmps := 0.0
					for _, el := range els {
						switch i % 3 {
						default:
							continue
						case 0:
							tmpl += el.MaxRate[i]
						case 1:
							tmps += el.MaxRate[i]
						}
					}
					if tmpl > vall {
						vall = tmpl
					}
					if tmps > vals {
						vals = tmps
					}
				}
			}
			if show.SrcanRate&SRCAN_L == 0 {
				vall = 0.0
			}
			if show.SrcanRate&SRCAN_S == 0 {
				vals = 0.0
			}
			if vall >= vals {
				return vall / float64(len(els)), nil
			} else {
				return vals / float64(len(els)), nil
			}
		}
	}
	if elem.IsLineElem() {
		return returnratemax(elem)
	} else {
		if elem.Children != nil {
			return returnratemax(elem.Children...)
		}
		return 0.0, errors.New("RateMax: no value")
	}
}

func (elem *Elem) Energy() (float64, error) {
	if val, ok := elem.Values["ENERGY"]; ok {
		if valb, ok := elem.Values["ENERGYB"]; ok {
			if val == 0.0 {
				return 0.0, nil
			} else {
				return (val - valb) / val, nil
			}
		}
	}
	return 0.0, errors.New("no energy")
}

func (elem *Elem) DistFromProjection(v *View) float64 {
	vec := make([]float64, 3)
	coord := elem.MidPoint()
	for i := 0; i < 3; i++ {
		vec[i] = coord[i] - v.Focus[i]
	}
	return v.Dists[0] - Dot(vec, v.Viewpoint[0], 3)
}

func (elem *Elem) CurrentValue(show *Show, max, abs bool) float64 {
	if show.ElemCaption&EC_NUM != 0 {
		return float64(elem.Num)
	}
	if show.ElemCaption&EC_SECT != 0 {
		return float64(elem.Sect.Num)
	}
	if show.ElemCaption&EC_SIZE != 0 {
		if elem.IsLineElem() {
			return elem.Length()
		} else {
			return elem.Area()
		}
	}
	if show.ElemCaption&EC_WIDTH != 0 {
		return elem.Width()
	}
	if show.ElemCaption&EC_HEIGHT != 0 {
		return elem.Height()
	}
	if show.ElemCaption&EC_PREST != 0 {
		if abs {
			return math.Abs(elem.Prestress) * show.Unit[0]
		} else {
			return elem.Prestress * show.Unit[0]
		}
	}
	if show.ElemCaption&EC_STIFF_X != 0 {
		if abs {
			return math.Abs(elem.LateralStiffness("X", false, false)) * show.Unit[0] / show.Unit[1]
		} else {
			return elem.LateralStiffness("X", false, false) * show.Unit[0] / show.Unit[1]
		}
	}
	if show.ElemCaption&EC_STIFF_Y != 0 {
		if abs {
			return math.Abs(elem.LateralStiffness("Y", false, false)) * show.Unit[0] / show.Unit[1]
		} else {
			return elem.LateralStiffness("Y", false, false) * show.Unit[0] / show.Unit[1]
		}
	}
	if show.ElemCaption&EC_DRIFT_X != 0 {
		return elem.StoryDrift(show.Period, 0, false)
	}
	if show.ElemCaption&EC_DRIFT_Y != 0 {
		return elem.StoryDrift(show.Period, 1, false)
	}
	if show.SrcanRate != 0 {
		val, err := elem.RateMax(show)
		if err != nil {
			return 0.0
		}
		return val
	}
	if show.Energy {
		val, err := elem.Energy()
		if err != nil {
			return 0.0
		}
		return val
	}
	if show.YieldFunction {
		f, err := elem.YieldFunction(show.Period)
		if err[0] != nil || err[1] != nil {
			return 0.0
		}
		if f[0] >= f[1] {
			return f[0]
		} else {
			return f[1]
		}
	}
	var flag uint
	if f, ok := show.Stress[elem.Sect.Num]; ok {
		flag = f
	} else if f, ok := show.Stress[elem.Etype]; ok {
		flag = f
	}
	if flag != 0 {
		for i, st := range []uint{STRESS_NZ, STRESS_QX, STRESS_QY, STRESS_MZ, STRESS_MX, STRESS_MY} {
			if flag&st != 0 {
				switch i {
				case 0:
					if abs {
						return math.Abs(elem.ReturnStress(show.Period, 0, i) * show.Unit[0])
					} else {
						return elem.ReturnStress(show.Period, 0, i) * show.Unit[0]
					}
				case 1, 2:
					v1 := elem.ReturnStress(show.Period, 0, i) * show.Unit[0]
					v2 := elem.ReturnStress(show.Period, 1, i) * show.Unit[0]
					if abs {
						v1 = math.Abs(v1)
						v2 = math.Abs(v2)
					}
					if max {
						if v1 >= v2 {
							return v1
						} else {
							return v2
						}
					} else {
						if v1 >= v2 {
							return v2
						} else {
							return v1
						}
					}
				case 3, 4, 5:
					v1 := elem.ReturnStress(show.Period, 0, i) * show.Unit[0] * show.Unit[1]
					v2 := elem.ReturnStress(show.Period, 1, i) * show.Unit[0] * show.Unit[1]
					if abs {
						v1 = math.Abs(v1)
						v2 = math.Abs(v2)
					}
					if max {
						if v1 >= v2 {
							return v1
						} else {
							return v2
						}
					} else {
						if v1 >= v2 {
							return v2
						} else {
							return v1
						}
					}
				}
			}
		}
	}
	return 0.0
}

func (elem *Elem) DrawDxfSection(d *drawing.Drawing, position []float64, scale float64, vertices [][]float64) {
	var vers [][]float64
	vers = make([][]float64, 0)
	size := 0
	for _, v := range vertices {
		if v == nil {
			d.Polyline(true, vers[:size]...)
			vers = make([][]float64, 0)
			size = 0
			continue
		}
		coord := make([]float64, 3)
		coord[0] = (position[0] + (v[0]*elem.Strong[0]+v[1]*elem.Weak[0])*0.01) * scale
		coord[1] = (position[1] + (v[0]*elem.Strong[1]+v[1]*elem.Weak[1])*0.01) * scale
		coord[2] = (position[2] + (v[0]*elem.Strong[2]+v[1]*elem.Weak[2])*0.01) * scale
		vers = append(vers, coord)
		size++
	}
	d.Polyline(true, vers[:size]...)
}
