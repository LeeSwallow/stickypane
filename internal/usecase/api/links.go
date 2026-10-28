package api

import (
	"path"
	"regexp"
	"slices"
	"strings"
)

// linkRe finds [[name]], [[name#item]] and [[name|alias]] (and the same
// with a "!" before it, an embed).
var linkRe = regexp.MustCompile(`\[\[([^\]|#]+)(?:#[^\]|]*)?(?:\|[^\]]*)?\]\]`)

// LinkTargets are the names a body links to, as written, in order, once
// each.
func LinkTargets(body string) []string {
	var out []string
	for _, m := range linkRe.FindAllStringSubmatch(body, -1) {
		if t := strings.TrimSpace(m[1]); t != "" && !slices.Contains(out, t) {
			out = append(out, t)
		}
	}
	return out
}

// Resolve finds the note a link names among the board's files: the name
// itself, with ".md", or the one file whose name without folder and
// extension it is. "" when none does, or more than one could.
func Resolve(target string, files []string) string {
	target = strings.TrimSpace(target)
	for _, f := range files {
		if f == target || f == target+".md" {
			return f
		}
	}
	found := ""
	for _, f := range files {
		base := path.Base(f)
		if strings.TrimSuffix(base, path.Ext(base)) == target || base == target {
			if found != "" {
				return ""
			}
			found = f
		}
	}
	return found
}

// linkUp resolves each entry's links and fills in the backlinks.
func linkUp(entries []Entry) {
	files := make([]string, len(entries))
	for i, e := range entries {
		files[i] = e.Name
	}
	back := map[string][]string{}
	for i := range entries {
		e := &entries[i]
		for _, t := range LinkTargets(e.body) {
			if f := Resolve(t, files); f != "" && f != e.Name && !slices.Contains(e.Links, f) {
				e.Links = append(e.Links, f)
				back[f] = append(back[f], e.Name)
			}
		}
		e.body = ""
	}
	for i := range entries {
		entries[i].Backlinks = back[entries[i].Name]
	}
}
