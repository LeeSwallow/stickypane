package widget

import (
	"testing"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

func TestVersionChangesWithContent(t *testing.T) {
	a := doc.Document{Body: "one\n"}
	b := doc.Document{Body: "one\n"}
	c := doc.Document{Body: "two\n"}
	if Version(a) != Version(b) {
		t.Error("the same document must have the same version")
	}
	if Version(a) == Version(c) {
		t.Error("a document with different text must have another version")
	}
	if n := len(Version(a)); n != 16 {
		t.Errorf("version is %d characters, want 16", n)
	}
}
