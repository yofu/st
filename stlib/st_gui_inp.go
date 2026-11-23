package st

import (
	"maps"
	"strconv"
	"strings"
)

// GUIInpBytes exports model inputs at round-trip precision for live editing.
func (frame *Frame) GUIInpBytes() []byte {
	inp, _, _ := frame.guiInpBytes()
	return inp
}

// Return the camera record byte range as well, so a snapshot and its
// camera-independent digest can share the expensive model serialization.
func (frame *Frame) guiInpBytes() ([]byte, int, int) {
	// Analysis braces are derived from their wall/slab parents. Exporting them
	// as additional input members would double their stiffness after reopening.
	input := *frame
	input.Elems = maps.Clone(frame.Elems)
	input.Sects = maps.Clone(frame.Sects)
	for id, e := range input.Elems {
		if e.Parent != nil && (e.Etype == WBRACE || e.Etype == SBRACE) {
			delete(input.Elems, id)
		}
	}
	referenced := make(map[int]bool)
	for _, e := range input.Elems {
		referenced[e.Sect.Num] = true
	}
	for id, s := range input.Sects {
		if !referenced[id] && s.Original != id && (s.Type == WBRACE || s.Type == SBRACE) {
			delete(input.Sects, id)
		}
	}
	lines := strings.SplitAfter(string(input.EditableInpBytes()), "\n")
	kind, id := "HEADER", 0
	fig := -1
	firstCamera, afterCamera := 0, 0
	for i, line := range lines {
		words := strings.Fields(line)
		if len(words) == 0 {
			continue
		}
		switch words[0] {
		case "NODE", "ELEM", "SECT", "PROP", "BOND", "PILE":
			kind = words[0]
			id, _ = strconv.Atoi(words[1])
			fig = -1
		case "#STMODEL", "CHAIN", "}", "NODESET", "ELEMSET":
			kind = ""
		}
		switch kind {
		case "HEADER":
			switch words[0] {
			case "GFACT":
				firstCamera = i
			case "DISTS":
				afterCamera = i + 1
			}
			ai, wind := frame.Ai, frame.Wind
			lines[i] = exactFields(line, map[string][]float64{
				"BASE": ai.Base, "LOCATE": {ai.Locate}, "TFACT": {ai.Tfact}, "GPERIOD": {ai.Gperiod},
				"HEIGHT": ai.Boundary, "NOKIHEIGHT": {ai.H0}, "BETAX": ai.BetaX, "BETAY": ai.BetaY,
				"VELOCITY": {wind.Velocity}, "WFACT": {wind.Factor},
			})
		case "PROP":
			p := frame.Props[id]
			lines[i] = exactFields(line, map[string][]float64{"HFACT": {p.HFactor}, "EFACT": {p.EFactor}, "HIJU": {p.Hiju()}, "E": {p.EL()}, "ES": {p.ES()}, "POI": {p.Poi()}})
		case "BOND":
			b := frame.Bonds[id]
			if b == nil && id == Pin.Num {
				b = Pin
			}
			if b != nil {
				lines[i] = exactFields(line, map[string][]float64{"KR": b.Stiffness})
			}
		case "SECT":
			sec := frame.Sects[id]
			if words[0] == "FIG" {
				fig++
			}
			values := map[string][]float64{"LLOAD": sec.Lload, "PERPL": sec.Perpl, "EXP": {sec.Exp}, "EXQ": {sec.Exq}}
			for j, name := range []string{"NZMAX", "NZMIN", "QXMAX", "QXMIN", "QYMAX", "QYMIN", "MZMAX", "MZMIN", "MXMAX", "MXMIN", "MYMAX", "MYMIN"} {
				values[name] = []float64{sec.Yield[j]}
			}
			if fig >= 0 && fig < len(sec.Figs) {
				v := sec.Figs[fig].Value
				switch words[0] {
				case "AREA", "IXX", "IYY", "VEN", "THICK", "KFACT", "SREIN":
					values[words[0]] = []float64{v[words[0]]}
				case "XFACE", "YFACE":
					values[words[0]] = []float64{v[words[0]], v[words[0]+"_H"]}
				}
			}
			lines[i] = exactFields(line, values)
		case "NODE":
			n := frame.Nodes[id]
			lines[i] = exactFields(line, map[string][]float64{"CORD": n.Coord, "VCON": n.Load, "PHASE": n.Phase, "MANUALWEIGHT": n.ManualWeight[:]})
		case "ELEM":
			e := frame.Elems[id]
			lines[i] = exactFields(line, map[string][]float64{"CANG": {e.Cang}, "CMQ": e.Cmq, "PREST": {e.Prestress}, "WRECT": e.Wrect})
		}
	}
	var start, end int
	for _, line := range lines[:firstCamera] {
		start += len(line)
	}
	end = start
	for _, line := range lines[firstCamera:afterCamera] {
		end += len(line)
	}
	return []byte(strings.Join(lines, "")), start, end
}

func exactFields(data string, values map[string][]float64) string {
	lines := strings.SplitAfter(data, "\n")
	for i, line := range lines {
		words := strings.Fields(line)
		changed := false
		for j, word := range words {
			if v := values[word]; len(v) > 0 && j+len(v) < len(words) {
				for k, f := range v {
					words[j+k+1] = strconv.FormatFloat(f, 'g', -1, 64)
				}
				changed = true
			}
		}
		if changed {
			lines[i] = strings.Join(words, " ") + "\r\n"
		}
	}
	return strings.Join(lines, "")
}
