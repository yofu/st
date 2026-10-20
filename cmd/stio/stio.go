package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gioui.org/app"
	"gioui.org/unit" // unit is used to define pixel-independent sizes
	"github.com/yofu/st/driver/gio"
	st "github.com/yofu/st/stlib"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "usage: stio [model.inp]")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() > 1 {
		log.Fatal("usage: stio [model.inp]")
	}
	go func() {
		w := new(app.Window)
		w.Option(app.Title("st — Gio"), app.Size(unit.Dp(1000), unit.Dp(1000)), app.Maximized.Option())
		stw := stgio.NewWindow(w)
		stw.SetCanvasRotateSpeedX(0.03)
		stw.SetCanvasRotateSpeedY(0.03)
		stw.SetCanvasMoveSpeedX(0.2)
		stw.SetCanvasMoveSpeedY(0.2)
		homeDirectory := os.Getenv("HOME")
		if homeDirectory == "" {
			homeDirectory, _ = os.UserHomeDir()
		}
		stw.SetHome(homeDirectory)
		if cwd, err := os.Getwd(); err == nil {
			stw.SetCwd(cwd)
		}
		if homeDirectory != "" {
			if err := stw.LoadHistory(filepath.Join(homeDirectory, ".st")); err != nil {
				stw.History(err.Error())
			}
		}
		if err := stw.LoadAliases(); err != nil {
			stw.History(fmt.Sprintf("st.pgpを読み込めません: %v", err))
		}
		if flag.NArg() == 1 {
			if err := st.OpenFile(stw, flag.Arg(0), false); err != nil {
				stw.History(fmt.Sprintf("ファイルを開けません: %v", err))
			}
		}
		if err := stw.Run(w); err != nil {
			log.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}()

	// This starts Gio main.
	app.Main()
}
