package stgio

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gioui.org/io/key"
	st "github.com/yofu/st/stlib"
)

func (h *uiHarness) typeText(text string) {
	for _, r := range text {
		start, end := h.w.ui.command.Selection()
		h.router.Queue(key.EditEvent{Range: key.Range{Start: start, End: end}, Text: string(r)}, key.SelectionEvent{Start: start + 1, End: start + 1})
		h.frame()
	}
}
func (h *uiHarness) modifiedKey(name key.Name, mod key.Modifiers) {
	h.router.Queue(key.Event{Name: name, State: key.Press, Modifiers: mod})
	h.frame()
}

func TestGioIUPInputCharacters(t *testing.T) {
	h := newUIHarness(t)
	w := h.w
	h.focus(&w.ui.command)
	h.typeText(";")
	if got := w.ui.command.Text(); got != ":" {
		t.Fatalf("leading semicolon: %q", got)
	}
	h.typeText("e ")
	if got := w.ui.command.Text(); got != ":edit " {
		t.Fatalf("Space abbreviation: %q", got)
	}
	h.typeText("C:\\model;01.inp")
	if got := w.ui.command.Text(); got != ":edit C:\\model;01.inp" {
		t.Fatalf("argument punctuation changed: %q", got)
	}
	h.modifiedKey("[", key.ModCtrl)
	if w.ui.command.Text() != "" {
		t.Fatal("Ctrl+[ did not clear input")
	}
	h.typeText(":")
	if got := w.ui.command.Text(); got != ";" {
		t.Fatalf("leading colon: %q", got)
	}
	h.modifiedKey("[", key.ModCtrl)
	h.typeText("'gf ")
	if got := w.ui.command.Text(); got != "'gfact " {
		t.Fatalf("figure abbreviation: %q", got)
	}
	h.modifiedKey("[", key.ModCtrl)
	h.typeText(";e!? ")
	if got := w.ui.command.Text(); got != ":edit!? " {
		t.Fatalf("bang/usage expansion: %q", got)
	}
}

func TestGioBatchedCommandInput(t *testing.T) {
	h := newUIHarness(t)
	w := h.w
	h.focus(&w.ui.command)
	text := ";'" // punctuation after the first character remains literal.
	for i, r := range text {
		h.router.Queue(key.Event{Name: key.Name(string(r)), State: key.Press}, key.EditEvent{Range: key.Range{Start: i, End: i}, Text: string(r)}, key.SelectionEvent{Start: i + 1, End: i + 1})
	}
	h.frame()
	if got := w.ui.command.Text(); got != ":'" {
		t.Fatalf("batched punctuation: %q", got)
	}
	h.modifiedKey("[", key.ModCtrl)
	text = ";view front"
	for i, r := range text {
		name := key.Name(string(r))
		if r == ' ' {
			name = key.NameSpace
		}
		h.router.Queue(key.Event{Name: name, State: key.Press}, key.EditEvent{Range: key.Range{Start: i, End: i}, Text: string(r)}, key.SelectionEvent{Start: i + 1, End: i + 1})
	}
	h.router.Queue(key.Event{Name: key.NameReturn, State: key.Press})
	h.frame()
	waitUI(t, w)
	finishViewAnimation(h)
	w.ui.modelMu.Lock()
	angle := append([]float64(nil), w.frame.View.Angle...)
	w.ui.modelMu.Unlock()
	if angle[0] != 0 || angle[1] != -90 {
		t.Fatalf("batched Enter did not execute normalized command: %v", angle)
	}
}

func TestGioIUPCompletionAndPrefixHistory(t *testing.T) {
	h := newUIHarness(t)
	w := h.w
	h.focus(&w.ui.command)
	w.ui.historyMu.Lock()
	for _, s := range []string{":edit A.inp", "'gfact 1", ":edit B.inp", "'gfact 2"} {
		w.addUICommandHistoryLocked(s)
	}
	w.ui.historyMu.Unlock()
	w.setCommandText(":edit ")
	h.modifiedKey("P", key.ModCtrl)
	if got := w.ui.command.Text(); got != ":edit B.inp" {
		t.Fatalf("prefix Ctrl+P: %q", got)
	}
	h.key(key.NameUpArrow)
	if got := w.ui.command.Text(); got != ":edit A.inp" {
		t.Fatalf("prefix Up: %q", got)
	}
	h.modifiedKey("N", key.ModCtrl)
	if got := w.ui.command.Text(); got != ":edit B.inp" {
		t.Fatalf("prefix Ctrl+N: %q", got)
	}
	h.key(key.NameDownArrow)
	if got := w.ui.command.Text(); got != ":edit " {
		t.Fatalf("prefix draft: %q", got)
	}
	w.setCommandText(":r")
	w.ui.lastCompletion = false
	h.key(key.NameTab)
	first := w.ui.command.Text()
	h.key(key.NameTab)
	second := w.ui.command.Text()
	if first == second {
		t.Fatal("Tab did not cycle")
	}
	h.modifiedKey(key.NameTab, key.ModShift)
	if got := w.ui.command.Text(); got != first {
		t.Fatalf("Shift+Tab: %q, want %q", got, first)
	}
	w.setCommandText(":view f")
	w.ui.lastCompletion = false
	h.key(key.NameTab)
	if got := w.ui.command.Text(); got != ":view front" {
		t.Fatalf("argument completion: %q", got)
	}
}

