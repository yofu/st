package stgio

import (
	"image"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/pointer"
	"gioui.org/unit"
)

const viewerINP = `ANGLE 20 35
VELOCITY 42
NODE 101 CORD -2 0 0
NODE 102 CORD 2 0 0
ELEM 1001 ESECT 101 ENODS 2 ENOD 101 102 TYPE GIRDER PREST 12.5
NODESET Ends 101 102
ELEMSET Beam 1001
`

func TestViewerLoadAndPreserveOnError(t *testing.T) {
	h := newUIHarness(t)
	h.w.viewer = &modelViewer{scale: 1, zoom: 1}
	h.frame()
	if !h.w.OpenViewerModel("構造.INP", []byte("\ufeff"+viewerINP), nil) {
		t.Fatal("queue failed")
	}
	waitUI(t, h.w)
	h.frame()
	good := h.w.frame
	if len(good.Nodes) != 2 || len(good.Elems) != 1 || good.Name != "構造.INP" {
		t.Fatalf("wrong frame: %q, %d nodes, %d elements", good.Name, len(good.Nodes), len(good.Elems))
	}
	answer := make(chan error, 1)
	if !h.w.OpenViewerModel("bad.inp", []byte("NODE\n"), func(err error) { answer <- err }) {
		t.Fatal("queue failed")
	}
	waitUI(t, h.w)
	h.frame()
	if err := <-answer; err == nil || h.w.frame != good {
		t.Fatalf("bad file replaced good frame; error=%v", err)
	}
}

func TestViewerTouchRouter(t *testing.T) {
	h := newUIHarness(t)
	h.metric = unit.Metric{PxPerDp: 2, PxPerSp: 2}
	w := h.w
	w.viewer = &modelViewer{scale: 2, zoom: 1}
	h.frame()
	if !w.OpenViewerModel("touch.inp", []byte(viewerINP), nil) {
		t.Fatal("queue failed")
	}
	waitUI(t, w)
	h.frame()
	event := func(kind pointer.Kind, id pointer.ID, pos f32.Point) {
		h.router.Queue(pointer.Event{Kind: kind, Source: pointer.Touch, PointerID: id, Position: pos})
		h.frame()
	}
	point := func(a []float64) f32.Point { return f32.Pt(float32(a[0]), float32(a[1])) }
	a, b := point(w.frame.Nodes[101].Pcoord), point(w.frame.Nodes[102].Pcoord)
	center := a.Add(b).Mul(.5)
	event(pointer.Press, 1, center)
	event(pointer.Release, 1, center)
	if len(w.SelectedElems()) != 1 || w.SelectedElems()[0].Num != 1001 {
		t.Fatal("tap did not pick member")
	}
	w.ViewerAction("node")
	waitUI(t, w)
	h.frame()
	event(pointer.Press, 1, a)
	event(pointer.Release, 1, a)
	if len(w.SelectedNodes()) != 1 || w.SelectedNodes()[0].Num != 101 {
		t.Fatal("tap did not pick node")
	}
	w.ViewerAction("clear")
	waitUI(t, w)
	h.frame()
	before := w.frame.View.Angle[1]
	event(pointer.Press, 1, center)
	event(pointer.Move, 1, center.Add(f32.Pt(80, 20)))
	event(pointer.Release, 1, center.Add(f32.Pt(80, 20)))
	if math.Abs(w.frame.View.Angle[1]-before) < 5 || len(w.SelectedNodes()) != 0 {
		t.Fatal("drag did not rotate independently of picking")
	}
	angle := w.frame.View.Angle[1]
	scale := w.frame.View.Gfact
	event(pointer.Press, 1, center.Sub(f32.Pt(60, 0)))
	event(pointer.Press, 2, center.Add(f32.Pt(60, 0)))
	event(pointer.Move, 1, center.Sub(f32.Pt(120, 0)))
	event(pointer.Move, 2, center.Add(f32.Pt(120, 0)))
	event(pointer.Release, 2, center.Add(f32.Pt(120, 0)))
	event(pointer.Release, 1, center.Sub(f32.Pt(120, 0)))
	if math.Abs(w.frame.View.Gfact/scale-2) > 1e-6 || w.frame.View.Angle[1] != angle || len(w.SelectedNodes()) != 0 {
		t.Fatalf("pinch changed wrong state: scale ratio %f", w.frame.View.Gfact/scale)
	}
	h.size = image.Pt(700, 1000)
	h.frame()
	if w.canvasSize != h.size || math.Abs(w.viewer.zoom-2) > 1e-6 {
		t.Fatal("resize did not preserve zoom")
	}
}

// Optional integration fixture: keep user models outside the repository.
func TestViewerRealINP(t *testing.T) {
	input := os.Getenv("ST_VIEWER_TEST_INP")
	if input == "" {
		t.Skip("set ST_VIEWER_TEST_INP to a local model")
	}
	data, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	var state ViewerState
	w := NewModelViewer(nil, func(s ViewerState) { state = s })
	t.Cleanup(w.ui.close)
	h := &uiHarness{w: w, size: image.Pt(1488, 1800), metric: unit.Metric{PxPerDp: 2, PxPerSp: 2}, now: time.Unix(1, 0)}
	h.frame()
	result := make(chan error, 1)
	if !w.OpenViewerModel(filepath.Base(input), data, func(err error) { result <- err }) {
		t.Fatal("file was not queued")
	}
	waitUI(t, w)
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	h.frame()
	if !w.ui.canvasReady || state.Nodes <= 0 || state.Members <= 0 {
		t.Fatalf("model did not produce drawing operations: %+v", state)
	}
	nodes, members := state.Nodes, state.Members
	viewerEditorCommand(t, h, ":addnode 123 456 789")
	added := w.SelectedNodes()[0].Num
	f := exportViewerForTest(t, h, "")
	if len(f.Nodes) != nodes+1 || len(f.Elems) != members || f.Nodes[added].Coord[2] != 789 {
		t.Fatalf("real-model round trip: %d nodes, %d members", len(f.Nodes), len(f.Elems))
	}
	t.Logf("%s: edited INP reopened with %d nodes, %d members", f.Name, len(f.Nodes), len(f.Elems))
}
