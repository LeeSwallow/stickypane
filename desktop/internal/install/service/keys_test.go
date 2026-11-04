package service

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/desktop/internal/install/model"
	"github.com/LeeSwallow/stickypane/desktop/internal/install/ports"
)

// keysRunner is a stickypane that knows two actions and keeps keys.json
// in memory: it lists, sets and resets as the real one does, and refuses
// q for archiving (quit's key).
type keysRunner struct {
	file map[string][]string
	ran  []ports.Cmd
	bad  string // a line printed before the list, as for an entry left out
}

func (r *keysRunner) Run(_ context.Context, c ports.Cmd) (string, error) {
	r.ran = append(r.ran, c)
	defaults := map[string][]string{"note-archive": {"x"}, "quit": {"q"}}
	words := map[string]string{"note-archive": "move to archive", "quit": "quit"}
	if slices.Contains(c.Env, "STICKYPANE_LANG=ko") {
		words = map[string]string{"note-archive": "보관함으로", "quit": "끝내기"}
	}
	switch a := c.Args; {
	case slices.Equal(a, []string{"keys", "--json"}):
		var bs []model.Binding
		for _, n := range []string{"note-archive", "quit"} {
			keys, changed := r.file[n]
			if !changed {
				keys = defaults[n]
			}
			bs = append(bs, model.Binding{Name: n, Keys: keys, Words: words[n], Changed: changed, Defaults: defaults[n], Group: "notes"})
		}
		b, _ := json.Marshal(bs)
		out := string(b) + "\n"
		if r.bad != "" {
			out = r.bad + "\n" + out
		}
		return out, nil
	case len(a) >= 4 && a[1] == "set" && a[3] == "--":
		if slices.Contains(a[4:], "q") {
			return "stickypane: q is quit's key too; keys.json is left as it was\n", errors.New("exit status 1")
		}
		r.file[a[2]] = append([]string{}, a[4:]...)
		return a[2] + ": " + strings.Join(a[4:], " ") + "\n", nil
	case len(a) == 3 && a[1] == "reset":
		delete(r.file, a[2])
		return "", nil
	case len(a) == 2 && a[1] == "reset":
		r.file = map[string][]string{}
		return "keys.json removed\n", nil
	}
	return "stickypane: unknown command " + c.Args[0] + "\nusage: stickypane ...\n", errors.New("exit status 2")
}

func keysRig(r *keysRunner) rig {
	g := newRig("/usr/bin")
	g.s.cfg.Runner = r
	return g
}

func TestKeysListsTheBoardsKeysInTheAppsLanguage(t *testing.T) {
	r := &keysRunner{file: map[string][]string{}, bad: "keys.json: nope: there is no such action; it keeps the keys it comes with"}
	g := keysRig(r)
	k, err := g.s.Keys(ctx, "ko")
	if err != nil {
		t.Fatal(err)
	}
	if len(k.Bindings) != 2 || k.Bindings[0].Words != "보관함으로" || k.Missing {
		t.Errorf("%+v", k)
	}
	if len(k.Problems) != 1 || !strings.Contains(k.Problems[0], "nope") {
		t.Errorf("problems %q", k.Problems)
	}
	c := r.ran[0]
	if c.Name != bundled || c.Dir != "/tmp/stickypane-setup-1" || !slices.Equal(c.Env, []string{"STICKYPANE_LANG=ko"}) {
		t.Errorf("ran %+v", c)
	}
	if g.f.dirs["/tmp/stickypane-setup-1"] {
		t.Error("the folder was left behind")
	}
}

func TestSetKeysSaysWhyTheBoardRefusedAndAnswersTheNewKeys(t *testing.T) {
	r := &keysRunner{file: map[string][]string{}}
	g := keysRig(r)
	if _, err := g.s.SetKeys(ctx, "en", "note-archive", []string{"q"}); err == nil || err.Error() != "q is quit's key too; keys.json is left as it was" {
		t.Errorf("err = %v", err)
	}
	k, err := g.s.SetKeys(ctx, "en", "note-archive", []string{"X", "d x"})
	if err != nil {
		t.Fatal(err)
	}
	if b := k.Bindings[0]; !b.Changed || !slices.Equal(b.Keys, []string{"X", "d x"}) {
		t.Errorf("%+v", b)
	}
	if k, _ = g.s.SetKeys(ctx, "en", "quit", nil); len(k.Bindings[1].Keys) != 0 || !k.Bindings[1].Changed {
		t.Errorf("unbound: %+v", k.Bindings[1])
	}
	if k, _ = g.s.ResetKeys(ctx, "en", "quit"); k.Bindings[1].Changed {
		t.Errorf("reset one: %+v", k.Bindings[1])
	}
	if k, _ = g.s.ResetKeys(ctx, "en", ""); k.Bindings[0].Changed {
		t.Errorf("reset all: %+v", k.Bindings[0])
	}
}

func TestKeysWithoutAStickypaneOrWithAnOldOne(t *testing.T) {
	g := newRig("/usr/bin", func(c *Config) { c.Bundled = func() string { return "" } })
	if k, err := g.s.Keys(ctx, ""); err != nil || !k.Missing {
		t.Errorf("none: %+v %v", k, err)
	}
	if _, err := g.s.SetKeys(ctx, "", "quit", nil); err == nil {
		t.Error("set with no stickypane")
	}
	old := keysRig(&keysRunner{})
	old.s.cfg.Runner = runFunc(func(ports.Cmd) (string, error) {
		return "stickypane: unknown command keys\nusage: stickypane ...\n", errors.New("exit status 2")
	})
	if _, err := old.s.Keys(ctx, ""); !errors.Is(err, model.ErrNoKeys) || !strings.Contains(err.Error(), "unknown command keys") {
		t.Errorf("old: %v", err)
	}
}

type runFunc func(ports.Cmd) (string, error)

func (f runFunc) Run(_ context.Context, c ports.Cmd) (string, error) { return f(c) }
