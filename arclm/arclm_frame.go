package arclm

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/yofu/st/matrix"
	"io"
	"io/ioutil"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	CRS    = iota
	CRS_CG
	LLS
	LLS_CG
	LLS_PCG
)

type Frame struct {
	Sects []*Sect
	Nodes []*Node
	Elems []*Elem
	Lapch chan int
	Endch chan error
}

func NewFrame() *Frame {
	af := new(Frame)
	af.Sects = make([]*Sect, 0)
	af.Nodes = make([]*Node, 0)
	af.Elems = make([]*Elem, 0)
	af.Lapch = make(chan int)
	af.Endch = make(chan error)
	return af
}

func (af *Frame) ReadInput(filename string) error {
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
	split := func(str string) []string {
		var words []string
		for _, k := range strings.Split(str, " ") {
			if k != "" {
				words = append(words, k)
			}
		}
		return words
	}
	words := split(lis[0])
	nums := make([]int, 3)
	for i := 0; i < 3; i++ {
		num, err := strconv.ParseInt(words[i], 10, 64)
		if err != nil {
			return err
		}
		nums[i] = int(num)
	}
	af.Nodes = make([]*Node, nums[0])
	af.Elems = make([]*Elem, nums[1])
	af.Sects = make([]*Sect, nums[2])
	// Sect
	for i, j := range lis[1 : 1+nums[2]] {
		words := split(j)
		if len(words) == 0 {
			continue
		}
		s, err := ParseArclmSect(words)
		if err != nil {
			return err
		}
		af.Sects[i] = s
	}
	// Node1
	for i, j := range lis[1+nums[2] : 1+nums[2]+nums[0]] {
		words := split(j)
		if len(words) == 0 {
			continue
		}
		n, err := ParseArclmNode(words)
		if err != nil {
			return err
		}
		n.Index = i
		af.Nodes[i] = n
	}
	// Elem
	for i, j := range lis[1+nums[2]+nums[0] : 1+nums[2]+nums[0]+nums[1]] {
		words := split(j)
		if len(words) == 0 {
			continue
		}
		el, err := ParseArclmElem(words, af.Sects, af.Nodes)
		if err != nil {
			return err
		}
		af.Elems[i] = el
	}
	// Node2
	for i, j := range lis[1+nums[2]+nums[0]+nums[1] : 1+nums[2]+nums[0]+nums[1]+nums[0]] {
		words := split(j)
		if len(words) == 0 {
			continue
		}
		num, err := strconv.ParseInt(words[0], 10, 64)
		if err != nil {
			return err
		}
		nnum := int(num)
		if af.Nodes[i].Num != nnum {
			return errors.New(fmt.Sprintf("ARCLM: NODE %d: format error", nnum))
		}
		af.Nodes[i].Parse(words)
	}
	return nil
}

func (frame *Frame) Initialise() {
	for _, n := range frame.Nodes {
		for i:=0; i<6; i++ {
			n.Disp[i] = 0.0
			n.Reaction[i] = 0.0
		}
	}
	for _, el := range frame.Elems {
		for i:=0; i<12; i++ {
			el.Stress[i] = el.Cmq[i]
		}
	}
}

func (frame *Frame) AssemGlobalMatrix(matf func(*Elem)([][]float64, error), vecf func(*Elem, [][]float64, []float64, float64)([]float64), safety float64) (*matrix.COOMatrix, []float64, error) { // TODO: UNDER CONSTRUCTION
	var err error
	var tmatrix, stiff [][]float64
	size := 6 * len(frame.Nodes)
	gmtx := matrix.NewCOOMatrix(size)
	gvct := make([]float64, size)
	for _, el := range frame.Elems {
		tmatrix, err = el.TransMatrix()
		if err != nil {
			return nil, nil, err
		}
		stiff, err = matf(el)
		if err != nil {
			return nil, nil, err
		}
		stiff, err = el.ModifyHinge(stiff)
		if err != nil {
			return nil, nil, err
		}
		stiff = Transformation(stiff, tmatrix)
		for n1 := 0; n1 < 2; n1++ {
			for i := 0; i < 6; i++ {
				row := 6*el.Enod[n1].Index + i
				for n2 := 0; n2 < 2; n2++ {
					for j := 0; j < 6; j++ {
						col := 6*el.Enod[n2].Index + j
						val := stiff[6*n1+i][6*n2+j]
						if val != 0.0 {
							gmtx.Add(row, col, val)
						}
					}
				}
			}
		}
		el.ModifyCMQ()
		gvct = vecf(el, tmatrix, gvct, safety)
	}
	return gmtx, gvct, nil
}

func (frame *Frame) KE(safety float64) (*matrix.COOMatrix, []float64, error) { // TODO: UNDER CONSTRUCTION
	matf := func(elem *Elem) ([][]float64, error) {
		return elem.StiffMatrix()
	}
	vecf := func(elem *Elem, tmatrix [][]float64, gvct []float64, safety float64) ([]float64) {
		return elem.AssemCMQ(tmatrix, gvct, safety)
	}
	return frame.AssemGlobalMatrix(matf, vecf, safety)
}

