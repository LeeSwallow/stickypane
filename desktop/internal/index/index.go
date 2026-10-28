// Package index keeps the app's index of every board used on this machine.
// The files under ~/.stickypane (events/, run/) are the truth, written and
// pruned by stickypane alone; the database stickypane.db is only an index
// of them and can be made again from events/ at any time (docs/desktop/08
// §8.4).
//
// This package only puts the parts together (ADR-0018): the domain
// (model), the use cases (service) over their ports (ports), and the
// adapters: the database (gormstore), the records (registryfs), the board
// folders (boardfs) and the processes (procs).
package index

import (
	"os"
	"path/filepath"
	"time"

	"github.com/LeeSwallow/stickypane/desktop/internal/index/boardfs"
	"github.com/LeeSwallow/stickypane/desktop/internal/index/gormstore"
	"github.com/LeeSwallow/stickypane/desktop/internal/index/model"
	"github.com/LeeSwallow/stickypane/desktop/internal/index/procs"
	"github.com/LeeSwallow/stickypane/desktop/internal/index/registryfs"
	"github.com/LeeSwallow/stickypane/desktop/internal/index/service"
)

// The home list's types, from the domain.
type (
	Project = model.Project
	Session = model.Session
	Row     = model.Row
	Day     = model.Day
	Groups  = model.Groups
)

// File is the database's name in the home folder.
const File = "stickypane.db"

// Index is the open index: the use cases of service, wired to this
// machine.
type Index struct {
	*service.Service
	store *gormstore.Store
}

// Open opens (making it if need be) home/stickypane.db. A database that is
// damaged, or of another schema version, is renamed to
// stickypane.db.broken-<time> and made again from events/.
func Open(home string) (*Index, error) {
	if err := os.MkdirAll(home, 0o755); err != nil {
		return nil, err
	}
	store, rebuilt, err := gormstore.Open(filepath.Join(home, File))
	if err != nil {
		return nil, err
	}
	ix := &Index{
		Service: &service.Service{
			Store:   store,
			Records: registryfs.Home(home),
			Boards:  boardfs.Disk{},
			Clock:   systemClock{},
			Procs:   procs.PS{},
		},
		store: store,
	}
	if rebuilt {
		if err := ix.Ingest(); err != nil { // offsets are empty: reads all of events/
			store.Close()
			return nil, err
		}
	}
	return ix, nil
}

// Close closes the database.
func (ix *Index) Close() error { return ix.store.Close() }

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }
