package st

import "fmt"

type Chain struct {
	frame      *Frame
	current    *Node
	next       *Elem
	condition  func(*Chain, *Elem) bool
	terminate  func(*Chain) bool
	checkerror func(*Chain) error
	sorter     func(*Chain, []*Elem) []*Elem
	err        error
	ind        int
	elems      []*Elem
}

func NewChain(frame *Frame, node *Node, elem *Elem, cond func(*Chain, *Elem) bool, terminate func(*Chain) bool, checkerror func(*Chain) error, sorter func(*Chain, []*Elem) []*Elem) *Chain {
	var elems []*Elem
	if elem == nil {
		elems = make([]*Elem, 0)
	} else {
		elems = []*Elem{elem}
	}
	return &Chain{
		frame:      frame,
		current:    node,
		next:       elem,
		condition:  cond,
		terminate:  terminate,
		checkerror: checkerror,
		sorter:     sorter,
		err:        nil,
		ind:        0,
		elems:      elems,
	}
}

func (c *Chain) Size() int {
	return len(c.elems)
}

func (c *Chain) Num() int {
	if c.elems == nil || len(c.elems) == 0 {
		return 0
	}
	return c.elems[0].Num
}

func (c *Chain) Snapshot(frame *Frame) *Chain {
	elems := make([]*Elem, len(c.elems))
	for i := 0; i < len(c.elems); i++ {
		elems[i] = frame.Elems[c.elems[i].Num]
	}
	return &Chain{
		frame:      frame,
		current:    elems[c.ind].Enod[1],
		next:       elems[c.ind],
		condition:  c.condition,
		terminate:  c.terminate,
		checkerror: c.checkerror,
		sorter:     c.sorter,
		err:        nil,
		ind:        c.ind,
		elems:      elems,
	}
}

func (c *Chain) Err() error {
	return c.err
}

func (c *Chain) Next() bool {
	if c.ind < len(c.elems) {
		c.current = c.elems[c.ind].Enod[1]
		c.next = c.elems[c.ind]
		c.ind++
		return true
	}
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
	if num == 0 {
		return false
	}
	if c.sorter != nil {
		els = c.sorter(c, els[:num])
	}
	for _, el := range els {
		if c.condition(c, el) {
			c.next = el
			c.current = el.Otherside(c.current)
			c.elems = append(c.elems, c.next)
			c.ind++
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

func (c *Chain) Elems() []*Elem {
	return c.elems
}

func (c *Chain) Append(elem *Elem) error {
	return c.AppendAt(elem, len(c.elems))
}

func (c *Chain) AppendAt(elem *Elem, ind int) error {
	if !elem.IsLineElem() {
		return fmt.Errorf("cannot append element to chain")
	}
	if ind > len(c.elems) {
		return fmt.Errorf("cannot append element at index %d", ind)
	}
	if len(c.elems) == 0 {
		c.current = elem.Enod[1]
		c.next = elem
		c.elems = append(c.elems, elem)
		return nil
	}
	if c.elems[ind-1].Enod[1].Num != elem.Enod[0].Num {
		if c.elems[ind-1].Enod[1].Num == elem.Enod[1].Num {
			elem.Invert()
		} else {
			return fmt.Errorf("cannot append element to chain")
		}
	}
	elems := make([]*Elem, len(c.elems)+1)
	for i := 0; i < ind; i++ {
		elems[i] = c.elems[i]
	}
	elems[ind] = elem
	for i := ind + 1; i <= len(c.elems); i++ {
		elems[i] = c.elems[i-1]
	}
	c.current = elem.Enod[1]
	c.next = elem
	c.elems = elems
	c.ind = ind
	return nil
}

func (c *Chain) Delete(elem *Elem) error {
	ind, err := c.Has(elem)
	if err != nil {
		return err
	}
	elem.Chain = nil
	if len(c.elems) == 1 {
		delete(c.frame.Chains, elem.Num)
		return nil
	}
	switch ind {
	case 0:
		delete(c.frame.Chains, elem.Num)
		c.frame.Chains[c.elems[1].Num] = c
		c.elems = c.elems[1:]
	case len(c.elems) - 1:
		c.elems = c.elems[:len(c.elems)-1]
	default:
		newchain := &Chain{
			frame:      c.frame,
			current:    c.elems[ind+1].Enod[1],
			next:       c.elems[ind+1],
			condition:  c.condition,
			terminate:  c.terminate,
			checkerror: c.checkerror,
			sorter:     c.sorter,
			err:        nil,
			ind:        0,
			elems:      c.elems[ind+1:],
		}
		c.frame.Chains[newchain.elems[0].Num] = newchain
		for _, el := range newchain.elems {
			el.Chain = newchain
		}
		c.elems = c.elems[:ind]
		c.ind = ind
	}
	return nil
}

func (c *Chain) DivideAt(ind int) error {
	if ind == 0 || ind == len(c.elems) {
		return fmt.Errorf("cannot divide chain at %d", ind)
	}
	newchain := &Chain{
		frame:      c.frame,
		current:    c.elems[ind].Enod[1],
		next:       c.elems[ind],
		condition:  c.condition,
		terminate:  c.terminate,
		checkerror: c.checkerror,
		sorter:     c.sorter,
		err:        nil,
		ind:        0,
		elems:      c.elems[ind:],
	}
	for _, el := range newchain.elems {
		el.Chain = newchain
	}
	c.frame.Chains[newchain.elems[0].Num] = newchain
	c.elems = c.elems[:ind]
	c.ind = ind - 1
	return nil
}

func (c *Chain) Has(elem *Elem) (int, error) {
	for i, c := range c.elems {
		if c.Num == elem.Num {
			return i, nil
		}
	}
	return -1, fmt.Errorf("chain doesn't have ELEM %d", elem.Num)
}

func (c *Chain) IsStraight(eps float64) bool {
	if c.elems == nil || len(c.elems) == 0 {
		return false
	}
	d := c.elems[0].Direction(true)
	for _, el := range c.elems[1:] {
		if !IsParallel(d, el.Direction(true), eps) {
			return false
		}
	}
	return true
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
