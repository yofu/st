//go:build !windows

package stgui

import "github.com/yofu/go-iup/cd"

func canFillTranslucentPolygon(canv *cd.Canvas) bool {
	return false
}

func fillTranslucentPolygon(canv *cd.Canvas, coords [][]float64, rgb int, alpha int) bool {
	return false
}
