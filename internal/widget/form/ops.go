package form

import (
	"strings"

	"github.com/LeeSwallow/stickypane/internal/doc"
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
