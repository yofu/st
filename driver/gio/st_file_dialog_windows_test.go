package stgio

import (
	"testing"
	"unsafe"

	"gioui.org/app"
	"golang.org/x/sys/windows"
)

func TestWindowsFileDialogABIAndFlags(t *testing.T) {
	var f openFilenameW
	wantSize, wantFile, wantFlags := uintptr(152), uintptr(48), uintptr(96)
	if unsafe.Sizeof(uintptr(0)) == 4 {
		wantSize, wantFile, wantFlags = 88, 28, 52
	}
	if unsafe.Sizeof(f) != wantSize || unsafe.Offsetof(f.File) != wantFile || unsafe.Offsetof(f.Flags) != wantFlags {
		t.Fatalf("OPENFILENAMEW ABI: size=%d File=%d Flags=%d", unsafe.Sizeof(f), unsafe.Offsetof(f.File), unsafe.Offsetof(f.Flags))
	}
	for _, save := range []bool{false, true} {
		o := dialogOptions("Save As", `C:\日本語 folder\model.inp`, "")
		o.Save = save
		f, buffer, err := prepareOpenFilename(123, o)
		if err != nil {
			t.Fatal(err)
		}
		if f.Size != uint32(wantSize) || f.Owner != 123 || f.MaxFile != uint32(len(buffer)) || windows.UTF16ToString(buffer) != "model.inp" || windows.UTF16PtrToString(f.InitialDir) != o.Directory || windows.UTF16PtrToString(f.DefaultExt) != "inp" {
			t.Fatalf("bad options: %+v", f)
		}
		if f.Flags&ofNoChangeDir == 0 || f.Flags&ofPathMustExist == 0 || f.Flags&ofExplorer == 0 {
			t.Fatal("missing flags")
		}
		if (f.Flags&ofOverwritePrompt != 0) != save || (f.Flags&ofFileMustExist != 0) == save {
			t.Fatal("incorrect save/open flags")
		}
	}
	o := dialogOptions("Open", "", "")
	o.Filter = "missing terminators"
	if _, _, err := prepareOpenFilename(0, o); err == nil {
		t.Fatal("invalid filter accepted")
	}
	o = dialogOptions("Open", "", "")
	o.Directory = "bad\x00path"
	if _, _, err := prepareOpenFilename(0, o); err == nil {
		t.Fatal("invalid directory accepted")
	}
	for _, proc := range []*windows.LazyProc{getOpenFileNameW, getSaveFileNameW, commDlgExtendedError} {
		if err := proc.Find(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestWindowsFileDialogParent(t *testing.T) {
	w := NewWindow(nil)
	defer w.ui.close()
	w.platformWindowEvent(app.Win32ViewEvent{HWND: 456})
	if w.nativeOwner.Load() != 456 {
		t.Fatal("parent HWND not captured")
	}
	w.platformWindowEvent(app.Win32ViewEvent{})
	if w.nativeOwner.Load() != 0 {
		t.Fatal("destroyed HWND retained")
	}
}
