package main

import (
    "github.com/visualfc/go-iup/iup"
    "github.com/yofu/st/stlib"
)

const (
    version = "0.1.0"
    modified = "LAST CHANGE:28-Feb-2014."
    HOME = "C:/D/CDOCS/Hogan"
)

func main() {
    iup.Open()
    defer iup.Close()
    sw := st.NewWindow(HOME)
    sw.Version = version
    sw.Modified = modified
    sw.Dlg.Show()
    iup.MainLoop()
}
