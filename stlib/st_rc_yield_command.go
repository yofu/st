package st

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/yofu/st/arclm"
)

const rcYieldUsage = ":rcyield [-sect=N[,N...] | -elem=N] [-ndiv=20] [-qx=My[kNm],Qx[kN]] [-qy=Mx[kNm],Qy[kN]] [-dx=mm] [-dy=mm] [-period=X@80 [-axis=x|y|both] [-member=N | -pick=min]] [-nm=block] [-apply=preview_id]"

type rcForcePair struct{ m, q float64 }
type rcYieldRequest struct {
	sections          []int
	divisions, member int
	explicit          [2]*rcForcePair // Qx paired with My; Qy with Mx
	depth             [2]float64
	resultAxes        [2]bool
	period            string
	minimum, block    bool
}

func (r rcYieldRequest) sectionIDs() []int { return r.sections }
func rcYieldCommand(stw ExModer, input []interface{}, args map[string]string) error {
	f := stw.Frame()
	if f == nil {
		return fmt.Errorf("rcyield: open a model first")
	}
	return runSectionYieldCommand(stw, "rcyield", args, &f.rcYieldReview, func() (sectionYieldRequest, error) {
		r := rcYieldRequest{divisions: 20}
		var err error
		r.sections, err = sectionYieldTargets("rcyield", stw, input, args)
		if err != nil {
			return nil, err
		}
		for key, raw := range args {
			switch key {
			case "SECT", "APPLY":
			case "ELEM", "MEMBER":
				if _, e := args["ELEM"]; e {
					if _, m := args["MEMBER"]; m {
						return nil, fmt.Errorf("choose -elem or -member")
					}
				}
				r.member, err = strconv.Atoi(raw)
				el := f.Elems[r.member]
				if err != nil || el == nil || el.Sect == nil || len(r.sections) != 1 || el.Sect.Num != r.sections[0] {
					return nil, fmt.Errorf("invalid reference ELEM %s for selected section", raw)
				}
			case "NDIV":
				r.divisions, err = strconv.Atoi(raw)
				if err != nil || r.divisions < 2 || r.divisions > 1000 {
					return nil, fmt.Errorf("-ndiv must be 2..1000")
				}
			case "DX", "DY":
				axis := 0
				if key == "DY" {
					axis = 1
				}
				r.depth[axis], err = strconv.ParseFloat(raw, 64)
				if err != nil || !positiveYieldInputs(r.depth[axis]) {
					return nil, fmt.Errorf("invalid %s (mm)", key)
				}
			case "QX", "QY":
				axis := 0
				if key == "QY" {
					axis = 1
				}
				m, q, ok := strings.Cut(raw, ",")
				mv, e1 := strconv.ParseFloat(m, 64)
				qv, e2 := strconv.ParseFloat(q, 64)
				if !ok || e1 != nil || e2 != nil || !finiteRC(mv) || !positiveYieldInputs(math.Abs(qv)) {
					return nil, fmt.Errorf("-%s requires moment [kN m], nonzero shear [kN]", strings.ToLower(key))
				}
				r.explicit[axis] = &rcForcePair{mv, qv}
			case "PERIOD":
				r.period = strings.ToUpper(raw)
				if raw == "" {
					return nil, fmt.Errorf("specify the loaded result key, e.g. -period=X@80")
				}
			case "AXIS":
				switch strings.ToLower(raw) {
				case "x":
					r.resultAxes[0] = true
				case "y":
					r.resultAxes[1] = true
				case "both":
					r.resultAxes = [2]bool{true, true}
				default:
					return nil, fmt.Errorf("invalid -axis")
				}
			case "PICK":
				if !strings.EqualFold(raw, "min") {
					return nil, fmt.Errorf("only -pick=min is supported")
				}
				r.minimum = true
			case "NM":
				if !strings.EqualFold(raw, "block") {
					return nil, fmt.Errorf("only -nm=block is supported")
				}
				r.block = true
			default:
				return nil, fmt.Errorf("rcyield: unknown option %s", key)
			}
		}
		if r.period != "" {
			if r.resultAxes == [2]bool{} {
				r.resultAxes = [2]bool{r.explicit[0] == nil, r.explicit[1] == nil}
			}
			if r.resultAxes == [2]bool{} {
				return nil, fmt.Errorf("no axis requested for -period")
			}
			for i, enabled := range r.resultAxes {
				if enabled && r.explicit[i] != nil {
					return nil, fmt.Errorf("duplicate stress source for Q%s", rcAxis(i))
				}
			}
		} else if r.resultAxes != [2]bool{} || r.minimum {
			return nil, fmt.Errorf("-axis/-pick require -period")
		}
		if r.minimum && r.member != 0 {
			return nil, fmt.Errorf("choose one reference member or -pick=min")
		}
		return r, nil
	})
}
func rcAxis(i int) string {
	if i == 0 {
		return "X"
	}
	return "Y"
}

