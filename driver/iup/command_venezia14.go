package stgui

import (
	"bytes"
	"fmt"
	"github.com/visualfc/go-iup/iup"
	"github.com/yofu/st/stlib"
	"math"
	"os"
	"sort"
)

var (
	VEN14ROTATEZERO = &Command{"#ROT", "VENEZIA14 ROTATE ZERO", "rotate model to zero point", ven14rotatezero}
	VEN14CUTTER     = &Command{"#CUT", "VENEZIA14 CUTTER", "cut model at regular intervals", ven14cutter}
	VEN14SURFACE    = &Command{"#SUR", "VENEZIA14 SURFACE", "output coord data to create surface", ven14surface}
	VEN14NORMAL     = &Command{"#NML", "VENEZIA14 NORMAL", "show normal vector of plate elems", ven14normal}
	VEN14SETCANG    = &Command{"#CAN", "VENEZIA14 SETCANG", "set cang of girders", ven14setcang}
	VEN14ERRORELEM  = &Command{"#ERR", "ERROR ELEM", "select elem whose max(rate)>1.15", ven14errorelem}
	VEN14DEPTH      = &Command{"#DEP", "VENEZIA14 DEPTH", "measure depth of surface", ven14depth}
)

func init() {
	Commands["VEN14ROTATEZERO"] = VEN14ROTATEZERO
	Commands["VEN14CUTTER"] = VEN14CUTTER
	Commands["VEN14SURFACE"] = VEN14SURFACE
	Commands["VEN14NORMAL"] = VEN14NORMAL
	Commands["VEN14SETCANG"] = VEN14SETCANG
	Commands["VEN14ERRORELEM"] = VEN14ERRORELEM
	Commands["VEN14DEPTH"] = VEN14DEPTH
}

func ven14rotatezero(stw *Window) {
	maxnum := 3
	getnnodes(stw, maxnum, func(num int) {
		if num >= 3 {
			ns := stw.selectNode[:3]
			stw.Frame.Move(-ns[0].Coord[0], -ns[0].Coord[1], -ns[0].Coord[2])
			l := 0.0
			for i := 0; i < 2; i++ {
				l += ns[1].Coord[i] * ns[1].Coord[i]
			}
			angle1 := -math.Atan2(ns[1].Coord[2], math.Sqrt(l))
			stw.Frame.Rotate(ns[0].Coord, []float64{ns[1].Coord[1], -ns[1].Coord[0], 0.0}, angle1)
			angle2 := math.Atan2(ns[1].Coord[0], ns[1].Coord[1])
			stw.Frame.Rotate(ns[0].Coord, []float64{0.0, 0.0, 1.0}, angle2)
			angle3 := math.Atan2(ns[2].Coord[2], ns[2].Coord[0])
			stw.Frame.Rotate(ns[0].Coord, []float64{0.0, 1.0, 0.0}, angle3)
		}
		stw.EscapeAll()
	})
}

func ven14cutter(stw *Window) {
	axis := 0
	compare := 1 - axis
	pitch := 0.15
	xmin, xmax, ymin, ymax, zmin, zmax := stw.Frame.Bbox(true)
	max := []float64{xmax, ymax, zmax}[axis]
	min := []float64{xmin, ymin, zmin}[axis]
	coord := 0.0
	for {
		els := stw.Frame.Fence(axis, coord, false)
		var keys []float64
		nodes := make(map[float64]*st.Node)
		for _, el := range els {
			n, _, _ := el.DivideAtAxis(axis, coord, EPS)
			nodes[n[0].Coord[compare]] = n[0]
			keys = append(keys, n[0].Coord[compare])
		}
		var num int
		for _, n := range stw.Frame.Nodes {
			if n.Coord[axis] == coord {
				nodes[n.Coord[compare]] = n
				keys = append(keys, n.Coord[compare])
				num++
			}
		}
		sort.Float64s(keys)
		sorted := make([]*st.Node, len(els)+num)
		for i, k := range keys {
			sorted[i] = nodes[k]
		}
		for i := 0; i < len(sorted)-1; i++ {
			sec := stw.Frame.Sects[502]
			stw.Frame.AddLineElem(-1, []*st.Node{sorted[i], sorted[i+1]}, sec, st.GIRDER)
		}
		coord += pitch
		if coord > max {
			break
		}
	}
	coord = 0.0
	for {
		coord -= pitch
		if coord < min {
			break
		}
		els := stw.Frame.Fence(axis, coord, false)
		var keys []float64
		nodes := make(map[float64]*st.Node)
		for _, el := range els {
			n, _, _ := el.DivideAtAxis(axis, coord, EPS)
			nodes[n[0].Coord[compare]] = n[0]
			keys = append(keys, n[0].Coord[compare])
		}
		var num int
		for _, n := range stw.Frame.Nodes {
			if n.Coord[axis] == coord {
				nodes[n.Coord[compare]] = n
				keys = append(keys, n.Coord[compare])
				num++
			}
		}
		sort.Float64s(keys)
		sorted := make([]*st.Node, len(els)+num)
		for i, k := range keys {
			sorted[i] = nodes[k]
		}
		for i := 0; i < len(sorted)-1; i++ {
			sec := stw.Frame.Sects[502]
			stw.Frame.AddLineElem(-1, []*st.Node{sorted[i], sorted[i+1]}, sec, st.GIRDER)
		}
	}
	stw.EscapeAll()
}

