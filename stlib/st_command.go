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

func Dists(stw Commander) chan bool {
	quit := make(chan bool)
	go func () {
		var n0 *Node
		nch := stw.GetNode()
		clickch := stw.GetClick()
		as := stw.AltSelectNode()
		stw.SetAltSelectNode(false)
	dists:
		for {
			select {
			case n := <-nch:
				if n0 == nil {
					stw.SelectNode([]*Node{n})
					n0 = n
				} else {
					dx, dy, dz, d := stw.Frame().Distance(n0, n)
					stw.History(fmt.Sprintf("NODE: %d - %d", n0.Num, n.Num))
					stw.History(fmt.Sprintf("DX: %.3f DY: %.3f DZ: %.3f D: %.3f", dx, dy, dz, d))
					stw.SetAltSelectNode(as)
					stw.DeselectNode()
					stw.EndCommand()
					break dists
				}
			case c := <-clickch:
				if c == ClickRight {
					stw.SetAltSelectNode(as)
					stw.Deselect()
					stw.EndCommand()
					break dists
				}
			case <-quit:
				stw.SetAltSelectNode(as)
				break dists
			}
		}
	}()
	return quit
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
					if c == ClickRight {
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
				if c == ClickRight {
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
				if c == ClickRight {
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
