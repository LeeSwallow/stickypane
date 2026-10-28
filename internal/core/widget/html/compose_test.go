package html

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/core/doc"
	"github.com/LeeSwallow/stickypane/internal/core/widget"
)

func TestComposeWritesAPageWithItsTitleEscaped(t *testing.T) {
	t.Parallel()
	b, err := Kind.Make("R&D <report>", map[string]string{"text": "fast & small\nsecond"})
	if err != nil {
		t.Fatal(err)
	}
	page := string(b)
	for _, want := range []string{"<title>R&amp;D &lt;report&gt;</title>", "<h1>R&amp;D &lt;report&gt;</h1>", "<p>fast &amp; small</p>", "<p>second</p>"} {
		if !strings.Contains(page, want) {
			t.Errorf("page lacks %q: %q", want, page)
		}
	}
	raw, _ := json.Marshal(data(doc.Document{Body: page}, widget.DefaultEnv()))
	var got struct {
		HTML string `json:"html"`
	}
	if json.Unmarshal(raw, &got); got.HTML != page {
		t.Errorf("the page reads back as written: %s", raw)
	}
}