func ven14surface(stw *Window) {
	if stw.Frame != nil {
		if name, ok := iup.GetSaveFile("", ""); ok {
			var result bytes.Buffer
			var nkeys []int
			result.WriteString(fmt.Sprintf("_SrfPtGrid\n_KeepPoint\n11\n%d\n", len(stw.Frame.Nodes)/11))
			for k := range stw.Frame.Nodes {
				nkeys = append(nkeys, k)
			}
			sort.Ints(nkeys)
			for _, k := range nkeys {
				n := stw.Frame.Nodes[k]
				result.WriteString(fmt.Sprintf("%.3f,%.3f,%.3f\n", n.Coord[0]*1000, n.Coord[1]*1000, n.Coord[2]*1000))
			}
			w, err := os.Create(name)
			if err != nil {
				stw.errormessage(err, st.ERROR)
				return
			}
			result.WriteTo(w)
			w.Close()
		}
	}
}

func ven14normal(stw *Window) {
	// stw.Show.NodeNormal = true
	stw.Frame.Show.ElemNormal = true
	stw.Redraw()
	stw.canv.SetCallback(func(arg *iup.CommonKeyAny) {
		key := iup.KeyState(arg.Key)
		switch key.Key() {
		default:
			stw.DefaultKeyAny(arg)
		case KEY_ESCAPE:
			// stw.Show.NodeNormal = false
			stw.Frame.Show.ElemNormal = false
			stw.EscapeAll()
		}
	})
}

func ven14setcang(stw *Window) {
	for _, el := range stw.Frame.Elems {
		if el.Etype == st.GIRDER {
			walls := stw.Frame.SearchElem(el.Enod...)
			vec := make([]float64, 3)
			for _, w := range walls {
				if w.Etype == st.WALL && (w.Sect.Num-el.Sect.Num)%100 == 0 {
					tmp := w.Normal(true)
					for i := 0; i < 3; i++ {
						vec[i] += tmp[i]
					}
				}
			}
			el.AxisToCang(vec, false)
		}
	}
	stw.EscapeAll()
}

func ven14errorelem(stw *Window) {
	iup.SetFocus(stw.canv)
	stw.Deselect()
	stw.SetColorMode(st.ECOLOR_RATE)
	stw.Frame.Show.ElemCaption |= st.EC_RATE_L
	stw.Frame.Show.ElemCaption |= st.EC_RATE_S
	stw.Labels["EC_RATE_L"].SetAttribute("FGCOLOR", labelFGColor)
	stw.Labels["EC_RATE_S"].SetAttribute("FGCOLOR", labelFGColor)
	stw.Redraw()
	tmpels := make([]*st.Elem, len(stw.Frame.Elems))
	i := 0
	for _, el := range stw.Frame.Elems {
		if el.Rate != nil {
			for _, val := range el.Rate {
				if val > 1.15 {
					tmpels[i] = el
					i++
					break
				}
			}
		}
	}
	stw.selectElem = make([]*st.Elem, i)
	for j := 0; j < i; j++ {
		stw.selectElem[j] = tmpels[j]
	}
}

func ven14depth(stw *Window) {
	ns := make([]*st.Node, 0)
	num := 0
	if stw.selectNode != nil {
		for _, n := range stw.selectNode {
			if n != nil {
				ns = append(ns, n)
				num++
			}
		}
	}
	if num == 0 {
		stw.EscapeAll()
		return
	}
	ns = ns[:num]
	maxnum := 4
	getnnodes(stw, maxnum, func(num int) {
		switch num {
		default:
			stw.EscapeAll()
		case 4:
			v1 := make([]float64, 3)
			v2 := make([]float64, 3)
			for i := 0; i < 3; i++ {
				v1[i] = stw.selectNode[1].Coord[i] - stw.selectNode[0].Coord[i]
				v2[i] = stw.selectNode[3].Coord[i] - stw.selectNode[2].Coord[i]
			}
			vec := st.Cross(v1, v2)
			l := 0.0
			for i := 0; i < 3; i++ {
				l += vec[i] * vec[i]
			}
			l = math.Sqrt(l)
			for i := 0; i < 3; i++ {
				vec[i] /= l
			}
			dmax := -1e10
			dmin := 1e10
			for _, n := range ns {
				dtmp := 0.0
				for i := 0; i < 3; i++ {
					dtmp += vec[i] * n.Coord[i]
				}
				if dtmp > dmax {
					dmax = dtmp
				} else if dtmp < dmin {
					dmin = dtmp
				}
			}
			stw.addHistory(fmt.Sprintf("Depth = %.3f [mm]", (dmax-dmin)*1000))
			stw.EscapeAll()
		}
	})
}