package stcui

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/yofu/st/stlib"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

var (
	undopos     int
	completepos int
	completes   []string
)

var (
	EPS = 1e-4
)

const (
	nRecentFiles = 3
	nUndo        = 10
)

const ResourceFileName = ".strc"

var (
	gopath      = os.Getenv("GOPATH")
	home        = os.Getenv("HOME")
	releasenote = filepath.Join(home, ".st/help/releasenote.html")
	pgpfile     = filepath.Join(home, ".st/st.pgp")
	recentfn    = filepath.Join(home, ".st/recent.dat")
	historyfn   = filepath.Join(home, ".st/history.dat")
	NOUNDO      = false
)

type Window struct {
	Home   string
	cwd    string
	prompt string

	Frame *st.Frame

	selectNode []*st.Node
	selectElem []*st.Elem

	textBox map[string]*TextBox

	papersize uint

	lastexcommand string
	lastfig2command string

	Changed bool

	recentfiles []string
	undostack   []*st.Frame
	taggedFrame map[string]*st.Frame

	quit chan int
}

func NewWindow(homedir string) *Window {
	stw := new(Window)
	stw.Home = homedir
	stw.cwd = homedir
	stw.prompt = ">"
	stw.selectNode = make([]*st.Node, 0)
	stw.selectElem = make([]*st.Elem, 0)
	stw.papersize = st.A4_TATE
	stw.textBox = make(map[string]*TextBox, 0)
	stw.Changed = false
	stw.recentfiles = make([]string, nRecentFiles)
	stw.SetRecently()
	stw.undostack = make([]*st.Frame, nUndo)
	stw.taggedFrame = make(map[string]*st.Frame)
	undopos = 0
	stw.quit = make(chan int)
	return stw
}

func (stw *Window) MainLoop() {
	go func() {
		select {
		case val := <-stw.quit:
			fmt.Printf("\r")
			os.Exit(val)
		}
	}()
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("%s ", stw.prompt)
	for scanner.Scan() {
		stw.Feed(scanner.Text())
	}
}

func (stw *Window) Feed(command string) {
	if command != "" {
		stw.ExecCommand(command)
	}
	stw.Redraw()
}

func (stw *Window) ExecCommand(command string) {
	if stw.Frame == nil {
		if strings.HasPrefix(command, ":") {
			err := st.ExMode(stw, stw.Frame, command)
			if err != nil {
				stw.ErrorMessage(err, st.ERROR)
			}
		} else if strings.HasPrefix(command, "'") {
			err := st.Fig2Mode(stw, stw.Frame, command)
			if err != nil {
				stw.ErrorMessage(err, st.ERROR)
			}
		}
		return
	}
	switch {
	default:
		stw.History(fmt.Sprintf("command doesn't exist: %s", command))
	case strings.HasPrefix(command, ":"):
		err := st.ExMode(stw, stw.Frame, command)
		if err != nil {
			stw.ErrorMessage(err, st.ERROR)
		}
	case strings.HasPrefix(command, "'"):
		err := st.Fig2Mode(stw, stw.Frame, command)
		if err != nil {
			stw.ErrorMessage(err, st.ERROR)
		}
	}
}

func (stw *Window) Redraw() {
	stw.DrawTexts()
	fmt.Printf("%s ", stw.prompt)
}

func (stw *Window) DrawTexts() {
	var s *st.Show
	if stw.Frame == nil {
		s = nil
	} else {
		s = stw.Frame.Show
	}
	for _, t := range stw.textBox {
		if !t.IsHidden(s) {
			for _, txt := range t.Text() {
				fmt.Println(txt)
			}
		}
	}
}

func (stw *Window) History(str string) {
	fmt.Println(str)
}

func (stw *Window) ErrorMessage(err error, level int) {
	if err == nil {
		return
	}
	var otp string
	if level >= st.ERROR {
		_, file, line, _ := runtime.Caller(1)
		otp = fmt.Sprintf("%s:%d: [%s]: %s", filepath.Base(file), line, st.LOGLEVEL[level], err.Error())
	} else {
		otp = fmt.Sprintf("[%s]: %s", st.LOGLEVEL[level], err.Error())
	}
	stw.History(otp)
}

func (stw *Window) LastExCommand() string {
	return stw.lastexcommand
}

func (stw *Window) SetLastExCommand(c string) {
	stw.lastexcommand = c
}

func (stw *Window) CompleteFileName(str string) string {
	path := ""
	if stw.Frame != nil {
		path = stw.Frame.Path
	}
	completes = st.CompleteFileName(str, path, stw.recentfiles)
	completepos = 0
	return completes[0]
}

func (stw *Window) Cwd() string {
	return stw.cwd
}

func (stw *Window) HomeDir() string {
	return stw.Home
}

func (stw *Window) Print() {
}

func (stw *Window) IsChanged() bool {
	return stw.Changed
}

func (stw *Window) Yn(title, question string) bool {
	fmt.Print(question)
	return false
}

func (stw *Window) Yna(title, question, another string) int {
	return 0
}

func (stw *Window) SaveAS() {
	fn := "hogtxt.inp"
	err := stw.SaveFile(fn)
	if err == nil && fn != stw.Frame.Path {
		stw.Copylsts(fn)
		stw.Rebase(fn)
	}
}

func (stw *Window) SaveFile(fn string) error {
	err := stw.Frame.WriteInp(fn)
	if err != nil {
		return err
	}
	stw.ErrorMessage(fmt.Errorf("SAVE: %s", fn), st.INFO)
	stw.Changed = false
	return nil
}

func (stw *Window) SaveFileSelected(fn string) error {
	els := stw.selectElem
	err := st.WriteInp(fn, stw.Frame.View, stw.Frame.Ai, els)
	if err != nil {
		return err
	}
	stw.ErrorMessage(fmt.Errorf("SAVE: %s", fn), st.INFO)
	stw.Changed = false
	return nil
}

func (stw *Window) SearchFile(fn string) (string, error) {
	if _, err := os.Stat(fn); err == nil {
		return fn, nil
	} else {
		pos1 := strings.IndexAny(fn, "0123456789.")
		if pos1 < 0 {
			return fn, fmt.Errorf("File not fount %s", fn)
		}
		pos2 := strings.IndexAny(fn, "_.")
		if pos2 < 0 {
			return fn, fmt.Errorf("File not fount %s", fn)
		}
		cand := filepath.Join(stw.Home, fn[:pos1], fn[:pos2], fn)
		if st.FileExists(cand) {
			return cand, nil
		} else {
			return fn, fmt.Errorf("File not fount %s", fn)
		}
	}
}

func (stw *Window) OpenFile(filename string, readrcfile bool) error {
	var err error
	var s *st.Show
	fn := st.ToUtf8string(filename)
	frame := st.NewFrame()
	if stw.Frame != nil {
		s = stw.Frame.Show
	}
	switch filepath.Ext(fn) {
	case ".inp":
		err = frame.ReadInp(fn, []float64{0.0, 0.0, 0.0}, 0.0, false)
		if err != nil {
			return err
		}
		stw.Frame = frame
	case ".dxf":
		err = frame.ReadDxf(fn, []float64{0.0, 0.0, 0.0}, EPS)
		if err != nil {
			return err
		}
		stw.Frame = frame
		frame.SetFocus(nil)
	}
	if s != nil {
		stw.Frame.Show = s
		for snum := range stw.Frame.Sects {
			if _, ok := stw.Frame.Show.Sect[snum]; !ok {
				stw.Frame.Show.Sect[snum] = true
			}
		}
	}
	openstr := fmt.Sprintf("OPEN: %s", fn)
	stw.History(openstr)
	stw.Frame.Home = stw.Home
	stw.cwd = filepath.Dir(fn)
	stw.AddRecently(fn)
	stw.Snapshot()
	stw.Changed = false
	if readrcfile {
		if rcfn := filepath.Join(stw.cwd, ResourceFileName); st.FileExists(rcfn) {
			stw.ReadResource(rcfn)
		}
	}
	return nil
}

