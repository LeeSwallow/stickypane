// Package editor is a small modal text editor in the manner of vi, for
// changing a note without leaving the board. It knows nothing about files:
// it holds the text, takes keys, and tells the caller when the user asked
// to save or quit.
package editor

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Mode is what the next key means.
type Mode int

const (
	Normal  Mode = iota // keys are commands
	Insert              // keys are text
	Command             // keys go to the line that starts with ":" or "/"
)

// Action is what a key asks the caller to do.
type Action int

const (
	None      Action = iota
	Save             // :w
	ForceSave        // :w!  write even if the file changed on disk
	Quit             // :q   on a buffer without changes
	ForceQuit        // :q!  drop the changes
	SaveQuit         // :wq, :x, ZZ
	Reload           // :e!  drop the changes and read the file again
)

const (
	defaultPage = 20   // lines a page has before the first View
	indentText  = "  " // what tab inserts
)

var (
	faint  = lipgloss.NewStyle().Faint(true)
	cursor = lipgloss.NewStyle().Reverse(true)
)

type snapshot struct {
	lines    [][]rune
	row, col int
}

// Editor is a text buffer with a cursor.
type Editor struct {
	lines    [][]rune
	row, col int
	top      int // the first line shown
	height   int // lines of the last View, which is one page

	mode    Mode
	pending string // a key waiting for the one that completes it: d c y g Z r
	prefix  rune   // ':' or '/', while in Command mode
	cmd     []rune
	search  string
	msg     string

	yank      string
	yankLines bool // the yanked text is whole lines

	undo, redo []snapshot
	saved      string // the text as last loaded or saved
	eol, crlf  bool   // the text ends with a newline; its lines end with "\r\n"
}

// New returns an editor holding text, with the cursor at its start.
func New(text string) *Editor {
	e := &Editor{}
	e.Load(text)
	return e
}

// Load replaces the buffer with text and forgets the history.
func (e *Editor) Load(text string) {
	e.saved = text
	e.crlf = strings.Contains(text, "\r\n")
	text = strings.ReplaceAll(text, "\r\n", "\n")
	e.eol = strings.HasSuffix(text, "\n")
	text = strings.TrimSuffix(text, "\n")
	e.lines = e.lines[:0]
	for _, l := range strings.Split(text, "\n") {
		e.lines = append(e.lines, []rune(l))
	}
	e.undo, e.redo, e.pending, e.mode = nil, nil, "", Normal
	e.row, e.col = min(e.row, len(e.lines)-1), 0
	e.clamp()
}

// Text returns the buffer as file content.
func (e *Editor) Text() string {
	parts := make([]string, len(e.lines))
	for i, l := range e.lines {
		parts[i] = string(l)
	}
	text := strings.Join(parts, "\n")
	if e.eol {
		text += "\n"
	}
	if e.crlf {
		text = strings.ReplaceAll(text, "\n", "\r\n")
	}
	return text
}

// Dirty reports whether the buffer differs from what was loaded or saved.
func (e *Editor) Dirty() bool { return e.Text() != e.saved }

// Saved tells the editor its text is now what the file holds.
func (e *Editor) Saved() { e.saved = e.Text() }

// Mode returns the current mode.
func (e *Editor) Mode() Mode { return e.mode }

// Message returns what the editor has to say about the last key, if anything.
func (e *Editor) Message() string { return e.msg }

// SetMessage shows a message until the next key.
func (e *Editor) SetMessage(s string) { e.msg = s }

// Scroll moves the cursor by delta lines, for the mouse wheel.
func (e *Editor) Scroll(delta int) {
	e.row += delta
	e.clamp()
}

// Status returns the two ends of a status line: what the editor is doing,
// and where the cursor is with the page it is on.
func (e *Editor) Status() (left, right string) {
	switch {
	case e.mode == Command:
		left = string(e.prefix) + string(e.cmd)
	case e.msg != "":
		left = e.msg
	case e.mode == Insert:
		left = "-- INSERT --"
	default:
		left = e.pending
	}
	page := e.page()
	return left, fmt.Sprintf("%d:%d  %d/%d", e.row+1, e.col+1, e.row/page+1, (len(e.lines)+page-1)/page)
}

