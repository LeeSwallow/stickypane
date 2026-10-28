// Package bridge opens the board to a screen other than the terminal: the
// desktop app runs `stickypane bridge` as its child and talks to it over
// standard input and output, one JSON object a line. The app asks
// (board.load, note.act, note.create, note.run, note.open,
// settings.get, settings.set, and the ones that arrange and manage notes
// as the terminal does: note.arrange, tab.order, note.move, note.archive,
// note.trash, note.restore, note.send, kinds.list, board.index) and the
// bridge answers each request by its id; when the files change, whoever
// changed them, it says so with board.changed. When the session ends,
// because the app closed the input or the context ended, a script it
// started is stopped: the bridge never outlives the app for it. There is no network
// listener: the files stay the truth,
// and the bridge is one more reader and writer of them, through the same
// api and store as the command line.
package bridge

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/LeeSwallow/stickypane/internal/api"
	"github.com/LeeSwallow/stickypane/internal/env"
	"github.com/LeeSwallow/stickypane/internal/i18n"
	"github.com/LeeSwallow/stickypane/internal/runner"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/theme"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// server is one bridge session.
type server struct {
	api *api.API
	reg widget.Registry

	ctx  context.Context // the session: a request that outlives a line ends with it
	mu   sync.Mutex      // one line at a time on out
	out  io.Writer
	fail func() // ends the session when out cannot be written
	runs sync.WaitGroup

	live   sync.Mutex // guards ending and scripts
	ending bool       // the session is over: a script that starts now is stopped
	script map[*runner.Run]struct{}
}

// Serve answers the requests read from in on out, and tells out of every
// change to the board, until in ends or ctx does. A line it cannot make
// sense of is answered with an error, never the end of the session. It
// returns once every request is answered; a script it started that still
// runs when the session ends is stopped, and the input is closed when it
// can be (a reader nobody reads from would wait for ever). Closing the
// input is how the goroutine that reads it ends with the session.
func Serve(ctx context.Context, a *api.API, reg widget.Registry, in io.Reader, out io.Writer) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	s := &server{ctx: ctx, api: a, reg: reg, out: out, fail: cancel, script: map[*runner.Run]struct{}{}}
	// The watcher first: a change made while the board is read is then seen
	// as a change, not lost between the reading and the watching.
	changed, err := a.Store().Watch(ctx)
	if err != nil {
		return err
	}
	trace("watching")
	before, err := a.Store().Load()
	if err != nil {
		return err
	}
	trace("read")
	var following sync.WaitGroup
	following.Add(1)
	go func() {
		defer following.Done()
		s.follow(ctx, changed, before)
	}()
	defer func() {
		cancel()
		s.stopScripts()
		s.runs.Wait()
		following.Wait()
		if c, ok := in.(io.Closer); ok {
			c.Close() // unblocks the reader
		}
	}()

	lines := make(chan []byte)
	go func() {
		defer close(lines)
		r := bufio.NewReader(in)
		for {
			line, err := r.ReadBytes('\n')
			if len(bytes.TrimSpace(line)) > 0 {
				select {
				case lines <- line:
				case <-ctx.Done():
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return nil
		case line, ok := <-lines:
			if !ok {
				return nil
			}
			s.handle(line)
		}
	}
}

// send writes one message as one line.
func (s *server) send(v any) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		b.Reset()
		enc.Encode(response{ID: json.RawMessage("null"), Error: &Error{CodeFailed, err.Error()}})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.out.Write(b.Bytes()); err != nil {
		s.fail()
	}
}

// reply answers a request with what its method returned.
func (s *server) reply(id json.RawMessage, result any, err error) {
	if len(id) == 0 {
		id = json.RawMessage("null")
	}
	if err != nil {
		s.send(response{ID: id, Error: answer(err)})
		return
	}
	s.send(response{ID: id, Result: result})
}

// methods are what a screen may ask, by name.
var methods = map[string]func(s *server, params json.RawMessage) (any, error){
	"hello":        (*server).hello,
	"board.load":   (*server).boardLoad,
	"note.act":     (*server).noteAct,
	"note.create":  (*server).noteCreate,
	"note.arrange": (*server).noteArrange,
	"tab.order":    (*server).tabOrder,
	"note.move":    (*server).noteMove,
	"note.archive": (*server).noteArchive,
	"note.trash":   (*server).noteTrash,
	"note.restore": (*server).noteRestore,
	"kinds.list":   (*server).kindsList,
	"board.index":  (*server).boardIndex,
	"note.open":    (*server).noteOpen,
	"settings.get": (*server).settingsGet,
	"settings.set": (*server).settingsSet,
}

