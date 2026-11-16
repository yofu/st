package st

import (
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type fig2NodalValue struct {
	axis                    int
	period                  string
	weight, force, pressure bool
	scale                   float64
}

func resetFig2View(stw Fig2Moder) {
	f := stw.Frame()
	for name := range f.Show.Fig2Boxes {
		b := stw.TextBox(name)
		b.Clear()
		b.Hide()
	}
	f.Show = NewShow(f)
	f.Show.ColorMode = stw.DefaultColorMode()
	f.Show.Bond, f.Show.Phinge, f.Show.Conf, f.Show.GlobalAxis = false, false, false, false
	f.Show.Fig2Document = true
	f.Show.Fig2Font = map[string]int{"NODE": 6, "ELEM": 6, "MEASURE": 6, "ECCENTRIC": 6}
	f.Show.KijunSize = 1.7
	f.Show.BondSize = 0.8
	f.Show.ConfSize = 1.5
	f.Show.StressTextColor, f.Show.KijunColor, f.Show.MeasureColor = BLACK, BLACK, BLACK
	f.View = NewView()
	stw.Deselect()
	ShowAll(stw)
	f.SetFocus(nil)
	w, h := stw.GetCanvasSize()
	f.View.Center[0], f.View.Center[1] = float64(w)*0.5, float64(h)*0.5
	for _, name := range []string{"PAGETITLE", "TITLE", "TEXT", "DATA", "INPUTFILE", "OUTPUTFILE", "VIEWPOINT"} {
		b := stw.TextBox(name)
		if b == nil {
			continue
		}
		b.Clear()
		b.Hide()
	}
}

func readFig2Data(stw Fig2Moder, args []string, dir string) error {
	if len(args) < 3 {
		return NotEnoughArgs("READ")
	}
	f := stw.Frame()
	filename := strings.ReplaceAll(strings.Join(args[2:], " "), "%", PruneExt(f.Path))
	if !filepath.IsAbs(filename) {
		path := filepath.Join(dir, filename)
		if !FileExists(path) {
			path = filepath.Join(filepath.Dir(f.Path), filename)
		}
		filename = path
	}
	ext := strings.ToLower(filepath.Ext(filename))
	switch strings.ToUpper(args[1]) {
	case "INPUT", "DATA":
		return f.ReadData(filename)
	case "ARCLM", "OUTPUT", "RESULT":
		if ext == ".otx" || ext == ".oty" || ext == ".inc" {
			if err := f.ReadResult(filename, UpdateResult); err != nil {
				return err
			}
			period, base := PeriodName(filename), strings.ToUpper(ext[1:])
			for _, n := range f.Nodes {
				if v, ok := n.Disp[period]; ok {
					n.Disp[base] = v
				}
				if v, ok := n.Reaction[period]; ok {
					n.Reaction[base] = v
				}
			}
			for _, e := range f.Elems {
				if v, ok := e.Stress[period]; ok {
					e.Stress[base] = v
				}
			}
			return nil
		}
		if ext == "" {
			return fmt.Errorf("missing result file extension")
		}
		return f.ReadResult(filename, UpdateResult)
	case "BCLNG":
		return f.ReadBuckling(filename)
	case "VIEIG":
		return f.ReadVibrationalEigenmode(filename)
	case "ZOUBUN", "OUTOU":
		return f.ReadZoubun(filename)
	case "SRCANJOINT", "RATJ":
		return readFig2JointRates(f, filename)
	case "SRCAN", "RAT", "RATE":
		return f.ReadRat(filename)
	case "LST":
		return f.ReadLst(filename)
	case "WGT", "WEIGHT":
		return f.ReadWgt(filename)
	case "KIJUN", "KJN":
		return f.ReadKjn(filename)
	default:
		return fmt.Errorf("unknown Fig2 READ method %q", args[1])
	}
}

func fig2Numbers(args []string, n int) ([]float64, error) {
	if len(args) < n {
		return nil, NotEnoughArgs("Fig2")
	}
	out := make([]float64, n)
	for i, arg := range args[:n] {
		v, err := strconv.ParseFloat(arg, 64)
		if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, fmt.Errorf("invalid Fig2 number %q", arg)
		}
		out[i] = v
	}
	return out, nil
}

func fig2NodeFilter(f *Frame, add bool, accepts func(*Node) bool) {
	if f.Show.Fig2Nodes == nil && add {
		return
	}
	if f.Show.Fig2Nodes == nil {
		f.Show.Fig2Nodes = make(map[int]bool, len(f.Nodes))
		for id := range f.Nodes {
			f.Show.Fig2Nodes[id] = true
		}
	}
	for id, n := range f.Nodes {
		match := accepts(n)
		if add {
			f.Show.Fig2Nodes[id] = f.Show.Fig2Nodes[id] || match
		} else {
			f.Show.Fig2Nodes[id] = f.Show.Fig2Nodes[id] && match
		}
	}
}

