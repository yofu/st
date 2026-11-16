package stgui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/yofu/go-iup/iup"
	st "github.com/yofu/st/stlib"
)

type liveAnalysisView struct {
	timer *iup.Handle
	dir   string
	frame *st.Frame
	step  int
}

func (stw *Window) liveAnalysisCommand(arg string) error {
	arg = strings.Trim(strings.TrimSpace(arg), "\"")
	if arg == "" || strings.EqualFold(arg, "off") {
		stw.stopLiveAnalysis()
		return nil
	}
	if stw.IsChanged() {
		return fmt.Errorf("save the current model before opening an MCP analysis preview")
	}
	dir, err := filepath.Abs(arg)
	if err != nil {
		return err
	}
	stw.stopLiveAnalysis()
	view := &liveAnalysisView{dir: dir, frame: stw.frame, timer: iup.Timer("TIME=200")}
	if err := view.timer.SetCallback(func(_ *iup.TimerAction) {
		if err := stw.refreshLiveAnalysis(view); err != nil {
			view.timer.SetAttribute("RUN", "NO")
			st.ErrorMessage(stw, err, st.ERROR)
		}
	}); err != nil {
		view.timer.Destroy()
		return err
	}
	enableTimerAction(view.timer)
	stw.liveAnalysis = view
	stw.CurrentLap("MCP: waiting for the edited model", 0, 0)
	view.timer.SetAttribute("RUN", "YES")
	return nil
}

func (stw *Window) stopLiveAnalysis() {
	if view := stw.liveAnalysis; view != nil {
		view.timer.SetAttribute("RUN", "NO")
		view.timer.Destroy()
		stw.liveAnalysis = nil
	}
}

func readLiveJSON(dir, name string, dst any) error {
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}

func (stw *Window) refreshLiveAnalysis(view *liveAnalysisView) (err error) {
	defer func() {
		if value := recover(); value != nil {
			err = fmt.Errorf("MCP preview: %v", value)
		}
	}()
	if stw.frame != view.frame || stw.IsChanged() {
		view.timer.SetAttribute("RUN", "NO")
		return fmt.Errorf("MCP preview stopped because the active model changed")
	}
	var status st.LiveAnalysisStatus
	statusErr := readLiveJSON(view.dir, st.LiveAnalysisStatusFile, &status)
	files, err := os.ReadDir(view.dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for i := len(files) - 1; i >= 0; i-- {
		name := files[i].Name()
		if !strings.HasPrefix(name, "step-") || !strings.HasSuffix(name, ".json") {
			continue
		}
		index, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(name, "step-"), ".json"))
		if err != nil || index <= view.step {
			continue
		}
		var step st.LiveAnalysisStep
		if err = readLiveJSON(view.dir, name, &step); err != nil {
			if statusErr == nil && !os.IsNotExist(err) {
				return err
			}
			continue
		}
		if step.Step != index {
			return fmt.Errorf("MCP preview step does not match file name")
		}
		frame := stw.frame
		if view.step == 0 {
			frame = st.NewFrame()
			if err = frame.ReadInp(filepath.Join(view.dir, st.LiveAnalysisModel), nil, 0, false); err != nil {
				return err
			}
			frame.AddAnalysisBraces()
		}
		if err = frame.ApplyLiveAnalysisStep(&step); err != nil {
			return err
		}
		if view.step == 0 {
			frame.Home = stw.Home()
			if stw.frame != nil && len(stw.frame.Nodes) > 0 {
				prev := stw.frame
				frame.View = prev.View.Copy()
				frame.Show = prev.Show.Copy()
				frame.Show.Frame = frame
				frame.Show.Dfact = prev.Show.Dfact
				for id := range frame.Sects {
					if _, ok := frame.Show.Sect[id]; !ok {
						frame.Show.Sect[id] = true
					}
				}
			}
			stw.Deselect()
			stw.SetFrame(frame)
			if view.frame == nil || len(view.frame.Nodes) == 0 {
				st.ShowCenter(stw)
			}
			view.frame = frame
			stw.Dlg.SetAttribute("TITLE", "ST - MCP analysis preview")
			st.DeformationOn(stw, false)
			hasStress := false
			for _, enabled := range frame.Show.Stress {
				if enabled != 0 {
					hasStress = true
					break
				}
			}
			if !hasStress {
				st.StressOn(stw, st.COLUMN, 4)
				st.StressOn(stw, st.GIRDER, 4)
				st.StressOn(stw, st.BRACE, 0)
				st.StressOn(stw, st.WBRACE, 0)
				st.StressOn(stw, st.SBRACE, 0)
			}
			stw.LinkTextValue()
		}
		st.SetPeriod(stw, st.LiveAnalysisPeriod)
		load := fmt.Sprintf("%g%s", step.Factor, step.Period)
		if step.Time != nil {
			load = fmt.Sprintf("t=%.6g s", *step.Time)
		} else if step.PreloadL {
			load = fmt.Sprintf("L%+g%s", step.Factor, step.Period)
		}
		stw.CurrentLap("MCP: "+load, step.Step, step.Total)
		view.step = step.Step
		stw.Redraw()
		break
	}
	if statusErr == nil && status.State != "" {
		view.timer.SetAttribute("RUN", "NO")
		message := "MCP preview: " + status.State
		if status.Error != "" {
			message += ": " + status.Error
		}
		stw.History(message)
		if view.step == 0 {
			stw.CurrentLap(message, 0, 0)
			stw.Redraw()
		}
	}
	return nil
}
