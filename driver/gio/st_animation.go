package stgio

import (
	"math"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	st "github.com/yofu/st/stlib"
)

const viewFrameInterval = time.Second / 60

type viewAnimation struct {
	owner    *st.Frame
	view     *st.View
	from, to *st.View
	start    time.Time
	started  bool
	duration time.Duration
}

// Animate follows IUP's linear center/focus/angle interpolation and geometric
// zoom interpolation, but advances on Gio frames instead of a blocking loop.
// Callers and frame updates hold modelMu.
func (w *Window) Animate(target *st.View) {
	if target == nil || w.frame == nil || w.frame.View == nil {
		return
	}
	from, to := w.frame.View.Copy(), target.Copy()
	speed := w.CanvasAnimateSpeed()
	w.cancelViewAnimation()
	if speed <= 0 || speed >= 1 || math.IsNaN(speed) || math.IsInf(speed, 0) || from.Perspective != to.Perspective {
		w.frame.View = to
		w.Redraw()
		return
	}
	// At 60 frames/s the default 0.02 gives the same 50 interpolation steps
	// as IUP (about 0.83 s); cap pathological settings before duration conversion.
	seconds := math.Min(10, 1/(60*speed))
	w.animation = &viewAnimation{owner: w.frame, view: w.frame.View, from: from, to: to, duration: time.Duration(seconds * float64(time.Second))}
	// CenterView projects nodes while computing the target. Restore picking
	// coordinates for the view currently on screen before returning to input.
	w.frame.View.Set(w.CanvasDirection())
	for _, n := range w.frame.Nodes {
		w.frame.View.ProjectNode(n)
		if w.frame.Show.PlotState&st.PLOT_DEFORMED != 0 {
			w.frame.View.ProjectDeformation(n, w.frame.Show)
		}
	}
	w.sceneDirty.Store(true)
	w.viewAnimating.Store(true)
	w.QueueRedrawAll()
}

func (w *Window) cancelViewAnimation() { w.animation = nil; w.viewAnimating.Store(false) }

func (w *Window) advanceViewAnimation(now time.Time) {
	a := w.animation
	if a == nil {
		return
	}
	if w.frame != a.owner || w.frame.View != a.view {
		w.cancelViewAnimation()
		return
	}
	if !a.started {
		a.start = now
		a.started = true
		return
	}
	t := math.Max(0, math.Min(1, float64(now.Sub(a.start))/float64(a.duration)))
	if t >= 1 {
		w.frame.View = a.to.Copy()
		w.cancelViewAnimation()
		w.sceneDirty.Store(true)
		return
	}
	v := a.view
	lerp := func(x, y float64) float64 { return x + (y-x)*t }
	zoom := func(x, y float64) float64 {
		if x > 0 && y > 0 {
			return math.Exp(lerp(math.Log(x), math.Log(y)))
		}
		return lerp(x, y)
	}
	for i := 0; i < 3; i++ {
		v.Focus[i] = lerp(a.from.Focus[i], a.to.Focus[i])
	}
	for i := 0; i < 2; i++ {
		v.Center[i] = lerp(a.from.Center[i], a.to.Center[i])
		d := a.to.Angle[i] - a.from.Angle[i]
		if i == 1 {
			d = math.Mod(d, 360)
			if d > 180 {
				d -= 360
			}
			if d < -180 {
				d += 360
			}
		}
		v.Angle[i] = a.from.Angle[i] + d*t
	}
	v.Gfact = zoom(a.from.Gfact, a.to.Gfact)
	v.Dists[0] = lerp(a.from.Dists[0], a.to.Dists[0])
	v.Dists[1] = zoom(a.from.Dists[1], a.to.Dists[1])
	w.sceneDirty.Store(true)
}

func (w *Window) scheduleViewAnimation(gtx layout.Context) {
	if w.viewAnimating.Load() {
		gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(viewFrameInterval)})
	}
}
