// Package service is the install use cases (docs/overlay/09 §9.4): put the
// app's own stickypane on the PATH as the terminal command, or take it
// off; say what a terminal would run instead; connect the agents through
// the bundled stickypane's setup; and make a board with its init. It
// reaches the world only through install/ports (ADR-0018).
package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/LeeSwallow/stickypane/overlay/internal/install/model"
	"github.com/LeeSwallow/stickypane/overlay/internal/install/ports"
)

// Errors the screen tells apart.
var (
	// ErrNoBundled: this build of the app carries no stickypane (a
	// development build), so there is nothing to link.
	ErrNoBundled = errors.New("this app has no stickypane of its own to install")
	// ErrOccupied: another program is where the link would go; the person
	// is asked before it is replaced.
	ErrOccupied = errors.New("another stickypane is there")
	// ErrUnsafeDir: setup or init was asked to run in the root or the home
	// folder.
	ErrUnsafeDir = errors.New("will not run there")
)

// Timeouts of the programs run.
var (
	pathTimeout    = 5 * time.Second
	versionTimeout = 10 * time.Second
)

// Config is what the service needs.
type Config struct {
	Files  ports.Files
	Runner ports.Runner
	Admin  ports.Admin
	Home   string   // the person's home folder
	Shell  string   // $SHELL, whose login PATH a terminal has
	PATH   []string // the app's own PATH (a GUI app's is short)
	// Bundled is the stickypane inside the app, "" when there is none.
	Bundled func() string
	// Find is the stickypane the app would use otherwise (bridge.Find).
	Find func() (string, error)
}

// Service is the use cases. Its methods are safe for concurrent use.
type Service struct {
	cfg Config

	mu        sync.Mutex
	loginPath []string // the PATH a terminal has, once read
	pathRead  bool
	versions  map[string]string
}

// New returns the service.
func New(cfg Config) *Service {
	if cfg.Bundled == nil {
		cfg.Bundled = func() string { return "" }
	}
	if cfg.Find == nil {
		cfg.Find = func() (string, error) { return "", errors.New("no stickypane found") }
	}
	return &Service{cfg: cfg, versions: map[string]string{}}
}

// Link is one place the terminal command can be linked.
type Link struct {
	Target model.Target `json:"target"`
	Path   string       `json:"path"`
	// State is "none" (nothing there), "apps" (a link to this app's
	// stickypane, or another copy of the app's) or "other" (a program the
	// app did not put there).
	State string `json:"state"`
	Real  string `json:"real,omitempty"`
}

// CommandState is what the settings show of the terminal command.
type CommandState struct {
	Bundled string `json:"bundled"` // "" when the app has none
	Version string `json:"version"` // the bundled stickypane's
	Links   []Link `json:"links"`
	// First is what `stickypane` in a terminal runs, nil when nothing.
	First *model.Copy `json:"first,omitempty"`
	// Shadow, when First is not the app's, is how to get past it.
	Shadow *model.Fix `json:"shadow,omitempty"`
	// UserDir is ~/.local/bin; OnPath says whether a terminal's PATH has
	// it, and RCFile and RCLine how to add it when not.
	UserDir string `json:"userDir"`
	OnPath  bool   `json:"onPath"`
	RCFile  string `json:"rcFile"`
	RCLine  string `json:"rcLine"`
}

// Command says where the terminal command stands.
func (s *Service) Command(ctx context.Context) CommandState {
	bundled := s.cfg.Bundled()
	path := s.LoginPath(ctx)
	st := CommandState{Bundled: bundled, UserDir: model.Dir(model.TargetUser, s.cfg.Home)}
	if bundled != "" {
		st.Version = s.version(ctx, bundled)
	}
	for _, t := range model.Targets {
		st.Links = append(st.Links, s.link(t, bundled))
	}
	st.OnPath = model.OnPath(st.UserDir, path)
	st.RCFile, st.RCLine = model.RCFile(s.cfg.Shell, s.cfg.Home, st.UserDir)
	if c := s.first(ctx, path, bundled); c != nil {
		st.First = c
		if !c.Apps {
			linkDir := st.UserDir
			for _, l := range st.Links {
				if l.State == "apps" {
					linkDir = filepath.Dir(l.Path)
					break
				}
			}
			f := model.FixFor(*c, linkDir)
			st.Shadow = &f
		}
	}
	return st
}

