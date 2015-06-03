package stgui

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/yofu/abbrev"
	"github.com/yofu/ps"
	"github.com/yofu/st/stlib"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	exabbrev = []string{
		"e/dit", "q/uit", "vi/m", "hk/you", "hw/eak", "rp/ipe", "cp/ipe", "tk/you", "ck/you", "pla/te", "fixr/otate", "fixm/ove", "noun/do", "un/do", "w/rite", "sav/e", "inc/rement", "c/heck", "r/ead",
		"ins/ert", "p/rop/s/ect", "w/rite/o/utput", "w/rite/rea/ction", "nmi/nteraction", "har/dcopy", "fi/g2", "fe/nce", "no/de", "xsc/ale", "ysc/ale", "zsc/ale", "pl/oad", "z/oubun/d/isp", "z/oubun/r/eaction",
		"fac/ts", "go/han/l/st", "el/em", "ave/rage", "bo/nd", "ax/is/2//c/ang", "resul/tant", "prest/ress", "therm/al", "div/ide", "e/lem/dup/lication", "i/ntersect/a/ll", "co/nf",
		"pi/le", "sec/tion", "c/urrent/v/alue", "an/alysis", "f/ilter", "h/eigh/t/", "h/eigh/t+/", "h/eigh/t-/", "sec/tion/+/", "col/or", "ex/tractarclm", "a/rclm/001/", "a/rclm/201/", "a/rclm/301/",
	}
)

func exmodecomplete(command string) (string, bool, bool) {
	usage := strings.HasSuffix(command, "?")
	cname := strings.TrimSuffix(command, "?")
	bang := strings.HasSuffix(cname, "!")
	cname = strings.TrimSuffix(cname, "!")
	cname = strings.ToLower(strings.TrimPrefix(cname, ":"))
	var rtn string
	for _, ab := range exabbrev {
		pat := abbrev.MustCompile(ab)
		if pat.MatchString(cname) {
			rtn = pat.Longest()
			break
		}
	}
	if rtn == "" {
		rtn = cname
	}
	return rtn, bang, usage
}

func (stw *Window) emptyexmodech() {
ex_empty:
	for {
		select {
		case <-time.After(time.Second):
			break ex_empty
		case <-stw.exmodeend:
			break ex_empty
		case <-stw.exmodech:
			continue ex_empty
		}
	}
}

func (stw *Window) exmode(command string) error {
	if !strings.Contains(command, "|") {
		return stw.excommand(command, false)
	}
	excms := strings.Split(command, "|")
	defer stw.emptyexmodech()
	for _, com := range excms {
		err := stw.excommand(com, true)
		if err != nil {
			return err
		}
	}
	return nil
}

