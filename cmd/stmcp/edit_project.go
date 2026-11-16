package main

import (
	"fmt"
	"math"
	"slices"
	"strconv"

	st "github.com/yofu/st/stlib"
)

var projectEditors = []string{
	"GRAVITY", "SMOOTH", "HANGING", "RING", "TENSEGRITYADD", "TENSEGRITYDELETE", "TENSEGRITYCONNECTED",
	"VEN14ROTATEZERO", "VEN14CUTTER", "VEN14SETCANG",
}

func (w *editWindow) runProjectEdit(name string, args []string) error {
	f, nodes, elems := w.frame, w.SelectedNodes(), w.SelectedElems()
	value := func(i int, defaultValue float64) (float64, error) {
		if i >= len(args) {
			return defaultValue, nil
		}
		return finiteArg(args[i])
	}

	zEditable := func(n *st.Node) bool { return !n.Conf[2] && !n.IsNotEditable(f.Show) }
	switch name {
	case "GRAVITY":
		height, err := value(0, 0.1)
		if err != nil {
			return err
		}
		r, err := value(1, 0.5)
		if err != nil {
			return err
		}
		if r <= 0 {
			return fmt.Errorf("GRAVITY radius must be positive")
		}
		for _, center := range nodes {
			for _, n := range f.Nodes {
				if !zEditable(n) {
					continue
				}
				d := math.Hypot(n.Coord[0]-center.Coord[0], n.Coord[1]-center.Coord[1]) / r
				if d <= 1 {
					n.Coord[2] += height * (1 - 0.5*d*d)
				} else if d <= 2 {
					n.Coord[2] += height * (1/d - 0.5)
				}
			}
		}
	case "SMOOTH":
		ratio, err := value(0, 0.5)
		if err != nil {
			return err
		}
		for _, n := range nodes {
			if !zEditable(n) {
				continue
			}
			for _, neighbour := range f.LineConnected(n) {
				if !neighbour.Conf[2] {
					neighbour.Coord[2] += ratio * (n.Coord[2] - neighbour.Coord[2])
				}
			}
		}
	case "HANGING":
		height, err := value(0, 2.9)
		if err != nil {
			return err
		}
		r, err := value(1, 0.4)
		if err != nil {
			return err
		}
		if r <= 0 {
			return fmt.Errorf("HANGING radius must be positive")
		}
		for _, n := range nodes {
			if !zEditable(n) {
				continue
			}
			n.Coord[2] = height
			for _, support := range f.Nodes {
				if !support.Conf[2] {
					continue
				}
				d := math.Sqrt(math.Hypot(n.Coord[0]-support.Coord[0], n.Coord[1]-support.Coord[1])) / r
				if d <= 1 {
					n.Coord[2] = min(n.Coord[2], 0.5*height*d*d)
				} else if d <= 2 {
					n.Coord[2] = min(n.Coord[2], 1.5*height-height/d)
				}
			}
		}
	case "RING":
		if len(nodes) < 3 {
			return fmt.Errorf("select at least three nodes")
		}
		var x, y float64
		for _, n := range nodes {
			x += n.Coord[0]
			y += n.Coord[1]
		}
		x /= float64(len(nodes))
		y /= float64(len(nodes))
		angle := func(n *st.Node) float64 { return math.Atan2(n.Coord[1]-y, n.Coord[0]-x) }
		slices.SortStableFunc(nodes, func(a, b *st.Node) int {
			if angle(a) < angle(b) {
				return -1
			}
			if angle(a) > angle(b) {
				return 1
			}
			return 0
		})
		for i, n := range nodes {
			f.AddLineElem(-1, []*st.Node{n, nodes[(i+1)%len(nodes)]}, f.DefaultSect(), st.COLUMN)
		}
	case "VEN14ROTATEZERO":
		if len(nodes) != 3 {
			return fmt.Errorf("select three reference nodes")
		}
		a, b, c := nodes[0], nodes[1], nodes[2]
		f.Move(-a.Coord[0], -a.Coord[1], -a.Coord[2])
		if st.OnTheSameLine(a.Coord, b.Coord, c.Coord, w.EPS()) {
			return fmt.Errorf("select noncollinear nodes")
		}
		if d := math.Hypot(b.Coord[0], b.Coord[1]); d > 0 {
			f.Rotate(a.Coord, []float64{b.Coord[1], -b.Coord[0], 0}, -math.Atan2(b.Coord[2], d))
		} else {
			f.Rotate(a.Coord, st.XAXIS, -math.Copysign(math.Pi/2, b.Coord[2]))
		}
		f.Rotate(a.Coord, st.ZAXIS, math.Atan2(b.Coord[0], b.Coord[1]))
		f.Rotate(a.Coord, st.YAXIS, math.Atan2(c.Coord[2], c.Coord[0]))
	case "VEN14CUTTER":
		return w.veneziaCut()
	case "VEN14SETCANG":
		for _, el := range f.Elems {
			if el.Etype != st.GIRDER {
				continue
			}
			vector := make([]float64, 3)
			for _, wall := range f.SearchElem(el.Enod...) {
				if wall.Etype == st.WALL && (wall.Sect.Num-el.Sect.Num)%100 == 0 {
					for i, v := range wall.Normal(true) {
						vector[i] += v
					}
				}
			}
			if vector[0] != 0 || vector[1] != 0 || vector[2] != 0 {
				if _, err := el.AxisToCang(vector, false); err != nil {
					return err
				}
			}
		}
	default:
		return w.tensegrityEdit(name, args, nodes, elems)
	}
	return nil
}

