package st

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
)

const (
	CONF_FREE  = 0
	CONF_ZROL  = 3
	CONF_XYROL = 4
	CONF_YROL  = 5
	CONF_XROL  = 6
	CONF_PIN   = 7
	CONF_FIX   = 63
)

var DispName = [6]string{"DX", "DY", "DZ", "TX", "TY", "TZ"}

type Node struct {
	Frame *Frame
	Num   int
	Coord []float64
	Conf  []bool
	Load  []float64
	Force map[string][]float64

	Weight []float64
	Factor float64

	Disp     map[string][]float64
	Reaction map[string][]float64

	Pile *Pile

	Pcoord []float64
	Dcoord []float64

	Hide bool
	Lock bool
}

// Sort// {{{
type Nodes []*Node

func (n Nodes) Len() int { return len(n) }
func (n Nodes) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}

type NodeByNum struct{ Nodes }

func (n NodeByNum) Less(i, j int) bool {
	return n.Nodes[i].Num < n.Nodes[j].Num
}

type NodeByXCoord struct{ Nodes }
type NodeByYCoord struct{ Nodes }
type NodeByZCoord struct{ Nodes }

func (n NodeByXCoord) Less(i, j int) bool {
	return n.Nodes[i].Coord[0] < n.Nodes[j].Coord[0]
}
func (n NodeByYCoord) Less(i, j int) bool {
	return n.Nodes[i].Coord[1] < n.Nodes[j].Coord[1]
}
func (n NodeByZCoord) Less(i, j int) bool {
	return n.Nodes[i].Coord[2] < n.Nodes[j].Coord[2]
}

type NodeByPcoordX struct{ Nodes }
type NodeByPcoordY struct{ Nodes }

func (n NodeByPcoordX) Less(i, j int) bool {
	return n.Nodes[i].Pcoord[0] < n.Nodes[j].Pcoord[0]
}
func (n NodeByPcoordY) Less(i, j int) bool {
	return n.Nodes[i].Pcoord[1] < n.Nodes[j].Pcoord[1]
}

// }}}

// New
func NewNode() *Node {
	return &Node{Coord: make([]float64, 3),
		Pcoord:   make([]float64, 2),
		Dcoord:   make([]float64, 2),
		Conf:     make([]bool, 6),
		Load:     make([]float64, 6),
		Weight:   make([]float64, 3),
		Disp:     make(map[string][]float64),
		Force:    make(map[string][]float64),
		Reaction: make(map[string][]float64)}
}

func (node *Node) InpString() string {
	var rtn bytes.Buffer
	rtn.WriteString(fmt.Sprintf("NODE %4d  CORD %7.3f %7.3f %7.3f  ICON ", node.Num, node.Coord[0], node.Coord[1], node.Coord[2]))
	for i := 0; i < 6; i++ {
		if node.Conf[i] {
			rtn.WriteString("1 ")
		} else {
			rtn.WriteString("0 ")
		}
	}
	rtn.WriteString(" VCON  ")
	for i := 0; i < 6; i++ {
		rtn.WriteString(fmt.Sprintf(" %12.8f", node.Load[i]))
	}
	if node.Pile != nil {
		rtn.WriteString(fmt.Sprintf("  PCON %d", node.Pile.Num))
	}
	rtn.WriteString("\n")
	return rtn.String()
}

func (node *Node) CopyString(x, y, z float64) string {
	var rtn bytes.Buffer
	rtn.WriteString(fmt.Sprintf("NODE %4d  CORD %7.3f %7.3f %7.3f  ICON ", node.Num, node.Coord[0]-x, node.Coord[1]-y, node.Coord[2]-z))
	for i := 0; i < 6; i++ {
		if node.Conf[i] {
			rtn.WriteString("1 ")
		} else {
			rtn.WriteString("0 ")
		}
	}
	rtn.WriteString(" VCON  ")
	for i := 0; i < 6; i++ {
		rtn.WriteString(fmt.Sprintf(" %12.8f", node.Load[i]))
	}
	if node.Pile != nil {
		rtn.WriteString(fmt.Sprintf("  PCON %d", node.Pile.Num))
	}
	rtn.WriteString("\n")
	return rtn.String()
}

