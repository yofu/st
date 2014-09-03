package stgui

import (
    "fmt"
    "strconv"
    "math"
    "sort"
    "time"
    "github.com/visualfc/go-iup/iup"
    "github.com/yofu/st/stlib"
)

var (
    TENSEGRITYADD       = &Command{"#ADD", "TENSEGRITYADD", "add tensegrity elem", tensegrityadd}
    TENSEGRITYDELETE    = &Command{"#DEL", "TENSEGRITYDELETE", "delete tensegrity elem", tensegritydelete}
    TENSEGRITYCONNECTED = &Command{"#CON", "TENSEGRITYCONNECTED", "select tensegrity elem", tensegrityconnected}
)

func init() {
    Commands["TENSEGRITYADD"]=TENSEGRITYADD
    Commands["TENSEGRITYDELETE"]=TENSEGRITYDELETE
    Commands["TENSEGRITYCONNECTED"]=TENSEGRITYCONNECTED
}

// TENSEGRITY// {{{
func tensegrity (stw *Window, f func(cs, ts, sz int), csect, tsect, size int) {
    iup.SetFocus(stw.canv)
    tsmode := "SIZE"
    settsect := func() {
        val, err := strconv.ParseInt(stw.cline.GetAttribute("VALUE"),10,64)
        if err == nil {
            tsect = int(val)
            stw.addHistory(fmt.Sprintf("TENSION = %d", tsect))
        }
        stw.cline.SetAttribute("VALUE", "")
    }
    setcsect := func() {
        val, err := strconv.ParseInt(stw.cline.GetAttribute("VALUE"),10,64)
        if err == nil {
            csect = int(val)
            stw.addHistory(fmt.Sprintf("COMPRESSION = %d", csect))
        }
        stw.cline.SetAttribute("VALUE", "")
    }
    setsize := func() {
        val, err := strconv.ParseInt(stw.cline.GetAttribute("VALUE"),10,64)
        if err == nil {
            size = int(val)
            stw.addHistory(fmt.Sprintf("SIZE = %d", size))
        }
        stw.cline.SetAttribute("VALUE", "")
    }
    stw.canv.SetCallback( func (arg *iup.CommonKeyAny) {
                              key := iup.KeyState(arg.Key)
                              switch key.Key() {
                              default:
                                  stw.cline.SetAttribute("INSERT", string(key.Key()))
                              case KEY_BS:
                                  val := stw.cline.GetAttribute("VALUE")
                                  if val != "" {
                                      stw.cline.SetAttribute("VALUE", val[:len(val)-1])
                                  }
                              case 'T','t':
                                  tsmode = "TENSION"
                                  stw.addHistory(fmt.Sprintf("SET MODE: TENSION (CURRENT SECT = %d)", tsect))
                              case 'C','c':
                                  tsmode = "COMPRESSION"
                                  stw.addHistory(fmt.Sprintf("SET MODE: COMPRESSION (CURRENT SECT = %d)", csect))
                              case 'S','s':
                                  tsmode = "SIZE"
                                  stw.addHistory(fmt.Sprintf("SET MODE: SIZE (CURRENT SIZE = %d)", size))
                              case KEY_ENTER:
                                  if key.IsCtrl() {
                                      f(csect, tsect, size)
                                  } else {
                                      switch tsmode {
                                      case "TENSION":
                                          settsect()
                                      case "COMPRESSION":
                                          setcsect()
                                      case "SIZE":
                                          setsize()
                                      }
                                  }
                              case KEY_ESCAPE:
                                  stw.EscapeAll()
                              }
                          })
    stw.canv.SetCallback( func (arg *iup.MouseButton) {
                              if stw.Frame != nil {
                                  switch arg.Button {
                                  case BUTTON_LEFT:
                                      if isAlt(arg.Status) {
                                          stw.SelectNodeStart(arg)
                                      } else {
                                          stw.SelectElemStart(arg)
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
                                              stw.startX = int(arg.X); stw.startY = int(arg.Y)
                                          }
                                      }
                                  case BUTTON_RIGHT:
                                      if arg.Pressed == 0 {
                                          f(csect, tsect, size)
                                      } else {
                                          pressed = time.Now()
                                      }
                                  }
                              }
                          })
}
// }}}


