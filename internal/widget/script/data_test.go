package script

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
	want := `{"source":"#!/bin/sh\ngo test ./...\n","lang":"sh"}`
	if string(b) != want {
		t.Errorf("Data = %s, want %s", b, want)
	}
}

func TestDataNamesPowerShell(t *testing.T) {
	b, _ := json.Marshal(Kind.Data(doc.Parse([]byte("#!/usr/bin/env pwsh\nGet-Date\n"))))
	want := `{"source":"#!/usr/bin/env pwsh\nGet-Date\n","lang":"ps1"}`
	if string(b) != want {
		t.Errorf("Data = %s, want %s", b, want)
	}
}
