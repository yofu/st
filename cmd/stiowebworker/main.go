//go:build js && wasm

package main

import (
	"bytes"
	"fmt"
	"runtime/debug"
	"syscall/js"
	"time"

	st "github.com/yofu/st/stlib"
)

type workerLog struct {
	buffer bytes.Buffer
	last   time.Time
}

func (l *workerLog) Write(data []byte) (int, error) {
	n, err := l.buffer.Write(data)
	if time.Since(l.last) >= 200*time.Millisecond || l.buffer.Len() > 64*1024 {
		l.flush()
	}
	return n, err
}

func (l *workerLog) flush() {
	if l.buffer.Len() == 0 {
		return
	}
	js.Global().Call("stWorkerLog", l.buffer.String())
	l.last = time.Now()
	l.buffer.Reset()
}

func main() {
	global := js.Global()
	run := js.FuncOf(func(_ js.Value, args []js.Value) any {
		array := args[0]
		data := make([]byte, array.Get("byteLength").Int())
		js.CopyBytesToGo(data, array)
		go func() {
			log := &workerLog{}
			defer func() {
				if p := recover(); p != nil {
					fmt.Fprintln(log, string(debug.Stack()))
					log.flush()
					global.Call("stWorkerDone", js.Null(), fmt.Sprintf("解析エラー: %v", p))
				}
			}()
			var result bytes.Buffer
			err := st.RunWorkerAnalysis(bytes.NewReader(data), &result, log, func(phase string, step, total int) {
				global.Call("stWorkerProgress", phase, step, total)
			})
			log.flush()
			if err != nil {
				global.Call("stWorkerDone", js.Null(), err.Error())
				return
			}
			value := global.Get("Uint8Array").New(result.Len())
			js.CopyBytesToJS(value, result.Bytes())
			global.Call("stWorkerDone", value, "")
		}()
		return nil
	})
	defer run.Release()
	global.Call("stWorkerReady", run)
	select {}
}
