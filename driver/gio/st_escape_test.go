package stgio

import (
	"testing"

	"gioui.org/io/key"
	"gioui.org/io/pointer"
	st "github.com/yofu/st/stlib"
)

func TestEscapeDeselectsNodesAndMembers(t *testing.T) {
	h := portFixture(t)
	w := h.w
	w.SelectNode([]*st.Node{w.frame.Nodes[1]})
	w.SelectElem([]*st.Elem{w.frame.Elems[1]})
	w.Redraw()
	h.frame()
	rev := w.ui.canvasRevision
	changed := w.changed
	h.key(key.NameEscape)
	waitUI(t, w)
	h.frame()
	if w.NodeSelected() || w.ElemSelected() {
		t.Fatal("Esc retained node or member selection")
	}
	if w.ui.canvasRevision <= rev {
		t.Fatal("deselection did not redraw highlights")
	}
	if w.changed != changed || len(w.frame.Elems) != 2 || len(w.frame.Nodes) != 4 {
		t.Fatal("deselection modified model")
	}
}
func TestEscapeDeselectsAfterInteractiveCancellation(t *testing.T) {
	h := portFixture(t)
	w := h.w
	w.SelectElem([]*st.Elem{w.frame.Elems[1]})
	s := startCommand(t, h, "DISTS")
	clickInteractive(t, h, s, pointOf(w.frame.Nodes[1]), pointer.ButtonPrimary)
	if !w.NodeSelected() {
		t.Fatal("command did not select first reference")
	}
	h.key(key.NameEscape)
	sessionDone(t, s)
	waitUI(t, w)
	h.frame()
	if w.Executing() || w.NodeSelected() || w.ElemSelected() {
		t.Fatal("Esc did not terminate and clear selection")
	}
	if len(w.ui.commandTail) != 0 || w.ui.selectionOverlay {
		t.Fatal("Esc retained command feedback")
	}
	if len(w.frame.Elems) != 2 || len(w.frame.Nodes) != 4 || w.changed {
		t.Fatal("cancel and deselect modified model")
	}
}
