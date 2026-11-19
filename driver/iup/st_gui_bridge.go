package stgui

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"

	"github.com/yofu/go-iup/iup"
	st "github.com/yofu/st/stlib"
)

type guiCall struct {
	ctx      context.Context
	request  st.GUIRequest
	response chan st.GUIResponse
}

type guiConnection struct {
	timer    *iup.Handle
	cancel   context.CancelFunc
	commands *guiCommandRunner
}

func (stw *Window) guiConnectionCommand(arg string) error {
	if strings.EqualFold(strings.TrimSpace(arg), "off") {
		stw.stopGUIConnection()
		return nil
	}
	if strings.TrimSpace(arg) != "" {
		return fmt.Errorf("use :mcpconnect or :mcpconnect off")
	}
	if err := stw.ensureMCPHTTP(); err != nil {
		return err
	}
	stw.stopGUIConnection()
	var key [16]byte
	if _, err := rand.Read(key[:]); err != nil {
		return err
	}
	id := hex.EncodeToString(key[:])
	ctx, cancel := context.WithCancel(context.Background())
	calls := make(chan guiCall)
	ready := make(chan error, 1)
	commands := &guiCommandRunner{jobs: make(map[uint64]*guiCommandJob)}
	go st.RunGUIPipe(ctx, id, func(q st.GUIRequest) st.GUIResponse {
		if q.Action == "command_status" {
			return commands.response(q.CommandJobID)
		}
		deadline, stop := context.WithTimeout(ctx, 15*time.Second)
		defer stop()
		call := guiCall{deadline, q, make(chan st.GUIResponse, 1)}
		select {
		case <-deadline.Done():
			return st.GUIResponse{Error: deadline.Err().Error()}
		case calls <- call:
		}
		select {
		case <-deadline.Done():
			return st.GUIResponse{Error: deadline.Err().Error()}
		case response := <-call.response:
			return response
		}
	}, ready)
	select {
	case err := <-ready:
		if err != nil {
			cancel()
			return err
		}
	case <-time.After(3 * time.Second):
		cancel()
		return fmt.Errorf("could not start ST GUI pipe")
	}
	timer := iup.Timer("TIME=75")
	if err := timer.SetCallback(func(*iup.TimerAction) {
		select {
		case call := <-calls:
			if call.ctx.Err() == nil {
				call.response <- stw.handleGUIRequest(call.request)
			}
		default:
		}
		stw.advanceGUICommands()
	}); err != nil {
		cancel()
		timer.Destroy()
		return err
	}
	enableTimerAction(timer)
	stw.guiConnection = &guiConnection{timer: timer, cancel: cancel, commands: commands}
	timer.SetAttribute("RUN", "YES")
	stw.History("MCP GUI connection ID: " + id)
	return nil
}

func (stw *Window) stopGUIConnection() {
	if conn := stw.guiConnection; conn != nil {
		conn.cancel()
		conn.timer.SetAttribute("RUN", "NO")
		conn.timer.Destroy()
		stw.guiConnection = nil
	}
}

