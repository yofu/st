package st

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"

	"golang.org/x/mobile/event/key"

	"github.com/atotto/clipboard"
	"github.com/yofu/dxf"
	"github.com/yofu/st/matrix"
)

var (
	Commands = map[string]func(Commander) chan bool{
		"DISTS":              Dists,
		"MATCHPROPERTY":      MatchProperty,
		"JOINLINEELEM":       JoinLineElem,
		"ERASE":              Erase,
		"ADDLINEELEM":        AddLineElem,
		"ADDARC":             AddArc,
		"ADDPLATEELEM":       AddPlateElem,
		"ADDPLATEELEMBYLINE": AddPlateElemByLine,
		"HATCHPLATEELEM":     HatchPlateElem,
		"COPYELEM":           CopyElem,
		"MOVENODE":           MoveNode,
		"MOVEELEM":           MoveElem,
		"MIRROR":             Mirror,
		"DIVIDEATPLANE":      DivideAtPlane,
		"ARRAYPOLAR":         ArrayPolar,
		"TRIM":               Trim,
		"EXTEND":             Extend,
		"OFFSET":             Offset,
		"MOVEUPDOWN":         MoveUpDown,
		"SPLINE":             Spline,
		"NOTICE1459":         Notice1459,
		"TOGGLEBOND":         ToggleBond,
		"COPYBOND":           CopyBond,
		"EDITPLATEELEM":      EditPlateElem,
	}
)

func Select(stw Commander) chan bool {
	quit := make(chan bool)
	go func() {
		nch := stw.GetNode()
		elch := stw.GetElem()
		mch := stw.GetModifier()
		shift := false
	sel:
		for {
			select {
			case n := <-nch:
				if n == nil {
					stw.DeselectNode()
				} else {
					if shift {
						RemoveSelection(stw, n)
					} else {
						AddSelection(stw, n)
					}
				}
			case el := <-elch:
				if el == nil {
					stw.DeselectElem()
				} else {
					if shift {
						RemoveSelection(stw, el)
					} else {
						AddSelection(stw, el)
					}
				}
			case m := <-mch:
				shift = m.Shift
			case <-quit:
				stw.EndCommand()
				break sel
			}
		}
	}()
	return quit
}

func onenode(stw Commander, f func(*Node) error) chan bool {
	quit := make(chan bool)
	go func() {
		nch := stw.GetNode()
		clickch := stw.GetClick()
		as := stw.AltSelectNode()
		stw.SetAltSelectNode(false)
	onenode:
		for {
			select {
			case n := <-nch:
				if n != nil {
					err := f(n)
					if err != nil {
						ErrorMessage(stw, err, ERROR)
					}
					stw.Redraw()
				}
			case c := <-clickch:
				if c.Button == ButtonRight {
					stw.SetAltSelectNode(as)
					stw.Deselect()
					stw.EndCommand()
					stw.Redraw()
					break onenode
				}
			case <-quit:
				stw.SetAltSelectNode(as)
				stw.EndCommand()
				stw.Redraw()
				break onenode
			}
		}
	}()
	return quit
}

func ToggleBond(stw Commander) chan bool {
	return onenode(stw, func(n *Node) error {
		for _, el := range stw.SelectedElems() {
			if el == nil {
				continue
			}
			el.ToggleBond(n.Num)
		}
		Snapshot(stw)
		return nil
	})
}

func ArrayPolar(stw Commander) chan bool {
	quit := make(chan bool)
	go func() {
		nch := stw.GetNode()
		ach := stw.GetAxis()
		keych := stw.GetKey()
		clickch := stw.GetClick()
		as := stw.AltSelectNode()
		stw.SetAltSelectNode(false)
		frame := stw.Frame()
		num := 0
		eps := stw.EPS()
	arraypolar:
		for {
			select {
			case n := <-nch:
				if n != nil {
					frame.LocalAxis = &Axis{
						Frame:  frame,
						Origin: n.Coord,
						Direction: [][]float64{
							[]float64{frame.Show.GlobalAxisSize, 0.0, 0.0},
							[]float64{0.0, frame.Show.GlobalAxisSize, 0.0},
							[]float64{0.0, 0.0, frame.Show.GlobalAxisSize},
						},
						Current: -1,
					}
					stw.SetAltSelectNode(as)
				}
			case <-ach:
			case k := <-keych:
				if frame.LocalAxis != nil {
					if k.Direction == key.DirRelease {
						switch k.Code {
						case key.CodeJ:
							num--
						case key.CodeK:
							num++
						case key.CodeReturnEnter:
							if num > 0 {
								dtheta := math.Pi / float64(num+1)
								theta := dtheta
								a := frame.LocalAxis
								for i := 0; i < num; i++ {
									for _, el := range stw.SelectedElems() {
										if el.IsNotEditable(frame.Show) {
											continue
										}
										el.CopyRotate(a.Origin, a.Direction[a.Current], theta, eps)
									}
									theta += dtheta
								}
								Snapshot(stw)
								frame.LocalAxis = nil
								stw.SetAltSelectNode(as)
								stw.Deselect()
								stw.EndCommand()
								stw.Redraw()
								break arraypolar
							}
						}
					}
				}
			case c := <-clickch:
				if c.Button == ButtonRight {
					frame.LocalAxis = nil
					stw.SetAltSelectNode(as)
					stw.Deselect()
					stw.EndCommand()
					stw.Redraw()
					break arraypolar
				}
			case <-quit:
				frame.LocalAxis = nil
				stw.SetAltSelectNode(as)
				stw.EndCommand()
				stw.Redraw()
				break arraypolar
			}
		}
	}()
	return quit
}