func tensegrityconnected (stw *Window) {
    if stw.SelectElem == nil || len(stw.SelectElem)==0 { return }
    tensegrity(stw, func (cs, ts, sz int) {
                        var tmpelems []*st.Elem
                        var num int
                        var el0 *st.Elem
                        for _, el := range stw.SelectElem {
                            if el != nil { el0 = el; break }
                        }
                        if el0 != nil {
                            switch el0.Sect.Num {
                            case cs:
                                loopcs:
                                    for _, el := range stw.Frame.Elems {
                                        if el.Sect.Num != cs { continue }
                                        for i:=0; i<2; i++ {
                                            n1 := el0.Enod[i]
                                            for j:=0; j<2; j++ {
                                                n2 := el.Enod[j]
                                                els := stw.Frame.NodeToElemAll(n1, n2)
                                                if len(els) >= 1 {
                                                    tmpelems = append(tmpelems, el)
                                                    num++
                                                    continue loopcs
                                                }
                                            }
                                        }
                                    }
                            case ts:
                                found := []bool{false, false}
                                for _, el := range stw.Frame.Elems {
                                    if el.Sect.Num != cs { continue }
                                    loopts:
                                        for _, en := range el.Enod {
                                            for i, n := range el0.Enod {
                                                if found[i] { continue }
                                                if en == n {
                                                    tmpelems = append(tmpelems, el)
                                                    found[i] = true
                                                    num++
                                                    break loopts
                                                }
                                            }
                                        }
                                    if num == 2 { break }
                                }
                            }
                            stw.SelectElem = tmpelems[:num]
                            stw.Redraw()
                        }
                    }, 201, 211, 0)
}

func tensegrityadd (stw *Window) {
    if stw.SelectElem == nil || len(stw.SelectElem)==0 { return }
    tensegrity(stw, func (cs, ts, sz int) {
                        var tmpelems []*st.Elem
                        var num int
                        for _, el := range stw.SelectElem {
                            if el == nil { continue }
                            if el.Sect.Num == cs {
                                for i:=0; i<2; i++ {
                                    for _, n := range el.BetweenNode(i, sz) {
                                        tmpelems = append(tmpelems, st.NewLineElem([]*st.Node{el.Enod[i], n}, stw.Frame.Sects[ts], st.COLUMN))
                                        num++
                                    }
                                }
                            }
                        }
                        stw.SelectElem = tmpelems[:num]
                        stw.Redraw()
                        if stw.Yn("TENSEGRITY ADD", "部材を追加しますか?") {
                            for _, el := range tmpelems[:num] {
                                stw.Frame.AddElem(-1, el)
                            }
                        }
                        stw.EscapeAll()
                    }, 201, 211, 2)
}

