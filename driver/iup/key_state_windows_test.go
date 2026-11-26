//go:build windows

package stgui

import (
	"runtime"
	"syscall"
	"testing"

	"github.com/yofu/go-iup/iup"
	st "github.com/yofu/st/stlib"
)

func TestKeyStateAndPrintableKeyText(t *testing.T) {
	if iup.IupVersionNumber < 309000 {
		t.Skipf("requires IUP >= 3.9 key encoding, got %d", iup.IupVersionNumber)
	}

	ctrlZ := iup.KeyState(0x20000000 | 'Z')
	if ctrlZ.Key() != 'Z' || !ctrlZ.IsCtrl() || ctrlZ.IsAlt() || ctrlZ.IsShift() {
		t.Fatalf("Ctrl+Z decoded unexpectedly: key=%#x ctrl=%v alt=%v shift=%v", ctrlZ.Key(), ctrlZ.IsCtrl(), ctrlZ.IsAlt(), ctrlZ.IsShift())
	}
	if _, ok := printableKeyText(ctrlZ); ok {
		t.Fatal("Ctrl+Z must not be treated as printable text")
	}

	altH := iup.KeyState(0x40000000 | 'H')
	if altH.Key() != 'H' || !altH.IsAlt() || altH.IsCtrl() {
		t.Fatalf("Alt+H decoded unexpectedly: key=%#x alt=%v ctrl=%v", altH.Key(), altH.IsAlt(), altH.IsCtrl())
	}
	if _, ok := printableKeyText(altH); ok {
		t.Fatal("Alt+H must not be treated as printable text")
	}

	plainA := iup.KeyState('a')
	if s, ok := printableKeyText(plainA); !ok || s != "a" {
		t.Fatalf("plain 'a' expected (\"a\", true), got (%q, %v)", s, ok)
	}

	// Modifier-only key (e.g. K_LCTRL = 0xFFE3) and special keys must not append junk to cline.
	for _, raw := range []int{0xFFE1, 0xFFE3, 0xFFE9, KEY_ESCAPE, KEY_DELETE, KEY_UPARROW, KEY_F4} {
		if s, ok := printableKeyText(iup.KeyState(raw)); ok {
			t.Fatalf("special/modifier key %#x unexpectedly treated as printable: %q", raw, s)
		}
	}
}

