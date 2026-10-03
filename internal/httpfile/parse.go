// Package httpfile reads and sends the requests of a .http file, the plain
// text format that resterm, the VS Code REST Client and JetBrains share:
// requests separated by "###" lines, each a request line, headers, a blank
// line and a body. Directives in comments ("# @assert status == 200") check
// the response, carry values to the next request and run hooks around it.
//
// The files: parse.go reads a file, vars.go fills in {{variables}} from the
// file, the environment and earlier responses, run.go sends a request and
// checks what came back, and log.go writes it down for the board and the
// agent.
package httpfile

import (
	"strings"
)

// Var is a variable the file declares: "@base = http://localhost" or
// "# @file base http://localhost". A value "env:NAME" is read from the
// environment when the request is sent.
type Var struct{ Name, Value string }

// Capture keeps a value of the response for the requests after it:
// "# @capture file token {{response.body.token}}".
type Capture struct{ Name, Expr string }

// Request is one request of the file.
type Request struct {
	Name     string // the "###" line's text, or "# @name", or empty
	Method   string
	URL      string
	Header   [][2]string // in order, as written
	Body     string
	Vars     []Var // declared for this request only
	Pre      []string
	Post     []string
	Captures []Capture
	Asserts  []string
	Line     int // the request line, counted from 0
	Start    int // the first line of the request's block, counted from 0
	End      int // one past its last line
}

// Title is how the request is called on the screen: its name, or else its
// method and URL.
func (r Request) Title() string {
	if r.Name != "" {
		return r.Name
	}
	return r.Method + " " + r.URL
}

// Hooks reports whether sending the request runs commands of the file.
func (r Request) Hooks() bool { return len(r.Pre)+len(r.Post) > 0 }

// File is a parsed .http file.
type File struct {
	Vars     []Var // declared for the whole file
	Env      string
	Requests []Request
}

// methods are the request line's first words that name a method.
var methods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true,
	"HEAD": true, "OPTIONS": true, "TRACE": true, "CONNECT": true,
}

// Parse reads a .http file. It never fails: what it does not understand is
// left out, so a half-written file still shows what it has.
func Parse(text string) File {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	var f File
	start, name := 0, ""
	flush := func(end int) {
		if r, ok := parseBlock(lines, start, end, name, &f); ok {
			f.Requests = append(f.Requests, r)
		}
	}
	for i, l := range lines {
		if t := strings.TrimSpace(l); strings.HasPrefix(t, "###") {
			flush(i)
			start, name = i, strings.TrimSpace(strings.TrimLeft(t, "#"))
		}
	}
	flush(len(lines))
	return f
}

// directive splits a comment line such as "# @assert status == 200" into
// "assert" and "status == 200". Bare "@word …" lines count too, as resterm
// writes them.
func directive(line string) (word, rest string, ok bool) {
	t := strings.TrimSpace(line)
	bare := true
	switch {
	case strings.HasPrefix(t, "#"):
		t, bare = strings.TrimSpace(strings.TrimLeft(t, "#")), false
	case strings.HasPrefix(t, "//"):
		t, bare = strings.TrimSpace(strings.TrimLeft(t, "/")), false
	}
	if !strings.HasPrefix(t, "@") || len(t) < 2 {
		return "", "", false
	}
	word, rest, _ = strings.Cut(t[1:], " ")
	word, rest = strings.ToLower(word), strings.TrimSpace(rest)
	// "@base = http://localhost" is a variable, not a directive.
	if bare && (!known[word] || strings.HasPrefix(rest, "=")) {
		return "", "", false
	}
	return word, rest, known[word]
}

// known are the directives Parse reads; others are comments.
var known = map[string]bool{
	"name": true, "pre": true, "post": true, "assert": true, "capture": true, "env": true,
	"file": true, "const": true, "global": true, "var": true, "request": true,
}

// isComment reports whether a line of a request's head is a comment.
func isComment(t string) bool { return strings.HasPrefix(t, "#") || strings.HasPrefix(t, "//") }

