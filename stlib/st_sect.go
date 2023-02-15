package st

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/yofu/st/arclm"
)

var (
	FIGKEYS = []string{"AREA", "IXX", "IYY", "VEN", "THICK", "SREIN", "XFACE", "YFACE", "KFACT"}
	REINSKEYS = []string{"TREIN", "BREIN", "RREIN", "WREIN", "LREIN", "CREIN", "HOOP", "KABURI"}
)

type Sect struct {
	Frame    *Frame
	Num      int
	Name     string
	Figs     []*Fig
	Exp      float64
	Exq      float64
	Lload    []float64
	Perpl    []float64
	Yield    []float64
	Type     int
	Original int
	Color    int
	Allow    SectionRate
}

type Fig struct {
	Num   int
	Name  string
	Prop  *Prop
	Shape Shape
	Value map[string]float64
	Reins map[string][]string
}

type Sects []*Sect

func (s Sects) Len() int { return len(s) }
func (s Sects) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type SectByNum struct{ Sects }

func (s SectByNum) Less(i, j int) bool {
	return s.Sects[i].Num < s.Sects[j].Num
}

func NewSect() *Sect {
	s := new(Sect)
	s.Figs = make([]*Fig, 0)
	s.Exp = arclm.EXPONENT
	s.Exq = arclm.EXPONENT
	s.Lload = make([]float64, 3)
	s.Perpl = make([]float64, 3)
	s.Yield = make([]float64, 12)
	s.Color = 16777215
	s.Allow = nil
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
		s.Perpl[i] = sect.Perpl[i]
	}
	for i := 0; i < 12; i++ {
		s.Yield[i] = sect.Yield[i]
	}
	s.Type = sect.Type
	s.Color = sect.Color
	if s.Allow != nil {
		s.Allow = sect.Allow.Snapshot()
	}
	return s
}

func NewFig() *Fig {
	f := new(Fig)
	f.Num = 1
	f.Name = ""
	f.Value = make(map[string]float64)
	f.Reins = make(map[string][]string)
	return f
}

func (fig *Fig) Snapshot(frame *Frame) *Fig {
	f := NewFig()
	f.Num = fig.Num
	f.Name = fig.Name
	f.Prop = frame.Props[fig.Prop.Num]
	f.Shape = fig.Shape
	for k, v := range fig.Value {
		f.Value[k] = v
	}
	for k, v := range fig.Reins {
		f.Reins[k] = v
	}
	return f
}

func (fig *Fig) SetShapeProperty(s Shape) {
	fig.Shape = s
	fac := 1.0
	if fig.Prop.Material != nil {
		fac = fig.Prop.EFactor
		if fac == 0.0 {
			fac = 1.0
		}
	}
	fig.Value["AREA"] = s.A() * 0.0001 / fac
	fig.Value["IXX"] = s.Ix() * 1e-8 / fac
	fig.Value["IYY"] = s.Iy() * 1e-8 / fac
	fig.Value["VEN"] = s.J() * 1e-8 / fac
}

func (fig *Fig) GetSectionRate(num, etype int) SectionRate {
	if fig.Prop.Material == nil {
		return nil
	}
	if etype <=BRACE && fig.Shape == nil {
		return nil
	}
	switch v := fig.Prop.Material.(type) {
	case Steel:
		switch etype {
		case COLUMN:
			return NewSColumn(num, fig.Shape, v)
		case GIRDER:
			return NewSGirder(num, fig.Shape, v)
		case BRACE:
			return NewSBrace(num, fig.Shape, v)
		default:
			return nil
		}
	case Concrete:
		switch etype {
		case COLUMN:
			rc := NewRCColumn(num)
			pl := fig.Shape.(PLATE)
			rc.CShape = NewCRect([]float64{-0.5*pl.B, -0.5*pl.H, 0.5*pl.B, 0.5*pl.H})
			rc.Concrete = v
			err := rc.AutoLayoutReins(fig.Reins)
			if err != nil {
				return nil
			}
			return rc
		case GIRDER:
			rc := NewRCGirder(num)
			pl := fig.Shape.(PLATE)
			rc.CShape = NewCRect([]float64{-0.5*pl.B, -0.5*pl.H, 0.5*pl.B, 0.5*pl.H})
			rc.Concrete = v
			err := rc.AutoLayoutReins(fig.Reins)
			if err != nil {
				return nil
			}
			return rc
		case WALL:
			rc := NewRCWall(num)
			rc.Concrete = v
			if val, ok := fig.Value["THICK"]; ok {
				rc.Thick = val*100 // m -> cm
			}
			if val, ok := fig.Reins["WREIN"]; ok {
				rc.SetWrein(val)
			}
			return rc
		case SLAB:
			rc := NewRCSlab(num)
			rc.Concrete = v
			if val, ok := fig.Value["THICK"]; ok {
				rc.Thick = val*100 // m -> cm
			}
			if val, ok := fig.Reins["WREIN"]; ok {
				rc.SetWrein(val)
			}
			return rc
		default:
			return nil
		}
	case Wood:
		switch etype {
		case COLUMN:
			return NewWoodColumn(num, fig.Shape, v)
		case GIRDER:
			return NewWoodGirder(num, fig.Shape, v)
		case WALL:
			ww := NewWoodWall(num)
			ww.Wood = v
			if val, ok := fig.Value["KFACT"]; ok {
				ww.Kfact = val
			} else if val, ok := fig.Value["THICK"]; ok {
				ww.Thick = val*100*0.250/0.225 // m -> cm
			} else {
				return nil
			}
			return ww
		case SLAB:
			ww := NewWoodSlab(num)
			ww.Wood = v
			if val, ok := fig.Value["KFACT"]; ok {
				ww.Kfact = val
			} else if val, ok := fig.Value["THICK"]; ok {
				ww.Thick = val*100*0.250/0.225 // m -> cm
			} else {
				return nil
			}
			return ww
		default:
			return nil
		}
	default:
		return nil
	}
}

