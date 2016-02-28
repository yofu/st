package st

import (
	"errors"
	"fmt"
	"github.com/yofu/abbrev"
	"github.com/yofu/complete"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	Fig2Abbrev = map[string]*complete.Complete{
		"gf/act":       complete.MustCompile("'gfact _", nil),
		"foc/us":       complete.MustCompile("'focus _ _ _", nil),
		"ang/le":       complete.MustCompile("'angle _ _", nil),
		"dist/s":       complete.MustCompile("'dists _ _", nil),
		"pers/pective": complete.MustCompile("'perspective", nil),
		"ax/onometric": complete.MustCompile("'axonometric", nil),
		"df/act":       complete.MustCompile("'dfact _", nil),
		"rf/act":       complete.MustCompile("'rfact _", nil),
		"qf/act":       complete.MustCompile("'qfact _", nil),
		"mf/act":       complete.MustCompile("'mfact _", nil),
		"gax/is":       complete.MustCompile("'gaxis _", nil),
		"eax/is":       complete.MustCompile("'eaxis _", nil),
		"noax/is":      complete.MustCompile("'noaxis", nil),
		"el/em": complete.MustCompile("'elem $ETYPE...",
			map[string][]string{
				"ETYPE": []string{"column", "girder", "brace", "wall", "wbrace", "slab", "sbrace"},
			}),
		"el/em/+/": complete.MustCompile("'elem+ $ETYPE...",
			map[string][]string{
				"ETYPE": []string{"column", "girder", "brace", "wall", "wbrace", "slab", "sbrace"},
			}),
		"el/em/-/": complete.MustCompile("'elem- $ETYPE...",
			map[string][]string{
				"ETYPE": []string{"column", "girder", "brace", "wall", "wbrace", "slab", "sbrace"},
			}),
		"sec/tion":      complete.MustCompile("'section _...", nil),
		"sec/tion/+/":   complete.MustCompile("'section+ _...", nil),
		"sec/tion/-/":   complete.MustCompile("'section- _...", nil),
		"k/ijun":        complete.MustCompile("'kijun", nil),
		"mea/sure":      complete.MustCompile("'measure", nil),
		"el/em/c/ode":   complete.MustCompile("'elemcode", nil),
		"sec/t/c/ode":   complete.MustCompile("'sectcode", nil),
		"wid/th":        complete.MustCompile("'width", nil),
		"h/eigh/t/":     complete.MustCompile("'height", nil),
		"sr/can/col/or": complete.MustCompile("'srcancolor", nil),
		"sr/can/ra/te":  complete.MustCompile("'srcanrate", nil),
		"en/ergy":       complete.MustCompile("'energy", nil),
		"st/ress": complete.MustCompile("'stress $ETYPE $PERIOD $NAME",
			map[string][]string{
				"ETYPE":  []string{"column", "girder", "brace", "wall", "wbrace", "slab", "sbrace"},
				"PERIOD": []string{"l", "x", "y"},
				"NAME":   []string{"n", "qx", "qy", "mz", "mx", "my"},
			}),
		"prest/ress": complete.MustCompile("'prestress", nil),
		"stiff/":     complete.MustCompile("'stiff", nil),
		"def/ormation": complete.MustCompile("'deformation $PERIOD",
			map[string][]string{
				"PERIOD": []string{"l", "x", "y"},
			}),
		"dis/p": complete.MustCompile("'disp $PERIOD $DIRECTION",
			map[string][]string{
				"PERIOD":    []string{"l", "x", "y"},
				"DIRECTION": []string{"z", "x", "y"},
			}),
		"ecc/entric": complete.MustCompile("'eccentric", nil),
		"dr/aw": complete.MustCompile("'draw $ETYPE...",
			map[string][]string{
				"ETYPE": []string{"column", "girder", "brace", "wall", "wbrace", "slab", "sbrace"},
			}),
		"al/ias":           complete.MustCompile("'alias", nil),
		"anon/ymous":       complete.MustCompile("'anonymous", nil),
		"no/de/c/ode":      complete.MustCompile("'nodecode", nil),
		"wei/ght":          complete.MustCompile("'weight", nil),
		"con/f":            complete.MustCompile("'conf", nil),
		"pi/lecode":        complete.MustCompile("'pilecode", nil),
		"fen/ce":           complete.MustCompile("'fence", nil),
		"per/iod":          complete.MustCompile("'period", nil),
		"per/iod/++/":      complete.MustCompile("'period++", nil),
		"per/iod/--/":      complete.MustCompile("'period--", nil),
		"nocap/tion":       complete.MustCompile("'nocaption", nil),
		"noleg/end":        complete.MustCompile("'nolegend", nil),
		"nos/hear/v/alue":  complete.MustCompile("'noshearvalue", nil),
		"nom/oment/v/alue": complete.MustCompile("'nomomentvalue", nil),
		"s/hear/ar/row":    complete.MustCompile("'sheararrow", nil),
		"m/oment/fig/ure":  complete.MustCompile("'momentfigure", nil),
		"ncol/or":          complete.MustCompile("'ncolor", nil),
		"p/age/tit/le":     complete.MustCompile("'pagetitle", nil),
		"tit/le":           complete.MustCompile("'title", nil),
		"pos/ition":        complete.MustCompile("'position", nil),
	}
)

