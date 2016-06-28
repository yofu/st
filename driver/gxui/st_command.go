package stgxui

import (
	"fmt"
	"github.com/google/gxui"
	gxmath "github.com/google/gxui/math"
	"github.com/yofu/st/stlib"
	"strconv"
)

var (
	Commands = make(map[string]*Command, 0)

	DISTS         = &Command{"DISTANCE", "DISTS", "measure distance", dists}
	DEFORMEDDISTS = &Command{"DEFORMEDDISTANCE", "DEFORMED DISTS", "measure deformed distance", deformeddists}
)

type Command struct {
	Display string
	Name    string
	command string
	call    func(*Window)
}

func (cmd *Command) Exec(stw *Window) {
	cmd.call(stw)
}

func init() {
	Commands["DISTS"] = DISTS
	Commands["DEFORMEDDISTS"] = DEFORMEDDISTS
}

// GET2NODES // {{{
func get2nodes(stw *Window, f func(n *st.Node), fdel func()) {
	// stw.canv.SetAttribute("CURSOR", "CROSS")
	stw.selectNode = make([]*st.Node, 2)
	stw.History("始端を指定[ダイアログ(D,R)]")
	setnnum := func() {
		nnum, err := strconv.ParseInt(stw.cline.Text(), 10, 64)
		if err == nil {
			if n, ok := stw.Frame.Nodes[int(nnum)]; ok {
				if stw.selectNode[0] != nil {
					f(n)
				} else {
					stw.selectNode[0] = n
					stw.History("終端を指定[ダイアログ(D,R)]")
				}
			}
		}
		stw.cline.SetText("")
	}
	stw.draw.OnMouseUp(func(ev gxui.MouseEvent) {
		if stw.Frame != nil {
			switch ev.Button {
			case gxui.MouseButtonLeft:
				if len(stw.selectNode) > 0 && stw.selectNode[0] != nil {
					if n := stw.Frame.PickNode(float64(ev.Point.X), float64(ev.Point.Y), float64(nodeSelectPixel)); n != nil {
						f(n)
					}
				} else {
					if n := stw.Frame.PickNode(float64(ev.Point.X), float64(ev.Point.Y), float64(nodeSelectPixel)); n != nil {
						stw.selectNode[0] = n
						stw.History("終端を指定[ダイアログ(D,R)]")
					}
				}
				stw.Redraw()
			case gxui.MouseButtonRight:
				if v := stw.cline.Text(); v != "" {
					setnnum()
				} else {
					stw.EscapeAll()
				}
			}
		}
	})
	stw.draw.OnMouseMove(func(ev gxui.MouseEvent) {
		fmt.Println("get2node")
		if stw.Frame != nil {
			// Snapping
			n := stw.Frame.PickNode(float64(ev.Point.X), float64(ev.Point.Y), float64(nodeSelectPixel))
			if n != nil {
				stw.rubber = stw.driver.CreateCanvas(gxmath.Size{W: stw.CanvasSize[0], H: stw.CanvasSize[1]})
				// Circle(stw.rubber, RubberPenSnap, int(n.Pcoord[0]), int(n.Pcoord[1]), nodeSelectPixel)
				stw.rubber.Complete()
				// stw.SetCoord(n.Coord[0], n.Coord[1], n.Coord[2])
			}
			///
			if ev.State.IsDown(gxui.MouseButtonMiddle) {
				stw.MoveOrRotate(ev)
				stw.RedrawNode()
			}
			if len(stw.selectNode) > 0 && stw.selectNode[0] != nil {
				stw.TailLine(int(stw.selectNode[0].Pcoord[0]), int(stw.selectNode[0].Pcoord[1]), ev)
			}
		}
	})
	// stw.canv.SetCallback(func(arg *iup.CommonKeyAny) {
	// 	key := iup.KeyState(arg.Key)
	// 	switch key.Key() {
	// 	default:
	// 		stw.DefaultKeyAny(arg)
	// 	case KEY_BS:
	// 		val := stw.cline.GetAttribute("VALUE")
	// 		if val != "" {
	// 			stw.cline.SetAttribute("VALUE", val[:len(val)-1])
	// 		}
	// 	case KEY_DELETE:
	// 		fdel()
	// 	case KEY_ENTER:
	// 		setnnum()
	// 	case KEY_ESCAPE:
	// 		stw.EscapeAll()
	// 	case 'D', 'd':
	// 		x, y, z, err := stw.QueryCoord("GET COORD")
	// 		if err == nil {
	// 			n, _ := stw.Frame.CoordNode(x, y, z, EPS)
	// 			stw.Redraw()
	// 			if stw.SelectNode[0] != nil {
	// 				f(n)
	// 			} else {
	// 				stw.SelectNode[0] = n
	// 				stw.cdcanv.Foreground(cd.CD_DARK_RED)
	// 				stw.cdcanv.WriteMode(cd.CD_XOR)
	// 				first = 1
	// 				stw.addHistory("終端を指定[ダイアログ(D,R)]")
	// 			}
	// 		}
	// 	case 'R', 'r':
	// 		x, y, z, err := stw.QueryCoord("GET COORD")
	// 		if err == nil {
	// 			if stw.SelectNode[0] != nil {
	// 				n, _ := stw.Frame.CoordNode(x+stw.SelectNode[0].Coord[0], y+stw.SelectNode[0].Coord[1], z+stw.SelectNode[0].Coord[2], EPS)
	// 				stw.Redraw()
	// 				f(n)
	// 			} else {
	// 				n, _ := stw.Frame.CoordNode(x, y, z, EPS)
	// 				stw.Redraw()
	// 				stw.SelectNode[0] = n
	// 				stw.cdcanv.Foreground(cd.CD_DARK_RED)
	// 				stw.cdcanv.WriteMode(cd.CD_XOR)
	// 				first = 1
	// 				stw.addHistory("終端を指定[ダイアログ(D,R)]")
	// 			}
	// 		}
	// 	}
	// })
}

// }}}

// DISTS// {{{
func dists(stw *Window) {
	get2nodes(stw, func(n *st.Node) {
		stw.selectNode[1] = n
		dx, dy, dz, d := stw.Frame.Distance(stw.selectNode[0], n)
		stw.History(fmt.Sprintf("NODE: %d - %d", stw.selectNode[0].Num, n.Num))
		stw.History(fmt.Sprintf("DX: %.3f DY: %.3f DZ: %.3f D: %.3f", dx, dy, dz, d))
		stw.EscapeAll()
	},
		func() {
			stw.selectNode = make([]*st.Node, 2)
			stw.Redraw()
		})
}

func deformeddists(stw *Window) {
	get2nodes(stw, func(n *st.Node) {
		stw.selectNode[1] = n
		dx, dy, dz, d := stw.Frame.Distance(stw.selectNode[0], n)
		stw.History(fmt.Sprintf("NODE: %d - %d", stw.selectNode[0].Num, n.Num))
		stw.History(fmt.Sprintf("DX: %.3f DY: %.3f DZ: %.3f D: %.3f", dx, dy, dz, d))
		dx, dy, dz, d = stw.Frame.DeformedDistance(stw.selectNode[0], n)
		stw.History(fmt.Sprintf("dx: %.3f dy: %.3f dz: %.3f d: %.3f", dx, dy, dz, d))
		stw.EscapeAll()
	},
		func() {
			stw.selectNode = make([]*st.Node, 2)
			stw.Redraw()
		})
}

// }}}
