// Package api is the note of HTTP requests: a .http file, written the way
// resterm, the VS Code REST Client and JetBrains write it, shown as a list
// of requests to send. The widget only picks one and asks for it to be
// sent; the app sends it and shows the response under the list.
package api

import (
	"fmt"
	"regexp"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/httpfile"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// Kind registers the requests note. It is the kind of every ".http" and
// ".rest" file.
var Kind = widget.Kind{
	Name:    "api",
	Exts:    []string{".http", ".rest"},
	New:     ".http",
	Label:   "API requests (exp)",
	Icon:    "⇄",
	Hint:    []string{"j k", "request", "enter", "send", "e", "environment"},
	Blurb:   "Experimental. REST, WebSocket and gRPC requests in a .http file, sent with enter; the response, its checks and what it captured show under them.",
	Command: "stickypane api api.http \"Log in\"",
	Usage: "A .http file as resterm and the VS Code REST Client write it: \"### name\" starts a request, then the request line, headers, a blank line and the body. " +
		"\"@base = http://localhost:8080\" and \"{{base}}\" are variables; a value \"env:NAME\" or \"{{$processEnv NAME}}\" comes from the environment or the project's .env, and rest-client.env.json (or resterm.env.json) holds environments, chosen with \"# @env dev\". " +
		"Under a request, in resterm's words so the file runs in resterm too: \"# @assert response.statusCode == 200\", \"# @assert response.json(\\\"user.id\\\") == 7\", \"# @assert \\\"json\\\" in response.header(\\\"Content-Type\\\")\" (== != < > <= >= in contains exists); \"# @capture file token {{response.json.token}}\" keeps a value for the requests after it. " +
		"A WebSocket request is a ws:// URL or \"# @websocket timeout=5s idle-timeout=1s subprotocols=a,b\" with \"# @ws send <text>\", send-json, send-base64, ping, pong, \"wait 500ms\" and \"close 1000 bye\" steps; its log is the transcript, \"response.received\" counts what came back and \"response.json\" is the array of it. " +
		"A gRPC request is \"GRPC host:port\" with \"# @grpc package.Service/Method\", \"# @grpc-plaintext true\" for HTTP/2 without TLS, \"# @grpc-metadata key: value\", and a JSON body; the descriptors come from \"# @grpc-descriptor file.protoset\", else server reflection, else fields go by number ({\"1\": 7}). \"response.grpc.status\" is the code (\"OK\", \"NOT_FOUND\"); unary and server-streaming calls. " +
		"Only stickypane reads \"# @pre <command>\", whose name=value lines become variables, and \"# @post <command>\", which reads the response as JSON on standard input. " +
		"enter asks the user, then sends the request; `stickypane api <file> <request>` sends it without asking. The response goes to a .log of the same name: the request as curl with credentials hidden, the body, the checks, and a last line such as \"[200 OK · 38ms · Log in · 2/2 ✔]\".",
	Example: "@base = https://httpbin.org\n\n### Echo\nPOST {{base}}/anything\nContent-Type: application/json\n\n{\"user\": \"min\"}\n# @assert response.statusCode == 200\n# @assert response.json(\"json.user\") == \"min\"\n",
	Keys:    "j k up down enter space e",
	Size:    func(doc.Document) string { return widget.SizeHalf },
	Template: func(string) []byte {
		return []byte("### Example\nGET https://httpbin.org/get\n# @assert response.statusCode == 200\n")
	},
	Parse: func(d doc.Document) widget.Widget { return parse(d) },
}

// API is the widget for a .http file.
type API struct {
	file   httpfile.File
	lines  []string
	cursor int
	drawn  []widget.Span // the lines of each request in the last Draw
}

func parse(d doc.Document) *API {
	return &API{file: httpfile.Parse(d.Body), lines: doc.Lines(d.Body)}
}

func (a *API) clamp() { a.cursor = max(min(a.cursor, len(a.file.Requests)-1), 0) }

// methodStyle colors a method by what it does: reads, writes, removes.
// A WebSocket session and a gRPC call get the accent, so the protocol
// stands out from the methods.
func methodStyle(m string) lipgloss.Style {
	switch m {
	case "WS", "GRPC":
		return widget.Accent
	case "GET", "HEAD", "OPTIONS":
		return widget.Info
	case "POST":
		return widget.Good
	case "DELETE":
		return widget.Bad
	default:
		return widget.Warn
	}
}

// Draw implements widget.Widget: the environment, then a line per request,
// and under the selected one, when active, the request as written, so what
// will be sent can be read first.
func (a *API) Draw(width int, active bool) (string, widget.Span) {
	a.drawn = a.drawn[:0]
	if len(a.file.Requests) == 0 {
		return widget.Fit(widget.Faint.Render(widget.T("(no requests yet: ### name, then GET https://…)")), width), widget.NoSpan
	}
	a.clamp()
	var out []string
	if a.file.Env != "" {
		out = append(out, widget.Faint.Render(widget.T("environment")+" "+a.file.Env), "")
	}
	at := widget.NoSpan
	for i, r := range a.file.Requests {
		start := len(out)
		mark := "  "
		if active && i == a.cursor {
			mark = "› "
		}
		badge := r.Badge()
		method := methodStyle(badge).Render(fmt.Sprintf("%-6s", badge))
		line := mark + method + " "
		where := r.URL
		switch r.Protocol() {
		case "grpc":
			if r.GRPC != nil && r.GRPC.Method != "" {
				where = r.GRPC.Method + " " + r.URL
			}
		case "websocket":
			if r.WS != nil && len(r.WS.Steps) > 0 {
				where += fmt.Sprintf("  %d %s", len(r.WS.Steps), map[bool]string{true: "step", false: "steps"}[len(r.WS.Steps) == 1])
			}
		}
		rest := where
		if r.Name != "" {
			rest = r.Name + "  " + widget.Faint.Render(where)
		}
		if n := len(r.Asserts); n > 0 {
			rest += widget.Faint.Render(fmt.Sprintf("  %d ✓", n))
		}
		if r.Hooks() {
			rest += widget.Faint.Render("  ⚙")
		}
		line = widget.Truncate(line+rest, width)
		if active && i == a.cursor {
			// The colors inside would cut the highlight short: the picked
			// line is one plain run of text on the highlight.
			line = widget.Selected.Render(widget.Pad(ansi.Strip(line), width))
		}
		out = append(out, line)
		if active && i == a.cursor {
			for _, l := range a.block(r) {
				for _, part := range widget.Wrap(widget.Clean(l), max(width-4, 1)) {
					out = append(out, "    "+widget.Faint.Render(part))
				}
			}
			at = widget.Span{Start: start, End: len(out)}
		}
		a.drawn = append(a.drawn, widget.Span{Start: start, End: len(out)})
	}
	return widget.Fit(strings.Join(out, "\n"), width), at
}

// block is the request as written, without its "###" line, its request
// line, which the line above shows, and the blank lines around it. The
// hooks before the request line are kept: they run when it is sent.
func (a *API) block(r httpfile.Request) []string {
	end := min(r.End, len(a.lines))
	var ls []string
	for i := r.Start; i < end; i++ {
		t := strings.TrimRight(a.lines[i], " \r")
		if i == r.Line || (i == r.Start && strings.HasPrefix(strings.TrimSpace(t), "###")) {
			continue
		}
		if len(ls) == 0 && strings.TrimSpace(t) == "" {
			continue
		}
		ls = append(ls, t)
	}
	for len(ls) > 0 && strings.TrimSpace(ls[len(ls)-1]) == "" {
		ls = ls[:len(ls)-1]
	}
	return ls
}

// Selected is the title of the request the cursor is on, whose response
// the board shows under the list.
func (a *API) Selected() string {
	a.clamp()
	if len(a.file.Requests) == 0 {
		return ""
	}
	return a.file.Requests[a.cursor].Title()
}

// Summary implements widget.Widget: how many requests.
func (a *API) Summary() string {
	switch n := len(a.file.Requests); n {
	case 0:
		return ""
	case 1:
		return widget.T("1 request")
	default:
		return fmt.Sprintf(widget.T("%d requests"), n)
	}
}

// Update implements widget.Widget: j and k pick a request, enter sends it,
// e chooses the environment.
func (a *API) Update(key string) (widget.Widget, widget.Result) {
	var res widget.Result
	switch key {
	case "j", "down":
		a.cursor++
	case "k", "up":
		a.cursor--
	case "enter", "space":
		a.clamp()
		if len(a.file.Requests) > 0 {
			res.Run, res.Part = true, a.cursor
		}
	case "e":
		res.Prompt = &widget.Prompt{
			Label:   widget.T("Environment"),
			Initial: a.file.Env,
			Empty:   true,
			Submit:  func(text string) doc.Op { return SetEnv{Name: strings.TrimSpace(text)} },
		}
	}
	a.clamp()
	return a, res
}

// Click implements widget.Clicker: a press picks a request, and a press on
// the picked one sends it.
func (a *API) Click(line, _ int) (widget.Widget, widget.Result, bool) {
	for i, s := range a.drawn {
		if line >= s.Start && line < s.End {
			if i == a.cursor {
				w, res := a.Update("enter")
				return w, res, true
			}
			a.cursor = i
			return a, widget.Result{}, true
		}
	}
	return a, widget.Result{}, false
}

// Sync implements widget.Widget: the file changed; the cursor stays.
func (a *API) Sync(d doc.Document) widget.Widget {
	n := parse(d)
	n.cursor = a.cursor
	n.clamp()
	return n
}

// envLine matches the line that chooses the environment.
var envLine = regexp.MustCompile(`^\s*(#|//)\s*@env\b`)

// SetEnv writes "# @env name" at the top of the file, or removes it when
// the name is empty.
type SetEnv struct{ Name string }

// Apply implements doc.Op.
func (o SetEnv) Apply(d doc.Document) (doc.Document, error) {
	lines := strings.Split(d.Body, "\n")
	first := len(lines)
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "###") {
			first = i
			break
		}
	}
	for i := 0; i < first; i++ {
		if envLine.MatchString(lines[i]) {
			if o.Name == "" {
				lines = append(lines[:i], lines[i+1:]...)
			} else {
				lines[i] = "# @env " + o.Name
			}
			d.Body = strings.Join(lines, "\n")
			return d, nil
		}
	}
	if o.Name != "" {
		d.Body = "# @env " + o.Name + "\n" + d.Body
	}
	return d, nil
}
