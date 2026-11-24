package st

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func steelLengthCommand(stw ExModer, input []interface{}, args map[string]string) error {
	frame := stw.Frame()
	var elems []*Elem
	if raw, ok := args["ELEM"]; ok {
		n, err := strconv.Atoi(raw)
		if err != nil || frame.Elems[n] == nil {
			return fmt.Errorf("yieldlength: invalid ELEM %q", raw)
		}
		elems = []*Elem{frame.Elems[n]}
	} else {
		elems = append(elems, currentelem(stw, input)...)
	}
	if len(elems) == 0 {
		return fmt.Errorf("yieldlength: select steel columns/girders or specify -elem=N")
	}
	for key := range args {
		switch key {
		case "ELEM", "DESIGN", "LK", "LB", "LKX", "LKY", "LBX", "LBY":
		default:
			return fmt.Errorf("yieldlength: unknown option %q", key)
		}
	}
	opt, err := parseSteelLengthOptions(args)
	if err != nil {
		return err
	}
	sort.Slice(elems, func(i, j int) bool { return elems[i].Num < elems[j].Num })
	resolver := NewSteelLengthResolver(frame)
	var lines []string
	for _, el := range elems {
		l, err := resolver.Resolve(el, opt)
		if err != nil {
			lines = append(lines, fmt.Sprintf("ELEM %d: %v", el.Num, err))
			continue
		}
		lines = append(lines, l.String())
	}
	return Message(strings.Join(lines, "\n"))
}

func parseSteelLengthOptions(args map[string]string) (SteelLengthOptions, error) {
	var opt SteelLengthOptions
	// Apply the two-axis shorthand first, then any axis-specific overrides.
	for _, key := range []string{"DESIGN", "LK", "LB", "LKX", "LKY", "LBX", "LBY"} {
		raw, ok := args[key]
		if !ok {
			continue
		}
		m, err := strconv.ParseFloat(raw, 64)
		cm := 100 * m
		if err != nil || !positiveYieldInputs(cm) {
			return opt, fmt.Errorf("-%s requires a positive length in m", strings.ToLower(key))
		}
		switch key {
		case "DESIGN":
			opt.DesignCM = cm
		case "LK":
			opt.BucklingCM = [2]float64{cm, cm}
		case "LB":
			opt.LateralCM = [2]float64{cm, cm}
		case "LKX":
			opt.BucklingCM[0] = cm
		case "LKY":
			opt.BucklingCM[1] = cm
		case "LBX":
			opt.LateralCM[0] = cm
		case "LBY":
			opt.LateralCM[1] = cm
		}
	}
	return opt, nil
}

func (l SteelMemberLengths) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "ELEM %d  SECT %d", l.Element, l.Section)
	switch l.DesignSource {
	case "geometry":
		fmt.Fprintf(&b, "  設計長 %.4f m (NODE %d–%d)", l.DesignCM/100, l.EndNodes[0], l.EndNodes[1])
	case "explicit":
		fmt.Fprintf(&b, "  設計長 %.4f m (明示)", l.DesignCM/100)
	}
	for i, v := range [4]SteelAdoptedLength{l.Buckling[0], l.Buckling[1], l.Lateral[0], l.Lateral[1]} {
		label := [4]string{"Lkx", "Lky", "Lbx", "Lby"}[i]
		fmt.Fprintf(&b, "\n  %s ", label)
		switch {
		case v.CM == 0 && v.Source == "design":
			b.WriteString("要確認 (設計長)")
		case v.CM == 0:
			fmt.Fprintf(&b, "要確認 (%s %g × 設計長)", v.Source, v.Factor)
		case v.Source == "explicit":
			fmt.Fprintf(&b, "%.4f m (明示)", v.CM/100)
		case v.Source == "design":
			fmt.Fprintf(&b, "%.4f m (設計長)", v.CM/100)
		case v.Factor != 0:
			fmt.Fprintf(&b, "%.4f m (%s %g × 設計長)", v.CM/100, v.Source, v.Factor)
		default:
			fmt.Fprintf(&b, "%.4f m (%s)", v.CM/100, v.Source)
		}
	}
	if l.Review != "" {
		fmt.Fprintf(&b, "\n  %s", l.Review)
	}
	return b.String()
}
