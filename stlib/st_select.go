package st

type Selector interface {
	Window
	SelectedElems() []*Elem
	SelectedNodes() []*Node
	SelectElem([]*Elem)
	SelectNode([]*Node)
	ElemSelected() bool
	NodeSelected() bool
	Deselect()
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
