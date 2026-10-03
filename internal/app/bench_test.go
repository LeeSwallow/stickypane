package app

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/kinds"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/theme"
)

// benchBoard is a busy board: notes of every shape, open, in two tabs.
func benchBoard(b *testing.B) *Model {
	b.Helper()
	dir := filepath.Join(b.TempDir(), store.DirName)
	must := func(err error) {
		if err != nil {
			b.Fatal(err)
		}
	}
	must(os.MkdirAll(filepath.Join(dir, "deploy"), 0o755))
	put := func(name, body string) { must(os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644)) }
	for i := range 8 {
		put(fmt.Sprintf("note%02d.md", i), "---\nopen: true\n---\n# Note\n\n"+numbered(40)+"\n- a list item that is long enough to wrap\n  over two lines\n")
		put(fmt.Sprintf("list%02d.md", i), checklistFile)
		put(fmt.Sprintf("board%02d.md", i), boardFile)
	}
	put("tokens.md", "---\ntype: chart\nopen: true\n---\na: 1\nb: 2\nc: 3\n")
	put("build.log", numbered(500))
	put("deploy/run.md", "deploy\n")
	m := New(store.Open(dir), kinds.Default(kinds.Markdown(theme.NewHolder(theme.Pick("auto", true)))), nil, theme.NewHolder(theme.Pick("auto", true)))
	m.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	return m
}

// A file changed: the board reloads and draws again. This is what every
// write by an agent costs.
func BenchmarkChangeAndDraw(b *testing.B) {
	m := benchBoard(b)
	b.ReportAllocs()
	for b.Loop() {
		m.Update(changedMsg{})
		_ = m.render()
	}
}

// A key moves the focus: no file is read, the screen is drawn again.
func BenchmarkKeyAndDraw(b *testing.B) {
	m := benchBoard(b)
	b.ReportAllocs()
	for b.Loop() {
		press(m, "tab")
		_ = m.render()
	}
}
