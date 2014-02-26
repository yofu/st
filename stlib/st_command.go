package st

// TODO: changeconf, copyconf

import (
    "bytes"
    "errors"
    "fmt"
    "strconv"
    "github.com/visualfc/go-iup/iup"
    "github.com/visualfc/go-iup/cd"
    "math"
    "strings"
)

var (
    Commands = make(map[string]*Command,0)

    DISTS = &Command{"DISTS", "measure distance", dists}
    TOGGLEBOND = &Command{"TOGGLE BOND", "toggle bond of selected elem", togglebond}
    COPYBOND = &Command{"COPY BOND", "copy bond of selected elem", copybond}
    BONDPIN = &Command{"BOND PIN", "set bond of selected elem to pin-pin", bondpin}
    BONDRIGID = &Command{"BOND RIGID", "set bond of selected elem to rigid-rigid", bondrigid}
    CONFFIX = &Command{"CONF FIX", "set conf of selected node to fix", conffix}
    CONFPIN = &Command{"CONF PIN", "set conf of selected node to pin", confpin}
    CONFFREE = &Command{"CONF FREE", "set conf of selected node to free", conffree}
    OPEN  = &Command{"OEPN INPUT", "open inp file", openinput}
    INSERT = &Command{"INSERT", "insert new frame", insert}
    SETFOCUS = &Command{"SET FOCUS", "set focus to the center", setfocus}
    SELECTNODE = &Command{"SELECT NODE", "select elem by number", selectnode}
    SELECTELEM = &Command{"SELECT ELEM", "select elem by number", selectelem}
    SELECTSECT = &Command{"SELECT SECT", "select elem by section", selectsect}
    ERRORELEM = &Command{"ERROR ELEM", "select elem whose max(rate)>1.0", errorelem}
    FENCE = &Command{"FENCE", "select elem by fence", fence}
    ADDLINEELEM = &Command{"ADD LINE ELEM", "add line elem", addlineelem}
    ADDPLATEELEM = &Command{"ADD PLATE ELEM", "add plate elem", addplateelem}
    ADDPLATEELEMBYLINE = &Command{"ADD PLATE ELEM BY LINE", "add plate elem by line", addplateelembyline}
    HATCHPLATEELEM = &Command{"HATCH PLATE ELEM", "add plate elem by hatching", hatchplateelem}
    EDITPLATEELEM = &Command{"EDIT PLATE ELEM", "edit plate elem", editplateelem}
    MATCHPROP = &Command{"MATCH PROPERTY", "match property", matchproperty}
    AXISTOCANG = &Command{"AXISTOCANG", "set cang by axis", axistocang}
    COPYELEM = &Command{"COPY ELEM", "copy selected elems", copyelem}
    MOVEELEM = &Command{"MOVE ELEM", "move selected elems", moveelem}
    MOVENODE = &Command{"MOVE NODE", "move selected nodes", movenode}
    ROTATE = &Command{"ROTATE", "rotate selected nodes", rotate}
    MIRROR = &Command{"MIRROR", "mirror selected elems", mirror}
    SEARCHELEM = &Command{"SEARCH ELEM", "search elems using node", searchelem}
    NODETOELEM = &Command{"NODE TO ELEM", "select elems using selected node", nodetoelemall}
    ELEMTONODE = &Command{"ELEM TO NODE", "select nodes used by selected elem", elemtonode}
    CONNECTED = &Command{"CONNECTED", "select nodes connected to selected node", connected}
    ONNODE = &Command{"ON NODE", "select nodes which is on selected elems", onnode}
    NODENOREFERENCE = &Command{"NODE NO REFERENCE", "delete nodes which are not refered by any elem", nodenoreference}
    ELEMSAMENODE = &Command{"ELEM SAME NODE", "delete elems which has duplicated enod", elemsamenode}
    NODEDUPLICATION = &Command{"NODE DUPLICATION", "delete duplicated nodes", nodeduplication}
    CATBYNODE = &Command{"CAT BY NODE", "join 2 elems using selected node", catbynode}
    JOINLINEELEM = &Command{"JOIN LINE ELEM", "join selected 2 elems", joinlineelem}
    EXTRACTARCLM = &Command{"EXTRACT ARCLM", "extract arclm", extractarclm}
    DIVIDEATONS = &Command{"DIVIDE AT ONS", "divide selected elems at onnode", divideatons}
    DIVIDEATMID = &Command{"DIVIDE AT MID", "divide selected elems at midpoint", divideatmid}
    INTERSECT = &Command{"INTERSECT", "divide selected elems at intersection", intersect}
    INTERSECTALL = &Command{"INTERSECT ALL", "divide selected elems at all intersection", intersectall}
    TRIM = &Command{"TRIM", "trim elements with selected elem", trim}
    EXTEND = &Command{"EXTEND", "extend elements to selected elem", extend}
    ERASE = &Command{"ERASE", "erase selected elems", erase}
    REACTION = &Command{"REACTION", "show sum of reaction", reaction}
    NOTICE1459 = &Command{"NOTICE1459", "shishou", notice1459}
)

