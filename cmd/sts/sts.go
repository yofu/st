package main

import (
	"log"
	"os"

	"github.com/yofu/st/driver/shiny"
	"golang.org/x/exp/shiny/driver"
	"golang.org/x/exp/shiny/screen"
)

func main() {
	fn := ""
	if len(os.Args) >= 2 {
		if _, err := os.Stat(os.Args[1]); err == nil {
			fn = os.Args[1]
		}
	}
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
		stw.Start(fn)
	})
}
