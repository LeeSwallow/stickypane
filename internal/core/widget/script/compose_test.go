package script

import (
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/core/doc"
	"github.com/LeeSwallow/stickypane/internal/core/widget"
)

func TestComposeWritesTheCommands(t *testing.T) {
	t.Parallel()
	b, err := Kind.Make("Build", map[string]string{"commands": "go vet ./...\ngo test ./..."}, widget.Env{})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "#!/bin/sh\ngo vet ./...\ngo test ./...\n" {
		t.Errorf("file = %q", b)
	}
	if got := parse(doc.Document{Body: string(b)}, widget.DefaultEnv()).lines; strings.Join(got, "|") != "#!/bin/sh|go vet ./...|go test ./..." {
		t.Errorf("read back %q", got)
	}
	if b, _ := Kind.Make("Empty", nil, widget.Env{}); string(b) != string(Kind.Template("Empty", widget.Env{})) {
		t.Errorf("no commands is the template: %q", b)
	}
}
