// Package mcp serves the notes folder to agents over the Model Context
// Protocol: JSON-RPC 2.0 messages, one per line, on standard input and output.
// It implements initialize, ping, the tools (tools/list, tools/call) and
// the plugin's skills as prompts (prompts/list, prompts/get).
//
// What it offers is layered as the plugin is: the guide tool is the way in
// and names the skills; a skill, as a prompt or as the guide's topic, says
// which tools to call for a job; the tools change one thing each.
package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LeeSwallow/stickypane/internal/adapter/git"
	"github.com/LeeSwallow/stickypane/internal/adapter/registry"
	"github.com/LeeSwallow/stickypane/internal/core/i18n"
	"github.com/LeeSwallow/stickypane/internal/usecase/api"
	"github.com/LeeSwallow/stickypane/internal/usecase/diffbook"
	"github.com/LeeSwallow/stickypane/internal/usecase/worktree"
)

// defaultProtocol is answered when the client does not name a version.
const defaultProtocol = "2025-06-18"

const (
	maxWait          = 600 // seconds read_answers and wait_event wait at most
	defaultEventWait = 60  // seconds wait_event waits when not told
	waitEvery        = 200 * time.Millisecond
)

// jsonrpcVersion is the version every response names.
const jsonrpcVersion = "2.0"

// The methods the server branches on before it answers.
const (
	methodToolsCall = "tools/call"
	methodCancelled = "notifications/cancelled"
)

// The tools that may wait for the user, so run on the side (mayWait).
const (
	toolReadAnswers = "read_answers"
	toolWaitEvent   = "wait_event"
)

// JSON-RPC error codes.
const (
	codeParse    = -32700
	codeNoMethod = -32601
	codeParams   = -32602
)

// Server answers MCP requests for one notes folder.
type Server struct {
	// API is the board the tools work on. A server may start before the
	// project has a board: then API has no store (api.New(nil, reg)), which
	// is enough for the guide, and Open is asked for the board.
	API *api.API
	// Open finds the board, or makes it when write is set and the project
	// can have one; its error is what a tool that needs the board returns.
	Open    func(write bool) (*api.API, error)
	Guide   string // the short agent guide, which the "guide" tool opens with
	Version string

	mu sync.Mutex // guards API while Open has not found the board
}

type request struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type toolResult struct {
	Content []content `json:"content"`
	IsError bool      `json:"isError"`
}

type tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func object(required []string, props map[string]any) map[string]any {
	schema := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func str(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

// tools is what tools/list returns, in this order.
var tools = []tool{
	{
		Name:        "guide",
		Description: "Start here. Without a topic: how the board works, the skills (which to follow for a job: tracking progress, asking the user, talking in a chat, connecting notes) and the shapes of note. With a topic: that skill in full, or that shape's file format with an example.",
		InputSchema: object(nil, map[string]any{
			"topic": str(`A topic, such as "tracking-the-work", a recipe of one, such as "approving-a-risky-action/several-at-once", or a shape, such as "board". Omit for the overview.`),
		}),
	},
	{
		Name:        "note",
		Description: "One note by its action. read: its Markdown file. write: create or replace it with content (read it first: the user may have changed it; how they arranged it is kept unless you set it). set: change front matter keys only. list: every note in a line, by tab, with what it counts and a gist; read this instead of every note. show: put it, or any file or folder of the project, in front of the user. hide: fold it away. link: show a file or folder of the project as a note. mv: rename it or move it into a tab or book. rm: to the trash. archive: out of sight. restore: back from the trash.",
		InputSchema: object([]string{"action"}, map[string]any{
			"action":  map[string]any{"type": "string", "enum": []string{"read", "write", "set", "list", "show", "hide", "link", "mv", "rm", "archive", "restore"}},
			"name":    str(`The note's file name, such as "plan", "plan.md" or "docs/plan"; for show and link, also a path such as "README.md" or "docs/". Prefix a number ("10-plan") to control the order.`),
			"content": str("For write: the whole note as Markdown, which may start with front matter. The guide tool gives the formats of boards, checklists, logs, charts and forms."),
			"type":    map[string]any{"type": "string", "enum": []string{"note", "board", "checklist", "log", "chart", "form", "chat"}, "description": "For write: the note's shape. Omit for a plain note."},
			"title":   str("For write: shown in the title bar and the note's border."),
			"open":    map[string]any{"type": "boolean", "description": "For write: true puts the note on the screen now."},
			"size":    map[string]any{"type": "string", "enum": []string{"1/4", "1/3", "1/2", "2/3", "3/4", "1", "card"}, "description": "For write: the width as a fraction of the screen; card is the narrowest that fits a small sticky note."},
			"keys":    map[string]any{"type": "object", "description": `For set: keys and their new values, such as {"open": true, "size": "1/2"}: title, open, size, rows, color, pin, view. An empty value removes the key.`},
			"to":      str(`For mv: a new name ("roadmap"), a folder ending in "/" ("docs/"), or "." for the top level. For link: the note's name (optional).`),
		}),
	},
	{
		Name:        "todo",
		Description: "Change a checklist without reading or rewriting it: add an item, or check or uncheck one. A checklist that does not exist yet is made. Returns the progress, such as \"plan.md: 2/5\".",
		InputSchema: object([]string{"name", "action", "item"}, map[string]any{
			"name":   str(`The checklist's file name, such as "plan".`),
			"action": map[string]any{"type": "string", "enum": []string{"add", "check", "uncheck"}},
			"item":   str(`For add, the new item's text. For check and uncheck, the item's text, a part of it that no other item has, or its position such as "#2".`),
		}),
	},
	{
		Name:        "card",
		Description: "Change a board without reading or rewriting it: add a card to a column, or move a card to another column. A board that does not exist yet is made.",
		InputSchema: object([]string{"name", "action", "card"}, map[string]any{
			"name":   str(`The board's file name, such as "work".`),
			"action": map[string]any{"type": "string", "enum": []string{"add", "move"}},
			"card":   str(`For add, the new card's text. For move, the card's text, a part of it that no other card has, or its position such as "#2".`),
			"to":     str("The column, by its heading or a part of it. Needed for move; for add the first column is used when it is left out, and a column that does not exist is made."),
		}),
	},
	{
		Name:        "chart",
		Description: "Change one value of a chart without reading or rewriting it: set a value, or count it up or down. A chart or a label that does not exist yet is made.",
		InputSchema: object([]string{"name", "action", "label", "value"}, map[string]any{
			"name":   str(`The chart's file name, such as "tokens".`),
			"action": map[string]any{"type": "string", "enum": []string{"set", "add"}, "description": "set writes the value; add counts the current value up by it, or down when it is negative."},
			"label":  str("Which value."),
			"value":  map[string]any{"type": []string{"number", "string"}, "description": "The number."},
		}),
	},
	{
		Name:        "log",
		Description: "Add one line at the end of a log note. A log that does not exist yet is made.",
		InputSchema: object([]string{"name", "line"}, map[string]any{
			"action": map[string]any{"type": "string", "enum": []string{"add"}, "description": "Default add."},
			"name":   str(`The log's file name, such as "worklog".`),
			"line":   str("The entry, one line."),
		}),
	},
	{
		Name:        "chat",
		Description: "Add a message to a chat note, the conversation with the user on the board. A chat that does not exist yet is made. The user answers there; wait_event with type message.added hears it.",
		InputSchema: object([]string{"action", "name", "text"}, map[string]any{
			"action": map[string]any{"type": "string", "enum": []string{"add"}},
			"name":   str(`The chat's file name, such as "chat".`),
			"text":   str("The message, one line."),
			"as":     str(`Who says it, such as "claude". Default "agent".`),
		}),
	},
	{
		Name:        "page",
		Description: "A book, a folder in a tab, page by page. add: a page named after text, as N on the board makes it; note write fills it. list: its pages in order, as JSON.",
		InputSchema: object([]string{"action", "name"}, map[string]any{
			"action": map[string]any{"type": "string", "enum": []string{"add", "list"}},
			"name":   str(`The book, such as "deploy/docs".`),
			"text":   str("For add: the page's words, which name it."),
		}),
	},
	{
		Name:        "worktree",
		Description: "The git worktrees the board follows, each with a folder worktrees/<name>/, as JSON. list: all, with state, branch and forms waiting. show: one by name.",
		InputSchema: object([]string{"action"}, map[string]any{
			"action": map[string]any{"type": "string", "enum": []string{"list", "show"}},
			"name":   str("For show: the worktree's folder name."),
		}),
	},
	{
		Name:        "diff",
		Description: "What the git worktrees changed, read from git; writes nothing. list: every worktree with the files and lines it changed and the files two of them changed, or with name the files that worktree changed, as JSON. show: one file's diff.",
		InputSchema: object([]string{"action"}, map[string]any{
			"action": map[string]any{"type": "string", "enum": []string{"list", "show"}},
			"name":   str("The worktree, by its branch. Needed for show."),
			"file":   str("For show: the file's path in the repository."),
		}),
	},
	{
		Name:        toolReadAnswers,
		Description: "Read what the user chose and wrote in a form note (type: form) and which button they pressed. With wait_seconds it waits up to that long for a button to be pressed, then returns the answers so far.",
		InputSchema: object([]string{"name"}, map[string]any{
			"name":         str(`The form's file name, such as "deploy" or "deploy.md".`),
			"wait_seconds": map[string]any{"type": "number", "description": "How long to wait for a button, at most 600. Omit to read the answers now."},
		}),
	},
	{
		Name:        toolWaitEvent,
		Description: "Wait for something to happen on the board and return it as JSON: a note made or removed, an item ticked, a card moved, a form sent, a line logged, a chart value changed. Use it to react to what the user does without being told.",
		InputSchema: object(nil, map[string]any{
			"notes":        str(`Only these notes or folders, comma-separated, such as "plan,deploy". Omit for all.`),
			"types":        str(`Only these events or their kinds, comma-separated, such as "item.ticked" or "card". Omit for all.`),
			"wait_seconds": map[string]any{"type": "number", "description": "How long to wait, at most 600. Default 60."},
		}),
	},
	{
		Name:        "api",
		Description: "Experimental. A .http note holds REST requests, WebSocket sessions (# @websocket, # @ws steps) and gRPC calls (GRPC host:port, # @grpc pkg.Service/Method), in resterm's syntax; `stickypane kinds api` or the guide topic \"api\" gives the format. With only name, list its requests; with request (its title, a part of it, or #2) send it, or with all send every one in order; returns each one's log: the status, the body or the WebSocket transcript or the gRPC messages, the checks and the captured values, as the board shows them. A failed run still returns the logs.",
		InputSchema: object([]string{"name"}, map[string]any{
			"name":    str(`The .http note, such as "api" or "api.http".`),
			"request": str(`The request to send: its title, a part of it, or "#2".`),
			"env":     str("The environment of the note to use (optional)."),
			"all":     map[string]any{"type": "boolean", "description": "Send every request, in order."},
		}),
	},
}

// Serve answers requests from in until it ends. A bad line gets an error
// response and does not stop the server.
//
// Requests are answered in order, except a call that waits for the user
// (read_answers): it runs on the side, so other requests are answered while
// it waits. Such a call ends early, with the answers so far, when the client
// cancels it or the input ends.
func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	r := bufio.NewReader(in)
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)

	var mu sync.Mutex // guards enc, werr and waits
	var werr error
	send := func(resp *response) {
		mu.Lock()
		defer mu.Unlock()
		if resp != nil && werr == nil {
			werr = enc.Encode(resp)
		}
	}
	ctx, stop := context.WithCancel(ctx)
	waits := map[string]context.CancelFunc{} // running waits by request id
	var running sync.WaitGroup
	finish := func() error {
		stop()
		running.Wait()
		mu.Lock()
		defer mu.Unlock()
		return werr
	}

	for {
		line, err := r.ReadBytes('\n')
		if len(bytes.TrimSpace(line)) > 0 {
			var req request
			_ = json.Unmarshal(line, &req)
			id := string(bytes.TrimSpace(req.ID))
			switch {
			case req.Method == methodCancelled:
				var p struct {
					RequestID json.RawMessage `json:"requestId"`
				}
				_ = json.Unmarshal(req.Params, &p)
				mu.Lock()
				if cancel, ok := waits[string(bytes.TrimSpace(p.RequestID))]; ok {
					cancel()
				}
				mu.Unlock()
			case req.ID != nil && mayWait(req):
				wctx, cancel := context.WithCancel(ctx)
				mu.Lock()
				waits[id] = cancel
				mu.Unlock()
				running.Add(1)
				go func() {
					defer running.Done()
					resp := s.handle(wctx, line)
					mu.Lock()
					delete(waits, id)
					mu.Unlock()
					cancel()
					send(resp)
				}()
			default:
				send(s.handle(ctx, line))
			}
		}
		mu.Lock()
		failed := werr
		mu.Unlock()
		switch {
		case failed != nil:
			return finish()
		case err == io.EOF:
			return finish()
		case err != nil:
			_ = finish()
			return err
		}
	}
}

