package stgui

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/yofu/go-iup/cd"
	"github.com/yofu/go-iup/iup"
	st "github.com/yofu/st/stlib"
)

var EPS = 1e-4

var (
	Commands = make(map[string]*Command, 0)

	DISTS               = &Command{"DISTANCE", "DISTS", "measure distance", dists}
	DEFORMEDDISTS       = &Command{"DEFORMEDDISTANCE", "DEFORMED DISTS", "measure deformed distance", deformeddists}
	MEASURE             = &Command{"MEASURE", "MEASURE", "create dimension line", measure}
	KIJUN               = &Command{"KIJUN", "KIJUN", "create kijun line", kijun}
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
	SHOWPLANE           = &Command{"PLNE", "PLANE", "show elems on the plane", showplane}
	FENCE               = &Command{"FNCE", "FENCE", "select elem by fence", fence}
	ADDLINEELEM         = &Command{"LINE", "ADD LINE ELEM", "add line elem", addlineelem}
	ADDPLATEELEM        = &Command{"PLATE(4pts)", "ADD PLATE ELEM", "add plate elem", addplateelem}
	ADDPLATEELEMBYLINE  = &Command{"PLATE(2lines)", "ADD PLATE ELEM BY LINE", "add plate elem by line", addplateelembyline}
	HATCHPLATEELEM      = &Command{"HATCHING", "HATCH PLATE ELEM", "add plate elem by hatching", hatchplateelem}
	ADDPLATEALL         = &Command{"PLATE(all)", "ADD PLATE ALL", "add all plate elem using selected nodes", addplateall}
	EDITLINEELEM        = &Command{"EDLI", "EDIT LINE ELEM", "edit line elem", editlineelem}
	EDITPLATEELEM       = &Command{"EDPL", "EDIT PLATE ELEM", "edit plate elem", editplateelem}
	EDITWRECT           = &Command{"EDWR", "EDIT WINDOW RECT", "edit window rectangular", editwrect}
	CONVEXHULL          = &Command{"CVXH", "CONVEX HULL", "draw convex hull", convexhull}
	MATCHPROPERTY       = &Command{"COPY PROPERTY", "MATCH PROPERTY", "match property", matchproperty}
	AXISTOCANG          = &Command{"CANG", "AXISTOCANG", "set cang by axis", axistocang}
	COPYELEM            = &Command{"COPY", "COPY ELEM", "copy selected elems", copyelem}
	DUPLICATEELEM       = &Command{"DUP.", "DUPLICATE ELEM", "duplicate selected elems", duplicateelem}
	MOVEELEM            = &Command{"MOVE", "MOVE ELEM", "move selected elems", moveelem}
	MOVENODE            = &Command{"MOVE", "MOVE NODE", "move selected nodes", movenode}
	MOVETOLINE          = &Command{"MOVETOLINE", "MOVE NODE ONTO LINE", "move selected nodes onto the line", movetoline}
	PINCHNODE           = &Command{"NDPC", "PINCH NODE", "pinch nodes", pinchnode}
	ROTATE              = &Command{"ROTE", "ROTATE", "rotate selected nodes", rotate}
	MIRROR              = &Command{"MIRR", "MIRROR", "mirror selected elems", mirror}
	DIVIDEATPLANE       = &Command{"DIVP", "DIVIDEATPLANE", "divide selected elems", divideatplane}
	SCALE               = &Command{"SCLE", "SCALE", "scale selected nodes", scale}
	SEARCHELEM          = &Command{"ELSR", "SEARCH ELEM", "search elems using node", searchelem}
	NODETOELEM          = &Command{"N->E", "NODE TO ELEM", "select elems using selected node", nodetoelemall}
	ELEMTONODE          = &Command{"E->N", "ELEM TO NODE", "select nodes used by selected elem", elemtonode}
	CONNECTED           = &Command{"N->N", "CONNECTED", "select nodes connected to selected node", connected}
	ONNODE              = &Command{"ONND", "ON NODE", "select nodes which is on selected elems", onnode}
	NODENOREFERENCE     = &Command{"NODE NO REF.", "NODE NO REFERENCE", "delete nodes which are not refered by any elem", nodenoreference}
	ELEMSAMENODE        = &Command{"ELEM SAME NODE", "ELEM SAME NODE", "delete elems which has duplicated enod", elemsamenode}
	SUSPICIOUS          = &Command{"SUSPICIOUS", "SUSPICIOUS", "select suspicious nodes/elems", suspicious}
	PRUNEENOD           = &Command{"PRUNE ENOD", "PRUNE ENOD", "prune duplicated enod", pruneenod}
	NODEDUPLICATION     = &Command{"DUPLICATIVE NODE", "NODE DUPLICATION", "delete duplicated nodes", nodeduplication}
	ELEMDUPLICATION     = &Command{"DUPLICATIVE ELEM", "ELEM DUPLICATION", "delete duplicated elems", elemduplication}
	NODESORT            = &Command{"NODE SORT", "NODE SORT", "node sort", nodesort}
	CATBYNODE           = &Command{"CATN", "CAT BY NODE", "join 2 elems using selected node", catbynode}
	CATINTERMEDIATENODE = &Command{"CATI", "CAT INTERMEDIATE NODE", "concatenate at intermediate node", catintermediatenode}
	JOINLINEELEM        = &Command{"JOIN LINE", "JOIN LINE ELEM", "join selected 2 elems", joinlineelem}
	JOINPLATEELEM       = &Command{"JOIN PLATE", "JOIN PLATE ELEM", "join selected 2 elems", joinplateelem}
	EXTRACTARCLM        = &Command{"EXAR", "EXTRACT ARCLM", "extract arclm", extractarclm}
	DIVIDEATONS         = &Command{"ON NODES", "DIVIDE AT ONS", "divide selected elems at onnode", divideatons}
	DIVIDEATMID         = &Command{"MID POINT", "DIVIDE AT MID", "divide selected elems at midpoint", divideatmid}
	DIVIDEINN           = &Command{"IN N", "DIVIDE IN N", "divide selected elems in n parts", divideinn}
	DIVIDEATELEM        = &Command{"AT ELEM", "DIVIDE AT ELEM", "divide selected elems at elem", divideatelem}
	INTERSECT           = &Command{"INTS", "INTERSECT", "divide selected elems at intersection", intersect}
	INTERSECTALL        = &Command{"INTA", "INTERSECT ALL", "divide selected elems at all intersection", intersectall}
	TRIM                = &Command{"TRIM", "TRIM", "trim elements with selected elem", trim}
	EXTEND              = &Command{"EXTEND", "EXTEND", "extend elements to selected elem", extend}
	OFFSET              = &Command{"OFFSET", "OFFSET", "offset element", offset}
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
	SETEPS              = &Command{"SEPS", "SET EPSIRON", "set epsiron", seteps}
)

func init() {
	Commands["DISTS"] = DISTS
	Commands["DEFORMEDDISTS"] = DEFORMEDDISTS
	Commands["MEASURE"] = MEASURE
	Commands["KIJUN"] = KIJUN
	Commands["TOGGLEBOND"] = TOGGLEBOND
	Commands["COPYBOND"] = COPYBOND
	Commands["BONDPIN"] = BONDPIN
	Commands["BONDRIGID"] = BONDRIGID
	Commands["CONFFIX"] = CONFFIX
	Commands["CONFPIN"] = CONFPIN
	Commands["CONFXYROLLER"] = CONFXYROLLER
	Commands["CONFFREE"] = CONFFREE
	Commands["OPEN"] = OPEN
	Commands["SAVE"] = SAVE
	Commands["WEIGHTCOPY"] = WEIGHTCOPY
	Commands["READPGP"] = READPGP
	Commands["INSERT"] = INSERT
	Commands["SETFOCUS"] = SETFOCUS
	Commands["SELECTNODE"] = SELECTNODE
	Commands["SELECTCOLUMNBASE"] = SELECTCOLUMNBASE
	Commands["SELECTCONFED"] = SELECTCONFED
	Commands["SELECTELEM"] = SELECTELEM
	Commands["SELECTSECT"] = SELECTSECT
	Commands["HIDESECTION"] = HIDESECTION
	Commands["HIDECURTAINWALL"] = HIDECURTAINWALL
	Commands["SELECTCHILDREN"] = SELECTCHILDREN
	Commands["ERRORELEM"] = ERRORELEM
	Commands["SHOWPLANE"] = SHOWPLANE
	Commands["FENCE"] = FENCE
	Commands["ADDLINEELEM"] = ADDLINEELEM
	Commands["ADDPLATEELEM"] = ADDPLATEELEM
	Commands["ADDPLATEELEMBYLINE"] = ADDPLATEELEMBYLINE
	Commands["HATCHPLATEELEM"] = HATCHPLATEELEM
	Commands["ADDPLATEALL"] = ADDPLATEALL
	Commands["EDITLINEELEM"] = EDITLINEELEM
	Commands["EDITPLATEELEM"] = EDITPLATEELEM
	Commands["EDITWRECT"] = EDITWRECT
	Commands["CONVEXHULL"] = CONVEXHULL
	Commands["MATCHPROPERTY"] = MATCHPROPERTY
	Commands["AXISTOCANG"] = AXISTOCANG
	Commands["COPYELEM"] = COPYELEM
	Commands["DUPLICATEELEM"] = DUPLICATEELEM
	Commands["MOVEELEM"] = MOVEELEM
	Commands["MOVENODE"] = MOVENODE
	Commands["MOVETOLINE"] = MOVETOLINE
	Commands["PINCHNODE"] = PINCHNODE
	Commands["ROTATE"] = ROTATE
	Commands["MIRROR"] = MIRROR
	Commands["DIVIDEATPLANE"] = DIVIDEATPLANE
	Commands["SCALE"] = SCALE
	Commands["SEARCHELEM"] = SEARCHELEM
	Commands["NODETOELEM"] = NODETOELEM
	Commands["ELEMTONODE"] = ELEMTONODE
	Commands["CONNECTED"] = CONNECTED
	Commands["ONNODE"] = ONNODE
	Commands["NODENOREFERENCE"] = NODENOREFERENCE
	Commands["ELEMSAMENODE"] = ELEMSAMENODE
	Commands["SUSPICIOUS"] = SUSPICIOUS
	Commands["PRUNEENOD"] = PRUNEENOD
	Commands["NODEDUPLICATION"] = NODEDUPLICATION
	Commands["ELEMDUPLICATION"] = ELEMDUPLICATION
	Commands["NODESORT"] = NODESORT
	Commands["CATBYNODE"] = CATBYNODE
	Commands["CATINTERMEDIATENODE"] = CATINTERMEDIATENODE
	Commands["JOINLINEELEM"] = JOINLINEELEM
	Commands["JOINPLATEELEM"] = JOINPLATEELEM
	Commands["EXTRACTARCLM"] = EXTRACTARCLM
	Commands["DIVIDEATONS"] = DIVIDEATONS
	Commands["DIVIDEATMID"] = DIVIDEATMID
	Commands["DIVIDEINN"] = DIVIDEINN
	Commands["DIVIDEATELEM"] = DIVIDEATELEM
	Commands["INTERSECT"] = INTERSECT
	Commands["INTERSECTALL"] = INTERSECTALL
	Commands["TRIM"] = TRIM
	Commands["EXTEND"] = EXTEND
	Commands["OFFSET"] = OFFSET
	Commands["MERGENODE"] = MERGENODE
	Commands["ERASE"] = ERASE
	Commands["FACTS"] = FACTS
	Commands["REACTION"] = REACTION
	Commands["SUMREACTION"] = SUMREACTION
	Commands["UPLIFT"] = UPLIFT
	Commands["NOTICE1459"] = NOTICE1459
	Commands["ZOUBUNDISP"] = ZOUBUNDISP
	Commands["ZOUBUNYIELD"] = ZOUBUNYIELD
	Commands["ZOUBUNREACTION"] = ZOUBUNREACTION
	Commands["AMOUNTPROP"] = AMOUNTPROP
	Commands["SETEPS"] = SETEPS
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
func getcoord(stw *Window, f func(x, y, z float64)) {
	stw.canv.SetAttribute("CURSOR", "CROSS")
	var snap *st.Node // for Snapping
	stw.cdcanv.Foreground(cd.CD_YELLOW)
	stw.cdcanv.WriteMode(cd.CD_XOR)
	stw.SelectNode(make([]*st.Node, 1))
	setnnum := func() {
		nnum, err := strconv.ParseInt(stw.cline.GetAttribute("VALUE"), 10, 64)
		if err == nil {
			if n, ok := stw.frame.Nodes[int(nnum)]; ok {
				f(n.Coord[0], n.Coord[1], n.Coord[2])
			}
		}
		stw.cline.SetAttribute("VALUE", "")
	}
	stw.canv.SetCallback(func(arg *iup.MouseButton) {
		if stw.frame != nil {
			stw.dbuff.UpdateYAxis(&arg.Y)
			switch arg.Button {
			case BUTTON_LEFT:
				if arg.Pressed == 0 { // Released
					if n := stw.frame.PickNode(float64(arg.X), float64(arg.Y), float64(nodeSelectPixel)); n != nil {
						f(n.Coord[0], n.Coord[1], n.Coord[2])
					}
					// stw.cdcanv.Line(int(stw.selectNode[0].Pcoord[0]), int(stw.selectNode[0].Pcoord[1]), stw.endX, stw.endY) // TODO
					stw.Redraw()
				}
			case BUTTON_CENTER:
				if arg.Pressed == 0 { // Released
					// if stw.selectNode[0] != nil { // TODO
					//     stw.cdcanv.Line(int(stw.selectNode[0].Pcoord[0]), int(stw.selectNode[0].Pcoord[1]), stw.endX, stw.endY)
					// }
					stw.Redraw()
				} else { // Pressed
					if isDouble(arg.Status) {
						stw.frame.SetFocus(nil)
						stw.RedrawNode()
						stw.ShowCenter()
					} else {
						stw.startX = int(arg.X)
						stw.startY = int(arg.Y)
					}
				}
			case BUTTON_RIGHT:
				if arg.Pressed == 0 {
					if v := stw.cline.GetAttribute("VALUE"); v != "" {
						setnnum()
					} else {
						// fmt.Println("CONTEXT MENU")
						stw.EscapeAll()
					}
				}
			}
		}
	})
	stw.canv.SetCallback(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		switch key.Key() {
		default:
			stw.DefaultKeyAny(arg)
		case KEY_BS:
			val := stw.cline.GetAttribute("VALUE")
			if val != "" {
				stw.cline.SetAttribute("VALUE", val[:len(val)-1])
			}
		case KEY_ENTER:
			setnnum()
		case KEY_ESCAPE:
			stw.EscapeAll()
		case 'D', 'd':
			x, y, z, err := stw.QueryCoord("GET COORD")
			if err == nil {
				f(x, y, z)
			}
		}
	})
	stw.canv.SetCallback(func(arg *iup.MouseMotion) {
		if stw.frame != nil {
			stw.dbuff.UpdateYAxis(&arg.Y)
			// Snapping
			stw.cdcanv.Foreground(cd.CD_YELLOW)
			if snap != nil {
				stw.cdcanv.FCircle(snap.Pcoord[0], snap.Pcoord[1], nodeSelectPixel)
			}
			n := stw.frame.PickNode(float64(arg.X), float64(arg.Y), float64(nodeSelectPixel))
			if n != nil {
				stw.cdcanv.FCircle(n.Pcoord[0], n.Pcoord[1], nodeSelectPixel)
				stw.SetCoord(n.Coord[0], n.Coord[1], n.Coord[2])
			}
			snap = n
			///
			switch statusKey(arg.Status) {
			case STATUS_CENTER:
				stw.MoveOrRotate(arg)
			}
		}
	})
}

