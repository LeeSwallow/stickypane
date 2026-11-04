package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// wait is the board's wait for another key running out.
func wait(m *Model) { m.Update(keyWaitMsg{m.seq.gen}) }

// oneColumn is five open notes a to e, one under the other.
func oneColumn(t *testing.T) (*Model, string) {
	t.Helper()
	m, dir := newModel(t, fiveNotes())
	writeView(t, dir, `{"notes":{"a.md":{"size":"1"},"b.md":{"size":"1"},"c.md":{"size":"1"},"d.md":{"size":"1"},"e.md":{"size":"1"}}}`)
	m.reload()
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 60})
	m.setFocus("a.md")
	m.relayout()
	return m, dir
}

// The tree follows a sequence key by key: a key that only starts one
// waits for the rest, and one that is also a whole key waits a moment.
func TestTheTreeFollowsASequence(t *testing.T) {
	t.Parallel()
	km := rebound(t, map[string][]string{"note-archive": {"d x"}, "note-pin": {"d"}, "note-color": {"g c d"}})
	d := km.board.next["d"]
	if d == nil || km.actions[d.act].name != "note-pin" || km.actions[d.next["x"].act].name != "note-archive" {
		t.Fatalf("d pins and d x archives: %+v", d)
	}
	if g := km.board.next["g"]; g == nil || g.act != -1 || km.actions[g.next["c"].next["d"].act].name != "note-color" {
		t.Errorf("g only starts g c d: %+v", g)
	}
	if _, err := newKeymap(append(actions(), action{name: "x", keys: []string{"d x"}, run: actions()[0].run})); err != nil {
		t.Errorf("a key that is no other's is fine: %v", err)
	}
	acts := actions()
	acts[0].keys, acts[1].keys = []string{"g g"}, []string{"g g"}
	if _, err := newKeymap(acts); err == nil {
		t.Error("a sequence bound to two actions is an error")
	}
	if _, ok := km.zoom.next["d"]; ok {
		t.Error("the zoomed note's tree has only the zoom actions")
	}
}

// d waits for x and says so on the bottom line; x archives. esc drops
// what waits, and so does a key that goes on nothing, which is dropped
// with it.
func TestASequenceWaitsShowsAndIsCancelled(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, map[string]string{"a.md": opened("one\n"), "b.md": opened("two\n")})
	m.keys = rebound(t, map[string][]string{"note-archive": {"d x"}, "note-delete": {"d d"}})
	m.setFocus("a.md")

	press(m, "d")
	foot := lastLine(screen(m))
	if !strings.HasPrefix(foot, "d…") || !strings.Contains(foot, "x move to archive") || !strings.Contains(foot, "esc cancel") {
		t.Fatalf("the bottom line shows what waits and what can come: %q", foot)
	}
	press(m, "esc")
	if m.seq.pending() || strings.Contains(lastLine(screen(m)), "…") {
		t.Fatalf("esc drops it: %q", lastLine(screen(m)))
	}
	press(m, "x")
	if len(m.items) != 2 {
		t.Fatal("x alone no longer archives")
	}
	press(m, "d", "tab")
	if m.focus != "a.md" || m.seq.pending() {
		t.Fatalf("a key that goes on no sequence is dropped with it: focus %q", m.focus)
	}
	press(m, "d", "x")
	if len(m.items) != 1 || !strings.Contains(readFile(t, dir, "archive/a.md"), "one") {
		t.Fatalf("d x archives: %d notes", len(m.items))
	}
	m.Update(keyWaitMsg{m.seq.gen - 1})
	if len(m.items) != 1 {
		t.Error("a stale wait does nothing")
	}
}

// When a key is whole and also starts a longer one, the wait running out
// runs it; the longer one runs when its keys come in time.
func TestAWholeKeyThatStartsAnotherRunsAfterTheWait(t *testing.T) {
	t.Parallel()
	m, _ := newModel(t, map[string]string{"a.md": opened("one\n")})
	m.keys = rebound(t, map[string][]string{"note-pin": {"g"}, "note-toggle": {"g g"}, "focus-first": {}})
	m.setFocus("a.md")
	press(m, "g")
	if m.pinned(m.items[0]) || !m.seq.pending() {
		t.Fatal("g waits")
	}
	wait(m)
	if !m.pinned(m.items[0]) || m.seq.pending() {
		t.Fatal("g runs once no key comes")
	}
	press(m, "g", "g")
	if m.isOpen(m.items[0]) {
		t.Error("g g closes the note")
	}
	if m.pinned(m.items[0]) == false {
		t.Error("g g is not g twice")
	}
}

// 3J moves a note three rows down, 10J as far as it goes; a count before
// an action that takes none is dropped.
func TestACountRepeatsAnAction(t *testing.T) {
	t.Parallel()
	m, dir := oneColumn(t)
	press(m, "3", "J")
	if got := order(m); got != "b c d a e" {
		t.Fatalf("3J: %q", got)
	}
	if got := readFile(t, dir, "sticky.json"); !strings.Contains(got, `"b.md"`) {
		t.Errorf("the order is written:\n%s", got)
	}
	press(m, "1", "0", "K")
	if got := order(m); got != "a b c d e" {
		t.Fatalf("10K: %q", got)
	}
	press(m, "3", "p")
	if !m.pinned(m.items[m.index("a.md")]) {
		t.Error("3p pins")
	}
	press(m, "3")
	if foot := lastLine(screen(m)); !strings.HasPrefix(foot, "3…") {
		t.Errorf("a count waits on the bottom line: %q", foot)
	}
}

