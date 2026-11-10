//go:build !js

package st

import "github.com/atotto/clipboard"

func readSystemClipboard() (string, error)    { return clipboard.ReadAll() }
func writeSystemClipboard(value string) error { return clipboard.WriteAll(value) }
