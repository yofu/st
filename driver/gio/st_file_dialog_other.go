//go:build !windows

package stgio

import (
	"fmt"
	"gioui.org/io/event"
)

func nativeDialogsSupported() bool                { return false }
func (w *Window) platformWindowEvent(event.Event) {}
func platformFileDialog(uintptr, fileDialogOptions) (string, bool, error) {
	return "", false, fmt.Errorf("native Windows file dialogs are unavailable")
}
