package api

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/widget/board"
	"github.com/LeeSwallow/stickypane/internal/widget/chart"
	"github.com/LeeSwallow/stickypane/internal/widget/checklist"
	"github.com/LeeSwallow/stickypane/internal/widget/logview"
)

// The functions in this file change one thing in a note: tick an item, move
// a card, count a value up, add a log line. The caller names the note and
// the thing; it does not read the note first and does not need to know the
// file format. Each returns one line saying where the note stands now, so
// the caller does not have to read it afterwards either.

// shaped is an Op that applies only to a note of one of the given kinds.
type shaped struct {
	api   *API
	file  string
	kinds []string
	op    doc.Op
}

func (s shaped) Apply(d doc.Document) (doc.Document, error) {
	if have := s.api.reg.For(s.file, d).Name; !slices.Contains(s.kinds, have) {
		return d, fmt.Errorf("%s is a %s, not a %s", s.file, have, s.kinds[0])
	}
	return s.op.Apply(d)
}

// change applies op to the note. A note that does not exist yet is made as
// the first of kinds, open on the screen. It returns the file name and the
// note as it is afterwards.
func (a *API) change(name string, kinds []string, op doc.Op) (string, doc.Document, error) {
	file, err := fileName(name)
	if err != nil {
		return "", doc.Document{}, err
	}
	if _, err := a.st.Read(file); errors.Is(err, fs.ErrNotExist) {
		// Only Markdown says in front matter what it is and that it is
		// open. Another file is what its extension makes it.
		var d doc.Document
		if strings.EqualFold(filepath.Ext(file), ".md") {
			base := filepath.Base(file)
			d = doc.Parse(a.reg.Lookup(kinds[0]).Template(strings.TrimSuffix(base, filepath.Ext(base)))).Set("open", "true")
		} else if have := a.reg.For(file, d).Name; !slices.Contains(kinds, have) {
			return file, d, fmt.Errorf("%s would be a %s, not a %s", file, have, kinds[0])
		}
		if d, err = op.Apply(d); err != nil {
			return file, d, err
		}
		return file, d, a.st.Write(file, d.Bytes())
	}
	if err := a.st.Apply(file, shaped{a, file, kinds, op}); err != nil {
		return file, doc.Document{}, err
	}
	b, err := a.st.Read(file)
	return file, doc.Parse(b), err
}

// status is the line an edit returns: the file and its summary.
func (a *API) status(file string, d doc.Document) string {
	if s := a.reg.For(file, d).Parse(d).Summary(); s != "" {
		return file + ": " + s
	}
	return file
}

// oneLine keeps text that becomes a line of a file to one line.
func oneLine(s string) (string, error) {
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return "", errors.New("the text is empty")
	}
	return s, nil
}

// Todo changes a checklist: action "add" appends item, "check" and
// "uncheck" set the item named by its text, a part of it, or "#2".
func (a *API) Todo(name, action, item string) (string, error) {
	item, err := oneLine(item)
	if err != nil {
		return "", err
	}
	var op doc.Op
	switch action {
	case "add":
		op = checklist.AddItem{Text: item}
	case "check", "uncheck":
		op = checklist.Check{Item: item, Checked: action == "check"}
	default:
		return "", fmt.Errorf("unknown action %q: use add, check or uncheck", action)
	}
	file, d, err := a.change(name, []string{"checklist"}, op)
	if err != nil {
		return "", err
	}
	return a.status(file, d), nil
}

// Card changes a board: action "add" puts a new card in column to (the
// first column when to is empty), "move" moves the card named by its text,
// a part of it, or "#2" to column to.
func (a *API) Card(name, action, card, to string) (string, error) {
	card, err := oneLine(card)
	if err != nil {
		return "", err
	}
	var op doc.Op
	switch action {
	case "add":
		op = board.Add{Card: card, To: strings.TrimSpace(to)}
	case "move":
		if strings.TrimSpace(to) == "" {
			return "", errors.New("a move needs the column to move the card to")
		}
		op = board.Move{Card: card, To: to}
	default:
		return "", fmt.Errorf("unknown action %q: use add or move", action)
	}
	file, d, err := a.change(name, []string{"board"}, op)
	if err != nil {
		return "", err
	}
	return a.status(file, d), nil
}

