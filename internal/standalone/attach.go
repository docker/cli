//go:build linux

package standalone

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/containerd/containerd/v2/core/runtime"
	"github.com/containerd/containerd/v2/pkg/cio"
	"github.com/containerd/containerd/v2/pkg/oci"
	cerrdefs "github.com/containerd/errdefs"
	"github.com/containerd/log"
	"github.com/containerd/typeurl/v2"
	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types"
	"github.com/moby/moby/client"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

// ---- container attach

// attachConn is the net.Conn handed to the CLI for "docker attach" and the
// foreground part of "docker run". Reads come from the logging helper's
// attach socket (dialed lazily, since the CLI attaches before it starts the
// container); writes go to the container's stdin FIFO.
type attachConn struct {
	ctx       context.Context
	sockPath  string
	stdinOpen bool
	isRunning func() bool

	// dialMu serializes connecting so that exactly one connection to the
	// logging helper is ever created, even when the output stream and the
	// pre-start handshake race.
	dialMu sync.Mutex

	mu         sync.Mutex
	sock       net.Conn
	closed     bool
	dialErr    error
	writeClose bool
}

// ensureDialed connects to the logging helper's attach socket, which only
// exists once the container's task has been created. It is called before the
// container is started so that the client misses no output.
func (a *attachConn) ensureDialed() error {
	_, err := a.dial()
	if errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

//nolint:gocyclo // retry loop with several terminal conditions
func (a *attachConn) dial() (net.Conn, error) {
	a.dialMu.Lock()
	defer a.dialMu.Unlock()

	a.mu.Lock()
	switch {
	case a.sock != nil:
		s := a.sock
		a.mu.Unlock()
		return s, nil
	case a.dialErr != nil:
		err := a.dialErr
		a.mu.Unlock()
		return nil, err
	case a.closed:
		a.mu.Unlock()
		return nil, net.ErrClosed
	}
	a.mu.Unlock()

	deadline := time.Now().Add(5 * time.Minute)
	for {
		conn, err := net.DialTimeout("unix", a.sockPath, time.Second)
		if err == nil {
			// Wait for the helper to confirm that it registered this client
			// before reporting the connection as established.
			if herr := readAttachHandshake(conn); herr != nil {
				_ = conn.Close()
				a.mu.Lock()
				a.dialErr = herr
				a.mu.Unlock()
				return nil, herr
			}
			a.mu.Lock()
			defer a.mu.Unlock()
			if a.closed {
				_ = conn.Close()
				return nil, net.ErrClosed
			}
			a.sock = conn
			if a.writeClose {
				if cw, ok := conn.(interface{ CloseWrite() error }); ok {
					_ = cw.CloseWrite()
				}
			}
			return conn, nil
		}
		if a.ctx.Err() != nil {
			return nil, a.ctx.Err()
		}
		a.mu.Lock()
		closed := a.closed
		a.mu.Unlock()
		if closed {
			return nil, net.ErrClosed
		}
		if time.Now().After(deadline) {
			a.mu.Lock()
			a.dialErr = fmt.Errorf("timed out waiting for the container to start: %w", err)
			err := a.dialErr
			a.mu.Unlock()
			return nil, err
		}
		if !errors.Is(err, unix.ENOENT) && !errors.Is(err, unix.ECONNREFUSED) {
			a.mu.Lock()
			a.dialErr = err
			a.mu.Unlock()
			return nil, err
		}
		if a.isRunning != nil && !a.isRunning() {
			if _, serr := os.Stat(a.sockPath); serr != nil {
				a.mu.Lock()
				a.dialErr = io.EOF
				a.mu.Unlock()
				return nil, io.EOF
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// readAttachHandshake consumes the single byte the logging helper sends once
// the client is registered.
func readAttachHandshake(conn net.Conn) error {
	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return err
	}
	var buf [1]byte
	if _, err := io.ReadFull(conn, buf[:]); err != nil {
		return fmt.Errorf("attaching to container: %w", err)
	}
	if buf[0] != attachHandshake {
		return errors.New("unexpected handshake from the container's logging helper")
	}
	return conn.SetReadDeadline(time.Time{})
}

func (a *attachConn) Read(p []byte) (int, error) {
	conn, err := a.dial()
	if err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
			return 0, io.EOF
		}
		return 0, err
	}
	return conn.Read(p)
}

// Write sends the client's input to the logging helper, which forwards it to
// the container's stdin.
func (a *attachConn) Write(p []byte) (int, error) {
	if !a.stdinOpen {
		return len(p), nil
	}
	conn, err := a.dial()
	if err != nil {
		return 0, err
	}
	return conn.Write(p)
}

// CloseWrite half-closes the connection, which makes the helper close the
// container's stdin.
func (a *attachConn) CloseWrite() error {
	a.mu.Lock()
	a.writeClose = true
	conn := a.sock
	a.mu.Unlock()
	if conn == nil {
		return nil
	}
	if cw, ok := conn.(interface{ CloseWrite() error }); ok {
		return cw.CloseWrite()
	}
	return nil
}

func (a *attachConn) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.closed = true
	if a.sock != nil {
		return a.sock.Close()
	}
	return nil
}

func (*attachConn) LocalAddr() net.Addr              { return unixAddr("standalone") }
func (*attachConn) RemoteAddr() net.Addr             { return unixAddr("container") }
func (*attachConn) SetDeadline(time.Time) error      { return nil }
func (*attachConn) SetReadDeadline(time.Time) error  { return nil }
func (*attachConn) SetWriteDeadline(time.Time) error { return nil }

type unixAddr string

func (unixAddr) Network() string  { return "unix" }
func (u unixAddr) String() string { return string(u) }

func (c *apiClient) ContainerAttach(ctx context.Context, ref string, options client.ContainerAttachOptions) (client.ContainerAttachResult, error) {
	ctr, m, err := c.eng.loadSynced(ctx, ref)
	if err != nil {
		return client.ContainerAttachResult{}, err
	}
	if m.State.Status == statusExited {
		return client.ContainerAttachResult{}, fmt.Errorf("you cannot attach to a stopped container, start it first: %w", cerrdefs.ErrConflict)
	}
	runDir := c.eng.cfg.containerRunDir(ctr.ID)
	conn := &attachConn{
		ctx:       ctx,
		sockPath:  filepath.Join(runDir, "attach.sock"),
		stdinOpen: options.Stdin && m.Config.OpenStdin,
		isRunning: func() bool {
			_, cur, err := c.eng.loadSynced(context.WithoutCancel(ctx), ctr.ID)
			return err == nil && cur.State.Status != statusExited
		},
	}
	mediaType := types.MediaTypeMultiplexedStream
	if m.Config.Tty {
		mediaType = types.MediaTypeRawStream
	}
	c.eng.registerAttach(ctr.ID, conn)
	return client.ContainerAttachResult{HijackedResponse: client.NewHijackedResponse(conn, mediaType)}, nil
}

// registerAttach records a pending attach connection so that the container
// start can make sure it is connected before the container runs.
func (e *engine) registerAttach(id string, conn *attachConn) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.attaches == nil {
		e.attaches = map[string]*attachConn{}
	}
	e.attaches[id] = conn
}

