package main

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	st "github.com/yofu/st/stlib"
	"golang.org/x/mobile/event/key"
)

type editInterfaces interface {
	st.ExModer
	st.Commander
}

type editWindow struct {
	editInterfaces
	selection *st.Selection
	frame     *st.Frame
	log       bytes.Buffer
	confirm   bool
	pointer   [2]int
	alt       bool
	waiting   chan []any
	abandon   chan struct{}
	done      chan struct{}
	failure   chan error
	prompts   map[chan st.CommandText]string
}

var _ st.Commander = (*editWindow)(nil)
var _ st.ExModer = (*editWindow)(nil)

func (w *editWindow) Frame() *st.Frame         { return w.frame }
func (w *editWindow) EPS() float64             { return 1e-3 }
func (w *editWindow) Cwd() string              { return "" }
func (w *editWindow) Recent() []string         { return nil }
func (w *editWindow) History(s string)         { fmt.Fprintln(&w.log, s) }
func (w *editWindow) HistoryWriter() io.Writer { return &w.log }
func (w *editWindow) Redraw() {
	w.frame.View.Set(1)
	for _, n := range w.frame.Nodes {
		w.frame.View.ProjectNode(n)
	}
}
func (w *editWindow) RedrawNode()             { w.Redraw() }
func (w *editWindow) Changed(bool)            {}
func (w *editWindow) UndoEnabled() bool       { return false }
func (w *editWindow) SetLastExCommand(string) {}
func (w *editWindow) Yn(string, string) bool  { return w.confirm }
func (w *editWindow) Yna(string, string, string) int {
	if w.confirm {
		return 1
	}
	return 0
}
func (w *editWindow) SectionData(*st.Sect)                  {}
func (w *editWindow) ShapeData(st.Shape)                    {}
func (w *editWindow) TextBox(string) *st.TextBox            { return nil }
func (w *editWindow) SetAltSelectNode(v bool)               { w.alt = v }
func (w *editWindow) AltSelectNode() bool                   { return w.alt }
func (w *editWindow) EndCommand()                           {}
func (w *editWindow) AddTail(*st.Node)                      {}
func (w *editWindow) EndTail()                              {}
func (w *editWindow) StopWheel()                            {}
func (w *editWindow) GetNode() chan *st.Node                { return make(chan *st.Node) }
func (w *editWindow) GetElem() chan *st.Elem                { return make(chan *st.Elem) }
func (w *editWindow) GetClick() chan st.Click               { return make(chan st.Click) }
func (w *editWindow) GetKey() chan st.Key                   { return make(chan st.Key) }
func (w *editWindow) GetAxis() chan *st.Axis                { return make(chan *st.Axis) }
func (w *editWindow) GetWheel() chan st.Wheel               { return make(chan st.Wheel) }
func (w *editWindow) GetModifier() chan st.Modifier         { return make(chan st.Modifier) }
func (w *editWindow) GetFenceGesture() chan st.FenceGesture { return make(chan st.FenceGesture) }
func (w *editWindow) CommandPrompt(title, initial string) chan st.CommandText {
	c := make(chan st.CommandText)
	w.prompts[c] = fmt.Sprintf("%s (default %q)", title, initial)
	return c
}

type editScope struct{ w *editWindow }

func (w *editWindow) BeginInteractiveCommand() st.InteractiveScope { return &editScope{w} }
func (g *editScope) Wait(ch ...any) {
	select {
	case g.w.waiting <- ch:
	case <-g.w.abandon:
		panic("interactive input cancelled")
	}
}
func (g *editScope) WaitDirect(ch ...any) { g.Wait(ch...) }
func (g *editScope) Resume()              {}
func (g *editScope) Close() {
	if v := recover(); v != nil {
		g.w.failure <- fmt.Errorf("interactive command: %v", v)
	}
	close(g.w.done)
}

