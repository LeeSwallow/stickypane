package widget

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/LeeSwallow/stickypane/internal/core/doc"
)

// ErrUnknownAction is what a kind's Act answers when it does not know the
// action's name.
var ErrUnknownAction = errors.New("unknown action")

// Action is something a screen other than the terminal asks of a note, such
// as ticking an item: Name says what, Args say which.
type Action struct {
	Name string            `json:"name"`
	Args map[string]string `json:"args,omitempty"`
}

// Version names the content of a note. A screen sends it back with an
// action, so the board can tell that the note changed since the screen drew
// it. It is the first 16 hex digits of the SHA-256 of the file's bytes.
func Version(d doc.Document) string {
	sum := sha256.Sum256(d.Bytes())
	return hex.EncodeToString(sum[:])[:16]
}
