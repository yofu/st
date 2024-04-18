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

type PGFCanvas struct {
	scale float64
	width float64
	height float64
	buffer bytes.Buffer
	linestyle map[string]string
	fillstyle map[string]string
	textstyle map[string]string
}

func NewPGFCanvas(width, height float64) *PGFCanvas {
	var otp bytes.Buffer
	return &PGFCanvas{
		scale: 0.01,
		width: width,
		height: height,
		buffer: otp,
		linestyle: make(map[string]string),
		fillstyle: make(map[string]string),
		textstyle: make(map[string]string),
	}
}

func (p *PGFCanvas) Draw(frame *Frame, textBox []*TextBox) {
	p.buffer.WriteString("\\documentclass[uplatex,a4paper,dvipdfmx,10pt]{jsarticle}\n\\usepackage{tikz}\n\\begin{document}\n\\begin{tikzpicture}\n")
	DrawFrame(p, frame, frame.Show.ColorMode, false)
	p.Foreground(BLACK)
	for _, t := range textBox {
		if !t.IsHidden(frame.Show) {
			DrawText(p, t)
		}
	}
	p.buffer.WriteString("\\end{tikzpicture}\n\\end{document}\n")
}

func (p *PGFCanvas) SaveAs(fn string) error {
	w, err := os.Create(fn)
	if err != nil {
		return err
	}
	defer w.Close()
	p.buffer.WriteTo(w)
	return nil
}

// TODO
func (p *PGFCanvas) linestylestring() string {
	return ""
}

// TODO
func (p *PGFCanvas) fillstylestring() string {
	return ""
}

// TODO
func (p *PGFCanvas) textstylestring() string {
	return ""
}

func (p *PGFCanvas) Line(x1, y1, x2, y2 float64) {
	p.buffer.WriteString(fmt.Sprintf("\\draw[%s] (%.3f,%.3f) -- (%.3f,%.3f);\n", p.linestylestring(), x1*p.scale, y1*p.scale, x2*p.scale, y2*p.scale))
}

func (p *PGFCanvas) Polyline(coord [][]float64) {
	if len(coord) < 2 {
		return
	}
	p.buffer.WriteString(fmt.Sprintf("\\draw[%s] (%.3f,%.3f)", p.linestylestring(), coord[0][0]*p.scale, coord[0][1]*p.scale))
	for i := 1; i < len(coord); i++ {
		p.buffer.WriteString(fmt.Sprintf(" -- (%.3f,%.3f)", coord[i][0]*p.scale, coord[i][1]*p.scale))
	}
	p.buffer.WriteString(";\n")
}

func (p *PGFCanvas) Polygon(coord [][]float64) {
	if len(coord) < 2 {
		return
	}
	p.buffer.WriteString(fmt.Sprintf("\\fill[%s] (%.3f,%.3f)", p.fillstylestring(), coord[0][0]*p.scale, coord[0][1]*p.scale))
	for i := 1; i < len(coord); i++ {
		p.buffer.WriteString(fmt.Sprintf(" -- (%.3f,%.3f)", coord[i][0]*p.scale, coord[i][1]*p.scale))
	}
	p.buffer.WriteString(";\n")
}

func (p *PGFCanvas) Circle(x, y, d float64) {
	p.buffer.WriteString(fmt.Sprintf("\\draw[%s] (%.3f,%.3f) circle (%.3f);\n", p.linestylestring(), x*p.scale, y*p.scale, 0.5*d*p.scale))
}


func (p *PGFCanvas) FilledCircle(x, y, d float64) {
	p.buffer.WriteString(fmt.Sprintf("\\fill[%s] (%.3f,%.3f) circle (%.3f);\n", p.fillstylestring(), x*p.scale, y*p.scale, 0.5*d*p.scale))
}


func (p *PGFCanvas) Text(x, y float64, str string) {
	p.buffer.WriteString(fmt.Sprintf("\\node[%s] at (%.3f,%.3f) {%s};\n", p.textstylestring(), x, y, str))
}

// TODO
func (p *PGFCanvas) Foreground(fg int) int {
	if fg == WHITE {
		fg = BLACK
	}
	// r, g, b := p.canvas.GetDrawColor()
	// lis := IntColorList(fg)
	p.linestyle["color"] = ""
	p.fillstyle["color"] = ""
	p.textstyle["color"] = ""
	return fg
}

func (p *PGFCanvas) LineStyle(ls int) {
	switch ls {
	case CONTINUOUS:
		p.linestyle["pattern"] = ""
	case DOTTED:
		p.linestyle["pattern"] = "dotted"
	case DASHED:
		p.linestyle["pattern"] = "dashed"
	case DASH_DOT:
		// TODO
		p.linestyle["pattern"] = "dashed"
	}
}

func (p *PGFCanvas) TextAlignment(ta int) {
	switch ta {
	case SOUTH:
		p.textstyle["align"] = "align=south"
	case NORTH:
		p.textstyle["align"] = "align=north"
	case WEST:
		p.textstyle["align"] = "align=west"
	case EAST:
		p.textstyle["align"] = "align=east"
	case CENTER:
		p.textstyle["align"] = "align=center"
	}
}

// TODO
func (p *PGFCanvas) TextOrientation(to float64) {
	if to == 0.0 {
		p.textstyle["orientation"] = ""
	} else {
		p.textstyle["orientation"] = fmt.Sprintf("rotate=%f", to)
	}
}

func (p *PGFCanvas) SectionAlias(s int) (string, bool) {
	return "", false
}

func (p *PGFCanvas) SelectedNodes() []*Node {
	return nil
}

func (p *PGFCanvas) SelectedElems() []*Elem {
	return nil
}

func (p *PGFCanvas) ElemSelected() bool {
	return false
}

func (p *PGFCanvas) DefaultStyle() {
	p.linestyle["color"] = ""
	p.fillstyle["color"] = ""
	p.textstyle["color"] = ""
	p.textstyle["anchor"] = ""
	p.textstyle["orientation"] = ""
}

func (p *PGFCanvas) BondStyle(show *Show) {
}

func (p *PGFCanvas) PhingeStyle(show *Show) {
}

func (p *PGFCanvas) ConfStyle(show *Show) {
}

func (p *PGFCanvas) SelectNodeStyle() {
}

func (p *PGFCanvas) SelectElemStyle() {
}

func (p *PGFCanvas) ShowPrintRange() bool {
	return false
}

func (p *PGFCanvas) GetCanvasSize() (int, int) {
	return int(p.width), int(p.height)
}

func (p *PGFCanvas) CanvasPaperSize() (float64, float64, error) {
	return p.width, p.height, nil
}

func (p *PGFCanvas) Flush() {
	p.buffer.WriteString("\\end{tikzpicture}")
}

func (p *PGFCanvas) CanvasDirection() int {
	return 1
}
