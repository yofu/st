package st

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"golang.org/x/mobile/event/key"
)

// CommandTextInput leaves text widgets to the driver. The buffered result and
// quit channel let a dialog be dismissed or replaced without blocking rendering.
type CommandText struct {
	Text      string
	Cancelled bool
}
type CommandTextInput interface {
	CommandPrompt(title, initial string) chan CommandText
}

var geometryCancelled = errors.New("command cancelled")

type geometryCommand struct {
	w          Commander
	scope      InteractiveScope
	quit       chan bool
	node       chan *Node
	click      chan Click
	key        chan Key
	nodeDigits string
}

func runGeometryCommand(w Commander, f func(*geometryCommand) error) chan bool {
	quit := make(chan bool)
	go func() {
		scope := commandScope(w)
		defer scope.Close()
		c := &geometryCommand{w: w, scope: scope, quit: quit, node: w.GetNode(), click: w.GetClick(), key: w.GetKey()}
		alt := w.AltSelectNode()
		w.SetAltSelectNode(false)
		defer func() { w.SetAltSelectNode(alt); w.EndTail(); w.EndCommand(); w.Redraw() }()
		if err := f(c); err != nil && !errors.Is(err, geometryCancelled) {
			ErrorMessage(w, err, ERROR)
		}
	}()
	return quit
}
func (c *geometryCommand) event() (*Node, *Key, error) {
	c.scope.Wait(c.node, c.key, c.click, c.quit)
	select {
	case n := <-c.node:
		c.scope.Resume()
		return n, nil, nil
	case k := <-c.key:
		c.scope.Resume()
		if k.Direction == key.DirRelease {
			if k.Rune >= '0' && k.Rune <= '9' {
				c.nodeDigits += string(k.Rune)
				c.w.History("NODE: " + c.nodeDigits)
				return nil, nil, nil
			}
			if k.Code == key.CodeDeleteBackspace && c.nodeDigits != "" {
				c.nodeDigits = c.nodeDigits[:len(c.nodeDigits)-1]
				return nil, nil, nil
			}
			if k.Code == key.CodeReturnEnter && c.nodeDigits != "" {
				num, _ := strconv.Atoi(c.nodeDigits)
				c.nodeDigits = ""
				n := c.w.Frame().Nodes[num]
				if n == nil || n.IsHidden(c.w.Frame().Show) {
					c.w.History("節点が見つかりません")
					return nil, nil, nil
				}
				return n, nil, nil
			}
		}
		return nil, &k, nil
	case click := <-c.click:
		c.scope.Resume()
		if click.Button == ButtonRight {
			return nil, nil, geometryCancelled
		}
		return nil, nil, nil
	case <-c.quit:
		c.scope.Resume()
		return nil, nil, geometryCancelled
	}
}
func (c *geometryCommand) nodes(count int, onKey func(Key) error) ([]*Node, error) {
	c.nodeDigits = ""
	ns := make([]*Node, 0, count)
	c.w.DeselectNode()
	c.w.EndTail()
	for len(ns) < count {
		n, k, err := c.event()
		if err != nil {
			return nil, err
		}
		if k != nil && k.Direction == key.DirRelease {
			if k.Code == key.CodeDeleteBackspace || k.Code == key.CodeDeleteForward {
				if len(ns) > 0 {
					ns = ns[:len(ns)-1]
					c.w.EndTail()
					for _, n := range ns {
						c.w.AddTail(n)
					}
					c.w.SelectNode(ns)
					c.w.Redraw()
				}
			} else if onKey != nil {
				if err := onKey(*k); err != nil {
					return nil, err
				}
			}
		}
		if n == nil {
			continue
		}
		duplicate := false
		for _, prev := range ns {
			if n == prev {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}
		ns = append(ns, n)
		c.w.SelectNode(append([]*Node(nil), ns...))
		c.w.AddTail(n)
		c.w.Redraw()
	}
	return ns, nil
}
func (c *geometryCommand) prompt(title, initial string) (string, error) {
	host, ok := c.w.(CommandTextInput)
	if !ok {
		return "", fmt.Errorf("このドライバは文字入力に対応していません")
	}
	result := host.CommandPrompt(title, initial)
	c.scope.Wait(result, c.quit)
	select {
	case value := <-result:
		c.scope.Resume()
		if value.Cancelled {
			return "", geometryCancelled
		}
		return strings.TrimSpace(value.Text), nil
	case <-c.quit:
		c.scope.Resume()
		return "", geometryCancelled
	}
}
func finiteNumber(text string) (float64, error) {
	v, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("数値が不正です: %s", text)
	}
	return v, nil
}
func (c *geometryCommand) number(title, initial string) (float64, error) {
	for {
		text, err := c.prompt(title, initial)
		if err != nil {
			return 0, err
		}
		v, err := finiteNumber(text)
		if err == nil {
			return v, nil
		}
		c.w.History(err.Error())
		initial = text
	}
}
func parseCommandVector(text string) ([]float64, error) {
	parts := strings.FieldsFunc(text, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' || r == '、' })
	if len(parts) != 3 {
		return nil, fmt.Errorf("X,Y,Zの3成分を入力してください")
	}
	v := make([]float64, 3)
	for i, p := range parts {
		var err error
		v[i], err = finiteNumber(p)
		if err != nil {
			return nil, err
		}
	}
	return v, nil
}
func (c *geometryCommand) vectorPrompt(title, initial string) ([]float64, error) {
	for {
		text, err := c.prompt(title, initial)
		if err != nil {
			return nil, err
		}
		v, err := parseCommandVector(text)
		if err == nil {
			return v, nil
		}
		c.w.History(err.Error())
		initial = text
	}
}
func validVector(v []float64) bool {
	if len(v) != 3 {
		return false
	}
	size := Dot(v, v, 3)
	return size > 1e-24 && !math.IsNaN(size) && !math.IsInf(size, 0)
}
func keyLetter(k Key) string { return strings.ToUpper(string(k.Rune)) }
func (c *geometryCommand) vector(onKey func(Key)) ([]float64, error) {
	c.w.DeselectNode()
	c.w.EndTail()
	var base *Node
	for {
		n, k, err := c.event()
		if err != nil {
			return nil, err
		}
		var v []float64
		if k != nil && k.Direction == key.DirRelease {
			if onKey != nil {
				onKey(*k)
			}
			switch keyLetter(*k) {
			case "X":
				v = []float64{1, 0, 0}
			case "Y":
				v = []float64{0, 1, 0}
			case "Z":
				v = []float64{0, 0, 1}
			case "D":
				title := "方向ベクトル X,Y,Z"
				if base != nil {
					title = "終点座標 X,Y,Z"
				}
				v, err = c.vectorPrompt(title, "0,0,1")
				if err != nil {
					return nil, err
				}
				if base != nil {
					for i := range v {
						v[i] -= base.Coord[i]
					}
				}
			}
		}
		if n != nil {
			if base == nil {
				base = n
				c.w.SelectNode([]*Node{n})
				c.w.AddTail(n)
			} else {
				v = Direction(base, n, false)
			}
		}
		if v != nil {
			if !validVector(v) {
				c.w.History("長さが0の方向は指定できません")
				continue
			}
			c.w.EndTail()
			return v, nil
		}
	}
}
func (c *geometryCommand) center() ([]float64, error) {
	c.w.DeselectNode()
	c.w.EndTail()
	for {
		n, k, err := c.event()
		if err != nil {
			return nil, err
		}
		if n != nil {
			return append([]float64(nil), n.Coord...), nil
		}
		if k != nil && k.Direction == key.DirRelease && keyLetter(*k) == "D" {
			return c.vectorPrompt("中心座標 X,Y,Z", "0,0,0")
		}
	}
}
func editableCommandNodes(w Commander, includeElems bool) []*Node {
	ns := append([]*Node(nil), w.SelectedNodes()...)
	if includeElems {
		for _, el := range w.SelectedElems() {
			if el != nil && !el.Lock && !el.IsHidden(w.Frame().Show) {
				ns = append(ns, el.Enod...)
			}
		}
	}
	seen := make(map[*Node]bool)
	result := make([]*Node, 0, len(ns))
	for _, n := range ns {
		if n == nil || seen[n] || n.Lock || n.IsHidden(w.Frame().Show) {
			continue
		}
		seen[n] = true
		result = append(result, n)
	}
	return result
}

// Compute every result before committing, including when a target is one of
// the reference nodes. Failed geometry must not leave a partially edited frame.
func transformCommandNodes(w Commander, ns []*Node, transform func(*Node) error) error {
	result := make([][]float64, len(ns))
	for i, n := range ns {
		copy := *n
		copy.Coord = append([]float64(nil), n.Coord...)
		if err := transform(&copy); err != nil {
			return err
		}
		for _, x := range copy.Coord {
			if math.IsNaN(x) || math.IsInf(x, 0) {
				return fmt.Errorf("座標が有限値ではありません")
			}
		}
		result[i] = copy.Coord
	}
	for i, n := range ns {
		n.MoveTo(result[i][0], result[i][1], result[i][2])
	}
	if len(ns) > 0 {
		Snapshot(w)
	}
	return nil
}
func referenceNode(n *Node) *Node { return &Node{Coord: append([]float64(nil), n.Coord...)} }

func MoveToLine(w Commander) chan bool {
	ns := editableCommandNodes(w, false)
	return runGeometryCommand(w, func(c *geometryCommand) error {
		if len(ns) == 0 {
			return fmt.Errorf("移動する節点がありません")
		}
		fixed := 0
		w.History("直線の2節点を指定。X/Y/Zで固定座標を切替（既定X）")
		refs, err := c.nodes(2, func(k Key) error {
			switch keyLetter(k) {
			case "X":
				fixed = 0
			case "Y":
				fixed = 1
			case "Z":
				fixed = 2
			}
			return nil
		})
		if err != nil {
			return err
		}
		a, b := referenceNode(refs[0]), referenceNode(refs[1])
		return transformCommandNodes(w, ns, func(n *Node) error { return n.MoveToLine(a, b, fixed) })
	})
}
func MoveToPlane(w Commander) chan bool {
	ns := editableCommandNodes(w, false)
	return runGeometryCommand(w, func(c *geometryCommand) error {
		if len(ns) == 0 {
			return fmt.Errorf("移動する節点がありません")
		}
		move := 2
		w.History("平面の3節点を指定。X/Y/Zで移動方向を切替（既定Z）")
		refs, err := c.nodes(3, func(k Key) error {
			switch keyLetter(k) {
			case "X":
				move = 0
			case "Y":
				move = 1
			case "Z":
				move = 2
			}
			return nil
		})
		if err != nil {
			return err
		}
		a, b, d := referenceNode(refs[0]), referenceNode(refs[1]), referenceNode(refs[2])
		return transformCommandNodes(w, ns, func(n *Node) error { return n.MoveToPlane(a, b, d, move) })
	})
}
func CreateMeasure(w Commander) chan bool {
	return runGeometryCommand(w, func(c *geometryCommand) error {
		w.History("寸法線の2節点を指定")
		refs, err := c.nodes(2, nil)
		if err != nil {
			return err
		}
		a, b := refs[0], refs[1]
		if !validVector(Direction(a, b, false)) {
			return fmt.Errorf("同じ位置の2節点は指定できません")
		}
		w.History("引き出し方向を指定[X/Y/Z/V]")
		for {
			// Freeze point input during the direction-only phase.
			c.scope.Wait(c.key, c.click, c.quit)
			var k Key
			select {
			case k = <-c.key:
				c.scope.Resume()
			case click := <-c.click:
				c.scope.Resume()
				if click.Button == ButtonRight {
					return geometryCancelled
				}
				continue
			case <-c.quit:
				c.scope.Resume()
				return geometryCancelled
			}
			if k.Direction != key.DirRelease {
				continue
			}
			var u, v []float64
			switch keyLetter(k) {
			case "X":
				u = []float64{1, 0, 0}
				v = []float64{0, 1, 0}
			case "Y":
				u = []float64{0, 1, 0}
				v = []float64{1, 0, 0}
			case "Z":
				u = []float64{0, 0, 1}
				v = Direction(a, b, true)
			case "V":
				v = Direction(a, b, true)
				u = Cross(v, ZAXIS)
			default:
				continue
			}
			if !validVector(u) {
				return fmt.Errorf("この2点では指定方向の寸法線を作成できません")
			}
			m := w.Frame().AddMeasure(append([]float64(nil), a.Coord...), append([]float64(nil), b.Coord...), u)
			m.Text = fmt.Sprintf("%.0f", VectorDistance(a, b, v)*1000)
			w.Frame().Show.Measure = true
			Snapshot(w)
			return nil
		}
	})
}
func CreateKijun(w Commander) chan bool {
	return runGeometryCommand(w, func(c *geometryCommand) error {
		w.History("基準線の2節点を指定")
		refs, err := c.nodes(2, nil)
		if err != nil {
			return err
		}
		if !validVector(Direction(refs[0], refs[1], false)) {
			return fmt.Errorf("基準線の長さが0です")
		}
		for {
			name, err := c.prompt("基準線名", "")
			if err != nil {
				return err
			}
			name = strings.ToLower(name)
			if name == "" {
				continue
			}
			_, err = w.Frame().AddKijun(name, append([]float64(nil), refs[0].Coord...), append([]float64(nil), refs[1].Coord...))
			if err != nil {
				w.History(err.Error())
				continue
			}
			w.Frame().Show.Kijun = true
			Snapshot(w)
			return nil
		}
	})
}
func AxisToCang(w Commander) chan bool {
	elems := append([]*Elem(nil), w.SelectedElems()...)
	return runGeometryCommand(w, func(c *geometryCommand) error {
		targets := make([]*Elem, 0, len(elems))
		for _, el := range elems {
			if el != nil && el.IsLineElem() && !el.Lock && !el.IsHidden(w.Frame().Show) {
				targets = append(targets, el)
			}
		}
		if len(targets) == 0 {
			return fmt.Errorf("部材軸を設定する線材がありません")
		}
		strong := true
		w.History("方向を2節点で指定。S:強軸 W:弱軸 T:切替 D:座標入力 X/Y/Z:軸方向")
		v, err := c.vector(func(k Key) {
			switch keyLetter(k) {
			case "S":
				strong = true
			case "W":
				strong = false
			case "T":
				strong = !strong
			}
		})
		if err != nil {
			return err
		}
		angles := make([]float64, len(targets))
		for i, el := range targets {
			axis := el.Direction(false)
			if !validVector(axis) || !validVector(Cross(Normalize(axis), Normalize(v))) {
				return fmt.Errorf("部材%dの軸と指定方向が平行です", el.Num)
			}
			copy := *el
			angles[i], err = copy.AxisToCang(v, strong)
			if err != nil || math.IsNaN(angles[i]) || math.IsInf(angles[i], 0) {
				return fmt.Errorf("部材%dの軸を設定できません", el.Num)
			}
		}
		for i, el := range targets {
			el.Cang = angles[i]
			el.SetPrincipalAxis()
		}
		Snapshot(w)
		return nil
	})
}
func RotateNodes(w Commander) chan bool {
	targets := editableCommandNodes(w, true)
	return runGeometryCommand(w, func(c *geometryCommand) error {
		if len(targets) == 0 {
			return fmt.Errorf("回転する節点がありません")
		}
		w.History("回転中心を指定。D:座標入力")
		center, err := c.center()
		if err != nil {
			return err
		}
		w.History("回転軸を2節点で指定。D:座標入力 X/Y/Z:軸方向")
		axis, err := c.vector(nil)
		if err != nil {
			return err
		}
		var angle float64
		initial := "0"
		for {
			text, err := c.prompt("回転角[度]（参照点から指定する場合はR）", initial)
			if err != nil {
				return err
			}
			if strings.EqualFold(text, "R") {
				w.History("参照点3点を指定[頂点・始点・終点]")
				refs, err := c.nodes(3, nil)
				if err != nil {
					return err
				}
				u := Normalize(axis)
				a, b := Direction(refs[0], refs[1], false), Direction(refs[0], refs[2], false)
				for i := 0; i < 3; i++ {
					a[i] -= u[i] * Dot(u, Direction(refs[0], refs[1], false), 3)
					b[i] -= u[i] * Dot(u, Direction(refs[0], refs[2], false), 3)
				}
				if !validVector(a) || !validVector(b) {
					return fmt.Errorf("参照方向が回転軸と平行です")
				}
				cos := Dot(Normalize(a), Normalize(b), 3)
				angle = math.Acos(math.Max(-1, math.Min(1, cos)))
				break
			}
			v, err := finiteNumber(text)
			if err != nil {
				w.History(err.Error())
				initial = text
				continue
			}
			angle = v * math.Pi / 180
			break
		}
		return transformCommandNodes(w, targets, func(n *Node) error { n.Rotate(center, axis, angle); return nil })
	})
}
func ScaleNodes(w Commander) chan bool {
	targets := editableCommandNodes(w, true)
	return runGeometryCommand(w, func(c *geometryCommand) error {
		if len(targets) == 0 {
			return fmt.Errorf("拡大縮小する節点がありません")
		}
		w.History("拡大縮小の中心を指定。D:座標入力")
		center, err := c.center()
		if err != nil {
			return err
		}
		scale, err := c.number("倍率", "1")
		if err != nil {
			return err
		}
		return transformCommandNodes(w, targets, func(n *Node) error { n.Scale(center, scale, scale, scale); return nil })
	})
}
