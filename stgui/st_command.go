package stgui

// TODO: changeconf, copyconf

import (
    "bytes"
    "errors"
    "fmt"
    "strconv"
    "math"
    "strings"
    "github.com/visualfc/go-iup/iup"
    "github.com/visualfc/go-iup/cd"
    "github.com/yofu/st/stlib"
    "os"
    "path/filepath"
    "sort"
)

var (
    Commands = make(map[string]*Command,0)

    DISTS               = &Command{"DISTANCE", "DISTS", "measure distance", dists}
    TOGGLEBOND          = &Command{"TOGGLE", "TOGGLE BOND", "toggle bond of selected elem", togglebond}
    COPYBOND            = &Command{"COPY", "COPY BOND", "copy bond of selected elem", copybond}
    BONDPIN             = &Command{"PIN", "BOND PIN", "set bond of selected elem to pin-pin", bondpin}
    BONDRIGID           = &Command{"RIGID", "BOND RIGID", "set bond of selected elem to rigid-rigid", bondrigid}
    CONFFIX             = &Command{"FIX", "CONF FIX", "set conf of selected node to fix", conffix}
    CONFPIN             = &Command{"PIN", "CONF PIN", "set conf of selected node to pin", confpin}
    CONFXYROLLER        = &Command{"XY ROLLER", "CONF XYROLLER", "set conf of selected node to xy-roller", confxyroller}
    CONFFREE            = &Command{"FREE", "CONF FREE", "set conf of selected node to free", conffree}
    OPEN                = &Command{"OPEN", "OEPN INPUT", "open inp file", openinput}
    SAVE                = &Command{"SAVE", "SAVE", "save inp file", saveinput}
    WEIGHTCOPY          = &Command{"WGCP", "WEIGHTCOPY", "copy wgt file", weightcopy}
    READPGP             = &Command{"READ PGP", "READ PGP", "read pgp file", readpgp}
    INSERT              = &Command{"INST", "INSERT", "insert new frame", insert}
    SETFOCUS            = &Command{"STFO", "SET FOCUS", "set focus to the center", setfocus}
    SELECTNODE          = &Command{"_NOD", "SELECT NODE", "select node by number", selectnode}
    SELECTCOLUMNBASE    = &Command{"_CLB", "SELECT COLUMNBASE", "select column base nodes", selectcolumnbase}
    SELECTCONFED        = &Command{"_CON", "SELECT CONFED", "select confed nodes", selectconfed}
    SELECTELEM          = &Command{"_ELM", "SELECT ELEM", "select elem by number", selectelem}
    SELECTSECT          = &Command{"_SEC", "SELECT SECT", "select elem by section", selectsect}
    HIDESECTION         = &Command{"HDSC", "HIDE SECTION", "hide section", hidesection}
    HIDECURTAINWALL     = &Command{"HDCW", "HIDE CURTAIN WALL", "hide curtain wall", hidecurtainwall}
    SELECTCHILDREN      = &Command{"_CLD", "SELECT CHILDREN", "select elem.Children", selectchildren}
    ERRORELEM           = &Command{"ERRO", "ERROR ELEM", "select elem whose max(rate)>1.0", errorelem}
    FENCE               = &Command{"FNCE", "FENCE", "select elem by fence", fence}
    ADDLINEELEM         = &Command{"LINE", "ADD LINE ELEM", "add line elem", addlineelem}
    ADDPLATEELEM        = &Command{"PLATE(4pts)", "ADD PLATE ELEM", "add plate elem", addplateelem}
    ADDPLATEELEMBYLINE  = &Command{"PLATE(2lines)", "ADD PLATE ELEM BY LINE", "add plate elem by line", addplateelembyline}
    HATCHPLATEELEM      = &Command{"HATCHING", "HATCH PLATE ELEM", "add plate elem by hatching", hatchplateelem}
    ADDPLATEALL         = &Command{"PLATE(all)", "ADD PLATE ALL", "add all plate elem using selected nodes", addplateall}
    EDITPLATEELEM       = &Command{"EDPL", "EDIT PLATE ELEM", "edit plate elem", editplateelem}
    MATCHPROP           = &Command{"COPY PROPERTY", "MATCH PROPERTY", "match property", matchproperty}
    AXISTOCANG          = &Command{"CANG", "AXISTOCANG", "set cang by axis", axistocang}
    COPYELEM            = &Command{"COPY", "COPY ELEM", "copy selected elems", copyelem}
    MOVEELEM            = &Command{"MOVE", "MOVE ELEM", "move selected elems", moveelem}
    MOVENODE            = &Command{"MOVE", "MOVE NODE", "move selected nodes", movenode}
    MOVETOLINE          = &Command{"MOVETOLINE", "MOVE NODE ONTO LINE", "move selected nodes onto the line", movetoline}
    PINCHNODE           = &Command{"NDPC", "PINCH NODE", "pinch nodes", pinchnode}
    ROTATE              = &Command{"ROTE", "ROTATE", "rotate selected nodes", rotate}
    MIRROR              = &Command{"MIRR", "MIRROR", "mirror selected elems", mirror}
    SCALE               = &Command{"SCLE", "SCALE", "scale selected nodes", scale}
    SEARCHELEM          = &Command{"ELSR", "SEARCH ELEM", "search elems using node", searchelem}
    NODETOELEM          = &Command{"N->E", "NODE TO ELEM", "select elems using selected node", nodetoelemall}
    ELEMTONODE          = &Command{"E->N", "ELEM TO NODE", "select nodes used by selected elem", elemtonode}
    CONNECTED           = &Command{"N->N", "CONNECTED", "select nodes connected to selected node", connected}
    ONNODE              = &Command{"ONND", "ON NODE", "select nodes which is on selected elems", onnode}
    NODENOREFERENCE     = &Command{"NODE NO REF.", "NODE NO REFERENCE", "delete nodes which are not refered by any elem", nodenoreference}
    ELEMSAMENODE        = &Command{"ELEM SAME NODE", "ELEM SAME NODE", "delete elems which has duplicated enod", elemsamenode}
    PRUNEENOD           = &Command{"PRUNE ENOD", "PRUNE ENOD", "prune duplicated enod", pruneenod}
    NODEDUPLICATION     = &Command{"DUPLICATIVE NODE", "NODE DUPLICATION", "delete duplicated nodes", nodeduplication}
    ELEMDUPLICATION     = &Command{"DUPLICATIVE ELEM", "ELEM DUPLICATION", "delete duplicated elems", elemduplication}
    NODESORT            = &Command{"NODE SORT", "NODE SORT", "node sort", nodesort}
    CATBYNODE           = &Command{"CATN", "CAT BY NODE", "join 2 elems using selected node", catbynode}
    CATINTERMEDIATENODE = &Command{"CATI", "CAT INTERMEDIATE NODE", "concatenate at intermediate node", catintermediatenode}
    JOINLINEELEM        = &Command{"JOIN LINE", "JOIN LINE ELEM", "join selected 2 elems", joinlineelem}
    JOINPLATEELEM       = &Command{"JOIN PLATE", "JOIN PLATE ELEM", "join selected 2 elems", joinplateelem}
    EXTRACTARCLM        = &Command{"EXAR", "EXTRACT ARCLM", "extract arclm", extractarclm}
    ARCLM001            = &Command{"A001", "ARCLM001", "arclm001", arclm001}
    DIVIDEATONS         = &Command{"ON NODES", "DIVIDE AT ONS", "divide selected elems at onnode", divideatons}
    DIVIDEATMID         = &Command{"MID POINT", "DIVIDE AT MID", "divide selected elems at midpoint", divideatmid}
    INTERSECT           = &Command{"INTS", "INTERSECT", "divide selected elems at intersection", intersect}
    INTERSECTALL        = &Command{"INTA", "INTERSECT ALL", "divide selected elems at all intersection", intersectall}
    TRIM                = &Command{"TRIM", "TRIM", "trim elements with selected elem", trim}
    EXTEND              = &Command{"EXTEND", "EXTEND", "extend elements to selected elem", extend}
    MERGENODE           = &Command{"MERGE", "MERGE NODE", "merge nodes", mergenode}
    ERASE               = &Command{"ERASE", "ERASE", "erase selected elems", erase}
    FACTS               = &Command{"FACT", "FACTS", "calculate eccentricity ratio and modulus of rigidity", facts}
    REACTION            = &Command{"RCTN", "REACTION", "output reaction", reaction}
    SUMREACTION         = &Command{"SRTN", "SUMREACTION", "show sum of reaction", sumreaction}
    UPLIFT              = &Command{"LIFT", "UPLIFT", "select uplifting nodes", uplift}
    NOTICE1459          = &Command{"1459", "NOTICE1459", "shishou", notice1459}
    ZOUBUNDISP          = &Command{"ZBDP", "ZOUBUNDISP", "output displacement", zoubundisp}
    ZOUBUNYIELD         = &Command{"ZBYD", "ZOUBUNYIELD", "output Fmax & Fmin", zoubunyield}
    ZOUBUNREACTION      = &Command{"ZBRC", "ZOUBUNREACTION", "output reaction", zoubunreaction}
    AMOUNTPROP          = &Command{"AMTP", "AMOUNTPROP", "total amount of PROP", amountprop}
)

func init() {
    Commands["DISTS"]=DISTS
    Commands["TOGGLEBOND"]=TOGGLEBOND
    Commands["COPYBOND"]=COPYBOND
    Commands["BONDPIN"]=BONDPIN
    Commands["BONDRIGID"]=BONDRIGID
    Commands["CONFFIX"]=CONFFIX
    Commands["CONFPIN"]=CONFPIN
    Commands["CONFXYROLLER"]=CONFXYROLLER
    Commands["CONFFREE"]=CONFFREE
    Commands["OPEN"]=OPEN
    Commands["SAVE"]=SAVE
    Commands["WEIGHTCOPY"]=WEIGHTCOPY
    Commands["READPGP"]=READPGP
    Commands["INSERT"]=INSERT
    Commands["SETFOCUS"]=SETFOCUS
    Commands["SELECTNODE"]=SELECTNODE
    Commands["SELECTCOLUMNBASE"]=SELECTCOLUMNBASE
    Commands["SELECTCONFED"]=SELECTCONFED
    Commands["SELECTELEM"]=SELECTELEM
    Commands["SELECTSECT"]=SELECTSECT
    Commands["HIDESECTION"]=HIDESECTION
    Commands["HIDECURTAINWALL"]=HIDECURTAINWALL
    Commands["SELECTCHILDREN"]=SELECTCHILDREN
    Commands["ERRORELEM"]=ERRORELEM
    Commands["FENCE"]=FENCE
    Commands["ADDLINEELEM"]=ADDLINEELEM
    Commands["ADDPLATEELEM"]=ADDPLATEELEM
    Commands["ADDPLATEELEMBYLINE"]=ADDPLATEELEMBYLINE
    Commands["HATCHPLATEELEM"]=HATCHPLATEELEM
    Commands["ADDPLATEALL"]=ADDPLATEALL
    Commands["EDITPLATEELEM"]=EDITPLATEELEM
    Commands["MATCHPROP"]=MATCHPROP
    Commands["AXISTOCANG"]=AXISTOCANG
    Commands["COPYELEM"]=COPYELEM
    Commands["MOVEELEM"]=MOVEELEM
    Commands["MOVENODE"]=MOVENODE
    Commands["MOVETOLINE"]=MOVETOLINE
    Commands["PINCHNODE"]=PINCHNODE
    Commands["ROTATE"]=ROTATE
    Commands["MIRROR"]=MIRROR
    Commands["SCALE"]=SCALE
    Commands["SEARCHELEM"]=SEARCHELEM
    Commands["NODETOELEM"]=NODETOELEM
    Commands["ELEMTONODE"]=ELEMTONODE
    Commands["CONNECTED"]=CONNECTED
    Commands["ONNODE"]=ONNODE
    Commands["NODENOREFERENCE"]=NODENOREFERENCE
    Commands["ELEMSAMENODE"]=ELEMSAMENODE
    Commands["PRUNEENOD"]=PRUNEENOD
    Commands["NODEDUPLICATION"]=NODEDUPLICATION
    Commands["ELEMDUPLICATION"]=ELEMDUPLICATION
    Commands["NODESORT"]=NODESORT
    Commands["CATBYNODE"]=CATBYNODE
    Commands["CATINTERMEDIATENODE"]=CATINTERMEDIATENODE
    Commands["JOINLINEELEM"]=JOINLINEELEM
    Commands["JOINPLATEELEM"]=JOINPLATEELEM
    Commands["EXTRACTARCLM"]=EXTRACTARCLM
    Commands["ARCLM001"]=ARCLM001
    Commands["DIVIDEATONS"]=DIVIDEATONS
    Commands["DIVIDEATMID"]=DIVIDEATMID
    Commands["INTERSECT"]=INTERSECT
    Commands["INTERSECTALL"]=INTERSECTALL
    Commands["TRIM"]=TRIM
    Commands["EXTEND"]=EXTEND
    Commands["MERGENODE"]=MERGENODE
    Commands["ERASE"]=ERASE
    Commands["FACTS"]=FACTS
    Commands["REACTION"]=REACTION
    Commands["SUMREACTION"]=SUMREACTION
    Commands["UPLIFT"]=UPLIFT
    Commands["NOTICE1459"]=NOTICE1459
    Commands["ZOUBUNDISP"]=ZOUBUNDISP
    Commands["ZOUBUNYIELD"]=ZOUBUNYIELD
    Commands["ZOUBUNREACTION"]=ZOUBUNREACTION
    Commands["AMOUNTPROP"]=AMOUNTPROP
}

type Command struct {
    Display string
    Name    string
    command string
    call    func(*Window)
}

func (cmd *Command) Exec(stw *Window) {
    cmd.call(stw)
}


