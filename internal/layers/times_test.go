package layers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// layoutOwner is the one package that holds time layouts: every other
// package asks it, so that a time is written the same way wherever it is.
const layoutOwner = "internal/core/when"

// layoutParts are what a Go layout of a date or a clock holds.
var layoutParts = []string{"2006", "15:04", "04:05"}

// layoutNames are the time package's own layouts.
var layoutNames = map[string]bool{
	"Layout": true, "ANSIC": true, "UnixDate": true, "RubyDate": true,
	"RFC822": true, "RFC822Z": true, "RFC850": true, "RFC1123": true, "RFC1123Z": true,
	"RFC3339": true, "RFC3339Nano": true, "Kitchen": true,
	"Stamp": true, "StampMilli": true, "StampMicro": true, "StampNano": true,
	"DateTime": true, "DateOnly": true, "TimeOnly": true,
}

// A time layout outside internal/core/when, a string such as "2006-01-02" or a
// name such as time.RFC3339, fails: the layout belongs in when, with a name.
func TestTimeLayoutsLiveInWhen(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if path != root && (strings.HasPrefix(name, ".") || name == "testdata" || name == "dist") {
				return filepath.SkipDir
			}
			// A nested module (desktop/) keeps its own times: it cannot
			// import this module's internal/core/when.
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
		rel, _ := filepath.Rel(root, path)
		if strings.HasPrefix(filepath.ToSlash(rel), layoutOwner+"/") {
			return nil
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		timeName := ""
		for _, imp := range f.Imports {
			if p, _ := strconv.Unquote(imp.Path.Value); p == "time" {
				timeName = "time"
				if imp.Name != nil {
					timeName = imp.Name.Name
				}
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.BasicLit:
				if n.Kind != token.STRING {
					break
				}
				s, _ := strconv.Unquote(n.Value)
				for _, part := range layoutParts {
					if strings.Contains(s, part) {
						t.Errorf("%s: the time layout %s belongs in %s", fset.Position(n.Pos()), n.Value, layoutOwner)
						break
					}
				}
			case *ast.SelectorExpr:
				if id, ok := n.X.(*ast.Ident); ok && timeName != "" && id.Name == timeName && layoutNames[n.Sel.Name] {
					t.Errorf("%s: time.%s is a time layout: it belongs in %s", fset.Position(n.Pos()), n.Sel.Name, layoutOwner)
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
