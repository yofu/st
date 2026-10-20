package stgio

import (
	"reflect"
	"sync"
	"sync/atomic"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	st "github.com/yofu/st/stlib"
	mobilekey "golang.org/x/mobile/event/key"
)

type commandInput struct {
	gesture     st.FenceGesture
	kind        string
	position    f32.Point
	node        *st.Node
	elem        *st.Elem
	axis        *st.Axis
	click       st.Click
	mod         st.Modifier
	key         st.Key
	wheel       st.Wheel
	preferNodes bool
	ack         chan struct{}
}
type commandWait struct {
	fence           chan st.FenceGesture
	node            chan *st.Node
	elem            chan *st.Elem
	axis            chan *st.Axis
	click           chan st.Click
	mod             chan st.Modifier
	key             chan st.Key
	wheel           chan st.Wheel
	pos             chan st.Position
	version         uint64
	waiting, direct bool
}
type interactiveCommand struct {
	*Window
	quit                 chan bool
	done, cancel, notify chan struct{}
	input                chan commandInput
	stop, finish         sync.Once
	mu                   sync.Mutex
	wait                 commandWait
	position             f32.Point
	fenceMode            atomic.Bool
	alt                  bool
	tail                 []*st.Node
	ended                bool
	accepting            bool
}
type interactiveScope struct {
	s    *interactiveCommand
	held bool
}

func (w *Window) startInteractiveCommand(f func(st.Commander) chan bool) {
	if w.interaction.Load() != nil {
		w.History("対話型コマンドの終了待ちです。")
		return
	}
	s := &interactiveCommand{Window: w, done: make(chan struct{}), cancel: make(chan struct{}), notify: make(chan struct{}, 1), input: make(chan commandInput, 128), alt: w.AltSelectNode(), accepting: true}
	w.interaction.Store(s)
	w.interactionFocus.Store(true)
	s.quit = f(s)
	if s.quit == nil {
		s.complete()
		return
	}
	go s.pump()
	w.History("対話入力: 左クリックで指定、右クリックで確定・終了、Escで中止。")
	w.QueueRedrawAll()
}

func (s *interactiveCommand) signal() {
	select {
	case s.notify <- struct{}{}:
	default:
	}
}
func (s *interactiveCommand) cancelCommand() { s.stop.Do(func() { close(s.cancel) }); s.signal() }
func (w *Window) stopInteractiveCommand() {
	if s := w.interaction.Load(); s != nil {
		s.cancelCommand()
		<-s.done
	}
}
func (w *Window) Executing() bool { return w.interaction.Load() != nil }

func (s *interactiveCommand) BeginInteractiveCommand() st.InteractiveScope {
	s.ui.modelMu.Lock()
	return &interactiveScope{s: s, held: true}
}
func (g *interactiveScope) Resume() {
	if !g.held {
		g.s.ui.modelMu.Lock()
		g.held = true
	}
	g.s.mu.Lock()
	g.s.wait.waiting = false
	g.s.mu.Unlock()
}
func (g *interactiveScope) Wait(channels ...any)       { g.wait(false, channels...) }
func (g *interactiveScope) WaitDirect(channels ...any) { g.wait(true, channels...) }
func (g *interactiveScope) wait(direct bool, channels ...any) {
	s := g.s
	s.mu.Lock()
	next := commandWait{version: s.wait.version + 1, waiting: true, direct: direct}
	for _, c := range channels {
		switch c := c.(type) {
		case chan st.FenceGesture:
			next.fence = c
		case chan *st.Node:
			next.node = c
		case chan *st.Elem:
			next.elem = c
		case chan *st.Axis:
			next.axis = c
		case chan st.Click:
			next.click = c
		case chan st.Modifier:
			next.mod = c
		case chan st.Key:
			next.key = c
		case chan st.Wheel:
			next.wheel = c
		case chan st.Position:
			next.pos = c
		}
	}
	s.wait = next
	s.mu.Unlock()
	if g.held {
		g.held = false
		s.ui.modelMu.Unlock()
	}
	s.signal()
	s.QueueRedrawAll()
}
func (g *interactiveScope) Close() {
	if !g.held {
		g.s.ui.modelMu.Lock()
		g.held = true
	}
	g.s.complete()
	g.held = false
	g.s.ui.modelMu.Unlock()
}
func (s *interactiveCommand) complete() {
	s.finish.Do(func() {
		s.mu.Lock()
		s.accepting = false
		s.mu.Unlock()
		s.ended = true
		s.tail = nil
		s.Window.interaction.CompareAndSwap(s, nil)
		s.Window.commandFocus.Store(true)
		s.Window.cancelCanvasDrag()
		s.Window.sceneDirty.Store(true)
		close(s.done)
		s.signal()
		s.QueueRedrawAll()
	})
}
func (s *interactiveCommand) GetFenceGesture() chan st.FenceGesture {
	s.fenceMode.Store(true)
	return make(chan st.FenceGesture)
}

