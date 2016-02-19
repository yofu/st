package st

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

const ResourceFileName = ".strc"

type Window interface {
	Frame() *Frame
	SetFrame(*Frame)
	Home() string
	SetHome(string)
	Cwd() string
	SetCwd(string)
	ExecCommand(string)
	History(string)
	AddRecent(string) error
	Snapshot()
	GetCanvasSize() (int, int)
	CanvasFitScale() float64
	SetCanvasFitScale(float64)
	CanvasAnimateSpeed() float64
	SetCanvasAnimateSpeed(float64)
	Changed(bool)
	IsChanged() bool
	Redraw()
	EPS() float64
	SetEPS(float64)
}

func ErrorMessage(stw Window, err error, level int) {
	if err == nil {
		return
	}
	var otp string
	if level >= ERROR {
		_, file, line, _ := runtime.Caller(1)
		otp = fmt.Sprintf("%s:%d: [%s]: %s", filepath.Base(file), line, LOGLEVEL[level], err.Error())
	} else {
		otp = fmt.Sprintf("[%s]: %s", LOGLEVEL[level], err.Error())
	}
	stw.History(otp)
}

func OpenFile(stw Window, filename string, readrcfile bool) error {
	var err error
	var s *Show
	fn := ToUtf8string(filename)
	newframe := NewFrame()
	frame := stw.Frame()
	if frame != nil {
		s = frame.Show
	}
	switch filepath.Ext(fn) {
	case ".inp":
		err = newframe.ReadInp(fn, []float64{0.0, 0.0, 0.0}, 0.0, false)
		if err != nil {
			return err
		}
		stw.SetFrame(newframe)
	case ".dxf":
		err = newframe.ReadDxf(fn, []float64{0.0, 0.0, 0.0}, stw.EPS())
		if err != nil {
			return err
		}
		newframe.SetFocus(nil)
		stw.SetFrame(newframe)
	}
	frame = stw.Frame()
	if s != nil {
		frame.Show = s
		for snum := range frame.Sects {
			if _, ok := frame.Show.Sect[snum]; !ok {
				frame.Show.Sect[snum] = true
			}
		}
	}
	stw.History(fmt.Sprintf("OPEN: %s", fn))
	frame.Home = stw.Home()
	stw.SetCwd(filepath.Dir(fn))
	stw.AddRecent(fn)
	stw.Snapshot()
	stw.Changed(false)
	if readrcfile {
		if rcfn := filepath.Join(stw.Cwd(), ResourceFileName); FileExists(rcfn) {
			ReadResource(stw, rcfn)
		}
	}
	return nil
}

func Rebase(stw Window, fn string) {
	frame := stw.Frame()
	frame.Name = filepath.Base(fn)
	frame.Project = ProjectName(fn)
	path, err := filepath.Abs(fn)
	if err != nil {
		frame.Path = fn
	} else {
		frame.Path = path
	}
	frame.Home = stw.Home()
	stw.AddRecent(fn)
}

func Reload(stw Window) {
	frame := stw.Frame()
	if frame != nil {
		if s, ok := stw.(Selector); ok {
			s.Deselect()
		}
		v := frame.View
		s := frame.Show
		OpenFile(stw, frame.Path, false)
		frame.View = v
		frame.Show = s
	}
}

func SaveFile(stw Window, filename string) error {
	var v *View
	frame := stw.Frame()
	if !frame.View.Perspective {
		v = frame.View.Copy()
		frame.View.Gfact = 1.0
		frame.View.Perspective = true
		for _, n := range frame.Nodes {
			frame.View.ProjectNode(n)
		}
		xmin, xmax, ymin, ymax := frame.Bbox2D(true)
		w, h := stw.GetCanvasSize()
		scale := math.Min(float64(w)/(xmax-xmin), float64(h)/(ymax-ymin)) * stw.CanvasFitScale()
		frame.View.Dists[1] *= scale
	}
	err := frame.WriteInp(filename)
	if v != nil {
		frame.View = v
	}
	if err != nil {
		return err
	}
	ErrorMessage(stw, fmt.Errorf("SAVE: %s", filename), INFO)
	stw.Changed(true)
	return nil
}

func ReadResource(stw Window, filename string) error {
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

func CompleteFileName(str string, percent string, sharp []string) []string {
	envval := regexp.MustCompile("[$]([a-zA-Z]+)")
	if envval.MatchString(str) {
		efs := envval.FindStringSubmatch(str)
		if len(efs) >= 2 {
			val := os.Getenv(strings.ToUpper(efs[1]))
			if val != "" {
				str = strings.Replace(str, efs[0], val, 1)
			}
		}
	}
	if strings.HasPrefix(str, "~") {
		home := os.Getenv("HOME")
		if home != "" {
			str = strings.Replace(str, "~", home, 1)
		}
	}
	if strings.Contains(str, "%") && percent != "" {
		str = strings.Replace(str, "%:h", filepath.Dir(percent), 1)
		str = strings.Replace(str, "%<", PruneExt(percent), 1)
		str = strings.Replace(str, "%", percent, 1)
	}
	if len(sharp) > 0 {
		sh := regexp.MustCompile("#([0-9]+)")
		if sh.MatchString(str) {
			sfs := sh.FindStringSubmatch(str)
			if len(sfs) >= 2 {
				tmp, err := strconv.ParseInt(sfs[1], 10, 64)
				if err == nil && int(tmp) < len(sharp) {
					str = strings.Replace(str, sfs[0], sharp[int(tmp)], 1)
				}
			}
		}
	}
	lis := strings.Split(str, " ")
	path := lis[len(lis)-1]
	if !filepath.IsAbs(path) {
		path = filepath.Join(filepath.Dir(percent), path)
	}
	var err error
	var completes []string
	tmp, err := filepath.Glob(path + "*")
	if err != nil || len(tmp) == 0 {
		completes = make([]string, 0)
	} else {
		completes = make([]string, len(tmp))
		for i := 0; i < len(tmp); i++ {
			stat, err := os.Stat(tmp[i])
			if err != nil {
				continue
			}
			if stat.IsDir() {
				tmp[i] += string(os.PathSeparator)
			}
			lis[len(lis)-1] = tmp[i]
			completes[i] = strings.Join(lis, " ")
		}
	}
	if len(completes) == 0 {
		completes = []string{path}
	}
	return completes
}
