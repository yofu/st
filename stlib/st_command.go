package st

type Commander interface {
	Selector
	GetElem() chan *Elem
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
