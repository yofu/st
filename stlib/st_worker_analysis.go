package st

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"maps"
	"os"
	"strings"

	"github.com/yofu/st/arclm"
)

type workerNode struct {
	Coord, Load, Weight, Factor, Phase []float64
	Conf                               []bool
	Disp, Reaction                     map[string][]float64
}

type workerElem struct {
	Num, Type, Section, Parent int
	Ends                       []int
	Bonds                      []int
	Cang, Prestress            float64
	Cmq, Wrect                 []float64
	Skip                       []bool
	Stress                     map[string]map[int][]float64
	Phinge                     map[string]map[int]bool
	Values                     map[string]float64
	Rate                       map[string][]float64
	MaxRate                    []float64
	SrcalTex                   string
	Lock                       bool
	Eldest                     bool
}

type workerSection struct {
	Num, Type, Original, Color int
	Exp, Exq                   float64
	Lload, Perpl, Yield        []float64
	Values                     []map[string]float64
	FigProps                   []int
	ElasticEnds                [2]bool
}

type workerModel struct {
	Input                           []byte
	Path, Cwd, Command, Period, Lst string
	Nodes                           map[int]workerNode
	Elems                           []workerElem
	Sections                        []workerSection
	Properties                      map[int][6]float64
	Bonds                           map[int][]float64
	Ai                              *Aiparameter
	Wind                            *Windparameter
	Frames                          map[string][]byte
	ResultFiles, DataFiles          map[string]string
	Laps                            map[string]int
}

type workerResult struct {
	Period                 string
	Deform                 bool
	Nodes                  map[int]workerNode
	Elems                  []workerElem
	Sections               []workerSection
	Frames                 map[string][]byte
	ResultFiles, DataFiles map[string]string
	Laps                   map[string]int
}

func workerFrames(frames map[string]*arclm.Frame) (map[string][]byte, error) {
	result := make(map[string][]byte, len(frames))
	for name, frame := range frames {
		if frame == nil {
			continue
		}
		var data bytes.Buffer
		if err := arclm.EncodeWorkerFrame(&data, frame); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		result[name] = data.Bytes()
	}
	return result, nil
}