func (w *editWindow) veneziaCut() error {
	f := w.frame
	sec := f.Sects[502]
	if sec == nil {
		return fmt.Errorf("VEN14CUTTER requires section 502")
	}
	lo, hi, _, _, _, _ := f.Bbox(true)
	if math.Max(math.Abs(lo), math.Abs(hi))/0.15 > 10000 {
		return fmt.Errorf("VEN14CUTTER exceeds 10000 slices")
	}
	for i := min(0, int(math.Ceil(lo/0.15))); i <= max(0, int(math.Floor(hi/0.15))); i++ {
		x := float64(i) * 0.15
		ns := make([]*st.Node, 0)
		for _, el := range f.Fence(0, x, false) {
			if !el.IsLineElem() {
				continue
			}
			added, _, err := el.DivideAtAxis(0, x, w.EPS())
			if err == nil {
				ns = append(ns, added...)
			}
		}
		for _, n := range f.Nodes {
			if math.Abs(n.Coord[0]-x) < 1e-8 && !slices.Contains(ns, n) {
				ns = append(ns, n)
			}
		}
		slices.SortFunc(ns, func(a, b *st.Node) int {
			if a.Coord[1] < b.Coord[1] {
				return -1
			}
			if a.Coord[1] > b.Coord[1] {
				return 1
			}
			return a.Num - b.Num
		})
		for j := 1; j < len(ns); j++ {
			if ns[j-1] != ns[j] {
				f.AddLineElem(-1, []*st.Node{ns[j-1], ns[j]}, sec, st.GIRDER)
			}
		}
	}
	return nil
}

func (w *editWindow) tensegrityEdit(name string, args []string, ns []*st.Node, els []*st.Elem) error {
	cs, ts, size := 201, 211, 2
	if name == "TENSEGRITYDELETE" {
		size = 6
	}
	if len(args) > 3 {
		return fmt.Errorf("use %s [compression_section tension_section size]", name)
	}
	for i, dst := range []*int{&cs, &ts, &size} {
		if i >= len(args) {
			break
		}
		n, err := strconv.Atoi(args[i])
		if err != nil || n < 0 {
			return fmt.Errorf("expected a nonnegative integer")
		}
		*dst = n
	}
	f := w.frame
	switch name {
	case "TENSEGRITYADD":
		candidates, err := st.TensegrityAddCandidates(f, els, cs, ts, size)
		if err != nil {
			return err
		}
		for _, el := range candidates {
			w.History(fmt.Sprintf("Tension: %d - %d", el.Enod[0].Num, el.Enod[1].Num))
			if w.confirm {
				f.AddElem(-1, el)
			}
		}
		w.Deselect()
	case "TENSEGRITYDELETE":
		candidates, err := st.TensegrityDeleteCandidates(f, ns, ts, size)
		if err != nil {
			return err
		}
		w.DeselectNode()
		w.SelectElem(candidates)
		if w.confirm {
			st.DeleteSelected(w)
		}
	case "TENSEGRITYCONNECTED":
		if len(els) == 0 || !els[0].IsLineElem() {
			return fmt.Errorf("select a compression or tension member")
		}
		w.SelectElem(st.TensegrityConnected(f, els[0], cs, ts))
	}
	return nil
}