func (stw *Window) handleGUIRequest(q st.GUIRequest) (result st.GUIResponse) {
	defer func() {
		if problem := recover(); problem != nil {
			result.Error = fmt.Sprintf("%v", problem)
		}
	}()
	f := stw.Frame()
	fail := func(err error) st.GUIResponse { return st.GUIResponse{Error: err.Error()} }
	if f != nil && q.Action != "status" && q.Action != "commands" {
		for _, analysis := range f.Arclms {
			if analysis != nil && analysis.Running() {
				return fail(fmt.Errorf("ST GUI analysis is running"))
			}
		}
	}
	if stw.guiCommandsBusy() && q.Action != "status" && q.Action != "commands" {
		return fail(fmt.Errorf("a GUI command job is running; check gui_command_status before changing the model"))
	}
	if q.Revision != "" && (f == nil || f.GUIRevision() != q.Revision) {
		return fail(fmt.Errorf("GUI model changed since the snapshot; call gui_get_model and review the edit again"))
	}
	switch q.Action {
	case "status":
	case "commands":
		return stw.startGUICommands(q)
	case "snapshot":
		if f == nil {
			return fail(fmt.Errorf("no model is open"))
		}
	case "display":
		if f == nil || q.Display == nil {
			return fail(fmt.Errorf("an open model and display settings are required"))
		}
		if err := stw.applyGUIDisplay(*q.Display); err != nil {
			return fail(err)
		}
		stw.LinkTextValue()
		stw.Redraw()
	case "undo", "redo":
		if f == nil {
			return fail(fmt.Errorf("no model is open"))
		}
		if !stw.UndoEnabled() {
			return fail(fmt.Errorf("GUI undo is disabled"))
		}
		var restored *st.Frame
		var err error
		if q.Action == "undo" {
			restored, err = stw.Undo()
		} else {
			restored, err = stw.Redo()
		}
		if err != nil {
			return fail(err)
		}
		restored.View = f.View.Copy()
		restored.Show = f.Show.Copy()
		restored.Show.Frame = restored
		stw.stopLiveAnalysis()
		stw.replaceGUIFrame(restored)
		stw.Changed(true)
		f = restored
	case "open_model":
		if stw.IsChanged() {
			return fail(fmt.Errorf("save the GUI model before opening another model"))
		}
		if !filepath.IsAbs(q.BasePath) || !strings.EqualFold(filepath.Ext(q.BasePath), ".inp") {
			return fail(fmt.Errorf("expected an absolute .inp path"))
		}

		parsed := st.NewFrame()
		if err := parsed.ReadInpSnapshot(q.INP, q.BasePath); err != nil {
			return fail(err)
		}
		if len(parsed.Nodes) == 0 {
			return fail(fmt.Errorf("INP contains no nodes"))
		}
		if len(q.LST) > 0 {
			if err := parsed.ReadLstString(string(q.LST), q.LSTPath); err != nil {
				return fail(err)
			}
		}
		if parsed.GUIModelDigest() != q.BaseDigest {
			return fail(fmt.Errorf("GUI model differs from the MCP source snapshot"))
		}
		stw.stopLiveAnalysis()
		stw.Deselect()
		st.OpenPreparedFrame(stw, parsed, q.BasePath, false)
		stw.Dlg.SetAttribute("TITLE", filepath.Base(q.BasePath))
		stw.LinkTextValue()
		stw.Redraw()
		f = parsed
	case "show_result":
		if f == nil || !strings.EqualFold(filepath.Clean(f.Path), filepath.Clean(q.BasePath)) || f.GUIModelDigest() != q.BaseDigest {
			return fail(fmt.Errorf("GUI model differs from the analyzed model; open the same saved INP before displaying the result"))
		}
		var axes [3]bool
		for _, axis := range q.DisplacementAxes {
			index := strings.Index("XYZ", strings.ToUpper(axis))
			if len(axis) != 1 || index < 0 || axes[index] {
				return fail(fmt.Errorf("displacement_axes must contain distinct X, Y or Z values"))
			}
			axes[index] = true
		}
		if err := f.LoadStaticResult(q.ResultPath, q.ResultData); err != nil {
			return fail(err)
		}
		stw.stopLiveAnalysis()
		st.SetPeriod(stw, st.StaticResultPeriod(q.ResultPath))
		st.DeformationOn(stw, false)
		for axis, show := range axes {
			if show {
				st.DispOn(stw, axis)
			} else {
				st.DispOff(stw, axis)
			}
		}
		stw.History("MCP result: " + q.ResultPath)
		stw.Redraw()
	case "select":
		if f == nil {
			return fail(fmt.Errorf("no model is open"))
		}
		var nodes []*st.Node
		var elems []*st.Elem
		if q.NodeIDs != nil {
			for _, id := range *q.NodeIDs {
				n := f.Nodes[id]
				if n == nil {
					return fail(fmt.Errorf("unknown GUI node %d", id))
				}
				nodes = append(nodes, n)
			}
		}
		if q.ElementIDs != nil {
			for _, id := range *q.ElementIDs {
				el := f.Elems[id]
				if el == nil {
					return fail(fmt.Errorf("unknown GUI element %d", id))
				}
				elems = append(elems, el)
			}
		}
		if q.NodeIDs != nil {
			stw.SelectNode(nodes)
		}
		if q.ElementIDs != nil {
			stw.SelectElem(elems)
		}
		stw.Redraw()
	case "follow":
		if err := stw.liveAnalysisCommand(q.PreviewDir); err != nil {
			return fail(err)
		}
		stw.Redraw()
	case "apply":
		if f == nil || !strings.EqualFold(filepath.Clean(f.Path), filepath.Clean(q.BasePath)) || f.GUIModelDigest() != q.BaseDigest {
			return fail(fmt.Errorf("GUI model differs from the proposal baseline; call gui_get_model and review the edit again"))
		}
		parsed := st.NewFrame()
		if err := parsed.ReadInpSnapshot(q.INP, f.Path); err != nil {
			return fail(err)
		}
		if len(q.LST) > 0 {
			if err := parsed.ReadLstString(string(q.LST), f.LstFileName); err != nil {
				return fail(err)
			}
		}
		parsed.Home = f.Home
		parsed.View = f.View.Copy()
		parsed.Show = f.Show.Copy()
		parsed.Show.Frame = parsed
		parsed.Show.Dfact = f.Show.Dfact
		for id := range parsed.Sects {
			if _, ok := parsed.Show.Sect[id]; !ok {
				parsed.Show.Sect[id] = true
			}
		}
		stw.stopLiveAnalysis()
		// Capture the actual baseline, including any unsnapshotted GUI edits.
		stw.CaptureUndoBaseline(f)
		stw.replaceGUIFrame(parsed)
		st.Snapshot(stw)
		f = parsed
	default:
		return fail(fmt.Errorf("unknown GUI action"))
	}
	if f != nil {
		result.SrcanRate = f.Show.SrcanRate
		for _, e := range f.Elems {
			if len(e.MaxRate) > 0 {
				result.RatedElements++
			}
		}
		result.Revision = f.GUIRevision()
		if q.Action == "snapshot" || q.Action == "apply" || q.Action == "undo" || q.Action == "redo" {
			result.Snapshot = &st.GUIModelSnapshot{INP: f.GUIInpBytes(), LST: f.GUILstBytes(), LSTPath: f.LstFileName}
		}
		result.Path = f.Path
		result.Period = f.Show.Period
		result.ResultPath = f.ResultFileName[result.Period]
		result.Deformation = f.Show.PlotState&st.PLOT_DEFORMED != 0
		result.DisplacementAxes = make([]string, 0, 3)
		for i, bit := range []uint{st.NC_DX, st.NC_DY, st.NC_DZ} {
			if f.Show.NodeCaption&bit != 0 {
				result.DisplacementAxes = append(result.DisplacementAxes, string("XYZ"[i]))
			}
		}
		stress := make([]st.GUIStress, 0)
		for etype := st.COLUMN; etype <= st.SLAB; etype++ {
			for component, name := range st.StressName {
				if f.Show.Stress[etype]&(1<<uint(component)) != 0 {
					stress = append(stress, st.GUIStress{Type: st.ETYPES[etype], Component: strings.ToUpper(name)})
				}
			}
		}
		scale := f.Show.Dfact
		result.Display = &st.GUIDisplay{Period: result.Period, Deformation: &result.Deformation, DeformationScale: &scale,
			DisplacementAxes: &result.DisplacementAxes, Stress: &stress,
			Angles: append([]float64(nil), f.View.Angle...), Focus: append([]float64(nil), f.View.Focus...)}
	}
	result.Changed = stw.IsChanged()
	result.Nodes = make([]int, 0)
	result.Elements = make([]int, 0)
	for _, n := range stw.SelectedNodes() {
		if n != nil {
			result.Nodes = append(result.Nodes, n.Num)
		}
	}
	for _, el := range stw.SelectedElems() {
		if el != nil {
			result.Elements = append(result.Elements, el.Num)
		}
	}
	return result
}

