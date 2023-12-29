package st

import (
	"bytes"
	"fmt"
)

type Bond struct {
	Num       int
	Name      string
	Stiffness []float64
	Plastic   bool
}

var Pin = &Bond{
	Num:       1,
	Name:      "PIN",
	Stiffness: []float64{0.0, 0.0},
	Plastic:   false,
}

type Bonds []*Bond

func (b Bonds) Len() int { return len(b) }
func (b Bonds) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

type BondByNum struct{ Bonds }

func (b BondByNum) Less(i, j int) bool {
	return b.Bonds[i].Num < b.Bonds[j].Num
}

func (bond *Bond) Snapshot() *Bond {
	b := new(Bond)
	b.Num = bond.Num
	b.Name = bond.Name
	b.Stiffness = make([]float64, 2)
	for i := 0; i < 2; i++ {
		b.Stiffness[i] = bond.Stiffness[i]
	}
	b.Plastic = bond.Plastic
	return b
}

func (bond *Bond) InpString() string {
	var rtn bytes.Buffer
	rtn.WriteString(fmt.Sprintf("BOND %d BNAME %s\n", bond.Num, bond.Name))
	rtn.WriteString(fmt.Sprintf("         KR %8.3f %8.3f\n", bond.Stiffness[0], bond.Stiffness[1]))
	return rtn.String()
}
