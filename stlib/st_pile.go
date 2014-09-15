package st

import (
    "bytes"
    "fmt"
)

type Pile struct {
    Num  int
    Name string
    Moment float64
}

// Sort// {{{
type Piles []*Pile
func (p Piles) Len() int { return len(p) }
func (p Piles) Swap(i, j int) {
    p[i], p[j] = p[j], p[i]
}

type PileByNum struct { Piles }
func (p PileByNum) Less(i, j int) bool {
    return p.Piles[i].Num < p.Piles[j].Num
}
// }}}


func (pile *Pile) InpString () string {
    var rtn bytes.Buffer
    rtn.WriteString(fmt.Sprintf("PILE %d INAME %s\n", pile.Num, pile.Name))
    rtn.WriteString(fmt.Sprintf("         MOMENT %8.3f\n", pile.Moment))
    return rtn.String()
}
