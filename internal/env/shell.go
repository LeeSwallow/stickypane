package env

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

// Kind is a family of shells that share a syntax.
type Kind int

const (
	Posix      Kind = iota // sh, bash, zsh, fish and the like
	PowerShell             // pwsh and Windows PowerShell
	Cmd                    // the Windows command prompt
)

// Shell is the user's shell.
type Shell struct {
	Name string // "zsh", "pwsh", "powershell", "cmd"
	Kind Kind
}

func detectShell(getenv func(string) string, goos string) Shell {
	if sh := getenv("SHELL"); sh != "" {
		name := strings.TrimSuffix(path.Base(strings.ReplaceAll(sh, `\`, "/")), ".exe")
		if name == "pwsh" || name == "powershell" {
			return Shell{Name: name, Kind: PowerShell}
		}
		return Shell{Name: name, Kind: Posix}
	}
	if goos != "windows" {
		return Shell{Name: "sh", Kind: Posix}
	}
	if inPowerShell(getenv("PSModulePath")) {
		return Shell{Name: "powershell", Kind: PowerShell}
	}
	return Shell{Name: "cmd", Kind: Cmd}
}

// SetVar is the line that sets an environment variable in this shell, for
// messages that tell the user how.
func (s Shell) SetVar(name, value string) string {
	switch s.Kind {
	case PowerShell:
		return powerShellSetVar(name, value)
	case Cmd:
		return fmt.Sprintf("set %s=%s", name, value)
	}
	return fmt.Sprintf("export %s=%s", name, value)
}

// Script returns the program and arguments that run a script file of the
// board, chosen by its extension: sh for .sh, PowerShell for .ps1. A
// missing interpreter is an error that says what to install.
func (e Env) Script(path string) (string, []string, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".sh":
		if _, err := e.lookPath("sh"); err != nil {
			if e.OS == "windows" {
				return "", nil, fmt.Errorf("running %s needs sh, which comes with Git for Windows", filepath.Base(path))
			}
			return "", nil, fmt.Errorf("running %s needs sh: %w", filepath.Base(path), err)
		}
		return "sh", []string{path}, nil
	case ".ps1":
		return e.powerShellScript(path)
	}
	return "", nil, fmt.Errorf("%s is not a script the board runs", filepath.Base(path))
}

// Command returns how this shell runs a command line the user wrote: sh -c
// for POSIX shells, PowerShell's -Command, cmd's /C.
func (s Shell) Command(line string) (string, []string) {
	switch s.Kind {
	case PowerShell:
		return powerShellCommand(s.Name, line)
	case Cmd:
		return "cmd", []string{"/C", line}
	}
	return "sh", []string{"-c", line}
}
