package stlibui

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/andlabs/ui"
	"github.com/yofu/abbrev"
	st "github.com/yofu/st/stlib"
)

var (
	canvaswidth          = 2400
	canvasheight         = 1300
	zoomfactor           = 1.0
	layernum             = 0
	layercbs             = make([]*ui.Checkbox, 0)
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
	blackbrush           = mkSolidBrush(0x000000, 1.0)
	selectfromleftbrush  = mkSolidBrush(0xc83200, 0.3)
	selectfromrightbrush = mkSolidBrush(0x0032c8, 0.3)
	tailbrush            = mkSolidBrush(0xc800c8, 0.3)
	windowzoombrush      = mkSolidBrush(0xc8c800, 0.3)
	dragrectbrush        = selectfromleftbrush
	openingfilename      = ""
	prevkey              *ui.AreaKeyEvent
	cline                string
	DoubleClickCommand   = []string{"TOGGLEBOND", "EDITPLATEELEM"}
	shapedata            st.Shape
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
	if len(tailnodes) == 0 {
		return
	}
	minx := tailnodes[0].Pcoord[0]
	maxx := tailnodes[0].Pcoord[0]
	miny := tailnodes[0].Pcoord[1]
	maxy := tailnodes[0].Pcoord[1]
	for _, n := range tailnodes[1:] {
		if n.Pcoord[0] < minx {
			minx = n.Pcoord[0]
		}
		if n.Pcoord[0] > maxx {
			maxx = n.Pcoord[0]
		}
		if n.Pcoord[1] < miny {
			miny = n.Pcoord[1]
		}
		if n.Pcoord[1] > maxy {
			maxy = n.Pcoord[1]
		}
	}
	stw.currentDrawParam.ClipX = minx
	stw.currentDrawParam.ClipY = miny
	stw.currentDrawParam.ClipWidth = maxx - minx
	stw.currentDrawParam.ClipHeight = maxy - miny
	stw.Redraw()
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
		stw.ClearTypewrite()
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
	shapedata = sh
	var tb *st.TextBox
	if t, tok := stw.textBox["SHAPE"]; tok {
		tb = t
	} else {
		tb = st.NewTextBox(NewFont())
		tb.Show()
		w, h := stw.GetCanvasSize()
		tb.SetPosition(float64(w-250), float64(h-225))
		stw.textBox["SHAPE"] = tb
	}
	var otp bytes.Buffer
	otp.WriteString(fmt.Sprintf("%s\n", sh.String()))
	otp.WriteString(fmt.Sprintf("A   = %10.4f [cm2]\n", sh.A()))
	otp.WriteString(fmt.Sprintf("Asx = %10.4f [cm2]\n", sh.Asx()))
	otp.WriteString(fmt.Sprintf("Asy = %10.4f [cm2]\n", sh.Asy()))
	otp.WriteString(fmt.Sprintf("Ix  = %10.4f [cm4]\n", sh.Ix()))
	otp.WriteString(fmt.Sprintf("Iy  = %10.4f [cm4]\n", sh.Iy()))
	otp.WriteString(fmt.Sprintf("J   = %10.4f [cm4]\n", sh.J()))
	otp.WriteString(fmt.Sprintf("Zx  = %10.4f [cm3]\n", sh.Zx()))
	otp.WriteString(fmt.Sprintf("Zy  = %10.4f [cm3]\n", sh.Zy()))
	tb.SetText(strings.Split(otp.String(), "\n"))
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
	var tb *st.TextBox
	if t, tok := stw.textBox["SECTION"]; tok {
		tb = t
	} else {
		tb = st.NewTextBox(NewFont())
		tb.Show()
		w, h := stw.GetCanvasSize()
		tb.SetPosition(float64(w-250), float64(h-500))
		stw.textBox["SECTION"] = tb
	}
	tb.SetText(strings.Split(sec.InpString(), "\n"))
	if al := sec.Allow; al != nil {
		tb.AddText(strings.Split(al.String(), "\n")...)
	}
	tb.ScrollToTop()
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
	return int(stw.currentDrawParam.AreaWidth), int(stw.currentDrawParam.AreaHeight)
}

