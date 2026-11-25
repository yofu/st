package st

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const tstUsage = ":tst [elem...] [-sect=N[,N...]] [-full] [-open|-vim] [-out=path] [-max] [-all|-limit=20] [-v|-verbose] [-fbold] [-torsion] [-reload] [-qfact=2.0] [-wfact=2.0] [-bfact=1.0] [-skipshort] [-temporary] [-moeshiro] [-sekisetsu]"

const defaultTstLimit = 20

// applySrcalConditionOptions applies shared :srcal / :tst condition flags to a
// snapshot of base (or NewCondition() when base is nil) and returns a summary.
func applySrcalConditionOptions(base *Condition, argdict map[string]string, strictNumbers bool) (*Condition, string, error) {
	var cond *Condition
	if base != nil {
		cond = base.Snapshot()
	} else {
		cond = NewCondition()
	}
	var m bytes.Buffer
	if _, ok := argdict["FBOLD"]; ok {
		m.WriteString("Fb: old\n")
		cond.FbOld = true
	}
	if qf, ok := argdict["QFACT"]; ok {
		val, err := strconv.ParseFloat(qf, 64)
		if err != nil {
			if strictNumbers {
				return nil, "", fmt.Errorf("invalid -qfact=%q: %w", qf, err)
			}
		} else {
			cond.Qfact = val
		}
	}
	m.WriteString(fmt.Sprintf("QFACT: %.3f\n", cond.Qfact))
	if wf, ok := argdict["WFACT"]; ok {
		val, err := strconv.ParseFloat(wf, 64)
		if err != nil {
			if strictNumbers {
				return nil, "", fmt.Errorf("invalid -wfact=%q: %w", wf, err)
			}
		} else {
			cond.Wfact = val
		}
	}
	m.WriteString(fmt.Sprintf("WFACT: %.3f\n", cond.Wfact))
	if bf, ok := argdict["BFACT"]; ok {
		val, err := strconv.ParseFloat(bf, 64)
		if err != nil {
			if strictNumbers {
				return nil, "", fmt.Errorf("invalid -bfact=%q: %w", bf, err)
			}
		} else {
			cond.Bfact = val
		}
	}
	m.WriteString(fmt.Sprintf("BFACT: %.3f\n", cond.Bfact))
	if _, ok := argdict["SKIPSHORT"]; ok {
		m.WriteString("SKIP SHORT\n")
		cond.Skipshort = true
	}
	if _, ok := argdict["TEMPORARY"]; ok {
		m.WriteString("TEMPORARY")
		cond.Temporary = "S"
	}
	if _, ok := argdict["MOESHIRO"]; ok {
		m.WriteString("MOESHIRO")
		cond.Skipshort = true
		cond.Temporary = "S"
	}
	if _, ok := argdict["SEKISETSU"]; ok {
		m.WriteString("SEKISETSU")
		cond.Skipshort = true
		cond.Temporary = "ML"
	}
	_, hasV := argdict["V"]
	_, hasVerbose := argdict["VERBOSE"]
	if hasV || hasVerbose {
		m.WriteString("VERBOSE")
		cond.Verbose = true
	}
	if _, ok := argdict["TORSION"]; ok {
		m.WriteString("TORSION")
		cond.RCTorsion = true
	}
	return cond, m.String(), nil
}

