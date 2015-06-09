package stgui

import (
	"errors"
	"fmt"
	"github.com/yofu/abbrev"
	"github.com/yofu/st/stlib"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	fig2abbrev = []string {
		"gf/act", "foc/us", "ang/le", "dist/s", "pers/pective", "ax/onometric", "df/act", "rf/act", "qf/act", "mf/act", "gax/is", "eax/is",
		"noax/is", "el/em", "el/em/+/", "el/em/-/", "sec/tion", "sec/tion/+/", "sec/tion/-/", "k/ijun", "mea/sure", "el/em/c/ode", "sec/t/c/ode",
		"wid/th", "h/eigh/t/", "sr/can/col/or", "sr/can/ra/te", "st/ress", "prest/ress", "stiff/", "def/ormation", "dis/p", "ecc/entric", "dr/aw",
		"al/ias", "anon/ymous", "no/de/c/ode", "wei/ght", "con/f", "pi/lecode", "fen/ce", "per/iod", "per/iod/++/", "per/iod/--/",
		"nocap/tion", "noleg/end", "nos/hear/v/alue", "nom/oment/v/alue", "s/hear/ar/row", "m/oment/fig/ure", "ncol/or", "p/age/tit/le", "tit/le", "pos/ition",
	}
)

func (stw *Window) fig2mode(command string) error {
	if len(command) == 1 {
		return st.NotEnoughArgs("fig2mode")
	}
	if command == "'." {
		return stw.fig2mode(stw.lastfig2command)
	}
	stw.lastfig2command = command
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
	return stw.fig2keyword(args, un)
}

func fig2keywordcomplete(command string) (string, bool) {
	usage := strings.HasSuffix(command, "?")
	cname := strings.TrimSuffix(command, "?")
	cname = strings.ToLower(strings.TrimPrefix(cname, "'"))
	var rtn string
	for _, ab := range fig2abbrev {
		pat := abbrev.MustCompile(ab)
		if pat.MatchString(cname) {
			rtn = pat.Longest()
			break
		}
	}
	if rtn == "" {
		rtn = cname
	}
	return rtn, usage
}