// plain writes a JSON value the way a note's file would: a number without
// an exponent, true and false as words, null as nothing.
func plain(v any) string {
	switch v := v.(type) {
	case nil:
		return ""
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
	return fmt.Sprint(v)
}

// mayWait reports whether a request is a tool call that may wait for the user.
func mayWait(req request) bool {
	if req.Method != methodToolsCall {
		return false
	}
	var p struct {
		Name string `json:"name"`
	}
	_ = json.Unmarshal(req.Params, &p)
	return p.Name == toolReadAnswers || p.Name == toolWaitEvent
}

// handle answers one message. Notifications, which carry no id, get no answer.
func (s *Server) handle(ctx context.Context, line []byte) *response {
	var req request
	if err := json.Unmarshal(line, &req); err != nil {
		return &response{JSONRPC: jsonrpcVersion, ID: json.RawMessage("null"), Error: &rpcError{codeParse, "the message is not valid JSON"}}
	}
	if req.ID == nil {
		return nil
	}
	resp := &response{JSONRPC: jsonrpcVersion, ID: req.ID}
	switch req.Method {
	case "initialize":
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		_ = json.Unmarshal(req.Params, &p)
		if p.ProtocolVersion == "" {
			p.ProtocolVersion = defaultProtocol
		}
		resp.Result = map[string]any{
			"protocolVersion": p.ProtocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{}, "prompts": map[string]any{}},
			"serverInfo":      map[string]any{"name": "stickypane", "version": s.Version},
		}
	case "ping":
		resp.Result = struct{}{}
	case "tools/list":
		resp.Result = map[string]any{"tools": tools}
	case "prompts/list":
		resp.Result = map[string]any{"prompts": prompts()}
	case "prompts/get":
		resp.Result, resp.Error = getPrompt(req.Params)
		if resp.Error != nil {
			resp.Result = nil
		}
	case methodToolsCall:
		result, rpcErr := s.call(ctx, req.Params)
		resp.Result, resp.Error = result, rpcErr
		if rpcErr != nil {
			resp.Result = nil
		}
	default:
		resp.Error = &rpcError{codeNoMethod, "unknown method " + req.Method}
	}
	return resp
}

