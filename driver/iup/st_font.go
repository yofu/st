package stgui

import (
	"github.com/visualfc/go-iup/cd"
)

var (
	defaultfontface  = fontface
	defaultfontsize  = 12
	defaultfontcolor = cd.CD_WHITE
)

type Font struct {
	face  string
	size  int
	color int
}

func NewFont() *Font {
	rtn := new(Font)
	rtn.face = defaultfontface
	rtn.size = defaultfontsize
	rtn.color = defaultfontcolor
	return rtn
}

func (f *Font) Face() string {
	return f.face
}

func (f *Font) SetFace(face string) {
	f.face = face
}

func (f *Font) Size() int {
	return f.size
}

func (f *Font) SetSize(s int) {
	f.size = s
}

func (f *Font) Color() int {
	return f.color
}

func (f *Font) SetColor(c int) {
	f.color = c
}
