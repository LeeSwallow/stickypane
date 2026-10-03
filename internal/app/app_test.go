package app

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/kinds"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/theme"
	"github.com/LeeSwallow/stickypane/internal/widget"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

const boardFile = "---\ntype: board\ntitle: Auth\nopen: true\n---\n## To do\n- payments\n## Doing\n- login API\n## Done\n- schema\n"

const checklistFile = "---\ntype: checklist\ntitle: Release\nopen: true\n---\n- [x] build\n- [ ] ship\n"

// opened returns a plain note that is open on the screen.
func opened(body string) string { return "---\nopen: true\n---\n" + body }

// numbered returns n lines "line 01" .. "line NN".
func numbered(n int) string {
	var sb strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&sb, "line %02d\n", i)
	}
	return sb.String()
}

// newModel builds the screen over a temporary notes folder, sized 80x24.
func newModel(t *testing.T, files map[string]string) (*Model, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), store.DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		writeFile(t, dir, name, content)
	}
	m := New(store.Open(dir), kinds.Default(note.Plain), nil, theme.NewHolder(theme.Pick("auto", true)))
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return m, dir
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// key builds the message Bubble Tea sends for a key name such as "enter",
// "shift+tab", "n" or "D".
func key(k string) tea.KeyPressMsg {
	switch k {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "shift+tab":
		return tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	case "ctrl+c":
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	}
	r := []rune(k)[0]
	return tea.KeyPressMsg{Code: unicode.ToLower(r), Text: k}
}

func press(m *Model, keys ...string) {
	for _, k := range keys {
		m.Update(key(k))
	}
}

