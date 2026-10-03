// Package arrange decides how a note is arranged on the screen: whether it
// is open, how wide and tall it is, whether it is pinned, its color and its
// name. It is the personal layer of the board, kept apart from what a note
// is (its kind, in package widget) and from where it is kept (package
// store).
//
// Every answer comes from three places, in this order: what the user did on
// the board, kept in sticky.json (a store.View); the note's own front
// matter, which is how an agent proposes an arrangement; and the note's
// kind. The screen and the command line both ask here, so they never
// disagree.
package arrange

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/theme"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// ValidSize reports whether s is a size a note can take.
func ValidSize(s string) bool {
	return s == widget.SizePage || s == widget.SizeHalf || s == widget.SizeCard
}

// Open reports whether the note is open, and whether anything said so at
// all. When nothing said, the screen may still show the note for a while
// (one that just appeared); that is the caller's business.
func Open(v store.View, front doc.Document) (open, said bool) {
	if v.Open != nil {
		return *v.Open, true
	}
	if s, ok := front.Get("open"); ok {
		return strings.EqualFold(s, "true"), true
	}
	return false, false
}

// Size returns the note's size. content is what the kind measures: for a
// book, the page in view; otherwise the note itself.
func Size(v store.View, front doc.Document, k widget.Kind, content doc.Document) string {
	if s := strings.ToLower(v.Size); ValidSize(s) {
		return s
	}
	if s, _ := front.Get("size"); ValidSize(strings.ToLower(s)) {
		return strings.ToLower(s)
	}
	if k.Size != nil {
		return k.Size(content)
	}
	return widget.SizePage
}

// Rows returns the height in lines the note asks for, or 0 when it asks for
// as much as its content takes.
func Rows(v store.View, front doc.Document, k widget.Kind) int {
	if v.Rows > 0 {
		return v.Rows
	}
	if s, ok := front.Get("rows"); ok {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			return n
		}
	}
	return k.Rows
}

// Pinned reports whether the note is kept first.
func Pinned(v store.View, front doc.Document) bool {
	if v.Pin != nil {
		return *v.Pin
	}
	return front.Pinned()
}

// Color returns the name of the note's chosen color, or "" when none was
// chosen and the screen derives one.
func Color(v store.View, front doc.Document) string {
	if v.Color != "" {
		return v.Color
	}
	c, _ := front.Get("color")
	return c
}

// Title names a note: the name sticky.json gives it, else its front matter
// title, else its file name without the folder and the extension. The text
// is as written; the screen cleans it before drawing.
func Title(v store.View, front doc.Document, name string) string {
	if v.Title != "" {
		return v.Title
	}
	if t, _ := front.Get("title"); t != "" {
		return t
	}
	return FileTitle(name)
}

// FileTitle is a file's name without the folder, the extension and the
// number in front that orders the notes ("10-plan.md" is "plan"). A name
// that is only numbers, such as a date, stays as it is.
func FileTitle(name string) string {
	base := path.Base(name)
	base = strings.TrimSuffix(base, path.Ext(base))
	if m := orderRe.FindStringSubmatch(base); m != nil {
		return m[1]
	}
	return base
}

// orderRe is a name with a number in front to order it: digits, a dash,
// an underscore or a space, then something that does not start with a
// digit.
var orderRe = regexp.MustCompile(`^\d+[-_ ]+(\D.*)$`)

// Change returns the change a key about the arrangement makes to a view,
// as `stickypane set` and the MCP server take it, or false when the key
// is about something else. An empty value takes the
// key back out, so the note arranges itself again.
func Change(key, value string) (func(*store.View), bool, error) {
	flag := func(set func(*store.View, *bool)) (func(*store.View), bool, error) {
		switch strings.ToLower(value) {
		case "":
			return func(v *store.View) { set(v, nil) }, true, nil
		case "true", "false":
			b := strings.EqualFold(value, "true")
			return func(v *store.View) { set(v, &b) }, true, nil
		}
		return nil, true, fmt.Errorf("%s is true or false, not %q", key, value)
	}
	switch key {
	case "open":
		return flag(func(v *store.View, b *bool) { v.Open = b })
	case "pin":
		return flag(func(v *store.View, b *bool) { v.Pin = b })
	case "size":
		if value != "" && !ValidSize(value) {
			return nil, true, fmt.Errorf("unknown size %q: use page, half or card", value)
		}
		return func(v *store.View) { v.Size = value }, true, nil
	case "rows":
		n := 0
		if value != "" {
			var err error
			if n, err = strconv.Atoi(value); err != nil || n < 1 {
				return nil, true, fmt.Errorf("rows is a number of lines, not %q", value)
			}
		}
		return func(v *store.View) { v.Rows = n }, true, nil
	case "color":
		if value != "" && !slices.Contains(theme.NoteNames[:], strings.ToLower(value)) {
			return nil, true, fmt.Errorf("unknown color %q: use %s", value, strings.Join(theme.NoteNames[:], ", "))
		}
		return func(v *store.View) { v.Color = strings.ToLower(value) }, true, nil
	}
	return nil, false, nil
}
