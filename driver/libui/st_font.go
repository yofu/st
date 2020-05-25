package stlibui

import "github.com/andlabs/ui"

type Font struct {
	uifont *ui.FontDescriptor
	name   string
	size   int
	color  int
}

func NewFont() *Font {
	return &Font{
		uifont: &ui.FontDescriptor{
			Family:  "IPA明朝",
			Size:    9,
			Weight:  400,
			Italic:  ui.TextItalicNormal,
			Stretch: ui.TextStretchCondensed,
		},
		name:  "IPA明朝",
		size:  9,
		color: 0x000000,
	}
}

func (f *Font) setUiFont() {
	f.uifont = &ui.FontDescriptor{
		Family:  ui.TextFamily(f.name),
		Size:    ui.TextSize(f.size),
		Weight:  400,
		Italic:  ui.TextItalicNormal,
		Stretch: ui.TextStretchCondensed,
	}
}

func (f *Font) Face() string {
	return f.name
}

func (f *Font) SetFace(n string) {
	f.name = n
	f.setUiFont()
}

func (f *Font) Size() int {
	return f.size
}

func (f *Font) SetSize(s int) {
	f.size = s
	f.setUiFont()
}

func (f *Font) Color() int {
	return f.color
}

func (f *Font) SetColor(col int) {
	f.color = col
}
