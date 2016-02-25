package st

type Chain struct {
	frame      *Frame
	current    *Node
	next       *Elem
	condition  func(*Chain, *Elem) bool
	terminate  func(*Chain) bool
	checkerror func(*Chain) error
	sorter     func(*Chain, []*Elem) []*Elem
	num        int
	err        error
}

func NewChain(frame *Frame, node *Node, elem *Elem, cond func(*Chain, *Elem) bool, terminate func(*Chain) bool, checkerror func(*Chain) error, sorter func(*Chain, []*Elem) []*Elem) *Chain {
	return &Chain{
		frame:      frame,
		current:    node,
		next:       elem,
		condition:  cond,
		terminate:  terminate,
		checkerror: checkerror,
		sorter:     sorter,
		num:        0,
		err:        nil,
	}
}

func (c *Chain) Num() int {
	return c.num
}

func (c *Chain) Err() error {
	return c.err
}

func (c *Chain) Next() bool {
	defer func() {
		c.num++
	}()
	if c.terminate != nil && c.terminate(c) {
		return false
	}
	if c.checkerror != nil {
		if err := c.checkerror(c); err != nil {
			c.err = err
			return false
		}
	}
	els := make([]*Elem, len(c.frame.Elems))
	num := 0
	for _, el := range c.frame.NodeToElemAny(c.current) {
		if el.IsHidden(c.frame.Show) || !el.IsLineElem() {
			continue
		}
		if el == c.next {
			continue
		}
		els[num] = el
		num++
	}
	if c.sorter != nil {
		els = c.sorter(c, els[:num])
	}
	for _, el := range els {
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
