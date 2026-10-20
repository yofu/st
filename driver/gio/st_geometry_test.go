package stgio

import (
	"math"
	"strings"
	"testing"
	"time"

	"gioui.org/io/key"
	"gioui.org/io/pointer"
	st "github.com/yofu/st/stlib"
)

func geometryFixture(t *testing.T) *uiHarness {
	h := interactiveFixture(t)
	w := h.w
	f := w.frame
	coords := [][]float64{{0, 0, 0}, {2, 0, 0}, {0, 2, 0}, {0, 0, 2}, {1, 1, 1}, {2, 2, 0}, {1, 1, 3}}
	for i, c := range coords {
		n := f.Nodes[i+1]
		if n == nil {
			n = st.NewNode()
			n.Num = i + 1
			n.Frame = f
			f.Nodes[n.Num] = n
		}
		n.Coord = c
	}
	f.View.Angle = []float64{20, 35}
	w.showCanvasCenter()
	w.Redraw()
	h.frame()
	// Replace fixture undo history with an exact baseline for this geometry.
	w.UndoStack = st.NewUndoStack(10)
	st.Snapshot(w)
	w.changed = false
	return h
}
func pickGeometry(t *testing.T, h *uiHarness, s *interactiveCommand, num int) {
	t.Helper()
	// Feed the same picked-node event as the router. Perspective picking itself
	// is covered by the routed two-node test below and existing command tests.
	n := h.w.frame.Nodes[num]
	ack := s.offer(commandInput{kind: "click", node: n, position: pointOf(n), click: st.ClickLeft(int(n.Pcoord[0]), int(n.Pcoord[1])), preferNodes: true})
	select {
	case <-ack:
	case <-time.After(3 * time.Second):
		t.Fatal("node input stalled")
	}
	h.frame()
}
func geometryKey(t *testing.T, h *uiHarness, s *interactiveCommand, name key.Name) {
	t.Helper()
	h.router.Queue(key.Event{Name: name, State: key.Press}, key.Event{Name: name, State: key.Release})
	h.frame()
	flushInteraction(t, h, s)
}
func awaitGeometryPrompt(t *testing.T, h *uiHarness, s *interactiveCommand) *pathDialog {
	t.Helper()
	end := time.Now().Add(3 * time.Second)
	for time.Now().Before(end) {
		h.frame()
		if d := h.w.ui.modal; d != nil && d.owner == s {
			h.frame()
			h.frame()
			return d
		}
		select {
		case <-s.done:
			t.Fatal("command finished before prompt")
		default:
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("text prompt did not appear")
	return nil
}
func answerGeometryPrompt(t *testing.T, h *uiHarness, s *interactiveCommand, text string) {
	t.Helper()
	d := awaitGeometryPrompt(t, h, s)
	s.mu.Lock()
	version := s.wait.version
	s.mu.Unlock()
	d.ed.SetText(text)
	d.ed.SetCaret(d.ed.Len(), d.ed.Len())
	h.key(key.NameReturn)
	end := time.Now().Add(3 * time.Second)
	for time.Now().Before(end) {
		h.frame()
		select {
		case <-s.done:
			h.frame()
			return
		default:
		}
		s.mu.Lock()
		advanced := s.wait.version > version && s.wait.waiting
		s.mu.Unlock()
		if advanced {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("prompt response did not advance command")
}
func assertCoordinate(t *testing.T, n *st.Node, want []float64) {
	t.Helper()
	for i, v := range want {
		if math.Abs(n.Coord[i]-v) > 1e-8 {
			t.Fatalf("node%d coordinate=%v want=%v", n.Num, n.Coord, want)
		}
	}
}
func TestElevenSelectionAndDeformedDistance(t *testing.T) {
	h := geometryFixture(t)
	w := h.w
	f := w.frame
	w.SelectNode([]*st.Node{f.Nodes[1], f.Nodes[2]})
	runPgpAlias(t, h, "NODETOELEM")
	if els := w.SelectedElems(); len(els) != 1 || els[0].Num != 1 {
		t.Fatalf("NODETOELEM selected %v", els)
	}
	w.DeselectNode()
	runPgpAlias(t, h, "ELEMTONODE")
	if len(w.SelectedNodes()) != 2 {
		t.Fatal("ELEMTONODE did not collect endpoints")
	}
	w.SelectNode([]*st.Node{f.Nodes[2]})
	runPgpAlias(t, h, "SEARCHELEM")
	if len(w.SelectedElems()) != 2 {
		t.Fatal("SEARCHELEM missed connected member")
	}
	w.Deselect()
	s := startCommand(t, h, "SEARCHELEM")
	geometryKey(t, h, s, "2")
	geometryKey(t, h, s, key.NameReturn)
	sessionDone(t, s)
	if len(w.SelectedElems()) != 2 {
		t.Fatal("numeric node input did not select members")
	}
	w.Deselect()
	f.Show.Period = "X"
	f.Show.Dfact = 100
	f.Nodes[2].Disp["X"] = []float64{0.5, 0, 0, 0, 0, 0}
	s = startCommand(t, h, "DEFORMEDDISTS")
	// Verify the real mouse route as well as the displayed period / true displacement.
	clickInteractive(t, h, s, pointOf(f.Nodes[1]), pointer.ButtonPrimary)
	clickInteractive(t, h, s, pointOf(f.Nodes[2]), pointer.ButtonPrimary)
	sessionDone(t, s)
	w.ui.mu.Lock()
	log := w.ui.log
	w.ui.mu.Unlock()
	if !strings.Contains(log, "DX: 2.000 DY: 0.000 DZ: 0.000 D: 2.000") || !strings.Contains(log, "dx: 2.500 dy: 0.000 dz: 0.000 d: 2.500") {
		t.Fatalf("distance output incorrect: %s", log)
	}
	if w.changed {
		t.Fatal("selection/measurement changed model")
	}
}
func TestElevenLinePlaneAndUndo(t *testing.T) {
	t.Run("line", func(t *testing.T) {
		h := geometryFixture(t)
		w := h.w
		target := w.frame.Nodes[5]
		w.SelectNode([]*st.Node{target})
		s := startCommand(t, h, "MOVETOLINE")
		pickGeometry(t, h, s, 1)
		pickGeometry(t, h, s, 2)
		sessionDone(t, s)
		assertCoordinate(t, target, []float64{1, 0, 0})
		f, err := w.Undo()
		if err != nil {
			t.Fatal(err)
		}
		assertCoordinate(t, f.Nodes[5], []float64{1, 1, 1})
	})
	t.Run("plane", func(t *testing.T) {
		h := geometryFixture(t)
		w := h.w
		target := w.frame.Nodes[5]
		w.SelectNode([]*st.Node{target})
		s := startCommand(t, h, "MOVETOPLANE")
		for _, i := range []int{1, 2, 3} {
			pickGeometry(t, h, s, i)
		}
		sessionDone(t, s)
		assertCoordinate(t, target, []float64{1, 1, 0})
	})
	t.Run("plane-axis", func(t *testing.T) {
		h := geometryFixture(t)
		w := h.w
		target := w.frame.Nodes[5]
		w.SelectNode([]*st.Node{target})
		s := startCommand(t, h, "MOVETOPLANE")
		geometryKey(t, h, s, "X")
		for _, i := range []int{1, 3, 4} {
			pickGeometry(t, h, s, i)
		}
		sessionDone(t, s)
		assertCoordinate(t, target, []float64{0, 1, 1})
	})
	t.Run("invalid-line-atomic", func(t *testing.T) {
		h := geometryFixture(t)
		w := h.w
		w.SelectNode([]*st.Node{w.frame.Nodes[5], w.frame.Nodes[7]})
		s := startCommand(t, h, "MOVETOLINE")
		pickGeometry(t, h, s, 1)
		pickGeometry(t, h, s, 3)
		sessionDone(t, s)
		assertCoordinate(t, w.frame.Nodes[5], []float64{1, 1, 1})
		assertCoordinate(t, w.frame.Nodes[7], []float64{1, 1, 3})
		if w.changed {
			t.Fatal("failed line move created undo state")
		}
	})
}
func TestElevenMeasureSnapshotAndCancel(t *testing.T) {
	for _, dir := range []string{"X", "Y", "Z", "V"} {
		t.Run(dir, func(t *testing.T) {
			h := geometryFixture(t)
			w := h.w
			s := startCommand(t, h, "MEASURE")
			pickGeometry(t, h, s, 1)
			pickGeometry(t, h, s, 6)
			geometryKey(t, h, s, key.Name(dir))
			sessionDone(t, s)
			if len(w.frame.Measures) != 1 || !w.frame.Show.Measure {
				t.Fatal("measure not created or hidden")
			}
			m := w.frame.Measures[0]
			want := "2000"
			if dir == "Z" || dir == "V" {
				want = "2828"
			}
			if m.Text != want {
				t.Fatalf("measure text=%s want=%s", m.Text, want)
			}
			snapshot := w.frame.Snapshot()
			if len(snapshot.Measures) != 1 || snapshot.Measures[0].Frame != snapshot {
				t.Fatal("measure lost from snapshot")
			}
			m.Start[0] = 99
			if snapshot.Measures[0].Start[0] != 0 {
				t.Fatal("measure snapshot aliases live coordinates")
			}
			f, err := w.Undo()
			if err != nil || len(f.Measures) != 0 {
				t.Fatalf("measure undo failed: %v", err)
			}
		})
	}
	h := geometryFixture(t)
	s := startCommand(t, h, "MEASURE")
	pickGeometry(t, h, s, 1)
	h.key(key.NameEscape)
	sessionDone(t, s)
	waitUI(t, h.w)
	if len(h.w.frame.Measures) != 0 || h.w.changed {
		t.Fatal("cancelled measure changed model")
	}
}
func TestElevenKijunPromptAndDuplicate(t *testing.T) {
	h := geometryFixture(t)
	w := h.w
	s := startCommand(t, h, "KIJUN")
	pickGeometry(t, h, s, 1)
	pickGeometry(t, h, s, 2)
	answerGeometryPrompt(t, h, s, "基準/A")
	sessionDone(t, s)
	if k := w.frame.Kijuns["基準/a"]; k == nil || !w.frame.Show.Kijun {
		t.Fatal("named kijun missing / path text was altered")
	}
	s = startCommand(t, h, "KIJUN")
	pickGeometry(t, h, s, 1)
	pickGeometry(t, h, s, 3)
	answerGeometryPrompt(t, h, s, "基準/A")
	answerGeometryPrompt(t, h, s, "B")
	sessionDone(t, s)
	if len(w.frame.Kijuns) != 2 {
		t.Fatal("duplicate name replaced original kijun")
	}
	f, err := w.Undo()
	if err != nil || len(f.Kijuns) != 1 {
		t.Fatalf("kijun undo failed: %v", err)
	}
}
func TestElevenAxisStrongWeakAndInvalid(t *testing.T) {
	for _, weak := range []bool{false, true} {
		t.Run(map[bool]string{false: "strong", true: "weak"}[weak], func(t *testing.T) {
			h := geometryFixture(t)
			w := h.w
			el := w.frame.Elems[1]
			w.SelectElem([]*st.Elem{el})
			s := startCommand(t, h, "AXISTOCANG")
			if weak {
				geometryKey(t, h, s, "W")
			}
			geometryKey(t, h, s, "D")
			answerGeometryPrompt(t, h, s, "0,0,1")
			sessionDone(t, s)
			direction := el.Strong
			if weak {
				direction = el.Weak
			}
			if math.Abs(st.Dot(direction, []float64{0, 0, 1}, 3)-1) > 1e-8 {
				t.Fatalf("axis direction=%v", direction)
			}
			if !w.changed {
				t.Fatal("axis edit not snapshotted")
			}
		})
	}
	h := geometryFixture(t)
	w := h.w
	el := w.frame.Elems[1]
	before := el.Cang
	w.SelectElem([]*st.Elem{el})
	s := startCommand(t, h, "AXISTOCANG")
	geometryKey(t, h, s, "X")
	sessionDone(t, s)
	if el.Cang != before || w.changed {
		t.Fatal("parallel axis edited element")
	}
}
func TestElevenRotateScaleAndLockedTargets(t *testing.T) {
	h := geometryFixture(t)
	w := h.w
	w.frame.Nodes[3].Lock = true
	w.SelectNode([]*st.Node{w.frame.Nodes[2], w.frame.Nodes[3]})
	s := startCommand(t, h, "ROTATE")
	pickGeometry(t, h, s, 1)
	geometryKey(t, h, s, "Z")
	answerGeometryPrompt(t, h, s, "90")
	sessionDone(t, s)
	assertCoordinate(t, w.frame.Nodes[2], []float64{0, 2, 0})
	assertCoordinate(t, w.frame.Nodes[3], []float64{0, 2, 0})
	f, err := w.Undo()
	if err != nil {
		t.Fatal(err)
	}
	assertCoordinate(t, f.Nodes[2], []float64{2, 0, 0})
	h = geometryFixture(t)
	w = h.w
	// Selected member and node share an endpoint: it must be transformed once.
	w.SelectElem([]*st.Elem{w.frame.Elems[1]})
	w.SelectNode([]*st.Node{w.frame.Nodes[2]})
	s = startCommand(t, h, "SCALE")
	geometryKey(t, h, s, "D")
	answerGeometryPrompt(t, h, s, "0,0,0")
	answerGeometryPrompt(t, h, s, "NaN")
	answerGeometryPrompt(t, h, s, "2")
	sessionDone(t, s)
	assertCoordinate(t, w.frame.Nodes[2], []float64{4, 0, 0})
	assertCoordinate(t, w.frame.Nodes[1], []float64{0, 0, 0})
}
func TestElevenRotateReferenceAndPromptCancellation(t *testing.T) {
	h := geometryFixture(t)
	w := h.w
	w.SelectNode([]*st.Node{w.frame.Nodes[5]})
	s := startCommand(t, h, "ROTATE")
	pickGeometry(t, h, s, 1)
	geometryKey(t, h, s, "Z")
	answerGeometryPrompt(t, h, s, "R")
	for _, i := range []int{1, 2, 3} {
		pickGeometry(t, h, s, i)
	}
	sessionDone(t, s)
	assertCoordinate(t, w.frame.Nodes[5], []float64{-1, 1, 1})
	h = geometryFixture(t)
	w = h.w
	w.SelectNode([]*st.Node{w.frame.Nodes[5]})
	s = startCommand(t, h, "SCALE")
	pickGeometry(t, h, s, 1)
	awaitGeometryPrompt(t, h, s)
	h.key(key.NameEscape)
	sessionDone(t, s)
	waitUI(t, w)
	h.frame()
	if w.ui.modal != nil || w.changed {
		t.Fatal("cancelled prompt retained dialog or edited model")
	}
	assertCoordinate(t, w.frame.Nodes[5], []float64{1, 1, 1})
	// Replacing a command while it waits for text must unblock its coroutine.
	w.SelectNode([]*st.Node{w.frame.Nodes[5]})
	s = startCommand(t, h, "SCALE")
	pickGeometry(t, h, s, 1)
	awaitGeometryPrompt(t, h, s)
	w.submitCommand("DISTS")
	sessionDone(t, s)
	next := awaitInteraction(t, h)
	h.frame()
	if w.ui.modal != nil {
		t.Fatal("stale prompt survived command replacement")
	}
	h.key(key.NameEscape)
	sessionDone(t, next)
	waitUI(t, w)
}

func TestElevenDegenerateGeometryIsAtomic(t *testing.T) {
	t.Run("collinear-plane", func(t *testing.T) {
		h := geometryFixture(t)
		w := h.w
		w.frame.Nodes[6].Coord = []float64{4, 0, 0}
		w.SelectNode([]*st.Node{w.frame.Nodes[5], w.frame.Nodes[7]})
		s := startCommand(t, h, "MOVETOPLANE")
		for _, n := range []int{1, 2, 6} {
			pickGeometry(t, h, s, n)
		}
		sessionDone(t, s)
		assertCoordinate(t, w.frame.Nodes[5], []float64{1, 1, 1})
		assertCoordinate(t, w.frame.Nodes[7], []float64{1, 1, 3})
		if w.changed {
			t.Fatal("degenerate plane modified model")
		}
	})
	t.Run("axis-multiple-targets", func(t *testing.T) {
		h := geometryFixture(t)
		w := h.w
		a, b := w.frame.Elems[1], w.frame.Elems[2]
		beforeA, beforeB := a.Cang, b.Cang
		w.SelectElem([]*st.Elem{a, b})
		s := startCommand(t, h, "AXISTOCANG")
		geometryKey(t, h, s, "D")
		answerGeometryPrompt(t, h, s, "-1,1,0")
		sessionDone(t, s)
		if a.Cang != beforeA || b.Cang != beforeB || w.changed {
			t.Fatal("invalid second member left a partial axis edit")
		}
	})
	t.Run("axis-zero-retry", func(t *testing.T) {
		h := geometryFixture(t)
		w := h.w
		w.SelectElem([]*st.Elem{w.frame.Elems[1]})
		s := startCommand(t, h, "AXISTOCANG")
		geometryKey(t, h, s, "D")
		answerGeometryPrompt(t, h, s, "0,0,0")
		geometryKey(t, h, s, "Z")
		sessionDone(t, s)
		if !w.changed {
			t.Fatal("zero-vector retry did not accept valid direction")
		}
	})
	t.Run("invalid-reference-angle", func(t *testing.T) {
		h := geometryFixture(t)
		w := h.w
		w.SelectNode([]*st.Node{w.frame.Nodes[5]})
		s := startCommand(t, h, "ROTATE")
		pickGeometry(t, h, s, 1)
		geometryKey(t, h, s, "Z")
		answerGeometryPrompt(t, h, s, "R")
		for _, n := range []int{1, 4, 7} {
			pickGeometry(t, h, s, n)
		}
		sessionDone(t, s)
		assertCoordinate(t, w.frame.Nodes[5], []float64{1, 1, 1})
		if w.changed {
			t.Fatal("invalid reference angle edited geometry")
		}
	})
}
