package stlibui

import (
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/andlabs/ui"
	st "github.com/yofu/st/stlib"
)

var (
	canvaswidth          = 2400
	canvasheight         = 1300
	home                 = os.Getenv("HOME")
	pgpfile              = filepath.Join(home, ".st/st.pgp")
	grid                 *ui.Grid
	lefttitle            *ui.Combobox
	leftarea             *ui.Box
	centerarea           *ui.Area
	entry                *ui.Entry
	altselectnode        = true
	showprintrange       = false
	LINE_THICKNESS       = 0.8
	PLATE_OPACITY        = 0.2
	tailnodes            []*st.Node
	selectfromleftbrush  = mkSolidBrush(0xc83200, 0.3)
	selectfromrightbrush = mkSolidBrush(0x0032c8, 0.3)
	windowzoombrush      = mkSolidBrush(0xc8c800, 0.3)
	dragrectbrush        = selectfromleftbrush
	openingfilename      = ""
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
	frame  *st.Frame
	window *ui.Window
	// history         *Dialog
	currentArea        *ui.Area
	currentDrawParam   *ui.AreaDrawParams
	currentStrokeParam *ui.DrawStrokeParams
	currentPen         *ui.DrawBrush
	currentBrush       *ui.DrawBrush
	currentFont        *Font
	papersize          uint
	changed            bool
	lastexcommand      string
	lastfig2command    string
	lastcommand        func(st.Commander) chan bool
	textBox            map[string]*st.TextBox
	textAlignment      int
	// glasses         map[string]*Glass
}

func NewWindow(w *ui.Window) *Window {
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
		window:        w,
		// history:         nil,
		currentStrokeParam: &ui.DrawStrokeParams{
			Cap:        ui.DrawLineCapFlat,
			Join:       ui.DrawLineJoinMiter,
			Thickness:  LINE_THICKNESS,
			MiterLimit: ui.DrawDefaultMiterLimit,
		},
		currentPen:      mkSolidBrush(0x000000, 1.0),
		currentBrush:    mkSolidBrush(0x000000, 1.0),
		currentFont:     NewFont(),
		papersize:       st.A4_TATE,
		changed:         false,
		lastexcommand:   "",
		lastfig2command: "",
		lastcommand:     nil,
		textBox:         make(map[string]*st.TextBox),
		// glasses:         make(map[string]*Glass),
	}
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
	// TODO
}

func (stw *Window) CurrentPointerPosition() []int {
	return []int{int(endX), int(endY)}
}

