package stshiny

import (
	"image/color"
	"io/ioutil"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

type Font struct {
	face   font.Face
	height fixed.Int26_6
	color  color.RGBA
}

var basicFont = &Font{
	face:   basicfont.Face7x13,
	height: 13,
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
		height: fixed.Int26_6(int(point*3) >> 2), // * 72/96
		color: color.RGBA{0xff, 0xff, 0xff, 0xff},
	}, nil
}

func (f *Font) Size() int {
	return int(f.height)
}
