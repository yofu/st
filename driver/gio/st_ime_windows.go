//go:build windows

package stgio

import (
	"syscall"

	"golang.org/x/sys/windows"
)

// Win32 IME constants (imm.h / winuser.h).
const (
	wmNCDestroy      = 0x0082
	wmIMEComposition = 0x010F
	gcsCompStr       = 0x0008
	gcsCursorPos     = 0x0080
	gcsDeltaStart    = 0x0100
	gcsResultStr     = 0x0800

	imeFixSubclassID = 0x5354494D // "STIM"
)

var (
	comctl32DLL              = windows.NewLazySystemDLL("comctl32.dll")
	setWindowSubclassProc    = comctl32DLL.NewProc("SetWindowSubclass")
	removeWindowSubclassProc = comctl32DLL.NewProc("RemoveWindowSubclass")
	defSubclassProc          = comctl32DLL.NewProc("DefSubclassProc")

	imeFixSubclassCallback uintptr
)

func init() {
	imeFixSubclassCallback = syscall.NewCallback(imeFixSubclassProc)
}

// filterIMECompositionLParam strips GCS_DELTASTART from WM_IME_COMPOSITION.
//
// In gioui.org v0.10.3 (app/os_windows.go), WM_IME_COMPOSITION adds
// GCS_DELTASTART to comp.Start before calling SetComposingRegion. In Windows
// IMM32, however, GCS_COMPSTR always returns the full composition string while
// GCS_DELTASTART is only the offset of the changed suffix within it. Advancing
// comp.Start detaches the unchanged prefix from Gio's composing region, causing
// every subsequent preedit update to append a duplicate copy of the prefix
// (e.g. "日本語" becoming "にににほにほにほん日本語").
func filterIMECompositionLParam(msg, lParam uintptr) uintptr {
	if msg == wmIMEComposition {
		return lParam &^ gcsDeltaStart
	}
	return lParam
}

func imeFixSubclassProc(hwnd, msg, wParam, lParam, uIdSubclass, dwRefData uintptr) uintptr {
	lParam = filterIMECompositionLParam(msg, lParam)
	if msg == wmNCDestroy {
		removeWindowSubclassProc.Call(hwnd, imeFixSubclassCallback, uIdSubclass)
	}
	ret, _, _ := defSubclassProc.Call(hwnd, msg, wParam, lParam)
	return ret
}

// installIMEFixSubclass must be called on the OS thread that owns hwnd (via app.Window.Run).
func installIMEFixSubclass(hwnd uintptr) bool {
	if hwnd == 0 {
		return false
	}
	for _, p := range []*windows.LazyProc{setWindowSubclassProc, removeWindowSubclassProc, defSubclassProc} {
		if p.Find() != nil {
			return false
		}
	}
	ok, _, _ := setWindowSubclassProc.Call(hwnd, imeFixSubclassCallback, imeFixSubclassID, 0)
	return ok != 0
}
