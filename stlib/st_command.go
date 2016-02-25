package st

import (
	"fmt"
)

type Commander interface {
	Selector
	GetElem() chan *Elem
	GetNode() chan *Node
	GetClick() chan Click
	EndCommand()
}

func twonodes(stw Commander, f func(*Node, *Node) error) chan bool {
	quit := make(chan bool)
	go func () {
		var n0 *Node
		nch := stw.GetNode()
		clickch := stw.GetClick()
		as := stw.AltSelectNode()
		stw.SetAltSelectNode(false)
	twonodes:
		for {
			select {
			case n := <-nch:
				if n0 == nil {
					stw.SelectNode([]*Node{n})
					n0 = n
				} else {
					err := f(n0, n)
					if err != nil {
						ErrorMessage(stw, err, ERROR)
					}
					stw.SetAltSelectNode(as)
					stw.DeselectNode()
					stw.EndCommand()
					break twonodes
				}
			case c := <-clickch:
				if c.Button == ButtonRight {
					stw.SetAltSelectNode(as)
					stw.Deselect()
					stw.EndCommand()
					break twonodes
				}
			case <-quit:
				stw.SetAltSelectNode(as)
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
		return nil
	})
}

func MatchProperty(stw Commander) chan bool {
	quit := make(chan bool)
	go func() {
		var sect *Sect
		var etype int
		elch := stw.GetElem()
		clickch := stw.GetClick()
		if !stw.ElemSelected() {
		matchproperty_get:
			for {
				select {
				case el := <-elch:
					stw.SelectElem([]*Elem{el})
					sect = el.Sect
					etype = el.Etype
					break matchproperty_get
				case c := <-clickch:
					if c.Button == ButtonRight {
						stw.Deselect()
						stw.EndCommand()
						return
					}
				case <-quit:
					return
				}
			}
		} else {
			el := stw.SelectedElems()[0]
			sect = el.Sect
			etype = el.Etype
		}
	matchproperty_paste:
		for {
			select {
			case el := <-elch:
				el.Sect = sect
				el.Etype = etype
			case c := <-clickch:
				if c.Button == ButtonRight {
					stw.Deselect()
					stw.EndCommand()
					break matchproperty_paste
				}
			case <-quit:
				break matchproperty_paste
			}
		}
	}()
	return quit
}

func JoinLineElem(stw Commander) chan bool {
	quit := make(chan bool)
	go func() {
		elch := stw.GetElem()
		clickch := stw.GetClick()
	joinlineelem:
		for {
			select {
			case el := <-elch:
				AddSelection(stw, el)
			case c := <-clickch:
				if c.Button == ButtonRight {
					els := make([]*Elem, 2)
					num := 0
					for _, el := range stw.SelectedElems() {
						if el != nil && el.IsLineElem() {
							els[num] = el
							num++
							if num >= 2 {
								break
							}
						}
					}
					if num == 2 {
						frame := stw.Frame()
						err := frame.JoinLineElem(els[0], els[1], true, true)
						if err != nil {
							ErrorMessage(stw, err, ERROR)
						}
						stw.Deselect()
						stw.EndCommand()
						break joinlineelem
					}
				}
			case <-quit:
				break joinlineelem
			}
		}
	}()
	return quit
}

func Erase(stw Commander) chan bool {
	del := func() {
		DeleteSelected(stw)
		stw.Deselect()
		frame := stw.Frame()
		ns := frame.NodeNoReference()
		if len(ns) != 0 {
			for _, n := range ns {
				frame.DeleteNode(n.Num)
			}
		}
	}
	if stw.ElemSelected() {
		del()
		return nil
	}
	quit := make(chan bool)
	go func() {
		elch := stw.GetElem()
		clickch := stw.GetClick()
	erase:
		for {
			select {
			case el := <-elch:
				AddSelection(stw, el)
			case c := <-clickch:
				if c.Button == ButtonRight {
					del()
					stw.EndCommand()
					break erase
				}
			case <-quit:
				break erase
			}
		}
	}()
	return quit
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
						break hatchplateelem
					}
					err = createhatch(ns)
					if err != nil {
						ErrorMessage(stw, err, ERROR)
						stw.EndCommand()
						break hatchplateelem
					}
				case ButtonRight:
					stw.EndCommand()
					break hatchplateelem
				}
			case <-quit:
				break hatchplateelem
			}
		}
	}()
	return quit
}

func Trim(stw Commander) chan bool {
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
					if el.IsLineElem() {
						el0 = el
						break trim_elem
					}
				case c := <-clickch:
					if c.Button == ButtonRight {
						stw.Deselect()
						stw.EndCommand()
						return
					}
				case <-quit:
					return
				}
			}
		}
		AddSelection(stw, el0)
		frame := stw.Frame()
		eps := stw.EPS()
		var err error
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
						if DotLine(el0.Enod[0].Pcoord[0], el0.Enod[0].Pcoord[1], el0.Enod[1].Pcoord[0], el0.Enod[1].Pcoord[1], float64(c.X), float64(c.Y))*DotLine(el0.Enod[0].Pcoord[0], el0.Enod[0].Pcoord[1], el0.Enod[1].Pcoord[0], el0.Enod[1].Pcoord[1], el.Enod[0].Pcoord[0], el.Enod[0].Pcoord[1]) < 0.0 {
							_, _, err = frame.Trim(el0, el, 1, eps)
						} else {
							_, _, err = frame.Trim(el0, el, -1, eps)
						}
						if err != nil {
							ErrorMessage(stw, err, ERROR)
						}
					}
				case ButtonRight:
					stw.Deselect()
					stw.EndCommand()
					break trim_click
				}
			case <-quit:
				break trim_click
			}
		}
	}()
	return quit
}