func (stw *Window) Reload() {
	if stw.Frame != nil {
		stw.Deselect()
		v := stw.Frame.View
		s := stw.Frame.Show
		stw.OpenFile(stw.Frame.Path, false)
		stw.Frame.View = v
		stw.Frame.Show = s
	}
}

func (stw *Window) Close(force bool) {
	if !force && stw.Changed {
		if stw.Yn("CHANGED", "変更を保存しますか") {
			stw.SaveAS()
		} else {
			return
		}
	}
	stw.quit <- 0
}

func (stw *Window) ReadResource(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		txt := s.Text()
		if strings.HasPrefix(txt, "#") {
			continue
		}
		stw.ExecCommand(txt)
	}
	if err := s.Err(); err != nil {
		return err
	}
	return nil
}

func (stw *Window) Checkout(name string) error {
	f, exists := stw.taggedFrame[name]
	if !exists {
		return fmt.Errorf("tag %s doesn't exist", name)
	}
	stw.Frame = f
	return nil
}

func (stw *Window) AddTag(name string, bang bool) error {
	if !bang {
		if _, exists := stw.taggedFrame[name]; exists {
			return fmt.Errorf("tag %s already exists", name)
		}
	}
	stw.taggedFrame[name] = stw.Frame.Snapshot()
	return nil
}

func (stw *Window) Copylsts(name string) {
	if stw.Yn("SAVE AS", ".lst, .fig2, .kjnファイルがあればコピーしますか?") {
		for _, ext := range []string{".lst", ".fig2", ".kjn"} {
			src := st.Ce(stw.Frame.Path, ext)
			dst := st.Ce(name, ext)
			if st.FileExists(src) {
				err := st.CopyFile(src, dst)
				if err == nil {
					stw.History(fmt.Sprintf("COPY: %s", dst))
				}
			}
		}
	}
}

func (stw *Window) ReadFile(filename string) error {
	var err error
	switch filepath.Ext(filename) {
	default:
		return errors.New("Unknown Format")
	case ".inp":
		x := 0.0
		y := 0.0
		z := 0.0
		err = stw.Frame.ReadInp(filename, []float64{x, y, z}, 0.0, false)
	case ".inl", ".ihx", ".ihy":
		err = stw.Frame.ReadData(filename)
	case ".otl", ".ohx", ".ohy":
		err = stw.Frame.ReadResult(filename, st.UpdateResult)
	case ".rat", ".rat2":
		err = stw.Frame.ReadRat(filename)
	case ".lst":
		err = stw.Frame.ReadLst(filename)
	case ".wgt":
		err = stw.Frame.ReadWgt(filename)
	case ".kjn":
		err = stw.Frame.ReadKjn(filename)
	case ".otp":
		err = stw.Frame.ReadBuckling(filename)
	case ".otx", ".oty", ".inc":
		err = stw.Frame.ReadZoubun(filename)
	}
	if err != nil {
		return err
	}
	return nil
}

func (stw *Window) ReadAll() {
	if stw.Frame != nil {
		var err error
		for _, el := range stw.Frame.Elems {
			switch el.Etype {
			case st.WBRACE, st.SBRACE:
				stw.Frame.DeleteElem(el.Num)
			case st.WALL, st.SLAB:
				el.Children = make([]*st.Elem, 2)
			}
		}
		exts := []string{".inl", ".ihx", ".ihy", ".otl", ".ohx", ".ohy", ".rat2", ".wgt", ".lst", ".kjn"}
		read := make([]string, 10)
		nread := 0
		for _, ext := range exts {
			name := st.Ce(stw.Frame.Path, ext)
			err = stw.ReadFile(name)
			if err != nil {
				if ext == ".rat2" {
					err = stw.ReadFile(st.Ce(stw.Frame.Path, ".rat"))
					if err == nil {
						continue
					}
				}
				stw.ErrorMessage(err, st.ERROR)
			} else {
				read[nread] = ext
				nread++
			}
		}
		stw.History(fmt.Sprintf("READ: %s", strings.Join(read, " ")))
	}
}

