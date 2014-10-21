package stgui

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/visualfc/go-iup/iup"
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
	// "github.com/visualfc/go-iup/iupim"
	"github.com/atotto/clipboard"
	"github.com/visualfc/go-iup/cd"
	"github.com/yofu/st/stlib"
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
)

const (
	windowSize   = "FULLxFULL"
	nRecentFiles = 3
)

// Font
var (
	commandFontFace = "IPA明朝"
	commandFontSize = "11"
	labelFGColor    = "0 0 0"
	// labelBGColor = "255 255 255"
	labelBGColor         = "212 208 200"
	commandFGColor       = "0 127 31"
	commandBGColor       = "191 191 191"
	historyBGColor       = "220 220 220"
	sectionLabelColor    = "204 51 0"
	labelOFFColor        = "160 160 160"
	canvasFontFace       = "IPA明朝"
	canvasFontSize       = 9
	canvasFontColor      = cd.CD_DARK_GRAY
	DefaultTextAlignment = cd.CD_BASE_LEFT
	printFontColor       = cd.CD_BLACK
	printFontFace        = "IPA明朝"
	printFontSize        = 8
)

const (
	FONT_COMMAND = iota
	FONT_CANVAS
)

// Draw
var (
	first            = 1
	PlateEdgeColor   = cd.CD_GRAY
	BondColor        = cd.CD_GRAY
	ConfColor        = cd.CD_GRAY
	MomentColor      = cd.CD_DARK_MAGENTA
	StressTextColor  = cd.CD_GRAY
	YieldedTextColor = cd.CD_YELLOW
	BrittleTextColor = cd.CD_RED
	fixRotate        = false
	fixMove          = false
	colordlg *iup.Handle
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
	axrn_minmax = regexp.MustCompile("([+-]?[-0-9.]+)<=?([XYZxyz]{1})<=?([+-]?[-0-9.]+)")
	axrn_min1   = regexp.MustCompile("([+-]?[-0-9.]+)<=?([XYZxyz]{1})")
	axrn_min2   = regexp.MustCompile("([XYZxyz]{1})>=?([+-]?[-0-9.]+)")
	axrn_max1   = regexp.MustCompile("([+-]?[-0-9.]+)>=?([XYZxyz]{1})")
	axrn_max2   = regexp.MustCompile("([XYZxyz]{1})<=?([+-]?[-0-9.]+)")
	axrn_eq     = regexp.MustCompile("([XYZxyz]{1})=([+-]?[-0-9.]+)")
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

	CanvasSize []float64 // width, height

	zb *iup.Handle

	lselect *iup.Handle

	SelectNode []*st.Node
	SelectElem []*st.Elem

	PageTitle *TextBox
	Title     *TextBox
	Text      *TextBox
	TextBox   []*TextBox

	Version  string
	Modified string

	startT time.Time
	startX int
	startY int
	endX   int
	endY   int

	lastcommand *Command

	Labels map[string]*iup.Handle

	Property bool
	Selected []*iup.Handle
	Props    []*iup.Handle

	Changed bool // TODO: Set stw.Changed to true when stw.Frame is changed

	comhist     []string
	recentfiles []string
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
						// if stw.Changed {
						//     if stw.Yn("CHANGED", "変更を保存しますか") {
						//         stw.SaveAS()
						//     }
						// }
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
						stw.SectionDialog2()
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
				if strings.Contains(tmp, "%") {
					stw.cline.SetAttribute("VALUE", stw.Interpolate(tmp))
					stw.cline.SetAttribute("CARETPOS", "100")
				}
				arg.Return = int32(iup.IGNORE)
			case KEY_UPARROW:
				stw.PrevCommand()
			case KEY_DOWNARROW:
				stw.NextCommand()
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
		"NAME=histroy",
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
				stw.canv,
			),
			iup.Hbox(stw.hist),
			iup.Hbox(stw.cname, stw.cline, stw.coord),
		),
		func(arg *iup.DialogClose) {
			// if stw.Changed {
			//     if stw.Yn("CHANGED", "変更を保存しますか") {
			//         stw.SaveAS()
			//     }
			// }
			arg.Return = iup.CLOSE
		},
		func(arg *iup.CommonGetFocus) {
			if stw.Frame != nil {
				stw.Redraw()
			}
		},
	)
	stw.Dlg.Map()
	w, h := stw.cdcanv.GetSize()
	stw.CanvasSize = []float64{float64(w), float64(h)}
	stw.dbuff.TextAlignment(DefaultTextAlignment)
	stw.PageTitle = NewTextBox()
	stw.PageTitle.Font.Size = 16
	stw.PageTitle.Position = []float64{30.0, stw.CanvasSize[1] - 30.0}
	stw.Title = NewTextBox()
	stw.Title.Position = []float64{30.0, stw.CanvasSize[1] - 80.0}
	stw.Text = NewTextBox()
	stw.Text.Position = []float64{120.0, 65.0}
	stw.TextBox = make([]*TextBox, 0)
	iup.SetHandle("mainwindow", stw.Dlg)
	stw.EscapeAll()
	stw.Changed = false
	stw.comhist = make([]string, CommandHistorySize)
	comhistpos = -1
	stw.SetCoord(0.0, 0.0, 0.0)
	stw.recentfiles = make([]string, nRecentFiles)
	stw.SetRecently()
	return stw
}

// }}}

func (stw *Window) FocusCanv() {
	iup.SetFocus(stw.canv)
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

func UpdateInps(dirname string, brk chan bool) {
	Inps = make([]string, 0)
	SearchingInps = true
	SearchingInpsDone = make(chan bool)
	inpch := st.SearchInp(dirname)
createinps:
	for {
		select {
		case <-brk:
			break createinps
		case fn := <-inpch:
			if fn == "" {
				break createinps
			} else {
				Inps = append(Inps, fn)
			}
		}
	}
	SearchingInps = false
	SearchingInpsDone <- true
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
	brk := make(chan bool)
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
			stw.addHistory(err.Error())
		}
		if searchinginps_r {
			brk <- true
		}
		dlg.Destroy()
	}
	go func() {
	searchinp:
		for {
			if !SearchingInps {
				break
			}
			select {
			case <-SearchingInpsDone:
				updateresult()
				break searchinp
			}
		}
	}()
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
			if searchinginps_r {
				brk <- true
			}
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
					go UpdateInps(basedir.GetAttribute("VALUE"), brk)
					go func() {
					searchinp_r:
						for {
							if !searchinginps_r {
								break
							}
							select {
							case <-SearchingInpsDone:
								updateresult()
								searchinginps_r = false
								break searchinp_r
							}
						}
					}()
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
		v := stw.Frame.View.Copy()
		s := stw.Frame.Show.Copy()
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

func (stw *Window) OpenFile(fn string) error {
	var err error
	var s *st.Show
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
		err = frame.ReadInp(fn, []float64{0.0, 0.0, 0.0}, 0.0)
		if err != nil {
			return err
		}
		stw.Frame = frame
	case ".dxf":
		err = frame.ReadDxf(fn, []float64{0.0, 0.0, 0.0})
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
	}
	openstr := fmt.Sprintf("OPEN: %s", fn)
	stw.addHistory(openstr)
	stw.Dlg.SetAttribute("TITLE", stw.Frame.Name)
	stw.Frame.Home = stw.Home
	stw.LinkTextValue()
	stw.Changed = false
	stw.Cwd = filepath.Dir(fn)
	stw.AddRecently(fn)
	return nil
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
	err := stw.Frame.WriteInp(fn)
	if v != nil {
		stw.Frame.View = v
	}
	if err != nil {
		return err
	}
	savestr := fmt.Sprintf("SAVE: %s", fn)
	stw.addHistory(savestr)
	stw.Changed = false
	return nil
}

// }}}

func (stw *Window) Close(force bool) {
	// if stw.Changed {
	//     if stw.Yn("CHANGED", "変更を保存しますか") {
	//         stw.SaveAS()
	//     }
	// }
	stw.Dlg.Destroy()
}

// Read// {{{
func (stw *Window) Read() {
	if stw.Frame != nil {
		if name, ok := iup.GetOpenFile("", ""); ok {
			err := stw.ReadFile(name)
			if err != nil {
				fmt.Println(err)
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
				delete(stw.Frame.Elems, el.Num)
			case st.WALL, st.SLAB:
				el.Children = make([]*st.Elem, 2)
			}
		}
		exts := []string{".inl", ".ihx", ".ihy", ".otl", ".ohx", ".ohy", ".rat2", ".wgt", ".kjn"}
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
				fmt.Println(err)
			}
		}
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
		err = stw.Frame.ReadInp(filename, []float64{x, y, z}, 0.0)
	case ".inl", ".ihx", ".ihy":
		err = stw.Frame.ReadData(filename)
	case ".otl", ".ohx", ".ohy":
		err = stw.Frame.ReadResult(filename, st.UPDATE_RESULT)
	case ".rat", ".rat2":
		err = stw.Frame.ReadRat(filename)
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
		stw.addHistory(fmt.Sprintf("NOT READ: %s", filename))
		stw.addHistory(fmt.Sprintf(">>>> %s", err.Error()))
		return err
	}
	stw.addHistory(fmt.Sprintf("READ: %s", filename))
	return nil
}

func (stw *Window) ReadBucklingFile(filename string) error {
	var err error
	err = stw.Frame.ReadBuckling(filename)
	if err != nil {
		stw.addHistory(fmt.Sprintf("NOT READ: %s", filename))
		stw.addHistory(fmt.Sprintf(">>>> %s", err.Error()))
		return err
	}
	stw.addHistory(fmt.Sprintf("READ: %s", filename))
	return nil
}

func (stw *Window) ReadZoubunFile(filename string) error {
	var err error
	err = stw.Frame.ReadZoubun(filename)
	if err != nil {
		stw.addHistory(fmt.Sprintf("NOT READ: %s", filename))
		stw.addHistory(fmt.Sprintf(">>>> %s", err.Error()))
		return err
	}
	stw.addHistory(fmt.Sprintf("READ: %s", filename))
	return nil
}

