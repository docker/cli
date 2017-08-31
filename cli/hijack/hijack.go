package hijack

import (
	"fmt"
	"io"
	"runtime"
	"sync"

	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/ioutils"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/docker/pkg/term"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

// The default escape key sequence: ctrl-p, ctrl-q
// TODO: This could be moved to `pkg/term`.
var defaultEscapeKeys = []byte{16, 17}

// StreamOptions used by Stream to configure streaming behaviour
type StreamOptions struct {
	Hijacked     types.HijackedResponse
	AttachStdin  bool
	AttachStdout bool
	AttachStderr bool
	Tty          bool
	DetachKeys   string
}

// Stream handles setting up the IO and then begins streaming stdin/stdout
// to/from the hijacked connection, blocking until it is either done reading
// output, the user inputs the detach key sequence when in TTY mode, or when
// the given context is cancelled.
func Stream(ctx context.Context, streams command.Streams, opts StreamOptions) error {
	restoreInput, inputStream, err := setupInput(streams, opts)
	if err != nil {
		return fmt.Errorf("unable to setup input stream: %s", err)
	}

	defer restoreInput()

	outputDone := beginOutputStream(streams, opts, restoreInput)
	inputDone, detached := beginInputStream(&opts, inputStream, restoreInput)

	select {
	case err := <-outputDone:
		return err
	case <-inputDone:
		// Input Stream has closed.
		if opts.AttachStdout || opts.AttachStderr {
			// Wait for output to complete streaming.
			select {
			case err := <-outputDone:
				return err
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	case err := <-detached:
		// Got a detach key sequence.
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func setupInput(streams command.Streams, opts StreamOptions) (restore func(), reader io.Reader, err error) {
	if !opts.AttachStdin || !opts.Tty {
		// No need to setup input TTY.
		// The restore func is a nop.
		return func() {}, streams.In(), nil
	}

	if err := setRawTerminal(streams); err != nil {
		return nil, nil, fmt.Errorf("unable to set IO streams as raw terminal: %s", err)
	}

	// Use sync.Once so we may call restore multiple times but ensure we
	// only restore the terminal once.
	var restoreOnce sync.Once
	restore = func() {
		restoreOnce.Do(func() {
			restoreTerminal(streams, opts.AttachStdin)
		})
	}

	// Wrap the input to detect detach escape sequence.
	// Use default escape keys if an invalid sequence is given.
	escapeKeys := defaultEscapeKeys
	if opts.DetachKeys != "" {
		customEscapeKeys, err := term.ToBytes(opts.DetachKeys)
		if err != nil {
			logrus.Warnf("invalid detach escape keys, using default: %s", err)
		} else {
			escapeKeys = customEscapeKeys
		}
	}
	inputStream := ioutils.NewReadCloserWrapper(term.NewEscapeProxy(streams.In(), escapeKeys), streams.In().Close)

	return restore, inputStream, nil
}

func beginOutputStream(streams command.Streams, opts StreamOptions, restoreInput func()) <-chan error {
	if !opts.AttachStdout && !opts.AttachStderr {
		// There is no need to copy output.
		return nil
	}

	outputDone := make(chan error)
	go func() {
		var err error

		// When TTY is ON, use regular copy
		if opts.AttachStdout && opts.Tty {
			_, err = io.Copy(streams.Out(), opts.Hijacked.Reader)
			// We should restore the terminal as soon as possible
			// once the connection ends so any following print
			// messages will be in normal type.
			restoreInput()
		} else {
			_, err = stdcopy.StdCopy(streams.Out(), streams.Err(), opts.Hijacked.Reader)
		}

		logrus.Debug("[hijack] End of stdout")

		if err != nil {
			logrus.Debugf("Error receiveStdout: %s", err)
		}

		outputDone <- err
	}()

	return outputDone
}

func beginInputStream(opts *StreamOptions, inputStream io.Reader, restoreInput func()) (doneC <-chan struct{}, detachedC <-chan error) {
	inputDone := make(chan struct{})
	detached := make(chan error)

	go func() {
		if opts.AttachStdin {
			_, err := io.Copy(opts.Hijacked.Conn, inputStream)
			// We should restore the terminal as soon as possible
			// once the connection ends so any following print
			// messages will be in normal type.
			restoreInput()

			logrus.Debug("[hijack] End of stdin")

			if _, ok := err.(term.EscapeError); ok {
				detached <- err
				return
			}

			if err != nil {
				// This error will also occur on the receive
				// side (from stdout) where it will be
				// propagated back to the caller.
				logrus.Debugf("Error sendStdin: %s", err)
			}
		}

		if err := opts.Hijacked.CloseWrite(); err != nil {
			logrus.Debugf("Couldn't send EOF: %s", err)
		}

		close(inputDone)
	}()

	return inputDone, detached
}

func setRawTerminal(streams command.Streams) error {
	if err := streams.In().SetRawTerminal(); err != nil {
		return err
	}
	return streams.Out().SetRawTerminal()
}

func restoreTerminal(streams command.Streams, in bool) error {
	streams.In().RestoreTerminal()
	streams.Out().RestoreTerminal()
	// WARNING: DO NOT REMOVE THE OS CHECKS !!!
	// For some reason this Close call blocks on darwin..
	// As the client exits right after, simply discard the close
	// until we find a better solution.
	//
	// This can also cause the client on Windows to get stuck in Win32 CloseHandle()
	// in some cases. See https://github.com/docker/docker/issues/28267#issuecomment-288237442
	// Tracked internally at Microsoft by VSO #11352156. In the
	// Windows case, you hit this if you are using the native/v2 console,
	// not the "legacy" console, and you start the client in a new window. eg
	// `start docker run --rm -it microsoft/nanoserver cmd /s /c echo foobar`
	// will hang. Remove start, and it won't repro.
	if in && runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		return streams.In().Close()
	}
	return nil
}
