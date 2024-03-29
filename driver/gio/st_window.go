package stgio

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"os"
	"sort"
	"strings"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/font/gofont"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"

	"gioui.org/op/clip"
	// "gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/yofu/abbrev"
	st "github.com/yofu/st/stlib"
)

var (
	closech        = make(chan int)
	redrawch       = make(chan int)
	start          f32.Point
	end            f32.Point
	press          pointer.Buttons
	altselectnode  = true
	drawall        = true
	tailnodes      []*st.Node
	showprintrange = false
	PLATE_OPACITY  = uint8(0x55)
	defaultMargin  = unit.Dp(10)
)

type Window struct {
	*st.DrawOption
	*st.Directory
	*st.RecentFiles
	*st.UndoStack
	*st.TagFrame
	*st.Selection
	*st.CommandBuffer
	*st.CommandLine
	*st.Alias
	frame   *st.Frame
	Theme   *material.Theme
	window  *app.Window
	context layout.Context
	// history         *Dialog
	currentPen      color.NRGBA
	currentBrush    color.NRGBA
	papersize       uint
	changed         bool
	lastexcommand   string
	lastfig2command string
	lastcommand     func(st.Commander) chan bool
	textBox         map[string]*st.TextBox
	textAlignment   int
	// glasses         map[string]*Glass
}

func NewWindow(w *app.Window) *Window {
	return &Window{
		DrawOption:    st.NewDrawOption(),
		Directory:     st.NewDirectory("", ""),
		RecentFiles:   st.NewRecentFiles(3),
		UndoStack:     st.NewUndoStack(10),
		TagFrame:      st.NewTagFrame(),
		Selection:     st.NewSelection(),
		CommandBuffer: st.NewCommandBuffer(),
		CommandLine:   st.NewCommandLine(),
		Alias:         st.NewAlias(),
		frame:         st.NewFrame(),
		Theme:         material.NewTheme(gofont.Collection()),
		window:        w,
		// history:         nil,
		// currentStrokeParam: &ui.DrawStrokeParams{
		// 	Cap:        ui.DrawLineCapFlat,
		// 	Join:       ui.DrawLineJoinMiter,
		// 	Thickness:  LINE_THICKNESS,
		// 	MiterLimit: ui.DrawDefaultMiterLimit,
		// },
		currentPen:      color.NRGBA{R: 0, G: 0, B: 0, A: 0xff},
		currentBrush:    color.NRGBA{R: 0, G: 0, B: 0, A: 0xff},
		papersize:       st.A4_TATE,
		changed:         false,
		lastexcommand:   "",
		lastfig2command: "",
		lastcommand:     nil,
		textBox:         make(map[string]*st.TextBox),
		// glasses:         make(map[string]*Glass),
	}
}

func (stw *Window) QueueRedrawAll() {
	stw.window.Invalidate()
	// redrawch <- 1
}

