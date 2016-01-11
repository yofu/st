package st

type TextBox interface {
	Hider
	Text() []string
	Clear()
	SetText([]string)
	AddText(...string)
	SetPosition(float64, float64)
	Linage() int
	Bbox() (float64, float64, float64, float64)
	Contains(float64, float64) bool
	Width() float64
	Height() float64
	ScrollDown(int)
	ScrollUp(int)
	ScrollToTop()
}
