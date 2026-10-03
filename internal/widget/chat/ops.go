package chat

import (
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/when"
)

// Say adds a message from Who at At, under the heading of its day, which
// it adds when the chat has none for that day yet. The text is kept to
// one line.
type Say struct {
	Who, Text string
	At        time.Time
}

// Apply implements doc.Op.
func (o Say) Apply(d doc.Document) (doc.Document, error) {
	day := o.At.Format(when.Date)
	last := ""
	for _, l := range doc.Lines(d.Body) {
		if m := dayRe.FindStringSubmatch(strings.TrimRight(l, "\r")); m != nil {
			last = m[1]
		}
	}
	if last != day {
		if strings.TrimSpace(d.Body) != "" {
			d.Body = strings.TrimRight(d.Body, "\n") + "\n\n"
		}
		d.Body = doc.AppendLine(d.Body, "## "+day)
	}
	who := strings.TrimPrefix(strings.Join(strings.Fields(o.Who), ""), "@")
	d.Body = doc.AppendLine(d.Body, "@"+who+" "+o.At.Format(when.Clock)+" "+doc.OneLine(o.Text))
	return d, nil
}