// dialPendingAttach connects a pending attach client for the container.
func (e *engine) dialPendingAttach(id string) error {
	e.mu.Lock()
	conn := e.attaches[id]
	e.mu.Unlock()
	if conn == nil {
		return nil
	}
	return conn.ensureDialed()
}

// ---- exec

type execProcess struct {
	id          string
	containerID string
	opts        client.ExecCreateOptions
	process     runtime.ExecProcess
	io          *cio.DirectIO
	mu          sync.Mutex
	started     bool
	exited      bool
	exitCode    int
	pid         int
	done        chan struct{}
}

func (c *apiClient) ExecCreate(ctx context.Context, ref string, options client.ExecCreateOptions) (client.ExecCreateResult, error) {
	ctr, m, err := c.eng.loadSynced(ctx, ref)
	if err != nil {
		return client.ExecCreateResult{}, err
	}
	if m.State.Status != statusRunning {
		if m.State.Status == statusPaused {
			return client.ExecCreateResult{}, fmt.Errorf("container %s is paused, unpause the container before exec: %w", shortID(ctr.ID), cerrdefs.ErrConflict)
		}
		return client.ExecCreateResult{}, fmt.Errorf("container %s is not running: %w", shortID(ctr.ID), cerrdefs.ErrConflict)
	}
	if len(options.Cmd) == 0 {
		return client.ExecCreateResult{}, fmt.Errorf("no command specified: %w", cerrdefs.ErrInvalidArgument)
	}
	ep := &execProcess{id: newID(), containerID: ctr.ID, opts: options, done: make(chan struct{})}
	c.eng.mu.Lock()
	c.eng.execs[ep.id] = ep
	c.eng.mu.Unlock()
	return client.ExecCreateResult{ID: ep.id}, nil
}