func TestGioHistoryRestartAndRecentModelCommands(t *testing.T) {
	directory := t.TempDir()
	store := filepath.Join(directory, ".st")
	if err := os.MkdirAll(store, 0755); err != nil {
		t.Fatal(err)
	}
	paths := []string{filepath.Join(directory, "直近 model A.inp"), filepath.Join(directory, "previous model B.inp")}
	for i, path := range paths {
		f := st.NewFrame()
		n := st.NewNode()
		n.Num = 101
		n.Frame = f
		n.Coord = []float64{float64(i + 1), 0, 0}
		f.Nodes[n.Num] = n
		if err := f.WriteInp(path); err != nil {
			t.Fatal(err)
		}
	}
	if err := writeHistoryLines(filepath.Join(store, "recent.dat"), paths); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store, "history.dat"), []byte("\xef\xbb\xbf'gfact 3\r\n:edit #0\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	h := newUIHarness(t)
	w := h.w
	if err := w.LoadHistory(store); err != nil {
		t.Fatal(err)
	}
	h.frame()
	h.focus(&w.ui.command)
	h.key(key.NameUpArrow)
	if got := w.ui.command.Text(); got != "'gfact 3" {
		t.Fatalf("startup command history: %q", got)
	}
	h.modifiedKey("[", key.ModCtrl)
	for i := 0; i < 2; i++ {
		h.typeText(fmt.Sprintf(";e #%d", i))
		h.key(key.NameReturn)
		waitUI(t, w)
		h.frame()
		w.ui.modelMu.Lock()
		actual := w.frame.Path
		node := w.frame.Nodes[101]
		if node == nil {
			w.ui.modelMu.Unlock()
			t.Fatalf("#%d did not load node 101; path=%q", i, actual)
		}
		coord := node.Coord[0]
		w.ui.modelMu.Unlock()
		if filepath.Clean(actual) != filepath.Clean(paths[i]) || coord != float64(i+1) {
			t.Fatalf("#%d opened %q (coord=%g), want %q", i, actual, coord, paths[i])
		}
	}
	second := NewWindow(nil)
	defer second.ui.close()
	if err := second.LoadHistory(store); err != nil {
		t.Fatal(err)
	}
	if got := filepath.FromSlash(second.Recent()[0]); got != paths[1] {
		t.Fatalf("recent order after restart: %q", got)
	}
	second.submitCommand(":e #0")
	waitUI(t, second)
	second.ui.modelMu.Lock()
	actual := second.frame.Path
	second.ui.modelMu.Unlock()
	if filepath.Clean(actual) != paths[1] {
		t.Fatalf("restart #0 opened %q", actual)
	}
	data, err := os.ReadFile(filepath.Join(store, "history.dat"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), ":e #0\r\n") {
		t.Fatalf("newest history is not first: %q", data)
	}
}

func TestGioPersistentHistoryLimitsAndDeduplication(t *testing.T) {
	store := t.TempDir()
	lines := []string{}
	for i := 0; i < 150; i++ {
		lines = append(lines, fmt.Sprintf(":edit model-%d.inp", i))
	}
	if err := writeHistoryLines(filepath.Join(store, "history.dat"), lines); err != nil {
		t.Fatal(err)
	}
	w := NewWindow(nil)
	defer w.ui.close()
	if err := w.LoadHistory(store); err != nil {
		t.Fatal(err)
	}
	w.ui.historyMu.Lock()
	if len(w.ui.commandHistory) != 100 {
		t.Fatalf("history size %d", len(w.ui.commandHistory))
	}
	w.addUICommandHistoryLocked(lines[50])
	w.addUICommandHistoryLocked(lines[0])
	w.ui.historyMu.Unlock()
	if err := w.SaveCommandHistory(""); err != nil {
		t.Fatal(err)
	}
	restored, err := readHistoryLines(filepath.Join(store, "history.dat"), 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(restored) != 100 || restored[0] != lines[0] || restored[1] != lines[50] {
		t.Fatalf("MRU order/limit: %v", restored[:3])
	}
	seen := map[string]bool{}
	for _, line := range restored {
		if seen[line] {
			t.Fatalf("duplicate entry: %s", line)
		}
		seen[line] = true
	}
}

func TestGioMalformedHistoryIsNotOverwritten(t *testing.T) {
	store := t.TempDir()
	path := filepath.Join(store, "history.dat")
	original := []byte{0xff, 0xfe, 0, 1}
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	w := NewWindow(nil)
	defer w.ui.close()
	if err := w.LoadHistory(store); err == nil {
		t.Fatal("invalid history accepted")
	}
	if err := w.SaveCommandHistory(""); err == nil {
		t.Fatal("overwrote unreadable history")
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(original) {
		t.Fatal("history was modified")
	}
}
