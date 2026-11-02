package remote

import (
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

func TestKeyRoundTrips(t *testing.T) {
	k := Key("dev-box", "/home/min/my app/")
	if k != "ssh://dev-box/home/min/my app" {
		t.Fatalf("Key = %q", k)
	}
	name, dir, ok := ParseKey(k)
	if !ok || name != "dev-box" || dir != "/home/min/my app" {
		t.Fatalf("ParseKey = %q %q %v", name, dir, ok)
	}
	if !IsRemote(k) || IsRemote("/Users/min/app") {
		t.Error("IsRemote tells a remote key from a local path")
	}
	for _, bad := range []string{"/local", "ssh://", "ssh:///x", "ssh://a b/x", "ssh://box"} {
		if _, _, ok := ParseKey(bad); ok {
			t.Errorf("ParseKey(%q) took it", bad)
		}
	}
}

func TestValidName(t *testing.T) {
	for _, ok := range []string{"dev", "dev-box.lan", "a_b9"} {
		if ValidName(ok) != nil {
			t.Errorf("%q refused", ok)
		}
	}
	for _, bad := range []string{"", "a b", "a/b", "ssh://x", "é"} {
		if ValidName(bad) == nil {
			t.Errorf("%q taken", bad)
		}
	}
}

func TestHostCheckRefusesWhatSSHWouldMisread(t *testing.T) {
	good := Host{Name: "dev", Target: "min@dev.lan", Boards: []string{"/srv/app"}}
	if err := good.Check(); err != nil {
		t.Fatal(err)
	}
	for _, h := range []Host{
		{Name: "dev", Target: ""},
		{Name: "dev", Target: "-oProxyCommand=evil"},
		{Name: "dev", Target: "a b"},
		{Name: "dev", Target: "dev;rm"},
		{Name: "dev", Target: "dev", Command: "x\ny"},
		{Name: "dev", Target: "dev", Boards: []string{"relative/path"}},
		{Name: "a b", Target: "dev"},
	} {
		if h.Check() == nil {
			t.Errorf("%+v taken", h)
		}
	}
}

func TestQuoteSurvivesTheShell(t *testing.T) {
	for _, s := range []string{"/plain", "/with space/x", "/it's", `/a"b`, "/$HOME/`x`", "/'; rm -rf /; '"} {
		out, err := exec.Command("sh", "-c", "printf %s "+Quote(s)).Output()
		if err != nil {
			t.Fatal(err)
		}
		if string(out) != s {
			t.Errorf("Quote(%q) came back as %q", s, out)
		}
	}
	if Quote("/it's") != `'/it'\''s'` {
		t.Errorf("Quote = %s", Quote("/it's"))
	}
}

func TestBridgeArgv(t *testing.T) {
	s := SSH{Bin: "ssh"}
	got := s.Bridge(Host{Name: "dev", Target: "dev"}, "/home/min/my app/it's")
	want := []string{"ssh", "-T", "-o", "BatchMode=yes", "-o", "ConnectTimeout=10", "dev", "--", `cd '/home/min/my app/it'\''s' && stickypane bridge`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Bridge =\n %q\nwant\n %q", got, want)
	}
	s = SSH{Bin: "/x/fake-ssh", ControlPath: "/tmp/sp-501/cm-%C"}
	got = s.Bridge(Host{Name: "dev", Target: "min@dev", Command: "~/go/bin/stickypane"}, "/srv")
	want = []string{"/x/fake-ssh", "-T", "-o", "BatchMode=yes", "-o", "ConnectTimeout=10",
		"-o", "ControlMaster=auto", "-o", "ControlPath=/tmp/sp-501/cm-%C", "-o", "ControlPersist=60",
		"min@dev", "--", "cd '/srv' && ~/go/bin/stickypane bridge"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Bridge =\n %q\nwant\n %q", got, want)
	}
}

func TestProbeArgvAndItsAnswer(t *testing.T) {
	got := SSH{}.Probe(Host{Name: "dev", Target: "dev"})
	if got[0] != "ssh" || got[len(got)-1] != "stickypane version && { stickypane help --all; true; }" {
		t.Fatalf("Probe = %q", got)
	}
	p := ReadProbe("stickypane 0.9.1\nUsage:\n  stickypane bridge   serve the board\n")
	if p.Version != "0.9.1" || !p.Bridge {
		t.Errorf("ReadProbe = %+v", p)
	}
	if p := ReadProbe("stickypane 0.3.0\nUsage:\n  stickypane list\n"); p.Bridge || p.Version != "0.3.0" {
		t.Errorf("an old copy: %+v", p)
	}
}

func TestResolve(t *testing.T) {
	if got := (SSH{}).Resolve("dev"); !reflect.DeepEqual(got, []string{"ssh", "-G", "dev"}) {
		t.Fatalf("Resolve = %q", got)
	}
	out := strings.Join([]string{"host dev", "user min", "hostname 10.0.0.5", "port 2222", "identityfile ~/.ssh/id_ed25519"}, "\n")
	r := ReadResolved(out)
	if r.String() != "min@10.0.0.5:2222" {
		t.Errorf("Resolved = %q", r.String())
	}
	if (Resolved{Hostname: "h"}).String() != "h" || (Resolved{}).String() != "" {
		t.Error("what is not known is left out")
	}
}
