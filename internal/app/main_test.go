package app

import (
	"os"
	"testing"
)

// The tests read the English screen, whatever the language of the machine
// that runs them.
func TestMain(m *testing.M) {
	os.Setenv("STICKYPANE_LANG", "en")
	os.Exit(m.Run())
}
