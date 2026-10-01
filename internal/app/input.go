package app

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

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
	in := textinput.New()
	in.Prompt = ""
	in.SetValue(initial)
	in.CursorEnd()
	in.SetWidth(max(m.width-widget.Width(label)-3, 1))
	in.Focus()
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
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return cmd
}