func (frame *Frame) KEKG(safety float64) (*matrix.COOMatrix, []float64, error) { // TODO: UNDER CONSTRUCTION
	matf := func(elem *Elem) ([][]float64, error) {
		estiff, err := elem.StiffMatrix()
		if err != nil {
			return nil, err
		}
		gstiff, err := elem.GeoStiffMatrix()
		if err != nil {
			return nil, err
		}
		stiff := make([][]float64, 12)
		for i := 0; i < 12; i++ {
			stiff[i] = make([]float64, 12)
		}
		for i:=0; i<12; i++ {
			for j:=0; j<12; j++ {
				stiff[i][j] = estiff[i][j] + gstiff[i][j]
			}
		}
		return stiff, nil
	}
	vecf := func(elem *Elem, tmatrix [][]float64, gvct []float64, safety float64) ([]float64) {
		gvct = elem.AssemCMQ(tmatrix, gvct, safety)
		gvct = elem.ModifyTrueForce(tmatrix, gvct)
		return gvct
	}
	return frame.AssemGlobalMatrix(matf, vecf, safety)
}

func (frame *Frame) AssemConf(gvct []float64, safety float64) (int, []bool, []float64) {
	size := 6 * len(frame.Nodes)
	csize := 0
	conf := make([]bool, size)
	vec := make([]float64, size)
	for i, n := range frame.Nodes {
		for j := 0; j < 6; j++ {
			if n.Conf[j] {
				conf[6*i+j] = true
				csize++
			} else {
				vec[6*i+j-csize] = gvct[6*i+j] + safety * n.Force[j]
			}
		}
	}
	return csize, conf, vec[:size-csize]
}

func (frame *Frame) FillConf(vec []float64) []float64 {
	ind := 0
	rtn := make([]float64, 6 * len(frame.Nodes))
	for i, n := range frame.Nodes {
		for j := 0; j < 6; j++ {
			if n.Conf[j] {
				ind++
				continue
			}
			rtn[6*i+j] = vec[6*i+j-ind]
		}
	}
	return rtn
}

func (frame *Frame) UpdateStress(vec []float64) ([][]float64, error) {
	rtn := make([][]float64, len(frame.Elems))
	for enum, el := range frame.Elems {
		gdisp := make([]float64, 12)
		for i := 0; i < 2; i++ {
			for j := 0; j < 6; j++ {
				gdisp[6*i+j] = vec[6*el.Enod[i].Index+j]
			}
		}
		df, err := el.ElemStress(gdisp)
		if err != nil {
			return nil, err
		}
		rtn[enum] = df
	}
	return rtn, nil
}

func (frame *Frame) UpdateReaction(gmtx *matrix.COOMatrix, vec []float64) []float64 {
	rtn := make([]float64, 6 * len(frame.Nodes))
	for i, n := range frame.Nodes {
		for j := 0; j < 6; j++ {
			if n.Conf[j] {
				val := 0.0
				for k := 0; k < gmtx.Size; k++ {
					stiff := gmtx.Query(6*i+j, k)
					val += stiff * vec[k]
				}
				n.Reaction[j] += val
				rtn[6*i+j] = val
			}
		}
	}
	return rtn
}

func (frame *Frame) UpdateForm(vec []float64) {
	for i, n := range frame.Nodes {
		for j := 0; j < 6; j++ {
			n.Disp[j] += vec[6*i+j]
		}
	}
}

func (frame *Frame) WriteTo(w io.Writer) {
	var otp bytes.Buffer
	var rea bytes.Buffer
	otp.WriteString("\n\n** FORCES OF MEMBER\n\n")
	otp.WriteString("  NO   KT NODE         N        Q1        Q2        MT        M1        M2\n\n")
	for _, el := range frame.Elems {
		otp.WriteString(el.OutputStress())
	}
	otp.WriteString("\n\n** DISPLACEMENT OF NODE\n\n")
	otp.WriteString("  NO          U          V          W         KSI         ETA       OMEGA\n\n")
	rea.WriteString("\n\n** REACTION\n\n")
	rea.WriteString("  NO  DIRECTION              R    NC\n\n")
	for _, n := range frame.Nodes {
		otp.WriteString(fmt.Sprintf("%4d", n.Num))
		for j := 0; j < 3; j++ {
			otp.WriteString(fmt.Sprintf(" %10.6f", n.Disp[j]))
			if n.Conf[j] {
				rea.WriteString(fmt.Sprintf("%4d %10d %14.6f     1\n", n.Num, j+1, n.Reaction[j]))
			}
		}
		for j := 3; j < 6; j++ {
			otp.WriteString(fmt.Sprintf(" %11.7f", n.Disp[j]))
			if n.Conf[j] {
				rea.WriteString(fmt.Sprintf("%4d %10d %14.6f     1\n", n.Num, j+1, n.Reaction[j]))
			}
		}
		otp.WriteString("\n")
	}
	otp.WriteTo(w)
	rea.WriteTo(w)
}

