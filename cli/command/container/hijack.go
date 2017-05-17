package container

import (
	"io"
	"runtime"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/ioutils"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/docker/pkg/term"
	"golang.org/x/net/context"
)

// The default escape key sequence: ctrl-p, ctrl-q
var defaultEscapeKeys = []byte{16, 17}

// holdHijackedConnection handles copying input to and output from streams to the
// connection
// nolint: gocyclo
func holdHijackedConnection(ctx context.Context, streams command.Streams, tty bool, detachKeys string, inputStream io.ReadCloser, outputStream, errorStream io.Writer, resp types.HijackedResponse) error {
	var (
		err         error
		restoreOnce sync.Once
	)
	if inputStream != nil && tty {
		if err := setRawTerminal(streams); err != nil {
			return err
		}
		defer func() {
			restoreOnce.Do(func() {
				restoreTerminal(streams, inputStream)
			})
		}()

		// Wrap the input to detect detach control sequence.
		// Use default detach sequence if an invalid sequence is given.
		escapeKeys, err := term.ToBytes(detachKeys)
		if len(escapeKeys) == 0 || err != nil {
			escapeKeys = defaultEscapeKeys
		}

		inputStream = ioutils.NewReadCloserWrapper(term.NewEscapeProxy(inputStream, escapeKeys), inputStream.Close)
	}

	receiveStdout := make(chan error, 1)
	if outputStream != nil || errorStream != nil {
		go func() {
			// When TTY is ON, use regular copy
			if tty && outputStream != nil {
				_, err = io.Copy(outputStream, resp.Reader)
				// we should restore the terminal as soon as possible once connection end
				// so any following print messages will be in normal type.
				if inputStream != nil {
					restoreOnce.Do(func() {
						restoreTerminal(streams, inputStream)
					})
				}
			} else {
				_, err = stdcopy.StdCopy(outputStream, errorStream, resp.Reader)
			}

			logrus.Debug("[hijack] End of stdout")
			receiveStdout <- err
		}()
	}

	stdinDone := make(chan struct{})
	detachedC := make(chan term.EscapeError)
	go func() {
		if inputStream != nil {
			_, inputErr := io.Copy(resp.Conn, inputStream)
			// we should restore the terminal as soon as possible once connection end
			// so any following print messages will be in normal type.
			if tty {
				restoreOnce.Do(func() {
					restoreTerminal(streams, inputStream)
				})
			}
			logrus.Debug("[hijack] End of stdin")

			if detached, ok := inputErr.(term.EscapeError); ok {
				detachedC <- detached
				return
			}
		}

		if err := resp.CloseWrite(); err != nil {
			logrus.Debugf("Couldn't send EOF: %s", err)
		}
		close(stdinDone)
	}()

	select {
	case err := <-receiveStdout:
		if err != nil {
			logrus.Debugf("Error receiveStdout: %s", err)
			return err
		}
	case <-stdinDone:
		if outputStream != nil || errorStream != nil {
			select {
			case err := <-receiveStdout:
				if err != nil {
					logrus.Debugf("Error receiveStdout: %s", err)
					return err
				}
			case <-ctx.Done():
			}
		}
	case err := <-detachedC:
		// Got a detach key sequence.
		return err
	case <-ctx.Done():
	}

	return nil
}

func setRawTerminal(streams command.Streams) error {
	if err := streams.In().SetRawTerminal(); err != nil {
		return err
	}
	return streams.Out().SetRawTerminal()
}

func restoreTerminal(streams command.Streams, in io.Closer) error {
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
	if in != nil && runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		return in.Close()
	}
	return nil
}