// GETCOORD // {{{
func getcoord (stw *Window, f func(x, y, z float64)) {
    stw.canv.SetAttribute("CURSOR", "CROSS")
    var snap *st.Node // for Snapping
    stw.cdcanv.Foreground(cd.CD_YELLOW)
    stw.cdcanv.WriteMode(cd.CD_XOR)
    stw.SelectNode = make([]*st.Node, 1)
    setnnum := func () {
        nnum, err := strconv.ParseInt(stw.cline.GetAttribute("VALUE"),10,64)
        if err == nil {
            if n, ok := stw.Frame.Nodes[int(nnum)]; ok {
                f(n.Coord[0], n.Coord[1], n.Coord[2])
            }
        }
        stw.cline.SetAttribute("VALUE", "")
    }
    stw.canv.SetCallback( func (arg *iup.MouseButton) {
                              if stw.Frame != nil {
                                  stw.dbuff.UpdateYAxis(&arg.Y)
                                  switch arg.Button {
                                  case BUTTON_LEFT:
                                      if arg.Pressed == 0 { // Released
                                          if n := stw.PickNode(int(arg.X), int(arg.Y)); n != nil {
                                              f(n.Coord[0], n.Coord[1], n.Coord[2])
                                          }
                                          // stw.cdcanv.Line(int(stw.SelectNode[0].Pcoord[0]), int(stw.SelectNode[0].Pcoord[1]), stw.endX, stw.endY) // TODO
                                          stw.Redraw()
                                      }
                                  case BUTTON_CENTER:
                                      if arg.Pressed == 0 { // Released
                                          // if stw.SelectNode[0] != nil { // TODO
                                          //     stw.cdcanv.Line(int(stw.SelectNode[0].Pcoord[0]), int(stw.SelectNode[0].Pcoord[1]), stw.endX, stw.endY)
                                          // }
                                          stw.Redraw()
                                      } else { // Pressed
                                          if isDouble(arg.Status) {
                                              stw.Frame.SetFocus()
                                              stw.DrawFrameNode()
                                              stw.ShowCenter()
                                          } else {
                                              stw.startX = int(arg.X); stw.startY = int(arg.Y)
                                          }
                                      }
                                  case BUTTON_RIGHT:
                                      if arg.Pressed == 0 {
                                          if v := stw.cline.GetAttribute("VALUE"); v!="" {
                                              setnnum()
                                          } else {
                                              // fmt.Println("CONTEXT MENU")
                                              stw.EscapeAll()
                                          }
                                      }
                                  }
                              }
                          })
    stw.canv.SetCallback( func (arg *iup.CommonKeyAny) {
                              key := iup.KeyState(arg.Key)
                              switch key.Key() {
                              default:
                                  stw.DefaultKeyAny(key)
                              case KEY_BS:
                                  val := stw.cline.GetAttribute("VALUE")
                                  if val != "" {
                                      stw.cline.SetAttribute("VALUE", val[:len(val)-1])
                                  }
                              case KEY_ENTER:
                                  setnnum()
                              case KEY_ESCAPE:
                                  stw.EscapeAll()
                              case 'D','d':
                                  x, y, z, err := stw.QueryCoord("GET COORD")
                                  if err == nil {
                                      f(x, y, z)
                                  }
                              }
                          })
    stw.canv.SetCallback( func (arg *iup.MouseMotion) {
                              if stw.Frame != nil {
                                  stw.dbuff.UpdateYAxis(&arg.Y)
                                  // Snapping
                                  stw.cdcanv.Foreground(cd.CD_YELLOW)
                                  if snap != nil {
                                      stw.cdcanv.FCircle(snap.Pcoord[0], snap.Pcoord[1], nodeSelectPixel)
                                  }
                                  n := stw.PickNode(int(arg.X), int(arg.Y))
                                  if n != nil {
                                      stw.cdcanv.FCircle(n.Pcoord[0], n.Pcoord[1],  nodeSelectPixel)
                                  }
                                  snap = n
                                  ///
                                  switch statusKey(arg.Status) {
                                  case STATUS_CENTER:
                                      if isShift(arg.Status) {
                                          stw.Frame.View.Center[0] += float64(int(arg.X) - stw.startX) * CanvasMoveSpeedX
                                          stw.Frame.View.Center[1] += float64(int(arg.Y) - stw.startY) * CanvasMoveSpeedY
                                      } else {
                                          stw.Frame.View.Angle[0] -= float64(int(arg.Y) - stw.startY) * CanvasRotateSpeedY
                                          stw.Frame.View.Angle[1] -= float64(int(arg.X) - stw.startX) * CanvasRotateSpeedX
                                      }
                                      stw.DrawFrameNode()
                                  }
                              }
                          })
}
// }}}

// GET1NODE // {{{
func get1node (stw *Window, f func(n *st.Node)) {
    stw.canv.SetAttribute("CURSOR", "CROSS")
    var snap *st.Node // for Snapping
    stw.cdcanv.Foreground(cd.CD_YELLOW)
    stw.cdcanv.WriteMode(cd.CD_XOR)
    stw.SelectNode = make([]*st.Node, 1)
    setnnum := func () {
        nnum, err := strconv.ParseInt(stw.cline.GetAttribute("VALUE"),10,64)
        if err == nil {
            if n, ok := stw.Frame.Nodes[int(nnum)]; ok {
                f(n)
            }
        }
        stw.cline.SetAttribute("VALUE", "")
    }
    stw.canv.SetCallback( func (arg *iup.MouseButton) {
                              if stw.Frame != nil {
                                  stw.dbuff.UpdateYAxis(&arg.Y)
                                  switch arg.Button {
                                  case BUTTON_LEFT:
                                      if arg.Pressed == 0 { // Released
                                          if n := stw.PickNode(int(arg.X), int(arg.Y)); n != nil {
                                              f(n)
                                          }
                                          // stw.cdcanv.Line(int(stw.SelectNode[0].Pcoord[0]), int(stw.SelectNode[0].Pcoord[1]), stw.endX, stw.endY) // TODO
                                          stw.Redraw()
                                      }
                                  case BUTTON_CENTER:
                                      if arg.Pressed == 0 { // Released
                                          // if stw.SelectNode[0] != nil { // TODO
                                          //     stw.cdcanv.Line(int(stw.SelectNode[0].Pcoord[0]), int(stw.SelectNode[0].Pcoord[1]), stw.endX, stw.endY)
                                          // }
                                          stw.Redraw()
                                      } else { // Pressed
                                          if isDouble(arg.Status) {
                                              stw.Frame.SetFocus()
                                              stw.DrawFrameNode()
                                              stw.ShowCenter()
                                          } else {
                                              stw.startX = int(arg.X); stw.startY = int(arg.Y)
                                          }
                                      }
                                  case BUTTON_RIGHT:
                                      if arg.Pressed == 0 {
                                          if v := stw.cline.GetAttribute("VALUE"); v!="" {
                                              setnnum()
                                          } else {
                                              // fmt.Println("CONTEXT MENU")
                                              stw.EscapeAll()
                                          }
                                      }
                                  }
                              }
                          })
    stw.canv.SetCallback( func (arg *iup.CommonKeyAny) {
                              key := iup.KeyState(arg.Key)
                              switch key.Key() {
                              default:
                                  stw.DefaultKeyAny(key)
                              case KEY_BS:
                                  val := stw.cline.GetAttribute("VALUE")
                                  if val != "" {
                                      stw.cline.SetAttribute("VALUE", val[:len(val)-1])
                                  }
                              case KEY_ENTER:
                                  setnnum()
                              case KEY_ESCAPE:
                                  stw.EscapeAll()
                              case 'D','d':
                                  x, y, z, err := stw.QueryCoord("GET COORD")
                                  if err == nil {
                                      n := stw.Frame.CoordNode(x, y, z)
                                      f(n)
                                  }
                              }
                          })
    stw.canv.SetCallback( func (arg *iup.MouseMotion) {
                              if stw.Frame != nil {
                                  stw.dbuff.UpdateYAxis(&arg.Y)
                                  // Snapping
                                  stw.cdcanv.Foreground(cd.CD_YELLOW)
                                  if snap != nil {
                                      stw.cdcanv.FCircle(snap.Pcoord[0], snap.Pcoord[1], nodeSelectPixel)
                                  }
                                  n := stw.PickNode(int(arg.X), int(arg.Y))
                                  if n != nil {
                                      stw.cdcanv.FCircle(n.Pcoord[0], n.Pcoord[1],  nodeSelectPixel)
                                  }
                                  snap = n
                                  ///
                                  switch statusKey(arg.Status) {
                                  case STATUS_CENTER:
                                      if isShift(arg.Status) {
                                          stw.Frame.View.Center[0] += float64(int(arg.X) - stw.startX) * CanvasMoveSpeedX
                                          stw.Frame.View.Center[1] += float64(int(arg.Y) - stw.startY) * CanvasMoveSpeedY
                                      } else {
                                          stw.Frame.View.Angle[0] -= float64(int(arg.Y) - stw.startY) * CanvasRotateSpeedY
                                          stw.Frame.View.Angle[1] -= float64(int(arg.X) - stw.startX) * CanvasRotateSpeedX
                                      }
                                      stw.DrawFrameNode()
                                  }
                              }
                          })
}
// }}}


// TOGGLEBOND// {{{
func togglebond (stw *Window) {
    get1node(stw, func (n *st.Node) {
                      if stw.SelectElem == nil { return }
                      for _, el := range stw.SelectElem {
                          if el == nil { continue }
                          el.ToggleBond(n.Num)
                          // if err == nil { break }
                      }
                  })
}
// }}}


// GET2NODES // {{{
// DISTS: TODO: When button is released, tail line remains. When 2nd node is selected in command line, tail line remains.
func get2nodes (stw *Window, f func(n *st.Node), fdel func()) {
    var snap *st.Node // for Snapping
    stw.canv.SetAttribute("CURSOR", "CROSS")
    stw.cdcanv.Foreground(cd.CD_YELLOW)
    stw.cdcanv.WriteMode(cd.CD_XOR)
    stw.SelectNode = make([]*st.Node, 2)
    stw.addHistory("始端を指定[ダイアログ(D)]")
    setnnum := func () {
        nnum, err := strconv.ParseInt(stw.cline.GetAttribute("VALUE"),10,64)
        if err == nil {
            if n, ok := stw.Frame.Nodes[int(nnum)]; ok {
                if stw.SelectNode[0] != nil {
                    f(n)
                } else {
                    stw.SelectNode[0] = n
                    stw.cdcanv.Foreground(cd.CD_DARK_RED)
                    stw.cdcanv.WriteMode(cd.CD_XOR)
                    first = 1
                    stw.addHistory("終端を指定[ダイアログ(D)]")
                }
            }
        }
        stw.cline.SetAttribute("VALUE", "")
    }
    stw.canv.SetCallback( func (arg *iup.MouseButton) {
                              if stw.Frame != nil {
                                  stw.dbuff.UpdateYAxis(&arg.Y)
                                  switch arg.Button {
                                  case BUTTON_LEFT:
                                      if arg.Pressed == 0 { // Released
                                          if stw.SelectNode[0] != nil {
                                              if n := stw.PickNode(int(arg.X), int(arg.Y)); n != nil {
                                                  f(n)
                                              }
                                              // stw.cdcanv.Line(int(stw.SelectNode[0].Pcoord[0]), int(stw.SelectNode[0].Pcoord[1]), stw.endX, stw.endY) // TODO
                                          } else {
                                              if n := stw.PickNode(int(arg.X), int(arg.Y)); n != nil {
                                                  stw.SelectNode[0] = n
                                                  stw.cdcanv.Foreground(cd.CD_DARK_RED)
                                                  stw.cdcanv.WriteMode(cd.CD_XOR)
                                                  first = 1
                                                  stw.addHistory("終端を指定[ダイアログ(D)]")
                                              }
                                          }
                                          stw.Redraw()
                                      }
                                  case BUTTON_CENTER:
                                      if arg.Pressed == 0 { // Released
                                          // if stw.SelectNode[0] != nil { // TODO
                                          //     stw.cdcanv.Line(int(stw.SelectNode[0].Pcoord[0]), int(stw.SelectNode[0].Pcoord[1]), stw.endX, stw.endY)
                                          // }
                                          stw.Redraw()
                                      } else { // Pressed
                                          if isDouble(arg.Status) {
                                              stw.Frame.SetFocus()
                                              stw.DrawFrameNode()
                                              stw.ShowCenter()
                                          } else {
                                              stw.startX = int(arg.X); stw.startY = int(arg.Y)
                                          }
                                      }
                                  case BUTTON_RIGHT:
                                      if arg.Pressed == 0 {
                                          if v := stw.cline.GetAttribute("VALUE"); v!="" {
                                              setnnum()
                                          } else {
                                              // fmt.Println("CONTEXT MENU")
                                              stw.EscapeAll()
                                          }
                                      }
                                  }
                              }
                          })
    stw.canv.SetCallback( func (arg *iup.MouseMotion) {
                              if stw.Frame != nil {
                                  stw.dbuff.UpdateYAxis(&arg.Y)
                                  // Snapping
                                  stw.cdcanv.Foreground(cd.CD_YELLOW)
                                  if snap != nil {
                                      stw.cdcanv.FCircle(snap.Pcoord[0], snap.Pcoord[1], nodeSelectPixel)
                                  }
                                  n := stw.PickNode(int(arg.X), int(arg.Y))
                                  if n != nil {
                                      stw.cdcanv.FCircle(n.Pcoord[0], n.Pcoord[1],  nodeSelectPixel)
                                  }
                                  snap = n
                                  ///
                                  switch statusKey(arg.Status) {
                                  case STATUS_CENTER:
                                      if isShift(arg.Status) {
                                          stw.Frame.View.Center[0] += float64(int(arg.X) - stw.startX) * CanvasMoveSpeedX
                                          stw.Frame.View.Center[1] += float64(int(arg.Y) - stw.startY) * CanvasMoveSpeedY
                                      } else {
                                          stw.Frame.View.Angle[0] -= float64(int(arg.Y) - stw.startY) * CanvasRotateSpeedY
                                          stw.Frame.View.Angle[1] -= float64(int(arg.X) - stw.startX) * CanvasRotateSpeedX
                                      }
                                      stw.DrawFrameNode()
                                  }
                                  if stw.SelectNode[0] != nil {
                                      stw.cdcanv.Foreground(cd.CD_DARK_RED)
                                      stw.TailLine(int(stw.SelectNode[0].Pcoord[0]), int(stw.SelectNode[0].Pcoord[1]), arg)
                                  }
                              }
                          })
    stw.canv.SetCallback( func (arg *iup.CommonKeyAny) {
                              key := iup.KeyState(arg.Key)
                              switch key.Key() {
                              default:
                                  stw.DefaultKeyAny(key)
                              case KEY_BS:
                                  val := stw.cline.GetAttribute("VALUE")
                                  if val != "" {
                                      stw.cline.SetAttribute("VALUE", val[:len(val)-1])
                                  }
                              case KEY_DELETE:
                                  fdel()
                              case KEY_ENTER:
                                  setnnum()
                              case KEY_ESCAPE:
                                  stw.EscapeAll()
                              case 'D','d':
                                  x, y, z, err := stw.QueryCoord("GET COORD")
                                  if err == nil {
                                      n := stw.Frame.CoordNode(x, y, z)
                                      stw.Redraw()
                                      if stw.SelectNode[0] != nil {
                                          f(n)
                                      } else {
                                          stw.SelectNode[0] = n
                                          stw.cdcanv.Foreground(cd.CD_DARK_RED)
                                          stw.cdcanv.WriteMode(cd.CD_XOR)
                                          first = 1
                                          stw.addHistory("終端を指定[ダイアログ(D)]")
                                      }
                                  }
                              }
                          })
}
// }}}


// DISTS// {{{
func dists (stw *Window) {
    get2nodes(stw, func (n *st.Node) {
                       stw.SelectNode[1] = n
                       dx, dy, dz, d := stw.Frame.Distance(stw.SelectNode[0], n)
                       stw.addHistory(fmt.Sprintf("NODE: %d - %d", stw.SelectNode[0].Num, n.Num))
                       stw.addHistory(fmt.Sprintf("DX: %.3f DY: %.3f DZ: %.3f D: %.3f", dx, dy, dz, d))
                       // stw.cdcanv.Foreground(cd.CD_WHITE)
                       // stw.cdcanv.WriteMode(cd.CD_REPLACE)
                       stw.EscapeAll()
                   },
                   func () {
                       stw.SelectNode = make([]*st.Node, 2)
                       stw.Redraw()
                   })
}
// }}}


// ADDLINEELEM// {{{
func addlineelem (stw *Window) {
    get2nodes(stw, func (n *st.Node) {
                       stw.SelectNode[1] = n
                       sec := stw.Frame.DefaultSect()
                       el := stw.Frame.AddLineElem(-1, stw.SelectNode, sec, st.NONE)
                       stw.addHistory(fmt.Sprintf("ELEM: %d (ENOD: %d - %d, SECT: %d)", el.Num, stw.SelectNode[0].Num, n.Num, sec.Num))
                       // stw.cdcanv.Foreground(cd.CD_WHITE)
                       // stw.cdcanv.WriteMode(cd.CD_REPLACE)
                       stw.EscapeAll()
                   },
                   func () {
                       stw.SelectNode = make([]*st.Node, 2)
                       stw.Redraw()
                   })
}
// }}}


