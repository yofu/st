package st

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/yofu/abbrev"
	"github.com/yofu/complete"
	"github.com/yofu/ps"
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
	ExAbbrev = map[string]*complete.Complete{
		"e/dit":             complete.MustCompile(":edit %g", nil),
		"q/uit":             complete.MustCompile(":quit", nil),
		"vi/m":              complete.MustCompile(":vim %g", nil),
		"exp/lorer":         complete.MustCompile(":explorer %g", nil),
		"hk/you":            complete.MustCompile(":hkyou _ _ _ _", nil),
		"hw/eak":            complete.MustCompile(":hweak _ _ _ _", nil),
		"rp/ipe":            complete.MustCompile(":rpipe _ _ _ _", nil),
		"cp/ipe":            complete.MustCompile(":cpipe _ _", nil),
		"tk/you":            complete.MustCompile(":tkyou _ _ _ _", nil),
		"ck/you":            complete.MustCompile(":ckyou _ _ _ _", nil),
		"pla/te":            complete.MustCompile(":plate _ _", nil),
		"fixr/otate":        complete.MustCompile(":fixrotate", nil),
		"fixm/ove":          complete.MustCompile(":fixmove", nil),
		"noun/do":           complete.MustCompile(":noundo", nil),
		"un/do":             complete.MustCompile(":undo", nil),
		"w/rite":            complete.MustCompile(":write %g", nil),
		"sav/e":             complete.MustCompile(":save [mkdir:] %g", nil),
		"inc/rement":        complete.MustCompile(":increment [times:_] _", nil),
		"c/heck":            complete.MustCompile(":check", nil),
		"r/ead":             complete.MustCompile(":read %g", nil),
		"ins/ert":           complete.MustCompile(":insert %g", nil),
		"p/rop/s/ect":       complete.MustCompile(":propsect %g", nil),
		"w/rite/o/utput":    complete.MustCompile(":writeoutput _", nil),
		"w/rite/rea/ction":  complete.MustCompile(":writereaction [confed:] _", nil),
		"w/rite/k/ijun":     complete.MustCompile(":writekijun _", nil),
		"p/late/w/eight":    complete.MustCompile(":plateweight", nil),
		"nmi/nteraction":    complete.MustCompile(":nminteraction [ndiv:] [output:]", nil),
		"kabe/ryo":          complete.MustCompile(":kaberyo [half:_] [fc:_] [alpha:_] [route:$ROUTE]",
			map[string][]string{
				"ROUTE": []string{"1", "2-1", "2-2"},
			}),
		"w/ei/g/htcopy":     complete.MustCompile(":weightcopy [si:]", nil),
		"har/dcopy":         complete.MustCompile(":hardcopy", nil),
		"fi/g2":             complete.MustCompile(":fig2", nil),
		"dxf/":              complete.MustCompile(":dxf [dimension:$DNUM] [scale:_]",
			map[string][]string{
				"DNUM": []string{"2", "3"},
			}),
		"fe/nce":            complete.MustCompile(":fence", nil),
		"no/de":             complete.MustCompile(":node", nil),
		"xsc/ale":           complete.MustCompile(":xscale _", nil),
		"ysc/ale":           complete.MustCompile(":yscale _", nil),
		"zsc/ale":           complete.MustCompile(":zscale _", nil),
		"pl/oad":            complete.MustCompile(":pload _", nil),
		"z/oubun/d/isp":     complete.MustCompile(":zoubundisp", nil),
		"z/oubun/r/eaction": complete.MustCompile(":zoubunreaction", nil),
		"fac/ts":            complete.MustCompile(":facts [skipany:_] [skipall:_]", nil),
		"go/han/l/st":       complete.MustCompile(":gohanlst _ _", nil),
		"el/em":             complete.MustCompile(":elem $TYPE _",
			map[string][]string{
				"TYPE": []string{"sect", "etype", "reaction"},
			}),
		"ave/rage":          complete.MustCompile(":average", nil),
		"bo/nd":             complete.MustCompile(":bond", nil),
		"ax/is/2//c/ang":    complete.MustCompile(":axis2cang", nil),
		"resul/tant":        complete.MustCompile(":resultant", nil),
		"prest/ress":        complete.MustCompile(":prestress _", nil),
		"therm/al":          complete.MustCompile(":thermal _", nil),
		"div/ide":           complete.MustCompile(":divide $TYPE",
			map[string][]string{
				"TYPE": []string{"mid", "n", "elem", "ons", "axis", "length"},
			}),
		"e/lem/dup/lication": complete.MustCompile(":elemduplication [ignoresect:]", nil),
		"i/ntersect/a/ll":    complete.MustCompile(":intersectall", nil),
		"co/nf":              complete.MustCompile(":conf", nil),
		"pi/le":              complete.MustCompile(":pile", nil),
		"sec/tion":           complete.MustCompile(":section [nodisp:]_", nil),
		"c/urrent/v/alue":    complete.MustCompile(":currentvalue [abs:]", nil),
		"len/gth":            complete.MustCompile(":length [deformed:]", nil),
		"are/a":              complete.MustCompile(":area [deformed:]", nil),
		"an/alysis":          complete.MustCompile(":analysis", nil),
		"f/ilter":            complete.MustCompile(":filter $CONDITION",
			map[string][]string{
				"CONDITION": []string{"//", "TT", "on", "adjoin", "cv"},
			}),
		"ra/nge":             complete.MustCompile(":range", nil),
		"h/eigh/t/":          complete.MustCompile(":height _ _", nil),
		"h/eigh/t+/":         complete.MustCompile(":height+", nil),
		"h/eigh/t-/":         complete.MustCompile(":height-", nil),
		"ang/le":             complete.MustCompile(":angle _ _", nil),
		"view/":              complete.MustCompile(":view $DIRECTION",
			map[string][]string{
				"DIRECTION": []string{"top", "front", "back", "right", "left"},
			}),
		"paper/":             complete.MustCompile(":paper $NAME",
			map[string][]string{
				"NAME": []string{"a3tate", "a3yoko", "a4tate", "a4yoko"},
			}),
		"sec/tion/+/":        complete.MustCompile(":section+ _", nil),
		"col/or":             complete.MustCompile(":color $NAME",
			map[string][]string{
				"NAME": []string{"n", "sect", "rate", "white", "mono", "strong"},
			}),
		"ex/tractarclm":      complete.MustCompile(":extractarclm", nil),
		"s/aveas/ar/clm":     complete.MustCompile(":saveasarclm", nil),
		"a/rclm/001/":        complete.MustCompile(":arclm001 [period:$PERIOD] [all:] [solver:$SOLVER] [eps:_] [noinit:] _",
			map[string][]string{
				"PERIOD": []string{"l", "x", "y"},
				"SOLVER": []string{"LLS", "CRS"},
			}),
		"a/rclm/201/":        complete.MustCompile(":arclm201 [period:$PERIOD] [lap:_] [safety:_] [max:_] [start:_] [noinit:] _",
			map[string][]string{
				"PERIOD": []string{"l", "x", "y"},
			}),
		"a/rclm/301/":        complete.MustCompile(":arclm301 [period:$PERIOD] [sects:_] [eps:_] [noinit:] _",
			map[string][]string{
				"PERIOD": []string{"l", "x", "y"},
			}),
	}
)

var (
	Re_etype      = regexp.MustCompile("(?i)^ *et(y(pe?)?)? *={0,2} *([a-zA-Z]+)")
	Re_column     = regexp.MustCompile("(?i)co(l(u(m(n)?)?)?)?$")
	Re_girder     = regexp.MustCompile("(?i)gi(r(d(e(r)?)?)?)?$")
	Re_brace      = regexp.MustCompile("(?i)br(a(c(e)?)?)?$")
	Re_wall       = regexp.MustCompile("(?i)wa(l){0,2}$")
	Re_slab       = regexp.MustCompile("(?i)sl(a(b)?)?$")
	Re_sectnum    = regexp.MustCompile("(?i)^ *sect? *={0,2} *(range[(]{1}){0,1}[[]?([0-9, ]+)[]]?")
	Re_orgsectnum = regexp.MustCompile("(?i)^ *osect? *={0,2} *[[]?([0-9, ]+)[]]?")
)

