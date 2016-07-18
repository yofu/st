package st

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"sort"

	"github.com/yofu/dxf"
	"github.com/yofu/st/matrix"
)

var (
	Commands = map[string]func(Commander) chan bool{
		"DISTS":          Dists,
		"MATCHPROPERTY":  MatchProperty,
		"JOINLINEELEM":   JoinLineElem,
		"ERASE":          Erase,
		"ADDLINEELEM":    AddLineElem,
		"ADDPLATEELEM":   AddPlateElem,
		"HATCHPLATEELEM": HatchPlateElem,
		"COPYELEM":       CopyElem,
		"TRIM":           Trim,
		"MOVEUPDOWN":     MoveUpDown,
		"ARCLM201":       Arclm201,
		"SPLINE":         Spline,
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
					stw.SetAltSelectNode(as)
					stw.DeselectNode()
					stw.EndCommand()
					stw.Redraw()
					break onenode
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

func twonodes(stw Commander, f func(*Node, *Node) error) chan bool {
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
					if n0 == nil {
						stw.SelectNode([]*Node{n})
						n0 = n
						stw.AddTail(n0)
					} else {
						err := f(n0, n)
						if err != nil {
							ErrorMessage(stw, err, ERROR)
						}
						stw.SetAltSelectNode(as)
						stw.DeselectNode()
						stw.EndTail()
						stw.EndCommand()
						stw.Redraw()
						break twonodes
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
	return twonodes(stw, func(n0, n *Node) error {
		dx, dy, dz, d := stw.Frame().Distance(n0, n)
		stw.History(fmt.Sprintf("NODE: %d - %d", n0.Num, n.Num))
		stw.History(fmt.Sprintf("DX: %.3f DY: %.3f DZ: %.3f D: %.3f", dx, dy, dz, d))
		return nil
	})
}

func AddLineElem(stw Commander) chan bool {
	return twonodes(stw, func(n0, n *Node) error {
		frame := stw.Frame()
		sec := frame.DefaultSect()
		el := frame.AddLineElem(-1, []*Node{n0, n}, sec, NONE)
		stw.History(fmt.Sprintf("ELEM: %d (ENOD: %d - %d, SECT: %d)", el.Num, n0.Num, n.Num, sec.Num))
		Snapshot(stw)
		return nil
	})
}

func multinodes(stw Commander, f func([]*Node) error, each bool) chan bool {
	quit := make(chan bool)
	go func() {
		nodes := make([]*Node, 0)
		nch := stw.GetNode()
		clickch := stw.GetClick()
		as := stw.AltSelectNode()
		stw.SetAltSelectNode(false)
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
						stw.SelectNode(nodes)
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
				if c.Button == ButtonRight {
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
				}
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
		el := frame.AddPlateElem(-1, ns, sec, NONE)
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

func multielems(stw Commander, f func([]*Elem) error) chan bool {
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
				if el != nil {
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
	return multielems(stw, func(elems []*Elem) error {
		els := make([]*Elem, 2)
		num := 0
		for _, el := range elems {
			if el != nil && el.IsLineElem() {
				els[num] = el
				num++
				if num >= 2 {
					break
				}
			}
		}
		if num < 2 {
			return fmt.Errorf("too few elems")
		}
		frame := stw.Frame()
		err := frame.JoinLineElem(els[0], els[1], true, true)
		if err != nil {
			return err
		}
		Snapshot(stw)
		return nil
	})
}

func Erase(stw Commander) chan bool {
	return multielems(stw, func(elems []*Elem) error {
		frame := stw.Frame()
		for _, el := range elems {
			if el != nil && !el.Lock {
				frame.DeleteElem(el.Num)
			}
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
			frame.AddPlateElem(-1, en, sec, NONE)
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

func onemultielem(stw Commander, cond func(*Elem) bool, f func(Click, *Elem, *Elem)) chan bool {
	quit := make(chan bool)
	var el0 *Elem
	if stw.ElemSelected() {
		for _, el := range stw.SelectedElems() {
			if el.IsLineElem() {
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
					if el == nil {
						ErrorMessage(stw, fmt.Errorf("no elem"), ERROR)
					} else {
						f(c, el0, el)
						stw.Redraw()
					}
				case ButtonRight:
					stw.Deselect()
					Snapshot(stw)
					stw.EndCommand()
					stw.Redraw()
					break trim_click
				}
			case <-quit:
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
	})
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

func Arclm201(stw Commander) chan bool {
	quit := make(chan bool)
	frame := stw.Frame()
	per := "L"
	af := frame.Arclms[per]
	otp := "temp.otp"
	init := true
	lap := 11000
	safety := 0.0001
	start := 0.0
	max := 1.0
	go func() {
		err := af.Arclm201(otp, init, lap, safety, start, max)
		if err != nil {
			fmt.Println(err)
		}
		af.Endch <- err
	}()
	next := make(chan bool)
	go func() {
	read201command:
		for {
			select {
			case <-af.Pivot:
			case <-af.Lapch:
				frame.ReadArclmData(af, per)
				stw.Redraw()
				<-next
				af.Lapch <- 1
			case <-af.Endch:
				stw.Redraw()
				break read201command
			}
		}
	}()
	go func() {
		clickch := stw.GetClick()
	arclm201command:
		for {
			select {
			case c := <-clickch:
				switch c.Button {
				case ButtonLeft:
					next <- true
				case ButtonRight:
					Snapshot(stw)
					stw.EndCommand()
					stw.Redraw()
					Snapshot(stw)
					af.Endch<-nil
					break arclm201command
				}
			case <-quit:
				stw.EndCommand()
				stw.Redraw()
				Snapshot(stw)
				af.Endch<-nil
				break arclm201command
			}
		}
	}()
	return quit
}

func createspline(fn string, nodes []*Node, d, z int, scale float64, ndiv int, original bool) error {
	if len(nodes) == 0 {
		return fmt.Errorf("no nodes")
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
	for i := 0; i < len(nodes) - 1; i++ {
		h[i] = nodes[i+1].Coord[d] - nodes[i].Coord[d]
		y[i] = (nodes[i+1].Coord[z] - nodes[i].Coord[z]) / h[i]
	}
	m := matrix.NewCOOMatrix(len(nodes) - 2)
	v := make([]float64, len(nodes) - 2)
	for i := 0; i < len(nodes) - 2; i++ {
		if i > 0 {
			m.Set(i-1, i, h[i])
		}
		m.Set(i, i, 2.0*(h[i] + h[i+1]))
		if i < len(nodes) - 3 {
			m.Set(i, i+1, h[i+1])
		}
		v[i] = 3.0 * (y[i+1] - y[i])
	}
	conf := make([]bool, m.Size)
	lls := m.ToLLS(0, conf)
	ans, err := lls.Solve(nil, v)
	if err != nil {
		return err
	}
	C := make([]float64, len(nodes))
	for i := 0; i < len(nodes) - 2; i++ {
		C[i+1] = ans[0][i]
	}
	D := make([]float64, len(nodes)-1)
	B := make([]float64, len(nodes)-1)
	A := make([]float64, len(nodes)-1)
	for i := 0; i < len(nodes) - 1; i++ {
		D[i] = (C[i+1] - C[i]) / (3.0*h[i])
		B[i] = y[i] - (C[i] + D[i]*h[i])*h[i]
		A[i] = nodes[i].Coord[z]
	}
	dw := dxf.NewDrawing()
	for i := 0; i < len(nodes) -1; i++ {
		if original {
			dw.Line(nodes[i].Coord[d]*scale, nodes[i].Coord[z]*scale, 0.0, nodes[i+1].Coord[d]*scale, nodes[i+1].Coord[z]*scale, 0.0)
		}
		sx := nodes[i].Coord[d]
		dcx := h[i] / float64(ndiv)
		ex := sx + dcx
		dx := ex - nodes[i].Coord[d]
		sy := A[i]
		ey := A[i] + B[i] * dx + C[i] * dx * dx + D[i] * dx * dx * dx
		for j := 0; j < ndiv; j++ {
			dw.Line(sx*scale, sy*scale, 0.0, ex*scale, ey*scale, 0.0)
			sx = ex
			sy = ey
			ex = ex + dcx
			dx = ex - nodes[i].Coord[d]
			ey = A[i] + B[i] * dx + C[i] * dx * dx + D[i] * dx * dx * dx
		}
	}
	return dw.SaveAs(fn)
}

func Spline(stw Commander) chan bool {
	d := 1
	z := 2
	scale := 1000.0
	ndiv := 4
	original := true
	return multielems(stw, func(elems []*Elem) error {
		nodes := stw.Frame().ElemToNode(elems...)
		return createspline("spline.dxf", nodes, d, z, scale, ndiv, original)
	})
}
