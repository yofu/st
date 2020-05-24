package stlibui

import (
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/andlabs/ui"
	st "github.com/yofu/st/stlib"
)

var (
	area          *ui.Area
	table         *ui.Table
	altselectnode = true
	PLATE_OPACITY = 0.2
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
	// buffer          screen.Buffer
	currentPen   *ui.DrawBrush
	currentBrush *ui.DrawBrush
	// font            *Font
	papersize       uint
	changed         bool
	lastexcommand   string
	lastfig2command string
	lastcommand     func(st.Commander) chan bool
	textBox         map[string]*st.TextBox
	textAlignment   int
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
		// buffer:          nil,
		currentPen:   mkSolidBrush(0xffffff, 1.0),
		currentBrush: mkSolidBrush(0xffffff, 1.0),
		// font:            basicFont,
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
	return 1200, 1200
}

func (stw *Window) Changed(c bool) {
	stw.changed = c
}

func (stw *Window) IsChanged() bool {
	return stw.changed
}

func (stw *Window) Redraw() {
	area.QueueRedrawAll()
}

func (stw *Window) RedrawNode() {
	drawall = false
	area.QueueRedrawAll()
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

const colorBlack = 0x000000

// helper to quickly set a brush color
func mkSolidBrush(color uint32, alpha float64) *ui.DrawBrush {
	factor := 1.0
	brush := new(ui.DrawBrush)
	brush.Type = ui.DrawBrushTypeSolid
	component := uint8((color >> 16) & 0xFF)
	brush.R = float64(component) / 255 * factor
	component = uint8((color >> 8) & 0xFF)
	brush.G = float64(component) / 255 * factor
	component = uint8(color & 0xFF)
	brush.B = float64(component) / 255 * factor
	brush.A = alpha
	return brush
}

var (
	font = &ui.FontDescriptor{
		Family:  "IPA明朝",
		Size:    9,
		Weight:  400,
		Italic:  ui.TextItalicNormal,
		Stretch: ui.TextStretchCondensed,
	}
	selectbox = []float64{0.0, 0.0, 0.0, 0.0}
	drawall   = true
)

func (stw *Window) Draw(a *ui.Area, p *ui.AreaDrawParams) {
	if stw.frame == nil {
		fmt.Println("Frame is nil")
		return
	}
	if drawall {
		stw.DrawFrame(a, p)
	} else {
		stw.DrawFrameNode(a, p)
	}
}

func (stw *Window) DrawFrame(a *ui.Area, p *ui.AreaDrawParams) {
	brush := mkSolidBrush(0xffffff, 1.0)
	path := ui.DrawNewPath(ui.DrawFillModeWinding)
	path.AddRectangle(0, 0, p.AreaWidth, p.AreaHeight)
	path.End()
	p.Context.Fill(path, brush)
	path.Free()

	sp := &ui.DrawStrokeParams{
		Cap:        ui.DrawLineCapFlat,
		Join:       ui.DrawLineJoinMiter,
		Thickness:  0.8,
		MiterLimit: ui.DrawDefaultMiterLimit,
	}
	sps := &ui.DrawStrokeParams{
		Cap:        ui.DrawLineCapFlat,
		Join:       ui.DrawLineJoinMiter,
		Thickness:  0.8,
		MiterLimit: ui.DrawDefaultMiterLimit,
		Dashes:     []float64{4, 2},
	}
	stw.frame.View.Set(1)
	for _, n := range stw.frame.Nodes {
		stw.frame.View.ProjectNode(n)
	}
	els := st.SortedElem(stw.frame.Elems, func(e *st.Elem) float64 { return -e.DistFromProjection(stw.frame.View) })
loop:
	for _, elem := range els {
		if elem.IsHidden(stw.frame.Show) {
			continue
		}
		for _, j := range stw.SelectedElems() {
			if j == elem {
				continue loop
			}
		}
		if elem.IsLineElem() {
			// brush := mkSolidBrush(uint32(elem.Sect.Color), 1.0)
			brush := mkSolidBrush(0x000000, 1.0)
			path := ui.DrawNewPath(ui.DrawFillModeWinding)
			icoord := elem.Enod[0].Pcoord
			jcoord := elem.Enod[1].Pcoord
			path.NewFigure(icoord[0], icoord[1])
			path.LineTo(jcoord[0], jcoord[1])
			path.End()
			p.Context.Stroke(path, brush, sp)
			path.Free()
		} else {
			fbrush := mkSolidBrush(uint32(elem.Sect.Color), 0.2)
			path := ui.DrawNewPath(ui.DrawFillModeWinding)
			path.NewFigure(elem.Enod[0].Pcoord[0], elem.Enod[0].Pcoord[1])
			for i := 1; i < elem.Enods; i++ {
				path.LineTo(elem.Enod[i].Pcoord[0], elem.Enod[i].Pcoord[1])
			}
			path.End()
			p.Context.Fill(path, fbrush)
			path.Free()
		}
	}
	for _, elem := range stw.SelectedElems() {
		if elem.IsLineElem() {
			brush := mkSolidBrush(0x000000, 1.0)
			path := ui.DrawNewPath(ui.DrawFillModeWinding)
			icoord := elem.Enod[0].Pcoord
			jcoord := elem.Enod[1].Pcoord
			path.NewFigure(icoord[0], icoord[1])
			path.LineTo(jcoord[0], jcoord[1])
			path.End()
			p.Context.Stroke(path, brush, sps)
			path.Free()
		} else {
			fbrush := mkSolidBrush(uint32(elem.Sect.Color), 0.7)
			path := ui.DrawNewPath(ui.DrawFillModeWinding)
			path.NewFigure(elem.Enod[0].Pcoord[0], elem.Enod[0].Pcoord[1])
			for i := 1; i < elem.Enods; i++ {
				path.LineTo(elem.Enod[i].Pcoord[0], elem.Enod[i].Pcoord[1])
			}
			path.End()
			p.Context.Fill(path, fbrush)
			path.Free()
		}
	}
	for _, node := range stw.frame.Nodes {
		str1 := ui.NewAttributedString(fmt.Sprintf("%d", node.Num))
		tl1 := ui.DrawNewTextLayout(&ui.DrawTextLayoutParams{
			String:      str1,
			DefaultFont: font,
			Width:       500,
			Align:       ui.DrawTextAlign(ui.DrawTextAlignLeft),
		})
		p.Context.Text(tl1, node.Pcoord[0], node.Pcoord[1]-float64(font.Size))
		tl1.Free()
	}

	if selectbox[2] != 0.0 && selectbox[3] != 0.0 {
		var brush *ui.DrawBrush
		if selectbox[2] > 0.0 {
			brush = mkSolidBrush(0xc83200, 0.3)
		} else {
			brush = mkSolidBrush(0x0032c8, 0.3)
		}
		path := ui.DrawNewPath(ui.DrawFillModeWinding)
		path.AddRectangle(selectbox[0], selectbox[1], selectbox[2], selectbox[3])
		path.End()
		p.Context.Fill(path, brush)
		path.Free()
	}
}

func (stw *Window) DrawFrameNode(a *ui.Area, p *ui.AreaDrawParams) {
	brush := mkSolidBrush(0xffffff, 1.0)
	path := ui.DrawNewPath(ui.DrawFillModeWinding)
	path.AddRectangle(0, 0, p.AreaWidth, p.AreaHeight)
	path.End()
	p.Context.Fill(path, brush)
	path.Free()

	sp := &ui.DrawStrokeParams{
		Cap:        ui.DrawLineCapFlat,
		Join:       ui.DrawLineJoinMiter,
		Thickness:  0.8,
		MiterLimit: ui.DrawDefaultMiterLimit,
	}
	sps := &ui.DrawStrokeParams{
		Cap:        ui.DrawLineCapFlat,
		Join:       ui.DrawLineJoinMiter,
		Thickness:  0.8,
		MiterLimit: ui.DrawDefaultMiterLimit,
		Dashes:     []float64{4, 2},
	}
	stw.frame.View.Set(1)
	for _, n := range stw.frame.Nodes {
		stw.frame.View.ProjectNode(n)
	}
	els := st.SortedElem(stw.frame.Elems, func(e *st.Elem) float64 { return -e.DistFromProjection(stw.frame.View) })
loop:
	for _, elem := range els {
		for _, j := range stw.SelectedElems() {
			if j == elem {
				continue loop
			}
		}
		if elem.IsLineElem() {
			// brush := mkSolidBrush(uint32(elem.Sect.Color), 1.0)
			brush := mkSolidBrush(0x00c832, 0.5)
			path := ui.DrawNewPath(ui.DrawFillModeWinding)
			icoord := elem.Enod[0].Pcoord
			jcoord := elem.Enod[1].Pcoord
			path.NewFigure(icoord[0], icoord[1])
			path.LineTo(jcoord[0], jcoord[1])
			path.End()
			p.Context.Stroke(path, brush, sp)
			path.Free()
		}
	}
	for _, elem := range stw.SelectedElems() {
		if elem.IsLineElem() {
			brush := mkSolidBrush(0x000000, 1.0)
			path := ui.DrawNewPath(ui.DrawFillModeWinding)
			icoord := elem.Enod[0].Pcoord
			jcoord := elem.Enod[1].Pcoord
			path.NewFigure(icoord[0], icoord[1])
			path.LineTo(jcoord[0], jcoord[1])
			path.End()
			p.Context.Stroke(path, brush, sps)
			path.Free()
		} else {
			fbrush := mkSolidBrush(uint32(elem.Sect.Color), 0.7)
			path := ui.DrawNewPath(ui.DrawFillModeWinding)
			path.NewFigure(elem.Enod[0].Pcoord[0], elem.Enod[0].Pcoord[1])
			for i := 1; i < elem.Enods; i++ {
				path.LineTo(elem.Enod[i].Pcoord[0], elem.Enod[i].Pcoord[1])
			}
			path.End()
			p.Context.Fill(path, fbrush)
			path.Free()
		}
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

var (
	nrow     = 25
	tablekey = []string{
		"GFACT",
		"R",
		"L",
		"φ",
		"θ",
		"FX",
		"FY",
		"FZ",
		"CX",
		"CY",
		"Xmax",
		"Xmin",
		"Ymax",
		"Ymin",
		"Zmax",
		"Zmin",
		"PERIOD",
		"GAXIS",
		"EAXIS",
		"BOND",
		"CONF",
		"DFACT",
		"QFACT",
		"MFACT",
		"RFACT",
	}
)

type modelHandler struct {
	window *Window
}

func newModelHandler(w *Window) *modelHandler {
	m := new(modelHandler)
	m.window = w
	return m
}

func (mh *modelHandler) ColumnTypes(m *ui.TableModel) []ui.TableValue {
	return []ui.TableValue{
		ui.TableString(""), // column 0 text
		ui.TableString(""), // column 1 text
	}
}

func (mh *modelHandler) NumRows(m *ui.TableModel) int {
	return nrow
}

func (mh *modelHandler) CellValue(m *ui.TableModel, row, column int) ui.TableValue {
	switch column {
	case 0:
		return ui.TableString(tablekey[row])
	case 1:
		switch row {
		case 0:
			return ui.TableString(fmt.Sprintf("%.3f", mh.window.Frame().View.Gfact))
		case 1:
			return ui.TableString(fmt.Sprintf("%.3f", mh.window.Frame().View.Dists[0]))
		case 2:
			return ui.TableString(fmt.Sprintf("%.3f", mh.window.Frame().View.Dists[1]))
		case 3:
			return ui.TableString(fmt.Sprintf("%.3f", mh.window.Frame().View.Angle[0]))
		case 4:
			return ui.TableString(fmt.Sprintf("%.3f", mh.window.Frame().View.Angle[1]))
		case 5:
			return ui.TableString(fmt.Sprintf("%.3f", mh.window.Frame().View.Focus[0]))
		case 6:
			return ui.TableString(fmt.Sprintf("%.3f", mh.window.Frame().View.Focus[1]))
		case 7:
			return ui.TableString(fmt.Sprintf("%.3f", mh.window.Frame().View.Focus[2]))
		case 8:
			return ui.TableString(fmt.Sprintf("%.3f", mh.window.Frame().View.Center[0]))
		case 9:
			return ui.TableString(fmt.Sprintf("%.3f", mh.window.Frame().View.Center[1]))
		case 10:
			return ui.TableString(fmt.Sprintf("%.3f", mh.window.Frame().Show.Xrange[0]))
		case 11:
			return ui.TableString(fmt.Sprintf("%.3f", mh.window.Frame().Show.Xrange[1]))
		case 12:
			return ui.TableString(fmt.Sprintf("%.3f", mh.window.Frame().Show.Yrange[0]))
		case 13:
			return ui.TableString(fmt.Sprintf("%.3f", mh.window.Frame().Show.Yrange[1]))
		case 14:
			return ui.TableString(fmt.Sprintf("%.3f", mh.window.Frame().Show.Zrange[0]))
		case 15:
			return ui.TableString(fmt.Sprintf("%.3f", mh.window.Frame().Show.Zrange[1]))
		case 16:
			return ui.TableString(fmt.Sprintf("%s", mh.window.Frame().Show.Period))
		case 17:
			return ui.TableString(fmt.Sprintf("%.3f", mh.window.Frame().Show.GlobalAxisSize))
		case 18:
			return ui.TableString(fmt.Sprintf("%.3f", mh.window.Frame().Show.ElementAxisSize))
		case 19:
			return ui.TableString(fmt.Sprintf("%.3f", mh.window.Frame().Show.BondSize))
		case 20:
			return ui.TableString(fmt.Sprintf("%.3f", mh.window.Frame().Show.ConfSize))
		case 21:
			return ui.TableString(fmt.Sprintf("%.3f", mh.window.Frame().Show.Dfact))
		case 22:
			return ui.TableString(fmt.Sprintf("%.3f", mh.window.Frame().Show.Qfact))
		case 23:
			return ui.TableString(fmt.Sprintf("%.3f", mh.window.Frame().Show.Mfact))
		case 24:
			return ui.TableString(fmt.Sprintf("%.3f", mh.window.Frame().Show.Rfact))
		}
	}
	panic("unreachable")
}

func (mh *modelHandler) SetCellValue(m *ui.TableModel, row, column int, value ui.TableValue) {
	if column == 1 {
		if row == 16 {
			str := strings.ToUpper(string(value.(ui.TableString)))
			for _, p := range st.PERIODS {
				if str == p {
					mh.window.Frame().Show.Period = p
				}
			}
		} else {
			val, err := strconv.ParseFloat(string(value.(ui.TableString)), 64)
			if err != nil {
				return
			}
			switch row {
			case 0:
				mh.window.Frame().View.Gfact = val
			case 1:
				mh.window.Frame().View.Dists[0] = val
			case 2:
				mh.window.Frame().View.Dists[1] = val
			case 3:
				mh.window.Frame().View.Angle[0] = val
			case 4:
				mh.window.Frame().View.Angle[1] = val
			case 5:
				mh.window.Frame().View.Focus[0] = val
			case 6:
				mh.window.Frame().View.Focus[1] = val
			case 7:
				mh.window.Frame().View.Focus[2] = val
			case 8:
				mh.window.Frame().View.Center[0] = val
			case 9:
				mh.window.Frame().View.Center[1] = val
			case 10:
				mh.window.Frame().Show.Xrange[0] = val
			case 11:
				mh.window.Frame().Show.Xrange[1] = val
			case 12:
				mh.window.Frame().Show.Yrange[0] = val
			case 13:
				mh.window.Frame().Show.Yrange[1] = val
			case 14:
				mh.window.Frame().Show.Zrange[0] = val
			case 15:
				mh.window.Frame().Show.Zrange[1] = val
			case 17:
				mh.window.Frame().Show.GlobalAxisSize = val
			case 18:
				mh.window.Frame().Show.ElementAxisSize = val
			case 19:
				mh.window.Frame().Show.BondSize = val
			case 20:
				mh.window.Frame().Show.ConfSize = val
			case 21:
				mh.window.Frame().Show.Dfact = val
			case 22:
				mh.window.Frame().Show.Qfact = val
			case 23:
				mh.window.Frame().Show.Mfact = val
			case 24:
				mh.window.Frame().Show.Rfact = val
			}
		}
	}
}

func SetupWindow(w *ui.Window, fn string) {
	mainwin := ui.NewWindow("st", 1600, 1200, true)
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
	}

	hbox := ui.NewHorizontalBox()
	hbox.SetPadded(true)
	mainwin.SetChild(hbox)

	grid := ui.NewGrid()
	grid.SetPadded(true)
	hbox.Append(grid, true)

	mh := newModelHandler(stw)
	model := ui.NewTableModel(mh)
	table = ui.NewTable(&ui.TableParams{
		Model:                         model,
		RowBackgroundColorModelColumn: -1,
	})
	table.AppendTextColumn("KEY",
		0, ui.TableModelColumnNeverEditable, nil)
	table.AppendTextColumn("VALUE",
		1, ui.TableModelColumnAlwaysEditable, nil)

	area = ui.NewArea(stw)

	entry := ui.NewEntry()

	grid.Append(table, 0, 0, 1, 2, false, ui.AlignFill, true, ui.AlignFill)
	grid.Append(area, 1, 0, 1, 1, true, ui.AlignFill, true, ui.AlignFill)
	grid.Append(entry, 1, 1, 1, 1, true, ui.AlignFill, false, ui.AlignFill)

	mainwin.Show()
}
