package stgui

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"time"

	st "github.com/yofu/st/stlib"
)

type guiCommandJob struct {
	nextPrompt                uint64
	answer                    *st.GUICommandInput
	status                    st.GUICommandStatus
	commands                  []string
	waiting, finished, failed bool
	executing                 bool
	cancelRequested           bool
	stopAnalysis              func() error
	beforeAnalysis            string
	reply                     st.GUIResponse
	modelRevision             string
	knownPath, knownRevision  string
}

type guiCommandRunner struct {
	mu              sync.Mutex
	next            uint64
	completion      uint64
	jobs            map[uint64]*guiCommandJob
	active, control *guiCommandJob
}

func (r *guiCommandRunner) response(id, knownCompletion uint64, compact bool) st.GUIResponse {
	r.mu.Lock()
	defer r.mu.Unlock()
	j := r.jobs[id]
	if j == nil {
		return st.GUIResponse{Error: fmt.Sprintf("unknown GUI command job %d", id)}
	}
	reply := j.reply
	if j.status.Completion != 0 && j.status.Completion <= knownCompletion {
		reply.Snapshot = nil
	}
	status := j.status
	status.Timing = slices.Clone(status.Timing)
	if compact && status.Completion == 0 {
		status.History = ""
	}
	reply.CommandJob = &status
	return reply
}