func typeText(m *Model, s string) {
	for _, r := range s {
		m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}

func screen(m *Model) string { return ansi.Strip(m.render()) }

func TestKeyHelperMatchesBubbleTeaNames(t *testing.T) {
	for _, k := range []string{"enter", "esc", "tab", "shift+tab", "space", "ctrl+c", "n", "D", "?", "+", "-"} {
		if got := key(k).String(); got != k {
			t.Errorf("key(%q).String() = %q", k, got)
		}
	}
}

func TestTitleBarListsEveryNoteAndOnlyOpenOnesAreDrawn(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": "closed note body\n", "b.md": boardFile})
	s := screen(m)
	bar := strings.Split(s, "\n")[0]
	if !strings.Contains(bar, "○ ✎ a") || !strings.Contains(bar, "● ▦ Auth") {
		t.Errorf("the title bar should list both notes with their state: %q", bar)
	}
	if !strings.Contains(s, "payments") || !strings.Contains(s, "To do (1)") {
		t.Errorf("the open board should be drawn:\n%s", s)
	}
	if strings.Contains(s, "closed note body") {
		t.Errorf("a closed note must not be drawn:\n%s", s)
	}
	if n := strings.Count(s, "\n") + 1; n != 24 {
		t.Errorf("screen is %d lines tall, want 24", n)
	}
}

func TestTheScreenHasVisualStructure(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": "one\n", "b.md": boardFile})
	press(m, "tab")
	lines := strings.Split(screen(m), "\n")
	if lines[1] != strings.Repeat("─", 80) {
		t.Errorf("a rule should separate the title bar from the notes: %q", lines[1])
	}
	if !strings.HasPrefix(lines[2], "╔ ▦ Auth ") || !strings.HasSuffix(lines[2], " 3 cards ╗") {
		t.Errorf("the frame should carry the shape's icon and the note's summary: %q", lines[2])
	}
	foot := lines[len(lines)-1]
	if !strings.Contains(foot, "H L shift card") || !strings.Contains(foot, "enter zoom") {
		t.Errorf("the bottom line should guide the focused note's keys: %q", foot)
	}
	press(m, "tab")
	if foot := strings.Split(screen(m), "\n")[23]; !strings.Contains(foot, "enter open") || strings.Contains(foot, "shift card") {
		t.Errorf("a closed note has no keys of its own: %q", foot)
	}
}

func TestEmptyStatesExplainWhatToDo(t *testing.T) {
	m, _ := newModel(t, nil)
	if s := screen(m); !strings.Contains(s, "No notes yet") {
		t.Errorf("screen = %q", s)
	}
	m, _ = newModel(t, map[string]string{"a.md": "one\n"})
	if s := screen(m); !strings.Contains(s, "Nothing is open") {
		t.Errorf("screen = %q", s)
	}
}

func TestOpenNotesAreNotCut(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("---\ntype: checklist\ntitle: Many\nopen: true\n---\n")
	for i := 1; i <= 12; i++ {
		fmt.Fprintf(&sb, "- [ ] item %02d\n", i)
	}
	m, _ := newModel(t, map[string]string{"c.md": sb.String()})
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	s := screen(m)
	if !strings.Contains(s, "item 01") || !strings.Contains(s, "item 12") || strings.Contains(s, "more") {
		t.Errorf("every item of an open note should be visible:\n%s", s)
	}
}

func TestFocusCyclesThroughEveryNote(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": "one\n", "b.md": boardFile})
	if m.focus != "a.md" {
		t.Fatalf("focus = %q, want a.md", m.focus)
	}
	if strings.Contains(screen(m), "╔") {
		t.Error("the focused note is closed, so no frame should be focused")
	}
	press(m, "tab")
	if m.focus != "b.md" || strings.Count(screen(m), "╔") != 1 {
		t.Errorf("after tab: focus = %q, focused frames = %d", m.focus, strings.Count(screen(m), "╔"))
	}
	press(m, "tab")
	if m.focus != "a.md" {
		t.Errorf("tab should wrap around, focus = %q", m.focus)
	}
	press(m, "shift+tab")
	if m.focus != "b.md" {
		t.Errorf("after shift+tab: focus = %q", m.focus)
	}
}

// isOpenIn reports what sticky.json says about a note being open: "true",
// "false", or "" when it says nothing.
func isOpenIn(t *testing.T, dir, name string) string {
	t.Helper()
	switch v := viewsOf(t, dir)[name].Open; {
	case v == nil:
		return ""
	case *v:
		return "true"
	}
	return "false"
}

func TestOTogglesOpen(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": "one\n"})
	press(m, "o")
	if got := isOpenIn(t, dir, "a.md"); got != "true" || readFile(t, dir, "a.md") != "one\n" {
		t.Fatalf("open = %q, file = %q", got, readFile(t, dir, "a.md"))
	}
	if s := screen(m); !strings.Contains(s, "one") || !strings.Contains(s, "● ✎ a") {
		t.Errorf("the note should be open now:\n%s", s)
	}
	press(m, "o")
	if got := isOpenIn(t, dir, "a.md"); got != "false" {
		t.Errorf("open = %q", got)
	}
	if s := screen(m); !strings.Contains(s, "○ ✎ a") || !strings.Contains(s, "Nothing is open") {
		t.Errorf("the note should be closed again:\n%s", s)
	}
}

func TestEnterOpensAClosedNoteThenZooms(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": "one\n"})
	press(m, "enter")
	if got := isOpenIn(t, dir, "a.md"); got != "true" || m.mode != modeBoard {
		t.Fatalf("enter on a closed note should open it: open = %q, mode = %v", got, m.mode)
	}
	press(m, "enter")
	if m.mode != modeZoom {
		t.Fatalf("enter on an open note should zoom, mode = %v", m.mode)
	}
	press(m, "esc")
	if m.mode != modeBoard {
		t.Errorf("esc should leave the zoom, mode = %v", m.mode)
	}
}

func TestNotesFromTheAgentOpenWithoutTouchingTheFile(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": "one\n"})
	writeFile(t, dir, "explain.md", "written by the agent\n")
	m.Update(changedMsg{})
	if s := screen(m); !strings.Contains(s, "written by the agent") {
		t.Errorf("a note that appears while running should be shown:\n%s", s)
	}
	if got := readFile(t, dir, "explain.md"); got != "written by the agent\n" {
		t.Errorf("showing it must not edit the file, got %q", got)
	}
	if strings.Contains(screen(m), "\n│ one") {
		t.Error("notes that were there at startup stay closed")
	}
	writeFile(t, dir, "quiet.md", "---\nopen: false\n---\nhidden on purpose\n")
	m.Update(changedMsg{})
	if strings.Contains(screen(m), "hidden on purpose") {
		t.Error("a new note that says open: false stays closed")
	}
}

func TestBackgroundReportDoesNotReopenNotes(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": "closed body\n", "b.md": opened("open body\n")})
	press(m, "tab")
	before := m.items[0].w
	m.Update(tea.BackgroundColorMsg{Color: color.White})
	if th := m.theme.Get(); th.Dark || th.Name != theme.DefaultLight {
		t.Errorf("a light terminal should get the light theme, got %q", th.Name)
	}
	if m.items[0].w == before {
		t.Error("widgets should be rebuilt so they redraw with the new colors")
	}
	s := screen(m)
	if m.focus != "b.md" || strings.Contains(s, "closed body") || !strings.Contains(s, "open body") {
		t.Errorf("focus and open state must survive the redraw, focus = %q:\n%s", m.focus, s)
	}
}

func TestARewriteThatDropsOpenDoesNotCloseTheNote(t *testing.T) {
	m, dir := newModel(t, map[string]string{"c.md": checklistFile})
	writeFile(t, dir, "c.md", "---\ntype: checklist\ntitle: Release\n---\n- [x] build\n- [x] ship\n") // the agent rewrote it
	m.Update(changedMsg{})
	if s := screen(m); !strings.Contains(s, "2/2") {
		t.Errorf("a note the user had open should stay on screen when a rewrite drops the key:\n%s", s)
	}
	writeFile(t, dir, "c.md", "---\ntype: checklist\nopen: false\n---\n- [x] build\n")
	m.Update(changedMsg{})
	if s := screen(m); strings.Contains(s, "1/1") {
		t.Errorf("an explicit open: false still closes it:\n%s", s)
	}
}

func TestATitleBarWithManyNotesLeavesRoomAndFollowsTheFocus(t *testing.T) {
	files := map[string]string{}
	for i := 0; i < 30; i++ {
		files[fmt.Sprintf("%02d-note-number.md", i)] = "body\n"
	}
	files["00-note-number.md"] = opened("first body\n")
	m, _ := newModel(t, files)
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 16})
	if s := screen(m); !strings.Contains(s, "first body") {
		t.Fatalf("the title bar must leave room for the open note:\n%s", s)
	}
	for i := 0; i < 30; i++ {
		lines := strings.Split(screen(m), "\n")
		bar := strings.Join(lines[:6], "\n")
		if want := fmt.Sprintf("%02d-note-number", i); !strings.Contains(bar, want) {
			t.Fatalf("the focused tab %s should be visible in the title bar:\n%s", want, bar)
		}
		press(m, "tab")
	}
}

