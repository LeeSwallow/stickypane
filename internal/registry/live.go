package registry

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/LeeSwallow/stickypane/internal/atomicfile"
	"github.com/LeeSwallow/stickypane/internal/when"
)

// touchEvery is how long a project's touch marker silences further touches.
const touchEvery = time.Minute

// Live records, through Default, that this process works on project.
func Live(project, kind string) (end func()) { return Default().Live(project, kind) }

// Touch records, through Default, that a command changed project.
func Touch(project, cmd string) { Default().Touch(project, cmd) }

// Live records that the process w.PID has started working on project, as
// kind ("tui", "bridge", "mcp", "watch"): a session.start event and a file
// run/<pid>.json that says so while the process lives. The returned end
// records session.end and removes the file; call it on a normal exit. A
// process that dies leaves its file behind, which Prune removes once the
// process is no longer alive.
func (w Writer) Live(project, kind string) (end func()) {
	if !exists(w.Home) {
		return func() {}
	}
	file := filepath.Join(w.Home, "run", strconv.Itoa(w.PID)+".json")
	if os.MkdirAll(filepath.Dir(file), 0o755) == nil {
		b, err := json.Marshal(Run{
			Kind:    kind,
			PID:     w.PID,
			Project: project,
			Started: w.now().Format(when.Moment),
		})
		if err == nil {
			// The app reads run/: it never sees half a file.
			_ = atomicfile.WriteFile(file, b, 0o644)
		}
	}
	w.Append(Event{Type: "session.start", Project: project, PID: w.PID, Kind: kind})
	return func() {
		w.Append(Event{Type: "session.end", Project: project, PID: w.PID})
		os.Remove(file)
	}
}

// Touch records that a command cmd changed something in project. The same
// project and command are recorded at most once a minute: the app wants to
// know a board is in use, not each keystroke of it. The minute is kept in
// the modification time of run/touch-<hash of project and command>, which
// is shared by every process and set from w's clock.
func (w Writer) Touch(project, cmd string) {
	if !exists(w.Home) {
		return
	}
	now := w.now()
	sum := sha1.Sum([]byte(project + "\x00" + cmd))
	marker := filepath.Join(w.Home, "run", touchPrefix+hex.EncodeToString(sum[:])[:12])
	if fi, err := os.Stat(marker); err == nil && now.Sub(fi.ModTime()) < touchEvery {
		return
	}
	if os.MkdirAll(filepath.Dir(marker), 0o755) != nil {
		return
	}
	if os.WriteFile(marker, nil, 0o644) != nil || os.Chtimes(marker, now, now) != nil {
		return
	}
	w.Append(Event{Type: "project.touch", Project: project, Cmd: cmd})
}

// touchPrefix begins the name of a touch marker in run/.
const touchPrefix = "touch-"