func (e *Editor) page() int {
	if e.height > 0 {
		return e.height
	}
	return defaultPage
}

// clamp keeps the cursor inside the text. In Normal mode it stays on a
// character; in Insert mode it may stand after the last one.
func (e *Editor) clamp() {
	e.row = max(min(e.row, len(e.lines)-1), 0)
	last := len(e.lines[e.row])
	if e.mode != Insert {
		last--
	}
	e.col = max(min(e.col, last), 0)
}

func (e *Editor) snap() snapshot {
	lines := make([][]rune, len(e.lines))
	for i, l := range e.lines {
		lines[i] = append([]rune(nil), l...)
	}
	return snapshot{lines, e.row, e.col}
}

// change remembers the buffer so the edit that follows can be undone.
func (e *Editor) change() {
	e.undo = append(e.undo, e.snap())
	e.redo = nil
}

func (e *Editor) restore(from, to *[]snapshot) {
	if len(*from) == 0 {
		return
	}
	*to = append(*to, e.snap())
	s := (*from)[len(*from)-1]
	*from = (*from)[:len(*from)-1]
	e.lines, e.row, e.col = s.lines, s.row, s.col
}

// Key handles one key press: its name, such as "esc" or "ctrl+r", and the
// text it types, if any. It returns what the caller should do.
func (e *Editor) Key(key, text string) Action {
	e.msg = ""
	var act Action
	switch e.mode {
	case Insert:
		e.insertKey(key, text)
	case Command:
		act = e.commandKey(key, text)
	default:
		act = e.normalKey(key, text)
	}
	e.clamp()
	return act
}

func (e *Editor) insertKey(key, text string) {
	line := e.lines[e.row]
	switch key {
	case "esc", "ctrl+c":
		e.mode = Normal
		e.col--
	case "enter":
		rest := append([]rune(indentOf(line)), line[e.col:]...)
		e.lines[e.row] = append([]rune(nil), line[:e.col]...)
		e.lines = insertLine(e.lines, e.row+1, rest)
		e.row, e.col = e.row+1, len(indentOf(line))
	case "backspace":
		switch {
		case e.col > 0:
			e.lines[e.row] = append(append([]rune(nil), line[:e.col-1]...), line[e.col:]...)
			e.col--
		case e.row > 0:
			prev := e.lines[e.row-1]
			e.col = len(prev)
			e.lines[e.row-1] = append(append([]rune(nil), prev...), line...)
			e.lines = append(e.lines[:e.row], e.lines[e.row+1:]...)
			e.row--
		}
	case "delete":
		switch {
		case e.col < len(line):
			e.lines[e.row] = append(append([]rune(nil), line[:e.col]...), line[e.col+1:]...)
		case e.row+1 < len(e.lines):
			e.lines[e.row] = append(append([]rune(nil), line...), e.lines[e.row+1]...)
			e.lines = append(e.lines[:e.row+1], e.lines[e.row+2:]...)
		}
	case "left":
		e.col--
	case "right":
		e.col++
	case "up":
		e.row--
	case "down":
		e.row++
	case "home":
		e.col = 0
	case "end":
		e.col = len(line)
	case "tab":
		e.insert(indentText)
	default:
		e.insert(text)
	}
}

// insert puts text at the cursor. Control characters are left out.
func (e *Editor) insert(text string) {
	var typed []rune
	for _, r := range text {
		if !unicode.IsControl(r) {
			typed = append(typed, r)
		}
	}
	line := e.lines[e.row]
	col := min(e.col, len(line))
	out := make([]rune, 0, len(line)+len(typed))
	out = append(append(append(out, line[:col]...), typed...), line[col:]...)
	e.lines[e.row] = out
	e.col = col + len(typed)
}

func insertLine(lines [][]rune, at int, line []rune) [][]rune {
	lines = append(lines, nil)
	copy(lines[at+1:], lines[at:])
	lines[at] = line
	return lines
}

// indentOf returns the spaces and tabs a line starts with.
func indentOf(line []rune) string {
	n := 0
	for n < len(line) && (line[n] == ' ' || line[n] == '\t') {
		n++
	}
	return string(line[:n])
}

