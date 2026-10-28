package bridge

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/internal/api"
	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/kinds"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/widget"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

func TestMain(m *testing.M) {
	// The bridge records nothing itself, but nothing a test runs may write
	// into the user's own app folder either.
	tmp, err := os.MkdirTemp("", "bridge")
	if err != nil {
		panic(err)
	}
	os.Setenv("STICKYPANE_HOME", filepath.Join(tmp, "no-registry"))
	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}

// board makes a notes folder with the files and returns it.
func board(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), store.DirName)
	for name, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func newAPI(dir string) *api.API { return api.New(store.Open(dir), kinds.Default(note.Plain)) }

// session is a bridge served over pipes, as the desktop app runs it.
type session struct {
	t     *testing.T
	in    *io.PipeWriter
	lines chan map[string]json.RawMessage
	done  chan error

	mu     sync.Mutex
	notice []map[string]json.RawMessage // board.changed, in the order they came
	id     int
}

func serve(t *testing.T, dir string) *session {
	t.Helper()
	return serveAPI(t, newAPI(dir))
}

// serveAPI serves a bridge on a over pipes.
func serveAPI(t *testing.T, a *api.API) *session {
	t.Helper()
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	s := &session{t: t, in: inW, lines: make(chan map[string]json.RawMessage, 100), done: make(chan error, 1)}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		s.done <- Serve(ctx, a, kinds.Default(note.Plain), inR, outW)
		outW.Close()
	}()
	go func() {
		sc := bufio.NewScanner(outR)
		sc.Buffer(nil, 1<<24)
		for sc.Scan() {
			var m map[string]json.RawMessage
			if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
				t.Errorf("not one JSON object a line: %q", sc.Text())
				continue
			}
			s.lines <- m
		}
		close(s.lines)
	}()
	t.Cleanup(func() {
		inW.Close()
		select {
		case err := <-s.done:
			if err != nil {
				t.Errorf("Serve: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("Serve did not end when its input did")
		}
		cancel()
	})
	return s
}

// send writes a raw line.
func (s *session) send(line string) {
	s.t.Helper()
	if _, err := io.WriteString(s.in, line+"\n"); err != nil {
		s.t.Fatal(err)
	}
}

// next returns the next message that is not a notification, keeping the
// notifications.
func (s *session) next() map[string]json.RawMessage {
	s.t.Helper()
	for {
		select {
		case m, ok := <-s.lines:
			if !ok {
				s.t.Fatal("the bridge stopped answering")
			}
			if _, isNotice := m["method"]; isNotice {
				s.mu.Lock()
				s.notice = append(s.notice, m)
				s.mu.Unlock()
				continue
			}
			return m
		case <-time.After(10 * time.Second):
			s.t.Fatal("no answer")
		}
	}
}

// reply is an answer: the result, or the error's code.
type reply struct {
	id     string
	result json.RawMessage
	code   string
	msg    string
}

// call sends a request and returns its answer.
func (s *session) call(method string, params any) reply {
	s.t.Helper()
	s.id++
	p, _ := json.Marshal(params)
	s.send(fmt.Sprintf(`{"id":%d,"method":%q,"params":%s}`, s.id, method, p))
	m := s.next()
	r := reply{id: string(m["id"]), result: m["result"]}
	if r.id != fmt.Sprint(s.id) {
		s.t.Fatalf("answer to %d came with id %s", s.id, r.id)
	}
	if e, ok := m["error"]; ok {
		var er struct{ Code, Message string }
		json.Unmarshal(e, &er)
		r.code, r.msg = er.Code, er.Message
	}
	return r
}

// ok calls and decodes the result into v, failing on an error.
func (s *session) ok(method string, params, v any) {
	s.t.Helper()
	r := s.call(method, params)
	if r.code != "" {
		s.t.Fatalf("%s: %s: %s", method, r.code, r.msg)
	}
	if v != nil {
		if err := json.Unmarshal(r.result, v); err != nil {
			s.t.Fatalf("%s: %v in %s", method, err, r.result)
		}
	}
}

// changed waits for a board.changed whose params satisfy match.
func (s *session) changed(within time.Duration, match func(changed) bool) changed {
	s.t.Helper()
	deadline := time.After(within)
	seen := 0
	for {
		s.mu.Lock()
		list := s.notice[seen:]
		seen = len(s.notice)
		s.mu.Unlock()
		for _, m := range list {
			var c changed
			if string(m["method"]) != `"board.changed"` {
				s.t.Errorf("unknown notification %s", m["method"])
			}
			if err := json.Unmarshal(m["params"], &c); err != nil {
				s.t.Fatal(err)
			}
			if match(c) {
				return c
			}
		}
		select {
		case m, ok := <-s.lines:
			if !ok {
				s.t.Fatal("the bridge stopped")
			}
			if _, isNotice := m["method"]; !isNotice {
				s.t.Fatalf("an answer nobody asked for: %v", m)
			}
			s.mu.Lock()
			s.notice = append(s.notice, m)
			s.mu.Unlock()
		case <-deadline:
			s.t.Fatal("no board.changed came")
		}
	}
}

type changed struct {
	Settings *struct {
		Theme string `json:"theme"`
		Tab   string `json:"tab"`
	} `json:"settings"`
	Palette *struct {
		Name string `json:"name"`
	} `json:"palette"`
	Notes   []loaded `json:"notes"`
	Removed []string `json:"removed"`
	Events  []struct {
		Note string `json:"note"`
		widget.Event
	} `json:"events"`
}

type loaded struct {
	Name    string          `json:"name"`
	Kind    string          `json:"kind"`
	Icon    string          `json:"icon"`
	Size    string          `json:"size"`
	Rows    int             `json:"rows"`
	Color   string          `json:"color"`
	Pinned  bool            `json:"pinned"`
	Version string          `json:"version"`
	Summary string          `json:"summary"`
	Data    json.RawMessage `json:"data"`
	ModTime time.Time       `json:"modTime"`
	Pages   []loaded        `json:"pages"`
	Title   string          `json:"title"`
	Linked  bool            `json:"linked"`
	Target  string          `json:"target"`
	Error   string          `json:"error"`
}

type loadedBoard struct {
	Title string `json:"title"`
	Tabs  []struct {
		Name  string   `json:"name"`
		Title string   `json:"title"`
		Notes []loaded `json:"notes"`
	} `json:"tabs"`
	Settings struct {
		Theme    string `json:"theme"`
		Style    string `json:"style"`
		View     string `json:"view"`
		Language struct {
			Choice   string `json:"choice"`
			Resolved string `json:"resolved"`
		} `json:"language"`
		Tab    string `json:"tab"`
		Time   string `json:"time"`
		Editor string `json:"editor"`
		Stale  string `json:"stale"`
	} `json:"settings"`
	Palette map[string]json.RawMessage `json:"palette"`
}

func (b loadedBoard) note(t *testing.T, name string) loaded {
	t.Helper()
	for _, tab := range b.Tabs {
		for _, n := range tab.Notes {
			if n.Name == name {
				return n
			}
		}
	}
	t.Fatalf("no note %s in the board", name)
	return loaded{}
}

func versionOf(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
	if err != nil {
		t.Fatal(err)
	}
	return widget.Version(doc.Parse(b))
}

const plan = "---\ntype: checklist\ntitle: Plan\n---\n- [ ] one\n- [x] two\n"

func TestBridgeSaysHello(t *testing.T) {
	dir := board(t, nil)
	s := serve(t, dir)
	var h struct {
		Protocol int    `json:"protocol"`
		Version  string `json:"version"`
		Project  string `json:"project"`
	}
	s.ok("hello", nil, &h)
	if h.Protocol != Protocol || Protocol != 1 || h.Version == "" || h.Project != filepath.Dir(dir) {
		t.Errorf("hello = %+v", h)
	}
}

func TestBridgeLoadsTheBoard(t *testing.T) {
	t.Setenv("STICKYPANE_LANG", "ko")
	dir := board(t, map[string]string{
		"plan.md":               plan,
		"sticky.json":           `{"notes":{"plan.md":{"size":"half","rows":7,"pin":true,"color":"blue"}},"theme":"auto"}`,
		"deploy/run.sh":         "echo hi\n",
		"deploy/win.ps1":        "Write-Output hi\n",
		"deploy/docs/intro.md":  "# Intro\n",
		"deploy/docs/later.md":  "---\ntype: checklist\n---\n- [ ] x\n",
		"deploy/build.log":      "one\ntwo\n",
		"deploy/sticky.json":    `{"title":"Deploy"}`,
		"notes-with-no-kind.md": "plain words\n",
	})
	s := serve(t, dir)
	var b loadedBoard
	s.ok("board.load", nil, &b)

	if b.Title == "" || len(b.Tabs) != 2 || b.Tabs[0].Name != "" || b.Tabs[1].Name != "deploy" || b.Tabs[1].Title != "Deploy" {
		t.Fatalf("tabs: title %q, %+v", b.Title, b.Tabs)
	}
	p := b.note(t, "plan.md")
	if p.Kind != "checklist" || p.Icon == "" || p.Size != "1/2" || p.Rows != 7 || !p.Pinned || p.Color != "blue" ||
		p.Version != versionOf(t, dir, "plan.md") || p.Summary != "1/2" || p.ModTime.IsZero() {
		t.Errorf("plan: %+v", p)
	}
	var items struct {
		Items []struct {
			Text    string `json:"text"`
			Checked bool   `json:"checked"`
		} `json:"items"`
	}
	if err := json.Unmarshal(p.Data, &items); err != nil || len(items.Items) != 2 || !items.Items[1].Checked {
		t.Errorf("plan data: %s", p.Data)
	}
	plain := b.note(t, "notes-with-no-kind.md")
	if plain.Kind != "note" || plain.Color == "" || plain.Pages != nil {
		t.Errorf("a plain note: %+v", plain)
	}

	// A script says the language it runs in, from its file's extension.
	for name, lang := range map[string]string{"deploy/run.sh": "sh", "deploy/win.ps1": "ps1"} {
		var d struct{ Lang, Source string }
		if err := json.Unmarshal(b.note(t, name).Data, &d); err != nil || d.Lang != lang || d.Source == "" {
			t.Errorf("%s: data %s", name, b.note(t, name).Data)
		}
	}

	// A book carries its pages, in the same shape, by file name.
	book := b.note(t, "deploy/docs")
	if book.Kind != "book" || len(book.Pages) != 2 || book.Pages[0].Name != "deploy/docs/intro.md" ||
		book.Pages[1].Kind != "checklist" || book.Pages[1].Version != versionOf(t, dir, "deploy/docs/later.md") {
		t.Errorf("book: %+v", book)
	}

	if b.Settings.Theme != "auto" || b.Settings.Style != "doodle" || b.Settings.View != "terminal" ||
		b.Settings.Language.Choice != "auto" || b.Settings.Language.Resolved != "ko" {
		t.Errorf("settings: %+v", b.Settings)
	}
	var notes []string
	if err := json.Unmarshal(b.Palette["notes"], &notes); err != nil || len(notes) != 6 || b.Palette["accent"] == nil {
		t.Errorf("palette: %v", b.Palette)
	}
}

func TestBridgeActChangesTheFileAndAnswersTheNewVersion(t *testing.T) {
	dir := board(t, map[string]string{"plan.md": plan})
	s := serve(t, dir)
	var got struct{ Version string }
	s.ok("note.act", map[string]any{
		"note": "plan.md", "version": versionOf(t, dir, "plan.md"),
		"action": widget.Action{Name: "toggle", Args: map[string]string{"i": "0"}},
	}, &got)
	b, _ := os.ReadFile(filepath.Join(dir, "plan.md"))
	if !strings.Contains(string(b), "- [x] one") {
		t.Errorf("the item is not ticked: %q", b)
	}
	if got.Version != versionOf(t, dir, "plan.md") {
		t.Errorf("version = %q, the file's is %q", got.Version, versionOf(t, dir, "plan.md"))
	}
}

func TestBridgeStaleActIsRefused(t *testing.T) {
	dir := board(t, map[string]string{"plan.md": plan})
	s := serve(t, dir)
	old := versionOf(t, dir, "plan.md")
	if _, err := newAPI(dir).Todo("plan", "add", "three"); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(filepath.Join(dir, "plan.md"))
	r := s.call("note.act", map[string]any{
		"note": "plan.md", "version": old,
		"action": widget.Action{Name: "toggle", Args: map[string]string{"i": "0"}},
	})
	if r.code != "stale" {
		t.Errorf("code = %q (%s), want stale", r.code, r.msg)
	}
	if after, _ := os.ReadFile(filepath.Join(dir, "plan.md")); string(after) != string(before) {
		t.Errorf("a stale action wrote: %q", after)
	}
}

func TestBridgeReportsAnAgentsMoveAsAnEvent(t *testing.T) {
	dir := board(t, map[string]string{"work.md": "---\ntype: board\n---\n## Doing\n- payments\n## Review\n"})
	s := serve(t, dir)
	s.ok("hello", nil, nil) // the board is being watched once the bridge answers
	if _, err := newAPI(dir).Card("work", "move", "payments", "Review"); err != nil {
		t.Fatal(err)
	}
	c := s.changed(2*time.Second, func(c changed) bool {
		for _, e := range c.Events {
			if e.Note == "work.md" && e.Type == "card.moved" {
				return true
			}
		}
		return false
	})
	var moved bool
	for _, e := range c.Events {
		moved = moved || (e.Type == "card.moved" && e.Item == "payments" && e.From == "Doing" && e.To == "Review")
	}
	if !moved || len(c.Notes) != 1 || c.Notes[0].Name != "work.md" || c.Notes[0].Version != versionOf(t, dir, "work.md") {
		t.Errorf("board.changed: %+v", c)
	}

	// A note that goes is named in removed.
	if err := os.Remove(filepath.Join(dir, "work.md")); err != nil {
		t.Fatal(err)
	}
	s.changed(2*time.Second, func(c changed) bool { return len(c.Removed) == 1 && c.Removed[0] == "work.md" })
}

func TestBridgeAnswersBadInputAndKeepsRunning(t *testing.T) {
	dir := board(t, map[string]string{"plan.md": plan, "n.md": "plain\n"})
	s := serve(t, dir)

	s.send("this is not JSON")
	if m := s.next(); string(m["id"]) != "null" || !strings.Contains(string(m["error"]), `"bad_request"`) {
		t.Errorf("a line that is not JSON: %v", m)
	}
	v := versionOf(t, dir, "plan.md")
	for _, c := range []struct {
		method string
		params any
		code   string
	}{
		{"no.such.method", nil, "bad_request"},
		{"note.act", "not an object", "bad_request"},
		{"note.act", map[string]any{"note": "gone.md", "version": v, "action": map[string]any{"name": "toggle"}}, "not_found"},
		{"note.act", map[string]any{"note": "../../etc/passwd", "version": v, "action": map[string]any{"name": "toggle"}}, "bad_request"},
		{"note.act", map[string]any{"note": "plan.md", "version": v, "action": map[string]any{"name": "dance"}}, "bad_request"},
		{"note.act", map[string]any{"note": "plan.md", "version": v, "action": map[string]any{"name": "toggle", "args": map[string]string{"i": "9"}}}, "bad_request"},
		{"note.act", map[string]any{"note": "n.md", "version": versionOf(t, dir, "n.md"), "action": map[string]any{"name": "toggle"}}, "bad_request"},
		{"note.arrange", map[string]any{"note": "plan.md", "size": "huge"}, "bad_request"},
		{"note.arrange", map[string]any{"note": "plan.md", "rows": -1}, "bad_request"},
		{"note.arrange", map[string]any{"note": "nope.md", "pin": true}, "not_found"},
		{"note.open", map[string]any{"note": "nope.md"}, "not_found"},
		{"note.run", map[string]any{"note": "plan.md"}, "bad_request"},
		{"note.create", map[string]any{"tab": "nowhere", "kind": "note", "title": "x"}, "not_found"},
		{"note.create", map[string]any{"kind": "spaceship", "title": "x"}, "bad_request"},
		{"settings.set", map[string]any{"theme": "no-such-theme"}, "bad_request"},
		{"settings.set", map[string]any{"language": "tlh"}, "bad_request"},
		{"settings.set", map[string]any{"view": "hologram"}, "bad_request"},
		{"settings.set", map[string]any{}, "bad_request"},
	} {
		if r := s.call(c.method, c.params); r.code != c.code || r.msg == "" {
			t.Errorf("%s %v: code %q (%s), want %q", c.method, c.params, r.code, r.msg, c.code)
		}
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "plan.md")); string(b) != plan {
		t.Errorf("bad requests changed the note: %q", b)
	}
	s.ok("hello", nil, nil) // still there
}

