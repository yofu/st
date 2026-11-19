package stgui

import (
	"fmt"
	"io"
	"strings"
	"sync"

	st "github.com/yofu/st/stlib"
)

type guiCommandJob struct {
	status                    st.GUICommandStatus
	commands                  []string
	waiting, finished, failed bool
	executing                 bool
	reply                     st.GUIResponse
	modelRevision             string
}

type guiCommandRunner struct {
	mu              sync.Mutex
	next            uint64
	completion      uint64
	jobs            map[uint64]*guiCommandJob
	active, control *guiCommandJob
}

func (r *guiCommandRunner) response(id uint64) st.GUIResponse {
	r.mu.Lock()
	defer r.mu.Unlock()
	j := r.jobs[id]
	if j == nil {
		return st.GUIResponse{Error: fmt.Sprintf("unknown GUI command job %d", id)}
	}
	reply := j.reply
	status := j.status
	reply.CommandJob = &status
	return reply
}

func (stw *Window) guiCommandsBusy() bool {
	if stw.guiConnection == nil {
		return false
	}
	r := stw.guiConnection.commands
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.active != nil
}

func stopCommand(commands []string) bool {
	if len(commands) != 1 {
		return false
	}
	words := strings.Fields(commands[0])
	if len(words) == 0 || !strings.HasPrefix(words[0], ":") {
		return false
	}
	name, _, _, _ := st.ExModeComplete(words[0])
	return name == "stop"
}

func (stw *Window) startGUICommands(q st.GUIRequest) st.GUIResponse {
	fail := func(err error) st.GUIResponse { return st.GUIResponse{Error: err.Error()} }
	if stw.guiConnection == nil {
		return fail(fmt.Errorf("GUI connection is not active"))
	}
	if len(q.Commands) == 0 || len(q.Commands) > 100 {
		return fail(fmt.Errorf("provide 1 to 100 ex or Fig2 commands"))
	}
	commands := make([]string, len(q.Commands))
	for i, command := range q.Commands {
		command = strings.TrimSpace(command)
		if len(command) < 2 || (command[0] != ':' && command[0] != '\'') {
			return fail(fmt.Errorf("command %d must start with a colon or apostrophe", i+1))
		}
		commands[i] = command
	}
	r := stw.guiConnection.commands
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.active != nil && (!stopCommand(commands) || r.control != nil) {
		return fail(fmt.Errorf("GUI command job %d is running; use gui_command_status or :stop period", r.active.status.ID))
	}
	r.next++
	j := &guiCommandJob{commands: commands, status: st.GUICommandStatus{ID: r.next, State: "running", Total: len(commands)}}
	if stw.Frame() != nil {
		j.modelRevision = stw.Frame().GUIRevision()
	}
	r.jobs[j.status.ID] = j
	// Retain recent job outcomes so polling an earlier ID remains meaningful.
	if r.next > 32 {
		if old := r.jobs[r.next-32]; old != r.active {
			delete(r.jobs, r.next-32)
		}
	}
	if r.active == nil {
		r.active = j
	} else {
		r.control = j
	}
	status := j.status
	return st.GUIResponse{CommandJob: &status}
}