func fig2ElemFilter(f *Frame, op byte, accepts func(*Elem) bool) {
	if f.Show.Fig2Elems == nil && op == '+' {
		return
	}
	if f.Show.Fig2Elems == nil {
		f.Show.Fig2Elems = make(map[int]bool, len(f.Elems))
		for id := range f.Elems {
			f.Show.Fig2Elems[id] = true
		}
	}
	for id, e := range f.Elems {
		match := accepts(e)
		switch op {
		case '+':
			f.Show.Fig2Elems[id] = f.Show.Fig2Elems[id] || match
		case '-':
			f.Show.Fig2Elems[id] = f.Show.Fig2Elems[id] && !match
		default:
			f.Show.Fig2Elems[id] = f.Show.Fig2Elems[id] && match
		}
	}
}

func fig2Diff(a, b []float64) []float64 { return []float64{a[0] - b[0], a[1] - b[1], a[2] - b[2]} }

func fig2Plane(f *Frame, args []string) ([]float64, []float64, error) {
	if len(args) < 3 {
		return nil, nil, NotEnoughArgs("PLANE")
	}
	var nodes [3]*Node
	for i, arg := range args[:3] {
		id, err := strconv.Atoi(arg)
		if err != nil || f.Nodes[id] == nil {
			return nil, nil, fmt.Errorf("unknown node %q", arg)
		}
		nodes[i] = f.Nodes[id]
	}
	origin := nodes[0].Coord
	n := Cross(fig2Diff(nodes[1].Coord, origin), fig2Diff(nodes[2].Coord, origin))
	if Dot(n, n, 3) == 0 {
		return nil, nil, fmt.Errorf("collinear plane nodes")
	}
	return origin, Normalize(n), nil
}

func fig2FloorIndex(f *Frame, arg string) (int, error) {
	switch strings.ToUpper(arg) {
	case "G":
		return 1, nil
	case "N", "R":
		return len(f.Ai.Boundary) - 1, nil
	}
	return strconv.Atoi(arg)
}

func fig2NodalSpec(args []string, force bool) (fig2NodalValue, error) {
	var value fig2NodalValue
	if len(args) < 1 {
		return value, NotEnoughArgs("REACTION/FORCE")
	}
	value.axis = -1
	for i, key := range [...]string{"X", "Y", "Z", "RX", "RY", "RZ"} {
		if strings.EqualFold(key, args[0]) {
			value.axis = i
			break
		}
	}
	if value.axis < 0 {
		return value, fmt.Errorf("unknown nodal component %s", args[0])
	}
	value.force, value.scale = force, 1
	terms := args[1:]
	if len(terms) > 0 {
		if scale, err := strconv.ParseFloat(terms[len(terms)-1], 64); err == nil {
			if math.IsNaN(scale) || math.IsInf(scale, 0) {
				return value, fmt.Errorf("invalid nodal scale")
			}
			value.scale, terms = scale, terms[:len(terms)-1]
		}
	}
	for _, t := range terms {
		t = strings.ToUpper(t)
		if t == "W" {
			value.weight = true
			continue
		}
		if value.period != "" && !strings.HasPrefix(t, "-") && !strings.HasPrefix(t, "+") {
			value.period += "+"
		}
		value.period += t
	}
	if value.period == "" && !value.weight {
		value.period = "L"
	}
	return value, nil
}