func (stw *Window) ReadPgp(string) error {
	return nil
}

func (stw *Window) ReadFig2(string) error {
	return nil
}

func (stw *Window) CheckFrame() {
}

func (stw *Window) SelectElem(els []*st.Elem) {
	stw.selectElem = els
}

func (stw *Window) SelectNode(ns []*st.Node) {
	stw.selectNode = ns
}

func (stw *Window) ElemSelected() bool {
	if stw.selectElem == nil || len(stw.selectElem) == 0 {
		return false
	} else {
		return true
	}
}

func (stw *Window) NodeSelected() bool {
	if stw.selectNode == nil || len(stw.selectNode) == 0 {
		return false
	} else {
		return true
	}
}

func (stw *Window) SelectedElems() []*st.Elem {
	return stw.selectElem
}

func (stw *Window) SelectedNodes() []*st.Node {
	return stw.selectNode
}

func (stw *Window) SelectConfed() {
}

func (stw *Window) Deselect() {
	stw.selectNode = make([]*st.Node, 0)
	stw.selectElem = make([]*st.Elem, 0)
}

func (stw *Window) Rebase(fn string) {
	stw.Frame.Name = filepath.Base(fn)
	stw.Frame.Project = st.ProjectName(fn)
	path, err := filepath.Abs(fn)
	if err != nil {
		stw.Frame.Path = fn
	} else {
		stw.Frame.Path = path
	}
	stw.Frame.Home = stw.Home
	stw.AddRecently(fn)
}

func (stw *Window) AddRecently(fn string) error {
	fn = filepath.ToSlash(fn)
	if st.FileExists(recentfn) {
		f, err := os.Open(recentfn)
		if err != nil {
			return err
		}
		stw.recentfiles[0] = fn
		s := bufio.NewScanner(f)
		num := 0
		for s.Scan() {
			if rfn := s.Text(); rfn != fn {
				stw.recentfiles[num+1] = rfn
				num++
			}
			if num >= nRecentFiles-1 {
				break
			}
		}
		f.Close()
		if err := s.Err(); err != nil {
			return err
		}
		w, err := os.Create(recentfn)
		if err != nil {
			return err
		}
		defer w.Close()
		for i := 0; i < nRecentFiles; i++ {
			w.WriteString(fmt.Sprintf("%s\n", stw.recentfiles[i]))
		}
		return nil
	} else {
		w, err := os.Create(recentfn)
		if err != nil {
			return err
		}
		defer w.Close()
		w.WriteString(fmt.Sprintf("%s\n", fn))
		stw.recentfiles[0] = fn
		return nil
	}
}

func (stw *Window) SetRecently() error {
	if st.FileExists(recentfn) {
		f, err := os.Open(recentfn)
		if err != nil {
			return err
		}
		s := bufio.NewScanner(f)
		num := 0
		for s.Scan() {
			if fn := s.Text(); fn != "" {
				stw.History(fmt.Sprintf("%d: %s", num, fn))
				stw.recentfiles[num] = fn
				num++
			}
		}
		if err := s.Err(); err != nil {
			return err
		}
		return nil
	} else {
		return errors.New(fmt.Sprintf("OpenRecently: %s doesn't exist", recentfn))
	}
}

func (stw *Window) ShowRecently() {
	for i, fn := range stw.recentfiles {
		if fn != "" {
			stw.History(fmt.Sprintf("%d: %s", i, fn))
		}
	}
}

