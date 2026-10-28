package bridge

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

const (
	maxRestarts   = 3
	helloTimeout  = 10 * time.Second
	closeTimeout  = 3 * time.Second
	notifications = 256
)

// Client is a running `stickypane bridge`. It is safe for concurrent use.
// When the bridge dies it is started again (at most three times over the
// client's life); a call that was waiting for an answer then fails with
// ErrConnectionLost, a later call goes to the new process, and a
// MethodRestarted notification says the board should be loaded again.
type Client struct {
	bin, project string
	hello        Hello
	notes        chan Notification
	quit         chan struct{}
	wg           sync.WaitGroup // readers and the supervisor: the senders on notes

	mu       sync.Mutex
	cur      *proc         // nil while a new bridge is being started
	ready    chan struct{} // closed when cur, down or closed changes from what a waiter saw
	restarts int
	down     bool
	closed   bool
}

// proc is one bridge process.
type proc struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
	wmu   sync.Mutex // one line at a time on stdin
	dead  chan struct{}

	pmu     sync.Mutex
	next    int64
	pending map[int64]chan Response
	gone    bool
}

// Start runs `stickypane bridge` in the project folder and checks its
// hello. The context ends the client as Close does.
func Start(ctx context.Context, stickypane, project string) (*Client, error) {
	c := &Client{
		bin:     stickypane,
		project: project,
		notes:   make(chan Notification, notifications),
		quit:    make(chan struct{}),
		ready:   make(chan struct{}),
	}
	p, h, err := c.launch(ctx)
	if err != nil {
		return nil, err
	}
	c.hello = h
	c.cur = p
	c.wg.Add(1)
	go c.supervise(p)
	go func() {
		select {
		case <-ctx.Done():
			c.Close()
		case <-c.quit:
		}
	}()
	return c, nil
}

// Hello is what the first bridge said of itself.
func (c *Client) Hello() Hello { return c.hello }

// Notifications is where the bridge's own lines (board.changed) arrive,
// and the client's bridge.restarted. It is closed by Close. It should be
// read: a full channel holds up the bridge's answers.
func (c *Client) Notifications() <-chan Notification { return c.notes }

// Call sends one request and waits for its answer. params and result may
// be nil; an error answer comes back as *Error. Any method name goes, so a
// method the bridge learns needs no change here.
func (c *Client) Call(ctx context.Context, method string, params, result any) error {
	var raw json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return err
		}
		raw = b
	}
	for {
		c.mu.Lock()
		switch {
		case c.closed:
			c.mu.Unlock()
			return ErrClosed
		case c.down:
			c.mu.Unlock()
			return ErrBridgeDown
		}
		p, wait := c.cur, c.ready
		c.mu.Unlock()
		if p != nil {
			return p.call(ctx, method, raw, result)
		}
		select { // a new bridge is on its way
		case <-wait:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Close ends the bridge by closing its input, as the app's quitting does,
// and kills it if it does not end. It is safe to call more than once.
func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	close(c.quit)
	p := c.cur
	c.settle()
	c.mu.Unlock()
	if p != nil {
		p.stdin.Close()
		select {
		case <-p.dead:
		case <-time.After(closeTimeout):
			p.cmd.Process.Kill()
			<-p.dead
		}
	}
	c.wg.Wait()
	close(c.notes)
	return nil
}

// settle wakes whoever waits on ready and arms it again if a bridge is
// still to come. The caller holds mu.
func (c *Client) settle() {
	close(c.ready)
	c.ready = make(chan struct{})
}

// launch starts one bridge and asks it hello.
func (c *Client) launch(ctx context.Context) (*proc, Hello, error) {
	cmd := exec.Command(c.bin, "bridge")
	cmd.Dir = c.project
	cmd.Env = os.Environ()
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, Hello{}, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, Hello{}, err
	}
	if err := cmd.Start(); err != nil {
		return nil, Hello{}, err
	}
	p := &proc{cmd: cmd, stdin: in, dead: make(chan struct{}), pending: map[int64]chan Response{}}
	c.wg.Add(1)
	go c.read(p, out)

	hctx, cancel := context.WithTimeout(ctx, helloTimeout)
	defer cancel()
	var h Hello
	if err := p.call(hctx, "hello", nil, &h); err != nil {
		p.kill()
		return nil, Hello{}, fmt.Errorf("bridge: hello: %w", err)
	}
	if h.Protocol != Protocol {
		p.kill()
		return nil, Hello{}, fmt.Errorf("%w: the bridge speaks %d, the app %d", ErrProtocol, h.Protocol, Protocol)
	}
	return p, h, nil
}