// }}}

// GET1NODE // {{{
func get1node(stw *Window, f func(n *st.Node)) {
	stw.canv.SetAttribute("CURSOR", "CROSS")
	var snap *st.Node // for Snapping
	stw.cdcanv.Foreground(cd.CD_YELLOW)
	stw.cdcanv.WriteMode(cd.CD_XOR)
	stw.SelectNode(make([]*st.Node, 1))
	setnnum := func() {
		nnum, err := strconv.ParseInt(stw.cline.GetAttribute("VALUE"), 10, 64)
		if err == nil {
			if n, ok := stw.frame.Nodes[int(nnum)]; ok {
				f(n)
			}
		}
		stw.cline.SetAttribute("VALUE", "")
	}
	stw.canv.SetCallback(func(arg *iup.MouseButton) {
		if stw.frame != nil {
			stw.dbuff.UpdateYAxis(&arg.Y)
			switch arg.Button {
			case BUTTON_LEFT:
				if arg.Pressed == 0 { // Released
					if n := stw.frame.PickNode(float64(arg.X), float64(arg.Y), float64(nodeSelectPixel)); n != nil {
						f(n)
					}
					// stw.cdcanv.Line(int(stw.selectNode[0].Pcoord[0]), int(stw.selectNode[0].Pcoord[1]), stw.endX, stw.endY) // TODO
					stw.Redraw()
				}
			case BUTTON_CENTER:
				if arg.Pressed == 0 { // Released
					// if stw.selectNode[0] != nil { // TODO
					//     stw.cdcanv.Line(int(stw.selectNode[0].Pcoord[0]), int(stw.selectNode[0].Pcoord[1]), stw.endX, stw.endY)
					// }
					stw.Redraw()
				} else { // Pressed
					if isDouble(arg.Status) {
						stw.frame.SetFocus(nil)
						stw.RedrawNode()
						stw.ShowCenter()
					} else {
						stw.startX = int(arg.X)
						stw.startY = int(arg.Y)
					}
				}
			case BUTTON_RIGHT:
				if arg.Pressed == 0 {
					if v := stw.cline.GetAttribute("VALUE"); v != "" {
						setnnum()
					} else {
						// fmt.Println("CONTEXT MENU")
						stw.EscapeAll()
					}
				}
			}
		}
	})
	stw.canv.SetCallback(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		switch key.Key() {
		default:
			stw.DefaultKeyAny(arg)
		case KEY_BS:
			val := stw.cline.GetAttribute("VALUE")
			if val != "" {
				stw.cline.SetAttribute("VALUE", val[:len(val)-1])
			}
		case KEY_ENTER:
			setnnum()
		case KEY_ESCAPE:
			stw.EscapeAll()
		case 'D', 'd':
			x, y, z, err := stw.QueryCoord("GET COORD")
			if err == nil {
				n, _ := stw.frame.CoordNode(x, y, z, EPS)
				f(n)
			}
		}
	})
	stw.canv.SetCallback(func(arg *iup.MouseMotion) {
		if stw.frame != nil {
			stw.dbuff.UpdateYAxis(&arg.Y)
			// Snapping
			stw.cdcanv.Foreground(cd.CD_YELLOW)
			if snap != nil {
				stw.cdcanv.FCircle(snap.Pcoord[0], snap.Pcoord[1], nodeSelectPixel)
			}
			n := stw.frame.PickNode(float64(arg.X), float64(arg.Y), float64(nodeSelectPixel))
			if n != nil {
				stw.cdcanv.FCircle(n.Pcoord[0], n.Pcoord[1], nodeSelectPixel)
				stw.SetCoord(n.Coord[0], n.Coord[1], n.Coord[2])
			}
			snap = n
			///
			switch statusKey(arg.Status) {
			case STATUS_CENTER:
				stw.MoveOrRotate(arg)
			}
		}
	})
}

// }}}

// TOGGLEBOND// {{{
func togglebond(stw *Window) {
	get1node(stw, func(n *st.Node) {
		if !stw.ElemSelected() {
			return
		}
		for _, el := range stw.SelectedElems() {
			if el == nil {
				continue
			}
			el.ToggleBond(n.Num)
			// if err == nil { break }
		}
		st.Snapshot(stw)
	})
}

// }}}

// GET2NODES // {{{
// DISTS: TODO: When button is released, tail line remains. When 2nd node is selected in command line, tail line remains.
func get2nodes(stw *Window, f func(n *st.Node), fdel func()) {
	var snap *st.Node // for Snapping
	stw.canv.SetAttribute("CURSOR", "CROSS")
	stw.cdcanv.Foreground(cd.CD_YELLOW)
	stw.cdcanv.WriteMode(cd.CD_XOR)
	stw.SelectNode(make([]*st.Node, 2))
	stw.addHistory("始端を指定[ダイアログ(D,R)]")
	setnnum := func() {
		nnum, err := strconv.ParseInt(stw.cline.GetAttribute("VALUE"), 10, 64)
		if err == nil {
			if n, ok := stw.frame.Nodes[int(nnum)]; ok {
				if stw.SelectedNodes()[0] != nil {
					f(n)
				} else {
					stw.SelectNode([]*st.Node{n})
					stw.cdcanv.Foreground(cd.CD_DARK_RED)
					stw.cdcanv.WriteMode(cd.CD_XOR)
					first = 1
					stw.addHistory("終端を指定[ダイアログ(D,R)]")
				}
			}
		}
		stw.cline.SetAttribute("VALUE", "")
	}
	stw.canv.SetCallback(func(arg *iup.MouseButton) {
		if stw.frame != nil {
			stw.dbuff.UpdateYAxis(&arg.Y)
			switch arg.Button {
			case BUTTON_LEFT:
				if arg.Pressed == 0 { // Released
					if stw.SelectedNodes()[0] != nil {
						if n := stw.frame.PickNode(float64(arg.X), float64(arg.Y), float64(nodeSelectPixel)); n != nil {
							f(n)
						}
						// stw.cdcanv.Line(int(stw.selectNode[0].Pcoord[0]), int(stw.selectNode[0].Pcoord[1]), stw.endX, stw.endY) // TODO
					} else {
						if n := stw.frame.PickNode(float64(arg.X), float64(arg.Y), float64(nodeSelectPixel)); n != nil {
							stw.SelectNode([]*st.Node{n})
							stw.cdcanv.Foreground(cd.CD_DARK_RED)
							stw.cdcanv.WriteMode(cd.CD_XOR)
							first = 1
							stw.addHistory("終端を指定[ダイアログ(D,R)]")
						}
					}
					stw.Redraw()
				}
			case BUTTON_CENTER:
				if arg.Pressed == 0 { // Released
					// if stw.selectNode[0] != nil { // TODO
					//     stw.cdcanv.Line(int(stw.selectNode[0].Pcoord[0]), int(stw.selectNode[0].Pcoord[1]), stw.endX, stw.endY)
					// }
					stw.Redraw()
				} else { // Pressed
					if isDouble(arg.Status) {
						stw.frame.SetFocus(nil)
						stw.RedrawNode()
						stw.ShowCenter()
					} else {
						stw.startX = int(arg.X)
						stw.startY = int(arg.Y)
					}
				}
			case BUTTON_RIGHT:
				if arg.Pressed == 0 {
					if v := stw.cline.GetAttribute("VALUE"); v != "" {
						setnnum()
					} else {
						// fmt.Println("CONTEXT MENU")
						stw.EscapeAll()
					}
				}
			}
		}
	})
	stw.canv.SetCallback(func(arg *iup.MouseMotion) {
		if stw.frame != nil {
			stw.dbuff.UpdateYAxis(&arg.Y)
			// Snapping
			stw.cdcanv.Foreground(cd.CD_YELLOW)
			if snap != nil {
				stw.cdcanv.FCircle(snap.Pcoord[0], snap.Pcoord[1], nodeSelectPixel)
			}
			n := stw.frame.PickNode(float64(arg.X), float64(arg.Y), float64(nodeSelectPixel))
			if n != nil {
				stw.cdcanv.FCircle(n.Pcoord[0], n.Pcoord[1], nodeSelectPixel)
				stw.SetCoord(n.Coord[0], n.Coord[1], n.Coord[2])
			}
			snap = n
			///
			switch statusKey(arg.Status) {
			case STATUS_CENTER:
				stw.MoveOrRotate(arg)
			}
			if n0 := stw.SelectedNodes()[0]; n0 != nil {
				stw.cdcanv.Foreground(cd.CD_DARK_RED)
				stw.TailLine(int(n0.Pcoord[0]), int(n0.Pcoord[1]), arg)
			}
		}
	})
	stw.canv.SetCallback(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		switch key.Key() {
		default:
			stw.DefaultKeyAny(arg)
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
		case 'D', 'd':
			x, y, z, err := stw.QueryCoord("GET COORD")
			if err == nil {
				n, _ := stw.frame.CoordNode(x, y, z, EPS)
				stw.Redraw()
				if stw.SelectedNodes()[0] != nil {
					f(n)
				} else {
					stw.SelectNode([]*st.Node{n})
					stw.cdcanv.Foreground(cd.CD_DARK_RED)
					stw.cdcanv.WriteMode(cd.CD_XOR)
					first = 1
					stw.addHistory("終端を指定[ダイアログ(D,R)]")
				}
			}
		case 'R', 'r':
			x, y, z, err := stw.QueryCoord("GET COORD")
			if err == nil {
				if n0 := stw.SelectedNodes()[0]; n0 != nil {
					n, _ := stw.frame.CoordNode(x+n0.Coord[0], y+n0.Coord[1], z+n0.Coord[2], EPS)
					stw.Redraw()
					f(n)
				} else {
					n, _ := stw.frame.CoordNode(x, y, z, EPS)
					stw.Redraw()
					stw.SelectNode([]*st.Node{n})
					stw.cdcanv.Foreground(cd.CD_DARK_RED)
					stw.cdcanv.WriteMode(cd.CD_XOR)
					first = 1
					stw.addHistory("終端を指定[ダイアログ(D,R)]")
				}
			}
		}
	})
}

// }}}

// DISTS// {{{
func dists(stw *Window) {
	get2nodes(stw, func(n *st.Node) {
		n0 := stw.SelectedNodes()[0]
		st.AddSelection(stw, n)
		dx, dy, dz, d := stw.frame.Distance(n0, n)
		stw.addHistory(fmt.Sprintf("NODE: %d - %d", n0.Num, n.Num))
		stw.addHistory(fmt.Sprintf("DX: %.3f DY: %.3f DZ: %.3f D: %.3f", dx, dy, dz, d))
		// stw.cdcanv.Foreground(cd.CD_WHITE)
		// stw.cdcanv.WriteMode(cd.CD_REPLACE)
		stw.EscapeAll()
	},
		func() {
			stw.SelectNode(make([]*st.Node, 2))
			stw.Redraw()
		})
}

func deformeddists(stw *Window) {
	get2nodes(stw, func(n *st.Node) {
		n0 := stw.SelectedNodes()[0]
		st.AddSelection(stw, n)
		dx, dy, dz, d := stw.frame.Distance(n0, n)
		stw.addHistory(fmt.Sprintf("NODE: %d - %d", n0.Num, n.Num))
		stw.addHistory(fmt.Sprintf("DX: %.3f DY: %.3f DZ: %.3f D: %.3f", dx, dy, dz, d))
		dx, dy, dz, d = stw.frame.DeformedDistance(n0, n)
		stw.addHistory(fmt.Sprintf("dx: %.3f dy: %.3f dz: %.3f d: %.3f", dx, dy, dz, d))
		// stw.cdcanv.Foreground(cd.CD_WHITE)
		// stw.cdcanv.WriteMode(cd.CD_REPLACE)
		stw.EscapeAll()
	},
		func() {
			stw.SelectNode(make([]*st.Node, 2))
			stw.Redraw()
		})
}

// }}}

func measure(stw *Window) {
	get2nodes(stw, func(n *st.Node) {
		n0 := stw.SelectedNodes()[0]
		st.AddSelection(stw, n)
		stw.addHistory("引き出し方向を指定[X, Y, Z, V]")
		createmeasure := func(direction string) {
			var u, v []float64
			switch direction {
			case "X":
				u = st.XAXIS
				v = st.YAXIS
			case "Y":
				u = st.YAXIS
				v = st.XAXIS
			case "Z":
				u = st.ZAXIS
			case "V":
				v = st.Direction(n0, n, true)
				u = st.Cross(v, st.ZAXIS)
			default:
				st.ErrorMessage(stw, errors.New("unknown direction"), st.ERROR)
				stw.EscapeAll()
				return
			}
			m := stw.frame.AddMeasure(n0.Coord, n.Coord, u)
			m.Text = fmt.Sprintf("%.0f", st.VectorDistance(n0, n, v)*1000)
			stw.EscapeAll()
		}
		stw.canv.SetCallback(func(arg *iup.CommonKeyAny) {
			key := iup.KeyState(arg.Key)
			switch key.Key() {
			default:
				stw.DefaultKeyAny(arg)
			case KEY_ESCAPE:
				stw.EscapeAll()
			case 'X', 'x':
				createmeasure("X")
			case 'Y', 'y':
				createmeasure("Y")
			case 'Z', 'z':
				createmeasure("Z")
			case 'V', 'v':
				createmeasure("V")
			}
		})
	},
		func() {
			stw.SelectNode(make([]*st.Node, 2))
			stw.Redraw()
		})
}

func kijun(stw *Window) {
	get2nodes(stw, func(n *st.Node) {
		n0 := stw.SelectedNodes()[0]
		st.AddSelection(stw, n)
		name, err := stw.Query("基準線名を入力")
		if err != nil {
			st.ErrorMessage(stw, err, st.ERROR)
			stw.EscapeAll()
			return
		}
		stw.frame.AddKijun(name, n0.Coord, n.Coord)
		stw.EscapeAll()
	},
		func() {
			stw.SelectNode(make([]*st.Node, 2))
			stw.Redraw()
		})
}

// ADDLINEELEM// {{{
func addlineelem(stw *Window) {
	get2nodes(stw, func(n *st.Node) {
		n0 := stw.SelectedNodes()[0]
		st.AddSelection(stw, n)
		sec := stw.frame.DefaultSect()
		el := stw.frame.AddLineElem(-1, stw.SelectedNodes()[:2], sec, st.NULL)
		stw.addHistory(fmt.Sprintf("ELEM: %d (ENOD: %d - %d, SECT: %d)", el.Num, n0.Num, n.Num, sec.Num))
		st.Snapshot(stw)
		// stw.cdcanv.Foreground(cd.CD_WHITE)
		// stw.cdcanv.WriteMode(cd.CD_REPLACE)
		stw.EscapeAll()
	},
		func() {
			stw.SelectNode(make([]*st.Node, 2))
			stw.Redraw()
		})
}

// }}}

