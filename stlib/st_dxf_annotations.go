package st

import (
	"fmt"
	"math"
	"strings"

	"github.com/yofu/dxf"
	"github.com/yofu/dxf/drawing"
)

// User annotations remain anchored to the model; offsets use paper mm so a
// new plot scale does not undo a previously reviewed text adjustment.
func (f *Frame) drawDXFDrawingAnnotations(d *drawing.Drawing, view DXFViewSpec, plotScale float64) error {
	if f.drawingSettings == nil {
		return nil
	}
	axis, x, y, lo, hi := 2, 0, 1, 0., 0.
	key := fmt.Sprintf("plan:%d", view.Floor)
	if view.Kind == "plan" {
		lo, hi = f.Ai.Boundary[view.Floor-1], f.Ai.Boundary[view.Floor]
	} else {
		var err error
		axis, lo, err = f.resolveDxfJiku(view.Grid)
		if err != nil {
			return err
		}
		hi = lo + .1
		lo -= .1
		x, y = 1-axis, 2
		key = "elevation:" + view.Grid
	}
	return f.drawDXFAnnotationsInRange(d, key, axis, lo, hi, x, y, plotScale)
}

func (f *Frame) drawDXFAnnotationsInRange(d *drawing.Drawing, key string, axis int, lo, hi float64, x, y int, plotScale float64) error {
	if f.drawingSettings == nil {
		return nil
	}
	anchor := func(id int) ([2]float64, bool) {
		e := f.Elems[id]
		var p [2]float64
		if e == nil || len(e.Enod) == 0 {
			return p, false
		}
		mn, mx := math.Inf(1), math.Inf(-1)
		for _, n := range e.Enod {
			p[0] += n.Coord[x] * 1000 / float64(len(e.Enod))
			p[1] += n.Coord[y] * 1000 / float64(len(e.Enod))
			mn = math.Min(mn, n.Coord[axis])
			mx = math.Max(mx, n.Coord[axis])
		}
		return p, mn <= hi && mx >= lo
	}
	if _, err := d.Style("ST_DRAWING", true); err != nil {
		d.AddStyle("ST_DRAWING", f.dxfDraftingStyle().Font, "", true)
	}
	d.AddLayer("DRAWING_NOTES", dxf.DefaultColor, dxf.DefaultLineType, true)
	text := func(value string, p [2]float64, offset [2]float64) error {
		offset = f.dxfPaperOffset(offset)
		for i, line := range strings.Split(value, "\n") {
			if _, err := d.Text(line, p[0]+offset[0]*plotScale, p[1]+(offset[1]-float64(i)*4)*plotScale, 0, f.dxfDraftingStyle().NoteTextHeightMM*plotScale); err != nil {
				return err
			}
		}
		return nil
	}
	for _, a := range f.drawingSettings.Annotations {
		if a.View != "all" && !strings.EqualFold(a.View, key) {
			continue
		}
		var p [2]float64
		if a.PositionMM != nil {
			p = *a.PositionMM
		} else {
			var ok bool
			p, ok = anchor(a.Element)
			if !ok {
				continue
			}
		}
		if err := text(a.Text, p, a.OffsetPaperMM); err != nil {
			return err
		}
	}
	// Iterate the already sorted resolution report to keep entity order stable.
	ids := SortedElem(f.Elems, func(e *Elem) float64 { return float64(e.Num) })
	for _, e := range ids {
		m := f.resolvedDXFMember(e)
		if m.Notes == nil || len(*m.Notes) == 0 {
			continue
		}
		p, ok := anchor(e.Num)
		if !ok {
			continue
		}
		if err := text(strings.Join(*m.Notes, "\n"), p, [2]float64{0, -6}); err != nil {
			return err
		}
	}
	return nil
}

func (f *Frame) dxfDrawingViewKey(axis int, min, max float64) string {
	if f.drawingView != "" {
		return f.drawingView
	}
	if axis == 2 {
		for i := 1; i < len(f.Ai.Boundary); i++ {
			if math.Abs(f.Ai.Boundary[i-1]-min) < 1e-5 && math.Abs(f.Ai.Boundary[i]-max) < 1e-5 {
				return fmt.Sprintf("plan:%d", i)
			}
		}
	}
	if axis < 2 {
		for _, g := range f.dxfGrids(axis) {
			if math.Abs(g.value-(min+max)*.5) < 1e-5 {
				return "elevation:" + g.name
			}
		}
	}
	return "crosssection"
}
