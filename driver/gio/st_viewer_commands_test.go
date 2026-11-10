package stgio

import (
	"image"
	"strings"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/pointer"
	"gioui.org/unit"
	st "github.com/yofu/st/stlib"
)

func setupViewerEditor(t *testing.T) (*uiHarness, *ViewerState) {
	t.Helper()
	state := new(ViewerState)
	w := NewModelViewer(nil, func(s ViewerState) { *state = s })
	t.Cleanup(w.ui.close)
	t.Cleanup(w.stopInteractiveCommand)
	h := &uiHarness{w: w, size: image.Pt(1200, 900), metric: unit.Metric{PxPerDp: 2, PxPerSp: 2}, now: time.Unix(1, 0)}
	h.frame()
	ch := make(chan error, 1)
	if !w.OpenViewerModel("editing.inp", []byte(viewerINP), func(err error) { ch <- err }) {
		t.Fatal("open rejected")
	}
	waitUI(t, w)
	h.frame()
	if err := <-ch; err != nil {
		t.Fatal(err)
	}
	return h, state
}
func viewerEditorTap(h *uiHarness, p f32.Point) {
	h.router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: 1, Position: p})
	h.frame()
	h.router.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Touch, PointerID: 1, Position: p})
	h.frame()
}
func viewerEditorCommand(t *testing.T, h *uiHarness, name string) {
	t.Helper()
	if !h.w.SubmitViewerCommand(name) {
		t.Fatalf("command not accepted: %s", name)
	}
	waitUI(t, h.w)
	h.frame()
}
func exportViewerForTest(t *testing.T, h *uiHarness, wantName string) *st.Frame {
	t.Helper()
	w := h.w
	type export struct {
		name  string
		bytes []byte
		err   error
	}
	result := make(chan export, 1)
	if !w.ExportViewerModel(func(name string, b []byte, err error) { result <- export{name, b, err} }) {
		t.Fatal("export rejected")
	}
	waitUI(t, w)
	h.frame()
	saved := <-result
	if saved.err != nil || (wantName != "" && saved.name != wantName) {
		t.Fatalf("export: %v, %q", saved.err, saved.name)
	}
	f, err := parseViewerModel(saved.name, saved.bytes)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestViewerEditAndExport(t *testing.T) {
	h, state := setupViewerEditor(t)
	w := h.w
	viewerEditorCommand(t, h, ":addnode 0 0 0")
	n := w.SelectedNodes()[0].Num
	viewerEditorCommand(t, h, "ADDLINEELEM")
	cmd := awaitInteraction(t, h)
	viewerEditorTap(h, pointOf(w.frame.Nodes[101]))
	flushInteraction(t, h, cmd)
	viewerEditorTap(h, pointOf(w.frame.Nodes[n]))
	flushInteraction(t, h, cmd)
	sessionDone(t, cmd)
	h.frame()
	if len(w.frame.Elems) != 2 {
		t.Fatal("two taps did not add a member")
	}
	w.ViewerAction("node")
	waitUI(t, w)
	h.frame()
	viewerEditorTap(h, pointOf(w.frame.Nodes[n]))
	viewerEditorCommand(t, h, ":conf 111000")
	if !w.frame.Nodes[n].Conf[0] {
		t.Fatal("support edit did not apply")
	}
	viewerEditorCommand(t, h, "UNDO")
	if w.frame.Nodes[n].Conf[0] {
		t.Fatal("undo did not restore supports")
	}
	viewerEditorCommand(t, h, "REDO")
	if !w.frame.Nodes[n].Conf[0] {
		t.Fatal("redo did not restore edit")
	}
	w.ViewerAction("member")
	waitUI(t, w)
	h.frame()
	p := pointOf(w.frame.Nodes[102]).Add(pointOf(w.frame.Nodes[n])).Mul(.5)
	viewerEditorTap(h, p)
	viewerEditorCommand(t, h, ":bond pin")
	if w.frame.Elems[1001].Bonds[4] != st.Pin {
		t.Fatal("picked member did not receive end conditions")
	}
	f := exportViewerForTest(t, h, "editing-edited.inp")
	if len(f.Nodes) != 3 || len(f.Elems) != 2 || !f.Nodes[n].Conf[0] || f.Elems[1001].Bonds[4] == nil {
		t.Fatal("saved INP lost the edits")
	}
	if f.Wind.Velocity != 42 || f.Elems[1001].Prestress != 12.5 || len(f.NodeSet["Ends"]) != 2 || len(f.ElemSet["Beam"]) != 1 {
		t.Fatal("undo and save did not preserve model data or sets")
	}
	if !strings.Contains(state.History, "> ADDLINEELEM") || !strings.Contains(state.History, "EXPORT: editing-edited.inp") {
		t.Fatalf("no edit history: %s", state.History)
	}
	w.ViewerAction("clear")
	waitUI(t, w)
	h.frame()
	viewerEditorCommand(t, h, "ERASE")
	cmd = awaitInteraction(t, h)
	viewerEditorTap(h, p)
	flushInteraction(t, h, cmd)
	if !w.ViewerAction("finish") {
		t.Fatal("finish action rejected")
	}
	sessionDone(t, cmd)
	h.frame()
	f = exportViewerForTest(t, h, "editing-edited.inp")
	if f.Elems[1001] != nil || f.Nodes[102] != nil || len(f.NodeSet["Ends"]) != 1 || len(f.ElemSet["Beam"]) != 0 {
		t.Fatal("confirm did not delete the picked member and clean up its sets")
	}

}

func TestViewerEditPromptAndCancel(t *testing.T) {
	h, state := setupViewerEditor(t)
	w := h.w
	w.ViewerAction("node")
	waitUI(t, w)
	h.frame()
	viewerEditorTap(h, pointOf(w.frame.Nodes[102]))
	viewerEditorCommand(t, h, "SCALE")
	s := awaitInteraction(t, h)
	viewerEditorTap(h, pointOf(w.frame.Nodes[101]))
	flushInteraction(t, h, s)
	end := time.Now().Add(3 * time.Second)
	for state.PromptID == 0 && time.Now().Before(end) {
		h.frame()
		time.Sleep(time.Millisecond)
	}
	if state.PromptID == 0 || !strings.Contains(state.PromptTitle, "倍率") {
		t.Fatalf("no SCALE prompt: %+v", state)
	}
	errch := make(chan error, 1)
	w.ExportViewerModel(func(_ string, _ []byte, err error) { errch <- err })
	waitUI(t, w)
	if <-errch == nil {
		t.Fatal("export proceeded with a command still active")
	}
	if !w.ReplyViewerPrompt(state.PromptID, "1.5", true) {
		t.Fatal("prompt reply rejected")
	}
	h.frame()
	sessionDone(t, s)
	h.frame()
	if x := w.frame.Nodes[102].Coord[0]; x != 4 {
		t.Fatalf("scaled X=%g, want 4", x)
	}
	viewerEditorCommand(t, h, "ADDLINEELEM")
	s = awaitInteraction(t, h)
	viewerEditorTap(h, pointOf(w.frame.Nodes[101]))
	flushInteraction(t, h, s)
	w.ViewerAction("cancel")
	waitUI(t, w)
	sessionDone(t, s)
	h.frame()
	if len(w.frame.Elems) != 1 || w.Executing() || w.NodeSelected() {
		t.Fatal("cancel did not discard pending line")
	}
	viewerEditorCommand(t, h, ":unknownweb")
	if !strings.Contains(state.History, "Web版で使えないコマンド") {
		t.Fatal("desktop file command was not rejected")
	}
}

func TestViewerExMode(t *testing.T) {
	h, state := setupViewerEditor(t)
	w := h.w
	viewerEditorCommand(t, h, ":.")
	if !strings.Contains(state.History, "繰り返すexコマンドがありません") {
		t.Fatal("empty repeat did not report an error")
	}
	viewerEditorCommand(t, h, ":no 9999,101 | :move 1 0 0")
	if x := w.frame.Nodes[101].Coord[0]; x != -1 || !state.Modified || state.SelectedNodes != 1 {
		t.Fatalf("abbreviated selection and move: X=%g, %+v", x, state)
	}
	viewerEditorCommand(t, h, ":.")
	if x := w.frame.Nodes[101].Coord[0]; x != 0 {
		t.Fatalf("repeated move: X=%g", x)
	}
	viewerEditorCommand(t, h, "UNDO")
	if x := w.frame.Nodes[101].Coord[0]; x != -1 {
		t.Fatalf("undo of repeated move: X=%g", x)
	}
	viewerEditorCommand(t, h, ":elem 9999,1001 | :bo pin | :count")
	if w.frame.Elems[1001].Bonds[4] != st.Pin || !strings.Contains(state.History, "NODES: 0, ELEMS: 1") {
		t.Fatal("element selection, bond change or count failed")
	}
	viewerEditorCommand(t, h, ":node 101 | :conf 1_t000")
	if c := w.frame.Nodes[101].Conf; !c[0] || c[1] || !c[2] {
		t.Fatalf("unexpected support flags: %v", c)
	}
	viewerEditorCommand(t, h, ":node 101 | :move 3 0 0 | :unknownweb")
	if x := w.frame.Nodes[101].Coord[0]; x != -1 || !strings.Contains(state.History, "Web版で使えないコマンド") {
		t.Fatal("an unsupported stage should reject the entire pipeline before editing")
	}
	viewerEditorCommand(t, h, ":conf")
	viewerEditorCommand(t, h, ":move NaN 0 0")
	viewerEditorCommand(t, h, ":node 101 | :add elem -sect=101 -etype=girder")
	if len(w.frame.Elems) != 1 || w.frame.Nodes[101].Coord[0] != -1 {
		t.Fatal("invalid arguments changed the model")
	}
	viewerEditorCommand(t, h, ":node 101,102 | :add elem -sect=101 -etype=brace")
	if len(w.frame.Elems) != 2 {
		t.Fatal("pipeline did not add a brace from the two selected nodes")
	}

	type saved struct {
		name string
		data []byte
		err  error
	}
	ch := make(chan saved, 1)
	export := func(name string) saved {
		t.Helper()
		if !w.ExportViewerModelAs(name, func(name string, data []byte, err error) { ch <- saved{name, data, err} }) {
			t.Fatal("export request rejected")
		}
		waitUI(t, w)
		h.frame()
		return <-ch
	}
	if result := export("../out.inp"); result.err == nil {
		t.Fatal("a path was accepted as a browser filename")
	}
	result := export("編集 済み.inp")
	if result.err != nil || result.name != "編集 済み.inp" {
		t.Fatalf("named export: %q, %v", result.name, result.err)
	}
	f, err := parseViewerModel(result.name, result.data)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Elems) != 2 || f.Elems[1001].Bonds[4] == nil || f.Nodes[101].Coord[0] != -1 || !f.Nodes[101].Conf[2] {
		t.Fatal("exported INP lost ex-mode edits")
	}
}