// Remap selections to the replacement frame; stale object pointers must never
// survive a transaction or an undo.
func (stw *Window) replaceGUIFrame(f *st.Frame) {
	var nodes []*st.Node
	var elems []*st.Elem
	for _, n := range stw.SelectedNodes() {
		if n != nil && f.Nodes[n.Num] != nil {
			nodes = append(nodes, f.Nodes[n.Num])
		}
	}
	for _, e := range stw.SelectedElems() {
		if e != nil && f.Elems[e.Num] != nil {
			elems = append(elems, f.Elems[e.Num])
		}
	}
	stw.Deselect()
	stw.SetFrame(f)
	stw.SelectNode(nodes)
	stw.SelectElem(elems)
	stw.LinkTextValue()
	stw.Redraw()
}

func (stw *Window) applyGUIDisplay(d st.GUIDisplay) error {
	f := stw.Frame()
	for _, values := range [][]float64{d.Angles, d.Focus} {
		for _, v := range values {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return fmt.Errorf("view values must be finite")
			}
		}
	}
	if d.Angles != nil && len(d.Angles) != 2 {
		return fmt.Errorf("angles requires two values")
	}
	if d.Focus != nil && len(d.Focus) != 3 {
		return fmt.Errorf("focus requires three values")
	}
	if d.DeformationScale != nil && (math.IsNaN(*d.DeformationScale) || math.IsInf(*d.DeformationScale, 0) || *d.DeformationScale <= 0) {
		return fmt.Errorf("deformation_scale must be positive and finite")
	}
	var axes [3]bool
	if d.DisplacementAxes != nil {
		for _, axis := range *d.DisplacementAxes {
			i := strings.Index("XYZ", strings.ToUpper(axis))
			if len(axis) != 1 || i < 0 || axes[i] {
				return fmt.Errorf("displacement_axes requires distinct X, Y or Z values")
			}
			axes[i] = true
		}
	}
	stress := make(map[int]uint)
	if d.Stress != nil {
		for _, item := range *d.Stress {
			etype, component := -1, -1
			if strings.EqualFold(item.Component, "N") {
				item.Component = "Nz"
			}
			for i, name := range st.ETYPES {
				if strings.EqualFold(name, item.Type) {
					etype = i
				}
			}
			for i, name := range st.StressName {
				if strings.EqualFold(name, item.Component) {
					component = i
				}
			}
			if etype < st.COLUMN || etype > st.SLAB || component < 0 {
				return fmt.Errorf("invalid stress type/component: %s %s", item.Type, item.Component)
			}
			stress[etype] |= 1 << uint(component)
		}
	}
	if d.Period != "" {
		st.SetPeriod(stw, strings.ToUpper(d.Period))
	}
	if d.Deformation != nil {
		if *d.Deformation {
			st.DeformationOn(stw, false)
		} else {
			st.DeformationOff(stw, true)
		}
	}
	if d.DeformationScale != nil {
		f.Show.Dfact = *d.DeformationScale
	}
	if d.DisplacementAxes != nil {
		for i, on := range axes {
			if on {
				st.DispOn(stw, i)
			} else {
				st.DispOff(stw, i)
			}
		}
	}
	if d.Stress != nil {
		for etype := 0; etype <= st.SLAB; etype++ {
			for component := range st.StressName {
				st.StressOff(stw, etype, uint(component))
				if stress[etype]&(1<<uint(component)) != 0 {
					st.StressOn(stw, etype, uint(component))
				}
			}
		}
	}
	if d.Angles != nil {
		copy(f.View.Angle, d.Angles)
	}
	if d.Focus != nil {
		copy(f.View.Focus, d.Focus)
	}
	return nil
}
