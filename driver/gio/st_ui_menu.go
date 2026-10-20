package stgio

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	st "github.com/yofu/st/stlib"
)

type menuItem struct{ label, action string }
type menuDefinition struct {
	title string
	items []menuItem
}

var menus = []menuDefinition{
	{"File", []menuItem{{"New", "new"}, {"Open...  Ctrl+O", "Open"}, {"Open DXF...", "Open DXF"}, {"Insert...", "Insert"}, {"Reload", "reload"}, {"Save  Ctrl+S", "save"}, {"Save As...", "Save As"}, {"Read...", "Read"}, {"Read All", "readall"}, {"Quit", "quit"}}},
	{"Edit", []menuItem{{"Undo", "cmd::undo"}, {"Redo", "redo"}, {"Deselect", "deselect"}, {"Delete Selected", "delete"}, {"Add Line", "cmd:ADDLINEELEM"}, {"Add Plate", "cmd:ADDPLATEELEM"}, {"Move Nodes", "cmd:MOVENODE"}, {"Move Members", "cmd:MOVEELEM"}, {"Copy Members", "cmd:COPYELEM"}, {"Match Property", "cmd:MATCHPROPERTY"}, {"Erase by Click", "cmd:ERASE"}, {"Intersect Members", "cmd:INTERSECT"}, {"Fence Selection", "cmd:FENCE"}, {"Move Nodes to Line", "cmd:MOVETOLINE"}, {"Move Nodes to Plane", "cmd:MOVETOPLANE"}, {"Rotate Geometry", "cmd:ROTATE"}, {"Scale Geometry", "cmd:SCALE"}, {"Set Member Axis", "cmd:AXISTOCANG"}, {"Add Dimension", "cmd:MEASURE"}, {"Add Reference Line", "cmd:KIJUN"}, {"Nodes to Members", "cmd:NODETOELEM"}, {"Members to Nodes", "cmd:ELEMTONODE"}, {"Search Members by Node", "cmd:SEARCHELEM"}, {"Clear History", "clear-history"}}},
	{"View", []menuItem{{"Fit / 全体表示", "fit"}, {"Top", "cmd::view top"}, {"Front", "cmd::view front"}, {"Back", "cmd::view back"}, {"Right", "cmd::view right"}, {"Left", "cmd::view left"}}},
	{"Show", []menuItem{{"Show All", "showall"}, {"Hide Selected", "hide"}, {"Show Selected Only", "isolate"}, {"Deformation", "deformation"}, {"Next Color Mode", "color"}}},
	{"Dialog", []menuItem{{"View Settings", "tab:0"}, {"Show / Section Settings", "tab:1"}, {"Selected Properties", "tab:2"}}},
	{"Command", []menuItem{{"Distance by Click", "cmd:DISTS"}, {"Deformed Distance", "cmd:DEFORMEDDISTS"}, {"Check Frame", "cmd::check"}, {"Selected Length", "cmd::length"}, {"Selected Area", "cmd::area"}, {"Extract ARCLM", "cmd::extractarclm"}, {"Static Analysis...", "fill::analysis -period=L %<.otl"}, {"Nonlinear Static...", "fill::arclm101 -period=X -nlap=80 -delta=0.0125"}, {"Time History...", "fill::timehistory wave.csv -rayleigh=0.02 -draw=10"}, {"Stop Time History", "cmd::stop TH"}}},
	{"Tool", []menuItem{{"RC lst", "tool:rclst/rclst.html"}, {"Fig2 Keyword", "tool:fig2/fig2.html"}}},
	{"Help", []menuItem{{"Keyboard / Mouse", "keys"}, {"About", "about"}}},
}

func (w *Window) layoutMenuBar(gtx layout.Context) layout.Dimensions {
	children := []layout.FlexChild{}
	for i, menu := range menus {
		i, menu := i, menu
		children = append(children, layout.Rigid(func(c layout.Context) layout.Dimensions {
			return w.smallButton(c, "menu-"+menu.title, menu.title, func() {
				w.ui.contextItems = nil
				if w.ui.menu == i {
					w.ui.menu = -1
				} else {
					w.ui.menu = i
					w.ui.menuList.Position = layout.Position{}
				}
			})
		}))
	}
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx, children...)
}

