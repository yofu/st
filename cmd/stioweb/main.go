//go:build js && wasm

package main

import (
	"fmt"
	"syscall/js"

	"gioui.org/app"
	stgio "github.com/yofu/st/driver/gio"
)

func main() {
	global := js.Global()
	go func() {
		window := new(app.Window)
		viewer := stgio.NewModelViewer(window, func(s stgio.ViewerState) {
			global.Call("stViewerState", s.Name, s.Nodes, s.Members, s.SelectedNodes, s.SelectedMembers, s.Editing, s.Modified, s.History, s.PromptID, s.PromptTitle, s.PromptInitial, s.Progress, s.Annotations, s.Question)
		})
		viewer.SetViewerBrowserAction(func(action string, paths []string) {
			values := make([]any, len(paths))
			for i, path := range paths {
				values[i] = path
			}
			global.Call("stViewerBrowserAction", action, js.ValueOf(values))
		})
		viewer.SetViewerAnalysisWorker(func(id int, data []byte) {
			array := global.Get("Uint8Array").New(len(data))
			js.CopyBytesToJS(array, data)
			global.Call("stViewerWorkerStart", id, array)
		}, func(id int) {
			global.Call("stViewerWorkerStop", id)
		})
		workerFinish := js.FuncOf(func(_ js.Value, args []js.Value) any {
			if len(args) != 3 {
				return false
			}
			var data []byte
			if !args[1].IsNull() {
				data = make([]byte, args[1].Get("byteLength").Int())
				js.CopyBytesToGo(data, args[1])
			}
			id := args[0].Int()
			return viewer.FinishViewerWorker(id, data, args[2].String(), func(err error) {
				message := ""
				if err != nil {
					message = err.Error()
				}
				global.Call("stViewerWorkerApplied", id, message)
			})
		})
		workerLog := js.FuncOf(func(_ js.Value, args []js.Value) any {
			if len(args) == 1 {
				viewer.ViewerWorkerUpdate(args[0].String())
			}
			return nil
		})
		defer workerFinish.Release()
		defer workerLog.Release()

		open := js.FuncOf(func(_ js.Value, args []js.Value) any {
			if len(args) != 2 {
				return false
			}
			size := args[1].Get("byteLength").Int()
			if size <= 0 || size > stgio.ViewerModelLimit {
				return false
			}
			data := make([]byte, size)
			if js.CopyBytesToGo(data, args[1]) != size {
				return false
			}
			return viewer.OpenViewerModel(args[0].String(), data, func(err error) {
				message := ""
				if err != nil {
					message = err.Error()
				}
				global.Call("stViewerLoaded", message)
			})
		})
		action := js.FuncOf(func(_ js.Value, args []js.Value) any {
			return len(args) == 1 && viewer.ViewerAction(args[0].String())
		})
		command := js.FuncOf(func(_ js.Value, args []js.Value) any {
			return len(args) == 1 && viewer.SubmitViewerCommand(args[0].String())
		})
		export := js.FuncOf(func(_ js.Value, args []js.Value) any {
			name := ""
			if len(args) > 0 {
				name = args[0].String()
			}
			return viewer.ExportViewerModelAs(name, func(name string, data []byte, err error) {
				if err != nil {
					global.Call("stViewerExported", "", js.Null(), err.Error())
					return
				}
				array := global.Get("Uint8Array").New(len(data))
				js.CopyBytesToJS(array, data)
				global.Call("stViewerExported", name, array, "")
			})
		})
		reply := js.FuncOf(func(_ js.Value, args []js.Value) any {
			return len(args) == 3 && viewer.ReplyViewerPrompt(args[0].Int(), args[1].String(), args[2].Bool())
		})
		answer := js.FuncOf(func(_ js.Value, args []js.Value) any {
			return len(args) == 2 && viewer.ReplyViewerQuestion(args[0].Int(), args[1].Int())
		})
		defer answer.Release()
		defer command.Release()
		defer export.Release()
		defer reply.Release()
		defer open.Release()
		defer action.Release()
		names := make([]any, 0)
		for _, name := range stgio.ViewerCommandNames() {
			names = append(names, name)
		}
		global.Call("stViewerReady", open, action, command, export, reply, js.ValueOf(names), answer, workerFinish, workerLog)
		if err := viewer.Run(window); err != nil {
			global.Call("stViewerFatal", fmt.Sprintf("Gio: %v", err))
		}
	}()
	app.Main()
}
