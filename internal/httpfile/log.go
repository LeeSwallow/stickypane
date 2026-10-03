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
	if len(r.Body) > 0 {
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
	rn := &Runner{
		Root:  root,
		Dirs:  []string{filepath.Dir(filepath.Join(dir, filepath.FromSlash(name))), dir, root},
		Env:   env,
		Saved: all[name],
	}
	res := rn.Send(ctx, f, part)
	if len(res.Captured) > 0 {
		if err := SaveSaved(saved, name, rn.Saved); err != nil {
			return res, err
		}
	}
	return res, notes.Write(LogName(name), []byte(Log(res)))
}
