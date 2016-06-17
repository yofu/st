package st

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

const ResourceFileName = ".strc"

const (
	dataareaheight = 150
)

type Window interface {
	// DrawOption
	CanvasFitScale() float64
	SetCanvasFitScale(float64)
	CanvasAnimateSpeed() float64
	SetCanvasAnimateSpeed(float64)
	// Directory
	Home() string
	SetHome(string)
	Cwd() string
	SetCwd(string)
	// RecentFiles
	Recent() []string
	AddRecent(string)
	// UndoStack
	UndoEnabled() bool
	PushUndo(*Frame)

	Frame() *Frame
	SetFrame(*Frame)
	ExecCommand(string)
	History(string)
	HistoryWriter() io.Writer
	Yn(string, string) bool
	Yna(string, string, string) int
	EnableLabel(string)
	DisableLabel(string)
	SetLabel(string, string)
	GetCanvasSize() (int, int)
	Changed(bool)
	IsChanged() bool
	Redraw()
	RedrawNode()
	EPS() float64
	SetEPS(float64)
	Close(bool)
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

func ShowCenter(stw Window) {
	frame := stw.Frame()
	if frame == nil {
		return
	}
	if dw, ok := stw.(Drawer); ok {
		w, h := stw.GetCanvasSize()
		frame.SetFocus(nil)
		frame.View.Set(dw.CanvasDirection())
		for _, n := range frame.Nodes {
			frame.View.ProjectNode(n)
		}
		xmin, xmax, ymin, ymax := frame.Bbox2D(true)
		var sx, sy float64
		if xmin == xmax {
			sx = 1e16
		} else {
			sx = float64(w) / (xmax - xmin)
		}
		if ymin == ymax {
			sy = 1e16
		} else {
			sy = float64(h) / (ymax - ymin)
		}
		scale := math.Min(sx, sy) * stw.CanvasFitScale()
		if frame.View.Perspective {
			frame.View.Dists[1] = frame.View.Dists[1] * scale
		} else {
			frame.View.Gfact = frame.View.Gfact * scale
		}
		frame.View.Center[0] = float64(w)*0.5 + scale*(frame.View.Center[0]-0.5*(xmax+xmin))
		frame.View.Center[1] = float64(h)*0.5 + scale*(frame.View.Center[1]-0.5*(ymax+ymin))
	}
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
	w, h := stw.GetCanvasSize()
	frame.View.Center[0] = float64(w) * 0.5
	frame.View.Center[1] = float64(h) * 0.5
	switch filepath.Ext(fn) {
	case ".inp":
		err = newframe.ReadInp(fn, []float64{0.0, 0.0, 0.0}, 0.0, false)
		if err != nil {
			return err
		}
	case ".dxf":
		err = newframe.ReadDxf(fn, []float64{0.0, 0.0, 0.0}, stw.EPS())
		if err != nil {
			return err
		}
		newframe.SetFocus(nil)
	}
	stw.SetFrame(newframe)
	frame = stw.Frame()
	if s != nil {
		frame.Show = s
		for snum := range frame.Sects {
			if _, ok := frame.Show.Sect[snum]; !ok {
				frame.Show.Sect[snum] = true
			}
		}
	}
	frame.Show.LegendPosition[0] = int(w) - 100
	frame.Show.LegendPosition[1] = dataareaheight - int(float64((len(RainbowColor)+1)*frame.Show.LegendSize)*frame.Show.LegendLineSep)
	if dw, ok := stw.(Drawer); ok {
		if dw.CanvasDirection() == 1 {
			frame.Show.LegendPosition[1] = h - frame.Show.LegendPosition[1]
		}
	}
	stw.History(fmt.Sprintf("OPEN: %s", fn))
	frame.Home = stw.Home()
	stw.SetCwd(filepath.Dir(fn))
	stw.AddRecent(fn)
	Snapshot(stw)
	stw.Changed(false)
	ShowCenter(stw)
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
	stw.Changed(false)
	return nil
}

func ReadFile(stw Window, filename string) error {
	var err error
	frame := stw.Frame()
	switch filepath.Ext(filename) {
	default:
		return fmt.Errorf("Unknown Format")
	case ".inp":
		err = frame.ReadInp(filename, []float64{0.0, 0.0, 0.0}, 0.0, false)
	case ".inl", ".ihx", ".ihy":
		err = frame.ReadData(filename)
	case ".otl", ".ohx", ".ohy":
		err = frame.ReadResult(filename, UpdateResult)
	case ".rat", ".rat2":
		err = frame.ReadRat(filename)
	case ".lst":
		err = frame.ReadLst(filename)
	case ".wgt":
		err = frame.ReadWgt(filename)
	case ".kjn":
		err = frame.ReadKjn(filename)
	case ".otp":
		err = frame.ReadBuckling(filename)
	case ".otx", ".oty", ".inc":
		err = frame.ReadZoubun(filename)
	}
	if err != nil {
		return err
	}
	return nil
}

func ReadAll(stw Window) {
	frame := stw.Frame()
	if frame == nil {
		return
	}
	var err error
	for _, el := range frame.Elems {
		switch el.Etype {
		case WBRACE, SBRACE:
			frame.DeleteElem(el.Num)
		case WALL, SLAB:
			el.Children = make([]*Elem, 2)
		}
	}
	exts := []string{".inl", ".ihx", ".ihy", ".otl", ".ohx", ".ohy", ".rat2", ".wgt", ".lst", ".kjn"}
	read := make([]string, 10)
	nread := 0
	for _, ext := range exts {
		name := Ce(frame.Path, ext)
		err = ReadFile(stw, name)
		if err != nil {
			if ext == ".rat2" {
				err = ReadFile(stw, Ce(frame.Path, ".rat"))
				if err == nil {
					continue
				}
			}
			ErrorMessage(stw, err, ERROR)
		} else {
			read[nread] = ext
			nread++
		}
	}
	stw.History(fmt.Sprintf("READ: %s", strings.Join(read, " ")))
}

func ReadFig2(stw Window, filename string) error {
	if _, ok := stw.(Fig2Moder); !ok {
		return fmt.Errorf("Window doesn't implement Fig2Moder")
	}
	frame := stw.Frame()
	fw := stw.(Fig2Moder)
	if frame == nil {
		return fmt.Errorf("ReadFig2: no frame opened")
	}
	tmp := make([][]string, 0)
	var err error
	err = ParseFile(filename, func(words []string) error {
		var err error
		first := strings.ToUpper(words[0])
		if strings.HasPrefix(first, "#") {
			return nil
		}
		switch first {
		default:
			tmp = append(tmp, words)
		case "PAGE", "FIGURE":
			err = ParseFig2(fw, tmp)
			tmp = [][]string{words}
		}
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	err = ParseFig2(fw, tmp)
	if err != nil {
		return err
	}
	stw.Redraw()
	return nil
}

func ParseFig2(stw Fig2Moder, lis [][]string) error {
	var err error
	if len(lis) == 0 || len(lis[0]) < 1 {
		return nil
	}
	first := strings.ToUpper(lis[0][0])
	switch first {
	case "PAGE":
		err = ParseFig2Page(stw, lis)
	case "FIGURE":
		err = ParseFig2Page(stw, lis)
	}
	return err
}

func ParseFig2Page(stw Fig2Moder, lis [][]string) error {
	for _, txt := range lis {
		if len(txt) < 1 {
			continue
		}
		var un bool
		if strings.HasPrefix(txt[0], "!") {
			un = true
			txt[0] = txt[0][1:]
		} else {
			un = false
		}
		err := Fig2Keyword(stw, txt, un)
		if err != nil {
			return err
		}
	}
	stw.Redraw()
	return nil
}

func Copylsts(stw Window, name string) {
	if stw.Yn("SAVE AS", ".lst, .fig2, .kjnファイルがあればコピーしますか?") {
		frame := stw.Frame()
		for _, ext := range []string{".lst", ".fig2", ".kjn"} {
			src := Ce(frame.Path, ext)
			dst := Ce(name, ext)
			if FileExists(src) {
				err := CopyFile(src, dst)
				if err == nil {
					stw.History(fmt.Sprintf("COPY: %s", dst))
				}
			}
		}
	}
}

func ShowRecent(stw Window) {
	for i, fn := range stw.Recent() {
		if fn != "" {
			stw.History(fmt.Sprintf("%d: %s", i, fn))
		}
	}
}

func Snapshot(stw Window) {
	stw.Changed(true)
	if !stw.UndoEnabled() {
		return
	}
	stw.PushUndo(stw.Frame())
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
	var complete func(string, string, string) string
	complete = func(orig, repl, fn string) string {
		rtn := orig
		successor := orig[strings.Index(orig, repl)+len(repl):]
		switch {
		case strings.HasPrefix(successor, ":h"):
			rtn = strings.Replace(orig, fmt.Sprintf("%s:h", repl), filepath.Dir(fn), 1)
		case strings.HasPrefix(successor, ":+"):
			inc, err := Increment(fn, "_", 1, 1)
			if err != nil {
				return rtn
			}
			return complete(strings.Replace(rtn, fmt.Sprintf("%s:+", repl), repl, 1), repl, inc)
		case strings.HasPrefix(successor, "<"):
			rtn = strings.Replace(rtn, fmt.Sprintf("%s<", repl), PruneExt(fn), 1)
		default:
			rtn = strings.Replace(rtn, repl, fn, 1)
		}
		return rtn
	}
	contain := false
	if strings.Contains(str, "%") && percent != "" {
		str = complete(str, "%", percent)
		contain = true
	}
	if len(sharp) > 0 {
		sh := regexp.MustCompile("#([0-9]+)")
		if sh.MatchString(str) {
			sfs := sh.FindStringSubmatch(str)
			if len(sfs) >= 2 {
				tmp, err := strconv.ParseInt(sfs[1], 10, 64)
				if err == nil && int(tmp) < len(sharp) {
					str = complete(str, sfs[0], sharp[int(tmp)])
					contain = true
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
	var num int
	if contain {
		completes = []string{path}
		num = 1
	} else {
		completes = []string{}
		num = 0
	}
	tmp, err := filepath.Glob(path + "*")
	if err != nil || len(tmp) == 0 {
		return []string{path}
	}
	for i := 0; i < len(tmp); i++ {
		stat, err := os.Stat(tmp[i])
		if err != nil {
			continue
		}
		if stat.IsDir() {
			tmp[i] += string(os.PathSeparator)
		}
		lis[len(lis)-1] = tmp[i]
		completes = append(completes, strings.Join(lis, " "))
		num++
	}
	return completes[:num]
}

func PickElem(stw Window, x1, y1, x2, y2 int) ([]*Elem, bool) {
	frame := stw.Frame()
	if frame == nil {
		return nil, false
	}
	fromleft := true
	left, right := x1, x2
	if left > right {
		fromleft = false
		left, right = right, left
	}
	bottom, top := y1, y2
	if bottom > top {
		bottom, top = top, bottom
	}
	if (right-left < elemSelectPixel) && (top-bottom < elemSelectPixel) {
		el := frame.PickLineElem(float64(left), float64(bottom), elemSelectPixel)
		if el == nil {
			els := frame.PickPlateElem(float64(left), float64(bottom))
			if len(els) > 0 {
				el = els[0]
			}
		}
		if el != nil {
			return []*Elem{el}, true
		} else {
			return nil, false
		}
	} else {
		tmpselectnode := make([]*Node, len(frame.Nodes))
		i := 0
		for _, v := range frame.Nodes {
			if float64(left) <= v.Pcoord[0] && v.Pcoord[0] <= float64(right) && float64(bottom) <= v.Pcoord[1] && v.Pcoord[1] <= float64(top) {
				tmpselectnode[i] = v
				i++
			}
		}
		tmpselectelem := make([]*Elem, len(frame.Elems))
		k := 0
		if fromleft {
			for _, el := range frame.Elems {
				if el.IsHidden(frame.Show) {
					continue
				}
				add := true
				for _, en := range el.Enod {
					var j int
					for j = 0; j < i; j++ {
						if en == tmpselectnode[j] {
							break
						}
					}
					if j == i {
						add = false
						break
					}
				}
				if add {
					tmpselectelem[k] = el
					k++
				}
			}
		} else {
			for _, el := range frame.Elems {
				if el.IsHidden(frame.Show) {
					continue
				}
				add := false
				for _, en := range el.Enod {
					found := false
					for j := 0; j < i; j++ {
						if en == tmpselectnode[j] {
							found = true
							break
						}
					}
					if found {
						add = true
						break
					}
				}
				if add {
					tmpselectelem[k] = el
					k++
				}
			}
		}
		return tmpselectelem[:k], true
	}
}

func PickNode(stw Window, x1, y1, x2, y2 int) ([]*Node, bool) {
	frame := stw.Frame()
	if frame == nil {
		return nil, false
	}
	left, right := x1, x2
	if left > right {
		left, right = right, left
	}
	bottom, top := y1, y2
	if bottom > top {
		bottom, top = top, bottom
	}
	if (right-left < nodeSelectPixel) && (top-bottom < nodeSelectPixel) {
		n := frame.PickNode(float64(left), float64(bottom), float64(nodeSelectPixel))
		if n != nil {
			return []*Node{n}, true
		} else {
			return nil, false
		}
	} else {
		tmpselect := make([]*Node, len(frame.Nodes))
		i := 0
		for _, v := range frame.Nodes {
			if v.IsHidden(frame.Show) {
				continue
			}
			if float64(left) <= v.Pcoord[0] && v.Pcoord[0] <= float64(right) && float64(bottom) <= v.Pcoord[1] && v.Pcoord[1] <= float64(top) {
				tmpselect[i] = v
				i++
			}
		}
		return tmpselect[:i], true
	}
}

func Animate(stw Window, view *View) {
	frame := stw.Frame()
	if frame == nil {
		return
	}
	scale := 1.0
	speed := stw.CanvasAnimateSpeed()
	if frame.View.Perspective {
		scale = math.Pow(view.Dists[1]/frame.View.Dists[1], speed)
	} else {
		scale = math.Pow(view.Gfact/frame.View.Gfact, speed)
	}
	center := make([]float64, 2)
	angle := make([]float64, 2)
	focus := make([]float64, 3)
	for i := 0; i < 3; i++ {
		focus[i] = speed * (view.Focus[i] - frame.View.Focus[i])
		if i >= 2 {
			break
		}
		center[i] = speed * (view.Center[i] - frame.View.Center[i])
		angle[i] = view.Angle[i] - frame.View.Angle[i]
		if i == 1 {
			for {
				if angle[1] <= 180.0 {
					break
				}
				angle[1] -= 360.0
			}
			for {
				if angle[1] >= -180.0 {
					break
				}
				angle[1] += 360.0
			}
		}
		angle[i] *= speed
	}
	for i := 0; i < int(1/speed); i++ {
		if frame.View.Perspective {
			frame.View.Dists[1] *= scale
		} else {
			frame.View.Gfact *= scale
		}
		for j := 0; j < 3; j++ {
			frame.View.Focus[j] += focus[j]
			if j >= 2 {
				break
			}
			frame.View.Center[j] += center[j]
			frame.View.Angle[j] += angle[j]
		}
		stw.RedrawNode()
	}
}

func CanvasCenterView(stw Window, angle []float64) *View {
	frame := stw.Frame()
	if frame == nil {
		return nil
	}
	if dw, ok := stw.(Drawer); ok {
		dir := dw.CanvasDirection()
		a0 := make([]float64, 2)
		f0 := make([]float64, 3)
		focus := make([]float64, 3)
		for i := 0; i < 3; i++ {
			f0[i] = frame.View.Focus[i]
			if i >= 2 {
				break
			}
			a0[i] = frame.View.Angle[i]
			frame.View.Angle[i] = angle[i]
		}
		frame.SetFocus(nil)
		frame.View.Set(dir)
		for _, n := range frame.Nodes {
			frame.View.ProjectNode(n)
		}
		xmin, xmax, ymin, ymax := frame.Bbox2D(true)
		for i := 0; i < 3; i++ {
			focus[i] = frame.View.Focus[i]
			frame.View.Focus[i] = f0[i]
			if i >= 2 {
				break
			}
			frame.View.Angle[i] = a0[i]
		}
		frame.View.Set(dir)
		if xmax - xmin <= 1e-6 && ymax - ymin <= 1e-6 {
			return frame.View
		}
		w, h := dw.GetCanvasSize()
		view := frame.View.Copy()
		view.Focus = focus
		scale := math.Min(float64(w)/(xmax-xmin), float64(h)/(ymax-ymin)) * stw.CanvasFitScale()
		if frame.View.Perspective {
			view.Dists[1] = frame.View.Dists[1] * scale
		} else {
			view.Gfact = frame.View.Gfact * scale
		}
		view.Center[0] = float64(w)*0.5 + scale*(frame.View.Center[0]-0.5*(xmax+xmin))
		view.Center[1] = float64(h)*0.5 + scale*(frame.View.Center[1]-0.5*(ymax+ymin))
		view.Angle = angle
		return view
	} else {
		return frame.View
	}
}

func SetPeriod(stw Window, per string) {
	frame := stw.Frame()
	if frame == nil {
		return
	}
	frame.Show.Period = per
	stw.SetLabel("PERIOD", per)
}

func IncrementPeriod(stw Window, num int) {
	frame := stw.Frame()
	if frame == nil {
		return
	}
	pat := regexp.MustCompile("([a-zA-Z]+)(@[0-9]+)")
	fs := pat.FindStringSubmatch(frame.Show.Period)
	if len(fs) < 3 {
		return
	}
	if nl, ok := frame.Nlap[strings.ToUpper(fs[1])]; ok {
		tmp, _ := strconv.ParseInt(fs[2][1:], 10, 64)
		val := int(tmp) + num
		if val < 1 || val > nl {
			return
		}
		per := strings.Replace(frame.Show.Period, fs[2], fmt.Sprintf("@%d", val), -1)
		SetPeriod(stw, per)
	}
}

func NodeCaptionOn(stw Window, name string) {
	frame := stw.Frame()
	if frame == nil {
		return
	}
	for i, j := range NODECAPTIONS {
		if j == name {
			stw.EnableLabel(name)
			frame.Show.NodeCaptionOn(1 << uint(i))
			return
		}
	}
}

func NodeCaptionOff(stw Window, name string) {
	frame := stw.Frame()
	if frame == nil {
		return
	}
	for i, j := range NODECAPTIONS {
		if j == name {
			stw.DisableLabel(name)
			frame.Show.NodeCaptionOff(1 << uint(i))
			return
		}
	}
}

func ElemCaptionOn(stw Window, name string) {
	frame := stw.Frame()
	if frame == nil {
		return
	}
	for i, j := range ELEMCAPTIONS {
		if j == name {
			stw.EnableLabel(name)
			frame.Show.ElemCaptionOn(1 << uint(i))
			return
		}
	}
}

func ElemCaptionOff(stw Window, name string) {
	frame := stw.Frame()
	if frame == nil {
		return
	}
	for i, j := range ELEMCAPTIONS {
		if j == name {
			stw.DisableLabel(name)
			frame.Show.ElemCaptionOff(1 << uint(i))
			return
		}
	}
}

func SrcanRateOn(stw Window, names ...string) {
	frame := stw.Frame()
	if frame == nil {
		return
	}
	defer func() {
		if frame.Show.SrcanRate != 0 {
			stw.EnableLabel("SRCAN_RATE")
			frame.Show.ColorMode = ECOLOR_RATE
		}
	}()
	if len(names) == 0 {
		for i, j := range SRCANS {
			frame.Show.SrcanRateOn(1 << uint(i))
			stw.EnableLabel(j)
		}
		return
	}
	for _, name := range names {
		for i, j := range SRCANS {
			if j == name {
				frame.Show.SrcanRateOn(1 << uint(i))
				stw.EnableLabel(j)
			}
		}
	}
}

func SrcanRateOff(stw Window, names ...string) {
	frame := stw.Frame()
	if frame == nil {
		return
	}
	defer func() {
		if frame.Show.SrcanRate == 0 {
			stw.DisableLabel("SRCAN_RATE")
			frame.Show.ColorMode = ECOLOR_SECT
		}
	}()
	if len(names) == 0 {
		for i, j := range SRCANS {
			frame.Show.SrcanRateOff(1 << uint(i))
			stw.DisableLabel(j)
		}
		return
	}
	for _, name := range names {
		for i, j := range SRCANS {
			if j == name {
				frame.Show.SrcanRateOff(1 << uint(i))
				stw.DisableLabel(j)
			}
		}
	}
}

func StressOn(stw Window, etype int, index uint) {
	frame := stw.Frame()
	if frame == nil {
		return
	}
	frame.Show.Stress[etype] |= (1 << index)
	if etype <= SLAB {
		stw.EnableLabel(fmt.Sprintf("%s_%s", ETYPES[etype], strings.ToUpper(StressName[index])))
	}
}

func StressOff(stw Window, etype int, index uint) {
	frame := stw.Frame()
	if frame == nil {
		return
	}
	frame.Show.Stress[etype] &= ^(1 << index)
	if etype <= SLAB {
		stw.DisableLabel(fmt.Sprintf("%s_%s", ETYPES[etype], strings.ToUpper(StressName[index])))
	}
}

func DeformationOn(stw Window) {
	frame := stw.Frame()
	if frame == nil {
		return
	}
	frame.Show.Deformation = true
	stw.EnableLabel("DEFORMATION")
}

func DeformationOff(stw Window) {
	frame := stw.Frame()
	if frame == nil {
		return
	}
	frame.Show.Deformation = false
	stw.DisableLabel("DEFORMATION")
}

func DispOn(stw Window, direction int) {
	frame := stw.Frame()
	if frame == nil {
		return
	}
	name := fmt.Sprintf("NC_%s", DispName[direction])
	for i, str := range NODECAPTIONS {
		if name == str {
			frame.Show.NodeCaption |= (1 << uint(i))
			stw.EnableLabel(name)
			return
		}
	}
}

func DispOff(stw Window, direction int) {
	frame := stw.Frame()
	if frame == nil {
		return
	}
	name := fmt.Sprintf("NC_%s", DispName[direction])
	for i, str := range NODECAPTIONS {
		if name == str {
			frame.Show.NodeCaption &= ^(1 << uint(i))
			stw.DisableLabel(name)
			return
		}
	}
}

func HideAllSection(stw Window) {
	frame := stw.Frame()
	if frame == nil {
		return
	}
	for i, _ := range frame.Show.Sect {
		frame.Show.Sect[i] = false
		stw.DisableLabel(fmt.Sprintf("%d", i))
	}
}

func ShowAllSection(stw Window) {
	frame := stw.Frame()
	if frame == nil {
		return
	}
	for i, _ := range frame.Show.Sect {
		frame.Show.Sect[i] = true
		stw.EnableLabel(fmt.Sprintf("%d", i))
	}
}

func HideSection(stw Window, snum int) {
	frame := stw.Frame()
	if frame == nil {
		return
	}
	frame.Show.Sect[snum] = false
	stw.DisableLabel(fmt.Sprintf("%d", snum))
}

func ShowSection(stw Window, snum int) {
	frame := stw.Frame()
	if frame == nil {
		return
	}
	frame.Show.Sect[snum] = true
	stw.EnableLabel(fmt.Sprintf("%d", snum))
}

func HideEtype(stw Window, etype int) {
	if etype == 0 {
		return
	}
	frame := stw.Frame()
	if frame == nil {
		return
	}
	frame.Show.Etype[etype] = false
	stw.DisableLabel(ETYPES[etype])
}

func ShowEtype(stw Window, etype int) {
	if etype == 0 {
		return
	}
	frame := stw.Frame()
	if frame == nil {
		return
	}
	frame.Show.Etype[etype] = true
	stw.EnableLabel(ETYPES[etype])
}

func ToggleEtype(stw Window, etype int) {
	frame := stw.Frame()
	if frame == nil {
		return
	}
	if frame.Show.Etype[etype] {
		HideEtype(stw, etype)
	} else {
		ShowEtype(stw, etype)
	}
}

func NextFloor(stw Window) {
	frame := stw.Frame()
	if frame == nil {
		return
	}
	for _, n := range frame.Nodes {
		n.Show()
	}
	for _, el := range frame.Elems {
		el.Show()
	}
	for i, z := range []string{"ZMIN", "ZMAX"} {
		tmpval := frame.Show.Zrange[i]
		ind := 0
		for _, ht := range frame.Ai.Boundary {
			if ht > tmpval {
				break
			}
			ind++
		}
		var val float64
		l := len(frame.Ai.Boundary)
		if ind >= l-1 {
			val = frame.Ai.Boundary[l-2+i]
		} else {
			val = frame.Ai.Boundary[ind]
		}
		frame.Show.Zrange[i] = val
		stw.SetLabel(z, fmt.Sprintf("%.3f", val))
	}
}

func PrevFloor(stw Window) {
	frame := stw.Frame()
	if frame == nil {
		return
	}
	for _, n := range frame.Nodes {
		n.Show()
	}
	for _, el := range frame.Elems {
		el.Show()
	}
	for i, z := range []string{"ZMIN", "ZMAX"} {
		tmpval := frame.Show.Zrange[i]
		ind := 0
		for _, ht := range frame.Ai.Boundary {
			if ht > tmpval {
				break
			}
			ind++
		}
		var val float64
		if ind <= 2 {
			val = frame.Ai.Boundary[i]
		} else {
			val = frame.Ai.Boundary[ind-2]
		}
		frame.Show.Zrange[i] = val
		stw.SetLabel(z, fmt.Sprintf("%.3f", val))
	}
}

func AxisRange(stw Window, axis int, min, max float64, any bool) {
	frame := stw.Frame()
	if frame == nil {
		return
	}
	tmpnodes := make([]*Node, 0)
	for _, n := range frame.Nodes {
		if !(min <= n.Coord[axis] && n.Coord[axis] <= max) {
			tmpnodes = append(tmpnodes, n)
			n.Hide()
		} else {
			n.Show()
		}
	}
	var tmpelems []*Elem
	if !any {
		tmpelems = frame.NodeToElemAny(tmpnodes...)
	} else {
		tmpelems = frame.NodeToElemAll(tmpnodes...)
	}
	for _, el := range frame.Elems {
		el.Show()
	}
	for _, el := range tmpelems {
		el.Hide()
	}
	switch axis {
	case 0:
		frame.Show.Xrange[0] = min
		frame.Show.Xrange[1] = max
		stw.SetLabel("XMIN", fmt.Sprintf("%.3f", min))
		stw.SetLabel("XMAX", fmt.Sprintf("%.3f", max))
	case 1:
		frame.Show.Yrange[0] = min
		frame.Show.Yrange[1] = max
		stw.SetLabel("YMIN", fmt.Sprintf("%.3f", min))
		stw.SetLabel("YMAX", fmt.Sprintf("%.3f", max))
	case 2:
		frame.Show.Zrange[0] = min
		frame.Show.Zrange[1] = max
		stw.SetLabel("ZMIN", fmt.Sprintf("%.3f", min))
		stw.SetLabel("ZMAX", fmt.Sprintf("%.3f", max))
	}
}

func ShowAll(stw Window) {
	frame := stw.Frame()
	if frame == nil {
		return
	}
	for _, el := range frame.Elems {
		el.Show()
	}
	for _, n := range frame.Nodes {
		n.Show()
	}
	for _, k := range frame.Kijuns {
		k.Show()
	}
	for i, et := range ETYPES {
		if i == WBRACE || i == SBRACE {
			continue
		}
		frame.Show.Etype[i] = true
		stw.EnableLabel(et)
	}
	ShowAllSection(stw)
	frame.Show.All()
	stw.SetLabel("XMIN", fmt.Sprintf("%.3f", frame.Show.Xrange[0]))
	stw.SetLabel("XMAX", fmt.Sprintf("%.3f", frame.Show.Xrange[1]))
	stw.SetLabel("YMIN", fmt.Sprintf("%.3f", frame.Show.Yrange[0]))
	stw.SetLabel("YMAX", fmt.Sprintf("%.3f", frame.Show.Yrange[1]))
	stw.SetLabel("ZMIN", fmt.Sprintf("%.3f", frame.Show.Zrange[0]))
	stw.SetLabel("ZMAX", fmt.Sprintf("%.3f", frame.Show.Zrange[1]))
	stw.Redraw()
}

func SetFocus(stw Window) {
	frame := stw.Frame()
	if frame == nil {
		return
	}
	var focus []float64
	if sel, ok := stw.(Selector); ok {
		if sel.ElemSelected() {
			for _, el := range sel.SelectedElems() {
				if el == nil {
					continue
				}
				focus = el.MidPoint()
			}
		}
		if focus == nil {
			if sel.NodeSelected() {
				for _, n := range sel.SelectedNodes() {
					if n == nil {
						continue
					}
					focus = n.Coord
				}
			}
		}
	}
	v := frame.View.Copy()
	frame.SetFocus(focus)
	w, h := stw.GetCanvasSize()
	frame.View.Center[0] = float64(w) * 0.5
	frame.View.Center[1] = float64(h) * 0.5
	view := frame.View.Copy()
	frame.View = v
	Animate(stw, view)
	stw.Redraw()
}
