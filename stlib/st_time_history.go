package st

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/yofu/st/arclm"
)

const timeHistoryUsage = ":timehistory wave.csv -rayleigh=zeta1[;zeta2] {-modes=1;3} {-period=L} {-dt=seconds} {-output=prefix} {-nodes=n1,n2} {-save=interval} {-draw=10} {-eps=1e-8} {-wait} (explicit periods: -rayleigh=T1;T2;zeta1[;zeta2])"

func prepareTimeHistory(frame *Frame, input string, args map[string]string, cwd string) (*arclm.Frame, arclm.GroundMotion, arclm.TimeHistoryOptions, error) {
	o := arclm.TimeHistoryOptions{EquilibratePreload: true}
	if input == "" {
		return nil, nil, o, fmt.Errorf("%s", timeHistoryUsage)
	}
	per := "L"
	if p := args["PERIOD"]; p != "" {
		per = strings.ToUpper(p)
	}
	source := frame.Arclms[per]
	if source == nil {
		return nil, nil, o, fmt.Errorf("timehistory: static preload %s not found; run :analysis -period=%s first", per, per)
	}
	if old := frame.Arclms["TH"]; old != nil && old.Running() {
		return nil, nil, o, fmt.Errorf("time history is already running")
	}
	if source.Running() {
		return nil, nil, o, fmt.Errorf("preload analysis is running")
	}
	if source.TimeHistoryState() != nil {
		return nil, nil, o, fmt.Errorf("time history requires a static preload")
	}
	for _, option := range []struct {
		key string
		set func(float64)
	}{
		{"DT", func(v float64) { o.TimeStep = v }}, {"EPS", func(v float64) { o.RelativeTolerance = v }},
	} {
		if s, ok := args[option.key]; ok {
			v, err := strconv.ParseFloat(s, 64)
			if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
				return nil, nil, o, fmt.Errorf("timehistory: invalid %s", option.key)
			}
			option.set(v)
		}
	}
	parts := strings.Split(args["RAYLEIGH"], ";")
	if len(parts) < 1 || len(parts) > 4 {
		return nil, nil, o, fmt.Errorf("timehistory: invalid -rayleigh; use damping ratios or T1;T2;zeta1[;zeta2]")
	}
	values := make([]float64, len(parts))
	for i, s := range parts {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, o, fmt.Errorf("invalid Rayleigh value %q", s)
		}
		values[i] = v
	}
	if len(parts) <= 2 {
		o.AutoRayleigh = true
		o.Damping1, o.Damping2 = values[0], values[0]
		if len(parts) == 2 {
			o.Damping2 = values[1]
		}
		if o.Damping1 < 0 || o.Damping2 < 0 {
			return nil, nil, o, fmt.Errorf("Rayleigh damping ratios must be nonnegative")
		}
	} else {
		o.Period1, o.Period2, o.Damping1, o.Damping2 = values[0], values[1], values[2], values[2]
		if len(parts) == 4 {
			o.Damping2 = values[3]
		}
		if _, _, err := arclm.RayleighCoefficients(o.Period1, o.Period2, o.Damping1, o.Damping2); err != nil {
			return nil, nil, o, err
		}
	}
	if s, ok := args["MODES"]; ok {
		if !o.AutoRayleigh {
			return nil, nil, o, fmt.Errorf("-modes cannot be combined with explicit Rayleigh periods")
		}
		pair := strings.Split(s, ";")
		if len(pair) != 2 {
			return nil, nil, o, fmt.Errorf("use -modes=mode1;mode2")
		}
		for i, s := range pair {
			n, err := strconv.Atoi(s)
			if err != nil || n < 1 {
				return nil, nil, o, fmt.Errorf("invalid Rayleigh mode %q", s)
			}
			o.RayleighModes[i] = n
		}
		if o.RayleighModes[1] <= o.RayleighModes[0] {
			return nil, nil, o, fmt.Errorf("Rayleigh modes must be distinct and ascending")
		}
	}
	if s, ok := args["SAVE"]; ok {
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 {
			return nil, nil, o, fmt.Errorf("timehistory: SAVE must be a nonnegative integer")
		}
		o.SaveEvery = n
	}
	if s, ok := args["NODES"]; ok {
		seen := make(map[int]bool)
		for _, p := range strings.Split(s, ",") {
			n, err := strconv.Atoi(p)
			if err != nil || n <= 0 || seen[n] {
				return nil, nil, o, fmt.Errorf("timehistory: invalid output node %q", p)
			}
			seen[n] = true
			o.OutputNodes = append(o.OutputNodes, n)
		}
		for n := range seen {
			found := false
			for _, node := range source.Nodes {
				found = found || node.Num == n
			}
			if !found {
				return nil, nil, o, fmt.Errorf("timehistory: output node %d not found", n)
			}
		}
	}
	o.Output = strings.TrimSuffix(input, filepath.Ext(input)) + ".response"
	if s, ok := args["OUTPUT"]; ok {
		if s == "" {
			return nil, nil, o, fmt.Errorf("timehistory: output prefix is empty")
		}
		o.Output = s
	}
	if !filepath.IsAbs(o.Output) {
		o.Output = filepath.Join(cwd, o.Output)
	}
	for _, suffix := range []string{".steps.csv", ".nodes.csv", ".floors.csv", ".stories.csv", ".peaks.csv"} {
		if _, err := os.Stat(o.Output + suffix); err == nil {
			return nil, nil, o, fmt.Errorf("response already exists: %s; choose another -output", o.Output+suffix)
		} else if !os.IsNotExist(err) {
			return nil, nil, o, err
		}
	}
	file, err := os.Open(input)
	if err != nil {
		return nil, nil, o, err
	}
	motion, readErr := arclm.ReadGroundMotionCSV(file)
	closeErr := file.Close()
	if readErr != nil {
		return nil, nil, o, readErr
	}
	if closeErr != nil {
		return nil, nil, o, closeErr
	}
	if o.TimeStep > 0 && motion[len(motion)-1].Time/o.TimeStep > float64(int(^uint(0)>>1))-1 {
		return nil, nil, o, fmt.Errorf("timehistory: too many requested steps")
	}
	af, err := source.CloneForTimeHistory()
	return af, motion, o, err
}

