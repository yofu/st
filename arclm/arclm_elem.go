package arclm

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/yofu/st/matrix"
)

const (
	RADIUS   = 0.95
	EXPONENT = 1.5
	QUFACT   = 1.25
)

type Sect struct {
	Num      int
	E        float64
	Poi      float64
	Value    []float64
	Yield    []float64
	Type     int
	Exp      float64
	Exq      float64
	Original int
}

var Rigid = &Sect{
	Num:      0,
	E:        0.0,
	Poi:      0.0,
	Value:    []float64{0.0, -1.0, -1.0, 0.0},
	Yield:    []float64{0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0},
	Type:     -1,
	Exp:      0.0,
	Exq:      0.0,
	Original: 0,
}

func NewRigid() *Sect {
	return &Sect{
		Num:      0,
		E:        0.0,
		Poi:      0.0,
		Value:    []float64{0.0, -1.0, -1.0, 0.0},
		Yield:    []float64{0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0},
		Type:     -1,
		Exp:      0.0,
		Exq:      0.0,
		Original: 0,
	}
}

func NewSect() *Sect {
	s := new(Sect)
	s.Value = make([]float64, 4)
	s.Yield = make([]float64, 12)
	return s
}

func (sect *Sect) Snapshot() *Sect {
	s := NewSect()
	s.Num = sect.Num
	s.E = sect.E
	s.Poi = sect.Poi
	for i := 0; i< 4; i++ {
		s.Value[i] = sect.Value[i]
	}
	for i := 0; i< 12; i++ {
		s.Yield[i] = sect.Yield[i]
	}
	s.Type = sect.Type
	s.Exp = sect.Exp
	s.Exq = sect.Exq
	s.Original = sect.Original
	return s
}

func (sect *Sect) InlString() string {
	var rtn bytes.Buffer
	rtn.WriteString(fmt.Sprintf("%5d ", sect.Num))
	rtn.WriteString(fmt.Sprintf("%11.5E %7.5f ", sect.E, sect.Poi))
	rtn.WriteString(fmt.Sprintf("%6.4f %10.8f %10.8f %10.8f", sect.Value[0], sect.Value[1], sect.Value[2], sect.Value[3]))
	for i := 0; i < 12; i++ {
		rtn.WriteString(fmt.Sprintf(" %9.3f", sect.Yield[i]))
	}
	rtn.WriteString(fmt.Sprintf(" %5d", sect.Type))
	rtn.WriteString(fmt.Sprintf(" %5d", sect.Original))
	rtn.WriteString("\n")
	return rtn.String()
}

func ParseArclmSect(words []string) (*Sect, error) {
	var val float64
	var err error
	s := NewSect()
	num, err := strconv.ParseInt(words[0], 10, 64)
	if err != nil {
		return s, err
	}
	s.Num = int(num)
	val, err = strconv.ParseFloat(words[1], 64)
	if err != nil {
		return s, err
	}
	s.E = val
	val, err = strconv.ParseFloat(words[2], 64)
	if err != nil {
		return s, err
	}
	s.Poi = val
	for i := 0; i < 4; i++ {
		val, err = strconv.ParseFloat(words[3+i], 64)
		if err != nil {
			return s, err
		}
		s.Value[i] = val
	}
	for i := 0; i < 12; i++ {
		val, err = strconv.ParseFloat(words[7+i], 64)
		if err != nil {
			return s, err
		}
		s.Yield[i] = val
	}
	if len(words) >= 20 {
		tp, err := strconv.ParseInt(words[19], 10, 64)
		if err != nil {
			return s, err
		}
		s.Type = int(tp)
	}
	return s, nil
}

type Node struct {
	Num      int
	Index    int
	Coord    []float64
	Conf     []bool
	Force    []float64
	Disp     []float64
	Reaction []float64
	Mass     float64
}

func NewNode() *Node {
	n := new(Node)
	n.Coord = make([]float64, 3)
	n.Conf = make([]bool, 6)
	n.Force = make([]float64, 6)
	n.Disp = make([]float64, 6)
	n.Reaction = make([]float64, 6)
	return n
}

func (node *Node) InlCoordString() string {
	return fmt.Sprintf("%5d %7.3f %7.3f %7.3f\n", node.Num, node.Coord[0], node.Coord[1], node.Coord[2])
}

func (node *Node) InlConditionString() string {
	var rtn bytes.Buffer
	rtn.WriteString(fmt.Sprintf("%5d", node.Num))
	for i := 0; i < 6; i++ {
		if node.Conf[i] {
			rtn.WriteString(" 1")
		} else {
			rtn.WriteString(" 0")
		}
	}
	for i := 0; i < 6; i++ {
		rtn.WriteString(fmt.Sprintf(" %12.8f", node.Force[i]))
	}
	rtn.WriteString("\n")
	return rtn.String()
}