func Fig2Mode(stw Fig2Moder, command string) error {
	if len(command) == 1 {
		return NotEnoughArgs("fig2mode")
	}
	if command == "'." {
		return Fig2Mode(stw, stw.LastFig2Command())
	}
	stw.SetLastFig2Command(command)
	command = command[1:]
	var un bool
	if strings.HasPrefix(command, "!") {
		un = true
		command = command[1:]
	} else {
		un = false
	}
	tmpargs := strings.Split(command, " ")
	args := make([]string, len(tmpargs))
	narg := 0
	for i := 0; i < len(tmpargs); i++ {
		if tmpargs[i] != "" {
			args[narg] = tmpargs[i]
			narg++
		}
	}
	args = args[:narg]
	return Fig2Keyword(stw, args, un)
}

func Fig2KeywordComplete(command string) (string, bool, *complete.Complete) {
	usage := strings.HasSuffix(command, "?")
	cname := strings.TrimSuffix(command, "?")
	cname = strings.ToLower(strings.TrimPrefix(cname, "'"))
	var rtn string
	var c *complete.Complete
	for ab, cp := range Fig2Abbrev {
		pat := abbrev.MustCompile(ab)
		if pat.MatchString(cname) {
			rtn = pat.Longest()
			c = cp
			break
		}
	}
	if rtn == "" {
		rtn = cname
	}
	return rtn, usage, c
}

