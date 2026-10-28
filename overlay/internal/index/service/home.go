package service

import (
	"time"

	"github.com/LeeSwallow/stickypane/overlay/internal/index/model"
)

// Prune settles what has been deleted (08 §8.4.4). A project whose folder
// or .sticky/ is gone is marked removed at now and returned (only once);
// one whose folder is back is restored; rows removed more than 30 days ago
// are deleted, with their days and sessions. The records in ~/.stickypane
// are left alone: stickypane removes its old events and dead run files.
func (s *Service) Prune(now time.Time) (removed []model.Project, err error) {
	projects := s.Store.Repos().Projects
	ps, err := projects.All()
	if err != nil {
		return nil, err
	}
	for _, p := range ps {
		switch model.SettleOf(p, s.Boards.Exists(p.Path), now) {
		case model.Remove:
			t := now
			if err := projects.SetRemoved(p.ID, &t); err != nil {
				return nil, err
			}
			p.RemovedAt = &t
			removed = append(removed, p)
		case model.Restore:
			if err := projects.SetRemoved(p.ID, nil); err != nil {
				return nil, err
			}
		case model.Forget:
			if err := projects.Delete(p.ID); err != nil {
				return nil, err
			}
		}
	}
	return removed, nil
}

// Sessions builds the home list at now (model.Group): live sessions come
// from run files whose process is alive, recent changes from the boards.
func (s *Service) Sessions(now time.Time) (model.Groups, error) {
	ps, err := s.Store.Repos().Projects.All()
	if err != nil {
		return model.Groups{}, err
	}
	runs, err := s.liveRuns(false)
	if err != nil {
		return model.Groups{}, err
	}
	live := map[string][]model.Session{}
	for _, r := range runs {
		p := model.CleanPath(r.run.Project)
		live[p] = append(live[p], model.Session{PID: r.run.PID, Kind: r.run.Kind, Started: r.started})
	}
	recent := map[string]bool{}
	for _, p := range ps {
		if t, ok := s.Boards.Changed(p.Path); ok && model.Recent(t, now) {
			recent[p.Path] = true
		}
	}
	return model.Group(ps, live, recent), nil
}
