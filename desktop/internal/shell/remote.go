package shell

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LeeSwallow/stickypane/desktop/internal/bridgeclient"
	"github.com/LeeSwallow/stickypane/desktop/internal/remote"
	"github.com/LeeSwallow/stickypane/desktop/internal/sshconfig"
)

// Remote boards (docs/desktop/02 §2.10): a board on another computer opens
// over the person's own ssh, which runs `stickypane bridge` there and
// speaks the same lines on its standard input and output. Nothing listens
// on the network, and the app keeps no password or key: only the host
// list, in overlay.json.

// Host is a computer the app reaches over ssh, and its boards there.
type Host = remote.Host

// How long the ssh calls of the settings may take.
const (
	checkTimeout   = 15 * time.Second
	resolveTimeout = 3 * time.Second
)

// ssh is how the shell runs ssh: Config.SSH, else $STICKYPANE_SSH (a
// check's stand-in) or ssh, with a socket shared by a host's connections.
func (s *Shell) ssh() remote.SSH {
	s.sshOnce.Do(func() {
		if s.cfg.SSH != nil {
			s.sshCfg = *s.cfg.SSH
			return
		}
		s.sshCfg = remote.SSH{Bin: os.Getenv("STICKYPANE_SSH"), ControlPath: controlPath(s.cfg.Home)}
	})
	return s.sshCfg
}

// socketMax is the longest socket path the systems take (104 on macOS),
// less a margin; "cm-" and ssh's %C, a hash of 40 characters, follow the
// folder.
const socketMax = 100

// controlPath is where a host's shared connection keeps its socket: a
// folder of ours, 0700, short enough for a socket's path. "" when there is
// none (Windows, or no folder will do): each board then connects alone.
func controlPath(home string) string {
	if runtime.GOOS == "windows" {
		return ""
	}
	return controlPathIn(
		filepath.Join(os.TempDir(), "stickypane-ssh"),
		filepath.Join(home, "ssh"),
		"/tmp/stickypane-"+strconv.Itoa(os.Getuid()),
	)
}

// controlPathIn is controlPath's choice among dirs, the first that will do.
func controlPathIn(dirs ...string) string {
	for _, d := range dirs {
		if len(d)+len("/cm-")+40 > socketMax {
			continue
		}
		if ownDir(d) {
			return filepath.Join(d, "cm-%C")
		}
	}
	return ""
}

// ownDir makes d (0700) or takes it when it is a real folder of this user;
// another's folder, or a link, is not used.
func ownDir(d string) bool {
	if err := os.MkdirAll(d, 0o700); err != nil {
		return false
	}
	fi, err := os.Lstat(d)
	if err != nil || !fi.IsDir() || fi.Mode()&os.ModeSymlink != 0 {
		return false
	}
	if !mine(fi) {
		return false
	}
	if fi.Mode().Perm() != 0o700 {
		if os.Chmod(d, 0o700) != nil {
			return false
		}
	}
	return true
}

// hostOf is the host of a remote board's key. The caller holds mu.
func (s *Shell) hostOf(key string) (Host, string, bool) {
	name, dir, ok := remote.ParseKey(key)
	if !ok {
		return Host{}, "", false
	}
	for _, h := range s.state.Hosts {
		if h.Name == name {
			return h, dir, true
		}
	}
	return Host{Name: name}, dir, false
}

// hostIndex is where the host named name is in the list, -1 when it is
// not. The caller holds mu.
func (s *Shell) hostIndex(name string) int {
	return slices.IndexFunc(s.state.Hosts, func(x Host) bool { return x.Name == name })
}

// remoteName is the host's name of a remote board's key, "" for a local one.
func remoteName(key string) string {
	name, _, _ := remote.ParseKey(key)
	return name
}

// connectRemote starts the bridge of a board on another computer, through
// ssh. It holds no lock while ssh connects: another board, here or there,
// never waits for it.
func (s *Shell) connectRemote(ctx context.Context, p *project) (Board, error) {
	s.mu.Lock()
	h, dir, ok := s.hostOf(p.path)
	s.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("%w in the app's hosts", noHost(h.Name))
	}
	b, err := s.cfg.StartArgv(ctx, s.ssh().Bridge(h, dir))
	if err != nil {
		return nil, &sshError{said: sshSaid(err), err: err}
	}
	return b, nil
}

// sshError is a remote board's failed start, said in ssh's words; under
// it is the start's own error.
type sshError struct {
	said string
	err  error
}

func (e *sshError) Error() string { return e.said }
func (e *sshError) Unwrap() error { return e.err }

// sshSaid is a failed start in ssh's words: when ssh ended before the
// bridge said hello, what it wrote on stderr is the whole story.
func sshSaid(err error) string {
	var se *bridgeclient.StartError
	if errors.As(err, &se) && errors.Is(se.Err, bridgeclient.ErrConnectionLost) && se.Stderr != "" {
		return se.Stderr
	}
	// An error that has lost its type on the way (a stand-in's for ssh)
	// still reads the same.
	s := err.Error()
	if rest, ok := strings.CutPrefix(s, "bridge: hello: "+bridgeclient.ErrConnectionLost.Error()+": "); ok && rest != "" {
		return rest
	}
	return s
}

