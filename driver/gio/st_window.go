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
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
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
		// Theme:         material.NewTheme(gofont.Collection()),
		Theme:         material.NewTheme(),
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
	var mousePos f32.Point
	events := make(chan event.Event)
	acks := make(chan struct{})
	go func() {
		for {
			ev := w.Event()
			events <- ev
			<-acks
			if _, ok := ev.(app.DestroyEvent); ok {
				return
			}
		}
	}()
	mousePresent := false
	for {
		select {
		case <-redrawch:
			w.Invalidate()
		case <-closech:
			return nil
		case e := <-events:
			// detect the type of the event.
			switch e := e.(type) {
			default:
			// this is sent when the application should re-render.
			case app.FrameEvent:
				gtx := app.NewContext(&ops, e)
				stw.context = gtx

				r := image.Rectangle{Max: image.Point{X: gtx.Constraints.Max.X, Y: gtx.Constraints.Max.Y}}
				area := clip.Rect(r).Push(&ops)
				event.Op(&ops, &mousePos)
				area.Pop()
				for {
					ev, ok := gtx.Event(pointer.Filter{
						Target: &mousePos,
						Kinds:  pointer.Move | pointer.Enter | pointer.Leave,
					})
					if !ok {
						break
					}
					switch ev := ev.(type) {
					case pointer.Event:
						switch ev.Kind {
						case pointer.Enter:
							mousePresent = true
						case pointer.Leave:
							mousePresent = false
						case pointer.Press:

						}
						mousePos = ev.Position
					}
				}
				if mousePresent {
					fmt.Println("Mouse Position: (%.2f, %.2f)", mousePos.X, mousePos.Y)
				}

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
				switch e.Kind {
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
			case app.DestroyEvent:
				acks <- struct{}{}
				return e.Err
			}
			acks <- struct{}{}
		}
	}

	return nil
}

func (stw *Window) Layout(gtx layout.Context) layout.Dimensions {
	th := stw.Theme


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
				layoutSelectionLayer(gtx)
				return layout.Dimensions{
					Size: image.Point{X: 2000, Y: 2000},
				}
			}),
		)
	})
}

// viewport models a region of a larger space. Offset is the location
// of the upper-left corner of the view within the larger space. size
// is the dimensions of the viewport within the larger space.
type viewport struct {
	offset f32.Point
	size   f32.Point
}
// subview modifies v to describe a smaller region by zooming into the
// space described by v using other.
func (v *viewport) subview(other *viewport) {
	v.offset.X += other.offset.X * v.size.X
	v.offset.Y += other.offset.Y * v.size.Y
	v.size.X *= other.size.X
	v.size.Y *= other.size.Y
}


var (
	selected    image.Rectangle
	selecting   = false
	view        *viewport
)
// ensureSquare returns a copy of the rectangle that has been padded to
// be square by increasing the maximum coordinate.
func ensureSquare(r image.Rectangle) image.Rectangle {
	dx := r.Dx()
	dy := r.Dy()
	if dx > dy {
		r.Max.Y = r.Min.Y + dx
	} else if dy > dx {
		r.Max.X = r.Min.X + dy
	}
	return r
}

func layoutSelectionLayer(gtx layout.Context) layout.Dimensions {
	for {
		event, ok := gtx.Event(pointer.Filter{
			Target: &selected,
			Kinds:  pointer.Press | pointer.Release | pointer.Drag,
		})
		if !ok {
			break
		}
		switch event := event.(type) {
		case pointer.Event:
			var intPt image.Point
			intPt.X = int(event.Position.X)
			intPt.Y = int(event.Position.Y)
			switch event.Kind {
			case pointer.Press:
				selecting = true
				selected.Min = intPt
				selected.Max = intPt
			case pointer.Drag:
				if intPt.X >= selected.Min.X && intPt.Y >= selected.Min.Y {
					selected.Max = intPt
				} else {
					selected.Min = intPt
				}
				selected = ensureSquare(selected)
			case pointer.Release:
				selecting = false
				newView := &viewport{
					offset: f32.Point{
						X: float32(selected.Min.X) / float32(gtx.Constraints.Max.X),
						Y: float32(selected.Min.Y) / float32(gtx.Constraints.Max.Y),
					},
					size: f32.Point{
						X: float32(selected.Dx()) / float32(gtx.Constraints.Max.X),
						Y: float32(selected.Dy()) / float32(gtx.Constraints.Max.Y),
					},
				}
				if view == nil {
					view = newView
				} else {
					view.subview(newView)
				}
			case pointer.Cancel:
				selecting = false
				selected = image.Rectangle{}
			}
		}
	}
	if selecting {
		paint.FillShape(gtx.Ops, color.NRGBA{R: 255, A: 100}, clip.Rect(selected).Op())
	}
	pr := clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops)
	pointer.CursorCrosshair.Add(gtx.Ops)
	event.Op(gtx.Ops, &selected)
	pr.Pop()

	return layout.Dimensions{Size: gtx.Constraints.Max}
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
