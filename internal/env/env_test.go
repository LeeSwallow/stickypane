package env

import (
	"errors"
	"strings"
	"testing"
)

func vars(kv ...string) func(string) string {
	m := map[string]string{}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return func(k string) string { return m[k] }
}

func TestThePaneToolIsRecognized(t *testing.T) {
	for _, c := range []struct {
		vars  []string
		goos  string
		want  string
		split string
	}{
		{[]string{"TMUX", "/tmp/tmux-501/default,1,0", "WEZTERM_PANE", "3"}, "darwin", "tmux", "tmux split-window -h stickypane"},
		{[]string{"ZELLIJ", "0"}, "linux", "zellij", "zellij run --direction right -- stickypane"},
		{[]string{"WEZTERM_PANE", "3"}, "linux", "wezterm", "wezterm cli split-pane --right -- stickypane"},
		{[]string{"TERM_PROGRAM", "WezTerm"}, "darwin", "wezterm", "wezterm cli split-pane --right -- stickypane"},
		{[]string{"WT_SESSION", "x"}, "windows", "windows-terminal", "wt -w 0 split-pane -V stickypane"},
		{nil, "linux", "", ""},
	} {
		e := detect(vars(c.vars...), c.goos, func() string { return "" })
		if e.Pane.Name != c.want || e.Pane.Split("stickypane") != c.split {
			t.Errorf("%v: pane = %q, split = %q", c.vars, e.Pane.Name, e.Pane.Split("stickypane"))
		}
	}
}

func TestTheShellIsRecognized(t *testing.T) {
	for _, c := range []struct {
		vars []string
		goos string
		want Kind
		name string
	}{
		{[]string{"SHELL", "/bin/zsh"}, "darwin", Posix, "zsh"},
		{[]string{"SHELL", "/usr/bin/fish"}, "linux", Posix, "fish"},
		{[]string{"SHELL", "/usr/bin/pwsh"}, "linux", PowerShell, "pwsh"},
		{[]string{"PSModulePath", `C:\Users\min\Documents\PowerShell\Modules;C:\Program Files\PowerShell\Modules`}, "windows", PowerShell, "powershell"},
		{[]string{"PSModulePath", `C:\Program Files\WindowsPowerShell\Modules`, "ComSpec", `C:\Windows\system32\cmd.exe`}, "windows", Cmd, "cmd"},
		{[]string{"SHELL", "/usr/bin/bash", "PSModulePath", `C:\x`}, "windows", Posix, "bash"},
	} {
		e := detect(vars(c.vars...), c.goos, func() string { return "" })
		if e.Shell.Kind != c.want || e.Shell.Name != c.name {
			t.Errorf("%v on %s: shell = %+v", c.vars, c.goos, e.Shell)
		}
	}
}

func TestTheLocaleComesFromTheEnvironmentThenTheSystem(t *testing.T) {
	system := func() string { return "ko-KR" }
	if e := detect(vars("LANG", "C", "LC_ALL", "ja_JP.UTF-8"), "linux", system); e.Locale != "ja_JP.UTF-8" || e.LocaleFrom != "LC_ALL" {
		t.Errorf("LC_ALL wins: %q from %q", e.Locale, e.LocaleFrom)
	}
	if e := detect(vars("STICKYPANE_LANG", "en", "LANG", "ko_KR.UTF-8"), "linux", system); e.Locale != "en" {
		t.Errorf("STICKYPANE_LANG wins: %q", e.Locale)
	}
	if e := detect(vars("LANG", "C"), "windows", system); e.Locale != "ko-KR" || e.LocaleFrom != "system" {
		t.Errorf("without a usable variable the system says: %q from %q", e.Locale, e.LocaleFrom)
	}
}

func TestScriptsRunWithTheirInterpreter(t *testing.T) {
	look := func(have ...string) func(string) (string, error) {
		return func(name string) (string, error) {
			for _, h := range have {
				if h == name {
					return "/bin/" + name, nil
				}
			}
			return "", errors.New("not found")
		}
	}
	e := detect(vars("SHELL", "/bin/zsh"), "darwin", func() string { return "" })
	e.lookPath = look("sh", "pwsh")
	if name, args, err := e.Script("deploy.sh"); err != nil || name != "sh" || args[0] != "deploy.sh" {
		t.Errorf(".sh = %q %q %v", name, args, err)
	}
	if name, args, err := e.Script("deploy.ps1"); err != nil || name != "pwsh" || args[len(args)-1] != "deploy.ps1" {
		t.Errorf(".ps1 = %q %q %v", name, args, err)
	}
	w := detect(vars("PSModulePath", `C:\Users\a\Documents\WindowsPowerShell\Modules`), "windows", func() string { return "" })
	w.lookPath = look("powershell")
	if name, _, err := w.Script("deploy.ps1"); err != nil || name != "powershell" {
		t.Errorf("Windows PowerShell when pwsh is missing: %q %v", name, err)
	}
	if _, _, err := w.Script("deploy.sh"); err == nil {
		t.Error("a .sh on Windows without sh says what is missing")
	}
}

