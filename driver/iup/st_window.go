package stgui

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/atotto/clipboard"
	"github.com/visualfc/go-iup/cd"
	"github.com/visualfc/go-iup/iup"
	"github.com/yofu/abbrev"
	"github.com/yofu/complete"
	"github.com/yofu/st/stlib"
	"gopkg.in/fsnotify.v1"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Constants & Variables// {{{
// General
var (
	aliases            map[string]*Command
	sectionaliases     map[int]string
	Inps               []string
	SearchingInps      bool
	SearchingInpsDone  chan bool
	DoubleClickCommand = []string{"TOGGLEBOND", "EDITPLATEELEM"}
	comhistpos         int
	prevkey            int
	clineinput         string
	logf               os.File
	logger             *log.Logger
	watcher            *fsnotify.Watcher
	passwatcher        bool
)
var (
	gopath          = os.Getenv("GOPATH")
	home            = os.Getenv("HOME")
	releasenote     = filepath.Join(home, ".st/help/releasenote.html")
	pgpfile         = filepath.Join(home, ".st/st.pgp")
	historyfn       = filepath.Join(home, ".st/history.dat")
	analysiscommand = "C:/an/an.exe"
	ALTSELECTNODE   = true
)

const LOGFILE = "_st.log"

const ResourceFileName = ".strc"

const (
	windowSize = "FULLxFULL"
)

// Font
var (
	fontface        = "IPA明朝"
	commandFontFace = fontface
	commandFontSize = "11"
	labelFGColor    = "0 0 0"
	// labelBGColor = "255 255 255"
	labelBGColor          = "212 208 200"
	commandFGColor        = "0 127 31"
	commandBGColor        = "191 191 191"
	historyBGColor        = "220 220 220"
	sectionLabelColor     = "204 51 0"
	labelOFFColor         = "160 160 160"
	canvasFontFace        = fontface
	canvasFontSize        = 9
	sectiondlgFontSize    = "9"
	sectiondlgFGColor     = "255 255 255"
	sectiondlgBGColor     = "66 66 66"
	sectiondlgSelectColor = "255 255 0"
	canvasFontColor       = cd.CD_DARK_GRAY
	pivotColor            = cd.CD_GREEN
	DefaultTextAlignment  = cd.CD_BASE_LEFT
	printFontColor        = cd.CD_BLACK
	printFontFace         = fontface
	printFontSize         = 6
	showprintrange        = false
)

const (
	FONT_COMMAND = iota
	FONT_CANVAS
)

// keymode
const (
	NORMAL = iota
	VIEWEDIT
)

// Draw
var (
	first           = 1
	fixRotate       = false
	fixMove         = false
	deg10           = 10.0 * math.Pi / 180.0
	RangeView       = st.NewView()
	RangeViewDists  = []float64{1000.0, 3000.0}
	RangeViewAngle  = []float64{20.0, 225.0}
	RangeViewCenter = []float64{100.0, 100.0}
	dataareaheight  = 150
	drawpivot       = false
	keymode         = NORMAL
)

var (
	STLOGO = &TextBox{
		value:    []string{"         software", "     forstructural", "   analysisthename", "  ofwhichstandsfor", "", " sigmatau  stress", "structure  steel", "andsometh  ing", " likethat"},
		position: []float64{100.0, 100.0},
		Angle:    0.0,
		Font:     NewFont(),
		hide:     true,
	}
)

var (
	LOCKED_ELEM_COLOR = cd.CD_DARK_GRAY
	LOCKED_NODE_COLOR = cd.CD_DARK_YELLOW
)

// DataLabel
const (
	datalabelwidth = 40
	datatextwidth  = 40
	dataheight     = 10
)

// Select
var selectDirection = 0

const nodeSelectPixel = 15
const dotSelectPixel = 5
const (
	SD_FROMLEFT = iota
	SD_FROMRIGHT
)

// Key & Button
const (
	KEY_BS         = 8
	KEY_TAB        = 9
	KEY_ENTER      = 13
	KEY_DELETE     = 139
	KEY_ESCAPE     = 141
	KEY_UPARROW    = 130
	KEY_LEFTARROW  = 132
	KEY_RIGHTARROW = 134
	KEY_DOWNARROW  = 136
	KEY_SPACE      = 32
	KEY_F4         = 146

	BUTTON_LEFT   = 49
	BUTTON_CENTER = 50
	BUTTON_RIGHT  = 51

	STATUS_LEFT   = 1
	STATUS_CENTER = 2
	STATUS_RIGHT  = 4
)

// Command
const (
	repeatcommand      = 0.2 // sec
	CommandHistorySize = 100
)

var (
	pressed time.Time
)

var (
	axrn_minmax = regexp.MustCompile("([+-]?[-0-9.]+)<=?([XYZxyz]{1})<=?([+-]?[-0-9.]+)")
	axrn_min1   = regexp.MustCompile("([+-]?[-0-9.]+)<=?([XYZxyz]{1})")
	axrn_min2   = regexp.MustCompile("([XYZxyz]{1})>=?([+-]?[-0-9.]+)")
	axrn_max1   = regexp.MustCompile("([+-]?[-0-9.]+)>=?([XYZxyz]{1})")
	axrn_max2   = regexp.MustCompile("([XYZxyz]{1})<=?([+-]?[-0-9.]+)")
	axrn_eq     = regexp.MustCompile("([XYZxyz]{1})=([+-]?[-0-9.]+)")
)

// }}}

type Window struct { // {{{
	*st.DrawOption
	*st.Directory
	*st.RecentFiles
	*st.UndoStack
	*st.TagFrame

	frame                  *st.Frame
	Dlg                    *iup.Handle
	SideBar                *iup.Handle
	canv                   *iup.Handle
	cdcanv                 *cd.Canvas
	dbuff                  *cd.Canvas
	cline                  *iup.Handle
	coord                  *iup.Handle
	hist                   *iup.Handle
	formattag, lsformattag *iup.Handle
	cname                  *iup.Handle
	context                *iup.Handle

	currentCanvas *cd.Canvas

	sectiondlg *iup.Handle

	CanvasSize []float64 // width, height

	zb *iup.Handle

	lselect *iup.Handle

	selectNode []*st.Node
	selectElem []*st.Elem

	textBox map[string]*TextBox

	papersize uint

	Version  string
	Modified string

	startT time.Time
	startX int
	startY int
	endX   int
	endY   int

	lastcommand     *Command
	lastexcommand   string
	lastfig2command string

	Labels map[string]*iup.Handle

	Property bool
	Selected []*iup.Handle
	Props    []*iup.Handle

	InpModified bool
	changed     bool

	comhist []string

	complete     *complete.Complete
	completepos  int
	completes    []string
	completefunc func(string) string
}

// }}}

