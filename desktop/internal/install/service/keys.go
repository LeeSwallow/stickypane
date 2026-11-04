package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/LeeSwallow/stickypane/desktop/internal/install/model"
	"github.com/LeeSwallow/stickypane/desktop/internal/install/ports"
)

// Keys is what the settings show of the terminal board's keys: every
// action, and what of keys.json the board left out. Missing says there is
// no stickypane to ask; the settings then say how to get one.
type Keys struct {
	Bindings []model.Binding `json:"bindings"`
	Problems []string        `json:"problems,omitempty"`
	Missing  bool            `json:"missing,omitempty"`
}

// Keys lists the terminal board's keys through `stickypane keys --json`,
// their words in lang ("ko", "en"; "" for the system's).
func (s *Service) Keys(ctx context.Context, lang string) (Keys, error) {
	cli, err := s.cli()
	if err != nil {
		return Keys{Missing: true}, nil
	}
	out, err := s.runKeys(ctx, cli, lang, model.KeysArgs())
	if err != nil {
		if model.RunError(out) != "" {
			return Keys{}, errors.Join(model.ErrNoKeys, errors.New(model.RunError(out)))
		}
		return Keys{}, err
	}
	bs, problems, err := model.ParseKeys(out)
	return Keys{Bindings: bs, Problems: problems}, err
}

// SetKeys gives action keys in keys.json (none: it has none), through
// `stickypane keys set`, which refuses a key it cannot read or one
// another action has; its sentence is the error. It answers the keys as
// they are then.
func (s *Service) SetKeys(ctx context.Context, lang, action string, keys []string) (Keys, error) {
	return s.changeKeys(ctx, lang, model.SetKeysArgs(action, keys))
}

// ResetKeys takes action's change out of keys.json, or, for "", every
// change, through `stickypane keys reset`.
func (s *Service) ResetKeys(ctx context.Context, lang, action string) (Keys, error) {
	return s.changeKeys(ctx, lang, model.ResetKeysArgs(action))
}

func (s *Service) changeKeys(ctx context.Context, lang string, args []string) (Keys, error) {
	cli, err := s.cli()
	if err != nil {
		return Keys{Missing: true}, err
	}
	if out, err := s.runKeys(ctx, cli, lang, args); err != nil {
		if why := model.RunError(out); why != "" {
			return Keys{}, errors.New(why)
		}
		return Keys{}, err
	}
	return s.Keys(ctx, lang)
}

// runKeys runs the keys command in a new empty folder of its own, so that
// no board around the app's folder gives the words its language.
func (s *Service) runKeys(ctx context.Context, cli, lang string, args []string) (string, error) {
	dir, err := s.cfg.Files.TempDir()
	if err != nil {
		return "", err
	}
	if !s.safe(dir) {
		return "", fmt.Errorf("%w: %s", ErrUnsafeDir, dir)
	}
	defer s.cfg.Files.RemoveAll(dir)
	c := ports.Cmd{Dir: dir, Name: cli, Args: args}
	if lang != "" {
		c.Env = []string{"STICKYPANE_LANG=" + lang}
	}
	return s.cfg.Runner.Run(ctx, c)
}
