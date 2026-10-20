package stgio

import (
	"math"
	"runtime"
	"strings"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	st "github.com/yofu/st/stlib"
)

func interactiveFixture(t *testing.T) *uiHarness {
	h := mouseFixture(t)
	h.w.SetCanvasAnimateSpeed(0)
	h.w.showCanvasCenter()
	h.frame()
	st.Snapshot(h.w)
	return h
}
func awaitInteraction(t *testing.T, h *uiHarness) *interactiveCommand {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if s := h.w.interaction.Load(); s != nil {
			s.mu.Lock()
			ready := s.wait.waiting
			s.mu.Unlock()
			if ready {
				h.frame()
				h.frame()
				return s
			}
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("interactive command did not reach an input wait")
	return nil
}
func flushInteraction(t *testing.T, h *uiHarness, s *interactiveCommand) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for len(h.w.canvasEvents) > 0 && time.Now().Before(deadline) {
		h.frame()
		time.Sleep(time.Millisecond)
	}
	ack := s.offer(commandInput{kind: "barrier"})
	select {
	case <-ack:
	case <-time.After(3 * time.Second):
		buf := make([]byte, 12000)
		n := runtime.Stack(buf, true)
		t.Logf("%s", buf[:n])
		t.Fatal("interactive input stalled")
	}
	h.frame()
}
func startCommand(t *testing.T, h *uiHarness, name string) *interactiveCommand {
	t.Helper()
	h.w.submitCommand(name)
	return awaitInteraction(t, h)
}
func clickInteractive(t *testing.T, h *uiHarness, s *interactiveCommand, p f32.Point, button pointer.Buttons) {
	t.Helper()
	routedMouse(h, pointer.Press, button, p, 0, 0)
	routedMouse(h, pointer.Release, 0, p, 0, 0)
	flushInteraction(t, h, s)
}
func pointOf(n *st.Node) f32.Point { return f32.Pt(float32(n.Pcoord[0]), float32(n.Pcoord[1])) }
func sessionDone(t *testing.T, s *interactiveCommand) {
	t.Helper()
	select {
	case <-s.done:
	case <-time.After(3 * time.Second):
		t.Fatal("command did not finish")
	}
}

func TestInteractiveDistanceAndTail(t *testing.T) {
	h := interactiveFixture(t)
	w := h.w
	w.SelectNode([]*st.Node{w.frame.Nodes[3]})
	s := startCommand(t, h, "DISTS")
	clickInteractive(t, h, s, pointOf(w.frame.Nodes[1]), pointer.ButtonPrimary)
	if len(w.ui.commandTail) != 1 || len(w.SelectedNodes()) != 1 {
		t.Fatal("first node did not start a tail")
	}
	clickInteractive(t, h, s, pointOf(w.frame.Nodes[2]), pointer.ButtonPrimary)
	sessionDone(t, s)
	if w.Executing() || len(w.ui.commandTail) != 0 {
		t.Fatal("distance command left active input or tail")
	}
	w.ui.mu.Lock()
	log := w.ui.log
	w.ui.mu.Unlock()
	if !strings.Contains(log, "DX: 0.000 DY: 2.000 DZ: 3.000 D: 3.606") {
		t.Fatalf("incorrect distance output: %s", log)
	}
}

func TestInteractiveAddLineAndUndo(t *testing.T) {
	h := interactiveFixture(t)
	w := h.w
	count := len(w.frame.Elems)
	s := startCommand(t, h, "ADDLINEELEM")
	clickInteractive(t, h, s, pointOf(w.frame.Nodes[1]), pointer.ButtonPrimary)
	clickInteractive(t, h, s, pointOf(w.frame.Nodes[3]), pointer.ButtonPrimary)
	sessionDone(t, s)
	if len(w.frame.Elems) != count+1 || !w.changed {
		t.Fatal("line was not created")
	}
	f, err := w.Undo()
	if err != nil || len(f.Elems) != count {
		t.Fatalf("line undo failed: %v", err)
	}
}

func TestInteractiveEraseCommitAndEscape(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		h := interactiveFixture(t)
		w := h.w
		el := w.frame.Elems[1]
		s := startCommand(t, h, "ERASE")
		p := pointOf(el.Enod[0]).Add(pointOf(el.Enod[1])).Mul(0.5)
		clickInteractive(t, h, s, p, pointer.ButtonPrimary)
		if len(w.SelectedElems()) != 1 {
			t.Fatal("member input did not select erase target")
		}
		if cancel {
			h.key(key.NameEscape)
			sessionDone(t, s)
			h.frame()
			if w.frame.Elems[1] == nil {
				t.Fatal("Esc committed deletion")
			}
		} else {
			clickInteractive(t, h, s, p, pointer.ButtonSecondary)
			sessionDone(t, s)
			if w.frame.Elems[1] != nil {
				t.Fatal("right click did not commit deletion")
			}
		}
	}
}

