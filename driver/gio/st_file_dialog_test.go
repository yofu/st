package stgio

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	st "github.com/yofu/st/stlib"
)

func TestNativeFileDialogOptions(t *testing.T) {
	current := filepath.Join(t.TempDir(), "日本語 model.inp")
	for _, kind := range []string{"Open", "Save As", "Open DXF", "Read", "Insert"} {
		o := dialogOptions(kind, current, "unused")
		if o.Directory != filepath.Dir(current) || !strings.HasSuffix(o.Filter, "\x00\x00") {
			t.Fatalf("%s: invalid directory/filter: %+v", kind, o)
		}
		if o.Save != (kind == "Save As") {
			t.Fatalf("%s: save flag", kind)
		}
		if o.Save && o.Filename != filepath.Base(current) {
			t.Fatal(o.Filename)
		}
	}
	if o := dialogOptions("Open", "", "initial"); o.Directory != "initial" {
		t.Fatal(o)
	}
}

func TestNativeFilePickerOpenSaveAndCancel(t *testing.T) {
	h := newUIHarness(t)
	w := h.w
	dir := t.TempDir()
	if err := w.LoadHistory(filepath.Join(dir, "history")); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "日本語 model.inp")
	f := st.NewFrame()
	n := st.NewNode()
	n.Num = 101
	n.Frame = f
	n.Coord = []float64{1, 2, 3}
	f.Nodes[n.Num] = n
	if err := f.WriteInp(path); err != nil {
		t.Fatal(err)
	}
	w.filePicker = func(o fileDialogOptions) (string, bool, error) {
		if o.Save {
			return "", false, fmt.Errorf("unexpected save")
		}
		return path, true, nil
	}
	w.fileDialog("Open")
	waitUI(t, w)
	if w.ui.modal != nil || w.frame.Path != path || w.frame.Nodes[101] == nil {
		t.Fatalf("native open failed: %s", w.frame.Path)
	}
	original := w.frame
	w.filePicker = func(fileDialogOptions) (string, bool, error) { return "", false, nil }
	w.fileDialog("Open")
	waitUI(t, w)
	if w.frame != original || w.frame.Path != path {
		t.Fatal("cancel changed model")
	}
	w.fileDialog("Save As")
	waitUI(t, w)
	if w.frame != original || w.frame.Path != path {
		t.Fatal("cancel changed save path")
	}

	// The Windows picker has already confirmed overwriting this existing file.
	saved := filepath.Join(dir, "保存 model.inp")
	if err := os.WriteFile(saved, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	w.filePicker = func(o fileDialogOptions) (string, bool, error) {
		if !o.Save || o.Filename != filepath.Base(w.frame.Path) {
			return "", false, fmt.Errorf("invalid save options")
		}
		return saved, true, nil
	}
	w.fileDialog("Save As")
	waitUI(t, w)
	if w.frame.Path != saved || w.ui.modal != nil {
		t.Fatal("save did not rebase / asked twice")
	}
	data, err := os.ReadFile(saved)
	if err != nil || !strings.Contains(string(data), "101") {
		t.Fatalf("saved data: %v", err)
	}

	// SaveAS is also called synchronously by commands on the command worker.
	commandPath := filepath.Join(dir, "command save")
	w.filePicker = func(fileDialogOptions) (string, bool, error) { return commandPath, true, nil }
	w.enqueue(w.SaveAS)
	waitUI(t, w)
	if w.frame.Path != commandPath+".inp" {
		t.Fatal(w.frame.Path)
	}
	w.filePicker = func(fileDialogOptions) (string, bool, error) { return "", false, fmt.Errorf("picker failed") }
	w.fileDialog("Open")
	waitUI(t, w)
	if w.frame.Path != commandPath+".inp" {
		t.Fatal("error changed model")
	}
}
