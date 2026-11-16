package st

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var fig2ValueCommand = regexp.MustCompile("^([^ ]+)\\s+(\"(?:[^\"\\\\]|\\\\.)*\")\\s+(.+)$")
var fig2ValueFormat = regexp.MustCompile("\\{(?:0)?(?::([-+ ]?)(0?)([0-9]{0,2})(?:\\.([0-9]{1,2}))?([efgds])?)?\\}")

func fig2FormatValue(format string, v any) (string, error) {
	if !fig2ValueFormat.MatchString(format) {
		return "", fmt.Errorf("expected a format field such as {:.3f}")
	}
	var formatErr error
	out := fig2ValueFormat.ReplaceAllStringFunc(format, func(field string) string {
		m := fig2ValueFormat.FindStringSubmatch(field)
		if m[5] == "" || m[5] == "s" {
			return fmt.Sprint(v)
		}
		n, ok := v.(float64)
		if !ok || !finitePDF(n) {
			formatErr = fmt.Errorf("format %s requires a finite number", field)
			return ""
		}
		if m[5] == "d" && (n != math.Trunc(n) || math.Abs(n) >= math.MaxInt64) {
			formatErr = fmt.Errorf("format %s requires an integer", field)
			return ""
		}
		sign := m[1]
		if sign == "-" {
			sign = ""
		}
		spec := "%" + sign + m[2] + m[3]
		if m[4] != "" {
			spec += "." + m[4]
		}
		spec += m[5]
		if m[5] == "d" {
			return fmt.Sprintf(spec, int64(n))
		}
		return fmt.Sprintf(spec, n)
	})
	return out, formatErr
}

func fig2EvalValue(expr fig2Expression, f *Frame, x any) (any, error) {
	env := map[string]any{"self": f, "x": x}
	for k, v := range f.Show.fig2Variables {
		env[k] = v
	}
	return expr(env)
}

func applyFig2Expression(f *Frame, key string, args []string, un bool) error {
	s := f.Show
	if un {
		switch key {
		case "rf":
			s.fig2RainbowValues = nil
		case "ef":
			s.Fig2Elems = nil
		case "nf":
			s.Fig2Nodes = nil
		case "ev":
			s.fig2ElemValues = nil
		case "nv":
			s.fig2NodeValues = nil
		case "eval":
			if len(args) == 1 {
				delete(s.fig2Variables, args[0])
			}
		}
		return nil
	}
	if len(args) == 0 {
		return NotEnoughArgs(key)
	}
	source := strings.Join(args, " ")
	var label, format string
	switch key {
	case "ev", "nv":
		m := fig2ValueCommand.FindStringSubmatch(source)
		if m == nil {
			return fmt.Errorf("%s requires a label, a quoted format and an expression", strings.ToUpper(key))
		}
		var err error
		format, err = strconv.Unquote(m[2])
		if err != nil {
			return err
		}
		label, source = m[1], m[3]
	case "eval":
		if len(args) < 2 {
			return NotEnoughArgs(key)
		}
		source = strings.Join(args[1:], " ")
	}
	expr, err := compileFig2Expr(source)
	if err != nil {
		return err
	}
	if key == "eval" {
		value, err := fig2EvalValue(expr, f, nil)
		if err != nil {
			return err
		}
		if s.fig2Variables == nil {
			s.fig2Variables = make(map[string]any)
		}
		s.fig2Variables[args[0]] = value
		return nil
	}
	if key == "ef" || key == "ev" || key == "rf" {
		selected := make(map[int]bool)
		captions := make(map[int]string)
		scalars := make(map[int]float64)
		for id, e := range f.Elems {
			if key == "ef" {
				if visible, ok := s.Fig2Elems[id]; ok && !visible {
					continue
				}
			} else if e.IsHidden(s) {
				continue
			}
			v, err := fig2EvalValue(expr, f, e)
			if err != nil {
				return fmt.Errorf("element %d: %w", id, err)
			}
			switch key {
			case "ef":
				selected[id] = fig2Truthy(v)
			case "ev":
				txt, err := fig2FormatValue(format, v)
				if err != nil {
					return fmt.Errorf("element %d: %w", id, err)
				}
				captions[id] = txt
			case "rf":
				value, ok := v.(float64)
				if !ok || !finitePDF(value) {
					return fmt.Errorf("RF requires a finite number on element %d", id)
				}
				scalars[id] = value
			}
		}
		switch key {
		case "ef":
			fig2ElemFilter(f, 0, func(e *Elem) bool { return selected[e.Num] })
		case "ev":
			if s.fig2ElemValues == nil {
				s.fig2ElemValues = make(map[int]string)
			}
			for id, txt := range captions {
				s.fig2ElemValues[id] += txt + "\n"
			}
		case "rf":
			s.fig2RainbowValues = scalars
		}
	} else {
		selected := make(map[int]bool)
		captions := make(map[int]string)
		for id, n := range f.Nodes {
			if n.IsHidden(s) {
				continue
			}
			v, err := fig2EvalValue(expr, f, n)
			if err != nil {
				return fmt.Errorf("node %d: %w", id, err)
			}
			if key == "nf" {
				selected[id] = fig2Truthy(v)
				continue
			}
			txt, err := fig2FormatValue(format, v)
			if err != nil {
				return fmt.Errorf("node %d: %w", id, err)
			}
			captions[id] = txt
		}
		if key == "nf" {
			fig2NodeFilter(f, false, func(n *Node) bool { return selected[n.Num] })
		} else {
			if s.fig2NodeValues == nil {
				s.fig2NodeValues = make(map[int]string)
			}
			for id, txt := range captions {
				s.fig2NodeValues[id] += txt + "\n"
			}
		}
	}
	if label != "" {
		s.fig2ValueLabels = append(s.fig2ValueLabels, label)
	}
	return nil
}