func init() {
    Commands["DISTS"]=DISTS
    Commands["TOGGLEBOND"]=TOGGLEBOND
    Commands["COPYBOND"]=COPYBOND
    Commands["BONDPIN"]=BONDPIN
    Commands["BONDRIGID"]=BONDRIGID
    Commands["CONFFIX"]=CONFFIX
    Commands["CONFPIN"]=CONFPIN
    Commands["CONFFREE"]=CONFFREE
    Commands["OPEN"]=OPEN
    Commands["INSERT"]=INSERT
    Commands["SETFOCUS"]=SETFOCUS
    Commands["SELECTNODE"]=SELECTNODE
    Commands["SELECTELEM"]=SELECTELEM
    Commands["SELECTSECT"]=SELECTSECT
    Commands["ERRORELEM"]=ERRORELEM
    Commands["FENCE"]=FENCE
    Commands["ADDLINEELEM"]=ADDLINEELEM
    Commands["ADDPLATEELEM"]=ADDPLATEELEM
    Commands["ADDPLATEELEMBYLINE"]=ADDPLATEELEMBYLINE
    Commands["HATCHPLATEELEM"]=HATCHPLATEELEM
    Commands["EDITPLATEELEM"]=EDITPLATEELEM
    Commands["MATCHPROP"]=MATCHPROP
    Commands["AXISTOCANG"]=AXISTOCANG
    Commands["COPYELEM"]=COPYELEM
    Commands["MOVEELEM"]=MOVEELEM
    Commands["MOVENODE"]=MOVENODE
    Commands["ROTATE"]=ROTATE
    Commands["MIRROR"]=MIRROR
    Commands["SEARCHELEM"]=SEARCHELEM
    Commands["NODETOELEM"]=NODETOELEM
    Commands["ELEMTONODE"]=ELEMTONODE
    Commands["CONNECTED"]=CONNECTED
    Commands["ONNODE"]=ONNODE
    Commands["NODENOREFERENCE"]=NODENOREFERENCE
    Commands["ELEMSAMENODE"]=ELEMSAMENODE
    Commands["NODEDUPLICATION"]=NODEDUPLICATION
    Commands["CATBYNODE"]=CATBYNODE
    Commands["JOINLINEELEM"]=JOINLINEELEM
    Commands["EXTRACTARCLM"]=EXTRACTARCLM
    Commands["DIVIDEATONS"]=DIVIDEATONS
    Commands["DIVIDEATMID"]=DIVIDEATMID
    Commands["INTERSECT"]=INTERSECT
    Commands["INTERSECTALL"]=INTERSECTALL
    Commands["TRIM"]=TRIM
    Commands["EXTEND"]=EXTEND
    Commands["ERASE"]=ERASE
    Commands["REACTION"]=REACTION
    Commands["NOTICE1459"]=NOTICE1459
}