type editInput struct {
	Point    *[3]float64 `json:"point,omitempty"`
	EndPoint *[3]float64 `json:"end_point,omitempty"`
	Kind     string      `json:"kind"`
	ID       int         `json:"id,omitempty"`
	XY       [2]int      `json:"xy,omitempty"`
	End      [2]int      `json:"end,omitempty"`
	Text     string      `json:"text,omitempty"`
	Shift    bool        `json:"shift,omitempty"`
	Ctrl     bool        `json:"ctrl,omitempty"`
}

func inputChannel[T any](channels []any) chan T {
	for _, c := range channels {
		if ch, ok := c.(chan T); ok && ch != nil {
			return ch
		}
	}
	return nil
}

func deliver[T any](ch chan T, value T) error {
	if ch == nil {
		return fmt.Errorf("command is waiting for a different input")
	}
	select {
	case ch <- value:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("command did not receive input")
	}
}

func (w *editWindow) sendInput(ch []any, e editInput) error {
	project := func(point *[3]float64, out *[2]int) error {
		if point == nil {
			return nil
		}
		p := w.frame.View.ProjectCoord(point[:])
		for i := range out {
			if math.IsNaN(p[i]) || math.IsInf(p[i], 0) || math.Abs(p[i]) > 1e9 {
				return fmt.Errorf("point is outside the projection")
			}
			out[i] = int(math.Round(p[i]))
		}
		return nil
	}
	if err := project(e.Point, &e.XY); err != nil {
		return err
	}
	if err := project(e.EndPoint, &e.End); err != nil {
		return err
	}
	w.pointer = e.XY
	switch e.Kind {
	case "node":
		n := w.frame.Nodes[e.ID]
		if n == nil {
			return fmt.Errorf("unknown node %d", e.ID)
		}
		return deliver(inputChannel[*st.Node](ch), n)
	case "element":
		el := w.frame.Elems[e.ID]
		if el == nil {
			return fmt.Errorf("unknown element %d", e.ID)
		}
		return deliver(inputChannel[*st.Elem](ch), el)
	case "axis":
		if w.frame.LocalAxis == nil || e.ID < 0 || e.ID > 2 {
			return fmt.Errorf("axis requires an active local axis and index 0 to 2")
		}
		axis := *w.frame.LocalAxis
		axis.Current = e.ID
		return deliver(inputChannel[*st.Axis](ch), &axis)
	case "click", "finish":
		button := st.ButtonLeft
		if e.Kind == "finish" {
			button = st.ButtonRight
		}
		return deliver(inputChannel[st.Click](ch), st.Click{Button: button, X: e.XY[0], Y: e.XY[1]})
	case "fence":
		return deliver(inputChannel[st.FenceGesture](ch), st.FenceGesture{Start: st.Position{X: e.XY[0], Y: e.XY[1]}, End: st.Position{X: e.End[0], Y: e.End[1]}, Modifier: st.Modifier{Shift: e.Shift, Ctrl: e.Ctrl}})
	case "text":
		return deliver(inputChannel[st.CommandText](ch), st.CommandText{Text: e.Text})
	case "wheel_up", "wheel_down":
		value := st.WheelUp
		if e.Kind == "wheel_down" {
			value = st.WheelDown
		}
		return deliver(inputChannel[st.Wheel](ch), value)
	case "modifier":
		return deliver(inputChannel[st.Modifier](ch), st.Modifier{Shift: e.Shift, Ctrl: e.Ctrl})
	case "key":
		var event key.Event
		event.Direction = key.DirRelease
		switch strings.ToUpper(e.Text) {
		case "ENTER":
			event.Code = key.CodeReturnEnter
		case "BACKSPACE":
			event.Code = key.CodeDeleteBackspace
		default:
			r := []rune(strings.ToLower(e.Text))
			if len(r) != 1 {
				return fmt.Errorf("key requires one letter or digit, ENTER, or BACKSPACE")
			}
			event.Rune = r[0]
			if r[0] >= 'a' && r[0] <= 'z' {
				event.Code = key.CodeA + key.Code(r[0]-'a')
			}
		}
		return deliver(inputChannel[st.Key](ch), st.Key(event))
	default:
		return fmt.Errorf("unknown input kind %q", e.Kind)
	}
}

