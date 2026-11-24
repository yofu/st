package stgui

import (
	"math"
	"reflect"
	"runtime"
	"testing"

	"github.com/yofu/go-iup/iup"
)

func TestMiddleDoubleClickWithoutElements(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	dir := t.TempDir()
	t.Chdir(dir)
	if err := iup.Open(); err != nil {
		t.Fatal(err)
	}
	defer iup.Close()
	w := NewWindow(dir)
	defer StopLogging()
	defer w.Dlg.Destroy()
	doubleClick := func() {
		w.canvasButton(&iup.MouseButton{Button: BUTTON_CENTER, Pressed: 1, Status: "   2 D    "})
		w.canvasButton(&iup.MouseButton{Button: BUTTON_CENTER, Pressed: 0, Status: "          "})
	}
	view := w.frame.View
	view.Focus[1] = 7
	view.Center[0] = 30
	checkUnchanged := func(model string) {
		t.Helper()
		before := view.Copy()
		doubleClick()
		if !reflect.DeepEqual(view, before) {
			t.Fatalf("%s: double-click changed the view from %+v to %+v", model, before, view)
		}
	}
	checkUnchanged("empty frame")
	first := w.frame.AddNode(0, 2, 0)
	checkUnchanged("single node")
	last := w.frame.AddNode(0, 6, 0)
	first.Hide()
	last.Hide()
	checkUnchanged("all nodes hidden")
	first.Show()
	last.Show()
	doubleClick()
	width, _ := w.cdcanv.GetSize()
	midX := (first.Pcoord[0] + last.Pcoord[0]) / 2
	if !(math.Abs(midX-float64(width)/2) < 1e-6 && last.Pcoord[0] > first.Pcoord[0]) {
		t.Fatalf("two unconnected nodes did not fit the canvas: x=%v, %v, width=%v", first.Pcoord[0], last.Pcoord[0], width)
	}
}