func NewWindow(homedir string) *Window { // {{{
	stw := &Window{
		DrawOption:  st.NewDrawOption(),
		Directory:   st.NewDirectory(homedir, homedir),
		RecentFiles: st.NewRecentFiles(3),
		UndoStack:   st.NewUndoStack(10),
		TagFrame:    st.NewTagFrame(),
	}
	stw.selectNode = make([]*st.Node, 0)
	stw.selectElem = make([]*st.Elem, 0)
	stw.Labels = make(map[string]*iup.Handle)
	stw.Labels["GAXIS"] = stw.displayLabel("GAXIS", true)
	stw.Labels["EAXIS"] = stw.displayLabel("EAXIS", false)
	stw.Labels["BOND"] = stw.displayLabel("BOND", true)
	stw.Labels["CONF"] = stw.displayLabel("CONF", true)
	stw.Labels["PHINGE"] = stw.displayLabel("PHINGE", true)
	stw.Labels["KIJUN"] = stw.displayLabel("KIJUN", false)
	stw.Labels["DEFORMATION"] = stw.displayLabel("DEFORMATION", false)
	stw.Labels["GFACT"] = datatext("0.0")
	stw.Labels["DISTR"] = datatext("0.0")
	stw.Labels["DISTL"] = datatext("0.0")
	stw.Labels["PHI"] = datatext("0.0")
	stw.Labels["THETA"] = datatext("0.0")
	stw.Labels["FOCUSX"] = datatext("0.0")
	stw.Labels["FOCUSY"] = datatext("0.0")
	stw.Labels["FOCUSZ"] = datatext("0.0")
	stw.Labels["CENTERX"] = datatext("0.0")
	stw.Labels["CENTERY"] = datatext("0.0")
	stw.Labels["COLUMN"] = stw.etypeLabel("  COLUMN", datalabelwidth+datatextwidth, st.COLUMN, true)
	stw.Labels["GIRDER"] = stw.etypeLabel("  GIRDER", datalabelwidth+datatextwidth, st.GIRDER, true)
	stw.Labels["BRACE"] = stw.etypeLabel("  BRACE", datalabelwidth+datatextwidth, st.BRACE, true)
	stw.Labels["WALL"] = stw.etypeLabel("  WALL", datalabelwidth-10, st.WALL, true)
	stw.Labels["SLAB"] = stw.etypeLabel("  SLAB", datalabelwidth-10, st.SLAB, true)
	stw.Labels["WBRACE"] = stw.etypeLabel("WBRACE", datatextwidth, st.WBRACE, false)
	stw.Labels["SBRACE"] = stw.etypeLabel("SBRACE", datatextwidth, st.SBRACE, false)
	stw.Labels["NC_NUM"] = stw.captionLabel("NODE", "  CODE", datalabelwidth+datatextwidth, st.NC_NUM, true)
	stw.Labels["NC_WEIGHT"] = stw.captionLabel("NODE", "  WEIGHT", datalabelwidth+datatextwidth, st.NC_WEIGHT, false)
	stw.Labels["NC_DX"] = stw.captionLabel("NODE", " dX", (datalabelwidth+datatextwidth)/6, st.NC_DX, false)
	stw.Labels["NC_DY"] = stw.captionLabel("NODE", " dY", (datalabelwidth+datatextwidth)/6, st.NC_DY, false)
	stw.Labels["NC_DZ"] = stw.captionLabel("NODE", " dZ", (datalabelwidth+datatextwidth)/6, st.NC_DZ, false)
	stw.Labels["NC_TX"] = stw.captionLabel("NODE", " tX", (datalabelwidth+datatextwidth)/6, st.NC_TX, false)
	stw.Labels["NC_TY"] = stw.captionLabel("NODE", " tY", (datalabelwidth+datatextwidth)/6, st.NC_TY, false)
	stw.Labels["NC_TZ"] = stw.captionLabel("NODE", " tZ", (datalabelwidth+datatextwidth)/6, st.NC_TZ, false)
	stw.Labels["NC_RX"] = stw.captionLabel("NODE", " Rx", (datalabelwidth+datatextwidth)/6, st.NC_RX, false)
	stw.Labels["NC_RY"] = stw.captionLabel("NODE", " Ry", (datalabelwidth+datatextwidth)/6, st.NC_RY, false)
	stw.Labels["NC_RZ"] = stw.captionLabel("NODE", " Rz", (datalabelwidth+datatextwidth)/6, st.NC_RZ, false)
	stw.Labels["NC_MX"] = stw.captionLabel("NODE", " Mx", (datalabelwidth+datatextwidth)/6, st.NC_MX, false)
	stw.Labels["NC_MY"] = stw.captionLabel("NODE", " My", (datalabelwidth+datatextwidth)/6, st.NC_MY, false)
	stw.Labels["NC_MZ"] = stw.captionLabel("NODE", " Mz", (datalabelwidth+datatextwidth)/6, st.NC_MZ, false)
	stw.Labels["EC_NUM"] = stw.captionLabel("ELEM", "  CODE", datalabelwidth+datatextwidth, st.EC_NUM, false)
	stw.Labels["EC_SECT"] = stw.captionLabel("ELEM", "  SECT", datalabelwidth+datatextwidth, st.EC_SECT, false)
	stw.Labels["SRCAN_RATE"] = stw.displayLabel("RATE", false)
	stw.Labels["SRCAN_L"] = stw.srcanLabel(" L", datatextwidth/4, st.SRCAN_L, false)
	stw.Labels["SRCAN_S"] = stw.srcanLabel(" S", datatextwidth/4, st.SRCAN_S, false)
	stw.Labels["SRCAN_Q"] = stw.srcanLabel(" Q", datatextwidth/4, st.SRCAN_Q, false)
	stw.Labels["SRCAN_M"] = stw.srcanLabel(" M", datatextwidth/4, st.SRCAN_M, false)
	stw.Labels["COLORMODE"] = stw.toggleLabel(2, st.ECOLORS)
	stw.Labels["PERIOD"] = datatext("L")
	stw.Labels["GAXISSIZE"] = datatext("1.0")
	stw.Labels["EAXISSIZE"] = datatext("0.5")
	stw.Labels["BONDSIZE"] = datatext("3.0")
	stw.Labels["CONFSIZE"] = datatext("9.0")
	stw.Labels["DFACT"] = datatext("100.0")
	stw.Labels["QFACT"] = datatext("0.5")
	stw.Labels["MFACT"] = datatext("0.5")
	stw.Labels["RFACT"] = datatext("0.5")
	stw.Labels["XMAX"] = datatext("1000.0")
	stw.Labels["XMIN"] = datatext("-100.0")
	stw.Labels["YMAX"] = datatext("1000.0")
	stw.Labels["YMIN"] = datatext("-100.0")
	stw.Labels["ZMAX"] = datatext("1000.0")
	stw.Labels["ZMIN"] = datatext("-100.0")
	stw.Labels["COLUMN_NZ"] = stw.stressLabel(st.COLUMN, 0)
	stw.Labels["COLUMN_QX"] = stw.stressLabel(st.COLUMN, 1)
	stw.Labels["COLUMN_QY"] = stw.stressLabel(st.COLUMN, 2)
	stw.Labels["COLUMN_MZ"] = stw.stressLabel(st.COLUMN, 3)
	stw.Labels["COLUMN_MX"] = stw.stressLabel(st.COLUMN, 4)
	stw.Labels["COLUMN_MY"] = stw.stressLabel(st.COLUMN, 5)
	stw.Labels["GIRDER_NZ"] = stw.stressLabel(st.GIRDER, 0)
	stw.Labels["GIRDER_QX"] = stw.stressLabel(st.GIRDER, 1)
	stw.Labels["GIRDER_QY"] = stw.stressLabel(st.GIRDER, 2)
	stw.Labels["GIRDER_MZ"] = stw.stressLabel(st.GIRDER, 3)
	stw.Labels["GIRDER_MX"] = stw.stressLabel(st.GIRDER, 4)
	stw.Labels["GIRDER_MY"] = stw.stressLabel(st.GIRDER, 5)
	stw.Labels["BRACE_NZ"] = stw.stressLabel(st.BRACE, 0)
	stw.Labels["BRACE_QX"] = stw.stressLabel(st.BRACE, 1)
	stw.Labels["BRACE_QY"] = stw.stressLabel(st.BRACE, 2)
	stw.Labels["BRACE_MZ"] = stw.stressLabel(st.BRACE, 3)
	stw.Labels["BRACE_MX"] = stw.stressLabel(st.BRACE, 4)
	stw.Labels["BRACE_MY"] = stw.stressLabel(st.BRACE, 5)
	stw.Labels["WALL_NZ"] = stw.stressLabel(st.WBRACE, 0)
	stw.Labels["WALL_QX"] = stw.stressLabel(st.WBRACE, 1)
	stw.Labels["WALL_QY"] = stw.stressLabel(st.WBRACE, 2)
	stw.Labels["WALL_MZ"] = stw.stressLabel(st.WBRACE, 3)
	stw.Labels["WALL_MX"] = stw.stressLabel(st.WBRACE, 4)
	stw.Labels["WALL_MY"] = stw.stressLabel(st.WBRACE, 5)
	stw.Labels["SLAB_NZ"] = stw.stressLabel(st.SBRACE, 0)
	stw.Labels["SLAB_QX"] = stw.stressLabel(st.SBRACE, 1)
	stw.Labels["SLAB_QY"] = stw.stressLabel(st.SBRACE, 2)
	stw.Labels["SLAB_MZ"] = stw.stressLabel(st.SBRACE, 3)
	stw.Labels["SLAB_MX"] = stw.stressLabel(st.SBRACE, 4)
	stw.Labels["SLAB_MY"] = stw.stressLabel(st.SBRACE, 5)
	stw.Labels["YIELD"] = stw.displayLabel("YIELD", false)

	iup.Menu(
		iup.Attrs("BGCOLOR", labelBGColor),
		iup.SubMenu("TITLE=File",
			iup.Menu(
				iup.Item(
					iup.Attr("TITLE", "New\tCtrl+N"),
					iup.Attr("TIP", "Create New file"),
					func(arg *iup.ItemAction) {
						stw.New()
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Open\tCtrl+O"),
					iup.Attr("TIP", "Open file"),
					func(arg *iup.ItemAction) {
						runtime.GC()
						stw.Open()
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Search Inp\t/"),
					iup.Attr("TIP", "Search Inp file"),
					func(arg *iup.ItemAction) {
						runtime.GC()
						stw.SearchInp()
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Insert\tCtrl+I"),
					iup.Attr("TIP", "Insert frame"),
					func(arg *iup.ItemAction) {
						stw.Insert()
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Reload\tCtrl+M"),
					func(arg *iup.ItemAction) {
						runtime.GC()
						st.Reload(stw)
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Open Dxf"),
					iup.Attr("TIP", "Open dxf file"),
					func(arg *iup.ItemAction) {
						runtime.GC()
						stw.OpenDxf()
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Save\tCtrl+S"),
					iup.Attr("TIP", "Save file"),
					func(arg *iup.ItemAction) {
						stw.Save()
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Save As"),
					iup.Attr("TIP", "Save file As"),
					func(arg *iup.ItemAction) {
						stw.SaveAS()
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Read"),
					iup.Attr("TIP", "Read file"),
					func(arg *iup.ItemAction) {
						stw.Read()
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Read All\tCtrl+R"),
					iup.Attr("TIP", "Read all file"),
					func(arg *iup.ItemAction) {
						st.ReadAll(stw)
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Print"),
					iup.Attr("TIP", "Print"),
					func(arg *iup.ItemAction) {
						stw.Print()
					},
				),
				iup.Separator(),
				iup.Item(
					iup.Attr("TITLE", "Quit"),
					iup.Attr("TIP", "Exit Application"),
					func(arg *iup.ItemAction) {
						if stw.changed {
							if stw.Yn("CHANGED", "変更を保存しますか") {
								stw.SaveAS()
							} else {
								return
							}
						}
						stw.SaveRecent()
						arg.Return = iup.CLOSE
					},
				),
			),
		),
		iup.SubMenu("TITLE=Edit",
			iup.Menu(
				iup.Item(
					iup.Attr("TITLE", "Command Font"),
					func(arg *iup.ItemAction) {
						stw.SetFont(FONT_COMMAND)
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Canvas Font"),
					func(arg *iup.ItemAction) {
						stw.SetFont(FONT_CANVAS)
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Edit Inp\tCtrl+E"),
					func(arg *iup.ItemAction) {
						stw.EditInp()
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Edit Pgp"),
					func(arg *iup.ItemAction) {
						EditPgp()
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Read Pgp"),
					func(arg *iup.ItemAction) {
						if name, ok := iup.GetOpenFile(stw.Cwd(), "*.pgp"); ok {
							err := stw.ReadPgp(name)
							if err != nil {
								stw.addHistory("ReadPgp: Cannot Read st.pgp")
							} else {
								stw.addHistory(fmt.Sprintf("ReadPgp: Read %s", name))
							}
						}
					},
				),
			),
		),
		iup.SubMenu("TITLE=View",
			iup.Menu(
				iup.Item(
					iup.Attr("TITLE", "Top"),
					func(arg *iup.ItemAction) {
						stw.SetAngle(90.0, -90.0)
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Front"),
					func(arg *iup.ItemAction) {
						stw.SetAngle(0.0, -90.0)
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Back"),
					func(arg *iup.ItemAction) {
						stw.SetAngle(0.0, 90.0)
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Right"),
					func(arg *iup.ItemAction) {
						stw.SetAngle(0.0, 0.0)
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Left"),
					func(arg *iup.ItemAction) {
						stw.SetAngle(0.0, 180.0)
					},
				),
			),
		),
		iup.SubMenu("TITLE=Show",
			iup.Menu(
				iup.Item(
					iup.Attr("TITLE", "Show All\tCtrl+S"),
					func(arg *iup.ItemAction) {
						stw.ShowAll()
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Hide Selected Elems\tCtrl+H"),
					func(arg *iup.ItemAction) {
						stw.HideSelected()
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Show Selected Elems\tCtrl+D"),
					func(arg *iup.ItemAction) {
						stw.HideNotSelected()
					},
				),
			),
		),
		iup.SubMenu("TITLE=Dialog",
			iup.Menu(
				iup.Item(
					iup.Attr("TITLE", "Section"),
					func(arg *iup.ItemAction) {
						if stw.sectiondlg != nil {
							iup.SetFocus(stw.sectiondlg)
						} else {
							stw.SectionDialog()
						}
					},
				),
				iup.Separator(),
				iup.Item(
					iup.Attr("TITLE", "Utility"),
					func(arg *iup.ItemAction) {
						stw.CommandDialogUtility()
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Node"),
					func(arg *iup.ItemAction) {
						stw.CommandDialogNode()
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Elem"),
					func(arg *iup.ItemAction) {
						stw.CommandDialogElem()
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Conf/Bond"),
					func(arg *iup.ItemAction) {
						stw.CommandDialogConfBond()
					},
				),
			),
		),
		iup.SubMenu("TITLE=Command",
			iup.Menu(iup.Item(iup.Attr("TITLE", "DISTS"), func(arg *iup.ItemAction) { stw.ExecCommand("DISTS") }),
				iup.Item(iup.Attr("TITLE", "SET FOCUS"), func(arg *iup.ItemAction) { stw.ExecCommand("SETFOCUS") }),
				iup.Separator(),
				iup.Item(iup.Attr("TITLE", "TOGGLEBOND"), func(arg *iup.ItemAction) { stw.ExecCommand("TOGGLEBOND") }),
				iup.Item(iup.Attr("TITLE", "COPYBOND"), func(arg *iup.ItemAction) { stw.ExecCommand("COPYBOND") }),
				iup.Separator(),
				iup.Item(iup.Attr("TITLE", "SELECT NODE"), func(arg *iup.ItemAction) { stw.ExecCommand("SELECTNODE") }),
				iup.Item(iup.Attr("TITLE", "SELECT ELEM"), func(arg *iup.ItemAction) { stw.ExecCommand("SELECTELEM") }),
				iup.Item(iup.Attr("TITLE", "SELECT SECT"), func(arg *iup.ItemAction) { stw.ExecCommand("SELECTSECT") }),
				iup.Item(iup.Attr("TITLE", "FENCE"), func(arg *iup.ItemAction) { stw.ExecCommand("FENCE") }),
				iup.Item(iup.Attr("TITLE", "ERROR ELEM"), func(arg *iup.ItemAction) { stw.ExecCommand("ERRORELEM") }),
				iup.Separator(),
				iup.Item(iup.Attr("TITLE", "ADD LINE ELEM"), func(arg *iup.ItemAction) { stw.ExecCommand("ADDLINEELEM") }),
				iup.Item(iup.Attr("TITLE", "ADD PLATE ELEM"), func(arg *iup.ItemAction) { stw.ExecCommand("ADDPLATEELEM") }),
				iup.Item(iup.Attr("TITLE", "HATCH PLATE ELEM"), func(arg *iup.ItemAction) { stw.ExecCommand("HATCHPLATEELEM") }),
				iup.Item(iup.Attr("TITLE", "EDIT PLATE ELEM"), func(arg *iup.ItemAction) { stw.ExecCommand("EDITPLATEELEM") }),
				iup.Separator(),
				iup.Item(iup.Attr("TITLE", "MATCH PROP"), func(arg *iup.ItemAction) { stw.ExecCommand("MATCHPROP") }),
				iup.Separator(),
				iup.Item(iup.Attr("TITLE", "COPY ELEM"), func(arg *iup.ItemAction) { stw.ExecCommand("COPYELEM") }),
				iup.Item(iup.Attr("TITLE", "MOVE ELEM"), func(arg *iup.ItemAction) { stw.ExecCommand("MOVEELEM") }),
				iup.Item(iup.Attr("TITLE", "MOVE NODE"), func(arg *iup.ItemAction) { stw.ExecCommand("MOVENODE") }),
				iup.Separator(),
				iup.Item(iup.Attr("TITLE", "SEARCH ELEM"), func(arg *iup.ItemAction) { stw.ExecCommand("SEARCHELEM") }),
				iup.Item(iup.Attr("TITLE", "NODE TO ELEM"), func(arg *iup.ItemAction) { stw.ExecCommand("NODETOELEM") }),
				iup.Item(iup.Attr("TITLE", "ELEM TO NODE"), func(arg *iup.ItemAction) { stw.ExecCommand("ELEMTONODE") }),
				iup.Item(iup.Attr("TITLE", "CONNECTED"), func(arg *iup.ItemAction) { stw.ExecCommand("CONNECTED") }),
				iup.Item(iup.Attr("TITLE", "ON NODE"), func(arg *iup.ItemAction) { stw.ExecCommand("ONNODE") }),
				iup.Separator(),
				iup.Item(iup.Attr("TITLE", "NODE NO REFERENCE"), func(arg *iup.ItemAction) { stw.ExecCommand("NODENOREFERENCE") }),
				iup.Item(iup.Attr("TITLE", "ELEM SAME NODE"), func(arg *iup.ItemAction) { stw.ExecCommand("ELEMSAMENODE") }),
				iup.Item(iup.Attr("TITLE", "NODE DUPLICATION"), func(arg *iup.ItemAction) { stw.ExecCommand("NODEDUPLICATION") }),
				iup.Separator(),
				iup.Item(iup.Attr("TITLE", "CAT BY NODE"), func(arg *iup.ItemAction) { stw.ExecCommand("CATBYNODE") }),
				iup.Item(iup.Attr("TITLE", "JOIN LINE ELEM"), func(arg *iup.ItemAction) { stw.ExecCommand("JOINLINEELEM") }),
				iup.Separator(),
				iup.Item(iup.Attr("TITLE", "EXTRACT ARCLM"), func(arg *iup.ItemAction) { stw.ExecCommand("EXTRACTARCLM") }),
				iup.Separator(),
				iup.Item(iup.Attr("TITLE", "DIVIDE AT ONS"), func(arg *iup.ItemAction) { stw.ExecCommand("DIVIDEATONS") }),
				iup.Item(iup.Attr("TITLE", "DIVIDE AT MID"), func(arg *iup.ItemAction) { stw.ExecCommand("DIVIDEATMID") }),
				iup.Separator(),
				iup.Item(iup.Attr("TITLE", "INTERSECT"), func(arg *iup.ItemAction) { stw.ExecCommand("INTERSECT") }),
				iup.Item(iup.Attr("TITLE", "TRIM"), func(arg *iup.ItemAction) { stw.ExecCommand("TRIM") }),
				iup.Item(iup.Attr("TITLE", "EXTEND"), func(arg *iup.ItemAction) { stw.ExecCommand("EXTEND") }),
				iup.Separator(),
				iup.Item(iup.Attr("TITLE", "REACTION"), func(arg *iup.ItemAction) { stw.ExecCommand("REACTION") }),
			),
		),
		iup.SubMenu("TITLE=Tool",
			iup.Menu(
				iup.Item(
					iup.Attr("TITLE", "RC lst"),
					func(arg *iup.ItemAction) {
						st.StartTool("rclst/rclst.html")
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Fig2 Keyword"),
					func(arg *iup.ItemAction) {
						st.StartTool("fig2/fig2.html")
					},
				),
			),
		),
		iup.SubMenu("TITLE=Help",
			iup.Menu(
				iup.Item(
					iup.Attr("TITLE", "Release Note"),
					func(arg *iup.ItemAction) {
						ShowReleaseNote()
					},
				),
				iup.Item(
					iup.Attr("TITLE", "About"),
					func(arg *iup.ItemAction) {
						stw.ShowAbout()
					},
				),
			),
		),
	).SetName("main_menu")
	// coms1 := iup.Hbox()
	// coms1.SetAttribute("TABTITLE", "File")
	// for _, k := range []string{"OPEN", "SAVE", "READPGP"} {
	//     com := Commands[k]
	//     coms1.Append(stw.commandButton(com.Display, com, "120x30"))
	// }
	// coms2 := iup.Hbox()
	// coms2.SetAttribute("TABTITLE", "Utility")
	// for _, k := range []string{"DISTS", "MATCHPROP", "NODEDUPLICATION", "NODENOREFERENCE", "ELEMDUPLICATION", "ELEMSAMENODE"} {
	//     com := Commands[k]
	//     coms2.Append(stw.commandButton(com.Display, com, "120x30"))
	// }
	// coms3 := iup.Hbox()
	// coms3.SetAttribute("TABTITLE", "Conf")
	// for _, k := range []string{"CONFFREE", "CONFPIN", "CONFFIX", "CONFXYROLLER"} {
	//     com := Commands[k]
	//     coms3.Append(stw.commandButton(com.Display, com, "80x30"))
	// }
	// coms4 := iup.Hbox()
	// coms4.SetAttribute("TABTITLE", "Bond")
	// for _, k := range []string{"BONDPIN", "BONDRIGID", "TOGGLEBOND", "COPYBOND"} {
	//     com := Commands[k]
	//     coms4.Append(stw.commandButton(com.Display, com, "80x30"))
	// }
	// coms5 := iup.Hbox()
	// coms5.SetAttribute("TABTITLE", "Node")
	// for _, k := range []string{"MOVENODE", "MERGENODE"} {
	//     com := Commands[k]
	//     coms5.Append(stw.commandButton(com.Display, com, "100x30"))
	// }
	// coms6 := iup.Hbox()
	// coms6.SetAttribute("TABTITLE", "Elem")
	// for _, k := range []string{"COPYELEM", "MOVEELEM", "JOINLINEELEM", "JOINPLATEELEM"} {
	//     com := Commands[k]
	//     coms6.Append(stw.commandButton(com.Display, com, "100x30"))
	// }
	// coms7 := iup.Hbox()
	// coms7.SetAttribute("TABTITLE", "Add Elem")
	// for _, k := range []string{"ADDLINEELEM", "ADDPLATEELEM", "ADDPLATEELEMBYLINE", "HATCHPLATEELEM"} {
	//     com := Commands[k]
	//     coms7.Append(stw.commandButton(com.Display, com, "100x30"))
	// }
	// buttons := iup.Tabs(coms1, coms2, coms3, coms4, coms5, coms6, coms7,)
	stw.canv = iup.Canvas(
		"CANFOCUS=YES",
		"BGCOLOR=\"0 0 0\"",
		"BORDER=YES",
		// "DRAWSIZE=1920x1080",
		"EXPAND=YES",
		"NAME=canvas",
		func(arg *iup.CommonMap) {
			stw.cdcanv = cd.CreateCanvas(cd.CD_IUP, stw.canv)
			stw.dbuff = cd.CreateCanvas(cd.CD_DBUFFER, stw.cdcanv)
			stw.cdcanv.Foreground(cd.CD_WHITE)
			stw.cdcanv.Background(cd.CD_BLACK)
			stw.cdcanv.LineStyle(cd.CD_CONTINUOUS)
			stw.cdcanv.LineWidth(1)
			stw.cdcanv.Font(canvasFontFace, cd.CD_PLAIN, canvasFontSize)
			stw.dbuff.Foreground(cd.CD_WHITE)
			stw.dbuff.Background(cd.CD_BLACK)
			stw.dbuff.LineStyle(cd.CD_CONTINUOUS)
			stw.dbuff.LineWidth(1)
			stw.dbuff.Font(canvasFontFace, cd.CD_PLAIN, canvasFontSize)
			stw.dbuff.Activate()
			stw.dbuff.Flush()
		},
		func(arg *iup.CanvasResize) {
			stw.dbuff.Activate()
			if stw.frame != nil {
				stw.Redraw()
			} else {
				stw.dbuff.Flush()
			}
		},
		func(arg *iup.CanvasAction) {
			stw.dbuff.Activate()
			if stw.frame != nil {
				stw.Redraw()
			} else {
				stw.dbuff.Flush()
			}
		},
		func(arg *iup.CanvasDropFiles) {
			switch filepath.Ext(arg.FileName) {
			case ".inp", ".dxf":
				st.OpenFile(stw, arg.FileName, true)
				stw.Redraw()
			default:
				if stw.frame != nil {
					st.ReadFile(stw, arg.FileName)
				}
			}
		},
	)
	stw.cline = iup.Text(
		fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
		fmt.Sprintf("FGCOLOR=\"%s\"", labelFGColor),
		fmt.Sprintf("BGCOLOR=\"%s\"", commandBGColor),
		"CANFOCUS=YES",
		"EXPAND=HORIZONTAL",
		"BORDER=NO",
		"CARET=YES",
		"CUEBANNER=\"Command\"",
		"SIZE=150x10",
		"NAME=command",
		func(arg *iup.CommonKeyAny) {
			key := iup.KeyState(arg.Key)
			setprev := true
			switch key.Key() {
			case KEY_ENTER:
				stw.feedCommand()
				stw.completefunc = stw.CompleteFileName
				stw.complete = nil
			case KEY_ESCAPE:
				stw.cline.SetAttribute("VALUE", "")
				stw.completefunc = stw.CompleteFileName
				stw.complete = nil
				iup.SetFocus(stw.canv)
			case KEY_TAB:
				str := stw.cline.GetAttribute("VALUE")
				lis := strings.Split(str, " ")
				tmp := lis[len(lis)-1]
				if prevkey == KEY_TAB {
					if key.IsShift() {
						lis[len(lis)-1] = stw.PrevComplete(tmp)
					} else {
						lis[len(lis)-1] = stw.NextComplete(tmp)
					}
				} else {
					if stw.complete != nil {
						if stw.complete.Context(str) == complete.FileName {
							stw.completefunc = stw.CompleteFileName
						} else {
							stw.completefunc = func(v string) string {
								lis := stw.complete.CompleteWord(str)
								if len(lis) == 0 {
									return str
								} else {
									stw.completepos = 0
									stw.completes = lis
									return stw.completes[0]
								}
							}
						}
					}
					lis[len(lis)-1] = stw.completefunc(tmp)
				}
				stw.cline.SetAttribute("VALUE", strings.Join(lis, " "))
				stw.cline.SetAttribute("CARETPOS", "100")
				arg.Return = int32(iup.IGNORE)
			case KEY_UPARROW:
				if !(prevkey == KEY_UPARROW || prevkey == KEY_DOWNARROW) {
					clineinput = stw.cline.GetAttribute("VALUE")
				}
				stw.PrevCommand(clineinput)
				arg.Return = int32(iup.IGNORE)
			case KEY_DOWNARROW:
				if !(prevkey == KEY_UPARROW || prevkey == KEY_DOWNARROW) {
					clineinput = stw.cline.GetAttribute("VALUE")
				}
				stw.NextCommand(clineinput)
				arg.Return = int32(iup.IGNORE)
			case KEY_SPACE:
				val := stw.cline.GetAttribute("VALUE")
				if strings.Contains(val, " ") {
					// lis := strings.Split(val, " ")
					// tmp := lis[len(lis)-1]
					// if stw.complete != nil {
					// 	con := stw.complete.Context(val)
					// 	if con == complete.FileName {
					// 		stw.completefunc = stw.CompleteFileName
					// 	} else {
					// 		stw.completefunc = func(v string) string {
					// 			lis := stw.complete.CompleteWord(val)
					// 			if len(lis) == 0 {
					// 				return val
					// 			} else {
					// 				stw.completepos = 0
					// 				stw.completes = lis
					// 				return stw.completes[0]
					// 			}
					// 		}
					// 	}
					// }
					// lis[len(lis)-1] = stw.completefunc(tmp)
					// stw.cline.SetAttribute("VALUE", fmt.Sprintf("%s", strings.Join(lis, " ")))
					// stw.cline.SetAttribute("CARETPOS", "100")
					// arg.Return = int32(iup.IGNORE)
					break
				}
				if strings.HasPrefix(val, ":") {
					c, bang, usage, comp := st.ExModeComplete(val)
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
					str := fmt.Sprintf(":%s%s%s ", c, b, u)
					stw.cline.SetAttribute("VALUE", str)
					stw.cline.SetAttribute("CARETPOS", "100")
					if comp != nil {
						stw.complete = comp
						stw.complete.Chdir(stw.Cwd())
						stw.completefunc = func(v string) string {
							lis := comp.CompleteWord(str)
							if len(lis) == 0 {
								return str
							} else {
								stw.completepos = 0
								stw.completes = lis
								return stw.completes[0]
							}
						}
					}
				} else if strings.HasPrefix(val, "'") {
					c, usage, comp := st.Fig2KeywordComplete(val)
					var u string
					if usage {
						u = "?"
					} else {
						u = ""
					}
					str := fmt.Sprintf("'%s%s ", c, u)
					stw.cline.SetAttribute("VALUE", str)
					stw.cline.SetAttribute("CARETPOS", "100")
					if comp != nil {
						stw.complete = comp
						stw.complete.Chdir(stw.Cwd())
						stw.completefunc = func(v string) string {
							lis := comp.CompleteWord(str)
							if len(lis) == 0 {
								return str
							} else {
								stw.completepos = 0
								stw.completes = lis
								return stw.completes[0]
							}
						}
					}
				} else {
					stw.cline.SetAttribute("INSERT", " ")
				}
				arg.Return = int32(iup.IGNORE)
			case ':':
				val := stw.cline.GetAttribute("VALUE")
				if val == "" {
					stw.cline.SetAttribute("INSERT", ";")
				} else {
					stw.cline.SetAttribute("INSERT", ":")
				}
				arg.Return = int32(iup.IGNORE)
			case ';':
				val := stw.cline.GetAttribute("VALUE")
				if val == "" {
					stw.cline.SetAttribute("INSERT", ":")
					stw.completefunc = stw.CompleteExcommand
				} else {
					stw.cline.SetAttribute("INSERT", ";")
				}
				arg.Return = int32(iup.IGNORE)
			case '\'':
				val := stw.cline.GetAttribute("VALUE")
				if val == "" {
					stw.completefunc = stw.CompleteFig2Keyword
				}
			case '[':
				if key.IsCtrl() {
					stw.cline.SetAttribute("VALUE", "")
				}
			case 'N':
				if key.IsCtrl() {
					if !(prevkey == KEY_UPARROW || prevkey == KEY_DOWNARROW) {
						clineinput = stw.cline.GetAttribute("VALUE")
					}
					stw.NextCommand(clineinput)
					arg.Return = int32(iup.IGNORE)
					prevkey = KEY_DOWNARROW
					setprev = false
				}
			case 'O':
				if key.IsCtrl() {
					stw.Open()
				}
			case 'P':
				if key.IsCtrl() {
					if !(prevkey == KEY_UPARROW || prevkey == KEY_DOWNARROW) {
						clineinput = stw.cline.GetAttribute("VALUE")
					}
					stw.PrevCommand(clineinput)
					arg.Return = int32(iup.IGNORE)
					prevkey = KEY_UPARROW
					setprev = false
				}
			case 'I':
				if key.IsCtrl() {
					stw.Insert()
				}
			case 'M':
				if key.IsCtrl() {
					st.Reload(stw)
				}
			case 'S':
				if key.IsCtrl() {
					stw.Save()
				}
			case 'A':
				if key.IsCtrl() {
					stw.ShowAll()
				}
			case 'R':
				if key.IsCtrl() {
					st.ReadAll(stw)
				}
			case 'E':
				if key.IsCtrl() {
					stw.EditInp()
				}
			case 'X':
				if key.IsCtrl() {
					if stw.sectiondlg != nil {
						iup.SetFocus(stw.sectiondlg)
					} else {
						stw.SectionDialog()
					}
				}
			}
			if setprev {
				prevkey = key.Key()
			}
		},
	)
	stw.coord = iup.Text(
		"CANFOCUS=NO",
		fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
		fmt.Sprintf("FGCOLOR=\"%s\"", labelFGColor),
		fmt.Sprintf("BGCOLOR=\"%s\"", historyBGColor),
		"READONLY=YES",
		"BORDER=NO",
		"CARET=NO",
		"SIZE=150x10",
		"NAME=coord",
	)
	stw.hist = iup.Text(
		"CANFOCUS=NO",
		fmt.Sprintf("BGCOLOR=\"%s\"", historyBGColor),
		"EXPAND=HORIZONTAL",
		"READONLY=YES",
		"FORMATTING=YES",
		"MULTILINE=YES",
		"AUTOHIDE=YES",
		"BORDER=NO",
		"CARET=NO",
		"SIZE=x50",
		"NAME=history",
	)
	stw.cname = iup.Text(
		fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
		fmt.Sprintf("FGCOLOR=\"%s\"", commandFGColor),
		fmt.Sprintf("BGCOLOR=\"%s\"", historyBGColor),
		"CANFOCUS=NO",
		"EXPAND=NO",
		"READONLY=YES",
		"MULTILINE=NO",
		"BORDER=NO",
		"CARET=NO",
		"SIZE=80x10",
		"VALUE=\"SELECT\"",
		"NAME=cname",
	)
	stw.formattag = iup.User(fmt.Sprintf("FGCOLOR=\"%s\"", labelFGColor),
		fmt.Sprintf("BGCOLOR=\"%s\"", historyBGColor),
		fmt.Sprintf("FONTFACE=%s", commandFontFace),
		fmt.Sprintf("FONTSIZE=%s", commandFontSize))
	iup.SetHandle("tag", stw.formattag)
	stw.hist.SetAttribute("ADDFORMATTAG", "tag")
	// DataLabel
	pers := iup.Text(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
		fmt.Sprintf("FGCOLOR=\"%s\"", labelFGColor),
		fmt.Sprintf("BGCOLOR=\"%s\"", labelBGColor),
		"VALUE=\"  PERSPECTIVE\"",
		"CANFOCUS=NO",
		"READONLY=YES",
		"BORDER=NO",
		fmt.Sprintf("SIZE=%dx%d", datalabelwidth+datatextwidth, dataheight))
	pers.SetCallback(func(arg *iup.MouseButton) {
		if stw.frame != nil {
			switch arg.Button {
			case BUTTON_LEFT:
				if arg.Pressed == 0 { // Released
					if pers.GetAttribute("VALUE") == "  PERSPECTIVE" {
						stw.frame.View.Perspective = false
						pers.SetAttribute("VALUE", "  AXONOMETRIC")
					} else {
						stw.frame.View.Perspective = true
						pers.SetAttribute("VALUE", "  PERSPECTIVE")
					}
					stw.Redraw()
					iup.SetFocus(stw.canv)
				}
			}
		}
	})
	vlabels := iup.Vbox(iup.Hbox(datasectionlabel("VIEW")),
		iup.Hbox(datalabel("GFACT"), stw.Labels["GFACT"]),
		iup.Hbox(pers),
		iup.Hbox(datasectionlabel("DISTS")),
		iup.Hbox(datalabel("R"), stw.Labels["DISTR"]),
		iup.Hbox(datalabel("L"), stw.Labels["DISTL"]),
		iup.Hbox(datasectionlabel("ANGLE")),
		iup.Hbox(datalabel("PHI"), stw.Labels["PHI"]),
		iup.Hbox(datalabel("THETA"), stw.Labels["THETA"]),
		iup.Hbox(datasectionlabel("FOCUS")),
		iup.Hbox(datalabel("X"), stw.Labels["FOCUSX"]),
		iup.Hbox(datalabel("Y"), stw.Labels["FOCUSY"]),
		iup.Hbox(datalabel("Z"), stw.Labels["FOCUSZ"]),
		iup.Hbox(datasectionlabel("CENTER")),
		iup.Hbox(datalabel("X"), stw.Labels["CENTERX"]),
		iup.Hbox(datalabel("Y"), stw.Labels["CENTERY"]),
		"MARGIN=0x0",
	)
	tgelem := iup.Vbox(datasectionlabel("ETYPE"),
		stw.Labels["COLUMN"],
		iup.Hbox(stw.Labels["COLUMN_NZ"],
			stw.Labels["COLUMN_QX"],
			stw.Labels["COLUMN_QY"],
			stw.Labels["COLUMN_MZ"],
			stw.Labels["COLUMN_MX"],
			stw.Labels["COLUMN_MY"]),
		stw.Labels["GIRDER"],
		iup.Hbox(stw.Labels["GIRDER_NZ"],
			stw.Labels["GIRDER_QX"],
			stw.Labels["GIRDER_QY"],
			stw.Labels["GIRDER_MZ"],
			stw.Labels["GIRDER_MX"],
			stw.Labels["GIRDER_MY"]),
		stw.Labels["BRACE"],
		iup.Hbox(stw.Labels["BRACE_NZ"],
			stw.Labels["BRACE_QX"],
			stw.Labels["BRACE_QY"],
			stw.Labels["BRACE_MZ"],
			stw.Labels["BRACE_MX"],
			stw.Labels["BRACE_MY"]),
		iup.Hbox(stw.Labels["WALL"], stw.switchLabel(st.WALL), stw.Labels["WBRACE"]),
		iup.Hbox(stw.Labels["WALL_NZ"],
			stw.Labels["WALL_QX"],
			stw.Labels["WALL_QY"],
			stw.Labels["WALL_MZ"],
			stw.Labels["WALL_MX"],
			stw.Labels["WALL_MY"]),
		iup.Hbox(stw.Labels["SLAB"], stw.switchLabel(st.SLAB), stw.Labels["SBRACE"]),
		iup.Hbox(stw.Labels["SLAB_NZ"],
			stw.Labels["SLAB_QX"],
			stw.Labels["SLAB_QY"],
			stw.Labels["SLAB_MZ"],
			stw.Labels["SLAB_MX"],
			stw.Labels["SLAB_MY"]))
	tgncap := iup.Vbox(datasectionlabel("NODE CAPTION"),
		stw.Labels["NC_NUM"],
		stw.Labels["NC_WEIGHT"],
		iup.Hbox(stw.Labels["NC_DX"],
			stw.Labels["NC_DY"],
			stw.Labels["NC_DZ"],
			stw.Labels["NC_TX"],
			stw.Labels["NC_TY"],
			stw.Labels["NC_TZ"]),
		iup.Hbox(stw.Labels["NC_RX"],
			stw.Labels["NC_RY"],
			stw.Labels["NC_RZ"],
			stw.Labels["NC_MX"],
			stw.Labels["NC_MY"],
			stw.Labels["NC_MZ"]))
	tgecap := iup.Vbox(datasectionlabel("ELEM CAPTION"),
		stw.Labels["EC_NUM"],
		stw.Labels["EC_SECT"],
		stw.Labels["SRCAN_RATE"],
		iup.Hbox(stw.Labels["SRCAN_L"], stw.Labels["SRCAN_S"], stw.Labels["SRCAN_Q"], stw.Labels["SRCAN_M"]))
	tgcolmode := iup.Vbox(datasectionlabel("COLOR MODE"), stw.Labels["COLORMODE"])
	dlperiod := datalabel("PERIOD")
	dlperiod.SetCallback(func(arg *iup.MouseButton) {
		if arg.Button == BUTTON_LEFT && arg.Pressed == 0 {
			stw.PeriodDialog(stw.Labels["PERIOD"].GetAttribute("VALUE"))
		}
	})
	tgparam := iup.Vbox(datasectionlabel("PARAMETER"),
		iup.Hbox(dlperiod, stw.Labels["PERIOD"]),
		iup.Hbox(datalabel("GAXIS"), stw.Labels["GAXISSIZE"]),
		iup.Hbox(datalabel("EAXIS"), stw.Labels["EAXISSIZE"]),
		iup.Hbox(datalabel("BOND"), stw.Labels["BONDSIZE"]),
		iup.Hbox(datalabel("CONF"), stw.Labels["CONFSIZE"]),
		iup.Hbox(datalabel("DFACT"), stw.Labels["DFACT"]),
		iup.Hbox(datalabel("QFACT"), stw.Labels["QFACT"]),
		iup.Hbox(datalabel("MFACT"), stw.Labels["MFACT"]),
		iup.Hbox(datalabel("RFACT"), stw.Labels["RFACT"]))
	tgshow := iup.Vbox(datasectionlabel("SHOW"),
		stw.Labels["GAXIS"],
		stw.Labels["EAXIS"],
		stw.Labels["BOND"],
		stw.Labels["CONF"],
		stw.Labels["PHINGE"],
		stw.Labels["YIELD"],
		stw.Labels["KIJUN"],
		stw.Labels["DEFORMATION"])
	tgrang := iup.Vbox(datasectionlabel("RANGE"),
		iup.Hbox(datalabel("Xmax"), stw.Labels["XMAX"]),
		iup.Hbox(datalabel("Xmin"), stw.Labels["XMIN"]),
		iup.Hbox(datalabel("Ymax"), stw.Labels["YMAX"]),
		iup.Hbox(datalabel("Ymin"), stw.Labels["YMIN"]),
		iup.Hbox(datalabel("Zmax"), stw.Labels["ZMAX"]),
		iup.Hbox(datalabel("Zmin"), stw.Labels["ZMIN"]))
	stw.PropertyDialog()
	stw.SideBar = iup.Tabs(iup.Vbox(vlabels, tgrang, "TABTITLE=View"),
		iup.Vbox(tgcolmode, tgparam, tgshow, tgncap, tgecap, tgelem, "TABTITLE=Show"),
		iup.Vbox(iup.Hbox(propertylabel("LINE"), stw.Selected[0]),
			iup.Hbox(propertylabel(" (LENGTH)"), stw.Selected[1]),
			iup.Hbox(propertylabel("PLATE"), stw.Selected[2]),
			iup.Hbox(propertylabel(" (AREA)"), stw.Selected[3]),
			iup.Hbox(propertylabel("CODE"), stw.Props[0]),
			iup.Hbox(propertylabel("SECTION"), stw.Props[1]),
			iup.Hbox(propertylabel("ETYPE"), stw.Props[2]),
			iup.Hbox(propertylabel("ENODS"), stw.Props[3]),
			iup.Hbox(propertylabel("ENOD"), stw.Props[4]),
			iup.Hbox(propertylabel(""), stw.Props[5]),
			iup.Hbox(propertylabel(""), stw.Props[6]),
			iup.Hbox(propertylabel(""), stw.Props[7]), "TABTITLE=Property"))
	stw.Dlg = iup.Dialog(
		iup.Attrs(
			"MENU", "main_menu",
			"TITLE", "st",
			"SHRINK", "YES",
			"SIZE", windowSize,
			"FGCOLOR", labelFGColor,
			"BGCOLOR", labelBGColor,
			"ICON", "STICON",
		),
		iup.Vbox(
			// buttons,
			iup.Hbox(
				stw.SideBar,
				iup.Vbox(stw.canv,
					iup.Hbox(stw.hist),
					iup.Hbox(stw.cname, stw.cline, stw.coord)),
			),
		),
		func(arg *iup.DialogClose) {
			if stw.changed {
				if stw.Yn("CHANGED", "変更を保存しますか") {
					stw.SaveAS()
				} else {
					return
				}
			}
			stw.SaveRecent()
			arg.Return = iup.CLOSE
		},
		func(arg *iup.CommonGetFocus) {
			if stw.frame != nil {
				if stw.InpModified {
					stw.InpModified = false
					if stw.Yn("RELOAD", fmt.Sprintf(".inpをリロードしますか？")) {
						st.Reload(stw)
					}
				}
				stw.Redraw()
			}
		},
	)
	stw.Dlg.Map()
	w, h := stw.cdcanv.GetSize()
	stw.CanvasSize = []float64{float64(w), float64(h)}
	stw.dbuff.TextAlignment(DefaultTextAlignment)
	stw.papersize = st.A4_TATE
	stw.textBox = make(map[string]*TextBox, 0)
	stw.textBox["PAGETITLE"] = NewTextBox()
	stw.textBox["PAGETITLE"].Font.Size = 16
	stw.textBox["PAGETITLE"].position = []float64{30.0, stw.CanvasSize[1] - 30.0}
	stw.textBox["TITLE"] = NewTextBox()
	stw.textBox["TITLE"].position = []float64{30.0, stw.CanvasSize[1] - 80.0}
	stw.textBox["TEXT"] = NewTextBox()
	stw.textBox["TEXT"].position = []float64{120.0, 65.0}
	iup.SetHandle("mainwindow", stw.Dlg)
	stw.EscapeAll()
	stw.changed = false
	stw.comhist = make([]string, CommandHistorySize)
	comhistpos = -1
	stw.SetCoord(0.0, 0.0, 0.0)
	stw.ReadRecent()
	st.ShowRecent(stw)
	stw.SetCommandHistory()
	StartLogging()
	err := stw.ReadPgp(pgpfile)
	if err != nil {
		stw.SetDefaultPgp()
	}
	stw.New()
	// stw.ShowLogo(3*time.Second)
	stw.completefunc = stw.CompleteFileName
	if rcfn := filepath.Join(stw.Cwd(), ResourceFileName); st.FileExists(rcfn) {
		stw.ReadResource(rcfn)
	}
	return stw
}

// }}}

func (stw *Window) Frame() *st.Frame {
	return stw.frame
}
func (stw *Window) SetFrame(frame *st.Frame) {
	stw.frame = frame
}

func (stw *Window) FocusCanv() {
	iup.SetFocus(stw.canv)
}

func (stw *Window) Chdir(dir string) error {
	if _, err := os.Stat(dir); err != nil {
		return err
	} else {
		stw.SetCwd(dir)
		return nil
	}
}

// New// {{{
func (stw *Window) New() {
	var s *st.Show
	frame := st.NewFrame()
	if stw.frame != nil {
		s = stw.frame.Show
	}
	w, h := stw.cdcanv.GetSize()
	stw.CanvasSize = []float64{float64(w), float64(h)}
	frame.View.Center[0] = stw.CanvasSize[0] * 0.5
	frame.View.Center[1] = stw.CanvasSize[1] * 0.5
	if s != nil {
		stw.frame.Show = s
	}
	stw.frame = frame
	stw.Dlg.SetAttribute("TITLE", "***")
	stw.frame.Home = stw.Home()
	stw.LinkTextValue()
	stw.changed = false
	stw.Redraw()
}

// }}}

// Open// {{{
func (stw *Window) Open() {
	if name, ok := iup.GetOpenFile(stw.Cwd(), "*.inp"); ok {
		err := st.OpenFile(stw, name, true)
		if err != nil {
			fmt.Println(err)
		}
		stw.Redraw()
	}
}

// func UpdateInps(dirname string, brk chan bool) {
func UpdateInps(dirname string) {
	tmp, err := filepath.Glob(dirname + "/**/**/*.inp")
	if err != nil {
		return
	}
	Inps = tmp
	// 	Inps = make([]string, 0)
	// 	SearchingInps = true
	// 	SearchingInpsDone = make(chan bool)
	// 	inpch := st.SearchInp(dirname)
	// createinps:
	// 	for {
	// 		select {
	// 		case <-brk:
	// 			break createinps
	// 		case fn := <-inpch:
	// 			if fn == "" {
	// 				break createinps
	// 			} else {
	// 				Inps = append(Inps, fn)
	// 			}
	// 		}
	// 	}
	// 	SearchingInps = false
	// 	SearchingInpsDone <- true
}

func (stw *Window) SearchInp() {
	var dlg, basedir, result, text *iup.Handle
	basedir = iup.Text(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
		fmt.Sprintf("FGCOLOR=\"%s\"", labelOFFColor),
		fmt.Sprintf("BGCOLOR=\"%s\"", labelBGColor),
		fmt.Sprintf("VALUE=\"%s\"", stw.Home()),
		"BORDER=NO",
		"CANFOCUS=NO",
		fmt.Sprintf("SIZE=%dx%d", datalabelwidth*8, dataheight),
	)
	result = iup.List(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
		fmt.Sprintf("FGCOLOR=\"%s\"", labelFGColor),
		fmt.Sprintf("BGCOLOR=\"%s\"", labelBGColor),
		"VALUE=\"\"",
		"CANFOCUS=NO",
		"BORDER=NO",
		fmt.Sprintf("SIZE=%dx%d", datalabelwidth*8, dataheight*20),
	)
	text = iup.Text(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
		fmt.Sprintf("FGCOLOR=\"%s\"", labelFGColor),
		fmt.Sprintf("BGCOLOR=\"%s\"", commandBGColor),
		"VALUE=\"\"",
		"BORDER=NO",
		"ALIGNMENT=ALEFT",
		fmt.Sprintf("SIZE=%dx%d", datalabelwidth*8, dataheight),
	)
	// brk := make(chan bool)
	searchinginps_r := false
	updateresult := func() {
		result.SetAttribute("REMOVEITEM", "ALL")
		word := text.GetAttribute("VALUE")
		re := regexp.MustCompile(word)
		for _, fn := range Inps {
			if re.MatchString(fn) {
				result.SetAttribute("APPENDITEM", fn)
			}
		}
		if SearchingInps {
			result.SetAttribute("APPENDITEM", "Searching...")
		}
	}
	openfile := func(fn string) {
		err := st.OpenFile(stw, fn, true)
		if err != nil {
			st.ErrorMessage(stw, err, st.ERROR)
		}
		// if searchinginps_r {
		// 	brk <- true
		// }
		dlg.Destroy()
	}
	// go func() {
	// searchinp:
	// 	for {
	// 		if !SearchingInps {
	// 			break
	// 		}
	// 		select {
	// 		case <-SearchingInpsDone:
	// 			updateresult()
	// 			break searchinp
	// 		}
	// 	}
	// }()
	basedir.SetCallback(func(arg *iup.CommonKillFocus) {
		updateresult()
	})
	text.SetCallback(func(arg *iup.ValueChanged) {
		updateresult()
	})
	text.SetCallback(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		switch key.Key() {
		case KEY_ENTER:
			val := result.GetAttribute("VALUE")
			if val == "0" {
				val = "1"
			}
			openfile(result.GetAttribute(val))
		case KEY_ESCAPE:
			// if searchinginps_r {
			// 	brk <- true
			// }
			dlg.Destroy()
		case 'J':
			if key.IsCtrl() {
				tmp := result.GetAttribute("VALUE")
				val, _ := strconv.ParseInt(tmp, 10, 64)
				tmp = result.GetAttribute("COUNT")
				size, _ := strconv.ParseInt(tmp, 10, 64)
				next := val + 1
				if next > size {
					next -= size
				}
				result.SetAttribute("VALUE", fmt.Sprintf("%d", next))
			}
		case 'K':
			if key.IsCtrl() {
				tmp := result.GetAttribute("VALUE")
				val, _ := strconv.ParseInt(tmp, 10, 64)
				tmp = result.GetAttribute("COUNT")
				size, _ := strconv.ParseInt(tmp, 10, 64)
				if val == 0 {
					result.SetAttribute("VALUE", fmt.Sprintf("%d", size))
				} else {
					next := val - 1
					if next <= 0 {
						next += size
					}
					result.SetAttribute("VALUE", fmt.Sprintf("%d", next))
				}
			}
		case 'R':
			if key.IsCtrl() {
				if !SearchingInps {
					result.SetAttribute("REMOVEITEM", "ALL")
					result.SetAttribute("1", "Searching...")
					searchinginps_r = true
					UpdateInps(basedir.GetAttribute("VALUE"))
					// go UpdateInps(basedir.GetAttribute("VALUE"), brk)
					// go func() {
					// searchinp_r:
					// 	for {
					// 		if !searchinginps_r {
					// 			break
					// 		}
					// 		select {
					// 		case <-SearchingInpsDone:
					// 			updateresult()
					// 			searchinginps_r = false
					// 			break searchinp_r
					// 		}
					// 	}
					// }()
				}
			}
		}
	})
	result.SetCallback(func(arg *iup.ListDblclick) {
		openfile(arg.Text)
	})
	dlg = iup.Dialog(iup.Vbox(basedir, result, text))
	dlg.SetAttribute("TITLE", "Search Inp")
	dlg.SetAttribute("PARENTDIALOG", "mainwindow")
	dlg.Popup(iup.CENTER, iup.CENTER)
	iup.SetFocus(text)
}

func (stw *Window) Insert() {
	stw.ExecCommand("INSERT")
}

func (stw *Window) OpenDxf() {
	if name, ok := iup.GetOpenFile(stw.Cwd(), "*.dxf"); ok {
		err := st.OpenFile(stw, name, true)
		if err != nil {
			fmt.Println(err)
		}
		stw.Redraw()
	}
}

func (stw *Window) WatchFile(fn string) {
	var err error
	if watcher != nil {
		watcher.Close()
	}
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		st.ErrorMessage(stw, err, st.ERROR)
	}
	read := true
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if read {
					if passwatcher {
						passwatcher = false
					} else {
						if event.Op&fsnotify.Remove == fsnotify.Remove {
							stw.InpModified = true
							stw.WatchFile(fn)
						} else if event.Op&fsnotify.Write == fsnotify.Write {
							stw.InpModified = true
						}
					}
				}
				read = !read
			case err := <-watcher.Errors:
				st.ErrorMessage(stw, err, st.ERROR)
			}
		}
	}()
	err = watcher.Add(fn)
	if err != nil {
		st.ErrorMessage(stw, err, st.ERROR)
	}
}

// }}}

// Save// {{{
func (stw *Window) Save() {
	st.SaveFile(stw, filepath.Join(stw.Home(), "hogtxt.inp"))
}

func (stw *Window) SaveAS() {
	var err error
	if name, ok := iup.GetSaveFile(filepath.Dir(stw.frame.Path), "*.inp"); ok {
		fn := st.Ce(name, ".inp")
		err = st.SaveFile(stw, fn)
		if err == nil && fn != stw.frame.Path {
			stw.Copylsts(name)
			st.Rebase(stw, fn)
		}
	}
}

func (stw *Window) Copylsts(name string) {
	if stw.Yn("SAVE AS", ".lst, .fig2, .kjnファイルがあればコピーしますか?") {
		for _, ext := range []string{".lst", ".fig2", ".kjn"} {
			src := st.Ce(stw.frame.Path, ext)
			dst := st.Ce(name, ext)
			if st.FileExists(src) {
				err := st.CopyFile(src, dst)
				if err == nil {
					stw.addHistory(fmt.Sprintf("COPY: %s", dst))
				}
			}
		}
	}
}

func (stw *Window) SaveFileSelected(fn string) error {
	var v *st.View
	els := stw.selectElem
	if !stw.frame.View.Perspective {
		v = stw.frame.View.Copy()
		stw.frame.View.Gfact = 1.0
		stw.frame.View.Perspective = true
		for _, n := range stw.frame.Nodes {
			stw.frame.View.ProjectNode(n)
		}
		xmin, xmax, ymin, ymax := stw.Bbox()
		w, h := stw.cdcanv.GetSize()
		scale := math.Min(float64(w)/(xmax-xmin), float64(h)/(ymax-ymin)) * stw.CanvasFitScale()
		stw.frame.View.Dists[1] *= scale
	}
	passwatcher = true
	err := st.WriteInp(fn, stw.frame.View, stw.frame.Ai, els)
	if v != nil {
		stw.frame.View = v
	}
	if err != nil {
		return err
	}
	st.ErrorMessage(stw, errors.New(fmt.Sprintf("SAVE: %s", fn)), st.INFO)
	stw.changed = false
	return nil
}

// }}}

func (stw *Window) Close(force bool) {
	if !force && stw.changed {
		if stw.Yn("CHANGED", "変更を保存しますか") {
			stw.SaveAS()
		} else {
			return
		}
	}
	stw.SaveRecent()
	stw.Dlg.Destroy()
}

// Read// {{{
func (stw *Window) Read() {
	if stw.frame != nil {
		if name, ok := iup.GetOpenFile("", ""); ok {
			err := st.ReadFile(stw, name)
			if err != nil {
				st.ErrorMessage(stw, err, st.ERROR)
			}
		}
	}
}

func (stw *Window) AddPropAndSect(filename string) error {
	if stw.frame != nil {
		err := stw.frame.AddPropAndSect(filename, true)
		return err
	} else {
		return errors.New("NO FRAME")
	}
}

// }}}

// Print
func SetPrinter(name string) (*cd.Canvas, error) {
	pcanv := cd.CreatePrinter(cd.CD_PRINTER, fmt.Sprintf("%s -d", name))
	if pcanv == nil {
		return nil, errors.New("SetPrinter: cannot create canvas")
	}
	pcanv.Background(cd.CD_WHITE)
	pcanv.Foreground(cd.CD_BLACK)
	pcanv.LineStyle(cd.CD_CONTINUOUS)
	pcanv.LineWidth(1)
	pcanv.Font(printFontFace, cd.CD_PLAIN, printFontSize)
	return pcanv, nil
}

func (stw *Window) FittoPrinter(pcanv *cd.Canvas) (*st.View, float64, error) {
	v := stw.frame.View.Copy()
	pw, ph := pcanv.GetSize() // seems to be [mm]/25.4*[dpi]*2
	w0, h0 := stw.dbuff.GetSize()
	stw.currentCanvas = stw.dbuff
	w, h, err := stw.CanvasPaperSize()
	if err != nil {
		return v, 0.0, err
	}
	factor := math.Min(float64(pw)/w, float64(ph)/h)
	stw.CanvasSize[0] = float64(pw)
	stw.CanvasSize[1] = float64(ph)
	stw.frame.View.Gfact *= factor
	stw.frame.View.Center[0] = float64(pw)*0.5 + factor*(stw.frame.View.Center[0]-0.5*float64(w0))
	stw.frame.View.Center[1] = float64(ph)*0.5 + factor*(stw.frame.View.Center[1]-0.5*float64(h0))
	stw.frame.Show.ConfSize *= factor
	stw.frame.Show.BondSize *= factor
	stw.frame.Show.KijunSize = 150
	stw.frame.Show.MassSize *= factor
	for _, m := range stw.frame.Measures {
		m.ArrowSize = 75.0
	}
	for i := 0; i < 2; i++ {
		stw.textBox["PAGETITLE"].position[i] *= factor
		stw.textBox["TITLE"].position[i] *= factor
		stw.textBox["TEXT"].position[i] *= factor
	}
	for _, t := range stw.textBox {
		for i := 0; i < 2; i++ {
			t.position[i] *= factor
		}
	}
	return v, factor, nil
}

func (stw *Window) CanvasPaperSize() (float64, float64, error) {
	w, h := stw.currentCanvas.GetSize()
	length := math.Min(float64(w), float64(h)) * 0.9
	val := 1.0 / math.Sqrt(2)
	switch stw.papersize {
	default:
		return 0.0, 0.0, errors.New("unknown papersize")
	case st.A4_TATE, st.A3_TATE:
		return length * val, length, nil
	case st.A4_YOKO, st.A3_YOKO:
		return length, length * val, nil
	}
}

func (stw *Window) Print() {
	if stw.frame == nil {
		return
	}
	pcanv, err := SetPrinter(stw.frame.Name)
	if err != nil {
		st.ErrorMessage(stw, err, st.ERROR)
		return
	}
	v, factor, err := stw.FittoPrinter(pcanv)
	if err != nil {
		st.ErrorMessage(stw, err, st.ERROR)
		return
	}
	// PlateEdgeColor = cd.CD_BLACK
	// BondColor = cd.CD_BLACK
	// ConfColor = cd.CD_BLACK
	// MomentColor = cd.CD_BLACK
	// KijunColor = cd.CD_BLACK
	// MeasureColor = cd.CD_BLACK
	// StressTextColor = cd.CD_BLACK
	// YieldedTextColor = cd.CD_BLACK
	// BrittleTextColor = cd.CD_BLACK
	switch stw.frame.Show.ColorMode {
	default:
		stw.DrawFrame(pcanv, stw.frame.Show.ColorMode, false)
	case st.ECOLOR_WHITE:
		stw.frame.Show.ColorMode = st.ECOLOR_BLACK
		stw.DrawFrame(pcanv, st.ECOLOR_BLACK, false)
		stw.frame.Show.ColorMode = st.ECOLOR_WHITE
	}
	stw.DrawTexts(pcanv, true)
	pcanv.Kill()
	w, h := stw.cdcanv.GetSize()
	stw.CanvasSize = []float64{float64(w), float64(h)}
	stw.frame.Show.ConfSize /= factor
	stw.frame.Show.BondSize /= factor
	stw.frame.Show.KijunSize = 12.0
	stw.frame.Show.MassSize /= factor
	for _, m := range stw.frame.Measures {
		m.ArrowSize = 6.0
	}
	stw.frame.View = v
	for i := 0; i < 2; i++ {
		stw.textBox["PAGETITLE"].position[i] /= factor
		stw.textBox["TITLE"].position[i] /= factor
		stw.textBox["TEXT"].position[i] /= factor
	}
	for _, t := range stw.textBox {
		for i := 0; i < 2; i++ {
			t.position[i] /= factor
		}
	}
	// PlateEdgeColor = defaultPlateEdgeColor
	// BondColor = defaultBondColor
	// ConfColor = defaultConfColor
	// MomentColor = defaultMomentColor
	// KijunColor = defaultKijunColor
	// MeasureColor = defaultMeasureColor
	// StressTextColor = defaultStressTextColor
	// YieldedTextColor = defaultYieldedTextColor
	// BrittleTextColor = defaultBrittleTextColor
	stw.Redraw()
}

func (stw *Window) PrintFig2(filename string) error {
	if stw.frame == nil {
		return errors.New("PrintFig2: no frame opened")
	}
	pcanv, err := SetPrinter(stw.frame.Name)
	if err != nil {
		return err
	}
	v, factor, err := stw.FittoPrinter(pcanv)
	if err != nil {
		return err
	}
	tmp := make([][]string, 0)
	err = st.ParseFile(filename, func(words []string) error {
		var err error
		first := strings.ToUpper(words[0])
		if strings.HasPrefix(first, "#") {
			return nil
		}
		switch first {
		default:
			tmp = append(tmp, words)
		case "PAGE", "FIGURE":
			err = stw.ParseFig2(pcanv, tmp)
			tmp = [][]string{words}
		}
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	err = stw.ParseFig2(pcanv, tmp)
	if err != nil {
		return err
	}
	pcanv.Kill()
	stw.frame.Show.ConfSize /= factor
	stw.frame.Show.BondSize /= factor
	stw.frame.View = v
	stw.Redraw()
	return nil
}

func (stw *Window) ReadFig2(filename string) error {
	if stw.frame == nil {
		return errors.New("ReadFig2: no frame opened")
	}
	tmp := make([][]string, 0)
	var err error
	err = st.ParseFile(filename, func(words []string) error {
		var err error
		first := strings.ToUpper(words[0])
		if strings.HasPrefix(first, "#") {
			return nil
		}
		switch first {
		default:
			tmp = append(tmp, words)
		case "PAGE", "FIGURE":
			err = stw.ParseFig2(stw.dbuff, tmp)
			tmp = [][]string{words}
		}
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	err = stw.ParseFig2(stw.dbuff, tmp)
	if err != nil {
		return err
	}
	stw.Redraw()
	return nil
}

func (stw *Window) ParseFig2(pcanv *cd.Canvas, lis [][]string) error {
	var err error
	if len(lis) == 0 || len(lis[0]) < 1 {
		return nil
	}
	first := strings.ToUpper(lis[0][0])
	switch first {
	case "PAGE":
		err = stw.ParseFig2Page(pcanv, lis)
	case "FIGURE":
		err = stw.ParseFig2Page(pcanv, lis)
	}
	return err
}

func (stw *Window) ParseFig2Page(pcanv *cd.Canvas, lis [][]string) error {
	for _, txt := range lis {
		if len(txt) < 1 {
			continue
		}
		var un bool
		if strings.HasPrefix(txt[0], "!") {
			un = true
			txt[0] = txt[0][1:]
		} else {
			un = false
		}
		err := st.Fig2Keyword(stw, txt, un)
		if err != nil {
			return err
		}
	}
	stw.DrawFrame(pcanv, stw.frame.Show.ColorMode, false)
	stw.DrawTexts(pcanv, true)
	pcanv.Flush()
	return nil
}

func (stw *Window) EditInp() {
	if stw.frame != nil {
		cmd := exec.Command("cmd", "/C", "start", stw.frame.Path)
		cmd.Start()
	}
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
		cand := filepath.Join(stw.Home(), fn[:pos1], fn[:pos2], fn)
		if st.FileExists(cand) {
			return cand, nil
		} else {
			return fn, errors.New(fmt.Sprintf("File not fount %s", fn))
		}
	}
}

// Help// {{{
func ShowReleaseNote() {
	cmd := exec.Command("cmd", "/C", "start", releasenote)
	cmd.Start()
}

func (stw *Window) ShowLogo(t time.Duration) {
	w, h := stw.dbuff.GetSize()
	STLOGO.position = []float64{float64(w) * 0.5, float64(h) * 0.5}
	STLOGO.Show()
	// go func() {
	// logo:
	// 	for {
	// 		select {
	// 		case <-time.After(t):
	// 			stw.HideLogo()
	// 			stw.Redraw()
	// 			break logo
	// 		}
	// 	}
	// }()
}

func (stw *Window) HideLogo() {
	STLOGO.Hide()
}

func (stw *Window) ShowAbout() {
	dlg := iup.MessageDlg(fmt.Sprintf("FONTFACE=%s", commandFontFace),
		fmt.Sprintf("FONTSIZE=%s", commandFontSize))
	dlg.SetAttribute("TITLE", "バージョン情報")
	if stw.frame != nil {
		dlg.SetAttribute("VALUE", fmt.Sprintf("VERSION: %s\n%s\n\nNAME\t: %s\nPROJECT\t: %s\nPATH\t: %s", stw.Version, stw.Modified, stw.frame.Name, stw.frame.Project, stw.frame.Path))
	} else {
		dlg.SetAttribute("VALUE", fmt.Sprintf("VERSION: %s\n%s\n\nNAME\t: -\nPROJECT\t: -\nPATH\t: -", stw.Version, stw.Modified))
	}
	dlg.Popup(iup.CENTER, iup.CENTER)
}

// }}}

// Font// {{{
func (stw *Window) SetFont(mode int) {
	dlg := iup.FontDlg()

	dlg.SetAttribute("VALUE", stw.cline.GetAttribute("FONT"))
	dlg.SetAttribute("COLOR", stw.cline.GetAttribute("FGCOLOR"))
	dlg.Popup(iup.CENTER, iup.CENTER)

	if dlg.GetInt("STATUS") == 1 {
		switch mode {
		case FONT_COMMAND:
			ff := strings.Split(dlg.GetAttribute("VALUE"), ",")
			commandFontFace = ff[0]
			commandFontSize = strings.Trim(ff[1], " ")
			labelFGColor = dlg.GetAttribute("COLOR")
			stw.formattag = iup.User(fmt.Sprintf("FGCOLOR=\"%s\"", labelFGColor),
				fmt.Sprintf("BGCOLOR=\"%s\"", commandBGColor),
				fmt.Sprintf("FONTFACE=%s", commandFontFace),
				fmt.Sprintf("FONTSIZE=%s", commandFontSize))
			iup.SetHandle("tag", stw.formattag)
			stw.hist.SetAttribute("ADDFORMATTAG", "tag")
			stw.cline.SetAttribute("FONT", dlg.GetAttribute("VALUE"))
			stw.cline.SetAttribute("FGCOLOR", dlg.GetAttribute("COLOR"))
			stw.cname.SetAttribute("FONT", dlg.GetAttribute("VALUE"))
			stw.cname.SetAttribute("FGCOLOR", dlg.GetAttribute("COLOR"))
		case FONT_CANVAS:
			ff := strings.Split(dlg.GetAttribute("VALUE"), ",")
			fs, _ := strconv.ParseInt(strings.Trim(ff[1], " "), 10, 64)
			canvasFontFace = ff[0]
			canvasFontSize = int(fs)
			canvasFontColor = st.ColorInt(dlg.GetAttribute("COLOR"))
			stw.cdcanv.Font(ff[0], cd.CD_PLAIN, int(fs))
			stw.dbuff.Font(ff[0], cd.CD_PLAIN, int(fs))
			stw.Redraw()
		}
	}
}

// }}}

// Command// {{{
func (stw *Window) feedCommand() {
	command := stw.cline.GetAttribute("VALUE")
	if command != "" {
		stw.addCommandHistory(stw.cline.GetAttribute("VALUE"))
		comhistpos = -1
		stw.cline.SetAttribute("VALUE", "")
		stw.ExecCommand(command)
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
			stw.cline.SetAttribute("VALUE", stw.comhist[comhistpos])
			stw.cline.SetAttribute("CARETPOS", fmt.Sprintf("%d", len(stw.comhist[comhistpos])))
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
			stw.cline.SetAttribute("VALUE", stw.comhist[comhistpos])
			stw.cline.SetAttribute("CARETPOS", fmt.Sprintf("%d", len(stw.comhist[comhistpos])))
			return
		}
	}
}

func (stw *Window) NextSideBarTab() {
	current := stw.SideBar.GetAttribute("VALUEPOS")
	pos, _ := strconv.ParseInt(current, 10, 64)
	size := stw.SideBar.GetAttribute("COUNT")
	count, _ := strconv.ParseInt(size, 10, 64)
	pos++
	if pos >= count {
		pos -= count
	}
	stw.SideBar.SetAttribute("VALUEPOS", fmt.Sprintf("%d", pos))
}

func (stw *Window) addHistory(str string) {
	if str == "" {
		return
	}
	for _, s := range strings.Split(strings.TrimSuffix(str, "\n"), "\n") {
		stw.hist.SetAttribute("APPEND", s)
	}
	lnum, err := strconv.ParseInt(stw.hist.GetAttribute("LINECOUNT"), 10, 64)
	if err != nil {
		fmt.Println(err)
		return
	}
	setpos := fmt.Sprintf("%d:1", int(lnum)+1)
	stw.hist.SetAttribute("SCROLLTO", setpos)
}

func (stw *Window) SetCoord(x, y, z float64) {
	stw.coord.SetAttribute("VALUE", fmt.Sprintf("X: %8.3f Y: %8.3f Z: %8.3f", x, y, z))
}

func (stw *Window) execCommand(com *Command) {
	stw.addHistory(com.Name)
	stw.cname.SetAttribute("VALUE", com.Name)
	stw.lastcommand = com
	com.Exec(stw)
}

func (stw *Window) ExecCommand(al string) {
	if stw.frame == nil {
		if strings.HasPrefix(al, ":") {
			err := st.ExMode(stw, al)
			if err != nil {
				st.ErrorMessage(stw, err, st.ERROR)
			}
		} else {
			stw.Open()
		}
		stw.FocusCanv()
		return
	}
	redraw := true
	alu := strings.ToUpper(al)
	if alu == "." {
		if stw.lastcommand != nil {
			stw.execCommand(stw.lastcommand)
		}
	} else if value, ok := aliases[alu]; ok {
		stw.execCommand(value)
	} else if value, ok := Commands[alu]; ok {
		stw.execCommand(value)
	} else {
		switch {
		default:
			stw.addHistory(fmt.Sprintf("command doesn't exist: %s", al))
		case strings.HasPrefix(al, ":"):
			err := st.ExMode(stw, al)
			if err != nil {
				if _, ok := err.(st.NotRedraw); ok {
					redraw = false
				} else {
					st.ErrorMessage(stw, err, st.ERROR)
				}
			}
		case strings.HasPrefix(al, "'"):
			err := st.Fig2Mode(stw, al)
			if err != nil {
				st.ErrorMessage(stw, err, st.ERROR)
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
			stw.addHistory(fmt.Sprintf("AxisRange: %.3f <= %s <= %.3f", min, tmp, max))
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
			stw.addHistory(fmt.Sprintf("AxisRange: %.3f <= %s", min, tmp))
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
			stw.addHistory(fmt.Sprintf("AxisRange: %.3f <= %s", min, tmp))
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
			stw.addHistory(fmt.Sprintf("AxisRange: %s <= %.3f", tmp, max))
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
			stw.addHistory(fmt.Sprintf("AxisRange: %s <= %.3f", tmp, max))
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
			stw.addHistory(fmt.Sprintf("AxisRange: %s = %.3f", tmp, val))
		}
	}
	if redraw {
		stw.Redraw()
	}
	if stw.cline.GetAttribute("VALUE") == "" {
		stw.FocusCanv()
	}
	return
}

func (stw *Window) CompleteFileName(str string) string {
	path := ""
	if stw.frame != nil {
		path = stw.frame.Path
	}
	stw.completes = st.CompleteFileName(str, path, stw.Recent())
	stw.completepos = 0
	if len(stw.completes) == 0 {
		return str
	}
	return stw.completes[0]
}

func (stw *Window) CompleteExcommand(str string) string {
	i := 0
	rtn := make([]string, len(st.ExAbbrev))
	for ab := range st.ExAbbrev {
		pat := abbrev.MustCompile(ab)
		l := fmt.Sprintf(":%s", pat.Longest())
		if strings.HasPrefix(l, str) {
			rtn[i] = l
			i++
		}
	}
	stw.completepos = 0
	stw.completes = rtn[:i]
	sort.Strings(stw.completes)
	if i > 0 {
		return stw.completes[0]
	} else {
		return str
	}
}

func (stw *Window) CompleteFig2Keyword(str string) string {
	i := 0
	rtn := make([]string, len(st.Fig2Abbrev))
	for ab := range st.Fig2Abbrev {
		pat := abbrev.MustCompile(ab)
		l := fmt.Sprintf("'%s", pat.Longest())
		if strings.HasPrefix(l, str) {
			rtn[i] = l
			i++
		}
	}
	stw.completepos = 0
	stw.completes = rtn[:i]
	if i > 0 {
		return stw.completes[0]
	} else {
		return str
	}
}

func (stw *Window) PrevComplete(str string) string {
	if stw.completes == nil || len(stw.completes) == 0 {
		return str
	}
	stw.completepos--
	if stw.completepos < 0 {
		stw.completepos = len(stw.completes) - 1
	}
	return stw.completes[stw.completepos]
}

func (stw *Window) NextComplete(str string) string {
	if stw.completes == nil || len(stw.completes) == 0 {
		return str
	}
	stw.completepos++
	if stw.completepos >= len(stw.completes) {
		stw.completepos = 0
	}
	return stw.completes[stw.completepos]
}

func (stw *Window) NextFloor() {
	for _, n := range stw.frame.Nodes {
		n.Show()
	}
	for _, el := range stw.frame.Elems {
		el.Show()
	}
	for i, z := range []string{"ZMIN", "ZMAX"} {
		tmpval := stw.frame.Show.Zrange[i]
		ind := 0
		for _, ht := range stw.frame.Ai.Boundary {
			if ht > tmpval {
				break
			}
			ind++
		}
		var val float64
		l := len(stw.frame.Ai.Boundary)
		if ind >= l-1 {
			val = stw.frame.Ai.Boundary[l-2+i]
		} else {
			val = stw.frame.Ai.Boundary[ind]
		}
		stw.frame.Show.Zrange[i] = val
		stw.Labels[z].SetAttribute("VALUE", fmt.Sprintf("%.3f", val))
	}
	stw.Redraw()
}

func (stw *Window) PrevFloor() {
	for _, n := range stw.frame.Nodes {
		n.Show()
	}
	for _, el := range stw.frame.Elems {
		el.Show()
	}
	for i, z := range []string{"ZMIN", "ZMAX"} {
		tmpval := stw.frame.Show.Zrange[i]
		ind := 0
		for _, ht := range stw.frame.Ai.Boundary {
			if ht > tmpval {
				break
			}
			ind++
		}
		var val float64
		if ind <= 2 {
			val = stw.frame.Ai.Boundary[i]
		} else {
			val = stw.frame.Ai.Boundary[ind-2]
		}
		stw.frame.Show.Zrange[i] = val
		stw.Labels[z].SetAttribute("VALUE", fmt.Sprintf("%.3f", val))
	}
	stw.Redraw()
}

func (stw *Window) AxisRange(axis int, min, max float64, any bool) {
	tmpnodes := make([]*st.Node, 0)
	for _, n := range stw.frame.Nodes {
		if !(min <= n.Coord[axis] && n.Coord[axis] <= max) {
			tmpnodes = append(tmpnodes, n)
			n.Hide()
		} else {
			n.Show()
		}
	}
	var tmpelems []*st.Elem
	if !any {
		tmpelems = stw.frame.NodeToElemAny(tmpnodes...)
	} else {
		tmpelems = stw.frame.NodeToElemAll(tmpnodes...)
	}
	for _, el := range stw.frame.Elems {
		el.Show()
	}
	for _, el := range tmpelems {
		el.Hide()
	}
	switch axis {
	case 0:
		stw.frame.Show.Xrange[0] = min
		stw.frame.Show.Xrange[1] = max
		stw.Labels["XMIN"].SetAttribute("VALUE", fmt.Sprintf("%.3f", min))
		stw.Labels["XMAX"].SetAttribute("VALUE", fmt.Sprintf("%.3f", max))
	case 1:
		stw.frame.Show.Yrange[0] = min
		stw.frame.Show.Yrange[1] = max
		stw.Labels["YMIN"].SetAttribute("VALUE", fmt.Sprintf("%.3f", min))
		stw.Labels["YMAX"].SetAttribute("VALUE", fmt.Sprintf("%.3f", max))
	case 2:
		stw.frame.Show.Zrange[0] = min
		stw.frame.Show.Zrange[1] = max
		stw.Labels["ZMIN"].SetAttribute("VALUE", fmt.Sprintf("%.3f", min))
		stw.Labels["ZMAX"].SetAttribute("VALUE", fmt.Sprintf("%.3f", max))
	}
	stw.Redraw()
}

// }}}

// Draw// {{{
func (stw *Window) DrawFrame(canv *cd.Canvas, color uint, flush bool) {
	stw.currentCanvas = canv
	canv.Hatch(cd.CD_FDIAGONAL)
	canv.Clear()
	st.DrawFrame(stw, stw.frame, color, flush)
	stw.SetViewData()
}

func (stw *Window) DrawRange(canv *cd.Canvas, view *st.View) {
	if stw.frame == nil {
		return
	}
	canv.Foreground(cd.CD_DARK_GREEN)
	view.Set(0)
	mins := make([]float64, 3)
	maxs := make([]float64, 3)
	coord := make([][]float64, 8)
	pcoord := make([][]float64, 8)
	for i := 0; i < 8; i++ {
		coord[i] = make([]float64, 3)
		pcoord[i] = make([]float64, 2)
	}
	mins[0], maxs[0], mins[1], maxs[1], mins[2], maxs[2] = stw.frame.Bbox(false)
	for i := 0; i < 3; i++ {
		view.Focus[i] = 0.5 * (mins[i] + maxs[i])
	}
	coord[0][0] = mins[0]
	coord[0][1] = mins[1]
	coord[0][2] = mins[2]
	coord[1][0] = maxs[0]
	coord[1][1] = mins[1]
	coord[1][2] = mins[2]
	coord[2][0] = maxs[0]
	coord[2][1] = maxs[1]
	coord[2][2] = mins[2]
	coord[3][0] = mins[0]
	coord[3][1] = maxs[1]
	coord[3][2] = mins[2]
	coord[4][0] = mins[0]
	coord[4][1] = mins[1]
	coord[4][2] = maxs[2]
	coord[5][0] = maxs[0]
	coord[5][1] = mins[1]
	coord[5][2] = maxs[2]
	coord[6][0] = maxs[0]
	coord[6][1] = maxs[1]
	coord[6][2] = maxs[2]
	coord[7][0] = mins[0]
	coord[7][1] = maxs[1]
	coord[7][2] = maxs[2]
	for i := 0; i < 8; i++ {
		pcoord[i] = view.ProjectCoord(coord[i])
	}
	canv.LineStyle(cd.CD_DOTTED)
	canv.FLine(pcoord[0][0], pcoord[0][1], pcoord[1][0], pcoord[1][1])
	canv.FLine(pcoord[1][0], pcoord[1][1], pcoord[2][0], pcoord[2][1])
	canv.FLine(pcoord[2][0], pcoord[2][1], pcoord[3][0], pcoord[3][1])
	canv.FLine(pcoord[3][0], pcoord[3][1], pcoord[0][0], pcoord[0][1])
	canv.FLine(pcoord[4][0], pcoord[4][1], pcoord[5][0], pcoord[5][1])
	canv.FLine(pcoord[5][0], pcoord[5][1], pcoord[6][0], pcoord[6][1])
	canv.FLine(pcoord[6][0], pcoord[6][1], pcoord[7][0], pcoord[7][1])
	canv.FLine(pcoord[7][0], pcoord[7][1], pcoord[4][0], pcoord[4][1])
	canv.FLine(pcoord[0][0], pcoord[0][1], pcoord[4][0], pcoord[4][1])
	canv.FLine(pcoord[1][0], pcoord[1][1], pcoord[5][0], pcoord[5][1])
	canv.FLine(pcoord[2][0], pcoord[2][1], pcoord[6][0], pcoord[6][1])
	canv.FLine(pcoord[3][0], pcoord[3][1], pcoord[7][0], pcoord[7][1])
	mins[0], maxs[0], mins[1], maxs[1], mins[2], maxs[2] = stw.frame.Bbox(true)
	coord[0][0] = mins[0]
	coord[0][1] = mins[1]
	coord[0][2] = mins[2]
	coord[1][0] = maxs[0]
	coord[1][1] = mins[1]
	coord[1][2] = mins[2]
	coord[2][0] = maxs[0]
	coord[2][1] = maxs[1]
	coord[2][2] = mins[2]
	coord[3][0] = mins[0]
	coord[3][1] = maxs[1]
	coord[3][2] = mins[2]
	coord[4][0] = mins[0]
	coord[4][1] = mins[1]
	coord[4][2] = maxs[2]
	coord[5][0] = maxs[0]
	coord[5][1] = mins[1]
	coord[5][2] = maxs[2]
	coord[6][0] = maxs[0]
	coord[6][1] = maxs[1]
	coord[6][2] = maxs[2]
	coord[7][0] = mins[0]
	coord[7][1] = maxs[1]
	coord[7][2] = maxs[2]
	for i := 0; i < 8; i++ {
		pcoord[i] = view.ProjectCoord(coord[i])
	}
	canv.LineStyle(cd.CD_CONTINUOUS)
	canv.FLine(pcoord[0][0], pcoord[0][1], pcoord[1][0], pcoord[1][1])
	canv.FLine(pcoord[1][0], pcoord[1][1], pcoord[2][0], pcoord[2][1])
	canv.FLine(pcoord[2][0], pcoord[2][1], pcoord[3][0], pcoord[3][1])
	canv.FLine(pcoord[3][0], pcoord[3][1], pcoord[0][0], pcoord[0][1])
	canv.FLine(pcoord[4][0], pcoord[4][1], pcoord[5][0], pcoord[5][1])
	canv.FLine(pcoord[5][0], pcoord[5][1], pcoord[6][0], pcoord[6][1])
	canv.FLine(pcoord[6][0], pcoord[6][1], pcoord[7][0], pcoord[7][1])
	canv.FLine(pcoord[7][0], pcoord[7][1], pcoord[4][0], pcoord[4][1])
	canv.FLine(pcoord[0][0], pcoord[0][1], pcoord[4][0], pcoord[4][1])
	canv.FLine(pcoord[1][0], pcoord[1][1], pcoord[5][0], pcoord[5][1])
	canv.FLine(pcoord[2][0], pcoord[2][1], pcoord[6][0], pcoord[6][1])
	canv.FLine(pcoord[3][0], pcoord[3][1], pcoord[7][0], pcoord[7][1])
}

func (stw *Window) DrawTexts(canv *cd.Canvas, black bool) {
	for _, t := range stw.textBox {
		if !t.IsHidden(stw.frame.Show) {
			if black {
				col := t.Font.Color
				t.Font.Color = cd.CD_BLACK
				DrawText(t, canv)
				t.Font.Color = col
			} else {
				DrawText(t, canv)
			}
		}
	}
	if !STLOGO.IsHidden(stw.frame.Show) {
		DrawText(STLOGO, canv)
	}
}

func (stw *Window) DrawPivot(nodes []*st.Node, pivot, end chan int) {
	stw.DrawFrameNode()
	stw.DrawTexts(stw.cdcanv, false)
	stw.cdcanv.Foreground(pivotColor)
	ind := 0
	nnum := 0
	for {
		select {
		case <-end:
			return
		case <-pivot:
			ind++
			if ind >= 6 {
				stw.currentCanvas = stw.cdcanv
				st.DrawNodeNum(stw, nodes[nnum])
				nnum++
				ind = 0
			}
		}
	}
}

func (stw *Window) Redraw() {
	stw.DrawFrame(stw.dbuff, stw.frame.Show.ColorMode, false)
	if stw.Property {
		stw.UpdatePropertyDialog()
	}
	stw.DrawTexts(stw.dbuff, false)
	stw.dbuff.Flush()
}

func (stw *Window) DrawFrameNode() {
	stw.dbuff.Clear()
	stw.currentCanvas = stw.dbuff
	st.DrawFrameNode(stw, stw.frame, stw.frame.Show.ColorMode, true)
}

// TODO: implement
func (stw *Window) DrawConvexHull() {
	if stw.frame == nil {
		return
	}
	var nnum int
	nodes := make([]*st.Node, len(stw.frame.Nodes))
	for _, n := range stw.frame.Nodes {
		nodes[nnum] = n
		nnum++
	}
	sort.Sort(st.NodeByPcoordY{nodes})
	hstart := nodes[0]
	hend := nodes[nnum-1]
	fmt.Printf("START: %d, END: %d\n", hstart.Num, hend.Num)
	cwnodes := make([]*st.Node, 1)
	cwnodes[0] = hstart
	cworigin := []float64{1000.0, cwnodes[0].Pcoord[1]}
	ccwnodes := make([]*st.Node, 1)
	ccwnodes[0] = hstart
	ccworigin := []float64{-1000.0, ccwnodes[0].Pcoord[1]}
	stopcw := false
	stopccw := false
	ncw := 0
	nccw := 0
	dcw := 0.0
	dccw := 0.0
	for {
		cwminangle := 10000.0
		ccwminangle := 10000.0
		var tmpcw *st.Node
		var tmpccw *st.Node
	chloop:
		for _, n := range nodes {
			for _, cw := range cwnodes {
				if n == cw {
					continue chloop
				}
			}
			for _, ccw := range ccwnodes {
				if n == ccw {
					continue chloop
				}
			}
			var cwangle, ccwangle float64
			var cw bool
			if !stopcw {
				if st.Distance(hstart, n) > dcw {
					if ncw == 0 {
						cwangle, cw = st.ClockWise2(cworigin, cwnodes[ncw].Pcoord, n.Pcoord)
					} else {
						cwangle, cw = st.ClockWise2(cwnodes[ncw-1].Pcoord, cwnodes[ncw].Pcoord, n.Pcoord)
					}
					cwangle = math.Abs(cwangle)
					if n.Num == 101 || n.Num == 170 {
						fmt.Printf("CWANGLE = %.3f (%d)\n", cwangle, n.Num)
						fmt.Print(cw)
					}
					if cw {
						if cwangle == cwminangle {
							if st.Distance(cwnodes[ncw], n) > st.Distance(cwnodes[ncw], tmpcw) {
								tmpcw = n
								cwminangle = cwangle
							}
						} else if cwangle < cwminangle {
							tmpcw = n
							cwminangle = cwangle
						}
					}
				}
			}
			if !stopccw {
				if st.Distance(hstart, n) > dccw {
					if nccw == 0 {
						ccwangle, cw = st.ClockWise2(ccworigin, ccwnodes[nccw].Pcoord, n.Pcoord)
					} else {
						ccwangle, cw = st.ClockWise2(ccwnodes[nccw-1].Pcoord, ccwnodes[nccw].Pcoord, n.Pcoord)
					}
					// if n.Num == 101 || n.Num == 170 {
					// 	fmt.Printf("CCWANGLE = %.3f (%d)\n", ccwangle, n.Num)
					// }
					if !cw {
						if ccwangle == ccwminangle {
							if st.Distance(cwnodes[ncw], n) > st.Distance(cwnodes[ncw], tmpcw) {
								tmpccw = n
								ccwminangle = ccwangle
							}
						} else if ccwangle < ccwminangle {
							tmpccw = n
							ccwminangle = ccwangle
						}
					}
				}
			}
		}
		if !stopcw && tmpcw != nil {
			fmt.Printf("CW: %d, ", tmpcw.Num)
			if tmpcw == hend {
				stopcw = true
			} else {
				ncw++
				cwnodes = append(cwnodes, tmpcw)
				dcw = st.Distance(hstart, tmpcw)
			}
		} else {
			fmt.Println("CW: -, ")
		}
		if !stopccw && tmpccw != nil {
			fmt.Printf("CCW: %d, ", tmpccw.Num)
			if tmpccw == hend {
				stopccw = true
			} else {
				nccw++
				ccwnodes = append(ccwnodes, tmpccw)
				dccw = st.Distance(hstart, tmpccw)
			}
		} else {
			fmt.Println("CCW: -")
		}
		fmt.Println(stopcw, stopccw)
		fmt.Println(cwnodes[len(cwnodes)-1].Num, ccwnodes[len(ccwnodes)-1].Num)
		if stopcw && stopccw {
			break
		}
	}
	chnodes := make([]*st.Node, ncw+nccw+2)
	chnodes[0] = hstart
	for i := 1; i < ncw+1; i++ {
		chnodes[i] = cwnodes[i]
	}
	chnodes[ncw+1] = hend
	for i := 1; i < nccw+1; i++ {
		chnodes[i+1+ncw] = ccwnodes[nccw+1-i]
	}
	for i, n := range chnodes {
		fmt.Printf("%d: %d\n", i, n.Num)
	}
	stw.dbuff.Clear()
	stw.dbuff.Begin(cd.CD_FILL)
	// stw.dbuff.Foreground(PlateEdgeColor)
	stw.dbuff.Begin(cd.CD_CLOSED_LINES)
	for i := 0; i < ncw+nccw+2; i++ {
		stw.dbuff.FVertex(chnodes[i].Pcoord[0], chnodes[i].Pcoord[1])
	}
	stw.dbuff.End()
	stw.dbuff.Flush()
}

func (stw *Window) SetSelectData() {
	if stw.frame != nil {
		stw.lselect.SetAttribute("VALUE", fmt.Sprintf("ELEM = %5d\nNODE = %5d", len(stw.selectElem), len(stw.selectNode)))
	} else {
		stw.lselect.SetAttribute("VALUE", "ELEM =     0\nNODE =     0")
	}
}

// DataLabel
func (stw *Window) SetViewData() {
	if stw.frame != nil {
		stw.Labels["GFACT"].SetAttribute("VALUE", fmt.Sprintf("%.1f", stw.frame.View.Gfact))
		stw.Labels["DISTR"].SetAttribute("VALUE", fmt.Sprintf("%.1f", stw.frame.View.Dists[0]))
		stw.Labels["DISTL"].SetAttribute("VALUE", fmt.Sprintf("%.1f", stw.frame.View.Dists[1]))
		stw.Labels["PHI"].SetAttribute("VALUE", fmt.Sprintf("%.1f", stw.frame.View.Angle[0]))
		stw.Labels["THETA"].SetAttribute("VALUE", fmt.Sprintf("%.1f", stw.frame.View.Angle[1]))
		stw.Labels["FOCUSX"].SetAttribute("VALUE", fmt.Sprintf("%.1f", stw.frame.View.Focus[0]))
		stw.Labels["FOCUSY"].SetAttribute("VALUE", fmt.Sprintf("%.1f", stw.frame.View.Focus[1]))
		stw.Labels["FOCUSZ"].SetAttribute("VALUE", fmt.Sprintf("%.1f", stw.frame.View.Focus[2]))
		stw.Labels["CENTERX"].SetAttribute("VALUE", fmt.Sprintf("%.1f", stw.frame.View.Center[0]))
		stw.Labels["CENTERY"].SetAttribute("VALUE", fmt.Sprintf("%.1f", stw.frame.View.Center[1]))
	}
}

func (stw *Window) Bbox() (xmin, xmax, ymin, ymax float64) {
	return stw.frame.Bbox2D(true)
}

func (stw *Window) SetShowRange() {
	xmin, xmax, ymin, ymax, zmin, zmax := stw.frame.Bbox(true)
	stw.frame.Show.Xrange[0] = xmin
	stw.frame.Show.Xrange[1] = xmax
	stw.frame.Show.Yrange[0] = ymin
	stw.frame.Show.Yrange[1] = ymax
	stw.frame.Show.Zrange[0] = zmin
	stw.frame.Show.Zrange[1] = zmax
	stw.Labels["XMAX"].SetAttribute("VALUE", fmt.Sprintf("%.3f", xmax))
	stw.Labels["XMIN"].SetAttribute("VALUE", fmt.Sprintf("%.3f", xmin))
	stw.Labels["YMAX"].SetAttribute("VALUE", fmt.Sprintf("%.3f", ymax))
	stw.Labels["YMIN"].SetAttribute("VALUE", fmt.Sprintf("%.3f", ymin))
	stw.Labels["ZMAX"].SetAttribute("VALUE", fmt.Sprintf("%.3f", zmax))
	stw.Labels["ZMIN"].SetAttribute("VALUE", fmt.Sprintf("%.3f", zmin))
}

func (stw *Window) HideNotSelected() {
	if stw.selectElem != nil {
		for _, n := range stw.frame.Nodes {
			n.Hide()
		}
		for _, el := range stw.frame.Elems {
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
		for _, n := range stw.frame.Nodes {
			n.Lock = true
		}
		for _, el := range stw.frame.Elems {
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

func (stw *Window) DeleteSelected() {
	if stw.selectElem != nil {
		for _, el := range stw.selectElem {
			if el != nil && !el.Lock {
				stw.frame.DeleteElem(el.Num)
			}
		}
	}
	stw.Deselect()
	st.Snapshot(stw)
	stw.Redraw()
}

func (stw *Window) SelectNotHidden() {
	if stw.frame == nil {
		return
	}
	stw.Deselect()
	stw.selectElem = make([]*st.Elem, len(stw.frame.Elems))
	num := 0
	for _, el := range stw.frame.Elems {
		if el.IsHidden(stw.frame.Show) {
			continue
		}
		stw.selectElem[num] = el
		num++
	}
	stw.selectElem = stw.selectElem[:num]
	stw.Redraw()
}

func (stw *Window) CopyClipboard() error {
	var rtn error
	if stw.selectElem == nil {
		return nil
	}
	var otp bytes.Buffer
	ns := stw.frame.ElemToNode(stw.selectElem...)
	getcoord(stw, func(x, y, z float64) {
		for _, n := range ns {
			otp.WriteString(n.CopyString(x, y, z))
		}
		for _, el := range stw.selectElem {
			if el != nil {
				otp.WriteString(el.InpString())
			}
		}
		err := clipboard.WriteAll(otp.String())
		if err != nil {
			rtn = err
		}
		stw.addHistory(fmt.Sprintf("%d ELEMs Copied", len(stw.selectElem)))
		stw.EscapeCB()
	})
	if rtn != nil {
		return rtn
	}
	return nil
}

// TODO: Test
func (stw *Window) PasteClipboard() error {
	text, err := clipboard.ReadAll()
	if err != nil {
		return err
	}
	getcoord(stw, func(x, y, z float64) {
		s := bufio.NewScanner(strings.NewReader(text))
		coord := []float64{x, y, z}
		angle := 0.0
		tmp := make([]string, 0)
		nodemap := make(map[int]int)
		var err error
		for s.Scan() {
			var words []string
			for _, k := range strings.Split(s.Text(), " ") {
				if k != "" {
					words = append(words, k)
				}
			}
			if len(words) == 0 {
				continue
			}
			first := words[0]
			switch first {
			default:
				tmp = append(tmp, words...)
			case "NODE", "ELEM":
				nodemap, err = stw.frame.ParseInp(tmp, coord, angle, nodemap, false)
				tmp = words
			}
			if err != nil {
				break
			}
		}
		nodemap, err = stw.frame.ParseInp(tmp, coord, angle, nodemap, false)
		stw.EscapeCB()
	})
	return nil
}

func (stw *Window) ShapeData(sh st.Shape) {
	var tb *TextBox
	if t, tok := stw.textBox["SHAPE"]; tok {
		tb = t
	} else {
		tb = NewTextBox()
		tb.Show()
		tb.position = []float64{stw.CanvasSize[0] - 300.0, 200.0}
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

func (stw *Window) SectionData(sec *st.Sect) {
	var tb *TextBox
	if t, tok := stw.textBox["SECTION"]; tok {
		tb = t
	} else {
		tb = NewTextBox()
		tb.Show()
		tb.position = []float64{stw.CanvasSize[0] - 500.0, float64(dataareaheight)}
		stw.textBox["SECTION"] = tb
	}
	tb.SetText(strings.Split(sec.InpString(), "\n"))
	if al, ok := stw.frame.Allows[sec.Num]; ok {
		tb.AddText(strings.Split(al.String(), "\n")...)
	}
	tb.ScrollToTop()
}

func (stw *Window) CurrentLap(comment string, nlap, laps int) {
	var tb *TextBox
	if t, tok := stw.textBox["LAP"]; tok {
		tb = t
	} else {
		tb = NewTextBox()
		tb.Show()
		tb.position = []float64{30.0, stw.CanvasSize[1] - 30.0}
		stw.textBox["LAP"] = tb
	}
	if comment == "" {
		tb.SetText([]string{fmt.Sprintf("LAP: %3d / %3d", nlap, laps)})
	} else {
		tb.SetText([]string{comment, fmt.Sprintf("LAP: %3d / %3d", nlap, laps)})
	}
}

func (stw *Window) ShowAll() {
	for _, el := range stw.frame.Elems {
		el.Show()
	}
	for _, n := range stw.frame.Nodes {
		n.Show()
	}
	for _, k := range stw.frame.Kijuns {
		k.Show()
	}
	for i, et := range st.ETYPES {
		if i == st.WBRACE || i == st.SBRACE {
			continue
		}
		if lb, ok := stw.Labels[et]; ok {
			lb.SetAttribute("FGCOLOR", labelFGColor)
		}
		stw.frame.Show.Etype[i] = true
	}
	stw.ShowAllSection()
	stw.frame.Show.All()
	stw.Labels["XMIN"].SetAttribute("VALUE", fmt.Sprintf("%.3f", stw.frame.Show.Xrange[0]))
	stw.Labels["XMAX"].SetAttribute("VALUE", fmt.Sprintf("%.3f", stw.frame.Show.Xrange[1]))
	stw.Labels["YMIN"].SetAttribute("VALUE", fmt.Sprintf("%.3f", stw.frame.Show.Yrange[0]))
	stw.Labels["YMAX"].SetAttribute("VALUE", fmt.Sprintf("%.3f", stw.frame.Show.Yrange[1]))
	stw.Labels["ZMIN"].SetAttribute("VALUE", fmt.Sprintf("%.3f", stw.frame.Show.Zrange[0]))
	stw.Labels["ZMAX"].SetAttribute("VALUE", fmt.Sprintf("%.3f", stw.frame.Show.Zrange[1]))
	stw.Redraw()
}

func (stw *Window) UnlockAll() {
	for _, el := range stw.frame.Elems {
		el.Lock = false
	}
	for _, n := range stw.frame.Nodes {
		n.Lock = false
	}
	stw.Redraw()
}

func (stw *Window) Animate(view *st.View) {
	scale := 1.0
	if stw.frame.View.Perspective {
		scale = math.Pow(view.Dists[1]/stw.frame.View.Dists[1], stw.CanvasAnimateSpeed())
	} else {
		scale = math.Pow(view.Gfact/stw.frame.View.Gfact, stw.CanvasAnimateSpeed())
	}
	center := make([]float64, 2)
	angle := make([]float64, 2)
	focus := make([]float64, 3)
	for i := 0; i < 3; i++ {
		focus[i] = stw.CanvasAnimateSpeed() * (view.Focus[i] - stw.frame.View.Focus[i])
		if i >= 2 {
			break
		}
		center[i] = stw.CanvasAnimateSpeed() * (view.Center[i] - stw.frame.View.Center[i])
		angle[i] = view.Angle[i] - stw.frame.View.Angle[i]
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
		angle[i] *= stw.CanvasAnimateSpeed()
	}
	for i := 0; i < int(1/stw.CanvasAnimateSpeed()); i++ {
		if stw.frame.View.Perspective {
			stw.frame.View.Dists[1] *= scale
		} else {
			stw.frame.View.Gfact *= scale
		}
		for j := 0; j < 3; j++ {
			stw.frame.View.Focus[j] += focus[j]
			if j >= 2 {
				break
			}
			stw.frame.View.Center[j] += center[j]
			stw.frame.View.Angle[j] += angle[j]
		}
		stw.DrawFrameNode()
	}
}

func (stw *Window) ShowAtPaperCenter(canv *cd.Canvas) {
	stw.frame.SetFocus(nil)
	for _, n := range stw.frame.Nodes {
		stw.frame.View.ProjectNode(n)
	}
	xmin, xmax, ymin, ymax := stw.Bbox()
	if xmax == xmin && ymax == ymin {
		return
	}
	stw.currentCanvas = canv
	w, h, err := stw.CanvasPaperSize()
	if err != nil {
		st.ErrorMessage(stw, err, st.ERROR)
		return
	}
	scale := math.Min(w/(xmax-xmin), h/(ymax-ymin)) * stw.CanvasFitScale()
	if stw.frame.View.Perspective {
		stw.frame.View.Dists[1] *= scale
	} else {
		stw.frame.View.Gfact *= scale
	}
	cw, ch := canv.GetSize()
	stw.frame.View.Center[0] = float64(cw)*0.5 + scale*(stw.frame.View.Center[0]-0.5*(xmax+xmin))
	stw.frame.View.Center[1] = float64(ch)*0.5 + scale*(stw.frame.View.Center[1]-0.5*(ymax+ymin))
}

func (stw *Window) CanvasCenterView(canv *cd.Canvas, angle []float64) *st.View {
	a0 := make([]float64, 2)
	f0 := make([]float64, 3)
	focus := make([]float64, 3)
	for i := 0; i < 3; i++ {
		f0[i] = stw.frame.View.Focus[i]
		if i >= 2 {
			break
		}
		a0[i] = stw.frame.View.Angle[i]
		stw.frame.View.Angle[i] = angle[i]
	}
	stw.frame.SetFocus(nil)
	stw.frame.View.Set(0)
	for _, n := range stw.frame.Nodes {
		stw.frame.View.ProjectNode(n)
	}
	xmin, xmax, ymin, ymax := stw.Bbox()
	for i := 0; i < 3; i++ {
		focus[i] = stw.frame.View.Focus[i]
		stw.frame.View.Focus[i] = f0[i]
		if i >= 2 {
			break
		}
		stw.frame.View.Angle[i] = a0[i]
	}
	stw.frame.View.Set(0)
	if xmax == xmin && ymax == ymin {
		return nil
	}
	w, h := canv.GetSize()
	view := stw.frame.View.Copy()
	view.Focus = focus
	scale := math.Min(float64(w)/(xmax-xmin), float64(h)/(ymax-ymin)) * stw.CanvasFitScale()
	if stw.frame.View.Perspective {
		view.Dists[1] = stw.frame.View.Dists[1] * scale
	} else {
		view.Gfact = stw.frame.View.Gfact * scale
	}
	view.Center[0] = float64(w)*0.5 + scale*(stw.frame.View.Center[0]-0.5*(xmax+xmin))
	view.Center[1] = float64(h)*0.5 + scale*(stw.frame.View.Center[1]-0.5*(ymax+ymin))
	view.Angle = angle
	return view
}

func (stw *Window) ShowAtCanvasCenter(canv *cd.Canvas) {
	view := stw.CanvasCenterView(canv, stw.frame.View.Angle)
	stw.Animate(view)
}

func (stw *Window) ShowCenter() {
	if showprintrange {
		stw.ShowAtPaperCenter(stw.cdcanv)
	} else {
		stw.ShowAtCanvasCenter(stw.cdcanv)
	}
	stw.Redraw()
}

func (stw *Window) SetAngle(phi, theta float64) {
	if stw.frame != nil {
		view := stw.CanvasCenterView(stw.cdcanv, []float64{phi, theta})
		stw.Animate(view)
	}
}

func (stw *Window) SetFocus() {
	var focus []float64
	if stw.selectElem != nil {
		for _, el := range stw.selectElem {
			if el == nil {
				continue
			}
			focus = el.MidPoint()
		}
	}
	if focus == nil {
		if stw.selectNode != nil {
			for _, n := range stw.selectNode {
				if n == nil {
					continue
				}
				focus = n.Coord
			}
		}
	}
	v := stw.frame.View.Copy()
	stw.frame.SetFocus(focus)
	w, h := stw.cdcanv.GetSize()
	stw.CanvasSize = []float64{float64(w), float64(h)}
	stw.frame.View.Center[0] = stw.CanvasSize[0] * 0.5
	stw.frame.View.Center[1] = stw.CanvasSize[1] * 0.5
	view := stw.frame.View.Copy()
	stw.frame.View = v
	stw.Animate(view)
	stw.Redraw()
	return
}

// }}}

// Select// {{{
func (stw *Window) SelectNodeStart(arg *iup.MouseButton) {
	stw.dbuff.UpdateYAxis(&arg.Y)
	if arg.Pressed == 0 { // Released
		left := min(stw.startX, stw.endX)
		right := max(stw.startX, stw.endX)
		bottom := min(stw.startY, stw.endY)
		top := max(stw.startY, stw.endY)
		if (right-left < nodeSelectPixel) && (top-bottom < nodeSelectPixel) {
			n := stw.frame.PickNode(float64(left), float64(bottom), float64(nodeSelectPixel))
			if n != nil {
				stw.MergeSelectNode([]*st.Node{n}, isShift(arg.Status))
			} else {
				stw.selectNode = make([]*st.Node, 0)
			}
		} else {
			tmpselect := make([]*st.Node, len(stw.frame.Nodes))
			i := 0
			for _, v := range stw.frame.Nodes {
				if v.IsHidden(stw.frame.Show) {
					continue
				}
				if float64(left) <= v.Pcoord[0] && v.Pcoord[0] <= float64(right) && float64(bottom) <= v.Pcoord[1] && v.Pcoord[1] <= float64(top) {
					tmpselect[i] = v
					i++
				}
			}
			stw.MergeSelectNode(tmpselect[:i], isShift(arg.Status))
		}
		stw.cdcanv.Rect(left, right, bottom, top)
		stw.cdcanv.WriteMode(cd.CD_REPLACE)
		stw.Redraw()
	} else { // Pressed
		stw.cdcanv.Foreground(cd.CD_RED)
		stw.cdcanv.WriteMode(cd.CD_XOR)
		stw.startX = int(arg.X)
		stw.startY = int(arg.Y)
		stw.endX = int(arg.X)
		stw.endY = int(arg.Y)
		first = 1
	}
}

func (stw *Window) SelectElemStart(arg *iup.MouseButton) {
	stw.dbuff.UpdateYAxis(&arg.Y)
	if arg.Pressed == 0 { // Released
		left := min(stw.startX, stw.endX)
		right := max(stw.startX, stw.endX)
		bottom := min(stw.startY, stw.endY)
		top := max(stw.startY, stw.endY)
		if (right-left < dotSelectPixel) && (top-bottom < dotSelectPixel) {
			el := stw.frame.PickLineElem(float64(left), float64(bottom), dotSelectPixel)
			if el == nil {
				els := stw.frame.PickPlateElem(float64(left), float64(bottom))
				if len(els) > 0 {
					el = els[0]
				}
			}
			if el != nil {
				st.MergeSelectElem(stw, []*st.Elem{el}, isShift(arg.Status))
			} else {
				stw.selectElem = make([]*st.Elem, 0)
			}
		} else {
			tmpselectnode := make([]*st.Node, len(stw.frame.Nodes))
			i := 0
			for _, v := range stw.frame.Nodes {
				if float64(left) <= v.Pcoord[0] && v.Pcoord[0] <= float64(right) && float64(bottom) <= v.Pcoord[1] && v.Pcoord[1] <= float64(top) {
					tmpselectnode[i] = v
					i++
				}
			}
			tmpselectelem := make([]*st.Elem, len(stw.frame.Elems))
			k := 0
			switch selectDirection {
			case SD_FROMLEFT:
				for _, el := range stw.frame.Elems {
					if el.IsHidden(stw.frame.Show) {
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
				for _, el := range stw.frame.Elems {
					if el.IsHidden(stw.frame.Show) {
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
			st.MergeSelectElem(stw, tmpselectelem[:k], isShift(arg.Status))
		}
		stw.cdcanv.Rect(left, right, bottom, top)
		stw.cdcanv.WriteMode(cd.CD_REPLACE)
		stw.Redraw()
	} else { // Pressed
		stw.cdcanv.Foreground(cd.CD_WHITE)
		stw.cdcanv.WriteMode(cd.CD_XOR)
		stw.startX = int(arg.X)
		stw.startY = int(arg.Y)
		stw.endX = int(arg.X)
		stw.endY = int(arg.Y)
		first = 1
	}
}

func abs(val int) int {
	if val >= 0 {
		return val
	} else {
		return -val
	}
}
func (stw *Window) SelectElemFenceStart(arg *iup.MouseButton) {
	stw.dbuff.UpdateYAxis(&arg.Y)
	if arg.Pressed == 0 { // Released
		els := stw.frame.FenceLine(float64(stw.startX), float64(stw.startY), float64(stw.endX), float64(stw.endY))
		st.MergeSelectElem(stw, els, isShift(arg.Status))
		stw.cdcanv.Line(stw.startX, stw.startY, stw.endX, stw.endY)
		stw.cdcanv.WriteMode(cd.CD_REPLACE)
		stw.Redraw()
		stw.EscapeCB()
	} else { // Pressed
		stw.cdcanv.Foreground(cd.CD_WHITE)
		stw.cdcanv.WriteMode(cd.CD_XOR)
		stw.startX = int(arg.X)
		stw.startY = int(arg.Y)
		stw.endX = int(arg.X)
		stw.endY = int(arg.Y)
		first = 1
	}
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

func (stw *Window) SelectNodeMotion(arg *iup.MouseMotion) {
	if stw.startX <= int(arg.X) {
		selectDirection = SD_FROMLEFT
		if first == 1 {
			first = 0
		} else {
			stw.cdcanv.Rect(min(stw.endX, stw.startX), max(stw.endX, stw.startX), min(stw.startY, stw.endY), max(stw.startY, stw.endY))
		}
		stw.cdcanv.Rect(int(arg.X), stw.startX, min(stw.startY, int(arg.Y)), max(stw.startY, int(arg.Y)))
		stw.endX = int(arg.X)
		stw.endY = int(arg.Y)
	} else {
		stw.cdcanv.LineStyle(cd.CD_DASHED)
		selectDirection = SD_FROMRIGHT
		if first == 1 {
			first = 0
		} else {
			stw.cdcanv.Rect(min(stw.endX, stw.startX), max(stw.endX, stw.startX), min(stw.startY, stw.endY), max(stw.startY, stw.endY))
		}
		stw.cdcanv.Rect(stw.startX, int(arg.X), min(stw.startY, int(arg.Y)), max(stw.startY, int(arg.Y)))
		stw.endX = int(arg.X)
		stw.endY = int(arg.Y)
		stw.cdcanv.LineStyle(cd.CD_CONTINUOUS)
	}
}

func (stw *Window) SelectElemMotion(arg *iup.MouseMotion) {
	selectDirection = SD_FROMLEFT
	if stw.startX <= int(arg.X) {
		if first == 1 {
			first = 0
		} else {
			stw.cdcanv.Rect(min(stw.endX, stw.startX), max(stw.endX, stw.startX), min(stw.startY, stw.endY), max(stw.startY, stw.endY))
		}
		stw.cdcanv.Rect(int(arg.X), stw.startX, min(stw.startY, int(arg.Y)), max(stw.startY, int(arg.Y)))
		stw.endX = int(arg.X)
		stw.endY = int(arg.Y)
	} else {
		selectDirection = SD_FROMRIGHT
		stw.cdcanv.LineStyle(cd.CD_DASHED)
		if first == 1 {
			first = 0
		} else {
			stw.cdcanv.Rect(min(stw.endX, stw.startX), max(stw.endX, stw.startX), min(stw.startY, stw.endY), max(stw.startY, stw.endY))
		}
		stw.cdcanv.Rect(stw.startX, int(arg.X), min(stw.startY, int(arg.Y)), max(stw.startY, int(arg.Y)))
		stw.endX = int(arg.X)
		stw.endY = int(arg.Y)
		stw.cdcanv.LineStyle(cd.CD_CONTINUOUS)
	}
}

func (stw *Window) SelectElemFenceMotion(arg *iup.MouseMotion) {
	if first == 1 {
		first = 0
	} else {
		stw.cdcanv.Line(stw.startX, stw.startY, stw.endX, stw.endY)
	}
	if isCtrl(arg.Status) {
		stw.cdcanv.Line(stw.startX, stw.startY, int(arg.X), stw.startY)
		stw.endX = int(arg.X)
		stw.endY = stw.startY
	} else {
		stw.cdcanv.Line(stw.startX, stw.startY, int(arg.X), int(arg.Y))
		stw.endX = int(arg.X)
		stw.endY = int(arg.Y)
	}
}

// type MouseButtonMotion interface {
//     GetX() int
//     GetY() int
// }
// type MyMouseMotion struct {
//     arg *iup.MouseMotion
// }
// func NewMyMouseMotion(arg *iup.MouseMotion) *MyMouseMotion {
//     mm := new(MyMouseMotion)
//     mm.arg = arg
//     return mm
// }
// func (mm *MyMouseMotion) GetX() int {
//     return mm.arg.X
// }
// func (mm *MyMouseMotion) GetY() int {
//     return mm.arg.Y
// }
// type MyMouseButton struct {
//     arg *iup.MouseButton
// }
// func NewMyMouseButton(arg *iup.MouseButton) *MyMouseButton {
//     mb := new(MyMouseButton)
//     mb.arg = arg
//     return mb
// }
// func (mb *MyMouseButton) GetX() int {
//     return mb.arg.X
// }
// func (mb *MyMouseButton) GetY() int {
//     return mb.arg.Y
// }

// func (stw *Window) TailLine(x, y int, arg MouseButtonMotion) {
//     // if first == 1 {
//     //     first = 0
//     // } else {
//     stw.cdcanv.Line(x, y, stw.endX, stw.endY)
//     // }
//     stw.cdcanv.Line(x, y, arg.GetX(), arg.GetY())
//     stw.endX = arg.GetX(); stw.endY = arg.GetY()
//     fmt.Printf("TAIL: %d %d %d %d %d %d\n", int(stw.selectNode[0].Pcoord[0]), int(stw.selectNode[0].Pcoord[1]), arg.GetX(), arg.GetY(), stw.endX, stw.endY)
// }
func (stw *Window) TailLine(x, y int, arg *iup.MouseMotion) {
	if first == 1 {
		first = 0
	} else {
		stw.cdcanv.Line(x, y, stw.endX, stw.endY)
	}
	stw.cdcanv.Line(x, y, int(arg.X), int(arg.Y))
	stw.endX = int(arg.X)
	stw.endY = int(arg.Y)
}

func (stw *Window) TailPolygon(ns []*st.Node, arg *iup.MouseMotion) {
	stw.dbuff.InteriorStyle(cd.CD_HATCH)
	stw.dbuff.Hatch(cd.CD_FDIAGONAL)
	if first == 1 {
		first = 0
	} else {
		l := len(ns) + 1
		num := 0
		coords := make([][]float64, l)
		for i := 0; i < l-1; i++ {
			if ns[i] == nil {
				continue
			}
			coords[num] = ns[i].Pcoord
			num++
		}
		coords[num] = []float64{float64(stw.endX), float64(stw.endY)}
		stw.cdcanv.Polygon(cd.CD_FILL, coords[:num+1]...)
		// stw.cdcanv.Polygon(cd.CD_FILL, coords[0], coords[num-1], coords[num])
	}
	l := len(ns) + 1
	num := 0
	coords := make([][]float64, l)
	for i := 0; i < l-1; i++ {
		if ns[i] == nil {
			continue
		}
		coords[num] = ns[i].Pcoord
		num++
	}
	coords[num] = []float64{float64(arg.X), float64(arg.Y)}
	stw.cdcanv.Polygon(cd.CD_FILL, coords[:num+1]...)
	stw.endX = int(arg.X)
	stw.endY = int(arg.Y)
}

func (stw *Window) Deselect() {
	stw.selectNode = make([]*st.Node, 0)
	stw.selectElem = make([]*st.Elem, 0)
}

// }}}

// Query// {{{
func (stw *Window) Yn(title, question string) bool {
	ans := iup.Alarm(title, question, "はい", "いいえ", "キャンセル")
	switch ans {
	default:
		return false
	case 1:
		return true
	case 2, 3:
		return false
	}
}

func (stw *Window) Yna(title, question, another string) int {
	ans := iup.Alarm(title, question, "はい", "いいえ", another)
	return ans
}

func (stw *Window) Query(question string) (rtn string, err error) {
	var ans string
	var er error
	func(title string, ans *string, er *error) {
		var dlg *iup.Handle
		var text *iup.Handle
		returnvalues := func() {
			*ans = text.GetAttribute("VALUE")
		}
		label := iup.Text(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
			fmt.Sprintf("FGCOLOR=\"%s\"", labelFGColor),
			fmt.Sprintf("BGCOLOR=\"%s\"", labelBGColor),
			fmt.Sprintf("VALUE=\"%s\"", question),
			"CANFOCUS=NO",
			"READONLY=YES",
			"BORDER=NO",
			fmt.Sprintf("SIZE=%dx%d", datalabelwidth+datatextwidth, dataheight),
		)
		text = iup.Text(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
			fmt.Sprintf("FGCOLOR=\"%s\"", labelFGColor),
			fmt.Sprintf("BGCOLOR=\"%s\"", commandBGColor),
			"VALUE=\"\"",
			"BORDER=NO",
			"ALIGNMENT=ALEFT",
			fmt.Sprintf("SIZE=%dx%d", datatextwidth, dataheight),
		)
		text.SetCallback(func(arg *iup.CommonKeyAny) {
			key := iup.KeyState(arg.Key)
			switch key.Key() {
			case KEY_ESCAPE:
				*er = errors.New("Query: Escaped")
				dlg.Destroy()
				return
			case KEY_ENTER:
				returnvalues()
				dlg.Destroy()
				return
			}
		})
		dlg = iup.Dialog(iup.Hbox(label, text))
		dlg.SetAttribute("TITLE", title)
		dlg.Popup(iup.CENTER, iup.CENTER)
	}(question, &ans, &er)
	return ans, er
}

func (stw *Window) QueryList(question string) (rtn []string, err error) {
	ans, err := stw.Query(question)
	if err != nil {
		return nil, err
	}
	tmp := strings.Split(ans, ",")
	l := len(tmp)
	rtn = make([]string, l)
	for i := 0; i < l; i++ {
		rtn[i] = strings.TrimLeft(tmp[i], " ")
	}
	return rtn, err
}

func (stw *Window) QueryCoord(title string) (x, y, z float64, err error) {
	var xx, yy, zz float64
	var er error
	func(title string, x, y, z *float64, er *error) {
		labels := make([]*iup.Handle, 3)
		texts := make([]*iup.Handle, 3)
		var dlg *iup.Handle
		returnvalues := func() {
			rtn := make([]float64, 3)
			for i := 0; i < 3; i++ {
				val := texts[i].GetAttribute("VALUE")
				if val == "" {
					rtn[i] = 0.0
				} else {
					tmp, err := strconv.ParseFloat(val, 64)
					if err != nil {
						*er = err
						dlg.Destroy()
						return
					}
					rtn[i] = tmp
				}
			}
			*x = rtn[0]
			*y = rtn[1]
			*z = rtn[2]
			*er = nil
		}
		for i, d := range []string{"X", "Y", "Z"} {
			labels[i] = datalabel(d)
			texts[i] = datatext("")
			texts[i].SetCallback(func(arg *iup.CommonKeyAny) {
				key := iup.KeyState(arg.Key)
				switch key.Key() {
				case KEY_ESCAPE:
					*er = errors.New("QueryCoord: Escaped")
					dlg.Destroy()
					return
				case KEY_ENTER:
					returnvalues()
					dlg.Destroy()
					return
				}
			})
		}
		dlg = iup.Dialog(
			iup.Hbox(labels[0], texts[0], labels[1], texts[1], labels[2], texts[2]),
		)
		dlg.SetAttribute("TITLE", title)
		dlg.Map()
		dlg.Popup(iup.CENTER, iup.CENTER)
	}(title, &xx, &yy, &zz, &er)
	return xx, yy, zz, er
}

// }}}

// Default CallBack// {{{
func (stw *Window) CB_MouseButton() {
	stw.canv.SetCallback(func(arg *iup.MouseButton) {
		if stw.frame != nil {
			switch arg.Button {
			case BUTTON_LEFT:
				if isAlt(arg.Status) == ALTSELECTNODE {
					stw.SelectNodeStart(arg)
				} else {
					stw.SelectElemStart(arg)
					if isDouble(arg.Status) {
						if stw.ElemSelected() {
							if stw.selectElem[0].IsLineElem() {
								stw.ExecCommand(DoubleClickCommand[0])
							} else {
								stw.ExecCommand(DoubleClickCommand[1])
							}
						}
					}
				}
			case BUTTON_CENTER:
				if arg.Pressed == 0 { // Released
					stw.Redraw()
				} else { // Pressed
					if isDouble(arg.Status) {
						if isAlt(arg.Status) {
							for i := 0; i < 2; i++ {
								RangeView.Angle[i] = stw.frame.View.Angle[i]
							}
						}
						stw.ShowCenter()
					} else {
						stw.dbuff.UpdateYAxis(&arg.Y)
						stw.startX = int(arg.X)
						stw.startY = int(arg.Y)
					}
				}
			case BUTTON_RIGHT:
				if arg.Pressed == 0 {
					if v := stw.cline.GetAttribute("VALUE"); v != "" {
						stw.feedCommand()
					} else {
						if time.Since(pressed).Seconds() < repeatcommand {
							if isShift(arg.Status) {
								if stw.lastexcommand != "" {
									st.ExMode(stw, stw.lastexcommand)
									stw.Redraw()
								}
							} else {
								if stw.lastcommand != nil {
									stw.execCommand(stw.lastcommand)
								}
							}
						} else {
							stw.context.Popup(iup.MOUSEPOS, iup.MOUSEPOS)
						}
					}
				} else {
					pressed = time.Now()
				}
			}
		} else {
			switch arg.Button {
			case BUTTON_LEFT:
				if isDouble(arg.Status) {
					stw.Open()
				}
			case BUTTON_CENTER:
				stw.ReadRecent()
			}
		}
	})
}

func (stw *Window) MoveOrRotate(arg *iup.MouseMotion) {
	if !fixMove && (isShift(arg.Status) || fixRotate) {
		if isAlt(arg.Status) {
			RangeView.Center[0] += float64(int(arg.X)-stw.startX) * stw.CanvasMoveSpeedX()
			RangeView.Center[1] += float64(int(arg.Y)-stw.startY) * stw.CanvasMoveSpeedY()
			stw.DrawRange(stw.dbuff, RangeView)
		} else {
			stw.frame.View.Center[0] += float64(int(arg.X)-stw.startX) * stw.CanvasMoveSpeedX()
			stw.frame.View.Center[1] += float64(int(arg.Y)-stw.startY) * stw.CanvasMoveSpeedY()
			stw.DrawFrameNode()
		}
	} else if !fixRotate {
		if isAlt(arg.Status) {
			RangeView.Angle[0] -= float64(int(arg.Y)-stw.startY) * stw.CanvasRotateSpeedY()
			RangeView.Angle[1] -= float64(int(arg.X)-stw.startX) * stw.CanvasRotateSpeedX()
			stw.DrawRange(stw.dbuff, RangeView)
		} else {
			stw.frame.View.Angle[0] -= float64(int(arg.Y)-stw.startY) * stw.CanvasRotateSpeedY()
			stw.frame.View.Angle[1] -= float64(int(arg.X)-stw.startX) * stw.CanvasRotateSpeedX()
			stw.DrawFrameNode()
		}
	}
}

func (stw *Window) CB_MouseMotion() {
	stw.canv.SetCallback(func(arg *iup.MouseMotion) {
		if stw.frame != nil {
			stw.dbuff.UpdateYAxis(&arg.Y)
			switch statusKey(arg.Status) {
			case STATUS_LEFT:
				if isAlt(arg.Status) == ALTSELECTNODE {
					stw.SelectNodeMotion(arg)
				} else {
					stw.SelectElemMotion(arg)
				}
			case STATUS_CENTER:
				stw.MoveOrRotate(arg)
			}
		}
	})
}

func (stw *Window) CB_CanvasWheel() {
	stw.canv.SetCallback(func(arg *iup.CanvasWheel) {
		if stw.frame != nil {
			stw.dbuff.UpdateYAxis(&arg.Y)
			x := arg.X
			if x > 65535 {
				x -= 65535
			}
			for _, tb := range stw.textBox {
				if tb.Contains(float64(x), float64(arg.Y)) {
					if arg.Delta >= 0 {
						tb.ScrollUp(1)
					} else {
						tb.ScrollDown(1)
					}
					stw.Redraw()
					return
				}
			}
			val := math.Pow(2.0, float64(arg.Delta)/stw.CanvasScaleSpeed())
			var v *st.View
			if isAlt(arg.Status) {
				v = RangeView
			} else {
				v = stw.frame.View
			}
			if !isCtrl(arg.Status) {
				v.Center[0] += (val - 1.0) * (v.Center[0] - float64(x))
				v.Center[1] += (val - 1.0) * (v.Center[1] - float64(arg.Y))
			}
			if v.Perspective {
				v.Dists[1] *= val
				if v.Dists[1] < 0.0 {
					v.Dists[1] = 0.0
				}
			} else {
				v.Gfact *= val
				if v.Gfact < 0.0 {
					v.Gfact = 0.0
				}
			}
			stw.Redraw()
		}
	})
}

func (stw *Window) DefaultKeyAny(arg *iup.CommonKeyAny) {
	key := iup.KeyState(arg.Key)
	switch key.Key() {
	default:
		// fmt.Println(key.Key())
		stw.cline.SetAttribute("APPEND", string(key.Key()))
		iup.SetFocus(stw.cline)
	case '/':
		if stw.cline.GetAttribute("VALUE") != "" {
			stw.cline.SetAttribute("APPEND", "/")
		} else {
			stw.SearchInp()
		}
	case '"':
		if stw.frame != nil {
			switch stw.frame.Project {
			default:
				stw.cline.SetAttribute("APPEND", "\"")
			case "venhira":
				stw.cline.SetAttribute("APPEND", "V4")
			}
		}
	case '\'':
		val := stw.cline.GetAttribute("VALUE")
		if val == "" {
			stw.completefunc = stw.CompleteFig2Keyword
		}
		stw.cline.SetAttribute("APPEND", "'")
	case ':':
		val := stw.cline.GetAttribute("VALUE")
		if val == "" {
			stw.cline.SetAttribute("APPEND", ";")
		} else {
			stw.cline.SetAttribute("APPEND", ":")
		}
	case ';':
		val := stw.cline.GetAttribute("VALUE")
		if val == "" {
			stw.cline.SetAttribute("APPEND", ":")
			stw.completefunc = stw.CompleteExcommand
		} else {
			stw.cline.SetAttribute("APPEND", ";")
		}
	case KEY_BS:
		val := stw.cline.GetAttribute("VALUE")
		if val != "" {
			stw.cline.SetAttribute("VALUE", val[:len(val)-1])
		}
	case KEY_DELETE:
		stw.DeleteSelected()
	case KEY_ENTER:
		stw.feedCommand()
	case KEY_ESCAPE:
		stw.EscapeAll()
	case KEY_TAB:
		if key.IsCtrl() {
			stw.NextSideBarTab()
		} else {
			lis := strings.Split(stw.cline.GetAttribute("VALUE"), " ")
			tmp := lis[len(lis)-1]
			switch {
			case strings.HasPrefix(tmp, ":"):
				lis[len(lis)-1] = stw.CompleteExcommand(tmp)
			case strings.HasPrefix(tmp, "'"):
				lis[len(lis)-1] = stw.CompleteFig2Keyword(tmp)
			default:
				lis[len(lis)-1] = stw.CompleteFileName(tmp)
			}
			stw.cline.SetAttribute("VALUE", strings.Join(lis, " "))
			stw.cline.SetAttribute("CARETPOS", "100")
		}
	case KEY_UPARROW:
		if key.IsCtrl() {
			stw.NextFloor()
		} else {
			if !(prevkey == KEY_UPARROW || prevkey == KEY_DOWNARROW) {
				clineinput = stw.cline.GetAttribute("VALUE")
			}
			stw.PrevCommand(clineinput)
			arg.Return = int32(iup.IGNORE)
		}
	case KEY_DOWNARROW:
		if key.IsCtrl() {
			stw.PrevFloor()
		} else {
			if !(prevkey == KEY_UPARROW || prevkey == KEY_DOWNARROW) {
				clineinput = stw.cline.GetAttribute("VALUE")
			}
			stw.NextCommand(clineinput)
			arg.Return = int32(iup.IGNORE)
		}
	case KEY_LEFTARROW:
		iup.SetFocus(stw.cline)
		current := stw.cline.GetAttribute("CARETPOS")
		val, err := strconv.ParseInt(current, 10, 64)
		if err != nil {
			return
		}
		var pos int
		if val == 0 {
			pos = 0
		} else {
			pos = int(val) - 1
		}
		stw.cline.SetAttribute("CARETPOS", fmt.Sprintf("%d", pos))
	case KEY_RIGHTARROW:
		iup.SetFocus(stw.cline)
		current := stw.cline.GetAttribute("CARETPOS")
		val, err := strconv.ParseInt(current, 10, 64)
		if err != nil {
			return
		}
		pos := int(val) + 1
		stw.cline.SetAttribute("CARETPOS", fmt.Sprintf("%d", pos))
	case 'N':
		if key.IsCtrl() {
			if stw.frame != nil {
				if strings.Contains(stw.frame.Show.Period, "-") {
					stw.SetPeriod(strings.Replace(stw.frame.Show.Period, "-", "+", -1))
					stw.Redraw()
				} else if strings.Contains(stw.frame.Show.Period, "+") {
					stw.SetPeriod(strings.Replace(stw.frame.Show.Period, "+", "-", -1))
					stw.Redraw()
				}
			}
			// stw.New()
		}
	case 'O':
		if key.IsCtrl() {
			stw.Open()
		}
	case 'I':
		if key.IsCtrl() {
			stw.Insert()
		}
	case 'P':
		if key.IsCtrl() {
			stw.Print()
		}
	case 'M':
		if key.IsCtrl() {
			// stw.Reload()
			if keymode == NORMAL {
				keymode = VIEWEDIT
				stw.addHistory("VIEWEDIT")
			} else if keymode == VIEWEDIT {
				keymode = NORMAL
				stw.addHistory("NORMAL")
			}
		}
	case 'F':
		if key.IsCtrl() {
			stw.SetFocus()
		}
	case 'D':
		if key.IsCtrl() {
			stw.HideNotSelected()
		}
	case 'S':
		if key.IsCtrl() {
			// stw.Save()
			stw.ShowAll()
		}
	case 'H':
		switch keymode {
		case NORMAL:
			if key.IsCtrl() {
				stw.HideSelected()
			}
		case VIEWEDIT:
			if key.IsCtrl() {
				stw.frame.View.Angle[1] += 5
				stw.Redraw()
			} else if key.IsAlt() {
				stw.frame.View.Center[0] -= 5
				stw.Redraw()
			}
		}
	case 'J':
		switch keymode {
		case VIEWEDIT:
			if key.IsCtrl() {
				stw.frame.View.Angle[0] += 5
				stw.Redraw()
			} else if key.IsAlt() {
				stw.frame.View.Center[1] -= 5
				stw.Redraw()
			}
		}
	case 'K':
		switch keymode {
		case VIEWEDIT:
			if key.IsCtrl() {
				stw.frame.View.Angle[0] -= 5
				stw.Redraw()
			} else if key.IsAlt() {
				stw.frame.View.Center[1] += 5
				stw.Redraw()
			}
		}
	case 'L':
		switch keymode {
		case NORMAL:
			if key.IsCtrl() {
				stw.LockSelected()
			}
		case VIEWEDIT:
			if key.IsCtrl() {
				stw.frame.View.Angle[1] -= 5
				stw.Redraw()
			} else if key.IsAlt() {
				stw.frame.View.Center[0] += 5
				stw.Redraw()
			}
		}
	case 'U':
		if key.IsCtrl() {
			if stw.frame != nil {
				if stw.frame.Show.Unit[0] == 1.0 && stw.frame.Show.Unit[1] == 1.0 {
					st.Fig2Keyword(stw, []string{"unit", "kN,m"}, false)
				} else {
					st.Fig2Keyword(stw, []string{"unit", "tf,m"}, false)
				}
				stw.Redraw()
			}
			// stw.UnlockAll()
		}
	case 'A':
		if key.IsCtrl() {
			// stw.ShowAll()
			stw.SelectNotHidden()
		}
	case 'R':
		if key.IsCtrl() {
			st.ReadAll(stw)
		}
	case 'E':
		if key.IsCtrl() {
			stw.EditInp()
		}
	case 'C':
		if key.IsCtrl() {
			stw.CopyClipboard()
		}
	case 'V':
		if key.IsCtrl() {
			stw.PasteClipboard()
		}
	case 'X':
		if key.IsCtrl() {
			if stw.sectiondlg != nil {
				iup.SetFocus(stw.sectiondlg)
			} else {
				stw.SectionDialog()
			}
		}
	case 'Y':
		if key.IsCtrl() {
			frame, err := stw.Redo()
			if err != nil {
				st.ErrorMessage(stw, err, st.ERROR)
			} else {
				stw.frame = frame
			}
		}
	case 'Z':
		switch keymode {
		case NORMAL:
			if key.IsCtrl() {
				frame, err := stw.Undo()
				if err != nil {
					st.ErrorMessage(stw, err, st.ERROR)
				} else {
					stw.frame = frame
				}
			}
		case VIEWEDIT:
			var val float64
			if key.IsCtrl() {
				val = math.Pow(2.0, 1.0/stw.CanvasScaleSpeed())
			} else if key.IsAlt() {
				val = math.Pow(2.0, -1.0/stw.CanvasScaleSpeed())
			} else {
				break
			}
			if stw.frame.View.Perspective {
				stw.frame.View.Dists[1] *= val
			} else {
				stw.frame.View.Gfact *= val
			}
			stw.Redraw()
		}
	}
	prevkey = key.Key()
}

func (stw *Window) CB_CommonKeyAny() {
	stw.canv.SetCallback(func(arg *iup.CommonKeyAny) {
		stw.DefaultKeyAny(arg)
	})
}

func (stw *Window) LinkProperty(index int, eval func()) {
	stw.Props[index].SetCallback(func(arg *iup.CommonGetFocus) {
		stw.Props[index].SetAttribute("SELECTION", "1:100")
	})
	stw.Props[index].SetCallback(func(arg *iup.CommonKillFocus) {
		if stw.frame != nil && stw.selectElem != nil {
			eval()
		}
	})
	stw.Props[index].SetCallback(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		switch key.Key() {
		case KEY_ENTER:
			if stw.frame != nil && stw.selectElem != nil {
				eval()
			}
		case KEY_TAB:
			if stw.frame != nil && stw.selectElem != nil {
				eval()
			}
		case KEY_ESCAPE:
			stw.UpdatePropertyDialog()
			iup.SetFocus(stw.canv)
		}
	})
}

func (stw *Window) PropertyDialog() {
	stw.Selected = make([]*iup.Handle, 4)
	stw.Props = make([]*iup.Handle, 8)
	for i := 0; i < 4; i++ {
		stw.Selected[i] = datatext("-")
	}
	for i := 0; i < 8; i++ {
		stw.Props[i] = datatext("-")
	}
	stw.LinkProperty(1, func() {
		val, err := strconv.ParseInt(stw.Props[1].GetAttribute("VALUE"), 10, 64)
		if err == nil {
			if sec, ok := stw.frame.Sects[int(val)]; ok {
				for _, el := range stw.selectElem {
					if el != nil && !el.Lock {
						el.Sect = sec
					}
				}
			}
		}
		stw.Redraw()
	})
	stw.LinkProperty(2, func() {
		word := stw.Props[2].GetAttribute("VALUE")
		var val int
		switch {
		case st.Re_column.MatchString(word):
			val = st.COLUMN
		case st.Re_girder.MatchString(word):
			val = st.GIRDER
		case st.Re_brace.MatchString(word):
			val = st.BRACE
		case st.Re_wall.MatchString(word):
			val = st.WALL
		case st.Re_slab.MatchString(word):
			val = st.SLAB
		}
		stw.Props[2].SetAttribute("VALUE", st.ETYPES[val])
		if val != 0 {
			for _, el := range stw.selectElem {
				if el != nil && !el.Lock {
					el.Etype = val
				}
			}
		}
		stw.Redraw()
	})
	stw.Property = true
}

func (stw *Window) UpdatePropertyDialog() {
	if stw.selectElem != nil {
		var selected bool
		var lines, plates int
		var length, area float64
		for _, el := range stw.selectElem {
			if el != nil {
				if el.IsLineElem() {
					lines++
					length += el.Length()
				} else {
					plates++
					area += el.Area()
				}
				if !selected {
					stw.Props[0].SetAttribute("VALUE", fmt.Sprintf("%d", el.Num))
					stw.Props[1].SetAttribute("VALUE", fmt.Sprintf("%d", el.Sect.Num))
					stw.Props[2].SetAttribute("VALUE", st.ETYPES[el.Etype])
					stw.Props[3].SetAttribute("VALUE", fmt.Sprintf("%d", el.Enods))
					for i := 0; i < el.Enods; i++ {
						if i >= 4 {
							break
						}
						stw.Props[4+i].SetAttribute("VALUE", fmt.Sprintf("%d", el.Enod[i].Num))
					}
				}
				selected = true
			}
		}
		if !selected {
			for i := 0; i < 4; i++ {
				stw.Selected[i].SetAttribute("VALUE", "-")
			}
			for i := 0; i < 8; i++ {
				stw.Props[i].SetAttribute("VALUE", "-")
			}
		}
		if lines > 0 {
			stw.Selected[0].SetAttribute("VALUE", fmt.Sprintf("%d", lines))
			stw.Selected[1].SetAttribute("VALUE", fmt.Sprintf("%.3f", length))
		} else {
			stw.Selected[0].SetAttribute("VALUE", "-")
			stw.Selected[1].SetAttribute("VALUE", "-")
		}
		if plates > 0 {
			stw.Selected[2].SetAttribute("VALUE", fmt.Sprintf("%d", plates))
			stw.Selected[3].SetAttribute("VALUE", fmt.Sprintf("%.3f", area))
		} else {
			stw.Selected[2].SetAttribute("VALUE", "-")
			stw.Selected[3].SetAttribute("VALUE", "-")
		}
	} else {
		for i := 0; i < 4; i++ {
			stw.Selected[i].SetAttribute("VALUE", "-")
		}
		for i := 0; i < 8; i++ {
			stw.Props[i].SetAttribute("VALUE", "-")
		}
	}
}

func (stw *Window) CMenu() {
	stw.context = iup.Menu(
		iup.Attrs("BGCOLOR", labelBGColor),
		iup.SubMenu("TITLE=File",
			iup.Menu(
				iup.Item(
					iup.Attr("TITLE", "Open\tCtrl+O"),
					iup.Attr("TIP", "Open file"),
					func(arg *iup.ItemAction) {
						runtime.GC()
						stw.Open()
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Search Inp\t/"),
					iup.Attr("TIP", "Search Inp file"),
					func(arg *iup.ItemAction) {
						runtime.GC()
						stw.SearchInp()
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Insert\tCtrl+I"),
					iup.Attr("TIP", "Insert frame"),
					func(arg *iup.ItemAction) {
						stw.Insert()
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Reload\tCtrl+M"),
					func(arg *iup.ItemAction) {
						runtime.GC()
						st.Reload(stw)
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Save\tCtrl+S"),
					iup.Attr("TIP", "Save file"),
					func(arg *iup.ItemAction) {
						stw.Save()
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Save As"),
					iup.Attr("TIP", "Save file As"),
					func(arg *iup.ItemAction) {
						stw.SaveAS()
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Read"),
					iup.Attr("TIP", "Read file"),
					func(arg *iup.ItemAction) {
						stw.Read()
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Read All\tCtrl+R"),
					iup.Attr("TIP", "Read all file"),
					func(arg *iup.ItemAction) {
						st.ReadAll(stw)
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Print"),
					func(arg *iup.ItemAction) {
						stw.Print()
					},
				),
				iup.Separator(),
				iup.Item(
					iup.Attr("TITLE", "Quit"),
					iup.Attr("TIP", "Exit Application"),
					func(arg *iup.ItemAction) {
						if stw.changed {
							if stw.Yn("CHANGED", "変更を保存しますか") {
								stw.SaveAS()
							} else {
								return
							}
						}
						stw.SaveRecent()
						arg.Return = iup.CLOSE
					},
				),
			),
		),
		iup.SubMenu("TITLE=Edit",
			iup.Menu(
				iup.Item(
					iup.Attr("TITLE", "Command Font"),
					func(arg *iup.ItemAction) {
						stw.SetFont(FONT_COMMAND)
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Canvas Font"),
					func(arg *iup.ItemAction) {
						stw.SetFont(FONT_CANVAS)
					},
				),
			),
		),
		iup.SubMenu("TITLE=View",
			iup.Menu(
				iup.Item(
					iup.Attr("TITLE", "Top"),
					func(arg *iup.ItemAction) {
						stw.SetAngle(90.0, -90.0)
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Front"),
					func(arg *iup.ItemAction) {
						stw.SetAngle(0.0, -90.0)
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Back"),
					func(arg *iup.ItemAction) {
						stw.SetAngle(0.0, 90.0)
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Right"),
					func(arg *iup.ItemAction) {
						stw.SetAngle(0.0, 0.0)
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Left"),
					func(arg *iup.ItemAction) {
						stw.SetAngle(0.0, 180.0)
					},
				),
			),
		),
		iup.SubMenu("TITLE=Show",
			iup.Menu(
				iup.Item(
					iup.Attr("TITLE", "Show All\tCtrl+S"),
					func(arg *iup.ItemAction) {
						stw.ShowAll()
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Hide Selected Elems\tCtrl+H"),
					func(arg *iup.ItemAction) {
						stw.HideSelected()
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Show Selected Elems\tCtrl+D"),
					func(arg *iup.ItemAction) {
						stw.HideNotSelected()
					},
				),
			),
		),
		iup.SubMenu("TITLE=Tool",
			iup.Menu(
				iup.Item(
					iup.Attr("TITLE", "RC lst"),
					func(arg *iup.ItemAction) {
						st.StartTool("rclst/rclst.html")
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Fig2 Keyword"),
					func(arg *iup.ItemAction) {
						st.StartTool("fig2/fig2.html")
					},
				),
			),
		),
		iup.SubMenu("TITLE=Help",
			iup.Menu(
				iup.Item(
					iup.Attr("TITLE", "Release Note"),
					func(arg *iup.ItemAction) {
						ShowReleaseNote()
					},
				),
				iup.Item(
					iup.Attr("TITLE", "About"),
					func(arg *iup.ItemAction) {
						stw.ShowAbout()
					},
				),
			),
		),
	)
}

func (stw *Window) PeriodDialog(defval string) {
	if stw.frame == nil {
		return
	}
	lis := strings.Split(defval, "@")
	var defpname, deflap string
	if len(lis) >= 2 {
		defpname = strings.ToUpper(lis[0])
		deflap = lis[1]
	} else {
		defpname = strings.ToUpper(lis[0])
		deflap = "1"
	}
	per := iup.Text(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
		fmt.Sprintf("FGCOLOR=\"%s\"", labelFGColor),
		fmt.Sprintf("BGCOLOR=\"%s\"", commandBGColor),
		fmt.Sprintf("VALUE=\"%s\"", defpname),
		"BORDER=NO",
		"ALIGNMENT=ARIGHT",
		fmt.Sprintf("SIZE=%dx%d", datatextwidth, dataheight))
	per.SetCallback(func(arg *iup.CommonGetFocus) {
		per.SetAttribute("SELECTION", "1:100")
	})
	var max int
	if n, ok := stw.frame.Nlap[defpname]; ok {
		max = n
	} else {
		max = 100
	}
	if max <= 1 {
		max = 2
	}
	lap := iup.Val("MIN=1",
		fmt.Sprintf("MAX=%d", max),
		fmt.Sprintf("STEP=%.2f", 1/(float64(max)-1)),
		fmt.Sprintf("VALUE=%s", deflap))
	lap.SetCallback(func(arg *iup.ValueChanged) {
		pname := fmt.Sprintf("%s@%s", strings.ToUpper(per.GetAttribute("VALUE")), strings.Split(lap.GetAttribute("VALUE"), ".")[0])
		stw.Labels["PERIOD"].SetAttribute("VALUE", pname)
		stw.frame.Show.Period = pname
		stw.Redraw()
	})
	dlg := iup.Dialog(iup.Hbox(per, lap))
	dlg.SetAttribute("TITLE", "Period")
	dlg.SetAttribute("DIALOGFRAME", "YES")
	dlg.SetAttribute("PARENTDIALOG", "mainwindow")
	dlg.Map()
	dlg.ShowXY(iup.MOUSEPOS, iup.MOUSEPOS)
}

func (stw *Window) SectionDialog() {
	if stw.frame == nil {
		return
	}
	sects := make([]*st.Sect, len(stw.frame.Sects))
	nsect := 0
	for _, sec := range stw.frame.Sects {
		if sec.Num >= 900 {
			continue
		}
		sects[nsect] = sec
		nsect++
	}
	sects = sects[:nsect]
	sort.Sort(st.SectByNum{sects})
	selstart := -1
	selend := -1
	lines := make([]*iup.Handle, nsect)
	codes := make([]*iup.Handle, nsect)
	snames := make([]*iup.Handle, nsect)
	hides := make([]*iup.Handle, nsect)
	colors := make([]*iup.Handle, nsect)
	sections := iup.Vbox()
	iup.Menu(
		iup.Attrs("FGCOLOR", sectiondlgFGColor),
		iup.Attrs("BGCOLOR", labelBGColor),
		iup.Item("TITLE=\"＋\"",
			func(arg *iup.ItemAction) {
				ans, err := stw.Query("SECT CODE")
				if err != nil {
					return
				}
				val, err := strconv.ParseInt(ans, 10, 64)
				if err != nil {
					return
				}
				if _, exist := stw.frame.Sects[int(val)]; exist {
					stw.addHistory(fmt.Sprintf("SECT: %d already exists", int(val)))
					return
				}
				sec := stw.frame.AddSect(int(val))
				sects = append(sects, sec)
				code := iup.Label(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, sectiondlgFontSize),
					fmt.Sprintf("FGCOLOR=\"%s\"", sectiondlgFGColor),
					fmt.Sprintf("TITLE=\"%d\"", sec.Num),
					fmt.Sprintf("SIZE=%dx%d", 20, dataheight))
				sname := iup.Label(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, sectiondlgFontSize),
					fmt.Sprintf("FGCOLOR=\"%s\"", sectiondlgFGColor),
					fmt.Sprintf("TITLE=\"%s\"", sec.Name),
					fmt.Sprintf("SIZE=%dx%d", 200, dataheight))
				hide := iup.Toggle("VALUE=ON", "CANFOCUS=NO", fmt.Sprintf("SIZE=%dx%d", 25, dataheight))
				color := iup.Canvas(fmt.Sprintf("SIZE=\"%dx%d\"", 20, dataheight), "BGCOLOR=\"255 255 255\"", "EXPAND=NO")
				line := iup.Hbox(code, sname, hide, color)
				codes = append(codes, code)
				snames = append(snames, sname)
				hides = append(hides, hide)
				colors = append(colors, color)
				lines = append(lines, line)
				sections.Append(line)
				stw.sectiondlg.Map() // TODO: display new section on the dialog
			}),
		iup.Item("TITLE=\"－\"",
			func(arg *iup.ItemAction) {
				if selstart < 0 {
					return
				}
			deletesect:
				for i := selstart; i < selend+1; i++ {
					for _, el := range stw.frame.Elems {
						if el.Sect.Num == sects[i].Num {
							continue deletesect
						}
					}
					stw.frame.DeleteSect(sects[i].Num)
					codes[i].SetAttribute("ACTIVE", "NO")
					snames[i].SetAttribute("ACTIVE", "NO")
					hides[i].SetAttribute("ACTIVE", "NO")
					colors[i].SetAttribute("ACTIVE", "NO")
				}
			}),
	).SetName("sectiondlg_menu")
	title := iup.Hbox(
		iup.Label(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, sectiondlgFontSize),
			fmt.Sprintf("FGCOLOR=\"%s\"", sectiondlgFGColor),
			fmt.Sprintf("SIZE=%dx%d", 20, dataheight),
			"TITLE=\"CODE\""),
		iup.Label(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, sectiondlgFontSize),
			fmt.Sprintf("FGCOLOR=\"%s\"", sectiondlgFGColor),
			fmt.Sprintf("SIZE=%dx%d", 200, dataheight),
			"TITLE=\"NAME\""),
		iup.Label(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, sectiondlgFontSize),
			fmt.Sprintf("FGCOLOR=\"%s\"", sectiondlgFGColor),
			fmt.Sprintf("SIZE=%dx%d", 25, dataheight),
			"TITLE=\"表示\""),
		iup.Label(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, sectiondlgFontSize),
			fmt.Sprintf("FGCOLOR=\"%s\"", sectiondlgFGColor),
			fmt.Sprintf("SIZE=%dx%d", 20, dataheight),
			"TITLE=\"色\""))
	for i, sec := range sects {
		codes[i] = iup.Label(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, sectiondlgFontSize),
			fmt.Sprintf("FGCOLOR=\"%s\"", sectiondlgFGColor),
			fmt.Sprintf("TITLE=\"%d\"", sec.Num),
			fmt.Sprintf("SIZE=%dx%d", 20, dataheight),
		)
		snames[i] = iup.Label(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, sectiondlgFontSize),
			fmt.Sprintf("FGCOLOR=\"%s\"", sectiondlgFGColor),
			fmt.Sprintf("TITLE=\"%s\"", sec.Name),
			fmt.Sprintf("SIZE=%dx%d", 200, dataheight),
		)
		func(snum, num int) {
			mbcb := func(arg *iup.MouseButton) {
				if isShift(arg.Status) {
					if selstart < 0 {
						selstart = num
						selend = num
					} else {
						if num < selstart {
							selstart = num
						} else if num > selend {
							selend = num
						}
					}
				} else {
					selstart = num
					selend = num
				}
				if isDouble(arg.Status) {
					stw.SectionProperty(stw.frame.Sects[snum])
				}
				for j := 0; j < nsect; j++ {
					if selstart <= j && j <= selend {
						codes[j].SetAttribute("FGCOLOR", sectiondlgSelectColor)
						snames[j].SetAttribute("FGCOLOR", sectiondlgSelectColor)
					} else {
						codes[j].SetAttribute("FGCOLOR", sectiondlgFGColor)
						snames[j].SetAttribute("FGCOLOR", sectiondlgFGColor)
					}
				}
			}
			codes[num].SetCallback(mbcb)
			snames[num].SetCallback(mbcb)
		}(sec.Num, i)
		var ishide string
		if stw.frame.Show.Sect[sec.Num] {
			ishide = "ON"
		} else {
			ishide = "OFF"
		}
		hides[i] = iup.Toggle(fmt.Sprintf("VALUE=%s", ishide),
			"CANFOCUS=NO",
			fmt.Sprintf("SIZE=%dx%d", 25, dataheight))
		func(snum, num int) {
			hides[num].SetCallback(func(arg *iup.ToggleAction) {
				if stw.frame != nil {
					if arg.State == 1 {
						if selstart <= num && num <= selend {
							for j := selstart; j < selend+1; j++ {
								if hides[j].GetAttribute("ACTIVE") == "NO" {
									continue
								}
								stw.frame.Show.Sect[sects[j].Num] = true
								hides[j].SetAttribute("VALUE", "ON")
							}
						} else {
							stw.frame.Show.Sect[snum] = true
						}
					} else {
						if selstart <= num && num <= selend {
							for j := selstart; j < selend+1; j++ {
								if hides[j].GetAttribute("ACTIVE") == "NO" {
									continue
								}
								stw.frame.Show.Sect[sects[j].Num] = false
								hides[j].SetAttribute("VALUE", "OFF")
							}
						} else {
							stw.frame.Show.Sect[snum] = false
						}
					}
					stw.Redraw()
				}
			})
		}(sec.Num, i)
		colors[i] = iup.Canvas(fmt.Sprintf("SIZE=\"%dx%d\"", 20, dataheight),
			fmt.Sprintf("BGCOLOR=\"%s\"", st.IntColor(sec.Color)),
			"EXPAND=NO")
		func(snum, num int) {
			colors[num].SetCallback(func(arg *iup.MouseButton) {
				if arg.Pressed == 0 {
					col, err := stw.ColorDialog()
					if err != nil {
						return
					}
					stw.frame.Sects[snum].Color = col
					colors[num].SetAttribute("BGCOLOR", st.IntColor(col))
				}
			})
		}(sec.Num, i)
		codes[i].SaveClassAttributes()
		lines[i] = iup.Hbox(codes[i], snames[i], hides[i], colors[i])
		sections.Append(lines[i])
	}
	stw.sectiondlg = iup.Dialog(iup.Vbox(title, iup.Label("SEPARATOR=HORIZONTAL"), iup.ScrollBox(sections, "SIZE=\"350x150\"")),
		fmt.Sprintf("BGCOLOR=\"%s\"", sectiondlgBGColor),
		"MENU=\"sectiondlg_menu\"",
		func(arg *iup.CommonKeyAny) {
			key := iup.KeyState(arg.Key)
			switch key.Key() {
			default:
				stw.FocusCanv()
				stw.DefaultKeyAny(arg)
			case KEY_ESCAPE:
				stw.sectiondlg.Destroy()
				stw.sectiondlg = nil
			case KEY_F4:
				if key.IsAlt() {
					stw.sectiondlg.Destroy()
					stw.sectiondlg = nil
				}
			}
		},
		func(arg *iup.DialogClose) {
			stw.sectiondlg = nil
		})
	stw.sectiondlg.SetAttribute("DIALOGFRAME", "YES")
	stw.sectiondlg.SetAttribute("OPACITY", "220")
	stw.sectiondlg.SetAttribute("PARENTDIALOG", "mainwindow")
	stw.sectiondlg.SetAttribute("TOOLBOX", "YES")
	iup.SetHandle("sectiondialog", stw.sectiondlg)
	stw.sectiondlg.Map()
	stw.sectiondlg.Show()
}

func (stw *Window) ColorDialog() (int, error) {
	var rtn int
	var err error
	func(col *int, er *error) {
		var colordlg *iup.Handle
		*er = errors.New("not selected")
		colors := iup.Vbox()
		browns := iup.Hbox()
		greys := iup.Hbox()
		bluegreys := iup.Hbox()
		// colnames := []string{"100", "200", "300", "400", "500", "600", "700", "800", "900"}
		colnames := []string{"200", "300", "400", "500", "600", "700", "800", "900"}
		// for _, gc := range []map[string]int{st.GOOGLE_RED, st.GOOGLE_PINK, st.GOOGLE_PURPLE, st.GOOGLE_DEEPPURPLE, st.GOOGLE_INDIGO, st.GOOGLE_BLUE, st.GOOGLE_LIGHTBLUE, st.GOOGLE_CYAN, st.GOOGLE_TEAL, st.GOOGLE_GREEN, st.GOOGLE_LIGHTGREEN, st.GOOGLE_LIME, st.GOOGLE_YELLOW, st.GOOGLE_AMBER, st.GOOGLE_ORANGE, st.GOOGLE_DEEPORANGE, st.GOOGLE_BROWN, st.GOOGLE_BLUEGREY, st.GOOGLE_GREY} {
		for _, gc := range []map[string]int{st.GOOGLE_RED, st.GOOGLE_PURPLE, st.GOOGLE_INDIGO, st.GOOGLE_BLUE, st.GOOGLE_LIGHTBLUE, st.GOOGLE_GREEN, st.GOOGLE_YELLOW, st.GOOGLE_DEEPORANGE, st.GOOGLE_GREY} {
			func(color map[string]int) {
				tmp := iup.Hbox()
				for _, name := range colnames {
					func(n string) {
						tmp.Append(iup.Canvas(fmt.Sprintf("SIZE=\"%dx%d\"", 20, dataheight),
							fmt.Sprintf("BGCOLOR=\"%s\"", st.IntColor(color[n])),
							"EXPAND=NO",
							func(arg *iup.MouseButton) {
								if arg.Pressed == 0 {
									colordlg.SetAttribute("TITLE", fmt.Sprintf("Color: %s", st.IntColor(color[n])))
								} else {
									if isDouble(arg.Status) {
										*col = color[n]
										*er = nil
										colordlg.Destroy()
									}
								}
							}))
					}(name)
				}
				colors.Append(tmp)
			}(gc)
		}
		colors.Append(browns)
		colors.Append(bluegreys)
		colors.Append(greys)
		colordlg = iup.Dialog(colors)
		colordlg.SetAttribute("TITLE", "Color")
		colordlg.SetAttribute("DIALOGFRAME", "YES")
		colordlg.SetAttribute("PARENTDIALOG", "sectiondialog")
		colordlg.Map()
		colordlg.Popup(iup.CENTER, iup.CENTER)
	}(&rtn, &err)
	return rtn, err
}

func (stw *Window) SectionProperty(sc *st.Sect) {
	var data, linedata, platedata *iup.Handle
	var addfig *iup.Handle
	// var drawarea *iup.Handle
	// var cdcanv *cd.Canvas
	// cl := func(cvs *cd.Canvas, position []float64, scale float64, vertices [][]float64) {
	// 	cvs.Begin(cd.CD_CLOSED_LINES)
	// 	for _, v := range vertices {
	// 		if v == nil {
	// 			cvs.End()
	// 			cvs.Begin(cd.CD_CLOSED_LINES)
	// 			continue
	// 		}
	// 		fmt.Println(position[0]+v[0]*0.01*scale, position[1]+v[1]*0.01*scale)
	// 		cvs.FVertex(position[0]+v[0]*0.01*scale, position[1]+v[1]*0.01*scale)
	// 	}
	// 	cvs.End()
	// }
	// draw := func(s *st.Sect, cvs *cd.Canvas) {
	// 	if al, ok := stw.frame.Allows[s.Num]; ok {
	// 		scale := 1000.0
	// 		w, h := cdcanv.GetSize()
	// 		position := []float64{float64(w), float64(h)}
	// 		switch al.(type) {
	// 		case *st.SColumn:
	// 			sh := al.(*st.SColumn).Shape
	// 			switch sh.(type) {
	// 			case st.HKYOU, st.HWEAK, st.RPIPE, st.CPIPE, st.PLATE:
	// 				vertices := sh.Vertices()
	// 				cl(cvs, position, scale, vertices)
	// 			}
	// 		case *st.RCColumn:
	// 			rc := al.(*st.RCColumn)
	// 			vertices := rc.CShape.Vertices()
	// 			cl(cvs, position, scale, vertices)
	// 			for _, reins := range rc.Reins {
	// 				vertices = reins.Vertices()
	// 				cl(cvs, position, scale, vertices)
	// 			}
	// 		case *st.RCGirder:
	// 			rg := al.(*st.RCGirder)
	// 			vertices := rg.CShape.Vertices()
	// 			cl(cvs, position, scale, vertices)
	// 			for _, reins := range rg.Reins {
	// 				vertices = reins.Vertices()
	// 				cl(cvs, position, scale, vertices)
	// 			}
	// 		case *st.WoodColumn:
	// 			sh := al.(*st.WoodColumn).Shape
	// 			switch sh.(type) {
	// 			case st.PLATE:
	// 				vertices := sh.Vertices()
	// 				cl(cvs, position, scale, vertices)
	// 			}
	// 		}
	// 	}
	// }
	// drawarea = iup.Canvas(
	// 	"CANFOCUS=YES",
	// 	"EXPAND=HORIZONTAL",
	// 	"SIZE=100x100",
	// 	"BGCOLOR=\"0 0 0\"",
	// 	"BORDER=YES",
	// 	"EXPAND=NO",
	// 	func(arg *iup.CommonMap) {
	// 		cdcanv = cd.CreateCanvas(cd.CD_IUP, drawarea)
	// 		cdcanv.Foreground(cd.CD_WHITE)
	// 		cdcanv.Background(cd.CD_BLACK)
	// 		cdcanv.LineStyle(cd.CD_CONTINUOUS)
	// 		cdcanv.LineWidth(1)
	// 		cdcanv.Activate()
	// 		draw(sc, cdcanv)
	// 		cdcanv.Flush()
	// 	},
	// 	func(arg *iup.CanvasResize) {
	// 		cdcanv.Activate()
	// 		draw(sc, cdcanv)
	// 		cdcanv.Flush()
	// 	},
	// 	func(arg *iup.CanvasAction) {
	// 		cdcanv.Activate()
	// 		draw(sc, cdcanv)
	// 		cdcanv.Flush()
	// 	},
	// )
	dataset := make(map[string]*iup.Handle, 0)
	dataset["NAME"] = datatext("-")
	dataset["NAME"].SetAttribute("EXPAND", "HORIZONTAL")
	dataset["CODE"] = datatext("-")
	dataset["PROP"] = datatext("-")
	dataset["AREA"] = datatext("-")
	dataset["IXX"] = datatext("-")
	dataset["IYY"] = datatext("-")
	dataset["VEN"] = datatext("-")
	dataset["THICK"] = datatext("-")
	dataset["LLOAD0"] = datatext("-")
	dataset["LLOAD1"] = datatext("-")
	dataset["LLOAD2"] = datatext("-")
	proplist := iup.List("SIZE=\"68x\"",
		"DROPDOWN=YES")
	linedata = iup.Vbox(iup.Hbox(propertylabel("AREA"), dataset["AREA"]),
		iup.Hbox(propertylabel("IXX"), dataset["IXX"]),
		iup.Hbox(propertylabel("IYY"), dataset["IYY"]),
		iup.Hbox(propertylabel("VEN"), dataset["VEN"]))
	platedata = iup.Vbox(iup.Hbox(propertylabel("THICK"), dataset["THICK"]),
		iup.Hbox(propertylabel("LLOAD"), dataset["LLOAD0"]),
		iup.Hbox(propertylabel(""), dataset["LLOAD1"]),
		iup.Hbox(propertylabel(""), dataset["LLOAD2"]))
	iup.SetHandle("linedata", linedata)
	iup.SetHandle("platedata", platedata)
	data = iup.Zbox(linedata, platedata)
	updatedata := func(sec *st.Sect, ind int) {
		if sec.Num > 700 && sec.Num < 900 {
			addfig.SetAttribute("ACTIVE", "YES")
			if len(sec.Figs) > ind {
				dataset["PROP"].SetAttribute("VALUE", fmt.Sprintf("%d", sec.Figs[ind].Prop.Num))
				dataset["THICK"].SetAttribute("VALUE", fmt.Sprintf("%.4f", sec.Figs[ind].Value["THICK"]))
				dataset["LLOAD0"].SetAttribute("VALUE", fmt.Sprintf("%.3f", sec.Lload[0]))
				dataset["LLOAD1"].SetAttribute("VALUE", fmt.Sprintf("%.3f", sec.Lload[1]))
				dataset["LLOAD2"].SetAttribute("VALUE", fmt.Sprintf("%.3f", sec.Lload[2]))
			} else {
				dataset["PROP"].SetAttribute("VALUE", "-")
				dataset["AREA"].SetAttribute("VALUE", "-")
				dataset["IXX"].SetAttribute("VALUE", "-")
				dataset["IYY"].SetAttribute("VALUE", "-")
				dataset["VEN"].SetAttribute("VALUE", "-")
			}
			data.SetAttribute("VALUE", "platedata")
		} else {
			addfig.SetAttribute("ACTIVE", "NO")
			if len(sec.Figs) > ind {
				dataset["PROP"].SetAttribute("VALUE", fmt.Sprintf("%d", sec.Figs[ind].Prop.Num))
				dataset["AREA"].SetAttribute("VALUE", fmt.Sprintf("%.4f", sec.Figs[ind].Value["AREA"]))
				dataset["IXX"].SetAttribute("VALUE", fmt.Sprintf("%.8f", sec.Figs[ind].Value["IXX"]))
				dataset["IYY"].SetAttribute("VALUE", fmt.Sprintf("%.8f", sec.Figs[ind].Value["IYY"]))
				dataset["VEN"].SetAttribute("VALUE", fmt.Sprintf("%.8f", sec.Figs[ind].Value["VEN"]))
			} else {
				dataset["PROP"].SetAttribute("VALUE", "-")
				dataset["AREA"].SetAttribute("VALUE", "-")
				dataset["IXX"].SetAttribute("VALUE", "-")
				dataset["IYY"].SetAttribute("VALUE", "-")
				dataset["VEN"].SetAttribute("VALUE", "-")
			}
			data.SetAttribute("VALUE", "linedata")
		}
	}
	updateproplist := func(sec *st.Sect) {
		proplist.SetAttribute("REMOVEITEM", "ALL")
		for _, f := range sec.Figs {
			proplist.SetAttribute("APPENDITEM", fmt.Sprintf("%d: %d", f.Num, f.Prop.Num))
		}
		proplist.SetAttribute("VALUE", "1")
	}
	proplist.SetCallback(func(arg *iup.ValueChanged) {
		var ind int
		fmt.Sscanf(proplist.GetAttribute("VALUE"), "%d", &ind)
		ind--
		updatedata(sc, ind)
	})
	addfig = iup.Button("TITLE=\"Add Figure\"", "ACTIVE=NO", "SIZE=\"60x\"")
	addfig.SetCallback(func(arg *iup.ButtonAction) {
		f := st.NewFig()
		f.Prop = stw.frame.DefaultProp()
		f.Num = len(sc.Figs) + 1
		sc.Figs = append(sc.Figs, f)
		updateproplist(sc)
		proplist.SetAttribute("VALUE", fmt.Sprintf("%d", f.Num))
		updatedata(sc, f.Num-1)
	})
	ctlg := iup.Button("TITLE=\"Catalog\"",
		"SIZE=\"100x20\"",
		"EXPAND=HORIZONTAL") // TODO: Section Catalog
	bt := iup.Button("TITLE=\"Set\"",
		"SIZE=\"100x20\"",
		"EXPAND=HORIZONTAL")
	bt.SetCallback(func(arg *iup.ButtonAction) {
		var ind int
		fmt.Sscanf(proplist.GetAttribute("VALUE"), "%d", &ind)
		ind--
		if sc.Num > 700 && sc.Num < 900 {
			if len(sc.Figs) > ind {
				var num int64
				var tmp float64
				var err error
				num, err = strconv.ParseInt(dataset["PROP"].GetAttribute("VALUE"), 10, 64)
				if err == nil {
					if p, ok := stw.frame.Props[int(num)]; ok {
						sc.Figs[ind].Prop = p
					}
				}
				tmp, err = strconv.ParseFloat(dataset["THICK"].GetAttribute("VALUE"), 64)
				if err == nil {
					sc.Figs[ind].Value["THICK"] = tmp
				}
				tmp, err = strconv.ParseFloat(dataset["LLOAD0"].GetAttribute("VALUE"), 64)
				if err == nil {
					sc.Lload[0] = tmp
				}
				tmp, err = strconv.ParseFloat(dataset["LLOAD1"].GetAttribute("VALUE"), 64)
				if err == nil {
					sc.Lload[1] = tmp
				}
				tmp, err = strconv.ParseFloat(dataset["LLOAD2"].GetAttribute("VALUE"), 64)
				if err == nil {
					sc.Lload[2] = tmp
				}
			}
		} else {
			if len(sc.Figs) > ind {
				var num int64
				var tmp float64
				var err error
				num, err = strconv.ParseInt(dataset["PROP"].GetAttribute("VALUE"), 10, 64)
				if err == nil {
					if p, ok := stw.frame.Props[int(num)]; ok {
						sc.Figs[ind].Prop = p
					}
				}
				tmp, err = strconv.ParseFloat(dataset["AREA"].GetAttribute("VALUE"), 64)
				if err == nil {
					sc.Figs[ind].Value["AREA"] = tmp
				}
				tmp, err = strconv.ParseFloat(dataset["IXX"].GetAttribute("VALUE"), 64)
				if err == nil {
					sc.Figs[ind].Value["IXX"] = tmp
				}
				tmp, err = strconv.ParseFloat(dataset["IYY"].GetAttribute("VALUE"), 64)
				if err == nil {
					sc.Figs[ind].Value["IYY"] = tmp
				}
				tmp, err = strconv.ParseFloat(dataset["VEN"].GetAttribute("VALUE"), 64)
				if err == nil {
					sc.Figs[ind].Value["VEN"] = tmp
				}
			}
		}
		updatedata(sc, ind)
	})
	// dlg := iup.Dialog(iup.Vbox(drawarea,
	dlg := iup.Dialog(iup.Vbox(dataset["NAME"],
		iup.Hbox(propertylabel("CODE"), dataset["CODE"]),
		iup.Hbox(proplist, addfig),
		iup.Hbox(propertylabel("PROP"), dataset["PROP"]),
		data,
		ctlg, bt))
	dlg.SetAttribute("TITLE", "Section")
	dlg.SetAttribute("DIALOGFRAME", "YES")
	dlg.SetAttribute("PARENTDIALOG", "mainwindow")
	dlg.Map()
	dataset["CODE"].SetAttribute("VALUE", fmt.Sprintf("%d", sc.Num))
	dataset["NAME"].SetAttribute("VALUE", fmt.Sprintf("%s", sc.Name))
	updateproplist(sc)
	updatedata(sc, 0)
	dlg.Show()
}

func (stw *Window) commandButton(name string, command *Command, size string) *iup.Handle {
	rtn := iup.Button()
	rtn.SetAttribute("TITLE", name)
	// img := iupim.LoadImage(filepath.Join(gopath,"src/github.com/yofu/st/dist.png"))
	// iup.SetHandle("image", img)
	// rtn.SetAttribute("IMAGE", "image")
	rtn.SetAttribute("RASTERSIZE", size)
	rtn.SetCallback(func(arg *iup.ButtonAction) {
		stw.execCommand(command)
		iup.SetFocus(stw.canv)
	})
	return rtn
}

func (stw *Window) CommandDialogConfBond() {
	coms1 := iup.Vbox()
	var l *iup.Handle
	l = datasectionlabel("CONF")
	l.SetAttribute("SIZE", "40x10")
	coms1.Append(l)
	for _, k := range []string{"CONFFREE", "CONFPIN", "CONFFIX", "CONFXYROLLER"} {
		com := Commands[k]
		coms1.Append(stw.commandButton(com.Display, com, "80x30"))
	}
	coms2 := iup.Vbox()
	l = datasectionlabel("BOND")
	l.SetAttribute("SIZE", "40x10")
	coms2.Append(l)
	for _, k := range []string{"BONDPIN", "BONDRIGID", "TOGGLEBOND", "COPYBOND"} {
		com := Commands[k]
		coms2.Append(stw.commandButton(com.Display, com, "80x30"))
	}
	dlg := iup.Dialog(iup.Hbox(coms1, coms2))
	dlg.SetAttribute("TITLE", "Conf/Bond")
	dlg.SetAttribute("DIALOGFRAME", "YES")
	dlg.SetAttribute("BGCOLOR", labelBGColor)
	dlg.SetAttribute("PARENTDIALOG", "mainwindow")
	dlg.Map()
	dlg.Show()
}

func (stw *Window) CommandDialogUtility() {
	coms1 := iup.Vbox()
	var l *iup.Handle
	l = datasectionlabel("UTILITY")
	l.SetAttribute("SIZE", "40x10")
	coms1.Append(l)
	for _, k := range []string{"DISTS", "MATCHPROP", "NODEDUPLICATION", "NODENOREFERENCE", "ELEMSAMENODE"} {
		com := Commands[k]
		coms1.Append(stw.commandButton(com.Display, com, "120x30"))
	}
	dlg := iup.Dialog(iup.Hbox(coms1))
	dlg.SetAttribute("TITLE", "Utility")
	dlg.SetAttribute("DIALOGFRAME", "YES")
	dlg.SetAttribute("BGCOLOR", labelBGColor)
	dlg.SetAttribute("PARENTDIALOG", "mainwindow")
	dlg.Map()
	dlg.Show()
}

func (stw *Window) CommandDialogNode() {
	coms1 := iup.Vbox()
	var l *iup.Handle
	l = datasectionlabel("NODE")
	l.SetAttribute("SIZE", "40x10")
	coms1.Append(l)
	for _, k := range []string{"MOVENODE", "MERGENODE"} {
		com := Commands[k]
		coms1.Append(stw.commandButton(com.Display, com, "100x30"))
	}
	dlg := iup.Dialog(iup.Hbox(coms1))
	dlg.SetAttribute("TITLE", "Node")
	dlg.SetAttribute("DIALOGFRAME", "YES")
	dlg.SetAttribute("BGCOLOR", labelBGColor)
	dlg.SetAttribute("PARENTDIALOG", "mainwindow")
	dlg.Map()
	dlg.Show()
}

func (stw *Window) CommandDialogElem() {
	coms1 := iup.Vbox()
	var l *iup.Handle
	l = datasectionlabel("ELEM")
	l.SetAttribute("SIZE", "40x10")
	coms1.Append(l)
	for _, k := range []string{"COPYELEM", "MOVEELEM", "JOINLINEELEM", "JOINPLATEELEM"} {
		com := Commands[k]
		coms1.Append(stw.commandButton(com.Display, com, "100x30"))
	}
	l = datasectionlabel("ADD")
	l.SetAttribute("SIZE", "40x10")
	coms1.Append(l)
	for _, k := range []string{"ADDLINEELEM", "ADDPLATEELEM", "ADDPLATEELEMBYLINE", "HATCHPLATEELEM"} {
		com := Commands[k]
		coms1.Append(stw.commandButton(com.Display, com, "100x30"))
	}
	dlg := iup.Dialog(iup.Hbox(coms1))
	dlg.SetAttribute("TITLE", "Elem")
	dlg.SetAttribute("DIALOGFRAME", "YES")
	dlg.SetAttribute("BGCOLOR", labelBGColor)
	dlg.SetAttribute("PARENTDIALOG", "mainwindow")
	dlg.Map()
	dlg.Show()
}

func datasectionlabel(val string) *iup.Handle {
	return iup.Text(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
		fmt.Sprintf("FGCOLOR=\"%s\"", sectionLabelColor),
		fmt.Sprintf("BGCOLOR=\"%s\"", labelBGColor),
		fmt.Sprintf("VALUE=\"%s\"", val),
		"CANFOCUS=NO",
		"READONLY=YES",
		"BORDER=NO",
		fmt.Sprintf("SIZE=%dx%d", datalabelwidth+datatextwidth, dataheight),
	)
}

func propertylabel(val string) *iup.Handle {
	return iup.Text(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
		fmt.Sprintf("FGCOLOR=\"%s\"", sectionLabelColor),
		fmt.Sprintf("BGCOLOR=\"%s\"", labelBGColor),
		fmt.Sprintf("VALUE=\"%s\"", val),
		"CANFOCUS=NO",
		"READONLY=YES",
		"BORDER=NO",
		fmt.Sprintf("SIZE=%dx%d", datalabelwidth, dataheight),
	)
}

func datalabel(val string) *iup.Handle {
	return iup.Text(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
		fmt.Sprintf("FGCOLOR=\"%s\"", labelFGColor),
		fmt.Sprintf("BGCOLOR=\"%s\"", labelBGColor),
		fmt.Sprintf("VALUE=\"  %s\"", val),
		"CANFOCUS=NO",
		"READONLY=YES",
		"BORDER=NO",
		fmt.Sprintf("SIZE=%dx%d", datalabelwidth, dataheight),
	)
}

func datatext(defval string) *iup.Handle {
	rtn := iup.Text(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
		fmt.Sprintf("FGCOLOR=\"%s\"", labelFGColor),
		fmt.Sprintf("BGCOLOR=\"%s\"", commandBGColor),
		fmt.Sprintf("VALUE=\"%s\"", defval),
		"BORDER=NO",
		"ALIGNMENT=ARIGHT",
		fmt.Sprintf("SIZE=%dx%d", datatextwidth, dataheight))
	rtn.SetCallback(func(arg *iup.CommonGetFocus) {
		rtn.SetAttribute("SELECTION", "1:100")
	})
	return rtn
}

func (stw *Window) HideAllSection() {
	for i, _ := range stw.frame.Show.Sect {
		if lb, ok := stw.Labels[fmt.Sprintf("%d", i)]; ok {
			lb.SetAttribute("FGCOLOR", labelOFFColor)
		}
		stw.frame.Show.Sect[i] = false
	}
}

func (stw *Window) ShowAllSection() {
	for i, _ := range stw.frame.Show.Sect {
		if lb, ok := stw.Labels[fmt.Sprintf("%d", i)]; ok {
			lb.SetAttribute("FGCOLOR", labelFGColor)
		}
		stw.frame.Show.Sect[i] = true
	}
}

func (stw *Window) HideSection(snum int) {
	if lb, ok := stw.Labels[fmt.Sprintf("%d", snum)]; ok {
		lb.SetAttribute("FGCOLOR", labelOFFColor)
	}
	stw.frame.Show.Sect[snum] = false
}

func (stw *Window) ShowSection(snum int) {
	if lb, ok := stw.Labels[fmt.Sprintf("%d", snum)]; ok {
		lb.SetAttribute("FGCOLOR", labelFGColor)
	}
	stw.frame.Show.Sect[snum] = true
}

func (stw *Window) HideEtype(etype int) {
	if etype == 0 {
		return
	}
	stw.frame.Show.Etype[etype] = false
	if lbl, ok := stw.Labels[st.ETYPES[etype]]; ok {
		lbl.SetAttribute("FGCOLOR", labelOFFColor)
	}
}

func (stw *Window) ShowEtype(etype int) {
	if etype == 0 {
		return
	}
	stw.frame.Show.Etype[etype] = true
	if lbl, ok := stw.Labels[st.ETYPES[etype]]; ok {
		lbl.SetAttribute("FGCOLOR", labelFGColor)
	}
}

func (stw *Window) ToggleEtype(etype int) {
	if stw.frame.Show.Etype[etype] {
		stw.HideEtype(etype)
	} else {
		stw.ShowEtype(etype)
	}
}

func (stw *Window) etypeLabel(name string, width int, etype int, defval bool) *iup.Handle {
	var col string
	if defval {
		col = labelFGColor
	} else {
		col = labelOFFColor
	}
	rtn := iup.Text(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
		fmt.Sprintf("FGCOLOR=\"%s\"", col),
		fmt.Sprintf("BGCOLOR=\"%s\"", labelBGColor),
		fmt.Sprintf("VALUE=\"%s\"", name),
		"CANFOCUS=NO",
		"READONLY=YES",
		"BORDER=NO",
		fmt.Sprintf("SIZE=%dx%d", width, dataheight))
	rtn.SetCallback(func(arg *iup.MouseButton) {
		if stw.frame != nil {
			switch arg.Button {
			case BUTTON_LEFT:
				if arg.Pressed == 0 { // Released
					stw.ToggleEtype(etype)
					stw.Redraw()
					iup.SetFocus(stw.canv)
				}
			}
		}
	})
	return rtn
}

func (stw *Window) switchLabel(etype int) *iup.Handle {
	var col string
	if stw.frame != nil && stw.frame.Show.Etype[etype] { // TODO: when stw.frame is created, set value
		col = labelFGColor
	} else {
		col = labelOFFColor
	}
	rtn := iup.Text(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
		fmt.Sprintf("FGCOLOR=\"%s\"", col),
		fmt.Sprintf("BGCOLOR=\"%s\"", labelBGColor),
		"VALUE=\"<>\"",
		"CANFOCUS=NO",
		"READONLY=YES",
		"BORDER=NO",
		fmt.Sprintf("SIZE=%dx%d", 10, dataheight))
	rtn.SetCallback(func(arg *iup.MouseButton) {
		if stw.frame != nil {
			switch arg.Button {
			case BUTTON_LEFT:
				if arg.Pressed == 0 { // Released
					if stw.frame.Show.Etype[etype] {
						if !stw.frame.Show.Etype[etype-2] {
							stw.HideEtype(etype)
							stw.ShowEtype(etype - 2)
						} else {
							stw.HideEtype(etype - 2)
						}
					} else {
						if stw.frame.Show.Etype[etype-2] {
							stw.ShowEtype(etype)
							stw.HideEtype(etype - 2)
						} else {
							stw.ShowEtype(etype)
						}
					}
					stw.Redraw()
					iup.SetFocus(stw.canv)
				}
			case BUTTON_CENTER:
				if arg.Pressed == 0 {
					stw.HideEtype(etype)
					stw.HideEtype(etype - 2)
					stw.Redraw()
					iup.SetFocus(stw.canv)
				}
			}
		}
	})
	return rtn
}

func (stw *Window) stressLabel(etype int, index uint) *iup.Handle {
	rtn := iup.Text(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
		fmt.Sprintf("FGCOLOR=\"%s\"", labelOFFColor),
		fmt.Sprintf("BGCOLOR=\"%s\"", labelBGColor),
		fmt.Sprintf("VALUE=\"%s\"", st.StressName[index]),
		"CANFOCUS=NO",
		"READONLY=YES",
		"BORDER=NO",
		fmt.Sprintf("SIZE=%dx%d", (datalabelwidth+datatextwidth)/6, dataheight))
	rtn.SetCallback(func(arg *iup.MouseButton) {
		if stw.frame != nil {
			switch arg.Button {
			case BUTTON_LEFT:
				if arg.Pressed == 0 { // Released
					if stw.frame.Show.Stress[etype]&(1<<index) != 0 {
						stw.frame.Show.Stress[etype] &= ^(1 << index)
						rtn.SetAttribute("FGCOLOR", labelOFFColor)
					} else {
						stw.frame.Show.Stress[etype] |= (1 << index)
						rtn.SetAttribute("FGCOLOR", labelFGColor)
					}
					stw.Redraw()
					iup.SetFocus(stw.canv)
				}
			}
		}
	})
	return rtn
}

func (stw *Window) displayLabel(name string, defval bool) *iup.Handle {
	var col string
	if defval {
		col = labelFGColor
	} else {
		col = labelOFFColor
	}
	rtn := iup.Text(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
		fmt.Sprintf("FGCOLOR=\"%s\"", col),
		fmt.Sprintf("BGCOLOR=\"%s\"", labelBGColor),
		fmt.Sprintf("VALUE=\"  %s\"", name),
		"CANFOCUS=NO",
		"READONLY=YES",
		"BORDER=NO",
		fmt.Sprintf("SIZE=%dx%d", datalabelwidth+datatextwidth, dataheight))
	rtn.SetCallback(func(arg *iup.MouseButton) {
		if stw.frame != nil {
			switch arg.Button {
			case BUTTON_LEFT:
				if arg.Pressed == 0 { // Released
					if rtn.GetAttribute("FGCOLOR") == labelFGColor {
						rtn.SetAttribute("FGCOLOR", labelOFFColor)
					} else {
						rtn.SetAttribute("FGCOLOR", labelFGColor)
					}
					switch name {
					case "GAXIS":
						stw.frame.Show.GlobalAxis = !stw.frame.Show.GlobalAxis
					case "EAXIS":
						stw.frame.Show.ElementAxis = !stw.frame.Show.ElementAxis
					case "BOND":
						stw.frame.Show.Bond = !stw.frame.Show.Bond
					case "CONF":
						stw.frame.Show.Conf = !stw.frame.Show.Conf
					case "PHINGE":
						stw.frame.Show.Phinge = !stw.frame.Show.Phinge
					case "KIJUN":
						stw.frame.Show.Kijun = !stw.frame.Show.Kijun
					case "DEFORMATION":
						stw.frame.Show.Deformation = !stw.frame.Show.Deformation
					case "YIELD":
						stw.frame.Show.YieldFunction = !stw.frame.Show.YieldFunction
					case "RATE":
						if stw.frame.Show.SrcanRate != 0 {
							stw.SrcanRateOff()
						} else {
							stw.SrcanRateOn()
						}
					}
					stw.Redraw()
					iup.SetFocus(stw.canv)
				}
			}
		}
	})
	return rtn
}

func (stw *Window) SetPeriod(per string) {
	stw.Labels["PERIOD"].SetAttribute("VALUE", per)
	stw.frame.Show.Period = per
}

func (stw *Window) IncrementPeriod(num int) {
	pat := regexp.MustCompile("([a-zA-Z]+)(@[0-9]+)")
	fs := pat.FindStringSubmatch(stw.frame.Show.Period)
	if len(fs) < 3 {
		return
	}
	if nl, ok := stw.frame.Nlap[strings.ToUpper(fs[1])]; ok {
		tmp, _ := strconv.ParseInt(fs[2][1:], 10, 64)
		val := int(tmp) + num
		if val < 1 || val > nl {
			return
		}
		per := strings.Replace(stw.frame.Show.Period, fs[2], fmt.Sprintf("@%d", val), -1)
		stw.Labels["PERIOD"].SetAttribute("VALUE", per)
		stw.frame.Show.Period = per
	}
}

func (stw *Window) NodeCaptionOn(name string) {
	for i, j := range st.NODECAPTIONS {
		if j == name {
			if lbl, ok := stw.Labels[name]; ok {
				lbl.SetAttribute("FGCOLOR", labelFGColor)
			}
			if stw.frame != nil {
				stw.frame.Show.NodeCaptionOn(1 << uint(i))
			}
		}
	}
}

func (stw *Window) NodeCaptionOff(name string) {
	for i, j := range st.NODECAPTIONS {
		if j == name {
			if lbl, ok := stw.Labels[name]; ok {
				lbl.SetAttribute("FGCOLOR", labelOFFColor)
			}
			if stw.frame != nil {
				stw.frame.Show.NodeCaptionOff(1 << uint(i))
			}
		}
	}
}

func (stw *Window) ElemCaptionOn(name string) {
	for i, j := range st.ELEMCAPTIONS {
		if j == name {
			if lbl, ok := stw.Labels[name]; ok {
				lbl.SetAttribute("FGCOLOR", labelFGColor)
			}
			if stw.frame != nil {
				stw.frame.Show.ElemCaptionOn(1 << uint(i))
			}
		}
	}
}

func (stw *Window) ElemCaptionOff(name string) {
	for i, j := range st.ELEMCAPTIONS {
		if j == name {
			if lbl, ok := stw.Labels[name]; ok {
				lbl.SetAttribute("FGCOLOR", labelOFFColor)
			}
			if stw.frame != nil {
				stw.frame.Show.ElemCaptionOff(1 << uint(i))
			}
		}
	}
}

func (stw *Window) SrcanRateOn(names ...string) {
	defer func() {
		if stw.frame.Show.SrcanRate != 0 {
			stw.Labels["SRCAN_RATE"].SetAttribute("FGCOLOR", labelFGColor)
		}
	}()
	if len(names) == 0 {
		for i, j := range st.SRCANS {
			if lbl, ok := stw.Labels[j]; ok {
				lbl.SetAttribute("FGCOLOR", labelFGColor)
			}
			if stw.frame != nil {
				stw.frame.Show.SrcanRateOn(1 << uint(i))
			}
		}
		return
	}
	for _, name := range names {
		for i, j := range st.SRCANS {
			if j == name {
				if lbl, ok := stw.Labels[name]; ok {
					lbl.SetAttribute("FGCOLOR", labelFGColor)
				}
				if stw.frame != nil {
					stw.frame.Show.SrcanRateOn(1 << uint(i))
				}
			}
		}
	}
}

func (stw *Window) SrcanRateOff(names ...string) {
	defer func() {
		if stw.frame.Show.SrcanRate == 0 {
			stw.Labels["SRCAN_RATE"].SetAttribute("FGCOLOR", labelOFFColor)
		}
	}()
	if len(names) == 0 {
		for i, j := range st.SRCANS {
			if lbl, ok := stw.Labels[j]; ok {
				lbl.SetAttribute("FGCOLOR", labelOFFColor)
			}
			if stw.frame != nil {
				stw.frame.Show.SrcanRateOff(1 << uint(i))
			}
		}
		return
	}
	for _, name := range names {
		for i, j := range st.SRCANS {
			if j == name {
				if lbl, ok := stw.Labels[name]; ok {
					lbl.SetAttribute("FGCOLOR", labelOFFColor)
				}
				if stw.frame != nil {
					stw.frame.Show.SrcanRateOff(1 << uint(i))
				}
			}
		}
	}
}

func (stw *Window) StressOn(etype int, index uint) {
	stw.frame.Show.Stress[etype] |= (1 << index)
	if etype <= st.SLAB {
		if lbl, ok := stw.Labels[fmt.Sprintf("%s_%s", st.ETYPES[etype], strings.ToUpper(st.StressName[index]))]; ok {
			lbl.SetAttribute("FGCOLOR", labelFGColor)
		}
	}
}

func (stw *Window) StressOff(etype int, index uint) {
	stw.frame.Show.Stress[etype] &= ^(1 << index)
	if etype <= st.SLAB {
		if lbl, ok := stw.Labels[fmt.Sprintf("%s_%s", st.ETYPES[etype], strings.ToUpper(st.StressName[index]))]; ok {
			lbl.SetAttribute("FGCOLOR", labelOFFColor)
		}
	}
}

func (stw *Window) DeformationOn() {
	stw.frame.Show.Deformation = true
	stw.Labels["DEFORMATION"].SetAttribute("FGCOLOR", labelFGColor)
}

func (stw *Window) DeformationOff() {
	stw.frame.Show.Deformation = false
	stw.Labels["DEFORMATION"].SetAttribute("FGCOLOR", labelOFFColor)
}

func (stw *Window) DispOn(direction int) {
	name := fmt.Sprintf("NC_%s", st.DispName[direction])
	for i, str := range st.NODECAPTIONS {
		if name == str {
			stw.frame.Show.NodeCaption |= (1 << uint(i))
			stw.Labels[name].SetAttribute("FGCOLOR", labelFGColor)
			return
		}
	}
}

func (stw *Window) DispOff(direction int) {
	name := fmt.Sprintf("NC_%s", st.DispName[direction])
	for i, str := range st.NODECAPTIONS {
		if name == str {
			stw.frame.Show.NodeCaption &= ^(1 << uint(i))
			stw.Labels[name].SetAttribute("FGCOLOR", labelOFFColor)
			return
		}
	}
}

func (stw *Window) captionLabel(ne string, name string, width int, val uint, on bool) *iup.Handle {
	var col string
	if on {
		col = labelFGColor
	} else {
		col = labelOFFColor
	}
	if stw.frame != nil { // TODO: when stw.frame is created, set value
		if on {
			switch ne {
			case "NODE":
				stw.frame.Show.NodeCaption |= val
			case "ELEM":
				stw.frame.Show.ElemCaption |= val
			}
		}
	}
	rtn := iup.Text(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
		fmt.Sprintf("FGCOLOR=\"%s\"", col),
		fmt.Sprintf("BGCOLOR=\"%s\"", labelBGColor),
		fmt.Sprintf("VALUE=\"%s\"", name),
		"CANFOCUS=NO",
		"READONLY=YES",
		"BORDER=NO",
		fmt.Sprintf("SIZE=%dx%d", width, dataheight))
	rtn.SetCallback(func(arg *iup.MouseButton) {
		if stw.frame != nil {
			switch arg.Button {
			case BUTTON_LEFT:
				if arg.Pressed == 0 { // Released
					switch ne {
					case "NODE":
						if stw.frame.Show.NodeCaption&val != 0 {
							rtn.SetAttribute("FGCOLOR", labelOFFColor)
							stw.frame.Show.NodeCaption &= ^val
						} else {
							rtn.SetAttribute("FGCOLOR", labelFGColor)
							stw.frame.Show.NodeCaption |= val
						}
					case "ELEM":
						if stw.frame.Show.ElemCaption&val != 0 {
							rtn.SetAttribute("FGCOLOR", labelOFFColor)
							stw.frame.Show.ElemCaption &= ^val
						} else {
							rtn.SetAttribute("FGCOLOR", labelFGColor)
							stw.frame.Show.ElemCaption |= val
						}
					}
					stw.Redraw()
					iup.SetFocus(stw.canv)
				}
			}
		}
	})
	return rtn
}

func (stw *Window) srcanLabel(name string, width int, val uint, on bool) *iup.Handle {
	var col string
	if on {
		col = labelFGColor
	} else {
		col = labelOFFColor
	}
	rtn := iup.Text(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
		fmt.Sprintf("FGCOLOR=\"%s\"", col),
		fmt.Sprintf("BGCOLOR=\"%s\"", labelBGColor),
		fmt.Sprintf("VALUE=\"%s\"", name),
		"CANFOCUS=NO",
		"READONLY=YES",
		"BORDER=NO",
		fmt.Sprintf("SIZE=%dx%d", width, dataheight))
	rtn.SetCallback(func(arg *iup.MouseButton) {
		if stw.frame != nil {
			switch arg.Button {
			case BUTTON_LEFT:
				if arg.Pressed == 0 { // Released
					if stw.frame.Show.SrcanRate&val != 0 {
						stw.SrcanRateOff(fmt.Sprintf("SRCAN_%s", strings.TrimLeft(name, " ")))
					} else {
						stw.SrcanRateOn(fmt.Sprintf("SRCAN_%s", strings.TrimLeft(name, " ")))
					}
					stw.Redraw()
					iup.SetFocus(stw.canv)
				}
			}
		}
	})
	if stw.frame != nil { // TODO: when stw.frame is created, set value
		if on {
			stw.SrcanRateOn(st.SRCANS[val])
		}
	}
	return rtn
}

func (stw *Window) toggleLabel(def uint, values []string) *iup.Handle {
	col := labelFGColor
	l := len(values)
	rtn := iup.Text(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
		fmt.Sprintf("FGCOLOR=\"%s\"", col),
		fmt.Sprintf("BGCOLOR=\"%s\"", labelBGColor),
		fmt.Sprintf("VALUE=\"  %s\"", values[def]),
		"CANFOCUS=NO",
		"READONLY=YES",
		"BORDER=NO",
		fmt.Sprintf("SIZE=%dx%d", datalabelwidth+datatextwidth, dataheight))
	rtn.SetCallback(func(arg *iup.MouseButton) {
		if stw.frame != nil {
			switch arg.Button {
			case BUTTON_LEFT:
				if arg.Pressed == 0 { // Released
					now := 0
					tmp := rtn.GetAttribute("VALUE")[2:]
					for i := 0; i < l; i++ {
						if tmp == values[i] {
							now = i
							break
						}
					}
					next := now + 1
					if next >= l {
						next = 0
					}
					if next == st.ECOLOR_BLACK {
						next++
					}
					rtn.SetAttribute("VALUE", fmt.Sprintf("  %s", values[next]))
					stw.frame.Show.ColorMode = uint(next)
				}
			case BUTTON_CENTER:
				if arg.Pressed == 0 { // Released
					now := 0
					tmp := rtn.GetAttribute("VALUE")[2:]
					for i := 0; i < l; i++ {
						if tmp == values[i] {
							now = i
							break
						}
					}
					next := now - 1
					if next < 0 {
						next = l - 1
					}
					if next == st.ECOLOR_BLACK {
						next--
					}
					rtn.SetAttribute("VALUE", fmt.Sprintf("  %s", values[next]))
					stw.frame.Show.ColorMode = uint(next)
				}
			}
			stw.Redraw()
			iup.SetFocus(stw.canv)
		}
	})
	return rtn
}

func (stw *Window) CB_TextValue(h *iup.Handle, valptr *float64) {
	h.SetCallback(func(arg *iup.CommonKillFocus) {
		if stw.frame != nil {
			val, err := strconv.ParseFloat(h.GetAttribute("VALUE"), 64)
			if err != nil {
				h.SetAttribute("VALUE", fmt.Sprintf("%.3f", *valptr))
			} else {
				*valptr = val
			}
			stw.Redraw()
		}
	})
	h.SetCallback(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		switch key.Key() {
		case KEY_ESCAPE:
			stw.FocusCanv()
		case KEY_ENTER:
			if stw.frame != nil {
				val, err := strconv.ParseFloat(h.GetAttribute("VALUE"), 64)
				if err != nil {
					h.SetAttribute("VALUE", fmt.Sprintf("%.3f", *valptr))
				} else {
					*valptr = val
				}
				stw.Redraw()
				iup.SetFocus(h)
			}
		}
	})
}

func (stw *Window) CB_RangeValue(h *iup.Handle, valptr *float64) {
	h.SetCallback(func(arg *iup.CommonKillFocus) {
		if stw.frame != nil {
			val, err := strconv.ParseFloat(h.GetAttribute("VALUE"), 64)
			if err == nil {
				*valptr = val
			}
			stw.Redraw()
		}
	})
	h.SetCallback(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		switch key.Key() {
		case KEY_ENTER:
			if stw.frame != nil {
				val, err := strconv.ParseFloat(h.GetAttribute("VALUE"), 64)
				if err == nil {
					*valptr = val
				}
				stw.Redraw()
				iup.SetFocus(h)
			}
		}
	})
}

func (stw *Window) SetColorMode(mode uint) {
	stw.Labels["COLORMODE"].SetAttribute("VALUE", fmt.Sprintf("  %s", st.ECOLORS[mode]))
	stw.frame.Show.ColorMode = mode
}

func (stw *Window) CB_Period(h *iup.Handle, valptr *string) {
	h.SetCallback(func(arg *iup.CommonKillFocus) {
		if stw.frame != nil {
			per := strings.ToUpper(h.GetAttribute("VALUE"))
			*valptr = per
			h.SetAttribute("VALUE", per)
			stw.Redraw()
		}
	})
	h.SetCallback(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		switch key.Key() {
		case KEY_ENTER:
			if stw.frame != nil {
				per := strings.ToUpper(h.GetAttribute("VALUE"))
				*valptr = per
				h.SetAttribute("VALUE", per)
				stw.Redraw()
				iup.SetFocus(h)
			}
		}
	})
}

// DataLabel
func (stw *Window) LinkTextValue() {
	stw.CB_TextValue(stw.Labels["GFACT"], &stw.frame.View.Gfact)
	stw.CB_TextValue(stw.Labels["DISTR"], &stw.frame.View.Dists[0])
	stw.CB_TextValue(stw.Labels["DISTL"], &stw.frame.View.Dists[1])
	stw.CB_TextValue(stw.Labels["PHI"], &stw.frame.View.Angle[0])
	stw.CB_TextValue(stw.Labels["THETA"], &stw.frame.View.Angle[1])
	stw.CB_TextValue(stw.Labels["FOCUSX"], &stw.frame.View.Focus[0])
	stw.CB_TextValue(stw.Labels["FOCUSY"], &stw.frame.View.Focus[1])
	stw.CB_TextValue(stw.Labels["FOCUSZ"], &stw.frame.View.Focus[2])
	stw.CB_TextValue(stw.Labels["CENTERX"], &stw.frame.View.Center[0])
	stw.CB_TextValue(stw.Labels["CENTERY"], &stw.frame.View.Center[1])
	stw.CB_RangeValue(stw.Labels["XMAX"], &stw.frame.Show.Xrange[1])
	stw.CB_RangeValue(stw.Labels["XMIN"], &stw.frame.Show.Xrange[0])
	stw.CB_RangeValue(stw.Labels["YMAX"], &stw.frame.Show.Yrange[1])
	stw.CB_RangeValue(stw.Labels["YMIN"], &stw.frame.Show.Yrange[0])
	stw.CB_RangeValue(stw.Labels["ZMAX"], &stw.frame.Show.Zrange[1])
	stw.CB_RangeValue(stw.Labels["ZMIN"], &stw.frame.Show.Zrange[0])
	stw.CB_Period(stw.Labels["PERIOD"], &stw.frame.Show.Period)
	stw.CB_TextValue(stw.Labels["GAXISSIZE"], &stw.frame.Show.GlobalAxisSize)
	stw.CB_TextValue(stw.Labels["EAXISSIZE"], &stw.frame.Show.ElementAxisSize)
	stw.CB_TextValue(stw.Labels["BONDSIZE"], &stw.frame.Show.BondSize)
	stw.CB_TextValue(stw.Labels["CONFSIZE"], &stw.frame.Show.ConfSize)
	stw.CB_TextValue(stw.Labels["DFACT"], &stw.frame.Show.Dfact)
	stw.CB_TextValue(stw.Labels["QFACT"], &stw.frame.Show.Qfact)
	stw.CB_TextValue(stw.Labels["MFACT"], &stw.frame.Show.Mfact)
	stw.CB_TextValue(stw.Labels["RFACT"], &stw.frame.Show.Rfact)
}

func (stw *Window) EscapeCB() {
	stw.cline.SetAttribute("VALUE", "")
	stw.cname.SetAttribute("VALUE", "SELECT")
	stw.canv.SetAttribute("CURSOR", "ARROW")
	stw.cdcanv.Foreground(cd.CD_WHITE)
	stw.cdcanv.WriteMode(cd.CD_REPLACE)
	stw.CMenu()
	stw.CB_MouseButton()
	stw.CB_MouseMotion()
	stw.CB_CanvasWheel()
	stw.CB_CommonKeyAny()
	if stw.frame != nil {
		stw.Redraw()
	}
	comhistpos = -1
	clineinput = ""
}

func (stw *Window) EscapeAll() {
	stw.Deselect()
	stw.EscapeCB()
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
		stw.ExecCommand(txt)
	}
	if err := s.Err(); err != nil {
		return err
	}
	return nil
}

// }}}

func EditPgp() {
	cmd := exec.Command("cmd", "/C", "start", pgpfile)
	cmd.Start()
}

func (stw *Window) Analysis(fn string, arg string) error {
	var err error
	var cmd *exec.Cmd
	if arg == "" {
		cmd = exec.Command(analysiscommand, fn)
	} else {
		cmd = exec.Command(analysiscommand, arg, fn)
	}
	err = cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	return err
}

func (stw *Window) SetDefaultPgp() {
	aliases = make(map[string]*Command, 0)
	aliases["D"] = DISTS
	aliases["M"] = MEASURE
	aliases["TB"] = TOGGLEBOND
	aliases["O"] = OPEN
	aliases["W"] = WEIGHTCOPY
	aliases["IN"] = INSERT
	aliases["SF"] = SETFOCUS
	aliases["NODE"] = SELECTNODE
	aliases["CB"] = SELECTCOLUMNBASE
	aliases["CD"] = SELECTCONFED
	aliases["ELEM"] = SELECTELEM
	aliases["SEC"] = SELECTSECT
	aliases["SEC-"] = HIDESECTION
	aliases["CW"] = HIDECURTAINWALL
	aliases["CH"] = SELECTCHILDREN
	aliases["ER"] = ERRORELEM
	aliases["F"] = FENCE
	aliases["PL"] = SHOWPLANE
	aliases["A"] = ADDLINEELEM
	aliases["AA"] = ADDPLATEELEM
	aliases["AAA"] = ADDPLATEELEMBYLINE
	aliases["AS"] = ADDPLATEALL
	aliases["WR"] = EDITWRECT
	aliases["H"] = HATCHPLATEELEM
	aliases["CV"] = CONVEXHULL
	aliases["Q"] = MATCHPROP
	aliases["QQ"] = COPYBOND
	aliases["AC"] = AXISTOCANG
	aliases["BP"] = BONDPIN
	aliases["BR"] = BONDRIGID
	aliases["CP"] = CONFPIN
	aliases["CZ"] = CONFXYROLLER
	aliases["CF"] = CONFFREE
	aliases["CX"] = CONFFIX
	aliases["C"] = COPYELEM
	aliases["DC"] = DUPLICATEELEM
	aliases["DD"] = MOVENODE
	aliases["ML"] = MOVETOLINE
	aliases["DDD"] = MOVEELEM
	aliases["PN"] = PINCHNODE
	aliases["R"] = ROTATE
	aliases["SC"] = SCALE
	aliases["MI"] = MIRROR
	aliases["SE"] = SEARCHELEM
	aliases["NE"] = NODETOELEM
	aliases["EN"] = ELEMTONODE
	aliases["CO"] = CONNECTED
	aliases["ON"] = ONNODE
	aliases["NN"] = NODENOREFERENCE
	aliases["ES"] = ELEMSAMENODE
	aliases["SUS"] = SUSPICIOUS
	aliases["PE"] = PRUNEENOD
	aliases["ND"] = NODEDUPLICATION
	aliases["ED"] = ELEMDUPLICATION
	aliases["NS"] = NODESORT
	aliases["CN"] = CATBYNODE
	aliases["CI"] = CATINTERMEDIATENODE
	aliases["J"] = JOINLINEELEM
	aliases["JP"] = JOINPLATEELEM
	aliases["EA"] = EXTRACTARCLM
	aliases["DO"] = DIVIDEATONS
	aliases["DM"] = DIVIDEATMID
	aliases["DN"] = DIVIDEINN
	aliases["DE"] = DIVIDEATELEM
	aliases["I"] = INTERSECT
	aliases["IA"] = INTERSECTALL
	aliases["Z"] = TRIM
	aliases["ZZ"] = EXTEND
	aliases["MN"] = MERGENODE
	aliases["E"] = ERASE
	aliases["FAC"] = FACTS
	aliases["REA"] = REACTION
	aliases["SR"] = SUMREACTION
	aliases["UL"] = UPLIFT
	aliases["SS"] = NOTICE1459
	aliases["ZD"] = ZOUBUNDISP
	aliases["ZY"] = ZOUBUNYIELD
	aliases["ZR"] = ZOUBUNREACTION
	aliases["AMT"] = AMOUNTPROP
	aliases["EPS"] = SETEPS
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
					pos := strings.Index(command, "_")
					iup.SetFocus(stw.cline)
					stw.cline.SetAttribute("VALUE", strings.Replace(command, "_", "", -1))
					stw.cline.SetAttribute("CARETPOS", fmt.Sprintf("%d", pos))
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
					err := st.ExMode(stw, currentcommand)
					if err != nil {
						st.ErrorMessage(stw, err, st.ERROR)
					}
					val++
				}}
			} else {
				aliases[strings.ToUpper(words[0])] = &Command{"", "", "", func(stw *Window) {
					err := st.ExMode(stw, command)
					if err != nil {
						st.ErrorMessage(stw, err, st.ERROR)
					}
				}}
			}
		} else if strings.HasPrefix(words[1], "'") {
			command := strings.Join(words[1:], " ")
			if strings.Contains(command, "_") {
				aliases[strings.ToUpper(words[0])] = &Command{"", "", "", func(stw *Window) {
					pos := strings.Index(command, "_")
					iup.SetFocus(stw.cline)
					stw.cline.SetAttribute("VALUE", strings.Replace(command, "_", "", -1))
					stw.cline.SetAttribute("CARETPOS", fmt.Sprintf("%d", pos))
				}}
			} else {
				aliases[strings.ToUpper(words[0])] = &Command{"", "", "", func(stw *Window) {
					err := st.Fig2Mode(stw, command)
					if err != nil {
						st.ErrorMessage(stw, err, st.ERROR)
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

// Initialize
func init() {
	sectionaliases = make(map[int]string, 0)
	RangeView.Dists = RangeViewDists
	RangeView.Angle = RangeViewAngle
	RangeView.Center = RangeViewCenter
}

// Utility
func isShift(status string) bool  { return status[0] == 'S' }
func isCtrl(status string) bool   { return status[1] == 'C' }
func isLeft(status string) bool   { return status[2] == '1' }
func isCenter(status string) bool { return status[3] == '2' }
func isRight(status string) bool  { return status[4] == '3' }
func isDouble(status string) bool { return status[5] == 'D' }
func isAlt(status string) bool    { return status[6] == 'A' }
func statusKey(status string) (rtn int) {
	if status[2] == '1' {
		rtn += 1
	} // Left
	if status[3] == '2' {
		rtn += 2
	} // Center
	if status[4] == '3' {
		rtn += 4
	} // Right
	return
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

func (stw *Window) EPS() float64 {
	return EPS
}

func (stw *Window) SetEPS(val float64) {
	EPS = val
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
	checkframe(stw)
}

func (stw *Window) TextBox(name string) st.TextBox {
	if _, tok := stw.textBox[name]; !tok {
		stw.textBox[name] = NewTextBox()
	}
	return stw.textBox[name]
}

func (stw *Window) Changed(c bool) {
	stw.changed = c
}
func (stw *Window) IsChanged() bool {
	return stw.changed
}

func (stw *Window) LastExCommand() string {
	return stw.lastexcommand
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

func (stw *Window) History(str string) {
	stw.addHistory(str)
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
	setconf(stw, lis)
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
	if l, ok := stw.Labels[key]; ok {
		l.SetAttribute("VALUE", value)
	}
}

func (stw *Window) EnableLabel(key string) {
	if l, ok := stw.Labels[key]; ok {
		l.SetAttribute("FGCOLOR", labelFGColor)
	}
}

func (stw *Window) DisableLabel(key string) {
	if l, ok := stw.Labels[key]; ok {
		l.SetAttribute("FGCOLOR", labelOFFColor)
	}
}

func (stw *Window) GetCanvasSize() (int, int) {
	return stw.cdcanv.GetSize()
}

func (stw *Window) SectionAlias(key int) (string, bool) {
	str, ok := sectionaliases[key]
	return str, ok
}

func (stw *Window) AddSectionAlias(key int, value string) {
	sectionaliases[key] = value
}

func (stw *Window) DeleteSectionAlias(key int) {
	delete(sectionaliases, key)
}

func (stw *Window) ClearSectionAlias() {
	sectionaliases = make(map[int]string, 0)
}

func (stw *Window) CanvasDirection() int {
	return 0
}
