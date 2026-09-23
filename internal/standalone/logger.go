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
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"time"

	eventstypes "github.com/containerd/containerd/api/events"
	bootapi "github.com/containerd/containerd/api/runtime/bootstrap/v1"
	taskapi "github.com/containerd/containerd/api/runtime/task/v3"
	eventsapi "github.com/containerd/containerd/api/services/ttrpc/events/v1"
	"github.com/containerd/containerd/v2/pkg/protobuf"
	"github.com/containerd/containerd/v2/pkg/shim"
	"github.com/containerd/fifo"
	"github.com/containerd/ttrpc"
	"github.com/containerd/typeurl/v2"
	"github.com/moby/moby/api/pkg/stdcopy"
	"golang.org/x/sys/unix"
	"google.golang.org/protobuf/types/known/emptypb"
)

// The logging helper is a copy of this binary, started by the CLI for each
// container before its task is created and detached from it (setsid), so
// that it outlives the CLI. It:
//
//   - reads the container's stdout/stderr FIFOs (which the shim writes to)
//     and appends them to a Docker-compatible json-file log;
//   - fans the live output out to "docker attach"/"docker run" clients over
//     a unix socket (multiplexed with the stdcopy framing unless the
//     container has a TTY);
//   - holds the container's stdin FIFO open so that stdin is not closed when
//     an attached client detaches (unless StdinOnce is set);
//   - serves the shim's ttrpc events endpoint, recording the container's
//     exit status in the metadata store when TaskExit arrives.
//
// It exits once the container's output streams are closed and the exit has
// been recorded.

const (
	// attachHandshake is sent by the helper to a newly attached client once
	// it is registered to receive output.
	attachHandshake = 1

	loggerConfigFile = "logger.json"
	// loggerReadyFD is the descriptor the helper uses to tell the CLI that
	// its sockets and FIFOs are ready.
	loggerReadyFD = 3
	// loggerStartupTimeout bounds how long the helper waits for the task to
	// show up before giving up (the CLI may have died before creating it).
	loggerStartupTimeout = 2 * time.Minute
)

// loggerConfig is written by the CLI before the helper is started.
type loggerConfig struct {
	ID           string `json:"id"`
	Root         string `json:"root"`
	RunDir       string `json:"runDir"`
	Snapshotter  string `json:"snapshotter"`
	LogPath      string `json:"logPath"`
	TTY          bool   `json:"tty"`
	OpenStdin    bool   `json:"openStdin"`
	StdinOnce    bool   `json:"stdinOnce"`
	StdinFifo    string `json:"stdinFifo"`
	StdoutFifo   string `json:"stdoutFifo"`
	StderrFifo   string `json:"stderrFifo"`
	AttachSocket string `json:"attachSocket"`
	EventsSocket string `json:"eventsSocket"`
	BundlePath   string `json:"bundlePath"`
}

func (lc *loggerConfig) write(dir string) error {
	data, err := json.Marshal(lc)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, loggerConfigFile), data, 0o600)
}

func readLoggerConfig(dir string) (*loggerConfig, error) {
	data, err := os.ReadFile(filepath.Join(dir, loggerConfigFile))
	if err != nil {
		return nil, err
	}
	var lc loggerConfig
	if err := json.Unmarshal(data, &lc); err != nil {
		return nil, err
	}
	return &lc, nil
}

// startLogger spawns the logging helper for a container and waits until it
// is ready to serve.
func (e *engine) startLogger(ctx context.Context, lc *loggerConfig, runDir string) error {
	if err := lc.write(runDir); err != nil {
		return err
	}
	readyR, readyW, err := os.Pipe()
	if err != nil {
		return err
	}
	defer readyR.Close()

	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		_ = readyW.Close()
		return err
	}
	defer devNull.Close()

	logFile, err := os.OpenFile(filepath.Join(runDir, "logger.err"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		_ = readyW.Close()
		return err
	}
	defer logFile.Close()

	//nolint:gosec // G204: the command is this binary, running its logging helper
	cmd := exec.Command(e.cfg.executable, loggerArg, runDir)
	cmd.Env = []string{"PATH=" + os.Getenv("PATH")}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = devNull, devNull, logFile
	cmd.ExtraFiles = []*os.File{readyW}
	cmd.SysProcAttr = &unix.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		_ = readyW.Close()
		return fmt.Errorf("starting logging helper: %w", err)
	}
	_ = readyW.Close()
	// Reap the helper if we outlive it; it is re-parented to init otherwise.
	go func() { _ = cmd.Wait() }()

	type result struct{ err error }
	done := make(chan result, 1)
	go func() {
		buf := make([]byte, 1)
		if _, err := io.ReadFull(readyR, buf); err != nil {
			done <- result{err: errors.New("logging helper exited before it became ready")}
			return
		}
		done <- result{}
	}()
	select {
	case r := <-done:
		if r.err != nil {
			if data, rerr := os.ReadFile(filepath.Join(runDir, "logger.err")); rerr == nil && len(data) > 0 {
				return fmt.Errorf("%w: %s", r.err, data)
			}
			return r.err
		}
		return nil
	case <-time.After(30 * time.Second):
		_ = cmd.Process.Kill()
		return errors.New("timed out waiting for the logging helper")
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		return ctx.Err()
	}
}

