package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadTheGoldenHome(t *testing.T) {
	runs, events, err := Read(golden)
	if err != nil {
		t.Fatal(err)
	}
	want := Run{Kind: "tui", PID: 4242, Project: "/work/demo", Started: "2026-10-28T14:02:11+09:00"}
	if len(runs) != 1 || runs[0] != want {
		t.Fatalf("runs = %+v", runs)
	}
	types := []string{"project.init", "session.start", "project.touch", "session.start", "session.end"}
	if len(events) != len(types) {
		t.Fatalf("events = %+v", events)
	}
	for i, ty := range types {
		if events[i].Type != ty || events[i].V != 1 || events[i].Project != "/work/demo" {
			t.Fatalf("event %d = %+v, want %s", i, events[i], ty)
		}
	}
	if events[2].Cmd != "card" || events[3].PID != 4343 || events[3].Kind != "bridge" {
		t.Fatalf("events = %+v", events)
	}
}

func TestReadWithoutTheHomeFolder(t *testing.T) {
	runs, events, err := Read(filepath.Join(t.TempDir(), "absent"))
	if err != nil || runs != nil || events != nil {
		t.Fatalf("Read = %v, %v, %v", runs, events, err)
	}
}

func TestReadSkipsWhatItCannotRead(t *testing.T) {
	home := t.TempDir()
	put(t, home, "run/touch-aaaaaaaaaaaa", fixed)
	put(t, home, "run/.tmp-1", fixed)
	put(t, home, "run/7.json", fixed) // half a file: "x"
	os.WriteFile(filepath.Join(home, "run", "8.json"), []byte(`{"kind":"mcp","pid":8,"project":"/q","started":"2026-10-28T14:02:11+09:00"}`), 0o644)
	os.MkdirAll(filepath.Join(home, "events"), 0o755)
	os.WriteFile(filepath.Join(home, "events", "2026-10-27.jsonl"), []byte("{\"v\":1,\"type\":\"project.init\",\"project\":\"/a\"}\nnot json\n"), 0o644)
	os.WriteFile(filepath.Join(home, "events", "2026-10-28.jsonl"), []byte("{\"v\":1,\"type\":\"project.init\",\"project\":\"/b\"}\n{\"v\":1,\"type\":\"proj"), 0o644)
	runs, events, err := Read(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || runs[0].PID != 8 {
		t.Fatalf("runs = %+v", runs)
	}
	if len(events) != 2 || events[0].Project != "/a" || events[1].Project != "/b" {
		t.Fatalf("events = %+v", events)
	}
}