func twonodes(stw Commander, f func(*Node, *Node) (bool, error)) chan bool {
	quit := make(chan bool)
	go func() {
		var n0 *Node
		nch := stw.GetNode()
		clickch := stw.GetClick()
		as := stw.AltSelectNode()
		stw.SetAltSelectNode(false)
	twonodes:
		for {
			select {
			case n := <-nch:
				if n != nil {
					if !stw.NodeSelected() {
						stw.SelectNode([]*Node{n})
						n0 = n
						stw.AddTail(n0)
					} else {
						end, err := f(n0, n)
						if err != nil {
							ErrorMessage(stw, err, ERROR)
						}
						if end {
							stw.SetAltSelectNode(as)
							stw.DeselectNode()
							stw.EndTail()
							stw.EndCommand()
							stw.Redraw()
							break twonodes
						} else {
							stw.DeselectNode()
							stw.EndTail()
							stw.Redraw()
						}
					}
				}
			case c := <-clickch:
				if c.Button == ButtonRight {
					stw.SetAltSelectNode(as)
					stw.Deselect()
					stw.EndTail()
					stw.EndCommand()
					stw.Redraw()
					break twonodes
				}
			case <-quit:
				stw.SetAltSelectNode(as)
				stw.EndTail()
				stw.EndCommand()
				stw.Redraw()
				break twonodes
			}
		}
	}()
	return quit
}

func Dists(stw Commander) chan bool {
	return twonodes(stw, func(n0, n *Node) (bool, error) {
		dx, dy, dz, d := stw.Frame().Distance(n0, n)
		stw.History(fmt.Sprintf("NODE: %d - %d", n0.Num, n.Num))
		stw.History(fmt.Sprintf("DX: %.3f DY: %.3f DZ: %.3f D: %.3f", dx, dy, dz, d))
		return true, nil
	})
}

func AddLineElem(stw Commander) chan bool {
	return twonodes(stw, func(n0, n *Node) (bool, error) {
		frame := stw.Frame()
		sec := frame.DefaultSect()
		el := frame.AddLineElem(-1, []*Node{n0, n}, sec, NULL)
		stw.History(fmt.Sprintf("ELEM: %d (ENOD: %d - %d, SECT: %d)", el.Num, n0.Num, n.Num, sec.Num))
		Snapshot(stw)
		return true, nil
	})
}

func EditPlateElem(stw Commander) chan bool {
	return twonodes(stw, func(n0, n *Node) (bool, error) {
		if n0 == n {
			return false, nil
		}
		for _, el := range stw.SelectedElems() {
			for i, en := range el.Enod {
				if en == n0 {
					el.Enod[i] = n
					break
				}
			}
		}
		Snapshot(stw)
		return false, nil
	})
}

func multinodes(stw Commander, f func([]*Node) error, each bool) chan bool {
	quit := make(chan bool)
	go func() {
		nodes := make([]*Node, 0)
		nch := stw.GetNode()
		clickch := stw.GetClick()
		keych := stw.GetKey()
		var wheelch chan Wheel
		as := stw.AltSelectNode()
		stw.SetAltSelectNode(false)
		delta := 1000.0
		dir := ""
		diff := map[string]float64{"X": 0.0, "Y": 0.0, "Z": 0.0}
		bydiff := func() {
			ns := stw.SelectedNodes()
			nodes = []*Node{
				&Node{
					Coord: []float64{0.000,
						0.000,
						0.000,
					},
				},
				&Node{
					Coord: []float64{diff["X"] * 0.001,
						diff["Y"] * 0.001,
						diff["Z"] * 0.001,
					},
				},
			}
			err := f(nodes)
			if err != nil {
				ErrorMessage(stw, err, ERROR)
			}
			nodes = ns
		}
	multinodes:
		for {
			select {
			case n := <-nch:
				if n != nil {
					if each {
						if len(nodes) == 0 {
							nodes = []*Node{n}
							stw.AddTail(n)
						} else {
							nodes = []*Node{nodes[0], n}
						}
						err := f(nodes)
						if err != nil {
							ErrorMessage(stw, err, ERROR)
						}
						stw.Redraw()
					} else {
						nodes = append(nodes, n)
						AddSelection(stw, n)
						stw.AddTail(n)
						stw.Redraw()
					}
				}
			case c := <-clickch:
				switch c.Button {
				case ButtonRight:
					if !each {
						if len(nodes) > 0 {
							err := f(nodes)
							if err != nil {
								ErrorMessage(stw, err, ERROR)
							}
						}
					}
					stw.SetAltSelectNode(as)
					stw.Deselect()
					stw.EndTail()
					stw.EndCommand()
					stw.Redraw()
					break multinodes
				case ButtonLeft:
					if each && dir != "" {
						bydiff()
					}
				}
			case k := <-keych:
				if k.Direction == key.DirRelease {
					switch k.Code {
					case key.CodeA, key.CodeH:
						delta *= 10.0
					case key.CodeS, key.CodeL:
						delta *= 0.1
					case key.CodeJ:
						diff[dir] -= delta
						stw.History(fmt.Sprintf("%s=%.3f\r", dir, diff[dir]))
					case key.CodeK:
						diff[dir] += delta
						stw.History(fmt.Sprintf("%s=%.3f\r", dir, diff[dir]))
					case key.CodeC:
						for _, d := range []string{"X", "Y", "Z"} {
							diff[d] = 0.0
						}
					case key.CodeReturnEnter:
						if each && dir != "" {
							bydiff()
							stw.Redraw()
						}
					case key.CodeX, key.CodeY, key.CodeZ:
						if wheelch == nil {
							wheelch = stw.GetWheel()
						}
						if dir != "" {
							delta = 1000.0
						}
						switch k.Code {
						case key.CodeX:
							dir = "X"
						case key.CodeY:
							dir = "Y"
						case key.CodeZ:
							dir = "Z"
						}
					}
				}
			case w := <-wheelch:
				if w == WheelUp {
					diff[dir] += delta
				} else {
					diff[dir] -= delta
				}
				stw.History(fmt.Sprintf("%s=%.3f\r", dir, diff[dir]))
			case <-quit:
				stw.SetAltSelectNode(as)
				stw.EndTail()
				stw.EndCommand()
				stw.Redraw()
				break multinodes
			}
		}
	}()
	return quit
}

