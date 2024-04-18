package st

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"
)

func DrawClosedLinePGF(vertices [][]float64) string {
	var otp bytes.Buffer
	otp.WriteString("\\begin{tikzpicture}\n")
	num := 0
	for _, v := range vertices {
		if v == nil {
			if num > 0 {
				otp.WriteString("-- cycle;\n")
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
		otp.WriteString("-- cycle;\n\\end{tikzpicture}\n")
	}
	return otp.String()
}

func ConvertNumberToAlphabet(val int) string {
	rtn := ""
	alpha := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	for _, s := range fmt.Sprintf("%d", val) {
		rtn += alpha[s-48]
	}
	return rtn
}

func SectionList(fn string, frame *Frame, scale float64) error {
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
		rtn.WriteString(fmt.Sprintf("\\begin{myverbbox}[\\footnotesize]{\\s%s}\n", ConvertNumberToAlphabet(al.Num())))
		for _, s := range strings.Split(al.String(), "\n") {
			if s != "" {
				rtn.WriteString(fmt.Sprintf("%s\n", s))
			}
		}
		rtn.WriteString("\\end{myverbbox}\n")
	}
	nx := 1
	ny := 4
	sizex := 4.0
	sizey := 5.5
	cx := 0.0
	cy := 0.0
	indx := 0
	indy := 0
	rtn.WriteString("\\begin{tikzpicture}\n")
	for _, sec := range sectlist {
		cx = sizex*(float64(indx*2)+0.5)
		cy = sizey*(float64(ny-1-indy)+0.5)
		rtn.WriteString(fmt.Sprintf("\\draw (%.3f,%.3f) -- ++(%.3f,0) -- ++(0,%.3f) -- ++(%.3f,0) --cycle;\n", cx-0.5*sizex, cy-0.5*sizey, sizex, sizey, -sizex))
		rtn.WriteString(fmt.Sprintf("\\draw[->] (%.3f,%.3f) -- +(%.3f,0) node [right] {$x$};\n\\draw[->] (%.3f,%.3f) -- ++(0,%.3f) node [right] {$y$};\n", cx, cy, 0.2*sizex, cx, cy, 0.2*sizey))
		al := sec.Allow
		switch al.(type) {
		case *SColumn:
			sh := al.(*SColumn).Shape
			switch sh.(type) {
			case HKYOU, HWEAK, CROSS, RPIPE, CPIPE, PLATE, TKYOU, CKYOU, CWEAK, ANGLE:
				str := sh.PgfString(cx, cy, scale)
				rtn.WriteString(str)
			}
		case *SGirder:
			sh := al.(*SGirder).Shape
			switch sh.(type) {
			case HKYOU, HWEAK, CROSS, RPIPE, CPIPE, PLATE, TKYOU, CKYOU, CWEAK, ANGLE:
				str := sh.PgfString(cx, cy, scale)
				rtn.WriteString(str)
			}
		case *RCColumn:
			str := al.(*RCColumn).PgfString(cx, cy, scale)
			rtn.WriteString(str)
		case *RCGirder:
			str := al.(*RCGirder).PgfString(cx, cy, scale)
			rtn.WriteString(str)
		case *WoodColumn:
			sh := al.(*WoodColumn).Shape
			switch sh.(type) {
			case PLATE:
				str := sh.PgfString(cx, cy, scale)
				rtn.WriteString(str)
			}
		}
		rtn.WriteString(fmt.Sprintf("\\node[anchor=north west, align=left] at (%.3f, %.3f) {\\s%s};\n", cx+0.5*sizex, cy+0.5*sizey, ConvertNumberToAlphabet(al.Num())))
		indy++
		if indy == ny {
			indy = 0
			indx++
			if indx == nx {
				rtn.WriteString("\\end{tikzpicture}\n\\newpage\n\\begin{tikzpicture}\n")
				indx = 0
			}
		}
	}
	rtn.WriteString("\\end{tikzpicture}\n")
	w, err := os.Create(fn)
	defer w.Close()
	if err != nil {
		return err
	}
	rtn = AddCR(rtn)
	rtn.WriteTo(w)
	return nil
}
