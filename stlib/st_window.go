package st

import (
    "errors"
    "fmt"
    "os"
    "os/exec"
    "strings"
    "path/filepath"
    "strconv"
    "runtime"
    "github.com/visualfc/go-iup/iup"
    "github.com/visualfc/go-iup/cd"
    "io/ioutil"
    "math"
    "regexp"
    "sort"
    "time"
)


// Constants & Variables// {{{
// General
var (
    aliases  map[string]*Command
    DoubleClickCommand = []string{"TOGGLEBOND", "EDITPLATEELEM"}
)
var (
    gopath = os.Getenv("GOPATH")
    releasenote = filepath.Join(gopath, "src/github.com/yofu/st/help/releasenote.html")
    pgpfile = filepath.Join(gopath, "src/github.com/yofu/st/st.pgp")
)
const (
    windowSize = "FULLxFULL"
)

// Font
var (
    commandFontFace = "IPA明朝"
    commandFontSize = "11"
    labelFGColor = "0 0 0"
    labelBGColor = "255 255 255"
    commandFGColor = "0 127 31"
    commandBGColor = "191 191 191"
    historyBGColor = "220 220 220"
    sectionLabelColor = "204 51 0"
    labelOFFColor = "160 160 160"
    canvasFontFace = "IPA明朝"
    canvasFontSize = 9
    canvasFontColor = cd.CD_DARK_GRAY
    DefaultTextAlignment = cd.CD_BASE_LEFT
)
const (
    FONT_COMMAND = iota
    FONT_CANVAS
)

// Draw
var (
    first = 1

)

const ( // NodeCaption
    NC_NUM = 1 << iota
    NC_ZCOORD
    NC_DX
    NC_DY
    NC_DZ
    NC_RX
    NC_RY
    NC_RZ
)
const ( // ElemCaption
    EC_NUM = 1 << iota
    EC_SECT
    EC_RATE_L
    EC_RATE_S
)

var (
    LOCKED_ELEM_COLOR = cd.CD_DARK_GRAY
    LOCKED_NODE_COLOR = cd.CD_DARK_YELLOW
)

// DataLabel
const (
    datalabelwidth = 35
    datatextwidth  = 30
    dataheight  = 10
)

// Select
var selectDirection = 0
const dotSelectPixel = 5
const (
    SD_FROMLEFT = iota
    SD_FROMRIGHT
)

// Key & Button
const (
    KEY_BS     = 8
    KEY_TAB    = 9
    KEY_ENTER  = 13
    KEY_DELETE = 139
    KEY_ESCAPE = 141

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
    CanvasMoveSpeedX = 0.05
    CanvasMoveSpeedY = 0.05
    CanvasScaleSpeed = 15
)

// Command
const (
    repeatcommand = 0.2 // sec
)
var (
    pressed time.Time
)

// Line Color
var ( // Boundary for Rainbow (length should be <= 6)
    RateBoundary = []float64{0.5, 0.6, 0.7, 0.71428, 0.9, 1.0}
    HeightBoundary = []float64{0.5, 1.0, 1.5, 2.0, 2.5, 3.0}
)
var (
    ECOLORS = []string{ "WHITE", "BLACK", "BY SECTION", "BY RATE", "BY HEIGHT" }
    PERIODS = []string{ "L", "X", "Y" }
)
const (
    ECOLOR_WHITE = iota
    ECOLOR_BLACK
    ECOLOR_SECT
    ECOLOR_RATE
    ECOLOR_HEIGHT
)

var (
    axrn_minmax = regexp.MustCompile("([+-]?[-0-9.]+)<=([XYZxyz]{1})<=([+-]?[-0-9.]+)")
    axrn_min = regexp.MustCompile("([+-]?[-0-9.]+)<=([XYZxyz]{1})")
    axrn_max = regexp.MustCompile("([XYZxyz]{1})<=([+-]?[-0-9.]+)")
    axrn_eq = regexp.MustCompile("([XYZxyz]{1})=([+-]?[-0-9.]+)")
)
// }}}


type Window struct {// {{{
    Home   string

    Show *Show

    Frame  *Frame
    Dlg    *iup.Handle
    canv   *iup.Handle
    cdcanv *cd.Canvas
    dbuff  *cd.Canvas
    cline  *iup.Handle
    hist   *iup.Handle
    formattag, lsformattag *iup.Handle
    cname  *iup.Handle
    context *iup.Handle

    CanvasSize []float64 // width, height

    zb *iup.Handle

    lselect *iup.Handle

    SelectNode []*Node
    SelectElem []*Elem

    Version string
    Modified string

    startT time.Time
    startX int
    startY int
    endX   int
    endY   int

    lastcommand *Command
}
// }}}


