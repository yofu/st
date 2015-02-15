package arclm

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"github.com/yofu/st/matrix"
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

func (frame *Frame) AssemGlobalMatrix() (*matrix.CRSMatrix, error) { // TODO: UNDER CONSTRUCTION
	var err error
	var tmatrix, estiff [][]float64
	gmtx := matrix.NewCOOMatrix(6 * len(frame.Nodes))
	fmt.Printf("MATRIX SIZE: %d\n", 6*len(frame.Nodes))
	start := time.Now()
	for _, el := range frame.Elems {
		tmatrix, err = el.TransMatrix()
		if err != nil {
			return nil, err
		}
		estiff, err = el.StiffMatrix()
		if err != nil {
			return nil, err
		}
		estiff, err = el.ModifyHinge(estiff)
		if err != nil {
			return nil, err
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
	}
	end := time.Now()
	fmt.Printf("ASSEM: %fsec\n", (end.Sub(start)).Seconds())
	rtn := gmtx.ToCRS()
	end = time.Now()
	fmt.Printf("TOCRS: %fsec\n", (end.Sub(start)).Seconds())
	rtn.LDLT()
	end = time.Now()
	fmt.Printf("LDLT : %fsec\n", (end.Sub(start)).Seconds())
	return rtn, nil
}