// GETNNODES // {{{
// DISTS: TODO: When button is released, tail line remains. When 2nd node is selected in command line, tail line remains.
func getnnodes (stw *Window, maxnum int, f func(int)) {
    stw.canv.SetAttribute("CURSOR", "CROSS")
    var snap *st.Node // for Snapping
    stw.cdcanv.Foreground(cd.CD_YELLOW)
    stw.cdcanv.WriteMode(cd.CD_XOR)
    stw.SelectNode = make([]*st.Node, maxnum)
    selected := 0
    setnnum := func () {
        if selected >= maxnum { stw.addHistory("TOO MANY NODES SELECTED"); return }
        nnum, err := strconv.ParseInt(stw.cline.GetAttribute("VALUE"),10,64)
        if err == nil {
            if n, ok := stw.Frame.Nodes[int(nnum)]; ok {
                if stw.SelectNode[0] != nil {
                    stw.SelectNode[selected] = n
                    selected++
                } else {
                    stw.SelectNode[0] = n
                    selected++
                    stw.cdcanv.Foreground(cd.CD_DARK_RED)
                    stw.cdcanv.WriteMode(cd.CD_XOR)
                    first = 1
                }
            }
        }
        stw.cline.SetAttribute("VALUE", "")
    }
    stw.canv.SetCallback( func (arg *iup.MouseButton) {
                              if stw.Frame != nil {
                                  stw.dbuff.UpdateYAxis(&arg.Y)
                                  switch arg.Button {
                                  case BUTTON_LEFT:
                                      if arg.Pressed == 0 { // Released
                                          if n := stw.PickNode(int(arg.X), int(arg.Y)); n != nil {
                                              if selected >= maxnum {
                                                  stw.addHistory("TOO MANY NODES SELECTED")
                                              } else if stw.SelectNode[0] != nil {
                                                  stw.SelectNode[selected] = n
                                                  selected++
                                                  // stw.cdcanv.Line(int(stw.SelectNode[0].Pcoord[0]), int(stw.SelectNode[0].Pcoord[1]), stw.endX, stw.endY) // TODO
                                              } else {
                                                  stw.SelectNode[0] = n
                                                  selected++
                                                  stw.cdcanv.Foreground(cd.CD_DARK_RED)
                                                  stw.cdcanv.WriteMode(cd.CD_XOR)
                                                  first = 1
                                              }
                                          }
                                          stw.Redraw()
                                      }
                                  case BUTTON_CENTER:
                                      if arg.Pressed == 0 { // Released
                                          // if stw.SelectNode[0] != nil { // TODO
                                          //     stw.cdcanv.Line(int(stw.SelectNode[0].Pcoord[0]), int(stw.SelectNode[0].Pcoord[1]), stw.endX, stw.endY)
                                          // }
                                          stw.Redraw()
                                      } else { // Pressed
                                          if isDouble(arg.Status) {
                                              stw.Frame.SetFocus()
                                              stw.DrawFrameNode()
                                              stw.ShowCenter()
                                          } else {
                                              stw.startX = int(arg.X); stw.startY = int(arg.Y)
                                          }
                                      }
                                  case BUTTON_RIGHT:
                                      if arg.Pressed == 0 {
                                          if v := stw.cline.GetAttribute("VALUE"); v!="" {
                                              setnnum()
                                          } else {
                                              // fmt.Println("CONTEXT MENU")
                                              f(selected)
                                          }
                                      }
                                  }
                              }
                          })
    stw.canv.SetCallback( func (arg *iup.MouseMotion) {
                              if stw.Frame != nil {
                                  stw.dbuff.UpdateYAxis(&arg.Y)
                                  // Snapping
                                  stw.cdcanv.Foreground(cd.CD_YELLOW)
                                  if snap != nil {
                                      stw.cdcanv.FCircle(snap.Pcoord[0], snap.Pcoord[1], nodeSelectPixel)
                                  }
                                  n := stw.PickNode(int(arg.X), int(arg.Y))
                                  if n != nil {
                                      stw.cdcanv.FCircle(n.Pcoord[0], n.Pcoord[1],  nodeSelectPixel)
                                  }
                                  snap = n
                                  ///
                                  switch statusKey(arg.Status) {
                                  case STATUS_CENTER:
                                      if isShift(arg.Status) {
                                          stw.Frame.View.Center[0] += float64(int(arg.X) - stw.startX) * CanvasMoveSpeedX
                                          stw.Frame.View.Center[1] += float64(int(arg.Y) - stw.startY) * CanvasMoveSpeedY
                                      } else {
                                          stw.Frame.View.Angle[0] -= float64(int(arg.Y) - stw.startY) * CanvasRotateSpeedY
                                          stw.Frame.View.Angle[1] -= float64(int(arg.X) - stw.startX) * CanvasRotateSpeedX
                                      }
                                      stw.DrawFrameNode()
                                  }
                                  if stw.SelectNode[0] != nil {
                                      if selected < 2 {
                                          stw.TailLine(int(stw.SelectNode[0].Pcoord[0]), int(stw.SelectNode[0].Pcoord[1]), arg)
                                      } else {
                                          stw.TailPolygon(stw.SelectNode, arg)
                                      }
                                  }
                              }
                          })
    stw.canv.SetCallback( func (arg *iup.CommonKeyAny) {
                              key := iup.KeyState(arg.Key)
                              switch key.Key() {
                              default:
                                  stw.DefaultKeyAny(key)
                              case KEY_BS:
                                  val := stw.cline.GetAttribute("VALUE")
                                  if val != "" {
                                      stw.cline.SetAttribute("VALUE", val[:len(val)-1])
                                  }
                              case KEY_DELETE:
                                  stw.SelectNode = make([]*st.Node, maxnum)
                                  selected++
                              case KEY_ENTER:
                                  if v := stw.cline.GetAttribute("VALUE"); v!="" {
                                      setnnum()
                                  } else {
                                      f(selected)
                                  }
                              case KEY_ESCAPE:
                                  stw.EscapeAll()
                              }
                          })
}
// }}}


// ADDPLATEELEM// {{{
func addplateelem (stw *Window) {
    maxnum := 4
    getnnodes(stw, maxnum, func (num int) {
                               if num >=3 {
                                   en := stw.SelectNode[:num]
                                   sec := stw.Frame.DefaultSect()
                                   el := stw.Frame.AddPlateElem(-1, en, sec, st.NONE)
                                   var buf bytes.Buffer
                                   buf.WriteString(fmt.Sprintf("ELEM: %d (ENOD: ", el.Num))
                                   for _, n := range en {
                                       buf.WriteString(fmt.Sprintf("%d ", n.Num))
                                   }
                                   buf.WriteString(fmt.Sprintf(", SECT: %d)", sec.Num))
                                   stw.addHistory(buf.String())
                               }
                               stw.EscapeAll()
                           })
}

func addplateelembyline (stw *Window) {
    getnelems(stw, 2, func (size int) {
                          els := make([]*st.Elem, 2)
                          num := 0
                          for _, el := range stw.SelectElem {
                              if el != nil && el.IsLineElem() {
                                  els[num] = el
                                  num++
                                  if num >= 2 { break }
                              }
                          }
                          if num == 2 {
                              ns := make([]*st.Node, 4)
                              ns[0] = els[0].Enod[0]
                              ns[1] = els[0].Enod[1]
                              _, cw1 := ClockWise(ns[0].Pcoord, ns[1].Pcoord, els[1].Enod[0].Pcoord)
                              _, cw2 := ClockWise(ns[0].Pcoord, els[1].Enod[0].Pcoord, els[1].Enod[1].Pcoord)
                              if cw1 == cw2 {
                                  ns[2] = els[1].Enod[0]; ns[3] = els[1].Enod[1]
                              } else {
                                  ns[2] = els[1].Enod[1]; ns[3] = els[1].Enod[0]
                              }
                              sec := stw.Frame.DefaultSect()
                              el := stw.Frame.AddPlateElem(-1, ns, sec, st.NONE)
                              var buf bytes.Buffer
                              buf.WriteString(fmt.Sprintf("ELEM: %d (ENOD: ", el.Num))
                              for _, n := range ns {
                                  buf.WriteString(fmt.Sprintf("%d ", n.Num))
                              }
                              buf.WriteString(fmt.Sprintf(", SECT: %d)", sec.Num))
                              stw.addHistory(buf.String())
                              stw.EscapeAll()
                          }
                      })
}
// }}}


// SEARCHELEM// {{{
func searchelem (stw *Window) {
    if stw.SelectNode != nil && len(stw.SelectNode)>=1 {
        stw.SelectElem = stw.Frame.SearchElem(stw.SelectNode...)
        stw.Redraw()
        stw.EscapeCB()
        return
    }
    stw.Deselect()
    iup.SetFocus(stw.canv)
    startsearch := func(n *st.Node) {
        stw.SelectElem = stw.Frame.SearchElem(n)
        stw.addHistory(fmt.Sprintf("Select Element Using NODE %d", n.Num))
        stw.EscapeCB()
    }
    setnnum := func () {
        nnum, err := strconv.ParseInt(stw.cline.GetAttribute("VALUE"),10,64)
        if err == nil {
            if n, ok := stw.Frame.Nodes[int(nnum)]; ok {
                startsearch(n)
            } else {
                stw.addHistory(fmt.Sprintf("NODE %d not found", nnum))
            }
        }
        stw.cline.SetAttribute("VALUE", "")
    }
    stw.canv.SetCallback( func (arg *iup.CommonKeyAny) {
                              key := iup.KeyState(arg.Key)
                              switch key.Key() {
                              default:
                                  stw.DefaultKeyAny(key)
                                  // stw.cline.SetAttribute("INSERT", string(key.Key()))
                              case KEY_BS:
                                  val := stw.cline.GetAttribute("VALUE")
                                  if val != "" {
                                      stw.cline.SetAttribute("VALUE", val[:len(val)-1])
                                  }
                              case KEY_ENTER:
                                  setnnum()
                              case KEY_ESCAPE:
                                  stw.EscapeAll()
                              }
                          })
    stw.canv.SetCallback( func (arg *iup.MouseButton) {
                              if stw.Frame != nil {
                                  stw.dbuff.UpdateYAxis(&arg.Y)
                                  switch arg.Button {
                                  case BUTTON_LEFT:
                                      if arg.Pressed == 0 { // Released
                                          if n:= stw.PickNode(int(arg.X), int(arg.Y)); n != nil {
                                              startsearch(n)
                                          }
                                          stw.Redraw()
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
                                              stw.startX = int(arg.X); stw.startY = int(arg.Y)
                                          }
                                      }
                                  case BUTTON_RIGHT:
                                      if arg.Pressed == 0 {
                                          if v := stw.cline.GetAttribute("VALUE"); v!="" {
                                              setnnum()
                                          } else {
                                              stw.EscapeAll()
                                          }
                                      }
                                  }
                              }
                          })
}// }}}


// NODE <-> ELEM// {{{
func nodetoelemany (stw *Window) {
    stw.SelectElem = stw.Frame.NodeToElemAny(stw.SelectNode...)
    stw.EscapeCB()
}
func nodetoelemall (stw *Window) {
    stw.SelectElem = stw.Frame.NodeToElemAll(stw.SelectNode...)
    stw.EscapeCB()
}
func elemtonode (stw *Window) {
    stw.SelectNode = stw.Frame.ElemToNode(stw.SelectElem...)
    stw.EscapeCB()
}
func connected (stw *Window) {
    if stw.SelectNode != nil && len(stw.SelectNode)>=1 && stw.SelectNode[0]!=nil {
        stw.SelectNode = stw.Frame.LineConnected(stw.SelectNode[0])
    }
    stw.EscapeCB()
}
func onnode (stw *Window) {
    stw.SelectNode = make([]*st.Node, 0)
    if stw.SelectElem != nil {
        for _, el := range stw.SelectElem {
            ns := el.OnNode(0, 1e-4)
            stw.SelectNode = append(stw.SelectNode, ns...)
        }
    }
    stw.EscapeCB()
}
// }}}


// GETVECTOR// {{{
func getvector (stw *Window, f func(x,y,z float64)) {
    var snap *st.Node // for Snapping
    stw.canv.SetAttribute("CURSOR", "CROSS")
    stw.cdcanv.Foreground(cd.CD_YELLOW)
    stw.cdcanv.WriteMode(cd.CD_XOR)
    var startpoint *st.Node
    funcbynode := func (n *st.Node) {
        c := make([]float64, 3)
        for i:=0; i<3; i++ {
            c[i] = n.Coord[i] - startpoint.Coord[i]
        }
        f(c[0], c[1], c[2])
    }
    setnnum := func () {
        nnum, err := strconv.ParseInt(stw.cline.GetAttribute("VALUE"),10,64)
        if err == nil {
            if n, ok := stw.Frame.Nodes[int(nnum)]; ok {
                if startpoint != nil {
                    funcbynode(n)
                } else {
                    startpoint = n
                    stw.cdcanv.Foreground(cd.CD_DARK_RED)
                    stw.cdcanv.WriteMode(cd.CD_XOR)
                    first = 1
                }
            }
        }
        stw.cline.SetAttribute("VALUE", "")
    }
    stw.canv.SetCallback( func (arg *iup.MouseButton) {
                              if stw.Frame != nil {
                                  stw.dbuff.UpdateYAxis(&arg.Y)
                                  switch arg.Button {
                                  case BUTTON_LEFT:
                                      if arg.Pressed == 0 { // Released
                                          if startpoint != nil {
                                              if n := stw.PickNode(int(arg.X), int(arg.Y)); n != nil {
                                                  funcbynode(n)
                                              }
                                              // stw.cdcanv.Line(int(stw.SelectNode[0].Pcoord[0]), int(stw.SelectNode[0].Pcoord[1]), stw.endX, stw.endY) // TODO
                                          } else {
                                              if n := stw.PickNode(int(arg.X), int(arg.Y)); n != nil {
                                                  startpoint = n
                                                  stw.cdcanv.Foreground(cd.CD_DARK_RED)
                                                  stw.cdcanv.WriteMode(cd.CD_XOR)
                                                  first = 1
                                              }
                                          }
                                          stw.Redraw()
                                      }
                                  case BUTTON_CENTER:
                                      if arg.Pressed == 0 { // Released
                                          // if stw.SelectNode[0] != nil { // TODO
                                          //     stw.cdcanv.Line(int(stw.SelectNode[0].Pcoord[0]), int(stw.SelectNode[0].Pcoord[1]), stw.endX, stw.endY)
                                          // }
                                          stw.Redraw()
                                      } else { // Pressed
                                          if isDouble(arg.Status) {
                                              stw.Frame.SetFocus()
                                              stw.DrawFrameNode()
                                              stw.ShowCenter()
                                          } else {
                                              stw.startX = int(arg.X); stw.startY = int(arg.Y)
                                          }
                                      }
                                  case BUTTON_RIGHT:
                                      if arg.Pressed == 0 {
                                          if v := stw.cline.GetAttribute("VALUE"); v!="" {
                                              setnnum()
                                          } else {
                                              stw.EscapeAll()
                                          }
                                      }
                                  }
                              }
                          })
    stw.canv.SetCallback( func (arg *iup.MouseMotion) {
                              if stw.Frame != nil {
                                  stw.dbuff.UpdateYAxis(&arg.Y)
                                  // Snapping
                                  stw.cdcanv.Foreground(cd.CD_YELLOW)
                                  if snap != nil {
                                      stw.cdcanv.FCircle(snap.Pcoord[0], snap.Pcoord[1], nodeSelectPixel)
                                  }
                                  n := stw.PickNode(int(arg.X), int(arg.Y))
                                  if n != nil {
                                      stw.cdcanv.FCircle(n.Pcoord[0], n.Pcoord[1],  nodeSelectPixel)
                                  }
                                  snap = n
                                  ///
                                  switch statusKey(arg.Status) {
                                  case STATUS_CENTER:
                                      if isShift(arg.Status) {
                                          stw.Frame.View.Center[0] += float64(int(arg.X) - stw.startX) * CanvasMoveSpeedX
                                          stw.Frame.View.Center[1] += float64(int(arg.Y) - stw.startY) * CanvasMoveSpeedY
                                      } else {
                                          stw.Frame.View.Angle[0] -= float64(int(arg.Y) - stw.startY) * CanvasRotateSpeedY
                                          stw.Frame.View.Angle[1] -= float64(int(arg.X) - stw.startX) * CanvasRotateSpeedX
                                      }
                                      stw.DrawFrameNode()
                                  }
                                  if startpoint != nil {
                                      stw.cdcanv.Foreground(cd.CD_DARK_RED)
                                      stw.TailLine(int(startpoint.Pcoord[0]), int(startpoint.Pcoord[1]), arg)
                                  }
                              }
                          })
    stw.canv.SetCallback( func (arg *iup.CommonKeyAny) {
                              key := iup.KeyState(arg.Key)
                              switch key.Key() {
                              default:
                                  stw.DefaultKeyAny(key)
                              case KEY_BS:
                                  val := stw.cline.GetAttribute("VALUE")
                                  if val != "" {
                                      stw.cline.SetAttribute("VALUE", val[:len(val)-1])
                                  }
                              case KEY_ENTER:
                                  setnnum()
                              case KEY_ESCAPE:
                                  stw.EscapeAll()
                              case 'D','d':
                                  x, y, z, err := stw.QueryCoord("GET COORD")
                                  if err == nil {
                                      if startpoint == nil {
                                          f(x, y, z)
                                      } else {
                                          f(x-startpoint.Coord[0], y-startpoint.Coord[1], z-startpoint.Coord[2])
                                      }
                                  }
                              }
                          })
}// }}}


