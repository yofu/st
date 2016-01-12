package stgxui

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/google/gxui"
	gxmath "github.com/google/gxui/math"
	"github.com/yofu/st/stlib"
	"github.com/yofu/st/stsvg"
	"log"
	"path/filepath"
	"math"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// Constants & Variables
// General
var (
	aliases            map[string]*Command
	sectionaliases     map[int]string
	Inps               []string
	SearchingInps      bool
	SearchingInpsDone  chan bool
	DoubleClickCommand = []string{"TOGGLEBOND", "EDITPLATEELEM"}
	comhistpos         int
	undopos            int
	completepos        int
	completes          []string
	prevkey            int
	clineinput         string
	logf               os.File
	logger             *log.Logger
)
const (
	nRecentFiles = 3
	nUndo        = 10
)
var (
	gopath          = os.Getenv("GOPATH")
	home            = os.Getenv("HOME")
	releasenote     = filepath.Join(home, ".st/help/releasenote.html")
	tooldir         = filepath.Join(home, ".st/tool")
	pgpfile         = filepath.Join(home, ".st/st.pgp")
	recentfn        = filepath.Join(home, ".st/recent.dat")
	historyfn       = filepath.Join(home, ".st/history.dat")
	NOUNDO          = false
	ALTSELECTNODE   = true
)

const LOGFILE = "_st.log"

const ResourceFileName = ".strc"

const (
	repeatcommand      = 0.2 // sec
	CommandHistorySize = 100
)

var (
	axrn_minmax   = regexp.MustCompile("([+-]?[-0-9.]+)<=?([XYZxyz]{1})<=?([+-]?[-0-9.]+)")
	axrn_min1     = regexp.MustCompile("([+-]?[-0-9.]+)<=?([XYZxyz]{1})")
	axrn_min2     = regexp.MustCompile("([XYZxyz]{1})>=?([+-]?[-0-9.]+)")
	axrn_max1     = regexp.MustCompile("([+-]?[-0-9.]+)>=?([XYZxyz]{1})")
	axrn_max2     = regexp.MustCompile("([XYZxyz]{1})<=?([+-]?[-0-9.]+)")
	axrn_eq       = regexp.MustCompile("([XYZxyz]{1})=([+-]?[-0-9.]+)")
	re_etype      = regexp.MustCompile("(?i)^ *et(y(pe?)?)? *={0,2} *([a-zA-Z]+)")
	re_column     = regexp.MustCompile("(?i)co(l(u(m(n)?)?)?)?$")
	re_girder     = regexp.MustCompile("(?i)gi(r(d(e(r)?)?)?)?$")
	re_brace      = regexp.MustCompile("(?i)br(a(c(e)?)?)?$")
	re_wall       = regexp.MustCompile("(?i)wa(l){0,2}$")
	re_slab       = regexp.MustCompile("(?i)sl(a(b)?)?$")
	re_sectnum    = regexp.MustCompile("(?i)^ *sect? *={0,2} *(range[(]{1}){0,1}[[]?([0-9, ]+)[]]?")
	re_orgsectnum = regexp.MustCompile("(?i)^ *osect? *={0,2} *[[]?([0-9, ]+)[]]?")
)

var (
	first                   = 1
	fixRotate               = false
	fixMove                 = false
	deg10                   = 10.0 * math.Pi / 180.0
	drawpivot               = false
)

var (
	showprintrange        = false
)

// Paper Size
const (
	A4_TATE = iota
	A4_YOKO
	A3_TATE
	A3_YOKO
)

const (
	CanvasRotateSpeedX = 0.01
	CanvasRotateSpeedY = 0.01
	CanvasMoveSpeedX   = 0.05
	CanvasMoveSpeedY   = 0.05
	CanvasScaleSpeed   = 500
)
var (
	CanvasFitScale     = 0.9
	CanvasAnimateSpeed    = 0.02
)


var selectDirection = 0

const nodeSelectPixel = 15
const dotSelectPixel = 5
const (
	SD_FROMLEFT = iota
	SD_FROMRIGHT
)

var (
	EPS = 1e-4
)

var (
	STLOGO = &TextBox{
		value:    []string{"         software", "     forstructural", "   analysisthename", "  ofwhichstandsfor", "", " sigmatau  stress", "structure  steel", "andsometh  ing", " likethat"},
		Position: []int{100, 100},
		Angle:    0.0,
		Font:     NewFont(),
		hide:     false,
	}
)

type Window struct { // {{{
	Home string
	cwd  string

	Frame *st.Frame

	driver  gxui.Driver
	theme   gxui.Theme
	dlg     gxui.Window
	draw    gxui.Image
	rubber  gxui.Canvas
	cline   gxui.TextBox
	history gxui.TextBox

	CanvasSize []int // width, height

	selectNode []*st.Node
	selectElem []*st.Elem

	PageTitle *TextBox
	Title     *TextBox
	Text      *TextBox
	textBox   map[string]*TextBox

	papersize uint

	Version  string
	Modified string

	startX int
	startY int
	endX   int
	endY   int

	lastcommand     *Command
	lastexcommand   string
	lastfig2command string

	Labels map[string]gxui.Label

	InpModified bool
	Changed     bool

	comhist     []string
	recentfiles []string
	undostack   []*st.Frame
	taggedFrame map[string]*st.Frame
}

// }}}

func (stw *Window) sideBar() gxui.PanelHolder {
	label := func(text string) gxui.Label {
		label := stw.theme.CreateLabel()
		label.SetText(text)
		return label
	}
	lblbx := func(text string) gxui.LinearLayout {
		label := stw.theme.CreateLabel()
		tbox := stw.theme.CreateTextBox()
		label.SetText(text)
		layout := stw.theme.CreateLinearLayout()
		layout.SetDirection(gxui.LeftToRight)
		layout.AddChild(label)
		layout.AddChild(tbox)
		return layout
	}
	holder := stw.theme.CreatePanelHolder()
	vpanel := stw.theme.CreateLinearLayout()
	vpanel.AddChild(label("VIEW"))
	vpanel.AddChild(lblbx("  GFACT"))
	vpanel.AddChild(label("  PERSPECTIVE"))
	vpanel.AddChild(label("DISTS"))
	vpanel.AddChild(lblbx("  R"))
	vpanel.AddChild(lblbx("  L"))
	vpanel.AddChild(label("ANGLE"))
	vpanel.AddChild(lblbx("  PHI"))
	vpanel.AddChild(lblbx("  THETA"))
	vpanel.AddChild(label("FOCUS"))
	vpanel.AddChild(lblbx("  X"))
	vpanel.AddChild(lblbx("  Y"))
	vpanel.AddChild(lblbx("  Z"))
	scrollvp := stw.theme.CreateScrollLayout()
	scrollvp.SetChild(vpanel)
	holder.AddPanel(scrollvp, "View")
	holder.AddPanel(label("Show"), "Show")
	holder.AddPanel(label("Property"), "Property")
	return holder
}

func (stw *Window) initHistoryArea() {
	stw.history = stw.theme.CreateTextBox()
	stw.history.SetMultiline(true)
	// stw.history.SetFocusable(false)
	stw.history.SetDesiredWidth(800)
}

func (stw *Window) initCommandLineArea() {
	stw.cline = stw.theme.CreateTextBox()
	stw.cline.OnKeyDown(func (ev gxui.KeyboardEvent) {
		switch ev.Key {
		default:
			break
		case gxui.KeyEscape:
			stw.cline.SetText("")
		case gxui.KeyEnter:
			stw.feedCommand()
		// case gxui.KeySemicolon:
		// 	val := stw.cline.Text()
		// 	if ev.Modifier.Shift() {
		// 		if val == "" {
		// 			stw.cline.SetText(";")
		// 		} else {
		// 			stw.cline.SetText(":")
		// 		}
		// 	} else {
		// 		if val == "" {
		// 			stw.cline.SetText(":")
		// 		} else {
		// 			stw.cline.SetText(";")
		// 		}
		// 	}
		}
	})
	stw.cline.SetDesiredWidth(800)
}

func (stw *Window) initDrawAreaCallback() {
	stw.draw.OnMouseUp(func (ev gxui.MouseEvent) {
		if stw.Frame != nil {
			switch ev.Button {
			case gxui.MouseButtonLeft:
				if ev.Modifier.Alt() {
					stw.SelectNodeUp(ev)
				} else {
					stw.SelectElemUp(ev)
				}
			}
		}
		stw.rubber = stw.driver.CreateCanvas(gxmath.Size{W: stw.CanvasSize[0], H: stw.CanvasSize[1]})
		stw.rubber.Complete()
		stw.Redraw()
	})
	stw.draw.OnMouseDown(func (ev gxui.MouseEvent) {
		stw.StartSelection(ev)
	})
	stw.draw.OnDoubleClick(func (ev gxui.MouseEvent) {
		switch ev.Button {
		case gxui.MouseButtonLeft:
			fmt.Println("DOUBLE: LEFT", ev.Point.X, ev.Point.Y)
		case gxui.MouseButtonMiddle:
			if stw.Frame != nil {
				stw.Frame.SetFocus(nil)
				stw.RedrawNode()
				stw.ShowCenter()
			}
		}
	})
	stw.draw.OnMouseMove(func (ev gxui.MouseEvent) {
		if stw.Frame != nil {
			if ev.State.IsDown(gxui.MouseButtonLeft) {
				if ev.Modifier.Alt() {
					stw.SelectNodeMotion(ev)
				} else {
					stw.SelectElemMotion(ev)
				}
				stw.RedrawNode()
			} else if ev.State.IsDown(gxui.MouseButtonMiddle) {
				stw.MoveOrRotate(ev)
				stw.RedrawNode()
			}
		}
	})
	stw.draw.OnMouseScroll(func (ev gxui.MouseEvent) {
		if stw.Frame != nil {
			val := math.Pow(2.0, float64(ev.ScrollY)/CanvasScaleSpeed)
			stw.Frame.View.Center[0] += (val - 1.0) * (stw.Frame.View.Center[0] - float64(ev.Point.X))
			stw.Frame.View.Center[1] += (val - 1.0) * (stw.Frame.View.Center[1] - float64(ev.Point.Y))
			if stw.Frame.View.Perspective {
				stw.Frame.View.Dists[1] *= val
				if stw.Frame.View.Dists[1] < 0.0 {
					stw.Frame.View.Dists[1] = 0.0
				}
			} else {
				stw.Frame.View.Gfact *= val
				if stw.Frame.View.Gfact < 0.0 {
					stw.Frame.View.Gfact = 0.0
				}
			}
			stw.Redraw()
		}
	})
}