func (stw *Window) Changed(c bool) {
	stw.changed = c
}

func (stw *Window) IsChanged() bool {
	return stw.changed
}

func (stw *Window) Redraw() {
	drawall = true
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
	drawall   = true
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
		if showprintrange {
			pw, ph, err := stw.CanvasPaperSize()
			if err == nil {
				path := ui.DrawNewPath(ui.DrawFillModeWinding)
				brush := mkSolidBrush(0xffffff, 1.0)
				path.AddRectangle(0.5*(p.AreaWidth-pw), 0.5*(p.AreaHeight-ph), pw, ph)
				path.End()
				p.Context.Fill(path, brush)
				path.Free()
			}
		}
		stw.SetTextboxPosition()
		if shapedata != nil {
			w, h := stw.GetCanvasSize()
			if showprintrange {
				_, sy, _, ph := st.GetClipCoord(stw)
				h = int(ph + sy)
			}
			scale := 5.0
			x0 := float64(w) - 175.0
			y0 := 200.0
			vertices := shapedata.Vertices()
			height := shapedata.Breadth(false)
			stw.textBox["SHAPE"].SetPosition(float64(w-250), float64(h-225)-scale*height*0.5)
			path := ui.DrawNewPath(ui.DrawFillModeWinding)
			path.NewFigure(x0+scale*vertices[0][0], y0+scale*vertices[0][1])
			closed := false
			closeind := 0
			for i := 1; i < len(vertices); i++ {
				if vertices[i] == nil {
					path.LineTo(x0+scale*vertices[0][0], y0+scale*vertices[0][1])
					path.End()
					p.Context.Stroke(path, blackbrush, stw.currentStrokeParam)
					closed = true
					closeind = i + 1
					path = ui.DrawNewPath(ui.DrawFillModeWinding)
					continue
				}
				if closed {
					path.NewFigure(x0+scale*vertices[i][0], y0+scale*vertices[i][1])
					closed = false
				} else {
					path.LineTo(x0+scale*vertices[i][0], y0+scale*vertices[i][1])
				}
			}
			path.LineTo(x0+scale*vertices[closeind][0], y0+scale*vertices[closeind][1])
			path.End()
			p.Context.Stroke(path, blackbrush, stw.currentStrokeParam)
			path.Free()
		}
		st.DrawFrame(stw, stw.frame, stw.frame.Show.ColorMode, true)
		for _, t := range stw.textBox {
			if t.IsHidden(stw.frame.Show) {
				continue
			}
			st.DrawText(stw, t)
		}
		if selectbox[2] != 0.0 && selectbox[3] != 0.0 {
			path := ui.DrawNewPath(ui.DrawFillModeWinding)
			path.AddRectangle(selectbox[0], selectbox[1], selectbox[2], selectbox[3])
			path.End()
			p.Context.Fill(path, dragrectbrush)
			path.Free()
		}
		if tailnodes != nil {
			path := ui.DrawNewPath(ui.DrawFillModeWinding)
			switch len(tailnodes) {
			case 0:
			case 1:
				path.NewFigure(tailnodes[0].Pcoord[0], tailnodes[0].Pcoord[1])
				path.LineTo(endX, endY)
				path.End()
				p.Context.Stroke(path, tailbrush, stw.currentStrokeParam)
				path.Free()
			default:
				path.NewFigure(tailnodes[0].Pcoord[0], tailnodes[0].Pcoord[1])
				for i := 1; i < len(tailnodes); i++ {
					path.LineTo(tailnodes[i].Pcoord[0], tailnodes[i].Pcoord[1])
				}
				path.LineTo(endX, endY)
				path.End()
				p.Context.Fill(path, tailbrush)
				path.Free()
			}
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
			if me.Down == 1 {
				if stw.ElemSelected() {
					els := stw.SelectedElems()
					if els[0].IsLineElem() {
						stw.ExecCommand(DoubleClickCommand[0])
					} else {
						stw.ExecCommand(DoubleClickCommand[1])
					}
				}
			} else if me.Down == 2 {
				stw.ShowCenter()
			}
		}
	} else if me.Up != 0 {
		endX = me.X
		endY = me.Y
		endT = time.Now()
		zoomfactor = 1.0
		drawall = true
		switch me.Up {
		case 1: // left click
			if stw.Executing() {
				stw.SendClick(st.ClickLeft(int(endX), int(endY)))
			}
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
				if stw.Executing() {
					if !picked {
						stw.SendElem(nil)
						a, d, picked := st.PickAxis(stw, int(endX), int(endY))
						if !picked {
							stw.SendAxis(nil)
						} else {
							a.Current = d
							stw.SendAxis(a)
						}
					} else {
						stw.SendModifier(st.Modifier{
							Shift: me.Modifiers&ui.Shift != 0,
						})
						for _, el := range els {
							stw.SendElem(el)
						}
					}
				} else {
					if !picked {
						stw.DeselectElem()
					} else {
						st.MergeSelectElem(stw, els, me.Modifiers&ui.Shift != 0)
						// TODO: double click command
					}
				}
			} else {
				ns, picked := st.PickNode(stw, int(startX), int(startY), int(endX), int(endY))
				if stw.Executing() {
					if !picked {
						stw.SendNode(nil)
					} else {
						stw.SendModifier(st.Modifier{
							Shift: me.Modifiers&ui.Shift != 0,
						})
						for _, n := range ns {
							stw.SendNode(n)
						}
					}
				} else {
					if !picked {
						stw.DeselectNode()
					} else {
						st.MergeSelectNode(stw, ns, me.Modifiers&ui.Shift != 0)
					}
				}
			}
			selectbox[2] = 0.0
			selectbox[3] = 0.0
		case 3: // right click
			if stw.Executing() {
				stw.SendClick(st.ClickRight(int(endX), int(endY)))
			} else {
				if stw.CommandLineString() != "" {
					stw.FeedCommand()
				} else if stw.lastcommand != nil {
					stw.Execute(stw.lastcommand(stw))
				}
			}
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
				if me.Modifiers&ui.Ctrl != 0 {
					factor := dy / 10.0
					val := stw.Zoom(factor-stw.CanvasScaleSpeed()*math.Log2(zoomfactor), startX, startY)
					zoomfactor *= val
				} else if me.Modifiers&ui.Shift != 0 {
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
		if tailnodes != nil {
			stw.SendPosition(int(dx), int(dy))
			stw.TailLine()
		}
	}
}