func (s *Service) link(t model.Target, bundled string) Link {
	l := Link{Target: t, Path: model.LinkPath(t, s.cfg.Home), State: "none"}
	e := s.cfg.Files.Stat(l.Path)
	switch {
	case !e.Exists:
	case e.Link && (e.Real == "" || model.IsApps(e.Real, bundled)):
		// a dangling link is left over from an app that was moved or removed
		l.State, l.Real = "apps", e.Real
	default:
		l.State, l.Real = "other", e.Real
	}
	return l
}

// first is the stickypane a terminal finds first on path.
func (s *Service) first(ctx context.Context, path []string, bundled string) *model.Copy {
	for _, dir := range path {
		p := filepath.Join(dir, model.Name)
		e := s.cfg.Files.Stat(p)
		if !e.Exists || !e.Exec {
			continue
		}
		c := model.Copy{Path: p, Real: e.Real, Apps: model.IsApps(e.Real, bundled)}
		if !c.Apps {
			c.Version = s.version(ctx, p)
		}
		return &c
	}
	return nil
}

// version asks a stickypane its version, once per path.
func (s *Service) version(ctx context.Context, bin string) string {
	s.mu.Lock()
	v, ok := s.versions[bin]
	s.mu.Unlock()
	if ok {
		return v
	}
	ctx, cancel := context.WithTimeout(ctx, versionTimeout)
	defer cancel()
	out, err := s.cfg.Runner.Run(ctx, ports.Cmd{Name: bin, Args: []string{"version"}})
	if err != nil {
		return ""
	}
	v = strings.TrimPrefix(strings.TrimSpace(out), model.Name+" ")
	s.mu.Lock()
	s.versions[bin] = v
	s.mu.Unlock()
	return v
}

// LoginPath is the PATH a terminal has: what the person's login shell
// says, read once; the app's own PATH when the shell will not say.
func (s *Service) LoginPath(ctx context.Context) []string {
	s.mu.Lock()
	if s.pathRead {
		defer s.mu.Unlock()
		return s.loginPath
	}
	s.mu.Unlock()
	shell := s.cfg.Shell
	if shell == "" {
		shell = "/bin/sh"
	}
	ctx, cancel := context.WithTimeout(ctx, pathTimeout)
	defer cancel()
	// An interactive login shell reads the files a terminal's does; what
	// they print, and how the shell ends, do not matter once it said PATH.
	out, _ := s.cfg.Runner.Run(ctx, ports.Cmd{Name: shell, Args: []string{"-ilc", model.PathProbe}})
	path := s.cfg.PATH
	if v, ok := model.ReadProbe(out); ok {
		path = model.SplitPath(v)
	}
	s.mu.Lock()
	s.loginPath, s.pathRead = path, true
	s.mu.Unlock()
	return path
}

// Install links the app's stickypane as the terminal command at target.
// Something that is not the app's already there is replaced only when
// replace says so (ErrOccupied otherwise); the system folder asks for an
// administrator's password.
func (s *Service) Install(ctx context.Context, t model.Target, replace bool) (CommandState, error) {
	bundled := s.cfg.Bundled()
	if bundled == "" {
		return s.Command(ctx), ErrNoBundled
	}
	l := s.link(t, bundled)
	if l.State == "apps" && same(l.Real, bundled) {
		return s.Command(ctx), nil
	}
	if l.State == "other" && !replace {
		return s.Command(ctx), fmt.Errorf("%w: %s", ErrOccupied, l.Path)
	}
	if err := s.put(ctx, t, bundled, l); err != nil {
		return s.Command(ctx), err
	}
	return s.Command(ctx), nil
}

func (s *Service) put(ctx context.Context, t model.Target, bundled string, l Link) error {
	if t == model.TargetSystem {
		return s.cfg.Admin.Run(ctx, model.LinkScript(bundled, l.Path))
	}
	if err := s.cfg.Files.MkdirAll(filepath.Dir(l.Path)); err != nil {
		return err
	}
	if l.State != "none" {
		if err := s.cfg.Files.Remove(l.Path); err != nil {
			return err
		}
	}
	return s.cfg.Files.Symlink(bundled, l.Path)
}

