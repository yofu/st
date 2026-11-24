package st

import (
	"fmt"
	"math"
	"strconv"
)

// Keep the usual one decimal for whole values without rounding off fractional
// lengths or factors (for example BTFAC 0.75) during an LST round trip.
func steelLengthPair(keyword string, values []float64) string {
	number := func(v float64) string {
		if v == math.Trunc(v) {
			return strconv.FormatFloat(v, 'f', 1, 64)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
	return fmt.Sprintf("         %s %5s %5s\n", keyword, number(values[0]), number(values[1]))
}

// SteelLengthOptions uses cm. Zero leaves an input unspecified. Array entries
// correspond to the local X and Y axes (the order of the two LST values).
type SteelLengthOptions struct {
	DesignCM              float64
	BucklingCM, LateralCM [2]float64
}

// SteelAdoptedLength records the winning rule. Factor is used only when the
// length comes from the design span. All lengths in this report are in cm.
type SteelAdoptedLength struct {
	CM     float64
	Source string
	Factor float64
}

type SteelMemberLengths struct {
	Element, Section  int
	DesignCM          float64
	DesignSource      string
	EndNodes          [2]int
	Buckling, Lateral [2]SteelAdoptedLength
	Review            string // why model geometry could not supply a design span
	rate              *SColumn
	girder            bool
}

// SteelLengthResolver indexes incident elements, including plates. Construct
// one per query/batch and rebuild after topology edits.
type SteelLengthResolver struct {
	frame    *Frame
	incident map[*Node][]*Elem
}

func NewSteelLengthResolver(f *Frame) *SteelLengthResolver {
	r := &SteelLengthResolver{frame: f, incident: make(map[*Node][]*Elem)}
	for _, el := range f.Elems {
		for _, n := range el.Enod {
			r.incident[n] = append(r.incident[n], el)
		}
	}
	return r
}

func steelLengthRate(el *Elem) (*SColumn, error) {
	rate := el.Sect.Allow
	if rate == nil && len(el.Sect.Figs) == 1 {
		fig := el.Sect.Figs[0]
		if fig != nil && fig.Prop != nil {
			if steel, ok := fig.Prop.Material.(Steel); ok && fig.Shape != nil {
				// Looking up lengths must not populate Sect.Allow during a preview.
				return NewSColumn(el.Sect.Num, fig.Shape, steel), nil
			}
		}
	}
	switch sr := rate.(type) {
	case *SColumn:
		return sr, nil
	case *SGirder:
		return &sr.SColumn, nil
	default:
		return nil, fmt.Errorf("SECT %d: steel column/girder section data required", el.Sect.Num)
	}
}

func axisSetting(values []float64, axis int) float64 {
	if axis < len(values) {
		return values[axis]
	}
	return 0
}

func steelLengthRule(manual, absolute, factor float64, absKey, factorKey string) (SteelAdoptedLength, error) {
	switch {
	case manual != 0:
		if !positiveYieldInputs(manual) {
			return SteelAdoptedLength{}, fmt.Errorf("invalid explicit %s length %g", absKey, manual)
		}
		return SteelAdoptedLength{CM: manual, Source: "explicit"}, nil
	case absolute != 0:
		if !positiveYieldInputs(absolute) {
			return SteelAdoptedLength{}, fmt.Errorf("invalid %s %g", absKey, absolute)
		}
		return SteelAdoptedLength{CM: absolute, Source: absKey}, nil
	case factor != 0:
		if !positiveYieldInputs(factor) {
			return SteelAdoptedLength{}, fmt.Errorf("invalid %s %g", factorKey, factor)
		}
		return SteelAdoptedLength{Factor: factor, Source: factorKey}, nil
	default:
		return SteelAdoptedLength{Factor: 1, Source: "design"}, nil
	}
}

// Resolve determines four local-axis lengths independently. If an inferred
// span is ambiguous, Review identifies the joint; absolute values remain usable
// and only the entries needing a design span have CM == 0.
func (r *SteelLengthResolver) Resolve(el *Elem, opt SteelLengthOptions) (SteelMemberLengths, error) {
	var out SteelMemberLengths
	if el == nil || el.Sect == nil || r.frame.Elems[el.Num] != el || (el.Etype != COLUMN && el.Etype != GIRDER) {
		return out, fmt.Errorf("steel lengths require a column or girder in the current frame")
	}
	rate, err := steelLengthRate(el)
	if err != nil {
		return out, err
	}
	out = SteelMemberLengths{Element: el.Num, Section: el.Sect.Num, rate: rate, girder: el.Etype == GIRDER}
	if opt.DesignCM != 0 && !positiveYieldInputs(opt.DesignCM) {
		return out, fmt.Errorf("invalid design length %g cm", opt.DesignCM)
	}
	var fromDesign []*SteelAdoptedLength
	for i := 0; i < 2; i++ {
		out.Buckling[i], err = steelLengthRule(opt.BucklingCM[i], axisSetting(rate.BBLength, i), axisSetting(rate.BBFactor, i), "BBLEN", "BBFAC")
		if err != nil {
			return out, err
		}
		out.Lateral[i], err = steelLengthRule(opt.LateralCM[i], axisSetting(rate.BTLength, i), axisSetting(rate.BTFactor, i), "BTLEN", "BTFAC")
		if err != nil {
			return out, err
		}
		for _, v := range []*SteelAdoptedLength{&out.Buckling[i], &out.Lateral[i]} {
			if v.Factor != 0 {
				fromDesign = append(fromDesign, v)
			}
		}
	}
	if opt.DesignCM != 0 {
		out.DesignCM, out.DesignSource = opt.DesignCM, "explicit"
	} else if len(fromDesign) > 0 {
		ends, err := r.designEnds(el)
		if err != nil {
			out.Review = err.Error()
			return out, nil
		}
		out.DesignCM = 100 * Distance(ends[0], ends[1])
		out.EndNodes = [2]int{ends[0].Num, ends[1].Num}
		out.DesignSource = "geometry"
	}
	for _, v := range fromDesign {
		v.CM = out.DesignCM * v.Factor
		if !positiveYieldInputs(v.CM) {
			return out, fmt.Errorf("invalid %s length for ELEM %d", v.Source, el.Num)
		}
	}
	return out, nil
}

// Walk through ordinary mesh nodes, independently of CHAIN tags, selection,
// display state and nodal loads. A collinear continuation at a multiway joint
// does not establish which direction is braced. Require a specified span there.
func (r *SteelLengthResolver) designEnds(el *Elem) ([2]*Node, error) {
	var ends [2]*Node
	seen := map[*Elem]bool{el: true}
	for side := range ends {
		current, node := el, el.Enod[side]
		for {
			ends[side] = node
			if node.Conf[0] && node.Conf[1] && node.Conf[2] {
				break
			}
			away := Direction(node, current.Otherside(node), true)
			var next *Elem
			bentChain := false
			for _, other := range r.incident[node] {
				if other == current || !other.IsLineElem() {
					continue
				}
				dir := Direction(node, other.Otherside(node), true)
				sameChain := current.Chain != nil && current.Chain == other.Chain
				if Dot(away, dir, 3) < 0 && (IsParallel(away, dir, 1e-7) ||
					sameChain && onSteelSpan(node, current.Otherside(node), other.Otherside(node))) {
					next = other
					break
				}
				if sameChain {
					bentChain = true
				}
			}
			if next == nil && bentChain {
				return ends, fmt.Errorf("NODE %d: bent chain; specify design or effective lengths", node.Num)
			}
			if next == nil {
				break
			}
			if len(r.incident[node]) != 2 || node.ConfState() != 0 || node.Pile != nil ||
				next.Sect != el.Sect || next.Etype != el.Etype || !sameSteelOrientation(current, next) ||
				steelEndReleased(current, node) || steelEndReleased(next, node) {
				return ends, fmt.Errorf("NODE %d: confirm bracing and specify design or effective lengths", node.Num)
			}
			if seen[next] {
				return ends, fmt.Errorf("NODE %d: cyclic design span", node.Num)
			}
			seen[next] = true
			current, node = next, next.Otherside(node)
		}
	}
	// Local roundoff at successive nodes must not join an actual curved member.
	for part := range seen {
		for _, node := range part.Enod {
			if !onSteelSpan(node, ends[0], ends[1]) {
				return ends, fmt.Errorf("NODE %d: non-straight design span; specify design or effective lengths", node.Num)
			}
		}
	}
	return ends, nil
}

// INP writes coordinates to 0.001 m. A rounded point and its rounded chord
// can differ by at most 0.001 m per coordinate.
func onSteelSpan(n, a, b *Node) bool {
	offset := Cross(Direction(a, b, true), Direction(a, n, false))
	return Dot(offset, offset, 3) <= 3e-6
}

func sameSteelOrientation(a, b *Elem) bool {
	if IsParallel(a.Strong, b.Strong, 1e-7) {
		return true
	}
	// The principal directions of a split oblique member can differ slightly
	// after an INP round trip, while its section angle and orientation persist.
	return a.Chain != nil && a.Chain == b.Chain && a.Cang == b.Cang &&
		Dot(a.Direction(true), b.Direction(true), 3) > 0 && math.Abs(Dot(a.Strong, b.Strong, 3)) > 0.99999
}

func steelEndReleased(el *Elem, n *Node) bool {
	start := 0
	if el.Enod[1] == n {
		start = 6
	}
	for _, bond := range el.Bonds[start : start+6] {
		if bond != nil {
			return true
		}
	}
	return false
}

// Yield feeds the resolved lengths directly to the section calculation. Supply
// a product for cold-formed columns. No LST length factors are applied here.
func (l SteelMemberLengths) Yield(product *ColdFormedTubeProduct, momentRatio, plateFactor, strengthFactor float64) ([12]float64, error) {
	var zero [12]float64
	if l.rate == nil {
		return zero, fmt.Errorf("unresolved steel member")
	}
	x, y := l.Buckling[0].CM, l.Buckling[1].CM
	bx, by := l.Lateral[0].CM, l.Lateral[1].CM
	c := SteelYieldCondition{StrongBucklingLength: x, WeakBucklingLength: y, EndMomentRatio: momentRatio, PlateBucklingFactor: plateFactor, StrengthFactor: strengthFactor}
	var majorY bool
	if l.girder {
		_, majorY = l.rate.Shape.(HWEAK)
	} else {
		if product == nil {
			return zero, fmt.Errorf("SECT %d requires cold-formed tube product properties", l.Section)
		}
		majorY = product.Iy > product.Ix
	}
	if majorY {
		c.StrongBucklingLength, c.WeakBucklingLength = y, x
	}
	c.LateralLength = bx
	if majorY {
		c.LateralLength = by
	}
	// The tube sheet reduces the axis with larger plastic modulus and both axes on a tie.
	if !l.girder {
		switch {
		case product.Zpx > product.Zpy:
			c.LateralLength = bx
		case product.Zpy > product.Zpx:
			c.LateralLength = by
		default:
			c.LateralLength = math.Min(bx, by)
		}
	}
	if !positiveYieldInputs(x, y, c.LateralLength) {
		return zero, fmt.Errorf("ELEM %d: unresolved effective length: %s", l.Element, l.Review)
	}
	if l.girder {
		return (&SGirder{*l.rate}).HBeamYield(c)
	}
	return l.rate.coldFormedTubeYield(*product, c, bx, by)
}