func (w *editWindow) interactive(command func(st.Commander) chan bool, inputs []editInput) error {
	w.waiting = make(chan []any)
	w.abandon = make(chan struct{})
	defer close(w.abandon)
	w.done = make(chan struct{})
	w.failure = make(chan error, 1)
	w.prompts = make(map[chan st.CommandText]string)
	quit := command(w)
	if quit == nil {
		if len(inputs) != 0 {
			return fmt.Errorf("command accepts no further inputs")
		}
		return nil
	}
	index := 0
	for {
		select {
		case <-w.done:
			select {
			case err := <-w.failure:
				return err
			default:
			}
			if index < len(inputs) {
				return fmt.Errorf("command completed before input %d", index+1)
			}
			return nil
		case channels := <-w.waiting:
			if index == len(inputs) {
				prompt := w.prompts[inputChannel[st.CommandText](channels)]
				err := deliver(quit, true)
				if err != nil {
					return err
				}
				select {
				case <-w.done:
				case <-time.After(5 * time.Second):
					return fmt.Errorf("interactive command did not stop")
				}
				if prompt != "" {
					return fmt.Errorf("missing text input: %s", prompt)
				}
				return fmt.Errorf("command needs one of %s; history: %s", availableInputs(channels), w.log.String())
			}
			if err := w.sendInput(channels, inputs[index]); err != nil {
				_ = deliver(quit, true)
				select {
				case <-w.done:
				case <-time.After(5 * time.Second):
					return fmt.Errorf("interactive command did not stop")
				}
				return fmt.Errorf("input %d: %w", index+1, err)
			}
			index++
		case <-time.After(30 * time.Second):
			return fmt.Errorf("interactive command did not report its next input")
		}
	}
}

func (w *editWindow) SelectedElems() []*st.Elem     { return w.selection.SelectedElems() }
func (w *editWindow) SelectedNodes() []*st.Node     { return w.selection.SelectedNodes() }
func (w *editWindow) SelectElem(v []*st.Elem)       { w.selection.SelectElem(v) }
func (w *editWindow) SelectNode(v []*st.Node)       { w.selection.SelectNode(v) }
func (w *editWindow) ElemSelected() bool            { return w.selection.ElemSelected() }
func (w *editWindow) NodeSelected() bool            { return w.selection.NodeSelected() }
func (w *editWindow) Deselect()                     { w.selection.Deselect() }
func (w *editWindow) DeselectElem()                 { w.selection.DeselectElem() }
func (w *editWindow) DeselectNode()                 { w.selection.DeselectNode() }
func (w *editWindow) ToggleAltSelectNode()          { w.alt = !w.alt }
func (w *editWindow) CurrentPointerPosition() []int { return w.pointer[:] }
func (w *editWindow) GetCanvasSize() (int, int)     { return 1024, 768 }

func (w *editWindow) SendNode(*st.Node)         {}
func (w *editWindow) SendElem(*st.Elem)         {}
func (w *editWindow) CanvasFitScale() float64   { return 0.9 }
func (w *editWindow) DefaultColorMode() uint    { return 0 }
func (w *editWindow) SetColorMode(uint)         {}
func (w *editWindow) SkipExFilenameCompletion() {}

func availableInputs(channels []any) string {
	var names []string
	for _, c := range channels {
		switch c.(type) {
		case chan *st.Node:
			names = append(names, "node")
		case chan *st.Elem:
			names = append(names, "element")
		case chan *st.Axis:
			names = append(names, "axis")
		case chan st.Click:
			names = append(names, "click/finish")
		case chan st.FenceGesture:
			names = append(names, "fence")
		case chan st.Key:
			names = append(names, "key")
		case chan st.Modifier:
			names = append(names, "modifier")
		case chan st.Wheel:
			names = append(names, "wheel_up/wheel_down")
		case chan st.CommandText:
			names = append(names, "text")
		}
	}
	return strings.Join(names, ", ")
}