func AddPlateElem(stw Commander) chan bool {
	return multinodes(stw, func(ns []*Node) error {
		if len(ns) < 3 {
			return fmt.Errorf("too few nodes")
		}
		frame := stw.Frame()
		sec := frame.DefaultSect()
		el := frame.AddPlateElem(-1, ns, sec, NULL)
		var buf bytes.Buffer
		buf.WriteString(fmt.Sprintf("ELEM: %d (ENOD: ", el.Num))
		for _, n := range ns {
			buf.WriteString(fmt.Sprintf("%d ", n.Num))
		}
		buf.WriteString(fmt.Sprintf(", SECT: %d)", sec.Num))
		stw.History(buf.String())
		Snapshot(stw)
		return nil
	}, false)
}

func AddArc(stw Commander) chan bool {
	return multinodes(stw, func(ns []*Node) error {
		if len(ns) < 3 {
			return fmt.Errorf("too few nodes")
		}
		frame := stw.Frame()
		eps := stw.EPS()
		coords := make([][]float64, 3)
		for i := 0; i < 3; i++ {
			coords[i] = make([]float64, 3)
			for j := 0; j < 3; j++ {
				coords[i][j] = ns[i].Coord[j]
			}
		}
		frame.AddArc(coords, eps)
		Snapshot(stw)
		return nil
	}, false)
}

func CopyElem(stw Commander) chan bool {
	return multinodes(stw, func(ns []*Node) error {
		if len(ns) < 2 {
			return nil
		}
		frame := stw.Frame()
		vec := Direction(ns[0], ns[len(ns)-1], false)
		if !(vec[0] == 0.0 && vec[1] == 0.0 && vec[2] == 0.0) {
			eps := stw.EPS()
			for _, el := range stw.SelectedElems() {
				if el == nil || el.IsHidden(frame.Show) || el.Lock {
					continue
				}
				el.Copy(vec[0], vec[1], vec[2], eps)
			}
			Snapshot(stw)
		}
		return nil
	}, true)
}

func MoveNode(stw Commander) chan bool {
	return multinodes(stw, func(ns []*Node) error {
		if len(ns) < 2 {
			return nil
		}
		frame := stw.Frame()
		vec := Direction(ns[0], ns[len(ns)-1], false)
		if !(vec[0] == 0.0 && vec[1] == 0.0 && vec[2] == 0.0) {
			for _, n := range frame.ElemToNode(stw.SelectedElems()...) {
				AddSelection(stw, n)
			}
			if !stw.NodeSelected() {
				return nil
			}
			for _, n := range stw.SelectedNodes() {
				if n == nil || n.IsHidden(frame.Show) || n.Lock {
					continue
				}
				n.Move(vec[0], vec[1], vec[2])
			}
			Snapshot(stw)
		}
		return nil
	}, true)
}

func MoveElem(stw Commander) chan bool {
	return multinodes(stw, func(ns []*Node) error {
		if len(ns) < 2 {
			return nil
		}
		frame := stw.Frame()
		vec := Direction(ns[0], ns[len(ns)-1], false)
		if !(vec[0] == 0.0 && vec[1] == 0.0 && vec[2] == 0.0) {
			eps := stw.EPS()
			for _, el := range stw.SelectedElems() {
				if el == nil || el.IsHidden(frame.Show) || el.Lock {
					continue
				}
				el.Move(vec[0], vec[1], vec[2], eps)
			}
			Snapshot(stw)
		}
		return nil
	}, true)
}

