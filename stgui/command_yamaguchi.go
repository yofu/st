package stgui

import (
    "fmt"
    "math"
    "sort"
    "time"
    "strconv"
    "github.com/visualfc/go-iup/iup"
    "github.com/yofu/st/stlib"
)

var (
    GRAVITY = &Command{"#GRV", "GRAVITY", "move nodes like gravitational potential", gravity}
    SMOOTH  = &Command{"#SMT", "SMOOTH", "move nodes towards flatten", smooth}
    HANGING = &Command{"#HNG", "HANGING", "hang nodes", hanging}
    RING    = &Command{"#RNG", "RING", "ring", ring}
)

func init() {
    Commands["GRAVITY"]=GRAVITY
    Commands["SMOOTH"]=SMOOTH
    Commands["HANGING"]=HANGING
    Commands["RING"]=RING
}

// GRAVITY// {{{
func gravity (stw *Window) {
    iup.SetFocus(stw.canv)
    stw.Frame.Show.NodeCaption |= st.NC_ZCOORD
    stw.Deselect()
    // stw.Frame.Show.ColorMode = st.ECOLOR_HEIGHT
    stw.Redraw()
    height := 0.1
    maxrange := 0.5
    t := maxrange
    mousetime := true
    gravitymode := "HEIGHT"
    sag := func(x, y, r, m float64) {
        var dx, dy, d float64
        for _, n := range(stw.Frame.Nodes) {
            if n.Conf[2] || n.Hide || n.Lock { continue }
            dx = math.Abs(n.Coord[0] - x)
            if dx <= 2*r {
                dy = math.Abs(n.Coord[1] - y)
                if dy <= 2*r {
                    d = math.Sqrt(math.Pow(dx, 2.0) + math.Pow(dy, 2.0))
                    if d <= r {
                        n.Coord[2] += -0.5*m*math.Pow(d, 2.0)/math.Pow(r, 3.0) + m/r
                    } else if d <= 2*r {
                        n.Coord[2] += m/d - 0.5*m/r
                    }
                }
            }
        }
    }
    setheight := func() {
        val, err := strconv.ParseFloat(stw.cline.GetAttribute("VALUE"),64)
        if err == nil {
            height = val
            stw.addHistory(fmt.Sprintf("HEIGHT = %.3f", height))
        }
        stw.cline.SetAttribute("VALUE", "")
    }
    setrange := func() {
        val, err := strconv.ParseFloat(stw.cline.GetAttribute("VALUE"),64)
        if err == nil {
            maxrange = val
            stw.addHistory(fmt.Sprintf("RANGE = %.3f", maxrange))
        }
        stw.cline.SetAttribute("VALUE", "")
    }
    stw.canv.SetCallback( func (arg *iup.MouseButton) {
                              if stw.Frame != nil {
                                  switch arg.Button {
                                  case BUTTON_LEFT:
                                      stw.dbuff.UpdateYAxis(&arg.Y)
                                      if arg.Pressed == 0 {
                                          n := stw.PickNode(int(arg.X), int(arg.Y))
                                          if n != nil {
                                              if mousetime {
                                                  t = math.Min(time.Since(stw.startT).Seconds(), maxrange)
                                              } else {
                                                  t = maxrange
                                              }
                                              sag(n.Coord[0], n.Coord[1], t, height*t)
                                              stw.Redraw()
                                              // stw.EscapeAll()
                                          }
                                      } else {
                                          stw.startT = time.Now()
                                      }
                                  case BUTTON_RIGHT:
                                      stw.dbuff.UpdateYAxis(&arg.Y)
                                      if arg.Pressed == 0 {
                                          n := stw.PickNode(int(arg.X), int(arg.Y))
                                          if n != nil {
                                              if mousetime {
                                                  t = math.Min(time.Since(stw.startT).Seconds(), maxrange)
                                              } else {
                                                  t = maxrange
                                              }
                                              sag(n.Coord[0], n.Coord[1], t, -height*t)
                                              stw.Redraw()
                                              // stw.EscapeAll()
                                          }
                                      } else {
                                          stw.startT = time.Now()
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
                                  }
                              }
                          })
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
                              case 'H','h':
                                  gravitymode = "HEIGHT"
                                  stw.addHistory(fmt.Sprintf("SET MODE: HEIGHT (CURRENT HEIGHT = %.3f", height))
                              case 'R','r':
                                  gravitymode = "RANGE"
                                  stw.addHistory(fmt.Sprintf("SET MODE: RANGE (CURRENT RANGE = %.3f", maxrange))
                              case 'M','m':
                                  if mousetime {
                                      mousetime = false
                                      stw.addHistory("MOUSE TIME: OFF")
                                  } else {
                                      mousetime = true
                                      stw.addHistory("MOUSE TIME: ON")
                                  }
                              case KEY_ENTER:
                                  switch gravitymode {
                                  case "HEIGHT":
                                      setheight()
                                  case "RANGE":
                                      setrange()
                                  }
                              case KEY_ESCAPE:
                                  stw.EscapeAll()
                              }
                          })
}
// }}}


