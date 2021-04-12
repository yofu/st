package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"

	"github.com/yofu/go-iup/iup"
	"github.com/yofu/st/driver/iup"
	"github.com/yofu/st/stlib"
)

const (
	version  = "0.1.0"
	modified = "LAST CHANGE:26-Mar-2021."
	HOME     = "C:/Users/yofu8/st"
	HOGAN    = "C:/Users/yofu8/st"
)

func main() {
	runtime.GOMAXPROCS(4)
	iup.Open()
	defer iup.Close()
	sw := stgui.NewWindow(HOME)
	if len(os.Args) >= 2 {
		if _, err := os.Stat(os.Args[1]); err == nil {
			st.OpenFile(sw, os.Args[1], true)
		}
	}
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
			w.WriteString(fmt.Sprintf("%v\n\n", r))
			os.Stderr = w
			debug.PrintStack()
		}
	}()
	iup.MainLoop()
}