func (stw *Window) Run(w *app.Window) error {
	st.OpenFile(stw, "C:\\Users\\yofu8\\st\\yamawa\\yamawa02\\yamawa02.inp", true)
	var ops op.Ops
	for {
		select {
		case <-redrawch:
			w.Invalidate()
		case <-closech:
			return nil
		case e := <-w.Events():
			// detect the type of the event.
			switch e := e.(type) {
			default:
			// this is sent when the application should re-render.
			case system.FrameEvent:
				// gtx is used to pass around rendering and event information.
				gtx := layout.NewContext(&ops, e)
				stw.context = gtx

				stw.Layout(gtx)
				// render and handle the operations from the UI.
				e.Frame(gtx.Ops)

			// handle a global key press.
			case key.Event:
				switch e.Name {
				// when we click escape, let's close the window.
				case key.NameEscape:
					return nil
				}

			case pointer.Event:
				switch e.Type {
				case pointer.Press:
					start = e.Position
					press = e.Buttons
					switch press {
					case pointer.ButtonTertiary:
						drawall = false
					}
				case pointer.Release:
					end = e.Position
					drawall = true
					press = 0
					w.Invalidate()
				case pointer.Move:
					end = e.Position
					dx := end.X - start.X
					dy := end.Y - start.Y
					switch press {
					case pointer.ButtonTertiary:
						fmt.Println(e)
						if e.Modifiers&key.ModShift != 0 {
							stw.frame.View.Center[0] += float64(dx) * stw.CanvasMoveSpeedX()
							stw.frame.View.Center[1] += float64(dy) * stw.CanvasMoveSpeedY()
						} else {
							stw.frame.View.Angle[0] += float64(dy) * stw.CanvasRotateSpeedY()
							stw.frame.View.Angle[1] -= float64(dx) * stw.CanvasRotateSpeedX()
						}
						w.Invalidate()
					}
				case pointer.Scroll:
					end = e.Position
					if e.Scroll.Y > 0 {
						stw.Zoom(-1.0, float64(end.X), float64(end.Y))
					} else {
						stw.Zoom(1.0, float64(end.X), float64(end.Y))
					}
					w.Invalidate()
				}

			// this is sent when the application is closed.
			case system.DestroyEvent:
				return e.Err
			}
		}
	}

	return nil
}

func (stw *Window) Layout(gtx layout.Context) layout.Dimensions {
	th := stw.Theme

	for _, e := range gtx.Events(stw) {
		switch e := e.(type) {
		default:
			fmt.Println(e)
		case key.Event:
			fmt.Print(e.Name)
		case pointer.Event:
			switch e.Type {
			case pointer.Press:
				start = e.Position
				press = e.Buttons
				switch press {
				case pointer.ButtonTertiary:
					drawall = false
				}
			case pointer.Release:
				end = e.Position
				drawall = true
				press = 0
				stw.QueueRedrawAll()
			case pointer.Drag:
				end = e.Position
				dx := end.X - start.X
				dy := end.Y - start.Y
				switch press {
				case pointer.ButtonTertiary:
					if e.Modifiers&key.ModShift != 0 {
						stw.frame.View.Center[0] += float64(dx) * stw.CanvasMoveSpeedX()
						stw.frame.View.Center[1] += float64(dy) * stw.CanvasMoveSpeedY()
					} else {
						stw.frame.View.Angle[0] += float64(dy) * stw.CanvasRotateSpeedY()
						stw.frame.View.Angle[1] -= float64(dx) * stw.CanvasRotateSpeedX()
					}
					stw.QueueRedrawAll()
				}
			case pointer.Scroll:
				end = e.Position
				if e.Scroll.Y > 0 {
					stw.Zoom(-1.0, float64(end.X), float64(end.Y))
				} else {
					stw.Zoom(1.0, float64(end.X), float64(end.Y))
				}
				stw.QueueRedrawAll()
			}
		}
	}

	area := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
	key.InputOp{
		Tag: stw,
		Keys: key.Set("Shift-A"),
	}.Add(gtx.Ops)
	pointer.InputOp{
		Tag: stw,
		Types: pointer.Press | pointer.Release | pointer.Drag | pointer.Scroll,
		ScrollBounds: image.Rect(-100,-100,100,100),
	}.Add(gtx.Ops)
	area.Pop()

	// inset is used to add padding around the window border.
	inset := layout.UniformInset(defaultMargin)
	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			// layout.Rigid(layout.Spacer{Height: th.TextSize}.Layout),
			layout.Rigid(layout.Spacer{Height: 12}.Layout),
			layout.Rigid(material.Body1(th, "st").Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if drawall {
					st.DrawFrame(stw, stw.frame, stw.frame.Show.ColorMode, false)
				} else {
					st.DrawFrameNode(stw, stw.frame, stw.frame.Show.ColorMode, false)
				}
				return layout.Dimensions{
					Size: image.Point{X: 2000, Y: 2000},
				}
			}),
		)
	})
}

func (stw *Window) Frame() *st.Frame {
	return stw.frame
}

func (stw *Window) SetFrame(frame *st.Frame) {
	stw.frame = frame
}

