package main

import (
	"bytes"
	"maps"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	st "github.com/yofu/st/stlib"
	"golang.org/x/text/encoding/japanese"
)

type inpRecord struct{ key, data string }

func editorINP(frame *st.Frame) []byte {
	// Node and element output from the GUI is intended for display-oriented
	// precision. Replace numeric fields with round-trip representations.
	data := frame.EditableInpBytes()
	records := editorRecords(data)
	for i, record := range records {
		kind, idText, _ := strings.Cut(record.key, " ")
		id, _ := strconv.Atoi(idText)
		switch kind {
		case "HEADER":
			ai, wind := frame.Ai, frame.Wind
			records[i].data = exactFields(record.data, map[string][]float64{
				"BASE": ai.Base, "LOCATE": {ai.Locate}, "TFACT": {ai.Tfact}, "GPERIOD": {ai.Gperiod},
				"HEIGHT": ai.Boundary, "NOKIHEIGHT": {ai.H0}, "BETAX": ai.BetaX, "BETAY": ai.BetaY,
				"VELOCITY": {wind.Velocity}, "WFACT": {wind.Factor},
			})
		case "PROP":
			p := frame.Props[id]
			records[i].data = exactFields(record.data, map[string][]float64{"HFACT": {p.HFactor}, "EFACT": {p.EFactor}, "HIJU": {p.Hiju()}, "E": {p.EL()}, "ES": {p.ES()}, "POI": {p.Poi()}})
		case "BOND":
			b := frame.Bonds[id]
			if b == nil && id == st.Pin.Num {
				b = st.Pin
			}
			if b != nil {
				records[i].data = exactFields(record.data, map[string][]float64{"KR": b.Stiffness})
			}
		case "SECT":
			records[i].data = exactSection(record.data, frame.Sects[id])
		case "NODE":
			n := frame.Nodes[id]
			records[i].data = exactFields(record.data, map[string][]float64{"CORD": n.Coord, "VCON": n.Load, "PHASE": n.Phase})
		case "ELEM":
			e := frame.Elems[id]
			records[i].data = exactFields(record.data, map[string][]float64{"CANG": {e.Cang}, "CMQ": e.Cmq, "PREST": {e.Prestress}, "WRECT": e.Wrect})
		}
	}
	return joinRecords(records)
}