// COPYELEM// {{{
func copyelem (stw *Window) {
    if stw.SelectElem == nil || len(stw.SelectElem)==0 { return }
    stw.addHistory("移動距離を指定[ダイアログ(D)]")
    getvector(stw, func (x, y, z float64) {
                      if !(x==0.0 && y==0.0 && z==0.0) {
                          for _, el := range stw.SelectElem {
                              if el == nil || el.IsHide(stw.Frame.Show) || el.Lock { continue }
                              el.Copy(x, y, z)
                          }
                          stw.Redraw()
                      }
                  })
}
// }}}


// MOVEELEM// {{{
func moveelem (stw *Window) {
    if stw.SelectElem == nil || len(stw.SelectElem)==0 { return }
    stw.addHistory("移動距離を指定[ダイアログ(D)]")
    getvector(stw, func (x, y, z float64) {
                      for _, el := range stw.SelectElem {
                          if el == nil || el.IsHide(stw.Frame.Show) || el.Lock { continue }
                          el.Move(x, y, z)
                      }
                      for _, n := range stw.Frame.NodeNoReference() {
                          delete(stw.Frame.Nodes, n.Num)
                      }
                      stw.Redraw()
                  })
}
// }}}


// MOVENODE// {{{
func movenode (stw *Window) {
    ns := make([]*st.Node, 0)
    if stw.SelectNode != nil {
        for _, n := range stw.SelectNode {
            if n != nil { ns = append(ns, n) }
        }
    }
    if stw.SelectElem != nil {
        en := stw.Frame.ElemToNode(stw.SelectElem...)
        var add bool
        for _, n := range en {
            add = true
            for _, m := range(ns) {
                if m == n { add = false; break}
            }
            if add { ns = append(ns, n) }
        }
    }
    if len(ns)==0 { return }
    stw.addHistory("移動距離を指定[ダイアログ(D)]")
    getvector(stw, func (x, y, z float64) {
                      for _, n := range ns {
                          if n == nil || n.Hide || n.Lock { continue }
                          n.Move(x, y, z)
                      }
                      stw.EscapeCB()
                  })
}
// }}}


// MOVETOLINE// {{{
func movetoline (stw *Window) {
    fixed := 0
    ns := make([]*st.Node, 0)
    if stw.SelectNode != nil {
        for _, n := range stw.SelectNode {
            if n != nil { ns = append(ns, n) }
        }
    }
    if len(ns)==0 {
        stw.addHistory("移動する節点がありません")
        stw.EscapeAll()
        return
    }
    stw.addHistory("直線を指定[Xを固定]")
    get2nodes(stw, func (n *st.Node) {
                       stw.SelectNode[1] = n
                       for _, n := range ns {
                           n.MoveToLine(stw.SelectNode[0], stw.SelectNode[1], fixed)
                       }
                       stw.cdcanv.Foreground(cd.CD_WHITE)
                       stw.cdcanv.WriteMode(cd.CD_REPLACE)
                       stw.EscapeAll()
                   },
                   func () {
                       stw.SelectNode = make([]*st.Node, 2)
                       stw.Redraw()
                   })
    stw.canv.SetCallback( func (arg *iup.CommonKeyAny) {
                              key := iup.KeyState(arg.Key)
                              switch key.Key() {
                              default:
                                  stw.DefaultKeyAny(key)
                              case KEY_BS:
                                  val := stw.cline.GetAttribute("VALUE")
                                  if val != "" {
                                      stw.cline.SetAttribute("VALUE", val[:len(val)-1])
                                  }
                              case KEY_DELETE:
                                  stw.SelectNode = make([]*st.Node, 2)
                                  stw.Redraw()
                              case KEY_ESCAPE:
                                  stw.EscapeAll()
                              case 'X','x':
                                  fixed = 0
                                  stw.addHistory("直線を指定[Xを固定]")
                              case 'Y','y':
                                  fixed = 1
                                  stw.addHistory("直線を指定[Yを固定]")
                              case 'Z','z':
                                  fixed = 2
                                  stw.addHistory("直線を指定[Zを固定]")
                              }
                          })
}
// }}}


// PINCHNODE TODO: UNDER CONSTRUCTION// {{{
func pinchnode (stw *Window) {
    var target *st.Node
    movefunc := func (node *st.Node, dx, dy float64, arg *iup.MouseButton) {
                    node.Coord[2] += dy*0.01
                }
    stw.canv.SetCallback(func (arg *iup.MouseButton) {
                             stw.dbuff.UpdateYAxis(&arg.Y)
                             switch arg.Button {
                             case BUTTON_LEFT:
                                 if stw.Frame != nil {
                                     if arg.Pressed == 0 { // Released
                                         if target != nil {
                                             movefunc(target, float64(int(arg.X)-stw.startX), float64(int(arg.Y)-stw.startY), arg)
                                             target = nil
                                             stw.Redraw()
                                         }
                                     } else { // Pressed
                                         target = stw.PickNode(int(arg.X), int(arg.Y))
                                         stw.startX = int(arg.X); stw.startY = int(arg.Y)
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
                                         stw.startX = int(arg.X); stw.startY = int(arg.Y)
                                     }
                                 }
                             }
                         })
    stw.canv.SetCallback(func (arg *iup.MouseMotion) {
                             stw.dbuff.UpdateYAxis(&arg.Y)
                             switch statusKey(arg.Status) {
                             case STATUS_LEFT:
                                 if target != nil {
                                     fmt.Println(target.Num, float64(int(arg.X)-stw.startX), float64(int(arg.Y)-stw.startY))
                                 }
                             case STATUS_CENTER:
                                 if isShift(arg.Status) {
                                     stw.Frame.View.Center[0] += float64(int(arg.X) - stw.startX) * CanvasMoveSpeedX
                                     stw.Frame.View.Center[1] += float64(int(arg.Y) - stw.startY) * CanvasMoveSpeedY
                                 } else {
                                     stw.Frame.View.Angle[0] -= float64(int(arg.Y) - stw.startY) * CanvasRotateSpeedY
                                     stw.Frame.View.Angle[1] -= float64(int(arg.X) - stw.startX) * CanvasRotateSpeedX
                                 }
                                 stw.DrawFrameNode()
                             }
                         })
}
// }}}


// ROTATE// {{{
func rotate (stw *Window) {
    ns := make([]*st.Node, 0)
    if stw.SelectNode != nil {
        for _, n := range stw.SelectNode {
            if n != nil { ns = append(ns, n) }
        }
    }
    if stw.SelectElem != nil {
        en := stw.Frame.ElemToNode(stw.SelectElem...)
        var add bool
        for _, n := range en {
            add = true
            for _, m := range(ns) {
                if m == n { add = false; break}
            }
            if add { ns = append(ns, n) }
        }
    }
    if len(ns)==0 { return }
    stw.addHistory("回転中心を選択")
    get1node(stw, func (n0 *st.Node) {
                      stw.addHistory(fmt.Sprintf("NODE: %d", n0.Num))
                      stw.addHistory("回転軸を選択[ダイアログ(D)]")
                      getvector(stw, func (x, y, z float64) {
                                        stw.addHistory(fmt.Sprintf("X: %.3f, Y: %.3f, Z: %.3f", x, y, z))
                                        if !(x==0.0 && y==0.0 && z==0.0) {
                                            stw.addHistory("回転角を指定[参照(R)]")
                                            rot := func (angle float64) {
                                                       for _, n := range ns {
                                                           if n == nil || n.Hide || n.Lock { continue }
                                                           n.Rotate(n0.Coord, []float64{x, y, z}, angle)
                                                       }
                                                   }
                                            stw.canv.SetCallback( func (arg *iup.CommonKeyAny) {
                                                                      key := iup.KeyState(arg.Key)
                                                                      switch key.Key() {
                                                                      default:
                                                                          stw.DefaultKeyAny(key)
                                                                      case KEY_BS:
                                                                          val := stw.cline.GetAttribute("VALUE")
                                                                          if val != "" {
                                                                              stw.cline.SetAttribute("VALUE", val[:len(val)-1])
                                                                          }
                                                                      case KEY_ENTER:
                                                                          val, err := strconv.ParseFloat(stw.cline.GetAttribute("VALUE"), 64)
                                                                          if err != nil {
                                                                              stw.EscapeCB()
                                                                              return
                                                                          }
                                                                          rot(val*math.Pi/180)
                                                                          stw.EscapeCB()
                                                                      case 'R','r':
                                                                          stw.addHistory("参照点を指定")
                                                                          getnnodes(stw, 3, func (num int) {
                                                                                                if num<3 {
                                                                                                    stw.EscapeCB()
                                                                                                    return
                                                                                                }
                                                                                                u := []float64{ x, y, z }
                                                                                                u = st.Normalize(u)
                                                                                                v1 := make([]float64, 3)
                                                                                                v2 := make([]float64, 3)
                                                                                                for i:=0; i<3; i++ {
                                                                                                    v1[i] = stw.SelectNode[1].Coord[i] - stw.SelectNode[0].Coord[i]
                                                                                                    v2[i] = stw.SelectNode[2].Coord[i] - stw.SelectNode[0].Coord[i]
                                                                                                }
                                                                                                uv1 := st.Dot(u, v1, 3)
                                                                                                uv2 := st.Dot(u, v2, 3)
                                                                                                v12 := st.Dot(v1, v2, 3)
                                                                                                v11 := st.Dot(v1, v1, 3)
                                                                                                v22 := st.Dot(v2, v2, 3)
                                                                                                w := math.Sqrt(v11-uv1*uv1) * math.Sqrt(v22-uv2*uv2)
                                                                                                if w != 0.0 {
                                                                                                    rot(math.Acos((v12-uv1*uv2)/w))
                                                                                                }
                                                                                                stw.SelectNode = make([]*st.Node, 0)
                                                                                                stw.EscapeCB()
                                                                                                return
                                                                                            })
                                                                      }
                                                                  })
                                        } else {
                                            stw.EscapeCB()
                                        }
                                    })
                  })
}
// }}}


// MIRROR// {{{
func mirror (stw *Window) {
    if stw.SelectElem == nil || len(stw.SelectElem) == 0 { return }
    ns := stw.Frame.ElemToNode(stw.SelectElem...)
    stw.addHistory("対称面を指定[1点又は3点]")
    maxnum := 3
    createmirrors := func (coord, vec []float64) {
                         nmap := make(map[int]*st.Node, len(ns))
                         for _, n := range ns {
                             c := n.MirrorCoord(coord, vec)
                             nmap[n.Num] = stw.Frame.CoordNode(c[0], c[1], c[2])
                             for i:=0; i<6; i++ {
                                 nmap[n.Num].Conf[i] = n.Conf[i]
                             }
                         }
                         for _, el := range stw.SelectElem {
                             newenod := make([]*st.Node, el.Enods)
                             var add bool
                             for i:=0; i<el.Enods; i++ {
                                 newenod[i] = nmap[el.Enod[i].Num]
                                 if !add && newenod[i] != el.Enod[i] { add = true }
                             }
                             if add {
                                 if el.IsLineElem() {
                                     e := stw.Frame.AddLineElem(-1, newenod, el.Sect, el.Etype)
                                     for i:=0; i<6*el.Enods; i++ {
                                         e.Bonds[i] = el.Bonds[i]
                                     }
                                 } else {
                                     stw.Frame.AddPlateElem(-1, newenod, el.Sect, el.Etype)
                                 }
                             }
                         }
                     }
    getnnodes(stw, maxnum, func (num int) {
                               switch num {
                               default:
                                   stw.EscapeAll()
                               case 1:
                                   stw.addHistory("法線方向を選択[ダイアログ(D)]")
                                   getvector(stw, func (x, y, z float64) {
                                                     createmirrors(stw.SelectNode[0].Coord, []float64{x, y, z})
                                                     stw.EscapeAll()
                                                 })
                               case 3:
                                   v1 := make([]float64, 3)
                                   v2 := make([]float64, 3)
                                   for i:=0; i<3; i++ {
                                       v1[i] = stw.SelectNode[1].Coord[i] - stw.SelectNode[0].Coord[i]
                                       v2[i] = stw.SelectNode[2].Coord[i] - stw.SelectNode[0].Coord[i]
                                   }
                                   vec := st.Cross(v1, v2)
                                   createmirrors(stw.SelectNode[0].Coord, vec)
                                   stw.EscapeAll()
                               }
                           })
}
// }}}


// SCALE// {{{
func scale (stw *Window) {
    ns := make([]*st.Node, 0)
    if stw.SelectNode != nil {
        for _, n := range stw.SelectNode {
            if n != nil { ns = append(ns, n) }
        }
    }
    if stw.SelectElem != nil {
        en := stw.Frame.ElemToNode(stw.SelectElem...)
        var add bool
        for _, n := range en {
            add = true
            for _, m := range(ns) {
                if m == n { add = false; break}
            }
            if add { ns = append(ns, n) }
        }
    }
    if len(ns)==0 { return }
    stw.addHistory("原点を指定[ダイアログ(D)]")
    get1node(stw, func (n0 *st.Node) {
                      tmp, err := stw.Query("倍率を指定")
                      if err != nil {
                          stw.addHistory(err.Error())
                          stw.EscapeCB()
                      }
                      val, err := strconv.ParseFloat(tmp, 64)
                      if err != nil {
                          stw.addHistory(err.Error())
                          stw.EscapeCB()
                      }
                      for _, n := range ns {
                          if n == nil || n.Hide || n.Lock { continue }
                          n.Scale(n0.Coord, val)
                      }
                      stw.EscapeCB()
                  })
}
// }}}


// OPEN// {{{
func openinput(stw *Window) {
    stw.Open()
    stw.EscapeAll()
}
// }}}


// SAVE// {{{
func saveinput(stw *Window) {
    stw.SaveFile(stw.Frame.Path)
    stw.EscapeAll()
}
// }}}


// WEIGHTCOPY// {{{
func weightcopy(stw *Window) {
    wgt := filepath.Join(stw.Home, "hogtxt.wgt")
    fn := st.Ce(stw.Frame.Path, ".wgt")
    if (!st.FileExists(fn) || stw.Yn("Copy Wgt", "上書きしますか")) {
        st.CopyFile(wgt, fn)
        stw.addHistory(fmt.Sprintf("COPY: %s", fn))
        err := stw.Frame.ReadWgt(fn)
        if err != nil {
            stw.addHistory(err.Error())
        }
    }
    stw.EscapeAll()
}
// }}}


// READPGP// {{{
func readpgp (stw *Window) {
    if name,ok := iup.GetOpenFile(stw.Cwd, "*.pgp"); ok {
        al := make(map[string]*Command,0)
        err := ReadPgp(name, al)
        if err != nil {
            stw.addHistory("ReadPgp: Cannot Read st.pgp")
        } else {
            aliases = al
            stw.addHistory(fmt.Sprintf("ReadPgp: Read %s", name))
        }
    }
    stw.EscapeAll()
}
// }}}