func (stw *Window) AddTail(n *st.Node) {
	if tailnodes == nil {
		tailnodes = []*st.Node{n}
	} else {
		tailnodes = append(tailnodes, n)
	}
}

func (stw *Window) EndTail() {
	tailnodes = nil
}

func (stw *Window) TailLine() {
}

func (stw *Window) CurrentPointerPosition() []int {
	return []int{int(end.X), int(end.Y)}
}

func (stw *Window) FeedCommand() {
	command := stw.CommandLineString()
	if command != "" {
		stw.AddCommandHistory(command)
		stw.ClearCommandLine()
		// stw.ClearTypewrite()
		stw.ExecCommand(command)
		stw.Redraw()
	}
}

func (stw *Window) ExecCommand(command string) {
	if stw.frame == nil {
		if strings.HasPrefix(command, ":") {
			err := st.ExMode(stw, command)
			if err != nil {
				if _, ok := err.(st.NotRedraw); ok {
					return
				} else {
					st.ErrorMessage(stw, err, st.ERROR)
				}
			}
		} else if strings.HasPrefix(command, "'") {
			err := st.Fig2Mode(stw, command)
			if err != nil {
				st.ErrorMessage(stw, err, st.ERROR)
			}
		}
		return
	}
	switch {
	default:
		if c, ok := stw.CommandAlias(strings.ToUpper(command)); ok {
			stw.lastcommand = c
			stw.Execute(c(stw))
		} else {
			stw.History(fmt.Sprintf("command doesn't exist: %s", command))
		}
	case strings.HasPrefix(command, ":"):
		err := st.ExMode(stw, command)
		if err != nil {
			if _, ok := err.(st.NotRedraw); ok {
				return
			} else {
				st.ErrorMessage(stw, err, st.ERROR)
			}
		}
	case strings.HasPrefix(command, "'"):
		err := st.Fig2Mode(stw, command)
		if err != nil {
			st.ErrorMessage(stw, err, st.ERROR)
		}
	}
}

func (stw *Window) LastExCommand() string {
	return stw.lastexcommand
}

func (stw *Window) SetLastExCommand(command string) {
	stw.lastexcommand = command
}

func (stw *Window) Complete() string {
	var rtn []string
	str := stw.LastWord()
	switch {
	case strings.HasPrefix(str, ":"):
		i := 0
		rtn = make([]string, len(st.ExAbbrev))
		for ab := range st.ExAbbrev {
			pat := abbrev.MustCompile(ab)
			l := fmt.Sprintf(":%s", pat.Longest())
			if strings.HasPrefix(l, str) {
				rtn[i] = l
				i++
			}
		}
		rtn = rtn[:i]
		sort.Strings(rtn)
	case strings.HasPrefix(str, "'"):
		i := 0
		rtn = make([]string, len(st.Fig2Abbrev))
		for ab := range st.Fig2Abbrev {
			pat := abbrev.MustCompile(ab)
			l := fmt.Sprintf("'%s", pat.Longest())
			if strings.HasPrefix(l, str) {
				rtn[i] = l
				i++
			}
		}
		rtn = rtn[:i]
		sort.Strings(rtn)
	default:
		if lis, ok := stw.ContextComplete(); ok {
			rtn = lis
		} else {
			rtn = st.CompleteFileName(str, stw.frame.Path, stw.Recent())
		}
	}
	if len(rtn) == 0 {
		return str
	} else {
		stw.StartCompletion(rtn)
		return rtn[0]
	}
}

func (stw *Window) LastFig2Command() string {
	return stw.lastfig2command
}

func (stw *Window) SetLastFig2Command(c string) {
	stw.lastfig2command = c
}

func (stw *Window) ShowCenter() {
	stw.SetAngle(stw.frame.View.Angle[0], stw.frame.View.Angle[1])
}

func (stw *Window) Print() {
}

func (stw *Window) SaveAS() {
	st.SaveFile(stw, "hogtxt.inp")
}

