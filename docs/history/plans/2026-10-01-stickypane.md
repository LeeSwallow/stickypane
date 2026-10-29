# stickypane Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** AI 에이전트가 `.stickypane/` 폴더에 마크다운 파일로 끄적인 노트를 터미널에서 보드로 보여 주고, 화면에서의 수정을 같은 파일에 되돌려 쓰는 TUI `stickypane`을 만든다.

**Architecture:** 노트 하나가 마크다운 파일 하나다. `doc`이 파일을 머리말과 본문으로 나누고, 모양별 위젯(`note`, `board`, `checklist`, `logview`)이 본문을 그리고 수정 의도(`doc.Op`)를 만든다. `store`가 폴더를 읽고 감시하며 의도를 새로 읽은 파일에 적용해 원자적으로 쓴다. `app`(Bubble Tea 모델)이 `layout`으로 노트를 배치하고 키 입력을 위젯과 `store`에 연결한다. `app`의 화면(보드, 모달, 입력창, 카탈로그 등)은 각자 파일에서 `init()`으로 처리기 표에 등록하므로, 기능을 추가할 때 기존 파일을 고치지 않는다.

**Tech Stack:** Go 1.26, Bubble Tea v2 (`charm.land/bubbletea/v2`), Lip Gloss v2, Bubbles v2 (`textinput`), Glamour v2, `github.com/charmbracelet/x/ansi`, fsnotify. 배포는 GoReleaser.

**Spec:** `docs/superpowers/specs/2026-10-01-stickypane-design.md`

## Global Constraints

- 모듈 경로는 `github.com/LeeSwallow/stickypane`, 실행 파일 이름은 `stickypane`이다.
- 의존성은 정확히 다음 여섯 개다. 다른 라이브러리를 추가하지 않는다.
  `charm.land/bubbletea/v2@v2.0.10`, `charm.land/lipgloss/v2@v2.0.6`, `charm.land/bubbles/v2@v2.2.1`, `charm.land/glamour/v2@v2.0.1`, `github.com/charmbracelet/x/ansi@v0.11.8`, `github.com/fsnotify/fsnotify@v1.10.1`
- 노트 폴더 이름은 `.stickypane`이다. 폴더 바로 아래의 `*.md` 파일만 노트다. 떼어 낸 노트는 `.stickypane/archive/`로 옮긴다.
- 설정 파일, 훅, MCP 서버, 터미널 종류별 코드를 만들지 않는다. 표준 입출력만 쓰는 일반 터미널 프로그램이다.
- README, 에이전트 안내문, 화면 문구, 코드 주석은 영어로 쓴다.
- 수치: 노트 최소 폭 36칸, 감시 디바운스 100ms, 파일 크기 상한 1MB, 큰 로그는 마지막 64KB, 파일 이름은 40자(rune)에서 자름.
- 색 이름은 `yellow`, `pink`, `blue`, `green`, `purple`, `orange` 여섯 개다.
- 안내문 표식은 `<!-- stickypane:start -->`와 `<!-- stickypane:end -->`다.
- 지원 플랫폼은 macOS와 Linux다. 라이선스는 MIT다.
- 위젯 패키지는 `doc`과 `widget`만 import한다. Bubble Tea, `store`, `app`, 다른 위젯을 import하지 않는다.
- 파일 내용은 화면에 그릴 때만 다듬는다(`widget.Clean`). 파일에 쓸 때는 바꾸려는 줄 외의 바이트를 보존한다.
- 모든 작업 디렉터리는 `/Users/min/Projects/stickypane`이다. 테스트는 `go test ./...`로 돌린다.

## Review Focus

설계 문서가 직접 말하지 않지만 실제로 쓰다 보면 만날 입력이다. 각 줄 끝의 괄호는 그 경우를 테스트로 고정한 작업이다.

1. **한글·이모지처럼 폭이 2칸인 글자.** 제목과 본문에 섞여도 테두리가 어긋나지 않고 모든 줄의 폭이 같아야 한다. (Task 3 `TestPreviewKeepsWidthWithWideText`, Task 8 `TestFrameKeepsWidthWithWideText`)
2. **CRLF 줄 끝 파일.** 에이전트나 에디터가 `\r\n`으로 저장한 파일을 읽고 고쳐도 줄 끝이 그대로여야 한다. (Task 1 `TestRoundTrip`·`TestSetKeepsCRLF`, Task 3 `TestMoveCardKeepsCRLF`, Task 4 `TestToggleKeepsCRLF`)
3. **아주 작은 터미널.** 폭이나 높이가 0, 1, 5칸이어도 죽지 않고, 그린 줄이 터미널 폭을 넘지 않아야 한다. (Task 8 `TestTinyTerminalDoesNotPanic`)
4. **모달을 연 사이에 노트가 지워지거나 모양이 바뀜.** 에이전트가 파일을 지우거나 `type`을 바꿔도 죽지 않고, 지워졌으면 모달을 닫고 알려야 한다. (Task 9 `TestModalClosesWhenNoteIsRemoved`·`TestModalSurvivesKindChange`)
5. **본문에 섞인 제어 문자와 탭.** 에이전트가 붙여 넣은 색 코드나 탭이 화면을 깨뜨리지 않아야 하고, 파일은 그대로 남아야 한다. (Task 2 `TestCleanDropsControlCharacters`·`TestPreviewDropsControlCharacters`)

---

### Task 1: 프로젝트 뼈대와 `doc` 패키지

노트 파일을 머리말과 본문으로 나누고, 머리말의 키 하나만 줄 단위로 고친다. YAML 전체를 해석하지 않는다. 닫히지 않은 머리말은 머리말이 없는 것으로 본다.

**Files:**
- Create: `go.mod`, `.gitignore`
- Create: `internal/doc/doc.go`
- Test: `internal/doc/doc_test.go`

**Interfaces:**
- Consumes: 없음
- Produces:
  - `doc.Parse(src []byte) doc.Document`
  - `doc.Document{HasFront bool; Front []string; Body string}`
  - `(doc.Document).Bytes() []byte`, `.Get(key string) (string, bool)`, `.Set(key, value string) doc.Document`, `.Type() string`, `.Pinned() bool`
  - `doc.Op` 인터페이스: `Apply(d doc.Document) (doc.Document, error)`
  - `doc.ErrConflict`, `doc.SetKey{Key, Value string}` (Op 구현)
  - `doc.Lines(body string) []string`, `doc.Join(lines []string) string`, `doc.EOL(body string) string`

- [ ] **Step 1: 모듈과 의존성을 준비한다**

```bash
cd /Users/min/Projects/stickypane
go mod init github.com/LeeSwallow/stickypane
go get charm.land/bubbletea/v2@v2.0.10 charm.land/lipgloss/v2@v2.0.6 charm.land/bubbles/v2/textinput@v2.2.1 charm.land/glamour/v2@v2.0.1 github.com/charmbracelet/x/ansi@v0.11.8 github.com/fsnotify/fsnotify@v1.10.1
```

`bubbles`는 모듈 루트가 아니라 실제로 import할 `textinput` 패키지 경로로 받는다. 그래야 그 패키지가 끌어오는 의존성까지 `go.sum`에 기록된다. `go mod tidy`는 Task 12까지 돌리지 않는다. 아직 import하지 않은 의존성이 `go.mod`에서 지워지기 때문이다.

**`.gitignore`**

```gitignore
/stickypane
/dist/
```

- [ ] **Step 2: 실패하는 테스트를 쓴다**

**`internal/doc/doc_test.go`**

```go
package doc

import (
	"errors"
	"reflect"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	cases := []string{
		"",
		"just a line\n",
		"no trailing newline",
		"---\ntype: board\ntitle: Auth\n---\n## To do\n- a\n",
		"---\ntype: log\n---",
		"---\r\ntype: board\r\n---\r\nbody\r\n",
		"---\nunclosed: true\nbody\n",
		"---\n",
		"--- not a fence\ntext\n",
	}
	for _, src := range cases {
		if got := string(Parse([]byte(src)).Bytes()); got != src {
			t.Errorf("round trip changed %q into %q", src, got)
		}
	}
}

func TestParseSplitsFrontAndBody(t *testing.T) {
	d := Parse([]byte("---\ntype: board\ntitle: Auth\n---\n## To do\n"))
	if !d.HasFront {
		t.Fatal("HasFront = false, want true")
	}
	if want := []string{"type: board", "title: Auth"}; !reflect.DeepEqual(d.Front, want) {
		t.Errorf("Front = %q, want %q", d.Front, want)
	}
	if d.Body != "## To do\n" {
		t.Errorf("Body = %q", d.Body)
	}
}

func TestUnclosedFrontIsBody(t *testing.T) {
	src := "---\ntitle: x\nstill body\n"
	d := Parse([]byte(src))
	if d.HasFront || d.Body != src {
		t.Errorf("got HasFront=%v Body=%q, want the whole file as body", d.HasFront, d.Body)
	}
}

func TestGet(t *testing.T) {
	d := Parse([]byte("---\ntype: Board\ntitle:   spaced  \nq1: \"a: b\"\nq2: 'it''s'\n nested: no\nlist:\n  - a\npin: true\r\n---\n"))
	cases := []struct {
		key, want string
		ok        bool
	}{
		{"type", "Board", true},
		{"title", "spaced", true},
		{"q1", "a: b", true},
		{"q2", "it's", true},
		{"nested", "", false},
		{"list", "", true},
		{"pin", "true", true},
		{"missing", "", false},
	}
	for _, c := range cases {
		got, ok := d.Get(c.key)
		if got != c.want || ok != c.ok {
			t.Errorf("Get(%q) = %q, %v; want %q, %v", c.key, got, ok, c.want, c.ok)
		}
	}
	if d.Type() != "board" {
		t.Errorf("Type() = %q, want lower-cased %q", d.Type(), "board")
	}
	if !d.Pinned() {
		t.Error("Pinned() = false, want true")
	}
}

func TestSet(t *testing.T) {
	src := "---\ntype: board\ntitle: Old\nowner: me\n---\nbody\n"
	cases := []struct {
		name, key, value, want string
	}{
		{"replace keeps order", "title", "New", "---\ntype: board\ntitle: New\nowner: me\n---\nbody\n"},
		{"append new key", "pin", "true", "---\ntype: board\ntitle: Old\nowner: me\npin: true\n---\nbody\n"},
		{"quote when needed", "title", "Auth: phase 2", "---\ntype: board\ntitle: \"Auth: phase 2\"\nowner: me\n---\nbody\n"},
	}
	for _, c := range cases {
		d := Parse([]byte(src))
		if got := string(d.Set(c.key, c.value).Bytes()); got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
		if string(d.Bytes()) != src {
			t.Errorf("%s: Set changed the original document", c.name)
		}
	}
}

func TestSetReadsBackQuotedValue(t *testing.T) {
	d := Parse([]byte("hello\n")).Set("title", "Auth: phase 2")
	if got, _ := Parse(d.Bytes()).Get("title"); got != "Auth: phase 2" {
		t.Errorf("Get after Set = %q", got)
	}
}

func TestSetCreatesFront(t *testing.T) {
	got := string(Parse([]byte("hello\n")).Set("pin", "true").Bytes())
	if want := "---\npin: true\n---\nhello\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSetKeepsCRLF(t *testing.T) {
	d := Parse([]byte("---\r\ntitle: Old\r\n---\r\nbody\r\n"))
	got := string(d.Set("title", "New").Set("pin", "true").Bytes())
	if want := "---\r\ntitle: New\r\npin: true\r\n---\r\nbody\r\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSetKeyOp(t *testing.T) {
	var op Op = SetKey{Key: "color", Value: "blue"}
	d, err := op.Apply(Parse([]byte("note\n")))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(d.Bytes()); got != "---\ncolor: blue\n---\nnote\n" {
		t.Errorf("got %q", got)
	}
}

func TestLinesJoinEOL(t *testing.T) {
	body := "a\r\nb\r\n"
	if got := Join(Lines(body)); got != body {
		t.Errorf("Join(Lines()) = %q", got)
	}
	if EOL(body) != "\r" || EOL("a\nb\n") != "" {
		t.Error("EOL should be \"\\r\" for CRLF bodies and empty otherwise")
	}
	if !errors.Is(ErrConflict, ErrConflict) {
		t.Error("ErrConflict must be comparable with errors.Is")
	}
}
```

- [ ] **Step 3: 테스트가 실패하는지 확인한다**

Run: `go test ./internal/doc/`
Expected: 컴파일 실패. `undefined: Parse` 같은 오류가 나온다.

- [ ] **Step 4: 구현한다**

**`internal/doc/doc.go`**

```go
// Package doc splits a note file into front matter and body, and edits front
// matter one line at a time so that every other byte of the file is preserved.
package doc

import (
	"errors"
	"strconv"
	"strings"
)

// ErrConflict reports that an Op could not find its target in the document,
// usually because the file changed after the screen last read it.
var ErrConflict = errors.New("note changed on disk")

// Op is an edit expressed as an intent ("move this card to that column") and
// applied to a freshly read document.
type Op interface {
	Apply(d Document) (Document, error)
}

// SetKey sets one front matter key.
type SetKey struct{ Key, Value string }

// Apply implements Op.
func (o SetKey) Apply(d Document) (Document, error) { return d.Set(o.Key, o.Value), nil }

// Document is a note file split into front matter lines and body.
type Document struct {
	// HasFront reports whether the file starts with a closed "---" block.
	HasFront bool
	// Front holds the lines between the fences, without their "\n".
	Front []string
	// Body is everything after the closing fence, byte for byte. Without
	// front matter it is the whole file.
	Body string

	open, close string // the fence lines as written, without "\n"
	closeNL     bool   // the closing fence ended with "\n"
}

// Parse splits src. A front matter block that is never closed is not front
// matter: the whole file becomes the body, so nothing is hidden.
func Parse(src []byte) Document {
	s := string(src)
	lines := strings.SplitAfter(s, "\n")
	if len(lines) < 2 || !isFence(lines[0]) {
		return Document{Body: s}
	}
	for i := 1; i < len(lines); i++ {
		if !isFence(lines[i]) {
			continue
		}
		d := Document{
			HasFront: true,
			open:     strings.TrimSuffix(lines[0], "\n"),
			close:    strings.TrimSuffix(lines[i], "\n"),
			closeNL:  strings.HasSuffix(lines[i], "\n"),
			Body:     strings.Join(lines[i+1:], ""),
		}
		for _, l := range lines[1:i] {
			d.Front = append(d.Front, strings.TrimSuffix(l, "\n"))
		}
		return d
	}
	return Document{Body: s}
}

func isFence(line string) bool { return strings.TrimRight(line, "\r\n") == "---" }

// Bytes returns the file content.
func (d Document) Bytes() []byte {
	if !d.HasFront {
		return []byte(d.Body)
	}
	var b strings.Builder
	b.WriteString(d.open)
	b.WriteString("\n")
	for _, l := range d.Front {
		b.WriteString(l)
		b.WriteString("\n")
	}
	b.WriteString(d.close)
	if d.closeNL {
		b.WriteString("\n")
	}
	b.WriteString(d.Body)
	return []byte(b.String())
}

// find returns the index of the unindented "key: value" line for key, or -1.
func (d Document) find(key string) int {
	for i, l := range d.Front {
		if strings.HasPrefix(l, " ") || strings.HasPrefix(l, "\t") {
			continue
		}
		if k, _, ok := strings.Cut(l, ":"); ok && strings.TrimSpace(k) == key {
			return i
		}
	}
	return -1
}

// Get returns the scalar value of a top-level front matter key.
func (d Document) Get(key string) (string, bool) {
	i := d.find(key)
	if i < 0 {
		return "", false
	}
	_, v, _ := strings.Cut(d.Front[i], ":")
	return unquote(strings.TrimSpace(v)), true
}

// Set returns a copy with key set to value. Only that key's line changes.
// A document without front matter gets a new block.
func (d Document) Set(key, value string) Document {
	line := key + ": " + quote(value)
	out := d
	if !d.HasFront {
		out.HasFront, out.open, out.close, out.closeNL = true, "---", "---", true
		out.Front = []string{line}
		return out
	}
	out.Front = append([]string(nil), d.Front...)
	if i := d.find(key); i >= 0 {
		if strings.HasSuffix(d.Front[i], "\r") {
			line += "\r"
		}
		out.Front[i] = line
		return out
	}
	if strings.HasSuffix(d.open, "\r") {
		line += "\r"
	}
	out.Front = append(out.Front, line)
	return out
}

// Type returns the lower-cased "type" key, or "" when it is absent.
func (d Document) Type() string {
	v, _ := d.Get("type")
	return strings.ToLower(v)
}

// Pinned reports whether "pin" is true.
func (d Document) Pinned() bool {
	v, _ := d.Get("pin")
	return strings.EqualFold(v, "true")
}

func quote(v string) string {
	if v == "" || v != strings.TrimSpace(v) ||
		strings.ContainsAny(v, ":#\"'\n") ||
		strings.ContainsAny(v[:1], "[]{}>|*&!%@`-") {
		return strconv.Quote(v)
	}
	return v
}

func unquote(v string) string {
	if len(v) < 2 {
		return v
	}
	switch {
	case v[0] == '"' && v[len(v)-1] == '"':
		if s, err := strconv.Unquote(v); err == nil {
			return s
		}
		return v[1 : len(v)-1]
	case v[0] == '\'' && v[len(v)-1] == '\'':
		return strings.ReplaceAll(v[1:len(v)-1], "''", "'")
	}
	return v
}

// Lines splits a body into lines without their "\n". A body that ends with a
// newline yields a final empty element, so Join(Lines(b)) == b.
func Lines(body string) []string { return strings.Split(body, "\n") }

// Join is the inverse of Lines.
func Join(lines []string) string { return strings.Join(lines, "\n") }

// EOL returns the suffix a new line needs to match the body's line endings:
// "\r" for CRLF bodies, "" otherwise.
func EOL(body string) string {
	if strings.Contains(body, "\r\n") {
		return "\r"
	}
	return ""
}
```

- [ ] **Step 5: 테스트가 통과하는지 확인한다**

Run: `go test ./internal/doc/`
Expected: `ok  	github.com/LeeSwallow/stickypane/internal/doc`

- [ ] **Step 6: 커밋한다**

```bash
git add go.mod go.sum .gitignore internal/doc
git commit -m "feat(doc): split notes into front matter and body with line-level edits"
```

---

### Task 2: `widget` 공통부와 일반 노트

위젯 인터페이스, 글자 폭 도우미, 일반 노트 위젯을 만든다. 일반 노트는 마크다운 렌더러를 주입받는다. 테스트에서는 줄바꿈만 하는 `note.Plain`을 쓴다.

**Files:**
- Create: `internal/widget/widget.go`, `internal/widget/text.go`
- Create: `internal/widget/note/note.go`
- Test: `internal/widget/widget_test.go`, `internal/widget/note/note_test.go`

**Interfaces:**
- Consumes: `doc.Document`, `doc.Op` (Task 1)
- Produces:
  - `widget.Widget` 인터페이스: `Preview(width int) string`, `View(width, height int) string`, `Update(key string) (widget.Widget, widget.Result)`, `Sync(d doc.Document) widget.Widget`
  - `widget.Result{Op doc.Op; Prompt *widget.Prompt}`, `widget.Prompt{Label string; Submit func(text string) doc.Op}`
  - `widget.Kind{Name, Label string; FullRow bool; Template func(title string) []byte; Parse func(d doc.Document) widget.Widget}`
  - `widget.Registry` (`[]widget.Kind`), `(widget.Registry).Lookup(name string) widget.Kind` (모르는 이름이면 첫 번째 Kind)
  - `widget.NewFile(kind, title, body string) []byte`
  - `widget.Clean(s string) string`, `widget.Width(s string) int`, `widget.Truncate(s string, w int) string`, `widget.Pad(s string, w int) string`
  - `widget.Window(lines []string, offset, height int) []string`, `widget.ClampOffset(offset, total, height int) int`, `widget.ScrollKey(offset int, key string) (int, bool)`
  - `widget.Bold`, `widget.Faint`, `widget.Selected` (`lipgloss.Style`)
  - `note.Renderer` (`func(markdown string, width int) string`), `note.Plain`, `note.NewKind(render note.Renderer) widget.Kind`

- [ ] **Step 1: 실패하는 테스트를 쓴다**

**`internal/widget/widget_test.go`**

```go
package widget

import (
	"reflect"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

func TestCleanDropsControlCharacters(t *testing.T) {
	got := Clean("a\tb \x1b[31mred\x1b[0m\x07 end\r\nnext")
	if want := "a    b red end\nnext"; got != want {
		t.Errorf("Clean = %q, want %q", got, want)
	}
}

func TestTruncateAndPadCountCells(t *testing.T) {
	if got := Truncate("한글 abcdef", 6); got != "한글 …" {
		t.Errorf("Truncate = %q", got)
	}
	if got := Truncate("anything", 0); got != "" {
		t.Errorf("Truncate to 0 = %q, want empty", got)
	}
	for _, s := range []string{"", "abc", "한글입니다", "📌 pinned note title"} {
		if w := Width(Pad(s, 8)); w != 8 {
			t.Errorf("Width(Pad(%q, 8)) = %d, want 8", s, w)
		}
	}
}

func TestWindow(t *testing.T) {
	lines := []string{"1", "2", "3", "4", "5"}
	cases := []struct {
		offset, height int
		want           []string
	}{
		{0, 2, []string{"1", "2"}},
		{3, 2, []string{"4", "5"}},
		{99, 2, []string{"4", "5"}},
		{-5, 2, []string{"1", "2"}},
		{0, 9, lines},
		{0, 0, nil},
	}
	for _, c := range cases {
		if got := Window(lines, c.offset, c.height); !reflect.DeepEqual(got, c.want) {
			t.Errorf("Window(offset=%d, height=%d) = %q, want %q", c.offset, c.height, got, c.want)
		}
	}
}

func TestScrollKey(t *testing.T) {
	if off, ok := ScrollKey(3, "j"); off != 4 || !ok {
		t.Errorf("j: got %d, %v", off, ok)
	}
	if off, ok := ScrollKey(0, "k"); off != 0 || !ok {
		t.Errorf("k at top: got %d, %v", off, ok)
	}
	if off, _ := ScrollKey(7, "g"); off != 0 {
		t.Errorf("g: got %d", off)
	}
	if _, ok := ScrollKey(0, "n"); ok {
		t.Error("n is not a scroll key")
	}
}

func TestRegistryLookupFallsBackToFirst(t *testing.T) {
	r := Registry{{Name: "note"}, {Name: "board"}}
	if r.Lookup("board").Name != "board" {
		t.Error("Lookup(board) should find board")
	}
	for _, name := range []string{"", "mystery"} {
		if got := r.Lookup(name).Name; got != "note" {
			t.Errorf("Lookup(%q) = %q, want the first kind", name, got)
		}
	}
}

func TestNewFile(t *testing.T) {
	cases := []struct{ kind, title, body, want string }{
		{"note", "", "hello\n", "hello\n"},
		{"note", "Plan", "", "---\ntitle: Plan\n---\n"},
		{"board", "Auth", "## To do\n", "---\ntype: board\ntitle: Auth\n---\n## To do\n"},
	}
	for _, c := range cases {
		if got := string(NewFile(c.kind, c.title, c.body)); got != c.want {
			t.Errorf("NewFile(%q, %q) = %q, want %q", c.kind, c.title, got, c.want)
		}
	}
	if doc.Parse(NewFile("log", "L", "")).Type() != "log" {
		t.Error("NewFile should write the type key")
	}
}
```

**`internal/widget/note/note_test.go`**

```go
package note

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

func parse(src string) widget.Widget {
	return NewKind(Plain).Parse(doc.Parse([]byte(src)))
}

func numbered(n int) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		b.WriteString("l")
		b.WriteString(strings.Repeat("i", i))
		b.WriteString("\n")
	}
	return b.String()
}

