package stshiny

import (
	"image"
	"image/color"
	"strings"

	"github.com/yofu/st/stlib"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
	"golang.org/x/exp/shiny/screen"
)

type Glass struct {
	position  image.Point
	size      image.Point
	text      []string
	margin    int
	linesep   int
	parent    *Window
	buffer    screen.Buffer
	texture   screen.Texture
	show      bool
	frame     bool
	cr        bool
	nl        bool
	minimized bool
}

func NewGlass(p *Window) *Glass {
	return &Glass{
		position: image.Point{0.0, 0.0},
		size:     image.Point{0.0, 0.0},
		text:     nil,
		margin:   5,
		linesep:  15,
		parent:   p,
		buffer:   nil,
		texture:  nil,
		show:     true,
		frame:    true,
	}
}

func (g *Glass) SetPosition(x, y int) {
	g.position = image.Point{x, y}
}

func (g *Glass) SetSize(x, y int) {
	g.size = image.Point{x, y + 10}
}

func (g *Glass) Show() {
	g.show = true
}

func (g *Glass) Hide() {
	g.show = false
}

func (g *Glass) SetText(str []string) {
	g.text = str
}

func (g *Glass) AddText(str []string) {
	g.text = append(g.text, str...)
}

func (g *Glass) Minimized() bool {
	return g.minimized
}

func (g *Glass) Minimize() {
	g.minimized = true
}

func (g *Glass) Maximize() {
	g.minimized = false
}

func (g *Glass) Redraw() error {
	if !g.show {
		return nil
	}
	var size image.Point
	var text []string
	if g.minimized {
		size = image.Point{g.size.X, g.parent.font.height.Ceil() + 2*g.margin}
		text = []string{g.text[len(g.text)-1]}
	} else {
		size = g.size
		text = g.text
	}
	if g.buffer != nil {
		g.buffer.Release()
	}
	b, err := g.parent.screen.NewBuffer(size)
	if err != nil {
		return err
	}
	g.buffer = b
	x := 0
	y := size.Y - 2*g.margin - (len(text) - 1)*g.linesep
	g.parent.Foreground(st.GREEN_A100)
	for _, t := range text {
		g.Text(x, y, t)
		y += g.linesep
	}
	g.parent.Foreground(st.WHITE)
	t, err := g.parent.screen.NewTexture(size)
	if err != nil {
		return err
	}
	if g.texture != nil {
		g.texture.Release()
	}
	g.texture = t
	t.Upload(image.Point{}, g.buffer, g.buffer.Bounds())
	g.Upload()
	return nil
}

func (g *Glass) Upload() {
	if !g.show || g.texture == nil {
		return
	}
	var size image.Point
	if g.minimized {
		size = image.Point{g.size.X, g.parent.font.height.Ceil() + 2*g.margin}
	} else {
		size = g.size
	}
	if g.frame {
		g.parent.window.Fill(image.Rect(g.position.X-g.margin-1, g.position.Y-size.Y-g.margin-1, g.position.X+size.X+g.margin+1, g.position.Y+1), color.RGBA{0xa2, 0xf7, 0x8d, 0xff}, screen.Over)
	}
	g.parent.window.Fill(image.Rect(g.position.X-g.margin, g.position.Y-size.Y-g.margin, g.position.X+size.X+g.margin, g.position.Y), color.RGBA{0x00, 0x00, 0x00, 0xff}, screen.Over)
	g.parent.window.Copy(image.Point{g.position.X, g.position.Y -size.Y}, g.texture, g.texture.Bounds(), screen.Over, nil)
	g.parent.window.Publish()
}

func (g *Glass) Text(x, y int, str string) {
	if str == "" {
		return
	}
	d := &font.Drawer{
		Dst:  g.buffer.RGBA(),
		Src:  image.NewUniform(g.parent.font.color),
		Face: g.parent.font.face,
		Dot:  fixed.Point26_6{fixed.I(x), fixed.I(y)},
	}
	d.DrawString(str)
}

func (g *Glass) TypeString(str string) {
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
	if g.text == nil {
		g.text = lis
	} else {
		if g.cr || g.nl {
			if lis[0] != "" {
				g.text[len(g.text)-1] = lis[0]
			}
			if len(lis) > 1 {
				g.text = append(g.text, lis[1:]...)
			}
		} else {
			g.text = append(g.text, lis...)
		}
	}
	g.cr = cr
	g.nl = nl
	g.Redraw()
}

func (g *Glass) Write(p []byte) (int, error) {
	g.TypeString(string(p))
	return len(p), nil
}
