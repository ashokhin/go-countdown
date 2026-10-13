package main

import (
	"context"
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/ashokhin/go-countdown/internal/app"
)

//go:embed all:frontend/dist
var assets embed.FS

// appIconPNG is used for the window/taskbar icon on Linux (see
// options.Linux.Icon below). Windows instead gets its icon compiled into the
// .exe itself from build/windows/icon.ico, and doesn't need this.
//
//go:embed assets/icon.png
var appIconPNG []byte

func main() {
	backend := app.New()
	prefs := app.LoadPrefs()

	err := wails.Run(&options.App{
		Title:  "Countdown Timers",
		Width:  prefs.WindowWidth,
		Height: prefs.WindowHeight,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        backend.Startup,
		OnBeforeClose: func(ctx context.Context) bool {
			width, height := wailsruntime.WindowGetSize(ctx)
			_ = app.SavePrefs(app.Prefs{WindowWidth: width, WindowHeight: height})
			return false // false = allow the window to actually close
		},
		Linux: &linux.Options{
			Icon: appIconPNG,
		},
		Bind: []interface{}{
			backend,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
