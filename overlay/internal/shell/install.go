package shell

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/LeeSwallow/stickypane/overlay/internal/bridge"
	"github.com/LeeSwallow/stickypane/overlay/internal/install/model"
	"github.com/LeeSwallow/stickypane/overlay/internal/install/osys"
	installsvc "github.com/LeeSwallow/stickypane/overlay/internal/install/service"
)

// defaultInstaller is the install use cases on this machine: the app's own
// stickypane, the person's home folder and login shell.
func defaultInstaller(find func() (string, error)) Installer {
	home, _ := os.UserHomeDir()
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}
	return installsvc.New(installsvc.Config{
		Files:  osys.Files{},
		Runner: osys.Runner{},
		Admin:  osys.Admin{},
		Home:   home,
		Shell:  shell,
		PATH:   filepath.SplitList(os.Getenv("PATH")),
		Bundled: func() string {
			if p, ok := bridge.BundledCLI(); ok {
				return p
			}
			return ""
		},
		Find: find,
	})
}

// How long each kind of run may take: a password dialog waits for the
// person, and setup for the agents' own installs.
const (
	installTimeout = 3 * time.Minute
	connectTimeout = 5 * time.Minute
)

// InstallAnswer is what the settings get back from installing or removing
// the terminal command.
type InstallAnswer struct {
	State installsvc.CommandState `json:"state"`
	// Occupied names a program that is not the app's where the link would
	// go: the settings ask before it is replaced.
	Occupied string `json:"occupied,omitempty"`
	// Cancelled: the person closed the password dialog.
	Cancelled bool `json:"cancelled,omitempty"`
}

func answerOf(st installsvc.CommandState, err error, at string) (InstallAnswer, error) {
	a := InstallAnswer{State: st}
	switch {
	case errors.Is(err, installsvc.ErrOccupied):
		a.Occupied = at
	case errors.Is(err, osys.ErrCancelled):
		a.Cancelled = true
	case err != nil:
		return a, err
	}
	return a, nil
}

// SetupService is what the settings call to put stickypane on the PATH
// and connect the agents (docs/overlay/09 §9.4); any window may call it.
type SetupService struct{ S *Shell }

// Command says where the terminal command stands.
func (x *SetupService) Command() installsvc.CommandState {
	ctx, cancel := context.WithTimeout(context.Background(), installTimeout)
	defer cancel()
	return x.S.cfg.Install.Command(ctx)
}

// InstallCommand links the app's stickypane at target ("user":
// ~/.local/bin, "system": /usr/local/bin with a password). replace says
// the person agreed to replace a program that is not the app's.
func (x *SetupService) InstallCommand(target string, replace bool) (InstallAnswer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), installTimeout)
	defer cancel()
	t := model.ParseTarget(target)
	st, err := x.S.cfg.Install.Install(ctx, t, replace)
	at := ""
	for _, l := range st.Links {
		if l.Target == t {
			at = l.Path
		}
	}
	return answerOf(st, err, at)
}

// UninstallCommand takes away the app's links, and nothing else.
func (x *SetupService) UninstallCommand() (InstallAnswer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), installTimeout)
	defer cancel()
	st, err := x.S.cfg.Install.Uninstall(ctx)
	return answerOf(st, err, "")
}

// ConnectPlan is what connecting the agents would run (setup --dry-run).
func (x *SetupService) ConnectPlan(mcp bool) (installsvc.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()
	return x.S.cfg.Install.Connect(ctx, true, mcp)
}

// Connect runs it.
func (x *SetupService) Connect(mcp bool) (installsvc.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()
	return x.S.cfg.Install.Connect(ctx, false, mcp)
}
