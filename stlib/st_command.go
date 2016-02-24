package st

import (
	"fmt"
)

type Commander interface {
	Selector
	GetElem() chan *Elem
	GetNode() chan *Node
	EndCommand()
}

func Dists(stw Commander) chan bool {
	quit := make(chan bool)
	go func () {
		var n0 *Node
		nch := stw.GetNode()
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
		if !stw.ElemSelected() {
		matchproperty_get:
			for {
				select {
				case el := <-elch:
					stw.SelectElem([]*Elem{el})
					sect = el.Sect
					etype = el.Etype
					break matchproperty_get
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
			case <-quit:
				break matchproperty_paste
			}
		}
	}()
	return quit
}