func extraFig2Keyword(stw Fig2Moder, key string, args []string, un bool) (bool, error) {
	f := stw.Frame()
	s := f.Show
	switch key {
	case "eval", "rf", "ef", "ev", "nf", "nv":
		return true, applyFig2Expression(f, key, args, un)
	case "welding":
		if un {
			s.fig2WeldValues = nil
			return true, nil
		}
		return true, fig2Welding(f, args)
	case "srcanjoint":
		return true, setupFig2Joint(f, args, un)
	case "xmeasure", "ymeasure", "zmeasure", "measure":
		if key == "measure" && (!s.Fig2Document || len(args) == 0 || strings.EqualFold(args[0], "kijun")) {
			return false, nil
		}
		if un {
			s.Measure = false
			break
		}
		if err := fig2Measure(f, key, args); err != nil {
			return true, err
		}
	case "textbox":
		if len(args) == 0 {
			return true, NotEnoughArgs(key)
		}
		name := strings.ToUpper(args[0])
		b := stw.TextBox(name)
		b.Clear()
		if s.Fig2Boxes == nil {
			s.Fig2Boxes = make(map[string]bool)
		}
		s.Fig2Boxes[name] = true
		if len(args) > 1 {
			pos, err := fig2Numbers(args[1:], 2)
			if err != nil {
				return true, err
			}
			b.SetPosition(pos[0], pos[1])
		}
	case "font":
		if len(args) < 2 {
			return true, NotEnoughArgs(key)
		}
		size, err := strconv.Atoi(args[1])
		if err != nil || size < 1 || size > 100 {
			return true, fmt.Errorf("invalid FONT size")
		}
		name := strings.ToUpper(args[0])
		if s.Fig2Font == nil {
			s.Fig2Font = make(map[string]int)
		}
		s.Fig2Font[name] = size
		switch name {
		case "NODE", "ELEM", "MEASURE":
		default:
			stw.TextBox(name).Font.SetSize(size)
		}
	case "label":
		if len(args) < 2 {
			return true, NotEnoughArgs(key)
		}
		name := strings.ToUpper(args[0])
		b := stw.TextBox(name)
		if s.Fig2Boxes == nil {
			s.Fig2Boxes = make(map[string]bool)
		}
		if s.Fig2Boxes[name] {
			label := stw.TextBox(name + "_LABEL")
			x, y := b.Position()
			label.SetPosition(x-5, y+5)
			label.SetText([]string{strings.Join(args[1:], " ")})
			label.Show()
			s.Fig2Boxes[name+"_LABEL"] = true
		}
	case "anchor":
		if len(args) < 4 {
			return true, NotEnoughArgs(key)
		}
		pos, err := fig2Numbers(args[2:], 2)
		if err != nil {
			return true, err
		}
		align := CENTER
		switch strings.ToUpper(args[1]) {
		case "N":
			align = NORTH
		case "S":
			align = SOUTH
		case "W":
			align = WEST
		case "E":
			align = EAST
		case "SW":
			align = SOUTH_WEST
		case "SE":
			align = SOUTH_EAST
		case "C":
		default:
			return true, fmt.Errorf("unknown ANCHOR alignment %s", args[1])
		}
		if s.Fig2Anchors == nil {
			s.Fig2Anchors = make(map[string]fig2Anchor)
		}
		s.Fig2Anchors[strings.ToUpper(args[0])] = fig2Anchor{align: align, x: pos[0], y: pos[1]}
	case "position":
		if len(args) < 3 {
			return false, nil
		}
		name := strings.ToUpper(args[0])
		if (strings.EqualFold(args[1], "node") || strings.EqualFold(args[1], "elem")) && s.Fig2Document {
			if s.Fig2TextAnchors == nil {
				s.Fig2TextAnchors = make(map[string][]string)
			}
			s.Fig2TextAnchors[name] = args[1:3]
			break
		}
		if !s.Fig2Boxes[name] && !s.Fig2Document {
			return false, nil
		}
		if name == "LEGEND" || name == "DATA" {
			return false, nil
		}
		b := stw.TextBox(name)
		x, y := b.Position()
		for i, target := range []*float64{&x, &y} {
			if args[i+1] == "_" {
				continue
			}
			v, err := fig2Numbers(args[i+1:], 1)
			if err != nil {
				return true, err
			}
			*target = v[0]
		}
		b.SetPosition(x, y)
	case "read", "read!":
		return true, readFig2Data(stw, append([]string{key}, args...), stw.Cwd())
	case "height", "story", "storey", "floor":
		if key == "height" && len(args) == 0 {
			return false, nil
		}
		if len(args) == 0 {
			return true, NotEnoughArgs(key)
		}
		a, err := fig2FloorIndex(f, args[0])
		if err != nil {
			return true, err
		}
		b := a
		if key == "height" {
			if len(args) < 2 {
				return true, NotEnoughArgs(key)
			}
			b, err = fig2FloorIndex(f, args[1])
			if err != nil {
				return true, err
			}
		} else {
			a--
			if key != "floor" {
				b++
			}
		}
		if a < 0 || b < a || b >= len(f.Ai.Boundary) {
			return true, fmt.Errorf("Fig2 %s: floor range %d–%d unavailable", key, a, b)
		}
		fig2NodeFilter(f, false, func(n *Node) bool { return f.Ai.Boundary[a] <= n.Coord[2] && n.Coord[2] <= f.Ai.Boundary[b] })
	case "x", "y", "z", "x-", "y-", "z-":
		if (len(key) == 1 && !s.Fig2Document) || len(args) < 2 {
			return false, nil
		}
		if len(args)%2 != 0 {
			return true, fmt.Errorf("%s requires pairs of bounds", key)
		}
		bounds, err := fig2Numbers(args, len(args))
		if err != nil {
			return true, err
		}
		axis := strings.IndexByte("xyz", key[0])
		fig2NodeFilter(f, false, func(n *Node) bool {
			match := false
			for i := 0; i < len(bounds); i += 2 {
				match = match || (bounds[i] <= n.Coord[axis] && n.Coord[axis] <= bounds[i+1])
			}
			return match == (len(key) == 1 && !un)
		})
	case "plane", "plane+":
		if len(args) < 5 {
			return true, NotEnoughArgs(key)
		}
		o, n, err := fig2Plane(f, args)
		if err != nil {
			return true, err
		}
		v, err := fig2Numbers(args[3:], 2)
		if err != nil {
			return true, err
		}
		fig2NodeFilter(f, key == "plane+", func(p *Node) bool { d := Dot(n, fig2Diff(p.Coord, o), 3); return v[0] <= d && d <= v[1] })
	case "circle":
		if len(args) < 4 {
			return true, NotEnoughArgs(key)
		}
		v, err := fig2Numbers(args, 3)
		if err != nil {
			return true, err
		}
		mode := strings.ToUpper(args[3])
		var tol float64
		if mode != "IN" && mode != "OUT" {
			ns, err := fig2Numbers(args[3:], 1)
			if err != nil {
				return true, err
			}
			tol = ns[0] * ns[0]
		}
		fig2NodeFilter(f, false, func(p *Node) bool {
			x, y := p.Coord[0]-v[0], p.Coord[1]-v[1]
			d := x*x + y*y - v[2]*v[2]
			switch mode {
			case "IN":
				return d <= 0
			case "OUT":
				return d >= 0
			default:
				return math.Abs(d) <= tol
			}
		})
	case "fence", "fence+", "fence-":
		if key == "fence" && !s.Fig2Document && len(args) == 2 {
			return false, nil
		}
		if len(args) < 2 {
			return true, NotEnoughArgs(key)
		}
		plate := strings.EqualFold(args[len(args)-1], "p")
		var distance func(*Node) float64
		dir := strings.ToUpper(args[0])
		switch dir {
		case "X", "Y", "Z", "HEIGHT":
			axis := strings.Index("XYZ", dir)
			var x float64
			if dir == "HEIGHT" {
				i, err := fig2FloorIndex(f, args[1])
				if err != nil || i < 0 || i >= len(f.Ai.Boundary) {
					return true, fmt.Errorf("invalid fence height %s", args[1])
				}
				axis, x = 2, f.Ai.Boundary[i]
			} else if v, err := fig2Numbers(args[1:], 1); err == nil {
				x = v[0]
			} else if k := f.Kijuns[strings.ToLower(args[1])]; k != nil {
				v, err := fig2Numbers(args[2:], 1)
				if err != nil {
					return true, err
				}
				x = k.Start[axis] + v[0]
			} else {
				return true, fmt.Errorf("invalid fence coordinate %s", args[1])
			}
			distance = func(n *Node) float64 { return n.Coord[axis] - x }
		default:
			if k := f.Kijuns[strings.ToLower(dir)]; k != nil {
				v, err := fig2Numbers(args[1:], 1)
				if err != nil {
					return true, err
				}
				d := k.Direction()
				distance = func(n *Node) float64 { return (n.Coord[0]-k.Start[0])*d[1] - (n.Coord[1]-k.Start[1])*d[0] - v[0] }
			} else {
				o, n, err := fig2Plane(f, args)
				if err != nil {
					return true, err
				}
				v, err := fig2Numbers(args[3:], 1)
				if err != nil {
					return true, err
				}
				distance = func(p *Node) float64 { return Dot(n, fig2Diff(p.Coord, o), 3) - v[0] }
			}
		}
		fig2ElemFilter(f, key[len(key)-1], func(e *Elem) bool {
			if !plate && !e.IsLineElem() {
				return false
			}
			positive, negative := false, false
			for _, n := range e.Enod {
				if distance(n) > 0 {
					positive = true
				} else {
					negative = true
				}
			}
			return positive && negative
		})
	case "on":
		if len(args) < 1 {
			return true, NotEnoughArgs(key)
		}
		axis := -1
		switch strings.ToUpper(args[0]) {
		case "YZ", "ZY":
			axis = 0
		case "ZX", "XZ":
			axis = 1
		case "XY", "YX":
			axis = 2
		}
		if axis < 0 {
			return true, fmt.Errorf("unknown plane %q", args[0])
		}
		fig2ElemFilter(f, 0, func(e *Elem) bool {
			for _, n := range e.Enod[1:] {
				if math.Abs(n.Coord[axis]-e.Enod[0].Coord[axis]) > stw.EPS() {
					return false
				}
			}
			return true
		})
	case "rate":
		v, err := fig2Numbers(args, 2)
		if err != nil {
			return true, err
		}
		fig2ElemFilter(f, 0, func(e *Elem) bool {
			if len(e.MaxRate) == 0 {
				return false
			}
			top := e.MaxRate[0]
			for _, r := range e.MaxRate[1:] {
				top = max(top, r)
			}
			return v[0] <= top && top <= v[1]
		})
	case "elem", "elem+", "elem-":
		ids, types := make(map[int]bool), make(map[int]bool)
		for _, arg := range args {
			for _, part := range strings.Split(arg, ",") {
				if id, err := strconv.Atoi(part); err == nil {
					ids[id] = true
					continue
				}
				for i, t := range ETYPES {
					if strings.EqualFold(t, part) {
						types[i] = true
					}
				}
			}
		}
		if len(ids) == 0 && s.Fig2Elems == nil && !s.Fig2Document {
			return false, nil
		}
		for id := range ids {
			if e := f.Elems[id]; e != nil {
				s.Etype[e.Etype] = true
			}
		}
		for typ := range types {
			s.Etype[typ] = true
		}
		op := key[len(key)-1]
		fig2ElemFilter(f, op, func(e *Elem) bool { return ids[e.Num] || types[e.Etype] })
	case "nodelayer", "elemlayer":
		if len(args) == 0 {
			return true, NotEnoughArgs(key)
		}
		ids := make(map[int]bool)
		found := false
		for _, name := range args {
			if key == "nodelayer" {
				nodes, ok := f.NodeSet[name]
				found = found || ok
				for _, n := range nodes {
					if n != nil {
						ids[n.Num] = true
					}
				}
			} else {
				elems, ok := f.ElemSet[name]
				found = found || ok
				for _, e := range elems {
					if e != nil {
						ids[e.Num] = true
					}
				}
			}
		}
		if !found {
			return true, fmt.Errorf("unknown %s %q", key, strings.Join(args, ", "))
		}
		if key == "nodelayer" {
			fig2NodeFilter(f, false, func(n *Node) bool { return ids[n.Num] })
		} else {
			fig2ElemFilter(f, 0, func(e *Elem) bool { return ids[e.Num] })
		}
	case "section", "section-":
		if !s.Fig2Document {
			return false, nil
		}
		ids := make(map[int]bool)
		for _, arg := range args {
			id, err := strconv.Atoi(arg)
			if err != nil {
				return true, err
			}
			ids[id] = true
		}
		fig2ElemFilter(f, key[len(key)-1], func(e *Elem) bool { return ids[e.OriginalSection().Num] })
	case "pen":
		if len(args) < 1 {
			return true, NotEnoughArgs(key)
		}
		p, err := fig2ReadPen(s, args[1:])
		if err != nil {
			return true, err
		}
		if s.Fig2Pens == nil {
			s.Fig2Pens = make(map[string]fig2Pen)
		}
		name := strings.ToUpper(args[0])
		if _, ok := s.Fig2Pens[name]; !ok {
			s.Fig2PenOrder = append(s.Fig2PenOrder, name)
		}
		s.Fig2Pens[name] = p
	case "dot":
		if len(args) < 2 {
			return true, NotEnoughArgs(key)
		}
		vals, err := fig2Numbers(args[1:], len(args)-1)
		if err != nil {
			return true, err
		}
		for _, v := range vals {
			if v < .001 {
				return true, fmt.Errorf("DOT lengths must be at least 0.001 pt")
			}
		}
		if s.Fig2Dots == nil {
			s.Fig2Dots = make(map[string][]float64)
		}
		s.Fig2Dots[strings.ToUpper(args[0])] = vals
	case "max":
		fig2SetMax(f, args)
		s.Fig2MaxPen = fig2Pen{width: 0.3, color: RED}
	case "maxpen":
		p, err := fig2ReadPen(s, args)
		if err != nil {
			return true, err
		}
		s.Fig2MaxPen = p
	case "elemcode":
		s.Fig2ElemCodeMax = false
		if len(args) == 1 && strings.EqualFold(args[0], "max") {
			s.Fig2ElemCodeMax = !un
		}
		return false, nil
	case "alias":
		if !s.Fig2Document || un {
			return false, nil
		}
		if len(args) < 2 {
			return true, NotEnoughArgs(key)
		}
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return true, fmt.Errorf("Fig2 ALIAS requires a section number")
		}
		text := args[1:]
		for i, word := range text {
			if strings.HasPrefix(word, "#") {
				text = text[:i]
				break
			}
		}
		stw.AddSectionAlias(id, strings.Join(text, " "))
	case "stress":
		if !s.Fig2Document || un {
			return false, nil
		}
		if len(args) < 3 {
			return true, NotEnoughArgs(key)
		}
		var targets []int
		for _, target := range strings.Split(strings.ToUpper(args[0]), ",") {
			id, err := strconv.Atoi(target)
			if err != nil {
				id = -1
				for i, name := range ETYPES {
					if target == name {
						id = i
						break
					}
				}
				if id < 0 {
					return true, fmt.Errorf("unknown STRESS target %s", target)
				}
			}
			targets = append(targets, id)
		}
		period := strings.ToUpper(args[1])
		side := uint8(3)
		components := args[2:]
		for i, arg := range components {
			if strings.EqualFold(arg, "side") {
				side = 0
				for _, end := range components[i+1:] {
					switch end {
					case "0":
						side |= 1
					case "1":
						side |= 2
					default:
						return true, fmt.Errorf("STRESS SIDE accepts 0 or 1")
					}
				}
				components = components[:i]
				break
			}
		}
		if s.Fig2Stress == nil {
			s.Fig2Stress = make(map[int][6]fig2StressValue)
		}
		for _, component := range components {
			index := -1
			for i, name := range [...]string{"N", "QX", "QY", "MZ", "MX", "MY"} {
				if strings.EqualFold(component, name) {
					index = i
					break
				}
			}
			if strings.EqualFold(component, "MT") {
				index = 3
			}
			if index < 0 {
				return true, fmt.Errorf("unsupported STRESS component %s", component)
			}
			for _, id := range targets {
				spec := s.Fig2Stress[id]
				spec[index] = fig2StressValue{period: period, side: side}
				s.Fig2Stress[id] = spec
				StressOn(stw, id, uint(index))
			}
		}
		SetPeriod(stw, period)
	case "draw":
		if !s.Fig2Document || un {
			return false, nil
		}
		for _, arg := range args {
			if size, err := strconv.ParseFloat(arg, 64); err == nil {
				s.DrawSize = size
				continue
			}
			match := false
			for i, t := range ETYPES {
				if strings.EqualFold(t, arg) {
					s.Draw[i] = true
					match = true
					break
				}
			}
			if !match {
				return true, fmt.Errorf("unknown DRAW member type %s", arg)
			}
		}
	case "eccentric":
		if !s.Fig2Document {
			return false, nil
		}
		s.Fig2Eccentric = nil
		if un {
			break
		}
		if f.Fes == nil || !f.Fes.Calced {
			return true, fmt.Errorf("ECCENTRIC requires computed FACTS floor data")
		}
		for _, arg := range args {
			index, err := strconv.Atoi(arg)
			if err != nil || index < 1 || index >= len(f.Fes.CentreOfWeight) || index > len(f.Fes.CentreOfRigid) {
				return true, fmt.Errorf("ECCENTRIC floor %q unavailable", arg)
			}
			s.Fig2Eccentric = append(s.Fig2Eccentric, index)
		}
	case "wrect":
		s.Fig2Wrect = "ON"
		if len(args) > 0 {
			s.Fig2Wrect = strings.Join(args, " ")
		}
		if un {
			s.Fig2Wrect = ""
		}
	case "rainbow":
		vals := []float64{0, 0.2, 0.4, 0.6, 0.8, 1}
		if len(args) > 0 {
			var err error
			vals, err = fig2Numbers(args, 6)
			if err != nil {
				return true, err
			}
			for i := 1; i < len(vals); i++ {
				if vals[i] <= vals[i-1] {
					return true, fmt.Errorf("RAINBOW levels must increase")
				}
			}
		}
		s.Fig2Rainbow = vals
		stw.SetColorMode(ECOLOR_RATE)
	case "color", "colour":
		stw.SetColorMode(ECOLOR_BLACKSECT)
	case "rdisp":
		if len(args) < 1 {
			return true, NotEnoughArgs(key)
		}
		a := strings.Index("XYZ", strings.ToUpper(args[0]))
		if a < 0 || len(args[0]) != 1 {
			return true, fmt.Errorf("invalid RDISP direction")
		}
		s.Fig2RelativeDisp = a + 1
		if un {
			s.Fig2RelativeDisp = 0
		}
	case "disp":
		if !s.Fig2Document || un {
			return false, nil
		}
		if len(args) == 0 {
			return true, NotEnoughArgs(key)
		}
		period := args[0]
		axis := "Z"
		if len(args) >= 2 {
			axis = strings.ToUpper(args[1])
		}
		switch axis {
		case "RX", "RY", "RZ":
			axis = "T" + axis[1:]
		}
		s.Fig2DispFactor = 100
		if strings.EqualFold(args[len(args)-1], "mm") {
			s.Fig2DispFactor = 1000
		}
		SetPeriod(stw, strings.ToUpper(period))
		for i, name := range [...]string{"X", "Y", "Z", "TX", "TY", "TZ"} {
			if axis == name {
				DispOn(stw, i)
				return true, nil
			}
		}
		return true, fmt.Errorf("unknown DISP direction %s", axis)
	case "pload":
		if !s.Fig2Document || un {
			return false, nil
		}
		s.PointedLoad = true
		s.Fig2LoadAxes = 7
		if len(args) > 0 {
			if v, err := strconv.ParseFloat(args[0], 64); err == nil {
				s.Pfact = v
				args = args[1:]
			}
		}
		var chosen uint8
		for _, arg := range args {
			if arg == "0" || arg == "1" || arg == "2" {
				chosen |= 1 << uint(arg[0]-'0')
			}
			if strings.EqualFold(arg, "noarrow") {
				s.Fig2LoadNoArrow = true
			}
		}
		if chosen != 0 {
			s.Fig2LoadAxes = chosen
		}
	case "nodecode":
		s.Fig2NodeCode = ""
		if len(args) > 0 {
			s.Fig2NodeCode = strings.ToUpper(args[0])
		}
		return false, nil
	case "pressurearea":
		if len(args) < 2 {
			return true, NotEnoughArgs(key)
		}
		area, err := fig2Numbers(args[1:], 1)
		id, idErr := strconv.Atoi(args[0])
		if err != nil || idErr != nil || area[0] <= 0 || f.Nodes[id] == nil {
			return true, fmt.Errorf("invalid PRESSUREAREA: expected a node number and a positive area in m²")
		}
		if s.Fig2PressureAreas == nil {
			s.Fig2PressureAreas = make(map[int]float64)
		}
		s.Fig2PressureAreas[id] = area[0]
	case "reaction", "force", "pressure":
		if un {
			s.Fig2NodalValues = nil
			break
		}
		v, err := fig2NodalSpec(args, key == "force")
		if err != nil {
			return true, err
		}
		v.pressure = key == "pressure"
		if v.pressure && v.axis >= 3 {
			return true, fmt.Errorf("PRESSURE requires X, Y, or Z")
		}
		s.Fig2NodalValues = append(s.Fig2NodalValues, v)
	case "nodata":
		for _, name := range []string{"DATA", "INPUTFILE", "OUTPUTFILE", "VIEWPOINT"} {
			stw.TextBox(name).Hide()
		}
	case "doc", "blank":
		fig2NodeFilter(f, false, func(*Node) bool { return false })
		s.GlobalAxis, s.ElementAxis, s.Kijun = false, false, false
		for _, name := range []string{"DATA", "INPUTFILE", "OUTPUTFILE", "VIEWPOINT"} {
			stw.TextBox(name).Hide()
		}
	case "nozero":
		s.Fig2NoZero = !un
	case "length", "area":
		s.Fig2Size = key
		if un {
			s.Fig2Size = ""
		}
	case "hinge":
		s.Bond = !un
	case "phinge":
		s.Phinge = !un
	case "yieldratio":
		s.YieldFunction = !un
	case "nocolour", "nocolor":
		stw.SetColorMode(ECOLOR_BLACK)
	case "axialcolor", "axialcolour":
		stw.SetColorMode(ECOLOR_N)
	case "ncolour":
		return true, Fig2Keyword(stw, append([]string{"ncolor"}, args...), un)
	case "gfact":
		if len(args) == 0 || !strings.EqualFold(args[0], "fit") {
			return false, nil
		}
		return true, Fig2Keyword(stw, []string{"fit"}, un)
	case "shift":
		v, err := fig2Numbers(args, 2)
		if err != nil {
			return true, err
		}
		f.View.Center[0] += v[0]
		direction := -1.0
		if d, ok := stw.(interface{ CanvasDirection() int }); ok && d.CanvasDirection() == 1 {
			direction = 1
		}
		f.View.Center[1] -= v[1] * direction
	case "digit":
		if len(args) < 2 {
			return true, NotEnoughArgs(key)
		}
		digits, err := strconv.Atoi(args[1])
		if err != nil || digits < 0 || digits > 15 {
			return true, fmt.Errorf("invalid DIGIT precision")
		}
		name := strings.ToUpper(args[0])
		format := "f"
		if name == "THETA" {
			format = "e"
		}
		s.Formats[name] = fmt.Sprintf("%%.%d%s", digits, format)
	case "timestamp", "header":
		b := stw.TextBox("PAGETITLE")
		now := time.Now().Format("2006-01-02 Mon 15:04")
		line := now
		if key == "header" {
			line = strings.NewReplacer("{name}", filepath.Base(f.Path), "{prj}", filepath.Base(filepath.Dir(f.Path)), "{time}", now).Replace(strings.Join(args, " "))
		}
		if un {
			b.Hide()
		} else {
			b.AddText(line)
			b.Show()
		}
	case "kijunmove":
		v, err := fig2Numbers(args, 1)
		if err != nil {
			return true, err
		}
		s.Fig2KijunOffset += v[0]
	case "kijun":
		if un {
			s.Fig2KijunOffset, s.Fig2KijunNoHead = 0, false
			return false, nil
		}
		s.Kijun = true
		stw.EnableLabel("KIJUN")
		var patterns []*regexp.Regexp
		var inverse []bool
		for _, arg := range args {
			switch {
			case strings.EqualFold(arg, "nohead"):
				s.Fig2KijunNoHead = true
			case strings.HasPrefix(strings.ToUpper(arg), "Z="):
				v, err := fig2Numbers([]string{arg[2:]}, 1)
				if err != nil {
					return true, err
				}
				s.Fig2KijunOffset += v[0]
			default:
				if v, err := strconv.ParseFloat(arg, 64); err == nil {
					s.KijunSize = v
					continue
				}
				name, neg := strings.CutPrefix(arg, "!")
				if f.Kijuns[strings.ToLower(name)] != nil {
					name = "^" + regexp.QuoteMeta(strings.ToUpper(name)) + "$"
				}
				pat, err := regexp.Compile(name)
				if err != nil {
					return true, err
				}
				patterns = append(patterns, pat)
				inverse = append(inverse, neg)
			}
		}
		for _, k := range f.Kijuns {
			on := len(patterns) == 0
			for i, pat := range patterns {
				on = on || pat.MatchString(k.Name) != inverse[i]
			}
			if on {
				k.Show()
			} else {
				k.Hide()
			}
		}
	default:
		if s.Fig2Boxes[strings.ToUpper(key)] {
			b := stw.TextBox(strings.ToUpper(key))
			b.AddText(strings.Join(args, " "))
			b.Show()
			return true, nil
		}
		if pen, ok := s.Fig2Pens[strings.ToUpper(key)]; ok {
			return true, fig2ApplyPen(f, pen, args)
		}
		k := f.Kijuns[strings.ToLower(key)]
		if k == nil {
			return false, nil
		}
		a, b := -0.1, 0.1
		if v, err := fig2Numbers(args, 2); err == nil {
			a, b = v[0], v[1]
		}
		d := k.Direction()
		reverse, noName := false, false
		for _, arg := range args {
			reverse = reverse || strings.EqualFold(arg, "reverse")
			noName = noName || strings.EqualFold(arg, "noname")
		}
		f.View.Angle[0], f.View.Angle[1] = 0, math.Atan2(d[1], d[0])*180/math.Pi-90
		if reverse {
			f.View.Angle[1] -= 180
		}
		fig2NodeFilter(f, false, func(n *Node) bool {
			var v float64
			switch {
			case IsParallel(d, XAXIS, stw.EPS()):
				v = n.Coord[1] - k.Start[1]
			case IsParallel(d, YAXIS, stw.EPS()):
				v = n.Coord[0] - k.Start[0]
			default:
				v = (n.Coord[0]-k.Start[0])*d[1] - (n.Coord[1]-k.Start[1])*d[0]
			}
			if s.Fig2Document && (IsParallel(d, XAXIS, stw.EPS()) || IsParallel(d, YAXIS, stw.EPS())) {
				return a-1e-4 <= v && v <= b+1e-4
			}
			return a <= v && v <= b
		})
		if !noName {
			b := stw.TextBox("PAGETITLE")
			b.AddText(k.Name + "通り")
			b.Show()
		}
	}
	return true, nil
}

