package chart

import (
	"encoding/json"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/core/doc"
	"github.com/LeeSwallow/stickypane/internal/core/widget"
)

func TestDataOfTheExample(t *testing.T) {
	t.Parallel()
	b, err := json.Marshal(Kind.Data(doc.Parse([]byte(Kind.Example)), widget.DefaultEnv()))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"view":"bar","prose":[],"points":[{"label":"app","text":"61","value":61},{"label":"board","text":"31","value":31},{"label":"checklist","text":"21","value":21},{"label":"store","text":"16","value":16}]}`
	if string(b) != want {
		t.Errorf("Data = %s, want %s", b, want)
	}
}

// A calendar with no dated value is drawn as bars, and Data says so, so
// the app draws what the terminal draws.
func TestDataSaysBarWhenAHeatHasNoDays(t *testing.T) {
	t.Parallel()
	for src, want := range map[string]string{
		"---\nview: heat\n---\napp: 3\nboard: 4\n":  "bar",
		"---\nview: heat\n---\n2026-10-01: 3\n":     "heat",
		"---\nview: heat\n---\n":                    "heat",
		"---\nview: spark\n---\napp: 3\n":           "spark",
		"---\nview: nonsense\n---\n2026-10-01: 3\n": "bar",
	} {
		if got := Kind.Data(doc.Parse([]byte(src)), widget.DefaultEnv()).(Data).View; got != want {
			t.Errorf("%q: view %q, want %q", src, got, want)
		}
	}
}
