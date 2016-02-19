package stshiny

import (
	"fmt"
	"github.com/yofu/st/stlib"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
	"image"
	"image/color"
	"log"
	"os"
	"strings"
)

var (
	completepos int
	completes   []string
)

var (
	blue0 = color.RGBA{0x00, 0x00, 0x1f, 0xff}
	red   = color.RGBA{0x7f, 0x00, 0x00, 0x7f}

	startX = 0
	startY = 0
	endX   = 0
	endY   = 0

	pressed = 0
)

const (
	ButtonLeft = 1 << iota
	ButtonMiddle
)

type Window struct {
	*st.DrawOption
	*st.Directory
	*st.RecentFiles
	*st.UndoStack
	*st.TagFrame
	frame        *st.Frame
	screen       screen.Screen
	window       screen.Window
	buffer       screen.Buffer
	currentPen   color.RGBA
	currentBrush color.RGBA
	cline        string
	changed      bool
}

func NewWindow(s screen.Screen) *Window {
	return &Window{
		DrawOption:   st.NewDrawOption(),
		Directory:    st.NewDirectory("", ""),
		RecentFiles:  st.NewRecentFiles(3),
		UndoStack:    st.NewUndoStack(10),
		TagFrame:     st.NewTagFrame(),
		frame:        st.NewFrame(),
		screen:       s,
		window:       nil,
		buffer:       nil,
		currentPen:   color.RGBA{0xff, 0xff, 0xff, 0xff},
		currentBrush: color.RGBA{0xff, 0xff, 0xff, 0x77},
		cline:        "",
		changed:      false,
	}
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
			Rune:      r,
			Code:      ev.Code,
			Modifiers: ev.Modifiers ^ key.ModShift,
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
	stw.ReadRecent()
	st.ShowRecent(stw)
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
					stw.FeedCommand()
				case key.CodeEscape:
					stw.Close(true)
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
					stw.window.Publish()
				}
			case mouse.DirRelease:
				endX = int(e.X)
				endY = int(e.Y)
				stw.Redraw()
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

func (stw *Window) SetFrame(frame *st.Frame) {
	stw.frame = frame
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
	stw.window.Upload(image.Point{}, stw.buffer, stw.buffer.Bounds())
}

func (stw *Window) FeedCommand() {
	command := stw.cline
	if command != "" {
		stw.cline = ""
		stw.ExecCommand(command)
		stw.Redraw()
	}
}

func (stw *Window) ExecCommand(command string) {
	if stw.frame == nil {
		if strings.HasPrefix(command, ":") {
			err := st.ExMode(stw, command)
			if err != nil {
				st.ErrorMessage(stw, err, st.ERROR)
			}
		} else if strings.HasPrefix(command, "'") {
			// err := st.Fig2Mode(stw, stw.frame, command)
			// if err != nil {
			// 	stw.ErrorMessage(err, st.ERROR)
			// }
		}
		return
	}
	switch {
	default:
		stw.History(fmt.Sprintf("command doesn't exist: %s", command))
	case strings.HasPrefix(command, ":"):
		err := st.ExMode(stw, command)
		if err != nil {
			st.ErrorMessage(stw, err, st.ERROR)
		}
		// case strings.HasPrefix(command, "'"):
		// 	err := st.Fig2Mode(stw, stw.frame, command)
		// 	if err != nil {
		// 		stw.ErrorMessage(err, st.ERROR)
		// 	}
	}
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

func (stw *Window) History(str string) {
	fmt.Println(str)
}

func (stw *Window) CompleteFileName(str string) string {
	path := ""
	if stw.frame != nil {
		path = stw.frame.Path
	}
	completes = st.CompleteFileName(str, path, stw.Recent())
	completepos = 0
	return completes[0]
}

func (stw *Window) Print() {
}

func (stw *Window) Changed(c bool) {
	stw.changed = c
}

func (stw *Window) IsChanged() bool {
	return stw.changed
}

func (stw *Window) Yn(string, string) bool {
	return false
}

func (stw *Window) Yna(string, string, string) int {
	return 0
}

func (stw *Window) SaveAS() {
	st.SaveFile(stw, "hogtxt.inp")
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

func (stw *Window) Close(bang bool) {
	if !bang && stw.changed {
		stw.History("changes are not saved")
		return
	}
	err := stw.SaveRecent()
	if err != nil {
		fmt.Println(err)
	}
	os.Exit(0)
}

func (stw *Window) ReadPgp(string) error {
	return nil
}

func (stw *Window) CheckFrame() {
}

func (stw *Window) ShapeData(st.Shape) {
}

func (stw *Window) EPS() float64 {
	return 1e-3
}

func (stw *Window) SetEPS(float64) {
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
