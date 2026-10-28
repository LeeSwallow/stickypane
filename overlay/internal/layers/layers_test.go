// Package layers holds the app module's layer tests (ADR-0018): its
// packages depend on each other in one direction only, the domain does no
// I/O, and the use cases reach the world only through their ports.
// docs/overlay/02 §2.6 draws the layers; this keeps the drawing true.
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

const module = "github.com/LeeSwallow/stickypane/overlay/"

// rank places every package of the module. A package may import only
// packages of a lower rank. A new package must be given one here.
var rank = map[string]int{
	// Domain: values and rules, no I/O.
	"internal/prefs":         0,
	"internal/index/model":   0,
	"internal/install/model": 0,
	// Ports: what the use cases need, as interfaces.
	"internal/index/ports":   10,
	"internal/install/ports": 10,
	// Use cases: orchestration and transactions, through the ports only.
	"internal/index/service":   20,
	"internal/install/service": 20,
	// Adapters: the database, the files, the processes, the bridge.
	"internal/index/gormstore":  30,
	"internal/index/registryfs": 30,
	"internal/index/boardfs":    30,
	"internal/index/procs":      30,
	"internal/bridge":           30,
	"internal/install/osys":     30,
	// Wiring of the index's parts.
	"internal/index": 40,
	// Screen: the Wails shell and the program.
	"internal/shell": 50,
	".":              60,
	// A development tool, apart from the app.
	"scripts/palettes": 0,
}

// pure are the packages that may not touch the world: the domain and the
// use cases. Each forbidden import is a prefix.
var pure = map[string]bool{
	"internal/prefs":           true,
	"internal/index/model":     true,
	"internal/index/ports":     true,
	"internal/index/service":   true,
	"internal/install/model":   true,
	"internal/install/ports":   true,
	"internal/install/service": true,
}

var forbidden = []string{
	"os", "io", "net", "syscall", "database/", "embed",
	"gorm.io/", "github.com/glebarez/", "modernc.org/", "github.com/wailsapp/",
}

func isForbidden(imp string) bool {
	for _, f := range forbidden {
		if imp == f || strings.HasPrefix(imp, strings.TrimSuffix(f, "/")+"/") {
			return true
		}
	}
	return false
}

// imports reads the non-test imports of every package of the module.
func imports(t *testing.T) map[string]map[string]bool {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]map[string]bool{}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if path != root && (strings.HasPrefix(name, ".") || name == "testdata" || name == "frontend" || name == "build" || name == "bin") {
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
		if out[pkg] == nil {
			out[pkg] = map[string]bool{}
		}
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			out[pkg][p] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestPackagesDependDownwardOnly(t *testing.T) {
	all := imports(t)
	var pkgs []string
	for p := range all {
		pkgs = append(pkgs, p)
	}
	sort.Strings(pkgs)
	for _, pkg := range pkgs {
		r, ok := rank[pkg]
		if !ok {
			t.Errorf("%s has no layer: give it a rank in this test and a place in docs/overlay/02", pkg)
			continue
		}
		for imp := range all[pkg] {
			dep, ok := strings.CutPrefix(imp, module)
			if !ok {
				continue
			}
			if d, ok := rank[dep]; ok && d >= r {
				t.Errorf("%s (layer %d) imports %s (layer %d): a package may only import a lower layer", pkg, r, dep, d)
			}
		}
	}
	for pkg := range rank {
		if all[pkg] == nil {
			t.Errorf("%s is ranked but has no Go files: take it out of the table", pkg)
		}
	}
}

func TestDomainAndUseCasesDoNoIO(t *testing.T) {
	all := imports(t)
	for pkg := range pure {
		if all[pkg] == nil {
			t.Errorf("%s is listed as pure but has no Go files", pkg)
		}
		for imp := range all[pkg] {
			if isForbidden(imp) {
				t.Errorf("%s imports %s: the domain and the use cases reach files, databases and processes only through ports", pkg, imp)
			}
		}
	}
}
