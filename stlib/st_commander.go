package st

type Click struct {
	Button int
	X      int
	Y      int
}

const (
	ButtonLeft int = iota
	ButtonMiddle
	ButtonRight
)

func ClickLeft(x, y int) Click {
	return Click{ButtonLeft, x, y}
}

func ClickMiddle(x, y int) Click {
	return Click{ButtonMiddle, x, y}
}

func ClickRight(x, y int) Click {
	return Click{ButtonRight, x, y}
}

type CommandBuffer struct {
	on    bool
	quit  chan bool
	elem  chan *Elem
	node  chan *Node
	click chan Click
}

func NewCommandBuffer() *CommandBuffer {
	return &CommandBuffer{
		on:    false,
		quit:  nil,
		elem:  nil,
		node:  nil,
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


func (cb *CommandBuffer) QuitCommand() {
	if cb.on && cb.quit != nil {
		cb.quit <- true
	}
}

func (cb *CommandBuffer) EndCommand() {
	if cb.on {
		cb.on = false
		cb.elem = nil
		cb.node = nil
		cb.click = nil
		cb.quit = nil
	}
}
