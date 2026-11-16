package st

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

type fig2Expression func(map[string]any) (any, error)
type fig2Callable func([]any) (any, error)

var fig2Tokens = regexp.MustCompile("\\s+|'(?:[^'\\\\]|\\\\.)*'|\"(?:[^\"\\\\]|\\\\.)*\"|(?:[0-9]+(?:\\.[0-9]*)?|\\.[0-9]+)(?:[eE][+-]?[0-9]+)?|[A-Za-z_][A-Za-z_0-9]*|\\*\\*|//|==|!=|<=|>=|[^\\s]")

type fig2ExprParser struct {
	tokens []string
	pos    int
}

func compileFig2Expr(source string) (fig2Expression, error) {
	if len(source) > 8192 {
		return nil, fmt.Errorf("Fig2 expression too long")
	}
	var tokens []string
	for _, t := range fig2Tokens.FindAllString(source, -1) {
		if strings.TrimSpace(t) != "" {
			tokens = append(tokens, t)
		}
	}
	if len(tokens) > 256 {
		return nil, fmt.Errorf("too many Fig2 expression tokens")
	}
	p := fig2ExprParser{tokens: tokens}
	x, err := p.expr(1)
	if err != nil {
		return nil, err
	}
	if p.peek() != "" {
		return nil, fmt.Errorf("unexpected expression token %q", p.peek())
	}
	return x, nil
}

func (p *fig2ExprParser) peek() string {
	if p.pos >= len(p.tokens) {
		return ""
	}
	return p.tokens[p.pos]
}
func (p *fig2ExprParser) take() string {
	t := p.peek()
	if t != "" {
		p.pos++
	}
	return t
}
func (p *fig2ExprParser) consume(t string) bool {
	if p.peek() != t {
		return false
	}
	p.pos++
	return true
}

var fig2Precedence = map[string]int{"or": 1, "and": 2, "==": 4, "!=": 4, "<": 4, "<=": 4, ">": 4, ">=": 4, "in": 4, "not in": 4, "+": 5, "-": 5, "*": 6, "/": 6, "//": 6, "%": 6, "**": 8}

func (p *fig2ExprParser) expr(minPrec int) (fig2Expression, error) {
	left, err := p.primary()
	if err != nil {
		return nil, err
	}
	var comparisons []fig2Expression
	for {
		op := p.peek()
		notIn := op == "not" && p.pos+1 < len(p.tokens) && p.tokens[p.pos+1] == "in"
		if notIn {
			op = "not in"
		}
		prec := fig2Precedence[op]
		if prec < minPrec {
			break
		}
		p.take()
		if notIn {
			p.take()
		}
		next := prec + 1
		if op == "**" {
			next = prec
		}
		right, err := p.expr(next)
		if err != nil {
			return nil, err
		}
		if prec == 4 {
			comparisons = append(comparisons, fig2Binary(op, left, right))
			left = right
			follow := p.peek()
			if fig2Precedence[follow] != 4 && !(follow == "not" && p.pos+1 < len(p.tokens) && p.tokens[p.pos+1] == "in") {
				left = comparisons[0]
				for _, comparison := range comparisons[1:] {
					left = fig2Binary("and", left, comparison)
				}
				comparisons = nil
			}
		} else {
			left = fig2Binary(op, left, right)
		}
	}
	if minPrec == 1 && p.consume("if") {
		condition, err := p.expr(1)
		if err != nil {
			return nil, err
		}
		if !p.consume("else") {
			return nil, fmt.Errorf("expected else")
		}
		other, err := p.expr(1)
		if err != nil {
			return nil, err
		}
		body := left
		left = func(env map[string]any) (any, error) {
			v, err := condition(env)
			if err != nil {
				return nil, err
			}
			if fig2Truthy(v) {
				return body(env)
			}
			return other(env)
		}
	}
	return left, nil
}

func (p *fig2ExprParser) list(close string) ([]fig2Expression, error) {
	var out []fig2Expression
	for !p.consume(close) {
		x, err := p.expr(1)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
		if p.consume(close) {
			return out, nil
		}
		if !p.consume(",") {
			return nil, fmt.Errorf("expected comma or %s", close)
		}
	}
	return out, nil
}