func tstCommand(stw ExModer, args []string, argdict map[string]string, input []interface{}, pipe bool, openArtifact func(string, ...string) bool) ([]interface{}, error) {
	frame := stw.Frame()
	if frame == nil {
		return nil, fmt.Errorf(":tst: open a model first")
	}
	for key := range argdict {
		switch key {
		case "FULL", "OPEN", "VIM", "OUT", "MAX", "ALL", "LIMIT", "SECT",
			"V", "VERBOSE", "FBOLD", "TORSION", "RELOAD", "NORELOAD",
			"QFACT", "WFACT", "BFACT", "SKIPSHORT", "TEMPORARY", "MOESHIRO", "SEKISETSU":
		default:
			return nil, fmt.Errorf(":tst: unknown option -%s", strings.ToLower(key))
		}
	}
	if _, ok := argdict["RELOAD"]; ok {
		if frame.Path == "" {
			return nil, fmt.Errorf(":tst: -reload requires a saved model path")
		}
		if err := ReadFile(stw, Ce(frame.Path, ".lst")); err != nil {
			return nil, err
		}
	}

	limit := defaultTstLimit
	if _, ok := argdict["ALL"]; ok {
		limit = 0
	}
	if raw, ok := argdict["LIMIT"]; ok {
		val, err := strconv.Atoi(raw)
		if err != nil || val < 1 {
			return nil, fmt.Errorf(":tst: invalid -limit=%q (must be >= 1)", raw)
		}
		limit = val
	}

	targets, skippedPlates, err := resolveTstTargets(stw, frame, args, argdict, input)
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		if len(skippedPlates) > 0 {
			return nil, fmt.Errorf(":tst: selected plate element(s) have no analysis braces (run :extractarclm first): %s", formatElemIDs(skippedPlates))
		}
		return nil, fmt.Errorf(":tst: no elements selected")
	}

	reports := make([]*elemRateReport, 0, len(targets))
	var skipped []int
	for _, el := range targets {
		cond, _, err := applySrcalConditionOptions(el.Condition, argdict, true)
		if err != nil {
			return nil, fmt.Errorf(":tst: %w", err)
		}
		rep, err := el.evaluateRate(cond, "L", "X", "X", "Y", "Y", -1.0, nil, 0.0)
		if err != nil {
			skipped = append(skipped, el.Num)
			continue
		}
		reports = append(reports, rep)
	}
	if len(reports) == 0 {
		return nil, fmt.Errorf(":tst: no elements with section rate (skipped: %s)", formatElemIDs(skipped))
	}

	if _, ok := argdict["MAX"]; ok && len(reports) > 1 {
		best := reports[0]
		bestVal := reportMaxRate(best)
		for _, r := range reports[1:] {
			if v := reportMaxRate(r); v > bestVal {
				best = r
				bestVal = v
			}
		}
		reports = []*elemRateReport{best}
	}

	var sender []interface{}
	if pipe {
		sender = make([]interface{}, len(reports))
		for i, r := range reports {
			sender[i] = r.Elem
		}
	}

	_, wantFull := argdict["FULL"]
	_, wantOpen := argdict["OPEN"]
	_, wantVim := argdict["VIM"]
	outPath, hasOut := argdict["OUT"]
	wantFile := wantOpen || wantVim || hasOut

	if wantFile {
		path, err := writeTstExtractFile(stw, frame, reports, outPath, hasOut)
		if err != nil {
			return sender, err
		}
		if hasOut {
			reportCommandFile(stw, path)
		}
		if wantOpen || wantVim {
			if openArtifact == nil || !openArtifact("text", path) {
				Vim(path)
			}
		}
		best := reports[0]
		bestVal := reportMaxRate(best)
		for _, r := range reports[1:] {
			if v := reportMaxRate(r); v > bestVal {
				best = r
				bestVal = v
			}
		}
		msg := fmt.Sprintf(":tst: wrote %d element(s) to %s (max=%.3f at ELEM %d)", len(reports), path, bestVal, best.Elem.Num)
		if len(skipped) > 0 {
			msg += fmt.Sprintf(" [skipped %d without section rate]", len(skipped))
		}
		return sender, Message(msg)
	}

	total := len(reports)
	shown := reports
	if limit > 0 && len(shown) > limit {
		shown = shown[:limit]
	}

	var out bytes.Buffer
	if wantFull {
		for _, rep := range shown {
			out.WriteString(rep.FullText)
		}
	} else {
		for i, rep := range shown {
			if i > 0 {
				out.WriteString("\n")
			}
			out.WriteString(formatCompactRateReport(frame, rep))
		}
	}
	if len(shown) < total {
		out.WriteString(fmt.Sprintf("\n... (showing %d of %d elements; use -all, -max, or -open)\n", len(shown), total))
	}
	if len(skipped) > 0 {
		out.WriteString(fmt.Sprintf("\nSkipped %d element(s) without section rate: %s\n", len(skipped), formatElemIDs(skipped)))
	}
	if len(skippedPlates) > 0 {
		out.WriteString(fmt.Sprintf("Skipped %d unbraced plate element(s): %s\n", len(skippedPlates), formatElemIDs(skippedPlates)))
	}
	return sender, Message(strings.TrimRight(out.String(), "\n"))
}