func TestTheSelectedCardsDetailsStayInView(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("---\ntype: board\ntitle: Tall\nopen: true\n---\n## A\n")
	for i := 1; i <= 12; i++ {
		fmt.Fprintf(&sb, "- card %02d\n  - detail a%02d\n  - detail b%02d\n  - detail c%02d\n", i, i, i, i)
	}
	m, _ := newModel(t, map[string]string{"b.md": sb.String()})
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 14})
	for i := 2; i <= 12; i++ {
		press(m, "j")
		s := screen(m)
		if want := fmt.Sprintf("detail c%02d", i); !strings.Contains(s, fmt.Sprintf("› card %02d", i)) || !strings.Contains(s, want) {
			t.Fatalf("card %02d and all its details should be in view:\n%s", i, s)
		}
	}
	press(m, "enter") // the same holds when zoomed
	for i := 11; i >= 1; i-- {
		press(m, "k")
		if s := screen(m); !strings.Contains(s, fmt.Sprintf("detail c%02d", i)) {
			t.Fatalf("zoomed: the details of card %02d should be in view:\n%s", i, s)
		}
	}
}

func TestAnEmptyColumnStaysInViewWhenSelected(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("---\ntype: board\ntitle: Narrow\nopen: true\n---\n## A\n")
	for i := 1; i <= 20; i++ {
		fmt.Fprintf(&sb, "- card %02d\n", i)
	}
	sb.WriteString("## Empty\n## C\n- x\n")
	m, _ := newModel(t, map[string]string{"b.md": sb.String()})
	m.Update(tea.WindowSizeMsg{Width: 36, Height: 14})
	for i := 0; i < 19; i++ {
		press(m, "j")
	}
	press(m, "l")
	if s := screen(m); !strings.Contains(s, "Empty (0)") {
		t.Errorf("the selected empty column should be on screen, where a new card would go:\n%s", s)
	}
}

