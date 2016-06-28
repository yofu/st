package stshiny

import (
	"image/color"
	"io/ioutil"

	"github.com/golang/freetype/truetype"
	"github.com/yofu/st/stlib"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

type Font struct {
	face   font.Face
	name   string
	height fixed.Int26_6
	color  color.RGBA
}

var basicFont = &Font{
	face:   basicfont.Face7x13,
	height: fixed.I(13),
	color:  color.RGBA{0xff, 0xff, 0xff, 0xff},
}

func LoadFontFace(path string, point float64) (*Font, error) {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ttf, err := truetype.Parse(f)
	if err != nil {
		return nil, err
	}
	return &Font{
		face: truetype.NewFace(ttf, &truetype.Options{
			Size:    point,
			Hinting: font.HintingFull,
		}),
		name: path,
		height: fixed.I(int(point*3) >> 2), // * 72/96
		color: color.RGBA{0xff, 0xff, 0xff, 0xff},
	}, nil
}

func (f *Font) Face() string {
	return f.name
}

func (f *Font) SetFace(face string) {
	f2, err := LoadFontFace(face, float64(f.height)*4.0/3.0)
	if err != nil {
		return
	}
	f.face = f2.face
	f.name = f2.name
	f.height = f2.height
	f.color = f2.color
}

func (f *Font) Size() int {
	return int(f.height)
}

func (f *Font) SetSize(s int) {
	f2, err := LoadFontFace(f.name, float64(s))
	if err != nil {
		return
	}
	f.face = f2.face
	f.name = f2.name
	f.height = f2.height
	f.color = f2.color
}

func (f *Font) Color() int {
	return int(f.color.R) * 65536 + int(f.color.G) * 256 + int(f.color.B)
}

func (f *Font) SetColor(c int) {
	col := st.IntColorList(c)
	f.color = color.RGBA{uint8(col[0]), uint8(col[1]), uint8(col[2]), 0xff}
}