type Command struct {
    Name    string
    command string
    call    func(*Window)
}

func (cmd *Command) Exec(stw *Window) {
    cmd.call(stw)
}


// GET1NODE // {{{
func get1node (stw *Window, f func(n *Node)) {
    stw.canv.SetAttribute("CURSOR", "CROSS")
    stw.SelectNode = make([]*Node, 1)
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
                                          if n := stw.PickNode(arg.X, arg.Y); n != nil {
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
                                              stw.startX = arg.X; stw.startY = arg.Y
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
                                  switch statusKey(arg.Status) {
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
// }}}


// TOGGLEBOND// {{{
func togglebond (stw *Window) {
    get1node(stw, func (n *Node) {
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
func get2nodes (stw *Window, f func(n *Node), fdel func()) {
    stw.canv.SetAttribute("CURSOR", "CROSS")
    stw.SelectNode = make([]*Node, 2)
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
                                              if n := stw.PickNode(arg.X, arg.Y); n != nil {
                                                  f(n)
                                              }
                                              // stw.cdcanv.Line(int(stw.SelectNode[0].Pcoord[0]), int(stw.SelectNode[0].Pcoord[1]), stw.endX, stw.endY) // TODO
                                          } else {
                                              if n := stw.PickNode(arg.X, arg.Y); n != nil {
                                                  stw.SelectNode[0] = n
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
                                              stw.startX = arg.X; stw.startY = arg.Y
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
                                  switch statusKey(arg.Status) {
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
                                  if stw.SelectNode[0] != nil {
                                      stw.TailLine(int(stw.SelectNode[0].Pcoord[0]), int(stw.SelectNode[0].Pcoord[1]), arg)
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
                              case KEY_DELETE:
                                  fdel()
                              case KEY_ENTER:
                                  setnnum()
                              case KEY_ESCAPE:
                                  stw.EscapeAll()
                              }
                          })
}
// }}}


// DISTS// {{{
func dists (stw *Window) {
    get2nodes(stw, func (n *Node) {
                       stw.SelectNode[1] = n
                       dx, dy, dz, d := stw.Frame.Distance(stw.SelectNode[0], n)
                       stw.addHistory(fmt.Sprintf("NODE: %d - %d", stw.SelectNode[0].Num, n.Num))
                       stw.addHistory(fmt.Sprintf("DX: %.3f DY: %.3f DZ: %.3f D: %.3f", dx, dy, dz, d))
                       stw.cdcanv.Foreground(cd.CD_WHITE)
                       stw.cdcanv.WriteMode(cd.CD_REPLACE)
                       stw.EscapeAll()
                   },
                   func () {
                       stw.SelectNode = make([]*Node, 2)
                       stw.Redraw()
                   })
}
// }}}


// ADDLINEELEM// {{{
func addlineelem (stw *Window) {
    get2nodes(stw, func (n *Node) {
                       stw.SelectNode[1] = n
                       sec := stw.Frame.DefaultSect()
                       el := stw.Frame.AddLineElem(stw.SelectNode, sec, NONE)
                       stw.addHistory(fmt.Sprintf("ELEM: %d (ENOD: %d - %d, SECT: %d)", el.Num, stw.SelectNode[0].Num, n.Num, sec.Num))
                       stw.cdcanv.Foreground(cd.CD_WHITE)
                       stw.cdcanv.WriteMode(cd.CD_REPLACE)
                       stw.EscapeAll()
                   },
                   func () {
                       stw.SelectNode = make([]*Node, 2)
                       stw.Redraw()
                   })
}
// }}}


// GETNNODES // {{{
// DISTS: TODO: When button is released, tail line remains. When 2nd node is selected in command line, tail line remains.
func getnnodes (stw *Window, maxnum int, f func(int)) {
    stw.canv.SetAttribute("CURSOR", "CROSS")
    stw.SelectNode = make([]*Node, maxnum)
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
                                          if n := stw.PickNode(arg.X, arg.Y); n != nil {
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
                                              stw.startX = arg.X; stw.startY = arg.Y
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
                                  switch statusKey(arg.Status) {
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
                                  stw.SelectNode = make([]*Node, maxnum)
                                  selected++
                              case KEY_ENTER:
                                  setnnum()
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
                                   el := stw.Frame.AddPlateElem(en, sec, NONE)
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
                          els := make([]*Elem, 2)
                          num := 0
                          for _, el := range stw.SelectElem {
                              if el != nil && el.IsLineElem() {
                                  els[num] = el
                                  num++
                                  if num >= 2 { break }
                              }
                          }
                          if num == 2 {
                              ns := make([]*Node, 4)
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
                              el := stw.Frame.AddPlateElem(ns, sec, NONE)
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
    startsearch := func(n *Node) {
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
                                          if n:= stw.PickNode(arg.X, arg.Y); n != nil {
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
                                              stw.startX = arg.X; stw.startY = arg.Y
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
    stw.SelectNode = make([]*Node, 0)
    if stw.SelectElem != nil {
        for _, el := range stw.SelectElem {
            ns := el.OnNode(0)
            stw.SelectNode = append(stw.SelectNode, ns...)
        }
    }
    stw.EscapeCB()
}


// GETCOORD// {{{
func getcoord (stw *Window, f func(x,y,z float64)) {
    stw.canv.SetAttribute("CURSOR", "CROSS")
    var startpoint *Node
    funcbynode := func (n *Node) {
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
                                              if n := stw.PickNode(arg.X, arg.Y); n != nil {
                                                  funcbynode(n)
                                              }
                                              // stw.cdcanv.Line(int(stw.SelectNode[0].Pcoord[0]), int(stw.SelectNode[0].Pcoord[1]), stw.endX, stw.endY) // TODO
                                          } else {
                                              if n := stw.PickNode(arg.X, arg.Y); n != nil {
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
                                              stw.startX = arg.X; stw.startY = arg.Y
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
                                  switch statusKey(arg.Status) {
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
                                  if startpoint != nil {
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
    getcoord(stw, func (x, y, z float64) {
                      if !(x==0.0 && y==0.0 && z==0.0) {
                          for _, el := range stw.SelectElem {
                              if el == nil || el.Hide || el.Lock { continue }
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
    getcoord(stw, func (x, y, z float64) {
                      for _, el := range stw.SelectElem {
                          if el == nil || el.Hide || el.Lock { continue }
                          el.Move(x, y, z)
                      }
                      stw.Redraw()
                  })
}
// }}}


// MOVENODE// {{{
func movenode (stw *Window) {
    ns := make([]*Node, 0)
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
    getcoord(stw, func (x, y, z float64) {
                      for _, n := range ns {
                          if n == nil || n.Hide || n.Lock { continue }
                          n.Move(x, y, z)
                      }
                      stw.EscapeCB()
                  })
}
// }}}


// ROTATE// {{{
func rotate (stw *Window) {
    ns := make([]*Node, 0)
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
    get1node(stw, func (n0 *Node) {
                      stw.addHistory(fmt.Sprintf("NODE: %d", n0.Num))
                      stw.addHistory("回転軸を選択[ダイアログ(D)]")
                      getcoord(stw, func (x, y, z float64) {
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
                                                                                                u = Normalize(u)
                                                                                                v1 := make([]float64, 3)
                                                                                                v2 := make([]float64, 3)
                                                                                                for i:=0; i<3; i++ {
                                                                                                    v1[i] = stw.SelectNode[1].Coord[i] - stw.SelectNode[0].Coord[i]
                                                                                                    v2[i] = stw.SelectNode[2].Coord[i] - stw.SelectNode[0].Coord[i]
                                                                                                }
                                                                                                uv1 := Dot(u, v1, 3)
                                                                                                uv2 := Dot(u, v2, 3)
                                                                                                v12 := Dot(v1, v2, 3)
                                                                                                v11 := Dot(v1, v1, 3)
                                                                                                v22 := Dot(v2, v2, 3)
                                                                                                w := math.Sqrt(v11-uv1*uv1) * math.Sqrt(v22-uv2*uv2)
                                                                                                if w != 0.0 {
                                                                                                    rot(math.Acos((v12-uv1*uv2)/w))
                                                                                                }
                                                                                                stw.SelectNode = make([]*Node, 0)
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
    stw.addHistory("対称面を指定")
    maxnum := 3
    createmirrors := func (coord, vec []float64) {
                         nmap := make(map[int]*Node, len(ns))
                         for _, n := range ns {
                             c := n.MirrorCoord(coord, vec)
                             nmap[n.Num] = stw.Frame.CoordNode(c[0], c[1], c[2])
                             for i:=0; i<6; i++ {
                                 nmap[n.Num].Conf[i] = n.Conf[i]
                             }
                         }
                         for _, el := range stw.SelectElem {
                             newenod := make([]*Node, el.Enods)
                             var add bool
                             for i:=0; i<el.Enods; i++ {
                                 newenod[i] = nmap[el.Enod[i].Num]
                                 if !add && newenod[i] != el.Enod[i] { add = true }
                             }
                             if add {
                                 if el.IsLineElem() {
                                     e := stw.Frame.AddLineElem(newenod, el.Sect, el.Etype)
                                     for i:=0; i<6*el.Enods; i++ {
                                         e.Bonds[i] = el.Bonds[i]
                                     }
                                 } else {
                                     stw.Frame.AddPlateElem(newenod, el.Sect, el.Etype)
                                 }
                             }
                         }
                     }
    getnnodes(stw, maxnum, func (num int) {
                               switch num {
                               default:
                                   stw.EscapeAll()
                               case 1:
                                   stw.addHistory("法線方向を選択")
                                   getcoord(stw, func (x, y, z float64) {
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
                                   vec := Cross(v1, v2)
                                   createmirrors(stw.SelectNode[0].Coord, vec)
                                   stw.EscapeAll()
                               }
                           })
}
// }}}



// OPEN
func openinput(stw *Window) {
    stw.Open()
    stw.EscapeAll()
}

func insert(stw *Window) {
    if name,ok := iup.GetOpenFile("",""); ok {
        get1node(stw, func (n *Node) {
                              err := stw.Frame.ReadInp(name, n.Coord)
                              if err != nil {
                                  stw.addHistory(err.Error())
                              }
                              stw.EscapeAll()
                          })
    } else {
        stw.EscapeAll()
    }
}


// SETFOCUS
func setfocus(stw *Window) {
    stw.Frame.SetFocus()
    stw.Redraw()
    stw.addHistory(fmt.Sprintf("FOCUS: %.1f %.1f %.1f", stw.Frame.View.Focus[0], stw.Frame.View.Focus[1], stw.Frame.View.Focus[2]))
    stw.EscapeAll()
}


// GET1ELEM// {{{
func get1elem (stw *Window, f func(*Elem, int, int), condition func(*Elem) bool) {
    stw.SelectElem = make([]*Elem, 1)
    selected := false
    stw.canv.SetCallback( func (arg *iup.MouseButton) {
                              if stw.Frame != nil {
                                  stw.dbuff.UpdateYAxis(&arg.Y)
                                  switch arg.Button {
                                  case BUTTON_LEFT:
                                      if arg.Pressed == 0 {
                                          if selected {
                                              if el := stw.PickElem(arg.X, arg.Y); el != nil {
                                                  f(el, arg.X, arg.Y)
                                              }
                                          } else {
                                              if el := stw.PickElem(arg.X, arg.Y); el != nil {
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
                                              stw.startX = arg.X; stw.startY = arg.Y
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
// }}}


// MATCHPROP// {{{
func matchproperty (stw *Window) {
    get1elem(stw, func (el *Elem, x, y int) {
                      el.Sect = stw.SelectElem[0].Sect
                      el.Etype = stw.SelectElem[0].Etype
                      stw.Redraw()
                  },
                  func (el *Elem) bool {
                      return true
                  })
}
// }}}


// COPYBOND// {{{
func copybond (stw *Window) {
    get1elem(stw, func (el *Elem, x, y int) {
                      if el.IsLineElem() {
                          for i:=0; i<12; i++ {
                              el.Bonds[i] = stw.SelectElem[0].Bonds[i]
                          }
                      }
                      stw.Redraw()
                  },
                  func (el *Elem) bool {
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
                              if el == nil || el.Hide || el.Lock { continue }
                              el.AxisToCang([]float64{x, y, z}, strong)
                          }
                          stw.EscapeAll()
                      }
                  }
    getcoord(stw, axisfunc)
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
    stw.EscapeAll()
}

// CONFPIN
func confpin (stw *Window) {
    if stw.SelectNode == nil { stw.EscapeAll(); return }
    for _, n := range stw.SelectNode {
        if n == nil || n.Lock { continue }
        fmt.Println(n.Num)
        for i:=0; i<3; i++ {
            n.Conf[i] = true
            n.Conf[i+3] = false
        }
    }
    stw.EscapeAll()
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
    stw.EscapeAll()
}
// }}}


// TRIM// {{{
func trim (stw *Window) {
    get1elem(stw, func (el *Elem, x, y int) {
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
                  func (el *Elem) bool {
                      return el.IsLineElem()
                  })
}
// }}}


// EXTEND// {{{
func extend (stw *Window) {
    get1elem(stw, func (el *Elem, x, y int) {
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
                  func (el *Elem) bool {
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
                stw.SelectNode = make([]*Node, 1)
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


// SELECTELEM// {{{
func selectelem (stw *Window) {
    stw.Deselect()
    iup.SetFocus(stw.canv)
    setenum := func () {
        enum, err := strconv.ParseInt(stw.cline.GetAttribute("VALUE"),10,64)
        if err == nil {
            if el, ok := stw.Frame.Elems[int(enum)]; ok {
                stw.addHistory(fmt.Sprintf("ELEM %d Selected", enum))
                stw.SelectElem = make([]*Elem, 1)
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
    stw.Show.ElemCaption |= EC_SECT
    stw.Show.Label["EC_SECT"].SetAttribute("FGCOLOR", labelFGColor)
    stw.Deselect()
    stw.Redraw()
    iup.SetFocus(stw.canv)
    setsnum := func () {
        tmp, err := strconv.ParseInt(stw.cline.GetAttribute("VALUE"),10,64)
        if err == nil {
            snum := int(tmp)
            if _, ok := stw.Frame.Sects[snum]; ok {
                stw.addHistory(fmt.Sprintf("SECT %d Selected", snum))
                stw.SelectElem = make([]*Elem, 0)
                for _, el := range stw.Frame.Elems {
                    if el.Hide { continue }
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
}// }}}


// NODENOREFERENCE
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


// ELEMSAMENODE
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


// NODEDUPLICATION
func nodeduplication (stw *Window) {
    stw.Deselect()
    nm := stw.Frame.NodeDuplication(1e-4)
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


// EXTRACTARCLM
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
        for _, ext := range InputExt {
            fn := Ce(name, ext)
            stw.addHistory(fmt.Sprintf("保存しました: %s", fn))
        }
    }
    stw.EscapeAll()
}


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
                                              stw.startX = arg.X; stw.startY = arg.Y
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
// }}}


// ERRORELEM// {{{
func errorelem (stw *Window) {
    iup.SetFocus(stw.canv)
    stw.Deselect()
    stw.Show.SetColorMode(ECOLOR_RATE)
    stw.Show.ElemCaption |= EC_RATE_L
    stw.Show.ElemCaption |= EC_RATE_S
    stw.Show.Label["EC_RATE_L"].SetAttribute("FGCOLOR", labelFGColor)
    stw.Show.Label["EC_RATE_S"].SetAttribute("FGCOLOR", labelFGColor)
    stw.Redraw()
    tmpels := make([]*Elem, len(stw.Frame.Elems))
    i:=0
    for _, el := range(stw.Frame.Elems) {
        if el.Rate != nil {
            for _, val := range(el.Rate) {
                if val > 1.0 { tmpels[i] = el; i++; break }
            }
        }
    }
    stw.SelectElem = make([]*Elem, i)
    for j:=0; j<i; j++ {
        stw.SelectElem[j] = tmpels[j]
    }
}
// }}}


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


// DIVIDE// {{{
func divide (stw *Window, divfunc func(*Elem)([]*Node, []*Elem, error)) {
    if stw.SelectElem != nil {
        tmpels := make([]*Elem, 0)
        for _, el := range stw.SelectElem {
            _, els, err := divfunc(el)
            if err ==  nil && len(els) > 0 {
                tmpels = append(tmpels, els...)
            }
        }
        stw.SelectElem = tmpels
        stw.EscapeCB()
    }
}
func divideatons (stw *Window) {
    divide(stw, func (el *Elem) ([]*Node, []*Elem, error) {
                    return el.DivideAtOns()
                })
}
func divideatmid (stw *Window) {
    divide(stw, func (el *Elem) ([]*Node, []*Elem, error) {
                    return el.DivideAtMid()
                })
}
// }}}


// INTERSECT// {{{
func intersect (stw *Window) {
    getnelems(stw, 2, func (size int) {
                          els := make([]*Elem, 2)
                          num := 0
                          for _, el := range stw.SelectElem {
                              if el != nil {
                                  els[num] = el
                                  num++
                                  if num >= 2 { break }
                              }
                          }
                          if num == 2 {
                              _, _, err := stw.Frame.Intersect(els[0], els[1], true, 1, 1, false, false)
                              fmt.Println(err)
                              if err != nil {
                                  stw.addHistory(err.Error())
                              } else {
                                  stw.Deselect()
                                  stw.Redraw()
                              }
                              stw.EscapeAll()
                          }
                      })
}
// }}}


// INTERSECTALL
func intersectall (stw *Window) {
    if stw.SelectElem == nil || len(stw.SelectElem) <= 1 { return }
    for {
        var next bool
        var tmpels []*Elem
        for i, el := range stw.SelectElem[:len(stw.SelectElem)-1] {
            if el == nil { continue }
            for _, el2 := range stw.SelectElem[i+1:] {
                if el2 == nil { continue }
                ns, els, _ := stw.Frame.Intersect(el, el2, true, 1, 1, false, false)
                if ns != nil {
                    next = true
                    tmpels = append(tmpels, els...)
                }
            }
        }
        stw.MergeSelectElem(tmpels, false)
        if !next { break }
    }
    stw.EscapeCB()
}


// Bounded Area// {{{
func (stw *Window) BoundedArea (arg *iup.MouseButton, f func(ns []*Node, els []*Elem)) {
    stw.dbuff.UpdateYAxis(&arg.Y)
    if arg.Pressed == 0 { // Released
        var cand *Elem
        xmin := 100000.0
        for _, el := range(stw.Frame.Elems) {
            if el.Hide || !el.IsLineElem() { continue }
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

func (stw *Window) Chain (x, y float64, el *Elem, maxdepth int) ([]*Node, []*Elem, error) {
    rtnns := make([]*Node, 2)
    rtnels := make([]*Elem, 1)
    rtnns[0] = el.Enod[0]; rtnns[1] = el.Enod[1]
    rtnels[0] = el
    origin := []float64{x, y}
    _, cw := ClockWise(origin, el.Enod[0].Pcoord, el.Enod[1].Pcoord)
    tmpel := el
    next := el.Enod[1]
    depth := 1
    for {
        minangle := 10000.0
        var addelem *Elem
        fmt.Printf("NODE: %d\n",next.Num)
        for _, cand := range stw.Frame.SearchElem(next) {
            if cand == nil { continue }
            if cand.Hide || !cand.IsLineElem() || cand == tmpel { continue }
            fmt.Printf("ELEM: %d\n",cand.Num)
            var otherside *Node
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


// HATCHPLATEELEM// {{{
func hatchplateelem (stw *Window) {
    stw.canv.SetAttribute("CURSOR", "PEN")
    createhatch := func(ns []*Node, els []*Elem) {
        en := ModifyEnod(ns)
        en = Upside(en)
        sec := stw.Frame.DefaultSect()
        el := stw.Frame.AddPlateElem(en, sec, NONE)
        var buf bytes.Buffer
        buf.WriteString(fmt.Sprintf("ELEM: %d (ENOD: ", el.Num))
        for _, n := range en {
            buf.WriteString(fmt.Sprintf("%d ", n.Num))
        }
        buf.WriteString(fmt.Sprintf(", SECT: %d)", sec.Num))
        stw.addHistory(buf.String())
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
                                              stw.startX = arg.X; stw.startY = arg.Y
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
// }}}


// EDITPLATEELEM// {{{
func editplateelem (stw *Window) {
    if stw.SelectElem == nil || len(stw.SelectElem) == 0 {
        stw.EscapeAll()
        return
    }
    replaceenod := func (n *Node) {
        for _, el := range stw.SelectElem {
            for i, en := range el.Enod {
                if en == stw.SelectNode[0] {
                    el.Enod[i] = n
                    break
                }
            }
        }
        stw.SelectNode = make([]*Node, 2)
        stw.Redraw()
    }
    prune := func () {
        for _, el := range stw.SelectElem {
            var ens int
            tmp := make([]*Node, el.Enods)
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
        stw.SelectNode = make([]*Node, 2)
        stw.Redraw()
    }
    get2nodes(stw, replaceenod, prune)
}
// }}}


// Reaction// {{{
func reaction (stw *Window) {
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
                                              stw.startX = arg.X; stw.startY = arg.Y
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



// NOTICE1459
func notice1459 (stw *Window) {
    getnnodes(stw, 3, func (num int) {
                          var delta float64
                          ds := make([]float64, num)
                          for i:=0; i<num; i++ {
                              ds[i] = -stw.SelectNode[i].ReturnDisp(stw.Show.Period, 2)*100
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



// CATBYNODE// {{{
func catbynode (stw *Window) {
    get1node(stw, func (n *Node) {
                      err :=stw.Frame.CatByNode(n, true)
                      if err != nil {
                          stw.addHistory(err.Error())
                      } else {
                          stw.Redraw()
                      }
                  })
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
                                              stw.startX = arg.X; stw.startY = arg.Y
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
                          els := make([]*Elem, 2)
                          num := 0
                          for _, el := range stw.SelectElem {
                              if el != nil {
                                  els[num] = el
                                  num++
                                  if num >= 2 { break }
                              }
                          }
                          if num == 2 {
                              err := stw.Frame.JoinLineElem(els[0], els[1], true)
                              if err != nil {
                                  stw.addHistory(err.Error())
                              } else {
                                  stw.EscapeAll()
                              }
                          }
                      })
}
// }}}


// ERASE// {{{
func erase (stw *Window) {
    stw.DeleteSelected()
}
// }}}