// RunLogger runs the logging helper. args are the process arguments
// (os.Args). It does not return.
func RunLogger(args []string) {
	dir := args[2]
	if err := runLogger(dir); err != nil {
		fmt.Fprintln(os.Stderr, "docker: logging helper:", err)
		os.Exit(1)
	}
	os.Exit(0)
}

// jsonLogEntry is the json-file log line format.
type jsonLogEntry struct {
	Log    string    `json:"log"`
	Stream string    `json:"stream"`
	Time   time.Time `json:"time"`
}

type logger struct {
	lc      *loggerConfig
	logFile *os.File
	logMu   sync.Mutex

	clientsMu sync.Mutex
	clients   map[net.Conn]struct{}

	// stdinMu guards the write end of the container's stdin FIFO. The
	// helper holds it open so that the container's stdin stays open while
	// no client is attached, and closes it when an attached client's input
	// ends, which propagates EOF to the container (as the daemon does).
	stdinMu sync.Mutex
	stdinW  io.WriteCloser

	exitCh chan struct{}
	exited sync.Once
}

//nolint:gocyclo // sets up the log file, both sockets and the FIFOs
func runLogger(dir string) error {
	lc, err := readLoggerConfig(dir)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}
	// Terminating signals are ignored: the helper must outlive the CLI that
	// started it (it shares its process group until setsid takes effect) and
	// exits when the container's streams are closed.
	signal.Ignore(unix.SIGINT, unix.SIGHUP, unix.SIGPIPE)

	if err := os.MkdirAll(filepath.Dir(lc.LogPath), 0o755); err != nil {
		return err
	}
	logFile, err := os.OpenFile(lc.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o640)
	if err != nil {
		return err
	}
	defer logFile.Close()

	l := &logger{lc: lc, logFile: logFile, clients: map[net.Conn]struct{}{}, exitCh: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Events endpoint for the shim.
	_ = os.Remove(lc.EventsSocket)
	evL, err := net.Listen("unix", lc.EventsSocket)
	if err != nil {
		return fmt.Errorf("listening on events socket: %w", err)
	}
	defer evL.Close()
	srv, err := ttrpc.NewServer()
	if err != nil {
		return err
	}
	eventsapi.RegisterTTRPCEventsService(srv, l)
	go func() { _ = srv.Serve(ctx, evL) }()
	defer srv.Close()

	// Attach endpoint for the CLI.
	_ = os.Remove(lc.AttachSocket)
	atL, err := net.Listen("unix", lc.AttachSocket)
	if err != nil {
		return fmt.Errorf("listening on attach socket: %w", err)
	}
	defer atL.Close()
	go l.acceptAttach(atL)

	// Open the output FIFOs for reading. The shim opens them for writing
	// when the task is created; the fifo package defers the open, so this
	// does not block here.
	stdout, err := fifo.OpenFifo(ctx, lc.StdoutFifo, unix.O_RDONLY|unix.O_NONBLOCK, 0)
	if err != nil {
		return fmt.Errorf("opening stdout fifo: %w", err)
	}
	defer stdout.Close()
	var stderr io.ReadCloser
	if lc.StderrFifo != "" {
		f, err := fifo.OpenFifo(ctx, lc.StderrFifo, unix.O_RDONLY|unix.O_NONBLOCK, 0)
		if err != nil {
			return fmt.Errorf("opening stderr fifo: %w", err)
		}
		stderr = f
		defer stderr.Close()
	}
	// Hold the container's stdin open for the helper's whole life, so that
	// clients can attach and detach repeatedly. EOF is propagated to the
	// container by the CLI through the task's CloseIO call, not by closing
	// this end: the shim keeps its own writer on the FIFO.
	if lc.OpenStdin && lc.StdinFifo != "" {
		w, err := fifo.OpenFifo(ctx, lc.StdinFifo, unix.O_WRONLY|unix.O_NONBLOCK, 0)
		if err != nil {
			return fmt.Errorf("opening stdin fifo: %w", err)
		}
		l.stdinW = w
		defer l.closeStdin()
	}

	// Signal readiness to the CLI.
	ready := os.NewFile(loggerReadyFD, "ready")
	if ready != nil {
		_, _ = ready.Write([]byte{0})
		_ = ready.Close()
	}

	// Give up if the task never materialises (the CLI died before creating
	// it), so that no helper is left behind forever. Closing the FIFOs is
	// what unblocks the copies below: the fifo package only honours the
	// context until the FIFO has been opened.
	go l.watchBundle(cancel)
	go func() {
		<-ctx.Done()
		_ = stdout.Close()
		if stderr != nil {
			_ = stderr.Close()
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		l.copyStream(stdout, "stdout", stdcopy.Stdout)
	}()
	if stderr != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l.copyStream(stderr, "stderr", stdcopy.Stderr)
		}()
	}
	wg.Wait()
	l.closeClients()

	// Give the shim a moment to deliver the exit event.
	select {
	case <-l.exitCh:
	case <-time.After(30 * time.Second):
	case <-ctx.Done():
	}
	return nil
}