func (stw *Window) SaveFileSelected(string) error {
	return nil
}

func (stw *Window) SearchFile(fn string) (string, error) {
	return fn, fmt.Errorf("file not found: %s", fn)
}

func (stw *Window) ShapeData(sh st.Shape) {
	// shapedata = sh
	// var tb *st.TextBox
	// if t, tok := stw.textBox["SHAPE"]; tok {
	// 	tb = t
	// } else {
	// 	tb = st.NewTextBox(NewFont())
	// 	tb.Show()
	// 	w, h := stw.GetCanvasSize()
	// 	tb.SetPosition(float64(w-250), float64(h-225))
	// 	stw.textBox["SHAPE"] = tb
	// }
	// var otp bytes.Buffer
	// otp.WriteString(fmt.Sprintf("%s\n", sh.String()))
	// otp.WriteString(fmt.Sprintf("A   = %10.4f [cm2]\n", sh.A()))
	// otp.WriteString(fmt.Sprintf("Asx = %10.4f [cm2]\n", sh.Asx()))
	// otp.WriteString(fmt.Sprintf("Asy = %10.4f [cm2]\n", sh.Asy()))
	// otp.WriteString(fmt.Sprintf("Ix  = %10.4f [cm4]\n", sh.Ix()))
	// otp.WriteString(fmt.Sprintf("Iy  = %10.4f [cm4]\n", sh.Iy()))
	// otp.WriteString(fmt.Sprintf("J   = %10.4f [cm4]\n", sh.J()))
	// otp.WriteString(fmt.Sprintf("Zx  = %10.4f [cm3]\n", sh.Zx()))
	// otp.WriteString(fmt.Sprintf("Zy  = %10.4f [cm3]\n", sh.Zy()))
	// tb.SetText(strings.Split(otp.String(), "\n"))
}

func (stw *Window) ToggleFixRotate() {
}

func (stw *Window) ToggleFixMove() {
}

func (stw *Window) SetShowPrintRange(val bool) {
	showprintrange = val
	st.SetPosition(stw)
}

func (stw *Window) ToggleShowPrintRange() {
	showprintrange = !showprintrange
	st.SetPosition(stw)
}

func (stw *Window) CurrentLap(string, int, int) {
}

func (stw *Window) SectionData(sec *st.Sect) {
	// var tb *st.TextBox
	// if t, tok := stw.textBox["SECTION"]; tok {
	// 	tb = t
	// } else {
	// 	tb = st.NewTextBox(NewFont())
	// 	tb.Show()
	// 	w, h := stw.GetCanvasSize()
	// 	tb.SetPosition(float64(w-250), float64(h-500))
	// 	stw.textBox["SECTION"] = tb
	// }
	// tb.SetText(strings.Split(sec.InpString(), "\n"))
	// if al := sec.Allow; al != nil {
	// 	tb.AddText(strings.Split(al.String(), "\n")...)
	// }
	// tb.ScrollToTop()
}

func (stw *Window) TextBox(name string) *st.TextBox {
	// if _, tok := stw.textBox[name]; !tok {
	// 	stw.textBox[name] = st.NewTextBox(stw.currentFont)
	// }
	// return stw.textBox[name]
	return nil
}

func (stw *Window) TextBoxes() []*st.TextBox {
	rtn := make([]*st.TextBox, len(stw.textBox))
	i := 0
	for _, t := range stw.textBox {
		rtn[i] = t
		i++
	}
	return rtn
}

func (stw *Window) SetAngle(phi, theta float64) {
	view := st.CenterView(stw, []float64{phi, theta})
	st.Animate(stw, view)
	stw.Redraw()
}

func (stw *Window) SetPaperSize(name uint) {
	stw.papersize = name
}

func (stw *Window) PaperSize() uint {
	return stw.papersize
}

func (stw *Window) SetPeriod(string) {
}

func (stw *Window) Pivot() bool {
	return false
}

func (stw *Window) DrawPivot([]*st.Node, chan int, chan int) {
}

