// Package lineedit is a one-line text editor for the bottom line of the
// screen: a cursor, the keys of a shell's line editor, and a view that
// scrolls to keep the cursor in sight. It replaces a general text input
// with the part of one the board uses.
package lineedit

import (
	"strings"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Line is the text being typed and where the cursor is in it.
type Line struct {
	text  []rune
	pos   int // the cursor, between runes
	Width int // cells the view may take
	Caret lipgloss.Style
}

// New returns a line holding initial with the cursor at its end.
func New(initial string, width int) *Line {
	l := &Line{Width: width, Caret: lipgloss.NewStyle().Reverse(true)}
	l.Type(initial)
	return l
}

// Value is the text.
func (l *Line) Value() string { return string(l.text) }

// Type inserts text at the cursor. Line breaks in pasted text become
// spaces and other control characters are dropped.
func (l *Line) Type(s string) {
	var add []rune
	for _, r := range s {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			if len(add) == 0 || add[len(add)-1] != ' ' {
				add = append(add, ' ')
			}
		case !unicode.IsControl(r):
			add = append(add, r)
		}
	}
	l.text = append(l.text[:l.pos], append(add, l.text[l.pos:]...)...)
	l.pos += len(add)
}

// Key handles a named key, as bubbletea names it ("left", "ctrl+w"), and
// reports whether it was one the line knows.
func (l *Line) Key(key string) bool {
	switch key {
	case "left", "ctrl+b":
		l.pos = max(l.pos-1, 0)
	case "right", "ctrl+f":
		l.pos = min(l.pos+1, len(l.text))
	case "home", "ctrl+a":
		l.pos = 0
	case "end", "ctrl+e":
		l.pos = len(l.text)
	case "alt+left", "alt+b", "ctrl+left":
		l.pos = l.wordBefore()
	case "alt+right", "alt+f", "ctrl+right":
		l.pos = l.wordAfter()
	case "backspace", "ctrl+h":
		if l.pos > 0 {
			l.cut(l.pos-1, l.pos)
		}
	case "delete", "ctrl+d":
		if l.pos < len(l.text) {
			l.cut(l.pos, l.pos+1)
		}
	case "ctrl+w", "alt+backspace":
		l.cut(l.wordBefore(), l.pos)
	case "alt+d", "alt+delete":
		l.cut(l.pos, l.wordAfter())
	case "ctrl+u":
		l.cut(0, l.pos)
	case "ctrl+k":
		l.cut(l.pos, len(l.text))
	default:
		return false
	}
	return true
}

func (l *Line) cut(from, to int) {
	l.text = append(l.text[:from], l.text[to:]...)
	l.pos = from
}

// wordBefore is where the word before the cursor starts.
func (l *Line) wordBefore() int {
	i := l.pos
	for i > 0 && unicode.IsSpace(l.text[i-1]) {
		i--
	}
	for i > 0 && !unicode.IsSpace(l.text[i-1]) {
		i--
	}
	return i
}

// wordAfter is where the word after the cursor ends.
func (l *Line) wordAfter() int {
	i := l.pos
	for i < len(l.text) && unicode.IsSpace(l.text[i]) {
		i++
	}
	for i < len(l.text) && !unicode.IsSpace(l.text[i]) {
		i++
	}
	return i
}

// View draws the line in its width, scrolled so that the cursor shows, the
// cursor as a reversed cell.
func (l *Line) View() string {
	width := max(l.Width, 1)
	cells := func(rs []rune) int { return ansi.StringWidth(string(rs)) }
	// Start far enough back that the cursor and its cell fit.
	start := l.pos
	for start > 0 && cells(l.text[start-1:l.pos])+1 <= width {
		start--
	}
	var b strings.Builder
	used := 0
	for i := start; i <= len(l.text); i++ {
		ch := " "
		if i < len(l.text) {
			ch = string(l.text[i])
		}
		w := max(ansi.StringWidth(ch), 1)
		if used+w > width {
			break
		}
		if i == l.pos {
			b.WriteString(l.Caret.Render(ch))
		} else if i < len(l.text) {
			b.WriteString(ch)
		}
		used += w
	}
	return b.String()
}
