package st

import (
	"reflect"
	"strings"
	"testing"
)

func TestGUISnapshotRetainsInputsAndIndependentUndo(t *testing.T) {
	f := yieldEndInputFrame()
	f.Props[1].Name = "test"
	n := f.AddNode(0.123456789, 0, 0)
	n.ManualWeight = [3]float64{1, 2, 3}
	n.Load[2] = -3.1e-11
	a, b := f.AddNode(1, 0, 0), f.AddNode(0, 1, 0)
	arc := NewArc([]*Node{n, a, b})
	arc.Frame = f
	f.Arcs = append(f.Arcs, arc)
	sec := f.AddSect(201)
	sec.Allow = NewRCGirder(201)
	rc := sec.Allow.(*RCGirder)
	rc.CShape = CRect{Left: -15, Right: 15, Upper: 30, Lower: -30}
	rc.Concrete = FC24
	rc.Hoops.Ps = []float64{.001, .002}
	before := f.GUIRevision()
	clone := f.Snapshot()
	if len(clone.Arcs) != 1 || clone.Arcs[0].Enod[0] != clone.Nodes[n.Num] || reflect.TypeOf(clone.Sects[201].Allow) != reflect.TypeOf(sec.Allow) {
		t.Fatal("snapshot lost arc references or LST type")
	}
	clone.Sects[201].Allow.(*RCGirder).Hoops.Ps[0] = .1
	if rc.Hoops.Ps[0] != .001 {
		t.Fatal("undo snapshot shares hoop reinforcement")
	}
	clone.View.Focus[0] = 10
	clone.Show.Dfact = 10
	if f.View.Focus[0] == 10 || f.Show.Dfact == 10 {
		t.Fatal("snapshot shares display state")
	}
	n.Coord[0] += 1e-9
	if f.GUIRevision() == before {
		t.Fatal("revision missed a submillimetre coordinate change")
	}
	restored := NewFrame()
	if err := restored.ReadInpSnapshot(f.GUIInpBytes(), "model.inp"); err != nil {
		t.Fatal(err)
	}
	if restored.Nodes[n.Num].Coord[0] != n.Coord[0] || restored.Nodes[n.Num].Load[2] != n.Load[2] || restored.Nodes[n.Num].ManualWeight != n.ManualWeight {
		t.Fatal("live serialization rounded node inputs")
	}
}

func TestUndoBaselineDoesNotDuplicateExistingEntry(t *testing.T) {
	f := yieldEndInputFrame()
	n := f.AddNode(0, 0, 0)
	u := NewUndoStack(5)
	u.PushUndo(f)
	u.CaptureUndoBaseline(f)
	n.Coord[0] = 1
	u.PushUndo(f)
	restored, err := u.Undo()
	if err != nil || restored.Nodes[n.Num].Coord[0] != 0 {
		t.Fatal("undo did not restore baseline", err)
	}
	if _, err := u.Undo(); err == nil {
		t.Fatal("duplicate baseline consumed another undo step")
	}
}

func TestGUIInputOmitsDerivedAnalysisBraces(t *testing.T) {
	f := yieldEndInputFrame()
	f.Props[1].Name = "test"
	sec, err := f.ParseSect(strings.Fields(yieldEndTestSection), false)
	if err != nil {
		t.Fatal(err)
	}
	a, b := f.AddNode(0, 0, 0), f.AddNode(1, 0, 1)
	c := f.AddNode(1, 0, 0)
	parent := NewPlateElem([]*Node{a, b, c}, sec, WALL)
	f.AddElem(1, parent)
	before := f.GUIRevision()
	derived := f.AddSect(901)
	derived.Original, derived.Type = sec.Num, WBRACE
	derived.Figs = []*Fig{sec.Figs[0].Snapshot(f)}
	child := NewLineElem([]*Node{a, b}, derived, WBRACE)
	f.AddElem(2, child)
	parent.Adopt(child)
	if got := f.GUIRevision(); got != before {
		t.Fatal("derived braces changed the input revision")
	}
	reopened := NewFrame()
	if err := reopened.ReadInpSnapshot(f.GUIInpBytes(), "model.inp"); err != nil {
		t.Fatal(err)
	}
	if len(reopened.Elems) != 1 || reopened.Sects[901] != nil {
		t.Fatal("analysis brace exported as an additional input member")
	}
	// Empty section names must remain empty when serialized and parsed.
	empty := f.AddSect(902)
	if _, err := reopened.ParseSect(strings.Fields(empty.InpString()), false); err != nil {
		t.Fatal(err)
	}
	if reopened.Sects[902].Name != "" {
		t.Fatal("empty section name consumed the following keyword")
	}
}
