// Package layers holds the tests that keep the packages of stickypane
// depending on each other in one direction only. docs/architecture.md
// draws the layers; this keeps the drawing true.
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

// folders are the layer folders under internal/, lowest first. Every
// package under internal/ sits in one of them, internal/layers (these
// tests) apart, and imports from its own folder or a lower one: core
// knows nothing outside itself, adapters reach the files, the system and
// the network for it, use cases go through both, and the front ends of
// surface over all three.
var folders = []string{
	"internal/core",    // the board's own ideas: notes, shapes, words, times
	"internal/adapter", // the world: files, the system, processes, the network
	"internal/usecase", // what every front end does, over core and adapters
	"internal/surface", // the front ends: the terminal, the bridge, MCP
}

// rank places every package. A package may import only packages of a
// lower rank. A new package must be given one here, in its folder, which
// is the moment to decide where it belongs.
var rank = map[string]int{
	".": 0, // the plugin's files, carried in the program

	// core: what a note is and how it is drawn, reading nothing outside.
	"internal/core/doc":             0,
	"internal/core/theme":           0,
	"internal/core/when":            0, // how times are written and read
	"internal/core/i18n":            0, // the words; the caller hands it the locale
	"internal/core/httpfile/syntax": 0, // .http files read, with nothing sent: what the board draws
	"internal/core/layout":          0,
	"internal/core/markdown":        0, // Markdown drawn for the terminal
	// The contract of a note, then the shapes.
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
	// The registry of shapes.
	"internal/core/kinds": 30,

	// adapter: files, the system, processes and the network.
	"internal/adapter/env":             0,  // the system, shell, pane tool and locale
	"internal/adapter/atomicfile":      0,  // the one way a file is replaced
	"internal/adapter/filelock":        0,  // the lock writers take around a read, change and write
	"internal/adapter/wsock":           0,  // a WebSocket client on the standard library
	"internal/adapter/proto":           0,  // protocol buffers without generated code
	"internal/adapter/proto/prototest": 1,  // a descriptor for tests
	"internal/adapter/rpc/rpctest":     2,  // a gRPC server for tests
	"internal/adapter/rpc":             3,  // a gRPC client on the standard library
	"internal/adapter/httpfile":        5,  // .http files: send, check; writes times with when
	"internal/adapter/registry":        5,  // which boards are used on this machine; writes times with when
	"internal/adapter/store":           10, // the notes folder
	"internal/adapter/runner":          12, // running a script and keeping its log, for every front end

	// usecase: what the command line, agents and every screen call.
	"internal/usecase/harness": 5,  // the plugin's skills, read for MCP and guide
	"internal/usecase/prefs":   15, // what a user may change, each with a default
	"internal/usecase/arrange": 30, // how a note is arranged
	"internal/usecase/api":     40,
	"internal/usecase/initcmd": 40,

	// surface: the front ends over it.
	"internal/surface/editor":   0,
	"internal/surface/lineedit": 0, // the one-line editor of the bottom line
	"internal/surface/mcp":      50,
	"internal/surface/app":      50,
	"internal/surface/bridge":   50, // the desktop app's, over standard input/output

	// The program.
	"cmd/stickypane": 60,
}

// folder is the place in folders of the layer folder pkg sits in, and
// whether it sits in one.
func folder(pkg string) (int, bool) {
	for i, f := range folders {
		if pkg == f || strings.HasPrefix(pkg, f+"/") {
			return i, true
		}
	}
	return 0, false
}

// Every package under internal/ sits in a layer folder, and imports
// nothing from a higher folder.
func TestPackagesSitInTheirLayersFolder(t *testing.T) {
	imports := importsOf(t)
	for _, pkg := range sorted(imports) {
		f, in := folder(pkg)
		if !in {
			if strings.HasPrefix(pkg, "internal/") {
				t.Errorf("%s is in no layer folder: move it under one of %s", pkg, strings.Join(folders, ", "))
			}
			continue
		}
		for _, dep := range sorted(imports[pkg]) {
			if d, ok := folder(dep); ok && d > f {
				t.Errorf("%s (in %s) imports %s (in %s): a package imports only from its own folder or a lower one", pkg, folders[f], dep, folders[d])
			}
		}
	}
	for _, f := range folders {
		if _, err := os.Stat(filepath.Join("..", "..", f)); err != nil {
			t.Errorf("%s: %v", f, err)
		}
	}
}

func TestPackagesDependDownwardOnly(t *testing.T) {
	imports := importsOf(t)
	for _, pkg := range sorted(imports) {
		r, ok := rank[pkg]
		if !ok {
			t.Errorf("%s has no layer: give it a rank in this test and a place in docs/architecture.md", pkg)
			continue
		}
		for _, dep := range sorted(imports[pkg]) {
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

// importsOf reads which of this module's packages each package of it
// imports, by the package's folder from the module root. Test files do
// not count: a test may reach up to build its fixture.
func importsOf(t *testing.T) map[string]map[string]bool {
	t.Helper()
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
	return imports
}

// sorted is the keys of m, in order, so that failures read the same way
// on every run.
func sorted[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
