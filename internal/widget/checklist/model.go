package checklist

import (
	"fmt"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

type entry struct {
	text    string
	item    bool
	checked bool
}

// list is what a checklist note says: the reader's side, with no screen
// state in it.
type list struct {
	entries []entry
	items   []int // indexes of the checkbox entries
}

func parse(d doc.Document) *Checklist { return &Checklist{list: read(d)} }

// read reads a checklist note.
func read(d doc.Document) list {
	var c list
	for _, line := range doc.Lines(d.Body) {
		line = strings.TrimRight(line, "\r")
		if m := itemRe.FindStringSubmatch(line); m != nil {
			c.items = append(c.items, len(c.entries))
			c.entries = append(c.entries, entry{text: strings.TrimSpace(m[4]), item: true, checked: m[2] != " "})
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
