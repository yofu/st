package st

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/yofu/abbrev"
	"github.com/yofu/complete"
	"github.com/yofu/ps"
	"github.com/yofu/st/arclm"
)

var (
	ExAbbrev = map[string]*complete.Complete{
		"e/dit":            complete.MustCompile(":edit %g", nil),
		"q/uit":            complete.MustCompile(":quit", nil),
		"res/ource":        complete.MustCompile(":resource %g", nil),
		"vi/m":             complete.MustCompile(":vim %g", nil),
		"exp/lorer":        complete.MustCompile(":explorer %g", nil),
		"hk/you":           complete.MustCompile(":hkyou _ _ _ _", nil),
		"hw/eak":           complete.MustCompile(":hweak _ _ _ _", nil),
		"cr/oss":           complete.MustCompile(":cross _ _ _ _", nil),
		"rp/ipe":           complete.MustCompile(":rpipe _ _ _ _", nil),
		"cp/ipe":           complete.MustCompile(":cpipe _ _", nil),
		"tk/you":           complete.MustCompile(":tkyou _ _ _ _", nil),
		"tw/eak":           complete.MustCompile(":tweak _ _ _ _", nil),
		"ck/you":           complete.MustCompile(":ckyou _ _ _ _", nil),
		"cw/eak":           complete.MustCompile(":cweak _ _ _ _", nil),
		"pla/te":           complete.MustCompile(":plate _ _", nil),
		"ang/le":           complete.MustCompile(":angle _ _ _ _", nil),
		"fixr/otate":       complete.MustCompile(":fixrotate", nil),
		"fixm/ove":         complete.MustCompile(":fixmove", nil),
		"noun/do":          complete.MustCompile(":noundo", nil),
		"un/do":            complete.MustCompile(":undo", nil),
		"w/rite":           complete.MustCompile(":write %g", nil),
		"sav/e":            complete.MustCompile(":save [mkdir:] %g", nil),
		"inc/rement":       complete.MustCompile(":increment [times:_] _", nil),
		"c/heck":           complete.MustCompile(":check", nil),
		"r/ead":            complete.MustCompile(":read %g", nil),
		"ins/ert":          complete.MustCompile(":insert %g", nil),
		"p/rop/s/ect":      complete.MustCompile(":propsect %g", nil),
		"w/rite/o/utput":   complete.MustCompile(":writeoutput _", nil),
		"w/rite/rea/ction": complete.MustCompile(":writereaction [confed:] _", nil),
		"w/rite/k/ijun":    complete.MustCompile(":writekijun _", nil),
		"p/late/w/eight":   complete.MustCompile(":plateweight", nil),
		"nmi/nteraction":   complete.MustCompile(":nminteraction [ndiv:] [output:]", nil),
		"kabe/ryo": complete.MustCompile(":kaberyo [half:_] [fc:_] [alpha:_] [route:$ROUTE]",
			map[string][]string{
				"ROUTE": []string{"1", "2-1", "2-2"},
			}),
		"w/ei/g/htcopy": complete.MustCompile(":weightcopy [si:]", nil),
		"har/dcopy":     complete.MustCompile(":hardcopy", nil),
		"fi/g2":         complete.MustCompile(":fig2", nil),
		"f/ig2/p/rint":  complete.MustCompile(":fig2print", nil),
		"dxf/": complete.MustCompile(":dxf [dimension:$DNUM] [scale:_]",
			map[string][]string{
				"DNUM": []string{"2", "3"},
			}),
		"plan/":             complete.MustCompile(":plan [floor:_]", nil),
		"jiku/":             complete.MustCompile(":jiku [name:_]", nil),
		"crosssec/tion":     complete.MustCompile(":crosssection [axis:_] [min:_] [max:_]", nil),
		"fe/nce":            complete.MustCompile(":fence", nil),
		"no/de":             complete.MustCompile(":node", nil),
		"xsc/ale":           complete.MustCompile(":xscale _", nil),
		"ysc/ale":           complete.MustCompile(":yscale _", nil),
		"zsc/ale":           complete.MustCompile(":zscale _", nil),
		"pl/oad":            complete.MustCompile(":pload _", nil),
		"z/oubun/d/isp":     complete.MustCompile(":zoubundisp", nil),
		"z/oubun/r/eaction": complete.MustCompile(":zoubunreaction", nil),
		"setb/oundary":      complete.MustCompile(":setboundary [eps:_]", nil),
		"fac/ts":            complete.MustCompile(":facts [skipany:_] [skipall:_]", nil),
		"1459/":             complete.MustCompile(":1459 [sect:_]", nil),
		"sl/ope":             complete.MustCompile(":slope", nil),
		"el/em/dis/tance":   complete.MustCompile(":elemdistance", nil),
		"go/han/l/st":       complete.MustCompile(":gohanlst _ _", nil),
		"el/em": complete.MustCompile(":elem $TYPE _",
			map[string][]string{
				"TYPE": []string{"sect", "etype", "curtain", "isgohan", "error", "reaction", "locked", "isolated"},
			}),
		"ave/rage":       complete.MustCompile(":average", nil),
		"bo/nd":          complete.MustCompile(":bond", nil),
		"ax/is/2//c/ang": complete.MustCompile(":axis2cang", nil),
		"resul/tant":     complete.MustCompile(":resultant", nil),
		"prest/ress":     complete.MustCompile(":prestress _", nil),
		"therm/al":       complete.MustCompile(":thermal _", nil),
		"div/ide": complete.MustCompile(":divide $TYPE",
			map[string][]string{
				"TYPE": []string{"mid", "n", "elem", "ons", "axis", "length"},
			}),
		"se/t": complete.MustCompile(":set [sect:_] [etype:$ETYPE]",
			map[string][]string{
				"ETYPE": []string{"column", "girder", "brace", "slab", "wall"},
			}),
		"n/ode/dup/lication": complete.MustCompile(":nodeduplication", nil),
		"e/lem/dup/lication": complete.MustCompile(":elemduplication [ignoresect:]", nil),
		"n/ode/n/oreference": complete.MustCompile(":nodenoreference", nil),
		"i/ntersect/a/ll":    complete.MustCompile(":intersectall", nil),
		"src/al":             complete.MustCompile(":srcal [fbold:] [noreload:] [qfact:_] [wfact:_] [bfact:_] [skipshort:] [temporary:] [moeshiro:]", nil),
		"co/nf":              complete.MustCompile(":conf", nil),
		"pi/le":              complete.MustCompile(":pile", nil),
		"sec/tion":           complete.MustCompile(":section [nodisp:]_", nil),
		"c/urrent/v/alue":    complete.MustCompile(":currentvalue [abs:]", nil),
		"len/gth":            complete.MustCompile(":length [deformed:]", nil),
		"are/a":              complete.MustCompile(":area [deformed:]", nil),
		"an/alysis": complete.MustCompile(":analysis [period:$PERIOD] [all:] [solver:$SOLVER] [eps:_] [nlgeom:] [nlmat:] [step:_] [noinit:] [wait:] [post:_] [sects:_] [comp:_] [z:_] _",
			map[string][]string{
				"PERIOD": []string{"l", "x", "y"},
				"SOLVER": []string{"LLS", "CRS", "CG", "PCG"},
			}),
		"f/ilter": complete.MustCompile(":filter $CONDITION",
			map[string][]string{
				"CONDITION": []string{"//", "TT", "on", "adjoin", "cv"},
			}),
		"ra/nge":      complete.MustCompile(":range", nil),
		"h/eigh/t/":   complete.MustCompile(":height _ _", nil),
		"h/eigh/t+/":  complete.MustCompile(":height+", nil),
		"h/eigh/t-/":  complete.MustCompile(":height-", nil),
		"s/et/ang/le": complete.MustCompile(":setangle _ _", nil),
		"view/": complete.MustCompile(":view $DIRECTION",
			map[string][]string{
				"DIRECTION": []string{"top", "front", "back", "right", "left"},
			}),
		"paper/": complete.MustCompile(":paper $NAME",
			map[string][]string{
				"NAME": []string{"a3tate", "a3yoko", "a4tate", "a4yoko"},
			}),
		"sec/tion/+/": complete.MustCompile(":section+ _", nil),
		"col/or": complete.MustCompile(":color $NAME",
			map[string][]string{
				"NAME": []string{"n", "sect", "rate", "white", "mono", "strong"},
			}),
		"ex/tractarclm":  complete.MustCompile(":extractarclm", nil),
		"s/aveas/ar/clm": complete.MustCompile(":saveasarclm", nil),
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

func ExMode(stw ExModer, command string) error {
	if command == ":." {
		return ExMode(stw, stw.LastExCommand())
	}
	exmodech := make(chan interface{})
	exmodeend := make(chan int)
	stw.SetLastExCommand(command)
	if !strings.Contains(command, "|") {
		err := exCommand(stw, command, false, exmodech, exmodeend)
		if err != nil {
			switch e := err.(type) {
			case NotRedraw:
				stw.History(e.Message())
				return err
			case Messager:
				stw.History(e.Message())
				return nil
			default:
				return err
			}
		} else {
			return nil
		}
	}
	excms := strings.Split(command, "|")
	defer emptyExModech(exmodech, exmodeend)
	for _, com := range excms {
		err := exCommand(stw, com, true, exmodech, exmodeend)
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

func exCommand(stw ExModer, command string, pipe bool, exmodech chan interface{}, exmodeend chan int) error {
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
	frame := stw.Frame()
	if narg < 2 {
		fn = ""
	} else {
		if frame == nil {
			fn = CompleteFileName(args[1], "", stw.Recent())[0]
		} else {
			fn = CompleteFileName(args[1], frame.Path, stw.Recent())[0]
		}
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
				err = OpenFile(stw, sfn, readrc)
				if err != nil {
					return err
				}
			} else {
				err := OpenFile(stw, fn, readrc)
				if err != nil {
					return err
				}
			}
		} else {
			Reload(stw)
		}
	case "quit":
		if usage {
			return Usage(":quit")
		}
		stw.Close(bang)
	case "resource":
		if usage {
			return Usage(":resource filename")
		}
		ReadResource(stw, fn)
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
		os.MkdirAll(fn, 0755)
	case "#":
		if usage {
			return Usage(":#")
		}
		ShowRecent(stw)
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
	case "cross":
		if usage {
			return Usage(":cross h b tw tf h b tw tf")
		}
		if narg < 9 {
			return NotEnoughArgs(":cross")
		}
		al, err := NewCROSS(args[1:9])
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
	case "tweak":
		if usage {
			return Usage(":tweak h b tw tf")
		}
		if narg < 5 {
			return NotEnoughArgs(":tweak")
		}
		al, err := NewTWEAK(args[1:5])
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
	case "cweak":
		if usage {
			return Usage(":cweak h b tw tf")
		}
		if narg < 5 {
			return NotEnoughArgs(":cweak")
		}
		al, err := NewCWEAK(args[1:5])
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
	case "angle":
		if usage {
			return Usage(":angle h b tw tf")
		}
		if narg < 5 {
			return NotEnoughArgs(":angle")
		}
		al, err := NewANGLE(args[1:5])
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
		Snapshot(stw)
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
	case "agc":
		if len(frame.Arcs) == 0 {
			return nil
		}
		for _, arc := range frame.Arcs {
			size := int(arc.Radius/0.1)*2 + 1
			coords := make([]float64, size)
			val := float64(int(arc.Radius/0.1)) * -0.1
			for i := 0; i < size; i++ {
				coords[i] = val
				val += 0.1
			}
			arc.DivideAtLocalAxis(0, coords, EPS)
		}
	case "redraw":
		stw.Redraw()
	case "write":
		if usage {
			return Usage(":write")
		}
		if fn == "" {
			SaveFile(stw, frame.Path)
		} else {
			if bang || (!FileExists(fn) || stw.Yn("Save", "上書きしますか")) {
				err := SaveFile(stw, fn)
				if err != nil {
					return err
				}
				if fn != frame.Path {
					Copylsts(stw, fn, bang)
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
				os.MkdirAll(filepath.Dir(fn), 0755)
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
				err = OpenFile(stw, fn, readrc)
				if err != nil {
					return err
				}
				Copylsts(stw, fn, bang)
			} else {
				err = SaveFile(stw, fn)
			}
			if err != nil {
				return err
			}
			if fn != frame.Path {
				Copylsts(stw, fn, bang)
			}
			Rebase(stw, fn)
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
			err := SaveFile(stw, fn)
			if err != nil {
				return err
			}
			if fn != frame.Path {
				Copylsts(stw, fn, bang)
			}
			Rebase(stw, fn)
			Snapshot(stw)
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
		err := stw.AddTag(frame, name, bang)
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
		f, err := stw.Checkout(name)
		if err != nil {
			return err
		}
		stw.SetFrame(f)
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
				ReadAll(stw)
			case "$data":
				for _, ext := range []string{".inl", ".ihx", ".ihy"} {
					err := frame.ReadData(Ce(frame.Path, ext))
					if err != nil {
						ErrorMessage(stw, err, ERROR)
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
						ErrorMessage(stw, err, ERROR)
					}
				}
			default:
				err := ReadFile(stw, fn)
				if err != nil {
					return err
				}
			}
			return nil
		}
		fn = CompleteFileName(args[2], frame.Path, stw.Recent())[0]
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
		case abbrev.For("v/ibration/e/igenmode", t):
			err := frame.ReadVibrationalEigenmode(fn)
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
			if cw, ok := stw.(Commander); ok {
				err := ReadPgp(cw, fn)
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("Window doesn't implement Commander interface")
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
			Snapshot(stw)
			if err != nil {
				return err
			}
		}
	case "propsect":
		if usage {
			return Usage(":propsect filename")
		}
		err := frame.AddPropAndSect(fn, true)
		Snapshot(stw)
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
			return Usage(":writereaction {-confed} {-nodeset=} filename direction")
		}
		if narg < 3 {
			return NotEnoughArgs(":wr")
		}
		tmp, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			return err
		}
		if _, ok := argdict["CONFED"]; ok {
			SelectConfed(stw)
		}
		if nname, ok := argdict["NODESET"]; ok {
			if ns, ok := frame.NodeSet[nname]; ok {
				stw.SelectNode(ns)
			} else {
				return fmt.Errorf("nodeset {} is not defined", nname)
			}
		}
		if !stw.NodeSelected() {
			return errors.New(":writereaction: no selected node")
		}
		ns := stw.SelectedNodes()
		sort.Sort(NodeByNum{ns})
		err = WriteReaction(fn, ns, int(tmp), frame.Show.Unit[0])
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
	case "writefig2":
		if usage {
			return Usage("writefig2 filename")
		}
		if fn == "" {
			fn = Ce(frame.Path, ".fig2")
		}
		err := WriteFig2(fn, frame.View, frame.Show)
		if err != nil {
			return err
		}
	case "writelst":
		if usage {
			return Usage("writelst filename")
		}
		if fn == "" {
			fn = Ce(frame.Path, ".lst")
		}
		snum := 0
		sects := make([]*Sect, len(frame.Sects))
		for _, sec := range frame.Sects {
			if sec.Num < 100 || sec.Num > 900 {
				continue
			}
			sects[snum] = sec
			snum++
		}
		sects = sects[:snum]
		sort.Sort(SectByNum{sects})
		err := WriteLst(fn, sects)
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
			return Usage(":weightcopy {-dir=} {-si}")
		}
		dir := stw.Home()
		if d, ok := argdict["DIR"]; ok {
			dir = os.ExpandEnv(d)
		}
		wgt := filepath.Join(dir, "hogtxt.wgt")
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
		err := ReadFig2(stw, fn)
		if err != nil {
			return err
		}
	case "fig2print":
		if usage {
			return Usage(":fig2print")
		}
		if fn == "" {
			fn = Ce(frame.Path, ".fig2")
		}
		cmd := exec.Command("fig2", frame.Path, fn)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return err
		}
	case "sectionlist":
		if usage {
			return Usage(":sectionlist {-scale=0.025}")
		}
		scale := 0.025
		if v, ok := argdict["SCALE"]; ok {
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}
			scale = val
		}
		if fn == "" {
			fn = filepath.Join(filepath.Dir(frame.Path), "sectionlist.tex")
		}
		err := SectionList(fn, frame, scale)
		if err != nil {
			return err
		}
	case "svg":
		if usage {
			return Usage(":svg {-size=a4tate} filename")
		}
		if name, ok := argdict["SIZE"]; ok {
			switch strings.ToUpper(name) {
			case "A4TATE":
				stw.SetPaperSize(A4_TATE)
			case "A4YOKO":
				stw.SetPaperSize(A4_YOKO)
			case "A3TATE":
				stw.SetPaperSize(A3_TATE)
			case "A3YOKO":
				stw.SetPaperSize(A3_YOKO)
			}
		}
		w, h := PaperSizemm(stw.PaperSize())
		err := PrintSVG(frame, stw.TextBoxes(), fn, w, h)
		if err != nil {
			return err
		}
	case "png":
		if usage {
			return Usage(":png filename")
		}
		w, h := stw.GetCanvasSize()
		err := PrintPNG(frame, fn, w, h)
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
	case "plan":
		if usage {
			return Usage(":plan filename {-floor=1} {-scale=1000} {-height=250} {-axissize=300}")
		}
		floor := 1
		scale := 1000.0
		textheight := 250.0
		axissize := 300.0
		if f, ok := argdict["FLOOR"]; ok {
			val, err := strconv.ParseInt(f, 10, 64)
			if err != nil {
				return err
			}
			floor = int(val)
		}
		if s, ok := argdict["SCALE"]; ok {
			val, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return err
			}
			scale = val
		}
		if h, ok := argdict["HEIGHT"]; ok {
			val, err := strconv.ParseFloat(h, 64)
			if err != nil {
				return err
			}
			textheight = val
		}
		if a, ok := argdict["AXISSIZE"]; ok {
			val, err := strconv.ParseFloat(a, 64)
			if err != nil {
				return err
			}
			axissize = val
		}
		if fn == "" {
			return fmt.Errorf("no file name")
		}
		err := frame.WriteDxfPlan(fn, floor, scale, textheight, axissize)
		if err != nil {
			return err
		}
	case "jiku":
		if usage {
			return Usage(":jiku filename {-name=} {-side=1} {-scale=1000} {-height=250} {-axissize=300}")
		}
		name := ""
		axis := 0
		min := -1.0
		max := 1.0
		side := 1
		scale := 1000.0
		textheight := 250.0
		axissize := 300.0
		if n, ok := argdict["NAME"]; ok {
			name = n
		}
		if name == "" {
			return fmt.Errorf("kijun name is required")
		}
		if k, ok := frame.Kijuns[strings.ToLower(name)]; ok {
			d := k.Direction()
			if d[0] < d[1] {
				axis = 0
			} else {
				axis = 1
			}
			min = k.Start[axis] - 0.1
			max = k.Start[axis] + 0.1
		} else {
			return fmt.Errorf("kijun %s doesn't exist", name)
		}
		if s, ok := argdict["SIDE"]; ok {
			val, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return err
			}
			side = int(val)
		}
		if s, ok := argdict["SCALE"]; ok {
			val, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return err
			}
			scale = val
		}
		if h, ok := argdict["HEIGHT"]; ok {
			val, err := strconv.ParseFloat(h, 64)
			if err != nil {
				return err
			}
			textheight = val
		}
		if a, ok := argdict["AXISSIZE"]; ok {
			val, err := strconv.ParseFloat(a, 64)
			if err != nil {
				return err
			}
			axissize = val
		}
		if fn == "" {
			return fmt.Errorf("no file name")
		}
		err := frame.WriteDxfCrosssection(fn, axis, min, max, side, scale, textheight, axissize)
		if err != nil {
			return err
		}
	case "crosssection":
		if usage {
			return Usage(":crosssection filename {-axis=2} {-min=-1.0} {-max=1.0} {-side=1} {-scale=1000} {-height=250} {-axissize=300}")
		}
		axis := 2
		min := -1.0
		max := 1.0
		side := 1
		scale := 1000.0
		textheight := 250.0
		axissize := 300.0
		if a, ok := argdict["AXIS"]; ok {
			val, err := strconv.ParseInt(a, 10, 64)
			if err != nil {
				return err
			}
			axis = int(val)
		}
		if m, ok := argdict["MIN"]; ok {
			val, err := strconv.ParseFloat(m, 64)
			if err != nil {
				return err
			}
			min = val
		}
		if m, ok := argdict["MAX"]; ok {
			val, err := strconv.ParseFloat(m, 64)
			if err != nil {
				return err
			}
			max = val
		}
		if s, ok := argdict["SIDE"]; ok {
			val, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return err
			}
			side = int(val)
		}
		if s, ok := argdict["SCALE"]; ok {
			val, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return err
			}
			scale = val
		}
		if h, ok := argdict["HEIGHT"]; ok {
			val, err := strconv.ParseFloat(h, 64)
			if err != nil {
				return err
			}
			textheight = val
		}
		if a, ok := argdict["AXISSIZE"]; ok {
			val, err := strconv.ParseFloat(a, 64)
			if err != nil {
				return err
			}
			axissize = val
		}
		if fn == "" {
			return fmt.Errorf("no file name")
		}
		err := frame.WriteDxfCrosssection(fn, axis, min, max, side, scale, textheight, axissize)
		if err != nil {
			return err
		}
	case "kunst":
		if narg < 2 {
			return NotEnoughArgs(":kunst")
		}
		switch args[1] {
		case "slab":
			sect := frame.Sects[502]
			etype := GIRDER
			var n1, n2 *Node
			var coord1, coord2 []float64
			for _, el := range frame.Elems {
				if el.Etype == SLAB {
					coord1 = el.EdgeDividingPoint(0, 0.25)
					coord2 = el.EdgeDividingPoint(2, 0.75)
					n1, _ = frame.CoordNode(coord1[0], coord1[1], coord1[2], EPS)
					n2, _ = frame.CoordNode(coord2[0], coord2[1], coord2[2], EPS)
					frame.AddLineElem(-1, []*Node{n1, n2}, sect, etype)
					coord1 = el.EdgeDividingPoint(1, 0.25)
					coord2 = el.EdgeDividingPoint(3, 0.75)
					n1, _ = frame.CoordNode(coord1[0], coord1[1], coord1[2], EPS)
					n2, _ = frame.CoordNode(coord2[0], coord2[1], coord2[2], EPS)
					frame.AddLineElem(-1, []*Node{n1, n2}, sect, etype)
					coord1 = el.EdgeDividingPoint(0, 0.5)
					coord2 = el.EdgeDividingPoint(2, 0.5)
					n1, _ = frame.CoordNode(coord1[0], coord1[1], coord1[2], EPS)
					n2, _ = frame.CoordNode(coord2[0], coord2[1], coord2[2], EPS)
					frame.AddLineElem(-1, []*Node{n1, n2}, sect, etype)
					coord1 = el.EdgeDividingPoint(1, 0.5)
					coord2 = el.EdgeDividingPoint(3, 0.5)
					n1, _ = frame.CoordNode(coord1[0], coord1[1], coord1[2], EPS)
					n2, _ = frame.CoordNode(coord2[0], coord2[1], coord2[2], EPS)
					frame.AddLineElem(-1, []*Node{n1, n2}, sect, etype)
					coord1 = el.EdgeDividingPoint(0, 0.75)
					coord2 = el.EdgeDividingPoint(2, 0.25)
					n1, _ = frame.CoordNode(coord1[0], coord1[1], coord1[2], EPS)
					n2, _ = frame.CoordNode(coord2[0], coord2[1], coord2[2], EPS)
					frame.AddLineElem(-1, []*Node{n1, n2}, sect, etype)
					coord1 = el.EdgeDividingPoint(1, 0.75)
					coord2 = el.EdgeDividingPoint(3, 0.25)
					n1, _ = frame.CoordNode(coord1[0], coord1[1], coord1[2], EPS)
					n2, _ = frame.CoordNode(coord2[0], coord2[1], coord2[2], EPS)
					frame.AddLineElem(-1, []*Node{n1, n2}, sect, etype)
				}
			}
		case "spline":
			d := 1
			if dv, ok := argdict["D"]; ok {
				switch dv {
				case "0", "x", "X":
					d = 0
				case "1", "y", "Y":
					d = 1
				case "2", "z", "Z":
					d = 2
				}
			}
			z := 2
			if zv, ok := argdict["Z"]; ok {
				switch zv {
				case "0", "x", "X":
					z = 0
				case "1", "y", "Y":
					z = 1
				case "2", "z", "Z":
					z = 2
				}
			}
			ndiv := 4
			if n, ok := argdict["N"]; ok {
				val, err := strconv.ParseInt(n, 10, 64)
				if err == nil {
					ndiv = int(val)
				}
			}
			var ns []*Node
			ns = currentnode(stw, exmodech, exmodeend)
			if len(ns) == 0 {
				els := currentelem(stw, exmodech, exmodeend)
				if len(els) == 0 {
					return fmt.Errorf("no nodes or elems selected")
				}
				ns = frame.ElemToNode(els...)
			}
			coords, err := splinecoord(ns, d, z, ndiv)
			if err != nil {
				return err
			}
			var n0, n1 *Node
			n0, _ = frame.CoordNode(coords[0][0], coords[0][1], coords[0][2], EPS)
			sect := frame.Sects[503]
			etype := GIRDER
			for _, c := range coords[1:] {
				n1, _ = frame.CoordNode(c[0], c[1], c[2], EPS)
				frame.AddLineElem(-1, []*Node{n0, n1}, sect, etype)
				n0 = n1
			}
		case "line":
			d := 1
			if dv, ok := argdict["D"]; ok {
				switch dv {
				case "0", "x", "X":
					d = 0
				case "1", "y", "Y":
					d = 1
				case "2", "z", "Z":
					d = 2
				}
			}
			var ns []*Node
			ns = currentnode(stw, exmodech, exmodeend)
			if len(ns) == 0 {
				els := currentelem(stw, exmodech, exmodeend)
				if len(els) == 0 {
					return fmt.Errorf("no nodes or elems selected")
				}
				ns = frame.ElemToNode(els...)
			}
			switch d {
			case 0:
				sort.Sort(NodeByXCoord{ns})
			case 1:
				sort.Sort(NodeByYCoord{ns})
			case 2:
				sort.Sort(NodeByZCoord{ns})
			}
			n0 := ns[0]
			var n1 *Node
			sect := frame.Sects[503]
			etype := GIRDER
			for _, n := range ns[1:] {
				n1 = n
				frame.AddLineElem(-1, []*Node{n0, n1}, sect, etype)
				n0 = n1
			}
		}
	case "spline":
		if usage {
			return Usage(":spline {-d=1} {-z=2} {-scale=1000.0} {-n=4} {-original} filename")
		}
		f := Ce(fn, ".dxf")
		d := 1
		if dv, ok := argdict["D"]; ok {
			switch dv {
			case "0", "x", "X":
				d = 0
			case "1", "y", "Y":
				d = 1
			case "2", "z", "Z":
				d = 2
			}
		}
		z := 2
		if zv, ok := argdict["Z"]; ok {
			switch zv {
			case "0", "x", "X":
				z = 0
			case "1", "y", "Y":
				z = 1
			case "2", "z", "Z":
				z = 2
			}
		}
		scale := 1000.0
		if s, ok := argdict["SCALE"]; ok {
			val, err := strconv.ParseFloat(s, 64)
			if err == nil {
				scale = val
			}
		}
		ndiv := 4
		if n, ok := argdict["N"]; ok {
			val, err := strconv.ParseInt(n, 10, 64)
			if err == nil {
				ndiv = int(val)
			}
		}
		original := false
		if _, ok := argdict["ORIGINAL"]; ok {
			original = true
		}
		var ns []*Node
		ns = currentnode(stw, exmodech, exmodeend)
		if len(ns) == 0 {
			els := currentelem(stw, exmodech, exmodeend)
			if len(els) == 0 {
				return fmt.Errorf("no nodes or elems selected")
			}
			ns = frame.ElemToNode(els...)
		}
		err := createspline(f, ns, d, z, scale, ndiv, original)
		if err != nil {
			return err
		}
	case "pdf":
		if usage {
			return Usage(":pdf filename")
		}
		paper := stw.PaperSize()
		w, h := PaperSizemm(paper)
		n, pl := PaperName(paper)
		pdf, err := NewPDFCanvas(n, pl, w, h)
		if err != nil {
			return err
		}
		v := frame.View.Copy()
		frame.View.Center[0] = 0.5 * w * 4
		frame.View.Center[1] = 0.5 * h * 4
		pdf.Draw(frame, stw.TextBoxes())
		err = pdf.SaveAs(Ce(fn, ".pdf"))
		frame.View = v
		if err != nil {
			return err
		}
	case "tex":
		if usage {
			return Usage(":tex filename")
		}
		paper := stw.PaperSize()
		w, h := PaperSizemm(paper)
		pgf := NewPGFCanvas(w, h)
		v := frame.View.Copy()
		frame.View.Center[0] = 0.5 * w
		frame.View.Center[1] = 0.5 * h
		pgf.Draw(frame, stw.TextBoxes())
		err := pgf.SaveAs(Ce(fn, ".tex"))
		frame.View = v
		if err != nil {
			return err
		}
	case "check":
		if usage {
			return Usage(":check")
		}
		CheckFrame(stw)
		return Message("CHECKED")
	case "check_london":
		if usage {
			return Usage(":check_london")
		}
		linesect := 201
		platesect := 703
		lineels := make([]*Elem, len(frame.Elems))
		plateels := make([]*Elem, len(frame.Elems))
		lnum := 0
		pnum := 0
		for _, el := range frame.Elems {
			switch el.Sect.Num {
			case linesect:
				lineels[lnum] = el
				lnum++
			case platesect:
				plateels[pnum] = el
				pnum++
			}
		}
		if lnum == 0 || pnum == 0 {
			return nil
		}
		lineelems := lineels[:lnum]
		plateelems := plateels[:pnum]
		for _, lel := range lineelems {
			for _, pel := range plateelems {
				b, _ := IntersectionLinePlate(lel, pel, EPS)
				if b {
					fmt.Printf("ELEM: %d, %d\n", lel.Num, pel.Num)
				}
			}
		}
	case "nodeduplication":
		if usage {
			return Usage(":nodeduplication")
		}
		stw.Deselect()
		nodes := frame.NodeDuplication(EPS)
		if len(nodes) != 0 {
			frame.ReplaceNode(nodes)
			Snapshot(stw)
		}
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
				es[enum] = k
				enum++
			}
			stw.SelectElem(es[:enum])
		}
		return Message(m.String())
	case "nodenoreference":
		if usage {
			return Usage(":nodenoreference")
		}
		ns := frame.NodeNoReference()
		if len(ns) != 0 {
			for _, n := range ns {
				frame.DeleteNode(n.Num)
			}
		}
		stw.Deselect()
		Snapshot(stw)
	case "mergenode":
		if usage {
			return Usage(":mergenode {-eps=1e-4}")
		}
		eps := EPS
		if e, ok := argdict["EPS"]; ok {
			v, err := strconv.ParseFloat(e, 64)
			if err != nil {
				return err
			}
			eps = v
		}
		ns := currentnode(stw, exmodech, exmodeend)
		if len(ns) < 2 {
			return fmt.Errorf("not enough nodes selected")
		}
		nmap := make(map[int]*Node, len(ns))
		for _, n := range ns {
			nmap[n.Num] = n
		}
		nodes := NodeDuplication(nmap, eps)
		for k, v := range nodes {
			frame.MergeNode([]*Node{k, v})
		}
		Snapshot(stw)
	case "nodesort":
		if usage {
			return Usage(":nodesort [x,y,z,0,1,2]")
		}
		if narg < 2 {
			return NotEnoughArgs(":nodesort")
		}
		bw := frame.BandWidth()
		stw.History(fmt.Sprintf("並び替え前: %d", bw))
		ns := func(d int) {
			bw, err := frame.NodeSort(d)
			if err != nil {
				stw.History("並び替えエラー")
			}
			stw.History(fmt.Sprintf("並び替え後: %d (%s方向)", bw, []string{"X", "Y", "Z"}[d]))
			Snapshot(stw)
		}
		switch strings.ToUpper(args[1]) {
		case "X", "0":
			ns(0)
		case "Y", "1":
			ns(1)
		case "Z", "2":
			ns(2)
		}
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
					fmt.Printf("LAP: %3d / %3d\r", nlap, l)
				case err = <-frame.Endch:
					if err != nil {
						stw.CurrentLap("Error", nlap, l)
						ErrorMessage(stw, err, ERROR)
					} else {
						stw.CurrentLap("Completed", nlap, l)
					}
					stw.Redraw()
					break iallloop
				}
			}
		}()
		Snapshot(stw)
	case "srcal":
		if usage {
			return Usage(":srcal {-verbose} {-fbold} {-torsion} {-noreload} {-qfact=2.0} {-wfact=2.0} {-bfact=1.0} {-skipshort} {-temporary} {-moeshiro} {-sekisetsu} filename")
		}
		var m bytes.Buffer
		cond := NewCondition()
		if _, ok := argdict["FBOLD"]; ok {
			m.WriteString("Fb: old\n")
			cond.FbOld = true
		}
		var otp string
		if fn == "" {
			otp = frame.Path
		} else {
			otp = fn
		}
		reload := true
		if _, ok := argdict["NORELOAD"]; ok {
			reload = false
		}
		if reload {
			err := ReadFile(stw, Ce(otp, ".lst"))
			if err != nil {
				return err
			}
		}
		if qf, ok := argdict["QFACT"]; ok {
			val, err := strconv.ParseFloat(qf, 64)
			if err == nil {
				cond.Qfact = val
			}
		}
		m.WriteString(fmt.Sprintf("QFACT: %.3f\n", cond.Qfact))
		if wf, ok := argdict["WFACT"]; ok {
			val, err := strconv.ParseFloat(wf, 64)
			if err == nil {
				cond.Wfact = val
			}
		}
		m.WriteString(fmt.Sprintf("WFACT: %.3f\n", cond.Wfact))
		if wf, ok := argdict["BFACT"]; ok {
			val, err := strconv.ParseFloat(wf, 64)
			if err == nil {
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
		if _, ok := argdict["V"]; ok {
			argdict["VERBOSE"] = ""
		}
		if _, ok := argdict["VERBOSE"]; ok {
			m.WriteString("VERBOSE")
			cond.Verbose = true
		}
		if _, ok := argdict["TORSION"]; ok {
			m.WriteString("TORSION")
			cond.RCTorsion = true
		}
		frame.SectionRateCalculation(otp, "L", "X", "X", "Y", "Y", -1.0, cond)
		return Message(m.String())
	case "srcalangle":
		if usage {
			return Usage(":srcalangle {-angle=0[deg]} {-fact=1.0}")
		}
		cond := NewCondition()
		alpha := []float64{1.0, 1.0}
		var otpfn string
		if fn == "" {
			otpfn = frame.Path
		} else {
			otpfn = fn
		}
		angle := 0.0
		if f, ok := argdict["FACT"]; ok {
			val, err := strconv.ParseFloat(f, 64)
			if err != nil {
				return err
			}
			cond.Nfact = val
			alpha[0] = val
			alpha[1] = val
		}
		if f, ok := argdict["XFACT"]; ok {
			val, err := strconv.ParseFloat(f, 64)
			if err != nil {
				return err
			}
			cond.Nfact = val
			alpha[0] = val
		}
		if f, ok := argdict["YFACT"]; ok {
			val, err := strconv.ParseFloat(f, 64)
			if err != nil {
				return err
			}
			cond.Nfact = val
			alpha[1] = val
		}
		if a, ok := argdict["ANGLE"]; ok {
			val, err := strconv.ParseFloat(a, 64)
			if err != nil {
				return err
			}
			angle = val
		}
		if _, ok := argdict["V"]; ok {
			argdict["VERBOSE"] = ""
		}
		if _, ok := argdict["VERBOSE"]; ok {
			cond.Verbose = true
		}
		reload := true
		if _, ok := argdict["NORELOAD"]; ok {
			reload = false
		}
		if reload {
			ReadFile(stw, Ce(otpfn, ".lst"))
		}
		if qf, ok := argdict["QFACT"]; ok {
			val, err := strconv.ParseFloat(qf, 64)
			if err == nil {
				cond.Qfact = val
			}
		}
		if wf, ok := argdict["WFACT"]; ok {
			val, err := strconv.ParseFloat(wf, 64)
			if err == nil {
				cond.Wfact = val
			}
		}
		els := stw.SelectedElems()
		sort.Sort(ElemByNum{els})
		maxrateelem := make(map[int][]*Elem)
		var otp,tex,rat bytes.Buffer
		otp.WriteString("断面算定 \"S,RC,SRC\" 長期,短期,終局\n")
		otp.WriteString("使用ファイル\n")
		otp.WriteString(fmt.Sprintf("入力データ               =%s\n", frame.DataFileName["L"]))
		otp.WriteString(fmt.Sprintf("鉛直荷重時解析結果       =%s\n", frame.ResultFileName["L"]))
		otp.WriteString(fmt.Sprintf("水平荷重時解析結果 X方向 =%s\n", frame.ResultFileName["X"]))
		otp.WriteString(fmt.Sprintf("                   Y方向 =%s\n", frame.ResultFileName["Y"]))
		otp.WriteString(fmt.Sprintf("仮定断面                 =%s\n", frame.LstFileName))
		otp.WriteString("\n単位系 tf(kN),tfm(kNm)\n")
		otp.WriteString("\nAs:鉄骨 Ar:主筋 Ac:コンクリート Ap:ＰＣストランド\n")
		otp.WriteString("N:軸力 Q:せん断力 Mt:ねじりモーメント M:曲げモーメント\n")
		otp.WriteString("添字 i:始端 j:終端 c:中央\n")
		otp.WriteString("a:許容 u:終局\n")
		for _, el := range els {
			if el == nil {
				continue
			}
			el.Condition = cond.Snapshot()
			al, _, err := el.GetSectionRate()
			if err != nil {
				fmt.Println(err)
				continue
			}
			_, t, err := el.OutputRateInformation("L", "X", "X", "Y", "Y", -1.0, alpha, angle)
			if err != nil {
				fmt.Println(err)
				continue
			}
			otp.WriteString(t)
			rat.WriteString(el.OutputMaxRate())
			switch el.Etype {
			case COLUMN, GIRDER:
				if mels, ok := maxrateelem[al.Num()]; ok {
					for ind, pos := range []int{0, 1, 3, 4} {
						if el.MaxRate[pos] > mels[ind].MaxRate[pos] {
							maxrateelem[al.Num()][ind] = el
						}
					}
				} else {
					maxrateelem[al.Num()] = []*Elem{el, el, el, el}
				}
			case BRACE, WBRACE, SBRACE:
				if mels, ok := maxrateelem[al.Num()]; ok {
					for ind, pos := range []int{0, 1} {
						if el.MaxRate[pos] > mels[ind].MaxRate[pos] {
							maxrateelem[al.Num()][ind] = el
						}
					}
				} else {
					maxrateelem[al.Num()] = []*Elem{el, el}
				}
			}
		}
		otp.WriteString("==========================================================================================================================================================================================================\n各断面種別の許容,終局曲げ安全率の最大値\n\n")
		keys := make([]int, len(maxrateelem))
		i := 0
		for k := range maxrateelem {
			keys[i] = k
			i++
		}
		sort.Ints(keys)
		maxql := 0.0
		maxqs := 0.0
		maxml := 0.0
		maxms := 0.0
		maxratestring := make([]string, 0)
		for _, k := range keys {
			otp.WriteString(fmt.Sprintf("断面記号: %d %s   As=   0.00[cm2] Ar=   0.00[cm2] Ac=    0.00[cm2] MAX:", k, frame.Sects[k].Allow.TypeString()))
			els := maxrateelem[k]
			switch els[0].Etype {
			case COLUMN, GIRDER:
				otp.WriteString(fmt.Sprintf("Q/QaL=%.5f Q/QaS=%.5f M/MaL=%.5f M/MaS=%.5f\n", els[0].MaxRate[0], els[1].MaxRate[1], els[2].MaxRate[3], els[3].MaxRate[4]))
				if els[0].MaxRate[0] > maxql {
					maxql = els[0].MaxRate[0]
				}
				if els[1].MaxRate[1] > maxqs {
					maxqs = els[1].MaxRate[1]
				}
				if els[2].MaxRate[3] > maxml {
					maxml = els[2].MaxRate[3]
				}
				if els[3].MaxRate[4] > maxms {
					maxms = els[3].MaxRate[4]
				}
				sort.Slice(els, func(i, j int) bool {
					ri, _ := els[i].RateMax(nil)
					rj, _ := els[j].RateMax(nil)
					return ri > rj
				})
				maxratestring = append(maxratestring, els[0].SrcalTex)
			case BRACE, WBRACE, SBRACE:
				otp.WriteString(fmt.Sprintf("Q/QaL=%.5f Q/QaS=%.5f\n", els[0].MaxRate[0], els[1].MaxRate[1]))
				if els[0].MaxRate[0] > maxql {
					maxql = els[0].MaxRate[0]
				}
				if els[1].MaxRate[1] > maxqs {
					maxqs = els[1].MaxRate[1]
				}
				sort.Slice(els, func(i, j int) bool {
					ri, _ := els[i].RateMax(nil)
					rj, _ := els[j].RateMax(nil)
					return ri > rj
				})
				e2, _ := els[0].Brother()
				if e2 == nil {
					maxratestring = append(maxratestring, els[0].SrcalTex)
				} else {
					if els[0].Eldest {
						maxratestring = append(maxratestring, els[0].SrcalTex)
						maxratestring = append(maxratestring, e2.SrcalTex)
					} else {
						maxratestring = append(maxratestring, e2.SrcalTex)
						maxratestring = append(maxratestring, els[0].SrcalTex)
					}
				}
			}
		}
		for i, str := range maxratestring {
			if i%2 == 0 {
				tex.WriteString("\\scalebox{0.9}[1]{\n{\\footnotesize\n\\begin{tabular}{l r r r r r r r r r r} \\hline\\\\\n")
			}
			tex.WriteString(str)
			if i%2 == 1 {
				tex.WriteString("\\end{tabular}\n}\n}\n\\newpage\n")
			}
		}
		if len(maxratestring)%2 != 0 {
			tex.WriteString("\\end{tabular}\n}\n}\n\n")
		}
		tex.WriteString("\\paragraph{各断面種別の許容,終局曲げ安全率の最大値}\n\n{\\small\n\\begin{tabular}{l l l l l}\\\\\n")
		for _, k := range keys {
			tex.WriteString(fmt.Sprintf("断面記号: %d %s & ", k, frame.Sects[k].Allow.TypeString()))
			els := maxrateelem[k]
			switch els[0].Etype {
			case COLUMN, GIRDER:
				tex.WriteString(fmt.Sprintf("$Q/Q_{aL}$=%.5f & $Q/Q_{aS}$=%.5f & $M/M_{aL}$=%.5f & $M/M_{aS}$=%.5f\\\\\n", els[0].MaxRate[0], els[1].MaxRate[1], els[2].MaxRate[3], els[3].MaxRate[4]))
			case BRACE, WBRACE, SBRACE:
				tex.WriteString(fmt.Sprintf("$Q/Q_{aL}$=%.5f & $Q/Q_{aS}$=%.5f\\\\\n", els[0].MaxRate[0], els[1].MaxRate[1]))
			}
		}
		tex.WriteString("\\end{tabular}\n}\n")
		otp.WriteString(fmt.Sprintf("\n安全率の最大値\n Q/QaL=%7.5f Q/QaS=%7.5f\n M/MaL=%7.5f M/MaS=%7.5f\n", maxql, maxqs, maxml, maxms))
		tex.WriteString(fmt.Sprintf("\\vspace{10mm}\n\\paragraph{安全率の最大値}\n{\\small\n\\begin{tabular}{l l}\\\\\n $Q/Q_{aL}$=%7.5f & $Q/Q_{aS}$=%7.5f\\\\\n $M/M_{aL}$=%7.5f & $M/M_{aS}$=%7.5f\\\\\n\\end{tabular}\n}\n\\newpage\n", maxql, maxqs, maxml, maxms))
		otp.WriteString("==========================================================================================================================================================================================================\n各断面種別の入力情報の確認\n\n")
		otp.WriteString("                              A[cm2]      Ix[cm4]      Iy[cm4]       J[cm4]\n")
		otp.WriteString("                              t[cm]\n")
		otp.WriteString(frame.CheckLst(keys))
		w, err := os.Create(Ce(otpfn, ".tst2"))
		defer w.Close()
		if err != nil {
			return err
		}
		otp = AddCR(otp)
		otp.WriteTo(w)
		wtex, err := os.Create(filepath.Join(filepath.Dir(frame.Path), "4_3angle.tex"))
		defer wtex.Close()
		if err != nil {
			return err
		}
		tex = AddCR(tex)
		tex.WriteTo(wtex)
		wrat, err := os.Create(Ce(otpfn, ".rat"))
		defer wrat.Close()
		if err != nil {
			return err
		}
		rat = AddCR(rat)
		rat.WriteTo(wrat)
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
		if al := frame.Sects[int(tmp)].Allow; al != nil {
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
			filename := filepath.Join(filepath.Dir(frame.Path), "nmi.txt")
			if o, ok := argdict["OUTPUT"]; ok {
				if o != "" {
					m.WriteString(fmt.Sprintf("OUTPUT: %s", o))
					filename = o
				}
			}
			if p, ok := argdict["PERIOD"]; ok {
				if p != "" {
					p = strings.ToUpper(p)
					m.WriteString(fmt.Sprintf("PERIOD: %s", p))
					cond.Period = p
				}
			}
			yield := make([]float64, 12)
			for i := 0; i < 2; i++ {
				if i == 0 {
					otp.WriteString("STRONG\n")
					cond.Strong = true
				} else {
					otp.WriteString("WEAK\n")
					cond.Strong = false
				}
				switch al.(type) {
				default:
					return nil
				case *RCColumn:
					nmax := al.(*RCColumn).Nmax(cond)
					nmin := al.(*RCColumn).Nmin(cond)
					for i := 0; i <= ndiv; i++ {
						cond.N = nmax - float64(i)*(nmax-nmin)/float64(ndiv)
						if cond.Period == "U" {
							otp.WriteString(fmt.Sprintf("%.5f %.5f\n", cond.N, al.(*RCColumn).Mu(cond)))
						} else {
							otp.WriteString(fmt.Sprintf("%.5f %.5f\n", cond.N, al.Ma(cond)))
						}
					}
					otp.WriteString(fmt.Sprintf("Qu %.5f\n", al.Qa(cond)))
				case *RCGirder:
					nmax := al.(*RCGirder).Nmax(cond)
					nmin := al.(*RCGirder).Nmin(cond)
					nmid := 0.0
					mmax := 0.0
					for i := 0; i <= ndiv; i++ {
						cond.N = nmax - float64(i)*(nmax-nmin)/float64(ndiv)
						if cond.Period == "U" {
							m := al.(*RCGirder).Mu(cond)
							if m > mmax {
								mmax = m
								nmid = cond.N
							}
							otp.WriteString(fmt.Sprintf("%.5f %.5f\n", cond.N, m))
						} else {
							otp.WriteString(fmt.Sprintf("%.5f %.5f\n", cond.N, al.Ma(cond)))
						}
					}
					qu := al.Qa(cond)
					if i == 0 {
						mt := al.Mza(cond)
						yield[0] = 2*nmid - nmin
						yield[1] = nmin
						yield[4] = qu
						yield[5] = -qu
						yield[6] = mt
						yield[7] = -mt
						yield[8] = mmax
						yield[9] = -mmax
					} else {
						yield[2] = qu
						yield[3] = -qu
						yield[10] = mmax
						yield[11] = -mmax
					}
				}
			}
			otp.WriteString(fmt.Sprintf("         NZMAX %9.3f NZMIN %9.3f\n", yield[0], yield[1]))
			otp.WriteString(fmt.Sprintf("         QXMAX %9.3f QXMIN %9.3f\n", yield[2], yield[3]))
			otp.WriteString(fmt.Sprintf("         QYMAX %9.3f QYMIN %9.3f\n", yield[4], yield[5]))
			otp.WriteString(fmt.Sprintf("         MZMAX %9.3f MZMIN %9.3f\n", yield[6], yield[7]))
			otp.WriteString(fmt.Sprintf("         MXMAX %9.3f MXMIN %9.3f\n", yield[8], yield[9]))
			otp.WriteString(fmt.Sprintf("         MYMAX %9.3f MYMIN %9.3f\n", yield[10], yield[11]))
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
	case "setboundary":
		if usage {
			return Usage(":setboundary nfloor")
		}
		if narg < 2 {
			return NotEnoughArgs(":setboundary")
		}
		val, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return err
		}
		eps := 0.001
		if e, ok := argdict["EPS"]; ok {
			if e != "" {
				tmp, err := strconv.ParseFloat(e, 64)
				if err == nil {
					eps = tmp
				}
			}
		}
		frame.SetBoundary(int(val), eps)
	case "facts":
		if usage {
			return Usage(":facts {-skipany=code} {-skipall=code} {-skiproof} {-period=X,Y} {-abs=false}")
		}
		var m bytes.Buffer
		fn = Ce(frame.Path, ".fes")
		abs := false
		if sabs, ok := argdict["ABS"]; ok {
			switch sabs {
			case "true", "True", "TRUE":
				abs = true
			case "false", "False", "FALSE":
				abs = false
			default:
				abs = false
			}
		}
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
		skiproof := false
		if _, ok := argdict["SKIPROOF"]; ok {
			skiproof = true
		}
		period := []string{"X", "Y"}
		if per, ok := argdict["PERIOD"]; ok {
			lis := strings.Split(per, ",")
			if len(lis) >= 2 {
				period[0] = strings.TrimSpace(lis[0])
				period[1] = strings.TrimSpace(lis[1])
			}
		}
		etypes := []int{COLUMN, GIRDER, BRACE, WBRACE, SBRACE}
		if skiproof {
			etypes = []int{COLUMN, BRACE, WBRACE}
		}
		err := frame.Facts(fn, etypes, skipany, skipall, period, abs)
		if err != nil {
			return err
		}
		m.WriteString(fmt.Sprintf("Output: %s", fn))
		return Message(m.String())
	case "factsbysect": // TODO: define nsect and esect as [][]int
		if usage {
			return Usage(":factsbysect {-node=code} {-sect=code} {-period=X,Y}")
		}
		var m bytes.Buffer
		fn = Ce(frame.Path, ".fes")
		var nsect, esect []int
		if ns, ok := argdict["NODE"]; ok {
			nsect = SplitNums(ns)
			m.WriteString(fmt.Sprintf("NODE: %v", nsect))
		}
		if ne, ok := argdict["ELEM"]; ok {
			esect = SplitNums(ne)
			m.WriteString(fmt.Sprintf("ELEM: %v", esect))
		}
		if nsect == nil || esect == nil {
			return NotEnoughArgs(":factsbysect")
		}
		period := []string{"X", "Y"}
		if per, ok := argdict["PERIOD"]; ok {
			lis := strings.Split(per, ",")
			if len(lis) >= 2 {
				period[0] = strings.TrimSpace(lis[0])
				period[1] = strings.TrimSpace(lis[1])
			}
		}
		err := frame.FactsBySect(fn, []int{COLUMN, GIRDER, BRACE, WBRACE, SBRACE}, [][]int{nsect}, [][]int{esect}, period)
		if err != nil {
			return err
		}
		m.WriteString(fmt.Sprintf("Output: %s", fn))
		return Message(m.String())
	case "unchain":
		if usage {
			return Usage(":unchain")
		}
		for _, v := range frame.Chains {
			v.Break()
		}
	case "593_2_3":
		if usage {
			return Usage(":593_2_3")
		}
		cond := NewCondition()
		var otpfn string
		if fn == "" {
			otpfn = frame.Path
		} else {
			otpfn = fn
		}
		reload := true
		if _, ok := argdict["NORELOAD"]; ok {
			reload = false
		}
		sects := make([][]int, 2)
		sects[0] = make([]int, 0)
		sects[1] = make([]int, 0)
		if sx, ok := argdict["SKIPX"]; ok {
			sects[0] = SplitNums(sx)
		}
		if sy, ok := argdict["SKIPY"]; ok {
			sects[1] = SplitNums(sy)
		}
		if reload {
			ReadFile(stw, Ce(otpfn, ".lst"))
		}
		els := stw.SelectedElems()
		sort.Sort(ElemByNum{els})
		a := 0.25
		var otp bytes.Buffer
		if len(frame.Ai.Ci[0]) == 0 {
			_, _, err := frame.AiDistribution()
			if err != nil {
				return err
			}
		}
		for _, el := range els {
			if el == nil {
				continue
			}
			el.Condition = cond.Snapshot()
			el.Condition.Nfact = 1.0
			Nl := el.ReturnStress("L", 0, 0)
			alpha := make([]float64, 2)
		loop593:
			for d, dstr := range []string{"X", "Y"} {
				for _, sx := range sects[d] {
					if el.Sect.Num == sx {
						alpha[d] = 1.0
						continue loop593
					}
				}
				Q := el.ReturnStress(dstr, 0, 2-d)
				height := el.Enod[0].Coord[2]
				Ci := 0.0
				if height < frame.Ai.Boundary[0] {
					Ci = frame.Ai.Ci[d][0]
				} else if height >= frame.Ai.Boundary[frame.Ai.Nfloor-1] {
					Ci = frame.Ai.Ci[d][frame.Ai.Nfloor-1]
				}
				for i := 0; i < frame.Ai.Nfloor-1; i++ {
					// fmt.Println(height, frame.Ai.Boundary[i+1])
					if height < frame.Ai.Boundary[i+1] {
						Ci = frame.Ai.Ci[d][i+1]
						break
					}
				}
				Q2 := a * Ci * Nl
				al := Q2/Q
				fmt.Println(Q, Q2, Ci, Nl, al)
				if math.Abs(al) < 1.0 {
					al = 1.0
				}
				alpha[d] = al
				otp.WriteString(fmt.Sprintf("%s方向: a=%.3f Ci=%.3f N=%.3f QE=%.3f Q=a×Ci×Nl=%.3f, α=max{Q/QE,1.0}=%.3f\n", dstr, a, Ci, Nl, Q, Q2, al))
			}
			_, t, err := el.OutputRateInformation("L", "X", "X", "Y", "Y", -1.0, alpha, 0.0)
			if err != nil {
				return err
			}
			otp.WriteString(t)
			otp.WriteString(strings.Repeat("-", 202))
			otp.WriteString("\n")
		}
		w, err := os.Create(Ce(otpfn, ".593"))
		defer w.Close()
		if err != nil {
			return err
		}
		otp = AddCR(otp)
		otp.WriteTo(w)
	case "1459":
		if usage {
			return Usage(":1459 [-sect= ]")
		}
		cond := func(el *Elem) bool {
			return el.IsLineElem()
		}
		if s, ok := argdict["SECT"]; ok {
			sects := SplitNums(s)
			cond = func(el *Elem) bool {
				for _, snum := range sects {
					if el.Sect.Num == snum {
						return true
					}
				}
				return false
			}
		}
		maxval := 0.0
		maxkey := -1
		maxnode := -1
		maxspan := 0.0
		maxdisp := make([]float64, 3)
		stw.Deselect()
		for k, v := range frame.Chains {
			elems := v.Elems()
			if len(elems) < 2 || !cond(elems[0]) {
				continue
			}
			tmpmax := 0.0
			tmpnode := -1
			start := elems[0].Enod[0]
			end := elems[len(elems)-1].Enod[1]
			average := -0.5 * (start.ReturnDisp(frame.Show.Period, 2) + end.ReturnDisp(frame.Show.Period, 2))
			span := Distance(start, end)
			for _, el := range elems {
				tmpval := (-1.0*el.Enod[1].ReturnDisp(frame.Show.Period, 2) - average) / span
				if tmpval > tmpmax {
					tmpmax = tmpval
					tmpnode = el.Enod[1].Num
				}
			}
			if tmpmax > maxval {
				maxval = tmpmax
				maxkey = k
				maxnode = tmpnode
				maxspan = span * 100.0
				maxdisp[0] = start.ReturnDisp(frame.Show.Period, 2) * (-100.0)
				maxdisp[1] = frame.Nodes[maxnode].ReturnDisp(frame.Show.Period, 2) * (-100.0)
				maxdisp[2] = end.ReturnDisp(frame.Show.Period, 2) * (-100.0)
			}
		}
		if maxkey > 0 {
			delta := maxdisp[1] - 0.5*(maxdisp[0]+maxdisp[2])
			stw.History(fmt.Sprintf("Disp: %.3f - (%.3f + %.3f)/2 = %.3f [cm]", maxdisp[1], maxdisp[0], maxdisp[2], delta))
			stw.History(fmt.Sprintf("Distance: %.3f[cm]", maxspan))
			if delta != 0.0 {
				stw.History(fmt.Sprintf("Slope: 1/%.1f", math.Abs(maxspan/delta)))
			}
			stw.SelectElem(frame.Chains[maxkey].Elems())
			stw.SelectNode([]*Node{frame.Nodes[maxnode]})
		}
	case "slope":
		if usage {
			return Usage(":slope")
		}
		els := currentelem(stw, exmodech, exmodeend)
		if len(els) < 1 {
			return errors.New(":slope: not enough selected elem")
		}
		if els[0].IsLineElem() {
			return errors.New(":slope: not plateelem")
		}
		normal := els[0].Normal(true)
		deg := math.Acos(normal[2])*180.0/math.Pi
		stw.History(fmt.Sprintf("SLOPE: %.3f[deg]", deg))
	case "elemdistance":
		if usage {
			return Usage(":elemdistance")
		}
		els := currentelem(stw, exmodech, exmodeend)
		if len(els) < 2 {
			return errors.New(":elemdistance: not enough selected elem")
		}
		if !els[0].IsLineElem() || !els[1].IsLineElem() {
			return errors.New(":elemdistance: not lineelem")
		}
		pos1, pos2, d, err := DistLineLine(els[0].Enod[0].Coord, els[0].Direction(false), els[1].Enod[1].Coord, els[1].Direction(false))
		if err != nil {
			return err
		}
		fmt.Println(pos1, pos2, d)
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
		fn := filepath.Join(filepath.Dir(frame.Path), "amountl.txt")
		err := frame.AmountLst(fn, sects...)
		if err != nil {
			return err
		}
	case "coord":
		if usage {
			return Usage(":coord x y z")
		}
		if narg < 4 {
			return NotEnoughArgs(":node")
		}
		coord := make([]float64, 3)
		for i := 0; i < 3; i++ {
			tmp, err := strconv.ParseFloat(args[i+1], 64)
			if err != nil {
				return err
			}
			coord[i] = tmp
		}
		n, _ := frame.CoordNode(coord[0], coord[1], coord[2], EPS)
		if cm, ok := stw.(Commander); ok {
			cm.SendNode(n)
		}
		stw.SelectNode([]*Node{n})
		if pipe {
			sender = make([]interface{}, 1)
			sender[0] = n
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
				if cm, ok := stw.(Commander); ok {
					for _, n := range ns[:nods] {
						cm.SendNode(n)
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
			case abbrev.For("/P/LOAD/ED", condition):
				f = func(n *Node) bool {
					for i := 0; i < 6; i++ {
						if n.Load[i] != 0.0 {
							return true
						}
					}
					return false
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
				if cm, ok := stw.(Commander); ok {
					for _, n := range ns[:num] {
						cm.SendNode(n)
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
			if cm, ok := stw.(Commander); ok {
				for _, n := range ns[:num] {
					cm.SendNode(n)
				}
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
	case "nodeset":
		if usage {
			return Usage(":nodeset [create,select]")
		}
		if narg < 2 {
			return NotEnoughArgs(":nodeset")
		}
		switch strings.ToUpper(args[1]) {
		case "CREATE":
			if narg < 3 {
				return NotEnoughArgs(":nodeset create")
			}
			if !stw.NodeSelected() {
				return errors.New(":nodeset create no selected node")
			}
			frame.AddNodeSet(args[2], stw.SelectedNodes())
		case "SELECT":
			if narg < 3 {
				return NotEnoughArgs(":nodeset select")
			}
			stw.DeselectNode()
			if ns, ok := frame.NodeSet[args[2]]; ok {
				stw.SelectNode(ns)
			} else {
				return fmt.Errorf("nodeset {} is not defined", args[2])
			}
		default:
			return fmt.Errorf(":nodeset {} is not defined", args[1])
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
			SetConf(stw, lis)
		} else {
			return NotEnoughArgs(":conf")
		}
	case "phase":
		if usage {
			return Usage(":phase {-degree} xfactor yfactor")
		}
		if !stw.NodeSelected() {
			return errors.New(":phase no selected node")
		}
		if narg < 3 {
			return NotEnoughArgs(":phase")
		}
		degree := false
		if _, ok := argdict["DEGREE"]; ok {
			degree = true
		}
		xfactor, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return err
		}
		if degree {
			xfactor = math.Cos(math.Pi/180.0*xfactor)
		}
		yfactor, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			return err
		}
		if degree {
			yfactor = math.Cos(math.Pi/180.0*yfactor)
		}
		for _, n := range stw.SelectedNodes() {
			if n == nil {
				continue
			}
			n.Phase[0] = xfactor
			n.Phase[1] = yfactor
		}
		Snapshot(stw)
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
			Snapshot(stw)
		} else {
			return errors.New(fmt.Sprintf(":pile PILE %d doesn't exist", val))
		}
	case "scale":
		if usage {
			return Usage(":scale factor cx cy cz")
		}
		if !stw.NodeSelected() {
			return errors.New(":scale no selected node")
		}
		if narg < 5 {
			return NotEnoughArgs(":scale")
		}
		factor, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return err
		}
		coords := make([]float64, 3)
		for i := 0; i < 3; i++ {
			coord, err := strconv.ParseFloat(args[2+i], 64)
			if err != nil {
				return err
			}
			coords[i] = coord
		}
		for _, n := range stw.SelectedNodes() {
			if n == nil {
				continue
			}
			n.Scale(coords, factor, factor, factor)
		}
		Snapshot(stw)
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
		Snapshot(stw)
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
		Snapshot(stw)
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
		Snapshot(stw)
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
		Snapshot(stw)
	case "cmq":
		if usage {
			return Usage(":cmq [period, zero, value]")
		}
		if narg < 2 {
			return NotEnoughArgs(":cmq")
		}
		els := currentelem(stw, exmodech, exmodeend)
		switch strings.ToUpper(args[1]) {
		case "ZERO":
			for _, el := range els {
				for i := 0; i < 12; i++ {
					el.Cmq[i] = 0.0
				}
			}
		case "VALUE":
			if narg < 14 {
				return NotEnoughArgs(":cmq value")
			}
			cmq := make([]float64, 12)
			for i := 0; i < 12; i++ {
				tmp, err := strconv.ParseFloat(args[2+i], 64)
				if err != nil {
					return err
				}
				cmq[i] = tmp
			}
			for _, el := range els {
				for i := 0; i < 12; i++ {
					el.Cmq[i] = cmq[i]
				}
			}
		default:
			if s, ok := frame.ResultFileName[args[1]]; ok {
				if s == "" {
					return errors.New(fmt.Sprintf("period %s: no data", args[1]))
				}
				for _, el := range els {
					for i := 0; i < 2; i++ {
						for j := 0; j < 6; j++ {
							el.Cmq[6*i+j] = el.Stress[args[1]][el.Enod[i].Num][j]
						}
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
			return Usage(":elem [elemcode,sect sectcode,etype,curtain,isgohan,error,reaction,locked,isolated,skip]")
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
				if cm, ok := stw.(Commander); ok {
					for _, el := range elems[:els] {
						cm.SendElem(el)
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
			case strings.EqualFold(condition, "isolated"):
				_, els := frame.Isolated()
				f = func(el *Elem) bool {
					for _, iel := range els {
						if el == iel {
							return true
						}
					}
					return false
				}
			case strings.EqualFold(condition, "skip"):
				f = func(el *Elem) bool {
					return el.IsSkipAny()
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
				if cm, ok := stw.(Commander); ok {
					for _, el := range els[:num] {
						cm.SendElem(el)
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
			if cm, ok := stw.(Commander); ok {
				for _, el := range els[:num] {
					cm.SendElem(el)
				}
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
			return Usage(":bond [pin,rigid,[01_t]{6}] [upper,lower,sect sectcode, nonrigid]")
		}
		if narg < 2 {
			return NotEnoughArgs(":bond")
		}
		els := currentelem(stw, exmodech, exmodeend)
		lis := make([]*Bond, 6)
		pat := regexp.MustCompile("[0-9_t]{6}")
		switch {
		case strings.EqualFold(args[1], "pin"):
			lis[4] = Pin
			lis[5] = Pin
		case pat.MatchString(args[1]):
			for i := 0; i < 6; i++ {
				switch args[1][i] {
				default:
					tmp, err := strconv.ParseInt(string(args[1][i]), 10, 64)
					if err == nil {
						if b, ok := frame.Bonds[int(tmp)]; ok {
							lis[i] = b
						}
					}
				case '0':
					lis[i] = nil
				case '1':
					lis[i] = Pin
				case '_':
					continue
				case 't':
					if lis[i] == nil {
						lis[i] = Pin
					} else {
						lis[i] = nil
					}
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
			case abbrev.For("nonr/igid", condition):
				f = func(el *Elem, ind int) bool {
					return !el.IsRigid(ind)
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
		Snapshot(stw)
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
		Snapshot(stw)
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
		Snapshot(stw)
	case "setcang":
		if narg < 2 {
			if usage {
				return Usage(":setcang [plate -enum=]")
			}
			return NotEnoughArgs(":setcang")
		}
		switch strings.ToLower(args[1]) {
		case "plate":
			var normal []float64
			if en, ok := argdict["ENUM"]; ok {
				enum, err := strconv.ParseInt(en, 10, 64)
				if err == nil {
					if el, ok := frame.Elems[int(enum)]; ok {
						if !el.IsLineElem() {
							normal = el.Normal(true)
						}
					}
				}
			}
			els := currentelem(stw, exmodech, exmodeend)
			var err error
			for _, el := range els {
				if el.IsNotEditable(frame.Show) || !el.IsLineElem() {
					continue
				}
				plates := frame.SearchElem(el.Enod...)
				if normal == nil {
					vec := make([]float64, 3)
					for _, p := range plates {
						if p.Etype == WALL || p.Etype == SLAB {
							tmp := p.Normal(true)
							for i := 0; i < 3; i++ {
								vec[i] += tmp[i]
							}
						}
					}
					_, err = el.AxisToCang(vec, false)
				} else {
					_, err = el.AxisToCang(normal, false)
				}
				if err != nil {
					return err
				}
			}
			Snapshot(stw)
		}
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
			if el.IsNotEditable(frame.Show) || !el.IsLineElem() {
				continue
			}
			_, err := el.AxisToCang(vec, strong)
			if err != nil {
				return err
			}
		}
		Snapshot(stw)
	case "invert":
		if usage {
			return Usage(":invert")
		}
		els := currentelem(stw, exmodech, exmodeend)
		for _, el := range els {
			el.Invert()
		}
		Snapshot(stw)
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
		Snapshot(stw)
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
				val := el.Sect.Figs[0].Prop.EL() * a * alpha * tmp
				el.Cmq[0] += val
				el.Cmq[6] -= val
			}
		}
		Snapshot(stw)
		return Message(m.String())
	case "join":
		if usage {
			return Usage(":join")
		}
		els := currentelem(stw, exmodech, exmodeend)
		if len(els) < 2 {
			return errors.New("not enough elems selected")
		}
		err := frame.JoinLineElem(els[0], els[1], !bang, !bang)
		if err != nil {
			return err
		}
		stw.Deselect()
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
			side := 1
			if sd, ok := argdict["SIDE"]; ok {
				switch sd {
				case "0":
					side = 1
				case "1":
					side = 2
				case "both", "BOTH":
					side = 3
				case "pin", "PIN":
					side = 4
				}
			}
			divfunc = func(el *Elem) ([]*Node, []*Elem, error) {
				switch side {
				case 1:
					return el.DivideAtLength(val, EPS)
				case 2:
					return el.DivideAtLength(el.Length()-val, EPS)
				case 3:
					ns, els, err := el.DivideAtLength(val, EPS)
					if err != nil {
						return ns, els, err
					}
					ns2, els2, err := els[1].DivideAtLength(els[1].Length()-val, EPS)
					if err != nil {
						return ns, els, err
					}
					ns = append(ns, ns2...)
					els = append(els, els2[1])
					return ns, els, nil
				case 4:
					ns := make([]*Node, 0)
					els := []*Elem{el}
					divelem := el
					if divelem.IsPin(0) {
						ns1, els1, err := divelem.DivideAtLength(val, EPS)
						if err != nil {
							return ns, els, err
						}
						ns = append(ns, ns1...)
						els = append(els, els1[1])
						divelem = els1[1]
					}
					if divelem.IsPin(1) {
						ns1, els1, err := divelem.DivideAtLength(divelem.Length()-val, EPS)
						if err != nil {
							return ns, els, err
						}
						ns = append(ns, ns1...)
						els = append(els, els1[1])
						divelem = els1[1]
					}
					return ns, els, nil
				default:
					return el.DivideAtLength(val, EPS)
				}
			}
		case "range":
			if usage {
				return Usage("divide range l")
			}
			if narg < 3 {
				return NotEnoughArgs(":divide length")
			}
			val, err := strconv.ParseFloat(args[2], 64)
			elems := stw.SelectedElems()
			en, err := CommonEnod(elems...)
			if err != nil {
				return err
			}
			if len(en) == 0 || en[0] == nil {
				return fmt.Errorf("no common enod")
			}
			common := en[0]
			divfunc = func(el *Elem) ([]*Node, []*Elem, error) {
				var length float64
				ind, _ := el.EnodIndex(common.Num)
				if ind == 0 {
					length = val
				} else {
					length = el.Length() - val
				}
				return el.DivideAtLength(length, EPS)
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
				ErrorMessage(stw, err, ERROR)
				continue
			}
			if err == nil && len(els) > 1 {
				tmpels = append(tmpels, els...)
				enum += len(els)
			}
		}
		stw.SelectElem(tmpels[:enum])
		Snapshot(stw)
	case "dividearc":
		if narg < 2 {
			if usage {
				return Usage(":dividearc [n]")
			}
			return NotEnoughArgs(":dividearc")
		}
		var divfunc func(*Arc) ([]*Node, []*Elem, error)
		switch strings.ToLower(args[1]) {
		case "n":
			if usage {
				return Usage(":dividearc n ndiv")
			}
			if narg < 3 {
				return NotEnoughArgs(":dividearc n")
			}
			val, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return err
			}
			ndiv := int(val)
			divfunc = func(arc *Arc) ([]*Node, []*Elem, error) {
				return arc.DivideInN(ndiv, EPS)
			}
		}
		if divfunc == nil {
			return errors.New(":dividearc: unknown format")
		}
		tmpels := make([]*Elem, 0)
		enum := 0
		for _, arc := range frame.Arcs {
			_, els, err := divfunc(arc)
			if err != nil {
				ErrorMessage(stw, err, ERROR)
				continue
			}
			if err == nil && len(els) > 1 {
				tmpels = append(tmpels, els...)
				enum += len(els)
			}
		}
		stw.SelectElem(tmpels[:enum])
		Snapshot(stw)
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
				if sec.Num > 900 || sec.Type < 0 {
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
		case "edge":
			if usage {
				return Usage(":add edge {-sect=code} {-etype=type}")
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
			els := currentelem(stw, exmodech, exmodeend)
			if len(els) == 0 {
				return fmt.Errorf("no elems selected")
			}
			rtn := make([]*Elem, 0)
			for _, el := range els {
				if el.IsLineElem() {
					continue
				}
				if el.Enods < 3 {
					continue
				}
				for i := 0; i < el.Enods; i++ {
					addedge := true
					ind1 := i
					ind2 := i+1
					if ind2 >= el.Enods {
						ind2 = 0
					}
					cels := frame.SearchElem(el.Enod[ind1], el.Enod[ind2])
					for _, cel := range cels {
						if cel.IsLineElem() {
							addedge = false
							break
						}
					}
					if addedge {
						newel := frame.AddLineElem(-1, []*Node{el.Enod[ind1], el.Enod[ind2]}, sect, etype)
						rtn = append(rtn, newel)
					}
				}
			}
			stw.Deselect()
			stw.SelectElem(rtn)
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
	case "move":
		if usage {
			return Usage(":move x y z")
		}
		if narg < 4 {
			return NotEnoughArgs(":move")
		}
		vec := make([]float64, 3)
		for i := 0; i < 3; i++ {
			val, err := strconv.ParseFloat(args[1+i], 64)
			if err != nil {
				return err
			}
			vec[i] = val
		}
		ns := make([]*Node, 0)
		if stw.NodeSelected() {
			for _, n := range stw.SelectedNodes() {
				if n != nil {
					ns = append(ns, n)
				}
			}
		}
		els := currentelem(stw, exmodech, exmodeend)
		if len(els) > 0 {
			en := frame.ElemToNode(els...)
			var add bool
			for _, n := range en {
				add = true
				for _, m := range ns {
					if m == n {
						add = false
						break
					}
				}
				if add {
					ns = append(ns, n)
				}
			}
		}
		if len(ns) == 0 {
			return nil
		}
		for _, n := range ns {
			if n.IsNotEditable(frame.Show) {
				continue
			}
			n.Move(vec[0], vec[1], vec[2])
		}
	case "copy":
		if narg < 2 {
			if usage {
				return Usage(":copy [amount,elem,sect,stress]")
			}
			return NotEnoughArgs(":copy")
		}
		switch strings.ToLower(args[1]) {
		case "elem":
			if usage {
				return Usage(":copy elem x y z")
			}
			if narg < 5 {
				return NotEnoughArgs(":copy elem")
			}
			vec := make([]float64, 3)
			for i := 0; i < 3; i++ {
				val, err := strconv.ParseFloat(args[2+i], 64)
				if err != nil {
					return err
				}
				vec[i] = val
			}
			for _, el := range currentelem(stw, exmodech, exmodeend) {
				if el.IsNotEditable(frame.Show) {
					continue
				}
				el.Copy(vec[0], vec[1], vec[2], EPS)
			}
			Snapshot(stw)
		case "stress":
			if usage {
				return Usage(":copy stress {-format=%.3f} [01]{0,12}")
			}
			inds := make([]bool, 12)
			if narg >= 3 {
				for i, s := range args[2] {
					if s == '1' {
						inds[i] = true
					}
				}
			}
			format := "%.3f"
			if f, ok := argdict["FORMAT"]; ok {
				format = f
			}
			period := frame.Show.Period
			var w bytes.Buffer
			for _, el := range currentelem(stw, exmodech, exmodeend) {
				w.WriteString(fmt.Sprintf("%d ", el.Num))
				for i := 0; i < 2; i++ {
					for j := 0; j < 6; j++ {
						if inds[6*i+j] {
							unit := frame.Show.Unit[0]
							if j >= 3 {
								unit *= frame.Show.Unit[1]
							}
							w.WriteString(fmt.Sprintf(" "+format, el.ReturnStress(period, i, j)*unit))
						}
					}
				}
				w.WriteString("\n")
			}
			err := clipboard.WriteAll(w.String())
			if err != nil {
				return err
			}
		case "amount":
			if usage {
				return Usage(":copy amount {-format=%.3f}")
			}
			format := "%.3f"
			if f, ok := argdict["FORMAT"]; ok {
				format = f
			}
			var w bytes.Buffer
			for _, el := range currentelem(stw, exmodech, exmodeend) {
				w.WriteString(fmt.Sprintf("%d ", el.Num))
				w.WriteString(fmt.Sprintf(format, el.Amount()))
				w.WriteString("\n")
			}
			err := clipboard.WriteAll(w.String())
			if err != nil {
				return err
			}
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
		case "disp":
			if usage {
				return Usage(":copy disp {-format=%.3f}")
			}
			format := "%.3f"
			if f, ok := argdict["FORMAT"]; ok {
				format = f
			}
			var w bytes.Buffer
			for _, n := range currentnode(stw, exmodech, exmodeend) {
				w.WriteString(fmt.Sprintf("%d", n.Num))
				for i := 0; i < 6; i++ {
					w.WriteString(fmt.Sprintf(" "+format, n.ReturnDisp(frame.Show.Period, i)))
				}
				w.WriteString("\n")
			}
			err := clipboard.WriteAll(w.String())
			if err != nil {
				return err
			}
		}
	case "arraycopy":
		if narg < 2 {
			if usage {
				return Usage(":arraycopy [rect,polar]")
			}
			return NotEnoughArgs(":arraycopy")
		}
		switch strings.ToLower(args[1]) {
		case "rect":
			if usage {
				return Usage(":arraycopy rect dx dy dz nx ny nz")
			}
			if narg < 8 {
				return NotEnoughArgs(":arraycopy rect")
			}
			d := make([]float64, 3)
			n := make([]int, 3)
			for i := 0; i < 3; i++ {
				dval, err := strconv.ParseFloat(args[2+i], 64)
				if err != nil {
					return err
				}
				d[i] = dval
				nval, err := strconv.ParseInt(args[5+i], 10, 64)
				if err != nil {
					return err
				}
				n[i] = int(nval)
			}
			els := currentelem(stw, exmodech, exmodeend)
			if len(els) == 0 {
				return fmt.Errorf("no nodes or elems selected")
			}
			vec := make([]float64, 3)
			first := true
			for i := 0; i < n[0]; i++ {
				for j := 0; j < n[1]; j++ {
					for k := 0; k < n[2]; k++ {
						if !first {
							for _, el := range els {
								if el.IsNotEditable(frame.Show) {
									continue
								}
								el.Copy(vec[0], vec[1], vec[2], EPS)
							}
						} else {
							first = false
						}
						vec[2] += d[2]
					}
					vec[1] += d[1]
					vec[2] = 0.0
				}
				vec[0] += d[0]
				vec[1] = 0.0
			}
			Snapshot(stw)
		case "polar":
			if usage {
				return Usage(":arraycopy polar nnum axis dtheta(deg) n")
			}
			if narg < 6 {
				return NotEnoughArgs(":arraycopy polar")
			}
			nnum, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return err
			}
			var center, vector []float64
			if n, ok := frame.Nodes[int(nnum)]; ok {
				center = n.Coord
			} else {
				return fmt.Errorf("NODE %d doesn't exist", nnum)
			}
			switch args[3] {
			case "0", "x", "X":
				vector = []float64{1, 0, 0}
			case "1", "y", "Y":
				vector = []float64{0, 1, 0}
			case "2", "z", "Z":
				vector = []float64{0, 0, 1}
			default:
				return fmt.Errorf("unknown axis")
			}
			dtheta, err := strconv.ParseFloat(args[4], 64)
			if err != nil {
				return err
			}
			dtheta *= math.Pi / 180.0
			n, err := strconv.ParseInt(args[5], 10, 64)
			if err != nil {
				return err
			}
			els := currentelem(stw, exmodech, exmodeend)
			if len(els) == 0 {
				return fmt.Errorf("no nodes or elems selected")
			}
			theta := dtheta
			for i := 0; i < int(n); i++ {
				for _, el := range els {
					if el.IsNotEditable(frame.Show) {
						continue
					}
					el.CopyRotate(center, vector, theta, EPS)
				}
				theta += dtheta
			}
		}
	case "set":
		if usage {
			return Usage(":set {-sect=} {-etype=}")
		}
		var fs []func(*Elem)
		if val, ok := argdict["SECT"]; ok {
			snum, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return err
			}
			if sec, ok := frame.Sects[int(snum)]; ok {
				fs = append(fs, func(el *Elem) {
					el.Sect = sec
				})
			} else {
				return fmt.Errorf("sect %d doesn't exist", snum)
			}
		}
		if et, ok := argdict["ETYPE"]; ok {
			var etype int
			switch {
			case Re_column.MatchString(et):
				etype = COLUMN
			case Re_girder.MatchString(et):
				etype = GIRDER
			case Re_brace.MatchString(et):
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
			if etype != NULL {
				fs = append(fs, func(el *Elem) {
					el.Etype = etype
				})
			}
		}
		if len(fs) > 0 {
			for _, el := range currentelem(stw, exmodech, exmodeend) {
				for _, f := range fs {
					f(el)
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
		for _, el := range currentelem(stw, exmodech, exmodeend) {
			frame.DeleteElem(el.Num)
		}
		for _, n := range currentnode(stw, exmodech, exmodeend) {
			frame.DeleteNode(n.Num)
		}
		ns := frame.NodeNoReference()
		if len(ns) != 0 {
			for _, n := range ns {
				frame.DeleteNode(n.Num)
			}
		}
		stw.Deselect()
		Snapshot(stw)
	case "count":
		if usage {
			return Usage(":count")
		}
		if stw.NodeSelected() || stw.ElemSelected() {
			ns := stw.SelectedNodes()
			els := stw.SelectedElems()
			return Message(fmt.Sprintf("NODES: %d, ELEMS: %d", len(ns), len(els)))
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
					sum += el.DeformedLength(frame.Show.Period, 1.0)
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
					sum += el.DeformedArea(frame.Show.Period, 1.0)
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
				AxisRange(stw, i, -100.0, 1000.0, false)
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
			AxisRange(stw, axis, -100.0, 1000.0, false)
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
		AxisRange(stw, axis, min, max, false)
	case "height":
		if usage {
			return Usage(":height f1 f2")
		}
		if narg == 1 {
			AxisRange(stw, 2, -100.0, 1000.0, false)
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
		AxisRange(stw, 2, frame.Ai.Boundary[min], frame.Ai.Boundary[max], false)
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
		return ExMode(stw, fmt.Sprintf("height %d %d", n-1, n+1))
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
		return ExMode(stw, fmt.Sprintf("height %d %d", n-1, n))
	case "height+":
		NextFloor(stw)
	case "height-":
		PrevFloor(stw)
	case "setangle":
		if usage {
			return Usage(":setangle phi theta")
		}
		if narg < 3 {
			return NotEnoughArgs(":setangle")
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
		if narg < 2 {
			return NotEnoughArgs(":view")
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
		case "ORTHO":
			if !stw.ElemSelected() {
				return nil
			}
			el := stw.SelectedElems()[0]
			if el.IsLineElem() {
				d := el.Direction(true)
				stw.SetAngle(0.0, math.Acos(d[0])*180.0/math.Pi-90.0)
			} else {
				n := el.Normal(true)
				theta := math.Acos(n[0])*180.0/math.Pi + 180.0
				phi := -math.Acos(n[2])*180.0/math.Pi + 90.0
				stw.SetAngle(phi, theta)
			}
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
				err := ExMode(stw, fmt.Sprintf(":paper %s", strings.Join(args[2:], " ")))
				if err != nil {
					return err
				}
			}
		case "OFF", "FALSE", "NO":
			stw.SetShowPrintRange(false)
		default:
			err := ExMode(stw, fmt.Sprintf(":paper %s", strings.Join(args[1:], " ")))
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
		if usage {
			return Usage(":postscript {-size=a4tate} filename")
		}
		if fn == "" {
			fn = filepath.Join(stw.Cwd(), "test.ps")
		}
		if name, ok := argdict["SIZE"]; ok {
			switch strings.ToUpper(name) {
			case "A4TATE":
				stw.SetPaperSize(A4_TATE)
			case "A4YOKO":
				stw.SetPaperSize(A4_YOKO)
			case "A3TATE":
				stw.SetPaperSize(A3_TATE)
			case "A3YOKO":
				stw.SetPaperSize(A3_YOKO)
			}
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
		err := frame.PrintPostScript(fn, paper)
		if err != nil {
			return err
		}
	case "object":
		if !stw.NodeSelected() || !stw.ElemSelected() {
			return nil
		}
		c := NewChain(frame, stw.SelectedNodes()[0], stw.SelectedElems()[0], Straight(1e-2), func(c *Chain) bool { return c.Elem().IsPin(c.Node().Num) }, nil, nil)
		for c.Next() {
			AddSelection(stw, c.Elem())
		}
	case "pdiv":
		if !stw.ElemSelected() {
			return nil
		}
		els, _, err := stw.SelectedElems()[0].PlateDivision(true)
		if err != nil {
			return err
		}
		for _, el := range els {
			frame.AddElem(-1, el)
		}
	case "suspicious":
		nodes, elems, err := frame.Suspicious()
		if err != nil {
			stw.SelectNode(nodes)
			stw.SelectElem(elems)
			ErrorMessage(stw, err, ERROR)
		}
	case "elimallpin":
		nodes := frame.EliminateAllPin()
		stw.SelectNode(nodes)
	case "keisansho":
		frame.OutputTex()
		// t, err := frame.Elems[1003].OutputRateInformation("L", "X", "X", "Y", "Y")
		// if err != nil {
		// 	return err
		// }
		// fmt.Println(t)
	case "extractarclm":
		if usage {
			return Usage(":extractarclm")
		}
		if fn == "" {
			fn = "hogtxt.wgt"
		}
		return frame.ExtractArclm(fn)
	case "saveasarclm":
		if usage {
			return Usage(":saveasarclm")
		}
		frame.SaveAsArclm("")
	case "all":
		frame.ExtractArclm(Ce(frame.Path, ".wgt"))
		frame.SaveAsArclm("")
		acond := arclm.NewAnalysisCondition()
		extra := make([][]float64, 2)
		for i, eper := range []string{"X", "Y"} {
			eaf := frame.Arclms[eper]
			_, _, vec, err := eaf.AssemGlobalVector(1.0)
			if err != nil {
				return err
			}
			extra[i] = vec
		}
		acond.SetExtra(extra)
		var otp string
		if fn == "" {
			otp = Ce(frame.Path, ".otp")
		} else {
			otp = fn
		}
		pers := []string{"L", "X", "Y"}
		af := frame.Arclms["L"]
		if af == nil {
			return fmt.Errorf(":all: frame isn't extracted to period L")
		}
		otps := []string{Ce(otp, "otl"), Ce(otp, "ohx"), Ce(otp, "ohy")}
		acond.SetOutput(otps)
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			err := af.StaticAnalysis(cancel, acond)
			af.Endch <- err
		}()
		ind := 0
		retval := 0
	ex_all:
		for {
			select {
			case <-ctx.Done():
				retval = 1
			case <-af.Pivot:
			case <-af.Lapch:
				frame.ReadArclmData(af, pers[ind])
				frame.ResultFileName[pers[ind]] = otps[ind]
				ind++
				af.Lapch <- retval
			case <-af.Endch:
				break ex_all
			}
		}
		ReadFile(stw, Ce(frame.Path, ".lst"))
		cond := NewCondition()
		frame.SectionRateCalculation(otp, "L", "X", "X", "Y", "Y", -1.0, cond)
	case "analysis":
		if usage {
			return Usage(":analysis {-period=name} {-all} {-solver=name} {-eps=value} {-nlgeom} {-nlmat} {-step=nlap;delta;start;max} {-noinit} {-wait} filename")
		}
		cond := arclm.NewAnalysisCondition()
		var otp string
		if fn == "" {
			otp = Ce(frame.Path, ".otp")
		} else {
			otp = fn
		}
		cond.SetOutput([]string{otp})
		if s, ok := argdict["SOLVER"]; ok {
			if s != "" {
				cond.SetSolver(strings.ToUpper(s))
			}
		}
		if e, ok := argdict["EPS"]; ok {
			if e != "" {
				tmp, err := strconv.ParseFloat(e, 64)
				if err == nil {
					cond.SetEps(tmp)
				}
			}
		}
		if _, ok := argdict["NLGEOM"]; ok {
			cond.SetNlgeometry(true)
		}
		if _, ok := argdict["NLMAT"]; ok {
			cond.SetNlmaterial(true)
		}
		if s, ok := argdict["STEP"]; ok { // NLAP;DELTA;START;MAX
			ns := strings.Count(s, ";")
			if ns < 3 {
				s += strings.Repeat(";", 3-ns)
			}
			lis := strings.Split(s, ";")
			val, err := strconv.ParseInt(lis[0], 10, 64)
			if err == nil {
				cond.SetNlap(int(val))
			}
			tmp, err := strconv.ParseFloat(lis[1], 64)
			if err == nil {
				cond.SetDelta(tmp)
			}
			tmp, err = strconv.ParseFloat(lis[2], 64)
			if err == nil {
				cond.SetStart(tmp)
			}
			tmp, err = strconv.ParseFloat(lis[3], 64)
			if err == nil {
				cond.SetMax(tmp)
			}
		}
		if _, ok := argdict["NOINIT"]; ok {
			cond.SetInit(false)
		}
		per := "L"
		if p, ok := argdict["PERIOD"]; ok {
			if p != "" {
				per = strings.ToUpper(p)
			}
		}
		pers := []string{per}
		if _, ok := argdict["ALL"]; ok {
			if cond.NonLinear() {
				return fmt.Errorf("\":analysis-all\" cannot be used for non-linear analysis")
			}
			extra := make([][]float64, 2)
			pers = []string{"L", "X", "Y"}
			otps := []string{Ce(otp, ".otl"), Ce(otp, ".ohx"), Ce(otp, ".ohy")}
			per = "L"
			for i, eper := range []string{"X", "Y"} {
				eaf := frame.Arclms[eper]
				if eaf == nil {
					return fmt.Errorf(":analysis: frame isn't extracted to period %s", eper)
				}
				_, _, vec, err := eaf.AssemGlobalVector(1.0)
				if err != nil {
					return err
				}
				extra[i] = vec
			}
			cond.SetOutput(otps)
			cond.SetExtra(extra)
		}
		af := frame.Arclms[per]
		if af == nil {
			return fmt.Errorf(":analysis: frame isn't extracted to period %s", per)
		}
		af.Output = stw.HistoryWriter()
		if af.Running() {
			return fmt.Errorf("analysis is running")
		}
		var m, m2 bytes.Buffer
		m.WriteString(fmt.Sprintf("PERIOD      : %s\n", per))
		if pp, ok := argdict["POST"]; ok {
			switch strings.ToUpper(pp) {
			case "FLOOR":
				zval := 0.002
				if z, zok := argdict["Z"]; zok {
					tmp, err := strconv.ParseFloat(z, 64)
					if err == nil {
						zval = tmp
					}
				}
				cond.SetPostprocess(func(frame *arclm.Frame, df [][]float64, du []float64, dr []float64) (float64, bool) {
					delta := 1.0
					for i, n := range frame.Nodes {
						current := n.Coord[2] + n.Disp[2]
						if !n.Conf[2] && current <= zval {
							if current < 0.0 {
								tmp := (zval-current)/du[6*i+2] + 1.0
								if tmp > 0.0 && tmp < delta {
									delta = tmp
								}
							}
							n.Conf[2] = true
						} else if n.Conf[2] && n.Reaction[2] < 0.0 {
							n.Conf[2] = false
						}
					}
					if delta < 0.99 {
						return delta * cond.Delta(), false
					} else {
						return cond.Delta(), true
					}
				})
			case "IMCOMP":
				comp := 0.0
				if c, ok := argdict["COMP"]; ok {
					val, err := strconv.ParseFloat(c, 64)
					if err == nil {
						comp = val
					}
				}
				var sects []int
				if s, ok := argdict["SECTS"]; ok {
					sects = SplitNums(s)
				}
				m2.WriteString(fmt.Sprintf("\nINCOMPRESSIBLE: %v", sects))
			incomp:
				for _, el := range af.Elems {
					for _, sec := range sects {
						if el.Sect.Num == sec {
							el.SetIncompressible(comp)
							continue incomp
						}
					}
				}
				cond.SetPostprocess(func(frame *arclm.Frame, df [][]float64, du []float64, dr []float64) (float64, bool) {
					next := true
					del := make([]int, 0)
					res := make([]int, 0)
					for _, el := range frame.Elems {
						checked := el.Check()
						switch checked {
						case arclm.DELETED:
							next = false
							del = append(del, el.Num)
						case arclm.RESTORED:
							res = append(res, el.Num)
						}
					}
					return cond.Delta(), next
				})
			}
		}
		m.WriteString(cond.String())
		m.WriteString(m2.String())
		wait := false
		var wch chan int
		if _, ok := argdict["WAIT"]; ok {
			wait = true
			wch = make(chan int)
		}
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			err := af.StaticAnalysis(cancel, cond)
			af.Endch <- err
		}()
		nlap := cond.Nlap()
		stw.CurrentLap("Calculating...", 0, nlap)
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
		pind := 0
		go func() {
			retval := 0
		readstatic:
			for {
				select {
				case <-ctx.Done():
					retval = 1
				case <-af.Pivot:
					if stw.Pivot() {
						pivot <- 1
					}
				case lap := <-af.Lapch:
					frame.ReadArclmData(af, pers[pind])
					pind++
					if pind == len(pers) {
						pind = 0
					}
					af.Lapch <- retval
					stw.CurrentLap("Calculating...", lap, nlap)
					stw.Redraw()
				case err := <-af.Endch:
					if stw.Pivot() {
						end <- 1
					}
					if err != nil {
						stw.History(err.Error())
					} else {
						stw.CurrentLap("Completed", nlap, nlap)
					}
					otps := cond.Output()
					if len(otps) > pind {
						frame.ResultFileName[pers[pind]] = otps[pind]
					}
					SetPeriod(stw, per)
					stw.Redraw()
					if wait {
						wch <- 1
					}
					break readstatic
				}
			}
		}()
		if wait {
			<-wch
		}
		return ArclmStart(m.String())
	case "stop":
		if usage {
			return Usage(":stop period")
		}
		if narg < 2 {
			return NotEnoughArgs(":stop")
		}
		per := strings.ToUpper(args[1])
		af := frame.Arclms[per]
		if af == nil {
			return fmt.Errorf(":stop: frame isn't extracted to period %s", per)
		}
		if !af.Running() {
			return fmt.Errorf("analysis is not running; %s", per)
		}
		stw.History(fmt.Sprintf("stopping analysis...: %s", per))
		err := af.Stop()
		if err != nil {
			return err
		}
	case "sleep":
		if usage {
			return Usage(":sleep millisecond")
		}
		if narg < 2 {
			return NotEnoughArgs(":sleep")
		}
		val, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return err
		}
		time.Sleep(time.Duration(int(val))*time.Millisecond)
	case "arclm001":
		return Usage("DEPRECATED: use :analysis {-period=name} {-all} {-solver=name} {-eps=value} {-noinit} {-wait} filename")
	case "arclm201":
		return Usage("DEPRECATED: use :analysis -nlgeom {-period=name} {-solver=name} {-eps=value} {-step=nlap;delta;start;max} {-noinit} {-wait} filename")
	case "arclm202":
		return Usage("DEPRECATED: use :analysis -nlgeom -pp=imcomp {-sects=val} {-comp=val} {-period=name} {-solver=name} {-eps=value} {-step=nlap;delta;start;max} {-noinit} {-wait} filename")
	case "arclm203":
		return Usage("DEPRECATED: use :analysis -nlgeom -pp=floor {-z=val} {-period=name} {-solver=name} {-eps=value} {-step=nlap;delta;start;max} {-noinit} {-wait} filename")
	case "arclm101":
		if usage {
			return Usage(":arclm101 {-period=name} {-eps=val} {-step=nlap;delta;start;max} filename")
		}
		cond := arclm.NewAnalysisCondition()
		if e, ok := argdict["EPS"]; ok {
			if e != "" {
				tmp, err := strconv.ParseFloat(e, 64)
				if err == nil {
					cond.SetEps(tmp)
				}
			}
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
		cond.SetOutput([]string{otp})
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
		pers := []string{per}
		if s, ok := argdict["STEP"]; ok { // NLAP;DELTA;START;MAX
			ns := strings.Count(s, ";")
			if ns < 3 {
				s += strings.Repeat(";", 3-ns)
			}
			lis := strings.Split(s, ";")
			val, err := strconv.ParseInt(lis[0], 10, 64)
			if err == nil {
				cond.SetNlap(int(val))
			}
			tmp, err := strconv.ParseFloat(lis[1], 64)
			if err == nil {
				cond.SetDelta(tmp)
			}
			tmp, err = strconv.ParseFloat(lis[2], 64)
			if err == nil {
				cond.SetStart(tmp)
			}
			tmp, err = strconv.ParseFloat(lis[3], 64)
			if err == nil {
				cond.SetMax(tmp)
			}
		}
		af := frame.Arclms[per]
		if _, ok := argdict["NOINIT"]; ok {
			af.RestoreState(frame.Arclms[frame.Show.Period].SaveState())
			cond.SetInit(false)
		}
		m.WriteString(fmt.Sprintf("PERIOD: %s\n", per))
		m.WriteString(fmt.Sprintf("OUTPUT: %s\n", otp))
		m.WriteString(fmt.Sprintf("EPS: %.3E", eps))
		if af == nil {
			return fmt.Errorf(":arclm101: frame isn't extracted to period %s", per)
		}
		af.Output = stw.HistoryWriter()
		if af.Running() {
			return fmt.Errorf("analysis is running")
		}
		m.WriteString(fmt.Sprintf("PERIOD      : %s\n", per))
		m.WriteString(cond.String())
		wait := false
		var wch chan int
		if _, ok := argdict["WAIT"]; ok {
			wait = true
			wch = make(chan int)
		}
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			err := af.Arclm101(cancel, cond)
			af.Endch <- err
		}()
		nlap := cond.Nlap()
		stw.CurrentLap("Calculating...", 0, nlap)
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
		pind := 0
		retval := 0
		currentlap := 0
		go func() {
		read101:
			for {
				select {
				case <-ctx.Done():
					retval = 1
				case <-af.Pivot:
					if stw.Pivot() {
						pivot <- 1
					}
				case lap := <-af.Lapch:
					currentlap = lap
					nper := fmt.Sprintf("%s@%d", pers[pind], lap)
					SetPeriod(stw, nper)
					frame.ReadArclmData(af, nper)
					af.Lapch <- retval
					stw.CurrentLap("Calculating...", lap, nlap)
					stw.Redraw()
				case err := <-af.Endch:
					if stw.Pivot() {
						end <- 1
					}
					if err != nil {
						stw.History(err.Error())
					} else {
						stw.CurrentLap("Completed", nlap, nlap)
					}
					otps := cond.Output()
					if len(otps) > pind {
						frame.ResultFileName[pers[pind]] = otps[pind]
					}
					SetPeriod(stw, fmt.Sprintf("%s@%d", per, currentlap))
					frame.Nlap[per] = currentlap
					frame.ZoubunSummary(filepath.Join(stw.Cwd(), "summary.txt"), per, cond)
					stw.Redraw()
					if wait {
						wch <- 1
					}
					break read101
				}
			}
		}()
		if wait {
			<-wch
		}
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
			m.WriteString(fmt.Sprintf("SOIL SPRING: %v\n", sects))
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
		m.WriteString(fmt.Sprintf("PERIOD: %s\n", per))
		m.WriteString(fmt.Sprintf("OUTPUT: %s\n", otp))
		m.WriteString(fmt.Sprintf("EPS: %.3E", eps))
		af := frame.Arclms[per]
		if af == nil {
			return fmt.Errorf(":arclm301: frame isn't extracted to period %s", per)
		}
		af.Output = stw.HistoryWriter()
		init := true
		if _, ok := argdict["NOINIT"]; ok {
			init = false
			m.WriteString("\nNO INITIALISATION")
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
		m.WriteString(fmt.Sprintf("PERIOD: %s\n", per))
		m.WriteString(fmt.Sprintf("OUTPUT: %s\n", otp))
		m.WriteString(fmt.Sprintf("EPS: %.3E", eps))
		af := frame.Arclms[per]
		if af == nil {
			return fmt.Errorf(":arclm401: frame isn't extracted to period %s", per)
		}
		af.Output = stw.HistoryWriter()
		init := true
		if _, ok := argdict["NOINIT"]; ok {
			init = false
			m.WriteString("\nNO INITIALISATION")
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
	case "bclng001":
		if usage {
			return Usage(":bclng001 {-period=name} {-eps=1e-12} {-noinit} {-mode=1} {-right=10.0} filename")
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
		eps := 1e-12
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
		nmode := 1
		if n, ok := argdict["MODE"]; ok {
			val, err := strconv.ParseInt(n, 10, 64)
			if err == nil {
				nmode = int(val)
			}
		}
		right := 10.0
		if n, ok := argdict["RIGHT"]; ok {
			val, err := strconv.ParseFloat(n, 64)
			if err == nil {
				right = val
			}
		}
		var m bytes.Buffer
		m.WriteString(fmt.Sprintf("PERIOD: %s MODE: %d EPS: %.1E RIGHT %.3f\n", per, nmode, eps, right))
		m.WriteString(fmt.Sprintf("OUTPUT: %s", otp))
		init := true
		if _, ok := argdict["NOINIT"]; ok {
			init = false
			m.WriteString("\nNO INITIALISATION")
		}
		af := frame.Arclms[per]
		if af == nil {
			return fmt.Errorf(":bclng001: frame isn't extracted to period %s", per)
		}
		af.Output = stw.HistoryWriter()
		go func() {
			err := af.Bclng001(otp, init, nmode, eps, right)
			if err != nil {
				fmt.Println(err)
			}
			af.Endch <- err
		}()
		stw.CurrentLap("Calculating...", 0, 1)
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
		readb001:
			for {
				select {
				case <-af.Pivot:
					if stw.Pivot() {
						pivot <- 1
					}
				case nlap := <-af.Lapch:
					frame.ReadArclmData(af, per)
					af.Lapch <- 1
					stw.CurrentLap("Calculating...", nlap, 1)
					if stw.Pivot() {
						end <- 1
						go stw.DrawPivot(nodes, pivot, end)
					} else {
						stw.Redraw()
					}
				case <-af.Endch:
					stw.CurrentLap("Completed", 1, 1)
					stw.Redraw()
					break readb001
				}
			}
		}()
		return ArclmStart(m.String())
	case "vibeig":
		if usage {
			return Usage(":vibeig {-period=name} {-eps=1e-12} {-noinit} {-mode=1} {-right=10.0} filename")
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
		eps := 1e-12
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
		nmode := 1
		if n, ok := argdict["MODE"]; ok {
			val, err := strconv.ParseInt(n, 10, 64)
			if err == nil {
				nmode = int(val)
			}
		}
		right := 10.0
		if n, ok := argdict["RIGHT"]; ok {
			val, err := strconv.ParseFloat(n, 64)
			if err == nil {
				right = val
			}
		}
		var m bytes.Buffer
		m.WriteString(fmt.Sprintf("PERIOD: %s MODE: %d EPS: %.1E RIGHT %.3f\n", per, nmode, eps, right))
		m.WriteString(fmt.Sprintf("OUTPUT: %s", otp))
		init := true
		if _, ok := argdict["NOINIT"]; ok {
			init = false
			m.WriteString("\nNO INITIALISATION")
		}
		af := frame.Arclms[per]
		if af == nil {
			return fmt.Errorf(":vibeig: frame isn't extracted to period %s", per)
		}
		af.Output = stw.HistoryWriter()
		go func() {
			err := af.VibrationalEigenAnalysis(otp, init, nmode, eps, right)
			if err != nil {
				fmt.Println(err)
			}
			af.Endch <- err
		}()
		stw.CurrentLap("Calculating...", 0, 1)
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
		readb001:
			for {
				select {
				case <-af.Pivot:
					if stw.Pivot() {
						pivot <- 1
					}
				case nlap := <-af.Lapch:
					frame.ReadArclmData(af, per)
					af.Lapch <- 1
					stw.CurrentLap("Calculating...", nlap, 1)
					if stw.Pivot() {
						end <- 1
						go stw.DrawPivot(nodes, pivot, end)
					} else {
						stw.Redraw()
					}
				case <-af.Endch:
					stw.CurrentLap("Completed", 1, 1)
					stw.Redraw()
					break readb001
				}
			}
		}()
		return ArclmStart(m.String())
	case "camber":
		if usage {
			return Usage(":camber [axis] [add] period factor")
		}
		if narg < 2 {
			return NotEnoughArgs(":camber")
		}
		if !stw.NodeSelected() {
			return errors.New(":camber: no selected node")
		}
		axis := []bool{true, true, true}
		if ax, ok := argdict["AXIS"]; ok {
			for i := 0; i < 3; i++ {
				axis[i] = false
			}
			for _, a := range strings.Split(ax, ",") {
				a2 := strings.TrimSpace(a)
				if strings.EqualFold(a2, "x") {
					axis[0] = true
				} else if strings.EqualFold(a2, "y") {
					axis[1] = true
				} else if strings.EqualFold(a2, "z") {
					axis[2] = true
				}
			}
		}
		add := false
		if _, ok := argdict["ADD"]; ok {
			add = true
		}
		period := strings.ToUpper(args[1])
		factor, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			return err
		}
		factor *= -1.0
		for _, n := range currentnode(stw, exmodech, exmodeend) {
			for i := 0; i < 3; i++ {
				if axis[i] {
					if add {
						n.Coord[i] += n.Disp[period][i] * factor
					} else {
						n.Coord[i] = n.Disp[period][i] * factor
					}
				}
			}
		}
		Snapshot(stw)
	case "parabola":
		d := 0.2
		if dv, ok := argdict["D"]; ok {
			val, err := strconv.ParseFloat(dv, 64)
			if err == nil {
				d = val
			}
		}
		l := 1.514
		for _, n := range frame.Nodes {
			n.Coord[2] += d / (l * l) * n.Coord[0] * n.Coord[0]
		}
		Snapshot(stw)
	case "skip":
		if usage {
			return Usage(":skip [n,l,x,y,s,[01]{3}]")
		}
		if narg < 2 {
			return NotEnoughArgs(":skip")
		}
		lis := make([]bool, 3)
		pat := regexp.MustCompile("[01]{3}")
		switch {
		case strings.EqualFold(args[1], "n"):
			lis[0] = false
			lis[1] = false
			lis[2] = false
		case strings.EqualFold(args[1], "l"):
			lis[0] = true
		case strings.EqualFold(args[1], "x"):
			lis[1] = true
		case strings.EqualFold(args[1], "y"):
			lis[2] = true
		case strings.EqualFold(args[1], "s"):
			lis[1] = true
			lis[2] = true
		case pat.MatchString(args[1]):
			for i := 0; i < 3; i++ {
				switch args[1][i] {
				case '0':
					lis[i] = false
				case '1':
					lis[i] = true
				}
			}
		}
		for _, el := range currentelem(stw, exmodech, exmodeend) {
			for i := 0; i < 3; i++ {
				el.Skip[i] = lis[i]
			}
		}
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

func currentnode(stw ExModer, exmodech chan interface{}, exmodeend chan int) []*Node {
	var ns []*Node
	if !stw.NodeSelected() {
		nnum := 0
		ns = make([]*Node, 0)
	currentn:
		for {
			select {
			case <-time.After(time.Second):
				break currentn
			case <-exmodeend:
				break currentn
			case n := <-exmodech:
				if n == nil {
					break currentn
				}
				if n, ok := n.(*Node); ok {
					ns = append(ns, n)
					nnum++
				}
			}
		}
		if nnum == 0 {
			return []*Node{}
		}
		ns = ns[:nnum]
	} else {
		ns = stw.SelectedNodes()
	}
	return ns
}
