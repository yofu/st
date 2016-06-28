package stcui

import (
	"github.com/mattn/go-runewidth"
	"github.com/yofu/st/stlib"
)

type TextBox struct {
	value    []string
	index    int
	position []int
	Angle    float64
	hide     bool
}

func NewTextBox() *TextBox {
	rtn := new(TextBox)
	rtn.value = make([]string, 0)
	rtn.position = []int{0, 0}
	rtn.hide = true
	return rtn
}

func (tb *TextBox) Hide() {
	tb.hide = true
}

func (tb *TextBox) Show() {
	tb.hide = false
}

func (tb *TextBox) IsHidden(*st.Show) bool {
	return tb.hide == true
}

func (tb *TextBox) SetPosition(x, y float64) {
	tb.position[0] = int(x)
	tb.position[1] = int(y)
}

func (tb *TextBox) Position() (float64, float64) {
	return float64(tb.position[0]), float64(tb.position[1])
}

func (tb *TextBox) Text() []string {
	return tb.value[tb.index:]
}

func (tb *TextBox) Clear() {
	tb.value = make([]string, 0)
	tb.index = 0
}

func (tb *TextBox) SetText(str []string) {
	tb.value = str
}

func (tb *TextBox) AddText(str ...string) {
	tb.value = append(tb.value, str...)
}

func (tb *TextBox) Linage() int {
	l := len(tb.value)
	for i := l - 1; i >= 0; i-- {
		if tb.value[i] != "" {
			break
		}
		l--
	}
	return l
}

func (tb *TextBox) Bbox() (float64, float64, float64, float64) {
	return float64(tb.position[0]), float64(tb.position[1]) - tb.Height(), float64(tb.position[0]) + tb.Width(), float64(tb.position[1])
}

func (tb *TextBox) Contains(x, y float64) bool {
	xmin, ymin, xmax, ymax := tb.Bbox()
	return xmin <= x && x <= xmax && ymin <= y && y <= ymax
}

func (tb *TextBox) Width() float64 {
	wmax := 0
	for _, s := range tb.value {
		w := runewidth.StringWidth(s)
		if w > wmax {
			wmax = w
		}
	}
	return float64(wmax)
}

func (tb *TextBox) Height() float64 {
	return float64(tb.Linage())
}

func (tb *TextBox) ScrollDown(n int) {
	if tb.index >= len(tb.value)-n {
		return
	}
	tb.index += n
}

func (tb *TextBox) ScrollUp(n int) {
	if tb.index < n {
		return
	}
	tb.index -= n
}

func (tb *TextBox) ScrollToTop() {
	tb.index = 0
}