func (stw *Window) MouseCrossed(a *ui.Area, left bool) {
	// do nothing
}

func (stw *Window) DragBroken(a *ui.Area) {
	// do nothing
}

func (stw *Window) escape() {
	stw.QuitCommand()
	stw.ClearCommandLine()
	stw.ClearTypewrite()
	stw.Deselect()
	stw.Redraw()
}

func (stw *Window) KeyEvent(a *ui.Area, ke *ui.AreaKeyEvent) (handled bool) {
	if ke.Up {
		return false
	}
	// fmt.Println(ke.ExtKey, ke.Key, ke.Modifier, ke.Modifiers)
	if stw.Executing() {
		if ke.ExtKey == ui.Escape {
			stw.escape()
		}
		// TODO: send key
	}
	if ke.ExtKey != 0 {
		switch ke.ExtKey {
		default:
			return false
		case ui.Escape:
			stw.escape()
			return true
		}
	} else if ke.Key != 0 {
		setprev := true
		typing := true
		switch ke.Key {
		default:
			if ke.Modifiers&ui.Shift != 0 {
				stw.TypeCommandLine(strings.ToUpper(string(ke.Key)))
			} else {
				stw.TypeCommandLine(string(ke.Key))
			}
		case 10: // Enter
			stw.FeedCommand()
		case 8: // Backspace
			stw.BackspaceCommandLine()
		case 32: // Space bar
			stw.EndCompletion()
			cl := stw.CommandLineString()
			if !stw.AtLast() {
				stw.TypeCommandLine(" ")
			} else if strings.Contains(cl, " ") {
				if lis, ok := stw.ContextComplete(); ok {
					cls := strings.Split(cl, " ")
					cls[len(cls)-1] = lis[0] + " "
					stw.SetCommandLineString(strings.Join(cls, " "))
				} else {
					stw.TypeCommandLine(" ")
				}
			} else if strings.HasPrefix(cl, ":") {
				c, bang, usage, comp := st.ExModeComplete(cl)
				var b, u string
				if bang {
					b = "!"
				} else {
					b = ""
				}
				if usage {
					u = "?"
				} else {
					u = ""
				}
				if comp != nil {
					str := fmt.Sprintf(":%s%s%s ", c, b, u)
					stw.SetCommandLineString(str)
					comp.Chdir(stw.Cwd())
					stw.SetComplete(comp)
					stw.History(comp.String())
				} else {
					stw.TypeCommandLine(" ")
				}
			} else if strings.HasPrefix(cl, "'") {
				c, usage, comp := st.Fig2KeywordComplete(cl)
				var u string
				if usage {
					u = "?"
				} else {
					u = ""
				}
				if comp != nil {
					str := fmt.Sprintf("'%s%s ", c, u)
					stw.SetCommandLineString(str)
					comp.Chdir(stw.Cwd())
					stw.SetComplete(comp)
					stw.History(comp.String())
				} else {
					stw.TypeCommandLine(" ")
				}
			} else {
				stw.TypeCommandLine(" ")
			}
		case 9: // Tab
			if prevkey.Key == 9 {
				if ke.Modifiers&ui.Shift != 0 {
					stw.PrevComplete()
				} else {
					stw.NextComplete()
				}
			} else {
				stw.Complete()
			}
		case '`':
			if ke.Modifiers&ui.Alt != 0 {
				return false
			} else if ke.Modifiers&ui.Shift != 0 {
				stw.TypeCommandLine("~")
			} else {
				stw.TypeCommandLine(string(ke.Key))
			}
		case '1':
			if ke.Modifiers&ui.Shift != 0 {
				stw.TypeCommandLine("!")
			} else {
				stw.TypeCommandLine(string(ke.Key))
			}
		case '2':
			if ke.Modifiers&ui.Shift != 0 {
				stw.TypeCommandLine("@")
			} else {
				stw.TypeCommandLine(string(ke.Key))
			}
		case '3':
			if ke.Modifiers&ui.Shift != 0 {
				stw.TypeCommandLine("#")
			} else {
				stw.TypeCommandLine(string(ke.Key))
			}
		case '4':
			if ke.Modifiers&ui.Shift != 0 {
				stw.TypeCommandLine("$")
			} else {
				stw.TypeCommandLine(string(ke.Key))
			}
		case '5':
			if ke.Modifiers&ui.Shift != 0 {
				stw.TypeCommandLine("%")
			} else {
				stw.TypeCommandLine(string(ke.Key))
			}
		case '6':
			if ke.Modifiers&ui.Shift != 0 {
				stw.TypeCommandLine("^")
			} else {
				stw.TypeCommandLine(string(ke.Key))
			}
		case '7':
			if ke.Modifiers&ui.Shift != 0 {
				stw.TypeCommandLine("&")
			} else {
				stw.TypeCommandLine(string(ke.Key))
			}
		case '8':
			if ke.Modifiers&ui.Shift != 0 {
				stw.TypeCommandLine("*")
			} else {
				stw.TypeCommandLine(string(ke.Key))
			}
		case '9':
			if ke.Modifiers&ui.Shift != 0 {
				stw.TypeCommandLine("(")
			} else {
				stw.TypeCommandLine(string(ke.Key))
			}
		case '0':
			if ke.Modifiers&ui.Shift != 0 {
				stw.TypeCommandLine(")")
			} else {
				stw.TypeCommandLine(string(ke.Key))
			}
		case '-':
			if ke.Modifiers&ui.Shift != 0 {
				stw.TypeCommandLine("_")
			} else {
				stw.TypeCommandLine(string(ke.Key))
			}
		case '=':
			if ke.Modifiers&ui.Shift != 0 {
				stw.TypeCommandLine("+")
			} else {
				stw.TypeCommandLine(string(ke.Key))
			}
		case ';':
			if ke.Modifiers&ui.Shift != 0 {
				stw.TypeCommandLine(";")
			} else {
				stw.TypeCommandLine(":")
			}
		case '\'':
			if ke.Modifiers&ui.Shift != 0 {
				stw.TypeCommandLine("\"")
			} else {
				stw.TypeCommandLine(string(ke.Key))
			}
		case '[':
			if ke.Modifiers&ui.Shift != 0 {
				stw.TypeCommandLine("{")
			} else {
				stw.TypeCommandLine(string(ke.Key))
			}
		case ']':
			if ke.Modifiers&ui.Shift != 0 {
				stw.TypeCommandLine("}")
			} else {
				stw.TypeCommandLine(string(ke.Key))
			}
		case ',':
			if ke.Modifiers&ui.Shift != 0 {
				stw.TypeCommandLine("<")
			} else {
				stw.TypeCommandLine(string(ke.Key))
			}
		case '.':
			if ke.Modifiers&ui.Shift != 0 {
				stw.TypeCommandLine(">")
			} else {
				stw.TypeCommandLine(string(ke.Key))
			}
		case '/':
			if ke.Modifiers&ui.Shift != 0 {
				stw.TypeCommandLine("?")
			} else {
				stw.TypeCommandLine(string(ke.Key))
			}
		case 'a':
			if ke.Modifiers&ui.Ctrl != 0 {
				st.SelectNotHidden(stw)
				stw.Redraw()
				typing = false
			} else if ke.Modifiers&ui.Shift != 0 {
				stw.TypeCommandLine(strings.ToUpper(string(ke.Key)))
			} else {
				stw.TypeCommandLine(string(ke.Key))
			}
		case 'd':
			if ke.Modifiers&ui.Ctrl != 0 {
				st.HideNotSelected(stw)
				stw.Redraw()
				typing = false
			} else if ke.Modifiers&ui.Shift != 0 {
				stw.TypeCommandLine(strings.ToUpper(string(ke.Key)))
			} else {
				stw.TypeCommandLine(string(ke.Key))
			}
		case 'n':
			if ke.Modifiers&ui.Ctrl != 0 {
				if !((prevkey.Key == 'p' || prevkey.Key == 'n') && prevkey.Modifiers&ui.Ctrl != 0) {
					cline = stw.CommandLineString()
				}
				stw.NextCommandHistory(cline)
			} else if ke.Modifiers&ui.Shift != 0 {
				stw.TypeCommandLine(strings.ToUpper(string(ke.Key)))
			} else {
				stw.TypeCommandLine(string(ke.Key))
			}
		case 'p':
			if ke.Modifiers&ui.Ctrl != 0 {
				if !((prevkey.Key == 'p' || prevkey.Key == 'n') && prevkey.Modifiers&ui.Ctrl != 0) {
					cline = stw.CommandLineString()
				}
				stw.PrevCommandHistory(cline)
			} else if ke.Modifiers&ui.Shift != 0 {
				stw.TypeCommandLine(strings.ToUpper(string(ke.Key)))
			} else {
				stw.TypeCommandLine(string(ke.Key))
			}
		case 'r':
			if ke.Modifiers&ui.Ctrl != 0 {
				st.ReadAll(stw)
			} else if ke.Modifiers&ui.Shift != 0 {
				stw.TypeCommandLine(strings.ToUpper(string(ke.Key)))
			} else {
				stw.TypeCommandLine(string(ke.Key))
			}
		case 's':
			if ke.Modifiers&ui.Ctrl != 0 {
				stw.ShowAll()
			} else if ke.Modifiers&ui.Shift != 0 {
				stw.TypeCommandLine(strings.ToUpper(string(ke.Key)))
			} else {
				stw.TypeCommandLine(string(ke.Key))
			}
		case 'y':
			if ke.Modifiers&ui.Ctrl != 0 {
				f, err := stw.Redo()
				if err != nil {
					st.ErrorMessage(stw, err, st.ERROR)
				} else {
					stw.frame = f
				}
				stw.Redraw()
				typing = false
			} else if ke.Modifiers&ui.Shift != 0 {
				stw.TypeCommandLine(strings.ToUpper(string(ke.Key)))
			} else {
				stw.TypeCommandLine(string(ke.Key))
			}
		case 'z':
			if ke.Modifiers&ui.Ctrl != 0 {
				f, err := stw.Undo()
				if err != nil {
					st.ErrorMessage(stw, err, st.ERROR)
				} else {
					stw.frame = f
				}
				stw.Redraw()
				typing = false
			} else if ke.Modifiers&ui.Shift != 0 {
				stw.TypeCommandLine(strings.ToUpper(string(ke.Key)))
			} else {
				stw.TypeCommandLine(string(ke.Key))
			}
		}
		if typing {
			stw.Typewrite(stw.CommandLineString())
		}
		if setprev {
			prevkey = ke
		}
		return true
	} else {
		return false
	}
}

