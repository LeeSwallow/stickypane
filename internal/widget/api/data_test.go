package api

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
	want := `{"source":"@base = https://httpbin.org\n\n### Echo\nPOST {{base}}/anything\nContent-Type: application/json\n\n{\"user\": \"min\"}\n# @assert response.statusCode == 200\n# @assert response.json(\"json.user\") == \"min\"\n"}`
	if string(b) != want {
		t.Errorf("Data = %s, want %s", b, want)
	}
}