func (stw *Window) ShapeData(sh st.Shape) {
	var tb *TextBox
	if t, tok := stw.textBox["SHAPE"]; tok {
		tb = t
	} else {
		tb = NewTextBox()
		tb.hide = false
		tb.position = []int{0, 200}
		stw.textBox["SHAPE"] = tb
	}
	var otp bytes.Buffer
	otp.WriteString(fmt.Sprintf("%s\n", sh.String()))
	otp.WriteString(fmt.Sprintf("A   = %10.4f [cm2]\n", sh.A()))
	otp.WriteString(fmt.Sprintf("Asx = %10.4f [cm2]\n", sh.Asx()))
	otp.WriteString(fmt.Sprintf("Asy = %10.4f [cm2]\n", sh.Asy()))
	otp.WriteString(fmt.Sprintf("Ix  = %10.4f [cm4]\n", sh.Ix()))
	otp.WriteString(fmt.Sprintf("Iy  = %10.4f [cm4]\n", sh.Iy()))
	otp.WriteString(fmt.Sprintf("J   = %10.4f [cm4]\n", sh.J()))
	otp.WriteString(fmt.Sprintf("Zx  = %10.4f [cm3]\n", sh.Zx()))
	otp.WriteString(fmt.Sprintf("Zy  = %10.4f [cm3]\n", sh.Zy()))
	tb.value = strings.Split(otp.String(), "\n")
}

func (stw *Window) Snapshot() {
	stw.Changed = true
	if NOUNDO {
		return
	}
	tmp := make([]*st.Frame, nUndo)
	tmp[0] = stw.Frame.Snapshot()
	for i := 0; i < nUndo-1-undopos; i++ {
		tmp[i+1] = stw.undostack[i+undopos]
	}
	stw.undostack = tmp
	undopos = 0
}

func (stw *Window) UseUndo(yes bool) {
	NOUNDO = !yes
}

func (stw *Window) EPS() float64 {
	return EPS
}

func (stw *Window) SetEPS(val float64) {
	EPS = val
}

func (stw *Window) CanvasFitScale() float64 {
	return 0.0
}

func (stw *Window) SetCanvasFitScale(val float64) {
}

func (stw *Window) CanvasAnimateSpeed() float64 {
	return 0.0
}

func (stw *Window) SetCanvasAnimateSpeed(val float64) {
}

func (stw *Window) ToggleFixRotate() {
}

func (stw *Window) ToggleFixMove() {
}

func (stw *Window) ToggleAltSelectNode() {
}

func (stw *Window) AltSelectNode() bool {
	return false
}

func (stw *Window) SetShowPrintRange(val bool) {
}

func (stw *Window) ToggleShowPrintRange() {
}

func (stw *Window) CurrentLap(comment string, nlap int, laps int) {
	var tb *TextBox
	if t, tok := stw.textBox["LAP"]; tok {
		tb = t
	} else {
		tb = NewTextBox()
		tb.position = []int{30, 30}
		tb.hide = false
		stw.textBox["LAP"] = tb
	}
	if comment == "" {
		tb.value = []string{fmt.Sprintf("LAP: %3d / %3d", nlap, laps)}
	} else {
		tb.value = []string{comment, fmt.Sprintf("LAP: %3d / %3d", nlap, laps)}
	}
}

func (stw *Window) SectionData(sec *st.Sect) {
	var tb *TextBox
	if t, tok := stw.textBox["SECTION"]; tok {
		tb = t
	} else {
		tb = NewTextBox()
		tb.hide = false
		tb.position = []int{0, 30}
		stw.textBox["SECTION"] = tb
	}
	tb.value = strings.Split(sec.InpString(), "\n")
	if al, ok := stw.Frame.Allows[sec.Num]; ok {
		tb.value = append(tb.value, strings.Split(al.String(), "\n")...)
	}
}

func (stw *Window) TextBox(name string) st.TextBox {
	if _, tok := stw.textBox[name]; !tok {
		stw.textBox[name] = NewTextBox()
	}
	return stw.textBox[name]
}

