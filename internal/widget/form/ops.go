package form

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/when"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// unsubmit takes the submission back: the answers are changing again.
func unsubmit(d doc.Document) doc.Document {
	return d.Unset(keySubmitted).Unset(keySubmittedAt)
}

// Choose sets an option to state On. The option is the one with Text; when
// several share that text, Nth counts from zero to say which. Choosing an
// option written with ( ) clears the others of its question. An option that
// is gone, or already in that state, is a conflict.
type Choose struct {
	Text string
	Nth  int
	On   bool
}

// Apply implements doc.Op.
func (o Choose) Apply(d doc.Document) (doc.Document, error) {
	raw := doc.Lines(d.Body)
	lines, _ := scan(d.Body)
	set := func(i int, on bool) {
		m := optionRe.FindStringSubmatch(strings.TrimSuffix(raw[i], "\r"))
		mark := " "
		if on {
			mark = "x"
		}
		doc.SetLine(raw, i, m[1]+m[2]+mark+m[4]+m[5]+m[6])
	}
	nth := o.Nth
	for i, l := range lines {
		if l.kind != option || l.text != o.Text {
			continue
		}
		if nth > 0 {
			nth--
			continue
		}
		if l.on == o.On {
			break // not the option the screen showed
		}
		set(i, o.On)
		if o.On && l.radio {
			for j, other := range lines {
				if j != i && other.kind == option && other.question == l.question && other.on {
					set(j, false)
				}
			}
		}
		d.Body = doc.Join(raw)
		return unsubmit(d), nil
	}
	return d, doc.ErrConflict
}

// Fill writes Text into the Nth field, counting from zero. A field that no
// longer reads Old is a conflict.
type Fill struct {
	Nth       int
	Old, Text string
}

// Apply implements doc.Op.
func (o Fill) Apply(d doc.Document) (doc.Document, error) {
	raw := doc.Lines(d.Body)
	lines, _ := scan(d.Body)
	nth := o.Nth
	for i, l := range lines {
		if l.kind != field {
			continue
		}
		if nth > 0 {
			nth--
			continue
		}
		if l.text != o.Old {
			break
		}
		if o.Text == o.Old {
			return d, nil // nothing changed, so nothing is taken back
		}
		doc.SetLine(raw, i, "> "+o.Text)
		d.Body = doc.Join(raw)
		return unsubmit(d), nil
	}
	return d, doc.ErrConflict
}

// Submit records that the button Label was pressed at time At. A button the
// form no longer has is a conflict.
type Submit struct{ Label, At string }

// Apply implements doc.Op.
func (o Submit) Apply(d doc.Document) (doc.Document, error) {
	lines, _ := scan(d.Body)
	for _, label := range buttonLabels(lines) {
		if label == o.Label {
			return d.Set(keySubmitted, o.Label).Set(keySubmittedAt, o.At), nil
		}
	}
	return d, doc.ErrConflict
}

// pick reads the args' line as a position in the lines Data lists and
// returns the form read and that position.
func pick(d doc.Document, verb string, a widget.Action) (content, int, error) {
	i, err := strconv.Atoi(a.Args["line"])
	if err != nil {
		return content{}, 0, fmt.Errorf("%s: line must be a number, not %q", verb, a.Args["line"])
	}
	c := read(d)
	if i < 0 || i >= len(c.lines) {
		return c, 0, fmt.Errorf("%s: no line %d, the form has %d", verb, i, len(c.lines))
	}
	return c, i, nil
}

// act is the form's Kind.Act: "choose" with line, a position in the lines
// of Data, which flips that option as enter does; "fill" with line and
// text; "submit" with label, a button of the form. The time of a submit is
// the board's, as when a button is pressed on the terminal.
func act(d doc.Document, a widget.Action) (doc.Op, error) {
	switch a.Name {
	case "choose":
		c, i, err := pick(d, a.Name, a)
		if err != nil {
			return nil, err
		}
		l := c.lines[i]
		if l.kind != option {
			return nil, fmt.Errorf("choose: line %d is not an option", i)
		}
		nth := 0
		for _, e := range c.lines[:i] {
			if e.kind == option && e.text == l.text {
				nth++
			}
		}
		return Choose{Text: l.text, Nth: nth, On: !l.on}, nil
	case "fill":
		c, i, err := pick(d, a.Name, a)
		if err != nil {
			return nil, err
		}
		l := c.lines[i]
		if l.kind != field {
			return nil, fmt.Errorf("fill: line %d is not a field", i)
		}
		text, ok := a.Args["text"]
		if !ok || strings.ContainsAny(text, "\r\n") {
			return nil, fmt.Errorf("fill: text must be one line")
		}
		nth := 0
		for _, e := range c.lines[:i] {
			if e.kind == field {
				nth++
			}
		}
		return Fill{Nth: nth, Old: l.text, Text: strings.TrimSpace(text)}, nil
	case "submit":
		lines, _ := scan(d.Body)
		for _, label := range buttonLabels(lines) {
			if label == a.Args["label"] {
				return Submit{Label: label, At: now().Format(when.Moment)}, nil
			}
		}
		return nil, fmt.Errorf("submit: the form has no button %q", a.Args["label"])
	}
	return nil, fmt.Errorf("%w: %q", widget.ErrUnknownAction, a.Name)
}
