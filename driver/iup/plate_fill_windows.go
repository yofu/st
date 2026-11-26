//go:build windows

package stgui

/*
#cgo windows LDFLAGS: -lcd
#include <stdint.h>
#include <cd.h>

static uintptr_t st_canvas_hdc(uintptr_t canvas) {
	if (canvas == 0) {
		return 0;
	}
	return (uintptr_t)cdCanvasGetAttribute((cdCanvas*)canvas, "HDC");
}
*/
import "C"
import (
	"sync"
	"syscall"
	"unsafe"

	"github.com/yofu/go-iup/cd"
)

type gdiplusStartupInput struct {
	GdiplusVersion           uint32
	DebugEventCallback       uintptr
	SuppressBackgroundThread int32
	SuppressExternalCodecs   int32
}

type gpPointF struct {
	X float32
	Y float32
}

const (
	gpStatusOk             = 0
	gpFillModeAlternate    = 0
	gpSmoothingModeDefault = 0
	gpSmoothingModeAA      = 4
)

var (
	modgdi32     = syscall.NewLazyDLL("gdi32.dll")
	procGdiFlush = modgdi32.NewProc("GdiFlush")

	modgdiplus               = syscall.NewLazyDLL("gdiplus.dll")
	procGdiplusStartup       = modgdiplus.NewProc("GdiplusStartup")
	procGdipCreateFromHDC    = modgdiplus.NewProc("GdipCreateFromHDC")
	procGdipDeleteGraphics   = modgdiplus.NewProc("GdipDeleteGraphics")
	procGdipSetSmoothingMode = modgdiplus.NewProc("GdipSetSmoothingMode")
	procGdipCreateSolidFill  = modgdiplus.NewProc("GdipCreateSolidFill")
	procGdipDeleteBrush      = modgdiplus.NewProc("GdipDeleteBrush")
	procGdipFillPolygon      = modgdiplus.NewProc("GdipFillPolygon")

	gdiplusInitOnce sync.Once
	gdiplusToken    uintptr
	gdiplusReady    bool
)

func ensureGdiplus() bool {
	gdiplusInitOnce.Do(func() {
		if err := modgdiplus.Load(); err != nil {
			return
		}
		input := gdiplusStartupInput{GdiplusVersion: 1}
		status, _, _ := procGdiplusStartup.Call(
			uintptr(unsafe.Pointer(&gdiplusToken)),
			uintptr(unsafe.Pointer(&input)),
			0,
		)
		gdiplusReady = (status == gpStatusOk)
	})
	return gdiplusReady
}

func canvasHDC(canv *cd.Canvas) uintptr {
	if canv == nil {
		return 0
	}
	return uintptr(C.st_canvas_hdc(C.uintptr_t(canv.Native())))
}

func canFillTranslucentPolygon(canv *cd.Canvas) bool {
	if canv == nil || !ensureGdiplus() {
		return false
	}
	return canvasHDC(canv) != 0
}

func fillTranslucentPolygonHDC(hdc uintptr, height int, coords [][]float64, rgb int, alpha int) bool {
	if hdc == 0 || len(coords) < 3 || height <= 0 || !ensureGdiplus() {
		return false
	}
	procGdiFlush.Call()

	var graphics uintptr
	status, _, _ := procGdipCreateFromHDC.Call(hdc, uintptr(unsafe.Pointer(&graphics)))
	if status != gpStatusOk || graphics == 0 {
		return false
	}
	defer procGdipDeleteGraphics.Call(graphics)

	procGdipSetSmoothingMode.Call(graphics, gpSmoothingModeAA)

	argb := (uint32(alpha&0xff) << 24) | (uint32(rgb) & 0x00ffffff)
	var brush uintptr
	status, _, _ = procGdipCreateSolidFill.Call(uintptr(argb), uintptr(unsafe.Pointer(&brush)))
	if status != gpStatusOk || brush == 0 {
		return false
	}
	defer procGdipDeleteBrush.Call(brush)

	pts := make([]gpPointF, len(coords))
	h := float32(height - 1)
	for i, c := range coords {
		pts[i] = gpPointF{
			X: float32(c[0]),
			Y: h - float32(c[1]),
		}
	}

	status, _, _ = procGdipFillPolygon.Call(
		graphics,
		brush,
		uintptr(unsafe.Pointer(&pts[0])),
		uintptr(len(pts)),
		gpFillModeAlternate,
	)
	return status == gpStatusOk
}

func fillTranslucentPolygon(canv *cd.Canvas, coords [][]float64, rgb int, alpha int) bool {
	hdc := canvasHDC(canv)
	if hdc == 0 {
		return false
	}
	_, h := canv.GetSize()
	return fillTranslucentPolygonHDC(hdc, h, coords, rgb, alpha)
}
