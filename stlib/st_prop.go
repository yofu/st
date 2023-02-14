package st

import (
	"bytes"
	"fmt"
)

type Prop struct {
	Material
	Num     int
	Name    string
	HFactor float64
	EFactor float64
	hiju    float64
	eL      float64
	eS      float64
	poi     float64
	Color   int
}

type Props []*Prop

func (p Props) Len() int { return len(p) }
func (p Props) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

type PropByNum struct{ Props }

func (p PropByNum) Less(i, j int) bool {
	return p.Props[i].Num < p.Props[j].Num
}

func (prop *Prop) Snapshot() *Prop {
	p := new(Prop)
	p.Material = prop.Material
	p.HFactor = prop.HFactor
	p.EFactor = prop.EFactor
	p.Num = prop.Num
	p.Name = prop.Name
	p.hiju = prop.hiju
	p.eL = prop.eL
	p.eS = prop.eS
	p.poi = prop.poi
	p.Color = prop.Color
	return p
}

func (p *Prop) Hiju() float64 {
	if p.Material != nil {
		return p.Material.Hiju()*p.HFactor
	}
	return p.hiju
}

func (p *Prop) EL() float64 {
	if p.Material != nil {
		return p.Material.EL()*1e4*p.EFactor
	}
	return p.eL
}

func (p *Prop) ES() float64 {
	if p.Material != nil {
		return p.Material.ES()*1e4*p.EFactor
	}
	return p.eS
}

func (p *Prop) Poi() float64 {
	if p.Material != nil {
		return p.Material.Poi()
	}
	return p.poi
}

func (prop *Prop) InpString() string {
	var rtn bytes.Buffer
	rtn.WriteString(fmt.Sprintf("PROP %d PNAME %s\n", prop.Num, prop.Name))
	if prop.Material != nil {
		rtn.WriteString(fmt.Sprintf("         MATERIAL %9s\n", prop.Material.Name()))
		rtn.WriteString(fmt.Sprintf("         HFACT %12.5f\n", prop.HFactor))
		rtn.WriteString(fmt.Sprintf("         EFACT %12.5f\n", prop.EFactor))
	} else {
		rtn.WriteString(fmt.Sprintf("         HIJU %13.5f\n", prop.Hiju()))
		rtn.WriteString(fmt.Sprintf("         E    %13.3f\n", prop.EL()))
		if prop.EL() != prop.ES() {
			rtn.WriteString(fmt.Sprintf("         ES   %13.3f\n", prop.ES()))
		}
		rtn.WriteString(fmt.Sprintf("         POI  %13.5f\n", prop.Poi()))
	}
	rtn.WriteString(fmt.Sprintf("         PCOLOR %s\n", IntColor(prop.Color)))
	return rtn.String()
}

func (prop *Prop) IsSteel(eps float64) bool {
	if prop.Material != nil {
		if _, ok := prop.Material.(Steel); ok {
			return true
		}
	}
	E0 := 2.1e7
	p0 := 7.8
	E := E0 * prop.Hiju() / p0
	if val := prop.EL()/E - 1.0; val < -eps || val > eps {
		return false
	}
	if val := prop.Poi()*3.0 - 1.0; val < -eps || val > eps {
		return false
	}
	return true
}

func (prop *Prop) IsRc(eps float64) bool {
	if prop.Material != nil {
		if _, ok := prop.Material.(Concrete); ok {
			return true
		}
	}
	if val := prop.EL()/2.1e6 - 1.0; val < -eps || val > eps {
		return false
	}
	if val := prop.Poi()*6.0 - 1.0; val < -eps || val > eps {
		return false
	}
	return true
}

func (prop *Prop) IsPc(eps float64) bool {
	if val := prop.EL()/3.6e6 - 1.0; val < -eps || val > eps {
		return false
	}
	if val := prop.Poi()*5.0 - 1.0; val < -eps || val > eps {
		return false
	}
	return true
}

func (prop *Prop) IsWood(E float64, eps float64) bool {
	if prop.Material != nil {
		if _, ok := prop.Material.(Wood); ok {
			return true
		}
	}
	// if val := prop.ES/E - 1.0; val < -eps || val > eps {
	// 	return false
	// }
	if val := prop.Poi()/6.5 - 1.0; val < -eps || val > eps {
		return false
	}
	return true
}

func (prop *Prop) IsGohan(eps float64) bool {
	if val := prop.ES()/4.5e5 - 1.0; val < -eps || val > eps {
		return false
	}
	if val := prop.Poi()/4.625 - 1.0; val < -eps || val > eps {
		return false
	}
	return true
}