func ExModeComplete(command string) (string, bool, bool, *complete.Complete) {
	usage := strings.HasSuffix(command, "?")
	cname := strings.TrimSuffix(command, "?")
	bang := strings.HasSuffix(cname, "!")
	cname = strings.TrimSuffix(cname, "!")
	cname = strings.ToLower(strings.TrimPrefix(cname, ":"))
	var rtn string
	var c *complete.Complete
	for ab, cp := range ExAbbrev {
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
	return rtn, bang, usage, c
}

func emptyExModech(ch chan interface{}, endch chan int) {
ex_empty:
	for {
		select {
		case <-time.After(time.Second):
			break ex_empty
		case <-endch:
			break ex_empty
		case <-ch:
			continue ex_empty
		}
	}
}

func ExMode(stw ExModer, frame *Frame, command string) error {
	if command == ":." {
		return ExMode(stw, frame, stw.LastExCommand())
	}
	exmodech := make(chan interface{})
	exmodeend := make(chan int)
	stw.SetLastExCommand(command)
	if !strings.Contains(command, "|") {
		err := exCommand(stw, frame, command, false, exmodech, exmodeend)
		if n, ok := err.(NotRedraw); ok {
			stw.History(n.Message())
			return err
		} else if u, ok := err.(Messager); ok {
			stw.History(u.Message())
			return nil
		} else {
			return err
		}
	}
	excms := strings.Split(command, "|")
	defer emptyExModech(exmodech, exmodeend)
	for _, com := range excms {
		err := exCommand(stw, frame, com, true, exmodech, exmodeend)
		if err != nil {
			if u, ok := err.(Messager); ok {
				stw.History(u.Message())
			} else {
				return err
			}
		}
	}
	return nil
}

func exCommand(stw ExModer, frame *Frame, command string, pipe bool, exmodech chan interface{}, exmodeend chan int) error {
	if len(command) == 1 {
		return NotEnoughArgs("exmode")
	}
	EPS := stw.EPS()
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
		fn = stw.CompleteFileName(args[1])
		if filepath.Dir(fn) == "." {
			fn = filepath.Join(stw.Cwd(), fn)
		}
	}
	cname, bang, usage, _ := ExModeComplete(args[0])
	evaluated := true
	var sender []interface{}
	defer func() {
		if pipe {
			go func(ents []interface{}) {
				for _, e := range ents {
					exmodech <- e
				}
				exmodeend <- 1
			}(sender)
		}
	}()
	switch cname {
	default:
		evaluated = false
	case "edit":
		if usage {
			return Usage(":edit filename {-u=.strc}")
		}
		if !bang && stw.IsChanged() {
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
			if !FileExists(fn) {
				sfn, err := stw.SearchFile(args[1])
				if err != nil {
					return err
				}
				err = stw.OpenFile(sfn, readrc)
				if err != nil {
					return err
				}
				// stw.Redraw()
			} else {
				err := stw.OpenFile(fn, readrc)
				if err != nil {
					return err
				}
				// stw.Redraw()
			}
		} else {
			stw.Reload()
		}
	case "quit":
		if usage {
			return Usage(":quit")
		}
		stw.Close(bang)
	case "eps":
		if usage {
			return Usage(":eps val")
		}
		if narg < 2 {
			return NotEnoughArgs(":eps")
		}
		val, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return err
		}
		stw.SetEPS(val)
		return Message(fmt.Sprintf("EPS=%.3E", EPS))
	case "fitscale":
		if usage {
			return Usage(":fitscale val")
		}
		if narg < 2 {
			return NotEnoughArgs(":fitscale")
		}
		val, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return err
		}
		stw.SetCanvasFitScale(val)
		return Message(fmt.Sprintf("FITSCALE=%.3E", stw.CanvasFitScale()))
	case "animatespeed":
		if usage {
			return Usage(":animatespeed val")
		}
		if narg < 2 {
			return NotEnoughArgs(":animatespeed")
		}
		val, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return err
		}
		stw.SetCanvasAnimateSpeed(val)
		return Message(fmt.Sprintf("ANIMATESPEED=%.3f", stw.CanvasAnimateSpeed()))
	case "mkdir":
		if usage {
			return Usage(":mkdir dirname")
		}
		os.MkdirAll(fn, 0644)
	case "#":
		if usage {
			return Usage(":#")
		}
		stw.ShowRecently()
	case "vim":
		if usage {
			return Usage(":vim filename")
		}
		Vim(fn)
	case "explorer":
		var dir string
		if narg < 2 {
			dir = stw.Cwd()
		} else {
			dir = args[1]
		}
		Explorer(dir)
	case "hkyou":
		if usage {
			return Usage(":hkyou h b tw tf")
		}
		if narg < 5 {
			return NotEnoughArgs(":hkyou")
		}
		al, err := NewHKYOU(args[1:5])
		if err != nil {
			return err
		}
		stw.ShapeData(al)
		if pipe {
			sender = []interface{}{al}
		}
	case "hweak":
		if usage {
			return Usage(":hweak h b tw tf")
		}
		if narg < 5 {
			return NotEnoughArgs(":hweak")
		}
		al, err := NewHWEAK(args[1:5])
		if err != nil {
			return err
		}
		stw.ShapeData(al)
		if pipe {
			sender = []interface{}{al}
		}
	case "rpipe":
		if usage {
			return Usage(":rpipe h b tw tf")
		}
		if narg < 5 {
			return NotEnoughArgs(":rpipe")
		}
		al, err := NewRPIPE(args[1:5])
		if err != nil {
			return err
		}
		stw.ShapeData(al)
		if pipe {
			sender = []interface{}{al}
		}
	case "cpipe":
		if usage {
			return Usage(":cpipe d t")
		}
		if narg < 3 {
			return NotEnoughArgs(":cpipe")
		}
		al, err := NewCPIPE(args[1:3])
		if err != nil {
			return err
		}
		stw.ShapeData(al)
		if pipe {
			sender = []interface{}{al}
		}
	case "tkyou":
		if usage {
			return Usage(":tkyou h b tw tf")
		}
		if narg < 5 {
			return NotEnoughArgs(":tkyou")
		}
		al, err := NewTKYOU(args[1:5])
		if err != nil {
			return err
		}
		stw.ShapeData(al)
		if pipe {
			sender = []interface{}{al}
		}
	case "ckyou":
		if usage {
			return Usage(":ckyou h b tw tf")
		}
		if narg < 5 {
			return NotEnoughArgs(":ckyou")
		}
		al, err := NewCKYOU(args[1:5])
		if err != nil {
			return err
		}
		stw.ShapeData(al)
		if pipe {
			sender = []interface{}{al}
		}
	case "plate":
		if usage {
			return Usage(":plate h b")
		}
		if narg < 3 {
			return NotEnoughArgs(":plate")
		}
		al, err := NewPLATE(args[1:3])
		if err != nil {
			return err
		}
		stw.ShapeData(al)
		if pipe {
			sender = []interface{}{al}
		}
	case "fixrotate":
		stw.ToggleFixRotate()
	case "fixmove":
		stw.ToggleFixMove()
	case "noundo":
		stw.UseUndo(false)
		return Message("undo/redo is off")
	case "undo":
		stw.UseUndo(true)
		stw.Snapshot()
		return Message("undo/redo is on")
	case "alt":
		stw.ToggleAltSelectNode()
		if stw.AltSelectNode() {
			return Message("select node with Alt key")
		} else {
			return Message("select elem with Alt key")
		}
	case "procs":
		if usage {
			return Usage(":procs numcpu")
		}
		if narg < 2 {
			current := runtime.GOMAXPROCS(-1)
			return Message(fmt.Sprintf("PROCS: %d", current))
		}
		tmp, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return err
		}
		val := int(tmp)
		if 1 <= val && val <= runtime.NumCPU() {
			old := runtime.GOMAXPROCS(val)
			return Message(fmt.Sprintf("PROCS: %d -> %d", old, val))
		}
	case "empty":
		if usage {
			return Usage(":empty")
		}
		emptyExModech(exmodech, exmodeend)
	}
	if evaluated {
		return nil
	}
	if frame == nil {
		return Message("frame is nil")
	}
	switch cname {
	default:
		return Message(fmt.Sprintf("no exmode command: %s", cname))
	case "write":
		if usage {
			return Usage(":write")
		}
		if fn == "" {
			stw.SaveFile(frame.Path)
		} else {
			if bang || (!FileExists(fn) || stw.Yn("Save", "上書きしますか")) {
				err := stw.SaveFile(fn)
				if err != nil {
					return err
				}
				if fn != frame.Path {
					stw.Copylsts(fn)
				}
			}
		}
	case "save":
		if usage {
			return Usage(":save filename {-u=.strc}")
		}
		if fn == "" {
			return NotEnoughArgs(":save")
		}
		if bang || (!FileExists(fn) || stw.Yn("Save", "上書きしますか")) {
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
			if stw.ElemSelected() {
				err = stw.SaveFileSelected(fn)
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
			if fn != frame.Path {
				stw.Copylsts(fn)
			}
			stw.Rebase(fn)
		}
	case "increment":
		if usage {
			return Usage(":increment {times:1}")
		}
		if !bang && stw.IsChanged() {
			switch stw.Yna("CHANGED", "変更を保存しますか", "キャンセル") {
			case 1:
				stw.SaveAS()
			case 2:
				fmt.Println("not saved")
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
		fn, err := Increment(frame.Path, "_", 1, times)
		if err != nil {
			return err
		}
		if bang || (!FileExists(fn) || stw.Yn("Save", "上書きしますか")) {
			err := stw.SaveFile(fn)
			if err != nil {
				return err
			}
			if fn != frame.Path {
				stw.Copylsts(fn)
			}
			stw.Rebase(fn)
			stw.Snapshot()
			EditReadme(filepath.Dir(fn))
		}
	case "tag":
		if usage {
			return Usage(":tag name")
		}
		if narg < 2 {
			return NotEnoughArgs(":tag")
		}
		name := args[1]
		err := stw.AddTag(name, bang)
		if err != nil {
			return err
		}
	case "checkout":
		if usage {
			return Usage(":checkout name")
		}
		if narg < 2 {
			return NotEnoughArgs(":checkout")
		}
		name := args[1]
		err := stw.Checkout(name)
		if err != nil {
			return err
		}
	case "read":
		if usage {
			return Usage(":read {type} filename")
		}
		if narg < 2 {
			return NotEnoughArgs(":read")
		}
		t := strings.ToLower(args[1])
		if narg < 3 {
			switch t {
			case "$all":
				stw.ReadAll()
			case "$data":
				for _, ext := range []string{".inl", ".ihx", ".ihy"} {
					err := frame.ReadData(Ce(frame.Path, ext))
					if err != nil {
						stw.ErrorMessage(err, ERROR)
					}
				}
			case "$results":
				mode := UpdateResult
				if _, ok := argdict["ADD"]; ok {
					mode = AddResult
					if _, ok2 := argdict["SEARCH"]; ok2 {
						mode = AddSearchResult
					}
				}
				for _, ext := range []string{".otl", ".ohx", ".ohy"} {
					err := frame.ReadResult(Ce(frame.Path, ext), uint(mode))
					if err != nil {
						stw.ErrorMessage(err, ERROR)
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
		fn = stw.CompleteFileName(args[2])
		if filepath.Dir(fn) == "." {
			fn = filepath.Join(stw.Cwd(), fn)
		}
		switch {
		case abbrev.For("d/ata", t):
			err := frame.ReadData(fn)
			if err != nil {
				return err
			}
		case abbrev.For("r/esult", t):
			mode := UpdateResult
			if _, ok := argdict["ADD"]; ok {
				mode = AddResult
				if _, ok2 := argdict["SEARCH"]; ok2 {
					mode = AddSearchResult
				}
			}
			err := frame.ReadResult(fn, uint(mode))
			if err != nil {
				return err
			}
		case abbrev.For("s/rcan", t):
			err := frame.ReadRat(fn)
			if err != nil {
				return err
			}
		case abbrev.For("l/ist", t):
			err := frame.ReadLst(fn)
			if err != nil {
				return err
			}
		case abbrev.For("w/eight", t):
			err := frame.ReadWgt(fn)
			if err != nil {
				return err
			}
		case abbrev.For("k/ijun", t):
			err := frame.ReadKjn(fn)
			if err != nil {
				return err
			}
		case abbrev.For("b/uckling", t):
			err := frame.ReadBuckling(fn)
			if err != nil {
				return err
			}
		case abbrev.For("z/oubun", t):
			err := frame.ReadZoubun(fn)
			if err != nil {
				return err
			}
		case t == "pgp":
			err := stw.ReadPgp(fn)
			if err != nil {
				return err
			}
		}
	case "insert":
		if usage {
			return Usage(":insert filename angle(deg)")
		}
		if narg > 2 && stw.NodeSelected() {
			angle, err := strconv.ParseFloat(args[2], 64)
			if err != nil {
				return err
			}
			err = frame.ReadInp(fn, stw.SelectedNodes()[0].Coord, angle*math.Pi/180.0, false)
			stw.Snapshot()
			if err != nil {
				return err
			}
			// stw.EscapeAll()
		}
	case "propsect":
		if usage {
			return Usage(":propsect filename")
		}
		err := frame.AddPropAndSect(fn, true)
		stw.Snapshot()
		if err != nil {
			return err
		}
	case "writeoutput":
		if usage {
			return Usage(":writeoutput filename period")
		}
		if narg < 3 {
			return NotEnoughArgs(":wo")
		}
		var err error
		period := strings.ToUpper(args[2])
		if stw.ElemSelected() {
			err = WriteOutput(fn, period, stw.SelectedElems())
		} else {
			err = frame.WriteOutput(fn, period)
		}
		if err != nil {
			return err
		}
	case "writereaction":
		if usage {
			return Usage(":writereaction filename direction")
		}
		if narg < 3 {
			return NotEnoughArgs(":wr")
		}
		tmp, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			return err
		}
		if _, ok := argdict["CONFED"]; ok {
			stw.SelectConfed()
		}
		if !stw.NodeSelected () {
			return errors.New(":writereaction: no selected node")
		}
		ns := stw.SelectedNodes()
		sort.Sort(NodeByNum{ns})
		err = WriteReaction(fn, ns, int(tmp))
		if err != nil {
			return err
		}
	case "writekijun":
		if usage {
			return Usage(":writekijun filename")
		}
		if fn == "" {
			fn = Ce(frame.Path, ".kjn")
		}
		err := frame.WriteKjn(fn)
		if err != nil {
			return err
		}
	case "zoubundisp":
		if usage {
			return Usage(":zoubundisp period direction")
		}
		if narg < 3 {
			return NotEnoughArgs(":zoubundisp")
		}
		if !stw.NodeSelected() {
			return NotEnoughArgs(":zoubundisp no selected node")
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
		fn := filepath.Join(filepath.Dir(frame.Path), "zoubunout.txt")
		err = frame.ReportZoubunDisp(fn, stw.SelectedNodes(), pers, d)
		if err != nil {
			return err
		}
	case "zoubunreaction":
		if usage {
			return Usage(":zoubunreaction period direction")
		}
		if narg < 3 {
			return NotEnoughArgs(":zoubunreaction")
		}
		if !stw.NodeSelected() {
			return NotEnoughArgs(":zoubunreaction no selected node")
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
		fn := filepath.Join(filepath.Dir(frame.Path), "zoubunout.txt")
		err = frame.ReportZoubunReaction(fn, stw.SelectedNodes(), pers, d)
		if err != nil {
			return err
		}
	case "plateweight":
		if usage {
			return Usage(":plateweight fn")
		}
		if fn == "" {
			fn = filepath.Join(filepath.Dir(frame.Path), "plateweight.txt")
		}
		err := frame.WritePlateWeight(fn)
		if err != nil {
			return err
		}
	case "weightcopy":
		if usage {
			return Usage(":weightcopy {-si}")
		}
		wgt := filepath.Join(stw.HomeDir(), "hogtxt.wgt")
		if fn == "" {
			fn = Ce(frame.Path, ".wgt")
		}
		si := false
		if _, ok := argdict["SI"]; ok {
			si = true
			ext := filepath.Ext(fn)
			fn = fmt.Sprintf("%ssi%s", PruneExt(fn), ext)
		}
		if !bang && FileExists(fn) {
			return errors.New(":weightcopy file already exists")
		}
		err := CopyFile(wgt, fn)
		if err != nil {
			return err
		}
		if !si {
			err = frame.ReadWgt(fn)
			if err != nil {
				return err
			}
		}
	case "hardcopy":
		if usage {
			return Usage(":hardcopy")
		}
		stw.Print()
	case "fig2":
		if usage {
			return Usage(":fig2 filename")
		}
		err := stw.ReadFig2(fn)
		if err != nil {
			return err
		}
	case "svg":
		if usage {
			return Usage(":svg filename")
		}
		err := stw.PrintSVG(fn)
		if err != nil {
			return err
		}
	case "dxf":
		if usage {
			return Usage(":dxf filename {-dimension=2,3} {-scale=val}")
		}
		dimension := 2
		if d, ok := argdict["DIMENSION"]; ok {
			switch d {
			case "2":
				dimension = 2
			case "3":
				dimension = 3
			}
		}
		scale := 1.0
		if v, ok := argdict["SCALE"]; ok {
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}
			scale = val
		}
		var err error
		switch dimension {
		case 2:
			err = frame.WriteDxf2D(Ce(fn, ".dxf"), scale)
		case 3:
			err = frame.WriteDxf3D(Ce(fn, ".dxf"), scale)
		default:
			return Message("unknown dimension")
		}
		if err != nil {
			return err
		}
	case "check":
		if usage {
			return Usage(":check")
		}
		stw.CheckFrame()
		return Message("CHECKED")
	case "elemduplication":
		if usage {
			return Usage(":elemduplication {-ignoresect=code}")
		}
		stw.Deselect()
		var isect []int
		var m bytes.Buffer
		if isec, ok := argdict["IGNORESECT"]; ok {
			if isec == "" {
				isect = nil
			} else {
				isect = SplitNums(isec)
				m.WriteString(fmt.Sprintf("IGNORE SECT: %v\n", isect))
			}
		} else {
			isect = nil
		}
		els := frame.ElemDuplication(isect)
		if len(els) != 0 {
			es := make([]*Elem, len(els))
			enum := 0
			for k := range els {
				es = append(es, k)
				enum++
			}
			stw.SelectElem(es[:enum])
		}
		return Message(m.String())
	case "intersectall":
		if usage {
			return Usage(":intersectall")
		}
		l := len(stw.SelectedElems())
		if l <= 1 {
			return nil
		}
		go func() {
			err := frame.IntersectAll(stw.SelectedElems(), EPS)
			frame.Endch <- err
		}()
		stw.CurrentLap("Calculating...", 0, l)
		go func() {
			var err error
			var nlap int
		iallloop:
			for {
				select {
				case nlap = <-frame.Lapch:
					stw.CurrentLap("Calculating...", nlap, l)
					stw.Redraw()
				case err = <-frame.Endch:
					if err != nil {
						stw.CurrentLap("Error", nlap, l)
						stw.ErrorMessage(err, ERROR)
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
			return Usage(":srcal {-fbold} {-noreload} {-tmp}")
		}
		var m bytes.Buffer
		cond := NewCondition()
		if _, ok := argdict["FBOLD"]; ok {
			m.WriteString("Fb: old")
			cond.FbOld = true
		}
		reload := true
		if _, ok := argdict["NORELOAD"]; ok {
			reload = false
		}
		if reload {
			stw.ReadFile(Ce(frame.Path, ".lst"))
		}
		otp := frame.Path
		if _, ok := argdict["TMP"]; ok {
			otp = "tmp"
		}
		frame.SectionRateCalculation(otp, "L", "X", "X", "Y", "Y", -1.0, cond)
		return Message(m.String())
	case "nminteraction":
		if usage {
			return Usage(":nminteraction sectcode")
		}
		if narg < 2 {
			return NotEnoughArgs(":nminteraction")
		}
		tmp, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return err
		}
		if al, ok := frame.Allows[int(tmp)]; ok {
			var otp bytes.Buffer
			var m bytes.Buffer
			cond := NewCondition()
			ndiv := 100
			if nd, ok := argdict["NDIV"]; ok {
				if nd != "" {
					m.WriteString(fmt.Sprintf("NDIV: %s", nd))
					tmp, err := strconv.ParseInt(nd, 10, 64)
					if err == nil {
						ndiv = int(tmp)
					}
				}
			}
			filename := "nmi.txt"
			if o, ok := argdict["OUTPUT"]; ok {
				if o != "" {
					m.WriteString(fmt.Sprintf("OUTPUT: %s", o))
					filename = o
				}
			}
			switch al.(type) {
			default:
				return nil
			case *RCColumn:
				nmax := al.(*RCColumn).Nmax(cond)
				nmin := al.(*RCColumn).Nmin(cond)
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
			return Message(m.String())
		}
	case "gohanlst":
		if usage {
			return Usage(":gohanlst factor sectcode...")
		}
		if narg < 3 {
			return NotEnoughArgs(":gohanlst")
		}
		val, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return err
		}
		sects := SplitNums(strings.Join(args[2:], " "))
		var otp bytes.Buffer
		var etype string
		for _, snum := range sects {
			if sec, ok := frame.Sects[snum]; ok {
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
		w, err := os.Create(filepath.Join(stw.Cwd(), "gohan.lst"))
		defer w.Close()
		if err != nil {
			return err
		}
		otp = AddCR(otp)
		otp.WriteTo(w)
	case "kaberyo":
		if usage {
			return Usage(":kaberyo {-half} {-fc=24} {-alpha=1.155} {-route}")
		}
		els := currentelem(stw, exmodech, exmodeend)
		var m bytes.Buffer
		var props []int
		if val, ok := argdict["HALF"]; ok {
			props = SplitNums(val)
			m.WriteString(fmt.Sprintf("HALF: %v", props))
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
		m.WriteString(fmt.Sprintf("ALPHA: %.3f", alpha))
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
		m.WriteString(fmt.Sprintf(" COEFFICIENT: COLUMN=%.1f WALL=%.1f", ccol, cwall))
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
		m.WriteString(fmt.Sprintf("COLUMN: %.3f WALL: %.3f TOTAL: %.3f", sumcol, sumwall, total))
		return Message(m.String())
	case "facts":
		if usage {
			return Usage(":facts {-skipany=code} {-skipall=code}")
		}
		var m bytes.Buffer
		fn = Ce(frame.Path, ".fes")
		var skipany, skipall []int
		if sany, ok := argdict["SKIPANY"]; ok {
			if sany == "" {
				skipany = nil
			} else {
				skipany = SplitNums(sany)
				m.WriteString(fmt.Sprintf("SKIP ANY: %v", skipany))
			}
		} else {
			skipany = nil
		}
		if sall, ok := argdict["SKIPALL"]; ok {
			if sall == "" {
				skipall = nil
			} else {
				skipall = SplitNums(sall)
				m.WriteString(fmt.Sprintf("SKIP ALL: %v", skipall))
			}
		} else {
			skipall = nil
		}
		err := frame.Facts(fn, []int{COLUMN, GIRDER, BRACE, WBRACE, SBRACE}, skipany, skipall)
		if err != nil {
			return err
		}
		m.WriteString(fmt.Sprintf("Output: %s", fn))
		return Message(m.String())
	case "amountprop":
		if usage {
			return Usage(":amountprop propcode")
		}
		if narg < 2 {
			return NotEnoughArgs(":amountprop")
		}
		props := SplitNums(strings.Join(args[1:], " "))
		if len(props) == 0 {
			return errors.New(":amountprop: no selected prop")
		}
		fn := filepath.Join(filepath.Dir(frame.Path), "amount.txt")
		err := frame.AmountProp(fn, props...)
		if err != nil {
			return err
		}
	case "amountlst":
		if usage {
			return Usage(":amountlst propcode")
		}
		var sects []int
		if narg < 2 {
			if _, ok := argdict["ALL"]; ok {
				sects = make([]int, len(frame.Sects))
				i := 0
				for _, sec := range frame.Sects {
					if sec.Num >= 900 {
						continue
					}
					sects[i] = sec.Num
					i++
				}
				sects = sects[:i]
				sort.Ints(sects)
			} else {
				return NotEnoughArgs(":amountlst")
			}
		}
		if sects == nil {
			sects = SplitNums(strings.Join(args[1:], " "))
		}
		if len(sects) == 0 {
			return errors.New(":amountlst: no selected sect")
		}
		fn := filepath.Join(filepath.Dir(frame.Path), "amountltxt")
		err := frame.AmountLst(fn, sects...)
		if err != nil {
			return err
		}
	case "node":
		if usage {
			var m bytes.Buffer
			m.WriteString(":node nnum\n")
			m.WriteString(":node [x,y,z] [>,<,=] coord\n")
			m.WriteString(":node {confed/pinned/fixed/free}\n")
			m.WriteString(":node pile num")
			m.WriteString(":node sect num")
			return Usage(m.String())
		}
		stw.Deselect()
		var f func(*Node) bool
		if narg >= 2 {
			condition := strings.ToUpper(strings.Join(args[1:], " "))
			coordstr := regexp.MustCompile("^ *([XYZ]) *([<!=>]{0,2}) *([-0-9.]+)")
			numstr := regexp.MustCompile("^[0-9, ]+$")
			pilestr := regexp.MustCompile("^ *PILE *([0-9, ]+)$")
			sectstr := regexp.MustCompile("^ *SECT *([0-9, ]+)$")
			switch {
			default:
				return errors.New(":node: unknown format")
			case numstr.MatchString(condition):
				nnums := SplitNums(condition)
				ns := make([]*Node, len(nnums))
				nods := 0
				for i, nnum := range nnums {
					if n, ok := frame.Nodes[nnum]; ok {
						ns[i] = n
						nods++
					}
				}
				stw.SelectNode(ns[:nods])
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
					f = func(n *Node) bool {
						if n.Coord[ind] == val {
							return true
						}
						return false
					}
				case "!=":
					f = func(n *Node) bool {
						if n.Coord[ind] != val {
							return true
						}
						return false
					}
				case ">":
					f = func(n *Node) bool {
						if n.Coord[ind] > val {
							return true
						}
						return false
					}
				case ">=":
					f = func(n *Node) bool {
						if n.Coord[ind] >= val {
							return true
						}
						return false
					}
				case "<":
					f = func(n *Node) bool {
						if n.Coord[ind] < val {
							return true
						}
						return false
					}
				case "<=":
					f = func(n *Node) bool {
						if n.Coord[ind] <= val {
							return true
						}
						return false
					}
				}
			case pilestr.MatchString(condition):
				fs := pilestr.FindStringSubmatch(condition)
				pnums := SplitNums(fs[1])
				f = func(n *Node) bool {
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
			case sectstr.MatchString(condition):
				fs := sectstr.FindStringSubmatch(condition)
				snums := SplitNums(fs[1])
				if _, ok := argdict["ALL"]; ok {
					nnums := make(map[int]int, len(frame.Nodes))
					for _, n := range frame.Nodes {
						nnums[n.Num] = 0
					}
				node_sect_all:
					for _, el := range frame.Elems {
						for _, snum := range snums {
							if el.Sect.Num == snum {
								continue node_sect_all
							}
						}
						for _, en := range el.Enod {
							nnums[en.Num]++
						}
					}
					f = func(n *Node) bool {
						if ref, ok := nnums[n.Num]; ok {
							if ref > 0 {
								return false
							}
						}
						return true
					}
				} else {
					els := make([]*Elem, len(frame.Elems))
					num := 0
				node_sect_any:
					for _, el := range frame.Elems {
						for _, snum := range snums {
							if el.Sect.Num == snum {
								els[num] = el
								num++
								continue node_sect_any
							}
						}
					}
					els = els[:num]
					f = func(n *Node) bool {
						for _, el := range els {
							for _, en := range el.Enod {
								if n == en {
									return true
								}
							}
						}
						return false
					}
				}
			case abbrev.For("CONF/ED", condition):
				f = func(n *Node) bool {
					for i := 0; i < 6; i++ {
						if n.Conf[i] {
							return true
						}
					}
					return false
				}
			case condition == "FREE":
				f = func(n *Node) bool {
					for i := 0; i < 6; i++ {
						if n.Conf[i] {
							return false
						}
					}
					return true
				}
			case abbrev.For("PIN/NED", condition):
				f = func(n *Node) bool {
					return n.IsPinned()
				}
			case abbrev.For("FIX/ED", condition):
				f = func(n *Node) bool {
					return n.IsFixed()
				}
			}
			if f != nil {
				ns := make([]*Node, len(frame.Nodes))
				num := 0
				for _, n := range frame.Nodes {
					if f(n) {
						ns[num] = n
						num++
					}
				}
				stw.SelectNode(ns[:num])
			}
		} else {
			ns := make([]*Node, len(frame.Nodes))
			num := 0
			for _, n := range frame.Nodes {
				ns[num] = n
				num++
			}
			stw.SelectNode(ns[:num])
		}
		if pipe {
			ns := stw.SelectedNodes()
			num := len(ns)
			sender = make([]interface{}, num)
			for i := 0; i < num; i++ {
				sender[i] = ns[i]
			}
		}
	case "conf":
		if usage {
			return Usage(":conf [0,1]{6}")
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
			stw.SetConf(lis)
		} else {
			return NotEnoughArgs(":conf")
		}
	case "pile":
		if usage {
			return Usage(":pile pilecode")
		}
		if !stw.NodeSelected() {
			return errors.New(":pile no selected node")
		}
		if narg < 2 {
			for _, n := range stw.SelectedNodes() {
				n.Pile = nil
			}
			break
		}
		val, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return err
		}
		if p, ok := frame.Piles[int(val)]; ok {
			for _, n := range stw.SelectedNodes() {
				n.Pile = p
			}
			stw.Snapshot()
		} else {
			return errors.New(fmt.Sprintf(":pile PILE %d doesn't exist", val))
		}
	case "xscale":
		if usage {
			return Usage(":xscale factor coord")
		}
		if !stw.NodeSelected() {
			return errors.New(":xscale no selected node")
		}
		if narg < 3 {
			return NotEnoughArgs(":xscale")
		}
		factor, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return err
		}
		coord, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			return err
		}
		for _, n := range stw.SelectedNodes() {
			if n == nil {
				continue
			}
			n.Scale([]float64{coord, 0.0, 0.0}, factor, 1.0, 1.0)
		}
		stw.Snapshot()
	case "yscale":
		if usage {
			return Usage(":yscale factor coord")
		}
		if !stw.NodeSelected() {
			return errors.New(":yscale no selected node")
		}
		if narg < 3 {
			return NotEnoughArgs(":yscale")
		}
		factor, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return err
		}
		coord, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			return err
		}
		for _, n := range stw.SelectedNodes() {
			if n == nil {
				continue
			}
			n.Scale([]float64{0.0, coord, 0.0}, 1.0, factor, 1.0)
		}
		stw.Snapshot()
	case "zscale":
		if usage {
			return Usage(":zscale factor coord")
		}
		if !stw.NodeSelected() {
			return errors.New(":zscale no selected node")
		}
		if narg < 3 {
			return NotEnoughArgs(":zscale")
		}
		factor, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return err
		}
		coord, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			return err
		}
		for _, n := range stw.SelectedNodes() {
			if n == nil {
				continue
			}
			n.Scale([]float64{0.0, 0.0, coord}, 1.0, 1.0, factor)
		}
		stw.Snapshot()
	case "pload":
		if usage {
			return Usage(":pload position value")
		}
		if !stw.NodeSelected() {
			return errors.New(":pload no selected node")
		}
		if narg < 3 {
			return NotEnoughArgs(":pload")
		}
		ind, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return err
		}
		val, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			return err
		}
		for _, n := range stw.SelectedNodes() {
			if n == nil {
				continue
			}
			n.Load[int(ind)] = val
		}
		stw.Snapshot()
	case "cmq":
		if usage {
			return Usage(":cmq period")
		}
		if narg < 2 {
			return NotEnoughArgs(":cmq")
		}
		els := currentelem(stw, exmodech, exmodeend)
		if strings.EqualFold(args[1], "zero") {
			for _, el := range els {
				for i:=0; i<12; i++ {
					el.Cmq[i] = 0.0
				}
			}
			return nil
		}
		if s, ok := frame.ResultFileName[args[1]]; ok {
			if s == "" {
				return errors.New(fmt.Sprintf("period %s: no data", args[1]))
			}
			for _, el := range els {
				for i:=0; i<2; i++ {
					for j:=0; j<6; j++ {
						el.Cmq[6*i+j] = el.Stress[args[1]][el.Enod[i].Num][j]
					}
				}
			}
		}
	case "lock":
		if usage {
			return Usage(":lock")
		}
		els := currentelem(stw, exmodech, exmodeend)
		for _, el := range els {
			el.Lock = true
		}
	case "unlock":
		if usage {
			return Usage(":unlock")
		}
		els := currentelem(stw, exmodech, exmodeend)
		for _, el := range els {
			el.Lock = false
		}
	case "elem":
		if usage {
			return Usage(":elem [elemcode,sect sectcode,etype,reaction,locked]")
		}
		stw.Deselect()
		var f func(*Elem) bool
		if narg >= 2 {
			condition := strings.ToUpper(strings.Join(args[1:], " "))
			numstr := regexp.MustCompile("^[0-9, ]+$")
			switch {
			default:
				return errors.New(":elem: unknown format")
			case numstr.MatchString(condition):
				enums := SplitNums(condition)
				elems := make([]*Elem, len(enums))
				els := 0
				for i, enum := range enums {
					if el, ok := frame.Elems[enum]; ok {
						elems[i] = el
						els++
					}
				}
				stw.SelectElem(elems[:els])
			case Re_sectnum.MatchString(condition):
				f, _ = SectFilter(condition)
				if f == nil {
					return errors.New(":elem sect: format error")
				}
			case Re_orgsectnum.MatchString(condition):
				f, _ = OriginalSectFilter(condition)
				if f == nil {
					return errors.New(":elem sect: format error")
				}
			case Re_etype.MatchString(condition):
				f, _ = EtypeFilter(condition)
				if f == nil {
					return errors.New(":elem etype: format error")
				}
			case strings.EqualFold(condition, "curtain"):
				f = func(el *Elem) bool {
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
				f = func(el *Elem) bool {
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
				f = func(el *Elem) bool {
					switch el.Etype {
					case COLUMN, GIRDER, BRACE, WALL, SLAB:
						val, err := el.RateMax(frame.Show)
						if err != nil {
							return false
						}
						if val > threshold {
							return true
						}
					}
					return false
				}
			case strings.EqualFold(condition, "reaction"):
				f = func(el *Elem) bool {
					return el.Sect.IsReaction()
				}
			case strings.EqualFold(condition, "locked"):
				f = func(el *Elem) bool {
					return el.Lock
				}
			}
			if f != nil {
				els := make([]*Elem, len(frame.Elems))
				num := 0
				for _, el := range frame.Elems {
					if f(el) {
						els[num] = el
						num++
					}
				}
				stw.SelectElem(els[:num])
			}
		} else {
			els := make([]*Elem, len(frame.Elems))
			num := 0
			for _, el := range frame.Elems {
				els[num] = el
				num++
			}
			stw.SelectElem(els[:num])
		}
		if pipe {
			els := stw.SelectedElems()
			num := len(els)
			sender = make([]interface{}, num)
			for i := 0; i < num; i++ {
				sender[i] = els[i]
			}
		}
	case "reaction":
		if usage {
			return Usage(":reaction")
		}
		w := 0.0
		a := 0.0
		i := 0
		sects := make([]*Sect, len(frame.Sects))
		for _, el := range frame.Elems {
			if el.Sect.Num >= 900 {
				continue
			}
			if el.Sect.IsReaction() {
				a += el.Area()
				add := true
				for _, s := range sects[:i] {
					if el.Sect == s {
						add = false
						break
					}
				}
				if add {
					sects[i] = el.Sect
					i++
				}
			}
			ws := el.Weight()
			w += ws[1]
		}
		if a == 0.0 {
			return nil
		}
		val := -w / a
		val = math.Floor(val*1000) * 0.001
		for _, s := range sects[:i] {
			s.Lload[1] += val
		}
	case "fence":
		if usage {
			return Usage(":fence axis coord {-plate}")
		}
		if narg < 3 {
			return NotEnoughArgs(":fence")
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
			if int(ind) > frame.Ai.Nfloor-1 {
				return errors.New(":fence height: index error")
			}
			val = frame.Ai.Boundary[int(ind)]
		}
		plate := false
		if _, ok := argdict["PLATE"]; ok {
			plate = true
		}
		stw.SelectElem(frame.Fence(axis, val, plate))
		if pipe {
			els := stw.SelectedElems()
			num := len(els)
			sender = make([]interface{}, num)
			for i := 0; i < num; i++ {
				sender[i] = els[i]
			}
		}
	case "filter":
		if usage {
			return Usage(":filter condition")
		}
		tmpels, err := FilterElem(frame, stw.SelectedElems(), strings.Join(args[1:], " "))
		if err != nil {
			return err
		}
		stw.SelectElem(tmpels)
		if pipe {
			els := stw.SelectedElems()
			num := len(els)
			sender = make([]interface{}, num)
			for i := 0; i < num; i++ {
				sender[i] = els[i]
			}
		}
	case "bond":
		if usage {
			return Usage(":bond [pin,rigid,[01_t]{6}] [upper,lower,sect sectcode]")
		}
		if narg < 2 {
			return NotEnoughArgs(":bond")
		}
		els := currentelem(stw, exmodech, exmodeend)
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
		f := func(el *Elem, ind int) bool {
			return true
		}
		if narg >= 3 {
			condition := strings.ToLower(strings.Join(args[2:], " "))
			switch {
			case abbrev.For("up/per", condition):
				f = func(el *Elem, ind int) bool {
					return el.Enod[ind].Coord[2] > el.Enod[1-ind].Coord[2]
				}
			case abbrev.For("lo/wer", condition):
				f = func(el *Elem, ind int) bool {
					return el.Enod[ind].Coord[2] < el.Enod[1-ind].Coord[2]
				}
			case Re_sectnum.MatchString(condition):
				tmpf, _ := SectFilter(condition)
				f = func(el *Elem, ind int) bool {
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
			return Usage(":section+ value")
		}
		if narg < 2 {
			return NotEnoughArgs(":section+")
		}
		tmp, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return err
		}
		if tmp == 0 {
			break
		}
		val := int(tmp)
		for _, el := range stw.SelectedElems() {
			if el == nil {
				continue
			}
			if sec, ok := frame.Sects[el.Sect.Num+val]; ok {
				el.Sect = sec
			}
		}
		stw.Snapshot()
	case "cang":
		if usage {
			return Usage(":cang val")
		}
		if narg < 2 {
			return NotEnoughArgs(":cang")
		}
		val, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return err
		}
		els := currentelem(stw, exmodech, exmodeend)
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
			return Usage(":axis2cang n1 n2 [strong,weak]")
		}
		if narg < 4 {
			return NotEnoughArgs(":axis2cang")
		}
		if !stw.ElemSelected() {
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
		var n1, n2 *Node
		var found bool
		if n1, found = frame.Nodes[int(nnum1)]; !found {
			return errors.New(fmt.Sprintf(":axis2cang: NODE %d not found", nnum1))
		}
		if n2, found = frame.Nodes[int(nnum2)]; !found {
			return errors.New(fmt.Sprintf(":axis2cang: NODE %d not found", nnum2))
		}
		vec := []float64{n2.Coord[0] - n1.Coord[0], n2.Coord[1] - n1.Coord[1], n2.Coord[2] - n1.Coord[2]}
		for _, el := range stw.SelectedElems() {
			if el == nil || el.IsHidden(frame.Show) || el.Lock || !el.IsLineElem() {
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
			return Usage(":invert")
		}
		els := currentelem(stw, exmodech, exmodeend)
		for _, el := range els {
			el.Invert()
		}
		stw.Snapshot()
	case "resultant":
		if !stw.ElemSelected() {
			return errors.New(":resultant no selected elem")
		}
		vec := make([]float64, 3)
		els := stw.SelectedElems()
		elems := make([]*Elem, len(els))
		enum := 0
		for _, el := range els {
			if el == nil || el.Lock || !el.IsLineElem() {
				continue
			}
			elems[enum] = el
			enum++
		}
		elems = elems[:enum]
		en, err := CommonEnod(elems...)
		if err != nil {
			return err
		}
		if en == nil || len(en) == 0 {
			return errors.New(":resultant no common enod")
		}
		axis := [][]float64{XAXIS, YAXIS, ZAXIS}
		per := frame.Show.Period
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
		var m bytes.Buffer
		m.WriteString(fmt.Sprintf("NODE: %d", en[0].Num))
		m.WriteString(fmt.Sprintf("X: %.3f Y: %.3f Z: %.3f F: %.3f", vec[0], vec[1], vec[2], v))
		return Message(m.String())
	case "prestress":
		if usage {
			return Usage(":prestress value")
		}
		if narg < 2 {
			return NotEnoughArgs(":prestress")
		}
		els := currentelem(stw, exmodech, exmodeend)
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
			return Usage(":thermal tmp[℃]")
		}
		if narg < 2 {
			return NotEnoughArgs(":thermal")
		}
		els := currentelem(stw, exmodech, exmodeend)
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
		var m bytes.Buffer
		m.WriteString(fmt.Sprintf("ALPHA: %.3E", alpha))
		for _, el := range els {
			if el == nil || el.Lock || !el.IsLineElem() {
				continue
			}
			if len(el.Sect.Figs) == 0 {
				continue
			}
			if a, ok := el.Sect.Figs[0].Value["AREA"]; ok {
				val := el.Sect.Figs[0].Prop.E * a * alpha * tmp
				el.Cmq[0] += val
				el.Cmq[6] -= val
			}
		}
		stw.Snapshot()
		return Message(m.String())
	case "divide":
		if narg < 2 {
			if usage {
				return Usage(":divide [mid, n, elem, ons, axis, length]")
			}
			return NotEnoughArgs(":divide")
		}
		var divfunc func(*Elem) ([]*Node, []*Elem, error)
		switch strings.ToLower(args[1]) {
		case "mid":
			if usage {
				return Usage(":divide mid")
			}
			divfunc = func(el *Elem) ([]*Node, []*Elem, error) {
				return el.DivideAtMid(EPS)
			}
		case "n":
			if usage {
				return Usage(":divide n div")
			}
			if narg < 3 {
				return NotEnoughArgs(":divide n")
			}
			val, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return err
			}
			ndiv := int(val)
			divfunc = func(el *Elem) ([]*Node, []*Elem, error) {
				return el.DivideInN(ndiv, EPS)
			}
		case "elem":
			if usage {
				return Usage(":divide elem (eps)")
			}
			eps := EPS
			if narg >= 3 {
				val, err := strconv.ParseFloat(args[2], 64)
				if err == nil {
					eps = val
				}
			}
			divfunc = func(el *Elem) ([]*Node, []*Elem, error) {
				els, err := el.DivideAtElem(eps)
				return nil, els, err
			}
		case "ons":
			if usage {
				return Usage(":divide ons (eps)")
			}
			eps := EPS
			if narg >= 3 {
				val, err := strconv.ParseFloat(args[2], 64)
				if err == nil {
					eps = val
				}
			}
			divfunc = func(el *Elem) ([]*Node, []*Elem, error) {
				return el.DivideAtOns(eps)
			}
		case "axis":
			if usage {
				return Usage(":divide axis [x, y, z] coord")
			}
			if narg < 4 {
				return NotEnoughArgs(":divide axis")
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
			divfunc = func(el *Elem) ([]*Node, []*Elem, error) {
				return el.DivideAtAxis(axis, val, EPS)
			}
		case "length":
			if usage {
				return Usage(":divide length l")
			}
			if narg < 3 {
				return NotEnoughArgs(":divide length")
			}
			val, err := strconv.ParseFloat(args[2], 64)
			if err != nil {
				return err
			}
			divfunc = func(el *Elem) ([]*Node, []*Elem, error) {
				return el.DivideAtLength(val, EPS)
			}
		}
		if divfunc == nil {
			return errors.New(":divide: unknown format")
		}
		if !stw.ElemSelected() {
			return errors.New(":divide: no selected elem")
		}
		tmpels := make([]*Elem, 0)
		enum := 0
		for _, el := range stw.SelectedElems() {
			if el == nil {
				continue
			}
			_, els, err := divfunc(el)
			if err != nil {
				stw.ErrorMessage(err, ERROR)
				continue
			}
			if err == nil && len(els) > 1 {
				tmpels = append(tmpels, els...)
				enum += len(els)
			}
		}
		stw.SelectElem(tmpels[:enum])
		stw.Snapshot()
	case "section":
		if usage {
			return Usage(":section sectcode {-nodisp}")
		}
		nodisp := false
		if _, ok := argdict["NODISP"]; ok {
			nodisp = true
		}
		if narg < 2 {
			if stw.ElemSelected() {
				if !nodisp {
					stw.SectionData(stw.SelectedElems()[0].Sect)
				}
				if pipe {
					sender = []interface{}{stw.SelectedElems()[0].Sect}
				}
				return nil
			}
			stw.TextBox("SECTION").Clear()
			return nil
		}
		switch {
		case strings.EqualFold(args[1], "off"):
			stw.TextBox("SECTION").Clear()
			return nil
		case strings.EqualFold(args[1], "curtain"):
			sects := make([]*Sect, len(frame.Sects))
			num := 0
			for _, sec := range frame.Sects {
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
			if !nodisp {
				stw.SectionData(sects[0])
			}
			if pipe {
				num := len(sects)
				sender = make([]interface{}, num)
				for i := 0; i < num; i++ {
					sender[i] = sects[i]
				}
			}
		default:
			tmp, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}
			snum := int(tmp)
			if sec, ok := frame.Sects[snum]; ok {
				if narg >= 3 && args[2] == "<-" {
					select {
					case <-time.After(time.Second):
						break
					case al := <-exmodech:
						if al == nil {
							break
						}
						switch al := al.(type) {
						case Shape:
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
			return Usage(":thick nfig val")
		}
		if narg < 3 {
			return NotEnoughArgs(":thick")
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
		case sec := <-exmodech:
			if sec == nil {
				break
			}
			if sec, ok := sec.(*Sect); ok {
				if sec.HasThick(ind) {
					sec.Figs[ind].Value["THICK"] = val
				}
			}
		}
	case "add":
		if narg < 2 {
			if usage {
				return Usage(":add [elem, sect]")
			}
			return NotEnoughArgs(":add")
		}
		switch strings.ToLower(args[1]) {
		case "elem":
			if usage {
				return Usage(":add elem {-sect=code} {-etype=type}")
			}
			var etype int
			if et, ok := argdict["ETYPE"]; ok {
				switch {
				case Re_column.MatchString(et):
					etype = COLUMN
				case Re_girder.MatchString(et):
					etype = GIRDER
				case Re_slab.MatchString(et):
					etype = BRACE
				case Re_wall.MatchString(et):
					etype = WALL
				case Re_slab.MatchString(et):
					etype = SLAB
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
			var sect *Sect
			if sc, ok := argdict["SECT"]; ok {
				tmp, err := strconv.ParseInt(sc, 10, 64)
				if err != nil {
					return err
				}
				if sec, ok := frame.Sects[int(tmp)]; ok {
					sect = sec
				} else {
					return errors.New(fmt.Sprintf(":add elem: SECT %d doesn't exist", tmp))
				}
			} else {
				return errors.New(":add elem: no sectcode selected")
			}
			enod := make([]*Node, 0)
			enods := 0
		ex_addelem:
			for {
				select {
				case <-time.After(time.Second):
					break ex_addelem
				case <-exmodeend:
					break ex_addelem
				case ent := <-exmodech:
					if n, ok := ent.(*Node); ok {
						enod = append(enod, n)
						enods++
					}
				}
			}
			enod = enod[:enods]
			switch etype {
			case COLUMN, GIRDER, BRACE:
				frame.AddLineElem(-1, enod[:2], sect, etype)
			case WALL, SLAB:
				if enods > 4 {
					enod = enod[:4]
				}
				frame.AddPlateElem(-1, enod, sect, etype)
			}
		case "sec", "sect":
			if usage {
				return Usage(":add sect sectcode")
			}
			if narg < 3 {
				return NotEnoughArgs(":add sect")
			}
			val, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return err
			}
			snum := int(val)
			if _, ok := frame.Sects[snum]; ok && !bang {
				return errors.New(fmt.Sprintf(":add sect: SECT %d already exists", snum))
			}
			sec := frame.AddSect(snum)
			select {
			case <-time.After(time.Second):
				break
			case al := <-exmodech:
				if a, ok := al.(Shape); ok {
					sec.Figs = make([]*Fig, 1)
					f := NewFig()
					if p, ok := frame.Props[101]; ok {
						f.Prop = p
					} else {
						f.Prop = frame.DefaultProp()
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
				return Usage(":copy [sect]")
			}
			return NotEnoughArgs(":copy")
		}
		switch strings.ToLower(args[1]) {
		case "sec", "sect":
			if usage {
				return Usage(":copy sect sectcode")
			}
			if narg < 3 {
				return NotEnoughArgs(":copy sect")
			}
			val, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return err
			}
			snum := int(val)
			if _, ok := frame.Sects[snum]; ok && !bang {
				return errors.New(fmt.Sprintf(":copy sect: SECT %d already exists", snum))
			}
			select {
			case <-time.After(time.Second):
				break
			case s := <-exmodech:
				if sec, ok := s.(*Sect); ok {
					as := sec.Snapshot(frame)
					as.Num = snum
					frame.Sects[snum] = as
					frame.Show.Sect[snum] = true
				}
			}
		}
	case "currentvalue":
		if usage {
			return Usage(":currentvalue {-abs}")
		}
		if stw.ElemSelected() {
			var valfunc func(*Elem) float64
			var m bytes.Buffer
			if _, ok := argdict["ABS"]; ok {
				valfunc = func(elem *Elem) float64 {
					return elem.CurrentValue(frame.Show, true, true)
				}
			} else {
				valfunc = func(elem *Elem) float64 {
					return elem.CurrentValue(frame.Show, true, false)
				}
			}
			for _, el := range stw.SelectedElems() {
				if el == nil {
					continue
				}
				m.WriteString(fmt.Sprintf("ELEM %d: %.3f", el.Num, valfunc(el)))
			}
			return Message(m.String())
		}
	case "max":
		if usage {
			return Usage(":max {-abs}")
		}
		if stw.ElemSelected() {
			maxval := -1e16
			var valfunc func(*Elem) float64
			var sel *Elem
			abs := false
			if _, ok := argdict["ABS"]; ok {
				valfunc = func(elem *Elem) float64 {
					return elem.CurrentValue(frame.Show, true, true)
				}
				abs = true
			} else {
				valfunc = func(elem *Elem) float64 {
					return elem.CurrentValue(frame.Show, true, false)
				}
			}
			for _, el := range stw.SelectedElems() {
				if el == nil {
					continue
				}
				if tmpval := valfunc(el); tmpval > maxval {
					sel = el
					maxval = tmpval
				}
			}
			if sel != nil {
				stw.SelectElem([]*Elem{sel})
				return Message(fmt.Sprintf("ELEM %d: %.3f", sel.Num, sel.CurrentValue(frame.Show, true, abs)))
			}
		} else if stw.NodeSelected() {
			maxval := -1e16
			var valfunc func(*Node) float64
			var sn *Node
			abs := false
			if _, ok := argdict["ABS"]; ok {
				valfunc = func(node *Node) float64 {
					return node.CurrentValue(frame.Show, true, true)
				}
				abs = true
			} else {
				valfunc = func(node *Node) float64 {
					return node.CurrentValue(frame.Show, true, false)
				}
			}
			for _, n := range stw.SelectedNodes() {
				if n == nil {
					continue
				}
				if tmpval := valfunc(n); tmpval > maxval {
					sn = n
					maxval = tmpval
				}
			}
			if sn != nil {
				stw.SelectNode([]*Node{sn})
				return Message(fmt.Sprintf("NODE %d: %.3f", sn.Num, sn.CurrentValue(frame.Show, true, abs)))
			}
		} else {
			return errors.New(":max no selected elem/node")
		}
	case "min":
		if usage {
			return Usage(":min {-abs}")
		}
		if stw.ElemSelected() {
			minval := 1e16
			var valfunc func(*Elem) float64
			var sel *Elem
			abs := false
			if _, ok := argdict["ABS"]; ok {
				valfunc = func(elem *Elem) float64 {
					return elem.CurrentValue(frame.Show, false, true)
				}
				abs = true
			} else {
				valfunc = func(elem *Elem) float64 {
					return elem.CurrentValue(frame.Show, false, false)
				}
			}
			for _, el := range stw.SelectedElems() {
				if el == nil {
					continue
				}
				if tmpval := valfunc(el); tmpval < minval {
					sel = el
					minval = tmpval
				}
			}
			if sel != nil {
				stw.SelectElem([]*Elem{sel})
				return Message(fmt.Sprintf("ELEM %d: %.3f", sel.Num, sel.CurrentValue(frame.Show, false, abs)))
			}
		} else if stw.NodeSelected() {
			minval := 1e16
			var valfunc func(*Node) float64
			var sn *Node
			abs := false
			if _, ok := argdict["ABS"]; ok {
				valfunc = func(node *Node) float64 {
					return node.CurrentValue(frame.Show, false, true)
				}
				abs = true
			} else {
				valfunc = func(node *Node) float64 {
					return node.CurrentValue(frame.Show, false, false)
				}
			}
			for _, n := range stw.SelectedNodes() {
				if n == nil {
					continue
				}
				if tmpval := valfunc(n); tmpval < minval {
					sn = n
					minval = tmpval
				}
			}
			if sn != nil {
				stw.SelectNode([]*Node{sn})
				return Message(fmt.Sprintf("NODE %d: %.3f", sn.Num, sn.CurrentValue(frame.Show, false, abs)))
			}
		} else {
			return errors.New(":min no selected elem/node")
		}
	case "average":
		if usage {
			return Usage(":average {-abs}")
		}
		if stw.ElemSelected() {
			var valfunc func(*Elem) float64
			if _, ok := argdict["ABS"]; ok {
				valfunc = func(elem *Elem) float64 {
					return elem.CurrentValue(frame.Show, false, true)
				}
			} else {
				valfunc = func(elem *Elem) float64 {
					return elem.CurrentValue(frame.Show, false, false)
				}
			}
			val := 0.0
			num := 0
			for _, el := range stw.SelectedElems() {
				if el == nil {
					continue
				}
				val += valfunc(el)
				num++
			}
			if num >= 1 {
				return Message(fmt.Sprintf("%d ELEMs : %.5f", num, val/float64(num)))
			}
		} else if stw.NodeSelected() {
			var valfunc func(*Node) float64
			if _, ok := argdict["ABS"]; ok {
				valfunc = func(node *Node) float64 {
					return node.CurrentValue(frame.Show, false, true)
				}
			} else {
				valfunc = func(node *Node) float64 {
					return node.CurrentValue(frame.Show, false, false)
				}
			}
			val := 0.0
			num := 0
			for _, n := range stw.SelectedNodes() {
				if n == nil {
					continue
				}
				val += valfunc(n)
				num++
			}
			if num >= 1 {
				return Message(fmt.Sprintf("%d NODEs: %.5f", num, val/float64(num)))
			}
		} else {
			return errors.New(":average no selected elem/node")
		}
	case "sum":
		if usage {
			return Usage(":sum {-abs}")
		}
		if stw.ElemSelected() {
			var valfunc func(*Elem) float64
			if _, ok := argdict["ABS"]; ok {
				valfunc = func(elem *Elem) float64 {
					return elem.CurrentValue(frame.Show, false, true)
				}
			} else {
				valfunc = func(elem *Elem) float64 {
					return elem.CurrentValue(frame.Show, false, false)
				}
			}
			val := 0.0
			num := 0
			for _, el := range stw.SelectedElems() {
				if el == nil {
					continue
				}
				val += valfunc(el)
				num++
			}
			if num >= 1 {
				return Message(fmt.Sprintf("%d ELEMs : %.5f", num, val))
			}
		} else if stw.NodeSelected() {
			var valfunc func(*Node) float64
			if _, ok := argdict["ABS"]; ok {
				valfunc = func(node *Node) float64 {
					return node.CurrentValue(frame.Show, false, true)
				}
			} else {
				valfunc = func(node *Node) float64 {
					return node.CurrentValue(frame.Show, false, false)
				}
			}
			val := 0.0
			num := 0
			for _, n := range stw.SelectedNodes() {
				if n == nil {
					continue
				}
				val += valfunc(n)
				num++
			}
			if num >= 1 {
				return Message(fmt.Sprintf("%d NODEs: %.5f", num, val))
			}
		} else {
			return errors.New(":sum no selected elem/node")
		}
	case "erase":
		if usage {
			return Usage(":erase")
		}
		stw.Deselect()
	ex_erase:
		for {
			select {
			case <-time.After(time.Second):
				break ex_erase
			case <-exmodeend:
				break ex_erase
			case ent := <-exmodech:
				switch ent := ent.(type) {
				case *Node:
					frame.DeleteNode(ent.Num)
				case *Elem:
					frame.DeleteElem(ent.Num)
				}
			}
		}
		ns := frame.NodeNoReference()
		if len(ns) != 0 {
			for _, n := range ns {
				frame.DeleteNode(n.Num)
			}
		}
		stw.Snapshot()
	case "count":
		if usage {
			return Usage(":count")
		}
		var nnode, nelem int
	ex_count:
		for {
			select {
			case <-time.After(time.Second):
				break ex_count
			case <-exmodeend:
				break ex_count
			case ent := <-exmodech:
				switch ent.(type) {
				case *Node:
					nnode++
				case *Elem:
					nelem++
				}
			}
		}
		return Message(fmt.Sprintf("NODES: %d, ELEMS: %d", nnode, nelem))
	case "length":
		if usage {
			return Usage(":length [-deformed]")
		}
		deformed := false
		if _, ok := argdict["DEFORMED"]; ok {
			deformed = true
		}
		sum := 0.0
		for _, el := range currentelem(stw, exmodech, exmodeend) {
			if el.IsLineElem() {
				if deformed {
					sum += el.DeformedLength(frame.Show.Period)
				} else {
					sum += el.Length()
				}
			}
		}
		if deformed {
			return Message(fmt.Sprintf("total length: %.3f (deformed)", sum))
		} else {
			return Message(fmt.Sprintf("total length: %.3f", sum))
		}
	case "area":
		if usage {
			return Usage(":area [-deformed]")
		}
		deformed := false
		if _, ok := argdict["DEFORMED"]; ok {
			deformed = true
		}
		sum := 0.0
		for _, el := range currentelem(stw, exmodech, exmodeend) {
			if !el.IsLineElem() {
				if deformed {
					sum += el.DeformedArea(frame.Show.Period)
				} else {
					sum += el.Area()
				}
			}
		}
		if deformed {
			return Message(fmt.Sprintf("total area: %.3f (deformed)", sum))
		} else {
			return Message(fmt.Sprintf("total area: %.3f", sum))
		}
	case "show":
		if usage {
			return Usage(":show")
		}
	ex_show:
		for {
			select {
			case <-time.After(time.Second):
				break ex_show
			case <-exmodeend:
				break ex_show
			case ent := <-exmodech:
				if h, ok := ent.(Hider); ok {
					h.Show()
				}
			}
		}
	case "hide":
		if usage {
			return Usage(":hide")
		}
	ex_hide:
		for {
			select {
			case <-time.After(time.Second):
				break ex_hide
			case <-exmodeend:
				break ex_hide
			case ent := <-exmodech:
				if h, ok := ent.(Hider); ok {
					h.Hide()
				}
			}
		}
	case "range":
		if usage {
			return Usage(":range [x,y,z] min max")
		}
		if narg == 1 {
			for i := 0; i < 3; i++ {
				stw.AxisRange(i, -100.0, 1000.0, false)
			}
			return nil
		}
		var axis int
		switch strings.ToLower(args[1]) {
		case "x":
			axis = 0
		case "y":
			axis = 1
		case "z":
			axis = 2
		default:
			return errors.New(":range unknown axis")
		}
		if narg == 2 {
			stw.AxisRange(axis, -100.0, 1000.0, false)
			return nil
		}
		var min, max float64
		tmp, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			return err
		}
		min = tmp
		if narg >= 4 {
			tmp, err := strconv.ParseFloat(args[3], 64)
			if err != nil {
				return err
			}
			max = tmp
		} else {
			max = min
		}
		stw.AxisRange(axis, min, max, false)
	case "height":
		if usage {
			return Usage(":height f1 f2")
		}
		if narg == 1 {
			stw.AxisRange(2, -100.0, 1000.0, false)
			return nil
		}
		if narg < 3 {
			return NotEnoughArgs(":height")
		}
		var min, max int
		if strings.EqualFold(args[1], "n") {
			min = frame.Ai.Nfloor
		} else {
			tmp, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}
			min = int(tmp)
		}
		if strings.EqualFold(args[2], "n") {
			max = frame.Ai.Nfloor
		} else {
			tmp, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return err
			}
			max = int(tmp)
		}
		l := len(frame.Ai.Boundary)
		if min < 0 || min >= l || max < 0 || max >= l {
			return errors.New(":height out of boundary")
		}
		stw.AxisRange(2, frame.Ai.Boundary[min], frame.Ai.Boundary[max], false)
	case "story", "storey":
		if usage {
			return Usage(":storey n")
		}
		if narg < 2 {
			return NotEnoughArgs(":storey")
		}
		tmp, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return err
		}
		n := int(tmp)
		if n <= 0 || n >= len(frame.Ai.Boundary)-1 {
			return errors.New(":storey out of boundary")
		}
		return ExMode(stw, frame, fmt.Sprintf("height %d %d", n-1, n+1))
	case "floor":
		if usage {
			return Usage(":floor n")
		}
		if narg < 2 {
			return NotEnoughArgs(":floor")
		}
		var n int
		switch strings.ToLower(args[1]) {
		case "g":
			n = 1
		case "r":
			n = len(frame.Ai.Boundary) - 1
		default:
			tmp, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}
			n = int(tmp)
			if n <= 0 || n >= len(frame.Ai.Boundary) {
				return errors.New(":floor out of boundary")
			}
		}
		return ExMode(stw, frame, fmt.Sprintf("height %d %d", n-1, n))
	case "height+":
		stw.NextFloor()
	case "height-":
		stw.PrevFloor()
	case "angle":
		if usage {
			return Usage(":angle phi theta")
		}
		if narg < 3 {
			return NotEnoughArgs(":angle")
		}
		angle := make([]float64, 2)
		for i := 0; i < 2; i++ {
			if args[1+i] == "_" {
				continue
			}
			val, err := strconv.ParseFloat(args[1+i], 64)
			if err != nil {
				return err
			}
			angle[i] = val
		}
		stw.SetAngle(angle[0], angle[1])
	case "view":
		if usage {
			return Usage(":view [top,front,back,right,left]")
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
			return Usage(":printrange [on,true,yes/off,false,no] [a3tate,a3yoko,a4tate,a4yoko]")
		}
		if narg < 2 {
			stw.ToggleShowPrintRange()
			break
		}
		switch strings.ToUpper(args[1]) {
		case "ON", "TRUE", "YES":
			stw.SetShowPrintRange(true)
			if narg >= 3 {
				err := ExMode(stw, frame, fmt.Sprintf(":paper %s", strings.Join(args[2:], " ")))
				if err != nil {
					return err
				}
			}
		case "OFF", "FALSE", "NO":
			stw.SetShowPrintRange(false)
		default:
			err := ExMode(stw, frame, fmt.Sprintf(":paper %s", strings.Join(args[1:], " ")))
			if err != nil {
				return err
			}
			stw.SetShowPrintRange(true)
		}
	case "paper":
		if usage {
			return Usage(":paper [a3tate,a3yoko,a4tate,a4yoko]")
		}
		if narg < 2 {
			return NotEnoughArgs(":paper")
		}
		tate := regexp.MustCompile("(?i)a([0-9]+) *t(a(t(e?)?)?)?")
		yoko := regexp.MustCompile("(?i)a([0-9]+) *y(o(k(o?)?)?)?")
		name := strings.Join(args[1:], " ")
		switch {
		case tate.MatchString(name):
			fs := tate.FindStringSubmatch(name)
			switch fs[1] {
			case "3":
				stw.SetPaperSize(A3_TATE)
			case "4":
				stw.SetPaperSize(A4_TATE)
			}
		case yoko.MatchString(name):
			fs := yoko.FindStringSubmatch(name)
			switch fs[1] {
			case "3":
				stw.SetPaperSize(A3_YOKO)
			case "4":
				stw.SetPaperSize(A4_YOKO)
			}
		default:
			return errors.New(":paper unknown papersize")
		}
	case "color":
		if usage {
			return Usage(":color [n,sect,rate,white,mono,strong]")
		}
		if narg < 2 {
			stw.SetColorMode(ECOLOR_SECT)
			break
		}
		switch strings.ToUpper(args[1]) {
		case "N":
			stw.SetColorMode(ECOLOR_N)
		case "SECT":
			stw.SetColorMode(ECOLOR_SECT)
		case "RATE":
			stw.SetColorMode(ECOLOR_RATE)
		case "WHIHTE", "MONO", "MONOCHROME":
			stw.SetColorMode(ECOLOR_WHITE)
		case "STRONG":
			stw.SetColorMode(ECOLOR_STRONG)
		}
	case "mono":
		stw.SetColorMode(ECOLOR_WHITE)
	case "postscript":
		if fn == "" {
			fn = filepath.Join(stw.Cwd(), "teps")
		}
		var paper ps.Paper
		switch stw.PaperSize() {
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
		v := frame.View.Copy()
		frame.SetFocus(nil)
		frame.CentringTo(paper)
		err := frame.PrintPostScript(fn, paper)
		frame.View = v
		if err != nil {
			return err
		}
	// case "analysis":
	// 	if usage {
	// 		return Usage(":analysis")
	// 	}
	// 	err := stw.SaveFile(frame.Path)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	var anarg string
	// 	if narg >= 3 {
	// 		anarg = args[2]
	// 	} else {
	// 		anarg = "-a"
	// 	}
	// 	err = stw.Analysis(filepath.ToSlash(frame.Path), anarg)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	stw.Reload()
	// 	stw.ReadAll()
	// 	stw.Redraw()
	case "extractarclm":
		if usage {
			return Usage(":extractarclm")
		}
		frame.ExtractArclm()
		// err = stw.Analysis(filepath.ToSlash(frame.Path), "")
		// if err != nil {
		// 	return err
		// }
		// for _, ext := range []string{".inl", ".ihx", ".ihy"} {
		// 	err := frame.ReadData(Ce(frame.Path, ext))
		// 	if err != nil {
		// 		stw.errormessage(err, ERROR)
		// 	}
		// }
	case "saveasarclm":
		if usage {
			return Usage(":saveasarclm")
		}
		frame.SaveAsArclm("")
	case "arclm001":
		if usage {
			return Usage(":arclm001 {-period=name} {-all} {-solver=name} {-eps=value} {-noinit} filename")
		}
		var otp string
		if fn == "" {
			otp = Ce(frame.Path, ".otp")
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
			otps = []string{Ce(otp, ".otl"), Ce(otp, ".ohx"), Ce(otp, ".ohy")}
			per = "L"
			lap = 3
			for i, eper := range []string{"X", "Y"} {
				eaf := frame.Arclms[eper]
				_, _, vec, err := eaf.AssemGlobalVector(1.0)
				if err != nil {
					return err
				}
				extra[i] = vec
			}
		}
		var m bytes.Buffer
		m.WriteString(fmt.Sprintf("PERIOD: %s", per))
		m.WriteString(fmt.Sprintf("OUTPUT: %s", otp))
		m.WriteString(fmt.Sprintf("SOLVER: %s", sol))
		m.WriteString(fmt.Sprintf("EPS: %.3E", eps))
		init := true
		if _, ok := argdict["NOINIT"]; ok {
			init = false
			m.WriteString("NO INITIALISATION")
		}
		af := frame.Arclms[per]
		go func() {
			err := af.Arclm001(otps, init, sol, eps, extra...)
			af.Endch <- err
		}()
		stw.CurrentLap("Calculating...", 0, lap)
		pivot := make(chan int)
		end := make(chan int)
		nodes := make([]*Node, len(frame.Nodes))
		i := 0
		for _, n := range frame.Nodes {
			nodes[i] = n
			i++
		}
		sort.Sort(NodeByNum{nodes})
		if stw.Pivot() {
			go stw.DrawPivot(nodes, pivot, end)
		} else {
			stw.Redraw()
		}
		go func() {
		read001:
			for {
				select {
				case <-af.Pivot:
					if stw.Pivot() {
						pivot <- 1
					}
				case nlap := <-af.Lapch:
					frame.ReadArclmData(af, pers[nlap])
					af.Lapch <- 1
					stw.CurrentLap("Calculating...", nlap, lap)
					stw.Redraw()
				case <-af.Endch:
					if stw.Pivot() {
						end <- 1
					}
					stw.CurrentLap("Completed", lap, lap)
					stw.SetPeriod(per)
					stw.Redraw()
					break read001
				}
			}
		}()
		return ArclmStart(m.String())
	case "arclm201":
		if usage {
			return Usage(":arclm201 {-period=name} {-lap=nlap} {-safety=val} {-max=val} {-start=val} {-noinit} filename")
		}
		var otp string
		if fn == "" {
			otp = Ce(frame.Path, ".otp")
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
		max := 1.0
		if s, ok := argdict["MAX"]; ok {
			tmp, err := strconv.ParseFloat(s, 64)
			if err == nil {
				max = tmp
			}
		}
		per := "L"
		if p, ok := argdict["PERIOD"]; ok {
			if p != "" {
				per = strings.ToUpper(p)
			}
		}
		var m bytes.Buffer
		m.WriteString(fmt.Sprintf("PERIOD: %s", per))
		m.WriteString(fmt.Sprintf("OUTPUT: %s", otp))
		m.WriteString(fmt.Sprintf("LAP: %d, SAFETY: %.3f, START: %.3f, MAX: %.3f", lap, safety, start, max))
		init := true
		if _, ok := argdict["NOINIT"]; ok {
			init = false
			m.WriteString("NO INITIALISATION")
		}
		af := frame.Arclms[per]
		go func() {
			err := af.Arclm201(otp, init, lap, safety, start, max)
			if err != nil {
				fmt.Println(err)
			}
			af.Endch <- err
		}()
		stw.CurrentLap("Calculating...", 0, lap)
		pivot := make(chan int)
		end := make(chan int)
		nodes := make([]*Node, len(frame.Nodes))
		i := 0
		for _, n := range frame.Nodes {
			nodes[i] = n
			i++
		}
		sort.Sort(NodeByNum{nodes})
		if stw.Pivot() {
			go stw.DrawPivot(nodes, pivot, end)
		} else {
			stw.Redraw()
		}
		go func() {
		read201:
			for {
				select {
				case <-af.Pivot:
					if stw.Pivot() {
						pivot <- 1
					}
				case nlap := <-af.Lapch:
					frame.ReadArclmData(af, per)
					af.Lapch <- 1
					stw.CurrentLap("Calculating...", nlap, lap)
					if stw.Pivot() {
						end <- 1
						go stw.DrawPivot(nodes, pivot, end)
					} else {
						stw.Redraw()
					}
				case <-af.Endch:
					stw.CurrentLap("Completed", lap, lap)
					stw.Redraw()
					break read201
				}
			}
		}()
		return ArclmStart(m.String())
	case "arclm202":
		if usage {
			return Usage(":arclm202 {-period=name} {-lap=nlap} {-safety=val} {-max=val} {-start=val} {-noinit} {-sects=val} {-comp=val} filename")
		}
		var otp string
		var sects []int
		if fn == "" {
			otp = Ce(frame.Path, ".otp")
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
		max := 1.0
		if s, ok := argdict["MAX"]; ok {
			tmp, err := strconv.ParseFloat(s, 64)
			if err == nil {
				max = tmp
			}
		}
		per := "L"
		if p, ok := argdict["PERIOD"]; ok {
			if p != "" {
				per = strings.ToUpper(p)
			}
		}
		comp := 0.0
		if c, ok := argdict["COMP"]; ok {
			val, err := strconv.ParseFloat(c, 64)
			if err == nil {
				comp = val
			}
		}
		var m bytes.Buffer
		m.WriteString(fmt.Sprintf("PERIOD: %s", per))
		m.WriteString(fmt.Sprintf("OUTPUT: %s", otp))
		m.WriteString(fmt.Sprintf("LAP: %d, SAFETY: %.3f, START: %.3f, MAX: %.3f, COMP: %.3f", lap, safety, start, max, comp))
		if s, ok := argdict["SECTS"]; ok {
			sects = SplitNums(s)
			m.WriteString(fmt.Sprintf("INCOMPRESSIBLE: %v", sects))
		}
		init := true
		if _, ok := argdict["NOINIT"]; ok {
			init = false
			m.WriteString("NO INITIALISATION")
		}
		af := frame.Arclms[per]
		go func() {
			err := af.Arclm202(otp, init, lap, safety, start, max, sects, comp)
			if err != nil {
				fmt.Println(err)
			}
			af.Endch <- err
		}()
		stw.CurrentLap("Calculating...", 0, lap)
		pivot := make(chan int)
		end := make(chan int)
		nodes := make([]*Node, len(frame.Nodes))
		i := 0
		for _, n := range frame.Nodes {
			nodes[i] = n
			i++
		}
		sort.Sort(NodeByNum{nodes})
		if stw.Pivot() {
			go stw.DrawPivot(nodes, pivot, end)
		} else {
			stw.Redraw()
		}
		go func() {
		read202:
			for {
				select {
				case <-af.Pivot:
					if stw.Pivot() {
						pivot <- 1
					}
				case nlap := <-af.Lapch:
					frame.ReadArclmData(af, per)
					af.Lapch <- 1
					stw.CurrentLap("Calculating...", nlap, lap)
					if stw.Pivot() {
						end <- 1
						go stw.DrawPivot(nodes, pivot, end)
					} else {
						stw.Redraw()
					}
				case <-af.Endch:
					stw.CurrentLap("Completed", lap, lap)
					stw.Redraw()
					break read202
				}
			}
		}()
		return ArclmStart(m.String())
	case "arclm301":
		if usage {
			return Usage(":arclm301 {-period=name} {-sects=val} {-eps=val} {-noinit} filename")
		}
		var otp string
		var sects []int
		var m bytes.Buffer
		if fn == "" {
			otp = Ce(frame.Path, ".otp")
		} else {
			otp = fn
		}
		if o, ok := argdict["OTP"]; ok {
			otp = o
		}
		if s, ok := argdict["SECTS"]; ok {
			sects = SplitNums(s)
			m.WriteString(fmt.Sprintf("SOIL SPRING: %v", sects))
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
		m.WriteString(fmt.Sprintf("PERIOD: %s", per))
		m.WriteString(fmt.Sprintf("OUTPUT: %s", otp))
		m.WriteString(fmt.Sprintf("EPS: %.3E", eps))
		af := frame.Arclms[per]
		init := true
		if _, ok := argdict["NOINIT"]; ok {
			init = false
			m.WriteString("NO INITIALISATION")
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
				case <-af.Pivot:
				case nlap := <-af.Lapch:
					frame.ReadArclmData(af, per)
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
		return ArclmStart(m.String())
	case "arclm401":
		if usage {
			return Usage(":arclm401 {-period=name} {-eps=val} {-noinit} filename")
		}
		var otp string
		var m bytes.Buffer
		if fn == "" {
			otp = Ce(frame.Path, ".otp")
		} else {
			otp = fn
		}
		if o, ok := argdict["OTP"]; ok {
			otp = o
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
		m.WriteString(fmt.Sprintf("PERIOD: %s", per))
		m.WriteString(fmt.Sprintf("OUTPUT: %s", otp))
		m.WriteString(fmt.Sprintf("EPS: %.3E", eps))
		af := frame.Arclms[per]
		init := true
		if _, ok := argdict["NOINIT"]; ok {
			init = false
			m.WriteString("NO INITIALISATION")
		}
		wgtdict := make(map[int]float64)
		for _, n := range frame.Nodes {
			wgtdict[n.Num] = n.Weight[1]
		}
		go func() {
			err := af.Arclm401(otp, init, eps, wgtdict)
			af.Endch <- err
		}()
		stw.CurrentLap("Calculating...", 0, 0)
		go func() {
		read401:
			for {
				select {
				case <-af.Pivot:
				case nlap := <-af.Lapch:
					frame.ReadArclmData(af, per)
					af.Lapch <- 1
					stw.CurrentLap("Calculating...", nlap, 0)
					stw.Redraw()
				case <-af.Endch:
					stw.CurrentLap("Completed", 0, 0)
					stw.Redraw()
					break read401
				}
			}
		}()
		return ArclmStart(m.String())
	}
	return nil
}

func currentelem(stw ExModer, exmodech chan interface{}, exmodeend chan int) []*Elem {
	var els []*Elem
	if !stw.ElemSelected() {
		enum := 0
		els = make([]*Elem, 0)
	current:
		for {
			select {
			case <-time.After(time.Second):
				break current
			case <-exmodeend:
				break current
			case el := <-exmodech:
				if el == nil {
					break current
				}
				if el, ok := el.(*Elem); ok {
					els = append(els, el)
					enum++
				}
			}
		}
		if enum == 0 {
			return []*Elem{}
		}
		els = els[:enum]
	} else {
		els = stw.SelectedElems()
	}
	return els
}