func (fig *Fig) Weight() float64 {
	if aval, ok := fig.Value["AREA"]; ok {
		return aval * fig.Prop.Hiju()
	} else if tval, ok := fig.Value["THICK"]; ok {
		return tval * fig.Prop.Hiju()
	} else {
		return 0.0
	}
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

func (sect *Sect) SplitName() (string, string) {
	sn := strings.Split(sect.Name, ":")
	if len(sn) < 2 {
		return "", sect.Name
	} else {
		return sn[0], sn[1]
	}
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
	if sect.Perpl[0] != 0.0 || sect.Perpl[1] != 0.0 || sect.Perpl[2] != 0.0 {
		rtn.WriteString(fmt.Sprintf("         PERPL %.3f %.3f %.3f\n", sect.Perpl[0], sect.Perpl[1], sect.Perpl[2]))
	}
	rtn.WriteString(fmt.Sprintf("         COLOR %s\n", col))
	return rtn.String()
}

func (fig *Fig) InpString() string {
	var rtn bytes.Buffer
	rtn.WriteString(fmt.Sprintf("         FIG %3d FPROP %d\n", fig.Num, fig.Prop.Num))
	if fig.Name != "" {
		rtn.WriteString(fmt.Sprintf("                 FNAME %s\n", fig.Name))
	}
	if fig.Shape != nil {
		rtn.WriteString(fmt.Sprintf("                 SHAPE %s\n", fig.Shape.String()))
	}
	for _, k := range REINSKEYS {
		if val, ok := fig.Reins[k]; ok {
			rtn.WriteString(fmt.Sprintf("                 %s %s\n", k, strings.Join(val, " ")))
		}
	}
	for _, k := range FIGKEYS {
		if val, ok := fig.Value[k]; ok {
			switch k {
			case "AREA":
				if val < 1e-4 {
					rtn.WriteString(fmt.Sprintf("                 AREA %7.4E\n", val))
				} else {
					rtn.WriteString(fmt.Sprintf("                 AREA %7.4f\n", val))
				}
			case "IXX", "IYY", "VEN":
				if val < 1e-8 {
					rtn.WriteString(fmt.Sprintf("                 %s  %11.8E\n", k, val))
				} else {
					rtn.WriteString(fmt.Sprintf("                 %s  %11.8f\n", k, val))
				}
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
			case "KFACT":
				rtn.WriteString(fmt.Sprintf("                 KFACT %.1f\n", val))
			case "SREIN":
				rtn.WriteString(fmt.Sprintf("                 SREIN %8.6f\n", val))
			case "XFACE", "YFACE":
				rtn.WriteString(fmt.Sprintf("                 %s %.3f %.3f\n", k, val, fig.Value[fmt.Sprintf("%s_H", k)]))
			}
		}
	}
	return rtn.String()
}

func (sect *Sect) ArclmValue() []float64 {
	return []float64{
		sect.Figs[0].Value["AREA"],
		sect.Figs[0].Value["IXX"],
		sect.Figs[0].Value["IYY"],
		sect.Figs[0].Value["VEN"],
	}
}

func (sect *Sect) HasBrace() bool {
	if len(sect.Figs) < 1 {
		return false
	}
	if _, ok := sect.Figs[0].Value["THICK"]; ok {
		if sect.Figs[0].Prop.ES() != 0.0 {
			return true
		} else {
			return false
		}
	} else if val, ok := sect.Figs[0].Value["KFACT"]; ok {
		if val > 0.0 {
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
	} else if _, ok := sect.Figs[ind].Value["KFACT"]; ok {
		return true
	} else {
		return false
	}
}

func (sect *Sect) Hiju(ind int) (float64, error) {
	if len(sect.Figs) < ind+1 {
		return 0.0, errors.New(fmt.Sprintf("Hiju: SECT %d has no Fig %d", sect.Num, ind))
	}
	return sect.Figs[ind].Prop.Hiju(), nil
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

func (sect *Sect) J(ind int) (float64, error) {
	if len(sect.Figs) < ind+1 {
		return 0.0, errors.New(fmt.Sprintf("J: SECT %d has no Fig %d", sect.Num, ind))
	}
	if val, ok := sect.Figs[ind].Value["VEN"]; ok {
		return val, nil
	}
	return 0.0, errors.New(fmt.Sprintf("J: SECT %d Fig %d doesn't have VEN", ind, sect.Num))
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

func (sect *Sect) Kfact(ind int) (float64, error) {
	if len(sect.Figs) < ind+1 {
		return 0.0, errors.New(fmt.Sprintf("Kfact: SECT %d has no Fig %d", sect.Num, ind))
	}
	if val, ok := sect.Figs[ind].Value["KFACT"]; ok {
		return val, nil
	}
	return 0.0, errors.New(fmt.Sprintf("Kfact: SECT %d Fig %d doesn't have KFACT", ind, sect.Num))
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
				return a * sd / (p * t * 10.0), nil
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
				sum += fig.Weight()
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
		sum += fig.Weight()
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

func (sect *Sect) IsReaction() bool {
	if sect.Lload == nil || len(sect.Lload) < 3 {
		return false
	}
	if sect.Lload[1] < 0.0 {
		return true
	} else {
		return false
	}
}