// Hosts is the host list.
func (s *Shell) Hosts() []Host {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneHosts(s.state.Hosts)
}

func cloneHosts(hs []Host) []Host {
	out := make([]Host, len(hs))
	for i, h := range hs {
		h.Boards = slices.Clone(h.Boards)
		out[i] = h
	}
	return out
}

// tidyHost is h as it is kept: words trimmed, boards clean and each once.
func tidyHost(h Host) Host {
	h.Name, h.Target, h.Command = strings.TrimSpace(h.Name), strings.TrimSpace(h.Target), strings.TrimSpace(h.Command)
	if h.Command == remote.DefaultCommand {
		h.Command = ""
	}
	var boards []string
	for _, b := range h.Boards {
		b = strings.TrimSpace(b)
		if remote.ValidPath(b) == nil {
			b = remote.CleanPath(b)
		}
		if !slices.Contains(boards, b) {
			boards = append(boards, b)
		}
	}
	h.Boards = boards
	return h
}

// SaveHost adds a host (old "") or changes the host named old. A host
// whose boards are open keeps its name; one with none open is renamed with
// the places of its boards' windows and notes.
func (s *Shell) SaveHost(h Host, old string) error {
	h = tidyHost(h)
	s.mu.Lock()
	i := s.hostIndex(old)
	if old != "" && i < 0 {
		s.mu.Unlock()
		return noHost(old)
	}
	if old != "" && h.Boards == nil {
		h.Boards = slices.Clone(s.state.Hosts[i].Boards)
	}
	if err := h.Check(); err != nil {
		s.mu.Unlock()
		return err
	}
	for j, x := range s.state.Hosts {
		if j != i && x.Name == h.Name {
			s.mu.Unlock()
			return hostTaken(h.Name)
		}
	}
	if old != "" && old != h.Name {
		if len(s.openOf(old)) > 0 {
			s.mu.Unlock()
			return &hostOpenError{old}
		}
		s.rekey(old, h.Name)
	}
	if i < 0 {
		s.state.Hosts = append(s.state.Hosts, h)
	} else {
		s.state.Hosts[i] = h
	}
	s.mu.Unlock()
	return s.hostsChanged()
}

// openOf are the open windows of a host's boards (dir "" for all of them).
// The caller holds mu.
func (s *Shell) openOf(name string, dir ...string) []*project {
	var out []*project
	for key, p := range s.projects {
		n, d, ok := remote.ParseKey(key)
		if !ok || n != name || (len(dir) > 0 && d != remote.CleanPath(dir[0])) {
			continue
		}
		out = append(out, p)
	}
	return out
}

// rekey moves what this computer keeps of a host's boards to its new
// name. The caller holds mu.
func (s *Shell) rekey(from, to string) {
	move := func(key string) (string, bool) {
		n, d, ok := remote.ParseKey(key)
		if !ok || n != from {
			return "", false
		}
		return remote.Key(to, d), true
	}
	for k, v := range s.state.Windows {
		if nk, ok := move(k); ok {
			delete(s.state.Windows, k)
			s.state.Windows[nk] = v
		}
	}
	for k, v := range s.state.Projects {
		if nk, ok := move(k); ok {
			delete(s.state.Projects, k)
			s.state.Projects[nk] = v
		}
	}
	for k, v := range s.state.Boards {
		if nk, ok := move(k); ok {
			delete(s.state.Boards, k)
			s.state.Boards[nk] = v
		}
	}
	if nk, ok := move(s.state.Last); ok {
		s.state.Last = nk
	}
}

// forget drops what this computer keeps of a remote board, or of every
// board of a host (dir ""). The caller holds mu.
func (s *Shell) forget(name, dir string) {
	gone := func(key string) bool {
		n, d, ok := remote.ParseKey(key)
		return ok && n == name && (dir == "" || d == remote.CleanPath(dir))
	}
	for k := range s.state.Windows {
		if gone(k) {
			delete(s.state.Windows, k)
		}
	}
	for k := range s.state.Projects {
		if gone(k) {
			delete(s.state.Projects, k)
		}
	}
	for k := range s.state.Boards {
		if gone(k) {
			delete(s.state.Boards, k)
		}
	}
	if gone(s.state.Last) {
		s.state.Last = ""
	}
}

// RemoveHost takes a host out of the list and closes its boards' windows.
func (s *Shell) RemoveHost(name string) error {
	s.mu.Lock()
	i := s.hostIndex(name)
	if i < 0 {
		s.mu.Unlock()
		return noHost(name)
	}
	s.state.Hosts = slices.Delete(s.state.Hosts, i, i+1)
	open := s.openOf(name)
	s.forget(name, "")
	s.mu.Unlock()
	s.closeAll(open)
	return s.hostsChanged()
}