func resolveTstTargets(stw ExModer, frame *Frame, args []string, argdict map[string]string, input []interface{}) ([]*Elem, []int, error) {
	var raw []*Elem
	if sectSpec, ok := argdict["SECT"]; ok {
		snums, err := parseTstIntList(sectSpec)
		if err != nil || len(snums) == 0 {
			return nil, nil, fmt.Errorf(":tst: invalid -sect=%q", sectSpec)
		}
		wantSect := make(map[int]bool, len(snums))
		for _, s := range snums {
			wantSect[s] = true
		}
		for _, el := range frame.Elems {
			if el == nil || el.Sect == nil {
				continue
			}
			if wantSect[el.Sect.Num] || wantSect[el.OriginalSection().Num] {
				raw = append(raw, el)
			}
		}
		if len(raw) == 0 {
			return nil, nil, fmt.Errorf(":tst: no elements match -sect=%s", sectSpec)
		}
	} else if len(args) >= 2 {
		joined := strings.Join(args[1:], ",")
		enums, err := parseTstIntList(joined)
		if err != nil || len(enums) == 0 {
			return nil, nil, fmt.Errorf(":tst: invalid element number(s): %s", strings.Join(args[1:], " "))
		}
		for _, id := range enums {
			el, ok := frame.Elems[id]
			if !ok || el == nil {
				return nil, nil, fmt.Errorf(":tst: ELEM %d not found", id)
			}
			raw = append(raw, el)
		}
	} else {
		var piped []*Elem
		for _, v := range input {
			if el, ok := v.(*Elem); ok && el != nil {
				piped = append(piped, el)
			}
		}
		if len(piped) > 0 {
			raw = piped
		} else if stw.ElemSelected() {
			raw = stw.SelectedElems()
		}
	}

	seen := make(map[int]bool, len(raw))
	var lineElems []*Elem
	var skippedPlates []int
	for _, el := range raw {
		if el == nil {
			continue
		}
		if el.IsLineElem() {
			if !seen[el.Num] {
				seen[el.Num] = true
				lineElems = append(lineElems, el)
			}
			continue
		}
		if el.Etype == WALL || el.Etype == SLAB {
			added := false
			for _, child := range el.Children {
				if child != nil && child.IsLineElem() {
					if !seen[child.Num] {
						seen[child.Num] = true
						lineElems = append(lineElems, child)
					}
					added = true
				}
			}
			if !added {
				skippedPlates = append(skippedPlates, el.Num)
			}
		}
	}
	sort.Sort(ElemByNum{lineElems})
	sort.Ints(skippedPlates)
	return lineElems, skippedPlates, nil
}

func parseTstIntList(spec string) ([]int, error) {
	fields := strings.FieldsFunc(spec, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t'
	})
	nums := make([]int, 0, len(fields))
	for _, f := range fields {
		v, err := strconv.Atoi(f)
		if err != nil {
			return nil, err
		}
		nums = append(nums, v)
	}
	return nums, nil
}