// Cancellation targets just this job, even while the GUI thread is in a native dialog.
func (r *guiCommandRunner) cancel(id uint64) st.GUIResponse {
	r.mu.Lock()
	j := r.jobs[id]
	if j != nil && j.status.Completion == 0 && j.status.Completed < len(j.commands) {
		j.cancelRequested = true
		if j.stopAnalysis != nil && !j.finished {
			_ = j.stopAnalysis()
		}
	}
	r.mu.Unlock()
	return r.response(id, 0, true)
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

func (stw *Window) startGUICommands(q st.GUIRequest, checked string) st.GUIResponse {
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
		if command == "" {
			return fail(fmt.Errorf("empty command %d", i+1))
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
	j := &guiCommandJob{commands: commands, status: st.GUICommandStatus{ID: r.next, State: "running", Total: len(commands), AcceptedAt: time.Now().UTC()}}
	if stw.Frame() != nil {
		j.modelRevision = checked
		if checked == "" {
			j.modelRevision = stw.Frame().GUIRevision()
		}
		j.knownRevision, j.knownPath = q.KnownRevision, q.KnownPath
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
	// A native dialog or -wait command may dispatch this timer recursively.
	// Retry early stop requests without starting the same command again.
	if j.executing || (j.waiting && !j.finished) {
		if j.cancelRequested && !j.finished && j.stopAnalysis != nil {
			_ = j.stopAnalysis()
		}
		r.mu.Unlock()
		return
	}
	if j.waiting {
		j.waiting = false
		j.stopAnalysis = nil
		if !j.failed && !j.cancelRequested {
			j.status.Completed++
		}
		state := "completed"
		if j.failed {
			state = "failed"
		}
		if j.cancelRequested {
			state = "cancelled"
		}
		j.status.Timing[len(j.status.Timing)-1].State = state
		j.status.Timing[len(j.status.Timing)-1].FinishedAt = time.Now().UTC()
		before := j.beforeAnalysis
		r.mu.Unlock()
		stw.captureCommandEdit(before)
		return
	}
	if j.cancelRequested || j.failed || j.status.Completed == len(j.commands) {
		r.mu.Unlock()
		stw.finishGUICommands(r, j)
		return
	}
	command := j.commands[j.status.Completed]
	j.status.Timing = append(j.status.Timing, st.GUICommandTiming{Command: command, StartedAt: time.Now().UTC()})
	j.status.Command = command
	j.finished = false
	j.executing = true
	r.mu.Unlock()
	w := &guiCommandWindow{Window: stw, runner: r, job: j, index: j.status.Completed}
	f := stw.Frame()
	before := ""
	if f != nil {
		before = f.GUIRevision()
		stw.CaptureUndoBaselineRevision(f, before)
	}
	err := w.run(command)
	_, started := err.(st.ArclmStart)
	if !started {
		stw.captureCommandEdit(before)
		stw.LinkTextValue()
		w.Redraw()
	}
	r.mu.Lock()
	if !started {
		state := "completed"
		if err != nil {
			state = "failed"
		}
		if j.cancelRequested {
			state = "cancelled"
		}
		j.status.Timing[len(j.status.Timing)-1].State = state
		j.status.Timing[len(j.status.Timing)-1].FinishedAt = time.Now().UTC()
	}
	j.executing = false
	if started {
		j.beforeAnalysis = before
		j.waiting = true
	} else if err != nil {
		j.failed = true
		j.status.Error = err.Error()
	} else {
		j.status.Completed++
	}
	r.mu.Unlock()
}

func (stw *Window) captureCommandEdit(before string) {
	if current := stw.Frame(); current != nil {
		if after := current.GUIRevision(); after != before {
			stw.CaptureUndoBaselineRevision(current, after)
			stw.Changed(true)
		}
	}
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
	reply := stw.handleGUIRequest(st.GUIRequest{Action: "status", OmitRevision: true})
	if f != nil {
		if j.knownRevision != "" && strings.EqualFold(f.Path, j.knownPath) {
			reply.Revision = f.GUIRevision()
		}
		if reply.Revision == "" || reply.Revision != j.knownRevision {
			reply.Snapshot, reply.Revision = f.GUIInputSnapshot()
			reply.DeltaSupported = true
		} else {
			reply.InputView = f.View.Copy()
		}
		if reply.Revision != j.modelRevision {
			reply.Changed = true
			stw.Changed(true)
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	j.reply = reply
	r.completion++
	j.status.Completion = r.completion
	j.status.FinishedAt = time.Now().UTC()
	j.status.State = "completed"
	if j.cancelRequested {
		j.status.State = "cancelled"
	} else if j.failed {
		j.status.State = "failed"
	}
}

type guiCommandWindow struct {
	*Window
	runner *guiCommandRunner
	job    *guiCommandJob
	index  int
}

func (w *guiCommandWindow) Redraw() {
	start := time.Now()
	w.Window.Redraw()
	elapsed := time.Since(start)
	w.runner.mu.Lock()
	defer w.runner.mu.Unlock()
	timing := &w.job.status.Timing[w.index]
	timing.Redraws++
	timing.RedrawMS += float64(elapsed) / float64(time.Millisecond)
}

func (w *guiCommandWindow) UseCurrentModelInputs() bool { return true }
func (w *guiCommandWindow) History(text string) {
	w.Window.appendHistory(text)
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
func (w *guiCommandWindow) BindAnalysisStop(stop func() error) {
	w.runner.mu.Lock()
	defer w.runner.mu.Unlock()
	w.job.stopAnalysis = stop
}

func (w *guiCommandWindow) FileWritten(path string) {
	w.runner.mu.Lock()
	defer w.runner.mu.Unlock()
	t := &w.job.status.Timing[w.index]
	if !slices.Contains(t.Files, path) {
		t.Files = append(t.Files, path)
	}
}
func (w *guiCommandWindow) AnalysisOutput(result st.GUIAnalysisOutput) {
	w.FileWritten(result.Path)
	w.runner.mu.Lock()
	defer w.runner.mu.Unlock()
	w.job.status.Timing[w.index].Analysis = &result
}

func (w *guiCommandWindow) AnalysisFinished() {
	w.runner.mu.Lock()
	defer w.runner.mu.Unlock()
	w.job.finished = true
}
func (w *guiCommandWindow) run(command string) (err error) {
	prev := w.commandContext
	w.commandContext = w
	defer func() {
		w.commandContext = prev
		if problem := recover(); problem != nil {
			if w.canvasInput {
				w.EscapeCB()
			}
			err = fmt.Errorf("%s: %v", command, problem)
		}
	}()
	err = w.execute(command)
	if err == nil && w.canvasInput {
		err = w.finishInteraction()
	}
	if err == nil {
		w.runner.mu.Lock()
		if w.job.failed {
			err = fmt.Errorf("%s", w.job.status.Error)
		}
		w.runner.mu.Unlock()
	}
	return err
}

func (w *guiCommandWindow) execute(command string) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("%s: %v", command, p)
		}
	}()
	w.History("MCP > " + command)
	if !strings.HasPrefix(command, ":") && !strings.HasPrefix(command, "'") {
		com := w.nativeCommand(command)
		if com == nil && w.Frame() != nil && w.axisRangeCommand(strings.ToUpper(command)) {
			return nil
		}
		if com == nil {
			return fmt.Errorf("unknown IUP command %q", command)
		}
		if expansion := pgpExpansions[com]; expansion != nil {
			line := expansion()
			if strings.Contains(line, "_") {
				in, e := w.prompt(line, "text")
				if e != nil {
					return e
				}
				line = strings.Replace(line, "_", in.Text, 1)
			}
			return w.execute(line)
		}
		if w.Frame() == nil && com != Commands["OPEN"] {
			return fmt.Errorf("an open model is required for %s", command)
		}
		w.execCommand(com)
		return nil
	}
	keyword, arg, _ := strings.Cut(command, " ")
	switch strings.ToLower(keyword) {
	case ":mcpconnect":
		return w.Window.guiConnectionCommand(arg)
	case ":mcplive":
		return w.Window.liveAnalysisCommand(arg)
	}
	if strings.HasPrefix(command, "'") {
		err = st.Fig2Mode(w, command)
	} else {
		err = st.ExMode(w, command)
	}
	if err != nil {
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
