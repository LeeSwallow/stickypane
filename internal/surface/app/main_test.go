package app

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// A board without a language of its own follows the environment's; the
	// tests pin it, so they do not depend on the machine's.
	os.Setenv("STICKYPANE_LANG", "en")
	os.Exit(m.Run())
}
