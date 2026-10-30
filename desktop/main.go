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

// appVersion is the app's version: build/config.yml's info.version
// (version_test.go keeps them the same).
const appVersion = "0.1.0"

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
			UniqueID: instanceID(os.Getenv("STICKYPANE_APP_INSTANCE")),
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
	s, err = shell.New(shell.Config{
		Home:        shell.DefaultHome(),
		Platform:    platform,
		FocusSettle: 150 * time.Millisecond,
		AppVersion:  appVersion,
		Autostart:   shell.WailsAutostart(app),
	})
	if err != nil {
		log.Fatal(err)
	}
	home.S, board.S, setup.S, layout.S = s, s, s, s
	platform.StartTray(s, shell.TrayIcons{Template: trayTemplate, Colour: trayColour})
	close(ready)

	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		s.Start()
		s.Launch(os.Args, "")
		debugTear(s)
		// STICKYPANE_APP_OPEN=settings[:section] opens the settings window
		// at launch, for the last board opened: a way to look at it
		// without pressing anything (docs/desktop/06).
		s.OpenAtLaunch(os.Getenv("STICKYPANE_APP_OPEN"))
	})
	// Clicking the Dock icon, or opening the app again, with every window
	// hidden brings back the board used last, or the home when none is open.
	app.Event.OnApplicationEvent(events.Mac.ApplicationShouldHandleReopen, func(*application.ApplicationEvent) {
		s.Reopen()
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// instanceID is the single instance's name. STICKYPANE_APP_INSTANCE=<name>
// gives a copy run for a check its own (with its own STICKYPANE_HOME), so
// that it does not hand its launch to an app already running.
func instanceID(name string) string {
	const id = "io.github.leeswallow.stickypane"
	if name == "" {
		return id
	}
	return id + "." + name
}

// debugTear is a debug hook, for checking tear-off in the built app
// without a mouse: STICKYPANE_APP_TEAR=<note> with --project=<path> tears
// that note off the project's board a moment after launch, through the
// same Shell.TearOff a click on the tear-off button calls, and logs what
// came of it. Unset, it does nothing.
func debugTear(s *shell.Shell) {
	note := os.Getenv("STICKYPANE_APP_TEAR")
	project := shell.ProjectArg(os.Args)
	if note == "" || project == "" {
		return
	}
	go func() {
		time.Sleep(3 * time.Second)
		id := s.OpenProject(project)
		tid, err := s.TearOff(id, note)
		log.Printf("STICKYPANE_APP_TEAR %q: window %d, err %v", note, tid, err)
	}()
}