func (node *Node) WgtString() string {
	return fmt.Sprintf("%9d  %10.3f %10.3f %10.3f\n", node.Num, node.Weight[0], node.Weight[1], node.Weight[2])
}

func (node *Node) InlCoordString() string {
	return fmt.Sprintf("%5d %7.3f %7.3f %7.3f\n", node.Num, node.Coord[0], node.Coord[1], node.Coord[2])
}

func (node *Node) InlConditionString(period int) string {
	var rtn bytes.Buffer
	rtn.WriteString(fmt.Sprintf("%5d", node.Num))
	for i := 0; i < 6; i++ {
		if node.Conf[i] {
			rtn.WriteString(" 1")
		} else {
			rtn.WriteString(" 0")
		}
	}
	for i := 0; i < 6; i++ {
		val := node.Load[i]
		if period == 0 && i == 2 {
			if !node.Conf[2] {
				val -= node.Weight[1]
			}
		} else if (period == 1 && i == 0) || (period == 2 && i == 1) {
			if !node.Conf[period-1] {
				val += node.Factor * node.Weight[2]
			}
		}
		rtn.WriteString(fmt.Sprintf(" %12.8f", val))
	}
	rtn.WriteString("\n")
	return rtn.String()
}

func (node *Node) OutputDisp(p string) string {
	var rtn bytes.Buffer
	rtn.WriteString(fmt.Sprintf("%4d ", node.Num))
	for i := 0; i < 3; i++ {
		rtn.WriteString(fmt.Sprintf("% 10.6f ", node.Disp[p][i]))
	}
	for i := 3; i < 5; i++ {
		rtn.WriteString(fmt.Sprintf("% 11.7f ", node.Disp[p][i]))
	}
	rtn.WriteString(fmt.Sprintf("% 11.7f\n", node.Disp[p][5]))
	return rtn.String()
}

func (node *Node) OutputReaction(p string, ind int) string {
	return fmt.Sprintf(" %4d %10d %14.6f     1\n", node.Num, ind+1, node.Reaction[p][ind])
}

func (node *Node) Move(x, y, z float64) {
	node.Coord[0] += x
	node.Coord[1] += y
	node.Coord[2] += z
}

func (node *Node) MoveTo(x, y, z float64) {
	node.Coord[0] = x
	node.Coord[1] = y
	node.Coord[2] = z
}

func (node *Node) MoveToLine(n1, n2 *Node, fixed int) error {
	if fixed < 0 || fixed > 3 {
		return errors.New("MoveToLine: Index out of range")
	}
	d := Direction(n1, n2, false)
	if d[fixed] == 0.0 {
		return errors.New("MoveToLine: Zero Division")
	}
	k := (node.Coord[fixed] - n1.Coord[fixed]) / d[fixed]
	for i := 0; i < 3; i++ {
		node.Coord[i] = n1.Coord[i] + k*d[i]
	}
	return nil
}

func (node *Node) Rotate(center, vector []float64, angle float64) {
	c := RotateVector(node.Coord, center, vector, angle)
	node.MoveTo(c[0], c[1], c[2])
}

func (node *Node) Scale(center []float64, val float64) {
	c := make([]float64, 3)
	for i := 0; i < 3; i++ {
		c[i] = (node.Coord[i]-center[i])*val + center[i]
	}
	node.MoveTo(c[0], c[1], c[2])
}

func (node *Node) MirrorCoord(coord, vec []float64) []float64 {
	vec = Normalize(vec)
	rtn := make([]float64, 3)
	var dot float64
	for j := 0; j < 3; j++ {
		dot += (node.Coord[j] - coord[j]) * vec[j]
	}
	for j := 0; j < 3; j++ {
		rtn[j] = node.Coord[j] - 2.0*dot*vec[j]
	}
	return rtn
}

