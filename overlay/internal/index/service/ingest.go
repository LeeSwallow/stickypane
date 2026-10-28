package service

import (
	"errors"
	"time"

	"github.com/LeeSwallow/stickypane/overlay/internal/index/model"
	"github.com/LeeSwallow/stickypane/overlay/internal/index/ports"
)

// Ingest applies the lines of events/ not yet read, then ends the sessions
// of run files whose process is no longer the one they name. A last line
// without its newline is left for next time; a line that is not an event
// is skipped and counted (Skipped). It deletes no file: stickypane prunes
// its own records.
func (s *Service) Ingest() error {
	files, err := s.Records.EventFiles()
	if err != nil {
		return err
	}
	present := map[string]bool{}
	for _, name := range files {
		present[name] = true
		if err := s.ingestFile(name); err != nil {
			return err
		}
	}
	log := s.Store.Repos().Ingest
	known, err := log.Files()
	if err != nil {
		return err
	}
	for _, f := range known { // what was applied stays; only the offset goes
		if !present[f] {
			if err := log.Forget(f); err != nil {
				return err
			}
		}
	}
	_, err = s.liveRuns(true)
	return err
}

func (s *Service) ingestFile(name string) error {
	// The offset is read inside the write transaction, which takes the lock
	// when it begins, so two instances never apply the same lines twice.
	return s.Store.Write(func(r ports.Repos) error {
		off, err := r.Ingest.Offset(name)
		if err != nil {
			return err
		}
		data, size, err := s.Records.ReadFrom(name, off)
		if errors.Is(err, ports.ErrGone) {
			return nil // removed since the listing
		}
		if err != nil {
			return err
		}
		// Known limit: a file truncated and then grown past its old offset
		// is not noticed, and the bytes before the offset are never re-read.
		if size < off { // a different, shorter file under the same name
			off = 0
			if data, _, err = s.Records.ReadFrom(name, 0); err != nil {
				return err
			}
		}
		lines, used := model.Lines(data)
		if used == 0 {
			return nil
		}
		skipped := 0
		for _, line := range lines {
			e, at, ok := model.ParseEvent(line)
			if !ok {
				skipped++
				continue
			}
			if err := s.apply(r, e, at); err != nil {
				return err
			}
		}
		if skipped > 0 {
			if err := r.Ingest.AddSkipped(skipped); err != nil {
				return err
			}
		}
		return r.Ingest.SetOffset(name, off+int64(used))
	})
}

// apply records one event: its project (made, or its last activity and
// removal mark updated), its day, and its session.
func (s *Service) apply(r ports.Repos, e model.Event, at time.Time) error {
	path := model.CleanPath(e.Project)
	p, found, err := r.Projects.ByPath(path)
	if err != nil {
		return err
	}
	if !found {
		p = model.Project{Path: path, Title: model.DefaultTitle(path), FirstSeen: at, LastSeen: at}
		if p.ID, err = r.Projects.Create(p); err != nil {
			return err
		}
	} else {
		if model.Later(p.LastSeen, at) {
			if err := r.Projects.SetLastSeen(p.ID, at); err != nil {
				return err
			}
		}
		if p.RemovedAt != nil && s.Boards.Exists(path) { // an event for a folder that is back
			if err := r.Projects.SetRemoved(p.ID, nil); err != nil {
				return err
			}
		}
	}

	date := model.LocalDate(at)
	d, _, err := r.Days.Get(p.ID, date)
	if err != nil {
		return err
	}
	d.Date = date
	if err := r.Days.Put(p.ID, d.Count(at)); err != nil {
		return err
	}

	if e.PID <= 0 {
		return nil
	}
	switch e.Type {
	case model.SessionStart:
		return r.Sessions.Start(p.ID, e.PID, e.SessionKind(), at)
	case model.SessionEnd:
		return r.Sessions.End(p.ID, e.PID, at)
	}
	return nil
}

// liveRun is a run file whose process is alive.
type liveRun struct {
	run     model.Run
	started time.Time
}

// liveRuns returns the run files whose process is alive. With settle, the
// sessions of the others are ended (their files stay, for stickypane to
// remove).
func (s *Service) liveRuns(settle bool) ([]liveRun, error) {
	files, err := s.Records.Runs()
	if err != nil {
		return nil, nil // no run/ folder yet
	}
	var live []liveRun
	for _, f := range files {
		if !model.IsRunName(f.Name) { // touch-* markers, .tmp-* files
			continue
		}
		r, started, ok := model.ParseRun(f.Data)
		if ok && s.Procs.Alive(r.PID, started) {
			live = append(live, liveRun{r, started})
			continue
		}
		if settle && r.PID > 0 {
			if err := s.Store.Repos().Sessions.EndPID(r.PID, s.Clock.Now()); err != nil {
				return nil, err
			}
		}
	}
	return live, nil
}
