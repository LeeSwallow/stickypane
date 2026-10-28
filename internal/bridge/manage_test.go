package bridge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/internal/kinds"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/widget"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

func newAPIKinds() widget.Registry { return kinds.Default(note.Plain) }

func read(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestBridgeArrangeWritesLikeSet: note.arrange leaves exactly the files
// that `stickypane set` leaves for the same keys.
func TestBridgeArrangeWritesLikeSet(t *testing.T) {
	viaBridge := board(t, map[string]string{"plan.md": plan})
	viaSet := board(t, map[string]string{"plan.md": plan})
	s := serve(t, viaBridge)
	s.ok("note.arrange", map[string]any{"note": "plan.md", "size": "card", "rows": 4, "open": true, "pin": true, "color": "Blue", "title": "Roadmap"}, nil)
	if _, err := newAPI(viaSet).Set("plan.md", []string{"size=card", "rows=4", "open=true", "pin=true", "color=Blue", "title=Roadmap"}); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"sticky.json", "plan.md"} {
		if got, want := read(t, viaBridge, f), read(t, viaSet, f); got != want {
			t.Errorf("%s differs from what set writes:\n%s\nwant:\n%s", f, got, want)
		}
	}
	// What is not given stays; empty values give the choice back.
	s.ok("note.arrange", map[string]any{"note": "plan.md", "size": "", "pin": false, "rows": 0, "title": ""}, nil)
	v, _ := store.Open(viaBridge).Views()
	if got := v["plan.md"]; got.Size != "" || got.Rows != 0 || got.Pin == nil || *got.Pin || got.Open == nil || !*got.Open || got.Color != "blue" {
		t.Errorf("after the second arrange: %+v", got)
	}
	var b loadedBoard
	s.ok("board.load", nil, &b)
	if p := b.note(t, "plan.md"); p.Title != "plan" || p.Color != "blue" {
		t.Errorf("board.load after arrange: %+v", p)
	}
}

func TestBridgeArrangeOfAPageIsItsBooks(t *testing.T) {
	dir := board(t, map[string]string{"t/docs/a.md": "a\n", "t/docs/b.md": "b\n"})
	s := serve(t, dir)
	s.ok("note.arrange", map[string]any{"note": "t/docs/b.md", "size": "half", "title": "Bee"}, nil)
	v, _ := store.Open(dir).Views()
	if v["t/docs"].Size != "half" || v["t/docs/b.md"].Size != "" {
		t.Errorf("a page is arranged by its book: %+v", v)
	}
	if got := read(t, dir, "t/docs/b.md"); !strings.Contains(got, "title: Bee") {
		t.Errorf("a page's title is its own: %q", got)
	}
	// A book is arranged by name.
	s.ok("note.arrange", map[string]any{"note": "t/docs", "pin": true}, nil)
}

func TestBridgeArrangeRefusesWhatSetRefuses(t *testing.T) {
	dir := board(t, map[string]string{"plan.md": plan})
	s := serve(t, dir)
	for name, p := range map[string]map[string]any{
		"size":    {"note": "plan.md", "size": "huge"},
		"rows":    {"note": "plan.md", "rows": -1},
		"color":   {"note": "plan.md", "color": "mauve"},
		"title":   {"note": "plan.md", "title": "a\nb"},
		"nothing": {"note": "plan.md"},
		"unknown": {"note": "plan.md", "wobble": 1},
	} {
		if r := s.call("note.arrange", p); r.code != CodeBadRequest {
			t.Errorf("%s: code = %q, want bad_request", name, r.code)
		}
	}
	if r := s.call("note.arrange", map[string]any{"note": "ghost.md", "pin": true}); r.code != CodeNotFound {
		t.Errorf("a note that is not there: %q", r.code)
	}
	if r := s.call("note.view", map[string]any{"note": "plan.md"}); r.code != CodeBadRequest {
		t.Errorf("note.view is gone: %q", r.code)
	}
	if _, err := os.Stat(filepath.Join(dir, "sticky.json")); err == nil {
		t.Errorf("refused requests wrote sticky.json: %s", read(t, dir, "sticky.json"))
	}
}

