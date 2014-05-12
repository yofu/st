package st

import (
    "bytes"
    "fmt"
)

type Prop struct {
    Num  int
    Name string
    Hiju float64
    E    float64
    Poi  float64
    Color int
}

// Sort// {{{
type Props []*Prop
func (p Props) Len() int { return len(p) }
func (p Props) Swap(i, j int) {
    p[i], p[j] = p[j], p[i]
}

type PropByNum struct { Props }
func (p PropByNum) Less(i, j int) bool {
    return p.Props[i].Num < p.Props[j].Num
}
// }}}


func (prop *Prop) InpString () string {
    var rtn bytes.Buffer
    rtn.WriteString(fmt.Sprintf("PROP %d PNAME %s\n", prop.Num, prop.Name))
    rtn.WriteString(fmt.Sprintf("         HIJU %13.5f\n", prop.Hiju))
    rtn.WriteString(fmt.Sprintf("         E    %13.3f\n", prop.E))
    rtn.WriteString(fmt.Sprintf("         POI  %13.5f\n", prop.Poi))
    rtn.WriteString(fmt.Sprintf("         PCOLOR %s\n", IntColor(prop.Color)))
    return rtn.String()
}
