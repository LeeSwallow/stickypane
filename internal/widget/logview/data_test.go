package logview

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

func TestDataOfTheExample(t *testing.T) {
	t.Parallel()
	b, err := json.Marshal(Kind.Data(doc.Parse([]byte(Kind.Example)), widget.DefaultEnv()))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"lines":["14:02 tests passed","14:10 started on review feedback","14:31 pushed the fix"]}`
	if string(b) != want {
		t.Errorf("Data = %s, want %s", b, want)
	}
}

func TestDataKeepsTheLastThousandLines(t *testing.T) {
	t.Parallel()
	var b strings.Builder
	for i := 0; i < 1005; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	got := Kind.Data(doc.Parse([]byte(b.String())), widget.DefaultEnv()).(struct {
		Lines []string `json:"lines"`
	}).Lines
	if len(got) != 1000 || got[0] != "line 5" || got[999] != "line 1004" {
		t.Errorf("got %d lines, from %q to %q", len(got), got[0], got[len(got)-1])
	}
}
