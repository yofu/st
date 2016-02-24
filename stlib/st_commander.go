package st

type Click int

const (
	ClickLeft Click = iota
	ClickMiddle
	ClickRight
)

type CommandBuffer struct {
	on   bool
	quit chan bool
	elem chan *Elem
	node chan *Node
	click chan Click
}

func NewCommandBuffer() *CommandBuffer {
	return &CommandBuffer{
		on:   false,
		quit: nil,
		elem: nil,
		node: nil,
		click: nil,
	}
}

func (cb *CommandBuffer) Executing() bool {
	return cb.on
}

func (cb *CommandBuffer) Execute(q chan bool) {
	if q == nil {
		return
	}
	cb.on = true
	cb.quit = q
}

func (cb *CommandBuffer) GetElem() chan *Elem {
	cb.elem = make(chan *Elem)
	return cb.elem
}

func (cb *CommandBuffer) SendElem(el *Elem) {
	if cb.elem != nil {
		cb.elem <- el
	}
}

func (cb *CommandBuffer) GetNode() chan *Node {
	cb.node = make(chan *Node)
	return cb.node
}

func (cb *CommandBuffer) SendNode(n *Node) {
	if cb.node != nil {
		cb.node <- n
	}
}

func (cb *CommandBuffer) GetClick() chan Click {
	cb.click = make(chan Click)
	return cb.click
}

func (cb *CommandBuffer) SendClick(c Click) {
	if cb.click != nil {
		cb.click <- c
	}
}

func (cb *CommandBuffer) EndCommand() {
	if cb.on {
		cb.on = false
		cb.quit <- true
		cb.quit = nil
		cb.elem = nil
		cb.node = nil
	}
}
