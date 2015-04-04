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
	"github.com/yofu/ps"
	"github.com/yofu/st/stlib"
	"github.com/yofu/st/stsvg"
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
	undopos            int
	completepos        int
	completes          []string
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
	tooldir         = filepath.Join(home, ".st/tool")
	pgpfile         = filepath.Join(home, ".st/st.pgp")
	recentfn        = filepath.Join(home, ".st/recent.dat")
	historyfn       = filepath.Join(home, ".st/history.dat")
	analysiscommand = "C:/an/an.exe"
	NOUNDO          = false
	ALTSELECTNODE   = true
)

const LOGFILE = "_st.log"

var (
	LOGLEVEL = []string{"DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"}
)

const (
	DEBUG = iota
	INFO
	WARNING
	ERROR
	CRITICAL
)

const (
	windowSize   = "FULLxFULL"
	nRecentFiles = 3
	nUndo        = 10
)

// Font
var (
	commandFontFace = "IPA明朝"
	commandFontSize = "11"
	labelFGColor    = "0 0 0"
	// labelBGColor = "255 255 255"
	labelBGColor          = "212 208 200"
	commandFGColor        = "0 127 31"
	commandBGColor        = "191 191 191"
	historyBGColor        = "220 220 220"
	sectionLabelColor     = "204 51 0"
	labelOFFColor         = "160 160 160"
	canvasFontFace        = "IPA明朝"
	canvasFontSize        = 9
	sectiondlgFontSize    = "9"
	sectiondlgFGColor     = "255 255 255"
	sectiondlgBGColor     = "66 66 66"
	sectiondlgSelectColor = "255 255 0"
	canvasFontColor       = cd.CD_DARK_GRAY
	DefaultTextAlignment  = cd.CD_BASE_LEFT
	printFontColor        = cd.CD_BLACK
	printFontFace         = "IPA明朝"
	printFontSize         = 8
	showprintrange        = false
)

const (
	FONT_COMMAND = iota
	FONT_CANVAS
)

// Paper Size
const (
	A4_TATE = iota
	A4_YOKO
	A3_TATE
	A3_YOKO
)

// Draw
var (
	first                   = 1
	defaultPlateEdgeColor   = cd.CD_GRAY
	defaultBondColor        = cd.CD_GRAY
	defaultConfColor        = cd.CD_GRAY
	defaultMomentColor      = cd.CD_DARK_MAGENTA
	defaultKijunColor       = cd.CD_GRAY
	defaultMeasureColor     = cd.CD_GRAY
	defaultStressTextColor  = cd.CD_GRAY
	defaultYieldedTextColor = cd.CD_YELLOW
	defaultBrittleTextColor = cd.CD_RED
	PlateEdgeColor          = defaultPlateEdgeColor
	BondColor               = defaultBondColor
	ConfColor               = defaultConfColor
	MomentColor             = defaultMomentColor
	KijunColor              = defaultKijunColor
	MeasureColor            = defaultMeasureColor
	StressTextColor         = defaultStressTextColor
	YieldedTextColor        = defaultYieldedTextColor
	BrittleTextColor        = defaultBrittleTextColor
	fixRotate               = false
	fixMove                 = false
	deg10                   = 10.0 * math.Pi / 180.0
)

