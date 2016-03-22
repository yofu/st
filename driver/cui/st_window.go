package stcui

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/yofu/st/stlib"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	completepos int
	completes   []string
)

var (
	EPS = 1e-4
)

const ResourceFileName = ".strc"

var (
	gopath      = os.Getenv("GOPATH")
	home        = os.Getenv("HOME")
	releasenote = filepath.Join(home, ".st/help/releasenote.html")
	pgpfile     = filepath.Join(home, ".st/st.pgp")
	historyfn   = filepath.Join(home, ".st/history.dat")
)

type Window struct {
	*st.DrawOption
	*st.Directory
	*st.RecentFiles
	*st.UndoStack
	*st.TagFrame
	*st.Selection
	*st.CommandBuffer
	*st.CommandLine
	*st.Alias
	prompt string

	frame *st.Frame

	textBox map[string]*TextBox

	papersize uint

	lastexcommand   string
	lastfig2command string

	changed bool

	quit chan int
}

func NewWindow(homedir string) *Window {
	stw := &Window{
		DrawOption:    st.NewDrawOption(),
		Directory:   st.NewDirectory(homedir, homedir),
		RecentFiles: st.NewRecentFiles(3),
		UndoStack:   st.NewUndoStack(10),
		TagFrame:    st.NewTagFrame(),
		Selection:   st.NewSelection(),
		CommandBuffer: st.NewCommandBuffer(),
		CommandLine:   st.NewCommandLine(),
		Alias:         st.NewAlias(),
	}
	stw.prompt = ">"
	stw.papersize = st.A4_TATE
	stw.textBox = make(map[string]*TextBox, 0)
	stw.changed = false
	stw.ReadRecent()
	st.ShowRecent(stw)
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

func (stw *Window) Frame() *st.Frame {
	return stw.frame
}

func (stw *Window) SetFrame(frame *st.Frame) {
	stw.frame = frame
}

func (stw *Window) ExecCommand(command string) {
	if stw.frame == nil {
		if strings.HasPrefix(command, ":") {
			err := st.ExMode(stw, command)
			if err != nil {
				stw.ErrorMessage(err, st.ERROR)
			}
		} else if strings.HasPrefix(command, "'") {
			err := st.Fig2Mode(stw, command)
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
		err := st.ExMode(stw, command)
		if err != nil {
			stw.ErrorMessage(err, st.ERROR)
		}
	case strings.HasPrefix(command, "'"):
		err := st.Fig2Mode(stw, command)
		if err != nil {
			stw.ErrorMessage(err, st.ERROR)
		}
	}
}

func (stw *Window) Redraw() {
	stw.DrawTexts()
	fmt.Printf("%s ", stw.prompt)
}

func (stw *Window) RedrawNode() {
	stw.DrawTexts()
	fmt.Printf("%s ", stw.prompt)
}

func (stw *Window) DrawTexts() {
	var s *st.Show
	if stw.frame == nil {
		s = nil
	} else {
		s = stw.frame.Show
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

func (stw *Window) HistoryWriter() io.Writer {
	return os.Stdout
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
	if stw.frame != nil {
		path = stw.frame.Path
	}
	completes = st.CompleteFileName(str, path, stw.Recent())
	completepos = 0
	return completes[0]
}

func (stw *Window) Print() {
}

func (stw *Window) Changed(c bool) {
	stw.changed = c
}

func (stw *Window) IsChanged() bool {
	return stw.changed
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
	if err == nil && fn != stw.frame.Path {
		st.Copylsts(stw, fn)
		st.Rebase(stw, fn)
	}
}

func (stw *Window) SaveFile(fn string) error {
	return st.SaveFile(stw, fn)
}

func (stw *Window) SaveFileSelected(fn string) error {
	els := stw.SelectedElems()
	err := st.WriteInp(fn, stw.frame.View, stw.frame.Ai, els)
	if err != nil {
		return err
	}
	stw.ErrorMessage(fmt.Errorf("SAVE: %s", fn), st.INFO)
	stw.changed = false
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
		cand := filepath.Join(stw.Home(), fn[:pos1], fn[:pos2], fn)
		if st.FileExists(cand) {
			return cand, nil
		} else {
			return fn, fmt.Errorf("File not fount %s", fn)
		}
	}
}

func (stw *Window) Close(force bool) {
	if !force && stw.changed {
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

func (stw *Window) ReadPgp(string) error {
	return nil
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

func (stw *Window) EPS() float64 {
	return EPS
}

func (stw *Window) SetEPS(val float64) {
	EPS = val
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

func (stw *Window) SetAltSelectNode(a bool) {
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
	if al, ok := stw.frame.Allows[sec.Num]; ok {
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
	if stw.frame != nil {
		stw.frame.View.Angle[0] = phi
		stw.frame.View.Angle[1] = theta
	}
}

func (stw *Window) SetPaperSize(s uint) {
	stw.papersize = s
}

func (stw *Window) PaperSize() uint {
	return stw.papersize
}

func (stw *Window) SetPeriod(per string) {
	stw.frame.Show.Period = per
}

func (stw *Window) Pivot() bool {
	return false
}

func (stw *Window) DrawPivot(nodes []*st.Node, pivot, end chan int) {
}

func (stw *Window) SetColorMode(mode uint) {
	stw.frame.Show.ColorMode = mode
}

func (stw *Window) SetConf(lis []bool) {
}

func (stw *Window) SetLabel(string, string) {
}

func (stw *Window) EnableLabel(string) {
}

func (stw *Window) DisableLabel(string) {
}

func (stw *Window) GetCanvasSize() (int, int) {
	return 0, 0
}

func (stw *Window) LastFig2Command() string {
	return stw.lastfig2command
}

func (stw *Window) SetLastFig2Command(c string) {
	stw.lastfig2command = c
}

func (stw *Window) ShowCenter() {
	stw.frame.SetFocus(nil)
	stw.frame.View.Center[0] = 500.0
	stw.frame.View.Center[1] = 500.0
}
