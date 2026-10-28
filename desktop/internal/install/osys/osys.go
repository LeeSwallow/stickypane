// Package osys is the install ports on the real system: programs run with
// os/exec, files with os, and an administrator's rights through the
// system's own password dialog (osascript on macOS, pkexec on Linux).
package osys

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/desktop/internal/install/ports"
)

// Runner runs programs.
type Runner struct{}

// Run runs c with no input, with c.Path as its PATH when given and c.Env
// set over the app's variables, and returns its output and errors
// together.
func (Runner) Run(ctx context.Context, c ports.Cmd) (string, error) {
	cmd := exec.CommandContext(ctx, c.Name, c.Args...)
	cmd.Dir = c.Dir
	cmd.WaitDelay = time.Second
	if c.Path != "" || len(c.Env) > 0 {
		env := os.Environ()
		if c.Path != "" {
			env = withPath(env, c.Path)
		}
		cmd.Env = append(env, c.Env...)
	}
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil && err != nil {
		err = fmt.Errorf("%s: %w", filepath.Base(c.Name), ctx.Err())
	}
	return string(out), err
}

func withPath(env []string, path string) []string {
	out := make([]string, 0, len(env)+1)
	for _, e := range env {
		if !strings.HasPrefix(e, "PATH=") {
			out = append(out, e)
		}
	}
	return append(out, "PATH="+path)
}

// Files are the file operations.
type Files struct{}

// Stat says what is at p.
func (Files) Stat(p string) ports.Entry {
	fi, err := os.Lstat(p)
	if err != nil {
		return ports.Entry{}
	}
	e := ports.Entry{Exists: true, Link: fi.Mode()&os.ModeSymlink != 0}
	if r, err := filepath.EvalSymlinks(p); err == nil {
		e.Real = r
		if ti, err := os.Stat(r); err == nil {
			e.Exec = ti.Mode().IsRegular() && ti.Mode().Perm()&0o111 != 0
		}
	}
	return e
}

func (Files) MkdirAll(dir string) error         { return os.MkdirAll(dir, 0o755) }
func (Files) Symlink(target, link string) error { return os.Symlink(target, link) }
func (Files) Remove(p string) error             { return os.Remove(p) }
func (Files) TempDir() (string, error)          { return os.MkdirTemp("", "stickypane-setup-") }
func (Files) RemoveAll(dir string) error        { return os.RemoveAll(dir) }

// Admin asks the system for an administrator's rights.
type Admin struct{}

// ErrCancelled: the person closed the password dialog.
var ErrCancelled = errors.New("cancelled")

// Run runs script as an administrator.
func (Admin) Run(ctx context.Context, script string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "/usr/bin/osascript", "-e", AppleScript(script))
	case "linux":
		cmd = exec.CommandContext(ctx, "pkexec", "/bin/sh", "-c", script)
	default:
		return fmt.Errorf("no administrator's dialog on %s", runtime.GOOS)
	}
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	text := strings.TrimSpace(string(out))
	// osascript says -128 when the dialog is cancelled; pkexec exits 126
	if strings.Contains(text, "-128") || cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 126 {
		return ErrCancelled
	}
	return fmt.Errorf("%v: %s", err, text)
}

// AppleScript is the osascript line that runs script with a password.
func AppleScript(script string) string {
	q := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(script)
	return `do shell script "` + q + `" with administrator privileges`
}