func Mirror(stw Commander) chan bool {
	return multinodes(stw, func(ns []*Node) error {
		if len(ns) < 3 {
			return nil
		}
		frame := stw.Frame()
		eps := stw.EPS()
		v1 := make([]float64, 3)
		v2 := make([]float64, 3)
		for i := 0; i < 3; i++ {
			v1[i] = ns[1].Coord[i] - ns[0].Coord[i]
			v2[i] = ns[2].Coord[i] - ns[0].Coord[i]
		}
		vec := Cross(v1, v2)
		nmap := make(map[int]*Node, len(ns))
		nodes := frame.ElemToNode(stw.SelectedElems()...)
		for _, n := range nodes {
			c := n.MirrorCoord(ns[0].Coord, vec)
			var created bool
			nmap[n.Num], created = frame.CoordNode(c[0], c[1], c[2], eps)
			if created {
				for i := 0; i < 6; i++ {
					nmap[n.Num].Conf[i] = n.Conf[i]
				}
			}
		}
		for _, el := range stw.SelectedElems() {
			newenod := make([]*Node, el.Enods)
			var add bool
			for i := 0; i < el.Enods; i++ {
				newenod[i] = nmap[el.Enod[i].Num]
				if !add && newenod[i] != el.Enod[i] {
					add = true
				}
			}
			if add {
				if el.IsLineElem() {
					e := frame.AddLineElem(-1, newenod, el.Sect, el.Etype)
					for i := 0; i < 6*el.Enods; i++ {
						e.Bonds[i] = el.Bonds[i]
					}
				} else {
					frame.AddPlateElem(-1, newenod, el.Sect, el.Etype)
				}
			}
		}
		Snapshot(stw)
		return nil
	}, false)
}

func DivideAtPlane(stw Commander) chan bool {
	return multinodes(stw, func(ns []*Node) error {
		if len(ns) < 3 {
			return nil
		}
		eps := stw.EPS()
		v1 := make([]float64, 3)
		v2 := make([]float64, 3)
		for i := 0; i < 3; i++ {
			v1[i] = ns[1].Coord[i] - ns[0].Coord[i]
			v2[i] = ns[2].Coord[i] - ns[0].Coord[i]
		}
		vec := Cross(v1, v2)
		n := Normalize(vec)
		h := Dot(n, ns[0].Coord, 3)
		for _, el := range stw.SelectedElems() {
			if !el.IsLineElem() {
				continue
			}
			d := el.Direction(true)
			l := el.Length()
			dot := Dot(d, n, 3)
			if dot == 0 {
				continue
			}
			val := Dot(n, el.Enod[0].Coord, 3)
			val2 := (h - val)/dot
			if val2 <= 0 || val2 >= l {
				continue
			}
			coord := make([]float64, 3)
			for i := 0; i < 3; i++ {
				coord[i] = el.Enod[0].Coord[i] + val2*d[i]
			}
			el.DivideAtCoord(coord[0], coord[1], coord[2], eps)
		}
		Snapshot(stw)
		return nil
	}, false)
}

func Notice1459(stw Commander) chan bool {
	return multinodes(stw, func(ns []*Node) error {
		if len(ns) < 2 {
			return fmt.Errorf("too few nodes")
		}
		var otp bytes.Buffer
		frame := stw.Frame()
		var delta float64
		ds := make([]float64, len(ns))
		for i, n := range ns {
			ds[i] = -n.ReturnDisp(frame.Show.Period, 2) * 100
		}
		var length float64
		switch len(ns) {
		default:
			return nil
		case 2:
			delta = ds[1] - ds[0]
			stw.History(fmt.Sprintf("Disp: %.3f - %.3f [cm]", ds[1], ds[0]))
			for i := 0; i < 2; i++ {
				length += math.Pow(ns[1].Coord[i]-ns[0].Coord[i], 2)
			}
			length = math.Sqrt(length) * 100
			otp.WriteString(fmt.Sprintf("梁G1(部材  、断面  、節点 %d)\nFL 、X通りY通り\n最大たわみ　δ= %.3f - %.3f = %.3f [cm]\n長さL= %.1f [cm]\n変形増大係数α= 1\nα×δ/L=1×%.3f/ %.3f = 1/%d\n", ns[1].Num, ds[1], ds[0], delta, length, delta, length, math.Abs(length/delta)))
			clipboard.WriteAll(otp.String())
		case 3:
			delta = ds[1] - 0.5*(ds[0]+ds[2])
			stw.History(fmt.Sprintf("Disp: %.3f - (%.3f + %.3f)/2 [cm]", ds[1], ds[0], ds[2]))
			for i := 0; i < 2; i++ {
				length += math.Pow(ns[2].Coord[i]-ns[0].Coord[i], 2)
			}
			length = math.Sqrt(length) * 100
			otp.WriteString(fmt.Sprintf("梁G1(部材  、断面  、節点 %d)\nFL 、X通りY通り\n最大たわみ　δ= %.3f - (%.3f + %.3f)/2 = %.3f [cm]\n長さL= %.1f [cm]\n変形増大係数α= 1\nα×δ/L=1×%.3f/ %.3f = 1/%d\n", ns[1].Num, ds[1], ds[0], ds[2], delta, length, delta, length, math.Abs(length/delta)))
			clipboard.WriteAll(otp.String())
		}
		if delta != 0.0 {
			stw.History(fmt.Sprintf("Distance: %.3f[cm]", length))
			stw.History(fmt.Sprintf("Slope: 1/%.1f", math.Abs(length/delta)))
		}
		return nil
	}, false)
}