// args are the arguments of every tool, each reading those it needs.
type args struct {
	Name    string         `json:"name"`
	Content *string        `json:"content"`
	Type    string         `json:"type"`
	Title   string         `json:"title"`
	Size    string         `json:"size"`
	Open    bool           `json:"open"`
	Wait    float64        `json:"wait_seconds"`
	Action  string         `json:"action"`
	Item    string         `json:"item"`
	Card    string         `json:"card"`
	To      string         `json:"to"`
	Label   string         `json:"label"`
	Value   any            `json:"value"`
	Line    string         `json:"line"`
	Text    string         `json:"text"`
	As      string         `json:"as"`
	Keys    map[string]any `json:"keys"`
	Notes   string         `json:"notes"`
	Request string         `json:"request"`
	Env     string         `json:"env"`
	All     bool           `json:"all"`
	Types   string         `json:"types"`
	Topic   string         `json:"topic"`
	File    string         `json:"file"`
}

// oldTools are the tools before every tool was a <noun> + action (#162),
// and the tool and action each stands for. They are listed nowhere and
// work as ever, for a client that calls them by name.
var oldTools = map[string][2]string{
	"read_note": {"note", "read"}, "write_note": {"note", "write"}, "set_keys": {"note", "set"},
	"index": {"note", "list"}, "say": {"chat", "add"},
}

// arranged are the actions of the old arrange_note and note's for them.
var arranged = map[string]string{"show": "show", "hide": "hide", "move": "mv", "remove": "rm", "restore": "restore"}