// Called by the IUP timer. Native commands and dialogs stay on the GUI thread.
func (stw *Window) advanceGUICommands() {
	if stw.guiConnection == nil {
		return
	}
	r := stw.guiConnection.commands
	r.mu.Lock()
	j := r.active
	if r.control != nil {
		j = r.control
	}
	if j == nil {
		r.mu.Unlock()
		return
	}
	// Native modal dialogs may pump IUP timer events recursively.
	if j.executing {
		r.mu.Unlock()
		return
	}
	if j.waiting && !j.finished {
		r.mu.Unlock()
		return
	}
	if j.failed || j.status.Completed == len(j.commands) {
		r.mu.Unlock()
		stw.finishGUICommands(r, j)
		return
	}
	if j.waiting {
		j.waiting = false
		j.status.Completed++
		r.mu.Unlock()
		return
	}
	command := j.commands[j.status.Completed]
	j.status.Command = command
	j.finished = false
	j.executing = true
	r.mu.Unlock()
	w := &guiCommandWindow{Window: stw, runner: r, job: j}
	f := stw.Frame()
	before := ""
	if f != nil {
		before = f.GUIRevision()
		stw.CaptureUndoBaseline(f)
	}
	err := w.execute(command)
	if current := stw.Frame(); current != nil && current.GUIRevision() != before {
		stw.CaptureUndoBaseline(current)
		stw.Changed(true)
	}
	r.mu.Lock()
	j.executing = false
	if _, started := err.(st.ArclmStart); started {
		j.waiting = true
	} else if err != nil {
		j.failed = true
		j.status.Error = err.Error()
	} else {
		j.status.Completed++
	}
	r.mu.Unlock()
	stw.LinkTextValue()
	stw.Redraw()
}

func (stw *Window) finishGUICommands(r *guiCommandRunner, j *guiCommandJob) {
	r.mu.Lock()
	if r.active == j {
		r.active = nil
	}
	if r.control == j {
		r.control = nil
	}
	r.mu.Unlock()
	f := stw.Frame()
	if f != nil && f.GUIRevision() != j.modelRevision {
		stw.Changed(true)
	}
	reply := stw.handleGUIRequest(st.GUIRequest{Action: "status"})
	if f != nil {
		reply.Snapshot = &st.GUIModelSnapshot{INP: f.GUIInpBytes(), LST: f.GUILstBytes(), LSTPath: f.LstFileName}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	j.reply = reply
	r.completion++
	j.status.Completion = r.completion
	j.status.State = "completed"
	if j.failed {
		j.status.State = "failed"
	}
}

type guiCommandWindow struct {
	*Window
	runner *guiCommandRunner
	job    *guiCommandJob
}

func (w *guiCommandWindow) UseCurrentModelInputs() bool { return true }
func (w *guiCommandWindow) History(text string) {
	w.Window.History(text)
	_, _ = w.Write([]byte(text + "\n"))
}
func (w *guiCommandWindow) HistoryWriter() io.Writer { return w }
func (w *guiCommandWindow) Write(data []byte) (int, error) {
	w.runner.mu.Lock()
	defer w.runner.mu.Unlock()
	history := w.job.status.History + string(data)
	if len(history) > 65536 {
		history = history[len(history)-65536:]
	}
	w.job.status.History = history
	return len(data), nil
}
func (w *guiCommandWindow) CurrentLap(message string, step, total int) {
	w.Window.CurrentLap(message, step, total)
	w.runner.mu.Lock()
	defer w.runner.mu.Unlock()
	w.job.status.Progress, w.job.status.Step, w.job.status.Steps = message, step, total
	if message == "Failed" || message == "Error" {
		w.job.failed = true
		w.job.status.Error = "analysis failed; inspect command history"
	}
}
func (w *guiCommandWindow) AnalysisFinished() {
	w.runner.mu.Lock()
	defer w.runner.mu.Unlock()
	w.job.finished = true
}
func (w *guiCommandWindow) execute(command string) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("%s: %v", command, p)
		}
	}()
	w.History("MCP > " + command)
	if strings.HasPrefix(command, "'") {
		return st.Fig2Mode(w, command)
	}
	keyword, arg, _ := strings.Cut(command, " ")
	switch strings.ToLower(keyword) {
	case ":mcpconnect":
		return w.Window.guiConnectionCommand(arg)
	case ":mcplive":
		return w.Window.liveAnalysisCommand(arg)
	}
	if err = st.ExMode(w, command); err != nil {
		if _, started := err.(st.ArclmStart); started {
			return err
		}
		if message, ok := err.(st.Messager); ok {
			w.History(message.Message())
			return nil
		}
	}
	return err
}
