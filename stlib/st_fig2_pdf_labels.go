package st

import (
	"fmt"
	"math"
	"sort"
)

type fig2StressLabel struct {
	x, y, angle        float64
	align, color, size int
	text               string
}

type fig2LabelRect struct{ x0, y0, x1, y1 float64 }

func (a fig2LabelRect) overlaps(b fig2LabelRect) bool {
	return a.x0 < b.x1 && b.x0 < a.x1 && a.y0 < b.y1 && b.y0 < a.y1
}

// QueueFig2Stress defers stress text until the other PDF annotations have
// reserved their positions. Each value stays attached to its quarter point.
func (p *PDFCanvas) QueueFig2Stress(x, y float64, value string) bool {
	if !p.fig2 {
		return false
	}
	if !finitePDF(x, y, p.torient) {
		if p.layoutErr == nil {
			p.layoutErr = fmt.Errorf("invalid position for result label %q", value)
		}
		return true
	}
	p.stressLabels = append(p.stressLabels, fig2StressLabel{x, y, p.torient, p.alignment, p.foreground, p.fig2FontSize, value})
	return true
}

func fig2TextOrigin(align int, width, size float64) (dx, dy float64) {
	// Baselines use downward PDF screen coordinates. ReportLab Label anchors
	// are on the text box, so north and south meet opposite edges of the box.
	switch align {
	case CENTER, SOUTH, NORTH:
		dx = -width / 2
	case EAST, SOUTH_EAST, fig2NorthEast:
		dx = -width
	}
	switch align {
	case NORTH, fig2NorthWest, fig2NorthEast:
		dy = size * .82
	case SOUTH, SOUTH_WEST, SOUTH_EAST:
		dy = -size * .20
	default:
		dy = size * .31
	}
	return
}

const fig2NorthWest = -1
const fig2NorthEast = -2

func fig2LabelBounds(x, y, angle, dx, dy, width, size float64) fig2LabelRect {
	r := fig2LabelRect{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
	c, s := math.Cos(angle*math.Pi/180), math.Sin(angle*math.Pi/180)
	for _, qx := range []float64{dx - .55, dx + width + .55} {
		for _, qy := range []float64{dy - .88*size - .4, dy + .24*size + .4} {
			a, b := x+c*qx-s*qy, y+s*qx+c*qy
			r.x0, r.x1 = min(r.x0, a), max(r.x1, a)
			r.y0, r.y1 = min(r.y0, b), max(r.y1, b)
		}
	}
	return r
}

func (p *PDFCanvas) drawFig2StressLabels() {
	p.placingLabels = true
	defer func() {
		p.placingLabels = false
		p.stressLabels = p.stressLabels[:0]
		p.reservedLabels = p.reservedLabels[:0]
		p.SetFig2FontSize(6)
	}()
	sort.SliceStable(p.stressLabels, func(i, j int) bool {
		a, b := p.stressLabels[i], p.stressLabels[j]
		if a.y != b.y {
			return a.y < b.y
		}
		if a.x != b.x {
			return a.x < b.x
		}
		if a.angle != b.angle {
			return a.angle < b.angle
		}
		return a.text < b.text
	})
	width, height := p.width*p.coordScale, p.height*p.coordScale
	for _, l := range p.stressLabels {
		if l.size < 1 {
			l.size = 6
		}
		p.SetFig2FontSize(l.size)
		tw, err := p.canvas.MeasureTextWidth(l.text)
		if err != nil {
			p.layoutErr = err
			return
		}
		size := float64(l.size)
		x, y := l.x*p.coordScale, l.y*p.coordScale
		dx, dy := fig2TextOrigin(l.align, tw, size)
		base := fig2LabelBounds(x, y, l.angle, dx, dy, tw, size)
		sx, sy, placed := fig2PlaceLabel(base, width, height, p.reservedLabels)
		if !placed {
			p.layoutErr = fmt.Errorf("could not place stress label %s in the figure viewport", l.text)
			return
		}
		base.x0 += sx
		base.x1 += sx
		base.y0 += sy
		base.y1 += sy
		p.reservedLabels = append(p.reservedLabels, base)
		if math.Hypot(sx, sy) > size*1.4 {
			p.Foreground(GRAY)
			p.SetFig2Stroke(.05, nil)
			// Terminate at the edge of the displaced label box.
			ex, ey := min(max(x, base.x0), base.x1), min(max(y, base.y0), base.y1)
			p.Line(l.x, l.y, ex/p.coordScale, ey/p.coordScale)
		}
		p.Foreground(l.color)
		p.TextOrientation(l.angle)
		p.TextAlignment(l.align)
		p.Text(l.x+sx/p.coordScale, l.y+sy/p.coordScale, l.text)
	}
	p.TextOrientation(0)
}

func fig2PlaceLabel(base fig2LabelRect, w, h float64, taken []fig2LabelRect) (float64, float64, bool) {
	fits := func(dx, dy float64) bool {
		box := fig2LabelRect{base.x0 + dx, base.y0 + dy, base.x1 + dx, base.y1 + dy}
		if box.x0 < 2 || box.y0 < 2 || box.x1 > w-2 || box.y1 > h-2 {
			return false
		}
		for _, a := range taken {
			if a.overlaps(box) {
				return false
			}
		}
		return true
	}
	if fits(0, 0) {
		return 0, 0, true
	}
	// Search in expanding rings to keep values near their member. The
	// viewport edges and original annotations constrain the available space.
	for r := 3.0; r <= max(w, h); r += 3 {
		for d := 0.0; d <= r; d += 3 {
			for _, p := range [][2]float64{{r, d}, {-r, d}, {r, -d}, {-r, -d}, {d, r}, {d, -r}, {-d, r}, {-d, -r}} {
				if fits(p[0], p[1]) {
					return p[0], p[1], true
				}
			}
		}
	}
	return 0, 0, false
}
