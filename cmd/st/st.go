package main

import (
	"github.com/visualfc/go-iup/iup"
	"github.com/yofu/st/driver/iup"
	"os"
	"runtime"
	"runtime/debug"
)

const (
	version  = "0.1.0"
	modified = "LAST CHANGE:11-Jan-2016."
	HOME     = "C:/D/CDOCS/Hogan/Debug"
	HOGAN    = "C:/D/CDOCS/Hogan/Debug"
)

func main() {
	runtime.GOMAXPROCS(4)
	iup.Open()
	defer iup.Close()
	sw := stgui.NewWindow(HOME)
	defer sw.SaveCommandHistory()
	defer stgui.StopLogging()
	sw.Version = version
	sw.Modified = modified
	sw.Dlg.Show()
	sw.FocusCanv()
	defer func() {
		if r := recover(); r != nil {
			w, err := os.Create("st_bugreport.txt")
			if err != nil {
				os.Exit(1)
			}
			os.Stderr = w
			debug.PrintStack()
		}
	}()
	iup.MainLoop()
}