func NewWindow(homedir string) *Window {// {{{
    stw := new(Window)
    stw.Home = homedir
    stw.Show = NewShow(stw)
    stw.SelectNode = make([]*Node, 0)
    stw.SelectElem = make([]*Elem, 0)
    iup.Menu(
        iup.Attrs("BGCOLOR", labelBGColor,),
        iup.SubMenu("TITLE=File",
            iup.Menu(
                iup.Item(
                    iup.Attr("TITLE","Open\tCtrl+O"),
                    iup.Attr("TIP","Open file"),
                    func (arg *iup.ItemAction) {
                        runtime.GC()
                        stw.Open()
                    },
                ),
                iup.Item(
                    iup.Attr("TITLE","Insert\tCtrl+I"),
                    iup.Attr("TIP","Insert frame"),
                    func (arg *iup.ItemAction) {
                        stw.Insert()
                    },
                ),
                iup.Item(
                    iup.Attr("TITLE","Reload\tCtrl+M"),
                    func (arg *iup.ItemAction) {
                        runtime.GC()
                        stw.Reload()
                    },
                ),
                iup.Item(
                    iup.Attr("TITLE","Open Dxf"),
                    iup.Attr("TIP","Open dxf file"),
                    func (arg *iup.ItemAction) {
                        runtime.GC()
                        stw.OpenDxf()
                    },
                ),
                iup.Item(
                    iup.Attr("TITLE","Save\tCtrl+S"),
                    iup.Attr("TIP","Save file"),
                    func (arg *iup.ItemAction) {
                        stw.Save()
                    },
                ),
                iup.Item(
                    iup.Attr("TITLE","Save As\tCtrl+A"),
                    iup.Attr("TIP","Save file As"),
                    func (arg *iup.ItemAction) {
                        stw.SaveAS()
                    },
                ),
                iup.Item(
                    iup.Attr("TITLE","Read"),
                    iup.Attr("TIP","Read file"),
                    func (arg *iup.ItemAction) {
                        stw.Read()
                    },
                ),
                iup.Item(
                    iup.Attr("TITLE","Read All\tCtrl+R"),
                    iup.Attr("TIP","Read all file"),
                    func (arg *iup.ItemAction) {
                        stw.ReadAll()
                    },
                ),
                iup.Item(
                    iup.Attr("TITLE","Print"),
                    iup.Attr("TIP","Print"),
                    func (arg *iup.ItemAction) {
                        stw.Print()
                    },
                ),
                // iup.Item(
                //     iup.Attr("TITLE","Print PDF"),
                //     func (arg *iup.ItemAction) {
                //         stw.PrintPDF()
                //     },
                // ),
                iup.Separator(),
                iup.Item(
                    iup.Attr("TITLE","Quit"),
                    iup.Attr("TIP","Exit Application"),
                    func (arg *iup.ItemAction) {
                        arg.Return = iup.CLOSE
                    },
                ),
            ),
        ),
        iup.SubMenu("TITLE=Edit",
            iup.Menu(
                iup.Item(
                    iup.Attr("TITLE","Command Font"),
                    func (arg *iup.ItemAction) {
                        stw.SetFont(FONT_COMMAND)
                    },
                ),
                iup.Item(
                    iup.Attr("TITLE","Canvas Font"),
                    func (arg *iup.ItemAction) {
                        stw.SetFont(FONT_CANVAS)
                    },
                ),
                iup.Item(
                    iup.Attr("TITLE","Property\tCtrl+Q"),
                    func (arg *iup.ItemAction) {
                        if !stw.Show.Property {
                            stw.PropertyDialog()
                        }
                    },
                ),
                iup.Item(
                    iup.Attr("TITLE","Section\tCtrl+Q"),
                    func (arg *iup.ItemAction) {
                        stw.SectionDialog()
                    },
                ),
                iup.Item(
                    iup.Attr("TITLE","Edit Inp\tCtrl+E"),
                    func (arg *iup.ItemAction) {
                        stw.EditInp()
                    },
                ),
                iup.Item(
                    iup.Attr("TITLE","Edit Pgp"),
                    func (arg *iup.ItemAction) {
                        EditPgp()
                    },
                ),
                iup.Item(
                    iup.Attr("TITLE","Read Pgp"),
                    func (arg *iup.ItemAction) {
                        al := make(map[string]*Command,0)
                        err := ReadPgp(pgpfile, al)
                        if err != nil {
                            stw.addHistory("ReadPgp: Cannot Read st.pgp")
                        } else {
                            aliases = al
                        }
                    },
                ),
            ),
        ),
        iup.SubMenu("TITLE=View",
            iup.Menu(
                iup.Item(
                    iup.Attr("TITLE","Top"),
                    func (arg *iup.ItemAction) {
                        stw.SetAngle(90.0, -90.0)
                    },
                ),
                iup.Item(
                    iup.Attr("TITLE","Front"),
                    func (arg *iup.ItemAction) {
                        stw.SetAngle(0.0, -90.0)
                    },
                ),
                iup.Item(
                    iup.Attr("TITLE","Back"),
                    func (arg *iup.ItemAction) {
                        stw.SetAngle(0.0, 90.0)
                    },
                ),
                iup.Item(
                    iup.Attr("TITLE","Right"),
                    func (arg *iup.ItemAction) {
                        stw.SetAngle(0.0, 0.0)
                    },
                ),
                iup.Item(
                    iup.Attr("TITLE","Left"),
                    func (arg *iup.ItemAction) {
                        stw.SetAngle(0.0, 180.0)
                    },
                ),
            ),
        ),
        iup.SubMenu("TITLE=Show",
            iup.Menu(
                iup.Item(
                    iup.Attr("TITLE","Show All\tCtrl+S"),
                    func (arg *iup.ItemAction) {
                        stw.ShowAll()
                    },
                ),
                iup.Item(
                    iup.Attr("TITLE","Hide Selected Elems\tCtrl+H"),
                    func (arg *iup.ItemAction) {
                        stw.HideSelected()
                    },
                ),
                iup.Item(
                    iup.Attr("TITLE","Show Selected Elems\tCtrl+D"),
                    func (arg *iup.ItemAction) {
                        stw.HideNotSelected()
                    },
                ),
            ),
        ),
        iup.SubMenu("TITLE=Command",
            iup.Menu( iup.Item( iup.Attr("TITLE","DISTS"), func (arg *iup.ItemAction) { stw.execAliasCommand("DISTS") },),
                      iup.Item( iup.Attr("TITLE","SET FOCUS"), func (arg *iup.ItemAction) { stw.execAliasCommand("SETFOCUS") },),
                      iup.Separator(),
                      iup.Item( iup.Attr("TITLE","TOGGLEBOND"), func (arg *iup.ItemAction) { stw.execAliasCommand("TOGGLEBOND") },),
                      iup.Item( iup.Attr("TITLE","COPYBOND"), func (arg *iup.ItemAction) { stw.execAliasCommand("COPYBOND") },),
                      iup.Separator(),
                      iup.Item( iup.Attr("TITLE","SELECT NODE"), func (arg *iup.ItemAction) { stw.execAliasCommand("SELECTNODE") },),
                      iup.Item( iup.Attr("TITLE","SELECT ELEM"), func (arg *iup.ItemAction) { stw.execAliasCommand("SELECTELEM") },),
                      iup.Item( iup.Attr("TITLE","SELECT SECT"), func (arg *iup.ItemAction) { stw.execAliasCommand("SELECTSECT") },),
                      iup.Item( iup.Attr("TITLE","FENCE"), func (arg *iup.ItemAction) { stw.execAliasCommand("FENCE") },),
                      iup.Item( iup.Attr("TITLE","ERROR ELEM"), func (arg *iup.ItemAction) { stw.execAliasCommand("ERRORELEM") },),
                      iup.Separator(),
                      iup.Item( iup.Attr("TITLE","ADD LINE ELEM"), func (arg *iup.ItemAction) { stw.execAliasCommand("ADDLINEELEM") },),
                      iup.Item( iup.Attr("TITLE","ADD PLATE ELEM"), func (arg *iup.ItemAction) { stw.execAliasCommand("ADDPLATEELEM") },),
                      iup.Item( iup.Attr("TITLE","HATCH PLATE ELEM"), func (arg *iup.ItemAction) { stw.execAliasCommand("HATCHPLATEELEM") },),
                      iup.Item( iup.Attr("TITLE","EDIT PLATE ELEM"), func (arg *iup.ItemAction) { stw.execAliasCommand("EDITPLATEELEM") },),
                      iup.Separator(),
                      iup.Item( iup.Attr("TITLE","MATCH PROP"), func (arg *iup.ItemAction) { stw.execAliasCommand("MATCHPROP") },),
                      iup.Separator(),
                      iup.Item( iup.Attr("TITLE","COPY ELEM"), func (arg *iup.ItemAction) { stw.execAliasCommand("COPYELEM") },),
                      iup.Item( iup.Attr("TITLE","MOVE ELEM"), func (arg *iup.ItemAction) { stw.execAliasCommand("MOVEELEM") },),
                      iup.Item( iup.Attr("TITLE","MOVE NODE"), func (arg *iup.ItemAction) { stw.execAliasCommand("MOVENODE") },),
                      iup.Separator(),
                      iup.Item( iup.Attr("TITLE","SEARCH ELEM"), func (arg *iup.ItemAction) { stw.execAliasCommand("SEARCHELEM") },),
                      iup.Item( iup.Attr("TITLE","NODE TO ELEM"), func (arg *iup.ItemAction) { stw.execAliasCommand("NODETOELEM") },),
                      iup.Item( iup.Attr("TITLE","ELEM TO NODE"), func (arg *iup.ItemAction) { stw.execAliasCommand("ELEMTONODE") },),
                      iup.Item( iup.Attr("TITLE","CONNECTED"), func (arg *iup.ItemAction) { stw.execAliasCommand("CONNECTED") },),
                      iup.Item( iup.Attr("TITLE","ON NODE"), func (arg *iup.ItemAction) { stw.execAliasCommand("ONNODE") },),
                      iup.Separator(),
                      iup.Item( iup.Attr("TITLE","NODE NO REFERENCE"), func (arg *iup.ItemAction) { stw.execAliasCommand("NODENOREFERENCE") },),
                      iup.Item( iup.Attr("TITLE","ELEM SAME NODE"), func (arg *iup.ItemAction) { stw.execAliasCommand("ELEMSAMENODE") },),
                      iup.Item( iup.Attr("TITLE","NODE DUPLICATION"), func (arg *iup.ItemAction) { stw.execAliasCommand("NODEDUPLICATION") },),
                      iup.Separator(),
                      iup.Item( iup.Attr("TITLE","CAT BY NODE"), func (arg *iup.ItemAction) { stw.execAliasCommand("CATBYNODE") },),
                      iup.Item( iup.Attr("TITLE","JOIN LINE ELEM"), func (arg *iup.ItemAction) { stw.execAliasCommand("JOINLINEELEM") },),
                      iup.Separator(),
                      iup.Item( iup.Attr("TITLE","EXTRACT ARCLM"), func (arg *iup.ItemAction) { stw.execAliasCommand("EXTRACTARCLM") },),
                      iup.Separator(),
                      iup.Item( iup.Attr("TITLE","DIVIDE AT ONS"), func (arg *iup.ItemAction) { stw.execAliasCommand("DIVIDEATONS") },),
                      iup.Item( iup.Attr("TITLE","DIVIDE AT MID"), func (arg *iup.ItemAction) { stw.execAliasCommand("DIVIDEATMID") },),
                      iup.Separator(),
                      iup.Item( iup.Attr("TITLE","INTERSECT"), func (arg *iup.ItemAction) { stw.execAliasCommand("INTERSECT") },),
                      iup.Item( iup.Attr("TITLE","TRIM"), func (arg *iup.ItemAction) { stw.execAliasCommand("TRIM") },),
                      iup.Item( iup.Attr("TITLE","EXTEND"), func (arg *iup.ItemAction) { stw.execAliasCommand("EXTEND") },),
                      iup.Separator(),
                      iup.Item( iup.Attr("TITLE","REACTION"), func (arg *iup.ItemAction) { stw.execAliasCommand("REACTION") },),
            ),
        ),
        iup.SubMenu("TITLE=Tool",
            iup.Menu(
                iup.Item(
                    iup.Attr("TITLE","RC lst"),
                    func (arg *iup.ItemAction) {
                        StartTool("tool/rclst/rclst.html")
                    },
                ),
                iup.Item(
                    iup.Attr("TITLE","Fig2 Keyword"),
                    func (arg *iup.ItemAction) {
                        StartTool("tool/fig2/fig2.html")
                    },
                ),
            ),
        ),
        iup.SubMenu("TITLE=Help",
            iup.Menu(
                iup.Item(
                    iup.Attr("TITLE","Release Note"),
                    func (arg *iup.ItemAction) {
                        ShowReleaseNote()
                    },
                ),
                iup.Item(
                    iup.Attr("TITLE","About"),
                    func (arg *iup.ItemAction) {
                        stw.ShowAbout()
                    },
                ),
            ),
        ),
    ).SetName("main_menu")
    stw.canv = iup.Canvas(
                "CANFOCUS=YES",
                "BGCOLOR=\"0 0 0\"",
                "BORDER=YES",
                // "DRAWSIZE=1920x1080",
                "EXPAND=YES",
                "NAME=canvas",
                func (arg *iup.CommonMap) {
                    stw.cdcanv = cd.CreateCanvas(cd.CD_IUP, stw.canv)
                    stw.dbuff  = cd.CreateCanvas(cd.CD_DBUFFER, stw.cdcanv)
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
                func (arg *iup.CanvasResize) {
                    stw.dbuff.Activate()
                    if stw.Frame != nil {
                        stw.Redraw()
                    } else {
                        stw.dbuff.Flush()
                    }
                },
                func (arg *iup.CanvasAction) {
                    stw.dbuff.Activate()
                    if stw.Frame != nil {
                        stw.Redraw()
                    } else {
                        stw.dbuff.Flush()
                    }
                },
                func (arg *iup.CanvasDropFiles) {
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
                func (arg *iup.CommonKeyAny) {
                    key := iup.KeyState(arg.Key)
                    switch key.Key() {
                    case KEY_ENTER:
                        stw.feedCommand()
                    case KEY_ESCAPE:
                        stw.cline.SetAttribute("VALUE", "")
                    case '[':
                        if key.IsCtrl() {
                            stw.cline.SetAttribute("VALUE", "")
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
                            stw.SaveAS()
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
                     fmt.Sprintf("SIZE=%dx%d",datalabelwidth+datatextwidth, dataheight),)
    pers.SetCallback(func (arg *iup.MouseButton) {
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
    vlabels := iup.Vbox(iup.Hbox(datasectionlabel("VIEW"),),
                            iup.Hbox(datalabel("GFACT"), stw.Show.Label["GFACT"], ),
                            iup.Hbox(pers),
                        iup.Hbox(datasectionlabel("DISTS"),),
                            iup.Hbox(datalabel("R"), stw.Show.Label["DISTR"], ),
                            iup.Hbox(datalabel("L"), stw.Show.Label["DISTL"], ),
                        iup.Hbox(datasectionlabel("ANGLE"),),
                            iup.Hbox(datalabel("PHI"), stw.Show.Label["PHI"], ),
                            iup.Hbox(datalabel("THETA"), stw.Show.Label["THETA"], ),
                        iup.Hbox(datasectionlabel("FOCUS"),),
                            iup.Hbox(datalabel("X"), stw.Show.Label["FOCUSX"], ),
                            iup.Hbox(datalabel("Y"), stw.Show.Label["FOCUSY"], ),
                            iup.Hbox(datalabel("Z"), stw.Show.Label["FOCUSZ"], ),
                        iup.Hbox(datasectionlabel("CENTER"),),
                            iup.Hbox(datalabel("X"), stw.Show.Label["CENTERX"], ),
                            iup.Hbox(datalabel("Y"), stw.Show.Label["CENTERY"], ),
                        "MARGIN=0x0",
                       )
    tgelem := iup.Vbox(datasectionlabel("ETYPE"),
                           stw.Show.Label["COLUMN"],
                           iup.Hbox(stw.Show.Label["COLUMN_NZ"],
                                    stw.Show.Label["COLUMN_QX"],
                                    stw.Show.Label["COLUMN_QY"],
                                    stw.Show.Label["COLUMN_MZ"],
                                    stw.Show.Label["COLUMN_MX"],
                                    stw.Show.Label["COLUMN_MY"]),
                           stw.Show.Label["GIRDER"],
                           iup.Hbox(stw.Show.Label["GIRDER_NZ"],
                                    stw.Show.Label["GIRDER_QX"],
                                    stw.Show.Label["GIRDER_QY"],
                                    stw.Show.Label["GIRDER_MZ"],
                                    stw.Show.Label["GIRDER_MX"],
                                    stw.Show.Label["GIRDER_MY"]),
                           stw.Show.Label["BRACE"],
                           iup.Hbox(stw.Show.Label["BRACE_NZ"],
                                    stw.Show.Label["BRACE_QX"],
                                    stw.Show.Label["BRACE_QY"],
                                    stw.Show.Label["BRACE_MZ"],
                                    stw.Show.Label["BRACE_MX"],
                                    stw.Show.Label["BRACE_MY"]),
                           iup.Hbox(stw.Show.Label["WALL"], stw.Show.switchLabel(WALL), stw.Show.Label["WBRACE"],),
                           iup.Hbox(stw.Show.Label["WALL_NZ"],
                                    stw.Show.Label["WALL_QX"],
                                    stw.Show.Label["WALL_QY"],
                                    stw.Show.Label["WALL_MZ"],
                                    stw.Show.Label["WALL_MX"],
                                    stw.Show.Label["WALL_MY"]),
                           iup.Hbox(stw.Show.Label["SLAB"], stw.Show.switchLabel(SLAB), stw.Show.Label["SBRACE"],),
                           iup.Hbox(stw.Show.Label["SLAB_NZ"],
                                    stw.Show.Label["SLAB_QX"],
                                    stw.Show.Label["SLAB_QY"],
                                    stw.Show.Label["SLAB_MZ"],
                                    stw.Show.Label["SLAB_MX"],
                                    stw.Show.Label["SLAB_MY"]),)
    tgncap := iup.Vbox(datasectionlabel("NODE CAPTION"),
                           stw.Show.Label["NC_NUM"],
                           iup.Hbox(stw.Show.Label["NC_DX"],
                                    stw.Show.Label["NC_DY"],
                                    stw.Show.Label["NC_DZ"],),
                           iup.Hbox(stw.Show.Label["NC_RX"],
                                    stw.Show.Label["NC_RY"],
                                    stw.Show.Label["NC_RZ"],),)
    tgecap := iup.Vbox(datasectionlabel("ELEM CAPTION"),
                           stw.Show.Label["EC_NUM"],
                           stw.Show.Label["EC_SECT"],
                           iup.Hbox(stw.Show.Label["EC_RATE_L"],stw.Show.Label["EC_RATE_S"]),)
    tgcolmode:= iup.Vbox(datasectionlabel("COLOR MODE"),stw.Show.Label["COLORMODE"])
    tgparam := iup.Vbox(datasectionlabel("PARAMETER"),
                           iup.Hbox(datalabel("PERIOD"), stw.Show.Label["PERIOD"],),
                           iup.Hbox(datalabel("GAXIS"), stw.Show.Label["GAXISSIZE"],),
                           iup.Hbox(datalabel("EAXIS"), stw.Show.Label["EAXISSIZE"],),
                           iup.Hbox(datalabel("BOND"), stw.Show.Label["BONDSIZE"],),
                           iup.Hbox(datalabel("CONF"), stw.Show.Label["CONFSIZE"],),
                           iup.Hbox(datalabel("DFACT"), stw.Show.Label["DFACT"],),
                           iup.Hbox(datalabel("MFACT"), stw.Show.Label["MFACT"],),)
    tgshow := iup.Vbox(datasectionlabel("SHOW"),
                           stw.Show.Label["GAXIS"],
                           stw.Show.Label["EAXIS"],
                           stw.Show.Label["BOND"],
                           stw.Show.Label["CONF"],
                           stw.Show.Label["KIJUN"],
                           stw.Show.Label["DEFORMATION"])
    tgrang := iup.Vbox(datasectionlabel("RANGE"),
                           iup.Hbox(datalabel("Xmax"), stw.Show.Label["XMAX"]),
                           iup.Hbox(datalabel("Xmin"), stw.Show.Label["XMIN"]),
                           iup.Hbox(datalabel("Ymax"), stw.Show.Label["YMAX"]),
                           iup.Hbox(datalabel("Ymin"), stw.Show.Label["YMIN"]),
                           iup.Hbox(datalabel("Zmax"), stw.Show.Label["ZMAX"]),
                           iup.Hbox(datalabel("Zmin"), stw.Show.Label["ZMIN"]),)
    stw.Dlg = iup.Dialog(
                  iup.Attrs(
                      "MENU", "main_menu",
                      "TITLE", "st",
                      "SHRINK", "YES",
                      "SIZE", windowSize,
                      "FGCOLOR", labelFGColor,
                      "BGCOLOR", labelBGColor,
                  ),
                  iup.Vbox(
                      iup.Hbox(iup.Vbox(vlabels,tgrang,tgelem,),
                               iup.Vbox(tgcolmode, tgparam, tgshow, tgncap, tgecap,),
                               stw.canv,
                               ),
                      iup.Hbox(stw.hist,),
                      iup.Hbox(stw.cname, stw.cline,),
                  ),
                  func (arg *iup.DialogClose) {
                      arg.Return = iup.CLOSE
                  },
                  func (arg *iup.CommonGetFocus) {
                      stw.Redraw()
                  },
              )
    stw.Dlg.Map()
    // stw.dbuff.VectorFont("ipam.ttf")
    // stw.dbuff.VectorFontSize(0.1, 0.1)
    stw.dbuff.TextAlignment(DefaultTextAlignment)
    iup.SetHandle("mainwindow", stw.Dlg)
    stw.EscapeAll()
    return stw
}
// }}}


type Show struct {// {{{
    Window *Window

    ColorMode uint

    NodeCaption uint
    ElemCaption uint

    GlobalAxis bool
    GlobalAxisSize float64
    ElementAxis bool
    ElementAxisSize float64

    NodeNormal bool
    NodeNormalSize float64
    ElemNormal bool
    ElemNormalSize float64

    PlateEdgeColor int

    Conf bool
    ConfSize float64
    ConfColor int

    Bond bool
    BondSize float64
    BondColor int

    Period string

    Deformation bool
    Dfact float64

    Stress   map[int]uint

    Mfact float64
    MomentColor int
    StressTextColor int

    Kijun bool
    KijunSize float64

    Select bool

    Sect     map[int]bool
    Etype    map[int]bool

    Label    map[string]*iup.Handle

    Xrange []float64
    Yrange []float64
    Zrange []float64

    Property bool
    Selected []*iup.Handle
    Props []*iup.Handle
}

func NewShow(stw *Window) *Show {
    s := new(Show)

    s.Window = stw

    s.ColorMode = ECOLOR_SECT

    s.NodeCaption = NC_NUM
    s.ElemCaption = 0

    s.GlobalAxis = true
    s.GlobalAxisSize = 1.0
    s.ElementAxis = false
    s.ElementAxisSize = 0.5

    s.NodeNormal = false
    s.NodeNormalSize = 0.2
    s.ElemNormal = false
    s.ElemNormalSize = 0.2

    s.PlateEdgeColor = cd.CD_GRAY

    s.Bond = true
    s.BondSize = 3.0
    s.BondColor = cd.CD_GRAY
    s.Conf = true
    s.ConfSize = 9.0
    s.ConfColor = cd.CD_GRAY

    s.Period = "L"

    s.Deformation = false
    s.Dfact = 100.0

    s.Stress = map[int]uint{COLUMN: 0, GIRDER: 0, BRACE: 0, WBRACE: 0, SBRACE: 0}
    s.Mfact = 0.5
    s.MomentColor = cd.CD_DARK_MAGENTA
    s.StressTextColor = cd.CD_GRAY

    s.Kijun = false
    s.KijunSize = 12.0

    s.Select = false

    s.Sect  = make(map[int]bool)
    s.Etype = make(map[int]bool)
    for i, _ := range ETYPES {
        if i == WBRACE || i == SBRACE {
            s.Etype[i] = false
        } else {
            s.Etype[i] = true
        }
    }
    s.Label = make(map[string]*iup.Handle)
    s.Label["GAXIS"]       = s.displayLabel("GAXIS", true)
    s.Label["EAXIS"]       = s.displayLabel("EAXIS", false)
    s.Label["BOND"]        = s.displayLabel("BOND", true)
    s.Label["CONF"]        = s.displayLabel("CONF", true)
    s.Label["KIJUN"]       = s.displayLabel("KIJUN", false)
    s.Label["DEFORMATION"] = s.displayLabel("DEFORMATION", false)
    s.Label["GFACT"]       = datatext("0.0")
    s.Label["DISTR"]       = datatext("0.0")
    s.Label["DISTL"]       = datatext("0.0")
    s.Label["PHI"]         = datatext("0.0")
    s.Label["THETA"]       = datatext("0.0")
    s.Label["FOCUSX"]      = datatext("0.0")
    s.Label["FOCUSY"]      = datatext("0.0")
    s.Label["FOCUSZ"]      = datatext("0.0")
    s.Label["CENTERX"]     = datatext("0.0")
    s.Label["CENTERY"]     = datatext("0.0")
    s.Label["COLUMN"]      = s.etypeLabel("  COLUMN", datalabelwidth+datatextwidth, COLUMN, true)
    s.Label["GIRDER"]      = s.etypeLabel("  GIRDER", datalabelwidth+datatextwidth, GIRDER, true)
    s.Label["BRACE"]       = s.etypeLabel("  BRACE",  datalabelwidth+datatextwidth, BRACE, true)
    s.Label["WALL"]        = s.etypeLabel("  WALL",   datalabelwidth-10, WALL, true)
    s.Label["SLAB"]        = s.etypeLabel("  SLAB",   datalabelwidth-10, SLAB, true)
    s.Label["WBRACE"]      = s.etypeLabel("WBRACE", datatextwidth, WBRACE, false)
    s.Label["SBRACE"]      = s.etypeLabel("SBRACE", datatextwidth, SBRACE, false)
    s.Label["NC_NUM"]      = s.captionLabel("NODE", "  CODE", datalabelwidth+datatextwidth, NC_NUM, true)
    s.Label["NC_DX"]       = s.captionLabel("NODE", "  dX", (datalabelwidth+datatextwidth)/3, NC_DX, false)
    s.Label["NC_DY"]       = s.captionLabel("NODE", "  dY", (datalabelwidth+datatextwidth)/3, NC_DY, false)
    s.Label["NC_DZ"]       = s.captionLabel("NODE", "  dZ", (datalabelwidth+datatextwidth)/3, NC_DZ, false)
    s.Label["NC_RX"]       = s.captionLabel("NODE", "  Rx", (datalabelwidth+datatextwidth)/3, NC_RX, false)
    s.Label["NC_RY"]       = s.captionLabel("NODE", "  Ry", (datalabelwidth+datatextwidth)/3, NC_RY, false)
    s.Label["NC_RZ"]       = s.captionLabel("NODE", "  Rz", (datalabelwidth+datatextwidth)/3, NC_RZ, false)
    s.Label["EC_NUM"]      = s.captionLabel("ELEM", "  CODE", datalabelwidth+datatextwidth, EC_NUM, false)
    s.Label["EC_SECT"]     = s.captionLabel("ELEM", "  SECT", datalabelwidth+datatextwidth, EC_SECT, false)
    s.Label["EC_RATE_L"]   = s.captionLabel("ELEM", "  RATE_L", datalabelwidth, EC_RATE_L, false)
    s.Label["EC_RATE_S"]   = s.captionLabel("ELEM", "RATE_S", datatextwidth, EC_RATE_S, false)
    s.Label["COLORMODE"]   = s.toggleLabel(2, ECOLORS)
    s.Label["PERIOD"]      = datatext("L")
    s.Label["GAXISSIZE"]   = datatext("1.0")
    s.Label["EAXISSIZE"]   = datatext("0.5")
    s.Label["BONDSIZE"]    = datatext("3.0")
    s.Label["CONFSIZE"]    = datatext("9.0")
    s.Label["DFACT"]       = datatext("100.0")
    s.Label["MFACT"]       = datatext("0.5")
    s.Label["XMAX"]        = datatext("1000.0")
    s.Label["XMIN"]        = datatext("-100.0")
    s.Label["YMAX"]        = datatext("1000.0")
    s.Label["YMIN"]        = datatext("-100.0")
    s.Label["ZMAX"]        = datatext("1000.0")
    s.Label["ZMIN"]        = datatext("-100.0")
    s.Label["COLUMN_NZ"]   = s.stressLabel(COLUMN, 0)
    s.Label["COLUMN_QX"]   = s.stressLabel(COLUMN, 1)
    s.Label["COLUMN_QY"]   = s.stressLabel(COLUMN, 2)
    s.Label["COLUMN_MZ"]   = s.stressLabel(COLUMN, 3)
    s.Label["COLUMN_MX"]   = s.stressLabel(COLUMN, 4)
    s.Label["COLUMN_MY"]   = s.stressLabel(COLUMN, 5)
    s.Label["GIRDER_NZ"]   = s.stressLabel(GIRDER, 0)
    s.Label["GIRDER_QX"]   = s.stressLabel(GIRDER, 1)
    s.Label["GIRDER_QY"]   = s.stressLabel(GIRDER, 2)
    s.Label["GIRDER_MZ"]   = s.stressLabel(GIRDER, 3)
    s.Label["GIRDER_MX"]   = s.stressLabel(GIRDER, 4)
    s.Label["GIRDER_MY"]   = s.stressLabel(GIRDER, 5)
    s.Label["BRACE_NZ"]    = s.stressLabel(BRACE, 0)
    s.Label["BRACE_QX"]    = s.stressLabel(BRACE, 1)
    s.Label["BRACE_QY"]    = s.stressLabel(BRACE, 2)
    s.Label["BRACE_MZ"]    = s.stressLabel(BRACE, 3)
    s.Label["BRACE_MX"]    = s.stressLabel(BRACE, 4)
    s.Label["BRACE_MY"]    = s.stressLabel(BRACE, 5)
    s.Label["WALL_NZ"]     = s.stressLabel(WBRACE, 0)
    s.Label["WALL_QX"]     = s.stressLabel(WBRACE, 1)
    s.Label["WALL_QY"]     = s.stressLabel(WBRACE, 2)
    s.Label["WALL_MZ"]     = s.stressLabel(WBRACE, 3)
    s.Label["WALL_MX"]     = s.stressLabel(WBRACE, 4)
    s.Label["WALL_MY"]     = s.stressLabel(WBRACE, 5)
    s.Label["SLAB_NZ"]     = s.stressLabel(SBRACE, 0)
    s.Label["SLAB_QX"]     = s.stressLabel(SBRACE, 1)
    s.Label["SLAB_QY"]     = s.stressLabel(SBRACE, 2)
    s.Label["SLAB_MZ"]     = s.stressLabel(SBRACE, 3)
    s.Label["SLAB_MX"]     = s.stressLabel(SBRACE, 4)
    s.Label["SLAB_MY"]     = s.stressLabel(SBRACE, 5)

    s.Xrange = []float64{ -100.0, 1000.0 }
    s.Yrange = []float64{ -100.0, 1000.0 }
    s.Zrange = []float64{ -100.0, 1000.0 }

    s.Property = false

    return s
}

func (show *Show) All () {
    for i, et := range ETYPES {
        show.Etype[i] = true
        if lb, ok := show.Label[et]; ok {
            lb.SetAttribute("FGCOLOR", labelFGColor)
        }
    }
    for i, _ := range show.Sect {
        show.Sect[i] = true
        if lb, ok := show.Label[fmt.Sprintf("%d",i)]; ok {
            lb.SetAttribute("FGCOLOR", labelFGColor)
        }
    }
    show.Xrange[0] = -100.0
    show.Xrange[1] = 1000.0
    show.Yrange[0] = -100.0
    show.Yrange[1] = 1000.0
    show.Zrange[0] = -100.0
    show.Zrange[1] = 1000.0
    show.Label["XMIN"].SetAttribute("VALUE", "-100.0")
    show.Label["XMAX"].SetAttribute("VALUE", "1000.0")
    show.Label["YMIN"].SetAttribute("VALUE", "-100.0")
    show.Label["YMAX"].SetAttribute("VALUE", "1000.0")
    show.Label["ZMIN"].SetAttribute("VALUE", "-100.0")
    show.Label["ZMAX"].SetAttribute("VALUE", "1000.0")
}

func (show *Show) SetColorMode (mode uint) {
    show.ColorMode = mode
    show.Label["COLORMODE"].SetAttribute("VALUE", fmt.Sprintf("  %s", ECOLORS[mode]))
}

// }}}


func (stw *Window) Chdir(dir string) error {
    if _, err := os.Stat(dir); err != nil {
        return err
    } else {
        stw.Home = dir
        stw.Frame.Home = dir
        return nil
    }
}


// Open// {{{
func (stw *Window) Open() {
    if name,ok := iup.GetOpenFile(stw.Home, "*.inp"); ok {
        err := stw.OpenFile(name)
        if err != nil {
            fmt.Println(err)
        }
        stw.Redraw()
    }
}

func (stw *Window) Insert() {
    stw.execAliasCommand("INSERT")
}

func (stw *Window) Reload() {
    if stw.Frame != nil {
        stw.Deselect()
        v := stw.Frame.View
        stw.OpenFile(stw.Frame.Path)
        stw.Frame.View = v
        stw.Redraw()
    }
}

func (stw *Window) OpenDxf() {
    if name,ok := iup.GetOpenFile(stw.Home, "*.dxf"); ok {
        err := stw.OpenFile(name)
        if err != nil {
            fmt.Println(err)
        }
        stw.Redraw()
    }
}

func (stw *Window) OpenFile(fn string) error {
    var err error
    stw.Frame = NewFrame()
    w, h := stw.cdcanv.GetSize()
    stw.CanvasSize = []float64{ float64(w), float64(h) }
    stw.Frame.View.Center[0]=stw.CanvasSize[0]*0.5
    stw.Frame.View.Center[1]=stw.CanvasSize[1]*0.5
    switch filepath.Ext(fn) {
    case ".inp":
        err = stw.Frame.ReadInp(fn, []float64{0.0,0.0,0.0})
        if err != nil {
            return err
        }
    case ".dxf":
        err = stw.Frame.ReadDxf(fn, []float64{0.0,0.0,0.0})
        if err != nil {
            return err
        }
        stw.Frame.SetFocus()
        stw.DrawFrameNode()
        stw.ShowCenter()
    }
    openstr := fmt.Sprintf("OPEN: %s", fn)
    stw.addHistory(openstr)
    stw.Dlg.SetAttribute("TITLE", stw.Frame.Name)
    stw.Frame.Home = stw.Home
    stw.LinkTextValue()
    return nil
}
// }}}


// Save// {{{
func (stw *Window) Save() {
    stw.SaveFile(filepath.Join(stw.Home, "hogtxt.inp"))
}

func (stw *Window) SaveAS() {
    var err error
    if name,ok := iup.GetSaveFile(filepath.Dir(stw.Frame.Path),"*.inp"); ok {
        fn := Ce(name, ".inp")
        err = stw.SaveFile(fn)
        if err == nil && fn != stw.Frame.Path {
            if stw.Yn("SAVE AS", ".lst, .fig2, .kjnファイルがあればコピーしますか?") {
                for _, ext := range []string{".lst", ".fig2", ".kjn"} {
                    src := Ce(stw.Frame.Path, ext)
                    dst := Ce(name, ext)
                    if FileExists(src) {
                        err = CopyFile(src, dst)
                        if err == nil {
                            stw.addHistory(fmt.Sprintf("COPY: %s", dst))
                        }
                    }
                }
            }
            stw.Frame.Name = filepath.Base(fn)
            stw.Frame.Project = ProjectName(fn)
            path, err := filepath.Abs(fn)
            if err != nil {
                stw.Frame.Path = fn
            } else {
                stw.Frame.Path = path
            }
            stw.Dlg.SetAttribute("TITLE", stw.Frame.Name)
            stw.Frame.Home = stw.Home
        }
    }
}

func (stw *Window) SaveFile(fn string) error {
    err := stw.Frame.WriteInp(fn)
    if err != nil {
        return err
    }
    savestr := fmt.Sprintf("SAVE: %s", fn)
    stw.addHistory(savestr)
    return nil
}
// }}}


// Read// {{{
func (stw *Window) Read() {
    if stw.Frame != nil {
        if name,ok := iup.GetOpenFile("",""); ok {
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
        exts := []string{".inl", ".ihx", ".ihy", ".otl", ".ohx", ".ohy", ".rat2", ".wgt", ".kjn"}
        for _, ext := range exts {
            name := Ce(stw.Frame.Path, ext)
            err = stw.ReadFile(name)
            if err != nil {
                if ext == ".rat2" {
                    err = stw.ReadFile(Ce(stw.Frame.Path, ".rat"))
                    if err == nil { continue }
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
            x = 0.0; y = 0.0; z = 0.0
        }
        err = stw.Frame.ReadInp(filename, []float64{x,y,z})
    case ".inl", ".ihx", ".ihy":
        err = stw.Frame.ReadData(filename)
    case ".otl", ".ohx", ".ohy":
        err = stw.Frame.ReadResult(filename)
    case ".rat", ".rat2":
        err = stw.Frame.ReadRat(filename)
    case ".wgt":
        err = stw.Frame.ReadWgt(filename)
    case ".kjn":
        err = stw.Frame.ReadKjn(filename)
    }
    if err != nil {
        stw.addHistory(fmt.Sprintf("NOT READ: %s", filename))
        return err
    }
    stw.addHistory(fmt.Sprintf("READ: %s", filename))
    return nil
}
// }}}


// Print
func (stw *Window) Print() {
    pcanv := cd.CreatePrinter(cd.CD_PRINTER, "test -d")
    if pcanv == nil {
        return
    }
    v := stw.Frame.View.Copy()
    pw, ph := pcanv.GetSize()
    factor := math.Min(float64(pw)/stw.CanvasSize[0],  float64(ph)/stw.CanvasSize[1])
    stw.Frame.View.Gfact *= factor
    stw.Frame.View.Center[0] = 0.5*float64(pw)
    stw.Frame.View.Center[1] = 0.5*float64(ph)
    stw.Show.ConfSize *= factor
    stw.Show.BondSize *= factor
    stw.DrawFrame(pcanv, ECOLOR_BLACK)
    pcanv.Kill()
    stw.Frame.View.Gfact /= factor
    stw.Frame.View.Center[0] = 0.5*stw.CanvasSize[0]
    stw.Frame.View.Center[1] = 0.5*stw.CanvasSize[1]
    stw.Show.ConfSize /= factor
    stw.Show.BondSize /= factor
    stw.Frame.View = v
    stw.Redraw()
}

// func (stw *Window) PrintPDF() {
//     pcanv := cd.CreatePrinter(cd.CD_PDF, "test.pdf")
//     if pcanv == nil {
//         return
//     }
//     stw.DrawFrame(pcanv, ECOLOR_BLACK)
//     pcanv.Kill()
// }


func (stw *Window) EditInp() {
    if stw.Frame != nil {
        cmd := exec.Command("cmd", "/C", "start", stw.Frame.Path)
        cmd.Start()
    }
}


func StartTool(fn string) {
    cmd := exec.Command("cmd", "/C", "start", fn)
    cmd.Start()
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
            ff := strings.Split(dlg.GetAttribute("VALUE"),",")
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
            ff := strings.Split(dlg.GetAttribute("VALUE"),",")
            fs, _ := strconv.ParseInt(strings.Trim(ff[1]," "), 10, 64)
            canvasFontFace = ff[0]
            canvasFontSize = int(fs)
            canvasFontColor = ColorInt(dlg.GetAttribute("COLOR"))
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
        stw.cline.SetAttribute("VALUE", "")
        stw.execAliasCommand(command)
    }
}

func (stw *Window) addHistory(str string) {
    stw.hist.SetAttribute("APPEND", str)
    lnum, err := strconv.ParseInt(stw.hist.GetAttribute("LINECOUNT"),10,64)
    if err != nil {
        fmt.Println(err)
        return
    }
    setpos := fmt.Sprintf("%d:1",int(lnum)+1)
    stw.hist.SetAttribute("SCROLLTO",setpos)
}

func (stw *Window) execCommand(com *Command) {
    stw.addHistory(com.Name)
    stw.cname.SetAttribute("VALUE", com.Name)
    stw.lastcommand = com
    com.Exec(stw)
}

func (stw *Window) execAliasCommand(al string) {
    if stw.Frame == nil {
        stw.Open()
        return
    }
    al = strings.ToUpper(al)
    if value,ok := aliases[al]; ok {
        stw.execCommand(value)
        return
    } else if value,ok := Commands[al]; ok{
        stw.execCommand(value)
        return
    } else {
        switch {
        default:
            stw.addHistory(fmt.Sprintf("command doesn't exist: %s", al))
        case axrn_minmax.MatchString(al):
            var axis int
            fs := axrn_minmax.FindStringSubmatch(al)
            min, _ := strconv.ParseFloat(fs[1], 64)
            max, _ := strconv.ParseFloat(fs[3], 64)
            tmp := strings.ToUpper(fs[2])
            for i, val := range []string{"X", "Y", "Z"} {
                if tmp == val { axis = i; break }
            }
            axisrange(stw, axis, min, max, false)
            stw.addHistory(fmt.Sprintf("AxisRange: %.3f <= %s <= %.3f", min, tmp, max))
        case axrn_min.MatchString(al):
            var axis int
            fs := axrn_min.FindStringSubmatch(al)
            min, _ := strconv.ParseFloat(fs[1], 64)
            tmp := strings.ToUpper(fs[2])
            for i, val := range []string{"X", "Y", "Z"} {
                if tmp == val { axis = i; break }
            }
            axisrange(stw, axis, min, 1000.0, false)
            stw.addHistory(fmt.Sprintf("AxisRange: %.3f <= %s", min, tmp))
        case axrn_max.MatchString(al):
            var axis int
            fs := axrn_max.FindStringSubmatch(al)
            max, _ := strconv.ParseFloat(fs[2], 64)
            tmp := strings.ToUpper(fs[1])
            for i, val := range []string{"X", "Y", "Z"} {
                if tmp == val { axis = i; break }
            }
            axisrange(stw, axis, -100.0, max, false)
            stw.addHistory(fmt.Sprintf("AxisRange: %s <= %.3f", tmp, max))
        case axrn_eq.MatchString(al):
            var axis int
            fs := axrn_eq.FindStringSubmatch(al)
            val, _ := strconv.ParseFloat(fs[2], 64)
            tmp := strings.ToUpper(fs[1])
            for i, val := range []string{"X", "Y", "Z"} {
                if tmp == val { axis = i; break }
            }
            axisrange(stw, axis, val, val, false)
            stw.addHistory(fmt.Sprintf("AxisRange: %s = %.3f", tmp, val))
        }
        return
    }
}

func axisrange(stw *Window, axis int, min, max float64, any bool) {
    tmpnodes := make([]*Node, 0)
    for _, n := range stw.Frame.Nodes {
        if !(min <= n.Coord[axis] && n.Coord[axis] <= max) {
            tmpnodes = append(tmpnodes, n)
            n.Hide = true
        } else {
            n.Hide = false
        }
    }
    var tmpelems []*Elem
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
        stw.Show.Xrange[0] = min
        stw.Show.Xrange[1] = max
        stw.Show.Label["XMIN"].SetAttribute("VALUE", fmt.Sprintf("%.3f", min))
        stw.Show.Label["XMAX"].SetAttribute("VALUE", fmt.Sprintf("%.3f", max))
    case 1:
        stw.Show.Yrange[0] = min
        stw.Show.Yrange[1] = max
        stw.Show.Label["YMIN"].SetAttribute("VALUE", fmt.Sprintf("%.3f", min))
        stw.Show.Label["YMAX"].SetAttribute("VALUE", fmt.Sprintf("%.3f", max))
    case 2:
        stw.Show.Zrange[0] = min
        stw.Show.Zrange[1] = max
        stw.Show.Label["ZMIN"].SetAttribute("VALUE", fmt.Sprintf("%.3f", min))
        stw.Show.Label["ZMAX"].SetAttribute("VALUE", fmt.Sprintf("%.3f", max))
    }
    stw.Redraw()
}
// }}}


// Draw// {{{
// TODO: drawing selected elem with dot line is not working
func (stw *Window) DrawFrame(canv *cd.Canvas, color uint) {
    if stw.Frame != nil {
        // stw.UpdateShowRange() // TODO: ShowRange
        canv.Hatch(cd.CD_FDIAGONAL)
        canv.Clear()
        stw.Frame.View.Set(0)
        if stw.Show.GlobalAxis {
            stw.DrawGlobalAxis(canv)
        }
        if stw.Show.Kijun {
            canv.TextAlignment(cd.CD_CENTER)
            canv.Foreground(cd.CD_GRAY)
            for _, k := range stw.Frame.Kijuns {
                if k.Hide { continue }
                k.Pstart = stw.Frame.View.ProjectCoord(k.Start)
                k.Pend   = stw.Frame.View.ProjectCoord(k.End)
                k.Draw(canv, stw.Show)
            }
            canv.TextAlignment(DefaultTextAlignment)
        }
        canv.Foreground(cd.CD_WHITE)
        for _, n := range(stw.Frame.Nodes) {
            stw.Frame.View.ProjectNode(n)
            if stw.Show.Deformation { stw.Frame.View.ProjectDeformation(n, stw.Show) }
            if n.Hide { continue }
            if n.Lock {
                canv.Foreground(LOCKED_NODE_COLOR)
            } else {
                canv.Foreground(canvasFontColor)
            }
            switch n.ConfState() {
            case CONF_PIN:
                canv.Foreground(cd.CD_GREEN)
            case CONF_FIX:
                canv.Foreground(cd.CD_DARK_GREEN)
            }
            for _, j := range(stw.SelectNode) {
                if j==n {canv.Foreground(cd.CD_RED); break}
            }
            n.Draw(canv, stw.Show)
        }
        canv.LineStyle(cd.CD_CONTINUOUS)
        canv.Hatch(cd.CD_FDIAGONAL)
        if !stw.Show.Select {
            els := SortedElem(stw.Frame.Elems, func (e *Elem) float64 { return -e.DistFromProjection() })
            loop:
                for _, el := range(els) {
                    if el.Hide { continue }
                    if !stw.Show.Etype[el.Etype] { el.Hide = true; continue }
                    if b, ok := stw.Show.Sect[el.Sect.Num]; ok {
                        if !b { el.Hide = true; continue }
                    }
                    canv.LineStyle(cd.CD_CONTINUOUS)
                    canv.Hatch(cd.CD_FDIAGONAL)
                    for _, j := range(stw.SelectElem) {
                        if j==el { continue loop }
                    }
                    if el.Lock {
                        canv.Foreground(LOCKED_ELEM_COLOR)
                    } else {
                        switch color {
                        case ECOLOR_WHITE:
                            canv.Foreground(cd.CD_WHITE)
                        case ECOLOR_BLACK:
                            canv.Foreground(cd.CD_BLACK)
                        case ECOLOR_SECT:
                            canv.Foreground(el.Sect.Color)
                        case ECOLOR_RATE:
                            canv.Foreground(Rainbow(el.RateMax(stw.Show), RateBoundary))
                        case ECOLOR_HEIGHT:
                            canv.Foreground(Rainbow(el.MidPoint()[2], HeightBoundary))
                        }
                    }
                    el.Draw(canv, stw.Show)
                }
        }
        canv.InteriorStyle(cd.CD_HATCH)
        canv.Hatch(cd.CD_DIAGCROSS)
        for _, el := range(stw.SelectElem) {
            canv.LineStyle(cd.CD_DOTTED)
            if el == nil || el.Hide { continue }
            if !stw.Show.Etype[el.Etype] { el.Hide = true; continue }
            if b, ok := stw.Show.Sect[el.Sect.Num]; ok {
                if !b { el.Hide = true; continue }
            }
            if el.Lock {
                canv.Foreground(LOCKED_ELEM_COLOR)
            } else {
                switch color {
                case ECOLOR_WHITE:
                    canv.Foreground(cd.CD_WHITE)
                case ECOLOR_BLACK:
                    canv.Foreground(cd.CD_BLACK)
                case ECOLOR_SECT:
                    canv.Foreground(el.Sect.Color)
                case ECOLOR_RATE:
                    canv.Foreground(Rainbow(el.RateMax(stw.Show), RateBoundary))
                case ECOLOR_HEIGHT:
                    canv.Foreground(Rainbow(el.MidPoint()[2], HeightBoundary))
                }
            }
            el.Draw(canv, stw.Show)
        }
        canv.Flush()
        stw.SetViewData()
    }
}

func (stw *Window) Redraw() {
    stw.DrawFrame(stw.dbuff,stw.Show.ColorMode)
    if stw.Show.Property {
        stw.UpdatePropertyDialog()
    }
}

func (stw *Window) DrawFrameNode() {
    if stw.Frame != nil {
        stw.dbuff.Clear()
        stw.Frame.View.Set(0)
        if stw.Show.GlobalAxis {
            stw.DrawGlobalAxis(stw.dbuff)
        }
        for _, n := range(stw.Frame.Nodes) {
            stw.Frame.View.ProjectNode(n)
            if n.Lock {
                stw.dbuff.Foreground(LOCKED_NODE_COLOR)
            } else {
                stw.dbuff.Foreground(canvasFontColor)
            }
            for _, j := range(stw.SelectNode) {
                if j==n {stw.dbuff.Foreground(cd.CD_RED); break}
            }
            n.Draw(stw.dbuff, stw.Show)
        }
        if len(stw.SelectElem) > 0  && stw.SelectElem[0] != nil {
            stw.dbuff.Hatch(cd.CD_DIAGCROSS)
            for _, el := range(stw.SelectElem) {
                stw.dbuff.LineStyle(cd.CD_DOTTED)
                if el == nil || el.Hide { continue }
                if el.Lock {
                    stw.dbuff.Foreground(LOCKED_ELEM_COLOR)
                } else {
                    stw.dbuff.Foreground(cd.CD_WHITE)
                }
                el.Draw(stw.dbuff, stw.Show)
            }
            stw.dbuff.LineStyle(cd.CD_CONTINUOUS)
            stw.dbuff.Hatch(cd.CD_FDIAGONAL)
        }
        stw.dbuff.Flush()
    }
}

func (stw *Window) DrawGlobalAxis (canv *cd.Canvas) {
    origin := stw.Frame.View.ProjectCoord([]float64{ 0.0, 0.0, 0.0 })
    xaxis  := stw.Frame.View.ProjectCoord([]float64{ stw.Show.GlobalAxisSize, 0.0, 0.0 })
    yaxis  := stw.Frame.View.ProjectCoord([]float64{ 0.0, stw.Show.GlobalAxisSize, 0.0 })
    zaxis  := stw.Frame.View.ProjectCoord([]float64{ 0.0, 0.0, stw.Show.GlobalAxisSize })
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

func Arrow (cvs *cd.Canvas, x1, y1, x2, y2, size, theta float64) {
    c := size*math.Cos(theta); s := size*math.Sin(theta)
    cvs.FLine(x1, y1, x2, y2)
    cvs.FLine(x2, y2, x2 + ((x1-x2)*c - (y1-y2)*s), y2 + ( (x1-x2)*s + (y1-y2)*c))
    cvs.FLine(x2, y2, x2 + ((x1-x2)*c + (y1-y2)*s), y2 + (-(x1-x2)*s + (y1-y2)*c))
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
        stw.Show.Label["GFACT"].SetAttribute("VALUE", fmt.Sprintf("%.1f", stw.Frame.View.Gfact))
        stw.Show.Label["DISTR"].SetAttribute("VALUE", fmt.Sprintf("%.1f", stw.Frame.View.Dists[0]))
        stw.Show.Label["DISTL"].SetAttribute("VALUE", fmt.Sprintf("%.1f", stw.Frame.View.Dists[1]))
        stw.Show.Label["PHI"].SetAttribute("VALUE", fmt.Sprintf("%.1f", stw.Frame.View.Angle[0]))
        stw.Show.Label["THETA"].SetAttribute("VALUE", fmt.Sprintf("%.1f", stw.Frame.View.Angle[1]))
        stw.Show.Label["FOCUSX"].SetAttribute("VALUE", fmt.Sprintf("%.1f", stw.Frame.View.Focus[0]))
        stw.Show.Label["FOCUSY"].SetAttribute("VALUE", fmt.Sprintf("%.1f", stw.Frame.View.Focus[1]))
        stw.Show.Label["FOCUSZ"].SetAttribute("VALUE", fmt.Sprintf("%.1f", stw.Frame.View.Focus[2]))
        stw.Show.Label["CENTERX"].SetAttribute("VALUE", fmt.Sprintf("%.1f", stw.Frame.View.Center[0]))
        stw.Show.Label["CENTERY"].SetAttribute("VALUE", fmt.Sprintf("%.1f", stw.Frame.View.Center[1]))
    }
}

func (stw *Window) Bbox() (xmin, xmax, ymin, ymax float64) {
    var mins, maxs [2]float64
    first := true
    for _, j := range(stw.Frame.Nodes) {
        if j.Hide { continue }
        if first {
            for k:=0; k<2; k++ {
                mins[k] = j.Pcoord[k]
                maxs[k] = j.Pcoord[k]
            }
            first = false
        } else {
            for k:=0; k<2; k++ {
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
    stw.Show.Xrange[0] = xmin; stw.Show.Xrange[1] = xmax
    stw.Show.Yrange[0] = ymin; stw.Show.Yrange[1] = ymax
    stw.Show.Zrange[0] = zmin; stw.Show.Zrange[1] = zmax
    stw.Show.Label["XMAX"].SetAttribute("VALUE", fmt.Sprintf("%.3f", xmax))
    stw.Show.Label["XMIN"].SetAttribute("VALUE", fmt.Sprintf("%.3f", xmin))
    stw.Show.Label["YMAX"].SetAttribute("VALUE", fmt.Sprintf("%.3f", ymax))
    stw.Show.Label["YMIN"].SetAttribute("VALUE", fmt.Sprintf("%.3f", ymin))
    stw.Show.Label["ZMAX"].SetAttribute("VALUE", fmt.Sprintf("%.3f", zmax))
    stw.Show.Label["ZMIN"].SetAttribute("VALUE", fmt.Sprintf("%.3f", zmin))
}

func (stw *Window) HideNodes () {
    for _, n := range stw.Frame.Nodes {
        n.Hide = true
    }
    for _, el := range stw.Frame.Elems {
        if el.Hide { continue }
        for _, en := range el.Enod {
            en.Hide = false
        }
    }
    stw.SetShowRange()
}

func (stw *Window) HideNotSelected() {
    if stw.SelectElem != nil{
        for _, n := range  stw.Frame.Nodes {
            n.Hide = true
        }
        for _, el := range(stw.Frame.Elems) {
            el.Hide = true
        }
        for _, el := range(stw.SelectElem) {
            if el != nil {
                el.Hide = false
                for _, en := range el.Enod {
                    en.Hide = false
                }
            }
        }
    }
    stw.SetShowRange()
    stw.Deselect()
    stw.Redraw()
}

func (stw *Window) HideSelected() {
    if stw.SelectElem != nil{
        for _, n := range  stw.Frame.Nodes {
            n.Hide = true
        }
        for _, el := range(stw.SelectElem) {
            if el != nil { el.Hide = true }
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
    stw.Deselect()
    stw.Redraw()
}

func (stw *Window) LockSelected() {
    if stw.SelectElem != nil{
        for _, el := range(stw.SelectElem) {
            if el != nil {
                el.Lock = true
                for _, en := range el.Enod {
                    en.Lock = true
                }
            }
        }
    }
    stw.Deselect()
    stw.Redraw()
}

func (stw *Window) DeleteSelected() {
    if stw.SelectElem != nil{
        for _, el := range(stw.SelectElem) {
            if el != nil && !el.Lock {
                delete(stw.Frame.Elems, el.Num)
            }
        }
    }
    stw.Deselect()
    stw.Redraw()
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
    stw.Show.All()
    stw.Redraw()
}

func (stw *Window) UnlockAll() {
    for _, el := range(stw.Frame.Elems) {
        el.Lock = false
    }
    for _, n := range stw.Frame.Nodes {
        n.Lock = false
    }
    stw.Redraw()
}

func (stw *Window) ShowAtCanvasCenter (canv *cd.Canvas) {
    for _, n := range(stw.Frame.Nodes) {
        stw.Frame.View.ProjectNode(n)
    }
    xmin, xmax, ymin, ymax := stw.Bbox()
    w, h := canv.GetSize()
    scale := math.Min(float64(w)/(xmax-xmin), float64(h)/(ymax-ymin))*0.9
    if stw.Frame.View.Perspective {
        stw.Frame.View.Dists[1] *= scale
    } else {
        stw.Frame.View.Gfact *= scale
    }
    stw.Frame.View.Center[0]=float64(w)*0.5 + scale*(stw.Frame.View.Center[0] - 0.5*(xmax+xmin))
    stw.Frame.View.Center[1]=float64(h)*0.5 + scale*(stw.Frame.View.Center[1] - 0.5*(ymax+ymin))
}

func (stw *Window) ShowCenter() {
    stw.ShowAtCanvasCenter(stw.cdcanv)
    stw.Redraw()
}

func (stw *Window) SetAngle (phi, theta float64) {
    if stw.Frame != nil {
        stw.Frame.View.Angle[0] = phi
        stw.Frame.View.Angle[1] = theta
        stw.ShowCenter()
    }
}
// }}}


// Select// {{{
func (stw *Window) PickNode(x, y int) (rtn *Node) {
    mindist := float64(dotSelectPixel)
    for _, v := range(stw.Frame.Nodes) {
        if v.Hide { continue }
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
        left := min(stw.startX, stw.endX); right := max(stw.startX, stw.endX)
        bottom := min(stw.startY, stw.endY); top := max(stw.startY, stw.endY)
        if (right - left < dotSelectPixel) && (top - bottom < dotSelectPixel) {
            n := stw.PickNode(left, bottom)
            stw.MergeSelectNode([]*Node{n}, isShift(arg.Status))
        } else {
            tmpselect := make([]*Node, len(stw.Frame.Nodes))
            i := 0
            for _, v := range(stw.Frame.Nodes) {
                if v.Hide { continue }
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
        stw.startX = arg.X; stw.startY = arg.Y
        stw.endX = arg.X; stw.endY = arg.Y
        first = 1
    }
}

// line: (x1, y1) -> (x2, y2), dot: (dx, dy)
// provided that x1*y2-x2*y1>0
//     if rtn>0: dot is the same side as (0, 0)
//     if rtn==0: dot is on the line
//     if rtn<0: dot is the opposite side to (0, 0)
func DotLine (x1, y1, x2, y2, dx, dy int) int {
    return (x1*y2 + x2*dy + dx*y1) - (x1*dy + x2*y1 + dx*y2)
}
func FDotLine (x1, y1, x2, y2, dx, dy float64) float64 {
    return (x1*y2 + x2*dy + dx*y1) - (x1*dy + x2*y1 + dx*y2)
}

func (stw *Window) PickElem (x, y int) (rtn *Elem) {
    el := stw.PickLineElem(x, y)
    if el == nil {
        els := stw.PickPlateElem(x, y)
        if len(els) > 0 { el = els[0] }
    }
    return el
}

func (stw *Window) PickLineElem(x, y int) (rtn *Elem) {
    mindist := float64(dotSelectPixel)
    for _, v := range(stw.Frame.Elems) {
        if v.Hide { continue }
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

func abs (val int) int {
    if val >= 0 {
        return val
    } else {
        return -val
    }
}
func (stw *Window) PickPlateElem(x, y int) []*Elem {
    rtn := make(map[int]*Elem)
    for _, el := range(stw.Frame.Elems) {
        if el.Hide { continue }
        if !el.IsLineElem() {
            add := true
            sign := 0
            for i:=0; i<int(el.Enods); i++ {
                var j int
                if i==int(el.Enods)-1 { j = 0 } else { j = i+1 }
                if FDotLine(el.Enod[i].Pcoord[0], el.Enod[i].Pcoord[1], el.Enod[j].Pcoord[0], el.Enod[j].Pcoord[1], float64(x), float64(y)) > 0 {
                    sign++
                } else {
                    sign--
                }
                if i+1 != abs(sign) { add = false; break }
            }
            if add { rtn[el.Num] = el }
        }
    }
    return SortedElem(rtn, func(e *Elem) float64 { return e.DistFromProjection() })
}

func (stw *Window) SelectElemStart(arg *iup.MouseButton) {
    stw.dbuff.UpdateYAxis(&arg.Y)
    if arg.Pressed == 0 { // Released
        left := min(stw.startX, stw.endX); right := max(stw.startX, stw.endX)
        bottom := min(stw.startY, stw.endY); top := max(stw.startY, stw.endY)
        if (right - left < dotSelectPixel) && (top - bottom < dotSelectPixel) {
            el := stw.PickLineElem(left, bottom)
            if el == nil {
                els := stw.PickPlateElem(left, bottom)
                if len(els) > 0 { el = els[0] }
            }
            if el != nil {
                stw.MergeSelectElem([]*Elem{el}, isShift(arg.Status))
            } else {
                stw.SelectElem = make([]*Elem, 0)
            }
        } else {
            tmpselectnode := make([]*Node, len(stw.Frame.Nodes))
            i := 0
            for _, v := range(stw.Frame.Nodes) {
                if float64(left) <= v.Pcoord[0] && v.Pcoord[0] <= float64(right) && float64(bottom) <= v.Pcoord[1] && v.Pcoord[1] <= float64(top) {
                    tmpselectnode[i] = v
                    i++
                }
            }
            tmpselectelem := make([]*Elem, len(stw.Frame.Elems))
            k := 0
            switch selectDirection {
            case SD_FROMLEFT:
                for _, el := range(stw.Frame.Elems) {
                    if el.Hide { continue }
                    add := true
                    for _, en := range(el.Enod) {
                        var j int
                        for j=0; j<i; j++ {
                            if en == tmpselectnode[j] { break }
                        }
                        if j==i { add = false; break }
                    }
                    if add { tmpselectelem[k] = el; k++ }
                }
            case SD_FROMRIGHT:
                for _, el := range(stw.Frame.Elems) {
                    if el.Hide { continue }
                    add := false
                    for _, en := range(el.Enod) {
                        found := false
                        for j:=0; j<i; j++ {
                            if en == tmpselectnode[j] { found = true; break }
                        }
                        if found { add = true; break }
                    }
                    if add { tmpselectelem[k] = el; k++ }
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
        stw.startX = arg.X; stw.startY = arg.Y
        stw.endX = arg.X; stw.endY = arg.Y
        first = 1
    }
}

func (stw *Window) SelectElemFenceStart(arg *iup.MouseButton) {
    stw.dbuff.UpdateYAxis(&arg.Y)
    if arg.Pressed == 0 { // Released
        tmpselectelem := make([]*Elem, len(stw.Frame.Elems))
        k := 0
        for _, el := range(stw.Frame.Elems) {
            if el.Hide { continue }
            add := false
            sign := 0
            for i, en := range(el.Enod) {
                if FDotLine(float64(stw.startX), float64(stw.startY), float64(stw.endX), float64(stw.endY), en.Pcoord[0], en.Pcoord[1]) > 0 {
                    sign++
                } else {
                    sign--
                }
                if i+1 != abs(sign) { add = true; break }
            }
            if add {
                if el.IsLineElem() {
                    if FDotLine(el.Enod[0].Pcoord[0], el.Enod[0].Pcoord[1], el.Enod[1].Pcoord[0], el.Enod[1].Pcoord[1], float64(stw.startX), float64(stw.startY)) * FDotLine(el.Enod[0].Pcoord[0], el.Enod[0].Pcoord[1], el.Enod[1].Pcoord[0], el.Enod[1].Pcoord[1], float64(stw.endX), float64(stw.endY)) < 0 {
                        tmpselectelem[k] = el
                        k++
                    }
                } else {
                    addx := false
                    sign := 0
                    for i, j := range(el.Enod) {
                        if float64(max(stw.startX, stw.endX)) < j.Pcoord[0] {
                            sign++
                        } else if j.Pcoord[0] < float64(min(stw.startX, stw.endX)) {
                            sign--
                        }
                        if i+1 != abs(sign) { addx = true; break }
                    }
                    if addx {
                        addy := false
                        sign := 0
                        for i, j := range(el.Enod) {
                            if float64(max(stw.startY, stw.endY)) < j.Pcoord[1] {
                                sign++
                            } else if j.Pcoord[1] < float64(min(stw.startY, stw.endY)) {
                                sign--
                            }
                            if i+1 != abs(sign) { addy = true; break }
                        }
                        if addy { tmpselectelem[k] = el; k++ }
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
        stw.startX = arg.X; stw.startY = arg.Y
        stw.endX = arg.X; stw.endY = arg.Y
        first = 1
    }
}

func (stw *Window) MergeSelectNode (nodes []*Node, isshift bool) {
    k := len(nodes)
    if isshift {
        for l:=0; l<k; l++ {
            for m, el := range(stw.SelectNode) {
                if el == nodes[l] {
                    if m == len(stw.SelectNode) - 1 {
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
        for l:=0; l<k; l++ {
            add = true
            for _, n := range(stw.SelectNode) {
                if n == nodes[l] { add = false; break}
            }
            if add { stw.SelectNode = append(stw.SelectNode, nodes[l]) }
        }
    }
}

func (stw *Window) MergeSelectElem (elems []*Elem, isshift bool) {
    k := len(elems)
    if isshift {
        for l:=0; l<k; l++ {
            for m, el := range(stw.SelectElem) {
                if el == elems[l] {
                    if m == len(stw.SelectElem) - 1 {
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
        for l:=0; l<k; l++ {
            add = true
            for _, n := range(stw.SelectElem) {
                if n == elems[l] { add = false; break}
            }
            if add { stw.SelectElem = append(stw.SelectElem, elems[l]) }
        }
    }
}

func (stw *Window) SelectNodeMotion(arg *iup.MouseMotion) {
    if stw.startX <= arg.X {
        selectDirection = SD_FROMLEFT
        if first == 1 {
            first = 0
        } else {
            stw.cdcanv.Rect(min(stw.endX, stw.startX), max(stw.endX, stw.startX), min(stw.startY, stw.endY), max(stw.startY, stw.endY))
        }
        stw.cdcanv.Rect(arg.X, stw.startX, min(stw.startY, arg.Y), max(stw.startY, arg.Y))
        stw.endX = arg.X; stw.endY = arg.Y
    } else {
        stw.cdcanv.LineStyle(cd.CD_DASHED)
        selectDirection = SD_FROMRIGHT
        if first == 1 {
            first = 0
        } else {
            stw.cdcanv.Rect(min(stw.endX, stw.startX), max(stw.endX, stw.startX), min(stw.startY, stw.endY), max(stw.startY, stw.endY))
        }
        stw.cdcanv.Rect(stw.startX, arg.X, min(stw.startY, arg.Y), max(stw.startY, arg.Y))
        stw.endX = arg.X; stw.endY = arg.Y
        stw.cdcanv.LineStyle(cd.CD_CONTINUOUS)
    }
}

func (stw *Window) SelectElemMotion(arg *iup.MouseMotion) {
    selectDirection = SD_FROMLEFT
    if stw.startX <= arg.X {
        if first == 1 {
            first = 0
        } else {
            stw.cdcanv.Rect(min(stw.endX, stw.startX), max(stw.endX, stw.startX), min(stw.startY, stw.endY), max(stw.startY, stw.endY))
        }
        stw.cdcanv.Rect(arg.X, stw.startX, min(stw.startY, arg.Y), max(stw.startY, arg.Y))
        stw.endX = arg.X; stw.endY = arg.Y
    } else {
        selectDirection = SD_FROMRIGHT
        stw.cdcanv.LineStyle(cd.CD_DASHED)
        if first == 1 {
            first = 0
        } else {
            stw.cdcanv.Rect(min(stw.endX, stw.startX), max(stw.endX, stw.startX), min(stw.startY, stw.endY), max(stw.startY, stw.endY))
        }
        stw.cdcanv.Rect(stw.startX, arg.X, min(stw.startY, arg.Y), max(stw.startY, arg.Y))
        stw.endX = arg.X; stw.endY = arg.Y
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
        stw.cdcanv.Line(stw.startX, stw.startY, arg.X, stw.startY)
        stw.endX = arg.X; stw.endY = stw.startY
    } else {
        stw.cdcanv.Line(stw.startX, stw.startY, arg.X, arg.Y)
        stw.endX = arg.X; stw.endY = arg.Y
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
    stw.cdcanv.Line(x, y, arg.X, arg.Y)
    stw.endX = arg.X; stw.endY = arg.Y
}

func (stw *Window) TailPolygon(ns []*Node, arg *iup.MouseMotion) {
    stw.dbuff.InteriorStyle(cd.CD_HATCH)
    stw.dbuff.Hatch(cd.CD_FDIAGONAL)
    if first == 1 {
        first = 0
    } else {
        l := len(ns)+1
        num := 0
        coords := make([][]float64, l)
        for i:=0; i<l-1; i++ {
            if ns[i] == nil { continue }
            coords[num] = ns[i].Pcoord
            num++
        }
        coords[num] = []float64{float64(stw.endX), float64(stw.endY)}
        stw.cdcanv.Polygon(cd.CD_FILL, coords[:num+1]...)
        // stw.cdcanv.Polygon(cd.CD_FILL, coords[0], coords[num-1], coords[num])
    }
    l := len(ns)+1
    num := 0
    coords := make([][]float64, l)
    for i:=0; i<l-1; i++ {
        if ns[i] == nil { continue }
        coords[num] = ns[i].Pcoord
        num++
    }
    coords[num] = []float64{float64(arg.X), float64(arg.Y)}
    stw.cdcanv.Polygon(cd.CD_FILL, coords[:num+1]...)
    stw.endX = arg.X; stw.endY = arg.Y
}

func (stw *Window) Deselect() {
    stw.SelectNode = make([]*Node, 0)
    stw.SelectElem = make([]*Elem, 0)
}
// }}}


// Query// {{{
func (stw *Window) Yn (title, question string) bool {
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

func (stw *Window) Yna (title, question, another string) int {
    ans := iup.Alarm(title, question, "はい", "いいえ", another)
    return ans
}

func (stw *Window) Query (question string) (rtn string, err error) {
    var ans string
    var er error
    func (title string, ans *string, er *error) {
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
                    fmt.Sprintf("SIZE=%dx%d",datalabelwidth+datatextwidth, dataheight),
                    )
        text = iup.Text(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
                        fmt.Sprintf("FGCOLOR=\"%s\"", labelFGColor),
                        fmt.Sprintf("BGCOLOR=\"%s\"", commandBGColor),
                        "VALUE=\"\"",
                        "BORDER=NO",
                        "ALIGNMENT=ALEFT",
                        fmt.Sprintf("SIZE=%dx%d",datatextwidth, dataheight),
                        )
        text.SetCallback( func (arg *iup.CommonKeyAny) {
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
        dlg = iup.Dialog(iup.Hbox(label, text,))
        dlg.SetAttribute("TITLE", title)
        dlg.Popup(iup.CENTER, iup.CENTER)
    }(question, &ans, &er)
    return ans, er
}

func (stw *Window) QueryCoord (title string) (x, y, z float64, err error) {
    var xx, yy, zz float64
    var er error
    func (title string, x, y, z *float64, er *error) {
        labels := make([]*iup.Handle, 3)
        texts := make([]*iup.Handle, 3)
        var dlg *iup.Handle
        returnvalues := func() {
            rtn := make([]float64, 3)
            for i:=0; i<3; i++ {
                val := texts[i].GetAttribute("VALUE")
                if val == "" {
                    rtn[i]=0.0
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
            *x = rtn[0]; *y = rtn[1]; *z = rtn[2]; *er = nil
        }
        for i, d := range []string{"X", "Y", "Z"} {
            labels[i] = datalabel(d)
            texts[i]  = datatext("")
            texts[i].SetCallback( func (arg *iup.CommonKeyAny) {
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
    stw.canv.SetCallback( func (arg *iup.MouseButton) {
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
                                              stw.Frame.SetFocus()
                                              stw.DrawFrameNode()
                                              stw.ShowCenter()
                                          } else {
                                              stw.dbuff.UpdateYAxis(&arg.Y)
                                              stw.startX = arg.X; stw.startY = arg.Y
                                          }
                                      }
                                  case BUTTON_RIGHT:
                                      if arg.Pressed == 0 {
                                          if v := stw.cline.GetAttribute("VALUE"); v!="" {
                                              stw.feedCommand()
                                          } else {
                                              if time.Since(pressed).Seconds() < repeatcommand {
                                                  if stw.lastcommand != nil {
                                                      stw.execCommand(stw.lastcommand)
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
                                      if isDouble(arg.Status) { stw.Open() }
                                  }
                              }
                          })
}

func (stw *Window) CB_MouseMotion() {
    stw.canv.SetCallback( func (arg *iup.MouseMotion) {
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
                                      if isShift(arg.Status) {
                                          stw.Frame.View.Center[0] += float64(arg.X - stw.startX) * CanvasMoveSpeedX
                                          stw.Frame.View.Center[1] += float64(arg.Y - stw.startY) * CanvasMoveSpeedY
                                      } else {
                                          stw.Frame.View.Angle[0] -= float64(arg.Y - stw.startY) * CanvasRotateSpeedY
                                          stw.Frame.View.Angle[1] -= float64(arg.X - stw.startX) * CanvasRotateSpeedX
                                      }
                                      stw.DrawFrameNode()
                                  }
                              }
                          })
}

func (stw *Window) CB_CanvasWheel() {
    stw.canv.SetCallback( func (arg *iup.CanvasWheel) {
                              if stw.Frame != nil {
                                  stw.dbuff.UpdateYAxis(&arg.Y)
                                  val := math.Pow(2.0, float64(arg.Delta)/CanvasScaleSpeed)
                                  stw.Frame.View.Center[0] += (val-1.0)*(stw.Frame.View.Center[0] - float64(arg.X))
                                  stw.Frame.View.Center[1] += (val-1.0)*(stw.Frame.View.Center[1] - float64(arg.Y))
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
        stw.cline.SetAttribute("INSERT", string(key.Key()))
        // iup.SetFocus(stw.cline)
    case ';':
        if stw.Frame != nil {
            switch stw.Frame.Project {
            default:
                stw.cline.SetAttribute("INSERT", ";")
            case "ven":
                stw.cline.SetAttribute("INSERT", "V4")
            }
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
    case 'Q':
        if key.IsCtrl() {
            if !stw.Show.Property {
                stw.PropertyDialog()
            }
        }
    case 'D':
        if key.IsCtrl() {
            stw.HideNotSelected()
        }
    case 'S':
        if key.IsCtrl() {
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
            stw.SaveAS()
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
}

func (stw *Window) CB_CommonKeyAny() {
    stw.canv.SetCallback( func (arg *iup.CommonKeyAny) {
                              key := iup.KeyState(arg.Key)
                              stw.DefaultKeyAny(key)
                          })
}

func (stw *Window) LinkProperty (index int, eval func ()) {
    stw.Show.Props[index].SetCallback(func (arg *iup.CommonGetFocus) {
                                          stw.Show.Props[index].SetAttribute("SELECTION", "1:100")
                                      })
    stw.Show.Props[index].SetCallback(func (arg *iup.CommonKillFocus) {
                                          if stw.Frame != nil && stw.SelectElem != nil {
                                              eval()
                                          }
                                      })
    stw.Show.Props[index].SetCallback(func (arg *iup.CommonKeyAny) {
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
                                          }
                                      })
}

func (stw *Window) PropertyDialog () {
    selected := make([]*iup.Handle, 4)
    stw.Show.Selected = make([]*iup.Handle, 4)
    selected[0] = datasectionlabel("LINE")
    selected[1] = datasectionlabel(" (TOTAL LENGTH)")
    selected[2] = datasectionlabel("PLATE")
    selected[3] = datasectionlabel(" (TOTAL AREA)")
    labels := make([]*iup.Handle, 8)
    stw.Show.Props = make([]*iup.Handle, 8)
    labels[0] = datasectionlabel("CODE")
    labels[1] = datasectionlabel("SECTION")
    labels[2] = datasectionlabel("ETYPE")
    labels[3] = datasectionlabel("ENODS")
    labels[4] = datasectionlabel("ENOD")
    labels[5] = datasectionlabel("")
    labels[6] = datasectionlabel("")
    labels[7] = datasectionlabel("")
    for i:=0; i<4; i++ {
        stw.Show.Selected[i] = datatext("-")
    }
    for i:=0; i<8; i++ {
        stw.Show.Props[i] = datatext("-")
    }
    stw.LinkProperty(1, func () {
                            val, err := strconv.ParseInt(stw.Show.Props[1].GetAttribute("VALUE"), 10, 64)
                            if err == nil {
                                if sec, ok := stw.Frame.Sects[int(val)]; ok {
                                    for _, el := range stw.SelectElem {
                                        if el != nil {
                                            el.Sect = sec
                                        }
                                    }
                                }
                            }
                            stw.Redraw()
                        })
    stw.LinkProperty(2, func () {
                            word := stw.Show.Props[2].GetAttribute("VALUE")
                            var val int
                            col := regexp.MustCompile("(?i)co(l(u(m(n){0,1}){0,1}){0,1}){0,1}$")
                            gir := regexp.MustCompile("(?i)gi(r(d(e(r){0,1}){0,1}){0,1}){0,1}$")
                            bra := regexp.MustCompile("(?i)br(a(c(e){0,1}){0,1}){0,1}$")
                            wal := regexp.MustCompile("(?i)wa(l){0,2}$")
                            sla := regexp.MustCompile("(?i)sl(a(b){0,1}){0,1}$")
                            switch {
                            case col.MatchString(word) :
                                val = COLUMN
                            case gir.MatchString(word) :
                                val = GIRDER
                            case bra.MatchString(word) :
                                val = BRACE
                            case wal.MatchString(word) :
                                val = WALL
                            case sla.MatchString(word) :
                                val = SLAB
                            }
                            stw.Show.Props[2].SetAttribute("VALUE", ETYPES[val])
                            if val != 0 {
                                for _, el := range stw.SelectElem {
                                    if el != nil {
                                        el.Etype = val
                                    }
                                }
                            }
                            stw.Redraw()
                        })
    dlg := iup.Dialog(
               iup.Vbox(iup.Hbox(selected[0], stw.Show.Selected[0]),
                        iup.Hbox(selected[1], stw.Show.Selected[1]),
                        iup.Hbox(selected[2], stw.Show.Selected[2]),
                        iup.Hbox(selected[3], stw.Show.Selected[3]),
                        iup.Hbox(labels[0], stw.Show.Props[0]),
                        iup.Hbox(labels[1], stw.Show.Props[1]),
                        iup.Hbox(labels[2], stw.Show.Props[2]),
                        iup.Hbox(labels[3], stw.Show.Props[3]),
                        iup.Hbox(labels[4], stw.Show.Props[4]),
                        iup.Hbox(labels[5], stw.Show.Props[5]),
                        iup.Hbox(labels[6], stw.Show.Props[6]),
                        iup.Hbox(labels[7], stw.Show.Props[7]),),
           )
    dlg.SetAttribute("TITLE", "Property")
    dlg.SetAttribute("PARENTDIALOG", "mainwindow")
    dlg.SetCallback(func (arg *iup.DialogClose) {
                        stw.Show.Property = false
                    })
    dlg.Map()
    dlg.Show()
    stw.Show.Property = true
}

func (stw *Window) UpdatePropertyDialog () {
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
                    stw.Show.Props[0].SetAttribute("VALUE", fmt.Sprintf("%d", el.Num))
                    stw.Show.Props[1].SetAttribute("VALUE", fmt.Sprintf("%d", el.Sect.Num))
                    stw.Show.Props[2].SetAttribute("VALUE", ETYPES[el.Etype])
                    stw.Show.Props[3].SetAttribute("VALUE", fmt.Sprintf("%d", el.Enods))
                    for i:=0; i<el.Enods; i++ {
                        if i>=4 { break }
                        stw.Show.Props[4+i].SetAttribute("VALUE", fmt.Sprintf("%d", el.Enod[i].Num))
                    }
                }
                selected = true
            }
        }
        if !selected {
            for i:=0; i<4; i++ {
                stw.Show.Selected[i].SetAttribute("VALUE", "-")
            }
            for i:=0; i<8; i++ {
                stw.Show.Props[i].SetAttribute("VALUE", "-")
            }
        }
        if lines > 0 {
            stw.Show.Selected[0].SetAttribute("VALUE", fmt.Sprintf("%d", lines))
            stw.Show.Selected[1].SetAttribute("VALUE", fmt.Sprintf("%.3f", length))
        }
        if plates > 0 {
            stw.Show.Selected[2].SetAttribute("VALUE", fmt.Sprintf("%d", plates))
            stw.Show.Selected[3].SetAttribute("VALUE", fmt.Sprintf("%.3f", area))
        }
    } else {
        for i:=0; i<4; i++ {
            stw.Show.Selected[i].SetAttribute("VALUE", "-")
        }
        for i:=0; i<8; i++ {
            stw.Show.Props[i].SetAttribute("VALUE", "-")
        }
    }
}

func (stw *Window) CMenu () {
    stw.context = iup.Menu(
                       iup.Attrs("BGCOLOR", labelBGColor,),
                       iup.SubMenu("TITLE=File",
                           iup.Menu(
                               iup.Item(
                                   iup.Attr("TITLE","Open\tCtrl+O"),
                                   iup.Attr("TIP","Open file"),
                                   func (arg *iup.ItemAction) {
                                       runtime.GC()
                                       stw.Open()
                                   },
                               ),
                               iup.Item(
                                   iup.Attr("TITLE","Insert\tCtrl+I"),
                                   iup.Attr("TIP","Insert frame"),
                                   func (arg *iup.ItemAction) {
                                       stw.Insert()
                                   },
                               ),
                               iup.Item(
                                   iup.Attr("TITLE","Reload\tCtrl+M"),
                                   func (arg *iup.ItemAction) {
                                       runtime.GC()
                                       stw.Reload()
                                   },
                               ),
                               iup.Item(
                                   iup.Attr("TITLE","Save\tCtrl+S"),
                                   iup.Attr("TIP","Save file"),
                                   func (arg *iup.ItemAction) {
                                       stw.Save()
                                   },
                               ),
                               iup.Item(
                                   iup.Attr("TITLE","Save As\tCtrl+A"),
                                   iup.Attr("TIP","Save file As"),
                                   func (arg *iup.ItemAction) {
                                       stw.SaveAS()
                                   },
                               ),
                               iup.Item(
                                   iup.Attr("TITLE","Read"),
                                   iup.Attr("TIP","Read file"),
                                   func (arg *iup.ItemAction) {
                                       stw.Read()
                                   },
                               ),
                               iup.Item(
                                   iup.Attr("TITLE","Read All\tCtrl+R"),
                                   iup.Attr("TIP","Read all file"),
                                   func (arg *iup.ItemAction) {
                                       stw.ReadAll()
                                   },
                               ),
                               iup.Item(
                                   iup.Attr("TITLE","Print"),
                                   func (arg *iup.ItemAction) {
                                       stw.Print()
                                   },
                               ),
                               // iup.Item(
                               //     iup.Attr("TITLE","Print PDF"),
                               //     func (arg *iup.ItemAction) {
                               //         stw.PrintPDF()
                               //     },
                               // ),
                               iup.Separator(),
                               iup.Item(
                                   iup.Attr("TITLE","Quit"),
                                   iup.Attr("TIP","Exit Application"),
                                   func (arg *iup.ItemAction) {
                                       arg.Return = iup.CLOSE
                                   },
                               ),
                           ),
                       ),
                       iup.SubMenu("TITLE=Edit",
                           iup.Menu(
                               iup.Item(
                                   iup.Attr("TITLE","Command Font"),
                                   func (arg *iup.ItemAction) {
                                       stw.SetFont(FONT_COMMAND)
                                   },
                               ),
                               iup.Item(
                                   iup.Attr("TITLE","Canvas Font"),
                                   func (arg *iup.ItemAction) {
                                       stw.SetFont(FONT_CANVAS)
                                   },
                               ),
                               iup.Item(
                                   iup.Attr("TITLE","Property\tCtrl+Q"),
                                   func (arg *iup.ItemAction) {
                                       if !stw.Show.Property {
                                           stw.PropertyDialog()
                                       }
                                   },
                               ),
                           ),
                       ),
                       iup.SubMenu("TITLE=View",
                           iup.Menu(
                               iup.Item(
                                   iup.Attr("TITLE","Top"),
                                   func (arg *iup.ItemAction) {
                                       stw.SetAngle(90.0, -90.0)
                                   },
                               ),
                               iup.Item(
                                   iup.Attr("TITLE","Front"),
                                   func (arg *iup.ItemAction) {
                                       stw.SetAngle(0.0, -90.0)
                                   },
                               ),
                               iup.Item(
                                   iup.Attr("TITLE","Back"),
                                   func (arg *iup.ItemAction) {
                                       stw.SetAngle(0.0, 90.0)
                                   },
                               ),
                               iup.Item(
                                   iup.Attr("TITLE","Right"),
                                   func (arg *iup.ItemAction) {
                                       stw.SetAngle(0.0, 0.0)
                                   },
                               ),
                               iup.Item(
                                   iup.Attr("TITLE","Left"),
                                   func (arg *iup.ItemAction) {
                                       stw.SetAngle(0.0, 180.0)
                                   },
                               ),
                           ),
                       ),
                       iup.SubMenu("TITLE=Show",
                           iup.Menu(
                               iup.Item(
                                   iup.Attr("TITLE","Show All\tCtrl+S"),
                                   func (arg *iup.ItemAction) {
                                       stw.ShowAll()
                                   },
                               ),
                               iup.Item(
                                   iup.Attr("TITLE","Hide Selected Elems\tCtrl+H"),
                                   func (arg *iup.ItemAction) {
                                       stw.HideSelected()
                                   },
                               ),
                               iup.Item(
                                   iup.Attr("TITLE","Show Selected Elems\tCtrl+D"),
                                   func (arg *iup.ItemAction) {
                                       stw.HideNotSelected()
                                   },
                               ),
                           ),
                       ),
                      iup.SubMenu("TITLE=Tool",
                          iup.Menu(
                              iup.Item(
                                  iup.Attr("TITLE","RC lst"),
                                  func (arg *iup.ItemAction) {
                                      StartTool("tool/rclst/rclst.html")
                                  },
                              ),
                              iup.Item(
                                  iup.Attr("TITLE","Fig2 Keyword"),
                                  func (arg *iup.ItemAction) {
                                      StartTool("tool/fig2/fig2.html")
                                  },
                              ),
                          ),
                      ),
                       iup.SubMenu("TITLE=Help",
                           iup.Menu(
                               iup.Item(
                                   iup.Attr("TITLE","Release Note"),
                                   func (arg *iup.ItemAction) {
                                       ShowReleaseNote()
                                   },
                               ),
                               iup.Item(
                                   iup.Attr("TITLE","About"),
                                   func (arg *iup.ItemAction) {
                                       stw.ShowAbout()
                                   },
                               ),
                           ),
                       ),
    )
}

func (stw *Window) SectionDialog () {
    if stw.Frame == nil { return }
    labels := make([]*iup.Handle, len(stw.Frame.Sects))
    var skeys []int
    var snum int
    for k := range(stw.Frame.Sects) {
        if k > 900 { continue }
        skeys = append(skeys, k)
    }
    sort.Ints(skeys)
    sects := iup.Vbox()
    for _, k := range(skeys) {
        labels[snum] = stw.Show.sectionLabel(k)
        stw.Show.Label[fmt.Sprintf("%d",k)] = labels[snum]
        sects.Append(labels[snum])
        snum++
    }
    dlg := iup.Dialog(sects)
    dlg.SetAttribute("TITLE", "Section")
    dlg.SetAttribute("PARENTDIALOG", "mainwindow")
    dlg.Map()
    dlg.Show()
}

func (show *Show) sectionLabel (val int) *iup.Handle {
    var col string
    if b, ok := show.Sect[val]; ok {
        if b {
            col = labelFGColor
        } else {
            col = labelOFFColor
        }
    } else {
        show.Sect[val] = true
        col = labelFGColor
    }
    rtn := iup.Text(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
                    fmt.Sprintf("FGCOLOR=\"%s\"", col),
                    fmt.Sprintf("BGCOLOR=\"%s\"", labelBGColor),
                    fmt.Sprintf("VALUE=\"  %d\"", val),
                    "CANFOCUS=NO",
                    "READONLY=YES",
                    "BORDER=NO",
                    fmt.Sprintf("SIZE=%dx%d",datalabelwidth, dataheight),
                    )
    rtn.SetCallback(func (arg *iup.MouseButton) {
                        if show.Window.Frame != nil {
                            switch arg.Button {
                            case BUTTON_LEFT:
                                if arg.Pressed == 0 { // Released
                                    if show.Sect[val] {
                                        show.Sect[val] = false
                                        for _, el := range(show.Window.Frame.Elems) {
                                            if el.Sect.Num == val { el.Hide = true }
                                        }
                                        rtn.SetAttribute("FGCOLOR", labelOFFColor)
                                    } else {
                                        show.Sect[val] = true
                                        for _, el := range(show.Window.Frame.Elems) {
                                            if el.Sect.Num == val { el.Hide = false }
                                        }
                                        rtn.SetAttribute("FGCOLOR", labelFGColor)
                                    }
                                    // show.Window.UpdateShowRange()
                                    show.Window.HideNodes()
                                    show.Window.Redraw()
                                    // iup.SetFocus(show.Window.canv)
                                }
                            }
                        }
                    })
    return rtn
}

func datasectionlabel (val string) *iup.Handle {
    return iup.Text(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
                    fmt.Sprintf("FGCOLOR=\"%s\"", sectionLabelColor),
                    fmt.Sprintf("BGCOLOR=\"%s\"", labelBGColor),
                    fmt.Sprintf("VALUE=\"%s\"", val),
                    "CANFOCUS=NO",
                    "READONLY=YES",
                    "BORDER=NO",
                    fmt.Sprintf("SIZE=%dx%d",datalabelwidth+datatextwidth, dataheight),
                    )
}

func datalabel (val string) *iup.Handle {
    return iup.Text(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
                    fmt.Sprintf("FGCOLOR=\"%s\"", labelFGColor),
                    fmt.Sprintf("BGCOLOR=\"%s\"", labelBGColor),
                    fmt.Sprintf("VALUE=\"  %s\"", val),
                    "CANFOCUS=NO",
                    "READONLY=YES",
                    "BORDER=NO",
                    fmt.Sprintf("SIZE=%dx%d",datalabelwidth, dataheight),
                    )
}

func datatext (defval string) *iup.Handle {
    rtn := iup.Text(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
                    fmt.Sprintf("FGCOLOR=\"%s\"", labelFGColor),
                    fmt.Sprintf("BGCOLOR=\"%s\"", commandBGColor),
                    fmt.Sprintf("VALUE=\"%s\"", defval),
                    "BORDER=NO",
                    "ALIGNMENT=ARIGHT",
                    fmt.Sprintf("SIZE=%dx%d",datatextwidth, dataheight),)
    rtn.SetCallback(func (arg *iup.CommonGetFocus) {
                        rtn.SetAttribute("SELECTION", "1:100")
                    })
    return rtn
}

func (show *Show) etypeLabel (name string, width int, etype int, defval bool) *iup.Handle {
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
                    fmt.Sprintf("SIZE=%dx%d",width, dataheight),)
    rtn.SetCallback(func (arg *iup.MouseButton) {
                        if show.Window.Frame != nil {
                            switch arg.Button {
                            case BUTTON_LEFT:
                                if arg.Pressed == 0 { // Released
                                    if show.Etype[etype] {
                                        show.Etype[etype] = false
                                        for _, el := range(show.Window.Frame.Elems) {
                                            if el.Etype == etype { el.Hide = true }
                                        }
                                        rtn.SetAttribute("FGCOLOR", labelOFFColor)
                                    } else {
                                        show.Etype[etype] = true
                                        for _, el := range(show.Window.Frame.Elems) {
                                            if el.Etype == etype { el.Hide = false }
                                        }
                                        show.Window.UpdateShowRange() // TODO: ShowRange
                                        rtn.SetAttribute("FGCOLOR", labelFGColor)
                                    }
                                    // show.Window.UpdateShowRange() // TODO: ShowRange
                                    // show.Window.HideNodes()
                                    show.Window.Redraw()
                                    iup.SetFocus(show.Window.canv)
                                }
                            }
                        }
                    })
    return rtn
}

func (show *Show) switchLabel (etype int) *iup.Handle {
    var col string
    if show.Etype[etype] {
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
                    fmt.Sprintf("SIZE=%dx%d",10, dataheight),)
    rtn.SetCallback(func (arg *iup.MouseButton) {
                        if show.Window.Frame != nil {
                            switch arg.Button {
                            case BUTTON_LEFT:
                                if arg.Pressed == 0 { // Released
                                    if show.Etype[etype] {
                                        if !show.Etype[etype-2] {
                                            show.Etype[etype] = false
                                            show.Etype[etype-2] = true
                                            show.Label[ETYPES[etype]].SetAttribute("FGCOLOR", labelOFFColor)
                                            show.Label[ETYPES[etype-2]].SetAttribute("FGCOLOR", labelFGColor)
                                            for _, el := range(show.Window.Frame.Elems) {
                                                if el.Etype == etype { el.Hide = true }
                                                if el.Etype == etype-2 { el.Hide = false }
                                            }
                                        } else {
                                            show.Etype[etype-2] = false
                                            show.Label[ETYPES[etype-2]].SetAttribute("FGCOLOR", labelOFFColor)
                                            for _, el := range(show.Window.Frame.Elems) {
                                                if el.Etype == etype-2 { el.Hide = true }
                                            }
                                        }
                                    } else {
                                        if show.Etype[etype-2] {
                                            show.Etype[etype] = true
                                            show.Etype[etype-2] = false
                                            show.Label[ETYPES[etype]].SetAttribute("FGCOLOR", labelFGColor)
                                            show.Label[ETYPES[etype-2]].SetAttribute("FGCOLOR", labelOFFColor)
                                            for _, el := range(show.Window.Frame.Elems) {
                                                if el.Etype == etype { el.Hide = false }
                                                if el.Etype == etype-2 { el.Hide = true }
                                            }
                                        } else {
                                            show.Etype[etype] = true
                                            show.Label[ETYPES[etype]].SetAttribute("FGCOLOR", labelFGColor)
                                            for _, el := range(show.Window.Frame.Elems) {
                                                if el.Etype == etype { el.Hide = false }
                                            }
                                        }
                                    }
                                    show.Window.UpdateShowRange()
                                    show.Window.HideNodes()
                                    show.Window.Redraw()
                                    iup.SetFocus(show.Window.canv)
                                }
                            case BUTTON_CENTER:
                                if arg.Pressed == 0 {
                                    show.Etype[etype] = false
                                    show.Etype[etype-2] = false
                                    show.Label[ETYPES[etype]].SetAttribute("FGCOLOR", labelOFFColor)
                                    show.Label[ETYPES[etype-2]].SetAttribute("FGCOLOR", labelOFFColor)
                                    for _, el := range(show.Window.Frame.Elems) {
                                        if el.Etype == etype { el.Hide = true }
                                        if el.Etype == etype-2 { el.Hide = true }
                                    }
                                    show.Window.UpdateShowRange()
                                    show.Window.HideNodes()
                                    show.Window.Redraw()
                                    iup.SetFocus(show.Window.canv)
                                }
                            }
                        }
                    })
    return rtn
}

func (show *Show) stressLabel (etype int, index uint) *iup.Handle {
    rtn := iup.Text(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
                    fmt.Sprintf("FGCOLOR=\"%s\"", labelOFFColor),
                    fmt.Sprintf("BGCOLOR=\"%s\"", labelBGColor),
                    fmt.Sprintf("VALUE=\"%s\"", StressName[index]),
                    "CANFOCUS=NO",
                    "READONLY=YES",
                    "BORDER=NO",
                    fmt.Sprintf("SIZE=%dx%d",(datalabelwidth+datatextwidth)/6, dataheight),)
    rtn.SetCallback(func (arg *iup.MouseButton) {
                        if show.Window.Frame != nil {
                            switch arg.Button {
                            case BUTTON_LEFT:
                                if arg.Pressed == 0 { // Released
                                    if show.Stress[etype] & (1 << index) != 0 {
                                        show.Stress[etype] &= ^(1 << index)
                                        rtn.SetAttribute("FGCOLOR", labelOFFColor)
                                    } else {
                                        show.Stress[etype] |= (1 << index)
                                        rtn.SetAttribute("FGCOLOR", labelFGColor)
                                    }
                                    show.Window.Redraw()
                                    iup.SetFocus(show.Window.canv)
                                }
                            }
                        }
                    })
    return rtn
}

func (show *Show) displayLabel (name string, defval bool) *iup.Handle {
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
                    fmt.Sprintf("SIZE=%dx%d",datalabelwidth+datatextwidth, dataheight),)
    rtn.SetCallback(func (arg *iup.MouseButton) {
                        if show.Window.Frame != nil {
                            switch arg.Button {
                            case BUTTON_LEFT:
                                if arg.Pressed == 0 { // Released
                                    switch name {
                                    case "GAXIS":
                                        show.GlobalAxis = !show.GlobalAxis
                                    case "EAXIS":
                                        show.ElementAxis = !show.ElementAxis
                                    case "BOND":
                                        show.Bond = !show.Bond
                                    case "CONF":
                                        show.Conf = !show.Conf
                                    case "KIJUN":
                                        show.Kijun = !show.Kijun
                                    case "DEFORMATION":
                                        show.Deformation = !show.Deformation
                                    }
                                    if rtn.GetAttribute("FGCOLOR") == labelFGColor {
                                        rtn.SetAttribute("FGCOLOR", labelOFFColor)
                                    } else {
                                        rtn.SetAttribute("FGCOLOR", labelFGColor)
                                    }
                                    show.Window.Redraw()
                                    iup.SetFocus(show.Window.canv)
                                }
                            }
                        }
                    })
    return rtn
}

func (show *Show) captionLabel (ne string, name string, width int, val uint, on bool) *iup.Handle {
    var col string
    if on {
        col = labelFGColor
    } else {
        col = labelOFFColor
    }
    if on {
        switch ne {
        case "NODE":
            show.NodeCaption |= val
        case "ELEM":
            show.ElemCaption |= val
        }
    }
    rtn := iup.Text(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
                    fmt.Sprintf("FGCOLOR=\"%s\"", col),
                    fmt.Sprintf("BGCOLOR=\"%s\"", labelBGColor),
                    fmt.Sprintf("VALUE=\"%s\"", name),
                    "CANFOCUS=NO",
                    "READONLY=YES",
                    "BORDER=NO",
                    fmt.Sprintf("SIZE=%dx%d",width, dataheight),)
    rtn.SetCallback(func (arg *iup.MouseButton) {
                        if show.Window.Frame != nil {
                            switch arg.Button {
                            case BUTTON_LEFT:
                                if arg.Pressed == 0 { // Released
                                    switch ne {
                                    case "NODE":
                                        if show.NodeCaption & val != 0{
                                            rtn.SetAttribute("FGCOLOR", labelOFFColor)
                                            show.NodeCaption &= ^val
                                        } else {
                                            rtn.SetAttribute("FGCOLOR", labelFGColor)
                                            show.NodeCaption |= val
                                        }
                                    case "ELEM":
                                        if show.ElemCaption & val != 0 {
                                            rtn.SetAttribute("FGCOLOR", labelOFFColor)
                                            show.ElemCaption &= ^val
                                        } else {
                                            rtn.SetAttribute("FGCOLOR", labelFGColor)
                                            show.ElemCaption |= val
                                        }
                                    }
                                    show.Window.Redraw()
                                    iup.SetFocus(show.Window.canv)
                                }
                            }
                        }
                    })
    return rtn
}

func (show *Show) toggleLabel (def uint, values []string) *iup.Handle {
    col := labelFGColor
    l := len(values)
    rtn := iup.Text(fmt.Sprintf("FONT=\"%s, %s\"", commandFontFace, commandFontSize),
                    fmt.Sprintf("FGCOLOR=\"%s\"", col),
                    fmt.Sprintf("BGCOLOR=\"%s\"", labelBGColor),
                    fmt.Sprintf("VALUE=\"  %s\"", values[def]),
                    "CANFOCUS=NO",
                    "READONLY=YES",
                    "BORDER=NO",
                    fmt.Sprintf("SIZE=%dx%d",datalabelwidth+datatextwidth, dataheight),)
    rtn.SetCallback(func (arg *iup.MouseButton) {
                        if show.Window.Frame != nil {
                            switch arg.Button {
                            case BUTTON_LEFT:
                                if arg.Pressed == 0 { // Released
                                    now := 0
                                    tmp := rtn.GetAttribute("VALUE")[2:]
                                    for i:=0; i<l; i++ {
                                        if tmp == values[i] { now = i; break }
                                    }
                                    next := now + 1
                                    if next >= l { next = 0 }
                                    if next == ECOLOR_BLACK { next++ }
                                    rtn.SetAttribute("VALUE", fmt.Sprintf("  %s",values[next]))
                                    show.ColorMode = uint(next)
                                }
                            case BUTTON_CENTER:
                                if arg.Pressed == 0 { // Released
                                    now := 0
                                    tmp := rtn.GetAttribute("VALUE")[2:]
                                    for i:=0; i<l; i++ {
                                        if tmp == values[i] { now = i; break }
                                    }
                                    next := now - 1
                                    if next < 0 { next = l -1 }
                                    if next == ECOLOR_BLACK { next-- }
                                    rtn.SetAttribute("VALUE", fmt.Sprintf("  %s",values[next]))
                                    show.ColorMode = uint(next)
                                }
                            }
                            show.Window.Redraw()
                            iup.SetFocus(show.Window.canv)
                        }
                    })
    return rtn
}

func (stw *Window) CB_TextValue (h *iup.Handle, valptr *float64) {
    h.SetCallback( func (arg *iup.CommonKillFocus) {
                       if stw.Frame != nil {
                           val, err := strconv.ParseFloat(h.GetAttribute("VALUE"), 64)
                           if err == nil {
                               *valptr = val
                           }
                           stw.Redraw()
                       }
                   })
    h.SetCallback( func (arg *iup.CommonKeyAny) {
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

func (stw *Window) CB_RangeValue (h *iup.Handle, valptr *float64) {
    h.SetCallback( func (arg *iup.CommonKillFocus) {
                       if stw.Frame != nil {
                           val, err := strconv.ParseFloat(h.GetAttribute("VALUE"), 64)
                           if err == nil {
                               *valptr = val
                           }
                           stw.UpdateShowRange()
                           stw.Redraw()
                       }
                   })
    h.SetCallback( func (arg *iup.CommonKeyAny) {
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

func (stw *Window) UpdateShowRange () {
    for _, n := range stw.Frame.Nodes {
        // if n.Hide { continue }
        n.Hide = false
        if n.Coord[0] < stw.Show.Xrange[0] || stw.Show.Xrange[1] < n.Coord[0] {
            n.Hide = true
        } else if n.Coord[1] < stw.Show.Yrange[0] || stw.Show.Yrange[1] < n.Coord[1] {
            n.Hide = true
        } else if n.Coord[2] < stw.Show.Zrange[0] || stw.Show.Zrange[1] < n.Coord[2] {
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
        if math.Abs(d[0])<1e-4 {
            if k.Start[0] < stw.Show.Xrange[0] || stw.Show.Xrange[1] < k.Start[0] { k.Hide = true }
            if k.End[0] < stw.Show.Xrange[0] || stw.Show.Xrange[1] < k.End[0] { k.Hide = true }
        }
        if math.Abs(d[1])<1e-4 {
            if k.Start[1] < stw.Show.Yrange[0] || stw.Show.Yrange[1] < k.Start[1] { k.Hide = true }
            if k.End[1] < stw.Show.Yrange[0] || stw.Show.Yrange[1] < k.End[1] { k.Hide = true }
        }
        if math.Abs(d[2])<1e-4 {
            if k.Start[2] < stw.Show.Zrange[0] || stw.Show.Zrange[1] < k.Start[2] { k.Hide = true }
            if k.End[2] < stw.Show.Zrange[0] || stw.Show.Zrange[1] < k.End[2] { k.Hide = true }
        }
    }
}

func (stw *Window) CB_Period (h *iup.Handle, valptr *string) {
    h.SetCallback( func (arg *iup.CommonKillFocus) {
                       if stw.Frame != nil {
                           per := strings.ToUpper(h.GetAttribute("VALUE"))
                           *valptr = per
                           h.SetAttribute("VALUE", per)
                           stw.Redraw()
                       }
                   })
    h.SetCallback( func (arg *iup.CommonKeyAny) {
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
    stw.CB_TextValue(stw.Show.Label["GFACT"], &stw.Frame.View.Gfact)
    stw.CB_TextValue(stw.Show.Label["DISTR"], &stw.Frame.View.Dists[0])
    stw.CB_TextValue(stw.Show.Label["DISTL"], &stw.Frame.View.Dists[1])
    stw.CB_TextValue(stw.Show.Label["PHI"], &stw.Frame.View.Angle[0])
    stw.CB_TextValue(stw.Show.Label["THETA"], &stw.Frame.View.Angle[1])
    stw.CB_TextValue(stw.Show.Label["FOCUSX"], &stw.Frame.View.Focus[0])
    stw.CB_TextValue(stw.Show.Label["FOCUSY"], &stw.Frame.View.Focus[1])
    stw.CB_TextValue(stw.Show.Label["FOCUSZ"], &stw.Frame.View.Focus[2])
    stw.CB_TextValue(stw.Show.Label["CENTERX"], &stw.Frame.View.Center[0])
    stw.CB_TextValue(stw.Show.Label["CENTERY"], &stw.Frame.View.Center[1])
    stw.CB_RangeValue(stw.Show.Label["XMAX"], &stw.Show.Xrange[1])
    stw.CB_RangeValue(stw.Show.Label["XMIN"], &stw.Show.Xrange[0])
    stw.CB_RangeValue(stw.Show.Label["YMAX"], &stw.Show.Yrange[1])
    stw.CB_RangeValue(stw.Show.Label["YMIN"], &stw.Show.Yrange[0])
    stw.CB_RangeValue(stw.Show.Label["ZMAX"], &stw.Show.Zrange[1])
    stw.CB_RangeValue(stw.Show.Label["ZMIN"], &stw.Show.Zrange[0])
    stw.CB_Period(stw.Show.Label["PERIOD"], &stw.Show.Period)
    stw.CB_TextValue(stw.Show.Label["GAXISSIZE"], &stw.Show.GlobalAxisSize)
    stw.CB_TextValue(stw.Show.Label["EAXISSIZE"], &stw.Show.ElementAxisSize)
    stw.CB_TextValue(stw.Show.Label["BONDSIZE"], &stw.Show.BondSize)
    stw.CB_TextValue(stw.Show.Label["CONFSIZE"], &stw.Show.ConfSize)
    stw.CB_TextValue(stw.Show.Label["DFACT"], &stw.Show.Dfact)
    stw.CB_TextValue(stw.Show.Label["MFACT"], &stw.Show.Mfact)
}

func (stw *Window) EscapeCB () {
    stw.cline.SetAttribute("VALUE", "")
    stw.cname.SetAttribute("VALUE", "SELECT")
    stw.canv.SetAttribute("CURSOR", "ARROW")
    stw.CMenu()
    stw.CB_MouseButton()
    stw.CB_MouseMotion()
    stw.CB_CanvasWheel()
    stw.CB_CommonKeyAny()
    // stw.Show.SetColorMode(ECOLOR_SECT)
    stw.Redraw()
}

func (stw *Window) EscapeAll () {
    stw.Deselect()
    stw.EscapeCB()
}
// }}}


func EditPgp() {
    cmd := exec.Command("cmd", "/C", "start", pgpfile)
    cmd.Start()
}


func ReadPgp(filename string, aliases map[string]*Command) error {
    f, err := ioutil.ReadFile(filename)
    if err != nil {
        return err
    }
    var lis []string
    if ok := strings.HasSuffix(string(f),"\r\n"); ok {
        lis = strings.Split(string(f),"\r\n")
    } else {
        lis = strings.Split(string(f),"\n")
    }
    for _, j := range lis {
        if strings.HasPrefix(j, "#") { continue }
        var words []string
        for _, k := range strings.Split(j," ") {
            if k!="" {
                words=append(words,k)
            }
        }
        if len(words)==0 {
            continue
        }
        if value, ok := Commands[strings.ToUpper(words[1])]; ok {
            aliases[strings.ToUpper(words[0])] = value
        }
    }
    return nil
}

// Initialize
func init() {
    aliases = make(map[string]*Command,0)
    err := ReadPgp(pgpfile, aliases)
    if err != nil {
        aliases["D"] = DISTS
        aliases["O"] = OPEN
        aliases["SF"] = SETFOCUS
        aliases["NODE"] = SELECTNODE
        aliases["ELEM"] = SELECTELEM
        aliases["ER"] = ERRORELEM
        aliases["F"] = FENCE
        aliases["AD"] = ADDLINEELEM
    }
}


// Utility
func isShift(status string) bool { return status[0]=='S' }
func isCtrl(status string) bool { return status[1]=='C' }
func isLeft(status string) bool { return status[2]=='1' }
func isCenter(status string) bool { return status[3]=='2' }
func isRight(status string) bool { return status[4]=='3' }
func isDouble(status string) bool { return status[5]=='D' }
func isAlt(status string) bool { return status[6]=='A' }
func statusKey(status string) (rtn int) {
    if status[2]=='1' { rtn+=1 } // Left
    if status[3]=='2' { rtn+=2 } // Center
    if status[4]=='3' { rtn+=4 } // Right
    return
}

func min(x, y int) int {
    if x >= y { return y } else { return x}
}
func max(x, y int) int {
    if x >= y { return x } else { return y}
}
