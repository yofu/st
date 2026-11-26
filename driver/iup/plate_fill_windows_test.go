//go:build windows

package stgui

import (
	"syscall"
	"testing"
	"unsafe"
)

type bitmapInfoHeader struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

type bitmapInfo struct {
	BmiHeader bitmapInfoHeader
	BmiColors [1]uint32
}

func TestFillTranslucentPolygonHDC(t *testing.T) {
	gdi32 := syscall.NewLazyDLL("gdi32.dll")
	procCreateCompatibleDC := gdi32.NewProc("CreateCompatibleDC")
	procDeleteDC := gdi32.NewProc("DeleteDC")
	procCreateDIBSection := gdi32.NewProc("CreateDIBSection")
	procSelectObject := gdi32.NewProc("SelectObject")
	procDeleteObject := gdi32.NewProc("DeleteObject")

	const width, height = 64, 64
	hdc, _, _ := procCreateCompatibleDC.Call(0)
	if hdc == 0 {
		t.Fatal("CreateCompatibleDC failed")
	}
	defer procDeleteDC.Call(hdc)

	bi := bitmapInfo{
		BmiHeader: bitmapInfoHeader{
			BiSize:     uint32(unsafe.Sizeof(bitmapInfoHeader{})),
			BiWidth:    width,
			BiHeight:   -height, // top-down DIB
			BiPlanes:   1,
			BiBitCount: 32,
		},
	}
	var bits unsafe.Pointer
	hbmp, _, _ := procCreateDIBSection.Call(
		hdc,
		uintptr(unsafe.Pointer(&bi)),
		0,
		uintptr(unsafe.Pointer(&bits)),
		0,
		0,
	)
	if hbmp == 0 || bits == nil {
		t.Fatal("CreateDIBSection failed")
	}
	defer procDeleteObject.Call(hbmp)

	oldBmp, _, _ := procSelectObject.Call(hdc, hbmp)
	defer procSelectObject.Call(hdc, oldBmp)

	// Draw a red (0xFF0000) translucent polygon at bottom-left in CD coords:
	// x in [10, 40], y in [10, 30] -> top-down Y in [33, 53].
	poly := [][]float64{
		{10, 10},
		{40, 10},
		{40, 30},
		{10, 30},
	}
	if !fillTranslucentPolygonHDC(hdc, height, poly, 0xFF0000, plateAlphaDefault) {
		t.Fatal("fillTranslucentPolygonHDC returned false")
	}
	procGdiFlush.Call()

	pixels := unsafe.Slice((*uint32)(bits), width*height)

	// Inside polygon: CD (25, 20) -> top-down (25, 63-20 = 43)
	inside := pixels[43*width+25]
	b := byte(inside & 0xff)
	g := byte((inside >> 8) & 0xff)
	r := byte((inside >> 16) & 0xff)
	if r < 0x50 || r > 0x5A || g != 0 || b != 0 {
		t.Fatalf("unexpected inside pixel RGB=(%d,%d,%d), want ~(%d,0,0)", r, g, b, plateAlphaDefault)
	}

	// Outside polygon: top-down (25, 10) -> CD y = 53 (above polygon)
	outside := pixels[10*width+25]
	if outside&0x00ffffff != 0 {
		t.Fatalf("expected outside pixel black, got 0x%06x", outside&0x00ffffff)
	}

	// Draw an overlapping blue (0x0000FF) translucent polygon over the same area
	// and verify both red and blue channels remain visible (true alpha blending).
	if !fillTranslucentPolygonHDC(hdc, height, poly, 0x0000FF, plateAlphaDefault) {
		t.Fatal("second fillTranslucentPolygonHDC returned false")
	}
	procGdiFlush.Call()

	blended := pixels[43*width+25]
	bb := byte(blended & 0xff)
	bg := byte((blended >> 8) & 0xff)
	br := byte((blended >> 16) & 0xff)
	if br == 0 || bb == 0 || bg != 0 {
		t.Fatalf("expected blended red+blue pixel, got RGB=(%d,%d,%d)", br, bg, bb)
	}
}
