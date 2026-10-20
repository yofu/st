package stgio

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gioui.org/io/pointer"
	st "github.com/yofu/st/stlib"
)

func writePgp(t *testing.T, home, content string) string {
	t.Helper()
	path := filepath.Join(home, ".st", "st.pgp")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
func runPgpAlias(t *testing.T, h *uiHarness, name string) {
	t.Helper()
	h.w.submitCommand(name)
	waitUI(t, h.w)
	h.frame()
	h.frame()
}
func TestPgpStartupAndInteractiveAlias(t *testing.T) {
	h := interactiveFixture(t)
	home := t.TempDir()
	// BOM, CRLF, tabs, comments, case-insensitive names and later definitions.
	writePgp(t, home, "\ufeff  # comment\r\nd\tDISTS\r\nA ADDLINEELEM\r\nX DISTS\r\nX ADDLINEELEM\r\nF IUP_ONLY_COMMAND\r\n")
	h.w.SetHome(home)
	if err := h.w.LoadAliases(); err != nil {
		t.Fatal(err)
	}
	s := startCommand(t, h, "d")
	clickInteractive(t, h, s, pointOf(h.w.frame.Nodes[1]), pointer.ButtonPrimary)
	clickInteractive(t, h, s, pointOf(h.w.frame.Nodes[2]), pointer.ButtonPrimary)
	sessionDone(t, s)
	count := len(h.w.frame.Elems)
	s = startCommand(t, h, "x")
	clickInteractive(t, h, s, pointOf(h.w.frame.Nodes[1]), pointer.ButtonPrimary)
	clickInteractive(t, h, s, pointOf(h.w.frame.Nodes[3]), pointer.ButtonPrimary)
	sessionDone(t, s)
	if len(h.w.frame.Elems) != count+1 {
		t.Fatal("last duplicate alias did not win")
	}
	if _, ok := st.Commands["D"]; ok {
		t.Fatal("PGP changed global Commands")
	}
	other := newUIHarness(t)
	if _, ok := other.w.CommandAlias("D"); ok {
		t.Fatal("PGP leaked to another window")
	}
	h.w.ui.mu.Lock()
	log := h.w.ui.log
	h.w.ui.mu.Unlock()
	if !strings.Contains(log, "IUP_ONLY_COMMAND") {
		t.Fatal("unknown target was not reported")
	}
}
func TestPgpExFig2TemplateAndCounter(t *testing.T) {
	h := interactiveFixture(t)
	home := t.TempDir()
	path := writePgp(t, home, "CP :conf 111000\nGF 'gfact 42\nNS :nodesort _\nJN :node @1\n")
	if err := h.w.ReadPgp(path); err != nil {
		t.Fatal(err)
	}
	n := h.w.frame.Nodes[1]
	h.w.SelectNode([]*st.Node{n})
	runPgpAlias(t, h, "CP")
	for i := 0; i < 6; i++ {
		if n.Conf[i] != (i < 3) {
			t.Fatalf("conf[%d]=%v", i, n.Conf[i])
		}
	}
	runPgpAlias(t, h, "GF")
	if h.w.frame.View.Gfact != 42 {
		t.Fatal("Fig2 alias did not execute")
	}
	runPgpAlias(t, h, "NS")
	if text := h.w.ui.command.Text(); text != ":nodesort " {
		t.Fatalf("template text=%q", text)
	}
	start, end := h.w.ui.command.Selection()
	if start != 10 || end != 10 {
		t.Fatalf("template caret=%d,%d", start, end)
	}
	if !h.router.Source().Focused(&h.w.ui.command) {
		t.Fatal("template did not focus editor")
	}
	for i := 1; i <= 2; i++ {
		runPgpAlias(t, h, "JN")
		ns := h.w.SelectedNodes()
		if len(ns) != 1 || ns[0].Num != i {
			t.Fatalf("counter %d selected %v", i, ns)
		}
	}
}
func TestPgpMissingReloadAndFailedRead(t *testing.T) {
	h := newUIHarness(t)
	home := t.TempDir()
	h.w.SetHome(home)
	if err := h.w.LoadAliases(); err != nil {
		t.Fatal(err)
	}
	if _, ok := h.w.CommandAlias("DISTS"); !ok {
		t.Fatal("missing PGP removed defaults")
	}
	path := writePgp(t, home, "D DISTS\n")
	if err := h.w.LoadAliases(); err != nil {
		t.Fatal(err)
	}
	// Runtime :read pgp uses the driver loader through the common dispatcher.
	if err := os.WriteFile(path, []byte("A ADDLINEELEM\n"), 0644); err != nil {
		t.Fatal(err)
	}
	h.w.ui.modelMu.Lock()
	err := st.ExMode(h.w, ":read pgp "+path)
	h.w.ui.modelMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := h.w.CommandAlias("D"); ok {
		t.Fatal("reload retained removed alias")
	}
	if _, ok := h.w.CommandAlias("A"); !ok {
		t.Fatal("reload did not add alias")
	}
	if err := os.WriteFile(path, []byte("B DISTS\n\xff\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := h.w.LoadAliases(); err == nil {
		t.Fatal("invalid UTF-8 accepted")
	}
	if _, ok := h.w.CommandAlias("A"); !ok {
		t.Fatal("failed read lost previous aliases")
	}
	if _, ok := h.w.CommandAlias("B"); ok {
		t.Fatal("failed read installed partial aliases")
	}
	if err := h.w.ReadPgp(t.TempDir()); err == nil {
		t.Fatal("directory accepted as PGP")
	}
}