func (frame *Frame) Arclm001(init bool, sol string) error { // TODO: speed up
	frame.Initialise()
	var solver int
	switch strings.ToUpper(sol) {
	default:
		solver = LLS
	case "CRS":
		solver = CRS_CG
	case "LLS":
		solver = LLS
	case "CG":
		solver = LLS_CG
	case "PCG":
		solver = LLS_PCG
	}
	start := time.Now()
	laptime := func (message string) {
		end := time.Now()
		fmt.Printf("%s: %fsec\n", message, (end.Sub(start)).Seconds())
	}
	gmtx, gvct, err := frame.KE(1.0)
	laptime("ASSEM")
	if err != nil {
		return err
	}
	csize, conf, vec := frame.AssemConf(gvct, 1.0)
	vecs := [][]float64{vec}
	laptime("VEC")
	var answers [][]float64
	switch solver {
	case CRS:
		mtx := gmtx.ToCRS(csize, conf)
		laptime("ToCRS")
		answers = mtx.Solve(vecs...)
		laptime("Solve")
	case CRS_CG:
		mtx := gmtx.ToCRS(csize, conf)
		answers = make([][]float64, len(vecs))
		for i, vec := range vecs {
			answers[i] = mtx.CG(vec)
		}
	case LLS:
		mtx := gmtx.ToLLS(csize, conf)
		laptime("ToLLS")
		answers = mtx.Solve(vecs...)
		laptime("Solve")
	case LLS_CG:
		dbg, _ := os.Create("debug.txt")
		os.Stdout = dbg
		mtx := gmtx.ToLLS(csize, conf)
		mtx.DiagUp()
		laptime("ToLLS")
		answers = make([][]float64, len(vecs))
		for i, vec := range vecs {
			answers[i] = mtx.CG(vec)
		}
		laptime("Solve")
	case LLS_PCG:
		dbg, _ := os.Create("debug.txt")
		os.Stdout = dbg
		mtx := gmtx.ToLLS(csize, conf)
		C := gmtx.ToLLS(csize, conf)
		mtx.DiagUp()
		laptime("ToLLS")
		answers = make([][]float64, len(vecs))
		for i, vec := range vecs {
			answers[i] = mtx.PCG(C,vec)
		}
		laptime("Solve")
	}
	for nans, ans := range answers {
		vec := frame.FillConf(ans)
		_, err := frame.UpdateStress(vec)
		if err != nil {
			return err
		}
		frame.UpdateReaction(gmtx, vec)
		frame.UpdateForm(vec)
		w, err := os.Create(fmt.Sprintf("hogtxt_%02d.otp", nans))
		if err != nil {
			return err
		}
		defer w.Close()
		frame.WriteTo(w)
	}
	return nil
}

func (frame *Frame) Arclm201(init bool, nlap int, dsafety float64) error { // TODO: speed up
	frame.Initialise()
	start := time.Now()
	laptime := func (message string) {
		end := time.Now()
		fmt.Printf("%s: %fsec\n", message, (end.Sub(start)).Seconds())
	}
	var err error
	var answers [][]float64
	var gmtx *matrix.COOMatrix
	var gvct, vec []float64
	var bnorm, rnorm float64
	var csize int
	var conf []bool
	safety := 0.0
	for lap:=0; lap<nlap; lap++ {
		safety += dsafety
		if safety > 1.0 {
			safety = 1.0
		}
		if lap == 0 { // K = KE
			gmtx, gvct, err = frame.KE(safety)
			csize, conf, vec = frame.AssemConf(gvct, safety)
			bnorm = math.Sqrt(Dot(vec, vec, len(vec)))
			for _, el := range frame.Elems {
				for i:=0; i<12; i++ {
					el.Stress[i] = 0.0
				}
			}
		} else {      // K = KE + KG
			gmtx, gvct, err = frame.KEKG(safety)
			csize, conf, vec = frame.AssemConf(gvct, safety)
			rnorm = math.Sqrt(Dot(vec, vec, len(vec)))
		}
		if err != nil {
			return err
		}
		laptime("Assem")
		mtx := gmtx.ToLLS(csize, conf)
		laptime("ToLLS")
		answers = mtx.Solve(vec)
		laptime("Solve")
		tmp := frame.FillConf(answers[0])
		_, err = frame.UpdateStress(tmp)
		if err != nil {
			return err
		}
		frame.UpdateReaction(gmtx, tmp)
		frame.UpdateForm(tmp)
		laptime(fmt.Sprintf("%04d / %04d: SAFETY = %.3f NORM = %.5E", lap+1, nlap, safety, rnorm / bnorm))
		frame.Lapch <- lap+1
		<-frame.Lapch
	}
	frame.Endch <- nil
	w, err := os.Create("hogtxt.otp")
	if err != nil {
		return err
	}
	defer w.Close()
	frame.WriteTo(w)
	return nil
}