func (stw *Window) AxisRange(axis int, min, max float64, any bool) {
}

func (stw *Window) NextFloor() {
}

func (stw *Window) PrevFloor() {
}

func (stw *Window) SetAngle(phi, theta float64) {
	if stw.Frame != nil {
		stw.Frame.View.Angle[0] = phi
		stw.Frame.View.Angle[1] = theta
	}
}

func (stw *Window) SetPaperSize(s uint) {
	stw.papersize = s
}

func (stw *Window) PaperSize() uint {
	return stw.papersize
}

func (stw *Window) SetPeriod(per string) {
	stw.Frame.Show.Period = per
}

func (stw *Window) Pivot() bool {
	return false
}

func (stw *Window) DrawPivot(nodes []*st.Node, pivot, end chan int) {
}

func (stw *Window) SetColorMode(mode uint) {
	stw.Frame.Show.ColorMode = mode
}

func (stw *Window) SetConf(lis []bool) {
}

func (stw *Window) AddSectionAliase(int, string) {
}

func (stw *Window) DeleteSectionAliase(int) {
}

func (stw *Window) ClearSectionAliase() {
}

func (stw *Window) DeformationOn() {
}

func (stw *Window) DeformationOff() {
}

func (stw *Window) DispOn(int) {
}

func (stw *Window) DispOff(int) {
}

func (stw *Window) SrcanRateOn(...string) {
}

func (stw *Window) SrcanRateOff(...string) {
}

func (stw *Window) StressOn(int, uint) {
}

func (stw *Window) StressOff(int, uint) {
}

func (stw *Window) SetLabel(string, string) {
}

func (stw *Window) EnableLabel(string) {
}

func (stw *Window) DisableLabel(string) {
}

func (stw *Window) ElemCaptionOn(string) {
}

func (stw *Window) ElemCaptionOff(string) {
}

func (stw *Window) NodeCaptionOn(name string) {
	for i, j := range st.NODECAPTIONS {
		if j == name {
			if stw.Frame != nil {
				stw.Frame.Show.NodeCaptionOn(1 << uint(i))
			}
		}
	}
}

func (stw *Window) NodeCaptionOff(name string) {
	for i, j := range st.NODECAPTIONS {
		if j == name {
			if stw.Frame != nil {
				stw.Frame.Show.NodeCaptionOff(1 << uint(i))
			}
		}
	}
}

func (stw *Window) GetCanvasSize() (int, int) {
	return 0, 0
}

func (stw *Window) HideAllSection() {
}

func (stw *Window) HideNotSelected() {
}

func (stw *Window) HideEtype(int) {
}

func (stw *Window) HideSection(int) {
}

func (stw *Window) ShowEtype(int) {
}

func (stw *Window) ShowSection(int) {
}

func (stw *Window) IncrementPeriod(num int) {
	pat := regexp.MustCompile("([a-zA-Z]+)(@[0-9]+)")
	fs := pat.FindStringSubmatch(stw.Frame.Show.Period)
	if len(fs) < 3 {
		return
	}
	if nl, ok := stw.Frame.Nlap[strings.ToUpper(fs[1])]; ok {
		tmp, _ := strconv.ParseInt(fs[2][1:], 10, 64)
		val := int(tmp) + num
		if val < 1 || val > nl {
			return
		}
		per := strings.Replace(stw.Frame.Show.Period, fs[2], fmt.Sprintf("@%d", val), -1)
		stw.Frame.Show.Period = per
	}
}

func (stw *Window) LastFig2Command() string {
	return stw.lastfig2command
}

func (stw *Window) SetLastFig2Command(c string) {
	stw.lastfig2command = c
}

func (stw *Window) ShowCenter() {
	stw.Frame.SetFocus(nil)
	stw.Frame.View.Center[0] = 500.0
	stw.Frame.View.Center[1] = 500.0
}