func (w *Window) layoutMenuPopup(gtx layout.Context) {
	u := w.ui
	index := u.menu
	contextMenu := len(u.contextItems) > 0
	if !contextMenu && (index < 0 || index >= len(menus)) {
		return
	}
	var items []menuItem
	if contextMenu {
		items = u.contextItems
	} else {
		items = menus[index].items
	}
	shield := u.button("menu-shield")
	for shield.Clicked(gtx) {
		u.menu = -1
		u.contextItems = nil
	}
	shield.Layout(gtx, func(c layout.Context) layout.Dimensions { return layout.Dimensions{Size: c.Constraints.Max} })
	width := min(gtx.Dp(270), gtx.Constraints.Max.X)
	x := min(gtx.Dp(unit.Dp(index*58)), max(0, gtx.Constraints.Max.X-width))
	y := gtx.Dp(34)
	if contextMenu {
		x = max(0, min(u.contextPoint.X, max(0, gtx.Constraints.Max.X-width)))
		y = max(0, min(u.contextPoint.Y, max(0, gtx.Constraints.Max.Y-gtx.Dp(unit.Dp(len(items)*34+8)))))
	}
	height := min(gtx.Dp(unit.Dp(len(items)*34+8)), max(0, gtx.Constraints.Max.Y-y))
	u.menuRect = image.Rect(x, y, x+width, y+height)
	region(gtx, u.menuRect, func(c layout.Context) layout.Dimensions {
		paint.Fill(c.Ops, color.NRGBA{R: 250, G: 252, B: 255, A: 255})
		return widget.Border{Color: color.NRGBA{R: 130, G: 140, B: 155, A: 255}, Width: unit.Dp(1)}.Layout(c, func(c layout.Context) layout.Dimensions {
			return material.List(w.Theme, &u.menuList).Layout(c, len(items), func(c layout.Context, i int) layout.Dimensions {
				item := items[i]
				c.Constraints.Min.X = c.Constraints.Max.X
				return w.smallButton(c, fmt.Sprintf("menu-item-%d-%d", index, i), item.label, func() { u.menu = -1; u.contextItems = nil; w.menuAction(item.action) })
			})
		})
	})
}

func (w *Window) menuAction(action string) {
	u := w.ui
	if strings.HasPrefix(action, "recent:") {
		path := strings.TrimPrefix(action, "recent:")
		w.enqueueInterrupting(func() { w.applyChosenFile("Open", path, true) })
		return
	}
	switch action {
	case "Open", "Open DXF", "Save As", "Read", "Insert":
		w.fileDialog(action)
		return
	case "tab:0":
		u.tab = 0
		return
	case "tab:1":
		u.tab = 1
		return
	case "tab:2":
		u.tab = 2
		return
	case "clear-history":
		u.mu.Lock()
		u.log = ""
		u.mu.Unlock()
		return
	case "keys":
		w.History("Enter: 実行 / ↑↓: コマンド履歴 / Tab: 補完 / Ctrl+L: コマンド欄 / Ctrl+O: 開く / Ctrl+S: 保存\n中ドラッグ: 回転 / Shift+中: 移動 / 中ダブルクリック: 中央全体表示 / Ctrl+ホイール: 中心固定拡縮\n左: 部材 / Alt+左: 節点 / 通常: 選択追加 / Shift: 選択解除 / Ctrl+左: チェーン / 左ダブルクリック: 端部・板節点編集\n右短クリック: 入力実行・前回編集再実行 / Shift+右: 前回Ex / 右長押し: メニュー / Esc: 解除")
		return
	case "about":
		w.History("st — Gio v0.10.3\n構造モデル表示・解析コマンド / github.com/yofu/st")
		return
	case "save":
		if u.values["PATH"] == "" {
			w.fileDialog("Save As")
			return
		}
	}
	if strings.HasPrefix(action, "cmd:") {
		w.submitCommand(strings.TrimPrefix(action, "cmd:"))
		return
	}
	if strings.HasPrefix(action, "fill:") {
		u.command.SetText(strings.TrimPrefix(action, "fill:"))
		u.command.SetCaret(u.command.Len(), u.command.Len())
		u.focusCommand = true
		return
	}
	w.enqueueInterrupting(func() {
		var err error
		switch action {
		case "new":
			if w.changed && !w.Yn("New", "現在の変更を破棄しますか？") {
				return
			}
			w.SetFrame(st.NewFrame())
			w.Deselect()
			w.changed = false
			st.Snapshot(w)
		case "save":
			err = st.SaveFile(w, w.frame.Path)
		case "reload":
			if !w.changed || w.Yn("Reload", "現在の変更を破棄して再読込しますか？") {
				err = st.OpenFile(w, w.frame.Path, false)
			}
		case "readall":
			st.ReadAll(w)
		case "edit-inp":
			if w.frame.Path != "" {
				st.Edit(w.frame.Path)
			}
		case "quit":
			w.Close(false)
		case "redo":
			var f *st.Frame
			f, err = w.Redo()
			if err == nil && f != nil {
				w.SetFrame(f)
				w.changed = true
			}
		case "fit":
			w.showCanvasCenter()
		case "showall":
			st.ShowAll(w)
		case "hide":
			st.HideSelected(w)
		case "isolate":
			st.HideNotSelected(w)
		case "deselect":
			w.Deselect()
		case "delete":
			if w.Yn("Delete", "選択した節点・部材を削除しますか？") {
				st.DeleteSelected(w)
				st.Snapshot(w)
				w.changed = true
			}
		case "deformation":
			w.applyFlag("DEFORMATION", !w.flagValue("DEFORMATION"))
		case "color":
			w.frame.Show.ColorMode = (w.frame.Show.ColorMode + 1) % uint(len(st.ECOLORS))
		default:
			if strings.HasPrefix(action, "tool:") {
				st.StartTool(strings.TrimPrefix(action, "tool:"))
			}
		}
		if err != nil {
			w.History(err.Error())
		}
		w.Redraw()
	})
}