func NewWindow(driver gxui.Driver, theme gxui.Theme, homedir string) *Window {
	stw := new(Window)

	stw.Home = homedir
	stw.cwd = homedir
	stw.selectNode = make([]*st.Node, 0)
	stw.selectElem = make([]*st.Elem, 0)

	stw.driver = driver
	stw.theme = theme
	stw.CanvasSize = []int{1000, 1000}

	sidedraw := theme.CreateSplitterLayout()
	sidedraw.SetOrientation(gxui.Horizontal)

	side := stw.sideBar()

	stw.draw = theme.CreateImage()
	stw.initDrawAreaCallback()

	sidedraw.AddChild(side)
	sidedraw.AddChild(stw.draw)
	sidedraw.SetChildWeight(side, 0.2)
	sidedraw.SetChildWeight(stw.draw, 0.8)

	stw.initHistoryArea()
	stw.initCommandLineArea()

	vll := theme.CreateLinearLayout()
	vll.SetDirection(gxui.TopToBottom)
	vll.SetSizeMode(gxui.ExpandToContent)
	vll.AddChild(stw.history)
	vll.AddChild(stw.cline)

	vsp := theme.CreateSplitterLayout()
	vsp.SetOrientation(gxui.Vertical)
	vsp.AddChild(sidedraw)
	vsp.AddChild(vll)
	vsp.SetChildWeight(sidedraw, 0.8)
	vsp.SetChildWeight(vll, 0.2)

	stw.dlg = theme.CreateWindow(1200, 900, "stx")
	stw.dlg.AddChild(vsp)
	stw.dlg.OnClose(driver.Terminate)
	stw.dlg.OnKeyDown(func (ev gxui.KeyboardEvent) {
		if _, ok := stw.dlg.Focus().(gxui.TextBox); ok {
			return
		}
		switch ev.Key {
		default:
			stw.dlg.SetFocus(stw.cline)
			return
		case gxui.KeyEscape:
			stw.Deselect()
		case gxui.KeyLeftShift, gxui.KeyRightShift, gxui.KeyLeftControl, gxui.KeyRightControl, gxui.KeyLeftAlt, gxui.KeyRightAlt:
			return
		case gxui.KeyA:
			if ev.Modifier.Control() {
				stw.SelectNotHidden()
			}
		}
		stw.Redraw()
	})

	stw.SetCanvasSize()

	stw.Changed = false
	stw.comhist = make([]string, CommandHistorySize)
	comhistpos = -1
	stw.recentfiles = make([]string, nRecentFiles)
	stw.SetRecently()
	stw.undostack = make([]*st.Frame, nUndo)
	stw.taggedFrame = make(map[string]*st.Frame)
	undopos = 0
	StartLogging()

	return stw
}

func (stw *Window) Snapshot() {
	stw.Changed = true
	if NOUNDO {
		return
	}
	tmp := make([]*st.Frame, nUndo)
	tmp[0] = stw.Frame.Snapshot()
	for i := 0; i < nUndo-1-undopos; i++ {
		tmp[i+1] = stw.undostack[i+undopos]
	}
	stw.undostack = tmp
	undopos = 0
}

func (stw *Window) Bbox() (xmin, xmax, ymin, ymax float64) {
	if stw.Frame == nil || len(stw.Frame.Nodes) == 0 {
		return 0.0, 0.0, 0.0, 0.0
	}
	var mins, maxs [2]float64
	first := true
	for _, j := range stw.Frame.Nodes {
		if j.IsHidden(stw.Frame.Show) {
			continue
		}
		if first {
			for k := 0; k < 2; k++ {
				mins[k] = j.Pcoord[k]
				maxs[k] = j.Pcoord[k]
			}
			first = false
		} else {
			for k := 0; k < 2; k++ {
				if j.Pcoord[k] < mins[k] {
					mins[k] = j.Pcoord[k]
				} else if maxs[k] < j.Pcoord[k] {
					maxs[k] = j.Pcoord[k]
				}
			}
		}
	}
	return mins[0], maxs[0], mins[1], maxs[1]
}

func (stw *Window) SetCanvasSize() {
	size := stw.draw.Size()
	stw.CanvasSize[0] = size.W
	stw.CanvasSize[1] = size.H
}

func (stw *Window) Animate(view *st.View) {
	scale := 1.0
	if stw.Frame.View.Perspective {
		scale = math.Pow(view.Dists[1] / stw.Frame.View.Dists[1], CanvasAnimateSpeed)
	} else {
		scale = math.Pow(view.Gfact / stw.Frame.View.Gfact, CanvasAnimateSpeed)
	}
	center := make([]float64, 2)
	angle := make([]float64, 2)
	focus := make([]float64, 3)
	for i:=0; i<3; i++ {
		focus[i] = CanvasAnimateSpeed*(view.Focus[i] - stw.Frame.View.Focus[i])
		if i >= 2 {
			break
		}
		center[i] = CanvasAnimateSpeed*(view.Center[i] - stw.Frame.View.Center[i])
		angle[i] = view.Angle[i] - stw.Frame.View.Angle[i]
		if i == 1 {
			for {
				if angle[1] <= 180.0 {
					break
				}
				angle[1] -= 360.0
			}
			for {
				if angle[1] >= -180.0 {
					break
				}
				angle[1] += 360.0
			}
		}
		angle[i] *= CanvasAnimateSpeed
	}
	for i:=0; i<int(1/CanvasAnimateSpeed); i++ {
		if stw.Frame.View.Perspective {
			stw.Frame.View.Dists[1] *= scale
		} else {
			stw.Frame.View.Gfact *= scale
		}
		for j:=0; j<3; j++ {
			stw.Frame.View.Focus[j] += focus[j]
			if j >= 2 {
				break
			}
			stw.Frame.View.Center[j] += center[j]
			stw.Frame.View.Angle[j] += angle[j]
		}
		stw.RedrawNode() // TODO: not working
	}
}

func (stw *Window) CanvasCenterView(angle []float64) *st.View {
	a0 := make([]float64, 2)
	f0 := make([]float64, 3)
	focus := make([]float64, 3)
	for i:=0; i<3; i++ {
		f0[i] = stw.Frame.View.Focus[i]
		if i >= 2 {
			break
		}
		a0[i] = stw.Frame.View.Angle[i]
		stw.Frame.View.Angle[i] = angle[i]
	}
	stw.Frame.SetFocus(nil)
	stw.Frame.View.Set(0)
	for _, n := range stw.Frame.Nodes {
		stw.Frame.View.ProjectNode(n)
	}
	xmin, xmax, ymin, ymax := stw.Bbox()
	stw.SetCanvasSize()
	for i:=0; i<3; i++ {
		focus[i] = stw.Frame.View.Focus[i]
		stw.Frame.View.Focus[i] = f0[i]
		if i >= 2 {
			break
		}
		stw.Frame.View.Angle[i] = a0[i]
	}
	stw.Frame.View.Set(0)
	if xmax == xmin && ymax == ymin {
		return nil
	}
	view := stw.Frame.View.Copy()
	view.Focus = focus
	scale := math.Min(float64(stw.CanvasSize[0])/(xmax-xmin), float64(stw.CanvasSize[1])/(ymax-ymin)) * CanvasFitScale
	if stw.Frame.View.Perspective {
		view.Dists[1] = stw.Frame.View.Dists[1] * scale
	} else {
		view.Gfact = stw.Frame.View.Gfact * scale
	}
	view.Center[0] = float64(stw.CanvasSize[0])*0.5 + scale*(stw.Frame.View.Center[0]-0.5*(xmax+xmin))
	view.Center[1] = float64(stw.CanvasSize[1])*0.5 + scale*(stw.Frame.View.Center[1]-0.5*(ymax+ymin))
	view.Angle = angle
	return view
}

func (stw *Window) ShowAtCanvasCenter() {
	view := stw.CanvasCenterView(stw.Frame.View.Angle)
	stw.Animate(view)
}

func (stw *Window) ShowCenter() {
	stw.ShowAtCanvasCenter()
	stw.Redraw()
}

func (stw *Window) SetAngle(phi, theta float64) {
	if stw.Frame != nil {
		view := stw.CanvasCenterView([]float64{phi, theta})
		stw.Animate(view)
	}
}

func (stw *Window) MoveOrRotate(ev gxui.MouseEvent) {
	if !fixMove && (ev.Modifier.Shift() || fixRotate) {
		stw.Frame.View.Center[0] += float64(ev.Point.X-stw.startX) * CanvasMoveSpeedX
		stw.Frame.View.Center[1] += float64(ev.Point.Y-stw.startY) * CanvasMoveSpeedY
	} else if !fixRotate {
		stw.Frame.View.Angle[0] += float64(ev.Point.Y-stw.startY) * CanvasRotateSpeedY
		stw.Frame.View.Angle[1] -= float64(ev.Point.X-stw.startX) * CanvasRotateSpeedX
	}
}

func (stw *Window) Save() error {
	return stw.SaveFile(filepath.Join(stw.Home, "hogtxt.inp"))
}

// func (stw *Window) SaveAS(string) error {
func (stw *Window) SaveAS() {
	fn := "hogtxt.inp"
	err := stw.SaveFile(fn)
	if err == nil && fn != stw.Frame.Path {
		stw.Copylsts(fn)
		stw.Rebase(fn)
	}
	// return err
}

func (stw *Window) Copylsts(name string) {
	if stw.Yn("SAVE AS", ".lst, .fig2, .kjnファイルがあればコピーしますか?") {
		for _, ext := range []string{".lst", ".fig2", ".kjn"} {
			src := st.Ce(stw.Frame.Path, ext)
			dst := st.Ce(name, ext)
			if st.FileExists(src) {
				err := st.CopyFile(src, dst)
				if err == nil {
					stw.History(fmt.Sprintf("COPY: %s", dst))
				}
			}
		}
	}
}

func (stw *Window) Rebase(fn string) {
	stw.Frame.Name = filepath.Base(fn)
	stw.Frame.Project = st.ProjectName(fn)
	path, err := filepath.Abs(fn)
	if err != nil {
		stw.Frame.Path = fn
	} else {
		stw.Frame.Path = path
	}
	// stw.dlg.SetAttribute("TITLE", stw.Frame.Name)
	stw.Frame.Home = stw.Home
	stw.AddRecently(fn)
}

func (stw *Window) SaveFile(fn string) error {
	var v *st.View
	if !stw.Frame.View.Perspective {
		v = stw.Frame.View.Copy()
		stw.Frame.View.Gfact = 1.0
		stw.Frame.View.Perspective = true
		for _, n := range stw.Frame.Nodes {
			stw.Frame.View.ProjectNode(n)
		}
		xmin, xmax, ymin, ymax := stw.Bbox()
		stw.SetCanvasSize()
		scale := math.Min(float64(stw.CanvasSize[0])/(xmax-xmin), float64(stw.CanvasSize[1])/(ymax-ymin)) * CanvasFitScale
		stw.Frame.View.Dists[1] *= scale
	}
	err := stw.Frame.WriteInp(fn)
	if v != nil {
		stw.Frame.View = v
	}
	if err != nil {
		return err
	}
	stw.ErrorMessage(errors.New(fmt.Sprintf("SAVE: %s", fn)), st.INFO)
	stw.Changed = false
	return nil
}

func (stw *Window) SaveFileSelected(fn string) error {
	var v *st.View
	els := stw.selectElem
	if !stw.Frame.View.Perspective {
		v = stw.Frame.View.Copy()
		stw.Frame.View.Gfact = 1.0
		stw.Frame.View.Perspective = true
		for _, n := range stw.Frame.Nodes {
			stw.Frame.View.ProjectNode(n)
		}
		xmin, xmax, ymin, ymax := stw.Bbox()
		stw.SetCanvasSize()
		scale := math.Min(float64(stw.CanvasSize[0])/(xmax-xmin), float64(stw.CanvasSize[1])/(ymax-ymin)) * CanvasFitScale
		stw.Frame.View.Dists[1] *= scale
	}
	err := st.WriteInp(fn, stw.Frame.View, stw.Frame.Ai, els)
	if v != nil {
		stw.Frame.View = v
	}
	if err != nil {
		return err
	}
	stw.ErrorMessage(errors.New(fmt.Sprintf("SAVE: %s", fn)), st.INFO)
	stw.Changed = false
	return nil
}

