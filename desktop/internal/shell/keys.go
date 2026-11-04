package shell

import (
	"context"
	"errors"
	"time"

	installsvc "github.com/LeeSwallow/stickypane/desktop/internal/install/service"
)

// Keyer is what the settings' 단축키 asks of the installer: the terminal
// board's keys, through the app's own stickypane (`stickypane keys`). The
// install service has it; an installer without it has no keys to show.
type Keyer interface {
	Keys(ctx context.Context, lang string) (installsvc.Keys, error)
	SetKeys(ctx context.Context, lang, action string, keys []string) (installsvc.Keys, error)
	ResetKeys(ctx context.Context, lang, action string) (installsvc.Keys, error)
}

// keysTimeout is how long a keys run may take.
const keysTimeout = 30 * time.Second

// errNoKeyer: the installer cannot change keys (a test's).
var errNoKeyer = errors.New("no stickypane to change the keys with")

func (x *SetupService) keyer() (Keyer, context.Context, context.CancelFunc, error) {
	k, ok := x.S.cfg.Install.(Keyer)
	if !ok {
		return nil, nil, nil, errNoKeyer
	}
	ctx, cancel := context.WithTimeout(context.Background(), keysTimeout)
	return k, ctx, cancel, nil
}

// Keys lists the terminal board's keys, their words in lang.
func (x *SetupService) Keys(lang string) (installsvc.Keys, error) {
	k, ctx, cancel, err := x.keyer()
	if err != nil {
		return installsvc.Keys{Missing: true}, nil
	}
	defer cancel()
	return k.Keys(ctx, lang)
}

// SetKeys gives an action keys in keys.json; none leaves it without one.
func (x *SetupService) SetKeys(lang, action string, keys []string) (installsvc.Keys, error) {
	k, ctx, cancel, err := x.keyer()
	if err != nil {
		return installsvc.Keys{}, err
	}
	defer cancel()
	return k.SetKeys(ctx, lang, action, keys)
}

// ResetKeys takes an action's change back, or, for "", every change.
func (x *SetupService) ResetKeys(lang, action string) (installsvc.Keys, error) {
	k, ctx, cancel, err := x.keyer()
	if err != nil {
		return installsvc.Keys{}, err
	}
	defer cancel()
	return k.ResetKeys(ctx, lang, action)
}