func restoreWorkerFrames(frames map[string][]byte) (map[string]*arclm.Frame, error) {
	result := make(map[string]*arclm.Frame, len(frames))
	for name, data := range frames {
		frame, err := arclm.DecodeWorkerFrame(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		result[name] = frame
	}
	return result, nil
}

func workerElement(el *Elem) workerElem {
	e := workerElem{Num: el.Num, Type: el.Etype, Section: el.Sect.Num, Cang: el.Cang, Prestress: el.Prestress,
		Cmq: el.Cmq, Wrect: el.Wrect, Skip: el.Skip, Stress: el.Stress, Phinge: el.Phinge, Values: el.Values,
		Rate: el.Rate, MaxRate: el.MaxRate, SrcalTex: el.SrcalTex, Lock: el.Lock, Eldest: el.Eldest}
	if el.Parent != nil {
		e.Parent = el.Parent.Num
	}
	for _, n := range el.Enod {
		e.Ends = append(e.Ends, n.Num)
	}
	for _, b := range el.Bonds {
		id := 0
		if b != nil {
			id = b.Num
		}
		e.Bonds = append(e.Bonds, id)
	}
	return e
}

func workerSections(f *Frame) []workerSection {
	var result []workerSection
	for _, sec := range f.Sects {
		s := workerSection{Num: sec.Num, Type: sec.Type, Original: sec.Original, Color: sec.Color, Exp: sec.Exp, Exq: sec.Exq,
			Lload: sec.Lload, Perpl: sec.Perpl, Yield: sec.Yield, ElasticEnds: sec.ElasticEnds}
		for _, fig := range sec.Figs {
			s.Values = append(s.Values, fig.Value)
			id := -1
			if fig.Prop != nil {
				id = fig.Prop.Num
			}
			s.FigProps = append(s.FigProps, id)
		}
		result = append(result, s)
	}
	return result
}

// WriteWorkerAnalysis sends the current model and exact solver state. The INP
// carries format-specific definitions; numeric analysis fields are kept in full precision.
func WriteWorkerAnalysis(out io.Writer, f *Frame, cwd, command string) error {
	frames, err := workerFrames(f.Arclms)
	if err != nil {
		return err
	}
	m := workerModel{Input: f.InpBytes(), Path: f.Path, Cwd: cwd, Command: command, Period: f.Show.Period, Lst: f.LstFileName,
		Nodes: make(map[int]workerNode, len(f.Nodes)), Sections: workerSections(f), Frames: frames, Ai: f.Ai, Wind: f.Wind,
		Properties: make(map[int][6]float64), Bonds: make(map[int][]float64),
		ResultFiles: f.ResultFileName, DataFiles: f.DataFileName, Laps: f.Nlap}
	name, _, _, _ := ExModeComplete(strings.Fields(command)[0])
	for id, n := range f.Nodes {
		data := workerNode{Coord: n.Coord, Conf: n.Conf, Load: n.Load, Weight: n.Weight, Factor: n.Factor, Phase: n.Phase,
			Disp: make(map[string][]float64), Reaction: make(map[string][]float64)}
		for period, d := range n.Disp {
			if period == "L" || period == "X" || period == "Y" ||
				(name == "arclm101" && strings.Contains(period, "@") && !strings.HasPrefix(period, "TH@")) {
				data.Disp[period] = d
				data.Reaction[period] = n.Reaction[period]
			}
		}
		m.Nodes[id] = data
	}
	for _, e := range f.Elems {
		data := workerElement(e)
		data.Stress, data.Phinge = nil, nil
		if name == "arclm101" {
			data.Stress = make(map[string]map[int][]float64)
			for period, values := range e.Stress {
				if strings.Contains(period, "@") && !strings.HasPrefix(period, "TH@") {
					data.Stress[period] = values
				}
			}
		}
		m.Elems = append(m.Elems, data)
	}
	for id, p := range f.Props {
		m.Properties[id] = [6]float64{p.Hiju(), p.EL(), p.ES(), p.Poi(), p.HFactor, p.EFactor}
	}
	for id, b := range f.Bonds {
		m.Bonds[id] = b.Stiffness
	}
	return gob.NewEncoder(out).Encode(m)
}

func (m *workerModel) frame() (*Frame, error) {
	f := NewFrame()
	if err := f.ReadInpString(string(m.Input), m.Path, nil, 0, false); err != nil {
		return nil, err
	}
	f.Path, f.Ai, f.Wind = m.Path, m.Ai, m.Wind
	f.Show.Period = m.Period
	for id, p := range m.Properties {
		if prop := f.Props[id]; prop != nil {
			prop.hiju, prop.eL, prop.eS, prop.poi = p[0], p[1], p[2], p[3]
			prop.HFactor, prop.EFactor = p[4], p[5]
		}
	}
	for id, stiffness := range m.Bonds {
		if b := f.Bonds[id]; b != nil {
			b.Stiffness = stiffness
		}
	}
	for _, s := range m.Sections {
		if err := installWorkerSection(f, s); err != nil {
			return nil, err
		}
	}
	for id, data := range m.Nodes {
		n := f.Nodes[id]
		if n == nil {
			return nil, fmt.Errorf("missing node %d", id)
		}
		n.Coord, n.Conf, n.Load = data.Coord, data.Conf, data.Load
		n.Weight, n.Factor, n.Phase = data.Weight, data.Factor, data.Phase
		maps.Copy(n.Disp, data.Disp)
		maps.Copy(n.Reaction, data.Reaction)
	}
	name, _, _, _ := ExModeComplete(strings.Fields(m.Command)[0])
	for _, e := range m.Elems {
		if (name == "extractarclm" || name == "all") && (e.Type == WBRACE || e.Type == SBRACE) {
			continue
		}
		if err := installWorkerElement(f, e); err != nil {
			return nil, err
		}
	}
	attachWorkerBraces(f, m.Elems)
	f.Nlap, f.ResultFileName, f.DataFileName = m.Laps, m.ResultFiles, m.DataFiles
	if m.Lst != "" && FileExists(m.Lst) {
		if err := f.ReadLst(m.Lst); err != nil {
			return nil, err
		}
	}
	var err error
	f.Arclms, err = restoreWorkerFrames(m.Frames)
	return f, err
}

func installWorkerSection(f *Frame, s workerSection) error {
	if len(s.Values) != len(s.FigProps) {
		return fmt.Errorf("invalid worker section %d", s.Num)
	}
	sec := f.Sects[s.Num]
	if sec == nil {
		sec = f.AddSect(s.Num)
	}
	sec.Type, sec.Original, sec.Color = s.Type, s.Original, s.Color
	sec.Exp, sec.Exq, sec.ElasticEnds = s.Exp, s.Exq, s.ElasticEnds
	sec.Lload, sec.Perpl, sec.Yield = s.Lload, s.Perpl, s.Yield
	for i, values := range s.Values {
		if i >= len(sec.Figs) {
			sec.Figs = append(sec.Figs, NewFig())
		}
		sec.Figs[i].Prop = f.Props[s.FigProps[i]]
		sec.Figs[i].Value = values
	}
	return nil
}

func installWorkerElement(f *Frame, data workerElem) error {
	e := f.Elems[data.Num]
	if e == nil {
		sect := f.Sects[data.Section]
		if sect == nil || len(data.Ends) != 2 {
			return fmt.Errorf("missing analysis element %d", data.Num)
		}
		n1, n2 := f.Nodes[data.Ends[0]], f.Nodes[data.Ends[1]]
		if n1 == nil || n2 == nil {
			return fmt.Errorf("missing nodes for element %d", data.Num)
		}
		e = f.AddLineElem(data.Num, []*Node{n1, n2}, sect, data.Type)
	}
	e.Cang, e.Prestress, e.Cmq, e.Wrect, e.Skip, e.Lock = data.Cang, data.Prestress, data.Cmq, data.Wrect, data.Skip, data.Lock
	if len(data.Stress) != 0 {
		maps.Copy(e.Stress, data.Stress)
	}
	if e.IsLineElem() {
		if len(e.Cmq) == 0 {
			e.Cmq = make([]float64, 12)
		}
		e.Bonds = make([]*Bond, 12)
		for i, b := range data.Bonds {
			if b == 1 {
				e.Bonds[i] = Pin
			} else if b != 0 {
				e.Bonds[i] = f.Bonds[b]
			}
		}
		if err := e.SetPrincipalAxis(); err != nil {
			return err
		}
	}
	return nil
}

func attachWorkerBraces(f *Frame, elems []workerElem) {
	for _, data := range elems {
		e, parent := f.Elems[data.Num], f.Elems[data.Parent]
		if e == nil || parent == nil || e.Parent != nil {
			continue
		}
		if len(parent.Children) != 2 {
			parent.Children = make([]*Elem, 2)
		}
		i := 1
		if data.Eldest {
			i = 0
		}
		parent.Children[i] = e
		e.Parent, e.Eldest = parent, data.Eldest
	}
}

// WorkerAnalysisCommand identifies computation routed off the browser UI.
func WorkerAnalysisCommand(name string) bool {
	switch name {
	case "analysis", "all", "extractarclm", "arclm101", "arclm301", "arclm401", "bclng001", "vibeig", "timehistory", "srcal", "srcalangle":
		return true
	}
	return false
}

type analysisWorkerWindow struct {
	ExModer
	f         *Frame
	cwd, last string
	report    func(string, int, int)
	writer    io.Writer
	done      chan struct{}
	failed    bool
}

func (w *analysisWorkerWindow) Frame() *Frame             { return w.f }
func (w *analysisWorkerWindow) Cwd() string               { return w.cwd }
func (w *analysisWorkerWindow) Home() string              { return "/workspace" }
func (w *analysisWorkerWindow) Recent() []string          { return nil }
func (w *analysisWorkerWindow) EPS() float64              { return 1e-3 }
func (w *analysisWorkerWindow) Yn(string, string) bool    { return false }
func (w *analysisWorkerWindow) Pivot() bool               { return false }
func (w *analysisWorkerWindow) Redraw()                   {}
func (w *analysisWorkerWindow) EnableLabel(string)        {}
func (w *analysisWorkerWindow) SetLabel(string, string)   {}
func (w *analysisWorkerWindow) SelectedNodes() []*Node    { return nil }
func (w *analysisWorkerWindow) SelectedElems() []*Elem    { return nil }
func (w *analysisWorkerWindow) ElemSelected() bool        { return false }
func (w *analysisWorkerWindow) Deselect()                 {}
func (w *analysisWorkerWindow) NodeSelected() bool        { return false }
func (w *analysisWorkerWindow) History(text string)       { fmt.Fprintln(w.writer, text) }
func (w *analysisWorkerWindow) HistoryWriter() io.Writer  { return w.writer }
func (w *analysisWorkerWindow) SetLastExCommand(s string) { w.last = s }
func (w *analysisWorkerWindow) LastExCommand() string     { return w.last }
func (w *analysisWorkerWindow) CurrentLap(message string, step, total int) {
	if message == "Failed" || message == "Error" {
		w.failed = true
	}
	w.report(message, step, total)
}

func workerAnalysisFinished(stw ExModer) {
	if w, ok := stw.(*analysisWorkerWindow); ok {
		close(w.done)
	}
}

// RunWorkerAnalysis runs one transaction. A failed job returns no model update.
func RunWorkerAnalysis(in io.Reader, out io.Writer, log io.Writer, progress func(string, int, int)) error {
	var m workerModel
	if err := gob.NewDecoder(in).Decode(&m); err != nil {
		return err
	}
	words := strings.Fields(m.Command)
	if len(words) == 0 {
		return fmt.Errorf("empty worker command")
	}
	name, _, usage, _ := ExModeComplete(words[0])
	if usage || strings.Contains(m.Command, "|") || !WorkerAnalysisCommand(name) {
		return fmt.Errorf("invalid worker command: %s", m.Command)
	}
	if m.Cwd != "" {
		if err := os.Chdir(m.Cwd); err != nil {
			return err
		}
	}
	f, err := m.frame()
	if err != nil {
		return err
	}
	w := &analysisWorkerWindow{f: f, cwd: m.Cwd, writer: log, report: progress, done: make(chan struct{})}
	progress("解析処理中", 0, 0)
	err = ExMode(w, m.Command)
	if _, started := err.(ArclmStart); started {
		<-w.done
	} else if err != nil {
		return err
	}
	if w.failed {
		return fmt.Errorf("%s: 解析に失敗しました。履歴を確認してください", name)
	}
	frames, err := workerFrames(f.Arclms)
	if err != nil {
		return err
	}
	res := workerResult{Period: f.Show.Period, Deform: name == "timehistory", Nodes: make(map[int]workerNode, len(f.Nodes)), Sections: workerSections(f),
		Frames: frames, ResultFiles: f.ResultFileName, DataFiles: f.DataFileName, Laps: f.Nlap}
	for id, n := range f.Nodes {
		res.Nodes[id] = workerNode{Disp: n.Disp, Reaction: n.Reaction}
	}
	for _, e := range f.Elems {
		res.Elems = append(res.Elems, workerElement(e))
	}
	return gob.NewEncoder(out).Encode(res)
}

// ApplyWorkerAnalysis updates the display results in place, retaining the
// user's viewport, selection and any previous result periods.
func ApplyWorkerAnalysis(in io.Reader, f *Frame) error {
	var res workerResult
	if err := gob.NewDecoder(in).Decode(&res); err != nil {
		return err
	}
	frames, err := restoreWorkerFrames(res.Frames)
	if err != nil {
		return err
	}
	for _, e := range res.Elems {
		if el := f.Elems[e.Num]; el == nil {
			if len(e.Ends) != 2 || f.Nodes[e.Ends[0]] == nil || f.Nodes[e.Ends[1]] == nil {
				return fmt.Errorf("worker element %d does not match the model", e.Num)
			}
		}
	}
	for _, s := range res.Sections {
		sec := f.Sects[s.Num]
		if sec == nil {
			if err := installWorkerSection(f, s); err != nil {
				return err
			}
		} else {
			sec.Type = s.Type
		}
	}
	for id, n := range res.Nodes {
		if dst := f.Nodes[id]; dst != nil {
			maps.Copy(dst.Disp, n.Disp)
			maps.Copy(dst.Reaction, n.Reaction)
		}
	}
	for _, data := range res.Elems {
		e := f.Elems[data.Num]
		if e == nil {
			if err := installWorkerElement(f, data); err != nil {
				return err
			}
			e = f.Elems[data.Num]
		}
		maps.Copy(e.Stress, data.Stress)
		maps.Copy(e.Phinge, data.Phinge)
		maps.Copy(e.Values, data.Values)
		if len(data.Rate) > 0 {
			if e.Rate == nil {
				e.Rate = make(map[string][]float64)
			}
			maps.Copy(e.Rate, data.Rate)
		}
		if data.MaxRate != nil {
			e.MaxRate = data.MaxRate
		}
		e.SrcalTex, e.Lock = data.SrcalTex, data.Lock
	}
	attachWorkerBraces(f, res.Elems)
	f.Arclms = frames
	maps.Copy(f.Nlap, res.Laps)
	maps.Copy(f.ResultFileName, res.ResultFiles)
	maps.Copy(f.DataFileName, res.DataFiles)
	if res.Deform {
		f.Show.PlotState |= PLOT_DEFORMED
	}
	if res.Period != "" {
		f.Show.Period = res.Period
	}
	return nil
}