// watchBundle cancels the helper's context if the task bundle never appears
// or disappears, which means the container is gone.
func (l *logger) watchBundle(cancel context.CancelFunc) {
	deadline := time.Now().Add(loggerStartupTimeout)
	seen := false
	for {
		time.Sleep(time.Second)
		_, err := os.Stat(l.lc.BundlePath)
		switch {
		case err == nil:
			seen = true
			deadline = time.Now().Add(loggerStartupTimeout)
		case seen:
			// The task was deleted; nothing more will be written.
			cancel()
			return
		case time.Now().After(deadline):
			cancel()
			return
		}
	}
}

// Forward implements the ttrpc events service used by the shim.
func (l *logger) Forward(ctx context.Context, req *eventsapi.ForwardRequest) (*emptypb.Empty, error) {
	env := req.GetEnvelope()
	if env == nil || env.GetEvent() == nil {
		return &emptypb.Empty{}, nil
	}
	ev, err := typeurl.UnmarshalAny(env.GetEvent())
	if err != nil {
		return &emptypb.Empty{}, nil
	}
	switch e := ev.(type) {
	case *eventstypes.TaskExit:
		if e.GetID() == e.GetContainerID() && e.GetContainerID() == l.lc.ID {
			l.recordExit(ctx, e.GetExitStatus(), protobuf.FromTimestamp(e.GetExitedAt()), false)
		}
	case *eventstypes.TaskOOM:
		if e.GetContainerID() == l.lc.ID {
			l.recordOOM(ctx)
		}
	}
	return &emptypb.Empty{}, nil
}

func (l *logger) cfg() *config {
	return &config{root: l.lc.Root, runDir: l.lc.RunDir, snapshotter: l.lc.Snapshotter}
}

func (l *logger) recordExit(ctx context.Context, status uint32, exitedAt time.Time, oom bool) {
	l.exited.Do(func() {
		defer close(l.exitCh)
		var err error
		for range 5 {
			if err = recordExit(ctx, l.cfg(), l.lc.ID, status, exitedAt, oom); err == nil {
				return
			}
			time.Sleep(200 * time.Millisecond)
		}
		fmt.Fprintln(os.Stderr, "recording container exit:", err)
	})
}

func (l *logger) recordOOM(ctx context.Context) {
	s, err := openSession(ctx, l.cfg())
	if err != nil {
		return
	}
	defer s.Close()
	ctx = nsCtx(ctx)
	c, err := s.containers().Get(ctx, l.lc.ID)
	if err != nil {
		return
	}
	m, err := unmarshalMeta(c)
	if err != nil {
		return
	}
	m.State.OOMKilled = true
	_ = updateMeta(ctx, s.containers(), c, m)
}

// copyStream reads one output stream, writing json-file entries and fanning
// out to attached clients.
func (l *logger) copyStream(r io.Reader, name string, st stdcopy.StdType) {
	buf := make([]byte, 32*1024)
	var partial []byte
	for {
		n, err := r.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			l.broadcast(st, chunk)
			partial = l.writeLog(name, partial, chunk)
		}
		if err != nil {
			if len(partial) > 0 {
				l.appendEntry(name, partial)
			}
			return
		}
	}
}

// writeLog appends complete lines from chunk to the log file and returns the
// remaining partial line.
func (l *logger) writeLog(stream string, partial, chunk []byte) []byte {
	data := make([]byte, 0, len(partial)+len(chunk))
	data = append(append(data, partial...), chunk...)
	for {
		i := indexByte(data, '\n')
		if i < 0 {
			break
		}
		l.appendEntry(stream, data[:i+1])
		data = data[i+1:]
	}
	if len(data) > 16*1024 {
		// Avoid unbounded buffering of very long lines.
		l.appendEntry(stream, data)
		return nil
	}
	return append([]byte(nil), data...)
}