func (stw *Window) ClearTypewrite() {
	entry.SetText("")
}

func (stw *Window) Typewrite(str string) {
	if str == "" {
		entry.SetText("")
		return
	}
	t := entry.Text()
	if !strings.HasPrefix(str, t) && !strings.HasPrefix(t, str) {
		return
	}
	entry.SetText(str)
}

func (stw *Window) ShowAll() {
	fmt.Println("ShowAll")
	st.ShowAllSection(stw)
	for _, c := range layercbs {
		c.SetChecked(true)
	}
	stw.Redraw()
}

func SetupLayerTab(stw *Window) *ui.Box {
	leftarea := ui.NewVerticalBox()

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

	leftarea.Append(ui.NewHorizontalSeparator(), false)

	layercbs = make([]*ui.Checkbox, 0)
	cb := ui.NewCheckbox("All Section")
	cb.OnToggled(func(c *ui.Checkbox) {
		if c.Checked() {
			stw.ShowAll()
		} else {
			st.HideAllSection(stw)
			for _, c := range layercbs {
				c.SetChecked(false)
			}
			centerarea.QueueRedrawAll()
		}
	})
	cb.SetChecked(true)
	leftarea.Append(cb, false)

	leftarea.Append(ui.NewHorizontalSeparator(), false)

	return leftarea
}

