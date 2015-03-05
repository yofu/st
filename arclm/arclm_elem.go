package arclm

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/yofu/st/matrix"
	"math"
	"strconv"
)

type Sect struct {
	Num   int
	E     float64
	Poi   float64
	Value []float64
	Yield []float64
	Type  int
	Exp   float64
	Exq   float64
}

func NewSect() *Sect {
	s := new(Sect)
	s.Value = make([]float64, 4)
	s.Yield = make([]float64, 12)
	return s
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

type Elem struct {
	Num    int
	Sect   *Sect
	Enod   []*Node
	Cang   float64
	Bonds  []int
	Strong []float64
	Weak   []float64
	Cmq    []float64
	Stress []float64
}

func NewElem() *Elem {
	el := new(Elem)
	el.Enod = make([]*Node, 2)
	el.Bonds = make([]int, 12)
	el.Cmq = make([]float64, 12)
	el.Stress = make([]float64, 12)
	return el
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
		return el, errors.New(fmt.Sprintf("ELEM :%d : sect not found", el.Num, sec))
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
			el.Bonds[i+3] = int(tmp)
		} else {
			el.Bonds[i+6] = int(tmp)
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

func (elem *Elem) Length() float64 {
	sum := 0.0
	for i := 0; i < 3; i++ {
		sum += math.Pow((elem.Enod[1].Coord[i] + elem.Enod[1].Disp[i] - elem.Enod[0].Coord[i] - elem.Enod[0].Disp[i]), 2)
		// sum += math.Pow((elem.Enod[1].Coord[i] - elem.Enod[0].Coord[i]), 2)
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
		// vec[i] = (elem.Enod[1].Coord[i] - elem.Enod[0].Coord[i]) / l
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
	estiff[0][0] = E * A * il
	estiff[0][6] = -estiff[0][0]
	estiff[1][1] = 12.0 * E * IY * il * il * il
	estiff[1][5] = 6.0 * E * IY * il * il
	estiff[1][7] = -estiff[1][1]
	estiff[1][11] = estiff[1][5]
	estiff[2][2] = 12.0 * E * IX * il * il * il
	estiff[2][4] = -6.0 * E * IX * il * il
	estiff[2][8] = -estiff[2][2]
	estiff[2][10] = estiff[2][4]
	estiff[3][3] = G * J * il
	estiff[3][9] = -estiff[3][3]
	estiff[4][2] = estiff[2][4]
	estiff[4][4] = 4.0 * E * IX * il
	estiff[4][8] = -estiff[2][4]
	estiff[4][10] = 2.0 * E * IX * il
	estiff[5][1] = estiff[1][5]
	estiff[5][5] = 4.0 * E * IY * il
	estiff[5][7] = -estiff[1][5]
	estiff[5][11] = 2.0 * E * IY * il
	estiff[6][0] = estiff[0][6]
	estiff[6][6] = estiff[0][0]
	estiff[7][1] = estiff[1][7]
	estiff[7][5] = estiff[5][7]
	estiff[7][7] = estiff[1][1]
	estiff[7][11] = estiff[5][7]
	estiff[8][2] = estiff[2][8]
	estiff[8][4] = estiff[4][8]
	estiff[8][8] = estiff[2][2]
	estiff[8][10] = estiff[4][8]
	estiff[9][3] = estiff[3][9]
	estiff[9][9] = estiff[3][3]
	estiff[10][2] = estiff[2][10]
	estiff[10][4] = estiff[4][10]
	estiff[10][8] = estiff[8][10]
	estiff[10][10] = estiff[4][4]
	estiff[11][1] = estiff[1][11]
	estiff[11][5] = estiff[5][11]
	estiff[11][7] = estiff[7][11]
	estiff[11][11] = estiff[5][5]
	return estiff, nil
}

func (elem *Elem) GeoStiffMatrix() ([][]float64, error) {
	l := elem.Length()
	il := 1.0 / l
	A := elem.Sect.Value[0]
	IX := elem.Sect.Value[1]
	IY := elem.Sect.Value[2]
	G  := -(IX+IY)/A
	N  := -elem.Stress[0]
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
			if elem.Bonds[6*n+i] == 1 {
				kk := 6*n + i
				if rtn[kk][kk] == 0.0 {
					return nil, errors.New(fmt.Sprintf("Modifyhinge: ELEM %d: Matrix Singular", elem.Num))
				}
				for ii := 0; ii < 12; ii++ {
					for jj := 0; jj < 12; jj++ {
						h[ii][jj] = -rtn[ii][kk] / rtn[kk][kk] * rtn[kk][jj]
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

func (elem *Elem) ModifyCMQ() {
	l := elem.Length()
	if elem.Bonds[4] == 1 && elem.Bonds[10] == 1 {
		elem.Cmq[4] = 0.0
		elem.Cmq[10] = 0.0
	}
	if elem.Bonds[5] == 1 && elem.Bonds[11] == 1 {
		elem.Cmq[5] = 0.0
		elem.Cmq[11] = 0.0
	}
	if elem.Bonds[4] == 1 && elem.Bonds[10] == 0 {
		elem.Cmq[10] -= elem.Cmq[4] * 0.5
		elem.Cmq[2] += elem.Cmq[4] * 1.5 / l
		elem.Cmq[8] -= elem.Cmq[4] * 1.5 / l
		elem.Cmq[4] = 0.0
	}
	if elem.Bonds[4] == 0 && elem.Bonds[10] == 1 {
		elem.Cmq[4] -= elem.Cmq[10] * 0.5
		elem.Cmq[2] += elem.Cmq[10] * 1.5 / l
		elem.Cmq[8] -= elem.Cmq[10] * 1.5 / l
		elem.Cmq[10] = 0.0
	}
	if elem.Bonds[5] == 1 && elem.Bonds[11] == 0 {
		elem.Cmq[11] -= elem.Cmq[5] * 0.5
		elem.Cmq[1] -= elem.Cmq[5] * 1.5 / l
		elem.Cmq[7] += elem.Cmq[5] * 1.5 / l
		elem.Cmq[5] = 0.0
	}
	if elem.Bonds[5] == 0 && elem.Bonds[11] == 1 {
		elem.Cmq[5] -= elem.Cmq[11] * 0.5
		elem.Cmq[1] -= elem.Cmq[11] * 1.5 / l
		elem.Cmq[7] += elem.Cmq[11] * 1.5 / l
		elem.Cmq[11] = 0.0
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