func (stw *Window) Close(force bool) {
	if !force && stw.Changed {
		if stw.Yn("CHANGED", "変更を保存しますか") {
			stw.SaveAS()
		} else {
			return
		}
	}
	stw.dlg.Close()
}

func (stw *Window) Open() {
	name := "hogtxt.inp"
	// if name, ok := iup.GetOpenFile(stw.Cwd, "*.inp"); ok {
	err := stw.OpenFile(name, true)
	if err != nil {
		fmt.Println(err)
	}
	stw.Redraw()
}

func (stw *Window) AddRecently(fn string) error {
	fn = filepath.ToSlash(fn)
	if st.FileExists(recentfn) {
		f, err := os.Open(recentfn)
		if err != nil {
			return err
		}
		stw.recentfiles[0] = fn
		s := bufio.NewScanner(f)
		num := 0
		for s.Scan() {
			if rfn := s.Text(); rfn != fn {
				stw.recentfiles[num+1] = rfn
				num++
			}
			if num >= nRecentFiles-1 {
				break
			}
		}
		f.Close()
		if err := s.Err(); err != nil {
			return err
		}
		w, err := os.Create(recentfn)
		if err != nil {
			return err
		}
		defer w.Close()
		for i := 0; i < nRecentFiles; i++ {
			w.WriteString(fmt.Sprintf("%s\n", stw.recentfiles[i]))
		}
		return nil
	} else {
		w, err := os.Create(recentfn)
		if err != nil {
			return err
		}
		defer w.Close()
		w.WriteString(fmt.Sprintf("%s\n", fn))
		stw.recentfiles[0] = fn
		return nil
	}
}

func (stw *Window) ShowRecently() {
	for i, fn := range stw.recentfiles {
		if fn != "" {
			stw.History(fmt.Sprintf("%d: %s", i, fn))
		}
	}
}

func (stw *Window) SetRecently() error {
	if st.FileExists(recentfn) {
		f, err := os.Open(recentfn)
		if err != nil {
			return err
		}
		s := bufio.NewScanner(f)
		num := 0
		for s.Scan() {
			if fn := s.Text(); fn != "" {
				stw.History(fmt.Sprintf("%d: %s", num, fn))
				stw.recentfiles[num] = fn
				num++
			}
		}
		if err := s.Err(); err != nil {
			return err
		}
		return nil
	} else {
		return errors.New(fmt.Sprintf("OpenRecently: %s doesn't exist", recentfn))
	}
}

func (stw *Window) Reload() {
	if stw.Frame != nil {
		stw.Deselect()
		v := stw.Frame.View
		s := stw.Frame.Show
		stw.OpenFile(stw.Frame.Path, false)
		stw.Frame.View = v
		stw.Frame.Show = s
		stw.Redraw()
	}
}

func (stw *Window) OpenFile(filename string, readrcfile bool) error {
	var err error
	var s *st.Show
	fn := st.ToUtf8string(filename)
	frame := st.NewFrame()
	if stw.Frame != nil {
		s = stw.Frame.Show
	}
	stw.SetCanvasSize()
	frame.View.Center[0] = float64(stw.CanvasSize[0]) * 0.5
	frame.View.Center[1] = float64(stw.CanvasSize[1]) * 0.5
	switch filepath.Ext(fn) {
	case ".inp":
		err = frame.ReadInp(fn, []float64{0.0, 0.0, 0.0}, 0.0, false)
		if err != nil {
			return err
		}
		stw.Frame = frame
	case ".dxf":
		err = frame.ReadDxf(fn, []float64{0.0, 0.0, 0.0}, EPS)
		if err != nil {
			return err
		}
		stw.Frame = frame
		frame.SetFocus(nil)
		stw.DrawFrameNode()
		stw.ShowCenter()
	}
	if s != nil {
		stw.Frame.Show = s
		for snum := range stw.Frame.Sects {
			if _, ok := stw.Frame.Show.Sect[snum]; !ok {
				stw.Frame.Show.Sect[snum] = true
			}
		}
	}
	openstr := fmt.Sprintf("OPEN: %s", fn)
	stw.History(openstr)
	stw.dlg.SetTitle(stw.Frame.Name)
	stw.Frame.Home = stw.Home
	// stw.LinkTextValue()
	stw.cwd = filepath.Dir(fn)
	stw.AddRecently(fn)
	stw.Snapshot()
	stw.Changed = false
	// stw.HideLogo()
	if readrcfile {
		if rcfn := filepath.Join(stw.cwd, ResourceFileName); st.FileExists(rcfn) {
			stw.ReadResource(rcfn)
		}
	}
	return nil
}

func (stw *Window) Read() {
	if stw.Frame != nil {
		name := "hogtxt.otl"
		// if name, ok := iup.GetOpenFile("", ""); ok {
		err := stw.ReadFile(name)
		if err != nil {
			stw.ErrorMessage(err, st.ERROR)
		}
	}
}

func (stw *Window) ReadAll() {
	if stw.Frame != nil {
		var err error
		for _, el := range stw.Frame.Elems {
			switch el.Etype {
			case st.WBRACE, st.SBRACE:
				stw.Frame.DeleteElem(el.Num)
			case st.WALL, st.SLAB:
				el.Children = make([]*st.Elem, 2)
			}
		}
		exts := []string{".inl", ".ihx", ".ihy", ".otl", ".ohx", ".ohy", ".rat2", ".wgt", ".lst", ".kjn"}
		read := make([]string, 10)
		nread := 0
		for _, ext := range exts {
			name := st.Ce(stw.Frame.Path, ext)
			err = stw.ReadFile(name)
			if err != nil {
				if ext == ".rat2" {
					err = stw.ReadFile(st.Ce(stw.Frame.Path, ".rat"))
					if err == nil {
						continue
					}
				}
				stw.ErrorMessage(err, st.ERROR)
			} else {
				read[nread] = ext
				nread++
			}
		}
		stw.History(fmt.Sprintf("READ: %s", strings.Join(read, " ")))
	}
}

func (stw *Window) ReadFile(filename string) error {
	var err error
	switch filepath.Ext(filename) {
	default:
		return errors.New("Unknown Format")
	case ".inp":
		x := 0.0
		y := 0.0
		z := 0.0
		err = stw.Frame.ReadInp(filename, []float64{x, y, z}, 0.0, false)
	case ".inl", ".ihx", ".ihy":
		err = stw.Frame.ReadData(filename)
	case ".otl", ".ohx", ".ohy":
		err = stw.Frame.ReadResult(filename, st.UpdateResult)
	case ".rat", ".rat2":
		err = stw.Frame.ReadRat(filename)
	case ".lst":
		err = stw.Frame.ReadLst(filename)
	case ".wgt":
		err = stw.Frame.ReadWgt(filename)
	case ".kjn":
		err = stw.Frame.ReadKjn(filename)
	case ".otp":
		err = stw.Frame.ReadBuckling(filename)
	case ".otx", ".oty", ".inc":
		err = stw.Frame.ReadZoubun(filename)
	}
	if err != nil {
		return err
	}
	return nil
}

func (stw *Window) AddPropAndSect(filename string) error {
	if stw.Frame != nil {
		err := stw.Frame.AddPropAndSect(filename, true)
		return err
	} else {
		return errors.New("NO FRAME")
	}
}

func (stw *Window) PrintSVG(filename string) error {
	err := stsvg.Print(stw.Frame, filename)
	if err != nil {
		return err
	}
	return nil
}

func (stw *Window) CompleteFileName(str string) string {
	envval := regexp.MustCompile("[$]([a-zA-Z]+)")
	if envval.MatchString(str) {
		efs := envval.FindStringSubmatch(str)
		if len(efs) >= 2 {
			val := os.Getenv(strings.ToUpper(efs[1]))
			if val != "" {
				str = strings.Replace(str, efs[0], val, 1)
			}
		}
	}
	if strings.Contains(str, "%") {
		str = strings.Replace(str, "%:h", stw.cwd, 1)
		if stw.Frame != nil {
			str = strings.Replace(str, "%<", st.PruneExt(stw.Frame.Path), 1)
			str = strings.Replace(str, "%", stw.Frame.Path, 1)
		}
	}
	sharp := regexp.MustCompile("#([0-9]+)")
	if sharp.MatchString(str) {
		sfs := sharp.FindStringSubmatch(str)
		if len(sfs) >= 2 {
			tmp, err := strconv.ParseInt(sfs[1], 10, 64)
			if err == nil && int(tmp) < nRecentFiles {
				str = strings.Replace(str, sfs[0], stw.recentfiles[int(tmp)], 1)
			}
		}
	}
	lis := strings.Split(str, " ")
	path := lis[len(lis)-1]
	if !filepath.IsAbs(path) {
		path = filepath.Join(stw.cwd, path)
	}
	var err error
	tmp, err := filepath.Glob(path + "*")
	if err != nil || len(tmp) == 0 {
		completes = make([]string, 0)
	} else {
		completes = make([]string, len(tmp))
		for i := 0; i < len(tmp); i++ {
			stat, err := os.Stat(tmp[i])
			if err != nil {
				continue
			}
			if stat.IsDir() {
				tmp[i] += string(os.PathSeparator)
			}
			lis[len(lis)-1] = tmp[i]
			completes[i] = strings.Join(lis, " ")
		}
		completepos = 0
		str = completes[0]
	}
	return str
}

func PrevComplete(str string) string {
	if completes == nil || len(completes) == 0 {
		return str
	}
	completepos--
	if completepos < 0 {
		completepos = len(completes) - 1
	}
	return completes[completepos]
}

func NextComplete(str string) string {
	if completes == nil || len(completes) == 0 {
		return str
	}
	completepos++
	if completepos >= len(completes) {
		completepos = 0
	}
	return completes[completepos]
}

func (stw *Window) SearchFile(fn string) (string, error) {
	if _, err := os.Stat(fn); err == nil {
		return fn, nil
	} else {
		pos1 := strings.IndexAny(fn, "0123456789.")
		if pos1 < 0 {
			return fn, errors.New(fmt.Sprintf("File not fount %s", fn))
		}
		pos2 := strings.IndexAny(fn, "_.")
		if pos2 < 0 {
			return fn, errors.New(fmt.Sprintf("File not fount %s", fn))
		}
		cand := filepath.Join(stw.Home, fn[:pos1], fn[:pos2], fn)
		if st.FileExists(cand) {
			return cand, nil
		} else {
			return fn, errors.New(fmt.Sprintf("File not fount %s", fn))
		}
	}
}

func (stw *Window) CurrentLap(comment string, nlap, laps int) {
	var tb *TextBox
	if t, tok := stw.textBox["LAP"]; tok {
		tb = t
	} else {
		tb = NewTextBox()
		tb.hide = false
		tb.Position = []int{30, stw.CanvasSize[1] - 30}
		stw.textBox["LAP"] = tb
	}
	if comment == "" {
		tb.value = []string{fmt.Sprintf("LAP: %3d / %3d", nlap, laps)}
	} else {
		tb.value = []string{comment, fmt.Sprintf("LAP: %3d / %3d", nlap, laps)}
	}
}

