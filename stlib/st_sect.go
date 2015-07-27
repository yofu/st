package st

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/yofu/st/arclm"
	"sort"
)

var (
	FIGKEYS = []string{"AREA", "IXX", "IYY", "VEN", "THICK", "SREIN", "XFACE", "YFACE"}
)

type Sect struct {
	Frame *Frame
	Num   int
	Name  string
	Figs  []*Fig
	Exp   float64
	Exq   float64
	Lload []float64
	Yield []float64
	Type  int
	Original int
	Color int
}

type Fig struct {
	Num   int
	Prop  *Prop
	Value map[string]float64
}

// Sort// {{{
type Sects []*Sect

func (s Sects) Len() int { return len(s) }
func (s Sects) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type SectByNum struct{ Sects }

func (s SectByNum) Less(i, j int) bool {
	return s.Sects[i].Num < s.Sects[j].Num
}

// }}}

func NewSect() *Sect {
	s := new(Sect)
	s.Figs = make([]*Fig, 0)
	s.Exp = arclm.EXPONENT
	s.Exq = arclm.EXPONENT
	s.Lload = make([]float64, 3)
	s.Yield = make([]float64, 12)
	s.Color = 16777215
	return s
}

func (sect *Sect) Snapshot(frame *Frame) *Sect {
	s := NewSect()
	s.Frame = frame
	s.Num = sect.Num
	s.Original = sect.Original
	s.Name = sect.Name
	s.Figs = make([]*Fig, len(sect.Figs))
	for i, f := range sect.Figs {
		s.Figs[i] = f.Snapshot(frame)
	}
	s.Exp = sect.Exp
	s.Exq = sect.Exq
	for i := 0; i < 3; i++ {
		s.Lload[i] = sect.Lload[i]
	}
	for i := 0; i < 12; i++ {
		s.Yield[i] = sect.Yield[i]
	}
	s.Type = sect.Type
	s.Color = sect.Color
	return s
}

func NewFig() *Fig {
	f := new(Fig)
	f.Num = 1
	f.Value = make(map[string]float64)
	return f
}

func (fig *Fig) Snapshot(frame *Frame) *Fig {
	f := NewFig()
	f.Num = fig.Num
	f.Prop = frame.Props[fig.Prop.Num]
	for k, v := range fig.Value {
		f.Value[k] = v
	}
	return f
}

func (fig *Fig) SetShapeProperty(s Shape) {
	fig.Value["AREA"] = s.A() * 0.0001
	fig.Value["IXX"] = s.Ix() * 1e-8
	fig.Value["IYY"] = s.Iy() * 1e-8
	fig.Value["VEN"] = s.J() * 1e-8
}

func (sect *Sect) Hide() {
	sect.Frame.Show.Sect[sect.Num] = false
}

func (sect *Sect) Show() {
	sect.Frame.Show.Sect[sect.Num] = true
}

func (sect *Sect) IsHidden(show *Show) bool {
	return !show.Sect[sect.Num]
}

