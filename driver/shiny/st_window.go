package stshiny

import (
	"fmt"
	"image"
	"image/color"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
	"github.com/yofu/st/stlib"
	"log"
	"os"
)

var (
	blue0 = color.RGBA{0x00, 0x00, 0x1f, 0xff}
	red   = color.RGBA{0x7f, 0x00, 0x00, 0x7f}

	startX = 0
	startY = 0
	endX = 0
	endY = 0

	pressed = 0
)

const (
	ButtonLeft = 1 << iota
	ButtonMiddle
)

var (
	CanvasFitScale = 0.9
)

type Window struct {
	frame *st.Frame
	screen screen.Screen
	window screen.Window
	buffer screen.Buffer
	currentPen color.RGBA
	currentBrush color.RGBA
	cline string
}

func NewWindow(s screen.Screen) *Window {
	return &Window{
		frame: st.NewFrame(),
		screen: s,
		window: nil,
		buffer: nil,
		currentPen: color.RGBA{0xff, 0xff, 0xff, 0xff},
		currentBrush: color.RGBA{0xff, 0xff, 0xff, 0x77},
		cline: "",
	}
}

func (stw *Window) OpenFile(fn string) error {
	frame := st.NewFrame()
	err := frame.ReadInp(fn, []float64{0.0, 0.0, 0.0}, 0.0, false)
	if err != nil {
		return err
	}
	stw.frame = frame
	return nil
}

func keymap(ev key.Event) key.Event {
	switch ev.Code {
	default:
		return ev
	case key.CodeSemicolon:
		r := ev.Rune
		if ev.Modifiers&key.ModShift != 0 {
			r = ';'
		} else {
			r = ':'
		}
		return key.Event{
			Rune: r,
			Code: ev.Code,
			Modifiers: ev.Modifiers^key.ModShift,
			Direction: ev.Direction,
		}
	}
}

func (stw *Window) Start() {
	w, err := stw.screen.NewWindow(nil)
	if err != nil {
		log.Fatal(err)
	}
	stw.window = w
	defer stw.window.Release()
	stw.OpenFile(fmt.Sprintf("%s/Downloads/yokofolly13.inp", os.Getenv("HOME")))
	stw.Redraw()
	var sz size.Event
	for {
		e := stw.window.NextEvent()
		switch e := e.(type) {
		case lifecycle.Event:
			if e.To == lifecycle.StageDead {
				return
			}
		case key.Event:
			if e.Direction == key.DirRelease {
				fmt.Println(e.Code)
				kc := keymap(e)
				switch kc.Code {
				default:
					stw.cline = fmt.Sprintf("%s%s", stw.cline, string(kc.Rune))
				case key.CodeDeleteBackspace:
					if len(stw.cline) >= 1 {
						stw.cline = stw.cline[:len(stw.cline)-1]
					}
				case key.CodeLeftShift:
				case key.CodeLeftAlt:
				case key.CodeReturnEnter:
				case key.CodeEscape:
					return
				}
				fmt.Printf("%s\r", stw.cline)
			}
		case mouse.Event:
			switch e.Direction {
			case mouse.DirPress:
				startX = int(e.X)
				startY = int(e.Y)
				switch e.Button {
				case mouse.ButtonLeft:
					pressed |= ButtonLeft
				case mouse.ButtonMiddle:
					pressed |= ButtonMiddle
				}
			case mouse.DirNone:
				endX = int(e.X)
				endY = int(e.Y)
				if pressed&ButtonLeft != 0 {
					stw.window.Upload(image.Point{}, stw.buffer, stw.buffer.Bounds())
					stw.window.Fill(image.Rect(startX, startY, endX, endY), red, screen.Over)
					stw.window.Publish()
				} else if pressed&ButtonMiddle != 0 {
					stw.frame.View.Angle[0] += float64(int(e.Y)-startY) * 0.01
					stw.frame.View.Angle[1] -= float64(int(e.X)-startX) * 0.01
					stw.Redraw()
					stw.window.Upload(image.Point{}, stw.buffer, stw.buffer.Bounds())
					stw.window.Publish()
				}
			case mouse.DirRelease:
				endX = int(e.X)
				endY = int(e.Y)
				stw.Redraw()
				stw.window.Upload(image.Point{}, stw.buffer, stw.buffer.Bounds())
				stw.window.Publish()
				switch e.Button {
				case mouse.ButtonLeft:
					pressed &= ^ButtonLeft
				case mouse.ButtonMiddle:
					pressed &= ^ButtonMiddle
				}
			}
		case paint.Event:
			stw.window.Fill(sz.Bounds(), blue0, screen.Src)
			stw.window.Upload(image.Point{}, stw.buffer, stw.buffer.Bounds())
			stw.window.Publish()
		case size.Event:
			sz = e
		case error:
			log.Print(e)
		}
	}
}

