package test

import (
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// linkSkip are the Markdown files whose links are not checked: the dated
// plans and specs, kept as they were written, and the changelog, whose
// links are to releases.
var linkSkip = []string{"docs/history/", "CHANGELOG.md"}

var (
	// [text](target), ![alt](target), with an optional "title".
	inlineLink = regexp.MustCompile(`!?\[[^\]]*\]\(\s*<?([^)\s>]+)>?(?:\s+"[^"]*")?\s*\)`)
	// [name]: target, a reference link's definition.
	refLink = regexp.MustCompile(`(?m)^\s{0,3}\[[^\]]+\]:\s+<?(\S+?)>?(?:\s|$)`)
	// <img src="target">, <a href="target">, <source srcset="target">.
	htmlLink = regexp.MustCompile(`(?i)\b(?:src|href|srcset)="([^"]+)"`)
	// a scheme: https:, mailto:...
	scheme = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)
	// `code`, whose brackets are not links.
	codeSpan = regexp.MustCompile("`+[^`]*`+")
)

// Every relative link and image in the repository's Markdown names a file
// or folder that is there, so a moved file cannot leave a reader at a 404.
func TestMarkdownLinksResolve(t *testing.T) {
	root, err := filepath.Abs(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := filepath.ToSlash(strings.TrimPrefix(path, root+string(filepath.Separator)))
		if d.IsDir() {
			name := d.Name()
			if path != root && (strings.HasPrefix(name, ".") || name == "node_modules" || name == "testdata" || name == "dist") {
				return filepath.SkipDir
			}
			if skipped(rel + "/") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name(d), ".md") || skipped(rel) {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, target := range links(string(b)) {
			checked++
			if broken(root, filepath.Dir(path), target) {
				t.Errorf("%s links %s, which is not there", rel, target)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked == 0 {
		t.Fatal("no links were checked: the walk found no Markdown")
	}
}

func name(d fs.DirEntry) string { return strings.ToLower(d.Name()) }

func skipped(rel string) bool {
	for _, s := range linkSkip {
		if strings.HasPrefix(rel, s) {
			return true
		}
	}
	return false
}

// links are the targets of the links and images in a Markdown text,
// outside code blocks and code spans.
func links(text string) []string {
	var out []string
	fence := ""
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if fence != "" {
			if strings.HasPrefix(trimmed, fence) {
				fence = ""
			}
			continue
		}
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			fence = trimmed[:3]
			continue
		}
		line = codeSpan.ReplaceAllString(line, "")
		for _, re := range []*regexp.Regexp{inlineLink, refLink, htmlLink} {
			for _, m := range re.FindAllStringSubmatch(line, -1) {
				out = append(out, m[1])
			}
		}
	}
	return out
}

// broken says whether a link's target, read from the folder dir, is a
// relative path to nothing. A link with a scheme or only an anchor is not
// a file, and is not checked.
func broken(root, dir, target string) bool {
	if scheme.MatchString(target) || strings.HasPrefix(target, "#") || strings.HasPrefix(target, "//") {
		return false
	}
	target, _, _ = strings.Cut(target, "#")
	target, _, _ = strings.Cut(target, "?")
	if p, err := url.PathUnescape(target); err == nil {
		target = p
	}
	if target == "" {
		return false
	}
	path := filepath.Join(dir, filepath.FromSlash(target))
	if strings.HasPrefix(target, "/") {
		path = filepath.Join(root, filepath.FromSlash(target))
	}
	_, err := os.Stat(path)
	return err != nil
}