// Drawing is independent of integration/output sampling. Zero draws only the
// final committed state; one restores the old every-step drawing behavior.
func timeHistoryDrawInterval(args map[string]string) (int, error) {
	n := 10
	if s, ok := args["DRAW"]; ok {
		value, err := strconv.Atoi(s)
		if err != nil || value < 0 {
			return 0, fmt.Errorf("timehistory: DRAW must be a nonnegative integer")
		}
		n = value
	}
	return n, nil
}

func runTimeHistoryCommand(stw ExModer, input string, args map[string]string) error {
	drawEvery, err := timeHistoryDrawInterval(args)
	if err != nil {
		return err
	}
	frame := stw.Frame()
	af, motion, o, err := prepareTimeHistory(frame, input, args, stw.Cwd())
	if err != nil {
		return err
	}
	var floors *TimeHistoryFloorOutput
	if frame.Ai != nil && frame.Ai.Nfloor >= 2 {
		floors, err = NewTimeHistoryFloorOutput(frame, af, o.Output)
		if err != nil {
			return err
		}
		o.OnStep = floors.Write
	} else {
		stw.History("timehistory: floor definitions unavailable; nodal responses only (define Ai.Nfloor and boundaries for floor/story exports)")
	}
	af.Output = stw.HistoryWriter()
	af.Lapch = make(chan int)
	frame.Arclms["TH"] = af
	nlap := len(motion) - 1
	if o.TimeStep > 0 {
		nlap = int(math.Ceil(motion[len(motion)-1].Time / o.TimeStep))
	}
	stw.CurrentLap("Time history...", 0, nlap)
	done := make(chan error, 1)
	go func() {
		err := af.TimeHistoryNewmark(context.Background(), motion, o)
		err = errors.Join(err, floors.Close())
		done <- err
	}()
	waitDone := make(chan struct{})
	go func() {
		defer close(waitDone)
		lastRead, lastDraw := -1, -1
		present := func(step int) {
			SetPeriod(stw, fmt.Sprintf("TH@%d", step))
			DeformationOn(stw, false)
			stw.CurrentLap("Time history...", step, nlap)
			stw.Redraw()
			lastDraw = step
		}
		finish := func(err error) {
			if s := af.TimeHistoryState(); s != nil {
				if lastRead != s.Step {
					frame.ReadArclmData(af, fmt.Sprintf("TH@%d", s.Step))
				}
				frame.Nlap["TH"] = s.Step
				if lastDraw != s.Step {
					present(s.Step)
				}
				if err == nil {
					stw.CurrentLap("Completed", s.Step, s.Step)
				}
			}
			if err != nil {
				stw.History(err.Error())
			}
		}
		for {
			select {
			case step := <-af.Lapch:
				// Retain every TH@step result for later navigation, even when
				// intermediate painting is skipped. Floor outputs are already
				// written from OnStep while the solver owns the committed state.
				frame.ReadArclmData(af, fmt.Sprintf("TH@%d", step))
				frame.Nlap["TH"], lastRead = step, step
				if drawEvery > 0 && step%drawEvery == 0 {
					present(step)
				}
				// Stop may cancel the solver while it awaits this acknowledgement.
				select {
				case af.Lapch <- 0:
				case err := <-done:
					finish(err)
					return
				}
			case err := <-done:
				finish(err)
				return
			}
		}
	}()
	if _, wait := args["WAIT"]; wait {
		<-waitDone
	}
	per := "L"
	if p := args["PERIOD"]; p != "" {
		per = strings.ToUpper(p)
	}
	return ArclmStart(fmt.Sprintf("TIME HISTORY: preload=%s wave=%s output=%s draw_every=%d", per, input, o.Output, drawEvery))
}