// startInsert enters Insert mode as one undoable change.
func (e *Editor) startInsert() {
	e.change()
	e.mode = Insert
}

func (e *Editor) normalKey(key, text string) Action {
	if p := e.pending; p != "" {
		e.pending = ""
		return e.complete(p, key, text)
	}
	line := e.lines[e.row]
	half := max(e.page()/2, 1)
	switch key {
	case "h", "left", "backspace":
		e.col--
	case "l", "right", "space":
		e.col++
	case "j", "down", "enter":
		e.row++
	case "k", "up":
		e.row--
	case "0", "home":
		e.col = 0
	case "^":
		e.col = len(indentOf(line))
	case "$", "end":
		e.col = len(line)
	case "w":
		e.row, e.col = e.nextWord(e.row, e.col)
	case "b":
		e.row, e.col = e.prevWord(e.row, e.col)
	case "e":
		e.row, e.col = e.wordEnd(e.row, e.col)
	case "G":
		e.row = len(e.lines) - 1
	case "ctrl+d":
		e.row += half
	case "ctrl+u":
		e.row -= half
	case "ctrl+f", "pgdown":
		e.row += e.page()
	case "ctrl+b", "pgup":
		e.row -= e.page()
	case "i":
		e.startInsert()
	case "a":
		e.startInsert()
		e.col = min(e.col+1, len(line))
	case "I":
		e.startInsert()
		e.col = len(indentOf(line))
	case "A":
		e.startInsert()
		e.col = len(line)
	case "o", "O":
		e.startInsert()
		at := e.row + 1
		if key == "O" {
			at = e.row
		}
		indent := indentOf(line)
		e.lines = insertLine(e.lines, at, []rune(indent))
		e.row, e.col = at, len([]rune(indent))
	case "x", "delete":
		if len(line) > 0 {
			e.change()
			e.cut(e.col, e.col+1)
		}
	case "D":
		e.change()
		e.cut(e.col, len(line))
	case "C":
		e.startInsert()
		e.cut(e.col, len(line))
		e.col = len(e.lines[e.row])
	case "J":
		if e.row+1 < len(e.lines) {
			e.change()
			next := []rune(strings.TrimLeft(string(e.lines[e.row+1]), " \t"))
			joined := append([]rune(nil), line...)
			if len(joined) > 0 && len(next) > 0 {
				joined = append(joined, ' ')
			}
			e.col = max(len(joined)-1, 0)
			e.lines[e.row] = append(joined, next...)
			e.lines = append(e.lines[:e.row+1], e.lines[e.row+2:]...)
		}
	case "p", "P":
		e.paste(key == "P")
	case "u":
		e.restore(&e.undo, &e.redo)
	case "ctrl+r":
		e.restore(&e.redo, &e.undo)
	case "d", "c", "y", "g", "Z", "r":
		e.pending = key
	case ":", "/":
		e.mode, e.prefix, e.cmd = Command, []rune(key)[0], nil
	case "n":
		e.find(e.search, true)
	case "N":
		e.find(e.search, false)
	}
	return None
}

// complete handles the key after d, c, y, g, Z or r. A key that completes
// nothing cancels the pending one.
func (e *Editor) complete(pending, key, text string) Action {
	line := e.lines[e.row]
	switch pending + key {
	case "gg":
		e.row = 0
	case "ZZ":
		return SaveQuit
	case "ZQ":
		return ForceQuit
	case "yy":
		e.yank, e.yankLines = string(line), true
	case "dd":
		e.change()
		e.yank, e.yankLines = string(line), true
		e.lines = append(e.lines[:e.row], e.lines[e.row+1:]...)
		if len(e.lines) == 0 {
			e.lines = [][]rune{{}}
		}
	case "dw":
		e.change()
		end := len(line)
		if r, c := e.nextWord(e.row, e.col); r == e.row && c > e.col {
			end = c
		}
		e.cut(e.col, end)
	case "d$":
		e.change()
		e.cut(e.col, len(line))
	case "cc":
		e.startInsert()
		e.yank, e.yankLines = string(line), true
		e.lines[e.row], e.col = []rune{}, 0
	case "cw":
		e.startInsert()
		r, end := e.wordEnd(e.row, e.col-1)
		if r != e.row {
			end = len(line) - 1
		}
		at := e.col
		e.cut(at, end+1)
		e.col = at
	case "c$":
		e.startInsert()
		e.cut(e.col, len(line))
		e.col = len(e.lines[e.row])
	default:
		if r := []rune(text); pending == "r" && len(r) == 1 && e.col < len(line) && !unicode.IsControl(r[0]) {
			e.change()
			e.lines[e.row] = append([]rune(nil), line...)
			e.lines[e.row][e.col] = r[0]
		}
	}
	return None
}

