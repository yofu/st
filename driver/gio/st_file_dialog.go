package stgio

import (
	"path/filepath"

	st "github.com/yofu/st/stlib"
)

type fileDialogOptions struct {
	Save                                          bool
	Title, Directory, Filename, Filter, Extension string
}

func dialogOptions(kind, current, cwd string) fileDialogOptions {
	o := fileDialogOptions{Title: "ファイルを開く", Directory: cwd, Extension: "inp", Filter: "STモデル (*.inp)\x00*.inp\x00すべてのファイル (*.*)\x00*.*\x00\x00"}
	if current != "" {
		o.Directory = filepath.Dir(current)
	}
	switch kind {
	case "Save As":
		o.Save = true
		o.Title = "名前を付けて保存"
		o.Filename = "model.inp"
		if current != "" {
			o.Filename = filepath.Base(current)
		}
	case "Open DXF":
		o.Title = "DXFファイルを開く"
		o.Extension = "dxf"
		o.Filter = "DXF (*.dxf)\x00*.dxf\x00すべてのファイル (*.*)\x00*.*\x00\x00"
	case "Read":
		o.Title = "解析結果等を読み込む"
		o.Extension = ""
		o.Filter = "STデータ (*.inp;*.inl;*.ihx;*.ihy;*.otl;*.ohx;*.ohy;*.rat;*.rat2;*.lst;*.kjn)\x00*.inp;*.inl;*.ihx;*.ihy;*.otl;*.ohx;*.ohy;*.rat;*.rat2;*.lst;*.kjn\x00すべてのファイル (*.*)\x00*.*\x00\x00"
	case "Insert":
		o.Title = "モデルを挿入"
	}
	return o
}

func (w *Window) usesNativeFileDialog() bool {
	return w.filePicker != nil || w.window != nil && nativeDialogsSupported()
}

// Called on the command worker, never the Gio frame goroutine.
func (w *Window) pickFile(kind string) (string, bool, error) {
	current := ""
	if w.frame != nil {
		current = w.frame.Path
	}
	o := dialogOptions(kind, current, w.Cwd())
	if w.filePicker != nil {
		return w.filePicker(o)
	}
	return platformFileDialog(w.nativeOwner.Load(), o)
}

func (w *Window) applyChosenFile(kind, path string, overwriteConfirmed bool) {
	var err error
	switch kind {
	case "Open", "Open DXF":
		if w.changed && !w.Yn("Open", "現在の変更を破棄して開きますか？") {
			return
		}
		err = st.OpenFile(w, path, false)
	case "Save As":
		if filepath.Ext(path) == "" {
			path += ".inp"
		}
		if !overwriteConfirmed && st.FileExists(path) && !w.Yn("Save", "上書きしますか？") {
			return
		}
		err = st.SaveFile(w, path)
		if err == nil {
			st.Rebase(w, path)
		}
	case "Read":
		err = st.ReadFile(w, path)
	case "Insert":
		position := []float64{0, 0, 0}
		if ns := w.SelectedNodes(); len(ns) > 0 {
			position = ns[0].Coord
		}
		err = w.frame.ReadInp(path, position, 0, false)
		if err == nil {
			st.Snapshot(w)
			w.changed = true
		}
	}
	if err != nil {
		w.History(err.Error())
	}
	w.Redraw()
}
