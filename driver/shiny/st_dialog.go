package stshiny

import (
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
	"image"
	"log"
)

const (
	width   = 1024
	linenum = 5
	linesep = 15
	margin  = 10
)

type Dialog struct {
	parent   *Window
	window   screen.Window
	buffer   screen.Buffer
	position int
	text     []string
}

func NewDialog(w *Window) *Dialog {
	return &Dialog{
		parent: w,
		window: nil,
		buffer: nil,
		text:   nil,
	}
}

func (d *Dialog) Start() chan bool {
	quit := make(chan bool)
	go func() {
		w, err := d.parent.screen.NewWindow(&screen.NewWindowOptions{
			Width:  width,
			Height: margin + linenum * linesep,
		})
		if err != nil {
			log.Fatal(err)
		}
		d.window = w
		defer d.window.Release()
		d.Redraw()
		var sz size.Event
		for {
			e := d.window.NextEvent()
			switch e := e.(type) {
			case lifecycle.Event:
				if e.To == lifecycle.StageDead {
					return
				}
			case key.Event:
				switch e.Direction {
				case key.DirPress:
					switch e.Code {
					case key.CodeDownArrow:
						d.position++
						if d.position > len(d.text) {
							d.position = len(d.text)
						}
					case key.CodeUpArrow:
						if d.position > linenum {
							d.position--
							if d.position < 0 {
								d.position = 0
							}
						}
					}
				}
				d.Redraw()
			case mouse.Event:
			case paint.Event:
				d.window.Fill(sz.Bounds(), blue0, screen.Src)
				d.window.Upload(image.Point{}, d.buffer, d.buffer.Bounds())
				d.window.Publish()
			case size.Event:
				sz = e
			case error:
				log.Print(e)
			}
		}
	}()
	return quit
}

func (d *Dialog) Redraw() {
	if d.buffer != nil {
		d.buffer.Release()
	}
	winSize := image.Point{width, margin + linenum * linesep}
	b, err := d.parent.screen.NewBuffer(winSize)
	if err != nil {
		log.Fatal(err)
	}
	d.buffer = b
	x := float64(margin)
	y := float64(linesep)
	start := d.position - linenum
	end := d.position
	if start < 0 {
		start = 0
	}
	if end > len(d.text) {
		end = len(d.text)
	}
	for _, t := range d.text[start:end] {
		d.Text(x, y, t)
		y += float64(linesep)
	}
	d.window.Upload(image.Point{}, d.buffer, d.buffer.Bounds())
	d.window.Publish()
}

func (d *Dialog) Text(x, y float64, str string) {
	if str == "" {
		return
	}
	dr := &font.Drawer{
		Dst:  d.buffer.RGBA(),
		Src:  image.NewUniform(d.parent.fontColor),
		Face: d.parent.fontFace,
		Dot:  fixed.Point26_6{fixed.Int26_6(x * 64), fixed.Int26_6(y * 64)},
	}
	dr.DrawString(str)
}

func (d *Dialog) ClearString() {
	d.text = nil
}

func (d *Dialog) TypeString(str string) {
	if d.text == nil {
		d.text = []string{str}
	} else {
		d.text = append(d.text, str)
	}
	d.position = len(d.text)
	d.Redraw()
}
