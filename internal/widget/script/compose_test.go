package script

import (
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

func TestComposeWritesTheCommands(t *testing.T) {
	b, err := Kind.Make("Build", map[string]string{"commands": "go vet ./...\ngo test ./..."})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "#!/bin/sh\ngo vet ./...\ngo test ./...\n" {
		t.Errorf("file = %q", b)
	}
	if got := parse(doc.Document{Body: string(b)}).lines; strings.Join(got, "|") != "#!/bin/sh|go vet ./...|go test ./..." {
		t.Errorf("read back %q", got)
	}
	if b, _ := Kind.Make("Empty", nil); string(b) != string(Kind.Template("Empty")) {
		t.Errorf("no commands is the template: %q", b)
	}
}
