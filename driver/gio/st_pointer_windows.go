package stgio

import (
	"runtime"
	"time"
	"unsafe"

	"gioui.org/f32"
	"golang.org/x/sys/windows"
)

var pointerUser32 = windows.NewLazySystemDLL("user32.dll")
var cursorPositionProc = pointerUser32.NewProc("GetCursorPos")
var screenToClientProc = pointerUser32.NewProc("ScreenToClient")
var windowDPIContextProc = pointerUser32.NewProc("GetWindowDpiAwarenessContext")
var threadDPIContextProc = pointerUser32.NewProc("SetThreadDpiAwarenessContext")

func platformClickSettings(hwnd uintptr) mouseClickSettings {
	s := mouseClickSettings{delay: 500 * time.Millisecond, radius: f32.Pt(2, 2)}
	p := pointerUser32.NewProc("GetDoubleClickTime")
	if p.Find() == nil {
		if v, _, _ := p.Call(); v > 0 {
			s.delay = time.Duration(v) * time.Millisecond
		}
	}
	dpi := uintptr(96)
	if hwnd != 0 {
		p = pointerUser32.NewProc("GetDpiForWindow")
		if p.Find() == nil {
			if v, _, _ := p.Call(hwnd); v > 0 {
				dpi = v
			}
		}
	}
	p = pointerUser32.NewProc("GetSystemMetricsForDpi")
	if p.Find() == nil {
		x, _, _ := p.Call(36, dpi)
		y, _, _ := p.Call(37, dpi)
		if x > 0 && y > 0 {
			s.radius = f32.Pt(float32(x)/2, float32(y)/2)
		}
	}
	return s
}

// Return client-area device pixels, matching Gio's FrameEvent.Size. The Gio
// render goroutine may run on a thread with a different DPI awareness context
// from the native window thread, so temporarily adopt the window's context.
func platformPointerPosition(hwnd uintptr) (f32.Point, bool) {
	if hwnd == 0 {
		return f32.Point{}, false
	}
	for _, p := range []*windows.LazyProc{cursorPositionProc, screenToClientProc, windowDPIContextProc, threadDPIContextProc} {
		if p.Find() != nil {
			return f32.Point{}, false
		}
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	context, _, _ := windowDPIContextProc.Call(hwnd)
	if context == 0 {
		return f32.Point{}, false
	}
	previous, _, _ := threadDPIContextProc.Call(context)
	if previous == 0 {
		return f32.Point{}, false
	}
	defer threadDPIContextProc.Call(previous)
	var point struct{ X, Y int32 }
	if ok, _, _ := cursorPositionProc.Call(uintptr(unsafe.Pointer(&point))); ok == 0 {
		return f32.Point{}, false
	}
	if ok, _, _ := screenToClientProc.Call(hwnd, uintptr(unsafe.Pointer(&point))); ok == 0 {
		return f32.Point{}, false
	}
	return f32.Pt(float32(point.X), float32(point.Y)), true
}
