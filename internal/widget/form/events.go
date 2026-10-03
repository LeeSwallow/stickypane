package form

import (
	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// events says that a button was pressed: the form was sent.
func events(before, after doc.Document) []widget.Event {
	was, _ := before.Get(keySubmitted)
	now, _ := after.Get(keySubmitted)
	if now != "" && now != was {
		return []widget.Event{{Type: "form.submitted", Item: now}}
	}
	return nil
}