func TestPreviewShowsShortNoteAsIs(t *testing.T) {
	if got := parse("check env before deploy\n").Preview(30); got != "check env before deploy" {
		t.Errorf("Preview = %q", got)
	}
}

func TestPreviewHidesFrontMatter(t *testing.T) {
	if got := parse("---\ntitle: T\n---\nhello\n").Preview(30); got != "hello" {
		t.Errorf("Preview = %q", got)
	}
}

func TestPreviewCapsAtSixLines(t *testing.T) {
	lines := strings.Split(ansi.Strip(parse(numbered(10)).Preview(30)), "\n")
	if len(lines) != 6 {
		t.Fatalf("got %d lines, want 6", len(lines))
	}
	if lines[4] != "liiiii" || lines[5] != "…" {
		t.Errorf("last lines = %q, %q", lines[4], lines[5])
	}
}

func TestPreviewOfEmptyNote(t *testing.T) {
	if got := ansi.Strip(parse("").Preview(30)); got != "(empty)" {
		t.Errorf("Preview = %q", got)
	}
}

func TestPreviewDropsControlCharacters(t *testing.T) {
	if got := parse("a\tb\x1b[31m red\x1b[0m\x07\n").Preview(30); got != "a    b red" {
		t.Errorf("Preview = %q", got)
	}
}

func TestPreviewWrapsToWidth(t *testing.T) {
	for _, line := range strings.Split(parse("토큰은 세션 쿠키로 보관한다 and some more words here\n").Preview(12), "\n") {
		if w := widget.Width(line); w > 12 {
			t.Errorf("line %q is %d cells wide, want at most 12", line, w)
		}
	}
}

func TestViewScrolls(t *testing.T) {
	w := parse(numbered(6))
	if got := w.View(20, 2); got != "li\nlii" {
		t.Fatalf("View = %q", got)
	}
	w, _ = w.Update("j")
	if got := w.View(20, 2); got != "lii\nliii" {
		t.Errorf("after j: %q", got)
	}
	w, _ = w.Update("G")
	if got := w.View(20, 2); got != "liiiii\nliiiiii" {
		t.Errorf("after G: %q", got)
	}
	w, _ = w.Update("k")
	if got := w.View(20, 2); got != "liiii\nliiiii" {
		t.Errorf("after G then k: %q", got)
	}
}

func TestSyncKeepsScrollPosition(t *testing.T) {
	w := parse(numbered(6))
	w, _ = w.Update("j")
	w.View(20, 2)
	w = w.Sync(doc.Parse([]byte("a\nb\nc\nd\n")))
	if got := w.View(20, 2); got != "b\nc" {
		t.Errorf("View after Sync = %q", got)
	}
}

func TestKind(t *testing.T) {
	k := NewKind(Plain)
	if k.Name != "note" || k.FullRow {
		t.Errorf("kind = %+v", k)
	}
	if got := string(k.Template("Plan")); got != "---\ntitle: Plan\n---\n" {
		t.Errorf("Template = %q", got)
	}
}
```

- [ ] **Step 2: 테스트가 실패하는지 확인한다**

Run: `go test ./internal/widget/...`
Expected: 컴파일 실패. `undefined: Clean`, `undefined: NewKind` 같은 오류가 나온다.

- [ ] **Step 3: 구현한다**

**`internal/widget/widget.go`**

```go
// Package widget defines how one shape of note is drawn and edited. A widget
// knows nothing about the terminal framework or the file system: it receives
// keys as strings and answers with an intent to apply to the file.
package widget

import "github.com/LeeSwallow/stickypane/internal/doc"

// Widget draws one note and reacts to keys while the note is open.
type Widget interface {
	// Preview draws the note on the board. The kind decides how tall it is.
	Preview(width int) string
	// View draws the open note in exactly the given area or less.
	View(width, height int) string
	// Update handles a key such as "j", "space" or "H".
	Update(key string) (Widget, Result)
	// Sync replaces the content after the file changed and keeps screen
	// state such as the cursor, clamped to the new content.
	Sync(d doc.Document) Widget
}

// Result is what a key press asks the app to do. Both fields may be nil.
type Result struct {
	Op     doc.Op  // apply to the file now
	Prompt *Prompt // collect one line of text first
}

// Prompt asks the app for a line of text and turns it into an Op.
type Prompt struct {
	Label  string
	Submit func(text string) doc.Op
}

// Kind registers one shape of note.
type Kind struct {
	Name     string // the front matter "type" value
	Label    string // shown in the catalog
	FullRow  bool   // takes a whole row on the board
	Template func(title string) []byte
	Parse    func(d doc.Document) Widget
}

// Registry lists the known kinds. The first one is the fallback and must exist.
type Registry []Kind

// Lookup returns the kind for a type value. Unknown values get the first kind.
func (r Registry) Lookup(name string) Kind {
	for _, k := range r {
		if k.Name == name {
			return k
		}
	}
	return r[0]
}

// NewFile builds the content of a new note file.
func NewFile(kind, title, body string) []byte {
	d := doc.Document{Body: body}
	if kind != "" && kind != "note" {
		d = d.Set("type", kind)
	}
	if title != "" {
		d = d.Set("title", title)
	}
	return d.Bytes()
}
```

**`internal/widget/text.go`**

```go
package widget

