package st

import (
	"bufio"
	"os"
	"strings"
)

type Commander interface {
	Selector
	GetElem() chan *Elem
	GetNode() chan *Node
	GetClick() chan Click
	AddTail(*Node)
	EndTail()
	EndCommand()
	AddCommandAlias(key string, value func(Commander) chan bool)
	DeleteCommandAlias(key string)
	ClearCommandAlias()
	CommandAlias(key string) (func(Commander) chan bool, bool)
}

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

func ReadPgp(stw Commander, filename string) error {
	aliases := make(map[string]func(Commander) chan bool)
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		txt := s.Text()
		if strings.HasPrefix(txt, "#") {
			continue
		}
		var words []string
		for _, k := range strings.Split(txt, " ") {
			if k != "" {
				words = append(words, k)
			}
		}
		if len(words) < 2 {
			continue
		}
		if value, ok := Commands[strings.ToUpper(words[1])]; ok {
			aliases[strings.ToUpper(words[0])] = value
		} else if strings.HasPrefix(words[1], ":") {
			command := strings.Join(words[1:], " ")
			aliases[strings.ToUpper(words[0])] = func(stw Commander) chan bool {
				if ew, ok := stw.(ExModer); ok {
					err := ExMode(ew, command)
					if err != nil {
						ErrorMessage(stw, err, ERROR)
					}
				}
				return nil
			}
		} else if strings.HasPrefix(words[1], "'") {
			command := strings.Join(words[1:], " ")
			aliases[strings.ToUpper(words[0])] = func(stw Commander) chan bool {
				if fw, ok := stw.(Fig2Moder); ok {
					err := Fig2Mode(fw, command)
					if err != nil {
						ErrorMessage(stw, err, ERROR)
					}
				}
				return nil
			}
		}
	}
	if err := s.Err(); err != nil {
		return err
	}
	stw.ClearCommandAlias()
	for k, v := range aliases {
		stw.AddCommandAlias(k, v)
	}
	return nil
}
