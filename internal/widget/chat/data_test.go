package chat

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

func TestDataOfTheExample(t *testing.T) {
	t.Parallel()
	b, err := json.Marshal(Kind.Data(doc.Parse([]byte(Kind.Example)), widget.DefaultEnv()))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"messages":[{"day":"2026-10-03","who":"min","clock":"14:02","text":"can you rerun the tests?"},{"day":"2026-10-03","who":"claude","clock":"14:03","text":"41/41 pass"}]}`
	if string(b) != want {
		t.Errorf("Data = %s, want %s", b, want)
	}
}

func TestActSayAppendsAMessageFromTheUser(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 10, 3, 14, 5, 0, 0, time.Local)
	d := doc.Parse([]byte(Kind.Example))
	op, err := Kind.Act(d, widget.Action{Name: "say", Args: map[string]string{"text": "thanks", "who": "you"}}, widget.Env{Clock: func() time.Time { return at }})

	if err != nil {
		t.Fatal(err)
	}
	got, err := op.Apply(d)
	if err != nil {
		t.Fatal(err)
	}
	if want := "@you 14:05 thanks\n"; !strings.HasSuffix(got.Body, want) {
		t.Errorf("body = %q, want it to end with %q", got.Body, want)
	}

	for _, text := range []string{"", "  "} {
		if _, err := Kind.Act(d, widget.Action{Name: "say", Args: map[string]string{"text": text, "who": "you"}}, widget.DefaultEnv()); err == nil {
			t.Errorf("say %q: no error", text)
		}
	}
	if _, err := Kind.Act(d, widget.Action{Name: "nope"}, widget.DefaultEnv()); !errors.Is(err, widget.ErrUnknownAction) {
		t.Errorf("unknown verb: err = %v, want ErrUnknownAction", err)
	}
}
