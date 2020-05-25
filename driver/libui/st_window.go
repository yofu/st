package stlibui

import (
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/andlabs/ui"
	st "github.com/yofu/st/stlib"
)

var (
	canvaswidth    = 2400
	canvasheight   = 1300
	leftarea       *ui.Box
	centerarea     *ui.Area
	altselectnode  = true
	showprintrange = false
	LINE_THICKNESS = 0.8
	PLATE_OPACITY  = 0.2
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
	currentFont        *ui.FontDescriptor
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
		currentPen:   mkSolidBrush(0x000000, 1.0),
		currentBrush: mkSolidBrush(0x000000, 1.0),
		currentFont: &ui.FontDescriptor{
			Family:  "IPA明朝",
			Size:    9,
			Weight:  400,
			Italic:  ui.TextItalicNormal,
			Stretch: ui.TextStretchCondensed,
		},
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

func (stw *Window) ExecCommand(command string) {
	// if stw.frame == nil {
	// 	if strings.HasPrefix(command, ":") {
	// 		err := st.ExMode(stw, command)
	// 		if err != nil {
	// 			if _, ok := err.(st.NotRedraw); ok {
	// 				return
	// 			} else {
	// 				st.ErrorMessage(stw, err, st.ERROR)
	// 			}
	// 		}
	// 	} else if strings.HasPrefix(command, "'") {
	// 		err := st.Fig2Mode(stw, command)
	// 		if err != nil {
	// 			st.ErrorMessage(stw, err, st.ERROR)
	// 		}
	// 	}
	// 	return
	// }
	// switch {
	// default:
	// 	if c, ok := stw.CommandAlias(strings.ToUpper(command)); ok {
	// 		stw.lastcommand = c
	// 		stw.Execute(c(stw))
	// 	} else {
	// 		stw.History(fmt.Sprintf("command doesn't exist: %s", command))
	// 	}
	// case strings.HasPrefix(command, ":"):
	// 	err := st.ExMode(stw, command)
	// 	if err != nil {
	// 		if _, ok := err.(st.NotRedraw); ok {
	// 			return
	// 		} else {
	// 			st.ErrorMessage(stw, err, st.ERROR)
	// 		}
	// 	}
	// case strings.HasPrefix(command, "'"):
	// 	err := st.Fig2Mode(stw, command)
	// 	if err != nil {
	// 		st.ErrorMessage(stw, err, st.ERROR)
	// 	}
	// }
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
	st.DrawFrame(stw, stw.frame, stw.frame.Show.ColorMode, true)
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
	if drawall {
		st.DrawFrame(stw, stw.frame, stw.frame.Show.ColorMode, true)
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
	if me.Down != 0 { // left click
		if me.Count == 1 { // single click
			startX = me.X
			startY = me.Y
			startT = time.Now()
		}
		if me.Down == 1 {
			selectbox[0] = startX
			selectbox[1] = startY
		} else if me.Down == 2 {
			drawall = false
		}
	} else if me.Up != 0 {
		endX = me.X
		endY = me.Y
		endT = time.Now()
		drawall = true
		switch me.Up {
		case 1:
			if (me.Modifiers&ui.Alt == 0) == altselectnode {
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
	// reject all keys
	return false
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
		if s >= 900 {
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
	if fn != "" {
		st.OpenFile(stw, fn, true)
		stw.window.SetTitle(fn)
		stw.frame.Show.ElemCaptionOn(st.EC_SECT)
	}

	hbox := ui.NewHorizontalBox()
	hbox.SetPadded(true)
	mainwin.SetChild(hbox)

	grid := ui.NewGrid()
	grid.SetPadded(true)
	hbox.Append(grid, true)

	leftarea = SetupLayerTab(stw)

	centerarea = ui.NewArea(stw)
	stw.currentArea = centerarea

	entry := ui.NewEntry()

	lefttitle := ui.NewCombobox()
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
