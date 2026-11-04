package shell

import (
	"errors"
	"testing"
)

// The shell's errors are told apart with errors.Is, and keep the words the
// windows have always shown.
func TestErrorsAreSentinelsWithTheirWords(t *testing.T) {
	r := remoteRig(t, &sshFake{}, devHost, Host{Name: "box", Target: "box"})
	cases := []struct {
		name   string
		err    error
		target error
		words  string
	}{
		{"pin", r.s.SetPin(99, true), ErrNoWindow, "no project window 99"},
		{"make board", r.s.MakeBoard(98), ErrNoWindow, "no project window 98"},
		{"layout", func() error { _, err := r.s.Layout(97); return err }(), ErrNoWindow, "no project window 97"},
		{"call", r.s.Call(t.Context(), 96, "board.load", nil, nil), ErrNoBoard, "no board in this window (closed)"},
		{"tear nothing", func() error { _, err := r.s.TearOff(1, " "); return err }(), ErrNoNote, "which note?"},
		{"remove host", r.s.RemoveHost("nobody"), ErrNoHost, `no host "nobody"`},
		{"add board", r.s.AddBoard("nobody", "/x"), ErrNoHost, `no host "nobody"`},
		{"taken", r.s.SaveHost(Host{Name: "box", Target: "box"}, "dev"), ErrHostTaken, `a host is already named "box"`},
		{"root", r.s.InitProject("/"), ErrUnsafeFolder, "will not make a board in the root folder"},
	}
	for _, c := range cases {
		if !errors.Is(c.err, c.target) {
			t.Errorf("%s: %v is not %v", c.name, c.err, c.target)
			continue
		}
		if c.err.Error() != c.words {
			t.Errorf("%s: %q, want %q", c.name, c.err, c.words)
		}
	}
	if got := r.s.CheckHost("nobody").Error; got != `no host "nobody"` {
		t.Errorf("CheckHost of no host: %q", got)
	}
}

func TestRenamingAHostWithOpenBoardsIsErrHostOpen(t *testing.T) {
	r := remoteRig(t, &sshFake{}, devHost)
	if _, err := r.s.OpenRemote("dev", "/home/min/my app"); err != nil {
		t.Fatal(err)
	}
	err := r.s.SaveHost(Host{Name: "dev2", Target: "min@dev.lan"}, "dev")
	if !errors.Is(err, ErrHostOpen) {
		t.Fatalf("rename with open boards: %v", err)
	}
	if want := `close the boards of "dev" before renaming it`; err.Error() != want {
		t.Errorf("%q, want %q", err, want)
	}
}
