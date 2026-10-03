package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/lineedit"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

func init() {
	handlers[modeInput] = inputUpdate
	footers[modeInput] = func(m *Model) string {
		return widget.Bold.Render(m.inputLabel) + ": " + m.input.View()
	}
}

// ask collects one line of text on the bottom line. Enter calls submit with
// the trimmed text unless it is empty; esc cancels. Either way the screen it
// was opened from comes back first. A caller that takes an empty line as an
// answer sets inputEmpty after asking.
func (m *Model) ask(label, initial string, submit func(text string)) {
	// The label comes from a note, so it is cleaned, and it never takes
	// more than a third of the line: the text being typed must show.
	label = widget.Truncate(widget.Clean(label), max(m.width/3, 8))
	in := lineedit.New(initial, max(m.width-widget.Width(label)-3, 1))
	m.input, m.inputLabel, m.onSubmit, m.inputEmpty = in, label, submit, false
	m.back, m.mode = m.mode, modeInput
}

func inputUpdate(m *Model, msg tea.Msg) tea.Cmd {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "esc":
			m.mode = m.back
			return nil
		case "enter":
			text := strings.TrimSpace(m.input.Value())
			m.mode = m.back
			if text != "" || m.inputEmpty {
				m.onSubmit(text)
			}
			return nil
		}
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if !m.input.Key(msg.String()) && msg.Text != "" {
			m.input.Type(msg.Text)
		}
	case tea.PasteMsg:
		m.input.Type(msg.Content)
	}
	return nil
}