import (
	"strings"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Shared text styles.
var (
	Bold     = lipgloss.NewStyle().Bold(true)
	Faint    = lipgloss.NewStyle().Faint(true)
	Selected = lipgloss.NewStyle().Reverse(true)
)

// Clean prepares file text for the screen: escape sequences and control
// characters are dropped and tabs become four spaces. Newlines are kept.
func Clean(s string) string {
	s = strings.ReplaceAll(ansi.Strip(s), "\t", "    ")
	return strings.Map(func(r rune) rune {
		if r != '\n' && unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}

// Width returns how many terminal cells s occupies.
func Width(s string) int { return ansi.StringWidth(s) }

// Truncate shortens s to at most w cells, ending with "…" when it was cut.
func Truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	return ansi.Truncate(s, w, "…")
}

// Pad truncates or right-pads s to exactly w cells.
func Pad(s string, w int) string {
	if w <= 0 {
		return ""
	}
	s = Truncate(s, w)
	if gap := w - Width(s); gap > 0 {
		s += strings.Repeat(" ", gap)
	}
	return s
}

// ClampOffset keeps a scroll offset inside [0, total-height].
func ClampOffset(offset, total, height int) int {
	if m := total - height; offset > m {
		offset = m
	}
	if offset < 0 {
		offset = 0
	}
	return offset
}

// Window returns the lines visible at offset in an area of the given height.
func Window(lines []string, offset, height int) []string {
	if height <= 0 {
		return nil
	}
	offset = ClampOffset(offset, len(lines), height)
	return lines[offset:min(offset+height, len(lines))]
}

// ScrollKey returns the offset after a scrolling key and whether key was one.
// The result may exceed the content; ClampOffset fixes that when drawing.
func ScrollKey(offset int, key string) (int, bool) {
	switch key {
	case "j", "down":
		offset++
	case "k", "up":
		offset--
	case "space", "pgdown":
		offset += 10
	case "b", "pgup":
		offset -= 10
	case "g":
		offset = 0
	case "G":
		offset = 1 << 30
	default:
		return offset, false
	}
	return max(offset, 0), true
}
```

**`internal/widget/note/note.go`**

```go
// Package note is the plain note: any Markdown, from one line to a report.
package note

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// previewLines is the tallest a plain note gets on the board.
const previewLines = 6

// Renderer turns Markdown into terminal text no wider than width.
type Renderer func(markdown string, width int) string

// Plain wraps text without styling it.
func Plain(markdown string, width int) string {
	if width <= 0 {
		return markdown
	}
	return ansi.Wrap(markdown, width, "")
}

// NewKind returns the plain note kind drawing with the given renderer.
func NewKind(render Renderer) widget.Kind {
	return widget.Kind{
		Name:     "note",
		Label:    "Note",
		Template: func(title string) []byte { return widget.NewFile("note", title, "") },
		Parse:    func(d doc.Document) widget.Widget { return newNote(d.Body, render, 0) },
	}
}

// Note is the widget for a plain note.
type Note struct {
	body   string
	render Renderer
	cache  map[int][]string // rendered lines by width
	offset int
}

func newNote(body string, render Renderer, offset int) *Note {
	return &Note{body: body, render: render, cache: map[int][]string{}, offset: offset}
}

// lines renders the body at width and trims blank lines from both ends.
func (n *Note) lines(width int) []string {
	if l, ok := n.cache[width]; ok {
		return l
	}
	l := strings.Split(n.render(widget.Clean(n.body), width), "\n")
	blank := func(s string) bool { return strings.TrimSpace(ansi.Strip(s)) == "" }
	for len(l) > 0 && blank(l[0]) {
		l = l[1:]
	}
	for len(l) > 0 && blank(l[len(l)-1]) {
		l = l[:len(l)-1]
	}
	n.cache[width] = l
	return l
}

// Preview implements widget.Widget.
func (n *Note) Preview(width int) string {
	l := n.lines(width)
	if len(l) == 0 {
		return widget.Faint.Render("(empty)")
	}
	if len(l) > previewLines {
		l = append(append([]string(nil), l[:previewLines-1]...), widget.Faint.Render("…"))
	}
	return strings.Join(l, "\n")
}

// View implements widget.Widget.
func (n *Note) View(width, height int) string {
	l := n.lines(width)
	n.offset = widget.ClampOffset(n.offset, len(l), height)
	return strings.Join(widget.Window(l, n.offset, height), "\n")
}

// Update implements widget.Widget. A plain note only scrolls.
func (n *Note) Update(key string) (widget.Widget, widget.Result) {
	n.offset, _ = widget.ScrollKey(n.offset, key)
	return n, widget.Result{}
}

// Sync implements widget.Widget.
func (n *Note) Sync(d doc.Document) widget.Widget {
	return newNote(d.Body, n.render, n.offset)
}
```

- [ ] **Step 4: 테스트가 통과하는지 확인한다**

Run: `go test ./internal/widget/...`
Expected: `internal/widget`와 `internal/widget/note` 둘 다 `ok`

- [ ] **Step 5: 커밋한다**

```bash
git add internal/widget
git commit -m "feat(widget): add widget contract, text helpers and the plain note"
```

---

### Task 3: `board` 위젯 (칸반)

`## ` 제목이 칸이고 그 아래 최상위 목록 항목이 카드다. 항목 아래 들여쓴 줄은 그 카드의 상세 내용이다. 카드 이동·순서 변경·추가는 본문의 줄을 옮기는 `doc.Op`로 표현한다. 카드는 "칸 제목 + 카드 글"로 찾고, 찾지 못하면 `doc.ErrConflict`를 돌려준다.

**Files:**
- Create: `internal/widget/board/board.go`, `internal/widget/board/ops.go`
- Test: `internal/widget/board/board_test.go`

**Interfaces:**
- Consumes: `doc.Document`, `doc.Op`, `doc.ErrConflict`, `doc.Lines`, `doc.Join`, `doc.EOL` (Task 1), `widget.Widget`, `widget.Result`, `widget.Prompt`, `widget.Kind`, `widget.NewFile`, `widget.Clean`, `widget.Truncate`, `widget.Pad`, `widget.Bold`, `widget.Faint`, `widget.Selected` (Task 2)
- Produces:
  - `board.Kind` (`widget.Kind`, `Name: "board"`, `FullRow: true`)
  - `board.MoveCard{From, To, Text string}`, `board.ReorderCard{Col, Text string; Delta int}`, `board.AddCard{Col, Text string}` (모두 `doc.Op`)
  - 모달 키: `h` `l` `left` `right` 칸 이동, `j` `k` `down` `up` 카드 이동, `H` `L` 카드를 옆 칸으로, `J` `K` 순서 변경, `n` 카드 추가(`Prompt`)

- [ ] **Step 1: 실패하는 테스트를 쓴다**

**`internal/widget/board/board_test.go`**

```go
package board

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

const src = "---\ntype: board\ntitle: Auth\n---\n" + body

const body = "## To do\n- payments\n## Doing\n- login API\n  - refresh token later\n## Done\n- schema\n"

func parseSrc(s string) widget.Widget { return Kind.Parse(doc.Parse([]byte(s))) }

func apply(t *testing.T, op doc.Op, body string) string {
	t.Helper()
	d, err := op.Apply(doc.Document{Body: body})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	return d.Body
}

func TestPreviewShowsColumnsSideBySide(t *testing.T) {
	got := ansi.Strip(parseSrc(src).Preview(60))
	want := "To do (1)           Doing (1)           Done (1)\n" +
		"payments            login API           schema"
	if got != want {
		t.Errorf("Preview:\n%s\nwant:\n%s", got, want)
	}
}

func TestPreviewStacksColumnsWhenNarrow(t *testing.T) {
	got := ansi.Strip(parseSrc(src).Preview(30))
	want := "To do (1)\n  payments\nDoing (1)\n  login API\nDone (1)\n  schema"
	if got != want {
		t.Errorf("Preview:\n%s\nwant:\n%s", got, want)
	}
}

func TestPreviewLimitsCardsPerColumn(t *testing.T) {
	got := ansi.Strip(parseSrc("## A\n- 1\n- 2\n- 3\n- 4\n- 5\n- 6\n- 7\n").Preview(40))
	if want := "A (7)\n1\n2\n3\n4\n5\n+2 more"; got != want {
		t.Errorf("Preview = %q, want %q", got, want)
	}
}

func TestPreviewWithoutColumnsExplainsFormat(t *testing.T) {
	got := ansi.Strip(parseSrc("just text\n").Preview(60))
	if !strings.Contains(got, `"## Name"`) {
		t.Errorf("Preview = %q, want a hint about headings", got)
	}
}

func TestPreviewKeepsWidthWithWideText(t *testing.T) {
	w := parseSrc("## 할 일\n- 결제 연동을 다음 주까지 마무리하기\n## 진행 중\n- 로그인 API 📌\n## 완료\n- DB 스키마\n")
	for _, width := range []int{20, 34, 60} {
		for _, line := range strings.Split(w.Preview(width), "\n") {
			if got := widget.Width(line); got > width {
				t.Errorf("width %d: line %q is %d cells wide", width, ansi.Strip(line), got)
			}
		}
	}
}

func TestMoveCard(t *testing.T) {
	got := apply(t, MoveCard{From: "Doing", To: "Done", Text: "login API"}, body)
	want := "## To do\n- payments\n## Doing\n## Done\n- schema\n- login API\n  - refresh token later\n"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestMoveCardIntoEmptyColumn(t *testing.T) {
	got := apply(t, MoveCard{From: "A", To: "B", Text: "x"}, "## A\n- x\n\n## B\n\n## C\n")
	if want := "## A\n\n## B\n- x\n\n## C\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMoveCardKeepsCRLF(t *testing.T) {
	got := apply(t, MoveCard{From: "A", To: "B", Text: "x"}, "## A\r\n- x\r\n## B\r\n")
	if want := "## A\r\n## B\r\n- x\r\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMoveCardConflicts(t *testing.T) {
	for _, op := range []doc.Op{
		MoveCard{From: "Doing", To: "Done", Text: "gone"},
		MoveCard{From: "Nowhere", To: "Done", Text: "login API"},
		MoveCard{From: "Doing", To: "Nowhere", Text: "login API"},
		ReorderCard{Col: "Doing", Text: "gone", Delta: 1},
		AddCard{Col: "Nowhere", Text: "x"},
	} {
		d, err := op.Apply(doc.Document{Body: body})
		if !errors.Is(err, doc.ErrConflict) {
			t.Errorf("%+v: err = %v, want ErrConflict", op, err)
		}
		if d.Body != body {
			t.Errorf("%+v: body changed on conflict", op)
		}
	}
}

func TestReorderCard(t *testing.T) {
	list := "## A\n- one\n  - detail\n- two\n- three\n"
	cases := []struct {
		text  string
		delta int
		want  string
	}{
		{"one", 1, "## A\n- two\n- one\n  - detail\n- three\n"},
		{"three", -1, "## A\n- one\n  - detail\n- three\n- two\n"},
		{"one", -1, list},
		{"three", 1, list},
	}
	for _, c := range cases {
		if got := apply(t, ReorderCard{Col: "A", Text: c.text, Delta: c.delta}, list); got != c.want {
			t.Errorf("reorder %q by %d: got %q, want %q", c.text, c.delta, got, c.want)
		}
	}
}

func TestAddCard(t *testing.T) {
	cases := []struct{ body, col, want string }{
		{"## A\n\n## B\n", "A", "## A\n- new\n\n## B\n"},
		{"## A\n- one\n  - detail\n## B\n", "A", "## A\n- one\n  - detail\n- new\n## B\n"},
		{"## A", "A", "## A\n- new"},
		{"## A\r\n- one\r\n", "A", "## A\r\n- one\r\n- new\r\n"},
	}
	for _, c := range cases {
		if got := apply(t, AddCard{Col: c.col, Text: "new"}, c.body); got != c.want {
			t.Errorf("add to %q: got %q, want %q", c.body, got, c.want)
		}
	}
}

func TestMoveKeyEmitsOpAndFollowsCard(t *testing.T) {
	w := parseSrc(src)
	w, res := w.Update("l")
	if res.Op != nil {
		t.Fatalf("l should only move the cursor, got %+v", res.Op)
	}
	w, res = w.Update("L")
	want := MoveCard{From: "Doing", To: "Done", Text: "login API"}
	if res.Op != want {
		t.Fatalf("Op = %+v, want %+v", res.Op, want)
	}
	d, err := res.Op.Apply(doc.Parse([]byte(src)))
	if err != nil {
		t.Fatal(err)
	}
	view := ansi.Strip(w.Sync(d).View(60, 10))
	if !strings.Contains(view, "› login API") {
		t.Errorf("cursor should follow the moved card:\n%s", view)
	}
}

func TestMoveKeyAtEdgeDoesNothing(t *testing.T) {
	_, res := parseSrc(src).Update("H")
	if res.Op != nil {
		t.Errorf("H in the first column should do nothing, got %+v", res.Op)
	}
}

func TestReorderKeys(t *testing.T) {
	w := parseSrc("## A\n- one\n- two\n")
	_, res := w.Update("J")
	if want := (ReorderCard{Col: "A", Text: "one", Delta: 1}); res.Op != want {
		t.Errorf("J: Op = %+v, want %+v", res.Op, want)
	}
	_, res = parseSrc("## A\n- one\n- two\n").Update("K")
	if res.Op != nil {
		t.Errorf("K on the first card should do nothing, got %+v", res.Op)
	}
}

func TestNewCardPrompt(t *testing.T) {
	_, res := parseSrc(src).Update("n")
	if res.Prompt == nil {
		t.Fatal("n should ask for the card text")
	}
	if res.Prompt.Label != "New card in To do" {
		t.Errorf("Label = %q", res.Prompt.Label)
	}
	if got, want := res.Prompt.Submit("write tests"), (AddCard{Col: "To do", Text: "write tests"}); got != want {
		t.Errorf("Submit = %+v, want %+v", got, want)
	}
}

func TestViewShowsDetailOfSelectedCard(t *testing.T) {
	w := parseSrc(src)
	w, _ = w.Update("l")
	view := ansi.Strip(w.View(60, 10))
	for _, want := range []string{"› login API", "refresh token later", "  payments"} {
		if !strings.Contains(view, want) {
			t.Errorf("View should contain %q:\n%s", want, view)
		}
	}
	if n := strings.Count(view, "\n") + 1; n > 10 {
		t.Errorf("View is %d lines tall, want at most 10", n)
	}
}

func TestViewScrollsColumnsWhenNarrow(t *testing.T) {
	w := parseSrc(src)
	w, _ = w.Update("l")
	w, _ = w.Update("l")
	view := ansi.Strip(w.View(20, 6))
	if !strings.Contains(view, "› schema") {
		t.Errorf("the selected column must stay visible:\n%s", view)
	}
	for _, line := range strings.Split(view, "\n") {
		if widget.Width(line) > 20 {
			t.Errorf("line %q is wider than 20 cells", line)
		}
	}
}

func TestEmptyBoardIgnoresKeys(t *testing.T) {
	w := parseSrc("no columns here\n")
	for _, key := range []string{"h", "l", "j", "k", "H", "L", "J", "K", "n"} {
		var res widget.Result
		w, res = w.Update(key)
		if res.Op != nil || res.Prompt != nil {
			t.Errorf("%s on an empty board returned %+v", key, res)
		}
	}
	w.View(40, 5)
}

func TestSyncClampsCursor(t *testing.T) {
	w := parseSrc(src)
	w, _ = w.Update("l")
	w, _ = w.Update("l")
	w = w.Sync(doc.Parse([]byte("## Only\n- card\n")))
	if view := ansi.Strip(w.View(40, 6)); !strings.Contains(view, "› card") {
		t.Errorf("cursor should clamp to the remaining column:\n%s", view)
	}
}

func TestKindTemplate(t *testing.T) {
	got := string(Kind.Template("Auth"))
	want := "---\ntype: board\ntitle: Auth\n---\n## To do\n\n## Doing\n\n## Done\n"
	if got != want {
		t.Errorf("Template = %q", got)
	}
	if !Kind.FullRow || Kind.Name != "board" {
		t.Errorf("Kind = %+v", Kind)
	}
}
```

- [ ] **Step 2: 테스트가 실패하는지 확인한다**

Run: `go test ./internal/widget/board/`
Expected: 컴파일 실패. `undefined: Kind`, `undefined: MoveCard` 같은 오류가 나온다.

- [ ] **Step 3: 본문 해석과 Op를 구현한다**

**`internal/widget/board/ops.go`**

```go
package board

import (
	"strings"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

// cardSpan is a card's place in the body: its first line and the indented
// detail lines right below it, as the half-open line range [start, end).
type cardSpan struct {
	text       string
	start, end int
}

// colSpan is a column: its "## " heading line and the cards below it.
type colSpan struct {
	title string
	head  int
	cards []cardSpan
}

func isCard(line string) bool {
	return strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ")
}

func isDetail(line string) bool {
	return (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) && strings.TrimSpace(line) != ""
}

// scan finds the columns and cards in body lines. Lines before the first
// heading and anything that is not a card are left alone.
func scan(lines []string) []colSpan {
	var cols []colSpan
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(line, "## ") {
			cols = append(cols, colSpan{title: strings.TrimSpace(line[3:]), head: i})
			continue
		}
		if len(cols) == 0 || !isCard(line) {
			continue
		}
		c := cardSpan{text: strings.TrimSpace(line[2:]), start: i, end: i + 1}
		for c.end < len(lines) && isDetail(lines[c.end]) {
			c.end++
		}
		last := &cols[len(cols)-1]
		last.cards = append(last.cards, c)
		i = c.end - 1
	}
	return cols
}

func findCol(cols []colSpan, title string) int {
	for i, c := range cols {
		if c.title == title {
			return i
		}
	}
	return -1
}

func findCard(col colSpan, text string) int {
	for i, c := range col.cards {
		if c.text == text {
			return i
		}
	}
	return -1
}

// insertAt is the line index where a new card goes: after the column's last
// card, or right below the heading when the column is empty.
func insertAt(col colSpan) int {
	if n := len(col.cards); n > 0 {
		return col.cards[n-1].end
	}
	return col.head + 1
}

func splice(lines []string, at int, block ...string) []string {
	out := make([]string, 0, len(lines)+len(block))
	out = append(out, lines[:at]...)
	out = append(out, block...)
	return append(out, lines[at:]...)
}

// MoveCard moves the card with Text from column From to the end of column To.
type MoveCard struct{ From, To, Text string }

// Apply implements doc.Op.
func (o MoveCard) Apply(d doc.Document) (doc.Document, error) {
	lines := doc.Lines(d.Body)
	cols := scan(lines)
	from := findCol(cols, o.From)
	if from < 0 || findCol(cols, o.To) < 0 {
		return d, doc.ErrConflict
	}
	i := findCard(cols[from], o.Text)
	if i < 0 {
		return d, doc.ErrConflict
	}
	c := cols[from].cards[i]
	block := append([]string(nil), lines[c.start:c.end]...)
	rest := append(append([]string(nil), lines[:c.start]...), lines[c.end:]...)
	cols = scan(rest)
	d.Body = doc.Join(splice(rest, insertAt(cols[findCol(cols, o.To)]), block...))
	return d, nil
}

// ReorderCard moves the card with Text up (Delta -1) or down (Delta 1)
// inside column Col. At the edge of the column it changes nothing.
type ReorderCard struct {
	Col, Text string
	Delta     int
}

// Apply implements doc.Op.
func (o ReorderCard) Apply(d doc.Document) (doc.Document, error) {
	lines := doc.Lines(d.Body)
	cols := scan(lines)
	ci := findCol(cols, o.Col)
	if ci < 0 {
		return d, doc.ErrConflict
	}
	cards := cols[ci].cards
	i := findCard(cols[ci], o.Text)
	if i < 0 {
		return d, doc.ErrConflict
	}
	j := i + o.Delta
	if j < 0 || j >= len(cards) {
		return d, nil
	}
	lo, hi := cards[min(i, j)], cards[max(i, j)]
	out := make([]string, 0, len(lines))
	out = append(out, lines[:lo.start]...)
	out = append(out, lines[hi.start:hi.end]...)
	out = append(out, lines[lo.end:hi.start]...)
	out = append(out, lines[lo.start:lo.end]...)
	out = append(out, lines[hi.end:]...)
	d.Body = doc.Join(out)
	return d, nil
}

// AddCard appends a card to the end of column Col.
type AddCard struct{ Col, Text string }

// Apply implements doc.Op.
func (o AddCard) Apply(d doc.Document) (doc.Document, error) {
	lines := doc.Lines(d.Body)
	cols := scan(lines)
	ci := findCol(cols, o.Col)
	if ci < 0 {
		return d, doc.ErrConflict
	}
	d.Body = doc.Join(splice(lines, insertAt(cols[ci]), "- "+o.Text+doc.EOL(d.Body)))
	return d, nil
}
```

- [ ] **Step 4: 위젯을 구현한다**

**`internal/widget/board/board.go`**

```go
// Package board is the kanban note: "## " headings are columns and the
// top-level list items below them are cards.
package board

import (
	"fmt"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

const (
	previewCards   = 5  // cards shown per column on the board
	stackedCards   = 3  // cards shown per column when columns are stacked
	minPreviewCol  = 12 // narrower than this, the preview stacks columns
	minModalCol    = 16 // narrower than this, the open board scrolls sideways
	detailRows     = 4  // separator plus three lines about the selected card
	minDetailModal = 8  // shorter than this, the open board hides the detail
	formatHint     = `no columns yet: add "## Name" headings`
)

// Kind registers the board.
var Kind = widget.Kind{
	Name:    "board",
	Label:   "Board (kanban)",
	FullRow: true,
	Template: func(title string) []byte {
		return widget.NewFile("board", title, "## To do\n\n## Doing\n\n## Done\n")
	},
	Parse: func(d doc.Document) widget.Widget { return parse(d) },
}

type card struct {
	text   string
	detail []string
}

type column struct {
	title string
	cards []card
}

// Board is the widget for a kanban note.
type Board struct {
	cols     []column
	col, row int // cursor
}

func parse(d doc.Document) *Board {
	lines := doc.Lines(d.Body)
	b := &Board{}
	for _, s := range scan(lines) {
		c := column{title: s.title}
		for _, cs := range s.cards {
			cd := card{text: cs.text}
			for _, l := range lines[cs.start+1 : cs.end] {
				cd.detail = append(cd.detail, strings.TrimSpace(l))
			}
			c.cards = append(c.cards, cd)
		}
		b.cols = append(b.cols, c)
	}
	return b
}

func (b *Board) clamp() {
	b.col = max(min(b.col, len(b.cols)-1), 0)
	n := 0
	if len(b.cols) > 0 {
		n = len(b.cols[b.col].cards)
	}
	b.row = max(min(b.row, n-1), 0)
}

func (b *Board) current() (card, bool) {
	if b.col < len(b.cols) && b.row < len(b.cols[b.col].cards) {
		return b.cols[b.col].cards[b.row], true
	}
	return card{}, false
}

func header(c column) string {
	return fmt.Sprintf("%s (%d)", widget.Clean(c.title), len(c.cards))
}

// Preview implements widget.Widget.
func (b *Board) Preview(width int) string {
	if len(b.cols) == 0 {
		return widget.Faint.Render(widget.Truncate(formatHint, width))
	}
	cw := width / len(b.cols)
	if cw < minPreviewCol {
		return b.stacked(width)
	}
	cells := make([][]string, len(b.cols))
	rows := 0
	for i, c := range b.cols {
		cells[i] = []string{widget.Bold.Render(widget.Truncate(header(c), cw-1))}
		for j, cd := range c.cards {
			if j == previewCards {
				cells[i] = append(cells[i], widget.Faint.Render(fmt.Sprintf("+%d more", len(c.cards)-j)))
				break
			}
			cells[i] = append(cells[i], widget.Truncate(widget.Clean(cd.text), cw-1))
		}
		rows = max(rows, len(cells[i]))
	}
	return joinColumns(cells, rows, cw)
}

func (b *Board) stacked(width int) string {
	var out []string
	for _, c := range b.cols {
		out = append(out, widget.Bold.Render(widget.Truncate(header(c), width)))
		for j, cd := range c.cards {
			if j == stackedCards {
				out = append(out, widget.Faint.Render(fmt.Sprintf("  +%d more", len(c.cards)-j)))
				break
			}
			out = append(out, "  "+widget.Truncate(widget.Clean(cd.text), width-2))
		}
	}
	return strings.Join(out, "\n")
}

// joinColumns lays cells out side by side, each column cw cells wide.
func joinColumns(cells [][]string, rows, cw int) string {
	out := make([]string, rows)
	for r := range out {
		var sb strings.Builder
		for _, col := range cells {
			s := ""
			if r < len(col) {
				s = col[r]
			}
			sb.WriteString(widget.Pad(s, cw))
		}
		out[r] = strings.TrimRight(sb.String(), " ")
	}
	return strings.Join(out, "\n")
}

// View implements widget.Widget.
func (b *Board) View(width, height int) string {
	if len(b.cols) == 0 {
		return widget.Faint.Render(widget.Truncate(formatHint, width))
	}
	b.clamp()
	detail := 0
	if height >= minDetailModal {
		detail = detailRows
	}
	listH := max(height-detail-1, 1)

	cw, first, visible := width/len(b.cols), 0, len(b.cols)
	if cw < minModalCol {
		cw = max(min(minModalCol, width), 1)
		visible = max(width/cw, 1)
		first = max(b.col-visible+1, 0)
	}

	var cells [][]string
	for i := first; i < first+visible && i < len(b.cols); i++ {
		c := b.cols[i]
		head := widget.Bold.Render(widget.Truncate(header(c), cw-1))
		if i == b.col {
			head = widget.Bold.Underline(true).Render(widget.Truncate(header(c), cw-1))
		}
		lines := []string{head}
		offset := 0
		if i == b.col {
			offset = max(b.row-listH+1, 0)
		}
		for r := offset; r < len(c.cards) && r < offset+listH; r++ {
			text := widget.Truncate(widget.Clean(c.cards[r].text), cw-3)
			if i == b.col && r == b.row {
				lines = append(lines, widget.Selected.Render("› "+text))
			} else {
				lines = append(lines, "  "+text)
			}
		}
		cells = append(cells, lines)
	}
	out := []string{joinColumns(cells, listH+1, cw)}
	if detail > 0 {
		out = append(out, widget.Faint.Render(strings.Repeat("─", width)))
		out = append(out, b.detail(width, detail-1)...)
	}
	return strings.Join(out, "\n")
}

// detail describes the selected card in at most n lines.
func (b *Board) detail(width, n int) []string {
	c, ok := b.current()
	if !ok {
		return []string{widget.Faint.Render("no card selected")}
	}
	lines := []string{widget.Bold.Render(widget.Truncate(widget.Clean(c.text), width))}
	for _, d := range c.detail {
		lines = append(lines, widget.Truncate(widget.Clean(d), width))
	}
	if len(c.detail) == 0 {
		lines = append(lines, widget.Faint.Render("(no details)"))
	}
	return lines[:min(len(lines), n)]
}

// Update implements widget.Widget.
func (b *Board) Update(key string) (widget.Widget, widget.Result) {
	var res widget.Result
	if len(b.cols) == 0 {
		return b, res
	}
	b.clamp()
	col := b.cols[b.col]
	switch key {
	case "h", "left":
		b.col--
	case "l", "right":
		b.col++
	case "j", "down":
		b.row++
	case "k", "up":
		b.row--
	case "H", "L":
		to := b.col - 1
		if key == "L" {
			to = b.col + 1
		}
		if c, ok := b.current(); ok && to >= 0 && to < len(b.cols) {
			res.Op = MoveCard{From: col.title, To: b.cols[to].title, Text: c.text}
			// The card lands at the end of the target column. Point the
			// cursor there now; the next Sync makes that position real.
			b.col, b.row = to, len(b.cols[to].cards)
			return b, res
		}
	case "J", "K":
		delta := 1
		if key == "K" {
			delta = -1
		}
		if c, ok := b.current(); ok && b.row+delta >= 0 && b.row+delta < len(col.cards) {
			res.Op = ReorderCard{Col: col.title, Text: c.text, Delta: delta}
			b.row += delta
		}
	case "n":
		title := col.title
		res.Prompt = &widget.Prompt{
			Label:  "New card in " + widget.Clean(title),
			Submit: func(text string) doc.Op { return AddCard{Col: title, Text: text} },
		}
	}
	b.clamp()
	return b, res
}

// Sync implements widget.Widget.
func (b *Board) Sync(d doc.Document) widget.Widget {
	nb := parse(d)
	nb.col, nb.row = b.col, b.row
	nb.clamp()
	return nb
}
```

- [ ] **Step 5: 테스트가 통과하는지 확인한다**

Run: `go test ./internal/widget/board/`
Expected: `ok  	github.com/LeeSwallow/stickypane/internal/widget/board`

- [ ] **Step 6: 커밋한다**

```bash
git add internal/widget/board
git commit -m "feat(board): add the kanban note with move, reorder and add-card intents"
```

---

### Task 4: `checklist` 위젯

`- [ ]`와 `- [x]` 줄이 항목이다. 체크박스가 아닌 줄은 그대로 보여 준다. 체크 전환은 "이 글을 가진, 아직 반대 상태인 항목"을 찾아 대괄호 안 글자만 바꾼다.

**Files:**
- Create: `internal/widget/checklist/checklist.go`
- Test: `internal/widget/checklist/checklist_test.go`

**Interfaces:**
- Consumes: Task 1의 `doc`, Task 2의 `widget`
- Produces:
  - `checklist.Kind` (`widget.Kind`, `Name: "checklist"`)
  - `checklist.Toggle{Text string; Checked bool}`, `checklist.AddItem{Text string}` (`doc.Op`)
  - 모달 키: `j` `k` `down` `up` 이동, `space` `x` 체크 전환, `n` 항목 추가(`Prompt`)

- [ ] **Step 1: 실패하는 테스트를 쓴다**

**`internal/widget/checklist/checklist_test.go`**

```go
package checklist

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

const body = "## Login API\n- [x] Add endpoint\n- [x] Validate input\n- [ ] Write tests\n"

func parseBody(s string) widget.Widget { return Kind.Parse(doc.Document{Body: s}) }

func apply(t *testing.T, op doc.Op, body string) string {
	t.Helper()
	d, err := op.Apply(doc.Document{Body: body})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	return d.Body
}

func TestPreviewShowsProgressAndOpenItems(t *testing.T) {
	got := ansi.Strip(parseBody(body).Preview(30))
	want := "▓▓▓▓▓▓▓▓▓▓▓▓▓░░░░░░░ 2/3\n☐ Write tests"
	if got != want {
		t.Errorf("Preview = %q, want %q", got, want)
	}
}

func TestPreviewWhenAllDone(t *testing.T) {
	got := ansi.Strip(parseBody("- [x] a\n- [X] b\n").Preview(30))
	if !strings.HasSuffix(got, " 2/2\n✓ all done") {
		t.Errorf("Preview = %q", got)
	}
}

func TestPreviewLimitsOpenItems(t *testing.T) {
	got := ansi.Strip(parseBody("- [ ] 1\n- [ ] 2\n- [ ] 3\n- [ ] 4\n- [ ] 5\n- [ ] 6\n").Preview(30))
	if !strings.HasSuffix(got, "☐ 4\n+2 more") {
		t.Errorf("Preview = %q", got)
	}
}

func TestPreviewWithoutItemsExplainsFormat(t *testing.T) {
	if got := ansi.Strip(parseBody("nothing here\n").Preview(40)); !strings.Contains(got, `"- [ ] task"`) {
		t.Errorf("Preview = %q", got)
	}
}

func TestPreviewFitsNarrowWidth(t *testing.T) {
	for _, line := range strings.Split(parseBody("- [ ] 아주 긴 한글 항목 이름입니다 정말로\n").Preview(12), "\n") {
		if w := widget.Width(line); w > 12 {
			t.Errorf("line %q is %d cells wide", ansi.Strip(line), w)
		}
	}
}

func TestToggle(t *testing.T) {
	got := apply(t, Toggle{Text: "Write tests", Checked: true}, body)
	if want := strings.Replace(body, "- [ ] Write tests", "- [x] Write tests", 1); got != want {
		t.Errorf("got %q", got)
	}
	got = apply(t, Toggle{Text: "Add endpoint", Checked: false}, body)
	if want := strings.Replace(body, "- [x] Add endpoint", "- [ ] Add endpoint", 1); got != want {
		t.Errorf("got %q", got)
	}
}

func TestToggleKeepsCRLF(t *testing.T) {
	got := apply(t, Toggle{Text: "a", Checked: true}, "  * [ ] a\r\n- [ ] b\r\n")
	if want := "  * [x] a\r\n- [ ] b\r\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestToggleConflicts(t *testing.T) {
	for _, op := range []doc.Op{
		Toggle{Text: "gone", Checked: true},
		Toggle{Text: "Add endpoint", Checked: true}, // already checked
	} {
		d, err := op.Apply(doc.Document{Body: body})
		if !errors.Is(err, doc.ErrConflict) || d.Body != body {
			t.Errorf("%+v: err = %v, body changed = %v", op, err, d.Body != body)
		}
	}
}

func TestAddItem(t *testing.T) {
	cases := []struct{ body, want string }{
		{"- [x] a\n- [ ] b\n\nnotes\n", "- [x] a\n- [ ] b\n- [ ] new\n\nnotes\n"},
		{"", "- [ ] new\n"},
		{"intro\n", "intro\n- [ ] new\n"},
		{"intro", "intro\n- [ ] new"},
		{"- [ ] a", "- [ ] a\n- [ ] new"},
		{"- [ ] a\r\n", "- [ ] a\r\n- [ ] new\r\n"},
	}
	for _, c := range cases {
		if got := apply(t, AddItem{Text: "new"}, c.body); got != c.want {
			t.Errorf("add to %q: got %q, want %q", c.body, got, c.want)
		}
	}
}

func TestSpaceTogglesItemUnderCursor(t *testing.T) {
	w := parseBody(body)
	w, _ = w.Update("j")
	w, _ = w.Update("j")
	w, res := w.Update("space")
	if want := (Toggle{Text: "Write tests", Checked: true}); res.Op != want {
		t.Fatalf("Op = %+v, want %+v", res.Op, want)
	}
	if view := ansi.Strip(w.View(40, 10)); !strings.Contains(view, "› ☑ Write tests") {
		t.Errorf("the item should look checked right away:\n%s", view)
	}
}

func TestNewItemPrompt(t *testing.T) {
	_, res := parseBody(body).Update("n")
	if res.Prompt == nil || res.Prompt.Label != "New item" {
		t.Fatalf("Prompt = %+v", res.Prompt)
	}
	if got, want := res.Prompt.Submit("deploy"), (AddItem{Text: "deploy"}); got != want {
		t.Errorf("Submit = %+v", got)
	}
}

func TestViewKeepsCursorVisible(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 20; i++ {
		sb.WriteString("- [ ] item ")
		sb.WriteString(strings.Repeat("x", i+1))
		sb.WriteString("\n")
	}
	w := parseBody(sb.String())
	for i := 0; i < 19; i++ {
		w, _ = w.Update("j")
	}
	view := ansi.Strip(w.View(40, 5))
	if !strings.Contains(view, "› ☐ item "+strings.Repeat("x", 20)) {
		t.Errorf("cursor line is not visible:\n%s", view)
	}
	if n := strings.Count(view, "\n") + 1; n > 5 {
		t.Errorf("View is %d lines tall, want at most 5", n)
	}
}

func TestSyncClampsCursor(t *testing.T) {
	w := parseBody(body)
	w, _ = w.Update("j")
	w, _ = w.Update("j")
	w = w.Sync(doc.Document{Body: "- [ ] only\n"})
	if view := ansi.Strip(w.View(40, 5)); !strings.Contains(view, "› ☐ only") {
		t.Errorf("View = %q", view)
	}
}

func TestKindTemplate(t *testing.T) {
	if got := string(Kind.Template("Release")); got != "---\ntype: checklist\ntitle: Release\n---\n- [ ] \n" {
		t.Errorf("Template = %q", got)
	}
}
```

- [ ] **Step 2: 테스트가 실패하는지 확인한다**

Run: `go test ./internal/widget/checklist/`
Expected: 컴파일 실패. `undefined: Kind` 같은 오류가 나온다.

- [ ] **Step 3: 구현한다**

**`internal/widget/checklist/checklist.go`**

```go
// Package checklist is the progress note: "- [ ]" and "- [x]" lines.
package checklist

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

const (
	previewOpen = 4  // open items shown on the board
	barWidth    = 20 // widest the progress bar gets
	formatHint  = `no items yet: add "- [ ] task" lines`
)

// itemRe splits a checkbox line into: prefix up to "[", the mark, "] ", text.
var itemRe = regexp.MustCompile(`^(\s*[-*] \[)([ xX])(\] ?)(.*)$`)

// Kind registers the checklist.
var Kind = widget.Kind{
	Name:     "checklist",
	Label:    "Checklist",
	Template: func(title string) []byte { return widget.NewFile("checklist", title, "- [ ] \n") },
	Parse:    func(d doc.Document) widget.Widget { return parse(d) },
}

type entry struct {
	text    string
	item    bool
	checked bool
}

// Checklist is the widget for a checklist note.
type Checklist struct {
	entries []entry
	items   []int // indexes of the checkbox entries
	cursor  int   // index into items
	offset  int
}

func parse(d doc.Document) *Checklist {
	c := &Checklist{}
	for _, line := range doc.Lines(d.Body) {
		line = strings.TrimRight(line, "\r")
		if m := itemRe.FindStringSubmatch(line); m != nil {
			c.items = append(c.items, len(c.entries))
			c.entries = append(c.entries, entry{text: strings.TrimSpace(m[4]), item: true, checked: m[2] != " "})
			continue
		}
		c.entries = append(c.entries, entry{text: strings.TrimRight(line, " \t")})
	}
	for n := len(c.entries); n > 0 && !c.entries[n-1].item && c.entries[n-1].text == ""; n = len(c.entries) {
		c.entries = c.entries[:n-1]
	}
	return c
}

func (c *Checklist) counts() (done, total int) {
	for _, i := range c.items {
		if c.entries[i].checked {
			done++
		}
	}
	return done, len(c.items)
}

func bar(done, total, width int) string {
	label := fmt.Sprintf(" %d/%d", done, total)
	w := min(barWidth, width-widget.Width(label))
	if w < 1 {
		return widget.Truncate(strings.TrimSpace(label), width)
	}
	fill := done * w / total
	return strings.Repeat("▓", fill) + strings.Repeat("░", w-fill) + label
}

// Preview implements widget.Widget.
func (c *Checklist) Preview(width int) string {
	done, total := c.counts()
	if total == 0 {
		return widget.Faint.Render(widget.Truncate(formatHint, width))
	}
	lines := []string{bar(done, total, width)}
	open := 0
	for _, i := range c.items {
		if e := c.entries[i]; !e.checked {
			if open++; open <= previewOpen {
				lines = append(lines, widget.Truncate("☐ "+widget.Clean(e.text), width))
			}
		}
	}
	switch {
	case open == 0:
		lines = append(lines, "✓ all done")
	case open > previewOpen:
		lines = append(lines, widget.Faint.Render(fmt.Sprintf("+%d more", open-previewOpen)))
	}
	return strings.Join(lines, "\n")
}

// View implements widget.Widget.
func (c *Checklist) View(width, height int) string {
	done, total := c.counts()
	if total == 0 {
		return widget.Faint.Render(widget.Truncate(formatHint, width))
	}
	c.clamp()
	lines := []string{bar(done, total, width), ""}
	cursorLine := 0
	for i, e := range c.entries {
		if !e.item {
			lines = append(lines, widget.Truncate(widget.Clean(e.text), width))
			continue
		}
		mark := "☐ "
		if e.checked {
			mark = "☑ "
		}
		text := widget.Truncate(mark+widget.Clean(e.text), width-2)
		if c.items[c.cursor] == i {
			cursorLine = len(lines)
			lines = append(lines, widget.Selected.Render("› "+text))
		} else {
			lines = append(lines, "  "+text)
		}
	}
	if cursorLine < c.offset {
		c.offset = cursorLine
	}
	if cursorLine >= c.offset+height {
		c.offset = cursorLine - height + 1
	}
	return strings.Join(widget.Window(lines, c.offset, height), "\n")
}

func (c *Checklist) clamp() {
	c.cursor = max(min(c.cursor, len(c.items)-1), 0)
}

// Update implements widget.Widget.
func (c *Checklist) Update(key string) (widget.Widget, widget.Result) {
	var res widget.Result
	switch key {
	case "j", "down":
		c.cursor++
	case "k", "up":
		c.cursor--
	case "space", "x":
		c.clamp()
		if len(c.items) > 0 {
			e := &c.entries[c.items[c.cursor]]
			res.Op = Toggle{Text: e.text, Checked: !e.checked}
			e.checked = !e.checked
		}
	case "n":
		res.Prompt = &widget.Prompt{
			Label:  "New item",
			Submit: func(text string) doc.Op { return AddItem{Text: text} },
		}
	}
	c.clamp()
	return c, res
}

// Sync implements widget.Widget.
func (c *Checklist) Sync(d doc.Document) widget.Widget {
	nc := parse(d)
	nc.cursor, nc.offset = c.cursor, c.offset
	nc.clamp()
	return nc
}

// Toggle sets the first item with Text that is not yet in state Checked.
type Toggle struct {
	Text    string
	Checked bool
}

// Apply implements doc.Op.
func (o Toggle) Apply(d doc.Document) (doc.Document, error) {
	lines := doc.Lines(d.Body)
	for i, line := range lines {
		raw := strings.TrimSuffix(line, "\r")
		m := itemRe.FindStringSubmatch(raw)
		if m == nil || strings.TrimSpace(m[4]) != o.Text || (m[2] != " ") == o.Checked {
			continue
		}
		mark := " "
		if o.Checked {
			mark = "x"
		}
		lines[i] = m[1] + mark + m[3] + m[4] + line[len(raw):]
		d.Body = doc.Join(lines)
		return d, nil
	}
	return d, doc.ErrConflict
}

// AddItem appends an unchecked item after the last checkbox line, or at the
// end of the body when there is none.
type AddItem struct{ Text string }

// Apply implements doc.Op.
func (o AddItem) Apply(d doc.Document) (doc.Document, error) {
	lines := doc.Lines(d.Body)
	at := len(lines)
	if lines[at-1] == "" {
		at-- // keep the final newline at the end
	}
	for i, line := range lines {
		if itemRe.MatchString(strings.TrimSuffix(line, "\r")) {
			at = i + 1
		}
	}
	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:at]...)
	out = append(out, "- [ ] "+o.Text+doc.EOL(d.Body))
	d.Body = doc.Join(append(out, lines[at:]...))
	return d, nil
}
```

- [ ] **Step 4: 테스트가 통과하는지 확인한다**

Run: `go test ./internal/widget/checklist/`
Expected: `ok  	github.com/LeeSwallow/stickypane/internal/widget/checklist`

- [ ] **Step 5: 커밋한다**

```bash
git add internal/widget/checklist
git commit -m "feat(checklist): add the checklist note with toggle and add-item intents"
```

---

### Task 5: `logview` 위젯과 기본 등록부

로그 노트는 본문의 각 줄이 한 항목이다. 열어 두면 끝을 따라간다. 이 작업에서 네 가지 모양을 등록부로 묶고, Glamour로 마크다운을 그리는 렌더러도 함께 만든다.

**Files:**
- Create: `internal/widget/logview/logview.go`
- Create: `internal/kinds/kinds.go`
- Test: `internal/widget/logview/logview_test.go`, `internal/kinds/kinds_test.go`

**Interfaces:**
- Consumes: Task 1의 `doc`, Task 2의 `widget`·`note`, Task 3의 `board.Kind`, Task 4의 `checklist.Kind`
- Produces:
  - `logview.Kind` (`widget.Kind`, `Name: "log"`)
  - `kinds.Default(render note.Renderer) widget.Registry` (순서: note, board, checklist, log)
  - `kinds.Theme{Dark bool}`: 렌더러가 참조하는 화면 밝기. 앱이 터미널의 배경색 응답을 받으면 `Dark`를 바꾼다.
  - `kinds.Markdown(theme *kinds.Theme) note.Renderer`
  - 모달 키: `j` `k` `space` `b` `g` 스크롤(따라가기 해제), `G` 끝으로 가서 따라가기

- [ ] **Step 1: 실패하는 테스트를 쓴다**

**`internal/widget/logview/logview_test.go`**

```go
package logview

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

func parseBody(s string) widget.Widget { return Kind.Parse(doc.Document{Body: s}) }

func numbered(n int) string {
	var sb strings.Builder
	for i := 1; i <= n; i++ {
		sb.WriteString("line ")
		sb.WriteString(strings.Repeat("i", i))
		sb.WriteString("\n")
	}
	return sb.String()
}

func TestPreviewShowsLastFiveLines(t *testing.T) {
	got := parseBody(numbered(8) + "\n\n").Preview(40)
	want := "line iiii\nline iiiii\nline iiiiii\nline iiiiiii\nline iiiiiiii"
	if got != want {
		t.Errorf("Preview = %q, want %q", got, want)
	}
}

func TestPreviewOfEmptyLog(t *testing.T) {
	if got := ansi.Strip(parseBody("\n").Preview(40)); got != "(no entries yet)" {
		t.Errorf("Preview = %q", got)
	}
}

func TestPreviewTruncatesAndCleans(t *testing.T) {
	got := parseBody("14:02 \x1b[32mtests passed\x1b[0m and a very long tail\r\n").Preview(18)
	if got != "14:02 tests passe…" {
		t.Errorf("Preview = %q", got)
	}
}

func TestViewFollowsTheEnd(t *testing.T) {
	w := parseBody(numbered(6))
	if got := w.View(40, 2); got != "line iiiii\nline iiiiii" {
		t.Fatalf("View = %q", got)
	}
	w = w.Sync(doc.Document{Body: numbered(7)})
	if got := w.View(40, 2); got != "line iiiiii\nline iiiiiii" {
		t.Errorf("after a new line: %q", got)
	}
}

func TestScrollingStopsFollowingAndGResumes(t *testing.T) {
	w := parseBody(numbered(6))
	w.View(40, 2)
	w, _ = w.Update("k")
	if got := w.View(40, 2); got != "line iiii\nline iiiii" {
		t.Fatalf("after k: %q", got)
	}
	w = w.Sync(doc.Document{Body: numbered(7)})
	if got := w.View(40, 2); got != "line iiii\nline iiiii" {
		t.Errorf("should stay put while not following: %q", got)
	}
	w, _ = w.Update("G")
	if got := w.View(40, 2); got != "line iiiiii\nline iiiiiii" {
		t.Errorf("after G: %q", got)
	}
}

func TestViewWrapsLongLines(t *testing.T) {
	for _, line := range strings.Split(parseBody("a very long log line that does not fit\n").View(10, 20), "\n") {
		if w := widget.Width(line); w > 10 {
			t.Errorf("line %q is %d cells wide", line, w)
		}
	}
}

func TestKindTemplate(t *testing.T) {
	if got := string(Kind.Template("Work log")); got != "---\ntype: log\ntitle: Work log\n---\n" {
		t.Errorf("Template = %q", got)
	}
}
```

**`internal/kinds/kinds_test.go`**

```go
package kinds

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/widget"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

func TestDefaultRegistry(t *testing.T) {
	reg := Default(note.Plain)
	var names []string
	for _, k := range reg {
		names = append(names, k.Name)
		if k.Label == "" || k.Template == nil || k.Parse == nil {
			t.Errorf("kind %q is incomplete", k.Name)
		}
	}
	if got := strings.Join(names, ","); got != "note,board,checklist,log" {
		t.Errorf("kinds = %s", got)
	}
	if reg.Lookup("mystery").Name != "note" {
		t.Error("unknown types should fall back to the plain note")
	}
}

func TestMarkdownFollowsTheTheme(t *testing.T) {
	theme := &Theme{Dark: true}
	render := Markdown(theme)
	dark := render("## Heading\n\ntext\n", 30)
	theme.Dark = false
	light := render("## Heading\n\ntext\n", 30)
	if dark == light {
		t.Error("flipping the theme should change the colors of later renders")
	}
	if ansi.Strip(dark) != ansi.Strip(light) {
		t.Error("the theme must only change colors, not the text")
	}
}

func TestMarkdownRendersWithinWidth(t *testing.T) {
	for _, dark := range []bool{true, false} {
		out := Markdown(&Theme{Dark: dark})("## 결정 사항\n\n토큰은 세션 쿠키로 보관한다. **Important** item.\n\n- a\n- b\n", 30)
		plain := ansi.Strip(out)
		if !strings.Contains(plain, "결정 사항") || !strings.Contains(plain, "Important") {
			t.Errorf("rendered text lost content: %q", plain)
		}
		for _, line := range strings.Split(out, "\n") {
			if w := widget.Width(line); w > 30 {
				t.Errorf("line %q is %d cells wide, want at most 30", ansi.Strip(line), w)
			}
		}
	}
}
```

- [ ] **Step 2: 테스트가 실패하는지 확인한다**

Run: `go test ./internal/widget/logview/ ./internal/kinds/`
Expected: 컴파일 실패. `undefined: Kind`, `undefined: Default` 같은 오류가 나온다.

- [ ] **Step 3: 구현한다**

**`internal/widget/logview/logview.go`**

```go
// Package logview is the log note: one entry per line, newest at the end.
package logview

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// previewLines is how many of the latest entries the board shows.
const previewLines = 5

// Kind registers the log.
var Kind = widget.Kind{
	Name:     "log",
	Label:    "Log",
	Template: func(title string) []byte { return widget.NewFile("log", title, "") },
	Parse:    func(d doc.Document) widget.Widget { return parse(d, 0, true) },
}

// Log is the widget for a log note.
type Log struct {
	lines  []string
	offset int
	follow bool // stay at the end as lines arrive
}

func parse(d doc.Document, offset int, follow bool) *Log {
	l := &Log{offset: offset, follow: follow}
	for _, line := range doc.Lines(d.Body) {
		l.lines = append(l.lines, strings.TrimRight(widget.Clean(line), " "))
	}
	for n := len(l.lines); n > 0 && l.lines[n-1] == ""; n = len(l.lines) {
		l.lines = l.lines[:n-1]
	}
	return l
}

// Preview implements widget.Widget.
func (l *Log) Preview(width int) string {
	var out []string
	for i := len(l.lines) - 1; i >= 0 && len(out) < previewLines; i-- {
		if l.lines[i] != "" {
			out = append([]string{widget.Truncate(l.lines[i], width)}, out...)
		}
	}
	if len(out) == 0 {
		return widget.Faint.Render("(no entries yet)")
	}
	return strings.Join(out, "\n")
}

// View implements widget.Widget.
func (l *Log) View(width, height int) string {
	var wrapped []string
	for _, line := range l.lines {
		if width > 0 {
			line = ansi.Wrap(line, width, "")
		}
		wrapped = append(wrapped, strings.Split(line, "\n")...)
	}
	if l.follow {
		l.offset = len(wrapped)
	}
	l.offset = widget.ClampOffset(l.offset, len(wrapped), height)
	return strings.Join(widget.Window(wrapped, l.offset, height), "\n")
}

// Update implements widget.Widget.
func (l *Log) Update(key string) (widget.Widget, widget.Result) {
	if key == "G" {
		l.follow = true
	} else if off, ok := widget.ScrollKey(l.offset, key); ok {
		l.offset, l.follow = off, false
	}
	return l, widget.Result{}
}

// Sync implements widget.Widget.
func (l *Log) Sync(d doc.Document) widget.Widget { return parse(d, l.offset, l.follow) }
```

**`internal/kinds/kinds.go`**

```go
// Package kinds bundles the built-in note shapes. To add a shape, create its
// package under internal/widget and add one line to Default.
package kinds

import (
	"charm.land/glamour/v2"
	"charm.land/glamour/v2/styles"

	"github.com/LeeSwallow/stickypane/internal/widget"
	"github.com/LeeSwallow/stickypane/internal/widget/board"
	"github.com/LeeSwallow/stickypane/internal/widget/checklist"
	"github.com/LeeSwallow/stickypane/internal/widget/logview"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

// Default returns the built-in kinds. The plain note comes first because it
// is the fallback for unknown types.
func Default(render note.Renderer) widget.Registry {
	return widget.Registry{
		note.NewKind(render),
		board.Kind,
		checklist.Kind,
		logview.Kind,
	}
}

// Theme tells the Markdown renderer which colors suit the terminal. The app
// flips Dark when the terminal reports its background color, so nothing has
// to block on a terminal query before the first draw.
type Theme struct{ Dark bool }

// Markdown returns a renderer that styles Markdown for the theme as it is at
// each call. It drops Glamour's page margins so short notes stay compact, and
// falls back to plain wrapping if Glamour fails.
func Markdown(theme *Theme) note.Renderer {
	type key struct {
		dark  bool
		width int
	}
	renderers := map[key]*glamour.TermRenderer{}
	return func(markdown string, width int) string {
		if width <= 0 {
			return markdown
		}
		k := key{theme.Dark, width}
		r, ok := renderers[k]
		if !ok {
			cfg := styles.LightStyleConfig
			if theme.Dark {
				cfg = styles.DarkStyleConfig
			}
			var zero uint
			cfg.Document.Margin = &zero
			cfg.Document.BlockPrefix = ""
			cfg.Document.BlockSuffix = ""
			var err error
			r, err = glamour.NewTermRenderer(glamour.WithStyles(cfg), glamour.WithWordWrap(width))
			if err != nil {
				return note.Plain(markdown, width)
			}
			renderers[k] = r
		}
		out, err := r.Render(markdown)
		if err != nil {
			return note.Plain(markdown, width)
		}
		return out
	}
}
```

- [ ] **Step 4: 테스트가 통과하는지 확인한다**

Run: `go test ./internal/widget/... ./internal/kinds/`
Expected: 모든 패키지가 `ok`

- [ ] **Step 5: 커밋한다**

```bash
git add internal/widget/logview internal/kinds
git commit -m "feat(kinds): add the log note and bundle the built-in shapes"
```

---

### Task 6: `store` 패키지 (폴더 읽기, 쓰기, 감시)

`.stickypane/` 폴더를 찾고, 노트를 읽고, 의도를 새로 읽은 파일에 적용해 원자적으로 쓰고, 폴더 변경을 알린다. `Apply`는 화면이 들고 있던 내용이 아니라 그 순간의 파일을 읽으므로, 그사이 에이전트가 고친 내용이 지워지지 않는다.

**Files:**
- Create: `internal/store/store.go`, `internal/store/watch.go`
- Test: `internal/store/store_test.go`, `internal/store/watch_test.go`

**Interfaces:**
- Consumes: `doc.Parse`, `doc.Document`, `doc.Op`, `doc.ErrConflict` (Task 1)
- Produces:
  - 상수 `store.DirName = ".stickypane"`, `store.ArchiveDir = "archive"`, `store.MaxSize = 1 << 20`, `store.Debounce = 100 * time.Millisecond`
  - `store.ErrNotFound`, `store.ErrTooLarge`
  - `store.Note{Name, Path string; Doc doc.Document; ModTime time.Time; Err error}`
  - `store.Find(start string) (string, error)`, `store.Resolve(arg string) (string, error)`
  - `store.Open(dir string) *store.Store`, 필드 `Store.Dir string`
  - `(*Store).Scan() ([]store.Note, error)` (파일 이름순)
  - `(*Store).Apply(name string, op doc.Op) error` (파일이 없으면 `doc.ErrConflict`)
  - `(*Store).Create(text string, content []byte, now time.Time) (name string, err error)`
  - `(*Store).Archive(name string) error`, `(*Store).Delete(name string) error`
  - `store.Slug(text string, now time.Time) string`
  - `(*Store).Watch(ctx context.Context) (<-chan struct{}, error)` (ctx가 끝나면 채널을 닫는다)

- [ ] **Step 1: 실패하는 테스트를 쓴다**

**`internal/store/store_test.go`**

```go
package store

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	dir := filepath.Join(t.TempDir(), DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return Open(dir)
}

func write(t *testing.T, s *Store, name, content string) {
	t.Helper()
	path := filepath.Join(s.Dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, s *Store, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(s.Dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func names(notes []Note) string {
	var out []string
	for _, n := range notes {
		out = append(out, n.Name)
	}
	return strings.Join(out, ",")
}

type failOp struct{}

func (failOp) Apply(d doc.Document) (doc.Document, error) { return d, doc.ErrConflict }

func TestResolve(t *testing.T) {
	root := t.TempDir()
	board := filepath.Join(root, "proj", DirName)
	deep := filepath.Join(root, "proj", "sub", "deep")
	for _, dir := range []string{board, deep} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, arg := range []string{deep, filepath.Join(root, "proj"), board} {
		got, err := Resolve(arg)
		if err != nil || got != board {
			t.Errorf("Resolve(%q) = %q, %v; want %q", arg, got, err, board)
		}
	}
	if _, err := Resolve(t.TempDir()); !errors.Is(err, ErrNotFound) {
		t.Errorf("Resolve without a board: err = %v, want ErrNotFound", err)
	}
}

func TestScanListsOnlyTopLevelMarkdown(t *testing.T) {
	s := newStore(t)
	write(t, s, "b.md", "---\ntype: board\n---\n## A\n")
	write(t, s, "a.md", "hello\n")
	write(t, s, "UPPER.MD", "x\n")
	write(t, s, ".hidden.md", "x\n")
	write(t, s, "notes.txt", "x\n")
	write(t, s, "archive/old.md", "x\n")
	notes, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if got := names(notes); got != "UPPER.MD,a.md,b.md" {
		t.Fatalf("names = %s", got)
	}
	if notes[2].Doc.Type() != "board" || notes[1].Doc.Body != "hello\n" {
		t.Errorf("documents were not parsed: %+v", notes)
	}
	if notes[1].ModTime.IsZero() || notes[1].Path != filepath.Join(s.Dir, "a.md") {
		t.Errorf("note metadata is missing: %+v", notes[1])
	}
}

func TestScanReplacesInvalidUTF8ForDisplayOnly(t *testing.T) {
	s := newStore(t)
	raw := "ok \xff\xfe bad\n"
	write(t, s, "a.md", raw)
	notes, _ := s.Scan()
	if !utf8.ValidString(notes[0].Doc.Body) {
		t.Errorf("Body is not valid UTF-8: %q", notes[0].Doc.Body)
	}
	if err := s.Apply("a.md", doc.SetKey{Key: "pin", Value: "true"}); err != nil {
		t.Fatal(err)
	}
	if got := read(t, s, "a.md"); !strings.HasSuffix(got, raw) {
		t.Errorf("Apply must keep the original bytes, got %q", got)
	}
}

func TestScanLargeFiles(t *testing.T) {
	s := newStore(t)
	var entries bytes.Buffer
	for i := 0; entries.Len() <= MaxSize; i++ {
		fmt.Fprintf(&entries, "entry %06d\n", i)
	}
	last := entries.String()[entries.Len()-len("entry 000000\n"):]
	write(t, s, "big-log.md", "---\ntype: log\n---\n"+entries.String())
	write(t, s, "big-note.md", entries.String())
	notes, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}
	log, plain := notes[0], notes[1]
	if log.Err != nil {
		t.Fatalf("a large log must still load: %v", log.Err)
	}
	if n := len(log.Doc.Body); n == 0 || n > 64<<10 {
		t.Errorf("log body is %d bytes, want the last 64 KB at most", n)
	}
	if !strings.HasPrefix(log.Doc.Body, "entry ") || !strings.HasSuffix(log.Doc.Body, last) {
		t.Errorf("log tail should start at a line boundary and end with the last entry")
	}
	if !errors.Is(plain.Err, ErrTooLarge) {
		t.Errorf("a large plain note: Err = %v, want ErrTooLarge", plain.Err)
	}
}

func TestScanReportsUnreadableFiles(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read everything")
	}
	s := newStore(t)
	write(t, s, "secret.md", "x\n")
	if err := os.Chmod(filepath.Join(s.Dir, "secret.md"), 0); err != nil {
		t.Fatal(err)
	}
	notes, err := s.Scan()
	if err != nil || len(notes) != 1 || notes[0].Err == nil {
		t.Errorf("want one note carrying a read error, got %+v, %v", notes, err)
	}
}

func TestApplyWritesAndKeepsMode(t *testing.T) {
	s := newStore(t)
	write(t, s, "a.md", "hello\n")
	path := filepath.Join(s.Dir, "a.md")
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := s.Apply("a.md", doc.SetKey{Key: "pin", Value: "true"}); err != nil {
		t.Fatal(err)
	}
	if got := read(t, s, "a.md"); got != "---\npin: true\n---\nhello\n" {
		t.Errorf("content = %q", got)
	}
	if fi, _ := os.Stat(path); fi.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v, want 0600", fi.Mode().Perm())
	}
}

func TestApplyUsesTheFileAsItIsNow(t *testing.T) {
	s := newStore(t)
	write(t, s, "a.md", "first\n")
	if _, err := s.Scan(); err != nil {
		t.Fatal(err)
	}
	write(t, s, "a.md", "first\nadded by the agent\n")
	if err := s.Apply("a.md", doc.SetKey{Key: "color", Value: "blue"}); err != nil {
		t.Fatal(err)
	}
	if got := read(t, s, "a.md"); got != "---\ncolor: blue\n---\nfirst\nadded by the agent\n" {
		t.Errorf("the agent's edit was lost: %q", got)
	}
}

func TestApplyConflictLeavesFileAlone(t *testing.T) {
	s := newStore(t)
	write(t, s, "a.md", "hello\n")
	if err := s.Apply("a.md", failOp{}); !errors.Is(err, doc.ErrConflict) {
		t.Errorf("err = %v, want ErrConflict", err)
	}
	if got := read(t, s, "a.md"); got != "hello\n" {
		t.Errorf("content = %q", got)
	}
	entries, _ := os.ReadDir(s.Dir)
	if len(entries) != 1 {
		t.Errorf("temporary files were left behind: %v", entries)
	}
}

func TestApplyOnMissingFileIsConflict(t *testing.T) {
	if err := newStore(t).Apply("gone.md", doc.SetKey{Key: "pin", Value: "true"}); !errors.Is(err, doc.ErrConflict) {
		t.Errorf("err = %v, want ErrConflict", err)
	}
}

func TestCreatePicksUniqueNames(t *testing.T) {
	s := newStore(t)
	now := time.Date(2026, 10, 1, 14, 2, 0, 0, time.UTC)
	var got []string
	for i := 0; i < 3; i++ {
		name, err := s.Create("Check env", []byte("Check env\n"), now)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, name)
	}
	if want := "check-env.md,check-env-2.md,check-env-3.md"; strings.Join(got, ",") != want {
		t.Errorf("names = %v, want %s", got, want)
	}
	if read(t, s, "check-env-2.md") != "Check env\n" {
		t.Error("content was not written")
	}
}

func TestSlug(t *testing.T) {
	now := time.Date(2026, 10, 1, 14, 2, 5, 0, time.UTC)
	cases := []struct{ text, want string }{
		{"Check env", "check-env"},
		{"배포 전에 env 확인!", "배포-전에-env-확인"},
		{"  Hello,   World_2  ", "hello-world-2"},
		{"a/b\\c:d", "abcd"},
		{"!!!", "note-20261001-140205"},
		{"", "note-20261001-140205"},
		{strings.Repeat("가", 50), strings.Repeat("가", 40)},
		{strings.Repeat("a", 39) + " tail", strings.Repeat("a", 39)},
	}
	for _, c := range cases {
		if got := Slug(c.text, now); got != c.want {
			t.Errorf("Slug(%q) = %q, want %q", c.text, got, c.want)
		}
	}
}

func TestArchiveMovesFile(t *testing.T) {
	s := newStore(t)
	write(t, s, "a.md", "first\n")
	if err := s.Archive("a.md"); err != nil {
		t.Fatal(err)
	}
	write(t, s, "a.md", "second\n")
	if err := s.Archive("a.md"); err != nil {
		t.Fatal(err)
	}
	if notes, _ := s.Scan(); len(notes) != 0 {
		t.Errorf("archived notes are still on the board: %s", names(notes))
	}
	if read(t, s, "archive/a.md") != "first\n" || read(t, s, "archive/a-2.md") != "second\n" {
		t.Error("archive should keep both files")
	}
}

func TestDelete(t *testing.T) {
	s := newStore(t)
	write(t, s, "a.md", "x\n")
	if err := s.Delete("a.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(s.Dir, "a.md")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("file still exists: %v", err)
	}
}
```

**`internal/store/watch_test.go`**

```go
package store

import (
	"context"
	"testing"
	"time"
)

func TestWatchSignalsChanges(t *testing.T) {
	s := newStore(t)
	ch, err := s.Watch(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	write(t, s, "a.md", "hello\n")
	select {
	case _, ok := <-ch:
		if !ok {
			t.Fatal("channel closed early")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no signal after a file was written")
	}
}

func TestWatchSignalsDuringContinuousWrites(t *testing.T) {
	s := newStore(t)
	ch, err := s.Watch(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
				write(t, s, "log.md", time.Now().String())
				time.Sleep(20 * time.Millisecond)
			}
		}
	}()
	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Error("a steady stream of writes must not starve the signal")
	}
	close(stop)
	<-done
}

func TestWatchClosesOnCancel(t *testing.T) {
	s := newStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := s.Watch(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("got a signal instead of a closed channel")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("channel was not closed after cancel")
	}
}
```

- [ ] **Step 2: 테스트가 실패하는지 확인한다**

Run: `go test ./internal/store/`
Expected: 컴파일 실패. `undefined: Open`, `undefined: Resolve` 같은 오류가 나온다.

- [ ] **Step 3: 읽기와 쓰기를 구현한다**

**`internal/store/store.go`**

```go
// Package store reads, writes and watches the notes folder.
package store

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

const (
	// DirName is the notes folder inside a project.
	DirName = ".stickypane"
	// ArchiveDir is where detached notes go, inside DirName.
	ArchiveDir = "archive"
	// MaxSize is the largest file read in full.
	MaxSize = 1 << 20

	logTail   = 64 << 10 // how much of an oversized log is shown
	slugRunes = 40
	tmpPrefix = ".stickypane-tmp-"
)

var (
	// ErrNotFound means no notes folder exists at or above the start path.
	ErrNotFound = errors.New("no " + DirName + " directory found")
	// ErrTooLarge marks a note that is too big to show.
	ErrTooLarge = errors.New("file is larger than 1 MB")
)

// Note is one file in the notes folder.
type Note struct {
	Name    string // file name, such as "10-plan.md"
	Path    string
	Doc     doc.Document
	ModTime time.Time
	Err     error // why the note cannot be shown; Doc is empty when set
}

// Store is a notes folder.
type Store struct{ Dir string }

// Open returns the store for a notes folder.
func Open(dir string) *Store { return &Store{Dir: dir} }

// Find walks up from start until it finds a notes folder.
func Find(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, DirName)
		if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotFound
		}
		dir = parent
	}
}

// Resolve accepts either a notes folder itself or any path inside a project.
func Resolve(arg string) (string, error) {
	abs, err := filepath.Abs(arg)
	if err != nil {
		return "", err
	}
	if filepath.Base(abs) == DirName {
		if fi, err := os.Stat(abs); err == nil && fi.IsDir() {
			return abs, nil
		}
	}
	return Find(abs)
}

// Scan reads every note, sorted by file name. A file that cannot be read
// still appears, carrying its error.
func (s *Store) Scan() ([]Note, error) {
	entries, err := os.ReadDir(s.Dir) // sorted by name
	if err != nil {
		return nil, err
	}
	var notes []Note
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasPrefix(name, ".") || !strings.EqualFold(filepath.Ext(name), ".md") {
			continue
		}
		notes = append(notes, s.read(name))
	}
	return notes, nil
}

func (s *Store) read(name string) Note {
	n := Note{Name: name, Path: filepath.Join(s.Dir, name)}
	fi, err := os.Stat(n.Path)
	if err != nil {
		n.Err = err
		return n
	}
	n.ModTime = fi.ModTime()
	if fi.Size() <= MaxSize {
		b, err := os.ReadFile(n.Path)
		if err != nil {
			n.Err = err
			return n
		}
		n.Doc = doc.Parse(displayable(b))
		return n
	}
	head, tail, err := readEnds(n.Path, fi.Size())
	if err != nil {
		n.Err = err
		return n
	}
	d := doc.Parse(displayable(head))
	if d.Type() != "log" {
		n.Err = ErrTooLarge
		return n
	}
	if i := bytes.IndexByte(tail, '\n'); i >= 0 {
		tail = tail[i+1:] // start at a line boundary
	}
	d.Body = string(displayable(tail))
	n.Doc = d
	return n
}

// displayable replaces invalid UTF-8 so the text is safe to draw. Only what
// the screen shows is changed; Apply always works on the file's real bytes.
func displayable(b []byte) []byte { return bytes.ToValidUTF8(b, []byte("\uFFFD")) }

// readEnds returns the first and last logTail bytes of a large file.
func readEnds(path string, size int64) (head, tail []byte, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	head = make([]byte, logTail)
	n, err := io.ReadFull(f, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, nil, err
	}
	head = head[:n]
	tail = make([]byte, logTail)
	n, err = f.ReadAt(tail, size-logTail)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, nil, err
	}
	return head, tail[:n], nil
}

// Apply reads the file as it is now, applies op and replaces the file in one
// step. The edit is never based on what the screen last saw, so changes made
// by someone else in the meantime survive. A missing file is a conflict.
func (s *Store) Apply(name string, op doc.Op) error {
	path := filepath.Join(s.Dir, name)
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return doc.ErrConflict
	}
	if err != nil {
		return err
	}
	out, err := op.Apply(doc.Parse(b))
	if err != nil {
		return err
	}
	return writeAtomic(path, out.Bytes())
}

func writeAtomic(path string, data []byte) error {
	mode := fs.FileMode(0o644)
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), tmpPrefix+"*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // a no-op once the rename succeeded
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// Create writes a new note named after text and returns the file name. It
// never overwrites: a taken name gets "-2", "-3" and so on.
func (s *Store) Create(text string, content []byte, now time.Time) (string, error) {
	base := Slug(text, now)
	for i := 1; ; i++ {
		name := base + ".md"
		if i > 1 {
			name = fmt.Sprintf("%s-%d.md", base, i)
		}
		f, err := os.OpenFile(filepath.Join(s.Dir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		_, werr := f.Write(content)
		if cerr := f.Close(); werr == nil {
			werr = cerr
		}
		return name, werr
	}
}

// Slug turns text into a file name stem: letters and digits are kept (Latin
// letters lower-cased), runs of spaces, "-" and "_" become one "-", and
// everything else is dropped. An empty result becomes "note-<timestamp>".
func Slug(text string, now time.Time) string {
	var b strings.Builder
	n, dash := 0, true // dash starts true so the slug never begins with "-"
	for _, r := range text {
		if n >= slugRunes {
			break
		}
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
			n, dash = n+1, false
		case (unicode.IsSpace(r) || r == '-' || r == '_') && !dash:
			b.WriteRune('-')
			n, dash = n+1, true
		}
	}
	if slug := strings.TrimRight(b.String(), "-"); slug != "" {
		return slug
	}
	return "note-" + now.Format("20060102-150405")
}

// Archive moves a note into the archive folder without overwriting.
func (s *Store) Archive(name string) error {
	dir := filepath.Join(s.Dir, ArchiveDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for i := 1; ; i++ {
		target := name
		if i > 1 {
			target = fmt.Sprintf("%s-%d%s", base, i, ext)
		}
		dst := filepath.Join(dir, target)
		if _, err := os.Lstat(dst); err == nil {
			continue
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return os.Rename(filepath.Join(s.Dir, name), dst)
	}
}

// Delete removes a note for good.
func (s *Store) Delete(name string) error {
	return os.Remove(filepath.Join(s.Dir, name))
}
```

- [ ] **Step 4: 감시를 구현한다**

**`internal/store/watch.go`**

```go
package store

import (
	"context"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Debounce is how long changes are collected before one signal is sent.
const Debounce = 100 * time.Millisecond

// Watch signals on the returned channel after the folder changed. Events are
// not interpreted per file, because editors and agents often save by writing
// a temporary file and renaming it; the receiver simply rescans. The timer
// starts at the first event and is not restarted by later ones, so a steady
// stream of writes still produces a signal every Debounce. The channel is
// closed when ctx ends.
func (s *Store) Watch(ctx context.Context) (<-chan struct{}, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := w.Add(s.Dir); err != nil {
		w.Close()
		return nil, err
	}
	ch := make(chan struct{}, 1)
	go func() {
		defer close(ch)
		defer w.Close()
		var fire <-chan time.Time
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-w.Events:
				if !ok {
					return
				}
				if fire == nil {
					fire = time.After(Debounce)
				}
			case _, ok := <-w.Errors:
				if !ok {
					return
				}
			case <-fire:
				fire = nil
				select {
				case ch <- struct{}{}:
				default: // a signal is already waiting
				}
			}
		}
	}()
	return ch, nil
}
```

- [ ] **Step 5: 테스트가 통과하는지 확인한다**

Run: `go test ./internal/store/`
Expected: `ok  	github.com/LeeSwallow/stickypane/internal/store`

- [ ] **Step 6: 커밋한다**

```bash
git add internal/store
git commit -m "feat(store): read, watch and atomically edit the notes folder"
```

---

### Task 7: `layout` 패키지 (배치 계산)

폭과 노트 높이 목록만으로 위치를 계산하는 순수 함수들이다. 열 수는 `폭 ÷ 36`이고, 일반 노트는 가장 짧은 열에 쌓고, 한 줄 전체를 쓰는 노트는 모든 열 아래에 놓는다.

**Files:**
- Create: `internal/layout/layout.go`
- Test: `internal/layout/layout_test.go`

**Interfaces:**
- Consumes: 없음
- Produces:
  - `layout.MinWidth = 36`
  - `layout.Rect{X, Y, W, H int}`, `layout.Item{Height int; FullRow bool}`
  - `layout.Columns(width int) (n, colWidth int)`
  - `layout.Flow(width int, items []layout.Item) []layout.Rect`
  - `layout.Dir`와 상수 `layout.Up`, `layout.Down`, `layout.Left`, `layout.Right`
  - `layout.Neighbor(rects []layout.Rect, cur int, d layout.Dir) int` (없으면 `cur`)
  - `layout.Compose(rects []layout.Rect, boxes []string) []string` (각 box의 줄은 정확히 `Rect.W`칸이어야 한다)

- [ ] **Step 1: 실패하는 테스트를 쓴다**

**`internal/layout/layout_test.go`**

```go
package layout

import (
	"reflect"
	"testing"
)

func TestColumns(t *testing.T) {
	cases := []struct{ width, n, cw int }{
		{0, 1, 0}, {20, 1, 20}, {40, 1, 40}, {71, 1, 71},
		{72, 2, 36}, {80, 2, 40}, {100, 2, 50}, {120, 3, 40},
	}
	for _, c := range cases {
		if n, cw := Columns(c.width); n != c.n || cw != c.cw {
			t.Errorf("Columns(%d) = %d, %d; want %d, %d", c.width, n, cw, c.n, c.cw)
		}
	}
}

func TestFlow(t *testing.T) {
	cases := []struct {
		name  string
		width int
		items []Item
		want  []Rect
	}{
		{
			"one column stacks everything", 40,
			[]Item{{Height: 3}, {Height: 4}, {Height: 5, FullRow: true}},
			[]Rect{{0, 0, 40, 3}, {0, 3, 40, 4}, {0, 7, 40, 5}},
		},
		{
			"two columns fill the shorter one", 80,
			[]Item{{Height: 4, FullRow: true}, {Height: 3}, {Height: 5}, {Height: 3}, {Height: 2}},
			[]Rect{{0, 0, 80, 4}, {0, 4, 40, 3}, {40, 4, 40, 5}, {0, 7, 40, 3}, {40, 9, 40, 2}},
		},
		{
			"three columns", 120,
			[]Item{{Height: 3}, {Height: 3}, {Height: 3}, {Height: 2}},
			[]Rect{{0, 0, 40, 3}, {40, 0, 40, 3}, {80, 0, 40, 3}, {0, 3, 40, 2}},
		},
		{
			"a full row goes below every column", 80,
			[]Item{{Height: 2}, {Height: 6}, {Height: 3, FullRow: true}, {Height: 1}},
			[]Rect{{0, 0, 40, 2}, {40, 0, 40, 6}, {0, 6, 80, 3}, {0, 9, 40, 1}},
		},
		{"nothing to place", 80, nil, []Rect{}},
	}
	for _, c := range cases {
		if got := Flow(c.width, c.items); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s:\n got %v\nwant %v", c.name, got, c.want)
		}
	}
}

func TestNeighbor(t *testing.T) {
	// A is a full row; B and D share the left column, C and E the right one.
	rects := []Rect{{0, 0, 80, 4}, {0, 4, 40, 3}, {40, 4, 40, 5}, {0, 7, 40, 3}, {40, 9, 40, 2}}
	const a, b, c, d, e = 0, 1, 2, 3, 4
	cases := []struct {
		cur  int
		dir  Dir
		want int
	}{
		{a, Down, b}, {a, Up, a}, {a, Left, a},
		{b, Right, c}, {b, Down, d}, {b, Up, a}, {b, Left, b},
		{c, Left, b}, {c, Down, e}, {c, Right, c},
		{d, Right, e}, {d, Down, d}, {d, Up, b},
		{e, Up, c}, {e, Left, d},
	}
	for _, tc := range cases {
		if got := Neighbor(rects, tc.cur, tc.dir); got != tc.want {
			t.Errorf("Neighbor(%d, %v) = %d, want %d", tc.cur, tc.dir, got, tc.want)
		}
	}
	if got := Neighbor(rects, 9, Down); got != 9 {
		t.Errorf("an index outside the list should come back unchanged, got %d", got)
	}
}

func TestCompose(t *testing.T) {
	rects := []Rect{{0, 0, 10, 2}, {0, 2, 5, 1}, {5, 2, 5, 2}}
	boxes := []string{"aaaaaaaaaa\nbbbbbbbbbb", "ccccc", "ddddd\neeeee"}
	want := []string{"aaaaaaaaaa", "bbbbbbbbbb", "cccccddddd", "     eeeee"}
	if got := Compose(rects, boxes); !reflect.DeepEqual(got, want) {
		t.Errorf("Compose = %q, want %q", got, want)
	}
	if got := Compose(nil, nil); len(got) != 0 {
		t.Errorf("Compose of nothing = %q", got)
	}
}
```

- [ ] **Step 2: 테스트가 실패하는지 확인한다**

Run: `go test ./internal/layout/`
Expected: 컴파일 실패. `undefined: Columns` 같은 오류가 나온다.

- [ ] **Step 3: 구현한다**

**`internal/layout/layout.go`**

```go
// Package layout places notes on the board. It is pure arithmetic: it knows
// widths and heights, not what a note contains.
package layout

import (
	"sort"
	"strings"
)

// MinWidth is the narrowest a note is drawn when the pane allows it.
const MinWidth = 36

// Rect is a note's place on the board, in terminal cells.
type Rect struct{ X, Y, W, H int }

// Item is what Flow needs to know about a note.
type Item struct {
	Height  int
	FullRow bool // spans every column
}

// Columns returns how many columns fit and how wide each one is.
func Columns(width int) (n, colWidth int) {
	n = max(width/MinWidth, 1)
	return n, width / n
}

// Flow places items in order. A regular item goes to the shortest column
// (the leftmost on a tie); a full-row item goes below every column.
func Flow(width int, items []Item) []Rect {
	n, cw := Columns(width)
	bottoms := make([]int, n)
	rects := make([]Rect, len(items))
	for i, it := range items {
		if it.FullRow {
			y := 0
			for _, b := range bottoms {
				y = max(y, b)
			}
			rects[i] = Rect{0, y, n * cw, it.Height}
			for c := range bottoms {
				bottoms[c] = y + it.Height
			}
			continue
		}
		col := 0
		for c, b := range bottoms {
			if b < bottoms[col] {
				col = c
			}
		}
		rects[i] = Rect{col * cw, bottoms[col], cw, it.Height}
		bottoms[col] += it.Height
	}
	return rects
}

// Dir is a direction for moving the focus.
type Dir int

// Directions.
const (
	Up Dir = iota
	Down
	Left
	Right
)

// overlap is the length shared by the ranges [a0, a1) and [b0, b1).
func overlap(a0, a1, b0, b1 int) int { return min(a1, b1) - max(a0, b0) }

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// Neighbor returns the index of the nearest rect in direction d that shares
// some extent with the current one, or cur when there is none.
func Neighbor(rects []Rect, cur int, d Dir) int {
	if cur < 0 || cur >= len(rects) {
		return cur
	}
	c := rects[cur]
	best, bestDist, bestTie := cur, 0, 0
	for i, r := range rects {
		if i == cur {
			continue
		}
		sameCols := overlap(c.X, c.X+c.W, r.X, r.X+r.W) > 0
		sameRows := overlap(c.Y, c.Y+c.H, r.Y, r.Y+r.H) > 0
		var dist, tie int
		switch {
		case d == Down && sameCols && r.Y >= c.Y+c.H:
			dist, tie = r.Y-(c.Y+c.H), r.X
		case d == Up && sameCols && r.Y+r.H <= c.Y:
			dist, tie = c.Y-(r.Y+r.H), r.X
		case d == Right && sameRows && r.X >= c.X+c.W:
			dist, tie = r.X-(c.X+c.W), abs(r.Y-c.Y)
		case d == Left && sameRows && r.X+r.W <= c.X:
			dist, tie = c.X-(r.X+r.W), abs(r.Y-c.Y)
		default:
			continue
		}
		if best == cur || dist < bestDist || (dist == bestDist && tie < bestTie) {
			best, bestDist, bestTie = i, dist, tie
		}
	}
	return best
}

// Compose paints each box at its rect and returns the board as lines. Every
// line of a box must be exactly its rect's width; gaps are filled with spaces.
func Compose(rects []Rect, boxes []string) []string {
	type segment struct {
		x, w int
		text string
	}
	height := 0
	for _, r := range rects {
		height = max(height, r.Y+r.H)
	}
	rows := make([][]segment, height)
	for i, r := range rects {
		lines := strings.Split(boxes[i], "\n")
		for dy := 0; dy < r.H; dy++ {
			text := strings.Repeat(" ", r.W)
			if dy < len(lines) {
				text = lines[dy]
			}
			rows[r.Y+dy] = append(rows[r.Y+dy], segment{r.X, r.W, text})
		}
	}
	out := make([]string, height)
	for y, segs := range rows {
		sort.Slice(segs, func(a, b int) bool { return segs[a].x < segs[b].x })
		var sb strings.Builder
		x := 0
		for _, s := range segs {
			if s.x > x {
				sb.WriteString(strings.Repeat(" ", s.x-x))
			}
			sb.WriteString(s.text)
			x = s.x + s.w
		}
		out[y] = sb.String()
	}
	return out
}
```

- [ ] **Step 4: 테스트가 통과하는지 확인한다**

Run: `go test ./internal/layout/`
Expected: `ok  	github.com/LeeSwallow/stickypane/internal/layout`

- [ ] **Step 5: 커밋한다**

```bash
git add internal/layout
git commit -m "feat(layout): flow notes into columns and find focus neighbors"
```

---

### Task 8: `app` 보드 화면

Bubble Tea 모델을 만들고 노트를 보드로 그린다. 포커스 이동, 고정 노트 우선 정렬, 갱신 표시(`●`), 스크롤, 빈 화면을 다룬다. 다른 화면(모달, 입력창 등)은 Task 9와 10에서 자기 파일의 `init()`으로 처리기 표에 등록한다. 그래서 이 작업의 `Model`에는 뒤 작업이 쓸 필드가 미리 선언되어 있고, 바닥 줄 안내에는 뒤 작업에서 동작하게 될 키(`n`, `enter`, `a`, `?`)가 이미 적혀 있다.

**Files:**
- Create: `internal/app/app.go`, `internal/app/screen.go`, `internal/app/frame.go`, `internal/app/boardscreen.go`
- Test: `internal/app/app_test.go`, `internal/app/frame_test.go`

**Interfaces:**
- Consumes: `store.Store`, `store.Note` (Task 6), `widget.Registry`, `widget.Kind`, `widget.Widget` (Task 2), `kinds.Default`, `note.Plain` (Task 5, 테스트용), `layout.Columns`, `layout.Flow`, `layout.Neighbor`, `layout.Compose` (Task 7), `doc.Op`, `doc.ErrConflict` (Task 1)
- Produces:
  - `app.New(st *store.Store, reg widget.Registry, watch <-chan struct{}) *app.Model` (`tea.Model` 구현, `watch`는 nil 가능)
  - `(*Model).SetStatus(s string)`
  - 필드 `Model.OnBackground func(dark bool)`: 터미널이 배경색을 알려 주면 불린다. 그 뒤 모든 노트를 다시 그린다.
  - 패키지 내부 등록 표: `handlers map[mode]func(*Model, tea.Msg) tea.Cmd`, `bodies map[mode]func(*Model, int) []string`, `footers map[mode]func(*Model) string`, `boardKeys map[string]func(*Model) tea.Cmd`
  - 패키지 내부 도우미: `(*Model).index(name string) int`, `(*Model).setFocus(name string)`, `(*Model).reload()`, `(*Model).apply(name string, op doc.Op)`, `(*Model).heading(it item) string`, `label(it item) string`, `frame(title, body string, width int, c color.Color, focused bool) string`, `palette`, `colorIndex(name, key string) int`
  - 모드 상수: `modeBoard`, `modeModal`, `modeInput`, `modeCatalog`, `modeConfirm`, `modeHelp`
  - 메시지: `changedMsg{}`, `reloadMsg{what string; err error}`
  - 테스트 도우미(`app_test.go`): `newModel`, `writeFile`, `readFile`, `key`, `press`, `typeText`, `screen`, 상수 `boardFile`

- [ ] **Step 1: 실패하는 테스트를 쓴다**

**`internal/app/app_test.go`**

```go
package app

import (
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
	"github.com/LeeSwallow/stickypane/internal/widget"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

const boardFile = "---\ntype: board\ntitle: Auth\n---\n## To do\n- payments\n## Doing\n- login API\n## Done\n- schema\n"

// newModel builds a board over a temporary notes folder, sized 80x24.
func newModel(t *testing.T, files map[string]string) (*Model, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), store.DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		writeFile(t, dir, name, content)
	}
	m := New(store.Open(dir), kinds.Default(note.Plain), nil)
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
	for _, k := range []string{"enter", "esc", "tab", "shift+tab", "space", "ctrl+c", "n", "D", "?"} {
		if got := key(k).String(); got != k {
			t.Errorf("key(%q).String() = %q", k, got)
		}
	}
}

func TestBoardShowsEveryNote(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": "check env\n", "b.md": boardFile})
	s := screen(m)
	for _, want := range []string{"check env", "Auth", "To do (1)", "payments", "n jot"} {
		if !strings.Contains(s, want) {
			t.Errorf("screen should contain %q:\n%s", want, s)
		}
	}
	if n := strings.Count(s, "\n") + 1; n != 24 {
		t.Errorf("screen is %d lines tall, want 24", n)
	}
}

func TestEmptyBoardInvitesToJot(t *testing.T) {
	m, _ := newModel(t, nil)
	if s := screen(m); !strings.Contains(s, "No notes yet") {
		t.Errorf("screen = %q", s)
	}
}

func TestFocusStartsOnFirstNoteAndCycles(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": "one\n", "b.md": "two\n"})
	if m.focus != "a.md" {
		t.Fatalf("focus = %q, want a.md", m.focus)
	}
	if n := strings.Count(screen(m), "╔"); n != 1 {
		t.Errorf("exactly one note should have the focus border, got %d", n)
	}
	press(m, "tab")
	if m.focus != "b.md" {
		t.Errorf("after tab: focus = %q", m.focus)
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

func TestArrowKeysFollowTheLayout(t *testing.T) {
	// 80 cells wide gives two columns: a and c on the left, b on the right.
	m, _ := newModel(t, map[string]string{"a.md": "one\n", "b.md": "two\n", "c.md": "three\n"})
	for _, step := range []struct{ key, want string }{
		{"l", "b.md"}, {"h", "a.md"}, {"j", "c.md"}, {"k", "a.md"}, {"k", "a.md"},
	} {
		press(m, step.key)
		if m.focus != step.want {
			t.Fatalf("after %s: focus = %q, want %q", step.key, m.focus, step.want)
		}
	}
}

func TestPinnedNotesComeFirst(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": "one\n", "z.md": "---\npin: true\n---\npinned\n"})
	if m.items[0].note.Name != "z.md" {
		t.Errorf("first note = %q, want the pinned one", m.items[0].note.Name)
	}
	if !strings.Contains(screen(m), "📌") {
		t.Error("a pinned note should show the pin")
	}
}

func TestChangedNotesAreMarkedUntilFocused(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": "one\n", "b.md": "two\n"})
	if strings.Contains(screen(m), "●") {
		t.Fatal("nothing changed yet")
	}
	writeFile(t, dir, "b.md", "two, edited by the agent\n")
	future := time.Now().Add(time.Minute)
	if err := os.Chtimes(filepath.Join(dir, "b.md"), future, future); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "c.md", "new from the agent\n")
	m.Update(changedMsg{})
	if n := strings.Count(screen(m), "●"); n != 2 {
		t.Fatalf("the edited note and the new note should be marked, got %d marks:\n%s", n, screen(m))
	}
	press(m, "tab", "tab")
	if strings.Contains(screen(m), "●") {
		t.Errorf("marks should clear once the notes were focused:\n%s", screen(m))
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

func TestScrollKeepsFocusedNoteVisible(t *testing.T) {
	files := map[string]string{}
	for i := 0; i < 30; i++ {
		files[string(rune('a'+i%26))+strings.Repeat("x", i/26)+".md"] = "note\n"
	}
	m, _ := newModel(t, files)
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	for i := 0; i < 29; i++ {
		press(m, "tab")
		if !strings.Contains(screen(m), "╔") {
			t.Fatalf("focused note scrolled out of view after %d tabs", i+1)
		}
	}
}

func TestTinyTerminalDoesNotPanic(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": "한글 메모입니다\n", "b.md": boardFile})
	for _, size := range [][2]int{{0, 0}, {1, 1}, {5, 1}, {5, 3}, {10, 3}, {35, 2}, {200, 2}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		press(m, "tab", "j", "l")
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

func TestBackgroundReportRedrawsNotes(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": "one\n", "b.md": "two\n"})
	press(m, "tab")
	var reports []bool
	m.OnBackground = func(dark bool) { reports = append(reports, dark) }
	before := m.items[0].w
	m.Update(tea.BackgroundColorMsg{Color: color.White})
	if len(reports) != 1 || reports[0] {
		t.Errorf("OnBackground calls = %v, want one call with dark=false", reports)
	}
	if m.items[0].w == before {
		t.Error("widgets should be rebuilt so they redraw with the new colors")
	}
	if m.focus != "b.md" || strings.Contains(screen(m), "●") {
		t.Errorf("focus and change marks must survive the redraw, focus = %q", m.focus)
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
```

**`internal/app/frame_test.go`**

```go
package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/widget"
)

func TestFrameTitle(t *testing.T) {
	c := palette[0].color
	cases := []struct {
		title   string
		focused bool
		top     string
	}{
		{"Plan", false, "╭ Plan " + strings.Repeat("─", 12) + "╮"},
		{"", false, "╭" + strings.Repeat("─", 18) + "╮"},
		{"Plan", true, "╔ Plan " + strings.Repeat("═", 12) + "╗"},
	}
	for _, tc := range cases {
		lines := strings.Split(ansi.Strip(frame(tc.title, "x", 20, c, tc.focused)), "\n")
		if lines[0] != tc.top {
			t.Errorf("top = %q, want %q", lines[0], tc.top)
		}
		if len(lines) != 3 {
			t.Errorf("a one-line body should give 3 lines, got %d", len(lines))
		}
	}
}

func TestFrameKeepsWidthWithWideText(t *testing.T) {
	body := "토큰은 세션 쿠키로\nshort\n" + strings.Repeat("긴", 40)
	for _, focused := range []bool{false, true} {
		for _, title := range []string{"", "인증 설계 보고서", "📌 ● 아주 긴 제목입니다 정말로 길어서 잘려야 하는 제목"} {
			for _, line := range strings.Split(frame(title, body, 36, palette[1].color, focused), "\n") {
				if w := widget.Width(line); w != 36 {
					t.Errorf("title %q: line %q is %d cells wide, want 36", title, ansi.Strip(line), w)
				}
			}
		}
	}
}

func TestColorIndex(t *testing.T) {
	if got := colorIndex("a.md", "blue"); palette[got].name != "blue" {
		t.Errorf("a named color should win, got %q", palette[got].name)
	}
	if colorIndex("a.md", "BLUE") != colorIndex("a.md", "blue") {
		t.Error("color names are case-insensitive")
	}
	if colorIndex("a.md", "") != colorIndex("a.md", "no-such-color") {
		t.Error("an unknown color should fall back to the file name's color")
	}
	if colorIndex("a.md", "") != colorIndex("a.md", "") {
		t.Error("the fallback color must be stable")
	}
	if len(palette) != 6 {
		t.Errorf("palette has %d colors, want 6", len(palette))
	}
}
```

- [ ] **Step 2: 테스트가 실패하는지 확인한다**

Run: `go test ./internal/app/`
Expected: 컴파일 실패. `undefined: New`, `undefined: frame` 같은 오류가 나온다.

- [ ] **Step 3: 모델을 구현한다**

**`internal/app/app.go`**

```go
// Package app is the terminal UI: it shows the notes of a store as a board
// and connects keys to widgets and to the store.
package app

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/layout"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

type mode int

const (
	modeBoard   mode = iota // the board with every note
	modeModal               // one note opened
	modeInput               // a one-line prompt over the previous screen
	modeCatalog             // choosing a shape for a new note
	modeConfirm             // a yes/no question over the previous screen
	modeHelp                // the key reference
)

// Screens register themselves in these tables from their own files, so
// adding a screen or a board key never means editing this file.
var (
	// handlers receive every message that is not handled globally.
	handlers = map[mode]func(*Model, tea.Msg) tea.Cmd{}
	// bodies draw everything above the bottom line, in at most the given
	// number of lines. A mode without a body shows the one it came from.
	bodies = map[mode]func(*Model, int) []string{}
	// footers draw the bottom line.
	footers = map[mode]func(*Model) string{}
	// boardKeys are the keys of the board screen.
	boardKeys = map[string]func(*Model) tea.Cmd{}
)

// item is a note on the board with the widget that draws it.
type item struct {
	note store.Note
	kind widget.Kind
	w    widget.Widget
}

// Model is the Bubble Tea model.
type Model struct {
	store *store.Store
	reg   widget.Registry
	watch <-chan struct{}
	now   func() time.Time

	// OnBackground, when set, is told whether the terminal background is
	// dark once the terminal reports it. Every note is redrawn afterwards.
	OnBackground func(dark bool)

	items []item
	focus string               // file name of the focused note
	seen  map[string]time.Time // modification time last looked at, by file name

	width, height int
	rects         []layout.Rect
	canvas        []string
	scroll        int

	mode   mode
	back   mode   // where input, confirm and help return to
	status string // shown on the bottom line until the next key

	modalName string // modal.go: file name of the open note

	input      textinput.Model // input.go
	inputLabel string
	onSubmit   func(string)

	catalogIdx int // create.go

	confirmMsg string // manage.go
	onConfirm  func()
}

// changedMsg arrives when the watched folder changed.
type changedMsg struct{}

// reloadMsg asks for a rescan and, when err is set, reports it first.
type reloadMsg struct {
	what string
	err  error
}

// New returns a model showing the notes in st. watch may be nil when the
// folder cannot be watched; the board then refreshes on "r" and after edits.
func New(st *store.Store, reg widget.Registry, watch <-chan struct{}) *Model {
	m := &Model{store: st, reg: reg, watch: watch, now: time.Now}
	m.reload()
	return m
}

// SetStatus shows a message on the bottom line until the next key.
func (m *Model) SetStatus(s string) { m.status = s }

// Init implements tea.Model. Asking for the background color here, instead
// of before the program starts, keeps startup instant even in a terminal
// that never answers.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(waitForChange(m.watch), tea.RequestBackgroundColor)
}

func waitForChange(ch <-chan struct{}) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		if _, ok := <-ch; !ok {
			return nil
		}
		return changedMsg{}
	}
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case changedMsg:
		m.reload()
		cmd = waitForChange(m.watch)
	case reloadMsg:
		if msg.err != nil {
			m.status = msg.what + ": " + msg.err.Error()
		}
		m.reload()
	case tea.BackgroundColorMsg:
		if m.OnBackground != nil {
			m.OnBackground(msg.IsDark())
		}
		m.items = nil // rebuild every widget so it redraws with the new colors
		m.reload()
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		m.status = ""
		cmd = m.dispatch(msg)
	default:
		cmd = m.dispatch(msg)
	}
	m.relayout()
	return m, cmd
}

func (m *Model) dispatch(msg tea.Msg) tea.Cmd {
	if h, ok := handlers[m.mode]; ok {
		return h(m, msg)
	}
	return nil
}

// index returns the position of the note with the given file name, or -1.
func (m *Model) index(name string) int {
	for i, it := range m.items {
		if it.note.Name == name {
			return i
		}
	}
	return -1
}

func (m *Model) setFocus(name string) {
	m.focus = name
	m.markSeen(name)
}

func (m *Model) markSeen(name string) {
	if i := m.index(name); i >= 0 {
		m.seen[name] = m.items[i].note.ModTime
	}
}

// changed reports whether the note changed since it was last focused.
func (m *Model) changed(it item) bool {
	seen, ok := m.seen[it.note.Name]
	return !ok || !seen.Equal(it.note.ModTime)
}

// reload rescans the folder. Widgets of notes whose body did not change are
// kept, and changed ones are synced, so cursors and scroll positions survive.
func (m *Model) reload() {
	notes, err := m.store.Scan()
	if err != nil {
		m.status = "Cannot read notes: " + err.Error()
		return
	}
	old := make(map[string]item, len(m.items))
	for _, it := range m.items {
		old[it.note.Name] = it
	}
	prev := m.index(m.focus)

	items := make([]item, 0, len(notes))
	for _, n := range notes {
		if n.Err != nil {
			n.Doc = doc.Document{Body: "Cannot show this note: " + n.Err.Error()}
		}
		it := item{note: n, kind: m.reg.Lookup(n.Doc.Type())}
		switch o, ok := old[n.Name]; {
		case !ok || o.kind.Name != it.kind.Name:
			it.w = it.kind.Parse(n.Doc)
		case o.note.Doc.Body == n.Doc.Body:
			it.w = o.w
		default:
			it.w = o.w.Sync(n.Doc)
		}
		items = append(items, it)
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].note.Doc.Pinned() && !items[j].note.Doc.Pinned()
	})
	m.items = items

	if m.seen == nil { // first load: nothing counts as changed yet
		m.seen = make(map[string]time.Time, len(items))
		for _, it := range items {
			m.seen[it.note.Name] = it.note.ModTime
		}
	}
	if m.index(m.focus) < 0 {
		m.focus = ""
		if len(items) > 0 {
			m.focus = items[max(min(prev, len(items)-1), 0)].note.Name
		}
	}
	m.markSeen(m.focus)
	if m.mode == modeModal && m.index(m.modalName) < 0 {
		m.mode = modeBoard
		m.status = "The note was removed."
	}
}

// apply writes an intent to a note and refreshes the board. On a conflict
// nothing is written and the screen simply catches up with the file.
func (m *Model) apply(name string, op doc.Op) {
	err := m.store.Apply(name, op)
	switch {
	case errors.Is(err, doc.ErrConflict):
		m.status = "The file changed on disk, so the change was not applied."
	case err != nil:
		m.status = "Write failed: " + err.Error()
	}
	m.reload()
	m.markSeen(name)
}

// label names a note in prompts: its title, or its file name without ".md".
func label(it item) string {
	if t, _ := it.note.Doc.Get("title"); t != "" {
		return widget.Clean(t)
	}
	return strings.TrimSuffix(it.note.Name, filepath.Ext(it.note.Name))
}

// heading is the text in a note's top border. A plain note without a title
// has none, like a sticky note; other shapes fall back to the file name.
func (m *Model) heading(it item) string {
	t, _ := it.note.Doc.Get("title")
	t = widget.Clean(t)
	if t == "" && (it.kind.Name != m.reg[0].Name || it.note.Err != nil) {
		t = label(it)
	}
	if m.changed(it) {
		t = strings.TrimSpace("● " + t)
	}
	if it.note.Doc.Pinned() {
		t = strings.TrimSpace("📌 " + t)
	}
	return t
}

// relayout redraws every note and places it. It runs after every update, so
// the positions used for moving the focus always match the screen.
func (m *Model) relayout() {
	if m.width <= 0 {
		m.rects, m.canvas = nil, nil
		return
	}
	n, cw := layout.Columns(m.width)
	boxes := make([]string, len(m.items))
	flow := make([]layout.Item, len(m.items))
	for i, it := range m.items {
		w := cw
		if it.kind.FullRow {
			w = n * cw
		}
		boxes[i] = frame(m.heading(it), it.w.Preview(max(w-4, 1)), w, noteColor(it), it.note.Name == m.focus)
		flow[i] = layout.Item{Height: strings.Count(boxes[i], "\n") + 1, FullRow: it.kind.FullRow}
	}
	m.rects = layout.Flow(m.width, flow)
	m.canvas = layout.Compose(m.rects, boxes)

	h := m.height - 1
	if i := m.index(m.focus); i >= 0 && h > 0 {
		r := m.rects[i]
		if r.Y+r.H > m.scroll+h {
			m.scroll = r.Y + r.H - h
		}
		if r.Y < m.scroll {
			m.scroll = r.Y // also wins when the note is taller than the screen
		}
	}
	m.scroll = widget.ClampOffset(m.scroll, len(m.canvas), h)
}
```

**`internal/app/screen.go`**

```go
package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// View implements tea.Model.
func (m *Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

// render draws the screen: the current body above one bottom line. No line
// is wider than the terminal and there are never more lines than rows.
func (m *Model) render() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}
	h := m.height - 1
	var lines []string
	if body, ok := bodies[m.bodyMode()]; ok {
		lines = append(lines, body(m, h)...)
	}
	for len(lines) < h {
		lines = append(lines, "")
	}
	lines = append(lines[:h], m.footer())
	for i, l := range lines {
		lines[i] = ansi.Truncate(l, m.width, "")
	}
	return strings.Join(lines, "\n")
}

// bodyMode is the mode whose body is drawn. Prompts and confirmations have
// no body of their own and keep showing the screen they were opened from.
func (m *Model) bodyMode() mode {
	if _, ok := bodies[m.mode]; ok {
		return m.mode
	}
	return m.back
}

func (m *Model) footer() string {
	asking := m.mode == modeInput || m.mode == modeConfirm
	if m.status != "" && !asking {
		return m.status
	}
	if f, ok := footers[m.mode]; ok {
		return f(m)
	}
	return ""
}
```

**`internal/app/frame.go`**

```go
package app

import (
	"hash/fnv"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/LeeSwallow/stickypane/internal/widget"
)

// palette is the set of note colors, in the order "c" cycles through them.
var palette = []struct {
	name  string
	color color.Color
}{
	{"yellow", lipgloss.Color("#F2D45C")},
	{"pink", lipgloss.Color("#F28FB1")},
	{"blue", lipgloss.Color("#7FB3F5")},
	{"green", lipgloss.Color("#8FD694")},
	{"purple", lipgloss.Color("#B79CF2")},
	{"orange", lipgloss.Color("#F5A962")},
}

// colorIndex picks a note's color: the named one, or one derived from the
// file name so that a note keeps its color between runs.
func colorIndex(name, key string) int {
	for i, p := range palette {
		if strings.EqualFold(p.name, key) {
			return i
		}
	}
	h := fnv.New32a()
	h.Write([]byte(name))
	return int(h.Sum32() % uint32(len(palette)))
}

func noteColor(it item) color.Color {
	key, _ := it.note.Doc.Get("color")
	return palette[colorIndex(it.note.Name, key)].color
}

// frame draws a note: a rounded border in the note's color, or a double
// border when focused, so focus is visible without color too. The title sits
// in the top border. Every line is exactly width cells wide.
func frame(title, body string, width int, c color.Color, focused bool) string {
	width = max(width, 8)
	tl, tr, bl, br, hz, vt := "╭", "╮", "╰", "╯", "─", "│"
	if focused {
		tl, tr, bl, br, hz, vt = "╔", "╗", "╚", "╝", "═", "║"
	}
	st := lipgloss.NewStyle().Foreground(c)

	top := st.Render(tl + strings.Repeat(hz, width-2) + tr)
	if title != "" {
		t := widget.Truncate(title, width-5)
		top = st.Render(tl+" ") + st.Bold(true).Render(t) +
			st.Render(" "+strings.Repeat(hz, width-4-widget.Width(t))+tr)
	}
	lines := []string{top}
	for _, l := range strings.Split(body, "\n") {
		lines = append(lines, st.Render(vt)+" "+widget.Pad(l, width-4)+" "+st.Render(vt))
	}
	lines = append(lines, st.Render(bl+strings.Repeat(hz, width-2)+br))
	return strings.Join(lines, "\n")
}
```

**`internal/app/boardscreen.go`**

```go
package app

import (
	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/layout"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

func init() {
	handlers[modeBoard] = boardUpdate
	bodies[modeBoard] = boardBody
	footers[modeBoard] = func(*Model) string { return "n jot  enter open  a add  ? help  q quit" }

	for key, dir := range map[string]layout.Dir{
		"up": layout.Up, "k": layout.Up,
		"down": layout.Down, "j": layout.Down,
		"left": layout.Left, "h": layout.Left,
		"right": layout.Right, "l": layout.Right,
	} {
		boardKeys[key] = func(m *Model) tea.Cmd { m.moveFocus(dir); return nil }
	}
	boardKeys["tab"] = func(m *Model) tea.Cmd { m.stepFocus(1); return nil }
	boardKeys["shift+tab"] = func(m *Model) tea.Cmd { m.stepFocus(-1); return nil }
	boardKeys["r"] = func(m *Model) tea.Cmd { m.reload(); return nil }
	boardKeys["q"] = func(*Model) tea.Cmd { return tea.Quit }
}

func boardUpdate(m *Model, msg tea.Msg) tea.Cmd {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	if f, ok := boardKeys[k.String()]; ok {
		return f(m)
	}
	return nil
}

func boardBody(m *Model, h int) []string {
	if len(m.items) == 0 {
		return []string{"", "  No notes yet. Press n to jot one down."}
	}
	return widget.Window(m.canvas, m.scroll, h)
}

// moveFocus moves to the nearest note in a direction on the screen.
func (m *Model) moveFocus(d layout.Dir) {
	i := m.index(m.focus)
	if j := layout.Neighbor(m.rects, i, d); j >= 0 && j < len(m.items) {
		m.setFocus(m.items[j].note.Name)
	}
}

// stepFocus moves to the next or previous note in board order, wrapping.
func (m *Model) stepFocus(delta int) {
	if n := len(m.items); n > 0 {
		i := (m.index(m.focus) + delta + n) % n
		m.setFocus(m.items[i].note.Name)
	}
}
```

- [ ] **Step 4: 테스트가 통과하는지 확인한다**

Run: `go test ./internal/app/`
Expected: `ok  	github.com/LeeSwallow/stickypane/internal/app`

- [ ] **Step 5: 커밋한다**

```bash
git add internal/app
git commit -m "feat(app): show notes as a board with focus, pins and change marks"
```

---

### Task 9: `app` 입력창과 모달

노트를 `enter`로 크게 열고, 열린 노트의 키를 위젯에 넘긴다. 위젯이 돌려준 의도는 `Model.apply`로 파일에 쓰고, 위젯이 글자 입력을 요청하면 바닥 줄의 한 줄 입력창으로 받는다. 입력창은 Task 10의 끄적이기와 제목 바꾸기도 함께 쓴다.

**Files:**
- Create: `internal/app/input.go`, `internal/app/modal.go`
- Test: `internal/app/modal_test.go`

**Interfaces:**
- Consumes: Task 8의 `Model`, 등록 표(`handlers`, `bodies`, `footers`, `boardKeys`), `(*Model).apply`, `(*Model).index`, `(*Model).heading`, `frame`, `noteColor`, 테스트 도우미. Task 3·4의 `board.MoveCard`, `board.AddCard`, `checklist.Toggle`은 위젯이 만들어 `widget.Result`로 넘어온다.
- Produces:
  - `(*Model).ask(label, initial string, submit func(text string))`: 한 줄을 입력받는다. `enter`에서 앞뒤 공백을 뗀 값이 비어 있지 않으면 `submit`을 부르고, `esc`는 취소한다. 끝나면 열기 전 화면으로 돌아간다.
  - 보드 키 `enter`: 포커스한 노트를 연다 (`m.mode = modeModal`, `m.modalName`)
  - 모달 키: `esc`·`q` 닫기, `e`는 `boardKeys["e"]`가 등록되어 있으면 그것을 부른다(Task 10), 나머지는 위젯에 전달

- [ ] **Step 1: 실패하는 테스트를 쓴다**

**`internal/app/modal_test.go`**

```go
package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const checklistFile = "---\ntype: checklist\ntitle: Release\n---\n- [x] build\n- [ ] ship\n"

func TestEnterOpensTheWholeNote(t *testing.T) {
	var body strings.Builder
	for i := 1; i <= 15; i++ {
		body.WriteString("line ")
		body.WriteString(strings.Repeat("i", i))
		body.WriteString("\n")
	}
	m, _ := newModel(t, map[string]string{"long.md": body.String()})
	last := "line " + strings.Repeat("i", 15)
	if strings.Contains(screen(m), last) {
		t.Fatal("the board preview should not show the whole note")
	}
	press(m, "enter")
	s := screen(m)
	if !strings.Contains(s, last) || !strings.Contains(s, "esc close") {
		t.Errorf("the open note should show every line and the modal hint:\n%s", s)
	}
	if !strings.Contains(s, "long.md") {
		t.Errorf("an untitled open note should be named by its file:\n%s", s)
	}
	press(m, "esc")
	if strings.Contains(screen(m), last) || m.mode != modeBoard {
		t.Error("esc should return to the board")
	}
}

func TestMovingACardWritesTheFile(t *testing.T) {
	m, dir := newModel(t, map[string]string{"b.md": boardFile})
	press(m, "enter", "L")
	want := "---\ntype: board\ntitle: Auth\n---\n## To do\n## Doing\n- login API\n- payments\n## Done\n- schema\n"
	if got := readFile(t, dir, "b.md"); got != want {
		t.Errorf("file = %q\nwant  %q", got, want)
	}
	if s := screen(m); !strings.Contains(s, "› payments") {
		t.Errorf("the cursor should follow the card:\n%s", s)
	}
}

func TestConflictIsReportedAndNothingIsWritten(t *testing.T) {
	m, dir := newModel(t, map[string]string{"b.md": boardFile})
	press(m, "enter")
	agent := "---\ntype: board\ntitle: Auth\n---\n## To do\n## Doing\n- login API\n## Done\n- schema\n- payments shipped\n"
	writeFile(t, dir, "b.md", agent) // the agent edits before the screen catches up
	press(m, "L")
	if got := readFile(t, dir, "b.md"); got != agent {
		t.Errorf("the agent's version must survive, got %q", got)
	}
	s := screen(m)
	if !strings.Contains(s, "changed on disk") {
		t.Errorf("the conflict should be reported:\n%s", s)
	}
	if !strings.Contains(s, "payments shipped") {
		t.Errorf("the screen should catch up with the file:\n%s", s)
	}
}

func TestPromptAddsACard(t *testing.T) {
	m, dir := newModel(t, map[string]string{"b.md": boardFile})
	press(m, "enter", "n")
	if s := screen(m); !strings.Contains(s, "New card in To do:") {
		t.Fatalf("the prompt should show on the bottom line:\n%s", s)
	}
	typeText(m, "write 테스트")
	press(m, "enter")
	if got := readFile(t, dir, "b.md"); !strings.Contains(got, "## To do\n- payments\n- write 테스트\n## Doing") {
		t.Errorf("file = %q", got)
	}
	if m.mode != modeModal {
		t.Errorf("mode = %v, want the modal again", m.mode)
	}
}

func TestPromptCanBeCancelledOrLeftEmpty(t *testing.T) {
	m, dir := newModel(t, map[string]string{"b.md": boardFile})
	press(m, "enter", "n")
	typeText(m, "q") // "q" is text here, not the close key
	press(m, "esc")
	press(m, "n")
	typeText(m, "   ")
	press(m, "enter")
	if got := readFile(t, dir, "b.md"); got != boardFile {
		t.Errorf("nothing should have been written, got %q", got)
	}
	if m.mode != modeModal {
		t.Errorf("mode = %v, want the modal", m.mode)
	}
}

func TestSpaceTogglesAChecklistItem(t *testing.T) {
	m, dir := newModel(t, map[string]string{"c.md": checklistFile})
	press(m, "enter", "j", "space")
	if got := readFile(t, dir, "c.md"); !strings.HasSuffix(got, "- [x] build\n- [x] ship\n") {
		t.Errorf("file = %q", got)
	}
	press(m, "esc")
	if s := screen(m); !strings.Contains(s, "2/2") {
		t.Errorf("the board preview should show the new progress:\n%s", s)
	}
}

func TestModalClosesWhenNoteIsRemoved(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": "one\n", "b.md": boardFile})
	press(m, "tab", "enter")
	if err := os.Remove(filepath.Join(dir, "b.md")); err != nil {
		t.Fatal(err)
	}
	m.Update(changedMsg{})
	if m.mode != modeBoard {
		t.Fatalf("mode = %v, want the board", m.mode)
	}
	if s := screen(m); !strings.Contains(s, "The note was removed.") {
		t.Errorf("the user should be told:\n%s", s)
	}
	press(m, "enter", "j", "esc") // keeps working on the remaining note
}

func TestModalSurvivesKindChange(t *testing.T) {
	m, dir := newModel(t, map[string]string{"b.md": boardFile})
	press(m, "enter", "l")
	writeFile(t, dir, "b.md", checklistFile)
	m.Update(changedMsg{})
	if s := screen(m); !strings.Contains(s, "1/2") {
		t.Errorf("the open note should now draw as a checklist:\n%s", s)
	}
	press(m, "j", "space")
	if got := readFile(t, dir, "b.md"); !strings.HasSuffix(got, "- [x] ship\n") {
		t.Errorf("keys should reach the new widget, file = %q", got)
	}
}

func TestExternalEditKeepsModalCursor(t *testing.T) {
	m, dir := newModel(t, map[string]string{"b.md": boardFile})
	press(m, "enter", "l")
	writeFile(t, dir, "b.md", strings.Replace(boardFile, "- schema\n", "- schema\n- docs\n", 1))
	m.Update(changedMsg{})
	if s := screen(m); !strings.Contains(s, "› login API") || !strings.Contains(s, "docs") {
		t.Errorf("the cursor should stay and the new card should appear:\n%s", s)
	}
}
```

- [ ] **Step 2: 테스트가 실패하는지 확인한다**

Run: `go test ./internal/app/ -run 'Enter|Moving|Conflict|Prompt|Space|Modal|External'`
Expected: FAIL. `enter`가 아직 아무 일도 하지 않아 `TestEnterOpensTheWholeNote` 등이 실패한다.

- [ ] **Step 3: 입력창을 구현한다**

**`internal/app/input.go`**

```go
package app

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/widget"
)

func init() {
	handlers[modeInput] = inputUpdate
	footers[modeInput] = func(m *Model) string { return m.inputLabel + ": " + m.input.View() }
}

// ask collects one line of text on the bottom line. Enter calls submit with
// the trimmed text unless it is empty; esc cancels. Either way the screen it
// was opened from comes back first.
func (m *Model) ask(label, initial string, submit func(text string)) {
	in := textinput.New()
	in.Prompt = ""
	in.SetValue(initial)
	in.CursorEnd()
	in.SetWidth(max(m.width-widget.Width(label)-3, 1))
	in.Focus()
	m.input, m.inputLabel, m.onSubmit = in, label, submit
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
			if text != "" {
				m.onSubmit(text)
			}
			return nil
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return cmd
}
```

- [ ] **Step 4: 모달을 구현한다**

**`internal/app/modal.go`**

```go
package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

func init() {
	handlers[modeModal] = modalUpdate
	bodies[modeModal] = modalBody
	footers[modeModal] = func(*Model) string { return "esc close  e edit" }

	boardKeys["enter"] = func(m *Model) tea.Cmd {
		if m.index(m.focus) >= 0 {
			m.modalName, m.mode = m.focus, modeModal
		}
		return nil
	}
}

func modalUpdate(m *Model, msg tea.Msg) tea.Cmd {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	i := m.index(m.modalName)
	if i < 0 {
		m.mode = modeBoard
		return nil
	}
	switch key := k.String(); key {
	case "esc", "q":
		m.mode = modeBoard
	case "e":
		if edit, ok := boardKeys["e"]; ok {
			return edit(m)
		}
	default:
		name := m.modalName
		w, res := m.items[i].w.Update(key)
		m.items[i].w = w
		if res.Op != nil {
			m.apply(name, res.Op)
		}
		if p := res.Prompt; p != nil {
			m.ask(p.Label, "", func(text string) { m.apply(name, p.Submit(text)) })
		}
	}
	return nil
}

// modalBody draws the open note in a focused frame that fills the screen.
func modalBody(m *Model, h int) []string {
	i := m.index(m.modalName)
	if i < 0 {
		return nil
	}
	it := m.items[i]
	title := m.heading(it)
	if title == "" {
		title = it.note.Name
	}
	rows := max(h-2, 1)
	lines := strings.Split(it.w.View(max(m.width-4, 1), rows), "\n")
	for len(lines) < rows {
		lines = append(lines, "")
	}
	return strings.Split(frame(title, strings.Join(lines[:rows], "\n"), m.width, noteColor(it), true), "\n")
}
```

- [ ] **Step 5: 테스트가 통과하는지 확인한다**

Run: `go test ./internal/app/`
Expected: `ok  	github.com/LeeSwallow/stickypane/internal/app`

- [ ] **Step 6: 커밋한다**

```bash
git add internal/app
git commit -m "feat(app): open notes in a modal and write widget intents to disk"
```

---

### Task 10: `app` 끄적이기, 카탈로그, 노트 관리, 도움말

보드 화면의 나머지 키를 붙인다. 모두 새 파일에서 `boardKeys`에 등록한다.

**Files:**
- Create: `internal/app/create.go`, `internal/app/manage.go`, `internal/app/help.go`
- Test: `internal/app/create_test.go`, `internal/app/manage_test.go`

**Interfaces:**
- Consumes: Task 8의 `Model`, 등록 표, `(*Model).apply`, `(*Model).reload`, `(*Model).setFocus`, `label`, `palette`, `colorIndex`, `reloadMsg`. Task 9의 `(*Model).ask`. Task 6의 `(*Store).Create`, `.Archive`, `.Delete`. Task 1의 `doc.SetKey`.
- Produces:
  - 보드 키: `n` 끄적이기, `a` 카탈로그, `p` 고정, `c` 색, `R` 제목, `x` 떼어 내기, `D` 삭제(확인), `e` 에디터, `?` 도움말
  - `(*Model).confirm(msg string, yes func())`
  - `editorCommand(path string) *exec.Cmd`

- [ ] **Step 1: 실패하는 테스트를 쓴다**

**`internal/app/create_test.go`**

```go
package app

import (
	"os"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/widget/board"
)

func TestJotCreatesAPlainNote(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": "one\n"})
	press(m, "n")
	if s := screen(m); !strings.Contains(s, "Jot:") {
		t.Fatalf("the jot prompt should show:\n%s", s)
	}
	typeText(m, "Check env before deploy")
	press(m, "enter")
	if got := readFile(t, dir, "check-env-before-deploy.md"); got != "Check env before deploy\n" {
		t.Errorf("file = %q", got)
	}
	if m.focus != "check-env-before-deploy.md" {
		t.Errorf("focus = %q, want the new note", m.focus)
	}
	s := screen(m)
	if !strings.Contains(s, "Check env before deploy") || strings.Contains(s, "●") {
		t.Errorf("the new note should show and not be marked as changed:\n%s", s)
	}
}

func TestJotKeepsKoreanInTheFileName(t *testing.T) {
	m, dir := newModel(t, nil)
	press(m, "n")
	typeText(m, "배포 전에 확인")
	press(m, "enter")
	if got := readFile(t, dir, "배포-전에-확인.md"); got != "배포 전에 확인\n" {
		t.Errorf("file = %q", got)
	}
}

func TestEmptyJotCreatesNothing(t *testing.T) {
	m, dir := newModel(t, nil)
	press(m, "n", "enter")
	press(m, "n", "esc")
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("no file should exist, got %v", entries)
	}
}

func TestCatalogCreatesTheChosenShape(t *testing.T) {
	m, dir := newModel(t, nil)
	press(m, "a")
	s := screen(m)
	for _, want := range []string{"Add a note", "Note", "Board (kanban)", "Checklist", "Log"} {
		if !strings.Contains(s, want) {
			t.Fatalf("catalog should list %q:\n%s", want, s)
		}
	}
	press(m, "j", "enter")
	if s := screen(m); !strings.Contains(s, "Title:") {
		t.Fatalf("choosing a shape should ask for a title:\n%s", s)
	}
	typeText(m, "Auth work")
	press(m, "enter")
	if got := readFile(t, dir, "auth-work.md"); got != string(board.Kind.Template("Auth work")) {
		t.Errorf("file = %q", got)
	}
	if m.focus != "auth-work.md" || m.mode != modeBoard {
		t.Errorf("focus = %q, mode = %v", m.focus, m.mode)
	}
}

func TestCatalogCanBeCancelled(t *testing.T) {
	m, dir := newModel(t, nil)
	press(m, "a", "j", "esc")
	if m.mode != modeBoard {
		t.Errorf("mode = %v", m.mode)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("no file should exist, got %v", entries)
	}
}
```

**`internal/app/manage_test.go`**

```go
package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPinToggles(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": "one\n", "b.md": "two\n"})
	press(m, "tab", "p")
	if got := readFile(t, dir, "b.md"); got != "---\npin: true\n---\ntwo\n" {
		t.Fatalf("file = %q", got)
	}
	if m.items[0].note.Name != "b.md" || m.focus != "b.md" {
		t.Errorf("the pinned note should move first and keep the focus")
	}
	press(m, "p")
	if got := readFile(t, dir, "b.md"); got != "---\npin: false\n---\ntwo\n" {
		t.Errorf("file = %q", got)
	}
}

func TestColorCycles(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": "one\n"})
	first := palette[(colorIndex("a.md", "")+1)%len(palette)].name
	second := palette[(colorIndex("a.md", "")+2)%len(palette)].name
	press(m, "c")
	if got := readFile(t, dir, "a.md"); got != "---\ncolor: "+first+"\n---\none\n" {
		t.Fatalf("file = %q, want color %s", got, first)
	}
	press(m, "c")
	if got := readFile(t, dir, "a.md"); got != "---\ncolor: "+second+"\n---\none\n" {
		t.Errorf("file = %q, want color %s", got, second)
	}
}

func TestRenameEditsTheTitle(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": "---\ntitle: Old\n---\nx\n"})
	press(m, "R")
	typeText(m, "er")
	press(m, "enter")
	if got := readFile(t, dir, "a.md"); got != "---\ntitle: Older\n---\nx\n" {
		t.Errorf("file = %q", got)
	}
	if !strings.Contains(screen(m), "Older") {
		t.Error("the new title should show")
	}
}

func TestArchiveMovesTheNoteAway(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": "one\n", "b.md": "two\n"})
	press(m, "x")
	if got := readFile(t, dir, "archive/a.md"); got != "one\n" {
		t.Errorf("archived file = %q", got)
	}
	if m.focus != "b.md" || len(m.items) != 1 {
		t.Errorf("focus = %q, notes = %d", m.focus, len(m.items))
	}
	if !strings.Contains(screen(m), "archive") {
		t.Error("the user should be told where the note went")
	}
}

func TestDeleteAsksFirst(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": "---\ntitle: Plan\n---\nx\n"})
	path := filepath.Join(dir, "a.md")
	press(m, "D")
	if s := screen(m); !strings.Contains(s, "Delete Plan? (y/n)") {
		t.Fatalf("a confirmation should show:\n%s", s)
	}
	press(m, "n")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("answering n must keep the file: %v", err)
	}
	press(m, "D", "y")
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("answering y should delete the file, stat err = %v", err)
	}
	if len(m.items) != 0 || m.mode != modeBoard {
		t.Errorf("notes = %d, mode = %v", len(m.items), m.mode)
	}
}

func TestManageKeysDoNothingOnAnEmptyBoard(t *testing.T) {
	m, _ := newModel(t, nil)
	press(m, "p", "c", "R", "x", "D", "e", "enter")
	if m.mode != modeBoard {
		t.Errorf("mode = %v", m.mode)
	}
}

func TestEditorCommand(t *testing.T) {
	t.Setenv("EDITOR", "code --wait")
	if got := editorCommand("/tmp/a.md").Args; strings.Join(got, " ") != "code --wait /tmp/a.md" {
		t.Errorf("Args = %q", got)
	}
	t.Setenv("EDITOR", "")
	if got := editorCommand("/tmp/a.md").Args; strings.Join(got, " ") != "vi /tmp/a.md" {
		t.Errorf("Args without EDITOR = %q", got)
	}
}

func TestEditorFailureIsReported(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": "one\n"})
	m.Update(reloadMsg{what: "Editor failed", err: errors.New("exit status 1")})
	if s := screen(m); !strings.Contains(s, "Editor failed: exit status 1") {
		t.Errorf("screen:\n%s", s)
	}
}

func TestHelpOpensAndCloses(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": "one\n"})
	press(m, "?")
	s := screen(m)
	for _, want := range []string{"jot a note", "move to archive", "press any key to close"} {
		if !strings.Contains(s, want) {
			t.Fatalf("help should mention %q:\n%s", want, s)
		}
	}
	press(m, "x") // any key closes help and must not archive
	if m.mode != modeBoard || len(m.items) != 1 {
		t.Errorf("mode = %v, notes = %d", m.mode, len(m.items))
	}
}
```

- [ ] **Step 2: 테스트가 실패하는지 확인한다**

Run: `go test ./internal/app/`
Expected: 컴파일 실패. `undefined: editorCommand`가 나온다.

- [ ] **Step 3: 끄적이기와 카탈로그를 구현한다**

**`internal/app/create.go`**

```go
package app

import (
	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/widget"
)

func init() {
	handlers[modeCatalog] = catalogUpdate
	bodies[modeCatalog] = catalogBody
	footers[modeCatalog] = func(*Model) string { return "enter choose  esc cancel" }

	// Jot: one line becomes a plain note. No shape, no title.
	boardKeys["n"] = func(m *Model) tea.Cmd {
		m.ask("Jot", "", func(text string) { m.create(text, []byte(text+"\n")) })
		return nil
	}
	boardKeys["a"] = func(m *Model) tea.Cmd {
		m.catalogIdx, m.mode = 0, modeCatalog
		return nil
	}
}

// create writes a new note named after text and focuses it.
func (m *Model) create(text string, content []byte) {
	name, err := m.store.Create(text, content, m.now())
	if err != nil {
		m.status = "Cannot create the note: " + err.Error()
		return
	}
	m.reload()
	m.setFocus(name)
}

func catalogUpdate(m *Model, msg tea.Msg) tea.Cmd {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	switch k.String() {
	case "esc", "q":
		m.mode = modeBoard
	case "j", "down":
		m.catalogIdx = min(m.catalogIdx+1, len(m.reg)-1)
	case "k", "up":
		m.catalogIdx = max(m.catalogIdx-1, 0)
	case "enter":
		kind := m.reg[m.catalogIdx]
		m.mode = modeBoard
		m.ask("Title", "", func(title string) { m.create(title, kind.Template(title)) })
	}
	return nil
}

func catalogBody(m *Model, _ int) []string {
	lines := []string{"", "  Add a note", ""}
	for i, k := range m.reg {
		if i == m.catalogIdx {
			lines = append(lines, "  "+widget.Selected.Render("› "+k.Label))
		} else {
			lines = append(lines, "    "+k.Label)
		}
	}
	return lines
}
```

- [ ] **Step 4: 노트 관리를 구현한다**

**`internal/app/manage.go`**

```go
package app

import (
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

func init() {
	handlers[modeConfirm] = confirmUpdate
	footers[modeConfirm] = func(m *Model) string { return m.confirmMsg }

	boardKeys["p"] = onFocused(func(m *Model, it item) tea.Cmd {
		value := "true"
		if it.note.Doc.Pinned() {
			value = "false"
		}
		m.apply(it.note.Name, doc.SetKey{Key: "pin", Value: value})
		return nil
	})
	boardKeys["c"] = onFocused(func(m *Model, it item) tea.Cmd {
		current, _ := it.note.Doc.Get("color")
		next := palette[(colorIndex(it.note.Name, current)+1)%len(palette)].name
		m.apply(it.note.Name, doc.SetKey{Key: "color", Value: next})
		return nil
	})
	boardKeys["R"] = onFocused(func(m *Model, it item) tea.Cmd {
		name := it.note.Name
		current, _ := it.note.Doc.Get("title")
		m.ask("Title", current, func(title string) {
			m.apply(name, doc.SetKey{Key: "title", Value: title})
		})
		return nil
	})
	boardKeys["x"] = onFocused(func(m *Model, it item) tea.Cmd {
		if err := m.store.Archive(it.note.Name); err != nil {
			m.status = "Cannot archive the note: " + err.Error()
		} else {
			m.status = "Moved to archive/."
		}
		m.reload()
		return nil
	})
	boardKeys["D"] = onFocused(func(m *Model, it item) tea.Cmd {
		name := it.note.Name
		m.confirm("Delete "+label(it)+"? (y/n)", func() {
			if err := m.store.Delete(name); err != nil {
				m.status = "Cannot delete the note: " + err.Error()
			}
			m.reload()
		})
		return nil
	})
	boardKeys["e"] = onFocused(func(_ *Model, it item) tea.Cmd {
		return tea.ExecProcess(editorCommand(it.note.Path), func(err error) tea.Msg {
			return reloadMsg{what: "Editor failed", err: err}
		})
	})
}

// onFocused runs f for the focused note and does nothing on an empty board.
func onFocused(f func(*Model, item) tea.Cmd) func(*Model) tea.Cmd {
	return func(m *Model) tea.Cmd {
		i := m.index(m.focus)
		if i < 0 {
			return nil
		}
		return f(m, m.items[i])
	}
}

// confirm asks a yes/no question on the bottom line.
func (m *Model) confirm(msg string, yes func()) {
	m.confirmMsg, m.onConfirm = msg, yes
	m.back, m.mode = m.mode, modeConfirm
}

func confirmUpdate(m *Model, msg tea.Msg) tea.Cmd {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	m.mode = m.back
	if s := k.String(); s == "y" || s == "Y" {
		m.onConfirm()
	}
	return nil
}

// editorCommand opens path in $EDITOR, which may carry arguments such as
// "code --wait". Without $EDITOR it falls back to vi.
func editorCommand(path string) *exec.Cmd {
	parts := strings.Fields(os.Getenv("EDITOR"))
	if len(parts) == 0 {
		parts = []string{"vi"}
	}
	return exec.Command(parts[0], append(parts[1:], path)...)
}
```

- [ ] **Step 5: 도움말을 구현한다**

**`internal/app/help.go`**

```go
package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

const helpText = `
  stickypane

  Board
    arrows, h j k l   move focus          tab   next note
    enter             open note           n     jot a note
    a                 add by shape        e     edit in $EDITOR
    p                 pin                 c     change color
    R                 rename              x     move to archive
    D                 delete              r     reload
    ?                 this help           q     quit

  Open note
    esc               close               e     edit in $EDITOR
    j k               move or scroll      G     jump to the end
    board             h l column, H L move card, J K reorder, n new card
    checklist         space toggle, n new item`

func init() {
	handlers[modeHelp] = func(m *Model, msg tea.Msg) tea.Cmd {
		if _, ok := msg.(tea.KeyPressMsg); ok {
			m.mode = modeBoard
		}
		return nil
	}
	bodies[modeHelp] = func(*Model, int) []string { return strings.Split(helpText, "\n") }
	footers[modeHelp] = func(*Model) string { return "press any key to close" }

	boardKeys["?"] = func(m *Model) tea.Cmd {
		m.mode = modeHelp
		return nil
	}
}
```

- [ ] **Step 6: 테스트가 통과하는지 확인한다**

Run: `go test ./internal/app/`
Expected: `ok  	github.com/LeeSwallow/stickypane/internal/app`

- [ ] **Step 7: 커밋한다**

```bash
git add internal/app
git commit -m "feat(app): add jot, shape catalog, note management and help"
```

---

### Task 11: `initcmd` 패키지 (`init`, `guide`)

프로젝트에 `.stickypane/`와 환영 노트를 만들고, 에이전트 안내문을 `AGENTS.md`·`CLAUDE.md`에 넣는다. 안내문은 표식 사이에 들어가므로 다시 실행하면 그 구간만 교체된다. 안내문과 환영 노트는 마크다운 파일로 두고 `embed`로 실행 파일에 넣는다.

**Files:**
- Create: `internal/initcmd/initcmd.go`, `internal/initcmd/guide.md`, `internal/initcmd/welcome.md`
- Test: `internal/initcmd/initcmd_test.go`

**Interfaces:**
- Consumes: `store.DirName` (Task 6)
- Produces:
  - `initcmd.Options{NoAgentDocs bool}`
  - `initcmd.Run(root string, opts initcmd.Options, out io.Writer) error`
  - `initcmd.Guide() string` (표식을 포함한 안내문 전체, 끝에 줄바꿈)

- [ ] **Step 1: 실패하는 테스트를 쓴다**

**`internal/initcmd/initcmd_test.go`**

```go
package initcmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func run(t *testing.T, root string, opts Options) string {
	t.Helper()
	var out bytes.Buffer
	if err := Run(root, opts, &out); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestGuideIsShortAndCoversEveryShape(t *testing.T) {
	g := Guide()
	if !strings.HasPrefix(g, startMark+"\n") || !strings.HasSuffix(g, endMark+"\n") {
		t.Error("the guide must be wrapped in the markers")
	}
	if n := strings.Count(g, "\n"); n > 30 {
		t.Errorf("the guide is %d lines, want at most 30", n)
	}
	for _, want := range []string{".stickypane/", "type: board", "type: checklist", "type: log", "- [ ]", "## Heading", "color", "pin: true"} {
		if !strings.Contains(g, want) {
			t.Errorf("the guide should mention %q", want)
		}
	}
}

func TestRunCreatesBoardWelcomeNoteAndAgentsFile(t *testing.T) {
	root := t.TempDir()
	out := run(t, root, Options{})
	welcome := doc.Parse([]byte(read(t, filepath.Join(root, ".stickypane", "welcome.md"))))
	if !welcome.Pinned() || !strings.Contains(welcome.Body, "jots a note") {
		t.Errorf("the welcome note should be pinned and explain the keys: %q", welcome.Body)
	}
	if got := read(t, filepath.Join(root, "AGENTS.md")); got != Guide() {
		t.Errorf("AGENTS.md = %q, want exactly the guide", got)
	}
	if _, err := os.Stat(filepath.Join(root, "CLAUDE.md")); !errors.Is(err, os.ErrNotExist) {
		t.Error("CLAUDE.md should not be created when it did not exist")
	}
	if !strings.Contains(out, "AGENTS.md") || !strings.Contains(out, "stickypane") {
		t.Errorf("output should say what happened: %q", out)
	}
}

func TestRunAppendsToExistingAgentFilesOnly(t *testing.T) {
	root := t.TempDir()
	claude := filepath.Join(root, "CLAUDE.md")
	if err := os.WriteFile(claude, []byte("# Project rules\n\nUse tabs."), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, root, Options{})
	if got, want := read(t, claude), "# Project rules\n\nUse tabs.\n\n"+Guide(); got != want {
		t.Errorf("CLAUDE.md = %q\nwant %q", got, want)
	}
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); !errors.Is(err, os.ErrNotExist) {
		t.Error("AGENTS.md should not be created when CLAUDE.md exists")
	}
}

func TestRunTwiceChangesNothing(t *testing.T) {
	root := t.TempDir()
	run(t, root, Options{})
	agents := filepath.Join(root, "AGENTS.md")
	welcome := filepath.Join(root, ".stickypane", "welcome.md")
	first := read(t, agents)
	if err := os.Remove(welcome); err != nil { // the user took the note down
		t.Fatal(err)
	}
	run(t, root, Options{})
	if got := read(t, agents); got != first {
		t.Errorf("a second run changed AGENTS.md:\n%q\nwas:\n%q", got, first)
	}
	if _, err := os.Stat(welcome); !errors.Is(err, os.ErrNotExist) {
		t.Error("a second run must not bring the welcome note back")
	}
}

func TestRunReplacesAnOldGuide(t *testing.T) {
	root := t.TempDir()
	agents := filepath.Join(root, "AGENTS.md")
	old := "intro\n\n" + startMark + "\nstale text\n" + endMark + "\n\noutro\n"
	if err := os.WriteFile(agents, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, root, Options{})
	if got, want := read(t, agents), "intro\n\n"+Guide()+"\noutro\n"; got != want {
		t.Errorf("AGENTS.md = %q\nwant %q", got, want)
	}
}

func TestNoAgentDocs(t *testing.T) {
	root := t.TempDir()
	run(t, root, Options{NoAgentDocs: true})
	if _, err := os.Stat(filepath.Join(root, ".stickypane")); err != nil {
		t.Errorf("the board should still be created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); !errors.Is(err, os.ErrNotExist) {
		t.Error("AGENTS.md should not be touched")
	}
}
```

- [ ] **Step 2: 테스트가 실패하는지 확인한다**

Run: `go test ./internal/initcmd/`
Expected: 컴파일 실패. `undefined: Run`, `undefined: Guide` 같은 오류가 나온다.

- [ ] **Step 3: 안내문과 환영 노트를 쓴다**

안내문은 표식을 포함해 30줄이다. 줄을 늘리지 않는다.

**`internal/initcmd/guide.md`**

```markdown
<!-- stickypane:start -->
## stickypane notes

The user keeps a note board open in a terminal pane next to you. It shows the
Markdown files in `.stickypane/`: one file is one note. Create a file to stick a
note, edit it to update the note, delete it to take the note down.

Jot freely: decisions, things to remember, progress. A one-line file is a
complete note. Add front matter only when a note needs a shape:

- Plain note: any Markdown. Optional keys: `title`, `pin: true`, and `color`
  (yellow, pink, blue, green, purple, orange).
- Board: `type: board`. Each `## Heading` is a column, each top-level `- item`
  below it is a card, and indented lines under a card are its details. Move a
  card by moving its lines under another heading.
- Checklist: `type: checklist`. Items are `- [ ]` and `- [x]` lines.
- Log: `type: log`. Append one line per entry at the end of the file.

    ---
    type: checklist
    title: Login API
    ---
    - [x] Add endpoint
    - [ ] Write tests

Keep notes short. When progress changes, update the matching note instead of
adding a new one. The user can edit notes from the board, so read a note again
before you change it. Notes are ordered by file name; prefix a number
(`10-plan.md`) to control the order.
<!-- stickypane:end -->
```

**`internal/initcmd/welcome.md`**

```markdown
---
title: Welcome to stickypane
pin: true
color: yellow
---
This board shows the Markdown files in `.stickypane/`. Each file is a note.

- **n** jots a note, **a** adds a board, checklist or log
- **enter** opens a note, **esc** closes it
- **p** pins, **c** changes color, **x** moves a note to `archive/`
- **?** lists every key

Your agent can stick notes here by writing files. Press **x** to take this
note down when you are done with it.
```

- [ ] **Step 4: 구현한다**

**`internal/initcmd/initcmd.go`**

```go
// Package initcmd sets a project up for stickypane: the notes folder, a
// welcome note, and a short guide for coding agents.
package initcmd

import (
	_ "embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/store"
)

//go:embed guide.md
var guide string

//go:embed welcome.md
var welcome string

// The guide lives between these markers so that a later run can replace it.
const (
	startMark = "<!-- stickypane:start -->"
	endMark   = "<!-- stickypane:end -->"
)

// agentFiles are the instruction files coding agents read.
var agentFiles = []string{"AGENTS.md", "CLAUDE.md"}

// Options adjusts what Run does.
type Options struct {
	NoAgentDocs bool // leave AGENTS.md and CLAUDE.md alone
}

// Guide returns the text Run adds to agent instruction files.
func Guide() string { return guide }

// Run prepares the project at root. It is safe to run again: an existing
// notes folder is left as it is, and the guide is replaced in place.
func Run(root string, opts Options, out io.Writer) error {
	dir := filepath.Join(root, store.DirName)
	switch _, err := os.Stat(dir); {
	case errors.Is(err, fs.ErrNotExist):
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, "welcome.md"), []byte(welcome), 0o644); err != nil {
			return err
		}
		fmt.Fprintf(out, "Created %s/ with a welcome note.\n", store.DirName)
	case err != nil:
		return err
	default:
		fmt.Fprintf(out, "%s/ already exists.\n", store.DirName)
	}

	if !opts.NoAgentDocs {
		var targets []string
		for _, name := range agentFiles {
			if _, err := os.Stat(filepath.Join(root, name)); err == nil {
				targets = append(targets, name)
			}
		}
		if len(targets) == 0 {
			targets = agentFiles[:1]
		}
		for _, name := range targets {
			path := filepath.Join(root, name)
			old, err := os.ReadFile(path)
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
			if err := os.WriteFile(path, []byte(inject(string(old), guide)), 0o644); err != nil {
				return err
			}
			fmt.Fprintf(out, "Added the stickypane guide to %s.\n", name)
		}
	}
	fmt.Fprintln(out, "Run `stickypane` in a pane next to your agent.")
	return nil
}

