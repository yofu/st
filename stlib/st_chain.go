package st

type Chain struct {
	frame *Frame
	current *Node
	next *Elem
	condition func(*Chain, *Elem) bool
	terminate func(*Chain) bool
}

func NewChain(frame *Frame, node *Node, elem *Elem, cond func(*Chain, *Elem) bool, terminate func(*Chain) bool) *Chain {
	return &Chain{
		frame: frame,
		current: node,
		next: elem,
		condition: cond,
		terminate: terminate,
	}
}

func (c *Chain) Next() bool {
	if c.terminate(c) {
		return false
	}
	for _, el := range c.frame.NodeToElemAny(c.current) {
		if !el.IsLineElem() {
			continue
		}
		if el == c.next {
			continue
		}
		if c.condition(c, el) {
			c.next = el
			c.current = el.Otherside(c.current)
			return true
		}
	}
	return false
}

func (c *Chain) Node() *Node {
	return c.current
}

func (c *Chain) Elem() *Elem {
	return c.next
}

func Straight(eps float64) func(*Chain, *Elem) bool {
	return func(c *Chain, el *Elem) bool {
		e0 := c.Elem()
		if e0.Sect != el.Sect {
			return false
		}
		if !IsParallel(e0.Direction(true), el.Direction(true), eps) {
			return false
		}
		return true
	}
}