func (s *interactiveCommand) EndCommand()                   { s.ended = true }
func (s *interactiveCommand) Execute(chan bool)             {}
func (s *interactiveCommand) Executing() bool               { return !s.ended }
func (s *interactiveCommand) Frame() *st.Frame              { return s.Window.frame }
func (s *interactiveCommand) GetNode() chan *st.Node        { return make(chan *st.Node) }
func (s *interactiveCommand) GetElem() chan *st.Elem        { return make(chan *st.Elem) }
func (s *interactiveCommand) GetAxis() chan *st.Axis        { return make(chan *st.Axis) }
func (s *interactiveCommand) GetClick() chan st.Click       { return make(chan st.Click) }
func (s *interactiveCommand) GetModifier() chan st.Modifier { return make(chan st.Modifier) }
func (s *interactiveCommand) GetKey() chan st.Key           { return make(chan st.Key) }
func (s *interactiveCommand) GetWheel() chan st.Wheel       { return make(chan st.Wheel) }
func (s *interactiveCommand) StopWheel()                    {}
func (s *interactiveCommand) GetPosition() chan st.Position { return make(chan st.Position) }
func (s *interactiveCommand) SetAltSelectNode(v bool)       { s.alt = v }
func (s *interactiveCommand) AltSelectNode() bool           { return s.alt }
func (s *interactiveCommand) AddTail(n *st.Node)            { s.tail = append(s.tail, n); s.Redraw() }
func (s *interactiveCommand) EndTail()                      { s.tail = nil; s.Redraw() }
func (s *interactiveCommand) CurrentPointerPosition() []int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return []int{int(s.position.X), int(s.position.Y)}
}

func (s *interactiveCommand) offer(e commandInput) <-chan struct{} {
	e.ack = make(chan struct{})
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.accepting {
		close(e.ack)
		return e.ack
	}
	select {
	case <-s.cancel:
		close(e.ack)
		return e.ack
	default:
	}
	select {
	case <-s.done:
		close(e.ack)
	case <-s.cancel:
		close(e.ack)
	case s.input <- e:
	default:
		s.History("対話型コマンドの入力待ちが多すぎるため終了します。")
		s.cancelCommand()
		close(e.ack)
	}
	return e.ack
}
func (s *interactiveCommand) stateAfter(version uint64) (commandWait, bool) {
	for {
		s.mu.Lock()
		v := s.wait
		s.mu.Unlock()
		if v.waiting && v.version > version {
			return v, true
		}
		select {
		case <-s.done:
			return v, false
		case <-s.cancel:
			return v, false
		case <-s.notify:
		}
	}
}
func (s *interactiveCommand) send(ch any, value any) bool {
	if ch == nil || reflect.ValueOf(ch).IsNil() {
		return false
	}
	cases := []reflect.SelectCase{{Dir: reflect.SelectSend, Chan: reflect.ValueOf(ch), Send: reflect.ValueOf(value)}, {Dir: reflect.SelectRecv, Chan: reflect.ValueOf(s.cancel)}, {Dir: reflect.SelectRecv, Chan: reflect.ValueOf(s.done)}}
	i, _, _ := reflect.Select(cases)
	return i == 0
}
func (s *interactiveCommand) pump() {
	defer func() {
		s.mu.Lock()
		s.accepting = false
		s.mu.Unlock()
		for {
			select {
			case e := <-s.input:
				close(e.ack)
			default:
				return
			}
		}
	}()
	for {
		select {
		case <-s.done:
			return
		case <-s.cancel:
			select {
			case s.quit <- true:
			case <-s.done:
			}
			return
		case e := <-s.input:
			s.dispatch(e)
			close(e.ack)
		}
	}
}
func (s *interactiveCommand) dispatch(e commandInput) {
	v, ok := s.stateAfter(0)
	if !ok {
		return
	}
	if e.kind == "barrier" {
		return
	}
	if e.kind != "key" {
		s.mu.Lock()
		s.position = e.position
		s.mu.Unlock()
	}
	if e.kind == "fence" {
		if v.fence != nil && s.send(v.fence, e.gesture) {
			s.stateAfter(v.version)
		}
		return
	}
	if e.kind == "key" {
		if v.key != nil && s.send(v.key, e.key) {
			s.stateAfter(v.version)
		}
		return
	}
	if e.kind == "wheel" {
		if v.wheel != nil && s.send(v.wheel, e.wheel) {
			s.stateAfter(v.version)
		}
		return
	}
	if e.kind == "position" {
		if v.pos != nil && s.send(v.pos, st.Position{X: int(e.position.X), Y: int(e.position.Y)}) {
			s.stateAfter(v.version)
		}
		return
	}
	if v.mod != nil {
		if !s.send(v.mod, e.mod) {
			return
		}
		v, ok = s.stateAfter(v.version)
		if !ok {
			return
		}
	}
	var sent bool
	if e.click.Button == st.ButtonLeft {
		switch {
		case v.node != nil && (v.elem == nil || e.preferNodes) && e.node != nil:
			sent = s.send(v.node, e.node)
		case v.elem != nil && e.elem != nil:
			sent = s.send(v.elem, e.elem)
		case v.axis != nil && e.axis != nil:
			sent = s.send(v.axis, e.axis)
		case v.click != nil:
			sent = s.send(v.click, e.click)
		case v.node != nil:
			sent = s.send(v.node, (*st.Node)(nil))
		case v.elem != nil:
			sent = s.send(v.elem, (*st.Elem)(nil))
		}
	} else if v.click != nil {
		sent = s.send(v.click, e.click)
	}
	if !sent {
		return
	}
	// A click-only phase may immediately request the clicked element as a
	// second-stage receive (Trim/Extend/MatchProperty/CopyBond).
	if next, ok := s.stateAfter(v.version); ok && next.direct && next.elem != nil {
		if s.send(next.elem, e.elem) {
			s.stateAfter(next.version)
		}
	}
}