var (
	STLOGO = &TextBox{
		Value:    []string{"         software", "     forstructural", "   analysisthename", "  ofwhichstandsfor", "", " sigmatau  stress", "structure  steel", "andsometh  ing", " likethat"},
		Position: []float64{100.0, 100.0},
		Angle:    0.0,
		Font:     NewFont(),
		Hide:     false,
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

// Speed
const (
	CanvasRotateSpeedX = 0.01
	CanvasRotateSpeedY = 0.01
	CanvasMoveSpeedX   = 0.05
	CanvasMoveSpeedY   = 0.05
	CanvasScaleSpeed   = 15
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
	exabbrev = []string{
		"e/dit", "q/uit", "vi/m", "hk/you", "hw/eak", "rp/ipe", "cp/ipe", "tk/you", "ck/you", "pla/te", "fixr/otate", "fixm/ove", "noun/do", "un/do", "w/rite", "sav/e", "inc/rement", "c/heck", "r/ead",
		"ins/ert", "p/rop/s/ect", "w/rite/o/utput", "w/rite/rea/ction", "nmi/nteraction", "fi/g2", "fe/nce", "no/de", "xsc/ale", "ysc/ale", "zsc/ale", "pl/oad", "z/oubun/d/isp", "z/oubun/r/eaction",
		"fac/ts", "go/han/l/st", "el/em", "ave/rage", "bo/nd", "ax/is/2//c/ang", "resul/tant", "prest/ress", "therm/al", "div/ide", "e/lem/dup/lication", "i/ntersect/a/ll", "co/nf",
		"pi/le", "sec/tion", "an/alysis", "f/ilter", "h/eigh/t/", "h/eigh/t+/", "h/eigh/t-/", "sec/tion/+/", "col/or", "a/rclm/001/", "a/rclm/201/", "a/rclm/301/",
	}
	fig2abbrev = []string {
		"gf/act", "foc/us", "ang/le", "dist/s", "pers/pective", "ax/onometric", "df/act", "rf/act", "mf/act", "gax/is", "eax/is",
		"noax/is", "el/em", "el/em/+/", "el/em/-/", "sec/tion", "sec/tion/+/", "sec/tion/-/", "k/ijun", "mea/sure", "el/em/c/ode", "sec/t/c/ode",
		"wid/th", "h/eigh/t/", "sr/can/col/or", "sr/can/ra/te", "st/ress", "prest/ress", "stiff/", "def/ormation", "dis/p", "ecc/entric", "dr/aw",
		"al/ias", "anon/ymous", "no/de/c/ode", "wei/ght", "con/f", "pi/le", "fen/ce", "per/iod", "per/iod/++/", "per/iod/--/",
		"nocap/tion", "noleg/end", "nom/oment/v/alue", "ncol/or", "p/age/tit/le", "tit/le", "pos/ition",
	}
)

// }}}

type Window struct { // {{{
	Home string
	Cwd  string

	Frame                  *st.Frame
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

	sectiondlg *iup.Handle

	CanvasSize []float64 // width, height

	zb *iup.Handle

	lselect *iup.Handle

	SelectNode []*st.Node
	SelectElem []*st.Elem

	PageTitle *TextBox
	Title     *TextBox
	Text      *TextBox
	TextBox   map[string]*TextBox

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
	Changed     bool

	exmodech  chan (interface{})
	exmodeend chan (int)

	comhist     []string
	recentfiles []string
	undostack   []*st.Frame
}

// }}}

func NewWindow(homedir string) *Window { // {{{
	stw := new(Window)
	stw.Home = homedir
	stw.Cwd = homedir
	stw.SelectNode = make([]*st.Node, 0)
	stw.SelectElem = make([]*st.Elem, 0)
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
	stw.Labels["EC_RATE_L"] = stw.captionLabel("ELEM", "  RATE_L", datalabelwidth, st.EC_RATE_L, false)
	stw.Labels["EC_RATE_S"] = stw.captionLabel("ELEM", "RATE_S", datatextwidth, st.EC_RATE_S, false)
	stw.Labels["COLORMODE"] = stw.toggleLabel(2, st.ECOLORS)
	stw.Labels["PERIOD"] = datatext("L")
	stw.Labels["GAXISSIZE"] = datatext("1.0")
	stw.Labels["EAXISSIZE"] = datatext("0.5")
	stw.Labels["BONDSIZE"] = datatext("3.0")
	stw.Labels["CONFSIZE"] = datatext("9.0")
	stw.Labels["DFACT"] = datatext("100.0")
	stw.Labels["MFACT"] = datatext("0.5")
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
						stw.Reload()
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
						stw.ReadAll()
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
						if stw.Changed {
							if stw.Yn("CHANGED", "変更を保存しますか") {
								stw.SaveAS()
							} else {
								return
							}
						}
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
						if name, ok := iup.GetOpenFile(stw.Cwd, "*.pgp"); ok {
							al := make(map[string]*Command, 0)
							err := ReadPgp(name, al)
							if err != nil {
								stw.addHistory("ReadPgp: Cannot Read st.pgp")
							} else {
								aliases = al
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
			iup.Menu(iup.Item(iup.Attr("TITLE", "DISTS"), func(arg *iup.ItemAction) { stw.execAliasCommand("DISTS") }),
				iup.Item(iup.Attr("TITLE", "SET FOCUS"), func(arg *iup.ItemAction) { stw.execAliasCommand("SETFOCUS") }),
				iup.Separator(),
				iup.Item(iup.Attr("TITLE", "TOGGLEBOND"), func(arg *iup.ItemAction) { stw.execAliasCommand("TOGGLEBOND") }),
				iup.Item(iup.Attr("TITLE", "COPYBOND"), func(arg *iup.ItemAction) { stw.execAliasCommand("COPYBOND") }),
				iup.Separator(),
				iup.Item(iup.Attr("TITLE", "SELECT NODE"), func(arg *iup.ItemAction) { stw.execAliasCommand("SELECTNODE") }),
				iup.Item(iup.Attr("TITLE", "SELECT ELEM"), func(arg *iup.ItemAction) { stw.execAliasCommand("SELECTELEM") }),
				iup.Item(iup.Attr("TITLE", "SELECT SECT"), func(arg *iup.ItemAction) { stw.execAliasCommand("SELECTSECT") }),
				iup.Item(iup.Attr("TITLE", "FENCE"), func(arg *iup.ItemAction) { stw.execAliasCommand("FENCE") }),
				iup.Item(iup.Attr("TITLE", "ERROR ELEM"), func(arg *iup.ItemAction) { stw.execAliasCommand("ERRORELEM") }),
				iup.Separator(),
				iup.Item(iup.Attr("TITLE", "ADD LINE ELEM"), func(arg *iup.ItemAction) { stw.execAliasCommand("ADDLINEELEM") }),
				iup.Item(iup.Attr("TITLE", "ADD PLATE ELEM"), func(arg *iup.ItemAction) { stw.execAliasCommand("ADDPLATEELEM") }),
				iup.Item(iup.Attr("TITLE", "HATCH PLATE ELEM"), func(arg *iup.ItemAction) { stw.execAliasCommand("HATCHPLATEELEM") }),
				iup.Item(iup.Attr("TITLE", "EDIT PLATE ELEM"), func(arg *iup.ItemAction) { stw.execAliasCommand("EDITPLATEELEM") }),
				iup.Separator(),
				iup.Item(iup.Attr("TITLE", "MATCH PROP"), func(arg *iup.ItemAction) { stw.execAliasCommand("MATCHPROP") }),
				iup.Separator(),
				iup.Item(iup.Attr("TITLE", "COPY ELEM"), func(arg *iup.ItemAction) { stw.execAliasCommand("COPYELEM") }),
				iup.Item(iup.Attr("TITLE", "MOVE ELEM"), func(arg *iup.ItemAction) { stw.execAliasCommand("MOVEELEM") }),
				iup.Item(iup.Attr("TITLE", "MOVE NODE"), func(arg *iup.ItemAction) { stw.execAliasCommand("MOVENODE") }),
				iup.Separator(),
				iup.Item(iup.Attr("TITLE", "SEARCH ELEM"), func(arg *iup.ItemAction) { stw.execAliasCommand("SEARCHELEM") }),
				iup.Item(iup.Attr("TITLE", "NODE TO ELEM"), func(arg *iup.ItemAction) { stw.execAliasCommand("NODETOELEM") }),
				iup.Item(iup.Attr("TITLE", "ELEM TO NODE"), func(arg *iup.ItemAction) { stw.execAliasCommand("ELEMTONODE") }),
				iup.Item(iup.Attr("TITLE", "CONNECTED"), func(arg *iup.ItemAction) { stw.execAliasCommand("CONNECTED") }),
				iup.Item(iup.Attr("TITLE", "ON NODE"), func(arg *iup.ItemAction) { stw.execAliasCommand("ONNODE") }),
				iup.Separator(),
				iup.Item(iup.Attr("TITLE", "NODE NO REFERENCE"), func(arg *iup.ItemAction) { stw.execAliasCommand("NODENOREFERENCE") }),
				iup.Item(iup.Attr("TITLE", "ELEM SAME NODE"), func(arg *iup.ItemAction) { stw.execAliasCommand("ELEMSAMENODE") }),
				iup.Item(iup.Attr("TITLE", "NODE DUPLICATION"), func(arg *iup.ItemAction) { stw.execAliasCommand("NODEDUPLICATION") }),
				iup.Separator(),
				iup.Item(iup.Attr("TITLE", "CAT BY NODE"), func(arg *iup.ItemAction) { stw.execAliasCommand("CATBYNODE") }),
				iup.Item(iup.Attr("TITLE", "JOIN LINE ELEM"), func(arg *iup.ItemAction) { stw.execAliasCommand("JOINLINEELEM") }),
				iup.Separator(),
				iup.Item(iup.Attr("TITLE", "EXTRACT ARCLM"), func(arg *iup.ItemAction) { stw.execAliasCommand("EXTRACTARCLM") }),
				iup.Separator(),
				iup.Item(iup.Attr("TITLE", "DIVIDE AT ONS"), func(arg *iup.ItemAction) { stw.execAliasCommand("DIVIDEATONS") }),
				iup.Item(iup.Attr("TITLE", "DIVIDE AT MID"), func(arg *iup.ItemAction) { stw.execAliasCommand("DIVIDEATMID") }),
				iup.Separator(),
				iup.Item(iup.Attr("TITLE", "INTERSECT"), func(arg *iup.ItemAction) { stw.execAliasCommand("INTERSECT") }),
				iup.Item(iup.Attr("TITLE", "TRIM"), func(arg *iup.ItemAction) { stw.execAliasCommand("TRIM") }),
				iup.Item(iup.Attr("TITLE", "EXTEND"), func(arg *iup.ItemAction) { stw.execAliasCommand("EXTEND") }),
				iup.Separator(),
				iup.Item(iup.Attr("TITLE", "REACTION"), func(arg *iup.ItemAction) { stw.execAliasCommand("REACTION") }),
			),
		),
		iup.SubMenu("TITLE=Tool",
			iup.Menu(
				iup.Item(
					iup.Attr("TITLE", "RC lst"),
					func(arg *iup.ItemAction) {
						StartTool(filepath.Join(tooldir, "rclst/rclst.html"))
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Fig2 Keyword"),
					func(arg *iup.ItemAction) {
						StartTool(filepath.Join(tooldir, "fig2/fig2.html"))
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
			if stw.Frame != nil {
				stw.Redraw()
			} else {
				stw.dbuff.Flush()
			}
		},
		func(arg *iup.CanvasAction) {
			stw.dbuff.Activate()
			if stw.Frame != nil {
				stw.Redraw()
			} else {
				stw.dbuff.Flush()
			}
		},
		func(arg *iup.CanvasDropFiles) {
			switch filepath.Ext(arg.FileName) {
			case ".inp", ".dxf":
				stw.OpenFile(arg.FileName)
				stw.Redraw()
			default:
				if stw.Frame != nil {
					stw.ReadFile(arg.FileName)
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
			switch key.Key() {
			case KEY_ENTER:
				stw.feedCommand()
			case KEY_ESCAPE:
				stw.cline.SetAttribute("VALUE", "")
				iup.SetFocus(stw.canv)
			case KEY_TAB:
				tmp := stw.cline.GetAttribute("VALUE")
				if prevkey == KEY_TAB {
					if key.IsShift() {
						stw.cline.SetAttribute("VALUE", PrevComplete(tmp))
					} else {
						stw.cline.SetAttribute("VALUE", NextComplete(tmp))
					}
				} else {
					stw.cline.SetAttribute("VALUE", stw.Complete(tmp))
				}
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
					break
				}
				if strings.HasPrefix(val, ":") {
					c, bang, usage := exmodecomplete(val)
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
					stw.cline.SetAttribute("VALUE", fmt.Sprintf(":%s%s%s ", c, b, u))
					stw.cline.SetAttribute("CARETPOS", "100")
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
				} else {
					stw.cline.SetAttribute("INSERT", ";")
				}
				arg.Return = int32(iup.IGNORE)
			case '[':
				if key.IsCtrl() {
					stw.cline.SetAttribute("VALUE", "")
				}
			case 'N':
				if key.IsCtrl() {
					stw.New()
				}
			case 'O':
				if key.IsCtrl() {
					stw.Open()
				}
			case 'I':
				if key.IsCtrl() {
					stw.Insert()
				}
			case 'M':
				if key.IsCtrl() {
					stw.Reload()
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
					stw.ReadAll()
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
			prevkey = key.Key()
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
		if stw.Frame != nil {
			switch arg.Button {
			case BUTTON_LEFT:
				if arg.Pressed == 0 { // Released
					if pers.GetAttribute("VALUE") == "  PERSPECTIVE" {
						stw.Frame.View.Perspective = false
						pers.SetAttribute("VALUE", "  AXONOMETRIC")
					} else {
						stw.Frame.View.Perspective = true
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
	// ratel := iup.Toggle(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
	//                  fmt.Sprintf("FGCOLOR=\"%s\"", labelFGColor),
	//                  fmt.Sprintf("BGCOLOR=\"%s\"", labelBGColor),
	//                  "TITLE=\"L\"",
	//                  "VALUE=ON",
	//                  "CANFOCUS=NO",
	//                  fmt.Sprintf("SIZE=x%d",dataheight),)
	// ratel.SetCallback(func (arg *iup.ToggleAction) {
	//                       if stw.Frame != nil {
	//                           if arg.State == 1 {
	//                               stw.Frame.Show.Rate += 1
	//                           } else {
	//                               stw.Frame.Show.Rate -= 1
	//                           }
	//                       }
	//                   })
	// rates := iup.Toggle(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
	//                  fmt.Sprintf("FGCOLOR=\"%s\"", labelFGColor),
	//                  fmt.Sprintf("BGCOLOR=\"%s\"", labelBGColor),
	//                  "TITLE=\"S\"",
	//                  "VALUE=ON",
	//                  "CANFOCUS=NO",
	//                  fmt.Sprintf("SIZE=x%d",dataheight),)
	// rateq := iup.Toggle(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
	//                  fmt.Sprintf("FGCOLOR=\"%s\"", labelFGColor),
	//                  fmt.Sprintf("BGCOLOR=\"%s\"", labelBGColor),
	//                  "TITLE=\"Q\"",
	//                  "VALUE=ON",
	//                  "CANFOCUS=NO",
	//                  fmt.Sprintf("SIZE=x%d",dataheight),)
	// ratem := iup.Toggle(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
	//                  fmt.Sprintf("FGCOLOR=\"%s\"", labelFGColor),
	//                  fmt.Sprintf("BGCOLOR=\"%s\"", labelBGColor),
	//                  "TITLE=\"M\"",
	//                  "VALUE=ON",
	//                  "CANFOCUS=NO",
	//                  fmt.Sprintf("SIZE=x%d",dataheight),)
	tgecap := iup.Vbox(datasectionlabel("ELEM CAPTION"),
		stw.Labels["EC_NUM"],
		stw.Labels["EC_SECT"],
		iup.Hbox(stw.Labels["EC_RATE_L"], stw.Labels["EC_RATE_S"]))
	// iup.Hbox(ratel, rates, rateq, ratem,),)
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
		iup.Hbox(datalabel("MFACT"), stw.Labels["MFACT"]))
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
			if stw.Changed {
				if stw.Yn("CHANGED", "変更を保存しますか") {
					stw.SaveAS()
				} else {
					return
				}
			}
			arg.Return = iup.CLOSE
		},
		func(arg *iup.CommonGetFocus) {
			if stw.Frame != nil {
				if stw.InpModified {
					stw.InpModified = false
					if stw.Yn("RELOAD", fmt.Sprintf(".inpをリロードしますか？")) {
						stw.Reload()
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
	stw.papersize = A4_TATE
	stw.PageTitle = NewTextBox()
	stw.PageTitle.Font.Size = 16
	stw.PageTitle.Position = []float64{30.0, stw.CanvasSize[1] - 30.0}
	stw.Title = NewTextBox()
	stw.Title.Position = []float64{30.0, stw.CanvasSize[1] - 80.0}
	stw.Text = NewTextBox()
	stw.Text.Position = []float64{120.0, 65.0}
	stw.TextBox = make(map[string]*TextBox, 0)
	iup.SetHandle("mainwindow", stw.Dlg)
	stw.EscapeAll()
	stw.Changed = false
	stw.comhist = make([]string, CommandHistorySize)
	comhistpos = -1
	stw.SetCoord(0.0, 0.0, 0.0)
	stw.recentfiles = make([]string, nRecentFiles)
	stw.SetRecently()
	stw.SetCommandHistory()
	stw.undostack = make([]*st.Frame, nUndo)
	undopos = 0
	StartLogging()
	stw.New()
	stw.ShowLogo()
	stw.exmodech = make(chan interface{})
	stw.exmodeend = make(chan int)
	return stw
}

// }}}

func (stw *Window) FocusCanv() {
	iup.SetFocus(stw.canv)
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

func (stw *Window) Redo() {
	if NOUNDO {
		stw.addHistory("undo/redo is off")
		return
	}
	undopos--
	if undopos < 0 {
		stw.addHistory("cannot redo any more")
		undopos = 0
		return
	}
	stw.Frame = stw.undostack[undopos].Snapshot()
	stw.Redraw()
}

func (stw *Window) Undo() {
	if NOUNDO {
		stw.addHistory("undo/redo is off")
		return
	}
	undopos++
	if undopos >= nUndo {
		stw.addHistory("cannot undo any more")
		undopos = nUndo - 1
		return
	}
	if stw.undostack[undopos] == nil {
		stw.addHistory("cannot undo any more")
		undopos--
		return
	}
	stw.Frame = stw.undostack[undopos].Snapshot()
	stw.Redraw()
}

func (stw *Window) Chdir(dir string) error {
	if _, err := os.Stat(dir); err != nil {
		return err
	} else {
		stw.Cwd = dir
		return nil
	}
}

// New// {{{
func (stw *Window) New() {
	var s *st.Show
	frame := st.NewFrame()
	if stw.Frame != nil {
		s = stw.Frame.Show
	}
	w, h := stw.cdcanv.GetSize()
	stw.CanvasSize = []float64{float64(w), float64(h)}
	frame.View.Center[0] = stw.CanvasSize[0] * 0.5
	frame.View.Center[1] = stw.CanvasSize[1] * 0.5
	if s != nil {
		stw.Frame.Show = s
	}
	stw.Frame = frame
	stw.Dlg.SetAttribute("TITLE", "***")
	stw.Frame.Home = stw.Home
	stw.LinkTextValue()
	stw.Changed = false
	stw.Redraw()
}

// }}}

// Open// {{{
func (stw *Window) Open() {
	if name, ok := iup.GetOpenFile(stw.Cwd, "*.inp"); ok {
		err := stw.OpenFile(name)
		if err != nil {
			fmt.Println(err)
		}
		stw.Redraw()
	}
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
			stw.addHistory(fmt.Sprintf("%d: %s", i, fn))
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
				stw.addHistory(fmt.Sprintf("%d: %s", num, fn))
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
		fmt.Sprintf("VALUE=\"%s\"", stw.Home),
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
		err := stw.OpenFile(fn)
		if err != nil {
			stw.errormessage(err, ERROR)
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
	stw.execAliasCommand("INSERT")
}

func (stw *Window) Reload() {
	if stw.Frame != nil {
		stw.Deselect()
		v := stw.Frame.View
		s := stw.Frame.Show
		stw.OpenFile(stw.Frame.Path)
		stw.Frame.View = v
		stw.Frame.Show = s
		stw.Redraw()
	}
}

func (stw *Window) OpenDxf() {
	if name, ok := iup.GetOpenFile(stw.Cwd, "*.dxf"); ok {
		err := stw.OpenFile(name)
		if err != nil {
			fmt.Println(err)
		}
		stw.Redraw()
	}
}

func (stw *Window) OpenFile(filename string) error {
	var err error
	var s *st.Show
	fn := st.ToUtf8string(filename)
	frame := st.NewFrame()
	if stw.Frame != nil {
		s = stw.Frame.Show
	}
	w, h := stw.cdcanv.GetSize()
	stw.CanvasSize = []float64{float64(w), float64(h)}
	frame.View.Center[0] = stw.CanvasSize[0] * 0.5
	frame.View.Center[1] = stw.CanvasSize[1] * 0.5
	switch filepath.Ext(fn) {
	case ".inp":
		err = frame.ReadInp(fn, []float64{0.0, 0.0, 0.0}, 0.0, false)
		if err != nil {
			return err
		}
		stw.Frame = frame
		stw.WatchFile(fn)
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
	stw.addHistory(openstr)
	stw.Dlg.SetAttribute("TITLE", stw.Frame.Name)
	stw.Frame.Home = stw.Home
	stw.LinkTextValue()
	stw.Cwd = filepath.Dir(fn)
	stw.AddRecently(fn)
	stw.Snapshot()
	stw.Changed = false
	stw.HideLogo()
	return nil
}

func (stw *Window) WatchFile(fn string) {
	var err error
	if watcher != nil {
		watcher.Close()
	}
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		stw.errormessage(err, ERROR)
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
				stw.errormessage(err, ERROR)
			}
		}
	}()
	err = watcher.Add(fn)
	if err != nil {
		stw.errormessage(err, ERROR)
	}
}

// }}}

// Save// {{{
func (stw *Window) Save() {
	stw.SaveFile(filepath.Join(stw.Home, "hogtxt.inp"))
}

func (stw *Window) SaveAS() {
	var err error
	if name, ok := iup.GetSaveFile(filepath.Dir(stw.Frame.Path), "*.inp"); ok {
		fn := st.Ce(name, ".inp")
		err = stw.SaveFile(fn)
		if err == nil && fn != stw.Frame.Path {
			stw.Copylsts(name)
			stw.Rebase(fn)
		}
	}
}

func (stw *Window) Copylsts(name string) {
	if stw.Yn("SAVE AS", ".lst, .fig2, .kjnファイルがあればコピーしますか?") {
		for _, ext := range []string{".lst", ".fig2", ".kjn"} {
			src := st.Ce(stw.Frame.Path, ext)
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

func (stw *Window) Rebase(fn string) {
	stw.Frame.Name = filepath.Base(fn)
	stw.Frame.Project = st.ProjectName(fn)
	path, err := filepath.Abs(fn)
	if err != nil {
		stw.Frame.Path = fn
	} else {
		stw.Frame.Path = path
	}
	stw.Dlg.SetAttribute("TITLE", stw.Frame.Name)
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
		w, h := stw.cdcanv.GetSize()
		scale := math.Min(float64(w)/(xmax-xmin), float64(h)/(ymax-ymin)) * 0.9
		stw.Frame.View.Dists[1] *= scale
	}
	passwatcher = true
	err := stw.Frame.WriteInp(fn)
	if v != nil {
		stw.Frame.View = v
	}
	if err != nil {
		return err
	}
	stw.errormessage(errors.New(fmt.Sprintf("SAVE: %s", fn)), INFO)
	stw.Changed = false
	return nil
}

func (stw *Window) SaveFileSelected(fn string, els []*st.Elem) error {
	var v *st.View
	if !stw.Frame.View.Perspective {
		v = stw.Frame.View.Copy()
		stw.Frame.View.Gfact = 1.0
		stw.Frame.View.Perspective = true
		for _, n := range stw.Frame.Nodes {
			stw.Frame.View.ProjectNode(n)
		}
		xmin, xmax, ymin, ymax := stw.Bbox()
		w, h := stw.cdcanv.GetSize()
		scale := math.Min(float64(w)/(xmax-xmin), float64(h)/(ymax-ymin)) * 0.9
		stw.Frame.View.Dists[1] *= scale
	}
	passwatcher = true
	err := st.WriteInp(fn, stw.Frame.View, stw.Frame.Ai, els)
	if v != nil {
		stw.Frame.View = v
	}
	if err != nil {
		return err
	}
	stw.errormessage(errors.New(fmt.Sprintf("SAVE: %s", fn)), INFO)
	stw.Changed = false
	return nil
}

// }}}

func (stw *Window) Close(force bool) {
	if !force && stw.Changed {
		if stw.Yn("CHANGED", "変更を保存しますか") {
			stw.SaveAS()
		} else {
			return
		}
	}
	stw.Dlg.Destroy()
}

// Read// {{{
func (stw *Window) Read() {
	if stw.Frame != nil {
		if name, ok := iup.GetOpenFile("", ""); ok {
			err := stw.ReadFile(name)
			if err != nil {
				stw.errormessage(err, ERROR)
			}
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
				stw.errormessage(err, ERROR)
			} else {
				read[nread] = ext
				nread++
			}
		}
		stw.addHistory(fmt.Sprintf("READ: %s", strings.Join(read, " ")))
	}
}

func (stw *Window) ReadFile(filename string) error {
	var err error
	switch filepath.Ext(filename) {
	default:
		fmt.Println("Unknown Format")
		return nil
	case ".inp":
		x, y, z, err := stw.QueryCoord("Open Input")
		if err != nil {
			x = 0.0
			y = 0.0
			z = 0.0
		}
		err = stw.Frame.ReadInp(filename, []float64{x, y, z}, 0.0, false)
	case ".inl", ".ihx", ".ihy":
		err = stw.Frame.ReadData(filename)
	case ".otl", ".ohx", ".ohy":
		err = stw.Frame.ReadResult(filename, st.UPDATE_RESULT)
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
	v := stw.Frame.View.Copy()
	pw, ph := pcanv.GetSize() // seems to be [mm]/25.4*[dpi]*2
	w0, h0 := stw.dbuff.GetSize()
	w, h, err := stw.PaperSize(stw.dbuff)
	if err != nil {
		return v, 0.0, err
	}
	factor := math.Min(float64(pw)/w, float64(ph)/h)
	stw.CanvasSize[0] = float64(pw)
	stw.CanvasSize[1] = float64(ph)
	stw.Frame.View.Gfact *= factor
	stw.Frame.View.Center[0] = float64(pw)*0.5 + factor*(stw.Frame.View.Center[0]-0.5*float64(w0))
	stw.Frame.View.Center[1] = float64(ph)*0.5 + factor*(stw.Frame.View.Center[1]-0.5*float64(h0))
	stw.Frame.Show.ConfSize *= factor
	stw.Frame.Show.BondSize *= factor
	stw.Frame.Show.KijunSize = 150
	stw.Frame.Show.MassSize *= factor
	for _, m := range stw.Frame.Measures {
		m.ArrowSize = 75.0
	}
	for i := 0; i < 2; i++ {
		stw.PageTitle.Position[i] *= factor
		stw.Title.Position[i] *= factor
		stw.Text.Position[i] *= factor
	}
	for _, t := range stw.TextBox {
		for i := 0; i < 2; i++ {
			t.Position[i] *= factor
		}
	}
	return v, factor, nil
}

func (stw *Window) PaperSize(canv *cd.Canvas) (float64, float64, error) {
	w, h := canv.GetSize()
	length := math.Min(float64(w), float64(h)) * 0.9
	val := 1.0 / math.Sqrt(2)
	switch stw.papersize {
	default:
		return 0.0, 0.0, errors.New("unknown papersize")
	case A4_TATE, A3_TATE:
		return length * val, length, nil
	case A4_YOKO, A3_YOKO:
		return length, length * val, nil
	}
}

func (stw *Window) Print() {
	if stw.Frame == nil {
		return
	}
	pcanv, err := SetPrinter(stw.Frame.Name)
	if err != nil {
		stw.errormessage(err, ERROR)
		return
	}
	v, factor, err := stw.FittoPrinter(pcanv)
	if err != nil {
		stw.errormessage(err, ERROR)
		return
	}
	PlateEdgeColor = cd.CD_BLACK
	BondColor = cd.CD_BLACK
	ConfColor = cd.CD_BLACK
	MomentColor = cd.CD_BLACK
	KijunColor = cd.CD_BLACK
	MeasureColor = cd.CD_BLACK
	StressTextColor = cd.CD_BLACK
	YieldedTextColor = cd.CD_BLACK
	BrittleTextColor = cd.CD_BLACK
	switch stw.Frame.Show.ColorMode {
	default:
		stw.DrawFrame(pcanv, stw.Frame.Show.ColorMode, false)
	case st.ECOLOR_WHITE:
		stw.Frame.Show.ColorMode = st.ECOLOR_BLACK
		stw.DrawFrame(pcanv, st.ECOLOR_BLACK, false)
		stw.Frame.Show.ColorMode = st.ECOLOR_WHITE
	}
	stw.DrawTexts(pcanv, true)
	pcanv.Kill()
	w, h := stw.cdcanv.GetSize()
	stw.CanvasSize = []float64{float64(w), float64(h)}
	stw.Frame.Show.ConfSize /= factor
	stw.Frame.Show.BondSize /= factor
	stw.Frame.Show.KijunSize = 12.0
	stw.Frame.Show.MassSize /= factor
	for _, m := range stw.Frame.Measures {
		m.ArrowSize = 6.0
	}
	stw.Frame.View = v
	for i := 0; i < 2; i++ {
		stw.PageTitle.Position[i] /= factor
		stw.Title.Position[i] /= factor
		stw.Text.Position[i] /= factor
	}
	for _, t := range stw.TextBox {
		for i := 0; i < 2; i++ {
			t.Position[i] /= factor
		}
	}
	PlateEdgeColor = defaultPlateEdgeColor
	BondColor = defaultBondColor
	ConfColor = defaultConfColor
	MomentColor = defaultMomentColor
	KijunColor = defaultKijunColor
	MeasureColor = defaultMeasureColor
	StressTextColor = defaultStressTextColor
	YieldedTextColor = defaultYieldedTextColor
	BrittleTextColor = defaultBrittleTextColor
	stw.Redraw()
}

func (stw *Window) PrintSVG(filename string) error {
	err := stsvg.Print(stw.Frame, filename)
	if err != nil {
		return err
	}
	return nil
}

func (stw *Window) PrintFig2(filename string) error {
	if stw.Frame == nil {
		return errors.New("PrintFig2: no frame opened")
	}
	pcanv, err := SetPrinter(stw.Frame.Name)
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
	stw.Frame.Show.ConfSize /= factor
	stw.Frame.Show.BondSize /= factor
	stw.Frame.View = v
	stw.Redraw()
	return nil
}

func (stw *Window) ReadFig2(filename string) error {
	if stw.Frame == nil {
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
		err := stw.fig2keyword(txt, un)
		if err != nil {
			return err
		}
	}
	stw.DrawFrame(pcanv, stw.Frame.Show.ColorMode, false)
	stw.DrawTexts(pcanv, true)
	pcanv.Flush()
	return nil
}

func (stw *Window) fig2mode(command string) error {
	if len(command) == 1 {
		return st.NotEnoughArgs("fig2mode")
	}
	if command == "'." {
		return stw.fig2mode(stw.lastfig2command)
	}
	stw.lastfig2command = command
	command = command[1:]
	var un bool
	if strings.HasPrefix(command, "!") {
		un = true
		command = command[1:]
	} else {
		un = false
	}
	tmpargs := strings.Split(command, " ")
	args := make([]string, len(tmpargs))
	narg := 0
	for i := 0; i < len(tmpargs); i++ {
		if tmpargs[i] != "" {
			args[narg] = tmpargs[i]
			narg++
		}
	}
	args = args[:narg]
	return stw.fig2keyword(args, un)
}

func fig2keywordcomplete(command string) (string, bool) {
	usage := strings.HasSuffix(command, "?")
	cname := strings.TrimSuffix(command, "?")
	cname = strings.ToLower(strings.TrimPrefix(cname, "'"))
	var rtn string
	for _, ab := range fig2abbrev {
		pat := abbrev.MustCompile(ab)
		if pat.MatchString(cname) {
			rtn = pat.Longest()
			break
		}
	}
	if rtn == "" {
		rtn = cname
	}
	return rtn, usage
}

func (stw *Window) fig2keyword(lis []string, un bool) error {
	if len(lis) < 1 {
		return st.NotEnoughArgs("Fig2Keyword")
	}
	showhtml := func(fn string) {
		f := filepath.Join(tooldir, "fig2/keywords", fn)
		if st.FileExists(f) {
			cmd := exec.Command("cmd", "/C", "start", f)
			cmd.Start()
		}
	}
	key, usage := fig2keywordcomplete(strings.ToLower(lis[0]))
	switch key {
	default:
		if k, ok := stw.Frame.Kijuns[key]; ok {
			d := k.Direction()
			var axis int
			if st.IsParallel(d, st.XAXIS, 1e-4) {
				axis = 1
			} else if st.IsParallel(d, st.YAXIS, 1e-4) {
				axis = 0
			}
			axisrange(stw, axis, k.Start[axis], k.Start[axis], false)
			return nil
		}
		stw.errormessage(errors.New(fmt.Sprintf("no fig2 keyword: %s", key)), INFO)
		return nil
	case "gfact":
		val, err := strconv.ParseFloat(lis[1], 64)
		if err != nil {
			return err
		}
		stw.Frame.View.Gfact = val
		stw.Labels["GFACT"].SetAttribute("VALUE", fmt.Sprintf("%f", stw.Frame.View.Gfact))
	case "focus":
		if len(lis) < 2 {
			return st.NotEnoughArgs("FOCUS")
		}
		switch strings.ToUpper(lis[1]) {
		case "CENTER", "CENTRE":
			stw.Frame.SetFocus(nil)
		case "NODE":
			val, err := strconv.ParseInt(lis[2], 10, 64)
			if err != nil {
				return err
			}
			if n, ok := stw.Frame.Nodes[int(val)]; ok {
				stw.Frame.SetFocus(n.Coord)
			}
			w, h := stw.cdcanv.GetSize()
			stw.Frame.View.Center[0] = float64(w) * 0.5
			stw.Frame.View.Center[1] = float64(h) * 0.5
		case "ELEM":
			val, err := strconv.ParseInt(lis[2], 10, 64)
			if err != nil {
				return err
			}
			if el, ok := stw.Frame.Elems[int(val)]; ok {
				stw.Frame.SetFocus(el.MidPoint())
			}
			w, h := stw.cdcanv.GetSize()
			stw.Frame.View.Center[0] = float64(w) * 0.5
			stw.Frame.View.Center[1] = float64(h) * 0.5
		default:
			if len(lis) < 4 {
				return st.NotEnoughArgs("FOCUS")
			}
			for i, str := range []string{"FOCUSX", "FOCUSY", "FOCUSZ"} {
				if lis[1+i] == "_" {
					continue
				}
				val, err := strconv.ParseFloat(lis[1+i], 64)
				if err != nil {
					return err
				}
				stw.Frame.View.Focus[i] = val
				stw.Labels[str].SetAttribute("VALUE", fmt.Sprintf("%f", stw.Frame.View.Focus[i]))
			}
		}
	case "fit":
		stw.Frame.SetFocus(nil)
		stw.DrawFrameNode()
		stw.ShowCenter()
	case "angle":
		if len(lis) < 3 {
			return st.NotEnoughArgs("ANGLE")
		}
		for i, str := range []string{"PHI", "THETA"} {
			if lis[1+i] == "_" {
				continue
			}
			val, err := strconv.ParseFloat(lis[1+i], 64)
			if err != nil {
				return err
			}
			stw.Frame.View.Angle[i] = val
			stw.Labels[str].SetAttribute("VALUE", fmt.Sprintf("%f", stw.Frame.View.Angle[i]))
		}
	case "dists":
		if len(lis) < 3 {
			return st.NotEnoughArgs("DISTS")
		}
		for i, str := range []string{"DISTR", "DISTL"} {
			if lis[1+i] == "_" {
				continue
			}
			val, err := strconv.ParseFloat(lis[1+i], 64)
			if err != nil {
				return err
			}
			stw.Frame.View.Dists[i] = val
			stw.Labels[str].SetAttribute("VALUE", fmt.Sprintf("%f", stw.Frame.View.Dists[i]))
		}
	case "perspective":
		stw.Frame.View.Perspective = true
	case "axonometric":
		stw.Frame.View.Perspective = false
	case "unit":
		if usage {
			stw.addHistory("'unit force,length")
			stw.addHistory(fmt.Sprintf("CURRENT FORCE UNIT: %s %.3f", stw.Frame.Show.UnitName[0], stw.Frame.Show.Unit[0]))
			stw.addHistory(fmt.Sprintf("CURRENT LENGTH UNIT: %s %.3f", stw.Frame.Show.UnitName[1], stw.Frame.Show.Unit[1]))
			showhtml("UNIT.html")
			return nil
		}
		if un {
			stw.Frame.Show.Unit = []float64{1.0, 1.0}
			stw.Frame.Show.UnitName = []string{"tf", "m"}
			return nil
		}
		if len(lis) < 2 {
			return st.NotEnoughArgs("UNIT")
		}
		ustr := strings.Split(strings.ToLower(lis[1]), ",")
		if len(ustr) < 2 {
			return errors.New("'unit: incorrect format")
		}
		switch ustr[0] {
		case "tf":
			stw.Frame.Show.Unit[0] = 1.0
			stw.Frame.Show.UnitName[0] = "tf"
		case "kgf":
			stw.Frame.Show.Unit[0] = 1000.0
			stw.Frame.Show.UnitName[0] = "kgf"
		case "kn":
			stw.Frame.Show.Unit[0] = st.SI
			stw.Frame.Show.UnitName[0] = "kN"
		}
		switch ustr[1] {
		case "m":
			stw.Frame.Show.Unit[1] = 1.0
			stw.Frame.Show.UnitName[1] = "m"
		case "cm":
			stw.Frame.Show.Unit[0] = 100.0
			stw.Frame.Show.UnitName[0] = "cm"
		case "mm":
			stw.Frame.Show.Unit[0] = 1000.0
			stw.Frame.Show.UnitName[0] = "mm"
		}
	case "dfact":
		if len(lis) < 2 {
			return st.NotEnoughArgs("DFACT")
		}
		val, err := strconv.ParseFloat(lis[1], 64)
		if err != nil {
			return err
		}
		stw.Frame.Show.Dfact = val
		stw.Labels["DFACT"].SetAttribute("VALUE", fmt.Sprintf("%f", val))
	case "rfact":
		if len(lis) < 2 {
			return st.NotEnoughArgs("RFACT")
		}
		val, err := strconv.ParseFloat(lis[1], 64)
		if err != nil {
			return err
		}
		stw.Frame.Show.Rfact = val
	case "mfact":
		if len(lis) < 2 {
			return st.NotEnoughArgs("MFACT")
		}
		val, err := strconv.ParseFloat(lis[1], 64)
		if err != nil {
			return err
		}
		stw.Frame.Show.Mfact = val
		stw.Labels["MFACT"].SetAttribute("VALUE", fmt.Sprintf("%f", val))
	case "gaxis":
		if un {
			stw.Frame.Show.GlobalAxis = false
			stw.Labels["GAXIS"].SetAttribute("FGCOLOR", labelOFFColor)
		} else {
			stw.Frame.Show.GlobalAxis = true
			stw.Labels["GAXIS"].SetAttribute("FGCOLOR", labelFGColor)
			if len(lis) >= 2 {
				val, err := strconv.ParseFloat(lis[1], 64)
				if err != nil {
					return err
				}
				stw.Frame.Show.GlobalAxisSize = val
				stw.Labels["GAXISSIZE"].SetAttribute("VALUE", fmt.Sprintf("%f", val))
			}
		}
	case "eaxis":
		if un {
			stw.Frame.Show.ElementAxis = false
			stw.Labels["EAXIS"].SetAttribute("FGCOLOR", labelOFFColor)
		} else {
			stw.Frame.Show.ElementAxis = true
			stw.Labels["EAXIS"].SetAttribute("FGCOLOR", labelFGColor)
			if len(lis) >= 2 {
				val, err := strconv.ParseFloat(lis[1], 64)
				if err != nil {
					return err
				}
				stw.Frame.Show.ElementAxisSize = val
				stw.Labels["EAXISSIZE"].SetAttribute("VALUE", fmt.Sprintf("%f", val))
			}
		}
	case "noaxis":
		stw.Frame.Show.GlobalAxis = false
		stw.Labels["GAXIS"].SetAttribute("FGCOLOR", labelOFFColor)
		stw.Frame.Show.ElementAxis = false
		stw.Labels["EAXIS"].SetAttribute("FGCOLOR", labelOFFColor)
	case "elem":
		for i, _ := range st.ETYPES {
			stw.HideEtype(i)
		}
		for _, val := range lis[1:] {
			et := strings.ToUpper(val)
			for i, e := range st.ETYPES {
				if et == e {
					stw.ShowEtype(i)
				}
			}
		}
	case "elem+":
		for _, val := range lis[1:] {
			et := strings.ToUpper(val)
			for i, e := range st.ETYPES {
				if et == e {
					stw.ShowEtype(i)
				}
			}
		}
	case "elem-":
		for _, val := range lis[1:] {
			et := strings.ToUpper(val)
			for i, e := range st.ETYPES {
				if et == e {
					stw.HideEtype(i)
				}
			}
		}
	case "section":
		stw.HideAllSection()
		for _, tmp := range lis[1:] {
			val, err := strconv.ParseInt(tmp, 10, 64)
			if err != nil {
				continue
			}
			stw.ShowSection(int(val))
		}
	case "section+":
		for _, tmp := range lis[1:] {
			val, err := strconv.ParseInt(tmp, 10, 64)
			if err != nil {
				continue
			}
			stw.ShowSection(int(val))
		}
	case "section-":
		for _, tmp := range lis[1:] {
			val, err := strconv.ParseInt(tmp, 10, 64)
			if err != nil {
				continue
			}
			stw.HideSection(int(val))
		}
	case "kijun":
		if un {
			stw.Frame.Show.Kijun = false
			stw.Labels["KIJUN"].SetAttribute("FGCOLOR", labelOFFColor)
		} else {
			stw.Frame.Show.Kijun = true
			stw.Labels["KIJUN"].SetAttribute("FGCOLOR", labelFGColor)
		}
	case "measure":
		if usage {
			stw.addHistory("'measure kijun x1 x2 offset dotsize rotate overwrite")
			stw.addHistory("'measure nnum1 nnum2 direction offset dotsize rotate overwrite")
			showhtml("MEASURE.html")
			return nil
		}
		if un {
			stw.Frame.Show.Measure = false
		} else {
			stw.Frame.Show.Measure = true
			if len(lis) < 4 {
				return st.NotEnoughArgs("MEASURE")
			}
			if abbrev.For("k/ijun", strings.ToLower(lis[1])) { // measure kijun x1 x2 offset dotsize rotate overwrite
				if k1, ok := stw.Frame.Kijuns[strings.ToLower(lis[2])]; ok {
					if k2, ok := stw.Frame.Kijuns[strings.ToLower(lis[3])]; ok {
						m := stw.Frame.AddMeasure(k1.Start, k2.Start, k1.Direction())
						m.Text = fmt.Sprintf("%.0f", k1.Distance(k2)*1000)
						if len(lis) < 5 {
							return nil
						}
						val, err := strconv.ParseFloat(lis[4], 64)
						if err != nil {
							return err
						}
						m.Gap = 2 * val
						m.Extension = -val
						if len(lis) < 6 {
							return nil
						}
						val, err = strconv.ParseFloat(lis[5], 64)
						if err != nil {
							return err
						}
						m.ArrowSize = val
						if len(lis) < 7 {
							return nil
						}
						val, err = strconv.ParseFloat(lis[6], 64)
						if err != nil {
							return err
						}
						m.Rotate = val
						if len(lis) < 8 {
							return nil
						}
						m.Text = st.ToUtf8string(lis[7])
					} else {
						return errors.New(fmt.Sprintf("no kijun named %s", lis[3]))
					}
				} else {
					return errors.New(fmt.Sprintf("no kijun named %s", lis[2]))
				}
			} else { // measure nnum1 nnum2 direction offset dotsize rotate overwrite
				nnum, err := strconv.ParseInt(lis[1], 10, 64)
				if err != nil {
					return err
				}
				if n1, ok := stw.Frame.Nodes[int(nnum)]; ok {
					nnum, err := strconv.ParseInt(lis[2], 10, 64)
					if err != nil {
						return err
					}
					if n2, ok := stw.Frame.Nodes[int(nnum)]; ok {
						var u, v []float64
						switch strings.ToUpper(lis[3]) {
						case "X":
							u = st.XAXIS
							v = st.YAXIS
						case "Y":
							u = st.YAXIS
							v = st.XAXIS
						case "Z":
							u = st.ZAXIS
						case "V":
							v = st.Direction(n1, n2, true)
							u = st.Cross(v, st.ZAXIS)
						default:
							return errors.New("unknown direction")
						}
						m := stw.Frame.AddMeasure(n1.Coord, n2.Coord, u)
						m.Text = fmt.Sprintf("%.0f", st.VectorDistance(n1, n2, v)*1000)
						if len(lis) < 5 {
							return nil
						}
						val, err := strconv.ParseFloat(lis[4], 64)
						if err != nil {
							return err
						}
						m.Extension = val
						if len(lis) < 6 {
							return nil
						}
						val, err = strconv.ParseFloat(lis[5], 64)
						if err != nil {
							return err
						}
						m.ArrowSize = val
						if len(lis) < 7 {
							return nil
						}
						val, err = strconv.ParseFloat(lis[6], 64)
						if err != nil {
							return err
						}
						m.Rotate = val
						if len(lis) < 8 {
							return nil
						}
						m.Text = st.ToUtf8string(lis[7])
					} else {
						return errors.New(fmt.Sprintf("no node %d", nnum))
					}
				} else {
					return errors.New(fmt.Sprintf("no node %d", nnum))
				}
			}
		}
	case "elemcode":
		if un {
			stw.ElemCaptionOff("EC_NUM")
		} else {
			stw.ElemCaptionOn("EC_NUM")
		}
	case "sectcode":
		if un {
			stw.ElemCaptionOff("EC_SECT")
		} else {
			stw.ElemCaptionOn("EC_SECT")
		}
	case "width":
		if un {
			stw.ElemCaptionOff("EC_WIDTH")
		} else {
			stw.ElemCaptionOn("EC_WIDTH")
		}
	case "height":
		if un {
			stw.ElemCaptionOff("EC_HEIGHT")
		} else {
			stw.ElemCaptionOn("EC_HEIGHT")
		}
	case "srcancolor":
		if un {
			stw.SetColorMode(st.ECOLOR_WHITE)
		} else {
			stw.SetColorMode(st.ECOLOR_RATE)
		}
	case "srcanrate":
		if usage {
			stw.addHistory("'srcanrate [long/short]")
			return nil
		}
		long := true
		short := true
		if len(lis) > 2 {
			if strings.EqualFold(lis[1], "long") {
				short = false
			} else if strings.EqualFold(lis[1], "short") {
				long = false
			}
		}
		if un {
			if long {
				stw.ElemCaptionOff("EC_RATE_L")
				stw.Labels["EC_RATE_L"].SetAttribute("FGCOLOR", labelOFFColor)
			}
			if short {
				stw.ElemCaptionOff("EC_RATE_S")
				stw.Labels["EC_RATE_S"].SetAttribute("FGCOLOR", labelOFFColor)
			}
		} else {
			if long {
				stw.ElemCaptionOn("EC_RATE_L")
				stw.Labels["EC_RATE_L"].SetAttribute("FGCOLOR", labelFGColor)
			}
			if short {
				stw.ElemCaptionOn("EC_RATE_S")
				stw.Labels["EC_RATE_S"].SetAttribute("FGCOLOR", labelFGColor)
			}
		}
	case "stress":
		if usage {
			stw.addHistory("'stress [etype/sectcode] [period] [stressname]")
			return nil
		}
		l := len(lis)
		if l < 2 {
			if un {
				for etype := st.COLUMN; etype <= st.SLAB; etype++ {
					for i := 0; i < 6; i++ {
						stw.StressOff(etype, uint(i))
					}
				}
				return nil
			} else {
				return st.NotEnoughArgs("STRESS")
			}
		}
		etype := -1
		var sects []int
		et := strings.ToLower(lis[1])
		switch {
		case abbrev.For("co/lumn", et):
			etype = st.COLUMN
		case abbrev.For("gi/rder", et):
			etype = st.GIRDER
		case abbrev.For("br/ace", et):
			etype = st.BRACE
		case abbrev.For("wb/race", et):
			etype = st.WBRACE
		case abbrev.For("sb/race", et):
			etype = st.SBRACE
		case abbrev.For("wa/ll", et):
			etype = st.WALL
		case abbrev.For("sl/ab", et):
			etype = st.SLAB
		}
		if etype == -1 {
			sectnum := regexp.MustCompile("^ *([0-9]+) *$")
			sectrange := regexp.MustCompile("(?i)^ *range *[(] *([0-9]+) *, *([0-9]+) *[)] *$")
			switch {
			case sectnum.MatchString(lis[1]):
				val, _ := strconv.ParseInt(lis[1], 10, 64)
				if _, ok := stw.Frame.Sects[int(val)]; ok {
					sects = []int{int(val)}
				}
			case sectrange.MatchString(lis[1]):
				fs := sectrange.FindStringSubmatch(lis[1])
				if len(fs) < 3 {
					return errors.New("STRESS, invalid input")
				}
				start, _ := strconv.ParseInt(fs[1], 10, 64)
				end, _ := strconv.ParseInt(fs[2], 10, 64)
				if start > end {
					return errors.New("STRESS, invalid input")
				}
				sects = make([]int, int(end-start))
				nsect := 0
				for i := int(start); i < int(end); i++ {
					if _, ok := stw.Frame.Sects[i]; ok {
						sects[nsect] = i
						nsect++
					}
				}
				sects = sects[:nsect]
			}
			if sects == nil || len(sects) == 0 {
				return errors.New("STRESS, invalid input")
			}
		}
		if l < 3 {
			if sects != nil && len(sects) > 0 {
				if un {
					for _, snum := range sects {
						delete(stw.Frame.Show.Stress, snum)
					}
				} else {
					for _, snum := range sects {
						for i := 0; i < 6; i++ {
							stw.StressOff(snum, uint(i))
						}
					}
				}
			} else {
				for i := 0; i < 6; i++ {
					stw.StressOff(etype, uint(i))
				}
			}
			break
		}
		if l < 4 {
			return st.NotEnoughArgs("STRESS")
		}
		period := strings.ToUpper(lis[2])
		stw.SetPeriod(period)
		index := -1
		val := strings.ToUpper(lis[3])
		for i, str := range []string{"N", "QX", "QY", "MZ", "MX", "MY"} {
			if val == str {
				index = i
				break
			}
		}
		if index == -1 {
			break
		}
		if un {
			if sects != nil && len(sects) > 0 {
				for _, snum := range sects {
					stw.StressOff(snum, uint(index))
				}
			} else {
				stw.StressOff(etype, uint(index))
			}
		} else {
			if sects != nil && len(sects) > 0 {
				for _, snum := range sects {
					stw.StressOn(snum, uint(index))
				}
			} else {
				stw.StressOn(etype, uint(index))
			}
		}
	case "prestress":
		if un {
			stw.ElemCaptionOff("EC_PREST")
		} else {
			stw.ElemCaptionOn("EC_PREST")
		}
	case "stiff":
		if usage {
			stw.addHistory("'stiff [x,y]")
			return nil
		}
		if len(lis) < 2 {
			if un {
				stw.ElemCaptionOff("EC_STIFF_X")
				stw.ElemCaptionOff("EC_STIFF_Y")
				return nil
			} else {
				return st.NotEnoughArgs("stiff")
			}
		}
		switch strings.ToUpper(lis[1]) {
		default:
			return errors.New("unknown period")
		case "X":
			if un {
				stw.ElemCaptionOff("EC_STIFF_X")
			} else {
				stw.ElemCaptionOn("EC_STIFF_X")
			}
		case "Y":
			if un {
				stw.ElemCaptionOff("EC_STIFF_Y")
			} else {
				stw.ElemCaptionOn("EC_STIFF_Y")
			}
		}
	case "deformation":
		if un {
			stw.DeformationOff()
		} else {
			if len(lis) >= 2 {
				stw.SetPeriod(strings.ToUpper(lis[1]))
			}
			stw.DeformationOn()
		}
	case "disp":
		if un {
			if len(lis) < 2 {
				for i := 0; i < 6; i++ {
					stw.DispOff(i)
				}
			} else {
				stw.SetPeriod(strings.ToUpper(lis[1]))
				dir := strings.ToUpper(lis[2])
				for i, str := range []string{"X", "Y", "Z", "TX", "TY", "TZ"} {
					if dir == str {
						stw.DispOff(i)
						break
					}
				}
			}
		} else {
			if len(lis) < 3 {
				return st.NotEnoughArgs("DISP")
			}
			stw.SetPeriod(strings.ToUpper(lis[1]))
			dir := strings.ToUpper(lis[2])
			for i, str := range []string{"X", "Y", "Z", "TX", "TY", "TZ"} {
				if dir == str {
					stw.DispOn(i)
					break
				}
			}
		}
	case "eccentric":
		if un {
			stw.Frame.Show.Fes = false
		} else {
			stw.Frame.Show.Fes = true
			if len(lis) >= 2 {
				val, err := strconv.ParseFloat(lis[1], 64)
				if err != nil {
					return err
				}
				stw.Frame.Show.MassSize = val
			}
		}
	case "draw":
		if un {
			for k := range stw.Frame.Show.Draw {
				stw.Frame.Show.Draw[k] = false
			}
		} else {
			if len(lis) < 2 {
				return st.NotEnoughArgs("DRAW")
			}
			var val int
			switch {
			case re_column.MatchString(lis[1]):
				val = st.COLUMN
			case re_girder.MatchString(lis[1]):
				val = st.GIRDER
			case re_slab.MatchString(lis[1]):
				val = st.BRACE
			case re_wall.MatchString(lis[1]):
				val = st.WALL
			case re_slab.MatchString(lis[1]):
				val = st.SLAB
			default:
				tmp, err := strconv.ParseInt(lis[1], 10, 64)
				if err != nil {
					return err
				}
				val = int(tmp)
			}
			if val != 0 {
				stw.Frame.Show.Draw[val] = true
			}
			if len(lis) >= 3 {
				tmp, err := strconv.ParseFloat(lis[2], 64)
				if err != nil {
					return err
				}
				stw.Frame.Show.DrawSize = tmp
			}
		}
	case "alias":
		if un {
			if len(lis) < 2 {
				sectionaliases = make(map[int]string, 0)
			} else {
				for _, j := range lis[1:] {
					val, err := strconv.ParseInt(j, 10, 64)
					if err != nil {
						continue
					}
					if _, ok := stw.Frame.Sects[int(val)]; ok {
						delete(sectionaliases, int(val))
					}
				}
			}
		} else {
			if len(lis) < 2 {
				return st.NotEnoughArgs("ALIAS")
			}
			val, err := strconv.ParseInt(lis[1], 10, 64)
			if err != nil {
				return err
			}
			if _, ok := stw.Frame.Sects[int(val)]; ok {
				if len(lis) < 3 {
					sectionaliases[int(val)] = ""
				} else {
					sectionaliases[int(val)] = st.ToUtf8string(lis[2])
				}
			}
		}
	case "anonymous":
		for _, str := range lis[1:] {
			val, err := strconv.ParseInt(str, 10, 64)
			if err != nil {
				continue
			}
			if _, ok := stw.Frame.Sects[int(val)]; ok {
				sectionaliases[int(val)] = ""
			}
		}
	case "nodecode":
		if un {
			stw.NodeCaptionOff("NC_NUM")
		} else {
			stw.NodeCaptionOn("NC_NUM")
		}
	case "weight":
		if un {
			stw.NodeCaptionOff("NC_WEIGHT")
		} else {
			stw.NodeCaptionOn("NC_WEIGHT")
		}
	case "conf":
		if un {
			stw.Frame.Show.Conf = false
			stw.Labels["CONF"].SetAttribute("FGCOLOR", labelOFFColor)
		} else {
			stw.Frame.Show.Conf = true
			stw.Labels["CONF"].SetAttribute("FGCOLOR", labelFGColor)
			if len(lis) >= 2 {
				val, err := strconv.ParseFloat(lis[1], 64)
				if err != nil {
					return err
				}
				stw.Frame.Show.ConfSize = val
				stw.Labels["CONFSIZE"].SetAttribute("VALUE", fmt.Sprintf("%.1f", val))
			}
		}
	case "pile":
		if un {
			stw.NodeCaptionOff("NC_PILE")
		} else {
			stw.NodeCaptionOn("NC_PILE")
		}
	case "fence":
		if len(lis) < 3 {
			return st.NotEnoughArgs("FENCE")
		}
		var axis int
		switch strings.ToUpper(lis[1]) {
		default:
			return errors.New("unknown direction")
		case "X":
			axis = 0
		case "Y":
			axis = 1
		case "Z":
			axis = 2
		}
		val, err := strconv.ParseFloat(lis[2], 64)
		if err != nil {
			return err
		}
		stw.SelectElem = stw.Frame.Fence(axis, val, false)
		stw.HideNotSelected()
	case "period":
		stw.SetPeriod(strings.ToUpper(lis[1]))
	case "period++":
		stw.IncrementPeriod(1)
	case "period--":
		stw.IncrementPeriod(-1)
	case "nocaption":
		for _, nc := range st.NODECAPTIONS {
			stw.NodeCaptionOff(nc)
		}
		for _, ec := range st.ELEMCAPTIONS {
			stw.ElemCaptionOff(ec)
		}
		for etype := range st.ETYPES[1:] {
			for i := 0; i < 6; i++ {
				stw.StressOff(etype, uint(i))
			}
		}
	case "nolegend":
		if un {
			stw.Frame.Show.NoLegend = false
		} else {
			stw.Frame.Show.NoLegend = true
		}
	case "nomomentvalue":
		if un {
			stw.Frame.Show.NoMomentValue = false
		} else {
			stw.Frame.Show.NoMomentValue = true
		}
	case "ncolor":
		stw.SetColorMode(st.ECOLOR_N)
	case "pagetitle":
		if un {
			stw.PageTitle.Value = make([]string, 0)
			stw.PageTitle.Hide = true
		} else {
			stw.PageTitle.Value = append(stw.PageTitle.Value, st.ToUtf8string(strings.Join(lis[1:], " ")))
			stw.PageTitle.Hide = false
		}
	case "title":
		if un {
			stw.Title.Value = make([]string, 0)
			stw.Title.Hide = true
		} else {
			stw.Title.Value = append(stw.Title.Value, st.ToUtf8string(strings.Join(lis[1:], " ")))
			stw.Title.Hide = false
		}
	case "text":
		if un {
			stw.Text.Value = make([]string, 0)
			stw.Text.Hide = true
		} else {
			stw.Text.Value = append(stw.Text.Value, st.ToUtf8string(strings.Join(lis[1:], " ")))
			stw.Text.Hide = false
		}
	case "position":
		if len(lis) < 4 {
			return st.NotEnoughArgs("POSITION")
		}
		xpos, err := strconv.ParseFloat(lis[2], 64)
		if err != nil {
			return err
		}
		ypos, err := strconv.ParseFloat(lis[3], 64)
		if err != nil {
			return err
		}
		switch strings.ToUpper(lis[1]) {
		case "PAGETITLE":
			stw.PageTitle.Position[0] = xpos
			stw.PageTitle.Position[1] = ypos
		case "TITLE":
			stw.Title.Position[0] = xpos
			stw.Title.Position[1] = ypos
		case "TEXT":
			stw.Text.Position[0] = xpos
			stw.Text.Position[1] = ypos
		case "LEGEND":
			stw.Frame.Show.LegendPosition[0] = int(xpos)
			stw.Frame.Show.LegendPosition[1] = int(ypos)
		}
	}
	return nil
}

func (stw *Window) Edit(fn string) {
	cmd := exec.Command("cmd", "/C", "start", fn)
	cmd.Start()
}

func (stw *Window) Vim(fn string) {
	cmd := exec.Command("gvim", fn)
	cmd.Start()
}

func (stw *Window) EditInp() {
	if stw.Frame != nil {
		cmd := exec.Command("cmd", "/C", "start", stw.Frame.Path)
		cmd.Start()
	}
}

func (stw *Window) EditReadme(dir string) {
	fn := filepath.Join(dir, "readme.txt")
	stw.Vim(fn)
}

func StartTool(fn string) {
	cmd := exec.Command("cmd", "/C", "start", fn)
	cmd.Start()
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

// Help// {{{
func ShowReleaseNote() {
	cmd := exec.Command("cmd", "/C", "start", releasenote)
	cmd.Start()
}

func (stw *Window) ShowLogo() {
	w, h := stw.dbuff.GetSize()
	STLOGO.Position = []float64{float64(w) * 0.5, float64(h) * 0.5}
	STLOGO.Hide = false
}

func (stw *Window) HideLogo() {
	STLOGO.Hide = true
}

func (stw *Window) ShowAbout() {
	dlg := iup.MessageDlg(fmt.Sprintf("FONTFACE=%s", commandFontFace),
		fmt.Sprintf("FONTSIZE=%s", commandFontSize))
	dlg.SetAttribute("TITLE", "バージョン情報")
	if stw.Frame != nil {
		dlg.SetAttribute("VALUE", fmt.Sprintf("VERSION: %s\n%s\n\nNAME\t: %s\nPROJECT\t: %s\nPATH\t: %s", stw.Version, stw.Modified, stw.Frame.Name, stw.Frame.Project, stw.Frame.Path))
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
	stw.hist.SetAttribute("APPEND", str)
	lnum, err := strconv.ParseInt(stw.hist.GetAttribute("LINECOUNT"), 10, 64)
	if err != nil {
		fmt.Println(err)
		return
	}
	setpos := fmt.Sprintf("%d:1", int(lnum)+1)
	stw.hist.SetAttribute("SCROLLTO", setpos)
}

func (stw *Window) errormessage(err error, level uint) {
	if err == nil {
		return
	}
	var otp string
	if level >= ERROR {
		_, file, line, _ := runtime.Caller(1)
		otp = fmt.Sprintf("%s:%d: [%s]: %s", filepath.Base(file), line, LOGLEVEL[level], err.Error())
	} else {
		otp = fmt.Sprintf("[%s]: %s", LOGLEVEL[level], err.Error())
	}
	stw.addHistory(otp)
	logger.Println(otp)
}

func (stw *Window) SetCoord(x, y, z float64) {
	stw.coord.SetAttribute("VALUE", fmt.Sprintf("X: %8.3f Y: %8.3f Z: %8.3f", x, y, z))
}

func (stw *Window) ExecCommand(com *Command) {
	stw.addHistory(com.Name)
	stw.cname.SetAttribute("VALUE", com.Name)
	stw.lastcommand = com
	com.Exec(stw)
}

func (stw *Window) execAliasCommand(al string) {
	if stw.Frame == nil {
		if strings.HasPrefix(al, ":") {
			err := stw.exmode(al)
			if err != nil {
				stw.errormessage(err, ERROR)
			}
		} else {
			stw.Open()
		}
		stw.FocusCanv()
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
			stw.addHistory(fmt.Sprintf("command doesn't exist: %s", al))
		case strings.HasPrefix(al, ":"):
			err := stw.exmode(al)
			if err != nil {
				stw.errormessage(err, ERROR)
			}
		case strings.HasPrefix(al, "'"):
			err := stw.fig2mode(al)
			if err != nil {
				stw.errormessage(err, ERROR)
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
			axisrange(stw, axis, min, max, false)
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
			axisrange(stw, axis, min, 1000.0, false)
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
			axisrange(stw, axis, min, 1000.0, false)
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
			axisrange(stw, axis, -100.0, max, false)
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
			axisrange(stw, axis, -100.0, max, false)
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
			axisrange(stw, axis, val, val, false)
			stw.addHistory(fmt.Sprintf("AxisRange: %s = %.3f", tmp, val))
		}
	}
	stw.Redraw()
	if stw.cline.GetAttribute("VALUE") == "" {
		stw.FocusCanv()
	}
	return
}

func (stw *Window) Complete(str string) string {
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
		str = strings.Replace(str, "%:h", stw.Cwd, 1)
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
		path = filepath.Join(stw.Cwd, path)
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

func exmodecomplete(command string) (string, bool, bool) {
	usage := strings.HasSuffix(command, "?")
	cname := strings.TrimSuffix(command, "?")
	bang := strings.HasSuffix(cname, "!")
	cname = strings.TrimSuffix(cname, "!")
	cname = strings.ToLower(strings.TrimPrefix(cname, ":"))
	var rtn string
	for _, ab := range exabbrev {
		pat := abbrev.MustCompile(ab)
		if pat.MatchString(cname) {
			rtn = pat.Longest()
			break
		}
	}
	if rtn == "" {
		rtn = cname
	}
	return rtn, bang, usage
}

func (stw *Window) emptyexmodech() {
ex_empty:
	for {
		select {
		case <-time.After(time.Second):
			break ex_empty
		case <-stw.exmodeend:
			continue ex_empty
		case <-stw.exmodech:
			continue ex_empty
		}
	}
}

func (stw *Window) exmode(command string) error {
	if len(command) == 1 {
		return st.NotEnoughArgs("exmode")
	}
	if command == ":." {
		return stw.exmode(stw.lastexcommand)
	}
	stw.lastexcommand = command
	tmpargs := strings.Split(command, " ")
	args := make([]string, len(tmpargs))
	argdict := make(map[string]string, 0)
	narg := 0
	for i := 0; i < len(tmpargs); i++ {
		if tmpargs[i] != "" {
			args[narg] = tmpargs[i]
			narg++
		}
	}
	args = args[:narg]
	unnamed := make([]string, narg)
	tmpnarg := 0
	namedarg := regexp.MustCompile("^ *-{1,2}([^-= ]+)(={0,1})([^ =]*) *$")
	for _, a := range args {
		if namedarg.MatchString(a) {
			fs := namedarg.FindStringSubmatch(a)
			if fs[2] == "" {
				argdict[strings.ToUpper(fs[1])] = ""
			} else {
				argdict[strings.ToUpper(fs[1])] = fs[3]
			}
		} else {
			unnamed[tmpnarg] = a
			tmpnarg++
		}
	}
	args = unnamed[:tmpnarg]
	narg = tmpnarg
	var fn string
	if narg < 2 {
		fn = ""
	} else {
		fn = stw.Complete(args[1])
		if filepath.Dir(fn) == "." {
			fn = filepath.Join(stw.Cwd, fn)
		}
	}
	cname, bang, usage := exmodecomplete(args[0])
	evaluated := true
	switch cname {
	default:
		evaluated = false
	case "edit":
		if usage {
			stw.addHistory(":edit [filename]")
			return nil
		}
		if !bang && stw.Changed {
			if stw.Yn("CHANGED", "変更を保存しますか") {
				stw.SaveAS()
			} else {
				return errors.New("not saved")
			}
		}
		if fn != "" {
			if !st.FileExists(fn) {
				sfn, err := stw.SearchFile(args[1])
				if err != nil {
					return err
				}
				err = stw.OpenFile(sfn)
				if err != nil {
					return err
				}
				stw.Redraw()
			} else {
				err := stw.OpenFile(fn)
				if err != nil {
					return err
				}
				stw.Redraw()
			}
		} else {
			stw.Reload()
		}
	case "quit":
		if usage {
			stw.addHistory(":quit")
			return nil
		}
		stw.Close(bang)
	case "eps":
		if usage {
			stw.addHistory(":eps val")
			return nil
		}
		if narg < 2 {
			return st.NotEnoughArgs(":eps")
		}
		val, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return err
		}
		EPS = val
		stw.addHistory(fmt.Sprintf("EPS=%.3E", EPS))
	case "mkdir":
		if usage {
			stw.addHistory(":mkdir dirname")
			return nil
		}
		os.MkdirAll(fn, 0644)
	case "#":
		if usage {
			stw.addHistory(":#")
			return nil
		}
		stw.ShowRecently()
	case "vim":
		if usage {
			stw.addHistory(":vim filename")
			return nil
		}
		stw.Vim(fn)
	case "hkyou":
		if usage {
			stw.addHistory(":hkyou h b tw tf")
			return nil
		}
		if narg < 5 {
			return st.NotEnoughArgs(":hkyou")
		}
		al, err := st.NewHKYOU(args[1:5])
		if err != nil {
			return err
		}
		stw.ShapeData(al)
		go func(a interface{}) {
			stw.exmodech <- a
		}(al)
	case "hweak":
		if usage {
			stw.addHistory(":hweak h b tw tf")
			return nil
		}
		if narg < 5 {
			return st.NotEnoughArgs(":hweak")
		}
		al, err := st.NewHWEAK(args[1:5])
		if err != nil {
			return err
		}
		stw.ShapeData(al)
		go func(a interface{}) {
			stw.exmodech <- a
		}(al)
	case "rpipe":
		if usage {
			stw.addHistory(":rpipe h b tw tf")
			return nil
		}
		if narg < 5 {
			return st.NotEnoughArgs(":rpipe")
		}
		al, err := st.NewRPIPE(args[1:5])
		if err != nil {
			return err
		}
		stw.ShapeData(al)
		go func(a interface{}) {
			stw.exmodech <- a
		}(al)
	case "cpipe":
		if usage {
			stw.addHistory(":cpipe d t")
			return nil
		}
		if narg < 3 {
			return st.NotEnoughArgs(":cpipe")
		}
		al, err := st.NewCPIPE(args[1:3])
		if err != nil {
			return err
		}
		stw.ShapeData(al)
		go func(a interface{}) {
			stw.exmodech <- a
		}(al)
	case "tkyou":
		if usage {
			stw.addHistory(":tkyou h b tw tf")
			return nil
		}
		if narg < 5 {
			return st.NotEnoughArgs(":tkyou")
		}
		al, err := st.NewTKYOU(args[1:5])
		if err != nil {
			return err
		}
		stw.ShapeData(al)
	case "ckyou":
		if usage {
			stw.addHistory(":ckyou h b tw tf")
			return nil
		}
		if narg < 5 {
			return st.NotEnoughArgs(":ckyou")
		}
		al, err := st.NewCKYOU(args[1:5])
		if err != nil {
			return err
		}
		stw.ShapeData(al)
	case "plate":
		if usage {
			stw.addHistory(":plate h b")
			return nil
		}
		if narg < 3 {
			return st.NotEnoughArgs(":plate")
		}
		al, err := st.NewPLATE(args[1:3])
		if err != nil {
			return err
		}
		stw.ShapeData(al)
	case "fixrotate":
		fixRotate = !fixRotate
	case "fixmove":
		fixMove = !fixMove
	case "noundo":
		NOUNDO = true
		stw.addHistory("undo/redo is off")
	case "undo":
		NOUNDO = false
		stw.Snapshot()
		stw.addHistory("undo/redo is on")
	case "alt":
		ALTSELECTNODE = !ALTSELECTNODE
		if ALTSELECTNODE {
			stw.addHistory("select node with Alt key")
		} else {
			stw.addHistory("select elem with Alt key")
		}
	case "procs":
		if usage {
			stw.addHistory(":procs numcpu")
			return nil
		}
		if narg < 2 {
			current := runtime.GOMAXPROCS(-1)
			stw.addHistory(fmt.Sprintf("PROCS: %d", current))
			return nil
		}
		tmp, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return err
		}
		val := int(tmp)
		if 1 <= val && val <= runtime.NumCPU() {
			old := runtime.GOMAXPROCS(val)
			stw.addHistory(fmt.Sprintf("PROCS: %d -> %d", old, val))
		}
	case "empty":
		if usage {
			stw.addHistory(":empty")
			return nil
		}
		stw.emptyexmodech()
	}
	if evaluated {
		return nil
	}
	if stw.Frame == nil {
		stw.errormessage(errors.New("frame is nil"), INFO)
		return nil
	}
	switch cname {
	default:
		stw.errormessage(errors.New(fmt.Sprintf("no exmode command: %s", cname)), INFO)
		return nil
	case "write":
		if usage {
			stw.addHistory(":write")
			return nil
		}
		if fn == "" {
			stw.SaveFile(stw.Frame.Path)
		} else {
			if bang || (!st.FileExists(fn) || stw.Yn("Save", "上書きしますか")) {
				err := stw.SaveFile(fn)
				if err != nil {
					return err
				}
				if fn != stw.Frame.Path {
					stw.Copylsts(fn)
				}
			}
		}
	case "save":
		if usage {
			stw.addHistory(":save filename")
			return nil
		}
		if fn == "" {
			return st.NotEnoughArgs(":save")
		} else {
			if bang || (!st.FileExists(fn) || stw.Yn("Save", "上書きしますか")) {
				if _, ok := argdict["MKDIR"]; ok {
					os.MkdirAll(filepath.Dir(fn), 0644)
				}
				var err error
				if stw.SelectElem != nil && len(stw.SelectElem) > 0 {
					err = stw.SaveFileSelected(fn, stw.SelectElem)
					if err != nil {
						return err
					}
					stw.Deselect()
					err = stw.OpenFile(fn)
					if err != nil {
						return err
					}
					stw.Copylsts(fn)
				} else {
					err = stw.SaveFile(fn)
				}
				if err != nil {
					return err
				}
				if fn != stw.Frame.Path {
					stw.Copylsts(fn)
				}
				stw.Rebase(fn)
			}
		}
	case "increment":
		if usage {
			stw.addHistory(":increment {times:1}")
			return nil
		}
		if !bang && stw.Changed {
			switch stw.Yna("CHANGED", "変更を保存しますか", "キャンセル") {
			case 1:
				stw.SaveAS()
			case 2:
				stw.addHistory("not saved")
			case 3:
				return errors.New(":inc cancelled")
			}
		}
		var times int
		if narg >= 2 {
			val, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}
			times = int(val)
		} else {
			times = 1
		}
		fn, err := st.Increment(stw.Frame.Path, "_", 1, times)
		if err != nil {
			return err
		}
		if bang || (!st.FileExists(fn) || stw.Yn("Save", "上書きしますか")) {
			err := stw.SaveFile(fn)
			if err != nil {
				return err
			}
			if fn != stw.Frame.Path {
				stw.Copylsts(fn)
			}
			stw.Rebase(fn)
			stw.EditReadme(filepath.Dir(fn))
		}
	case "read":
		if usage {
			stw.addHistory(":read filename")
			stw.addHistory(":read type filename")
			return nil
		}
		if narg < 2 {
			return st.NotEnoughArgs(":read")
		}
		t := strings.ToLower(args[1])
		if narg < 3 {
			switch t {
			case "$all":
				stw.ReadAll()
			case "$data":
				for _, ext := range []string{".inl", ".ihx", ".ihy"} {
					err := stw.Frame.ReadData(st.Ce(stw.Frame.Path, ext))
					if err != nil {
						stw.errormessage(err, ERROR)
					}
				}
			case "$results":
				mode := st.UPDATE_RESULT
				if _, ok := argdict["ADD"]; ok {
					mode = st.ADD_RESULT
					if _, ok2 := argdict["SEARCH"]; ok2 {
						mode = st.ADDSEARCH_RESULT
					}
				}
				for _, ext := range []string{".otl", ".ohx", ".ohy"} {
					err := stw.Frame.ReadResult(st.Ce(stw.Frame.Path, ext), uint(mode))
					if err != nil {
						stw.errormessage(err, ERROR)
					}
				}
			default:
				err := stw.ReadFile(fn)
				if err != nil {
					return err
				}
			}
			return nil
		}
		fn = stw.Complete(args[2])
		if filepath.Dir(fn) == "." {
			fn = filepath.Join(stw.Cwd, fn)
		}
		switch {
		case abbrev.For("d/ata", t):
			err := stw.Frame.ReadData(fn)
			if err != nil {
				return err
			}
		case abbrev.For("r/esult", t):
			mode := st.UPDATE_RESULT
			if _, ok := argdict["ADD"]; ok {
				mode = st.ADD_RESULT
				if _, ok2 := argdict["SEARCH"]; ok2 {
					mode = st.ADDSEARCH_RESULT
				}
			}
			err := stw.Frame.ReadResult(fn, uint(mode))
			if err != nil {
				return err
			}
		case abbrev.For("s/rcan", t):
			err := stw.Frame.ReadRat(fn)
			if err != nil {
				return err
			}
		case abbrev.For("l/ist", t):
			err := stw.Frame.ReadLst(fn)
			if err != nil {
				return err
			}
		case abbrev.For("w/eight", t):
			err := stw.Frame.ReadWgt(fn)
			if err != nil {
				return err
			}
		case abbrev.For("k/ijun", t):
			err := stw.Frame.ReadKjn(fn)
			if err != nil {
				return err
			}
		case abbrev.For("b/uckling", t):
			err := stw.Frame.ReadBuckling(fn)
			if err != nil {
				return err
			}
		case abbrev.For("z/oubun", t):
			err := stw.Frame.ReadZoubun(fn)
			if err != nil {
				return err
			}
		case t == "pgp":
			al := make(map[string]*Command, 0)
			err := ReadPgp(fn, al)
			if err != nil {
				return err
			}
			aliases = al
		}
	case "insert":
		if usage {
			stw.addHistory(":insert filename angle(deg)")
			return nil
		}
		if narg > 2 && len(stw.SelectNode) >= 1 {
			angle, err := strconv.ParseFloat(args[2], 64)
			if err != nil {
				return err
			}
			err = stw.Frame.ReadInp(fn, stw.SelectNode[0].Coord, angle*math.Pi/180.0, false)
			stw.Snapshot()
			if err != nil {
				return err
			}
			stw.EscapeAll()
		}
	case "propsect":
		if usage {
			stw.addHistory(":propsect filename")
			return nil
		}
		err := stw.AddPropAndSect(fn)
		stw.Snapshot()
		if err != nil {
			return err
		}
	case "writeoutput":
		if usage {
			stw.addHistory(":writeoutput filename period")
			return nil
		}
		if narg < 3 {
			return st.NotEnoughArgs(":wo")
		}
		var err error
		period := strings.ToUpper(args[2])
		if stw.SelectElem != nil && len(stw.SelectElem) > 0 {
			err = st.WriteOutput(fn, period, stw.SelectElem)
		} else {
			err = stw.Frame.WriteOutput(fn, period)
		}
		if err != nil {
			return err
		}
	case "writereaction":
		if usage {
			stw.addHistory(":writereaction filename direction")
			return nil
		}
		if narg < 3 {
			return st.NotEnoughArgs(":wr")
		}
		tmp, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			return err
		}
		if _, ok := argdict["CONFED"]; ok {
			selectconfed(stw)
		}
		if stw.SelectNode == nil || len(stw.SelectNode) == 0 {
			return errors.New(":writereaction: no selected node")
		}
		sort.Sort(st.NodeByNum{stw.SelectNode})
		err = st.WriteReaction(fn, stw.SelectNode, int(tmp))
		if err != nil {
			return err
		}
	case "zoubundisp":
		if usage {
			stw.addHistory(":zoubundisp period direction")
			return nil
		}
		if narg < 3 {
			return st.NotEnoughArgs(":zoubundisp")
		}
		if stw.SelectNode == nil || len(stw.SelectNode) == 0 {
			return st.NotEnoughArgs(":zoubundisp no selected node")
		}
		pers := []string{args[1]}
		val, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			return errors.New(":zoubundisp unknown direction")
		}
		d := int(val)
		if d < 0 || d > 5 {
			return errors.New(":zoubundisp direction should be between 0 ~ 6")
		}
		fn := filepath.Join(filepath.Dir(stw.Frame.Path), "zoubunout.txt")
		err = stw.Frame.ReportZoubunDisp(fn, stw.SelectNode, pers, d)
		if err != nil {
			return err
		}
	case "zoubunreaction":
		if usage {
			stw.addHistory(":zoubunreaction period direction")
			return nil
		}
		if narg < 3 {
			return st.NotEnoughArgs(":zoubunreaction")
		}
		if stw.SelectNode == nil || len(stw.SelectNode) == 0 {
			return st.NotEnoughArgs(":zoubunreaction no selected node")
		}
		pers := []string{args[1]}
		val, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			return errors.New(":zoubunreaction unknown direction")
		}
		d := int(val)
		if d < 0 || d > 5 {
			return errors.New(":zoubunreaction direction should be between 0 ~ 6")
		}
		fn := filepath.Join(filepath.Dir(stw.Frame.Path), "zoubunout.txt")
		err = stw.Frame.ReportZoubunReaction(fn, stw.SelectNode, pers, d)
		if err != nil {
			return err
		}
	case "fig2":
		if usage {
			stw.addHistory(":fig2 filename")
			return nil
		}
		err := stw.ReadFig2(fn)
		if err != nil {
			return err
		}
	case "svg":
		if usage {
			stw.addHistory(":svg filename")
			return nil
		}
		err := stw.PrintSVG(fn)
		if err != nil {
			return err
		}
	case "check":
		if usage {
			stw.addHistory(":check")
			return nil
		}
		checkframe(stw)
		stw.addHistory("CHECKED")
	case "elemduplication":
		if usage {
			stw.addHistory(":elemduplication {-ignoresect=code}")
			return nil
		}
		stw.Deselect()
		var isect []int
		if isec, ok := argdict["IGNORESECT"]; ok {
			if isec == "" {
				isect = nil
			} else {
				stw.addHistory(fmt.Sprintf("IGNORE SECT: %s", isec))
				isect = SplitNums(isec)
			}
		} else {
			isect = nil
		}
		els := stw.Frame.ElemDuplication(isect)
		if len(els) != 0 {
			enum := 0
			for k := range els {
				stw.SelectElem = append(stw.SelectElem, k)
				enum++
			}
			stw.SelectElem = stw.SelectElem[:enum]
		}
	case "intersectall":
		l := len(stw.SelectElem)
		if l <= 1 {
			return nil
		}
		go func() {
			err := stw.Frame.IntersectAll(stw.SelectElem, EPS)
			stw.Frame.Endch <- err
		}()
		stw.CurrentLap("Calculating...", 0, l)
		go func() {
			var err error
			var nlap int
		iallloop:
			for {
				select {
				case nlap = <-stw.Frame.Lapch:
					stw.CurrentLap("Calculating...", nlap, l)
					stw.Redraw()
				case err = <-stw.Frame.Endch:
					if err != nil {
						stw.CurrentLap("Error", nlap, l)
						stw.errormessage(err, ERROR)
					} else {
						stw.CurrentLap("Completed", nlap, l)
					}
					stw.Redraw()
					break iallloop
				}
			}
		}()
		stw.Snapshot()
	case "srcal":
		if usage {
			stw.addHistory(":srcal")
			return nil
		}
		cond := st.NewCondition()
		if _, ok := argdict["FBOLD"]; ok {
			stw.addHistory("Fb: old")
			cond.FbOld = true
		}
		if _, ok := argdict["RELOAD"]; ok {
			stw.ReadFile(st.Ce(stw.Frame.Path, ".lst"))
		}
		stw.Frame.SectionRateCalculation("L", "X", "X", "Y", "Y", -1.0, cond)
	case "nminteraction":
		if usage {
			stw.addHistory(":nminteraction sectcode")
			return nil
		}
		if narg < 2 {
			return st.NotEnoughArgs(":nminteraction")
		}
		tmp, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return err
		}
		if al, ok := stw.Frame.Allows[int(tmp)]; ok {
			var otp bytes.Buffer
			cond := st.NewCondition()
			ndiv := 100
			if nd, ok := argdict["NDIV"]; ok {
				if nd != "" {
					stw.addHistory(fmt.Sprintf("NDIV: %s", nd))
					tmp, err := strconv.ParseInt(nd, 10, 64)
					if err == nil {
						ndiv = int(tmp)
					}
				}
			}
			filename := "nmi.txt"
			if o, ok := argdict["OUTPUT"]; ok {
				if o != "" {
					stw.addHistory(fmt.Sprintf("OUTPUT: %s", o))
					filename = o
				}
			}
			switch al.(type) {
			default:
				return nil
			case *st.RCColumn:
				nmax := al.(*st.RCColumn).Nmax(cond)
				nmin := al.(*st.RCColumn).Nmin(cond)
				for i := 0; i <= ndiv; i++ {
					cond.N = nmax - float64(i)*(nmax-nmin)/float64(ndiv)
					otp.WriteString(fmt.Sprintf("%.5f %.5f\n", cond.N, al.Ma(cond)))
				}
			}
			w, err := os.Create(filename)
			defer w.Close()
			if err != nil {
				return err
			}
			otp.WriteTo(w)
		}
	case "gohanlst":
		if usage {
			stw.addHistory(":gohanlst factor sectcode...")
			return nil
		}
		if narg < 3 {
			return st.NotEnoughArgs(":gohanlst")
		}
		val, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return err
		}
		sects := SplitNums(strings.Join(args[2:], " "))
		var otp bytes.Buffer
		var etype string
		for _, snum := range sects {
			if sec, ok := stw.Frame.Sects[snum]; ok {
				for _, s := range sec.BraceSection() {
					if s.Type == 5 {
						etype = "WALL"
					} else if s.Type == 6 {
						etype = "SLAB"
					}
					otp.WriteString(fmt.Sprintf("CODE %4d WOOD %s                                                \"%s(%3d)\"\n", s.Num, etype, etype[:1], snum))
					otp.WriteString(fmt.Sprintf("         THICK %5.3f       GOHAN                                     \"x%3.1f\"\n\n", val/12.0, val)) // 2[kgf/cm] / 24[kgf/cm2] = 1/12[cm]
				}
			}
		}
		w, err := os.Create(filepath.Join(stw.Cwd, "gohan.lst"))
		defer w.Close()
		if err != nil {
			return err
		}
		otp = st.AddCR(otp)
		otp.WriteTo(w)
	case "kaberyo":
		if usage {
			stw.addHistory(":kaberyo")
			return nil
		}
		var els []*st.Elem
		if stw.SelectElem == nil || len(stw.SelectElem) == 0 {
			enum := 0
			els = make([]*st.Elem, 0)
		ex_kaberyo:
			for {
				select {
				case <-time.After(time.Second):
					break ex_kaberyo
				case <-stw.exmodeend:
					break ex_kaberyo
				case el := <-stw.exmodech:
					if el, ok := el.(*st.Elem); ok {
						els = append(els, el)
						enum++
					}
				}
			}
			if enum == 0 {
				return errors.New(":kaberyo no selected elem")
			}
			els = els[:enum]
		} else {
			els = stw.SelectElem
		}
		var props []int
		if val, ok := argdict["HALF"]; ok {
			props = SplitNums(val)
			stw.addHistory(fmt.Sprintf("HALF: %s", val))
		}
		alpha := math.Sqrt(24.0 / 18.0)
		if val, ok := argdict["FC"]; ok {
			fc, err := strconv.ParseFloat(val, 64)
			if err == nil {
				alpha = math.Min(math.Sqrt2, math.Sqrt(fc/18.0))
			}
		}
		if val, ok := argdict["ALPHA"]; ok {
			a, err := strconv.ParseFloat(val, 64)
			if err == nil {
				alpha = math.Min(math.Sqrt2, a)
			}
		}
		stw.addHistory(fmt.Sprintf("ALPHA: %.3f", alpha))
		ccol := 7.0
		cwall := 25.0
		if r, ok := argdict["ROUTE"]; ok {
			switch strings.ToLower(r) {
			case "1", "2-1", "2_1", "2.1":
				ccol = 7.0
				cwall = 25.0
			case "2-2", "2_2", "2.2":
				ccol = 18.0
				cwall = 18.0
			}
		}
		stw.addHistory(fmt.Sprintf("COEFFICIENT: COLUMN=%.1f WALL=%.1f", ccol, cwall))
		sumcol := 0.0
		sumwall := 0.0
		for _, el := range els {
			if !el.Sect.IsRc(EPS) {
				continue
			}
			if el.IsLineElem() {
				a, err := el.Sect.Area(0)
				if err != nil {
					continue
				}
				for _, p := range props {
					if el.Sect.Figs[0].Prop.Num == p {
						a *= 0.5
						break
					}
				}
				sumcol += a
			} else {
				t, err := el.Sect.Thick(0)
				if err != nil {
					continue
				}
				sumwall += t * el.EffectiveWidth()
			}
		}
		total := alpha * (ccol*sumcol + cwall*sumwall)
		stw.addHistory(fmt.Sprintf("COLUMN: %.3f WALL: %.3f TOTAL: %.3f", sumcol, sumwall, total))
	case "facts":
		if usage {
			stw.addHistory(":facts {-skipany=code} {-skipall=code}")
			return nil
		}
		fn = st.Ce(stw.Frame.Path, ".fes")
		var skipany, skipall []int
		if sany, ok := argdict["SKIPANY"]; ok {
			if sany == "" {
				skipany = nil
			} else {
				stw.addHistory(fmt.Sprintf("SKIP ANY: %s", sany))
				skipany = SplitNums(sany)
			}
		} else {
			skipany = nil
		}
		if sall, ok := argdict["SKIPALL"]; ok {
			if sall == "" {
				skipall = nil
			} else {
				stw.addHistory(fmt.Sprintf("SKIP ALL: %s", sall))
				skipall = SplitNums(sall)
			}
		} else {
			skipall = nil
		}
		err := stw.Frame.Facts(fn, []int{st.COLUMN, st.GIRDER, st.BRACE, st.WBRACE, st.SBRACE}, skipany, skipall)
		if err != nil {
			return err
		}
		stw.addHistory(fmt.Sprintf("Output: %s", fn))
	case "amountprop":
		if usage {
			stw.addHistory(":amountprop propcode")
			return nil
		}
		if narg < 2 {
			return st.NotEnoughArgs(":amountprop")
		}
		props := SplitNums(strings.Join(args[1:], " "))
		if len(props) == 0 {
			return errors.New(":amountprop: no selected prop")
		}
		fn := filepath.Join(filepath.Dir(stw.Frame.Path), "amount.txt")
		err := stw.Frame.AmountProp(fn, props...)
		if err != nil {
			return err
		}
	case "amountlst":
		if usage {
			stw.addHistory(":amountlst propcode")
			return nil
		}
		var sects []int
		if narg < 2 {
			if _, ok := argdict["ALL"]; ok {
				sects = make([]int, len(stw.Frame.Sects))
				i := 0
				for _, sec := range stw.Frame.Sects {
					if sec.Num >= 900 {
						continue
					}
					sects[i] = sec.Num
					i++
				}
				sects = sects[:i]
				sort.Ints(sects)
			} else {
				return st.NotEnoughArgs(":amountlst")
			}
		}
		if sects == nil {
			sects = SplitNums(strings.Join(args[1:], " "))
		}
		if len(sects) == 0 {
			return errors.New(":amountlst: no selected sect")
		}
		fn := filepath.Join(filepath.Dir(stw.Frame.Path), "amountlst.txt")
		err := stw.Frame.AmountLst(fn, sects...)
		if err != nil {
			return err
		}
	case "node":
		if usage {
			stw.addHistory(":node nnum")
			stw.addHistory(":node [x,y,z] [>,<,=] coord")
			stw.addHistory(":node {confed/pinned/fixed/free}")
			stw.addHistory(":node pile num")
			return nil
		}
		stw.Deselect()
		var f func(*st.Node) bool
		if narg >= 2 {
			condition := strings.ToUpper(strings.Join(args[1:], " "))
			coordstr := regexp.MustCompile("^ *([XYZ]) *([<!=>]{0,2}) *([0-9.]+)")
			numstr := regexp.MustCompile("^[0-9, ]+$")
			pilestr := regexp.MustCompile("^ *PILE *([0-9, ]+)$")
			switch {
			default:
				return errors.New(":node: unknown format")
			case numstr.MatchString(condition):
				nnums := SplitNums(condition)
				stw.SelectNode = make([]*st.Node, len(nnums))
				nods := 0
				for i, nnum := range nnums {
					if n, ok := stw.Frame.Nodes[nnum]; ok {
						stw.SelectNode[i] = n
						nods++
					}
				}
				stw.SelectNode = stw.SelectNode[:nods]
			case coordstr.MatchString(condition):
				fs := coordstr.FindStringSubmatch(condition)
				if len(fs) < 4 {
					return errors.New(":node invalid input")
				}
				var ind int
				switch fs[1] {
				case "X":
					ind = 0
				case "Y":
					ind = 1
				case "Z":
					ind = 2
				}
				val, err := strconv.ParseFloat(fs[3], 64)
				if err != nil {
					return err
				}
				switch fs[2] {
				case "", "=", "==":
					f = func(n *st.Node) bool {
						if n.Coord[ind] == val {
							return true
						}
						return false
					}
				case "!=":
					f = func(n *st.Node) bool {
						if n.Coord[ind] != val {
							return true
						}
						return false
					}
				case ">":
					f = func(n *st.Node) bool {
						if n.Coord[ind] > val {
							return true
						}
						return false
					}
				case ">=":
					f = func(n *st.Node) bool {
						if n.Coord[ind] >= val {
							return true
						}
						return false
					}
				case "<":
					f = func(n *st.Node) bool {
						if n.Coord[ind] < val {
							return true
						}
						return false
					}
				case "<=":
					f = func(n *st.Node) bool {
						if n.Coord[ind] <= val {
							return true
						}
						return false
					}
				}
			case pilestr.MatchString(condition):
				fs := pilestr.FindStringSubmatch(condition)
				pnums := SplitNums(fs[1])
				f = func(n *st.Node) bool {
					if n.Pile == nil {
						return false
					}
					for _, pnum := range pnums {
						if n.Pile.Num == pnum {
							return true
						}
					}
					return false
				}
			case abbrev.For("CONF/ED", condition):
				f = func(n *st.Node) bool {
					for i := 0; i < 6; i++ {
						if n.Conf[i] {
							return true
						}
					}
					return false
				}
			case condition == "FREE":
				f = func(n *st.Node) bool {
					for i := 0; i < 6; i++ {
						if n.Conf[i] {
							return false
						}
					}
					return true
				}
			case abbrev.For("PIN/NED", condition):
				f = func(n *st.Node) bool {
					return n.IsPinned()
				}
			case abbrev.For("FIX/ED", condition):
				f = func(n *st.Node) bool {
					return n.IsFixed()
				}
			}
			if f != nil {
				stw.SelectNode = make([]*st.Node, len(stw.Frame.Nodes))
				num := 0
				for _, n := range stw.Frame.Nodes {
					if f(n) {
						stw.SelectNode[num] = n
						num++
					}
				}
				stw.SelectNode = stw.SelectNode[:num]
			}
		} else {
			stw.SelectNode = make([]*st.Node, len(stw.Frame.Nodes))
			num := 0
			for _, n := range stw.Frame.Nodes {
				stw.SelectNode[num] = n
				num++
			}
			stw.SelectNode = stw.SelectNode[:num]
		}
		go func(ns []*st.Node) {
			for _, n := range ns {
				stw.exmodech <- n
			}
			stw.exmodeend <- 1
		}(stw.SelectNode)
	case "conf":
		if usage {
			stw.addHistory(":conf [0,1]{6}")
			return nil
		}
		lis := make([]bool, 6)
		if len(args[1]) >= 6 {
			for i := 0; i < 6; i++ {
				switch args[1][i] {
				default:
					lis[i] = false
				case '0':
					lis[i] = false
				case '1':
					lis[i] = true
				case '_':
					continue
				case 't':
					lis[i] = !lis[i]
				}
			}
			setconf(stw, lis)
		} else {
			return st.NotEnoughArgs(":conf")
		}
	case "pile":
		if usage {
			stw.addHistory(":pile pilecode")
			return nil
		}
		if stw.SelectNode == nil || len(stw.SelectNode) == 0 {
			return errors.New(":pile no selected node")
		}
		if narg < 2 {
			for _, n := range stw.SelectNode {
				n.Pile = nil
			}
			break
		}
		val, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return err
		}
		if p, ok := stw.Frame.Piles[int(val)]; ok {
			for _, n := range stw.SelectNode {
				n.Pile = p
			}
			stw.Snapshot()
		} else {
			return errors.New(fmt.Sprintf(":pile PILE %d doesn't exist", val))
		}
	case "xscale":
		if usage {
			stw.addHistory(":xscale factor coord")
			return nil
		}
		if stw.SelectNode == nil || len(stw.SelectNode) == 0 {
			return errors.New(":xscale no selected node")
		}
		if narg < 3 {
			return st.NotEnoughArgs(":xscale")
		}
		factor, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return err
		}
		coord, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			return err
		}
		for _, n := range stw.SelectNode {
			if n == nil {
				continue
			}
			n.Scale([]float64{coord, 0.0, 0.0}, factor, 1.0, 1.0)
		}
		stw.Snapshot()
	case "yscale":
		if usage {
			stw.addHistory(":yscale factor coord")
			return nil
		}
		if stw.SelectNode == nil || len(stw.SelectNode) == 0 {
			return errors.New(":yscale no selected node")
		}
		if narg < 3 {
			return st.NotEnoughArgs(":yscale")
		}
		factor, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return err
		}
		coord, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			return err
		}
		for _, n := range stw.SelectNode {
			if n == nil {
				continue
			}
			n.Scale([]float64{0.0, coord, 0.0}, 1.0, factor, 1.0)
		}
		stw.Snapshot()
	case "zscale":
		if usage {
			stw.addHistory(":zscale factor coord")
			return nil
		}
		if stw.SelectNode == nil || len(stw.SelectNode) == 0 {
			return errors.New(":zscale no selected node")
		}
		if narg < 3 {
			return st.NotEnoughArgs(":zscale")
		}
		factor, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return err
		}
		coord, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			return err
		}
		for _, n := range stw.SelectNode {
			if n == nil {
				continue
			}
			n.Scale([]float64{0.0, 0.0, coord}, 1.0, 1.0, factor)
		}
		stw.Snapshot()
	case "pload":
		if usage {
			stw.addHistory(":pload position value")
			return nil
		}
		if stw.SelectNode == nil || len(stw.SelectNode) == 0 {
			return errors.New(":pload no selected node")
		}
		if narg < 3 {
			return st.NotEnoughArgs(":pload")
		}
		ind, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return err
		}
		val, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			return err
		}
		for _, n := range stw.SelectNode {
			if n == nil {
				continue
			}
			n.Load[int(ind)] = val
		}
		stw.Snapshot()
	case "elem":
		if usage {
			stw.addHistory(":elem [elemcode,sect sectcode,etype]")
			return nil
		}
		stw.Deselect()
		var f func(*st.Elem) bool
		if narg >= 2 {
			condition := strings.ToUpper(strings.Join(args[1:], " "))
			numstr := regexp.MustCompile("^[0-9, ]+$")
			switch {
			default:
				return errors.New(":elem: unknown format")
			case numstr.MatchString(condition):
				enums := SplitNums(condition)
				stw.SelectElem = make([]*st.Elem, len(enums))
				els := 0
				for i, enum := range enums {
					if el, ok := stw.Frame.Elems[enum]; ok {
						stw.SelectElem[i] = el
						els++
					}
				}
				stw.SelectElem = stw.SelectElem[:els]
			case re_sectnum.MatchString(condition):
				f, _ = SectFilter(condition)
				if f == nil {
					return errors.New(":elem sect: format error")
				}
			case re_orgsectnum.MatchString(condition):
				f, _ = OriginalSectFilter(condition)
				if f == nil {
					return errors.New(":elem sect: format error")
				}
			case re_etype.MatchString(condition):
				f, _ = EtypeFilter(condition)
				if f == nil {
					return errors.New(":elem etype: format error")
				}
			case strings.EqualFold(condition, "curtain"):
				f = func(el *st.Elem) bool {
					if el.Sect.Num > 900 {
						return false
					}
					if el.Sect.HasArea(0) {
						return false
					}
					if !el.Sect.HasBrace() {
						return true
					}
					return false
				}
			case strings.EqualFold(condition, "isgohan"):
				f = func(el *st.Elem) bool {
					return el.Sect.IsGohan(EPS)
				}
			case strings.EqualFold(condition, "error"):
				threshold := 1.0
				if v, ok := argdict["THRESHOLD"]; ok {
					val, err := strconv.ParseFloat(v, 64)
					if err == nil {
						threshold = val
					}
				}
				stw.addHistory(fmt.Sprintf("THRESHOLD: %.3f", threshold))
				f = func(el *st.Elem) bool {
					switch el.Etype {
					case st.COLUMN, st.GIRDER, st.BRACE, st.WALL, st.SLAB:
						val, err := el.RateMax(stw.Frame.Show)
						if err != nil {
							return false
						}
						if val > threshold {
							return true
						}
					}
					return false
				}
			}
			if f != nil {
				stw.SelectElem = make([]*st.Elem, len(stw.Frame.Elems))
				num := 0
				for _, el := range stw.Frame.Elems {
					if f(el) {
						stw.SelectElem[num] = el
						num++
					}
				}
				stw.SelectElem = stw.SelectElem[:num]
			}
		} else {
			stw.SelectElem = make([]*st.Elem, len(stw.Frame.Elems))
			num := 0
			for _, el := range stw.Frame.Elems {
				stw.SelectElem[num] = el
				num++
			}
			stw.SelectElem = stw.SelectElem[:num]
		}
		go func(els []*st.Elem) {
			for _, el := range els {
				stw.exmodech <- el
			}
			stw.exmodeend <- 1
		}(stw.SelectElem)
	case "fence":
		if usage {
			stw.addHistory(":fence axis coord")
			return nil
		}
		if narg < 3 {
			return st.NotEnoughArgs(":fence")
		}
		var axis int
		var err error
		var val float64
		switch strings.ToUpper(args[1]) {
		default:
			return errors.New(":fence unknown direction")
		case "X":
			axis = 0
			val, err = strconv.ParseFloat(args[2], 64)
			if err != nil {
				return err
			}
		case "Y":
			axis = 1
			val, err = strconv.ParseFloat(args[2], 64)
			if err != nil {
				return err
			}
		case "Z":
			axis = 2
			val, err = strconv.ParseFloat(args[2], 64)
			if err != nil {
				return err
			}
		case "HEIGHT":
			axis = 2
			ind, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return err
			}
			if int(ind) > stw.Frame.Ai.Nfloor-1 {
				return errors.New(":fence height: index error")
			}
			val = stw.Frame.Ai.Boundary[int(ind)]
		}
		stw.SelectElem = stw.Frame.Fence(axis, val, false)
		go func(els []*st.Elem) {
			for _, el := range els {
				stw.exmodech <- el
			}
			stw.exmodeend <- 1
		}(stw.SelectElem)
	case "filter":
		if usage {
			stw.addHistory(":filter condition")
			return nil
		}
		tmpels, err := stw.FilterElem(stw.SelectElem, strings.Join(args[1:], " "))
		if err != nil {
			return err
		}
		stw.SelectElem = tmpels
		go func(els []*st.Elem) {
			for _, el := range els {
				stw.exmodech <- el
			}
			stw.exmodeend <- 1
		}(stw.SelectElem)
	case "bond":
		if usage {
			stw.addHistory(":bond [pin,rigid] [upper,lower,sect sectcode]")
			return nil
		}
		if narg < 2 {
			return st.NotEnoughArgs(":bond")
		}
		var els []*st.Elem
		if stw.SelectElem == nil || len(stw.SelectElem) == 0 {
			enum := 0
			els = make([]*st.Elem, 0)
		ex_bond:
			for {
				select {
				case <-time.After(time.Second):
					break ex_bond
				case <-stw.exmodeend:
					break ex_bond
				case el := <-stw.exmodech:
					if el, ok := el.(*st.Elem); ok {
						els = append(els, el)
						enum++
					}
				}
			}
			if enum == 0 {
				return errors.New(":bond no selected elem")
			}
			els = els[:enum]
		} else {
			els = stw.SelectElem
		}
		lis := make([]bool, 6)
		switch strings.ToUpper(args[1]) {
		case "PIN":
			lis[4] = true
			lis[5] = true
		}
		f := func(el *st.Elem, ind int) bool {
			return true
		}
		if narg >= 3 {
			condition := strings.ToLower(strings.Join(args[2:], " "))
			switch {
			case abbrev.For("up/per", condition):
				f = func(el *st.Elem, ind int) bool {
					return el.Enod[ind].Coord[2] > el.Enod[1-ind].Coord[2]
				}
			case abbrev.For("lo/wer", condition):
				f = func(el *st.Elem, ind int) bool {
					return el.Enod[ind].Coord[2] < el.Enod[1-ind].Coord[2]
				}
			case re_sectnum.MatchString(condition):
				tmpf, _ := SectFilter(condition)
				f = func(el *st.Elem, ind int) bool {
					for _, sel := range el.Frame.SearchElem(el.Enod[ind]) {
						if sel.Num == el.Num {
							continue
						}
						if tmpf(sel) {
							return true
						}
					}
					return false
				}
			}
		}
		for _, el := range els {
			if !el.IsLineElem() {
				continue
			}
			for i := 0; i < 2; i++ {
				if !f(el, i) {
					continue
				}
				for j := 0; j < 6; j++ {
					el.Bonds[6*i+j] = lis[j]
				}
			}
		}
		stw.Snapshot()
	case "section+":
		if usage {
			stw.addHistory(":section+ value")
			return nil
		}
		if narg < 2 {
			return st.NotEnoughArgs(":section+")
		}
		tmp, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return err
		}
		if tmp == 0 {
			break
		}
		val := int(tmp)
		for _, el := range stw.SelectElem {
			if el == nil {
				continue
			}
			if sec, ok := stw.Frame.Sects[el.Sect.Num+val]; ok {
				el.Sect = sec
			}
		}
		stw.Snapshot()
	case "axis2cang":
		if usage {
			stw.addHistory(":axis2cang n1 n2 [strong,weak]")
			return nil
		}
		if narg < 4 {
			return st.NotEnoughArgs(":axis2cang")
		}
		if stw.SelectElem == nil || len(stw.SelectElem) == 0 {
			return errors.New(":axis2cang no selected elem")
		}
		nnum1, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return err
		}
		nnum2, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			return err
		}
		var strong bool
		if strings.EqualFold(args[3], "strong") {
			strong = true
		} else if strings.EqualFold(args[3], "weak") {
			strong = false
		} else {
			return errors.New(":axis2cang: last argument must be strong or weak")
		}
		var n1, n2 *st.Node
		var found bool
		if n1, found = stw.Frame.Nodes[int(nnum1)]; !found {
			return errors.New(fmt.Sprintf(":axis2cang: NODE %d not found", nnum1))
		}
		if n2, found = stw.Frame.Nodes[int(nnum2)]; !found {
			return errors.New(fmt.Sprintf(":axis2cang: NODE %d not found", nnum2))
		}
		vec := []float64{n2.Coord[0] - n1.Coord[0], n2.Coord[1] - n1.Coord[1], n2.Coord[2] - n1.Coord[2]}
		for _, el := range stw.SelectElem {
			if el == nil || el.IsHidden(stw.Frame.Show) || el.Lock || !el.IsLineElem() {
				continue
			}
			_, err := el.AxisToCang(vec, strong)
			if err != nil {
				return err
			}
		}
		stw.Snapshot()
	case "resultant":
		if stw.SelectElem == nil || len(stw.SelectElem) == 0 {
			return errors.New(":resultant no selected elem")
		}
		vec := make([]float64, 3)
		elems := make([]*st.Elem, len(stw.SelectElem))
		enum := 0
		for _, el := range stw.SelectElem {
			if el == nil || el.Lock || !el.IsLineElem() {
				continue
			}
			elems[enum] = el
			enum++
		}
		elems = elems[:enum]
		en, err := st.CommonEnod(elems...)
		if err != nil {
			return err
		}
		if en == nil || len(en) == 0 {
			return errors.New(":resultant no common enod")
		}
		axis := [][]float64{st.XAXIS, st.YAXIS, st.ZAXIS}
		per := stw.Frame.Show.Period
		for _, el := range elems {
			for i := 0; i < 3; i++ {
				vec[i] += el.VectorStress(per, en[0].Num, axis[i])
			}
		}
		v := 0.0
		for i := 0; i < 3; i++ {
			v += vec[i] * vec[i]
		}
		v = math.Sqrt(v)
		stw.addHistory(fmt.Sprintf("NODE: %d", en[0].Num))
		stw.addHistory(fmt.Sprintf("X: %.3f Y: %.3f Z: %.3f F: %.3f", vec[0], vec[1], vec[2], v))
	case "prestress":
		if usage {
			stw.addHistory(":prestress value")
			return nil
		}
		if narg < 2 {
			return st.NotEnoughArgs(":prestress")
		}
		var els []*st.Elem
		if stw.SelectElem == nil || len(stw.SelectElem) == 0 {
			enum := 0
			els = make([]*st.Elem, 0)
		ex_prestress:
			for {
				select {
				case <-time.After(time.Second):
					break ex_prestress
				case <-stw.exmodeend:
					break ex_prestress
				case el := <-stw.exmodech:
					if el == nil {
						break ex_prestress
					}
					if el, ok := el.(*st.Elem); ok {
						els = append(els, el)
						enum++
					}
				}
			}
			if enum == 0 {
				return errors.New(":prestress no selected elem")
			}
			els = els[:enum]
		} else {
			els = stw.SelectElem
		}
		val, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return err
		}
		for _, el := range els {
			if el == nil || el.Lock || !el.IsLineElem() {
				continue
			}
			el.Prestress = val
		}
		stw.Snapshot()
	case "thermal":
		if usage {
			stw.addHistory(":thermal tmp[℃]")
			return nil
		}
		if narg < 2 {
			return st.NotEnoughArgs(":thermal")
		}
		var els []*st.Elem
		if stw.SelectElem == nil || len(stw.SelectElem) == 0 {
			enum := 0
			els = make([]*st.Elem, 0)
		ex_thermal:
			for {
				select {
				case <-time.After(time.Second):
					break ex_thermal
				case <-stw.exmodeend:
					break ex_thermal
				case el := <-stw.exmodech:
					if el == nil {
						break ex_thermal
					}
					if el, ok := el.(*st.Elem); ok {
						els = append(els, el)
						enum++
					}
				}
			}
			if enum == 0 {
				return errors.New(":thermal no selected elem")
			}
			els = els[:enum]
		} else {
			els = stw.SelectElem
		}
		tmp, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return err
		}
		alpha := 12.0 * 1e-6
		if al, ok := argdict["ALPHA"]; ok {
			tmpal, err := strconv.ParseFloat(al, 64)
			if err == nil {
				alpha = tmpal
			}
		}
		stw.addHistory(fmt.Sprintf("ALPHA: %.3E", alpha))
		for _, el := range els {
			if el == nil || el.Lock || !el.IsLineElem() {
				continue
			}
			if len(el.Sect.Figs) == 0 {
				continue
			}
			if a, ok := el.Sect.Figs[0].Value["AREA"]; ok {
				val := el.Sect.Figs[0].Prop.E * a * alpha * tmp
				el.Cmq[0] = val
				el.Cmq[6] = -val
			}
		}
		stw.Snapshot()
	case "divide":
		if usage {
			stw.addHistory(":divide [mid, n, elem, ons, axis, length]")
			return nil
		}
		if narg < 2 {
			return st.NotEnoughArgs(":divide")
		}
		if stw.SelectElem == nil || len(stw.SelectElem) == 0 {
			return errors.New(":divide: no selected elem")
		}
		var divfunc func(*st.Elem) ([]*st.Node, []*st.Elem, error)
		switch strings.ToLower(args[1]) {
		case "mid":
			if usage {
				stw.addHistory(":divide mid")
				return nil
			}
			divfunc = func(el *st.Elem) ([]*st.Node, []*st.Elem, error) {
				return el.DivideAtMid(EPS)
			}
		case "n":
			if usage {
				stw.addHistory(":divide n div")
				return nil
			}
			if narg < 3 {
				return st.NotEnoughArgs(":divide n")
			}
			val, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return err
			}
			ndiv := int(val)
			divfunc = func(el *st.Elem) ([]*st.Node, []*st.Elem, error) {
				return el.DivideInN(ndiv, EPS)
			}
		case "elem":
			if usage {
				stw.addHistory(":divide elem (eps)")
				return nil
			}
			eps := EPS
			if narg >= 3 {
				val, err := strconv.ParseFloat(args[2], 64)
				if err == nil {
					eps = val
				}
			}
			divfunc = func(el *st.Elem) ([]*st.Node, []*st.Elem, error) {
				els, err := el.DivideAtElem(eps)
				return nil, els, err
			}
		case "ons":
			if usage {
				stw.addHistory(":divide ons (eps)")
				return nil
			}
			eps := EPS
			if narg >= 3 {
				val, err := strconv.ParseFloat(args[2], 64)
				if err == nil {
					eps = val
				}
			}
			divfunc = func(el *st.Elem) ([]*st.Node, []*st.Elem, error) {
				return el.DivideAtOns(eps)
			}
		case "axis":
			if usage {
				stw.addHistory(":divide axis [x, y, z] coord")
				return nil
			}
			if narg < 4 {
				return st.NotEnoughArgs(":divide axis")
			}
			var axis int
			switch args[2] {
			default:
				return errors.New(":divide axis: unknown axis")
			case "x", "X":
				axis = 0
			case "y", "Y":
				axis = 1
			case "z", "Z":
				axis = 2
			}
			val, err := strconv.ParseFloat(args[3], 64)
			if err != nil {
				return err
			}
			divfunc = func(el *st.Elem) ([]*st.Node, []*st.Elem, error) {
				return el.DivideAtAxis(axis, val, EPS)
			}
		case "length":
			if usage {
				stw.addHistory(":divide length l")
				return nil
			}
			if narg < 3 {
				return st.NotEnoughArgs(":divide length")
			}
			val, err := strconv.ParseFloat(args[2], 64)
			if err != nil {
				return err
			}
			divfunc = func(el *st.Elem) ([]*st.Node, []*st.Elem, error) {
				return el.DivideAtLength(val, EPS)
			}
		}
		if divfunc == nil {
			return errors.New(":divide: unknown format")
		}
		tmpels := make([]*st.Elem, 0)
		enum := 0
		for _, el := range stw.SelectElem {
			if el == nil {
				continue
			}
			_, els, err := divfunc(el)
			if err != nil {
				stw.errormessage(err, ERROR)
				continue
			}
			if err == nil && len(els) > 1 {
				tmpels = append(tmpels, els...)
				enum += len(els)
			}
		}
		stw.SelectElem = tmpels[:enum]
		stw.Snapshot()
	case "section":
		if usage {
			stw.addHistory(":section sectcode")
			return nil
		}
		if narg < 2 {
			if stw.SelectElem != nil && len(stw.SelectElem) >= 1 {
				stw.SectionData(stw.SelectElem[0].Sect)
				go func(sec *st.Sect) {
					stw.exmodech <- sec
				}(stw.SelectElem[0].Sect)
				return nil
			}
			if t, tok := stw.TextBox["SECTION"]; tok {
				t.Value = make([]string, 0)
			}
			return nil
		}
		switch {
		case strings.EqualFold(args[1], "off"):
			if t, tok := stw.TextBox["SECTION"]; tok {
				t.Value = make([]string, 0)
			}
			return nil
		case strings.EqualFold(args[1], "curtain"):
			sects := make([]*st.Sect, len(stw.Frame.Sects))
			num := 0
			for _, sec := range stw.Frame.Sects {
				if sec.Num > 900 {
					continue
				}
				if sec.HasArea(0) {
					continue
				}
				if !sec.HasBrace() {
					sects[num] = sec
					num++
				}
			}
			if num == 0 {
				return nil
			}
			sects = sects[:num]
			stw.SectionData(sects[0])
			go func(ss []*st.Sect) {
				for _, sec := range ss {
					stw.exmodech <- sec
				}
				stw.exmodeend <- 1
			}(sects)
		default:
			tmp, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}
			snum := int(tmp)
			if sec, ok := stw.Frame.Sects[snum]; ok {
				if narg >= 3 && args[2] == "<-" {
					select {
					case <-time.After(time.Second):
						break
					case al := <-stw.exmodech:
						if al == nil {
							break
						}
						switch al := al.(type) {
						case st.Shape:
							if sec.HasArea(0) {
								sec.Figs[0].SetShapeProperty(al)
								sec.Name = al.Description()
							}
						}
					}
				}
				stw.SectionData(sec)
				go func(s *st.Sect) {
					stw.exmodech <- s
				}(sec)
			} else {
				return errors.New(fmt.Sprintf(":section SECT %d doesn't exist", snum))
			}
		}
	case "thick":
		if usage {
			stw.addHistory(":thick nfig val")
			return nil
		}
		if narg < 3 {
			return st.NotEnoughArgs(":thick")
		}
		tmp, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return err
		}
		val, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			return err
		}
		ind := int(tmp) - 1
		select {
		case <-time.After(time.Second):
			break
		case sec := <-stw.exmodech:
			if sec == nil {
				break
			}
			if sec, ok := sec.(*st.Sect); ok {
				if sec.HasThick(ind) {
					sec.Figs[ind].Value["THICK"] = val
				}
			}
		}
	case "add":
		if usage {
			stw.addHistory(":add sect sectcode")
			return nil
		}
		if narg < 2 {
			return st.NotEnoughArgs(":add")
		}
		switch strings.ToLower(args[1]) {
		case "elem":
			var etype int
			if et, ok := argdict["ETYPE"]; ok {
				switch {
				case re_column.MatchString(et):
					etype = st.COLUMN
				case re_girder.MatchString(et):
					etype = st.GIRDER
				case re_slab.MatchString(et):
					etype = st.BRACE
				case re_wall.MatchString(et):
					etype = st.WALL
				case re_slab.MatchString(et):
					etype = st.SLAB
				default:
					tmp, err := strconv.ParseInt(et, 10, 64)
					if err != nil {
						return err
					}
					etype = int(tmp)
				}
			} else {
				return errors.New(":add elem: no etype selected")
			}
			var sect *st.Sect
			if sc, ok := argdict["SECT"]; ok {
				tmp, err := strconv.ParseInt(sc, 10, 64)
				if err != nil {
					return err
				}
				if sec, ok := stw.Frame.Sects[int(tmp)]; ok {
					sect = sec
				} else {
					return errors.New(fmt.Sprintf(":add elem: SECT %d doesn't exist", tmp))
				}
			} else {
				return errors.New(":add elem: no sectcode selected")
			}
			enod := make([]*st.Node, 0)
			enods := 0
		ex_addelem:
			for {
				select {
				case <-time.After(time.Second):
					break ex_addelem
				case <-stw.exmodeend:
					break ex_addelem
				case ent := <-stw.exmodech:
					if n, ok := ent.(*st.Node); ok {
						enod = append(enod, n)
						enods++
					}
				}
			}
			enod = enod[:enods]
			switch etype {
			case st.COLUMN, st.GIRDER, st.BRACE:
				stw.Frame.AddLineElem(-1, enod[:2], sect, etype)
			case st.WALL, st.SLAB:
				if enods > 4 {
					enod = enod[:4]
				}
				stw.Frame.AddPlateElem(-1, enod, sect, etype)
			}
		case "sec", "sect":
			if narg < 3 {
				return st.NotEnoughArgs(":add sect")
			}
			val, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return err
			}
			snum := int(val)
			if _, ok := stw.Frame.Sects[snum]; ok && !bang {
				return errors.New(fmt.Sprintf(":add sect: SECT %d already exists", snum))
			}
			sec := stw.Frame.AddSect(snum)
			select {
			case <-time.After(time.Second):
				break
			case al := <-stw.exmodech:
				if a, ok := al.(st.Shape); ok {
					sec.Figs = make([]*st.Fig, 1)
					f := st.NewFig()
					if p, ok := stw.Frame.Props[101]; ok {
						f.Prop = p
					} else {
						f.Prop = stw.Frame.DefaultProp()
					}
					sec.Figs[0] = f
					sec.Figs[0].SetShapeProperty(a)
					sec.Name = a.Description()
				}
			}
		}
	case "copy":
		if usage {
			stw.addHistory(":copy sect sectcode")
			return nil
		}
		if narg < 2 {
			return st.NotEnoughArgs(":copy")
		}
		switch strings.ToLower(args[1]) {
		case "sec", "sect":
			if narg < 3 {
				return st.NotEnoughArgs(":copy sect")
			}
			val, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return err
			}
			snum := int(val)
			if _, ok := stw.Frame.Sects[snum]; ok && !bang {
				return errors.New(fmt.Sprintf(":copy sect: SECT %d already exists", snum))
			}
			select {
			case <-time.After(time.Second):
				break
			case s := <-stw.exmodech:
				if sec, ok := s.(*st.Sect); ok {
					as := sec.Snapshot(stw.Frame)
					as.Num = snum
					stw.Frame.Sects[snum] = as
					stw.Frame.Show.Sect[snum] = true
				}
			}
		}
	case "max":
		if stw.SelectElem != nil && len(stw.SelectElem) >= 1 {
			maxval := -1e16
			var valfunc func(*st.Elem) float64
			var sel *st.Elem
			abs := false
			if _, ok := argdict["ABS"]; ok {
				valfunc = func(elem *st.Elem) float64 {
					return elem.CurrentValue(stw.Frame.Show, true, true)
				}
				abs = true
			} else {
				valfunc = func(elem *st.Elem) float64 {
					return elem.CurrentValue(stw.Frame.Show, true, false)
				}
			}
			for _, el := range stw.SelectElem {
				if el == nil {
					continue
				}
				if tmpval := valfunc(el); tmpval > maxval {
					sel = el
					maxval = tmpval
				}
			}
			if sel != nil {
				stw.SelectElem = []*st.Elem{sel}
				stw.addHistory(fmt.Sprintf("ELEM %d: %.3f", sel.Num, sel.CurrentValue(stw.Frame.Show, true, abs)))
			}
		} else if stw.SelectNode != nil && len(stw.SelectNode) >= 1 {
			maxval := -1e16
			var valfunc func(*st.Node) float64
			var sn *st.Node
			abs := false
			if _, ok := argdict["ABS"]; ok {
				valfunc = func(node *st.Node) float64 {
					return node.CurrentValue(stw.Frame.Show, true, true)
				}
				abs = true
			} else {
				valfunc = func(node *st.Node) float64 {
					return node.CurrentValue(stw.Frame.Show, true, false)
				}
			}
			for _, n := range stw.SelectNode {
				if n == nil {
					continue
				}
				if tmpval := valfunc(n); tmpval > maxval {
					sn = n
					maxval = tmpval
				}
			}
			if sn != nil {
				stw.SelectNode = []*st.Node{sn}
				stw.addHistory(fmt.Sprintf("NODE %d: %.3f", sn.Num, sn.CurrentValue(stw.Frame.Show, true, abs)))
			}
		} else {
			return errors.New(":max no selected elem/node")
		}
	case "min":
		if stw.SelectElem != nil && len(stw.SelectElem) >= 1 {
			minval := 1e16
			var valfunc func(*st.Elem) float64
			var sel *st.Elem
			abs := false
			if _, ok := argdict["ABS"]; ok {
				valfunc = func(elem *st.Elem) float64 {
					return elem.CurrentValue(stw.Frame.Show, false, true)
				}
				abs = true
			} else {
				valfunc = func(elem *st.Elem) float64 {
					return elem.CurrentValue(stw.Frame.Show, false, false)
				}
			}
			for _, el := range stw.SelectElem {
				if el == nil {
					continue
				}
				if tmpval := valfunc(el); tmpval < minval {
					sel = el
					minval = tmpval
				}
			}
			if sel != nil {
				stw.SelectElem = []*st.Elem{sel}
				stw.addHistory(fmt.Sprintf("ELEM %d: %.3f", sel.Num, sel.CurrentValue(stw.Frame.Show, false, abs)))
			}
		} else if stw.SelectNode != nil && len(stw.SelectNode) >= 1 {
			minval := 1e16
			var valfunc func(*st.Node) float64
			var sn *st.Node
			abs := false
			if _, ok := argdict["ABS"]; ok {
				valfunc = func(node *st.Node) float64 {
					return node.CurrentValue(stw.Frame.Show, false, true)
				}
				abs = true
			} else {
				valfunc = func(node *st.Node) float64 {
					return node.CurrentValue(stw.Frame.Show, false, false)
				}
			}
			for _, n := range stw.SelectNode {
				if n == nil {
					continue
				}
				if tmpval := valfunc(n); tmpval < minval {
					sn = n
					minval = tmpval
				}
			}
			if sn != nil {
				stw.SelectNode = []*st.Node{sn}
				stw.addHistory(fmt.Sprintf("NODE %d: %.3f", sn.Num, sn.CurrentValue(stw.Frame.Show, false, abs)))
			}
		} else {
			return errors.New(":min no selected elem/node")
		}
	case "average":
		if stw.SelectElem != nil && len(stw.SelectElem) >= 1 {
			var valfunc func(*st.Elem) float64
			if _, ok := argdict["ABS"]; ok {
				valfunc = func(elem *st.Elem) float64 {
					return elem.CurrentValue(stw.Frame.Show, false, true)
				}
			} else {
				valfunc = func(elem *st.Elem) float64 {
					return elem.CurrentValue(stw.Frame.Show, false, false)
				}
			}
			val := 0.0
			num := 0
			for _, el := range stw.SelectElem {
				if el == nil {
					continue
				}
				val += valfunc(el)
				num++
			}
			if num >= 1 {
				stw.addHistory(fmt.Sprintf("%d ELEMs : %.5f", num, val/float64(num)))
			}
		} else if stw.SelectNode != nil && len(stw.SelectNode) >= 1 {
			var valfunc func(*st.Node) float64
			if _, ok := argdict["ABS"]; ok {
				valfunc = func(node *st.Node) float64 {
					return node.CurrentValue(stw.Frame.Show, false, true)
				}
			} else {
				valfunc = func(node *st.Node) float64 {
					return node.CurrentValue(stw.Frame.Show, false, false)
				}
			}
			val := 0.0
			num := 0
			for _, n := range stw.SelectNode {
				if n == nil {
					continue
				}
				val += valfunc(n)
				num++
			}
			if num >= 1 {
				stw.addHistory(fmt.Sprintf("%d NODEs: %.5f", num, val/float64(num)))
			}
		} else {
			return errors.New(":average no selected elem/node")
		}
	case "sum":
		if stw.SelectElem != nil && len(stw.SelectElem) >= 1 {
			var valfunc func(*st.Elem) float64
			if _, ok := argdict["ABS"]; ok {
				valfunc = func(elem *st.Elem) float64 {
					return elem.CurrentValue(stw.Frame.Show, false, true)
				}
			} else {
				valfunc = func(elem *st.Elem) float64 {
					return elem.CurrentValue(stw.Frame.Show, false, false)
				}
			}
			val := 0.0
			num := 0
			for _, el := range stw.SelectElem {
				if el == nil {
					continue
				}
				val += valfunc(el)
				num++
			}
			if num >= 1 {
				stw.addHistory(fmt.Sprintf("%d ELEMs : %.5f", num, val))
			}
		} else if stw.SelectNode != nil && len(stw.SelectNode) >= 1 {
			var valfunc func(*st.Node) float64
			if _, ok := argdict["ABS"]; ok {
				valfunc = func(node *st.Node) float64 {
					return node.CurrentValue(stw.Frame.Show, false, true)
				}
			} else {
				valfunc = func(node *st.Node) float64 {
					return node.CurrentValue(stw.Frame.Show, false, false)
				}
			}
			val := 0.0
			num := 0
			for _, n := range stw.SelectNode {
				if n == nil {
					continue
				}
				val += valfunc(n)
				num++
			}
			if num >= 1 {
				stw.addHistory(fmt.Sprintf("%d NODEs: %.5f", num, val))
			}
		} else {
			return errors.New(":sum no selected elem/node")
		}
	case "hide":
	ex_hide:
		for {
			select {
			case <-time.After(time.Second):
				break ex_hide
			case <-stw.exmodeend:
				break ex_hide
			case ent := <-stw.exmodech:
				if h, ok := ent.(st.Hider); ok {
					h.Hide()
				}
			}
		}
	case "height":
		if usage {
			stw.addHistory(":height f1 f2")
			return nil
		}
		if narg == 1 {
			axisrange(stw, 2, -100.0, 1000.0, false)
			return nil
		}
		if narg < 3 {
			return st.NotEnoughArgs(":ht")
		}
		var min, max int
		if strings.EqualFold(args[1], "n") {
			min = stw.Frame.Ai.Nfloor
		} else {
			tmp, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}
			min = int(tmp)
		}
		if strings.EqualFold(args[2], "n") {
			max = stw.Frame.Ai.Nfloor
		} else {
			tmp, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return err
			}
			max = int(tmp)
		}
		l := len(stw.Frame.Ai.Boundary)
		if min < 0 || min >= l || max < 0 || max >= l {
			return errors.New(":ht out of boundary")
		}
		axisrange(stw, 2, stw.Frame.Ai.Boundary[min], stw.Frame.Ai.Boundary[max], false)
	case "height+":
		stw.NextFloor()
	case "height-":
		stw.PrevFloor()
	case "view":
		if usage {
			stw.addHistory(":view [top,front,back,right,left]")
			return nil
		}
		switch strings.ToUpper(args[1]) {
		case "TOP":
			stw.SetAngle(90.0, -90.0)
		case "FRONT":
			stw.SetAngle(0.0, -90.0)
		case "BACK":
			stw.SetAngle(0.0, 90.0)
		case "RIGHT":
			stw.SetAngle(0.0, 0.0)
		case "LEFT":
			stw.SetAngle(0.0, 180.0)
		}
	case "printrange":
		if usage {
			stw.addHistory(":printrange [on,true,yes] [a3tate,a3yoko,a4tate,a4yoko]")
			stw.addHistory(":printrange [off,false,no]")
			return nil
		}
		if narg < 2 {
			showprintrange = !showprintrange
			break
		}
		switch strings.ToUpper(args[1]) {
		case "ON", "TRUE", "YES":
			showprintrange = true
			if narg >= 3 {
				err := stw.exmode(fmt.Sprintf(":paper %s", strings.Join(args[2:], " ")))
				if err != nil {
					return err
				}
			}
		case "OFF", "FALSE", "NO":
			showprintrange = false
		default:
			err := stw.exmode(fmt.Sprintf(":paper %s", strings.Join(args[1:], " ")))
			if err != nil {
				return err
			}
			showprintrange = true
		}
	case "paper":
		if usage {
			stw.addHistory(":paper [a3tate,a3yoko,a4tate,a4yoko]")
			return nil
		}
		if narg < 2 {
			return st.NotEnoughArgs(":paper")
		}
		tate := regexp.MustCompile("(?i)a([0-9]+) *t(a(t(e?)?)?)?")
		yoko := regexp.MustCompile("(?i)a([0-9]+) *y(o(k(o?)?)?)?")
		name := strings.Join(args[1:], " ")
		switch {
		case tate.MatchString(name):
			fs := tate.FindStringSubmatch(name)
			switch fs[1] {
			case "3":
				stw.papersize = A3_TATE
			case "4":
				stw.papersize = A4_TATE
			}
		case yoko.MatchString(name):
			fs := yoko.FindStringSubmatch(name)
			switch fs[1] {
			case "3":
				stw.papersize = A3_YOKO
			case "4":
				stw.papersize = A4_YOKO
			}
		default:
			return errors.New(":paper unknown papersize")
		}
	case "color":
		if usage {
			stw.addHistory(":color [n,sect,rate,white,mono,strong]")
			return nil
		}
		if narg < 2 {
			stw.SetColorMode(st.ECOLOR_SECT)
			break
		}
		switch strings.ToUpper(args[1]) {
		case "N":
			stw.SetColorMode(st.ECOLOR_N)
		case "SECT":
			stw.SetColorMode(st.ECOLOR_SECT)
		case "RATE":
			stw.SetColorMode(st.ECOLOR_RATE)
		case "WHIHTE", "MONO", "MONOCHROME":
			stw.SetColorMode(st.ECOLOR_WHITE)
		case "STRONG":
			stw.SetColorMode(st.ECOLOR_STRONG)
		}
	case "mono":
		stw.SetColorMode(st.ECOLOR_WHITE)
	case "postscript":
		if fn == "" {
			fn = filepath.Join(stw.Cwd, "test.ps")
		}
		var paper ps.Paper
		switch stw.papersize {
		default:
			paper = ps.A4Portrait
		case A4_TATE:
			paper = ps.A4Portrait
		case A4_YOKO:
			paper = ps.A4Landscape
		case A3_TATE:
			paper = ps.A3Portrait
		case A3_YOKO:
			paper = ps.A3Landscape
		}
		v := stw.Frame.View.Copy()
		stw.Frame.SetFocus(nil)
		stw.Frame.CentringTo(paper)
		err := stw.Frame.PrintPostScript(fn, paper)
		stw.Frame.View = v
		if err != nil {
			return err
		}
	case "analysis":
		if usage {
			stw.addHistory(":analysis")
			return nil
		}
		err := stw.SaveFile(stw.Frame.Path)
		if err != nil {
			return err
		}
		var anarg string
		if narg >= 3 {
			anarg = args[2]
		} else {
			anarg = "-a"
		}
		err = stw.Analysis(filepath.ToSlash(stw.Frame.Path), anarg)
		if err != nil {
			return err
		}
		stw.Reload()
		stw.ReadAll()
		stw.Redraw()
	case "arclm001":
		if usage {
			stw.addHistory(":arclm001 {-period=name} {-all} {-solver=name} {-eps=value} {-noinit} filename")
			return nil
		}
		var otp string
		if fn == "" {
			otp = st.Ce(stw.Frame.Path, ".otp")
		} else {
			otp = fn
		}
		otps := []string{otp}
		if o, ok := argdict["OTP"]; ok {
			otp = o
		}
		sol := "LLS"
		if s, ok := argdict["SOLVER"]; ok {
			if s != "" {
				sol = strings.ToUpper(s)
			}
		}
		eps := 1e-16
		if e, ok := argdict["EPS"]; ok {
			if e != "" {
				tmp, err := strconv.ParseFloat(e, 64)
				if err == nil {
					eps = tmp
				}
			}
		}
		per := "L"
		if p, ok := argdict["PERIOD"]; ok {
			if p != "" {
				per = strings.ToUpper(p)
			}
		}
		var pers []string
		pers = []string{per}
		lap := 1
		var extra [][]float64
		if _, ok := argdict["ALL"]; ok {
			extra = make([][]float64, 2)
			pers = []string{"L", "X", "Y"}
			otps = []string{st.Ce(otp, ".otl"), st.Ce(otp, ".ohx"), st.Ce(otp, ".ohy")}
			per = "L"
			lap = 3
			for i, eper := range []string{"X", "Y"} {
				eaf := stw.Frame.Arclms[eper]
				_, _, vec, err := eaf.AssemGlobalVector(1.0)
				if err != nil {
					return err
				}
				extra[i] = vec
			}
		}
		stw.addHistory(fmt.Sprintf("PERIOD: %s", per))
		stw.addHistory(fmt.Sprintf("OUTPUT: %s", otp))
		stw.addHistory(fmt.Sprintf("SOLVER: %s", sol))
		stw.addHistory(fmt.Sprintf("EPS: %.3E", eps))
		init := true
		if _, ok := argdict["NOINIT"]; ok {
			init = false
			stw.addHistory("NO INITIALISATION")
		}
		af := stw.Frame.Arclms[per]
		go func() {
			err := af.Arclm001(otps, init, sol, eps, extra...)
			af.Endch <- err
		}()
		stw.CurrentLap("Calculating...", 0, lap)
		go func() {
		read001:
			for {
				select {
				case nlap := <-af.Lapch:
					stw.Frame.ReadArclmData(af, pers[nlap])
					af.Lapch <- 1
					stw.CurrentLap("Calculating...", nlap, lap)
					stw.Redraw()
				case <-af.Endch:
					stw.CurrentLap("Completed", lap, lap)
					stw.SetPeriod(per)
					stw.Redraw()
					break read001
				}
			}
		}()
	case "arclm201":
		if usage {
			stw.addHistory(":arclm201 {-period=name} {-lap=nlap} {-safety=val} {-start=val} {-noinit} filename")
			return nil
		}
		var otp string
		if fn == "" {
			otp = st.Ce(stw.Frame.Path, ".otp")
		} else {
			otp = fn
		}
		if o, ok := argdict["OTP"]; ok {
			otp = o
		}
		lap := 1
		if l, ok := argdict["LAP"]; ok {
			tmp, err := strconv.ParseInt(l, 10, 64)
			if err == nil {
				lap = int(tmp)
			}
		}
		safety := 1.0
		if s, ok := argdict["SAFETY"]; ok {
			tmp, err := strconv.ParseFloat(s, 64)
			if err == nil {
				safety = tmp
			}
		}
		start := 0.0
		if s, ok := argdict["START"]; ok {
			tmp, err := strconv.ParseFloat(s, 64)
			if err == nil {
				start = tmp
			}
		}
		per := "L"
		if p, ok := argdict["PERIOD"]; ok {
			if p != "" {
				per = strings.ToUpper(p)
			}
		}
		stw.addHistory(fmt.Sprintf("PERIOD: %s", per))
		stw.addHistory(fmt.Sprintf("OUTPUT: %s", otp))
		stw.addHistory(fmt.Sprintf("LAP: %d, SAFETY: %.3f, START: %.3f", lap, safety, start))
		init := true
		if _, ok := argdict["NOINIT"]; ok {
			init = false
			stw.addHistory("NO INITIALISATION")
		}
		af := stw.Frame.Arclms[per]
		go func() {
			err := af.Arclm201(otp, init, lap, safety, start)
			af.Endch <- err
		}()
		stw.CurrentLap("Calculating...", 0, lap)
		go func() {
		read201:
			for {
				select {
				case nlap := <-af.Lapch:
					stw.Frame.ReadArclmData(af, per)
					af.Lapch <- 1
					stw.CurrentLap("Calculating...", nlap, lap)
					stw.Redraw()
				case <-af.Endch:
					stw.CurrentLap("Completed", lap, lap)
					stw.Redraw()
					break read201
				}
			}
		}()
	case "arclm301":
		if usage {
			stw.addHistory(":arclm301 {-period=name} {-sects=val} {-eps=val} {-noinit} filename")
			return nil
		}
		var otp string
		var sects []int
		if fn == "" {
			otp = st.Ce(stw.Frame.Path, ".otp")
		} else {
			otp = fn
		}
		if o, ok := argdict["OTP"]; ok {
			otp = o
		}
		if s, ok := argdict["SECTS"]; ok {
			stw.addHistory(fmt.Sprintf("SOIL SPRING: %s", s))
			sects = SplitNums(s)
		}
		eps := 1E-3
		if s, ok := argdict["EPS"]; ok {
			tmp, err := strconv.ParseFloat(s, 64)
			if err == nil {
				eps = tmp
			}
		}
		per := "L"
		if p, ok := argdict["PERIOD"]; ok {
			if p != "" {
				per = strings.ToUpper(p)
			}
		}
		stw.addHistory(fmt.Sprintf("PERIOD: %s", per))
		stw.addHistory(fmt.Sprintf("OUTPUT: %s", otp))
		stw.addHistory(fmt.Sprintf("EPS: %.3E", eps))
		af := stw.Frame.Arclms[per]
		init := true
		if _, ok := argdict["NOINIT"]; ok {
			init = false
			stw.addHistory("NO INITIALISATION")
		}
		go func() {
			err := af.Arclm301(otp, init, sects, eps)
			af.Endch <- err
		}()
		stw.CurrentLap("Calculating...", 0, 0)
		go func() {
		read301:
			for {
				select {
				case nlap := <-af.Lapch:
					stw.Frame.ReadArclmData(af, per)
					af.Lapch <- 1
					stw.CurrentLap("Calculating...", nlap, 0)
					stw.Redraw()
				case <-af.Endch:
					stw.CurrentLap("Completed", 0, 0)
					stw.Redraw()
					break read301
				}
			}
		}()
	}
	return nil
}

