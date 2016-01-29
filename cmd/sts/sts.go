package main

import (
	"image"
	"image/color"
	"log"
	"math"

	"golang.org/x/exp/shiny/driver"
	"golang.org/x/exp/shiny/screen"
	// "golang.org/x/image/math/f64"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
	"github.com/yofu/st/driver/shiny"
)

var (
	blue0 = color.RGBA{0x00, 0x00, 0x1f, 0xff}
	blue1 = color.RGBA{0x00, 0x00, 0x3f, 0xff}
	red   = color.RGBA{0x7f, 0x00, 0x00, 0x7f}

	cos30 = math.Cos(math.Pi / 6)
	sin30 = math.Sin(math.Pi / 6)

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

func main() {
	driver.Main(func(s screen.Screen) {
		w, err := s.NewWindow(nil)
		if err != nil {
			log.Fatal(err)
		}
		defer w.Release()

		winSize := image.Point{1024, 1024}
		b, err := s.NewBuffer(winSize)
		if err != nil {
			log.Fatal(err)
		}
		defer b.Release()

		stw := stshiny.NewWindow(b.RGBA())
		stw.OpenFile("/home/fukushima/Downloads/yokofolly13.inp")
		stw.Redraw()

		t, err := s.NewTexture(winSize)
		if err != nil {
			log.Fatal(err)
		}
		defer t.Release()
		t.Upload(image.Point{}, b, b.Bounds())

		var sz size.Event
		for {
			e := w.NextEvent()
			switch e := e.(type) {
			case lifecycle.Event:
				if e.To == lifecycle.StageDead {
					return
				}
			case key.Event:
				if e.Code == key.CodeEscape {
					return
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
						b.Release()
						b, err := s.NewBuffer(winSize)
						if err != nil {
							log.Fatal(err)
						}
						stw.SetBuffer(b.RGBA())
						stw.Redraw()
						w.Upload(image.Point{}, b, b.Bounds())
						w.Fill(image.Rect(startX, startY, endX, endY), red, screen.Over)
					} else if pressed&ButtonMiddle != 0 {
						stw.Frame.View.Angle[0] += float64(int(e.Y)-startY) * 0.01
						stw.Frame.View.Angle[1] -= float64(int(e.X)-startX) * 0.01
						b.Release()
						b, err := s.NewBuffer(winSize)
						if err != nil {
							log.Fatal(err)
						}
						stw.SetBuffer(b.RGBA())
						stw.Redraw()
						w.Upload(image.Point{}, b, b.Bounds())
					}
				case mouse.DirRelease:
					endX = int(e.X)
					endY = int(e.Y)
					b.Release()
					b, err := s.NewBuffer(winSize)
					if err != nil {
						log.Fatal(err)
					}
					stw.SetBuffer(b.RGBA())
					stw.Redraw()
					w.Upload(image.Point{}, b, b.Bounds())
					switch e.Button {
					case mouse.ButtonLeft:
						pressed &= ^ButtonLeft
					case mouse.ButtonMiddle:
						pressed &= ^ButtonMiddle
						// stw.Frame.View.Angle[0] += float64(int(e.Y)-startY) * 0.01
						// stw.Frame.View.Angle[1] -= float64(int(e.X)-startX) * 0.01
						// b.Release()
						// b, err := s.NewBuffer(winSize)
						// if err != nil {
						// 	log.Fatal(err)
						// }
						// stw.SetBuffer(b.RGBA())
						// stw.Redraw()
						// w.Upload(image.Point{}, b, b.Bounds())
					}
				}
			case paint.Event:
				w.Fill(sz.Bounds(), blue0, screen.Src)
				w.Upload(image.Point{}, b, b.Bounds())
				w.Publish()
			case size.Event:
				sz = e
			case error:
				log.Print(e)
			}
		}
	})
}