// Uninstall takes away the links that are the app's, and nothing else.
func (s *Service) Uninstall(ctx context.Context) (CommandState, error) {
	bundled := s.cfg.Bundled()
	var errs []error
	for _, t := range model.Targets {
		l := s.link(t, bundled)
		if l.State != "apps" {
			continue
		}
		var err error
		if t == model.TargetSystem {
			err = s.cfg.Admin.Run(ctx, model.UnlinkScript(l.Path))
		} else {
			err = s.cfg.Files.Remove(l.Path)
		}
		if err != nil {
			errs = append(errs, err)
		}
	}
	return s.Command(ctx), errors.Join(errs...)
}

// cli is the stickypane to run setup and init with: the app's own first.
func (s *Service) cli() (string, error) {
	if b := s.cfg.Bundled(); b != "" {
		return b, nil
	}
	return s.cfg.Find()
}

// mcpCommand is the full path the MCP server is registered with: the
// app's terminal command when it is linked (it stays right when the app
// is updated), else the app's own stickypane, else the one found.
func (s *Service) mcpCommand(cli string) string {
	bundled := s.cfg.Bundled()
	if bundled != "" {
		for _, t := range model.Targets {
			if l := s.link(t, bundled); l.State == "apps" && same(l.Real, bundled) {
				return l.Path
			}
		}
		return bundled
	}
	return cli
}

// runPath is the PATH setup and init run with: a terminal's, then the
// app's, then where agents are usually installed, so that setup finds
// claude and codex though a GUI app's PATH is short.
func (s *Service) runPath(ctx context.Context) string {
	return model.JoinPath(s.LoginPath(ctx), s.cfg.PATH, []string{
		filepath.Join(s.cfg.Home, ".local", "bin"), "/opt/homebrew/bin", "/usr/local/bin", "/usr/bin", "/bin",
	})
}

// Result is what a run printed, and the command it was.
type Result struct {
	Command string `json:"command"`
	Output  string `json:"output"`
}

// Connect runs the stickypane setup for every agent found, at the user's
// scope, in a new empty folder of its own (never the root or the home
// folder: setup makes no board, but an older one did). dry only prints the
// plan. With mcp the MCP server is registered with a full path.
func (s *Service) Connect(ctx context.Context, dry, mcp bool) (Result, error) {
	cli, err := s.cli()
	if err != nil {
		return Result{}, err
	}
	dir, err := s.cfg.Files.TempDir()
	if err != nil {
		return Result{}, err
	}
	if !s.safe(dir) { // checked before anything is run in it, or taken away
		return Result{}, fmt.Errorf("%w: %s", ErrUnsafeDir, dir)
	}
	defer s.cfg.Files.RemoveAll(dir)
	args := model.SetupArgs(dry, mcp, s.mcpCommand(cli))
	res := Result{Command: model.Name + " " + strings.Join(args, " ")}
	res.Output, err = s.cfg.Runner.Run(ctx, ports.Cmd{Dir: dir, Path: s.runPath(ctx), Name: cli, Args: args})
	res.Output = strings.TrimSpace(res.Output)
	return res, err
}

// Init runs stickypane init in dir, a project's folder. lang is the app's
// language ("ko", "en"; "" for the system's): a new board's welcome note
// is written in it, and the board's own setting is left unset, so the
// board goes on following the app.
func (s *Service) Init(ctx context.Context, dir, lang string) (Result, error) {
	if !s.safe(dir) {
		return Result{}, fmt.Errorf("%w: %s", ErrUnsafeDir, dir)
	}
	cli, err := s.cli()
	if err != nil {
		return Result{}, err
	}
	res := Result{Command: model.Name + " init"}
	c := ports.Cmd{Dir: dir, Path: s.runPath(ctx), Name: cli, Args: []string{"init"}}
	if lang != "" {
		c.Env = []string{"STICKYPANE_LANG=" + lang}
	}
	res.Output, err = s.cfg.Runner.Run(ctx, c)
	res.Output = strings.TrimSpace(res.Output)
	if err != nil {
		return res, fmt.Errorf("stickypane init: %v: %s", err, res.Output)
	}
	return res, nil
}

// safe says whether a program may run in dir: not the root, not the home
// folder, either as written or through its symlinks.
func (s *Service) safe(dir string) bool {
	home := s.cfg.Home
	if !model.SafeWorkDir(dir, home) {
		return false
	}
	real := s.cfg.Files.Stat(dir).Real
	if real == "" {
		return true
	}
	hr := s.cfg.Files.Stat(home).Real
	if hr == "" {
		hr = home
	}
	return model.SafeWorkDir(real, hr)
}

func same(a, b string) bool { return a != "" && filepath.Clean(a) == filepath.Clean(b) }
