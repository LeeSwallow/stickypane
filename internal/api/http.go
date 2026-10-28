package api

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/httpfile"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// HTTP lists the requests of a .http note, or sends one (named by its title,
// a part of it, or #2) or all of them, and returns what to show: the list,
// or each response's log with its checks and captures. failed counts the
// requests that failed. Experimental, as the .http note is.
func (a *API) HTTP(ctx context.Context, name, request, env string, all bool) (out string, failed int, err error) {
	name = strings.TrimSpace(name)
	if filepath.Ext(name) == "" {
		for _, ext := range []string{".http", ".rest"} {
			if _, err := os.Stat(filepath.Join(a.st.Dir, filepath.FromSlash(name+ext))); err == nil {
				name += ext
				break
			}
		}
	}
	data, err := a.st.Read(name)
	if err != nil {
		return "", 0, fmt.Errorf("cannot read %s: %w", name, err)
	}
	f := httpfile.Parse(string(data))
	if len(f.Requests) == 0 {
		return "", 0, fmt.Errorf("%s has no requests yet", name)
	}
	var parts []int
	switch {
	case all:
		for i := range f.Requests {
			parts = append(parts, i)
		}
	case request != "":
		i, err := pickRequest(f, request)
		if err != nil {
			return "", 0, err
		}
		parts = []int{i}
	default:
		var b strings.Builder
		for i, r := range f.Requests {
			where := ""
			if r.Protocol() == "grpc" && r.GRPC != nil {
				where = "  " + r.GRPC.Method
			}
			fmt.Fprintf(&b, "#%d  %-6s %s%s\n", i+1, r.Badge(), r.Title(), where)
		}
		return b.String(), 0, nil
	}
	var b strings.Builder
	for n, i := range parts {
		res, err := a.SendPart(ctx, name, i, env)
		if err != nil {
			return b.String(), failed, err
		}
		if n > 0 {
			b.WriteString("\n")
		}
		b.WriteString(strings.TrimPrefix(res.Said, "$ "))
		if res.Failed {
			failed++
		}
	}
	if failed > 0 {
		return b.String(), failed, fmt.Errorf("%d of %d requests failed; the last one's log is in %s", failed, len(parts), httpfile.LogName(name))
	}
	return b.String(), 0, nil
}

// Sent is how one request of a .http note ended, for a screen to say.
type Sent struct {
	Status int           // the HTTP status, 0 when none came
	Text   string        // "200 OK"
	Took   time.Duration // how long it took
	Failed bool          // not sent, an error status, a hook or a check that failed
	End    string        // the last line of its log, without its brackets
	Said   string        // the whole log of this request
	Log    string        // the note that holds the log
}

// Request is the place among a .http note's requests of the one named by
// want: its title, a part of it, or "#2".
func (a *API) Request(name, want string) (int, error) {
	data, err := a.st.Read(name)
	if err != nil {
		return 0, err
	}
	return pickRequest(httpfile.Parse(string(data)), want)
}

func pickRequest(f httpfile.File, want string) (int, error) {
	titles := make([]string, len(f.Requests))
	for i, r := range f.Requests {
		titles[i] = r.Title()
	}
	return widget.Pick(titles, want, "request")
}

// SendPart sends request part (from 0) of the .http note name with the
// variables of env ("" for the file's own) and writes its answer into the
// note's log. It is the one way a request is sent: by the terminal after
// the user said yes, by the bridge for the desktop app, and by HTTP for the
// command line and MCP. The error is about the log not being written; a
// request that failed is Sent.Failed.
func (a *API) SendPart(ctx context.Context, name string, part int, env string) (Sent, error) {
	res, err := httpfile.SendNote(ctx, a.st, a.st.Dir, name, part, env)
	return Sent{
		Status: res.Status, Text: res.Text, Took: res.Took, Failed: res.Failed(),
		End: strings.Trim(httpfile.End(res), "[]"), Said: httpfile.Log(res), Log: httpfile.LogName(name),
	}, err
}
