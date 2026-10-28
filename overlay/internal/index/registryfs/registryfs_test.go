package registryfs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LeeSwallow/stickypane/overlay/internal/index/model"
	"github.com/LeeSwallow/stickypane/overlay/internal/index/ports"
)

// golden is the home folder stickypane's own tests write and pin
// (internal/adapter/registry/testdata/home in the main module). Reading it here is
// what keeps the two modules' idea of the records the same.
const golden = "../../../../internal/adapter/registry/testdata/home"

func TestReadsTheGoldenHome(t *testing.T) {
	h := Home(golden)
	files, err := h.EventFiles()
	if err != nil || len(files) != 1 || files[0] != "2026-10-28.jsonl" {
		t.Fatalf("files=%v err=%v", files, err)
	}
	data, size, err := h.ReadFrom(files[0], 0)
	if err != nil || int64(len(data)) != size {
		t.Fatalf("read %d of %d: %v", len(data), size, err)
	}
	lines, used := model.Lines(data)
	if used != len(data) {
		t.Fatal("the golden file does not end in a newline")
	}
	var got []model.Event
	for _, l := range lines {
		e, at, ok := model.ParseEvent(l)
		if !ok {
			t.Fatalf("not an event: %s", l)
		}
		if at.Format("2006-01-02T15:04:05Z07:00") != "2026-10-28T14:02:11+09:00" {
			t.Errorf("at = %v", at)
		}
		got = append(got, e)
	}
	want := []model.Event{
		{V: 1, Type: model.ProjectInit, Project: "/work/demo"},
		{V: 1, Type: model.SessionStart, Project: "/work/demo", PID: 4242, Kind: "tui"},
		{V: 1, Type: model.ProjectTouch, Project: "/work/demo", Cmd: "card"},
		{V: 1, Type: model.SessionStart, Project: "/work/demo", PID: 4343, Kind: "bridge"},
		{V: 1, Type: model.SessionEnd, Project: "/work/demo", PID: 4343},
	}
	if len(got) != len(want) {
		t.Fatalf("events = %+v", got)
	}
	for i := range want {
		want[i].At = got[i].At
		if got[i] != want[i] {
			t.Errorf("event %d = %+v, want %+v", i, got[i], want[i])
		}
	}

	runs, err := h.Runs()
	if err != nil || len(runs) != 1 || runs[0].Name != "4242.json" || !model.IsRunName(runs[0].Name) {
		t.Fatalf("runs=%v err=%v", runs, err)
	}
	r, started, ok := model.ParseRun(runs[0].Data)
	if !ok || r != (model.Run{Kind: "tui", PID: 4242, Project: "/work/demo", Started: "2026-10-28T14:02:11+09:00"}) || started.IsZero() {
		t.Fatalf("run = %+v ok=%v", r, ok)
	}
}

func TestReadFrom(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, "events"), 0o755)
	os.WriteFile(filepath.Join(home, "events", "d.jsonl"), []byte("abcdef"), 0o644)
	os.WriteFile(filepath.Join(home, "events", "notes.txt"), nil, 0o644)
	h := Home(home)
	if fs, _ := h.EventFiles(); len(fs) != 1 {
		t.Fatalf("files = %v", fs)
	}
	if b, n, err := h.ReadFrom("d.jsonl", 2); string(b) != "cdef" || n != 6 || err != nil {
		t.Fatalf("%q %d %v", b, n, err)
	}
	if b, n, err := h.ReadFrom("d.jsonl", 10); b != nil || n != 6 || err != nil {
		t.Fatalf("past the end: %q %d %v", b, n, err)
	}
	if _, _, err := h.ReadFrom("gone.jsonl", 0); err != ports.ErrGone {
		t.Fatalf("err = %v", err)
	}
	if fs, err := Home(filepath.Join(home, "none")).EventFiles(); fs != nil || err != nil {
		t.Fatalf("a home without events/: %v %v", fs, err)
	}
}
