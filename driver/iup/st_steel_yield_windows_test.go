package stgui

import (
	"fmt"
	"strings"
	"time"

	st "github.com/yofu/st/stlib"
)

func exerciseMCPSteelYield(tool func(string, map[string]any, any) error) error {
	var original st.GUIResponse
	if err := tool("gui_status", map[string]any{}, &original); err != nil {
		return err
	}
	run := func(cmd string) (st.GUIResponse, error) {
		var v st.GUIResponse
		err := tool("gui_execute_commands", map[string]any{"commands": []string{cmd}, "wait_ms": 5000}, &v)
		for err == nil && v.CommandJob != nil && v.CommandJob.State == "running" {
			time.Sleep(25 * time.Millisecond)
			err = tool("gui_command_status", map[string]any{"job_id": v.CommandJob.ID}, &v)
		}
		if err == nil && (v.CommandJob == nil || v.CommandJob.State != "completed" || v.SyncError != "") {
			return v, fmt.Errorf("IUP %s failed: job=%+v sync=%s", cmd, v.CommandJob, v.SyncError)
		}
		return v, err
	}
	p, err := run(":steelyield -sect=501,502,503,504,511,512,513 -design=2 -factor=0.9")
	if err != nil {
		return err
	}
	history := p.CommandJob.History
	if original.Revision == "" || p.Revision != original.Revision || p.Changed || !strings.Contains(history, "SECT 501  使用部材") || !strings.Contains(history, "ELEM 2399") || !strings.Contains(history, "BTFAC") || !strings.Contains(history, "MYMIN") {
		return fmt.Errorf("incomplete or mutating preview (history %d bytes, revision changed=%t, dirty=%t)", len(history), p.Revision != original.Revision, p.Changed)
	}
	_, tail, ok := strings.Cut(history, "preview_id=")
	id, _, _ := strings.Cut(tail, "\n")
	if !ok || len(id) != 24 {
		return fmt.Errorf("missing review ID in IUP")
	}
	applied, err := run(":steelyield -apply=" + id)
	if err != nil {
		return err
	}
	if applied.Revision == original.Revision || !applied.Changed {
		return fmt.Errorf("IUP did not apply reviewed changes")
	}
	var v st.GUIResponse
	if err = tool("gui_undo", map[string]any{}, &v); err != nil {
		return err
	}
	if v.Revision != original.Revision {
		return fmt.Errorf("one GUI undo did not restore all sections")
	}
	if err = tool("gui_redo", map[string]any{}, &v); err != nil {
		return err
	}
	if v.Revision != applied.Revision {
		return fmt.Errorf("GUI redo did not restore reviewed input")
	}
	return nil
}
