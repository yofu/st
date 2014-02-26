package st

import (
    "fmt"
    "strconv"
    "math"
    "sort"
    "time"
    "github.com/visualfc/go-iup/iup"
)

var (
    TENSEGRITYADD = &Command{"TENSEGRITYADD", "add tensegrity elem", tensegrityadd}
    TENSEGRITYDELETE = &Command{"TENSEGRITYDELETE", "delete tensegrity elem", tensegritydelete}
    TENSEGRITYCONNECTED = &Command{"TENSEGRITYCONNECTED", "select tensegrity elem", tensegrityconnected}
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
                                              stw.startX = arg.X; stw.startY = arg.Y
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
                        var tmpelems []*Elem
                        var num int
                        var el0 *Elem
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
                        var tmpelems []*Elem
                        var num int
                        for _, el := range stw.SelectElem {
                            if el == nil { continue }
                            if el.Sect.Num == cs {
                                for i:=0; i<2; i++ {
                                    for _, n := range el.BetweenNode(i, sz) {
                                        tmpelems = append(tmpelems, NewLineElem([]*Node{el.Enod[i], n}, stw.Frame.Sects[ts], COLUMN))
                                        num++
                                    }
                                }
                            }
                        }
                        stw.SelectElem = tmpelems[:num]
                        stw.Redraw()
                        if stw.Yn("TENSEGRITY ADD", "部材を追加しますか?") {
                            stw.Frame.AddElem(tmpelems[:num]...)
                        }
                        stw.EscapeAll()
                    }, 201, 211, 2)
}

func tensegritydelete (stw *Window) {
    if stw.SelectNode == nil || len(stw.SelectNode)==0 { return }
    tensegrity(stw, func (cs, ts, sz int) {
                        del := false
                        stw.SelectElem = make([]*Elem, 0)
                        for _, n := range stw.SelectNode {
                            var tens  []*Elem
                            var tnum int
                            for _, el := range stw.Frame.SearchElem(n) {
                                if el.Sect.Num == ts {
                                    tens = append(tens, el)
                                    tnum++
                                }
                            }
                            if tnum > sz {
                                del = true
                                elems := make(map[float64]*Elem, tnum)
                                sortedelems := make([]*Elem, tnum)
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
//                         elems := make(map[float64]*Elem, tnum)
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
//                         stw.SelectElem = make([]*Elem, 0)
//                         sortedelems := make([]*Elem, tnum)
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
//                                 comps := make([]*Elem, 2)
//                                 count := 0
//                                 for _, en := range el.Enod {
//                                     var comp *Elem
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

func (elem *Elem) BetweenNode (index, size int) []*Node {
    var rtn []*Node
    var dst []float64
    var all bool
    if size < 0 {
        all = true
        rtn = make([]*Node, 0)
    } else {
        all = false
        rtn = make([]*Node, size)
        dst = make([]float64, size)
    }
    if size==0 || !elem.IsLineElem() { return rtn }
    d := elem.Direction(true)
    L := elem.Length()
    maxlen := 1000.0
    cand := 0
    for _, n := range elem.Frame.Nodes {
        if n.Hide { continue }
        if n == elem.Enod[0] || n == elem.Enod[1] { continue }
        d2 := Direction(elem.Enod[index], n, false)
        var ip float64
        if index == 0 {
            ip = Dot(d, d2, 3)
        } else {
            ip = -Dot(d, d2, 3)
        }
        if 0 < ip && ip < L {
            if all {
                rtn = append(rtn, n)
            } else {
                tmpd := Distance(elem.Enod[index], n)
                if cand < size {
                    last := true
                    for i:=0; i<cand; i++ {
                        if tmpd < dst[i] {
                            for j:=cand; j>i; j-- {
                                rtn[j] = rtn[j-1]
                                dst[j]=dst[j-1]
                            }
                            rtn[i] = n
                            dst[i] = tmpd
                            last = false
                            break
                        }
                    }
                    if last {
                        rtn[cand] = n
                        dst[cand] = tmpd
                    }
                    maxlen  = dst[cand]
                } else {
                    if tmpd < maxlen {
                        first := true
                        for i:=size-1; i>0; i-- {
                            if tmpd > dst[i-1] {
                                for j:=size-1; j>i; j-- {
                                    rtn[j] = rtn[j-1]
                                    dst[j] = dst[j-1]
                                }
                                rtn[i] = n
                                dst[i] = tmpd
                                first = false
                                break
                            }
                        }
                        if first {
                            for i:=size-1; i>0; i-- {
                                rtn[i] = rtn[i-1]
                                dst[i] = dst[i-1]
                            }
                            rtn[0] = n
                            dst[0] = tmpd
                        }
                        maxlen = dst[size-1]
                    }
                }
            }
            cand++
        }
    }
    if all {
        return rtn[:cand]
    } else {
        return rtn[:size]
    }
}

