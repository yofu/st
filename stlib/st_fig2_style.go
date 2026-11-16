package st

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type fig2Pen struct {
	color int
	width float64
	dash  []float64
	elems map[int]bool
	nodes map[int]bool
}

func fig2ReadPen(show *Show, args []string) (fig2Pen, error) {
	p := fig2Pen{width: 0.05, color: BLACK}
	if len(args) < 4 {
		return p, NotEnoughArgs("PEN")
	}
	v, err := fig2Numbers(args, 4)
	if err != nil {
		return p, err
	}
	p.width = v[0]
	if p.width < 0 {
		return p, fmt.Errorf("negative pen width")
	}
	for _, c := range v[1:] {
		if c < 0 || c > 255 || c != math.Trunc(c) {
			return p, fmt.Errorf("PEN RGB values must be integers from 0 to 255")
		}
	}
	p.color = int(v[1])<<16 | int(v[2])<<8 | int(v[3])
	if len(args) > 4 {
		key := strings.ToUpper(args[4])
		p.dash = show.Fig2Dots[key]
		if p.dash == nil {
			p.dash = map[string][]float64{"DOT": {0.5, 0.5}, "HIDDEN": {2, 1}, "DASH": {4, 1}, "DASHDOT": {4, 1, 1, 1}}[key]
		}
	}
	p.elems, p.nodes = make(map[int]bool), make(map[int]bool)
	return p, nil
}

func fig2ApplyPen(f *Frame, pen fig2Pen, args []string) error {
	if len(args) < 1 {
		return NotEnoughArgs("PEN selection")
	}
	key := strings.ToUpper(args[0])
	codes := make(map[string]bool)
	for _, arg := range args[1:] {
		codes[strings.ToUpper(arg)] = true
	}
	switch key {
	case "NODE", "NODELAYER", "VCON", "VCONPOS", "VCONNEG":
		axis := 0
		if key == "VCONPOS" || key == "VCONNEG" {
			if len(args) < 2 {
				return NotEnoughArgs(key)
			}
			i, err := strconv.Atoi(args[1])
			if err != nil || i < 0 || i > 5 {
				return fmt.Errorf("invalid %s component", key)
			}
			axis = i
		}
		for id, n := range f.Nodes {
			switch key {
			case "NODE":
				if len(codes) == 0 || codes[strconv.Itoa(id)] {
					pen.nodes[id] = true
				}
			case "VCON":
				if showFig2NodeCode(n, "PLOAD") {
					pen.nodes[id] = true
				}
			case "VCONPOS":
				if n.Load[axis] > 0 {
					pen.nodes[id] = true
				}
			case "VCONNEG":
				if n.Load[axis] < 0 {
					pen.nodes[id] = true
				}
			}
		}
		if key == "NODELAYER" {
			for _, name := range args[1:] {
				for _, n := range f.NodeSet[name] {
					if n != nil {
						pen.nodes[n.Num] = true
					}
				}
			}
		}
	case "ELEMLAYER":
		for _, name := range args[1:] {
			for _, e := range f.ElemSet[name] {
				if e != nil {
					pen.elems[e.Num] = true
				}
			}
		}
	case "FENCE":
		if len(args) < 3 {
			return NotEnoughArgs("PEN FENCE")
		}
		axis := strings.Index("XYZ", strings.ToUpper(args[1]))
		v, err := fig2Numbers(args[2:], 1)
		if err != nil || axis < 0 || len(args[1]) != 1 {
			return fmt.Errorf("invalid PEN FENCE")
		}
		for _, e := range f.Fence(axis, v[0], false) {
			pen.elems[e.Num] = true
		}
	case "ELEM", "ETYPE", "SECT":
		for id, e := range f.Elems {
			switch key {
			case "ELEM":
				if len(codes) == 0 || codes[strconv.Itoa(id)] {
					pen.elems[id] = true
				}
			case "ETYPE":
				if codes[ETYPES[e.Etype]] {
					pen.elems[id] = true
				}
			case "SECT":
				if codes[strconv.Itoa(e.OriginalSection().Num)] {
					pen.elems[id] = true
				}
			}
		}
	default:
		return fmt.Errorf("unsupported PEN selection %q", key)
	}
	return nil
}

func fig2SetMax(f *Frame, args []string) {
	excluded := make(map[int]bool)
	for _, arg := range args {
		if id, err := strconv.Atoi(arg); err == nil {
			excluded[id] = true
		}
	}
	type entry struct {
		id    int
		value float64
	}
	top := make(map[int]entry)
	for id, e := range f.Elems {
		sect := e.OriginalSection().Num
		if excluded[sect] || len(e.MaxRate) == 0 {
			continue
		}
		value := e.MaxRate[0]
		for _, r := range e.MaxRate[1:] {
			value = max(value, r)
		}
		if old, ok := top[sect]; !ok || value > old.value || (value == old.value && id < old.id) {
			top[sect] = entry{id, value}
		}
	}
	f.Show.Fig2Max = make(map[int]bool, len(top))
	for _, p := range top {
		f.Show.Fig2Max[p.id] = true
	}
}

func drawFig2Pen(stw Drawer, p fig2Pen) {
	stw.Foreground(p.color)
	if styler, ok := stw.(interface{ SetFig2Stroke(float64, []float64) }); ok {
		styler.SetFig2Stroke(p.width, p.dash)
	}
}

func fig2DispFactor(s *Show) float64 {
	if s.Fig2DispFactor != 0 {
		return s.Fig2DispFactor
	}
	return 100
}

func fig2RateBoundary(s *Show) []float64 {
	if s.Fig2Rainbow != nil {
		return s.Fig2Rainbow
	}
	return RateBoundary
}

type fig2Anchor struct {
	align int
	x, y  float64
}

func drawFig2Caption(stw Drawer, show *Show, kind string, x, y float64, text string) {
	if a, ok := show.Fig2Anchors[kind]; ok {
		stw.TextAlignment(a.align)
		defer stw.TextAlignment(show.DefaultTextAlignment)
		x += a.x
		if stw.CanvasDirection() == 1 {
			y -= a.y
		} else {
			y += a.y
		}
	}
	stw.Text(x, y, text)
}

func fig2SetTextSize(stw Drawer, size int) func() {
	if font, ok := stw.(interface{ SetFig2FontSize(int) }); ok && size > 0 {
		font.SetFig2FontSize(size)
		return func() { font.SetFig2FontSize(6) }
	}
	return func() {}
}

type fig2StressValue struct {
	period string
	side   uint8
}

func drawFig2Eccentric(stw Drawer, f *Frame) {
	facts := f.Fes
	for _, i := range f.Show.Fig2Eccentric {
		if i >= len(facts.AverageLevel) || i >= len(facts.CentreOfWeight) || i > len(facts.CentreOfRigid) {
			continue
		}
		weight, rigid := facts.CentreOfWeight[i], facts.CentreOfRigid[i-1]
		if len(weight) < 2 || len(rigid) < 2 || !finitePDF(weight[0], weight[1], rigid[0], rigid[1], facts.AverageLevel[i]) {
			continue
		}
		for j, p := range [][]float64{weight, rigid} {
			at := f.View.ProjectCoord([]float64{p[0], p[1], facts.AverageLevel[i]})
			stw.Foreground([]int{RED, BLUE}[j])
			stw.Circle(at[0], at[1], 2)
			drawFig2Caption(stw, f.Show, "ECCENTRIC", at[0]+1, at[1], []string{"重心", "剛心"}[j])
		}
	}
}