func indexByte(b []byte, c byte) int {
	for i, x := range b {
		if x == c {
			return i
		}
	}
	return -1
}

func (l *logger) appendEntry(stream string, line []byte) {
	entry, err := json.Marshal(jsonLogEntry{Log: string(line), Stream: stream, Time: time.Now().UTC()})
	if err != nil {
		return
	}
	l.logMu.Lock()
	defer l.logMu.Unlock()
	_, _ = l.logFile.Write(append(entry, '\n'))
}

func (l *logger) acceptAttach(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		l.clientsMu.Lock()
		l.clients[conn] = struct{}{}
		l.clientsMu.Unlock()
		// Tell the client that it is registered: a connection is queued by
		// the kernel before it is accepted, so without this handshake the
		// container could produce output before the client is attached and
		// that output would be dropped.
		if _, err := conn.Write([]byte{attachHandshake}); err != nil {
			l.clientsMu.Lock()
			delete(l.clients, conn)
			l.clientsMu.Unlock()
			_ = conn.Close()
			continue
		}
		// The client's input is forwarded to the container's stdin. The
		// client stays registered for output when its input ends: a
		// half-close only ends the input direction.
		go func() {
			if w := l.stdin(); w != nil {
				_, _ = io.Copy(w, conn)
				// The client's input ended. With StdinOnce (as set by
				// "docker run -i"), the container's stdin is closed for
				// good, which requires closing both writers of the FIFO:
				// this one and the one the shim holds (through CloseIO).
				// Without StdinOnce the container's stdin stays open so
				// that clients can attach again later.
				if l.lc.StdinOnce {
					l.closeStdin()
					l.closeContainerStdin()
				}
				return
			}
			_, _ = io.Copy(io.Discard, conn)
		}()
	}
}

// broadcast sends a chunk of output to all attached clients.
func (l *logger) broadcast(st stdcopy.StdType, chunk []byte) {
	l.clientsMu.Lock()
	defer l.clientsMu.Unlock()
	if len(l.clients) == 0 {
		return
	}
	var frame []byte
	if l.lc.TTY {
		frame = chunk
	} else {
		frame = make([]byte, 8+len(chunk))
		frame[0] = byte(st)
		binary.BigEndian.PutUint32(frame[4:8], uint32(len(chunk)))
		copy(frame[8:], chunk)
	}
	for conn := range l.clients {
		_ = conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
		if _, err := conn.Write(frame); err != nil {
			delete(l.clients, conn)
			_ = conn.Close()
		}
	}
}

// closeContainerStdin asks the shim to close the container's stdin. A short
// grace period gives the shim time to drain the FIFO first.
func (l *logger) closeContainerStdin() {
	time.Sleep(150 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ctx = nsCtx(ctx)

	params, err := readBootstrap(l.lc.BundlePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "reading shim bootstrap:", err)
		return
	}
	conn, err := shim.AnonReconnectDialer(params.Address, 5*time.Second)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connecting to shim:", err)
		return
	}
	client := ttrpc.NewClient(conn)
	defer client.Close()
	if _, err := taskapi.NewTTRPCTaskClient(client).CloseIO(ctx, &taskapi.CloseIORequest{
		ID:    l.lc.ID,
		Stdin: true,
	}); err != nil {
		fmt.Fprintln(os.Stderr, "closing container stdin:", err)
	}
}

// readBootstrap reads the shim's connection parameters from its bundle.
func readBootstrap(bundle string) (*bootapi.BootstrapResult, error) {
	data, err := os.ReadFile(filepath.Join(bundle, "bootstrap.json"))
	if err != nil {
		return nil, err
	}
	var params bootapi.BootstrapResult
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, err
	}
	if params.Address == "" {
		return nil, errors.New("shim bootstrap has no address")
	}
	return &params, nil
}

func (l *logger) stdin() io.WriteCloser {
	l.stdinMu.Lock()
	defer l.stdinMu.Unlock()
	return l.stdinW
}

func (l *logger) closeStdin() {
	l.stdinMu.Lock()
	defer l.stdinMu.Unlock()
	if l.stdinW != nil {
		_ = l.stdinW.Close()
		l.stdinW = nil
	}
}

func (l *logger) closeClients() {
	l.clientsMu.Lock()
	defer l.clientsMu.Unlock()
	for conn := range l.clients {
		if cw, ok := conn.(interface{ CloseWrite() error }); ok {
			_ = cw.CloseWrite()
		} else {
			_ = conn.Close()
		}
		delete(l.clients, conn)
	}
}