func multielems(stw Commander, cond func(*Elem) bool, f func([]*Elem) error) chan bool {
	if stw.ElemSelected() {
		f(stw.SelectedElems())
		stw.EndCommand()
		return nil
	}
	quit := make(chan bool)
	go func() {
		elems := make([]*Elem, 0)
		elch := stw.GetElem()
		clickch := stw.GetClick()
	erase:
		for {
			select {
			case el := <-elch:
				if el != nil && cond(el) {
					elems = append(elems, el)
					AddSelection(stw, el)
					stw.Redraw()
				}
			case c := <-clickch:
				if c.Button == ButtonRight {
					if len(elems) > 0 {
						err := f(elems)
						if err != nil {
							ErrorMessage(stw, err, ERROR)
						}
					}
					stw.Deselect()
					stw.EndCommand()
					stw.Redraw()
					break erase
				}
			case <-quit:
				stw.EndCommand()
				stw.Redraw()
				break erase
			}
		}
	}()
	return quit
}

func JoinLineElem(stw Commander) chan bool {
	return multielems(stw, func(el *Elem) bool {
		return el.IsLineElem() && !el.Lock
	}, func(elems []*Elem) error {
		if len(elems) < 2 {
			return fmt.Errorf("too few elems")
		}
		frame := stw.Frame()
		err := frame.JoinLineElem(elems[0], elems[1], true, true)
		if err != nil {
			return err
		}
		Snapshot(stw)
		return nil
	})
}

func Erase(stw Commander) chan bool {
	return multielems(stw, func(el *Elem) bool {
		return !el.Lock
	}, func(elems []*Elem) error {
		frame := stw.Frame()
		for _, el := range elems {
			frame.DeleteElem(el.Num)
		}
		stw.Deselect()
		ns := frame.NodeNoReference()
		if len(ns) != 0 {
			for _, n := range ns {
				frame.DeleteNode(n.Num)
			}
		}
		Snapshot(stw)
		return nil
	})
}

func AddPlateElemByLine(stw Commander) chan bool {
	return multielems(stw, func(el *Elem) bool {
		return el.IsLineElem()
	}, func(elems []*Elem) error {
		frame := stw.Frame()
		if len(elems) < 2 {
			return fmt.Errorf("too few elems")
		}
		ns := make([]*Node, 4)
		ns[0] = elems[0].Enod[0]
		ns[1] = elems[0].Enod[1]
		_, cw1 := ClockWise(ns[0].Pcoord, ns[1].Pcoord, elems[1].Enod[0].Pcoord)
		_, cw2 := ClockWise(ns[0].Pcoord, elems[1].Enod[0].Pcoord, elems[1].Enod[1].Pcoord)
		if cw1 == cw2 {
			ns[2] = elems[1].Enod[0]
			ns[3] = elems[1].Enod[1]
		} else {
			ns[2] = elems[1].Enod[1]
			ns[3] = elems[1].Enod[0]
		}
		sec := frame.DefaultSect()
		el := frame.AddPlateElem(-1, ns, sec, NULL)
		var buf bytes.Buffer
		buf.WriteString(fmt.Sprintf("ELEM: %d (ENOD: ", el.Num))
		for _, n := range ns {
			buf.WriteString(fmt.Sprintf("%d ", n.Num))
		}
		buf.WriteString(fmt.Sprintf(", SECT: %d)", sec.Num))
		stw.History(buf.String())
		Snapshot(stw)
		return nil
	})
}

func HatchPlateElem(stw Commander) chan bool {
	quit := make(chan bool)
	createhatch := func(ns []*Node) error {
		en := ModifyEnod(ns)
		en = Upside(en)
		frame := stw.Frame()
		sec := frame.DefaultSect()
		switch len(en) {
		case 0, 1, 2:
			return fmt.Errorf("too few nodes")
		case 3, 4:
			if len(frame.SearchElem(en...)) != 0 {
				return fmt.Errorf("elem already exists")
			}
			frame.AddPlateElem(-1, en, sec, NULL)
			return nil
		default:
			return fmt.Errorf("too many nodes")
		}
	}
	go func() {
		clickch := stw.GetClick()
	hatchplateelem:
		for {
			select {
			case c := <-clickch:
				switch c.Button {
				case ButtonLeft:
					ns, _, err := stw.Frame().BoundedArea(float64(c.X), float64(c.Y), 100)
					if err != nil {
						ErrorMessage(stw, err, ERROR)
						stw.EndCommand()
						stw.Redraw()
						break hatchplateelem
					}
					err = createhatch(ns)
					if err != nil {
						ErrorMessage(stw, err, ERROR)
						stw.EndCommand()
						stw.Redraw()
						break hatchplateelem
					}
					stw.Redraw()
				case ButtonRight:
					Snapshot(stw)
					stw.EndCommand()
					stw.Redraw()
					break hatchplateelem
				}
			case <-quit:
				stw.EndCommand()
				stw.Redraw()
				break hatchplateelem
			}
		}
	}()
	return quit
}

