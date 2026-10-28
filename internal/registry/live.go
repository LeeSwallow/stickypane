package registry

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/LeeSwallow/stickypane/internal/when"
)

// touchEvery is how long a project's touch marker silences further touches.
const touchEvery = time.Minute

// Live records that this process has started working on project, as kind
// ("tui", "bridge", "mcp", "watch"): a session.start event and a file
// run/<pid>.json that says so while the process lives. The returned end
// records session.end and removes the file; call it on a normal exit. A
// process that dies leaves its file behind, which the app tells by the
// process no longer being alive.
func Live(project, kind string) (end func()) {
	if !Enabled() {
		return func() {}
	}
	pid := os.Getpid()
	file := filepath.Join(Home(), "run", strconv.Itoa(pid)+".json")
	if os.MkdirAll(filepath.Dir(file), 0o755) == nil {
		b, err := json.Marshal(map[string]any{
			"pid":     pid,
			"project": project,
			"kind":    kind,
			"started": time.Now().Format(when.Moment),
		})
		if err == nil {
			writeAtomic(file, b)
		}
	}
	Append(Event{Type: "session.start", Project: project, PID: pid, Kind: kind})
	return func() {
		Append(Event{Type: "session.end", Project: project, PID: pid})
		os.Remove(file)
	}
}

// Touch records that a command cmd changed something in project. The same
// project and command are recorded at most once a minute: the app wants to
// know a board is in use, not each keystroke of it. The minute is kept in
// the modification time of run/touch-<hash of project and command>, which
// is shared by every process.
func Touch(project, cmd string) {
	if !Enabled() {
		return
	}
	sum := sha1.Sum([]byte(project + "\x00" + cmd))
	marker := filepath.Join(Home(), "run", "touch-"+hex.EncodeToString(sum[:])[:12])
	if fi, err := os.Stat(marker); err == nil && time.Since(fi.ModTime()) < touchEvery {
		return
	}
	if os.MkdirAll(filepath.Dir(marker), 0o755) != nil {
		return
	}
	// Writing the file sets its modification time to now.
	if os.WriteFile(marker, nil, 0o644) != nil {
		return
	}
	Append(Event{Type: "project.touch", Project: project, Cmd: cmd})
}

// writeAtomic writes b to path through a temporary file in the same folder
// and a rename, so that the app, which reads run/, never sees half a file.
func writeAtomic(path string, b []byte) {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return
	}
	_, werr := tmp.Write(b)
	cerr := tmp.Close()
	if werr != nil || cerr != nil || os.Rename(tmp.Name(), path) != nil {
		os.Remove(tmp.Name())
		return
	}
	os.Chmod(path, 0o644)
}