func formatElemIDs(ids []int) string {
	const maxList = 10
	parts := make([]string, 0, min(len(ids), maxList))
	for i, id := range ids {
		if i >= maxList {
			parts = append(parts, fmt.Sprintf("...(+%d)", len(ids)-maxList))
			break
		}
		parts = append(parts, strconv.Itoa(id))
	}
	return strings.Join(parts, ", ")
}

func reportMaxRate(rep *elemRateReport) float64 {
	best := 0.0
	for _, r := range rep.MaxRate {
		if r > best {
			best = r
		}
	}
	return best
}

func writeTstExtractFile(stw ExModer, frame *Frame, reports []*elemRateReport, outPath string, hasOut bool) (string, error) {
	var target string
	if hasOut && outPath != "" {
		target = outPath
		if !filepath.IsAbs(target) {
			cwd := stw.Cwd()
			if cwd != "" {
				target = filepath.Join(cwd, target)
			}
		}
	} else {
		base := "selected"
		dir := os.TempDir()
		if frame.Path != "" {
			stem := strings.TrimSuffix(filepath.Base(frame.Path), filepath.Ext(frame.Path))
			if stem != "" {
				base = stem + "_selected"
			}
			if d := filepath.Dir(frame.Path); d != "" && d != "." {
				if info, err := os.Stat(d); err == nil && info.IsDir() {
					dir = d
				}
			}
		}
		target = filepath.Join(dir, base+".tst")
	}

	var buf bytes.Buffer
	buf.WriteString("断面算定 \"S,RC,SRC\" 抜粋 (selected elements)\n")
	if frame.Path != "" {
		buf.WriteString(fmt.Sprintf("モデル                   =%s\n", frame.Path))
	}
	if frame.LstFileName != "" {
		buf.WriteString(fmt.Sprintf("仮定断面                 =%s\n", frame.LstFileName))
	}
	buf.WriteString("単位系 tf(kN),tfm(kNm)\n\n")

	sectSet := make(map[int]bool)
	for _, rep := range reports {
		buf.WriteString(rep.FullText)
		if rep.Allow != nil {
			sectSet[rep.Allow.Num()] = true
		}
	}
	keys := make([]int, 0, len(sectSet))
	for k := range sectSet {
		if sec, ok := frame.Sects[k]; ok && sec != nil && sec.Allow != nil {
			keys = append(keys, k)
		}
	}
	sort.Ints(keys)
	if len(keys) > 0 {
		buf.WriteString("==========================================================================================================================================================================================================\n各断面種別の入力情報の確認\n\n")
		buf.WriteString("                              A[cm2]      Ix[cm4]      Iy[cm4]       J[cm4]\n")
		buf.WriteString("                              t[cm]\n")
		buf.WriteString(frame.CheckLst(keys))
	}

	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return "", err
	}
	data := AddCR(buf)
	if err := os.WriteFile(target, data.Bytes(), 0666); err != nil {
		return "", err
	}
	return target, nil
}

func displayForceUnits(frame *Frame) (fScale, mScale float64, fName, mName string) {
	if frame != nil && frame.Show != nil && len(frame.Show.Unit) >= 2 && len(frame.Show.UnitName) >= 2 && frame.Show.Unit[0] > 0 && frame.Show.Unit[1] > 0 {
		fScale = frame.Show.Unit[0]
		mScale = frame.Show.Unit[0] * frame.Show.Unit[1]
		fName = frame.Show.UnitName[0]
		mName = frame.Show.UnitName[0] + frame.Show.UnitName[1]
		return
	}
	return SI, SI, "kN", "kNm"
}

