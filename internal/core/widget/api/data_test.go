package api

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
	want := `{"source":"@base = https://httpbin.org\n\n### Echo\nPOST {{base}}/anything\nContent-Type: application/json\n\n{\"user\": \"min\"}\n# @assert response.statusCode == 200\n# @assert response.json(\"json.user\") == \"min\"\n","env":"","requests":[{"title":"Echo","method":"POST","where":"{{base}}/anything","asserts":2,"hooks":false,"block":["Content-Type: application/json","","{\"user\": \"min\"}","# @assert response.statusCode == 200","# @assert response.json(\"json.user\") == \"min\""]}]}`
	if string(b) != want {
		t.Errorf("Data = %s, want %s", b, want)
	}
}

func TestDataSaysTheEnvironmentAndEveryRequest(t *testing.T) {
	src := "# @env dev\n\n### One\nGET https://a.example/x\n\n###\nDELETE https://a.example/y\n"
	b, err := json.Marshal(Kind.Data(doc.Document{Body: src}, widget.DefaultEnv()))
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Env      string
		Requests []Requested
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Env != "dev" || len(got.Requests) != 2 {
		t.Fatalf("Data = %s, want env dev and two requests", b)
	}
	if r := got.Requests[1]; r.Title != "DELETE https://a.example/y" || r.Method != "DELETE" || r.Asserts != 0 {
		t.Errorf("second request = %+v", r)
	}
	b, _ = json.Marshal(Kind.Data(doc.Document{}, widget.DefaultEnv()))
	if want := `{"source":"","env":"","requests":[]}`; string(b) != want {
		t.Errorf("Data of an empty file = %s, want %s", b, want)
	}
}
