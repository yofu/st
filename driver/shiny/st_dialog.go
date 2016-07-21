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
	"strings"
)

var (
	width   = 300
	linenum = 30
	linesep = 15
	margin  = 10
)

type Dialog struct {
	parent   *Window
	window   screen.Window
	buffer   screen.Buffer
	position int
	index    int
	maxindex int
	cr       bool
	nl       bool
	text     []string
}

func NewDialog(w *Window) *Dialog {
	return &Dialog{
		parent:   w,
		window:   nil,
		buffer:   nil,
		position: linenum,
		index:    0,
		maxindex: 0,
		cr:       false,
		nl:       false,
		text:     nil,
	}
}

func (d *Dialog) Start() chan bool {
	quit := make(chan bool)
	go func() {
		w, err := d.parent.screen.NewWindow(&screen.NewWindowOptions{
			Width:  width,
			Height: margin + linenum*linesep,
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
				case key.DirPress, key.DirNone:
					switch e.Code {
					case key.CodeRightArrow:
						d.index++
						if d.index > d.maxindex {
							d.index = d.maxindex
						}
					case key.CodeLeftArrow:
						d.index--
						if d.index < 0 {
							d.index = 0
						}
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
				width = sz.WidthPx
				linenum = (sz.HeightPx - margin) / linesep
				d.Redraw()
				d.window.Publish()
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
	winSize := image.Point{width, margin + linenum*linesep}
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
		Src:  image.NewUniform(d.parent.font.color),
		Face: d.parent.font.face,
		Dot:  fixed.Point26_6{fixed.I(int(x)), fixed.I(int(y))},
	}
	if len(str) > d.index {
		dr.DrawString(str[d.index:])
	}
}

func (d *Dialog) ClearString() {
	d.text = nil
}

func (d *Dialog) TypeString(str string) {
	cr := false
	nl := false
	if strings.HasSuffix(str, "\r") {
		str = strings.TrimSuffix(str, "\r")
		cr = true
	}
	lis := strings.Split(str, "\n")
	if lis[len(lis)-1] == "" {
		nl = true
	}
	if len(lis) == 0 {
		return
	}
	if d.text == nil {
		d.text = lis
	} else {
		if d.cr || d.nl {
			if lis[0] != "" {
				d.text[len(d.text)-1] = lis[0]
			}
			if len(lis) > 1 {
				d.text = append(d.text, lis[1:]...)
			}
		} else {
			d.text = append(d.text, lis...)
		}
	}
	for _, s := range lis {
		if len(s) > d.maxindex {
			d.maxindex = len(s)
		}
	}
	d.position = len(d.text)
	d.cr = cr
	d.nl = nl
	d.Redraw()
}

func (d *Dialog) Write(p []byte) (int, error) {
	d.TypeString(string(p))
	return len(p), nil
}