// long are the methods that are answered when they are done, while the
// session goes on.
var long = map[string]func(s *server, params json.RawMessage) (any, error){
	"note.run":  (*server).noteRun,
	"note.send": (*server).noteSend,
}

// handle answers one line.
func (s *server) handle(line []byte) {
	var req request
	if err := json.Unmarshal(line, &req); err != nil {
		s.reply(nil, nil, badRequest("a request is one JSON object a line: "+err.Error()))
		return
	}
	if long, ok := long[req.Method]; ok {
		// A script or a request may take long; the session goes on meanwhile.
		s.runs.Add(1)
		go func() {
			defer s.runs.Done()
			res, err := long(s, req.Params)
			s.reply(req.ID, res, err)
		}()
		return
	}
	m, ok := methods[req.Method]
	if !ok {
		s.reply(req.ID, nil, badRequest(fmt.Sprintf("there is no method %q", req.Method)))
		return
	}
	res, err := m(s, req.Params)
	s.reply(req.ID, res, err)
}

// decode reads a method's parameters; none at all reads as {}.
func decode(params json.RawMessage, v any) error {
	if len(params) == 0 || string(params) == "null" {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(params))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return badRequest("params: " + err.Error())
	}
	return nil
}

// project is the folder the notes folder is in.
func (s *server) project() string { return filepath.Dir(s.api.Store().Dir) }

func (s *server) hello(json.RawMessage) (any, error) {
	return map[string]any{"protocol": Protocol, "version": Version, "project": s.project()}, nil
}

func (s *server) boardLoad(json.RawMessage) (any, error) {
	b, err := s.api.Store().Load()
	if err != nil {
		return nil, err
	}
	return s.load(b), nil
}

// checkName refuses what is not the name of a note in the notes folder.
func checkName(name string) error {
	if strings.TrimSpace(name) == "" {
		return badRequest("which note? params.note is empty")
	}
	for _, part := range strings.Split(name, "/") {
		if part == "" || strings.HasPrefix(part, ".") || strings.ContainsAny(part, "\\\x00") {
			return badRequest(fmt.Sprintf("%q is not a note's name: use one board.load gave", name))
		}
	}
	return nil
}

// find looks a note up by the name board.load gave it: a note, a book or
// a page. owner is the note itself, or the book a page is in.
func (s *server) find(name string) (n, owner store.Note, err error) {
	if strings.TrimSpace(name) == "" {
		return n, owner, badRequest("which note? params.note is empty")
	}
	if err := checkName(name); err != nil {
		return n, owner, err
	}
	b, err := s.api.Store().Load()
	if err != nil {
		return n, owner, err
	}
	for _, t := range b.Tabs {
		for _, note := range t.Notes {
			if note.Name == name {
				return note, note, nil
			}
			for _, p := range note.Pages {
				if p.Name == name {
					return p, note, nil
				}
			}
		}
	}
	return n, owner, notFound(fmt.Sprintf("there is no note %q on the board", name))
}

func (s *server) noteAct(params json.RawMessage) (any, error) {
	var p actParams
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	if p.Action == nil || p.Action.Name == "" {
		return nil, badRequest("params.action needs a name")
	}
	n, _, err := s.find(p.Note)
	if err != nil {
		return nil, err
	}
	if n.Book() {
		return nil, badRequest(fmt.Sprintf("%s is a book: act on one of its pages", p.Note))
	}
	if n.Err != nil {
		return nil, &Error{CodeFailed, n.Err.Error()}
	}
	if widget.Version(n.Doc) != p.Version {
		return nil, api.ErrStale
	}
	// The action is tried on the note as read, so that a bad one is the
	// screen's mistake (bad_request) and not a failure to write.
	k := s.reg.For(n.Name, n.Doc)
	if k.Act == nil {
		return nil, badRequest(fmt.Sprintf("%s is a %s: it takes no action from a screen", n.Name, k.Name))
	}
	if k.Name == "chat" && p.Action.Name == "say" && strings.TrimSpace(p.Action.Args["who"]) == "" {
		// Said by the user, who is named as the terminal names them.
		args := map[string]string{"who": userName()}
		for k, v := range p.Action.Args {
			if k != "who" {
				args[k] = v
			}
		}
		act := *p.Action
		act.Args = args
		p.Action = &act
	}
	if _, err := k.Act(n.Doc, *p.Action); err != nil {
		return nil, badRequest(err.Error())
	}
	v, err := s.api.Apply(n.Name, p.Version, *p.Action)
	if err != nil {
		return nil, err
	}
	return map[string]string{"version": v}, nil
}