// AddBoard adds a folder on a host to its boards.
func (s *Shell) AddBoard(name, dir string) error {
	dir = strings.TrimSpace(dir)
	if err := remote.ValidPath(dir); err != nil {
		return err
	}
	dir = remote.CleanPath(dir)
	s.mu.Lock()
	i := s.hostIndex(name)
	if i < 0 {
		s.mu.Unlock()
		return noHost(name)
	}
	if !slices.Contains(s.state.Hosts[i].Boards, dir) {
		s.state.Hosts[i].Boards = append(s.state.Hosts[i].Boards, dir)
	}
	s.mu.Unlock()
	return s.hostsChanged()
}

// RemoveBoard takes a folder out of a host's boards, and closes its window.
func (s *Shell) RemoveBoard(name, dir string) error {
	dir = remote.CleanPath(strings.TrimSpace(dir))
	s.mu.Lock()
	i := s.hostIndex(name)
	if i < 0 {
		s.mu.Unlock()
		return noHost(name)
	}
	s.state.Hosts[i].Boards = slices.DeleteFunc(s.state.Hosts[i].Boards, func(b string) bool { return b == dir })
	open := s.openOf(name, dir)
	s.forget(name, dir)
	s.mu.Unlock()
	s.closeAll(open)
	return s.hostsChanged()
}

// closeAll closes project windows, outside the lock.
func (s *Shell) closeAll(ps []*project) {
	for _, p := range ps {
		s.mu.Lock()
		w := p.win
		s.mu.Unlock()
		if w != nil {
			w.Close()
		}
	}
}

func (s *Shell) hostsChanged() error {
	err := s.save()
	s.cfg.Platform.Emit(HostsEvent, s.Hosts())
	return err
}

// OpenRemote opens the window of a board on a host.
func (s *Shell) OpenRemote(name, dir string) (uint, error) {
	if err := remote.ValidPath(dir); err != nil {
		return 0, err
	}
	s.mu.Lock()
	ok := s.hostIndex(name) >= 0
	s.mu.Unlock()
	if !ok {
		return 0, noHost(name)
	}
	return s.OpenProject(remote.Key(name, dir)), nil
}

// HostCheck is what a host's stickypane said, or why ssh could not ask it.
type HostCheck struct {
	OK      bool   `json:"ok"`                // ssh connected and stickypane answered
	Version string `json:"version,omitempty"` // its version
	Bridge  bool   `json:"bridge"`            // it has `stickypane bridge`
	Error   string `json:"error,omitempty"`   // ssh's words, or the shell's
}

// CheckHost asks a host's stickypane its version and whether it has the
// bridge, over ssh, within 15 seconds.
func (s *Shell) CheckHost(name string) HostCheck {
	s.mu.Lock()
	i := s.hostIndex(name)
	var h Host
	if i >= 0 {
		h = s.state.Hosts[i]
	}
	s.mu.Unlock()
	if i < 0 {
		return HostCheck{Error: noHost(name).Error()}
	}
	ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
	defer cancel()
	out, err := s.cfg.RunArgv(ctx, s.ssh().Probe(h))
	if err != nil {
		return HostCheck{Error: err.Error()}
	}
	p := remote.ReadProbe(out)
	return HostCheck{OK: true, Version: p.Version, Bridge: p.Bridge}
}

// SSHAlias is a host of the person's ssh config, and where ssh would go
// for it (user@hostname:port, from ssh -G), "" when ssh did not say.
type SSHAlias struct {
	Name     string `json:"name"`
	Resolved string `json:"resolved,omitempty"`
}

// SSHAliases are the hosts of the ssh config that are not in the list
// yet, each with where ssh would go. ssh is asked, never the file, for
// that: it knows ProxyJump, Match and the rest.
func (s *Shell) SSHAliases() []SSHAlias {
	names := s.cfg.SSHAliases()
	s.mu.Lock()
	var out []SSHAlias
	for _, n := range names {
		if slices.ContainsFunc(s.state.Hosts, func(h Host) bool { return h.Target == n || h.Name == n }) {
			continue
		}
		out = append(out, SSHAlias{Name: n})
	}
	s.mu.Unlock()
	ssh := s.ssh()
	var wg sync.WaitGroup
	for i := range out {
		wg.Add(1)
		go func(a *SSHAlias) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), resolveTimeout)
			defer cancel()
			if res, err := s.cfg.RunArgv(ctx, ssh.Resolve(a.Name)); err == nil {
				a.Resolved = remote.ReadResolved(res).String()
			}
		}(&out[i])
	}
	wg.Wait()
	return out
}

// defaultAliases reads the person's ssh config.
func defaultAliases() []string {
	home, err := userHomeDir()
	if err != nil {
		return nil
	}
	return sshconfig.Aliases(home)
}