// cut removes the characters from..to of the current line and yanks them.
func (e *Editor) cut(from, to int) {
	line := e.lines[e.row]
	from, to = max(min(from, len(line)), 0), max(min(to, len(line)), 0)
	if from >= to {
		return
	}
	e.yank, e.yankLines = string(line[from:to]), false
	e.lines[e.row] = append(append([]rune(nil), line[:from]...), line[to:]...)
	e.col = from
}

// paste puts the yanked text after the cursor, or before it.
func (e *Editor) paste(before bool) {
	if e.yank == "" && !e.yankLines {
		return
	}
	e.change()
	if e.yankLines {
		at := e.row + 1
		if before {
			at = e.row
		}
		e.lines = insertLine(e.lines, at, []rune(e.yank))
		e.row, e.col = at, 0
		return
	}
	line := e.lines[e.row]
	at := min(e.col+1, len(line))
	if before {
		at = e.col
	}
	text := []rune(e.yank)
	out := append(append(append([]rune(nil), line[:at]...), text...), line[at:]...)
	e.lines[e.row] = out
	e.col = at + len(text) - 1
}

// class sorts a character for word motions: space, word, or punctuation.
func class(r rune) int {
	switch {
	case unicode.IsSpace(r):
		return 0
	case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r > unicode.MaxASCII:
		return 1
	}
	return 2
}

// nextWord returns where the next word starts.
func (e *Editor) nextWord(row, col int) (int, int) {
	line := e.lines[row]
	if col < len(line) {
		if c := class(line[col]); c != 0 {
			for col < len(line) && class(line[col]) == c {
				col++
			}
		}
		for col < len(line) && class(line[col]) == 0 {
			col++
		}
		if col < len(line) {
			return row, col
		}
	}
	for row+1 < len(e.lines) {
		row++
		line, col = e.lines[row], 0
		for col < len(line) && class(line[col]) == 0 {
			col++
		}
		if col < len(line) || len(line) == 0 {
			return row, col
		}
	}
	return row, max(len(e.lines[row])-1, 0)
}

// prevWord returns where the word before the cursor starts.
func (e *Editor) prevWord(row, col int) (int, int) {
	for {
		line := e.lines[row]
		col = min(col, len(line)) - 1
		for col >= 0 && class(line[col]) == 0 {
			col--
		}
		if col >= 0 {
			for c := class(line[col]); col > 0 && class(line[col-1]) == c; {
				col--
			}
			return row, col
		}
		if row == 0 {
			return 0, 0
		}
		row--
		col = len(e.lines[row])
	}
}

// wordEnd returns where the word at or after the cursor ends.
func (e *Editor) wordEnd(row, col int) (int, int) {
	col++
	for {
		line := e.lines[row]
		for col < len(line) && class(line[col]) == 0 {
			col++
		}
		if col < len(line) {
			for c := class(line[col]); col+1 < len(line) && class(line[col+1]) == c; {
				col++
			}
			return row, col
		}
		if row+1 >= len(e.lines) {
			return row, max(len(line)-1, 0)
		}
		row, col = row+1, 0
	}
}

// find moves to the next or previous place the text occurs, wrapping
// around the ends of the buffer.
func (e *Editor) find(text string, forward bool) {
	if text == "" {
		return
	}
	needle := []rune(text)
	n := len(e.lines)
	for step := 0; step <= n; step++ {
		row := (e.row + step) % n
		if !forward {
			row = ((e.row-step)%n + n) % n
		}
		line := e.lines[row]
		best := -1
		for col := 0; col+len(needle) <= len(line); col++ {
			if string(line[col:col+len(needle)]) != text {
				continue
			}
			switch {
			case forward && (step > 0 || col > e.col):
				e.row, e.col = row, col
				return
			case !forward && (step > 0 || col < e.col):
				best = col // keep the last one before the cursor
			}
		}
		if best >= 0 {
			e.row, e.col = row, best
			return
		}
	}
	e.msg = "Pattern not found: " + text
}