func rcDesignSection(s *Sect, etype int) (*RCColumn, string, error) {
	if etype != GIRDER && etype != COLUMN {
		return nil, "", fmt.Errorf("RC GIRDER / COLUMN required")
	}
	al := s.Allow
	source := "算定断面 (LST/Allow)"
	if al == nil {
		if len(s.Figs) != 1 || s.Figs[0] == nil || s.Figs[0].Prop == nil {
			return nil, "", fmt.Errorf("single rectangular FIG or LST required")
		}
		fig := *s.Figs[0]
		if _, ok := fig.Shape.(PLATE); !ok {
			return nil, "", fmt.Errorf("INP SHAPE PLATE or LST CRECT required")
		}
		if _, ok := fig.Prop.Material.(Concrete); !ok {
			return nil, "", fmt.Errorf("INP MATERIAL missing or not concrete; load LST CRECT/REINS/HOOPS")
		}
		// AutoLayoutReins expands CREIN in place. Preview must not edit the live FIG.
		fig.Reins = maps.Clone(fig.Reins)
		al = fig.GetSectionRate(s.Num, etype)
		source = "INP SHAPE/配筋"
	}
	var rc *RCColumn
	switch v := al.(type) {
	case *RCGirder:
		if etype == GIRDER {
			rc = &v.RCColumn
		}
	case *RCColumn:
		if etype == COLUMN {
			rc = v
		}
	}
	if rc == nil {
		return nil, "", fmt.Errorf("RC design section and member type do not match")
	}
	if err := checkRCSection(rc); err != nil {
		return nil, "", err
	}
	return rcCentered(rc), source, nil
}

