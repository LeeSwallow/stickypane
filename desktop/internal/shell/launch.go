package shell

import (
	"path/filepath"
	"strings"

	"github.com/LeeSwallow/stickypane/desktop/internal/remote"
)

// Launch handles the arguments of a launch, the first or a second one
// passed on by the single instance: --project=<path> (relative to dir)
// opens that project, anything else shows the home window.
func (s *Shell) Launch(args []string, dir string) {
	path := ProjectArg(args)
	if path == "" {
		s.ShowHome()
		return
	}
	if !filepath.IsAbs(path) && !remote.IsRemote(path) && dir != "" {
		path = filepath.Join(dir, path)
	}
	s.OpenProject(path)
}

// ProjectArg is the value of --project in args (args[0] is the program).
func ProjectArg(args []string) string {
	for i := 1; i < len(args); i++ {
		a := args[i]
		for _, pre := range []string{"--project=", "-project="} {
			if strings.HasPrefix(a, pre) {
				return strings.TrimPrefix(a, pre)
			}
		}
		if (a == "--project" || a == "-project") && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// OpenAtLaunch opens what a launch asks for by name: "settings" or
// "settings:<section>", for the last board opened (STICKYPANE_APP_OPEN,
// to look at a window without pressing anything). Anything else is
// nothing.
func (s *Shell) OpenAtLaunch(what string) {
	name, section, _ := strings.Cut(strings.TrimSpace(what), ":")
	if name != "settings" {
		return
	}
	s.mu.Lock()
	var from uint
	if p := s.projects[s.state.Last]; p != nil && p.win != nil {
		from = p.win.ID()
	}
	s.mu.Unlock()
	s.OpenSettings(from, section)
}
