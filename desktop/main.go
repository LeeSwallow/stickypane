// Command stickypane-app is the desktop second screen for stickypane
// boards: a home window listing the boards on this machine, one window per
// project, a tray icon and a global hotkey (docs/desktop/08).
//
//	stickypane-app [--project=<path>]
//
// It runs once: a second launch hands its --project to the first and ends.
package main

import (
	"embed"
	"log"
	"os"
	"time"

	"github.com/LeeSwallow/stickypane/desktop/internal/shell"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// The frontend build output is embedded into the binary.
//
//go:embed all:frontend/dist
var assets embed.FS

// The tray icon: a template image for the macOS menu bar (36x36, drawn at
// the bar's height) and a colour one for Linux; build/darwin/tray.sh makes
// both.
var (
	//go:embed build/darwin/tray-template@2x.png
	trayTemplate []byte
	//go:embed build/tray.png
	trayColour []byte
)

func main() {
	var s *shell.Shell // made once the app is, used by its callbacks
	ready := make(chan struct{})
	home := &shell.HomeService{}
	board := &shell.ProjectService{}
	setup := &shell.SetupService{}
	layout := &shell.CanvasService{}

	app := application.New(application.Options{
		Name:        "stickypane",
		Description: "A second screen for stickypane boards",
		Services: []application.Service{
			application.NewService(home),
			application.NewService(board),
			application.NewService(setup),
			application.NewService(layout),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			// The app lives on in the tray when its windows are closed.
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "io.github.leeswallow.stickypane",
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
	s, err = shell.New(shell.Config{Home: shell.DefaultHome(), Platform: platform, FocusSettle: 150 * time.Millisecond})
	if err != nil {
		log.Fatal(err)
	}
	home.S, board.S, setup.S, layout.S = s, s, s, s
	platform.StartTray(s, shell.TrayIcons{Template: trayTemplate, Colour: trayColour})
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