func (stw *Window) NextFloor() {
	for i, z := range []string{"ZMIN", "ZMAX"} {
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
		stw.Labels[z].SetAttribute("VALUE", fmt.Sprintf("%.3f", val))
	}
	stw.Redraw()
}

func (stw *Window) PrevFloor() {
	for i, z := range []string{"ZMIN", "ZMAX"} {
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
		stw.Labels[z].SetAttribute("VALUE", fmt.Sprintf("%.3f", val))
	}
	stw.Redraw()
}

func axisrange(stw *Window, axis int, min, max float64, any bool) {
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
		stw.Labels["XMIN"].SetAttribute("VALUE", fmt.Sprintf("%.3f", min))
		stw.Labels["XMAX"].SetAttribute("VALUE", fmt.Sprintf("%.3f", max))
	case 1:
		stw.Frame.Show.Yrange[0] = min
		stw.Frame.Show.Yrange[1] = max
		stw.Labels["YMIN"].SetAttribute("VALUE", fmt.Sprintf("%.3f", min))
		stw.Labels["YMAX"].SetAttribute("VALUE", fmt.Sprintf("%.3f", max))
	case 2:
		stw.Frame.Show.Zrange[0] = min
		stw.Frame.Show.Zrange[1] = max
		stw.Labels["ZMIN"].SetAttribute("VALUE", fmt.Sprintf("%.3f", min))
		stw.Labels["ZMAX"].SetAttribute("VALUE", fmt.Sprintf("%.3f", max))
	}
	stw.Redraw()
}