func ParseArclmNode(words []string) (*Node, error) {
	var val float64
	var err error
	n := NewNode()
	num, err := strconv.ParseInt(words[0], 10, 64)
	if err != nil {
		return n, err
	}
	n.Num = int(num)
	for i := 0; i < 3; i++ {
		val, err = strconv.ParseFloat(words[1+i], 64)
		if err != nil {
			return n, err
		}
		n.Coord[i] = val
	}
	return n, nil
}

func (node *Node) Parse(words []string) error {
	for i := 0; i < 6; i++ {
		val, err := strconv.ParseInt(words[1+i], 10, 64)
		if err != nil {
			return err
		}
		switch int(val) {
		case 0:
			node.Conf[i] = false
		case 1:
			node.Conf[i] = true
		}
	}
	for i := 0; i < 6; i++ {
		val, err := strconv.ParseFloat(words[7+i], 64)
		if err != nil {
			return err
		}
		node.Force[i] = val
	}
	return nil
}

func (node *Node) OutputDisp() string {
	var otp bytes.Buffer
	otp.WriteString(fmt.Sprintf("%4d", node.Num))
	for j := 0; j < 3; j++ {
		otp.WriteString(fmt.Sprintf(" %10.6f", node.Disp[j]))
	}
	for j := 0; j < 3; j++ {
		otp.WriteString(fmt.Sprintf(" %11.7f", node.Disp[3+j]))
	}
	otp.WriteString("\n")
	return otp.String()
}

const (
	ASIS = iota
	DELETED
	RESTORED
)

type Elem struct {
	Num       int
	Sect      *Sect
	Enod      []*Node
	Cang      float64
	Bonds     []*Sect
	Phinge    []bool
	Strong    []float64
	Weak      []float64
	Cmq       []float64
	Stress    []float64
	Energy    float64
	Energyb   float64
	IsValid   bool
	CheckFunc func() int
}

func NewElem() *Elem {
	el := new(Elem)
	el.Enod = make([]*Node, 2)
	el.Bonds = make([]*Sect, 12)
	for i := 0; i < 12; i++ {
		el.Bonds[i] = NewRigid()
	}
	el.Phinge = make([]bool, 2)
	el.Cmq = make([]float64, 12)
	el.Stress = make([]float64, 12)
	el.IsValid = true
	return el
}

func (elem *Elem) InlString() string {
	var rtn bytes.Buffer
	rtn.WriteString(fmt.Sprintf("%5d %6d ", elem.Num, elem.Sect.Num))
	rtn.WriteString(fmt.Sprintf(" %5d %5d ", elem.Enod[0].Num, elem.Enod[1].Num))
	rtn.WriteString(fmt.Sprintf("%8.5f", elem.Cang))
	for i := 0; i < 2; i++ {
		for j := 3; j < 6; j++ {
			rtn.WriteString(fmt.Sprintf(" %d", elem.Bonds[6*i+j].Num))
		}
	}
	for i := 0; i < 12; i++ {
		rtn.WriteString(fmt.Sprintf(" %10.8f", elem.Cmq[i]))
	}
	rtn.WriteString("\n")
	return rtn.String()
}

func (elem *Elem) Number() int {
	return elem.Num
}

func (elem *Elem) Enode(ind int) int {
	return elem.Enod[ind].Num
}

func (elem *Elem) IsBrace() bool {
	for i := 1; i <= 3; i++ { // IXX, IYY, VEN
		if elem.Sect.Value[i] != 0.0 {
			return false
		}
	}
	return true
}

func ParseArclmElem(words []string, sects []*Sect, nodes []*Node) (*Elem, error) {
	el := NewElem()
	num, err := strconv.ParseInt(words[0], 10, 64)
	if err != nil {
		return el, err
	}
	el.Num = int(num)
	num, err = strconv.ParseInt(words[1], 10, 64)
	if err != nil {
		return el, err
	}
	sec := int(num)
	for _, s := range sects {
		if s.Num == sec {
			el.Sect = s
			break
		}
	}
	if el.Sect == nil {
		return el, errors.New(fmt.Sprintf("ELEM :%d : sect %d not found", el.Num, sec))
	}
	for i := 0; i < 2; i++ {
		tmp, err := strconv.ParseInt(words[2+i], 10, 64)
		if err != nil {
			return el, err
		}
		enod := int(tmp)
		for _, n := range nodes {
			if n.Num == enod {
				el.Enod[i] = n
				break
			}
		}
		if el.Enod[i] == nil {
			return el, errors.New(fmt.Sprintf("ELEM :%d : enod %d not found", el.Num, enod))
		}
	}
	val, err := strconv.ParseFloat(words[4], 64)
	if err != nil {
		return el, err
	}
	el.Cang = val
	for i := 0; i < 6; i++ {
		tmp, err := strconv.ParseInt(words[5+i], 10, 64)
		if err != nil {
			return el, err
		}
		if i < 3 {
			snum := int(tmp)
			for _, s := range sects {
				if s.Num == snum {
					el.Bonds[i+3] = s.Snapshot()
					break
				}
			}
			if el.Bonds[i+3] == nil {
				el.Bonds[i+3] = NewRigid()
			}
		} else {
			snum := int(tmp)
			for _, s := range sects {
				if s.Num == snum {
					el.Bonds[i+6] = s.Snapshot()
					break
				}
			}
			if el.Bonds[i+6] == nil {
				el.Bonds[i+6] = NewRigid()
			}
		}
	}
	for i := 0; i < 12; i++ {
		val, err := strconv.ParseFloat(words[11+i], 64)
		if err != nil {
			return el, err
		}
		el.Cmq[i] = val
		el.Stress[i] = val
	}
	return el, nil
}

