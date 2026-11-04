// Package shell is the app's desktop side: the home window, one
// window per project with its bridge, the tray, the global hotkey, pins,
// and the second launch of a single-instance app (docs/desktop/08 §8.1,
// §8.5; ADR-0007). The desktop is reached through Platform and Window, so
// that everything here runs in tests without a display; windows.go and
// tray.go put Wails behind them.
//
// One file a concern, all of them methods of the one Shell and its lock:
//
//	app.go         Shell, Config, New, Start, Shutdown, the state's save
//	project.go     a board's window and its bridge
//	torn.go        notes torn off into windows of their own, the canvas
//	settingswin.go the settings window
//	home.go        the home window, the hotkey's toggle, the Dock's reopen
//	remote.go      boards on other computers, over ssh
//	indexing.go    the index of boards, the home list, the tray's list
//	initboard.go   making a board (stickypane init) in a folder
//	hotkeys.go     the global hotkey
//	notices.go     the home window's notices, the tray hint
//	prefs.go       the app's settings
//	launch.go      what a launch asks for
//	events.go      the events sent to the frontend
//	services.go    the Wails services the frontend calls
package shell

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/LeeSwallow/stickypane/desktop/internal/bridgeclient"
	"github.com/LeeSwallow/stickypane/desktop/internal/canvas"
	"github.com/LeeSwallow/stickypane/desktop/internal/index"
	"github.com/LeeSwallow/stickypane/desktop/internal/install/model"
	installsvc "github.com/LeeSwallow/stickypane/desktop/internal/install/service"
	"github.com/LeeSwallow/stickypane/desktop/internal/remote"
)

// Board is a running bridge: *bridgeclient.Client.
type Board interface {
	Call(ctx context.Context, method string, params, result any) error
	Notifications() <-chan bridgeclient.Notification
	Hello() bridgeclient.Hello
	Close() error
}

// Index is the index of boards on this machine: *index.Index.
type Index interface {
	Ingest() error
	Prune(now time.Time) ([]index.Project, error)
	Restore(id int64) error
	Retitle(path, title string) error
	Sessions(now time.Time) (index.Groups, error)
	Close() error
}

// Config is what New needs. Find, Start and OpenIndex default to the real
// bridge and index.
type Config struct {
	Home          string // ~/.stickypane, or $STICKYPANE_HOME
	Platform      Platform
	Find          func() (string, error)                   // a stickypane with the bridge
	Probe         func(path string) bridgeclient.Candidate // what a stickypane says of itself
	Start         func(ctx context.Context, bin, project string) (Board, error)
	OpenIndex     func(home string) (Index, error)
	IndexDeadline time.Duration // how long Sessions waits for the index to open
	// Install puts the app's stickypane on the PATH, connects the agents
	// and makes boards (install/service); a test passes a fake.
	Install Installer
	// FocusSettle is how long a board that lost the focus waits to see
	// where it went before it hides; 0 decides at once.
	FocusSettle time.Duration
	// AppVersion is the app's own version (build/config.yml's).
	AppVersion string
	// Autostart opens the app at login; nil where the desktop cannot.
	Autostart Autostart
	// StartArgv starts a bridge from a command line: a remote board's ssh
	// (remote.go). RunArgv runs a short one and answers what it printed:
	// a host's check, ssh -G. A test passes fakes.
	StartArgv func(ctx context.Context, argv []string) (Board, error)
	RunArgv   func(ctx context.Context, argv []string) (string, error)
	// SSH is ssh's program and the socket a host's connections share; nil
	// is $STICKYPANE_SSH or ssh, with a socket of the app's own.
	SSH *remote.SSH
	// SSHAliases lists the hosts of the person's ssh config; nil reads
	// ~/.ssh/config.
	SSHAliases func() []string
	// Now is the clock the home list is made by; nil is the machine's.
	Now func() time.Time
}

// Installer is the install use cases: *installsvc.Service.
type Installer interface {
	Command(ctx context.Context) installsvc.CommandState
	Install(ctx context.Context, t model.Target, replace bool) (installsvc.CommandState, error)
	Uninstall(ctx context.Context) (installsvc.CommandState, error)
	Connect(ctx context.Context, dry, mcp bool) (installsvc.Result, error)
	Init(ctx context.Context, dir, lang string) (installsvc.Result, error)
}

// project is one open project window.
type project struct {
	path   string
	win    Window // nil while it is being made
	board  Board
	state  string
	err    string
	ctx    context.Context // ends when the window closes
	cancel context.CancelFunc
	closed bool
	torn   map[string]*tornWin // its notes torn off into windows, by note
}

// Shell is the app. Its methods are safe for concurrent use; none holds
// the lock while it calls the platform, whose calls wait for the UI thread.
type Shell struct {
	cfg       Config
	statePath string
	saveMu    sync.Mutex // one write of overlay.json at a time
	hotkeyMu  sync.Mutex // one change of the hotkey at a time

	mu       sync.Mutex
	state    State
	home     Window
	projects map[string]*project // by clean path
	byWin    map[uint]*project
	// tornByWin are the windows of torn-off notes (torn.go)
	tornByWin map[uint]*tornWin
	notices   []Notice
	hotkey    string // registered accelerator, "" when none
	hello     bridgeclient.Hello
	started   bool
	settings  *settingsWin // the settings window (settingswin.go), nil when closed

	sshOnce sync.Once // ssh() is decided once
	sshCfg  remote.SSH

	ix      Index
	ixErr   error
	ixReady chan struct{} // closed when the first open ended or timed out
	quit    chan struct{}
	stopped bool
}

