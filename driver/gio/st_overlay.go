package stgio

import (
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// Only the frame goroutine owns this transient front-canvas state. It must
// follow input without waiting for the model/analysis command worker.
func (u *userInterface) updateSelectionOverlay(ev event.Event) {
	switch e := ev.(type) {
	case pointer.Event:
		switch e.Kind {
		case pointer.Press:
			if !u.selectionOverlay && e.Buttons&pointer.ButtonPrimary != 0 && e.Buttons&pointer.ButtonTertiary == 0 {
				u.selectionOverlay = true
				u.selectionPointer = e.PointerID
				u.selectionStart, u.selectionEnd = e.Position, e.Position
			}
		case pointer.Drag:
			if u.selectionOverlay && u.selectionPointer == e.PointerID {
				u.selectionEnd = e.Position
				if u.fenceOverlay && e.Modifiers&key.ModCtrl != 0 {
					u.selectionEnd.Y = u.selectionStart.Y
				}
			}
		case pointer.Release:
			if u.selectionOverlay && u.selectionPointer == e.PointerID && e.Buttons&pointer.ButtonPrimary == 0 {
				u.selectionOverlay = false
			}
		case pointer.Cancel:
			u.selectionOverlay = false
		}
	case key.Event:
		if e.Name == key.NameEscape && e.State == key.Press {
			u.selectionOverlay = false
		}
	}
}

func (w *Window) drawCanvasOverlay(gtx layout.Context) {
	u := w.ui
	if len(u.commandTail) > 0 {
		var p clip.Path
		p.Begin(gtx.Ops)
		p.MoveTo(u.commandTail[0])
		for _, point := range u.commandTail[1:] {
			p.LineTo(point)
		}
		p.LineTo(u.cursorPosition)
		if len(u.commandTail) > 1 {
			p.LineTo(u.commandTail[0])
		}
		paint.FillShape(gtx.Ops, color.NRGBA{G: 130, B: 220, A: 255}, clip.Stroke{Path: p.End(), Width: 1}.Op())
	}
	if !u.selectionOverlay {
		return
	}
	start, end := u.selectionStart, u.selectionEnd
	if u.fenceOverlay {
		var path clip.Path
		path.Begin(gtx.Ops)
		path.MoveTo(start)
		path.LineTo(end)
		paint.FillShape(gtx.Ops, color.NRGBA{R: 255, A: 255}, clip.Stroke{Path: path.End(), Width: 1}.Op())
		return
	}
	points := []f32.Point{start, f32.Pt(end.X, start.Y), end, f32.Pt(start.X, end.Y), start}
	var path clip.Path
	path.Begin(gtx.Ops)
	for i := 0; i < 4; i++ {
		a, b := points[i], points[i+1]
		if end.X >= start.X {
			path.MoveTo(a)
			path.LineTo(b)
			continue
		}
		// IUP uses dashed outlines for right-to-left drag selection.
		d := b.Sub(a)
		length := float32(math.Hypot(float64(d.X), float64(d.Y)))
		for s := float32(0); s < length; s += 8 {
			path.MoveTo(a.Add(d.Mul(s / length)))
			path.LineTo(a.Add(d.Mul(min(s+4, length) / length)))
		}
	}
	paint.FillShape(gtx.Ops, color.NRGBA{R: 255, A: 255}, clip.Stroke{Path: path.End(), Width: 1}.Op())
}
