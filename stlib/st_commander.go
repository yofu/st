package st

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/atotto/clipboard"
)

type Commander interface {
	Selector
	Execute(chan bool)
	GetElem() chan *Elem
	SendElem(*Elem)
	GetNode() chan *Node
	SendNode(*Node)
	GetClick() chan Click
	SendClick(Click)
	GetModifier() chan Modifier
	SendModifier(Modifier)
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

type Modifier struct {
	Shift bool
	Ctrl  bool
	Alt   bool
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
	mod   chan Modifier
}

func NewCommandBuffer() *CommandBuffer {
	return &CommandBuffer{
		on:    false,
		quit:  nil,
		elem:  nil,
		node:  nil,
		click: nil,
		mod:   nil,
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

func (cb *CommandBuffer) GetModifier() chan Modifier {
	cb.mod = make(chan Modifier)
	return cb.mod
}

func (cb *CommandBuffer) SendModifier(m Modifier) {
	if cb.mod != nil {
		cb.mod <- m
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
		cb.mod = nil
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

func CopyClipboard(stw Commander) error {
	frame := stw.Frame()
	if !stw.ElemSelected() {
		return nil
	}
	var otp bytes.Buffer
	ns := frame.ElemToNode(stw.SelectedElems()...)
	stw.Execute(onenode(stw, func(n0 * Node) error {
		for _, n := range ns {
			otp.WriteString(n.CopyString(n0.Coord[0], n0.Coord[1], n0.Coord[2]))
		}
		for _, el := range stw.SelectedElems() {
			if el != nil {
				otp.WriteString(el.InpString())
			}
		}
		err := clipboard.WriteAll(otp.String())
		if err != nil {
			return err
		}
		stw.History(fmt.Sprintf("%d ELEMs Copied", len(stw.SelectedElems())))
		return nil
	}))
	return nil
}

// TODO: Test
func PasteClipboard(stw Commander) error {
	frame := stw.Frame()
	text, err := clipboard.ReadAll()
	if err != nil {
		return err
	}
	stw.Execute(onenode(stw, func(n *Node) error {
		s := bufio.NewScanner(strings.NewReader(text))
		coord := make([]float64, 3)
		for i := 0; i < 3; i++ {
			coord[i] = n.Coord[i]
		}
		angle := 0.0
		tmp := make([]string, 0)
		nodemap := make(map[int]int)
		var err error
		for s.Scan() {
			var words []string
			for _, k := range strings.Split(s.Text(), " ") {
				if k != "" {
					words = append(words, k)
				}
			}
			if len(words) == 0 {
				continue
			}
			first := words[0]
			switch first {
			default:
				tmp = append(tmp, words...)
			case "NODE", "ELEM":
				nodemap, err = frame.ParseInp(tmp, coord, angle, nodemap, false)
				tmp = words
			}
			if err != nil {
				break
			}
		}
		if err != nil {
			return err
		}
		nodemap, err = frame.ParseInp(tmp, coord, angle, nodemap, false)
		return err
	}))
	return nil
}