func Fig2Keyword(stw Fig2Moder, lis []string, un bool) error {
	if len(lis) < 1 {
		return NotEnoughArgs("Fig2Keyword")
	}
	frame := stw.Frame()
	EPS := stw.EPS()
	showhtml := func(fn string) {
		f := filepath.Join(tooldir, "fig2/keywords", fn)
		if FileExists(f) {
			cmd := exec.Command("cmd", "/C", "start", f)
			cmd.Start()
		}
	}
	key, usage, _ := Fig2KeywordComplete(strings.ToLower(lis[0]))
	switch key {
	default:
		if k, ok := frame.Kijuns[key]; ok {
			min := -EPS
			max := EPS
			if len(lis) >= 2 {
				tmp, err := strconv.ParseFloat(lis[1], 64)
				if err == nil {
					min = tmp
				}
			}
			if len(lis) >= 3 {
				tmp, err := strconv.ParseFloat(lis[2], 64)
				if err == nil {
					max = tmp
				}
			}
			d := k.Direction()
			if IsParallel(d, XAXIS, EPS) {
				AxisRange(stw, 1, k.Start[1]+min, k.Start[1]+max, false)
			} else if IsParallel(d, YAXIS, EPS) {
				AxisRange(stw, 0, k.Start[0]+min, k.Start[0]+max, false)
			} else {
				for _, n := range frame.Nodes {
					n.Hide()
					ok, err := k.Contains(n.Coord, ZAXIS, min, max)
					if err != nil {
						continue
					}
					if ok {
						n.Show()
					}
				}
			}
			return nil
		}
		ErrorMessage(stw, errors.New(fmt.Sprintf("no fig2 keyword: %s", key)), INFO)
		return nil
	case "gfact":
		val, err := strconv.ParseFloat(lis[1], 64)
		if err != nil {
			return err
		}
		frame.View.Gfact = val
		stw.SetLabel("GFACT", fmt.Sprintf("%f", frame.View.Gfact))
	case "focus":
		if len(lis) < 2 {
			return NotEnoughArgs("FOCUS")
		}
		switch strings.ToUpper(lis[1]) {
		case "CENTER", "CENTRE":
			frame.SetFocus(nil)
		case "NODE":
			val, err := strconv.ParseInt(lis[2], 10, 64)
			if err != nil {
				return err
			}
			if n, ok := frame.Nodes[int(val)]; ok {
				frame.SetFocus(n.Coord)
			}
			w, h := stw.GetCanvasSize()
			frame.View.Center[0] = float64(w) * 0.5
			frame.View.Center[1] = float64(h) * 0.5
		case "ELEM":
			val, err := strconv.ParseInt(lis[2], 10, 64)
			if err != nil {
				return err
			}
			if el, ok := frame.Elems[int(val)]; ok {
				frame.SetFocus(el.MidPoint())
			}
			w, h := stw.GetCanvasSize()
			frame.View.Center[0] = float64(w) * 0.5
			frame.View.Center[1] = float64(h) * 0.5
		default:
			if len(lis) < 4 {
				return NotEnoughArgs("FOCUS")
			}
			for i, str := range []string{"FOCUSX", "FOCUSY", "FOCUSZ"} {
				if lis[1+i] == "_" {
					continue
				}
				val, err := strconv.ParseFloat(lis[1+i], 64)
				if err != nil {
					return err
				}
				frame.View.Focus[i] = val
				stw.SetLabel(str, fmt.Sprintf("%f", frame.View.Focus[i]))
			}
		}
	case "fit":
		frame.SetFocus(nil)
		// stw.DrawFrameNode()
		stw.ShowCenter()
	case "angle":
		if len(lis) < 3 {
			return NotEnoughArgs("ANGLE")
		}
		for i, str := range []string{"PHI", "THETA"} {
			if lis[1+i] == "_" {
				continue
			}
			val, err := strconv.ParseFloat(lis[1+i], 64)
			if err != nil {
				return err
			}
			frame.View.Angle[i] = val
			stw.SetLabel(str, fmt.Sprintf("%f", frame.View.Angle[i]))
		}
	case "dists":
		if len(lis) < 3 {
			return NotEnoughArgs("DISTS")
		}
		for i, str := range []string{"DISTR", "DISTL"} {
			if lis[1+i] == "_" {
				continue
			}
			val, err := strconv.ParseFloat(lis[1+i], 64)
			if err != nil {
				return err
			}
			frame.View.Dists[i] = val
			stw.SetLabel(str, fmt.Sprintf("%f", frame.View.Dists[i]))
		}
	case "perspective":
		frame.View.Perspective = true
	case "axonometric":
		frame.View.Perspective = false
	case "unit":
		if usage {
			stw.History("'unit force,length")
			stw.History(fmt.Sprintf("CURRENT FORCE UNIT: %s %.3f", frame.Show.UnitName[0], frame.Show.Unit[0]))
			stw.History(fmt.Sprintf("CURRENT LENGTH UNIT: %s %.3f", frame.Show.UnitName[1], frame.Show.Unit[1]))
			showhtml("UNIT.html")
			return nil
		}
		if un {
			frame.Show.Unit = []float64{1.0, 1.0}
			frame.Show.UnitName = []string{"tf", "m"}
			return nil
		}
		if len(lis) < 2 {
			return NotEnoughArgs("UNIT")
		}
		ustr := strings.Split(strings.ToLower(lis[1]), ",")
		if len(ustr) < 2 {
			return errors.New("'unit: incorrect format")
		}
		switch ustr[0] {
		case "tf":
			frame.Show.Unit[0] = 1.0
			frame.Show.UnitName[0] = "tf"
		case "kgf":
			frame.Show.Unit[0] = 1000.0
			frame.Show.UnitName[0] = "kgf"
		case "kn":
			frame.Show.Unit[0] = SI
			frame.Show.UnitName[0] = "kN"
		}
		switch ustr[1] {
		case "m":
			frame.Show.Unit[1] = 1.0
			frame.Show.UnitName[1] = "m"
		case "cm":
			frame.Show.Unit[0] = 100.0
			frame.Show.UnitName[0] = "cm"
		case "mm":
			frame.Show.Unit[0] = 1000.0
			frame.Show.UnitName[0] = "mm"
		}
	case "dfact":
		if len(lis) < 2 {
			return NotEnoughArgs("DFACT")
		}
		val, err := strconv.ParseFloat(lis[1], 64)
		if err != nil {
			return err
		}
		frame.Show.Dfact = val
		stw.SetLabel("DFACT", fmt.Sprintf("%f", val))
	case "rfact":
		if len(lis) < 2 {
			return NotEnoughArgs("RFACT")
		}
		val, err := strconv.ParseFloat(lis[1], 64)
		if err != nil {
			return err
		}
		frame.Show.Rfact = val
	case "qfact":
		if len(lis) < 2 {
			return NotEnoughArgs("QFACT")
		}
		val, err := strconv.ParseFloat(lis[1], 64)
		if err != nil {
			return err
		}
		frame.Show.Qfact = val
		stw.SetLabel("QFACT", fmt.Sprintf("%f", val))
	case "mfact":
		if len(lis) < 2 {
			return NotEnoughArgs("MFACT")
		}
		val, err := strconv.ParseFloat(lis[1], 64)
		if err != nil {
			return err
		}
		frame.Show.Mfact = val
		stw.SetLabel("MFACT", fmt.Sprintf("%f", val))
	case "gaxis":
		if un {
			frame.Show.GlobalAxis = false
			stw.DisableLabel("GAXIS")
		} else {
			frame.Show.GlobalAxis = true
			stw.EnableLabel("GAXIS")
			if len(lis) >= 2 {
				val, err := strconv.ParseFloat(lis[1], 64)
				if err != nil {
					return err
				}
				frame.Show.GlobalAxisSize = val
				stw.SetLabel("GAXISSIZE", fmt.Sprintf("%f", val))
			}
		}
	case "eaxis":
		if un {
			frame.Show.ElementAxis = false
			stw.DisableLabel("EAXIS")
		} else {
			frame.Show.ElementAxis = true
			stw.EnableLabel("EAXIS")
			if len(lis) >= 2 {
				val, err := strconv.ParseFloat(lis[1], 64)
				if err != nil {
					return err
				}
				frame.Show.ElementAxisSize = val
				stw.SetLabel("EAXISSIZE", fmt.Sprintf("%f", val))
			}
		}
	case "noaxis":
		frame.Show.GlobalAxis = false
		stw.DisableLabel("GAXIS")
		frame.Show.ElementAxis = false
		stw.DisableLabel("EAXIS")
	case "elem":
		for i, _ := range ETYPES {
			HideEtype(stw, i)
		}
		for _, val := range lis[1:] {
			et := strings.ToUpper(val)
			for i, e := range ETYPES {
				if et == e {
					ShowEtype(stw, i)
				}
			}
		}
	case "elem+":
		for _, val := range lis[1:] {
			et := strings.ToUpper(val)
			for i, e := range ETYPES {
				if et == e {
					ShowEtype(stw, i)
				}
			}
		}
	case "elem-":
		for _, val := range lis[1:] {
			et := strings.ToUpper(val)
			for i, e := range ETYPES {
				if et == e {
					HideEtype(stw, i)
				}
			}
		}
	case "section":
		HideAllSection(stw)
		for _, tmp := range lis[1:] {
			val, err := strconv.ParseInt(tmp, 10, 64)
			if err != nil {
				continue
			}
			ShowSection(stw, int(val))
		}
	case "section+":
		for _, tmp := range lis[1:] {
			val, err := strconv.ParseInt(tmp, 10, 64)
			if err != nil {
				continue
			}
			ShowSection(stw, int(val))
		}
	case "section-":
		for _, tmp := range lis[1:] {
			val, err := strconv.ParseInt(tmp, 10, 64)
			if err != nil {
				continue
			}
			HideSection(stw, int(val))
		}
	case "kijun":
		if usage {
			stw.History("'kijun <name...>")
			return nil
		}
		if un {
			frame.Show.Kijun = false
			stw.DisableLabel("KIJUN")
		} else {
			frame.Show.Kijun = true
			stw.EnableLabel("KIJUN")
			if len(lis) > 1 {
				for _, w := range lis[1:] {
					pat, err := regexp.Compile(w)
					if err != nil {
						continue
					}
					for _, k := range frame.Kijuns {
						k.Hide()
						if pat.MatchString(k.Name) {
							k.Show()
						}
					}
				}
			}
		}
	case "measure":
		if usage {
			stw.History("'measure kijun x1 x2 offset dotsize rotate overwrite")
			stw.History("'measure nnum1 nnum2 direction offset dotsize rotate overwrite")
			showhtml("MEASURE.html")
			return nil
		}
		if un {
			frame.Show.Measure = false
		} else {
			frame.Show.Measure = true
			if len(lis) < 4 {
				return NotEnoughArgs("MEASURE")
			}
			if abbrev.For("k/ijun", strings.ToLower(lis[1])) { // measure kijun x1 x2 offset dotsize rotate overwrite
				if k1, ok := frame.Kijuns[strings.ToLower(lis[2])]; ok {
					if k2, ok := frame.Kijuns[strings.ToLower(lis[3])]; ok {
						m := frame.AddMeasure(k1.Start, k2.Start, k1.Direction())
						m.Text = fmt.Sprintf("%.0f", k1.Distance(k2)*1000)
						if len(lis) < 5 {
							return nil
						}
						val, err := strconv.ParseFloat(lis[4], 64)
						if err != nil {
							return err
						}
						m.Gap = 2 * val
						m.Extension = -val
						if len(lis) < 6 {
							return nil
						}
						val, err = strconv.ParseFloat(lis[5], 64)
						if err != nil {
							return err
						}
						m.ArrowSize = val
						if len(lis) < 7 {
							return nil
						}
						val, err = strconv.ParseFloat(lis[6], 64)
						if err != nil {
							return err
						}
						m.Rotate = val
						if len(lis) < 8 {
							return nil
						}
						m.Text = ToUtf8string(lis[7])
					} else {
						return errors.New(fmt.Sprintf("no kijun named %s", lis[3]))
					}
				} else {
					return errors.New(fmt.Sprintf("no kijun named %s", lis[2]))
				}
			} else { // measure nnum1 nnum2 direction offset dotsize rotate overwrite
				nnum, err := strconv.ParseInt(lis[1], 10, 64)
				if err != nil {
					return err
				}
				if n1, ok := frame.Nodes[int(nnum)]; ok {
					nnum, err := strconv.ParseInt(lis[2], 10, 64)
					if err != nil {
						return err
					}
					if n2, ok := frame.Nodes[int(nnum)]; ok {
						var u, v []float64
						switch strings.ToUpper(lis[3]) {
						case "X":
							u = XAXIS
							v = YAXIS
						case "Y":
							u = YAXIS
							v = XAXIS
						case "Z":
							u = ZAXIS
						case "V":
							v = Direction(n1, n2, true)
							u = Cross(v, ZAXIS)
						default:
							return errors.New("unknown direction")
						}
						m := frame.AddMeasure(n1.Coord, n2.Coord, u)
						m.Text = fmt.Sprintf("%.0f", VectorDistance(n1, n2, v)*1000)
						if len(lis) < 5 {
							return nil
						}
						val, err := strconv.ParseFloat(lis[4], 64)
						if err != nil {
							return err
						}
						m.Extension = val
						if len(lis) < 6 {
							return nil
						}
						val, err = strconv.ParseFloat(lis[5], 64)
						if err != nil {
							return err
						}
						m.ArrowSize = val
						if len(lis) < 7 {
							return nil
						}
						val, err = strconv.ParseFloat(lis[6], 64)
						if err != nil {
							return err
						}
						m.Rotate = val
						if len(lis) < 8 {
							return nil
						}
						m.Text = ToUtf8string(lis[7])
					} else {
						return errors.New(fmt.Sprintf("no node %d", nnum))
					}
				} else {
					return errors.New(fmt.Sprintf("no node %d", nnum))
				}
			}
		}
	case "elemcode":
		if un {
			ElemCaptionOff(stw, "EC_NUM")
		} else {
			ElemCaptionOn(stw, "EC_NUM")
		}
	case "sectcode":
		if un {
			ElemCaptionOff(stw, "EC_SECT")
		} else {
			ElemCaptionOn(stw, "EC_SECT")
		}
	case "width":
		if un {
			ElemCaptionOff(stw, "EC_WIDTH")
		} else {
			ElemCaptionOn(stw, "EC_WIDTH")
		}
	case "height":
		if un {
			ElemCaptionOff(stw, "EC_HEIGHT")
		} else {
			ElemCaptionOn(stw, "EC_HEIGHT")
		}
	case "srcancolor":
		if un {
			stw.SetColorMode(ECOLOR_WHITE)
		} else {
			stw.SetColorMode(ECOLOR_RATE)
		}
	case "srcanrate":
		if usage {
			stw.History("'srcanrate [long/short]")
			return nil
		}
		onoff := []bool{false, false, false, false} // long, short, q, m
		if len(lis) >= 2 {
			for _, str := range lis[1:] {
				switch {
				case strings.EqualFold(str, "long"):
					onoff[0] = true
				case strings.EqualFold(str, "short"):
					onoff[1] = true
				case strings.EqualFold(str, "q"):
					onoff[2] = true
				case strings.EqualFold(str, "m"):
					onoff[3] = true
				}
			}
		}
		if onoff[0] == false && onoff[1] == false {
			onoff[0] = true
			onoff[1] = true
		}
		if onoff[2] == false && onoff[3] == false {
			onoff[2] = true
			onoff[3] = true
		}
		names := make([]string, 4)
		ind := 0
		for i := 0; i < 4; i++ {
			if onoff[i] {
				names[ind] = SRCANS[i]
				ind++
			}
		}
		names = names[:ind]
		if un {
			SrcanRateOff(stw, names...)
		} else {
			SrcanRateOn(stw, names...)
		}
	case "energy":
		if un {
			stw.SetColorMode(ECOLOR_WHITE)
			frame.Show.Energy = false
		} else {
			stw.SetColorMode(ECOLOR_ENERGY)
			frame.Show.Energy = true
		}
	case "stress":
		if usage {
			stw.History("'stress [etype/sectcode] [period] [stressname]")
			return nil
		}
		l := len(lis)
		if l < 2 {
			if un {
				for etype := COLUMN; etype <= SLAB; etype++ {
					for i := 0; i < 6; i++ {
						StressOff(stw, etype, uint(i))
					}
				}
				return nil
			} else {
				return NotEnoughArgs("STRESS")
			}
		}
		etype := -1
		var sects []int
		et := strings.ToLower(lis[1])
		switch {
		case abbrev.For("co/lumn", et):
			etype = COLUMN
		case abbrev.For("gi/rder", et):
			etype = GIRDER
		case abbrev.For("br/ace", et):
			etype = BRACE
		case abbrev.For("wb/race", et):
			etype = WBRACE
		case abbrev.For("sb/race", et):
			etype = SBRACE
		case abbrev.For("tr/uss", et):
			etype = TRUSS
		case abbrev.For("wa/ll", et):
			etype = WALL
		case abbrev.For("sl/ab", et):
			etype = SLAB
		}
		if etype == -1 {
			sectnum := regexp.MustCompile("^ *([0-9]+) *$")
			sectrange := regexp.MustCompile("(?i)^ *range *[(] *([0-9]+) *, *([0-9]+) *[)] *$")
			switch {
			case sectnum.MatchString(lis[1]):
				val, _ := strconv.ParseInt(lis[1], 10, 64)
				if _, ok := frame.Sects[int(val)]; ok {
					sects = []int{int(val)}
				}
			case sectrange.MatchString(lis[1]):
				fs := sectrange.FindStringSubmatch(lis[1])
				if len(fs) < 3 {
					return errors.New("STRESS, invalid input")
				}
				start, _ := strconv.ParseInt(fs[1], 10, 64)
				end, _ := strconv.ParseInt(fs[2], 10, 64)
				if start > end {
					return errors.New("STRESS, invalid input")
				}
				sects = make([]int, int(end-start))
				nsect := 0
				for i := int(start); i < int(end); i++ {
					if _, ok := frame.Sects[i]; ok {
						sects[nsect] = i
						nsect++
					}
				}
				sects = sects[:nsect]
			}
			if sects == nil || len(sects) == 0 {
				return errors.New("STRESS, invalid input")
			}
		}
		if l < 3 {
			if sects != nil && len(sects) > 0 {
				if un {
					for _, snum := range sects {
						delete(frame.Show.Stress, snum)
					}
				} else {
					for _, snum := range sects {
						for i := 0; i < 6; i++ {
							StressOff(stw, snum, uint(i))
						}
					}
				}
			} else {
				for i := 0; i < 6; i++ {
					StressOff(stw, etype, uint(i))
				}
			}
			break
		}
		if l < 4 {
			return NotEnoughArgs("STRESS")
		}
		period := strings.ToUpper(lis[2])
		SetPeriod(stw, period)
		index := -1
		val := strings.ToUpper(lis[3])
		for i, str := range []string{"N", "QX", "QY", "MZ", "MX", "MY"} {
			if val == str {
				index = i
				break
			}
		}
		if index == -1 {
			break
		}
		if un {
			if sects != nil && len(sects) > 0 {
				for _, snum := range sects {
					StressOff(stw, snum, uint(index))
				}
			} else {
				StressOff(stw, etype, uint(index))
			}
		} else {
			if sects != nil && len(sects) > 0 {
				for _, snum := range sects {
					StressOn(stw, snum, uint(index))
				}
			} else {
				StressOn(stw, etype, uint(index))
			}
		}
	case "prestress":
		if un {
			ElemCaptionOff(stw, "EC_PREST")
		} else {
			ElemCaptionOn(stw, "EC_PREST")
		}
	case "stiff":
		if usage {
			stw.History("'stiff [x,y]")
			return nil
		}
		if len(lis) < 2 {
			if un {
				ElemCaptionOff(stw, "EC_STIFF_X")
				ElemCaptionOff(stw, "EC_STIFF_Y")
				return nil
			} else {
				return NotEnoughArgs("stiff")
			}
		}
		switch strings.ToUpper(lis[1]) {
		default:
			return errors.New("unknown period")
		case "X":
			if un {
				ElemCaptionOff(stw, "EC_STIFF_X")
			} else {
				ElemCaptionOn(stw, "EC_STIFF_X")
			}
		case "Y":
			if un {
				ElemCaptionOff(stw, "EC_STIFF_Y")
			} else {
				ElemCaptionOn(stw, "EC_STIFF_Y")
			}
		}
	case "drift":
		if usage {
			stw.History("'drift [x,y]")
			return nil
		}
		if len(lis) < 2 {
			if un {
				ElemCaptionOff(stw, "EC_DRIFT_X")
				ElemCaptionOff(stw, "EC_DRIFT_Y")
				return nil
			} else {
				return NotEnoughArgs("drift")
			}
		}
		switch strings.ToUpper(lis[1]) {
		default:
			return errors.New("unknown period")
		case "X":
			if un {
				ElemCaptionOff(stw, "EC_DRIFT_X")
			} else {
				ElemCaptionOn(stw, "EC_DRIFT_X")
			}
		case "Y":
			if un {
				ElemCaptionOff(stw, "EC_DRIFT_Y")
			} else {
				ElemCaptionOn(stw, "EC_DRIFT_Y")
			}
		}
	case "deformation":
		if un {
			DeformationOff(stw)
		} else {
			if len(lis) >= 2 {
				SetPeriod(stw, strings.ToUpper(lis[1]))
			}
			DeformationOn(stw)
		}
	case "disp":
		if un {
			if len(lis) < 2 {
				for i := 0; i < 6; i++ {
					DispOff(stw, i)
				}
			} else {
				SetPeriod(stw, strings.ToUpper(lis[1]))
				dir := strings.ToUpper(lis[2])
				for i, str := range []string{"X", "Y", "Z", "TX", "TY", "TZ"} {
					if dir == str {
						DispOff(stw, i)
						break
					}
				}
			}
		} else {
			if len(lis) < 3 {
				return NotEnoughArgs("DISP")
			}
			SetPeriod(stw, strings.ToUpper(lis[1]))
			dir := strings.ToUpper(lis[2])
			for i, str := range []string{"X", "Y", "Z", "TX", "TY", "TZ"} {
				if dir == str {
					DispOn(stw, i)
					break
				}
			}
		}
	case "eccentric":
		if un {
			frame.Show.Fes = false
		} else {
			frame.Show.Fes = true
			if len(lis) >= 2 {
				val, err := strconv.ParseFloat(lis[1], 64)
				if err != nil {
					return err
				}
				frame.Show.MassSize = val
			}
		}
	case "draw":
		if un {
			for k := range frame.Show.Draw {
				frame.Show.Draw[k] = false
			}
		} else {
			if len(lis) < 2 {
				return NotEnoughArgs("DRAW")
			}
			var val int
			switch {
			case Re_column.MatchString(lis[1]):
				val = COLUMN
			case Re_girder.MatchString(lis[1]):
				val = GIRDER
			case Re_slab.MatchString(lis[1]):
				val = BRACE
			case Re_wall.MatchString(lis[1]):
				val = WALL
			case Re_slab.MatchString(lis[1]):
				val = SLAB
			default:
				tmp, err := strconv.ParseInt(lis[1], 10, 64)
				if err != nil {
					return err
				}
				val = int(tmp)
			}
			if val != 0 {
				frame.Show.Draw[val] = true
			}
			if len(lis) >= 3 {
				tmp, err := strconv.ParseFloat(lis[2], 64)
				if err != nil {
					return err
				}
				frame.Show.DrawSize = tmp
			}
		}
	case "alias":
		if un {
			if len(lis) < 2 {
				stw.ClearSectionAlias()
			} else {
				for _, j := range lis[1:] {
					val, err := strconv.ParseInt(j, 10, 64)
					if err != nil {
						continue
					}
					if _, ok := frame.Sects[int(val)]; ok {
						stw.DeleteSectionAlias(int(val))
					}
				}
			}
		} else {
			if len(lis) < 2 {
				return NotEnoughArgs("ALIAS")
			}
			val, err := strconv.ParseInt(lis[1], 10, 64)
			if err != nil {
				return err
			}
			if _, ok := frame.Sects[int(val)]; ok {
				if len(lis) < 3 {
					stw.AddSectionAlias(int(val), "")
				} else {
					stw.AddSectionAlias(int(val), ToUtf8string(lis[2]))
				}
			}
		}
	case "anonymous":
		for _, str := range lis[1:] {
			val, err := strconv.ParseInt(str, 10, 64)
			if err != nil {
				continue
			}
			if _, ok := frame.Sects[int(val)]; ok {
				stw.AddSectionAlias(int(val), "")
			}
		}
	case "nodecode":
		if un {
			NodeCaptionOff(stw, "NC_NUM")
		} else {
			NodeCaptionOn(stw, "NC_NUM")
		}
	case "weight":
		if un {
			NodeCaptionOff(stw, "NC_WEIGHT")
		} else {
			NodeCaptionOn(stw, "NC_WEIGHT")
		}
	case "conf":
		if un {
			frame.Show.Conf = false
			stw.DisableLabel("CONF")
		} else {
			frame.Show.Conf = true
			stw.EnableLabel("CONF")
			if len(lis) >= 2 {
				val, err := strconv.ParseFloat(lis[1], 64)
				if err != nil {
					return err
				}
				frame.Show.ConfSize = val
				stw.SetLabel("CONFSIZE", fmt.Sprintf("%.1f", val))
			}
		}
	case "pilecode":
		if un {
			NodeCaptionOff(stw, "NC_PILE")
		} else {
			NodeCaptionOn(stw, "NC_PILE")
		}
	case "fence":
		if len(lis) < 3 {
			return NotEnoughArgs("FENCE")
		}
		var axis int
		switch strings.ToUpper(lis[1]) {
		default:
			return errors.New("unknown direction")
		case "X":
			axis = 0
		case "Y":
			axis = 1
		case "Z":
			axis = 2
		}
		val, err := strconv.ParseFloat(lis[2], 64)
		if err != nil {
			return err
		}
		stw.SelectElem(frame.Fence(axis, val, false))
		HideNotSelected(stw)
	case "period":
		if len(lis) < 2 {
			return NotEnoughArgs("PERIOD")
		}
		SetPeriod(stw, strings.ToUpper(lis[1]))
	case "period++":
		IncrementPeriod(stw, 1)
	case "period--":
		IncrementPeriod(stw, -1)
	case "nocaption":
		for _, nc := range NODECAPTIONS {
			NodeCaptionOff(stw, nc)
		}
		for _, ec := range ELEMCAPTIONS {
			ElemCaptionOff(stw, ec)
		}
		for etype := range ETYPES[1:] {
			for i := 0; i < 6; i++ {
				StressOff(stw, etype, uint(i))
			}
		}
	case "nolegend":
		if un {
			frame.Show.NoLegend = false
		} else {
			frame.Show.NoLegend = true
		}
	case "noshearvalue":
		if un {
			frame.Show.NoShearValue = false
		} else {
			frame.Show.NoShearValue = true
		}
	case "nomomentvalue":
		if un {
			frame.Show.NoMomentValue = false
		} else {
			frame.Show.NoMomentValue = true
		}
	case "sheararrow":
		if un {
			frame.Show.ShearArrow = false
		} else {
			frame.Show.ShearArrow = true
		}
	case "momentfigure":
		if un {
			frame.Show.MomentFigure = false
		} else {
			frame.Show.MomentFigure = true
		}
	case "ncolor":
		stw.SetColorMode(ECOLOR_N)
	case "pagetitle":
		if un {
			stw.TextBox("PAGETITLE").Clear()
			stw.TextBox("PAGETITLE").Hide()
		} else {
			stw.TextBox("PAGETITLE").AddText(ToUtf8string(strings.Join(lis[1:], " ")))
			stw.TextBox("PAGETITLE").Show()
		}
	case "title":
		if un {
			stw.TextBox("TITLE").Clear()
			stw.TextBox("TITLE").Hide()
		} else {
			stw.TextBox("TITLE").AddText(ToUtf8string(strings.Join(lis[1:], " ")))
			stw.TextBox("TITLE").Show()
		}
	case "text":
		if un {
			stw.TextBox("TEXT").Clear()
			stw.TextBox("TEXT").Hide()
		} else {
			stw.TextBox("TEXT").AddText(ToUtf8string(strings.Join(lis[1:], " ")))
			stw.TextBox("TEXT").Show()
		}
	case "position":
		if len(lis) < 4 {
			return NotEnoughArgs("POSITION")
		}
		xpos, err := strconv.ParseFloat(lis[2], 64)
		if err != nil {
			return err
		}
		ypos, err := strconv.ParseFloat(lis[3], 64)
		if err != nil {
			return err
		}
		switch strings.ToUpper(lis[1]) {
		case "PAGETITLE":
			stw.TextBox("PAGETITLE").SetPosition(xpos, ypos)
		case "TITLE":
			stw.TextBox("TITLE").SetPosition(xpos, ypos)
		case "TEXT":
			stw.TextBox("TEXT").SetPosition(xpos, ypos)
		case "LEGEND":
			frame.Show.LegendPosition[0] = int(xpos)
			frame.Show.LegendPosition[1] = int(ypos)
		}
	}
	return nil
}