func onemultielem(stw Commander, cond func(*Elem) bool, f func(Click, *Elem, *Elem), exitfunc func()) chan bool {
	quit := make(chan bool)
	var el0 *Elem
	if stw.ElemSelected() {
		for _, el := range stw.SelectedElems() {
			if cond(el) {
				el0 = el
				break
			}
		}
	}
	go func() {
		elch := stw.GetElem()
		clickch := stw.GetClick()
		if el0 == nil {
		trim_elem:
			for {
				select {
				case el := <-elch:
					if el != nil && cond(el) {
						el0 = el
						break trim_elem
					}
				case c := <-clickch:
					if c.Button == ButtonRight {
						stw.Deselect()
						stw.EndCommand()
						stw.Redraw()
						return
					}
				case <-quit:
					stw.EndCommand()
					stw.Redraw()
					return
				}
			}
		}
		AddSelection(stw, el0)
		stw.Redraw()
	trim_click:
		for {
			select {
			case c := <-clickch:
				switch c.Button {
				case ButtonLeft:
					el := <-elch
					if el == nil || !cond(el) {
						ErrorMessage(stw, fmt.Errorf("no elem"), ERROR)
					} else {
						f(c, el0, el)
						stw.Redraw()
					}
				case ButtonRight:
					exitfunc()
					stw.Deselect()
					Snapshot(stw)
					stw.EndCommand()
					stw.Redraw()
					break trim_click
				}
			case <-quit:
				exitfunc()
				stw.EndCommand()
				stw.Redraw()
				break trim_click
			}
		}
	}()
	return quit
}

func MatchProperty(stw Commander) chan bool {
	return onemultielem(stw, func(el *Elem) bool {
		return true
	}, func(c Click, el0 *Elem, el *Elem) {
		el.Sect = el0.Sect
		el.Etype = el0.Etype
	}, func() {})
}

func CopyBond(stw Commander) chan bool {
	return onemultielem(stw, func(el *Elem) bool {
		return el.IsLineElem()
	}, func(c Click, el0 *Elem, el *Elem) {
		for i := 0; i < 12; i++ {
			el.Bonds[i] = el0.Bonds[i]
		}
	}, func() {})
}

func Trim(stw Commander) chan bool {
	return onemultielem(stw, func(el *Elem) bool {
		return el.IsLineElem()
	}, func(c Click, el0 *Elem, el *Elem) {
		frame := stw.Frame()
		eps := stw.EPS()
		var err error
		if DotLine(el0.Enod[0].Pcoord[0], el0.Enod[0].Pcoord[1], el0.Enod[1].Pcoord[0], el0.Enod[1].Pcoord[1], float64(c.X), float64(c.Y))*DotLine(el0.Enod[0].Pcoord[0], el0.Enod[0].Pcoord[1], el0.Enod[1].Pcoord[0], el0.Enod[1].Pcoord[1], el.Enod[0].Pcoord[0], el.Enod[0].Pcoord[1]) < 0.0 {
			_, _, err = frame.Trim(el0, el, 1, eps)
		} else {
			_, _, err = frame.Trim(el0, el, -1, eps)
		}
		if err != nil {
			ErrorMessage(stw, err, ERROR)
		}
	}, func() {
		for _, el := range stw.SelectedElems() {
			el.DivideAtOns(stw.EPS())
			break
		}
	})
}

func Extend(stw Commander) chan bool {
	return onemultielem(stw, func(el *Elem) bool {
		return el.IsLineElem()
	}, func(c Click, el0 *Elem, el *Elem) {
		frame := stw.Frame()
		eps := stw.EPS()
		var err error
		_, _, err = frame.Extend(el0, el, eps)
		if err != nil {
			ErrorMessage(stw, err, ERROR)
		}
	}, func() {
		for _, el := range stw.SelectedElems() {
			el.DivideAtOns(stw.EPS())
			break
		}
	})
}

func getside(stw Commander, cond func(*Elem) bool, f func(*Elem, int, int, map[string]float64)) chan bool {
	quit := make(chan bool)
	var el0 *Elem
	if stw.ElemSelected() {
		for _, el := range stw.SelectedElems() {
			if cond(el) {
				el0 = el
				break
			}
		}
	}
	values := make(map[string]float64)
	go func() {
		elch := stw.GetElem()
		clickch := stw.GetClick()
		keych := stw.GetKey()
		var wheelch chan Wheel
		delta := 1000.0
		name := ""
		for {
			select {
			case el := <-elch:
				if el0 == nil {
					if el != nil && cond(el) {
						el0 = el
						AddSelection(stw, el0)
						stw.Redraw()
					}
				}
			case c := <-clickch:
				switch c.Button {
				case ButtonLeft:
					if el0 != nil {
						if wheelch != nil {
							stw.StopWheel()
							wheelch = nil
						}
						f(el0, c.X, c.Y, values)
						Snapshot(stw)
						stw.Redraw()
						stw.Deselect()
						el0 = nil
					}
				case ButtonRight:
					stw.Deselect()
					stw.EndCommand()
					stw.Redraw()
					return
				}
			case k := <-keych:
				if k.Direction == key.DirRelease {
					switch k.Code {
					default:
						if name != "" {
							delta = 1000.0
						}
						name = strings.ToUpper(string(k.Rune))
						if wheelch == nil {
							wheelch = stw.GetWheel()
						}
					case key.CodeA, key.CodeH:
						delta *= 10.0
					case key.CodeS, key.CodeL:
						delta *= 0.1
					case key.CodeJ:
						if name != "" {
							values[name] -= delta
							stw.History(fmt.Sprintf("%s=%.3f\r", name, values[name]))
						}
					case key.CodeK:
						if name != "" {
							values[name] += delta
							stw.History(fmt.Sprintf("%s=%.3f\r", name, values[name]))
						}
					case key.CodeC:
						if name != "" {
							values[name] = 0.0
							stw.History(fmt.Sprintf("%s=%.3f\r", name, values[name]))
						}
					}
				}
			case w := <-wheelch:
				if name != "" {
					if w == WheelUp {
						values[name] += delta
					} else {
						values[name] -= delta
					}
					stw.History(fmt.Sprintf("%s=%.3f\r", name, values[name]))
				}
			case <-quit:
				stw.EndCommand()
				stw.Redraw()
				return
			}
		}
	}()
	return quit
}

