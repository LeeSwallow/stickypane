// Package remote is what a board on another computer is to the app: its
// key among the boards (ssh://<host><path>), and the ssh command lines that
// reach it. Nothing here runs ssh or reads a file; the shell does, with
// these. Signing in is the person's own ssh: its config, its agent, its
// keys. The app stores only the host list (docs/desktop/02 §2.10).
package remote

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"
)

// Scheme starts the key of a remote board.
const Scheme = "ssh://"

// DefaultCommand is the stickypane a host runs when none is named.
const DefaultCommand = "stickypane"

// Host is a computer the app reaches over ssh, and its boards there.
type Host struct {
	// Name is the host's name in the app and in its boards' keys.
	Name string `json:"name"`
	// Target is what ssh is given: an alias of ~/.ssh/config or user@host.
	Target string `json:"target"`
	// Command is the stickypane on the host; "" is DefaultCommand.
	Command string `json:"command,omitempty"`
	// Boards are the folders of its boards, absolute paths on the host.
	Boards []string `json:"boards,omitempty"`
}

var nameRE = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// ValidName says why a host's name cannot be used, or nil.
func ValidName(name string) error {
	if !nameRE.MatchString(name) {
		return fmt.Errorf("host name %q: use letters, digits, '.', '_' or '-'", name)
	}
	return nil
}

// Check says what is wrong with a host, or nil. A target or a command
// that ssh could read as an option, or that spans lines, is refused.
func (h Host) Check() error {
	if err := ValidName(h.Name); err != nil {
		return err
	}
	t := h.Target
	switch {
	case t == "":
		return errors.New("no ssh target")
	case strings.HasPrefix(t, "-"):
		return fmt.Errorf("ssh target %q starts with '-'", t)
	case strings.ContainsAny(t, " \t\r\n'\"\\;&|`$"):
		return fmt.Errorf("ssh target %q: one word, an alias or user@host", t)
	}
	if strings.ContainsAny(h.Command, "\r\n") {
		return errors.New("the command is one line")
	}
	for _, b := range h.Boards {
		if err := ValidPath(b); err != nil {
			return err
		}
	}
	return nil
}

// ValidPath says why a board's folder cannot be used, or nil.
func ValidPath(p string) error {
	switch {
	case !strings.HasPrefix(p, "/"):
		return fmt.Errorf("board folder %q: an absolute path on the host, from /", p)
	case strings.ContainsAny(p, "\x00\r\n"):
		return fmt.Errorf("board folder %q is one line", p)
	}
	return nil
}

// CleanPath is a board's folder as the app keeps it: absolute, clean.
func CleanPath(p string) string { return path.Clean(p) }

// Cmd is the host's stickypane.
func (h Host) Cmd() string {
	if c := strings.TrimSpace(h.Command); c != "" {
		return c
	}
	return DefaultCommand
}

// Key is the board's key among the app's boards: ssh://<name><path>.
func Key(name, dir string) string { return Scheme + name + CleanPath(dir) }

// IsRemote says whether a key is a remote board's.
func IsRemote(key string) bool { return strings.HasPrefix(key, Scheme) }

// ParseKey splits a remote board's key into its host's name and folder.
func ParseKey(key string) (name, dir string, ok bool) {
	rest, ok := strings.CutPrefix(key, Scheme)
	if !ok {
		return "", "", false
	}
	i := strings.IndexByte(rest, '/')
	if i <= 0 {
		return "", "", false
	}
	name, dir = rest[:i], rest[i:]
	if ValidName(name) != nil {
		return "", "", false
	}
	return name, CleanPath(dir), true
}

// Quote is s in single quotes for a POSIX shell. A single quote inside
// ends the quoting, is escaped with a backslash, and the quoting begins again.
func Quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// SSH is how the app runs ssh: the program, and the socket a host's
// connections share ("" shares none).
type SSH struct {
	Bin         string // "ssh", or $STICKYPANE_SSH
	ControlPath string // e.g. /tmp/stickypane-501/cm-%C
}

// options are the ssh flags of every connection: no terminal, no
// questions (a password prompt would hang a window), a bounded wait, and
// one shared connection per host.
func (s SSH) options() []string {
	o := []string{"-T", "-o", "BatchMode=yes", "-o", "ConnectTimeout=10"}
	if s.ControlPath != "" {
		o = append(o, "-o", "ControlMaster=auto", "-o", "ControlPath="+s.ControlPath, "-o", "ControlPersist=60")
	}
	return o
}

func (s SSH) bin() string {
	if s.Bin == "" {
		return "ssh"
	}
	return s.Bin
}

// Run is the argv that runs a shell line on the host.
func (s SSH) Run(target, line string) []string {
	argv := append([]string{s.bin()}, s.options()...)
	return append(argv, target, "--", line)
}

// Bridge is the argv of the board's bridge on the host:
// cd '<dir>' && <command> bridge.
func (s SSH) Bridge(h Host, dir string) []string {
	return s.Run(h.Target, "cd "+Quote(CleanPath(dir))+" && "+h.Cmd()+" bridge")
}

// Probe is the argv that asks the host's stickypane its version, then its
// full help (whose usage lines name `stickypane bridge` when it has one).
// An old copy's help may fail; its words still count.
func (s SSH) Probe(h Host) []string {
	c := h.Cmd()
	return s.Run(h.Target, c+" version && { "+c+" help --all; true; }")
}

// Resolve is the argv that prints ssh's whole config for an alias without
// connecting (ssh -G).
func (s SSH) Resolve(alias string) []string {
	return []string{s.bin(), "-G", alias}
}

// Probed is what a host's stickypane said of itself.
type Probed struct {
	Version string `json:"version"`
	Bridge  bool   `json:"bridge"`
}

// ReadProbe reads what Probe's command printed.
func ReadProbe(out string) Probed {
	first, rest, _ := strings.Cut(strings.TrimLeft(out, "\r\n"), "\n")
	return Probed{
		Version: strings.TrimPrefix(strings.TrimSpace(first), "stickypane "),
		Bridge:  strings.Contains(rest, "stickypane bridge"),
	}
}

// Resolved is where ssh would go for an alias.
type Resolved struct {
	User     string `json:"user,omitempty"`
	Hostname string `json:"hostname,omitempty"`
	Port     string `json:"port,omitempty"`
}

// ReadResolved reads ssh -G's output: one "keyword value" a line.
func ReadResolved(out string) Resolved {
	var r Resolved
	for _, line := range strings.Split(out, "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), " ")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		switch strings.ToLower(k) {
		case "user":
			r.User = v
		case "hostname":
			r.Hostname = v
		case "port":
			r.Port = v
		}
	}
	return r
}

// String is user@hostname:port, leaving out what is not known.
func (r Resolved) String() string {
	s := r.Hostname
	if r.User != "" && s != "" {
		s = r.User + "@" + s
	}
	if r.Port != "" && s != "" {
		s += ":" + r.Port
	}
	return s
}
