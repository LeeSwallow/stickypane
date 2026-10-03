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
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LeeSwallow/stickypane/internal/api"
)

// defaultProtocol is answered when the client does not name a version.
const defaultProtocol = "2025-06-18"

const (
	maxWait   = 600 // seconds read_answers waits at most
	waitEvery = 200 * time.Millisecond
)

// JSON-RPC error codes.
const (
	codeParse    = -32700
	codeNoMethod = -32601
	codeParams   = -32602
)

// Server answers MCP requests for one notes folder.
type Server struct {
	API     *api.API
	Guide   string // the short agent guide, which the "guide" tool opens with
	Version string
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
			"topic": str(`A skill, such as "tracking-progress", or a shape, such as "board". Omit for the overview.`),
		}),
	},
	{
		Name:        "index",
		Description: "Every note of the board in a line, by tab: its title, what it counts (2/5, 7 cards) and a gist (its summary, the next open item, the latest log line). Read this to know what is on the board, instead of every note.",
		InputSchema: object(nil, map[string]any{}),
	},
	{
		Name:        "read_note",
		Description: "Read one note's Markdown file, front matter included. Read a note before rewriting it: the user may have changed it from the board.",
		InputSchema: object([]string{"name"}, map[string]any{"name": str(`The note's file name, such as "plan" or "plan.md".`)}),
	},
	{
		Name:        "write_note",
		Description: "Create a note or replace its content. The content is Markdown and may start with front matter. How the user arranged an existing note (open, size, color, pin) is kept unless you set it. Call the guide tool for the formats of boards, checklists, logs, charts and forms.",
		InputSchema: object([]string{"name", "content"}, map[string]any{
			"name":    str(`The note's file name, such as "plan" or "plan.md". Prefix a number ("10-plan") to control the order.`),
			"content": str("The whole note as Markdown."),
			"type":    map[string]any{"type": "string", "enum": []string{"note", "board", "checklist", "log", "chart", "form", "chat"}, "description": "The note's shape. Omit for a plain note."},
			"title":   str("Shown in the title bar and the note's border."),
			"open":    map[string]any{"type": "boolean", "description": "true puts the note on the screen now."},
			"size":    map[string]any{"type": "string", "enum": []string{"page", "half", "card"}, "description": "page is the whole width, half is half of it, card is a small sticky note."},
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
			"name": str(`The log's file name, such as "worklog".`),
			"line": str("The entry, one line."),
		}),
	},
	{
		Name:        "say",
		Description: "Add a message to a chat note, the conversation with the user on the board. A chat that does not exist yet is made. The user answers there; wait_event with type message.added hears it.",
		InputSchema: object([]string{"name", "text"}, map[string]any{
			"name": str(`The chat's file name, such as "chat".`),
			"text": str("The message, one line."),
			"as":   str(`Who says it, such as "claude". Default "agent".`),
		}),
	},
	{
		Name:        "set_keys",
		Description: "Change front matter keys of an existing note and nothing else: title, open, size, rows, color, pin, view. An empty value removes the key.",
		InputSchema: object([]string{"name", "keys"}, map[string]any{
			"name": str("The note's file name."),
			"keys": map[string]any{"type": "object", "description": `Keys and their new values, such as {"open": true, "size": "half"}.`},
		}),
	},
	{
		Name:        "arrange_note",
		Description: "Where a note is: show puts it in front of the user (a path to any file or folder of the project is linked onto the board), hide folds it away, move renames it or moves it into a tab or book, remove moves it to the trash, restore brings back what was removed last under that name.",
		InputSchema: object([]string{"name", "action"}, map[string]any{
			"name":   str(`A note's file name, such as "plan", "build.log" or "docs/plan"; for show, also a path such as "README.md" or "docs/".`),
			"action": map[string]any{"type": "string", "enum": []string{"show", "hide", "move", "remove", "restore"}},
			"to":     str(`For move: a new name ("roadmap"), a folder ending in "/" ("docs/"), or "." for the top level.`),
		}),
	},
	{
		Name:        "read_answers",
		Description: "Read what the user chose and wrote in a form note (type: form) and which button they pressed. With wait_seconds it waits up to that long for a button to be pressed, then returns the answers so far.",
		InputSchema: object([]string{"name"}, map[string]any{
			"name":         str(`The form's file name, such as "deploy" or "deploy.md".`),
			"wait_seconds": map[string]any{"type": "number", "description": "How long to wait for a button, at most 600. Omit to read the answers now."},
		}),
	},
	{
		Name:        "wait_event",
		Description: "Wait for something to happen on the board and return it as JSON: a note made or removed, an item ticked, a card moved, a form sent, a line logged, a chart value changed. Use it to react to what the user does without being told.",
		InputSchema: object(nil, map[string]any{
			"notes":        str(`Only these notes or folders, comma-separated, such as "plan,deploy". Omit for all.`),
			"types":        str(`Only these events or their kinds, comma-separated, such as "item.ticked" or "card". Omit for all.`),
			"wait_seconds": map[string]any{"type": "number", "description": "How long to wait, at most 600. Default 60."},
		}),
	},
	{
		Name:        "api",
		Description: "Experimental. A .http note holds REST requests (resterm and VS Code REST Client syntax). With only name, list its requests; with request (its title, a part of it, or #2) send it, or with all send every one in order; returns each response's status, headers, body, checks and captured values, as the board shows them.",
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
			case req.Method == "notifications/cancelled":
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
	if req.Method != "tools/call" {
		return false
	}
	var p struct {
		Name string `json:"name"`
	}
	_ = json.Unmarshal(req.Params, &p)
	return p.Name == "read_answers" || p.Name == "wait_event"
}

