package app

import (
	"fmt"
	"strings"
	"testing"
)

const deployLog = "$ sh deploy.sh   (2026-10-03 14:02:00)\nhello\n[exit 0 · 1.02s]\n"

// A script and its log are one pane: the script with its Run button, and
// under it the log of its last run, with how it ended.
func TestAScriptShowsItsLogInItsOwnPane(t *testing.T) {
	m, dir := newModel(t, map[string]string{"deploy.sh": "echo hello\n", "deploy.log": deployLog})
	writeView(t, dir, `{"notes":{"deploy.sh":{"open":true}}}`)
	m.Update(changedMsg{})
	s := screen(m)
	bar := strings.SplitN(s, "\n", 2)[0]
	if strings.Contains(bar, "≣") {
		t.Errorf("the log is not a note of its own next to its script:\n%s", bar)
	}
	for _, want := range []string{"▶ Run", "echo hello", "log", "hello", "exit 0"} {
		if !strings.Contains(s, want) {
			t.Errorf("the pane should show %q:\n%s", want, s)
		}
	}
}

// A log with no script of the same name stays a note of its own.
func TestALogWithoutItsScriptIsANoteOfItsOwn(t *testing.T) {
	m, _ := newModel(t, map[string]string{"other.log": "x\n", "deploy.sh": "echo\n"})
	if bar := strings.SplitN(screen(m), "\n", 2)[0]; !strings.Contains(bar, "≣ other") {
		t.Errorf("other.log is its own note:\n%s", bar)
	}
}

// The pane shows the end of a long log; zoomed in, all of it.
func TestAZoomedScriptShowsItsWholeLog(t *testing.T) {
	var log strings.Builder
	for i := 1; i <= 40; i++ {
		fmt.Fprintf(&log, "output %02d\n", i)
	}
	m, dir := newModel(t, map[string]string{"deploy.sh": "echo\n", "deploy.log": log.String()})
	writeView(t, dir, `{"notes":{"deploy.sh":{"open":true}}}`)
	m.Update(changedMsg{})
	if s := screen(m); !strings.Contains(s, "output 40") || strings.Contains(s, "output 01") {
		t.Errorf("the pane shows the end of the log:\n%s", s)
	}
	press(m, "z")
	m.zoomScroll = 0
	m.layoutZoom()
	if s := screen(m); !strings.Contains(s, "output 01") {
		t.Errorf("zoomed in, the whole log is there:\n%s", s)
	}
}

// A form with a log of the same name shows it under the form: what the
// agent did with the answers.
func TestAFormShowsWhatTheAgentDidWithIt(t *testing.T) {
	form := "---\ntype: form\ntitle: Deploy now?\nopen: true\n---\n- ( ) staging\n- ( ) production\n"
	m, _ := newModel(t, map[string]string{"deploy.md": form, "deploy.log": "14:05 deployed to production\n"})
	s := screen(m)
	if bar := strings.SplitN(s, "\n", 2)[0]; strings.Contains(bar, "≣") {
		t.Errorf("the log belongs to the form, not beside it:\n%s", bar)
	}
	if !strings.Contains(s, "output") || !strings.Contains(s, "deployed to production") {
		t.Errorf("the form's pane should show its output:\n%s", s)
	}
}

// A .http note shows, under its requests, the response of the one the
// cursor is on.
func TestARequestsNoteShowsTheResponseOfThePickedRequest(t *testing.T) {
	file := "### Health\nGET http://x/health\n\n### Chat\nGET ws://x/chat\n"
	log := "### Health\n$ curl -sS 'http://x/health'\n{\"ok\": true}\n[200 OK · 3ms · Health]\n\n" +
		"### Chat\n$ websocat 'ws://x/chat'\n← 14:02:01.120  {\"type\":\"welcome\"}\n[101 Switching Protocols · 9ms · Chat · 0↑ 1↓]\n"
	m, dir := newModel(t, map[string]string{"svc.http": file, "svc.log": log})
	writeView(t, dir, `{"notes":{"svc.http":{"open":true}}}`)
	m.Update(changedMsg{})
	s := screen(m)
	if !strings.Contains(s, `"ok": true`) || !strings.Contains(s, "200 OK · 3ms · Health") || strings.Contains(s, "welcome") {
		t.Errorf("the first request's response:\n%s", s)
	}
	press(m, "j")
	s = screen(m)
	if !strings.Contains(s, "welcome") || !strings.Contains(s, "101 Switching Protocols") || strings.Contains(s, `"ok": true`) {
		t.Errorf("the second request's response:\n%s", s)
	}
}