func (elem *Elem) Length0() float64 {
	sum := 0.0
	for i := 0; i < 3; i++ {
		sum += math.Pow((elem.Enod[1].Coord[i] - elem.Enod[0].Coord[i]), 2)
	}
	return math.Sqrt(sum)
}

func (elem *Elem) Length() float64 {
	sum := 0.0
	for i := 0; i < 3; i++ {
		sum += math.Pow((elem.Enod[1].Coord[i] + elem.Enod[1].Disp[i] - elem.Enod[0].Coord[i] - elem.Enod[0].Disp[i]), 2)
	}
	return math.Sqrt(sum)
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
		vec[i] = (elem.Enod[1].Coord[i] + elem.Enod[1].Disp[i] - elem.Enod[0].Coord[i] - elem.Enod[0].Disp[i]) / l
	}
	return vec
}

func (elem *Elem) PrincipalAxis(cang float64) ([]float64, []float64, error) {
	d := elem.Direction(true)
	c := math.Cos(cang)
	s := math.Sin(cang)
	strong := make([]float64, 3)
	weak := make([]float64, 3)
	dl1 := 0.0
	dl2 := 0.0
	for i := 0; i < 3; i++ {
		dl1 += d[i] * d[i]
		if i == 2 {
			break
		}
		dl2 += d[i] * d[i]
	}
	dl1 = math.Sqrt(dl1)
	dl2 = math.Sqrt(dl2)
	if dl1 == 0 {
		return strong, weak, errors.New("PrincipalAxis: Length = 0")
	} else if dl2 == 0 {
		strong = []float64{-s, c, 0.0}
		weak = []float64{-c, -s, 0.0}
	} else if dl2/dl1 < 0.1 {
		strong = Cross(d, []float64{c, s, 0.0})
		weak = Cross(d, []float64{-s, c, 0.0})
	} else {
		x := Normalize([]float64{-d[1], d[0], 0.0})
		y := Cross(d, x)
		for i := 0; i < 3; i++ {
			strong[i] = c*x[i] + s*y[i]
			weak[i] = -s*x[i] + c*y[i]
		}
	}
	return Normalize(strong), Normalize(weak), nil
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

func (elem *Elem) Check() int {
	if elem.CheckFunc == nil {
		return ASIS
	}
	return elem.CheckFunc()
}

func (elem *Elem) SetIncompressible(val float64) {
	elem.CheckFunc = func() int {
		if !elem.IsValid {
			if elem.Length() > elem.Length0() {
				elem.IsValid = true
				return RESTORED
			} else {
				return ASIS
			}
		} else {
			if elem.Stress[0] > val { // Comressed
				elem.IsValid = false
				for i := 0; i < 12; i++ {
					elem.Stress[i] = 0.0
				}
				return DELETED
			} else {
				return ASIS
			}
		}
	}
}

func (elem *Elem) Coefficients(estiff [][]float64) ([]float64, [][]float64, [][]float64, [][]float64, []error) {
	err := make([]error, 2)
	fc := make([]float64, 6)
	fu := make([]float64, 6)
	f := make([]float64, 2)
	f1 := make([]float64, 2)
	f2 := make([]float64, 2)
	var v, value, unit float64
	values := make([]float64, 6)
	units := make([]float64, 6)
	dfdp := make([][]float64, 2)
	for j := 0; j < 6; j++ {
		fc[j] = 0.5 * (elem.Sect.Yield[2*j] + elem.Sect.Yield[2*j+1])
		fu[j] = 0.5 * (elem.Sect.Yield[2*j] - elem.Sect.Yield[2*j+1])
	}
	for i := 0; i < 2; i++ {
		for j := 0; j < 6; j++ {
			if elem.Bonds[6*i+j].Num == 1 {
				continue
			}
			switch j {
			case 0, 3: // Nz, Mz
				v = elem.Stress[6*i+j] - math.Pow(-1.0, float64(i))*fc[j]
				value = math.Abs(v) / fu[j]
				values[j] = value
				f1[i] += math.Pow(value, elem.Sect.Exp)
			case 1, 2: // Qx, Qy
				v = elem.Stress[6*i+j] - fc[j]
				value = math.Abs(v) / fu[j]
				values[j] = value
				if value*QUFACT > 1.0 {
					err[i] = BrittleFailure(elem, i)
				}
				f2[i] += math.Pow(value, elem.Sect.Exp)
			case 4, 5: // Mx, My
				v = elem.Stress[6*i+j] - fc[j]
				value = math.Abs(v) / fu[j]
				values[j] = value
				f1[i] += math.Pow(value, elem.Sect.Exp)
			}
			if v == 0.0 {
				unit = 0.0
			} else {
				unit = v / math.Abs(v)
			}
			units[j] = unit
		}
		f[i] = math.Pow(math.Pow(f1[i], elem.Sect.Exq/elem.Sect.Exp)+math.Pow(f2[i], elem.Sect.Exq/elem.Sect.Exp), 1.0/elem.Sect.Exq)
		if f[i] > RADIUS && err[i] == nil {
			err[i] = Yielded(elem, i)
		}
		dfdp[i] = make([]float64, 6)
		for j := 0; j < 6; j++ {
			if elem.Bonds[6*i+j].Num == 1 {
				continue
			}
			if elem.Sect.Exp == elem.Sect.Exq {
				dfdp[i][j] = unit / fu[j] * math.Pow(values[j], elem.Sect.Exp-1.0)
			} else {
				if j == 1 || j == 2 { // Qx, Qy
					dfdp[i][j] = unit / fu[j] * math.Pow(f2[i], elem.Sect.Exq/elem.Sect.Exp-1.0) * math.Pow(values[j], elem.Sect.Exp-1.0)
				} else {
					dfdp[i][j] = unit / fu[j] * math.Pow(f1[i], elem.Sect.Exq/elem.Sect.Exp-1.0) * math.Pow(values[j], elem.Sect.Exp-1.0)
				}
			}
		}
	}
	q := make([][]float64, 12)
	for i := 0; i < 12; i++ {
		q[i] = make([]float64, 12)
		if elem.Bonds[i].Num == 1 {
			continue
		}
		for j := 0; j < 2; j++ {
			for k := 0; k < 6; k++ {
				if elem.Bonds[6*j+k].Num == 1 {
					continue
				}
				q[i][j] += estiff[i][6*j+k] * dfdp[j][k]
			}
		}
	}
	a := make([][]float64, 2)
	for i := 0; i < 2; i++ {
		a[i] = make([]float64, 2)
		for j := 0; j < 2; j++ {
			for k := 0; k < 6; k++ {
				if elem.Bonds[6*i+k].Num == -1 && elem.Bonds[6*j+k].Num == -1 {
					a[i][j] += dfdp[i][k] * q[6*i+k][j]
				}
			}
		}
	}
	return f, dfdp, q, a, err
}

func (elem *Elem) TransMatrix() ([][]float64, error) {
	t := make([][]float64, 12)
	vecs := make([][]float64, 3)
	err := elem.SetPrincipalAxis()
	if err != nil {
		return nil, err
	}
	vecs[0] = elem.Direction(true)
	vecs[1] = elem.Strong
	vecs[2] = elem.Weak
	for n := 0; n < 4; n++ {
		for i := 0; i < 3; i++ {
			t[3*n+i] = make([]float64, 12)
			for j := 0; j < 3; j++ {
				t[3*n+i][3*n+j] = vecs[i][j]
			}
		}
	}
	return t, nil
}

func (elem *Elem) StiffMatrix() ([][]float64, error) {
	il := 1.0 / elem.Length()
	E := elem.Sect.E
	Poi := elem.Sect.Poi
	A := elem.Sect.Value[0]
	IX := elem.Sect.Value[1]
	IY := elem.Sect.Value[2]
	J := elem.Sect.Value[3]
	G := 0.5 * E / (1.0 + Poi)
	estiff := make([][]float64, 12)
	for i := 0; i < 12; i++ {
		estiff[i] = make([]float64, 12)
	}
	// K11
	estiff[0][0] = E * A * il
	estiff[1][1] = 12.0 * E * IY * il * il * il
	estiff[1][5] = 6.0 * E * IY * il * il
	estiff[2][2] = 12.0 * E * IX * il * il * il
	estiff[2][4] = -6.0 * E * IX * il * il
	estiff[3][3] = G * J * il
	estiff[4][4] = 4.0 * E * IX * il
	estiff[5][5] = 4.0 * E * IY * il
	// K12
	estiff[0][6] = -estiff[0][0]
	estiff[1][7] = -estiff[1][1]
	estiff[1][11] = estiff[1][5]
	estiff[2][8] = -estiff[2][2]
	estiff[2][10] = estiff[2][4]
	estiff[3][9] = -estiff[3][3]
	estiff[4][8] = -estiff[2][4]
	estiff[4][10] = 2.0 * E * IX * il
	estiff[5][7] = -estiff[1][5]
	estiff[5][11] = 2.0 * E * IY * il
	// K22
	estiff[6][6] = estiff[0][0]
	estiff[7][7] = estiff[1][1]
	estiff[7][11] = -estiff[1][5]
	estiff[8][8] = estiff[2][2]
	estiff[8][10] = -estiff[2][4]
	estiff[9][9] = estiff[3][3]
	estiff[10][10] = estiff[4][4]
	estiff[11][11] = estiff[5][5]
	// SYM
	estiff[4][2] = estiff[2][4]
	estiff[5][1] = estiff[1][5]
	estiff[6][0] = estiff[0][6]
	estiff[7][1] = estiff[1][7]
	estiff[7][5] = estiff[5][7]
	estiff[8][2] = estiff[2][8]
	estiff[8][4] = estiff[4][8]
	estiff[9][3] = estiff[3][9]
	estiff[10][2] = estiff[2][10]
	estiff[10][4] = estiff[4][10]
	estiff[10][8] = estiff[8][10]
	estiff[11][1] = estiff[1][11]
	estiff[11][5] = estiff[5][11]
	estiff[11][7] = estiff[7][11]
	return estiff, nil
}

func (elem *Elem) PlasticMatrix(estiff [][]float64) ([][]float64, error) {
	_, _, q, a, _ := elem.Coefficients(estiff)
	p := make([][]float64, 12)
	for i := 0; i < 12; i++ {
		p[i] = make([]float64, 12)
	}
	switch {
	case a[0][0] != 0.0 && a[1][1] == 0.0: // I=PLASTIC J=ELASTIC
		for i := 0; i < 12; i++ {
			if elem.Bonds[i].Num == 1 || elem.Bonds[i].Num == -2 || elem.Bonds[i].Num == -3 {
				continue
			}
			for j := 0; j < 12; j++ {
				if elem.Bonds[j].Num == 1 || elem.Bonds[j].Num == -2 || elem.Bonds[j].Num == -3 {
					continue
				}
				p[i][j] = -1.0 / a[0][0] * q[i][0] * q[j][0]
			}
		}
	case a[0][0] == 0.0 && a[1][1] != 0.0: // I=ELASTIC J=PLASTIC
		for i := 0; i < 12; i++ {
			if elem.Bonds[i].Num == 1 || elem.Bonds[i].Num == -2 || elem.Bonds[i].Num == -3 {
				continue
			}
			for j := 0; j < 12; j++ {
				if elem.Bonds[j].Num == 1 || elem.Bonds[j].Num == -2 || elem.Bonds[j].Num == -3 {
					continue
				}
				p[i][j] = -1.0 / a[1][1] * q[i][1] * q[j][1]
			}
		}
	case a[0][0] != 0.0 && a[1][1] != 0.0: // I=PLASTIC J=PLASTIC
		for i := 0; i < 12; i++ {
			if elem.Bonds[i].Num == 1 || elem.Bonds[i].Num == -2 || elem.Bonds[i].Num == -3 {
				continue
			}
			for j := 0; j < 12; j++ {
				if elem.Bonds[j].Num == 1 || elem.Bonds[j].Num == -2 || elem.Bonds[j].Num == -3 {
					continue
				}
				det := a[0][0]*a[1][1] - a[0][1]*a[1][0]
				if det == 0.0 {
					return nil, errors.New(fmt.Sprintf("ELEM %d: matrix singular", elem.Num))
				} else {
					p[i][j] = -1.0 / det * (a[1][1]*q[i][0]*q[j][0] - a[0][1]*q[i][0]*q[j][1] - a[1][0]*q[i][1]*q[j][0] + a[0][0]*q[i][1]*q[j][1])
				}
			}
		}
	}
	for i := 0; i < 12; i++ {
		if elem.Bonds[i].Num == 1 || elem.Bonds[i].Num == -2 || elem.Bonds[i].Num == -3 {
			continue
		}
		for j := 0; j < 12; j++ {
			if elem.Bonds[j].Num == 1 || elem.Bonds[j].Num == -2 || elem.Bonds[j].Num == -3 {
				continue
			}
			estiff[i][j] += p[i][j]
		}
	}
	return estiff, nil
}

func (elem *Elem) PlasticMatrix2(estiff [][]float64) ([][]float64, error) { // ModifyHingeを参考にした版。valが回転剛性
	val := 0.0
	h := make([][]float64, 12)
	rtn := make([][]float64, 12)
	for i := 0; i < 12; i++ {
		h[i] = make([]float64, 12)
		rtn[i] = make([]float64, 12)
		for j := 0; j < 12; j++ {
			rtn[i][j] = estiff[i][j]
		}
	}
	for n := 0; n < 2; n++ {
		if elem.Phinge[n] {
			fmt.Printf("PlasticMatrix2: ELEM %d %d\n", elem.Num, n)
			for i := 4; i < 6; i++ { // TODO check
				kk := 6*n+i
				if estiff[kk][kk] == 0.0 {
					return nil, errors.New(fmt.Sprintf("PlasticMatrix2: ELEM %d: Matrix Singular", elem.Num))
				}
				l := elem.Length()
				k1 := (val * l) / (4.0 * elem.Sect.E * elem.Sect.Value[1])
				k2 := (val * l) / (4.0 * elem.Sect.E * elem.Sect.Value[2])
				for ii := 0; ii < 12; ii++ {
					for jj := 0; jj < 12; jj++ {
						if ii == 2 || ii == 4 || ii == 8 || ii == 10 {
							h[ii][jj] = -rtn[ii][kk] / rtn[kk][kk] * rtn[kk][jj] * 1.0 / (k1 + 1.0)
						} else {
							h[ii][jj] = -rtn[ii][kk] / rtn[kk][kk] * rtn[kk][jj] * 1.0 / (k2 + 1.0)
						}
					}
				}
				for ii := 0; ii < 12; ii++ {
					for jj := 0; jj < 12; jj++ {
						rtn[ii][jj] += h[ii][jj]
					}
				}
			}
		}
	}
	return rtn, nil
}

func (elem *Elem) GeoStiffMatrix() ([][]float64, error) {
	l := elem.Length()
	il := 1.0 / l
	A := elem.Sect.Value[0]
	IX := elem.Sect.Value[1]
	IY := elem.Sect.Value[2]
	G := -(IX + IY) / A
	N := -elem.Stress[0]
	Qx := elem.Stress[1]
	Qy := elem.Stress[2]
	Mxi := elem.Stress[4]
	Myi := elem.Stress[5]
	Mxj := elem.Stress[10]
	Myj := elem.Stress[11]
	estiff := make([][]float64, 12)
	for i := 0; i < 12; i++ {
		estiff[i] = make([]float64, 12)
	}
	estiff[1][1] = 1.2 * N * il
	estiff[1][3] = -Mxi * il
	estiff[3][1] = estiff[1][3]
	estiff[1][5] = 0.1 * N
	estiff[5][1] = estiff[1][5]
	estiff[1][7] = -estiff[1][1]
	estiff[7][1] = estiff[1][7]
	estiff[1][9] = -Mxj * il
	estiff[9][1] = estiff[1][9]
	estiff[1][11] = estiff[1][5]
	estiff[11][1] = estiff[1][11]
	estiff[2][2] = estiff[1][1]
	estiff[2][3] = Myi * il
	estiff[3][2] = estiff[2][3]
	estiff[2][4] = -estiff[1][5]
	estiff[4][2] = estiff[2][4]
	estiff[2][8] = -estiff[1][1]
	estiff[8][2] = estiff[2][8]
	estiff[2][9] = Myj * il
	estiff[9][2] = estiff[2][9]
	estiff[2][10] = -estiff[1][5]
	estiff[10][2] = estiff[2][10]
	estiff[3][3] = N * G * il
	estiff[3][4] = Qx * l / 6.0
	estiff[4][3] = estiff[3][4]
	estiff[3][5] = Qy * l / 6.0
	estiff[5][3] = estiff[3][5]
	estiff[3][7] = -estiff[1][3]
	estiff[7][3] = estiff[3][7]
	estiff[3][8] = -estiff[2][3]
	estiff[8][3] = estiff[3][8]
	estiff[3][9] = -estiff[3][3]
	estiff[9][3] = estiff[3][9]
	estiff[3][10] = -estiff[3][4]
	estiff[10][3] = estiff[3][10]
	estiff[3][11] = -estiff[3][5]
	estiff[11][3] = estiff[3][11]
	estiff[4][4] = N * l / 7.5
	estiff[4][8] = estiff[1][5]
	estiff[8][4] = estiff[4][8]
	estiff[4][9] = -estiff[3][4]
	estiff[9][4] = estiff[4][9]
	estiff[4][10] = -N * l / 30.0
	estiff[10][4] = estiff[4][10]
	estiff[5][5] = estiff[4][4]
	estiff[5][7] = -estiff[1][5]
	estiff[7][5] = estiff[5][7]
	estiff[5][9] = -estiff[3][5]
	estiff[9][5] = estiff[5][9]
	estiff[5][11] = estiff[4][10]
	estiff[11][5] = estiff[5][11]
	estiff[7][7] = estiff[1][1]
	estiff[7][9] = -estiff[1][9]
	estiff[9][7] = estiff[7][9]
	estiff[7][11] = -estiff[1][5]
	estiff[11][7] = estiff[7][11]
	estiff[8][8] = estiff[1][1]
	estiff[8][9] = -estiff[2][9]
	estiff[9][8] = estiff[8][9]
	estiff[8][10] = estiff[1][5]
	estiff[10][8] = estiff[8][10]
	estiff[9][9] = estiff[3][3]
	estiff[9][10] = estiff[3][4]
	estiff[10][9] = estiff[9][10]
	estiff[9][11] = estiff[3][5]
	estiff[11][9] = estiff[9][11]
	estiff[10][10] = estiff[4][4]
	estiff[11][11] = estiff[4][4]
	return estiff, nil
}

func (elem *Elem) ModifyHinge(estiff [][]float64) ([][]float64, error) {
	h := make([][]float64, 12)
	rtn := make([][]float64, 12)
	for i := 0; i < 12; i++ {
		h[i] = make([]float64, 12)
		rtn[i] = make([]float64, 12)
		for j := 0; j < 12; j++ {
			rtn[i][j] = estiff[i][j]
		}
	}
	for n := 0; n < 2; n++ {
		for i := 0; i < 6; i++ {
			// if elem.Bonds[6*n+i] != Rigid {
			if elem.Bonds[6*n+i].Num != 0 {
				kk := 6*n + i
				if rtn[kk][kk] == 0.0 {
					return nil, errors.New(fmt.Sprintf("Modifyhinge: ELEM %d: Matrix Singular", elem.Num))
				}
				l := elem.Length()
				k1 := (elem.Bonds[6*n+i].Value[1] * l) / (4.0 * elem.Sect.E * elem.Sect.Value[1])
				k2 := (elem.Bonds[6*n+i].Value[2] * l) / (4.0 * elem.Sect.E * elem.Sect.Value[2])
				for ii := 0; ii < 12; ii++ {
					for jj := 0; jj < 12; jj++ {
						if ii == 2 || ii == 4 || ii == 8 || ii == 10 {
							h[ii][jj] = -rtn[ii][kk] / rtn[kk][kk] * rtn[kk][jj] * 1.0 / (k1 + 1.0)
						} else {
							h[ii][jj] = -rtn[ii][kk] / rtn[kk][kk] * rtn[kk][jj] * 1.0 / (k2 + 1.0)
						}
					}
				}
				for ii := 0; ii < 12; ii++ {
					for jj := 0; jj < 12; jj++ {
						rtn[ii][jj] += h[ii][jj]
					}
				}
			}
		}
	}
	return rtn, nil
}

// TODO: rotational spring
func (elem *Elem) ModifyCMQ() {
	l := elem.Length()
	if elem.Bonds[4].Num == 1 && elem.Bonds[10].Num == 1 {
		elem.Cmq[4] = 0.0
		elem.Cmq[10] = 0.0
		elem.Stress[4] = 0.0
		elem.Stress[10] = 0.0
	}
	if elem.Bonds[5].Num == 1 && elem.Bonds[11].Num == 1 {
		elem.Cmq[5] = 0.0
		elem.Cmq[11] = 0.0
		elem.Stress[5] = 0.0
		elem.Stress[11] = 0.0
	}
	if elem.Bonds[4].Num == 1 && elem.Bonds[10].Num == 0 {
		elem.Cmq[10] -= elem.Cmq[4] * 0.5
		elem.Cmq[2] += elem.Cmq[4] * 1.5 / l
		elem.Cmq[8] -= elem.Cmq[4] * 1.5 / l
		elem.Cmq[4] = 0.0
		elem.Stress[10] -= elem.Stress[4] * 0.5
		elem.Stress[2] += elem.Stress[4] * 1.5 / l
		elem.Stress[8] -= elem.Stress[4] * 1.5 / l
		elem.Stress[4] = 0.0
	}
	if elem.Bonds[4].Num == 0 && elem.Bonds[10].Num == 1 {
		elem.Cmq[4] -= elem.Cmq[10] * 0.5
		elem.Cmq[2] += elem.Cmq[10] * 1.5 / l
		elem.Cmq[8] -= elem.Cmq[10] * 1.5 / l
		elem.Cmq[10] = 0.0
		elem.Stress[4] -= elem.Stress[10] * 0.5
		elem.Stress[2] += elem.Stress[10] * 1.5 / l
		elem.Stress[8] -= elem.Stress[10] * 1.5 / l
		elem.Stress[10] = 0.0
	}
	if elem.Bonds[5].Num == 1 && elem.Bonds[11].Num == 0 {
		elem.Cmq[11] -= elem.Cmq[5] * 0.5
		elem.Cmq[1] -= elem.Cmq[5] * 1.5 / l
		elem.Cmq[7] += elem.Cmq[5] * 1.5 / l
		elem.Cmq[5] = 0.0
		elem.Stress[11] -= elem.Stress[5] * 0.5
		elem.Stress[1] -= elem.Stress[5] * 1.5 / l
		elem.Stress[7] += elem.Stress[5] * 1.5 / l
		elem.Stress[5] = 0.0
	}
	if elem.Bonds[5].Num == 0 && elem.Bonds[11].Num == 1 {
		elem.Cmq[5] -= elem.Cmq[11] * 0.5
		elem.Cmq[1] -= elem.Cmq[11] * 1.5 / l
		elem.Cmq[7] += elem.Cmq[11] * 1.5 / l
		elem.Cmq[11] = 0.0
		elem.Stress[5] -= elem.Stress[11] * 0.5
		elem.Stress[1] -= elem.Stress[11] * 1.5 / l
		elem.Stress[7] += elem.Stress[11] * 1.5 / l
		elem.Stress[11] = 0.0
	}
}

func (elem *Elem) AssemCMQ(tmatrix [][]float64, vec []float64, safety float64) []float64 {
	rtn := make([]float64, len(vec))
	for i := 0; i < len(vec); i++ {
		rtn[i] = vec[i]
	}
	tt := matrix.MatrixTranspose(tmatrix)
	load := matrix.MatrixVector(tt, elem.Cmq)
	for i := 0; i < 2; i++ {
		for j := 0; j < 6; j++ {
			if !elem.Enod[i].Conf[j] {
				ind := 6*elem.Enod[i].Index + j
				rtn[ind] -= safety * load[6*i+j]
			}
		}
	}
	return rtn
}

func (elem *Elem) ModifyTrueForce(tmatrix [][]float64, vec []float64) []float64 {
	rtn := make([]float64, len(vec))
	for i := 0; i < len(vec); i++ {
		rtn[i] = vec[i]
	}
	tt := matrix.MatrixTranspose(tmatrix)
	tforce := matrix.MatrixVector(tt, elem.Stress)
	for i := 0; i < 2; i++ {
		for j := 0; j < 6; j++ {
			if !elem.Enod[i].Conf[j] {
				ind := 6*elem.Enod[i].Index + j
				rtn[ind] -= tforce[6*i+j]
			}
		}
	}
	return rtn
}

func Transformation(estiff, tmatrix [][]float64) [][]float64 {
	e := matrix.MatrixMatrix(estiff, tmatrix)
	tt := matrix.MatrixTranspose(tmatrix)
	rtn := matrix.MatrixMatrix(tt, e)
	return rtn
}

func (elem *Elem) ElemStress(gdisp []float64) ([]float64, error) {
	tmatrix, err := elem.TransMatrix()
	if err != nil {
		return nil, err
	}
	estiff, err := elem.StiffMatrix()
	if err != nil {
		return nil, err
	}
	estiff, err = elem.ModifyHinge(estiff)
	if err != nil {
		return nil, err
	}
	edisp := matrix.MatrixVector(tmatrix, gdisp)
	estress := matrix.MatrixVector(estiff, edisp)
	for i := 0; i < 12; i++ {
		elem.Stress[i] += estress[i]
	}
	return estress, nil
}

func (elem *Elem) OutputStress() string {
	var otp bytes.Buffer
	for i := 0; i < 2; i++ {
		if i == 0 {
			otp.WriteString(fmt.Sprintf("%5d %4d", elem.Num, elem.Sect.Num))
		} else {
			otp.WriteString("          ")
		}
		otp.WriteString(fmt.Sprintf(" %4d", elem.Enod[i].Num))
		for j := 0; j < 6; j++ {
			otp.WriteString(fmt.Sprintf(" %15.12f", elem.Stress[6*i+j]))
		}
		otp.WriteString("\n")
	}
	return otp.String()
}

func (elem *Elem) StainEnergy(gdisp []float64) (float64, error) {
	tmatrix, err := elem.TransMatrix()
	if err != nil {
		return 0.0, err
	}
	estiff, err := elem.StiffMatrix()
	if err != nil {
		return 0.0, err
	}
	estiff, err = elem.ModifyHinge(estiff)
	if err != nil {
		return 0.0, err
	}
	edisp := matrix.MatrixVector(tmatrix, gdisp)
	estress := matrix.MatrixVector(estiff, edisp)
	Ee := Dot(edisp, estress, 12)
	elem.Energy += Ee
	return Ee, nil
}

func (elem *Elem) BucklingEnergy(gdisp []float64) (float64, error) {
	tmatrix, err := elem.TransMatrix()
	if err != nil {
		return 0.0, err
	}
	estiff, err := elem.StiffMatrix()
	if err != nil {
		return 0.0, err
	}
	gstiff, err := elem.GeoStiffMatrix()
	if err != nil {
		return 0.0, err
	}
	estiff, err = elem.ModifyHinge(estiff)
	if err != nil {
		return 0.0, err
	}
	gstiff, err = elem.ModifyHinge(gstiff)
	if err != nil {
		return 0.0, err
	}
	edisp := matrix.MatrixVector(tmatrix, gdisp)
	estress := matrix.MatrixVector(estiff, edisp)
	gstress := matrix.MatrixVector(gstiff, edisp)
	Ee := Dot(edisp, estress, 12)
	Eb := Dot(edisp, gstress, 12)
	elem.Energy += Ee
	elem.Energyb += Eb
	if Ee == 0.0 {
		return 0.0, nil
	} else {
		return (Ee - Eb) / Ee, nil
	}
}
