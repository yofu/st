package st

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Preserve the LST hoop ratio. At its six-decimal precision, recover discarded
// digits only if the INP diameter, spacing, leg count and grade agree.
func rcHoopRatio(rc *RCColumn, s *Sect, axis int) (float64, string) {
	pw := rc.Hoops.Ps[axis]
	if len(s.Figs) != 1 || s.Figs[0] == nil {
		return pw, "HOOPS"
	}
	spec := s.Figs[0].Reins["HOOP"]
	if len(spec) != 2 || spec[1] != rc.Hoops.Material.Name {
		return pw, "HOOPS"
	}
	match := hoopstr.FindStringSubmatch(spec[0])
	if match == nil {
		return pw, "HOOPS"
	}
	legs, _ := strconv.Atoi(match[1+axis])
	pitch, _ := strconv.ParseFloat(match[4], 64)
	precise := float64(legs) * reinsarea[match[3]] * 100 / (pitch * rc.Breadth(axis == 1) * 10)
	if positiveYieldInputs(precise) && (math.Abs(precise-pw) < 1e-14 || math.Abs(math.Round(precise*1e6)/1e6-pw) < 1e-12) {
		return precise, "HOOP " + spec[0]
	}
	return pw, "HOOPS"
}
func rcShearSection(rc *RCColumn, s *Sect, axis int, depth float64) (RCShearInput, string, error) {
	strong := axis == 1
	height := rc.Height(strong) * 10
	if depth == 0 {
		depth = height - 100
	}
	if !positiveYieldInputs(depth) || depth > height {
		return RCShearInput{}, "", fmt.Errorf("invalid effective depth %g mm (D=%g)", depth, height)
	}
	var side [2]float64
	for _, bar := range rc.Reins {
		x := bar.Position[axis]
		if x < 0 {
			side[0] += bar.Area
		} else if x > 0 {
			side[1] += bar.Area
		}
	}
	var fy float64
	switch rc.Hoops.Material.Name {
	case "SD295":
		fy = 295
	case "SD345":
		fy = 345
	case "SD390":
		fy = 390
	default:
		return RCShearInput{}, "", fmt.Errorf("unsupported hoop grade %q", rc.Hoops.Material.Name)
	}
	pw, source := rcHoopRatio(rc, s, axis)
	// Use the smaller tension face for a single signed pair of shear capacities.
	return RCShearInput{Width: rc.Breadth(strong) * 10, Depth: depth, TensionArea: math.Min(side[0], side[1]) * 100, HoopRatio: pw, HoopStrength: fy, ConcreteStrength: rc.fc * 100}, source, nil
}

// Worksheet M and Q are magnitudes. Take the element's maxima over its two
// ends for the selected result key; never combine different members or steps.
func rcMemberForces(el *Elem, period string, axis int) (rcForcePair, error) {
	var p rcForcePair
	stresses := el.Stress[period]
	if len(el.Enod) != 2 {
		return p, fmt.Errorf("ELEM %d has no pair of end nodes", el.Num)
	}
	for _, n := range el.Enod {
		v := stresses[n.Num]
		if len(v) < 6 {
			return p, fmt.Errorf("ELEM %d NODE %d: no complete forces for %s", el.Num, n.Num, period)
		}
		m, q := v[5-axis], v[1+axis]
		if !finiteRC(m) || !finiteRC(q) {
			return p, fmt.Errorf("ELEM %d: nonfinite result", el.Num)
		}
		p.m = math.Max(p.m, math.Abs(m)*rcGravity)
		p.q = math.Max(p.q, math.Abs(q)*rcGravity)
	}
	if p.q == 0 {
		return p, fmt.Errorf("ELEM %d %s: Q%s=0; specify a different result or an M,Q pair", el.Num, period, rcAxis(axis))
	}
	return p, nil
}

func (r rcYieldRequest) selectShear(out *strings.Builder, rc *RCColumn, s *Sect, members []*Elem, axis int) (float64, error) {
	v, src, err := rcShearSection(rc, s, axis, r.depth[axis])
	if err != nil {
		return 0, err
	}
	fmt.Fprintf(out, "  Q%s 梁Qsu式 B=%g d=%g j=%g mm At=%g mm² pw=%g (%s) Fc=%g σwy=%g N/mm²\n", rcAxis(axis), v.Width, v.Depth, .875*v.Depth, v.TensionArea, v.HoopRatio, src, v.ConcreteStrength, v.HoopStrength)
	report := func(p rcForcePair, source string) (float64, error) {
		v.Moment, v.Shear = p.m, p.q
		q, err := v.UltimateShear()
		if err != nil {
			return 0, err
		}
		fmt.Fprintf(out, "    %s M=%g kN·m Q=%g kN |M/Qd|=%.6f Qsu=%.6f kN", source, p.m, p.q, q.SpanRatio, q.KN)
		if q.SpanRatio < 1 || q.SpanRatio > 3 {
			fmt.Fprint(out, "  1～3の範囲外（Excel式、制限なし）")
		}
		fmt.Fprintln(out)
		return q.KN / rcGravity, nil
	}
	if p := r.explicit[axis]; p != nil {
		return report(*p, "明示値")
	}
	best := math.Inf(1)
	winner := 0
	var first float64
	conflict := false
	for _, el := range members {
		if r.member != 0 && r.member != el.Num {
			continue
		}
		p, err := rcMemberForces(el, r.period, axis)
		if err != nil {
			return 0, err
		}
		q, err := report(p, fmt.Sprintf("%s ELEM %d", r.period, el.Num))
		if err != nil {
			return 0, err
		}
		if winner == 0 {
			first = q
		} else if math.Round(q*1000) != math.Round(first*1000) {
			conflict = true
		}
		if q < best {
			best = q
			winner = el.Num
		}
	}
	if winner == 0 {
		return 0, fmt.Errorf("reference member no longer belongs to SECT %d", s.Num)
	}
	if conflict && !r.minimum {
		return 0, fmt.Errorf("Q%s candidates differ: specify -member=N or -pick=min", rcAxis(axis))
	}
	fmt.Fprintf(out, "    Q%s 採用 ELEM %d (%s)\n", rcAxis(axis), winner, r.period)
	return best, nil
}
