// Package model is the domain of putting the app's own stickypane where a
// terminal and an agent find it (docs/overlay/09 §9.4): where the link
// goes, whether a copy on the PATH is the app's, what shadows it and how
// to fix that, and where setup may run. It does no I/O (ADR-0018).
package model

import (
	"path/filepath"
	"strings"
)

// Name is the command's name.
const Name = "stickypane"

// Target is where the terminal command is linked.
type Target string

const (
	// TargetUser is ~/.local/bin: no password, the person's own folder.
	TargetUser Target = "user"
	// TargetSystem is /usr/local/bin: on every PATH, with an administrator's
	// password, and only when the person picks it.
	TargetSystem Target = "system"
)

// SystemDir is the folder of TargetSystem.
const SystemDir = "/usr/local/bin"

// Dir is the folder a target links into.
func Dir(t Target, home string) string {
	if t == TargetSystem {
		return SystemDir
	}
	return filepath.Join(home, ".local", "bin")
}

// LinkPath is the link a target makes.
func LinkPath(t Target, home string) string { return filepath.Join(Dir(t, home), Name) }

// Targets are the targets, in the order they are offered.
var Targets = []Target{TargetUser, TargetSystem}

// ParseTarget reads a target's name; anything else is TargetUser.
func ParseTarget(s string) Target {
	if Target(s) == TargetSystem {
		return TargetSystem
	}
	return TargetUser
}

// IsApps says whether real (a path with its symlinks followed) is the
// stickypane inside a stickypane app: the bundled one, or the one inside
// another copy of the app (an older one, or one moved). Such a link is the
// app's own to replace or remove.
func IsApps(real, bundled string) bool {
	if real == "" {
		return false
	}
	if bundled != "" && filepath.Clean(real) == filepath.Clean(bundled) {
		return true
	}
	return strings.HasSuffix(filepath.ToSlash(real), ".app/Contents/Helpers/"+Name)
}

// OnPath says whether dir is one of the PATH's folders.
func OnPath(dir string, path []string) bool {
	dir = filepath.Clean(dir)
	for _, p := range path {
		if p != "" && filepath.Clean(p) == dir {
			return true
		}
	}
	return false
}

// SplitPath splits a PATH value, dropping empty parts.
func SplitPath(v string) []string {
	var out []string
	for _, p := range strings.Split(v, ":") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// JoinPath puts folders together as a PATH value, each once, in order.
func JoinPath(parts ...[]string) string {
	seen := map[string]bool{}
	var out []string
	for _, ps := range parts {
		for _, p := range ps {
			p = filepath.Clean(p)
			if p == "" || p == "." || seen[p] {
				continue
			}
			seen[p] = true
			out = append(out, p)
		}
	}
	return strings.Join(out, ":")
}

// Copy is one stickypane on the PATH.
type Copy struct {
	Path    string `json:"path"`              // as the PATH finds it
	Real    string `json:"real"`              // through its symlinks
	Version string `json:"version,omitempty"` // what `stickypane version` said
	Apps    bool   `json:"apps"`              // the app's own (IsApps)
}

// Fix is a way to stop an older copy from coming first on the PATH.
type Fix struct {
	// Kind is "brew" (brew uninstall stickypane), "cask" (brew uninstall
	// --cask stickypane) or "remove" (take the file away); the screen words it.
	Kind string `json:"kind"`
	// Command is what to type, for brew; for remove, the file.
	Command string `json:"command"`
	// Before is the folder to put first on the PATH instead, and After the
	// folder of the copy that shadows it ("" when the link is not made yet).
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
}

// FixFor says how to get past a copy that comes before the app's: remove
// it the way it came (Homebrew's, or by hand), or put linkDir before its
// folder in the PATH.
func FixFor(c Copy, linkDir string) Fix {
	f := Fix{Kind: "remove", Command: "rm " + ShQuote(c.Path), Before: linkDir, After: filepath.Dir(c.Path)}
	real := filepath.ToSlash(c.Real)
	switch {
	case strings.Contains(real, "/Caskroom/"):
		f.Kind, f.Command = "cask", "brew uninstall --cask "+Name
	case strings.Contains(real, "/Cellar/"):
		f.Kind, f.Command = "brew", "brew uninstall "+Name
	}
	return f
}

// RCFile is the file a shell reads at start where a PATH line goes, and the
// line that puts dir first. An unknown shell is given ~/.profile.
func RCFile(shell, home, dir string) (file, line string) {
	switch filepath.Base(shell) {
	case "zsh":
		file = filepath.Join(home, ".zshrc")
	case "bash":
		file = filepath.Join(home, ".bashrc")
	case "fish":
		return filepath.Join(home, ".config", "fish", "config.fish"), "fish_add_path " + ShQuote(dir)
	default:
		file = filepath.Join(home, ".profile")
	}
	shown := dir
	if rel, ok := strings.CutPrefix(dir, home+"/"); ok {
		shown = "$HOME/" + rel
	}
	return file, `export PATH="` + shown + `:$PATH"`
}

// ShQuote quotes s for a POSIX shell.
func ShQuote(s string) string {
	if s != "" && strings.Trim(s, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789/._-+") == "" {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// LinkScript is the shell script that makes the link with an
// administrator's rights: the folder, then a link that replaces what is
// there.
func LinkScript(bundled, link string) string {
	return "mkdir -p " + ShQuote(filepath.Dir(link)) + " && ln -sfn " + ShQuote(bundled) + " " + ShQuote(link)
}

// UnlinkScript removes a link with an administrator's rights.
func UnlinkScript(link string) string { return "rm -f " + ShQuote(link) }

// SafeWorkDir says whether setup may run in dir: never the root or the
// home folder (resolved paths), where an older stickypane would have made
// a board.
func SafeWorkDir(dir, home string) bool {
	d := filepath.Clean(dir)
	if d == "" || d == "." || d == "/" || !filepath.IsAbs(d) {
		return false
	}
	return home == "" || d != filepath.Clean(home)
}

// SetupArgs are the arguments of the bundled stickypane's setup: for every
// agent found, at the user's scope (setup runs in a folder of its own, no
// project's), with the MCP server run by command when mcp is asked for.
// dry asks for the plan only.
func SetupArgs(dry, mcp bool, command string) []string {
	args := []string{"setup", "--scope", "user"}
	if mcp {
		args = append(args, "--mcp")
		if command != "" {
			args = append(args, "--command", command)
		}
	}
	if dry {
		return append(args, "--dry-run")
	}
	return append(args, "--yes")
}

// Marker brackets the PATH a login shell prints, so that what its start
// files print around it is left out.
const Marker = "__stickypane_path__"

// PathProbe is the command a login shell runs to say its PATH.
const PathProbe = `printf '` + Marker + `%s` + Marker + `' "$PATH"`

// ReadProbe takes the PATH out of what the shell printed.
func ReadProbe(out string) (string, bool) {
	_, rest, ok := strings.Cut(out, Marker)
	if !ok {
		return "", false
	}
	v, _, ok := strings.Cut(rest, Marker)
	return v, ok
}