func (stw *Window) SectionData(sec *st.Sect) {
	var tb *TextBox
	if t, tok := stw.textBox["SECTION"]; tok {
		tb = t
	} else {
		tb = NewTextBox()
		tb.hide = false
		tb.Position = []int{stw.CanvasSize[0] - 400, stw.CanvasSize[1] - 30}
		stw.textBox["SECTION"] = tb
	}
	tb.value = strings.Split(sec.InpString(), "\n")
	if al, ok := stw.Frame.Allows[sec.Num]; ok {
		tb.value = append(tb.value, strings.Split(al.String(), "\n")...)
	}
}

func SplitNums(nums string) []int {
	sectrange := regexp.MustCompile("(?i)^ *range *[(] *([0-9]+) *, *([0-9]+) *[)] *$")
	if sectrange.MatchString(nums) {
		fs := sectrange.FindStringSubmatch(nums)
		start, err := strconv.ParseInt(fs[1], 10, 64)
		end, err := strconv.ParseInt(fs[2], 10, 64)
		if err != nil {
			return nil
		}
		if start > end {
			return nil
		}
		sects := make([]int, int(end-start))
		for i := 0; i < int(end-start); i++ {
			sects[i] = i + int(start)
		}
		return sects
	} else {
		splitter := regexp.MustCompile("[, ]")
		tmp := splitter.Split(nums, -1)
		rtn := make([]int, len(tmp))
		i := 0
		for _, numstr := range tmp {
			val, err := strconv.ParseInt(strings.Trim(numstr, " "), 10, 64)
			if err != nil {
				continue
			}
			rtn[i] = int(val)
			i++
		}
		return rtn[:i]
	}
}

func SectFilter(str string) (func(*st.Elem) bool, string) {
	var filterfunc func(el *st.Elem) bool
	var hstr string
	var snums []int
	fs := re_sectnum.FindStringSubmatch(str)
	if fs[1] != "" {
		snums = SplitNums(fmt.Sprintf("range(%s)", fs[2]))
	} else {
		snums = SplitNums(fs[2])
	}
	filterfunc = func(el *st.Elem) bool {
		for _, snum := range snums {
			if el.Sect.Num == snum {
				return true
			}
		}
		return false
	}
	hstr = fmt.Sprintf("Sect == %s", fs[1])
	return filterfunc, hstr
}

func OriginalSectFilter(str string) (func(*st.Elem) bool, string) {
	var filterfunc func(el *st.Elem) bool
	var hstr string
	fs := re_orgsectnum.FindStringSubmatch(str)
	if len(fs) < 2 {
		return nil, ""
	}
	snums := SplitNums(fs[1])
	filterfunc = func(el *st.Elem) bool {
		if el.Etype != st.WBRACE && el.Etype != st.SBRACE {
			return false
		}
		for _, snum := range snums {
			if el.OriginalSection().Num == snum {
				return true
			}
		}
		return false
	}
	hstr = fmt.Sprintf("Sect == %s", fs[1])
	return filterfunc, hstr
}

func EtypeFilter(str string) (func(*st.Elem) bool, string) {
	var filterfunc func(el *st.Elem) bool
	var hstr string
	fs := re_etype.FindStringSubmatch(str)
	l := len(fs)
	if l >= 4 {
		var val int
		switch {
		case re_column.MatchString(fs[l-1]):
			val = st.COLUMN
		case re_girder.MatchString(fs[l-1]):
			val = st.GIRDER
		case re_brace.MatchString(fs[l-1]):
			val = st.BRACE
		case re_wall.MatchString(fs[l-1]):
			val = st.WALL
		case re_slab.MatchString(fs[l-1]):
			val = st.SLAB
		}
		filterfunc = func(el *st.Elem) bool {
			return el.Etype == val
		}
		hstr = fmt.Sprintf("Etype == %s", st.ETYPES[val])
	}
	return filterfunc, hstr
}

func (stw *Window) FilterElem(els []*st.Elem, str string) ([]*st.Elem, error) {
	l := len(els)
	if els == nil || l == 0 {
		return nil, errors.New("number of input elems is zero")
	}
	parallel := regexp.MustCompile("(?i)^ *// *([xyz]{1})")
	ortho := regexp.MustCompile("^ *TT *([xyzXYZ]{1})")
	onplane := regexp.MustCompile("(?i)^ *on *([xyz]{2})")
	adjoin := regexp.MustCompile("^ *ad(j(o(in?)?)?)? (.*)")
	var filterfunc func(el *st.Elem) bool
	var hstr string
	switch {
	case parallel.MatchString(str):
		var axis []float64
		fs := parallel.FindStringSubmatch(str)
		if len(fs) < 2 {
			break
		}
		tmp := strings.ToUpper(fs[1])
		axes := [][]float64{st.XAXIS, st.YAXIS, st.ZAXIS}
		for i, val := range []string{"X", "Y", "Z"} {
			if tmp == val {
				axis = axes[i]
				break
			}
		}
		filterfunc = func(el *st.Elem) bool {
			return el.IsParallel(axis, 1e-4)
		}
		hstr = fmt.Sprintf("Parallel to %sAXIS", tmp)
	case ortho.MatchString(str):
		var axis []float64
		fs := ortho.FindStringSubmatch(str)
		if len(fs) < 2 {
			break
		}
		tmp := strings.ToUpper(fs[1])
		axes := [][]float64{st.XAXIS, st.YAXIS, st.ZAXIS}
		for i, val := range []string{"X", "Y", "Z"} {
			if tmp == val {
				axis = axes[i]
				break
			}
		}
		filterfunc = func(el *st.Elem) bool {
			return el.IsOrthogonal(axis, 1e-4)
		}
		hstr = fmt.Sprintf("Orthogonal to %sAXIS", tmp)
	case onplane.MatchString(str):
		var axis int
		fs := onplane.FindStringSubmatch(str)
		if len(fs) < 2 {
			break
		}
		tmp := strings.ToUpper(fs[1])
		for i, val := range []string{"X", "Y", "Z"} {
			if strings.Contains(tmp, val) {
				continue
			}
			axis = i
		}
		filterfunc = func(el *st.Elem) bool {
			if el.IsLineElem() {
				return el.Direction(false)[axis] == 0.0
			} else {
				n := el.Normal(false)
				if n == nil {
					return false
				}
				for i := 0; i < 3; i++ {
					if i == axis {
						continue
					}
					if n[i] != 0.0 {
						return false
					}
				}
				return true
			}
		}
	case re_sectnum.MatchString(str):
		filterfunc, hstr = SectFilter(str)
	case re_etype.MatchString(str):
		filterfunc, hstr = EtypeFilter(str)
	case adjoin.MatchString(str):
		fs := adjoin.FindStringSubmatch(str)
		if len(fs) >= 5 {
			condition := fs[4]
			var fil func(*st.Elem) bool
			var hst string
			switch {
			case re_sectnum.MatchString(condition):
				fil, hst = SectFilter(condition)
			case re_etype.MatchString(condition):
				fil, hst = EtypeFilter(condition)
			}
			if fil == nil {
				break
			}
			filterfunc = func(el *st.Elem) bool {
				for _, en := range el.Enod {
					for _, sel := range stw.Frame.SearchElem(en) {
						if sel.Num == el.Num {
							continue
						}
						if fil(sel) {
							return true
						}
					}
				}
				return false
			}
			hstr = fmt.Sprintf("ADJOIN TO %s", hst)
		}
	}
	if filterfunc != nil {
		tmpels := make([]*st.Elem, l)
		enum := 0
		for _, el := range els {
			if el == nil {
				continue
			}
			if filterfunc(el) {
				tmpels[enum] = el
				enum++
			}
		}
		rtn := tmpels[:enum]
		stw.History(fmt.Sprintf("FILTER: %s", hstr))
		return rtn, nil
	} else {
		return els, errors.New("no filtering")
	}
}

func (stw *Window) NextFloor() {
	for _, n := range stw.Frame.Nodes {
		n.Show()
	}
	for _, el := range stw.Frame.Elems {
		el.Show()
	}
	for i, _ := range []string{"ZMIN", "ZMAX"} {
		tmpval := stw.Frame.Show.Zrange[i]
		ind := 0
		for _, ht := range stw.Frame.Ai.Boundary {
			if ht > tmpval {
				break
			}
			ind++
		}
		var val float64
		l := len(stw.Frame.Ai.Boundary)
		if ind >= l-1 {
			val = stw.Frame.Ai.Boundary[l-2+i]
		} else {
			val = stw.Frame.Ai.Boundary[ind]
		}
		stw.Frame.Show.Zrange[i] = val
		// stw.Labels[z].SetAttribute("VALUE", fmt.Sprintf("%.3f", val))
	}
	stw.Redraw()
}

func (stw *Window) PrevFloor() {
	for _, n := range stw.Frame.Nodes {
		n.Show()
	}
	for _, el := range stw.Frame.Elems {
		el.Show()
	}
	for i, _ := range []string{"ZMIN", "ZMAX"} {
		tmpval := stw.Frame.Show.Zrange[i]
		ind := 0
		for _, ht := range stw.Frame.Ai.Boundary {
			if ht > tmpval {
				break
			}
			ind++
		}
		var val float64
		if ind <= 2 {
			val = stw.Frame.Ai.Boundary[i]
		} else {
			val = stw.Frame.Ai.Boundary[ind-2]
		}
		stw.Frame.Show.Zrange[i] = val
		// stw.Labels[z].SetAttribute("VALUE", fmt.Sprintf("%.3f", val))
	}
	stw.Redraw()
}

func (stw *Window) AxisRange(axis int, min, max float64, any bool) {
	tmpnodes := make([]*st.Node, 0)
	for _, n := range stw.Frame.Nodes {
		if !(min <= n.Coord[axis] && n.Coord[axis] <= max) {
			tmpnodes = append(tmpnodes, n)
			n.Hide()
		} else {
			n.Show()
		}
	}
	var tmpelems []*st.Elem
	if !any {
		tmpelems = stw.Frame.NodeToElemAny(tmpnodes...)
	} else {
		tmpelems = stw.Frame.NodeToElemAll(tmpnodes...)
	}
	for _, el := range stw.Frame.Elems {
		el.Show()
	}
	for _, el := range tmpelems {
		el.Hide()
	}
	switch axis {
	case 0:
		stw.Frame.Show.Xrange[0] = min
		stw.Frame.Show.Xrange[1] = max
		// stw.Labels["XMIN"].SetAttribute("VALUE", fmt.Sprintf("%.3f", min))
		// stw.Labels["XMAX"].SetAttribute("VALUE", fmt.Sprintf("%.3f", max))
	case 1:
		stw.Frame.Show.Yrange[0] = min
		stw.Frame.Show.Yrange[1] = max
		// stw.Labels["YMIN"].SetAttribute("VALUE", fmt.Sprintf("%.3f", min))
		// stw.Labels["YMAX"].SetAttribute("VALUE", fmt.Sprintf("%.3f", max))
	case 2:
		stw.Frame.Show.Zrange[0] = min
		stw.Frame.Show.Zrange[1] = max
		// stw.Labels["ZMIN"].SetAttribute("VALUE", fmt.Sprintf("%.3f", min))
		// stw.Labels["ZMAX"].SetAttribute("VALUE", fmt.Sprintf("%.3f", max))
	}
	stw.Redraw()
}

// Command
func (stw *Window) feedCommand() {
	command := stw.cline.Text()
	if command != "" {
		stw.addCommandHistory(command)
		comhistpos = -1
		stw.cline.SetText("")
		stw.execAliasCommand(command)
	}
}