func (c *apiClient) getExec(id string) (*execProcess, error) {
	c.eng.mu.Lock()
	defer c.eng.mu.Unlock()
	ep, ok := c.eng.execs[id]
	if !ok {
		return nil, fmt.Errorf("no such exec instance %q: %w", id, cerrdefs.ErrNotFound)
	}
	return ep, nil
}

// execSpec builds the process spec for an exec from the container's spec.
func (e *engine) execSpec(ctx context.Context, ctrID string, opts client.ExecCreateOptions) (*specs.Process, error) {
	var proc *specs.Process
	err := e.withSession(ctx, func(ctx context.Context, s *session) error {
		ctr, _, err := loadContainer(ctx, s.containers(), ctrID)
		if err != nil {
			return err
		}
		spec, err := specFromAny(ctr.Spec)
		if err != nil {
			return err
		}
		if spec.Process == nil {
			return errors.New("container spec has no process")
		}
		p := *spec.Process
		p.Args = opts.Cmd
		p.Terminal = opts.TTY
		p.Env = append(append([]string{}, spec.Process.Env...), opts.Env...)
		if opts.WorkingDir != "" {
			p.Cwd = opts.WorkingDir
		}
		if opts.TTY && opts.ConsoleSize.Height > 0 {
			p.ConsoleSize = &specs.Box{Height: opts.ConsoleSize.Height, Width: opts.ConsoleSize.Width}
		}
		spec.Process = &p
		if opts.User != "" {
			if err := oci.ApplyOpts(ctx, s, &ctr, spec, oci.WithUser(opts.User), oci.WithAdditionalGIDs(opts.User)); err != nil {
				return fmt.Errorf("resolving user %q: %w", opts.User, err)
			}
		}
		if opts.Privileged {
			if err := oci.ApplyOpts(ctx, s, &ctr, spec, oci.WithAllKnownCapabilities); err != nil {
				return err
			}
		}
		proc = spec.Process
		return nil
	})
	return proc, err
}

