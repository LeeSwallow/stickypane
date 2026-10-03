package api

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

const file = `@base = http://localhost

### Log in
# @pre ./make-user.sh
POST {{base}}/login
Content-Type: application/json

{"user": "min"}
# @assert status == 200

### Me
GET {{base}}/me
`

func TestDrawAndSend(t *testing.T) {
	var w widget.Widget = Kind.Parse(doc.Document{Body: file})
	out, at := w.Draw(60, true)
	plain := ansi.Strip(out)
	if !strings.Contains(plain, "› POST   Log in") || !strings.Contains(plain, "GET    Me") {
		t.Errorf("each request should be a line:\n%s", plain)
	}
	if !strings.Contains(plain, `{"user": "min"}`) || !strings.Contains(plain, "@pre ./make-user.sh") || !at.Ok() {
		t.Errorf("the picked request should unfold under it:\n%s", plain)
	}
	if w.Summary() != "2 requests" {
		t.Errorf("summary %q", w.Summary())
	}
	w, _ = w.Update("j")
	_, res := w.Update("enter")
	if !res.Run || res.Part != 1 {
		t.Errorf("enter should ask to send the second request: %+v", res)
	}
}

func TestSetEnv(t *testing.T) {
	d, _ := SetEnv{Name: "prod"}.Apply(doc.Document{Body: file})
	if !strings.HasPrefix(d.Body, "# @env prod\n@base") {
		t.Fatalf("%q", d.Body)
	}
	d, _ = SetEnv{Name: "dev"}.Apply(d)
	if strings.Count(d.Body, "@env") != 1 || !strings.HasPrefix(d.Body, "# @env dev\n") {
		t.Fatalf("%q", d.Body)
	}
	d, _ = SetEnv{}.Apply(d)
	if d.Body != file {
		t.Fatalf("an empty name should take the line out: %q", d.Body)
	}
}

func TestAWebSocketAndAGRPCCallShowTheirProtocol(t *testing.T) {
	src := "### Chat\n# @ws send hi\n# @ws close\nGET ws://localhost/chat\n\n### Get user\n# @grpc users.v1.Users/Get\nGRPC localhost:9090\n\n{\"id\": 7}\n"
	out, _ := Kind.Parse(doc.Document{Body: src}).Draw(80, false)
	plain := ansi.Strip(out)
	for _, want := range []string{"WS     Chat  ws://localhost/chat  2 steps", "GRPC   Get user  users.v1.Users/Get localhost:9090"} {
		if !strings.Contains(plain, want) {
			t.Errorf("want %q in:\n%s", want, plain)
		}
	}
}
