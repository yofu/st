package stshiny

import (
	"fmt"
	"image"
	"image/color"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
	"github.com/yofu/st/stlib"
	"log"
	"os"
)

var (
	blue0 = color.RGBA{0x00, 0x00, 0x1f, 0xff}
	red   = color.RGBA{0x7f, 0x00, 0x00, 0x7f}

	startX = 0
	startY = 0
	endX = 0
	endY = 0

	pressed = 0
)

const (
	ButtonLeft = 1 << iota
	ButtonMiddle
)

type Window struct {
	Frame *st.Frame
	screen screen.Screen
	window screen.Window
	buffer screen.Buffer
	currentPen color.RGBA
	currentBrush color.RGBA
	cline string
}

func NewWindow(s screen.Screen) *Window {
	return &Window{
		Frame: st.NewFrame(),
		screen: s,
		window: nil,
		buffer: nil,
		currentPen: color.RGBA{0xff, 0xff, 0xff, 0xff},
		currentBrush: color.RGBA{0xff, 0xff, 0xff, 0x77},
		cline: "",
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

func keymap(ev key.Event) key.Event {
	switch ev.Code {
	default:
		return ev
	case key.CodeSemicolon:
		r := ev.Rune
		if ev.Modifiers&key.ModShift != 0 {
			r = ';'
		} else {
			r = ':'
		}
		return key.Event{
			Rune: r,
			Code: ev.Code,
			Modifiers: ev.Modifiers^key.ModShift,
			Direction: ev.Direction,
		}
	}
}

func (stw *Window) Start() {
	w, err := stw.screen.NewWindow(nil)
	if err != nil {
		log.Fatal(err)
	}
	stw.window = w
	defer stw.window.Release()
	stw.OpenFile(fmt.Sprintf("%s/Downloads/yokofolly13.inp", os.Getenv("HOME")))
	stw.Redraw()
	var sz size.Event
	for {
		e := stw.window.NextEvent()
		switch e := e.(type) {
		case lifecycle.Event:
			if e.To == lifecycle.StageDead {
				return
			}
		case key.Event:
			if e.Direction == key.DirRelease {
				fmt.Println(e.Code)
				kc := keymap(e)
				switch kc.Code {
				default:
					stw.cline = fmt.Sprintf("%s%s", stw.cline, string(kc.Rune))
				case key.CodeDeleteBackspace:
					if len(stw.cline) >= 1 {
						stw.cline = stw.cline[:len(stw.cline)-1]
					}
				case key.CodeLeftShift:
				case key.CodeLeftAlt:
				case key.CodeReturnEnter:
				case key.CodeEscape:
					return
				}
				fmt.Printf("%s\r", stw.cline)
			}
		case mouse.Event:
			switch e.Direction {
			case mouse.DirPress:
				startX = int(e.X)
				startY = int(e.Y)
				switch e.Button {
				case mouse.ButtonLeft:
					pressed |= ButtonLeft
				case mouse.ButtonMiddle:
					pressed |= ButtonMiddle
				}
			case mouse.DirNone:
				endX = int(e.X)
				endY = int(e.Y)
				if pressed&ButtonLeft != 0 {
					stw.window.Upload(image.Point{}, stw.buffer, stw.buffer.Bounds())
					stw.window.Fill(image.Rect(startX, startY, endX, endY), red, screen.Over)
					stw.window.Publish()
				} else if pressed&ButtonMiddle != 0 {
					stw.Frame.View.Angle[0] += float64(int(e.Y)-startY) * 0.01
					stw.Frame.View.Angle[1] -= float64(int(e.X)-startX) * 0.01
					stw.Redraw()
					stw.window.Upload(image.Point{}, stw.buffer, stw.buffer.Bounds())
					stw.window.Publish()
				}
			case mouse.DirRelease:
				endX = int(e.X)
				endY = int(e.Y)
				stw.Redraw()
				stw.window.Upload(image.Point{}, stw.buffer, stw.buffer.Bounds())
				stw.window.Publish()
				switch e.Button {
				case mouse.ButtonLeft:
					pressed &= ^ButtonLeft
				case mouse.ButtonMiddle:
					pressed &= ^ButtonMiddle
				}
			}
		case paint.Event:
			stw.window.Fill(sz.Bounds(), blue0, screen.Src)
			stw.window.Upload(image.Point{}, stw.buffer, stw.buffer.Bounds())
			stw.window.Publish()
		case size.Event:
			sz = e
		case error:
			log.Print(e)
		}
	}
}

func (stw *Window) Redraw() {
	if stw.Frame == nil {
		return
	}
	if stw.buffer != nil {
		stw.buffer.Release()
	}
	winSize := image.Point{1024, 1024}
	b, err := stw.screen.NewBuffer(winSize)
	if err != nil {
		log.Fatal(err)
	}
	stw.buffer = b
	stw.Frame.View.Center[0] = 512
	stw.Frame.View.Center[1] = 512
	st.DrawFrame(stw, stw.Frame, st.ECOLOR_SECT, true)
}
