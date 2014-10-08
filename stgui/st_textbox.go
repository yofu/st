package stgui

import (
	"github.com/visualfc/go-iup/cd"
)

const (
	defaultfontface = "IPA明朝"
	defaultfontsize = 12
)

var (
	defaultfontcolor = cd.CD_WHITE
)

type TextBox struct {
	Value    []string
	Position []float64
	Angle    float64
	Font     *Font
	Hide     bool
}

type Font struct {
	Face  string
	Size  int
	Color int
}

func NewTextBox() *TextBox {
	rtn := new(TextBox)
	rtn.Value = make([]string, 0)
	rtn.Position = []float64{0.0, 0.0}
	rtn.Font = NewFont()
	rtn.Hide = true
	return rtn
}

func NewFont() *Font {
	rtn := new(Font)
	rtn.Face = defaultfontface
	rtn.Size = defaultfontsize
	rtn.Color = defaultfontcolor
	return rtn
}
