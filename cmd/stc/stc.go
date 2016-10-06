package main

import (
	"github.com/yofu/st/driver/cui"
	"runtime"
)

const (
	version = "0.1.0"
	HOME    = "C:/D/CDOCS/Hogan/Debug"
)

func main() {
	runtime.GOMAXPROCS(4)
	sw := stcui.NewWindow(HOME)
	sw.MainLoop()
}
