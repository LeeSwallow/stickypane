package shell

import (
	"context"
	"testing"

	"github.com/LeeSwallow/stickypane/desktop/internal/install/model"
	installsvc "github.com/LeeSwallow/stickypane/desktop/internal/install/service"
)

// keyedInstaller is a fakeInstaller that keeps keys.json in memory.
type keyedInstaller struct {
	fakeInstaller
	file  map[string][]string
	langs []string
}

func (k *keyedInstaller) Keys(_ context.Context, lang string) (installsvc.Keys, error) {
	k.langs = append(k.langs, lang)
	keys, changed := k.file["jot"]
	if !changed {
		keys = []string{"N", "n"}
	}
	return installsvc.Keys{Bindings: []model.Binding{{Name: "jot", Keys: keys, Changed: changed, Defaults: []string{"N", "n"}, Group: "notes"}}}, nil
}
func (k *keyedInstaller) SetKeys(ctx context.Context, lang, action string, keys []string) (installsvc.Keys, error) {
	k.file[action] = keys
	return k.Keys(ctx, lang)
}
func (k *keyedInstaller) ResetKeys(ctx context.Context, lang, action string) (installsvc.Keys, error) {
	delete(k.file, action)
	return k.Keys(ctx, lang)
}

func TestTheSettingsChangeTheBoardsKeysThroughTheInstaller(t *testing.T) {
	r := newRig(t)
	k := &keyedInstaller{file: map[string][]string{}}
	r.s.cfg.Install = k
	x := &SetupService{S: r.s}
	got, err := x.SetKeys("ko", "jot", []string{"ctrl+n"})
	if err != nil || !got.Bindings[0].Changed || got.Bindings[0].Keys[0] != "ctrl+n" {
		t.Fatalf("%+v %v", got, err)
	}
	if got, _ = x.ResetKeys("ko", "jot"); got.Bindings[0].Changed {
		t.Errorf("reset: %+v", got)
	}
	if got, _ = x.Keys("en"); got.Bindings[0].Keys[0] != "N" || k.langs[len(k.langs)-1] != "en" {
		t.Errorf("%+v %q", got, k.langs)
	}
}

// An installer without keys (a build's stand-in) has none to show, and
// says so as a stickypane that is missing.
func TestAnInstallerWithoutKeysHasNone(t *testing.T) {
	r := newRig(t)
	x := &SetupService{S: r.s}
	if got, err := x.Keys("en"); err != nil || !got.Missing {
		t.Errorf("%+v %v", got, err)
	}
	if _, err := x.SetKeys("en", "jot", nil); err == nil {
		t.Error("set without a keyer")
	}
}