// Chart changes one value of a chart: action "set" writes value as given,
// "add" counts the value up by it (down when it is negative).
func (a *API) Chart(name, action, label, value string) (string, error) {
	label, err := oneLine(label)
	if err != nil {
		return "", err
	}
	var op doc.Op
	switch action {
	case "set":
		op = chart.Set{Label: label, Value: value}
	case "add":
		delta, err := strconv.ParseFloat(strings.NewReplacer(",", "", "_", "").Replace(strings.TrimSpace(value)), 64)
		if err != nil {
			return "", fmt.Errorf("%q is not a number", value)
		}
		op = chart.Add{Label: label, Delta: delta}
	default:
		return "", fmt.Errorf("unknown action %q: use set or add", action)
	}
	file, d, err := a.change(name, []string{"chart"}, op)
	if err != nil {
		return "", err
	}
	now, _ := chart.Value(d, label)
	return fmt.Sprintf("%s: %s = %s", file, label, now), nil
}

// Log adds one line at the end of a log, or of a plain note.
func (a *API) Log(name, line string) (string, error) {
	line, err := oneLine(line)
	if err != nil {
		return "", err
	}
	file, d, err := a.change(name, []string{"log", "note"}, logview.Append{Line: line})
	if err != nil {
		return "", err
	}
	return a.status(file, d), nil
}

// keyRe is what a front matter key looks like.
var keyRe = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// setKeys sets and removes front matter keys in one write.
type setKeys struct {
	set    [][2]string
	remove []string
}

func (o setKeys) Apply(d doc.Document) (doc.Document, error) {
	for _, kv := range o.set {
		d = d.Set(kv[0], kv[1])
	}
	for _, k := range o.remove {
		d = d.Unset(k)
	}
	return d, nil
}

// arrange returns the change a key about the arrangement makes to a view,
// or false when the key is about something else. An empty value takes the
// key back out, so the note arranges itself again.
func arrange(key, value string) (func(*store.View), bool, error) {
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
		if value != "" && !validSize(value) {
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
		if value != "" && !slices.Contains(colors, strings.ToLower(value)) {
			return nil, true, fmt.Errorf("unknown color %q: use %s", value, strings.Join(colors, ", "))
		}
		return func(v *store.View) { v.Color = strings.ToLower(value) }, true, nil
	}
	return nil, false, nil
}

// colors are the note colors.
var colors = []string{"yellow", "pink", "blue", "green", "purple", "orange"}

// Set changes keys of an existing note and nothing else. Each pair is
// "key=value"; "key=" removes the key. Keys about where the note is on the
// screen (open, size, rows, color, pin) go to sticky.json, where the user's
// own arrangement is kept; the others go to the note's front matter. A book
// is named by its folder and takes only the first kind.
func (a *API) Set(name string, pairs []string) (string, error) {
	file, err := fileName(name)
	if err != nil {
		return "", err
	}
	if len(pairs) == 0 {
		return "", errors.New("nothing to set: give key=value pairs")
	}
	book := false
	if fi, err := os.Stat(filepath.Join(a.st.Dir, strings.TrimSpace(name))); err == nil && fi.IsDir() && !strings.Contains(name, "/") {
		file, book = strings.TrimSpace(name), true
	} else if _, err := a.st.Read(file); err != nil {
		return "", fmt.Errorf("there is no note %s", file)
	}
	var op setKeys
	var views []func(*store.View)
	var set, removed []string
	for _, p := range pairs {
		key, value, ok := strings.Cut(p, "=")
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if !ok || !keyRe.MatchString(key) || strings.ContainsAny(value, "\r\n") {
			return "", fmt.Errorf("%q is not key=value", p)
		}
		if value == "" {
			removed = append(removed, key)
		} else {
			set = append(set, key)
		}
		change, isView, err := arrange(key, value)
		switch {
		case err != nil:
			return "", err
		case isView:
			views = append(views, change)
		case book:
			return "", fmt.Errorf("%s is a folder: it has no %s, only open, size, rows, color and pin", file, key)
		case !strings.EqualFold(filepath.Ext(file), ".md"):
			return "", fmt.Errorf("only a Markdown note has front matter to keep %s in", key)
		case value == "":
			op.remove = append(op.remove, key)
		default:
			op.set = append(op.set, [2]string{key, value})
		}
	}
	if len(views) > 0 {
		err := a.st.SetView(file, func(v *store.View) {
			for _, change := range views {
				change(v)
			}
		})
		if err != nil {
			return "", err
		}
	}
	if len(op.set)+len(op.remove) > 0 {
		if err := a.st.Apply(file, op); err != nil {
			return "", err
		}
	}
	op.remove = removed
	var parts []string
	if len(set) > 0 {
		parts = append(parts, "set "+strings.Join(set, ", "))
	}
	if len(op.remove) > 0 {
		parts = append(parts, "removed "+strings.Join(op.remove, ", "))
	}
	return file + ": " + strings.Join(parts, "; "), nil
}
