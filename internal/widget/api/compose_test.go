package api

import (
	"errors"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/httpfile"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

func TestComposeWritesOneRequest(t *testing.T) {
	b, err := Kind.Make("Log in", map[string]string{"method": "post", "url": "http://localhost:8080/login", "status": "201"})
	if err != nil {
		t.Fatal(err)
	}
	f := httpfile.Parse(string(b))
	if len(f.Requests) != 1 {
		t.Fatalf("requests in %q: %d", b, len(f.Requests))
	}
	r := f.Requests[0]
	if r.Name != "Log in" || r.Method != "POST" || r.URL != "http://localhost:8080/login" || len(r.Asserts) != 1 || r.Asserts[0] != "response.statusCode == 201" {
		t.Errorf("request = %+v", r)
	}
	b, _ = Kind.Make("", map[string]string{"status": ""})
	if r := httpfile.Parse(string(b)).Requests[0]; r.Name == "" || len(r.Asserts) != 0 || r.Method != "GET" {
		t.Errorf("defaults without a check: %+v", r)
	}
	if _, err := Kind.Make("x", map[string]string{"url": ""}); !errors.Is(err, widget.ErrBadOption) {
		t.Errorf("a request needs a URL: %v", err)
	}
}