func (stw *Window) fig2keyword(lis []string, un bool) error {
	if len(lis) < 1 {
		return st.NotEnoughArgs("Fig2Keyword")
	}
	showhtml := func(fn string) {
		f := filepath.Join(tooldir, "fig2/keywords", fn)
		if st.FileExists(f) {
			cmd := exec.Command("cmd", "/C", "start", f)
			cmd.Start()
		}
	}
	key, usage := fig2keywordcomplete(strings.ToLower(lis[0]))
	switch key {
	default:
		if k, ok := stw.Frame.Kijuns[key]; ok {
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
			if st.IsParallel(d, st.XAXIS, EPS) {
				axisrange(stw, 1, k.Start[1] + min, k.Start[1] + max, false)
			} else if st.IsParallel(d, st.YAXIS, EPS) {
				axisrange(stw, 0, k.Start[0] + min, k.Start[0] + max, false)
			} else {
				for _, n := range stw.Frame.Nodes {
					n.Hide()
					ok, err := k.Contains(n.Coord, st.ZAXIS, min, max)
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
		stw.errormessage(errors.New(fmt.Sprintf("no fig2 keyword: %s", key)), INFO)
		return nil
	case "gfact":
		val, err := strconv.ParseFloat(lis[1], 64)
		if err != nil {
			return err
		}
		stw.Frame.View.Gfact = val
		stw.Labels["GFACT"].SetAttribute("VALUE", fmt.Sprintf("%f", stw.Frame.View.Gfact))
	case "focus":
		if len(lis) < 2 {
			return st.NotEnoughArgs("FOCUS")
		}
		switch strings.ToUpper(lis[1]) {
		case "CENTER", "CENTRE":
			stw.Frame.SetFocus(nil)
		case "NODE":
			val, err := strconv.ParseInt(lis[2], 10, 64)
			if err != nil {
				return err
			}
			if n, ok := stw.Frame.Nodes[int(val)]; ok {
				stw.Frame.SetFocus(n.Coord)
			}
			w, h := stw.cdcanv.GetSize()
			stw.Frame.View.Center[0] = float64(w) * 0.5
			stw.Frame.View.Center[1] = float64(h) * 0.5
		case "ELEM":
			val, err := strconv.ParseInt(lis[2], 10, 64)
			if err != nil {
				return err
			}
			if el, ok := stw.Frame.Elems[int(val)]; ok {
				stw.Frame.SetFocus(el.MidPoint())
			}
			w, h := stw.cdcanv.GetSize()
			stw.Frame.View.Center[0] = float64(w) * 0.5
			stw.Frame.View.Center[1] = float64(h) * 0.5
		default:
			if len(lis) < 4 {
				return st.NotEnoughArgs("FOCUS")
			}
			for i, str := range []string{"FOCUSX", "FOCUSY", "FOCUSZ"} {
				if lis[1+i] == "_" {
					continue
				}
				val, err := strconv.ParseFloat(lis[1+i], 64)
				if err != nil {
					return err
				}
				stw.Frame.View.Focus[i] = val
				stw.Labels[str].SetAttribute("VALUE", fmt.Sprintf("%f", stw.Frame.View.Focus[i]))
			}
		}
	case "fit":
		stw.Frame.SetFocus(nil)
		stw.DrawFrameNode()
		stw.ShowCenter()
	case "angle":
		if len(lis) < 3 {
			return st.NotEnoughArgs("ANGLE")
		}
		for i, str := range []string{"PHI", "THETA"} {
			if lis[1+i] == "_" {
				continue
			}
			val, err := strconv.ParseFloat(lis[1+i], 64)
			if err != nil {
				return err
			}
			stw.Frame.View.Angle[i] = val
			stw.Labels[str].SetAttribute("VALUE", fmt.Sprintf("%f", stw.Frame.View.Angle[i]))
		}
	case "dists":
		if len(lis) < 3 {
			return st.NotEnoughArgs("DISTS")
		}
		for i, str := range []string{"DISTR", "DISTL"} {
			if lis[1+i] == "_" {
				continue
			}
			val, err := strconv.ParseFloat(lis[1+i], 64)
			if err != nil {
				return err
			}
			stw.Frame.View.Dists[i] = val
			stw.Labels[str].SetAttribute("VALUE", fmt.Sprintf("%f", stw.Frame.View.Dists[i]))
		}
	case "perspective":
		stw.Frame.View.Perspective = true
	case "axonometric":
		stw.Frame.View.Perspective = false
	case "unit":
		if usage {
			stw.addHistory("'unit force,length")
			stw.addHistory(fmt.Sprintf("CURRENT FORCE UNIT: %s %.3f", stw.Frame.Show.UnitName[0], stw.Frame.Show.Unit[0]))
			stw.addHistory(fmt.Sprintf("CURRENT LENGTH UNIT: %s %.3f", stw.Frame.Show.UnitName[1], stw.Frame.Show.Unit[1]))
			showhtml("UNIT.html")
			return nil
		}
		if un {
			stw.Frame.Show.Unit = []float64{1.0, 1.0}
			stw.Frame.Show.UnitName = []string{"tf", "m"}
			return nil
		}
		if len(lis) < 2 {
			return st.NotEnoughArgs("UNIT")
		}
		ustr := strings.Split(strings.ToLower(lis[1]), ",")
		if len(ustr) < 2 {
			return errors.New("'unit: incorrect format")
		}
		switch ustr[0] {
		case "tf":
			stw.Frame.Show.Unit[0] = 1.0
			stw.Frame.Show.UnitName[0] = "tf"
		case "kgf":
			stw.Frame.Show.Unit[0] = 1000.0
			stw.Frame.Show.UnitName[0] = "kgf"
		case "kn":
			stw.Frame.Show.Unit[0] = st.SI
			stw.Frame.Show.UnitName[0] = "kN"
		}
		switch ustr[1] {
		case "m":
			stw.Frame.Show.Unit[1] = 1.0
			stw.Frame.Show.UnitName[1] = "m"
		case "cm":
			stw.Frame.Show.Unit[0] = 100.0
			stw.Frame.Show.UnitName[0] = "cm"
		case "mm":
			stw.Frame.Show.Unit[0] = 1000.0
			stw.Frame.Show.UnitName[0] = "mm"
		}
	case "dfact":
		if len(lis) < 2 {
			return st.NotEnoughArgs("DFACT")
		}
		val, err := strconv.ParseFloat(lis[1], 64)
		if err != nil {
			return err
		}
		stw.Frame.Show.Dfact = val
		stw.Labels["DFACT"].SetAttribute("VALUE", fmt.Sprintf("%f", val))
	case "rfact":
		if len(lis) < 2 {
			return st.NotEnoughArgs("RFACT")
		}
		val, err := strconv.ParseFloat(lis[1], 64)
		if err != nil {
			return err
		}
		stw.Frame.Show.Rfact = val
	case "qfact":
		if len(lis) < 2 {
			return st.NotEnoughArgs("QFACT")
		}
		val, err := strconv.ParseFloat(lis[1], 64)
		if err != nil {
			return err
		}
		stw.Frame.Show.Qfact = val
		stw.Labels["QFACT"].SetAttribute("VALUE", fmt.Sprintf("%f", val))
	case "mfact":
		if len(lis) < 2 {
			return st.NotEnoughArgs("MFACT")
		}
		val, err := strconv.ParseFloat(lis[1], 64)
		if err != nil {
			return err
		}
		stw.Frame.Show.Mfact = val
		stw.Labels["MFACT"].SetAttribute("VALUE", fmt.Sprintf("%f", val))
	case "gaxis":
		if un {
			stw.Frame.Show.GlobalAxis = false
			stw.Labels["GAXIS"].SetAttribute("FGCOLOR", labelOFFColor)
		} else {
			stw.Frame.Show.GlobalAxis = true
			stw.Labels["GAXIS"].SetAttribute("FGCOLOR", labelFGColor)
			if len(lis) >= 2 {
				val, err := strconv.ParseFloat(lis[1], 64)
				if err != nil {
					return err
				}
				stw.Frame.Show.GlobalAxisSize = val
				stw.Labels["GAXISSIZE"].SetAttribute("VALUE", fmt.Sprintf("%f", val))
			}
		}
	case "eaxis":
		if un {
			stw.Frame.Show.ElementAxis = false
			stw.Labels["EAXIS"].SetAttribute("FGCOLOR", labelOFFColor)
		} else {
			stw.Frame.Show.ElementAxis = true
			stw.Labels["EAXIS"].SetAttribute("FGCOLOR", labelFGColor)
			if len(lis) >= 2 {
				val, err := strconv.ParseFloat(lis[1], 64)
				if err != nil {
					return err
				}
				stw.Frame.Show.ElementAxisSize = val
				stw.Labels["EAXISSIZE"].SetAttribute("VALUE", fmt.Sprintf("%f", val))
			}
		}
	case "noaxis":
		stw.Frame.Show.GlobalAxis = false
		stw.Labels["GAXIS"].SetAttribute("FGCOLOR", labelOFFColor)
		stw.Frame.Show.ElementAxis = false
		stw.Labels["EAXIS"].SetAttribute("FGCOLOR", labelOFFColor)
	case "elem":
		for i, _ := range st.ETYPES {
			stw.HideEtype(i)
		}
		for _, val := range lis[1:] {
			et := strings.ToUpper(val)
			for i, e := range st.ETYPES {
				if et == e {
					stw.ShowEtype(i)
				}
			}
		}
	case "elem+":
		for _, val := range lis[1:] {
			et := strings.ToUpper(val)
			for i, e := range st.ETYPES {
				if et == e {
					stw.ShowEtype(i)
				}
			}
		}
	case "elem-":
		for _, val := range lis[1:] {
			et := strings.ToUpper(val)
			for i, e := range st.ETYPES {
				if et == e {
					stw.HideEtype(i)
				}
			}
		}
	case "section":
		stw.HideAllSection()
		for _, tmp := range lis[1:] {
			val, err := strconv.ParseInt(tmp, 10, 64)
			if err != nil {
				continue
			}
			stw.ShowSection(int(val))
		}
	case "section+":
		for _, tmp := range lis[1:] {
			val, err := strconv.ParseInt(tmp, 10, 64)
			if err != nil {
				continue
			}
			stw.ShowSection(int(val))
		}
	case "section-":
		for _, tmp := range lis[1:] {
			val, err := strconv.ParseInt(tmp, 10, 64)
			if err != nil {
				continue
			}
			stw.HideSection(int(val))
		}
	case "kijun":
		if un {
			stw.Frame.Show.Kijun = false
			stw.Labels["KIJUN"].SetAttribute("FGCOLOR", labelOFFColor)
		} else {
			stw.Frame.Show.Kijun = true
			stw.Labels["KIJUN"].SetAttribute("FGCOLOR", labelFGColor)
		}
	case "measure":
		if usage {
			stw.addHistory("'measure kijun x1 x2 offset dotsize rotate overwrite")
			stw.addHistory("'measure nnum1 nnum2 direction offset dotsize rotate overwrite")
			showhtml("MEASURE.html")
			return nil
		}
		if un {
			stw.Frame.Show.Measure = false
		} else {
			stw.Frame.Show.Measure = true
			if len(lis) < 4 {
				return st.NotEnoughArgs("MEASURE")
			}
			if abbrev.For("k/ijun", strings.ToLower(lis[1])) { // measure kijun x1 x2 offset dotsize rotate overwrite
				if k1, ok := stw.Frame.Kijuns[strings.ToLower(lis[2])]; ok {
					if k2, ok := stw.Frame.Kijuns[strings.ToLower(lis[3])]; ok {
						m := stw.Frame.AddMeasure(k1.Start, k2.Start, k1.Direction())
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
						m.Text = st.ToUtf8string(lis[7])
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
				if n1, ok := stw.Frame.Nodes[int(nnum)]; ok {
					nnum, err := strconv.ParseInt(lis[2], 10, 64)
					if err != nil {
						return err
					}
					if n2, ok := stw.Frame.Nodes[int(nnum)]; ok {
						var u, v []float64
						switch strings.ToUpper(lis[3]) {
						case "X":
							u = st.XAXIS
							v = st.YAXIS
						case "Y":
							u = st.YAXIS
							v = st.XAXIS
						case "Z":
							u = st.ZAXIS
						case "V":
							v = st.Direction(n1, n2, true)
							u = st.Cross(v, st.ZAXIS)
						default:
							return errors.New("unknown direction")
						}
						m := stw.Frame.AddMeasure(n1.Coord, n2.Coord, u)
						m.Text = fmt.Sprintf("%.0f", st.VectorDistance(n1, n2, v)*1000)
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
						m.Text = st.ToUtf8string(lis[7])
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
			stw.ElemCaptionOff("EC_NUM")
		} else {
			stw.ElemCaptionOn("EC_NUM")
		}
	case "sectcode":
		if un {
			stw.ElemCaptionOff("EC_SECT")
		} else {
			stw.ElemCaptionOn("EC_SECT")
		}
	case "width":
		if un {
			stw.ElemCaptionOff("EC_WIDTH")
		} else {
			stw.ElemCaptionOn("EC_WIDTH")
		}
	case "height":
		if un {
			stw.ElemCaptionOff("EC_HEIGHT")
		} else {
			stw.ElemCaptionOn("EC_HEIGHT")
		}
	case "srcancolor":
		if un {
			stw.SetColorMode(st.ECOLOR_WHITE)
		} else {
			stw.SetColorMode(st.ECOLOR_RATE)
		}
	case "srcanrate":
		if usage {
			stw.addHistory("'srcanrate [long/short]")
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
		for i:=0; i<4; i++ {
			if onoff[i] {
				names[ind] = st.SRCANS[i]
				ind++
			}
		}
		names = names[:ind]
		if un {
			stw.SrcanRateOff(names...)
		} else {
			stw.SrcanRateOn(names...)
		}
	case "stress":
		if usage {
			stw.addHistory("'stress [etype/sectcode] [period] [stressname]")
			return nil
		}
		l := len(lis)
		if l < 2 {
			if un {
				for etype := st.COLUMN; etype <= st.SLAB; etype++ {
					for i := 0; i < 6; i++ {
						stw.StressOff(etype, uint(i))
					}
				}
				return nil
			} else {
				return st.NotEnoughArgs("STRESS")
			}
		}
		etype := -1
		var sects []int
		et := strings.ToLower(lis[1])
		switch {
		case abbrev.For("co/lumn", et):
			etype = st.COLUMN
		case abbrev.For("gi/rder", et):
			etype = st.GIRDER
		case abbrev.For("br/ace", et):
			etype = st.BRACE
		case abbrev.For("wb/race", et):
			etype = st.WBRACE
		case abbrev.For("sb/race", et):
			etype = st.SBRACE
		case abbrev.For("tr/uss", et):
			etype = st.TRUSS
		case abbrev.For("wa/ll", et):
			etype = st.WALL
		case abbrev.For("sl/ab", et):
			etype = st.SLAB
		}
		if etype == -1 {
			sectnum := regexp.MustCompile("^ *([0-9]+) *$")
			sectrange := regexp.MustCompile("(?i)^ *range *[(] *([0-9]+) *, *([0-9]+) *[)] *$")
			switch {
			case sectnum.MatchString(lis[1]):
				val, _ := strconv.ParseInt(lis[1], 10, 64)
				if _, ok := stw.Frame.Sects[int(val)]; ok {
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
					if _, ok := stw.Frame.Sects[i]; ok {
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
						delete(stw.Frame.Show.Stress, snum)
					}
				} else {
					for _, snum := range sects {
						for i := 0; i < 6; i++ {
							stw.StressOff(snum, uint(i))
						}
					}
				}
			} else {
				for i := 0; i < 6; i++ {
					stw.StressOff(etype, uint(i))
				}
			}
			break
		}
		if l < 4 {
			return st.NotEnoughArgs("STRESS")
		}
		period := strings.ToUpper(lis[2])
		stw.SetPeriod(period)
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
					stw.StressOff(snum, uint(index))
				}
			} else {
				stw.StressOff(etype, uint(index))
			}
		} else {
			if sects != nil && len(sects) > 0 {
				for _, snum := range sects {
					stw.StressOn(snum, uint(index))
				}
			} else {
				stw.StressOn(etype, uint(index))
			}
		}
	case "prestress":
		if un {
			stw.ElemCaptionOff("EC_PREST")
		} else {
			stw.ElemCaptionOn("EC_PREST")
		}
	case "stiff":
		if usage {
			stw.addHistory("'stiff [x,y]")
			return nil
		}
		if len(lis) < 2 {
			if un {
				stw.ElemCaptionOff("EC_STIFF_X")
				stw.ElemCaptionOff("EC_STIFF_Y")
				return nil
			} else {
				return st.NotEnoughArgs("stiff")
			}
		}
		switch strings.ToUpper(lis[1]) {
		default:
			return errors.New("unknown period")
		case "X":
			if un {
				stw.ElemCaptionOff("EC_STIFF_X")
			} else {
				stw.ElemCaptionOn("EC_STIFF_X")
			}
		case "Y":
			if un {
				stw.ElemCaptionOff("EC_STIFF_Y")
			} else {
				stw.ElemCaptionOn("EC_STIFF_Y")
			}
		}
	case "drift":
		if usage {
			stw.addHistory("'drift [x,y]")
			return nil
		}
		if len(lis) < 2 {
			if un {
				stw.ElemCaptionOff("EC_DRIFT_X")
				stw.ElemCaptionOff("EC_DRIFT_Y")
				return nil
			} else {
				return st.NotEnoughArgs("drift")
			}
		}
		switch strings.ToUpper(lis[1]) {
		default:
			return errors.New("unknown period")
		case "X":
			if un {
				stw.ElemCaptionOff("EC_DRIFT_X")
			} else {
				stw.ElemCaptionOn("EC_DRIFT_X")
			}
		case "Y":
			if un {
				stw.ElemCaptionOff("EC_DRIFT_Y")
			} else {
				stw.ElemCaptionOn("EC_DRIFT_Y")
			}
		}
	case "deformation":
		if un {
			stw.DeformationOff()
		} else {
			if len(lis) >= 2 {
				stw.SetPeriod(strings.ToUpper(lis[1]))
			}
			stw.DeformationOn()
		}
	case "disp":
		if un {
			if len(lis) < 2 {
				for i := 0; i < 6; i++ {
					stw.DispOff(i)
				}
			} else {
				stw.SetPeriod(strings.ToUpper(lis[1]))
				dir := strings.ToUpper(lis[2])
				for i, str := range []string{"X", "Y", "Z", "TX", "TY", "TZ"} {
					if dir == str {
						stw.DispOff(i)
						break
					}
				}
			}
		} else {
			if len(lis) < 3 {
				return st.NotEnoughArgs("DISP")
			}
			stw.SetPeriod(strings.ToUpper(lis[1]))
			dir := strings.ToUpper(lis[2])
			for i, str := range []string{"X", "Y", "Z", "TX", "TY", "TZ"} {
				if dir == str {
					stw.DispOn(i)
					break
				}
			}
		}
	case "eccentric":
		if un {
			stw.Frame.Show.Fes = false
		} else {
			stw.Frame.Show.Fes = true
			if len(lis) >= 2 {
				val, err := strconv.ParseFloat(lis[1], 64)
				if err != nil {
					return err
				}
				stw.Frame.Show.MassSize = val
			}
		}
	case "draw":
		if un {
			for k := range stw.Frame.Show.Draw {
				stw.Frame.Show.Draw[k] = false
			}
		} else {
			if len(lis) < 2 {
				return st.NotEnoughArgs("DRAW")
			}
			var val int
			switch {
			case re_column.MatchString(lis[1]):
				val = st.COLUMN
			case re_girder.MatchString(lis[1]):
				val = st.GIRDER
			case re_slab.MatchString(lis[1]):
				val = st.BRACE
			case re_wall.MatchString(lis[1]):
				val = st.WALL
			case re_slab.MatchString(lis[1]):
				val = st.SLAB
			default:
				tmp, err := strconv.ParseInt(lis[1], 10, 64)
				if err != nil {
					return err
				}
				val = int(tmp)
			}
			if val != 0 {
				stw.Frame.Show.Draw[val] = true
			}
			if len(lis) >= 3 {
				tmp, err := strconv.ParseFloat(lis[2], 64)
				if err != nil {
					return err
				}
				stw.Frame.Show.DrawSize = tmp
			}
		}
	case "alias":
		if un {
			if len(lis) < 2 {
				sectionaliases = make(map[int]string, 0)
			} else {
				for _, j := range lis[1:] {
					val, err := strconv.ParseInt(j, 10, 64)
					if err != nil {
						continue
					}
					if _, ok := stw.Frame.Sects[int(val)]; ok {
						delete(sectionaliases, int(val))
					}
				}
			}
		} else {
			if len(lis) < 2 {
				return st.NotEnoughArgs("ALIAS")
			}
			val, err := strconv.ParseInt(lis[1], 10, 64)
			if err != nil {
				return err
			}
			if _, ok := stw.Frame.Sects[int(val)]; ok {
				if len(lis) < 3 {
					sectionaliases[int(val)] = ""
				} else {
					sectionaliases[int(val)] = st.ToUtf8string(lis[2])
				}
			}
		}
	case "anonymous":
		for _, str := range lis[1:] {
			val, err := strconv.ParseInt(str, 10, 64)
			if err != nil {
				continue
			}
			if _, ok := stw.Frame.Sects[int(val)]; ok {
				sectionaliases[int(val)] = ""
			}
		}
	case "nodecode":
		if un {
			stw.NodeCaptionOff("NC_NUM")
		} else {
			stw.NodeCaptionOn("NC_NUM")
		}
	case "weight":
		if un {
			stw.NodeCaptionOff("NC_WEIGHT")
		} else {
			stw.NodeCaptionOn("NC_WEIGHT")
		}
	case "conf":
		if un {
			stw.Frame.Show.Conf = false
			stw.Labels["CONF"].SetAttribute("FGCOLOR", labelOFFColor)
		} else {
			stw.Frame.Show.Conf = true
			stw.Labels["CONF"].SetAttribute("FGCOLOR", labelFGColor)
			if len(lis) >= 2 {
				val, err := strconv.ParseFloat(lis[1], 64)
				if err != nil {
					return err
				}
				stw.Frame.Show.ConfSize = val
				stw.Labels["CONFSIZE"].SetAttribute("VALUE", fmt.Sprintf("%.1f", val))
			}
		}
	case "pilecode":
		if un {
			stw.NodeCaptionOff("NC_PILE")
		} else {
			stw.NodeCaptionOn("NC_PILE")
		}
	case "fence":
		if len(lis) < 3 {
			return st.NotEnoughArgs("FENCE")
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
		stw.SelectElem = stw.Frame.Fence(axis, val, false)
		stw.HideNotSelected()
	case "period":
		if len(lis) < 2 {
			return st.NotEnoughArgs("PERIOD")
		}
		stw.SetPeriod(strings.ToUpper(lis[1]))
	case "period++":
		stw.IncrementPeriod(1)
	case "period--":
		stw.IncrementPeriod(-1)
	case "nocaption":
		for _, nc := range st.NODECAPTIONS {
			stw.NodeCaptionOff(nc)
		}
		for _, ec := range st.ELEMCAPTIONS {
			stw.ElemCaptionOff(ec)
		}
		for etype := range st.ETYPES[1:] {
			for i := 0; i < 6; i++ {
				stw.StressOff(etype, uint(i))
			}
		}
	case "nolegend":
		if un {
			stw.Frame.Show.NoLegend = false
		} else {
			stw.Frame.Show.NoLegend = true
		}
	case "noshearvalue":
		if un {
			stw.Frame.Show.NoShearValue = false
		} else {
			stw.Frame.Show.NoShearValue = true
		}
	case "nomomentvalue":
		if un {
			stw.Frame.Show.NoMomentValue = false
		} else {
			stw.Frame.Show.NoMomentValue = true
		}
	case "sheararrow":
		if un {
			stw.Frame.Show.ShearArrow = false
		} else {
			stw.Frame.Show.ShearArrow = true
		}
	case "momentfigure":
		if un {
			stw.Frame.Show.MomentFigure = false
		} else {
			stw.Frame.Show.MomentFigure = true
		}
	case "ncolor":
		stw.SetColorMode(st.ECOLOR_N)
	case "pagetitle":
		if un {
			stw.PageTitle.Value = make([]string, 0)
			stw.PageTitle.Hide = true
		} else {
			stw.PageTitle.Value = append(stw.PageTitle.Value, st.ToUtf8string(strings.Join(lis[1:], " ")))
			stw.PageTitle.Hide = false
		}
	case "title":
		if un {
			stw.Title.Value = make([]string, 0)
			stw.Title.Hide = true
		} else {
			stw.Title.Value = append(stw.Title.Value, st.ToUtf8string(strings.Join(lis[1:], " ")))
			stw.Title.Hide = false
		}
	case "text":
		if un {
			stw.Text.Value = make([]string, 0)
			stw.Text.Hide = true
		} else {
			stw.Text.Value = append(stw.Text.Value, st.ToUtf8string(strings.Join(lis[1:], " ")))
			stw.Text.Hide = false
		}
	case "position":
		if len(lis) < 4 {
			return st.NotEnoughArgs("POSITION")
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
			stw.PageTitle.Position[0] = xpos
			stw.PageTitle.Position[1] = ypos
		case "TITLE":
			stw.Title.Position[0] = xpos
			stw.Title.Position[1] = ypos
		case "TEXT":
			stw.Text.Position[0] = xpos
			stw.Text.Position[1] = ypos
		case "LEGEND":
			stw.Frame.Show.LegendPosition[0] = int(xpos)
			stw.Frame.Show.LegendPosition[1] = int(ypos)
		}
	}
	return nil
}

