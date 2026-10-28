package chart

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
	want := `{"view":"bar","prose":[],"points":[{"label":"app","text":"61","value":61},{"label":"board","text":"31","value":31},{"label":"checklist","text":"21","value":21},{"label":"store","text":"16","value":16}]}`
	if string(b) != want {
		t.Errorf("Data = %s, want %s", b, want)
	}
}
