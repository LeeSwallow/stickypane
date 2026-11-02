// Package sshconfig lists the hosts a person's ssh config names, for the
// app to offer them. It reads ~/.ssh/config and what it includes, and only
// lists aliases: where an alias goes is ssh's own answer (ssh -G), and
// nothing here reads keys, passwords or any other setting, or writes.
package sshconfig

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// maxDepth bounds Include within Include (ssh's own limit is 16).
const maxDepth = 16

// Aliases are the names of the Host lines in home/.ssh/config and the
// files it includes, in order, each once. A pattern (*, ?, !) is not a
// host one can go to, and Match blocks name none. A file that cannot be
// read is passed over.
func Aliases(home string) []string {
	r := reader{home: home, sshDir: filepath.Join(home, ".ssh"), seen: map[string]bool{}}
	r.file(filepath.Join(r.sshDir, "config"), 0)
	return r.out
}

type reader struct {
	home, sshDir string
	seen         map[string]bool
	out          []string
}

func (r *reader) file(path string, depth int) {
	if depth > maxDepth {
		return
	}
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		key, args := split(sc.Text())
		switch strings.ToLower(key) {
		case "host":
			for _, a := range args {
				if strings.ContainsAny(a, "*?!") || r.seen[a] {
					continue
				}
				r.seen[a] = true
				r.out = append(r.out, a)
			}
		case "include":
			for _, a := range args {
				r.include(a, depth+1)
			}
		}
	}
}

// include reads the files a pattern names: ~ is the home, and a relative
// path is under ~/.ssh, as ssh reads the user's config.
func (r *reader) include(pattern string, depth int) {
	switch {
	case pattern == "~":
		pattern = r.home
	case strings.HasPrefix(pattern, "~/"):
		pattern = filepath.Join(r.home, pattern[2:])
	case !filepath.IsAbs(pattern):
		pattern = filepath.Join(r.sshDir, pattern)
	}
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return
	}
	for _, p := range paths {
		if fi, err := os.Stat(p); err == nil && fi.Mode().IsRegular() {
			r.file(p, depth)
		}
	}
}

// split reads one line: a keyword, then its arguments, which may be in
// double quotes. "Keyword=value" and "Keyword = value" are the same as
// "Keyword value"; # starts a comment.
func split(line string) (string, []string) {
	line = strings.TrimSpace(line)
	if line == "" || line[0] == '#' {
		return "", nil
	}
	i := strings.IndexAny(line, " \t=")
	if i < 0 {
		return line, nil
	}
	key, rest := line[:i], strings.TrimLeft(line[i:], " \t")
	rest = strings.TrimLeft(strings.TrimPrefix(rest, "="), " \t")
	var args []string
	for rest != "" {
		if rest[0] == '#' {
			break
		}
		var a string
		if rest[0] == '"' {
			end := strings.IndexByte(rest[1:], '"')
			if end < 0 {
				a, rest = rest[1:], ""
			} else {
				a, rest = rest[1:end+1], rest[end+2:]
			}
		} else {
			end := strings.IndexAny(rest, " \t")
			if end < 0 {
				a, rest = rest, ""
			} else {
				a, rest = rest[:end], rest[end:]
			}
		}
		if a != "" {
			args = append(args, a)
		}
		rest = strings.TrimLeft(rest, " \t")
	}
	return key, args
}