// INSERT// {{{
func insert(stw *Window) {
    if name,ok := iup.GetOpenFile("",""); ok {
        get1node(stw, func (n *st.Node) {
                          // TODO: 角度を指定
                          err := stw.Frame.ReadInp(name, n.Coord, 0.0)
                          if err != nil {
                              stw.addHistory(err.Error())
                          }
                          stw.EscapeAll()
                      })
    } else {
        stw.EscapeAll()
    }
    stw.EscapeAll()
}
// }}}


// SETFOCUS// {{{
func setfocus(stw *Window) {
    stw.Frame.SetFocus()
    stw.Redraw()
    stw.addHistory(fmt.Sprintf("FOCUS: %.1f %.1f %.1f", stw.Frame.View.Focus[0], stw.Frame.View.Focus[1], stw.Frame.View.Focus[2]))
    stw.EscapeAll()
}
// }}}


// GET1ELEM// {{{
func get1elem (stw *Window, f func(*st.Elem, int, int), condition func(*st.Elem) bool) {
    stw.SelectElem = make([]*st.Elem, 1)
    selected := false
    stw.canv.SetCallback( func (arg *iup.MouseButton) {
                              if stw.Frame != nil {
                                  stw.dbuff.UpdateYAxis(&arg.Y)
                                  switch arg.Button {
                                  case BUTTON_LEFT:
                                      if arg.Pressed == 0 {
                                          if selected {
                                              if el := stw.PickElem(int(arg.X), int(arg.Y)); el != nil {
                                                  f(el, int(arg.X), int(arg.Y))
                                              }
                                          } else {
                                              if el := stw.PickElem(int(arg.X), int(arg.Y)); el != nil {
                                                  if condition(el) {
                                                      stw.SelectElem[0] = el
                                                      selected = true
                                                      stw.canv.SetAttribute("CURSOR", "PEN")
                                                  }
                                                  stw.Redraw()
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
                                              stw.startX = int(arg.X); stw.startY = int(arg.Y)
                                          }
                                      }
                                  case BUTTON_RIGHT:
                                      if arg.Pressed == 0 {
                                          stw.EscapeAll()
                                      }
                                  }
                              }
                          })
    stw.canv.SetCallback( func (arg *iup.MouseMotion) {
                              if stw.Frame != nil {
                                  stw.dbuff.UpdateYAxis(&arg.Y)
                                  switch statusKey(arg.Status) {
                                  case STATUS_CENTER:
                                      if isShift(arg.Status) {
                                          stw.Frame.View.Center[0] += float64(int(arg.X) - stw.startX) * CanvasMoveSpeedX
                                          stw.Frame.View.Center[1] += float64(int(arg.Y) - stw.startY) * CanvasMoveSpeedY
                                      } else {
                                          stw.Frame.View.Angle[0] -= float64(int(arg.Y) - stw.startY) * CanvasRotateSpeedY
                                          stw.Frame.View.Angle[1] -= float64(int(arg.X) - stw.startX) * CanvasRotateSpeedX
                                      }
                                      stw.DrawFrameNode()
                                  }
                              }
                          })
}
// }}}


// MATCHPROP// {{{
func matchproperty (stw *Window) {
    get1elem(stw, func (el *st.Elem, x, y int) {
                      el.Sect = stw.SelectElem[0].Sect
                      el.Etype = stw.SelectElem[0].Etype
                      stw.Redraw()
                  },
                  func (el *st.Elem) bool {
                      return true
                  })
}
// }}}


// COPYBOND// {{{
func copybond (stw *Window) {
    get1elem(stw, func (el *st.Elem, x, y int) {
                      if el.IsLineElem() {
                          for i:=0; i<12; i++ {
                              el.Bonds[i] = stw.SelectElem[0].Bonds[i]
                          }
                      }
                      stw.Redraw()
                  },
                  func (el *st.Elem) bool {
                      return el.IsLineElem()
                  })
}
// }}}


// AXISTOCANG// {{{
func axistocang (stw *Window) {
    if stw.SelectElem == nil || len(stw.SelectElem) == 0 { stw.EscapeAll(); return }
    stw.addHistory("部材軸の方向を指定[強軸(S)/弱軸(W)/切替(T)/ダイアログ(D)]<強軸>")
    strong := true
    axisfunc :=   func (x, y, z float64) {
                      if !(x==0.0 && y==0.0 && z==0.0) {
                          for _, el := range stw.SelectElem {
                              if el == nil || el.IsHide(stw.Frame.Show) || el.Lock { continue }
                              _, err := el.AxisToCang([]float64{x, y, z}, strong)
                              if err != nil {
                                  stw.addHistory(fmt.Sprintf("部材軸を設定できません: ELEM %d", el.Num))
                              }
                          }
                          stw.EscapeAll()
                      }
                  }
    getvector(stw, axisfunc)
    stw.canv.SetCallback( func (arg *iup.CommonKeyAny) {
                              key := iup.KeyState(arg.Key)
                              switch key.Key() {
                              default:
                                  stw.DefaultKeyAny(key)
                              case KEY_BS:
                                  val := stw.cline.GetAttribute("VALUE")
                                  if val != "" {
                                      stw.cline.SetAttribute("VALUE", val[:len(val)-1])
                                  }
                              case KEY_ENTER:
                              case KEY_ESCAPE:
                                  stw.EscapeAll()
                              case 'D','d':
                                  x, y, z, err := stw.QueryCoord("GET COORD")
                                  if err == nil {
                                      axisfunc(x, y, z)
                                  }
                              case 'S','s':
                                  stw.addHistory("<強軸>方向を指定")
                                  strong = true
                              case 'W','w':
                                  stw.addHistory("<弱軸>方向を指定")
                                  strong = false
                              case 'T','t':
                                  if strong {
                                      stw.addHistory("<弱軸>方向を指定")
                                  } else {
                                      stw.addHistory("<強軸>方向を指定")
                                  }
                                  strong = !strong
                              }
                          })
}
// }}}


// BOND// {{{
// BONDPIN
func bondpin (stw *Window) {
    if stw.SelectElem == nil { stw.EscapeAll(); return }
    for _, el := range stw.SelectElem {
        if el == nil || el.Lock { continue }
        if el.IsLineElem() {
            for i:=0; i<2; i++ {
                for j:=4; j<6; j++ {
                    el.Bonds[6*i+j] = true
                }
            }
        }
    }
    stw.EscapeAll()
}

func bondrigid (stw *Window) {
    if stw.SelectElem == nil { stw.EscapeAll(); return }
    for _, el := range stw.SelectElem {
        if el == nil || el.Lock { continue }
        if el.IsLineElem() {
            for i:=0; i<2; i++ {
                for j:=0; j<6; j++ {
                    el.Bonds[6*i+j] = false
                }
            }
        }
    }
    stw.EscapeAll()
}
// }}}


// CONF// {{{
func setconf (stw *Window, lis []bool) {
    if stw.SelectNode == nil { stw.EscapeAll(); return }
    for _, n := range stw.SelectNode {
        if n == nil || n.Lock { continue }
        for i:=0; i<6; i++ {
            n.Conf[i] = lis[i]
        }
    }
    stw.EscapeCB()
}

// CONFFIX
func conffix (stw *Window) {
    if stw.SelectNode == nil { stw.EscapeAll(); return }
    for _, n := range stw.SelectNode {
        if n == nil || n.Lock { continue }
        for i:=0; i<3; i++ {
            n.Conf[i] = true
            n.Conf[i+3] = true
        }
    }
    stw.EscapeCB()
}

// CONFPIN
func confpin (stw *Window) {
    if stw.SelectNode == nil { stw.EscapeAll(); return }
    for _, n := range stw.SelectNode {
        if n == nil || n.Lock { continue }
        for i:=0; i<3; i++ {
            n.Conf[i] = true
            n.Conf[i+3] = false
        }
    }
    stw.EscapeCB()
}

// CONFXYROLLER
func confxyroller (stw *Window) {
    if stw.SelectNode == nil { stw.EscapeAll(); return }
    for _, n := range stw.SelectNode {
        if n == nil || n.Lock { continue }
        for i:=0; i<3; i++ {
            n.Conf[i] = false
            n.Conf[i+3] = false
        }
        n.Conf[2] = true
    }
    stw.EscapeCB()
}

// CONFFREE
func conffree (stw *Window) {
    if stw.SelectNode == nil { stw.EscapeAll(); return }
    for _, n := range stw.SelectNode {
        if n == nil || n.Lock { continue }
        for i:=0; i<3; i++ {
            n.Conf[i] = false
            n.Conf[i+3] = false
        }
    }
    stw.EscapeCB()
}
// }}}


// TRIM// {{{
func trim (stw *Window) {
    get1elem(stw, func (el *st.Elem, x, y int) {
                      if el.IsLineElem() {
                          var err error
                          if FDotLine(stw.SelectElem[0].Enod[0].Pcoord[0], stw.SelectElem[0].Enod[0].Pcoord[1], stw.SelectElem[0].Enod[1].Pcoord[0], stw.SelectElem[0].Enod[1].Pcoord[1], float64(x), float64(y)) * FDotLine(stw.SelectElem[0].Enod[0].Pcoord[0], stw.SelectElem[0].Enod[0].Pcoord[1], stw.SelectElem[0].Enod[1].Pcoord[0], stw.SelectElem[0].Enod[1].Pcoord[1], el.Enod[0].Pcoord[0], el.Enod[0].Pcoord[1]) < 0.0 {
                              _, _, err = stw.Frame.Trim(stw.SelectElem[0], el, 1)
                          } else {
                              _, _, err = stw.Frame.Trim(stw.SelectElem[0], el, -1)
                          }
                          if err != nil {
                              stw.addHistory(err.Error())
                          } else {
                              stw.Deselect()
                              stw.Redraw()
                          }
                          stw.EscapeAll()
                      }
                      stw.Redraw()
                  },
                  func (el *st.Elem) bool {
                      return el.IsLineElem()
                  })
}
// }}}


// EXTEND// {{{
func extend (stw *Window) {
    get1elem(stw, func (el *st.Elem, x, y int) {
                      if el.IsLineElem() {
                          var err error
                          _, _, err = stw.Frame.Extend(stw.SelectElem[0], el)
                          if err != nil {
                              stw.addHistory(err.Error())
                          } else {
                              stw.Deselect()
                              stw.Redraw()
                          }
                          stw.EscapeAll()
                      }
                      stw.Redraw()
                  },
                  func (el *st.Elem) bool {
                      return el.IsLineElem()
                  })
}
// }}}


// SELECTNODE// {{{
func selectnode (stw *Window) {
    stw.Deselect()
    iup.SetFocus(stw.canv)
    setnnum := func () {
        nnum, err := strconv.ParseInt(stw.cline.GetAttribute("VALUE"),10,64)
        if err == nil {
            if n, ok := stw.Frame.Nodes[int(nnum)]; ok {
                stw.addHistory(fmt.Sprintf("NODE %d Selected", nnum))
                stw.SelectNode = make([]*st.Node, 1)
                stw.SelectNode[0] = n
                stw.EscapeCB()
            } else {
                stw.addHistory(fmt.Sprintf("NODE %d not found", nnum))
            }
        }
        stw.cline.SetAttribute("VALUE", "")
    }
    stw.canv.SetCallback( func (arg *iup.CommonKeyAny) {
                              key := iup.KeyState(arg.Key)
                              switch key.Key() {
                              default:
                                  stw.DefaultKeyAny(key)
                                  // stw.cline.SetAttribute("INSERT", string(key.Key()))
                              case KEY_BS:
                                  val := stw.cline.GetAttribute("VALUE")
                                  if val != "" {
                                      stw.cline.SetAttribute("VALUE", val[:len(val)-1])
                                  }
                              case KEY_ENTER:
                                  setnnum()
                              case KEY_ESCAPE:
                                  stw.EscapeAll()
                              }
                          })
}// }}}


// SELECTCOLUMNBASE// {{{
func selectcolumnbase (stw *Window) {
    stw.Deselect()
    var min, max float64
    nnum := 0
    ns := make([]*st.Node, len(stw.Frame.Nodes))
    if len(stw.Frame.Ai.Boundary) >= 2 {
        min = stw.Frame.Ai.Boundary[0]
        max = stw.Frame.Ai.Boundary[1]
    } else {
        min = 0.0
        max = 0.0
    }
    for _, el := range stw.Frame.Elems {
        if el.Etype != st.COLUMN || el.IsHide(stw.Frame.Show) { continue }
        for _, n := range el.Enod {
            if min<=n.Coord[2] && n.Coord[2]<=max {
                ns[nnum] = n
                nnum++
            }
        }
    }
    stw.SelectNode = ns[:nnum]
    stw.EscapeCB()
}
// }}}


// SELECTCONFED// {{{
func selectconfed (stw *Window) {
    stw.Deselect()
    num := 0
    for _, n := range stw.Frame.Nodes {
        if n.Hide { continue }
        for i:=0; i<6; i++ {
            if n.Conf[i] {
                stw.SelectNode = append(stw.SelectNode, n)
                num++
                break
            }
        }
    }
    stw.SelectNode = stw.SelectNode[:num]
    stw.EscapeCB()
}
// }}}


// SELECTELEM// {{{
func selectelem (stw *Window) {
    stw.Deselect()
    iup.SetFocus(stw.canv)
    setenum := func () {
        enum, err := strconv.ParseInt(stw.cline.GetAttribute("VALUE"),10,64)
        if err == nil {
            if el, ok := stw.Frame.Elems[int(enum)]; ok {
                stw.addHistory(fmt.Sprintf("ELEM %d Selected", enum))
                stw.SelectElem = make([]*st.Elem, 1)
                stw.SelectElem[0] = el
                stw.EscapeCB()
            } else {
                stw.addHistory(fmt.Sprintf("ELEM %d not found", enum))
            }
        }
        stw.cline.SetAttribute("VALUE", "")
    }
    stw.canv.SetCallback( func (arg *iup.CommonKeyAny) {
                              key := iup.KeyState(arg.Key)
                              switch key.Key() {
                              default:
                                  stw.DefaultKeyAny(key)
                                  // stw.cline.SetAttribute("INSERT", string(key.Key()))
                              case KEY_BS:
                                  val := stw.cline.GetAttribute("VALUE")
                                  if val != "" {
                                      stw.cline.SetAttribute("VALUE", val[:len(val)-1])
                                  }
                              case KEY_ENTER:
                                  setenum()
                              case KEY_ESCAPE:
                                  stw.EscapeAll()
                              }
                          })
}// }}}


// SELECTSECT// {{{
func selectsect (stw *Window) {
    stw.Frame.Show.ElemCaption |= st.EC_SECT
    stw.Labels["EC_SECT"].SetAttribute("FGCOLOR", labelFGColor)
    stw.Deselect()
    stw.Redraw()
    iup.SetFocus(stw.canv)
    setsnum := func () {
        tmp, err := strconv.ParseInt(stw.cline.GetAttribute("VALUE"),10,64)
        if err == nil {
            snum := int(tmp)
            if _, ok := stw.Frame.Sects[snum]; ok {
                stw.addHistory(fmt.Sprintf("SECT %d Selected", snum))
                stw.SelectElem = make([]*st.Elem, 0)
                for _, el := range stw.Frame.Elems {
                    if el.IsHide(stw.Frame.Show) { continue }
                    if el.Sect.Num == snum {
                        stw.SelectElem = append(stw.SelectElem, el)
                    }
                }
                stw.EscapeCB()
            } else {
                stw.addHistory(fmt.Sprintf("SECT %d not found", snum))
            }
        }
        stw.cline.SetAttribute("VALUE", "")
    }
    stw.canv.SetCallback( func (arg *iup.CommonKeyAny) {
                              key := iup.KeyState(arg.Key)
                              switch key.Key() {
                              default:
                                  stw.DefaultKeyAny(key)
                                  // stw.cline.SetAttribute("INSERT", string(key.Key()))
                              case KEY_BS:
                                  val := stw.cline.GetAttribute("VALUE")
                                  if val != "" {
                                      stw.cline.SetAttribute("VALUE", val[:len(val)-1])
                                  }
                              case KEY_ENTER:
                                  setsnum()
                              case KEY_ESCAPE:
                                  stw.EscapeAll()
                              }
                          })
}
// }}}


// HIDESECTION// {{{
func hidesection (stw *Window) {
    hide := func () {
        tmp, err := strconv.ParseInt(stw.cline.GetAttribute("VALUE"),10,64)
        if err == nil {
            snum := int(tmp)
            stw.Frame.Show.Sect[snum] = false
            stw.Redraw()
        }
        stw.cline.SetAttribute("VALUE", "")
    }
    stw.canv.SetCallback( func (arg *iup.CommonKeyAny) {
                              key := iup.KeyState(arg.Key)
                              switch key.Key() {
                              default:
                                  stw.DefaultKeyAny(key)
                              case KEY_BS:
                                  val := stw.cline.GetAttribute("VALUE")
                                  if val != "" {
                                      stw.cline.SetAttribute("VALUE", val[:len(val)-1])
                                  }
                              case KEY_ENTER:
                                  hide()
                              case KEY_ESCAPE:
                                  stw.EscapeAll()
                              }
                          })
}
// }}}


// HIDECURTAINWALL// {{{
func hidecurtainwall (stw *Window) {
    for _, sec := range stw.Frame.Sects {
        if sec.Num > 900 { continue }
        if sec.HasArea() { continue }
        if !sec.HasBrace() {
            stw.Frame.Show.Sect[sec.Num] = false
        }
    }
    stw.EscapeAll()
}
// }}}


// SELECTCHILDREN// {{{
func selectchildren (stw *Window) {
    getnelems(stw, 10, func (size int) {
                           els := make([]*st.Elem, 2)
                           num := 0
                           for _, el := range stw.SelectElem {
                               if el != nil && !el.IsLineElem() {
                                   els[num] = el
                                   num++
                               }
                           }
                           if num > 0 {
                               tmpels := make([]*st.Elem, num*2)
                               nc := 0
                               for _, el := range stw.SelectElem {
                                   if el.Children != nil {
                                       for _, c := range el.Children {
                                           if c != nil {
                                               tmpels[nc] = c
                                               nc++
                                           }
                                       }
                                   }
                               }
                               stw.SelectElem = tmpels[:nc]
                               stw.EscapeCB()
                           }
                       })
}
// }}}


// NODENOREFERENCE// {{{
func nodenoreference (stw *Window) {
    stw.Deselect()
    ns := stw.Frame.NodeNoReference()
    if len(ns) != 0 {
        stw.SelectNode = ns
        stw.Redraw()
        if stw.Yn("NODE NO REFERENCE", "不要な節点を削除しますか?") {
            for _, n := range ns {
                delete(stw.Frame.Nodes, n.Num)
            }
        }
    }
    stw.EscapeAll()
}
// }}}


// ELEMSAMENODE// {{{
func elemsamenode (stw *Window) {
    stw.Deselect()
    els := stw.Frame.ElemSameNode()
    if len(els) !=0 {
        stw.SelectNode = stw.Frame.ElemToNode(els...)
        stw.Redraw()
        if stw.Yn("ELEM SAME NODE", "部材を削除しますか?") {
            for _, el := range els {
                if el.Lock { continue }
                delete(stw.Frame.Elems, el.Num)
            }
        }
    }
    stw.EscapeAll()
}
// }}}


// PRUNEENOD// {{{
func pruneenod (stw *Window) {
    stw.Deselect()
    tmpels := stw.Frame.ElemSameNode()
    l := len(tmpels)
    if l !=0 {
        els := make([]*st.Elem, l)
        enum := 0
        for _, el := range tmpels {
            if !el.IsLineElem() {
                els[enum] = el
                enum++
            }
        }
        stw.SelectElem = els[:enum]
        stw.Redraw()
        if stw.Yn("ELEM SAME NODE", "部材の不要なENODを削除しますか?") {
            for _, el := range els[:enum] {
                if el.Lock { continue }
                el.PruneEnod()
            }
        }
    }
    stw.EscapeAll()
}
// }}}


// NODEDUPLICATION// {{{
func nodeduplication (stw *Window) {
    stw.Deselect()
    nm := stw.Frame.NodeDuplication(5e-3)
    if len(nm) != 0 {
        for k := range nm {
            stw.SelectNode = append(stw.SelectNode, k)
        }
        stw.Redraw()
        if stw.Yn("NODE DUPLICATION", "重なった節点を削除しますか?") {
            stw.Frame.ReplaceNode(nm)
        }
    }
    stw.EscapeAll()
}
// }}}


// ELEMDUPLICATION// {{{
func elemduplication (stw *Window) {
    stw.Deselect()
    els := stw.Frame.ElemDuplication()
    if len(els) != 0 {
        for k := range els {
            stw.SelectElem = append(stw.SelectElem, k)
        }
        stw.Redraw()
        if stw.Yn("ELEM DUPLICATION", "重なった部材を削除しますか?") {
            for el := range els {
                if el.Lock { continue }
                delete(stw.Frame.Elems, el.Num)
            }
            stw.EscapeAll()
        } else {
            stw.EscapeCB()
        }
    } else {
        stw.EscapeAll()
    }
}
// }}}


// CHECKFRAME// {{{
func checkframe (stw *Window) {
    stw.Deselect()
    nodenoreference(stw)
    nodeduplication(stw)
    elemsamenode(stw)
    elemduplication(stw)
    eall := true
    ns, els, ok := stw.Frame.Check()
    if !ok {
        stw.SelectNode = ns
        stw.SelectElem = els
        stw.Redraw()
        if stw.Yn("CHECK FRAME", "無効な節点と部材を削除しますか？") {
            for _, n := range els {
                if n.Lock { continue }
                delete(stw.Frame.Nodes, n.Num)
            }
            for _, el := range els {
                if el.Lock { continue }
                delete(stw.Frame.Elems, el.Num)
            }
        } else {
            eall = false
        }
    }
    if !stw.Frame.IsUpside() {
        if stw.Yn("CHECK FRAME", "部材の向きを修正しますか？") {
            stw.Frame.Upside()
        }
    }
    if eall {
        stw.EscapeAll()
    } else {
        stw.EscapeCB()
    }
}
// }}}


// NODESORT// {{{
func nodesort (stw *Window) {
    bw := stw.Frame.BandWidth()
    stw.addHistory(fmt.Sprintf("並び替え前: %d", bw))
    ns := func (d int) {
        bw, err := stw.Frame.NodeSort(d)
        if err != nil {
            stw.addHistory("並び替えエラー")
            stw.EscapeAll()
        }
        stw.addHistory(fmt.Sprintf("並び替え後: %d (%s方向)", bw, []string{"X", "Y", "Z"}[d]))
        stw.Redraw()
    }
    stw.canv.SetCallback( func (arg *iup.CommonKeyAny) {
                              key := iup.KeyState(arg.Key)
                              switch key.Key() {
                              default:
                                  stw.DefaultKeyAny(key)
                                  // stw.cline.SetAttribute("INSERT", string(key.Key()))
                              case '0', 'X', 'x':
                                  ns(0)
                              case '1', 'Y', 'y':
                                  ns(1)
                              case '2', 'Z', 'z':
                                  ns(2)
                              case KEY_ESCAPE:
                                  stw.EscapeAll()
                              }
                          })
}
// }}}


// EXTRACTARCLM// {{{
func extractarclm (stw *Window) {
    var err error
    var name string
    var ok bool
    saved := true
    stw.Frame.ExtractArclm()
    ans := stw.Yna("Extract Arclm", ".inl, .ihx, .ihyを保存しますか?", "別名で保存")
    switch ans {
    default:
        saved = false
    case 1:
        name = stw.Frame.Name
        err = stw.Frame.SaveAsArclm("")
        if err != nil { saved = false }
    case 3:
        name, ok = iup.GetSaveFile("", "")
        if ok {
            err = stw.Frame.SaveAsArclm(name)
            if err != nil { saved = false }
        } else {
            saved = false
        }
    }
    if saved {
        for _, ext := range st.InputExt {
            fn := st.Ce(name, ext)
            stw.addHistory(fmt.Sprintf("保存しました: %s", fn))
        }
    }
    stw.EscapeAll()
}
// }}}


// ARCLM001 TODO: UNDER CONSTRUCTION // {{{
func arclm001 (stw *Window) {
    stw.Frame.AssemGlobalMatrix()
    stw.EscapeAll()
}
// }}}


// FENCE// {{{
func fence (stw *Window) {
    iup.SetFocus(stw.canv)
    stw.canv.SetCallback( func (arg *iup.MouseButton) {
                              if stw.Frame != nil {
                                  switch arg.Button {
                                  case BUTTON_LEFT:
                                      stw.SelectElemFenceStart(arg)
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
                                              stw.startX = int(arg.X); stw.startY = int(arg.Y)
                                          }
                                      }
                                  }
                              }
                          })
    stw.canv.SetCallback( func (arg *iup.MouseMotion) {
                              if stw.Frame != nil {
                                  stw.dbuff.UpdateYAxis(&arg.Y)
                                  switch statusKey(arg.Status) {
                                  case STATUS_LEFT:
                                      stw.SelectElemFenceMotion(arg)
                                  case STATUS_CENTER:
                                      if isShift(arg.Status) {
                                          stw.Frame.View.Center[0] += float64(int(arg.X) - stw.startX) * CanvasMoveSpeedX
                                          stw.Frame.View.Center[1] += float64(int(arg.Y) - stw.startY) * CanvasMoveSpeedY
                                      } else {
                                          stw.Frame.View.Angle[0] -= float64(int(arg.Y) - stw.startY) * CanvasRotateSpeedY
                                          stw.Frame.View.Angle[1] -= float64(int(arg.X) - stw.startX) * CanvasRotateSpeedX
                                      }
                                      stw.DrawFrameNode()
                                  }
                              }
                          })
}
// }}}