func TestBridgeCreatesANote(t *testing.T) {
	dir := board(t, map[string]string{"deploy/x.md": "x\n"})
	s := serve(t, dir)
	var got struct{ Note string }
	s.ok("note.create", map[string]any{"tab": "deploy", "kind": "checklist", "title": "Ship it", "body": "- [ ] tag\n"}, &got)
	if got.Note != "deploy/ship-it.md" {
		t.Fatalf("note = %q", got.Note)
	}
	b, err := os.ReadFile(filepath.Join(dir, "deploy", "ship-it.md"))
	if err != nil || !strings.Contains(string(b), "type: checklist") || !strings.Contains(string(b), "title: Ship it") || !strings.Contains(string(b), "- [ ] tag") {
		t.Errorf("file: %q, %v", b, err)
	}
	// The same title again never overwrites.
	s.ok("note.create", map[string]any{"tab": "deploy", "kind": "checklist", "title": "Ship it"}, &got)
	if got.Note != "deploy/ship-it-2.md" {
		t.Errorf("second note = %q", got.Note)
	}
	s.ok("note.create", map[string]any{"title": "Root note", "body": "hi"}, &got)
	if got.Note != "root-note.md" {
		t.Errorf("root note = %q", got.Note)
	}
}

func TestBridgeRunsAScript(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh")
	}
	dir := board(t, map[string]string{"run.sh": "echo ran\nexit 3\n"})
	s := serve(t, dir)
	var got struct {
		Code int   `json:"code"`
		Took int64 `json:"took"`
	}
	s.ok("note.run", map[string]any{"note": "run.sh"}, &got)
	if got.Code != 3 || got.Took < 0 {
		t.Errorf("run = %+v", got)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "run.log")); !strings.Contains(string(b), "ran") || !strings.Contains(string(b), "[exit 3") {
		t.Errorf("log: %q", b)
	}
}