func (p *fig2ExprParser) primary() (fig2Expression, error) {
	t := p.take()
	var result fig2Expression
	switch t {
	case "+", "-", "not":
		prec := 7
		if t == "not" {
			prec = 3
		}
		x, err := p.expr(prec)
		if err != nil {
			return nil, err
		}
		return func(env map[string]any) (any, error) {
			v, err := x(env)
			if err != nil {
				return nil, err
			}
			if t == "not" {
				return !fig2Truthy(v), nil
			}
			n, ok := v.(float64)
			if !ok {
				return nil, fmt.Errorf("%s requires a number", t)
			}
			if t == "-" {
				n = -n
			}
			return n, nil
		}, nil
	case "(", "[":
		close := ")"
		if t == "[" {
			close = "]"
		}
		values, err := p.list(close)
		if err != nil {
			return nil, err
		}
		if t == "(" && len(values) == 1 {
			result = values[0]
		} else {
			result = func(env map[string]any) (any, error) { return fig2EvalList(values, env) }
		}
	case "True", "False", "None":
		result = func(map[string]any) (any, error) {
			if t == "None" {
				return nil, nil
			}
			return t == "True", nil
		}
	case "@":
		result = func(env map[string]any) (any, error) { return env["self"], nil }
	case "":
		return nil, fmt.Errorf("missing expression")
	default:
		if t[0] == '\'' || t[0] == '"' {
			var value strings.Builder
			quoted := t
			for len(quoted) > 2 {
				r, _, tail, err := strconv.UnquoteChar(quoted[1:len(quoted)-1], t[0])
				if err != nil {
					return nil, err
				}
				value.WriteRune(r)
				quoted = t[:1] + tail + t[:1]
			}
			result = func(map[string]any) (any, error) { return value.String(), nil }
		} else if n, err := strconv.ParseFloat(t, 64); err == nil {
			result = func(map[string]any) (any, error) { return n, nil }
		} else if t[0] >= 'A' && t[0] <= 'Z' || t[0] >= 'a' && t[0] <= 'z' {
			result = func(env map[string]any) (any, error) {
				if t == "math" || t == "np" || t == "numpy" {
					return fig2MathNamespace{}, nil
				}
				if v, ok := env[t]; ok {
					return v, nil
				}
				if v, ok := fig2Builtin(t); ok {
					return v, nil
				}
				return nil, fmt.Errorf("unknown name %s", t)
			}
		} else {
			return nil, fmt.Errorf("unexpected token %q", t)
		}
	}
	for {
		base := result
		switch {
		case p.consume("."):
			name := p.take()
			result = func(env map[string]any) (any, error) {
				v, err := base(env)
				if err != nil {
					return nil, err
				}
				return fig2Property(v, name)
			}
		case p.consume("["):
			index, err := p.expr(1)
			if err != nil {
				return nil, err
			}
			if !p.consume("]") {
				return nil, fmt.Errorf("expected ]")
			}
			result = func(env map[string]any) (any, error) {
				v, err := base(env)
				if err != nil {
					return nil, err
				}
				i, err := index(env)
				if err != nil {
					return nil, err
				}
				return fig2Index(v, i)
			}
		case p.consume("("):
			args, err := p.list(")")
			if err != nil {
				return nil, err
			}
			result = func(env map[string]any) (any, error) {
				v, err := base(env)
				if err != nil {
					return nil, err
				}
				f, ok := v.(fig2Callable)
				if !ok {
					return nil, fmt.Errorf("value is not callable")
				}
				values, err := fig2EvalList(args, env)
				if err != nil {
					return nil, err
				}
				return f(values)
			}
		default:
			return result, nil
		}
	}
}

func fig2EvalList(xs []fig2Expression, env map[string]any) ([]any, error) {
	out := make([]any, len(xs))
	for i, x := range xs {
		v, err := x(env)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

func fig2Truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case float64:
		return x != 0
	case string:
		return x != ""
	case []any:
		return len(x) > 0
	default:
		return true
	}
}