func formatCompactRateReport(frame *Frame, rep *elemRateReport) string {
	el := rep.Elem
	fScale, mScale, fName, mName := displayForceUnits(frame)
	typeStr := strings.TrimSpace(strings.ReplaceAll(rep.Allow.TypeString(), "　", " "))
	for strings.Contains(typeStr, "  ") {
		typeStr = strings.ReplaceAll(typeStr, "  ", " ")
	}
	status := "OK"
	if reportMaxRate(rep) >= 1.0 {
		status = "NG"
	}
	var b bytes.Buffer
	var sectLabel string
	origSect := el.OriginalSection()
	origName := ""
	if origSect != nil && origSect.Name != "" {
		origName = "(" + origSect.Name + ")"
	}
	if el.Etype == WBRACE || el.Etype == SBRACE {
		sectLabel = fmt.Sprintf("%d%s/%d=%s", origSect.Num, origName, el.Sect.Num, typeStr)
	} else {
		sectLabel = fmt.Sprintf("%d%s=%s", el.Sect.Num, origName, typeStr)
	}
	unitTag := fmt.Sprintf("[%s, %s]", fName, mName)
	if el.Etype == BRACE || el.Etype == WBRACE || el.Etype == SBRACE {
		unitTag = fmt.Sprintf("[%s]", fName)
	}
	b.WriteString(fmt.Sprintf("ELEM %d (N%d-N%d) SECT %s L=%.1fcm %s",
		el.Num, el.Enod[0].Num, el.Enod[1].Num, sectLabel, rep.Condition.Length, unitTag))
	if !rep.HasStress {
		b.WriteString(" [注意: 解析結果未読込]")
	}
	b.WriteString("\n")

	switch el.Etype {
	case COLUMN, GIRDER:
		if rep.Condition.Skipshort {
			b.WriteString(fmt.Sprintf("  MAX: Q/QaL=%.3f  M/MaL=%.3f  [%s]\n",
				rep.MaxRate[0], rep.MaxRate[3], status))
		} else {
			b.WriteString(fmt.Sprintf("  MAX: Q/QaL=%.3f  Q/QaS=%.3f(%s)  M/MaL=%.3f  M/MaS=%.3f(%s)  [%s]\n",
				rep.MaxRate[0], rep.MaxRate[1], rep.MaxQSCase, rep.MaxRate[3], rep.MaxRate[4], rep.MaxMSCase, status))
		}
		if len(rep.Cases) > 0 {
			b.WriteString(formatCompactBeamCase("長期", rep.Cases[0], rep.Condition.RCTorsion, fScale, mScale))
		}
		if !rep.Condition.Skipshort && len(rep.Cases) > 1 {
			dom := dominantBeamShortCase(rep.Cases[1:])
			b.WriteString(formatCompactBeamCase("支配", dom, rep.Condition.RCTorsion, fScale, mScale))
		}
		if rep.Condition.Verbose {
			if rep.SectionInfo != "" {
				b.WriteString(rep.SectionInfo)
			}
			for _, c := range selectVerboseCases(rep.Cases, el.Etype) {
				if c.Verbose != "" {
					b.WriteString(fmt.Sprintf("  [%s]\n", c.PeriodLabel))
					b.WriteString(c.Verbose)
				}
			}
		}
	case BRACE, WBRACE, SBRACE:
		if rep.Condition.Skipshort {
			b.WriteString(fmt.Sprintf("  MAX: Q/QaL=%.3f  [%s]\n", rep.MaxRate[0], status))
		} else {
			b.WriteString(fmt.Sprintf("  MAX: Q/QaL=%.3f  Q/QaS=%.3f(%s)  [%s]\n",
				rep.MaxRate[0], rep.MaxRate[1], rep.MaxQSCase, status))
		}
		if len(rep.Cases) > 0 {
			b.WriteString(formatCompactBraceCase("長期", rep.Cases[0], fScale))
		}
		if !rep.Condition.Skipshort && len(rep.Cases) > 1 {
			dom := dominantBraceShortCase(rep.Cases[1:])
			b.WriteString(formatCompactBraceCase("支配", dom, fScale))
		}
		if rep.Condition.Verbose {
			for _, c := range selectVerboseCases(rep.Cases, el.Etype) {
				if c.Verbose != "" {
					b.WriteString(fmt.Sprintf("  [%s]\n", c.PeriodLabel))
					b.WriteString(c.Verbose)
				}
			}
		}
	}
	return b.String()
}

