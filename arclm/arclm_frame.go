package arclm

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"github.com/yofu/st/matrix"
	"os"
	"time"
)

type Frame struct {
	Sects []*Sect
	Nodes []*Node
	Elems []*Elem
}

func NewFrame() *Frame {
	af := new(Frame)
	af.Sects = make([]*Sect, 0)
	af.Nodes = make([]*Node, 0)
	af.Elems = make([]*Elem, 0)
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

func (frame *Frame) AssemGlobalMatrix() (*matrix.COOMatrix, []float64, error) { // TODO: UNDER CONSTRUCTION
	var err error
	var tmatrix, estiff [][]float64
	size := 6*len(frame.Nodes)
	gmtx := matrix.NewCOOMatrix(size)
	gvct := make([]float64, size)
	fmt.Printf("MATRIX SIZE: %d\n", size)
	start := time.Now()
	for _, el := range frame.Elems {
		tmatrix, err = el.TransMatrix()
		if err != nil {
			return nil, nil, err
		}
		estiff, err = el.StiffMatrix()
		if err != nil {
			return nil, nil, err
		}
		estiff, err = el.ModifyHinge(estiff)
		if err != nil {
			return nil, nil, err
		}
		estiff = Transformation(estiff, tmatrix)
		for n1 := 0; n1 < 2; n1++ {
			for i := 0; i < 6; i++ {
				row := 6*el.Enod[n1].Index + i
				for n2 := 0; n2 < 2; n2++ {
					for j := 0; j < 6; j++ {
						col := 6*el.Enod[n2].Index + j
						if row >= col {
							val := estiff[6*n1+i][6*n2+j]
							if val != 0.0 {
								gmtx.Add(row, col, val)
							}
						}
					}
				}
			}
		}
		el.ModifyCMQ()
		gvct = el.AssemCMQ(tmatrix, gvct)
	}
	end := time.Now()
	fmt.Printf("ASSEM: %fsec\n", (end.Sub(start)).Seconds())
	return gmtx, gvct, nil
}

func (frame *Frame) Arclm001() error { // TODO: test
	var otp bytes.Buffer
	gmtx, gvct, err := frame.AssemGlobalMatrix()
	if err != nil {
		return err
	}
	mtx := gmtx.ToCRS()
	size := 6*len(frame.Nodes)
	conf := make([]bool, size)
	vecs := make([][]float64, 1)
	vecs[0] = make([]float64, size)
	for i, n := range frame.Nodes {
		for j:=0; j<6; j++ {
			vecs[0][6*i+j] = gvct[6*i+j]
			if n.Conf[j] {
				conf[6*i+j] = true
			} else {
				vecs[0][6*i+j] += n.Force[j]
			}
		}
	}
	answers := mtx.Solve(conf, vecs...)
	for nans, ans := range answers {
		fmt.Println("STRESS")
		otp.WriteString("\n\n** FORCES OF MEMBER\n\n")
		otp.WriteString("  NO   KT NODE         N        Q1        Q2        MT        M1        M2\n\n")
		for _, el := range frame.Elems {
			gdisp := make([]float64, 12)
			for i:=0; i<2; i++ {
				for j:=0; j<6; j++ {
					gdisp[6*i+j] = ans[6*el.Enod[i].Index+j]
				}
			}
			_, err := el.ElemStress(gdisp)
			if err != nil {
				return err
			}
			otp.WriteString(el.OutputStress())
		}
		fmt.Println("DISPLACEMENT")
		otp.WriteString("\n\n** DISPLACEMENT OF NODE\n\n")
		otp.WriteString("  NO          U          V          W         KSI         ETA       OMEGA\n\n")
		for _, n := range frame.Nodes {
			otp.WriteString(n.OutputDisp())
		}
		fmt.Println("REACTION");
		otp.WriteString("\n\n** REACTION\n\n")
		otp.WriteString("  NO  DIRECTION              R    NC\n\n")
		for i, n := range frame.Nodes {
			for j:=0; j<6; j++ {
				if n.Conf[j] {
					val := 0.0
					for k:=0; k<mtx.Size; k++ {
						stiff := mtx.Query(6*i+j, k)
						val += stiff * ans[6*i+j]
					}
					n.Reaction[j] += val
					otp.WriteString(fmt.Sprintf("%4d %10d %14.6f     1\n", n.Num, j+1, val))
				}
			}
		}
		fmt.Println("SET DISPLACEMENT")
		for i, n := range frame.Nodes {
			for j:=0; j<6; j++ {
				n.Disp[j] += ans[6*i+j]
			}
		}
		w, err := os.Create(fmt.Sprintf("hogtxt_%02d.otp", nans))
		if err != nil {
			return err
		}
		defer w.Close()
		otp.WriteTo(w)
	}
	return nil
}