func (stw *Window) AddResult(filename string, search bool) error {
	var err error
	if search {
		err = stw.Frame.ReadResult(filename, st.ADDSEARCH_RESULT)
	} else {
		err = stw.Frame.ReadResult(filename, st.ADD_RESULT)
	}
	if err != nil {
		stw.addHistory(fmt.Sprintf("NOT READ: %s", filename))
		stw.addHistory(fmt.Sprintf(">>>> %s", err.Error()))
		return err
	}
	stw.addHistory(fmt.Sprintf("READ: %s", filename))
	return nil
}

func (stw *Window) AddPropAndSect(filename string) error {
	if stw.Frame != nil {
		err := stw.Frame.AddPropAndSect(filename)
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

func (stw *Window) FittoPrinter(pcanv *cd.Canvas) (*st.View, float64) {
	v := stw.Frame.View.Copy()
	pw, ph := pcanv.GetSize()
	factor := math.Min(float64(pw)/stw.CanvasSize[0], float64(ph)/stw.CanvasSize[1])
	stw.CanvasSize[0] = float64(pw)
	stw.CanvasSize[1] = float64(ph)
	stw.Frame.View.Gfact *= factor
	stw.Frame.View.Center[0] = 0.5 * float64(pw)
	stw.Frame.View.Center[1] = 0.5 * float64(ph)
	stw.Frame.Show.ConfSize *= factor
	stw.Frame.Show.BondSize *= factor
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
	return v, factor
}

func (stw *Window) Print() {
	if stw.Frame == nil {
		return
	}
	pcanv, err := SetPrinter(stw.Frame.Name)
	if err != nil {
		stw.addHistory(err.Error())
		return
	}
	v, factor := stw.FittoPrinter(pcanv)
	switch stw.Frame.Show.ColorMode {
	default:
		stw.DrawFrame(pcanv, stw.Frame.Show.ColorMode, false)
	case st.ECOLOR_WHITE:
		stw.DrawFrame(pcanv, st.ECOLOR_BLACK, false)
	}
	stw.DrawTexts(pcanv, true)
	pcanv.Flush()
	pcanv.Kill()
	w, h := stw.cdcanv.GetSize()
	stw.CanvasSize = []float64{float64(w), float64(h)}
	stw.Frame.Show.ConfSize /= factor
	stw.Frame.Show.BondSize /= factor
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
	stw.Redraw()
}

func (stw *Window) PrintFig2(filename string) error {
	if stw.Frame == nil {
		return errors.New("PrintFig2: no frame opened")
	}
	pcanv, err := SetPrinter(stw.Frame.Name)
	if err != nil {
		return err
	}
	v, factor := stw.FittoPrinter(pcanv)
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

func (stw *Window) fig2keyword(lis []string, un bool) error {
	if len(lis) < 1 {
		return st.NotEnoughArgs("Fig2Keyword")
	}
	key := strings.ToUpper(lis[0])
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
		}
	case "GFACT":
		val, err := strconv.ParseFloat(lis[1], 64)
		if err != nil {
			return err
		}
		stw.Frame.View.Gfact = val
		stw.Labels["GFACT"].SetAttribute("VALUE", fmt.Sprintf("%f", stw.Frame.View.Gfact))
	case "FOCUS":
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
		case "ELEM":
			val, err := strconv.ParseInt(lis[2], 10, 64)
			if err != nil {
				return err
			}
			if el, ok := stw.Frame.Elems[int(val)]; ok {
				stw.Frame.SetFocus(el.MidPoint())
			}
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
	case "FIT":
		stw.ShowCenter()
	case "ANGLE":
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
	case "DISTS":
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
	case "PERSPECTIVE":
		stw.Frame.View.Perspective = true
	case "AXONOMETRIC":
		stw.Frame.View.Perspective = false
	case "DFACT":
		if len(lis) < 2 {
			return st.NotEnoughArgs("DFACT")
		}
		val, err := strconv.ParseFloat(lis[1], 64)
		if err != nil {
			return err
		}
		stw.Frame.Show.Dfact = val
		stw.Labels["DFACT"].SetAttribute("VALUE", fmt.Sprintf("%f", val))
	case "MFACT":
		if len(lis) < 2 {
			return st.NotEnoughArgs("MFACT")
		}
		val, err := strconv.ParseFloat(lis[1], 64)
		if err != nil {
			return err
		}
		stw.Frame.Show.Mfact = val
		stw.Labels["MFACT"].SetAttribute("VALUE", fmt.Sprintf("%f", val))
	case "GAXIS":
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
	case "EAXIS":
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
	case "NOAXIS":
		stw.Frame.Show.GlobalAxis = false
		stw.Labels["GAXIS"].SetAttribute("FGCOLOR", labelOFFColor)
		stw.Frame.Show.ElementAxis = false
		stw.Labels["EAXIS"].SetAttribute("FGCOLOR", labelOFFColor)
	case "ELEM":
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
	case "ELEM+":
		for _, val := range lis[1:] {
			et := strings.ToUpper(val)
			for i, e := range st.ETYPES {
				if et == e {
					stw.ShowEtype(i)
				}
			}
		}
	case "ELEM-":
		for _, val := range lis[1:] {
			et := strings.ToUpper(val)
			for i, e := range st.ETYPES {
				if et == e {
					stw.HideEtype(i)
				}
			}
		}
	case "SECTION":
		stw.HideAllSection()
		for _, tmp := range lis[1:] {
			val, err := strconv.ParseInt(tmp, 10, 64)
			if err != nil {
				continue
			}
			stw.ShowSection(int(val))
		}
	case "SECTION-":
		for _, tmp := range lis[1:] {
			val, err := strconv.ParseInt(tmp, 10, 64)
			if err != nil {
				continue
			}
			stw.HideSection(int(val))
		}
	case "KIJUN":
		if un {
			stw.Frame.Show.Kijun = false
			stw.Labels["KIJUN"].SetAttribute("FGCOLOR", labelOFFColor)
		} else {
			stw.Frame.Show.Kijun = true
			stw.Labels["KIJUN"].SetAttribute("FGCOLOR", labelFGColor)
		}
	case "ELEMCODE":
		if un {
			stw.ElemCaptionOff("EC_NUM")
		} else {
			stw.ElemCaptionOn("EC_NUM")
		}
	case "SECTCODE":
		if un {
			stw.ElemCaptionOff("EC_SECT")
		} else {
			stw.ElemCaptionOn("EC_SECT")
		}
	case "SRCANCOLOR":
		if un {
			stw.SetColorMode(st.ECOLOR_WHITE)
		} else {
			stw.SetColorMode(st.ECOLOR_RATE)
		}
	case "SRCANRATE":
		if un {
			stw.ElemCaptionOff("EC_RATE_L")
			stw.ElemCaptionOff("EC_RATE_S")
			stw.Labels["EC_RATE_L"].SetAttribute("FGCOLOR", labelOFFColor)
			stw.Labels["EC_RATE_S"].SetAttribute("FGCOLOR", labelOFFColor)
		} else {
			stw.ElemCaptionOn("EC_RATE_L")
			stw.ElemCaptionOn("EC_RATE_S")
			stw.Labels["EC_RATE_L"].SetAttribute("FGCOLOR", labelFGColor)
			stw.Labels["EC_RATE_S"].SetAttribute("FGCOLOR", labelFGColor)
		}
	case "STRESS":
		l := len(lis)
		if l < 2 {
			return st.NotEnoughArgs("STRESS")
		}
		etype := -1
		et := strings.ToUpper(lis[1])
		for i, e := range st.ETYPES {
			if et == e {
				etype = i
			}
		}
		if etype == -1 {
			break
		}
		if l < 3 && un {
			for i := 0; i < 6; i++ {
				stw.StressOff(etype, uint(i))
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
			stw.StressOff(etype, uint(index))
		} else {
			stw.StressOn(etype, uint(index))
		}
	case "DEFORMATION":
		if un {
			stw.DeformationOff()
		} else {
			if len(lis) >= 2 {
				stw.SetPeriod(strings.ToUpper(lis[1]))
			}
			stw.DeformationOn()
		}
	case "DISP":
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
	case "ECCENTRIC":
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
	case "ALIAS":
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
					sectionaliases[int(val)] = lis[2]
				}
			}
		}
	case "ANONYMOUS":
		for _, str := range lis[1:] {
			val, err := strconv.ParseInt(str, 10, 64)
			if err != nil {
				continue
			}
			if _, ok := stw.Frame.Sects[int(val)]; ok {
				sectionaliases[int(val)] = ""
			}
		}
	case "NODECODE":
		if un {
			stw.NodeCaptionOff("NC_NUM")
		} else {
			stw.NodeCaptionOn("NC_NUM")
		}
	case "CONF":
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
	case "PILE":
		if un {
			stw.NodeCaptionOff("NC_PILE")
		} else {
			stw.NodeCaptionOn("NC_PILE")
		}
	case "FENCE":
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
	case "PERIOD":
		stw.SetPeriod(strings.ToUpper(lis[1]))
	case "NOMOMENTVALUE":
		if un {
			stw.Frame.Show.NoMomentValue = false
		} else {
			stw.Frame.Show.NoMomentValue = true
		}
	case "NCOLOR":
		stw.SetColorMode(st.ECOLOR_N)
	case "PAGETITLE":
		if un {
			stw.PageTitle.Value = make([]string, 0)
			stw.PageTitle.Hide = true
		} else {
			stw.PageTitle.Value = append(stw.PageTitle.Value, strings.Join(lis[1:], " "))
			stw.PageTitle.Hide = false
		}
	case "TITLE":
		if un {
			stw.Title.Value = make([]string, 0)
			stw.Title.Hide = true
		} else {
			stw.Title.Value = append(stw.Title.Value, strings.Join(lis[1:], " "))
			stw.Title.Hide = false
		}
	case "TEXT":
		if un {
			stw.Text.Value = make([]string, 0)
			stw.Text.Hide = true
		} else {
			stw.Text.Value = append(stw.Text.Value, strings.Join(lis[1:], " "))
			stw.Text.Hide = false
		}
	case "POSITION":
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
		return filepath.Join(stw.Home, fn[:pos1], fn[:pos2], fn), nil
	}
}

