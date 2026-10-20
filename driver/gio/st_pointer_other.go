//go:build !windows

package stgio

import (
	"gioui.org/f32"
	"time"
)

func platformClickSettings(uintptr) mouseClickSettings {
	return mouseClickSettings{delay: 500 * time.Millisecond, radius: f32.Pt(2, 2)}
}

func platformPointerPosition(uintptr) (f32.Point, bool) { return f32.Point{}, false }