func fig2Binary(op string, left, right fig2Expression) fig2Expression {
	return func(env map[string]any) (any, error) {
		a, err := left(env)
		if err != nil {
			return nil, err
		}
		if op == "and" && !fig2Truthy(a) || op == "or" && fig2Truthy(a) {
			return a, nil
		}
		b, err := right(env)
		if err != nil {
			return nil, err
		}
		if op == "and" || op == "or" {
			return b, nil
		}
		if op == "==" || op == "!=" {
			return fig2Equal(a, b) == (op == "=="), nil
		}
		if op == "in" || op == "not in" {
			found := false
			switch seq := b.(type) {
			case []any:
				for _, v := range seq {
					found = found || fig2Equal(a, v)
				}
			case map[any]any:
				for k := range seq {
					found = found || fig2Equal(a, k)
				}
			case string:
				if s, ok := a.(string); ok {
					found = strings.Contains(seq, s)
				} else {
					return nil, fmt.Errorf("string membership requires a string")
				}
			default:
				return nil, fmt.Errorf("invalid membership operand")
			}
			return found == (op == "in"), nil
		}
		if s, ok := a.(string); ok {
			if t, ok := b.(string); ok {
				switch op {
				case "+":
					return s + t, nil
				case "<":
					return s < t, nil
				case "<=":
					return s <= t, nil
				case ">":
					return s > t, nil
				case ">=":
					return s >= t, nil
				}
			}
		}
		x, xok := a.(float64)
		y, yok := b.(float64)
		if !xok || !yok {
			return nil, fmt.Errorf("%s requires numeric operands", op)
		}
		switch op {
		case "<":
			return x < y, nil
		case "<=":
			return x <= y, nil
		case ">":
			return x > y, nil
		case ">=":
			return x >= y, nil
		case "+":
			return x + y, nil
		case "-":
			return x - y, nil
		case "*":
			return x * y, nil
		case "**":
			return math.Pow(x, y), nil
		case "/", "//", "%":
			if y == 0 {
				return nil, fmt.Errorf("division by zero")
			}
			switch op {
			case "/":
				return x / y, nil
			case "//":
				return math.Floor(x / y), nil
			default:
				return x - y*math.Floor(x/y), nil
			}
		}
		return nil, fmt.Errorf("unknown operator %s", op)
	}
}

func fig2Equal(a, b any) bool {
	switch x := a.(type) {
	case nil:
		return b == nil
	case float64:
		y, ok := b.(float64)
		return ok && x == y
	case string:
		y, ok := b.(string)
		return ok && x == y
	case bool:
		y, ok := b.(bool)
		return ok && x == y
	}
	return false
}

func fig2Index(v, key any) (any, error) {
	if m, ok := v.(map[any]any); ok {
		for k, v := range m {
			if fig2Equal(k, key) {
				return v, nil
			}
		}
		return nil, fmt.Errorf("unknown index %v", key)
	}
	n, ok := key.(float64)
	if !ok || n != math.Trunc(n) || !finitePDF(n) {
		return nil, fmt.Errorf("expected integer index")
	}
	switch seq := v.(type) {
	case []any:
		if n < 0 {
			n += float64(len(seq))
		}
		if n >= 0 && n < float64(len(seq)) {
			return seq[int(n)], nil
		}
	case string:
		r := []rune(seq)
		if n < 0 {
			n += float64(len(r))
		}
		if n >= 0 && n < float64(len(r)) {
			return string(r[int(n)]), nil
		}
	}
	return nil, fmt.Errorf("index out of range")
}

func fig2Builtin(name string) (fig2Callable, bool) {
	switch name {
	case "abs", "sqrt", "min", "max", "len", "sum", "round", "int", "float", "str":
	default:
		return nil, false
	}
	return func(args []any) (any, error) {
		if len(args) == 0 {
			return nil, fmt.Errorf("%s requires arguments", name)
		}
		if name == "str" && len(args) == 1 {
			return fmt.Sprint(args[0]), nil
		}
		if name == "len" && len(args) == 1 {
			switch x := args[0].(type) {
			case []any:
				return float64(len(x)), nil
			case map[any]any:
				return float64(len(x)), nil
			case string:
				return float64(len([]rune(x))), nil
			}
		}
		if name == "min" || name == "max" || name == "sum" {
			if seq, ok := args[0].([]any); ok && len(args) == 1 {
				args = seq
			}
			if len(args) == 0 && name != "sum" {
				return nil, fmt.Errorf("empty %s", name)
			}
			r := 0.0
			for i, arg := range args {
				n, ok := arg.(float64)
				if !ok {
					return nil, fmt.Errorf("%s requires numbers", name)
				}
				switch name {
				case "sum":
					r += n
				case "min":
					if i == 0 || n < r {
						r = n
					}
				case "max":
					if i == 0 || n > r {
						r = n
					}
				}
			}
			return r, nil
		}
		n, ok := args[0].(float64)
		if !ok && (name == "float" || name == "int") {
			if s, okString := args[0].(string); okString {
				var err error
				n, err = strconv.ParseFloat(s, 64)
				ok = err == nil
			}
		}
		if ok && len(args) == 1 {
			switch name {
			case "abs":
				return math.Abs(n), nil
			case "sqrt":
				if n >= 0 {
					return math.Sqrt(n), nil
				}
			case "int":
				return math.Trunc(n), nil
			case "float":
				return n, nil
			case "round":
				return math.RoundToEven(n), nil
			}
		}
		if ok && name == "round" && len(args) == 2 {
			if d, ok := args[1].(float64); ok && d == math.Trunc(d) && math.Abs(d) <= 15 {
				s := math.Pow(10, d)
				return math.RoundToEven(n*s) / s, nil
			}
		}
		return nil, fmt.Errorf("invalid arguments to %s", name)
	}, true
}