func (sect *Sect) InpString() string {
	var rtn bytes.Buffer
	rtn.WriteString(fmt.Sprintf("SECT %3d SNAME %s\n", sect.Num, sect.Name))
	col := IntColor(sect.Color)
	if len(sect.Figs) < 1 {
		rtn.WriteString("         SROLE HOJO\n")
		rtn.WriteString(fmt.Sprintf("         COLOR %s\n", col))
		return rtn.String()
	}
	rtn.WriteString(fmt.Sprintf("         NFIG %d\n", len(sect.Figs)))
	for _, f := range sect.Figs {
		rtn.WriteString(f.InpString())
	}
	if _, ok := sect.Figs[0].Value["AREA"]; ok {
		rtn.WriteString(fmt.Sprintf("         EXP %5.3f\n", sect.Exp))
		if sect.Exq != sect.Exp {
			rtn.WriteString(fmt.Sprintf("         EXQ %5.3f\n", sect.Exq))
		}
		rtn.WriteString(fmt.Sprintf("         NZMAX %9.3f NZMIN %9.3f\n", sect.Yield[0], sect.Yield[1]))
		rtn.WriteString(fmt.Sprintf("         QXMAX %9.3f QXMIN %9.3f\n", sect.Yield[2], sect.Yield[3]))
		rtn.WriteString(fmt.Sprintf("         QYMAX %9.3f QYMIN %9.3f\n", sect.Yield[4], sect.Yield[5]))
		rtn.WriteString(fmt.Sprintf("         MZMAX %9.3f MZMIN %9.3f\n", sect.Yield[6], sect.Yield[7]))
		rtn.WriteString(fmt.Sprintf("         MXMAX %9.3f MXMIN %9.3f\n", sect.Yield[8], sect.Yield[9]))
		rtn.WriteString(fmt.Sprintf("         MYMAX %9.3f MYMIN %9.3f\n", sect.Yield[10], sect.Yield[11]))
	}
	if sect.Lload[0] != 0.0 || sect.Lload[1] != 0.0 || sect.Lload[2] != 0.0 {
		rtn.WriteString(fmt.Sprintf("         LLOAD %.3f %.3f %.3f\n", sect.Lload[0], sect.Lload[1], sect.Lload[2]))
	}
	rtn.WriteString(fmt.Sprintf("         COLOR %s\n", col))
	return rtn.String()
}

func (fig *Fig) InpString() string {
	var rtn bytes.Buffer
	rtn.WriteString(fmt.Sprintf("         FIG %3d FPROP %d\n", fig.Num, fig.Prop.Num))
	for _, k := range FIGKEYS {
		if val, ok := fig.Value[k]; ok {
			switch k {
			case "AREA":
				rtn.WriteString(fmt.Sprintf("                 AREA %7.4f\n", val))
			case "IXX", "IYY", "VEN":
				rtn.WriteString(fmt.Sprintf("                 %s  %11.8f\n", k, val))
			case "THICK":
				rtn.WriteString(fmt.Sprintf("                 THICK %7.5f\n", val))
				if val, ok = fig.Value["FC"]; ok {
					rtn.WriteString(fmt.Sprintf("                 SIGMA FC%.0f", val))
					if val, ok = fig.Value["SD"]; ok {
						rtn.WriteString(fmt.Sprintf(" SD%.0f", val))
					} else {
						rtn.WriteString(" SD295")
					}
					if val, ok = fig.Value["RD"]; ok {
						rtn.WriteString(fmt.Sprintf(" D%.0f", val))
					} else {
						rtn.WriteString(" D0")
					}
					if val, ok = fig.Value["RA"]; ok {
						rtn.WriteString(fmt.Sprintf(" %.3f", val))
					} else {
						rtn.WriteString(" 0.000")
					}
					if val, ok = fig.Value["PITCH"]; ok {
						rtn.WriteString(fmt.Sprintf(" @%.0f", val))
					} else {
						rtn.WriteString(" @0")
					}
					if val, ok = fig.Value["SINDOU"]; ok {
						rtn.WriteString(fmt.Sprintf(" %.0f\n", val))
					} else {
						rtn.WriteString(" 1\n")
					}
				}
			case "SREIN":
				rtn.WriteString(fmt.Sprintf("                 SREIN %8.6f\n", val))
			case "XFACE", "YFACE":
				rtn.WriteString(fmt.Sprintf("                 %s %.3f %.3f\n", k, val, fig.Value[fmt.Sprintf("%s_H", k)]))
			}
		}
	}
	return rtn.String()
}

