package checklist

import (
	"encoding/json"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

func TestComposeWritesOpenItems(t *testing.T) {
	t.Parallel()
	b, err := Kind.Make("Release", map[string]string{"items": "build\n\n tag \npublish"})
	if err != nil {
		t.Fatal(err)
	}
	d := doc.Parse(b)
	if d.Type() != "checklist" {
		t.Errorf("type = %q", d.Type())
	}
	raw, _ := json.Marshal(data(d, widget.DefaultEnv()))
	var got struct {
		Items []thing `json:"items"`
		Done  int     `json:"done"`
		Total int     `json:"total"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	want := []string{"build", "tag", "publish"}
	if got.Total != 3 || got.Done != 0 || len(got.Items) != 3 {
		t.Fatalf("data = %s", raw)
	}
	for i, w := range want {
		if got.Items[i].Text != w || got.Items[i].Checked {
			t.Errorf("item %d = %+v", i, got.Items[i])
		}
	}
}