// startExec creates and starts the exec process with the given I/O.
//
//nolint:gocyclo // process creation with several optional I/O paths
func (c *apiClient) startExec(ctx context.Context, ep *execProcess, withIO bool) error {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	if ep.started {
		return fmt.Errorf("exec %s has already been started: %w", shortID(ep.id), cerrdefs.ErrConflict)
	}
	proc, err := c.eng.execSpec(ctx, ep.containerID, ep.opts)
	if err != nil {
		return err
	}
	procAny, err := typeurl.MarshalAnyToProto(proc)
	if err != nil {
		return err
	}
	tm, err := c.eng.taskManagerFor(ctx, ep.containerID)
	if err != nil {
		return err
	}
	task, err := tm.get(ctx, ep.containerID)
	if err != nil {
		return fmt.Errorf("container %s is not running: %w", shortID(ep.containerID), cerrdefs.ErrConflict)
	}

	taskIO := runtime.IO{Terminal: ep.opts.TTY}
	if withIO {
		fifoDir := filepath.Join(c.eng.cfg.containerRunDir(ep.containerID), "exec")
		if err := os.MkdirAll(fifoDir, 0o700); err != nil {
			return err
		}
		fifos, err := cio.NewFIFOSetInDir(fifoDir, shortID(ep.id), ep.opts.TTY)
		if err != nil {
			return err
		}
		if !ep.opts.AttachStdin {
			fifos.Stdin = ""
		}
		dio, err := cio.NewDirectIO(context.WithoutCancel(ctx), fifos)
		if err != nil {
			return err
		}
		ep.io = dio
		cfg := dio.Config()
		taskIO.Stdin, taskIO.Stdout, taskIO.Stderr = cfg.Stdin, cfg.Stdout, cfg.Stderr
	}
	nctx := nsCtx(ctx)
	process, err := task.Exec(nctx, shortID(ep.id), runtime.ExecOpts{Spec: procAny, IO: taskIO})
	if err != nil {
		if ep.io != nil {
			_ = ep.io.Close()
		}
		return fmt.Errorf("creating exec: %w", err)
	}
	if err := process.Start(nctx); err != nil {
		_, _ = process.Delete(nsCtx(context.WithoutCancel(ctx)))
		if ep.io != nil {
			_ = ep.io.Close()
		}
		return fmt.Errorf("starting exec: %w", translateRuntimeError(err))
	}
	ep.process = process
	ep.started = true
	if st, err := process.State(nctx); err == nil {
		ep.pid = int(st.Pid)
	}
	go func() {
		bg := nsCtx(context.WithoutCancel(ctx))
		exit, err := process.Wait(bg)
		ep.mu.Lock()
		ep.exited = true
		if err == nil {
			ep.exitCode = int(exit.Status)
		} else {
			ep.exitCode = 255
		}
		ep.mu.Unlock()
		_, _ = process.Delete(bg)
		close(ep.done)
	}()
	return nil
}

func (c *apiClient) ExecStart(ctx context.Context, execID string, _ client.ExecStartOptions) (client.ExecStartResult, error) {
	ep, err := c.getExec(execID)
	if err != nil {
		return client.ExecStartResult{}, err
	}
	return client.ExecStartResult{}, c.startExec(ctx, ep, false)
}

func (c *apiClient) ExecAttach(ctx context.Context, execID string, options client.ExecAttachOptions) (client.ExecAttachResult, error) {
	ep, err := c.getExec(execID)
	if err != nil {
		return client.ExecAttachResult{}, err
	}
	if options.TTY {
		ep.opts.TTY = true
	}
	if options.ConsoleSize.Height > 0 {
		ep.opts.ConsoleSize = options.ConsoleSize
	}
	if err := c.startExec(ctx, ep, true); err != nil {
		return client.ExecAttachResult{}, err
	}
	conn := newExecConn(ep)
	mediaType := types.MediaTypeMultiplexedStream
	if ep.opts.TTY {
		mediaType = types.MediaTypeRawStream
	}
	return client.ExecAttachResult{HijackedResponse: client.NewHijackedResponse(conn, mediaType)}, nil
}

func (c *apiClient) ExecInspect(_ context.Context, execID string, _ client.ExecInspectOptions) (client.ExecInspectResult, error) {
	ep, err := c.getExec(execID)
	if err != nil {
		return client.ExecInspectResult{}, err
	}
	// The stream may end slightly before the exit is observed; give the
	// waiter a moment so that the exit code is accurate.
	select {
	case <-ep.done:
	case <-time.After(2 * time.Second):
	}
	ep.mu.Lock()
	defer ep.mu.Unlock()
	return client.ExecInspectResult{
		ID:          ep.id,
		ContainerID: ep.containerID,
		Running:     ep.started && !ep.exited,
		ExitCode:    ep.exitCode,
		PID:         ep.pid,
	}, nil
}

func (c *apiClient) ExecResize(ctx context.Context, execID string, options client.ExecResizeOptions) (client.ExecResizeResult, error) {
	ep, err := c.getExec(execID)
	if err != nil {
		return client.ExecResizeResult{}, err
	}
	ep.mu.Lock()
	p := ep.process
	ep.mu.Unlock()
	if p == nil {
		return client.ExecResizeResult{}, fmt.Errorf("exec %s is not running: %w", shortID(execID), cerrdefs.ErrConflict)
	}
	return client.ExecResizeResult{}, p.ResizePty(nsCtx(ctx), runtime.ConsoleSize{Width: uint32(options.Width), Height: uint32(options.Height)})
}