func (r rcYieldRequest) preview(f *Frame) (string, map[int][12]float64, bool) {
	var report strings.Builder
	fmt.Fprintf(&report, "RC降伏耐力プレビュー  NDIV=%d  半径=%g  NZ,Q [tf]  M [tf·m]\n", r.divisions, arclm.RADIUS)
	changes := make(map[int][12]float64)
	ready := true
	for _, id := range r.sections {
		s := f.Sects[id]
		var members []*Elem
		if s != nil {
			for _, el := range f.Elems {
				if el.Sect == s {
					members = append(members, el)
				}
			}
		}
		slices.SortFunc(members, func(a, b *Elem) int { return a.Num - b.Num })
		fmt.Fprintf(&report, "SECT %d  使用部材 %d 本\n", id, len(members))
		y, err := r.candidate(&report, s, members)
		if err != nil {
			fmt.Fprintf(&report, "  要確認: %v\n", err)
			ready = false
			continue
		}
		for i, name := range sectionYieldNames {
			fmt.Fprintf(&report, "  %s %g → %.3f\n", name, s.Yield[i], y[i])
		}
		if !slices.Equal(s.Yield, y[:]) {
			changes[id] = y
		}
	}
	return report.String(), changes, ready
}
func (r rcYieldRequest) candidate(out *strings.Builder, s *Sect, members []*Elem) (y [12]float64, err error) {
	if s == nil || len(s.Yield) != 12 || len(members) == 0 {
		return y, fmt.Errorf("使用部材またはSECTの12値がありません")
	}
	if !positiveYieldInputs(s.Exp-1, s.Exq-1) {
		return y, fmt.Errorf("EXP and EXQ must exceed one for material analysis")
	}
	etype := members[0].Etype
	if r.member != 0 {
		if !slices.ContainsFunc(members, func(e *Elem) bool { return e.Num == r.member }) {
			return y, fmt.Errorf("reference ELEM %d no longer belongs to SECT %d", r.member, s.Num)
		}
		fmt.Fprintf(out, "  指定部材 ELEM %d\n", r.member)
	}
	for _, e := range members {
		if e.Etype != etype {
			return y, fmt.Errorf("mixed member types share SECT %d", s.Num)
		}
	}
	rc, source, err := rcDesignSection(s, etype)
	if err != nil {
		return y, err
	}
	copy(y[:], s.Yield)
	symmetric := rcSymmetric(rc)
	uniform := true
	for _, b := range rc.Reins {
		if b.Material.Fs != rc.Reins[0].Material.Fs {
			uniform = false
		}
	}
	var curves [2]rcNMCurve
	fmt.Fprintf(out, "  %s  B×D=%g×%g mm  %s  主筋 %d 本  EXP=%g EXQ=%g\n", source, 10*rc.Breadth(true), 10*rc.Height(true), rc.Concrete.Name(), len(rc.Reins), s.Exp, s.Exq)
	if r.block {
		curves, err = rcBlockCurves(rc, r.divisions)
		if err != nil {
			return y, err
		}
		fmt.Fprintln(out, "  N–M: 平面保持・等価矩形応力ブロック (εcu=0.003, 0.85Fc)。各軸力で正負曲げの小さい側を両材端に採用。")
		fmt.Fprintln(out, "  各軸力で4方向の断面力から直交曲げゼロの軸切片を補間。二軸曲げはSECTのEXPによる近似。")
	} else {
		if !symmetric || !uniform {
			return y, fmt.Errorf("非対称配筋・異種主筋: 正負曲げを個別に評価する -nm=block を指定してください")
		}
		c := NewCondition()
		c.Period = "U"
		for i := range curves {
			c.Strong = i == 0
			curves[i] = rcNMReference(rc, c, r.divisions)
		}
		fmt.Fprintln(out, "  N–M: :nminteraction -period=U")
	}
	y[1] = curves[0].points[r.divisions].n
	y[0] = 2*curves[0].peak.n - y[1]
	for i, cv := range curves {
		j := 8 + 2*i
		y[j], y[j+1] = cv.peak.m, -cv.peak.m
		c := NewCondition()
		c.Strong = i == 0
		a, b := rc.ReinsCentroid(c)
		fmt.Fprintf(out, "  M%s  鉄筋重心間距離=%g cm  最大曲げ時 N=%.6f tf\n", rcAxis(i), a-b, cv.peak.n)
	}
	c := NewCondition()
	c.Period = "S"
	torsion := *rc
	// The legacy torsion expression takes the first bar's strength for
	// all longitudinal bars. Use the weakest grade in mixed layouts.
	if !uniform {
		torsion.Reins = slices.Clone(rc.Reins)
		for _, bar := range rc.Reins {
			if bar.Material.Fs < torsion.Reins[0].Material.Fs {
				torsion.Reins[0].Material = bar.Material
			}
		}
		fmt.Fprintf(out, "  MZ 主筋の採用強度: %s（最小）\n", torsion.Reins[0].Material.Name)
	}
	y[6] = torsion.Mza(c)
	y[7] = -y[6]
	fmt.Fprintln(out, "  MZ: 短期 Mza(S)、既存式のかぶり5 cm・補強筋径1.1 cm")
	if etype == COLUMN && (r.period != "" || r.explicit[0] != nil || r.explicit[1] != nil) {
		fmt.Fprintln(out, "  柱のQにも指定の梁シート式を使用（軸力項なし）")
	}
	for i := range r.explicit {
		if r.explicit[i] == nil && !r.resultAxes[i] {
			fmt.Fprintf(out, "  Q%s 保留（応力未指定、現在値を保持）\n", rcAxis(i))
			continue
		}
		q, err := r.selectShear(out, rc, s, members, i)
		if err != nil {
			return y, err
		}
		y[2+2*i], y[3+2*i] = q, -q
	}
	for j, v := range y {
		if !positiveYieldInputs(v * float64(1-2*(j%2))) {
			return y, fmt.Errorf("invalid %s=%g; supply both shear directions for a new section", sectionYieldNames[j], v)
		}
	}
	rounded, err := sectionYieldINPLimits(y)
	if err != nil {
		return y, err
	}
	for i := range r.explicit {
		if r.explicit[i] == nil && !r.resultAxes[i] {
			copy(rounded[2+2*i:4+2*i], s.Yield[2+2*i:4+2*i])
		}
	}
	// If a retained shear pair is asymmetric, its normalized value at Q=0
	// reduces the radius available for N and M in arclm's nested norm.
	var shearSum float64
	for _, j := range []int{2, 4} {
		shearSum += math.Pow(math.Abs((rounded[j]+rounded[j+1])/(rounded[j]-rounded[j+1])), s.Exp)
	}
	shearNorm := math.Pow(shearSum, 1/s.Exp)
	radius := arclm.RADIUS * math.Pow(1-math.Pow(shearNorm/arclm.RADIUS, s.Exq), 1/s.Exq)
	axialAtZero := math.Abs((rounded[0] + rounded[1]) / (rounded[0] - rounded[1]))
	if !positiveYieldInputs(radius) || radius <= axialAtZero {
		return y, fmt.Errorf("無応力状態が近似降伏曲面の内部にありません")
	}
	for i, cv := range curves {
		gap := cv.excess(rounded[0], rounded[1], rounded[8+2*i], s.Exp, radius)
		fmt.Fprintf(out, "  M%s 基準点での最大超過=%.6f tf·m (N=%.6f tf)\n", rcAxis(i), gap.m, gap.n)
	}
	return rounded, nil
}
