package main

import (
	"os"

	"github.com/andlabs/ui"
	_ "github.com/andlabs/ui/winmanifest"
	stlibui "github.com/yofu/st/driver/libui"
)

func setupUI() {
	fn := "sapporo02.inp"
	if len(os.Args) >= 2 {
		if _, err := os.Stat(os.Args[1]); err == nil {
			fn = os.Args[1]
		}
	}

	mainwin := ui.NewWindow("st", 1200, 1200, true)
	stlibui.SetupWindow(mainwin, fn)
}

func main() {
	ui.Main(setupUI)
}
