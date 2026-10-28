package chat

import (
	"encoding/json"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

func TestDataOfTheExample(t *testing.T) {
	b, err := json.Marshal(Kind.Data(doc.Parse([]byte(Kind.Example))))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"messages":[{"day":"2026-10-03","who":"min","clock":"14:02","text":"can you rerun the tests?"},{"day":"2026-10-03","who":"claude","clock":"14:03","text":"41/41 pass"}]}`
	if string(b) != want {
		t.Errorf("Data = %s, want %s", b, want)
	}
}
