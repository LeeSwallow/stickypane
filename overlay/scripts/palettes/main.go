// Command palettes prints the main module's built-in colour themes as JSON,
// in the shape of the bridge's palette, so the frontend's mirror
// (frontend/src/lib/theme/palettes.ts) can be checked against them. This
// module cannot import the main one's internal/theme, so it reads the Go
// source: the `all` table in palettes.go and the string constants it uses.
//
//	go run ./scripts/palettes ../internal/theme > frontend/src/lib/theme/palettes.golden.json
//
// Task A11 replaces the mirror with a bridge `themes.list`; this goes then.
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type palette struct {
	Name   string    `json:"name"`
	Dark   bool      `json:"dark"`
	Base   string    `json:"base"`
	Text   string    `json:"text"`
	Muted  string    `json:"muted"`
	Accent string    `json:"accent"`
	Select string    `json:"select"`
	Good   string    `json:"good"`
	Warn   string    `json:"warn"`
	Bad    string    `json:"bad"`
	Info   string    `json:"info"`
	Notes  [6]string `json:"notes"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: palettes DIR (the main module's internal/theme)")
		os.Exit(2)
	}
	out, err := read(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "palettes:", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, "palettes:", err)
		os.Exit(1)
	}
}

// read parses the package in dir and evaluates its `all` table.
func read(dir string) ([]palette, error) {
	fset := token.NewFileSet()
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return nil, err
	}
	consts := map[string]string{}
	var table *ast.CompositeLit
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			return nil, err
		}
		for _, d := range file.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, s := range g.Specs {
				v, ok := s.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, n := range v.Names {
					if i >= len(v.Values) {
						continue
					}
					if g.Tok == token.CONST {
						if lit, ok := v.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
							consts[n.Name], _ = strconv.Unquote(lit.Value)
						}
					}
					if g.Tok == token.VAR && n.Name == "all" {
						table, _ = v.Values[i].(*ast.CompositeLit)
					}
				}
			}
		}
	}
	if table == nil {
		return nil, fmt.Errorf("no `var all = []Theme{...}` in %s", dir)
	}
	str := func(e ast.Expr) (string, error) {
		switch x := e.(type) {
		case *ast.BasicLit:
			return strconv.Unquote(x.Value)
		case *ast.Ident:
			if s, ok := consts[x.Name]; ok {
				return s, nil
			}
		}
		return "", fmt.Errorf("not a string: %T", e)
	}
	var out []palette
	for _, el := range table.Elts {
		lit, ok := el.(*ast.CompositeLit)
		if !ok {
			return nil, fmt.Errorf("a theme is not a literal")
		}
		var p palette
		for _, kv := range lit.Elts {
			f, ok := kv.(*ast.KeyValueExpr)
			if !ok {
				return nil, fmt.Errorf("a theme field has no name")
			}
			key := f.Key.(*ast.Ident).Name
			var err error
			switch key {
			case "Dark":
				p.Dark = f.Value.(*ast.Ident).Name == "true"
			case "Notes":
				notes, ok := f.Value.(*ast.CompositeLit)
				if !ok || len(notes.Elts) != 6 {
					return nil, fmt.Errorf("%s: Notes needs six colours", p.Name)
				}
				for i, n := range notes.Elts {
					if p.Notes[i], err = str(n); err != nil {
						return nil, err
					}
				}
			case "Source":
			default:
				dst := map[string]*string{
					"Name": &p.Name, "Base": &p.Base, "Text": &p.Text, "Muted": &p.Muted, "Accent": &p.Accent,
					"Select": &p.Select, "Good": &p.Good, "Warn": &p.Warn, "Bad": &p.Bad, "Info": &p.Info,
				}[key]
				if dst == nil {
					return nil, fmt.Errorf("unknown field %s", key)
				}
				if *dst, err = str(f.Value); err != nil {
					return nil, err
				}
			}
		}
		out = append(out, p)
	}
	return out, nil
}