func AddLayer(stw *Window) {
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
		layernum++
	}
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
	entry.OnChanged(func(en *ui.Entry) {
		stw.SetCommandLineString(en.Text())
	})

	lefttitle = ui.NewCombobox()
	lefttitle.Append("LAYER")
	lefttitle.SetSelected(0)

	centertitle := ui.NewHorizontalBox()
	button := ui.NewButton("Open File")
	button.OnClicked(func(*ui.Button) {
		filename := ui.OpenFile(mainwin)
		if filename == "" {
			return
		}
		err := st.OpenFile(stw, filename, false)
		if err != nil {
			stw.History(err.Error())
		}
	})
	centertitle.Append(button, true)

	grid.Append(centertitle, 0, 0, 1, 1, false, ui.AlignFill, false, ui.AlignFill)
	grid.Append(centerarea, 0, 1, 1, 1, true, ui.AlignFill, true, ui.AlignFill)
	grid.Append(entry, 0, 2, 1, 1, true, ui.AlignFill, false, ui.AlignFill)

	grid.Append(lefttitle, 1, 0, 1, 1, false, ui.AlignFill, false, ui.AlignFill)
	grid.Append(leftarea, 1, 1, 1, 2, false, ui.AlignFill, true, ui.AlignFill)

	mainwin.Show()
}

func (stw *Window) UpdateWindow(fn string) {
	openingfilename = fn
	stw.window.SetTitle(fn)
	for i := 0; i < layernum; i++ {
		leftarea.Delete(8)
	}
	layernum = 0
	layercbs = make([]*ui.Checkbox, 0)
	AddLayer(stw)
}
