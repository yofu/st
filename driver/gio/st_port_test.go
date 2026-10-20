package stgio

import (
	"math"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/unit"
	st "github.com/yofu/st/stlib"
)

func portFixture(t *testing.T) *uiHarness {
	h := newUIHarness(t)
	h.metric = unit.Metric{PxPerDp: 1.5, PxPerSp: 1.5}
	h.frame()
	w := h.w
	f := w.frame
	sect := st.NewSect()
	sect.Num = 101
	sect.Frame = f
	f.Sects[101] = sect
	f.Show.Sect[101] = true
	coords := [][]float64{{-2, 0, 0}, {2, 0, 0}, {0, -2, 0}, {0, 2, 0}}
	for i, c := range coords {
		n := st.NewNode()
		n.Num = i + 1
		n.Frame = f
		n.Coord = c
		f.Nodes[n.Num] = n
	}
	f.AddLineElem(1, []*st.Node{f.Nodes[1], f.Nodes[2]}, sect, st.GIRDER)
	f.AddLineElem(2, []*st.Node{f.Nodes[3], f.Nodes[4]}, sect, st.GIRDER)
	f.View.Perspective = false
	f.View.Angle = []float64{90, 0}
	w.SetCanvasAnimateSpeed(0)
	w.showCanvasCenter()
	w.Redraw()
	h.frame()
	st.Snapshot(w)
	w.changed = false
	// Exercise the actual PGP names used in the user's configuration.
	path := writePgp(t, t.TempDir(), "F FENCE\nI INTERSECT\n")
	if err := w.ReadPgp(path); err != nil {
		t.Fatal(err)
	}
	return h
}
func memberQuarter(el *st.Elem) f32.Point {
	return pointOf(el.Enod[0]).Mul(0.75).Add(pointOf(el.Enod[1]).Mul(0.25))
}
func assertIntersection(t *testing.T, w *Window) {
	t.Helper()
	f := w.frame
	if len(f.Elems) != 4 || len(f.Nodes) != 5 {
		t.Fatalf("intersection size elems=%d nodes=%d", len(f.Elems), len(f.Nodes))
	}
	var center *st.Node
	for _, n := range f.Nodes {
		if math.Abs(n.Coord[0])+math.Abs(n.Coord[1])+math.Abs(n.Coord[2]) < 1e-9 {
			center = n
		}
	}
	if center == nil {
		t.Fatal("intersection node missing")
	}
	count := 0
	for _, el := range f.Elems {
		if math.Abs(el.Length()-2) > 1e-9 {
			t.Fatalf("split length=%v", el.Length())
		}
		for _, n := range el.Enod {
			if n == center {
				count++
			}
		}
	}
	if count != 4 {
		t.Fatalf("intersection is not connected: %d", count)
	}
	if len(w.SelectedElems()) != 2 {
		t.Fatal("IUP result selection not retained")
	}
	for _, el := range w.SelectedElems() {
		if f.Elems[el.Num] != el {
			t.Fatal("selected obsolete element")
		}
	}
}
func TestPortIntersectClickCommitUndo(t *testing.T) {
	h := portFixture(t)
	w := h.w
	s := startCommand(t, h, "I")
	p1, p2 := memberQuarter(w.frame.Elems[1]), memberQuarter(w.frame.Elems[2])
	clickInteractive(t, h, s, p1, pointer.ButtonPrimary)
	clickInteractive(t, h, s, p1, pointer.ButtonPrimary)
	clickInteractive(t, h, s, p1, pointer.ButtonSecondary)
	if !w.Executing() || len(w.frame.Elems) != 2 {
		t.Fatal("duplicate picks committed intersection")
	}
	clickInteractive(t, h, s, p2, pointer.ButtonPrimary)
	if len(w.frame.Elems) != 2 {
		t.Fatal("intersection committed before right click")
	}
	clickInteractive(t, h, s, p2, pointer.ButtonSecondary)
	sessionDone(t, s)
	assertIntersection(t, w)
	f, err := w.Undo()
	if err != nil || len(f.Nodes) != 4 || len(f.Elems) != 2 {
		t.Fatalf("intersection undo: %v", err)
	}
}
func TestPortIntersectPreselectionAndEscape(t *testing.T) {
	h := portFixture(t)
	w := h.w
	w.SelectElem([]*st.Elem{w.frame.Elems[1], w.frame.Elems[2]})
	runPgpAlias(t, h, "I")
	assertIntersection(t, w)
	h = portFixture(t)
	w = h.w
	s := startCommand(t, h, "I")
	clickInteractive(t, h, s, memberQuarter(w.frame.Elems[1]), pointer.ButtonPrimary)
	h.key(key.NameEscape)
	sessionDone(t, s)
	if len(w.frame.Elems) != 2 || len(w.frame.Nodes) != 4 || w.changed {
		t.Fatal("cancelled intersection changed model")
	}
}
func TestPortIntersectInvalidGeometryAndLock(t *testing.T) {
	for _, kind := range []string{"parallel", "outside", "skew", "locked", "duplicate"} {
		t.Run(kind, func(t *testing.T) {
			h := portFixture(t)
			w := h.w
			f := w.frame
			switch kind {
			case "parallel":
				f.Nodes[3].Coord = []float64{-2, 1, 0}
				f.Nodes[4].Coord = []float64{2, 1, 0}
			case "outside":
				f.Nodes[3].Coord = []float64{3, -2, 0}
				f.Nodes[4].Coord = []float64{3, 2, 0}
			case "skew":
				f.Nodes[3].Coord[2] = 1
				f.Nodes[4].Coord[2] = 1
			case "locked":
				f.Elems[1].Lock = true
			}
			pair := []*st.Elem{f.Elems[1], f.Elems[2]}
			if kind == "duplicate" {
				pair[1] = pair[0]
			}
			if err := st.IntersectElems(w, pair); err == nil {
				t.Fatal("invalid intersection accepted")
			}
			if len(f.Nodes) != 4 || len(f.Elems) != 2 || w.changed {
				t.Fatal("invalid intersection changed model")
			}
		})
	}
}
func fenceAcross(el *st.Elem) (f32.Point, f32.Point) {
	mid := memberQuarter(el)
	d := pointOf(el.Enod[1]).Sub(pointOf(el.Enod[0]))
	p := f32.Pt(-d.Y, d.X).Mul(25 / float32(math.Hypot(float64(d.X), float64(d.Y))))
	return mid.Sub(p), mid.Add(p)
}
func dragFence(t *testing.T, h *uiHarness, s *interactiveCommand, a, b f32.Point, mods key.Modifiers) {
	t.Helper()
	routedMouse(h, pointer.Press, pointer.ButtonPrimary, a, mods, 0)
	routedMouse(h, pointer.Move, pointer.ButtonPrimary, b, mods, 0)
	if !h.w.ui.selectionOverlay || !h.w.ui.fenceOverlay {
		t.Fatal("fence line not visible during drag")
	}
	end := b
	if mods&key.ModCtrl != 0 {
		end.Y = a.Y
	}
	if h.w.ui.selectionStart != a || h.w.ui.selectionEnd != end {
		t.Fatalf("fence cursor mismatch start=%v end=%v", h.w.ui.selectionStart, h.w.ui.selectionEnd)
	}
	routedMouse(h, pointer.Release, 0, b, mods, 0)
	flushInteraction(t, h, s)
	sessionDone(t, s)
	h.frame()
	if h.w.ui.selectionOverlay {
		t.Fatal("fence overlay remained after completion")
	}
}
func TestPortFenceDragAddSubtract(t *testing.T) {
	h := portFixture(t)
	w := h.w
	a, b := fenceAcross(w.frame.Elems[1])
	s := startCommand(t, h, "F")
	dragFence(t, h, s, a, b, 0)
	if els := w.SelectedElems(); len(els) != 1 || els[0].Num != 1 {
		t.Fatalf("fence selected %v", els)
	}
	a, b = fenceAcross(w.frame.Elems[2])
	s = startCommand(t, h, "F")
	dragFence(t, h, s, a, b, 0)
	if len(w.SelectedElems()) != 2 {
		t.Fatal("fence did not add selection")
	}
	s = startCommand(t, h, "F")
	dragFence(t, h, s, a, b, key.ModShift)
	if els := w.SelectedElems(); len(els) != 1 || els[0].Num != 1 {
		t.Fatal("Shift fence did not remove selection")
	}
	if w.changed || len(w.frame.Elems) != 2 {
		t.Fatal("fence modified geometry")
	}
}
func TestPortFenceCtrlEscapeAndReleasePosition(t *testing.T) {
	h := portFixture(t)
	w := h.w
	a, b := fenceAcross(w.frame.Elems[1])
	// Member 1 projects vertically; a horizontal fence intersects it.
	if math.Abs(float64(a.Y-b.Y)) > 1 {
		t.Fatalf("fixture orientation unexpected: %v %v", a, b)
	}
	s := startCommand(t, h, "F")
	dragFence(t, h, s, a, b.Add(f32.Pt(0, 50)), key.ModCtrl)
	if els := w.SelectedElems(); len(els) != 1 || els[0].Num != 1 {
		t.Fatal("Ctrl horizontal fence failed")
	}
	w.Deselect()
	s = startCommand(t, h, "F")
	routedMouse(h, pointer.Press, pointer.ButtonPrimary, a, 0, 0)
	routedMouse(h, pointer.Move, pointer.ButtonPrimary, b, 0, 0)
	h.key(key.NameEscape)
	sessionDone(t, s)
	h.frame()
	if len(w.SelectedElems()) != 0 || w.ui.selectionOverlay {
		t.Fatal("Esc committed fence or retained overlay")
	}
	// No motion event: release alone must supply the actual endpoint.
	s = startCommand(t, h, "F")
	routedMouse(h, pointer.Press, pointer.ButtonPrimary, a, 0, 0)
	routedMouse(h, pointer.Release, 0, b, 0, 0)
	flushInteraction(t, h, s)
	sessionDone(t, s)
	if els := w.SelectedElems(); len(els) != 1 || els[0].Num != 1 {
		t.Fatal("release endpoint was lost")
	}
}
func TestPortFenceHiddenAndZeroLength(t *testing.T) {
	h := portFixture(t)
	w := h.w
	a, b := fenceAcross(w.frame.Elems[1])
	w.frame.Elems[1].Hide()
	st.SelectElemFence(w, st.FenceGesture{Start: st.Position{X: int(a.X), Y: int(a.Y)}, End: st.Position{X: int(b.X), Y: int(b.Y)}})
	if w.ElemSelected() {
		t.Fatal("hidden member selected by fence")
	}
	w.frame.Elems[1].Show()
	st.SelectElemFence(w, st.FenceGesture{Start: st.Position{X: 10, Y: 10}, End: st.Position{X: 10, Y: 10}})
	if w.ElemSelected() {
		t.Fatal("zero-length fence changed selection")
	}
}

func TestPortFenceReplacementClearsOverlay(t *testing.T) {
	h := portFixture(t)
	w := h.w
	a, b := fenceAcross(w.frame.Elems[1])
	s := startCommand(t, h, "F")
	routedMouse(h, pointer.Press, pointer.ButtonPrimary, a, 0, 0)
	routedMouse(h, pointer.Move, pointer.ButtonPrimary, b, 0, 0)
	w.submitCommand("DISTS")
	sessionDone(t, s)
	next := awaitInteraction(t, h)
	h.frame()
	if w.ui.selectionOverlay || w.ui.fenceOverlay || w.ElemSelected() {
		t.Fatal("replaced fence retained overlay or committed selection")
	}
	h.key(key.NameEscape)
	sessionDone(t, next)
}