// inject puts the guide into content: in place of an earlier guide when the
// markers are there, otherwise at the end after a blank line.
func inject(content, guide string) string {
	block := strings.TrimRight(guide, "\n")
	start := strings.Index(content, startMark)
	end := strings.Index(content, endMark)
	if start >= 0 && end > start {
		return content[:start] + block + content[end+len(endMark):]
	}
	if content != "" {
		content = strings.TrimRight(content, "\n") + "\n\n"
	}
	return content + block + "\n"
}
```

- [ ] **Step 5: 테스트가 통과하는지 확인한다**

Run: `go test ./internal/initcmd/`
Expected: `ok  	github.com/LeeSwallow/stickypane/internal/initcmd`

- [ ] **Step 6: 커밋한다**

```bash
git add internal/initcmd
git commit -m "feat(initcmd): create the notes folder and add the agent guide"
```

---

### Task 12: 명령어 진입점, README, 배포 설정

`stickypane` 명령을 조립하고, 영어 README와 라이선스, GoReleaser·CI 설정을 추가한다. 이 작업은 파일만 만든다. GitHub 레포 생성과 릴리스는 Task 13에서 사용자 확인을 받고 한다.

**Files:**
- Create: `cmd/stickypane/main.go`
- Test: `cmd/stickypane/main_test.go`
- Create: `README.md`, `LICENSE`, `.goreleaser.yaml`, `.github/workflows/ci.yml`, `.github/workflows/release.yml`
- Modify: `go.mod`, `go.sum` (`go mod tidy`)

**Interfaces:**
- Consumes: `store.Resolve`, `store.Open`, `store.ErrNotFound`, `(*Store).Watch` (Task 6), `kinds.Default`, `kinds.Markdown`, `kinds.Theme` (Task 5), `app.New`, `(*Model).SetStatus`, `Model.OnBackground` (Task 8), `initcmd.Run`, `initcmd.Guide`, `initcmd.Options` (Task 11)
- Produces: 실행 파일 `stickypane`. 명령: `stickypane [path]`, `stickypane init [--no-agent-docs]`, `stickypane guide`, `stickypane version`, `stickypane help`. 종료 코드: 성공 0, 실행 오류 1, 사용법 오류 2.

- [ ] **Step 1: 실패하는 테스트를 쓴다**

**`cmd/stickypane/main_test.go`**

```go
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/initcmd"
)

func exec(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestGuidePrintsTheAgentGuide(t *testing.T) {
	code, out, _ := exec(t, "guide")
	if code != 0 || out != initcmd.Guide() {
		t.Errorf("code = %d, out = %q", code, out)
	}
}

func TestVersionAndHelp(t *testing.T) {
	for _, arg := range []string{"version", "--version", "-v"} {
		if code, out, _ := exec(t, arg); code != 0 || out != "stickypane dev\n" {
			t.Errorf("%s: code = %d, out = %q", arg, code, out)
		}
	}
	for _, arg := range []string{"help", "--help", "-h"} {
		if code, out, _ := exec(t, arg); code != 0 || !strings.Contains(out, "stickypane init") {
			t.Errorf("%s: code = %d, out = %q", arg, code, out)
		}
	}
}

func TestUsageErrors(t *testing.T) {
	for _, args := range [][]string{{"--nope"}, {"a", "b"}, {"init", "--nope"}} {
		if code, _, errOut := exec(t, args...); code != 2 || errOut == "" {
			t.Errorf("%v: code = %d, stderr = %q; want 2 and a message", args, code, errOut)
		}
	}
}

func TestInitSetsUpTheCurrentDirectory(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	code, out, errOut := exec(t, "init", "--no-agent-docs")
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, errOut)
	}
	if _, err := os.Stat(filepath.Join(root, ".stickypane", "welcome.md")); err != nil {
		t.Errorf("welcome note is missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); err == nil {
		t.Error("--no-agent-docs must not create AGENTS.md")
	}
	if !strings.Contains(out, ".stickypane") {
		t.Errorf("out = %q", out)
	}
}

