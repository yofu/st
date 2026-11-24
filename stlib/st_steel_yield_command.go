package st

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
)

type steelYieldRequest struct {
	sections         []int
	lengths          SteelLengthOptions
	product          *ColdFormedTubeProduct
	ratio, k, factor float64
}

func (r steelYieldRequest) sectionIDs() []int { return r.sections }

func parseSteelYieldRequest(stw ExModer, input []interface{}, args map[string]string) (steelYieldRequest, error) {
	r := steelYieldRequest{ratio: -1, k: 5.4, factor: 1}
	var err error
	r.sections, err = sectionYieldTargets("steelyield", stw, input, args)
	if err != nil {
		return r, err
	}
	r.lengths, err = parseSteelLengthOptions(args)
	if err != nil {
		return r, err
	}
	for key, raw := range args {
		switch key {
		case "SECT", "ELEM", "DESIGN", "LK", "LB", "LKX", "LKY", "LBX", "LBY", "APPLY":
		case "RATIO", "K", "FACTOR":
			v, err := strconv.ParseFloat(raw, 64)
			if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || (key == "RATIO" && math.Abs(v) > 1) || (key != "RATIO" && v <= 0) {
				return r, fmt.Errorf("steelyield: invalid %s %q", key, raw)
			}
			switch key {
			case "RATIO":
				r.ratio = v
			case "K":
				r.k = v
			case "FACTOR":
				r.factor = v
			}
		case "PRODUCT":
			fields := strings.Split(raw, ",")
			if len(fields) != 9 {
				return r, fmt.Errorf("steelyield: -product requires H,B,tw,tf,A,Ix,Iy,Zpx,Zpy (cm, cm², cm⁴, cm³)")
			}
			var p [9]float64
			for i, field := range fields {
				p[i], err = strconv.ParseFloat(field, 64)
				if err != nil || !positiveYieldInputs(p[i]) {
					return r, fmt.Errorf("steelyield: invalid product property %q", field)
				}
			}
			r.product = &ColdFormedTubeProduct{Name: raw, Shape: RPIPE{H: p[0], B: p[1], Tw: p[2], Tf: p[3]}, Area: p[4], Ix: p[5], Iy: p[6], Zpx: p[7], Zpy: p[8]}
		default:
			return r, fmt.Errorf("steelyield: unknown option %q", key)
		}
	}
	return r, nil
}

func (r steelYieldRequest) preview(f *Frame) (string, map[int][12]float64, bool) {
	resolver := NewSteelLengthResolver(f)
	members := make(map[int][]*Elem)
	for _, id := range r.sections {
		members[id] = nil
	}
	for _, el := range f.Elems {
		if el.Sect == nil {
			continue
		}
		if _, selected := members[el.Sect.Num]; selected {
			members[el.Sect.Num] = append(members[el.Sect.Num], el)
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "降伏耐力プレビュー  ratio=%g  k=%g  factor=%g  (NZ/Q: tf, M: tf·m)\n", r.ratio, r.k, r.factor)
	if r.product != nil {
		fmt.Fprintf(&b, "製品断面 H,B,tw,tf,A,Ix,Iy,Zpx,Zpy (cm系): %s\n", r.product.Name)
	}
	changes := make(map[int][12]float64)
	ready := true
	for _, id := range r.sections {
		sec := f.Sects[id]
		group := members[id]
		fmt.Fprintf(&b, "SECT %d  使用部材 %d 本\n", id, len(group))
		if sec == nil || len(sec.Yield) != 12 || !sec.HasArea(0) || len(group) == 0 {
			fmt.Fprintln(&b, "  要確認: 断面積、降伏耐力12値、使用部材を確認してください。")
			ready = false
			continue
		}
		slices.SortFunc(group, func(a, b *Elem) int { return cmp.Compare(a.Num, b.Num) })
		var candidates [][12]float64
		candidateIndex := make(map[[12]float64]int)
		for _, el := range group {
			l, err := resolver.Resolve(el, r.lengths)
			if err != nil {
				fmt.Fprintf(&b, "  ELEM %d  要確認: %v\n", el.Num, err)
				ready = false
				continue
			}
			fmt.Fprint(&b, strings.ReplaceAll(l.String(), "\n  ", "  "))
			y, err := l.Yield(r.product, r.ratio, r.k, r.factor)
			if err == nil {
				y, err = sectionYieldINPLimits(y)
			}
			if err != nil {
				fmt.Fprintf(&b, "  要確認: %v\n", err)
				ready = false
				continue
			}
			i, seen := candidateIndex[y]
			if !seen {
				i = len(candidates)
				candidateIndex[y] = i
				candidates = append(candidates, y)
			}
			fmt.Fprintf(&b, "  → 案%d\n", i+1)
		}
		for i, candidate := range candidates {
			fmt.Fprintf(&b, "  SECT %d 案%d  変更前 → 変更後\n", id, i+1)
			for j, v := range candidate {
				fmt.Fprintf(&b, "    %s %g → %.3f", sectionYieldNames[j], sec.Yield[j], v)
				if j%2 == 1 {
					b.WriteByte('\n')
				}
			}
		}
		switch len(candidates) {
		case 1:
			if !slices.Equal(sec.Yield, candidates[0][:]) {
				changes[id] = candidates[0]
			}
		case 0:
			ready = false
		default:
			fmt.Fprintln(&b, "  要確認: 同じSECTに複数の耐力候補があります。補剛条件を確認するか、SECTを分けてください。")
			ready = false
		}
	}
	return b.String(), changes, ready
}

func steelYieldCommand(stw ExModer, input []interface{}, args map[string]string) error {
	f := stw.Frame()
	if f == nil {
		return fmt.Errorf("steelyield: open a model first")
	}
	return runSectionYieldCommand(stw, "steelyield", args, &f.steelYieldReview, func() (sectionYieldRequest, error) {
		return parseSteelYieldRequest(stw, input, args)
	})
}
