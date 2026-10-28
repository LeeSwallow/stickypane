// Package ports is what the index's use cases (../service) need from the
// world, as interfaces: the database (repositories and a unit of work),
// the records in ~/.stickypane, the board folders, the clock and the
// processes. The adapters implement them; the service sees nothing else
// (ADR-0018).
package ports

import (
	"errors"
	"time"

	"github.com/LeeSwallow/stickypane/desktop/internal/index/model"
)

// ErrGone: the file asked for is no longer there.
var ErrGone = errors.New("index: file gone")

// Clock tells the time.
type Clock interface{ Now() time.Time }

// ProcessChecker says whether the process pid, recorded as started at
// started, is still that process (not one that reused its pid).
type ProcessChecker interface {
	Alive(pid int, started time.Time) bool
}

// Boards looks at project folders.
type Boards interface {
	// Exists says whether path and its .sticky/ are there.
	Exists(path string) bool
	// Changed is when path/.sticky/ last changed; ok false when it is not
	// there.
	Changed(path string) (t time.Time, ok bool)
}

// RunFile is one file of run/, unread.
type RunFile struct {
	Name string
	Data []byte
}

// Records reads the records stickypane keeps in its home folder. It only
// reads: stickypane alone writes and deletes them.
type Records interface {
	// EventFiles lists the names of the files in events/.
	EventFiles() ([]string, error)
	// ReadFrom returns the bytes of events/name from offset on, and the
	// file's size. When the file is shorter than offset it returns no
	// bytes. ErrGone when the file is not there.
	ReadFrom(name string, offset int64) (data []byte, size int64, err error)
	// Runs returns the files of run/.
	Runs() ([]RunFile, error)
}

// ProjectRepository keeps the project rows.
type ProjectRepository interface {
	All() ([]model.Project, error) // oldest first
	ByPath(path string) (model.Project, bool, error)
	Create(p model.Project) (id int64, err error)
	SetLastSeen(id int64, t time.Time) error
	SetRemoved(id int64, at *time.Time) error // nil: not removed
	SetTitle(path, title string) error
	Delete(id int64) error // and its days and sessions
}

// DayRepository keeps what each project did each local day.
type DayRepository interface {
	Get(projectID int64, date string) (model.DayStat, bool, error)
	Put(projectID int64, d model.DayStat) error
}

// SessionRepository keeps the sessions of processes on projects.
type SessionRepository interface {
	// Start records a session; one already recorded is left as it is.
	Start(projectID int64, pid int, kind string, at time.Time) error
	// End ends the open session of pid on projectID.
	End(projectID int64, pid int, at time.Time) error
	// EndPID ends every open session of pid.
	EndPID(pid int, at time.Time) error
}

// IngestLog keeps how far each events file has been read, and how many
// lines were not events.
type IngestLog interface {
	Offset(file string) (int64, error) // 0 for a file not read yet
	SetOffset(file string, offset int64) error
	Files() ([]string, error)
	Forget(file string) error
	AddSkipped(n int) error
	Skipped() (int, error)
}

// Repos are the repositories of one unit of work.
type Repos struct {
	Projects ProjectRepository
	Days     DayRepository
	Sessions SessionRepository
	Ingest   IngestLog
}

// Store is the database.
type Store interface {
	// Repos works outside a transaction.
	Repos() Repos
	// Write runs f in one write transaction, taken when it begins, so two
	// processes never read the same state and both write after it. f's
	// error rolls it back.
	Write(f func(Repos) error) error
}