func TestBoardWithoutNotesFolderExplainsInit(t *testing.T) {
	code, _, errOut := exec(t, t.TempDir())
	if code != 1 || !strings.Contains(errOut, "stickypane init") {
		t.Errorf("code = %d, stderr = %q", code, errOut)
	}
}
```

- [ ] **Step 2: 테스트가 실패하는지 확인한다**

Run: `go test ./cmd/stickypane/`
Expected: 컴파일 실패. `undefined: run`이 나온다.

- [ ] **Step 3: 진입점을 구현한다**

**`cmd/stickypane/main.go`**

```go
// Command stickypane shows the notes in .stickypane/ as a board.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/app"
	"github.com/LeeSwallow/stickypane/internal/initcmd"
	"github.com/LeeSwallow/stickypane/internal/kinds"
	"github.com/LeeSwallow/stickypane/internal/store"
)

// version is set by the release build.
var version = "dev"

const usage = `stickypane - a sticky-note board for you and your coding agent

Usage:
  stickypane [path]      open the board of the project at path (default: here)
  stickypane init        create .stickypane/ and add the agent guide to
                         AGENTS.md or CLAUDE.md (--no-agent-docs skips that)
  stickypane guide       print the agent guide
  stickypane version     print the version
`

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

// run executes the command line and returns the exit code: 0 on success,
// 1 when something failed, 2 when the arguments make no sense.
func run(args []string, stdout, stderr io.Writer) int {
	start := "."
	if len(args) > 0 {
		switch args[0] {
		case "init":
			flags := flag.NewFlagSet("stickypane init", flag.ContinueOnError)
			flags.SetOutput(stderr)
			noDocs := flags.Bool("no-agent-docs", false, "leave AGENTS.md and CLAUDE.md alone")
			if err := flags.Parse(args[1:]); err != nil {
				return 2
			}
			if err := initcmd.Run(".", initcmd.Options{NoAgentDocs: *noDocs}, stdout); err != nil {
				fmt.Fprintln(stderr, "stickypane:", err)
				return 1
			}
			return 0
		case "guide":
			fmt.Fprint(stdout, initcmd.Guide())
			return 0
		case "version", "--version", "-v":
			fmt.Fprintln(stdout, "stickypane", version)
			return 0
		case "help", "--help", "-h":
			fmt.Fprint(stdout, usage)
			return 0
		}
		if strings.HasPrefix(args[0], "-") || len(args) > 1 {
			fmt.Fprint(stderr, usage)
			return 2
		}
		start = args[0]
	}

	dir, err := store.Resolve(start)
	if errors.Is(err, store.ErrNotFound) {
		fmt.Fprintln(stderr, "No .stickypane directory found. Run `stickypane init` in your project first.")
		return 1
	}
	if err != nil {
		fmt.Fprintln(stderr, "stickypane:", err)
		return 1
	}
	if err := board(dir); err != nil {
		fmt.Fprintln(stderr, "stickypane:", err)
		return 1
	}
	return 0
}