// SMOOTH// {{{
func smooth (stw *Window) {
    iup.SetFocus(stw.canv)
    stw.Frame.Show.NodeCaption |= st.NC_ZCOORD
    stw.Deselect()
    // stw.Frame.Show.ColorMode = st.ECOLOR_HEIGHT
    stw.Redraw()
    ratio := 0.5
    flatten := func(n *st.Node) {
        if n.Conf[2] || n.Hide || n.Lock { return }
        z0 := n.Coord[2]
        for _, cn := range(stw.Frame.LineConnected(n)) {
            if cn.Conf[2] { continue }
            cn.Coord[2] += ratio*(z0-cn.Coord[2])
        }
    }
    setratio := func() {
        val, err := strconv.ParseFloat(stw.cline.GetAttribute("VALUE"),64)
        if err == nil {
            ratio = val
            stw.addHistory(fmt.Sprintf("RATIO = %.3f", ratio))
        }
        stw.cline.SetAttribute("VALUE", "")
    }
    stw.canv.SetCallback( func (arg *iup.MouseButton) {
                              if stw.Frame != nil {
                                  switch arg.Button {
                                  case BUTTON_LEFT:
                                      stw.dbuff.UpdateYAxis(&arg.Y)
                                      if arg.Pressed == 0 {
                                          n := stw.PickNode(int(arg.X), int(arg.Y))
                                          if n != nil {
                                              flatten(n)
                                              stw.Redraw()
                                              // stw.EscapeAll()
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
                                              stw.startX = int(arg.X); stw.startY = int(arg.Y)
                                          }
                                      }
                                  }
                              }
                          })
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
                              case KEY_ENTER:
                                  setratio()
                              case KEY_ESCAPE:
                                  stw.EscapeAll()
                              }
                          })
}
// }}}


// HANGING// {{{
func hanging (stw *Window) {
    iup.SetFocus(stw.canv)
    stw.Frame.Show.NodeCaption |= st.NC_ZCOORD
    // stw.Frame.Show.ColorMode = st.ECOLOR_HEIGHT
    stw.Redraw()
    confed := make([]*st.Node, 0)
    height := 2.9
    radius := 0.4
    hangmode := "HEIGHT"
    for _, n := range stw.Frame.Nodes {
        if n.Conf[2] {
            confed = append(confed, n)
        }
    }
    potential := func(d float64) float64 {
        if d <= radius {
            return 0.5*height*math.Pow(d, 2.0)/math.Pow(radius, 2.0)
        } else if d <= 2.0*radius {
            return 1.5*height-height*radius/d
        } else {
            return height
        }
    }
    starthanging := func() {
        moved := make([]*st.Node, 0)
        for _, n := range stw.SelectNode {
            if n == nil { continue }
            if n.Conf[2] || n.Hide || n.Lock { continue }
            n.Coord[2] = height
            moved = append(moved, n)
        }
        for i, n := range confed {
            for _, m := range moved {
                tmp := 0.0
                for j:=0; j<2; j++ {
                    tmp += math.Pow(m.Coord[j]-n.Coord[j], 2.0)
                }
                d := math.Sqrt(tmp)
                newheight := potential(math.Sqrt(d))
                if m.Coord[2] >  newheight {
                    m.Coord[2] = newheight
                }
            }
            if i%200 == 0 { stw.Redraw() }
        }
        // stw.EscapeCB()
    }
    setheight := func() {
        val, err := strconv.ParseFloat(stw.cline.GetAttribute("VALUE"),64)
        if err == nil {
            height = val
            stw.addHistory(fmt.Sprintf("HEIGHT = %.3f", height))
        }
        stw.cline.SetAttribute("VALUE", "")
    }
    setradius := func() {
        val, err := strconv.ParseFloat(stw.cline.GetAttribute("VALUE"),64)
        if err == nil {
            radius = val
            stw.addHistory(fmt.Sprintf("RADIUS = %.3f", radius))
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
                              case 'H','h':
                                  hangmode = "HEIGHT"
                                  stw.addHistory(fmt.Sprintf("SET MODE: HEIGHT (CURRENT HEIGHT = %.3f", height))
                              case 'R','r':
                                  hangmode = "RADIUS"
                                  stw.addHistory(fmt.Sprintf("SET MODE: RADIUS (CURRENT RADIUS = %.3f", radius))
                              case KEY_ENTER:
                                  if key.IsCtrl() {
                                      starthanging()
                                  } else {
                                      switch hangmode {
                                      case "HEIGHT":
                                          setheight()
                                      case "RADIUS":
                                          setradius()
                                      }
                                  }
                              case KEY_ESCAPE:
                                  stw.EscapeAll()
                              }
                          })
}
// }}}


// RING// {{{
func ring (stw *Window) {
    if stw.SelectNode == nil {
        stw.EscapeAll()
        return
    }
    mid := make([]float64, 2)
    num := 0
    for _, n := range stw.SelectNode {
        if n != nil {
            num++
            for i:=0; i<2; i++ {
                mid[i] += n.Coord[i]
            }
        }
    }
    if num <= 2 { stw.EscapeAll(); return }
    for i:=0; i<2; i++ {
        mid[i] /= float64(num)
    }
    nodes := make(map[float64]*st.Node, num)
    sortednodes := make([]*st.Node, num)
    var keys []float64
    for _, n := range stw.SelectNode {
        if n != nil {
            nodes[math.Atan2(n.Coord[1]-mid[1], n.Coord[0]-mid[0])] = n
        }
    }
    for k := range(nodes) {
        keys = append(keys, k)
    }
    sort.Float64s(keys)
    for i, k := range(keys) {
        sortednodes[i] = nodes[k]
    }
    sec := stw.Frame.DefaultSect()
    for i:=0; i<num-1; i++ {
        stw.Frame.AddLineElem(-1, []*st.Node{sortednodes[i], sortednodes[i+1]}, sec, st.COLUMN)
    }
    stw.Frame.AddLineElem(-1, []*st.Node{sortednodes[num-1], sortednodes[0]}, sec, st.COLUMN)
    stw.EscapeAll()
}
// }}}