// execConn multiplexes the exec's stdout/stderr FIFOs into a single stream
// for the CLI and forwards writes to the exec's stdin.
type execConn struct {
	ep   *execProcess
	pr   *io.PipeReader
	pw   *io.PipeWriter
	once sync.Once
}

func newExecConn(ep *execProcess) *execConn {
	pr, pw := io.Pipe()
	c := &execConn{ep: ep, pr: pr, pw: pw}
	go c.pump()
	return c
}

func (c *execConn) pump() {
	var wg sync.WaitGroup
	copyStream := func(r io.Reader, st stdcopy.StdType) {
		defer wg.Done()
		if r == nil {
			return
		}
		buf := make([]byte, 32*1024)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				var frame []byte
				if c.ep.opts.TTY {
					frame = buf[:n]
				} else {
					frame = make([]byte, 8+n)
					frame[0] = byte(st)
					binary.BigEndian.PutUint32(frame[4:8], uint32(n))
					copy(frame[8:], buf[:n])
				}
				if _, werr := c.pw.Write(frame); werr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}
	wg.Add(2)
	go copyStream(c.ep.io.Stdout, stdcopy.Stdout)
	go copyStream(c.ep.io.Stderr, stdcopy.Stderr)
	wg.Wait()
	_ = c.pw.Close()
}

func (c *execConn) Read(p []byte) (int, error) { return c.pr.Read(p) }

func (c *execConn) Write(p []byte) (int, error) {
	if c.ep.io == nil || c.ep.io.Stdin == nil {
		return len(p), nil
	}
	return c.ep.io.Stdin.Write(p)
}

func (c *execConn) CloseWrite() error {
	c.ep.mu.Lock()
	p := c.ep.process
	c.ep.mu.Unlock()
	if p != nil {
		// As for containers, the shim keeps its own writer on the stdin
		// FIFO; CloseIO is what makes the process see EOF.
		if err := p.CloseIO(nsCtx(context.Background())); err != nil {
			log.G(context.Background()).WithError(err).Debug("closing exec stdin")
		}
	}
	if c.ep.io != nil && c.ep.io.Stdin != nil {
		return c.ep.io.Stdin.Close()
	}
	return nil
}

func (c *execConn) Close() error {
	var err error
	c.once.Do(func() {
		_ = c.pr.Close()
		if c.ep.io != nil {
			err = c.ep.io.Close()
		}
	})
	return err
}

func (*execConn) LocalAddr() net.Addr              { return unixAddr("standalone") }
func (*execConn) RemoteAddr() net.Addr             { return unixAddr("exec") }
func (*execConn) SetDeadline(time.Time) error      { return nil }
func (*execConn) SetReadDeadline(time.Time) error  { return nil }
func (*execConn) SetWriteDeadline(time.Time) error { return nil }

// ---- logs

func (c *apiClient) ContainerLogs(ctx context.Context, ref string, options client.ContainerLogsOptions) (client.ContainerLogsResult, error) {
	ctr, m, err := c.eng.loadSynced(ctx, ref)
	if err != nil {
		return nil, err
	}
	if !options.ShowStdout && !options.ShowStderr {
		return nil, fmt.Errorf("you must choose at least one stream: %w", cerrdefs.ErrInvalidArgument)
	}
	f, err := os.Open(m.LogPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return io.NopCloser(strings.NewReader("")), nil
		}
		return nil, err
	}
	since, err := parseLogTime(options.Since)
	if err != nil {
		return nil, err
	}
	until, err := parseLogTime(options.Until)
	if err != nil {
		return nil, err
	}
	tail := -1
	if options.Tail != "" && options.Tail != "all" {
		tail = parseIntDefault(options.Tail, -1)
	}
	pr, pw := io.Pipe()
	lr := &logReader{
		file:       f,
		out:        pw,
		tty:        m.Config.Tty,
		showStdout: options.ShowStdout,
		showStderr: options.ShowStderr,
		timestamps: options.Timestamps,
		since:      since,
		until:      until,
		tail:       tail,
		follow:     options.Follow,
		isRunning: func() bool {
			_, cur, err := c.eng.loadSynced(context.WithoutCancel(ctx), ctr.ID)
			return err == nil && (cur.State.Status == statusRunning || cur.State.Status == statusPaused)
		},
	}
	go lr.run(ctx)
	return pr, nil
}

func parseLogTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	if secs, frac, ok := strings.Cut(s, "."); ok {
		sec := parseIntDefault(secs, -1)
		nsec := parseIntDefault(frac, 0)
		if sec >= 0 {
			return time.Unix(int64(sec), int64(nsec)), nil
		}
	} else if sec := parseIntDefault(s, -1); sec >= 0 {
		return time.Unix(int64(sec), 0), nil
	}
	return time.Time{}, fmt.Errorf("invalid timestamp %q: %w", s, cerrdefs.ErrInvalidArgument)
}

type logReader struct {
	file       *os.File
	out        *io.PipeWriter
	tty        bool
	showStdout bool
	showStderr bool
	timestamps bool
	since      time.Time
	until      time.Time
	tail       int
	follow     bool
	isRunning  func() bool

	// offset is the position of the first byte not yet decoded; tailed
	// buffers the last entries when --tail is used.
	offset int64
	tailed []jsonLogEntry
}

// emit writes one log entry to the stream, applying the filters. It
// reports whether the reader is still there.
func (lr *logReader) emit(e jsonLogEntry) bool {
	if e.Stream == "stdout" && !lr.showStdout || e.Stream == "stderr" && !lr.showStderr {
		return true
	}
	if !lr.since.IsZero() && e.Time.Before(lr.since) {
		return true
	}
	if !lr.until.IsZero() && e.Time.After(lr.until) {
		return true
	}
	line := e.Log
	if lr.timestamps {
		line = e.Time.UTC().Format(time.RFC3339Nano) + " " + line
	}
	var frame []byte
	if lr.tty {
		frame = []byte(line)
	} else {
		st := stdcopy.Stdout
		if e.Stream == "stderr" {
			st = stdcopy.Stderr
		}
		frame = make([]byte, 8+len(line))
		frame[0] = byte(st)
		binary.BigEndian.PutUint32(frame[4:8], uint32(len(line)))
		copy(frame[8:], line)
	}
	_, err := lr.out.Write(frame)
	return err == nil
}

// readAll decodes the complete entries starting at the current offset,
// leaving the file positioned after the last complete entry so that
// partially written lines are retried on the next call. It reports whether
// the reader is still there.
func (lr *logReader) readAll() bool {
	if _, err := lr.file.Seek(lr.offset, io.SeekStart); err != nil {
		return false
	}
	dec := json.NewDecoder(lr.file)
	base := lr.offset
	for {
		var e jsonLogEntry
		if err := dec.Decode(&e); err != nil {
			return true
		}
		lr.offset = base + dec.InputOffset()
		if lr.tail >= 0 {
			lr.tailed = append(lr.tailed, e)
			if len(lr.tailed) > lr.tail {
				lr.tailed = lr.tailed[1:]
			}
			continue
		}
		if !lr.emit(e) {
			return false
		}
	}
}

func (lr *logReader) run(ctx context.Context) {
	defer lr.file.Close()
	defer lr.out.Close()

	if !lr.readAll() {
		return
	}
	for _, e := range lr.tailed {
		if !lr.emit(e) {
			return
		}
	}
	lr.tailed, lr.tail = nil, -1
	if !lr.follow {
		return
	}
	// Follow the file until the container stops and the rest of its output
	// has been read.
	lastCheck := time.Now()
	running := lr.isRunning()
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(150 * time.Millisecond):
		}
		if !lr.readAll() {
			return
		}
		if !running {
			lr.readAll()
			return
		}
		if time.Since(lastCheck) > time.Second {
			running = lr.isRunning()
			lastCheck = time.Now()
		}
	}
}
