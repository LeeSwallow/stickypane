// Package service is the index's use cases: read new records into the
// database, settle what was deleted, build the home list, restore and
// retitle a project. It orchestrates the domain (../model) through the
// ports (../ports) and holds the transaction boundaries; it touches no
// file, database or process itself (ADR-0018).
package service

import (
	"github.com/LeeSwallow/stickypane/overlay/internal/index/model"
	"github.com/LeeSwallow/stickypane/overlay/internal/index/ports"
)

// Service is the index. Every field is required.
type Service struct {
	Store   ports.Store
	Records ports.Records
	Boards  ports.Boards
	Clock   ports.Clock
	Procs   ports.ProcessChecker
}

// Restore undoes the removal of project id (the "undo" of the home screen).
func (s *Service) Restore(id int64) error {
	return s.Store.Repos().Projects.SetRemoved(id, nil)
}

// Retitle sets the title of the project at path, as its board names it
// (board.load). An empty title leaves it as it is; an unknown path is
// not an error.
func (s *Service) Retitle(path, title string) error {
	if title == "" {
		return nil
	}
	return s.Store.Repos().Projects.SetTitle(model.CleanPath(path), title)
}

// Projects returns every project row, oldest first.
func (s *Service) Projects() ([]model.Project, error) {
	return s.Store.Repos().Projects.All()
}

// Skipped is how many broken event lines have been skipped so far.
func (s *Service) Skipped() int {
	n, _ := s.Store.Repos().Ingest.Skipped()
	return n
}
