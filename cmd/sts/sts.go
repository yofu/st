package main

import (
	"golang.org/x/exp/shiny/driver"
	"golang.org/x/exp/shiny/screen"
	"github.com/yofu/st/driver/shiny"
)

func main() {
	driver.Main(func(s screen.Screen) {
		stw := stshiny.NewWindow(s)
		stw.Start()
	})
}
