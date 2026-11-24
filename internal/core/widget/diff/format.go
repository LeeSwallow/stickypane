package diff

import (
	"strconv"
	"strings"
)

// Parse reads a unified diff as git writes it: a "diff --git" line for
// each file, the lines that say how the file changed (new, deleted,
// renamed, its mode, binary), then its @@ hunks. A hunk's header says how
// many lines it holds, so a removed line that reads "-- x" is never taken
// for the "--- a/x" of the next file. What it does not know it skips.
func Parse(text string) []File {
	var files []File
	var f *File
	var h *Hunk
	oldLeft, newLeft := 0, 0
	for _, l := range strings.Split(strings.TrimSuffix(text, "\n"), "\n") {
		if h != nil && (oldLeft > 0 || newLeft > 0 || strings.HasPrefix(l, `\`)) {
			op, rest := byte(' '), "" // a blank line in a hunk is a kept blank line
			if l != "" {
				op, rest = l[0], l[1:]
			}
			switch op {
			case ' ':
				oldLeft, newLeft = oldLeft-1, newLeft-1
			case '-':
				oldLeft--
			case '+':
				newLeft--
			case '\\':
			default:
				op = 0
			}
			if op != 0 {
				h.Lines = append(h.Lines, Line{Op: op, Text: rest})
				continue
			}
		}
		h = nil
		switch {
		case strings.HasPrefix(l, "diff --git "):
			files = append(files, File{Status: "M"})
			f = &files[len(files)-1]
			f.Old, f.New = headerPaths(l[len("diff --git "):])
		case f == nil:
		case strings.HasPrefix(l, "@@ "):
			f.Hunks = append(f.Hunks, Hunk{Header: l})
			h = &f.Hunks[len(f.Hunks)-1]
			oldLeft, newLeft = hunkSizes(l)
		case strings.HasPrefix(l, "--- "):
			f.Old = side(l[4:], "a/")
		case strings.HasPrefix(l, "+++ "):
			f.New = side(l[4:], "b/")
		case strings.HasPrefix(l, "new file mode "):
			f.Status, f.Old = "A", ""
		case strings.HasPrefix(l, "deleted file mode "):
			f.Status, f.New = "D", ""
		case strings.HasPrefix(l, "old mode "):
			f.OldMode = l[len("old mode "):]
		case strings.HasPrefix(l, "new mode "):
			f.NewMode = l[len("new mode "):]
		case strings.HasPrefix(l, "rename from "):
			f.Status, f.Old = "R", unquote(l[len("rename from "):])
		case strings.HasPrefix(l, "rename to "):
			f.New = unquote(l[len("rename to "):])
		case strings.HasPrefix(l, "copy from "):
			f.Status, f.Old = "C", unquote(l[len("copy from "):])
		case strings.HasPrefix(l, "copy to "):
			f.New = unquote(l[len("copy to "):])
		case strings.HasPrefix(l, "similarity index "):
			f.Similarity = l[len("similarity index "):]
		case strings.HasPrefix(l, "Binary files "), l == "GIT binary patch":
			f.Binary = true
		}
	}
	return files
}

// headerPaths reads the two paths of a "diff --git a/x b/y" line. A path
// with a space is ambiguous there; the lines after the header (---, +++,
// rename) say it again where it matters, and a file whose name did not
// change has two halves that are the same.
func headerPaths(s string) (old, new string) {
	if strings.HasPrefix(s, `"`) {
		if q, err := strconv.QuotedPrefix(s); err == nil {
			return side(q, "a/"), side(strings.TrimSpace(s[len(q):]), "b/")
		}
	}
	if len(s)%2 == 1 {
		a, b := s[:len(s)/2], s[len(s)/2+1:]
		if strings.TrimPrefix(a, "a/") == strings.TrimPrefix(b, "b/") {
			return side(a, "a/"), side(b, "b/")
		}
	}
	if i := strings.LastIndex(s, " b/"); i >= 0 {
		return side(s[:i], "a/"), side(s[i+1:], "b/")
	}
	return s, s
}

// side is a path of a ---, +++ or header line without its a/ or b/, ""
// for /dev/null.
func side(p, prefix string) string {
	p = unquote(p)
	if p == "/dev/null" {
		return ""
	}
	return strings.TrimPrefix(p, prefix)
}

// unquote reads a path git put in quotes because of a character it
// escapes; any other path is as it is.
func unquote(p string) string {
	if strings.HasPrefix(p, `"`) {
		if u, err := strconv.Unquote(p); err == nil {
			return u
		}
	}
	return p
}

// hunkSizes reads how many old and new lines a hunk holds from its header,
// "@@ -41,7 +41,9 @@": 7 and 9. A count left out is 1.
func hunkSizes(header string) (old, new int) {
	f := strings.Fields(header)
	if len(f) < 3 {
		return 0, 0
	}
	size := func(r string) int {
		_, n, ok := strings.Cut(r[1:], ",")
		if !ok {
			return 1
		}
		v, _ := strconv.Atoi(n)
		return v
	}
	return size(f[1]), size(f[2])
}
