// Command stickypane-overlay is the desktop second screen for stickypane
// boards: a home window listing the boards on this machine, one window per
// project, a tray icon and a global hotkey (docs/overlay/08).
//
//	stickypane-overlay [--project=<path>]
//
// It runs once: a second launch hands its --project to the first and ends.
package main

import (
	"embed"
	"log"
	"os"

	"github.com/LeeSwallow/stickypane/overlay/internal/shell"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// The frontend build output is embedded into the binary.
//
//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/tray.png
var trayIcon []byte

func main() {
	var s *shell.Shell // made once the app is, used by its callbacks
	ready := make(chan struct{})
	home := &shell.HomeService{}
	board := &shell.ProjectService{}

	app := application.New(application.Options{
		Name:        "stickypane",
		Description: "A second screen for stickypane boards",
		Services: []application.Service{
			application.NewService(home),
			application.NewService(board),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			// The app lives on in the tray when its windows are closed.
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "io.github.leeswallow.stickypane.overlay",
			OnSecondInstanceLaunch: func(d application.SecondInstanceData) {
				<-ready
				s.Launch(d.Args, d.WorkingDir)
			},
		},
		OnShutdown: func() {
			if s != nil {
				s.Shutdown()
			}
		},
	})

	// The menu bar: the app menu's Quit (Cmd+Q) ends the app, not a window.
	app.Menu.Set(shell.AppMenu())

	platform := shell.NewWails(app)
	var err error
	s, err = shell.New(shell.Config{Home: shell.DefaultHome(), Platform: platform})
	if err != nil {
		log.Fatal(err)
	}
	home.S, board.S = s, s
	platform.StartTray(s, trayIcon)
	close(ready)

	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		s.Start()
		s.Launch(os.Args, "")
	})
	// Clicking the Dock icon with every window hidden brings the home back.
	app.Event.OnApplicationEvent(events.Mac.ApplicationShouldHandleReopen, func(*application.ApplicationEvent) {
		s.ShowHome()
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
