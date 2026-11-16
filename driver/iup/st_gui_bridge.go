package stgui

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
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
	timer  *iup.Handle
	cancel context.CancelFunc
}

func (stw *Window) guiConnectionCommand(arg string) error {
	if strings.EqualFold(strings.TrimSpace(arg), "off") {
		stw.stopGUIConnection()
		return nil
	}
	if strings.TrimSpace(arg) != "" {
		return fmt.Errorf("use :mcpconnect or :mcpconnect off")
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
	go st.RunGUIPipe(ctx, id, func(q st.GUIRequest) st.GUIResponse {
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
	}); err != nil {
		cancel()
		timer.Destroy()
		return err
	}
	enableTimerAction(timer)
	stw.guiConnection = &guiConnection{timer, cancel}
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
	if f != nil && (q.Action == "apply" || q.Action == "follow" || q.Action == "open_model" || q.Action == "show_result") {
		for _, analysis := range f.Arclms {
			if analysis != nil && analysis.Running() {
				return fail(fmt.Errorf("ST GUI analysis is running"))
			}
		}
	}
	switch q.Action {
	case "status":
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
		if f == nil || stw.IsChanged() || !strings.EqualFold(filepath.Clean(f.Path), filepath.Clean(q.BasePath)) || f.GUIModelDigest() != q.BaseDigest {
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
		if f == nil || stw.IsChanged() || !strings.EqualFold(filepath.Clean(f.Path), filepath.Clean(q.BasePath)) || f.GUIModelDigest() != q.BaseDigest {
			return fail(fmt.Errorf("GUI model differs from the proposal baseline; save or reopen the baseline model before applying"))
		}
		parsed := st.NewFrame()
		if err := parsed.ReadInpString(string(q.INP), f.Path, nil, 0, false); err != nil {
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
		stw.Deselect()
		stw.SetFrame(parsed)
		stw.LinkTextValue()
		st.Snapshot(stw)
		stw.Redraw()
		f = parsed
	default:
		return fail(fmt.Errorf("unknown GUI action"))
	}
	if f != nil {
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
