package shell

// The environment the shell reads. The app's debug hooks
// (STICKYPANE_APP_*) are the program's, in main; stickypane's own
// (STICKYPANE_BIN) the bridge client's.
const (
	// EnvHome is the home folder in place of ~/.stickypane: a check's own,
	// the same variable stickypane reads.
	EnvHome = "STICKYPANE_HOME"
	// EnvSSH is a program to run in place of ssh: a check's stand-in
	// (docs/desktop/02 §2.10).
	EnvSSH = "STICKYPANE_SSH"
	// envShell is the person's login shell, for the PATH the terminal
	// command goes on; defaultShell when it is not set.
	envShell     = "SHELL"
	defaultShell = "/bin/zsh"
)

// homeDirName is the home folder's name under the person's home.
const homeDirName = ".stickypane"