func (sect *Sect) InlString() string {
	if len(sect.Figs) < 1 {
		return ""
	}
	var rtn bytes.Buffer
	rtn.WriteString(fmt.Sprintf("%5d ", sect.Num))
	rtn.WriteString(fmt.Sprintf("%11.5E %7.5f ", sect.Figs[0].Prop.E, sect.Figs[0].Prop.Poi))
	rtn.WriteString(fmt.Sprintf("%6.4f %10.8f %10.8f %10.8f", sect.Figs[0].Value["AREA"], sect.Figs[0].Value["IXX"], sect.Figs[0].Value["IYY"], sect.Figs[0].Value["VEN"]))
	for i := 0; i < 12; i++ {
		rtn.WriteString(fmt.Sprintf(" %9.3f", sect.Yield[i]))
	}
	rtn.WriteString(fmt.Sprintf(" %5d", sect.Type))
	rtn.WriteString(fmt.Sprintf(" %5d", sect.Original))
	rtn.WriteString("\n")
	return rtn.String()
}

func (sect *Sect) HasBrace() bool {
	if len(sect.Figs) < 1 {
		return false
	}
	if _, ok := sect.Figs[0].Value["THICK"]; ok {
		if sect.Figs[0].Prop.E != 0.0 {
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}

func (sect *Sect) HasArea(ind int) bool {
	if len(sect.Figs) < ind+1 {
		return false
	}
	if _, ok := sect.Figs[ind].Value["AREA"]; ok {
		return true
	} else {
		return false
	}
}

func (sect *Sect) HasThick(ind int) bool {
	if len(sect.Figs) < ind+1 {
		return false
	}
	if _, ok := sect.Figs[ind].Value["THICK"]; ok {
		return true
	} else {
		return false
	}
}

func (sect *Sect) Hiju(ind int) (float64, error) {
	if len(sect.Figs) < ind+1 {
		return 0.0, errors.New(fmt.Sprintf("Hiju: SECT %d has no Fig %d", sect.Num, ind))
	}
	return sect.Figs[ind].Prop.Hiju, nil
}

func (sect *Sect) Area(ind int) (float64, error) {
	if len(sect.Figs) < ind+1 {
		return 0.0, errors.New(fmt.Sprintf("Area: SECT %d has no Fig %d", sect.Num, ind))
	}
	if val, ok := sect.Figs[ind].Value["AREA"]; ok {
		return val, nil
	}
	return 0.0, errors.New(fmt.Sprintf("Area: SECT %d Fig %d doesn't have AREA", ind, sect.Num))
}

func (sect *Sect) Ix(ind int) (float64, error) {
	if len(sect.Figs) < ind+1 {
		return 0.0, errors.New(fmt.Sprintf("Ix: SECT %d has no Fig %d", sect.Num, ind))
	}
	if val, ok := sect.Figs[ind].Value["IXX"]; ok {
		return val, nil
	}
	return 0.0, errors.New(fmt.Sprintf("Ix: SECT %d Fig %d doesn't have IXX", ind, sect.Num))
}

func (sect *Sect) Iy(ind int) (float64, error) {
	if len(sect.Figs) < ind+1 {
		return 0.0, errors.New(fmt.Sprintf("Iy: SECT %d has no Fig %d", sect.Num, ind))
	}
	if val, ok := sect.Figs[ind].Value["IYY"]; ok {
		return val, nil
	}
	return 0.0, errors.New(fmt.Sprintf("Iy: SECT %d Fig %d doesn't have IYY", ind, sect.Num))
}

func (sect *Sect) Thick(ind int) (float64, error) {
	if len(sect.Figs) < ind+1 {
		return 0.0, errors.New(fmt.Sprintf("Thick: SECT %d has no Fig %d", sect.Num, ind))
	}
	if val, ok := sect.Figs[ind].Value["THICK"]; ok {
		return val, nil
	}
	return 0.0, errors.New(fmt.Sprintf("Thick: SECT %d Fig %d doesn't have THICK", ind, sect.Num))
}

func (sect *Sect) Srein(ind int) (float64, error) {
	if len(sect.Figs) < ind+1 {
		return 0.0, errors.New(fmt.Sprintf("Srein: SECT %d has no Fig %d", sect.Num, ind))
	}
	if val, ok := sect.Figs[ind].Value["SREIN"]; ok {
		return val, nil
	}
	t, err := sect.Thick(ind)
	if err != nil {
		return 0.0, err
	}
	if a, ok := sect.Figs[ind].Value["RA"]; ok {
		if p, ok := sect.Figs[ind].Value["PITCH"]; ok {
			if sd, ok := sect.Figs[ind].Value["SINDOU"]; ok {
				return a * sd / (p*t*10.0), nil
			}
		}
	}
	return 0.0, errors.New(fmt.Sprintf("Srein: SECT %d Fig %d doesn't have SREIN", ind, sect.Num))
}

func (sect *Sect) Xface(ind int) (float64, float64, error) {
	if len(sect.Figs) < ind+1 {
		return 0.0, 0.0, errors.New(fmt.Sprintf("Xface: SECT %d has no Fig %d", sect.Num, ind))
	}
	if val, ok := sect.Figs[ind].Value["XFACE"]; ok {
		return val, sect.Figs[ind].Value["XFACE_H"], nil
	}
	return 0.0, 0.0, errors.New(fmt.Sprintf("Xface: SECT %d Fig %d doesn't have XFACE", ind, sect.Num))
}

func (sect *Sect) PropSize(props []int) float64 {
	if len(sect.Figs) == 0 {
		return 0.0
	}
	sum := 0.0
	for _, fig := range sect.Figs {
		for _, num := range props {
			if fig.Prop.Num == num {
				if aval, ok := fig.Value["AREA"]; ok {
					sum += aval
				} else if tval, ok := fig.Value["THICK"]; ok {
					sum += tval
				}
				break
			}
		}
	}
	return sum
}

func (sect *Sect) PropWeight(props []int) float64 {
	if len(sect.Figs) == 0 {
		return 0.0
	}
	sum := 0.0
	for _, fig := range sect.Figs {
		for _, num := range props {
			if fig.Prop.Num == num {
				if aval, ok := fig.Value["AREA"]; ok {
					sum += aval * fig.Prop.Hiju
				} else if tval, ok := fig.Value["THICK"]; ok {
					sum += tval * fig.Prop.Hiju
				}
				break
			}
		}
	}
	return sum
}

func (sect *Sect) TotalAmount() float64 {
	sum := 0.0
	for _, el := range sect.Frame.Elems {
		if el.Sect == sect {
			sum += el.Amount()
		}
	}
	return sum
}

func (sect *Sect) Weight() []float64 {
	if len(sect.Figs) == 0 {
		return []float64{0.0, 0.0, 0.0}
	}
	rtn := make([]float64, 3)
	sum := 0.0
	for _, fig := range sect.Figs {
		if aval, ok := fig.Value["AREA"]; ok {
			sum += aval * fig.Prop.Hiju
		} else if tval, ok := fig.Value["THICK"]; ok {
			sum += tval * fig.Prop.Hiju
		}
	}
	for i := 0; i < 3; i++ {
		rtn[i] = sum + sect.Lload[i]
	}
	return rtn
}

func (sect *Sect) BraceSection() []*Sect {
	rtn := make([]*Sect, 0)
	enum := 0
	var add bool
	for _, el := range sect.Frame.Elems {
		if el.Etype == WBRACE || el.Etype == SBRACE {
			if el.OriginalSection() == sect {
				add = true
				for _, v := range rtn {
					if el.Sect == v {
						add = false
						break
					}
				}
				if add {
					rtn = append(rtn, el.Sect)
					enum++
				}
			}
		}
	}
	rtn = rtn[:enum]
	sort.Sort(SectByNum{rtn})
	return rtn
}

func (sect *Sect) IsRc(eps float64) bool {
	if sect.Figs == nil || len(sect.Figs) < 1 {
		return false
	}
	return sect.Figs[0].Prop.IsRc(eps)
}

func (sect *Sect) IsGohan(eps float64) bool {
	if sect.Figs == nil || len(sect.Figs) < 1 {
		return false
	}
	return sect.Figs[0].Prop.IsGohan(eps)
}