func dominantBeamShortCase(cases []elemRateCase) elemRateCase {
	best := cases[0]
	bestVal := max(best.MRate, best.QRate)
	for _, c := range cases[1:] {
		if v := max(c.MRate, c.QRate); v > bestVal {
			best = c
			bestVal = v
		}
	}
	return best
}

func dominantBraceShortCase(cases []elemRateCase) elemRateCase {
	best := cases[0]
	for _, c := range cases[1:] {
		if c.QRate > best.QRate {
			best = c
		}
	}
	return best
}

func selectVerboseCases(cases []elemRateCase, etype int) []elemRateCase {
	if len(cases) == 0 {
		return nil
	}
	out := []elemRateCase{cases[0]}
	if len(cases) > 1 {
		var dom elemRateCase
		if etype == COLUMN || etype == GIRDER {
			dom = dominantBeamShortCase(cases[1:])
		} else {
			dom = dominantBraceShortCase(cases[1:])
		}
		out = append(out, dom)
	}
	return out
}

func formatCompactBeamCase(tag string, c elemRateCase, torsion bool, fScale, mScale float64) string {
	if c.Err != nil {
		return fmt.Sprintf("  %s(%-2s): [ERROR: %v]\n", tag, c.PeriodLabel, c.Err)
	}
	if len(c.Stress) < 12 || len(c.Allow) < 12 || len(c.Rate) < 12 {
		return fmt.Sprintf("  %s(%-2s): [no data]\n", tag, c.PeriodLabel)
	}
	qIdx := []int{1, 2, 7, 8}
	qNames := []string{"Qxi", "Qyi", "Qxj", "Qyj"}
	bestQ := 0
	for k, idx := range qIdx {
		if c.Rate[idx] > c.Rate[qIdx[bestQ]] {
			bestQ = k
		}
	}
	mIdx := []int{4, 5, 10, 11}
	mNames := []string{"Mxi", "Myi", "Mxj", "Myj"}
	bestM := 0
	for k, idx := range mIdx {
		if c.Rate[idx] > c.Rate[mIdx[bestM]] {
			bestM = k
		}
	}
	qi := qIdx[bestQ]
	mi := mIdx[bestM]
	var b bytes.Buffer
	b.WriteString(fmt.Sprintf("  %s(%-2s): N=%.2f/%.2f(%.3f)  %s=%.2f/%.2f(%.3f)  %s=%.2f/%.2f(%.3f)",
		tag, c.PeriodLabel,
		c.Stress[0]*fScale, c.Allow[0]*fScale, c.Rate[0],
		qNames[bestQ], c.Stress[qi]*fScale, c.Allow[qi]*fScale, c.Rate[qi],
		mNames[bestM], c.Stress[mi]*mScale, c.Allow[mi]*mScale, c.Rate[mi],
	))
	if torsion {
		b.WriteString(fmt.Sprintf("  Mt=%.2f/%.2f(%.3f)", c.Stress[3]*mScale, c.Allow[3]*mScale, c.Rate[3]))
	}
	b.WriteString(fmt.Sprintf(" -> M/Ma=%.3f\n", c.MRate))
	return b.String()
}

func formatCompactBraceCase(tag string, c elemRateCase, fScale float64) string {
	if c.Err != nil {
		return fmt.Sprintf("  %s(%-2s): [ERROR: %v]\n", tag, c.PeriodLabel, c.Err)
	}
	if len(c.Stress) < 1 || len(c.Allow) < 1 || len(c.Rate) < 1 {
		return fmt.Sprintf("  %s(%-2s): [no data]\n", tag, c.PeriodLabel)
	}
	return fmt.Sprintf("  %s(%-2s): N=%.2f/%.2f(%.3f)\n",
		tag, c.PeriodLabel, c.Stress[0]*fScale, c.Allow[0]*fScale, c.Rate[0])
}