func (stw *Window) SetCommandHistory() error {
	if st.FileExists(historyfn) {
		tmp := make([]string, CommandHistorySize)
		f, err := os.Open(historyfn)
		if err != nil {
			return err
		}
		s := bufio.NewScanner(f)
		num := 0
		for s.Scan() {
			if com := s.Text(); com != "" {
				tmp[num] = com
				num++
			}
		}
		if err := s.Err(); err != nil {
			return err
		}
		stw.comhist = tmp
		return nil
	}
	return errors.New("SetCommandHistory: file doesn't exist")
}

func (stw *Window) SaveCommandHistory() error {
	var otp bytes.Buffer
	for _, com := range stw.comhist {
		if com != "" {
			otp.WriteString(com)
			otp.WriteString("\n")
		}
	}
	w, err := os.Create(historyfn)
	defer w.Close()
	if err != nil {
		return err
	}
	otp = st.AddCR(otp)
	otp.WriteTo(w)
	return nil
}

func (stw *Window) addCommandHistory(str string) {
	tmp := make([]string, CommandHistorySize)
	tmp[0] = str
	for i := 0; i < CommandHistorySize-1; i++ {
		tmp[i+1] = stw.comhist[i]
	}
	stw.comhist = tmp
}

func (stw *Window) PrevCommand(str string) {
	for {
		comhistpos++
		if comhistpos >= CommandHistorySize {
			comhistpos = CommandHistorySize - 1
			return
		}
		if strings.HasPrefix(stw.comhist[comhistpos], str) {
			stw.cline.SetText(stw.comhist[comhistpos])
			// stw.cline.SetAttribute("CARETPOS", fmt.Sprintf("%d", len(stw.comhist[comhistpos])))
			return
		}
	}
}

func (stw *Window) NextCommand(str string) {
	for {
		comhistpos--
		if comhistpos <= 0 {
			comhistpos = -1
			return
		}
		if strings.HasPrefix(stw.comhist[comhistpos], str) {
			stw.cline.SetText(stw.comhist[comhistpos])
			// stw.cline.SetAttribute("CARETPOS", fmt.Sprintf("%d", len(stw.comhist[comhistpos])))
			return
		}
	}
}

func (stw *Window) ExecCommand(com *Command) {
	stw.History(com.Name)
	// stw.cname.SetAttribute("VALUE", com.Name)
	stw.lastcommand = com
	com.Exec(stw)
}

func (stw *Window) execAliasCommand(al string) {
	if stw.Frame == nil {
		if strings.HasPrefix(al, ":") {
			err := st.ExMode(stw, stw.Frame, al)
			if err != nil {
				stw.ErrorMessage(err, st.ERROR)
			}
		} else {
			stw.Open()
		}
		// stw.FocusCanv()
		return
	}
	alu := strings.ToUpper(al)
	if alu == "." {
		if stw.lastcommand != nil {
			stw.ExecCommand(stw.lastcommand)
		}
	} else if value, ok := aliases[alu]; ok {
		stw.ExecCommand(value)
	} else if value, ok := Commands[alu]; ok {
		stw.ExecCommand(value)
	} else {
		switch {
		default:
			stw.History(fmt.Sprintf("command doesn't exist: %s", al))
		case strings.HasPrefix(al, ":"):
			err := st.ExMode(stw, stw.Frame, al)
			if err != nil {
				stw.ErrorMessage(err, st.ERROR)
			}
		case strings.HasPrefix(al, "'"):
			err := st.Fig2Mode(stw, stw.Frame, al)
			if err != nil {
				stw.ErrorMessage(err, st.ERROR)
			}
		case axrn_minmax.MatchString(alu):
			var axis int
			fs := axrn_minmax.FindStringSubmatch(alu)
			min, _ := strconv.ParseFloat(fs[1], 64)
			max, _ := strconv.ParseFloat(fs[3], 64)
			tmp := strings.ToUpper(fs[2])
			for i, val := range []string{"X", "Y", "Z"} {
				if tmp == val {
					axis = i
					break
				}
			}
			stw.AxisRange(axis, min, max, false)
			stw.History(fmt.Sprintf("AxisRange: %.3f <= %s <= %.3f", min, tmp, max))
		case axrn_min1.MatchString(alu):
			var axis int
			fs := axrn_min1.FindStringSubmatch(alu)
			min, _ := strconv.ParseFloat(fs[1], 64)
			tmp := strings.ToUpper(fs[2])
			for i, val := range []string{"X", "Y", "Z"} {
				if tmp == val {
					axis = i
					break
				}
			}
			stw.AxisRange(axis, min, 1000.0, false)
			stw.History(fmt.Sprintf("AxisRange: %.3f <= %s", min, tmp))
		case axrn_min2.MatchString(alu):
			var axis int
			fs := axrn_min2.FindStringSubmatch(alu)
			min, _ := strconv.ParseFloat(fs[2], 64)
			tmp := strings.ToUpper(fs[1])
			for i, val := range []string{"X", "Y", "Z"} {
				if tmp == val {
					axis = i
					break
				}
			}
			stw.AxisRange(axis, min, 1000.0, false)
			stw.History(fmt.Sprintf("AxisRange: %.3f <= %s", min, tmp))
		case axrn_max1.MatchString(alu):
			var axis int
			fs := axrn_max1.FindStringSubmatch(alu)
			max, _ := strconv.ParseFloat(fs[1], 64)
			tmp := strings.ToUpper(fs[2])
			for i, val := range []string{"X", "Y", "Z"} {
				if tmp == val {
					axis = i
					break
				}
			}
			stw.AxisRange(axis, -100.0, max, false)
			stw.History(fmt.Sprintf("AxisRange: %s <= %.3f", tmp, max))
		case axrn_max2.MatchString(alu):
			var axis int
			fs := axrn_max2.FindStringSubmatch(alu)
			max, _ := strconv.ParseFloat(fs[2], 64)
			tmp := strings.ToUpper(fs[1])
			for i, val := range []string{"X", "Y", "Z"} {
				if tmp == val {
					axis = i
					break
				}
			}
			stw.AxisRange(axis, -100.0, max, false)
			stw.History(fmt.Sprintf("AxisRange: %s <= %.3f", tmp, max))
		case axrn_eq.MatchString(alu):
			var axis int
			fs := axrn_eq.FindStringSubmatch(alu)
			val, _ := strconv.ParseFloat(fs[2], 64)
			tmp := strings.ToUpper(fs[1])
			for i, val := range []string{"X", "Y", "Z"} {
				if tmp == val {
					axis = i
					break
				}
			}
			stw.AxisRange(axis, val, val, false)
			stw.History(fmt.Sprintf("AxisRange: %s = %.3f", tmp, val))
		}
	}
	stw.Redraw()
	if stw.cline.Text() == "" {
		// stw.FocusCanv()
	}
	return
}

func (stw *Window) EscapeCB() {
	stw.cline.SetText("")
	stw.initDrawAreaCallback()
	if stw.Frame != nil {
		stw.Redraw()
	}
	comhistpos = -1
	clineinput = ""
}

func (stw *Window) EscapeAll() {
	stw.Deselect()
	stw.EscapeCB()
}

// Message
func (stw *Window) History(str string) {
	if str == "" {
		return
	}
	current := stw.history.Text()
	newstr := fmt.Sprintf("%s\n%s", current, str)
	stw.history.SetText(newstr)
}

func (stw *Window) ErrorMessage(err error, level int) {
	if err == nil {
		return
	}
	var otp string
	if level >= st.ERROR {
		_, file, line, _ := runtime.Caller(1)
		otp = fmt.Sprintf("%s:%d: [%s]: %s", filepath.Base(file), line, st.LOGLEVEL[level], err.Error())
	} else {
		otp = fmt.Sprintf("[%s]: %s", st.LOGLEVEL[level], err.Error())
	}
	stw.History(otp)
	logger.Println(otp)
}

func (stw *Window) ReadPgp(filename string) error {
	aliases = make(map[string]*Command, 0)
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		txt := s.Text()
		if strings.HasPrefix(txt, "#") {
			continue
		}
		var words []string
		for _, k := range strings.Split(txt, " ") {
			if k != "" {
				words = append(words, k)
			}
		}
		if len(words) < 2 {
			continue
		}
		if value, ok := Commands[strings.ToUpper(words[1])]; ok {
			aliases[strings.ToUpper(words[0])] = value
		} else if strings.HasPrefix(words[1], ":") {
			command := strings.Join(words[1:], " ")
			if strings.Contains(command, "_") {
				aliases[strings.ToUpper(words[0])] = &Command{"", "", "", func(stw *Window) {
					// pos := strings.Index(command, "_")
					// iup.SetFocus(stw.cline)
					// stw.cline.SetAttribute("VALUE", strings.Replace(command, "_", "", -1))
					// stw.cline.SetAttribute("CARETPOS", fmt.Sprintf("%d", pos))
				}}
			} else if strings.Contains(command, "@") {
				pat := regexp.MustCompile("@[0-9-]*")
				str := pat.FindString(command)
				var tmp int64
				tmp, err := strconv.ParseInt(str[1:], 10, 64)
				if err != nil {
					tmp = 0
				}
				val := int(tmp)
				aliases[strings.ToUpper(words[0])] = &Command{"", "", "", func(stw *Window) {
					currentcommand := strings.Replace(command, str, fmt.Sprintf("%d", val), -1)
					err := st.ExMode(stw, stw.Frame, currentcommand)
					if err != nil {
						stw.ErrorMessage(err, st.ERROR)
					}
					val++
				}}
			} else {
				aliases[strings.ToUpper(words[0])] = &Command{"", "", "", func(stw *Window) {
					err := st.ExMode(stw, stw.Frame, command)
					if err != nil {
						stw.ErrorMessage(err, st.ERROR)
					}
				}}
			}
		} else if strings.HasPrefix(words[1], "'") {
			command := strings.Join(words[1:], " ")
			if strings.Contains(command, "_") {
				aliases[strings.ToUpper(words[0])] = &Command{"", "", "", func(stw *Window) {
					// pos := strings.Index(command, "_")
					// iup.SetFocus(stw.cline)
					// stw.cline.SetAttribute("VALUE", strings.Replace(command, "_", "", -1))
					// stw.cline.SetAttribute("CARETPOS", fmt.Sprintf("%d", pos))
				}}
			} else {
				aliases[strings.ToUpper(words[0])] = &Command{"", "", "", func(stw *Window) {
					err := st.Fig2Mode(stw, stw.Frame, command)
					if err != nil {
						stw.ErrorMessage(err, st.ERROR)
					}
				}}
			}
		}
	}
	if err := s.Err(); err != nil {
		return err
	}
	return nil
}

func StartLogging() {
	logf, err := os.OpenFile(LOGFILE, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println(err.Error())
	}
	logger = log.New(logf, "", log.LstdFlags)
	logger.Println("session started")
}

func StopLogging() {
	logger.Println("session closed")
	logf.Close()
}
///

// Select
func (stw *Window) PickNode(x, y int) (rtn *st.Node) {
	mindist := float64(nodeSelectPixel)
	for _, v := range stw.Frame.Nodes {
		if v.IsHidden(stw.Frame.Show) {
			continue
		}
		dist := math.Hypot(float64(x)-v.Pcoord[0], float64(y)-v.Pcoord[1])
		if dist < mindist {
			mindist = dist
			rtn = v
		} else if dist == mindist {
			if rtn.DistFromProjection(stw.Frame.View) > v.DistFromProjection(stw.Frame.View) {
				rtn = v
			}
		}
	}
	if stw.Frame.Show.Kijun {
		for _, k := range stw.Frame.Kijuns {
			if k.IsHidden(stw.Frame.Show) {
				continue
			}
			for _, n := range [][]float64{k.Start, k.End} {
				pc := stw.Frame.View.ProjectCoord(n)
				dist := math.Hypot(float64(x)-pc[0], float64(y)-pc[1])
				if dist < mindist {
					mindist = dist
					rtn = st.NewNode()
					rtn.Coord = n
					rtn.Pcoord = pc
				}
			}
		}
	}
	return
}