func (w *Window) interactiveClick(e pointer.Event, button pointer.Buttons) bool {
	s := w.interaction.Load()
	if s == nil {
		return false
	}
	if s.fenceMode.Load() && button&pointer.ButtonPrimary != 0 {
		s.offer(commandInput{kind: "fence", position: e.Position, gesture: st.FenceGesture{
			Start:    st.Position{X: int(w.start.X), Y: int(w.start.Y)},
			End:      st.Position{X: int(e.Position.X), Y: int(e.Position.Y)},
			Modifier: st.Modifier{Shift: e.Modifiers&key.ModShift != 0, Ctrl: e.Modifiers&key.ModCtrl != 0},
		}})
		return true
	}
	c := st.ClickLeft(int(e.Position.X), int(e.Position.Y))
	if button&pointer.ButtonSecondary != 0 {
		c = st.ClickRight(c.X, c.Y)
	} else if button&pointer.ButtonTertiary != 0 {
		c = st.ClickMiddle(c.X, c.Y)
	}
	x, y := float64(e.Position.X), float64(e.Position.Y)
	n := w.frame.PickNode(x, y, 15)
	el := w.frame.PickElem(x, y, 5)
	a, ind := w.frame.PickAxis(x, y, 5)
	if a != nil {
		a.Current = ind
	}
	s.offer(commandInput{kind: "click", position: e.Position, node: n, elem: el, axis: a, click: c, mod: st.Modifier{Shift: e.Modifiers&key.ModShift != 0, Ctrl: e.Modifiers&key.ModCtrl != 0, Alt: e.Modifiers&key.ModAlt != 0}, preferNodes: (e.Modifiers&key.ModAlt != 0) == s.alt})
	return true
}

// Hover positions also reach commands that request GetPosition. The latest
// position stays available to wheel/key handlers without queueing unused moves.
func (w *Window) interactivePosition(p f32.Point) {
	s := w.interaction.Load()
	if s == nil {
		return
	}
	s.mu.Lock()
	s.position = p
	wants := s.wait.pos != nil
	s.mu.Unlock()
	if wants {
		s.offer(commandInput{kind: "position", position: p})
	}
}

func (w *Window) interactiveWheel(e pointer.Event) bool {
	s := w.interaction.Load()
	if s == nil {
		return false
	}
	s.mu.Lock()
	wants := s.wait.wheel != nil
	s.mu.Unlock()
	if !wants {
		return false
	}
	d := st.WheelDown
	if e.Scroll.Y < 0 {
		d = st.WheelUp
	}
	s.offer(commandInput{kind: "wheel", position: e.Position, wheel: d})
	return true
}

