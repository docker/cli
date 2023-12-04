package system

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// newDialStdioCommand creates a new cobra.Command for `docker system dial-stdio`
func newDialStdioCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "dial-stdio",
		Short:  "Proxy the stdio stream to the daemon connection. Should not be invoked manually.",
		Args:   cli.NoArgs,
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDialStdio(dockerCli)
		},
		ValidArgsFunction: completion.NoComplete,
	}
	return cmd
}

func runDialStdio(dockerCli command.Cli) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dialer := dockerCli.Client().Dialer()
	conn, err := dialer(ctx)
	if err != nil {
		return fmt.Errorf("failed to open the raw stream connection: %w", err)
	}
	defer conn.Close()

	var connHalfCloser halfCloser
	switch t := conn.(type) {
	case halfCloser:
		connHalfCloser = t
	case halfReadWriteCloser:
		connHalfCloser = &nopCloseReader{t}
	default:
		return errors.New("the raw stream connection does not implement halfCloser")
	}

	stdin2conn := make(chan error, 1)
	conn2stdout := make(chan error, 1)
	go func() {
		stdin2conn <- copier(connHalfCloser, &halfReadCloserWrapper{os.Stdin}, "stdin to stream")
	}()
	go func() {
		conn2stdout <- copier(&halfWriteCloserWrapper{os.Stdout}, connHalfCloser, "stream to stdout")
	}()
	select {
	case err = <-stdin2conn:
		if err != nil {
			return err
		}
		// wait for stdout
		err = <-conn2stdout
	case err = <-conn2stdout:
		// return immediately without waiting for stdin to be closed.
		// (stdin is never closed when tty)
	}
	return err
}

func copier(to halfWriteCloser, from halfReadCloser, debugDescription string) error {
	defer func() {
		if err := from.CloseRead(); err != nil {
			logrus.Errorf("error while CloseRead (%s): %v", debugDescription, err)
		}
		if err := to.CloseWrite(); err != nil {
			logrus.Errorf("error while CloseWrite (%s): %v", debugDescription, err)
		}
	}()
	if _, err := io.Copy(to, from); err != nil {
		return fmt.Errorf("error while Copy (%s): %w", debugDescription, err)
	}
	return nil
}

type halfReadCloser interface {
	io.Reader
	CloseRead() error
}

type halfWriteCloser interface {
	io.Writer
	CloseWrite() error
}

type halfCloser interface {
	halfReadCloser
	halfWriteCloser
}

type halfReadWriteCloser interface {
	io.Reader
	halfWriteCloser
}

type nopCloseReader struct {
	halfReadWriteCloser
}

func (x *nopCloseReader) CloseRead() error {
	return nil
}

type halfReadCloserWrapper struct {
	io.ReadCloser
}

func (x *halfReadCloserWrapper) CloseRead() error {
	return x.Close()
}

type halfWriteCloserWrapper struct {
	io.WriteCloser
}

func (x *halfWriteCloserWrapper) CloseWrite() error {
	return x.Close()
}
