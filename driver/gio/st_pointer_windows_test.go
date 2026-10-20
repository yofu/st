package stgio

import (
	"runtime"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestWindowsCursorClientPixelsAndDPIRestore(t *testing.T) {
	// A hidden owned test window exercises the real Win32 coordinate backend
	// without moving the user's mouse or displaying a native window.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	for _, p := range []*windows.LazyProc{cursorPositionProc, screenToClientProc, windowDPIContextProc, threadDPIContextProc} {
		if err := p.Find(); err != nil {
			t.Skip(err)
		}
	}
	old, _, _ := threadDPIContextProc.Call(^uintptr(3)) // PER_MONITOR_AWARE_V2 (-4)
	if old == 0 {
		t.Skip("per-monitor DPI context unavailable")
	}
	defer threadDPIContextProc.Call(old)
	class, _ := windows.UTF16PtrFromString("STATIC")
	create := pointerUser32.NewProc("CreateWindowExW")
	destroy := pointerUser32.NewProc("DestroyWindow")
	hwnd, _, err := create.Call(0, uintptr(unsafe.Pointer(class)), 0, 0x80000000, 120, 160, 640, 480, 0, 0, 0, 0)
	if hwnd == 0 {
		t.Fatal(err)
	}
	defer destroy.Call(hwnd)
	// First obtain a reference under the native window's awareness context.
	var before, after struct{ X, Y int32 }
	cursorPositionProc.Call(uintptr(unsafe.Pointer(&before)))
	screenToClientProc.Call(hwnd, uintptr(unsafe.Pointer(&before)))
	// Simulate a render thread with DPI_UNAWARE, unlike the window thread.
	threadDPIContextProc.Call(^uintptr(0)) // -1
	getThread := pointerUser32.NewProc("GetThreadDpiAwarenessContext")
	initial, _, _ := getThread.Call()
	p, ok := platformPointerPosition(hwnd)
	if !ok {
		t.Fatal("native client cursor lookup failed")
	}
	restored, _, _ := getThread.Call()
	if initial != restored {
		t.Fatal("DPI context leaked onto render thread")
	}
	threadDPIContextProc.Call(^uintptr(3))
	cursorPositionProc.Call(uintptr(unsafe.Pointer(&after)))
	screenToClientProc.Call(hwnd, uintptr(unsafe.Pointer(&after)))
	// The user may move the mouse concurrently; allow its before/after range.
	if p.X < float32(min(before.X, after.X)-2) || p.X > float32(max(before.X, after.X)+2) || p.Y < float32(min(before.Y, after.Y)-2) || p.Y > float32(max(before.Y, after.Y)+2) {
		t.Fatalf("native cursor used different DPI units: before=%v got=%v after=%v", before, p, after)
	}
	dpi, _, _ := pointerUser32.NewProc("GetDpiForWindow").Call(hwnd)
	t.Logf("native window DPI=%d (scale %.2f); client cursor=%v", dpi, float64(dpi)/96, p)
	if _, ok := platformPointerPosition(0); ok {
		t.Fatal("zero HWND accepted")
	}
}