func (stw *Window) StartSelection(ev gxui.MouseEvent) {
	stw.startX = ev.Point.X
	stw.startY = ev.Point.Y
	stw.endX = ev.Point.X
	stw.endY = ev.Point.Y
	first = 1
}

func (stw *Window) SelectNodeUp(ev gxui.MouseEvent) {
	left := min(stw.startX, stw.endX)
	right := max(stw.startX, stw.endX)
	bottom := min(stw.startY, stw.endY)
	top := max(stw.startY, stw.endY)
	if (right-left < nodeSelectPixel) && (top-bottom < nodeSelectPixel) {
		n := stw.PickNode(left, bottom)
		if n != nil {
			stw.MergeSelectNode([]*st.Node{n}, ev.Modifier.Shift())
		} else {
			stw.selectNode = make([]*st.Node, 0)
		}
	} else {
		tmpselect := make([]*st.Node, len(stw.Frame.Nodes))
		i := 0
		for _, v := range stw.Frame.Nodes {
			if v.IsHidden(stw.Frame.Show) {
				continue
			}
			if float64(left) <= v.Pcoord[0] && v.Pcoord[0] <= float64(right) && float64(bottom) <= v.Pcoord[1] && v.Pcoord[1] <= float64(top) {
				tmpselect[i] = v
				i++
			}
		}
		stw.MergeSelectNode(tmpselect[:i], ev.Modifier.Shift())
	}
}

// line: (x1, y1) -> (x2, y2), dot: (dx, dy)
// provided that x1*y2-x2*y1>0
//     if rtn>0: dot is the same side as (0, 0)
//     if rtn==0: dot is on the line
//     if rtn<0: dot is the opposite side to (0, 0)
func DotLine(x1, y1, x2, y2, dx, dy int) int {
	return (x1*y2 + x2*dy + dx*y1) - (x1*dy + x2*y1 + dx*y2)
}
func FDotLine(x1, y1, x2, y2, dx, dy float64) float64 {
	return (x1*y2 + x2*dy + dx*y1) - (x1*dy + x2*y1 + dx*y2)
}

func (stw *Window) PickElem(x, y int) (rtn *st.Elem) {
	el := stw.PickLineElem(x, y)
	if el == nil {
		els := stw.PickPlateElem(x, y)
		if len(els) > 0 {
			el = els[0]
		}
	}
	return el
}

func (stw *Window) PickLineElem(x, y int) (rtn *st.Elem) {
	mindist := float64(dotSelectPixel)
	for _, v := range stw.Frame.Elems {
		if v.IsHidden(stw.Frame.Show) {
			continue
		}
		if v.IsLineElem() && (math.Min(v.Enod[0].Pcoord[0], v.Enod[1].Pcoord[0]) <= float64(x+dotSelectPixel) && math.Max(v.Enod[0].Pcoord[0], v.Enod[1].Pcoord[0]) >= float64(x-dotSelectPixel)) && (math.Min(v.Enod[0].Pcoord[1], v.Enod[1].Pcoord[1]) <= float64(y+dotSelectPixel) && math.Max(v.Enod[0].Pcoord[1], v.Enod[1].Pcoord[1]) >= float64(y-dotSelectPixel)) {
			dist := math.Abs(FDotLine(v.Enod[0].Pcoord[0], v.Enod[0].Pcoord[1], v.Enod[1].Pcoord[0], v.Enod[1].Pcoord[1], float64(x), float64(y)))
			if plen := math.Hypot(v.Enod[0].Pcoord[0]-v.Enod[1].Pcoord[0], v.Enod[0].Pcoord[1]-v.Enod[1].Pcoord[1]); plen > 1E-12 {
				dist /= plen
			}
			if dist < mindist {
				mindist = dist
				rtn = v
			}
		}
	}
	return
}

func abs(val int) int {
	if val >= 0 {
		return val
	} else {
		return -val
	}
}
func (stw *Window) PickPlateElem(x, y int) []*st.Elem {
	rtn := make(map[int]*st.Elem)
	for _, el := range stw.Frame.Elems {
		if el.IsHidden(stw.Frame.Show) {
			continue
		}
		if !el.IsLineElem() {
			add := true
			sign := 0
			for i := 0; i < int(el.Enods); i++ {
				var j int
				if i == int(el.Enods)-1 {
					j = 0
				} else {
					j = i + 1
				}
				if FDotLine(el.Enod[i].Pcoord[0], el.Enod[i].Pcoord[1], el.Enod[j].Pcoord[0], el.Enod[j].Pcoord[1], float64(x), float64(y)) > 0 {
					sign++
				} else {
					sign--
				}
				if i+1 != abs(sign) {
					add = false
					break
				}
			}
			if add {
				rtn[el.Num] = el
			}
		}
	}
	return st.SortedElem(rtn, func(e *st.Elem) float64 { return e.DistFromProjection(stw.Frame.View) })
}

func (stw *Window) SelectElemUp(ev gxui.MouseEvent) {
	left := min(stw.startX, stw.endX)
	right := max(stw.startX, stw.endX)
	bottom := min(stw.startY, stw.endY)
	top := max(stw.startY, stw.endY)
	if (right-left < dotSelectPixel) && (top-bottom < dotSelectPixel) {
		el := stw.PickLineElem(left, bottom)
		if el == nil {
			els := stw.PickPlateElem(left, bottom)
			if len(els) > 0 {
				el = els[0]
			}
		}
		if el != nil {
			stw.MergeSelectElem([]*st.Elem{el}, ev.Modifier.Shift())
		} else {
			stw.selectElem = make([]*st.Elem, 0)
		}
	} else {
		tmpselectnode := make([]*st.Node, len(stw.Frame.Nodes))
		i := 0
		for _, v := range stw.Frame.Nodes {
			if float64(left) <= v.Pcoord[0] && v.Pcoord[0] <= float64(right) && float64(bottom) <= v.Pcoord[1] && v.Pcoord[1] <= float64(top) {
				tmpselectnode[i] = v
				i++
			}
		}
		tmpselectelem := make([]*st.Elem, len(stw.Frame.Elems))
		k := 0
		switch selectDirection {
		case SD_FROMLEFT:
			for _, el := range stw.Frame.Elems {
				if el.IsHidden(stw.Frame.Show) {
					continue
				}
				add := true
				for _, en := range el.Enod {
					var j int
					for j = 0; j < i; j++ {
						if en == tmpselectnode[j] {
							break
						}
					}
					if j == i {
						add = false
						break
					}
				}
				if add {
					tmpselectelem[k] = el
					k++
				}
			}
		case SD_FROMRIGHT:
			for _, el := range stw.Frame.Elems {
				if el.IsHidden(stw.Frame.Show) {
					continue
				}
				add := false
				for _, en := range el.Enod {
					found := false
					for j := 0; j < i; j++ {
						if en == tmpselectnode[j] {
							found = true
							break
						}
					}
					if found {
						add = true
						break
					}
				}
				if add {
					tmpselectelem[k] = el
					k++
				}
			}
		}
		stw.MergeSelectElem(tmpselectelem[:k], ev.Modifier.Shift())
	}
}

func (stw *Window) SelectElemFenceUp(ev gxui.MouseEvent) {
	tmpselectelem := make([]*st.Elem, len(stw.Frame.Elems))
	k := 0
	for _, el := range stw.Frame.Elems {
		if el.IsHidden(stw.Frame.Show) {
			continue
		}
		add := false
		sign := 0
		for i, en := range el.Enod {
			if FDotLine(float64(stw.startX), float64(stw.startY), float64(stw.endX), float64(stw.endY), en.Pcoord[0], en.Pcoord[1]) > 0 {
				sign++
			} else {
				sign--
			}
			if i+1 != abs(sign) {
				add = true
				break
			}
		}
		if add {
			if el.IsLineElem() {
				if FDotLine(el.Enod[0].Pcoord[0], el.Enod[0].Pcoord[1], el.Enod[1].Pcoord[0], el.Enod[1].Pcoord[1], float64(stw.startX), float64(stw.startY))*FDotLine(el.Enod[0].Pcoord[0], el.Enod[0].Pcoord[1], el.Enod[1].Pcoord[0], el.Enod[1].Pcoord[1], float64(stw.endX), float64(stw.endY)) < 0 {
					tmpselectelem[k] = el
					k++
				}
			} else {
				addx := false
				sign := 0
				for i, j := range el.Enod {
					if float64(max(stw.startX, stw.endX)) < j.Pcoord[0] {
						sign++
					} else if j.Pcoord[0] < float64(min(stw.startX, stw.endX)) {
						sign--
					}
					if i+1 != abs(sign) {
						addx = true
						break
					}
				}
				if addx {
					addy := false
					sign := 0
					for i, j := range el.Enod {
						if float64(max(stw.startY, stw.endY)) < j.Pcoord[1] {
							sign++
						} else if j.Pcoord[1] < float64(min(stw.startY, stw.endY)) {
							sign--
						}
						if i+1 != abs(sign) {
							addy = true
							break
						}
					}
					if addy {
						tmpselectelem[k] = el
						k++
					}
				}
			}
		}
	}
	stw.MergeSelectElem(tmpselectelem[:k], ev.Modifier.Shift())
	stw.Redraw()
}

func (stw *Window) MergeSelectNode(nodes []*st.Node, isshift bool) {
	k := len(nodes)
	if isshift {
		for l := 0; l < k; l++ {
			for m, el := range stw.selectNode {
				if el == nodes[l] {
					if m == len(stw.selectNode)-1 {
						stw.selectNode = stw.selectNode[:m]
					} else {
						stw.selectNode = append(stw.selectNode[:m], stw.selectNode[m+1:]...)
					}
					break
				}
			}
		}
	} else {
		var add bool
		for l := 0; l < k; l++ {
			add = true
			for _, n := range stw.selectNode {
				if n == nodes[l] {
					add = false
					break
				}
			}
			if add {
				stw.selectNode = append(stw.selectNode, nodes[l])
			}
		}
	}
}

func (stw *Window) MergeSelectElem(elems []*st.Elem, isshift bool) {
	k := len(elems)
	if isshift {
		for l := 0; l < k; l++ {
			for m, el := range stw.selectElem {
				if el == elems[l] {
					if m == len(stw.selectElem)-1 {
						stw.selectElem = stw.selectElem[:m]
					} else {
						stw.selectElem = append(stw.selectElem[:m], stw.selectElem[m+1:]...)
					}
					break
				}
			}
		}
	} else {
		var add bool
		for l := 0; l < k; l++ {
			add = true
			for _, n := range stw.selectElem {
				if n == elems[l] {
					add = false
					break
				}
			}
			if add {
				stw.selectElem = append(stw.selectElem, elems[l])
			}
		}
	}
}

