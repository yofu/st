//go:build js && wasm

package st

import (
	"errors"
	"syscall/js"
)

var errBrowserClipboard = errors.New("desktop clipboard is unavailable in the browser viewer")

func readSystemClipboard() (string, error) { return "", errBrowserClipboard }

func writeSystemClipboard(value string) error {
	receive := js.Global().Get("stViewerBrowserAction")
	if receive.Type() != js.TypeFunction {
		return errBrowserClipboard
	}
	receive.Invoke("clipboard", []any{value})
	return nil
}
