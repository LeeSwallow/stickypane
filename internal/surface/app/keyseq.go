package app

import (
	"maps"
	"slices"
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/core/widget"
)

// keyWait is how long the board waits after keys that are a whole key
// and also the start of a longer one ("g" when "g g" is bound too, or a
// digit that is a tab's key and may start a count) before it runs them.
// Keys that only start a longer one wait for the rest as long as it takes.
const keyWait = 600 * time.Millisecond

// maxCount is the largest count: more is read as this.
const maxCount = 99

// keySeq is what the board has been given of a key so far: a count, and
// the keys of a sequence. Esc drops it; so does a key that goes on no
// sequence, which is dropped with it.
type keySeq struct {
	count int
	keys  []string
	node  *keyNode // where keys lead; nil while there are none
	mode  mode     // the screen the keys were pressed on
	gen   int      // counts the keys pressed, so a late wait is told apart
}

// pending reports whether there is a count or keys waiting for more.
func (s *keySeq) pending() bool { return s.count > 0 || len(s.keys) > 0 }

// reset forgets the count and the keys.
func (s *keySeq) reset() {
	s.count, s.keys, s.node = 0, nil, nil
	s.gen++
}

// keyWaitMsg is the end of a wait after keys that may go on. gen is the
// keySeq's gen when it began: a key pressed since makes it stale.
type keyWaitMsg struct{ gen int }

// keyRoot is the tree of the screen's keys: the zoomed note's or the
// board's.
func (m *Model) keyRoot() *keyNode {
	if m.mode == modeZoom {
		return m.keys.zoom
	}
	return m.keys.board
}

// feedKey gives key to the keys pressed so far. It is used when it is a
// digit of a count, esc dropping what is pending, or a key of a sequence;
// the action runs once its sequence is whole. A key that goes on no
// sequence drops what is pending and is used up with it, as in vi; on its
// own it is not used, and the caller can give it to something else.
func (m *Model) feedKey(key string) (bool, tea.Cmd) {
	s := &m.seq
	if s.pending() && s.mode != m.mode {
		s.reset()
	}
	if key == "esc" && s.pending() {
		s.reset()
		return true, nil
	}
	root := m.keyRoot()
	if len(s.keys) == 0 && isDigit(key) && (key != "0" || s.count > 0) {
		s.count = min(s.count*10+int(key[0]-'0'), maxCount)
		s.mode = m.mode
		s.gen++
		// A lone digit that is a key of its own (a tab's) runs when no
		// more comes.
		if n := root.next[key]; s.count < 10 && n != nil && n.act >= 0 {
			return true, m.waitKey()
		}
		return true, nil
	}
	at := root
	if s.node != nil {
		at = s.node
	}
	next := at.next[key]
	if next == nil {
		used := len(s.keys) > 0
		s.reset()
		return used, nil
	}
	if len(next.next) == 0 {
		count := s.count
		s.reset()
		return true, m.runAction(next.act, count)
	}
	s.keys, s.node, s.mode = append(s.keys, key), next, m.mode
	s.gen++
	if next.act >= 0 {
		return true, m.waitKey()
	}
	return true, nil
}

// waitKey waits keyWait for another key.
func (m *Model) waitKey() tea.Cmd {
	gen := m.seq.gen
	return tea.Tick(keyWait, func(time.Time) tea.Msg { return keyWaitMsg{gen} })
}

// keyWaited runs what the keys pressed so far are, when no key came after
// them: the action they are a whole key of, or a lone digit's own action
// (a tab), without a count.
func (m *Model) keyWaited(msg keyWaitMsg) tea.Cmd {
	s := &m.seq
	if msg.gen != s.gen || !s.pending() || s.mode != m.mode {
		return nil
	}
	node, count := s.node, s.count
	if node == nil {
		node, count = m.keyRoot().next[strconv.Itoa(s.count)], 0
	}
	s.reset()
	if node == nil || node.act < 0 {
		return nil
	}
	return m.runAction(node.act, count)
}

// runAction runs the action at index i; one that takes a count runs count
// times, laid out again between times so each one sees where the last left
// the notes.
func (m *Model) runAction(i, count int) tea.Cmd {
	a := m.keys.actions[i]
	if !a.repeat || count < 1 {
		count = 1
	}
	var cmds []tea.Cmd
	for n := range count {
		if n > 0 {
			m.relayout()
		}
		cmds = append(cmds, a.run(m))
	}
	return tea.Batch(cmds...)
}

// pendingFooter is the bottom line while keys wait for more: what was
// pressed ("3g…"), the keys that can come next with what they do, and esc
// to drop it. It is "" when nothing waits.
func (m *Model) pendingFooter() string {
	s := &m.seq
	if !s.pending() || s.mode != m.mode {
		return ""
	}
	typed := ""
	if s.count > 0 {
		typed = strconv.Itoa(s.count)
	}
	for _, k := range s.keys {
		typed += showKey(k)
	}
	typed = m.env.Styles.Accent.Bold(true).Render(typed + "…")
	var pairs []string
	if s.node != nil {
		for _, k := range slices.Sorted(maps.Keys(s.node.next)) {
			if n := s.node.next[k]; n.act >= 0 {
				pairs = append(pairs, k, m.keys.actions[n.act].describe(m.tr))
			} else {
				pairs = append(pairs, k+"…", "")
			}
		}
	}
	return typed + "  " + m.hintsThen(m.width-widget.Width(typed)-2, "esc", "cancel", pairs...)
}

func isDigit(key string) bool { return len(key) == 1 && key[0] >= '0' && key[0] <= '9' }