func (stw *Window) Frame() *st.Frame {
	return stw.frame
}

func (stw *Window) Redraw() {
	if stw.frame == nil {
		return
	}
	if stw.buffer != nil {
		stw.buffer.Release()
	}
	winSize := image.Point{1024, 1024}
	b, err := stw.screen.NewBuffer(winSize)
	if err != nil {
		log.Fatal(err)
	}
	stw.buffer = b
	stw.frame.View.Center[0] = 512
	stw.frame.View.Center[1] = 512
	st.DrawFrame(stw, stw.frame, st.ECOLOR_SECT, true)
}

func (stw *Window) SelectedElems() []*st.Elem {
	return nil
}

func (stw *Window) SelectedNodes() []*st.Node {
	return nil
}

func (stw *Window) SelectElem([]*st.Elem) {
}

func (stw *Window) SelectNode([]*st.Node) {
}

func (stw *Window) ElemSelected() bool {
	return false
}

func (stw *Window) NodeSelected() bool {
	return false
}

func (stw *Window) Deselect() {
}

func (stw *Window) LastExCommand() string {
	return ""
}

func (stw *Window) SetLastExCommand(string) {
}

func (stw *Window) History(string) {
}

func (stw *Window) ErrorMessage(error, int) {
}

func (stw *Window) CompleteFileName(string) string {
	return ""
}

func (stw *Window) Cwd() string {
	return ""
}

func (stw *Window) HomeDir() string {
	return ""
}

func (stw *Window) Print() {
}

func (stw *Window) Changed(bool) {
}

func (stw *Window) IsChanged() bool {
	return false
}

func (stw *Window) Yn(string, string) bool {
	return false
}

func (stw *Window) Yna(string, string, string) int {
	return 0
}

func (stw *Window) SaveAS() {
	stw.SaveFile("hogtxt.inp")
}

func (stw *Window) SaveFile(fn string) error {
	return st.SaveFile(stw, fn)
}

func (stw *Window) GetCanvasSize() (int, int) {
	return 1024, 1024
}

func (stw *Window) SaveFileSelected(string) error {
	return nil
}

func (stw *Window) SearchFile(string) (string, error) {
	return "", nil
}

func (stw *Window) Reload() {
}

func (stw *Window) Close(bool) {
}

func (stw *Window) Checkout(string) error {
	return nil
}

func (stw *Window) AddTag(string, bool) error {
	return nil
}

func (stw *Window) Copylsts(string) {
}

func (stw *Window) ReadFile(string) error {
	return nil
}

func (stw *Window) ReadAll() {
}

func (stw *Window) ReadPgp(string) error {
	return nil
}

func (stw *Window) ReadFig2(string) error {
	return nil
}

func (stw *Window) CheckFrame() {
}

func (stw *Window) SelectConfed() {
}

func (stw *Window) Rebase(string) {
}

func (stw *Window) ShowRecently() {
}

func (stw *Window) ShapeData(st.Shape) {
}

func (stw *Window) Snapshot() {
}

func (stw *Window) UseUndo(bool) {
}

func (stw *Window) EPS() float64 {
	return 1e-3
}

func (stw *Window) SetEPS(float64) {
}

func (stw *Window) CanvasFitScale() float64 {
	return CanvasFitScale
}

func (stw *Window) SetCanvasFitScale(val float64) {
	CanvasFitScale = val
}

func (stw *Window) CanvasAnimateSpeed() float64 {
	return 0.0
}

func (stw *Window) SetCanvasAnimateSpeed(float64) {
}

func (stw *Window) ToggleFixRotate() {
}

func (stw *Window) ToggleFixMove() {
}

func (stw *Window) ToggleAltSelectNode() {
}

func (stw *Window) AltSelectNode() bool {
	return false
}

func (stw *Window) SetShowPrintRange(bool) {
}

func (stw *Window) ToggleShowPrintRange() {
}

func (stw *Window) CurrentLap(string, int, int) {
}

func (stw *Window) SectionData(*st.Sect) {
}

func (stw *Window) TextBox(string) st.TextBox {
	return nil
}

func (stw *Window) AxisRange(int, float64, float64, bool) {
}

func (stw *Window) NextFloor() {
}

func (stw *Window) PrevFloor() {
}

func (stw *Window) SetAngle(float64, float64) {
}

func (stw *Window) SetPaperSize(uint) {
}

func (stw *Window) PaperSize() uint {
	return st.A4_TATE
}

func (stw *Window) SetPeriod(string) {
}

func (stw *Window) Pivot() bool {
	return false
}

func (stw *Window) DrawPivot([]*st.Node, chan int, chan int) {
}

func (stw *Window) SetColorMode(uint) {
}

func (stw *Window) SetConf([]bool) {
}
