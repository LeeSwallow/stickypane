package env

import (
	"path"
	"slices"
	"strings"
)

// waitFlags are the flags that make a GUI editor wait until the file is
// closed, by the editor's command name, with the spellings that mean the
// same. Without one, a GUI editor returns at once and the board would take
// the edit for finished. Terminal editors (vim, nvim, hx, nano, emacs -nw,
// micro, kak) run in the terminal and need nothing.
var waitFlags = map[string][]string{
	"code": {"--wait", "-w"}, "code-insiders": {"--wait", "-w"}, "codium": {"--wait", "-w"},
	"cursor": {"--wait", "-w"}, "windsurf": {"--wait", "-w"}, "zed": {"--wait", "-w"},
	"subl": {"--wait", "-w"}, "mate": {"-w", "--wait"},
	"idea": {"--wait"}, "goland": {"--wait"}, "pycharm": {"--wait"}, "webstorm": {"--wait"},
	"phpstorm": {"--wait"}, "rubymine": {"--wait"}, "clion": {"--wait"}, "rider": {"--wait"},
	"rustrover": {"--wait"}, "datagrip": {"--wait"},
	"kate": {"--block", "-b"}, "gvim": {"-f", "--nofork"}, "mvim": {"-f", "--nofork"},
	"open": {"-W", "--wait-apps"},
}

// terminalEditors are tried in order when neither $VISUAL nor $EDITOR
// names one.
var terminalEditors = []string{"nvim", "vim", "vi", "nano"}

// Editor is the user's editor as a command line: $VISUAL or $EDITOR, which
// may carry arguments ("code --wait"), else the first terminal editor found,
// else vi, or Notepad on Windows.
func (e Env) Editor() []string {
	if len(e.editor) > 0 {
		return e.editor
	}
	for _, v := range []string{"VISUAL", "EDITOR"} {
		if parts := strings.Fields(e.getenv(v)); len(parts) > 0 {
			return parts
		}
	}
	if e.OS == "windows" {
		return []string{"notepad"}
	}
	for _, name := range terminalEditors {
		if _, err := e.lookPath(name); err == nil {
			return []string{name}
		}
	}
	return []string{"vi"}
}

// EditorCommand is the command that edits file and returns when the user is
// done with it: the editor, the flag that makes a GUI editor wait unless it
// is given already, then the file.
func (e Env) EditorCommand(file string) []string {
	cmd := append([]string(nil), e.Editor()...)
	name := strings.ToLower(path.Base(strings.ReplaceAll(cmd[0], `\`, "/")))
	for _, ext := range []string{".exe", ".cmd", ".bat"} {
		name = strings.TrimSuffix(name, ext)
	}
	if flags, ok := waitFlags[name]; ok && !slices.ContainsFunc(cmd[1:], func(a string) bool { return slices.Contains(flags, a) }) {
		cmd = append(cmd, flags[0])
	}
	return append(cmd, file)
}

// knownEditors are the editors the settings panel offers when they are
// installed, terminal editors first.
var knownEditors = []string{"nvim", "vim", "hx", "nano", "micro", "emacs", "kak", "code", "cursor", "zed", "subl", "windsurf", "codium", "idea", "kate", "notepad"}

// InstalledEditors are the known editors found on the PATH.
func (e Env) InstalledEditors() []string {
	var out []string
	for _, name := range knownEditors {
		if _, err := e.lookPath(name); err == nil {
			out = append(out, name)
		}
	}
	return out
}

// WithEditor is the environment with the editor the settings chose; "" or
// "auto" keeps the one the environment names.
func (e Env) WithEditor(cmd string) Env {
	if cmd != "" && cmd != "auto" {
		e.editor = strings.Fields(cmd)
	}
	return e
}