// ERRORELEM// {{{
func errorelem (stw *Window) {
    iup.SetFocus(stw.canv)
    stw.Deselect()
    stw.SetColorMode(st.ECOLOR_RATE)
    stw.Frame.Show.ElemCaption |= st.EC_RATE_L
    stw.Frame.Show.ElemCaption |= st.EC_RATE_S
    stw.Labels["EC_RATE_L"].SetAttribute("FGCOLOR", labelFGColor)
    stw.Labels["EC_RATE_S"].SetAttribute("FGCOLOR", labelFGColor)
    stw.Redraw()
    tmpels := make([]*st.Elem, len(stw.Frame.Elems))
    i:=0
    for _, el := range(stw.Frame.Elems) {
        switch el.Etype {
        case st.COLUMN, st.GIRDER, st.BRACE, st.WALL, st.SLAB:
            val := el.RateMax(stw.Frame.Show)
            if val > 1.0 { tmpels[i] = el; i++; }
        }
    }
    stw.SelectElem = tmpels[:i]
    stw.EscapeCB()
}
// }}}


// CUTTER TODO: UNDER CONSTRUCTION // {{{
func cutter (stw *Window) {
    var ans string
    var axis int
    var err error
    ans, err = stw.Query("Axis")
    tmp, err := strconv.ParseInt(ans, 10, 64)
    if err != nil {
        switch strings.ToUpper(ans) {
        default:
            stw.EscapeAll()
        case "X":
            axis = 0
        case "Y":
            axis = 1
        case "Z":
            axis = 2
        }
    }
    axis = int(tmp)
    ans, err = stw.Query("Coord")
    coord, err := strconv.ParseFloat(ans, 64)
    if err != nil { return }
    stw.Frame.Cutter(axis, coord)
    stw.EscapeAll()
}
// }}}


// DIVIDE// {{{
func divide (stw *Window, divfunc func(*st.Elem)([]*st.Node, []*st.Elem, error)) {
    if stw.SelectElem != nil {
        tmpels := make([]*st.Elem, 0)
        for _, el := range stw.SelectElem {
            _, els, err := divfunc(el)
            if err ==  nil && len(els) > 1 {
                tmpels = append(tmpels, els...)
            }
        }
        stw.SelectElem = tmpels
        stw.EscapeCB()
    }
}
func divideatons (stw *Window) {
    divide(stw, func (el *st.Elem) ([]*st.Node, []*st.Elem, error) {
                    return el.DivideAtOns(1e-4)
                })
}
func divideatmid (stw *Window) {
    divide(stw, func (el *st.Elem) ([]*st.Node, []*st.Elem, error) {
                    return el.DivideAtMid()
                })
}
// }}}


// INTERSECT// {{{
func intersect (stw *Window) {
    getnelems(stw, 2, func (size int) {
                          els := make([]*st.Elem, 2)
                          num := 0
                          for _, el := range stw.SelectElem {
                              if el != nil {
                                  els[num] = el
                                  num++
                                  if num >= 2 { break }
                              }
                          }
                          if num == 2 {
                              _, els, err := stw.Frame.Intersect(els[0], els[1], true, 1, 1, false, false)
                              if err != nil {
                                  stw.addHistory(err.Error())
                                  stw.Deselect()
                              } else {
                                  stw.Deselect()
                                  switch len(els) {
                                  case 4:
                                      stw.SelectElem = []*st.Elem{els[1], els[3]}
                                  }
                                  stw.Redraw()
                              }
                              stw.EscapeCB()
                          }
                      })
}
// }}}


// INTERSECTALL // {{{
func intersectall (stw *Window) {
    l := len(stw.SelectElem)
    if stw.SelectElem == nil || l <= 1 { stw.EscapeAll(); return }
    checked := make([]*st.Elem, 1)
    sort.Sort(st.ElemByNum{stw.SelectElem})
    ind := 0
    for {
        if stw.SelectElem[ind].IsLineElem() {
            _, els, err := stw.SelectElem[ind].DivideAtOns(1e-4)
            if err != nil {
                stw.EscapeAll()
                return
            }
            checked = els
            break
        }
        ind++
        if ind >= l-1 { stw.EscapeAll(); return }
    }
    for _, el := range stw.SelectElem[ind+1:] {
        if !el.IsLineElem() { continue }
        for _, ce := range checked {
            _, els, err := stw.Frame.CutByElem(el, ce, true, 1, false)
            if err != nil {
                continue
            }
            checked = append(checked, els[1])
        }
        _, els, err := el.DivideAtOns(1e-4)
        if err != nil {
            continue
        }
        checked = append(checked, els...)
    }
    stw.EscapeAll()
}
// }}}


// Bounded Area// {{{
func (stw *Window) BoundedArea (arg *iup.MouseButton, f func(ns []*st.Node, els []*st.Elem)) {
    stw.dbuff.UpdateYAxis(&arg.Y)
    if arg.Pressed == 0 { // Released
        var cand *st.Elem
        xmin := 100000.0
        for _, el := range(stw.Frame.Elems) {
            if el.IsHide(stw.Frame.Show) || !el.IsLineElem() { continue }
            if el.Enod[0].Pcoord[1] == el.Enod[1].Pcoord[1] { continue }
            if (el.Enod[0].Pcoord[1] - float64(arg.Y)) * (el.Enod[1].Pcoord[1] - float64(arg.Y)) < 0 {
                xval := el.Enod[0].Pcoord[0] + (el.Enod[1].Pcoord[0]-el.Enod[0].Pcoord[0])*((float64(arg.Y)-el.Enod[0].Pcoord[1])/(el.Enod[1].Pcoord[1]-el.Enod[0].Pcoord[1])) - float64(arg.X)
                if xval > 0 && xval < xmin {
                    cand = el
                    xmin = xval
                }
            }
        }
        if cand != nil {
            ns, els, err := stw.Chain(float64(arg.X), float64(arg.Y), cand, 100)
            if err == nil {
                f(ns, els)
            } else {
                fmt.Println(err)
            }
            stw.Redraw()
        }
    }
}

