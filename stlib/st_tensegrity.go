package st

import (
	"fmt"
	"math"
	"slices"
)

// TensegrityAddCandidates constructs bars from each selected compression end
// to the nearest nodes inside the member's longitudinal span.
func TensegrityAddCandidates(frame *Frame, elems []*Elem, compression, tension, size int) ([]*Elem, error) {
	if size < 0 {
		return nil, fmt.Errorf("size must be nonnegative")
	}
	sec := frame.Sects[tension]
	if sec == nil {
		return nil, fmt.Errorf("unknown tension section %d", tension)
	}
	var added []*Elem
	for _, el := range elems {
		if el == nil || !el.IsLineElem() || el.Sect.Num != compression {
			continue
		}
		for end, origin := range el.Enod {
			nodes := el.BetweenNode(end, -1)
			slices.SortFunc(nodes, func(a, b *Node) int {
				da, db := Distance(origin, a), Distance(origin, b)
				if da < db {
					return -1
				}
				if da > db {
					return 1
				}
				return a.Num - b.Num
			})
			for _, n := range nodes[:min(size, len(nodes))] {
				added = append(added, NewLineElem([]*Node{origin, n}, sec, COLUMN))
			}
		}
	}
	return added, nil
}

// TensegrityDeleteCandidates ranks tension bars by their minimum absolute
// design axial force. Missing force results make the ranking invalid.
func TensegrityDeleteCandidates(frame *Frame, nodes []*Node, tension, size int) ([]*Elem, error) {
	if size < 0 {
		return nil, fmt.Errorf("size must be nonnegative")
	}
	var remove []*Elem
	seen := make(map[*Elem]bool)
	strength := make(map[*Elem]float64)
	for _, n := range nodes {
		if n == nil {
			continue
		}
		var bars []*Elem
		for _, el := range frame.SearchElem(n) {
			if el.IsLineElem() && el.Sect.Num == tension {
				bars = append(bars, el)
			}
		}
		if len(bars) <= size {
			continue
		}
		slices.SortFunc(bars, func(a, b *Elem) int { return a.Num - b.Num })
		for _, el := range bars {
			if _, ok := strength[el]; ok {
				continue
			}
			for _, p := range []string{"L", "X", "Y"} {
				v := el.Stress[p][el.Enod[0].Num]
				if len(v) == 0 || math.IsNaN(v[0]) || math.IsInf(v[0], 0) {
					return nil, fmt.Errorf("element %d lacks valid %s axial-force results; load matching L, X and Y result files", el.Num, p)
				}
			}
			value := 1.5 * math.Abs(el.N("L", 0))
			for _, p := range []string{"L+X", "L-X", "L+Y", "L-Y"} {
				value = min(value, math.Abs(el.N(p, 0)))
			}
			strength[el] = value
		}
		slices.SortStableFunc(bars, func(a, b *Elem) int {
			if strength[a] < strength[b] {
				return -1
			}
			if strength[a] > strength[b] {
				return 1
			}
			return 0
		})
		for _, el := range bars[:len(bars)-size] {
			if !seen[el] {
				remove = append(remove, el)
				seen[el] = true
			}
		}
	}
	return remove, nil
}

// TensegrityConnected selects connected compression bars in element order.
// For a tension bar it selects at most one distinct compression bar per end.
func TensegrityConnected(frame *Frame, first *Elem, compression, tension int) []*Elem {
	if first == nil || !first.IsLineElem() {
		return nil
	}
	var bars []*Elem
	for _, el := range frame.Elems {
		if el.IsLineElem() && el.Sect.Num == compression {
			bars = append(bars, el)
		}
	}
	slices.SortFunc(bars, func(a, b *Elem) int { return a.Num - b.Num })
	var selected []*Elem
	switch first.Sect.Num {
	case compression:
		linked := make(map[*Node]bool)
		for _, el := range frame.Elems {
			if el.IsLineElem() {
				a, b := el.Enod[0], el.Enod[1]
				if slices.Contains(first.Enod, a) {
					linked[b] = true
				}
				if slices.Contains(first.Enod, b) {
					linked[a] = true
				}
			}
		}
		for _, el := range bars {
			if linked[el.Enod[0]] || linked[el.Enod[1]] {
				selected = append(selected, el)
			}
		}
	case tension:
		var found [2]bool
		for _, el := range bars {
			for i, n := range first.Enod {
				if !found[i] && slices.Contains(el.Enod, n) {
					found[i] = true
					selected = append(selected, el)
					break
				}
			}
			if found[0] && found[1] {
				break
			}
		}
	}
	return selected
}
