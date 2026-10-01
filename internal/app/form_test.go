package app

import (
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

const formFile = "---\ntype: form\ntitle: Deploy\nopen: true\n---\n## Where?\n- ( ) staging\n- ( ) production\n\n## Note\n> \n\n[ Go ] [ Cancel ]\n"

func TestAFormIsAnsweredOnTheMainScreen(t *testing.T) {
	m, dir := newModel(t, map[string]string{"f.md": formFile})
	press(m, "j", "enter")
	if got := readFile(t, dir, "f.md"); !strings.Contains(got, "- (x) production") || m.mode != modeBoard {
		t.Fatalf("enter should choose the option, not zoom (mode %v):\n%s", m.mode, got)
	}

	press(m, "j", "enter")
	if m.mode != modeInput || !strings.Contains(screen(m), "Note:") {
		t.Fatalf("enter on a field should ask for its text:\n%s", screen(m))
	}
	typeText(m, "after lunch")
	press(m, "enter")
	if got := readFile(t, dir, "f.md"); !strings.Contains(got, "> after lunch\n") {
		t.Fatalf("the answer should be in the file:\n%s", got)
	}

	press(m, "j", "enter")
	d := doc.Parse([]byte(readFile(t, dir, "f.md")))
	if v, _ := d.Get("submitted"); v != "Go" {
		t.Fatalf("enter on a button should submit:\n%s", d.Bytes())
	}
	if s := screen(m); !strings.Contains(s, "✓ Go") {
		t.Errorf("the screen should show what was sent:\n%s", s)
	}
}

func TestAFieldCanBeEditedAndCleared(t *testing.T) {
	filled := strings.Replace(formFile, "> \n", "> after lunch\n", 1)
	m, dir := newModel(t, map[string]string{"f.md": filled})
	press(m, "j", "j", "enter")
	if s := screen(m); !strings.Contains(s, "Note: after lunch") {
		t.Fatalf("the prompt should start with the field's text:\n%s", s)
	}
	for range "after lunch" {
		m.Update(key("backspace"))
	}
	press(m, "enter")
	if got := readFile(t, dir, "f.md"); !strings.Contains(got, "## Note\n> \n") {
		t.Errorf("an empty line clears the field:\n%s", got)
	}
}

func TestZZoomsANoteThatTakesEnter(t *testing.T) {
	m, _ := newModel(t, map[string]string{"f.md": formFile})
	if s := screen(m); !strings.Contains(s, "z zoom") || strings.Contains(s, "enter zoom") {
		t.Errorf("a form uses enter itself, so the footer should offer z:\n%s", s)
	}
	press(m, "z")
	if m.mode != modeZoom {
		t.Fatalf("z should zoom, mode = %v", m.mode)
	}
	press(m, "esc")

	n, _ := newModel(t, map[string]string{"c.md": checklistFile})
	press(n, "z")
	if n.mode != modeZoom {
		t.Errorf("z zooms any open note, mode = %v", n.mode)
	}
	if s := screen(n); !strings.Contains(s, "esc") {
		t.Errorf("zoomed screen:\n%s", s)
	}
}

func TestZDoesNotZoomAClosedNote(t *testing.T) {
	closed := strings.Replace(formFile, "open: true\n", "", 1)
	m, _ := newModel(t, map[string]string{"f.md": closed})
	// A note that appears at startup without "open" is folded away.
	press(m, "z")
	if m.mode != modeBoard {
		t.Errorf("z on a closed note should do nothing, mode = %v", m.mode)
	}
}

func TestAFrontMatterChangeReachesTheWidget(t *testing.T) {
	m, dir := newModel(t, map[string]string{"f.md": formFile})
	pressed := strings.Replace(formFile, "open: true\n", "open: true\nsubmitted: Go\n", 1)
	writeFile(t, dir, "f.md", pressed)
	press(m, "r")
	if s := screen(m); !strings.Contains(s, "✓ Go") {
		t.Fatalf("a submission written to the file should show:\n%s", s)
	}
	writeFile(t, dir, "f.md", formFile)
	press(m, "r")
	if s := screen(m); strings.Contains(s, "✓ Go") {
		t.Errorf("a form asked again must not look submitted:\n%s", s)
	}
}

func TestALongQuestionLeavesRoomToType(t *testing.T) {
	long := strings.Replace(formFile, "## Note\n", strings.Repeat("a very long question ", 8)+"\n", 1)
	m, _ := newModel(t, map[string]string{"f.md": long})
	press(m, "j", "j", "enter")
	typeText(m, "typed")
	lines := strings.Split(screen(m), "\n")
	if last := lines[len(lines)-1]; !strings.Contains(last, "typed") {
		t.Errorf("the text being typed must stay visible: %q", last)
	}
}