func ClockWise (p1, p2, p3 []float64) (float64, bool) {
    v1 := []float64{p2[0]-p1[0], p2[1]-p1[1]}
    v2 := []float64{p3[0]-p1[0], p3[1]-p1[1]}
    var sum1, sum2 float64
    for i:=0; i<2; i++ {
        sum1 += v1[i]*v1[i]; sum2 += v2[i]*v2[i]
    }
    if sum1 == 0 || sum2 == 0 { return 0.0, false }
    sum1 = math.Sqrt(sum1); sum2 = math.Sqrt(sum2)
    for i:=0; i<2; i++ {
        v1[i] /= sum1; v2[i] /= sum2
    }
    if val := v2[0]*v1[1]-v2[1]*v1[0]; val > 0 {
        return math.Atan2(val, v1[0]*v2[0]+v1[1]*v2[1]), true
    } else {
        return math.Atan2(val, v1[0]*v2[0]+v1[1]*v2[1]), false
    }
}

func (stw *Window) Chain (x, y float64, el *st.Elem, maxdepth int) ([]*st.Node, []*st.Elem, error) {
    rtnns := make([]*st.Node, 2)
    rtnels := make([]*st.Elem, 1)
    rtnns[0] = el.Enod[0]; rtnns[1] = el.Enod[1]
    rtnels[0] = el
    origin := []float64{x, y}
    _, cw := ClockWise(origin, el.Enod[0].Pcoord, el.Enod[1].Pcoord)
    tmpel := el
    next := el.Enod[1]
    depth := 1
    for {
        minangle := 10000.0
        var addelem *st.Elem
        for _, cand := range stw.Frame.SearchElem(next) {
            if cand == nil { continue }
            if cand.IsHide(stw.Frame.Show) || !cand.IsLineElem() || cand == tmpel { continue }
            var otherside *st.Node
            for _, en := range cand.Enod {
                if en != next { otherside = en; break }
            }
            angle, tmpcw := ClockWise(next.Pcoord, origin, otherside.Pcoord)
            angle = math.Abs(angle)
            if cw != tmpcw && angle < minangle {
                addelem = cand
                minangle = angle
            }
        }
        if addelem == nil { return nil, nil, errors.New("Chain: Not Bounded") }
        for _, en := range addelem.Enod {
            if en == el.Enod[0] {
                rtnels = append(rtnels, addelem)
                return rtnns[:depth+1], rtnels[:depth+1], nil
            }
            if en != next { next = en; break }
        }
        rtnns = append(rtnns, next)
        rtnels = append(rtnels, addelem)
        tmpel = addelem
        depth++
        if depth > maxdepth { return rtnns[:depth], rtnels[:depth], errors.New("Chain: Too Much Recursion") }
    }
}
// }}}


// divideenods// {{{
func divideenods (ns []*st.Node, maxlen int) [][]*st.Node {
    if len(ns) <= maxlen { return [][]*st.Node{ns} }
    rtn := make([][]*st.Node, 0)
    num := 0
    seek := 0
    tmp := make([]*st.Node, maxlen)
    for {
        var ind int
        for i, n := range ns[seek:] {
            ind = i%maxlen
            tmp[ind] = n
            if ind+1 == maxlen {
                rtn = append(rtn, tmp)
                tmp = make([]*st.Node, maxlen)
                num++
                seek = ind
                break
            }
        }
        if ind+1 < maxlen {
            tmp[ind+1] = ns[0]
            rtn = append(rtn, tmp[:ind+2])
            num++
            break
        }
    }
    return rtn[:num]
}
// }}}


// HATCHPLATEELEM// {{{
func hatchplateelem (stw *Window) {
    stw.canv.SetAttribute("CURSOR", "PEN")
    createhatch := func(ns []*st.Node, els []*st.Elem) {
        en := st.ModifyEnod(ns)
        en = st.Upside(en)
        sec := stw.Frame.DefaultSect()
        switch len(en) {
        case 0,1,2:
            return
        case 3, 4:
            if len(stw.Frame.SearchElem(en...)) == 0 {
                el := stw.Frame.AddPlateElem(-1, en, sec, st.NONE)
                var buf bytes.Buffer
                buf.WriteString(fmt.Sprintf("ELEM: %d (ENOD: ", el.Num))
                for _, n := range en {
                    buf.WriteString(fmt.Sprintf("%d ", n.Num))
                }
                buf.WriteString(fmt.Sprintf(", SECT: %d)", sec.Num))
                stw.addHistory(buf.String())
            } else {
                var buf bytes.Buffer
                buf.WriteString("ELEM already exists: ")
                for _, n := range en {
                    buf.WriteString(fmt.Sprintf("%d ", n.Num))
                }
                stw.addHistory(buf.String())
            }
        default:
            ens := divideenods(en, 4)
            for _, eni := range ens {
                if len(stw.Frame.SearchElem(eni...)) == 0 {
                    el := stw.Frame.AddPlateElem(-1, eni, sec, st.NONE)
                    var buf bytes.Buffer
                    buf.WriteString(fmt.Sprintf("ELEM: %d (ENOD: ", el.Num))
                    for _, n := range eni {
                        buf.WriteString(fmt.Sprintf("%d ", n.Num))
                    }
                    buf.WriteString(fmt.Sprintf(", SECT: %d)", sec.Num))
                    stw.addHistory(buf.String())
                } else {
                    var buf bytes.Buffer
                    buf.WriteString("ELEM already exists: ")
                    for _, n := range en {
                        buf.WriteString(fmt.Sprintf("%d ", n.Num))
                    }
                    stw.addHistory(buf.String())
                }
            }
        }
    }
    stw.canv.SetCallback( func (arg *iup.MouseButton) {
                              if stw.Frame != nil {
                                  switch arg.Button {
                                  case BUTTON_LEFT:
                                      stw.BoundedArea(arg, createhatch)
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
                                              stw.startX = int(arg.X); stw.startY = int(arg.Y)
                                          }
                                      }
                                  case BUTTON_RIGHT:
                                      if arg.Pressed == 0 {
                                          stw.EscapeAll()
                                      }
                                  }
                              }
                          })
    stw.canv.SetCallback( func (arg *iup.MouseMotion) {
                              if stw.Frame != nil {
                                  stw.dbuff.UpdateYAxis(&arg.Y)
                                  switch statusKey(arg.Status) {
                                  case STATUS_CENTER:
                                      if isShift(arg.Status) {
                                          stw.Frame.View.Center[0] += float64(int(arg.X) - stw.startX) * CanvasMoveSpeedX
                                          stw.Frame.View.Center[1] += float64(int(arg.Y) - stw.startY) * CanvasMoveSpeedY
                                      } else {
                                          stw.Frame.View.Angle[0] -= float64(int(arg.Y) - stw.startY) * CanvasRotateSpeedY
                                          stw.Frame.View.Angle[1] -= float64(int(arg.X) - stw.startX) * CanvasRotateSpeedX
                                      }
                                      stw.DrawFrameNode()
                                  }
                              }
                          })
}
// }}}


// ADDPLATEALL TODO: UNDER CONSTRUCTION// {{{
func addplateall (stw *Window) {
    if stw.SelectElem == nil {
        ns := stw.Frame.ElemToNode(stw.SelectElem...)
        stw.SelectNode = append(stw.SelectNode, ns...)
    }
    if stw.SelectNode == nil || len(stw.SelectNode) < 3 {
        stw.EscapeAll()
        return
    }
}
// }}}


// EDITPLATEELEM// {{{
func editplateelem (stw *Window) {
    if stw.SelectElem == nil || len(stw.SelectElem) == 0 {
        stw.EscapeAll()
        return
    }
    replaceenod := func (n *st.Node) {
        for _, el := range stw.SelectElem {
            for i, en := range el.Enod {
                if en == stw.SelectNode[0] {
                    el.Enod[i] = n
                    break
                }
            }
        }
        stw.SelectNode = make([]*st.Node, 2)
        stw.Redraw()
    }
    prune := func () {
        for _, el := range stw.SelectElem {
            var ens int
            tmp := make([]*st.Node, el.Enods)
            prnloop:
                for _, en := range el.Enod {
                    for _, n := range stw.SelectNode {
                        if en == n {
                            continue prnloop
                        }
                    }
                    tmp[ens] = en
                    ens++
                }
            el.Enod = tmp[:ens]
            el.Enods = ens
        }
        stw.SelectNode = make([]*st.Node, 2)
        stw.Redraw()
    }
    get2nodes(stw, replaceenod, prune)
}
// }}}


// REACTION// {{{
func reaction (stw *Window) {
    tmp, err := stw.Query("方向を指定[0～5]")
    if err != nil {
        stw.addHistory(err.Error())
        stw.EscapeCB()
    }
    val, err := strconv.ParseInt(tmp, 10, 64)
    if err != nil {
        stw.addHistory(err.Error())
        stw.EscapeCB()
    }
    d := int(val)
    var otp bytes.Buffer
    selectconfed(stw)
    sort.Sort(st.NodeByNum{stw.SelectNode})
    r := make([]float64, 3)
    otp.WriteString("NODE   XCOORD   YCOORD   ZCOORD     WEIGHT       LONG      XSEIS      YSEIS        W+L      W+L+X      W+L-X      W+L+Y      W+L-Y\n")
    for _, n := range stw.SelectNode {
        if n == nil { continue }
        otp.WriteString(fmt.Sprintf("%4d %8.3f %8.3f %8.3f", n.Num, n.Coord[0], n.Coord[1], n.Coord[2]))
        wgt := n.Weight[1]
        for i, per := range []string{"L", "X", "Y"} {
            if rea, ok := n.Reaction[per]; ok {
                r[i] = rea[d]
            } else {
                r[i] = 0.0
            }
        }
        otp.WriteString(fmt.Sprintf(" %10.3f %10.3f %10.3f %10.3f %10.3f %10.3f %10.3f %10.3f %10.3f\n", wgt, r[0], r[1], r[2], wgt+r[0], wgt+r[0]+r[1], wgt+r[0]-r[1], wgt+r[0]+r[2], wgt+r[0]-r[2]))
    }
    fn := st.Ce(stw.Frame.Path, ".rct")
    w, err := os.Create(fn)
    defer w.Close()
    if err != nil {
        stw.addHistory(err.Error())
        stw.EscapeCB()
    }
    otp.WriteTo(w)
    stw.addHistory(fmt.Sprintf("OUTPUT: %s", fn))
    stw.EscapeCB()
}
// }}}


// SUMREACTION// {{{
func sumreaction (stw *Window) {
    if stw.SelectNode == nil || len(stw.SelectNode) == 0 {
        stw.EscapeAll()
    }
    showval := func() {
        val := make([]float64, 5)
        for _, n := range stw.SelectNode {
            if n == nil { continue }
            for i:=0; i<5; i++ {
                val[i] += n.Weight[1]
            }
            if rea, ok := n.Reaction["L"]; ok {
                for i:=0; i<5; i++ {
                    val[i] += rea[2]
                }
            }
            if rea, ok := n.Reaction["X"]; ok {
                val[1] += rea[2]
                val[2] -= rea[2]
            }
            if rea, ok := n.Reaction["Y"]; ok {
                val[3] += rea[2]
                val[4] -= rea[2]
            }
        }
        var result bytes.Buffer
        for i, name := range []string{ "  ", "+X", "-X", "+Y", "-Y" } {
            if i != 0 { result.WriteString(" ") }
            result.WriteString(fmt.Sprintf("W+L%s: %.3f", name, val[i]))
        }
        stw.addHistory(result.String())
    }
    stw.canv.SetCallback( func (arg *iup.MouseButton) {
                              if stw.Frame != nil {
                                  stw.dbuff.UpdateYAxis(&arg.Y)
                                  switch arg.Button {
                                  case BUTTON_CENTER:
                                      if arg.Pressed == 0 { // Released
                                          // if stw.SelectNode[0] != nil { // TODO
                                          //     stw.cdcanv.Line(int(stw.SelectNode[0].Pcoord[0]), int(stw.SelectNode[0].Pcoord[1]), stw.endX, stw.endY)
                                          // }
                                          stw.Redraw()
                                      } else { // Pressed
                                          if isDouble(arg.Status) {
                                              stw.Frame.SetFocus()
                                              stw.DrawFrameNode()
                                              stw.ShowCenter()
                                          } else {
                                              stw.startX = int(arg.X); stw.startY = int(arg.Y)
                                          }
                                      }
                                  case BUTTON_RIGHT:
                                      if arg.Pressed == 0 {
                                          showval()
                                          stw.EscapeAll()
                                      }
                                  }
                              }
                          })
    stw.canv.SetCallback( func (arg *iup.CommonKeyAny) {
                              key := iup.KeyState(arg.Key)
                              switch key.Key() {
                              default:
                                  stw.DefaultKeyAny(key)
                                  // stw.cline.SetAttribute("INSERT", string(key.Key()))
                              case KEY_BS:
                                  val := stw.cline.GetAttribute("VALUE")
                                  if val != "" {
                                      stw.cline.SetAttribute("VALUE", val[:len(val)-1])
                                  }
                              case KEY_ENTER:
                                  if key.IsCtrl() {
                                      showval()
                                  }
                              case KEY_ESCAPE:
                                  stw.EscapeAll()
                              }
                          })
}
// }}}


// UPLIFT// {{{
func uplift (stw *Window) {
    stw.Deselect()
    selectuplift := func (lis []string) {
                        periods := make([]string, 0)
                        for _, p := range lis {
                            p = strings.ToUpper(p)
                            switch p {
                            case "L":
                                periods = append(periods, "L")
                            case "X", "Y":
                                periods = append(periods, fmt.Sprintf("L+%s", p))
                            case "-X", "-Y":
                                periods = append(periods, fmt.Sprintf("L%s", p))
                            }
                        }
                        fmt.Println(periods)
                        num := 0
                        for _, n := range stw.Frame.Nodes {
                            if !n.Conf[2] { continue }
                            for _, p := range periods {
                                val := n.Weight[1] + n.ReturnReaction(p, 2)
                                fmt.Printf("NODE %d VAL %.3f\n", n.Num, val)
                                if val < 0.0 {
                                    stw.SelectNode = append(stw.SelectNode, n)
                                    num++
                                }
                            }
                        }
                        if num == 0 {
                            stw.addHistory("NO UPLIFTS")
                        } else {
                            stw.addHistory(fmt.Sprintf("%d UPLIFTS", num))
                            stw.SelectNode = stw.SelectNode[:num]
                        }
                        stw.EscapeCB()
                    }
    stw.canv.SetCallback( func (arg *iup.CommonKeyAny) {
                              key := iup.KeyState(arg.Key)
                              switch key.Key() {
                              default:
                                  stw.DefaultKeyAny(key)
                              case KEY_BS:
                                  val := stw.cline.GetAttribute("VALUE")
                                  if val != "" {
                                      stw.cline.SetAttribute("VALUE", val[:len(val)-1])
                                  }
                              case KEY_ENTER:
                                  val := stw.cline.GetAttribute("VALUE")
                                  selectuplift(strings.Split(val, " "))
                                  stw.cline.SetAttribute("VALUE", "")
                              case KEY_ESCAPE:
                                  stw.EscapeAll()
                              }
                          })
}
// }}}


// NOTICE1459// {{{
func notice1459 (stw *Window) {
    getnnodes(stw, 3, func (num int) {
                          var delta float64
                          ds := make([]float64, num)
                          for i:=0; i<num; i++ {
                              ds[i] = -stw.SelectNode[i].ReturnDisp(stw.Frame.Show.Period, 2)*100
                          }
                          switch num {
                          default:
                              stw.EscapeAll()
                              return
                          case 2:
                              delta = ds[1] - ds[0]
                              stw.addHistory(fmt.Sprintf("Disp: %.3f - %.3f [cm]", ds[1], ds[0]))
                          case 3:
                              delta = ds[1] - 0.5*(ds[0]+ds[2])
                              stw.addHistory(fmt.Sprintf("Disp: %.3f - (%.3f + %.3f)/2 [cm]", ds[1], ds[0], ds[2]))
                          }
                          var length float64
                          for i:=0; i<2; i++ {
                              length += math.Pow(stw.SelectNode[2].Coord[i] - stw.SelectNode[0].Coord[i], 2)
                          }
                          length = math.Sqrt(length)*100
                          if delta != 0.0 {
                              stw.addHistory(fmt.Sprintf("Distance: %.3f[cm]", length))
                              stw.addHistory(fmt.Sprintf("Slope: 1/%.1f",math.Abs(length/delta)))
                          }
                          stw.EscapeAll()
                      })
}
// }}}


