package git

import (
	"os"
	"testing"
)

// The tests run git in folders of their own. Started from a git hook or a
// rebase --exec, git's variables would point it at the repository being
// worked on instead, and change that.
func TestMain(m *testing.M) {
	for _, v := range Local {
		os.Unsetenv(v)
	}
	os.Exit(m.Run())
}
