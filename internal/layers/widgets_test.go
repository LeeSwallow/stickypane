package layers

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// unseen are the files of a shape that know nothing of the terminal: how
// the file is written, what it says, how it changes, what happened in it,
// and its kind. Only view.go draws, so only it may import the terminal.
var unseen = map[string]bool{"format.go": true, "model.go": true, "ops.go": true, "events.go": true, "kind.go": true}

// terminal are the packages that draw on, or run, the terminal.
var terminal = []string{"charm.land/lipgloss", "charm.land/bubbletea"}

// sending are what talks to the network or runs a program. A widget reads
// a file and answers with intents; sending a request or running a script
// belongs to a front end, so no widget imports them. A .http file is read
// with internal/httpfile/syntax, never with internal/httpfile.
var sending = map[string]bool{"net": true, "net/http": true, "os/exec": true, module + "internal/httpfile": true, module + "internal/runner": true}

// The shapes under internal/widget keep their reading apart from their
// drawing, and none of them sends anything.
func TestWidgetsDrawOnlyInViewAndSendNothing(t *testing.T) {
	t.Parallel()
	root, err := filepath.Abs("../../internal/widget")
	if err != nil {
		t.Fatal(err)
	}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(filepath.Dir(root), path)
		rel = filepath.ToSlash(rel)
		shape := filepath.Dir(path) != root
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			if shape && unseen[filepath.Base(path)] {
				for _, term := range terminal {
					if p == term || strings.HasPrefix(p, term+"/") {
						t.Errorf("%s imports %s: only a shape's view.go draws", rel, p)
					}
				}
			}
			if sending[p] || strings.HasPrefix(p, "net/http/") {
				t.Errorf("%s imports %s: a widget sends nothing and runs nothing", rel, p)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