// CATBYNODE// {{{
func catbynode (stw *Window) {
    get1node(stw, func (n *st.Node) {
                      err :=stw.Frame.CatByNode(n, true)
                      if err != nil {
                          stw.addHistory(err.Error())
                      } else {
                          stw.Redraw()
                      }
                  })
}
// }}}


// CATINTERMEDIATENODE TODO: TEST // {{{
func catintermediatenode (stw *Window) {
    stw.Deselect()
    ns := make([]*st.Node, 0)
    nnum := 0
    for _, n := range stw.Frame.Nodes {
        els := stw.Frame.SearchElem(n)
        l := len(els)
        if l < 2 { continue }
        ind := 0
        lels := make([]*st.Elem, l)
        for _, el := range els {
            if el.IsLineElem() {
                lels[ind] = el
                ind++
            }
        }
        d := lels[0].Direction(false)
        sec := lels[0].Sect
        ci := true
        for _, el := range lels[1:] {
            if el.Sect != sec {
                ci = false
                break
            }
            if !el.IsParallel(d, 1e-4) {
                ci = false
                break
            }
        }
        if ci {
            ns = append(ns, n)
            nnum++
        }
    }
    if nnum > 0 {
        stw.SelectNode = ns[:nnum]
        stw.Redraw()
        if stw.Yn("CAT INTERMEDIATE NODE", "中間節点を除去しますか？") {
            for _, n := range stw.SelectNode {
                stw.Frame.CatByNode(n, true)
            }
        }
    }
    stw.EscapeAll()
}
// }}}


// GETNELEMS// {{{
func getnelems (stw *Window, size int, f func(int)) {
    selected := 0
    for _, el := range stw.SelectElem {
        if el != nil { selected++ }
    }
    if selected >= size {
        f(selected)
        return
    }
    setenum := func () {
        enum, err := strconv.ParseInt(stw.cline.GetAttribute("VALUE"),10,64)
        if err == nil {
            if el, ok := stw.Frame.Elems[int(enum)]; ok {
                stw.SelectElem = append(stw.SelectElem, el)
            }
        }
        stw.cline.SetAttribute("VALUE", "")
    }
    stw.canv.SetCallback( func (arg *iup.MouseButton) {
                              if stw.Frame != nil {
                                  switch arg.Button {
                                  case BUTTON_LEFT:
                                      stw.SelectElemStart(arg)
                                  case BUTTON_CENTER:
                                      stw.dbuff.UpdateYAxis(&arg.Y)
                                      if arg.Pressed == 0 { // Released
                                          // if stw.SelectNode[0] != nil { // TODO
                                          //     stw.cdcanv.Line(int(stw.SelectNode[0].Pcoord[0]), int(stw.SelectNode[0].Pcoord[1]), stw.endX, stw.endY)
                                          // }
                                          stw.Redraw()
                                      } else { // Pressed
                                          if isDouble(arg.Status) {
                                              stw.Frame.SetFocus()
                                              stw.DrawFrameNode()
                                              stw.ShowCenter()
                                          } else {
                                              stw.startX = int(arg.X); stw.startY = int(arg.Y)
                                          }
                                      }
                                  case BUTTON_RIGHT:
                                      if arg.Pressed == 0 {
                                          if v := stw.cline.GetAttribute("VALUE"); v!="" {
                                              setenum()
                                          } else {
                                              // fmt.Println("CONTEXT MENU")
                                              f(len(stw.SelectElem))
                                          }
                                      }
                                  }
                              }
                          })
    stw.canv.SetCallback( func (arg *iup.CommonKeyAny) {
                              key := iup.KeyState(arg.Key)
                              switch key.Key() {
                              default:
                                  stw.DefaultKeyAny(key)
                              case KEY_BS:
                                  val := stw.cline.GetAttribute("VALUE")
                                  if val != "" {
                                      stw.cline.SetAttribute("VALUE", val[:len(val)-1])
                                  }
                              case KEY_ENTER:
                                  setenum()
                              case KEY_ESCAPE:
                                  stw.EscapeAll()
                              }
                          })
}
// }}}


// JOINLINEELEM// {{{
func joinlineelem (stw *Window) {
    getnelems(stw, 2, func (size int) {
                          els := make([]*st.Elem, 2)
                          num := 0
                          for _, el := range stw.SelectElem {
                              if el != nil && el.IsLineElem() {
                                  els[num] = el
                                  num++
                                  if num >= 2 { break }
                              }
                          }
                          if num == 2 {
                              err := stw.Frame.JoinLineElem(els[0], els[1], true)
                              if err != nil {
                                  switch err.(type) {
                                  default:
                                      stw.addHistory(err.Error())
                                  case st.ParallelError:
                                      if stw.Yn("JOIN LINE ELEM", "平行でない部材を結合しますか") {
                                          err := stw.Frame.JoinLineElem(els[0], els[1], false)
                                          if err != nil {
                                              stw.addHistory(err.Error())
                                          }
                                      }
                                  }
                              }
                              stw.EscapeAll()
                          }
                      })
}
// }}}


// JOINPLATEELEM// {{{
func joinplateelem (stw *Window) {
    getnelems(stw, 2, func (size int) {
                          els := make([]*st.Elem, 2)
                          num := 0
                          for _, el := range stw.SelectElem {
                              if el != nil && !el.IsLineElem() {
                                  els[num] = el
                                  num++
                                  if num >= 2 { break }
                              }
                          }
                          if num == 2 {
                              err := stw.Frame.JoinPlateElem(els[0], els[1])
                              if err != nil {
                                  stw.addHistory(err.Error())
                              } else {
                                  stw.EscapeAll()
                              }
                          }
                      })
}
// }}}


// MERGENODE// {{{
func mergenode (stw *Window) {
    merge := func (ns []*st.Node) {
                 c := make([]float64, 3)
                 num := 0
                 for _, n := range ns {
                     if n == nil { continue }
                     for i:=0; i<3; i++ {
                         c[i] += n.Coord[i]
                     }
                     num++
                 }
                 if num > 0 {
                     for i:=0; i<3; i++ {
                         c[i]/=float64(num)
                     }
                     var del bool
                     delmap := make(map[*st.Node]*st.Node)
                     var n0 *st.Node
                     for _, n := range ns {
                         if n == nil { continue }
                         if del {
                             delmap[n] = n0
                         } else {
                             n.Coord = c
                             n0 = n
                             del = true
                         }
                     }
                     stw.Frame.ReplaceNode(delmap)
                 }
             }
    if stw.SelectNode != nil {
        merge(stw.SelectNode)
        stw.EscapeAll()
        return
    }
    getnnodes(stw, len(stw.Frame.Nodes), func (num int) { merge(stw.SelectNode); stw.EscapeAll() })
}
// }}}


// ERASE// {{{
func erase (stw *Window) {
    stw.DeleteSelected()
    stw.Deselect()
    ns := stw.Frame.NodeNoReference()
    if len(ns) != 0 {
        for _, n := range ns {
            delete(stw.Frame.Nodes, n.Num)
        }
    }
    stw.EscapeAll()
}
// }}}


// FACTS// {{{
func facts (stw *Window) {
    fn := st.Ce(stw.Frame.Path, ".fes")
    err := stw.Frame.Facts(fn, []int{st.COLUMN, st.GIRDER, st.BRACE, st.WBRACE, st.SBRACE})
    if err != nil {
        stw.addHistory(err.Error())
    } else {
        stw.addHistory(fmt.Sprintf("Output: %s", fn))
    }
    stw.EscapeAll()
}
// }}}


// ZOUBUNDISP// {{{
func zoubundisp (stw *Window) {
    if stw.SelectNode == nil || len(stw.SelectNode) == 0 { stw.EscapeAll(); return }
    pers, err := stw.QueryList("PERIODを指定")
    if err != nil {
        stw.addHistory(err.Error())
        stw.EscapeCB()
    }
    tmp, err := stw.Query("方向を指定[0～5]")
    if err != nil {
        stw.addHistory(err.Error())
        stw.EscapeCB()
    }
    val, err := strconv.ParseInt(tmp, 10, 64)
    if err != nil {
        stw.addHistory(err.Error())
        stw.EscapeCB()
    }
    d := int(val)
    var otp bytes.Buffer
    sort.Sort(st.NodeByZCoord{stw.SelectNode})
    for _, per := range pers {
        per = strings.ToUpper(per)
        if nlap, ok := stw.Frame.Nlap[per]; ok {
            otp.WriteString(fmt.Sprintf("ZOUBUN DISP: %s\n", stw.Frame.Name))
            otp.WriteString(fmt.Sprintf("PERIOD: %s, DIRECTION: %d\n", per, d))
            otp.WriteString("LAP     ")
            for _, n := range stw.SelectNode {
                otp.WriteString(fmt.Sprintf(" %8d", n.Num))
            }
            otp.WriteString("\n")
            for i:=0; i<nlap; i++ {
                nper := fmt.Sprintf("%s@%d", per, i+1)
                otp.WriteString(fmt.Sprintf("%8s", nper))
                for _, n := range stw.SelectNode {
                    otp.WriteString(fmt.Sprintf(" %8.5f", n.Disp[nper][d]))
                }
                otp.WriteString("\n")
            }
            otp.WriteString("\n")
        }
    }
    fn := filepath.Join(filepath.Dir(stw.Frame.Path), "zoubunout.txt")
    w, err := os.Create(fn)
    defer w.Close()
    if err != nil {
        stw.addHistory(err.Error())
        stw.EscapeCB()
    }
    otp.WriteTo(w)
    stw.addHistory(fmt.Sprintf("OUTPUT: %s", fn))
    stw.EscapeCB()
}
// }}}


// ZOUBUNYIELD// {{{
func zoubunyield (stw *Window) {
    var otp bytes.Buffer
    var skeys []int
    for k := range(stw.Frame.Sects) {
        skeys = append(skeys, k)
    }
    sort.Ints(skeys)
    otp.WriteString("種別　断面番号　  　　　 Ｎ[tf]　Ｑx[tf]　 Ｑy[tf]　Ｍz[tfm]　Ｍx[tfm]　Ｍy[tfm]\n")
    for _, k := range(skeys) {
        sec := stw.Frame.Sects[k]
        if sec.Num < 700 {
            otp.WriteString(fmt.Sprintf("         %4d     最大   ", sec.Num))
            for i:=0; i<6; i++ {
                otp.WriteString(fmt.Sprintf(" %8.1f", sec.Yield[2*i]))
            }
            otp.WriteString("\n")
            otp.WriteString("                  最小   ")
            for i:=0; i<6; i++ {
                otp.WriteString(fmt.Sprintf(" %8.1f", sec.Yield[2*i+1]))
            }
            otp.WriteString("\n")
        } else if sec.Num > 900 {
            otp.WriteString(fmt.Sprintf("         %4d     最大   ", sec.Num))
            otp.WriteString(fmt.Sprintf(" %8.1f", sec.Yield[0]))
            otp.WriteString("\n")
            otp.WriteString(fmt.Sprintf("                  最小   ", sec.Num))
            otp.WriteString(fmt.Sprintf(" %8.1f", sec.Yield[1]))
            otp.WriteString("\n")
        }
    }
    fn := filepath.Join(filepath.Dir(stw.Frame.Path), "zoubunyield.txt")
    w, err := os.Create(fn)
    defer w.Close()
    if err != nil {
        stw.addHistory(err.Error())
        stw.EscapeCB()
    }
    otp.WriteTo(w)
    stw.addHistory(fmt.Sprintf("OUTPUT: %s", fn))
    stw.EscapeAll()
}
// }}}


// ZOUBUNREACTION// {{{
func zoubunreaction (stw *Window) {
    if stw.SelectNode == nil || len(stw.SelectNode) == 0 { stw.EscapeAll(); return }
    pers, err := stw.QueryList("PERIODを指定")
    if err != nil {
        stw.addHistory(err.Error())
        stw.EscapeCB()
    }
    tmp, err := stw.Query("方向を指定[0～5]")
    if err != nil {
        stw.addHistory(err.Error())
        stw.EscapeCB()
    }
    val, err := strconv.ParseInt(tmp, 10, 64)
    if err != nil {
        stw.addHistory(err.Error())
        stw.EscapeCB()
    }
    d := int(val)
    var otp bytes.Buffer
    sort.Sort(st.NodeByNum{stw.SelectNode})
    for _, per := range pers {
        per = strings.ToUpper(per)
        if nlap, ok := stw.Frame.Nlap[per]; ok {
            otp.WriteString(fmt.Sprintf("ZOUBUN REACTION: %s\n", stw.Frame.Name))
            otp.WriteString(fmt.Sprintf("PERIOD: %s, DIRECTION: %d\n", per, d))
            otp.WriteString("LAP     ")
            for _, n := range stw.SelectNode {
                otp.WriteString(fmt.Sprintf(" %8d", n.Num))
            }
            otp.WriteString("\n")
            for i:=0; i<nlap; i++ {
                nper := fmt.Sprintf("%s@%d", per, i+1)
                otp.WriteString(fmt.Sprintf("%8s", nper))
                for _, n := range stw.SelectNode {
                    otp.WriteString(fmt.Sprintf(" %8.3f", n.Reaction[nper][d]))
                }
                otp.WriteString("\n")
            }
            otp.WriteString("\n")
        }
    }
    fn := filepath.Join(filepath.Dir(stw.Frame.Path), "zoubunout.txt")
    w, err := os.Create(fn)
    defer w.Close()
    if err != nil {
        stw.addHistory(err.Error())
        stw.EscapeCB()
    }
    otp.WriteTo(w)
    stw.addHistory(fmt.Sprintf("OUTPUT: %s", fn))
    stw.EscapeCB()
}
// }}}


// AMOUNTPROP// {{{
func amountprop (stw *Window) {
    tmp, err := stw.QueryList("PROP")
    if err != nil {
        stw.EscapeAll()
        return
    }
    l := len(tmp)
    props := make([]int, l)
    for i:=0; i<l; i++ {
        val, err := strconv.ParseInt(tmp[i], 10, 64)
        if err != nil {
            stw.EscapeAll()
            return
        }
        props[i] = int(val)
    }
    total := 0.0
    sects := make([]*st.Sect, len(stw.Frame.Sects))
    snum := 0
    for _, sec := range stw.Frame.Sects {
        if sec.Num > 900 { continue }
        sects[snum] = sec
        snum++
    }
    sects = sects[:snum]
    sort.Sort(st.SectByNum{sects})
    var otp bytes.Buffer
    otp.WriteString("断面 名前                                     長さ     断面積   単位重量 重量\n")
    otp.WriteString("                                              面積     板厚\n")
    for _, sec := range sects {
        size := sec.PropSize(props)
        if size == 0.0 { continue }
        amount := sec.TotalAmount()
        weight := sec.PropWeight(props)
        totalweight := amount * weight
        otp.WriteString(fmt.Sprintf("%4d %-40s %8.3f %8.4f %8.4f %8.3f\n", sec.Num, sec.Name, amount, size, weight, totalweight))
        total += totalweight
    }
    otp.WriteString(fmt.Sprintf("%79s\n", fmt.Sprintf("合計: %8.3f", total)))
    fn := filepath.Join(filepath.Dir(stw.Frame.Path), "amount.txt")
    w, err := os.Create(fn)
    defer w.Close()
    if err != nil {
        stw.addHistory(err.Error())
        stw.EscapeCB()
    }
    otp.WriteTo(w)
    stw.addHistory(fmt.Sprintf("OUTPUT: %s", fn))
    stw.EscapeAll()
}
// }}}
