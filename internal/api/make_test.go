package api

import (
	"errors"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/widget"
)

func TestMakeComposesByTheKindsOwnCode(t *testing.T) {
	a, _ := newAPI(t, nil)
	body, ext, err := a.Make("checklist", "Release", map[string]string{"items": "build\nship"})
	if err != nil || ext != ".md" || !strings.Contains(string(body), "- [ ] build\n- [ ] ship\n") {
		t.Errorf("checklist: %q %q %v", body, ext, err)
	}
	if body, ext, err := a.Make("script", "Build", map[string]string{"commands": "make"}); err != nil || ext != ".sh" || string(body) != "#!/bin/sh\nmake\n" {
		t.Errorf("script: %q %q %v", body, ext, err)
	}
	if _, _, err := a.Make("", "x", map[string]string{"text": "hi"}); err != nil {
		t.Errorf("no kind is a plain note: %v", err)
	}
	if _, _, err := a.Make("spaceship", "x", nil); !errors.Is(err, ErrUnknownType) {
		t.Errorf("unknown kind: %v", err)
	}
	if _, _, err := a.Make("chart", "x", map[string]string{"view": "pie"}); !errors.Is(err, widget.ErrBadOption) {
		t.Errorf("bad option: %v", err)
	}
}
