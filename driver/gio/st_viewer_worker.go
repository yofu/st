package stgio

import (
	"bytes"
	"fmt"

	st "github.com/yofu/st/stlib"
)

// SetViewerAnalysisWorker connects the browser's dedicated analysis runtime.
func (w *Window) SetViewerAnalysisWorker(start func(int, []byte), stop func(int)) {
	w.viewer.startWorker = start
	w.viewer.stopWorker = stop
}

func (w *Window) startViewerWorker(command string) error {
	var data bytes.Buffer
	if err := st.WriteWorkerAnalysis(&data, w.frame, w.Cwd(), command); err != nil {
		return err
	}
	w.viewer.workerSerial++
	id := w.viewer.workerSerial
	w.viewer.workerJob = id
	w.SetLastExCommand(command)
	w.CurrentLap("Workerを準備中", 0, 0)
	w.viewer.startWorker(id, data.Bytes())
	return nil
}

// FinishViewerWorker installs only the matching completed job. A failed or
// stopped job never touches the displayed model or the last committed state.
func (w *Window) FinishViewerWorker(id int, data []byte, message string, done func(error)) bool {
	return w.tryEnqueueJob(uiJob{run: func() {
		if w.viewer.workerJob != id {
			done(fmt.Errorf("解析ジョブは既に終了しています"))
			return
		}
		w.viewer.workerJob = 0
		var err error
		if message != "" {
			w.History(message)
			w.CurrentLap(message, 0, 0)
		} else if err = st.ApplyWorkerAnalysis(bytes.NewReader(data), w.frame); err != nil {
			w.History(err.Error())
			w.CurrentLap("結果を反映できません", 0, 0)
		} else {
			w.CurrentLap("解析完了", 0, 0)
		}
		w.Redraw()
		done(err)
	}})
}

func (w *Window) ViewerWorkerUpdate(log string) {
	w.History(log)
}
