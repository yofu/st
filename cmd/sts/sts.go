package main

import (
	"log"

	"golang.org/x/exp/shiny/driver"
	"golang.org/x/exp/shiny/screen"
	"github.com/yofu/st/driver/shiny"
)

func main() {
	driver.Main(func(s screen.Screen) {
		w, err := s.NewWindow(&screen.NewWindowOptions{
			Width:  1024,
			Height: 1024,
		})
		if err != nil {
			log.Fatal(err)
		}
		defer w.Release()
		stw := stshiny.NewWindow(s, w)
		stw.Start()
	})
}