func (node *Node) Normal(normalize bool) []float64 {
	ns := node.Frame.LineConnected(node)
	if len(ns) == 0 {
		return nil
	}
	vec := make([]float64, 3)
	for _, n := range ns {
		tmp := Direction(node, n, true)
		for j := 0; j < 3; j++ {
			vec[j] += tmp[j]
		}
	}
	var l float64
	if normalize {
		l = Dot(vec, vec, 3)
		l = math.Sqrt(l)
	} else {
		l = float64(len(ns))
	}
	for j := 0; j < 3; j++ {
		vec[j] /= l
	}
	return vec
}

// Disp
func (node *Node) ReturnDisp(period string, index int) float64 {
	if period == "" {
		return 0.0
	}
	if pind := strings.Index(period, "+"); pind >= 0 {
		return node.ReturnDisp(period[:pind], index) + node.ReturnDisp(period[pind+1:], index)
	}
	if mind := strings.Index(period, "-"); mind >= 0 {
		ps := strings.Split(period, "-")
		val := node.ReturnDisp(ps[0], index)
		for i := 1; i < len(ps); i++ {
			val -= node.ReturnDisp(ps[i], index)
		}
		return val
	}
	if val, ok := node.Disp[period]; ok {
		return val[index]
	} else {
		return 0.0
	}
}

func (node *Node) ReturnReaction(period string, index int) float64 {
	if period == "" {
		return 0.0
	}
	if pind := strings.Index(period, "+"); pind >= 0 {
		return node.ReturnReaction(period[:pind], index) + node.ReturnReaction(period[pind+1:], index)
	}
	if mind := strings.Index(period, "-"); mind >= 0 {
		ps := strings.Split(period, "-")
		val := node.ReturnReaction(ps[0], index)
		for i := 1; i < len(ps); i++ {
			val -= node.ReturnReaction(ps[i], index)
		}
		return val
	}
	if val, ok := node.Reaction[period]; ok {
		return val[index]
	} else {
		return 0.0
	}
}

// Conf
func (node *Node) IsPinned() bool {
	for i := 3; i < 6; i++ {
		if node.Conf[i] {
			return false
		}
	}
	for i := 0; i < 3; i++ {
		if !node.Conf[i] {
			return false
		}
	}
	return true
}

func (node *Node) IsRollered() bool {
	for i := 3; i < 6; i++ {
		if node.Conf[i] {
			return false
		}
	}
	for i := 0; i < 3; i++ {
		if !node.Conf[i] {
			return false
		}
	}
	return true
}

func (node *Node) IsFixed() bool {
	for i := 5; i >= 0; i-- {
		if !node.Conf[i] {
			return false
		}
	}
	return true
}

func (node *Node) ConfState() int {
	rtn := 0
	tmp := 1
	for i := 0; i < 6; i++ {
		if node.Conf[i] {
			rtn += tmp
		}
		tmp <<= 1
	}
	return rtn
}

// Miscellaneous// {{{
func Distance(n1, n2 *Node) float64 {
	sum := 0.0
	for i := 0; i < 3; i++ {
		sum += math.Pow((n2.Coord[i] - n1.Coord[i]), 2)
	}
	return math.Sqrt(sum)
}

func Direction(n1, n2 *Node, normalize bool) []float64 {
	vec := make([]float64, 3)
	var l float64
	if normalize {
		l = Distance(n1, n2)
	} else {
		l = 1.0
	}
	for i := 0; i < 3; i++ {
		vec[i] = (n2.Coord[i] - n1.Coord[i]) / l
	}
	return vec
}

