package stshiny

import (
	"image"
	"image/color"
	"github.com/yofu/st/stlib"
)

type Window struct {
	Frame *st.Frame
	currentCanvas *image.RGBA
	currentPen color.RGBA
}

func NewWindow(r *image.RGBA) *Window {
	return &Window{
		Frame: st.NewFrame(),
		currentCanvas: r,
		currentPen: color.RGBA{0xff, 0xff, 0xff, 0xff},
	}
}

func (stw *Window) OpenFile(fn string) error {
	frame := st.NewFrame()
	err := frame.ReadInp(fn, []float64{0.0, 0.0, 0.0}, 0.0, false)
	if err != nil {
		return err
	}
	stw.Frame = frame
	return nil
}

func (stw *Window) Redraw() {
	if stw.Frame == nil {
		return
	}
	stw.Frame.View.Center[0] = 512
	stw.Frame.View.Center[1] = 512
	st.DrawFrame(stw, stw.Frame, st.ECOLOR_SECT, true)
}