func showFig2NodeCode(n *Node, mode string) bool {
	switch mode {
	case "CONF":
		for _, fixed := range n.Conf {
			if fixed {
				return true
			}
		}
		return false
	case "PLOAD":
		for _, load := range n.Load {
			if load != 0 {
				return true
			}
		}
		return false
	default:
		return true
	}
}

func fig2NodeValue(n *Node, spec fig2NodalValue) (float64, bool) {
	i := spec.axis
	if !spec.force && !showFig2NodeCode(n, "CONF") {
		return 0, false
	}
	var v float64
	if spec.period != "" {
		if spec.force {
			v = PeriodValue(spec.period, func(p string, s float64) float64 {
				if a := n.Force[p]; i < len(a) {
					return a[i] * s
				}
				return 0
			})
		} else {
			v = n.ReturnReaction(spec.period, i)
		}
	}
	if spec.weight && i == 2 {
		v += n.Weight[1]
	}
	if spec.pressure {
		area := n.Frame.Show.Fig2PressureAreas[n.Num]
		if area <= 0 {
			return 0, false
		}
		v /= area
	}
	return v, true
}

func addFig2Measure(f *Frame, start, end, direction []float64) *Measure {
	if !f.Show.Fig2Document {
		return f.AddMeasure(start, end, direction)
	}
	m := NewMeasure(start, end, direction)
	m.Frame = f
	f.Show.Fig2Measures = append(f.Show.Fig2Measures, m)
	return m
}

