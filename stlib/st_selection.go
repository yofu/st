package st

type Selection struct {
	elems []*Elem
	nodes []*Node
}

func NewSelection() *Selection {
	return &Selection{
		elems: make([]*Elem, 0),
		nodes: make([]*Node, 0),
	}
}

func (s *Selection) SelectElem(els []*Elem) {
	s.elems = els
}

func (s *Selection) SelectNode(ns []*Node) {
	s.nodes = ns
}

func (s *Selection) ElemSelected() bool {
	if s.elems == nil || len(s.elems) == 0 {
		return false
	} else {
		return true
	}
}

func (s *Selection) NodeSelected() bool {
	if s.nodes == nil || len(s.nodes) == 0 {
		return false
	} else {
		return true
	}
}

func (s *Selection) SelectedElems() []*Elem {
	return s.elems
}

func (s *Selection) SelectedNodes() []*Node {
	return s.nodes
}

func (s *Selection) Deselect() {
	s.elems = make([]*Elem, 0)
	s.nodes = make([]*Node, 0)
}