func ModifyEnod(ns []*Node) []*Node {
	l := len(ns)
	vecs := make([][]float64, l)
	for i := 0; i < l-1; i++ {
		tmp := make([]float64, 3)
		for j := 0; j < 3; j++ {
			tmp[j] = ns[i+1].Coord[j] - ns[i].Coord[j]
		}
		vecs[i] = tmp
	}
	tmp := make([]float64, 3)
	for j := 0; j < 3; j++ {
		tmp[j] = ns[0].Coord[j] - ns[l-1].Coord[j]
	}
	vecs[l-1] = tmp
	rtn := make([]*Node, l)
	rtn[0] = ns[0]
	num := 1
	for i := 0; i < l-1; i++ {
		if !IsParallel(vecs[i], vecs[i+1], 5e-3) {
			rtn[num] = ns[i+1]
			num++
		}
	}
	if !IsParallel(vecs[l-1], vecs[0], 5e-3) {
		return rtn[:num]
	} else {
		return rtn[1:num]
	}
}

func IsUpside(ns []*Node) bool {
	l := len(ns)
	if l <= 1 {
		return true
	}
	rtn := false
	for i := 2; i >= 0; i-- {
		tmp := ns[0].Coord[i]
		for _, n := range ns[1:] {
			if n.Coord[i] < tmp {
				return false
			} else if n.Coord[i] > tmp {
				tmp = n.Coord[i]
				rtn = true
			}
		}
		if rtn {
			return true
		}
	}
	return true
}

func Upside(ns []*Node) []*Node {
	l := len(ns)
	if l <= 1 {
		return ns
	}
	if l == 2 {
		if ns[0].Coord[2] == ns[1].Coord[2] {
			if ns[0].Coord[1] == ns[1].Coord[1] {
				if ns[0].Coord[0] < ns[1].Coord[0] {
					return []*Node{ns[0], ns[1]}
				} else {
					return []*Node{ns[1], ns[0]}
				}
			} else {
				if ns[0].Coord[1] < ns[1].Coord[1] {
					return []*Node{ns[0], ns[1]}
				} else {
					return []*Node{ns[1], ns[0]}
				}
			}
		} else {
			if ns[0].Coord[2] < ns[1].Coord[2] {
				return []*Node{ns[0], ns[1]}
			} else {
				return []*Node{ns[1], ns[0]}
			}
		}
	}
	rtn := make([]*Node, l)
	miny := ns[0].Coord[1]
	minz := ns[0].Coord[2]
	indz := 0
	indy := 0
	var ind, compare int
	sortbyz := false
	for i, n := range ns[1:] {
		tmpy := n.Coord[1]
		tmpz := n.Coord[2]
		if tmpz != minz {
			sortbyz = true
			if tmpz < minz {
				minz = tmpz
				indz = i + 1
			}
		}
		if tmpy < miny {
			miny = tmpy
			indy = i + 1
		}
	}
	if sortbyz {
		compare = 2
		ind = indz
	} else {
		compare = 1
		ind = indy
	}
	next := func(n int) int {
		rtn := n + 1
		for {
			if rtn < l {
				return rtn
			}
			rtn -= l
		}
	}
	previous := func(n int) int {
		rtn := n - 1
		for {
			if rtn >= 0 {
				return rtn
			}
			rtn += l
		}
	}
	ring := func(n int) int {
		rtn := n
		for {
			if rtn < 0 {
				rtn += l
			} else if rtn >= l {
				rtn -= l
			} else {
				return rtn
			}
		}
	}
	rtn[0] = ns[ind]
	if ns[next(ind)].Coord[compare] < ns[previous(ind)].Coord[compare] {
		for i := 1; i < l; i++ {
			rtn[i] = ns[ring(ind+i)]
		}
	} else {
		for i := 1; i < l; i++ {
			rtn[i] = ns[ring(ind-i)]
		}
	}
	return rtn
}

func CompareNodes(ns1, ns2 []*Node) bool {
	if len(ns1) != len(ns2) {
		return false
	}
	n1 := make([]*Node, len(ns1))
	n2 := make([]*Node, len(ns2))
	for i := 0; i < len(n1); i++ {
		n1[i] = ns1[i]
		n2[i] = ns2[i]
	}
	sort.Sort(NodeByNum{n1})
	sort.Sort(NodeByNum{n2})
	for i := 0; i < len(n1); i++ {
		if n1[i] != n2[i] {
			return false
		}
	}
	return true
}

// }}}
