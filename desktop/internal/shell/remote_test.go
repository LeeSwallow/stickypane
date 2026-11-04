package shell

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/LeeSwallow/stickypane/desktop/internal/bridgeclient"
	"github.com/LeeSwallow/stickypane/desktop/internal/remote"
)

// sshFake stands in for ssh: it records each command line and answers
// as the test says.
type sshFake struct {
	mu    sync.Mutex
	argvs [][]string
	start func(argv []string) (Board, error)
	run   func(argv []string) (string, error)
}

func (f *sshFake) record(argv []string) {
	f.mu.Lock()
	f.argvs = append(f.argvs, append([]string(nil), argv...))
	f.mu.Unlock()
}

func (f *sshFake) seen() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][]string(nil), f.argvs...)
}

func remoteRig(t *testing.T, f *sshFake, hosts ...Host) *rig {
	t.Helper()
	r := newRig(t, func(c *Config) {
		c.SSH = &remote.SSH{Bin: "/fake/ssh", ControlPath: "/tmp/sp/cm-%C"}
		c.StartArgv = func(ctx context.Context, argv []string) (Board, error) {
			f.record(argv)
			if f.start != nil {
				return f.start(argv)
			}
			return nil, errors.New("no bridge in this test")
		}
		c.RunArgv = func(ctx context.Context, argv []string) (string, error) {
			f.record(argv)
			if f.run != nil {
				return f.run(argv)
			}
			return "", errors.New("no ssh in this test")
		}
		c.SSHAliases = func() []string { return []string{"dev", "box", "work"} }
		c.Find = func() (string, error) {
			t.Error("a remote board does not look for this computer's stickypane")
			return "", errors.New("no")
		}
	})
	for _, h := range hosts {
		if err := r.s.SaveHost(h, ""); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

var devHost = Host{Name: "dev", Target: "min@dev.lan", Boards: []string{"/home/min/my app"}}

func TestOpenRemoteRunsTheBridgeOverSSH(t *testing.T) {
	f := &sshFake{}
	r := remoteRig(t, f, devHost)
	f.start = func([]string) (Board, error) { return r.board, nil }

	id, err := r.s.OpenRemote("dev", "/home/min/my app/")
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, "ready", func() bool { return r.s.Status(id).State == StateReady })
	want := []string{"/fake/ssh", "-T", "-o", "BatchMode=yes", "-o", "ConnectTimeout=10",
		"-o", "ControlMaster=auto", "-o", "ControlPath=/tmp/sp/cm-%C", "-o", "ControlPersist=60",
		"min@dev.lan", "--", "cd '/home/min/my app' && stickypane bridge"}
	if got := f.seen(); len(got) != 1 || !reflect.DeepEqual(got[0], want) {
		t.Fatalf("ssh ran %q, want %q", got, want)
	}
	st := r.s.Status(id)
	if st.Path != "ssh://dev/home/min/my app" || st.Remote != "dev" {
		t.Errorf("status = %+v", st)
	}
	w := r.p.projectWindows()[0]
	if w.opts.Title != "my app — dev" {
		t.Errorf("title %q", w.opts.Title)
	}
	// the same board again is the same window
	if again, _ := r.s.OpenRemote("dev", "/home/min/my app"); again != id {
		t.Errorf("second open made window %d, want %d", again, id)
	}
	if _, err := r.s.OpenRemote("nobody", "/x"); err == nil {
		t.Error("a host not in the list does not open")
	}
}

func TestARemoteBoardThatCannotConnectSaysWhatSSHSaid(t *testing.T) {
	f := &sshFake{start: func([]string) (Board, error) {
		return nil, errors.New("bridge: hello: bridge: connection lost: ssh: Could not resolve hostname dev.lan")
	}}
	r := remoteRig(t, f, devHost)
	id, _ := r.s.OpenRemote("dev", "/home/min/my app")
	waitFor(t, "failed", func() bool { return r.s.Status(id).State == StateFailed })
	if e := r.s.Status(id).Error; e != "ssh: Could not resolve hostname dev.lan" {
		t.Errorf("error %q", e)
	}
}

// The client's own *StartError reads the same: ssh's words, and nothing
// of the bridge's.
func TestARemoteStartErrorSaysWhatSSHSaid(t *testing.T) {
	f := &sshFake{start: func([]string) (Board, error) {
		return nil, &bridgeclient.StartError{Err: bridgeclient.ErrConnectionLost, Stderr: "ssh: connect to host dev.lan port 22: Connection refused"}
	}}
	r := remoteRig(t, f, devHost)
	id, _ := r.s.OpenRemote("dev", "/home/min/my app")
	waitFor(t, "failed", func() bool { return r.s.Status(id).State == StateFailed })
	if e := r.s.Status(id).Error; e != "ssh: connect to host dev.lan port 22: Connection refused" {
		t.Errorf("error %q", e)
	}
	// a start that failed for another reason says it whole
	other := &bridgeclient.StartError{Err: bridgeclient.ErrProtocol, Stderr: "warning"}
	if got := sshSaid(other); got != other.Error() {
		t.Errorf("sshSaid(%v) = %q", other, got)
	}
}

// A remote board that is slow to connect keeps no other board waiting.
func TestALocalBoardDoesNotWaitForARemoteOne(t *testing.T) {
	release := make(chan struct{})
	f := &sshFake{}
	r := remoteRig(t, f, devHost)
	f.start = func([]string) (Board, error) { <-release; return nil, errors.New("late") }
	defer close(release)
	r.s.OpenRemote("dev", "/home/min/my app")
	r.s.cfg.Find = func() (string, error) { return "/bin/stickypane", nil }
	id := r.s.OpenProject(filepath.Join(r.home, "here"))
	waitFor(t, "the local board ready", func() bool { return r.s.Status(id).State == StateReady })
	if hs := r.s.Hosts(); len(hs) != 1 { // and the shell answers meanwhile
		t.Errorf("hosts %v", hs)
	}
}

func TestARemoteBoardRefusesTheEditorAndInit(t *testing.T) {
	f := &sshFake{}
	r := remoteRig(t, f, devHost)
	f.start = func([]string) (Board, error) { return r.board, nil }
	r.s.Start()
	id, _ := r.s.OpenRemote("dev", "/home/min/my app")
	waitFor(t, "ready", func() bool { return r.s.Status(id).State == StateReady })
	var res json.RawMessage
	if err := r.s.Call(context.Background(), id, "note.open", noteParams{"a.md"}, &res); !errors.Is(err, ErrRemote) {
		t.Errorf("note.open: %v, want ErrRemote", err)
	}
	if _, err := r.s.InitHere(id); !errors.Is(err, ErrRemote) {
		t.Errorf("InitHere: %v", err)
	}
	if err := r.s.MakeBoardIn(id, "ko"); !errors.Is(err, ErrRemote) {
		t.Errorf("MakeBoardIn: %v", err)
	}
	r.inst.mu.Lock()
	inits := len(r.inst.inits)
	r.inst.mu.Unlock()
	if inits != 0 {
		t.Error("init ran for a remote board")
	}

	// board.load answers, but the index is this computer's boards only
	if _, err := r.s.Sessions(); err != nil {
		t.Fatal(err)
	}
	if err := r.s.Call(context.Background(), id, "board.load", nil, &res); err != nil {
		t.Fatal(err)
	}
	r.ix.mu.Lock()
	defer r.ix.mu.Unlock()
	if len(r.ix.titles) != 0 {
		t.Errorf("a remote board was named in the index: %v", r.ix.titles)
	}
}

func TestRemoveHostClosesItsBoards(t *testing.T) {
	f := &sshFake{}
	other := Host{Name: "box", Target: "box", Boards: []string{"/srv"}}
	r := remoteRig(t, f, devHost, other)
	var mu sync.Mutex
	var boards []*fakeBoard
	f.start = func([]string) (Board, error) {
		b := newFakeBoard()
		mu.Lock()
		boards = append(boards, b)
		mu.Unlock()
		return b, nil
	}
	a, _ := r.s.OpenRemote("dev", "/home/min/my app")
	b, _ := r.s.OpenRemote("box", "/srv")
	waitFor(t, "both ready", func() bool {
		return r.s.Status(a).State == StateReady && r.s.Status(b).State == StateReady
	})
	if err := r.s.RemoveHost("dev"); err != nil {
		t.Fatal(err)
	}
	if r.s.Status(a).State != StateClosed {
		t.Error("the removed host's board is still open")
	}
	if r.s.Status(b).State != StateReady {
		t.Error("another host's board was closed")
	}
	if hs := r.s.Hosts(); len(hs) != 1 || hs[0].Name != "box" {
		t.Errorf("hosts %+v", hs)
	}
	if err := r.s.RemoveBoard("box", "/srv"); err != nil {
		t.Fatal(err)
	}
	if r.s.Status(b).State != StateClosed {
		t.Error("a board taken out of the list stays open")
	}
}

func TestSaveHost(t *testing.T) {
	r := remoteRig(t, &sshFake{})
	for _, bad := range []Host{
		{Name: "a b", Target: "dev"},
		{Name: "dev", Target: "-oProxyCommand=x"},
		{Name: "dev", Target: ""},
	} {
		if err := r.s.SaveHost(bad, ""); err == nil {
			t.Errorf("%+v saved", bad)
		}
	}
	if err := r.s.SaveHost(Host{Name: " dev ", Target: "dev", Command: "stickypane"}, ""); err != nil {
		t.Fatal(err)
	}
	if err := r.s.SaveHost(Host{Name: "dev", Target: "other"}, ""); err == nil {
		t.Error("a second host of the same name")
	}
	if err := r.s.AddBoard("dev", "relative"); err == nil {
		t.Error("a relative folder was taken")
	}
	r.s.AddBoard("dev", "/srv/app/")
	r.s.AddBoard("dev", "/srv/app")
	if hs := r.s.Hosts(); len(hs) != 1 || hs[0].Name != "dev" || hs[0].Command != "" || !reflect.DeepEqual(hs[0].Boards, []string{"/srv/app"}) {
		t.Fatalf("hosts %+v", hs)
	}
	if len(r.p.emitted(HostsEvent)) == 0 {
		t.Error("no window was told")
	}

	// renaming moves the places of its boards, and keeps its boards
	r.s.mu.Lock()
	r.s.state.Windows = map[string]WindowState{"ssh://dev/srv/app": {Width: 500, Height: 400}, "/local": {Width: 1}}
	r.s.mu.Unlock()
	if err := r.s.SaveHost(Host{Name: "devbox", Target: "dev", Command: "~/bin/stickypane"}, "dev"); err != nil {
		t.Fatal(err)
	}
	hs := r.s.Hosts()
	if hs[0].Name != "devbox" || hs[0].Command != "~/bin/stickypane" || !reflect.DeepEqual(hs[0].Boards, []string{"/srv/app"}) {
		t.Fatalf("hosts %+v", hs)
	}
	r.s.mu.Lock()
	_, moved := r.s.state.Windows["ssh://devbox/srv/app"]
	_, local := r.s.state.Windows["/local"]
	r.s.mu.Unlock()
	if !moved || !local {
		t.Error("the board's window place did not follow the new name")
	}

	// a host with a board open keeps its name
	f := &sshFake{}
	r2 := remoteRig(t, f, devHost)
	f.start = func([]string) (Board, error) { return r2.board, nil }
	id, _ := r2.s.OpenRemote("dev", "/home/min/my app")
	waitFor(t, "ready", func() bool { return r2.s.Status(id).State == StateReady })
	if err := r2.s.SaveHost(Host{Name: "dev2", Target: "dev"}, "dev"); err == nil {
		t.Error("renamed with a board open")
	}
}

func TestHostsRoundTripInOverlayJSON(t *testing.T) {
	r := remoteRig(t, &sshFake{}, devHost, Host{Name: "box", Target: "box", Command: "/opt/sp/stickypane"})
	b, err := os.ReadFile(filepath.Join(r.home, StateFile))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"hosts"`) || !strings.Contains(string(b), `"target": "min@dev.lan"`) {
		t.Fatalf("overlay.json:\n%s", b)
	}
	st, err := LoadState(filepath.Join(r.home, StateFile))
	if err != nil {
		t.Fatal(err)
	}
	want := []Host{devHost, {Name: "box", Target: "box", Command: "/opt/sp/stickypane"}}
	if !reflect.DeepEqual(st.Hosts, want) {
		t.Errorf("hosts read back %+v, want %+v", st.Hosts, want)
	}
	if strings.Contains(string(b), "password") || strings.Contains(string(b), "identity") {
		t.Error("something besides the host list was kept")
	}
}

func TestCheckHost(t *testing.T) {
	f := &sshFake{run: func(argv []string) (string, error) {
		return "stickypane 0.9.0\nUsage:\n  stickypane bridge  serve the board\n", nil
	}}
	r := remoteRig(t, f, devHost)
	c := r.s.CheckHost("dev")
	if !c.OK || c.Version != "0.9.0" || !c.Bridge {
		t.Fatalf("check %+v", c)
	}
	last := f.seen()[0]
	if last[len(last)-1] != "stickypane version && { stickypane help --all; true; }" || last[len(last)-3] != "min@dev.lan" {
		t.Errorf("ssh ran %q", last)
	}
	f.run = func([]string) (string, error) { return "stickypane 0.3.0\nUsage:\n  stickypane list\n", nil }
	if c := r.s.CheckHost("dev"); !c.OK || c.Bridge {
		t.Errorf("an old stickypane: %+v", c)
	}
	f.run = func([]string) (string, error) { return "", errors.New("min@dev.lan: Permission denied (publickey).") }
	if c := r.s.CheckHost("dev"); c.OK || c.Error != "min@dev.lan: Permission denied (publickey)." {
		t.Errorf("a failed check: %+v", c)
	}
	if c := r.s.CheckHost("nobody"); c.OK || c.Error == "" {
		t.Errorf("no such host: %+v", c)
	}
}

func TestSSHAliasesLeaveOutHostsInTheList(t *testing.T) {
	f := &sshFake{run: func(argv []string) (string, error) {
		if argv[1] != "-G" {
			t.Errorf("ssh ran %q", argv)
		}
		if argv[2] == "work" {
			return "", errors.New("late")
		}
		return "user min\nhostname " + argv[2] + ".lan\nport 22\n", nil
	}}
	r := remoteRig(t, f, Host{Name: "mine", Target: "dev"})
	got := r.s.SSHAliases()
	want := []SSHAlias{{Name: "box", Resolved: "min@box.lan:22"}, {Name: "work"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("aliases %+v, want %+v", got, want)
	}
}

// A second launch with --project=ssh://... opens the remote board, not a
// folder of that name under the launch's folder.
func TestLaunchOpensARemoteKey(t *testing.T) {
	f := &sshFake{}
	r := remoteRig(t, f, devHost)
	f.start = func([]string) (Board, error) { return r.board, nil }
	r.s.Launch([]string{"app", "--project=ssh://dev/home/min/my app"}, "/Users/min")
	waitFor(t, "a window", func() bool { return len(r.p.projectWindows()) == 1 })
	id := r.p.projectWindows()[0].ID()
	waitFor(t, "ready", func() bool { return r.s.Status(id).State == StateReady })
	if p := r.s.Status(id).Path; p != "ssh://dev/home/min/my app" {
		t.Errorf("opened %q", p)
	}
}

func TestCleanPathLeavesRemoteKeysAlone(t *testing.T) {
	if got := cleanPath("ssh://dev/srv/app/"); got != "ssh://dev/srv/app" {
		t.Errorf("cleanPath = %q", got)
	}
}

func TestControlPathIsShortAndPrivate(t *testing.T) {
	short := filepath.Join(t.TempDir(), "s")
	long := filepath.Join(t.TempDir(), strings.Repeat("x", 80))
	if len(short)+44 > socketMax {
		t.Skip("the temp folder is too long for a socket here")
	}
	p := controlPathIn(long, short)
	if p != filepath.Join(short, "cm-%C") {
		t.Fatalf("controlPath = %q", p)
	}
	fi, err := os.Lstat(short)
	if err != nil || fi.Mode().Perm() != 0o700 {
		t.Errorf("%s: %v %v", short, fi.Mode(), err)
	}
	if _, err := os.Lstat(long); err == nil {
		t.Error("a folder too long for a socket was made")
	}
	// a link is not taken
	link := filepath.Join(t.TempDir(), "l")
	if err := os.Symlink(short, link); err != nil {
		t.Fatal(err)
	}
	if p := controlPathIn(link); p != "" {
		t.Errorf("a link was taken: %q", p)
	}
}