// }}}

// Draw// {{{
func (stw *Window) DrawFrame(canv *cd.Canvas, color uint, flush bool) {
	if stw.Frame == nil {
		return
	}
	canv.Hatch(cd.CD_FDIAGONAL)
	canv.Clear()
	stw.Frame.View.Set(0)
	if stw.Frame.Show.GlobalAxis {
		stw.DrawGlobalAxis(canv, color)
	}
	if stw.Frame.Show.Kijun {
		canv.Foreground(KijunColor)
		for _, k := range stw.Frame.Kijuns {
			if k.IsHidden(stw.Frame.Show) {
				continue
			}
			k.Pstart = stw.Frame.View.ProjectCoord(k.Start)
			k.Pend = stw.Frame.View.ProjectCoord(k.End)
			DrawKijun(k, canv, stw.Frame.Show)
		}
	}
	if stw.Frame.Show.Measure {
		canv.TextAlignment(cd.CD_SOUTH)
		canv.InteriorStyle(cd.CD_SOLID)
		canv.Foreground(MeasureColor)
		for _, m := range stw.Frame.Measures {
			if m.IsHidden(stw.Frame.Show) {
				continue
			}
			DrawMeasure(m, canv, stw.Frame.Show)
		}
		canv.TextAlignment(DefaultTextAlignment)
		canv.InteriorStyle(cd.CD_HATCH)
	}
	canv.Foreground(cd.CD_WHITE)
	for _, n := range stw.Frame.Nodes {
		stw.Frame.View.ProjectNode(n)
		if stw.Frame.Show.Deformation {
			stw.Frame.View.ProjectDeformation(n, stw.Frame.Show)
		}
		if n.IsHidden(stw.Frame.Show) {
			continue
		}
		if color == st.ECOLOR_BLACK {
			canv.Foreground(cd.CD_BLACK)
		} else {
			if n.Lock {
				canv.Foreground(LOCKED_NODE_COLOR)
			} else {
				switch n.ConfState() {
				case st.CONF_FREE:
					canv.Foreground(canvasFontColor)
				case st.CONF_PIN:
					canv.Foreground(cd.CD_GREEN)
				case st.CONF_FIX:
					canv.Foreground(cd.CD_DARK_GREEN)
				default:
					canv.Foreground(cd.CD_CYAN)
				}
			}
			for _, j := range stw.SelectNode {
				if j == n {
					canv.Foreground(cd.CD_RED)
					break
				}
			}
		}
		DrawNode(n, canv, stw.Frame.Show)
	}
	canv.LineStyle(cd.CD_CONTINUOUS)
	canv.Hatch(cd.CD_FDIAGONAL)
	if !stw.Frame.Show.Select {
		els := st.SortedElem(stw.Frame.Elems, func(e *st.Elem) float64 { return -e.DistFromProjection(stw.Frame.View) })
	loop:
		for _, el := range els {
			if el.IsHidden(stw.Frame.Show) {
				continue
			}
			canv.LineStyle(cd.CD_CONTINUOUS)
			canv.Hatch(cd.CD_FDIAGONAL)
			for _, j := range stw.SelectElem {
				if j == el {
					continue loop
				}
			}
			if el.Lock {
				canv.Foreground(LOCKED_ELEM_COLOR)
			} else {
				switch color {
				case st.ECOLOR_WHITE:
					canv.Foreground(cd.CD_WHITE)
				case st.ECOLOR_BLACK:
					canv.Foreground(cd.CD_BLACK)
				case st.ECOLOR_SECT:
					canv.Foreground(el.Sect.Color)
				case st.ECOLOR_RATE:
					val, err := el.RateMax(stw.Frame.Show)
					if err != nil {
						canv.Foreground(cd.CD_DARK_GRAY)
					} else {
						canv.Foreground(st.Rainbow(val, st.RateBoundary))
					}
				// case st.ECOLOR_HEIGHT:
				//     canv.Foreground(st.Rainbow(el.MidPoint()[2], st.HeightBoundary))
				case st.ECOLOR_N:
					if el.N(stw.Frame.Show.Period, 0) >= 0.0 {
						canv.Foreground(st.RainbowColor[0]) // Compression: Blue
					} else {
						canv.Foreground(st.RainbowColor[6]) // Tension: Red
					}
				case st.ECOLOR_STRONG:
					if el.IsLineElem() {
						Ix, err := el.Sect.Ix(0)
						if err != nil {
							canv.Foreground(cd.CD_WHITE)
						}
						Iy, err := el.Sect.Iy(0)
						if err != nil {
							canv.Foreground(cd.CD_WHITE)
						}
						if Ix > Iy {
							canv.Foreground(st.RainbowColor[0]) // Strong: Blue
						} else if Ix == Iy {
							canv.Foreground(st.RainbowColor[4]) // Same: Yellow
						} else {
							canv.Foreground(st.RainbowColor[6]) // Weak: Red
						}
					} else {
						canv.Foreground(el.Sect.Color)
					}
				}
			}
			DrawElem(el, canv, stw.Frame.Show)
		}
	}
	canv.Hatch(cd.CD_DIAGCROSS)
	nomv := stw.Frame.Show.NoMomentValue
	stw.Frame.Show.NoMomentValue = false
	for _, el := range stw.SelectElem {
		canv.LineStyle(cd.CD_DOTTED)
		if el == nil || el.IsHidden(stw.Frame.Show) {
			continue
		}
		if el.Lock {
			canv.Foreground(LOCKED_ELEM_COLOR)
		} else {
			switch color {
			case st.ECOLOR_WHITE:
				canv.Foreground(cd.CD_WHITE)
			case st.ECOLOR_BLACK:
				canv.Foreground(cd.CD_BLACK)
			case st.ECOLOR_SECT:
				canv.Foreground(el.Sect.Color)
			case st.ECOLOR_RATE:
				val, err := el.RateMax(stw.Frame.Show)
				if err != nil {
					canv.Foreground(cd.CD_DARK_GRAY)
				} else {
					canv.Foreground(st.Rainbow(val, st.RateBoundary))
				}
			// case st.ECOLOR_HEIGHT:
			//     canv.Foreground(st.Rainbow(el.MidPoint()[2], st.HeightBoundary))
			case st.ECOLOR_N:
				if el.N(stw.Frame.Show.Period, 0) >= 0.0 {
					canv.Foreground(st.RainbowColor[0]) // Compression: Blue
				} else {
					canv.Foreground(st.RainbowColor[6]) // Tension: Red
				}
			}
		}
		DrawElem(el, canv, stw.Frame.Show)
	}
	stw.Frame.Show.NoMomentValue = nomv
	if stw.Frame.Fes != nil {
		DrawEccentric(stw.Frame, canv, stw.Frame.Show)
	}
	if showprintrange {
		canv.LineStyle(cd.CD_CONTINUOUS)
		if color == st.ECOLOR_BLACK {
			canv.Foreground(cd.CD_BLACK)
		} else {
			canv.Foreground(cd.CD_GRAY)
		}
		DrawPrintRange(stw)
	}
	DrawLegend(canv, stw.Frame.Show)
	if flush {
		canv.Flush()
	}
	stw.SetViewData()
}