// Help// {{{
func ShowReleaseNote() {
	cmd := exec.Command("cmd", "/C", "start", releasenote)
	cmd.Start()
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
	iup.SetFocus(stw.canv)
}

func (stw *Window) addCommandHistory(str string) {
	tmp := make([]string, CommandHistorySize)
	tmp[0] = str
	for i := 0; i < CommandHistorySize-1; i++ {
		tmp[i+1] = stw.comhist[i]
	}
	stw.comhist = tmp
}

func (stw *Window) PrevCommand() {
	if comhistpos < CommandHistorySize {
		comhistpos++
		stw.cline.SetAttribute("VALUE", stw.comhist[comhistpos])
	}
}

func (stw *Window) NextCommand() {
	if comhistpos > 0 {
		comhistpos--
		stw.cline.SetAttribute("VALUE", stw.comhist[comhistpos])
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
			stw.exmode(al)
		} else {
			stw.Open()
		}
		return
	}
	alu := strings.ToUpper(al)
	if value, ok := aliases[alu]; ok {
		stw.ExecCommand(value)
	} else if value, ok := Commands[alu]; ok {
		stw.ExecCommand(value)
	} else {
		switch {
		default:
			stw.addHistory(fmt.Sprintf("command doesn't exist: %s", al))
		case strings.HasPrefix(al, ":"):
			stw.exmode(al)
		case strings.HasPrefix(al, "'"):
			if len(al) == 1 {
				break
			}
			al = al[1:]
			var un bool
			if strings.HasPrefix(al, "!") {
				un = true
				al = al[1:]
			} else {
				un = false
			}
			tmpargs := strings.Split(al, " ")
			args := make([]string, len(tmpargs))
			narg := 0
			for i := 0; i < len(tmpargs); i++ {
				if tmpargs[i] != "" {
					args[narg] = tmpargs[i]
					narg++
				}
			}
			args = args[:narg]
			stw.fig2keyword(args, un)
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
	iup.SetFocus(stw.canv)
	return
}

func (stw *Window) Interpolate(str string) string {
	str = strings.Replace(str, "%:h", stw.Cwd, 1)
	if stw.Frame == nil {
		return str
	}
	str = strings.Replace(str, "%<", st.PruneExt(stw.Frame.Path), 1)
	str = strings.Replace(str, "%", stw.Frame.Path, 1)
	return str
}

func (stw *Window) exmode(command string) {
	if len(command) == 1 {
		return
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
	var fn string
	if narg < 2 {
		fn = ""
	} else {
		if ok, _ := regexp.MatchString("^#[0-9]+$", args[1]); ok {
			val, _ := strconv.ParseInt(strings.TrimPrefix(args[1], "#"), 10, 64)
			if int(val) < nRecentFiles {
				fn = stw.recentfiles[int(val)]
			} else {
				fn = ""
			}
		} else if strings.Contains(args[1], "%") {
			fn = stw.Interpolate(args[1])
		} else {
			if filepath.Dir(args[1]) == "." {
				fn = filepath.Join(stw.Cwd, args[1])
			} else {
				fn = args[1]
			}
		}
	}
	bang := strings.HasSuffix(args[0], "!")
	cname := strings.TrimSuffix(strings.TrimPrefix(args[0], ":"), "!")
	if stw.Frame != nil {
		switch cname {
		case "w", "sav":
			if fn == "" {
				stw.SaveFile(stw.Frame.Path)
			} else {
				if bang || (!st.FileExists(fn) || stw.Yn("Save", "上書きしますか")) {
					err := stw.SaveFile(fn)
					if err == nil && fn != stw.Frame.Path {
						stw.Copylsts(fn)
					}
					if cname == "sav" {
						stw.Rebase(fn)
					}
				}
			}
		case "inc":
			var times int
			if narg >= 2 {
				val, err := strconv.ParseInt(args[1], 10, 64)
				if err != nil {
					stw.addHistory(err.Error())
					times = 1
				} else {
					times = int(val)
				}
			} else {
				times = 1
			}
			fn, err := st.Increment(stw.Frame.Path, "_", 1, times)
			if err != nil {
				stw.addHistory(err.Error())
				break
			}
			if bang || (!st.FileExists(fn) || stw.Yn("Save", "上書きしますか")) {
				err := stw.SaveFile(fn)
				if err == nil && fn != stw.Frame.Path {
					stw.Copylsts(fn)
				}
				stw.Rebase(fn)
				stw.EditReadme(filepath.Dir(fn))
			}
		case "e":
			// if !bang && stw.Changed {
			//     if stw.Yn("CHANGED", "変更を保存しますか") {
			//         stw.SaveAS()
			//     }
			// }
			if fn != "" {
				if !st.FileExists(fn) {
					sfn, err := stw.SearchFile(args[1])
					if err != nil {
						stw.addHistory(err.Error())
						break
					}
					stw.OpenFile(sfn)
					stw.Redraw()
				} else {
					stw.OpenFile(fn)
					stw.Redraw()
				}
			} else {
				stw.Reload()
			}
		case "q":
			stw.Close(bang)
		case "c":
			checkframe(stw)
			stw.addHistory("CHECKED")
		case "#":
			stw.ShowRecently()
		case "vim":
			stw.Vim(fn)
		case "read":
			stw.ReadFile(fn)
		case "insert":
			if narg > 2 && len(stw.SelectNode) >= 1 {
				angle, err := strconv.ParseFloat(args[2], 64)
				if err != nil {
					stw.addHistory(err.Error())
				}
				err = stw.Frame.ReadInp(fn, stw.SelectNode[0].Coord, angle*math.Pi/180.0)
				if err != nil {
					stw.addHistory(err.Error())
				}
				stw.EscapeAll()
			}
		case "ps":
			stw.AddPropAndSect(fn)
		case "rb":
			stw.ReadBucklingFile(fn)
		case "rz":
			stw.ReadZoubunFile(fn)
		case "add":
			stw.AddResult(fn, false)
		case "adds":
			stw.AddResult(fn, true)
		case "wo":
			if narg < 3 {
				stw.addHistory("Not enough arguments")
			} else {
				stw.Frame.WriteOutput(fn, args[2])
			}
		case "fig2":
			err := stw.ReadFig2(fn)
			if err != nil {
				fmt.Println(err)
			}
		case "conf":
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
				stw.addHistory("Not enough arguments")
			}
		case "pile":
			if stw.SelectNode == nil || len(stw.SelectNode) == 0 {
				return
			}
			if narg < 2 {
				for _, n := range stw.SelectNode {
					n.Pile = nil
				}
				return
			}
			val, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return
			}
			if p, ok := stw.Frame.Piles[int(val)]; ok {
				for _, n := range stw.SelectNode {
					n.Pile = p
				}
			} else {
				stw.addHistory(fmt.Sprintf("PILE %d doesn't exist", val))
			}
		case "an":
			stw.SaveFile(stw.Frame.Path)
			var anarg string
			if narg >= 3 {
				anarg = args[2]
			} else {
				anarg = "-a"
			}
			err := stw.Analysis(filepath.ToSlash(stw.Frame.Path), anarg)
			if err != nil {
				stw.addHistory("Analysis Failed")
			} else {
				stw.Reload()
				stw.ReadAll()
				stw.Redraw()
			}
		case "f":
			stw.FilterSelectedElem(strings.Join(args[1:], " "))
		case "ht":
			if narg == 1 {
				axisrange(stw, 2, -100.0, 1000.0, false)
			}
			if narg < 3 {
				return
			}
			tmp, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return
			}
			min := int(tmp)
			tmp, err = strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return
			}
			max := int(tmp)
			l := len(stw.Frame.Ai.Boundary)
			if min < 0 || min > l || max < 0 || max > l {
				return
			}
			axisrange(stw, 2, stw.Frame.Ai.Boundary[min], stw.Frame.Ai.Boundary[max], false)
		case "ht+":
			stw.NextFloor()
		case "ht-":
			stw.PrevFloor()
		case "section+":
			if narg < 2 {
				return
			}
			tmp, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return
			}
			if tmp == 0 {
				return
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
		case "view":
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
		case "fixr":
			fixRotate = !fixRotate
		case "fixm":
			fixMove = !fixMove
		}
	} else {
		switch cname {
		case "q":
			stw.Close(bang)
		case "e":
			if fn != "" {
				if !st.FileExists(fn) {
					sfn, err := stw.SearchFile(args[1])
					if err != nil {
						stw.addHistory(err.Error())
						break
					}
					stw.OpenFile(sfn)
					stw.Redraw()
				} else {
					stw.OpenFile(fn)
					stw.Redraw()
				}
			} else {
				stw.Open()
			}
		case "vim":
			stw.Vim(fn)
		}
	}
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
	stw.UpdateShowRange()
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
	stw.UpdateShowRange()
	stw.Redraw()
}

