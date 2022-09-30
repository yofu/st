package st

import (
	"bytes"
	"fmt"
	"os"
	"sort"
)

func DrawClosedLinePGF(vertices [][]float64) string {
	var otp bytes.Buffer
	otp.WriteString("\\begin{tikzpicture}\\n")
	num := 0
	for _, v := range vertices {
		if v == nil {
			if num > 0 {
				otp.WriteString("\\n")
				num = 0
			}
			continue
		}
		switch num {
		case 0:
			otp.WriteString(fmt.Sprintf("  \\draw (%.3f,%.3f)", v[0], v[1]))
		default:
			otp.WriteString(fmt.Sprintf("--(%.3f,%.3f)", v[0], v[1]))
		}
		num++
	}
	if num > 0 {
		otp.WriteString("\\n\\end{tikzpicture}\\n")
	}
	return otp.String()
}

func SectionList(frame *Frame) error {
	var rtn bytes.Buffer
	sects := make(map[int]int, 0)
	for _, el := range frame.Elems {
		snum := el.Sect.Num
		if _, ok := sects[snum]; ok {
			sects[snum]++
		} else {
			sects[snum] = 1
		}
	}
	sectlist := make([]*Sect, len(sects))
	ind := 0
	for k := range sects {
		if frame.Sects[k].Allow != nil {
			sectlist[ind] = frame.Sects[k]
			ind++
		}
	}
	sectlist = sectlist[:ind]
	sort.Sort(SectByNum{sectlist})
	for _, sec := range sectlist {
		al := sec.Allow
		switch al.(type) {
		case *SColumn:
			sh := al.(*SColumn).Shape
			switch sh.(type) {
			case HKYOU, HWEAK, CROSS, RPIPE, CPIPE, PLATE, TKYOU, CKYOU, CWEAK, ANGLE:
				vertices := sh.Vertices()
				str := DrawClosedLinePGF(vertices)
				rtn.WriteString(str)
			}
		case *RCColumn:
			rc := al.(*RCColumn)
			vertices := rc.CShape.Vertices()
			str := DrawClosedLinePGF(vertices)
			rtn.WriteString(str)
			for _, reins := range rc.Reins {
				vertices = reins.Vertices()
				str := DrawClosedLinePGF(vertices)
				rtn.WriteString(str)
			}
		case *RCGirder:
			rg := al.(*RCGirder)
			vertices := rg.CShape.Vertices()
			str := DrawClosedLinePGF(vertices)
			rtn.WriteString(str)
			for _, reins := range rg.Reins {
				vertices = reins.Vertices()
				str := DrawClosedLinePGF(vertices)
				rtn.WriteString(str)
			}
		case *WoodColumn:
			sh := al.(*WoodColumn).Shape
			switch sh.(type) {
			case PLATE:
				vertices := sh.Vertices()
				str := DrawClosedLinePGF(vertices)
				rtn.WriteString(str)
			}
		}
	}
	w, err := os.Create("sectionlist.tex")
	defer w.Close()
	if err != nil {
		return err
	}
	rtn = AddCR(rtn)
	rtn.WriteTo(w)
	return nil
}