func (stw *Window) excommand(command string, pipe bool) error {
	if len(command) == 1 {
		return st.NotEnoughArgs("exmode")
	}
	if command == ":." {
		return stw.exmode(stw.lastexcommand)
	}
	stw.lastexcommand = command
	tmpargs := strings.Split(command, " ")
	args := make([]string, len(tmpargs))
	argdict := make(map[string]string, 0)
	narg := 0
	for i := 0; i < len(tmpargs); i++ {
		if tmpargs[i] != "" {
			args[narg] = tmpargs[i]
			narg++
		}
	}
	args = args[:narg]
	unnamed := make([]string, narg)
	tmpnarg := 0
	namedarg := regexp.MustCompile("^ *-{1,2}([a-zA-Z]+)(={0,1})([^ =]*) *$")
	for _, a := range args {
		if namedarg.MatchString(a) {
			fs := namedarg.FindStringSubmatch(a)
			if fs[2] == "" {
				argdict[strings.ToUpper(fs[1])] = ""
			} else {
				argdict[strings.ToUpper(fs[1])] = fs[3]
			}
		} else {
			unnamed[tmpnarg] = a
			tmpnarg++
		}
	}
	args = unnamed[:tmpnarg]
	narg = tmpnarg
	var fn string
	if narg < 2 {
		fn = ""
	} else {
		fn = stw.Complete(args[1])
		if filepath.Dir(fn) == "." {
			fn = filepath.Join(stw.Cwd, fn)
		}
	}
	cname, bang, usage := exmodecomplete(args[0])
	evaluated := true
	var sender []interface{}
	defer func() {
		if pipe {
			go func(ents []interface{}) {
				for _, e := range ents {
					stw.exmodech <- e
				}
				stw.exmodeend <- 1
			}(sender)
		}
	}()
	switch cname {
	default:
		evaluated = false
	case "edit":
		if usage {
			stw.addHistory(":edit filename {-u=.strc}")
			return nil
		}
		if !bang && stw.Changed {
			if stw.Yn("CHANGED", "変更を保存しますか") {
				stw.SaveAS()
			} else {
				return errors.New("not saved")
			}
		}
		readrc := true
		if rc, ok := argdict["U"]; ok {
			if rc == "NONE" || rc == "" {
				readrc = false
			}
		}
		if fn != "" {
			if !st.FileExists(fn) {
				sfn, err := stw.SearchFile(args[1])
				if err != nil {
					return err
				}
				err = stw.OpenFile(sfn, readrc)
				if err != nil {
					return err
				}
				stw.Redraw()
			} else {
				err := stw.OpenFile(fn, readrc)
				if err != nil {
					return err
				}
				stw.Redraw()
			}
		} else {
			stw.Reload()
		}
	case "quit":
		if usage {
			stw.addHistory(":quit")
			return nil
		}
		stw.Close(bang)
	case "eps":
		if usage {
			stw.addHistory(":eps val")
			return nil
		}
		if narg < 2 {
			return st.NotEnoughArgs(":eps")
		}
		val, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return err
		}
		EPS = val
		stw.addHistory(fmt.Sprintf("EPS=%.3E", EPS))
	case "fitscale":
		if usage {
			stw.addHistory(":fitscale val")
			return nil
		}
		if narg < 2 {
			return st.NotEnoughArgs(":fitscale")
		}
		val, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return err
		}
		CanvasFitScale = val
		stw.addHistory(fmt.Sprintf("FITSCALE=%.3E", CanvasFitScale))
	case "mkdir":
		if usage {
			stw.addHistory(":mkdir dirname")
			return nil
		}
		os.MkdirAll(fn, 0644)
	case "#":
		if usage {
			stw.addHistory(":#")
			return nil
		}
		stw.ShowRecently()
	case "vim":
		if usage {
			stw.addHistory(":vim filename")
			return nil
		}
		stw.Vim(fn)
	case "hkyou":
		if usage {
			stw.addHistory(":hkyou h b tw tf")
			return nil
		}
		if narg < 5 {
			return st.NotEnoughArgs(":hkyou")
		}
		al, err := st.NewHKYOU(args[1:5])
		if err != nil {
			return err
		}
		stw.ShapeData(al)
		if pipe {
			sender = []interface{}{al}
		}
	case "hweak":
		if usage {
			stw.addHistory(":hweak h b tw tf")
			return nil
		}
		if narg < 5 {
			return st.NotEnoughArgs(":hweak")
		}
		al, err := st.NewHWEAK(args[1:5])
		if err != nil {
			return err
		}
		stw.ShapeData(al)
		if pipe {
			sender = []interface{}{al}
		}
	case "rpipe":
		if usage {
			stw.addHistory(":rpipe h b tw tf")
			return nil
		}
		if narg < 5 {
			return st.NotEnoughArgs(":rpipe")
		}
		al, err := st.NewRPIPE(args[1:5])
		if err != nil {
			return err
		}
		stw.ShapeData(al)
		if pipe {
			sender = []interface{}{al}
		}
	case "cpipe":
		if usage {
			stw.addHistory(":cpipe d t")
			return nil
		}
		if narg < 3 {
			return st.NotEnoughArgs(":cpipe")
		}
		al, err := st.NewCPIPE(args[1:3])
		if err != nil {
			return err
		}
		stw.ShapeData(al)
		if pipe {
			sender = []interface{}{al}
		}
	case "tkyou":
		if usage {
			stw.addHistory(":tkyou h b tw tf")
			return nil
		}
		if narg < 5 {
			return st.NotEnoughArgs(":tkyou")
		}
		al, err := st.NewTKYOU(args[1:5])
		if err != nil {
			return err
		}
		stw.ShapeData(al)
		if pipe {
			sender = []interface{}{al}
		}
	case "ckyou":
		if usage {
			stw.addHistory(":ckyou h b tw tf")
			return nil
		}
		if narg < 5 {
			return st.NotEnoughArgs(":ckyou")
		}
		al, err := st.NewCKYOU(args[1:5])
		if err != nil {
			return err
		}
		stw.ShapeData(al)
		if pipe {
			sender = []interface{}{al}
		}
	case "plate":
		if usage {
			stw.addHistory(":plate h b")
			return nil
		}
		if narg < 3 {
			return st.NotEnoughArgs(":plate")
		}
		al, err := st.NewPLATE(args[1:3])
		if err != nil {
			return err
		}
		stw.ShapeData(al)
		if pipe {
			sender = []interface{}{al}
		}
	case "fixrotate":
		fixRotate = !fixRotate
	case "fixmove":
		fixMove = !fixMove
	case "noundo":
		NOUNDO = true
		stw.addHistory("undo/redo is off")
	case "undo":
		NOUNDO = false
		stw.Snapshot()
		stw.addHistory("undo/redo is on")
	case "alt":
		ALTSELECTNODE = !ALTSELECTNODE
		if ALTSELECTNODE {
			stw.addHistory("select node with Alt key")
		} else {
			stw.addHistory("select elem with Alt key")
		}
	case "procs":
		if usage {
			stw.addHistory(":procs numcpu")
			return nil
		}
		if narg < 2 {
			current := runtime.GOMAXPROCS(-1)
			stw.addHistory(fmt.Sprintf("PROCS: %d", current))
			return nil
		}
		tmp, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return err
		}
		val := int(tmp)
		if 1 <= val && val <= runtime.NumCPU() {
			old := runtime.GOMAXPROCS(val)
			stw.addHistory(fmt.Sprintf("PROCS: %d -> %d", old, val))
		}
	case "empty":
		if usage {
			stw.addHistory(":empty")
			return nil
		}
		stw.emptyexmodech()
	}
	if evaluated {
		return nil
	}
	if stw.Frame == nil {
		stw.errormessage(errors.New("frame is nil"), INFO)
		return nil
	}
	switch cname {
	default:
		stw.errormessage(errors.New(fmt.Sprintf("no exmode command: %s", cname)), INFO)
		return nil
	case "write":
		if usage {
			stw.addHistory(":write")
			return nil
		}
		if fn == "" {
			stw.SaveFile(stw.Frame.Path)
		} else {
			if bang || (!st.FileExists(fn) || stw.Yn("Save", "上書きしますか")) {
				err := stw.SaveFile(fn)
				if err != nil {
					return err
				}
				if fn != stw.Frame.Path {
					stw.Copylsts(fn)
				}
			}
		}
	case "save":
		if usage {
			stw.addHistory(":save filename {-u=.strc}")
			return nil
		}
		if fn == "" {
			return st.NotEnoughArgs(":save")
		}
		if bang || (!st.FileExists(fn) || stw.Yn("Save", "上書きしますか")) {
			if _, ok := argdict["MKDIR"]; ok {
				os.MkdirAll(filepath.Dir(fn), 0644)
			}
			var err error
			readrc := true
			if rc, ok := argdict["U"]; ok {
				if rc == "NONE" || rc == "" {
					readrc = false
				}
			}
			if stw.SelectElem != nil && len(stw.SelectElem) > 0 {
				err = stw.SaveFileSelected(fn, stw.SelectElem)
				if err != nil {
					return err
				}
				stw.Deselect()
				err = stw.OpenFile(fn, readrc)
				if err != nil {
					return err
				}
				stw.Copylsts(fn)
			} else {
				err = stw.SaveFile(fn)
			}
			if err != nil {
				return err
			}
			if fn != stw.Frame.Path {
				stw.Copylsts(fn)
			}
			stw.Rebase(fn)
		}
	case "increment":
		if usage {
			stw.addHistory(":increment {times:1}")
			return nil
		}
		if !bang && stw.Changed {
			switch stw.Yna("CHANGED", "変更を保存しますか", "キャンセル") {
			case 1:
				stw.SaveAS()
			case 2:
				stw.addHistory("not saved")
			case 3:
				return errors.New(":inc cancelled")
			}
		}
		var times int
		if narg >= 2 {
			val, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}
			times = int(val)
		} else {
			times = 1
		}
		fn, err := st.Increment(stw.Frame.Path, "_", 1, times)
		if err != nil {
			return err
		}
		if bang || (!st.FileExists(fn) || stw.Yn("Save", "上書きしますか")) {
			err := stw.SaveFile(fn)
			if err != nil {
				return err
			}
			if fn != stw.Frame.Path {
				stw.Copylsts(fn)
			}
			stw.Rebase(fn)
			stw.Snapshot()
			stw.EditReadme(filepath.Dir(fn))
		}
	case "tag":
		if usage {
			stw.addHistory(":tag name")
			return nil
		}
		if narg < 2 {
			return st.NotEnoughArgs(":tag")
		}
		name := args[1]
		if !bang {
			if _, exists := stw.taggedFrame[name]; exists {
				return errors.New(fmt.Sprintf("tag %s already exists", name))
			}
		}
		stw.taggedFrame[name] = stw.Frame.Snapshot()
	case "checkout":
		if usage {
			stw.addHistory(":checkout name")
			return nil
		}
		if narg < 2 {
			return st.NotEnoughArgs(":checkout")
		}
		name := args[1]
		if f, exists := stw.taggedFrame[name]; exists {
			stw.Frame = f
		} else {
			return errors.New(fmt.Sprintf("tag %s doesn't exist", name))
		}
	case "read":
		if usage {
			stw.addHistory(":read filename")
			stw.addHistory(":read type filename")
			return nil
		}
		if narg < 2 {
			return st.NotEnoughArgs(":read")
		}
		t := strings.ToLower(args[1])
		if narg < 3 {
			switch t {
			case "$all":
				stw.ReadAll()
			case "$data":
				for _, ext := range []string{".inl", ".ihx", ".ihy"} {
					err := stw.Frame.ReadData(st.Ce(stw.Frame.Path, ext))
					if err != nil {
						stw.errormessage(err, ERROR)
					}
				}
			case "$results":
				mode := st.UpdateResult
				if _, ok := argdict["ADD"]; ok {
					mode = st.AddResult
					if _, ok2 := argdict["SEARCH"]; ok2 {
						mode = st.AddSearchResult
					}
				}
				for _, ext := range []string{".otl", ".ohx", ".ohy"} {
					err := stw.Frame.ReadResult(st.Ce(stw.Frame.Path, ext), uint(mode))
					if err != nil {
						stw.errormessage(err, ERROR)
					}
				}
			default:
				err := stw.ReadFile(fn)
				if err != nil {
					return err
				}
			}
			return nil
		}
		fn = stw.Complete(args[2])
		if filepath.Dir(fn) == "." {
			fn = filepath.Join(stw.Cwd, fn)
		}
		switch {
		case abbrev.For("d/ata", t):
			err := stw.Frame.ReadData(fn)
			if err != nil {
				return err
			}
		case abbrev.For("r/esult", t):
			mode := st.UpdateResult
			if _, ok := argdict["ADD"]; ok {
				mode = st.AddResult
				if _, ok2 := argdict["SEARCH"]; ok2 {
					mode = st.AddSearchResult
				}
			}
			err := stw.Frame.ReadResult(fn, uint(mode))
			if err != nil {
				return err
			}
		case abbrev.For("s/rcan", t):
			err := stw.Frame.ReadRat(fn)
			if err != nil {
				return err
			}
		case abbrev.For("l/ist", t):
			err := stw.Frame.ReadLst(fn)
			if err != nil {
				return err
			}
		case abbrev.For("w/eight", t):
			err := stw.Frame.ReadWgt(fn)
			if err != nil {
				return err
			}
		case abbrev.For("k/ijun", t):
			err := stw.Frame.ReadKjn(fn)
			if err != nil {
				return err
			}
		case abbrev.For("b/uckling", t):
			err := stw.Frame.ReadBuckling(fn)
			if err != nil {
				return err
			}
		case abbrev.For("z/oubun", t):
			err := stw.Frame.ReadZoubun(fn)
			if err != nil {
				return err
			}
		case t == "pgp":
			al := make(map[string]*Command, 0)
			err := ReadPgp(fn, al)
			if err != nil {
				return err
			}
			aliases = al
		}
	case "insert":
		if usage {
			stw.addHistory(":insert filename angle(deg)")
			return nil
		}
		if narg > 2 && len(stw.SelectNode) >= 1 {
			angle, err := strconv.ParseFloat(args[2], 64)
			if err != nil {
				return err
			}
			err = stw.Frame.ReadInp(fn, stw.SelectNode[0].Coord, angle*math.Pi/180.0, false)
			stw.Snapshot()
			if err != nil {
				return err
			}
			stw.EscapeAll()
		}
	case "propsect":
		if usage {
			stw.addHistory(":propsect filename")
			return nil
		}
		err := stw.AddPropAndSect(fn)
		stw.Snapshot()
		if err != nil {
			return err
		}
	case "writeoutput":
		if usage {
			stw.addHistory(":writeoutput filename period")
			return nil
		}
		if narg < 3 {
			return st.NotEnoughArgs(":wo")
		}
		var err error
		period := strings.ToUpper(args[2])
		if stw.SelectElem != nil && len(stw.SelectElem) > 0 {
			err = st.WriteOutput(fn, period, stw.SelectElem)
		} else {
			err = stw.Frame.WriteOutput(fn, period)
		}
		if err != nil {
			return err
		}
	case "writereaction":
		if usage {
			stw.addHistory(":writereaction filename direction")
			return nil
		}
		if narg < 3 {
			return st.NotEnoughArgs(":wr")
		}
		tmp, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			return err
		}
		if _, ok := argdict["CONFED"]; ok {
			selectconfed(stw)
		}
		if stw.SelectNode == nil || len(stw.SelectNode) == 0 {
			return errors.New(":writereaction: no selected node")
		}
		sort.Sort(st.NodeByNum{stw.SelectNode})
		err = st.WriteReaction(fn, stw.SelectNode, int(tmp))
		if err != nil {
			return err
		}
	case "zoubundisp":
		if usage {
			stw.addHistory(":zoubundisp period direction")
			return nil
		}
		if narg < 3 {
			return st.NotEnoughArgs(":zoubundisp")
		}
		if stw.SelectNode == nil || len(stw.SelectNode) == 0 {
			return st.NotEnoughArgs(":zoubundisp no selected node")
		}
		pers := []string{args[1]}
		val, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			return errors.New(":zoubundisp unknown direction")
		}
		d := int(val)
		if d < 0 || d > 5 {
			return errors.New(":zoubundisp direction should be between 0 ~ 6")
		}
		fn := filepath.Join(filepath.Dir(stw.Frame.Path), "zoubunout.txt")
		err = stw.Frame.ReportZoubunDisp(fn, stw.SelectNode, pers, d)
		if err != nil {
			return err
		}
	case "zoubunreaction":
		if usage {
			stw.addHistory(":zoubunreaction period direction")
			return nil
		}
		if narg < 3 {
			return st.NotEnoughArgs(":zoubunreaction")
		}
		if stw.SelectNode == nil || len(stw.SelectNode) == 0 {
			return st.NotEnoughArgs(":zoubunreaction no selected node")
		}
		pers := []string{args[1]}
		val, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			return errors.New(":zoubunreaction unknown direction")
		}
		d := int(val)
		if d < 0 || d > 5 {
			return errors.New(":zoubunreaction direction should be between 0 ~ 6")
		}
		fn := filepath.Join(filepath.Dir(stw.Frame.Path), "zoubunout.txt")
		err = stw.Frame.ReportZoubunReaction(fn, stw.SelectNode, pers, d)
		if err != nil {
			return err
		}
	case "hardcopy":
		if usage {
			stw.addHistory(":hardcopy")
			return nil
		}
		stw.Print()
	case "fig2":
		if usage {
			stw.addHistory(":fig2 filename")
			return nil
		}
		err := stw.ReadFig2(fn)
		if err != nil {
			return err
		}
	case "svg":
		if usage {
			stw.addHistory(":svg filename")
			return nil
		}
		err := stw.PrintSVG(fn)
		if err != nil {
			return err
		}
	case "check":
		if usage {
			stw.addHistory(":check")
			return nil
		}
		checkframe(stw)
		stw.addHistory("CHECKED")
	case "elemduplication":
		if usage {
			stw.addHistory(":elemduplication {-ignoresect=code}")
			return nil
		}
		stw.Deselect()
		var isect []int
		if isec, ok := argdict["IGNORESECT"]; ok {
			if isec == "" {
				isect = nil
			} else {
				isect = SplitNums(isec)
				stw.addHistory(fmt.Sprintf("IGNORE SECT: %v", isect))
			}
		} else {
			isect = nil
		}
		els := stw.Frame.ElemDuplication(isect)
		if len(els) != 0 {
			enum := 0
			for k := range els {
				stw.SelectElem = append(stw.SelectElem, k)
				enum++
			}
			stw.SelectElem = stw.SelectElem[:enum]
		}
	case "intersectall":
		if usage {
			stw.addHistory(":intersectall")
			return nil
		}
		l := len(stw.SelectElem)
		if l <= 1 {
			return nil
		}
		go func() {
			err := stw.Frame.IntersectAll(stw.SelectElem, EPS)
			stw.Frame.Endch <- err
		}()
		stw.CurrentLap("Calculating...", 0, l)
		go func() {
			var err error
			var nlap int
		iallloop:
			for {
				select {
				case nlap = <-stw.Frame.Lapch:
					stw.CurrentLap("Calculating...", nlap, l)
					stw.Redraw()
				case err = <-stw.Frame.Endch:
					if err != nil {
						stw.CurrentLap("Error", nlap, l)
						stw.errormessage(err, ERROR)
					} else {
						stw.CurrentLap("Completed", nlap, l)
					}
					stw.Redraw()
					break iallloop
				}
			}
		}()
		stw.Snapshot()
	case "srcal":
		if usage {
			stw.addHistory(":srcal {-fbold} {-noreload} {-tmp}")
			return nil
		}
		cond := st.NewCondition()
		if _, ok := argdict["FBOLD"]; ok {
			stw.addHistory("Fb: old")
			cond.FbOld = true
		}
		reload := true
		if _, ok := argdict["NORELOAD"]; ok {
			reload = false
		}
		if reload {
			stw.ReadFile(st.Ce(stw.Frame.Path, ".lst"))
		}
		otp := stw.Frame.Path
		if _, ok := argdict["TMP"]; ok {
			otp = "tmp"
		}
		stw.Frame.SectionRateCalculation(otp, "L", "X", "X", "Y", "Y", -1.0, cond)
	case "nminteraction":
		if usage {
			stw.addHistory(":nminteraction sectcode")
			return nil
		}
		if narg < 2 {
			return st.NotEnoughArgs(":nminteraction")
		}
		tmp, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return err
		}
		if al, ok := stw.Frame.Allows[int(tmp)]; ok {
			var otp bytes.Buffer
			cond := st.NewCondition()
			ndiv := 100
			if nd, ok := argdict["NDIV"]; ok {
				if nd != "" {
					stw.addHistory(fmt.Sprintf("NDIV: %s", nd))
					tmp, err := strconv.ParseInt(nd, 10, 64)
					if err == nil {
						ndiv = int(tmp)
					}
				}
			}
			filename := "nmi.txt"
			if o, ok := argdict["OUTPUT"]; ok {
				if o != "" {
					stw.addHistory(fmt.Sprintf("OUTPUT: %s", o))
					filename = o
				}
			}
			switch al.(type) {
			default:
				return nil
			case *st.RCColumn:
				nmax := al.(*st.RCColumn).Nmax(cond)
				nmin := al.(*st.RCColumn).Nmin(cond)
				for i := 0; i <= ndiv; i++ {
					cond.N = nmax - float64(i)*(nmax-nmin)/float64(ndiv)
					otp.WriteString(fmt.Sprintf("%.5f %.5f\n", cond.N, al.Ma(cond)))
				}
			}
			w, err := os.Create(filename)
			defer w.Close()
			if err != nil {
				return err
			}
			otp.WriteTo(w)
		}
	case "gohanlst":
		if usage {
			stw.addHistory(":gohanlst factor sectcode...")
			return nil
		}
		if narg < 3 {
			return st.NotEnoughArgs(":gohanlst")
		}
		val, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return err
		}
		sects := SplitNums(strings.Join(args[2:], " "))
		var otp bytes.Buffer
		var etype string
		for _, snum := range sects {
			if sec, ok := stw.Frame.Sects[snum]; ok {
				for _, s := range sec.BraceSection() {
					if s.Type == 5 {
						etype = "WALL"
					} else if s.Type == 6 {
						etype = "SLAB"
					}
					otp.WriteString(fmt.Sprintf("CODE %4d WOOD %s                                                \"%s(%3d)\"\n", s.Num, etype, etype[:1], snum))
					otp.WriteString(fmt.Sprintf("         THICK %5.3f       GOHAN                                     \"x%3.1f\"\n\n", val/12.0, val)) // 2[kgf/cm] / 24[kgf/cm2] = 1/12[cm]
				}
			}
		}
		w, err := os.Create(filepath.Join(stw.Cwd, "gohan.lst"))
		defer w.Close()
		if err != nil {
			return err
		}
		otp = st.AddCR(otp)
		otp.WriteTo(w)
	case "kaberyo":
		if usage {
			stw.addHistory(":kaberyo")
			return nil
		}
		var els []*st.Elem
		if stw.SelectElem == nil || len(stw.SelectElem) == 0 {
			enum := 0
			els = make([]*st.Elem, 0)
		ex_kaberyo:
			for {
				select {
				case <-time.After(time.Second):
					break ex_kaberyo
				case <-stw.exmodeend:
					break ex_kaberyo
				case el := <-stw.exmodech:
					if el, ok := el.(*st.Elem); ok {
						els = append(els, el)
						enum++
					}
				}
			}
			if enum == 0 {
				return errors.New(":kaberyo no selected elem")
			}
			els = els[:enum]
		} else {
			els = stw.SelectElem
		}
		var props []int
		if val, ok := argdict["HALF"]; ok {
			props = SplitNums(val)
			stw.addHistory(fmt.Sprintf("HALF: %v", props))
		}
		alpha := math.Sqrt(24.0 / 18.0)
		if val, ok := argdict["FC"]; ok {
			fc, err := strconv.ParseFloat(val, 64)
			if err == nil {
				alpha = math.Min(math.Sqrt2, math.Sqrt(fc/18.0))
			}
		}
		if val, ok := argdict["ALPHA"]; ok {
			a, err := strconv.ParseFloat(val, 64)
			if err == nil {
				alpha = math.Min(math.Sqrt2, a)
			}
		}
		stw.addHistory(fmt.Sprintf("ALPHA: %.3f", alpha))
		ccol := 7.0
		cwall := 25.0
		if r, ok := argdict["ROUTE"]; ok {
			switch strings.ToLower(r) {
			case "1", "2-1", "2_1", "2.1":
				ccol = 7.0
				cwall = 25.0
			case "2-2", "2_2", "2.2":
				ccol = 18.0
				cwall = 18.0
			}
		}
		stw.addHistory(fmt.Sprintf("COEFFICIENT: COLUMN=%.1f WALL=%.1f", ccol, cwall))
		sumcol := 0.0
		sumwall := 0.0
		for _, el := range els {
			if !el.Sect.IsRc(EPS) {
				continue
			}
			if el.IsLineElem() {
				a, err := el.Sect.Area(0)
				if err != nil {
					continue
				}
				for _, p := range props {
					if el.Sect.Figs[0].Prop.Num == p {
						a *= 0.5
						break
					}
				}
				sumcol += a
			} else {
				t, err := el.Sect.Thick(0)
				if err != nil {
					continue
				}
				sumwall += t * el.EffectiveWidth()
			}
		}
		total := alpha * (ccol*sumcol + cwall*sumwall)
		stw.addHistory(fmt.Sprintf("COLUMN: %.3f WALL: %.3f TOTAL: %.3f", sumcol, sumwall, total))
	case "facts":
		if usage {
			stw.addHistory(":facts {-skipany=code} {-skipall=code}")
			return nil
		}
		fn = st.Ce(stw.Frame.Path, ".fes")
		var skipany, skipall []int
		if sany, ok := argdict["SKIPANY"]; ok {
			if sany == "" {
				skipany = nil
			} else {
				skipany = SplitNums(sany)
				stw.addHistory(fmt.Sprintf("SKIP ANY: %v", skipany))
			}
		} else {
			skipany = nil
		}
		if sall, ok := argdict["SKIPALL"]; ok {
			if sall == "" {
				skipall = nil
			} else {
				skipall = SplitNums(sall)
				stw.addHistory(fmt.Sprintf("SKIP ALL: %v", skipall))
			}
		} else {
			skipall = nil
		}
		err := stw.Frame.Facts(fn, []int{st.COLUMN, st.GIRDER, st.BRACE, st.WBRACE, st.SBRACE}, skipany, skipall)
		if err != nil {
			return err
		}
		stw.addHistory(fmt.Sprintf("Output: %s", fn))
	case "amountprop":
		if usage {
			stw.addHistory(":amountprop propcode")
			return nil
		}
		if narg < 2 {
			return st.NotEnoughArgs(":amountprop")
		}
		props := SplitNums(strings.Join(args[1:], " "))
		if len(props) == 0 {
			return errors.New(":amountprop: no selected prop")
		}
		fn := filepath.Join(filepath.Dir(stw.Frame.Path), "amount.txt")
		err := stw.Frame.AmountProp(fn, props...)
		if err != nil {
			return err
		}
	case "amountlst":
		if usage {
			stw.addHistory(":amountlst propcode")
			return nil
		}
		var sects []int
		if narg < 2 {
			if _, ok := argdict["ALL"]; ok {
				sects = make([]int, len(stw.Frame.Sects))
				i := 0
				for _, sec := range stw.Frame.Sects {
					if sec.Num >= 900 {
						continue
					}
					sects[i] = sec.Num
					i++
				}
				sects = sects[:i]
				sort.Ints(sects)
			} else {
				return st.NotEnoughArgs(":amountlst")
			}
		}
		if sects == nil {
			sects = SplitNums(strings.Join(args[1:], " "))
		}
		if len(sects) == 0 {
			return errors.New(":amountlst: no selected sect")
		}
		fn := filepath.Join(filepath.Dir(stw.Frame.Path), "amountlst.txt")
		err := stw.Frame.AmountLst(fn, sects...)
		if err != nil {
			return err
		}
	case "node":
		if usage {
			stw.addHistory(":node nnum")
			stw.addHistory(":node [x,y,z] [>,<,=] coord")
			stw.addHistory(":node {confed/pinned/fixed/free}")
			stw.addHistory(":node pile num")
			return nil
		}
		stw.Deselect()
		var f func(*st.Node) bool
		if narg >= 2 {
			condition := strings.ToUpper(strings.Join(args[1:], " "))
			coordstr := regexp.MustCompile("^ *([XYZ]) *([<!=>]{0,2}) *([0-9.]+)")
			numstr := regexp.MustCompile("^[0-9, ]+$")
			pilestr := regexp.MustCompile("^ *PILE *([0-9, ]+)$")
			switch {
			default:
				return errors.New(":node: unknown format")
			case numstr.MatchString(condition):
				nnums := SplitNums(condition)
				stw.SelectNode = make([]*st.Node, len(nnums))
				nods := 0
				for i, nnum := range nnums {
					if n, ok := stw.Frame.Nodes[nnum]; ok {
						stw.SelectNode[i] = n
						nods++
					}
				}
				stw.SelectNode = stw.SelectNode[:nods]
			case coordstr.MatchString(condition):
				fs := coordstr.FindStringSubmatch(condition)
				if len(fs) < 4 {
					return errors.New(":node invalid input")
				}
				var ind int
				switch fs[1] {
				case "X":
					ind = 0
				case "Y":
					ind = 1
				case "Z":
					ind = 2
				}
				val, err := strconv.ParseFloat(fs[3], 64)
				if err != nil {
					return err
				}
				switch fs[2] {
				case "", "=", "==":
					f = func(n *st.Node) bool {
						if n.Coord[ind] == val {
							return true
						}
						return false
					}
				case "!=":
					f = func(n *st.Node) bool {
						if n.Coord[ind] != val {
							return true
						}
						return false
					}
				case ">":
					f = func(n *st.Node) bool {
						if n.Coord[ind] > val {
							return true
						}
						return false
					}
				case ">=":
					f = func(n *st.Node) bool {
						if n.Coord[ind] >= val {
							return true
						}
						return false
					}
				case "<":
					f = func(n *st.Node) bool {
						if n.Coord[ind] < val {
							return true
						}
						return false
					}
				case "<=":
					f = func(n *st.Node) bool {
						if n.Coord[ind] <= val {
							return true
						}
						return false
					}
				}
			case pilestr.MatchString(condition):
				fs := pilestr.FindStringSubmatch(condition)
				pnums := SplitNums(fs[1])
				f = func(n *st.Node) bool {
					if n.Pile == nil {
						return false
					}
					for _, pnum := range pnums {
						if n.Pile.Num == pnum {
							return true
						}
					}
					return false
				}
			case abbrev.For("CONF/ED", condition):
				f = func(n *st.Node) bool {
					for i := 0; i < 6; i++ {
						if n.Conf[i] {
							return true
						}
					}
					return false
				}
			case condition == "FREE":
				f = func(n *st.Node) bool {
					for i := 0; i < 6; i++ {
						if n.Conf[i] {
							return false
						}
					}
					return true
				}
			case abbrev.For("PIN/NED", condition):
				f = func(n *st.Node) bool {
					return n.IsPinned()
				}
			case abbrev.For("FIX/ED", condition):
				f = func(n *st.Node) bool {
					return n.IsFixed()
				}
			}
			if f != nil {
				stw.SelectNode = make([]*st.Node, len(stw.Frame.Nodes))
				num := 0
				for _, n := range stw.Frame.Nodes {
					if f(n) {
						stw.SelectNode[num] = n
						num++
					}
				}
				stw.SelectNode = stw.SelectNode[:num]
			}
		} else {
			stw.SelectNode = make([]*st.Node, len(stw.Frame.Nodes))
			num := 0
			for _, n := range stw.Frame.Nodes {
				stw.SelectNode[num] = n
				num++
			}
			stw.SelectNode = stw.SelectNode[:num]
		}
		if pipe {
			num := len(stw.SelectNode)
			sender = make([]interface{}, num)
			for i:=0; i<num; i++ {
				sender[i] = stw.SelectNode[i]
			}
		}
	case "conf":
		if usage {
			stw.addHistory(":conf [0,1]{6}")
			return nil
		}
		lis := make([]bool, 6)
		if len(args[1]) >= 6 {
			for i := 0; i < 6; i++ {
				switch args[1][i] {
				default:
					lis[i] = false
				case '0':
					lis[i] = false
				case '1':
					lis[i] = true
				case '_':
					continue
				case 't':
					lis[i] = !lis[i]
				}
			}
			setconf(stw, lis)
		} else {
			return st.NotEnoughArgs(":conf")
		}
	case "pile":
		if usage {
			stw.addHistory(":pile pilecode")
			return nil
		}
		if stw.SelectNode == nil || len(stw.SelectNode) == 0 {
			return errors.New(":pile no selected node")
		}
		if narg < 2 {
			for _, n := range stw.SelectNode {
				n.Pile = nil
			}
			break
		}
		val, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return err
		}
		if p, ok := stw.Frame.Piles[int(val)]; ok {
			for _, n := range stw.SelectNode {
				n.Pile = p
			}
			stw.Snapshot()
		} else {
			return errors.New(fmt.Sprintf(":pile PILE %d doesn't exist", val))
		}
	case "xscale":
		if usage {
			stw.addHistory(":xscale factor coord")
			return nil
		}
		if stw.SelectNode == nil || len(stw.SelectNode) == 0 {
			return errors.New(":xscale no selected node")
		}
		if narg < 3 {
			return st.NotEnoughArgs(":xscale")
		}
		factor, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return err
		}
		coord, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			return err
		}
		for _, n := range stw.SelectNode {
			if n == nil {
				continue
			}
			n.Scale([]float64{coord, 0.0, 0.0}, factor, 1.0, 1.0)
		}
		stw.Snapshot()
	case "yscale":
		if usage {
			stw.addHistory(":yscale factor coord")
			return nil
		}
		if stw.SelectNode == nil || len(stw.SelectNode) == 0 {
			return errors.New(":yscale no selected node")
		}
		if narg < 3 {
			return st.NotEnoughArgs(":yscale")
		}
		factor, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return err
		}
		coord, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			return err
		}
		for _, n := range stw.SelectNode {
			if n == nil {
				continue
			}
			n.Scale([]float64{0.0, coord, 0.0}, 1.0, factor, 1.0)
		}
		stw.Snapshot()
	case "zscale":
		if usage {
			stw.addHistory(":zscale factor coord")
			return nil
		}
		if stw.SelectNode == nil || len(stw.SelectNode) == 0 {
			return errors.New(":zscale no selected node")
		}
		if narg < 3 {
			return st.NotEnoughArgs(":zscale")
		}
		factor, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return err
		}
		coord, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			return err
		}
		for _, n := range stw.SelectNode {
			if n == nil {
				continue
			}
			n.Scale([]float64{0.0, 0.0, coord}, 1.0, 1.0, factor)
		}
		stw.Snapshot()
	case "pload":
		if usage {
			stw.addHistory(":pload position value")
			return nil
		}
		if stw.SelectNode == nil || len(stw.SelectNode) == 0 {
			return errors.New(":pload no selected node")
		}
		if narg < 3 {
			return st.NotEnoughArgs(":pload")
		}
		ind, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return err
		}
		val, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			return err
		}
		for _, n := range stw.SelectNode {
			if n == nil {
				continue
			}
			n.Load[int(ind)] = val
		}
		stw.Snapshot()
	case "elem":
		if usage {
			stw.addHistory(":elem [elemcode,sect sectcode,etype]")
			return nil
		}
		stw.Deselect()
		var f func(*st.Elem) bool
		if narg >= 2 {
			condition := strings.ToUpper(strings.Join(args[1:], " "))
			numstr := regexp.MustCompile("^[0-9, ]+$")
			switch {
			default:
				return errors.New(":elem: unknown format")
			case numstr.MatchString(condition):
				enums := SplitNums(condition)
				stw.SelectElem = make([]*st.Elem, len(enums))
				els := 0
				for i, enum := range enums {
					if el, ok := stw.Frame.Elems[enum]; ok {
						stw.SelectElem[i] = el
						els++
					}
				}
				stw.SelectElem = stw.SelectElem[:els]
			case re_sectnum.MatchString(condition):
				f, _ = SectFilter(condition)
				if f == nil {
					return errors.New(":elem sect: format error")
				}
			case re_orgsectnum.MatchString(condition):
				f, _ = OriginalSectFilter(condition)
				if f == nil {
					return errors.New(":elem sect: format error")
				}
			case re_etype.MatchString(condition):
				f, _ = EtypeFilter(condition)
				if f == nil {
					return errors.New(":elem etype: format error")
				}
			case strings.EqualFold(condition, "curtain"):
				f = func(el *st.Elem) bool {
					if el.Sect.Num > 900 {
						return false
					}
					if el.Sect.HasArea(0) {
						return false
					}
					if !el.Sect.HasBrace() {
						return true
					}
					return false
				}
			case strings.EqualFold(condition, "isgohan"):
				f = func(el *st.Elem) bool {
					return el.Sect.IsGohan(EPS)
				}
			case strings.EqualFold(condition, "error"):
				threshold := 1.0
				if v, ok := argdict["THRESHOLD"]; ok {
					val, err := strconv.ParseFloat(v, 64)
					if err == nil {
						threshold = val
					}
				}
				stw.addHistory(fmt.Sprintf("THRESHOLD: %.3f", threshold))
				f = func(el *st.Elem) bool {
					switch el.Etype {
					case st.COLUMN, st.GIRDER, st.BRACE, st.WALL, st.SLAB:
						val, err := el.RateMax(stw.Frame.Show)
						if err != nil {
							return false
						}
						if val > threshold {
							return true
						}
					}
					return false
				}
			}
			if f != nil {
				stw.SelectElem = make([]*st.Elem, len(stw.Frame.Elems))
				num := 0
				for _, el := range stw.Frame.Elems {
					if f(el) {
						stw.SelectElem[num] = el
						num++
					}
				}
				stw.SelectElem = stw.SelectElem[:num]
			}
		} else {
			stw.SelectElem = make([]*st.Elem, len(stw.Frame.Elems))
			num := 0
			for _, el := range stw.Frame.Elems {
				stw.SelectElem[num] = el
				num++
			}
			stw.SelectElem = stw.SelectElem[:num]
		}
		if pipe {
			num := len(stw.SelectElem)
			sender = make([]interface{}, num)
			for i:=0; i<num; i++ {
				sender[i] = stw.SelectElem[i]
			}
		}
	case "fence":
		if usage {
			stw.addHistory(":fence axis coord {-plate}")
			return nil
		}
		if narg < 3 {
			return st.NotEnoughArgs(":fence")
		}
		var axis int
		var err error
		var val float64
		switch strings.ToUpper(args[1]) {
		default:
			return errors.New(":fence unknown direction")
		case "X":
			axis = 0
			val, err = strconv.ParseFloat(args[2], 64)
			if err != nil {
				return err
			}
		case "Y":
			axis = 1
			val, err = strconv.ParseFloat(args[2], 64)
			if err != nil {
				return err
			}
		case "Z":
			axis = 2
			val, err = strconv.ParseFloat(args[2], 64)
			if err != nil {
				return err
			}
		case "HEIGHT":
			axis = 2
			ind, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return err
			}
			if int(ind) > stw.Frame.Ai.Nfloor-1 {
				return errors.New(":fence height: index error")
			}
			val = stw.Frame.Ai.Boundary[int(ind)]
		}
		plate := false
		if _, ok := argdict["PLATE"]; ok {
			plate = true
		}
		stw.SelectElem = stw.Frame.Fence(axis, val, plate)
		if pipe {
			num := len(stw.SelectElem)
			sender = make([]interface{}, num)
			for i:=0; i<num; i++ {
				sender[i] = stw.SelectElem[i]
			}
		}
	case "filter":
		if usage {
			stw.addHistory(":filter condition")
			return nil
		}
		tmpels, err := stw.FilterElem(stw.SelectElem, strings.Join(args[1:], " "))
		if err != nil {
			return err
		}
		stw.SelectElem = tmpels
		if pipe {
			num := len(stw.SelectElem)
			sender = make([]interface{}, num)
			for i:=0; i<num; i++ {
				sender[i] = stw.SelectElem[i]
			}
		}
	case "bond":
		if usage {
			stw.addHistory(":bond [pin,rigid,[01_t]{6}] [upper,lower,sect sectcode]")
			return nil
		}
		if narg < 2 {
			return st.NotEnoughArgs(":bond")
		}
		var els []*st.Elem
		if stw.SelectElem == nil || len(stw.SelectElem) == 0 {
			enum := 0
			els = make([]*st.Elem, 0)
		ex_bond:
			for {
				select {
				case <-time.After(time.Second):
					break ex_bond
				case <-stw.exmodeend:
					break ex_bond
				case el := <-stw.exmodech:
					if el, ok := el.(*st.Elem); ok {
						els = append(els, el)
						enum++
					}
				}
			}
			if enum == 0 {
				return errors.New(":bond no selected elem")
			}
			els = els[:enum]
		} else {
			els = stw.SelectElem
		}
		lis := make([]bool, 6)
		pat := regexp.MustCompile("[01_t]{6}")
		switch {
		case strings.EqualFold(args[1], "pin"):
			lis[4] = true
			lis[5] = true
		case pat.MatchString(args[1]):
			for i := 0; i < 6; i++ {
				switch args[1][i] {
				default:
					lis[i] = false
				case '0':
					lis[i] = false
				case '1':
					lis[i] = true
				case '_':
					continue
				case 't':
					lis[i] = !lis[i]
				}
			}
		}
		f := func(el *st.Elem, ind int) bool {
			return true
		}
		if narg >= 3 {
			condition := strings.ToLower(strings.Join(args[2:], " "))
			switch {
			case abbrev.For("up/per", condition):
				f = func(el *st.Elem, ind int) bool {
					return el.Enod[ind].Coord[2] > el.Enod[1-ind].Coord[2]
				}
			case abbrev.For("lo/wer", condition):
				f = func(el *st.Elem, ind int) bool {
					return el.Enod[ind].Coord[2] < el.Enod[1-ind].Coord[2]
				}
			case re_sectnum.MatchString(condition):
				tmpf, _ := SectFilter(condition)
				f = func(el *st.Elem, ind int) bool {
					for _, sel := range el.Frame.SearchElem(el.Enod[ind]) {
						if sel.Num == el.Num {
							continue
						}
						if tmpf(sel) {
							return true
						}
					}
					return false
				}
			}
		}
		for _, el := range els {
			if !el.IsLineElem() {
				continue
			}
			for i := 0; i < 2; i++ {
				if !f(el, i) {
					continue
				}
				for j := 0; j < 6; j++ {
					el.Bonds[6*i+j] = lis[j]
				}
			}
		}
		stw.Snapshot()
	case "section+":
		if usage {
			stw.addHistory(":section+ value")
			return nil
		}
		if narg < 2 {
			return st.NotEnoughArgs(":section+")
		}
		tmp, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return err
		}
		if tmp == 0 {
			break
		}
		val := int(tmp)
		for _, el := range stw.SelectElem {
			if el == nil {
				continue
			}
			if sec, ok := stw.Frame.Sects[el.Sect.Num+val]; ok {
				el.Sect = sec
			}
		}
		stw.Snapshot()
	case "cang":
		if usage {
			stw.addHistory(":cang val")
			return nil
		}
		if narg < 2 {
			return st.NotEnoughArgs(":cang")
		}
		val, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return err
		}
		var els []*st.Elem
		if stw.SelectElem == nil || len(stw.SelectElem) == 0 {
			enum := 0
			els = make([]*st.Elem, 0)
		ex_cang:
			for {
				select {
				case <-time.After(time.Second):
					break ex_cang
				case <-stw.exmodeend:
					break ex_cang
				case el := <-stw.exmodech:
					if el, ok := el.(*st.Elem); ok {
						els = append(els, el)
						enum++
					}
				}
			}
			if enum == 0 {
				return errors.New(":cang no selected elem")
			}
			els = els[:enum]
		} else {
			els = stw.SelectElem
		}
		for _, el := range els {
			if !el.IsLineElem() {
				continue
			}
			el.Cang = val
			el.SetPrincipalAxis()
		}
		stw.Snapshot()
	case "axis2cang":
		if usage {
			stw.addHistory(":axis2cang n1 n2 [strong,weak]")
			return nil
		}
		if narg < 4 {
			return st.NotEnoughArgs(":axis2cang")
		}
		if stw.SelectElem == nil || len(stw.SelectElem) == 0 {
			return errors.New(":axis2cang no selected elem")
		}
		nnum1, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return err
		}
		nnum2, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			return err
		}
		var strong bool
		if strings.EqualFold(args[3], "strong") {
			strong = true
		} else if strings.EqualFold(args[3], "weak") {
			strong = false
		} else {
			return errors.New(":axis2cang: last argument must be strong or weak")
		}
		var n1, n2 *st.Node
		var found bool
		if n1, found = stw.Frame.Nodes[int(nnum1)]; !found {
			return errors.New(fmt.Sprintf(":axis2cang: NODE %d not found", nnum1))
		}
		if n2, found = stw.Frame.Nodes[int(nnum2)]; !found {
			return errors.New(fmt.Sprintf(":axis2cang: NODE %d not found", nnum2))
		}
		vec := []float64{n2.Coord[0] - n1.Coord[0], n2.Coord[1] - n1.Coord[1], n2.Coord[2] - n1.Coord[2]}
		for _, el := range stw.SelectElem {
			if el == nil || el.IsHidden(stw.Frame.Show) || el.Lock || !el.IsLineElem() {
				continue
			}
			_, err := el.AxisToCang(vec, strong)
			if err != nil {
				return err
			}
		}
		stw.Snapshot()
	case "invert":
		if usage {
			stw.addHistory(":invert")
			return nil
		}
		var els []*st.Elem
		if stw.SelectElem == nil || len(stw.SelectElem) == 0 {
			enum := 0
			els = make([]*st.Elem, 0)
		ex_invert:
			for {
				select {
				case <-time.After(time.Second):
					break ex_invert
				case <-stw.exmodeend:
					break ex_invert
				case el := <-stw.exmodech:
					if el, ok := el.(*st.Elem); ok {
						els = append(els, el)
						enum++
					}
				}
			}
			if enum == 0 {
				return errors.New(":invert no selected elem")
			}
			els = els[:enum]
		} else {
			els = stw.SelectElem
		}
		for _, el := range els {
			el.Invert()
		}
		stw.Snapshot()
	case "resultant":
		if stw.SelectElem == nil || len(stw.SelectElem) == 0 {
			return errors.New(":resultant no selected elem")
		}
		vec := make([]float64, 3)
		elems := make([]*st.Elem, len(stw.SelectElem))
		enum := 0
		for _, el := range stw.SelectElem {
			if el == nil || el.Lock || !el.IsLineElem() {
				continue
			}
			elems[enum] = el
			enum++
		}
		elems = elems[:enum]
		en, err := st.CommonEnod(elems...)
		if err != nil {
			return err
		}
		if en == nil || len(en) == 0 {
			return errors.New(":resultant no common enod")
		}
		axis := [][]float64{st.XAXIS, st.YAXIS, st.ZAXIS}
		per := stw.Frame.Show.Period
		for _, el := range elems {
			for i := 0; i < 3; i++ {
				vec[i] += el.VectorStress(per, en[0].Num, axis[i])
			}
		}
		v := 0.0
		for i := 0; i < 3; i++ {
			v += vec[i] * vec[i]
		}
		v = math.Sqrt(v)
		stw.addHistory(fmt.Sprintf("NODE: %d", en[0].Num))
		stw.addHistory(fmt.Sprintf("X: %.3f Y: %.3f Z: %.3f F: %.3f", vec[0], vec[1], vec[2], v))
	case "prestress":
		if usage {
			stw.addHistory(":prestress value")
			return nil
		}
		if narg < 2 {
			return st.NotEnoughArgs(":prestress")
		}
		var els []*st.Elem
		if stw.SelectElem == nil || len(stw.SelectElem) == 0 {
			enum := 0
			els = make([]*st.Elem, 0)
		ex_prestress:
			for {
				select {
				case <-time.After(time.Second):
					break ex_prestress
				case <-stw.exmodeend:
					break ex_prestress
				case el := <-stw.exmodech:
					if el == nil {
						break ex_prestress
					}
					if el, ok := el.(*st.Elem); ok {
						els = append(els, el)
						enum++
					}
				}
			}
			if enum == 0 {
				return errors.New(":prestress no selected elem")
			}
			els = els[:enum]
		} else {
			els = stw.SelectElem
		}
		val, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return err
		}
		for _, el := range els {
			if el == nil || el.Lock || !el.IsLineElem() {
				continue
			}
			el.Prestress = val
		}
		stw.Snapshot()
	case "thermal":
		if usage {
			stw.addHistory(":thermal tmp[℃]")
			return nil
		}
		if narg < 2 {
			return st.NotEnoughArgs(":thermal")
		}
		var els []*st.Elem
		if stw.SelectElem == nil || len(stw.SelectElem) == 0 {
			enum := 0
			els = make([]*st.Elem, 0)
		ex_thermal:
			for {
				select {
				case <-time.After(time.Second):
					break ex_thermal
				case <-stw.exmodeend:
					break ex_thermal
				case el := <-stw.exmodech:
					if el == nil {
						break ex_thermal
					}
					if el, ok := el.(*st.Elem); ok {
						els = append(els, el)
						enum++
					}
				}
			}
			if enum == 0 {
				return errors.New(":thermal no selected elem")
			}
			els = els[:enum]
		} else {
			els = stw.SelectElem
		}
		tmp, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return err
		}
		alpha := 12.0 * 1e-6
		if al, ok := argdict["ALPHA"]; ok {
			tmpal, err := strconv.ParseFloat(al, 64)
			if err == nil {
				alpha = tmpal
			}
		}
		stw.addHistory(fmt.Sprintf("ALPHA: %.3E", alpha))
		for _, el := range els {
			if el == nil || el.Lock || !el.IsLineElem() {
				continue
			}
			if len(el.Sect.Figs) == 0 {
				continue
			}
			if a, ok := el.Sect.Figs[0].Value["AREA"]; ok {
				val := el.Sect.Figs[0].Prop.E * a * alpha * tmp
				el.Cmq[0] = val
				el.Cmq[6] = -val
			}
		}
		stw.Snapshot()
	case "divide":
		if narg < 2 {
			if usage {
				stw.addHistory(":divide [mid, n, elem, ons, axis, length]")
				return nil
			}
			return st.NotEnoughArgs(":divide")
		}
		var divfunc func(*st.Elem) ([]*st.Node, []*st.Elem, error)
		switch strings.ToLower(args[1]) {
		case "mid":
			if usage {
				stw.addHistory(":divide mid")
				return nil
			}
			divfunc = func(el *st.Elem) ([]*st.Node, []*st.Elem, error) {
				return el.DivideAtMid(EPS)
			}
		case "n":
			if usage {
				stw.addHistory(":divide n div")
				return nil
			}
			if narg < 3 {
				return st.NotEnoughArgs(":divide n")
			}
			val, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return err
			}
			ndiv := int(val)
			divfunc = func(el *st.Elem) ([]*st.Node, []*st.Elem, error) {
				return el.DivideInN(ndiv, EPS)
			}
		case "elem":
			if usage {
				stw.addHistory(":divide elem (eps)")
				return nil
			}
			eps := EPS
			if narg >= 3 {
				val, err := strconv.ParseFloat(args[2], 64)
				if err == nil {
					eps = val
				}
			}
			divfunc = func(el *st.Elem) ([]*st.Node, []*st.Elem, error) {
				els, err := el.DivideAtElem(eps)
				return nil, els, err
			}
		case "ons":
			if usage {
				stw.addHistory(":divide ons (eps)")
				return nil
			}
			eps := EPS
			if narg >= 3 {
				val, err := strconv.ParseFloat(args[2], 64)
				if err == nil {
					eps = val
				}
			}
			divfunc = func(el *st.Elem) ([]*st.Node, []*st.Elem, error) {
				return el.DivideAtOns(eps)
			}
		case "axis":
			if usage {
				stw.addHistory(":divide axis [x, y, z] coord")
				return nil
			}
			if narg < 4 {
				return st.NotEnoughArgs(":divide axis")
			}
			var axis int
			switch args[2] {
			default:
				return errors.New(":divide axis: unknown axis")
			case "x", "X":
				axis = 0
			case "y", "Y":
				axis = 1
			case "z", "Z":
				axis = 2
			}
			val, err := strconv.ParseFloat(args[3], 64)
			if err != nil {
				return err
			}
			divfunc = func(el *st.Elem) ([]*st.Node, []*st.Elem, error) {
				return el.DivideAtAxis(axis, val, EPS)
			}
		case "length":
			if usage {
				stw.addHistory(":divide length l")
				return nil
			}
			if narg < 3 {
				return st.NotEnoughArgs(":divide length")
			}
			val, err := strconv.ParseFloat(args[2], 64)
			if err != nil {
				return err
			}
			divfunc = func(el *st.Elem) ([]*st.Node, []*st.Elem, error) {
				return el.DivideAtLength(val, EPS)
			}
		}
		if divfunc == nil {
			return errors.New(":divide: unknown format")
		}
		if stw.SelectElem == nil || len(stw.SelectElem) == 0 {
			return errors.New(":divide: no selected elem")
		}
		tmpels := make([]*st.Elem, 0)
		enum := 0
		for _, el := range stw.SelectElem {
			if el == nil {
				continue
			}
			_, els, err := divfunc(el)
			if err != nil {
				stw.errormessage(err, ERROR)
				continue
			}
			if err == nil && len(els) > 1 {
				tmpels = append(tmpels, els...)
				enum += len(els)
			}
		}
		stw.SelectElem = tmpels[:enum]
		stw.Snapshot()
	case "section":
		if usage {
			stw.addHistory(":section sectcode {-nodisp}")
			return nil
		}
		nodisp := false
		if _, ok := argdict["NODISP"]; ok {
			nodisp = true
		}
		if narg < 2 {
			if stw.SelectElem != nil && len(stw.SelectElem) >= 1 {
				if !nodisp {
					stw.SectionData(stw.SelectElem[0].Sect)
				}
				if pipe {
					sender = []interface{}{stw.SelectElem[0].Sect}
				}
				return nil
			}
			if t, tok := stw.TextBox["SECTION"]; tok {
				t.Value = make([]string, 0)
			}
			return nil
		}
		switch {
		case strings.EqualFold(args[1], "off"):
			if t, tok := stw.TextBox["SECTION"]; tok {
				t.Value = make([]string, 0)
			}
			return nil
		case strings.EqualFold(args[1], "curtain"):
			sects := make([]*st.Sect, len(stw.Frame.Sects))
			num := 0
			for _, sec := range stw.Frame.Sects {
				if sec.Num > 900 {
					continue
				}
				if sec.HasArea(0) {
					continue
				}
				if !sec.HasBrace() {
					sects[num] = sec
					num++
				}
			}
			if num == 0 {
				return nil
			}
			sects = sects[:num]
			if !nodisp{
				stw.SectionData(sects[0])
			}
			if pipe {
				num := len(sects)
				sender = make([]interface{}, num)
				for i:=0; i<num; i++ {
					sender[i] = sects[i]
				}
			}
		default:
			tmp, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}
			snum := int(tmp)
			if sec, ok := stw.Frame.Sects[snum]; ok {
				if narg >= 3 && args[2] == "<-" {
					select {
					case <-time.After(time.Second):
						break
					case al := <-stw.exmodech:
						if al == nil {
							break
						}
						switch al := al.(type) {
						case st.Shape:
							if sec.HasArea(0) {
								sec.Figs[0].SetShapeProperty(al)
								sec.Name = al.Description()
							}
						}
					}
				}
				if !nodisp {
					stw.SectionData(sec)
				}
				if pipe {
					sender = []interface{}{sec}
				}
			} else {
				return errors.New(fmt.Sprintf(":section SECT %d doesn't exist", snum))
			}
		}
	case "thick":
		if usage {
			stw.addHistory(":thick nfig val")
			return nil
		}
		if narg < 3 {
			return st.NotEnoughArgs(":thick")
		}
		tmp, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return err
		}
		val, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			return err
		}
		ind := int(tmp) - 1
		select {
		case <-time.After(time.Second):
			break
		case sec := <-stw.exmodech:
			if sec == nil {
				break
			}
			if sec, ok := sec.(*st.Sect); ok {
				if sec.HasThick(ind) {
					sec.Figs[ind].Value["THICK"] = val
				}
			}
		}
	case "add":
		if narg < 2 {
			if usage {
				stw.addHistory(":add [elem, sect]")
				return nil
			}
			return st.NotEnoughArgs(":add")
		}
		switch strings.ToLower(args[1]) {
		case "elem":
			if usage {
				stw.addHistory(":add elem {-sect=code} {-etype=type}")
				return nil
			}
			var etype int
			if et, ok := argdict["ETYPE"]; ok {
				switch {
				case re_column.MatchString(et):
					etype = st.COLUMN
				case re_girder.MatchString(et):
					etype = st.GIRDER
				case re_slab.MatchString(et):
					etype = st.BRACE
				case re_wall.MatchString(et):
					etype = st.WALL
				case re_slab.MatchString(et):
					etype = st.SLAB
				default:
					tmp, err := strconv.ParseInt(et, 10, 64)
					if err != nil {
						return err
					}
					etype = int(tmp)
				}
			} else {
				return errors.New(":add elem: no etype selected")
			}
			var sect *st.Sect
			if sc, ok := argdict["SECT"]; ok {
				tmp, err := strconv.ParseInt(sc, 10, 64)
				if err != nil {
					return err
				}
				if sec, ok := stw.Frame.Sects[int(tmp)]; ok {
					sect = sec
				} else {
					return errors.New(fmt.Sprintf(":add elem: SECT %d doesn't exist", tmp))
				}
			} else {
				return errors.New(":add elem: no sectcode selected")
			}
			enod := make([]*st.Node, 0)
			enods := 0
		ex_addelem:
			for {
				select {
				case <-time.After(time.Second):
					break ex_addelem
				case <-stw.exmodeend:
					break ex_addelem
				case ent := <-stw.exmodech:
					if n, ok := ent.(*st.Node); ok {
						enod = append(enod, n)
						enods++
					}
				}
			}
			enod = enod[:enods]
			switch etype {
			case st.COLUMN, st.GIRDER, st.BRACE:
				stw.Frame.AddLineElem(-1, enod[:2], sect, etype)
			case st.WALL, st.SLAB:
				if enods > 4 {
					enod = enod[:4]
				}
				stw.Frame.AddPlateElem(-1, enod, sect, etype)
			}
		case "sec", "sect":
			if usage {
				stw.addHistory(":add sect sectcode")
				return nil
			}
			if narg < 3 {
				return st.NotEnoughArgs(":add sect")
			}
			val, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return err
			}
			snum := int(val)
			if _, ok := stw.Frame.Sects[snum]; ok && !bang {
				return errors.New(fmt.Sprintf(":add sect: SECT %d already exists", snum))
			}
			sec := stw.Frame.AddSect(snum)
			select {
			case <-time.After(time.Second):
				break
			case al := <-stw.exmodech:
				if a, ok := al.(st.Shape); ok {
					sec.Figs = make([]*st.Fig, 1)
					f := st.NewFig()
					if p, ok := stw.Frame.Props[101]; ok {
						f.Prop = p
					} else {
						f.Prop = stw.Frame.DefaultProp()
					}
					sec.Figs[0] = f
					sec.Figs[0].SetShapeProperty(a)
					sec.Name = a.Description()
				}
			}
		}
	case "copy":
		if narg < 2 {
			if usage {
				stw.addHistory(":copy [sect]")
				return nil
			}
			return st.NotEnoughArgs(":copy")
		}
		switch strings.ToLower(args[1]) {
		case "sec", "sect":
			if usage {
				stw.addHistory(":copy sect sectcode")
				return nil
			}
			if narg < 3 {
				return st.NotEnoughArgs(":copy sect")
			}
			val, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return err
			}
			snum := int(val)
			if _, ok := stw.Frame.Sects[snum]; ok && !bang {
				return errors.New(fmt.Sprintf(":copy sect: SECT %d already exists", snum))
			}
			select {
			case <-time.After(time.Second):
				break
			case s := <-stw.exmodech:
				if sec, ok := s.(*st.Sect); ok {
					as := sec.Snapshot(stw.Frame)
					as.Num = snum
					stw.Frame.Sects[snum] = as
					stw.Frame.Show.Sect[snum] = true
				}
			}
		}
	case "currentvalue":
		if usage {
			stw.addHistory(":currentvalue {-abs}")
			return nil
		}
		if stw.SelectElem != nil && len(stw.SelectElem) >= 1 {
			var valfunc func(*st.Elem) float64
			if _, ok := argdict["ABS"]; ok {
				valfunc = func(elem *st.Elem) float64 {
					return elem.CurrentValue(stw.Frame.Show, true, true)
				}
			} else {
				valfunc = func(elem *st.Elem) float64 {
					return elem.CurrentValue(stw.Frame.Show, true, false)
				}
			}
			for _, el := range stw.SelectElem {
				if el == nil {
					continue
				}
				stw.addHistory(fmt.Sprintf("ELEM %d: %.3f", el.Num, valfunc(el)))
			}
		}
	case "max":
		if usage {
			stw.addHistory(":max {-abs}")
			return nil
		}
		if stw.SelectElem != nil && len(stw.SelectElem) >= 1 {
			maxval := -1e16
			var valfunc func(*st.Elem) float64
			var sel *st.Elem
			abs := false
			if _, ok := argdict["ABS"]; ok {
				valfunc = func(elem *st.Elem) float64 {
					return elem.CurrentValue(stw.Frame.Show, true, true)
				}
				abs = true
			} else {
				valfunc = func(elem *st.Elem) float64 {
					return elem.CurrentValue(stw.Frame.Show, true, false)
				}
			}
			for _, el := range stw.SelectElem {
				if el == nil {
					continue
				}
				if tmpval := valfunc(el); tmpval > maxval {
					sel = el
					maxval = tmpval
				}
			}
			if sel != nil {
				stw.SelectElem = []*st.Elem{sel}
				stw.addHistory(fmt.Sprintf("ELEM %d: %.3f", sel.Num, sel.CurrentValue(stw.Frame.Show, true, abs)))
			}
		} else if stw.SelectNode != nil && len(stw.SelectNode) >= 1 {
			maxval := -1e16
			var valfunc func(*st.Node) float64
			var sn *st.Node
			abs := false
			if _, ok := argdict["ABS"]; ok {
				valfunc = func(node *st.Node) float64 {
					return node.CurrentValue(stw.Frame.Show, true, true)
				}
				abs = true
			} else {
				valfunc = func(node *st.Node) float64 {
					return node.CurrentValue(stw.Frame.Show, true, false)
				}
			}
			for _, n := range stw.SelectNode {
				if n == nil {
					continue
				}
				if tmpval := valfunc(n); tmpval > maxval {
					sn = n
					maxval = tmpval
				}
			}
			if sn != nil {
				stw.SelectNode = []*st.Node{sn}
				stw.addHistory(fmt.Sprintf("NODE %d: %.3f", sn.Num, sn.CurrentValue(stw.Frame.Show, true, abs)))
			}
		} else {
			return errors.New(":max no selected elem/node")
		}
	case "min":
		if usage {
			stw.addHistory(":min {-abs}")
			return nil
		}
		if stw.SelectElem != nil && len(stw.SelectElem) >= 1 {
			minval := 1e16
			var valfunc func(*st.Elem) float64
			var sel *st.Elem
			abs := false
			if _, ok := argdict["ABS"]; ok {
				valfunc = func(elem *st.Elem) float64 {
					return elem.CurrentValue(stw.Frame.Show, false, true)
				}
				abs = true
			} else {
				valfunc = func(elem *st.Elem) float64 {
					return elem.CurrentValue(stw.Frame.Show, false, false)
				}
			}
			for _, el := range stw.SelectElem {
				if el == nil {
					continue
				}
				if tmpval := valfunc(el); tmpval < minval {
					sel = el
					minval = tmpval
				}
			}
			if sel != nil {
				stw.SelectElem = []*st.Elem{sel}
				stw.addHistory(fmt.Sprintf("ELEM %d: %.3f", sel.Num, sel.CurrentValue(stw.Frame.Show, false, abs)))
			}
		} else if stw.SelectNode != nil && len(stw.SelectNode) >= 1 {
			minval := 1e16
			var valfunc func(*st.Node) float64
			var sn *st.Node
			abs := false
			if _, ok := argdict["ABS"]; ok {
				valfunc = func(node *st.Node) float64 {
					return node.CurrentValue(stw.Frame.Show, false, true)
				}
				abs = true
			} else {
				valfunc = func(node *st.Node) float64 {
					return node.CurrentValue(stw.Frame.Show, false, false)
				}
			}
			for _, n := range stw.SelectNode {
				if n == nil {
					continue
				}
				if tmpval := valfunc(n); tmpval < minval {
					sn = n
					minval = tmpval
				}
			}
			if sn != nil {
				stw.SelectNode = []*st.Node{sn}
				stw.addHistory(fmt.Sprintf("NODE %d: %.3f", sn.Num, sn.CurrentValue(stw.Frame.Show, false, abs)))
			}
		} else {
			return errors.New(":min no selected elem/node")
		}
	case "average":
		if usage {
			stw.addHistory(":average {-abs}")
			return nil
		}
		if stw.SelectElem != nil && len(stw.SelectElem) >= 1 {
			var valfunc func(*st.Elem) float64
			if _, ok := argdict["ABS"]; ok {
				valfunc = func(elem *st.Elem) float64 {
					return elem.CurrentValue(stw.Frame.Show, false, true)
				}
			} else {
				valfunc = func(elem *st.Elem) float64 {
					return elem.CurrentValue(stw.Frame.Show, false, false)
				}
			}
			val := 0.0
			num := 0
			for _, el := range stw.SelectElem {
				if el == nil {
					continue
				}
				val += valfunc(el)
				num++
			}
			if num >= 1 {
				stw.addHistory(fmt.Sprintf("%d ELEMs : %.5f", num, val/float64(num)))
			}
		} else if stw.SelectNode != nil && len(stw.SelectNode) >= 1 {
			var valfunc func(*st.Node) float64
			if _, ok := argdict["ABS"]; ok {
				valfunc = func(node *st.Node) float64 {
					return node.CurrentValue(stw.Frame.Show, false, true)
				}
			} else {
				valfunc = func(node *st.Node) float64 {
					return node.CurrentValue(stw.Frame.Show, false, false)
				}
			}
			val := 0.0
			num := 0
			for _, n := range stw.SelectNode {
				if n == nil {
					continue
				}
				val += valfunc(n)
				num++
			}
			if num >= 1 {
				stw.addHistory(fmt.Sprintf("%d NODEs: %.5f", num, val/float64(num)))
			}
		} else {
			return errors.New(":average no selected elem/node")
		}
	case "sum":
		if usage {
			stw.addHistory(":sum {-abs}")
			return nil
		}
		if stw.SelectElem != nil && len(stw.SelectElem) >= 1 {
			var valfunc func(*st.Elem) float64
			if _, ok := argdict["ABS"]; ok {
				valfunc = func(elem *st.Elem) float64 {
					return elem.CurrentValue(stw.Frame.Show, false, true)
				}
			} else {
				valfunc = func(elem *st.Elem) float64 {
					return elem.CurrentValue(stw.Frame.Show, false, false)
				}
			}
			val := 0.0
			num := 0
			for _, el := range stw.SelectElem {
				if el == nil {
					continue
				}
				val += valfunc(el)
				num++
			}
			if num >= 1 {
				stw.addHistory(fmt.Sprintf("%d ELEMs : %.5f", num, val))
			}
		} else if stw.SelectNode != nil && len(stw.SelectNode) >= 1 {
			var valfunc func(*st.Node) float64
			if _, ok := argdict["ABS"]; ok {
				valfunc = func(node *st.Node) float64 {
					return node.CurrentValue(stw.Frame.Show, false, true)
				}
			} else {
				valfunc = func(node *st.Node) float64 {
					return node.CurrentValue(stw.Frame.Show, false, false)
				}
			}
			val := 0.0
			num := 0
			for _, n := range stw.SelectNode {
				if n == nil {
					continue
				}
				val += valfunc(n)
				num++
			}
			if num >= 1 {
				stw.addHistory(fmt.Sprintf("%d NODEs: %.5f", num, val))
			}
		} else {
			return errors.New(":sum no selected elem/node")
		}
	case "erase":
		if usage {
			stw.addHistory(":erase")
			return nil
		}
		stw.Deselect()
	ex_erase:
		for {
			select {
			case <-time.After(time.Second):
				break ex_erase
			case <-stw.exmodeend:
				break ex_erase
			case ent := <-stw.exmodech:
				switch ent := ent.(type) {
				case *st.Node:
					stw.Frame.DeleteNode(ent.Num)
				case *st.Elem:
					stw.Frame.DeleteElem(ent.Num)
				}
			}
		}
		ns := stw.Frame.NodeNoReference()
		if len(ns) != 0 {
			for _, n := range ns {
				stw.Frame.DeleteNode(n.Num)
			}
		}
		stw.Snapshot()
	case "count":
		if usage {
			stw.addHistory(":count")
			return nil
		}
		var nnode, nelem int
	ex_count:
		for {
			select {
			case <-time.After(time.Second):
				break ex_count
			case <-stw.exmodeend:
				break ex_count
			case ent := <-stw.exmodech:
				switch ent.(type) {
				case *st.Node:
					nnode++
				case *st.Elem:
					nelem++
				}
			}
		}
		stw.addHistory(fmt.Sprintf("NODES: %d, ELEMS: %d", nnode, nelem))
	case "show":
		if usage {
			stw.addHistory(":show")
			return nil
		}
	ex_show:
		for {
			select {
			case <-time.After(time.Second):
				break ex_show
			case <-stw.exmodeend:
				break ex_show
			case ent := <-stw.exmodech:
				if h, ok := ent.(st.Hider); ok {
					h.Show()
				}
			}
		}
	case "hide":
		if usage {
			stw.addHistory(":hide")
			return nil
		}
	ex_hide:
		for {
			select {
			case <-time.After(time.Second):
				break ex_hide
			case <-stw.exmodeend:
				break ex_hide
			case ent := <-stw.exmodech:
				if h, ok := ent.(st.Hider); ok {
					h.Hide()
				}
			}
		}
	case "height":
		if usage {
			stw.addHistory(":height f1 f2")
			return nil
		}
		if narg == 1 {
			axisrange(stw, 2, -100.0, 1000.0, false)
			return nil
		}
		if narg < 3 {
			return st.NotEnoughArgs(":height")
		}
		var min, max int
		if strings.EqualFold(args[1], "n") {
			min = stw.Frame.Ai.Nfloor
		} else {
			tmp, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}
			min = int(tmp)
		}
		if strings.EqualFold(args[2], "n") {
			max = stw.Frame.Ai.Nfloor
		} else {
			tmp, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return err
			}
			max = int(tmp)
		}
		l := len(stw.Frame.Ai.Boundary)
		if min < 0 || min >= l || max < 0 || max >= l {
			return errors.New(":height out of boundary")
		}
		axisrange(stw, 2, stw.Frame.Ai.Boundary[min], stw.Frame.Ai.Boundary[max], false)
	case "story", "storey":
		if usage {
			stw.addHistory(":storey n")
			return nil
		}
		if narg < 2 {
			return st.NotEnoughArgs(":storey")
		}
		tmp, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return err
		}
		n := int(tmp)
		if n <= 0 || n >= len(stw.Frame.Ai.Boundary) - 1 {
			return errors.New(":storey out of boundary")
		}
		return stw.exmode(fmt.Sprintf("height %d %d", n-1, n+1))
	case "floor":
		if usage {
			stw.addHistory(":floor n")
			return nil
		}
		if narg < 2 {
			return st.NotEnoughArgs(":floor")
		}
		var n int
		switch strings.ToLower(args[1]) {
		case "g":
			n = 1
		case "r":
			n = len(stw.Frame.Ai.Boundary) - 1
		default:
			tmp, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}
			n = int(tmp)
			if n <= 0 || n >= len(stw.Frame.Ai.Boundary) {
				return errors.New(":floor out of boundary")
			}
		}
		return stw.exmode(fmt.Sprintf("height %d %d", n-1, n))
	case "height+":
		stw.NextFloor()
	case "height-":
		stw.PrevFloor()
	case "view":
		if usage {
			stw.addHistory(":view [top,front,back,right,left]")
			return nil
		}
		switch strings.ToUpper(args[1]) {
		case "TOP":
			stw.SetAngle(90.0, -90.0)
		case "FRONT":
			stw.SetAngle(0.0, -90.0)
		case "BACK":
			stw.SetAngle(0.0, 90.0)
		case "RIGHT":
			stw.SetAngle(0.0, 0.0)
		case "LEFT":
			stw.SetAngle(0.0, 180.0)
		}
	case "printrange":
		if usage {
			stw.addHistory(":printrange [on,true,yes] [a3tate,a3yoko,a4tate,a4yoko]")
			stw.addHistory(":printrange [off,false,no]")
			return nil
		}
		if narg < 2 {
			showprintrange = !showprintrange
			break
		}
		switch strings.ToUpper(args[1]) {
		case "ON", "TRUE", "YES":
			showprintrange = true
			if narg >= 3 {
				err := stw.exmode(fmt.Sprintf(":paper %s", strings.Join(args[2:], " ")))
				if err != nil {
					return err
				}
			}
		case "OFF", "FALSE", "NO":
			showprintrange = false
		default:
			err := stw.exmode(fmt.Sprintf(":paper %s", strings.Join(args[1:], " ")))
			if err != nil {
				return err
			}
			showprintrange = true
		}
	case "paper":
		if usage {
			stw.addHistory(":paper [a3tate,a3yoko,a4tate,a4yoko]")
			return nil
		}
		if narg < 2 {
			return st.NotEnoughArgs(":paper")
		}
		tate := regexp.MustCompile("(?i)a([0-9]+) *t(a(t(e?)?)?)?")
		yoko := regexp.MustCompile("(?i)a([0-9]+) *y(o(k(o?)?)?)?")
		name := strings.Join(args[1:], " ")
		switch {
		case tate.MatchString(name):
			fs := tate.FindStringSubmatch(name)
			switch fs[1] {
			case "3":
				stw.papersize = A3_TATE
			case "4":
				stw.papersize = A4_TATE
			}
		case yoko.MatchString(name):
			fs := yoko.FindStringSubmatch(name)
			switch fs[1] {
			case "3":
				stw.papersize = A3_YOKO
			case "4":
				stw.papersize = A4_YOKO
			}
		default:
			return errors.New(":paper unknown papersize")
		}
	case "color":
		if usage {
			stw.addHistory(":color [n,sect,rate,white,mono,strong]")
			return nil
		}
		if narg < 2 {
			stw.SetColorMode(st.ECOLOR_SECT)
			break
		}
		switch strings.ToUpper(args[1]) {
		case "N":
			stw.SetColorMode(st.ECOLOR_N)
		case "SECT":
			stw.SetColorMode(st.ECOLOR_SECT)
		case "RATE":
			stw.SetColorMode(st.ECOLOR_RATE)
		case "WHIHTE", "MONO", "MONOCHROME":
			stw.SetColorMode(st.ECOLOR_WHITE)
		case "STRONG":
			stw.SetColorMode(st.ECOLOR_STRONG)
		}
	case "mono":
		stw.SetColorMode(st.ECOLOR_WHITE)
	case "postscript":
		if fn == "" {
			fn = filepath.Join(stw.Cwd, "test.ps")
		}
		var paper ps.Paper
		switch stw.papersize {
		default:
			paper = ps.A4Portrait
		case A4_TATE:
			paper = ps.A4Portrait
		case A4_YOKO:
			paper = ps.A4Landscape
		case A3_TATE:
			paper = ps.A3Portrait
		case A3_YOKO:
			paper = ps.A3Landscape
		}
		v := stw.Frame.View.Copy()
		stw.Frame.SetFocus(nil)
		stw.Frame.CentringTo(paper)
		err := stw.Frame.PrintPostScript(fn, paper)
		stw.Frame.View = v
		if err != nil {
			return err
		}
	case "analysis":
		if usage {
			stw.addHistory(":analysis")
			return nil
		}
		err := stw.SaveFile(stw.Frame.Path)
		if err != nil {
			return err
		}
		var anarg string
		if narg >= 3 {
			anarg = args[2]
		} else {
			anarg = "-a"
		}
		err = stw.Analysis(filepath.ToSlash(stw.Frame.Path), anarg)
		if err != nil {
			return err
		}
		stw.Reload()
		stw.ReadAll()
		stw.Redraw()
	case "extractarclm":
		if usage {
			stw.addHistory(":extractarclm")
			return nil
		}
		err := stw.SaveFile(stw.Frame.Path)
		if err != nil {
			return err
		}
		err = stw.Analysis(filepath.ToSlash(stw.Frame.Path), "")
		if err != nil {
			return err
		}
		for _, ext := range []string{".inl", ".ihx", ".ihy"} {
			err := stw.Frame.ReadData(st.Ce(stw.Frame.Path, ext))
			if err != nil {
				stw.errormessage(err, ERROR)
			}
		}
	case "arclm001":
		if usage {
			stw.addHistory(":arclm001 {-period=name} {-all} {-solver=name} {-eps=value} {-noinit} filename")
			return nil
		}
		var otp string
		if fn == "" {
			otp = st.Ce(stw.Frame.Path, ".otp")
		} else {
			otp = fn
		}
		otps := []string{otp}
		if o, ok := argdict["OTP"]; ok {
			otp = o
		}
		sol := "LLS"
		if s, ok := argdict["SOLVER"]; ok {
			if s != "" {
				sol = strings.ToUpper(s)
			}
		}
		eps := 1e-16
		if e, ok := argdict["EPS"]; ok {
			if e != "" {
				tmp, err := strconv.ParseFloat(e, 64)
				if err == nil {
					eps = tmp
				}
			}
		}
		per := "L"
		if p, ok := argdict["PERIOD"]; ok {
			if p != "" {
				per = strings.ToUpper(p)
			}
		}
		var pers []string
		pers = []string{per}
		lap := 1
		var extra [][]float64
		if _, ok := argdict["ALL"]; ok {
			extra = make([][]float64, 2)
			pers = []string{"L", "X", "Y"}
			otps = []string{st.Ce(otp, ".otl"), st.Ce(otp, ".ohx"), st.Ce(otp, ".ohy")}
			per = "L"
			lap = 3
			for i, eper := range []string{"X", "Y"} {
				eaf := stw.Frame.Arclms[eper]
				_, _, vec, err := eaf.AssemGlobalVector(1.0)
				if err != nil {
					return err
				}
				extra[i] = vec
			}
		}
		stw.addHistory(fmt.Sprintf("PERIOD: %s", per))
		stw.addHistory(fmt.Sprintf("OUTPUT: %s", otp))
		stw.addHistory(fmt.Sprintf("SOLVER: %s", sol))
		stw.addHistory(fmt.Sprintf("EPS: %.3E", eps))
		init := true
		if _, ok := argdict["NOINIT"]; ok {
			init = false
			stw.addHistory("NO INITIALISATION")
		}
		af := stw.Frame.Arclms[per]
		go func() {
			err := af.Arclm001(otps, init, sol, eps, extra...)
			af.Endch <- err
		}()
		stw.CurrentLap("Calculating...", 0, lap)
		go func() {
		read001:
			for {
				select {
				case nlap := <-af.Lapch:
					stw.Frame.ReadArclmData(af, pers[nlap])
					af.Lapch <- 1
					stw.CurrentLap("Calculating...", nlap, lap)
					stw.Redraw()
				case <-af.Endch:
					stw.CurrentLap("Completed", lap, lap)
					stw.SetPeriod(per)
					stw.Redraw()
					break read001
				}
			}
		}()
	case "arclm201":
		if usage {
			stw.addHistory(":arclm201 {-period=name} {-lap=nlap} {-safety=val} {-start=val} {-noinit} filename")
			return nil
		}
		var otp string
		if fn == "" {
			otp = st.Ce(stw.Frame.Path, ".otp")
		} else {
			otp = fn
		}
		if o, ok := argdict["OTP"]; ok {
			otp = o
		}
		lap := 1
		if l, ok := argdict["LAP"]; ok {
			tmp, err := strconv.ParseInt(l, 10, 64)
			if err == nil {
				lap = int(tmp)
			}
		}
		safety := 1.0
		if s, ok := argdict["SAFETY"]; ok {
			tmp, err := strconv.ParseFloat(s, 64)
			if err == nil {
				safety = tmp
			}
		}
		start := 0.0
		if s, ok := argdict["START"]; ok {
			tmp, err := strconv.ParseFloat(s, 64)
			if err == nil {
				start = tmp
			}
		}
		per := "L"
		if p, ok := argdict["PERIOD"]; ok {
			if p != "" {
				per = strings.ToUpper(p)
			}
		}
		stw.addHistory(fmt.Sprintf("PERIOD: %s", per))
		stw.addHistory(fmt.Sprintf("OUTPUT: %s", otp))
		stw.addHistory(fmt.Sprintf("LAP: %d, SAFETY: %.3f, START: %.3f", lap, safety, start))
		init := true
		if _, ok := argdict["NOINIT"]; ok {
			init = false
			stw.addHistory("NO INITIALISATION")
		}
		af := stw.Frame.Arclms[per]
		go func() {
			err := af.Arclm201(otp, init, lap, safety, start)
			af.Endch <- err
		}()
		stw.CurrentLap("Calculating...", 0, lap)
		go func() {
		read201:
			for {
				select {
				case nlap := <-af.Lapch:
					stw.Frame.ReadArclmData(af, per)
					af.Lapch <- 1
					stw.CurrentLap("Calculating...", nlap, lap)
					stw.Redraw()
				case <-af.Endch:
					stw.CurrentLap("Completed", lap, lap)
					stw.Redraw()
					break read201
				}
			}
		}()
	case "arclm301":
		if usage {
			stw.addHistory(":arclm301 {-period=name} {-sects=val} {-eps=val} {-noinit} filename")
			return nil
		}
		var otp string
		var sects []int
		if fn == "" {
			otp = st.Ce(stw.Frame.Path, ".otp")
		} else {
			otp = fn
		}
		if o, ok := argdict["OTP"]; ok {
			otp = o
		}
		if s, ok := argdict["SECTS"]; ok {
			sects = SplitNums(s)
			stw.addHistory(fmt.Sprintf("SOIL SPRING: %v", sects))
		}
		eps := 1E-3
		if s, ok := argdict["EPS"]; ok {
			tmp, err := strconv.ParseFloat(s, 64)
			if err == nil {
				eps = tmp
			}
		}
		per := "L"
		if p, ok := argdict["PERIOD"]; ok {
			if p != "" {
				per = strings.ToUpper(p)
			}
		}
		stw.addHistory(fmt.Sprintf("PERIOD: %s", per))
		stw.addHistory(fmt.Sprintf("OUTPUT: %s", otp))
		stw.addHistory(fmt.Sprintf("EPS: %.3E", eps))
		af := stw.Frame.Arclms[per]
		init := true
		if _, ok := argdict["NOINIT"]; ok {
			init = false
			stw.addHistory("NO INITIALISATION")
		}
		go func() {
			err := af.Arclm301(otp, init, sects, eps)
			af.Endch <- err
		}()
		stw.CurrentLap("Calculating...", 0, 0)
		go func() {
		read301:
			for {
				select {
				case nlap := <-af.Lapch:
					stw.Frame.ReadArclmData(af, per)
					af.Lapch <- 1
					stw.CurrentLap("Calculating...", nlap, 0)
					stw.Redraw()
				case <-af.Endch:
					stw.CurrentLap("Completed", 0, 0)
					stw.Redraw()
					break read301
				}
			}
		}()
	}
	return nil
}