// DefaultHome is $STICKYPANE_HOME, else ~/.stickypane.
func DefaultHome() string {
	if h := os.Getenv("STICKYPANE_HOME"); h != "" {
		return h
	}
	h, err := os.UserHomeDir()
	if err != nil || h == "" {
		return ".stickypane"
	}
	return filepath.Join(h, ".stickypane")
}

// New reads the state and returns the shell; Start registers the hotkey
// and opens the index.
func New(cfg Config) (*Shell, error) {
	if cfg.Home == "" {
		cfg.Home = DefaultHome()
	}
	if cfg.Find == nil {
		cfg.Find = bridgeclient.Find
	}
	if cfg.Probe == nil {
		cfg.Probe = bridgeclient.Probe
	}
	if cfg.Start == nil {
		cfg.Start = func(ctx context.Context, bin, project string) (Board, error) {
			c, err := bridgeclient.Start(ctx, bin, project)
			if err != nil {
				return nil, err
			}
			return c, nil
		}
	}
	if cfg.StartArgv == nil {
		cfg.StartArgv = func(ctx context.Context, argv []string) (Board, error) {
			c, err := bridgeclient.StartArgv(ctx, argv)
			if err != nil {
				return nil, err
			}
			return c, nil
		}
	}
	if cfg.RunArgv == nil {
		cfg.RunArgv = bridgeclient.RunArgv
	}
	if cfg.SSHAliases == nil {
		cfg.SSHAliases = defaultAliases
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.OpenIndex == nil {
		cfg.OpenIndex = func(home string) (Index, error) {
			ix, err := index.Open(home)
			if err != nil {
				return nil, err
			}
			return ix, nil
		}
	}
	if cfg.Install == nil {
		cfg.Install = defaultInstaller(cfg.Find)
	}
	if cfg.IndexDeadline == 0 {
		cfg.IndexDeadline = defaultIndexDeadline
	}
	path := filepath.Join(cfg.Home, StateFile)
	st, err := LoadState(path)
	s := &Shell{
		cfg:       cfg,
		statePath: path,
		state:     st,
		projects:  map[string]*project{},
		byWin:     map[uint]*project{},
		tornByWin: map[uint]*tornWin{},
		ixReady:   make(chan struct{}),
		quit:      make(chan struct{}),
	}
	if err != nil { // a damaged file costs the window places, not the app
		s.notices = append(s.notices, Notice{Code: "state_unreadable", Detail: err.Error()})
	}
	return s, nil
}

// Start runs once the app runs: it registers the hotkey and opens the
// index off the caller's thread, bounded by IndexDeadline.
func (s *Shell) Start() {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return
	}
	s.started = true
	accel := s.state.Hotkey
	s.mu.Unlock()

	s.registerHotkey(accel)
	go s.openIndex()
}

// Quit ends the app: every window, every bridge, the tray.
func (s *Shell) Quit() { s.cfg.Platform.Quit() }

// save writes the state file.
// Saves are one at a time, each taking the state when its turn comes, so
// an older state never lands after a newer one.
func (s *Shell) save() error {
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	s.mu.Lock()
	st := s.state.clone()
	s.mu.Unlock()
	return st.Save(s.statePath)
}

// Shutdown closes every bridge and the index and saves the state. It is
// safe to call more than once.
func (s *Shell) Shutdown() {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	close(s.quit)
	var boards []Board
	var cancels []context.CancelFunc
	var open []*project
	var torn []*tornWin
	for _, p := range s.projects {
		if p.win != nil {
			open = append(open, p)
		}
		for _, tw := range p.torn {
			if tw.win != nil && !tw.closed {
				torn = append(torn, tw)
			}
		}
		if p.board != nil {
			boards = append(boards, p.board)
		}
		if p.cancel != nil {
			cancels = append(cancels, p.cancel)
		}
	}
	ix := s.ix
	s.ix = nil
	s.mu.Unlock()
	for _, p := range open { // where each window is now; the platform is asked outside the lock
		if b := p.win.Bounds(); b.Width > 0 && b.Height > 0 {
			s.mu.Lock()
			s.place(p, b)
			s.mu.Unlock()
		}
	}
	for _, tw := range torn { // and each torn-off note's, kept for the next launch
		b := tw.win.Bounds()
		s.mu.Lock()
		if f, ok := s.frameOf(tw, b); ok {
			s.edit(tw.p, func(c canvas.Project) canvas.Project { return c.Tear(tw.note, f) })
		}
		s.mu.Unlock()
	}
	for _, c := range cancels {
		c()
	}
	for _, b := range boards {
		b.Close()
	}
	if ix != nil {
		ix.Close()
	}
	s.save()
}
