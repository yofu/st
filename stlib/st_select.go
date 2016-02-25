package st

const (
	nodeSelectPixel = 15
	elemSelectPixel = 5
)

type Selector interface {
	Window

	// Selection
	SelectedElems() []*Elem
	SelectedNodes() []*Node
	SelectElem([]*Elem)
	SelectNode([]*Node)
	ElemSelected() bool
	NodeSelected() bool
	Deselect()
	DeselectElem()
	DeselectNode()

	ToggleAltSelectNode()
	AltSelectNode() bool
	SetAltSelectNode(bool)
}

func SelectConfed(stw Selector) {
	stw.Deselect()
	num := 0
	frame := stw.Frame()
	nodes := make([]*Node, len(frame.Nodes))
	for _, n := range frame.Nodes {
		if n.IsHidden(frame.Show) {
			continue
		}
		for i := 0; i < 6; i++ {
			if n.Conf[i] {
				nodes = append(nodes, n)
				num++
				break
			}
		}
	}
	stw.SelectNode(nodes[:num])
}

func CheckFrame(stw Selector) {
	stw.Deselect()
	frame := stw.Frame()
	ns := frame.NodeNoReference()
	if len(ns) != 0 {
		stw.SelectNode(ns)
		stw.Redraw()
		if stw.Yn("NODE NO REFERENCE", "不要な節点を削除しますか?") {
			for _, n := range ns {
				frame.DeleteNode(n.Num)
			}
			Snapshot(stw)
		} else {
			return
		}
	}
	stw.Deselect()
	nm := frame.NodeDuplication(stw.EPS())
	if len(nm) != 0 {
		nodes := make([]*Node, len(nm))
		num := 0
		for k := range nm {
			nodes[num] = k
			num++
		}
		stw.SelectNode(nodes[:num])
		stw.Redraw()
		if stw.Yn("NODE DUPLICATION", "重なった節点を削除しますか?") {
			frame.ReplaceNode(nm)
			Snapshot(stw)
		} else {
			return
		}
	}
	stw.Deselect()
	els := frame.ElemSameNode()
	if len(els) != 0 {
		stw.SelectNode(frame.ElemToNode(els...))
		stw.SelectElem(els)
		stw.Redraw()
		if stw.Yn("ELEM SAME NODE", "部材を削除しますか?") {
			for _, el := range els {
				if el.Lock {
					continue
				}
				frame.DeleteElem(el.Num)
			}
			Snapshot(stw)
		} else {
			return
		}
	}
	stw.Deselect()
	els2 := frame.ElemDuplication(nil)
	if len(els) != 0 {
		sels := make([]*Elem, len(els2))
		num := 0
		for k := range els2 {
			sels[num] = k
			num++
		}
		stw.SelectElem(sels[:num])
		stw.Redraw()
		if stw.Yn("ELEM DUPLICATION", "重なった部材を削除しますか?") {
			for el := range els2 {
				if el.Lock {
					continue
				}
				frame.DeleteElem(el.Num)
			}
			Snapshot(stw)
		} else {
			return
		}
	}
	stw.Deselect()
	ns, els, ok := frame.Check()
	if !ok {
		stw.SelectNode(ns)
		stw.SelectElem(els)
		stw.Redraw()
		if stw.Yn("CHECK FRAME", "無効な節点と部材を削除しますか？") {
			for _, n := range els {
				if n.Lock {
					continue
				}
				frame.DeleteNode(n.Num)
			}
			for _, el := range els {
				if el.Lock {
					continue
				}
				frame.DeleteElem(el.Num)
			}
			Snapshot(stw)
		} else {
			return
		}
	}
	stw.Deselect()
	if !frame.IsUpside() {
		if stw.Yn("CHECK FRAME", "部材の向きを修正しますか？") {
			frame.Upside()
			Snapshot(stw)
		} else {
			return
		}
	}
	stw.Deselect()
}

func AddSelection(stw Selector, entity interface{}) {
	switch en := entity.(type) {
	case *Node:
		var ns []*Node
		if stw.NodeSelected() {
			ns = stw.SelectedNodes()
			ns = append(ns, en)
		} else {
			ns = []*Node{en}
		}
		stw.SelectNode(ns)
	case *Elem:
		var els []*Elem
		if stw.ElemSelected() {
			els = stw.SelectedElems()
			els = append(els, en)
		} else {
			els = []*Elem{en}
		}
		stw.SelectElem(els)
	}
}

func MergeSelectElem(stw Selector, elems []*Elem, deselect bool) {
	k := len(elems)
	els := stw.SelectedElems()
	if deselect {
		for l := 0; l < k; l++ {
			for m, el := range els {
				if el == elems[l] {
					if m == len(els)-1 {
						els = els[:m]
					} else {
						els = append(els[:m], els[m+1:]...)
					}
					break
				}
			}
		}
	} else {
		var add bool
		for l := 0; l < k; l++ {
			add = true
			for _, n := range els {
				if n == elems[l] {
					add = false
					break
				}
			}
			if add {
				els = append(els, elems[l])
			}
		}
	}
	stw.SelectElem(els)
}

func MergeSelectNode(stw Selector, nodes []*Node, isshift bool) {
	k := len(nodes)
	ns := stw.SelectedNodes()
	if isshift {
		for l := 0; l < k; l++ {
			for m, el := range ns {
				if el == nodes[l] {
					if m == len(ns)-1 {
						ns = ns[:m]
					} else {
						ns = append(ns[:m], ns[m+1:]...)
					}
					break
				}
			}
		}
	} else {
		var add bool
		for l := 0; l < k; l++ {
			add = true
			for _, n := range ns {
				if n == nodes[l] {
					add = false
					break
				}
			}
			if add {
				ns = append(ns, nodes[l])
			}
		}
	}
	stw.SelectNode(ns)
}

func DeleteSelected(stw Selector) {
	if stw.ElemSelected() {
		frame := stw.Frame()
		for _, el := range stw.SelectedElems() {
			if el != nil && !el.Lock {
				frame.DeleteElem(el.Num)
			}
		}
	}
	stw.Deselect()
}

func HideNotSelected(stw Selector) {
	if stw.ElemSelected() {
		frame := stw.Frame()
		for _, n := range frame.Nodes {
			n.Hide()
		}
		for _, el := range frame.Elems {
			el.Hide()
		}
		for _, el := range stw.SelectedElems() {
			if el != nil {
				el.Show()
				for _, en := range el.Enod {
					en.Show()
				}
			}
		}
	}
}

func HideSelected(stw Selector) {
	if stw.ElemSelected() {
		for _, el := range stw.SelectedElems() {
			if el != nil {
				el.Hide()
			}
		}
	}
}
