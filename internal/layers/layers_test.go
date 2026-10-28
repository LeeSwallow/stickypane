// Package layers holds one test: that the packages of stickypane depend
// on each other in one direction only. docs/architecture.md draws the
// layers; this keeps the drawing true.
package layers

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const module = "github.com/LeeSwallow/stickypane/"

// rank places every package. A package may import only packages of a
// lower rank. A new package must be given one here, which is the moment to
// decide where it belongs.
var rank = map[string]int{
	// Leaves: they know nothing of the board.
	"internal/core/doc":                0,
	"internal/core/theme":              0,
	"internal/adapter/env":             0, // the system, shell, pane tool and locale
	"internal/core/when":               0, // how times are written and read
	"internal/adapter/atomicfile":      0, // the one way a file is replaced
	"internal/adapter/filelock":        0, // the lock writers take around a read, change and write
	"internal/core/i18n":               0, // the words; the caller hands it the locale
	".":                                0, // the plugin's files, carried in the program
	"internal/usecase/harness":         5, // the plugin's skills, read for MCP and guide
	"internal/adapter/httpfile":        5, // .http files: send, check; writes times with when
	"internal/core/httpfile/syntax":    0, // .http files read, with nothing sent: what the board draws
	"internal/adapter/registry":        5, // which boards are used on this machine; writes times with when
	"internal/core/layout":             0,
	"internal/surface/editor":          0,
	"internal/surface/lineedit":        0, // the one-line editor of the bottom line
	"internal/core/markdown":           0, // Markdown drawn for the terminal
	"internal/adapter/wsock":           0, // a WebSocket client on the standard library
	"internal/adapter/proto":           0, // protocol buffers without generated code
	"internal/adapter/proto/prototest": 1, // a descriptor for tests
	"internal/adapter/rpc":             3, // a gRPC client on the standard library
	"internal/adapter/rpc/rpctest":     2, // a gRPC server for tests
	// The notes folder.
	"internal/adapter/store": 10,
	// Running a script and keeping its log, for every front end.
	"internal/adapter/runner": 12,
	// What a user may change, each with a default.
	"internal/usecase/prefs": 15,
	// What a note is and how it is drawn: the contract, then the shapes.
	"internal/core/widget":           20,
	"internal/core/widget/note":      21, // the form draws its prose as a note does
	"internal/core/widget/board":     22,
	"internal/core/widget/chart":     22,
	"internal/core/widget/checklist": 22,
	"internal/core/widget/form":      22,
	"internal/core/widget/logview":   22,
	"internal/core/widget/chat":      22,
	"internal/core/widget/script":    22,
	"internal/core/widget/api":       22,
	"internal/core/widget/html":      22,
	// The registry of shapes, and how a note is arranged.
	"internal/core/kinds":      30,
	"internal/usecase/arrange": 30,
	// What the command line and agents call.
	"internal/usecase/api":     40,
	"internal/usecase/initcmd": 40,
	// The front ends over it.
	"internal/surface/mcp":    50,
	"internal/surface/app":    50,
	"internal/surface/bridge": 50, // the desktop app's, over standard input/output
	// The program.
	"cmd/stickypane": 60,
}

func TestPackagesDependDownwardOnly(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	imports := map[string]map[string]bool{}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if path != root && (strings.HasPrefix(name, ".") || name == "testdata" || name == "dist") {
				return filepath.SkipDir
			}
			// A nested module (overlay/) is not part of this one's layers.
			if path != root {
				if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(root, filepath.Dir(path))
		pkg := filepath.ToSlash(rel)
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		if imports[pkg] == nil {
			imports[pkg] = map[string]bool{}
		}
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			if strings.HasPrefix(p, module) {
				imports[pkg][strings.TrimPrefix(p, module)] = true
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var pkgs []string
	for p := range imports {
		pkgs = append(pkgs, p)
	}
	sort.Strings(pkgs)
	for _, pkg := range pkgs {
		r, ok := rank[pkg]
		if !ok {
			t.Errorf("%s has no layer: give it a rank in this test and a place in docs/architecture.md", pkg)
			continue
		}
		for dep := range imports[pkg] {
			if d, ok := rank[dep]; ok && d >= r {
				t.Errorf("%s (layer %d) imports %s (layer %d): a package may only import a lower layer", pkg, r, dep, d)
			}
		}
	}
	for pkg := range rank {
		if imports[pkg] == nil {
			t.Errorf("%s is ranked but has no Go files: take it out of the table", pkg)
		}
	}
}
