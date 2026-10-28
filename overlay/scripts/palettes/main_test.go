package main

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

// TestGoldenIsCurrent fails when internal/core/theme changed and the frontend's
// fixture was not made again (see the command in main.go).
func TestGoldenIsCurrent(t *testing.T) {
	got, err := read("../../../internal/core/theme")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("../../frontend/src/demo/themes.json")
	if err != nil {
		t.Fatal(err)
	}
	var want []palette
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	if len(got) < 10 {
		t.Fatalf("read %d themes, want the ten built-in ones", len(got))
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("demo/themes.json is out of date: run go run ./scripts/palettes ../internal/core/theme > frontend/src/demo/themes.json")
	}
}