func (stw *Window) SelectNodeMotion(ev gxui.MouseEvent) {
	if stw.startX <= ev.Point.X {
		selectDirection = SD_FROMLEFT
		stw.rubber = stw.driver.CreateCanvas(gxmath.Size{W: stw.CanvasSize[0], H: stw.CanvasSize[1]})
		Rect(stw.rubber, RubberPenNode, RubberBrushNode, int(ev.Point.X), stw.startX, min(stw.startY, int(ev.Point.Y)), max(stw.startY, int(ev.Point.Y)))
		stw.rubber.Complete()
		stw.endX = ev.Point.X
		stw.endY = ev.Point.Y
	} else {
		selectDirection = SD_FROMRIGHT
		stw.rubber = stw.driver.CreateCanvas(gxmath.Size{W: stw.CanvasSize[0], H: stw.CanvasSize[1]})
		Rect(stw.rubber, RubberPenNode, RubberBrushNode, stw.startX, int(ev.Point.X), min(stw.startY, int(ev.Point.Y)), max(stw.startY, int(ev.Point.Y)))
		stw.rubber.Complete()
		stw.endX = ev.Point.X
		stw.endY = ev.Point.Y
	}
}

func (stw *Window) SelectElemMotion(ev gxui.MouseEvent) {
	if stw.startX <= ev.Point.X {
		selectDirection = SD_FROMLEFT
		stw.rubber = stw.driver.CreateCanvas(gxmath.Size{W: stw.CanvasSize[0], H: stw.CanvasSize[1]})
		Rect(stw.rubber, RubberPenLeft, RubberBrushLeft, int(ev.Point.X), stw.startX, min(stw.startY, int(ev.Point.Y)), max(stw.startY, int(ev.Point.Y)))
		stw.rubber.Complete()
		stw.endX = ev.Point.X
		stw.endY = ev.Point.Y
	} else {
		selectDirection = SD_FROMRIGHT
		stw.rubber = stw.driver.CreateCanvas(gxmath.Size{W: stw.CanvasSize[0], H: stw.CanvasSize[1]})
		Rect(stw.rubber, RubberPenRight, RubberBrushRight, stw.startX, int(ev.Point.X), min(stw.startY, int(ev.Point.Y)), max(stw.startY, int(ev.Point.Y)))
		stw.rubber.Complete()
		stw.endX = ev.Point.X
		stw.endY = ev.Point.Y
	}
}

func (stw *Window) SelectElemFenceMotion(ev gxui.MouseEvent) {
	if ev.Modifier.Control() {
		stw.rubber = stw.driver.CreateCanvas(gxmath.Size{W: stw.CanvasSize[0], H: stw.CanvasSize[1]})
		Line(stw.rubber, RubberPenLeft, stw.startX, stw.startY, int(ev.Point.X), stw.startY)
		stw.rubber.Complete()
		stw.endX = ev.Point.X
		stw.endY = stw.startY
	} else {
		stw.rubber = stw.driver.CreateCanvas(gxmath.Size{W: stw.CanvasSize[0], H: stw.CanvasSize[1]})
		Line(stw.rubber, RubberPenLeft, stw.startX, stw.startY, int(ev.Point.X), int(ev.Point.Y))
		stw.rubber.Complete()
		stw.endX = ev.Point.X
		stw.endY = ev.Point.Y
	}
}

func (stw *Window) TailLine(x, y int, ev gxui.MouseEvent) {
	stw.rubber = stw.driver.CreateCanvas(gxmath.Size{W: stw.CanvasSize[0], H: stw.CanvasSize[1]})
	Line(stw.rubber, RubberPenLeft, x, y, int(ev.Point.X), int(ev.Point.Y))
	stw.rubber.Complete()
	stw.endX = ev.Point.X
	stw.endY = ev.Point.Y
}

func (stw *Window) TailPolygon(ns []*st.Node, ev gxui.MouseEvent) {
	l := len(ns) + 1
	num := 0
	coords := make([][]int, l)
	for i := 0; i < l-1; i++ {
		if ns[i] == nil {
			continue
		}
		coords[num] = []int{int(ns[i].Pcoord[0]), int(ns[i].Pcoord[1])}
		num++
	}
	coords[num] = []int{ev.Point.X, ev.Point.Y}
	Polygon(stw.rubber, RubberPenLeft, RubberBrushLeft, coords[:num+1])
	stw.rubber.Complete()
	stw.endX = ev.Point.X
	stw.endY = ev.Point.Y
}

func (stw *Window) SelectNotHidden() {
	if stw.Frame == nil {
		return
	}
	stw.Deselect()
	stw.selectElem = make([]*st.Elem, len(stw.Frame.Elems))
	num := 0
	for _, el := range stw.Frame.Elems {
		if el.IsHidden(stw.Frame.Show) {
			continue
		}
		stw.selectElem[num] = el
		num++
	}
	stw.selectElem = stw.selectElem[:num]
}

func (stw *Window) Deselect() {
	stw.selectNode = make([]*st.Node, 0)
	stw.selectElem = make([]*st.Elem, 0)
}
///


// Show&Hide
func (stw *Window) SetShowRange() {
	xmin, xmax, ymin, ymax, zmin, zmax := stw.Frame.Bbox(true)
	stw.Frame.Show.Xrange[0] = xmin
	stw.Frame.Show.Xrange[1] = xmax
	stw.Frame.Show.Yrange[0] = ymin
	stw.Frame.Show.Yrange[1] = ymax
	stw.Frame.Show.Zrange[0] = zmin
	stw.Frame.Show.Zrange[1] = zmax
	// stw.Labels["XMAX"].SetAttribute("VALUE", fmt.Sprintf("%.3f", xmax))
	// stw.Labels["XMIN"].SetAttribute("VALUE", fmt.Sprintf("%.3f", xmin))
	// stw.Labels["YMAX"].SetAttribute("VALUE", fmt.Sprintf("%.3f", ymax))
	// stw.Labels["YMIN"].SetAttribute("VALUE", fmt.Sprintf("%.3f", ymin))
	// stw.Labels["ZMAX"].SetAttribute("VALUE", fmt.Sprintf("%.3f", zmax))
	// stw.Labels["ZMIN"].SetAttribute("VALUE", fmt.Sprintf("%.3f", zmin))
}

func (stw *Window) HideNotSelected() {
	if stw.selectElem != nil {
		for _, n := range stw.Frame.Nodes {
			n.Hide()
		}
		for _, el := range stw.Frame.Elems {
			el.Hide()
		}
		for _, el := range stw.selectElem {
			if el != nil {
				el.Show()
				for _, en := range el.Enod {
					en.Show()
				}
			}
		}
	}
	stw.SetShowRange()
	stw.Redraw()
}

func (stw *Window) HideSelected() {
	if stw.selectElem != nil {
		for _, el := range stw.selectElem {
			if el != nil {
				el.Hide()
			}
		}
	}
	stw.SetShowRange()
	stw.Redraw()
}

func (stw *Window) LockNotSelected() {
	if stw.selectElem != nil {
		for _, n := range stw.Frame.Nodes {
			n.Lock = true
		}
		for _, el := range stw.Frame.Elems {
			el.Lock = true
		}
		for _, el := range stw.selectElem {
			if el != nil {
				el.Lock = false
				for _, en := range el.Enod {
					en.Lock = false
				}
			}
		}
	}
	stw.Redraw()
}

func (stw *Window) LockSelected() {
	if stw.selectElem != nil {
		for _, el := range stw.selectElem {
			if el != nil {
				el.Lock = true
				for _, en := range el.Enod {
					en.Lock = true
				}
			}
		}
	}
	stw.Redraw()
}

func (stw *Window) HideAllSection() {
	for i, _ := range stw.Frame.Show.Sect {
		// if lb, ok := stw.Labels[fmt.Sprintf("%d", i)]; ok {
		// 	lb.SetAttribute("FGCOLOR", labelOFFColor)
		// }
		stw.Frame.Show.Sect[i] = false
	}
}

func (stw *Window) ShowAllSection() {
	for i, _ := range stw.Frame.Show.Sect {
		// if lb, ok := stw.Labels[fmt.Sprintf("%d", i)]; ok {
		// 	lb.SetAttribute("FGCOLOR", labelFGColor)
		// }
		stw.Frame.Show.Sect[i] = true
	}
}

func (stw *Window) HideSection(snum int) {
	// if lb, ok := stw.Labels[fmt.Sprintf("%d", snum)]; ok {
	// 	lb.SetAttribute("FGCOLOR", labelOFFColor)
	// }
	stw.Frame.Show.Sect[snum] = false
}

func (stw *Window) ShowSection(snum int) {
	// if lb, ok := stw.Labels[fmt.Sprintf("%d", snum)]; ok {
	// 	lb.SetAttribute("FGCOLOR", labelFGColor)
	// }
	stw.Frame.Show.Sect[snum] = true
}

func (stw *Window) HideEtype(etype int) {
	if etype == 0 {
		return
	}
	stw.Frame.Show.Etype[etype] = false
	// if lbl, ok := stw.Labels[st.ETYPES[etype]]; ok {
	// 	lbl.SetAttribute("FGCOLOR", labelOFFColor)
	// }
}

func (stw *Window) ShowEtype(etype int) {
	if etype == 0 {
		return
	}
	stw.Frame.Show.Etype[etype] = true
	// if lbl, ok := stw.Labels[st.ETYPES[etype]]; ok {
	// 	lbl.SetAttribute("FGCOLOR", labelFGColor)
	// }
}

func (stw *Window) ToggleEtype(etype int) {
	if stw.Frame.Show.Etype[etype] {
		stw.HideEtype(etype)
	} else {
		stw.ShowEtype(etype)
	}
}
/// Show/Hide


// Query
func (stw *Window) Yn(title, question string) bool {
	return true
}

func (stw *Window) Yna(title, question, another string) int {
	return 0
}
/// Query


// Label
func (stw *Window) SetColorMode(mode uint) {
	// stw.Labels["COLORMODE"].SetAttribute("VALUE", fmt.Sprintf("  %s", st.ECOLORS[mode]))
	stw.Frame.Show.ColorMode = mode
}

func (stw *Window) SetPeriod(per string) {
	// stw.Labels["PERIOD"].SetAttribute("VALUE", per)
	stw.Frame.Show.Period = per
}

func (stw *Window) IncrementPeriod(num int) {
	pat := regexp.MustCompile("([a-zA-Z]+)(@[0-9]+)")
	fs := pat.FindStringSubmatch(stw.Frame.Show.Period)
	if len(fs) < 3 {
		return
	}
	if nl, ok := stw.Frame.Nlap[strings.ToUpper(fs[1])]; ok {
		tmp, _ := strconv.ParseInt(fs[2][1:], 10, 64)
		val := int(tmp) + num
		if val < 1 || val > nl {
			return
		}
		per := strings.Replace(stw.Frame.Show.Period, fs[2], fmt.Sprintf("@%d", val), -1)
		// stw.Labels["PERIOD"].SetAttribute("VALUE", per)
		stw.Frame.Show.Period = per
	}
}

func (stw *Window) NodeCaptionOn(name string) {
	for i, j := range st.NODECAPTIONS {
		if j == name {
			// if lbl, ok := stw.Labels[name]; ok {
				// lbl.SetAttribute("FGCOLOR", labelFGColor)
			// }
			if stw.Frame != nil {
				stw.Frame.Show.NodeCaptionOn(1 << uint(i))
			}
		}
	}
}