func TestBridgeSetsSettings(t *testing.T) {
	t.Setenv("STICKYPANE_LANG", "en")
	dir := board(t, nil)
	s := serve(t, dir)
	s.ok("settings.set", map[string]any{"theme": "stickypane light", "style": "neat", "view": "both", "language": "ko"}, nil)
	st := store.Open(dir)
	if st.Theme() != "stickypane-light" || st.Style() != "neat" || st.ViewMode() != "both" || st.Language() != "ko" {
		t.Errorf("stored: theme %q style %q view %q language %q", st.Theme(), st.Style(), st.ViewMode(), st.Language())
	}
	var b loadedBoard
	s.ok("board.load", nil, &b)
	if b.Settings.Theme != "stickypane-light" || b.Settings.Language.Choice != "ko" || b.Settings.Language.Resolved != "ko" {
		t.Errorf("settings: %+v", b.Settings)
	}
	var dark bool
	json.Unmarshal(b.Palette["dark"], &dark)
	if dark {
		t.Errorf("the palette is the chosen light theme's: %v", b.Palette)
	}
	// Back to auto: it follows the environment again.
	s.ok("settings.set", map[string]any{"language": "auto"}, nil)
	s.ok("board.load", nil, &b)
	if b.Settings.Language.Choice != "auto" || b.Settings.Language.Resolved != "en" {
		t.Errorf("language after auto: %+v", b.Settings.Language)
	}
}