func fig2Measure(f *Frame, key string, args []string) error {
	if len(args) < 4 {
		return NotEnoughArgs(key)
	}
	var nodes [2]*Node
	for i, x := range args[:2] {
		id, err := strconv.Atoi(x)
		if err != nil || f.Nodes[id] == nil {
			return fmt.Errorf("unknown node %q", x)
		}
		nodes[i] = f.Nodes[id]
	}
	axis := strings.Index("XYZ", strings.ToUpper(args[2]))
	if axis < 0 || len(args[2]) != 1 {
		return fmt.Errorf("unknown measure direction %s", args[2])
	}
	dir := make([]float64, 3)
	dir[axis] = 1
	offset, err := fig2Numbers(args[3:], 1)
	if err != nil {
		return err
	}
	a, b := nodes[0].Coord, nodes[1].Coord
	d := fig2Diff(b, a)
	length := math.Sqrt(Dot(d, d, 3))
	if key != "measure" {
		component := strings.IndexByte("xyz", key[0])
		length = math.Abs(d[component])
		b = append([]float64(nil), a...)
		b[component] += d[component]
	}
	m := addFig2Measure(f, a, b, dir)
	m.Extension, m.ArrowSize = offset[0], 0.1
	m.Text = fmt.Sprintf("%.0f", length*1000)
	if len(args) > 4 {
		switch strings.ToUpper(args[4]) {
		case "NONE":
			m.Text = ""
		case "DEFAULT":
		default:
			m.Text = strings.ReplaceAll(args[4], "_", m.Text)
		}
	}
	if len(args) > 5 {
		if v, err := fig2Numbers(args[5:], 1); err == nil {
			m.ArrowSize = v[0]
		}
	}
	if len(args) > 6 {
		if v, err := fig2Numbers(args[6:], 1); err == nil {
			m.Rotate = v[0]
		}
	}
	f.Show.Measure = true
	return nil
}