func (s *server) noteCreate(params json.RawMessage) (any, error) {
	var p createParams
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	tab := strings.Trim(p.Tab, "/")
	if tab != "" {
		tabs, err := s.api.Store().Tabs()
		if err != nil {
			return nil, err
		}
		found := false
		for _, t := range tabs {
			found = found || t.Name == tab
		}
		if !found {
			return nil, notFound(fmt.Sprintf("there is no tab %q", tab))
		}
	}
	kind := p.Kind
	if kind == "note" {
		kind = ""
	}
	if kind != "" && s.reg.Lookup(kind).Name != kind {
		return nil, badRequest(fmt.Sprintf("there is no kind %q a screen can make", p.Kind))
	}
	text := p.Title
	if strings.TrimSpace(text) == "" {
		text, _, _ = strings.Cut(strings.TrimSpace(p.Body), "\n")
	}
	stem := store.Slug(text, time.Now())
	if tab != "" {
		stem = tab + "/" + stem
	}
	// Each name is made exclusively: a file that appears between one try
	// and the next is not written over, the next name is tried.
	for i := 1; i <= maxSameName; i++ {
		name := stem + ".md"
		if i > 1 {
			name = fmt.Sprintf("%s-%d.md", stem, i)
		}
		file, err := s.api.Create(name, api.Options{Type: kind, Title: p.Title}, []byte(p.Body))
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if errors.Is(err, api.ErrUnknownType) {
			return nil, badRequest(err.Error())
		}
		if err != nil {
			return nil, err
		}
		return map[string]string{"note": file}, nil
	}
	return nil, &Error{CodeFailed, fmt.Sprintf("there are %d notes called %q already", maxSameName, stem)}
}

// maxSameName is how many notes of one title note.create makes before it
// gives up.
const maxSameName = 1000

// userName is who says what the user says in a chat: $STICKYPANE_USER,
// else git's user.name, else the login, as the terminal names them.
func userName() string { return env.Detect().User() }

// stopScripts ends the session's hold on its scripts: those running are
// killed, and one started after this is killed as it starts.
func (s *server) stopScripts() {
	s.live.Lock()
	defer s.live.Unlock()
	s.ending = true
	for r := range s.script {
		r.Kill()
	}
}

// running keeps a script to stop it when the session ends; it returns what
// to call when the script is over.
func (s *server) running(r *runner.Run) (done func()) {
	s.live.Lock()
	defer s.live.Unlock()
	if s.ending {
		r.Kill()
	}
	s.script[r] = struct{}{}
	return func() {
		s.live.Lock()
		delete(s.script, r)
		s.live.Unlock()
	}
}

// runResult is what note.run answers: the exit code, and how long the
// script ran in milliseconds.
type runResult struct {
	Code int   `json:"code"`
	Took int64 `json:"took"`
}

// noteRun runs a script note and waits for it. The app has asked the user
// already, as the terminal does before it runs one.
func (s *server) noteRun(params json.RawMessage) (any, error) {
	var p noteParams
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	n, _, err := s.find(p.Note)
	if err != nil {
		return nil, err
	}
	if k := s.reg.For(n.Name, n.Doc); n.Book() || k.Name != "script" {
		return nil, badRequest(fmt.Sprintf("%s is not a script", p.Note))
	}
	r, err := runner.Start(s.api.Store(), n.Name, time.Now)
	if err != nil {
		return nil, err
	}
	defer s.running(r)()
	code, took, err := r.Wait()
	if err != nil {
		return nil, err
	}
	return runResult{Code: code, Took: took.Milliseconds()}, nil
}

// start begins a program and does not wait for it. Tests replace it.
var start = func(cmd []string) error {
	c := exec.Command(cmd[0], cmd[1:]...)
	if err := c.Start(); err != nil {
		return err
	}
	go c.Wait() // reaped when it ends; nobody waits for the editor
	return nil
}

// noteOpen shows a note's file in an editor. The bridge has no terminal,
// so the editor is a GUI one or the system's own; what the user writes
// comes back through the watch.
func (s *server) noteOpen(params json.RawMessage) (any, error) {
	var p noteParams
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	n, _, err := s.find(p.Note)
	if err != nil {
		return nil, err
	}
	file := filepath.Join(s.api.Store().Dir, filepath.FromSlash(n.Name))
	cmd := env.Detect().OpenCommand(file)
	if cmd == nil {
		return nil, &Error{CodeFailed, "this system has no way to open a file without a terminal"}
	}
	if err := start(cmd); err != nil {
		return nil, err
	}
	return empty{}, nil
}

func (s *server) settingsGet(json.RawMessage) (any, error) { return s.settings(), nil }

