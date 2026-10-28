package bridge

// The scenes of docs/desktop/01 §1.7, end to end at the level a test can
// reach without a window: the real `stickypane` of this repository, built
// from ../../.., runs as the bridge on a project in a temporary folder;
// other processes run the commands an agent runs; the test is the app's
// side and checks what arrives. HOME and STICKYPANE_HOME point into the
// temporary folder, so nothing of the person's own is read or written.
//
// With -update the payloads the app received are written to
// frontend/src/scenes/fixtures, where the frontend's tests draw them.

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

var update = flag.Bool("update", false, "write the scenes' payloads to frontend/src/scenes/fixtures")

var (
	realOnce sync.Once
	realBin  string
	realErr  error
)

// realStickypane builds the repository's stickypane once for the package.
func realStickypane(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("builds stickypane; not in -short")
	}
	realOnce.Do(func() {
		dir, err := os.MkdirTemp("", "scenes")
		if err != nil {
			realErr = err
			return
		}
		realBin = filepath.Join(dir, "stickypane")
		if runtime.GOOS == "windows" {
			realBin += ".exe"
		}
		cmd := exec.Command("go", "build", "-o", realBin, "./cmd/stickypane")
		cmd.Dir = filepath.Join("..", "..", "..")
		if out, err := cmd.CombinedOutput(); err != nil {
			realErr = fmt.Errorf("go build: %v\n%s", err, out)
		}
	})
	if realErr != nil {
		t.Fatal(realErr)
	}
	return realBin
}

// scene is one project in a temporary folder with its own HOME.
type scene struct {
	t       *testing.T
	bin     string
	project string
	board   string // project/.sticky
}

