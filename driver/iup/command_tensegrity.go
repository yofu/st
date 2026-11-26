package stgui

import (
	"fmt"
	"strconv"
	"time"

	"github.com/yofu/go-iup/iup"
	"github.com/yofu/st/stlib"
)

var (
	TENSEGRITYADD       = &Command{"#ADD", "TENSEGRITYADD", "add tensegrity elem", tensegrityadd}
	TENSEGRITYDELETE    = &Command{"#DEL", "TENSEGRITYDELETE", "delete tensegrity elem", tensegritydelete}
	TENSEGRITYCONNECTED = &Command{"#CON", "TENSEGRITYCONNECTED", "select tensegrity elem", tensegrityconnected}
)

func init() {
	Commands["TENSEGRITYADD"] = TENSEGRITYADD
	Commands["TENSEGRITYDELETE"] = TENSEGRITYDELETE
	Commands["TENSEGRITYCONNECTED"] = TENSEGRITYCONNECTED
}

// TENSEGRITY// {{{
func tensegrity(stw *Window, f func(cs, ts, sz int), csect, tsect, size int) {
	iup.SetFocus(stw.canv)
	tsmode := "SIZE"
	settsect := func() {
		val, err := strconv.ParseInt(stw.cline.GetAttribute("VALUE"), 10, 64)
		if err == nil {
			tsect = int(val)
			stw.addHistory(fmt.Sprintf("TENSION = %d", tsect))
		}
		stw.cline.SetAttribute("VALUE", "")
	}
	setcsect := func() {
		val, err := strconv.ParseInt(stw.cline.GetAttribute("VALUE"), 10, 64)
		if err == nil {
			csect = int(val)
			stw.addHistory(fmt.Sprintf("COMPRESSION = %d", csect))
		}
		stw.cline.SetAttribute("VALUE", "")
	}
	setsize := func() {
		val, err := strconv.ParseInt(stw.cline.GetAttribute("VALUE"), 10, 64)
		if err == nil {
			size = int(val)
			stw.addHistory(fmt.Sprintf("SIZE = %d", size))
		}
		stw.cline.SetAttribute("VALUE", "")
	}
	stw.setCanvasInput(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		switch key.Key() {
		default:
			if s, ok := printableKeyText(key); ok {
				stw.cline.SetAttribute("INSERT", s)
			}
		case KEY_BS:
			val := stw.cline.GetAttribute("VALUE")
			if val != "" {
				stw.cline.SetAttribute("VALUE", val[:len(val)-1])
			}
		case 'T', 't':
			tsmode = "TENSION"
			stw.addHistory(fmt.Sprintf("SET MODE: TENSION (CURRENT SECT = %d)", tsect))
		case 'C', 'c':
			tsmode = "COMPRESSION"
			stw.addHistory(fmt.Sprintf("SET MODE: COMPRESSION (CURRENT SECT = %d)", csect))
		case 'S', 's':
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
	stw.setCanvasInput(func(arg *iup.MouseButton) {
		if stw.frame != nil {
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
						stw.frame.SetFocus(nil)
						stw.RedrawNode()
						stw.ShowCenter()
					} else {
						stw.dbuff.UpdateYAxis(&arg.Y)
						stw.startX = int(arg.X)
						stw.startY = int(arg.Y)
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

func tensegrityconnected(stw *Window) {
	if !stw.ElemSelected() {
		return
	}
	tensegrity(stw, func(cs, ts, sz int) {
		for _, el := range stw.SelectedElems() {
			if el != nil {
				stw.SelectElem(st.TensegrityConnected(stw.frame, el, cs, ts))
				stw.Redraw()
				return
			}
		}
	}, 201, 211, 0)
}

func tensegrityadd(stw *Window) {
	if !stw.ElemSelected() {
		return
	}
	tensegrity(stw, func(cs, ts, sz int) {
		if sz < 0 {
			return
		}
		candidates, err := st.TensegrityAddCandidates(stw.frame, stw.SelectedElems(), cs, ts, sz)
		if err != nil {
			st.ErrorMessage(stw, err, st.ERROR)
			return
		}
		stw.SelectElem(candidates)
		stw.Redraw()
		if len(candidates) > 0 && stw.Yn("TENSEGRITY ADD", "部材を追加しますか?") {
			for _, el := range candidates {
				stw.frame.AddElem(-1, el)
			}
			st.Snapshot(stw)
		}
		stw.EscapeAll()
	}, 201, 211, 2)
}

func tensegritydelete(stw *Window) {
	if !stw.NodeSelected() {
		return
	}
	tensegrity(stw, func(cs, ts, sz int) {
		if sz < 0 {
			return
		}
		candidates, err := st.TensegrityDeleteCandidates(stw.frame, stw.SelectedNodes(), ts, sz)
		if err != nil {
			st.ErrorMessage(stw, err, st.ERROR)
			return
		}
		stw.SelectElem(candidates)
		stw.Redraw()
		if len(candidates) > 0 && stw.Yn("TENSEGRITY DELETE", "部材を削除しますか?") {
			stw.DeleteSelected()
		}
		stw.EscapeCB()
	}, 201, 211, 6)
}

// func tensegritydelete2 (stw *Window) {
//     if stw.selectElem == nil || len(stw.selectElem)==0 { return }
//     tensegrity(stw, func (cs, ts, sz int) {
//                         del := false
//                         tnum := 0
//                         ref := make(map[int]int)
//                         for _, el := range stw.frame.Elems {
//                         }
//                         elems := make(map[float64]*st.Elem, tnum)
//                         for _, el := range stw.selectElem {
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
//                         stw.selectElem = make([]*st.Elem, 0)
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
//                                     for _, se := range stw.frame.SearchElem(en) {
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
//                                     stw.selectElem = append(stw.selectElem, el)
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
