package st

import (
	"cmp"
	"crypto/sha256"
	"fmt"
	"maps"
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

// A review belongs to the live frame. A snapshot, reopened file or undo starts
// without one. Independent MCP proposal frames can reproduce the same review ID.
type steelYieldReview struct {
	request steelYieldRequest
	id      string
}

func steelYieldTargets(stw ExModer, input []interface{}, args map[string]string) ([]int, error) {
	f := stw.Frame()
	sect, bySect := args["SECT"]
	elem, byElem := args["ELEM"]
	if bySect && byElem {
		return nil, fmt.Errorf("steelyield: choose either -sect or -elem")
	}
	ids := make(map[int]bool)
	switch {
	case bySect:
		for _, raw := range strings.Split(sect, ",") {
			n, err := strconv.Atoi(raw)
			if err != nil || f.Sects[n] == nil {
				return nil, fmt.Errorf("steelyield: unknown SECT %q", raw)
			}
			ids[n] = true
		}
	case byElem:
		n, err := strconv.Atoi(elem)
		el := f.Elems[n]
		if err != nil || el == nil || el.Sect == nil {
			return nil, fmt.Errorf("steelyield: unknown ELEM %q", elem)
		}
		ids[el.Sect.Num] = true
	default:
		for _, el := range currentelem(stw, input) {
			if el != nil && el.Sect != nil && f.Elems[el.Num] == el {
				ids[el.Sect.Num] = true
			}
		}
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("steelyield: select members or specify -sect=N or -elem=N")
	}
	return slices.Sorted(maps.Keys(ids)), nil
}

func parseSteelYieldRequest(stw ExModer, input []interface{}, args map[string]string) (steelYieldRequest, error) {
	r := steelYieldRequest{ratio: -1, k: 5.4, factor: 1}
	var err error
	r.sections, err = steelYieldTargets(stw, input, args)
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

var steelYieldNames = [...]string{"NZMAX", "NZMIN", "QXMAX", "QXMIN", "QYMAX", "QYMIN", "MZMAX", "MZMIN", "MXMAX", "MXMIN", "MYMAX", "MYMIN"}

// Store precisely the decimal values that Sect.InpString writes, so an analysis
// run in the GUI and one run after reopening the INP use identical limits.
func steelYieldINPLimits(y [12]float64) ([12]float64, error) {
	for i, v := range y {
		x, err := strconv.ParseFloat(fmt.Sprintf("%.3f", v), 64)
		if err != nil || math.IsNaN(x) || math.IsInf(x, 0) || x == 0 {
			return y, fmt.Errorf("%s (%g) is not a usable capacity at INP precision", steelYieldNames[i], v)
		}
		y[i] = x
	}
	return y, nil
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
				y, err = steelYieldINPLimits(y)
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
				fmt.Fprintf(&b, "    %s %g → %.3f", steelYieldNames[j], sec.Yield[j], v)
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
	token, apply := args["APPLY"]
	last := f.steelYieldReview
	if !apply {
		f.steelYieldReview = nil
	}
	var r steelYieldRequest
	if apply && len(args) == 1 {
		if last == nil {
			return fmt.Errorf("steelyield: preview the target sections before applying")
		}
		r = last.request
	} else {
		var err error
		r, err = parseSteelYieldRequest(stw, input, args)
		if err != nil {
			return err
		}
	}
	report, changes, ready := r.preview(f)
	if !ready {
		f.steelYieldReview = nil
		if apply {
			return fmt.Errorf("steelyield: cannot apply\n%s", report)
		}
		return Message(report + "未反映: 要確認の項目があります。")
	}
	digest := sha256.Sum256([]byte(report))
	id := fmt.Sprintf("%x", digest[:12])
	if !apply {
		f.steelYieldReview = &steelYieldReview{request: r, id: id}
		// Give detached MCP calls a replayable command. Bind the targets to
		// explicit SECT numbers even when the preview came from a selection.
		var sections []string
		for _, n := range r.sections {
			sections = append(sections, strconv.Itoa(n))
		}
		command := []string{":steelyield", "-sect=" + strings.Join(sections, ",")}
		for _, k := range slices.Sorted(maps.Keys(args)) {
			if k != "ELEM" && k != "SECT" {
				command = append(command, "-"+strings.ToLower(k)+"="+args[k])
			}
		}
		command = append(command, "-apply="+id)
		return Message(report + "preview_id=" + id + "\n確認後に実行  " + strings.Join(command, " ") + "\n同じGUIでは :steelyield -apply でも反映できます。")
	}
	if token == "" && last != nil {
		token = last.id
	}
	if token != id {
		return fmt.Errorf("steelyield: preview does not match the current sections or options; preview again")
	}
	if len(changes) == 0 {
		f.steelYieldReview = nil
		return Message("steelyield: 変更なし")
	}
	for _, af := range f.Arclms {
		if af != nil && af.Running() {
			return fmt.Errorf("steelyield: wait for the running analysis before applying")
		}
	}
	if baseline, ok := stw.(interface{ CaptureUndoBaseline(*Frame) }); ok && stw.UndoEnabled() {
		baseline.CaptureUndoBaseline(f)
	}
	for n, y := range changes {
		copy(f.Sects[n].Yield, y[:])
	}
	clear(f.Arclms)
	f.steelYieldReview = nil
	Snapshot(stw)
	return Message(fmt.Sprintf("steelyield: SECT %v に降伏耐力を反映しました。", slices.Sorted(maps.Keys(changes))))
}