func fig2NumbersAsValues[T ~float64 | ~int](values []T) []any {
	out := make([]any, len(values))
	for i, v := range values {
		out[i] = float64(v)
	}
	return out
}

type fig2MathNamespace struct{}

func fig2Property(v any, name string) (any, error) {
	switch x := v.(type) {
	case fig2MathNamespace:
		switch name {
		case "pi":
			return math.Pi, nil
		case "e":
			return math.E, nil
		case "abs", "sqrt", "min", "max", "sum", "round":
			fn, _ := fig2Builtin(name)
			return fn, nil
		case "norm":
			return fig2Callable(func(args []any) (any, error) {
				if len(args) == 1 {
					if seq, ok := args[0].([]any); ok {
						total := 0.0
						for _, v := range seq {
							n, ok := v.(float64)
							if !ok {
								return nil, fmt.Errorf("norm requires numeric values")
							}
							total = math.Hypot(total, n)
						}
						return total, nil
					}
				}
				return nil, fmt.Errorf("norm requires a numeric sequence")
			}), nil
		case "linalg":
			return x, nil
		}
	case *Frame:
		if x == nil {
			break
		}
		switch name {
		case "name":
			return PruneExt(x.Path), nil
		case "levels":
			return fig2NumbersAsValues(x.Ai.Boundary), nil
		case "angle":
			return fig2NumbersAsValues(x.View.Angle), nil
		case "focus":
			return fig2NumbersAsValues(x.View.Focus), nil
		case "gfact":
			return x.View.Gfact, nil
		case "nodes":
			m := make(map[any]any, len(x.Nodes))
			for id, n := range x.Nodes {
				m[float64(id)] = n
			}
			return m, nil
		case "elems":
			m := make(map[any]any, len(x.Elems))
			for id, e := range x.Elems {
				m[float64(id)] = e
			}
			return m, nil
		}
	case *Node:
		switch name {
		case "num":
			return float64(x.Num), nil
		case "coord":
			return fig2NumbersAsValues(x.Coord), nil
		case "weight":
			return fig2NumbersAsValues(x.Weight), nil
		case "load":
			return fig2NumbersAsValues(x.Load), nil
		case "conf":
			values := make([]any, len(x.Conf))
			for i, v := range x.Conf {
				values[i] = v
			}
			return values, nil
		case "frame":
			return x.Frame, nil
		case "disp", "reaction":
			m := make(map[any]any)
			values := x.Disp
			if name == "reaction" {
				values = x.Reaction
			}
			for p, a := range values {
				m[p] = fig2NumbersAsValues(a)
			}
			return m, nil
		}
	case *Elem:
		switch name {
		case "num":
			return float64(x.Num), nil
		case "sect":
			return float64(x.Sect.Num), nil
		case "etype":
			return ETYPES[x.Etype], nil
		case "enods":
			return float64(x.Enods), nil
		case "enod":
			out := make([]any, len(x.Enod))
			for i, n := range x.Enod {
				out[i] = float64(n.Num)
			}
			return out, nil
		case "rate":
			return fig2NumbersAsValues(x.MaxRate), nil
		case "cang":
			return x.Cang, nil
		case "frame":
			return x.Frame, nil
		case "length", "area", "mid_point", "originalsection":
			return fig2Callable(func(args []any) (any, error) {
				if len(args) != 0 {
					return nil, fmt.Errorf("%s takes no arguments", name)
				}
				switch name {
				case "length":
					return x.Length(), nil
				case "area":
					return x.Area(), nil
				case "mid_point":
					return fig2NumbersAsValues(x.MidPoint()), nil
				default:
					return float64(x.OriginalSection().Num), nil
				}
			}), nil
		case "N", "QX", "QY", "MT", "MX", "MY":
			return fig2Callable(func(args []any) (any, error) {
				var period string
				end := 0
				if len(args) > 0 {
					period, _ = args[0].(string)
				}
				if len(args) == 2 {
					n, ok := args[1].(float64)
					if !ok || (n != 0 && n != 1) {
						return nil, fmt.Errorf("invalid stress end")
					}
					end = int(n)
				}
				if !x.IsLineElem() || period == "" || len(args) > 2 {
					return nil, fmt.Errorf("expected line member stress method with period and optional end")
				}
				index := map[string]int{"N": 0, "QX": 1, "QY": 2, "MT": 3, "MX": 4, "MY": 5}[name]
				return x.ReturnStress(strings.ToUpper(period), end, index), nil
			}), nil
		}
	}
	return nil, fmt.Errorf("unsupported property %s on %T", name, v)
}
