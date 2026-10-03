package api

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
		titles := make([]string, len(f.Requests))
		for i, r := range f.Requests {
			titles[i] = r.Title()
		}
		i, err := widget.Pick(titles, request, "request")
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
		res, err := httpfile.SendNote(ctx, a.st, a.st.Dir, name, i, env)
		if err != nil {
			return b.String(), failed, err
		}
		if n > 0 {
			b.WriteString("\n")
		}
		b.WriteString(strings.TrimPrefix(httpfile.Log(res), "$ "))
		if res.Failed() {
			failed++
		}
	}
	if failed > 0 {
		return b.String(), failed, fmt.Errorf("%d of %d requests failed; the last one's log is in %s", failed, len(parts), httpfile.LogName(name))
	}
	return b.String(), 0, nil
}