func TestSizeKeysDoNothingWhereTheyCannotShow(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": "closed\n", "b.md": boardFile})
	press(m, "+", "-")
	if got := readFile(t, dir, "a.md"); got != "closed\n" {
		t.Errorf("resizing a closed note shows nothing, so it should write nothing: %q", got)
	}
	press(m, "tab", "+")
	if got := readFile(t, dir, "b.md"); got != boardFile {
		t.Errorf("a board is already a page: + should write nothing, got %q", got)
	}
}

func TestTheBottomLineAlwaysOffersHelp(t *testing.T) {
	m, _ := newModel(t, map[string]string{"b.md": boardFile})
	for _, width := range []int{80, 40, 24} {
		m.Update(tea.WindowSizeMsg{Width: width, Height: 20})
		lines := strings.Split(screen(m), "\n")
		if foot := lines[len(lines)-1]; !strings.Contains(foot, "? help") {
			t.Errorf("width %d: the way to the key list must not be dropped: %q", width, foot)
		}
	}
}

func TestSizeKeysWriteTheSize(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": opened("one\n")})
	steps := []struct{ key, size string }{{"+", "half"}, {"+", "page"}, {"+", "page"}, {"-", "half"}, {"-", "card"}, {"-", "card"}}
	for _, step := range steps {
		press(m, step.key)
		if got := viewsOf(t, dir)["a.md"].Size; got != step.size || readFile(t, dir, "a.md") != opened("one\n") {
			t.Fatalf("after %s: size = %q, want %s (file %q)", step.key, got, step.size, readFile(t, dir, "a.md"))
		}
	}
}

func TestSizesDecideTheLayout(t *testing.T) {
	second := strings.Replace(checklistFile, "Release", "Second", 1)
	m, _ := newModel(t, map[string]string{"1.md": checklistFile, "2.md": second, "3.md": boardFile})
	// frameTop reports whether line is the top border of a frame naming every title.
	frameTop := func(line string, titles ...string) bool {
		if !strings.ContainsAny(line, "╭╔") {
			return false
		}
		for _, title := range titles {
			if !strings.Contains(line, " "+title+" ") {
				return false
			}
		}
		return true
	}
	var sameRow, boardAlone bool
	for _, line := range strings.Split(screen(m), "\n") {
		if frameTop(line, "Release", "Second") {
			sameRow = true
		}
		if frameTop(line, "Auth") && !frameTop(line, "Release") && widget.Width(strings.TrimRight(line, " ")) == 80 {
			boardAlone = true
		}
	}
	if !sameRow {
		t.Errorf("two half-size notes should sit side by side:\n%s", screen(m))
	}
	if !boardAlone {
		t.Errorf("a page-size note should take the whole width:\n%s", screen(m))
	}
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 30})
	for _, line := range strings.Split(screen(m), "\n") {
		if frameTop(line, "Release", "Second") {
			t.Errorf("in a narrow pane half-size notes take the full width:\n%s", screen(m))
		}
	}
}