// handle answers one message. Notifications, which carry no id, get no answer.
func (s *Server) handle(ctx context.Context, line []byte) *response {
	var req request
	if err := json.Unmarshal(line, &req); err != nil {
		return &response{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{codeParse, "the message is not valid JSON"}}
	}
	if req.ID == nil {
		return nil
	}
	resp := &response{JSONRPC: "2.0", ID: req.ID}
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
	case "tools/call":
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

// call runs a tool. A tool that fails returns a result marked isError, which
// the agent can read and act on; only an unknown tool or unreadable
// arguments is a protocol error.
func (s *Server) call(ctx context.Context, params json.RawMessage) (*toolResult, *rpcError) {
	var p struct {
		Name      string `json:"name"`
		Arguments struct {
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
		} `json:"arguments"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &rpcError{codeParams, "cannot read the tool arguments: " + err.Error()}
	}
	text, err := "", error(nil)
	switch a := p.Arguments; p.Name {
	case "read_note":
		b, rerr := s.API.Cat(a.Name)
		text, err = string(b), rerr
	case "write_note":
		if a.Content == nil {
			// Without this check a call that forgot its content would
			// replace the note with nothing and report success.
			err = errors.New("write_note needs content: the whole note as Markdown")
			break
		}
		file, werr := s.API.Write(a.Name, api.Options{Type: a.Type, Title: a.Title, Size: a.Size, Open: a.Open}, []byte(*a.Content))
		text, err = "wrote "+file, werr
	case "todo":
		text, err = s.API.Todo(a.Name, a.Action, a.Item)
	case "card":
		text, err = s.API.Card(a.Name, a.Action, a.Card, a.To)
	case "chart":
		text, err = s.API.Chart(a.Name, a.Action, a.Label, plain(a.Value))
	case "log":
		text, err = s.API.Log(a.Name, a.Line)
	case "say":
		text, err = s.API.Say(a.Name, a.As, a.Text)
	case "arrange_note":
		text, err = s.arrange(a.Name, a.Action, a.To)
	case "set_keys":
		keys := make([]string, 0, len(a.Keys))
		for k := range a.Keys {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		pairs := make([]string, len(keys))
		for i, k := range keys {
			pairs[i] = k + "=" + plain(a.Keys[k])
		}
		text, err = s.API.Set(a.Name, pairs)
	case "read_answers":
		wctx, cancel := context.WithTimeout(ctx, time.Duration(min(max(a.Wait, 0), maxWait)*float64(time.Second)))
		got, aerr := s.API.Wait(wctx, a.Name, waitEvery)
		cancel()
		if errors.Is(aerr, context.DeadlineExceeded) || errors.Is(aerr, context.Canceled) {
			aerr = nil // the answers so far are the result
		}
		text, err = got.String(), aerr
	case "api":
		text, _, err = s.API.HTTP(ctx, a.Name, a.Request, a.Env, a.All)
	case "index":
		var idx []api.Entry
		if idx, err = s.API.Index(); err == nil {
			text = api.IndexMarkdown(idx, "board")
		}
	case "wait_event":
		wait := a.Wait
		if wait <= 0 {
			wait = 60
		}
		wctx, cancel := context.WithTimeout(ctx, time.Duration(min(wait, maxWait)*float64(time.Second)))
		f := api.Filter{Notes: split(a.Notes), Types: split(a.Types)}
		var got *api.Event
		werr := s.API.Watch(wctx, f, func(e api.Event) error {
			got = &e
			return errFound
		}, nil)
		cancel()
		switch {
		case got != nil:
			b, _ := json.Marshal(got)
			text = string(b)
		case errors.Is(werr, context.DeadlineExceeded) || errors.Is(werr, context.Canceled):
			text = "nothing happened"
		default:
			err = werr
		}
	case "guide":
		text, err = s.API.Guide(a.Topic, s.Guide)
	default:
		return nil, &rpcError{codeParams, "unknown tool " + p.Name}
	}
	if err != nil {
		// What was done before it failed is kept: the logs of the requests
		// that were sent are what the agent needs to see why.
		msg := err.Error()
		if strings.TrimSpace(text) != "" {
			msg = strings.TrimRight(text, "\n") + "\n\n" + msg
		}
		return &toolResult{Content: []content{{Type: "text", Text: msg}}, IsError: true}, nil
	}
	return &toolResult{Content: []content{{Type: "text", Text: text}}}, nil
}

// arrange runs the arrange_note tool.
func (s *Server) arrange(name, action, to string) (string, error) {
	switch action {
	case "show":
		return s.API.Show(name)
	case "hide":
		return s.API.Hide(name)
	case "move":
		if to == "" {
			return "", errors.New(`move needs "to": a new name, a folder ending in "/", or "."`)
		}
		return s.API.Move(name, to)
	case "remove":
		return s.API.Remove(name)
	case "restore":
		return s.API.Restore(name)
	}
	return "", fmt.Errorf("unknown action %q: show, hide, move, remove or restore", action)
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
