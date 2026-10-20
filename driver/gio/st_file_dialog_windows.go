package stgio

import (
	"fmt"
	"runtime"
	"unicode/utf16"
	"unsafe"

	"gioui.org/app"
	"gioui.org/io/event"
	"golang.org/x/sys/windows"
)

// Windows SDK OPENFILENAMEW, including the Windows 2000+ reserved fields.
// The SDK's _MAC-only fields must not be present in the Windows ABI.
type openFilenameW struct {
	Size                         uint32
	Owner, Instance              uintptr
	Filter, CustomFilter         *uint16
	MaxCustomFilter, FilterIndex uint32
	File                         *uint16
	MaxFile                      uint32
	FileTitle                    *uint16
	MaxFileTitle                 uint32
	InitialDir, Title            *uint16
	Flags                        uint32
	FileOffset, FileExtension    uint16
	DefaultExt                   *uint16
	CustomData, Hook             uintptr
	TemplateName                 *uint16
	Reserved                     uintptr
	ReservedWord, FlagsEx        uint32
}

const (
	ofOverwritePrompt = 0x2
	ofHideReadOnly    = 0x4
	ofNoChangeDir     = 0x8
	ofPathMustExist   = 0x800
	ofFileMustExist   = 0x1000
	ofExplorer        = 0x80000
	ofEnableSizing    = 0x800000
)

var commonDialogs = windows.NewLazySystemDLL("comdlg32.dll")
var getOpenFileNameW = commonDialogs.NewProc("GetOpenFileNameW")
var getSaveFileNameW = commonDialogs.NewProc("GetSaveFileNameW")
var commDlgExtendedError = commonDialogs.NewProc("CommDlgExtendedError")

func nativeDialogsSupported() bool { return true }
func (w *Window) platformWindowEvent(e event.Event) {
	if e, ok := e.(app.Win32ViewEvent); ok {
		w.nativeOwner.Store(e.HWND)
	}
}

func prepareOpenFilename(owner uintptr, o fileDialogOptions) (openFilenameW, []uint16, error) {
	var f openFilenameW
	name, err := windows.UTF16FromString(o.Filename)
	if err != nil {
		return f, nil, err
	}
	buffer := make([]uint16, 32768)
	if len(name) > len(buffer) {
		return f, nil, fmt.Errorf("initial filename is too long")
	}
	copy(buffer, name)
	directory, err := windows.UTF16PtrFromString(o.Directory)
	if err != nil {
		return f, nil, err
	}
	title, err := windows.UTF16PtrFromString(o.Title)
	if err != nil {
		return f, nil, err
	}
	ext, err := windows.UTF16PtrFromString(o.Extension)
	if err != nil {
		return f, nil, err
	}
	filter := utf16.Encode([]rune(o.Filter))
	if len(filter) < 2 || filter[len(filter)-1] != 0 || filter[len(filter)-2] != 0 {
		return f, nil, fmt.Errorf("file filter must end with two NULs")
	}
	f = openFilenameW{Size: uint32(unsafe.Sizeof(f)), Owner: owner, Filter: &filter[0], FilterIndex: 1, File: &buffer[0], MaxFile: uint32(len(buffer)), InitialDir: directory, Title: title, DefaultExt: ext,
		Flags: ofExplorer | ofEnableSizing | ofHideReadOnly | ofNoChangeDir | ofPathMustExist}
	if o.Save {
		f.Flags |= ofOverwritePrompt
	} else {
		f.Flags |= ofFileMustExist
	}
	return f, buffer, nil
}

func platformFileDialog(owner uintptr, o fileDialogOptions) (string, bool, error) {
	// The modal dialog and its error code must stay on the same OS thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	f, buffer, err := prepareOpenFilename(owner, o)
	if err != nil {
		return "", false, err
	}
	proc := getOpenFileNameW
	if o.Save {
		proc = getSaveFileNameW
	}
	if err := proc.Find(); err != nil {
		return "", false, err
	}
	if err := commDlgExtendedError.Find(); err != nil {
		return "", false, err
	}
	ok, _, _ := proc.Call(uintptr(unsafe.Pointer(&f)))
	runtime.KeepAlive(f)
	if ok == 0 {
		code, _, _ := commDlgExtendedError.Call()
		if code != 0 {
			return "", false, fmt.Errorf("Windows file dialog failed (CommDlgExtendedError=0x%X)", code)
		}
		return "", false, nil // User cancelled; no model or history changes.
	}
	return windows.UTF16ToString(buffer), true, nil
}
