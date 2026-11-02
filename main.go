package main

import (
	"embed"
	"fmt"
	"os"

	"sendtophone/internal/cli"
	"sendtophone/internal/shell"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

var version = "dev"

func main() {
	if version != "" {
		cli.Version = version
	}
	args := os.Args[1:]
	if cli.WantsCLI(args) {
		attachParentConsole()
		os.Exit(cli.Run(args, os.Stdout, os.Stderr))
	}

	ok, err := shell.ClaimLiveInstance()
	if err == nil && !ok {
		if paths := cli.FileArgs(args); len(paths) > 0 {
			_ = shell.OfferPaths(paths)
		}
		shell.FocusMainWindow()
		return
	}

	if err := ensureWebView2(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	app := NewApp()
	app.pending = cli.FileArgs(args)
	err = wails.Run(&options.App{
		Title:            "Air-Link",
		Width:            420,
		Height:           720,
		MinWidth:         400,
		MinHeight:        560,
		DisableResize:    false,
		Frameless:        false,
		BackgroundColour: &options.RGBA{R: 36, G: 37, B: 41, A: 255},
		AssetServer:      &assetserver.Options{Assets: assets},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		OnBeforeClose:    app.beforeClose,
		Bind:             []interface{}{app},
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: true,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			Theme:                windows.Dark,
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