func newScene(t *testing.T) *scene {
	bin := realStickypane(t)
	root := t.TempDir()
	for _, d := range []string{"home", "sp", "project"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", filepath.Join(root, "home"))
	t.Setenv("STICKYPANE_HOME", filepath.Join(root, "sp"))
	t.Setenv("STICKYPANE_USER", "min")
	t.Setenv("LANG", "en_US.UTF-8")
	s := &scene{t: t, bin: bin, project: filepath.Join(root, "project"), board: filepath.Join(root, "project", ".sticky")}
	s.agent("init", "--no-agent-docs")
	return s
}

// agent runs a command in the project as an agent would, from another process.
func (s *scene) agent(args ...string) string {
	s.t.Helper()
	return s.agentIn("", args...)
}

func (s *scene) agentIn(stdin string, args ...string) string {
	s.t.Helper()
	cmd := exec.Command(s.bin, args...)
	cmd.Dir = s.project
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.CombinedOutput()
	if err != nil {
		s.t.Fatalf("stickypane %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// write puts a file into .sticky/ as an agent that writes files does.
func (s *scene) write(name, text string) {
	s.t.Helper()
	p := filepath.Join(s.board, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		s.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		s.t.Fatal(err)
	}
}

// open starts the bridge as the app does, and waits until it answers.
func (s *scene) open() *Client {
	s.t.Helper()
	c, err := Start(ctx(s.t), s.bin, s.project)
	if err != nil {
		s.t.Fatal(err)
	}
	s.t.Cleanup(func() { c.Close() })
	return c
}

// The parts of board.load and board.changed the scenes look at.
type sceneNote struct {
	Name    string          `json:"name"`
	Kind    string          `json:"kind"`
	Title   string          `json:"title"`
	Version string          `json:"version"`
	Data    json.RawMessage `json:"data"`
}

type sceneBoard struct {
	Tabs []struct {
		Notes []sceneNote `json:"notes"`
	} `json:"tabs"`
}

func (b sceneBoard) note(t *testing.T, name string) sceneNote {
	t.Helper()
	for _, tab := range b.Tabs {
		for _, n := range tab.Notes {
			if n.Name == name {
				return n
			}
		}
	}
	t.Fatalf("board.load has no %s", name)
	return sceneNote{}
}

type sceneEvent struct {
	Note string `json:"note"`
	Type string `json:"type"`
	Item string `json:"item"`
	From string `json:"from"`
	To   string `json:"to"`
	Who  string `json:"who"`
}

type sceneChange struct {
	Notes   []sceneNote  `json:"notes"`
	Removed []string     `json:"removed"`
	Events  []sceneEvent `json:"events"`
}

// load is board.load, raw and read.
func load(t *testing.T, c *Client) (json.RawMessage, sceneBoard) {
	t.Helper()
	var raw json.RawMessage
	if err := c.Call(ctx(t), "board.load", nil, &raw); err != nil {
		t.Fatal(err)
	}
	var b sceneBoard
	if err := json.Unmarshal(raw, &b); err != nil {
		t.Fatal(err)
	}
	return raw, b
}

// changed waits for a board.changed that ok accepts and returns it, raw and read.
func changed(t *testing.T, c *Client, ok func(sceneChange) bool) (json.RawMessage, sceneChange) {
	t.Helper()
	deadline := time.After(10 * time.Second)
	for {
		select {
		case n := <-c.Notifications():
			if n.Method != "board.changed" {
				continue
			}
			var ch sceneChange
			if err := json.Unmarshal(n.Params, &ch); err != nil {
				t.Fatal(err)
			}
			if ok(ch) {
				return n.Params, ch
			}
		case <-deadline:
			t.Fatal("no board.changed came in 10s")
		}
	}
}

// record writes what the app received for a scene, with -update.
func record(t *testing.T, name string, payloads map[string]json.RawMessage) {
	t.Helper()
	if !*update {
		return
	}
	var buf bytes.Buffer
	buf.WriteString("{\n")
	keys := make([]string, 0, len(payloads))
	for k := range payloads {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i, k := range keys {
		var v bytes.Buffer
		if err := json.Indent(&v, payloads[k], "  ", "  "); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&buf, "  %q: %s", k, v.String())
		if i < len(keys)-1 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}
	buf.WriteString("}\n")
	out := filepath.Join("..", "..", "frontend", "src", "scenes", "fixtures", name+".json")
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		t.Fatal(err)
	}
	// the temporary folder is the test's; the fixture says <project>
	text := strings.ReplaceAll(buf.String(), os.Getenv("HOME"), "<home>")
	if err := os.WriteFile(out, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Scene A: three agents work on one board. An agent moves a card from
// another process; the app hears card.moved with who and where, and the
// card that has not moved for half an hour is stalled in what it loads.
func TestSceneAAgentsOnOneBoard(t *testing.T) {
	s := newScene(t)
	old := time.Now().Add(-40 * time.Minute)
	stamp := fmt.Sprintf("@{%s} @@{%s}", old.Format("2006-01-02"), old.Format("15:04"))
	s.write("work.md", "---\ntype: board\n---\n## Todo\n- docs @claude-1\n## Doing\n- payments @codex-1\n- schema migration @codex-2 "+stamp+"\n## Review\n\n## Done\n- login @claude-1\n")
	c := s.open()
	before, b := load(t, c)
	var data struct {
		Columns []struct {
			Title string `json:"title"`
			Cards []struct {
				Text    string   `json:"text"`
				Who     []string `json:"who"`
				Stalled bool     `json:"stalled"`
			} `json:"cards"`
		} `json:"columns"`
	}
	if err := json.Unmarshal(b.note(t, "work.md").Data, &data); err != nil {
		t.Fatal(err)
	}
	stalled := map[string]bool{}
	for _, col := range data.Columns {
		for _, card := range col.Cards {
			stalled[card.Text] = card.Stalled
		}
	}
	if !stalled["schema migration"] || stalled["payments"] || stalled["docs"] {
		t.Errorf("stalled cards = %v, want only schema migration", stalled)
	}

	s.agent("card", "work", "move", "payments", "--to", "Review")
	raw, ch := changed(t, c, func(ch sceneChange) bool {
		for _, e := range ch.Events {
			if e.Type == "card.moved" {
				return true
			}
		}
		return false
	})
	var moved *sceneEvent
	for i, e := range ch.Events {
		if e.Type == "card.moved" {
			moved = &ch.Events[i]
		}
	}
	if moved.Note != "work.md" || moved.Item != "payments" || moved.From != "Doing" || moved.To != "Review" {
		t.Errorf("card.moved = %+v", *moved)
	}
	if len(ch.Notes) != 1 || ch.Notes[0].Name != "work.md" || ch.Notes[0].Version == b.note(t, "work.md").Version {
		t.Errorf("board.changed notes = %+v; want work.md at a new version", ch.Notes)
	}
	after, _ := load(t, c)
	record(t, "scene-a", map[string]json.RawMessage{"before": before, "changed": raw, "after": after})
}

// Scene B: an agent asks with a form and waits; the user answers in the
// app; `stickypane wait` returns the answer and the file says it.
func TestSceneBTheAgentAsksTheUserAnswers(t *testing.T) {
	s := newScene(t)
	s.agent("write", "ship", "--type", "form", "--title", "How should 0.2.0 go out?",
		"--opt", "question=Where to?", "--opt", "choices=GitHub release only\nHomebrew too", "--opt", "buttons=Ship\nHold")
	c := s.open()

	// the agent waits, as it would, in another process
	wait := exec.Command(s.bin, "wait", "ship", "--json", "--timeout", "30s")
	wait.Dir = s.project
	var out bytes.Buffer
	wait.Stdout, wait.Stderr = &out, &out
	if err := wait.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- wait.Wait() }()

	waiting, b := load(t, c)
	n := b.note(t, "ship.md")
	var form struct {
		Lines []struct {
			Kind string `json:"kind"`
			Text string `json:"text"`
		} `json:"lines"`
		Submitted string `json:"submitted"`
	}
	if err := json.Unmarshal(n.Data, &form); err != nil {
		t.Fatal(err)
	}
	if n.Kind != "form" || form.Submitted != "" {
		t.Fatalf("the form before an answer: kind %q, submitted %q", n.Kind, form.Submitted)
	}
	line := -1
	for i, l := range form.Lines {
		if l.Text == "Homebrew too" {
			line = i
		}
	}
	if line < 0 {
		t.Fatalf("no option Homebrew too in %+v", form.Lines)
	}

	// the clicks: choose, then the button; each answers the note's new version
	var res struct {
		Version string `json:"version"`
	}
	act := func(name string, args map[string]string) {
		t.Helper()
		p := map[string]any{"note": "ship.md", "version": n.Version, "action": map[string]any{"name": name, "args": args}}
		if err := c.Call(ctx(t), "note.act", p, &res); err != nil {
			t.Fatalf("note.act %s: %v", name, err)
		}
		n.Version = res.Version
	}
	act("choose", map[string]string{"line": fmt.Sprint(line)})
	act("submit", map[string]string{"label": "Ship"})

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("stickypane wait: %v\n%s", err, out.String())
		}
	case <-time.After(20 * time.Second):
		wait.Process.Kill()
		t.Fatal("stickypane wait did not return after the answer")
	}
	var answer struct {
		Submitted bool   `json:"submitted"`
		Button    string `json:"button"`
		Answers   []struct {
			Values []string `json:"values"`
		} `json:"answers"`
	}
	if err := json.Unmarshal(out.Bytes(), &answer); err != nil || !answer.Submitted || answer.Button != "Ship" ||
		len(answer.Answers) != 1 || len(answer.Answers[0].Values) != 1 || answer.Answers[0].Values[0] != "Homebrew too" {
		t.Errorf("wait --json printed %q, want Ship with Homebrew too", out.String())
	}
	file, err := os.ReadFile(filepath.Join(s.board, "ship.md"))
	if err != nil || !strings.Contains(string(file), "submitted: Ship") || !strings.Contains(string(file), "- (x) Homebrew too") {
		t.Errorf("ship.md after the answer:\n%s", file)
	}
	// the terminal side reads the same answer
	if a := s.agent("answers", "ship"); !strings.Contains(a, "Homebrew too") {
		t.Errorf("stickypane answers = %q", a)
	}
	answered, _ := load(t, c)
	record(t, "scene-b", map[string]json.RawMessage{"waiting": waiting, "answered": answered, "wait": bytes.TrimSpace(out.Bytes())})
}

// Scene C: an agent leaves a report as an .html file; the app receives the
// page as it is, to draw in its sandbox.
func TestSceneCAnHTMLReport(t *testing.T) {
	s := newScene(t)
	c := s.open()
	load(t, c) // the board is watched once the bridge answers
	page := "<!doctype html>\n<title>Bench</title>\n<h1>Bench</h1>\n<p>p50 12 ms</p>\n<script>document.title = 'ran'</script>\n"
	s.write(filepath.Join("03-log", "bench.html"), page)
	raw, ch := changed(t, c, func(ch sceneChange) bool {
		for _, n := range ch.Notes {
			if n.Name == "03-log/bench.html" {
				return true
			}
		}
		return false
	})
	var got sceneNote
	for _, n := range ch.Notes {
		if n.Name == "03-log/bench.html" {
			got = n
		}
	}
	var data struct {
		HTML string `json:"html"`
	}
	if err := json.Unmarshal(got.Data, &data); err != nil || got.Kind != "html" || data.HTML != page {
		t.Errorf("bench.html arrived as kind %q, data %s", got.Kind, got.Data)
	}
	after, _ := load(t, c)
	record(t, "scene-c", map[string]json.RawMessage{"changed": raw, "after": after})
}