func TestSettingAVariableIsSaidInTheUsersShell(t *testing.T) {
	for kind, want := range map[Kind]string{
		Posix:      `export STICKYPANE_LANG=ko`,
		PowerShell: `$env:STICKYPANE_LANG = "ko"`,
		Cmd:        `set STICKYPANE_LANG=ko`,
	} {
		if got := (Shell{Kind: kind}).SetVar("STICKYPANE_LANG", "ko"); got != want {
			t.Errorf("%v: %q", kind, got)
		}
	}
}

func TestTheEditorFallsBackPerSystem(t *testing.T) {
	if got := detect(vars("EDITOR", "code --wait"), "linux", func() string { return "" }).Editor(); len(got) != 2 || got[0] != "code" {
		t.Errorf("EDITOR = %q", got)
	}
	if got := detect(vars(), "windows", func() string { return "" }).Editor(); got[0] != "notepad" {
		t.Errorf("windows = %q", got)
	}
	none := detect(vars(), "darwin", func() string { return "" })
	none.lookPath = func(string) (string, error) { return "", errors.New("not found") }
	if got := none.Editor(); got[0] != "vi" {
		t.Errorf("unix = %q", got)
	}
}

// A command line given by the user runs in their shell's own way.
func TestACommandRunsInTheUsersShell(t *testing.T) {
	for _, c := range []struct {
		shell Shell
		name  string
		args  string
	}{
		{Shell{Name: "zsh", Kind: Posix}, "sh", "-c|echo hi"},
		{Shell{Name: "pwsh", Kind: PowerShell}, "pwsh", "-NoProfile|-Command|echo hi"},
		{Shell{Name: "cmd", Kind: Cmd}, "cmd", "/C|echo hi"},
	} {
		name, args := c.shell.Command("echo hi")
		if name != c.name || strings.Join(args, "|") != c.args {
			t.Errorf("%s: %s %q", c.shell.Name, name, args)
		}
	}
}

// Any editor works for E: a GUI editor gets the flag that makes it wait
// until the file is closed, a terminal editor runs as it is, and without
// $VISUAL or $EDITOR the first editor found is used.
func TestEveryEditorOpensAndWaits(t *testing.T) {
	look := func(have ...string) func(string) (string, error) {
		return func(name string) (string, error) {
			for _, h := range have {
				if h == name {
					return "/bin/" + name, nil
				}
			}
			return "", errors.New("not found")
		}
	}
	for _, c := range []struct{ editor, want string }{
		{"code", "code --wait plan.md"},
		{"code --wait", "code --wait plan.md"},
		{"/usr/local/bin/cursor", "/usr/local/bin/cursor --wait plan.md"},
		{"zed", "zed --wait plan.md"},
		{"subl", "subl --wait plan.md"},
		{"idea", "idea --wait plan.md"},
		{"kate", "kate --block plan.md"},
		{"mate", "mate -w plan.md"},
		{"open -a TextEdit", "open -a TextEdit -W plan.md"},
		{"nvim", "nvim plan.md"},
		{"vim -u NONE", "vim -u NONE plan.md"},
		{"hx", "hx plan.md"},
		{"emacs -nw", "emacs -nw plan.md"},
		{"nano", "nano plan.md"},
		{"micro", "micro plan.md"},
	} {
		e := detect(vars("EDITOR", c.editor), "darwin", func() string { return "" })
		if got := strings.Join(e.EditorCommand("plan.md"), " "); got != c.want {
			t.Errorf("EDITOR=%q: %q, want %q", c.editor, got, c.want)
		}
	}
	e := detect(vars(), "linux", func() string { return "" })
	e.lookPath = look("vim", "nano")
	if got := strings.Join(e.EditorCommand("a.md"), " "); got != "vim a.md" {
		t.Errorf("without EDITOR the first editor found: %q", got)
	}
	e.lookPath = look()
	if got := strings.Join(e.EditorCommand("a.md"), " "); got != "vi a.md" {
		t.Errorf("with nothing found, vi: %q", got)
	}
	w := detect(vars(), "windows", func() string { return "" })
	w.lookPath = look("code")
	if got := strings.Join(w.EditorCommand("a.md"), " "); got != "notepad a.md" {
		t.Errorf("on Windows without EDITOR, notepad: %q", got)
	}
}
