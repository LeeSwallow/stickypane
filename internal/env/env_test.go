package env

import (
	"errors"
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
	if got := detect(vars(), "darwin", func() string { return "" }).Editor(); got[0] != "vi" {
		t.Errorf("unix = %q", got)
	}
}
