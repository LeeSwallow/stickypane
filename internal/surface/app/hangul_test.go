package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// The whole 2-set layout, key by key, as it is printed on a Korean
// keyboard: the row of letters and the jamo above them, then shift.
func TestTheTwoSetLayout(t *testing.T) {
	t.Parallel()
	latin := []rune("qwertyuiopasdfghjklzxcvbnmQWERTOP")
	jamo := []rune("ㅂㅈㄷㄱㅅㅛㅕㅑㅐㅔㅁㄴㅇㄹㅎㅗㅓㅏㅣㅋㅌㅊㅍㅠㅜㅡㅃㅉㄸㄲㅆㅒㅖ")
	if len(twoSet) != len(jamo) {
		t.Errorf("the table has %d jamo, the layout %d", len(twoSet), len(jamo))
	}
	for i, j := range jamo {
		if got := latinKey(string(j)); got != string(latin[i]) {
			t.Errorf("latinKey(%q) = %q, want %q", string(j), got, string(latin[i]))
		}
	}
	for in, want := range map[string]string{
		"shift+ㅓ": "J", "ctrl+ㅓ": "ctrl+j", "ctrl+shift+ㅓ": "ctrl+shift+j", "alt+ㅃ": "alt+Q",
		"j": "j", "가": "가", "enter": "enter", "ctrl++": "ctrl++", "+": "+",
	} {
		if got := latinKey(in); got != want {
			t.Errorf("latinKey(%q) = %q, want %q", in, got, want)
		}
	}
}

// hangul is the message for a jamo typed with the Korean layout on.
func hangul(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

// On the board a jamo is its Latin key: ㅌ archives as x does, ㅇㅇ is
// d d, and ㅣ is l to a kanban.
func TestTheBoardReadsTheKoreanLayout(t *testing.T) {
	t.Parallel()
	m, _ := newModel(t, map[string]string{"a.md": opened("one\n"), "b.md": opened("two\n"), "k.md": boardFile})
	m.keys = rebound(t, map[string][]string{"note-delete": {"d d"}})
	m.setFocus("a.md")
	m.Update(hangul('ㅌ'))
	if m.index("a.md") >= 0 {
		t.Fatal("ㅌ archives like x")
	}
	m.setFocus("b.md")
	m.Update(hangul('ㅇ'))
	if !strings.HasPrefix(lastLine(screen(m)), "d…") {
		t.Fatalf("ㅇ waits as d: %q", lastLine(screen(m)))
	}
	m.Update(hangul('ㅇ'))
	if m.mode != modeConfirm {
		t.Fatal("ㅇㅇ asks to delete like d d")
	}
	press(m, "esc")
	m.setFocus("k.md")
	drawn := func() string { s, _ := m.items[m.index("k.md")].w.Draw(60, true); return s }
	before := drawn()
	m.Update(hangul('ㅣ'))
	if drawn() == before {
		t.Error("ㅣ moves the kanban's cursor like l")
	}
}

// A prompt and the editor take the jamo as it is typed.
func TestTypingKeepsTheKoreanLayout(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, map[string]string{"a.md": opened("one\n")})
	m.setFocus("a.md")
	press(m, "R")
	if m.mode != modeInput {
		t.Fatalf("R asks for a title: %v", m.mode)
	}
	m.Update(hangul('ㅓ'))
	if v := m.input.Value(); !strings.HasSuffix(v, "ㅓ") {
		t.Errorf("the prompt has ㅓ: %q", v)
	}
	press(m, "esc")
	press(m, "e")
	typeKeys(m, "GAㅓㅏ<esc>:wq<enter>")
	if got := readFile(t, dir, "a.md"); got != opened("oneㅓㅏ\n") {
		t.Errorf("the editor typed the jamo: %q", got)
	}
}
