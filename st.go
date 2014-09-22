package main

import (
    "runtime"
    "github.com/visualfc/go-iup/iup"
    "github.com/yofu/st/stgui"
)

const (
    version = "0.1.0"
    modified = "LAST CHANGE:22-Sep-2014."
    HOME = "C:/D/CDOCS/Hogan/Debug"
    HOGAN = "C:/D/CDOCS/Hogan/Debug"
)

func main() {
    runtime.GOMAXPROCS(4)
    iup.Open()
    defer iup.Close()
    sw := stgui.NewWindow(HOME)
    sw.Version = version
    sw.Modified = modified
    sw.Dlg.Show()
    sw.FocusCanv()
    // brk := make(chan bool)
    // go stgui.UpdateInps("C:/D/CDOCS/Hogan", brk)
    iup.MainLoop()
}
