//go:build windows

package stgio

import (
	"runtime"
	"syscall"
	"testing"
	"unsafe"

	"gioui.org/io/key"
	"golang.org/x/sys/windows"
)

func TestFilterIMECompositionLParamStripsDeltaStart(t *testing.T) {
	in := uintptr(gcsCompStr | gcsDeltaStart | gcsCursorPos | gcsResultStr)
	got := filterIMECompositionLParam(wmIMEComposition, in)
	want := uintptr(gcsCompStr | gcsCursorPos | gcsResultStr)
	if got != want {
		t.Fatalf("filterIMECompositionLParam(WM_IME_COMPOSITION, %#x) = %#x, want %#x", in, got, want)
	}
	if other := filterIMECompositionLParam(0x0100, in); other != in {
		t.Fatalf("unrelated message modified: %#x != %#x", other, in)
	}
}

func TestIMEFixSubclassStripsDeltaStartOnRealHWND(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	user32 := windows.NewLazySystemDLL("user32.dll")
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	registerClassExW := user32.NewProc("RegisterClassExW")
	unregisterClassW := user32.NewProc("UnregisterClassW")
	createWindowExW := user32.NewProc("CreateWindowExW")
	destroyWindow := user32.NewProc("DestroyWindow")
	sendMessageW := user32.NewProc("SendMessageW")
	defWindowProcW := user32.NewProc("DefWindowProcW")
	getModuleHandleW := kernel32.NewProc("GetModuleHandleW")

	hinst, _, _ := getModuleHandleW.Call(0)
	var receivedLParam uintptr
	wndProc := syscall.NewCallback(func(hwnd, msg, wParam, lParam uintptr) uintptr {
		if msg == wmIMEComposition {
			receivedLParam = lParam
			return 1
		}
		ret, _, _ := defWindowProcW.Call(hwnd, msg, wParam, lParam)
		return ret
	})

	className, err := windows.UTF16PtrFromString("STGioIMEFixTestClass")
	if err != nil {
		t.Fatal(err)
	}
	type wndClassExW struct {
		Size       uint32
		Style      uint32
		WndProc    uintptr
		ClsExtra   int32
		WndExtra   int32
		Instance   uintptr
		Icon       uintptr
		Cursor     uintptr
		Background uintptr
		MenuName   *uint16
		ClassName  *uint16
		IconSm     uintptr
	}
	wc := wndClassExW{
		Size:      uint32(unsafe.Sizeof(wndClassExW{})),
		WndProc:   wndProc,
		Instance:  hinst,
		ClassName: className,
	}
	atom, _, err := registerClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		t.Fatalf("RegisterClassExW failed: %v", err)
	}
	defer unregisterClassW.Call(uintptr(unsafe.Pointer(className)), hinst)

	const hwndMessage = ^uintptr(2) // (HWND)-3
	hwnd, _, err := createWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		0,
		0,
		0, 0, 0, 0,
		hwndMessage,
		0,
		hinst,
		0,
	)
	if hwnd == 0 {
		t.Fatalf("CreateWindowExW failed: %v", err)
	}
	defer destroyWindow.Call(hwnd)

	if !installIMEFixSubclass(hwnd) {
		t.Fatal("installIMEFixSubclass returned false")
	}

	raw := uintptr(gcsCompStr | gcsDeltaStart | gcsCursorPos)
	sendMessageW.Call(hwnd, wmIMEComposition, 0, raw)
	want := uintptr(gcsCompStr | gcsCursorPos)
	if receivedLParam != want {
		t.Fatalf("WndProc received lParam=%#x, want %#x (GCS_DELTASTART not stripped)", receivedLParam, want)
	}
}

func TestGioCommandEditorJapaneseIMEComposition(t *testing.T) {
	h := newUIHarness(t)
	w := h.w
	h.focus(&w.ui.command)

	// Simulate Windows IME composition after GCS_DELTASTART is stripped:
	// each preedit replaces the full composing range [0, prevLen] and final
	// RESULTSTR commits "日本語" cleanly without leaving preedit fragments.
	comp := key.Range{Start: -1, End: -1}
	replacePreedit := func(text string) {
		rng := comp
		if rng.Start == -1 {
			rng = key.Range{Start: 0, End: 0}
		}
		end := rng.Start + len([]rune(text))
		comp = key.Range{Start: rng.Start, End: end}
		h.router.Queue(
			key.EditEvent{Range: rng, Text: text},
			key.CompositionEvent(comp),
			key.SelectionEvent{Start: end, End: end},
		)
		h.frame()
	}
	for _, preedit := range []string{"ｎ", "に", "にｈ", "にほ", "にほｎ", "にほん", "にほんｇ", "にほんご", "日本語"} {
		replacePreedit(preedit)
	}
	// Commit via GCS_RESULTSTR.
	h.router.Queue(
		key.EditEvent{Range: comp, Text: "日本語"},
		key.CompositionEvent{Start: -1, End: -1},
		key.SelectionEvent{Start: 3, End: 3},
	)
	h.frame()

	if got := w.ui.command.Text(); got != "日本語" {
		t.Fatalf("committed Japanese command text = %q, want %q", got, "日本語")
	}
}