func exactSection(data string, sec *st.Sect) string {
	values := map[string][]float64{"LLOAD": sec.Lload, "PERPL": sec.Perpl, "EXP": {sec.Exp}, "EXQ": {sec.Exq}}
	for i, name := range []string{"NZMAX", "NZMIN", "QXMAX", "QXMIN", "QYMAX", "QYMIN", "MZMAX", "MZMIN", "MXMAX", "MXMIN", "MYMAX", "MYMIN"} {
		values[name] = []float64{sec.Yield[i]}
	}
	lines := strings.SplitAfter(data, "\n")
	fig := -1
	for i, line := range lines {
		words := strings.Fields(line)
		if len(words) == 0 {
			continue
		}
		if words[0] == "FIG" {
			fig++
		}
		if fig >= 0 && fig < len(sec.Figs) {
			v := sec.Figs[fig].Value
			switch words[0] {
			case "AREA", "IXX", "IYY", "VEN", "THICK", "KFACT", "SREIN":
				lines[i] = exactFields(line, map[string][]float64{words[0]: {v[words[0]]}})
				continue
			case "XFACE", "YFACE":
				name := words[0]
				lines[i] = exactFields(line, map[string][]float64{name: {v[name], v[name+"_H"]}})
				continue
			}
		}
		lines[i] = exactFields(line, values)
	}
	return strings.Join(lines, "")
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

func editorRecords(data []byte) []inpRecord {
	text := string(data)
	var records []inpRecord
	var key, chain string
	var index int
	hasRecord := false
	special := make(map[string]int)
	for offset := 0; offset < len(text); {
		end := strings.IndexByte(text[offset:], '\n')
		if end < 0 {
			end = len(text) - offset
		} else {
			end++
		}
		next := offset + end
		line := text[offset:next]
		words := strings.Fields(line)
		newKey := ""
		if len(words) > 0 {
			switch words[0] {
			case "NODE", "ELEM", "SECT", "PROP", "PILE", "BOND", "NODESET", "ELEMSET":
				if len(words) > 1 {
					newKey = words[0] + " " + words[1]
					if chain == "?" && words[0] == "ELEM" {
						chain = words[1]
						key = "CHAIN " + chain
					}
				}
			case "CHAIN":
				chain = "?"
				newKey = "CHAIN ?"
			case "}":
				newKey = "ENDCHAIN " + chain
				chain = ""
			case "#STMODEL":
				if len(words) >= 3 {
					prefix := words[0] + " " + words[1]
					special[prefix]++
					newKey = prefix + " " + strconv.Itoa(special[prefix])
				}
			default:
				if !hasRecord && !strings.HasPrefix(words[0], "#") {
					newKey = "HEADER " + words[0]
					if strings.HasPrefix(words[0], "\"") {
						newKey = "HEADER TITLE"
					}
				}
			}
		}
		if newKey != "" {
			if offset > index {
				records = append(records, inpRecord{key, text[index:offset]})
			}
			index, key = offset, newKey
			hasRecord = hasRecord || !strings.HasPrefix(newKey, "HEADER ")
		}
		offset = next
	}
	if index < len(text) {
		records = append(records, inpRecord{key, text[index:]})
	}
	return records
}

func joinRecords(records []inpRecord) []byte {
	var b bytes.Buffer
	for _, r := range records {
		b.WriteString(r.data)
		if r.data != "" && !strings.HasSuffix(r.data, "\n") {
			b.WriteByte('\n')
		}
	}
	return b.Bytes()
}

func recordMap(records []inpRecord) map[string]string {
	out := make(map[string]string, len(records))
	for _, r := range records {
		out[r.key] = r.data
	}
	return out
}

func recordValue(data string) string {
	var lines []string
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && (!strings.HasPrefix(line, "#") || strings.HasPrefix(line, "#STMODEL ")) {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

func mergeEditorINP(source, before, after []byte) []byte {
	original := editorRecords(source)
	originalByKey := recordMap(original)
	previous := recordMap(editorRecords(before))
	updated := editorRecords(after)
	afterByKey := recordMap(updated)
	out := make([]inpRecord, 0, len(updated)+1)
	lineEnd := "\n"
	if bytes.Contains(source, []byte("\r\n")) {
		lineEnd = "\r\n"
	}
	// Preserve the source preamble, and source header records the ST writer
	// does not produce. Header records unaffected by the edit retain precision.
	for _, r := range original {
		if r.key == "" || strings.HasPrefix(r.key, "HEADER ") && previous[r.key] == "" && afterByKey[r.key] == "" {
			out = append(out, r)
		}
	}
	for _, r := range updated {
		if raw, ok := originalByKey[r.key]; ok && recordValue(previous[r.key]) == recordValue(r.data) {
			out = append(out, inpRecord{r.key, raw})
			continue
		}
		out = append(out, r)
		// Retain comments attached to a modified record.
		if !strings.HasPrefix(r.key, "#STMODEL ") {
			for _, line := range strings.Split(originalByKey[r.key], "\n") {
				if strings.HasPrefix(strings.TrimSpace(line), "#") {
					out = append(out, inpRecord{data: strings.TrimRight(line, "\r") + lineEnd})
				}
			}
		}
	}
	// Preserve comments from deleted model records.
	for _, r := range original {
		if r.key == "" || strings.HasPrefix(r.key, "HEADER ") || strings.HasPrefix(r.key, "#STMODEL ") || afterByKey[r.key] != "" {
			continue
		}
		for _, line := range strings.Split(r.data, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "#") {
				out = append(out, inpRecord{data: strings.TrimRight(line, "\r") + lineEnd})
			}
		}
	}
	return joinRecords(out)
}

func diffEditorINP(before, after []byte) []change {
	a, b := recordMap(editorRecords(before)), recordMap(editorRecords(after))
	keys := make(map[string]bool, len(a)+len(b))
	for k := range a {
		keys[k] = true
	}
	for k := range b {
		keys[k] = true
	}
	result := make([]change, 0)
	view := func(key, value string) any {
		value = recordValue(value)
		if value == "" {
			return nil
		}
		if !utf8.ValidString(value) {
			if decoded, err := japanese.ShiftJIS.NewDecoder().String(value); err == nil {
				value = decoded
			}
		}
		return map[string]string{"record": key, "inp": value}
	}
	for _, key := range slices.Sorted(maps.Keys(keys)) {
		if recordValue(a[key]) != recordValue(b[key]) {
			result = append(result, change{view(key, a[key]), view(key, b[key])})
		}
	}
	return result
}

func sameModelTopology(a, b *st.Frame) bool {
	if len(a.Nodes) != len(b.Nodes) || len(a.Elems) != len(b.Elems) || len(a.Sects) != len(b.Sects) {
		return false
	}
	for id, n := range a.Nodes {
		if m := b.Nodes[id]; m == nil || !slices.Equal(n.Coord, m.Coord) || !slices.Equal(n.Conf, m.Conf) || !slices.Equal(n.Load, m.Load) {
			return false
		}
	}
	for id, e := range a.Elems {
		f := b.Elems[id]
		if f == nil || e.Etype != f.Etype || e.Sect.Num != f.Sect.Num || len(e.Enod) != len(f.Enod) {
			return false
		}
		for i, n := range e.Enod {
			if n.Num != f.Enod[i].Num || a.Nodes[n.Num] != n {
				return false
			}
		}
	}
	return true
}
