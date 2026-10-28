// Package ports are what the install use cases need from the world: a way
// to run a program, the files, and an administrator's rights. The
// adapters are in install/osys; tests pass fakes (ADR-0018).
package ports

import (
	"context"
)

// Cmd is one program to run.
type Cmd struct {
	Dir  string   // where it runs; never the root or the home folder for setup
	Path string   // the PATH it runs with ("" keeps the app's)
	Name string   // the program, a full path or a name looked up on Path
	Args []string // its arguments
	Env  []string // variables set over the app's, "NAME=value"
}

// Runner runs a program without input and returns what it printed, its
// output and errors together.
type Runner interface {
	Run(ctx context.Context, c Cmd) (string, error)
}

// Entry is what is at a path.
type Entry struct {
	Exists bool   // something is there, a dangling link too
	Link   bool   // it is a symbolic link
	Real   string // the path through its symlinks; "" when that fails
	Exec   bool   // a program that may run (a link's target)
}

// Files are the file operations the installer makes.
type Files interface {
	Stat(path string) Entry
	MkdirAll(dir string) error
	Symlink(target, link string) error
	Remove(path string) error
	// TempDir makes a new empty folder of the app's own; RemoveAll takes it
	// away again.
	TempDir() (string, error)
	RemoveAll(dir string) error
}

// Admin runs a shell script with an administrator's rights, after the
// system asks the person for them.
type Admin interface {
	Run(ctx context.Context, script string) error
}