func (stw *Window) DrawTexts(canv *cd.Canvas, black bool) {
	if !stw.PageTitle.Hide {
		if black {
			col := stw.PageTitle.Font.Color
			stw.PageTitle.Font.Color = cd.CD_BLACK
			DrawText(stw.PageTitle, canv)
			stw.PageTitle.Font.Color = col
		} else {
			DrawText(stw.PageTitle, canv)
		}
	}
	if !stw.Title.Hide {
		if black {
			col := stw.Title.Font.Color
			stw.Title.Font.Color = cd.CD_BLACK
			DrawText(stw.Title, canv)
			stw.Title.Font.Color = col
		} else {
			DrawText(stw.Title, canv)
		}
	}
	if !stw.Text.Hide {
		if black {
			col := stw.Text.Font.Color
			stw.Text.Font.Color = cd.CD_BLACK
			DrawText(stw.Text, canv)
			stw.Text.Font.Color = col
		} else {
			DrawText(stw.Text, canv)
		}
	}
	for _, t := range stw.TextBox {
		if !t.Hide {
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
	if !STLOGO.Hide {
		DrawText(STLOGO, canv)
	}
}

func (stw *Window) Redraw() {
	stw.DrawFrame(stw.dbuff, stw.Frame.Show.ColorMode, false)
	if stw.Property {
		stw.UpdatePropertyDialog()
	}
	stw.DrawTexts(stw.dbuff, false)
	stw.dbuff.Flush()
}

func (stw *Window) DrawFrameNode() {
	if stw.Frame == nil {
		return
	}
	stw.dbuff.Clear()
	stw.Frame.View.Set(0)
	if stw.Frame.Show.GlobalAxis {
		stw.DrawGlobalAxis(stw.dbuff, stw.Frame.Show.ColorMode)
	}
	for _, n := range stw.Frame.Nodes {
		stw.Frame.View.ProjectNode(n)
		if n.Lock {
			stw.dbuff.Foreground(LOCKED_NODE_COLOR)
		} else {
			stw.dbuff.Foreground(canvasFontColor)
		}
		for _, j := range stw.SelectNode {
			if j == n {
				stw.dbuff.Foreground(cd.CD_RED)
				break
			}
		}
		DrawNode(n, stw.dbuff, stw.Frame.Show)
	}
	if len(stw.SelectElem) > 0 && stw.SelectElem[0] != nil {
		nomv := stw.Frame.Show.NoMomentValue
		stw.Frame.Show.NoMomentValue = false
		stw.dbuff.Hatch(cd.CD_DIAGCROSS)
		for _, el := range stw.SelectElem {
			stw.dbuff.LineStyle(cd.CD_DOTTED)
			if el == nil || el.IsHidden(stw.Frame.Show) {
				continue
			}
			if el.Lock {
				stw.dbuff.Foreground(LOCKED_ELEM_COLOR)
			} else {
				stw.dbuff.Foreground(cd.CD_WHITE)
			}
			DrawElem(el, stw.dbuff, stw.Frame.Show)
		}
		stw.Frame.Show.NoMomentValue = nomv
		stw.dbuff.LineStyle(cd.CD_CONTINUOUS)
		stw.dbuff.Hatch(cd.CD_FDIAGONAL)
	}
	stw.dbuff.Flush()
}

// TODO: implement
func (stw *Window) DrawConvexHull() {
	if stw.Frame == nil {
		return
	}
	var nnum int
	nodes := make([]*st.Node, len(stw.Frame.Nodes))
	for _, n := range stw.Frame.Nodes {
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
	stw.dbuff.Foreground(PlateEdgeColor)
	stw.dbuff.Begin(cd.CD_CLOSED_LINES)
	for i := 0; i < ncw+nccw+2; i++ {
		stw.dbuff.FVertex(chnodes[i].Pcoord[0], chnodes[i].Pcoord[1])
	}
	stw.dbuff.End()
	stw.dbuff.Flush()
}

func (stw *Window) SetSelectData() {
	if stw.Frame != nil {
		stw.lselect.SetAttribute("VALUE", fmt.Sprintf("ELEM = %5d\nNODE = %5d", len(stw.SelectElem), len(stw.SelectNode)))
	} else {
		stw.lselect.SetAttribute("VALUE", "ELEM =     0\nNODE =     0")
	}
}

// DataLabel
func (stw *Window) SetViewData() {
	if stw.Frame != nil {
		stw.Labels["GFACT"].SetAttribute("VALUE", fmt.Sprintf("%.1f", stw.Frame.View.Gfact))
		stw.Labels["DISTR"].SetAttribute("VALUE", fmt.Sprintf("%.1f", stw.Frame.View.Dists[0]))
		stw.Labels["DISTL"].SetAttribute("VALUE", fmt.Sprintf("%.1f", stw.Frame.View.Dists[1]))
		stw.Labels["PHI"].SetAttribute("VALUE", fmt.Sprintf("%.1f", stw.Frame.View.Angle[0]))
		stw.Labels["THETA"].SetAttribute("VALUE", fmt.Sprintf("%.1f", stw.Frame.View.Angle[1]))
		stw.Labels["FOCUSX"].SetAttribute("VALUE", fmt.Sprintf("%.1f", stw.Frame.View.Focus[0]))
		stw.Labels["FOCUSY"].SetAttribute("VALUE", fmt.Sprintf("%.1f", stw.Frame.View.Focus[1]))
		stw.Labels["FOCUSZ"].SetAttribute("VALUE", fmt.Sprintf("%.1f", stw.Frame.View.Focus[2]))
		stw.Labels["CENTERX"].SetAttribute("VALUE", fmt.Sprintf("%.1f", stw.Frame.View.Center[0]))
		stw.Labels["CENTERY"].SetAttribute("VALUE", fmt.Sprintf("%.1f", stw.Frame.View.Center[1]))
	}
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

func (stw *Window) SetShowRange() {
	xmin, xmax, ymin, ymax, zmin, zmax := stw.Frame.Bbox()
	stw.Frame.Show.Xrange[0] = xmin
	stw.Frame.Show.Xrange[1] = xmax
	stw.Frame.Show.Yrange[0] = ymin
	stw.Frame.Show.Yrange[1] = ymax
	stw.Frame.Show.Zrange[0] = zmin
	stw.Frame.Show.Zrange[1] = zmax
	stw.Labels["XMAX"].SetAttribute("VALUE", fmt.Sprintf("%.3f", xmax))
	stw.Labels["XMIN"].SetAttribute("VALUE", fmt.Sprintf("%.3f", xmin))
	stw.Labels["YMAX"].SetAttribute("VALUE", fmt.Sprintf("%.3f", ymax))
	stw.Labels["YMIN"].SetAttribute("VALUE", fmt.Sprintf("%.3f", ymin))
	stw.Labels["ZMAX"].SetAttribute("VALUE", fmt.Sprintf("%.3f", zmax))
	stw.Labels["ZMIN"].SetAttribute("VALUE", fmt.Sprintf("%.3f", zmin))
}

func (stw *Window) HideNotSelected() {
	if stw.SelectElem != nil {
		for _, n := range stw.Frame.Nodes {
			n.Hide()
		}
		for _, el := range stw.Frame.Elems {
			el.Hide()
		}
		for _, el := range stw.SelectElem {
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
	if stw.SelectElem != nil {
		for _, n := range stw.Frame.Nodes {
			n.Hide()
		}
		for _, el := range stw.SelectElem {
			if el != nil {
				el.Hide()
			}
		}
		for _, el := range stw.Frame.Elems {
			if !el.IsHidden(stw.Frame.Show) {
				for _, en := range el.Enod {
					en.Show()
				}
			}
		}
	}
	stw.SetShowRange()
	stw.Redraw()
}

func (stw *Window) LockNotSelected() {
	if stw.SelectElem != nil {
		for _, n := range stw.Frame.Nodes {
			n.Lock = true
		}
		for _, el := range stw.Frame.Elems {
			el.Lock = true
		}
		for _, el := range stw.SelectElem {
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
	if stw.SelectElem != nil {
		for _, el := range stw.SelectElem {
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
	if stw.SelectElem != nil {
		for _, el := range stw.SelectElem {
			if el != nil && !el.Lock {
				stw.Frame.DeleteElem(el.Num)
			}
		}
	}
	stw.Deselect()
	stw.Snapshot()
	stw.Redraw()
}

func (stw *Window) SelectNotHidden() {
	if stw.Frame == nil {
		return
	}
	stw.Deselect()
	stw.SelectElem = make([]*st.Elem, len(stw.Frame.Elems))
	num := 0
	for _, el := range stw.Frame.Elems {
		if el.IsHidden(stw.Frame.Show) {
			continue
		}
		stw.SelectElem[num] = el
		num++
	}
	stw.SelectElem = stw.SelectElem[:num]
	stw.Redraw()
}

func (stw *Window) CopyClipboard() error {
	var rtn error
	if stw.SelectElem == nil {
		return nil
	}
	var otp bytes.Buffer
	ns := stw.Frame.ElemToNode(stw.SelectElem...)
	getcoord(stw, func(x, y, z float64) {
		for _, n := range ns {
			otp.WriteString(n.CopyString(x, y, z))
		}
		for _, el := range stw.SelectElem {
			if el != nil {
				otp.WriteString(el.InpString())
			}
		}
		err := clipboard.WriteAll(otp.String())
		if err != nil {
			rtn = err
		}
		stw.addHistory(fmt.Sprintf("%d ELEMs Copied", len(stw.SelectElem)))
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
				nodemap, err = stw.Frame.ParseInp(tmp, coord, angle, nodemap, false)
				tmp = words
			}
			if err != nil {
				break
			}
		}
		nodemap, err = stw.Frame.ParseInp(tmp, coord, angle, nodemap, false)
		stw.EscapeCB()
	})
	return nil
}

func (stw *Window) ShapeData(sh st.Shape) {
	var tb *TextBox
	if t, tok := stw.TextBox["SHAPE"]; tok {
		tb = t
	} else {
		tb = NewTextBox()
		tb.Hide = false
		tb.Position = []float64{stw.CanvasSize[0] - 300.0, 200.0}
		stw.TextBox["SHAPE"] = tb
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
	tb.Value = strings.Split(otp.String(), "\n")
}

func (stw *Window) SectionData(sec *st.Sect) {
	var tb *TextBox
	if t, tok := stw.TextBox["SECTION"]; tok {
		tb = t
	} else {
		tb = NewTextBox()
		tb.Hide = false
		tb.Position = []float64{stw.CanvasSize[0] - 400.0, stw.CanvasSize[1] - 30.0}
		stw.TextBox["SECTION"] = tb
	}
	tb.Value = strings.Split(sec.InpString(), "\n")
	if al, ok := stw.Frame.Allows[sec.Num]; ok {
		tb.Value = append(tb.Value, strings.Split(al.String(), "\n")...)
	}
}

func (stw *Window) CurrentLap(comment string, nlap, laps int) {
	var tb *TextBox
	if t, tok := stw.TextBox["LAP"]; tok {
		tb = t
	} else {
		tb = NewTextBox()
		tb.Hide = false
		tb.Position = []float64{30.0, stw.CanvasSize[1] - 30.0}
		stw.TextBox["LAP"] = tb
	}
	if comment == "" {
		tb.Value = []string{fmt.Sprintf("LAP: %3d / %3d", nlap, laps)}
	} else {
		tb.Value = []string{comment, fmt.Sprintf("LAP: %3d / %3d", nlap, laps)}
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
		stw.addHistory(fmt.Sprintf("FILTER: %s", hstr))
		return rtn, nil
	} else {
		return els, errors.New("no filtering")
	}
}

func (stw *Window) ShowAll() {
	for _, el := range stw.Frame.Elems {
		el.Show()
	}
	for _, n := range stw.Frame.Nodes {
		n.Show()
	}
	for _, k := range stw.Frame.Kijuns {
		k.Show()
	}
	for i, et := range st.ETYPES {
		if i == st.WBRACE || i == st.SBRACE {
			continue
		}
		if lb, ok := stw.Labels[et]; ok {
			lb.SetAttribute("FGCOLOR", labelFGColor)
		}
		stw.Frame.Show.Etype[i] = true
	}
	stw.ShowAllSection()
	stw.Frame.Show.All()
	stw.Labels["XMIN"].SetAttribute("VALUE", fmt.Sprintf("%.3f", stw.Frame.Show.Xrange[0]))
	stw.Labels["XMAX"].SetAttribute("VALUE", fmt.Sprintf("%.3f", stw.Frame.Show.Xrange[1]))
	stw.Labels["YMIN"].SetAttribute("VALUE", fmt.Sprintf("%.3f", stw.Frame.Show.Yrange[0]))
	stw.Labels["YMAX"].SetAttribute("VALUE", fmt.Sprintf("%.3f", stw.Frame.Show.Yrange[1]))
	stw.Labels["ZMIN"].SetAttribute("VALUE", fmt.Sprintf("%.3f", stw.Frame.Show.Zrange[0]))
	stw.Labels["ZMAX"].SetAttribute("VALUE", fmt.Sprintf("%.3f", stw.Frame.Show.Zrange[1]))
	stw.Redraw()
}

func (stw *Window) UnlockAll() {
	for _, el := range stw.Frame.Elems {
		el.Lock = false
	}
	for _, n := range stw.Frame.Nodes {
		n.Lock = false
	}
	stw.Redraw()
}

func (stw *Window) ShowAtPaperCenter(canv *cd.Canvas) {
	for _, n := range stw.Frame.Nodes {
		stw.Frame.View.ProjectNode(n)
	}
	xmin, xmax, ymin, ymax := stw.Bbox()
	if xmax == xmin && ymax == ymin {
		return
	}
	w, h, err := stw.PaperSize(canv)
	if err != nil {
		stw.errormessage(err, ERROR)
		return
	}
	scale := math.Min(w/(xmax-xmin), h/(ymax-ymin)) * 0.9
	if stw.Frame.View.Perspective {
		stw.Frame.View.Dists[1] *= scale
	} else {
		stw.Frame.View.Gfact *= scale
	}
	cw, ch := canv.GetSize()
	stw.Frame.View.Center[0] = float64(cw)*0.5 + scale*(stw.Frame.View.Center[0]-0.5*(xmax+xmin))
	stw.Frame.View.Center[1] = float64(ch)*0.5 + scale*(stw.Frame.View.Center[1]-0.5*(ymax+ymin))
}

func (stw *Window) ShowAtCanvasCenter(canv *cd.Canvas) {
	for _, n := range stw.Frame.Nodes {
		stw.Frame.View.ProjectNode(n)
	}
	xmin, xmax, ymin, ymax := stw.Bbox()
	if xmax == xmin && ymax == ymin {
		return
	}
	w, h := canv.GetSize()
	scale := math.Min(float64(w)/(xmax-xmin), float64(h)/(ymax-ymin)) * 0.9
	if stw.Frame.View.Perspective {
		stw.Frame.View.Dists[1] *= scale
	} else {
		stw.Frame.View.Gfact *= scale
	}
	stw.Frame.View.Center[0] = float64(w)*0.5 + scale*(stw.Frame.View.Center[0]-0.5*(xmax+xmin))
	stw.Frame.View.Center[1] = float64(h)*0.5 + scale*(stw.Frame.View.Center[1]-0.5*(ymax+ymin))
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
	if stw.Frame != nil {
		stw.Frame.View.Angle[0] = phi
		stw.Frame.View.Angle[1] = theta
		stw.Redraw()
		stw.ShowCenter()
	}
}

// }}}

// Select// {{{
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

func (stw *Window) SelectNodeStart(arg *iup.MouseButton) {
	stw.dbuff.UpdateYAxis(&arg.Y)
	if arg.Pressed == 0 { // Released
		left := min(stw.startX, stw.endX)
		right := max(stw.startX, stw.endX)
		bottom := min(stw.startY, stw.endY)
		top := max(stw.startY, stw.endY)
		if (right-left < nodeSelectPixel) && (top-bottom < nodeSelectPixel) {
			n := stw.PickNode(left, bottom)
			if n != nil {
				stw.MergeSelectNode([]*st.Node{n}, isShift(arg.Status))
			} else {
				stw.SelectNode = make([]*st.Node, 0)
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

func (stw *Window) SelectElemStart(arg *iup.MouseButton) {
	stw.dbuff.UpdateYAxis(&arg.Y)
	if arg.Pressed == 0 { // Released
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
				stw.MergeSelectElem([]*st.Elem{el}, isShift(arg.Status))
			} else {
				stw.SelectElem = make([]*st.Elem, 0)
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
			stw.MergeSelectElem(tmpselectelem[:k], isShift(arg.Status))
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

func (stw *Window) SelectElemFenceStart(arg *iup.MouseButton) {
	stw.dbuff.UpdateYAxis(&arg.Y)
	if arg.Pressed == 0 { // Released
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
		stw.MergeSelectElem(tmpselectelem[:k], isShift(arg.Status))
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
			for m, el := range stw.SelectNode {
				if el == nodes[l] {
					if m == len(stw.SelectNode)-1 {
						stw.SelectNode = stw.SelectNode[:m]
					} else {
						stw.SelectNode = append(stw.SelectNode[:m], stw.SelectNode[m+1:]...)
					}
					break
				}
			}
		}
	} else {
		var add bool
		for l := 0; l < k; l++ {
			add = true
			for _, n := range stw.SelectNode {
				if n == nodes[l] {
					add = false
					break
				}
			}
			if add {
				stw.SelectNode = append(stw.SelectNode, nodes[l])
			}
		}
	}
}

func (stw *Window) MergeSelectElem(elems []*st.Elem, isshift bool) {
	k := len(elems)
	if isshift {
		for l := 0; l < k; l++ {
			for m, el := range stw.SelectElem {
				if el == elems[l] {
					if m == len(stw.SelectElem)-1 {
						stw.SelectElem = stw.SelectElem[:m]
					} else {
						stw.SelectElem = append(stw.SelectElem[:m], stw.SelectElem[m+1:]...)
					}
					break
				}
			}
		}
	} else {
		var add bool
		for l := 0; l < k; l++ {
			add = true
			for _, n := range stw.SelectElem {
				if n == elems[l] {
					add = false
					break
				}
			}
			if add {
				stw.SelectElem = append(stw.SelectElem, elems[l])
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
//     fmt.Printf("TAIL: %d %d %d %d %d %d\n", int(stw.SelectNode[0].Pcoord[0]), int(stw.SelectNode[0].Pcoord[1]), arg.GetX(), arg.GetY(), stw.endX, stw.endY)
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
	stw.SelectNode = make([]*st.Node, 0)
	stw.SelectElem = make([]*st.Elem, 0)
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
		if stw.Frame != nil {
			switch arg.Button {
			case BUTTON_LEFT:
				if isAlt(arg.Status) == ALTSELECTNODE {
					stw.SelectNodeStart(arg)
				} else {
					stw.SelectElemStart(arg)
					if isDouble(arg.Status) {
						if stw.SelectElem != nil && len(stw.SelectElem) > 0 {
							if stw.SelectElem[0].IsLineElem() {
								stw.execAliasCommand(DoubleClickCommand[0])
							} else {
								stw.execAliasCommand(DoubleClickCommand[1])
							}
						}
					}
				}
			case BUTTON_CENTER:
				if arg.Pressed == 0 { // Released
					stw.Redraw()
				} else { // Pressed
					if isDouble(arg.Status) {
						stw.Frame.SetFocus(nil)
						stw.DrawFrameNode()
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
									stw.exmode(stw.lastexcommand)
									stw.Redraw()
								}
							} else {
								if stw.lastcommand != nil {
									stw.ExecCommand(stw.lastcommand)
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
				stw.SetRecently()
			}
		}
	})
}

func (stw *Window) MoveOrRotate(arg *iup.MouseMotion) {
	if !fixMove && (isShift(arg.Status) || fixRotate) {
		stw.Frame.View.Center[0] += float64(int(arg.X)-stw.startX) * CanvasMoveSpeedX
		stw.Frame.View.Center[1] += float64(int(arg.Y)-stw.startY) * CanvasMoveSpeedY
		stw.DrawFrameNode()
	} else if !fixRotate {
		stw.Frame.View.Angle[0] -= float64(int(arg.Y)-stw.startY) * CanvasRotateSpeedY
		stw.Frame.View.Angle[1] -= float64(int(arg.X)-stw.startX) * CanvasRotateSpeedX
		stw.DrawFrameNode()
	}
}

func (stw *Window) CB_MouseMotion() {
	stw.canv.SetCallback(func(arg *iup.MouseMotion) {
		if stw.Frame != nil {
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
		if stw.Frame != nil {
			stw.dbuff.UpdateYAxis(&arg.Y)
			val := math.Pow(2.0, float64(arg.Delta)/CanvasScaleSpeed)
			x := arg.X
			if x > 65535 {
				x -= 65535
			}
			stw.Frame.View.Center[0] += (val - 1.0) * (stw.Frame.View.Center[0] - float64(x))
			stw.Frame.View.Center[1] += (val - 1.0) * (stw.Frame.View.Center[1] - float64(arg.Y))
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
		if stw.Frame != nil {
			switch stw.Frame.Project {
			default:
				stw.cline.SetAttribute("APPEND", "\"")
			case "venhira":
				stw.cline.SetAttribute("APPEND", "V4")
			}
		}
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
			tmp := stw.cline.GetAttribute("VALUE")
			stw.cline.SetAttribute("VALUE", stw.Complete(tmp))
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
			stw.New()
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
			stw.Reload()
		}
	case 'H':
		if key.IsCtrl() {
			stw.HideSelected()
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
	case 'L':
		if key.IsCtrl() {
			stw.LockSelected()
		}
	case 'U':
		if key.IsCtrl() {
			stw.UnlockAll()
		}
	case 'A':
		if key.IsCtrl() {
			// stw.ShowAll()
			stw.SelectNotHidden()
		}
	case 'R':
		if key.IsCtrl() {
			stw.ReadAll()
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
			stw.Redo()
		}
	case 'Z':
		if key.IsCtrl() {
			stw.Undo()
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
		if stw.Frame != nil && stw.SelectElem != nil {
			eval()
		}
	})
	stw.Props[index].SetCallback(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		switch key.Key() {
		case KEY_ENTER:
			if stw.Frame != nil && stw.SelectElem != nil {
				eval()
			}
		case KEY_TAB:
			if stw.Frame != nil && stw.SelectElem != nil {
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
			if sec, ok := stw.Frame.Sects[int(val)]; ok {
				for _, el := range stw.SelectElem {
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
		case re_column.MatchString(word):
			val = st.COLUMN
		case re_girder.MatchString(word):
			val = st.GIRDER
		case re_brace.MatchString(word):
			val = st.BRACE
		case re_wall.MatchString(word):
			val = st.WALL
		case re_slab.MatchString(word):
			val = st.SLAB
		}
		stw.Props[2].SetAttribute("VALUE", st.ETYPES[val])
		if val != 0 {
			for _, el := range stw.SelectElem {
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
	if stw.SelectElem != nil {
		var selected bool
		var lines, plates int
		var length, area float64
		for _, el := range stw.SelectElem {
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
						stw.Reload()
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
						stw.ReadAll()
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
						if stw.Changed {
							if stw.Yn("CHANGED", "変更を保存しますか") {
								stw.SaveAS()
							} else {
								return
							}
						}
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
						StartTool(filepath.Join(tooldir, "rclst/rclst.html"))
					},
				),
				iup.Item(
					iup.Attr("TITLE", "Fig2 Keyword"),
					func(arg *iup.ItemAction) {
						StartTool(filepath.Join(tooldir, "fig2/fig2.html"))
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
	if stw.Frame == nil {
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
	if n, ok := stw.Frame.Nlap[defpname]; ok {
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
		stw.Frame.Show.Period = pname
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
	if stw.Frame == nil {
		return
	}
	sects := make([]*st.Sect, len(stw.Frame.Sects))
	nsect := 0
	for _, sec := range stw.Frame.Sects {
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
				if _, exist := stw.Frame.Sects[int(val)]; exist {
					stw.addHistory(fmt.Sprintf("SECT: %d already exists", int(val)))
					return
				}
				sec := stw.Frame.AddSect(int(val))
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
					for _, el := range stw.Frame.Elems {
						if el.Sect.Num == sects[i].Num {
							continue deletesect
						}
					}
					stw.Frame.DeleteSect(sects[i].Num)
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
					stw.SectionProperty(stw.Frame.Sects[snum])
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
		if stw.Frame.Show.Sect[sec.Num] {
			ishide = "ON"
		} else {
			ishide = "OFF"
		}
		hides[i] = iup.Toggle(fmt.Sprintf("VALUE=%s", ishide),
			"CANFOCUS=NO",
			fmt.Sprintf("SIZE=%dx%d", 25, dataheight))
		func(snum, num int) {
			hides[num].SetCallback(func(arg *iup.ToggleAction) {
				if stw.Frame != nil {
					if arg.State == 1 {
						if selstart <= num && num <= selend {
							for j := selstart; j < selend+1; j++ {
								if hides[j].GetAttribute("ACTIVE") == "NO" {
									continue
								}
								stw.Frame.Show.Sect[sects[j].Num] = true
								hides[j].SetAttribute("VALUE", "ON")
							}
						} else {
							stw.Frame.Show.Sect[snum] = true
						}
					} else {
						if selstart <= num && num <= selend {
							for j := selstart; j < selend+1; j++ {
								if hides[j].GetAttribute("ACTIVE") == "NO" {
									continue
								}
								stw.Frame.Show.Sect[sects[j].Num] = false
								hides[j].SetAttribute("VALUE", "OFF")
							}
						} else {
							stw.Frame.Show.Sect[snum] = false
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
					stw.Frame.Sects[snum].Color = col
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
		f.Prop = stw.Frame.DefaultProp()
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
					if p, ok := stw.Frame.Props[int(num)]; ok {
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
					if p, ok := stw.Frame.Props[int(num)]; ok {
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
		stw.ExecCommand(command)
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
	for i, _ := range stw.Frame.Show.Sect {
		if lb, ok := stw.Labels[fmt.Sprintf("%d", i)]; ok {
			lb.SetAttribute("FGCOLOR", labelOFFColor)
		}
		stw.Frame.Show.Sect[i] = false
	}
}

func (stw *Window) ShowAllSection() {
	for i, _ := range stw.Frame.Show.Sect {
		if lb, ok := stw.Labels[fmt.Sprintf("%d", i)]; ok {
			lb.SetAttribute("FGCOLOR", labelFGColor)
		}
		stw.Frame.Show.Sect[i] = true
	}
}

func (stw *Window) HideSection(snum int) {
	if lb, ok := stw.Labels[fmt.Sprintf("%d", snum)]; ok {
		lb.SetAttribute("FGCOLOR", labelOFFColor)
	}
	stw.Frame.Show.Sect[snum] = false
}

func (stw *Window) ShowSection(snum int) {
	if lb, ok := stw.Labels[fmt.Sprintf("%d", snum)]; ok {
		lb.SetAttribute("FGCOLOR", labelFGColor)
	}
	stw.Frame.Show.Sect[snum] = true
}

func (stw *Window) HideEtype(etype int) {
	if etype == 0 {
		return
	}
	stw.Frame.Show.Etype[etype] = false
	if lbl, ok := stw.Labels[st.ETYPES[etype]]; ok {
		lbl.SetAttribute("FGCOLOR", labelOFFColor)
	}
}

func (stw *Window) ShowEtype(etype int) {
	if etype == 0 {
		return
	}
	stw.Frame.Show.Etype[etype] = true
	if lbl, ok := stw.Labels[st.ETYPES[etype]]; ok {
		lbl.SetAttribute("FGCOLOR", labelFGColor)
	}
}

func (stw *Window) ToggleEtype(etype int) {
	if stw.Frame.Show.Etype[etype] {
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
		if stw.Frame != nil {
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
	if stw.Frame != nil && stw.Frame.Show.Etype[etype] { // TODO: when stw.Frame is created, set value
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
		if stw.Frame != nil {
			switch arg.Button {
			case BUTTON_LEFT:
				if arg.Pressed == 0 { // Released
					if stw.Frame.Show.Etype[etype] {
						if !stw.Frame.Show.Etype[etype-2] {
							stw.Frame.Show.Etype[etype] = false
							stw.Frame.Show.Etype[etype-2] = true
							stw.Labels[st.ETYPES[etype]].SetAttribute("FGCOLOR", labelOFFColor)
							stw.Labels[st.ETYPES[etype-2]].SetAttribute("FGCOLOR", labelFGColor)
							for _, el := range stw.Frame.Elems {
								if el.Etype == etype {
									el.Hide()
								}
								if el.Etype == etype-2 {
									el.Show()
								}
							}
						} else {
							stw.Frame.Show.Etype[etype-2] = false
							stw.Labels[st.ETYPES[etype-2]].SetAttribute("FGCOLOR", labelOFFColor)
							for _, el := range stw.Frame.Elems {
								if el.Etype == etype-2 {
									el.Hide()
								}
							}
						}
					} else {
						if stw.Frame.Show.Etype[etype-2] {
							stw.Frame.Show.Etype[etype] = true
							stw.Frame.Show.Etype[etype-2] = false
							stw.Labels[st.ETYPES[etype]].SetAttribute("FGCOLOR", labelFGColor)
							stw.Labels[st.ETYPES[etype-2]].SetAttribute("FGCOLOR", labelOFFColor)
							for _, el := range stw.Frame.Elems {
								if el.Etype == etype {
									el.Show()
								}
								if el.Etype == etype-2 {
									el.Hide()
								}
							}
						} else {
							stw.Frame.Show.Etype[etype] = true
							stw.Labels[st.ETYPES[etype]].SetAttribute("FGCOLOR", labelFGColor)
							for _, el := range stw.Frame.Elems {
								if el.Etype == etype {
									el.Show()
								}
							}
						}
					}
					stw.Redraw()
					iup.SetFocus(stw.canv)
				}
			case BUTTON_CENTER:
				if arg.Pressed == 0 {
					stw.Frame.Show.Etype[etype] = false
					stw.Frame.Show.Etype[etype-2] = false
					stw.Labels[st.ETYPES[etype]].SetAttribute("FGCOLOR", labelOFFColor)
					stw.Labels[st.ETYPES[etype-2]].SetAttribute("FGCOLOR", labelOFFColor)
					for _, el := range stw.Frame.Elems {
						if el.Etype == etype {
							el.Hide()
						}
						if el.Etype == etype-2 {
							el.Hide()
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
		if stw.Frame != nil {
			switch arg.Button {
			case BUTTON_LEFT:
				if arg.Pressed == 0 { // Released
					if stw.Frame.Show.Stress[etype]&(1<<index) != 0 {
						stw.Frame.Show.Stress[etype] &= ^(1 << index)
						rtn.SetAttribute("FGCOLOR", labelOFFColor)
					} else {
						stw.Frame.Show.Stress[etype] |= (1 << index)
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
		if stw.Frame != nil {
			switch arg.Button {
			case BUTTON_LEFT:
				if arg.Pressed == 0 { // Released
					switch name {
					case "GAXIS":
						stw.Frame.Show.GlobalAxis = !stw.Frame.Show.GlobalAxis
					case "EAXIS":
						stw.Frame.Show.ElementAxis = !stw.Frame.Show.ElementAxis
					case "BOND":
						stw.Frame.Show.Bond = !stw.Frame.Show.Bond
					case "CONF":
						stw.Frame.Show.Conf = !stw.Frame.Show.Conf
					case "PHINGE":
						stw.Frame.Show.Phinge = !stw.Frame.Show.Phinge
					case "KIJUN":
						stw.Frame.Show.Kijun = !stw.Frame.Show.Kijun
					case "DEFORMATION":
						stw.Frame.Show.Deformation = !stw.Frame.Show.Deformation
					case "YIELD":
						stw.Frame.Show.YieldFunction = !stw.Frame.Show.YieldFunction
					}
					if rtn.GetAttribute("FGCOLOR") == labelFGColor {
						rtn.SetAttribute("FGCOLOR", labelOFFColor)
					} else {
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

func (stw *Window) SetPeriod(per string) {
	stw.Labels["PERIOD"].SetAttribute("VALUE", per)
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
		stw.Labels["PERIOD"].SetAttribute("VALUE", per)
		stw.Frame.Show.Period = per
	}
}

func (stw *Window) NodeCaptionOn(name string) {
	for i, j := range st.NODECAPTIONS {
		if j == name {
			if lbl, ok := stw.Labels[name]; ok {
				lbl.SetAttribute("FGCOLOR", labelFGColor)
			}
			if stw.Frame != nil {
				stw.Frame.Show.NodeCaptionOn(1 << uint(i))
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
			if stw.Frame != nil {
				stw.Frame.Show.NodeCaptionOff(1 << uint(i))
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
			if stw.Frame != nil {
				stw.Frame.Show.ElemCaptionOn(1 << uint(i))
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
			if stw.Frame != nil {
				stw.Frame.Show.ElemCaptionOff(1 << uint(i))
			}
		}
	}
}

func (stw *Window) StressOn(etype int, index uint) {
	stw.Frame.Show.Stress[etype] |= (1 << index)
	if etype <= st.SLAB {
		if lbl, ok := stw.Labels[fmt.Sprintf("%s_%s", st.ETYPES[etype], strings.ToUpper(st.StressName[index]))]; ok {
			lbl.SetAttribute("FGCOLOR", labelFGColor)
		}
	}
}

func (stw *Window) StressOff(etype int, index uint) {
	stw.Frame.Show.Stress[etype] &= ^(1 << index)
	if etype <= st.SLAB {
		if lbl, ok := stw.Labels[fmt.Sprintf("%s_%s", st.ETYPES[etype], strings.ToUpper(st.StressName[index]))]; ok {
			lbl.SetAttribute("FGCOLOR", labelOFFColor)
		}
	}
}

func (stw *Window) DeformationOn() {
	stw.Frame.Show.Deformation = true
	stw.Labels["DEFORMATION"].SetAttribute("FGCOLOR", labelFGColor)
}

func (stw *Window) DeformationOff() {
	stw.Frame.Show.Deformation = false
	stw.Labels["DEFORMATION"].SetAttribute("FGCOLOR", labelOFFColor)
}

func (stw *Window) DispOn(direction int) {
	name := fmt.Sprintf("NC_%s", st.DispName[direction])
	for i, str := range st.NODECAPTIONS {
		if name == str {
			stw.Frame.Show.NodeCaption |= (1 << uint(i))
			stw.Labels[name].SetAttribute("FGCOLOR", labelFGColor)
			return
		}
	}
}

func (stw *Window) DispOff(direction int) {
	name := fmt.Sprintf("NC_%s", st.DispName[direction])
	for i, str := range st.NODECAPTIONS {
		if name == str {
			stw.Frame.Show.NodeCaption &= ^(1 << uint(i))
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
	if stw.Frame != nil { // TODO: when stw.Frame is created, set value
		if on {
			switch ne {
			case "NODE":
				stw.Frame.Show.NodeCaption |= val
			case "ELEM":
				stw.Frame.Show.ElemCaption |= val
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
		if stw.Frame != nil {
			switch arg.Button {
			case BUTTON_LEFT:
				if arg.Pressed == 0 { // Released
					switch ne {
					case "NODE":
						if stw.Frame.Show.NodeCaption&val != 0 {
							rtn.SetAttribute("FGCOLOR", labelOFFColor)
							stw.Frame.Show.NodeCaption &= ^val
						} else {
							rtn.SetAttribute("FGCOLOR", labelFGColor)
							stw.Frame.Show.NodeCaption |= val
						}
					case "ELEM":
						if stw.Frame.Show.ElemCaption&val != 0 {
							rtn.SetAttribute("FGCOLOR", labelOFFColor)
							stw.Frame.Show.ElemCaption &= ^val
						} else {
							rtn.SetAttribute("FGCOLOR", labelFGColor)
							stw.Frame.Show.ElemCaption |= val
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
		if stw.Frame != nil {
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
					stw.Frame.Show.ColorMode = uint(next)
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
					stw.Frame.Show.ColorMode = uint(next)
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
		if stw.Frame != nil {
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
			if stw.Frame != nil {
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

func (stw *Window) CB_RangeValue(h *iup.Handle, valptr *float64) {
	h.SetCallback(func(arg *iup.CommonKillFocus) {
		if stw.Frame != nil {
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
			if stw.Frame != nil {
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
	stw.Frame.Show.ColorMode = mode
}

func (stw *Window) CB_Period(h *iup.Handle, valptr *string) {
	h.SetCallback(func(arg *iup.CommonKillFocus) {
		if stw.Frame != nil {
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
			if stw.Frame != nil {
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
	stw.CB_TextValue(stw.Labels["GFACT"], &stw.Frame.View.Gfact)
	stw.CB_TextValue(stw.Labels["DISTR"], &stw.Frame.View.Dists[0])
	stw.CB_TextValue(stw.Labels["DISTL"], &stw.Frame.View.Dists[1])
	stw.CB_TextValue(stw.Labels["PHI"], &stw.Frame.View.Angle[0])
	stw.CB_TextValue(stw.Labels["THETA"], &stw.Frame.View.Angle[1])
	stw.CB_TextValue(stw.Labels["FOCUSX"], &stw.Frame.View.Focus[0])
	stw.CB_TextValue(stw.Labels["FOCUSY"], &stw.Frame.View.Focus[1])
	stw.CB_TextValue(stw.Labels["FOCUSZ"], &stw.Frame.View.Focus[2])
	stw.CB_TextValue(stw.Labels["CENTERX"], &stw.Frame.View.Center[0])
	stw.CB_TextValue(stw.Labels["CENTERY"], &stw.Frame.View.Center[1])
	stw.CB_RangeValue(stw.Labels["XMAX"], &stw.Frame.Show.Xrange[1])
	stw.CB_RangeValue(stw.Labels["XMIN"], &stw.Frame.Show.Xrange[0])
	stw.CB_RangeValue(stw.Labels["YMAX"], &stw.Frame.Show.Yrange[1])
	stw.CB_RangeValue(stw.Labels["YMIN"], &stw.Frame.Show.Yrange[0])
	stw.CB_RangeValue(stw.Labels["ZMAX"], &stw.Frame.Show.Zrange[1])
	stw.CB_RangeValue(stw.Labels["ZMIN"], &stw.Frame.Show.Zrange[0])
	stw.CB_Period(stw.Labels["PERIOD"], &stw.Frame.Show.Period)
	stw.CB_TextValue(stw.Labels["GAXISSIZE"], &stw.Frame.Show.GlobalAxisSize)
	stw.CB_TextValue(stw.Labels["EAXISSIZE"], &stw.Frame.Show.ElementAxisSize)
	stw.CB_TextValue(stw.Labels["BONDSIZE"], &stw.Frame.Show.BondSize)
	stw.CB_TextValue(stw.Labels["CONFSIZE"], &stw.Frame.Show.ConfSize)
	stw.CB_TextValue(stw.Labels["DFACT"], &stw.Frame.Show.Dfact)
	stw.CB_TextValue(stw.Labels["MFACT"], &stw.Frame.Show.Mfact)
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

// }}}

func EditPgp() {
	cmd := exec.Command("cmd", "/C", "start", pgpfile)
	cmd.Start()
}

func (stw *Window) Analysis(fn string, arg string) error {
	var err error
	cmd := exec.Command(analysiscommand, arg, fn)
	err = cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	return err
}

func ReadPgp(filename string, aliases map[string]*Command) error {
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
					err := stw.exmode(currentcommand)
					if err != nil {
						stw.errormessage(err, ERROR)
					}
					val++
				}}
			} else {
				aliases[strings.ToUpper(words[0])] = &Command{"", "", "", func(stw *Window) {
					err := stw.exmode(command)
					if err != nil {
						stw.errormessage(err, ERROR)
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
					err := stw.fig2mode(command)
					if err != nil {
						stw.errormessage(err, ERROR)
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
	aliases = make(map[string]*Command, 0)
	sectionaliases = make(map[int]string, 0)
	err := ReadPgp(pgpfile, aliases)
	if err != nil {
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
