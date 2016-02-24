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
	cb.elem = make(chan *Elem)
	cb.node = make(chan *Node)
	cb.click = make(chan Click)
}

func (cb *CommandBuffer) GetElem() chan *Elem {
	return cb.elem
}

func (cb *CommandBuffer) SendElem(el *Elem) {
	cb.elem <- el
}

func (cb *CommandBuffer) GetNode() chan *Node {
	return cb.node
}

func (cb *CommandBuffer) SendNode(n *Node) {
	cb.node <- n
}

func (cb *CommandBuffer) GetClick() chan Click {
	return cb.click
}

func (cb *CommandBuffer) SendClick(c Click) {
	cb.click <- c
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