// board runs the terminal UI until the user quits. Markdown starts with dark
// colors and switches when the terminal reports a light background.
func board(dir string) error {
	st := store.Open(dir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	watch, watchErr := st.Watch(ctx)

	theme := &kinds.Theme{Dark: true}
	m := app.New(st, kinds.Default(kinds.Markdown(theme)), watch)
	m.OnBackground = func(dark bool) { theme.Dark = dark }
	if watchErr != nil {
		m.SetStatus("File watching is unavailable. Press r to refresh.")
	}
	_, err := tea.NewProgram(m).Run()
	return err
}
```

- [ ] **Step 4: 의존성을 정리하고 전체를 확인한다**

```bash
go mod tidy
go vet ./...
go test -race ./...
go build -o stickypane ./cmd/stickypane
./stickypane version
```

Expected: `go vet`는 출력이 없고, 모든 패키지 테스트가 `ok`이고, 마지막 줄이 `stickypane dev`다. `go.mod`의 직접 의존성은 Global Constraints의 여섯 개와 버전이 같아야 한다.

- [ ] **Step 5: README와 라이선스를 쓴다**

**`README.md`**

````markdown
# stickypane

A sticky-note board in your terminal, for you and your coding agent.

```
╔ 📌 Auth work ════════════════════════════════════════════════════════════════╗
║ To do (1)                Doing (1)                Done (1)                   ║
║ payments                 login API                schema                     ║
╚══════════════════════════════════════════════════════════════════════════════╝
╭ Login API ───────────────────────────╮╭ Auth design ─────────────────────────╮
│ ▓▓▓▓▓▓▓▓▓▓▓▓▓░░░░░░░ 2/3             ││ Decisions                            │
│ ☐ Write tests                        ││                                      │
╰──────────────────────────────────────╯│ Tokens live in a session cookie.     │
╭──────────────────────────────────────╮╰──────────────────────────────────────╯
│ check env before deploy              │╭ ● Work log ──────────────────────────╮
╰──────────────────────────────────────╯│ 14:02 tests passed                   │
                                        │ 14:10 started on review feedback     │
                                        ╰──────────────────────────────────────╯
n jot  enter open  a add  ? help  q quit
```

Your agent jots things down as plain Markdown files. stickypane shows them as
notes in a pane next to it: plain notes, kanban boards, checklists and logs.
Move a card or tick a box on the board and the change lands in the same file,
so the agent sees it too.

- **Nothing to configure.** No config file, no server, no MCP, no hooks.
- **Any agent.** If it can edit files, it can stick notes: Claude Code, Codex
  and others.
- **Any terminal.** A plain window, tmux, WezTerm, Zellij. stickypane is a
  visualization layer, not a terminal plugin.

## Install

```sh
brew install --cask LeeSwallow/tap/stickypane
# or
go install github.com/LeeSwallow/stickypane/cmd/stickypane@latest
```

macOS and Linux are supported.

## Quick start

```sh
cd your-project
stickypane init    # creates .stickypane/ and tells your agent about it
stickypane         # opens the board
```

`init` adds a short guide to `AGENTS.md` (or `CLAUDE.md` if you have one), so
your agent knows the board exists. Then ask it for anything: "keep a checklist
of this refactor on the board", "jot down what we decided".

Keep the board next to your agent:

```sh
tmux split-window -h stickypane             # tmux pane
tmux display-popup -E stickypane            # tmux popup
wezterm cli split-pane --right -- stickypane
```

## Notes are files

One Markdown file in `.stickypane/` is one note. Create a file to stick a
note, edit it to update the note, delete it to take the note down. A one-line
file is a complete note:

```markdown
check env before deploy
```

Front matter gives a note a shape. Every key is optional.

| Key     | Values                                           |
| ------- | ------------------------------------------------ |
| `type`  | `note` (default), `board`, `checklist`, `log`    |
| `title` | shown in the note's border                       |
| `color` | `yellow`, `pink`, `blue`, `green`, `purple`, `orange` |
| `pin`   | `true` keeps the note at the top                 |

**Board.** Each `## Heading` is a column and each top-level list item is a
card. Indented lines under a card are its details.

```markdown
---
type: board
title: Auth work
---
## To do
- payments
## Doing
- login API
  - refresh tokens come later
## Done
- schema
```

**Checklist.** `- [ ]` and `- [x]` lines, shown with a progress bar.

**Log.** One entry per line. The board shows the latest lines and follows
along as the file grows.

Notes are ordered by file name, pinned ones first. Prefix a number
(`10-plan.md`) to control the order. A file that does not fit its shape is
still shown, never hidden.

## Keys

| Board                 |                    | Open note |                         |
| --------------------- | ------------------ | --------- | ----------------------- |
| arrows, `h j k l`     | move focus         | `esc`     | close                   |
| `enter`               | open note          | `j` `k`   | move or scroll          |
| `n`                   | jot a note         | `G`       | jump to the end         |
| `a`                   | add by shape       | `H` `L`   | move a card sideways    |
| `p` / `c` / `R`       | pin, color, rename | `J` `K`   | reorder a card          |
| `x` / `D`             | archive, delete    | `space`   | tick a checklist item   |
| `e`                   | edit in `$EDITOR`  | `n`       | new card or item        |
| `?`                   | help               | `e`       | edit in `$EDITOR`       |

## Edits are safe

The board never overwrites a file with what it last saw. An edit is an intent
("move this card to that column") applied to the file as it is at that moment,
then written in one atomic step. If your agent changed that card in the
meantime, nothing is written and the board tells you.

## Adding a shape

A shape is one package under `internal/widget/` that implements the
`widget.Widget` interface, plus one line in `internal/kinds/kinds.go`.

## License

MIT
````

**`LICENSE`**

```text
MIT License

Copyright (c) 2026 LeeSwallow

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

- [ ] **Step 6: 배포와 CI 설정을 쓴다**

GoReleaser v2 형식이다. Homebrew는 `brews`가 폐기되어 `homebrew_casks`를 쓴다. 서명하지 않은 실행 파일이라 macOS에서 설치 후 격리 속성을 지우는 훅을 둔다.

**`.goreleaser.yaml`**

```yaml
version: 2

project_name: stickypane

before:
  hooks:
    - go mod tidy

builds:
  - main: ./cmd/stickypane
    env:
      - CGO_ENABLED=0
    goos:
      - darwin
      - linux
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w -X main.version={{ .Version }}

archives:
  - formats:
      - tar.gz

checksum:
  name_template: checksums.txt

homebrew_casks:
  - repository:
      owner: LeeSwallow
      name: homebrew-tap
      token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"
    homepage: https://github.com/LeeSwallow/stickypane
    description: A sticky-note board in your terminal, for you and your coding agent
    license: MIT
    hooks:
      post:
        install: |
          if OS.mac?
            system_command "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "#{staged_path}/stickypane"]
          end
```

**`.github/workflows/ci.yml`**

```yaml
name: ci

on:
  push:
    branches: [main]
  pull_request:

jobs:
  test:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - run: go vet ./...
      - run: go test -race ./...
```

**`.github/workflows/release.yml`**

```yaml
name: release

on:
  push:
    tags: ["v*"]

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: goreleaser/goreleaser-action@v6
        with:
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          HOMEBREW_TAP_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }}
```

- [ ] **Step 7: 커밋한다**

```bash
git add cmd README.md LICENSE .goreleaser.yaml .github go.mod go.sum
git commit -m "feat: add the stickypane command, README and release setup"
```

---

### Task 13: 직접 써 보고 릴리스한다

자동 테스트가 다루지 못하는 것을 사람이 확인한다. 실제 터미널에서의 모습, 그리고 에이전트가 안내문만 읽고 노트를 다루는지다. 릴리스 단계는 바깥에 공개되는 작업이므로 **각 단계마다 사용자 확인을 받은 뒤에만** 실행한다.

**Files:**
- Modify: 확인 중 발견한 문제가 있으면 해당 파일. 문제마다 실패하는 테스트를 먼저 추가한다.

**Interfaces:**
- Consumes: Task 12의 실행 파일
- Produces: 공개된 `v0.1.0` 릴리스

- [ ] **Step 1: 실제 터미널에서 화면을 확인한다**

```bash
go build -o stickypane ./cmd/stickypane
mkdir -p /tmp/sp-demo && cd /tmp/sp-demo && /Users/min/Projects/stickypane/stickypane init
/Users/min/Projects/stickypane/stickypane
```

확인할 것:
- 환영 노트가 고정되어 맨 위에 보이고, `n`으로 끄적인 노트가 바로 붙는다.
- `a`로 board, checklist, log를 하나씩 만들고, 열어서 카드 이동·체크·스크롤이 된다.
- 터미널 폭을 40칸 아래로 줄여도 노트가 한 줄에 하나씩 쌓이고 테두리가 어긋나지 않는다. 한글 제목과 본문으로도 확인한다.
- 다른 창에서 `.stickypane/` 안의 파일을 고치면 1초 안에 화면이 바뀌고 `●`가 붙는다.
- `NO_COLOR=1 stickypane`에서 색 없이도 포커스한 노트가 이중 테두리로 구분된다.
- `tmux split-window -h stickypane`과 `tmux display-popup -E stickypane`에서 똑같이 동작한다. WezTerm에서도 `wezterm cli split-pane --right -- stickypane`으로 확인한다.

- [ ] **Step 2: 에이전트가 안내문만으로 노트를 다루는지 확인한다**

`/tmp/sp-demo`에서 Claude Code와 Codex를 각각 열고, 보드 사용법을 따로 설명하지 않은 채 다음을 시킨다. 옆 pane의 보드에 결과가 나타나야 한다.

1. "Jot down on the board that we decided to use session cookies."
2. "Make a checklist on the board for adding a login endpoint, and tick off the first item."
3. "Make a kanban board for this work and move one card to Doing."
4. "Start a work log on the board and add two entries."
5. 보드에서 체크 하나를 직접 전환한 뒤: "What's left on the checklist?" (에이전트가 파일을 다시 읽고 바뀐 상태를 답해야 한다.)

에이전트가 형식을 틀리면 `internal/initcmd/guide.md`의 문구를 고친다. 30줄 상한은 유지한다.

- [ ] **Step 3: (사용자 확인 후) GitHub 레포를 만들고 올린다**

```bash
gh repo create LeeSwallow/stickypane --public --source . --description "A sticky-note board in your terminal, for you and your coding agent" --push
```

CI가 macOS와 Linux에서 통과하는지 확인한다: `gh run watch`

원격 레포가 생겼으니 배포 설정도 검사한다.

```bash
HOMEBREW_TAP_TOKEN=x go run github.com/goreleaser/goreleaser/v2@latest check
```

Expected: `1 configuration file(s) validated`

- [ ] **Step 4: (사용자 확인 후) Homebrew tap을 준비한다**

```bash
gh repo create LeeSwallow/homebrew-tap --public --description "Homebrew tap"
```

tap 레포에 쓸 수 있는 토큰(fine-grained, `homebrew-tap`의 Contents 읽기·쓰기)을 사용자가 만들어 `stickypane` 레포의 시크릿 `HOMEBREW_TAP_TOKEN`으로 등록한다: `gh secret set HOMEBREW_TAP_TOKEN --repo LeeSwallow/stickypane`

- [ ] **Step 5: (사용자 확인 후) 릴리스한다**

```bash
git tag v0.1.0
git push origin v0.1.0
gh run watch
```

릴리스가 끝나면 설치 경로 두 가지를 확인한다.

```bash
brew install --cask LeeSwallow/tap/stickypane && stickypane version
go install github.com/LeeSwallow/stickypane/cmd/stickypane@v0.1.0
```

Expected: `stickypane 0.1.0`