// settingsSet changes the settings given. Every value is checked before
// any is written, so a bad one changes nothing.
func (s *server) settingsSet(params json.RawMessage) (any, error) {
	var p settingsParams
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	if p.Theme == nil && p.Style == nil && p.View == nil && p.Language == nil {
		return nil, badRequest("settings.set needs theme, style, view or language")
	}
	st := s.api.Store()
	var writes []func() error
	if p.Theme != nil {
		name := "auto"
		if !strings.EqualFold(strings.TrimSpace(*p.Theme), "auto") {
			t, ok := theme.Lookup(*p.Theme)
			if !ok {
				return nil, badRequest(fmt.Sprintf("there is no theme %q", *p.Theme))
			}
			name = t.Name
		}
		writes = append(writes, func() error { return st.SetTheme(name) })
	}
	if p.Style != nil {
		if *p.Style != store.StyleDoodle && *p.Style != store.StyleNeat {
			return nil, badRequest(fmt.Sprintf("style %q is not doodle or neat", *p.Style))
		}
		writes = append(writes, func() error { return st.SetStyle(*p.Style) })
	}
	if p.View != nil {
		switch *p.View {
		case store.ViewTerminal, store.ViewOverlay, store.ViewBoth:
		default:
			return nil, badRequest(fmt.Sprintf("view %q is not terminal, overlay or both", *p.View))
		}
		writes = append(writes, func() error { return st.SetViewMode(*p.View) })
	}
	if p.Language != nil {
		code := strings.ToLower(strings.TrimSpace(*p.Language))
		if code != "auto" {
			if _, err := i18n.Load(code); err != nil || code == "" {
				return nil, badRequest(fmt.Sprintf("there is no translation for %q: use auto, %s", *p.Language, strings.Join(i18n.Languages(), ", ")))
			}
		}
		// Stored as `stickypane language` stores it.
		writes = append(writes, func() error { return st.SetLanguage(code) })
	}
	for _, w := range writes {
		if err := w(); err != nil {
			return nil, err
		}
	}
	return empty{}, nil
}

// changes is what board.changed says: the notes that are new or differ,
// in board.load's shape, the names of those that went, and what happened
// in them in their kinds' words, as `stickypane watch --json` says it.
type changes struct {
	Notes   []Note      `json:"notes"`
	Removed []string    `json:"removed"`
	Events  []api.Event `json:"events"`
	// Settings, when the root settings changed (the tab an agent's show
	// chose, a theme or language chosen in the terminal), are all of them
	// as board.load says them; Palette comes with them when the theme
	// changed.
	Settings *Settings `json:"settings,omitempty"`
	Palette  *Palette  `json:"palette,omitempty"`
}

// follow tells the screen of every change to the board until ctx ends.
func (s *server) follow(ctx context.Context, changed <-chan struct{}, before store.Board) {
	seen := s.drawn(before)
	known := settingsOf(before.Settings)
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-changed:
			if !ok {
				return
			}
			after, err := s.api.Store().Load()
			if err != nil {
				continue // a board caught in the middle of a change; the next signal reads it whole
			}
			now := s.drawn(after)
			c := changes{Notes: []Note{}, Removed: []string{}, Events: s.api.Changes(before, after)}
			for _, name := range now.order {
				if string(seen.json[name]) != string(now.json[name]) {
					c.Notes = append(c.Notes, now.notes[name])
				}
			}
			for _, name := range seen.order {
				if _, ok := now.json[name]; !ok {
					c.Removed = append(c.Removed, name)
				}
			}
			if c.Events == nil {
				c.Events = []api.Event{}
			}
			if set := settingsOf(after.Settings); set != known {
				c.Settings = &set
				if set.Theme != known.Theme {
					p := palette(set.Theme)
					c.Palette = &p
				}
				known = set
			}
			before, seen = after, now
			if len(c.Notes) > 0 || len(c.Removed) > 0 || len(c.Events) > 0 || c.Settings != nil {
				s.send(notification{Method: "board.changed", Params: c})
			}
		}
	}
}

// drawing is every note of a board as board.load says it, and that as
// JSON, to tell which notes changed.
type drawing struct {
	order []string
	notes map[string]Note
	json  map[string][]byte
}

func (s *server) drawn(b store.Board) drawing {
	d := drawing{notes: map[string]Note{}, json: map[string][]byte{}}
	views := b.Settings.Views()
	for _, t := range b.Tabs {
		for _, n := range inOrder(t, b.Settings, views) {
			note := s.note(t.Name, n, views)
			raw, _ := json.Marshal(note)
			d.order = append(d.order, n.Name)
			d.notes[n.Name], d.json[n.Name] = note, raw
		}
	}
	return d
}

// trace tells a test the order of the session's first steps.
var trace = func(string) {}