func tensegritydelete (stw *Window) {
    if stw.SelectNode == nil || len(stw.SelectNode)==0 { return }
    tensegrity(stw, func (cs, ts, sz int) {
                        del := false
                        stw.SelectElem = make([]*st.Elem, 0)
                        for _, n := range stw.SelectNode {
                            var tens  []*st.Elem
                            var tnum int
                            for _, el := range stw.Frame.SearchElem(n) {
                                if el.Sect.Num == ts {
                                    tens = append(tens, el)
                                    tnum++
                                }
                            }
                            if tnum > sz {
                                del = true
                                elems := make(map[float64]*st.Elem, tnum)
                                sortedelems := make([]*st.Elem, tnum)
                                var keys []float64
                                for _, el := range tens[:tnum] {
                                    val := math.Abs(el.N("L", 0) * 1.5)
                                    // fmt.Printf("ELEM: %4d PERIOD: L   N= %.3f\n", el.Num, val)
                                    for _, per := range []string{"+X", "-X", "+Y", "-Y"} {
                                        tmp := math.Abs(el.N(fmt.Sprintf("L%s", per), 0))
                                        // fmt.Printf("           PERIOD: L%s N= %.3f\n", per, tmp)
                                        if tmp < val {
                                            val = tmp
                                        }
                                    }
                                    elems[val] = el
                                }
                                for k := range elems {
                                    keys = append(keys, k)
                                }
                                sort.Float64s(keys)
                                for i, k := range keys {
                                    sortedelems[i] = elems[k]
                                }
                                stw.SelectElem = append(stw.SelectElem, sortedelems[:tnum-sz]...) // valが小さいものを削除
                                // stw.SelectElem = append(stw.SelectElem, sortedelems[sz:]...) // valが大きいものを削除
                            }
                        }
                        stw.Redraw()
                        if del && stw.Yn("TENSEGRITY DELETE", "部材を削除しますか?") {
                            stw.DeleteSelected()
                        } else {
                            fmt.Println("No Relevant Elems")
                        }
                        stw.EscapeCB()
                    }, 201, 211, 6)
}

// func tensegritydelete2 (stw *Window) {
//     if stw.SelectElem == nil || len(stw.SelectElem)==0 { return }
//     tensegrity(stw, func (cs, ts, sz int) {
//                         del := false
//                         tnum := 0
//                         ref := make(map[int]int)
//                         for _, el := range stw.Frame.Elems {
//                         }
//                         elems := make(map[float64]*st.Elem, tnum)
//                         for _, el := range stw.SelectElem {
//                             if el.Sect.Num == ts {
//                                 val := math.Abs(el.N("L", 0) * 1.5)
//                                 for _, per := range []string{"+X", "-X", "+Y", "-Y"} {
//                                     tmp := math.Abs(el.N(fmt.Sprintf("L%s", per), 0))
//                                     if tmp > val {
//                                         val = tmp
//                                     }
//                                 }
//                                 elems[val] = el
//                                 tnum++
//                             }
//                         }
//                         stw.SelectElem = make([]*st.Elem, 0)
//                         sortedelems := make([]*st.Elem, tnum)
//                         var keys []float64
//                         for k := range elems {
//                             keys = append(keys, k)
//                         }
//                         sort.Float64s(keys)
//                         for i, k := range keys {
//                             sortedelems[i] = elems[k]
//                         }
//                         looptsd2:
//                             for _, el := range sortedelems {
//                                 comps := make([]*st.Elem, 2)
//                                 count := 0
//                                 for _, en := range el.Enod {
//                                     var comp *st.Elem
//                                     for _, se := range stw.Frame.SearchElem(en) {
//                                         if se.Sect.Num == cs {
//                                             comp = se
//                                             break
//                                         }
//                                     }
//                                     if comp == nil { continue looptsd2 }
//                                     for i, cen := range comp.Enod {
//                                         if cen == en {
//                                             if i==0 {
//                                                 if ref[comp.Num] <= sz { continue looptsd2 }
//                                             } else {
//                                                 if ref[-comp.Num] <= sz { continue looptsd2 }
//                                             }
//                                         }
//                                     }
//                                     count++
//                                 }
//                                 if count == 2 {
//                                     for _, cm := range comps {
//                                         ref[cm.Num]--
//                                     }
//                                     del = true
//                                     stw.SelectElem = append(stw.SelectElem, el)
//                                 }
//                             }
//                         stw.DrawFrame(ColorMode)
//                         if del && stw.Yn("TENSEGRITY DELETE", "部材を削除しますか?") {
//                             stw.DeleteSelected()
//                         } else {
//                             fmt.Println("No Relevant Elems")
//                         }
//                         stw.EscapeCB()
//                     }, 201, 211, 6)
// }