// 0 alone is a key (the content height, as it comes: see
// TestZeroMakesALogAsTallAsItsContent); after a digit it is a digit.
func TestZeroIsAKeyAloneAndADigitAfterOne(t *testing.T) {
	t.Parallel()
	m, _ := oneColumn(t)
	m.keys = rebound(t, map[string][]string{"note-fit": {}, "note-pin": {"0"}})
	press(m, "0")
	if m.seq.pending() || !m.pinned(m.items[m.index("a.md")]) {
		t.Fatal("0 alone runs its action")
	}
	press(m, "2", "0")
	if m.seq.count != 20 {
		t.Errorf("2 0 is the count 20: %d", m.seq.count)
	}
}

// A digit with no key after it goes to its tab; a digit and then an
// action that takes a count is a count; esc drops it.
func TestADigitIsATabOnceTheWaitRunsOut(t *testing.T) {
	t.Parallel()
	m, _ := twoTabs(t)
	press(m, "2")
	if m.tab != 0 {
		t.Fatal("2 waits")
	}
	press(m, "esc")
	wait(m)
	if m.tab != 0 {
		t.Fatal("esc dropped the 2")
	}
	press(m, "2")
	wait(m)
	if m.tab != 1 {
		t.Fatalf("2 then nothing is tab 2: %d", m.tab)
	}
	press(m, "1", "g", "t")
	wait(m)
	if m.tab != 1 {
		t.Errorf("1gt is gt once, not tab 1: %d", m.tab)
	}
}

// A count does not take a key from the focused note: a kanban still
// moves its cursor with ↓ after a 3. A key of a sequence waiting does.
func TestASequenceWaitingComesBeforeTheNote(t *testing.T) {
	t.Parallel()
	m, _ := newModel(t, map[string]string{"b.md": boardFile})
	m.keys = rebound(t, map[string][]string{"note-pin": {"g n"}})
	m.setFocus("b.md")
	press(m, "3", "down")
	if m.seq.pending() {
		t.Error("the note took ↓ and the count went")
	}
	press(m, "g", "n")
	if !m.pinned(m.items[0]) || m.mode != modeBoard {
		t.Error("g n pins, though the kanban uses n")
	}
}

// A zoomed note shows and drops a waiting key too.
func TestTheZoomedNoteWaitsForKeys(t *testing.T) {
	t.Parallel()
	m, _ := newModel(t, map[string]string{"a.md": opened("one\n")})
	m.keys = rebound(t, map[string][]string{"help": {"g h"}})
	m.setFocus("a.md")
	press(m, "z")
	press(m, "g")
	if m.mode != modeZoom || !strings.HasPrefix(lastLine(screen(m)), "g…") {
		t.Fatalf("g waits on the zoomed note: %q", lastLine(screen(m)))
	}
	press(m, "esc")
	if m.mode != modeZoom || m.seq.pending() {
		t.Fatal("esc drops the g and stays")
	}
	press(m, "g", "h")
	if m.mode != modeHelp {
		t.Errorf("g h opens the help: %v", m.mode)
	}
}

// The help, the bottom line and stickypane keys show a sequence of
// characters side by side.
func TestASequenceIsShownSideBySide(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{"g g": "gg", "x": "x", "d X": "dX", "ctrl+w j": "<ctrl+w>j", "g enter": "g<enter>"} {
		if got := showKey(in); got != want {
			t.Errorf("showKey(%q) = %q, want %q", in, got, want)
		}
	}
	m, _ := newModel(t, map[string]string{"a.md": opened("one\n")})
	m.keys = rebound(t, map[string][]string{"note-archive": {"d x"}, "note-toggle": {"z c"}, "note-zoom": {"z z"}})
	m.Update(tea.WindowSizeMsg{Width: 160, Height: 24})
	help := strings.Join(m.keys.helpLines(m.tr), "\n")
	if !strings.Contains(help, "\n  dx               move to archive") {
		t.Errorf("the help shows dx:\n%s", help)
	}
	if foot := lastLine(screen(m)); !strings.Contains(foot, "zc close") {
		t.Errorf("the bottom line shows zc: %q", foot)
	}
}

// keys.json can give an action a sequence. One that starts another, or
// that a key of its own starts, is no clash: the wait tells them apart.
// The same sequence twice is.
func TestAKeyFileGivesSequences(t *testing.T) {
	t.Parallel()
	_, _, problems := bindKeys(map[string][]string{"note-pin": {"g"}, "note-toggle": {"g g"}, "note-color": {"g g c"}, "note-archive": {"x x"}, "focus-first": {}})
	if len(problems) > 0 {
		t.Errorf("prefixes are no clash: %v", problems)
	}
	km, _, problems := bindKeys(map[string][]string{"note-pin": {"ㅎ ㅎ"}, "note-color": {"g g"}, "focus-first": {}})
	if len(problems) != 1 || !strings.Contains(problems[0].Why, "g g is") || len(km.keysOf("note-pin")) != 1 {
		t.Errorf("ㅎ ㅎ is g g, which only one action can have: %v", problems)
	}
	m, dir := newModel(t, map[string]string{"a.md": opened("one\n")})
	if said := m.UseKeys("keys.json", map[string][]string{"note-archive": {"d x"}}, nil); said != "" {
		t.Fatal(said)
	}
	m.setFocus("a.md")
	press(m, "d", "x")
	if !strings.Contains(readFile(t, dir, "archive/a.md"), "one") {
		t.Error("d x archives")
	}
}
