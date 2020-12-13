package main

import (
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/unit" // unit is used to define pixel-independent sizes
	"github.com/yofu/st/driver/gio"
)

func main() {

	go func() {
		w := app.NewWindow(
			app.Title("st"),
			app.Size(unit.Dp(1000), unit.Dp(1000)),
		)
		stw := stgio.NewWindow(w)
		if err := stw.Run(w); err != nil {
			log.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}()

	// This starts Gio main.
	app.Main()
}
