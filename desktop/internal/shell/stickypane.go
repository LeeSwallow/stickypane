package shell

import (
	"errors"

	"github.com/LeeSwallow/stickypane/desktop/internal/bridgeclient"
)

// StickypaneState is whether a stickypane was found, whether it has the
// bridge the app needs, and what it says of itself.
type StickypaneState struct {
	Found       bool                     `json:"found"`
	Bridge      bool                     `json:"bridge"` // false: every copy found is too old
	Path        string                   `json:"path,omitempty"`
	Version     string                   `json:"version,omitempty"`
	Protocol    int                      `json:"protocol"`             // the bridge's, 0 until a bridge has said hello
	AppProtocol int                      `json:"appProtocol"`          // the one this app speaks
	Candidates  []bridgeclient.Candidate `json:"candidates,omitempty"` // the copies found, when all are too old
	Error       string                   `json:"error,omitempty"`
}

// StickypaneStatus finds the stickypane the app would use. The protocol is
// known once a project window's bridge has said hello.
func (s *Shell) StickypaneStatus() StickypaneState {
	st := StickypaneState{AppProtocol: bridgeclient.Protocol}
	bin, err := s.cfg.Find()
	var old *bridgeclient.TooOldError
	switch {
	case errors.As(err, &old):
		st.Found, st.Candidates, st.Error = true, old.Found, err.Error()
		if len(old.Found) > 0 {
			st.Path, st.Version = old.Found[0].Path, old.Found[0].Version
		}
		return st
	case err != nil:
		st.Error = err.Error()
		return st
	}
	c := s.cfg.Probe(bin)
	st.Found, st.Bridge, st.Path, st.Version = true, true, bin, c.Version
	s.mu.Lock()
	st.Protocol = s.hello.Protocol
	s.mu.Unlock()
	return st
}

// About is what the settings' 정보 shows: the app's version and the
// stickypane the app uses.
type About struct {
	App        string          `json:"app"`
	Stickypane StickypaneState `json:"stickypane"`
}

// About is the app's version and its stickypane's.
func (s *Shell) About() About {
	return About{App: s.cfg.AppVersion, Stickypane: s.StickypaneStatus()}
}