func Offset(stw Commander) chan bool {
	return getside(stw, func(el *Elem) bool {
		return el.IsLineElem()
	}, func(el *Elem, x, y int, val map[string]float64) {
		if val["V"] == 0.0 {
			return
		}
		value := val["V"] * 0.001
		angle := val["T"] * math.Pi / 180.0
		frame := stw.Frame()
		mid := el.MidPoint()
		c := math.Cos(angle)
		s := math.Sin(angle)
		for i := 0; i < 3; i++ {
			mid[i] += el.Strong[i]*c + el.Weak[i]*s
		}
		st1 := frame.View.ProjectCoord(mid)
		if DotLine(el.Enod[0].Pcoord[0], el.Enod[0].Pcoord[1], el.Enod[1].Pcoord[0], el.Enod[1].Pcoord[1], float64(x), float64(y))*DotLine(el.Enod[0].Pcoord[0], el.Enod[0].Pcoord[1], el.Enod[1].Pcoord[0], el.Enod[1].Pcoord[1], st1[0], st1[1]) < 0.0 {
			el.Offset(-value, angle, stw.EPS())
		} else {
			el.Offset(value, angle, stw.EPS())
		}
		Snapshot(stw)
	})
}

func MoveUpDown(stw Commander) chan bool {
	quit := make(chan bool)
	frame := stw.Frame()
	mv := func(n0 *Node, r, m float64) {
		var dx, dy, d float64
		for _, n := range frame.Nodes {
			if n.IsHidden(frame.Show) || n.Lock {
				continue
			}
			dx = math.Abs(n.Coord[0] - n0.Coord[0])
			if dx <= 2*r {
				dy = math.Abs(n.Coord[1] - n0.Coord[1])
				if dy <= 2*r {
					d = math.Sqrt(math.Pow(dx, 2.0) + math.Pow(dy, 2.0))
					if d <= r {
						n.Coord[2] += -0.5*m*math.Pow(d, 2.0)/math.Pow(r, 3.0) + m/r
					} else if d <= 2*r {
						n.Coord[2] += m/d - 0.5*m/r
					}
				}
			}
		}
	}
	go func() {
		clickch := stw.GetClick()
		wheelch := stw.GetWheel()
	moveupdown:
		for {
			select {
			case w := <-wheelch:
				var val float64
				switch w {
				case WheelUp:
					val = 0.05
				case WheelDown:
					val = -0.05
				}
				pos := stw.CurrentPointerPosition()
				ns, picked := PickNode(stw, pos[0], pos[1], pos[0], pos[1])
				if picked {
					mv(ns[0], 0.25+rand.Float64()*0.5, val)
				}
				stw.Redraw()
			case c := <-clickch:
				switch c.Button {
				case ButtonRight:
					Snapshot(stw)
					stw.EndCommand()
					stw.Redraw()
					Snapshot(stw)
					break moveupdown
				}
			case <-quit:
				stw.EndCommand()
				stw.Redraw()
				Snapshot(stw)
				break moveupdown
			}
		}
	}()
	return quit
}

func createspline(fn string, nodes []*Node, d, z int, scale float64, ndiv int, original bool) error {
	A, B, C, D, h, err := splinecoefficient(nodes, d, z)
	if err != nil {
		return err
	}
	dw := dxf.NewDrawing()
	dw.AddLayer("ORIGINAL", dxf.DefaultColor, dxf.DefaultLineType, false)
	dw.AddLayer("SPLINE", dxf.DefaultColor, dxf.DefaultLineType, true)
	mind := nodes[0].Coord[d]
	maxd := nodes[0].Coord[d]
	minz := nodes[0].Coord[z]
	maxz := nodes[0].Coord[z]
	for i := 0; i < len(nodes)-1; i++ {
		if nodes[i+1].Coord[d] > maxd {
			maxd = nodes[i+1].Coord[d]
		} else if nodes[i+1].Coord[d] < mind {
			mind = nodes[i+1].Coord[d]
		}
		if nodes[i+1].Coord[z] > maxz {
			maxz = nodes[i+1].Coord[z]
		} else if nodes[i+1].Coord[z] < minz {
			minz = nodes[i+1].Coord[z]
		}
		if nodes[i+1].Coord[d]-nodes[i].Coord[d] <= 0.15 {
			dw.Line(nodes[i].Coord[d]*scale, nodes[i].Coord[z]*scale, 0.0, nodes[i+1].Coord[d]*scale, nodes[i+1].Coord[z]*scale, 0.0)
			continue
		}
		if original {
			dw.Layer("ORIGINAL", true)
			dw.Line(nodes[i].Coord[d]*scale, nodes[i].Coord[z]*scale, 0.0, nodes[i+1].Coord[d]*scale, nodes[i+1].Coord[z]*scale, 0.0)
			dw.Layer("SPLINE", true)
		}
		sx := nodes[i].Coord[d]
		dcx := h[i] / float64(ndiv)
		ex := sx + dcx
		dx := ex - nodes[i].Coord[d]
		sy := A[i]
		ey := A[i] + B[i]*dx + C[i]*dx*dx + D[i]*dx*dx*dx
		for j := 0; j < ndiv; j++ {
			if ey > maxz {
				maxz = ey
			} else if ey < minz {
				minz = ey
			}
			dw.Line(sx*scale, sy*scale, 0.0, ex*scale, ey*scale, 0.0)
			sx = ex
			sy = ey
			ex = ex + dcx
			dx = ex - nodes[i].Coord[d]
			ey = A[i] + B[i]*dx + C[i]*dx*dx + D[i]*dx*dx*dx
		}
	}
	dw.AddLayer("BOUNDARY", dxf.DefaultColor, dxf.DefaultLineType, true)
	dw.Line(mind*scale, minz*scale, 0.0, maxd*scale, minz*scale, 0.0)
	dw.Line(mind*scale, maxz*scale, 0.0, maxd*scale, maxz*scale, 0.0)
	return dw.SaveAs(fn)
}