func TestOpenPrefersAGUIEditorAndFallsBackToTheSystem(t *testing.T) {
	var mu sync.Mutex
	var ran [][]string
	was := start
	start = func(cmd []string) error {
		mu.Lock()
		ran = append(ran, cmd)
		mu.Unlock()
		return nil
	}
	t.Cleanup(func() { start = was })
	dir := board(t, map[string]string{"plan.md": plan})
	s := serve(t, dir)
	file := filepath.Join(dir, "plan.md")

	t.Setenv("VISUAL", "/usr/local/bin/code --new-window")
	s.ok("note.open", map[string]any{"note": "plan.md"}, nil)
	mu.Lock()
	if len(ran) != 1 || strings.Join(ran[0], " ") != "/usr/local/bin/code --new-window "+file {
		t.Errorf("GUI editor: %q", ran)
	}
	mu.Unlock()

	// The overlay is for macOS and Linux (docs/overlay/adr/0014): Windows
	// has no system fallback.
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("no system way to open a file on " + runtime.GOOS + "; the overlay is for macOS and Linux")
	}
	t.Setenv("VISUAL", "nvim") // a terminal editor: the bridge has no terminal
	s.ok("note.open", map[string]any{"note": "plan.md"}, nil)

	mu.Lock()
	defer mu.Unlock()
	if len(ran) != 2 {
		t.Fatalf("ran %v", ran)
	}
	var want string
	switch runtime.GOOS {
	case "darwin":
		want = "open -t " + file
	case "linux":
		want = "xdg-open " + file
	}
	if strings.Join(ran[1], " ") != want {
		t.Errorf("fallback: %q, want %q", ran[1], want)
	}
}