func (e *Editor) commandKey(key, text string) Action {
	switch key {
	case "esc", "ctrl+c":
		e.mode = Normal
	case "backspace":
		if len(e.cmd) == 0 {
			e.mode = Normal
			break
		}
		e.cmd = e.cmd[:len(e.cmd)-1]
	case "enter":
		e.mode = Normal
		line := strings.TrimSpace(string(e.cmd))
		if e.prefix == '/' {
			e.search = line
			e.find(line, true)
			return None
		}
		return e.run(line)
	default:
		for _, r := range text {
			if !unicode.IsControl(r) {
				e.cmd = append(e.cmd, r)
			}
		}
	}
	return None
}

// run carries out a ":" command.
func (e *Editor) run(cmd string) Action {
	switch cmd {
	case "":
	case "w":
		return Save
	case "w!":
		return ForceSave
	case "wq", "x":
		return SaveQuit
	case "q!":
		return ForceQuit
	case "e!":
		return Reload
	case "q":
		if !e.Dirty() {
			return Quit
		}
		e.msg = "No write since last change (:q! drops the changes, :wq saves them)"
	default:
		if n, err := strconv.Atoi(cmd); err == nil {
			e.row, e.col = n-1, 0
			break
		}
		e.msg = "Not an editor command: " + cmd
	}
	return None
}

// show returns how a character is drawn.
func show(r rune) string {
	switch {
	case r == '\t':
		return indentText
	case unicode.IsControl(r):
		return ""
	}
	return string(r)
}

// View draws the buffer in exactly height lines of at most width cells:
// line numbers, the text with long lines wrapped, the cursor in reverse
// video, and "~" below the last line. It scrolls to keep the cursor in view.
func (e *Editor) View(width, height int) []string {
	e.clamp()
	if height <= 0 {
		return nil
	}
	e.height = height
	out := make([]string, height)
	if width <= 0 {
		return out
	}
	gutter := len(strconv.Itoa(len(e.lines))) + 1
	if width < gutter+2 {
		gutter = 0
	}
	text := width - gutter

	e.top = max(min(e.top, e.row), 0)
	var rows []string
	for {
		var at int
		rows, at = e.rows(e.top, text, gutter, height)
		if at < height || e.top >= e.row {
			break
		}
		e.top++
	}
	for i := range out {
		switch {
		case i < len(rows):
			out[i] = ansi.Truncate(rows[i], width, "")
		default:
			out[i] = faint.Render("~")
		}
	}
	return out
}

// rows draws the lines from top until limit rows are filled and the cursor
// has been drawn. It returns the rows and which one holds the cursor.
func (e *Editor) rows(top, width, gutter, limit int) (rows []string, at int) {
	at = -1
	for i := top; i < len(e.lines) && (len(rows) < limit || at < 0); i++ {
		if i > e.row && len(rows) >= limit {
			break
		}
		number := ""
		if gutter > 0 {
			number = fmt.Sprintf("%*d ", gutter-1, i+1)
		}
		var b strings.Builder
		used, first := 0, true
		flush := func() {
			lead := strings.Repeat(" ", gutter)
			if first {
				lead = faint.Render(number)
			}
			rows = append(rows, lead+b.String())
			b.Reset()
			used, first = 0, false
		}
		line := e.lines[i]
		for col := 0; col <= len(line); col++ {
			here := i == e.row && col == e.col
			cell := " " // the cursor after the last character
			if col < len(line) {
				cell = show(line[col])
			} else if !here {
				break
			}
			w := ansi.StringWidth(cell)
			if used > 0 && used+w > width {
				flush()
			}
			if here {
				at = len(rows)
				cell = cursor.Render(cell)
			}
			b.WriteString(cell)
			used += w
		}
		flush()
	}
	return rows, at
}