// call runs a tool. A tool that fails returns a result marked isError, which
// the agent can read and act on; only an unknown tool or unreadable
// arguments is a protocol error.
func (s *Server) call(ctx context.Context, params json.RawMessage) (*toolResult, *rpcError) {
	var p struct {
		Name      string `json:"name"`
		Arguments args   `json:"arguments"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &rpcError{codeParams, "cannot read the tool arguments: " + err.Error()}
	}
	a := &p.Arguments
	if old, ok := oldTools[p.Name]; ok {
		p.Name, a.Action = old[0], old[1]
	}
	if p.Name == "arrange_note" {
		action, ok := arranged[a.Action]
		if !ok {
			return failed(fmt.Errorf("unknown action %q: show, hide, move, remove or restore", a.Action)), nil
		}
		p.Name, a.Action = "note", action
	}
	if p.Name == "log" && a.Action == "" {
		a.Action = "add"
	}
	if !slices.ContainsFunc(tools, func(t tool) bool { return t.Name == p.Name }) {
		return nil, &rpcError{codeParams, "unknown tool " + p.Name}
	}
	on, write := onBoard(p.Name, a.Action)
	b, err := s.board(on, write)
	if err != nil {
		return failed(err), nil
	}
	text := ""
	switch p.Name {
	case "note":
		text, err = noteTool(b, *a)
	case "todo":
		text, err = said(b.Todo(a.Name, a.Action, a.Item))(status)
	case "card":
		text, err = said(b.Card(a.Name, a.Action, a.Card, a.To))(status)
	case "chart":
		text, err = said(b.Chart(a.Name, a.Action, a.Label, plain(a.Value)))(valued)
	case "log":
		if err = only(a.Action, "add"); err == nil {
			text, err = said(b.Log(a.Name, a.Line))(status)
		}
	case "chat":
		if err = only(a.Action, "add"); err == nil {
			text, err = said(b.Say(a.Name, a.As, a.Text))(status)
		}
	case "page":
		text, err = pageTool(b, *a)
	case "worktree":
		text, err = worktreeTool(b, *a)
	case "diff":
		text, err = diffTool(*a)
	case toolReadAnswers:
		wctx, cancel := context.WithTimeout(ctx, time.Duration(min(max(a.Wait, 0), maxWait)*float64(time.Second)))
		got, aerr := b.Wait(wctx, a.Name, waitEvery)
		cancel()
		if errors.Is(aerr, context.DeadlineExceeded) || errors.Is(aerr, context.Canceled) {
			aerr = nil // the answers so far are the result
		}
		text, err = got.String(), aerr
	case "api":
		text, _, err = b.HTTP(ctx, a.Name, a.Request, a.Env, a.All)
	case toolWaitEvent:
		wait := a.Wait
		if wait <= 0 {
			wait = defaultEventWait
		}
		wctx, cancel := context.WithTimeout(ctx, time.Duration(min(wait, maxWait)*float64(time.Second)))
		f := api.Filter{Notes: split(a.Notes), Types: split(a.Types)}
		var got *api.Event
		werr := b.Watch(wctx, f, func(e api.Event) error {
			got = &e
			return errFound
		}, nil)
		cancel()
		switch {
		case got != nil:
			var b []byte
			if b, err = json.Marshal(got); err == nil {
				text = string(b)
			}
		case errors.Is(werr, context.DeadlineExceeded) || errors.Is(werr, context.Canceled):
			text = "nothing happened"
		default:
			err = werr
		}
	case "guide":
		text, err = b.Guide(a.Topic, s.Guide)
	}
	if err != nil {
		// What was done before it failed is kept: the logs of the requests
		// that were sent are what the agent needs to see why.
		msg := err.Error()
		if strings.TrimSpace(text) != "" {
			msg = strings.TrimRight(text, "\n") + "\n\n" + msg
		}
		return failed(errors.New(msg)), nil
	}
	return &toolResult{Content: []content{{Type: "text", Text: text}}}, nil
}

// failed is the result of a tool that failed.
func failed(err error) *toolResult {
	return &toolResult{Content: []content{{Type: "text", Text: err.Error()}}, IsError: true}
}

// only fails for an action other than the one a tool has.
func only(action, want string) error {
	if action != want {
		return fmt.Errorf("unknown action %q: %s", action, want)
	}
	return nil
}

// noteTool runs the note tool.
func noteTool(b *api.API, a args) (string, error) {
	switch a.Action {
	case "read":
		body, err := b.Cat(a.Name)
		return string(body), err
	case "write":
		if a.Content == nil {
			// Without this check a call that forgot its content would
			// replace the note with nothing and report success.
			return "", errors.New("write needs content: the whole note as Markdown")
		}
		file, err := b.Write(a.Name, api.Options{Type: a.Type, Title: a.Title, Size: a.Size, Open: a.Open}, []byte(*a.Content))
		return "wrote " + file, err
	case "set":
		keys := make([]string, 0, len(a.Keys))
		for k := range a.Keys {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		pairs := make([]string, len(keys))
		for i, k := range keys {
			pairs[i] = k + "=" + plain(a.Keys[k])
		}
		return said(b.Set(a.Name, pairs))(keyed)
	case "list":
		idx, err := b.Index()
		if err != nil {
			return "", err
		}
		return api.IndexMarkdown(idx, "board"), nil
	case "show":
		return said(b.Show(a.Name))(shown)
	case "hide":
		return said(b.Hide(a.Name))(shown)
	case "link":
		return said(b.Link(a.Name, a.To))(linked)
	case "mv":
		if a.To == "" {
			return "", errors.New(`mv needs "to": a new name, a folder ending in "/", or "."`)
		}
		return said(b.Move(a.Name, a.To))(movedTo)
	case "rm":
		return said(b.Remove(a.Name))(removed)
	case "archive":
		return said(b.Archive(a.Name))(movedTo)
	case "restore":
		return said(b.Restore(a.Name))(restored)
	}
	return "", fmt.Errorf("unknown action %q: read, write, set, list, show, hide, link, mv, rm, archive or restore", a.Action)
}

// pageTool runs the page tool, as stickypane page add and page list.
func pageTool(b *api.API, a args) (string, error) {
	switch a.Action {
	case "add":
		if !strings.Contains(strings.Trim(a.Name, "/"), "/") {
			return "", fmt.Errorf("%s is not a book: a book is a folder in a tab, such as deploy/docs", a.Name)
		}
		var body []byte
		if a.Text != "" {
			body = []byte(a.Text + "\n")
		}
		file, err := b.CreateFrom(a.Name, a.Text, "", api.Options{}, body)
		if err != nil {
			return "", err
		}
		return "added " + file, nil
	case "list":
		pages, err := b.Pages(a.Name)
		if err != nil {
			return "", err
		}
		return asJSON(pages)
	}
	return "", fmt.Errorf("unknown action %q: add or list", a.Action)
}

// worktreeTool runs the worktree tool, as stickypane worktree list and
// worktree show: the board is brought in step with git first.
func worktreeTool(b *api.API, a args) (string, error) {
	if a.Action != "list" && a.Action != "show" {
		return "", fmt.Errorf("unknown action %q: list or show", a.Action)
	}
	list, _ := worktree.Board{Store: b.Store(), Records: registry.Default(), Now: b.Now}.Sync() // best effort, as the command line
	if a.Action == "list" {
		return asJSON(list)
	}
	i := slices.IndexFunc(list, func(w worktree.Worktree) bool { return w.Name == a.Name })
	if i < 0 {
		return "", i18n.Err("worktree.unknown", map[string]any{"Name": a.Name})
	}
	return asJSON(list[i])
}

// diffTool runs the diff tool, as stickypane diff list and diff show.
func diffTool(a args) (string, error) {
	if a.Action != "list" && a.Action != "show" {
		return "", fmt.Errorf("unknown action %q: list or show", a.Action)
	}
	if a.Action == "show" && (a.Name == "" || a.File == "") {
		return "", errors.New(`show needs "name", the worktree, and "file"`)
	}
	book, err := diffbook.Read(".", a.Name, "")
	if err != nil {
		return "", err
	}
	if a.Name != "" && book.Overview.Chosen != a.Name {
		return "", i18n.Err("diff.unknown", map[string]any{"Name": a.Name})
	}
	switch {
	case a.Action == "show":
		i := slices.IndexFunc(book.Changes, func(c git.Change) bool { return c.Path == a.File })
		if i < 0 {
			return "", i18n.Err("diff.nofile", map[string]any{"Name": a.Name, "File": a.File})
		}
		text, _, err := book.Diff(book.Changes[i])
		return text, err
	case a.Name != "":
		return asJSON(book.Changes)
	}
	return asJSON(book.Overview)
}

// asJSON is v as the command line's --json prints it.
func asJSON(v any) (string, error) {
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	err := enc.Encode(v)
	return b.String(), err
}

// onBoard says whether a tool works on the board, and whether this call
// changes it, so that a missing board may be made for it. The guide and
// diff need no board.
func onBoard(tool, action string) (on, write bool) {
	switch tool {
	case "guide", "diff":
		return false, false
	case "todo", "card", "chart", "log", "chat":
		return true, true
	case "note":
		// as stickypane note show, putting a note up makes the board
		return true, action == "write" || action == "set" || action == "show" || action == "link"
	}
	return true, false
}

// board is the API a tool works on. Until the server has a board, Open is
// asked for one at each tool that needs it, and the first found is kept.
func (s *Server) board(onBoard, write bool) (*api.API, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !onBoard || s.Open == nil || s.API.Store() != nil {
		return s.API, nil
	}
	a, err := s.Open(write)
	if err != nil {
		return nil, err
	}
	s.API = a
	return a, nil
}

// errFound ends a wait_event that has its event.
var errFound = errors.New("found")

// split reads a comma-separated argument.
func split(s string) []string {
	var out []string
	for _, v := range strings.Split(s, ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}
