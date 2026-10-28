package note

import (
	"encoding/json"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

func TestDataOfTheExample(t *testing.T) {
	k := NewKind(Plain)
	b, err := json.Marshal(k.Data(doc.Parse([]byte(k.Example)), widget.DefaultEnv()))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"markdown":"Decided: tokens live in a session cookie.\n\n- revisit when we add SSO\n"}`
	if string(b) != want {
		t.Errorf("Data = %s, want %s", b, want)
	}
}