// Selection coordinates are canvas-local, including when the canvas is offset
// below the menu bar. The same projected model is used by the Property tab.
func (w *Window) finishSelection(modifiers key.Modifiers) {
	if w.frame == nil {
		return
	}
	x, y := float64(w.end.X), float64(w.end.Y)
	dx, dy := math.Abs(float64(w.end.X-w.start.X)), math.Abs(float64(w.end.Y-w.start.Y))
	inside := func(n *st.Node) bool {
		return n.Pcoord[0] >= float64(w.selected.Min.X) && n.Pcoord[0] <= float64(w.selected.Max.X) && n.Pcoord[1] >= float64(w.selected.Min.Y) && n.Pcoord[1] <= float64(w.selected.Max.Y)
	}
	remove := modifiers&key.ModShift != 0
	if w.selectsNodes(modifiers) {
		var nodes []*st.Node
		if dx < 15 && dy < 15 {
			if n := w.frame.PickNode(x, y, 15); n != nil {
				nodes = append(nodes, n)
			} else {
				w.SelectNode(nil)
				return
			}
		} else {
			for _, n := range w.frame.Nodes {
				if !n.IsHidden(w.frame.Show) && inside(n) {
					nodes = append(nodes, n)
				}
			}
		}
		st.MergeSelectNode(w, nodes, remove)
	} else if dx < 5 && dy < 5 {
		if e := w.frame.PickElem(x, y, 5); e != nil {
			els := []*st.Elem{e}
			if e.Chain != nil && modifiers&key.ModCtrl != 0 {
				els = e.Chain.Elems()
			}
			st.MergeSelectElem(w, els, remove)
		} else {
			w.SelectElem(nil)
		}
	} else {
		var elems []*st.Elem
		for _, e := range w.frame.Elems {
			if e.IsHidden(w.frame.Show) {
				continue
			}
			all, any := true, false
			for _, n := range e.Enod {
				in := inside(n)
				all = all && in
				any = any || in
			}
			if w.end.X >= w.start.X && all || w.end.X < w.start.X && any {
				elems = append(elems, e)
			}
		}
		st.MergeSelectElem(w, elems, remove)
	}
}

// Keep a visible border between the drawing area and settings.
func panelBorder(gtx layout.Context) {
	paint.FillShape(gtx.Ops, color.NRGBA{R: 190, G: 200, B: 213, A: 255}, clip.Rect(image.Rect(0, 0, 1, gtx.Constraints.Max.Y)).Op())
}