func TestKeysReachTheFocusedOpenNote(t *testing.T) {
	m, dir := newModel(t, map[string]string{"b.md": boardFile})
	press(m, "L")
	want := "---\ntype: board\ntitle: Auth\nopen: true\n---\n## To do\n## Doing\n- login API\n- payments\n## Done\n- schema\n"
	if got := readFile(t, dir, "b.md"); got != want {
		t.Errorf("file = %q\nwant  %q", got, want)
	}
	if s := screen(m); !strings.Contains(s, "› payments") || m.mode != modeBoard {
		t.Errorf("the card should move without zooming in:\n%s", s)
	}
}

func TestSpaceTogglesAChecklistItemInPlace(t *testing.T) {
	m, dir := newModel(t, map[string]string{"c.md": checklistFile})
	press(m, "j", "space")
	if got := readFile(t, dir, "c.md"); !strings.HasSuffix(got, "- [x] build\n- [x] ship\n") {
		t.Errorf("file = %q", got)
	}
	if s := screen(m); !strings.Contains(s, "2/2") {
		t.Errorf("the progress should update:\n%s", s)
	}
}

func TestClosedNotesDoNotTakeWidgetKeys(t *testing.T) {
	closed := strings.Replace(boardFile, "open: true\n", "", 1)
	m, dir := newModel(t, map[string]string{"b.md": closed})
	press(m, "L", "J", "space")
	if got := readFile(t, dir, "b.md"); got != closed {
		t.Errorf("a closed note must not be edited by widget keys, file = %q", got)
	}
}

func TestPinnedNotesComeFirst(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": "one\n", "z.md": "---\npin: true\n---\npinned\n"})
	bar := strings.Split(screen(m), "\n")[0]
	if m.items[0].note.Name != "z.md" || strings.Index(bar, "z") > strings.Index(bar, "○ ✎ a") || !strings.Contains(bar, "📌") {
		t.Errorf("the pinned note should come first and show its pin: %q", bar)
	}
}

func TestChangedNotesAreMarkedUntilFocused(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": "one\n", "b.md": "two\n"})
	if strings.Contains(screen(m), "*") {
		t.Fatal("nothing changed yet")
	}
	writeFile(t, dir, "b.md", "two, edited by the agent\n")
	future := time.Now().Add(time.Minute)
	if err := os.Chtimes(filepath.Join(dir, "b.md"), future, future); err != nil {
		t.Fatal(err)
	}
	m.Update(changedMsg{})
	if bar := strings.Split(screen(m), "\n")[0]; strings.Count(bar, "*") != 1 {
		t.Fatalf("the edited note should be marked in the title bar: %q", bar)
	}
	press(m, "tab")
	if strings.Contains(screen(m), "*") {
		t.Errorf("the mark should clear once the note was focused:\n%s", screen(m))
	}
}

func TestFocusSurvivesWhenFocusedNoteDisappears(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": "one\n", "b.md": "two\n", "c.md": "three\n"})
	press(m, "tab")
	os.Remove(filepath.Join(dir, "b.md"))
	m.Update(changedMsg{})
	if m.focus != "c.md" {
		t.Errorf("focus = %q, want the note that took its place", m.focus)
	}
}

func TestUnreadableNoteShowsWhy(t *testing.T) {
	m, _ := newModel(t, map[string]string{"big.md": strings.Repeat("x", store.MaxSize+1)})
	s := screen(m)
	if !strings.Contains(s, "Cannot show this note") || !strings.Contains(s, "big") {
		t.Errorf("screen should explain the problem and name the file:\n%s", s)
	}
}

func TestScrollFollowsTheCursorInATallNote(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("---\ntype: checklist\ntitle: Long\nopen: true\n---\n")
	for i := 1; i <= 60; i++ {
		fmt.Fprintf(&sb, "- [ ] item %02d\n", i)
	}
	m, _ := newModel(t, map[string]string{"c.md": sb.String()})
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 12})
	for i := 0; i < 50; i++ {
		press(m, "j")
		if !strings.Contains(screen(m), "› ☐ item") {
			t.Fatalf("the cursor scrolled out of view after %d steps:\n%s", i+1, screen(m))
		}
	}
	if !strings.Contains(screen(m), "› ☐ item 51") {
		t.Errorf("the cursor should be on item 51:\n%s", screen(m))
	}
}

