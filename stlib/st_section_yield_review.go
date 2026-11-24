package st

import (
	"crypto/sha256"
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
)

// A review belongs to the live frame. A snapshot, reopened file or undo starts
// without one. Independent MCP proposal frames can reproduce the same review ID.
type sectionYieldRequest interface {
	preview(*Frame) (string, map[int][12]float64, bool)
	sectionIDs() []int
}

type sectionYieldReview struct {
	request sectionYieldRequest
	id      string
}

func sectionYieldTargets(name string, stw ExModer, input []interface{}, args map[string]string) ([]int, error) {
	f := stw.Frame()
	sect, bySect := args["SECT"]
	elem, byElem := args["ELEM"]
	if bySect && byElem {
		return nil, fmt.Errorf("%s: choose either -sect or -elem", name)
	}
	ids := make(map[int]bool)
	switch {
	case bySect:
		for _, raw := range strings.Split(sect, ",") {
			n, err := strconv.Atoi(raw)
			if err != nil || f.Sects[n] == nil {
				return nil, fmt.Errorf("%s: unknown SECT %q", name, raw)
			}
			ids[n] = true
		}
	case byElem:
		n, err := strconv.Atoi(elem)
		el := f.Elems[n]
		if err != nil || el == nil || el.Sect == nil {
			return nil, fmt.Errorf("%s: unknown ELEM %q", name, elem)
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
		return nil, fmt.Errorf("%s: select members or specify -sect=N or -elem=N", name)
	}
	return slices.Sorted(maps.Keys(ids)), nil
}

var sectionYieldNames = [...]string{"NZMAX", "NZMIN", "QXMAX", "QXMIN", "QYMAX", "QYMIN", "MZMAX", "MZMIN", "MXMAX", "MXMIN", "MYMAX", "MYMIN"}

// Store precisely the decimal values that Sect.InpString writes, so an analysis
// run in the GUI and one run after reopening the INP use identical limits.
func sectionYieldINPLimits(y [12]float64) ([12]float64, error) {
	for i, v := range y {
		x, err := strconv.ParseFloat(fmt.Sprintf("%.3f", v), 64)
		if err != nil || math.IsNaN(x) || math.IsInf(x, 0) || x == 0 {
			return y, fmt.Errorf("%s (%g) is not a usable capacity at INP precision", sectionYieldNames[i], v)
		}
		y[i] = x
	}
	return y, nil
}

func runSectionYieldCommand(stw ExModer, name string, args map[string]string, review **sectionYieldReview, parse func() (sectionYieldRequest, error)) error {
	f := stw.Frame()
	token, apply := args["APPLY"]
	last := *review
	if !apply {
		*review = nil
	}
	var r sectionYieldRequest
	if apply && len(args) == 1 {
		if last == nil {
			return fmt.Errorf("%s: preview the target sections before applying", name)
		}
		r = last.request
	} else {
		var err error
		r, err = parse()
		if err != nil {
			return err
		}
	}
	report, changes, ready := r.preview(f)
	if !ready {
		*review = nil
		if apply {
			return fmt.Errorf("%s: cannot apply\n%s", name, report)
		}
		return Message(report + "未反映: 要確認の項目があります。")
	}
	digest := sha256.Sum256([]byte(report))
	id := fmt.Sprintf("%x", digest[:12])
	if !apply {
		*review = &sectionYieldReview{request: r, id: id}
		// Give detached MCP calls a replayable command. Bind the targets to
		// explicit SECT numbers even when the preview came from a selection.
		var sections []string
		for _, n := range r.sectionIDs() {
			sections = append(sections, strconv.Itoa(n))
		}
		command := []string{":" + name, "-sect=" + strings.Join(sections, ",")}
		for _, k := range slices.Sorted(maps.Keys(args)) {
			if k == "ELEM" && name == "rcyield" {
				command = append(command, "-member="+args[k])
			}
			if k != "ELEM" && k != "SECT" {
				command = append(command, "-"+strings.ToLower(k)+"="+args[k])
			}
		}
		command = append(command, "-apply="+id)
		return Message(report + "preview_id=" + id + "\n確認後に実行  " + strings.Join(command, " ") + "\n同じGUIでは :" + name + " -apply でも反映できます。")
	}
	if token == "" && last != nil {
		token = last.id
	}
	if token != id {
		return fmt.Errorf("%s: preview does not match the current sections or options; preview again", name)
	}
	if len(changes) == 0 {
		*review = nil
		return Message(name + ": 変更なし")
	}
	for _, af := range f.Arclms {
		if af != nil && af.Running() {
			return fmt.Errorf("%s: wait for the running analysis before applying", name)
		}
	}
	if baseline, ok := stw.(interface{ CaptureUndoBaseline(*Frame) }); ok && stw.UndoEnabled() {
		baseline.CaptureUndoBaseline(f)
	}
	for n, y := range changes {
		copy(f.Sects[n].Yield, y[:])
	}
	clear(f.Arclms)
	*review = nil
	Snapshot(stw)
	return Message(fmt.Sprintf("%s: SECT %v に降伏耐力を反映しました。", name, slices.Sorted(maps.Keys(changes))))
}