func (stw *Window) NodeCaptionOff(name string) {
	for i, j := range st.NODECAPTIONS {
		if j == name {
			// if lbl, ok := stw.Labels[name]; ok {
			// 	lbl.SetAttribute("FGCOLOR", labelOFFColor)
			// }
			if stw.Frame != nil {
				stw.Frame.Show.NodeCaptionOff(1 << uint(i))
			}
		}
	}
}

func (stw *Window) ElemCaptionOn(name string) {
	for i, j := range st.ELEMCAPTIONS {
		if j == name {
			// if lbl, ok := stw.Labels[name]; ok {
			// 	lbl.SetAttribute("FGCOLOR", labelFGColor)
			// }
			if stw.Frame != nil {
				stw.Frame.Show.ElemCaptionOn(1 << uint(i))
			}
		}
	}
}

func (stw *Window) ElemCaptionOff(name string) {
	for i, j := range st.ELEMCAPTIONS {
		if j == name {
			// if lbl, ok := stw.Labels[name]; ok {
			// 	lbl.SetAttribute("FGCOLOR", labelOFFColor)
			// }
			if stw.Frame != nil {
				stw.Frame.Show.ElemCaptionOff(1 << uint(i))
			}
		}
	}
}

func (stw *Window) SrcanRateOn(names ...string) {
	defer func() {
		if stw.Frame.Show.SrcanRate != 0 {
			// stw.Labels["SRCAN_RATE"].SetAttribute("FGCOLOR", labelFGColor)
		}
	}()
	if len(names) == 0 {
		for i, _ := range st.SRCANS {
			// if lbl, ok := stw.Labels[j]; ok {
			// 	lbl.SetAttribute("FGCOLOR", labelFGColor)
			// }
			if stw.Frame != nil {
				stw.Frame.Show.SrcanRateOn(1 << uint(i))
			}
		}
		return
	}
	for _, name := range names {
		for i, j := range st.SRCANS {
			if j == name {
				// if lbl, ok := stw.Labels[name]; ok {
				// 	lbl.SetAttribute("FGCOLOR", labelFGColor)
				// }
				if stw.Frame != nil {
					stw.Frame.Show.SrcanRateOn(1 << uint(i))
				}
			}
		}
	}
}

func (stw *Window) SrcanRateOff(names ...string) {
	defer func() {
		if stw.Frame.Show.SrcanRate == 0 {
			// stw.Labels["SRCAN_RATE"].SetAttribute("FGCOLOR", labelOFFColor)
		}
	}()
	if len(names) == 0 {
		for i, _ := range st.SRCANS {
			// if lbl, ok := stw.Labels[j]; ok {
			// 	lbl.SetAttribute("FGCOLOR", labelOFFColor)
			// }
			if stw.Frame != nil {
				stw.Frame.Show.SrcanRateOff(1 << uint(i))
			}
		}
		return
	}
	for _, name := range names {
		for i, j := range st.SRCANS {
			if j == name {
				// if lbl, ok := stw.Labels[name]; ok {
				// 	lbl.SetAttribute("FGCOLOR", labelOFFColor)
				// }
				if stw.Frame != nil {
					stw.Frame.Show.SrcanRateOff(1 << uint(i))
				}
			}
		}
	}
}

func (stw *Window) StressOn(etype int, index uint) {
	stw.Frame.Show.Stress[etype] |= (1 << index)
	// if etype <= st.SLAB {
	// 	if lbl, ok := stw.Labels[fmt.Sprintf("%s_%s", st.ETYPES[etype], strings.ToUpper(st.StressName[index]))]; ok {
	// 		lbl.SetAttribute("FGCOLOR", labelFGColor)
	// 	}
	// }
}

func (stw *Window) StressOff(etype int, index uint) {
	stw.Frame.Show.Stress[etype] &= ^(1 << index)
	// if etype <= st.SLAB {
	// 	if lbl, ok := stw.Labels[fmt.Sprintf("%s_%s", st.ETYPES[etype], strings.ToUpper(st.StressName[index]))]; ok {
	// 		lbl.SetAttribute("FGCOLOR", labelOFFColor)
	// 	}
	// }
}

func (stw *Window) DeformationOn() {
	stw.Frame.Show.Deformation = true
	// stw.Labels["DEFORMATION"].SetAttribute("FGCOLOR", labelFGColor)
}

func (stw *Window) DeformationOff() {
	stw.Frame.Show.Deformation = false
	// stw.Labels["DEFORMATION"].SetAttribute("FGCOLOR", labelOFFColor)
}

func (stw *Window) DispOn(direction int) {
	name := fmt.Sprintf("NC_%s", st.DispName[direction])
	for i, str := range st.NODECAPTIONS {
		if name == str {
			stw.Frame.Show.NodeCaption |= (1 << uint(i))
			// stw.Labels[name].SetAttribute("FGCOLOR", labelFGColor)
			return
		}
	}
}

func (stw *Window) DispOff(direction int) {
	name := fmt.Sprintf("NC_%s", st.DispName[direction])
	for i, str := range st.NODECAPTIONS {
		if name == str {
			stw.Frame.Show.NodeCaption &= ^(1 << uint(i))
			// stw.Labels[name].SetAttribute("FGCOLOR", labelOFFColor)
			return
		}
	}
}
/// Label


func (stw *Window) DrawPivot(nodes []*st.Node, pivot, end chan int) {
	stw.DrawFrameNode()
	// stw.DrawTexts(stw.cdcanv, false)
	// stw.cdcanv.Foreground(pivotColor)
	ind := 0
	nnum := 0
	for {
		select {
		case <-end:
			return
		case <-pivot:
			ind++
			if ind >= 6 {
				// DrawNodeNum(nodes[nnum], stw.cdcanv)
				nnum++
				ind = 0
			}
		}
	}
}

func (stw *Window) RedrawNode() {
	canvas := stw.DrawFrameNode()
	stw.draw.SetCanvas(canvas)
}

func (stw *Window) Redraw() {
	canvas := stw.DrawFrame()
	stw.draw.SetCanvas(canvas)
}

func (stw *Window) ShapeData(sh st.Shape) {
	var tb *TextBox
	if t, tok := stw.textBox["SHAPE"]; tok {
		tb = t
	} else {
		tb = NewTextBox()
		tb.hide = false
		tb.Position = []int{stw.CanvasSize[0] - 300, 200}
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
	tb.value = strings.Split(otp.String(), "\n")
}

func min(x, y int) int {
	if x >= y {
		return y
	} else {
		return x
	}
}
func max(x, y int) int {
	if x >= y {
		return x
	} else {
		return y
	}
}

func (stw *Window) ReadResource(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		txt := s.Text()
		if strings.HasPrefix(txt, "#") {
			continue
		}
		stw.execAliasCommand(txt)
	}
	if err := s.Err(); err != nil {
		return err
	}
	return nil
}

func (stw *Window) Checkout(name string) error {
	f, exists := stw.taggedFrame[name]
	if !exists {
		return fmt.Errorf("tag %s doesn't exist", name)
	}
	stw.Frame = f
	return nil
}

func (stw *Window) AddTag(name string, bang bool) error {
	if !bang {
		if _, exists := stw.taggedFrame[name]; exists {
			return fmt.Errorf("tag %s already exists", name)
		}
	}
	stw.taggedFrame[name] = stw.Frame.Snapshot()
	return nil
}

func (stw *Window) UseUndo(yes bool) {
	NOUNDO = !yes
}

func (stw *Window) EPS() float64 {
	return EPS
}

func (stw *Window) SetEPS(val float64) {
	EPS = val
}

func (stw *Window) CanvasFitScale() float64 {
	return CanvasFitScale
}

func (stw *Window) SetCanvasFitScale(val float64) {
	CanvasFitScale = val
}

func (stw *Window) CanvasAnimateSpeed() float64 {
	return CanvasAnimateSpeed
}

func (stw *Window) SetCanvasAnimateSpeed(val float64) {
	CanvasAnimateSpeed = val
}

func (stw *Window) ToggleFixRotate() {
	fixRotate = !fixRotate
}

func (stw *Window) ToggleFixMove() {
	fixMove = !fixMove
}

func (stw *Window) ToggleAltSelectNode() {
	ALTSELECTNODE = !ALTSELECTNODE
}

func (stw *Window) AltSelectNode() bool {
	return ALTSELECTNODE
}

func (stw *Window) CheckFrame() {
	// checkframe(stw)
}

func (stw *Window) TextBox(name string) st.TextBox {
	if _, tok := stw.textBox[name]; !tok {
		stw.textBox[name] = NewTextBox()
	}
	return stw.textBox[name]
}

func (stw *Window) Cwd() string {
	return stw.cwd
}
func (stw *Window) HomeDir() string {
	return stw.Home
}
func (stw *Window) IsChanged() bool {
	return stw.Changed
}

func (stw *Window) SelectElem(els []*st.Elem) {
	stw.selectElem = els
}

func (stw *Window) SelectNode(ns []*st.Node) {
	stw.selectNode = ns
}

func (stw *Window) ElemSelected() bool {
	if stw.selectElem == nil || len(stw.selectElem) == 0 {
		return false
	} else {
		return true
	}
}

func (stw *Window) NodeSelected() bool {
	if stw.selectNode == nil || len(stw.selectNode) == 0 {
		return false
	} else {
		return true
	}
}

func (stw *Window) SelectedElems() []*st.Elem {
	return stw.selectElem
}

func (stw *Window) SelectedNodes() []*st.Node {
	return stw.selectNode
}

func (stw *Window) SelectConfed() {
	// selectconfed(stw)
}

func (stw *Window) SetPaperSize(s uint) {
	stw.papersize = s
}

func (stw *Window) PaperSize() uint {
	return stw.papersize
}

func (stw *Window) Pivot() bool {
	return drawpivot
}

func (stw *Window) SetConf(lis []bool) {
	// setconf(stw, lis)
}

func (stw *Window) LastExCommand() string {
	return stw.lastexcommand
}

func (stw *Window) SetLastExCommand(c string) {
	stw.lastexcommand = c
}

func (stw *Window) SetShowPrintRange(val bool) {
	showprintrange = val
}

func (stw *Window) ToggleShowPrintRange() {
	showprintrange = !showprintrange
}

func (stw *Window) LastFig2Command() string {
	return stw.lastfig2command
}

func (stw *Window) SetLastFig2Command(c string) {
	stw.lastfig2command = c
}

func (stw *Window) SetLabel(key, value string) {
	// if l, ok := stw.Labels[key]; ok {
	// 	l.SetAttribute("VALUE", value)
	// }
}

func (stw *Window) EnableLabel(key string) {
	// if l, ok := stw.Labels[key]; ok {
	// 	l.SetAttribute("FGCOLOR", labelFGColor)
	// }
}

func (stw *Window) DisableLabel(key string) {
	// if l, ok := stw.Labels[key]; ok {
	// 	l.SetAttribute("FGCOLOR", labelOFFColor)
	// }
}

func (stw *Window) GetCanvasSize() (int, int) {
	stw.SetCanvasSize()
	return stw.CanvasSize[0], stw.CanvasSize[1]
}

func (stw *Window) AddSectionAliase(key int, value string) {
	sectionaliases[key] = value
}

func (stw *Window) DeleteSectionAliase(key int) {
	delete(sectionaliases, key)
}

func (stw *Window) ClearSectionAliase() {
	sectionaliases = make(map[int]string, 0)
}

func (stw *Window) Print() {
}

func (stw *Window) ReadFig2(filename string) error {
	return nil
}