func TestScrollKeysMoveTheScreenForNotesWithoutACursor(t *testing.T) {
	m, _ := newModel(t, map[string]string{"long.md": opened(numbered(40))})
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 12})
	if s := screen(m); !strings.Contains(s, "line 01") || strings.Contains(s, "line 40") {
		t.Fatalf("the top of the note should show first:\n%s", s)
	}
	press(m, "G")
	if s := screen(m); !strings.Contains(s, "line 40") || strings.Contains(s, "line 01") {
		t.Errorf("G should scroll to the end:\n%s", s)
	}
	press(m, "g")
	if !strings.Contains(screen(m), "line 01") {
		t.Errorf("g should scroll back to the top:\n%s", screen(m))
	}
}

// longNote is a note that takes all the room it is given.
var longNote = "---\nopen: true\nsize: page\n---\n" + strings.Repeat("filler\n", 80)

func TestALogAsksForTenLines(t *testing.T) {
	m, _ := newModel(t, map[string]string{
		"log.md": "---\ntype: log\ntitle: Work log\nopen: true\nsize: page\n---\n" + numbered(30),
		"z.md":   longNote,
	})
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	s := screen(m)
	if !strings.Contains(s, "line 30") || !strings.Contains(s, "line 21") || strings.Contains(s, "line 20") {
		t.Errorf("next to a note that wants the room, a log shows its last ten lines:\n%s", s)
	}
}

func TestRowsSetTheHeightANoteAsksFor(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("---\ntype: checklist\ntitle: Fixed\nopen: true\nsize: page\nrows: 6\n---\n")
	for i := 1; i <= 20; i++ {
		fmt.Fprintf(&sb, "- [ ] item %02d\n", i)
	}
	m, _ := newModel(t, map[string]string{"c.md": sb.String(), "z.md": longNote})
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	if n := strings.Count(screen(m), "☐"); n > 6 {
		t.Errorf("rows: 6 should limit the height, %d items visible:\n%s", n, screen(m))
	}
	for i := 0; i < 15; i++ {
		press(m, "j")
	}
	if !strings.Contains(screen(m), "› ☐ item 16") {
		t.Errorf("the cursor should stay visible inside a fixed-height note:\n%s", screen(m))
	}
}

func TestTinyTerminalDoesNotPanic(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": opened("한글 메모입니다\n"), "b.md": boardFile, "c.md": checklistFile})
	for _, size := range [][2]int{{0, 0}, {1, 1}, {5, 1}, {5, 3}, {10, 3}, {35, 2}, {200, 2}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		press(m, "tab", "j", "l", "enter", "j", "esc", "?", "j", "esc")
		out := m.render()
		if out == "" {
			continue
		}
		lines := strings.Split(out, "\n")
		if len(lines) > size[1] {
			t.Errorf("%v: %d lines, want at most %d", size, len(lines), size[1])
		}
		for _, line := range lines {
			if w := widget.Width(line); w > size[0] {
				t.Errorf("%v: line is %d cells wide", size, w)
			}
		}
	}
}

func TestStatusShowsUntilNextKey(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": "one\n"})
	m.SetStatus("File watching is unavailable.")
	if !strings.Contains(screen(m), "File watching is unavailable.") {
		t.Fatal("status should replace the hint line")
	}
	press(m, "tab")
	if strings.Contains(screen(m), "File watching is unavailable.") {
		t.Error("status should clear on the next key")
	}
}

func TestQuitKeys(t *testing.T) {
	m, _ := newModel(t, nil)
	for _, k := range []string{"q", "ctrl+c"} {
		if _, cmd := m.Update(key(k)); cmd == nil {
			t.Errorf("%s should return the quit command", k)
		}
	}
}