func TestViewerFig2(t *testing.T) {
	h, state := setupViewerEditor(t)
	w := h.w
	viewerEditorCommand(t, h, "'.")
	if !strings.Contains(state.History, "no previous Fig2 command") {
		t.Fatal("repeat without a previous Fig2 command was not handled")
	}
	viewerEditorCommand(t, h, "'nodecode")
	if w.frame.Show.NodeCaption&st.NC_NUM == 0 {
		t.Fatal("Fig2 did not enable node labels")
	}
	viewerEditorCommand(t, h, "'!nodecode")
	if w.frame.Show.NodeCaption&st.NC_NUM != 0 {
		t.Fatal("Fig2 negation did not hide node labels")
	}
	viewerEditorCommand(t, h, "'ang 15 35")
	if w.frame.View.Angle[0] != 15 || w.frame.View.Angle[1] != 35 {
		t.Fatal("abbreviated Fig2 command did not update the view")
	}
	viewerEditorCommand(t, h, "'title 試験モデル")
	viewerEditorCommand(t, h, "'position title 20 100")
	if !strings.Contains(state.Annotations, "試験モデル") {
		t.Fatalf("missing Japanese annotation: %q", state.Annotations)
	}
	viewerEditorCommand(t, h, "'!title")
	if strings.Contains(state.Annotations, "試験モデル") {
		t.Fatal("hidden title remained in the overlay")
	}
	viewerEditorCommand(t, h, "'gfact")
	if !strings.Contains(state.History, "GFACT") {
		t.Fatal("missing Fig2 argument did not produce a diagnostic")
	}
}
