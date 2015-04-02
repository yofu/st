package st

import (
	"bytes"
	"fmt"
)

type Prop struct {
	Num   int
	Name  string
	Hiju  float64
	E     float64
	Poi   float64
	Color int
}

// Sort// {{{
type Props []*Prop

func (p Props) Len() int { return len(p) }
func (p Props) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

type PropByNum struct{ Props }

func (p PropByNum) Less(i, j int) bool {
	return p.Props[i].Num < p.Props[j].Num
}

// }}}

func (prop *Prop) Snapshot() *Prop {
	p := new(Prop)
	p.Num = prop.Num
	p.Name = prop.Name
	p.Hiju = prop.Hiju
	p.E = prop.E
	p.Poi = prop.Poi
	p.Color = prop.Color
	return p
}

func (prop *Prop) InpString() string {
	var rtn bytes.Buffer
	rtn.WriteString(fmt.Sprintf("PROP %d PNAME %s\n", prop.Num, prop.Name))
	rtn.WriteString(fmt.Sprintf("         HIJU %13.5f\n", prop.Hiju))
	rtn.WriteString(fmt.Sprintf("         E    %13.3f\n", prop.E))
	rtn.WriteString(fmt.Sprintf("         POI  %13.5f\n", prop.Poi))
	rtn.WriteString(fmt.Sprintf("         PCOLOR %s\n", IntColor(prop.Color)))
	return rtn.String()
}

func (prop *Prop) IsSteel(eps float64) bool {
	if val := prop.E/2.1e7 - 1.0; val < -eps || val > eps {
		return false
	}
	if val := prop.Poi*3.0 - 1.0; val < -eps || val > eps {
		return false
	}
	return true
}

func (prop *Prop) IsRc(eps float64) bool {
	if val := prop.E/2.1e6 - 1.0; val < -eps || val > eps {
		return false
	}
	if val := prop.Poi*6.0 - 1.0; val < -eps || val > eps {
		return false
	}
	return true
}

func (prop *Prop) IsGohan(eps float64) bool {
	if val := prop.E/4.5e5 - 1.0; val < -eps || val > eps {
		return false
	}
	if val := prop.Poi/4.625 - 1.0; val < -eps || val > eps {
		return false
	}
	return true
}
