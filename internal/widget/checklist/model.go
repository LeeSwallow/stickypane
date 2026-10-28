package checklist

import (
	"fmt"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/when"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

type entry struct {
	text    string
	item    bool
	checked bool
	done    time.Time // when a finished item was done, if the line says
}

// list is what a checklist note says: the reader's side, with no screen
// state in it.
type list struct {
	entries []entry
	items   []int // indexes of the checkbox entries
}

func parse(d doc.Document, e widget.Env) *Checklist { return &Checklist{list: read(d), env: e} }

// read reads a checklist note.
func read(d doc.Document) list {
	var c list
	for _, line := range doc.Lines(d.Body) {
		line = strings.TrimRight(line, "\r")
		if m := itemRe.FindStringSubmatch(line); m != nil {
			c.items = append(c.items, len(c.entries))
			text, done := splitStamp(strings.TrimSpace(m[4]))
			c.entries = append(c.entries, entry{text: text, item: true, checked: m[2] != " ", done: done})
			continue
		}
		c.entries = append(c.entries, entry{text: strings.TrimRight(line, " \t")})
	}
	for n := len(c.entries); n > 0 && !c.entries[n-1].item && c.entries[n-1].text == ""; n = len(c.entries) {
		c.entries = c.entries[:n-1]
	}
	return c
}

func (c list) counts() (done, total int) {
	for _, i := range c.items {
		if c.entries[i].checked {
			done++
		}
	}
	return done, len(c.items)
}

// Summary implements widget.Widget: done over total.
func (c list) Summary() string {
	done, total := c.counts()
	if total == 0 {
		return ""
	}
	return fmt.Sprintf("%d/%d", done, total)
}

// thing is an item as a screen other than the terminal reads it.
type thing struct {
	Text    string `json:"text"`
	Checked bool   `json:"checked"`
	Done    string `json:"done,omitempty"` // RFC 3339, when the line says
}

// data is the items and the lines between them, from the same read the
// board draws.
func data(d doc.Document, _ widget.Env) any {
	c := read(d)
	out := struct {
		Items []thing  `json:"items"`
		Prose []string `json:"prose"`
		Done  int      `json:"done"`
		Total int      `json:"total"`
	}{Items: []thing{}, Prose: []string{}}
	for _, e := range c.entries {
		switch {
		case e.item:
			t := thing{Text: e.text, Checked: e.checked}
			if !e.done.IsZero() {
				t.Done = e.done.Format(when.Moment)
			}
			out.Items = append(out.Items, t)
		case e.text != "":
			out.Prose = append(out.Prose, e.text)
		}
	}
	out.Done, out.Total = c.counts()
	return out
}