func (w *Window) interactiveKeyboard(gtx layout.Context) {
	if w.commandFocus.Swap(false) {
		w.ui.focusCommand = true
	}
	s := w.interaction.Load()
	if s == nil {
		return
	}
	w.ui.mu.Lock()
	blocked := w.ui.question != nil
	w.ui.mu.Unlock()
	if blocked || w.ui.modal != nil {
		return
	}
	if w.interactionFocus.Swap(false) {
		w.ui.focusCommand = false
		gtx.Execute(key.FocusCmd{Tag: &w.pointerTag})
	}
	for {
		ev, ok := gtx.Event(key.FocusFilter{Target: &w.pointerTag}, key.Filter{Focus: &w.pointerTag, Optional: key.ModShift | key.ModCtrl | key.ModAlt | key.ModSuper}, key.Filter{Focus: &w.pointerTag, Name: key.NameEscape}, key.Filter{Focus: &w.pointerTag, Name: key.NameReturn}, key.Filter{Focus: &w.pointerTag, Name: key.NameTab})
		if !ok {
			break
		}
		k, ok := ev.(key.Event)
		if !ok {
			continue
		}
		// Focused canvas filters receive these before the global shortcuts.
		// Keep command entry and file operations available during input waits.
		if k.Modifiers&key.ModCtrl != 0 {
			global := true
			switch k.Name {
			case "L", "O", "S", "I", "M", "R":
			default:
				global = false
			}
			if global {
				if k.State == key.Press {
					switch k.Name {
					case "L":
						gtx.Execute(key.FocusCmd{Tag: &w.ui.command})
					case "O":
						w.fileDialog("Open")
					case "S":
						w.menuAction("save")
					case "I":
						w.fileDialog("Insert")
					case "M":
						w.menuAction("reload")
					case "R":
						w.menuAction("readall")
					}
				}
				continue
			}
		}
		if k.Name == key.NameEscape && k.State == key.Press {
			s.cancelCommand()
			w.ui.selectionOverlay = false
			w.deselectOnEscape()
			continue
		}
		m := mobilekey.Modifiers(0)
		if k.Modifiers&key.ModShift != 0 {
			m |= mobilekey.ModShift
		}
		if k.Modifiers&key.ModCtrl != 0 {
			m |= mobilekey.ModControl
		}
		if k.Modifiers&key.ModAlt != 0 {
			m |= mobilekey.ModAlt
		}
		if k.Modifiers&key.ModSuper != 0 {
			m |= mobilekey.ModMeta
		}
		d := mobilekey.DirPress
		if k.State == key.Release {
			d = mobilekey.DirRelease
		}
		code := mobilekey.CodeUnknown
		r := rune(0)
		rs := []rune(k.Name)
		if len(rs) == 1 {
			r = rs[0]
			if r >= 'A' && r <= 'Z' {
				code = mobilekey.Code(int(mobilekey.CodeA) + int(r-'A'))
				if k.Modifiers&key.ModShift == 0 {
					r += 'a' - 'A'
				}
			}
		}
		switch k.Name {
		case key.NameReturn:
			code = mobilekey.CodeReturnEnter
		case key.NameEscape:
			code = mobilekey.CodeEscape
		case key.NameTab:
			code = mobilekey.CodeTab
		case key.NameDeleteBackward:
			code = mobilekey.CodeDeleteBackspace
		case key.NameDeleteForward:
			code = mobilekey.CodeDeleteForward
		}
		s.offer(commandInput{kind: "key", key: st.Key{Rune: r, Code: code, Modifiers: m, Direction: d}})
	}
}

func (w *Window) snapshotCommandTail() {
	w.ui.commandTail = nil
	if s := w.interaction.Load(); s != nil {
		for _, n := range s.tail {
			if n != nil {
				p := n.Pcoord
				if w.frame.Show.PlotState&st.PLOT_UNDEFORMED == 0 {
					p = n.Dcoord
				}
				w.ui.commandTail = append(w.ui.commandTail, f32.Pt(float32(p[0]), float32(p[1])))
			}
		}
	}
}

var _ st.Commander = (*interactiveCommand)(nil)

func (w *Window) fenceCommandActive() bool {
	s := w.interaction.Load()
	return s != nil && s.fenceMode.Load()
}