func (stw *Window) FeedCommand() {
	command := stw.CommandLineString()
	if command != "" {
		stw.AddCommandHistory(command)
		stw.ClearCommandLine()
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
	stw.History(fmt.Sprintf("%s\n", sh.String()))
	stw.History(fmt.Sprintf("A   = %10.4f [cm2]\n", sh.A()))
	stw.History(fmt.Sprintf("Asx = %10.4f [cm2]\n", sh.Asx()))
	stw.History(fmt.Sprintf("Asy = %10.4f [cm2]\n", sh.Asy()))
	stw.History(fmt.Sprintf("Ix  = %10.4f [cm4]\n", sh.Ix()))
	stw.History(fmt.Sprintf("Iy  = %10.4f [cm4]\n", sh.Iy()))
	stw.History(fmt.Sprintf("J   = %10.4f [cm4]\n", sh.J()))
	stw.History(fmt.Sprintf("Zx  = %10.4f [cm3]\n", sh.Zx()))
	stw.History(fmt.Sprintf("Zy  = %10.4f [cm3]\n", sh.Zy()))
}

func (stw *Window) ToggleFixRotate() {
}

func (stw *Window) ToggleFixMove() {
}

func (stw *Window) SetShowPrintRange(val bool) {
	showprintrange = val
}

func (stw *Window) ToggleShowPrintRange() {
	showprintrange = !showprintrange
}

func (stw *Window) CurrentLap(string, int, int) {
}

func (stw *Window) SectionData(*st.Sect) {
}

func (stw *Window) TextBox(name string) *st.TextBox {
	if _, tok := stw.textBox[name]; !tok {
		stw.textBox[name] = st.NewTextBox(stw.currentFont)
	}
	return stw.textBox[name]
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
	view := st.CanvasCenterView(stw, []float64{phi, theta})
	st.Animate(stw, view)
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
	return canvaswidth, canvasheight
}

func (stw *Window) Changed(c bool) {
	stw.changed = c
}

func (stw *Window) IsChanged() bool {
	return stw.changed
}

func (stw *Window) Redraw() {
	stw.currentArea.QueueRedrawAll()
}

func (stw *Window) RedrawNode() {
	drawall = false
	stw.currentArea.QueueRedrawAll()
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
	stw.window.Destroy()
	ui.Quit()
	os.Exit(0)
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

func (stw *Window) Zoom(factor float64, x, y float64) {
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
	drawall   = true
)

func (stw *Window) Draw(a *ui.Area, p *ui.AreaDrawParams) {
	stw.currentArea = a
	stw.currentDrawParam = p
	if stw.frame == nil {
		fmt.Println("Frame is nil")
		return
	}
	if stw.frame.Name != openingfilename {
		stw.UpdateWindow(stw.frame.Name)
	}
	if drawall {
		st.DrawFrame(stw, stw.frame, stw.frame.Show.ColorMode, true)
		if selectbox[2] != 0.0 && selectbox[3] != 0.0 {
			path := ui.DrawNewPath(ui.DrawFillModeWinding)
			path.AddRectangle(selectbox[0], selectbox[1], selectbox[2], selectbox[3])
			path.End()
			p.Context.Fill(path, dragrectbrush)
			path.Free()
		}
	} else {
		st.DrawFrameNode(stw, stw.frame, stw.frame.Show.ColorMode, true)
	}
}

var (
	startX = 0.0
	startY = 0.0
	endX   = 0.0
	endY   = 0.0
	startT time.Time
	endT   time.Time
)

func (stw *Window) MouseEvent(a *ui.Area, me *ui.AreaMouseEvent) {
	if me.Down != 0 { // click
		switch me.Count {
		case 1: // single click
			startX = me.X
			startY = me.Y
			startT = time.Now()
			if me.Down == 1 { // left click
				selectbox[0] = startX
				selectbox[1] = startY
			} else if me.Down == 2 { // center click
				drawall = false
			}
		case 2: // double click
			if me.Down == 2 {
				stw.ShowCenter()
			}
		}
	} else if me.Up != 0 {
		endX = me.X
		endY = me.Y
		endT = time.Now()
		drawall = true
		switch me.Up {
		case 1:
			if me.Modifiers&ui.Ctrl != 0 {
				if selectbox[2] == 0.0 || selectbox[3] == 0.0 {
					selectbox[2] = 0.0
					selectbox[3] = 0.0
					return
				}
				rate1 := float64(canvaswidth) / selectbox[2]
				rate2 := float64(canvasheight) / selectbox[3]
				if rate1 < rate2 {
					stw.WindowZoom(rate1, (startX+endX)*0.5, (startY+endY)*0.5)
				} else {
					stw.WindowZoom(rate2, (startX+endX)*0.5, (startY+endY)*0.5)
				}
			} else if (me.Modifiers&ui.Alt == 0) == altselectnode {
				els, picked := st.PickElem(stw, int(startX), int(startY), int(endX), int(endY))
				if !picked {
					stw.DeselectElem()
				} else {
					st.MergeSelectElem(stw, els, me.Modifiers&ui.Shift != 0)
				}
			} else {
				ns, picked := st.PickNode(stw, int(startX), int(startY), int(endX), int(endY))
				if !picked {
					stw.DeselectNode()
				} else {
					st.MergeSelectNode(stw, ns, me.Modifiers&ui.Shift != 0)
				}
			}
			selectbox[2] = 0.0
			selectbox[3] = 0.0
		case 4:
			stw.Zoom(startT.Sub(endT).Seconds(), endX, endY)
		case 5:
			stw.Zoom(endT.Sub(startT).Seconds(), endX, endY)
		}
		a.QueueRedrawAll()
	} else {
		endX = me.X
		endY = me.Y
		endT = time.Now()
		dx := endX - startX
		dy := endY - startY
		for _, i := range me.Held {
			if i == 1 {
				selectbox[2] = dx
				selectbox[3] = dy
				if me.Modifiers&ui.Ctrl != 0 {
					dragrectbrush = windowzoombrush
				} else if dx > 0.0 {
					dragrectbrush = selectfromleftbrush
				} else {
					dragrectbrush = selectfromrightbrush
				}
				a.QueueRedrawAll()
				break
			} else if i == 2 {
				if me.Modifiers&ui.Shift != 0 {
					stw.frame.View.Center[0] += float64(dx) * stw.CanvasMoveSpeedX()
					stw.frame.View.Center[1] += float64(dy) * stw.CanvasMoveSpeedY()
				} else {
					stw.frame.View.Angle[0] += float64(dy) * stw.CanvasRotateSpeedY()
					stw.frame.View.Angle[1] -= float64(dx) * stw.CanvasRotateSpeedX()
				}
				a.QueueRedrawAll()
				break
			}
		}
	}
}

func (stw *Window) MouseCrossed(a *ui.Area, left bool) {
	// do nothing
}

func (stw *Window) DragBroken(a *ui.Area) {
	// do nothing
}

func (stw *Window) KeyEvent(a *ui.Area, ke *ui.AreaKeyEvent) (handled bool) {
	if ke.Up {
		return false
	}
	// fmt.Println(ke.ExtKey, ke.Key, ke.Modifier, ke.Modifiers)
	// TODO: use CommandLine
	if ke.ExtKey != 0 {
		switch ke.ExtKey {
		default:
			return false
		case ui.Escape:
			stw.Deselect()
			stw.Redraw()
			entry.SetText("")
			return true
		}
	} else {
		switch ke.Key {
		default:
			entry.SetText(entry.Text() + string(ke.Key))
			return true
		case 10: // Enter
			command := entry.Text()
			stw.ExecCommand(command)
			entry.SetText("")
			stw.Redraw()
			return true
		case 8: // Backspace
			t := entry.Text()
			entry.SetText(t[:len(t)-1])
			return true
		case '1':
			if ke.Modifiers&ui.Shift != 0 {
				entry.SetText(entry.Text() + "!")
			} else {
				entry.SetText(entry.Text() + string(ke.Key))
			}
			return true
		case ';':
			if ke.Modifiers&ui.Shift != 0 {
				entry.SetText(entry.Text() + ";")
			} else {
				entry.SetText(entry.Text() + ":")
			}
			return true
		}
	}
}

func SetupLayerTab(stw *Window) *ui.Box {
	leftarea := ui.NewVerticalBox()
	layercbs := make([]*ui.Checkbox, 0)
	cb := ui.NewCheckbox("All")
	cb.OnToggled(func(c *ui.Checkbox) {
		if c.Checked() {
			st.ShowAllSection(stw)
			for _, c := range layercbs {
				c.SetChecked(true)
			}
		} else {
			st.HideAllSection(stw)
			for _, c := range layercbs {
				c.SetChecked(false)
			}
		}
		centerarea.QueueRedrawAll()
	})
	cb.SetChecked(true)
	leftarea.Append(cb, false)

	leftarea.Append(ui.NewHorizontalSeparator(), false)

	snums := make([]int, len(stw.Frame().Sects))
	ind := 0
	for s := range stw.Frame().Sects {
		if s <= 100 || s >= 900 {
			continue
		}
		snums[ind] = s
		ind++
	}
	snums = snums[:ind]
	sort.Ints(snums)
	for _, s := range snums {
		snum := s
		cb := ui.NewCheckbox(fmt.Sprintf("%d: %s", snum, stw.Frame().Sects[snum].Name))
		cb.OnToggled(func(c *ui.Checkbox) {
			if c.Checked() {
				st.ShowSection(stw, snum)
			} else {
				st.HideSection(stw, snum)
			}
			centerarea.QueueRedrawAll()
		})
		cb.SetChecked(true)
		leftarea.Append(cb, false)
		layercbs = append(layercbs, cb)
	}

	leftarea.Append(ui.NewHorizontalSeparator(), false)

	for i, et := range st.ETYPES {
		etype := i
		if etype == st.NULL || etype == st.TRUSS || etype == st.WBRACE || etype == st.SBRACE {
			continue
		}
		etypename := et
		cb := ui.NewCheckbox(fmt.Sprintf("%s", etypename))
		cb.OnToggled(func(c *ui.Checkbox) {
			if c.Checked() {
				st.ShowEtype(stw, etype)
			} else {
				st.HideEtype(stw, etype)
			}
			centerarea.QueueRedrawAll()
		})
		cb.SetChecked(true)
		leftarea.Append(cb, false)
	}

	return leftarea
}

func SetupWindow(fn string) {
	mainwin := ui.NewWindow("st", canvaswidth, canvasheight, true)
	mainwin.SetMargined(true)
	mainwin.OnClosing(func(*ui.Window) bool {
		mainwin.Destroy()
		ui.Quit()
		return false
	})
	ui.OnShouldQuit(func() bool {
		mainwin.Destroy()
		return true
	})

	stw := NewWindow(mainwin)
	stw.ReadRecent()
	if fn != "" {
		st.OpenFile(stw, fn, true)
		openingfilename = fn
		stw.UpdateWindow(fn)
	} else {
		st.ShowRecent(stw)
	}
	st.ReadPgp(stw, pgpfile)
	stw.ReadCommandHistory("")
	stw.frame.View.Center[0] = 0.5 * float64(canvaswidth)
	stw.frame.View.Center[1] = 0.5 * float64(canvasheight)
	stw.frame.Show.PlateEdge = false
	stw.frame.Show.ColorMode = st.ECOLOR_BLACKSECT

	hbox := ui.NewHorizontalBox()
	hbox.SetPadded(true)
	mainwin.SetChild(hbox)

	grid = ui.NewGrid()
	grid.SetPadded(true)
	hbox.Append(grid, true)

	leftarea = SetupLayerTab(stw)

	centerarea = ui.NewArea(stw)
	stw.currentArea = centerarea

	entry = ui.NewEntry()

	lefttitle = ui.NewCombobox()
	centertitle := ui.NewCombobox()
	lefttitle.Append("LAYER")
	lefttitle.SetSelected(0)
	centertitle.Append("MODEL")
	centertitle.SetSelected(0)

	grid.Append(lefttitle, 0, 0, 1, 1, false, ui.AlignFill, false, ui.AlignFill)
	grid.Append(leftarea, 0, 1, 1, 2, false, ui.AlignFill, true, ui.AlignFill)

	grid.Append(centertitle, 1, 0, 1, 1, false, ui.AlignFill, false, ui.AlignFill)
	grid.Append(centerarea, 1, 1, 1, 1, true, ui.AlignFill, true, ui.AlignFill)
	grid.Append(entry, 1, 2, 1, 1, true, ui.AlignFill, false, ui.AlignFill)

	mainwin.Show()
}

func (stw *Window) UpdateWindow(fn string) {
	openingfilename = fn
	stw.window.SetTitle(fn)
	// TODO: update LAYER
	// leftarea.Destroy()
	// newleftarea := SetupLayerTab(stw)
	// grid.InsertAt(newleftarea, lefttitle, ui.Bottom, 1, 2, false, ui.AlignFill, true, ui.AlignFill)
	// leftarea = newleftarea
}