func positionFig2Annotations(w Fig2Moder) error {
	f := w.Frame()
	if len(f.Show.Fig2TextAnchors) == 0 {
		return nil
	}
	direction := 0
	if d, ok := w.(interface{ CanvasDirection() int }); ok {
		direction = d.CanvasDirection()
	}
	f.View.Set(direction)
	for name, p := range f.Show.Fig2TextAnchors {
		id, err := strconv.Atoi(p[1])
		if err != nil {
			return err
		}
		var coord []float64
		if strings.EqualFold(p[0], "node") {
			if node := f.Nodes[id]; node != nil {
				coord = node.Coord
			}
		} else if el := f.Elems[id]; el != nil {
			coord = el.MidPoint()
		}
		if coord == nil {
			return fmt.Errorf("POSITION %s: unknown %s %d", name, p[0], id)
		}
		xy := f.View.ProjectCoord(coord)
		if direction == 1 {
			_, h := w.GetCanvasSize()
			xy[1] = float64(h) - xy[1]
		}
		w.TextBox(name).SetPosition(xy[0], xy[1])
	}
	return nil
}

func validateFig2Pressure(f *Frame) error {
	for _, spec := range f.Show.Fig2NodalValues {
		if !spec.pressure {
			continue
		}
		for id, n := range f.Nodes {
			if n.IsHidden(f.Show) || f.Show.Fig2PressureAreas[id] > 0 {
				continue
			}
			base := spec
			base.pressure = false
			if value, ok := fig2NodeValue(n, base); ok && value != 0 {
				return fmt.Errorf("PRESSURE: node %d requires PRESSUREAREA %d <area in m²>", id, id)
			}
		}
	}
	return nil
}