// GETNNODES // {{{
// DISTS: TODO: When button is released, tail line remains. When 2nd node is selected in command line, tail line remains.
func getnnodes(stw *Window, maxnum int, f func(int)) {
	stw.canv.SetAttribute("CURSOR", "CROSS")
	var snap *st.Node // for Snapping
	stw.cdcanv.Foreground(cd.CD_YELLOW)
	stw.cdcanv.WriteMode(cd.CD_XOR)
	stw.SelectNode(make([]*st.Node, maxnum))
	selected := 0
	setnnum := func() {
		if selected >= maxnum {
			stw.addHistory("TOO MANY NODES SELECTED")
			return
		}
		nnum, err := strconv.ParseInt(stw.cline.GetAttribute("VALUE"), 10, 64)
		if err == nil {
			if n, ok := stw.frame.Nodes[int(nnum)]; ok {
				if stw.SelectedNodes()[0] != nil {
					st.AddSelection(stw, n)
					selected++
				} else {
					stw.SelectNode([]*st.Node{n})
					selected++
					stw.cdcanv.Foreground(cd.CD_DARK_RED)
					stw.cdcanv.WriteMode(cd.CD_XOR)
					first = 1
				}
			}
		}
		stw.cline.SetAttribute("VALUE", "")
	}
	stw.canv.SetCallback(func(arg *iup.MouseButton) {
		if stw.frame != nil {
			stw.dbuff.UpdateYAxis(&arg.Y)
			switch arg.Button {
			case BUTTON_LEFT:
				if arg.Pressed == 0 { // Released
					if n := stw.frame.PickNode(float64(arg.X), float64(arg.Y), float64(nodeSelectPixel)); n != nil {
						if selected >= maxnum {
							stw.addHistory("TOO MANY NODES SELECTED")
						} else if stw.SelectedNodes()[0] != nil {
							st.AddSelection(stw, n)
							selected++
							// stw.cdcanv.Line(int(stw.selectNode[0].Pcoord[0]), int(stw.selectNode[0].Pcoord[1]), stw.endX, stw.endY) // TODO
						} else {
							stw.SelectNode([]*st.Node{n})
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
					// if stw.selectNode[0] != nil { // TODO
					//     stw.cdcanv.Line(int(stw.selectNode[0].Pcoord[0]), int(stw.selectNode[0].Pcoord[1]), stw.endX, stw.endY)
					// }
					stw.Redraw()
				} else { // Pressed
					if isDouble(arg.Status) {
						stw.frame.SetFocus(nil)
						stw.RedrawNode()
						stw.ShowCenter()
					} else {
						stw.startX = int(arg.X)
						stw.startY = int(arg.Y)
					}
				}
			case BUTTON_RIGHT:
				if arg.Pressed == 0 {
					if v := stw.cline.GetAttribute("VALUE"); v != "" {
						setnnum()
					} else {
						// fmt.Println("CONTEXT MENU")
						f(selected)
					}
				}
			}
		}
	})
	stw.canv.SetCallback(func(arg *iup.MouseMotion) {
		if stw.frame != nil {
			stw.dbuff.UpdateYAxis(&arg.Y)
			// Snapping
			stw.cdcanv.Foreground(cd.CD_YELLOW)
			if snap != nil {
				stw.cdcanv.FCircle(snap.Pcoord[0], snap.Pcoord[1], nodeSelectPixel)
			}
			n := stw.frame.PickNode(float64(arg.X), float64(arg.Y), float64(nodeSelectPixel))
			if n != nil {
				stw.cdcanv.FCircle(n.Pcoord[0], n.Pcoord[1], nodeSelectPixel)
				stw.SetCoord(n.Coord[0], n.Coord[1], n.Coord[2])
			}
			snap = n
			///
			switch statusKey(arg.Status) {
			case STATUS_CENTER:
				stw.MoveOrRotate(arg)
			}
			if n0 := stw.SelectedNodes()[0]; n0 != nil {
				if selected < 2 {
					stw.TailLine(int(n0.Pcoord[0]), int(n0.Pcoord[1]), arg)
				} else {
					stw.TailPolygon(stw.SelectedNodes(), arg)
				}
			}
		}
	})
	stw.canv.SetCallback(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		switch key.Key() {
		default:
			stw.DefaultKeyAny(arg)
		case KEY_BS:
			val := stw.cline.GetAttribute("VALUE")
			if val != "" {
				stw.cline.SetAttribute("VALUE", val[:len(val)-1])
			}
		case KEY_DELETE:
			stw.SelectNode(make([]*st.Node, maxnum))
			selected++
		case KEY_ENTER:
			if v := stw.cline.GetAttribute("VALUE"); v != "" {
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
func addplateelem(stw *Window) {
	maxnum := 4
	getnnodes(stw, maxnum, func(num int) {
		if num >= 3 {
			en := stw.SelectedNodes()[:num]
			sec := stw.frame.DefaultSect()
			el := stw.frame.AddPlateElem(-1, en, sec, st.NULL)
			var buf bytes.Buffer
			buf.WriteString(fmt.Sprintf("ELEM: %d (ENOD: ", el.Num))
			for _, n := range en {
				buf.WriteString(fmt.Sprintf("%d ", n.Num))
			}
			buf.WriteString(fmt.Sprintf(", SECT: %d)", sec.Num))
			stw.addHistory(buf.String())
			st.Snapshot(stw)
		}
		stw.EscapeAll()
	})
}

func addplateelembyline(stw *Window) {
	getnelems(stw, 2, func(size int) {
		els := make([]*st.Elem, 2)
		num := 0
		for _, el := range stw.SelectedElems() {
			if el != nil && el.IsLineElem() {
				els[num] = el
				num++
				if num >= 2 {
					break
				}
			}
		}
		if num == 2 {
			ns := make([]*st.Node, 4)
			ns[0] = els[0].Enod[0]
			ns[1] = els[0].Enod[1]
			_, cw1 := st.ClockWise(ns[0].Pcoord, ns[1].Pcoord, els[1].Enod[0].Pcoord)
			_, cw2 := st.ClockWise(ns[0].Pcoord, els[1].Enod[0].Pcoord, els[1].Enod[1].Pcoord)
			if cw1 == cw2 {
				ns[2] = els[1].Enod[0]
				ns[3] = els[1].Enod[1]
			} else {
				ns[2] = els[1].Enod[1]
				ns[3] = els[1].Enod[0]
			}
			sec := stw.frame.DefaultSect()
			el := stw.frame.AddPlateElem(-1, ns, sec, st.NULL)
			var buf bytes.Buffer
			buf.WriteString(fmt.Sprintf("ELEM: %d (ENOD: ", el.Num))
			for _, n := range ns {
				buf.WriteString(fmt.Sprintf("%d ", n.Num))
			}
			buf.WriteString(fmt.Sprintf(", SECT: %d)", sec.Num))
			stw.addHistory(buf.String())
			st.Snapshot(stw)
			stw.EscapeAll()
		}
	})
}

// }}}

// SEARCHELEM// {{{
func searchelem(stw *Window) {
	ns := stw.SelectedNodes()
	if ns != nil && len(ns) >= 1 {
		stw.SelectElem(stw.frame.SearchElem(ns...))
		stw.Redraw()
		stw.EscapeCB()
		return
	}
	stw.Deselect()
	iup.SetFocus(stw.canv)
	startsearch := func(n *st.Node) {
		stw.SelectElem(stw.frame.SearchElem(n))
		stw.addHistory(fmt.Sprintf("Select Element Using NODE %d", n.Num))
		stw.EscapeCB()
	}
	setnnum := func() {
		nnum, err := strconv.ParseInt(stw.cline.GetAttribute("VALUE"), 10, 64)
		if err == nil {
			if n, ok := stw.frame.Nodes[int(nnum)]; ok {
				startsearch(n)
			} else {
				stw.addHistory(fmt.Sprintf("NODE %d not found", nnum))
			}
		}
		stw.cline.SetAttribute("VALUE", "")
	}
	stw.canv.SetCallback(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		switch key.Key() {
		default:
			stw.DefaultKeyAny(arg)
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
	stw.canv.SetCallback(func(arg *iup.MouseButton) {
		if stw.frame != nil {
			stw.dbuff.UpdateYAxis(&arg.Y)
			switch arg.Button {
			case BUTTON_LEFT:
				if arg.Pressed == 0 { // Released
					if n := stw.frame.PickNode(float64(arg.X), float64(arg.Y), float64(nodeSelectPixel)); n != nil {
						startsearch(n)
					}
					stw.Redraw()
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
						stw.startX = int(arg.X)
						stw.startY = int(arg.Y)
					}
				}
			case BUTTON_RIGHT:
				if arg.Pressed == 0 {
					if v := stw.cline.GetAttribute("VALUE"); v != "" {
						setnnum()
					} else {
						stw.EscapeAll()
					}
				}
			}
		}
	})
} // }}}

// NODE <-> ELEM// {{{
func nodetoelemany(stw *Window) {
	stw.SelectElem(stw.frame.NodeToElemAny(stw.SelectedNodes()...))
	stw.EscapeCB()
}
func nodetoelemall(stw *Window) {
	stw.SelectElem(stw.frame.NodeToElemAll(stw.SelectedNodes()...))
	stw.EscapeCB()
}
func elemtonode(stw *Window) {
	stw.SelectNode(stw.frame.ElemToNode(stw.SelectedElems()...))
	stw.EscapeCB()
}
func connected(stw *Window) {
	ns := stw.SelectedNodes()
	if ns != nil && len(ns) >= 1 && ns[0] != nil {
		stw.SelectNode(stw.frame.LineConnected(ns[0]))
	}
	stw.EscapeCB()
}
func onnode(stw *Window) {
	stw.SelectNode(make([]*st.Node, 0))
	if stw.ElemSelected() {
		for _, el := range stw.SelectedElems() {
			for _, n := range el.OnNode(0, EPS) {
				st.AddSelection(stw, n)
			}
		}
	}
	stw.EscapeCB()
}

// }}}

// GETVECTOR// {{{
func getvector(stw *Window, f func(x, y, z float64)) {
	var snap *st.Node // for Snapping
	stw.canv.SetAttribute("CURSOR", "CROSS")
	stw.cdcanv.Foreground(cd.CD_YELLOW)
	stw.cdcanv.WriteMode(cd.CD_XOR)
	var startpoint *st.Node
	funcbynode := func(n *st.Node) {
		c := make([]float64, 3)
		for i := 0; i < 3; i++ {
			c[i] = n.Coord[i] - startpoint.Coord[i]
		}
		f(c[0], c[1], c[2])
	}
	setnnum := func() {
		nnum, err := strconv.ParseInt(stw.cline.GetAttribute("VALUE"), 10, 64)
		if err == nil {
			if n, ok := stw.frame.Nodes[int(nnum)]; ok {
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
	stw.canv.SetCallback(func(arg *iup.MouseButton) {
		if stw.frame != nil {
			stw.dbuff.UpdateYAxis(&arg.Y)
			switch arg.Button {
			case BUTTON_LEFT:
				if arg.Pressed == 0 { // Released
					if startpoint != nil {
						if n := stw.frame.PickNode(float64(arg.X), float64(arg.Y), float64(nodeSelectPixel)); n != nil {
							funcbynode(n)
						}
						// stw.cdcanv.Line(int(stw.selectNode[0].Pcoord[0]), int(stw.selectNode[0].Pcoord[1]), stw.endX, stw.endY) // TODO
					} else {
						if n := stw.frame.PickNode(float64(arg.X), float64(arg.Y), float64(nodeSelectPixel)); n != nil {
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
					// if stw.selectNode[0] != nil { // TODO
					//     stw.cdcanv.Line(int(stw.selectNode[0].Pcoord[0]), int(stw.selectNode[0].Pcoord[1]), stw.endX, stw.endY)
					// }
					stw.Redraw()
				} else { // Pressed
					if isDouble(arg.Status) {
						stw.frame.SetFocus(nil)
						stw.RedrawNode()
						stw.ShowCenter()
					} else {
						stw.startX = int(arg.X)
						stw.startY = int(arg.Y)
					}
				}
			case BUTTON_RIGHT:
				if arg.Pressed == 0 {
					if v := stw.cline.GetAttribute("VALUE"); v != "" {
						setnnum()
					} else {
						stw.EscapeAll()
					}
				}
			}
		}
	})
	stw.canv.SetCallback(func(arg *iup.MouseMotion) {
		if stw.frame != nil {
			stw.dbuff.UpdateYAxis(&arg.Y)
			// Snapping
			stw.cdcanv.Foreground(cd.CD_YELLOW)
			if snap != nil {
				stw.cdcanv.FCircle(snap.Pcoord[0], snap.Pcoord[1], nodeSelectPixel)
			}
			n := stw.frame.PickNode(float64(arg.X), float64(arg.Y), float64(nodeSelectPixel))
			if n != nil {
				stw.cdcanv.FCircle(n.Pcoord[0], n.Pcoord[1], nodeSelectPixel)
				stw.SetCoord(n.Coord[0], n.Coord[1], n.Coord[2])
			}
			snap = n
			///
			switch statusKey(arg.Status) {
			case STATUS_CENTER:
				stw.MoveOrRotate(arg)
			}
			if startpoint != nil {
				stw.cdcanv.Foreground(cd.CD_DARK_RED)
				stw.TailLine(int(startpoint.Pcoord[0]), int(startpoint.Pcoord[1]), arg)
			}
		}
	})
	stw.canv.SetCallback(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		switch key.Key() {
		default:
			stw.DefaultKeyAny(arg)
		case KEY_BS:
			val := stw.cline.GetAttribute("VALUE")
			if val != "" {
				stw.cline.SetAttribute("VALUE", val[:len(val)-1])
			}
		case KEY_ENTER:
			setnnum()
		case KEY_ESCAPE:
			stw.EscapeAll()
		case 'D', 'd':
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
} // }}}

// COPYELEM// {{{
func copyelem(stw *Window) {
	if !stw.ElemSelected() {
		return
	}
	stw.addHistory("移動距離を指定[ダイアログ(D)]")
	getvector(stw, func(x, y, z float64) {
		if !(x == 0.0 && y == 0.0 && z == 0.0) {
			for _, el := range stw.SelectedElems() {
				if el == nil || el.IsHidden(stw.frame.Show) || el.Lock {
					continue
				}
				el.Copy(x, y, z, EPS)
			}
			st.Snapshot(stw)
			stw.Redraw()
		}
	})
}

// }}}

// DUPLICATEELEM// {{{
func duplicateelem(stw *Window) {
	if !stw.ElemSelected() {
		return
	}
	newels := make([]*st.Elem, len(stw.SelectedElems()))
	enum := 0
	for _, el := range stw.SelectedElems() {
		if el == nil || el.IsHidden(stw.frame.Show) || el.Lock {
			continue
		}
		newel := el.Snapshot(stw.frame)
		newels[enum] = newel
		enum++
	}
	newels = newels[:enum]
	sort.Sort(st.ElemByNum{newels})
	for _, el := range newels {
		stw.frame.AddElem(-1, el)
	}
	stw.SelectElem(newels)
	st.Snapshot(stw)
	stw.EscapeCB()
	stw.Redraw()
}

// }}}

// MOVEELEM// {{{
func moveelem(stw *Window) {
	if !stw.ElemSelected() {
		return
	}
	stw.addHistory("移動距離を指定[ダイアログ(D)]")
	getvector(stw, func(x, y, z float64) {
		for _, el := range stw.SelectedElems() {
			if el == nil || el.IsHidden(stw.frame.Show) || el.Lock {
				continue
			}
			el.Move(x, y, z, EPS)
		}
		for _, n := range stw.frame.NodeNoReference() {
			stw.frame.DeleteNode(n.Num)
		}
		st.Snapshot(stw)
		stw.Redraw()
	})
}

// }}}

// MOVENODE// {{{
func movenode(stw *Window) {
	ns := make([]*st.Node, 0)
	if stw.NodeSelected() {
		for _, n := range stw.SelectedNodes() {
			if n != nil {
				ns = append(ns, n)
			}
		}
	}
	if stw.ElemSelected() {
		en := stw.frame.ElemToNode(stw.SelectedElems()...)
		var add bool
		for _, n := range en {
			add = true
			for _, m := range ns {
				if m == n {
					add = false
					break
				}
			}
			if add {
				ns = append(ns, n)
			}
		}
	}
	if len(ns) == 0 {
		return
	}
	stw.addHistory("移動距離を指定[ダイアログ(D)]")
	getvector(stw, func(x, y, z float64) {
		for _, n := range ns {
			if n == nil || n.IsHidden(stw.frame.Show) || n.Lock {
				continue
			}
			n.Move(x, y, z)
		}
		st.Snapshot(stw)
		stw.EscapeCB()
	})
}

// }}}

// MOVETOLINE// {{{
func movetoline(stw *Window) {
	fixed := 0
	ns := make([]*st.Node, 0)
	if stw.NodeSelected() {
		for _, n := range stw.SelectedNodes() {
			if n != nil {
				ns = append(ns, n)
			}
		}
	}
	if len(ns) == 0 {
		stw.addHistory("移動する節点がありません")
		stw.EscapeAll()
		return
	}
	stw.addHistory("直線を指定[Xを固定]")
	get2nodes(stw, func(n *st.Node) {
		sns := stw.SelectedNodes()
		sns = append(sns, n)
		for _, n := range ns {
			n.MoveToLine(sns[0], sns[1], fixed)
		}
		stw.cdcanv.Foreground(cd.CD_WHITE)
		stw.cdcanv.WriteMode(cd.CD_REPLACE)
		st.Snapshot(stw)
		stw.EscapeAll()
	},
		func() {
			stw.SelectNode(make([]*st.Node, 2))
			stw.Redraw()
		})
	stw.canv.SetCallback(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		switch key.Key() {
		default:
			stw.DefaultKeyAny(arg)
		case KEY_BS:
			val := stw.cline.GetAttribute("VALUE")
			if val != "" {
				stw.cline.SetAttribute("VALUE", val[:len(val)-1])
			}
		case KEY_DELETE:
			stw.SelectNode(make([]*st.Node, 2))
			stw.Redraw()
		case KEY_ESCAPE:
			stw.EscapeAll()
		case 'X', 'x':
			fixed = 0
			stw.addHistory("直線を指定[Xを固定]")
		case 'Y', 'y':
			fixed = 1
			stw.addHistory("直線を指定[Yを固定]")
		case 'Z', 'z':
			fixed = 2
			stw.addHistory("直線を指定[Zを固定]")
		}
	})
}

// }}}

// PINCHNODE TODO: UNDER CONSTRUCTION// {{{
func pinchnode(stw *Window) {
	var target *st.Node
	movefunc := func(node *st.Node, dx, dy float64, arg *iup.MouseButton) {
		node.Coord[2] += dy * 0.01
	}
	stw.canv.SetCallback(func(arg *iup.MouseButton) {
		stw.dbuff.UpdateYAxis(&arg.Y)
		switch arg.Button {
		case BUTTON_LEFT:
			if stw.frame != nil {
				if arg.Pressed == 0 { // Released
					if target != nil {
						movefunc(target, float64(int(arg.X)-stw.startX), float64(int(arg.Y)-stw.startY), arg)
						target = nil
						stw.Redraw()
					}
				} else { // Pressed
					target = stw.frame.PickNode(float64(arg.X), float64(arg.Y), float64(nodeSelectPixel))
					stw.startX = int(arg.X)
					stw.startY = int(arg.Y)
				}
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
					stw.startX = int(arg.X)
					stw.startY = int(arg.Y)
				}
			}
		}
	})
	stw.canv.SetCallback(func(arg *iup.MouseMotion) {
		stw.dbuff.UpdateYAxis(&arg.Y)
		switch statusKey(arg.Status) {
		case STATUS_LEFT:
			if target != nil {
				fmt.Println(target.Num, float64(int(arg.X)-stw.startX), float64(int(arg.Y)-stw.startY))
			}
		case STATUS_CENTER:
			stw.MoveOrRotate(arg)
		}
	})
}

// }}}

// ROTATE// {{{
func rotate(stw *Window) {
	ns := make([]*st.Node, 0)
	if stw.NodeSelected() {
		for _, n := range stw.SelectedNodes() {
			if n != nil {
				ns = append(ns, n)
			}
		}
	}
	if stw.ElemSelected() {
		en := stw.frame.ElemToNode(stw.SelectedElems()...)
		var add bool
		for _, n := range en {
			add = true
			for _, m := range ns {
				if m == n {
					add = false
					break
				}
			}
			if add {
				ns = append(ns, n)
			}
		}
	}
	if len(ns) == 0 {
		return
	}
	stw.addHistory("回転中心を選択")
	get1node(stw, func(n0 *st.Node) {
		stw.addHistory(fmt.Sprintf("NODE: %d", n0.Num))
		stw.addHistory("回転軸を選択[ダイアログ(D)]")
		getvector(stw, func(x, y, z float64) {
			stw.addHistory(fmt.Sprintf("X: %.3f, Y: %.3f, Z: %.3f", x, y, z))
			if !(x == 0.0 && y == 0.0 && z == 0.0) {
				stw.addHistory("回転角を指定[参照(R)]")
				rot := func(angle float64) {
					for _, n := range ns {
						if n == nil || n.IsHidden(stw.frame.Show) || n.Lock {
							continue
						}
						n.Rotate(n0.Coord, []float64{x, y, z}, angle)
					}
					st.Snapshot(stw)
				}
				stw.canv.SetCallback(func(arg *iup.CommonKeyAny) {
					key := iup.KeyState(arg.Key)
					switch key.Key() {
					default:
						stw.cline.SetAttribute("APPEND", string(key.Key()))
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
						rot(val * math.Pi / 180)
						stw.EscapeCB()
					case 'R', 'r':
						stw.addHistory("参照点を指定")
						getnnodes(stw, 3, func(num int) {
							if num < 3 {
								stw.EscapeCB()
								return
							}
							u := []float64{x, y, z}
							u = st.Normalize(u)
							v1 := make([]float64, 3)
							v2 := make([]float64, 3)
							sns := stw.SelectedNodes()
							for i := 0; i < 3; i++ {
								v1[i] = sns[1].Coord[i] - sns[0].Coord[i]
								v2[i] = sns[2].Coord[i] - sns[0].Coord[i]
							}
							uv1 := st.Dot(u, v1, 3)
							uv2 := st.Dot(u, v2, 3)
							v12 := st.Dot(v1, v2, 3)
							v11 := st.Dot(v1, v1, 3)
							v22 := st.Dot(v2, v2, 3)
							w := math.Sqrt(v11-uv1*uv1) * math.Sqrt(v22-uv2*uv2)
							if w != 0.0 {
								rot(math.Acos((v12 - uv1*uv2) / w))
							}
							stw.SelectNode(make([]*st.Node, 0))
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
func mirror(stw *Window) {
	if !stw.ElemSelected() {
		return
	}
	ns := stw.frame.ElemToNode(stw.SelectedElems()...)
	stw.addHistory("対称面を指定[1点又は3点]")
	maxnum := 3
	createmirrors := func(coord, vec []float64) {
		nmap := make(map[int]*st.Node, len(ns))
		for _, n := range ns {
			c := n.MirrorCoord(coord, vec)
			var created bool
			nmap[n.Num], created = stw.frame.CoordNode(c[0], c[1], c[2], EPS)
			if created {
				for i := 0; i < 6; i++ {
					nmap[n.Num].Conf[i] = n.Conf[i]
				}
			}
		}
		for _, el := range stw.SelectedElems() {
			newenod := make([]*st.Node, el.Enods)
			var add bool
			for i := 0; i < el.Enods; i++ {
				newenod[i] = nmap[el.Enod[i].Num]
				if !add && newenod[i] != el.Enod[i] {
					add = true
				}
			}
			if add {
				if el.IsLineElem() {
					e := stw.frame.AddLineElem(-1, newenod, el.Sect, el.Etype)
					for i := 0; i < 6*el.Enods; i++ {
						e.Bonds[i] = el.Bonds[i]
					}
				} else {
					stw.frame.AddPlateElem(-1, newenod, el.Sect, el.Etype)
				}
			}
		}
		st.Snapshot(stw)
	}
	getnnodes(stw, maxnum, func(num int) {
		switch num {
		default:
			stw.EscapeAll()
		case 1:
			stw.addHistory("法線方向を選択[ダイアログ(D)]")
			getvector(stw, func(x, y, z float64) {
				createmirrors(stw.SelectedNodes()[0].Coord, []float64{x, y, z})
				stw.EscapeAll()
			})
		case 3:
			v1 := make([]float64, 3)
			v2 := make([]float64, 3)
			sns := stw.SelectedNodes()
			for i := 0; i < 3; i++ {
				v1[i] = sns[1].Coord[i] - sns[0].Coord[i]
				v2[i] = sns[2].Coord[i] - sns[0].Coord[i]
			}
			vec := st.Cross(v1, v2)
			createmirrors(sns[0].Coord, vec)
			stw.EscapeAll()
		}
	})
}

// }}}

// DIVIDEATPLANE// {{{
func divideatplane(stw *Window) {
	if !stw.ElemSelected() {
		return
	}
	stw.addHistory("対称面を指定[1点又は3点]")
	maxnum := 3
	divide := func(n []float64, h float64) {
		for _, el := range stw.SelectedElems() {
			if !el.IsLineElem() {
				continue
			}
			d := el.Direction(true)
			l := el.Length()
			dot := st.Dot(d, n, 3)
			if dot == 0 {
				continue
			}
			val := st.Dot(n, el.Enod[0].Coord, 3)
			val2 := (h - val)/dot
			if val2 <= 0 || val2 >= l {
				continue
			}
			coord := make([]float64, 3)
			for i := 0; i < 3; i++ {
				coord[i] = el.Enod[0].Coord[i] + val2*d[i]
			}
			el.DivideAtCoord(coord[0], coord[1], coord[2], EPS)
		}
		st.Snapshot(stw)
	}
	getnnodes(stw, maxnum, func(num int) {
		switch num {
		default:
			stw.EscapeAll()
		case 1:
			stw.addHistory("法線方向を選択[ダイアログ(D)]")
			getvector(stw, func(x, y, z float64) {
				n := st.Normalize([]float64{x, y, z})
				h := st.Dot(n, stw.SelectedNodes()[0].Coord, 3)
				divide(n, h)
				stw.EscapeAll()
			})
		case 3:
			v1 := make([]float64, 3)
			v2 := make([]float64, 3)
			sns := stw.SelectedNodes()
			for i := 0; i < 3; i++ {
				v1[i] = sns[1].Coord[i] - sns[0].Coord[i]
				v2[i] = sns[2].Coord[i] - sns[0].Coord[i]
			}
			vec := st.Cross(v1, v2)
			n := st.Normalize(vec)
			a1 := st.Dot(n, sns[0].Coord, 3)
			divide(n, a1)
			stw.EscapeAll()
		}
	})
}

// }}}

// SCALE// {{{
func scale(stw *Window) {
	ns := make([]*st.Node, 0)
	if stw.NodeSelected() {
		for _, n := range stw.SelectedNodes() {
			if n != nil {
				ns = append(ns, n)
			}
		}
	}
	if stw.ElemSelected() {
		en := stw.frame.ElemToNode(stw.SelectedElems()...)
		var add bool
		for _, n := range en {
			add = true
			for _, m := range ns {
				if m == n {
					add = false
					break
				}
			}
			if add {
				ns = append(ns, n)
			}
		}
	}
	if len(ns) == 0 {
		return
	}
	stw.addHistory("原点を指定[ダイアログ(D)]")
	get1node(stw, func(n0 *st.Node) {
		tmp, err := stw.Query("倍率を指定")
		if err != nil {
			st.ErrorMessage(stw, err, st.ERROR)
			stw.EscapeCB()
		}
		val, err := strconv.ParseFloat(tmp, 64)
		if err != nil {
			st.ErrorMessage(stw, err, st.ERROR)
			stw.EscapeCB()
		}
		for _, n := range ns {
			if n == nil || n.IsHidden(stw.frame.Show) || n.Lock {
				continue
			}
			n.Scale(n0.Coord, val, val, val)
		}
		st.Snapshot(stw)
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
	st.SaveFile(stw, stw.frame.Path)
	stw.EscapeAll()
}

// }}}

// WEIGHTCOPY// {{{
func weightcopy(stw *Window) {
	wgt := filepath.Join(stw.Home(), "hogtxt.wgt")
	fn := st.Ce(stw.frame.Path, ".wgt")
	if !st.FileExists(fn) || stw.Yn("Copy Wgt", "上書きしますか") {
		err := st.CopyFile(wgt, fn)
		if err != nil {
			st.ErrorMessage(stw, err, st.ERROR)
			stw.EscapeAll()
			return
		}
		stw.addHistory(fmt.Sprintf("COPY: %s", fn))
		err = stw.frame.ReadWgt(fn)
		if err != nil {
			st.ErrorMessage(stw, err, st.ERROR)
		}
	}
	stw.EscapeAll()
}

// }}}

// READPGP// {{{
func readpgp(stw *Window) {
	if name, ok := iup.GetOpenFile(stw.Cwd(), "*.pgp"); ok {
		err := stw.ReadPgp(name)
		if err != nil {
			stw.addHistory("ReadPgp: Cannot Read st.pgp")
		} else {
			stw.addHistory(fmt.Sprintf("ReadPgp: Read %s", name))
		}
	}
	stw.EscapeAll()
}

// }}}

// INSERT// {{{
func insert(stw *Window) {
	if name, ok := iup.GetOpenFile("", ""); ok {
		get1node(stw, func(n *st.Node) {
			// TODO: 角度を指定
			err := stw.frame.ReadInp(name, n.Coord, 0.0, false)
			if err != nil {
				st.ErrorMessage(stw, err, st.ERROR)
			}
			st.Snapshot(stw)
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
	stw.frame.SetFocus(nil)
	stw.Redraw()
	stw.addHistory(fmt.Sprintf("FOCUS: %.1f %.1f %.1f", stw.frame.View.Focus[0], stw.frame.View.Focus[1], stw.frame.View.Focus[2]))
	stw.EscapeAll()
}

// }}}

// GET1ELEM// {{{
func get1elem(stw *Window, f func(*st.Elem, int, int), condition func(*st.Elem) bool, exitfunc func()) {
	stw.SelectElem(make([]*st.Elem, 1))
	selected := false
	stw.canv.SetCallback(func(arg *iup.MouseButton) {
		if stw.frame != nil {
			stw.dbuff.UpdateYAxis(&arg.Y)
			switch arg.Button {
			case BUTTON_LEFT:
				if arg.Pressed == 0 {
					if selected {
						if el := stw.frame.PickElem(float64(arg.X), float64(arg.Y), dotSelectPixel); el != nil {
							f(el, int(arg.X), int(arg.Y))
						}
					} else {
						if el := stw.frame.PickElem(float64(arg.X), float64(arg.Y), dotSelectPixel); el != nil {
							if condition(el) {
								stw.SelectElem([]*st.Elem{el})
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
						stw.frame.SetFocus(nil)
						stw.RedrawNode()
						stw.ShowCenter()
					} else {
						stw.startX = int(arg.X)
						stw.startY = int(arg.Y)
					}
				}
			case BUTTON_RIGHT:
				if arg.Pressed == 0 {
					exitfunc()
					st.Snapshot(stw)
					stw.EscapeAll()
				}
			}
		}
	})
	stw.canv.SetCallback(func(arg *iup.MouseMotion) {
		if stw.frame != nil {
			stw.dbuff.UpdateYAxis(&arg.Y)
			switch statusKey(arg.Status) {
			case STATUS_CENTER:
				stw.MoveOrRotate(arg)
			}
		}
	})
	stw.canv.SetCallback(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		switch key.Key() {
		default:
			stw.DefaultKeyAny(arg)
		case KEY_ESCAPE:
			exitfunc()
			st.Snapshot(stw)
			stw.EscapeAll()
		}
	})
}

// }}}

// GET1SIDE // {{{
func get1side(stw *Window, f func(*st.Elem, int, int), condition func(*st.Elem) bool, exitfunc func()) {
	stw.SelectElem(make([]*st.Elem, 1))
	selected := false
	stw.canv.SetCallback(func(arg *iup.MouseButton) {
		if stw.frame != nil {
			stw.dbuff.UpdateYAxis(&arg.Y)
			switch arg.Button {
			case BUTTON_LEFT:
				if arg.Pressed == 0 {
					if selected {
						f(stw.SelectedElems()[0], int(arg.X), int(arg.Y))
						selected = false
						stw.SelectElem(make([]*st.Elem, 1))
					} else {
						if el := stw.frame.PickElem(float64(arg.X), float64(arg.Y), EPS); el != nil {
							if condition(el) {
								stw.SelectElem([]*st.Elem{el})
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
						stw.frame.SetFocus(nil)
						stw.RedrawNode()
						stw.ShowCenter()
					} else {
						stw.startX = int(arg.X)
						stw.startY = int(arg.Y)
					}
				}
			case BUTTON_RIGHT:
				if arg.Pressed == 0 {
					exitfunc()
					st.Snapshot(stw)
					stw.EscapeAll()
				}
			}
		}
	})
	stw.canv.SetCallback(func(arg *iup.MouseMotion) {
		if stw.frame != nil {
			stw.dbuff.UpdateYAxis(&arg.Y)
			switch statusKey(arg.Status) {
			case STATUS_CENTER:
				stw.MoveOrRotate(arg)
			}
		}
	})
	stw.canv.SetCallback(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		switch key.Key() {
		default:
			stw.DefaultKeyAny(arg)
		case KEY_ESCAPE:
			exitfunc()
			st.Snapshot(stw)
			stw.EscapeAll()
		}
	})
}

// }}}

// MATCHPROP// {{{
func matchproperty(stw *Window) {
	get1elem(stw, func(el *st.Elem, x, y int) {
		el0 := stw.SelectedElems()[0]
		el.Sect = el0.Sect
		el.Etype = el0.Etype
		stw.Redraw()
	},
		func(el *st.Elem) bool {
			return true
		},
		func() {})
}

// }}}

// COPYBOND// {{{
func copybond(stw *Window) {
	get1elem(stw, func(el *st.Elem, x, y int) {
		if el.IsLineElem() {
			for i := 0; i < 12; i++ {
				el.Bonds[i] = stw.SelectedElems()[0].Bonds[i]
			}
		}
		stw.Redraw()
	},
		func(el *st.Elem) bool {
			return el.IsLineElem()
		},
		func() {})
}

// }}}

// AXISTOCANG// {{{
func axistocang(stw *Window) {
	if !stw.ElemSelected() {
		stw.EscapeAll()
		return
	}
	stw.addHistory("部材軸の方向を指定[強軸(S)/弱軸(W)/切替(T)/ダイアログ(D)]<強軸>")
	strong := true
	axisfunc := func(x, y, z float64) {
		if !(x == 0.0 && y == 0.0 && z == 0.0) {
			for _, el := range stw.SelectedElems() {
				if el == nil || el.IsHidden(stw.frame.Show) || el.Lock {
					continue
				}
				_, err := el.AxisToCang([]float64{x, y, z}, strong)
				if err != nil {
					stw.addHistory(fmt.Sprintf("部材軸を設定できません: ELEM %d", el.Num))
				}
			}
			st.Snapshot(stw)
			stw.EscapeAll()
		}
	}
	getvector(stw, axisfunc)
	stw.canv.SetCallback(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		switch key.Key() {
		default:
			stw.DefaultKeyAny(arg)
		case KEY_BS:
			val := stw.cline.GetAttribute("VALUE")
			if val != "" {
				stw.cline.SetAttribute("VALUE", val[:len(val)-1])
			}
		case KEY_ENTER:
		case KEY_ESCAPE:
			stw.EscapeAll()
		case 'D', 'd':
			x, y, z, err := stw.QueryCoord("GET COORD")
			if err == nil {
				axisfunc(x, y, z)
			}
		case 'S', 's':
			stw.addHistory("<強軸>方向を指定")
			strong = true
		case 'W', 'w':
			stw.addHistory("<弱軸>方向を指定")
			strong = false
		case 'T', 't':
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
func bondpin(stw *Window) {
	if !stw.ElemSelected() {
		stw.EscapeAll()
		return
	}
	for _, el := range stw.SelectedElems() {
		if el == nil || el.Lock {
			continue
		}
		if el.IsLineElem() {
			for i := 0; i < 2; i++ {
				for j := 4; j < 6; j++ {
					el.Bonds[6*i+j] = st.Pin
				}
			}
		}
	}
	st.Snapshot(stw)
	stw.EscapeAll()
}

func bondrigid(stw *Window) {
	if !stw.ElemSelected() {
		stw.EscapeAll()
		return
	}
	for _, el := range stw.SelectedElems() {
		if el == nil || el.Lock {
			continue
		}
		if el.IsLineElem() {
			for i := 0; i < 2; i++ {
				for j := 0; j < 6; j++ {
					el.Bonds[6*i+j] = nil
				}
			}
		}
	}
	st.Snapshot(stw)
	stw.EscapeAll()
}

// }}}

// CONF// {{{
func setconf(stw *Window, lis []bool) {
	if !stw.NodeSelected() {
		stw.EscapeAll()
		return
	}
	for _, n := range stw.SelectedNodes() {
		if n == nil || n.Lock {
			continue
		}
		for i := 0; i < 6; i++ {
			n.Conf[i] = lis[i]
		}
	}
	st.Snapshot(stw)
	stw.EscapeCB()
}

// CONFFIX
func conffix(stw *Window) {
	if !stw.NodeSelected() {
		stw.EscapeAll()
		return
	}
	for _, n := range stw.SelectedNodes() {
		if n == nil || n.Lock {
			continue
		}
		for i := 0; i < 3; i++ {
			n.Conf[i] = true
			n.Conf[i+3] = true
		}
	}
	st.Snapshot(stw)
	stw.EscapeCB()
}

// CONFPIN
func confpin(stw *Window) {
	if !stw.NodeSelected() {
		stw.EscapeAll()
		return
	}
	for _, n := range stw.SelectedNodes() {
		if n == nil || n.Lock {
			continue
		}
		for i := 0; i < 3; i++ {
			n.Conf[i] = true
			n.Conf[i+3] = false
		}
	}
	st.Snapshot(stw)
	stw.EscapeCB()
}

// CONFXYROLLER
func confxyroller(stw *Window) {
	if !stw.NodeSelected() {
		stw.EscapeAll()
		return
	}
	for _, n := range stw.SelectedNodes() {
		if n == nil || n.Lock {
			continue
		}
		for i := 0; i < 3; i++ {
			n.Conf[i] = false
			n.Conf[i+3] = false
		}
		n.Conf[2] = true
	}
	st.Snapshot(stw)
	stw.EscapeCB()
}

// CONFFREE
func conffree(stw *Window) {
	if !stw.NodeSelected() {
		stw.EscapeAll()
		return
	}
	for _, n := range stw.SelectedNodes() {
		if n == nil || n.Lock {
			continue
		}
		for i := 0; i < 3; i++ {
			n.Conf[i] = false
			n.Conf[i+3] = false
		}
		if n.Pile != nil {
			n.Pile = nil
		}
	}
	st.Snapshot(stw)
	stw.EscapeCB()
}

// }}}

// TRIM// {{{
func trim(stw *Window) {
	get1elem(stw, func(el *st.Elem, x, y int) {
		if el.IsLineElem() {
			var err error
			el0 := stw.SelectedElems()[0]
			if st.DotLine(el0.Enod[0].Pcoord[0], el0.Enod[0].Pcoord[1], el0.Enod[1].Pcoord[0], el0.Enod[1].Pcoord[1], float64(x), float64(y))*st.DotLine(el0.Enod[0].Pcoord[0], el0.Enod[0].Pcoord[1], el0.Enod[1].Pcoord[0], el0.Enod[1].Pcoord[1], el.Enod[0].Pcoord[0], el.Enod[0].Pcoord[1]) < 0.0 {
				_, _, err = stw.frame.Trim(el0, el, 1, EPS)
			} else {
				_, _, err = stw.frame.Trim(el0, el, -1, EPS)
			}
			if err != nil {
				st.ErrorMessage(stw, err, st.ERROR)
			} else {
				stw.Redraw()
			}
		}
		stw.Redraw()
	},
		func(el *st.Elem) bool {
			return el.IsLineElem()
		},
		func() {
			if stw.ElemSelected() {
				if el0 := stw.SelectedElems()[0]; el0 != nil {
					el0.DivideAtOns(EPS)
				}
			}
		})
}

// }}}

// EXTEND// {{{
func extend(stw *Window) {
	get1elem(stw, func(el *st.Elem, x, y int) {
		if el.IsLineElem() {
			var err error
			_, _, err = stw.frame.Extend(stw.SelectedElems()[0], el, EPS)
			if err != nil {
				st.ErrorMessage(stw, err, st.ERROR)
			} else {
				stw.Redraw()
			}
		}
		stw.Redraw()
	},
		func(el *st.Elem) bool {
			return el.IsLineElem()
		},
		func() {
			if stw.ElemSelected() {
				if el0 := stw.SelectedElems()[0]; el0 != nil {
					el0.DivideAtOns(EPS)
				}
			}
		})
}

// }}}

// OFFSET// {{{
func offset(stw *Window) {
	tmp, err := stw.Query("オフセット距離を指定")
	if err != nil {
		st.ErrorMessage(stw, err, st.ERROR)
		stw.EscapeCB()
		return
	}
	val, err := strconv.ParseFloat(tmp, 64)
	if err != nil {
		st.ErrorMessage(stw, err, st.ERROR)
		stw.EscapeCB()
		return
	}
	angle := 0.0
	tmp, err = stw.Query("オフセット方向を指定[°]")
	if err == nil {
		a, err := strconv.ParseFloat(tmp, 64)
		if err == nil {
			angle = a * math.Pi / 180.0
		}
	}
	get1side(stw, func(el *st.Elem, x, y int) {
		if el.IsLineElem() {
			st0 := make([]float64, 3)
			mid := el.MidPoint()
			c := math.Cos(angle)
			s := math.Sin(angle)
			for i := 0; i < 3; i++ {
				st0[i] = mid[i] + el.Strong[i]*c + el.Weak[i]*s
			}
			st1 := el.Frame.View.ProjectCoord(st0)
			if st.DotLine(el.Enod[0].Pcoord[0], el.Enod[0].Pcoord[1], el.Enod[1].Pcoord[0], el.Enod[1].Pcoord[1], float64(x), float64(y))*st.DotLine(el.Enod[0].Pcoord[0], el.Enod[0].Pcoord[1], el.Enod[1].Pcoord[0], el.Enod[1].Pcoord[1], st1[0], st1[1]) < 0.0 {
				el.Offset(-val, angle, EPS)
			} else {
				el.Offset(val, angle, EPS)
			}
			stw.Redraw()
		}
	},
		func(el *st.Elem) bool {
			return el.IsLineElem()
		},
		func() {})
}

// }}}

// SELECTNODE// {{{
func selectnode(stw *Window) {
	stw.Deselect()
	iup.SetFocus(stw.canv)
	setnnum := func() {
		nnum, err := strconv.ParseInt(stw.cline.GetAttribute("VALUE"), 10, 64)
		if err == nil {
			if n, ok := stw.frame.Nodes[int(nnum)]; ok {
				stw.addHistory(fmt.Sprintf("NODE %d Selected", nnum))
				stw.SelectNode([]*st.Node{n})
				stw.EscapeCB()
			} else {
				stw.addHistory(fmt.Sprintf("NODE %d not found", nnum))
			}
		}
		stw.cline.SetAttribute("VALUE", "")
	}
	stw.canv.SetCallback(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		switch key.Key() {
		default:
			stw.cline.SetAttribute("APPEND", string(key.Key()))
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
} // }}}

// SELECTCOLUMNBASE// {{{
func selectcolumnbase(stw *Window) {
	stw.Deselect()
	var min, max float64
	nnum := 0
	ns := make([]*st.Node, len(stw.frame.Nodes))
	if len(stw.frame.Ai.Boundary) >= 2 {
		min = stw.frame.Ai.Boundary[0]
		max = stw.frame.Ai.Boundary[1]
	} else {
		min = 0.0
		max = 0.0
	}
	for _, el := range stw.frame.Elems {
		if el.Etype != st.COLUMN || el.IsHidden(stw.frame.Show) {
			continue
		}
		for _, n := range el.Enod {
			if min <= n.Coord[2] && n.Coord[2] <= max {
				ns[nnum] = n
				nnum++
			}
		}
	}
	stw.SelectNode(ns[:nnum])
	stw.EscapeCB()
}

// }}}

// SELECTCONFED// {{{
func selectconfed(stw *Window) {
	stw.Deselect()
	ns := make([]*st.Node, len(stw.frame.Nodes))
	num := 0
	for _, n := range stw.frame.Nodes {
		if n.IsHidden(stw.frame.Show) {
			continue
		}
		for i := 0; i < 6; i++ {
			if n.Conf[i] {
				ns = append(ns, n)
				num++
				break
			}
		}
	}
	stw.SelectNode(ns[:num])
	stw.EscapeCB()
}

// }}}

// SELECTELEM// {{{
func selectelem(stw *Window) {
	stw.Deselect()
	iup.SetFocus(stw.canv)
	setenum := func() {
		enum, err := strconv.ParseInt(stw.cline.GetAttribute("VALUE"), 10, 64)
		if err == nil {
			if el, ok := stw.frame.Elems[int(enum)]; ok {
				stw.addHistory(fmt.Sprintf("ELEM %d Selected", enum))
				stw.SelectElem([]*st.Elem{el})
				stw.EscapeCB()
			} else {
				stw.addHistory(fmt.Sprintf("ELEM %d not found", enum))
			}
		}
		stw.cline.SetAttribute("VALUE", "")
	}
	stw.canv.SetCallback(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		switch key.Key() {
		default:
			stw.cline.SetAttribute("APPEND", string(key.Key()))
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
} // }}}

// SELECTSECT// {{{
func selectsect(stw *Window) {
	stw.frame.Show.ElemCaption |= st.EC_SECT
	stw.Labels["EC_SECT"].SetAttribute("FGCOLOR", labelFGColor)
	stw.Deselect()
	stw.Redraw()
	iup.SetFocus(stw.canv)
	setsnum := func() {
		tmp, err := strconv.ParseInt(stw.cline.GetAttribute("VALUE"), 10, 64)
		if err == nil {
			snum := int(tmp)
			if _, ok := stw.frame.Sects[snum]; ok {
				stw.addHistory(fmt.Sprintf("SECT %d Selected", snum))
				stw.SelectElem(make([]*st.Elem, 0))
				for _, el := range stw.frame.Elems {
					if el.IsHidden(stw.frame.Show) {
						continue
					}
					if el.Sect.Num == snum {
						st.AddSelection(stw, el)
					}
				}
				stw.EscapeCB()
			} else {
				stw.addHistory(fmt.Sprintf("SECT %d not found", snum))
			}
		}
		stw.cline.SetAttribute("VALUE", "")
	}
	stw.canv.SetCallback(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		switch key.Key() {
		default:
			stw.cline.SetAttribute("APPEND", string(key.Key()))
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
func hidesection(stw *Window) {
	hide := func() {
		tmp, err := strconv.ParseInt(stw.cline.GetAttribute("VALUE"), 10, 64)
		if err == nil {
			snum := int(tmp)
			stw.frame.Show.Sect[snum] = false
			stw.Redraw()
		}
		stw.cline.SetAttribute("VALUE", "")
	}
	stw.canv.SetCallback(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		switch key.Key() {
		default:
			stw.cline.SetAttribute("APPEND", string(key.Key()))
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
func hidecurtainwall(stw *Window) {
	for _, sec := range stw.frame.Sects {
		if sec.Num > 900 {
			continue
		}
		if sec.HasArea(0) {
			continue
		}
		if !sec.HasBrace() {
			stw.frame.Show.Sect[sec.Num] = false
		}
	}
	stw.EscapeAll()
}

// }}}

// SELECTCHILDREN// {{{
func selectchildren(stw *Window) {
	getnelems(stw, 10, func(size int) {
		els := make([]*st.Elem, 2)
		num := 0
		for _, el := range stw.SelectedElems() {
			if el != nil && !el.IsLineElem() {
				els[num] = el
				num++
			}
		}
		if num > 0 {
			tmpels := make([]*st.Elem, num*2)
			nc := 0
			for _, el := range stw.SelectedElems() {
				if el.Children != nil {
					for _, c := range el.Children {
						if c != nil {
							tmpels[nc] = c
							nc++
						}
					}
				}
			}
			stw.SelectElem(tmpels[:nc])
			stw.EscapeCB()
		}
	})
}

// }}}

// NODENOREFERENCE// {{{
func nodenoreference(stw *Window) {
	stw.Deselect()
	ns := stw.frame.NodeNoReference()
	if len(ns) != 0 {
		stw.SelectNode(ns)
		stw.Redraw()
		if stw.Yn("NODE NO REFERENCE", "不要な節点を削除しますか?") {
			for _, n := range ns {
				stw.frame.DeleteNode(n.Num)
			}
			st.Snapshot(stw)
		}
	}
	stw.EscapeAll()
}

// }}}

// ELEMSAMENODE// {{{
func elemsamenode(stw *Window) {
	stw.Deselect()
	els := stw.frame.ElemSameNode()
	if len(els) != 0 {
		stw.SelectNode(stw.frame.ElemToNode(els...))
		stw.SelectElem(els)
		stw.Redraw()
		if stw.Yn("ELEM SAME NODE", "部材を削除しますか?") {
			for _, el := range els {
				if el.Lock {
					continue
				}
				stw.frame.DeleteElem(el.Num)
			}
			st.Snapshot(stw)
			stw.EscapeAll()
		} else {
			stw.EscapeCB()
		}
	} else {
		stw.EscapeAll()
	}
}

// }}}

func suspicious(stw *Window) {
	stw.Deselect()
	ns, els, err := stw.frame.Suspicious()
	if err != nil {
		stw.SelectNode(ns)
		stw.SelectElem(els)
		st.ErrorMessage(stw, err, st.ERROR)
	}
	stw.EscapeCB()
}

// PRUNEENOD// {{{
func pruneenod(stw *Window) {
	stw.Deselect()
	tmpels := stw.frame.ElemSameNode()
	l := len(tmpels)
	if l != 0 {
		els := make([]*st.Elem, l)
		enum := 0
		for _, el := range tmpels {
			if !el.IsLineElem() {
				els[enum] = el
				enum++
			}
		}
		stw.SelectElem(els[:enum])
		stw.Redraw()
		if stw.Yn("ELEM SAME NODE", "部材の不要なENODを削除しますか?") {
			for _, el := range els[:enum] {
				if el.Lock {
					continue
				}
				el.PruneEnod()
			}
			st.Snapshot(stw)
		}
	}
	stw.EscapeAll()
}

// }}}

// NODEDUPLICATION// {{{
func nodeduplication(stw *Window) {
	stw.Deselect()
	nm := stw.frame.NodeDuplication(EPS)
	if len(nm) != 0 {
		for k := range nm {
			st.AddSelection(stw, k)
		}
		stw.Redraw()
		if stw.Yn("NODE DUPLICATION", "重なった節点を削除しますか?") {
			stw.frame.ReplaceNode(nm)
			st.Snapshot(stw)
		} else {
			stw.EscapeCB()
		}
	} else {
		stw.EscapeAll()
	}
}

// }}}

// ELEMDUPLICATION// {{{
func elemduplication(stw *Window) {
	stw.Deselect()
	els := stw.frame.ElemDuplication(nil)
	if len(els) != 0 {
		for k := range els {
			st.AddSelection(stw, k)
		}
		stw.Redraw()
		if stw.Yn("ELEM DUPLICATION", "重なった部材を削除しますか?") {
			for el := range els {
				if el.Lock {
					continue
				}
				stw.frame.DeleteElem(el.Num)
			}
			st.Snapshot(stw)
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
func checkframe(stw *Window) {
	stw.Deselect()
	nodenoreference(stw)
	nodeduplication(stw)
	elemsamenode(stw)
	elemduplication(stw)
	eall := true
	ns, els, ok := stw.frame.Check()
	if !ok {
		stw.SelectNode(ns)
		stw.SelectElem(els)
		stw.Redraw()
		if stw.Yn("CHECK FRAME", "無効な節点と部材を削除しますか？") {
			for _, n := range els {
				if n.Lock {
					continue
				}
				stw.frame.DeleteNode(n.Num)
			}
			for _, el := range els {
				if el.Lock {
					continue
				}
				stw.frame.DeleteElem(el.Num)
			}
		} else {
			eall = false
		}
	}
	if !stw.frame.IsUpside() {
		if stw.Yn("CHECK FRAME", "部材の向きを修正しますか？") {
			stw.frame.Upside()
		}
		st.Snapshot(stw)
	}
	if eall {
		stw.EscapeAll()
	} else {
		stw.EscapeCB()
	}
}

// }}}

// NODESORT// {{{
func nodesort(stw *Window) {
	bw := stw.frame.BandWidth()
	stw.addHistory(fmt.Sprintf("並び替え前: %d", bw))
	ns := func(d int) {
		bw, err := stw.frame.NodeSort(d)
		if err != nil {
			stw.addHistory("並び替えエラー")
			stw.EscapeAll()
		}
		stw.addHistory(fmt.Sprintf("並び替え後: %d (%s方向)", bw, []string{"X", "Y", "Z"}[d]))
		st.Snapshot(stw)
		stw.Redraw()
	}
	stw.canv.SetCallback(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		switch key.Key() {
		default:
			stw.DefaultKeyAny(arg)
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
func extractarclm(stw *Window) {
	var err error
	var name string
	var ok bool
	saved := true
	stw.frame.ExtractArclm("")
	ans := stw.Yna("Extract Arclm", ".inl, .ihx, .ihyを保存しますか?", "別名で保存")
	switch ans {
	default:
		saved = false
	case 1:
		name = stw.frame.Name
		err = stw.frame.SaveAsArclm("")
		if err != nil {
			saved = false
		}
	case 3:
		name, ok = iup.GetSaveFile("", "")
		if ok {
			err = stw.frame.SaveAsArclm(name)
			if err != nil {
				saved = false
			}
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

// FENCE// {{{
func fence(stw *Window) {
	iup.SetFocus(stw.canv)
	stw.canv.SetCallback(func(arg *iup.MouseButton) {
		if stw.frame != nil {
			switch arg.Button {
			case BUTTON_LEFT:
				stw.SelectElemFenceStart(arg)
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
			}
		}
	})
	stw.canv.SetCallback(func(arg *iup.MouseMotion) {
		if stw.frame != nil {
			stw.dbuff.UpdateYAxis(&arg.Y)
			switch statusKey(arg.Status) {
			case STATUS_LEFT:
				stw.SelectElemFenceMotion(arg)
			case STATUS_CENTER:
				stw.MoveOrRotate(arg)
			}
		}
	})
}

// }}}

// ERRORELEM// {{{
func errorelem(stw *Window) {
	iup.SetFocus(stw.canv)
	stw.Deselect()
	stw.SetColorMode(st.ECOLOR_RATE)
	st.SrcanRateOn(stw)
	stw.Redraw()
	tmpels := make([]*st.Elem, len(stw.frame.Elems))
	i := 0
	for _, el := range stw.frame.Elems {
		switch el.Etype {
		case st.COLUMN, st.GIRDER, st.BRACE, st.WALL, st.SLAB:
			val, err := el.RateMax(stw.frame.Show)
			if err != nil {
				continue
			}
			if val > 1.0 {
				tmpels[i] = el
				i++
			}
		}
	}
	stw.SelectElem(tmpels[:i])
	stw.EscapeCB()
}

// }}}

// SHOWPLANE
func showplane(stw *Window) {
	stw.addHistory("面を指定[3点]")
	maxnum := 3
	getnnodes(stw, maxnum, func(num int) {
		if num >= 3 {
			sns := stw.SelectedNodes()
			err := stw.frame.ShowPlane(sns[0], sns[1], sns[2], EPS)
			if err != nil {
				st.ErrorMessage(stw, err, st.ERROR)
			}
			stw.EscapeAll()
		}
	})
}

// CUTTER TODO: UNDER CONSTRUCTION // {{{
func cutter(stw *Window) {
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
	if err != nil {
		return
	}
	stw.frame.Cutter(axis, coord, EPS)
	stw.EscapeAll()
}

// }}}

// DIVIDE// {{{
func divide(stw *Window, divfunc func(*st.Elem) ([]*st.Node, []*st.Elem, error)) {
	if stw.ElemSelected() {
		tmpels := make([]*st.Elem, 0)
		for _, el := range stw.SelectedElems() {
			_, els, err := divfunc(el)
			if err != nil {
				st.ErrorMessage(stw, err, st.ERROR)
				continue
			}
			if err == nil && len(els) > 1 {
				tmpels = append(tmpels, els...)
			}
		}
		stw.SelectElem(tmpels)
		st.Snapshot(stw)
		stw.EscapeCB()
	}
}
func divideatons(stw *Window) {
	divide(stw, func(el *st.Elem) ([]*st.Node, []*st.Elem, error) {
		return el.DivideAtOns(EPS)
	})
}
func divideatmid(stw *Window) {
	divide(stw, func(el *st.Elem) ([]*st.Node, []*st.Elem, error) {
		return el.DivideAtMid(EPS)
	})
}
func divideinn(stw *Window) {
	ans, err := stw.Query("分割数")
	if err != nil {
		st.ErrorMessage(stw, err, st.ERROR)
		stw.EscapeCB()
		return
	}
	val, err := strconv.ParseInt(ans, 10, 64)
	if err != nil {
		st.ErrorMessage(stw, err, st.ERROR)
		stw.EscapeCB()
		return
	}
	n := int(val)
	divide(stw, func(el *st.Elem) ([]*st.Node, []*st.Elem, error) {
		return el.DivideInN(n, EPS)
	})
}
func divideatelem(stw *Window) {
	divide(stw, func(el *st.Elem) ([]*st.Node, []*st.Elem, error) {
		els, err := el.DivideAtElem(EPS)
		return nil, els, err
	})
}

// }}}

// INTERSECT// {{{
func intersect(stw *Window) {
	getnelems(stw, 2, func(size int) {
		els := make([]*st.Elem, 2)
		num := 0
		for _, el := range stw.SelectedElems() {
			if el != nil {
				els[num] = el
				num++
				if num >= 2 {
					break
				}
			}
		}
		if num == 2 {
			_, els, err := stw.frame.Intersect(els[0], els[1], true, 1, 1, false, false, EPS)
			if err != nil {
				st.ErrorMessage(stw, err, st.ERROR)
				stw.Deselect()
			} else {
				stw.Deselect()
				switch len(els) {
				case 4:
					stw.SelectElem([]*st.Elem{els[1], els[3]})
				}
				stw.Redraw()
			}
			st.Snapshot(stw)
			stw.EscapeCB()
		}
	})
}

// }}}

// INTERSECTALL
func intersectall(stw *Window) {
	err := stw.frame.IntersectAll(stw.SelectedElems(), EPS)
	if err != nil {
		st.ErrorMessage(stw, err, st.ERROR)
		stw.EscapeAll()
		return
	}
	st.Snapshot(stw)
	stw.EscapeAll()
}

// INTERSECTALL2 // {{{
func intersectall2(stw *Window) {
	elems := stw.SelectedElems()
	l := len(elems)
	if elems == nil || l <= 1 {
		stw.EscapeAll()
		return
	}
	sort.Sort(st.ElemByNum{elems})
	ind := 0
	for {
		if elems[ind].IsLineElem() {
			break
		}
		ind++
		if ind >= l-1 {
			stw.EscapeAll()
			return
		}
	}
	for _, el1 := range elems[ind:] {
		if !el1.IsLineElem() {
			continue
		}
		for _, el2 := range elems[ind+1:] {
			if !el2.IsLineElem() {
				continue
			}
			stw.frame.IntersectionPoint(el1, el2, true, EPS)
		}
		_, _, err := el1.DivideAtOns(EPS)
		if err != nil {
			continue
		}
	}
	st.Snapshot(stw)
	stw.EscapeAll()
}

// }}}

// Bounded Area// {{{
func (stw *Window) BoundedArea(arg *iup.MouseButton, f func(ns []*st.Node, els []*st.Elem)) {
	stw.dbuff.UpdateYAxis(&arg.Y)
	if arg.Pressed == 0 { // Released
		var cand *st.Elem
		xmin := 100000.0
		for _, el := range stw.frame.Elems {
			if el.IsHidden(stw.frame.Show) || !el.IsLineElem() {
				continue
			}
			if el.Enod[0].Pcoord[1] == el.Enod[1].Pcoord[1] {
				continue
			}
			if (el.Enod[0].Pcoord[1]-float64(arg.Y))*(el.Enod[1].Pcoord[1]-float64(arg.Y)) < 0 {
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

func (stw *Window) Chain(x, y float64, el *st.Elem, maxdepth int) ([]*st.Node, []*st.Elem, error) {
	rtnns := make([]*st.Node, 2)
	rtnels := make([]*st.Elem, 1)
	rtnns[0] = el.Enod[0]
	rtnns[1] = el.Enod[1]
	rtnels[0] = el
	origin := []float64{x, y}
	_, cw := st.ClockWise(origin, el.Enod[0].Pcoord, el.Enod[1].Pcoord)
	tmpel := el
	next := el.Enod[1]
	depth := 1
	for {
		minangle := 10000.0
		var addelem *st.Elem
		for _, cand := range stw.frame.SearchElem(next) {
			if cand == nil {
				continue
			}
			if cand.IsHidden(stw.frame.Show) || !cand.IsLineElem() || cand == tmpel {
				continue
			}
			var otherside *st.Node
			for _, en := range cand.Enod {
				if en != next {
					otherside = en
					break
				}
			}
			if otherside == nil { // case cand has same node
				continue
			}
			angle, tmpcw := st.ClockWise(next.Pcoord, origin, otherside.Pcoord)
			angle = math.Abs(angle)
			if cw != tmpcw && angle < minangle {
				addelem = cand
				minangle = angle
			}
		}
		if addelem == nil {
			return nil, nil, errors.New("Chain: Not Bounded")
		}
		for _, en := range addelem.Enod {
			if en == el.Enod[0] {
				rtnels = append(rtnels, addelem)
				return rtnns[:depth+1], rtnels[:depth+1], nil
			}
			if en != next {
				next = en
				break
			}
		}
		rtnns = append(rtnns, next)
		rtnels = append(rtnels, addelem)
		tmpel = addelem
		depth++
		if depth > maxdepth {
			return rtnns[:depth], rtnels[:depth], errors.New("Chain: Too Much Recursion")
		}
	}
}

// }}}

// divideenods// {{{
func divideenods(ns []*st.Node, maxlen int) [][]*st.Node {
	if len(ns) <= maxlen {
		return [][]*st.Node{ns}
	}
	rtn := make([][]*st.Node, 0)
	num := 0
	seek := 0
	tmp := make([]*st.Node, maxlen)
	for {
		var ind int
		for i, n := range ns[seek:] {
			ind = i % maxlen
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
func hatchplateelem(stw *Window) {
	stw.canv.SetAttribute("CURSOR", "PEN")
	createhatch := func(ns []*st.Node, els []*st.Elem) {
		en := st.ModifyEnod(ns)
		en = st.Upside(en)
		sec := stw.frame.DefaultSect()
		switch len(en) {
		case 0, 1, 2:
			return
		case 3, 4:
			if len(stw.frame.SearchElem(en...)) == 0 {
				el := stw.frame.AddPlateElem(-1, en, sec, st.NULL)
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
				if len(stw.frame.SearchElem(eni...)) == 0 {
					el := stw.frame.AddPlateElem(-1, eni, sec, st.NULL)
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
		st.Snapshot(stw)
	}
	stw.canv.SetCallback(func(arg *iup.MouseButton) {
		if stw.frame != nil {
			switch arg.Button {
			case BUTTON_LEFT:
				stw.BoundedArea(arg, createhatch)
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
					stw.EscapeAll()
				}
			}
		}
	})
	stw.canv.SetCallback(func(arg *iup.MouseMotion) {
		if stw.frame != nil {
			stw.dbuff.UpdateYAxis(&arg.Y)
			switch statusKey(arg.Status) {
			case STATUS_CENTER:
				stw.MoveOrRotate(arg)
			}
		}
	})
}

// }}}

// ADDPLATEALL // {{{
func addplateall(stw *Window) {
	sels := stw.SelectedElems()
	if sels == nil || len(sels) < 1 {
		stw.EscapeAll()
		return
	}
	elems := make([]*st.Elem, len(sels))
	enum := 0
	for _, el := range sels {
		if el != nil && el.IsLineElem() {
			elems[enum] = el
			enum++
		}
	}
	elems = elems[:enum]
	sort.Sort(st.ElemByNum{elems})
	search := func(ns ...*st.Node) []*st.Elem {
		els := make([]*st.Elem, 0)
		num := 0
		l := len(ns)
		for _, el := range elems {
			count := 0
			found := make([]bool, len(el.Enod))
		loopse:
			for _, n := range ns {
				for i, en := range el.Enod {
					if found[i] {
						continue
					}
					if en == n {
						found[i] = true
						count++
						continue loopse
					}
				}
			}
			if count == l {
				els = append(els, el)
				num++
			}
		}
		return els[:num]
	}
	maxsize := 4
	sec := stw.frame.DefaultSect()
	etype := st.NULL
	add := 0
	added := make([]*st.Elem, 0)
	var found bool
	var n1, n2 *st.Node
	var en []*st.Node
	start := func() {
		for _, el := range elems {
			found = false
			for _, el1 := range search(el.Enod[0]) {
				if el1 == el || !el1.IsLineElem() {
					continue
				}
				n1 = el1.Otherside(el.Enod[0])
				if n1 == nil {
					continue
				}
				for _, sel := range search(n1, el.Enod[1]) {
					if sel.IsLineElem() {
						found = true
						break
					}
				}
				if found {
					en = []*st.Node{el.Enod[0], el.Enod[1], n1}
					if len(stw.frame.SearchElem(en...)) == 0 {
						el := stw.frame.AddPlateElem(-1, en, sec, etype)
						added = append(added, el)
						add++
					}
					found = false
					continue
				}
				if maxsize >= 4 {
					for _, el2 := range search(el.Enod[1]) {
						if el2 == el || !el2.IsLineElem() {
							continue
						}
						n2 = el2.Otherside(el.Enod[1])
						if n2 == nil {
							continue
						}
						for _, sel := range search(n1, n2) {
							if sel.IsLineElem() {
								found = true
								break
							}
						}
						if found {
							en = []*st.Node{el.Enod[0], el.Enod[1], n2, n1}
							if len(stw.frame.SearchElem(en...)) == 0 {
								el := stw.frame.AddPlateElem(-1, en, sec, etype)
								added = append(added, el)
								add++
							}
							found = false
						}
					}
				}
			}
		}
		stw.SelectElem(added[:add])
		var buf bytes.Buffer
		buf.WriteString(fmt.Sprintf("%d ELEM (SECT %d, ETYPE %s) added", add, sec.Num, st.ETYPES[etype]))
		stw.addHistory(buf.String())
		stw.EscapeCB()
	}
	stw.canv.SetCallback(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		switch key.Key() {
		default:
			stw.DefaultKeyAny(arg)
		case '3':
			maxsize = 3
		case '4':
			maxsize = 4
		case KEY_ENTER:
			start()
		}
	})
}

// }}}

// EDITLINEELEM// {{{
func editlineelem(stw *Window) {
	if !stw.ElemSelected() {
		stw.EscapeAll()
		return
	}
	replaceenod := func(n *st.Node) {
		for _, el := range stw.SelectedElems() {
			for i, en := range el.Enod {
				if en == stw.SelectedNodes()[0] {
					el.Enod[i] = n
					break
				}
			}
		}
		stw.SelectNode(make([]*st.Node, 2))
		st.Snapshot(stw)
		stw.Redraw()
	}
	get2nodes(stw, replaceenod, func() {
		stw.SelectNode(make([]*st.Node, 2))
		stw.Redraw()
	})
}

// }}}

// EDITPLATEELEM// {{{
func editplateelem(stw *Window) {
	if !stw.ElemSelected() {
		stw.EscapeAll()
		return
	}
	replaceenod := func(n *st.Node) {
		for _, el := range stw.SelectedElems() {
			for i, en := range el.Enod {
				if en == stw.SelectedNodes()[0] {
					el.Enod[i] = n
					break
				}
			}
		}
		stw.SelectNode(make([]*st.Node, 2))
		st.Snapshot(stw)
		stw.Redraw()
	}
	prune := func() {
		for _, el := range stw.SelectedElems() {
			var ens int
			tmp := make([]*st.Node, el.Enods)
		prnloop:
			for _, en := range el.Enod {
				for _, n := range stw.SelectedNodes() {
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
		stw.SelectNode(make([]*st.Node, 2))
		stw.Redraw()
	}
	get2nodes(stw, replaceenod, prune)
}

// }}}

// EDITWRECT// {{{
func editwrect(stw *Window) {
	setwrect := func(els ...*st.Elem) {
		ans, err := stw.Query("開口長さ h[m]")
		if err != nil {
			st.ErrorMessage(stw, err, st.ERROR)
			stw.EscapeCB()
			return
		}
		w, err := strconv.ParseFloat(ans, 64)
		if err != nil {
			st.ErrorMessage(stw, err, st.ERROR)
			stw.EscapeCB()
			return
		}
		ans, err = stw.Query("開口高さ l[m]")
		if err != nil {
			st.ErrorMessage(stw, err, st.ERROR)
			stw.EscapeCB()
			return
		}
		h, err := strconv.ParseFloat(ans, 64)
		if err != nil {
			st.ErrorMessage(stw, err, st.ERROR)
			stw.EscapeCB()
			return
		}
		for _, el := range els {
			if !el.IsLineElem() {
				el.Wrect[0] = w
				el.Wrect[1] = h
			}
		}
	}
	selected := false
	if stw.ElemSelected() {
		els := make([]*st.Elem, 0)
		for _, el := range stw.SelectedElems() {
			if el == nil {
				continue
			}
			if el.IsLineElem() {
				continue
			}
			els = append(els, el)
			selected = true
		}
		if selected {
			setwrect(els...)
			stw.EscapeCB()
			return
		}
	}
	get1elem(stw, func(el *st.Elem, x, y int) {
		setwrect(el)
		stw.Redraw()
	},
		func(el *st.Elem) bool {
			return !el.IsLineElem()
		},
		func() {})
}

// }}}

// CONVEXHULL
func convexhull(stw *Window) {
	stw.DrawConvexHull()
	// stw.EscapeAll()
}

// REACTION// {{{
func reaction(stw *Window) {
	tmp, err := stw.Query("方向を指定[0～5]")
	if err != nil {
		st.ErrorMessage(stw, err, st.ERROR)
		stw.EscapeCB()
		return
	}
	val, err := strconv.ParseInt(tmp, 10, 64)
	if err != nil {
		st.ErrorMessage(stw, err, st.ERROR)
		stw.EscapeCB()
		return
	}
	d := int(val)
	if !stw.NodeSelected() {
		selectconfed(stw)
	}
	sns := stw.SelectedNodes()
	sort.Sort(st.NodeByNum{sns})
	fn := st.Ce(stw.frame.Path, ".rct")
	err = st.WriteReaction(fn, sns, d, stw.frame.Show.Unit[0])
	if err != nil {
		st.ErrorMessage(stw, err, st.ERROR)
		stw.EscapeCB()
		return
	}
	stw.addHistory(fmt.Sprintf("OUTPUT: %s", fn))
	stw.EscapeCB()
}

// }}}

// SUMREACTION// {{{
func sumreaction(stw *Window) {
	if !stw.NodeSelected() {
		stw.EscapeAll()
	}
	showval := func() {
		val := make([]float64, 5)
		for _, n := range stw.SelectedNodes() {
			if n == nil {
				continue
			}
			for i := 0; i < 5; i++ {
				val[i] += n.Weight[1]
			}
			if rea, ok := n.Reaction["L"]; ok {
				for i := 0; i < 5; i++ {
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
		for i, name := range []string{"  ", "+X", "-X", "+Y", "-Y"} {
			if i != 0 {
				result.WriteString(" ")
			}
			result.WriteString(fmt.Sprintf("W+L%s: %.3f", name, val[i]))
		}
		stw.addHistory(result.String())
	}
	stw.canv.SetCallback(func(arg *iup.MouseButton) {
		if stw.frame != nil {
			stw.dbuff.UpdateYAxis(&arg.Y)
			switch arg.Button {
			case BUTTON_CENTER:
				if arg.Pressed == 0 { // Released
					// if stw.selectNode[0] != nil { // TODO
					//     stw.cdcanv.Line(int(stw.selectNode[0].Pcoord[0]), int(stw.selectNode[0].Pcoord[1]), stw.endX, stw.endY)
					// }
					stw.Redraw()
				} else { // Pressed
					if isDouble(arg.Status) {
						stw.frame.SetFocus(nil)
						stw.RedrawNode()
						stw.ShowCenter()
					} else {
						stw.startX = int(arg.X)
						stw.startY = int(arg.Y)
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
	stw.canv.SetCallback(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		switch key.Key() {
		default:
			stw.DefaultKeyAny(arg)
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
func uplift(stw *Window) {
	stw.Deselect()
	selectuplift := func(lis []string) {
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
		for _, n := range stw.frame.Nodes {
			if !n.Conf[2] {
				continue
			}
			for _, p := range periods {
				val := n.Weight[1] + n.ReturnReaction(p, 2)
				fmt.Printf("NODE %d VAL %.3f\n", n.Num, val)
				if val < 0.0 {
					st.AddSelection(stw, n)
					num++
				}
			}
		}
		if num == 0 {
			stw.addHistory("NO UPLIFTS")
		} else {
			stw.addHistory(fmt.Sprintf("%d UPLIFTS", num))
		}
		stw.EscapeCB()
	}
	stw.canv.SetCallback(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		switch key.Key() {
		default:
			stw.cline.SetAttribute("APPEND", string(key.Key()))
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
func notice1459(stw *Window) {
	getnnodes(stw, 3, func(num int) {
		var delta float64
		ds := make([]float64, num)
		sns := stw.SelectedNodes()
		for i := 0; i < num; i++ {
			ds[i] = -sns[i].ReturnDisp(stw.frame.Show.Period, 2) * 100
		}
		var length float64
		var otp bytes.Buffer
		sname := "G1"
		alpha := 1.0
		switch num {
		default:
			stw.EscapeAll()
			return
		case 2:
			delta = ds[1] - ds[0]
			stw.addHistory(fmt.Sprintf("Disp: %.3f - %.3f [cm]", ds[1], ds[0]))
			for i := 0; i < 2; i++ {
				length += math.Pow(sns[1].Coord[i]-sns[0].Coord[i], 2)
			}
			length = math.Sqrt(length) * 100
			var enum, snum int
			for _, el := range stw.frame.SearchElem(sns[0], sns[1]) {
				if el.Etype == st.GIRDER {
					enum = el.Num
					snum = el.Sect.Num
					sname = el.Sect.Name
					break
				}
			}
			if _, ok := stw.frame.Sects[snum]; ok {
				switch {
				case stw.frame.Sects[snum].IsRc(1e-3):
					alpha = 8.0
				}
			}
			otp.WriteString(fmt.Sprintf("梁%s(部材 %d 、断面 %d 、節点 %d)\nFL 、X通りY通り\n最大たわみ　δ= %.3f - %.3f = %.3f [cm]\n長さL= %.1f [cm]\n変形増大係数α= %.0f\nα×δ/L=%.0f×%.3f/ %.3f = 1/%d\n", sname, enum, snum, sns[1].Num, ds[1], ds[0], delta, length, alpha, alpha, delta, length, math.Abs(length/(alpha*delta))))
			clipboard.WriteAll(otp.String())
		case 3:
			delta = ds[1] - 0.5*(ds[0]+ds[2])
			stw.addHistory(fmt.Sprintf("Disp: %.3f - (%.3f + %.3f)/2 [cm]", ds[1], ds[0], ds[2]))
			for i := 0; i < 2; i++ {
				length += math.Pow(sns[2].Coord[i]-sns[0].Coord[i], 2)
			}
			length = math.Sqrt(length) * 100
			var enum1, enum2, snum int
			for _, el := range stw.frame.SearchElem(sns[0], sns[1]) {
				if el.Etype == st.GIRDER {
					enum1 = el.Num
					snum = el.Sect.Num
					sname = el.Sect.Name
					break
				}
			}
			for _, el := range stw.frame.SearchElem(sns[1], sns[2]) {
				if el.Etype == st.GIRDER {
					enum2 = el.Num
					snum = el.Sect.Num
					sname = el.Sect.Name
					break
				}
			}
			if _, ok := stw.frame.Sects[snum]; ok {
				switch {
				case stw.frame.Sects[snum].IsRc(1e-3):
					alpha = 8.0
				}
			}
			otp.WriteString(fmt.Sprintf("梁%s(部材 %d, %d 、断面 %d 、節点 %d)\nFL 、X通りY通り\n最大たわみ　δ= %.3f - (%.3f + %.3f)/2 = %.3f [cm]\n長さL= %.1f [cm]\n変形増大係数α= %.0f\nα×δ/L=%.0f×%.3f/ %.1f = 1/%d\n", sname, enum1, enum2, snum, sns[1].Num, ds[1], ds[0], ds[2], delta, length, alpha, alpha, delta, length, int(math.Abs(length/(alpha*delta)))))
			clipboard.WriteAll(otp.String())
		}
		if delta != 0.0 {
			stw.addHistory(fmt.Sprintf("Distance: %.3f[cm]", length))
			stw.addHistory(fmt.Sprintf("Alpha: %.0f", alpha))
			stw.addHistory(fmt.Sprintf("Slope: 1/%.1f", math.Abs(length/(alpha*delta))))
		}
		stw.EscapeAll()
	})
}

// }}}

// CATBYNODE// {{{
func catbynode(stw *Window) {
	get1node(stw, func(n *st.Node) {
		err := stw.frame.CatByNode(n, true)
		if err != nil {
			st.ErrorMessage(stw, err, st.ERROR)
		} else {
			st.Snapshot(stw)
			stw.Redraw()
		}
	})
}

// }}}

// CATINTERMEDIATENODE TODO: TEST // {{{
func catintermediatenode(stw *Window) {
	stw.Deselect()
	ns := make([]*st.Node, 0)
	nnum := 0
	for _, n := range stw.frame.Nodes {
		els := stw.frame.SearchElem(n)
		l := len(els)
		if l < 2 {
			continue
		}
		ind := 0
		lels := make([]*st.Elem, l)
		for _, el := range els {
			if el.IsLineElem() {
				lels[ind] = el
				ind++
			}
		}
		if ind < 2 {
			continue
		}
		d := lels[0].Direction(false)
		sec := lels[0].Sect
		ci := true
		for _, el := range lels[1:ind] {
			if el.Sect != sec {
				ci = false
				break
			}
			if !el.IsParallel(d, EPS) {
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
		stw.SelectNode(ns[:nnum])
		stw.Redraw()
		if stw.Yn("CAT INTERMEDIATE NODE", "中間節点を除去しますか？") {
			for _, n := range stw.SelectedNodes() {
				stw.frame.CatByNode(n, true)
			}
			st.Snapshot(stw)
		}
	}
	stw.EscapeAll()
}

// }}}

// GETNELEMS// {{{
func getnelems(stw *Window, size int, f func(int)) {
	selected := 0
	for _, el := range stw.SelectedElems() {
		if el != nil {
			selected++
		}
	}
	if selected >= size {
		f(selected)
		return
	}
	setenum := func() {
		enum, err := strconv.ParseInt(stw.cline.GetAttribute("VALUE"), 10, 64)
		if err == nil {
			if el, ok := stw.frame.Elems[int(enum)]; ok {
				st.AddSelection(stw, el)
			}
		}
		stw.cline.SetAttribute("VALUE", "")
	}
	stw.canv.SetCallback(func(arg *iup.MouseButton) {
		if stw.frame != nil {
			switch arg.Button {
			case BUTTON_LEFT:
				stw.SelectElemStart(arg)
			case BUTTON_CENTER:
				stw.dbuff.UpdateYAxis(&arg.Y)
				if arg.Pressed == 0 { // Released
					// if stw.selectNode[0] != nil { // TODO
					//     stw.cdcanv.Line(int(stw.selectNode[0].Pcoord[0]), int(stw.selectNode[0].Pcoord[1]), stw.endX, stw.endY)
					// }
					stw.Redraw()
				} else { // Pressed
					if isDouble(arg.Status) {
						stw.frame.SetFocus(nil)
						stw.RedrawNode()
						stw.ShowCenter()
					} else {
						stw.startX = int(arg.X)
						stw.startY = int(arg.Y)
					}
				}
			case BUTTON_RIGHT:
				if arg.Pressed == 0 {
					if v := stw.cline.GetAttribute("VALUE"); v != "" {
						setenum()
					} else {
						// fmt.Println("CONTEXT MENU")
						f(len(stw.SelectedElems()))
					}
				}
			}
		}
	})
	stw.canv.SetCallback(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		switch key.Key() {
		default:
			stw.DefaultKeyAny(arg)
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
func joinlineelem(stw *Window) {
	getnelems(stw, 2, func(size int) {
		els := make([]*st.Elem, 2)
		num := 0
		for _, el := range stw.SelectedElems() {
			if el != nil && el.IsLineElem() {
				els[num] = el
				num++
				if num >= 2 {
					break
				}
			}
		}
		if num == 2 {
			err := stw.frame.JoinLineElem(els[0], els[1], true, true)
			if err != nil {
				switch err.(type) {
				default:
					st.ErrorMessage(stw, err, st.ERROR)
					return
				case st.ParallelError:
					if stw.Yn("JOIN LINE ELEM", "平行でない部材を結合しますか") {
						err := stw.frame.JoinLineElem(els[0], els[1], false, true)
						if err != nil {
							st.ErrorMessage(stw, err, st.ERROR)
							return
						}
					}
				}
			}
			st.Snapshot(stw)
			stw.EscapeAll()
		}
	})
}

// }}}

// JOINPLATEELEM// {{{
func joinplateelem(stw *Window) {
	getnelems(stw, 2, func(size int) {
		els := make([]*st.Elem, 2)
		num := 0
		for _, el := range stw.SelectedElems() {
			if el != nil && !el.IsLineElem() {
				els[num] = el
				num++
				if num >= 2 {
					break
				}
			}
		}
		if num == 2 {
			err := stw.frame.JoinPlateElem(els[0], els[1])
			if err != nil {
				st.ErrorMessage(stw, err, st.ERROR)
			} else {
				st.Snapshot(stw)
				stw.EscapeAll()
			}
		}
	})
}

// }}}

// MERGENODE// {{{
func mergenode(stw *Window) {
	merge := func(ns []*st.Node) {
		c := make([]float64, 3)
		num := 0
		for _, n := range ns {
			if n == nil {
				continue
			}
			for i := 0; i < 3; i++ {
				c[i] += n.Coord[i]
			}
			num++
		}
		if num > 0 {
			for i := 0; i < 3; i++ {
				c[i] /= float64(num)
			}
			var del bool
			delmap := make(map[*st.Node]*st.Node)
			var n0 *st.Node
			for _, n := range ns {
				if n == nil {
					continue
				}
				if del {
					delmap[n] = n0
				} else {
					n.Coord = c
					n0 = n
					del = true
				}
			}
			stw.frame.ReplaceNode(delmap)
			st.Snapshot(stw)
		}
	}
	if stw.NodeSelected() {
		merge(stw.SelectedNodes())
		stw.EscapeAll()
		return
	}
	getnnodes(stw, len(stw.frame.Nodes), func(num int) { merge(stw.SelectedNodes()); stw.EscapeAll() })
}

// }}}

// ERASE// {{{
func erase(stw *Window) {
	stw.DeleteSelected()
	stw.Deselect()
	ns := stw.frame.NodeNoReference()
	if len(ns) != 0 {
		for _, n := range ns {
			stw.frame.DeleteNode(n.Num)
		}
	}
	stw.EscapeAll()
}

// }}}

// FACTS// {{{
func facts(stw *Window) {
	fn := st.Ce(stw.frame.Path, ".fes")
	err := stw.frame.Facts(fn, []int{st.COLUMN, st.GIRDER, st.BRACE, st.WBRACE, st.SBRACE}, nil, nil, []string{"X", "Y"}, false)
	if err != nil {
		st.ErrorMessage(stw, err, st.ERROR)
	} else {
		stw.addHistory(fmt.Sprintf("Output: %s", fn))
	}
	stw.EscapeAll()
}

// }}}

// ZOUBUNDISP// {{{
func zoubundisp(stw *Window) {
	if !stw.NodeSelected() {
		stw.EscapeAll()
		return
	}
	pers, err := stw.QueryList("PERIODを指定")
	if err != nil {
		st.ErrorMessage(stw, err, st.ERROR)
		stw.EscapeCB()
	}
	tmp, err := stw.Query("方向を指定[0～5]")
	if err != nil {
		st.ErrorMessage(stw, err, st.ERROR)
		stw.EscapeCB()
	}
	val, err := strconv.ParseInt(tmp, 10, 64)
	if err != nil {
		st.ErrorMessage(stw, err, st.ERROR)
		stw.EscapeCB()
	}
	d := int(val)
	if d < 0 || d > 5 {
		st.ErrorMessage(stw, errors.New(":zoubundisp direction should be between 0 ~ 6"), st.ERROR)
		stw.EscapeCB()
		return
	}
	fn := filepath.Join(filepath.Dir(stw.frame.Path), "zoubunout.txt")
	err = stw.frame.ReportZoubunDisp(fn, stw.SelectedNodes(), pers, d)
	if err != nil {
		st.ErrorMessage(stw, err, st.ERROR)
		stw.EscapeCB()
		return
	}
	stw.addHistory(fmt.Sprintf("OUTPUT: %s", fn))
	stw.EscapeCB()
}

// }}}

// ZOUBUNYIELD// {{{
func zoubunyield(stw *Window) {
	var otp bytes.Buffer
	var skeys []int
	for k := range stw.frame.Sects {
		skeys = append(skeys, k)
	}
	sort.Ints(skeys)
	otp.WriteString("種別　断面番号　  　　　 Ｎ[tf]　Ｑx[tf]　 Ｑy[tf]　Ｍz[tfm]　Ｍx[tfm]　Ｍy[tfm]\n")
	for _, k := range skeys {
		sec := stw.frame.Sects[k]
		if sec.Num < 700 {
			otp.WriteString(fmt.Sprintf("         %4d     最大   ", sec.Num))
			for i := 0; i < 6; i++ {
				otp.WriteString(fmt.Sprintf(" %8.1f", sec.Yield[2*i]))
			}
			otp.WriteString("\n")
			otp.WriteString("                  最小   ")
			for i := 0; i < 6; i++ {
				otp.WriteString(fmt.Sprintf(" %8.1f", sec.Yield[2*i+1]))
			}
			otp.WriteString("\n")
		} else if sec.Num > 900 {
			otp.WriteString(fmt.Sprintf("         %4d     最大   ", sec.Num))
			otp.WriteString(fmt.Sprintf(" %8.1f", sec.Yield[0]))
			otp.WriteString("\n")
			otp.WriteString("                  最小   ")
			otp.WriteString(fmt.Sprintf(" %8.1f", sec.Yield[1]))
			otp.WriteString("\n")
		}
	}
	fn := filepath.Join(filepath.Dir(stw.frame.Path), "zoubunyield.txt")
	w, err := os.Create(fn)
	defer w.Close()
	if err != nil {
		st.ErrorMessage(stw, err, st.ERROR)
		stw.EscapeCB()
	}
	otp.WriteTo(w)
	stw.addHistory(fmt.Sprintf("OUTPUT: %s", fn))
	stw.EscapeAll()
}

// }}}

// ZOUBUNREACTION// {{{
func zoubunreaction(stw *Window) {
	if !stw.NodeSelected() {
		stw.EscapeAll()
		return
	}
	pers, err := stw.QueryList("PERIODを指定")
	if err != nil {
		st.ErrorMessage(stw, err, st.ERROR)
		stw.EscapeCB()
	}
	tmp, err := stw.Query("方向を指定[0～5]")
	if err != nil {
		st.ErrorMessage(stw, err, st.ERROR)
		stw.EscapeCB()
	}
	val, err := strconv.ParseInt(tmp, 10, 64)
	if err != nil {
		st.ErrorMessage(stw, err, st.ERROR)
		stw.EscapeCB()
	}
	d := int(val)
	if d < 0 || d > 5 {
		st.ErrorMessage(stw, errors.New(":zoubundisp direction should be between 0 ~ 6"), st.ERROR)
		stw.EscapeCB()
		return
	}
	fn := filepath.Join(filepath.Dir(stw.frame.Path), "zoubunout.txt")
	err = stw.frame.ReportZoubunReaction(fn, stw.SelectedNodes(), pers, d)
	if err != nil {
		st.ErrorMessage(stw, err, st.ERROR)
		stw.EscapeCB()
		return
	}
	stw.addHistory(fmt.Sprintf("OUTPUT: %s", fn))
	stw.EscapeCB()
}

// }}}

// AMOUNTPROP// {{{
func amountprop(stw *Window) {
	tmp, err := stw.QueryList("PROP")
	if err != nil {
		stw.EscapeAll()
		return
	}
	l := len(tmp)
	props := make([]int, l)
	for i := 0; i < l; i++ {
		val, err := strconv.ParseInt(tmp[i], 10, 64)
		if err != nil {
			stw.EscapeAll()
			return
		}
		props[i] = int(val)
	}
	fn := filepath.Join(filepath.Dir(stw.frame.Path), "amount.txt")
	err = stw.frame.AmountProp(fn, props...)
	if err != nil {
		st.ErrorMessage(stw, err, st.ERROR)
		stw.EscapeCB()
		return
	}
	stw.addHistory(fmt.Sprintf("OUTPUT: %s", fn))
	stw.EscapeAll()
}

// }}}

func seteps(stw *Window) {
	ans, err := stw.Query("許容差[m]")
	if err != nil {
		st.ErrorMessage(stw, err, st.ERROR)
		stw.addHistory(fmt.Sprintf("EPS=%.3E", EPS))
		stw.EscapeCB()
		return
	}
	val, err := strconv.ParseFloat(ans, 64)
	if err != nil {
		st.ErrorMessage(stw, err, st.ERROR)
		stw.addHistory(fmt.Sprintf("EPS=%.3E", EPS))
		stw.EscapeCB()
		return
	}
	EPS = val
	stw.addHistory(fmt.Sprintf("EPS=%.3E", EPS))
}
