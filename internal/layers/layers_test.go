// Package layers holds one test: that the packages of stickypane depend
// on each other in one direction only. docs/architecture.md draws the
// layers; this keeps the drawing true.
package layers

import (
	"go/parser"
	"go/token"
	"io/fs"
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
	"internal/doc":    0,
	"internal/theme":  0,
	"internal/env":    0, // the system, shell, pane tool and locale
	"internal/when":   0, // how times are written and read
	"internal/i18n":   5, // reads the locale from env
	"internal/layout": 0,
	"internal/editor": 0,
	// The notes folder.
	"internal/store": 10,
	// What a note is and how it is drawn: the contract, then the shapes.
	"internal/widget":           20,
	"internal/widget/note":      21, // the form draws its prose as a note does
	"internal/widget/board":     22,
	"internal/widget/chart":     22,
	"internal/widget/checklist": 22,
	"internal/widget/form":      22,
	"internal/widget/logview":   22,
	"internal/widget/script":    22,
	// The registry of shapes, and how a note is arranged.
	"internal/kinds":   30,
	"internal/arrange": 30,
	// What the command line and agents call.
	"internal/api":     40,
	"internal/initcmd": 40,
	// The two front ends over it.
	"internal/mcp": 50,
	"internal/app": 50,
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