func axisrange(stw *Window, axis int, min, max float64, any bool) {
	tmpnodes := make([]*st.Node, 0)
	for _, n := range stw.Frame.Nodes {
		if !(min <= n.Coord[axis] && n.Coord[axis] <= max) {
			tmpnodes = append(tmpnodes, n)
			n.Hide = true
		} else {
			n.Hide = false
		}
	}
	var tmpelems []*st.Elem
	if !any {
		tmpelems = stw.Frame.NodeToElemAny(tmpnodes...)
	} else {
		tmpelems = stw.Frame.NodeToElemAll(tmpnodes...)
	}
	for _, el := range stw.Frame.Elems {
		el.Hide = false
	}
	for _, el := range tmpelems {
		el.Hide = true
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
	if stw.Frame != nil {
		// stw.UpdateShowRange() // TODO: ShowRange
		canv.Hatch(cd.CD_FDIAGONAL)
		canv.Clear()
		stw.Frame.View.Set(0)
		if stw.Frame.Show.GlobalAxis {
			stw.DrawGlobalAxis(canv)
		}
		if stw.Frame.Show.Kijun {
			canv.TextAlignment(cd.CD_CENTER)
			canv.Foreground(cd.CD_GRAY)
			for _, k := range stw.Frame.Kijuns {
				if k.Hide {
					continue
				}
				k.Pstart = stw.Frame.View.ProjectCoord(k.Start)
				k.Pend = stw.Frame.View.ProjectCoord(k.End)
				DrawKijun(k, canv, stw.Frame.Show)
			}
			canv.TextAlignment(DefaultTextAlignment)
		}
		canv.Foreground(cd.CD_WHITE)
		for _, n := range stw.Frame.Nodes {
			stw.Frame.View.ProjectNode(n)
			if stw.Frame.Show.Deformation {
				stw.Frame.View.ProjectDeformation(n, stw.Frame.Show)
			}
			if n.Hide {
				continue
			}
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
			DrawNode(n, canv, stw.Frame.Show)
		}
		canv.LineStyle(cd.CD_CONTINUOUS)
		canv.Hatch(cd.CD_FDIAGONAL)
		if !stw.Frame.Show.Select {
			els := st.SortedElem(stw.Frame.Elems, func(e *st.Elem) float64 { return -e.DistFromProjection() })
		loop:
			for _, el := range els {
				if el.IsHide(stw.Frame.Show) {
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
						canv.Foreground(st.Rainbow(el.RateMax(stw.Frame.Show), st.RateBoundary))
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
							Ix, err := el.Sect.Ix()
							if err != nil {
								canv.Foreground(cd.CD_WHITE)
							}
							Iy, err := el.Sect.Iy()
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
		for _, el := range stw.SelectElem {
			canv.LineStyle(cd.CD_DOTTED)
			if el == nil || el.IsHide(stw.Frame.Show) {
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
					canv.Foreground(st.Rainbow(el.RateMax(stw.Frame.Show), st.RateBoundary))
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
		if stw.Frame.Fes != nil {
			DrawEccentric(stw.Frame, canv, stw.Frame.Show)
		}
		if flush {
			canv.Flush()
		}
		stw.SetViewData()
	}
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
	if stw.Frame != nil {
		stw.dbuff.Clear()
		stw.Frame.View.Set(0)
		if stw.Frame.Show.GlobalAxis {
			stw.DrawGlobalAxis(stw.dbuff)
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
			stw.dbuff.Hatch(cd.CD_DIAGCROSS)
			for _, el := range stw.SelectElem {
				stw.dbuff.LineStyle(cd.CD_DOTTED)
				if el == nil || el.IsHide(stw.Frame.Show) {
					continue
				}
				if el.Lock {
					stw.dbuff.Foreground(LOCKED_ELEM_COLOR)
				} else {
					stw.dbuff.Foreground(cd.CD_WHITE)
				}
				DrawElem(el, stw.dbuff, stw.Frame.Show)
			}
			stw.dbuff.LineStyle(cd.CD_CONTINUOUS)
			stw.dbuff.Hatch(cd.CD_FDIAGONAL)
		}
		stw.dbuff.Flush()
	}
}

func (stw *Window) DrawGlobalAxis(canv *cd.Canvas) {
	origin := stw.Frame.View.ProjectCoord([]float64{0.0, 0.0, 0.0})
	xaxis := stw.Frame.View.ProjectCoord([]float64{stw.Frame.Show.GlobalAxisSize, 0.0, 0.0})
	yaxis := stw.Frame.View.ProjectCoord([]float64{0.0, stw.Frame.Show.GlobalAxisSize, 0.0})
	zaxis := stw.Frame.View.ProjectCoord([]float64{0.0, 0.0, stw.Frame.Show.GlobalAxisSize})
	size := 0.3
	theta := 10.0 * math.Pi / 180.0
	canv.LineStyle(cd.CD_CONTINUOUS)
	canv.Foreground(cd.CD_RED)
	Arrow(canv, origin[0], origin[1], xaxis[0], xaxis[1], size, theta)
	canv.Foreground(cd.CD_GREEN)
	Arrow(canv, origin[0], origin[1], yaxis[0], yaxis[1], size, theta)
	canv.Foreground(cd.CD_BLUE)
	Arrow(canv, origin[0], origin[1], zaxis[0], zaxis[1], size, theta)
	canv.Foreground(cd.CD_WHITE)
}

func Arrow(cvs *cd.Canvas, x1, y1, x2, y2, size, theta float64) {
	c := size * math.Cos(theta)
	s := size * math.Sin(theta)
	cvs.FLine(x1, y1, x2, y2)
	cvs.FLine(x2, y2, x2+((x1-x2)*c-(y1-y2)*s), y2+((x1-x2)*s+(y1-y2)*c))
	cvs.FLine(x2, y2, x2+((x1-x2)*c+(y1-y2)*s), y2+(-(x1-x2)*s+(y1-y2)*c))
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
	cworigin := []float64{-1000.0, cwnodes[0].Pcoord[1]}
	ccwnodes := make([]*st.Node, 1)
	ccwnodes[0] = hstart
	ccworigin := []float64{1000.0, ccwnodes[0].Pcoord[1]}
	stopcw := false
	stopccw := false
	ncw := 0
	nccw := 0
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
				if ncw == 0 {
					cwangle, cw = st.ClockWise(cworigin, cwnodes[ncw].Pcoord, n.Pcoord)
				} else {
					cwangle, cw = st.ClockWise(cwnodes[ncw-1].Pcoord, cwnodes[ncw].Pcoord, n.Pcoord)
				}
				cwangle = math.Abs(cwangle)
				// if n.Num == 727 {
				//     fmt.Printf("CWANGLE = %.3f\n", cwangle)
				// }
				if cw && cwangle < cwminangle {
					tmpcw = n
					cwminangle = cwangle
				}
			}
			if !stopccw {
				if nccw == 0 {
					ccwangle, cw = st.ClockWise(ccworigin, ccwnodes[nccw].Pcoord, n.Pcoord)
				} else {
					ccwangle, cw = st.ClockWise(ccwnodes[nccw-1].Pcoord, ccwnodes[nccw].Pcoord, n.Pcoord)
				}
				ccwangle = math.Abs(ccwangle)
				// if n.Num == 727 {
				//     fmt.Printf("CCWANGLE = %.3f\n", ccwangle)
				// }
				if !cw && ccwangle < ccwminangle {
					tmpccw = n
					ccwminangle = ccwangle
				}
			}
		}
		fmt.Printf("CW: %d, CCW: %d\n", tmpcw.Num, tmpccw.Num)
		if !stopcw && tmpcw != nil {
			fmt.Printf("CW: %d, ", tmpcw.Num)
			ncw++
			cwnodes = append(cwnodes, tmpcw)
			if tmpcw == hend {
				stopcw = true
			}
		} else {
			fmt.Println("CW: -, ")
		}
		if !stopccw && tmpccw != nil {
			fmt.Printf("CCW: %d, ", tmpccw.Num)
			nccw++
			ccwnodes = append(ccwnodes, tmpccw)
			if tmpccw == hend {
				stopccw = true
			}
		} else {
			fmt.Println("CCW: -")
		}
		fmt.Println(stopcw, stopccw)
		if stopcw && stopccw {
			break
		}
	}
	chnodes := make([]*st.Node, ncw+nccw-2)
	fmt.Println(ncw, len(cwnodes))
	fmt.Println(nccw, len(ccwnodes))
	fmt.Println(len(chnodes))
	for i := 0; i < ncw; i++ {
		chnodes[i] = cwnodes[i]
	}
	for i := nccw - 2; i > 0; i-- {
		chnodes[i+ncw] = ccwnodes[i]
	}
	stw.dbuff.Begin(cd.CD_FILL)
	stw.dbuff.Foreground(PlateEdgeColor)
	stw.dbuff.Begin(cd.CD_CLOSED_LINES)
	for i := 0; i < ncw+nccw-2; i++ {
		stw.dbuff.FVertex(chnodes[i].Pcoord[0], chnodes[i].Pcoord[1])
	}
	stw.dbuff.End()
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
	var mins, maxs [2]float64
	first := true
	for _, j := range stw.Frame.Nodes {
		if j.Hide {
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

func (stw *Window) HideNodes() {
	for _, n := range stw.Frame.Nodes {
		n.Hide = true
	}
	for _, el := range stw.Frame.Elems {
		if el.Hide {
			continue
		}
		for _, en := range el.Enod {
			en.Hide = false
		}
	}
	stw.SetShowRange()
}

func (stw *Window) HideNotSelected() {
	if stw.SelectElem != nil {
		for _, n := range stw.Frame.Nodes {
			n.Hide = true
		}
		for _, el := range stw.Frame.Elems {
			el.Hide = true
		}
		for _, el := range stw.SelectElem {
			if el != nil {
				el.Hide = false
				for _, en := range el.Enod {
					en.Hide = false
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
			n.Hide = true
		}
		for _, el := range stw.SelectElem {
			if el != nil {
				el.Hide = true
			}
		}
		for _, el := range stw.Frame.Elems {
			if !el.Hide {
				for _, en := range el.Enod {
					en.Hide = false
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
				delete(stw.Frame.Elems, el.Num)
			}
		}
	}
	stw.Deselect()
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
				nodemap, err = stw.Frame.ParseInp(tmp, coord, angle, nodemap)
				tmp = words
			}
			if err != nil {
				break
			}
		}
		nodemap, err = stw.Frame.ParseInp(tmp, coord, angle, nodemap)
		stw.EscapeCB()
	})
	return nil
}

func (stw *Window) FilterSelectedElem(str string) {
	l := len(stw.SelectElem)
	if stw.SelectElem == nil || l == 0 {
		return
	}
	parallel := regexp.MustCompile("(?i)^ *// *([xyz]{1})")
	ortho := regexp.MustCompile("^ *TT *([xyzXYZ]{1})")
	sectnum := regexp.MustCompile("^ *sect? *==? *([0-9]+)")
	etypestr := regexp.MustCompile("^ *et(y(p(e?)?)?) *==? *([a-zA-Z]+)")
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
	case sectnum.MatchString(str):
		fs := sectnum.FindStringSubmatch(str)
		if len(fs) < 2 {
			break
		}
		tmp, err := strconv.ParseInt(fs[1], 10, 64)
		if err != nil {
			break
		}
		snum := int(tmp)
		filterfunc = func(el *st.Elem) bool {
			return el.Sect.Num == snum
		}
		hstr = fmt.Sprintf("Sect == %d", snum)
	case etypestr.MatchString(str):
		fs := etypestr.FindStringSubmatch(str)
		l := len(fs)
		if l >= 4 {
			col := regexp.MustCompile("(?i)co(l(u(m(n)?)?)?)?$")
			gir := regexp.MustCompile("(?i)gi(r(d(e(r)?)?)?)?$")
			bra := regexp.MustCompile("(?i)br(a(c(e)?)?)?$")
			wal := regexp.MustCompile("(?i)wa(l){0,2}$")
			sla := regexp.MustCompile("(?i)sl(a(b)?)?$")
			var val int
			switch {
			case col.MatchString(fs[l-1]):
				val = st.COLUMN
			case gir.MatchString(fs[l-1]):
				val = st.GIRDER
			case bra.MatchString(fs[l-1]):
				val = st.BRACE
			case wal.MatchString(fs[l-1]):
				val = st.WALL
			case sla.MatchString(fs[l-1]):
				val = st.SLAB
			}
			filterfunc = func(el *st.Elem) bool {
				return el.Etype == val
			}
			hstr = fmt.Sprintf("Etype == %s", st.ETYPES[val])
		} else {
			break
		}
	}
	if filterfunc != nil {
		tmpels := make([]*st.Elem, l)
		enum := 0
		for _, el := range stw.SelectElem {
			if el == nil {
				continue
			}
			if filterfunc(el) {
				tmpels[enum] = el
				enum++
			}
		}
		stw.SelectElem = tmpels[:enum]
		stw.addHistory(fmt.Sprintf("FILTER: %s", hstr))
		stw.Redraw()
	}
}

func (stw *Window) ShowAll() {
	for _, el := range stw.Frame.Elems {
		el.Hide = false
	}
	for _, n := range stw.Frame.Nodes {
		n.Hide = false
	}
	for _, k := range stw.Frame.Kijuns {
		k.Hide = false
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

func (stw *Window) ShowAtCanvasCenter(canv *cd.Canvas) {
	for _, n := range stw.Frame.Nodes {
		stw.Frame.View.ProjectNode(n)
	}
	xmin, xmax, ymin, ymax := stw.Bbox()
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
	stw.ShowAtCanvasCenter(stw.cdcanv)
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
		if v.Hide {
			continue
		}
		dist := math.Hypot(float64(x)-v.Pcoord[0], float64(y)-v.Pcoord[1])
		if dist < mindist {
			mindist = dist
			rtn = v
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
			stw.MergeSelectNode([]*st.Node{n}, isShift(arg.Status))
		} else {
			tmpselect := make([]*st.Node, len(stw.Frame.Nodes))
			i := 0
			for _, v := range stw.Frame.Nodes {
				if v.Hide {
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
		if v.IsHide(stw.Frame.Show) {
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
		if el.IsHide(stw.Frame.Show) {
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
	return st.SortedElem(rtn, func(e *st.Elem) float64 { return e.DistFromProjection() })
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
					if el.IsHide(stw.Frame.Show) {
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
					if el.IsHide(stw.Frame.Show) {
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
			if el.IsHide(stw.Frame.Show) {
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
				if isAlt(arg.Status) {
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
							if stw.lastcommand != nil {
								stw.ExecCommand(stw.lastcommand)
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
				if isAlt(arg.Status) {
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

func (stw *Window) DefaultKeyAny(key iup.KeyState) {
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
			if strings.Contains(tmp, "%") {
				stw.cline.SetAttribute("VALUE", stw.Interpolate(tmp))
				stw.cline.SetAttribute("CARETPOS", "100")
			}
		}
	case KEY_UPARROW:
		if key.IsCtrl() {
			stw.NextFloor()
		} else {
			stw.PrevCommand()
		}
	case KEY_DOWNARROW:
		if key.IsCtrl() {
			stw.PrevFloor()
		} else {
			stw.NextCommand()
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
			stw.Save()
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
	case 'C':
		if key.IsCtrl() {
			stw.CopyClipboard()
		}
	case 'V':
		if key.IsCtrl() {
			stw.PasteClipboard()
		}
	}
}

func (stw *Window) CB_CommonKeyAny() {
	stw.canv.SetCallback(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		stw.DefaultKeyAny(key)
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
		col := regexp.MustCompile("(?i)co(l(u(m(n)?)?)?)?$")
		gir := regexp.MustCompile("(?i)gi(r(d(e(r)?)?)?)?$")
		bra := regexp.MustCompile("(?i)br(a(c(e)?)?)?$")
		wal := regexp.MustCompile("(?i)wa(l){0,2}$")
		sla := regexp.MustCompile("(?i)sl(a(b)?)?$")
		switch {
		case col.MatchString(word):
			val = st.COLUMN
		case gir.MatchString(word):
			val = st.GIRDER
		case bra.MatchString(word):
			val = st.BRACE
		case wal.MatchString(word):
			val = st.WALL
		case sla.MatchString(word):
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
		}
		if plates > 0 {
			stw.Selected[2].SetAttribute("VALUE", fmt.Sprintf("%d", plates))
			stw.Selected[3].SetAttribute("VALUE", fmt.Sprintf("%.3f", area))
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
						// if stw.Changed {
						//     if stw.Yn("CHANGED", "変更を保存しますか") {
						//         stw.SaveAS()
						//     }
						// }
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
	var list *iup.Handle
	var sbt, hbt *iup.Handle
	var data, linedata, platedata *iup.Handle
	var addfig *iup.Handle
	var rcolor, gcolor, bcolor, color *iup.Handle
	sects := make([]*st.Sect, len(stw.Frame.Sects))
	snum := 0
	for _, sec := range stw.Frame.Sects {
		if sec.Num == 900 {
			continue
		}
		sects[snum] = sec
		snum++
	}
	sects = sects[:snum]
	sort.Sort(st.SectByNum{sects})
	sbt = iup.Button("TITLE=\"Show\"", "SIZE=60x")
	hbt = iup.Button("TITLE=\"Hide\"", "SIZE=60x")
	sbt.SetCallback(func(arg *iup.ButtonAction) {
		onoff := list.GetAttribute("VALUE")
		for i := 0; i < len(sects); i++ {
			switch onoff[i] {
			case '+':
				stw.Frame.Show.Sect[sects[i].Num] = true
				list.SetAttribute(fmt.Sprintf("%d", i+1), fmt.Sprintf(" %d", sects[i].Num))
			}
		}
		stw.Redraw()
	})
	hbt.SetCallback(func(arg *iup.ButtonAction) {
		onoff := list.GetAttribute("VALUE")
		for i := 0; i < len(sects); i++ {
			switch onoff[i] {
			case '+':
				stw.Frame.Show.Sect[sects[i].Num] = false
				list.SetAttribute(fmt.Sprintf("%d", i+1), fmt.Sprintf("(%d)", sects[i].Num))
			}
		}
		stw.Redraw()
	})
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
	list = iup.List("SIZE=60x200",
		"MULTIPLE=YES")
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
		col := st.IntColorList(sec.Color)
		rcolor.SetAttribute("VALUE", col[0])
		gcolor.SetAttribute("VALUE", col[1])
		bcolor.SetAttribute("VALUE", col[2])
		dataset["RED"].SetAttribute("VALUE", col[0])
		dataset["GREEN"].SetAttribute("VALUE", col[1])
		dataset["BLUE"].SetAttribute("VALUE", col[2])
		color.SetAttribute("BGCOLOR", strings.Join(col, " "))
	}
	updateproplist := func(sec *st.Sect) {
		proplist.SetAttribute("REMOVEITEM", "ALL")
		for _, f := range sec.Figs {
			proplist.SetAttribute("APPENDITEM", fmt.Sprintf("%d: %d", f.Num, f.Prop.Num))
		}
		proplist.SetAttribute("VALUE", "1")
	}
	list.SetCallback(func(arg *iup.ValueChanged) {
		onoff := list.GetAttribute("VALUE")
		for i, val := range onoff {
			if val == '+' {
				sec := sects[i]
				dataset["CODE"].SetAttribute("VALUE", fmt.Sprintf("%d", sec.Num))
				dataset["NAME"].SetAttribute("VALUE", fmt.Sprintf("%s", sec.Name))
				updateproplist(sec)
				updatedata(sec, 0)
				break
			}
		}
	})
	proplist.SetCallback(func(arg *iup.ValueChanged) {
		var sec *st.Sect
		onoff := list.GetAttribute("VALUE")
		for i, val := range onoff {
			if val == '+' {
				sec = sects[i]
				break
			}
		}
		var ind int
		fmt.Sscanf(proplist.GetAttribute("VALUE"), "%d", &ind)
		ind--
		updatedata(sec, ind)
	})
	addfig = iup.Button("TITLE=\"Add Figure\"", "ACTIVE=NO", "SIZE=\"60x\"")
	addfig.SetCallback(func(arg *iup.ButtonAction) {
		var sec *st.Sect
		onoff := list.GetAttribute("VALUE")
		for i, val := range onoff {
			if val == '+' {
				sec = sects[i]
				break
			}
		}
		f := st.NewFig()
		f.Prop = stw.Frame.DefaultProp()
		f.Num = len(sec.Figs) + 1
		sec.Figs = append(sec.Figs, f)
		updateproplist(sec)
		proplist.SetAttribute("VALUE", fmt.Sprintf("%d", f.Num))
		updatedata(sec, f.Num-1)
	})
	rcolor = iup.Val("MIN=0",
		"MAX=255",
		fmt.Sprintf("STEP=%.2f", 1.0/255),
		"VALUE=255",
		"CANFOCUS=NO")
	dataset["RED"] = datatext("255")
	gcolor = iup.Val("MIN=0",
		"MAX=255",
		fmt.Sprintf("STEP=%.2f", 1.0/255),
		"VALUE=255",
		"CANFOCUS=NO")
	dataset["GREEN"] = datatext("255")
	bcolor = iup.Val("MIN=0",
		"MAX=255",
		fmt.Sprintf("STEP=%.2f", 1.0/255),
		"VALUE=255",
		"CANFOCUS=NO")
	dataset["BLUE"] = datatext("255")
	dataset["RED"].SetAttribute("SIZE", "30x10")
	dataset["GREEN"].SetAttribute("SIZE", "30x10")
	dataset["BLUE"].SetAttribute("SIZE", "30x10")
	color = iup.Canvas("SIZE=\"120x20\"",
		"BGCOLOR=\"255 255 255\"",
		"EXPAND=HORIZONTAL")
	ctlg := iup.Button("TITLE=\"Catalog\"",
		"SIZE=\"100x20\"",
		"EXPAND=HORIZONTAL") // TODO: Section Catalog
	bt := iup.Button("TITLE=\"Set\"",
		"SIZE=\"100x20\"",
		"EXPAND=HORIZONTAL")
	bt.SetCallback(func(arg *iup.ButtonAction) {
		var sec *st.Sect
		onoff := list.GetAttribute("VALUE")
		for i, val := range onoff {
			if val == '+' {
				sec = sects[i]
				break
			}
		}
		var ind int
		fmt.Sscanf(proplist.GetAttribute("VALUE"), "%d", &ind)
		ind--
		if sec.Num > 700 && sec.Num < 900 {
			if len(sec.Figs) > ind {
				var num int64
				var tmp float64
				var err error
				num, err = strconv.ParseInt(dataset["PROP"].GetAttribute("VALUE"), 10, 64)
				if err == nil {
					if p, ok := stw.Frame.Props[int(num)]; ok {
						sec.Figs[ind].Prop = p
					}
				}
				tmp, err = strconv.ParseFloat(dataset["THICK"].GetAttribute("VALUE"), 64)
				if err == nil {
					sec.Figs[ind].Value["THICK"] = tmp
				}
				tmp, err = strconv.ParseFloat(dataset["LLOAD0"].GetAttribute("VALUE"), 64)
				if err == nil {
					sec.Lload[0] = tmp
				}
				tmp, err = strconv.ParseFloat(dataset["LLOAD1"].GetAttribute("VALUE"), 64)
				if err == nil {
					sec.Lload[1] = tmp
				}
				tmp, err = strconv.ParseFloat(dataset["LLOAD2"].GetAttribute("VALUE"), 64)
				if err == nil {
					sec.Lload[2] = tmp
				}
			}
		} else {
			if len(sec.Figs) > ind {
				var num int64
				var tmp float64
				var err error
				num, err = strconv.ParseInt(dataset["PROP"].GetAttribute("VALUE"), 10, 64)
				if err == nil {
					if p, ok := stw.Frame.Props[int(num)]; ok {
						sec.Figs[ind].Prop = p
					}
				}
				tmp, err = strconv.ParseFloat(dataset["AREA"].GetAttribute("VALUE"), 64)
				if err == nil {
					sec.Figs[ind].Value["AREA"] = tmp
				}
				tmp, err = strconv.ParseFloat(dataset["IXX"].GetAttribute("VALUE"), 64)
				if err == nil {
					sec.Figs[ind].Value["IXX"] = tmp
				}
				tmp, err = strconv.ParseFloat(dataset["IYY"].GetAttribute("VALUE"), 64)
				if err == nil {
					sec.Figs[ind].Value["IYY"] = tmp
				}
				tmp, err = strconv.ParseFloat(dataset["VEN"].GetAttribute("VALUE"), 64)
				if err == nil {
					sec.Figs[ind].Value["VEN"] = tmp
				}
			}
		}
		sec.Color = st.ColorInt(fmt.Sprintf("%s %s %s", dataset["RED"].GetAttribute("VALUE"), dataset["GREEN"].GetAttribute("VALUE"), dataset["BLUE"].GetAttribute("VALUE")))
		updatedata(sec, ind)
		stw.Redraw()
	})
	changecolor := func() {
		color.SetAttribute("BGCOLOR", strings.Join([]string{strings.Split(rcolor.GetAttribute("VALUE"), ".")[0],
			strings.Split(gcolor.GetAttribute("VALUE"), ".")[0],
			strings.Split(bcolor.GetAttribute("VALUE"), ".")[0]}, " "))
	}
	rcolor.SetCallback(func(arg *iup.ValueChanged) {
		dataset["RED"].SetAttribute("VALUE", strings.Split(rcolor.GetAttribute("VALUE"), ".")[0])
		changecolor()
	})
	gcolor.SetCallback(func(arg *iup.ValueChanged) {
		dataset["GREEN"].SetAttribute("VALUE", strings.Split(gcolor.GetAttribute("VALUE"), ".")[0])
		changecolor()
	})
	bcolor.SetCallback(func(arg *iup.ValueChanged) {
		dataset["BLUE"].SetAttribute("VALUE", strings.Split(bcolor.GetAttribute("VALUE"), ".")[0])
		changecolor()
	})
	dataset["RED"].SetCallback(func(arg *iup.CommonKillFocus) {
		val, err := strconv.ParseInt(dataset["RED"].GetAttribute("VALUE"), 10, 64)
		if err != nil {
			dataset["RED"].SetAttribute("VALUE", strings.Split(rcolor.GetAttribute("VALUE"), ".")[0])
		}
		rcolor.SetAttribute("VALUE", fmt.Sprintf("%d", int(val)))
		changecolor()
	})
	dataset["RED"].SetCallback(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		switch key.Key() {
		case KEY_ENTER:
			tmp, err := strconv.ParseInt(dataset["RED"].GetAttribute("VALUE"), 10, 64)
			if err != nil {
				dataset["RED"].SetAttribute("VALUE", strings.Split(rcolor.GetAttribute("VALUE"), ".")[0])
			}
			rcolor.SetAttribute("VALUE", fmt.Sprintf("%d", int(tmp)))
			changecolor()
		case KEY_ESCAPE:
			dataset["RED"].SetAttribute("VALUE", strings.Split(rcolor.GetAttribute("VALUE"), ".")[0])
		}
	})
	dataset["GREEN"].SetCallback(func(arg *iup.CommonKillFocus) {
		val, err := strconv.ParseInt(dataset["GREEN"].GetAttribute("VALUE"), 10, 64)
		if err != nil {
			dataset["GREEN"].SetAttribute("VALUE", strings.Split(gcolor.GetAttribute("VALUE"), ".")[0])
		}
		gcolor.SetAttribute("VALUE", fmt.Sprintf("%d", int(val)))
		changecolor()
	})
	dataset["GREEN"].SetCallback(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		switch key.Key() {
		case KEY_ENTER:
			tmp, err := strconv.ParseInt(dataset["GREEN"].GetAttribute("VALUE"), 10, 64)
			if err != nil {
				dataset["GREEN"].SetAttribute("VALUE", strings.Split(gcolor.GetAttribute("VALUE"), ".")[0])
			}
			gcolor.SetAttribute("VALUE", fmt.Sprintf("%d", int(tmp)))
			changecolor()
		case KEY_ESCAPE:
			dataset["GREEN"].SetAttribute("VALUE", strings.Split(gcolor.GetAttribute("VALUE"), ".")[0])
		}
	})
	dataset["BLUE"].SetCallback(func(arg *iup.CommonKillFocus) {
		val, err := strconv.ParseInt(dataset["BLUE"].GetAttribute("VALUE"), 10, 64)
		if err != nil {
			dataset["BLUE"].SetAttribute("VALUE", strings.Split(bcolor.GetAttribute("VALUE"), ".")[0])
		}
		bcolor.SetAttribute("VALUE", fmt.Sprintf("%d", int(val)))
		changecolor()
	})
	dataset["BLUE"].SetCallback(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		switch key.Key() {
		case KEY_ENTER:
			tmp, err := strconv.ParseInt(dataset["BLUE"].GetAttribute("VALUE"), 10, 64)
			if err != nil {
				dataset["BLUE"].SetAttribute("VALUE", strings.Split(bcolor.GetAttribute("VALUE"), ".")[0])
			}
			bcolor.SetAttribute("VALUE", fmt.Sprintf("%d", int(tmp)))
			changecolor()
		case KEY_ESCAPE:
			dataset["BLUE"].SetAttribute("VALUE", strings.Split(bcolor.GetAttribute("VALUE"), ".")[0])
		}
	})
	dlg := iup.Dialog(iup.Hbox(iup.Vbox(list, sbt, hbt),
		iup.Vbox(dataset["NAME"],
			iup.Hbox(propertylabel("CODE"), dataset["CODE"]),
			iup.Hbox(proplist, addfig),
			iup.Hbox(propertylabel("PROP"), dataset["PROP"]),
			data,
			color,
			iup.Vbox(iup.Hbox(rcolor, dataset["RED"]),
				iup.Hbox(gcolor, dataset["GREEN"]),
				iup.Hbox(bcolor, dataset["BLUE"])),
			ctlg, bt)))
	dlg.SetAttribute("TITLE", "Section")
	dlg.SetAttribute("DIALOGFRAME", "YES")
	dlg.SetAttribute("PARENTDIALOG", "mainwindow")
	dlg.Map()
	for _, sec := range sects {
		if stw.Frame.Show.Sect[sec.Num] {
			list.SetAttribute("APPENDITEM", fmt.Sprintf(" %d", sec.Num))
		} else {
			list.SetAttribute("APPENDITEM", fmt.Sprintf("(%d)", sec.Num))
		}
	}
	dlg.Show()
}

// func ColorSet (cvs *iup.Handle) (val, text []*iup.Handle) {
//     val  = make([]*iup.Handle, 3)
//     text = make([]*iup.Handle, 3)
//     changecolor := func () {
//                        cvs.SetAttribute("BGCOLOR", strings.Join([]string{strings.Split(val[0].GetAttribute("VALUE"), ".")[0],
//                                                                          strings.Split(val[1].GetAttribute("VALUE"), ".")[0],
//                                                                          strings.Split(val[2].GetAttribute("VALUE"), ".")[0]}, " "))
//                                                                       }
//     for i:=0; i<3; i++ {
//         val[i] = iup.Val("MIN=0",
//                          "MAX=255",
//                          fmt.Sprintf("STEP=%.2f", 1.0/255),
//                          "VALUE=255",
//                          "CANFOCUS=NO",)
//         val[i].SetCallback(func (arg *iup.ValueChanged) {
//                                text[i].SetAttribute("VALUE", strings.Split(val[i].GetAttribute("VALUE"), ".")[0])
//                                changecolor()
//                            })
//         text[i] = datatext("255")
//         text[i].SetCallback(func (arg *iup.CommonKillFocus) {
//                                 tmp, err := strconv.ParseInt(text[i].GetAttribute("VALUE"), 10, 64)
//                                 if err != nil {
//                                     text[i].SetAttribute("VALUE", strings.Split(val[i].GetAttribute("VALUE"), ".")[0])
//                                 }
//                                 val[i].SetAttribute("VALUE", fmt.Sprintf("%d", int(tmp)))
//                                 changecolor()
//                             })
//         text[i].SetCallback(func (arg *iup.CommonKeyAny) {
//                                 key := iup.KeyState(arg.Key)
//                                 switch key.Key() {
//                                 case KEY_ENTER:
//                                     tmp, err := strconv.ParseInt(text[i].GetAttribute("VALUE"), 10, 64)
//                                     if err != nil {
//                                         text[i].SetAttribute("VALUE", strings.Split(val[i].GetAttribute("VALUE"), ".")[0])
//                                     }
//                                     val[i].SetAttribute("VALUE", fmt.Sprintf("%d", int(tmp)))
//                                     changecolor()
//                                 case KEY_ESCAPE:
//                                     text[i].SetAttribute("VALUE", strings.Split(val[i].GetAttribute("VALUE"), ".")[0])
//                                 }
//                             })
//     }
//     return
// }

func (stw *Window) SectionDialog2() {
	if stw.Frame == nil {
		return
	}
	sects := make([]*st.Sect, len(stw.Frame.Sects))
	snum := 0
	for _, sec := range stw.Frame.Sects {
		if sec.Num >= 900 {
			continue
		}
		sects[snum] = sec
		snum++
	}
	sects = sects[:snum]
	sort.Sort(st.SectByNum{sects})
	selstart := -1; selend := -1
	codes := make([]*iup.Handle, snum)
	snames := make([]*iup.Handle, snum)
	hides := make([]*iup.Handle, snum)
	colors := make([]*iup.Handle, snum)
	title := iup.Hbox(
		iup.Label(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, "9"),
			fmt.Sprintf("FGCOLOR=\"%s\"", labelFGColor),
			fmt.Sprintf("BGCOLOR=\"%s\"", labelBGColor),
			fmt.Sprintf("SIZE=%dx%d", 20, dataheight),
			"TITLE=\"CODE\""),
		iup.Label(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, "9"),
			fmt.Sprintf("FGCOLOR=\"%s\"", labelFGColor),
			fmt.Sprintf("BGCOLOR=\"%s\"", labelBGColor),
			fmt.Sprintf("SIZE=%dx%d", 200, dataheight),
			"TITLE=\"NAME\""),
		iup.Label(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, "9"),
			fmt.Sprintf("FGCOLOR=\"%s\"", labelFGColor),
			fmt.Sprintf("BGCOLOR=\"%s\"", labelBGColor),
			fmt.Sprintf("SIZE=%dx%d", 20, dataheight),
			"TITLE=\"表示\""))
	sections := iup.Vbox()
	for i, sec := range sects {
		codes[i] = iup.Label(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, "9"),
			fmt.Sprintf("FGCOLOR=\"%s\"", labelFGColor),
			fmt.Sprintf("BGCOLOR=\"%s\"", labelBGColor),
			fmt.Sprintf("TITLE=\"%d\"", sec.Num),
			fmt.Sprintf("SIZE=%dx%d", 20, dataheight),
		)
		snames[i] = iup.Label(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, "9"),
			fmt.Sprintf("FGCOLOR=\"%s\"", labelFGColor),
			fmt.Sprintf("BGCOLOR=\"%s\"", labelBGColor),
			fmt.Sprintf("TITLE=\"%s\"", sec.Name),
			fmt.Sprintf("SIZE=%dx%d", 200, dataheight),
		)
		func (num int) {
			codes[num].SetCallback(func(arg *iup.MouseButton) {
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
				for j:=0; j<snum; j++ {
					codes[j].SetAttribute("FGCOLOR", labelFGColor)
					snames[j].SetAttribute("FGCOLOR", labelFGColor)
				}
				for j:=selstart; j<selend+1; j++ {
					if j>snum { break }
					codes[j].SetAttribute("FGCOLOR", "42  54 177")
					snames[j].SetAttribute("FGCOLOR", "42  54 177")
				}
			})
			snames[num].SetCallback(func(arg *iup.MouseButton) {
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
				for j:=0; j<snum; j++ {
					codes[j].SetAttribute("FGCOLOR", labelFGColor)
					snames[j].SetAttribute("FGCOLOR", labelFGColor)
				}
				for j:=selstart; j<selend+1; j++ {
					if j>snum { break }
					codes[j].SetAttribute("FGCOLOR", "42  54 177")
					snames[j].SetAttribute("FGCOLOR", "42  54 177")
				}
			})
		}(i)
		var ishide string
		if stw.Frame.Show.Sect[sec.Num] {
			ishide = "ON"
		} else {
			ishide = "OFF"
		}
		hides[i] = iup.Toggle(fmt.Sprintf("FGCOLOR=\"%s\"", labelFGColor),
			fmt.Sprintf("BGCOLOR=\"%s\"", labelBGColor),
			fmt.Sprintf("VALUE=%s", ishide),
			"CANFOCUS=NO",
			fmt.Sprintf("SIZE=%dx%d", 20, dataheight))
		func(snum, num int) {
			hides[num].SetCallback(func(arg *iup.ToggleAction) {
				if stw.Frame != nil {
					if arg.State == 1 {
						if selstart <= num && num <= selend {
							for j:=selstart; j<selend+1; j++ {
								stw.Frame.Show.Sect[sects[j].Num] = true
								hides[j].SetAttribute("VALUE", "ON")
							}
						} else {
							stw.Frame.Show.Sect[snum] = true
						}
					} else {
						if selstart <= num && num <= selend {
							for j:=selstart; j<selend+1; j++ {
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
					fmt.Printf("Clicked: %d\n", num)
					col, err := stw.ColorDialog()
					if err != nil { return }
					fmt.Println(col)
				}
			})
		}(sec.Num, i)
		sbox := iup.Hbox(codes[i], snames[i], hides[i], colors[i])
		sections.Append(sbox)
	}
	dlg := iup.Dialog(iup.Vbox(title, iup.Label("SEPARATOR=HORIZONTAL"), iup.ScrollBox(sections, "SIZE=\"350x150\"")))
	dlg.SetAttribute("TITLE", "Section")
	dlg.SetAttribute("DIALOGFRAME", "YES")
	dlg.SetAttribute("PARENTDIALOG", "mainwindow")
	iup.SetHandle("sectiondialog", dlg)
	dlg.Map()
	dlg.Show()
}

func (stw *Window) ColorDialog () (int, error) {
	if colordlg == nil {
		reds := iup.Hbox()
		pinks := iup.Hbox()
		purples := iup.Hbox()
		deeppurples := iup.Hbox()
		indigos := iup.Hbox()
		blues := iup.Hbox()
		lightblues := iup.Hbox()
		cyans := iup.Hbox()
		teals := iup.Hbox()
		greens := iup.Hbox()
		lightgreens := iup.Hbox()
		limes := iup.Hbox()
		yellows := iup.Hbox()
		ambers := iup.Hbox()
		oranges := iup.Hbox()
		deeporanges := iup.Hbox()
		browns := iup.Hbox()
		greys := iup.Hbox()
		bluegreys := iup.Hbox()
		colnames := []string{"100", "200", "300", "400", "500", "600", "700", "800", "900", "A100", "A200", "A400", "A700"}
		for _, n := range colnames {
			reds.Append(iup.Canvas(fmt.Sprintf("SIZE=\"%dx%d\"", 20, dataheight),
				fmt.Sprintf("BGCOLOR=\"%s\"", st.IntColor(st.GOOGLE_RED[n])),
				"EXPAND=NO"))
			pinks.Append(iup.Canvas(fmt.Sprintf("SIZE=\"%dx%d\"", 20, dataheight),
				fmt.Sprintf("BGCOLOR=\"%s\"", st.IntColor(st.GOOGLE_PINK[n])),
				"EXPAND=NO"))
			purples.Append(iup.Canvas(fmt.Sprintf("SIZE=\"%dx%d\"", 20, dataheight),
				fmt.Sprintf("BGCOLOR=\"%s\"", st.IntColor(st.GOOGLE_PURPLE[n])),
				"EXPAND=NO"))
			deeppurples.Append(iup.Canvas(fmt.Sprintf("SIZE=\"%dx%d\"", 20, dataheight),
				fmt.Sprintf("BGCOLOR=\"%s\"", st.IntColor(st.GOOGLE_DEEPPURPLE[n])),
				"EXPAND=NO"))
			indigos.Append(iup.Canvas(fmt.Sprintf("SIZE=\"%dx%d\"", 20, dataheight),
				fmt.Sprintf("BGCOLOR=\"%s\"", st.IntColor(st.GOOGLE_INDIGO[n])),
				"EXPAND=NO"))
			blues.Append(iup.Canvas(fmt.Sprintf("SIZE=\"%dx%d\"", 20, dataheight),
				fmt.Sprintf("BGCOLOR=\"%s\"", st.IntColor(st.GOOGLE_BLUE[n])),
				"EXPAND=NO"))
			lightblues.Append(iup.Canvas(fmt.Sprintf("SIZE=\"%dx%d\"", 20, dataheight),
				fmt.Sprintf("BGCOLOR=\"%s\"", st.IntColor(st.GOOGLE_LIGHTBLUE[n])),
				"EXPAND=NO"))
			cyans.Append(iup.Canvas(fmt.Sprintf("SIZE=\"%dx%d\"", 20, dataheight),
				fmt.Sprintf("BGCOLOR=\"%s\"", st.IntColor(st.GOOGLE_CYAN[n])),
				"EXPAND=NO"))
			teals.Append(iup.Canvas(fmt.Sprintf("SIZE=\"%dx%d\"", 20, dataheight),
				fmt.Sprintf("BGCOLOR=\"%s\"", st.IntColor(st.GOOGLE_TEAL[n])),
				"EXPAND=NO"))
			greens.Append(iup.Canvas(fmt.Sprintf("SIZE=\"%dx%d\"", 20, dataheight),
				fmt.Sprintf("BGCOLOR=\"%s\"", st.IntColor(st.GOOGLE_GREEN[n])),
				"EXPAND=NO"))
			lightgreens.Append(iup.Canvas(fmt.Sprintf("SIZE=\"%dx%d\"", 20, dataheight),
				fmt.Sprintf("BGCOLOR=\"%s\"", st.IntColor(st.GOOGLE_LIGHTGREEN[n])),
				"EXPAND=NO"))
			limes.Append(iup.Canvas(fmt.Sprintf("SIZE=\"%dx%d\"", 20, dataheight),
				fmt.Sprintf("BGCOLOR=\"%s\"", st.IntColor(st.GOOGLE_LIME[n])),
				"EXPAND=NO"))
			yellows.Append(iup.Canvas(fmt.Sprintf("SIZE=\"%dx%d\"", 20, dataheight),
				fmt.Sprintf("BGCOLOR=\"%s\"", st.IntColor(st.GOOGLE_YELLOW[n])),
				"EXPAND=NO"))
			ambers.Append(iup.Canvas(fmt.Sprintf("SIZE=\"%dx%d\"", 20, dataheight),
				fmt.Sprintf("BGCOLOR=\"%s\"", st.IntColor(st.GOOGLE_AMBER[n])),
				"EXPAND=NO"))
			oranges.Append(iup.Canvas(fmt.Sprintf("SIZE=\"%dx%d\"", 20, dataheight),
				fmt.Sprintf("BGCOLOR=\"%s\"", st.IntColor(st.GOOGLE_ORANGE[n])),
				"EXPAND=NO"))
			deeporanges.Append(iup.Canvas(fmt.Sprintf("SIZE=\"%dx%d\"", 20, dataheight),
				fmt.Sprintf("BGCOLOR=\"%s\"", st.IntColor(st.GOOGLE_DEEPORANGE[n])),
				"EXPAND=NO"))
		}
		for _, n := range []string{"100", "200", "300", "400", "500", "600", "700", "800", "900"} {
			browns.Append(iup.Canvas(fmt.Sprintf("SIZE=\"%dx%d\"", 20, dataheight),
				fmt.Sprintf("BGCOLOR=\"%s\"", st.IntColor(st.GOOGLE_BROWN[n])),
				"EXPAND=NO"))
			bluegreys.Append(iup.Canvas(fmt.Sprintf("SIZE=\"%dx%d\"", 20, dataheight),
				fmt.Sprintf("BGCOLOR=\"%s\"", st.IntColor(st.GOOGLE_BLUEGREY[n])),
				"EXPAND=NO"))
		}
		for _, n := range []string{"0", "100", "200", "300", "400", "500", "600", "700", "800", "900", "1000"} {
			greys.Append(iup.Canvas(fmt.Sprintf("SIZE=\"%dx%d\"", 20, dataheight),
				fmt.Sprintf("BGCOLOR=\"%s\"", st.IntColor(st.GOOGLE_GREY[n])),
				"EXPAND=NO"))
		}
		colordlg = iup.Dialog(iup.Vbox(reds,
			pinks,
			purples,
			deeppurples,
			indigos,
			blues,
			lightblues,
			cyans,
			teals,
			greens,
			lightgreens,
			limes,
			yellows,
			ambers,
			oranges,
			deeporanges,
			browns,
			bluegreys,
			greys))
		colordlg.SetAttribute("TITLE", "Color")
		colordlg.SetAttribute("DIALOGFRAME", "YES")
		colordlg.SetAttribute("PARENTDIALOG", "sectiondialog")
		colordlg.Map()
	}
	colordlg.Show()
	return 0, nil
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
	stw.Labels[st.ETYPES[etype]].SetAttribute("FGCOLOR", labelOFFColor)
}

func (stw *Window) ShowEtype(etype int) {
	if etype == 0 {
		return
	}
	stw.Frame.Show.Etype[etype] = true
	stw.UpdateShowRange() // TODO: ShowRange
	stw.Labels[st.ETYPES[etype]].SetAttribute("FGCOLOR", labelFGColor)
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
									el.Hide = true
								}
								if el.Etype == etype-2 {
									el.Hide = false
								}
							}
						} else {
							stw.Frame.Show.Etype[etype-2] = false
							stw.Labels[st.ETYPES[etype-2]].SetAttribute("FGCOLOR", labelOFFColor)
							for _, el := range stw.Frame.Elems {
								if el.Etype == etype-2 {
									el.Hide = true
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
									el.Hide = false
								}
								if el.Etype == etype-2 {
									el.Hide = true
								}
							}
						} else {
							stw.Frame.Show.Etype[etype] = true
							stw.Labels[st.ETYPES[etype]].SetAttribute("FGCOLOR", labelFGColor)
							for _, el := range stw.Frame.Elems {
								if el.Etype == etype {
									el.Hide = false
								}
							}
						}
					}
					stw.UpdateShowRange()
					stw.HideNodes()
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
							el.Hide = true
						}
						if el.Etype == etype-2 {
							el.Hide = true
						}
					}
					stw.UpdateShowRange()
					stw.HideNodes()
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
	stw.Labels[fmt.Sprintf("%s_%s", st.ETYPES[etype], strings.ToUpper(st.StressName[index]))].SetAttribute("FGCOLOR", labelFGColor)
}

func (stw *Window) StressOff(etype int, index uint) {
	stw.Frame.Show.Stress[etype] &= ^(1 << index)
	stw.Labels[fmt.Sprintf("%s_%s", st.ETYPES[etype], strings.ToUpper(st.StressName[index]))].SetAttribute("FGCOLOR", labelOFFColor)
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
			stw.UpdateShowRange()
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
				stw.UpdateShowRange()
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

func (stw *Window) UpdateShowRange() {
	for _, n := range stw.Frame.Nodes {
		// if n.Hide { continue }
		n.Hide = false
		if n.Coord[0] < stw.Frame.Show.Xrange[0] || stw.Frame.Show.Xrange[1] < n.Coord[0] {
			n.Hide = true
		} else if n.Coord[1] < stw.Frame.Show.Yrange[0] || stw.Frame.Show.Yrange[1] < n.Coord[1] {
			n.Hide = true
		} else if n.Coord[2] < stw.Frame.Show.Zrange[0] || stw.Frame.Show.Zrange[1] < n.Coord[2] {
			n.Hide = true
		}
	}
	for _, el := range stw.Frame.Elems {
		// if el.Hide { continue }
		el.Hide = false
		for _, en := range el.Enod {
			if en.Hide {
				el.Hide = true
				break
			}
		}
	}
	for _, k := range stw.Frame.Kijuns {
		// if k.Hide { continue }
		k.Hide = false
		d := k.Direction()
		if math.Abs(d[0]) < 1e-4 {
			if k.Start[0] < stw.Frame.Show.Xrange[0] || stw.Frame.Show.Xrange[1] < k.Start[0] {
				k.Hide = true
			}
			if k.End[0] < stw.Frame.Show.Xrange[0] || stw.Frame.Show.Xrange[1] < k.End[0] {
				k.Hide = true
			}
		}
		if math.Abs(d[1]) < 1e-4 {
			if k.Start[1] < stw.Frame.Show.Yrange[0] || stw.Frame.Show.Yrange[1] < k.Start[1] {
				k.Hide = true
			}
			if k.End[1] < stw.Frame.Show.Yrange[0] || stw.Frame.Show.Yrange[1] < k.End[1] {
				k.Hide = true
			}
		}
		if math.Abs(d[2]) < 1e-4 {
			if k.Start[2] < stw.Frame.Show.Zrange[0] || stw.Frame.Show.Zrange[1] < k.Start[2] {
				k.Hide = true
			}
			if k.End[2] < stw.Frame.Show.Zrange[0] || stw.Frame.Show.Zrange[1] < k.End[2] {
				k.Hide = true
			}
		}
	}
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
		if len(words) == 0 {
			continue
		}
		if value, ok := Commands[strings.ToUpper(words[1])]; ok {
			aliases[strings.ToUpper(words[0])] = value
		}
	}
	if err := s.Err(); err != nil {
		return err
	}
	return nil
}

// Initialize
func init() {
	aliases = make(map[string]*Command, 0)
	sectionaliases = make(map[int]string, 0)
	err := ReadPgp(pgpfile, aliases)
	if err != nil {
		aliases["D"] = DISTS
		aliases["TB"] = TOGGLEBOND
		aliases["O"] = OPEN
		aliases["IN"] = INSERT
		aliases["SF"] = SETFOCUS
		aliases["NODE"] = SELECTNODE
		aliases["ELEM"] = SELECTELEM
		aliases["SEC"] = SELECTSECT
		aliases["CH"] = SELECTCHILDREN
		aliases["ER"] = ERRORELEM
		aliases["F"] = FENCE
		aliases["A"] = ADDLINEELEM
		aliases["AA"] = ADDPLATEELEM
		aliases["AAA"] = ADDPLATEELEMBYLINE
		aliases["H"] = HATCHPLATEELEM
		aliases["ED"] = EDITPLATEELEM
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
		aliases["DD"] = MOVENODE
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
		aliases["ND"] = NODEDUPLICATION
		aliases["CN"] = CATBYNODE
		aliases["J"] = JOINLINEELEM
		aliases["JP"] = JOINPLATEELEM
		aliases["EA"] = EXTRACTARCLM
		aliases["DO"] = DIVIDEATONS
		aliases["DM"] = DIVIDEATMID
		aliases["I"] = INTERSECT
		aliases["IA"] = INTERSECTALL
		aliases["Z"] = TRIM
		aliases["ZZ"] = EXTEND
		aliases["M"] = MERGENODE
		aliases["E"] = ERASE
		aliases["FAC"] = FACTS
		aliases["RA"] = REACTION
		aliases["SS"] = NOTICE1459
		aliases["ZD"] = ZOUBUNDISP
		aliases["ZY"] = ZOUBUNYIELD
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
