package shell

import (
	"encoding/json"
	"testing"
)

// The app's create dialog sends options as the frontend wrote them, and
// no body: the bridge's note.create refuses both at once.
func TestCreateParamsCarryOptionsNotBody(t *testing.T) {
	var c CreateParams
	in := `{"tab":"work","kind":"board","title":"Sprint","options":{"columns":["To do","Done"],"cards":"a\nb"}}`
	if err := json.Unmarshal([]byte(in), &c); err != nil {
		t.Fatal(err)
	}
	out, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	var back map[string]any
	if err := json.Unmarshal(out, &back); err != nil {
		t.Fatal(err)
	}
	if _, ok := back["body"]; ok {
		t.Errorf("an empty body is sent: %s", out)
	}
	opts, _ := back["options"].(map[string]any)
	cols, _ := opts["columns"].([]any)
	if len(cols) != 2 || cols[0] != "To do" || opts["cards"] != "a\nb" {
		t.Errorf("options changed on the way: %s", out)
	}
}