func (stw *Window) SetColorMode(mode uint) {
	stw.frame.Show.ColorMode = mode
}

func (stw *Window) DefaultColorMode() uint {
	return st.ECOLOR_BLACKSECT
}

func (stw *Window) History(str string) {
	if str == "" {
		return
	}
	if strings.HasSuffix(str, "\n") {
		fmt.Printf(str)
	} else {
		fmt.Println(str)
	}
}

func (stw *Window) HistoryWriter() io.Writer {
	return os.Stdout
}

func (stw *Window) Yn(string, string) bool {
	return false
}

func (stw *Window) Yna(string, string, string) int {
	return 0
}

func (stw *Window) EnableLabel(string) {
}

func (stw *Window) DisableLabel(string) {
}

func (stw *Window) SetLabel(k, v string) {
}

func (stw *Window) GetCanvasSize() (int, int) {
	return 2000, 2000
}

func (stw *Window) Changed(c bool) {
	stw.changed = c
}

func (stw *Window) IsChanged() bool {
	return stw.changed
}

func (stw *Window) Redraw() {
	drawall = true
	redrawch <- 1
}

func (stw *Window) RedrawNode() {
	drawall = false
	redrawch <- 1
}

func (stw *Window) EPS() float64 {
	return 1e-3
}

func (stw *Window) SetEPS(float64) {
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
	err = stw.SaveCommandHistory("")
	if err != nil {
		fmt.Println(err)
	}
	closech <- 1
}

// for st.Selector
func (stw *Window) ToggleAltSelectNode() {
	altselectnode = !altselectnode
}

func (stw *Window) AltSelectNode() bool {
	return altselectnode
}

func (stw *Window) SetAltSelectNode(a bool) {
	altselectnode = a
}

func (stw *Window) Zoom(factor float64, x, y float64) float64 {
	val := math.Pow(2.0, factor/stw.CanvasScaleSpeed())
	stw.frame.View.Center[0] += (val - 1.0) * (stw.frame.View.Center[0] - x)
	stw.frame.View.Center[1] += (val - 1.0) * (stw.frame.View.Center[1] - y)
	if stw.frame.View.Perspective {
		stw.frame.View.Dists[1] *= val
		if stw.frame.View.Dists[1] < 0.0 {
			stw.frame.View.Dists[1] = 0.0
		}
	} else {
		stw.frame.View.Gfact *= val
		if stw.frame.View.Gfact < 0.0 {
			stw.frame.View.Gfact = 0.0
		}
	}
	return val
}

func (stw *Window) WindowZoom(factor float64, x, y float64) {
	stw.frame.View.Center[0] += (factor - 1.0) * (stw.frame.View.Center[0] - x)
	stw.frame.View.Center[1] += (factor - 1.0) * (stw.frame.View.Center[1] - y)
	if stw.frame.View.Perspective {
		stw.frame.View.Dists[1] *= factor
		if stw.frame.View.Dists[1] < 0.0 {
			stw.frame.View.Dists[1] = 0.0
		}
	} else {
		stw.frame.View.Gfact *= factor
		if stw.frame.View.Gfact < 0.0 {
			stw.frame.View.Gfact = 0.0
		}
	}
}

var (
	selectbox = []float64{0.0, 0.0, 0.0, 0.0}
)

func (stw *Window) SetTextboxPosition() {
	sx, sy, _, ph := st.GetClipCoord(stw)
	w, _ := stw.GetCanvasSize()
	cw := float64(w)
	// ch := float64(h)
	if t, tok := stw.textBox["SECTION"]; tok {
		t.SetPosition(cw-250, ph+sy-500)
	}
	if t, tok := stw.textBox["PAGETITLE"]; tok {
		t.SetPosition(sx+100, ph-100)
	}
	if t, tok := stw.textBox["TITLE"]; tok {
		t.SetPosition(sx+100, ph-200)
	}
	if t, tok := stw.textBox["DATA"]; tok {
		t.SetPosition(sx+100, 200)
	}
}