// An html page is a note: the board lists it, as an html kind with its source.
func TestBridgeLoadsAnHTMLNote(t *testing.T) {
	dir := board(t, map[string]string{"report.html": "<h1>Bench</h1>\n", "old.htm": "<p>old</p>\n"})
	s := serve(t, dir)
	var b loadedBoard
	s.ok("board.load", nil, &b)
	for name, src := range map[string]string{"report.html": "<h1>Bench</h1>\n", "old.htm": "<p>old</p>\n"} {
		n := b.note(t, name)
		var d struct{ HTML string }
		if err := json.Unmarshal(n.Data, &d); err != nil || n.Kind != "html" || d.HTML != src {
			t.Errorf("%s: kind %q, data %s", name, n.Kind, n.Data)
		}
	}
}

// A shape that is a file of its own, a script or a page, is made as the
// catalog makes it: its own extension, its template or the given body.
func TestBridgeCreatesFileTypeNotes(t *testing.T) {
	dir := board(t, map[string]string{"deploy/x.md": "x\n"})
	s := serve(t, dir)
	var got struct{ Note string }
	s.ok("note.create", map[string]any{"tab": "deploy", "kind": "script", "title": "Build"}, &got)
	if got.Note != "deploy/build.sh" || read(t, dir, "deploy/build.sh") != "#!/bin/sh\n" {
		t.Errorf("script: %q, %q", got.Note, readOrEmpty(dir, "deploy/build.sh"))
	}
	s.ok("note.create", map[string]any{"kind": "html", "title": "Report", "body": "<h1>Hi</h1>\n"}, &got)
	if got.Note != "report.html" || read(t, dir, "report.html") != "<h1>Hi</h1>\n" {
		t.Errorf("html: %q, %q", got.Note, readOrEmpty(dir, "report.html"))
	}
	// Again: the next name, never over the first.
	s.ok("note.create", map[string]any{"kind": "html", "title": "Report"}, &got)
	if got.Note != "report-2.html" || read(t, dir, "report.html") != "<h1>Hi</h1>\n" {
		t.Errorf("second html: %q", got.Note)
	}
	if r := s.call("note.create", map[string]any{"kind": "nope", "title": "x"}); r.code != CodeBadRequest {
		t.Errorf("unknown kind: %q", r.code)
	}
}