func TestLiveEscapeKeyDelivery(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := iup.Open(); err != nil {
		t.Fatal(err)
	}
	defer iup.Close()

	t.Logf("runtime IUP VERSION=%q header=%q (%d) KEY_ESCAPE=%#x", iup.GetGlobal("VERSION"), iup.IupVersion, iup.IupVersionNumber, KEY_ESCAPE)

	var canvKeys []int32
	var textKeys []int32
	canv := iup.Canvas("CANFOCUS=YES", "SIZE=100x100", func(arg *iup.CommonKeyAny) {
		canvKeys = append(canvKeys, arg.Key)
	})
	txt := iup.Text("CANFOCUS=YES", "SIZE=100x10", func(arg *iup.CommonKeyAny) {
		textKeys = append(textKeys, arg.Key)
	})
	dlg := iup.Dialog(iup.Vbox(canv, txt))
	defer dlg.Destroy()
	dlg.Map()

	user32 := syscall.NewLazyDLL("user32.dll")
	procSendMessageW := user32.NewProc("SendMessageW")
	procPostMessageW := user32.NewProc("PostMessageW")
	const (
		wmKeyDown = 0x0100
		wmKeyUp   = 0x0101
		vkEscape  = 0x1B
		vkDelete  = 0x2E
		vkUp      = 0x26
	)

	canvHWND := canv.GetAttributeData("WID")
	txtHWND := txt.GetAttributeData("WID")
	if canvHWND == 0 || txtHWND == 0 {
		t.Fatalf("missing WID: canv=%#x txt=%#x", canvHWND, txtHWND)
	}

	procSendMessageW.Call(canvHWND, wmKeyDown, vkEscape, 0x00010001)
	procSendMessageW.Call(canvHWND, wmKeyDown, vkDelete, 0x01530001)
	procSendMessageW.Call(canvHWND, wmKeyDown, vkUp, 0x01480001)
	procSendMessageW.Call(txtHWND, wmKeyDown, vkEscape, 0x00010001)

	t.Logf("SendMessage canvKeys=%#x textKeys=%#x", canvKeys, textKeys)
	if len(canvKeys) == 0 || int(iup.KeyState(canvKeys[0]).Key()) != KEY_ESCAPE {
		t.Fatalf("canvas Escape mismatch: got %#x, want KEY_ESCAPE=%#x", canvKeys, KEY_ESCAPE)
	}
	if len(textKeys) == 0 || int(iup.KeyState(textKeys[0]).Key()) != KEY_ESCAPE {
		t.Fatalf("text Escape mismatch: got %#x, want KEY_ESCAPE=%#x", textKeys, KEY_ESCAPE)
	}

	// Also verify delivery through IUP's message loop (PostMessage + LoopStep).
	canvKeys = nil
	textKeys = nil
	procPostMessageW.Call(canvHWND, wmKeyDown, vkEscape, 0x00010001)
	procPostMessageW.Call(canvHWND, wmKeyUp, vkEscape, 0xC0010001)
	procPostMessageW.Call(txtHWND, wmKeyDown, vkEscape, 0x00010001)
	procPostMessageW.Call(txtHWND, wmKeyUp, vkEscape, 0xC0010001)
	for i := 0; i < 20; i++ {
		iup.LoopStep()
	}
	t.Logf("LoopStep canvKeys=%#x textKeys=%#x", canvKeys, textKeys)
	if len(canvKeys) == 0 || int(iup.KeyState(canvKeys[0]).Key()) != KEY_ESCAPE {
		t.Fatalf("LoopStep canvas Escape mismatch: got %#x, want KEY_ESCAPE=%#x", canvKeys, KEY_ESCAPE)
	}
	if len(textKeys) == 0 || int(iup.KeyState(textKeys[0]).Key()) != KEY_ESCAPE {
		t.Fatalf("LoopStep text Escape mismatch: got %#x, want KEY_ESCAPE=%#x", textKeys, KEY_ESCAPE)
	}
}

func TestNewWindowEscapeKey(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	dir := t.TempDir()
	t.Chdir(dir)
	if err := iup.Open(); err != nil {
		t.Fatal(err)
	}
	defer iup.Close()

	w := NewWindow(dir)
	defer StopLogging()
	defer w.Dlg.Destroy()

	user32 := syscall.NewLazyDLL("user32.dll")
	procPostMessageW := user32.NewProc("PostMessageW")
	const (
		wmKeyDown = 0x0100
		wmKeyUp   = 0x0101
		vkEscape  = 0x1B
	)

	// Case 1: focus on cline with text typed -> Escape clears cline and moves focus to canv.
	w.cline.SetAttribute("VALUE", ":test")
	iup.SetFocus(w.cline)
	clineHWND := w.cline.GetAttributeData("WID")
	procPostMessageW.Call(clineHWND, wmKeyDown, vkEscape, 0x00010001)
	procPostMessageW.Call(clineHWND, wmKeyUp, vkEscape, 0xC0010001)
	for i := 0; i < 20; i++ {
		iup.LoopStep()
	}
	if got := w.cline.GetAttribute("VALUE"); got != "" {
		t.Fatalf("Escape on cline did not clear VALUE: got %q", got)
	}

	// Case 2: focus on canv with node selected and custom mode -> Escape runs EscapeAll().
	n := w.frame.AddNode(0, 0, 0)
	w.SelectNode([]*st.Node{n})
	w.cname.SetAttribute("VALUE", "CUSTOM")
	iup.SetFocus(w.canv)
	canvHWND := w.canv.GetAttributeData("WID")
	procPostMessageW.Call(canvHWND, wmKeyDown, vkEscape, 0x00010001)
	procPostMessageW.Call(canvHWND, wmKeyUp, vkEscape, 0xC0010001)
	for i := 0; i < 20; i++ {
		iup.LoopStep()
	}
	if got := w.cname.GetAttribute("VALUE"); got != "SELECT" {
		t.Fatalf("Escape on canv did not reset mode to SELECT: got %q", got)
	}
	if len(w.SelectedNodes()) != 0 {
		t.Fatalf("Escape on canv did not deselect nodes: got %d", len(w.SelectedNodes()))
	}
}