// parseBlock reads the lines [start, end) of one "###" block. File
// variables found before the request line of the first block belong to the
// file.
func parseBlock(lines []string, start, end int, name string, f *File) (Request, bool) {
	r := Request{Name: name, Line: -1, Start: start, End: end}
	const (
		head = iota
		headers
		body
	)
	state := head
	var bodyLines []string
	for i := start; i < end; i++ {
		l := lines[i]
		t := strings.TrimSpace(l)
		if i == start && strings.HasPrefix(t, "###") {
			continue
		}
		if word, rest, ok := directive(l); ok && (state != body || isComment(t)) {
			fileWide := r.Line < 0 && len(f.Requests) == 0
			apply(word, rest, &r, f, fileWide)
			continue
		}
		switch state {
		case head:
			if t == "" || isComment(t) {
				continue
			}
			if strings.HasPrefix(t, "@") { // "@name = value"
				v := assignment(t[1:])
				if r.Line < 0 && len(f.Requests) == 0 {
					f.Vars = append(f.Vars, v)
				} else {
					r.Vars = append(r.Vars, v)
				}
				continue
			}
			method, url := requestLine(t)
			if url == "" {
				continue
			}
			r.Method, r.URL, r.Line = method, url, i
			state = headers
		case headers:
			if t == "" {
				state = body
				continue
			}
			if isComment(t) {
				continue
			}
			if k, v, ok := strings.Cut(t, ":"); ok {
				r.Header = append(r.Header, [2]string{strings.TrimSpace(k), strings.TrimSpace(v)})
			}
		case body:
			bodyLines = append(bodyLines, l)
		}
	}
	r.Body = strings.TrimRight(strings.Join(bodyLines, "\n"), "\n \t")
	return r, r.Line >= 0
}

// apply records one directive.
func apply(word, rest string, r *Request, f *File, fileWide bool) {
	fields := strings.Fields(rest)
	switch word {
	case "name":
		r.Name = rest
	case "pre":
		r.Pre = append(r.Pre, rest)
	case "post":
		r.Post = append(r.Post, rest)
	case "assert":
		r.Asserts = append(r.Asserts, rest)
	case "capture":
		// The scope (file, global, request) is optional; every capture
		// lasts until the board's state is cleared.
		if len(fields) > 0 && scopes[fields[0]] {
			fields = fields[1:]
		}
		if len(fields) >= 2 {
			r.Captures = append(r.Captures, Capture{Name: fields[0], Expr: strings.Join(fields[1:], " ")})
		}
	case "env":
		if fileWide && len(fields) > 0 {
			f.Env = fields[0]
		}
	case "file", "const", "global", "var", "request":
		if len(fields) == 0 {
			return
		}
		v := assignment(rest)
		if word == "request" || !fileWide {
			r.Vars = append(r.Vars, v)
		} else {
			f.Vars = append(f.Vars, v)
		}
	}
}

// scopes are the words a capture or a variable may start with.
var scopes = map[string]bool{"file": true, "global": true, "request": true, "const": true}

// assignment reads "name = value" or "name value".
func assignment(s string) Var {
	if k, v, ok := strings.Cut(s, "="); ok {
		return Var{Name: strings.TrimSpace(k), Value: strings.TrimSpace(v)}
	}
	k, v, _ := strings.Cut(strings.TrimSpace(s), " ")
	return Var{Name: k, Value: strings.TrimSpace(v)}
}

// requestLine reads "POST {{base}}/login HTTP/1.1", or a bare URL, which
// is a GET.
func requestLine(t string) (method, url string) {
	fields := strings.Fields(t)
	if len(fields) == 0 {
		return "", ""
	}
	if methods[strings.ToUpper(fields[0])] {
		if len(fields) < 2 {
			return "", ""
		}
		return strings.ToUpper(fields[0]), fields[1]
	}
	if strings.Contains(fields[0], "://") || strings.HasPrefix(fields[0], "/") || strings.HasPrefix(fields[0], "{{") {
		return "GET", fields[0]
	}
	return "", ""
}