// A script's language: its extension when it says (.sh, .ps1), as the
// runner runs it; else the shebang the script's own data read.
func TestScriptLanguageIsTheExtensionThenTheShebang(t *testing.T) {
	for _, c := range []struct{ name, hint, want string }{
		{"run.sh", "ps1", "sh"},
		{"run.SH", "ps1", "sh"},
		{"win.ps1", "sh", "ps1"},
		{"run", "ps1", "ps1"},
		{"run.txt", "sh", "sh"},
	} {
		var got struct{ Lang string }
		json.Unmarshal(data(c.name, map[string]string{"lang": c.hint}), &got)
		if got.Lang != c.want {
			t.Errorf("%s with hint %s: lang %q, want %q", c.name, c.hint, got.Lang, c.want)
		}
	}
}

// A book is a folder, not a file to edit: note.open asks for one of its
// pages instead, and opens nothing.
func TestOpenRefusesABook(t *testing.T) {
	ran := 0
	was := start
	start = func([]string) error { ran++; return nil }
	t.Cleanup(func() { start = was })
	t.Setenv("VISUAL", "/usr/local/bin/code")
	dir := board(t, map[string]string{"t/docs/a.md": "a\n", "t/keep.md": "k\n"})
	s := serve(t, dir)
	if r := s.call("note.open", map[string]any{"note": "t/docs"}); r.code != CodeBadRequest || !strings.Contains(r.msg, "page") {
		t.Errorf("open a book: %q %s", r.code, r.msg)
	}
	s.ok("note.open", map[string]any{"note": "t/docs/a.md"}, nil)
	if ran != 1 {
		t.Errorf("ran %d commands, want only the page's", ran)
	}
}
