package httpfile

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/when"
)

// maxLogLines is how many lines of a response body the log keeps.
const maxLogLines = 300

// Curl writes a request as a curl command, so that it can be sent again by
// hand. Values are quoted for a POSIX shell. Send hides credentials in the
// headers it hands in, so a copied command needs them filled back in.
func Curl(method, url string, header [][2]string, body string) string {
	parts := []string{"curl", "-sS"}
	if method != "GET" || body != "" {
		parts = append(parts, "-X", method)
	}
	parts = append(parts, quote(url))
	for _, h := range header {
		parts = append(parts, "-H", quote(h[0]+": "+h[1]))
	}
	if body != "" {
		parts = append(parts, "--data-raw", quote(body))
	}
	return strings.Join(parts, " ")
}

func quote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

// EndLine matches the last line Log writes: "[200 OK · 38ms · login · 2/2 ✔]"
// or "[error · …]". The board reads how the request ended from it.
var EndLine = regexp.MustCompile(`^\[(\d{3}|error)\b(.*)\]$`)

// Log writes what sending a request came to, for the board's panel and for
// the agent: the request as curl, the body, the checks, and how it ended.
func Log(r Result) string {
	var b strings.Builder
	if r.Curl != "" {
		// A curl line spans one line on the board, as a script's command does.
		fmt.Fprintf(&b, "$ %s\n", strings.ReplaceAll(r.Curl, "\n", `\n`))
	} else {
		fmt.Fprintf(&b, "$ %s %s\n", r.Request.Method, r.Request.URL)
	}
	switch {
	case r.Protocol == "websocket":
		for _, f := range r.Frames {
			b.WriteString(frameLine(f) + "\n")
		}
	case r.GRPC != nil && r.GRPC.Schema != "" && r.GRPC.Schema != "descriptor":
		fmt.Fprintf(&b, "descriptors: %s\n", r.GRPC.Schema)
	}
	if len(r.Body) > 0 && r.Protocol != "websocket" {
		body := pretty(r.Body)
		ls := strings.Split(strings.TrimRight(body, "\n"), "\n")
		if len(ls) > maxLogLines {
			ls = append(ls[:maxLogLines], fmt.Sprintf("… %d more lines", len(ls)-maxLogLines))
		}
		b.WriteString(strings.Join(ls, "\n") + "\n")
	}
	if len(r.Checks)+len(r.Captured)+len(r.Hooks) > 0 {
		b.WriteString("\n")
	}
	for _, c := range r.Checks {
		if c.OK {
			fmt.Fprintf(&b, "✔ %s\n", c.Expr)
		} else {
			fmt.Fprintf(&b, "✘ %s · got %s\n", c.Expr, c.Got)
		}
	}
	for _, name := range r.Captured {
		fmt.Fprintf(&b, "→ captured %s\n", name)
	}
	for _, l := range r.Hooks {
		fmt.Fprintf(&b, "%s\n", l)
	}
	b.WriteString(End(r) + "\n")
	return b.String()
}

// End is the last line of the log: how the request ended, in one line.
func End(r Result) string {
	parts := []string{}
	if r.Status > 0 {
		parts = append(parts, r.Text, took(r.Took))
	} else {
		parts = append(parts, "error")
	}
	parts = append(parts, r.Request.Title())
	if sent, received := r.Counts(); r.Protocol == "websocket" && r.Status > 0 {
		parts = append(parts, fmt.Sprintf("%d↑ %d↓", sent, received))
	}
	if r.GRPC != nil && r.GRPC.Code != 0 && r.GRPC.Message != "" {
		parts = append(parts, r.GRPC.Message)
	}
	if r.Env != "" {
		parts = append(parts, r.Env)
	}
	if n := len(r.Checks); n > 0 {
		mark := "✔"
		if r.Passed() < n {
			mark = "✘"
		}
		parts = append(parts, fmt.Sprintf("%d/%d %s", r.Passed(), n, mark))
	}
	if r.Err != nil {
		parts = append(parts, strings.ReplaceAll(r.Err.Error(), "\n", " "))
	}
	return "[" + strings.Join(parts, " · ") + "]"
}

// frameLine writes one line of a WebSocket transcript: which way, when to
// the millisecond, and what.
func frameLine(f Frame) string {
	dir := "←"
	if f.Out {
		dir = "→"
	}
	if f.Kind == "wait" {
		return "· wait " + f.Data
	}
	at := f.At.Format("15:04:05.000")
	switch f.Kind {
	case "text":
		return dir + " " + at + "  " + strings.ReplaceAll(f.Data, "\n", " ")
	case "close":
		return strings.TrimSpace(fmt.Sprintf("%s %s  close %d %s", dir, at, f.Code, f.Data))
	}
	return strings.TrimSpace(dir + " " + at + "  " + f.Kind + " " + f.Data)
}