func TestInteractiveMoveNodeWithReferencePoints(t *testing.T) {
	h := interactiveFixture(t)
	w := h.w
	n := w.frame.Nodes[3]
	before := append([]float64(nil), n.Coord...)
	w.SelectNode([]*st.Node{n})
	s := startCommand(t, h, "MOVENODE")
	if !h.router.Source().Focused(&w.pointerTag) {
		t.Fatalf("command canvas did not receive keyboard focus; editor focused=%v", h.router.Source().Focused(&w.ui.command))
	}
	clickInteractive(t, h, s, pointOf(w.frame.Nodes[1]), pointer.ButtonPrimary)
	clickInteractive(t, h, s, pointOf(w.frame.Nodes[2]), pointer.ButtonPrimary)
	if math.Abs(n.Coord[1]-before[1]-2) > 1e-9 || math.Abs(n.Coord[2]-before[2]-3) > 1e-9 {
		t.Fatalf("reference displacement was not applied: %v", n.Coord)
	}
	clickInteractive(t, h, s, f32.Pt(300, 300), pointer.ButtonSecondary)
	sessionDone(t, s)
}

func TestInteractiveKeyWheelAndCancellation(t *testing.T) {
	h := interactiveFixture(t)
	w := h.w
	target := w.frame.Nodes[3]
	before := target.Coord[0]
	w.SelectNode([]*st.Node{target})
	s := startCommand(t, h, "MOVENODE")
	// Shared commands use key release to select the numeric X/Y/Z direction.
	h.router.Queue(key.Event{Name: "X", State: key.Press}, key.Event{Name: "X", State: key.Release})
	h.frame()
	flushInteraction(t, h, s)
	s.mu.Lock()
	wants := s.wait.wheel != nil
	s.mu.Unlock()
	if !wants {
		t.Fatal("X key did not enable command wheel input")
	}
	scale := w.frame.View.Gfact
	h.router.Queue(pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Position: f32.Pt(300, 300+float32(w.ui.canvasRect.Min.Y)), Scroll: f32.Pt(0, -20)})
	h.frame()
	flushInteraction(t, h, s)
	h.router.Queue(key.Event{Name: key.NameReturn, State: key.Press}, key.Event{Name: key.NameReturn, State: key.Release})
	h.frame()
	flushInteraction(t, h, s)
	if math.Abs(target.Coord[0]-before-1) > 1e-9 {
		t.Fatalf("numeric wheel displacement failed: %v", target.Coord)
	}
	if w.frame.View.Gfact != scale {
		t.Fatal("command wheel input also zoomed the model")
	}
	h.key(key.NameEscape)
	sessionDone(t, s)
	h.frame()
	if w.Executing() || len(w.ui.commandTail) > 0 {
		t.Fatal("Esc did not release command")
	}
}

func TestInteractiveReplacementAndInputQueue(t *testing.T) {
	h := interactiveFixture(t)
	w := h.w
	s := startCommand(t, h, "DISTS")
	w.submitCommand("ADDLINEELEM")
	sessionDone(t, s)
	next := awaitInteraction(t, h)
	if next == s {
		t.Fatal("new command reused old session")
	}
	// Queue both clicks without waiting between them.
	p1, p2 := pointOf(w.frame.Nodes[1]), pointOf(w.frame.Nodes[3])
	for _, p := range []f32.Point{p1, p2} {
		routedMouse(h, pointer.Press, pointer.ButtonPrimary, p, 0, 0)
		routedMouse(h, pointer.Release, 0, p, 0, 0)
	}
	flushInteraction(t, h, next)
	sessionDone(t, next)
	if len(w.frame.Elems) != 3 {
		t.Fatal("queued node inputs were lost")
	}
}

func TestInteractivePropertyTwoStageAndEscape(t *testing.T) {
	h := interactiveFixture(t)
	w := h.w
	a, b := w.frame.Elems[1], w.frame.Elems[2]
	s := startCommand(t, h, "MATCHPROPERTY")
	center := func(el *st.Elem) f32.Point { return pointOf(el.Enod[0]).Add(pointOf(el.Enod[1])).Mul(0.5) }
	clickInteractive(t, h, s, center(a), pointer.ButtonPrimary)
	clickInteractive(t, h, s, center(b), pointer.ButtonPrimary)
	if b.Etype != a.Etype || b.Sect != a.Sect {
		t.Fatal("second-stage clicked element was not delivered")
	}
	clickInteractive(t, h, s, f32.Pt(300, 300), pointer.ButtonSecondary)
	sessionDone(t, s)
	// Cancel during the click-only phase without supplying the second element.
	s = startCommand(t, h, "MATCHPROPERTY")
	clickInteractive(t, h, s, center(a), pointer.ButtonPrimary)
	h.key(key.NameEscape)
	sessionDone(t, s)
}