// TestBridgeOrderMatchesTheTerminalSwap: tab.order writes what the
// terminal's { and } write, store.SetOrder of the full names.
func TestBridgeOrderMatchesTheTerminalSwap(t *testing.T) {
	files := map[string]string{"t/a.md": "a\n", "t/b.md": "b\n", "t/c.md": "c\n", "x.md": "x\n"}
	viaBridge, viaTerminal := board(t, files), board(t, files)
	s := serve(t, viaBridge)
	s.ok("tab.order", map[string]any{"tab": "t", "order": []string{"t/b.md", "t/a.md", "t/c.md"}}, nil)
	// The terminal swaps the focused note with its neighbour and saves every name.
	if err := store.Open(viaTerminal).SetOrder("t", []string{"t/b.md", "t/a.md", "t/c.md"}); err != nil {
		t.Fatal(err)
	}
	if got, want := read(t, viaBridge, "t/sticky.json"), read(t, viaTerminal, "t/sticky.json"); got != want {
		t.Errorf("order file:\n%s\nwant:\n%s", got, want)
	}
	var b loadedBoard
	s.ok("board.load", nil, &b)
	var names []string
	for _, tab := range b.Tabs {
		if tab.Name == "t" {
			for _, n := range tab.Notes {
				names = append(names, n.Name)
			}
		}
	}
	if strings.Join(names, ",") != "t/b.md,t/a.md,t/c.md" {
		t.Errorf("board.load order = %v", names)
	}
	for name, p := range map[string]map[string]any{
		"stranger":  {"tab": "t", "order": []string{"x.md"}},
		"duplicate": {"tab": "t", "order": []string{"t/a.md", "t/a.md"}},
		"empty":     {"tab": "t", "order": []string{}},
	} {
		if r := s.call("tab.order", p); r.code != CodeBadRequest {
			t.Errorf("%s: code = %q", name, r.code)
		}
	}
	if r := s.call("tab.order", map[string]any{"tab": "nope", "order": []string{"a.md"}}); r.code != CodeNotFound {
		t.Errorf("no such tab: %q", r.code)
	}
}

func TestBridgeMoveIntoABook(t *testing.T) {
	dir := board(t, map[string]string{"t/a.md": "a\n", "t/keep.md": "k\n"})
	s := serve(t, dir)
	var got struct{ Note string }
	s.ok("note.move", map[string]any{"note": "t/a.md", "to": "t/book/"}, &got)
	if got.Note != "t/book/a.md" {
		t.Errorf("moved to %q", got.Note)
	}
	if _, err := os.Stat(filepath.Join(dir, "t", "book", "a.md")); err != nil {
		t.Error(err)
	}
	s.ok("note.move", map[string]any{"note": "t/book/a.md", "to": "."}, &got)
	if got.Note != "a.md" {
		t.Errorf("moved out to %q", got.Note)
	}
	if r := s.call("note.move", map[string]any{"note": "ghost.md", "to": "."}); r.code != CodeNotFound {
		t.Errorf("a note that is not there: %q", r.code)
	}
	if r := s.call("note.move", map[string]any{"note": "a.md", "to": ""}); r.code != CodeBadRequest {
		t.Errorf("nowhere: %q", r.code)
	}
	if r := s.call("note.move", map[string]any{"note": "a.md", "to": "t/keep.md"}); r.code != CodeFailed {
		t.Errorf("over a note: %q", r.code)
	}
}

func TestBridgeTrashThenRestore(t *testing.T) {
	dir := board(t, map[string]string{"plan.md": plan, "old.md": "old\n"})
	s := serve(t, dir)
	s.ok("note.trash", map[string]any{"note": "plan.md"}, nil)
	if _, err := os.Stat(filepath.Join(dir, "plan.md")); err == nil {
		t.Fatal("plan.md is still on the board")
	}
	if _, err := os.Stat(filepath.Join(dir, store.TrashDir, "plan.md")); err != nil {
		t.Fatalf("not in the trash: %v", err)
	}
	s.ok("note.restore", map[string]any{"note": "plan.md"}, nil)
	if read(t, dir, "plan.md") != plan {
		t.Error("restored note differs")
	}
	if r := s.call("note.restore", map[string]any{"note": "plan.md"}); r.code != CodeNotFound {
		t.Errorf("restoring what is not in the trash: %q", r.code)
	}
	s.ok("note.archive", map[string]any{"note": "old.md"}, nil)
	if _, err := os.Stat(filepath.Join(dir, store.ArchiveDir, "old.md")); err != nil {
		t.Errorf("not archived: %v", err)
	}
	for _, m := range []string{"note.trash", "note.archive"} {
		if r := s.call(m, map[string]any{"note": "ghost.md"}); r.code != CodeNotFound {
			t.Errorf("%s of a ghost: %q", m, r.code)
		}
	}
}

func TestBridgeSendRunsTheRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"hi":"there"}`))
	}))
	defer srv.Close()
	dir := board(t, map[string]string{"live.http": "### ping\nGET " + srv.URL + "/ping\n\n### pong\nGET " + srv.URL + "/pong\n", "plan.md": plan})
	s := serve(t, dir)
	var got struct {
		Status int    `json:"status"`
		Text   string `json:"text"`
		Failed bool   `json:"failed"`
		End    string `json:"end"`
		Log    string `json:"log"`
	}
	s.ok("note.send", map[string]any{"note": "live.http", "request": "pong"}, &got)
	if got.Status != 200 || got.Failed || !strings.Contains(got.End, "pong") || got.Log != "live.log" {
		t.Errorf("send = %+v", got)
	}
	if log := read(t, dir, "live.log"); !strings.Contains(log, "200") || !strings.Contains(log, "pong") {
		t.Errorf("log: %s", log)
	}
	s.ok("note.send", map[string]any{"note": "live.http", "request": 1}, &got) // 1-based, as #1
	if !strings.Contains(got.End, "ping") {
		t.Errorf("request 1 = %+v", got)
	}
	if r := s.call("note.send", map[string]any{"note": "live.http", "request": "nothing like it"}); r.code != CodeBadRequest {
		t.Errorf("no such request: %q", r.code)
	}
	if r := s.call("note.send", map[string]any{"note": "plan.md", "request": "x"}); r.code != CodeBadRequest {
		t.Errorf("not an api note: %q", r.code)
	}
}

func TestBridgeKindsListsEveryShapeWithItsExampleData(t *testing.T) {
	s := serve(t, board(t, nil))
	var kinds []struct {
		Name    string          `json:"name"`
		Icon    string          `json:"icon"`
		Label   string          `json:"label"`
		Blurb   string          `json:"blurb"`
		Example string          `json:"example"`
		Data    json.RawMessage `json:"data"`
	}
	s.ok("kinds.list", nil, &kinds)
	reg := newAPIKinds()
	if len(kinds) != len(reg) {
		t.Fatalf("%d kinds, registry has %d", len(kinds), len(reg))
	}
	byName := map[string]int{}
	for i, k := range kinds {
		byName[k.Name] = i
		if k.Name != reg[i].Name || k.Icon == "" || k.Label == "" || k.Blurb == "" || k.Example == "" || len(k.Data) == 0 {
			t.Errorf("kind %d = %+v", i, k)
		}
	}
	var cl struct {
		Items []struct{ Text string } `json:"items"`
	}
	if i, ok := byName["checklist"]; !ok || json.Unmarshal(kinds[i].Data, &cl) != nil || len(cl.Items) == 0 {
		t.Errorf("the checklist's example data: %s", kinds[byName["checklist"]].Data)
	}
	var sc struct {
		Lang string `json:"lang"`
	}
	if i, ok := byName["script"]; ok {
		json.Unmarshal(kinds[i].Data, &sc)
		if sc.Lang != "sh" {
			t.Errorf("script example lang = %q", sc.Lang)
		}
	}
}

func TestBridgeIndexHasBacklinks(t *testing.T) {
	dir := board(t, map[string]string{"a.md": "see [[b]]\n", "b.md": "# B\nhello\n"})
	s := serve(t, dir)
	var idx struct {
		Entries []struct {
			Name      string   `json:"name"`
			Kind      string   `json:"kind"`
			Links     []string `json:"links"`
			Backlinks []string `json:"backlinks"`
			Gist      string   `json:"gist"`
		} `json:"entries"`
	}
	s.ok("board.index", nil, &idx)
	got := map[string][]string{}
	for _, e := range idx.Entries {
		got[e.Name] = e.Backlinks
	}
	if len(idx.Entries) != 2 || len(got["b.md"]) != 1 || got["b.md"][0] != "a.md" {
		t.Errorf("index = %+v", idx)
	}
}

func TestBridgeDerivesChartsAfterAChange(t *testing.T) {
	dir := board(t, map[string]string{
		"chart.md":                  "---\ntype: chart\nfrom: todo\n---\nstale: 1\n",
		store.TrashDir + "/todo.md": "---\ntype: checklist\n---\n- [x] a\n- [ ] b\n- [ ] c\n",
	})
	s := serve(t, dir)
	s.ok("note.restore", map[string]any{"note": "todo.md"}, nil)
	got := read(t, dir, "chart.md")
	if strings.Contains(got, "stale") || !strings.Contains(got, "done") || !strings.Contains(got, "open") {
		t.Errorf("the chart was not derived after the restore:\n%s", got)
	}
}

func TestBridgeSayUsesTheTerminalName(t *testing.T) {
	t.Setenv("STICKYPANE_USER", "dana")
	dir := board(t, map[string]string{"talk.md": "---\ntype: chat\n---\n"})
	s := serve(t, dir)
	say := func(args map[string]string) {
		t.Helper()
		var b loadedBoard
		s.ok("board.load", nil, &b)
		s.ok("note.act", map[string]any{"note": "talk.md", "version": b.note(t, "talk.md").Version, "action": map[string]any{"name": "say", "args": args}}, nil)
	}
	say(map[string]string{"text": "hello"})
	say(map[string]string{"text": "from the agent", "who": "claude"})
	got := read(t, dir, "talk.md")
	if !strings.Contains(got, "@dana ") || !strings.Contains(got, "@claude ") || strings.Contains(got, "@agent") {
		t.Errorf("chat:\n%s", got)
	}
}

func TestBridgeLoadsLinkedNotesAndMoreSettings(t *testing.T) {
	dir := board(t, map[string]string{"plan.md": plan})
	outside := filepath.Join(filepath.Dir(dir), "README.md")
	if err := os.WriteFile(outside, []byte("# outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("..", "README.md"), filepath.Join(dir, "readme.md")); err != nil {
		t.Skip("no symbolic links here")
	}
	if err := os.WriteFile(filepath.Join(dir, "broken.md"), []byte("---\ntype: checklist\ntitle: [\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	st := store.Open(dir)
	st.SetTab("")
	st.SetRoot("time", "iso")
	st.SetRoot("stale", "1h")
	s := serve(t, dir)
	var b loadedBoard
	s.ok("board.load", nil, &b)
	if n := b.note(t, "readme.md"); !n.Linked || n.Target == "" || !strings.HasSuffix(n.Target, "README.md") {
		t.Errorf("linked note = %+v", n)
	}
	if n := b.note(t, "plan.md"); n.Linked || n.Target != "" || n.Title != "Plan" {
		t.Errorf("plain note = %+v", n)
	}
	if b.Settings.Time != "iso" || b.Settings.Stale != "1h" || b.Settings.Editor != "auto" || b.Settings.Tab != "" {
		t.Errorf("settings = %+v", b.Settings)
	}
}

// TestBridgeChangedCarriesTheSettingsThatChanged: the app follows an
// agent's `show` and a theme change made in the terminal.
func TestBridgeChangedCarriesTheSettingsThatChanged(t *testing.T) {
	dir := board(t, map[string]string{"a/x.md": "x\n", "b/y.md": "y\n"})
	s := serve(t, dir)
	s.ok("hello", nil, nil)
	st := store.Open(dir)
	if err := st.SetTab("b"); err != nil {
		t.Fatal(err)
	}
	c := s.changed(5*time.Second, func(c changed) bool { return c.Settings != nil })
	if c.Settings.Tab != "b" || c.Palette != nil {
		t.Errorf("after show: settings %+v palette %+v", c.Settings, c.Palette)
	}
	if err := st.SetTheme("nord"); err != nil {
		t.Fatal(err)
	}
	c = s.changed(5*time.Second, func(c changed) bool { return c.Palette != nil })
	if c.Settings == nil || c.Settings.Theme != "nord" || c.Palette.Name != "nord" {
		t.Errorf("after the theme: settings %+v palette %+v", c.Settings, c.Palette)
	}
}