// took writes how long a request took: milliseconds under a second, as
// requests mostly are, and else as every other duration on the board.
func took(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return when.Duration(d)
}

// pretty indents a JSON body and leaves anything else as it is.
func pretty(body []byte) string {
	var out bytes.Buffer
	if json.Valid(body) && json.Indent(&out, body, "", "  ") == nil {
		return out.String()
	}
	return string(body)
}

// LoadSaved reads the values captured for each .http file, from the file at
// path. A missing file holds nothing.
func LoadSaved(path string) (map[string]map[string]string, error) {
	all := map[string]map[string]string{}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return all, nil
	}
	if err != nil {
		return all, err
	}
	if err := json.Unmarshal(data, &all); err != nil {
		return map[string]map[string]string{}, nil // a broken file is started again
	}
	return all, nil
}

// SaveSaved writes back what LoadSaved read, with the values of one file
// replaced.
func SaveSaved(path, file string, vars map[string]string) error {
	all, err := LoadSaved(path)
	if err != nil {
		return err
	}
	all[file] = vars
	data, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

// Notes is the part of the notes folder SendNote needs.
type Notes interface {
	Read(name string) ([]byte, error)
	Write(name string, content []byte) error
}

// SavedFile is where the values captured from responses are kept, inside
// the notes folder, so that the next request, sent from the board or the
// command line, can use them.
const SavedFile = ".http-vars.json"

// LogName is the log of a .http file: the same name with ".log", which the
// board shows under the requests.
func LogName(name string) string { return strings.TrimSuffix(name, path.Ext(name)) + ".log" }

// SendNote sends the request part of the .http note name in the notes
// folder dir, writes its log and keeps what it captured. env chooses the
// environment; empty takes the file's. The board and the command line
// share it.
func SendNote(ctx context.Context, notes Notes, dir, name string, part int, env string) (Result, error) {
	data, err := notes.Read(name)
	if err != nil {
		return Result{}, err
	}
	f := Parse(string(data))
	if part < 0 || part >= len(f.Requests) {
		return Result{}, fmt.Errorf("%s has no request %d", name, part+1)
	}
	root := filepath.Dir(dir)
	saved := filepath.Join(dir, SavedFile)
	all, _ := LoadSaved(saved)
	here := filepath.Dir(filepath.Join(dir, filepath.FromSlash(name)))
	rn := &Runner{
		Root:  root,
		Here:  here,
		Dirs:  []string{here, dir, root},
		Env:   env,
		Saved: all[name],
	}
	res := rn.Send(ctx, f, part)
	if len(res.Captured) > 0 {
		if err := SaveSaved(saved, name, rn.Saved); err != nil {
			return res, err
		}
	}
	order := make([]string, len(f.Requests))
	for i, r := range f.Requests {
		order[i] = r.Title()
	}
	old, _ := notes.Read(LogName(name))
	return res, notes.Write(LogName(name), []byte(PutSection(string(old), res.Request.Title(), Log(res), order)))
}

// sectionHead starts a request's part of a .http file's log.
const sectionHead = "### "

// sections splits a log into its requests' parts, by title. A log with no
// "### " line has none.
func sections(log string) (titles []string, parts map[string]string) {
	parts = map[string]string{}
	var title string
	var b strings.Builder
	flush := func() {
		if title != "" {
			parts[title] = strings.TrimRight(b.String(), "\n") + "\n"
			titles = append(titles, title)
		}
		b.Reset()
	}
	for _, l := range strings.Split(log, "\n") {
		if strings.HasPrefix(l, sectionHead) {
			flush()
			title = strings.TrimSpace(strings.TrimPrefix(l, sectionHead))
			continue
		}
		if title != "" {
			b.WriteString(l + "\n")
		}
	}
	flush()
	return titles, parts
}

// Section is the part of a .http file's log that a request wrote when it
// was last sent: from its "### title" line to the next.
func Section(log, title string) (string, bool) {
	_, parts := sections(log)
	s, ok := parts[title]
	return s, ok
}

// HasSections reports whether a log is written in sections, one per request.
func HasSections(log string) bool {
	titles, _ := sections(log)
	return len(titles) > 0
}

// PutSection puts what a request got into the log under its title,
// replacing what it got last time and keeping the other requests' parts,
// in the order of order (the file's requests). A log from before sections
// is dropped.
func PutSection(log, title, content string, order []string) string {
	titles, parts := sections(log)
	parts[title] = strings.TrimRight(content, "\n") + "\n"
	seen := map[string]bool{}
	var b strings.Builder
	write := func(t string) {
		if seen[t] {
			return
		}
		if p, ok := parts[t]; ok {
			seen[t] = true
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString(sectionHead + t + "\n" + p)
		}
	}
	for _, t := range order {
		write(t)
	}
	for _, t := range append(titles, title) { // requests renamed or gone since
		write(t)
	}
	return b.String()
}
