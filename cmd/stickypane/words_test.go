package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/core/i18n"
)

// personFiles print what a person reads, so every sentence in them goes
// through i18n. The note commands (notes.go, edit.go, said.go, kinds.go,
// watchcmd.go, ...) speak to agents and scripts, in English, and are not
// here.
var personFiles = []string{
	"setupcmd.go", "configcmd.go", "settings.go", "keyscmd.go", "envcmd.go",
	"errors.go", "desktop.go", "help.go",
	"../../internal/usecase/initcmd/initcmd.go", "../../internal/usecase/initcmd/setup.go",
}

// wordsAllowed are the words a format may hold without translation: what
// is typed (the command, help, auto, [path]), and the row names of
// stickypane env, which are its --json keys.
var wordsAllowed = []string{"stickypane", "help", "auto", "path", "system", "shell", "pane", "language", "scripts", "editor", "sh", "ps"}

// sentinels are errors a person never reads as they are: a command tells
// them apart with errors.Is and says something else.
var sentinels = []string{"the stickypane app is not installed", "will not make a board here"}

var (
	verb = regexp.MustCompile(`%[-+# 0-9.*]*[a-zA-Z%]`)
	word = regexp.MustCompile(`[A-Za-z]{2,}`)
)

// A sentence a person reads is never written into a print in English: the
// prints in personFiles format only what i18n said, and every key they
// name is a sentence i18n has.
func TestPersonFacingOutputGoesThroughI18n(t *testing.T) {
	keys := i18n.SaysKeys()
	for _, file := range personFiles {
		f, err := parser.ParseFile(token.NewFileSet(), filepath.FromSlash(file), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			name := callee(call)
			switch name {
			case "fmt.Fprint", "fmt.Fprintf", "fmt.Fprintln", "fmt.Errorf", "fmt.Sprintf", "errors.New", "usagef", "io.WriteString":
				for _, arg := range call.Args {
					lit, ok := arg.(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						continue
					}
					text, _ := strconv.Unquote(lit.Value)
					if name == "errors.New" && slices.Contains(sentinels, text) {
						continue
					}
					for _, w := range word.FindAllString(verb.ReplaceAllString(text, ""), -1) {
						if !slices.Contains(wordsAllowed, w) {
							t.Errorf("%s: %s(%s) writes %q to a person in English: say it with i18n (Strings.Say)", file, name, lit.Value, w)
							break
						}
					}
				}
			case "Say", "i18n.Err", "usageSay":
				for _, arg := range call.Args {
					lit, ok := arg.(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						continue
					}
					key, _ := strconv.Unquote(lit.Value)
					if strings.Contains(key, ".") && !slices.Contains(keys, key) && !strings.HasSuffix(key, ".") {
						t.Errorf("%s: %s names %q, which is no sentence of i18n", file, name, key)
					}
				}
			}
			return true
		})
	}
}

// callee is the name a call is made by: "fmt.Fprintf", "usagef", or the
// method's name ("Say") for a call on a value.
func callee(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return fn.Name
	case *ast.SelectorExpr:
		if pkg, ok := fn.X.(*ast.Ident); ok && (pkg.Name == "fmt" || pkg.Name == "errors" || pkg.Name == "io" || pkg.Name == "i18n") {
			return pkg.Name + "." + fn.Sel.Name
		}
		return fn.Sel.Name
	}
	return ""
}