func splinecoefficient(nodes []*Node, d, z int) ([]float64, []float64, []float64, []float64, []float64, error) {
	if len(nodes) == 0 {
		return nil, nil, nil, nil, nil, fmt.Errorf("no nodes")
	}
	switch d {
	case 0:
		sort.Sort(NodeByXCoord{nodes})
	case 1:
		sort.Sort(NodeByYCoord{nodes})
	case 2:
		sort.Sort(NodeByZCoord{nodes})
	}
	h := make([]float64, len(nodes)-1)
	y := make([]float64, len(nodes)-1)
	for i := 0; i < len(nodes)-1; i++ {
		h[i] = nodes[i+1].Coord[d] - nodes[i].Coord[d]
		y[i] = (nodes[i+1].Coord[z] - nodes[i].Coord[z]) / h[i]
	}
	m := matrix.NewCOOMatrix(len(nodes) - 2)
	v := make([]float64, len(nodes)-2)
	for i := 0; i < len(nodes)-2; i++ {
		if i > 0 {
			m.Set(i-1, i, h[i])
		}
		m.Set(i, i, 2.0*(h[i]+h[i+1]))
		if i < len(nodes)-3 {
			m.Set(i, i+1, h[i+1])
		}
		v[i] = 3.0 * (y[i+1] - y[i])
	}
	conf := make([]bool, m.Size)
	lls := m.ToLLS(0, conf)
	ans, err := lls.Solve(nil, v)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	C := make([]float64, len(nodes))
	for i := 0; i < len(nodes)-2; i++ {
		C[i+1] = ans[0][i]
	}
	D := make([]float64, len(nodes)-1)
	B := make([]float64, len(nodes)-1)
	A := make([]float64, len(nodes)-1)
	for i := 0; i < len(nodes)-1; i++ {
		D[i] = (C[i+1] - C[i]) / (3.0 * h[i])
		B[i] = y[i] - (C[i]+D[i]*h[i])*h[i]
		A[i] = nodes[i].Coord[z]
	}
	return A, B, C, D, h, nil
}

func splinecoord(nodes []*Node, d, z int, ndiv int) ([][]float64, error) {
	A, B, C, D, h, err := splinecoefficient(nodes, d, z)
	if err != nil {
		return nil, err
	}
	coords := make([][]float64, ndiv*(len(nodes)-1)+1)
	var c int
	for i := 0; i < 3; i++ {
		if i == d {
			continue
		} else if i == z {
			continue
		} else {
			c = i
			break
		}
	}
	ind := 0
	coords[ind] = make([]float64, 3)
	coords[ind][d] = nodes[0].Coord[d]
	coords[ind][c] = nodes[0].Coord[c]
	coords[ind][z] = nodes[0].Coord[z]
	ind++
	for i := 0; i < len(nodes)-1; i++ {
		sx := nodes[i].Coord[d]
		dcx := h[i] / float64(ndiv)
		ex := sx + dcx
		dx := ex - nodes[i].Coord[d]
		sy := A[i]
		ey := sy + B[i]*dx + C[i]*dx*dx + D[i]*dx*dx*dx
		coords[ind] = make([]float64, 3)
		coords[ind][d] = ex
		coords[ind][c] = nodes[i].Coord[c]
		coords[ind][z] = ey
		ind++
		for j := 0; j < ndiv-1; j++ {
			sx = ex
			sy = ey
			ex = ex + dcx
			dx = ex - nodes[i].Coord[d]
			ey = A[i] + B[i]*dx + C[i]*dx*dx + D[i]*dx*dx*dx
			coords[ind] = make([]float64, 3)
			coords[ind][d] = ex
			coords[ind][c] = nodes[i].Coord[c]
			coords[ind][z] = ey
			ind++
		}
	}
	return coords, nil
}

func Spline(stw Commander) chan bool {
	d := 1
	z := 2
	scale := 1000.0
	ndiv := 4
	original := true
	return multielems(stw, func(el *Elem) bool {
		return true
	}, func(elems []*Elem) error {
		nodes := stw.Frame().ElemToNode(elems...)
		return createspline("spline.dxf", nodes, d, z, scale, ndiv, original)
	})
}