func fig2Welding(f *Frame, args []string) error {
	period := "L"
	if len(args) > 0 {
		period = strings.ToUpper(args[0])
	}
	sections := map[int]bool{201: true, 205: true, 206: true, 501: true, 503: true, 601: true}
	if len(args) > 1 {
		sections = make(map[int]bool)
		for _, arg := range args[1:] {
			id, err := strconv.Atoi(arg)
			if err != nil {
				return err
			}
			sections[id] = true
		}
	}
	connected := make(map[int][]*Elem)
	for _, e := range f.Elems {
		for _, n := range e.Enod {
			connected[n.Num] = append(connected[n.Num], e)
		}
	}
	out := make(map[int]float64)
	for id, elems := range connected {
		sort.Slice(elems, func(i, j int) bool { return elems[i].Num < elems[j].Num })
		var col *Elem
		for _, e := range elems {
			if e.Etype == COLUMN && e.IsLineElem() {
				col = e
				break
			}
		}
		if col == nil || f.Nodes[id].IsHidden(f.Show) {
			continue
		}
		global := make([]float64, 3)
		for _, e := range elems {
			if !sections[e.Sect.Num] {
				continue
			}
			if e.IsLineElem() {
				addFig2WeldForce(global, e, id, period)
			} else {
				for _, child := range e.Children {
					if child != nil && child.IsLineElem() {
						addFig2WeldForce(global, child, id, period)
					}
				}
			}
		}
		a, b, c := Dot(global, col.Weak, 3), global[2], Dot(global, col.Strong, 3)
		out[id] = math.Sqrt(a*a + b*b + c*c)
	}
	f.Show.fig2WeldValues = out
	return nil
}

func addFig2WeldForce(total []float64, e *Elem, id int, period string) {
	end, sign := 0, 1.0
	if e.Enod[1].Num == id {
		end, sign = 1, -1
	} else if e.Enod[0].Num != id {
		return
	}
	direction := e.Direction(true)
	if !finitePDF(direction...) {
		return
	}
	for i, axis := range [3][]float64{direction, e.Strong, e.Weak} {
		if len(axis) < 3 {
			continue
		}
		v := sign * e.ReturnStress(period, end, i)
		for j := 0; j < 3; j++ {
			total[j] += v * axis[j]
		}
	}
}

type fig2JointRates map[int]map[int]map[string]float64

func readFig2JointRates(f *Frame, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	rates := make(fig2JointRates)
	enum := 0
	for i, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if fields[0] == "ELEM" && len(fields) >= 7 && fields[2] == "NODE" && fields[5] == "SECT" {
			enum, err = strconv.Atoi(fields[1])
			if err != nil {
				return fmt.Errorf("%s:%d: invalid member number", path, i+1)
			}
			continue
		}
		if len(fields) < 11 || fields[1] != "NODE" || fields[3] != "PERIOD" {
			continue
		}
		n, e1 := strconv.Atoi(fields[2])
		v, e2 := strconv.ParseFloat(fields[len(fields)-1], 64)
		e := f.Elems[enum]
		if e1 != nil || e2 != nil || !finitePDF(v) {
			return fmt.Errorf("%s:%d: invalid joint rate", path, i+1)
		}
		if e == nil || !e.IsLineElem() || (e.Enod[0].Num != n && e.Enod[1].Num != n) {
			continue
		}
		if rates[enum] == nil {
			rates[enum] = make(map[int]map[string]float64)
		}
		if rates[enum][n] == nil {
			rates[enum][n] = make(map[string]float64)
		}
		rates[enum][n][strings.ToUpper(fields[4])] = v
	}
	if len(rates) == 0 {
		return fmt.Errorf("%s contains no joint rates for this model", path)
	}
	f.fig2JointRates = rates
	return nil
}

func setupFig2Joint(f *Frame, args []string, un bool) error {
	f.Show.fig2JointPeriod = ""
	if un {
		return nil
	}
	period := "ULTIMATE"
	if len(args) > 0 {
		period = strings.ToUpper(args[0])
	}
	switch period {
	case "LONG", "SHORT", "ULTIMATE":
	default:
		return fmt.Errorf("SRCANJOINT requires LONG, SHORT, or ULTIMATE")
	}
	if len(args) > 1 || len(f.fig2JointRates) == 0 {
		path := Ce(f.Path, ".ratj")
		if len(args) > 1 {
			path = strings.Join(args[1:], " ")
			if !filepath.IsAbs(path) {
				path = filepath.Join(filepath.Dir(f.Path), path)
			}
		}
		if err := readFig2JointRates(f, path); err != nil {
			return err
		}
	}
	f.Show.fig2JointPeriod = period
	return nil
}

func fig2JointRate(e *Elem, mode string) (float64, bool) {
	if mode == "" || !e.IsLineElem() {
		return 0, false
	}
	if len(e.Bonds) < 12 || (e.Bonds[4] != Pin && e.Bonds[10] != Pin) {
		return 0, false
	}
	top, ok := 0.0, false
	for _, rates := range e.Frame.fig2JointRates[e.Num] {
		for period, v := range rates {
			if mode == "LONG" && period != "L" || mode == "SHORT" && period == "L" {
				continue
			}
			if !ok || v > top {
				top = v
				ok = true
			}
		}
	}
	return top, ok
}
