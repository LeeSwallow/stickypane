package main

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/adapter/store"
	"github.com/LeeSwallow/stickypane/internal/core/doc"
	"github.com/LeeSwallow/stickypane/internal/core/when"
	"github.com/LeeSwallow/stickypane/internal/core/widget/form"
)

// codexArgs is one codex exec run of a case, the prompt on standard input.
// The board of a worktree lives in the main one, hence --add-dir.
func codexArgs(repo, cwd, arm, model string) []string {
	args := []string{"exec", "--json", "--ephemeral", "-s", "workspace-write", "-C", cwd, "--add-dir", repo}
	if arm == "without" {
		args = append(args, "--disable", "plugins")
	}
	if model != "" {
		args = append(args, "-m", model)
	}
	return append(args, "-")
}

// skillRead is how Codex opens a skill: it reads the file.
var skillRead = regexp.MustCompile(`skills/([a-z-]+)/SKILL\.md`)

// codexEvents turns codex exec --json lines into Events. Codex stamps no
// times, so a command is placed when its start was read, which is why
// this reads the stream as it comes.
func codexEvents(r io.Reader, now func() time.Time) ([]Event, Usage, error) {
	var events []Event
	var use Usage
	started := map[string]time.Time{}
	sc := bufio.NewScanner(r)
	sc.Buffer(nil, 16<<20)
	for sc.Scan() {
		t := now()
		var line struct {
			Type string `json:"type"`
			Item struct {
				ID      string `json:"id"`
				Type    string `json:"type"`
				Command string `json:"command"`
				Exit    *int   `json:"exit_code"`
				Text    string `json:"text"`
				Changes []struct {
					Path string `json:"path"`
				} `json:"changes"`
			} `json:"item"`
			Usage struct {
				Input  int `json:"input_tokens"`
				Cached int `json:"cached_input_tokens"`
				Output int `json:"output_tokens"`
			} `json:"usage"`
		}
		if json.Unmarshal(sc.Bytes(), &line) != nil {
			continue // not an event: Codex's own words
		}
		it := line.Item
		switch line.Type {
		case "item.started":
			started[it.ID] = t
		case "item.completed":
			if s, ok := started[it.ID]; ok {
				t = s
			}
			switch it.Type {
			case "command_execution":
				e := Event{At: t, Kind: "command", Text: it.Command}
				if it.Exit != nil {
					e.Exit = *it.Exit
				}
				events = append(events, e)
				if m := skillRead.FindStringSubmatch(it.Command); m != nil {
					events = append(events, Event{At: t, Kind: "skill", Text: m[1]})
				}
			case "file_change":
				for _, c := range it.Changes {
					events = append(events, Event{At: t, Kind: "write", Text: c.Path})
				}
			case "agent_message":
				events = append(events, Event{At: t, Kind: "message", Text: it.Text})
			}
		case "turn.completed":
			use.Input += line.Usage.Input
			use.Cached += line.Usage.Cached
			use.Output += line.Usage.Output
		}
	}
	return events, use, sc.Err()
}

// presses are the cases where the user says yes: a stand-in presses the
// first button of each form the agent makes.
var presses = map[string]bool{"08-approval": true}

var buttonRe = regexp.MustCompile(`\[ ([^\]]+?) \]`)

// press stands in for the user until ctx ends: a form the agent made gets
// its first button pressed, through the store as the board would.
func press(ctx context.Context, sticky string, before map[string]bool, now func() time.Time) {
	s := store.Open(sticky)
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
		for path := range files(sticky) {
			b, err := s.Read(path)
			if before[path] || err != nil {
				continue
			}
			d := doc.Parse(b)
			if !form.Open(d) {
				continue
			}
			lines := strings.Split(strings.TrimRight(d.Body, "\n"), "\n")
			if m := buttonRe.FindStringSubmatch(lines[len(lines)-1]); m != nil {
				_ = s.Apply(path, form.Submit{Label: m[1], At: now().Format(when.Moment)})
			}
		}
	}
}

// runCodex runs a case once in a new repository: the case's scaffold,
// then codex exec, then the graders on what it did and left.
func runCodex(c Case, arm, model string) (Result, error) {
	tmp, err := os.MkdirTemp("", "stickyeval-")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(tmp)
	repo := filepath.Join(tmp, "repo")
	env := append(os.Environ(), "GIT_AUTHOR_NAME=eval", "GIT_AUTHOR_EMAIL=eval@example.invalid",
		"GIT_COMMITTER_NAME=eval", "GIT_COMMITTER_EMAIL=eval@example.invalid")
	sh := func(name string, args ...string) error {
		cmd := exec.Command(name, args...)
		cmd.Dir, cmd.Env = repo, env
		_, err := cmd.CombinedOutput()
		return err
	}
	if err := os.Mkdir(repo, 0o755); err != nil {
		return Result{}, err
	}
	if err := sh("git", "init", "-q"); err != nil {
		return Result{}, err
	}
	if c.Context.Scaffold != "" {
		script, _ := filepath.Abs(filepath.Join(c.Dir, c.Context.Scaffold))
		if err := sh("bash", script); err != nil {
			return Result{}, err
		}
	}
	cwd := repo
	if b, err := os.ReadFile(filepath.Join(repo, ".eval-cwd")); err == nil {
		cwd = filepath.Join(repo, strings.TrimSpace(string(b)))
	}
	sticky := filepath.Join(repo, ".sticky")
	before := files(sticky)

	timeout := time.Duration(c.Execution.Timeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if presses[c.Name] {
		go press(ctx, sticky, before, time.Now)
	}
	cmd := exec.CommandContext(ctx, "codex", codexArgs(repo, cwd, arm, model)...)
	cmd.Dir, cmd.Env = cwd, env
	cmd.Stdin = strings.NewReader(c.Execution.Prompt)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return Result{}, err
	}
	if err := cmd.Start(); err != nil {
		return Result{}, err
	}
	events, use, _ := codexEvents(stdout, time.Now)
	_ = cmd.Wait() // a run cut off by its time is still graded
	r := grade(Run{Case: c.Name, Events: events, Sticky: sticky, Before: before})
	r.Agent, r.Arm, r.Use = "codex", arm, use
	return r, nil
}
