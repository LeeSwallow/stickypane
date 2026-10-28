package bridge

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	maxRestarts  = 3
	helloTimeout = 10 * time.Second
	closeTimeout = 3 * time.Second
	queueMax     = 256  // notifications held for a reader that is slow
	stderrTail   = 4096 // bytes of the bridge's stderr kept
	waitDelay    = 2 * time.Second
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
	qmu          sync.Mutex
	queue        []Notification // waiting for notes; the reader never waits for the consumer
	wake         chan struct{}
	quit         chan struct{}
	wg           sync.WaitGroup // readers and the supervisor: the senders on notes

	mu       sync.Mutex
	cur      *proc         // nil while a new bridge is being started
	ready    chan struct{} // closed when cur, down or closed changes from what a waiter saw
	restarts int
	down     bool
	downErr  error
	closed   bool
}

// proc is one bridge process.
type proc struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	wsem   chan struct{} // one line at a time on stdin; taken with the call's context
	dead   chan struct{}
	waited chan struct{} // the process has exited
	errs   *tail

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
		notes:   make(chan Notification, 16),
		wake:    make(chan struct{}, 1),
		quit:    make(chan struct{}),
		ready:   make(chan struct{}),
	}
	p, h, err := c.launch(ctx)
	if err != nil {
		return nil, err
	}
	c.hello = h
	c.cur = p
	c.wg.Add(2)
	go c.supervise(p)
	go c.pump()
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
// and the client's bridge.restarted. It is closed by Close. A consumer
// that is slow loses the oldest board.changed (the app loads the board
// again anyway); bridge.restarted is never lost, and the bridge never
// waits for the consumer.
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
			err := c.downErr
			c.mu.Unlock()
			return err
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
	// Our own pipe for stdout, so that no copying goroutine makes Wait wait
	// for a grandchild that holds it.
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, Hello{}, err
	}
	errs := &tail{}
	cmd.Stdout = pw
	cmd.Stderr = errs
	cmd.WaitDelay = waitDelay
	if err := cmd.Start(); err != nil {
		pr.Close()
		pw.Close()
		return nil, Hello{}, err
	}
	pw.Close()
	p := &proc{cmd: cmd, stdin: in, wsem: make(chan struct{}, 1), dead: make(chan struct{}), waited: make(chan struct{}), errs: errs, pending: map[int64]chan Response{}}
	readerDone := make(chan struct{})
	go func() { // reaps the process, and frees the reader if a grandchild keeps stdout open
		cmd.Wait()
		close(p.waited)
		select {
		case <-readerDone:
		case <-time.After(waitDelay):
			pr.Close()
		}
	}()
	c.wg.Add(1)
	go func() {
		defer close(readerDone)
		c.read(p, pr)
		pr.Close()
	}()

	hctx, cancel := context.WithTimeout(ctx, helloTimeout)
	defer cancel()
	var h Hello
	if err := p.call(hctx, "hello", nil, &h); err != nil {
		p.kill()
		return nil, Hello{}, fmt.Errorf("bridge: hello: %w%s", err, p.errs.suffix())
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
	p.kill() // a bridge that closed its output is no use; a dead one is already reaped
	<-p.waited
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

// notify queues a notification and never waits. A full queue drops its
// oldest board.changed.
func (c *Client) notify(n Notification) {
	c.qmu.Lock()
	if len(c.queue) >= queueMax {
		for i, o := range c.queue {
			if o.Method != MethodRestarted {
				c.queue = append(c.queue[:i], c.queue[i+1:]...)
				break
			}
		}
	}
	c.queue = append(c.queue, n)
	c.qmu.Unlock()
	select {
	case c.wake <- struct{}{}:
	default:
	}
}

// pump hands the queue to the consumer.
func (c *Client) pump() {
	defer c.wg.Done()
	for {
		c.qmu.Lock()
		if len(c.queue) == 0 {
			c.qmu.Unlock()
			select {
			case <-c.wake:
				continue
			case <-c.quit:
				return
			}
		}
		n := c.queue[0]
		c.queue = c.queue[1:]
		c.qmu.Unlock()
		select {
		case c.notes <- n:
		case <-c.quit:
			return
		}
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
				c.downErr = fmt.Errorf("%w%s", ErrBridgeDown, p.errs.suffix())
				c.settle()
				c.mu.Unlock()
				return
			}
			c.mu.Unlock()
			np, _, err := c.launch(context.Background())
			c.mu.Lock()
			if c.closed {
				c.mu.Unlock()
				if np != nil {
					np.kill()
				}
				return
			}
			if errors.Is(err, ErrProtocol) { // another version: starting again will not help
				c.down = true
				c.downErr = err
				c.settle()
				c.mu.Unlock()
				return
			}
			if err != nil {
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
	select { // the line goes out whole, one at a time, but a call does not wait for the pipe past its context
	case p.wsem <- struct{}{}:
	case <-ctx.Done():
		forget()
		return ctx.Err()
	}
	wrote := make(chan error, 1)
	go func() {
		_, err := p.stdin.Write(append(b, '\n'))
		<-p.wsem
		wrote <- err
	}()
	select {
	case err := <-wrote:
		if err != nil {
			forget()
			return ErrConnectionLost
		}
	case <-ctx.Done():
		forget()
		return ctx.Err()
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

// tail keeps the last bytes written to it: the end of the bridge's stderr.
type tail struct {
	mu sync.Mutex
	b  []byte
}

func (t *tail) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.b = append(t.b, p...)
	if len(t.b) > stderrTail {
		t.b = append([]byte(nil), t.b[len(t.b)-stderrTail:]...)
	}
	return len(p), nil
}

// suffix is ": " and the tail, or nothing when the bridge said nothing.
func (t *tail) suffix() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if s := strings.TrimSpace(string(t.b)); s != "" {
		return ": " + s
	}
	return ""
}