// read dispatches what one process says until it ends.
func (c *Client) read(p *proc, out io.Reader) {
	defer c.wg.Done()
	r := bufio.NewReader(out)
	for {
		line, err := r.ReadBytes('\n')
		if len(line) > 0 {
			c.dispatch(p, line)
		}
		if err != nil {
			break
		}
	}
	p.cmd.Wait()
	c.mu.Lock()
	if c.cur == p { // from here a call waits for the new bridge
		c.cur = nil
	}
	c.mu.Unlock()
	p.fail()
	close(p.dead)
}

func (c *Client) dispatch(p *proc, line []byte) {
	var m struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
		Result json.RawMessage `json:"result"`
		Error  *Error          `json:"error"`
	}
	if json.Unmarshal(line, &m) != nil {
		return
	}
	if m.Method != "" && len(m.ID) == 0 {
		c.notify(Notification{Method: m.Method, Params: m.Params})
		return
	}
	var id int64
	if json.Unmarshal(m.ID, &id) != nil {
		return // an answer to no request of ours (id null)
	}
	p.pmu.Lock()
	ch := p.pending[id]
	delete(p.pending, id)
	p.pmu.Unlock()
	if ch != nil {
		ch <- Response{ID: m.ID, Result: m.Result, Error: m.Error}
	}
}

func (c *Client) notify(n Notification) {
	select {
	case c.notes <- n:
	case <-c.quit:
	}
}

// supervise starts the bridge again each time it dies, up to three times.
func (c *Client) supervise(p *proc) {
	defer c.wg.Done()
	for {
		select {
		case <-p.dead:
		case <-c.quit:
			return
		}
		for {
			c.mu.Lock()
			if c.closed {
				c.mu.Unlock()
				return
			}
			c.restarts++
			if c.restarts > maxRestarts {
				c.down = true
				c.settle()
				c.mu.Unlock()
				return
			}
			c.mu.Unlock()
			np, h, err := c.launch(context.Background())
			c.mu.Lock()
			if c.closed {
				c.mu.Unlock()
				if np != nil {
					np.kill()
				}
				return
			}
			if err != nil || h.Protocol != Protocol {
				c.mu.Unlock()
				continue // counts as one more death
			}
			c.cur = np
			c.settle()
			c.mu.Unlock()
			c.notify(Notification{Method: MethodRestarted})
			p = np
			break
		}
	}
}

// call writes one request to the process and waits for its answer.
func (p *proc) call(ctx context.Context, method string, params json.RawMessage, result any) error {
	ch := make(chan Response, 1)
	p.pmu.Lock()
	if p.gone {
		p.pmu.Unlock()
		return ErrConnectionLost
	}
	p.next++
	id := p.next
	p.pending[id] = ch
	p.pmu.Unlock()
	forget := func() {
		p.pmu.Lock()
		delete(p.pending, id)
		p.pmu.Unlock()
	}

	b, err := json.Marshal(Request{ID: id, Method: method, Params: params})
	if err != nil {
		forget()
		return err
	}
	p.wmu.Lock()
	_, err = p.stdin.Write(append(b, '\n'))
	p.wmu.Unlock()
	if err != nil {
		forget()
		return ErrConnectionLost
	}
	select {
	case resp, ok := <-ch:
		if !ok {
			return ErrConnectionLost
		}
		if resp.Error != nil {
			return resp.Error
		}
		if result != nil && len(resp.Result) > 0 {
			return json.Unmarshal(resp.Result, result)
		}
		return nil
	case <-ctx.Done():
		forget()
		return ctx.Err()
	}
}

// fail ends every call that waits on a process that is gone.
func (p *proc) fail() {
	p.pmu.Lock()
	p.gone = true
	for id, ch := range p.pending {
		close(ch)
		delete(p.pending, id)
	}
	p.pmu.Unlock()
}

func (p *proc) kill() {
	p.stdin.Close()
	p.cmd.Process.Kill()
}

// searchDirs are where Find looks after PATH: a GUI app does not get the
// shell's PATH, so the usual homes of a command line program are tried.
var searchDirs = func() []string {
	dirs := []string{"/opt/homebrew/bin", "/usr/local/bin"}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, "go", "bin"))
	}
	return dirs
}

// Find returns the path of the stickypane program: STICKYPANE_BIN if set,
// else PATH, /opt/homebrew/bin, /usr/local/bin, ~/go/bin.
func Find() (string, error) {
	if p := os.Getenv("STICKYPANE_BIN"); p != "" {
		if isExecutable(p) {
			return p, nil
		}
		return "", fmt.Errorf("%w: STICKYPANE_BIN=%s", ErrNotFound, p)
	}
	if p, err := exec.LookPath("stickypane"); err == nil {
		return p, nil
	}
	name := "stickypane"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	for _, d := range searchDirs() {
		if p := filepath.Join(d, name); isExecutable(p) {
			return p, nil
		}
	}
	return "", ErrNotFound
}

func isExecutable(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Mode().IsRegular() && (runtime.GOOS == "windows" || fi.Mode().Perm()&0o111 != 0)
}