func TestInteractivePlateAndToggleBond(t *testing.T) {
	h := interactiveFixture(t)
	w := h.w
	f := w.frame
	// Make the third point non-collinear for a real plate.
	f.Nodes[3].Coord[0] = 1
	w.Redraw()
	h.frame()
	s := startCommand(t, h, "ADDPLATEELEM")
	for i := 1; i <= 3; i++ {
		clickInteractive(t, h, s, pointOf(f.Nodes[i]), pointer.ButtonPrimary)
	}
	if len(f.Elems) != 2 {
		t.Fatal("plate was committed before right click")
	}
	clickInteractive(t, h, s, f32.Pt(300, 300), pointer.ButtonSecondary)
	sessionDone(t, s)
	if len(f.Elems) != 3 {
		t.Fatal("plate was not committed")
	}
	w.SelectElem([]*st.Elem{f.Elems[1]})
	s = startCommand(t, h, "TOGGLEBOND")
	clickInteractive(t, h, s, pointOf(f.Nodes[1]), pointer.ButtonPrimary)
	if f.Elems[1].BondState() == 0 {
		t.Fatal("node input did not toggle selected member bond")
	}
	h.key(key.NameEscape)
	sessionDone(t, s)
}

// Every registered shared command must release its renderer lock while waiting
// and accept cancellation, including commands with no selected targets yet.
func TestInteractiveAllSharedCommandsCancel(t *testing.T) {
	for name := range st.Commands {
		t.Run(name, func(t *testing.T) {
			h := interactiveFixture(t)
			// Some registered operations finish synchronously without input.
			h.w.submitCommand(name)
			waitUI(t, h.w)
			s := h.w.interaction.Load()
			if s == nil {
				return
			}
			deadline := time.Now().Add(3 * time.Second)
			for time.Now().Before(deadline) {
				select {
				case <-s.done:
					return
				default:
				}
				s.mu.Lock()
				ready := s.wait.waiting
				s.mu.Unlock()
				if ready {
					break
				}
				h.frame()
				time.Sleep(time.Millisecond)
			}
			h.frame()
			h.frame()
			if !h.w.ui.modelMu.TryLock() {
				t.Fatal("input wait retained model lock")
			}
			h.w.ui.modelMu.Unlock()
			h.key(key.NameEscape)
			sessionDone(t, s)
		})
	}
}

func TestInteractiveOffsetSideClick(t *testing.T) {
	h := interactiveFixture(t)
	w := h.w
	s := startCommand(t, h, "OFFSET")
	el := w.frame.Elems[1]
	midpoint := pointOf(el.Enod[0]).Add(pointOf(el.Enod[1])).Mul(0.5)
	clickInteractive(t, h, s, midpoint, pointer.ButtonPrimary)
	s.mu.Lock()
	wantsElem := s.wait.elem != nil
	s.mu.Unlock()
	if wantsElem {
		t.Fatal("offset did not switch from reference to side click")
	}
	h.router.Queue(key.Event{Name: "V", State: key.Release}, key.Event{Name: "K", State: key.Release})
	h.frame()
	flushInteraction(t, h, s)
	count := len(w.frame.Elems)
	clickInteractive(t, h, s, midpoint.Add(f32.Pt(30, 0)), pointer.ButtonPrimary)
	if len(w.frame.Elems) != count+1 {
		t.Fatal("offset side click was not applied")
	}
	h.key(key.NameEscape)
	sessionDone(t, s)
}

func TestInteractiveHoverAndFrameReplacement(t *testing.T) {
	h := interactiveFixture(t)
	w := h.w
	s := startCommand(t, h, "DISTS")
	p := f32.Pt(230, 220)
	routedMouse(h, pointer.Move, 0, p, 0, 0)
	flushInteraction(t, h, s)
	if pos := s.CurrentPointerPosition(); pos[0] != 230 || pos[1] != 220 {
		t.Fatalf("hover position = %v", pos)
	}
	if w.ui.cursorPosition != p {
		t.Fatalf("tail endpoint = %v", w.ui.cursorPosition)
	}
	h.key(key.NameEscape)
	sessionDone(t, s)
	// A synchronous alias can replace the model and immediately query it again.
	w.ui.modelMu.Lock()
	w.startInteractiveCommand(func(host st.Commander) chan bool {
		f := st.NewFrame()
		host.SetFrame(f)
		if host.Frame() != f {
			t.Error("alias retained previous model")
		}
		return nil
	})
	w.ui.modelMu.Unlock()
	if w.Executing() {
		t.Fatal("synchronous alias remained active")
	}
}

func TestInteractiveCommandFieldShortcut(t *testing.T) {
	h := interactiveFixture(t)
	s := startCommand(t, h, "DISTS")
	h.router.Queue(key.Event{Name: "L", State: key.Press, Modifiers: key.ModCtrl})
	h.frame()
	h.frame()
	if !h.router.Source().Focused(&h.w.ui.command) {
		t.Fatal("Ctrl+L did not return focus to command field")
	}
	// Esc from the editor still terminates the canvas command.
	h.key(key.NameEscape)
	sessionDone(t, s)
}
