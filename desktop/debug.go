package main

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/desktop/internal/shell"
)

// The hooks for checking the built app without pressing anything: each
// is an environment variable, and unset each does nothing. They stay in
// the release build, which is the build they check (docs/desktop/06).
const (
	// STICKYPANE_APP_INSTANCE=<name> gives a copy run for a check a
	// single instance of its own (with its own STICKYPANE_HOME), so that
	// it does not hand its launch to an app already running.
	envInstance = "STICKYPANE_APP_INSTANCE"
	// STICKYPANE_APP_OPEN=settings[:section] opens the settings window at
	// launch, for the last board opened (Shell.OpenAtLaunch).
	envOpen = "STICKYPANE_APP_OPEN"
	// STICKYPANE_APP_TEAR=<note>[,<note>...] with --project=<path> tears
	// those notes off the project's board a moment after launch
	// (debugTear).
	envTear = "STICKYPANE_APP_TEAR"
)

// tearDelay is how long after launch envTear tears its notes off: time
// for the board's window and its bridge to come up.
const tearDelay = 3 * time.Second

// instanceID is the single instance's name: the app's, or the app's and
// the check's (envInstance).
func instanceID(name string) string {
	const id = "io.github.leeswallow.stickypane"
	if name == "" {
		return id
	}
	return id + "." + name
}

// debugTear is a debug hook, for checking tear-off in the built app
// without a mouse: STICKYPANE_APP_TEAR=<note> with --project=<path>
// tears that note off the project's board a moment after launch, through
// the same Shell.TearOff a click on the tear-off button calls, and logs
// what came of it. Notes joined by commas go through Shell.TearOffAll, as ⋯ → 이 탭
// 노트 모두 떼어내기 does. Unset, it does nothing.
func debugTear(s *shell.Shell) {
	note := os.Getenv(envTear)
	project := shell.ProjectArg(os.Args)
	if note == "" || project == "" {
		return
	}
	go func() {
		time.Sleep(tearDelay)
		id := s.OpenProject(project)
		if strings.Contains(note, ",") {
			ids, err := s.TearOffAll(id, strings.Split(note, ","))
			log.Printf("%s %q: windows %v, err %v", envTear, note, ids, err)
			return
		}
		tid, err := s.TearOff(id, note)
		log.Printf("%s %q: window %d, err %v", envTear, note, tid, err)
	}()
}
