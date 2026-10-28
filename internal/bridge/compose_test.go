package bridge

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

// kinds.list says what each shape asks before it is made, so a screen
// draws its create form from data.
func TestBridgeKindsListsEachShapesOptions(t *testing.T) {
	s := serve(t, board(t, nil))
	var kinds []struct {
		Name    string `json:"name"`
		Options []struct {
			Name    string   `json:"name"`
			Type    string   `json:"type"`
			Label   string   `json:"label"`
			Choices []string `json:"choices"`
			Default string   `json:"default"`
		} `json:"options"`
	}
	s.ok("kinds.list", nil, &kinds)
	byName := map[string]int{}
	for i, k := range kinds {
		byName[k.Name] = i
		if k.Options == nil {
			t.Errorf("%s: options should be a list, [] when there are none", k.Name)
		}
	}
	board := kinds[byName["board"]].Options
	if len(board) == 0 || board[0].Name != "columns" || board[0].Type != "lines" || board[0].Label != "Columns" || board[0].Default != "To do\nDoing\nDone" {
		t.Errorf("board options = %+v", board)
	}
	var view bool
	for _, o := range kinds[byName["chart"]].Options {
		if o.Name == "view" {
			view = o.Type == "choice" && strings.Join(o.Choices, ",") == "bar,spark,heat" && o.Default == "bar"
		}
	}
	if !view {
		t.Errorf("chart options = %+v", kinds[byName["chart"]].Options)
	}
}

// note.create with options has the shape's own code write the file; the
// screen never writes a note's format itself.
func TestBridgeCreatesANoteFromOptions(t *testing.T) {
	dir := board(t, map[string]string{"deploy/x.md": "x\n"})
	s := serve(t, dir)
	var got struct{ Note string }
	s.ok("note.create", map[string]any{"tab": "deploy", "kind": "board", "title": "Sprint", "options": map[string]any{
		"columns": []string{"Backlog", "Doing"},
		"cards":   "Doing: login API @codex-1",
	}}, &got)
	if got.Note != "deploy/sprint.md" {
		t.Fatalf("note = %q", got.Note)
	}
	d := doc.Parse([]byte(read(t, dir, "deploy/sprint.md")))
	if title, _ := d.Get("title"); d.Type() != "board" || title != "Sprint" || !strings.Contains(d.Body, "## Backlog\n\n## Doing\n- login API @codex-1") {
		t.Errorf("file: %q", read(t, dir, "deploy/sprint.md"))
	}
	if _, ok := d.Get("created"); !ok {
		t.Errorf("a new note says when it was made: %q", read(t, dir, "deploy/sprint.md"))
	}
	s.ok("note.create", map[string]any{"kind": "chart", "title": "Tokens", "options": map[string]any{"view": "spark", "values": "in: 3"}}, &got)
	if b := read(t, dir, "tokens.md"); !strings.Contains(b, "view: spark") || !strings.Contains(b, "in: 3") {
		t.Errorf("chart: %q", b)
	}
	s.ok("note.create", map[string]any{"kind": "api", "title": "Ping", "options": map[string]any{"status": 204}}, &got)
	if got.Note != "ping.http" || !strings.Contains(read(t, dir, "ping.http"), "response.statusCode == 204") {
		t.Errorf("api: %q %q", got.Note, readOrEmpty(dir, "ping.http"))
	}
	s.ok("note.create", map[string]any{"title": "Plain", "options": map[string]any{"text": "# hi"}}, &got)
	if b := read(t, dir, "plain.md"); !strings.Contains(b, "# hi\n") || strings.Contains(b, "type:") {
		t.Errorf("note: %q", b)
	}
}

func TestBridgeRefusesBadOptionsClearly(t *testing.T) {
	dir := board(t, nil)
	s := serve(t, dir)
	for _, c := range []struct {
		params map[string]any
		says   string
	}{
		{map[string]any{"kind": "chart", "title": "x", "options": map[string]any{"view": "pie"}}, "pie"},
		{map[string]any{"kind": "board", "title": "x", "options": map[string]any{"colour": "red"}}, "colour"},
		{map[string]any{"kind": "chat", "title": "x", "options": map[string]any{"x": "1"}}, "no options"},
		{map[string]any{"kind": "form", "title": "x", "options": map[string]any{"choices": map[string]any{}}}, "choices"},
		{map[string]any{"kind": "checklist", "title": "x", "body": "- [ ] a\n", "options": map[string]any{}}, "body or options"},
	} {
		r := s.call("note.create", c.params)
		if r.code != CodeBadRequest || !strings.Contains(r.msg, c.says) {
			t.Errorf("%v: %q %q, want bad_request saying %q", c.params, r.code, r.msg, c.says)
		}
	}
	if notes, _ := filepath.Glob(filepath.Join(dir, "*.md")); len(notes) > 0 {
		t.Errorf("bad options wrote notes: %v", notes)
	}
}
